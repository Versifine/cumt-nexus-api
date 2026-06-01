package voterepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/voteusecase"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ voteusecase.PostVoteRepository = (*PostgresPostVoteRepository)(nil)

type PostgresPostVoteRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPostVoteRepository(pool *pgxpool.Pool) *PostgresPostVoteRepository {
	return &PostgresPostVoteRepository{
		pool: pool,
	}
}

func (repo *PostgresPostVoteRepository) Upsert(ctx context.Context, vote votedomain.PostVote) error {
	const query = `
		INSERT INTO post_votes (
			post_id,
			user_id,
			value,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5)
		ON CONFLICT (post_id, user_id)
		DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = EXCLUDED.updated_at
	`

	_, err := repo.pool.Exec(
		ctx,
		query,
		vote.PostID().String(),
		vote.UserID().String(),
		vote.Value().Int(),
		vote.CreatedAt(),
		vote.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("upsert post vote", err)
	}

	return nil
}

func (repo *PostgresPostVoteRepository) DeleteByPostAndUser(ctx context.Context, postID postdomain.PostID, userID userdomain.UserID) error {
	const query = `
		DELETE FROM post_votes
		WHERE post_id = $1::uuid
			AND user_id = $2::uuid
	`

	if _, err := repo.pool.Exec(ctx, query, postID.String(), userID.String()); err != nil {
		return fmt.Errorf("delete post vote: %w", err)
	}

	return nil
}

func (repo *PostgresPostVoteRepository) FindByPostAndUser(ctx context.Context, postID postdomain.PostID, userID userdomain.UserID) (*votedomain.PostVote, error) {
	const query = `
		SELECT
			post_id::text,
			user_id::text,
			value,
			created_at,
			updated_at
		FROM post_votes
		WHERE post_id = $1::uuid
			AND user_id = $2::uuid
		LIMIT 1
	`

	vote, err := scanPostVote(repo.pool.QueryRow(ctx, query, postID.String(), userID.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeNotFound, "post vote not found")
		}
		return nil, err
	}

	return vote, nil
}

func (repo *PostgresPostVoteRepository) FindByPostIDsAndUser(ctx context.Context, postIDs []postdomain.PostID, userID userdomain.UserID) (map[postdomain.PostID]votedomain.VoteValue, error) {
	result := make(map[postdomain.PostID]votedomain.VoteValue, len(postIDs))
	if len(postIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			post_id::text,
			value
		FROM post_votes
		WHERE user_id = $1::uuid
			AND post_id = ANY($2::uuid[])
	`

	rows, err := repo.pool.Query(ctx, query, userID.String(), postIDStrings(postIDs))
	if err != nil {
		return nil, fmt.Errorf("find post votes by user: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rawPostID string
		var rawValue int
		if err := rows.Scan(&rawPostID, &rawValue); err != nil {
			return nil, err
		}
		postID, err := postdomain.NewPostID(rawPostID)
		if err != nil {
			return nil, fmt.Errorf("rehydrate vote post id: %v", err)
		}
		value, err := votedomain.NewVoteValue(rawValue)
		if err != nil {
			return nil, fmt.Errorf("rehydrate vote value: %v", err)
		}
		result[postID] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate post votes by user: %w", err)
	}

	return result, nil
}

func (repo *PostgresPostVoteRepository) SummarizeByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID]voteusecase.PostVoteSummary, error) {
	result := make(map[postdomain.PostID]voteusecase.PostVoteSummary, len(postIDs))
	if len(postIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			post_id::text,
			COUNT(*) FILTER (WHERE value = 1)::int,
			COUNT(*) FILTER (WHERE value = -1)::int
		FROM post_votes
		WHERE post_id = ANY($1::uuid[])
		GROUP BY post_id
	`

	rows, err := repo.pool.Query(ctx, query, postIDStrings(postIDs))
	if err != nil {
		return nil, fmt.Errorf("summarize post votes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rawPostID string
		var upvoteCount int
		var downvoteCount int
		if err := rows.Scan(&rawPostID, &upvoteCount, &downvoteCount); err != nil {
			return nil, err
		}
		postID, err := postdomain.NewPostID(rawPostID)
		if err != nil {
			return nil, fmt.Errorf("rehydrate vote summary post id: %v", err)
		}
		result[postID] = voteusecase.PostVoteSummary{
			PostID:        postID,
			UpvoteCount:   upvoteCount,
			DownvoteCount: downvoteCount,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate post vote summaries: %w", err)
	}

	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPostVote(row rowScanner) (*votedomain.PostVote, error) {
	var rawPostID string
	var rawUserID string
	var rawValue int
	var createdAt time.Time
	var updatedAt time.Time

	if err := row.Scan(&rawPostID, &rawUserID, &rawValue, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	postID, err := postdomain.NewPostID(rawPostID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate vote post id: %v", err)
	}
	userID, err := userdomain.NewUserID(rawUserID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate vote user id: %v", err)
	}
	value, err := votedomain.NewVoteValue(rawValue)
	if err != nil {
		return nil, fmt.Errorf("rehydrate vote value: %v", err)
	}
	vote, err := votedomain.RehydratePostVote(postID, userID, value, createdAt, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("rehydrate post vote: %v", err)
	}

	return vote, nil
}

func postIDStrings(postIDs []postdomain.PostID) []string {
	rawIDs := make([]string, 0, len(postIDs))
	for _, postID := range postIDs {
		rawIDs = append(rawIDs, postID.String())
	}
	return rawIDs
}

func mapPostgresWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23503" {
			return apperr.New(apperr.CodeNotFound, "related record not found")
		}
		if pgErr.Code == "23514" {
			return apperr.New(apperr.CodeInvalidArgument, "post vote is invalid")
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

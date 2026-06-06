package commentrepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentusecase"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ commentusecase.CommentRepository = (*PostgresCommentRepository)(nil)
var _ commentusecase.CommentMetadataRepository = (*PostgresCommentRepository)(nil)
var _ commentusecase.CommentVoteRepository = (*PostgresCommentRepository)(nil)

type PostgresCommentRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresCommentRepository(pool *pgxpool.Pool) *PostgresCommentRepository {
	return &PostgresCommentRepository{
		pool: pool,
	}
}

func (repo *PostgresCommentRepository) Create(ctx context.Context, comment commentdomain.Comment) error {
	const query = `
		INSERT INTO comments (
			id,
			post_id,
			author_id,
			parent_id,
			body,
			status,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8)
	`

	parentID, hasParentID := comment.ParentID()
	_, err := repo.pool.Exec(
		ctx,
		query,
		comment.ID().String(),
		comment.PostID().String(),
		comment.AuthorID().String(),
		nullableCommentIDValue(parentID, hasParentID),
		comment.Body().String(),
		comment.Status().String(),
		comment.CreatedAt(),
		comment.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("create comment", err)
	}

	return nil
}

func (repo *PostgresCommentRepository) FindVisibleByID(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
	const query = `
		SELECT
			id::text,
			post_id::text,
			author_id::text,
			parent_id::text,
			body,
			status,
			created_at,
			updated_at
		FROM comments
		WHERE id = $1::uuid
			AND status = 'visible'
		LIMIT 1
	`

	comment, err := scanComment(repo.pool.QueryRow(ctx, query, id.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeNotFound, "comment not found")
		}
		return nil, err
	}

	return comment, nil
}

func (repo *PostgresCommentRepository) UpdateContent(ctx context.Context, comment commentdomain.Comment) error {
	const query = `
		UPDATE comments
		SET
			body = $2,
			updated_at = $3
		WHERE id = $1::uuid
			AND status = 'visible'
	`

	tag, err := repo.pool.Exec(
		ctx,
		query,
		comment.ID().String(),
		comment.Body().String(),
		comment.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("update comment content", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "comment not found")
	}

	return nil
}

func (repo *PostgresCommentRepository) MarkDeleted(ctx context.Context, comment commentdomain.Comment) error {
	const query = `
		UPDATE comments
		SET
			status = 'deleted',
			updated_at = $2
		WHERE id = $1::uuid
			AND status = 'visible'
	`

	tag, err := repo.pool.Exec(ctx, query, comment.ID().String(), comment.UpdatedAt())
	if err != nil {
		return mapPostgresWriteError("delete comment", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "comment not found")
	}

	return nil
}

func (repo *PostgresCommentRepository) ListVisibleByPost(ctx context.Context, postID postdomain.PostID, limit int, offset int) ([]commentdomain.Comment, error) {
	const query = `
		SELECT
			id::text,
			post_id::text,
			author_id::text,
			parent_id::text,
			body,
			status,
			created_at,
			updated_at
		FROM comments
		WHERE post_id = $1::uuid
			AND status = 'visible'
		ORDER BY created_at DESC, id DESC
		LIMIT $2
		OFFSET $3
	`

	rows, err := repo.pool.Query(ctx, query, postID.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list visible comments by post: %w", err)
	}
	defer rows.Close()

	var comments []commentdomain.Comment
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, *comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate visible comments by post: %w", err)
	}

	return comments, nil
}

func (repo *PostgresCommentRepository) ListVisibleTreeByPost(ctx context.Context, postID postdomain.PostID) ([]commentdomain.Comment, error) {
	const query = `
		SELECT
			id::text,
			post_id::text,
			author_id::text,
			parent_id::text,
			body,
			status,
			created_at,
			updated_at
		FROM comments
		WHERE post_id = $1::uuid
			AND status = 'visible'
		ORDER BY created_at DESC, id DESC
	`

	rows, err := repo.pool.Query(ctx, query, postID.String())
	if err != nil {
		return nil, fmt.Errorf("list visible comment tree by post: %w", err)
	}
	defer rows.Close()

	var comments []commentdomain.Comment
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, *comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate visible comment tree by post: %w", err)
	}

	return comments, nil
}

func (repo *PostgresCommentRepository) ListVisibleByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID, limit int, offset int) ([]commentdomain.Comment, error) {
	const query = `
		SELECT
			comments.id::text,
			comments.post_id::text,
			comments.author_id::text,
			comments.parent_id::text,
			comments.body,
			comments.status,
			comments.created_at,
			comments.updated_at
		FROM comments
		INNER JOIN posts ON posts.id = comments.post_id
		INNER JOIN communities ON communities.id = posts.community_id
		WHERE comments.author_id = $1::uuid
			AND comments.status = 'visible'
			AND posts.status = 'visible'
			AND communities.status = 'active'
			AND communities.visibility = 'public'
		ORDER BY comments.created_at DESC, comments.id DESC
		LIMIT $2
		OFFSET $3
	`

	rows, err := repo.pool.Query(ctx, query, authorID.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list visible comments by author in public communities: %w", err)
	}
	defer rows.Close()

	var comments []commentdomain.Comment
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, *comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate visible comments by author in public communities: %w", err)
	}

	return comments, nil
}

func (repo *PostgresCommentRepository) LoadMetadataByCommentIDs(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID]commentusecase.CommentMetadata, error) {
	result := make(map[commentdomain.CommentID]commentusecase.CommentMetadata, len(commentIDs))
	if len(commentIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			comments.id::text,
			users.id::text,
			users.username
		FROM comments
		INNER JOIN users ON users.id = comments.author_id
		WHERE comments.id = ANY($1::uuid[])
	`

	rows, err := repo.pool.Query(ctx, query, commentIDStrings(commentIDs))
	if err != nil {
		return nil, fmt.Errorf("load comment metadata: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rawCommentID string
		var rawAuthorID string
		var rawUsername string
		if err := rows.Scan(&rawCommentID, &rawAuthorID, &rawUsername); err != nil {
			return nil, err
		}
		commentID, err := commentdomain.NewCommentID(rawCommentID)
		if err != nil {
			return nil, fmt.Errorf("rehydrate metadata comment id: %v", err)
		}
		result[commentID] = commentusecase.CommentMetadata{
			Author: postusecase.UserSummary{
				ID:          rawAuthorID,
				Username:    rawUsername,
				DisplayName: rawUsername,
				Badges:      []string{},
			},
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comment metadata: %w", err)
	}

	return result, nil
}

func commentIDStrings(commentIDs []commentdomain.CommentID) []string {
	rawIDs := make([]string, 0, len(commentIDs))
	for _, commentID := range commentIDs {
		rawIDs = append(rawIDs, commentID.String())
	}
	return rawIDs
}

func (repo *PostgresCommentRepository) UpsertCommentVote(ctx context.Context, vote votedomain.CommentVote) error {
	const query = `
		INSERT INTO comment_votes (
			comment_id,
			user_id,
			value,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5)
		ON CONFLICT (comment_id, user_id)
		DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = EXCLUDED.updated_at
	`

	_, err := repo.pool.Exec(
		ctx,
		query,
		vote.CommentID().String(),
		vote.UserID().String(),
		vote.Value().Int(),
		vote.CreatedAt(),
		vote.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("upsert comment vote", err)
	}

	return nil
}

func (repo *PostgresCommentRepository) DeleteCommentVote(ctx context.Context, commentID commentdomain.CommentID, userID userdomain.UserID) error {
	const query = `
		DELETE FROM comment_votes
		WHERE comment_id = $1::uuid
			AND user_id = $2::uuid
	`

	if _, err := repo.pool.Exec(ctx, query, commentID.String(), userID.String()); err != nil {
		return mapPostgresWriteError("delete comment vote", err)
	}

	return nil
}

func (repo *PostgresCommentRepository) FindCommentVotesByIDsAndUser(ctx context.Context, commentIDs []commentdomain.CommentID, userID userdomain.UserID) (map[commentdomain.CommentID]votedomain.VoteValue, error) {
	result := make(map[commentdomain.CommentID]votedomain.VoteValue, len(commentIDs))
	if len(commentIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			comment_id::text,
			value
		FROM comment_votes
		WHERE comment_id = ANY($1::uuid[])
			AND user_id = $2::uuid
	`

	rows, err := repo.pool.Query(ctx, query, commentIDStrings(commentIDs), userID.String())
	if err != nil {
		return nil, fmt.Errorf("find comment votes by user: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rawCommentID string
		var rawValue int
		if err := rows.Scan(&rawCommentID, &rawValue); err != nil {
			return nil, err
		}
		commentID, err := commentdomain.NewCommentID(rawCommentID)
		if err != nil {
			return nil, fmt.Errorf("rehydrate comment vote id: %v", err)
		}
		value, err := votedomain.NewVoteValue(rawValue)
		if err != nil {
			return nil, fmt.Errorf("rehydrate comment vote value: %v", err)
		}
		result[commentID] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comment votes by user: %w", err)
	}

	return result, nil
}

func (repo *PostgresCommentRepository) SummarizeCommentVotesByIDs(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID]votedomain.CommentVoteSummary, error) {
	result := make(map[commentdomain.CommentID]votedomain.CommentVoteSummary, len(commentIDs))
	if len(commentIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			comment_id::text,
			COUNT(*) FILTER (WHERE value = 1)::int,
			COUNT(*) FILTER (WHERE value = -1)::int
		FROM comment_votes
		WHERE comment_id = ANY($1::uuid[])
		GROUP BY comment_id
	`

	rows, err := repo.pool.Query(ctx, query, commentIDStrings(commentIDs))
	if err != nil {
		return nil, fmt.Errorf("summarize comment votes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rawCommentID string
		var upvoteCount int
		var downvoteCount int
		if err := rows.Scan(&rawCommentID, &upvoteCount, &downvoteCount); err != nil {
			return nil, err
		}
		commentID, err := commentdomain.NewCommentID(rawCommentID)
		if err != nil {
			return nil, fmt.Errorf("rehydrate comment vote summary id: %v", err)
		}
		result[commentID] = votedomain.CommentVoteSummary{
			CommentID:     commentID,
			UpvoteCount:   upvoteCount,
			DownvoteCount: downvoteCount,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comment vote summaries: %w", err)
	}

	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanComment(row rowScanner) (*commentdomain.Comment, error) {
	var rawID string
	var rawPostID string
	var rawAuthorID string
	var rawParentID pgtype.Text
	var rawBody string
	var rawStatus string
	var createdAt time.Time
	var updatedAt time.Time

	if err := row.Scan(
		&rawID,
		&rawPostID,
		&rawAuthorID,
		&rawParentID,
		&rawBody,
		&rawStatus,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	id, err := commentdomain.NewCommentID(rawID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate comment id: %v", err)
	}
	postID, err := postdomain.NewPostID(rawPostID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate comment post id: %v", err)
	}
	authorID, err := userdomain.NewUserID(rawAuthorID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate comment author id: %v", err)
	}
	var parentID *commentdomain.CommentID
	if rawParentID.Valid {
		parsedParentID, err := commentdomain.NewCommentID(rawParentID.String)
		if err != nil {
			return nil, fmt.Errorf("rehydrate comment parent id: %v", err)
		}
		parentID = &parsedParentID
	}
	body, err := commentdomain.NewCommentBody(rawBody)
	if err != nil {
		return nil, fmt.Errorf("rehydrate comment body: %v", err)
	}
	status, err := commentdomain.NewCommentStatus(rawStatus)
	if err != nil {
		return nil, fmt.Errorf("rehydrate comment status: %v", err)
	}

	comment, err := commentdomain.RehydrateComment(id, postID, authorID, parentID, body, status, createdAt, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("rehydrate comment: %v", err)
	}

	return comment, nil
}

func nullableCommentIDValue(id commentdomain.CommentID, ok bool) any {
	if !ok {
		return nil
	}
	return id.String()
}

func mapPostgresWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return apperr.New(apperr.CodeConflict, "comment already exists")
		}
		if pgErr.Code == "23503" {
			return apperr.New(apperr.CodeNotFound, "related record not found")
		}
		if pgErr.Code == "23514" {
			if pgErr.ConstraintName == "comment_votes_value_check" || pgErr.ConstraintName == "comment_votes_updated_at_check" {
				return apperr.New(apperr.CodeInvalidArgument, "comment vote is invalid")
			}
			return apperr.New(apperr.CodeInvalidArgument, "comment is invalid")
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

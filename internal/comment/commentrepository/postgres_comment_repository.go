package commentrepository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentusecase"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
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
var _ commentusecase.ContentRefRepository = (*PostgresCommentRepository)(nil)
var _ commentusecase.CommentEffectRepository = (*PostgresCommentRepository)(nil)

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

func (repo *PostgresCommentRepository) ListVisibleByPost(ctx context.Context, postID postdomain.PostID, sort commentusecase.CommentListSort, limit int, offset int) ([]commentdomain.Comment, error) {
	query := fmt.Sprintf(`
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
		%s
		WHERE comments.post_id = $1::uuid
			AND comments.status = 'visible'
		ORDER BY %s
		LIMIT $2
		OFFSET $3
	`, commentListStatsJoin(sort), commentListOrderBy(sort))

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

func commentListStatsJoin(sort commentusecase.CommentListSort) string {
	switch sort {
	case commentusecase.CommentListSortBest, commentusecase.CommentListSortTop, commentusecase.CommentListSortControversial:
		return `
		LEFT JOIN (
			SELECT
				comment_id,
				COUNT(*)::int AS vote_count,
				COUNT(*) FILTER (WHERE value = 1)::int AS upvote_count,
				COUNT(*) FILTER (WHERE value = -1)::int AS downvote_count,
				COALESCE(SUM(value), 0)::int AS net_score
			FROM comment_votes
			GROUP BY comment_id
		) AS comment_vote_stats ON comment_vote_stats.comment_id = comments.id
	`
	default:
		return ""
	}
}

func commentListOrderBy(sort commentusecase.CommentListSort) string {
	switch sort {
	case commentusecase.CommentListSortOld:
		return "comments.created_at ASC, comments.id ASC"
	case commentusecase.CommentListSortTop:
		return `
			COALESCE(comment_vote_stats.net_score, 0) DESC,
			COALESCE(comment_vote_stats.upvote_count, 0) DESC,
			comments.created_at DESC,
			comments.id DESC
		`
	case commentusecase.CommentListSortBest:
		return `
			CASE
				WHEN COALESCE(comment_vote_stats.vote_count, 0) = 0 THEN 0
				ELSE (
					(
						(COALESCE(comment_vote_stats.upvote_count, 0)::double precision / COALESCE(comment_vote_stats.vote_count, 0)::double precision)
						+ (1.9208 / COALESCE(comment_vote_stats.vote_count, 0)::double precision)
						- (
							1.96 * sqrt(
								(
									(
										(COALESCE(comment_vote_stats.upvote_count, 0)::double precision / COALESCE(comment_vote_stats.vote_count, 0)::double precision)
										* (1 - (COALESCE(comment_vote_stats.upvote_count, 0)::double precision / COALESCE(comment_vote_stats.vote_count, 0)::double precision))
									)
									+ (0.9604 / COALESCE(comment_vote_stats.vote_count, 0)::double precision)
								) / COALESCE(comment_vote_stats.vote_count, 0)::double precision
							)
						)
					) / (1 + (3.8416 / COALESCE(comment_vote_stats.vote_count, 0)::double precision))
				)
			END DESC,
			COALESCE(comment_vote_stats.net_score, 0) DESC,
			comments.created_at DESC,
			comments.id DESC
		`
	case commentusecase.CommentListSortControversial:
		return `
			(
				COALESCE(comment_vote_stats.vote_count, 0)
				* (
					1 - (
						ABS(COALESCE(comment_vote_stats.upvote_count, 0) - COALESCE(comment_vote_stats.downvote_count, 0))::double precision
						/ GREATEST(COALESCE(comment_vote_stats.vote_count, 0), 1)::double precision
					)
				)
			) DESC,
			COALESCE(comment_vote_stats.vote_count, 0) DESC,
			comments.created_at DESC,
			comments.id DESC
		`
	default:
		return "comments.created_at DESC, comments.id DESC"
	}
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

func (repo *PostgresCommentRepository) ListCommentsByCommunityForManagement(ctx context.Context, communityID communitydomain.CommunityID, status *commentdomain.CommentStatus, limit int, offset int) ([]commentdomain.Comment, error) {
	whereClause := "posts.community_id = $1::uuid"
	queryArgs := []any{communityID.String()}
	if status != nil {
		queryArgs = append(queryArgs, status.String())
		whereClause = fmt.Sprintf("%s AND comments.status = $%d", whereClause, len(queryArgs))
	}
	limitPlaceholder := len(queryArgs) + 1
	offsetPlaceholder := len(queryArgs) + 2

	query := fmt.Sprintf(`
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
		WHERE %s
		ORDER BY comments.created_at DESC, comments.id DESC
		LIMIT $%d
		OFFSET $%d
	`, whereClause, limitPlaceholder, offsetPlaceholder)

	queryArgs = append(queryArgs, limit, offset)
	rows, err := repo.pool.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("list community comments for management: %w", err)
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
		return nil, fmt.Errorf("iterate community comments for management: %w", err)
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
			users.username,
			users.display_name,
			users.avatar_url,
			users.headline
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
		var rawDisplayName string
		var rawAvatarURL string
		var rawHeadline string
		if err := rows.Scan(&rawCommentID, &rawAuthorID, &rawUsername, &rawDisplayName, &rawAvatarURL, &rawHeadline); err != nil {
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
				DisplayName: fallbackDisplayName(rawDisplayName, rawUsername),
				AvatarURL:   rawAvatarURL,
				Headline:    rawHeadline,
				Badges:      []string{},
			},
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comment metadata: %w", err)
	}

	return result, nil
}

func (repo *PostgresCommentRepository) ListCommentEffectsByCommentIDs(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID][]commentusecase.CommentEffectSummary, error) {
	result := make(map[commentdomain.CommentID][]commentusecase.CommentEffectSummary, len(commentIDs))
	if len(commentIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			comment_effects.comment_id::text,
			comment_effects.id::text,
			comment_effects.effect_id,
			effects.name,
			effects.asset_url,
			effects.animation_key,
			users.id::text,
			users.username,
			users.display_name,
			users.avatar_url,
			users.headline,
			comment_effects.points_spent,
			comment_effects.created_at
		FROM comment_effects
		INNER JOIN effects ON effects.id = comment_effects.effect_id
		INNER JOIN users ON users.id = comment_effects.user_id
		WHERE comment_effects.comment_id = ANY($1::uuid[])
		ORDER BY comment_effects.comment_id ASC, comment_effects.created_at DESC, comment_effects.id DESC
	`

	rows, err := repo.pool.Query(ctx, query, commentIDStrings(commentIDs))
	if err != nil {
		return nil, fmt.Errorf("list comment effects: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rawCommentID string
		var effect commentusecase.CommentEffectSummary
		var rawAppliedByUserID string
		var rawUsername string
		var rawDisplayName string
		var rawAvatarURL string
		var rawHeadline string
		if err := rows.Scan(
			&rawCommentID,
			&effect.ID,
			&effect.EffectID,
			&effect.Name,
			&effect.AssetURL,
			&effect.AnimationKey,
			&rawAppliedByUserID,
			&rawUsername,
			&rawDisplayName,
			&rawAvatarURL,
			&rawHeadline,
			&effect.PointsSpent,
			&effect.CreatedAt,
		); err != nil {
			return nil, err
		}
		commentID, err := commentdomain.NewCommentID(rawCommentID)
		if err != nil {
			return nil, fmt.Errorf("rehydrate comment effect comment id: %v", err)
		}
		effect.AppliedByUser = postusecase.UserSummary{
			ID:          rawAppliedByUserID,
			Username:    rawUsername,
			DisplayName: fallbackDisplayName(rawDisplayName, rawUsername),
			AvatarURL:   rawAvatarURL,
			Headline:    rawHeadline,
			Badges:      []string{},
		}
		result[commentID] = append(result[commentID], effect)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comment effects: %w", err)
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

func fallbackDisplayName(displayName string, username string) string {
	if displayName != "" {
		return displayName
	}
	return username
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

func (repo *PostgresCommentRepository) ReplaceCommentContentRefs(ctx context.Context, commentID commentdomain.CommentID, refs []postusecase.ContentRef, now time.Time) error {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace comment content refs: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `DELETE FROM comment_content_refs WHERE comment_id = $1::uuid`, commentID.String()); err != nil {
		return mapPostgresWriteError("delete comment content refs", err)
	}

	if len(refs) > 0 {
		batch := &pgx.Batch{}
		const query = `
			INSERT INTO comment_content_refs (
				comment_id,
				position,
				kind,
				ref_id,
				created_at
			)
			VALUES ($1::uuid, $2, $3, $4, $5)
		`
		for position, ref := range refs {
			batch.Queue(query, commentID.String(), position, ref.Kind, ref.RefID, now)
		}
		results := tx.SendBatch(ctx, batch)
		if err := results.Close(); err != nil {
			return mapPostgresWriteError("insert comment content refs", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit replace comment content refs: %w", err)
	}
	return nil
}

func (repo *PostgresCommentRepository) ListCommentContentRefsByCommentIDs(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID][]postusecase.ContentRef, error) {
	result := make(map[commentdomain.CommentID][]postusecase.ContentRef, len(commentIDs))
	if len(commentIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			comment_id::text,
			kind,
			ref_id
		FROM comment_content_refs
		WHERE comment_id = ANY($1::uuid[])
		ORDER BY comment_id ASC, position ASC
	`
	rows, err := repo.pool.Query(ctx, query, commentIDStrings(commentIDs))
	if err != nil {
		return nil, fmt.Errorf("list comment content refs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rawCommentID string
		var ref postusecase.ContentRef
		if err := rows.Scan(&rawCommentID, &ref.Kind, &ref.RefID); err != nil {
			return nil, err
		}
		commentID, err := commentdomain.NewCommentID(rawCommentID)
		if err != nil {
			return nil, fmt.Errorf("rehydrate comment content ref comment id: %v", err)
		}
		result[commentID] = append(result[commentID], ref)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comment content refs: %w", err)
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
			if strings.Contains(pgErr.ConstraintName, "content_refs") {
				return apperr.New(apperr.CodeInvalidArgument, "comment content refs are invalid")
			}
			if pgErr.ConstraintName == "comment_votes_value_check" || pgErr.ConstraintName == "comment_votes_updated_at_check" {
				return apperr.New(apperr.CodeInvalidArgument, "comment vote is invalid")
			}
			return apperr.New(apperr.CodeInvalidArgument, "comment is invalid")
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

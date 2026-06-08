package postrepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ postusecase.PostRepository = (*PostgresPostRepository)(nil)
var _ postusecase.PostMetadataRepository = (*PostgresPostRepository)(nil)
var _ postusecase.PostSaveRepository = (*PostgresPostRepository)(nil)

type PostgresPostRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPostRepository(pool *pgxpool.Pool) *PostgresPostRepository {
	return &PostgresPostRepository{
		pool: pool,
	}
}

func (repo *PostgresPostRepository) Create(ctx context.Context, post postdomain.Post) error {
	const query = `
		INSERT INTO posts (
			id,
			community_id,
			author_id,
			title,
			body,
			status,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8)
	`

	_, err := repo.pool.Exec(
		ctx,
		query,
		post.ID().String(),
		post.CommunityID().String(),
		post.AuthorID().String(),
		post.Title().String(),
		post.Body().String(),
		post.Status().String(),
		post.CreatedAt(),
		post.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("create post", err)
	}

	return nil
}

func (repo *PostgresPostRepository) FindVisibleByID(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
	const query = `
		SELECT
			id::text,
			community_id::text,
			author_id::text,
			title,
			body,
			status,
			created_at,
			updated_at
		FROM posts
		WHERE id = $1::uuid
			AND status = 'visible'
		LIMIT 1
	`

	post, err := scanPost(repo.pool.QueryRow(ctx, query, id.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeNotFound, "post not found")
		}
		return nil, err
	}

	return post, nil
}

func (repo *PostgresPostRepository) UpdateContent(ctx context.Context, post postdomain.Post) error {
	const query = `
		UPDATE posts
		SET
			title = $2,
			body = $3,
			updated_at = $4
		WHERE id = $1::uuid
			AND status = 'visible'
	`

	tag, err := repo.pool.Exec(
		ctx,
		query,
		post.ID().String(),
		post.Title().String(),
		post.Body().String(),
		post.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("update post content", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "post not found")
	}

	return nil
}

func (repo *PostgresPostRepository) MarkDeleted(ctx context.Context, post postdomain.Post) error {
	const query = `
		UPDATE posts
		SET
			status = 'deleted',
			updated_at = $2
		WHERE id = $1::uuid
			AND status = 'visible'
	`

	tag, err := repo.pool.Exec(ctx, query, post.ID().String(), post.UpdatedAt())
	if err != nil {
		return mapPostgresWriteError("delete post", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "post not found")
	}

	return nil
}

func (repo *PostgresPostRepository) ListVisibleByCommunity(ctx context.Context, communityID communitydomain.CommunityID, sort postusecase.PostListSort, limit int, offset int) ([]postdomain.Post, error) {
	return repo.listVisiblePosts(
		ctx,
		"list visible posts by community",
		"",
		"posts.community_id = $1::uuid AND posts.status = 'visible'",
		[]any{communityID.String()},
		sort,
		limit,
		offset,
	)
}

func (repo *PostgresPostRepository) ListVisibleInPublicCommunities(ctx context.Context, sort postusecase.PostListSort, limit int, offset int) ([]postdomain.Post, error) {
	return repo.listVisiblePosts(
		ctx,
		"list visible posts in public communities",
		"INNER JOIN communities ON communities.id = posts.community_id",
		"posts.status = 'visible' AND communities.status = 'active' AND communities.visibility = 'public'",
		nil,
		sort,
		limit,
		offset,
	)
}

func (repo *PostgresPostRepository) ListVisibleByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID, sort postusecase.PostListSort, limit int, offset int) ([]postdomain.Post, error) {
	return repo.listVisiblePosts(
		ctx,
		"list visible posts by author in public communities",
		"INNER JOIN communities ON communities.id = posts.community_id",
		"posts.author_id = $1::uuid AND posts.status = 'visible' AND communities.status = 'active' AND communities.visibility = 'public'",
		[]any{authorID.String()},
		sort,
		limit,
		offset,
	)
}

func (repo *PostgresPostRepository) listVisiblePosts(ctx context.Context, operation string, scopeJoin string, whereClause string, args []any, sort postusecase.PostListSort, limit int, offset int) ([]postdomain.Post, error) {
	limitPlaceholder := len(args) + 1
	offsetPlaceholder := len(args) + 2
	query := fmt.Sprintf(`
		SELECT
			posts.id::text,
			posts.community_id::text,
			posts.author_id::text,
			posts.title,
			posts.body,
			posts.status,
			posts.created_at,
			posts.updated_at
		FROM posts
		%s
		%s
		WHERE %s
		ORDER BY %s
		LIMIT $%d
		OFFSET $%d
	`, scopeJoin, postListStatsJoin(sort), whereClause, postListOrderBy(sort), limitPlaceholder, offsetPlaceholder)

	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := repo.pool.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	defer rows.Close()

	var posts []postdomain.Post
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, *post)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s: %w", operation, err)
	}

	return posts, nil
}

func postListStatsJoin(sort postusecase.PostListSort) string {
	if sort == postusecase.PostListSortNew {
		return ""
	}
	return `
		LEFT JOIN (
			SELECT
				post_id,
				COUNT(*)::int AS vote_count,
				COUNT(*) FILTER (WHERE value = 1)::int AS upvote_count,
				COUNT(*) FILTER (WHERE value = -1)::int AS downvote_count,
				COALESCE(SUM(value), 0)::int AS net_score,
				COUNT(*) FILTER (WHERE updated_at >= NOW() - INTERVAL '6 hours')::int AS recent_vote_count
			FROM post_votes
			GROUP BY post_id
		) AS post_vote_stats ON post_vote_stats.post_id = posts.id
		LEFT JOIN (
			SELECT
				post_id,
				COUNT(*)::int AS comment_count,
				COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '6 hours')::int AS recent_comment_count
			FROM comments
			WHERE status = 'visible'
			GROUP BY post_id
		) AS post_comment_stats ON post_comment_stats.post_id = posts.id
		LEFT JOIN (
			SELECT
				post_id,
				COUNT(*)::int AS save_count,
				COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '6 hours')::int AS recent_save_count
			FROM post_saves
			GROUP BY post_id
		) AS post_save_stats ON post_save_stats.post_id = posts.id
	`
}

func postListOrderBy(sort postusecase.PostListSort) string {
	switch sort {
	case postusecase.PostListSortBest:
		return `
			CASE
				WHEN COALESCE(post_vote_stats.vote_count, 0) = 0 THEN 0
				ELSE (
					(
						(COALESCE(post_vote_stats.upvote_count, 0)::double precision / COALESCE(post_vote_stats.vote_count, 0)::double precision)
						+ (1.9208 / COALESCE(post_vote_stats.vote_count, 0)::double precision)
						- (
							1.96 * sqrt(
								(
									(
										(COALESCE(post_vote_stats.upvote_count, 0)::double precision / COALESCE(post_vote_stats.vote_count, 0)::double precision)
										* (1 - (COALESCE(post_vote_stats.upvote_count, 0)::double precision / COALESCE(post_vote_stats.vote_count, 0)::double precision))
									)
									+ (0.9604 / COALESCE(post_vote_stats.vote_count, 0)::double precision)
								) / COALESCE(post_vote_stats.vote_count, 0)::double precision
							)
						)
					) / (1 + (3.8416 / COALESCE(post_vote_stats.vote_count, 0)::double precision))
				)
			END DESC,
			COALESCE(post_vote_stats.net_score, 0) DESC,
			posts.created_at DESC,
			posts.id DESC
		`
	case postusecase.PostListSortHot:
		return `
			(
				COALESCE(post_vote_stats.net_score, 0) * 3
				+ COALESCE(post_comment_stats.comment_count, 0) * 2
				+ COALESCE(post_save_stats.save_count, 0) * 4
			) / POWER(GREATEST(EXTRACT(EPOCH FROM (NOW() - posts.created_at)) / 3600.0, 0) + 2, 1.3) DESC,
			COALESCE(post_vote_stats.upvote_count, 0) DESC,
			posts.created_at DESC,
			posts.id DESC
		`
	case postusecase.PostListSortTop:
		return `
			COALESCE(post_vote_stats.net_score, 0) DESC,
			COALESCE(post_vote_stats.upvote_count, 0) DESC,
			posts.created_at DESC,
			posts.id DESC
		`
	case postusecase.PostListSortRising:
		return `
			(
				COALESCE(post_vote_stats.recent_vote_count, 0) * 3
				+ COALESCE(post_comment_stats.recent_comment_count, 0) * 2
				+ COALESCE(post_save_stats.recent_save_count, 0) * 4
			) / POWER(GREATEST(EXTRACT(EPOCH FROM (NOW() - posts.created_at)) / 3600.0, 0) + 0.5, 1.2) DESC,
			posts.created_at DESC,
			posts.id DESC
		`
	default:
		return "posts.created_at DESC, posts.id DESC"
	}
}

func (repo *PostgresPostRepository) LoadMetadataByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID]postusecase.PostMetadata, error) {
	result := make(map[postdomain.PostID]postusecase.PostMetadata, len(postIDs))
	if len(postIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			posts.id::text,
			users.id::text,
			users.username,
			communities.id::text,
			communities.slug,
			communities.name,
			communities.description,
			(
				SELECT COUNT(*)::int
				FROM comments
				WHERE comments.post_id = posts.id
					AND comments.status = 'visible'
			) AS comment_count,
			(
				SELECT COUNT(*)::int
				FROM posts AS community_posts
				WHERE community_posts.community_id = communities.id
					AND community_posts.status = 'visible'
			) AS community_post_count,
			(
				SELECT COUNT(*)::int
				FROM community_memberships
				WHERE community_memberships.community_id = communities.id
					AND community_memberships.status = 'active'
			) AS community_member_count
		FROM posts
		INNER JOIN users ON users.id = posts.author_id
		INNER JOIN communities ON communities.id = posts.community_id
		WHERE posts.id = ANY($1::uuid[])
	`

	rows, err := repo.pool.Query(ctx, query, postIDStrings(postIDs))
	if err != nil {
		return nil, fmt.Errorf("load post metadata: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rawPostID string
		var rawAuthorID string
		var rawUsername string
		var rawCommunityID string
		var rawCommunitySlug string
		var rawCommunityName string
		var rawCommunityDescription string
		var commentCount int
		var communityPostCount int
		var communityMemberCount int

		if err := rows.Scan(
			&rawPostID,
			&rawAuthorID,
			&rawUsername,
			&rawCommunityID,
			&rawCommunitySlug,
			&rawCommunityName,
			&rawCommunityDescription,
			&commentCount,
			&communityPostCount,
			&communityMemberCount,
		); err != nil {
			return nil, err
		}

		postID, err := postdomain.NewPostID(rawPostID)
		if err != nil {
			return nil, fmt.Errorf("rehydrate metadata post id: %v", err)
		}
		result[postID] = postusecase.PostMetadata{
			Author: postusecase.UserSummary{
				ID:          rawAuthorID,
				Username:    rawUsername,
				DisplayName: rawUsername,
				Badges:      []string{},
			},
			Community: postusecase.CommunitySummary{
				ID:          rawCommunityID,
				Slug:        rawCommunitySlug,
				Name:        rawCommunityName,
				Description: rawCommunityDescription,
				PostCount:   communityPostCount,
				MemberCount: communityMemberCount,
			},
			CommentCount: commentCount,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate post metadata: %w", err)
	}

	return result, nil
}

func postIDStrings(postIDs []postdomain.PostID) []string {
	rawIDs := make([]string, 0, len(postIDs))
	for _, postID := range postIDs {
		rawIDs = append(rawIDs, postID.String())
	}
	return rawIDs
}

func (repo *PostgresPostRepository) SavePost(ctx context.Context, postID postdomain.PostID, userID userdomain.UserID, now time.Time) error {
	const query = `
		INSERT INTO post_saves (
			post_id,
			user_id,
			created_at
		)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (post_id, user_id) DO NOTHING
	`

	if _, err := repo.pool.Exec(ctx, query, postID.String(), userID.String(), now); err != nil {
		return mapPostgresWriteError("save post", err)
	}
	return nil
}

func (repo *PostgresPostRepository) DeletePostSave(ctx context.Context, postID postdomain.PostID, userID userdomain.UserID) error {
	const query = `
		DELETE FROM post_saves
		WHERE post_id = $1::uuid
			AND user_id = $2::uuid
	`

	if _, err := repo.pool.Exec(ctx, query, postID.String(), userID.String()); err != nil {
		return mapPostgresWriteError("delete post save", err)
	}
	return nil
}

func (repo *PostgresPostRepository) ListSavedVisiblePosts(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]postdomain.Post, error) {
	const query = `
		SELECT
			posts.id::text,
			posts.community_id::text,
			posts.author_id::text,
			posts.title,
			posts.body,
			posts.status,
			posts.created_at,
			posts.updated_at
		FROM post_saves
		INNER JOIN posts ON posts.id = post_saves.post_id
		INNER JOIN communities ON communities.id = posts.community_id
		WHERE post_saves.user_id = $1::uuid
			AND posts.status = 'visible'
			AND communities.status = 'active'
			AND communities.visibility = 'public'
		ORDER BY post_saves.created_at DESC, posts.id DESC
		LIMIT $2
		OFFSET $3
	`

	rows, err := repo.pool.Query(ctx, query, userID.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list saved visible posts: %w", err)
	}
	defer rows.Close()

	var posts []postdomain.Post
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, *post)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate saved visible posts: %w", err)
	}

	return posts, nil
}

func (repo *PostgresPostRepository) FindSavedPostIDsByUser(ctx context.Context, postIDs []postdomain.PostID, userID userdomain.UserID) (map[postdomain.PostID]bool, error) {
	result := make(map[postdomain.PostID]bool, len(postIDs))
	if len(postIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT post_id::text
		FROM post_saves
		WHERE post_id = ANY($1::uuid[])
			AND user_id = $2::uuid
	`

	rows, err := repo.pool.Query(ctx, query, postIDStrings(postIDs), userID.String())
	if err != nil {
		return nil, fmt.Errorf("find saved post ids by user: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rawPostID string
		if err := rows.Scan(&rawPostID); err != nil {
			return nil, err
		}
		postID, err := postdomain.NewPostID(rawPostID)
		if err != nil {
			return nil, fmt.Errorf("rehydrate saved post id: %v", err)
		}
		result[postID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate saved post ids: %w", err)
	}

	return result, nil
}

func (repo *PostgresPostRepository) SummarizeSavesByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID]int, error) {
	result := make(map[postdomain.PostID]int, len(postIDs))
	if len(postIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			post_id::text,
			COUNT(*)::int
		FROM post_saves
		WHERE post_id = ANY($1::uuid[])
		GROUP BY post_id
	`

	rows, err := repo.pool.Query(ctx, query, postIDStrings(postIDs))
	if err != nil {
		return nil, fmt.Errorf("summarize post saves: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rawPostID string
		var saveCount int
		if err := rows.Scan(&rawPostID, &saveCount); err != nil {
			return nil, err
		}
		postID, err := postdomain.NewPostID(rawPostID)
		if err != nil {
			return nil, fmt.Errorf("rehydrate post save summary id: %v", err)
		}
		result[postID] = saveCount
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate post save summaries: %w", err)
	}

	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPost(row rowScanner) (*postdomain.Post, error) {
	var rawID string
	var rawCommunityID string
	var rawAuthorID string
	var rawTitle string
	var rawBody string
	var rawStatus string
	var createdAt time.Time
	var updatedAt time.Time

	if err := row.Scan(
		&rawID,
		&rawCommunityID,
		&rawAuthorID,
		&rawTitle,
		&rawBody,
		&rawStatus,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	id, err := postdomain.NewPostID(rawID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate post id: %v", err)
	}
	communityID, err := communitydomain.NewCommunityID(rawCommunityID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate post community id: %v", err)
	}
	authorID, err := userdomain.NewUserID(rawAuthorID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate post author id: %v", err)
	}
	title, err := postdomain.NewPostTitle(rawTitle)
	if err != nil {
		return nil, fmt.Errorf("rehydrate post title: %v", err)
	}
	body, err := postdomain.NewPostBody(rawBody)
	if err != nil {
		return nil, fmt.Errorf("rehydrate post body: %v", err)
	}
	status, err := postdomain.NewPostStatus(rawStatus)
	if err != nil {
		return nil, fmt.Errorf("rehydrate post status: %v", err)
	}

	post, err := postdomain.RehydratePost(id, communityID, authorID, title, body, status, createdAt, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("rehydrate post: %v", err)
	}

	return post, nil
}

func mapPostgresWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return apperr.New(apperr.CodeConflict, "post already exists")
		}
		if pgErr.Code == "23503" {
			return apperr.New(apperr.CodeNotFound, "related record not found")
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

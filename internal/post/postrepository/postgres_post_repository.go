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
	if sort == postusecase.PostListSortHot {
		return repo.listVisibleByCommunityHot(ctx, communityID, limit, offset)
	}
	return repo.listVisibleByCommunityNew(ctx, communityID, limit, offset)
}

func (repo *PostgresPostRepository) listVisibleByCommunityNew(ctx context.Context, communityID communitydomain.CommunityID, limit int, offset int) ([]postdomain.Post, error) {
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
		WHERE community_id = $1::uuid
			AND status = 'visible'
		ORDER BY created_at DESC, id DESC
		LIMIT $2
		OFFSET $3
	`

	rows, err := repo.pool.Query(ctx, query, communityID.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list visible posts by community: %w", err)
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
		return nil, fmt.Errorf("iterate visible posts by community: %w", err)
	}

	return posts, nil
}

func (repo *PostgresPostRepository) listVisibleByCommunityHot(ctx context.Context, communityID communitydomain.CommunityID, limit int, offset int) ([]postdomain.Post, error) {
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
		FROM posts
		LEFT JOIN post_votes ON post_votes.post_id = posts.id
		WHERE posts.community_id = $1::uuid
			AND posts.status = 'visible'
		GROUP BY
			posts.id,
			posts.community_id,
			posts.author_id,
			posts.title,
			posts.body,
			posts.status,
			posts.created_at,
			posts.updated_at
		ORDER BY
			COALESCE(SUM(post_votes.value), 0) DESC,
			COUNT(post_votes.value) FILTER (WHERE post_votes.value = 1) DESC,
			posts.created_at DESC,
			posts.id DESC
		LIMIT $2
		OFFSET $3
	`

	rows, err := repo.pool.Query(ctx, query, communityID.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list visible hot posts by community: %w", err)
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
		return nil, fmt.Errorf("iterate visible hot posts by community: %w", err)
	}

	return posts, nil
}

func (repo *PostgresPostRepository) ListVisibleInPublicCommunities(ctx context.Context, sort postusecase.PostListSort, limit int, offset int) ([]postdomain.Post, error) {
	if sort == postusecase.PostListSortHot {
		return repo.listVisibleInPublicCommunitiesHot(ctx, limit, offset)
	}
	return repo.listVisibleInPublicCommunitiesNew(ctx, limit, offset)
}

func (repo *PostgresPostRepository) listVisibleInPublicCommunitiesNew(ctx context.Context, limit int, offset int) ([]postdomain.Post, error) {
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
		FROM posts
		INNER JOIN communities ON communities.id = posts.community_id
		WHERE posts.status = 'visible'
			AND communities.status = 'active'
			AND communities.visibility = 'public'
		ORDER BY posts.created_at DESC, posts.id DESC
		LIMIT $1
		OFFSET $2
	`

	rows, err := repo.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list visible posts in public communities: %w", err)
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
		return nil, fmt.Errorf("iterate visible posts in public communities: %w", err)
	}

	return posts, nil
}

func (repo *PostgresPostRepository) listVisibleInPublicCommunitiesHot(ctx context.Context, limit int, offset int) ([]postdomain.Post, error) {
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
		FROM posts
		INNER JOIN communities ON communities.id = posts.community_id
		LEFT JOIN post_votes ON post_votes.post_id = posts.id
		WHERE posts.status = 'visible'
			AND communities.status = 'active'
			AND communities.visibility = 'public'
		GROUP BY
			posts.id,
			posts.community_id,
			posts.author_id,
			posts.title,
			posts.body,
			posts.status,
			posts.created_at,
			posts.updated_at
		ORDER BY
			COALESCE(SUM(post_votes.value), 0) DESC,
			COUNT(post_votes.value) FILTER (WHERE post_votes.value = 1) DESC,
			posts.created_at DESC,
			posts.id DESC
		LIMIT $1
		OFFSET $2
	`

	rows, err := repo.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list visible hot posts in public communities: %w", err)
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
		return nil, fmt.Errorf("iterate visible hot posts in public communities: %w", err)
	}

	return posts, nil
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

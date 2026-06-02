package searchrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/Versifine/cumt-nexus-api/internal/search/searchusecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ searchusecase.Repository = (*PostgresSearchRepository)(nil)

type PostgresSearchRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSearchRepository(pool *pgxpool.Pool) *PostgresSearchRepository {
	return &PostgresSearchRepository{
		pool: pool,
	}
}

func (repo *PostgresSearchRepository) SearchCommunities(ctx context.Context, query string, limit int, offset int) ([]searchusecase.CommunityResult, error) {
	const sql = `
		SELECT
			id::text,
			slug,
			name,
			description,
			kind,
			status,
			visibility,
			created_at,
			updated_at
		FROM communities
		WHERE status = 'active'
			AND visibility = 'public'
			AND name ILIKE $1 ESCAPE '\'
		ORDER BY name ASC, id ASC
		LIMIT $2
		OFFSET $3
	`

	rows, err := repo.pool.Query(ctx, sql, likePattern(query), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("search communities: %w", err)
	}
	defer rows.Close()

	results := make([]searchusecase.CommunityResult, 0)
	for rows.Next() {
		var result searchusecase.CommunityResult
		if err := rows.Scan(
			&result.ID,
			&result.Slug,
			&result.Name,
			&result.Description,
			&result.Kind,
			&result.Status,
			&result.Visibility,
			&result.CreatedAt,
			&result.UpdatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate community search results: %w", err)
	}
	return results, nil
}

func (repo *PostgresSearchRepository) SearchPosts(ctx context.Context, query string, limit int, offset int) ([]searchusecase.PostResult, error) {
	const sql = `
		SELECT
			posts.id::text,
			posts.community_id::text,
			communities.slug,
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
			AND (
				posts.title ILIKE $1 ESCAPE '\'
				OR posts.body ILIKE $1 ESCAPE '\'
			)
		ORDER BY posts.created_at DESC, posts.id DESC
		LIMIT $2
		OFFSET $3
	`

	rows, err := repo.pool.Query(ctx, sql, likePattern(query), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("search posts: %w", err)
	}
	defer rows.Close()

	results := make([]searchusecase.PostResult, 0)
	for rows.Next() {
		var result searchusecase.PostResult
		var body string
		if err := rows.Scan(
			&result.ID,
			&result.CommunityID,
			&result.CommunitySlug,
			&result.AuthorID,
			&result.Title,
			&body,
			&result.Status,
			&result.CreatedAt,
			&result.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result.BodyExcerpt = excerptBody(body)
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate post search results: %w", err)
	}
	return results, nil
}

func likePattern(query string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + replacer.Replace(query) + "%"
}

func excerptBody(raw string) string {
	const maxRunes = 160
	trimmed := strings.TrimSpace(raw)
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return string(runes[:maxRunes]) + "..."
}

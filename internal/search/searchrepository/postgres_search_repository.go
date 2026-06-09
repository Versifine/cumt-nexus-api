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
		WITH search_query AS (
			SELECT websearch_to_tsquery('simple', $1) AS query
		),
		ranked_communities AS (
			SELECT
				communities.id,
				communities.slug,
				communities.name,
				communities.description,
				communities.kind,
				communities.status,
				communities.visibility,
				communities.created_at,
				communities.updated_at,
				ts_rank_cd(community_search.document, search_query.query) AS rank_score,
				1.0 / (1.0 + GREATEST(EXTRACT(EPOCH FROM (NOW() - communities.updated_at)), 0) / 604800.0) AS recency_score
			FROM communities
			CROSS JOIN search_query
			CROSS JOIN LATERAL (
				SELECT
					setweight(to_tsvector('simple', COALESCE(communities.name, '')), 'A') ||
					setweight(to_tsvector('simple', COALESCE(communities.description, '')), 'B') AS document
			) AS community_search
			WHERE communities.status = 'active'
				AND communities.visibility = 'public'
				AND community_search.document @@ search_query.query
		)
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
		FROM ranked_communities
		ORDER BY rank_score DESC, recency_score DESC, updated_at DESC, id ASC
		LIMIT $2
		OFFSET $3
	`

	rows, err := repo.pool.Query(ctx, sql, query, limit, offset)
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
		WITH search_query AS (
			SELECT websearch_to_tsquery('simple', $1) AS query
		),
		ranked_posts AS (
			SELECT
				posts.id,
				posts.community_id,
				communities.slug AS community_slug,
				posts.author_id,
				posts.title,
				posts.body,
				posts.status,
				posts.created_at,
				posts.updated_at,
				ts_rank_cd(post_search.document, search_query.query) AS rank_score,
				1.0 / (1.0 + GREATEST(EXTRACT(EPOCH FROM (NOW() - posts.created_at)), 0) / 604800.0) AS recency_score
			FROM posts
			INNER JOIN communities ON communities.id = posts.community_id
			CROSS JOIN search_query
			CROSS JOIN LATERAL (
				SELECT
					setweight(to_tsvector('simple', COALESCE(posts.title, '')), 'A') ||
					setweight(to_tsvector('simple', COALESCE(posts.body, '')), 'C') AS post_document,
					setweight(to_tsvector('simple', COALESCE(communities.name, '')), 'B') ||
					setweight(to_tsvector('simple', COALESCE(communities.slug, '')), 'B') AS community_document,
					setweight(to_tsvector('simple', COALESCE(posts.title, '')), 'A') ||
					setweight(to_tsvector('simple', COALESCE(communities.name, '')), 'B') ||
					setweight(to_tsvector('simple', COALESCE(communities.slug, '')), 'B') ||
					setweight(to_tsvector('simple', COALESCE(posts.body, '')), 'C') AS document
			) AS post_search
			WHERE posts.status = 'visible'
				AND communities.status = 'active'
				AND communities.visibility = 'public'
				AND (
					post_search.post_document @@ search_query.query
					OR post_search.community_document @@ search_query.query
				)
		)
		SELECT
			id::text,
			community_id::text,
			community_slug,
			author_id::text,
			title,
			body,
			status,
			created_at,
			updated_at
		FROM ranked_posts
		ORDER BY rank_score DESC, recency_score DESC, created_at DESC, id DESC
		LIMIT $2
		OFFSET $3
	`

	rows, err := repo.pool.Query(ctx, sql, query, limit, offset)
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

func excerptBody(raw string) string {
	const maxRunes = 160
	trimmed := strings.TrimSpace(raw)
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return string(runes[:maxRunes]) + "..."
}

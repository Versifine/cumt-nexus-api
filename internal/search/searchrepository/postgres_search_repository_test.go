package searchrepository

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/search/searchusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresSearchRepositorySearchCommunities(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresSearchRepository(pool)

	creatorID := insertTestUser(ctx, t, pool)
	matching := insertTestCommunity(ctx, t, pool, creatorID, "Campus Search "+randomSuffix(), "active")
	suspended := insertTestCommunity(ctx, t, pool, creatorID, "Campus Suspended "+randomSuffix(), "suspended")

	results, err := repo.SearchCommunities(ctx, "campus", 20, 0)
	if err != nil {
		t.Fatalf("SearchCommunities returned error: %v", err)
	}

	if !containsCommunity(results, matching.String()) {
		t.Fatalf("expected results to include active matching community %q, got %#v", matching.String(), results)
	}
	if containsCommunity(results, suspended.String()) {
		t.Fatalf("expected results to exclude suspended community %q, got %#v", suspended.String(), results)
	}
}

func TestPostgresSearchRepositorySearchPosts(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresSearchRepository(pool)

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "Search Posts "+randomSuffix(), "active")
	titleMatch := insertTestPost(ctx, t, pool, communityID, authorID, "Needle title", "ordinary body", "visible")
	bodyMatch := insertTestPost(ctx, t, pool, communityID, authorID, "ordinary title", "body has needle", "visible")
	removed := insertTestPost(ctx, t, pool, communityID, authorID, "Needle removed", "body", "removed")

	results, err := repo.SearchPosts(ctx, "needle", 20, 0)
	if err != nil {
		t.Fatalf("SearchPosts returned error: %v", err)
	}

	if !containsPost(results, titleMatch.String()) {
		t.Fatalf("expected title match %q, got %#v", titleMatch.String(), results)
	}
	if !containsPost(results, bodyMatch.String()) {
		t.Fatalf("expected body match %q, got %#v", bodyMatch.String(), results)
	}
	if containsPost(results, removed.String()) {
		t.Fatalf("expected removed post %q to be excluded, got %#v", removed.String(), results)
	}
}

func TestPostgresSearchRepositoryRanksPostTitleAboveBody(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresSearchRepository(pool)

	term := "ranktitle" + randomSuffix()
	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "Search Ranking "+randomSuffix(), "active")
	titleMatch := insertTestPostAt(ctx, t, pool, communityID, authorID, "Question about "+term, "ordinary body", "visible", testNow().Add(-2*time.Hour))
	bodyMatch := insertTestPostAt(ctx, t, pool, communityID, authorID, "ordinary title", "body has "+term, "visible", testNow())

	results, err := repo.SearchPosts(ctx, term, 20, 0)
	if err != nil {
		t.Fatalf("SearchPosts returned error: %v", err)
	}

	titleIndex := postIndex(results, titleMatch.String())
	bodyIndex := postIndex(results, bodyMatch.String())
	if titleIndex < 0 || bodyIndex < 0 {
		t.Fatalf("expected both title/body matches, got %#v", results)
	}
	if titleIndex > bodyIndex {
		t.Fatalf("expected title match to rank before body match, got title=%d body=%d results=%#v", titleIndex, bodyIndex, results)
	}
}

func TestPostgresSearchRepositoryRanksCommunityNameAbovePostBody(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresSearchRepository(pool)

	term := "rankcommunity" + randomSuffix()
	authorID := insertTestUser(ctx, t, pool)
	communityNameMatchID := insertTestCommunity(ctx, t, pool, authorID, "Community "+term, "active")
	bodyMatchCommunityID := insertTestCommunity(ctx, t, pool, authorID, "Search Ranking "+randomSuffix(), "active")
	communityNameMatch := insertTestPostAt(ctx, t, pool, communityNameMatchID, authorID, "ordinary title", "ordinary body", "visible", testNow().Add(-2*time.Hour))
	bodyMatch := insertTestPostAt(ctx, t, pool, bodyMatchCommunityID, authorID, "ordinary title", "body has "+term, "visible", testNow())

	results, err := repo.SearchPosts(ctx, term, 20, 0)
	if err != nil {
		t.Fatalf("SearchPosts returned error: %v", err)
	}

	communityIndex := postIndex(results, communityNameMatch.String())
	bodyIndex := postIndex(results, bodyMatch.String())
	if communityIndex < 0 || bodyIndex < 0 {
		t.Fatalf("expected both community/body matches, got %#v", results)
	}
	if communityIndex > bodyIndex {
		t.Fatalf("expected community name match to rank before body match, got community=%d body=%d results=%#v", communityIndex, bodyIndex, results)
	}
}

func TestPostgresSearchRepositoryHandlesPunctuationQueries(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresSearchRepository(pool)

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "Wildcard "+randomSuffix(), "active")
	matching := insertTestPost(ctx, t, pool, communityID, authorID, "literal 100% match", "body", "visible")
	nonMatching := insertTestPost(ctx, t, pool, communityID, authorID, "literal 1000 match", "body", "visible")

	results, err := repo.SearchPosts(ctx, "100%", 20, 0)
	if err != nil {
		t.Fatalf("SearchPosts returned error: %v", err)
	}

	if !containsPost(results, matching.String()) {
		t.Fatalf("expected literal percent match %q, got %#v", matching.String(), results)
	}
	if containsPost(results, nonMatching.String()) {
		t.Fatalf("expected non literal match %q to be excluded, got %#v", nonMatching.String(), results)
	}
}

func newTestPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	pool, err := db.Open(ctx, testPostgresConfig())
	if err != nil {
		t.Skipf("skip repository integration test: open postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.Ping(ctx, pool); err != nil {
		t.Skipf("skip repository integration test: ping postgres: %v", err)
	}

	requireSearchSchema(ctx, t, pool)
	return ctx, pool
}

func requireSearchSchema(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	for _, table := range []string{"users", "communities", "posts"} {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'public'
					AND table_name = $1
			)
		`, table).Scan(&exists); err != nil {
			t.Fatalf("check table %s exists: %v", table, err)
		}
		if !exists {
			t.Skipf("%s table does not exist; run go run ./cmd/migrate up before repository tests", table)
		}
	}
	for _, indexName := range []string{"communities_public_search_fts_idx", "posts_visible_search_fts_idx"} {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_indexes
				WHERE schemaname = 'public'
					AND indexname = $1
			)
		`, indexName).Scan(&exists); err != nil {
			t.Fatalf("check index %s exists: %v", indexName, err)
		}
		if !exists {
			t.Skipf("%s index does not exist; run go run ./cmd/migrate up before repository tests", indexName)
		}
	}
}

func testPostgresConfig() config.PostgresConfig {
	return config.PostgresConfig{
		Host:            envString("POSTGRES_HOST", "localhost"),
		Port:            envInt("POSTGRES_PORT", 5432),
		User:            envString("POSTGRES_USER", "postgres"),
		Password:        envString("POSTGRES_PASSWORD", "postgres"),
		Database:        envString("POSTGRES_DATABASE", "cumt_nexus"),
		SSLMode:         envString("POSTGRES_SSL_MODE", "disable"),
		MaxConns:        5,
		MaxConnLifetime: time.Minute,
		MaxConnIdleTime: time.Minute,
	}
}

func envString(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func insertTestUser(ctx context.Context, t *testing.T, pool *pgxpool.Pool) userdomain.UserID {
	t.Helper()

	id := userdomain.NewGeneratedUserID()
	username := "search_repo_" + randomSuffix()
	_, err := pool.Exec(ctx, `
		INSERT INTO users (
			id,
			username,
			password_hash,
			status,
			is_platform_staff,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2, $3, 'active', false, $4, $4)
	`, id.String(), username, "hashed-password-"+username, testNow())
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup test user %q: %v", id.String(), err)
		}
	})

	return id
}

func insertTestCommunity(ctx context.Context, t *testing.T, pool *pgxpool.Pool, createdBy userdomain.UserID, name string, status string) communitydomain.CommunityID {
	t.Helper()

	id := communitydomain.NewGeneratedCommunityID()
	slug := "search-" + randomSuffix()
	_, err := pool.Exec(ctx, `
		INSERT INTO communities (
			id,
			slug,
			name,
			description,
			kind,
			status,
			visibility,
			created_by,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2, $3, '', 'user_created', $4, 'public', $5::uuid, $6, $6)
	`, id.String(), slug, name, status, createdBy.String(), testNow())
	if err != nil {
		t.Fatalf("insert test community: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM communities WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup test community %q: %v", id.String(), err)
		}
	})

	return id
}

func insertTestPost(ctx context.Context, t *testing.T, pool *pgxpool.Pool, communityID communitydomain.CommunityID, authorID userdomain.UserID, title string, body string, status string) postdomain.PostID {
	t.Helper()

	return insertTestPostAt(ctx, t, pool, communityID, authorID, title, body, status, testNow())
}

func insertTestPostAt(ctx context.Context, t *testing.T, pool *pgxpool.Pool, communityID communitydomain.CommunityID, authorID userdomain.UserID, title string, body string, status string, createdAt time.Time) postdomain.PostID {
	t.Helper()

	id := postdomain.NewGeneratedPostID()
	_, err := pool.Exec(ctx, `
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
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $7)
	`, id.String(), communityID.String(), authorID.String(), title, body, status, createdAt)
	if err != nil {
		t.Fatalf("insert test post: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM posts WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup test post %q: %v", id.String(), err)
		}
	})

	return id
}

func containsCommunity(results []searchusecase.CommunityResult, id string) bool {
	for _, result := range results {
		if result.ID == id {
			return true
		}
	}
	return false
}

func containsPost(results []searchusecase.PostResult, id string) bool {
	for _, result := range results {
		if result.ID == id {
			return true
		}
	}
	return false
}

func postIndex(results []searchusecase.PostResult, id string) int {
	for index, result := range results {
		if result.ID == id {
			return index
		}
	}
	return -1
}

func testNow() time.Time {
	return time.Date(2026, 6, 2, 6, 0, 0, 0, time.UTC)
}

func randomSuffix() string {
	return strings.ReplaceAll(uuid.NewString()[:8], "-", "")
}

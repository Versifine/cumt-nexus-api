package postrepository

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresPostRepositoryCreateFindListAndNotFound(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "post-"+randomSuffix())
	otherCommunityID := insertTestCommunity(ctx, t, pool, authorID, "post-"+randomSuffix())

	olderPost := mustPost(t, communityID, authorID, "Older", now)
	newerPost := mustPost(t, communityID, authorID, "Newer", now.Add(time.Minute))
	otherPost := mustPost(t, otherCommunityID, authorID, "Other", now.Add(2*time.Minute))

	if err := repo.Create(ctx, *olderPost); err != nil {
		t.Fatalf("Create older post returned error: %v", err)
	}
	cleanupPost(ctx, t, pool, olderPost.ID())
	if err := repo.Create(ctx, *newerPost); err != nil {
		t.Fatalf("Create newer post returned error: %v", err)
	}
	cleanupPost(ctx, t, pool, newerPost.ID())
	if err := repo.Create(ctx, *otherPost); err != nil {
		t.Fatalf("Create other post returned error: %v", err)
	}
	cleanupPost(ctx, t, pool, otherPost.ID())

	got, err := repo.FindVisibleByID(ctx, olderPost.ID())
	if err != nil {
		t.Fatalf("FindVisibleByID returned error: %v", err)
	}
	if got.ID() != olderPost.ID() || got.Title() != olderPost.Title() {
		t.Fatalf("unexpected post: got id=%q title=%q", got.ID().String(), got.Title().String())
	}

	posts, err := repo.ListVisibleByCommunity(ctx, communityID, 20, 0)
	if err != nil {
		t.Fatalf("ListVisibleByCommunity returned error: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected two posts, got %d", len(posts))
	}
	if posts[0].ID() != newerPost.ID() || posts[1].ID() != olderPost.ID() {
		t.Fatalf("expected newest-first ordering, got %#v", []postdomain.PostID{posts[0].ID(), posts[1].ID()})
	}

	if _, err := repo.FindVisibleByID(ctx, postdomain.NewGeneratedPostID()); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing post, got %v", err)
	}
}

func TestPostgresPostRepositoryListVisibleInPublicCommunities(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "latest-"+randomSuffix())
	otherCommunityID := insertTestCommunity(ctx, t, pool, authorID, "latest-"+randomSuffix())
	suspendedCommunityID := insertTestCommunityWithStatus(ctx, t, pool, authorID, "latest-"+randomSuffix(), "suspended")

	olderPost := mustPost(t, communityID, authorID, "Older latest", now)
	newerPost := mustPost(t, otherCommunityID, authorID, "Newer latest", now.Add(time.Minute))
	suspendedPost := mustPost(t, suspendedCommunityID, authorID, "Suspended latest", now.Add(2*time.Minute))
	removedPost := mustPostWithStatus(t, communityID, authorID, "Removed latest", postdomain.PostStatusRemoved, now.Add(3*time.Minute))

	for _, post := range []*postdomain.Post{olderPost, newerPost, suspendedPost, removedPost} {
		if err := repo.Create(ctx, *post); err != nil {
			t.Fatalf("Create post %q returned error: %v", post.Title().String(), err)
		}
		cleanupPost(ctx, t, pool, post.ID())
	}

	posts, err := repo.ListVisibleInPublicCommunities(ctx, 20, 0)
	if err != nil {
		t.Fatalf("ListVisibleInPublicCommunities returned error: %v", err)
	}

	var gotIDs []postdomain.PostID
	for _, post := range posts {
		if post.ID() == newerPost.ID() || post.ID() == olderPost.ID() || post.ID() == suspendedPost.ID() || post.ID() == removedPost.ID() {
			gotIDs = append(gotIDs, post.ID())
		}
	}

	if len(gotIDs) != 2 {
		t.Fatalf("expected two visible public posts from this test, got %#v", gotIDs)
	}
	if gotIDs[0] != newerPost.ID() || gotIDs[1] != olderPost.ID() {
		t.Fatalf("expected newest-first visible public posts, got %#v", gotIDs)
	}
}

func TestPostgresPostRepositoryMapsForeignKeyFailure(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	post := mustPost(t, communitydomain.NewGeneratedCommunityID(), authorID, "Missing community", now)
	if err := repo.Create(ctx, *post); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing related community, got %v", err)
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

	requirePostSchema(ctx, t, pool)

	return ctx, pool
}

func requirePostSchema(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
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
	username := "post_repo_" + randomSuffix()

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

func insertTestCommunity(ctx context.Context, t *testing.T, pool *pgxpool.Pool, createdBy userdomain.UserID, rawSlug string) communitydomain.CommunityID {
	return insertTestCommunityWithStatus(ctx, t, pool, createdBy, rawSlug, "active")
}

func insertTestCommunityWithStatus(ctx context.Context, t *testing.T, pool *pgxpool.Pool, createdBy userdomain.UserID, rawSlug string, status string) communitydomain.CommunityID {
	t.Helper()

	id := communitydomain.NewGeneratedCommunityID()
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
	`, id.String(), rawSlug, "Post Repo "+rawSlug, status, createdBy.String(), testNow())
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

func cleanupPost(ctx context.Context, t *testing.T, pool *pgxpool.Pool, id postdomain.PostID) {
	t.Helper()

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM posts WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup post %q: %v", id.String(), err)
		}
	})
}

func mustPost(t *testing.T, communityID communitydomain.CommunityID, authorID userdomain.UserID, title string, now time.Time) *postdomain.Post {
	return mustPostWithStatus(t, communityID, authorID, title, postdomain.PostStatusVisible, now)
}

func mustPostWithStatus(t *testing.T, communityID communitydomain.CommunityID, authorID userdomain.UserID, title string, status postdomain.PostStatus, now time.Time) *postdomain.Post {
	t.Helper()

	postTitle, err := postdomain.NewPostTitle(title)
	if err != nil {
		t.Fatalf("NewPostTitle returned error: %v", err)
	}
	body, err := postdomain.NewPostBody("Body for " + title)
	if err != nil {
		t.Fatalf("NewPostBody returned error: %v", err)
	}
	post, err := postdomain.RehydratePost(postdomain.NewGeneratedPostID(), communityID, authorID, postTitle, body, status, now, now)
	if err != nil {
		t.Fatalf("NewPost returned error: %v", err)
	}
	return post
}

func testNow() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

func randomSuffix() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
}

func hasAppCode(err error, code apperr.Code) bool {
	if err == nil {
		return false
	}

	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Code() == code
}

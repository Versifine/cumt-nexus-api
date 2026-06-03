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
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
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

	posts, err := repo.ListVisibleByCommunity(ctx, communityID, postusecase.PostListSortNew, 20, 0)
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

	posts, err := repo.ListVisibleInPublicCommunities(ctx, postusecase.PostListSortNew, 20, 0)
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

func TestPostgresPostRepositoryUpdateContentAndMarkDeleted(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "post-update-"+randomSuffix())
	post := mustPost(t, communityID, authorID, "Original update", now)
	if err := repo.Create(ctx, *post); err != nil {
		t.Fatalf("Create post returned error: %v", err)
	}
	cleanupPost(ctx, t, pool, post.ID())

	if err := post.Edit(mustPostTitle(t, "Updated update"), mustPostBody(t, "Updated body"), now.Add(time.Minute)); err != nil {
		t.Fatalf("Edit post returned error: %v", err)
	}
	if err := repo.UpdateContent(ctx, *post); err != nil {
		t.Fatalf("UpdateContent returned error: %v", err)
	}

	updated, err := repo.FindVisibleByID(ctx, post.ID())
	if err != nil {
		t.Fatalf("FindVisibleByID after update returned error: %v", err)
	}
	if updated.Title().String() != "Updated update" || updated.Body().String() != "Updated body" {
		t.Fatalf("unexpected updated content: title=%q body=%q", updated.Title().String(), updated.Body().String())
	}

	if err := post.MarkDeleted(now.Add(2 * time.Minute)); err != nil {
		t.Fatalf("MarkDeleted returned error: %v", err)
	}
	if err := repo.MarkDeleted(ctx, *post); err != nil {
		t.Fatalf("MarkDeleted repository returned error: %v", err)
	}
	if _, err := repo.FindVisibleByID(ctx, post.ID()); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found after delete, got %v", err)
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM posts WHERE id = $1::uuid`, post.ID().String()).Scan(&status); err != nil {
		t.Fatalf("query deleted post status: %v", err)
	}
	if status != postdomain.PostStatusDeleted.String() {
		t.Fatalf("expected deleted status, got %q", status)
	}
}

func TestPostgresPostRepositoryListVisibleByCommunityHotSort(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "hot-community-"+randomSuffix())

	simplePost := mustPost(t, communityID, authorID, "Simple hot", now.Add(2*time.Minute))
	balancedPost := mustPost(t, communityID, authorID, "Balanced hot", now)
	coldPost := mustPost(t, communityID, authorID, "Cold hot", now.Add(3*time.Minute))

	for _, post := range []*postdomain.Post{simplePost, balancedPost, coldPost} {
		if err := repo.Create(ctx, *post); err != nil {
			t.Fatalf("Create post %q returned error: %v", post.Title().String(), err)
		}
		cleanupPost(ctx, t, pool, post.ID())
	}

	insertTestPostVote(ctx, t, pool, simplePost.ID(), insertTestUser(ctx, t, pool), 1)
	insertTestPostVote(ctx, t, pool, balancedPost.ID(), insertTestUser(ctx, t, pool), 1)
	insertTestPostVote(ctx, t, pool, balancedPost.ID(), insertTestUser(ctx, t, pool), 1)
	insertTestPostVote(ctx, t, pool, balancedPost.ID(), insertTestUser(ctx, t, pool), -1)

	posts, err := repo.ListVisibleByCommunity(ctx, communityID, postusecase.PostListSortHot, 20, 0)
	if err != nil {
		t.Fatalf("ListVisibleByCommunity hot returned error: %v", err)
	}
	if len(posts) != 3 {
		t.Fatalf("expected three posts, got %d", len(posts))
	}

	gotIDs := []postdomain.PostID{posts[0].ID(), posts[1].ID(), posts[2].ID()}
	wantIDs := []postdomain.PostID{balancedPost.ID(), simplePost.ID(), coldPost.ID()}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("expected hot order %#v, got %#v", wantIDs, gotIDs)
		}
	}
}

func TestPostgresPostRepositoryListVisibleInPublicCommunitiesHotSort(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "global-hot-"+randomSuffix())
	suspendedCommunityID := insertTestCommunityWithStatus(ctx, t, pool, authorID, "global-hot-"+randomSuffix(), "suspended")

	hotPost := mustPost(t, communityID, authorID, "Global hot", now)
	warmPost := mustPost(t, communityID, authorID, "Global warm", now.Add(time.Minute))
	suspendedPost := mustPost(t, suspendedCommunityID, authorID, "Suspended hot", now.Add(2*time.Minute))
	removedPost := mustPostWithStatus(t, communityID, authorID, "Removed hot", postdomain.PostStatusRemoved, now.Add(3*time.Minute))

	for _, post := range []*postdomain.Post{hotPost, warmPost, suspendedPost, removedPost} {
		if err := repo.Create(ctx, *post); err != nil {
			t.Fatalf("Create post %q returned error: %v", post.Title().String(), err)
		}
		cleanupPost(ctx, t, pool, post.ID())
	}

	for i := 0; i < 8; i++ {
		insertTestPostVote(ctx, t, pool, hotPost.ID(), insertTestUser(ctx, t, pool), 1)
	}
	for i := 0; i < 3; i++ {
		insertTestPostVote(ctx, t, pool, warmPost.ID(), insertTestUser(ctx, t, pool), 1)
	}
	for i := 0; i < 10; i++ {
		insertTestPostVote(ctx, t, pool, suspendedPost.ID(), insertTestUser(ctx, t, pool), 1)
		insertTestPostVote(ctx, t, pool, removedPost.ID(), insertTestUser(ctx, t, pool), 1)
	}

	posts, err := repo.ListVisibleInPublicCommunities(ctx, postusecase.PostListSortHot, 200, 0)
	if err != nil {
		t.Fatalf("ListVisibleInPublicCommunities hot returned error: %v", err)
	}

	var gotIDs []postdomain.PostID
	for _, post := range posts {
		switch post.ID() {
		case hotPost.ID(), warmPost.ID(), suspendedPost.ID(), removedPost.ID():
			gotIDs = append(gotIDs, post.ID())
		}
	}

	if len(gotIDs) != 2 {
		t.Fatalf("expected only two visible public test posts, got %#v", gotIDs)
	}
	if gotIDs[0] != hotPost.ID() || gotIDs[1] != warmPost.ID() {
		t.Fatalf("expected public hot order [%s %s], got %#v", hotPost.ID().String(), warmPost.ID().String(), gotIDs)
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

	for _, table := range []string{"users", "communities", "posts", "post_votes"} {
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

func insertTestPostVote(ctx context.Context, t *testing.T, pool *pgxpool.Pool, postID postdomain.PostID, userID userdomain.UserID, value int) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO post_votes (
			post_id,
			user_id,
			value,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $4)
	`, postID.String(), userID.String(), value, testNow())
	if err != nil {
		t.Fatalf("insert test post vote: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `
			DELETE FROM post_votes
			WHERE post_id = $1::uuid
				AND user_id = $2::uuid
		`, postID.String(), userID.String()); err != nil {
			t.Fatalf("cleanup test post vote post=%q user=%q: %v", postID.String(), userID.String(), err)
		}
	})
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

func mustPostTitle(t *testing.T, raw string) postdomain.PostTitle {
	t.Helper()

	title, err := postdomain.NewPostTitle(raw)
	if err != nil {
		t.Fatalf("NewPostTitle returned error: %v", err)
	}
	return title
}

func mustPostBody(t *testing.T, raw string) postdomain.PostBody {
	t.Helper()

	body, err := postdomain.NewPostBody(raw)
	if err != nil {
		t.Fatalf("NewPostBody returned error: %v", err)
	}
	return body
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

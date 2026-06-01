package voterepository

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
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresPostVoteRepositoryUpsertFindSummarizeAndDelete(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostVoteRepository(pool)
	now := testNow()

	userID := insertTestUser(ctx, t, pool)
	otherUserID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, userID, "vote-"+randomSuffix())
	postID := insertTestPost(ctx, t, pool, communityID, userID, "Vote Post")
	otherPostID := insertTestPost(ctx, t, pool, communityID, userID, "Other Vote Post")

	upvote := mustPostVote(t, postID, userID, votedomain.VoteValueUp, now)
	if err := repo.Upsert(ctx, *upvote); err != nil {
		t.Fatalf("Upsert upvote returned error: %v", err)
	}

	got, err := repo.FindByPostAndUser(ctx, postID, userID)
	if err != nil {
		t.Fatalf("FindByPostAndUser returned error: %v", err)
	}
	if got.Value() != votedomain.VoteValueUp {
		t.Fatalf("expected upvote, got %d", got.Value().Int())
	}

	myVotes, err := repo.FindByPostIDsAndUser(ctx, []postdomain.PostID{postID, otherPostID}, userID)
	if err != nil {
		t.Fatalf("FindByPostIDsAndUser returned error: %v", err)
	}
	if myVotes[postID] != votedomain.VoteValueUp {
		t.Fatalf("expected my upvote, got %d", myVotes[postID].Int())
	}
	if _, ok := myVotes[otherPostID]; ok {
		t.Fatalf("expected no vote for other post")
	}

	summaries, err := repo.SummarizeByPostIDs(ctx, []postdomain.PostID{postID, otherPostID})
	if err != nil {
		t.Fatalf("SummarizeByPostIDs returned error: %v", err)
	}
	if summaries[postID].UpvoteCount != 1 || summaries[postID].DownvoteCount != 0 || summaries[postID].Score() != 1 {
		t.Fatalf("unexpected summary after upvote: %#v", summaries[postID])
	}
	if _, ok := summaries[otherPostID]; ok {
		t.Fatalf("expected no summary for other post")
	}

	otherDownvote := mustPostVote(t, postID, otherUserID, votedomain.VoteValueDown, now.Add(time.Minute))
	if err := repo.Upsert(ctx, *otherDownvote); err != nil {
		t.Fatalf("Upsert other downvote returned error: %v", err)
	}

	changedVote := mustPostVote(t, postID, userID, votedomain.VoteValueDown, now.Add(2*time.Minute))
	if err := repo.Upsert(ctx, *changedVote); err != nil {
		t.Fatalf("Upsert changed vote returned error: %v", err)
	}

	summaries, err = repo.SummarizeByPostIDs(ctx, []postdomain.PostID{postID})
	if err != nil {
		t.Fatalf("SummarizeByPostIDs after change returned error: %v", err)
	}
	if summaries[postID].UpvoteCount != 0 || summaries[postID].DownvoteCount != 2 || summaries[postID].Score() != -2 {
		t.Fatalf("unexpected summary after vote change: %#v", summaries[postID])
	}

	if err := repo.DeleteByPostAndUser(ctx, postID, userID); err != nil {
		t.Fatalf("DeleteByPostAndUser returned error: %v", err)
	}
	if err := repo.DeleteByPostAndUser(ctx, postID, userID); err != nil {
		t.Fatalf("DeleteByPostAndUser should be idempotent, got error: %v", err)
	}
	if _, err := repo.FindByPostAndUser(ctx, postID, userID); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found after delete, got %v", err)
	}

	myVotes, err = repo.FindByPostIDsAndUser(ctx, []postdomain.PostID{postID}, userID)
	if err != nil {
		t.Fatalf("FindByPostIDsAndUser after delete returned error: %v", err)
	}
	if len(myVotes) != 0 {
		t.Fatalf("expected no my votes after delete, got %#v", myVotes)
	}
	summaries, err = repo.SummarizeByPostIDs(ctx, []postdomain.PostID{postID})
	if err != nil {
		t.Fatalf("SummarizeByPostIDs after delete returned error: %v", err)
	}
	if summaries[postID].UpvoteCount != 0 || summaries[postID].DownvoteCount != 1 || summaries[postID].Score() != -1 {
		t.Fatalf("unexpected summary after delete: %#v", summaries[postID])
	}
}

func TestPostgresPostVoteRepositoryMapsForeignKeyFailure(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostVoteRepository(pool)
	now := testNow()

	userID := insertTestUser(ctx, t, pool)
	vote := mustPostVote(t, postdomain.NewGeneratedPostID(), userID, votedomain.VoteValueUp, now)
	if err := repo.Upsert(ctx, *vote); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing related post, got %v", err)
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

	requireVoteSchema(ctx, t, pool)

	return ctx, pool
}

func requireVoteSchema(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
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
	username := "vote_repo_" + randomSuffix()

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
		if _, err := pool.Exec(context.Background(), `DELETE FROM post_votes WHERE user_id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup post votes for user %q: %v", id.String(), err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup test user %q: %v", id.String(), err)
		}
	})

	return id
}

func insertTestCommunity(ctx context.Context, t *testing.T, pool *pgxpool.Pool, createdBy userdomain.UserID, rawSlug string) communitydomain.CommunityID {
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
		VALUES ($1::uuid, $2, $3, '', 'user_created', 'active', 'public', $4::uuid, $5, $5)
	`, id.String(), rawSlug, "Vote Repo "+rawSlug, createdBy.String(), testNow())
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

func insertTestPost(ctx context.Context, t *testing.T, pool *pgxpool.Pool, communityID communitydomain.CommunityID, authorID userdomain.UserID, title string) postdomain.PostID {
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
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, 'visible', $6, $6)
	`, id.String(), communityID.String(), authorID.String(), title, "Body for "+title, testNow())
	if err != nil {
		t.Fatalf("insert test post: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM post_votes WHERE post_id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup post votes for post %q: %v", id.String(), err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM posts WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup test post %q: %v", id.String(), err)
		}
	})

	return id
}

func mustPostVote(t *testing.T, postID postdomain.PostID, userID userdomain.UserID, value votedomain.VoteValue, now time.Time) *votedomain.PostVote {
	t.Helper()

	vote, err := votedomain.NewPostVote(postID, userID, value, now)
	if err != nil {
		t.Fatalf("NewPostVote returned error: %v", err)
	}
	return vote
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

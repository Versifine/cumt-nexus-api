package effectrepository

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/effect/effectusecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresEffectRepositoryCatalogPointsAndCommentEffect(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresEffectRepository(pool)
	now := testNow()

	userID := insertTestUser(ctx, t, pool)
	commentID := insertTestComment(ctx, t, pool, userID)
	effectID := insertTestEffect(ctx, t, pool, 10)

	effects, err := repo.ListActiveEffects(ctx)
	if err != nil {
		t.Fatalf("ListActiveEffects returned error: %v", err)
	}
	if !containsEffect(effects, effectID) {
		t.Fatalf("expected effect %q in catalog, got %#v", effectID, effects)
	}

	account, err := repo.GetOrCreatePointAccount(ctx, userID, effectusecase.InitialPointBalance, now)
	if err != nil {
		t.Fatalf("GetOrCreatePointAccount returned error: %v", err)
	}
	if account.Balance != 100 || account.LifetimeEarned != 100 || account.LifetimeSpent != 0 {
		t.Fatalf("unexpected initial account: %#v", account)
	}

	effect, err := repo.FindActiveEffectByID(ctx, effectID)
	if err != nil {
		t.Fatalf("FindActiveEffectByID returned error: %v", err)
	}
	if effect.CostPoints != 10 {
		t.Fatalf("expected effect cost 10, got %d", effect.CostPoints)
	}

	result, err := repo.ApplyCommentEffect(ctx, effectusecase.ApplyCommentEffectRecordInput{
		ID:           uuid.NewString(),
		CommentID:    commentID,
		EffectID:     effectID,
		UserID:       userID,
		PointsSpent:  10,
		InitialGrant: effectusecase.InitialPointBalance,
		Now:          now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("ApplyCommentEffect returned error: %v", err)
	}
	if result.Points.Balance != 90 || result.Points.LifetimeSpent != 10 {
		t.Fatalf("unexpected points after effect: %#v", result.Points)
	}
	if result.CommentEffect.CommentID != commentID.String() || result.CommentEffect.EffectID != effectID || result.CommentEffect.PointsSpent != 10 {
		t.Fatalf("unexpected comment effect: %#v", result.CommentEffect)
	}

	var transactionCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM point_transactions
		WHERE user_id = $1::uuid
	`, userID.String()).Scan(&transactionCount); err != nil {
		t.Fatalf("count point transactions: %v", err)
	}
	if transactionCount != 2 {
		t.Fatalf("expected initial grant and comment effect transactions, got %d", transactionCount)
	}
}

func TestPostgresEffectRepositoryRejectsInsufficientPoints(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresEffectRepository(pool)
	now := testNow()

	userID := insertTestUser(ctx, t, pool)
	commentID := insertTestComment(ctx, t, pool, userID)
	effectID := insertTestEffect(ctx, t, pool, 150)

	_, err := repo.ApplyCommentEffect(ctx, effectusecase.ApplyCommentEffectRecordInput{
		ID:           uuid.NewString(),
		CommentID:    commentID,
		EffectID:     effectID,
		UserID:       userID,
		PointsSpent:  150,
		InitialGrant: effectusecase.InitialPointBalance,
		Now:          now,
	})
	if !apperr.IsCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for insufficient points, got %v", err)
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

	requireEffectSchema(ctx, t, pool)

	return ctx, pool
}

func requireEffectSchema(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	for _, table := range []string{"users", "communities", "posts", "comments", "effects", "user_points", "comment_effects", "point_transactions"} {
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
	username := "effect_repo_" + randomSuffix()
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
		if _, err := pool.Exec(context.Background(), `DELETE FROM point_transactions WHERE user_id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup point transactions for user %q: %v", id.String(), err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM comment_effects WHERE user_id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup comment effects for user %q: %v", id.String(), err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM user_points WHERE user_id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup user points for user %q: %v", id.String(), err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM comments WHERE author_id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup comments for user %q: %v", id.String(), err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM posts WHERE author_id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup posts for user %q: %v", id.String(), err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM communities WHERE created_by = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup communities for user %q: %v", id.String(), err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup test user %q: %v", id.String(), err)
		}
	})

	return id
}

func insertTestComment(ctx context.Context, t *testing.T, pool *pgxpool.Pool, authorID userdomain.UserID) commentdomain.CommentID {
	t.Helper()

	communityID := uuid.NewString()
	postID := postdomain.NewGeneratedPostID()
	commentID := commentdomain.NewGeneratedCommentID()
	now := testNow()

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
	`, communityID, "effect-"+randomSuffix(), "Effect Repo", authorID.String(), now)
	if err != nil {
		t.Fatalf("insert test community: %v", err)
	}

	_, err = pool.Exec(ctx, `
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
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'Effect Post', 'Effect body', 'visible', $4, $4)
	`, postID.String(), communityID, authorID.String(), now)
	if err != nil {
		t.Fatalf("insert test post: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO comments (
			id,
			post_id,
			author_id,
			body,
			status,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'Effect comment', 'visible', $4, $4)
	`, commentID.String(), postID.String(), authorID.String(), now)
	if err != nil {
		t.Fatalf("insert test comment: %v", err)
	}

	return commentID
}

func insertTestEffect(ctx context.Context, t *testing.T, pool *pgxpool.Pool, costPoints int) string {
	t.Helper()

	effectID := "effect_repo_" + randomSuffix()
	now := testNow()
	_, err := pool.Exec(ctx, `
		INSERT INTO effects (
			id,
			name,
			description,
			cost_points,
			asset_url,
			animation_key,
			is_active,
			created_at,
			updated_at
		)
		VALUES ($1, 'Effect Repo Test', 'Effect repository test effect.', $2, '', $1, true, $3, $3)
	`, effectID, costPoints, now)
	if err != nil {
		t.Fatalf("insert test effect: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM comment_effects WHERE effect_id = $1`, effectID); err != nil {
			t.Fatalf("cleanup comment effects for effect %q: %v", effectID, err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM effects WHERE id = $1`, effectID); err != nil {
			t.Fatalf("cleanup test effect %q: %v", effectID, err)
		}
	})

	return effectID
}

func containsEffect(effects []effectusecase.Effect, effectID string) bool {
	for _, effect := range effects {
		if effect.ID == effectID {
			return true
		}
	}
	return false
}

func randomSuffix() string {
	return strings.ReplaceAll(uuid.NewString()[:8], "-", "")
}

func testNow() time.Time {
	return time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
}

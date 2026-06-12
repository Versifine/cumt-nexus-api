package userrepository

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresUserRepositoryCreateAndFind(t *testing.T) {
	ctx, repo, pool := newTestRepository(t)
	username := newTestUsername(t)
	defer cleanupUserByUsername(ctx, t, pool, username)

	user := newTestUser(t, username)
	if err := repo.Create(ctx, *user); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	gotByID, err := repo.FindByID(ctx, user.ID())
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	assertSameUser(t, gotByID, user)

	gotByUsername, err := repo.FindByUsername(ctx, user.Username())
	if err != nil {
		t.Fatalf("FindByUsername returned error: %v", err)
	}
	assertSameUser(t, gotByUsername, user)
}

func TestPostgresUserRepositoryDuplicateUsernameReturnsConflict(t *testing.T) {
	ctx, repo, pool := newTestRepository(t)
	username := newTestUsername(t)
	defer cleanupUserByUsername(ctx, t, pool, username)

	first := newTestUser(t, username)
	if err := repo.Create(ctx, *first); err != nil {
		t.Fatalf("Create first user returned error: %v", err)
	}

	second := newTestUser(t, username)
	if err := repo.Create(ctx, *second); !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for duplicate username, got %v", err)
	}
}

func TestPostgresUserRepositoryFindNotFound(t *testing.T) {
	ctx, repo, _ := newTestRepository(t)

	missingID := userdomain.NewGeneratedUserID()
	if _, err := repo.FindByID(ctx, missingID); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing id, got %v", err)
	}

	missingUsername := newTestUsername(t)
	if _, err := repo.FindByUsername(ctx, missingUsername); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing username, got %v", err)
	}
}

func TestPostgresUserRepositoryUpdateProfile(t *testing.T) {
	ctx, repo, pool := newTestRepository(t)
	username := newTestUsername(t)
	defer cleanupUserByUsername(ctx, t, pool, username)

	user := newTestUser(t, username)
	if err := repo.Create(ctx, *user); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	displayName, err := userdomain.NewDisplayName("Alice")
	if err != nil {
		t.Fatalf("NewDisplayName returned error: %v", err)
	}
	avatarURL, err := userdomain.NewAvatarURL("https://example.com/avatar.jpg")
	if err != nil {
		t.Fatalf("NewAvatarURL returned error: %v", err)
	}
	bannerURL, err := userdomain.NewBannerURL("https://example.com/banner.jpg")
	if err != nil {
		t.Fatalf("NewBannerURL returned error: %v", err)
	}
	headline, err := userdomain.NewHeadline("Building CUMT Nexus")
	if err != nil {
		t.Fatalf("NewHeadline returned error: %v", err)
	}
	bio, err := userdomain.NewBio("Backend builder")
	if err != nil {
		t.Fatalf("NewBio returned error: %v", err)
	}

	updatedAt := user.CreatedAt().Add(time.Minute)
	if err := user.UpdateProfile(displayName, avatarURL, bannerURL, headline, bio, updatedAt); err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}
	if err := repo.UpdateProfile(ctx, *user); err != nil {
		t.Fatalf("UpdateProfile repository returned error: %v", err)
	}

	got, err := repo.FindByID(ctx, user.ID())
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	assertSameUser(t, got, user)
}

func newTestRepository(t *testing.T) (context.Context, *PostgresUserRepository, *pgxpool.Pool) {
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

	var usersTableExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
				AND table_name = 'users'
		)
	`).Scan(&usersTableExists); err != nil {
		t.Fatalf("check users table exists: %v", err)
	}
	if !usersTableExists {
		t.Fatal("users table does not exist; run go run ./cmd/migrate up before repository tests")
	}

	return ctx, NewPostgresUserRepository(pool), pool
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

func newTestUser(t *testing.T, username userdomain.Username) *userdomain.User {
	t.Helper()

	hash, err := userdomain.NewPasswordHash("hashed-password-" + username.String())
	if err != nil {
		t.Fatalf("NewPasswordHash returned error: %v", err)
	}

	user, err := userdomain.NewUser(userdomain.NewGeneratedUserID(), username, hash, time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		t.Fatalf("NewUser returned error: %v", err)
	}
	return user
}

func newTestUsername(t *testing.T) userdomain.Username {
	t.Helper()

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
	username, err := userdomain.NewUsername("repo_" + suffix)
	if err != nil {
		t.Fatalf("NewUsername returned error: %v", err)
	}
	return username
}

func cleanupUserByUsername(ctx context.Context, t *testing.T, pool *pgxpool.Pool, username userdomain.Username) {
	t.Helper()

	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE username = $1`, username.String()); err != nil {
		t.Fatalf("cleanup user %q: %v", username.String(), err)
	}
}

func assertSameUser(t *testing.T, got *userdomain.User, want *userdomain.User) {
	t.Helper()

	if got.ID() != want.ID() {
		t.Fatalf("expected user id %q, got %q", want.ID().String(), got.ID().String())
	}
	if got.Username() != want.Username() {
		t.Fatalf("expected username %q, got %q", want.Username().String(), got.Username().String())
	}
	if got.PasswordHash() != want.PasswordHash() {
		t.Fatalf("expected password hash %q, got %q", want.PasswordHash().Raw(), got.PasswordHash().Raw())
	}
	if got.Status() != want.Status() {
		t.Fatalf("expected status %q, got %q", want.Status().String(), got.Status().String())
	}
	if got.DisplayName() != want.DisplayName() {
		t.Fatalf("expected display name %q, got %q", want.DisplayName().String(), got.DisplayName().String())
	}
	if got.AvatarURL() != want.AvatarURL() {
		t.Fatalf("expected avatar url %q, got %q", want.AvatarURL().String(), got.AvatarURL().String())
	}
	if got.BannerURL() != want.BannerURL() {
		t.Fatalf("expected banner url %q, got %q", want.BannerURL().String(), got.BannerURL().String())
	}
	if got.Headline() != want.Headline() {
		t.Fatalf("expected headline %q, got %q", want.Headline().String(), got.Headline().String())
	}
	if got.Bio() != want.Bio() {
		t.Fatalf("expected bio %q, got %q", want.Bio().String(), got.Bio().String())
	}
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

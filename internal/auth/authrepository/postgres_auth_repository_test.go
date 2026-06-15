package authrepository

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresAuthRepositoryDeleteAccountReleasesRegistrationIdentifiers(t *testing.T) {
	ctx, repo, pool := newAuthRepositoryTest(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := userdomain.NewGeneratedUserID()
	newUserID := userdomain.NewGeneratedUserID()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
	username := "auth_" + suffix
	email := "auth_" + suffix + "@cumt.edu.cn"

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id IN ($1::uuid, $2::uuid)`, userID.String(), newUserID.String())
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (
			id,
			username,
			password_hash,
			email,
			email_verified_at,
			status,
			is_platform_staff,
			display_name,
			avatar_url,
			banner_url,
			headline,
			bio,
			created_at,
			updated_at
		)
		VALUES ($1::uuid,$2,$3,$4,$5,'active',true,'Alice','https://example.com/a.png','https://example.com/b.png','headline','bio',$5,$5)
	`, userID.String(), username, "hashed-password-"+username, email, now); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if err := repo.DeleteAccountByUserID(ctx, userID, now.Add(time.Minute)); err != nil {
		t.Fatalf("DeleteAccountByUserID returned error: %v", err)
	}

	var gotUsername string
	var gotEmail string
	var gotStatus string
	var gotStaff bool
	var gotDeletedAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT username, email, status, is_platform_staff, deleted_at
		FROM users
		WHERE id = $1::uuid
	`, userID.String()).Scan(&gotUsername, &gotEmail, &gotStatus, &gotStaff, &gotDeletedAt); err != nil {
		t.Fatalf("select deleted user: %v", err)
	}
	if gotUsername == username || !strings.HasPrefix(gotUsername, "deleted_") || len(gotUsername) > 32 {
		t.Fatalf("expected tombstone username within limit, got %q", gotUsername)
	}
	if gotEmail != "" {
		t.Fatalf("expected email to be released, got %q", gotEmail)
	}
	if gotStatus != "deleted" || gotStaff {
		t.Fatalf("expected deleted non-staff user, got status=%q staff=%v", gotStatus, gotStaff)
	}
	if gotDeletedAt.IsZero() {
		t.Fatal("expected deleted_at to be set")
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (
			id,
			username,
			password_hash,
			email,
			email_verified_at,
			status,
			created_at,
			updated_at
		)
		VALUES ($1::uuid,$2,$3,$4,$5,'active',$5,$5)
	`, newUserID.String(), username, "hashed-password-new-"+username, email, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("expected released username/email to be reusable: %v", err)
	}
}

func newAuthRepositoryTest(t *testing.T) (context.Context, *PostgresAuthRepository, *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	pool, err := db.Open(ctx, authRepositoryTestPostgresConfig())
	if err != nil {
		t.Skipf("skip repository integration test: open postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.Ping(ctx, pool); err != nil {
		t.Skipf("skip repository integration test: ping postgres: %v", err)
	}
	requireAuthRepositorySchema(ctx, t, pool)

	return ctx, NewPostgresAuthRepository(pool), pool
}

func requireAuthRepositorySchema(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	for _, column := range []string{
		"email",
		"email_verified_at",
		"display_name",
		"avatar_url",
		"banner_url",
		"headline",
		"bio",
		"is_platform_staff",
		"tokens_revoked_after",
		"deleted_at",
	} {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
					AND table_name = 'users'
					AND column_name = $1
			)
		`, column).Scan(&exists); err != nil {
			t.Fatalf("check users.%s exists: %v", column, err)
		}
		if !exists {
			t.Skipf("users.%s column does not exist; run go run ./cmd/migrate up before repository tests", column)
		}
	}
}

func authRepositoryTestPostgresConfig() config.PostgresConfig {
	return config.PostgresConfig{
		Host:            authRepositoryEnvString("POSTGRES_HOST", "localhost"),
		Port:            authRepositoryEnvInt("POSTGRES_PORT", 5432),
		User:            authRepositoryEnvString("POSTGRES_USER", "postgres"),
		Password:        authRepositoryEnvString("POSTGRES_PASSWORD", "postgres"),
		Database:        authRepositoryEnvString("POSTGRES_DATABASE", "cumt_nexus"),
		SSLMode:         authRepositoryEnvString("POSTGRES_SSL_MODE", "disable"),
		MaxConns:        5,
		MaxConnLifetime: time.Minute,
		MaxConnIdleTime: time.Minute,
	}
}

func authRepositoryEnvString(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func authRepositoryEnvInt(key string, fallback int) int {
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

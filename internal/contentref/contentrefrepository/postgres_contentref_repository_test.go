package contentrefrepository

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/contentref/contentrefusecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresContentRefRepositoryUpsertEmbed(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresContentRefRepository(pool)
	now := time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)
	videoID := "7123456789012345678"
	first := contentrefusecase.Embed{
		ID:            uuid.NewString(),
		Provider:      contentrefusecase.ProviderDouyin,
		ProviderRef:   videoID,
		URL:           "https://v.douyin.com/abc/",
		CanonicalURL:  "https://www.douyin.com/video/" + videoID,
		EmbedURL:      "https://open.douyin.com/player/video?autoplay=0&vid=" + videoID,
		IframeAllowed: true,
		Title:         "First title",
		Description:   "First description",
		ImageURL:      "https://www.douyin.com/cover-first.jpg",
		AuthorName:    "Alice",
		Status:        contentrefusecase.EmbedStatusReady,
	}

	stored, err := repo.UpsertEmbed(ctx, first, now)
	if err != nil {
		t.Fatalf("UpsertEmbed returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM embeds WHERE id = $1::uuid`, stored.ID)
	})
	if stored.ID != first.ID || stored.ProviderRef != videoID || stored.Title != "First title" {
		t.Fatalf("unexpected first stored embed: %#v", stored)
	}

	second := first
	second.ID = uuid.NewString()
	second.URL = "https://www.douyin.com/video/" + videoID
	second.Title = "Updated title"
	second.Description = "Updated description"
	second.ImageURL = "https://www.douyin.com/cover-updated.jpg"
	second.AuthorName = "Bob"
	second.Status = contentrefusecase.EmbedStatusUnavailable

	updated, err := repo.UpsertEmbed(ctx, second, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("second UpsertEmbed returned error: %v", err)
	}
	if updated.ID != stored.ID {
		t.Fatalf("expected upsert to keep id %q, got %q", stored.ID, updated.ID)
	}
	if updated.Title != "Updated title" ||
		updated.Description != "Updated description" ||
		updated.ImageURL != "https://www.douyin.com/cover-updated.jpg" ||
		updated.AuthorName != "Bob" ||
		updated.Status != contentrefusecase.EmbedStatusUnavailable {
		t.Fatalf("unexpected updated embed: %#v", updated)
	}
	if !updated.UpdatedAt.After(stored.UpdatedAt) {
		t.Fatalf("expected updated_at to advance, first=%s updated=%s", stored.UpdatedAt, updated.UpdatedAt)
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

	requireContentRefSchema(ctx, t, pool)

	return ctx, pool
}

func requireContentRefSchema(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
				AND table_name = 'embeds'
		)
	`).Scan(&exists); err != nil {
		t.Fatalf("check embeds table exists: %v", err)
	}
	if !exists {
		t.Skip("embeds table does not exist; run go run ./cmd/migrate up before repository tests")
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

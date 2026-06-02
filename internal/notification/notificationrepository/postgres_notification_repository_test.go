package notificationrepository

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/notification/notificationusecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresNotificationRepositoryListByRecipient(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresNotificationRepository(pool)
	now := testNow()

	recipientID := insertTestUser(ctx, t, pool)
	otherID := insertTestUser(ctx, t, pool)
	unreadID := insertTestNotification(ctx, t, pool, recipientID, nil, now)
	readAt := now.Add(time.Minute)
	readID := insertTestNotification(ctx, t, pool, recipientID, &readAt, now.Add(time.Minute))
	otherNotificationID := insertTestNotification(ctx, t, pool, otherID, nil, now)

	unread, err := repo.ListByRecipient(ctx, recipientID, notificationusecase.StatusFilterUnread, 20, 0)
	if err != nil {
		t.Fatalf("ListByRecipient unread returned error: %v", err)
	}
	if !containsNotification(unread, unreadID) || containsNotification(unread, readID) || containsNotification(unread, otherNotificationID) {
		t.Fatalf("unexpected unread results: %#v", unread)
	}

	read, err := repo.ListByRecipient(ctx, recipientID, notificationusecase.StatusFilterRead, 20, 0)
	if err != nil {
		t.Fatalf("ListByRecipient read returned error: %v", err)
	}
	if !containsNotification(read, readID) || containsNotification(read, unreadID) {
		t.Fatalf("unexpected read results: %#v", read)
	}

	all, err := repo.ListByRecipient(ctx, recipientID, notificationusecase.StatusFilterAll, 20, 0)
	if err != nil {
		t.Fatalf("ListByRecipient all returned error: %v", err)
	}
	if !containsNotification(all, unreadID) || !containsNotification(all, readID) || containsNotification(all, otherNotificationID) {
		t.Fatalf("unexpected all results: %#v", all)
	}
}

func TestPostgresNotificationRepositoryMarkReadIsIdempotent(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresNotificationRepository(pool)
	now := testNow()

	recipientID := insertTestUser(ctx, t, pool)
	notificationID := insertTestNotification(ctx, t, pool, recipientID, nil, now)
	readAt := now.Add(time.Minute)

	read, err := repo.MarkRead(ctx, notificationID, recipientID, readAt)
	if err != nil {
		t.Fatalf("MarkRead returned error: %v", err)
	}
	if read.ReadAt == nil || !read.ReadAt.Equal(readAt) {
		t.Fatalf("expected read_at %s, got %#v", readAt, read.ReadAt)
	}

	secondReadAt := readAt.Add(time.Minute)
	readAgain, err := repo.MarkRead(ctx, notificationID, recipientID, secondReadAt)
	if err != nil {
		t.Fatalf("second MarkRead returned error: %v", err)
	}
	if readAgain.ReadAt == nil || !readAgain.ReadAt.Equal(readAt) {
		t.Fatalf("expected idempotent read_at %s, got %#v", readAt, readAgain.ReadAt)
	}
}

func TestPostgresNotificationRepositoryMarkReadFiltersRecipient(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresNotificationRepository(pool)
	now := testNow()

	recipientID := insertTestUser(ctx, t, pool)
	otherID := insertTestUser(ctx, t, pool)
	notificationID := insertTestNotification(ctx, t, pool, recipientID, nil, now)

	_, err := repo.MarkRead(ctx, notificationID, otherID, now.Add(time.Minute))
	if !apperr.IsCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for other recipient, got %v", err)
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

	requireNotificationSchema(ctx, t, pool)
	return ctx, pool
}

func requireNotificationSchema(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	for _, table := range []string{"users", "notifications"} {
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
	username := "notification_repo_" + randomSuffix()
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

func insertTestNotification(ctx context.Context, t *testing.T, pool *pgxpool.Pool, recipientID userdomain.UserID, readAt *time.Time, createdAt time.Time) string {
	t.Helper()

	id := uuid.NewString()
	_, err := pool.Exec(ctx, `
		INSERT INTO notifications (
			id,
			recipient_id,
			type,
			title,
			body,
			source_type,
			source_id,
			read_at,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, 'system', 'Test notification', 'Body', 'test', $3, $4, $5, $5)
	`, id, recipientID.String(), "source-"+id, readAt, createdAt)
	if err != nil {
		t.Fatalf("insert test notification: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM notifications WHERE id = $1::uuid`, id); err != nil {
			t.Fatalf("cleanup notification %q: %v", id, err)
		}
	})

	return id
}

func containsNotification(notifications []notificationusecase.Notification, id string) bool {
	for _, notification := range notifications {
		if notification.ID == id {
			return true
		}
	}
	return false
}

func testNow() time.Time {
	return time.Date(2026, 6, 2, 7, 30, 0, 0, time.UTC)
}

func randomSuffix() string {
	return strings.ReplaceAll(uuid.NewString()[:8], "-", "")
}

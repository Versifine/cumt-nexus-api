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

	unread, err := repo.ListByRecipient(ctx, recipientID, notificationusecase.CategoryFilterAll, notificationusecase.StatusFilterUnread, 20, 0)
	if err != nil {
		t.Fatalf("ListByRecipient unread returned error: %v", err)
	}
	if !containsNotification(unread, unreadID) || containsNotification(unread, readID) || containsNotification(unread, otherNotificationID) {
		t.Fatalf("unexpected unread results: %#v", unread)
	}

	read, err := repo.ListByRecipient(ctx, recipientID, notificationusecase.CategoryFilterAll, notificationusecase.StatusFilterRead, 20, 0)
	if err != nil {
		t.Fatalf("ListByRecipient read returned error: %v", err)
	}
	if !containsNotification(read, readID) || containsNotification(read, unreadID) {
		t.Fatalf("unexpected read results: %#v", read)
	}

	all, err := repo.ListByRecipient(ctx, recipientID, notificationusecase.CategoryFilterAll, notificationusecase.StatusFilterAll, 20, 0)
	if err != nil {
		t.Fatalf("ListByRecipient all returned error: %v", err)
	}
	if !containsNotification(all, unreadID) || !containsNotification(all, readID) || containsNotification(all, otherNotificationID) {
		t.Fatalf("unexpected all results: %#v", all)
	}
}

func TestPostgresNotificationRepositoryListByCategory(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresNotificationRepository(pool)
	now := testNow()

	recipientID := insertTestUser(ctx, t, pool)
	replyID := insertTestNotificationWithType(ctx, t, pool, recipientID, "reply", nil, now)
	likeID := insertTestNotificationWithType(ctx, t, pool, recipientID, "post_like", nil, now.Add(time.Minute))
	mentionID := insertTestNotificationWithType(ctx, t, pool, recipientID, "mention", nil, now.Add(2*time.Minute))
	systemID := insertTestNotificationWithType(ctx, t, pool, recipientID, "system", nil, now.Add(3*time.Minute))

	replies, err := repo.ListByRecipient(ctx, recipientID, notificationusecase.CategoryFilterReplies, notificationusecase.StatusFilterAll, 20, 0)
	if err != nil {
		t.Fatalf("ListByRecipient replies returned error: %v", err)
	}
	if !containsNotification(replies, replyID) || containsNotification(replies, likeID) {
		t.Fatalf("unexpected replies results: %#v", replies)
	}

	likes, err := repo.ListByRecipient(ctx, recipientID, notificationusecase.CategoryFilterLikes, notificationusecase.StatusFilterAll, 20, 0)
	if err != nil {
		t.Fatalf("ListByRecipient likes returned error: %v", err)
	}
	if !containsNotification(likes, likeID) || containsNotification(likes, replyID) {
		t.Fatalf("unexpected likes results: %#v", likes)
	}

	mentions, err := repo.ListByRecipient(ctx, recipientID, notificationusecase.CategoryFilterMentions, notificationusecase.StatusFilterAll, 20, 0)
	if err != nil {
		t.Fatalf("ListByRecipient mentions returned error: %v", err)
	}
	if !containsNotification(mentions, mentionID) || containsNotification(mentions, systemID) {
		t.Fatalf("unexpected mentions results: %#v", mentions)
	}

	system, err := repo.ListByRecipient(ctx, recipientID, notificationusecase.CategoryFilterSystem, notificationusecase.StatusFilterAll, 20, 0)
	if err != nil {
		t.Fatalf("ListByRecipient system returned error: %v", err)
	}
	if !containsNotification(system, systemID) || containsNotification(system, mentionID) {
		t.Fatalf("unexpected system results: %#v", system)
	}

	interactions, err := repo.ListByRecipient(ctx, recipientID, notificationusecase.CategoryFilterInteractions, notificationusecase.StatusFilterAll, 20, 0)
	if err != nil {
		t.Fatalf("ListByRecipient interactions returned error: %v", err)
	}
	if !containsNotification(interactions, replyID) || !containsNotification(interactions, likeID) || !containsNotification(interactions, mentionID) || containsNotification(interactions, systemID) {
		t.Fatalf("unexpected interactions results: %#v", interactions)
	}
}

func TestPostgresNotificationRepositoryListIncludesActorAndContext(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresNotificationRepository(pool)
	now := testNow()

	recipientID := insertTestUser(ctx, t, pool)
	actorID := insertTestUser(ctx, t, pool)
	updateTestUserProfile(ctx, t, pool, actorID, "Alice", "https://example.com/avatar.jpg")
	communityID, communitySlug := insertTestCommunity(ctx, t, pool, actorID)
	postID := insertTestPost(ctx, t, pool, communityID, actorID, "Context post")
	commentID := insertTestComment(ctx, t, pool, postID, actorID, nil, "Context comment body")
	notificationID := insertTestNotificationWithSource(ctx, t, pool, recipientID, actorID, notificationusecase.NotificationTypeCommentReply, notificationusecase.NotificationSourceComment, commentID, now)

	notifications, err := repo.ListByRecipient(ctx, recipientID, notificationusecase.CategoryFilterInteractions, notificationusecase.StatusFilterAll, 20, 0)
	if err != nil {
		t.Fatalf("ListByRecipient returned error: %v", err)
	}
	if len(notifications) != 1 || notifications[0].ID != notificationID {
		t.Fatalf("expected context notification, got %#v", notifications)
	}
	got := notifications[0]
	if got.Actor == nil || got.Actor.Username == "" || got.Actor.DisplayName != "Alice" || got.Actor.AvatarURL == "" {
		t.Fatalf("expected actor summary, got %#v", got.Actor)
	}
	if got.LastActor == nil || got.LastActor.ID != actorID.String() {
		t.Fatalf("expected last actor summary, got %#v", got.LastActor)
	}
	if got.Context.PostID != postID || got.Context.CommentID != commentID || got.Context.PostTitle != "Context post" {
		t.Fatalf("unexpected content context: %#v", got.Context)
	}
	if got.Context.Community == nil || got.Context.Community.ID != communityID || got.Context.Community.Slug != communitySlug {
		t.Fatalf("unexpected community context: %#v", got.Context.Community)
	}
	if got.Context.Permalink == "" || !strings.Contains(got.Context.Permalink, "#comment-"+commentID) {
		t.Fatalf("expected comment permalink, got %q", got.Context.Permalink)
	}
}

func TestPostgresNotificationRepositoryUnreadSummaryAndMarkAllRead(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresNotificationRepository(pool)
	now := testNow()

	recipientID := insertTestUser(ctx, t, pool)
	readAt := now.Add(time.Minute)
	insertTestNotificationWithType(ctx, t, pool, recipientID, "reply", nil, now)
	insertTestNotificationWithType(ctx, t, pool, recipientID, "mention", nil, now)
	insertTestNotificationWithType(ctx, t, pool, recipientID, "post_like", nil, now)
	insertTestNotificationWithType(ctx, t, pool, recipientID, "system", nil, now)
	insertTestNotificationWithType(ctx, t, pool, recipientID, "system", &readAt, now)

	summary, err := repo.CountUnreadByCategory(ctx, recipientID)
	if err != nil {
		t.Fatalf("CountUnreadByCategory returned error: %v", err)
	}
	if summary.Total != 4 || summary.Replies != 1 || summary.Mentions != 1 || summary.Likes != 1 || summary.System != 1 {
		t.Fatalf("unexpected unread summary: %#v", summary)
	}

	updated, err := repo.MarkAllRead(ctx, recipientID, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("MarkAllRead returned error: %v", err)
	}
	if updated != 4 {
		t.Fatalf("expected 4 updated notifications, got %d", updated)
	}

	summary, err = repo.CountUnreadByCategory(ctx, recipientID)
	if err != nil {
		t.Fatalf("CountUnreadByCategory after mark all returned error: %v", err)
	}
	if summary.Total != 0 || summary.Replies != 0 || summary.Mentions != 0 || summary.Likes != 0 || summary.System != 0 {
		t.Fatalf("expected no unread notifications, got %#v", summary)
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

func TestPostgresNotificationRepositoryUpsertAggregatedIncrementsUnreadAggregate(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresNotificationRepository(pool)
	now := testNow()

	recipientID := insertTestUser(ctx, t, pool)
	actorID := insertTestUser(ctx, t, pool)
	nextActorID := insertTestUser(ctx, t, pool)
	notification := notificationusecase.Notification{
		ID:             uuid.NewString(),
		RecipientID:    recipientID.String(),
		Type:           notificationusecase.NotificationTypePostLike,
		Title:          "新的点赞",
		Body:           "有人赞了你的帖子",
		SourceType:     notificationusecase.NotificationSourcePost,
		SourceID:       "post-1",
		AggregateKey:   "post_like:post-1:2026060207",
		AggregateCount: 1,
		LastActorID:    actorID.String(),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repo.UpsertAggregated(ctx, notification); err != nil {
		t.Fatalf("first UpsertAggregated returned error: %v", err)
	}

	notification.ID = uuid.NewString()
	notification.LastActorID = nextActorID.String()
	notification.CreatedAt = now.Add(time.Minute)
	notification.UpdatedAt = now.Add(time.Minute)
	if err := repo.UpsertAggregated(ctx, notification); err != nil {
		t.Fatalf("second UpsertAggregated returned error: %v", err)
	}

	notifications, err := repo.ListByRecipient(ctx, recipientID, notificationusecase.CategoryFilterLikes, notificationusecase.StatusFilterUnread, 20, 0)
	if err != nil {
		t.Fatalf("ListByRecipient returned error: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected one aggregated notification, got %#v", notifications)
	}
	got := notifications[0]
	if got.AggregateCount != 2 || got.LastActorID != nextActorID.String() {
		t.Fatalf("unexpected aggregate result: %#v", got)
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

	for _, table := range []string{"users", "communities", "posts", "comments", "notifications"} {
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
	for _, column := range []string{"aggregate_key", "aggregate_count", "last_actor_id"} {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
					AND table_name = 'notifications'
					AND column_name = $1
			)
		`, column).Scan(&exists); err != nil {
			t.Fatalf("check notifications.%s exists: %v", column, err)
		}
		if !exists {
			t.Skipf("notifications.%s column does not exist; run go run ./cmd/migrate up before repository tests", column)
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

func updateTestUserProfile(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userID userdomain.UserID, displayName string, avatarURL string) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		UPDATE users
		SET display_name = $2,
			avatar_url = $3
		WHERE id = $1::uuid
	`, userID.String(), displayName, avatarURL)
	if err != nil {
		t.Fatalf("update test user profile: %v", err)
	}
}

func insertTestCommunity(ctx context.Context, t *testing.T, pool *pgxpool.Pool, creatorID userdomain.UserID) (string, string) {
	t.Helper()

	id := uuid.NewString()
	slug := "notification-" + randomSuffix()
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
		VALUES ($1::uuid, $2, 'Notification Community', '', 'user_created', 'active', 'public', $3::uuid, $4, $4)
	`, id, slug, creatorID.String(), testNow())
	if err != nil {
		t.Fatalf("insert test community: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM communities WHERE id = $1::uuid`, id); err != nil {
			t.Fatalf("cleanup community %q: %v", id, err)
		}
	})
	return id, slug
}

func insertTestPost(ctx context.Context, t *testing.T, pool *pgxpool.Pool, communityID string, authorID userdomain.UserID, title string) string {
	t.Helper()

	id := uuid.NewString()
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
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, 'Body', 'visible', $5, $5)
	`, id, communityID, authorID.String(), title, testNow())
	if err != nil {
		t.Fatalf("insert test post: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM posts WHERE id = $1::uuid`, id); err != nil {
			t.Fatalf("cleanup post %q: %v", id, err)
		}
	})
	return id
}

func insertTestComment(ctx context.Context, t *testing.T, pool *pgxpool.Pool, postID string, authorID userdomain.UserID, parentID *string, body string) string {
	t.Helper()

	id := uuid.NewString()
	_, err := pool.Exec(ctx, `
		INSERT INTO comments (
			id,
			post_id,
			author_id,
			parent_id,
			body,
			status,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, 'visible', $6, $6)
	`, id, postID, authorID.String(), parentID, body, testNow())
	if err != nil {
		t.Fatalf("insert test comment: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM comments WHERE id = $1::uuid`, id); err != nil {
			t.Fatalf("cleanup comment %q: %v", id, err)
		}
	})
	return id
}

func insertTestNotification(ctx context.Context, t *testing.T, pool *pgxpool.Pool, recipientID userdomain.UserID, readAt *time.Time, createdAt time.Time) string {
	t.Helper()

	return insertTestNotificationWithType(ctx, t, pool, recipientID, "system", readAt, createdAt)
}

func insertTestNotificationWithType(ctx context.Context, t *testing.T, pool *pgxpool.Pool, recipientID userdomain.UserID, notificationType string, readAt *time.Time, createdAt time.Time) string {
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
		VALUES ($1::uuid, $2::uuid, $3, 'Test notification', 'Body', 'test', $4, $5, $6, $6)
	`, id, recipientID.String(), notificationType, "source-"+id, readAt, createdAt)
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

func insertTestNotificationWithSource(ctx context.Context, t *testing.T, pool *pgxpool.Pool, recipientID userdomain.UserID, actorID userdomain.UserID, notificationType string, sourceType string, sourceID string, createdAt time.Time) string {
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
			aggregate_count,
			last_actor_id,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3, 'Test notification', 'Body', $4, $5, 2, $6::uuid, $7, $7)
	`, id, recipientID.String(), notificationType, sourceType, sourceID, actorID.String(), createdAt)
	if err != nil {
		t.Fatalf("insert test notification with source: %v", err)
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

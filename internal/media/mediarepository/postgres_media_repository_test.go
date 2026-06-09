package mediarepository

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresMediaRepositoryCreateAndFindByID(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresMediaRepository(pool)
	now := testNow()

	uploaderID := insertTestUser(ctx, t, pool)
	attachment := mustAttachment(t, uploaderID, now)
	if err := repo.Create(ctx, *attachment); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	cleanupAttachment(ctx, t, pool, attachment.ID())

	got, err := repo.FindByID(ctx, attachment.ID())
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got.ID() != attachment.ID() || got.UploaderID() != uploaderID {
		t.Fatalf("unexpected attachment: %#v", got)
	}
	if got.Status() != mediadomain.AttachmentStatusReady || got.MimeType() != "image/png" {
		t.Fatalf("unexpected status/mime: %s %s", got.Status(), got.MimeType())
	}

	if _, err := repo.FindByID(ctx, mediadomain.NewGeneratedAttachmentID()); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing attachment, got %v", err)
	}
}

func TestPostgresMediaRepositoryMapsConflict(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresMediaRepository(pool)
	now := testNow()

	uploaderID := insertTestUser(ctx, t, pool)
	objectKey := "images/" + randomSuffix() + ".png"
	first := mustAttachmentWithObjectKey(t, uploaderID, objectKey, now)
	second := mustAttachmentWithObjectKey(t, uploaderID, objectKey, now)
	if err := repo.Create(ctx, *first); err != nil {
		t.Fatalf("Create first returned error: %v", err)
	}
	cleanupAttachment(ctx, t, pool, first.ID())
	if err := repo.Create(ctx, *second); !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for duplicate object key, got %v", err)
	}
}

func TestPostgresMediaRepositoryBindReadyImagesToPost(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresMediaRepository(pool)
	now := testNow()

	uploaderID := insertTestUser(ctx, t, pool)
	postID := postdomain.NewGeneratedPostID()
	attachment := mustAttachment(t, uploaderID, now)
	if err := repo.Create(ctx, *attachment); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	cleanupAttachment(ctx, t, pool, attachment.ID())

	bound, err := repo.BindReadyImagesToPost(ctx, postID, uploaderID, []mediadomain.AttachmentID{attachment.ID()}, 9, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("BindReadyImagesToPost returned error: %v", err)
	}
	if len(bound) != 1 || bound[0].OwnerType() != mediadomain.OwnerTypePost || bound[0].OwnerID() != postID.String() {
		t.Fatalf("unexpected bound attachment: %#v", bound)
	}

	listed, err := repo.ListReadyImagesByPostIDs(ctx, []postdomain.PostID{postID})
	if err != nil {
		t.Fatalf("ListReadyImagesByPostIDs returned error: %v", err)
	}
	if len(listed[postID]) != 1 || listed[postID][0].ID() != attachment.ID() {
		t.Fatalf("unexpected listed attachments: %#v", listed)
	}
}

func TestPostgresMediaRepositoryBindReadyImagesToComment(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresMediaRepository(pool)
	now := testNow()

	uploaderID := insertTestUser(ctx, t, pool)
	commentID := commentdomain.NewGeneratedCommentID()
	attachment := mustAttachment(t, uploaderID, now)
	if err := repo.Create(ctx, *attachment); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	cleanupAttachment(ctx, t, pool, attachment.ID())

	bound, err := repo.BindReadyImagesToComment(ctx, commentID, uploaderID, []mediadomain.AttachmentID{attachment.ID()}, 1, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("BindReadyImagesToComment returned error: %v", err)
	}
	if len(bound) != 1 || bound[0].OwnerType() != mediadomain.OwnerTypeComment || bound[0].OwnerID() != commentID.String() {
		t.Fatalf("unexpected bound attachment: %#v", bound)
	}

	listed, err := repo.ListReadyImagesByCommentIDs(ctx, []commentdomain.CommentID{commentID})
	if err != nil {
		t.Fatalf("ListReadyImagesByCommentIDs returned error: %v", err)
	}
	if len(listed[commentID]) != 1 || listed[commentID][0].ID() != attachment.ID() {
		t.Fatalf("unexpected listed attachments: %#v", listed)
	}
}

func TestPostgresMediaRepositoryReplaceReadyImagesForPost(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresMediaRepository(pool)
	now := testNow()

	uploaderID := insertTestUser(ctx, t, pool)
	postID := postdomain.NewGeneratedPostID()
	first := mustAttachment(t, uploaderID, now)
	second := mustAttachment(t, uploaderID, now)
	for _, attachment := range []*mediadomain.Attachment{first, second} {
		if err := repo.Create(ctx, *attachment); err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		cleanupAttachment(ctx, t, pool, attachment.ID())
	}
	if _, err := repo.BindReadyImagesToPost(ctx, postID, uploaderID, []mediadomain.AttachmentID{first.ID()}, 9, now.Add(time.Minute)); err != nil {
		t.Fatalf("BindReadyImagesToPost returned error: %v", err)
	}

	replaced, err := repo.ReplaceReadyImagesForPost(ctx, postID, uploaderID, []mediadomain.AttachmentID{second.ID()}, 9, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("ReplaceReadyImagesForPost returned error: %v", err)
	}
	if len(replaced) != 1 || replaced[0].ID() != second.ID() || replaced[0].OwnerType() != mediadomain.OwnerTypePost {
		t.Fatalf("unexpected replacement result: %#v", replaced)
	}

	listed, err := repo.ListReadyImagesByPostIDs(ctx, []postdomain.PostID{postID})
	if err != nil {
		t.Fatalf("ListReadyImagesByPostIDs returned error: %v", err)
	}
	if len(listed[postID]) != 1 || listed[postID][0].ID() != second.ID() {
		t.Fatalf("expected only second attachment after replacement, got %#v", listed)
	}
	unbound, err := repo.FindByID(ctx, first.ID())
	if err != nil {
		t.Fatalf("FindByID first returned error: %v", err)
	}
	if unbound.OwnerType() != mediadomain.OwnerTypeNone || unbound.OwnerID() != "" {
		t.Fatalf("expected first attachment to be unbound, got owner=%s %q", unbound.OwnerType(), unbound.OwnerID())
	}

	cleared, err := repo.ReplaceReadyImagesForPost(ctx, postID, uploaderID, []mediadomain.AttachmentID{}, 9, now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("ReplaceReadyImagesForPost clear returned error: %v", err)
	}
	if len(cleared) != 0 {
		t.Fatalf("expected empty clear result, got %#v", cleared)
	}
	listed, err = repo.ListReadyImagesByPostIDs(ctx, []postdomain.PostID{postID})
	if err != nil {
		t.Fatalf("ListReadyImagesByPostIDs after clear returned error: %v", err)
	}
	if len(listed[postID]) != 0 {
		t.Fatalf("expected no attachments after clear, got %#v", listed)
	}
}

func TestPostgresMediaRepositoryReplaceReadyImagesForComment(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresMediaRepository(pool)
	now := testNow()

	uploaderID := insertTestUser(ctx, t, pool)
	commentID := commentdomain.NewGeneratedCommentID()
	first := mustAttachment(t, uploaderID, now)
	second := mustAttachment(t, uploaderID, now)
	for _, attachment := range []*mediadomain.Attachment{first, second} {
		if err := repo.Create(ctx, *attachment); err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		cleanupAttachment(ctx, t, pool, attachment.ID())
	}
	if _, err := repo.BindReadyImagesToComment(ctx, commentID, uploaderID, []mediadomain.AttachmentID{first.ID()}, 1, now.Add(time.Minute)); err != nil {
		t.Fatalf("BindReadyImagesToComment returned error: %v", err)
	}

	replaced, err := repo.ReplaceReadyImagesForComment(ctx, commentID, uploaderID, []mediadomain.AttachmentID{second.ID()}, 1, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("ReplaceReadyImagesForComment returned error: %v", err)
	}
	if len(replaced) != 1 || replaced[0].ID() != second.ID() || replaced[0].OwnerType() != mediadomain.OwnerTypeComment {
		t.Fatalf("unexpected replacement result: %#v", replaced)
	}

	listed, err := repo.ListReadyImagesByCommentIDs(ctx, []commentdomain.CommentID{commentID})
	if err != nil {
		t.Fatalf("ListReadyImagesByCommentIDs returned error: %v", err)
	}
	if len(listed[commentID]) != 1 || listed[commentID][0].ID() != second.ID() {
		t.Fatalf("expected only second attachment after replacement, got %#v", listed)
	}
	unbound, err := repo.FindByID(ctx, first.ID())
	if err != nil {
		t.Fatalf("FindByID first returned error: %v", err)
	}
	if unbound.OwnerType() != mediadomain.OwnerTypeNone || unbound.OwnerID() != "" {
		t.Fatalf("expected first attachment to be unbound, got owner=%s %q", unbound.OwnerType(), unbound.OwnerID())
	}
}

func TestPostgresMediaRepositoryBindRejectsOtherUploader(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresMediaRepository(pool)
	now := testNow()

	uploaderID := insertTestUser(ctx, t, pool)
	otherID := insertTestUser(ctx, t, pool)
	attachment := mustAttachment(t, uploaderID, now)
	if err := repo.Create(ctx, *attachment); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	cleanupAttachment(ctx, t, pool, attachment.ID())

	_, err := repo.BindReadyImagesToPost(ctx, postdomain.NewGeneratedPostID(), otherID, []mediadomain.AttachmentID{attachment.ID()}, 9, now.Add(time.Minute))
	if !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for other uploader, got %v", err)
	}
}

func TestPostgresMediaRepositoryCleanupCandidates(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresMediaRepository(pool)
	now := testNow()

	uploaderID := insertTestUser(ctx, t, pool)
	oldReady := mustAttachmentWithStatus(t, uploaderID, "images/"+randomSuffix()+"-old.png", mediadomain.OwnerTypeNone, "", mediadomain.AttachmentStatusReady, now.Add(-48*time.Hour), now.Add(-48*time.Hour))
	youngReady := mustAttachmentWithStatus(t, uploaderID, "images/"+randomSuffix()+"-young.png", mediadomain.OwnerTypeNone, "", mediadomain.AttachmentStatusReady, now, now)
	boundReady := mustAttachmentWithStatus(t, uploaderID, "images/"+randomSuffix()+"-bound.png", mediadomain.OwnerTypePost, postdomain.NewGeneratedPostID().String(), mediadomain.AttachmentStatusReady, now.Add(-48*time.Hour), now.Add(-48*time.Hour))
	oldFailed := mustAttachmentWithStatus(t, uploaderID, "images/"+randomSuffix()+"-failed.png", mediadomain.OwnerTypeNone, "", mediadomain.AttachmentStatusFailed, now.Add(-48*time.Hour), now.Add(-48*time.Hour))
	oldBlocked := mustAttachmentWithStatus(t, uploaderID, "images/"+randomSuffix()+"-blocked.png", mediadomain.OwnerTypeNone, "", mediadomain.AttachmentStatusBlocked, now.Add(-48*time.Hour), now.Add(-48*time.Hour))
	for _, attachment := range []*mediadomain.Attachment{oldReady, youngReady, boundReady, oldFailed, oldBlocked} {
		if err := repo.Create(ctx, *attachment); err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		cleanupAttachment(ctx, t, pool, attachment.ID())
	}

	listed, err := repo.ListCleanupCandidates(ctx, now.Add(-24*time.Hour), now.Add(-24*time.Hour), 10)
	if err != nil {
		t.Fatalf("ListCleanupCandidates returned error: %v", err)
	}
	assertAttachmentIDs(t, listed, []mediadomain.AttachmentID{oldReady.ID(), oldFailed.ID(), oldBlocked.ID()})

	taken, err := repo.TakeCleanupCandidates(ctx, now.Add(-24*time.Hour), now.Add(-24*time.Hour), 10)
	if err != nil {
		t.Fatalf("TakeCleanupCandidates returned error: %v", err)
	}
	assertAttachmentIDs(t, taken, []mediadomain.AttachmentID{oldReady.ID(), oldFailed.ID(), oldBlocked.ID()})

	for _, id := range []mediadomain.AttachmentID{oldReady.ID(), oldFailed.ID(), oldBlocked.ID()} {
		if _, err := repo.FindByID(ctx, id); !hasAppCode(err, apperr.CodeNotFound) {
			t.Fatalf("expected cleaned attachment %s to be deleted, got %v", id.String(), err)
		}
	}
	for _, id := range []mediadomain.AttachmentID{youngReady.ID(), boundReady.ID()} {
		if _, err := repo.FindByID(ctx, id); err != nil {
			t.Fatalf("expected retained attachment %s to remain, got %v", id.String(), err)
		}
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

	requireMediaSchema(ctx, t, pool)

	return ctx, pool
}

func requireMediaSchema(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	for _, table := range []string{"users", "media_attachments"} {
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
	username := "media_repo_" + randomSuffix()
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

func cleanupAttachment(ctx context.Context, t *testing.T, pool *pgxpool.Pool, id mediadomain.AttachmentID) {
	t.Helper()

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM media_attachments WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup media attachment %q: %v", id.String(), err)
		}
	})
}

func assertAttachmentIDs(t *testing.T, got []mediadomain.Attachment, want []mediadomain.AttachmentID) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d attachments, got %#v", len(want), got)
	}
	gotIDs := make(map[mediadomain.AttachmentID]bool, len(got))
	for _, attachment := range got {
		gotIDs[attachment.ID()] = true
	}
	for _, id := range want {
		if !gotIDs[id] {
			t.Fatalf("expected attachment %s in %#v", id.String(), got)
		}
	}
}

func mustAttachment(t *testing.T, uploaderID userdomain.UserID, now time.Time) *mediadomain.Attachment {
	t.Helper()

	return mustAttachmentWithObjectKey(t, uploaderID, "images/"+randomSuffix()+".png", now)
}

func mustAttachmentWithObjectKey(t *testing.T, uploaderID userdomain.UserID, objectKey string, now time.Time) *mediadomain.Attachment {
	t.Helper()

	attachment, err := mediadomain.NewReadyImageAttachment(mediadomain.NewAttachmentParams{
		ID:              mediadomain.NewGeneratedAttachmentID(),
		UploaderID:      uploaderID,
		StorageProvider: mediadomain.StorageProviderLocal,
		Bucket:          "local",
		ObjectKey:       objectKey,
		PublicURL:       "http://localhost:8080/uploads/image.png",
		SizeBytes:       100,
		MimeType:        "image/png",
		AltText:         "Campus",
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("NewReadyImageAttachment returned error: %v", err)
	}
	return attachment
}

func mustAttachmentWithStatus(
	t *testing.T,
	uploaderID userdomain.UserID,
	objectKey string,
	ownerType mediadomain.OwnerType,
	ownerID string,
	status mediadomain.AttachmentStatus,
	createdAt time.Time,
	updatedAt time.Time,
) *mediadomain.Attachment {
	t.Helper()

	attachment, err := mediadomain.RehydrateAttachment(mediadomain.NewAttachmentParams{
		ID:              mediadomain.NewGeneratedAttachmentID(),
		OwnerType:       ownerType,
		OwnerID:         ownerID,
		UploaderID:      uploaderID,
		Kind:            mediadomain.AttachmentKindImage,
		StorageProvider: mediadomain.StorageProviderLocal,
		Bucket:          "local",
		ObjectKey:       objectKey,
		PublicURL:       "http://localhost:8080/uploads/" + objectKey,
		SizeBytes:       100,
		MimeType:        "image/png",
		AltText:         "Campus",
		Status:          status,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	})
	if err != nil {
		t.Fatalf("RehydrateAttachment returned error: %v", err)
	}
	return attachment
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

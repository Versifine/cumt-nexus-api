package moderationrepository

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
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresModerationRepositoryCreatePostReportAndConflict(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	now := testNow()

	reporterID := insertTestUser(ctx, t, pool)
	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "mod-"+randomSuffix())
	post := insertTestPost(ctx, t, pool, communityID, authorID, "Reported post")
	target, err := moderationdomain.NewPostTarget(post)
	if err != nil {
		t.Fatalf("NewPostTarget returned error: %v", err)
	}
	report := mustReport(t, target, reporterID, "spam", now)

	if err := repo.CreateReport(ctx, *report); err != nil {
		t.Fatalf("CreateReport returned error: %v", err)
	}
	cleanupReport(ctx, t, pool, report.ID())

	duplicate := mustReport(t, target, reporterID, "spam again", now.Add(time.Minute))
	if err := repo.CreateReport(ctx, *duplicate); !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for duplicate pending report, got %v", err)
	}
}

func TestPostgresModerationRepositoryCreateCommentReport(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	now := testNow()

	reporterID := insertTestUser(ctx, t, pool)
	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "mod-"+randomSuffix())
	post := insertTestPost(ctx, t, pool, communityID, authorID, "Comment parent")
	comment := insertTestComment(ctx, t, pool, post, authorID)
	target, err := moderationdomain.NewCommentTarget(comment)
	if err != nil {
		t.Fatalf("NewCommentTarget returned error: %v", err)
	}
	report := mustReport(t, target, reporterID, "abuse", now)

	if err := repo.CreateReport(ctx, *report); err != nil {
		t.Fatalf("CreateReport returned error: %v", err)
	}
	cleanupReport(ctx, t, pool, report.ID())
}

func TestPostgresModerationRepositoryMapsForeignKeyFailure(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	reporterID := insertTestUser(ctx, t, pool)
	target, err := moderationdomain.NewPostTarget(postdomain.NewGeneratedPostID())
	if err != nil {
		t.Fatalf("NewPostTarget returned error: %v", err)
	}
	report := mustReport(t, target, reporterID, "missing post", testNow())

	if err := repo.CreateReport(ctx, *report); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing related record, got %v", err)
	}
}

func TestPostgresModerationRepositoryListReportsAndFindReportByID(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	now := testNow().Add(24 * time.Hour)

	reporterID := insertTestUser(ctx, t, pool)
	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "mod-"+randomSuffix())
	post := insertTestPost(ctx, t, pool, communityID, authorID, "Listed report")
	target, err := moderationdomain.NewPostTarget(post)
	if err != nil {
		t.Fatalf("NewPostTarget returned error: %v", err)
	}
	report := mustReport(t, target, reporterID, "list me", now)
	if err := repo.CreateReport(ctx, *report); err != nil {
		t.Fatalf("CreateReport returned error: %v", err)
	}
	cleanupReport(ctx, t, pool, report.ID())

	reports, err := repo.ListReports(ctx, moderationdomain.ReportStatusPending, 10, 0)
	if err != nil {
		t.Fatalf("ListReports returned error: %v", err)
	}
	if !containsReportID(reports, report.ID()) {
		t.Fatalf("expected listed reports to contain %q, got %#v", report.ID().String(), reports)
	}

	found, err := repo.FindReportByID(ctx, report.ID())
	if err != nil {
		t.Fatalf("FindReportByID returned error: %v", err)
	}
	if found.ID() != report.ID() || found.Status() != moderationdomain.ReportStatusPending {
		t.Fatalf("unexpected found report: %#v", found)
	}
}

func TestPostgresModerationRepositoryFindMissingReportReturnsNotFound(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)

	_, err := repo.FindReportByID(ctx, moderationdomain.NewGeneratedContentReportID())
	if !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing report, got %v", err)
	}
}

func TestPostgresModerationRepositoryDismissReport(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	now := testNow()

	reporterID := insertTestUser(ctx, t, pool)
	reviewerID := insertTestUser(ctx, t, pool)
	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "mod-"+randomSuffix())
	post := insertTestPost(ctx, t, pool, communityID, authorID, "Dismissed report")
	target, err := moderationdomain.NewPostTarget(post)
	if err != nil {
		t.Fatalf("NewPostTarget returned error: %v", err)
	}
	report := mustReport(t, target, reporterID, "not actually abuse", now)
	if err := repo.CreateReport(ctx, *report); err != nil {
		t.Fatalf("CreateReport returned error: %v", err)
	}
	cleanupReport(ctx, t, pool, report.ID())

	reviewedAt := now.Add(time.Minute)
	dismissed, err := repo.DismissReport(ctx, report.ID(), reviewerID, reviewedAt)
	if err != nil {
		t.Fatalf("DismissReport returned error: %v", err)
	}

	if dismissed.Status() != moderationdomain.ReportStatusDismissed {
		t.Fatalf("expected dismissed status, got %q", dismissed.Status())
	}
	gotReviewerID, ok := dismissed.ReviewedBy()
	if !ok || gotReviewerID != reviewerID {
		t.Fatalf("expected reviewer %q, got %q/%v", reviewerID.String(), gotReviewerID.String(), ok)
	}
	gotReviewedAt, ok := dismissed.ReviewedAt()
	if !ok || !gotReviewedAt.Equal(reviewedAt) {
		t.Fatalf("expected reviewed_at %s, got %s/%v", reviewedAt, gotReviewedAt, ok)
	}
	assertReportStatusInDB(t, ctx, pool, report.ID(), "dismissed")

	if _, err := repo.DismissReport(ctx, report.ID(), reviewerID, reviewedAt.Add(time.Minute)); !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for already dismissed report, got %v", err)
	}
}

func TestPostgresModerationRepositoryRemovePostWithAction(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	now := testNow()

	reporterID := insertTestUser(ctx, t, pool)
	actorID := insertTestUser(ctx, t, pool)
	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "mod-"+randomSuffix())
	post := insertTestPost(ctx, t, pool, communityID, authorID, "Removed post")
	target, err := moderationdomain.NewPostTarget(post)
	if err != nil {
		t.Fatalf("NewPostTarget returned error: %v", err)
	}
	report := mustReport(t, target, reporterID, "spam", now)
	if err := repo.CreateReport(ctx, *report); err != nil {
		t.Fatalf("CreateReport returned error: %v", err)
	}
	cleanupReport(ctx, t, pool, report.ID())
	action := mustAction(t, target, actorID, "policy violation", now.Add(time.Minute))

	if err := repo.RemovePostWithAction(ctx, *action); err != nil {
		t.Fatalf("RemovePostWithAction returned error: %v", err)
	}
	cleanupAction(ctx, t, pool, action.ID())

	assertContentStatus(t, ctx, pool, "posts", post.String(), "removed")
	assertReportStatusInDB(t, ctx, pool, report.ID(), "resolved")
	assertActionExists(t, ctx, pool, action.ID())
}

func containsReportID(reports []moderationdomain.ContentReport, id moderationdomain.ContentReportID) bool {
	for _, report := range reports {
		if report.ID() == id {
			return true
		}
	}
	return false
}

func TestPostgresModerationRepositoryRemoveCommentWithAction(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	now := testNow()

	reporterID := insertTestUser(ctx, t, pool)
	actorID := insertTestUser(ctx, t, pool)
	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "mod-"+randomSuffix())
	post := insertTestPost(ctx, t, pool, communityID, authorID, "Comment parent")
	comment := insertTestComment(ctx, t, pool, post, authorID)
	target, err := moderationdomain.NewCommentTarget(comment)
	if err != nil {
		t.Fatalf("NewCommentTarget returned error: %v", err)
	}
	report := mustReport(t, target, reporterID, "abuse", now)
	if err := repo.CreateReport(ctx, *report); err != nil {
		t.Fatalf("CreateReport returned error: %v", err)
	}
	cleanupReport(ctx, t, pool, report.ID())
	action := mustAction(t, target, actorID, "policy violation", now.Add(time.Minute))

	if err := repo.RemoveCommentWithAction(ctx, *action); err != nil {
		t.Fatalf("RemoveCommentWithAction returned error: %v", err)
	}
	cleanupAction(ctx, t, pool, action.ID())

	assertContentStatus(t, ctx, pool, "comments", comment.String(), "removed")
	assertReportStatusInDB(t, ctx, pool, report.ID(), "resolved")
	assertActionExists(t, ctx, pool, action.ID())
}

func TestPostgresModerationRepositoryRemoveReportedTargetWithAction(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	now := testNow()

	reporterID := insertTestUser(ctx, t, pool)
	actorID := insertTestUser(ctx, t, pool)
	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "mod-"+randomSuffix())
	post := insertTestPost(ctx, t, pool, communityID, authorID, "Reported target removal")
	target, err := moderationdomain.NewPostTarget(post)
	if err != nil {
		t.Fatalf("NewPostTarget returned error: %v", err)
	}
	report := mustReport(t, target, reporterID, "spam", now)
	if err := repo.CreateReport(ctx, *report); err != nil {
		t.Fatalf("CreateReport returned error: %v", err)
	}
	cleanupReport(ctx, t, pool, report.ID())
	action := mustAction(t, target, actorID, "policy violation", now.Add(time.Minute))

	if err := repo.RemoveReportedTargetWithAction(ctx, report.ID(), *action); err != nil {
		t.Fatalf("RemoveReportedTargetWithAction returned error: %v", err)
	}
	cleanupAction(ctx, t, pool, action.ID())

	assertContentStatus(t, ctx, pool, "posts", post.String(), "removed")
	assertReportStatusInDB(t, ctx, pool, report.ID(), "resolved")
	assertActionExists(t, ctx, pool, action.ID())
}

func TestPostgresModerationRepositoryRemoveReportedTargetRejectsNonPendingReport(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	now := testNow()

	reporterID := insertTestUser(ctx, t, pool)
	reviewerID := insertTestUser(ctx, t, pool)
	actorID := insertTestUser(ctx, t, pool)
	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "mod-"+randomSuffix())
	post := insertTestPost(ctx, t, pool, communityID, authorID, "Dismissed reported target")
	target, err := moderationdomain.NewPostTarget(post)
	if err != nil {
		t.Fatalf("NewPostTarget returned error: %v", err)
	}
	report := mustReport(t, target, reporterID, "not actually abuse", now)
	if err := repo.CreateReport(ctx, *report); err != nil {
		t.Fatalf("CreateReport returned error: %v", err)
	}
	cleanupReport(ctx, t, pool, report.ID())
	if _, err := repo.DismissReport(ctx, report.ID(), reviewerID, now.Add(time.Minute)); err != nil {
		t.Fatalf("DismissReport returned error: %v", err)
	}
	action := mustAction(t, target, actorID, "policy violation", now.Add(2*time.Minute))

	if err := repo.RemoveReportedTargetWithAction(ctx, report.ID(), *action); !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for non-pending report, got %v", err)
	}

	assertContentStatus(t, ctx, pool, "posts", post.String(), "visible")
	assertReportStatusInDB(t, ctx, pool, report.ID(), "dismissed")
}

func TestPostgresModerationRepositoryRemoveMissingContentReturnsNotFound(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	actorID := insertTestUser(ctx, t, pool)
	target, err := moderationdomain.NewPostTarget(postdomain.NewGeneratedPostID())
	if err != nil {
		t.Fatalf("NewPostTarget returned error: %v", err)
	}
	action := mustAction(t, target, actorID, "missing", testNow())

	if err := repo.RemovePostWithAction(ctx, *action); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing post, got %v", err)
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

	requireModerationSchema(ctx, t, pool)

	return ctx, pool
}

func requireModerationSchema(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	for _, table := range []string{"users", "communities", "posts", "comments", "content_reports", "moderation_actions"} {
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
	username := "mod_repo_" + randomSuffix()
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
	`, id.String(), rawSlug, "Moderation Repo "+rawSlug, createdBy.String(), testNow())
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
		if _, err := pool.Exec(context.Background(), `DELETE FROM posts WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup test post %q: %v", id.String(), err)
		}
	})

	return id
}

func insertTestComment(ctx context.Context, t *testing.T, pool *pgxpool.Pool, postID postdomain.PostID, authorID userdomain.UserID) commentdomain.CommentID {
	t.Helper()

	id := commentdomain.NewGeneratedCommentID()
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
		VALUES ($1::uuid, $2::uuid, $3::uuid, NULL, $4, 'visible', $5, $5)
	`, id.String(), postID.String(), authorID.String(), "Reported comment", testNow())
	if err != nil {
		t.Fatalf("insert test comment: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM comments WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup test comment %q: %v", id.String(), err)
		}
	})

	return id
}

func cleanupReport(ctx context.Context, t *testing.T, pool *pgxpool.Pool, id moderationdomain.ContentReportID) {
	t.Helper()

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM content_reports WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup content report %q: %v", id.String(), err)
		}
	})
}

func cleanupAction(ctx context.Context, t *testing.T, pool *pgxpool.Pool, id moderationdomain.ModerationActionID) {
	t.Helper()

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM moderation_actions WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup moderation action %q: %v", id.String(), err)
		}
	})
}

func mustReport(t *testing.T, target moderationdomain.Target, reporterID userdomain.UserID, reason string, now time.Time) *moderationdomain.ContentReport {
	t.Helper()

	parsedReason, err := moderationdomain.NewReason(reason)
	if err != nil {
		t.Fatalf("NewReason returned error: %v", err)
	}
	report, err := moderationdomain.NewContentReport(moderationdomain.NewGeneratedContentReportID(), target, reporterID, parsedReason, now)
	if err != nil {
		t.Fatalf("NewContentReport returned error: %v", err)
	}
	return report
}

func mustAction(t *testing.T, target moderationdomain.Target, actorID userdomain.UserID, reason string, now time.Time) *moderationdomain.ModerationAction {
	t.Helper()

	parsedReason, err := moderationdomain.NewReason(reason)
	if err != nil {
		t.Fatalf("NewReason returned error: %v", err)
	}
	action, err := moderationdomain.NewModerationAction(moderationdomain.NewGeneratedModerationActionID(), target, actorID, moderationdomain.ActionTypeRemove, parsedReason, now)
	if err != nil {
		t.Fatalf("NewModerationAction returned error: %v", err)
	}
	return action
}

func assertContentStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string, id string, want string) {
	t.Helper()

	var got string
	query := `SELECT status FROM ` + table + ` WHERE id = $1::uuid`
	if err := pool.QueryRow(ctx, query, id).Scan(&got); err != nil {
		t.Fatalf("query %s status: %v", table, err)
	}
	if got != want {
		t.Fatalf("expected %s status %q, got %q", table, want, got)
	}
}

func assertReportStatusInDB(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id moderationdomain.ContentReportID, want string) {
	t.Helper()

	var got string
	if err := pool.QueryRow(ctx, `SELECT status FROM content_reports WHERE id = $1::uuid`, id.String()).Scan(&got); err != nil {
		t.Fatalf("query report status: %v", err)
	}
	if got != want {
		t.Fatalf("expected report status %q, got %q", want, got)
	}
}

func assertActionExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id moderationdomain.ModerationActionID) {
	t.Helper()

	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM moderation_actions WHERE id = $1::uuid)`, id.String()).Scan(&exists); err != nil {
		t.Fatalf("query moderation action: %v", err)
	}
	if !exists {
		t.Fatalf("expected moderation action %q to exist", id.String())
	}
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

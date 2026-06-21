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
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationusecase"
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
	if found.Report.ID() != report.ID() || found.Report.Status() != moderationdomain.ReportStatusPending {
		t.Fatalf("unexpected found report: %#v", found)
	}
	if found.TargetPreview == nil {
		t.Fatal("expected post target preview")
	}
	if found.TargetPreview.TargetType != moderationdomain.TargetTypePost.String() || found.TargetPreview.PostID != post.String() {
		t.Fatalf("unexpected post target preview: %#v", found.TargetPreview)
	}
	if found.TargetPreview.Title != "Listed report" || found.TargetPreview.Status != "visible" {
		t.Fatalf("unexpected post preview content: %#v", found.TargetPreview)
	}
}

func TestPostgresModerationRepositoryListReportsByCommunityForManagement(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	now := testNow().Add(25 * time.Hour)

	reporterID := insertTestUser(ctx, t, pool)
	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "mod-manage-"+randomSuffix())
	otherCommunityID := insertTestCommunity(ctx, t, pool, authorID, "mod-manage-"+randomSuffix())
	post := insertTestPost(ctx, t, pool, communityID, authorID, "Managed post report")
	commentPost := insertTestPost(ctx, t, pool, communityID, authorID, "Managed comment report parent")
	comment := insertTestComment(ctx, t, pool, commentPost, authorID)
	otherPost := insertTestPost(ctx, t, pool, otherCommunityID, authorID, "Other managed report")

	postTarget, err := moderationdomain.NewPostTarget(post)
	if err != nil {
		t.Fatalf("NewPostTarget returned error: %v", err)
	}
	commentTarget, err := moderationdomain.NewCommentTarget(comment)
	if err != nil {
		t.Fatalf("NewCommentTarget returned error: %v", err)
	}
	otherTarget, err := moderationdomain.NewPostTarget(otherPost)
	if err != nil {
		t.Fatalf("NewPostTarget other returned error: %v", err)
	}
	reports := []*moderationdomain.ContentReport{
		mustReport(t, postTarget, reporterID, "post report", now),
		mustReport(t, commentTarget, reporterID, "comment report", now.Add(time.Minute)),
		mustReport(t, otherTarget, reporterID, "other report", now.Add(2*time.Minute)),
	}
	for _, report := range reports {
		if err := repo.CreateReport(ctx, *report); err != nil {
			t.Fatalf("CreateReport returned error: %v", err)
		}
		cleanupReport(ctx, t, pool, report.ID())
	}

	records, err := repo.ListReportsByCommunityForManagement(ctx, communityID, moderationdomain.ReportStatusPending, 20, 0)
	if err != nil {
		t.Fatalf("ListReportsByCommunityForManagement returned error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected two community reports, got %#v", records)
	}
	if !containsReportID(records, reports[0].ID()) || !containsReportID(records, reports[1].ID()) || containsReportID(records, reports[2].ID()) {
		t.Fatalf("unexpected community report set: %#v", records)
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

func TestPostgresModerationRepositoryFindCommentReportIncludesTargetPreview(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	now := testNow()

	reporterID := insertTestUser(ctx, t, pool)
	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "mod-"+randomSuffix())
	post := insertTestPost(ctx, t, pool, communityID, authorID, "Comment preview parent")
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

	found, err := repo.FindReportByID(ctx, report.ID())
	if err != nil {
		t.Fatalf("FindReportByID returned error: %v", err)
	}
	if found.TargetPreview == nil {
		t.Fatal("expected comment target preview")
	}
	if found.TargetPreview.TargetType != moderationdomain.TargetTypeComment.String() || found.TargetPreview.CommentID != comment.String() {
		t.Fatalf("unexpected comment target preview: %#v", found.TargetPreview)
	}
	if found.TargetPreview.PostID != post.String() || found.TargetPreview.Status != "visible" {
		t.Fatalf("unexpected comment preview context: %#v", found.TargetPreview)
	}
}

func TestPostgresModerationRepositoryGetModQueueItemAndSummary(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	now := testNow()

	reporterID := insertTestUser(ctx, t, pool)
	actorID := insertTestUser(ctx, t, pool)
	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "mod-queue-"+randomSuffix())
	post := insertTestPost(ctx, t, pool, communityID, authorID, "Queue detail")
	target, err := moderationdomain.NewPostTarget(post)
	if err != nil {
		t.Fatalf("NewPostTarget returned error: %v", err)
	}
	report := mustReport(t, target, reporterID, "queue report", now)
	if err := repo.CreateReport(ctx, *report); err != nil {
		t.Fatalf("CreateReport returned error: %v", err)
	}
	cleanupReport(ctx, t, pool, report.ID())

	action, err := repo.ApplyModerationAction(ctx, moderationusecase.ApplyModerationActionRecordInput{
		ActorID:    actorID,
		TargetType: moderationdomain.TargetTypePost,
		TargetID:   post.String(),
		Action:     moderationdomain.ActionTypeLock,
		Reason:     "needs review",
		CreatedAt:  now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("ApplyModerationAction returned error: %v", err)
	}
	cleanupActionByRawID(ctx, t, pool, action.ID)
	cleanupCommunityModLogs(ctx, t, pool, communityID)

	detail, err := repo.GetModQueueItem(ctx, moderationusecase.GetModQueueItemRecordInput{
		TargetType: moderationdomain.TargetTypePost,
		TargetID:   post.String(),
	})
	if err != nil {
		t.Fatalf("GetModQueueItem returned error: %v", err)
	}
	if detail.Item.ID != "post:"+post.String() || detail.Item.Queue != "reports" || detail.Item.ReportCount != 1 {
		t.Fatalf("unexpected queue item: %#v", detail.Item)
	}
	if detail.TargetPreview.Title != "Queue detail" || detail.TargetPreview.PostID != post.String() {
		t.Fatalf("unexpected target preview: %#v", detail.TargetPreview)
	}
	if len(detail.Reports) != 1 || detail.Reports[0].ID != report.ID().String() {
		t.Fatalf("expected report in detail, got %#v", detail.Reports)
	}
	if len(detail.RecentActions) != 1 || detail.RecentActions[0].Action != moderationdomain.ActionTypeLock.String() {
		t.Fatalf("expected recent lock action, got %#v", detail.RecentActions)
	}

	summary, err := repo.GetModQueueSummary(ctx, moderationusecase.GetModQueueSummaryRecordInput{PriorityItemLimit: 200})
	if err != nil {
		t.Fatalf("GetModQueueSummary returned error: %v", err)
	}
	if countForQueue(summary.Queues, "reports") < 1 || countForQueue(summary.Queues, "needs_review") < 1 {
		t.Fatalf("expected reports and needs_review counts, got %#v", summary.Queues)
	}
	if !containsModQueueItem(summary.PriorityItems, "post:"+post.String()) {
		t.Fatalf("expected priority items to include reported post, got %#v", summary.PriorityItems)
	}
}

func TestPostgresModerationRepositoryAutomodAndContentControls(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	now := testNow()

	actorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, actorID, "automod-"+randomSuffix())

	defaultConfig, err := repo.GetAutomodConfig(ctx, communityID)
	if err != nil {
		t.Fatalf("GetAutomodConfig default returned error: %v", err)
	}
	if defaultConfig.CommunityID != communityID.String() || defaultConfig.Version != 0 || string(defaultConfig.Rules) != "{}" {
		t.Fatalf("unexpected default automod config: %#v", defaultConfig)
	}

	config, err := repo.UpsertAutomodConfig(ctx, moderationusecase.UpsertAutomodConfigRecordInput{
		CommunityID: communityID,
		ActorID:     actorID,
		ConfigText:  "filter spamword",
		Rules:       []byte(`{"rules":[{"name":"spamword"}]}`),
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("UpsertAutomodConfig returned error: %v", err)
	}
	cleanupCommunityModLogs(ctx, t, pool, communityID)
	if config.Version != 1 || config.UpdatedBy != actorID.String() || string(config.Rules) == "{}" {
		t.Fatalf("unexpected automod config: %#v", config)
	}
	versions, err := repo.ListAutomodVersions(ctx, communityID, 10, 0)
	if err != nil {
		t.Fatalf("ListAutomodVersions returned error: %v", err)
	}
	if len(versions) != 1 || versions[0].Version != 1 || versions[0].ConfigText != "filter spamword" {
		t.Fatalf("unexpected automod versions: %#v", versions)
	}

	controls, err := repo.UpsertContentControls(ctx, moderationusecase.UpsertContentControlsRecordInput{
		CommunityID:             communityID,
		ActorID:                 actorID,
		BlockedKeywords:         []string{"spamword"},
		BlockedDomains:          []string{"bad.example"},
		MinAccountAgeDays:       7,
		PostRateLimitPerHour:    3,
		CommentRateLimitPerHour: 8,
		BlockNewAccounts:        true,
		FilterLinks:             true,
		UpdatedAt:               now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("UpsertContentControls returned error: %v", err)
	}
	if controls.CommunityID != communityID.String() || len(controls.BlockedKeywords) != 1 || controls.BlockedDomains[0] != "bad.example" || !controls.BlockNewAccounts || !controls.FilterLinks {
		t.Fatalf("unexpected content controls: %#v", controls)
	}
	gotControls, err := repo.GetContentControls(ctx, communityID)
	if err != nil {
		t.Fatalf("GetContentControls returned error: %v", err)
	}
	if gotControls.PostRateLimitPerHour != 3 || gotControls.CommentRateLimitPerHour != 8 {
		t.Fatalf("unexpected persisted content controls: %#v", gotControls)
	}
}

func TestPostgresModerationRepositoryModmail(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresModerationRepository(pool)
	now := testNow()

	actorID := insertTestUser(ctx, t, pool)
	userID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, actorID, "modmail-"+randomSuffix())

	created, err := repo.CreateModmailConversation(ctx, moderationusecase.CreateModmailConversationRecordInput{
		ID:          uuid.NewString(),
		MessageID:   uuid.NewString(),
		CommunityID: communityID,
		ActorID:     actorID,
		UserID:      userID,
		Subject:     "Need help",
		Body:        "Initial message",
		CreatedAt:   now,
	})
	if err != nil {
		t.Fatalf("CreateModmailConversation returned error: %v", err)
	}
	cleanupCommunityModLogs(ctx, t, pool, communityID)
	if created.Conversation.Subject != "Need help" || created.Conversation.Folder != "inbox" || len(created.Messages) != 1 {
		t.Fatalf("unexpected created modmail detail: %#v", created)
	}

	conversations, err := repo.ListModmailConversations(ctx, moderationusecase.ListModmailConversationsRecordInput{
		CommunityID: communityID,
		ActorID:     actorID,
		Folder:      "inbox",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("ListModmailConversations returned error: %v", err)
	}
	if len(conversations) != 1 || conversations[0].ID != created.Conversation.ID {
		t.Fatalf("unexpected modmail conversations: %#v", conversations)
	}

	withMessage, err := repo.AddModmailMessage(ctx, moderationusecase.AddModmailMessageRecordInput{
		ID:             uuid.NewString(),
		CommunityID:    communityID,
		ActorID:        actorID,
		ConversationID: created.Conversation.ID,
		Body:           "Public reply",
		IsInternal:     false,
		CreatedAt:      now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("AddModmailMessage returned error: %v", err)
	}
	if len(withMessage.Messages) != 2 {
		t.Fatalf("expected two messages, got %#v", withMessage.Messages)
	}

	withNote, err := repo.AddModmailMessage(ctx, moderationusecase.AddModmailMessageRecordInput{
		ID:             uuid.NewString(),
		CommunityID:    communityID,
		ActorID:        actorID,
		ConversationID: created.Conversation.ID,
		Body:           "Internal note",
		IsInternal:     true,
		CreatedAt:      now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("AddModmailMessage internal returned error: %v", err)
	}
	if len(withNote.Messages) != 3 || !withNote.Messages[2].IsInternal {
		t.Fatalf("expected internal note in detail, got %#v", withNote.Messages)
	}

	updated, err := repo.UpdateModmailConversation(ctx, moderationusecase.UpdateModmailConversationRecordInput{
		CommunityID:    communityID,
		ActorID:        actorID,
		ConversationID: created.Conversation.ID,
		Folder:         "in_progress",
		Status:         "in_progress",
		AssignedTo:     actorID.String(),
		MarkRead:       true,
		UpdatedAt:      now.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatalf("UpdateModmailConversation returned error: %v", err)
	}
	if updated.Folder != "in_progress" || updated.Status != "in_progress" || updated.AssignedTo != actorID.String() {
		t.Fatalf("unexpected updated modmail conversation: %#v", updated)
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

	found, err := repo.FindReportByID(ctx, report.ID())
	if err != nil {
		t.Fatalf("FindReportByID after removal returned error: %v", err)
	}
	if found.TargetPreview == nil || found.TargetPreview.Status != "removed" {
		t.Fatalf("expected removed target preview, got %#v", found.TargetPreview)
	}
}

func containsReportID(records []moderationusecase.ContentReportRecord, id moderationdomain.ContentReportID) bool {
	for _, record := range records {
		if record.Report.ID() == id {
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

	for _, table := range []string{"users", "communities", "posts", "comments", "content_reports", "moderation_actions", "community_moderation_logs", "community_automod_configs", "community_automod_config_versions", "community_content_controls", "community_modmail_conversations", "community_modmail_messages", "community_modmail_reads"} {
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

func cleanupActionByRawID(ctx context.Context, t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM moderation_actions WHERE id = $1::uuid`, id); err != nil {
			t.Fatalf("cleanup moderation action %q: %v", id, err)
		}
	})
}

func cleanupCommunityModLogs(ctx context.Context, t *testing.T, pool *pgxpool.Pool, communityID communitydomain.CommunityID) {
	t.Helper()

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM community_moderation_logs WHERE community_id = $1::uuid`, communityID.String()); err != nil {
			t.Fatalf("cleanup community moderation logs for %q: %v", communityID.String(), err)
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

func countForQueue(counts []moderationusecase.ModQueueCount, queue string) int {
	for _, count := range counts {
		if count.Queue == queue {
			return count.Count
		}
	}
	return 0
}

func containsModQueueItem(items []moderationusecase.ModQueueItem, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
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

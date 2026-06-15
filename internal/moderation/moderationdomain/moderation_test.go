package moderationdomain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestTargetTypes(t *testing.T) {
	assertTargetType(t, "post", TargetTypePost)
	assertTargetType(t, "comment", TargetTypeComment)
	if _, err := NewTargetType("user"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid target type, got %v", err)
	}
}

func TestReportStatuses(t *testing.T) {
	assertReportStatus(t, "pending", ReportStatusPending)
	assertReportStatus(t, "resolved", ReportStatusResolved)
	assertReportStatus(t, "dismissed", ReportStatusDismissed)
	if _, err := NewReportStatus("open"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid report status, got %v", err)
	}
}

func TestActionTypes(t *testing.T) {
	assertActionType(t, "remove", ActionTypeRemove)
	assertActionType(t, "approve", ActionTypeApprove)
	assertActionType(t, "spam", ActionTypeSpam)
	assertActionType(t, "ignore_reports", ActionTypeIgnoreReports)
	assertActionType(t, "lock", ActionTypeLock)
	assertActionType(t, "pin", ActionTypePin)
	assertActionType(t, "mark_nsfw", ActionTypeMarkNSFW)
	assertActionType(t, "mark_spoiler", ActionTypeMarkSpoiler)
	assertActionType(t, "set_flair", ActionTypeSetFlair)
	if _, err := NewActionType("ban"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid action type, got %v", err)
	}
}

func TestNewContentReportCreatesPendingReport(t *testing.T) {
	now := testNow()
	target, err := NewPostTarget(postdomain.NewGeneratedPostID())
	if err != nil {
		t.Fatalf("NewPostTarget returned error: %v", err)
	}
	reason := mustReason(t, "spam")
	reporterID := userdomain.NewGeneratedUserID()

	report, err := NewContentReport(NewGeneratedContentReportID(), target, reporterID, reason, now)
	if err != nil {
		t.Fatalf("NewContentReport returned error: %v", err)
	}
	if report.Status() != ReportStatusPending {
		t.Fatalf("expected pending status, got %q", report.Status().String())
	}
	if report.ReporterID() != reporterID {
		t.Fatalf("expected reporter %q, got %q", reporterID.String(), report.ReporterID().String())
	}
	if _, ok := report.ReviewedBy(); ok {
		t.Fatal("pending report should not have reviewer")
	}
}

func TestNewContentReportSupportsCommentTarget(t *testing.T) {
	target, err := NewCommentTarget(commentdomain.NewGeneratedCommentID())
	if err != nil {
		t.Fatalf("NewCommentTarget returned error: %v", err)
	}
	report, err := NewContentReport(NewGeneratedContentReportID(), target, userdomain.NewGeneratedUserID(), mustReason(t, "abuse"), testNow())
	if err != nil {
		t.Fatalf("NewContentReport returned error: %v", err)
	}
	if _, ok := report.Target().CommentID(); !ok {
		t.Fatal("expected comment target")
	}
}

func TestRehydrateContentReportValidatesReviewFields(t *testing.T) {
	now := testNow()
	target := mustPostTarget(t)
	reviewerID := userdomain.NewGeneratedUserID()
	reviewedAt := now.Add(time.Minute)

	if _, err := RehydrateContentReport(NewGeneratedContentReportID(), target, userdomain.NewGeneratedUserID(), mustReason(t, "spam"), ReportStatusPending, &reviewerID, &reviewedAt, now, now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for pending report with reviewer, got %v", err)
	}

	if _, err := RehydrateContentReport(NewGeneratedContentReportID(), target, userdomain.NewGeneratedUserID(), mustReason(t, "spam"), ReportStatusResolved, nil, nil, now, now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for resolved report without reviewer, got %v", err)
	}

	report, err := RehydrateContentReport(NewGeneratedContentReportID(), target, userdomain.NewGeneratedUserID(), mustReason(t, "spam"), ReportStatusResolved, &reviewerID, &reviewedAt, now, reviewedAt)
	if err != nil {
		t.Fatalf("RehydrateContentReport returned error: %v", err)
	}
	if got, ok := report.ReviewedBy(); !ok || got != reviewerID {
		t.Fatalf("expected reviewer %q, got %q ok=%v", reviewerID.String(), got.String(), ok)
	}
}

func TestNewModerationActionCreatesRemoveAction(t *testing.T) {
	target := mustPostTarget(t)
	actorID := userdomain.NewGeneratedUserID()
	reason := mustReason(t, "policy violation")
	now := testNow()

	action, err := NewModerationAction(NewGeneratedModerationActionID(), target, actorID, ActionTypeRemove, reason, now)
	if err != nil {
		t.Fatalf("NewModerationAction returned error: %v", err)
	}
	if action.Action() != ActionTypeRemove {
		t.Fatalf("expected remove action, got %q", action.Action().String())
	}
	if action.ActorID() != actorID {
		t.Fatalf("expected actor %q, got %q", actorID.String(), action.ActorID().String())
	}
}

func TestInvalidReasonAndTimesReturnInvalidArgument(t *testing.T) {
	if _, err := NewReason(" "); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for blank reason, got %v", err)
	}
	if _, err := NewReason(strings.Repeat("a", MaxReasonRunes+1)); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for long reason, got %v", err)
	}

	now := testNow()
	if _, err := RehydrateContentReport(NewGeneratedContentReportID(), mustPostTarget(t), userdomain.NewGeneratedUserID(), mustReason(t, "spam"), ReportStatusPending, nil, nil, now, now.Add(-time.Second)); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid report time range, got %v", err)
	}
	if _, err := NewModerationAction(NewGeneratedModerationActionID(), mustPostTarget(t), userdomain.NewGeneratedUserID(), ActionTypeRemove, mustReason(t, "spam"), time.Time{}); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for zero action time, got %v", err)
	}
}

func assertTargetType(t *testing.T, raw string, want TargetType) {
	t.Helper()
	got, err := NewTargetType(raw)
	if err != nil {
		t.Fatalf("NewTargetType returned error: %v", err)
	}
	if got != want {
		t.Fatalf("expected target type %q, got %q", want.String(), got.String())
	}
}

func assertReportStatus(t *testing.T, raw string, want ReportStatus) {
	t.Helper()
	got, err := NewReportStatus(raw)
	if err != nil {
		t.Fatalf("NewReportStatus returned error: %v", err)
	}
	if got != want {
		t.Fatalf("expected report status %q, got %q", want.String(), got.String())
	}
}

func assertActionType(t *testing.T, raw string, want ActionType) {
	t.Helper()
	got, err := NewActionType(raw)
	if err != nil {
		t.Fatalf("NewActionType returned error: %v", err)
	}
	if got != want {
		t.Fatalf("expected action type %q, got %q", want.String(), got.String())
	}
}

func mustReason(t *testing.T, raw string) Reason {
	t.Helper()
	reason, err := NewReason(raw)
	if err != nil {
		t.Fatalf("NewReason returned error: %v", err)
	}
	return reason
}

func mustPostTarget(t *testing.T) Target {
	t.Helper()
	target, err := NewPostTarget(postdomain.NewGeneratedPostID())
	if err != nil {
		t.Fatalf("NewPostTarget returned error: %v", err)
	}
	return target
}

func testNow() time.Time {
	return time.Date(2026, 6, 2, 3, 0, 0, 0, time.UTC)
}

func hasAppCode(err error, code apperr.Code) bool {
	if err == nil {
		return false
	}

	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return appErr.Code() == code
	}
	return false
}

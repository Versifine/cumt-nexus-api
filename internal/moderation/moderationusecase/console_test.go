package moderationusecase

import (
	"context"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestConsoleListReportsDefaultsPendingAndNormalizesPagination(t *testing.T) {
	now := testNow()
	actorID := userdomain.NewGeneratedUserID()
	report := mustContentReport(t, mustPostTarget(t), userdomain.NewGeneratedUserID(), "spam", now)
	reports := &fakeReportQueryRepository{
		listFunc: func(ctx context.Context, status moderationdomain.ReportStatus, limit int, offset int) ([]moderationdomain.ContentReport, error) {
			if status != moderationdomain.ReportStatusPending {
				t.Fatalf("expected pending status, got %q", status)
			}
			if limit != 20 || offset != 0 {
				t.Fatalf("expected default limit/offset, got %d/%d", limit, offset)
			}
			return []moderationdomain.ContentReport{*report}, nil
		},
	}
	uc := NewConsoleUseCase(reports, reports, &fakeStaffRepository{isStaff: true}, func() time.Time { return now })

	result, err := uc.ListReports(context.Background(), ListReportsInput{
		ActorID: actorID,
	})
	if err != nil {
		t.Fatalf("ListReports returned error: %v", err)
	}

	if !reports.listCalled {
		t.Fatal("expected ListReports repository to be called")
	}
	if result.Limit != 20 || result.Offset != 0 || len(result.Reports) != 1 {
		t.Fatalf("unexpected list result: %#v", result)
	}
}

func TestConsoleListReportsSupportsStatusAndClampsLimit(t *testing.T) {
	reports := &fakeReportQueryRepository{
		listFunc: func(ctx context.Context, status moderationdomain.ReportStatus, limit int, offset int) ([]moderationdomain.ContentReport, error) {
			if status != moderationdomain.ReportStatusResolved {
				t.Fatalf("expected resolved status, got %q", status)
			}
			if limit != 50 || offset != 3 {
				t.Fatalf("expected clamped limit and offset, got %d/%d", limit, offset)
			}
			return nil, nil
		},
	}
	uc := NewConsoleUseCase(reports, reports, &fakeStaffRepository{isStaff: true}, time.Now)

	result, err := uc.ListReports(context.Background(), ListReportsInput{
		ActorID: userdomain.NewGeneratedUserID(),
		Status:  "resolved",
		Limit:   99,
		Offset:  3,
	})
	if err != nil {
		t.Fatalf("ListReports returned error: %v", err)
	}
	if result.Limit != 50 || result.Offset != 3 {
		t.Fatalf("unexpected pagination result: %#v", result)
	}
}

func TestConsoleGetReportReturnsReport(t *testing.T) {
	now := testNow()
	report := mustContentReport(t, mustPostTarget(t), userdomain.NewGeneratedUserID(), "spam", now)
	reports := &fakeReportQueryRepository{
		findFunc: func(ctx context.Context, id moderationdomain.ContentReportID) (*moderationdomain.ContentReport, error) {
			if id != report.ID() {
				t.Fatalf("expected report id %q, got %q", report.ID().String(), id.String())
			}
			return report, nil
		},
	}
	uc := NewConsoleUseCase(reports, reports, &fakeStaffRepository{isStaff: true}, time.Now)

	result, err := uc.GetReport(context.Background(), GetReportInput{
		ActorID:  userdomain.NewGeneratedUserID(),
		ReportID: report.ID().String(),
	})
	if err != nil {
		t.Fatalf("GetReport returned error: %v", err)
	}

	if !reports.findCalled {
		t.Fatal("expected FindReportByID repository to be called")
	}
	if result.Report.ID != report.ID().String() {
		t.Fatalf("unexpected report result: %#v", result.Report)
	}
}

func TestConsoleRejectsMissingActorNonStaffAndInvalidInputs(t *testing.T) {
	uc := NewConsoleUseCase(&fakeReportQueryRepository{}, &fakeReportQueryRepository{}, &fakeStaffRepository{isStaff: true}, time.Now)

	_, err := uc.ListReports(context.Background(), ListReportsInput{})
	if !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated for missing actor, got %v", err)
	}

	uc = NewConsoleUseCase(&fakeReportQueryRepository{}, &fakeReportQueryRepository{}, &fakeStaffRepository{isStaff: false}, time.Now)
	_, err = uc.ListReports(context.Background(), ListReportsInput{
		ActorID: userdomain.NewGeneratedUserID(),
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for non staff, got %v", err)
	}

	uc = NewConsoleUseCase(&fakeReportQueryRepository{}, &fakeReportQueryRepository{}, &fakeStaffRepository{isStaff: true}, time.Now)
	_, err = uc.ListReports(context.Background(), ListReportsInput{
		ActorID: userdomain.NewGeneratedUserID(),
		Status:  "unknown",
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid status, got %v", err)
	}

	_, err = uc.ListReports(context.Background(), ListReportsInput{
		ActorID: userdomain.NewGeneratedUserID(),
		Limit:   -1,
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for negative limit, got %v", err)
	}

	_, err = uc.GetReport(context.Background(), GetReportInput{
		ActorID:  userdomain.NewGeneratedUserID(),
		ReportID: "not-a-uuid",
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid report id, got %v", err)
	}
}

func TestConsolePropagatesRepositoryError(t *testing.T) {
	query := &fakeReportQueryRepository{
		findFunc: func(ctx context.Context, id moderationdomain.ContentReportID) (*moderationdomain.ContentReport, error) {
			return nil, apperr.New(apperr.CodeNotFound, "content report not found")
		},
	}
	uc := NewConsoleUseCase(
		query,
		query,
		&fakeStaffRepository{isStaff: true},
		time.Now,
	)

	_, err := uc.GetReport(context.Background(), GetReportInput{
		ActorID:  userdomain.NewGeneratedUserID(),
		ReportID: moderationdomain.NewGeneratedContentReportID().String(),
	})
	if !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found from repository, got %v", err)
	}
}

func TestConsoleDismissReportMarksPendingReportDismissed(t *testing.T) {
	now := testNow()
	reviewedAt := now.Add(time.Minute)
	actorID := userdomain.NewGeneratedUserID()
	report := mustContentReport(t, mustPostTarget(t), userdomain.NewGeneratedUserID(), "spam", now)
	reports := &fakeReportQueryRepository{
		findFunc: func(ctx context.Context, id moderationdomain.ContentReportID) (*moderationdomain.ContentReport, error) {
			return report, nil
		},
		dismissFunc: func(ctx context.Context, id moderationdomain.ContentReportID, reviewerID userdomain.UserID, gotReviewedAt time.Time) (*moderationdomain.ContentReport, error) {
			if id != report.ID() || reviewerID != actorID || !gotReviewedAt.Equal(reviewedAt) {
				t.Fatalf("unexpected dismiss args: %q %q %s", id.String(), reviewerID.String(), gotReviewedAt)
			}
			dismissed, err := moderationdomain.RehydrateContentReport(report.ID(), report.Target(), report.ReporterID(), report.Reason(), moderationdomain.ReportStatusDismissed, &reviewerID, &gotReviewedAt, report.CreatedAt(), gotReviewedAt)
			if err != nil {
				t.Fatalf("RehydrateContentReport returned error: %v", err)
			}
			return dismissed, nil
		},
	}
	uc := NewConsoleUseCase(reports, reports, &fakeStaffRepository{isStaff: true}, func() time.Time { return reviewedAt })

	result, err := uc.DismissReport(context.Background(), DismissReportInput{
		ActorID:  actorID,
		ReportID: report.ID().String(),
	})
	if err != nil {
		t.Fatalf("DismissReport returned error: %v", err)
	}

	if !reports.findCalled || !reports.dismissCalled {
		t.Fatal("expected find and dismiss repositories to be called")
	}
	if result.Report.Status != moderationdomain.ReportStatusDismissed.String() || result.Report.ReviewedBy != actorID.String() {
		t.Fatalf("unexpected dismiss result: %#v", result.Report)
	}
	if result.Report.ReviewedAt == nil || !result.Report.ReviewedAt.Equal(reviewedAt) {
		t.Fatalf("expected reviewed_at %s, got %#v", reviewedAt, result.Report.ReviewedAt)
	}
}

func TestConsoleDismissRejectsNonPendingReport(t *testing.T) {
	now := testNow()
	reviewerID := userdomain.NewGeneratedUserID()
	report := mustContentReport(t, mustPostTarget(t), userdomain.NewGeneratedUserID(), "spam", now)
	dismissed, err := moderationdomain.RehydrateContentReport(report.ID(), report.Target(), report.ReporterID(), report.Reason(), moderationdomain.ReportStatusDismissed, &reviewerID, &now, report.CreatedAt(), now)
	if err != nil {
		t.Fatalf("RehydrateContentReport returned error: %v", err)
	}
	reports := &fakeReportQueryRepository{
		findFunc: func(ctx context.Context, id moderationdomain.ContentReportID) (*moderationdomain.ContentReport, error) {
			return dismissed, nil
		},
	}
	uc := NewConsoleUseCase(reports, reports, &fakeStaffRepository{isStaff: true}, time.Now)

	_, err = uc.DismissReport(context.Background(), DismissReportInput{
		ActorID:  userdomain.NewGeneratedUserID(),
		ReportID: report.ID().String(),
	})
	if !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for non-pending report, got %v", err)
	}
	if reports.dismissCalled {
		t.Fatal("dismiss repository should not be called for non-pending report")
	}
}

type fakeReportQueryRepository struct {
	listCalled    bool
	findCalled    bool
	dismissCalled bool
	listFunc      func(ctx context.Context, status moderationdomain.ReportStatus, limit int, offset int) ([]moderationdomain.ContentReport, error)
	findFunc      func(ctx context.Context, id moderationdomain.ContentReportID) (*moderationdomain.ContentReport, error)
	dismissFunc   func(ctx context.Context, id moderationdomain.ContentReportID, reviewerID userdomain.UserID, reviewedAt time.Time) (*moderationdomain.ContentReport, error)
}

func (f *fakeReportQueryRepository) ListReports(ctx context.Context, status moderationdomain.ReportStatus, limit int, offset int) ([]moderationdomain.ContentReport, error) {
	f.listCalled = true
	if f.listFunc != nil {
		return f.listFunc(ctx, status, limit, offset)
	}
	return nil, nil
}

func (f *fakeReportQueryRepository) FindReportByID(ctx context.Context, id moderationdomain.ContentReportID) (*moderationdomain.ContentReport, error) {
	f.findCalled = true
	if f.findFunc != nil {
		return f.findFunc(ctx, id)
	}
	return nil, apperr.New(apperr.CodeNotFound, "content report not found")
}

func (f *fakeReportQueryRepository) DismissReport(ctx context.Context, id moderationdomain.ContentReportID, reviewerID userdomain.UserID, reviewedAt time.Time) (*moderationdomain.ContentReport, error) {
	f.dismissCalled = true
	if f.dismissFunc != nil {
		return f.dismissFunc(ctx, id, reviewerID, reviewedAt)
	}
	return nil, nil
}

func mustPostTarget(t *testing.T) moderationdomain.Target {
	t.Helper()

	target, err := moderationdomain.NewPostTarget(mustPost(t, time.Now()).ID())
	if err != nil {
		t.Fatalf("NewPostTarget returned error: %v", err)
	}
	return target
}

func mustContentReport(t *testing.T, target moderationdomain.Target, reporterID userdomain.UserID, reason string, now time.Time) *moderationdomain.ContentReport {
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

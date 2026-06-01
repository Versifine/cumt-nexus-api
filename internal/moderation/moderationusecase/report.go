package moderationusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type ReportUseCase struct {
	reports  ContentReportRepository
	posts    PostRepository
	comments CommentRepository
	now      func() time.Time
}

func NewReportUseCase(reports ContentReportRepository, posts PostRepository, comments CommentRepository, now func() time.Time) *ReportUseCase {
	if now == nil {
		now = time.Now
	}

	return &ReportUseCase{
		reports:  reports,
		posts:    posts,
		comments: comments,
		now:      now,
	}
}

type ReportPostInput struct {
	PostID     string
	ReporterID userdomain.UserID
	Reason     string
}

type ReportCommentInput struct {
	CommentID  string
	ReporterID userdomain.UserID
	Reason     string
}

type ReportContentResult struct {
	Report ContentReport
}

type ContentReport struct {
	ID         string
	TargetType string
	PostID     string
	CommentID  string
	ReporterID string
	Reason     string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (uc *ReportUseCase) ReportPost(ctx context.Context, input ReportPostInput) (ReportContentResult, error) {
	if err := requireReporter(input.ReporterID); err != nil {
		return ReportContentResult{}, err
	}
	reason, err := moderationdomain.NewReason(input.Reason)
	if err != nil {
		return ReportContentResult{}, err
	}
	postID, err := postdomain.NewPostID(input.PostID)
	if err != nil {
		return ReportContentResult{}, err
	}

	post, err := uc.posts.FindVisibleByID(ctx, postID)
	if err != nil {
		return ReportContentResult{}, fmt.Errorf("find post for report: %w", err)
	}
	target, err := moderationdomain.NewPostTarget(post.ID())
	if err != nil {
		return ReportContentResult{}, err
	}

	return uc.createReport(ctx, target, input.ReporterID, reason)
}

func (uc *ReportUseCase) ReportComment(ctx context.Context, input ReportCommentInput) (ReportContentResult, error) {
	if err := requireReporter(input.ReporterID); err != nil {
		return ReportContentResult{}, err
	}
	reason, err := moderationdomain.NewReason(input.Reason)
	if err != nil {
		return ReportContentResult{}, err
	}
	commentID, err := commentdomain.NewCommentID(input.CommentID)
	if err != nil {
		return ReportContentResult{}, err
	}

	comment, err := uc.comments.FindVisibleByID(ctx, commentID)
	if err != nil {
		return ReportContentResult{}, fmt.Errorf("find comment for report: %w", err)
	}
	target, err := moderationdomain.NewCommentTarget(comment.ID())
	if err != nil {
		return ReportContentResult{}, err
	}

	return uc.createReport(ctx, target, input.ReporterID, reason)
}

func (uc *ReportUseCase) createReport(ctx context.Context, target moderationdomain.Target, reporterID userdomain.UserID, reason moderationdomain.Reason) (ReportContentResult, error) {
	now := uc.now().UTC()
	report, err := moderationdomain.NewContentReport(moderationdomain.NewGeneratedContentReportID(), target, reporterID, reason, now)
	if err != nil {
		return ReportContentResult{}, err
	}

	if err := uc.reports.CreateReport(ctx, *report); err != nil {
		return ReportContentResult{}, fmt.Errorf("create content report: %w", err)
	}

	return ReportContentResult{
		Report: toContentReportDTO(*report),
	}, nil
}

func requireReporter(reporterID userdomain.UserID) error {
	if strings.TrimSpace(reporterID.String()) == "" {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	return nil
}

func toContentReportDTO(report moderationdomain.ContentReport) ContentReport {
	target := report.Target()
	postID := ""
	if id, ok := target.PostID(); ok {
		postID = id.String()
	}
	commentID := ""
	if id, ok := target.CommentID(); ok {
		commentID = id.String()
	}

	return ContentReport{
		ID:         report.ID().String(),
		TargetType: target.Type().String(),
		PostID:     postID,
		CommentID:  commentID,
		ReporterID: report.ReporterID().String(),
		Reason:     report.Reason().String(),
		Status:     report.Status().String(),
		CreatedAt:  report.CreatedAt(),
		UpdatedAt:  report.UpdatedAt(),
	}
}

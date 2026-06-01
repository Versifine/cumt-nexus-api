package moderationusecase

import (
	"context"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type ContentReportRepository interface {
	CreateReport(ctx context.Context, report moderationdomain.ContentReport) error
}

type ContentReportQueryRepository interface {
	ListReports(ctx context.Context, status moderationdomain.ReportStatus, limit int, offset int) ([]moderationdomain.ContentReport, error)
	FindReportByID(ctx context.Context, id moderationdomain.ContentReportID) (*moderationdomain.ContentReport, error)
}

type ContentReportReviewRepository interface {
	DismissReport(ctx context.Context, id moderationdomain.ContentReportID, reviewerID userdomain.UserID, reviewedAt time.Time) (*moderationdomain.ContentReport, error)
}

type ContentRemovalRepository interface {
	RemovePostWithAction(ctx context.Context, action moderationdomain.ModerationAction) error
	RemoveCommentWithAction(ctx context.Context, action moderationdomain.ModerationAction) error
}

type ReportedTargetRemovalRepository interface {
	RemoveReportedTargetWithAction(ctx context.Context, reportID moderationdomain.ContentReportID, action moderationdomain.ModerationAction) error
}

type PlatformStaffRepository interface {
	IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error)
}

type PostRepository interface {
	FindVisibleByID(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error)
}

type CommentRepository interface {
	FindVisibleByID(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error)
}

package moderationhttp

import (
	"context"
	"net/http"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	reports ReportUseCase
}

type ReportUseCase interface {
	ReportPost(ctx context.Context, input moderationusecase.ReportPostInput) (moderationusecase.ReportContentResult, error)
	ReportComment(ctx context.Context, input moderationusecase.ReportCommentInput) (moderationusecase.ReportContentResult, error)
}

type reportContentRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type contentReportResponse struct {
	ID         string    `json:"id"`
	TargetType string    `json:"target_type"`
	PostID     string    `json:"post_id,omitempty"`
	CommentID  string    `json:"comment_id,omitempty"`
	ReporterID string    `json:"reporter_id"`
	Reason     string    `json:"reason"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type reportContentResponse struct {
	Report contentReportResponse `json:"report"`
}

func NewHandler(reports ReportUseCase) *Handler {
	return &Handler{
		reports: reports,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.POST("/posts/:id/reports", handler.ReportPost)
	group.POST("/comments/:id/reports", handler.ReportComment)
}

func (h *Handler) ReportPost(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req reportContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid content report request"))
		c.Abort()
		return
	}

	result, err := h.reports.ReportPost(c.Request.Context(), moderationusecase.ReportPostInput{
		PostID:     c.Param("id"),
		ReporterID: userID,
		Reason:     req.Reason,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, reportContentResponse{
		Report: toContentReportResponse(result.Report),
	})
}

func (h *Handler) ReportComment(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req reportContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid content report request"))
		c.Abort()
		return
	}

	result, err := h.reports.ReportComment(c.Request.Context(), moderationusecase.ReportCommentInput{
		CommentID:  c.Param("id"),
		ReporterID: userID,
		Reason:     req.Reason,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, reportContentResponse{
		Report: toContentReportResponse(result.Report),
	})
}

func toContentReportResponse(report moderationusecase.ContentReport) contentReportResponse {
	return contentReportResponse{
		ID:         report.ID,
		TargetType: report.TargetType,
		PostID:     report.PostID,
		CommentID:  report.CommentID,
		ReporterID: report.ReporterID,
		Reason:     report.Reason,
		Status:     report.Status,
		CreatedAt:  report.CreatedAt,
		UpdatedAt:  report.UpdatedAt,
	}
}

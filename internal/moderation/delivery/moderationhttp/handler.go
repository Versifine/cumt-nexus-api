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
	reports  ReportUseCase
	removals RemoveUseCase
}

type ReportUseCase interface {
	ReportPost(ctx context.Context, input moderationusecase.ReportPostInput) (moderationusecase.ReportContentResult, error)
	ReportComment(ctx context.Context, input moderationusecase.ReportCommentInput) (moderationusecase.ReportContentResult, error)
}

type RemoveUseCase interface {
	RemovePost(ctx context.Context, input moderationusecase.RemovePostInput) (moderationusecase.RemoveContentResult, error)
	RemoveComment(ctx context.Context, input moderationusecase.RemoveCommentInput) (moderationusecase.RemoveContentResult, error)
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

type moderationActionResponse struct {
	ID         string    `json:"id"`
	TargetType string    `json:"target_type"`
	PostID     string    `json:"post_id,omitempty"`
	CommentID  string    `json:"comment_id,omitempty"`
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

type removeContentResponse struct {
	Action moderationActionResponse `json:"action"`
}

func NewHandler(reports ReportUseCase, removals RemoveUseCase) *Handler {
	return &Handler{
		reports:  reports,
		removals: removals,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.POST("/posts/:id/reports", handler.ReportPost)
	group.POST("/comments/:id/reports", handler.ReportComment)
	group.POST("/posts/:id/moderation/remove", handler.RemovePost)
	group.POST("/comments/:id/moderation/remove", handler.RemoveComment)
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

func (h *Handler) RemovePost(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req reportContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid moderation remove request"))
		c.Abort()
		return
	}

	result, err := h.removals.RemovePost(c.Request.Context(), moderationusecase.RemovePostInput{
		PostID:  c.Param("id"),
		ActorID: userID,
		Reason:  req.Reason,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, removeContentResponse{
		Action: toModerationActionResponse(result.Action),
	})
}

func (h *Handler) RemoveComment(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req reportContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid moderation remove request"))
		c.Abort()
		return
	}

	result, err := h.removals.RemoveComment(c.Request.Context(), moderationusecase.RemoveCommentInput{
		CommentID: c.Param("id"),
		ActorID:   userID,
		Reason:    req.Reason,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, removeContentResponse{
		Action: toModerationActionResponse(result.Action),
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

func toModerationActionResponse(action moderationusecase.ModerationAction) moderationActionResponse {
	return moderationActionResponse{
		ID:         action.ID,
		TargetType: action.TargetType,
		PostID:     action.PostID,
		CommentID:  action.CommentID,
		ActorID:    action.ActorID,
		Action:     action.Action,
		Reason:     action.Reason,
		CreatedAt:  action.CreatedAt,
	}
}

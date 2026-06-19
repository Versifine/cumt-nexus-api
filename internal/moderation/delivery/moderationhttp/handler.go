package moderationhttp

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	reports  ReportUseCase
	removals RemoveUseCase
	console  ConsoleUseCase
	tools    ToolsUseCase
}

type ReportUseCase interface {
	ReportPost(ctx context.Context, input moderationusecase.ReportPostInput) (moderationusecase.ReportContentResult, error)
	ReportComment(ctx context.Context, input moderationusecase.ReportCommentInput) (moderationusecase.ReportContentResult, error)
}

type RemoveUseCase interface {
	RemovePost(ctx context.Context, input moderationusecase.RemovePostInput) (moderationusecase.RemoveContentResult, error)
	RemoveComment(ctx context.Context, input moderationusecase.RemoveCommentInput) (moderationusecase.RemoveContentResult, error)
	RemoveCommunityPost(ctx context.Context, input moderationusecase.RemoveCommunityPostInput) (moderationusecase.RemoveContentResult, error)
	RemoveCommunityComment(ctx context.Context, input moderationusecase.RemoveCommunityCommentInput) (moderationusecase.RemoveContentResult, error)
}

type ConsoleUseCase interface {
	ListReports(ctx context.Context, input moderationusecase.ListReportsInput) (moderationusecase.ListReportsResult, error)
	GetReport(ctx context.Context, input moderationusecase.GetReportInput) (moderationusecase.GetReportResult, error)
	DismissReport(ctx context.Context, input moderationusecase.DismissReportInput) (moderationusecase.DismissReportResult, error)
	RemoveReportedTarget(ctx context.Context, input moderationusecase.RemoveReportedTargetInput) (moderationusecase.RemoveReportedTargetResult, error)
}

type ToolsUseCase interface {
	ListModQueue(ctx context.Context, input moderationusecase.ListModQueueInput) (moderationusecase.ListModQueueResult, error)
	GetAdminModQueueItem(ctx context.Context, input moderationusecase.GetModQueueItemInput) (moderationusecase.GetModQueueItemResult, error)
	GetAdminModQueueSummary(ctx context.Context, input moderationusecase.GetModQueueSummaryInput) (moderationusecase.GetModQueueSummaryResult, error)
	ApplyBulkAction(ctx context.Context, input moderationusecase.BulkActionInput) (moderationusecase.BulkActionResult, error)
	IgnoreCommunityReport(ctx context.Context, input moderationusecase.IgnoreCommunityReportInput) (moderationusecase.IgnoreCommunityReportResult, error)
	ListCommunityModLogs(ctx context.Context, input moderationusecase.ListCommunityModLogsInput) (moderationusecase.ListCommunityModLogsResult, error)
	ListRemovalReasons(ctx context.Context, actorID userdomain.UserID, slug string) (moderationusecase.ListModerationTemplatesResult, error)
	CreateRemovalReason(ctx context.Context, input moderationusecase.ModerationTemplateInput) (moderationusecase.ModerationTemplateResult, error)
	UpdateRemovalReason(ctx context.Context, input moderationusecase.ModerationTemplateInput) (moderationusecase.ModerationTemplateResult, error)
	DeleteRemovalReason(ctx context.Context, input moderationusecase.DeleteModerationTemplateInput) error
	ListSavedResponses(ctx context.Context, actorID userdomain.UserID, slug string) (moderationusecase.ListModerationTemplatesResult, error)
	CreateSavedResponse(ctx context.Context, input moderationusecase.ModerationTemplateInput) (moderationusecase.ModerationTemplateResult, error)
	UpdateSavedResponse(ctx context.Context, input moderationusecase.ModerationTemplateInput) (moderationusecase.ModerationTemplateResult, error)
	DeleteSavedResponse(ctx context.Context, input moderationusecase.DeleteModerationTemplateInput) error
	ListUserStates(ctx context.Context, input moderationusecase.ListUserStatesInput) (moderationusecase.ListUserStatesResult, error)
	UpsertUserState(ctx context.Context, input moderationusecase.WriteUserStateInput) (moderationusecase.UserStateResult, error)
	DeleteUserState(ctx context.Context, input moderationusecase.DeleteUserStateInput) error
	GetUserProfile(ctx context.Context, input moderationusecase.GetUserProfileInput) (moderationusecase.GetUserProfileResult, error)
	ListModeratorNotes(ctx context.Context, input moderationusecase.ListModeratorNotesInput) (moderationusecase.ListModeratorNotesResult, error)
	CreateModeratorNote(ctx context.Context, input moderationusecase.CreateModeratorNoteInput) (moderationusecase.ModeratorNoteResult, error)
	DeleteModeratorNote(ctx context.Context, input moderationusecase.DeleteModeratorNoteInput) error
	GetAutomodConfig(ctx context.Context, actorID userdomain.UserID, slug string) (moderationusecase.AutomodConfigResult, error)
	UpdateAutomodConfig(ctx context.Context, input moderationusecase.AutomodConfigInput) (moderationusecase.AutomodConfigResult, error)
	ListAutomodVersions(ctx context.Context, input moderationusecase.AutomodVersionsInput) (moderationusecase.AutomodVersionsResult, error)
	DryRunAutomod(ctx context.Context, input moderationusecase.AutomodDryRunInput) (moderationusecase.AutomodDryRunResult, error)
	GetContentControls(ctx context.Context, actorID userdomain.UserID, slug string) (moderationusecase.ContentControlsResult, error)
	UpdateContentControls(ctx context.Context, input moderationusecase.ContentControlsInput) (moderationusecase.ContentControlsResult, error)
	ListModmailConversations(ctx context.Context, input moderationusecase.ListModmailConversationsInput) (moderationusecase.ListModmailConversationsResult, error)
	CreateModmailConversation(ctx context.Context, input moderationusecase.CreateModmailConversationInput) (moderationusecase.ModmailConversationResult, error)
	GetModmailConversation(ctx context.Context, input moderationusecase.GetModmailConversationInput) (moderationusecase.ModmailConversationResult, error)
	AddModmailMessage(ctx context.Context, input moderationusecase.ModmailMessageInput) (moderationusecase.ModmailConversationResult, error)
	UpdateModmailConversation(ctx context.Context, input moderationusecase.UpdateModmailConversationInput) (moderationusecase.UpdateModmailConversationResult, error)
	GetCommunityInsightsSummary(ctx context.Context, input moderationusecase.CommunityInsightsInput) (moderationusecase.CommunityInsightsSummaryResult, error)
	GetCommunityModerationInsights(ctx context.Context, input moderationusecase.CommunityInsightsInput) (moderationusecase.CommunityModerationInsightsResult, error)
	ListCommunityTrainingQueue(ctx context.Context, input moderationusecase.CommunityInsightsInput) (moderationusecase.CommunityTrainingQueueResult, error)
}

type reportContentRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type contentReportResponse struct {
	ID            string                       `json:"id"`
	TargetType    string                       `json:"target_type"`
	PostID        string                       `json:"post_id,omitempty"`
	CommentID     string                       `json:"comment_id,omitempty"`
	ReporterID    string                       `json:"reporter_id"`
	Reason        string                       `json:"reason"`
	Status        string                       `json:"status"`
	ReviewedBy    string                       `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time                   `json:"reviewed_at,omitempty"`
	TargetPreview *reportTargetPreviewResponse `json:"target_preview,omitempty"`
	CreatedAt     time.Time                    `json:"created_at"`
	UpdatedAt     time.Time                    `json:"updated_at"`
}

type reportTargetPreviewResponse struct {
	TargetType  string    `json:"target_type"`
	PostID      string    `json:"post_id,omitempty"`
	CommentID   string    `json:"comment_id,omitempty"`
	AuthorID    string    `json:"author_id"`
	Status      string    `json:"status"`
	Title       string    `json:"title,omitempty"`
	BodyExcerpt string    `json:"body_excerpt"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type reportContentResponse struct {
	Report contentReportResponse `json:"report"`
}

type listReportsResponse struct {
	Reports    []contentReportResponse `json:"reports"`
	Limit      int                     `json:"limit"`
	Offset     int                     `json:"offset"`
	NextOffset int                     `json:"next_offset"`
	HasMore    bool                    `json:"has_more"`
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

func NewHandler(reports ReportUseCase, removals RemoveUseCase, console ConsoleUseCase) *Handler {
	return &Handler{
		reports:  reports,
		removals: removals,
		console:  console,
	}
}

func (h *Handler) SetToolsUseCase(tools ToolsUseCase) {
	h.tools = tools
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.POST("/posts/:id/reports", handler.ReportPost)
	group.POST("/comments/:id/reports", handler.ReportComment)
	group.POST("/posts/:id/moderation/remove", handler.RemovePost)
	group.POST("/comments/:id/moderation/remove", handler.RemoveComment)
	group.POST("/communities/:slug/moderation/posts/:id/remove", handler.RemoveCommunityPost)
	group.POST("/communities/:slug/moderation/comments/:id/remove", handler.RemoveCommunityComment)
	group.GET("/moderation/reports", handler.ListReports)
	group.GET("/moderation/reports/:id", handler.GetReport)
	group.POST("/moderation/reports/:id/dismiss", handler.DismissReport)
	group.POST("/moderation/reports/:id/remove-target", handler.RemoveReportedTarget)
	group.GET("/admin/mod-queues", handler.ListAdminModQueue)
	group.GET("/admin/mod-queues/summary", handler.GetAdminModQueueSummary)
	group.POST("/admin/mod-queues/actions", handler.ApplyAdminModQueueAction)
	group.GET("/admin/mod-queues/:item_id", handler.GetAdminModQueueItem)
	group.GET("/communities/:slug/mod-queues", handler.ListCommunityModQueue)
	group.POST("/communities/:slug/mod-queues/actions", handler.ApplyCommunityModQueueAction)
	group.POST("/communities/:slug/moderation/posts/:id/approve", handler.ApproveCommunityPost)
	group.POST("/communities/:slug/moderation/comments/:id/approve", handler.ApproveCommunityComment)
	group.POST("/communities/:slug/moderation/posts/:id/spam", handler.SpamCommunityPost)
	group.POST("/communities/:slug/moderation/comments/:id/spam", handler.SpamCommunityComment)
	group.POST("/communities/:slug/moderation/reports/:id/ignore", handler.IgnoreCommunityReport)
	group.POST("/communities/:slug/moderation/posts/:id/lock", handler.LockCommunityPost)
	group.POST("/communities/:slug/moderation/posts/:id/pin", handler.PinCommunityPost)
	group.POST("/communities/:slug/moderation/posts/:id/mark-nsfw", handler.MarkCommunityPostNSFW)
	group.POST("/communities/:slug/moderation/posts/:id/mark-spoiler", handler.MarkCommunityPostSpoiler)
	group.POST("/communities/:slug/moderation/posts/:id/flair", handler.SetCommunityPostFlair)
	group.GET("/communities/:slug/moderation/removal-reasons", handler.ListRemovalReasons)
	group.POST("/communities/:slug/moderation/removal-reasons", handler.CreateRemovalReason)
	group.PATCH("/communities/:slug/moderation/removal-reasons/:id", handler.UpdateRemovalReason)
	group.DELETE("/communities/:slug/moderation/removal-reasons/:id", handler.DeleteRemovalReason)
	group.POST("/communities/:slug/moderation/removal-reasons/:id/apply", handler.ApplyRemovalReason)
	group.GET("/communities/:slug/moderation/saved-responses", handler.ListSavedResponses)
	group.POST("/communities/:slug/moderation/saved-responses", handler.CreateSavedResponse)
	group.PATCH("/communities/:slug/moderation/saved-responses/:id", handler.UpdateSavedResponse)
	group.DELETE("/communities/:slug/moderation/saved-responses/:id", handler.DeleteSavedResponse)
	group.GET("/communities/:slug/manage/banned-users", handler.ListBannedUsers)
	group.POST("/communities/:slug/manage/banned-users", handler.UpsertBannedUser)
	group.DELETE("/communities/:slug/manage/banned-users/:user_id", handler.DeleteBannedUser)
	group.GET("/communities/:slug/manage/muted-users", handler.ListMutedUsers)
	group.POST("/communities/:slug/manage/muted-users", handler.UpsertMutedUser)
	group.DELETE("/communities/:slug/manage/muted-users/:user_id", handler.DeleteMutedUser)
	group.GET("/communities/:slug/manage/approved-users", handler.ListApprovedUsers)
	group.POST("/communities/:slug/manage/approved-users", handler.UpsertApprovedUser)
	group.DELETE("/communities/:slug/manage/approved-users/:user_id", handler.DeleteApprovedUser)
	group.GET("/communities/:slug/moderation/users/:user_id/profile", handler.GetModerationUserProfile)
	group.GET("/communities/:slug/moderation/users/:user_id/notes", handler.ListModeratorNotes)
	group.POST("/communities/:slug/moderation/users/:user_id/notes", handler.CreateModeratorNote)
	group.DELETE("/communities/:slug/moderation/users/:user_id/notes/:note_id", handler.DeleteModeratorNote)
	group.GET("/communities/:slug/moderation/logs", handler.ListCommunityModLogs)
	group.GET("/communities/:slug/moderation/automod/config", handler.GetAutomodConfig)
	group.PATCH("/communities/:slug/moderation/automod/config", handler.UpdateAutomodConfig)
	group.GET("/communities/:slug/moderation/automod/versions", handler.ListAutomodVersions)
	group.POST("/communities/:slug/moderation/automod/dry-run", handler.DryRunAutomod)
	group.GET("/communities/:slug/moderation/content-controls", handler.GetContentControls)
	group.PATCH("/communities/:slug/moderation/content-controls", handler.UpdateContentControls)
	group.GET("/communities/:slug/modmail/conversations", handler.ListModmailConversations)
	group.POST("/communities/:slug/modmail/conversations", handler.CreateModmailConversation)
	group.GET("/communities/:slug/modmail/conversations/:conversation_id", handler.GetModmailConversation)
	group.POST("/communities/:slug/modmail/conversations/:conversation_id/messages", handler.AddModmailMessage)
	group.POST("/communities/:slug/modmail/conversations/:conversation_id/internal-notes", handler.AddModmailInternalNote)
	group.PATCH("/communities/:slug/modmail/conversations/:conversation_id", handler.UpdateModmailConversation)
	group.GET("/communities/:slug/insights/summary", handler.GetCommunityInsightsSummary)
	group.GET("/communities/:slug/insights/moderation", handler.GetCommunityModerationInsights)
	group.GET("/communities/:slug/insights/training-queue", handler.ListCommunityTrainingQueue)
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

func (h *Handler) RemoveCommunityPost(c *gin.Context) {
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

	result, err := h.removals.RemoveCommunityPost(c.Request.Context(), moderationusecase.RemoveCommunityPostInput{
		CommunitySlug: c.Param("slug"),
		PostID:        c.Param("id"),
		ActorID:       userID,
		Reason:        req.Reason,
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

func (h *Handler) RemoveCommunityComment(c *gin.Context) {
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

	result, err := h.removals.RemoveCommunityComment(c.Request.Context(), moderationusecase.RemoveCommunityCommentInput{
		CommunitySlug: c.Param("slug"),
		CommentID:     c.Param("id"),
		ActorID:       userID,
		Reason:        req.Reason,
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

func (h *Handler) ListReports(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	limit, err := parseOptionalIntQuery(c, "limit")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	offset, err := parseOptionalIntQuery(c, "offset")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	result, err := h.console.ListReports(c.Request.Context(), moderationusecase.ListReportsInput{
		ActorID: userID,
		Status:  c.Query("status"),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listReportsResponse{
		Reports:    make([]contentReportResponse, 0, len(result.Reports)),
		Limit:      result.Limit,
		Offset:     result.Offset,
		NextOffset: result.NextOffset,
		HasMore:    result.HasMore,
	}
	for _, report := range result.Reports {
		response.Reports = append(response.Reports, toContentReportResponse(report))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetReport(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	result, err := h.console.GetReport(c.Request.Context(), moderationusecase.GetReportInput{
		ActorID:  userID,
		ReportID: c.Param("id"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, reportContentResponse{
		Report: toContentReportResponse(result.Report),
	})
}

func (h *Handler) DismissReport(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	result, err := h.console.DismissReport(c.Request.Context(), moderationusecase.DismissReportInput{
		ActorID:  userID,
		ReportID: c.Param("id"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, reportContentResponse{
		Report: toContentReportResponse(result.Report),
	})
}

func (h *Handler) RemoveReportedTarget(c *gin.Context) {
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

	result, err := h.console.RemoveReportedTarget(c.Request.Context(), moderationusecase.RemoveReportedTargetInput{
		ActorID:  userID,
		ReportID: c.Param("id"),
		Reason:   req.Reason,
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

func parseOptionalIntQuery(c *gin.Context, key string) (int, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, apperr.New(apperr.CodeInvalidArgument, "invalid "+key+" query")
	}
	return value, nil
}

func toContentReportResponse(report moderationusecase.ContentReport) contentReportResponse {
	response := contentReportResponse{
		ID:         report.ID,
		TargetType: report.TargetType,
		PostID:     report.PostID,
		CommentID:  report.CommentID,
		ReporterID: report.ReporterID,
		Reason:     report.Reason,
		Status:     report.Status,
		ReviewedBy: report.ReviewedBy,
		ReviewedAt: report.ReviewedAt,
		CreatedAt:  report.CreatedAt,
		UpdatedAt:  report.UpdatedAt,
	}
	if report.TargetPreview != nil {
		preview := toReportTargetPreviewResponse(*report.TargetPreview)
		response.TargetPreview = &preview
	}
	return response
}

func toReportTargetPreviewResponse(preview moderationusecase.ReportTargetPreview) reportTargetPreviewResponse {
	return reportTargetPreviewResponse{
		TargetType:  preview.TargetType,
		PostID:      preview.PostID,
		CommentID:   preview.CommentID,
		AuthorID:    preview.AuthorID,
		Status:      preview.Status,
		Title:       preview.Title,
		BodyExcerpt: preview.BodyExcerpt,
		CreatedAt:   preview.CreatedAt,
		UpdatedAt:   preview.UpdatedAt,
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

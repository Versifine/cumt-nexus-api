package moderationhttp

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

type modQueueResponse struct {
	Items      []modQueueItemResponse `json:"items"`
	Queue      string                 `json:"queue"`
	Limit      int                    `json:"limit"`
	Offset     int                    `json:"offset"`
	NextOffset int                    `json:"next_offset"`
	HasMore    bool                   `json:"has_more"`
}

type modQueueItemResponse struct {
	ID            string    `json:"id"`
	TargetType    string    `json:"target_type"`
	TargetID      string    `json:"target_id"`
	PostID        string    `json:"post_id"`
	CommunityID   string    `json:"community_id"`
	CommunitySlug string    `json:"community_slug"`
	AuthorID      string    `json:"author_id"`
	ReportCount   int       `json:"report_count"`
	Queue         string    `json:"queue"`
	Status        string    `json:"status"`
	Preview       string    `json:"preview"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type modQueueItemDetailResponse struct {
	Item          modQueueItemResponse        `json:"item"`
	TargetPreview reportTargetPreviewResponse `json:"target_preview"`
	Reports       []modQueueReportResponse    `json:"reports"`
	RecentActions []moderationActionResponse  `json:"recent_actions"`
}

type modQueueReportResponse struct {
	ID         string    `json:"id"`
	ReporterID string    `json:"reporter_id"`
	Reason     string    `json:"reason"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type modQueueSummaryResponse struct {
	Queues        []modQueueCountResponse `json:"queues"`
	PriorityItems []modQueueItemResponse  `json:"priority_items"`
}

type modQueueCountResponse struct {
	Queue string `json:"queue"`
	Count int    `json:"count"`
}

type moderationTargetRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
}

type bulkActionRequest struct {
	Action          string                    `json:"action" binding:"required"`
	TargetType      string                    `json:"target_type"`
	TargetIDs       []string                  `json:"target_ids"`
	Targets         []moderationTargetRequest `json:"targets"`
	Reason          string                    `json:"reason"`
	RemovalReasonID string                    `json:"removal_reason_id"`
	NotifyAuthor    bool                      `json:"notify_author"`
	Confirm         bool                      `json:"confirm"`
	Value           *bool                     `json:"value"`
	FlairText       string                    `json:"flair_text"`
}

type applyRemovalReasonRequest struct {
	TargetType   string                    `json:"target_type"`
	TargetIDs    []string                  `json:"target_ids"`
	Targets      []moderationTargetRequest `json:"targets"`
	Reason       string                    `json:"reason"`
	NotifyAuthor bool                      `json:"notify_author"`
	Confirm      bool                      `json:"confirm"`
}

type singleActionRequest struct {
	Reason          string `json:"reason"`
	RemovalReasonID string `json:"removal_reason_id"`
	NotifyAuthor    bool   `json:"notify_author"`
	Confirm         bool   `json:"confirm"`
	Value           *bool  `json:"value"`
	FlairText       string `json:"flair_text"`
}

type bulkActionResponse struct {
	Results []bulkActionItemResponse `json:"results"`
}

type bulkActionItemResponse struct {
	TargetType   string                    `json:"target_type"`
	TargetID     string                    `json:"target_id"`
	OK           bool                      `json:"ok"`
	Action       *moderationActionResponse `json:"action,omitempty"`
	ErrorCode    string                    `json:"error_code,omitempty"`
	ErrorMessage string                    `json:"error_message,omitempty"`
}

type templateListResponse struct {
	Items []moderationTemplateResponse `json:"items"`
}

type templateResponse struct {
	Item moderationTemplateResponse `json:"item"`
}

type moderationTemplateResponse struct {
	ID          string    `json:"id"`
	CommunityID string    `json:"community_id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	RuleID      string    `json:"rule_id,omitempty"`
	IsActive    bool      `json:"is_active"`
	Position    int       `json:"position"`
	CreatedBy   string    `json:"created_by"`
	UpdatedBy   string    `json:"updated_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type templateRequest struct {
	Title    string `json:"title" binding:"required"`
	Body     string `json:"body"`
	RuleID   string `json:"rule_id"`
	Position int    `json:"position"`
}

type userStateRequest struct {
	UserID    string     `json:"user_id" binding:"required"`
	Reason    string     `json:"reason"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type userStateListResponse struct {
	Users      []communityUserStateResponse `json:"users"`
	Kind       string                       `json:"kind"`
	Limit      int                          `json:"limit"`
	Offset     int                          `json:"offset"`
	NextOffset int                          `json:"next_offset"`
	HasMore    bool                         `json:"has_more"`
}

type userStateResponse struct {
	User communityUserStateResponse `json:"user"`
}

type communityUserStateResponse struct {
	ID          string     `json:"id"`
	CommunityID string     `json:"community_id"`
	UserID      string     `json:"user_id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	AvatarURL   string     `json:"avatar_url"`
	Kind        string     `json:"kind"`
	Reason      string     `json:"reason"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedBy   string     `json:"created_by"`
	UpdatedBy   string     `json:"updated_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type moderationUserProfileResponse struct {
	UserID       string                  `json:"user_id"`
	Username     string                  `json:"username"`
	DisplayName  string                  `json:"display_name"`
	AvatarURL    string                  `json:"avatar_url"`
	Headline     string                  `json:"headline"`
	Status       string                  `json:"status"`
	PostCount    int                     `json:"post_count"`
	CommentCount int                     `json:"comment_count"`
	ReportCount  int                     `json:"report_count"`
	RemovedCount int                     `json:"removed_count"`
	IsBanned     bool                    `json:"is_banned"`
	IsMuted      bool                    `json:"is_muted"`
	IsApproved   bool                    `json:"is_approved"`
	RecentNotes  []moderatorNoteResponse `json:"recent_notes"`
}

type moderatorNoteResponse struct {
	ID          string    `json:"id"`
	CommunityID string    `json:"community_id"`
	UserID      string    `json:"user_id"`
	AuthorID    string    `json:"author_id"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
}

type moderatorNotesResponse struct {
	Notes      []moderatorNoteResponse `json:"notes"`
	Limit      int                     `json:"limit"`
	Offset     int                     `json:"offset"`
	NextOffset int                     `json:"next_offset"`
	HasMore    bool                    `json:"has_more"`
}

type moderatorNoteRequest struct {
	Body string `json:"body" binding:"required"`
}

type moderatorNoteMutationResponse struct {
	Note moderatorNoteResponse `json:"note"`
}

type modLogsResponse struct {
	Logs       []modLogResponse `json:"logs"`
	Limit      int              `json:"limit"`
	Offset     int              `json:"offset"`
	NextOffset int              `json:"next_offset"`
	HasMore    bool             `json:"has_more"`
}

type modLogResponse struct {
	ID          string         `json:"id"`
	CommunityID string         `json:"community_id"`
	ActorID     string         `json:"actor_id"`
	Action      string         `json:"action"`
	TargetType  string         `json:"target_type"`
	TargetID    string         `json:"target_id"`
	BatchID     string         `json:"batch_id,omitempty"`
	Before      map[string]any `json:"before"`
	After       map[string]any `json:"after"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   time.Time      `json:"created_at"`
}

type automodConfigRequest struct {
	ConfigText string          `json:"config_text"`
	Rules      json.RawMessage `json:"rules"`
}

type automodConfigResponse struct {
	Config automodConfigView `json:"config"`
}

type automodConfigView struct {
	CommunityID string          `json:"community_id"`
	ConfigText  string          `json:"config_text"`
	Rules       json.RawMessage `json:"rules"`
	Version     int             `json:"version"`
	UpdatedBy   string          `json:"updated_by,omitempty"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type automodVersionsResponse struct {
	Versions   []automodVersionView `json:"versions"`
	Limit      int                  `json:"limit"`
	Offset     int                  `json:"offset"`
	NextOffset int                  `json:"next_offset"`
	HasMore    bool                 `json:"has_more"`
}

type automodVersionView struct {
	ID          string          `json:"id"`
	CommunityID string          `json:"community_id"`
	Version     int             `json:"version"`
	ConfigText  string          `json:"config_text"`
	Rules       json.RawMessage `json:"rules"`
	UpdatedBy   string          `json:"updated_by"`
	CreatedAt   time.Time       `json:"created_at"`
}

type automodDryRunRequest struct {
	TargetType string   `json:"target_type" binding:"required"`
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	AuthorID   string   `json:"author_id"`
	Links      []string `json:"links"`
}

type automodDryRunResponse struct {
	Matches         []automodDryRunMatchView `json:"matches"`
	SuggestedAction string                   `json:"suggested_action"`
	Reasons         []string                 `json:"reasons"`
}

type automodDryRunMatchView struct {
	Rule   string `json:"rule"`
	Action string `json:"action"`
	Reason string `json:"reason"`
}

type contentControlsRequest struct {
	BlockedKeywords         []string `json:"blocked_keywords"`
	BlockedDomains          []string `json:"blocked_domains"`
	MinAccountAgeDays       int      `json:"min_account_age_days"`
	PostRateLimitPerHour    int      `json:"post_rate_limit_per_hour"`
	CommentRateLimitPerHour int      `json:"comment_rate_limit_per_hour"`
	BlockNewAccounts        bool     `json:"block_new_accounts"`
	FilterLinks             bool     `json:"filter_links"`
}

type contentControlsResponse struct {
	Controls contentControlsView `json:"controls"`
}

type contentControlsView struct {
	CommunityID             string    `json:"community_id"`
	BlockedKeywords         []string  `json:"blocked_keywords"`
	BlockedDomains          []string  `json:"blocked_domains"`
	MinAccountAgeDays       int       `json:"min_account_age_days"`
	PostRateLimitPerHour    int       `json:"post_rate_limit_per_hour"`
	CommentRateLimitPerHour int       `json:"comment_rate_limit_per_hour"`
	BlockNewAccounts        bool      `json:"block_new_accounts"`
	FilterLinks             bool      `json:"filter_links"`
	UpdatedBy               string    `json:"updated_by,omitempty"`
	UpdatedAt               time.Time `json:"updated_at"`
}

func (h *Handler) ListAdminModQueue(c *gin.Context) {
	h.listModQueue(c, "")
}

func (h *Handler) ListCommunityModQueue(c *gin.Context) {
	h.listModQueue(c, c.Param("slug"))
}

func (h *Handler) GetAutomodConfig(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	result, err := h.tools.GetAutomodConfig(c.Request.Context(), userID, c.Param("slug"))
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, automodConfigResponse{Config: toAutomodConfigView(result.Config)})
}

func (h *Handler) UpdateAutomodConfig(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req automodConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid automod config request"))
		c.Abort()
		return
	}
	result, err := h.tools.UpdateAutomodConfig(c.Request.Context(), moderationusecase.AutomodConfigInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		ConfigText:    req.ConfigText,
		Rules:         req.Rules,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, automodConfigResponse{Config: toAutomodConfigView(result.Config)})
}

func (h *Handler) ListAutomodVersions(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
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
	result, err := h.tools.ListAutomodVersions(c.Request.Context(), moderationusecase.AutomodVersionsInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := automodVersionsResponse{
		Versions:   make([]automodVersionView, 0, len(result.Versions)),
		Limit:      result.Limit,
		Offset:     result.Offset,
		NextOffset: result.NextOffset,
		HasMore:    result.HasMore,
	}
	for _, version := range result.Versions {
		response.Versions = append(response.Versions, toAutomodVersionView(version))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) DryRunAutomod(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req automodDryRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid automod dry-run request"))
		c.Abort()
		return
	}
	result, err := h.tools.DryRunAutomod(c.Request.Context(), moderationusecase.AutomodDryRunInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		TargetType:    req.TargetType,
		Title:         req.Title,
		Body:          req.Body,
		AuthorID:      req.AuthorID,
		Links:         req.Links,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := automodDryRunResponse{
		Matches:         make([]automodDryRunMatchView, 0, len(result.Matches)),
		SuggestedAction: result.SuggestedAction,
		Reasons:         result.Reasons,
	}
	for _, match := range result.Matches {
		response.Matches = append(response.Matches, automodDryRunMatchView(match))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetContentControls(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	result, err := h.tools.GetContentControls(c.Request.Context(), userID, c.Param("slug"))
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, contentControlsResponse{Controls: toContentControlsView(result.Controls)})
}

func (h *Handler) UpdateContentControls(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req contentControlsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid content controls request"))
		c.Abort()
		return
	}
	result, err := h.tools.UpdateContentControls(c.Request.Context(), moderationusecase.ContentControlsInput{
		ActorID:                 userID,
		CommunitySlug:           c.Param("slug"),
		BlockedKeywords:         req.BlockedKeywords,
		BlockedDomains:          req.BlockedDomains,
		MinAccountAgeDays:       req.MinAccountAgeDays,
		PostRateLimitPerHour:    req.PostRateLimitPerHour,
		CommentRateLimitPerHour: req.CommentRateLimitPerHour,
		BlockNewAccounts:        req.BlockNewAccounts,
		FilterLinks:             req.FilterLinks,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, contentControlsResponse{Controls: toContentControlsView(result.Controls)})
}

func (h *Handler) GetAdminModQueueItem(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	if h.tools == nil {
		_ = c.Error(apperr.New(apperr.CodeInternal, "moderation tools are not configured"))
		c.Abort()
		return
	}
	result, err := h.tools.GetAdminModQueueItem(c.Request.Context(), moderationusecase.GetModQueueItemInput{
		ActorID: userID,
		ItemID:  c.Param("item_id"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, toModQueueItemDetailResponse(result.Detail))
}

func (h *Handler) GetAdminModQueueSummary(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	if h.tools == nil {
		_ = c.Error(apperr.New(apperr.CodeInternal, "moderation tools are not configured"))
		c.Abort()
		return
	}
	result, err := h.tools.GetAdminModQueueSummary(c.Request.Context(), moderationusecase.GetModQueueSummaryInput{
		ActorID: userID,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, toModQueueSummaryResponse(result.Summary))
}

func (h *Handler) ApplyAdminModQueueAction(c *gin.Context) {
	h.applyBulkAction(c, "")
}

func (h *Handler) ApplyCommunityModQueueAction(c *gin.Context) {
	h.applyBulkAction(c, c.Param("slug"))
}

func (h *Handler) ApproveCommunityPost(c *gin.Context) {
	h.applySingleAction(c, c.Param("slug"), "post", c.Param("id"), "approve")
}

func (h *Handler) ApproveCommunityComment(c *gin.Context) {
	h.applySingleAction(c, c.Param("slug"), "comment", c.Param("id"), "approve")
}

func (h *Handler) SpamCommunityPost(c *gin.Context) {
	h.applySingleAction(c, c.Param("slug"), "post", c.Param("id"), "spam")
}

func (h *Handler) SpamCommunityComment(c *gin.Context) {
	h.applySingleAction(c, c.Param("slug"), "comment", c.Param("id"), "spam")
}

func (h *Handler) IgnoreCommunityReport(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	result, err := h.tools.IgnoreCommunityReport(c.Request.Context(), moderationusecase.IgnoreCommunityReportInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		ReportID:      c.Param("id"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, reportContentResponse{Report: toContentReportResponse(result.Report)})
}

func (h *Handler) LockCommunityPost(c *gin.Context) {
	h.applySingleAction(c, c.Param("slug"), "post", c.Param("id"), "lock")
}

func (h *Handler) PinCommunityPost(c *gin.Context) {
	h.applySingleAction(c, c.Param("slug"), "post", c.Param("id"), "pin")
}

func (h *Handler) MarkCommunityPostNSFW(c *gin.Context) {
	h.applySingleAction(c, c.Param("slug"), "post", c.Param("id"), "mark_nsfw")
}

func (h *Handler) MarkCommunityPostSpoiler(c *gin.Context) {
	h.applySingleAction(c, c.Param("slug"), "post", c.Param("id"), "mark_spoiler")
}

func (h *Handler) SetCommunityPostFlair(c *gin.Context) {
	h.applySingleAction(c, c.Param("slug"), "post", c.Param("id"), "set_flair")
}

func (h *Handler) listModQueue(c *gin.Context, slug string) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	if h.tools == nil {
		_ = c.Error(apperr.New(apperr.CodeInternal, "moderation tools are not configured"))
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
	result, err := h.tools.ListModQueue(c.Request.Context(), moderationusecase.ListModQueueInput{
		ActorID:       userID,
		CommunitySlug: slug,
		Queue:         c.Query("queue"),
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := modQueueResponse{
		Items:      make([]modQueueItemResponse, 0, len(result.Items)),
		Queue:      result.Queue,
		Limit:      result.Limit,
		Offset:     result.Offset,
		NextOffset: result.NextOffset,
		HasMore:    result.HasMore,
	}
	for _, item := range result.Items {
		response.Items = append(response.Items, toModQueueItemResponse(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) applyBulkAction(c *gin.Context, slug string) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req bulkActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid moderation bulk action request"))
		c.Abort()
		return
	}
	targets := make([]moderationusecase.ModerationTargetInput, 0, len(req.Targets))
	for _, target := range req.Targets {
		targets = append(targets, moderationusecase.ModerationTargetInput(target))
	}
	result, err := h.tools.ApplyBulkAction(c.Request.Context(), moderationusecase.BulkActionInput{
		ActorID:         userID,
		CommunitySlug:   slug,
		Action:          req.Action,
		TargetType:      req.TargetType,
		TargetIDs:       req.TargetIDs,
		Targets:         targets,
		Reason:          req.Reason,
		RemovalReasonID: req.RemovalReasonID,
		NotifyAuthor:    req.NotifyAuthor,
		Confirm:         req.Confirm,
		Value:           req.Value,
		FlairText:       req.FlairText,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, toBulkActionResponse(result))
}

func (h *Handler) applySingleAction(c *gin.Context, slug string, targetType string, targetID string, action string) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req singleActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid moderation action request"))
		c.Abort()
		return
	}
	result, err := h.tools.ApplyBulkAction(c.Request.Context(), moderationusecase.BulkActionInput{
		ActorID:         userID,
		CommunitySlug:   slug,
		Action:          action,
		TargetType:      targetType,
		TargetIDs:       []string{targetID},
		Reason:          req.Reason,
		RemovalReasonID: req.RemovalReasonID,
		NotifyAuthor:    req.NotifyAuthor,
		Confirm:         req.Confirm,
		Value:           req.Value,
		FlairText:       req.FlairText,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, toBulkActionResponse(result))
}

func (h *Handler) ListRemovalReasons(c *gin.Context) {
	h.listTemplates(c, "removal_reason")
}

func (h *Handler) CreateRemovalReason(c *gin.Context) {
	h.createTemplate(c, "removal_reason")
}

func (h *Handler) UpdateRemovalReason(c *gin.Context) {
	h.updateTemplate(c, "removal_reason")
}

func (h *Handler) DeleteRemovalReason(c *gin.Context) {
	h.deleteTemplate(c, "removal_reason")
}

func (h *Handler) ApplyRemovalReason(c *gin.Context) {
	var req applyRemovalReasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid removal reason apply request"))
		c.Abort()
		return
	}
	h.applyBulkActionWithRequest(c, c.Param("slug"), bulkActionRequest{
		Action:          "remove",
		TargetType:      req.TargetType,
		TargetIDs:       req.TargetIDs,
		Targets:         req.Targets,
		Reason:          req.Reason,
		RemovalReasonID: c.Param("id"),
		NotifyAuthor:    req.NotifyAuthor,
		Confirm:         req.Confirm,
	})
}

func (h *Handler) ListSavedResponses(c *gin.Context) {
	h.listTemplates(c, "saved_response")
}

func (h *Handler) CreateSavedResponse(c *gin.Context) {
	h.createTemplate(c, "saved_response")
}

func (h *Handler) UpdateSavedResponse(c *gin.Context) {
	h.updateTemplate(c, "saved_response")
}

func (h *Handler) DeleteSavedResponse(c *gin.Context) {
	h.deleteTemplate(c, "saved_response")
}

func (h *Handler) listTemplates(c *gin.Context, kind string) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var result moderationusecase.ListModerationTemplatesResult
	var err error
	if kind == "saved_response" {
		result, err = h.tools.ListSavedResponses(c.Request.Context(), userID, c.Param("slug"))
	} else {
		result, err = h.tools.ListRemovalReasons(c.Request.Context(), userID, c.Param("slug"))
	}
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := templateListResponse{Items: make([]moderationTemplateResponse, 0, len(result.Templates))}
	for _, item := range result.Templates {
		response.Items = append(response.Items, toTemplateResponse(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) createTemplate(c *gin.Context, kind string) {
	h.writeTemplate(c, kind, true)
}

func (h *Handler) updateTemplate(c *gin.Context, kind string) {
	h.writeTemplate(c, kind, false)
}

func (h *Handler) writeTemplate(c *gin.Context, kind string, create bool) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req templateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid moderation template request"))
		c.Abort()
		return
	}
	input := moderationusecase.ModerationTemplateInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		ID:            c.Param("id"),
		Title:         req.Title,
		Body:          req.Body,
		RuleID:        req.RuleID,
		Position:      req.Position,
	}
	var result moderationusecase.ModerationTemplateResult
	var err error
	if kind == "saved_response" && create {
		result, err = h.tools.CreateSavedResponse(c.Request.Context(), input)
	} else if kind == "saved_response" {
		result, err = h.tools.UpdateSavedResponse(c.Request.Context(), input)
	} else if create {
		result, err = h.tools.CreateRemovalReason(c.Request.Context(), input)
	} else {
		result, err = h.tools.UpdateRemovalReason(c.Request.Context(), input)
	}
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	status := http.StatusOK
	if create {
		status = http.StatusCreated
	}
	c.JSON(status, templateResponse{Item: toTemplateResponse(result.Template)})
}

func (h *Handler) deleteTemplate(c *gin.Context, kind string) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	input := moderationusecase.DeleteModerationTemplateInput{ActorID: userID, CommunitySlug: c.Param("slug"), ID: c.Param("id")}
	var err error
	if kind == "saved_response" {
		err = h.tools.DeleteSavedResponse(c.Request.Context(), input)
	} else {
		err = h.tools.DeleteRemovalReason(c.Request.Context(), input)
	}
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBannedUsers(c *gin.Context) {
	h.listUserStates(c, moderationusecase.UserStateBanned)
}
func (h *Handler) UpsertBannedUser(c *gin.Context) {
	h.upsertUserState(c, moderationusecase.UserStateBanned)
}
func (h *Handler) DeleteBannedUser(c *gin.Context) {
	h.deleteUserState(c, moderationusecase.UserStateBanned)
}
func (h *Handler) ListMutedUsers(c *gin.Context) {
	h.listUserStates(c, moderationusecase.UserStateMuted)
}
func (h *Handler) UpsertMutedUser(c *gin.Context) {
	h.upsertUserState(c, moderationusecase.UserStateMuted)
}
func (h *Handler) DeleteMutedUser(c *gin.Context) {
	h.deleteUserState(c, moderationusecase.UserStateMuted)
}
func (h *Handler) ListApprovedUsers(c *gin.Context) {
	h.listUserStates(c, moderationusecase.UserStateApproved)
}
func (h *Handler) UpsertApprovedUser(c *gin.Context) {
	h.upsertUserState(c, moderationusecase.UserStateApproved)
}
func (h *Handler) DeleteApprovedUser(c *gin.Context) {
	h.deleteUserState(c, moderationusecase.UserStateApproved)
}

func (h *Handler) listUserStates(c *gin.Context, kind string) {
	userID, ok := h.currentUserID(c)
	if !ok {
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
	result, err := h.tools.ListUserStates(c.Request.Context(), moderationusecase.ListUserStatesInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		Kind:          kind,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := userStateListResponse{Users: make([]communityUserStateResponse, 0, len(result.Users)), Kind: result.Kind, Limit: result.Limit, Offset: result.Offset, NextOffset: result.NextOffset, HasMore: result.HasMore}
	for _, user := range result.Users {
		response.Users = append(response.Users, toUserStateResponse(user))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) upsertUserState(c *gin.Context, kind string) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req userStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid community user moderation request"))
		c.Abort()
		return
	}
	result, err := h.tools.UpsertUserState(c.Request.Context(), moderationusecase.WriteUserStateInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		Kind:          kind,
		UserID:        req.UserID,
		Reason:        req.Reason,
		ExpiresAt:     req.ExpiresAt,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, userStateResponse{User: toUserStateResponse(result.User)})
}

func (h *Handler) deleteUserState(c *gin.Context, kind string) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	if err := h.tools.DeleteUserState(c.Request.Context(), moderationusecase.DeleteUserStateInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		Kind:          kind,
		UserID:        c.Param("user_id"),
	}); err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetModerationUserProfile(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	result, err := h.tools.GetUserProfile(c.Request.Context(), moderationusecase.GetUserProfileInput{ActorID: userID, CommunitySlug: c.Param("slug"), UserID: c.Param("user_id")})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, toModerationUserProfileResponse(result.Profile))
}

func (h *Handler) ListModeratorNotes(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
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
	result, err := h.tools.ListModeratorNotes(c.Request.Context(), moderationusecase.ListModeratorNotesInput{ActorID: userID, CommunitySlug: c.Param("slug"), UserID: c.Param("user_id"), Limit: limit, Offset: offset})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := moderatorNotesResponse{Notes: make([]moderatorNoteResponse, 0, len(result.Notes)), Limit: result.Limit, Offset: result.Offset, NextOffset: result.NextOffset, HasMore: result.HasMore}
	for _, note := range result.Notes {
		response.Notes = append(response.Notes, toModeratorNoteResponse(note))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) CreateModeratorNote(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req moderatorNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid moderator note request"))
		c.Abort()
		return
	}
	result, err := h.tools.CreateModeratorNote(c.Request.Context(), moderationusecase.CreateModeratorNoteInput{ActorID: userID, CommunitySlug: c.Param("slug"), UserID: c.Param("user_id"), Body: req.Body})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusCreated, moderatorNoteMutationResponse{Note: toModeratorNoteResponse(result.Note)})
}

func (h *Handler) DeleteModeratorNote(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	if err := h.tools.DeleteModeratorNote(c.Request.Context(), moderationusecase.DeleteModeratorNoteInput{ActorID: userID, CommunitySlug: c.Param("slug"), UserID: c.Param("user_id"), NoteID: c.Param("note_id")}); err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListCommunityModLogs(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
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
	result, err := h.tools.ListCommunityModLogs(c.Request.Context(), moderationusecase.ListCommunityModLogsInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		Action:        c.Query("action"),
		ActorFilterID: c.Query("actor_id"),
		TargetType:    c.Query("target_type"),
		TargetID:      c.Query("target_id"),
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := modLogsResponse{Logs: make([]modLogResponse, 0, len(result.Logs)), Limit: result.Limit, Offset: result.Offset, NextOffset: result.NextOffset, HasMore: result.HasMore}
	for _, log := range result.Logs {
		response.Logs = append(response.Logs, toModLogResponse(log))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) applyBulkActionWithRequest(c *gin.Context, slug string, req bulkActionRequest) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	targets := make([]moderationusecase.ModerationTargetInput, 0, len(req.Targets))
	for _, target := range req.Targets {
		targets = append(targets, moderationusecase.ModerationTargetInput(target))
	}
	result, err := h.tools.ApplyBulkAction(c.Request.Context(), moderationusecase.BulkActionInput{
		ActorID:         userID,
		CommunitySlug:   slug,
		Action:          req.Action,
		TargetType:      req.TargetType,
		TargetIDs:       req.TargetIDs,
		Targets:         targets,
		Reason:          req.Reason,
		RemovalReasonID: req.RemovalReasonID,
		NotifyAuthor:    req.NotifyAuthor,
		Confirm:         req.Confirm,
		Value:           req.Value,
		FlairText:       req.FlairText,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, toBulkActionResponse(result))
}

func (h *Handler) currentUserID(c *gin.Context) (userdomain.UserID, bool) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return "", false
	}
	return userID, true
}

func toModQueueItemResponse(item moderationusecase.ModQueueItem) modQueueItemResponse {
	return modQueueItemResponse(item)
}

func toModQueueItemDetailResponse(detail moderationusecase.ModQueueItemDetail) modQueueItemDetailResponse {
	response := modQueueItemDetailResponse{
		Item:          toModQueueItemResponse(detail.Item),
		TargetPreview: toReportTargetPreviewResponse(detail.TargetPreview),
		Reports:       make([]modQueueReportResponse, 0, len(detail.Reports)),
		RecentActions: make([]moderationActionResponse, 0, len(detail.RecentActions)),
	}
	for _, report := range detail.Reports {
		response.Reports = append(response.Reports, modQueueReportResponse(report))
	}
	for _, action := range detail.RecentActions {
		response.RecentActions = append(response.RecentActions, toModerationActionResponse(action))
	}
	return response
}

func toModQueueSummaryResponse(summary moderationusecase.ModQueueSummary) modQueueSummaryResponse {
	response := modQueueSummaryResponse{
		Queues:        make([]modQueueCountResponse, 0, len(summary.Queues)),
		PriorityItems: make([]modQueueItemResponse, 0, len(summary.PriorityItems)),
	}
	for _, count := range summary.Queues {
		response.Queues = append(response.Queues, modQueueCountResponse(count))
	}
	for _, item := range summary.PriorityItems {
		response.PriorityItems = append(response.PriorityItems, toModQueueItemResponse(item))
	}
	return response
}

func toAutomodConfigView(config moderationusecase.AutomodConfig) automodConfigView {
	return automodConfigView{
		CommunityID: config.CommunityID,
		ConfigText:  config.ConfigText,
		Rules:       config.Rules,
		Version:     config.Version,
		UpdatedBy:   config.UpdatedBy,
		UpdatedAt:   config.UpdatedAt,
	}
}

func toAutomodVersionView(version moderationusecase.AutomodVersion) automodVersionView {
	return automodVersionView{
		ID:          version.ID,
		CommunityID: version.CommunityID,
		Version:     version.Version,
		ConfigText:  version.ConfigText,
		Rules:       version.Rules,
		UpdatedBy:   version.UpdatedBy,
		CreatedAt:   version.CreatedAt,
	}
}

func toContentControlsView(controls moderationusecase.ContentControls) contentControlsView {
	return contentControlsView{
		CommunityID:             controls.CommunityID,
		BlockedKeywords:         controls.BlockedKeywords,
		BlockedDomains:          controls.BlockedDomains,
		MinAccountAgeDays:       controls.MinAccountAgeDays,
		PostRateLimitPerHour:    controls.PostRateLimitPerHour,
		CommentRateLimitPerHour: controls.CommentRateLimitPerHour,
		BlockNewAccounts:        controls.BlockNewAccounts,
		FilterLinks:             controls.FilterLinks,
		UpdatedBy:               controls.UpdatedBy,
		UpdatedAt:               controls.UpdatedAt,
	}
}

func toBulkActionResponse(result moderationusecase.BulkActionResult) bulkActionResponse {
	response := bulkActionResponse{Results: make([]bulkActionItemResponse, 0, len(result.Results))}
	for _, item := range result.Results {
		var action *moderationActionResponse
		if item.Action != nil {
			converted := toModerationActionResponse(*item.Action)
			action = &converted
		}
		response.Results = append(response.Results, bulkActionItemResponse{TargetType: item.TargetType, TargetID: item.TargetID, OK: item.OK, Action: action, ErrorCode: item.ErrorCode, ErrorMessage: item.ErrorMessage})
	}
	return response
}

func toTemplateResponse(item moderationusecase.ModerationTemplate) moderationTemplateResponse {
	return moderationTemplateResponse(item)
}

func toUserStateResponse(user moderationusecase.CommunityUserState) communityUserStateResponse {
	return communityUserStateResponse(user)
}

func toModerationUserProfileResponse(profile moderationusecase.ModerationUserProfile) moderationUserProfileResponse {
	response := moderationUserProfileResponse{
		UserID:       profile.UserID,
		Username:     profile.Username,
		DisplayName:  profile.DisplayName,
		AvatarURL:    profile.AvatarURL,
		Headline:     profile.Headline,
		Status:       profile.Status,
		PostCount:    profile.PostCount,
		CommentCount: profile.CommentCount,
		ReportCount:  profile.ReportCount,
		RemovedCount: profile.RemovedCount,
		IsBanned:     profile.IsBanned,
		IsMuted:      profile.IsMuted,
		IsApproved:   profile.IsApproved,
		RecentNotes:  make([]moderatorNoteResponse, 0, len(profile.RecentNotes)),
	}
	for _, note := range profile.RecentNotes {
		response.RecentNotes = append(response.RecentNotes, toModeratorNoteResponse(note))
	}
	return response
}

func toModeratorNoteResponse(note moderationusecase.ModeratorNote) moderatorNoteResponse {
	return moderatorNoteResponse(note)
}

func toModLogResponse(log moderationusecase.CommunityModLog) modLogResponse {
	return modLogResponse(log)
}

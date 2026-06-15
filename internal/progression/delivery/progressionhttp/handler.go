package progressionhttp

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/progression/progressionusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	progression UseCase
}

type UseCase interface {
	GetMyProgression(ctx context.Context, input progressionusecase.GetMyProgressionInput) (progressionusecase.GetMyProgressionResult, error)
	ListMyXPEvents(ctx context.Context, input progressionusecase.ListMyXPEventsInput) (progressionusecase.ListMyXPEventsResult, error)
	ListMyTitles(ctx context.Context, input progressionusecase.ListMyTitlesInput) (progressionusecase.ListMyTitlesResult, error)
	SetActiveTitle(ctx context.Context, input progressionusecase.SetActiveTitleInput) (progressionusecase.SetActiveTitleResult, error)
	ListTitles(ctx context.Context, input progressionusecase.ListTitlesInput) (progressionusecase.ListTitlesResult, error)
	CreateTitle(ctx context.Context, input progressionusecase.CreateTitleInput) (progressionusecase.CreateTitleResult, error)
	UpdateTitle(ctx context.Context, input progressionusecase.UpdateTitleInput) (progressionusecase.UpdateTitleResult, error)
	ListUserTitleGrants(ctx context.Context, input progressionusecase.ListUserTitleGrantsInput) (progressionusecase.ListUserTitleGrantsResult, error)
	GrantTitle(ctx context.Context, input progressionusecase.GrantTitleInput) (progressionusecase.GrantTitleResult, error)
	RevokeTitle(ctx context.Context, input progressionusecase.RevokeTitleInput) (progressionusecase.RevokeTitleResult, error)
}

type progressionResponse struct {
	UserID         string                `json:"user_id"`
	XPTotal        int                   `json:"xp_total"`
	Level          int                   `json:"level"`
	LevelName      string                `json:"level_name"`
	CurrentLevelXP int                   `json:"current_level_xp"`
	NextLevelXP    *int                  `json:"next_level_xp"`
	LevelProgress  float64               `json:"level_progress"`
	ActiveTitle    *titleSummaryResponse `json:"active_title"`
	TitlesCount    int                   `json:"titles_count"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

type getMyProgressionResponse struct {
	Progression progressionResponse `json:"progression"`
}

type xpEventResponse struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Delta        int       `json:"delta"`
	XPTotalAfter int       `json:"xp_total_after"`
	Reason       string    `json:"reason"`
	SourceType   string    `json:"source_type"`
	SourceID     string    `json:"source_id"`
	ActorID      string    `json:"actor_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type listMyXPEventsResponse struct {
	Events     []xpEventResponse `json:"events"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
	NextOffset int               `json:"next_offset"`
	HasMore    bool              `json:"has_more"`
}

type titleSummaryResponse struct {
	GrantID   string `json:"grant_id"`
	TitleID   string `json:"title_id"`
	Name      string `json:"name"`
	ScopeType string `json:"scope_type"`
	ScopeID   string `json:"scope_id"`
}

type titleResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ScopeType   string    `json:"scope_type"`
	ScopeID     string    `json:"scope_id"`
	IsActive    bool      `json:"is_active"`
	CreatedBy   string    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type titleGrantResponse struct {
	ID        string        `json:"id"`
	UserID    string        `json:"user_id"`
	Title     titleResponse `json:"title"`
	GrantedBy string        `json:"granted_by,omitempty"`
	Reason    string        `json:"reason"`
	ExpiresAt *time.Time    `json:"expires_at"`
	RevokedAt *time.Time    `json:"revoked_at"`
	CreatedAt time.Time     `json:"created_at"`
}

type listMyTitlesResponse struct {
	Titles     []titleGrantResponse `json:"titles"`
	Limit      int                  `json:"limit"`
	Offset     int                  `json:"offset"`
	NextOffset int                  `json:"next_offset"`
	HasMore    bool                 `json:"has_more"`
}

type setActiveTitleRequest struct {
	TitleGrantID *string `json:"title_grant_id"`
}

type setActiveTitleResponse struct {
	Progression progressionResponse `json:"progression"`
}

type listAdminTitlesResponse struct {
	Titles     []titleResponse `json:"titles"`
	Limit      int             `json:"limit"`
	Offset     int             `json:"offset"`
	NextOffset int             `json:"next_offset"`
	HasMore    bool            `json:"has_more"`
}

type createAdminTitleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ScopeType   string `json:"scope_type"`
	ScopeID     string `json:"scope_id"`
}

type createAdminTitleResponse struct {
	Title titleResponse `json:"title"`
}

type updateAdminTitleRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

type updateAdminTitleResponse struct {
	Title titleResponse `json:"title"`
}

type listAdminUserTitleGrantsResponse struct {
	Titles     []titleGrantResponse `json:"titles"`
	Limit      int                  `json:"limit"`
	Offset     int                  `json:"offset"`
	NextOffset int                  `json:"next_offset"`
	HasMore    bool                 `json:"has_more"`
}

type grantAdminUserTitleRequest struct {
	TitleID   string     `json:"title_id" binding:"required"`
	Reason    string     `json:"reason"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type grantAdminUserTitleResponse struct {
	Grant titleGrantResponse `json:"grant"`
}

type revokeAdminUserTitleResponse struct {
	Grant titleGrantResponse `json:"grant"`
}

func NewHandler(progression UseCase) *Handler {
	return &Handler{progression: progression}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/me/progression", handler.GetMyProgression)
	group.GET("/me/xp-events", handler.ListMyXPEvents)
	group.GET("/me/titles", handler.ListMyTitles)
	group.PATCH("/me/title", handler.SetActiveTitle)
}

func RegisterAdminRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/admin/titles", handler.ListTitles)
	group.POST("/admin/titles", handler.CreateTitle)
	group.PATCH("/admin/titles/:id", handler.UpdateTitle)
	group.GET("/admin/users/:id/titles", handler.ListUserTitleGrants)
	group.POST("/admin/users/:id/titles", handler.GrantTitle)
	group.DELETE("/admin/users/:id/titles/:grant_id", handler.RevokeTitle)
}

func (h *Handler) GetMyProgression(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	result, err := h.progression.GetMyProgression(c.Request.Context(), progressionusecase.GetMyProgressionInput{UserID: userID})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, getMyProgressionResponse{Progression: toProgressionResponse(result.Progression)})
}

func (h *Handler) ListMyXPEvents(c *gin.Context) {
	userID, ok := currentUserID(c)
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
	result, err := h.progression.ListMyXPEvents(c.Request.Context(), progressionusecase.ListMyXPEventsInput{UserID: userID, Limit: limit, Offset: offset})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := listMyXPEventsResponse{Events: make([]xpEventResponse, 0, len(result.Events)), Limit: result.Limit, Offset: result.Offset, NextOffset: result.NextOffset, HasMore: result.HasMore}
	for _, event := range result.Events {
		response.Events = append(response.Events, toXPEventResponse(event))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) ListMyTitles(c *gin.Context) {
	userID, ok := currentUserID(c)
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
	result, err := h.progression.ListMyTitles(c.Request.Context(), progressionusecase.ListMyTitlesInput{UserID: userID, Limit: limit, Offset: offset})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := listMyTitlesResponse{Titles: make([]titleGrantResponse, 0, len(result.Titles)), Limit: result.Limit, Offset: result.Offset, NextOffset: result.NextOffset, HasMore: result.HasMore}
	for _, grant := range result.Titles {
		response.Titles = append(response.Titles, toTitleGrantResponse(grant))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) SetActiveTitle(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req setActiveTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid active title request"))
		c.Abort()
		return
	}
	result, err := h.progression.SetActiveTitle(c.Request.Context(), progressionusecase.SetActiveTitleInput{UserID: userID, TitleGrantID: req.TitleGrantID})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, setActiveTitleResponse{Progression: toProgressionResponse(result.Progression)})
}

func (h *Handler) ListTitles(c *gin.Context) {
	userID, ok := currentUserID(c)
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
	result, err := h.progression.ListTitles(c.Request.Context(), progressionusecase.ListTitlesInput{ActorID: userID, ScopeType: c.Query("scope_type"), Active: c.Query("active"), Limit: limit, Offset: offset})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := listAdminTitlesResponse{Titles: make([]titleResponse, 0, len(result.Titles)), Limit: result.Limit, Offset: result.Offset, NextOffset: result.NextOffset, HasMore: result.HasMore}
	for _, title := range result.Titles {
		response.Titles = append(response.Titles, toTitleResponse(title))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) CreateTitle(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req createAdminTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid admin title request"))
		c.Abort()
		return
	}
	result, err := h.progression.CreateTitle(c.Request.Context(), progressionusecase.CreateTitleInput{ActorID: userID, Name: req.Name, Description: req.Description, ScopeType: req.ScopeType, ScopeID: req.ScopeID})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusCreated, createAdminTitleResponse{Title: toTitleResponse(result.Title)})
}

func (h *Handler) UpdateTitle(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req updateAdminTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid admin title request"))
		c.Abort()
		return
	}
	result, err := h.progression.UpdateTitle(c.Request.Context(), progressionusecase.UpdateTitleInput{ActorID: userID, TitleID: c.Param("id"), Name: req.Name, Description: req.Description, IsActive: req.IsActive})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, updateAdminTitleResponse{Title: toTitleResponse(result.Title)})
}

func (h *Handler) ListUserTitleGrants(c *gin.Context) {
	userID, ok := currentUserID(c)
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
	result, err := h.progression.ListUserTitleGrants(c.Request.Context(), progressionusecase.ListUserTitleGrantsInput{ActorID: userID, UserID: c.Param("id"), Limit: limit, Offset: offset})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := listAdminUserTitleGrantsResponse{Titles: make([]titleGrantResponse, 0, len(result.Titles)), Limit: result.Limit, Offset: result.Offset, NextOffset: result.NextOffset, HasMore: result.HasMore}
	for _, grant := range result.Titles {
		response.Titles = append(response.Titles, toTitleGrantResponse(grant))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) GrantTitle(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req grantAdminUserTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid title grant request"))
		c.Abort()
		return
	}
	result, err := h.progression.GrantTitle(c.Request.Context(), progressionusecase.GrantTitleInput{ActorID: userID, UserID: c.Param("id"), TitleID: req.TitleID, Reason: req.Reason, ExpiresAt: req.ExpiresAt})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusCreated, grantAdminUserTitleResponse{Grant: toTitleGrantResponse(result.Grant)})
}

func (h *Handler) RevokeTitle(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	result, err := h.progression.RevokeTitle(c.Request.Context(), progressionusecase.RevokeTitleInput{ActorID: userID, UserID: c.Param("id"), GrantID: c.Param("grant_id")})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, revokeAdminUserTitleResponse{Grant: toTitleGrantResponse(result.Grant)})
}

func currentUserID(c *gin.Context) (userdomain.UserID, bool) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return "", false
	}
	return userID, true
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

func toProgressionResponse(progression progressionusecase.Progression) progressionResponse {
	return progressionResponse{
		UserID:         progression.UserID,
		XPTotal:        progression.XPTotal,
		Level:          progression.Level,
		LevelName:      progression.LevelName,
		CurrentLevelXP: progression.CurrentLevelXP,
		NextLevelXP:    progression.NextLevelXP,
		LevelProgress:  progression.LevelProgress,
		ActiveTitle:    toTitleSummaryResponse(progression.ActiveTitle),
		TitlesCount:    progression.TitlesCount,
		UpdatedAt:      progression.UpdatedAt,
	}
}

func toXPEventResponse(event progressionusecase.XPEvent) xpEventResponse {
	return xpEventResponse{
		ID:           event.ID,
		UserID:       event.UserID,
		Delta:        event.Delta,
		XPTotalAfter: event.XPTotalAfter,
		Reason:       event.Reason,
		SourceType:   event.SourceType,
		SourceID:     event.SourceID,
		ActorID:      event.ActorID,
		CreatedAt:    event.CreatedAt,
	}
}

func toTitleSummaryResponse(title *progressionusecase.TitleSummary) *titleSummaryResponse {
	if title == nil {
		return nil
	}
	return &titleSummaryResponse{GrantID: title.GrantID, TitleID: title.TitleID, Name: title.Name, ScopeType: title.ScopeType, ScopeID: title.ScopeID}
}

func toTitleResponse(title progressionusecase.Title) titleResponse {
	return titleResponse{
		ID:          title.ID,
		Name:        title.Name,
		Description: title.Description,
		ScopeType:   title.ScopeType,
		ScopeID:     title.ScopeID,
		IsActive:    title.IsActive,
		CreatedBy:   title.CreatedBy,
		CreatedAt:   title.CreatedAt,
		UpdatedAt:   title.UpdatedAt,
	}
}

func toTitleGrantResponse(grant progressionusecase.TitleGrant) titleGrantResponse {
	return titleGrantResponse{
		ID:        grant.ID,
		UserID:    grant.UserID,
		Title:     toTitleResponse(grant.Title),
		GrantedBy: grant.GrantedBy,
		Reason:    grant.Reason,
		ExpiresAt: grant.ExpiresAt,
		RevokedAt: grant.RevokedAt,
		CreatedAt: grant.CreatedAt,
	}
}

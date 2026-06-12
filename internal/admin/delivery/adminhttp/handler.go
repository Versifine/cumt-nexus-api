package adminhttp

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/admin/adminusecase"
	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	admin UseCase
}

type UseCase interface {
	ListUsers(ctx context.Context, input adminusecase.ListUsersInput) (adminusecase.ListUsersResult, error)
	UpdateUser(ctx context.Context, input adminusecase.UpdateUserInput) (adminusecase.UpdateUserResult, error)
	ListCommunities(ctx context.Context, input adminusecase.ListCommunitiesInput) (adminusecase.ListCommunitiesResult, error)
	UpdateCommunityStatus(ctx context.Context, input adminusecase.UpdateCommunityStatusInput) (adminusecase.UpdateCommunityStatusResult, error)
	ListEffects(ctx context.Context, input adminusecase.ListEffectsInput) (adminusecase.ListEffectsResult, error)
	UpdateEffectActive(ctx context.Context, input adminusecase.UpdateEffectActiveInput) (adminusecase.UpdateEffectActiveResult, error)
	ListSettings(ctx context.Context, input adminusecase.ListSettingsInput) (adminusecase.ListSettingsResult, error)
	UpdateSetting(ctx context.Context, input adminusecase.UpdateSettingInput) (adminusecase.UpdateSettingResult, error)
	ListAuditLogs(ctx context.Context, input adminusecase.ListAuditLogsInput) (adminusecase.ListAuditLogsResult, error)
}

type adminUserResponse struct {
	ID              string    `json:"id"`
	Username        string    `json:"username"`
	Status          string    `json:"status"`
	IsPlatformStaff bool      `json:"is_platform_staff"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type listAdminUsersResponse struct {
	Users  []adminUserResponse `json:"users"`
	Status string              `json:"status"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}

type updateAdminUserRequest struct {
	Status          *string `json:"status"`
	IsPlatformStaff *bool   `json:"is_platform_staff"`
}

type updateAdminUserResponse struct {
	User adminUserResponse `json:"user"`
}

type adminCommunityResponse struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"`
	Visibility  string    `json:"visibility"`
	CreatedBy   string    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type listAdminCommunitiesResponse struct {
	Communities []adminCommunityResponse `json:"communities"`
	Status      string                   `json:"status"`
	Limit       int                      `json:"limit"`
	Offset      int                      `json:"offset"`
}

type updateAdminCommunityStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type updateAdminCommunityStatusResponse struct {
	Community adminCommunityResponse `json:"community"`
}

type adminEffectResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	CostPoints   int       `json:"cost_points"`
	AssetURL     string    `json:"asset_url"`
	AnimationKey string    `json:"animation_key"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type listAdminEffectsResponse struct {
	Effects []adminEffectResponse `json:"effects"`
	Active  string                `json:"active"`
	Limit   int                   `json:"limit"`
	Offset  int                   `json:"offset"`
}

type updateAdminEffectRequest struct {
	IsActive *bool `json:"is_active" binding:"required"`
}

type updateAdminEffectResponse struct {
	Effect adminEffectResponse `json:"effect"`
}

type adminSettingResponse struct {
	Key       string    `json:"key"`
	Enabled   bool      `json:"enabled"`
	UpdatedBy string    `json:"updated_by,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type listAdminSettingsResponse struct {
	Settings []adminSettingResponse `json:"settings"`
}

type updateAdminSettingRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

type updateAdminSettingResponse struct {
	Setting adminSettingResponse `json:"setting"`
}

type adminAuditLogResponse struct {
	ID         string         `json:"id"`
	ActorID    string         `json:"actor_id"`
	Action     string         `json:"action"`
	TargetType string         `json:"target_type"`
	TargetID   string         `json:"target_id"`
	Before     map[string]any `json:"before"`
	After      map[string]any `json:"after"`
	CreatedAt  time.Time      `json:"created_at"`
}

type listAdminAuditLogsResponse struct {
	AuditLogs []adminAuditLogResponse `json:"audit_logs"`
	Limit     int                     `json:"limit"`
	Offset    int                     `json:"offset"`
}

func NewHandler(admin UseCase) *Handler {
	return &Handler{
		admin: admin,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/admin/users", handler.ListUsers)
	group.PATCH("/admin/users/:id", handler.UpdateUser)
	group.GET("/admin/communities", handler.ListCommunities)
	group.PATCH("/admin/communities/:id", handler.UpdateCommunityStatus)
	group.GET("/admin/effects", handler.ListEffects)
	group.PATCH("/admin/effects/:id", handler.UpdateEffectActive)
	group.GET("/admin/settings", handler.ListSettings)
	group.PATCH("/admin/settings/:key", handler.UpdateSetting)
	group.GET("/admin/audit-logs", handler.ListAuditLogs)
}

func (h *Handler) ListUsers(c *gin.Context) {
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
	result, err := h.admin.ListUsers(c.Request.Context(), adminusecase.ListUsersInput{
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
	response := listAdminUsersResponse{
		Users:  make([]adminUserResponse, 0, len(result.Users)),
		Status: result.Status,
		Limit:  result.Limit,
		Offset: result.Offset,
	}
	for _, user := range result.Users {
		response.Users = append(response.Users, toAdminUserResponse(user))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req updateAdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid admin user request"))
		c.Abort()
		return
	}
	result, err := h.admin.UpdateUser(c.Request.Context(), adminusecase.UpdateUserInput{
		ActorID:         userID,
		UserID:          c.Param("id"),
		Status:          req.Status,
		IsPlatformStaff: req.IsPlatformStaff,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, updateAdminUserResponse{User: toAdminUserResponse(result.User)})
}

func (h *Handler) ListCommunities(c *gin.Context) {
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
	result, err := h.admin.ListCommunities(c.Request.Context(), adminusecase.ListCommunitiesInput{
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
	response := listAdminCommunitiesResponse{
		Communities: make([]adminCommunityResponse, 0, len(result.Communities)),
		Status:      result.Status,
		Limit:       result.Limit,
		Offset:      result.Offset,
	}
	for _, community := range result.Communities {
		response.Communities = append(response.Communities, toAdminCommunityResponse(community))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) UpdateCommunityStatus(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req updateAdminCommunityStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid admin community request"))
		c.Abort()
		return
	}
	result, err := h.admin.UpdateCommunityStatus(c.Request.Context(), adminusecase.UpdateCommunityStatusInput{
		ActorID:     userID,
		CommunityID: c.Param("id"),
		Status:      req.Status,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, updateAdminCommunityStatusResponse{Community: toAdminCommunityResponse(result.Community)})
}

func (h *Handler) ListEffects(c *gin.Context) {
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
	result, err := h.admin.ListEffects(c.Request.Context(), adminusecase.ListEffectsInput{
		ActorID: userID,
		Active:  c.Query("active"),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := listAdminEffectsResponse{
		Effects: make([]adminEffectResponse, 0, len(result.Effects)),
		Active:  result.Active,
		Limit:   result.Limit,
		Offset:  result.Offset,
	}
	for _, effect := range result.Effects {
		response.Effects = append(response.Effects, toAdminEffectResponse(effect))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) UpdateEffectActive(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req updateAdminEffectRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.IsActive == nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid admin effect request"))
		c.Abort()
		return
	}
	result, err := h.admin.UpdateEffectActive(c.Request.Context(), adminusecase.UpdateEffectActiveInput{
		ActorID:  userID,
		EffectID: c.Param("id"),
		IsActive: *req.IsActive,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, updateAdminEffectResponse{Effect: toAdminEffectResponse(result.Effect)})
}

func (h *Handler) ListSettings(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	result, err := h.admin.ListSettings(c.Request.Context(), adminusecase.ListSettingsInput{
		ActorID: userID,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := listAdminSettingsResponse{
		Settings: make([]adminSettingResponse, 0, len(result.Settings)),
	}
	for _, setting := range result.Settings {
		response.Settings = append(response.Settings, toAdminSettingResponse(setting))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) UpdateSetting(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req updateAdminSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid admin setting request"))
		c.Abort()
		return
	}
	result, err := h.admin.UpdateSetting(c.Request.Context(), adminusecase.UpdateSettingInput{
		ActorID: userID,
		Key:     c.Param("key"),
		Enabled: *req.Enabled,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, updateAdminSettingResponse{Setting: toAdminSettingResponse(result.Setting)})
}

func (h *Handler) ListAuditLogs(c *gin.Context) {
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
	result, err := h.admin.ListAuditLogs(c.Request.Context(), adminusecase.ListAuditLogsInput{
		ActorID:    userID,
		TargetType: c.Query("target_type"),
		TargetID:   c.Query("target_id"),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := listAdminAuditLogsResponse{
		AuditLogs: make([]adminAuditLogResponse, 0, len(result.AuditLogs)),
		Limit:     result.Limit,
		Offset:    result.Offset,
	}
	for _, log := range result.AuditLogs {
		response.AuditLogs = append(response.AuditLogs, toAdminAuditLogResponse(log))
	}
	c.JSON(http.StatusOK, response)
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

func toAdminUserResponse(user adminusecase.User) adminUserResponse {
	return adminUserResponse{
		ID:              user.ID,
		Username:        user.Username,
		Status:          user.Status,
		IsPlatformStaff: user.IsPlatformStaff,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}
}

func toAdminCommunityResponse(community adminusecase.Community) adminCommunityResponse {
	return adminCommunityResponse{
		ID:          community.ID,
		Slug:        community.Slug,
		Name:        community.Name,
		Description: community.Description,
		Kind:        community.Kind,
		Status:      community.Status,
		Visibility:  community.Visibility,
		CreatedBy:   community.CreatedBy,
		CreatedAt:   community.CreatedAt,
		UpdatedAt:   community.UpdatedAt,
	}
}

func toAdminEffectResponse(effect adminusecase.Effect) adminEffectResponse {
	return adminEffectResponse{
		ID:           effect.ID,
		Name:         effect.Name,
		Description:  effect.Description,
		CostPoints:   effect.CostPoints,
		AssetURL:     effect.AssetURL,
		AnimationKey: effect.AnimationKey,
		IsActive:     effect.IsActive,
		CreatedAt:    effect.CreatedAt,
		UpdatedAt:    effect.UpdatedAt,
	}
}

func toAdminSettingResponse(setting adminusecase.Setting) adminSettingResponse {
	return adminSettingResponse{
		Key:       setting.Key,
		Enabled:   setting.Enabled,
		UpdatedBy: setting.UpdatedBy,
		UpdatedAt: setting.UpdatedAt,
	}
}

func toAdminAuditLogResponse(log adminusecase.AuditLog) adminAuditLogResponse {
	return adminAuditLogResponse{
		ID:         log.ID,
		ActorID:    log.ActorID,
		Action:     log.Action,
		TargetType: log.TargetType,
		TargetID:   log.TargetID,
		Before:     log.Before,
		After:      log.After,
		CreatedAt:  log.CreatedAt,
	}
}

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
	UpdateUserPlatformRole(ctx context.Context, input adminusecase.UpdateUserPlatformRoleInput) (adminusecase.UpdateUserPlatformRoleResult, error)
	GetCurrentOwnerTransfer(ctx context.Context, input adminusecase.GetCurrentOwnerTransferInput) (adminusecase.GetCurrentOwnerTransferResult, error)
	CreateOwnerTransfer(ctx context.Context, input adminusecase.CreateOwnerTransferInput) (adminusecase.CreateOwnerTransferResult, error)
	CancelOwnerTransfer(ctx context.Context, input adminusecase.CancelOwnerTransferInput) (adminusecase.CancelOwnerTransferResult, error)
	GetOwnerTransfer(ctx context.Context, input adminusecase.GetOwnerTransferInput) (adminusecase.GetOwnerTransferResult, error)
	AcceptOwnerTransfer(ctx context.Context, input adminusecase.AcceptOwnerTransferInput) (adminusecase.AcceptOwnerTransferResult, error)
	ListCommunities(ctx context.Context, input adminusecase.ListCommunitiesInput) (adminusecase.ListCommunitiesResult, error)
	UpdateCommunityStatus(ctx context.Context, input adminusecase.UpdateCommunityStatusInput) (adminusecase.UpdateCommunityStatusResult, error)
	UpdateCommunityOwner(ctx context.Context, input adminusecase.UpdateCommunityOwnerInput) (adminusecase.UpdateCommunityOwnerResult, error)
	ListEffects(ctx context.Context, input adminusecase.ListEffectsInput) (adminusecase.ListEffectsResult, error)
	UpdateEffectActive(ctx context.Context, input adminusecase.UpdateEffectActiveInput) (adminusecase.UpdateEffectActiveResult, error)
	ListSettings(ctx context.Context, input adminusecase.ListSettingsInput) (adminusecase.ListSettingsResult, error)
	UpdateSetting(ctx context.Context, input adminusecase.UpdateSettingInput) (adminusecase.UpdateSettingResult, error)
	ListAuditLogs(ctx context.Context, input adminusecase.ListAuditLogsInput) (adminusecase.ListAuditLogsResult, error)
	ListPointTransactions(ctx context.Context, input adminusecase.ListPointTransactionsInput) (adminusecase.ListPointTransactionsResult, error)
	AdjustUserPoints(ctx context.Context, input adminusecase.AdjustUserPointsInput) (adminusecase.AdjustUserPointsResult, error)
	CreateUserSanction(ctx context.Context, input adminusecase.CreateUserSanctionInput) (adminusecase.CreateUserSanctionResult, error)
	ListUserSanctions(ctx context.Context, input adminusecase.ListUserSanctionsInput) (adminusecase.ListUserSanctionsResult, error)
	RevokeUserSanction(ctx context.Context, input adminusecase.RevokeUserSanctionInput) (adminusecase.RevokeUserSanctionResult, error)
}

type adminUserResponse struct {
	ID              string    `json:"id"`
	Username        string    `json:"username"`
	Status          string    `json:"status"`
	IsPlatformStaff bool      `json:"is_platform_staff"`
	PlatformRole    string    `json:"platform_role"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type listAdminUsersResponse struct {
	Users      []adminUserResponse `json:"users"`
	Status     string              `json:"status"`
	Query      string              `json:"q,omitempty"`
	Limit      int                 `json:"limit"`
	Offset     int                 `json:"offset"`
	NextOffset int                 `json:"next_offset"`
	HasMore    bool                `json:"has_more"`
}

type updateAdminUserRequest struct {
	Status          *string `json:"status"`
	IsPlatformStaff *bool   `json:"is_platform_staff"`
}

type updateAdminUserResponse struct {
	User adminUserResponse `json:"user"`
}

type updateAdminUserPlatformRoleRequest struct {
	Role *string `json:"role"`
}

type updateAdminUserPlatformRoleResponse struct {
	User adminUserResponse `json:"user"`
}

type createOwnerTransferRequest struct {
	TargetUserID      string  `json:"target_user_id"`
	PreviousOwnerRole *string `json:"previous_owner_role"`
	Reason            string  `json:"reason"`
	CurrentPassword   string  `json:"current_password"`
}

type acceptOwnerTransferRequest struct {
	CurrentPassword string `json:"current_password"`
}

type ownerTransferResponse struct {
	Transfer *adminOwnerTransferResponse `json:"transfer"`
}

type adminOwnerTransferResponse struct {
	ID                  string     `json:"id"`
	Status              string     `json:"status"`
	InitiatedByID       string     `json:"initiated_by_id"`
	InitiatedByUsername string     `json:"initiated_by_username"`
	TargetUserID        string     `json:"target_user_id"`
	TargetUsername      string     `json:"target_username"`
	PreviousOwnerRole   string     `json:"previous_owner_role"`
	Reason              string     `json:"reason"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	ExpiresAt           time.Time  `json:"expires_at"`
	AcceptedAt          *time.Time `json:"accepted_at"`
	CancelledAt         *time.Time `json:"cancelled_at"`
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
	Query       string                   `json:"q,omitempty"`
	Limit       int                      `json:"limit"`
	Offset      int                      `json:"offset"`
	NextOffset  int                      `json:"next_offset"`
	HasMore     bool                     `json:"has_more"`
}

type updateAdminCommunityStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type updateAdminCommunityStatusResponse struct {
	Community adminCommunityResponse `json:"community"`
}

type updateAdminCommunityOwnerRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Reason string `json:"reason"`
}

type adminCommunityOwnerResponse struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type updateAdminCommunityOwnerResponse struct {
	Community adminCommunityResponse      `json:"community"`
	Owner     adminCommunityOwnerResponse `json:"owner"`
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
	Effects    []adminEffectResponse `json:"effects"`
	Active     string                `json:"active"`
	Limit      int                   `json:"limit"`
	Offset     int                   `json:"offset"`
	NextOffset int                   `json:"next_offset"`
	HasMore    bool                  `json:"has_more"`
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
	AuditLogs  []adminAuditLogResponse `json:"audit_logs"`
	Query      string                  `json:"q,omitempty"`
	Limit      int                     `json:"limit"`
	Offset     int                     `json:"offset"`
	NextOffset int                     `json:"next_offset"`
	HasMore    bool                    `json:"has_more"`
}

type adminPointAccountResponse struct {
	UserID         string    `json:"user_id"`
	Balance        int       `json:"balance"`
	LifetimeEarned int       `json:"lifetime_earned"`
	LifetimeSpent  int       `json:"lifetime_spent"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type adminPointTransactionResponse struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Delta        int       `json:"delta"`
	BalanceAfter int       `json:"balance_after"`
	Reason       string    `json:"reason"`
	SourceType   string    `json:"source_type"`
	SourceID     string    `json:"source_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type listAdminPointTransactionsResponse struct {
	Transactions []adminPointTransactionResponse `json:"transactions"`
	Limit        int                             `json:"limit"`
	Offset       int                             `json:"offset"`
	NextOffset   int                             `json:"next_offset"`
	HasMore      bool                            `json:"has_more"`
}

type adjustAdminUserPointsRequest struct {
	Delta  int    `json:"delta" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

type adjustAdminUserPointsResponse struct {
	Account     adminPointAccountResponse     `json:"account"`
	Transaction adminPointTransactionResponse `json:"transaction"`
}

type adminUserSanctionResponse struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Type      string     `json:"type"`
	Status    string     `json:"status"`
	Reason    string     `json:"reason"`
	CreatedBy string     `json:"created_by"`
	StartsAt  time.Time  `json:"starts_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	RevokedBy string     `json:"revoked_by,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type createAdminUserSanctionRequest struct {
	Type     string `json:"type" binding:"required"`
	Duration string `json:"duration" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
}

type createAdminUserSanctionResponse struct {
	Sanction adminUserSanctionResponse `json:"sanction"`
}

type listAdminUserSanctionsResponse struct {
	Sanctions  []adminUserSanctionResponse `json:"sanctions"`
	Limit      int                         `json:"limit"`
	Offset     int                         `json:"offset"`
	NextOffset int                         `json:"next_offset"`
	HasMore    bool                        `json:"has_more"`
}

type revokeAdminUserSanctionResponse struct {
	Sanction adminUserSanctionResponse `json:"sanction"`
}

func NewHandler(admin UseCase) *Handler {
	return &Handler{
		admin: admin,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/admin/users", handler.ListUsers)
	group.PATCH("/admin/users/:id", handler.UpdateUser)
	group.PATCH("/admin/users/:id/platform-role", handler.UpdateUserPlatformRole)
	group.GET("/admin/owner-transfer", handler.GetCurrentOwnerTransfer)
	group.POST("/admin/owner-transfer", handler.CreateOwnerTransfer)
	group.DELETE("/admin/owner-transfer/:transfer_id", handler.CancelOwnerTransfer)
	group.GET("/admin/communities", handler.ListCommunities)
	group.PATCH("/admin/communities/:id", handler.UpdateCommunityStatus)
	group.POST("/admin/communities/:id/owner", handler.UpdateCommunityOwner)
	group.GET("/admin/effects", handler.ListEffects)
	group.PATCH("/admin/effects/:id", handler.UpdateEffectActive)
	group.GET("/admin/settings", handler.ListSettings)
	group.PATCH("/admin/settings/:key", handler.UpdateSetting)
	group.GET("/admin/audit-logs", handler.ListAuditLogs)
	group.GET("/admin/point-transactions", handler.ListPointTransactions)
	group.POST("/admin/users/:id/points/adjust", handler.AdjustUserPoints)
	group.POST("/admin/users/:id/sanctions", handler.CreateUserSanction)
	group.GET("/admin/users/:id/sanctions", handler.ListUserSanctions)
	group.POST("/admin/user-sanctions/:sanction_id/revoke", handler.RevokeUserSanction)
	group.GET("/owner-transfer/:transfer_id", handler.GetOwnerTransfer)
	group.POST("/owner-transfer/:transfer_id/accept", handler.AcceptOwnerTransfer)
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
		Query:   c.Query("q"),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := listAdminUsersResponse{
		Users:      make([]adminUserResponse, 0, len(result.Users)),
		Status:     result.Status,
		Query:      result.Query,
		Limit:      result.Limit,
		Offset:     result.Offset,
		NextOffset: result.NextOffset,
		HasMore:    result.HasMore,
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

func (h *Handler) UpdateUserPlatformRole(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req updateAdminUserPlatformRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid admin user platform role request"))
		c.Abort()
		return
	}
	result, err := h.admin.UpdateUserPlatformRole(c.Request.Context(), adminusecase.UpdateUserPlatformRoleInput{
		ActorID: userID,
		UserID:  c.Param("id"),
		Role:    req.Role,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, updateAdminUserPlatformRoleResponse{User: toAdminUserResponse(result.User)})
}

func (h *Handler) GetCurrentOwnerTransfer(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	result, err := h.admin.GetCurrentOwnerTransfer(c.Request.Context(), adminusecase.GetCurrentOwnerTransferInput{ActorID: userID})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, ownerTransferResponse{Transfer: toAdminOwnerTransferResponsePtr(result.Transfer)})
}

func (h *Handler) CreateOwnerTransfer(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req createOwnerTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid owner transfer request"))
		c.Abort()
		return
	}
	result, err := h.admin.CreateOwnerTransfer(c.Request.Context(), adminusecase.CreateOwnerTransferInput{
		ActorID:           userID,
		TargetUserID:      req.TargetUserID,
		PreviousOwnerRole: req.PreviousOwnerRole,
		Reason:            req.Reason,
		CurrentPassword:   req.CurrentPassword,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := toAdminOwnerTransferResponse(result.Transfer)
	c.JSON(http.StatusCreated, ownerTransferResponse{Transfer: &response})
}

func (h *Handler) CancelOwnerTransfer(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	result, err := h.admin.CancelOwnerTransfer(c.Request.Context(), adminusecase.CancelOwnerTransferInput{
		ActorID:    userID,
		TransferID: c.Param("transfer_id"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := toAdminOwnerTransferResponse(result.Transfer)
	c.JSON(http.StatusOK, ownerTransferResponse{Transfer: &response})
}

func (h *Handler) GetOwnerTransfer(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	result, err := h.admin.GetOwnerTransfer(c.Request.Context(), adminusecase.GetOwnerTransferInput{
		ActorID:    userID,
		TransferID: c.Param("transfer_id"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := toAdminOwnerTransferResponse(result.Transfer)
	c.JSON(http.StatusOK, ownerTransferResponse{Transfer: &response})
}

func (h *Handler) AcceptOwnerTransfer(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req acceptOwnerTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid owner transfer accept request"))
		c.Abort()
		return
	}
	result, err := h.admin.AcceptOwnerTransfer(c.Request.Context(), adminusecase.AcceptOwnerTransferInput{
		ActorID:         userID,
		TransferID:      c.Param("transfer_id"),
		CurrentPassword: req.CurrentPassword,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := toAdminOwnerTransferResponse(result.Transfer)
	c.JSON(http.StatusOK, ownerTransferResponse{Transfer: &response})
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
		Query:   c.Query("q"),
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
		Query:       result.Query,
		Limit:       result.Limit,
		Offset:      result.Offset,
		NextOffset:  result.NextOffset,
		HasMore:     result.HasMore,
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

func (h *Handler) UpdateCommunityOwner(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req updateAdminCommunityOwnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid admin community owner request"))
		c.Abort()
		return
	}
	result, err := h.admin.UpdateCommunityOwner(c.Request.Context(), adminusecase.UpdateCommunityOwnerInput{
		ActorID:     userID,
		CommunityID: c.Param("id"),
		UserID:      req.UserID,
		Reason:      req.Reason,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, updateAdminCommunityOwnerResponse{
		Community: toAdminCommunityResponse(result.Community),
		Owner:     toAdminCommunityOwnerResponse(result.Owner),
	})
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
		Effects:    make([]adminEffectResponse, 0, len(result.Effects)),
		Active:     result.Active,
		Limit:      result.Limit,
		Offset:     result.Offset,
		NextOffset: result.NextOffset,
		HasMore:    result.HasMore,
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
		Query:      c.Query("q"),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := listAdminAuditLogsResponse{
		AuditLogs:  make([]adminAuditLogResponse, 0, len(result.AuditLogs)),
		Query:      result.Query,
		Limit:      result.Limit,
		Offset:     result.Offset,
		NextOffset: result.NextOffset,
		HasMore:    result.HasMore,
	}
	for _, log := range result.AuditLogs {
		response.AuditLogs = append(response.AuditLogs, toAdminAuditLogResponse(log))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) ListPointTransactions(c *gin.Context) {
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
	result, err := h.admin.ListPointTransactions(c.Request.Context(), adminusecase.ListPointTransactionsInput{
		ActorID: userID,
		UserID:  c.Query("user_id"),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := listAdminPointTransactionsResponse{
		Transactions: make([]adminPointTransactionResponse, 0, len(result.Transactions)),
		Limit:        result.Limit,
		Offset:       result.Offset,
		NextOffset:   result.NextOffset,
		HasMore:      result.HasMore,
	}
	for _, transaction := range result.Transactions {
		response.Transactions = append(response.Transactions, toAdminPointTransactionResponse(transaction))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) AdjustUserPoints(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req adjustAdminUserPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid admin point adjustment request"))
		c.Abort()
		return
	}
	result, err := h.admin.AdjustUserPoints(c.Request.Context(), adminusecase.AdjustUserPointsInput{
		ActorID: userID,
		UserID:  c.Param("id"),
		Delta:   req.Delta,
		Reason:  req.Reason,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, adjustAdminUserPointsResponse{
		Account:     toAdminPointAccountResponse(result.Account),
		Transaction: toAdminPointTransactionResponse(result.Transaction),
	})
}

func (h *Handler) CreateUserSanction(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req createAdminUserSanctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid admin user sanction request"))
		c.Abort()
		return
	}
	result, err := h.admin.CreateUserSanction(c.Request.Context(), adminusecase.CreateUserSanctionInput{
		ActorID:  userID,
		UserID:   c.Param("id"),
		Type:     req.Type,
		Duration: req.Duration,
		Reason:   req.Reason,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusCreated, createAdminUserSanctionResponse{Sanction: toAdminUserSanctionResponse(result.Sanction)})
}

func (h *Handler) ListUserSanctions(c *gin.Context) {
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
	result, err := h.admin.ListUserSanctions(c.Request.Context(), adminusecase.ListUserSanctionsInput{
		ActorID: userID,
		UserID:  c.Param("id"),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := listAdminUserSanctionsResponse{
		Sanctions:  make([]adminUserSanctionResponse, 0, len(result.Sanctions)),
		Limit:      result.Limit,
		Offset:     result.Offset,
		NextOffset: result.NextOffset,
		HasMore:    result.HasMore,
	}
	for _, sanction := range result.Sanctions {
		response.Sanctions = append(response.Sanctions, toAdminUserSanctionResponse(sanction))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) RevokeUserSanction(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	result, err := h.admin.RevokeUserSanction(c.Request.Context(), adminusecase.RevokeUserSanctionInput{
		ActorID:    userID,
		SanctionID: c.Param("sanction_id"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, revokeAdminUserSanctionResponse{Sanction: toAdminUserSanctionResponse(result.Sanction)})
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
		PlatformRole:    user.PlatformRole,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}
}

func toAdminOwnerTransferResponsePtr(transfer *adminusecase.OwnerTransfer) *adminOwnerTransferResponse {
	if transfer == nil {
		return nil
	}
	response := toAdminOwnerTransferResponse(*transfer)
	return &response
}

func toAdminOwnerTransferResponse(transfer adminusecase.OwnerTransfer) adminOwnerTransferResponse {
	return adminOwnerTransferResponse{
		ID:                  transfer.ID,
		Status:              transfer.Status,
		InitiatedByID:       transfer.InitiatedByID,
		InitiatedByUsername: transfer.InitiatedByUsername,
		TargetUserID:        transfer.TargetUserID,
		TargetUsername:      transfer.TargetUsername,
		PreviousOwnerRole:   transfer.PreviousOwnerRole,
		Reason:              transfer.Reason,
		CreatedAt:           transfer.CreatedAt,
		UpdatedAt:           transfer.UpdatedAt,
		ExpiresAt:           transfer.ExpiresAt,
		AcceptedAt:          transfer.AcceptedAt,
		CancelledAt:         transfer.CancelledAt,
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

func toAdminCommunityOwnerResponse(owner adminusecase.CommunityOwnerMember) adminCommunityOwnerResponse {
	return adminCommunityOwnerResponse{
		UserID:    owner.UserID,
		Username:  owner.Username,
		Role:      owner.Role,
		Status:    owner.Status,
		UpdatedAt: owner.UpdatedAt,
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

func toAdminPointAccountResponse(account adminusecase.PointAccount) adminPointAccountResponse {
	return adminPointAccountResponse{
		UserID:         account.UserID,
		Balance:        account.Balance,
		LifetimeEarned: account.LifetimeEarned,
		LifetimeSpent:  account.LifetimeSpent,
		UpdatedAt:      account.UpdatedAt,
	}
}

func toAdminPointTransactionResponse(transaction adminusecase.PointTransaction) adminPointTransactionResponse {
	return adminPointTransactionResponse{
		ID:           transaction.ID,
		UserID:       transaction.UserID,
		Delta:        transaction.Delta,
		BalanceAfter: transaction.BalanceAfter,
		Reason:       transaction.Reason,
		SourceType:   transaction.SourceType,
		SourceID:     transaction.SourceID,
		CreatedAt:    transaction.CreatedAt,
	}
}

func toAdminUserSanctionResponse(sanction adminusecase.UserSanction) adminUserSanctionResponse {
	return adminUserSanctionResponse{
		ID:        sanction.ID,
		UserID:    sanction.UserID,
		Type:      sanction.Type,
		Status:    sanction.Status,
		Reason:    sanction.Reason,
		CreatedBy: sanction.CreatedBy,
		StartsAt:  sanction.StartsAt,
		ExpiresAt: sanction.ExpiresAt,
		RevokedBy: sanction.RevokedBy,
		RevokedAt: sanction.RevokedAt,
		CreatedAt: sanction.CreatedAt,
		UpdatedAt: sanction.UpdatedAt,
	}
}

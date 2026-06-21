package userhttp

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
	"github.com/Versifine/cumt-nexus-api/internal/user/userusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	currentUser    CurrentUserUseCase
	publicUsers    PublicUserUseCase
	profileUpdater ProfileUpdateUseCase
}

type CurrentUserUseCase interface {
	GetCurrentUser(ctx context.Context, input userusecase.CurrentUserInput) (userusecase.CurrentUserResult, error)
}

type PublicUserUseCase interface {
	GetPublicUser(ctx context.Context, input userusecase.GetPublicUserInput) (userusecase.GetPublicUserResult, error)
	FollowUser(ctx context.Context, input userusecase.FollowUserInput) (userusecase.FollowUserResult, error)
	DeleteUserFollow(ctx context.Context, input userusecase.DeleteUserFollowInput) (userusecase.DeleteUserFollowResult, error)
	ListFollowedUsers(ctx context.Context, input userusecase.ListFollowedUsersInput) (userusecase.ListFollowedUsersResult, error)
}

type ProfileUpdateUseCase interface {
	UpdateProfile(ctx context.Context, input userusecase.UpdateProfileInput) (userusecase.UpdateProfileResult, error)
}

type currentUserResponse struct {
	ID              string    `json:"id"`
	Username        string    `json:"username"`
	Status          string    `json:"status"`
	IsPlatformStaff bool      `json:"is_platform_staff"`
	PlatformRole    string    `json:"platform_role"`
	CreatedAt       time.Time `json:"created_at"`
}

type publicUserResponse struct {
	ID                string                         `json:"id"`
	Username          string                         `json:"username"`
	DisplayName       string                         `json:"display_name"`
	AvatarURL         string                         `json:"avatar_url"`
	BannerURL         string                         `json:"banner_url"`
	Headline          string                         `json:"headline"`
	Bio               string                         `json:"bio"`
	Badges            []string                       `json:"badges"`
	Roles             []string                       `json:"roles"`
	Status            string                         `json:"status"`
	IsPlatformStaff   bool                           `json:"is_platform_staff"`
	PlatformRole      string                         `json:"platform_role"`
	Stats             publicUserStatsResponse        `json:"stats"`
	Progression       publicUserProgressionResponse  `json:"progression"`
	DMCapability      publicUserDMCapabilityResponse `json:"dm_capability"`
	ViewerIsFollowing bool                           `json:"viewer_is_following"`
	CreatedAt         time.Time                      `json:"created_at"`
}

type publicUserDMCapabilityResponse struct {
	CanStart             bool    `json:"can_start"`
	RequiresRequest      bool    `json:"requires_request"`
	Reason               *string `json:"reason"`
	DirectConversationID *string `json:"direct_conversation_id"`
	ViewerRelation       string  `json:"viewer_relation"`
}

type publicUserProgressionResponse struct {
	Level          int                      `json:"level"`
	LevelName      string                   `json:"level_name"`
	XPTotal        int                      `json:"xp_total"`
	CurrentLevelXP int                      `json:"current_level_xp"`
	NextLevelXP    *int                     `json:"next_level_xp"`
	LevelProgress  float64                  `json:"level_progress"`
	ActiveTitle    *publicUserTitleResponse `json:"active_title"`
	TitlesCount    int                      `json:"titles_count"`
}

type publicUserTitleResponse struct {
	GrantID   string `json:"grant_id"`
	TitleID   string `json:"title_id"`
	Name      string `json:"name"`
	ScopeType string `json:"scope_type"`
	ScopeID   string `json:"scope_id"`
}

type publicUserStatsResponse struct {
	PostCount      int `json:"post_count"`
	CommentCount   int `json:"comment_count"`
	FollowerCount  int `json:"follower_count"`
	FollowingCount int `json:"following_count"`
}

type getPublicUserResponse struct {
	User publicUserResponse `json:"user"`
}

type listFollowedUsersResponse struct {
	Users      []publicUserResponse `json:"users"`
	Limit      int                  `json:"limit"`
	Offset     int                  `json:"offset"`
	NextOffset int                  `json:"next_offset"`
	HasMore    bool                 `json:"has_more"`
}

type updateProfileRequest struct {
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	BannerURL   *string `json:"banner_url"`
	Headline    *string `json:"headline"`
	Bio         *string `json:"bio"`
}

type updateProfileResponse struct {
	User publicUserResponse `json:"user"`
}

func NewHandler(currentUser CurrentUserUseCase, publicUsers ...PublicUserUseCase) *Handler {
	var publicUserUC PublicUserUseCase
	if len(publicUsers) > 0 {
		publicUserUC = publicUsers[0]
	}
	return &Handler{
		currentUser: currentUser,
		publicUsers: publicUserUC,
	}
}

func (h *Handler) SetProfileUpdater(profileUpdater ProfileUpdateUseCase) {
	h.profileUpdater = profileUpdater
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/me", handler.Me)
	group.PATCH("/me/profile", handler.UpdateProfile)
	group.GET("/me/followed-users", handler.ListFollowedUsers)
	group.POST("/users/:username/follow", handler.FollowUser)
	group.DELETE("/users/:username/follow", handler.DeleteUserFollow)
}

func RegisterPublicRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/users/:username", handler.GetPublicUser)
}

func (h *Handler) Me(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	result, err := h.currentUser.GetCurrentUser(c.Request.Context(), userusecase.CurrentUserInput{
		UserID: userID,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, currentUserResponse{
		ID:              result.User.ID,
		Username:        result.User.Username,
		Status:          result.User.Status,
		IsPlatformStaff: result.User.IsPlatformStaff,
		PlatformRole:    result.User.PlatformRole,
		CreatedAt:       result.User.CreatedAt,
	})
}

func (h *Handler) GetPublicUser(c *gin.Context) {
	if h.publicUsers == nil {
		_ = c.Error(apperr.New(apperr.CodeInternal, "public user usecase is not configured"))
		c.Abort()
		return
	}

	result, err := h.publicUsers.GetPublicUser(c.Request.Context(), userusecase.GetPublicUserInput{
		Username: c.Param("username"),
		ViewerID: userIDFromContext(c),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, getPublicUserResponse{
		User: toPublicUserResponse(result.User),
	})
}

func (h *Handler) ListFollowedUsers(c *gin.Context) {
	if h.publicUsers == nil {
		_ = c.Error(apperr.New(apperr.CodeInternal, "public user usecase is not configured"))
		c.Abort()
		return
	}
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

	result, err := h.publicUsers.ListFollowedUsers(c.Request.Context(), userusecase.ListFollowedUsersInput{UserID: userID, Limit: limit, Offset: offset})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listFollowedUsersResponse{
		Users:      make([]publicUserResponse, 0, len(result.Users)),
		Limit:      result.Limit,
		Offset:     result.Offset,
		NextOffset: result.NextOffset,
		HasMore:    result.HasMore,
	}
	for _, user := range result.Users {
		response.Users = append(response.Users, toPublicUserResponse(user))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) FollowUser(c *gin.Context) {
	if h.publicUsers == nil {
		_ = c.Error(apperr.New(apperr.CodeInternal, "public user usecase is not configured"))
		c.Abort()
		return
	}
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	if _, err := h.publicUsers.FollowUser(c.Request.Context(), userusecase.FollowUserInput{Username: c.Param("username"), UserID: userID}); err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteUserFollow(c *gin.Context) {
	if h.publicUsers == nil {
		_ = c.Error(apperr.New(apperr.CodeInternal, "public user usecase is not configured"))
		c.Abort()
		return
	}
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	if _, err := h.publicUsers.DeleteUserFollow(c.Request.Context(), userusecase.DeleteUserFollowInput{Username: c.Param("username"), UserID: userID}); err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	if h.profileUpdater == nil {
		_ = c.Error(apperr.New(apperr.CodeInternal, "profile update usecase is not configured"))
		c.Abort()
		return
	}

	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid profile update request"))
		c.Abort()
		return
	}

	result, err := h.profileUpdater.UpdateProfile(c.Request.Context(), userusecase.UpdateProfileInput{
		UserID:      userID,
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
		BannerURL:   req.BannerURL,
		Headline:    req.Headline,
		Bio:         req.Bio,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, updateProfileResponse{
		User: toPublicUserResponse(result.User),
	})
}

func toPublicUserResponse(user userusecase.PublicUser) publicUserResponse {
	return publicUserResponse{
		ID:              user.ID,
		Username:        user.Username,
		DisplayName:     user.DisplayName,
		AvatarURL:       user.AvatarURL,
		BannerURL:       user.BannerURL,
		Headline:        user.Headline,
		Bio:             user.Bio,
		Badges:          user.Badges,
		Roles:           user.Roles,
		Status:          user.Status,
		IsPlatformStaff: user.IsPlatformStaff,
		PlatformRole:    user.PlatformRole,
		Stats: publicUserStatsResponse{
			PostCount:      user.Stats.PostCount,
			CommentCount:   user.Stats.CommentCount,
			FollowerCount:  user.Stats.FollowerCount,
			FollowingCount: user.Stats.FollowingCount,
		},
		Progression: toPublicUserProgressionResponse(user.Progression),
		DMCapability: publicUserDMCapabilityResponse{
			CanStart:             user.DMCapability.CanStart,
			RequiresRequest:      user.DMCapability.RequiresRequest,
			Reason:               user.DMCapability.Reason,
			DirectConversationID: user.DMCapability.DirectConversationID,
			ViewerRelation:       user.DMCapability.ViewerRelation,
		},
		ViewerIsFollowing: user.ViewerIsFollowing,
		CreatedAt:         user.CreatedAt,
	}
}

func toPublicUserProgressionResponse(progression userusecase.PublicUserProgression) publicUserProgressionResponse {
	return publicUserProgressionResponse{
		Level:          progression.Level,
		LevelName:      progression.LevelName,
		XPTotal:        progression.XPTotal,
		CurrentLevelXP: progression.CurrentLevelXP,
		NextLevelXP:    progression.NextLevelXP,
		LevelProgress:  progression.LevelProgress,
		ActiveTitle:    toPublicUserTitleResponse(progression.ActiveTitle),
		TitlesCount:    progression.TitlesCount,
	}
}

func userIDFromContext(c *gin.Context) userdomain.UserID {
	userID, _ := authcontext.CurrentUserID(c.Request.Context())
	return userID
}

func parseOptionalIntQuery(c *gin.Context, key string) (int, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, apperr.New(apperr.CodeInvalidArgument, key+" must be an integer")
	}
	return value, nil
}

func toPublicUserTitleResponse(title *progressionusecase.TitleSummary) *publicUserTitleResponse {
	if title == nil {
		return nil
	}
	return &publicUserTitleResponse{
		GrantID:   title.GrantID,
		TitleID:   title.TitleID,
		Name:      title.Name,
		ScopeType: title.ScopeType,
		ScopeID:   title.ScopeID,
	}
}

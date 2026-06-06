package userhttp

import (
	"context"
	"net/http"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/user/userusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	currentUser CurrentUserUseCase
	publicUsers PublicUserUseCase
}

type CurrentUserUseCase interface {
	GetCurrentUser(ctx context.Context, input userusecase.CurrentUserInput) (userusecase.CurrentUserResult, error)
}

type PublicUserUseCase interface {
	GetPublicUser(ctx context.Context, input userusecase.GetPublicUserInput) (userusecase.GetPublicUserResult, error)
}

type currentUserResponse struct {
	ID              string    `json:"id"`
	Username        string    `json:"username"`
	Status          string    `json:"status"`
	IsPlatformStaff bool      `json:"is_platform_staff"`
	CreatedAt       time.Time `json:"created_at"`
}

type publicUserResponse struct {
	ID          string                  `json:"id"`
	Username    string                  `json:"username"`
	DisplayName string                  `json:"display_name"`
	AvatarURL   string                  `json:"avatar_url"`
	Headline    string                  `json:"headline"`
	Bio         string                  `json:"bio"`
	Badges      []string                `json:"badges"`
	Roles       []string                `json:"roles"`
	Status      string                  `json:"status"`
	Stats       publicUserStatsResponse `json:"stats"`
	CreatedAt   time.Time               `json:"created_at"`
}

type publicUserStatsResponse struct {
	PostCount    int `json:"post_count"`
	CommentCount int `json:"comment_count"`
}

type getPublicUserResponse struct {
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

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/me", handler.Me)
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

func toPublicUserResponse(user userusecase.PublicUser) publicUserResponse {
	return publicUserResponse{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		Headline:    user.Headline,
		Bio:         user.Bio,
		Badges:      user.Badges,
		Roles:       user.Roles,
		Status:      user.Status,
		Stats: publicUserStatsResponse{
			PostCount:    user.Stats.PostCount,
			CommentCount: user.Stats.CommentCount,
		},
		CreatedAt: user.CreatedAt,
	}
}

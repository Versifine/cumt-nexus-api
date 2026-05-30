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
}

type CurrentUserUseCase interface {
	GetCurrentUser(ctx context.Context, input userusecase.CurrentUserInput) (userusecase.CurrentUserResult, error)
}

type currentUserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func NewHandler(currentUser CurrentUserUseCase) *Handler {
	return &Handler{
		currentUser: currentUser,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/me", handler.Me)
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
		ID:        result.User.ID,
		Username:  result.User.Username,
		Status:    result.User.Status,
		CreatedAt: result.User.CreatedAt,
	})
}

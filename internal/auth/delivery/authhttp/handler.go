package authhttp

import (
	"context"
	"net/http"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	register RegisterUseCase
}

func NewHandler(register RegisterUseCase) *Handler {
	return &Handler{
		register: register,
	}
}

type RegisterUseCase interface {
	Register(ctx context.Context, input authusecase.RegisterInput) (authusecase.RegisterResult, error)
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid register request"))
		c.Abort()
		return
	}

	result, err := h.register.Register(c.Request.Context(), authusecase.RegisterInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusCreated, registerResponse{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresIn:   result.ExpiresIn,
		User: userResponse{
			ID:        result.User.ID,
			Username:  result.User.Username,
			Status:    result.User.Status,
			CreatedAt: result.User.CreatedAt,
		},
	})
}

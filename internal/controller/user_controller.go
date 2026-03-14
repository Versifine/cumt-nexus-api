package controller

import (
	"context"
	"cumt-nexus-api/internal/service"
	"cumt-nexus-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserSvc interface {
	GetMe(ctx context.Context, userID uint64) (*service.ProfileDTO, error)
}

type UserController struct {
	UserSvc UserSvc
}

func NewUserController(userSvc UserSvc) *UserController {
	return &UserController{
		UserSvc: userSvc,
	}
}

func (uc *UserController) GetMe(c *gin.Context) {
	ctx := c.Request.Context()
	value, exists := c.Get("user_id")
	if !exists {
		code, msg := service.MapError(service.ErrUnauthorized)
		response.Fail(c, code, msg)
		return
	}
	userID, ok := value.(uint64)
	if !ok {
		code, msg := service.MapError(service.ErrUnauthorized)
		response.Fail(c, code, msg)
		return
	}
	//service层
	out, err := uc.UserSvc.GetMe(ctx, userID)
	if err != nil {
		code, msg := service.MapError(err)
		response.Fail(c, code, msg)
		return
	}

	//响应
	response.Success(c, out)
}

package controller

import (
	"context"
	"cumt-nexus-api/internal/service"
	"cumt-nexus-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserSvc interface {
	GetProfile(ctx context.Context, userID uint64) (string, error)
}

type UserController struct {
	UserSvc UserSvc
}

func NewUserController(userSvc UserSvc) *UserController {
	return &UserController{
		UserSvc: userSvc,
	}
}

func (uc *UserController) GetProfile(c *gin.Context) {
	ctx := c.Request.Context()
	userID := uint64(1) // 这里暂时写死，后续可以从 JWT 或者请求参数中获取

	//service层
	out, err := uc.UserSvc.GetProfile(ctx, userID)
	if err != nil {
		code, msg := service.MapError(err)
		response.Fail(c, code, msg)
		return
	}

	//响应
	response.Success(c, out)
}

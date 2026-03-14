package controller

import (
	"context"
	"cumt-nexus-api/internal/service"
	"cumt-nexus-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	AuthSvc AuthSvc
}

type AuthSvc interface {
	Login(ctx context.Context, username, password string) (*service.LoginResponse, error)
	Register(ctx context.Context, username, password, nickname string) (*service.RegisterResponse, error)
}

func NewAuthController(authSvc AuthSvc) *AuthController {
	return &AuthController{
		AuthSvc: authSvc,
	}
}

func (ac *AuthController) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		code, msg := service.MapError(service.ErrParamInvalid)
		response.Fail(c, code, msg)
		return
	}
	ctx := c.Request.Context()
	out, err := ac.AuthSvc.Login(ctx, req.Username, req.Password)
	if err != nil {
		code, msg := service.MapError(err)
		response.Fail(c, code, msg)
		return
	}
	response.Success(c, out)
}

func (ac *AuthController) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		code, msg := service.MapError(service.ErrParamInvalid)
		response.Fail(c, code, msg)
		return
	}
	ctx := c.Request.Context()
	out, err := ac.AuthSvc.Register(ctx, req.Username, req.Password, req.Nickname)
	if err != nil {
		code, msg := service.MapError(err)
		response.Fail(c, code, msg)
		return
	}
	response.Success(c, out)
}

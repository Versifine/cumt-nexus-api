package controller

import (
	"cumt-nexus-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserController struct {
}

func NewUserController() *UserController {
	return &UserController{}
}

func (uc *UserController) GetProfile(c *gin.Context) {

	//service层

	//响应
	response.Success(c, map[string]string{"info": "此为测试暂时使用信息"})
}

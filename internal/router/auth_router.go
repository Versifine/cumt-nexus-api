package router

import (
	"cumt-nexus-api/internal/controller"

	"github.com/gin-gonic/gin"
)

func InitAuthRouter(rg *gin.RouterGroup, authCtrl *controller.AuthController) {

	authRouter := rg.Group("/auth")
	{
		authRouter.POST("/login", authCtrl.Login)
		authRouter.POST("/register", authCtrl.Register)
	}
}

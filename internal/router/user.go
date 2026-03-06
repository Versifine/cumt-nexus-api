package router

import (
	"cumt-nexus-api/internal/controller"

	"github.com/gin-gonic/gin"
)

func InitUserRouter(rg *gin.RouterGroup) {
	userCtrl := controller.NewUserController()

	userRouter := rg.Group("/user")
	{
		userRouter.GET("/profile", userCtrl.GetProfile)
	}
}

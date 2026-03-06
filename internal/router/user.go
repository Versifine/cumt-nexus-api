package router

import (
	"cumt-nexus-api/internal/controller"

	"github.com/gin-gonic/gin"
)

func InitUserRouter(rg *gin.RouterGroup, userCtrl *controller.UserController) {

	userRouter := rg.Group("/user")
	{
		userRouter.GET("/profile", userCtrl.GetProfile)
	}
}

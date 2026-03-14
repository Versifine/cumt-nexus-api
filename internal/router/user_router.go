package router

import (
	"cumt-nexus-api/internal/controller"
	"cumt-nexus-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func InitUserRouter(rg *gin.RouterGroup, userCtrl *controller.UserController) {

	userRouter := rg.Group("/users")
	{
		userRouter.GET("/me", middleware.Auth(), userCtrl.GetMe)

	}
}

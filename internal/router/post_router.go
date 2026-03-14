package router

import (
	"cumt-nexus-api/internal/controller"
	"cumt-nexus-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func InitPostRouter(rg *gin.RouterGroup, postCtrl *controller.PostController) {
	postRouter := rg.Group("/posts")
	{
		postRouter.POST("", middleware.Auth(), postCtrl.CreatePost)
		postRouter.GET("", postCtrl.ListPosts)
		postRouter.GET("/:id", postCtrl.GetPostDetail)
	}
}

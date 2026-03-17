package router

import (
	"cumt-nexus-api/internal/controller"
	"cumt-nexus-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func InitCommentRouter(rg *gin.RouterGroup, commentCtrl *controller.CommentController) {

	{
		commentRouter := rg.Group("/posts")
		// Define comment-related routes here, e.g.:
		commentRouter.POST("/:post_id/comments", middleware.Auth(), commentCtrl.CreateComment)

		commentRouter.GET("/:post_id/comments", commentCtrl.ListComment)
	}
}

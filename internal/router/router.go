package router

import (
	"cumt-nexus-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.GinLogger())
	r.Use(middleware.Recovery())
	r.Use(middleware.Cors())
	v1 := r.Group("/api/v1")
	{
		InitUserRouter(v1)
	}
	return r
}

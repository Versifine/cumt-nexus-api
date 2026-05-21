package httpserver

import "github.com/gin-gonic/gin"

func NewRouter() *gin.Engine {
	router := gin.New()

	router.Use(RecoveryMiddleware())
	router.Use(ErrorMiddleware())

	RegisterHealthRoutes(router)

	return router
}

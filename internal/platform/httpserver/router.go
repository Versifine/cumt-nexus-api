package httpserver

import "github.com/gin-gonic/gin"

func NewRouter() *gin.Engine {
	router := gin.New()

	RegisterHealthRoutes(router)

	return router
}

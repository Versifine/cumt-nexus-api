package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterHealthRoutes(router gin.IRouter) {
	router.GET("/healthz", healthzHandler)
}

func healthzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

package httpserver

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

func NewRouter(logger *slog.Logger) *gin.Engine {
	router := gin.New()

	router.Use(RecoveryMiddleware())
	router.Use(RequestIDMiddleware())
	router.Use(RequestLoggerMiddleware(logger))
	router.Use(ErrorMiddleware())

	RegisterHealthRoutes(router)

	return router
}

package httpserver

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

func NewRouter(log *slog.Logger) *gin.Engine {
	router := gin.New()

	router.Use(RecoveryMiddleware(log))
	router.Use(RequestIDMiddleware())
	router.Use(RequestLoggerMiddleware(log))
	router.Use(ErrorMiddleware())

	RegisterHealthRoutes(router)

	return router
}

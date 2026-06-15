package httpserver

import (
	"log/slog"

	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/gin-gonic/gin"
)

func NewRouter(log *slog.Logger, cfg config.HTTPConfig) *gin.Engine {
	router := gin.New()

	router.Use(RecoveryMiddleware(log))
	router.Use(RequestIDMiddleware())
	router.Use(RequestLoggerMiddleware(log))
	router.Use(CORSMiddleware(cfg.CORSAllowedOrigins))
	router.Use(ErrorMiddleware(log))

	RegisterHealthRoutes(router)

	return router
}

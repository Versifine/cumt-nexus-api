package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	loggerpkg "github.com/Versifine/cumt-nexus-api/internal/platform/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	log := loggerpkg.New(cfg.Log).With(
		"app", cfg.App.Name,
		"env", cfg.App.Env,
	)

	if err := run(cfg, log); err != nil {
		log.Error("service exited", "error", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, log *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	pool, err := openDB(ctx, cfg.Postgres)
	if err != nil {
		return err
	}
	defer db.Close(pool)
	log.Info("database connected")

	router := httpserver.NewRouter(log)
	server := httpserver.NewServer(cfg.HTTP, router)

	return serveHTTP(server, cfg.HTTP, log)
}

func openDB(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	pool, err := db.Open(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(ctx, pool); err != nil {
		db.Close(pool)
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

func serveHTTP(server *http.Server, cfg config.HTTPConfig, log *slog.Logger) error {
	serverErr := make(chan error, 1)

	go func() {
		log.Info("http server listening", "addr", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignal)

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	case sig := <-shutdownSignal:
		log.Info("shutdown signal received", "signal", sig.String())
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server:%w", err)
	}
	if err := <-serverErr; err != nil {
		return fmt.Errorf("server close: %w", err)
	}
	return nil
}

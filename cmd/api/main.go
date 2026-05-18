package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	pool, err := db.Open(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close(pool)

	if err := db.Ping(ctx, pool); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	fmt.Println("Successfully connected to the database!")

	router := httpserver.NewRouter()
	server := httpserver.NewServer(cfg.HTTP, router)

	serverErr := make(chan error, 1)

	go func() {
		fmt.Printf("HTTP server listen on %s\n", cfg.HTTP.Addr)
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
		fmt.Printf("received signal: %s\n", sig)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server:%w", err)
	}
	if err := <-serverErr; err != nil {
		return fmt.Errorf("server close: %w", err)
	}

	return nil
}

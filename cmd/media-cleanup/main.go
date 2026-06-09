package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/media/mediarepository"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediausecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/storage"
)

func main() {
	unboundTTL := flag.Duration("unbound-ttl", 24*time.Hour, "TTL for ready image attachments that are not bound to a post or comment")
	failedTTL := flag.Duration("failed-ttl", 24*time.Hour, "TTL for failed or blocked image attachments that are not bound to a post or comment")
	limit := flag.Int("limit", 100, "maximum number of attachments to clean in one run")
	timeout := flag.Duration("timeout", time.Minute, "maximum cleanup runtime")
	dryRun := flag.Bool("dry-run", false, "list eligible attachments without deleting metadata or objects")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		exitWithError("load config", err)
	}

	startupCtx, startupCancel := context.WithTimeout(context.Background(), cfg.App.StartupTimeout)
	defer startupCancel()

	pool, err := db.Open(startupCtx, cfg.Postgres)
	if err != nil {
		exitWithError("open database", err)
	}
	defer db.Close(pool)
	if err := db.Ping(startupCtx, pool); err != nil {
		exitWithError("ping database", err)
	}

	var objectStorage mediausecase.ObjectStorage
	if !*dryRun {
		objectStorage, err = storage.NewObjectStorage(startupCtx, cfg.Storage)
		if err != nil {
			exitWithError("create object storage", err)
		}
	}

	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), *timeout)
	defer cleanupCancel()

	mediaRepo := mediarepository.NewPostgresMediaRepository(pool)
	mediaUC := mediausecase.NewUseCase(mediaRepo, objectStorage, mediausecase.UploadLimits{
		ImageMaxBytes: cfg.Upload.ImageMaxBytes,
	}, time.Now)
	result, err := mediaUC.CleanupExpiredAttachments(cleanupCtx, mediausecase.CleanupExpiredAttachmentsInput{
		UnboundTTL: *unboundTTL,
		FailedTTL:  *failedTTL,
		Limit:      *limit,
		DryRun:     *dryRun,
	})
	if err != nil {
		exitWithError("cleanup media attachments", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		exitWithError("write result", err)
	}
}

func exitWithError(operation string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", operation, err)
	os.Exit(1)
}

package storage

import (
	"context"
	"fmt"

	"github.com/Versifine/cumt-nexus-api/internal/media/mediausecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
)

func NewObjectStorage(ctx context.Context, cfg config.ObjectStorageConfig) (mediausecase.ObjectStorage, error) {
	switch cfg.Provider {
	case "local":
		return NewLocalObjectStorage(cfg), nil
	case "r2":
		client, err := NewR2ObjectStorage(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("create r2 object storage: %w", err)
		}
		return client, nil
	default:
		return nil, fmt.Errorf("unsupported object storage provider %q", cfg.Provider)
	}
}

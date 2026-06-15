package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Versifine/cumt-nexus-api/internal/media/mediausecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
)

var _ mediausecase.ObjectStorage = (*LocalObjectStorage)(nil)

type LocalObjectStorage struct {
	root          string
	publicBaseURL string
	bucket        string
}

func NewLocalObjectStorage(cfg config.ObjectStorageConfig) *LocalObjectStorage {
	return &LocalObjectStorage{
		root:          cfg.LocalRoot,
		publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
		bucket:        "local",
	}
}

func (storage *LocalObjectStorage) PutObject(ctx context.Context, input mediausecase.PutObjectInput) (mediausecase.PutObjectResult, error) {
	if err := validateObjectKey(input.ObjectKey); err != nil {
		return mediausecase.PutObjectResult{}, err
	}
	target, err := storage.targetPath(input.ObjectKey)
	if err != nil {
		return mediausecase.PutObjectResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return mediausecase.PutObjectResult{}, fmt.Errorf("create local object directory: %w", err)
	}

	file, err := os.Create(target)
	if err != nil {
		return mediausecase.PutObjectResult{}, fmt.Errorf("create local object: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, input.Body); err != nil {
		return mediausecase.PutObjectResult{}, fmt.Errorf("write local object: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return mediausecase.PutObjectResult{}, err
	}

	return mediausecase.PutObjectResult{
		StorageProvider: "local",
		Bucket:          storage.bucket,
		ObjectKey:       input.ObjectKey,
		PublicURL:       storage.publicURL(input.ObjectKey),
	}, nil
}

func (storage *LocalObjectStorage) DeleteObject(ctx context.Context, objectKey string) error {
	if err := validateObjectKey(objectKey); err != nil {
		return err
	}
	target, err := storage.targetPath(objectKey)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete local object: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (storage *LocalObjectStorage) targetPath(objectKey string) (string, error) {
	root, err := filepath.Abs(storage.root)
	if err != nil {
		return "", fmt.Errorf("resolve local storage root: %w", err)
	}
	target, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(objectKey)))
	if err != nil {
		return "", fmt.Errorf("resolve local object path: %w", err)
	}
	if target != root && !strings.HasPrefix(target, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("local object path escapes storage root")
	}
	return target, nil
}

func (storage *LocalObjectStorage) publicURL(objectKey string) string {
	return storage.publicBaseURL + "/" + strings.TrimLeft(objectKey, "/")
}

func validateObjectKey(objectKey string) error {
	if strings.TrimSpace(objectKey) == "" {
		return fmt.Errorf("object key is required")
	}
	if len([]rune(strings.TrimSpace(objectKey))) > 1024 {
		return fmt.Errorf("object key is too long")
	}
	if strings.HasPrefix(objectKey, "/") || strings.Contains(objectKey, "\\") {
		return fmt.Errorf("object key is invalid")
	}
	for _, part := range strings.Split(objectKey, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("object key is invalid")
		}
	}
	return nil
}

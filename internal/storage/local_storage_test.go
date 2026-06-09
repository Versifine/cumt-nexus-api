package storage

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Versifine/cumt-nexus-api/internal/media/mediausecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
)

func TestLocalObjectStoragePutObject(t *testing.T) {
	root := t.TempDir()
	storage := NewLocalObjectStorage(config.ObjectStorageConfig{
		LocalRoot:     root,
		PublicBaseURL: "http://localhost:8080/uploads",
	})

	result, err := storage.PutObject(context.Background(), mediausecase.PutObjectInput{
		ObjectKey:   "images/2026/06/image.png",
		ContentType: "image/png",
		SizeBytes:   3,
		Body:        bytes.NewReader([]byte("png")),
	})
	if err != nil {
		t.Fatalf("PutObject returned error: %v", err)
	}
	if result.StorageProvider != "local" || result.Bucket != "local" || result.PublicURL != "http://localhost:8080/uploads/images/2026/06/image.png" {
		t.Fatalf("unexpected result: %#v", result)
	}
	content, err := os.ReadFile(filepath.Join(root, "images", "2026", "06", "image.png"))
	if err != nil {
		t.Fatalf("read local object: %v", err)
	}
	if string(content) != "png" {
		t.Fatalf("unexpected content %q", string(content))
	}
}

func TestLocalObjectStorageRejectsUnsafeObjectKey(t *testing.T) {
	storage := NewLocalObjectStorage(config.ObjectStorageConfig{
		LocalRoot:     t.TempDir(),
		PublicBaseURL: "http://localhost:8080/uploads",
	})

	_, err := storage.PutObject(context.Background(), mediausecase.PutObjectInput{
		ObjectKey: "../escape.png",
		Body:      bytes.NewReader([]byte("png")),
	})
	if err == nil {
		t.Fatal("expected unsafe object key error")
	}
}

func TestLocalObjectStorageDeleteObject(t *testing.T) {
	root := t.TempDir()
	storage := NewLocalObjectStorage(config.ObjectStorageConfig{
		LocalRoot:     root,
		PublicBaseURL: "http://localhost:8080/uploads",
	})
	objectKey := "images/2026/06/image.png"
	if _, err := storage.PutObject(context.Background(), mediausecase.PutObjectInput{
		ObjectKey: objectKey,
		Body:      bytes.NewReader([]byte("png")),
	}); err != nil {
		t.Fatalf("PutObject returned error: %v", err)
	}

	if err := storage.DeleteObject(context.Background(), objectKey); err != nil {
		t.Fatalf("DeleteObject returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "images", "2026", "06", "image.png")); !os.IsNotExist(err) {
		t.Fatalf("expected local object to be deleted, stat err=%v", err)
	}
	if err := storage.DeleteObject(context.Background(), objectKey); err != nil {
		t.Fatalf("DeleteObject should ignore missing object, got %v", err)
	}
}

func TestLocalObjectStorageDeleteObjectRejectsUnsafeObjectKey(t *testing.T) {
	storage := NewLocalObjectStorage(config.ObjectStorageConfig{
		LocalRoot:     t.TempDir(),
		PublicBaseURL: "http://localhost:8080/uploads",
	})

	if err := storage.DeleteObject(context.Background(), "../escape.png"); err == nil {
		t.Fatal("expected unsafe object key error")
	}
}

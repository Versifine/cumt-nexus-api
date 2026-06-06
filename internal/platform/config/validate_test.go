package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidateAcceptsLocalObjectStorage(t *testing.T) {
	cfg := validConfig()

	if err := validate(&cfg); err != nil {
		t.Fatalf("validate returned error: %v", err)
	}
}

func TestValidateAcceptsR2ObjectStorage(t *testing.T) {
	cfg := validConfig()
	cfg.Storage = ObjectStorageConfig{
		Provider:        "r2",
		Endpoint:        "https://example.r2.cloudflarestorage.com",
		Region:          "auto",
		Bucket:          "cumt-nexus-media",
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
		PublicBaseURL:   "https://media.example.com",
		ForcePathStyle:  true,
	}

	if err := validate(&cfg); err != nil {
		t.Fatalf("validate returned error: %v", err)
	}
}

func TestValidateRejectsInvalidObjectStorage(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		message string
	}{
		{
			name: "unknown provider",
			mutate: func(cfg *Config) {
				cfg.Storage.Provider = "s3"
			},
			message: "OBJECT_STORAGE_PROVIDER",
		},
		{
			name: "missing r2 bucket",
			mutate: func(cfg *Config) {
				cfg.Storage = ObjectStorageConfig{
					Provider:        "r2",
					Endpoint:        "https://example.r2.cloudflarestorage.com",
					Region:          "auto",
					AccessKeyID:     "access-key",
					SecretAccessKey: "secret-key",
					PublicBaseURL:   "https://media.example.com",
				}
			},
			message: "OBJECT_STORAGE_BUCKET",
		},
		{
			name: "r2 public base url uses s3 endpoint",
			mutate: func(cfg *Config) {
				cfg.Storage = ObjectStorageConfig{
					Provider:        "r2",
					Endpoint:        "https://example.r2.cloudflarestorage.com",
					Region:          "auto",
					Bucket:          "cumt-nexus-media",
					AccessKeyID:     "access-key",
					SecretAccessKey: "secret-key",
					PublicBaseURL:   "https://example.r2.cloudflarestorage.com/cumt-nexus-media",
					ForcePathStyle:  true,
				}
			},
			message: "OBJECT_STORAGE_PUBLIC_BASE_URL",
		},
		{
			name: "empty local root",
			mutate: func(cfg *Config) {
				cfg.Storage.LocalRoot = ""
			},
			message: "OBJECT_STORAGE_LOCAL_ROOT",
		},
		{
			name: "invalid upload max bytes",
			mutate: func(cfg *Config) {
				cfg.Upload.ImageMaxBytes = 0
			},
			message: "UPLOAD_IMAGE_MAX_BYTES",
		},
		{
			name: "invalid post image count",
			mutate: func(cfg *Config) {
				cfg.Upload.ImageMaxCountPerPost = 0
			},
			message: "UPLOAD_IMAGE_MAX_COUNT_PER_POST",
		},
		{
			name: "invalid comment image count",
			mutate: func(cfg *Config) {
				cfg.Upload.ImageMaxCountPerComment = 0
			},
			message: "UPLOAD_IMAGE_MAX_COUNT_PER_COMMENT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)

			err := validate(&cfg)
			if err == nil {
				t.Fatal("expected validate error")
			}
			if !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("expected error containing %q, got %v", tt.message, err)
			}
		})
	}
}

func validConfig() Config {
	return Config{
		App: AppConfig{
			Name:           "cumt-nexus-api",
			Env:            "local",
			StartupTimeout: 10 * time.Second,
		},
		Postgres: PostgresConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "postgres",
			Password:        "postgres",
			Database:        "cumt_nexus",
			SSLMode:         "disable",
			MaxConns:        25,
			MaxConnLifetime: 5 * time.Minute,
			MaxConnIdleTime: 2 * time.Minute,
		},
		HTTP: HTTPConfig{
			Addr:               ":8080",
			ReadTimeout:        5 * time.Second,
			WriteTimeout:       10 * time.Second,
			ShutdownTimeout:    15 * time.Second,
			CORSAllowedOrigins: []string{"http://localhost:5173"},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
		Auth: AuthConfig{
			TokenSecret:    "secret",
			AccessTokenTTL: 24 * time.Hour,
		},
		Storage: ObjectStorageConfig{
			Provider:       "local",
			PublicBaseURL:  "http://localhost:8080/uploads",
			ForcePathStyle: true,
			LocalRoot:      "var/uploads",
		},
		Upload: UploadConfig{
			ImageMaxBytes:           5 * 1024 * 1024,
			ImageMaxCountPerPost:    9,
			ImageMaxCountPerComment: 1,
		},
	}
}

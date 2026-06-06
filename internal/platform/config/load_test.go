package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

var configEnvKeys = []string{
	"APP_NAME",
	"APP_ENV",
	"APP_STARTUP_TIMEOUT",
	"POSTGRES_HOST",
	"POSTGRES_PORT",
	"POSTGRES_USER",
	"POSTGRES_PASSWORD",
	"POSTGRES_DATABASE",
	"POSTGRES_SSL_MODE",
	"POSTGRES_MAX_CONNS",
	"POSTGRES_MAX_CONN_LIFETIME",
	"POSTGRES_MAX_CONN_IDLE_TIME",
	"HTTP_ADDR",
	"HTTP_READ_TIMEOUT",
	"HTTP_WRITE_TIMEOUT",
	"HTTP_SHUTDOWN_TIMEOUT",
	"HTTP_CORS_ALLOWED_ORIGINS",
	"LOG_LEVEL",
	"LOG_FORMAT",
	"AUTH_TOKEN_SECRET",
	"AUTH_ACCESS_TOKEN_TTL",
	"OBJECT_STORAGE_PROVIDER",
	"OBJECT_STORAGE_ENDPOINT",
	"OBJECT_STORAGE_REGION",
	"OBJECT_STORAGE_BUCKET",
	"OBJECT_STORAGE_ACCESS_KEY_ID",
	"OBJECT_STORAGE_SECRET_ACCESS_KEY",
	"OBJECT_STORAGE_PUBLIC_BASE_URL",
	"OBJECT_STORAGE_FORCE_PATH_STYLE",
	"OBJECT_STORAGE_LOCAL_ROOT",
	"UPLOAD_IMAGE_MAX_BYTES",
	"UPLOAD_IMAGE_MAX_COUNT_PER_POST",
	"UPLOAD_IMAGE_MAX_COUNT_PER_COMMENT",
}

func TestLoadAppliesLocalDefaults(t *testing.T) {
	clearConfigEnv(t)
	t.Chdir(t.TempDir())
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.App.Env != "local" {
		t.Fatalf("expected APP_ENV default local, got %q", cfg.App.Env)
	}
	if cfg.App.StartupTimeout != 10*time.Second {
		t.Fatalf("expected startup timeout 10s, got %v", cfg.App.StartupTimeout)
	}
	if cfg.Postgres.Port != 5432 {
		t.Fatalf("expected postgres port 5432, got %d", cfg.Postgres.Port)
	}
	if cfg.Postgres.SSLMode != "disable" {
		t.Fatalf("expected postgres ssl mode disable, got %q", cfg.Postgres.SSLMode)
	}
	if cfg.Postgres.MaxConns != 25 {
		t.Fatalf("expected postgres max conns 25, got %d", cfg.Postgres.MaxConns)
	}
	if cfg.Postgres.MaxConnLifetime != 5*time.Minute {
		t.Fatalf("expected postgres max lifetime 5m, got %v", cfg.Postgres.MaxConnLifetime)
	}
	if cfg.Postgres.MaxConnIdleTime != 2*time.Minute {
		t.Fatalf("expected postgres idle time 2m, got %v", cfg.Postgres.MaxConnIdleTime)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("expected HTTP addr :8080, got %q", cfg.HTTP.Addr)
	}
	if cfg.HTTP.ReadTimeout != 5*time.Second {
		t.Fatalf("expected HTTP read timeout 5s, got %v", cfg.HTTP.ReadTimeout)
	}
	if cfg.HTTP.WriteTimeout != 10*time.Second {
		t.Fatalf("expected HTTP write timeout 10s, got %v", cfg.HTTP.WriteTimeout)
	}
	if cfg.HTTP.ShutdownTimeout != 15*time.Second {
		t.Fatalf("expected HTTP shutdown timeout 15s, got %v", cfg.HTTP.ShutdownTimeout)
	}
	if cfg.HTTP.CORSAllowedOrigins != nil {
		t.Fatalf("expected no CORS origins by default, got %#v", cfg.HTTP.CORSAllowedOrigins)
	}
	if cfg.Log.Level != "info" {
		t.Fatalf("expected log level info, got %q", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Fatalf("expected log format json, got %q", cfg.Log.Format)
	}
	if cfg.Auth.AccessTokenTTL != 24*time.Hour {
		t.Fatalf("expected access token ttl 24h, got %v", cfg.Auth.AccessTokenTTL)
	}
	if cfg.Storage.Provider != "local" {
		t.Fatalf("expected local storage provider, got %q", cfg.Storage.Provider)
	}
	if cfg.Storage.PublicBaseURL != "http://localhost:8080/uploads" {
		t.Fatalf("expected local public base URL fallback, got %q", cfg.Storage.PublicBaseURL)
	}
	if cfg.Storage.LocalRoot != "var/uploads" {
		t.Fatalf("expected local root var/uploads, got %q", cfg.Storage.LocalRoot)
	}
	if !cfg.Storage.ForcePathStyle {
		t.Fatal("expected force path style default true")
	}
	if cfg.Upload.ImageMaxBytes != 5*1024*1024 {
		t.Fatalf("expected image max bytes 5242880, got %d", cfg.Upload.ImageMaxBytes)
	}
	if cfg.Upload.ImageMaxCountPerPost != 9 {
		t.Fatalf("expected post image count 9, got %d", cfg.Upload.ImageMaxCountPerPost)
	}
	if cfg.Upload.ImageMaxCountPerComment != 1 {
		t.Fatalf("expected comment image count 1, got %d", cfg.Upload.ImageMaxCountPerComment)
	}
}

func TestLoadParsesR2Storage(t *testing.T) {
	clearConfigEnv(t)
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("OBJECT_STORAGE_PROVIDER", "r2")
	t.Setenv("OBJECT_STORAGE_ENDPOINT", "https://example.r2.cloudflarestorage.com")
	t.Setenv("OBJECT_STORAGE_REGION", "auto")
	t.Setenv("OBJECT_STORAGE_BUCKET", "cumt-nexus-media")
	t.Setenv("OBJECT_STORAGE_ACCESS_KEY_ID", "access-key")
	t.Setenv("OBJECT_STORAGE_SECRET_ACCESS_KEY", "secret-key")
	t.Setenv("OBJECT_STORAGE_PUBLIC_BASE_URL", "https://media.example.com")
	t.Setenv("OBJECT_STORAGE_FORCE_PATH_STYLE", "false")
	t.Setenv("HTTP_CORS_ALLOWED_ORIGINS", "http://localhost:5173, https://web.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Storage.Provider != "r2" {
		t.Fatalf("expected r2 provider, got %q", cfg.Storage.Provider)
	}
	if cfg.Storage.Endpoint != "https://example.r2.cloudflarestorage.com" {
		t.Fatalf("unexpected endpoint: %q", cfg.Storage.Endpoint)
	}
	if cfg.Storage.Bucket != "cumt-nexus-media" {
		t.Fatalf("unexpected bucket: %q", cfg.Storage.Bucket)
	}
	if cfg.Storage.AccessKeyID != "access-key" {
		t.Fatalf("unexpected access key id: %q", cfg.Storage.AccessKeyID)
	}
	if cfg.Storage.SecretAccessKey != "secret-key" {
		t.Fatalf("unexpected secret access key: %q", cfg.Storage.SecretAccessKey)
	}
	if cfg.Storage.PublicBaseURL != "https://media.example.com" {
		t.Fatalf("unexpected public base URL: %q", cfg.Storage.PublicBaseURL)
	}
	if cfg.Storage.ForcePathStyle {
		t.Fatal("expected force path style to parse false")
	}
	if got, want := cfg.HTTP.CORSAllowedOrigins, []string{"http://localhost:5173", "https://web.example.com"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected CORS origins: %#v", got)
	}
}

func TestLoadRejectsMissingR2Credentials(t *testing.T) {
	clearConfigEnv(t)
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("OBJECT_STORAGE_PROVIDER", "r2")
	t.Setenv("OBJECT_STORAGE_ENDPOINT", "https://example.r2.cloudflarestorage.com")
	t.Setenv("OBJECT_STORAGE_REGION", "auto")
	t.Setenv("OBJECT_STORAGE_PUBLIC_BASE_URL", "https://media.example.com")

	_, err := Load()
	if err == nil {
		t.Fatal("expected Load error")
	}
	for _, want := range []string{
		"OBJECT_STORAGE_BUCKET",
		"OBJECT_STORAGE_ACCESS_KEY_ID",
		"OBJECT_STORAGE_SECRET_ACCESS_KEY",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error containing %q, got %v", want, err)
		}
	}
}

func TestLoadRejectsR2PublicBaseURLUsingS3Endpoint(t *testing.T) {
	clearConfigEnv(t)
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("OBJECT_STORAGE_PROVIDER", "r2")
	t.Setenv("OBJECT_STORAGE_ENDPOINT", "https://example.r2.cloudflarestorage.com")
	t.Setenv("OBJECT_STORAGE_REGION", "auto")
	t.Setenv("OBJECT_STORAGE_BUCKET", "cumt-nexus-media")
	t.Setenv("OBJECT_STORAGE_ACCESS_KEY_ID", "access-key")
	t.Setenv("OBJECT_STORAGE_SECRET_ACCESS_KEY", "secret-key")
	t.Setenv("OBJECT_STORAGE_PUBLIC_BASE_URL", "https://example.r2.cloudflarestorage.com/cumt-nexus-media")

	_, err := Load()
	if err == nil {
		t.Fatal("expected Load error")
	}
	if !strings.Contains(err.Error(), "OBJECT_STORAGE_PUBLIC_BASE_URL") {
		t.Fatalf("expected error containing OBJECT_STORAGE_PUBLIC_BASE_URL, got %v", err)
	}
}

func TestLoadRejectsInvalidPrimitiveValues(t *testing.T) {
	clearConfigEnv(t)
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("POSTGRES_PORT", "not-a-port")
	t.Setenv("HTTP_READ_TIMEOUT", "not-a-duration")
	t.Setenv("OBJECT_STORAGE_FORCE_PATH_STYLE", "not-bool")

	_, err := Load()
	if err == nil {
		t.Fatal("expected Load error")
	}
	for _, want := range []string{
		"POSTGRES_PORT",
		"HTTP_READ_TIMEOUT",
		"OBJECT_STORAGE_FORCE_PATH_STYLE",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error containing %q, got %v", want, err)
		}
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	previous := make(map[string]string, len(configEnvKeys))
	present := make(map[string]bool, len(configEnvKeys))
	for _, key := range configEnvKeys {
		value, ok := os.LookupEnv(key)
		if ok {
			previous[key] = value
			present[key] = true
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}
	t.Cleanup(func() {
		for _, key := range configEnvKeys {
			if present[key] {
				_ = os.Setenv(key, previous[key])
			} else {
				_ = os.Unsetenv(key)
			}
		}
	})
}

func setRequiredEnv(t *testing.T) {
	t.Helper()

	t.Setenv("APP_NAME", "cumt-nexus-api")
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_USER", "postgres")
	t.Setenv("POSTGRES_PASSWORD", "postgres")
	t.Setenv("POSTGRES_DATABASE", "cumt_nexus")
	t.Setenv("AUTH_TOKEN_SECRET", "secret")
}

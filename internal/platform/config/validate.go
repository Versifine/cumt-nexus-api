package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func validate(cfg *Config) error {
	var errs []error

	switch cfg.App.Env {
	case "local", "dev", "test", "prod":
	default:
		errs = append(errs, fmt.Errorf("APP_ENV must be one of local/dev/test/prod"))
	}
	if cfg.App.StartupTimeout <= 0 {
		errs = append(errs, fmt.Errorf("APP_STARTUP_TIMEOUT must be > 0"))
	}
	switch cfg.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Errorf("LOG_LEVEL must be one of debug/info/warn/error"))
	}
	switch cfg.Log.Format {
	case "json", "text":
	default:
		errs = append(errs, fmt.Errorf("LOG_FORMAT must be one of json/text"))
	}
	switch cfg.Postgres.SSLMode {
	case "disable", "require", "verify-ca", "verify-full":
	default:
		errs = append(errs, fmt.Errorf("POSTGRES_SSL_MODE must be one of disable/require/verify-ca/verify-full"))
	}

	if cfg.HTTP.Addr == "" {
		errs = append(errs, fmt.Errorf("HTTP_ADDR cannot be empty"))
	}
	if cfg.HTTP.WriteTimeout <= 0 {
		errs = append(errs, fmt.Errorf("HTTP_WRITE_TIMEOUT must be > 0"))
	}
	if cfg.HTTP.ShutdownTimeout <= 0 {
		errs = append(errs, fmt.Errorf("HTTP_SHUTDOWN_TIMEOUT must be > 0"))
	}
	if cfg.HTTP.ReadTimeout <= 0 {
		errs = append(errs, fmt.Errorf("HTTP_READ_TIMEOUT must be > 0"))
	}
	for _, origin := range cfg.HTTP.CORSAllowedOrigins {
		if strings.TrimSpace(origin) == "" {
			errs = append(errs, fmt.Errorf("HTTP_CORS_ALLOWED_ORIGINS cannot contain empty origins"))
		}
	}
	if cfg.Postgres.MaxConns <= 0 {
		errs = append(errs, fmt.Errorf("POSTGRES_MAX_CONNS must be > 0"))
	}
	if cfg.Postgres.MaxConnIdleTime < 0 {
		errs = append(errs, fmt.Errorf("POSTGRES_MAX_CONN_IDLE_TIME must be >= 0"))
	}
	if cfg.Postgres.MaxConnLifetime < 0 {
		errs = append(errs, fmt.Errorf("POSTGRES_MAX_CONN_LIFETIME must be >= 0"))
	}
	if cfg.Postgres.Port <= 0 || cfg.Postgres.Port > 65535 {
		errs = append(errs, fmt.Errorf("POSTGRES_PORT must be in [1,65535]"))
	}
	if cfg.Auth.TokenSecret == "" {
		errs = append(errs, fmt.Errorf("AUTH_TOKEN_SECRET cannot be empty"))
	}
	if cfg.Auth.AccessTokenTTL <= 0 {
		errs = append(errs, fmt.Errorf("AUTH_ACCESS_TOKEN_TTL must be > 0"))
	}
	switch cfg.Storage.Provider {
	case "local":
		if strings.TrimSpace(cfg.Storage.LocalRoot) == "" {
			errs = append(errs, fmt.Errorf("OBJECT_STORAGE_LOCAL_ROOT cannot be empty for local storage"))
		}
		if strings.TrimSpace(cfg.Storage.PublicBaseURL) == "" {
			errs = append(errs, fmt.Errorf("OBJECT_STORAGE_PUBLIC_BASE_URL cannot be empty for local storage"))
		}
	case "r2":
		if strings.TrimSpace(cfg.Storage.Endpoint) == "" {
			errs = append(errs, fmt.Errorf("OBJECT_STORAGE_ENDPOINT is required for r2 storage"))
		}
		if strings.TrimSpace(cfg.Storage.Region) == "" {
			errs = append(errs, fmt.Errorf("OBJECT_STORAGE_REGION is required for r2 storage"))
		}
		if strings.TrimSpace(cfg.Storage.Bucket) == "" {
			errs = append(errs, fmt.Errorf("OBJECT_STORAGE_BUCKET is required for r2 storage"))
		}
		if strings.TrimSpace(cfg.Storage.AccessKeyID) == "" {
			errs = append(errs, fmt.Errorf("OBJECT_STORAGE_ACCESS_KEY_ID is required for r2 storage"))
		}
		if strings.TrimSpace(cfg.Storage.SecretAccessKey) == "" {
			errs = append(errs, fmt.Errorf("OBJECT_STORAGE_SECRET_ACCESS_KEY is required for r2 storage"))
		}
		if strings.TrimSpace(cfg.Storage.PublicBaseURL) == "" {
			errs = append(errs, fmt.Errorf("OBJECT_STORAGE_PUBLIC_BASE_URL is required for r2 storage"))
		} else if usesR2S3EndpointAsPublicBaseURL(cfg.Storage.PublicBaseURL) {
			errs = append(errs, fmt.Errorf("OBJECT_STORAGE_PUBLIC_BASE_URL must be an R2 public development URL or custom media domain, not the R2 S3 API endpoint"))
		}
	default:
		errs = append(errs, fmt.Errorf("OBJECT_STORAGE_PROVIDER must be one of local/r2"))
	}
	if cfg.Upload.ImageMaxBytes <= 0 {
		errs = append(errs, fmt.Errorf("UPLOAD_IMAGE_MAX_BYTES must be > 0"))
	}
	if cfg.Upload.ImageMaxCountPerPost <= 0 {
		errs = append(errs, fmt.Errorf("UPLOAD_IMAGE_MAX_COUNT_PER_POST must be > 0"))
	}
	if cfg.Upload.ImageMaxCountPerComment <= 0 {
		errs = append(errs, fmt.Errorf("UPLOAD_IMAGE_MAX_COUNT_PER_COMMENT must be > 0"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func usesR2S3EndpointAsPublicBaseURL(rawURL string) bool {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}

	host := strings.ToLower(parsedURL.Hostname())
	return host == "r2.cloudflarestorage.com" || strings.HasSuffix(host, ".r2.cloudflarestorage.com")
}

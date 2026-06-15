package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func Load() (*Config, error) {
	if err := loadDotEnvIfPresent(); err != nil {
		return nil, err
	}

	var cfg Config
	var errs []error
	cfg.App.Name = requiredString("APP_NAME", &errs)
	cfg.App.Env = stringDefault("APP_ENV", "local")
	cfg.App.StartupTimeout = durationDefault("APP_STARTUP_TIMEOUT", 10*time.Second, &errs)

	cfg.Postgres.Host = requiredString("POSTGRES_HOST", &errs)
	cfg.Postgres.Port = intDefault("POSTGRES_PORT", 5432, &errs)
	cfg.Postgres.User = requiredString("POSTGRES_USER", &errs)
	cfg.Postgres.Password = requiredString("POSTGRES_PASSWORD", &errs)
	cfg.Postgres.Database = requiredString("POSTGRES_DATABASE", &errs)
	cfg.Postgres.SSLMode = stringDefault("POSTGRES_SSL_MODE", "disable")
	cfg.Postgres.MaxConns = intDefault("POSTGRES_MAX_CONNS", 25, &errs)
	cfg.Postgres.MaxConnLifetime = durationDefault("POSTGRES_MAX_CONN_LIFETIME", 5*time.Minute, &errs)
	cfg.Postgres.MaxConnIdleTime = durationDefault("POSTGRES_MAX_CONN_IDLE_TIME", 2*time.Minute, &errs)

	cfg.HTTP.Addr = stringDefault("HTTP_ADDR", ":8080")
	cfg.HTTP.ReadTimeout = durationDefault("HTTP_READ_TIMEOUT", 5*time.Second, &errs)
	cfg.HTTP.WriteTimeout = durationDefault("HTTP_WRITE_TIMEOUT", 10*time.Second, &errs)
	cfg.HTTP.ShutdownTimeout = durationDefault("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second, &errs)
	cfg.HTTP.CORSAllowedOrigins = stringListDefault("HTTP_CORS_ALLOWED_ORIGINS", nil)

	cfg.Log.Level = stringDefault("LOG_LEVEL", "info")
	cfg.Log.Format = stringDefault("LOG_FORMAT", "json")

	cfg.Auth.TokenSecret = requiredString("AUTH_TOKEN_SECRET", &errs)
	cfg.Auth.AccessTokenTTL = durationDefault("AUTH_ACCESS_TOKEN_TTL", 24*time.Hour, &errs)
	cfg.Auth.EmailAllowedDomains = splitStringList(stringDefault("AUTH_EMAIL_ALLOWED_DOMAINS", "cumt.edu.cn,mail.cumt.edu.cn"))
	cfg.Auth.EmailCodeTTL = durationDefault("AUTH_EMAIL_CODE_TTL", 10*time.Minute, &errs)
	cfg.Auth.EmailCodeResendInterval = durationDefault("AUTH_EMAIL_CODE_RESEND_INTERVAL", time.Minute, &errs)
	cfg.Auth.EmailCodeMaxAttempts = intDefault("AUTH_EMAIL_CODE_MAX_ATTEMPTS", 5, &errs)
	cfg.Auth.EmailCodeDailyLimit = intDefault("AUTH_EMAIL_CODE_DAILY_LIMIT", 10, &errs)
	cfg.Auth.EmailCodeIPHourlyLimit = intDefault("AUTH_EMAIL_CODE_IP_HOURLY_LIMIT", 30, &errs)
	cfg.Auth.EmailCodeLength = intDefault("AUTH_EMAIL_CODE_LENGTH", 6, &errs)

	cfg.Mail.Provider = stringDefault("MAIL_PROVIDER", "log")
	cfg.Mail.SMTP.Host = stringDefault("SMTP_HOST", "")
	cfg.Mail.SMTP.Port = intDefault("SMTP_PORT", 587, &errs)
	cfg.Mail.SMTP.Username = stringDefault("SMTP_USERNAME", "")
	cfg.Mail.SMTP.Password = stringDefault("SMTP_PASSWORD", "")
	cfg.Mail.SMTP.From = stringDefault("SMTP_FROM", "")
	cfg.Mail.SMTP.TLSMode = stringDefault("SMTP_TLS_MODE", "starttls")

	cfg.Storage.Provider = stringDefault("OBJECT_STORAGE_PROVIDER", "local")
	cfg.Storage.Endpoint = stringDefault("OBJECT_STORAGE_ENDPOINT", "")
	cfg.Storage.Region = stringDefault("OBJECT_STORAGE_REGION", "auto")
	cfg.Storage.Bucket = stringDefault("OBJECT_STORAGE_BUCKET", "")
	cfg.Storage.AccessKeyID = stringDefault("OBJECT_STORAGE_ACCESS_KEY_ID", "")
	cfg.Storage.SecretAccessKey = stringDefault("OBJECT_STORAGE_SECRET_ACCESS_KEY", "")
	cfg.Storage.PublicBaseURL = stringDefault("OBJECT_STORAGE_PUBLIC_BASE_URL", "")
	cfg.Storage.ForcePathStyle = boolDefault("OBJECT_STORAGE_FORCE_PATH_STYLE", true, &errs)
	cfg.Storage.LocalRoot = stringDefault("OBJECT_STORAGE_LOCAL_ROOT", "var/uploads")
	if cfg.Storage.Provider == "local" && cfg.Storage.PublicBaseURL == "" {
		cfg.Storage.PublicBaseURL = "http://localhost:8080/uploads"
	}

	cfg.Upload.ImageMaxBytes = intDefault("UPLOAD_IMAGE_MAX_BYTES", 5*1024*1024, &errs)
	cfg.Upload.ImageMaxCountPerPost = intDefault("UPLOAD_IMAGE_MAX_COUNT_PER_POST", 9, &errs)
	cfg.Upload.ImageMaxCountPerComment = intDefault("UPLOAD_IMAGE_MAX_COUNT_PER_COMMENT", 1, &errs)

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
func requiredString(key string, errs *[]error) string {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		*errs = append(*errs, fmt.Errorf("%s is required", key))
		return ""
	}
	return strings.TrimSpace(v)
}

func stringDefault(key, defaultValue string) string {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return defaultValue
	}
	return strings.TrimSpace(v)
}

func stringListDefault(key string, defaultValue []string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return defaultValue
	}

	values := splitStringList(v)
	if len(values) == 0 {
		return defaultValue
	}
	return values
}

func splitStringList(v string) []string {
	parts := strings.Split(v, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func intDefault(key string, defaultValue int, errs *[]error) int {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		*errs = append(*errs, fmt.Errorf("invalid value for %s: %v", key, err))
		return defaultValue
	}
	return i
}

func boolDefault(key string, defaultValue bool, errs *[]error) bool {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		*errs = append(*errs, fmt.Errorf("invalid value for %s: %v", key, err))
		return defaultValue
	}
	return value
}

func durationDefault(key string, defaultValue time.Duration, errs *[]error) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		*errs = append(*errs, fmt.Errorf("invalid value for %s: %v", key, err))
		return defaultValue
	}
	return d
}

func loadDotEnvIfPresent() error {
	if _, err := os.Stat(".env"); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err := godotenv.Load(".env"); err != nil {
		return fmt.Errorf("load .env: %w", err)
	}

	return nil
}

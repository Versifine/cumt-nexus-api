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

	cfg.Log.Level = stringDefault("LOG_LEVEL", "info")
	cfg.Log.Format = stringDefault("LOG_FORMAT", "json")

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

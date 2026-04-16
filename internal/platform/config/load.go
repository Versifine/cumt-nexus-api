package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func Load() (*Config, error) {
	var cfg Config
	var errs []error
	cfg.App.Name = requiredString("APP_NAME", &errs)
	cfg.App.Env = stringDefault("APP_ENV", "local")

	cfg.DB.Host = requiredString("DB_HOST", &errs)
	cfg.DB.Port = intDefault("DB_PORT", 5432, &errs)
	cfg.DB.User = requiredString("DB_USER", &errs)
	cfg.DB.Password = requiredString("DB_PASSWORD", &errs)
	cfg.DB.Name = requiredString("DB_NAME", &errs)
	cfg.DB.SSLMode = stringDefault("DB_SSLMODE", "disable")
	cfg.DB.MaxOpenConns = intDefault("DB_MAX_OPEN_CONNS", 25, &errs)
	cfg.DB.MaxIdleConns = intDefault("DB_MAX_IDLE_CONNS", 25, &errs)

	cfg.HTTP.Addr = stringDefault("HTTP_ADDR", ":8080")
	cfg.HTTP.ReadTimeout = durationDefault("HTTP_READ_TIMEOUT", 5*time.Second, &errs)
	cfg.HTTP.WriteTimeout = durationDefault("HTTP_WRITE_TIMEOUT", 10*time.Second, &errs)
	cfg.HTTP.ShutdownTimeout = durationDefault("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second, &errs)

	cfg.Logger.Level = stringDefault("LOGGER_LEVEL", "info")
	cfg.Logger.Format = stringDefault("LOGGER_FORMAT", "json")

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

package config

import (
	"errors"
	"fmt"
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

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

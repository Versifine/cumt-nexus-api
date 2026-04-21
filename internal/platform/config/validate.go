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

	if cfg.HTTP.Addr == "" {
		errs = append(errs, fmt.Errorf("HTTP_ADDR cannot be empty"))
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

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

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
	if cfg.DB.MaxOpenConns <= 0 {
		errs = append(errs, fmt.Errorf("DB_MAX_OPEN_CONNS must be > 0"))
	}
	if cfg.DB.MaxIdleConns < 0 {
		errs = append(errs, fmt.Errorf("DB_MAX_IDLE_CONNS must be >= 0"))
	}
	if cfg.DB.MaxIdleConns > cfg.DB.MaxOpenConns {
		errs = append(errs, fmt.Errorf("DB_MAX_IDLE_CONNS cannot be greater than DB_MAX_OPEN_CONNS"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

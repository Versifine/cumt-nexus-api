package db

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	dsn := BuildDSN(cfg)
	poolconfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolconfig.MaxConns = int32(cfg.MaxConns)
	poolconfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolconfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	return pool, nil

}

func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	return nil
}

func Close(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}

// func BuildDSN(cfg config.PostgresConfig) string {
// 	return "postgres://" + cfg.User + ":" + cfg.Password + "@" + cfg.Host + ":" + strconv.Itoa(cfg.Port) + "/" + cfg.Database + "?sslmode=" + cfg.SSLMode
// }

func BuildDSN(cfg config.PostgresConfig) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port)),
		Path:   "/" + cfg.Database,
		RawQuery: url.Values{
			"sslmode": []string{cfg.SSLMode},
		}.Encode(),
	}
	return u.String()
}

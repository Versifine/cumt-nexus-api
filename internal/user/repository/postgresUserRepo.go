package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{
		pool: pool,
	}
}
func (ur *PostgresUserRepository) Create(ctx context.Context, user domain.User) error {
	const query = `
		INSERT INTO users (
			id,
			username,
			password_hash,
			status,
			created_at,
			updated_at
		)
		VALUES($1::uuid,$2,$3,$4,$5,$6)
	`
	_, err := ur.pool.Exec(
		ctx,
		query,
		user.ID().String(),
		user.Username().String(),
		user.PasswordHash().Raw(),
		user.Status().String(),
		user.CreatedAt(),
		user.UpdatedAt(),
	)
	//错误判断和映射怎么写?
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "users_username_uq" {
				return apperr.New(apperr.CodeConflict, "username already exists")
			}
		}
		return fmt.Errorf("create user: %w", err)
	}
	return err
}
func (ur *PostgresUserRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	const query = `
		SELECT 
			id::text,
			username,
			password_hash,
			status,
			created_at,
			updated_at
		FROM users
		WHERE id = $1::uuid
		LIMIT 1
	`
	row := ur.pool.QueryRow(ctx, query, id.String())

	return scanUser(row)
}
func (ur *PostgresUserRepository) FindByUsername(ctx context.Context, username domain.Username) (*domain.User, error) {
	const query = `
		SELECT 
			id::text,
			username,
			password_hash,
			status,
			created_at,
			updated_at
		FROM users
		WHERE username = $1
		LIMIT 1
	`
	row := ur.pool.QueryRow(ctx, query, username.String())

	return scanUser(row)
}

func scanUser(row pgx.Row) (*domain.User, error) {
	var rawID string
	var rawUsername string
	var rawPasswordHash string
	var rawStatus string
	var createdAt time.Time
	var updatedAt time.Time

	if err := row.Scan(
		&rawID,
		&rawUsername,
		&rawPasswordHash,
		&rawStatus,
		&createdAt,
		&updatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return nil, fmt.Errorf("scan user row: %w", err)
	}

	userID, err := domain.NewUserID(rawID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate user id: %v", err)
	}

	username, err := domain.NewUsername(rawUsername)
	if err != nil {
		return nil, fmt.Errorf("rehydrate username: %v", err)
	}

	passwordHash, err := domain.NewPasswordHash(rawPasswordHash)
	if err != nil {
		return nil, fmt.Errorf("rehydrate password hash: %v", err)
	}

	status, err := domain.NewUserStatus(rawStatus)
	if err != nil {
		return nil, fmt.Errorf("rehydrate user status: %v", err)
	}

	return domain.RehydrateUser(
		userID,
		username,
		passwordHash,
		status,
		createdAt,
		updatedAt,
	)
}

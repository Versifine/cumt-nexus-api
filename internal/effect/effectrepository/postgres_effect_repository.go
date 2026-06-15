package effectrepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/effect/effectusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ effectusecase.Repository = (*PostgresEffectRepository)(nil)

type PostgresEffectRepository struct {
	pool *pgxpool.Pool
}

type pointQuery interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func NewPostgresEffectRepository(pool *pgxpool.Pool) *PostgresEffectRepository {
	return &PostgresEffectRepository{
		pool: pool,
	}
}

func (repo *PostgresEffectRepository) ListActiveEffects(ctx context.Context) ([]effectusecase.Effect, error) {
	const query = `
		SELECT
			id,
			name,
			description,
			cost_points,
			asset_url,
			animation_key,
			is_active,
			created_at,
			updated_at
		FROM effects
		WHERE is_active = true
		ORDER BY cost_points ASC, id ASC
	`

	rows, err := repo.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list active effects: %w", err)
	}
	defer rows.Close()

	effects := make([]effectusecase.Effect, 0)
	for rows.Next() {
		effect, err := scanEffect(rows)
		if err != nil {
			return nil, err
		}
		effects = append(effects, effect)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active effects: %w", err)
	}

	return effects, nil
}

func (repo *PostgresEffectRepository) FindActiveEffectByID(ctx context.Context, effectID string) (effectusecase.Effect, error) {
	const query = `
		SELECT
			id,
			name,
			description,
			cost_points,
			asset_url,
			animation_key,
			is_active,
			created_at,
			updated_at
		FROM effects
		WHERE id = $1
			AND is_active = true
		LIMIT 1
	`

	effect, err := scanEffect(repo.pool.QueryRow(ctx, query, effectID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return effectusecase.Effect{}, apperr.New(apperr.CodeNotFound, "effect not found")
		}
		return effectusecase.Effect{}, err
	}
	return effect, nil
}

func (repo *PostgresEffectRepository) GetOrCreatePointAccount(ctx context.Context, userID userdomain.UserID, initialBalance int, now time.Time) (effectusecase.PointAccount, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return effectusecase.PointAccount{}, fmt.Errorf("begin point account transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	account, inserted, err := ensurePointAccount(ctx, tx, userID, initialBalance, now)
	if err != nil {
		return effectusecase.PointAccount{}, err
	}
	if inserted && initialBalance > 0 {
		if err := insertPointTransaction(ctx, tx, uuid.NewString(), userID, initialBalance, account.Balance, "initial_grant", "user_points", userID.String(), now); err != nil {
			return effectusecase.PointAccount{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return effectusecase.PointAccount{}, fmt.Errorf("commit point account transaction: %w", err)
	}

	return account, nil
}

func (repo *PostgresEffectRepository) ListPointTransactions(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]effectusecase.PointTransaction, error) {
	const query = `
		SELECT
			id::text,
			user_id::text,
			delta,
			balance_after,
			reason,
			source_type,
			source_id,
			created_at
		FROM point_transactions
		WHERE user_id = $1::uuid
		ORDER BY created_at DESC, id DESC
		LIMIT $2
		OFFSET $3
	`

	rows, err := repo.pool.Query(ctx, query, userID.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list point transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]effectusecase.PointTransaction, 0)
	for rows.Next() {
		transaction, err := scanPointTransaction(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate point transactions: %w", err)
	}

	return transactions, nil
}

func (repo *PostgresEffectRepository) ApplyCommentEffect(ctx context.Context, input effectusecase.ApplyCommentEffectRecordInput) (effectusecase.ApplyCommentEffectRecordResult, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return effectusecase.ApplyCommentEffectRecordResult{}, fmt.Errorf("begin comment effect transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	account, inserted, err := ensurePointAccount(ctx, tx, input.UserID, input.InitialGrant, input.Now)
	if err != nil {
		return effectusecase.ApplyCommentEffectRecordResult{}, err
	}
	if inserted && input.InitialGrant > 0 {
		if err := insertPointTransaction(ctx, tx, uuid.NewString(), input.UserID, input.InitialGrant, account.Balance, "initial_grant", "user_points", input.UserID.String(), input.Now); err != nil {
			return effectusecase.ApplyCommentEffectRecordResult{}, err
		}
	}
	if account.Balance < input.PointsSpent {
		return effectusecase.ApplyCommentEffectRecordResult{}, apperr.New(apperr.CodeForbidden, "insufficient points")
	}

	commentEffect, err := insertCommentEffect(ctx, tx, input)
	if err != nil {
		return effectusecase.ApplyCommentEffectRecordResult{}, err
	}
	updatedAccount, err := deductPoints(ctx, tx, input.UserID, input.PointsSpent, input.Now)
	if err != nil {
		return effectusecase.ApplyCommentEffectRecordResult{}, err
	}
	if input.PointsSpent > 0 {
		if err := insertPointTransaction(ctx, tx, uuid.NewString(), input.UserID, -input.PointsSpent, updatedAccount.Balance, "comment_effect", "comment_effect", commentEffect.ID, input.Now); err != nil {
			return effectusecase.ApplyCommentEffectRecordResult{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return effectusecase.ApplyCommentEffectRecordResult{}, fmt.Errorf("commit comment effect transaction: %w", err)
	}

	return effectusecase.ApplyCommentEffectRecordResult{
		CommentEffect: commentEffect,
		Points:        updatedAccount,
	}, nil
}

func ensurePointAccount(ctx context.Context, q pointQuery, userID userdomain.UserID, initialBalance int, now time.Time) (effectusecase.PointAccount, bool, error) {
	if initialBalance < 0 {
		return effectusecase.PointAccount{}, false, apperr.New(apperr.CodeInvalidArgument, "initial point balance is invalid")
	}

	const insertQuery = `
		INSERT INTO user_points (
			user_id,
			balance,
			lifetime_earned,
			lifetime_spent,
			updated_at
		)
		VALUES ($1::uuid, $2, $2, 0, $3)
		ON CONFLICT (user_id) DO NOTHING
		RETURNING
			user_id::text,
			balance,
			lifetime_earned,
			lifetime_spent,
			updated_at
	`

	account, err := scanPointAccount(q.QueryRow(ctx, insertQuery, userID.String(), initialBalance, now))
	if err == nil {
		return account, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return effectusecase.PointAccount{}, false, mapEffectPostgresWriteError("ensure point account", err)
	}

	const selectQuery = `
		SELECT
			user_id::text,
			balance,
			lifetime_earned,
			lifetime_spent,
			updated_at
		FROM user_points
		WHERE user_id = $1::uuid
		FOR UPDATE
	`

	account, err = scanPointAccount(q.QueryRow(ctx, selectQuery, userID.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return effectusecase.PointAccount{}, false, apperr.New(apperr.CodeNotFound, "point account not found")
		}
		return effectusecase.PointAccount{}, false, err
	}
	return account, false, nil
}

func insertCommentEffect(ctx context.Context, q pointQuery, input effectusecase.ApplyCommentEffectRecordInput) (effectusecase.CommentEffect, error) {
	const query = `
		INSERT INTO comment_effects (
			id,
			comment_id,
			effect_id,
			user_id,
			points_spent,
			created_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4::uuid, $5, $6)
		RETURNING
			id::text,
			comment_id::text,
			effect_id,
			user_id::text,
			points_spent,
			created_at
	`

	commentEffect, err := scanCommentEffect(q.QueryRow(ctx, query, input.ID, input.CommentID.String(), input.EffectID, input.UserID.String(), input.PointsSpent, input.Now))
	if err != nil {
		return effectusecase.CommentEffect{}, mapEffectPostgresWriteError("insert comment effect", err)
	}
	return commentEffect, nil
}

func deductPoints(ctx context.Context, q pointQuery, userID userdomain.UserID, pointsSpent int, now time.Time) (effectusecase.PointAccount, error) {
	if pointsSpent < 0 {
		return effectusecase.PointAccount{}, apperr.New(apperr.CodeInvalidArgument, "points spent is invalid")
	}

	const query = `
		UPDATE user_points
		SET
			balance = balance - $2,
			lifetime_spent = lifetime_spent + $2,
			updated_at = $3
		WHERE user_id = $1::uuid
			AND balance >= $2
		RETURNING
			user_id::text,
			balance,
			lifetime_earned,
			lifetime_spent,
			updated_at
	`

	account, err := scanPointAccount(q.QueryRow(ctx, query, userID.String(), pointsSpent, now))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return effectusecase.PointAccount{}, apperr.New(apperr.CodeForbidden, "insufficient points")
		}
		return effectusecase.PointAccount{}, err
	}
	return account, nil
}

func insertPointTransaction(ctx context.Context, q pointQuery, id string, userID userdomain.UserID, delta int, balanceAfter int, reason string, sourceType string, sourceID string, now time.Time) error {
	const query = `
		INSERT INTO point_transactions (
			id,
			user_id,
			delta,
			balance_after,
			reason,
			source_type,
			source_id,
			created_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8)
	`

	_, err := q.Exec(ctx, query, id, userID.String(), delta, balanceAfter, reason, sourceType, sourceID, now)
	if err != nil {
		return mapEffectPostgresWriteError("insert point transaction", err)
	}
	return nil
}

func scanEffect(row pgx.Row) (effectusecase.Effect, error) {
	var effect effectusecase.Effect
	err := row.Scan(
		&effect.ID,
		&effect.Name,
		&effect.Description,
		&effect.CostPoints,
		&effect.AssetURL,
		&effect.AnimationKey,
		&effect.IsActive,
		&effect.CreatedAt,
		&effect.UpdatedAt,
	)
	return effect, err
}

func scanPointAccount(row pgx.Row) (effectusecase.PointAccount, error) {
	var account effectusecase.PointAccount
	err := row.Scan(
		&account.UserID,
		&account.Balance,
		&account.LifetimeEarned,
		&account.LifetimeSpent,
		&account.UpdatedAt,
	)
	return account, err
}

func scanCommentEffect(row pgx.Row) (effectusecase.CommentEffect, error) {
	var commentEffect effectusecase.CommentEffect
	err := row.Scan(
		&commentEffect.ID,
		&commentEffect.CommentID,
		&commentEffect.EffectID,
		&commentEffect.UserID,
		&commentEffect.PointsSpent,
		&commentEffect.CreatedAt,
	)
	return commentEffect, err
}

func scanPointTransaction(row pgx.Row) (effectusecase.PointTransaction, error) {
	var transaction effectusecase.PointTransaction
	err := row.Scan(
		&transaction.ID,
		&transaction.UserID,
		&transaction.Delta,
		&transaction.BalanceAfter,
		&transaction.Reason,
		&transaction.SourceType,
		&transaction.SourceID,
		&transaction.CreatedAt,
	)
	return transaction, err
}

func mapEffectPostgresWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return apperr.New(apperr.CodeNotFound, "related resource not found")
		case "23505":
			return apperr.New(apperr.CodeConflict, "effect resource already exists")
		case "23514", "22P02":
			return apperr.New(apperr.CodeInvalidArgument, "effect write is invalid")
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

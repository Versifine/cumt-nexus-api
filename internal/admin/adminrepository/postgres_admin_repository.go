package adminrepository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/admin/adminusecase"
	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/platform/settings"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ adminusecase.Repository = (*PostgresAdminRepository)(nil)
var _ adminusecase.TransactionManager = (*PostgresAdminRepository)(nil)
var _ settings.Reader = (*PostgresAdminRepository)(nil)

type postgresExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresAdminRepository struct {
	pool *pgxpool.Pool
	db   postgresExecutor
}

func NewPostgresAdminRepository(pool *pgxpool.Pool) *PostgresAdminRepository {
	return &PostgresAdminRepository{
		pool: pool,
		db:   pool,
	}
}

func (repo *PostgresAdminRepository) WithinTx(ctx context.Context, fn func(ctx context.Context, repository adminusecase.Repository) error) (err error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin admin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	txRepo := &PostgresAdminRepository{
		pool: repo.pool,
		db:   tx,
	}
	if err := fn(ctx, txRepo); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit admin transaction: %w", err)
	}
	committed = true
	return nil
}

func (repo *PostgresAdminRepository) IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error) {
	const query = `
		SELECT is_platform_staff
		FROM users
		WHERE id = $1::uuid
			AND status = 'active'
		LIMIT 1
	`
	var isPlatformStaff bool
	if err := repo.db.QueryRow(ctx, query, userID.String()).Scan(&isPlatformStaff); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return false, fmt.Errorf("check admin platform staff: %w", err)
	}
	return isPlatformStaff, nil
}

func (repo *PostgresAdminRepository) ListUsers(ctx context.Context, status string, limit int, offset int) ([]adminusecase.User, error) {
	query := `
		SELECT
			id::text,
			username,
			status,
			is_platform_staff,
			created_at,
			updated_at
		FROM users
	`
	args := []any{limit, offset}
	if status != "all" {
		query += " WHERE status = $3"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2"

	rows, err := repo.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list admin users: %w", err)
	}
	defer rows.Close()

	users := make([]adminusecase.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin users: %w", err)
	}
	return users, nil
}

func (repo *PostgresAdminRepository) FindUserByID(ctx context.Context, userID userdomain.UserID) (adminusecase.User, error) {
	const query = `
		SELECT
			id::text,
			username,
			status,
			is_platform_staff,
			created_at,
			updated_at
		FROM users
		WHERE id = $1::uuid
		LIMIT 1
	`
	user, err := scanUser(repo.db.QueryRow(ctx, query, userID.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.User{}, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return adminusecase.User{}, err
	}
	return user, nil
}

func (repo *PostgresAdminRepository) UpdateUser(ctx context.Context, userID userdomain.UserID, input adminusecase.UpdateUserRecordInput) (adminusecase.User, error) {
	const query = `
		UPDATE users
		SET status = $2,
			is_platform_staff = $3,
			updated_at = $4
		WHERE id = $1::uuid
		RETURNING
			id::text,
			username,
			status,
			is_platform_staff,
			created_at,
			updated_at
	`
	user, err := scanUser(repo.db.QueryRow(ctx, query, userID.String(), input.Status, input.IsPlatformStaff, input.UpdatedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.User{}, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return adminusecase.User{}, mapAdminWriteError("update admin user", err)
	}
	return user, nil
}

func (repo *PostgresAdminRepository) ListCommunities(ctx context.Context, status string, limit int, offset int) ([]adminusecase.Community, error) {
	query := `
		SELECT
			id::text,
			slug,
			name,
			description,
			kind,
			status,
			visibility,
			created_by::text,
			created_at,
			updated_at
		FROM communities
	`
	args := []any{limit, offset}
	if status != "all" {
		query += " WHERE status = $3"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2"

	rows, err := repo.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list admin communities: %w", err)
	}
	defer rows.Close()

	communities := make([]adminusecase.Community, 0)
	for rows.Next() {
		community, err := scanCommunity(rows)
		if err != nil {
			return nil, err
		}
		communities = append(communities, community)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin communities: %w", err)
	}
	return communities, nil
}

func (repo *PostgresAdminRepository) FindCommunityByID(ctx context.Context, communityID communitydomain.CommunityID) (adminusecase.Community, error) {
	const query = `
		SELECT
			id::text,
			slug,
			name,
			description,
			kind,
			status,
			visibility,
			created_by::text,
			created_at,
			updated_at
		FROM communities
		WHERE id = $1::uuid
		LIMIT 1
	`
	community, err := scanCommunity(repo.db.QueryRow(ctx, query, communityID.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.Community{}, apperr.New(apperr.CodeNotFound, "community not found")
		}
		return adminusecase.Community{}, err
	}
	return community, nil
}

func (repo *PostgresAdminRepository) UpdateCommunityStatus(ctx context.Context, communityID communitydomain.CommunityID, status communitydomain.CommunityStatus, updatedAt time.Time) (adminusecase.Community, error) {
	const query = `
		UPDATE communities
		SET status = $2,
			updated_at = $3
		WHERE id = $1::uuid
		RETURNING
			id::text,
			slug,
			name,
			description,
			kind,
			status,
			visibility,
			created_by::text,
			created_at,
			updated_at
	`
	community, err := scanCommunity(repo.db.QueryRow(ctx, query, communityID.String(), status.String(), updatedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.Community{}, apperr.New(apperr.CodeNotFound, "community not found")
		}
		return adminusecase.Community{}, mapAdminWriteError("update admin community status", err)
	}
	return community, nil
}

func (repo *PostgresAdminRepository) ListEffects(ctx context.Context, active *bool, limit int, offset int) ([]adminusecase.Effect, error) {
	query := `
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
	`
	args := []any{limit, offset}
	if active != nil {
		query += " WHERE is_active = $3"
		args = append(args, *active)
	}
	query += " ORDER BY cost_points ASC, id ASC LIMIT $1 OFFSET $2"

	rows, err := repo.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list admin effects: %w", err)
	}
	defer rows.Close()

	effects := make([]adminusecase.Effect, 0)
	for rows.Next() {
		effect, err := scanEffect(rows)
		if err != nil {
			return nil, err
		}
		effects = append(effects, effect)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin effects: %w", err)
	}
	return effects, nil
}

func (repo *PostgresAdminRepository) FindEffectByID(ctx context.Context, effectID string) (adminusecase.Effect, error) {
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
		LIMIT 1
	`
	effect, err := scanEffect(repo.db.QueryRow(ctx, query, effectID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.Effect{}, apperr.New(apperr.CodeNotFound, "effect not found")
		}
		return adminusecase.Effect{}, err
	}
	return effect, nil
}

func (repo *PostgresAdminRepository) UpdateEffectActive(ctx context.Context, effectID string, active bool, updatedAt time.Time) (adminusecase.Effect, error) {
	const query = `
		UPDATE effects
		SET is_active = $2,
			updated_at = $3
		WHERE id = $1
		RETURNING
			id,
			name,
			description,
			cost_points,
			asset_url,
			animation_key,
			is_active,
			created_at,
			updated_at
	`
	effect, err := scanEffect(repo.db.QueryRow(ctx, query, effectID, active, updatedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.Effect{}, apperr.New(apperr.CodeNotFound, "effect not found")
		}
		return adminusecase.Effect{}, mapAdminWriteError("update admin effect active state", err)
	}
	return effect, nil
}

func (repo *PostgresAdminRepository) ListSettings(ctx context.Context) ([]adminusecase.Setting, error) {
	const query = `
		SELECT
			key,
			bool_value,
			updated_by::text,
			updated_at
		FROM admin_settings
		ORDER BY key ASC
	`
	rows, err := repo.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list admin settings: %w", err)
	}
	defer rows.Close()

	settingsRows := make([]adminusecase.Setting, 0)
	for rows.Next() {
		row, err := scanSetting(rows)
		if err != nil {
			return nil, err
		}
		settingsRows = append(settingsRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin settings: %w", err)
	}
	return settingsRows, nil
}

func (repo *PostgresAdminRepository) FindSettingByKey(ctx context.Context, key string) (adminusecase.Setting, error) {
	if _, err := settings.NormalizeKey(key); err != nil {
		return adminusecase.Setting{}, err
	}
	const query = `
		SELECT
			key,
			bool_value,
			updated_by::text,
			updated_at
		FROM admin_settings
		WHERE key = $1
		LIMIT 1
	`
	row, err := scanSetting(repo.db.QueryRow(ctx, query, key))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.Setting{}, apperr.New(apperr.CodeNotFound, "admin setting not found")
		}
		return adminusecase.Setting{}, err
	}
	return row, nil
}

func (repo *PostgresAdminRepository) SetSetting(ctx context.Context, key string, enabled bool, updatedBy userdomain.UserID, updatedAt time.Time) (adminusecase.Setting, error) {
	if _, err := settings.NormalizeKey(key); err != nil {
		return adminusecase.Setting{}, err
	}
	const query = `
		UPDATE admin_settings
		SET bool_value = $2,
			updated_by = $3::uuid,
			updated_at = $4
		WHERE key = $1
		RETURNING
			key,
			bool_value,
			updated_by::text,
			updated_at
	`
	row, err := scanSetting(repo.db.QueryRow(ctx, query, key, enabled, updatedBy.String(), updatedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.Setting{}, apperr.New(apperr.CodeNotFound, "admin setting not found")
		}
		return adminusecase.Setting{}, mapAdminWriteError("update admin setting", err)
	}
	return row, nil
}

func (repo *PostgresAdminRepository) IsEnabled(ctx context.Context, key string) (bool, error) {
	normalizedKey, err := settings.NormalizeKey(key)
	if err != nil {
		return false, err
	}
	const query = `
		SELECT bool_value
		FROM admin_settings
		WHERE key = $1
		LIMIT 1
	`
	var enabled bool
	if err := repo.db.QueryRow(ctx, query, normalizedKey).Scan(&enabled); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No row means the setting has not been explicitly configured; default to enabled.
			return true, nil
		}
		return false, fmt.Errorf("read admin setting: %w", err)
	}
	return enabled, nil
}

func (repo *PostgresAdminRepository) CreateAuditLog(ctx context.Context, log adminusecase.AuditLog) error {
	beforeState, err := json.Marshal(log.Before)
	if err != nil {
		return fmt.Errorf("marshal admin audit before state: %w", err)
	}
	afterState, err := json.Marshal(log.After)
	if err != nil {
		return fmt.Errorf("marshal admin audit after state: %w", err)
	}
	const query = `
		INSERT INTO admin_audit_logs (
			id,
			actor_id,
			action,
			target_type,
			target_id,
			before_state,
			after_state,
			created_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::jsonb, $7::jsonb, $8)
	`
	if _, err := repo.db.Exec(ctx, query, log.ID, log.ActorID, log.Action, log.TargetType, log.TargetID, string(beforeState), string(afterState), log.CreatedAt); err != nil {
		return mapAdminWriteError("create admin audit log", err)
	}
	return nil
}

func (repo *PostgresAdminRepository) ListAuditLogs(ctx context.Context, targetType string, targetID string, limit int, offset int) ([]adminusecase.AuditLog, error) {
	query := `
		SELECT
			id::text,
			actor_id::text,
			action,
			target_type,
			target_id,
			before_state,
			after_state,
			created_at
		FROM admin_audit_logs
	`
	args := []any{limit, offset}
	where := ""
	if targetType != "" {
		where = " WHERE target_type = $3"
		args = append(args, targetType)
	}
	if targetID != "" {
		if where == "" {
			where = " WHERE target_id = $" + fmt.Sprint(len(args)+1)
		} else {
			where += " AND target_id = $" + fmt.Sprint(len(args)+1)
		}
		args = append(args, targetID)
	}
	query += where + " ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2"

	rows, err := repo.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list admin audit logs: %w", err)
	}
	defer rows.Close()

	logs := make([]adminusecase.AuditLog, 0)
	for rows.Next() {
		log, err := scanAuditLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin audit logs: %w", err)
	}
	return logs, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (adminusecase.User, error) {
	var user adminusecase.User
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Status,
		&user.IsPlatformStaff,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	return user, err
}

func scanCommunity(row rowScanner) (adminusecase.Community, error) {
	var community adminusecase.Community
	var createdBy pgtype.Text
	if err := row.Scan(
		&community.ID,
		&community.Slug,
		&community.Name,
		&community.Description,
		&community.Kind,
		&community.Status,
		&community.Visibility,
		&createdBy,
		&community.CreatedAt,
		&community.UpdatedAt,
	); err != nil {
		return adminusecase.Community{}, err
	}
	if createdBy.Valid {
		community.CreatedBy = createdBy.String
	}
	return community, nil
}

func scanEffect(row rowScanner) (adminusecase.Effect, error) {
	var effect adminusecase.Effect
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

func scanSetting(row rowScanner) (adminusecase.Setting, error) {
	var setting adminusecase.Setting
	var updatedBy pgtype.Text
	if err := row.Scan(
		&setting.Key,
		&setting.Enabled,
		&updatedBy,
		&setting.UpdatedAt,
	); err != nil {
		return adminusecase.Setting{}, err
	}
	if updatedBy.Valid {
		setting.UpdatedBy = updatedBy.String
	}
	return setting, nil
}

func scanAuditLog(row rowScanner) (adminusecase.AuditLog, error) {
	var log adminusecase.AuditLog
	var beforeState []byte
	var afterState []byte
	if err := row.Scan(
		&log.ID,
		&log.ActorID,
		&log.Action,
		&log.TargetType,
		&log.TargetID,
		&beforeState,
		&afterState,
		&log.CreatedAt,
	); err != nil {
		return adminusecase.AuditLog{}, err
	}
	if len(beforeState) > 0 {
		if err := json.Unmarshal(beforeState, &log.Before); err != nil {
			return adminusecase.AuditLog{}, fmt.Errorf("unmarshal admin audit before state: %w", err)
		}
	}
	if len(afterState) > 0 {
		if err := json.Unmarshal(afterState, &log.After); err != nil {
			return adminusecase.AuditLog{}, fmt.Errorf("unmarshal admin audit after state: %w", err)
		}
	}
	if log.Before == nil {
		log.Before = map[string]any{}
	}
	if log.After == nil {
		log.After = map[string]any{}
	}
	return log, nil
}

func mapAdminWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return apperr.New(apperr.CodeNotFound, "related record not found")
		case "23514", "22P02":
			return apperr.New(apperr.CodeInvalidArgument, "admin write is invalid")
		case "23505":
			return apperr.New(apperr.CodeConflict, "admin record already exists")
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

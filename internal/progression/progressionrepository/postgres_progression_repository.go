package progressionrepository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/progression/progressionusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ progressionusecase.Repository = (*PostgresProgressionRepository)(nil)
var _ progressionusecase.TransactionManager = (*PostgresProgressionRepository)(nil)

type postgresExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresProgressionRepository struct {
	pool *pgxpool.Pool
	db   postgresExecutor
}

func NewPostgresProgressionRepository(pool *pgxpool.Pool) *PostgresProgressionRepository {
	return &PostgresProgressionRepository{pool: pool, db: pool}
}

func (repo *PostgresProgressionRepository) WithinTx(ctx context.Context, fn func(ctx context.Context, repository progressionusecase.Repository) error) (err error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin progression transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	txRepo := &PostgresProgressionRepository{pool: repo.pool, db: tx}
	if err := fn(ctx, txRepo); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit progression transaction: %w", err)
	}
	committed = true
	return nil
}

func (repo *PostgresProgressionRepository) IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error) {
	const query = `
		SELECT is_platform_staff
		FROM users
		WHERE id = $1::uuid
			AND status = 'active'
		LIMIT 1
	`
	var isStaff bool
	if err := repo.db.QueryRow(ctx, query, userID.String()).Scan(&isStaff); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return false, fmt.Errorf("check progression platform staff: %w", err)
	}
	return isStaff, nil
}

func (repo *PostgresProgressionRepository) FindUserByID(ctx context.Context, userID userdomain.UserID) (progressionusecase.User, error) {
	const query = `
		SELECT id::text, username, status
		FROM users
		WHERE id = $1::uuid
		LIMIT 1
	`
	var user progressionusecase.User
	if err := repo.db.QueryRow(ctx, query, userID.String()).Scan(&user.ID, &user.Username, &user.Status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return progressionusecase.User{}, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return progressionusecase.User{}, err
	}
	return user, nil
}

func (repo *PostgresProgressionRepository) GetOrCreateProgression(ctx context.Context, userID userdomain.UserID, now time.Time) (progressionusecase.ProgressionRecord, error) {
	if _, err := repo.db.Exec(ctx, `
		INSERT INTO user_progressions (user_id, xp_total, updated_at)
		VALUES ($1::uuid, 0, $2)
		ON CONFLICT (user_id) DO NOTHING
	`, userID.String(), now); err != nil {
		return progressionusecase.ProgressionRecord{}, mapProgressionWriteError("ensure user progression", err)
	}
	return repo.GetPublicProgression(ctx, userID, now)
}

func (repo *PostgresProgressionRepository) GetPublicProgression(ctx context.Context, userID userdomain.UserID, now time.Time) (progressionusecase.ProgressionRecord, error) {
	account, err := repo.findProgressionRow(ctx, userID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return progressionusecase.ProgressionRecord{}, err
		}
		account = progressionusecase.ProgressionRecord{
			UserID:    userID.String(),
			XPTotal:   0,
			UpdatedAt: now,
		}
	}
	count, err := repo.countActiveTitleGrants(ctx, userID, now)
	if err != nil {
		return progressionusecase.ProgressionRecord{}, err
	}
	account.TitlesCount = count
	activeTitle, err := repo.findActiveTitleSummary(ctx, userID, now)
	if err != nil {
		return progressionusecase.ProgressionRecord{}, err
	}
	account.ActiveTitle = activeTitle
	return account, nil
}

func (repo *PostgresProgressionRepository) ListXPEvents(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]progressionusecase.XPEvent, error) {
	const query = `
		SELECT
			id::text,
			user_id::text,
			delta,
			xp_total_after,
			reason,
			source_type,
			source_id,
			actor_id::text,
			created_at
		FROM xp_events
		WHERE user_id = $1::uuid
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := repo.db.Query(ctx, query, userID.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list xp events: %w", err)
	}
	defer rows.Close()

	events := make([]progressionusecase.XPEvent, 0)
	for rows.Next() {
		event, err := scanXPEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate xp events: %w", err)
	}
	return events, nil
}

func (repo *PostgresProgressionRepository) GrantXP(ctx context.Context, input progressionusecase.GrantXPRecordInput) (progressionusecase.GrantXPRecordResult, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return progressionusecase.GrantXPRecordResult{}, fmt.Errorf("begin xp transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	txRepo := &PostgresProgressionRepository{pool: repo.pool, db: tx}
	if _, err := txRepo.GetOrCreateProgression(ctx, input.UserID, input.CreatedAt); err != nil {
		return progressionusecase.GrantXPRecordResult{}, err
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO xp_event_claims (user_id, source_type, source_id, created_at)
		VALUES ($1::uuid, $2, $3, $4)
		ON CONFLICT (user_id, source_type, source_id) DO NOTHING
	`, input.UserID.String(), input.SourceType, input.SourceID, input.CreatedAt)
	if err != nil {
		return progressionusecase.GrantXPRecordResult{}, mapProgressionWriteError("claim xp event", err)
	}
	if tag.RowsAffected() == 0 {
		progression, err := txRepo.GetPublicProgression(ctx, input.UserID, input.CreatedAt)
		if err != nil {
			return progressionusecase.GrantXPRecordResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return progressionusecase.GrantXPRecordResult{}, fmt.Errorf("commit duplicate xp transaction: %w", err)
		}
		committed = true
		return progressionusecase.GrantXPRecordResult{Progression: progression, Granted: false}, nil
	}

	dayStart := time.Date(input.CreatedAt.Year(), input.CreatedAt.Month(), input.CreatedAt.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	var grantedToday int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(delta), 0)
		FROM xp_events
		WHERE user_id = $1::uuid
			AND source_type = $2
			AND created_at >= $3
			AND created_at < $4
	`, input.UserID.String(), input.SourceType, dayStart, dayEnd).Scan(&grantedToday); err != nil {
		return progressionusecase.GrantXPRecordResult{}, fmt.Errorf("sum daily xp events: %w", err)
	}
	remaining := input.DailyCap - grantedToday
	if remaining <= 0 {
		progression, err := txRepo.GetPublicProgression(ctx, input.UserID, input.CreatedAt)
		if err != nil {
			return progressionusecase.GrantXPRecordResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return progressionusecase.GrantXPRecordResult{}, fmt.Errorf("commit capped xp transaction: %w", err)
		}
		committed = true
		return progressionusecase.GrantXPRecordResult{Progression: progression, Granted: false}, nil
	}
	delta := input.Delta
	if delta > remaining {
		delta = remaining
	}
	progression, err := scanProgressionRecord(tx.QueryRow(ctx, `
		UPDATE user_progressions
		SET xp_total = xp_total + $2,
			updated_at = $3
		WHERE user_id = $1::uuid
		RETURNING user_id::text, xp_total, updated_at
	`, input.UserID.String(), delta, input.CreatedAt))
	if err != nil {
		return progressionusecase.GrantXPRecordResult{}, mapProgressionWriteError("update user progression", err)
	}
	event, err := scanXPEvent(tx.QueryRow(ctx, `
		INSERT INTO xp_events (
			id,
			user_id,
			delta,
			xp_total_after,
			reason,
			source_type,
			source_id,
			actor_id,
			created_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, NULLIF($8, '')::uuid, $9)
		RETURNING
			id::text,
			user_id::text,
			delta,
			xp_total_after,
			reason,
			source_type,
			source_id,
			actor_id::text,
			created_at
	`, input.EventID, input.UserID.String(), delta, progression.XPTotal, input.Reason, input.SourceType, input.SourceID, input.ActorID.String(), input.CreatedAt))
	if err != nil {
		return progressionusecase.GrantXPRecordResult{}, mapProgressionWriteError("insert xp event", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return progressionusecase.GrantXPRecordResult{}, fmt.Errorf("commit xp transaction: %w", err)
	}
	committed = true
	progression, err = repo.GetPublicProgression(ctx, input.UserID, input.CreatedAt)
	if err != nil {
		return progressionusecase.GrantXPRecordResult{}, err
	}
	return progressionusecase.GrantXPRecordResult{Event: &event, Progression: progression, Granted: true}, nil
}

func (repo *PostgresProgressionRepository) ListTitles(ctx context.Context, filter progressionusecase.TitleFilter, limit int, offset int) ([]progressionusecase.Title, error) {
	query := `
		SELECT
			id::text,
			name,
			description,
			scope_type,
			scope_id,
			is_active,
			created_by::text,
			created_at,
			updated_at
		FROM titles
	`
	args := []any{limit, offset}
	where := ""
	if filter.ScopeType != "" && filter.ScopeType != "all" {
		where = " WHERE scope_type = $" + fmt.Sprint(len(args)+1)
		args = append(args, filter.ScopeType)
	}
	if filter.Active != nil {
		if where == "" {
			where = " WHERE is_active = $" + fmt.Sprint(len(args)+1)
		} else {
			where += " AND is_active = $" + fmt.Sprint(len(args)+1)
		}
		args = append(args, *filter.Active)
	}
	query += where + " ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2"

	rows, err := repo.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list titles: %w", err)
	}
	defer rows.Close()

	titles := make([]progressionusecase.Title, 0)
	for rows.Next() {
		title, err := scanTitle(rows)
		if err != nil {
			return nil, err
		}
		titles = append(titles, title)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate titles: %w", err)
	}
	return titles, nil
}

func (repo *PostgresProgressionRepository) CreateTitle(ctx context.Context, input progressionusecase.CreateTitleRecordInput) (progressionusecase.Title, error) {
	return scanTitle(repo.db.QueryRow(ctx, `
		INSERT INTO titles (
			id,
			name,
			description,
			scope_type,
			scope_id,
			is_active,
			created_by,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2, $3, $4, $5, true, $6::uuid, $7, $7)
		RETURNING
			id::text,
			name,
			description,
			scope_type,
			scope_id,
			is_active,
			created_by::text,
			created_at,
			updated_at
	`, input.ID, input.Name, input.Description, input.ScopeType, input.ScopeID, input.CreatedBy.String(), input.CreatedAt))
}

func (repo *PostgresProgressionRepository) UpdateTitle(ctx context.Context, input progressionusecase.UpdateTitleRecordInput) (progressionusecase.Title, error) {
	title, err := scanTitle(repo.db.QueryRow(ctx, `
		UPDATE titles
		SET name = $2,
			description = $3,
			is_active = $4,
			updated_at = $5
		WHERE id = $1::uuid
		RETURNING
			id::text,
			name,
			description,
			scope_type,
			scope_id,
			is_active,
			created_by::text,
			created_at,
			updated_at
	`, input.TitleID, input.Name, input.Description, input.IsActive, input.UpdatedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return progressionusecase.Title{}, apperr.New(apperr.CodeNotFound, "title not found")
		}
		return progressionusecase.Title{}, mapProgressionWriteError("update title", err)
	}
	return title, nil
}

func (repo *PostgresProgressionRepository) FindTitleByID(ctx context.Context, titleID string) (progressionusecase.Title, error) {
	title, err := scanTitle(repo.db.QueryRow(ctx, `
		SELECT
			id::text,
			name,
			description,
			scope_type,
			scope_id,
			is_active,
			created_by::text,
			created_at,
			updated_at
		FROM titles
		WHERE id = $1::uuid
		LIMIT 1
	`, titleID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return progressionusecase.Title{}, apperr.New(apperr.CodeNotFound, "title not found")
		}
		return progressionusecase.Title{}, err
	}
	return title, nil
}

func (repo *PostgresProgressionRepository) ListUserTitleGrants(ctx context.Context, userID userdomain.UserID, now time.Time, limit int, offset int) ([]progressionusecase.TitleGrant, error) {
	rows, err := repo.db.Query(ctx, titleGrantSelectSQL()+`
		WHERE tg.user_id = $1::uuid
			AND tg.revoked_at IS NULL
			AND (tg.expires_at IS NULL OR tg.expires_at > $2)
			AND t.is_active = true
		ORDER BY tg.created_at DESC, tg.id DESC
		LIMIT $3 OFFSET $4
	`, userID.String(), now, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list user title grants: %w", err)
	}
	defer rows.Close()

	grants := make([]progressionusecase.TitleGrant, 0)
	for rows.Next() {
		grant, err := scanTitleGrant(rows)
		if err != nil {
			return nil, err
		}
		grants = append(grants, grant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user title grants: %w", err)
	}
	return grants, nil
}

func (repo *PostgresProgressionRepository) GrantTitle(ctx context.Context, input progressionusecase.GrantTitleRecordInput) (progressionusecase.TitleGrant, error) {
	grant, err := scanTitleGrant(repo.db.QueryRow(ctx, titleGrantInsertSQL(), input.ID, input.UserID.String(), input.TitleID, input.GrantedBy.String(), input.Reason, input.ExpiresAt, input.CreatedAt))
	if err != nil {
		return progressionusecase.TitleGrant{}, mapProgressionWriteError("grant title", err)
	}
	return grant, nil
}

func (repo *PostgresProgressionRepository) RevokeTitle(ctx context.Context, grantID string, revokedAt time.Time) (progressionusecase.TitleGrant, error) {
	grant, err := scanTitleGrant(repo.db.QueryRow(ctx, titleGrantSelectSQL()+`
		WHERE tg.id = $1::uuid
		LIMIT 1
	`, grantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return progressionusecase.TitleGrant{}, apperr.New(apperr.CodeNotFound, "title grant not found")
		}
		return progressionusecase.TitleGrant{}, err
	}
	grant, err = scanTitleGrant(repo.db.QueryRow(ctx, titleGrantSelectSQL()+`
		WHERE tg.id = $1::uuid
	`, grantID))
	if err != nil {
		return progressionusecase.TitleGrant{}, err
	}
	if grant.RevokedAt != nil {
		return grant, nil
	}
	grant, err = scanTitleGrant(repo.db.QueryRow(ctx, `
		WITH updated AS (
			UPDATE title_grants
			SET revoked_at = $2
			WHERE id = $1::uuid
			RETURNING *
		)
		SELECT
			updated.id::text,
			updated.user_id::text,
			t.id::text,
			t.name,
			t.description,
			t.scope_type,
			t.scope_id,
			t.is_active,
			t.created_by::text,
			t.created_at,
			t.updated_at,
			updated.granted_by::text,
			updated.reason,
			updated.expires_at,
			updated.revoked_at,
			updated.created_at
		FROM updated
		JOIN titles t ON t.id = updated.title_id
	`, grantID, revokedAt))
	if err != nil {
		return progressionusecase.TitleGrant{}, mapProgressionWriteError("revoke title", err)
	}
	_, _ = repo.db.Exec(ctx, `
		UPDATE user_progressions
		SET active_title_grant_id = NULL,
			updated_at = $2
		WHERE user_id = $1::uuid
			AND active_title_grant_id = $3::uuid
	`, grant.UserID, revokedAt, grantID)
	return grant, nil
}

func (repo *PostgresProgressionRepository) SetActiveTitle(ctx context.Context, userID userdomain.UserID, grantID *string, now time.Time) (progressionusecase.ProgressionRecord, error) {
	if _, err := repo.GetOrCreateProgression(ctx, userID, now); err != nil {
		return progressionusecase.ProgressionRecord{}, err
	}
	if grantID == nil {
		if _, err := repo.db.Exec(ctx, `
			UPDATE user_progressions
			SET active_title_grant_id = NULL,
				updated_at = $2
			WHERE user_id = $1::uuid
		`, userID.String(), now); err != nil {
			return progressionusecase.ProgressionRecord{}, mapProgressionWriteError("clear active title", err)
		}
		return repo.GetPublicProgression(ctx, userID, now)
	}
	tag, err := repo.db.Exec(ctx, `
		UPDATE user_progressions
		SET active_title_grant_id = $2::uuid,
			updated_at = $3
		WHERE user_id = $1::uuid
			AND EXISTS (
				SELECT 1
				FROM title_grants tg
				JOIN titles t ON t.id = tg.title_id
				WHERE tg.id = $2::uuid
					AND tg.user_id = $1::uuid
					AND tg.revoked_at IS NULL
					AND (tg.expires_at IS NULL OR tg.expires_at > $3)
					AND t.is_active = true
			)
	`, userID.String(), *grantID, now)
	if err != nil {
		return progressionusecase.ProgressionRecord{}, mapProgressionWriteError("set active title", err)
	}
	if tag.RowsAffected() == 0 {
		return progressionusecase.ProgressionRecord{}, apperr.New(apperr.CodeNotFound, "title grant not found")
	}
	return repo.GetPublicProgression(ctx, userID, now)
}

func (repo *PostgresProgressionRepository) CreateAdminAuditLog(ctx context.Context, log progressionusecase.AdminAuditLog) error {
	beforeState, err := json.Marshal(log.Before)
	if err != nil {
		return fmt.Errorf("marshal progression audit before state: %w", err)
	}
	afterState, err := json.Marshal(log.After)
	if err != nil {
		return fmt.Errorf("marshal progression audit after state: %w", err)
	}
	_, err = repo.db.Exec(ctx, `
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
	`, log.ID, log.ActorID, log.Action, log.TargetType, log.TargetID, string(beforeState), string(afterState), log.CreatedAt)
	if err != nil {
		return mapProgressionWriteError("create progression admin audit log", err)
	}
	return nil
}

func (repo *PostgresProgressionRepository) findProgressionRow(ctx context.Context, userID userdomain.UserID) (progressionusecase.ProgressionRecord, error) {
	return scanProgressionRecord(repo.db.QueryRow(ctx, `
		SELECT user_id::text, xp_total, updated_at
		FROM user_progressions
		WHERE user_id = $1::uuid
		LIMIT 1
	`, userID.String()))
}

func (repo *PostgresProgressionRepository) countActiveTitleGrants(ctx context.Context, userID userdomain.UserID, now time.Time) (int, error) {
	var count int
	err := repo.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM title_grants tg
		JOIN titles t ON t.id = tg.title_id
		WHERE tg.user_id = $1::uuid
			AND tg.revoked_at IS NULL
			AND (tg.expires_at IS NULL OR tg.expires_at > $2)
			AND t.is_active = true
	`, userID.String(), now).Scan(&count)
	return count, err
}

func (repo *PostgresProgressionRepository) findActiveTitleSummary(ctx context.Context, userID userdomain.UserID, now time.Time) (*progressionusecase.TitleSummary, error) {
	var summary progressionusecase.TitleSummary
	err := repo.db.QueryRow(ctx, `
		SELECT
			tg.id::text,
			t.id::text,
			t.name,
			t.scope_type,
			t.scope_id
		FROM user_progressions up
		JOIN title_grants tg ON tg.id = up.active_title_grant_id
		JOIN titles t ON t.id = tg.title_id
		WHERE up.user_id = $1::uuid
			AND tg.user_id = $1::uuid
			AND tg.revoked_at IS NULL
			AND (tg.expires_at IS NULL OR tg.expires_at > $2)
			AND t.is_active = true
		LIMIT 1
	`, userID.String(), now).Scan(&summary.GrantID, &summary.TitleID, &summary.Name, &summary.ScopeType, &summary.ScopeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &summary, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProgressionRecord(row rowScanner) (progressionusecase.ProgressionRecord, error) {
	var record progressionusecase.ProgressionRecord
	err := row.Scan(&record.UserID, &record.XPTotal, &record.UpdatedAt)
	return record, err
}

func scanXPEvent(row rowScanner) (progressionusecase.XPEvent, error) {
	var event progressionusecase.XPEvent
	var actorID pgtype.Text
	if err := row.Scan(
		&event.ID,
		&event.UserID,
		&event.Delta,
		&event.XPTotalAfter,
		&event.Reason,
		&event.SourceType,
		&event.SourceID,
		&actorID,
		&event.CreatedAt,
	); err != nil {
		return progressionusecase.XPEvent{}, err
	}
	if actorID.Valid {
		event.ActorID = actorID.String
	}
	return event, nil
}

func scanTitle(row rowScanner) (progressionusecase.Title, error) {
	var title progressionusecase.Title
	var createdBy pgtype.Text
	if err := row.Scan(
		&title.ID,
		&title.Name,
		&title.Description,
		&title.ScopeType,
		&title.ScopeID,
		&title.IsActive,
		&createdBy,
		&title.CreatedAt,
		&title.UpdatedAt,
	); err != nil {
		return progressionusecase.Title{}, err
	}
	if createdBy.Valid {
		title.CreatedBy = createdBy.String
	}
	return title, nil
}

func scanTitleGrant(row rowScanner) (progressionusecase.TitleGrant, error) {
	var grant progressionusecase.TitleGrant
	var titleCreatedBy pgtype.Text
	var grantedBy pgtype.Text
	var expiresAt pgtype.Timestamptz
	var revokedAt pgtype.Timestamptz
	if err := row.Scan(
		&grant.ID,
		&grant.UserID,
		&grant.Title.ID,
		&grant.Title.Name,
		&grant.Title.Description,
		&grant.Title.ScopeType,
		&grant.Title.ScopeID,
		&grant.Title.IsActive,
		&titleCreatedBy,
		&grant.Title.CreatedAt,
		&grant.Title.UpdatedAt,
		&grantedBy,
		&grant.Reason,
		&expiresAt,
		&revokedAt,
		&grant.CreatedAt,
	); err != nil {
		return progressionusecase.TitleGrant{}, err
	}
	if grantedBy.Valid {
		grant.GrantedBy = grantedBy.String
	}
	if titleCreatedBy.Valid {
		grant.Title.CreatedBy = titleCreatedBy.String
	}
	if expiresAt.Valid {
		value := expiresAt.Time
		grant.ExpiresAt = &value
	}
	if revokedAt.Valid {
		value := revokedAt.Time
		grant.RevokedAt = &value
	}
	return grant, nil
}

func titleGrantSelectSQL() string {
	return `
		SELECT
			tg.id::text,
			tg.user_id::text,
			t.id::text,
			t.name,
			t.description,
			t.scope_type,
			t.scope_id,
			t.is_active,
			t.created_by::text,
			t.created_at,
			t.updated_at,
			tg.granted_by::text,
			tg.reason,
			tg.expires_at,
			tg.revoked_at,
			tg.created_at
		FROM title_grants tg
		JOIN titles t ON t.id = tg.title_id
	`
}

func titleGrantInsertSQL() string {
	return `
		WITH inserted AS (
			INSERT INTO title_grants (
				id,
				user_id,
				title_id,
				granted_by,
				reason,
				expires_at,
				created_at
			)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7)
			RETURNING *
		)
		SELECT
			inserted.id::text,
			inserted.user_id::text,
			t.id::text,
			t.name,
			t.description,
			t.scope_type,
			t.scope_id,
			t.is_active,
			t.created_by::text,
			t.created_at,
			t.updated_at,
			inserted.granted_by::text,
			inserted.reason,
			inserted.expires_at,
			inserted.revoked_at,
			inserted.created_at
		FROM inserted
		JOIN titles t ON t.id = inserted.title_id
	`
}

func mapProgressionWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return apperr.New(apperr.CodeNotFound, "related record not found")
		case "23514", "22P02":
			return apperr.New(apperr.CodeInvalidArgument, "progression write is invalid")
		case "23505":
			return apperr.New(apperr.CodeConflict, "progression record already exists")
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

package adminrepository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/admin/adminusecase"
	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/platform/settings"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
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

func (repo *PostgresAdminRepository) ListUsers(ctx context.Context, status string, searchQuery string, limit int, offset int) ([]adminusecase.User, error) {
	query := "SELECT " + adminUserSelectFields("") + " FROM users"
	args := []any{limit, offset}
	where := make([]string, 0, 2)
	if status != "all" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if searchQuery != "" {
		args = append(args, "%"+escapeLikePattern(strings.ToLower(searchQuery))+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		where = append(where, fmt.Sprintf("(LOWER(username) LIKE %[1]s ESCAPE '\\' OR LOWER(display_name) LIKE %[1]s ESCAPE '\\' OR id::text ILIKE %[1]s ESCAPE '\\')", placeholder))
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
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
	query := "SELECT " + adminUserSelectFields("") + " FROM users WHERE id = $1::uuid LIMIT 1"
	user, err := scanUser(repo.db.QueryRow(ctx, query, userID.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.User{}, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return adminusecase.User{}, err
	}
	return user, nil
}

func (repo *PostgresAdminRepository) FindUserPasswordHash(ctx context.Context, userID userdomain.UserID) (userdomain.PasswordHash, error) {
	const query = `
		SELECT password_hash
		FROM users
		WHERE id = $1::uuid
		LIMIT 1
	`
	var rawHash string
	if err := repo.db.QueryRow(ctx, query, userID.String()).Scan(&rawHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", apperr.New(apperr.CodeNotFound, "user not found")
		}
		return "", fmt.Errorf("find user password hash: %w", err)
	}
	hash, err := userdomain.NewPasswordHash(rawHash)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func (repo *PostgresAdminRepository) UpdateUser(ctx context.Context, userID userdomain.UserID, input adminusecase.UpdateUserRecordInput) (adminusecase.User, error) {
	query := `
		UPDATE users
		SET status = $2,
			is_platform_staff = $3,
			platform_role = CASE WHEN $3 THEN COALESCE(platform_role, 'staff') ELSE NULL END,
			updated_at = $4
		WHERE id = $1::uuid
		RETURNING ` + adminUserSelectFields("")
	user, err := scanUser(repo.db.QueryRow(ctx, query, userID.String(), input.Status, input.IsPlatformStaff, input.UpdatedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.User{}, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return adminusecase.User{}, mapAdminWriteError("update admin user", err)
	}
	return user, nil
}

func (repo *PostgresAdminRepository) UpdateUserPlatformRole(ctx context.Context, userID userdomain.UserID, role string, updatedAt time.Time) (adminusecase.User, error) {
	query := `
		UPDATE users
		SET platform_role = NULLIF($2, ''),
			is_platform_staff = ($2 <> ''),
			updated_at = $3
		WHERE id = $1::uuid
		RETURNING ` + adminUserSelectFields("")
	user, err := scanUser(repo.db.QueryRow(ctx, query, userID.String(), role, updatedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.User{}, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return adminusecase.User{}, mapAdminWriteError("update user platform role", err)
	}
	return user, nil
}

func (repo *PostgresAdminRepository) CountPlatformOwners(ctx context.Context) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM users
		WHERE status = 'active'
			AND platform_role = 'owner'
	`
	var count int
	if err := repo.db.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("count platform owners: %w", err)
	}
	return count, nil
}

func (repo *PostgresAdminRepository) FindCurrentOwnerTransfer(ctx context.Context, now time.Time) (adminusecase.OwnerTransfer, error) {
	if err := repo.expireOwnerTransfers(ctx, now); err != nil {
		return adminusecase.OwnerTransfer{}, err
	}
	transfer, err := scanOwnerTransfer(repo.db.QueryRow(ctx, ownerTransferSelectSQL()+`
		WHERE platform_owner_transfers.status = 'pending'
		ORDER BY platform_owner_transfers.created_at DESC, platform_owner_transfers.id DESC
		LIMIT 1
	`, now))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.OwnerTransfer{}, apperr.New(apperr.CodeNotFound, "owner transfer not found")
		}
		return adminusecase.OwnerTransfer{}, err
	}
	return transfer, nil
}

func (repo *PostgresAdminRepository) FindOwnerTransferByID(ctx context.Context, transferID string, now time.Time) (adminusecase.OwnerTransfer, error) {
	if err := repo.expireOwnerTransfers(ctx, now); err != nil {
		return adminusecase.OwnerTransfer{}, err
	}
	transfer, err := scanOwnerTransfer(repo.db.QueryRow(ctx, ownerTransferSelectSQL()+`
		WHERE platform_owner_transfers.id = $2::uuid
		LIMIT 1
	`, now, transferID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.OwnerTransfer{}, apperr.New(apperr.CodeNotFound, "owner transfer not found")
		}
		return adminusecase.OwnerTransfer{}, err
	}
	return transfer, nil
}

func (repo *PostgresAdminRepository) ListOwnerTransfersByTarget(ctx context.Context, targetUserID userdomain.UserID, status string, now time.Time, limit int, offset int) ([]adminusecase.OwnerTransfer, error) {
	if err := repo.expireOwnerTransfers(ctx, now); err != nil {
		return nil, err
	}
	query := ownerTransferSelectSQL() + `
		WHERE platform_owner_transfers.target_user_id = $2::uuid
	`
	args := []any{now, targetUserID.String(), limit, offset}
	switch status {
	case adminusecase.OwnerTransferStatusPending:
		query += ` AND platform_owner_transfers.status = 'pending' AND platform_owner_transfers.expires_at > $1`
	case adminusecase.OwnerTransferStatusAccepted:
		query += ` AND platform_owner_transfers.status = 'accepted' AND $1::timestamptz IS NOT NULL`
	case adminusecase.OwnerTransferStatusCancelled:
		query += ` AND platform_owner_transfers.status IN ('cancelled', 'canceled') AND $1::timestamptz IS NOT NULL`
	case adminusecase.OwnerTransferStatusExpired:
		query += ` AND (platform_owner_transfers.status = 'expired' OR (platform_owner_transfers.status = 'pending' AND platform_owner_transfers.expires_at <= $1))`
	case "all":
		query += ` AND $1::timestamptz IS NOT NULL`
	default:
		return nil, apperr.New(apperr.CodeInvalidArgument, "owner transfer status is invalid")
	}
	query += `
		ORDER BY platform_owner_transfers.created_at DESC, platform_owner_transfers.id DESC
		LIMIT $3
		OFFSET $4
	`
	rows, err := repo.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list owner transfers by target: %w", err)
	}
	defer rows.Close()

	transfers := make([]adminusecase.OwnerTransfer, 0)
	for rows.Next() {
		transfer, err := scanOwnerTransfer(rows)
		if err != nil {
			return nil, err
		}
		transfers = append(transfers, transfer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate owner transfers by target: %w", err)
	}
	return transfers, nil
}

func (repo *PostgresAdminRepository) CreateOwnerTransfer(ctx context.Context, input adminusecase.CreateOwnerTransferRecordInput) (adminusecase.OwnerTransfer, error) {
	if err := repo.expireOwnerTransfers(ctx, input.CreatedAt); err != nil {
		return adminusecase.OwnerTransfer{}, err
	}
	transfer, err := scanOwnerTransfer(repo.db.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO platform_owner_transfers (
				id,
				status,
				initiated_by,
				target_user_id,
				previous_owner_role,
				reason,
				expires_at,
				created_at,
				updated_at
			)
			VALUES ($1::uuid, 'pending', $2::uuid, $3::uuid, NULLIF($4, ''), $5, $6, $7, $7)
			RETURNING *
		)
		SELECT
			inserted.id::text,
			inserted.status,
			inserted.initiated_by::text,
			initiator.username,
			COALESCE(NULLIF(initiator.display_name, ''), initiator.username),
			inserted.target_user_id::text,
			target.username,
			COALESCE(NULLIF(target.display_name, ''), target.username),
			COALESCE(inserted.previous_owner_role, ''),
			inserted.reason,
			inserted.created_at,
			inserted.updated_at,
			inserted.expires_at,
			inserted.accepted_at,
			inserted.cancelled_at
		FROM inserted
		INNER JOIN users AS initiator ON initiator.id = inserted.initiated_by
		INNER JOIN users AS target ON target.id = inserted.target_user_id
	`, input.ID, input.InitiatedByID.String(), input.TargetUserID.String(), input.PreviousOwnerRole, input.Reason, input.ExpiresAt, input.CreatedAt))
	if err != nil {
		return adminusecase.OwnerTransfer{}, mapAdminWriteError("create owner transfer", err)
	}
	return transfer, nil
}

func (repo *PostgresAdminRepository) CancelOwnerTransfer(ctx context.Context, transferID string, cancelledAt time.Time) (adminusecase.OwnerTransfer, error) {
	transfer, err := scanOwnerTransfer(repo.db.QueryRow(ctx, `
		WITH updated AS (
			UPDATE platform_owner_transfers
			SET status = 'cancelled',
				cancelled_at = $2,
				updated_at = $2
			WHERE id = $1::uuid
				AND status = 'pending'
				AND expires_at > $2
			RETURNING *
		)
		SELECT
			updated.id::text,
			updated.status,
			updated.initiated_by::text,
			initiator.username,
			COALESCE(NULLIF(initiator.display_name, ''), initiator.username),
			updated.target_user_id::text,
			target.username,
			COALESCE(NULLIF(target.display_name, ''), target.username),
			COALESCE(updated.previous_owner_role, ''),
			updated.reason,
			updated.created_at,
			updated.updated_at,
			updated.expires_at,
			updated.accepted_at,
			updated.cancelled_at
		FROM updated
		INNER JOIN users AS initiator ON initiator.id = updated.initiated_by
		INNER JOIN users AS target ON target.id = updated.target_user_id
	`, transferID, cancelledAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.OwnerTransfer{}, apperr.New(apperr.CodeConflict, "owner transfer is not pending")
		}
		return adminusecase.OwnerTransfer{}, mapAdminWriteError("cancel owner transfer", err)
	}
	return transfer, nil
}

func (repo *PostgresAdminRepository) AcceptOwnerTransfer(ctx context.Context, transferID string, acceptedAt time.Time) (adminusecase.OwnerTransfer, error) {
	var initiatedBy string
	var targetUserID string
	var previousOwnerRole pgtype.Text
	if err := repo.db.QueryRow(ctx, `
		SELECT initiated_by::text, target_user_id::text, previous_owner_role
		FROM platform_owner_transfers
		WHERE id = $1::uuid
			AND status = 'pending'
			AND expires_at > $2
		LIMIT 1
		FOR UPDATE
	`, transferID, acceptedAt).Scan(&initiatedBy, &targetUserID, &previousOwnerRole); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.OwnerTransfer{}, apperr.New(apperr.CodeConflict, "owner transfer is not pending")
		}
		return adminusecase.OwnerTransfer{}, fmt.Errorf("lock owner transfer: %w", err)
	}
	if err := lockOwnerTransferUsers(ctx, repo.db, initiatedBy, targetUserID); err != nil {
		return adminusecase.OwnerTransfer{}, err
	}
	if _, err := repo.db.Exec(ctx, `
		UPDATE users
		SET platform_role = NULL,
			is_platform_staff = false,
			updated_at = $1
		WHERE status = 'active'
			AND platform_role = 'owner'
			AND id <> $2::uuid
			AND id <> $3::uuid
	`, acceptedAt, targetUserID, initiatedBy); err != nil {
		return adminusecase.OwnerTransfer{}, mapAdminWriteError("clear extra platform owners", err)
	}
	previousRole := ""
	if previousOwnerRole.Valid {
		previousRole = previousOwnerRole.String
	}
	if _, err := repo.db.Exec(ctx, `
		UPDATE users
		SET platform_role = NULLIF($2, ''),
			is_platform_staff = ($2 <> ''),
			tokens_revoked_after = $3,
			updated_at = $3
		WHERE id = $1::uuid
	`, initiatedBy, previousRole, acceptedAt); err != nil {
		return adminusecase.OwnerTransfer{}, mapAdminWriteError("downgrade previous platform owner", err)
	}
	if _, err := repo.db.Exec(ctx, `
		UPDATE users
		SET platform_role = 'owner',
			is_platform_staff = true,
			updated_at = $2
		WHERE id = $1::uuid
			AND status = 'active'
	`, targetUserID, acceptedAt); err != nil {
		return adminusecase.OwnerTransfer{}, mapAdminWriteError("promote new platform owner", err)
	}
	transfer, err := scanOwnerTransfer(repo.db.QueryRow(ctx, `
		WITH updated AS (
			UPDATE platform_owner_transfers
			SET status = 'accepted',
				accepted_at = $2,
				updated_at = $2
			WHERE id = $1::uuid
			RETURNING *
		)
		SELECT
			updated.id::text,
			updated.status,
			updated.initiated_by::text,
			initiator.username,
			COALESCE(NULLIF(initiator.display_name, ''), initiator.username),
			updated.target_user_id::text,
			target.username,
			COALESCE(NULLIF(target.display_name, ''), target.username),
			COALESCE(updated.previous_owner_role, ''),
			updated.reason,
			updated.created_at,
			updated.updated_at,
			updated.expires_at,
			updated.accepted_at,
			updated.cancelled_at
		FROM updated
		INNER JOIN users AS initiator ON initiator.id = updated.initiated_by
		INNER JOIN users AS target ON target.id = updated.target_user_id
	`, transferID, acceptedAt))
	if err != nil {
		return adminusecase.OwnerTransfer{}, mapAdminWriteError("mark owner transfer accepted", err)
	}
	return transfer, nil
}

func (repo *PostgresAdminRepository) BootstrapOwner(ctx context.Context, input adminusecase.BootstrapOwnerRecordInput) (adminusecase.User, error) {
	query := `
		UPDATE users
		SET platform_role = 'owner',
			is_platform_staff = true,
			updated_at = $2
		WHERE id = $1::uuid
			AND status = 'active'
		RETURNING ` + adminUserSelectFields("")
	user, err := scanUser(repo.db.QueryRow(ctx, query, input.UserID.String(), input.UpdatedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.User{}, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return adminusecase.User{}, mapAdminWriteError("bootstrap platform owner", err)
	}
	return user, nil
}

func (repo *PostgresAdminRepository) RecoverOwner(ctx context.Context, input adminusecase.RecoverOwnerRecordInput) (adminusecase.OwnerRecoveryRecordResult, error) {
	if err := lockOwnerTransferUsers(ctx, repo.db, input.CompromisedUserID.String(), input.NewOwnerID.String()); err != nil {
		return adminusecase.OwnerRecoveryRecordResult{}, err
	}
	if _, err := repo.db.Exec(ctx, `
		UPDATE users
		SET platform_role = NULL,
			is_platform_staff = false,
			updated_at = $1
		WHERE status = 'active'
			AND platform_role = 'owner'
			AND id <> $2::uuid
			AND id <> $3::uuid
	`, input.UpdatedAt, input.NewOwnerID.String(), input.CompromisedUserID.String()); err != nil {
		return adminusecase.OwnerRecoveryRecordResult{}, mapAdminWriteError("clear extra platform owners", err)
	}
	compromisedStatusExpr := "status"
	if input.DisableCompromised {
		compromisedStatusExpr = "'disabled'"
	}
	revokeExpr := "tokens_revoked_after"
	if input.RevokeSessions {
		revokeExpr = "$2"
	}
	compromisedQuery := fmt.Sprintf(`
		UPDATE users
		SET platform_role = NULL,
			is_platform_staff = false,
			status = %s,
			tokens_revoked_after = %s,
			updated_at = $2
		WHERE id = $1::uuid
		RETURNING %s
	`, compromisedStatusExpr, revokeExpr, adminUserSelectFields(""))
	compromised, err := scanUser(repo.db.QueryRow(ctx, compromisedQuery, input.CompromisedUserID.String(), input.UpdatedAt))
	if err != nil {
		return adminusecase.OwnerRecoveryRecordResult{}, mapAdminWriteError("remove compromised platform owner", err)
	}
	newOwnerQuery := `
		UPDATE users
		SET platform_role = 'owner',
			is_platform_staff = true,
			updated_at = $2
		WHERE id = $1::uuid
			AND status = 'active'
		RETURNING ` + adminUserSelectFields("")
	newOwner, err := scanUser(repo.db.QueryRow(ctx, newOwnerQuery, input.NewOwnerID.String(), input.UpdatedAt))
	if err != nil {
		return adminusecase.OwnerRecoveryRecordResult{}, mapAdminWriteError("promote recovered platform owner", err)
	}
	return adminusecase.OwnerRecoveryRecordResult{
		NewOwner:        newOwner,
		CompromisedUser: compromised,
	}, nil
}

func (repo *PostgresAdminRepository) ListCommunities(ctx context.Context, status string, searchQuery string, limit int, offset int) ([]adminusecase.Community, error) {
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
	where := make([]string, 0, 2)
	if status != "all" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if searchQuery != "" {
		args = append(args, "%"+escapeLikePattern(strings.ToLower(searchQuery))+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		where = append(where, fmt.Sprintf(`(
			LOWER(slug) LIKE %[1]s ESCAPE '\'
			OR LOWER(name) LIKE %[1]s ESCAPE '\'
			OR LOWER(description) LIKE %[1]s ESCAPE '\'
			OR id::text ILIKE %[1]s ESCAPE '\'
			OR created_by::text ILIKE %[1]s ESCAPE '\'
		)`, placeholder))
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
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

func (repo *PostgresAdminRepository) TransferCommunityOwner(ctx context.Context, communityID communitydomain.CommunityID, newOwnerID userdomain.UserID, updatedAt time.Time) (adminusecase.CommunityOwnerChange, error) {
	before, err := scanCommunityOwnerMember(repo.db.QueryRow(ctx, `
		SELECT
			users.id::text,
			users.username,
			community_memberships.role,
			community_memberships.status,
			community_memberships.updated_at
		FROM community_memberships
		INNER JOIN users ON users.id = community_memberships.user_id
		WHERE community_memberships.community_id = $1::uuid
			AND community_memberships.role = 'owner'
			AND community_memberships.status = 'active'
		LIMIT 1
		FOR UPDATE OF community_memberships
	`, communityID.String()), "community owner not found")
	if err != nil {
		if !apperr.IsCode(err, apperr.CodeNotFound) {
			return adminusecase.CommunityOwnerChange{}, fmt.Errorf("find current community owner: %w", err)
		}
	}
	if _, err := scanCommunityOwnerMember(repo.db.QueryRow(ctx, `
		SELECT
			id::text,
			username,
			'' AS role,
			status,
			updated_at
		FROM users
		WHERE id = $1::uuid
			AND status = 'active'
		LIMIT 1
	`, newOwnerID.String()), "user not found"); err != nil {
		return adminusecase.CommunityOwnerChange{}, fmt.Errorf("find new community owner: %w", err)
	}
	if _, err := repo.db.Exec(ctx, `
		UPDATE community_memberships
		SET role = 'member',
			updated_at = $2
		WHERE community_id = $1::uuid
			AND role = 'owner'
			AND status = 'active'
	`, communityID.String(), updatedAt); err != nil {
		return adminusecase.CommunityOwnerChange{}, fmt.Errorf("demote current community owner: %w", err)
	}
	if _, err := repo.db.Exec(ctx, `
		INSERT INTO community_memberships (
			community_id,
			user_id,
			role,
			status,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, 'owner', 'active', $3, $3)
		ON CONFLICT (community_id, user_id) DO UPDATE
		SET role = 'owner',
			status = 'active',
			updated_at = EXCLUDED.updated_at
	`, communityID.String(), newOwnerID.String(), updatedAt); err != nil {
		return adminusecase.CommunityOwnerChange{}, mapAdminWriteError("upsert community owner", err)
	}
	after, err := scanCommunityOwnerMember(repo.db.QueryRow(ctx, `
		SELECT
			users.id::text,
			users.username,
			community_memberships.role,
			community_memberships.status,
			community_memberships.updated_at
		FROM community_memberships
		INNER JOIN users ON users.id = community_memberships.user_id
		WHERE community_memberships.community_id = $1::uuid
			AND community_memberships.user_id = $2::uuid
			AND community_memberships.status = 'active'
		LIMIT 1
	`, communityID.String(), newOwnerID.String()), "community owner not found")
	if err != nil {
		return adminusecase.CommunityOwnerChange{}, fmt.Errorf("find updated community owner: %w", err)
	}
	var beforeOwner *adminusecase.CommunityOwnerMember
	if before.UserID != "" {
		beforeOwner = &before
	}
	return adminusecase.CommunityOwnerChange{BeforeOwner: beforeOwner, AfterOwner: after}, nil
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

func (repo *PostgresAdminRepository) ListPointTransactions(ctx context.Context, userID *userdomain.UserID, limit int, offset int) ([]adminusecase.PointTransaction, error) {
	query := `
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
	`
	args := []any{limit, offset}
	if userID != nil {
		query += " WHERE user_id = $3::uuid"
		args = append(args, userID.String())
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2"

	rows, err := repo.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list admin point transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]adminusecase.PointTransaction, 0)
	for rows.Next() {
		transaction, err := scanPointTransaction(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin point transactions: %w", err)
	}
	return transactions, nil
}

func (repo *PostgresAdminRepository) AdjustUserPoints(ctx context.Context, input adminusecase.AdjustUserPointsRecordInput) (adminusecase.AdjustUserPointsRecordResult, error) {
	if input.Delta == 0 {
		return adminusecase.AdjustUserPointsRecordResult{}, apperr.New(apperr.CodeInvalidArgument, "point adjustment delta must not be zero")
	}
	if _, err := repo.db.Exec(ctx, `
		INSERT INTO user_points (
			user_id,
			balance,
			lifetime_earned,
			lifetime_spent,
			updated_at
		)
		VALUES ($1::uuid, 0, 0, 0, $2)
		ON CONFLICT (user_id) DO NOTHING
	`, input.UserID.String(), input.CreatedAt); err != nil {
		return adminusecase.AdjustUserPointsRecordResult{}, mapAdminWriteError("ensure admin point account", err)
	}

	account, err := scanPointAccount(repo.db.QueryRow(ctx, `
		SELECT
			user_id::text,
			balance,
			lifetime_earned,
			lifetime_spent,
			updated_at
		FROM user_points
		WHERE user_id = $1::uuid
		FOR UPDATE
	`, input.UserID.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.AdjustUserPointsRecordResult{}, apperr.New(apperr.CodeNotFound, "point account not found")
		}
		return adminusecase.AdjustUserPointsRecordResult{}, err
	}

	if account.Balance+input.Delta < 0 {
		return adminusecase.AdjustUserPointsRecordResult{}, apperr.New(apperr.CodeForbidden, "insufficient points")
	}

	lifetimeEarnedDelta := 0
	lifetimeSpentDelta := 0
	if input.Delta > 0 {
		lifetimeEarnedDelta = input.Delta
	} else {
		lifetimeSpentDelta = -input.Delta
	}

	updatedAccount, err := scanPointAccount(repo.db.QueryRow(ctx, `
		UPDATE user_points
		SET
			balance = balance + $2,
			lifetime_earned = lifetime_earned + $3,
			lifetime_spent = lifetime_spent + $4,
			updated_at = $5
		WHERE user_id = $1::uuid
		RETURNING
			user_id::text,
			balance,
			lifetime_earned,
			lifetime_spent,
			updated_at
	`, input.UserID.String(), input.Delta, lifetimeEarnedDelta, lifetimeSpentDelta, input.CreatedAt))
	if err != nil {
		return adminusecase.AdjustUserPointsRecordResult{}, mapAdminWriteError("adjust admin point account", err)
	}

	transaction, err := scanPointTransaction(repo.db.QueryRow(ctx, `
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
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, 'admin_adjustment', $6, $7)
		RETURNING
			id::text,
			user_id::text,
			delta,
			balance_after,
			reason,
			source_type,
			source_id,
			created_at
	`, input.TransactionID, input.UserID.String(), input.Delta, updatedAccount.Balance, input.Reason, input.ActorID.String(), input.CreatedAt))
	if err != nil {
		return adminusecase.AdjustUserPointsRecordResult{}, mapAdminWriteError("insert admin point transaction", err)
	}

	return adminusecase.AdjustUserPointsRecordResult{
		Account:     updatedAccount,
		Transaction: transaction,
	}, nil
}

func (repo *PostgresAdminRepository) CreateUserSanction(ctx context.Context, input adminusecase.CreateUserSanctionRecordInput) (adminusecase.UserSanction, error) {
	sanction, err := scanUserSanction(repo.db.QueryRow(ctx, `
		INSERT INTO user_sanctions (
			id,
			user_id,
			type,
			status,
			reason,
			created_by,
			starts_at,
			expires_at,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3, 'active', $4, $5::uuid, $6, $7, $8, $8)
		RETURNING
			id::text,
			user_id::text,
			type,
			status,
			reason,
			created_by::text,
			starts_at,
			expires_at,
			revoked_by::text,
			revoked_at,
			created_at,
			updated_at
	`, input.ID, input.UserID.String(), input.Type, input.Reason, input.CreatedBy.String(), input.StartsAt, input.ExpiresAt, input.CreatedAt))
	if err != nil {
		return adminusecase.UserSanction{}, mapAdminWriteError("create user sanction", err)
	}
	return sanction, nil
}

func (repo *PostgresAdminRepository) ListUserSanctions(ctx context.Context, userID userdomain.UserID, limit int, offset int, now time.Time) ([]adminusecase.UserSanction, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT
			id::text,
			user_id::text,
			type,
			CASE
				WHEN status = 'active' AND expires_at IS NOT NULL AND expires_at <= $4 THEN 'expired'
				ELSE status
			END AS status,
			reason,
			created_by::text,
			starts_at,
			expires_at,
			revoked_by::text,
			revoked_at,
			created_at,
			updated_at
		FROM user_sanctions
		WHERE user_id = $1::uuid
		ORDER BY created_at DESC, id DESC
		LIMIT $2
		OFFSET $3
	`, userID.String(), limit, offset, now)
	if err != nil {
		return nil, fmt.Errorf("list user sanctions: %w", err)
	}
	defer rows.Close()

	sanctions := make([]adminusecase.UserSanction, 0)
	for rows.Next() {
		sanction, err := scanUserSanction(rows)
		if err != nil {
			return nil, err
		}
		sanctions = append(sanctions, sanction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user sanctions: %w", err)
	}
	return sanctions, nil
}

func (repo *PostgresAdminRepository) FindUserSanctionByID(ctx context.Context, sanctionID string, now time.Time) (adminusecase.UserSanction, error) {
	sanction, err := scanUserSanction(repo.db.QueryRow(ctx, `
		SELECT
			id::text,
			user_id::text,
			type,
			CASE
				WHEN status = 'active' AND expires_at IS NOT NULL AND expires_at <= $2 THEN 'expired'
				ELSE status
			END AS status,
			reason,
			created_by::text,
			starts_at,
			expires_at,
			revoked_by::text,
			revoked_at,
			created_at,
			updated_at
		FROM user_sanctions
		WHERE id = $1::uuid
		LIMIT 1
	`, sanctionID, now))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.UserSanction{}, apperr.New(apperr.CodeNotFound, "user sanction not found")
		}
		return adminusecase.UserSanction{}, err
	}
	return sanction, nil
}

func (repo *PostgresAdminRepository) RevokeUserSanction(ctx context.Context, sanctionID string, actorID userdomain.UserID, revokedAt time.Time) (adminusecase.UserSanction, error) {
	sanction, err := scanUserSanction(repo.db.QueryRow(ctx, `
		UPDATE user_sanctions
		SET status = 'revoked',
			revoked_by = $2::uuid,
			revoked_at = $3,
			updated_at = $3
		WHERE id = $1::uuid
			AND status = 'active'
			AND (expires_at IS NULL OR expires_at > $3)
		RETURNING
			id::text,
			user_id::text,
			type,
			status,
			reason,
			created_by::text,
			starts_at,
			expires_at,
			revoked_by::text,
			revoked_at,
			created_at,
			updated_at
	`, sanctionID, actorID.String(), revokedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.UserSanction{}, apperr.New(apperr.CodeConflict, "user sanction is not active")
		}
		return adminusecase.UserSanction{}, mapAdminWriteError("revoke user sanction", err)
	}
	return sanction, nil
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
			actor_ref,
			action,
			target_type,
			target_id,
			before_state,
			after_state,
			created_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9)
	`
	var actorID *string
	if parsed, err := uuid.Parse(strings.TrimSpace(log.ActorID)); err == nil {
		value := parsed.String()
		actorID = &value
	}
	if _, err := repo.db.Exec(ctx, query, log.ID, actorID, strings.TrimSpace(log.ActorID), log.Action, log.TargetType, log.TargetID, string(beforeState), string(afterState), log.CreatedAt); err != nil {
		return mapAdminWriteError("create admin audit log", err)
	}
	return nil
}

func (repo *PostgresAdminRepository) ListAuditLogs(ctx context.Context, targetType string, targetID string, searchQuery string, limit int, offset int) ([]adminusecase.AuditLog, error) {
	query := `
		SELECT
			id::text,
			COALESCE(actor_ref, actor_id::text),
			action,
			target_type,
			target_id,
			before_state,
			after_state,
			created_at
		FROM admin_audit_logs
	`
	args := []any{limit, offset}
	where := make([]string, 0, 3)
	if targetType != "" {
		args = append(args, targetType)
		where = append(where, fmt.Sprintf("target_type = $%d", len(args)))
	}
	if targetID != "" {
		args = append(args, targetID)
		where = append(where, fmt.Sprintf("target_id = $%d", len(args)))
	}
	if searchQuery != "" {
		args = append(args, "%"+escapeLikePattern(strings.ToLower(searchQuery))+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		where = append(where, fmt.Sprintf(`(
			LOWER(action) LIKE %[1]s ESCAPE '\'
			OR LOWER(target_type) LIKE %[1]s ESCAPE '\'
			OR target_id ILIKE %[1]s ESCAPE '\'
			OR COALESCE(actor_ref, actor_id::text) ILIKE %[1]s ESCAPE '\'
		)`, placeholder))
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2"

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

func adminUserSelectFields(prefix string) string {
	if prefix != "" && !strings.HasSuffix(prefix, ".") {
		prefix += "."
	}
	return fmt.Sprintf(`
			%[1]sid::text,
			%[1]susername,
			%[1]sdisplay_name,
			%[1]savatar_url,
			%[1]sheadline,
			%[1]sstatus,
			%[1]sis_platform_staff,
			COALESCE(%[1]splatform_role, ''),
			%[1]screated_at,
			%[1]supdated_at
	`, prefix)
}

func scanUser(row rowScanner) (adminusecase.User, error) {
	var user adminusecase.User
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Headline,
		&user.Status,
		&user.IsPlatformStaff,
		&user.PlatformRole,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	return user, err
}

func scanOwnerTransfer(row rowScanner) (adminusecase.OwnerTransfer, error) {
	var transfer adminusecase.OwnerTransfer
	var acceptedAt pgtype.Timestamptz
	var cancelledAt pgtype.Timestamptz
	if err := row.Scan(
		&transfer.ID,
		&transfer.Status,
		&transfer.InitiatedByID,
		&transfer.InitiatedByUsername,
		&transfer.InitiatedByDisplayName,
		&transfer.TargetUserID,
		&transfer.TargetUsername,
		&transfer.TargetDisplayName,
		&transfer.PreviousOwnerRole,
		&transfer.Reason,
		&transfer.CreatedAt,
		&transfer.UpdatedAt,
		&transfer.ExpiresAt,
		&acceptedAt,
		&cancelledAt,
	); err != nil {
		return adminusecase.OwnerTransfer{}, err
	}
	if acceptedAt.Valid {
		value := acceptedAt.Time
		transfer.AcceptedAt = &value
	}
	if cancelledAt.Valid {
		value := cancelledAt.Time
		transfer.CancelledAt = &value
	}
	return transfer, nil
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

func scanCommunityOwnerMember(row rowScanner, notFoundMessage string) (adminusecase.CommunityOwnerMember, error) {
	var member adminusecase.CommunityOwnerMember
	if err := row.Scan(&member.UserID, &member.Username, &member.Role, &member.Status, &member.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminusecase.CommunityOwnerMember{}, apperr.New(apperr.CodeNotFound, notFoundMessage)
		}
		return adminusecase.CommunityOwnerMember{}, err
	}
	return member, nil
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

func scanPointAccount(row rowScanner) (adminusecase.PointAccount, error) {
	var account adminusecase.PointAccount
	err := row.Scan(
		&account.UserID,
		&account.Balance,
		&account.LifetimeEarned,
		&account.LifetimeSpent,
		&account.UpdatedAt,
	)
	return account, err
}

func scanPointTransaction(row rowScanner) (adminusecase.PointTransaction, error) {
	var transaction adminusecase.PointTransaction
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

func scanUserSanction(row rowScanner) (adminusecase.UserSanction, error) {
	var sanction adminusecase.UserSanction
	var expiresAt pgtype.Timestamptz
	var revokedBy pgtype.Text
	var revokedAt pgtype.Timestamptz
	if err := row.Scan(
		&sanction.ID,
		&sanction.UserID,
		&sanction.Type,
		&sanction.Status,
		&sanction.Reason,
		&sanction.CreatedBy,
		&sanction.StartsAt,
		&expiresAt,
		&revokedBy,
		&revokedAt,
		&sanction.CreatedAt,
		&sanction.UpdatedAt,
	); err != nil {
		return adminusecase.UserSanction{}, err
	}
	if expiresAt.Valid {
		value := expiresAt.Time
		sanction.ExpiresAt = &value
	}
	if revokedBy.Valid {
		sanction.RevokedBy = revokedBy.String
	}
	if revokedAt.Valid {
		value := revokedAt.Time
		sanction.RevokedAt = &value
	}
	return sanction, nil
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

func ownerTransferSelectSQL() string {
	return `
		SELECT
			platform_owner_transfers.id::text,
			CASE
				WHEN platform_owner_transfers.status = 'pending'
					AND platform_owner_transfers.expires_at <= $1 THEN 'expired'
				ELSE platform_owner_transfers.status
			END AS status,
			platform_owner_transfers.initiated_by::text,
			initiator.username,
			COALESCE(NULLIF(initiator.display_name, ''), initiator.username),
			platform_owner_transfers.target_user_id::text,
			target.username,
			COALESCE(NULLIF(target.display_name, ''), target.username),
			COALESCE(platform_owner_transfers.previous_owner_role, ''),
			platform_owner_transfers.reason,
			platform_owner_transfers.created_at,
			platform_owner_transfers.updated_at,
			platform_owner_transfers.expires_at,
			platform_owner_transfers.accepted_at,
			platform_owner_transfers.cancelled_at
		FROM platform_owner_transfers
		INNER JOIN users AS initiator ON initiator.id = platform_owner_transfers.initiated_by
		INNER JOIN users AS target ON target.id = platform_owner_transfers.target_user_id
	`
}

func (repo *PostgresAdminRepository) expireOwnerTransfers(ctx context.Context, now time.Time) error {
	rows, err := repo.db.Query(ctx, `
		WITH candidates AS (
			SELECT
				platform_owner_transfers.id,
				platform_owner_transfers.status AS previous_status,
				platform_owner_transfers.initiated_by,
				initiator.username AS initiated_by_username,
				COALESCE(NULLIF(initiator.display_name, ''), initiator.username) AS initiated_by_display_name,
				platform_owner_transfers.target_user_id,
				target.username AS target_username,
				COALESCE(NULLIF(target.display_name, ''), target.username) AS target_display_name,
				COALESCE(platform_owner_transfers.previous_owner_role, '') AS previous_owner_role,
				platform_owner_transfers.reason,
				platform_owner_transfers.created_at,
				platform_owner_transfers.updated_at AS previous_updated_at,
				platform_owner_transfers.expires_at,
				platform_owner_transfers.accepted_at,
				platform_owner_transfers.cancelled_at
			FROM platform_owner_transfers
			INNER JOIN users AS initiator ON initiator.id = platform_owner_transfers.initiated_by
			INNER JOIN users AS target ON target.id = platform_owner_transfers.target_user_id
			WHERE platform_owner_transfers.status = 'pending'
				AND platform_owner_transfers.expires_at <= $1
		),
		updated AS (
			UPDATE platform_owner_transfers
			SET status = 'expired',
				updated_at = $1
			FROM candidates
			WHERE platform_owner_transfers.id = candidates.id
				AND platform_owner_transfers.status = 'pending'
			RETURNING
				platform_owner_transfers.id::text,
				candidates.previous_status,
				candidates.initiated_by::text,
				candidates.initiated_by_username,
				candidates.initiated_by_display_name,
				candidates.target_user_id::text,
				candidates.target_username,
				candidates.target_display_name,
				candidates.previous_owner_role,
				candidates.reason,
				candidates.created_at,
				candidates.previous_updated_at,
				candidates.expires_at,
				candidates.accepted_at,
				candidates.cancelled_at,
				platform_owner_transfers.status,
				platform_owner_transfers.updated_at
		)
		SELECT *
		FROM updated
	`, now)
	if err != nil {
		return mapAdminWriteError("expire owner transfers", err)
	}
	defer rows.Close()

	logs := make([]adminusecase.AuditLog, 0)
	for rows.Next() {
		var before adminusecase.OwnerTransfer
		var afterStatus string
		var afterUpdatedAt time.Time
		var acceptedAt pgtype.Timestamptz
		var cancelledAt pgtype.Timestamptz
		if err := rows.Scan(
			&before.ID,
			&before.Status,
			&before.InitiatedByID,
			&before.InitiatedByUsername,
			&before.InitiatedByDisplayName,
			&before.TargetUserID,
			&before.TargetUsername,
			&before.TargetDisplayName,
			&before.PreviousOwnerRole,
			&before.Reason,
			&before.CreatedAt,
			&before.UpdatedAt,
			&before.ExpiresAt,
			&acceptedAt,
			&cancelledAt,
			&afterStatus,
			&afterUpdatedAt,
		); err != nil {
			return err
		}
		if acceptedAt.Valid {
			value := acceptedAt.Time
			before.AcceptedAt = &value
		}
		if cancelledAt.Valid {
			value := cancelledAt.Time
			before.CancelledAt = &value
		}
		after := before
		after.Status = afterStatus
		after.UpdatedAt = afterUpdatedAt
		logs = append(logs, adminusecase.AuditLog{
			ID:         uuid.NewString(),
			ActorID:    "system:owner-transfer-expiry",
			Action:     "admin.owner_transfer.expire",
			TargetType: "owner_transfer",
			TargetID:   after.ID,
			Before:     ownerTransferRepositoryAuditState(before),
			After:      ownerTransferRepositoryAuditState(after),
			CreatedAt:  now,
		})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate expired owner transfers: %w", err)
	}
	rows.Close()
	for _, log := range logs {
		if err := repo.CreateAuditLog(ctx, log); err != nil {
			return fmt.Errorf("create owner transfer expire audit log: %w", err)
		}
	}
	return nil
}

func ownerTransferRepositoryAuditState(transfer adminusecase.OwnerTransfer) map[string]any {
	return map[string]any{
		"id":                        transfer.ID,
		"status":                    transfer.Status,
		"initiated_by_id":           transfer.InitiatedByID,
		"initiated_by_username":     transfer.InitiatedByUsername,
		"initiated_by_display_name": transfer.InitiatedByDisplayName,
		"target_user_id":            transfer.TargetUserID,
		"target_username":           transfer.TargetUsername,
		"target_display_name":       transfer.TargetDisplayName,
		"previous_owner_role":       transfer.PreviousOwnerRole,
		"reason":                    transfer.Reason,
		"created_at":                transfer.CreatedAt,
		"updated_at":                transfer.UpdatedAt,
		"expires_at":                transfer.ExpiresAt,
		"accepted_at":               transfer.AcceptedAt,
		"cancelled_at":              transfer.CancelledAt,
	}
}

func lockOwnerTransferUsers(ctx context.Context, db postgresExecutor, firstUserID string, secondUserID string) error {
	rows, err := db.Query(ctx, `
		SELECT id
		FROM users
		WHERE id IN ($1::uuid, $2::uuid)
			OR (status = 'active' AND platform_role = 'owner')
		FOR UPDATE
	`, firstUserID, secondUserID)
	if err != nil {
		return fmt.Errorf("lock owner transfer users: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate locked owner transfer users: %w", err)
	}
	return nil
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

func escapeLikePattern(value string) string {
	return strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	).Replace(value)
}

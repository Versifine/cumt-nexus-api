package authrepository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAuthRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAuthRepository(pool *pgxpool.Pool) *PostgresAuthRepository {
	return &PostgresAuthRepository{pool: pool}
}

func (repo *PostgresAuthRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	if err := repo.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, normalizeEmail(email)).Scan(&exists); err != nil {
		return false, fmt.Errorf("check email exists: %w", err)
	}
	return exists, nil
}

func (repo *PostgresAuthRepository) FindAuthUserByEmail(ctx context.Context, email string) (authusecase.AuthUserRecord, error) {
	row := repo.pool.QueryRow(ctx, authUserSelectSQL()+` WHERE email = $1 LIMIT 1`, normalizeEmail(email))
	return scanAuthUser(row)
}

func (repo *PostgresAuthRepository) FindAuthUserByIdentifier(ctx context.Context, identifier string) (authusecase.AuthUserRecord, error) {
	normalized := strings.ToLower(strings.TrimSpace(identifier))
	if strings.Contains(normalized, "@") {
		row := repo.pool.QueryRow(ctx, authUserSelectSQL()+` WHERE email = $1 LIMIT 1`, normalized)
		return scanAuthUser(row)
	}
	row := repo.pool.QueryRow(ctx, authUserSelectSQL()+` WHERE username = $1 LIMIT 1`, normalized)
	return scanAuthUser(row)
}

func (repo *PostgresAuthRepository) CreateEmailCode(ctx context.Context, code authusecase.EmailCodeRecord) (err error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create email code: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		err = tx.Commit(ctx)
	}()

	if _, err = tx.Exec(ctx, `
		UPDATE auth_email_codes
		SET status = 'expired', updated_at = $3
		WHERE email = $1
			AND purpose = $2
			AND status = 'pending'
	`, code.Email, code.Purpose.String(), code.CreatedAt); err != nil {
		return fmt.Errorf("expire previous email codes: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO auth_email_codes (
			id,
			email,
			purpose,
			code_hash,
			status,
			attempt_count,
			sent_count,
			last_sent_at,
			expires_at,
			request_ip,
			user_agent,
			created_at,
			updated_at
		)
		VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`,
		code.ID,
		code.Email,
		code.Purpose.String(),
		code.CodeHash,
		code.Status,
		code.AttemptCount,
		code.SentCount,
		code.LastSentAt,
		code.ExpiresAt,
		code.RequestIP,
		code.UserAgent,
		code.CreatedAt,
		code.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert email code: %w", err)
	}
	return nil
}

func (repo *PostgresAuthRepository) LatestEmailCode(ctx context.Context, email string, purpose authusecase.EmailPurpose) (authusecase.EmailCodeRecord, error) {
	row := repo.pool.QueryRow(ctx, emailCodeSelectSQL()+`
		WHERE email = $1 AND purpose = $2
			AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT 1
	`, normalizeEmail(email), purpose.String())
	return scanEmailCode(row)
}

func (repo *PostgresAuthRepository) CountEmailCodesSince(ctx context.Context, email string, purpose authusecase.EmailPurpose, since time.Time) (int, error) {
	var count int
	if err := repo.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM auth_email_codes
		WHERE email = $1
			AND purpose = $2
			AND created_at >= $3
	`, normalizeEmail(email), purpose.String(), since).Scan(&count); err != nil {
		return 0, fmt.Errorf("count email codes since: %w", err)
	}
	return count, nil
}

func (repo *PostgresAuthRepository) CountEmailCodesByIPSince(ctx context.Context, requestIP string, since time.Time) (int, error) {
	if strings.TrimSpace(requestIP) == "" {
		return 0, nil
	}
	var count int
	if err := repo.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM auth_email_codes
		WHERE request_ip = $1
			AND created_at >= $2
	`, requestIP, since).Scan(&count); err != nil {
		return 0, fmt.Errorf("count email codes by ip since: %w", err)
	}
	return count, nil
}

func (repo *PostgresAuthRepository) FindPendingEmailCode(ctx context.Context, email string, purpose authusecase.EmailPurpose, now time.Time) (authusecase.EmailCodeRecord, error) {
	if _, err := repo.pool.Exec(ctx, `
		UPDATE auth_email_codes
		SET status = 'expired', updated_at = $3
		WHERE email = $1
			AND purpose = $2
			AND status = 'pending'
			AND expires_at <= $3
	`, normalizeEmail(email), purpose.String(), now); err != nil {
		return authusecase.EmailCodeRecord{}, fmt.Errorf("expire pending email code: %w", err)
	}
	row := repo.pool.QueryRow(ctx, emailCodeSelectSQL()+`
		WHERE email = $1
			AND purpose = $2
			AND status = 'pending'
			AND expires_at > $3
		ORDER BY created_at DESC
		LIMIT 1
	`, normalizeEmail(email), purpose.String(), now)
	return scanEmailCode(row)
}

func (repo *PostgresAuthRepository) MarkEmailCodeUsed(ctx context.Context, id string, now time.Time) error {
	result, err := repo.pool.Exec(ctx, `
		UPDATE auth_email_codes
		SET status = 'used',
			consumed_at = $2,
			updated_at = $2
		WHERE id = $1::uuid
	`, id, now)
	if err != nil {
		return fmt.Errorf("mark email code used: %w", err)
	}
	return ensureUpdated(result.RowsAffected(), "email code not found")
}

func (repo *PostgresAuthRepository) MarkEmailCodeAttempt(ctx context.Context, id string, attemptCount int, status string, now time.Time) error {
	result, err := repo.pool.Exec(ctx, `
		UPDATE auth_email_codes
		SET attempt_count = $2,
			status = $3,
			updated_at = $4
		WHERE id = $1::uuid
	`, id, attemptCount, status, now)
	if err != nil {
		return fmt.Errorf("mark email code attempt: %w", err)
	}
	return ensureUpdated(result.RowsAffected(), "email code not found")
}

func (repo *PostgresAuthRepository) MarkEmailCodeFailed(ctx context.Context, id string, now time.Time) error {
	result, err := repo.pool.Exec(ctx, `
		UPDATE auth_email_codes
		SET status = 'expired',
			updated_at = $2
		WHERE id = $1::uuid
	`, id, now)
	if err != nil {
		return fmt.Errorf("mark email code failed: %w", err)
	}
	return ensureUpdated(result.RowsAffected(), "email code not found")
}

func (repo *PostgresAuthRepository) CreateUserWithEmail(ctx context.Context, user userdomain.User, email string, verifiedAt time.Time, passwordUpdatedAt time.Time) error {
	_, err := repo.pool.Exec(ctx, `
		INSERT INTO users (
			id,
			username,
			password_hash,
			display_name,
			avatar_url,
			banner_url,
			headline,
			bio,
			email,
			email_verified_at,
			password_updated_at,
			status,
			created_at,
			updated_at
		)
		VALUES($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`,
		user.ID().String(),
		user.Username().String(),
		user.PasswordHash().Raw(),
		user.DisplayName().String(),
		user.AvatarURL().String(),
		user.BannerURL().String(),
		user.Headline().String(),
		user.Bio().String(),
		normalizeEmail(email),
		verifiedAt,
		passwordUpdatedAt,
		user.Status().String(),
		user.CreatedAt(),
		user.UpdatedAt(),
	)
	if err != nil {
		return mapUserWriteError(err, "create email user")
	}
	return nil
}

func (repo *PostgresAuthRepository) UpdateLastLogin(ctx context.Context, userID userdomain.UserID, loginAt time.Time, loginIP string) error {
	result, err := repo.pool.Exec(ctx, `
		UPDATE users
		SET last_login_at = $2,
			last_login_ip = $3,
			updated_at = $2
		WHERE id = $1::uuid
	`, userID.String(), loginAt, loginIP)
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return ensureUpdated(result.RowsAffected(), "user not found")
}

func (repo *PostgresAuthRepository) UpdatePasswordByEmail(ctx context.Context, email string, passwordHash userdomain.PasswordHash, updatedAt time.Time) error {
	result, err := repo.pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $2,
			password_updated_at = $3,
			tokens_revoked_after = $3,
			updated_at = $3
		WHERE email = $1
			AND status = 'active'
			AND email_verified_at IS NOT NULL
	`, normalizeEmail(email), passwordHash.Raw(), updatedAt)
	if err != nil {
		return fmt.Errorf("update password by email: %w", err)
	}
	return ensureUpdated(result.RowsAffected(), "user not found")
}

func (repo *PostgresAuthRepository) GetSecurityByUserID(ctx context.Context, userID userdomain.UserID) (authusecase.SecurityInfo, error) {
	row := repo.pool.QueryRow(ctx, `
		SELECT
			id::text,
			email,
			email_verified_at,
			password_hash,
			last_login_at,
			created_at,
			tokens_revoked_after
		FROM users
		WHERE id = $1::uuid
		LIMIT 1
	`, userID.String())
	return scanSecurityInfo(row)
}

func (repo *PostgresAuthRepository) UpdateEmailByUserID(ctx context.Context, userID userdomain.UserID, email string, verifiedAt time.Time) (authusecase.SecurityInfo, error) {
	row := repo.pool.QueryRow(ctx, `
		UPDATE users
		SET email = $2,
			email_verified_at = $3,
			updated_at = $3
		WHERE id = $1::uuid
		RETURNING
			id::text,
			email,
			email_verified_at,
			password_hash,
			last_login_at,
			created_at,
			tokens_revoked_after
	`, userID.String(), normalizeEmail(email), verifiedAt)
	info, err := scanSecurityInfo(row)
	if err != nil {
		return authusecase.SecurityInfo{}, mapUserWriteError(err, "change email")
	}
	return info, nil
}

func (repo *PostgresAuthRepository) UpdatePasswordByUserID(ctx context.Context, userID userdomain.UserID, passwordHash userdomain.PasswordHash, updatedAt time.Time) error {
	result, err := repo.pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $2,
			password_updated_at = $3,
			tokens_revoked_after = $3,
			updated_at = $3
		WHERE id = $1::uuid
	`, userID.String(), passwordHash.Raw(), updatedAt)
	if err != nil {
		return fmt.Errorf("update password by user id: %w", err)
	}
	return ensureUpdated(result.RowsAffected(), "user not found")
}

func (repo *PostgresAuthRepository) RevokeTokensByUserID(ctx context.Context, userID userdomain.UserID, revokedAfter time.Time) error {
	result, err := repo.pool.Exec(ctx, `
		UPDATE users
		SET tokens_revoked_after = $2,
			updated_at = $2
		WHERE id = $1::uuid
	`, userID.String(), revokedAfter)
	if err != nil {
		return fmt.Errorf("revoke tokens: %w", err)
	}
	return ensureUpdated(result.RowsAffected(), "user not found")
}

func (repo *PostgresAuthRepository) DeleteAccountByUserID(ctx context.Context, userID userdomain.UserID, deletedAt time.Time) error {
	result, err := repo.pool.Exec(ctx, `
		UPDATE users
		SET status = 'deleted',
			username = 'deleted_' || substring(replace(id::text, '-', ''), 1, 24),
			email = '',
			email_verified_at = NULL,
			display_name = '',
			avatar_url = '',
			banner_url = '',
			headline = '',
			bio = '',
			is_platform_staff = false,
			tokens_revoked_after = $2,
			deleted_at = $2,
			updated_at = $2
		WHERE id = $1::uuid
			AND status = 'active'
	`, userID.String(), deletedAt)
	if err != nil {
		return fmt.Errorf("delete account by user id: %w", err)
	}
	return ensureUpdated(result.RowsAffected(), "user not found")
}

func (repo *PostgresAuthRepository) ValidateAccessToken(ctx context.Context, userID userdomain.UserID, issuedAt time.Time) error {
	var status string
	var revokedAfter pgtype.Timestamptz
	if err := repo.pool.QueryRow(ctx, `
		SELECT status, tokens_revoked_after
		FROM users
		WHERE id = $1::uuid
		LIMIT 1
	`, userID.String()).Scan(&status, &revokedAfter); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.New(apperr.CodeUnauthenticated, "invalid token")
		}
		return fmt.Errorf("validate access token user: %w", err)
	}
	if status != "active" {
		return apperr.New(apperr.CodeUnauthenticated, "invalid token")
	}
	if revokedAfter.Valid && !issuedAt.After(revokedAfter.Time) {
		return apperr.New(apperr.CodeUnauthenticated, "invalid token")
	}
	banned, err := repo.HasActiveAccountBan(ctx, userID, time.Now().UTC())
	if err != nil {
		return err
	}
	if banned {
		return apperr.New(apperr.CodeForbidden, "user is forbidden")
	}
	return nil
}

func (repo *PostgresAuthRepository) HasActiveAccountBan(ctx context.Context, userID userdomain.UserID, now time.Time) (bool, error) {
	var exists bool
	if err := repo.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_sanctions
			WHERE user_id = $1::uuid
				AND type = 'account_ban'
				AND status = 'active'
				AND starts_at <= $2
				AND (expires_at IS NULL OR expires_at > $2)
		)
	`, userID.String(), now).Scan(&exists); err != nil {
		return false, fmt.Errorf("check active account ban: %w", err)
	}
	return exists, nil
}

func (repo *PostgresAuthRepository) RecordSecurityEvent(ctx context.Context, event authusecase.SecurityEvent) error {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal security event metadata: %w", err)
	}
	var rawUserID any
	if event.UserID != nil {
		rawUserID = event.UserID.String()
	}
	_, err = repo.pool.Exec(ctx, `
		INSERT INTO auth_security_events (
			id,
			user_id,
			email,
			action,
			ip,
			user_agent,
			metadata,
			created_at
		)
		VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6,$7::jsonb,$8)
	`, event.ID, rawUserID, normalizeEmail(event.Email), event.Action, event.IP, event.UserAgent, string(metadata), event.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert security event: %w", err)
	}
	return nil
}

func (repo *PostgresAuthRepository) CountSecurityEventsSince(ctx context.Context, action string, identity string, requestIP string, since time.Time) (int, error) {
	if strings.TrimSpace(action) == "" {
		return 0, nil
	}
	var count int
	if err := repo.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM auth_security_events
		WHERE action = $1
			AND created_at >= $2
			AND ($3 = '' OR email = $3)
			AND ($4 = '' OR ip = $4)
	`, action, since, normalizeEmail(identity), strings.TrimSpace(requestIP)).Scan(&count); err != nil {
		return 0, fmt.Errorf("count security events since: %w", err)
	}
	return count, nil
}

func authUserSelectSQL() string {
	return `
		SELECT
			id::text,
			username,
			password_hash,
			display_name,
			avatar_url,
			banner_url,
			headline,
			bio,
			status,
			created_at,
			updated_at,
			email,
			email_verified_at,
			last_login_at,
			password_updated_at,
			tokens_revoked_after,
			is_platform_staff,
			COALESCE(platform_role, '')
		FROM users
	`
}

func scanAuthUser(row pgx.Row) (authusecase.AuthUserRecord, error) {
	var rawID string
	var rawUsername string
	var rawPasswordHash string
	var rawDisplayName string
	var rawAvatarURL string
	var rawBannerURL string
	var rawHeadline string
	var rawBio string
	var rawStatus string
	var createdAt time.Time
	var updatedAt time.Time
	var email string
	var emailVerifiedAt pgtype.Timestamptz
	var lastLoginAt pgtype.Timestamptz
	var passwordUpdatedAt pgtype.Timestamptz
	var tokensRevokedAfter pgtype.Timestamptz
	var isPlatformStaff bool
	var platformRole string

	if err := row.Scan(
		&rawID,
		&rawUsername,
		&rawPasswordHash,
		&rawDisplayName,
		&rawAvatarURL,
		&rawBannerURL,
		&rawHeadline,
		&rawBio,
		&rawStatus,
		&createdAt,
		&updatedAt,
		&email,
		&emailVerifiedAt,
		&lastLoginAt,
		&passwordUpdatedAt,
		&tokensRevokedAfter,
		&isPlatformStaff,
		&platformRole,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return authusecase.AuthUserRecord{}, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return authusecase.AuthUserRecord{}, fmt.Errorf("scan auth user: %w", err)
	}

	user, err := rehydrateUser(rawID, rawUsername, rawPasswordHash, rawDisplayName, rawAvatarURL, rawBannerURL, rawHeadline, rawBio, rawStatus, createdAt, updatedAt)
	if err != nil {
		return authusecase.AuthUserRecord{}, err
	}
	return authusecase.AuthUserRecord{
		User:               user,
		Email:              email,
		EmailVerifiedAt:    nullableTime(emailVerifiedAt),
		LastLoginAt:        nullableTime(lastLoginAt),
		PasswordUpdatedAt:  nullableTime(passwordUpdatedAt),
		TokensRevokedAfter: nullableTime(tokensRevokedAfter),
		IsPlatformStaff:    isPlatformStaff,
		PlatformRole:       platformRole,
	}, nil
}

func emailCodeSelectSQL() string {
	return `
		SELECT
			id::text,
			email,
			purpose,
			code_hash,
			status,
			attempt_count,
			sent_count,
			last_sent_at,
			expires_at,
			request_ip,
			user_agent,
			created_at,
			updated_at
		FROM auth_email_codes
	`
}

func scanEmailCode(row pgx.Row) (authusecase.EmailCodeRecord, error) {
	var record authusecase.EmailCodeRecord
	var purpose string
	if err := row.Scan(
		&record.ID,
		&record.Email,
		&purpose,
		&record.CodeHash,
		&record.Status,
		&record.AttemptCount,
		&record.SentCount,
		&record.LastSentAt,
		&record.ExpiresAt,
		&record.RequestIP,
		&record.UserAgent,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return authusecase.EmailCodeRecord{}, apperr.New(apperr.CodeNotFound, "email code not found")
		}
		return authusecase.EmailCodeRecord{}, fmt.Errorf("scan email code: %w", err)
	}
	record.Purpose = authusecase.EmailPurpose(purpose)
	return record, nil
}

func scanSecurityInfo(row pgx.Row) (authusecase.SecurityInfo, error) {
	var rawID string
	var email string
	var emailVerifiedAt pgtype.Timestamptz
	var rawPasswordHash string
	var lastLoginAt pgtype.Timestamptz
	var createdAt time.Time
	var tokensRevokedAfter pgtype.Timestamptz
	if err := row.Scan(&rawID, &email, &emailVerifiedAt, &rawPasswordHash, &lastLoginAt, &createdAt, &tokensRevokedAfter); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return authusecase.SecurityInfo{}, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return authusecase.SecurityInfo{}, fmt.Errorf("scan security info: %w", err)
	}
	userID, err := userdomain.NewUserID(rawID)
	if err != nil {
		return authusecase.SecurityInfo{}, fmt.Errorf("rehydrate security user id: %w", err)
	}
	passwordHash, err := userdomain.NewPasswordHash(rawPasswordHash)
	if err != nil {
		return authusecase.SecurityInfo{}, fmt.Errorf("rehydrate security password hash: %w", err)
	}
	return authusecase.SecurityInfo{
		UserID:             userID,
		Email:              email,
		EmailVerifiedAt:    nullableTime(emailVerifiedAt),
		PasswordHash:       passwordHash,
		LastLoginAt:        nullableTime(lastLoginAt),
		CreatedAt:          createdAt,
		TokensRevokedAfter: nullableTime(tokensRevokedAfter),
	}, nil
}

func rehydrateUser(rawID string, rawUsername string, rawPasswordHash string, rawDisplayName string, rawAvatarURL string, rawBannerURL string, rawHeadline string, rawBio string, rawStatus string, createdAt time.Time, updatedAt time.Time) (*userdomain.User, error) {
	userID, err := userdomain.NewUserID(rawID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate user id: %w", err)
	}
	username, err := userdomain.NewUsername(rawUsername)
	if err != nil {
		return nil, fmt.Errorf("rehydrate username: %w", err)
	}
	passwordHash, err := userdomain.NewPasswordHash(rawPasswordHash)
	if err != nil {
		return nil, fmt.Errorf("rehydrate password hash: %w", err)
	}
	displayName, err := userdomain.NewDisplayName(rawDisplayName)
	if err != nil {
		return nil, fmt.Errorf("rehydrate display name: %w", err)
	}
	avatarURL, err := userdomain.NewAvatarURL(rawAvatarURL)
	if err != nil {
		return nil, fmt.Errorf("rehydrate avatar url: %w", err)
	}
	bannerURL, err := userdomain.NewBannerURL(rawBannerURL)
	if err != nil {
		return nil, fmt.Errorf("rehydrate banner url: %w", err)
	}
	headline, err := userdomain.NewHeadline(rawHeadline)
	if err != nil {
		return nil, fmt.Errorf("rehydrate headline: %w", err)
	}
	bio, err := userdomain.NewBio(rawBio)
	if err != nil {
		return nil, fmt.Errorf("rehydrate bio: %w", err)
	}
	status, err := userdomain.NewUserStatus(rawStatus)
	if err != nil {
		return nil, fmt.Errorf("rehydrate status: %w", err)
	}
	return userdomain.RehydrateUserWithProfile(userID, username, passwordHash, displayName, avatarURL, bannerURL, headline, bio, status, createdAt, updatedAt)
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ensureUpdated(rows int64, message string) error {
	if rows == 0 {
		return apperr.New(apperr.CodeNotFound, message)
	}
	return nil
}

func mapUserWriteError(err error, fallback string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_username_uq":
				return apperr.New(apperr.CodeConflict, "username already exists")
			case "users_email_lower_uq":
				return apperr.New(apperr.CodeConflict, "email already exists")
			}
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.New(apperr.CodeNotFound, "user not found")
	}
	return fmt.Errorf("%s: %w", fallback, err)
}

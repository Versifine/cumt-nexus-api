package userrepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
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
func (ur *PostgresUserRepository) Create(ctx context.Context, user userdomain.User) error {
	const query = `
		INSERT INTO users (
			id,
			username,
			password_hash,
			display_name,
			avatar_url,
			banner_url,
			headline,
			bio,
			status,
			created_at,
			updated_at
		)
		VALUES($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`
	_, err := ur.pool.Exec(
		ctx,
		query,
		user.ID().String(),
		user.Username().String(),
		user.PasswordHash().Raw(),
		user.DisplayName().String(),
		user.AvatarURL().String(),
		user.BannerURL().String(),
		user.Headline().String(),
		user.Bio().String(),
		user.Status().String(),
		user.CreatedAt(),
		user.UpdatedAt(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "users_username_uq" {
				return apperr.New(apperr.CodeConflict, "username already exists")
			}
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}
func (ur *PostgresUserRepository) FindByID(ctx context.Context, id userdomain.UserID) (*userdomain.User, error) {
	const query = `
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
			updated_at
		FROM users
		WHERE id = $1::uuid
		LIMIT 1
	`
	row := ur.pool.QueryRow(ctx, query, id.String())

	return scanUser(row)
}
func (ur *PostgresUserRepository) FindByUsername(ctx context.Context, username userdomain.Username) (*userdomain.User, error) {
	const query = `
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
			updated_at
		FROM users
		WHERE username = $1
		LIMIT 1
	`
	row := ur.pool.QueryRow(ctx, query, username.String())

	return scanUser(row)
}

func (ur *PostgresUserRepository) CountVisiblePostsByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID) (int, error) {
	const query = `
		SELECT COUNT(*)::int
		FROM posts
		INNER JOIN communities ON communities.id = posts.community_id
		WHERE posts.author_id = $1::uuid
			AND posts.status = 'visible'
			AND communities.status = 'active'
			AND communities.visibility = 'public'
	`

	var count int
	if err := ur.pool.QueryRow(ctx, query, authorID.String()).Scan(&count); err != nil {
		return 0, fmt.Errorf("count visible public posts by author: %w", err)
	}
	return count, nil
}

func (ur *PostgresUserRepository) CountVisibleCommentsByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID) (int, error) {
	const query = `
		SELECT COUNT(*)::int
		FROM comments
		INNER JOIN posts ON posts.id = comments.post_id
		INNER JOIN communities ON communities.id = posts.community_id
		WHERE comments.author_id = $1::uuid
			AND comments.status = 'visible'
			AND posts.status = 'visible'
			AND communities.status = 'active'
			AND communities.visibility = 'public'
	`

	var count int
	if err := ur.pool.QueryRow(ctx, query, authorID.String()).Scan(&count); err != nil {
		return 0, fmt.Errorf("count visible public comments by author: %w", err)
	}
	return count, nil
}

func (ur *PostgresUserRepository) UpdateProfile(ctx context.Context, user userdomain.User) error {
	const query = `
		UPDATE users
		SET
			display_name = $2,
			avatar_url = $3,
			banner_url = $4,
			headline = $5,
			bio = $6,
			updated_at = $7
		WHERE id = $1::uuid
	`

	result, err := ur.pool.Exec(
		ctx,
		query,
		user.ID().String(),
		user.DisplayName().String(),
		user.AvatarURL().String(),
		user.BannerURL().String(),
		user.Headline().String(),
		user.Bio().String(),
		user.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("update user profile: %w", err)
	}
	if result.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "user not found")
	}
	return nil
}

func scanUser(row pgx.Row) (*userdomain.User, error) {
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
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return nil, fmt.Errorf("scan user row: %w", err)
	}

	userID, err := userdomain.NewUserID(rawID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate user id: %v", err)
	}

	username, err := userdomain.NewUsername(rawUsername)
	if err != nil {
		return nil, fmt.Errorf("rehydrate username: %v", err)
	}

	passwordHash, err := userdomain.NewPasswordHash(rawPasswordHash)
	if err != nil {
		return nil, fmt.Errorf("rehydrate password hash: %v", err)
	}

	displayName, err := userdomain.NewDisplayName(rawDisplayName)
	if err != nil {
		return nil, fmt.Errorf("rehydrate display name: %v", err)
	}

	avatarURL, err := userdomain.NewAvatarURL(rawAvatarURL)
	if err != nil {
		return nil, fmt.Errorf("rehydrate avatar url: %v", err)
	}

	bannerURL, err := userdomain.NewBannerURL(rawBannerURL)
	if err != nil {
		return nil, fmt.Errorf("rehydrate banner url: %v", err)
	}

	headline, err := userdomain.NewHeadline(rawHeadline)
	if err != nil {
		return nil, fmt.Errorf("rehydrate headline: %v", err)
	}

	bio, err := userdomain.NewBio(rawBio)
	if err != nil {
		return nil, fmt.Errorf("rehydrate bio: %v", err)
	}

	status, err := userdomain.NewUserStatus(rawStatus)
	if err != nil {
		return nil, fmt.Errorf("rehydrate user status: %v", err)
	}

	user, err := userdomain.RehydrateUserWithProfile(
		userID,
		username,
		passwordHash,
		displayName,
		avatarURL,
		bannerURL,
		headline,
		bio,
		status,
		createdAt,
		updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("rehydrate user: %v", err)
	}

	return user, nil
}

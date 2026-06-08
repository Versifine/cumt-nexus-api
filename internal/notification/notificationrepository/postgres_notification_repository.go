package notificationrepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/notification/notificationusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ notificationusecase.Repository = (*PostgresNotificationRepository)(nil)

type PostgresNotificationRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresNotificationRepository(pool *pgxpool.Pool) *PostgresNotificationRepository {
	return &PostgresNotificationRepository{
		pool: pool,
	}
}

func (repo *PostgresNotificationRepository) ListByRecipient(ctx context.Context, recipientID userdomain.UserID, category notificationusecase.CategoryFilter, status notificationusecase.StatusFilter, limit int, offset int) ([]notificationusecase.Notification, error) {
	query := `
		SELECT
			id::text,
			recipient_id::text,
			type,
			title,
			body,
			source_type,
			source_id,
			read_at,
			created_at,
			updated_at
		FROM notifications
		WHERE recipient_id = $1::uuid
	`
	args := []any{recipientID.String()}
	query += notificationCategoryPredicate(category)
	switch status {
	case notificationusecase.StatusFilterUnread:
		query += ` AND read_at IS NULL`
	case notificationusecase.StatusFilterRead:
		query += ` AND read_at IS NOT NULL`
	case notificationusecase.StatusFilterAll:
	default:
		return nil, apperr.New(apperr.CodeInvalidArgument, "notification status is invalid")
	}
	query += `
		ORDER BY created_at DESC, id DESC
		LIMIT $2
		OFFSET $3
	`
	args = append(args, limit, offset)

	rows, err := repo.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	notifications := make([]notificationusecase.Notification, 0)
	for rows.Next() {
		notification, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}
	return notifications, nil
}

func (repo *PostgresNotificationRepository) CountUnreadByCategory(ctx context.Context, recipientID userdomain.UserID) (notificationusecase.UnreadSummary, error) {
	const query = `
		SELECT
			COUNT(*) FILTER (WHERE read_at IS NULL),
			COUNT(*) FILTER (WHERE read_at IS NULL AND type IN ('reply', 'comment_reply', 'post_reply')),
			COUNT(*) FILTER (WHERE read_at IS NULL AND type = 'mention'),
			COUNT(*) FILTER (WHERE read_at IS NULL AND type IN ('like', 'post_like', 'comment_like', 'post_upvote', 'comment_upvote')),
			COUNT(*) FILTER (WHERE read_at IS NULL AND type = 'system')
		FROM notifications
		WHERE recipient_id = $1::uuid
	`

	var total int64
	var replies int64
	var mentions int64
	var likes int64
	var system int64
	if err := repo.pool.QueryRow(ctx, query, recipientID.String()).Scan(
		&total,
		&replies,
		&mentions,
		&likes,
		&system,
	); err != nil {
		return notificationusecase.UnreadSummary{}, fmt.Errorf("count unread notifications: %w", err)
	}
	return notificationusecase.UnreadSummary{
		Total:    int(total),
		Replies:  int(replies),
		Mentions: int(mentions),
		Likes:    int(likes),
		System:   int(system),
	}, nil
}

func (repo *PostgresNotificationRepository) MarkRead(ctx context.Context, id string, recipientID userdomain.UserID, readAt time.Time) (notificationusecase.Notification, error) {
	const query = `
		UPDATE notifications
		SET read_at = COALESCE(read_at, $3),
			updated_at = CASE WHEN read_at IS NULL THEN $3 ELSE updated_at END
		WHERE id = $1::uuid
			AND recipient_id = $2::uuid
		RETURNING
			id::text,
			recipient_id::text,
			type,
			title,
			body,
			source_type,
			source_id,
			read_at,
			created_at,
			updated_at
	`

	notification, err := scanNotification(repo.pool.QueryRow(ctx, query, id, recipientID.String(), readAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notificationusecase.Notification{}, apperr.New(apperr.CodeNotFound, "notification not found")
		}
		return notificationusecase.Notification{}, err
	}
	return notification, nil
}

func (repo *PostgresNotificationRepository) MarkAllRead(ctx context.Context, recipientID userdomain.UserID, readAt time.Time) (int, error) {
	const query = `
		UPDATE notifications
		SET read_at = $2,
			updated_at = $2
		WHERE recipient_id = $1::uuid
			AND read_at IS NULL
	`

	commandTag, err := repo.pool.Exec(ctx, query, recipientID.String(), readAt)
	if err != nil {
		return 0, fmt.Errorf("mark all notifications read: %w", err)
	}
	return int(commandTag.RowsAffected()), nil
}

func notificationCategoryPredicate(category notificationusecase.CategoryFilter) string {
	switch category {
	case notificationusecase.CategoryFilterAll:
		return ""
	case notificationusecase.CategoryFilterReplies:
		return ` AND type IN ('reply', 'comment_reply', 'post_reply')`
	case notificationusecase.CategoryFilterMentions:
		return ` AND type = 'mention'`
	case notificationusecase.CategoryFilterLikes:
		return ` AND type IN ('like', 'post_like', 'comment_like', 'post_upvote', 'comment_upvote')`
	case notificationusecase.CategoryFilterSystem:
		return ` AND type = 'system'`
	default:
		return ` AND false`
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNotification(row rowScanner) (notificationusecase.Notification, error) {
	var notification notificationusecase.Notification
	var readAt pgtype.Timestamptz
	if err := row.Scan(
		&notification.ID,
		&notification.RecipientID,
		&notification.Type,
		&notification.Title,
		&notification.Body,
		&notification.SourceType,
		&notification.SourceID,
		&readAt,
		&notification.CreatedAt,
		&notification.UpdatedAt,
	); err != nil {
		return notificationusecase.Notification{}, err
	}
	if readAt.Valid {
		value := readAt.Time
		notification.ReadAt = &value
	}
	return notification, nil
}

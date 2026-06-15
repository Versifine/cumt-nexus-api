package notificationrepository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/notification/notificationusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (repo *PostgresNotificationRepository) Create(ctx context.Context, notification notificationusecase.Notification) error {
	const query = `
		INSERT INTO notifications (
			id,
			recipient_id,
			type,
			title,
			body,
			source_type,
			source_id,
			aggregate_key,
			aggregate_count,
			last_actor_id,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10::uuid, $11, $12)
	`

	_, err := repo.pool.Exec(
		ctx,
		query,
		notification.ID,
		notification.RecipientID,
		notification.Type,
		notification.Title,
		notification.Body,
		notification.SourceType,
		notification.SourceID,
		notification.AggregateKey,
		normalizeAggregateCount(notification.AggregateCount),
		nullableUUID(notification.LastActorID),
		notification.CreatedAt,
		notification.UpdatedAt,
	)
	if err != nil {
		return mapPostgresNotificationWriteError("create notification", err)
	}
	return nil
}

func (repo *PostgresNotificationRepository) UpsertAggregated(ctx context.Context, notification notificationusecase.Notification) error {
	const query = `
		INSERT INTO notifications (
			id,
			recipient_id,
			type,
			title,
			body,
			source_type,
			source_id,
			aggregate_key,
			aggregate_count,
			last_actor_id,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10::uuid, $11, $12)
		ON CONFLICT (recipient_id, type, aggregate_key)
			WHERE read_at IS NULL
				AND aggregate_key <> ''
		DO UPDATE
		SET
			aggregate_count = notifications.aggregate_count + 1,
			last_actor_id = EXCLUDED.last_actor_id,
			updated_at = EXCLUDED.updated_at
	`

	if strings.TrimSpace(notification.AggregateKey) == "" {
		return repo.Create(ctx, notification)
	}
	_, err := repo.pool.Exec(
		ctx,
		query,
		notification.ID,
		notification.RecipientID,
		notification.Type,
		notification.Title,
		notification.Body,
		notification.SourceType,
		notification.SourceID,
		notification.AggregateKey,
		normalizeAggregateCount(notification.AggregateCount),
		nullableUUID(notification.LastActorID),
		notification.CreatedAt,
		notification.UpdatedAt,
	)
	if err != nil {
		return mapPostgresNotificationWriteError("upsert aggregated notification", err)
	}
	return nil
}

func (repo *PostgresNotificationRepository) ListByRecipient(ctx context.Context, recipientID userdomain.UserID, category notificationusecase.CategoryFilter, status notificationusecase.StatusFilter, limit int, offset int) ([]notificationusecase.Notification, error) {
	query := `
		SELECT
			notifications.id::text,
			notifications.recipient_id::text,
			notifications.type,
			notifications.title,
			notifications.body,
			notifications.source_type,
			notifications.source_id,
			notifications.aggregate_key,
			notifications.aggregate_count,
			notifications.last_actor_id::text,
			last_actor.username,
			last_actor.display_name,
			last_actor.avatar_url,
			source_post.id::text,
			source_comment.id::text,
			source_post.title,
			source_comment.body,
			CASE
				WHEN source_comment.id IS NULL THEN 0
				WHEN source_comment.parent_id IS NULL THEN 0
				ELSE 1
			END,
			source_community.id::text,
			source_community.slug,
			source_community.name,
			notifications.read_at,
			notifications.created_at,
			notifications.updated_at
		FROM notifications
		LEFT JOIN users AS last_actor
			ON last_actor.id = notifications.last_actor_id
		LEFT JOIN comments AS source_comment
			ON notifications.source_type = 'comment'
			AND source_comment.id::text = notifications.source_id
		LEFT JOIN posts AS source_post
			ON (
				notifications.source_type = 'post'
				AND source_post.id::text = notifications.source_id
			)
			OR source_post.id = source_comment.post_id
		LEFT JOIN communities AS source_community
			ON source_community.id = source_post.community_id
		WHERE notifications.recipient_id = $1::uuid
	`
	args := []any{recipientID.String()}
	query += notificationCategoryPredicate(category)
	switch status {
	case notificationusecase.StatusFilterUnread:
		query += ` AND notifications.read_at IS NULL`
	case notificationusecase.StatusFilterRead:
		query += ` AND notifications.read_at IS NOT NULL`
	case notificationusecase.StatusFilterAll:
	default:
		return nil, apperr.New(apperr.CodeInvalidArgument, "notification status is invalid")
	}
	query += `
		ORDER BY notifications.created_at DESC, notifications.id DESC
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
		WITH updated_notification AS (
			UPDATE notifications
			SET read_at = COALESCE(read_at, $3),
				updated_at = CASE WHEN read_at IS NULL THEN $3 ELSE updated_at END
			WHERE id = $1::uuid
				AND recipient_id = $2::uuid
			RETURNING *
		)
		SELECT
			updated_notification.id::text,
			updated_notification.recipient_id::text,
			updated_notification.type,
			updated_notification.title,
			updated_notification.body,
			updated_notification.source_type,
			updated_notification.source_id,
			updated_notification.aggregate_key,
			updated_notification.aggregate_count,
			updated_notification.last_actor_id::text,
			last_actor.username,
			last_actor.display_name,
			last_actor.avatar_url,
			source_post.id::text,
			source_comment.id::text,
			source_post.title,
			source_comment.body,
			CASE
				WHEN source_comment.id IS NULL THEN 0
				WHEN source_comment.parent_id IS NULL THEN 0
				ELSE 1
			END,
			source_community.id::text,
			source_community.slug,
			source_community.name,
			updated_notification.read_at,
			updated_notification.created_at,
			updated_notification.updated_at
		FROM updated_notification
		LEFT JOIN users AS last_actor
			ON last_actor.id = updated_notification.last_actor_id
		LEFT JOIN comments AS source_comment
			ON updated_notification.source_type = 'comment'
			AND source_comment.id::text = updated_notification.source_id
		LEFT JOIN posts AS source_post
			ON (
				updated_notification.source_type = 'post'
				AND source_post.id::text = updated_notification.source_id
			)
			OR source_post.id = source_comment.post_id
		LEFT JOIN communities AS source_community
			ON source_community.id = source_post.community_id
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
	case notificationusecase.CategoryFilterInteractions:
		return ` AND notifications.type IN ('reply', 'comment_reply', 'post_reply', 'mention', 'like', 'post_like', 'comment_like', 'post_upvote', 'comment_upvote')`
	case notificationusecase.CategoryFilterReplies:
		return ` AND notifications.type IN ('reply', 'comment_reply', 'post_reply')`
	case notificationusecase.CategoryFilterMentions:
		return ` AND notifications.type = 'mention'`
	case notificationusecase.CategoryFilterLikes:
		return ` AND notifications.type IN ('like', 'post_like', 'comment_like', 'post_upvote', 'comment_upvote')`
	case notificationusecase.CategoryFilterSystem:
		return ` AND notifications.type = 'system'`
	default:
		return ` AND false`
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNotification(row rowScanner) (notificationusecase.Notification, error) {
	var notification notificationusecase.Notification
	var lastActorID pgtype.Text
	var actorUsername pgtype.Text
	var actorDisplayName pgtype.Text
	var actorAvatarURL pgtype.Text
	var postID pgtype.Text
	var commentID pgtype.Text
	var postTitle pgtype.Text
	var commentBody pgtype.Text
	var commentDepth int
	var communityID pgtype.Text
	var communitySlug pgtype.Text
	var communityName pgtype.Text
	var readAt pgtype.Timestamptz
	if err := row.Scan(
		&notification.ID,
		&notification.RecipientID,
		&notification.Type,
		&notification.Title,
		&notification.Body,
		&notification.SourceType,
		&notification.SourceID,
		&notification.AggregateKey,
		&notification.AggregateCount,
		&lastActorID,
		&actorUsername,
		&actorDisplayName,
		&actorAvatarURL,
		&postID,
		&commentID,
		&postTitle,
		&commentBody,
		&commentDepth,
		&communityID,
		&communitySlug,
		&communityName,
		&readAt,
		&notification.CreatedAt,
		&notification.UpdatedAt,
	); err != nil {
		return notificationusecase.Notification{}, err
	}
	if lastActorID.Valid {
		notification.LastActorID = lastActorID.String
	}
	if lastActorID.Valid {
		actor := &notificationusecase.NotificationActor{
			ID:          lastActorID.String,
			Username:    actorUsername.String,
			DisplayName: actorDisplayName.String,
			AvatarURL:   actorAvatarURL.String,
		}
		notification.LastActor = actor
		notification.Actor = actor
	}
	if postID.Valid {
		notification.Context.PostID = postID.String
	}
	if commentID.Valid {
		notification.Context.CommentID = commentID.String
	}
	if postTitle.Valid {
		notification.Context.PostTitle = postTitle.String
	}
	if commentBody.Valid {
		notification.Context.CommentExcerpt = truncateRunes(strings.TrimSpace(commentBody.String), 120)
		notification.Context.CommentDepth = commentDepth
	}
	if communityID.Valid || communitySlug.Valid || communityName.Valid {
		notification.Context.Community = &notificationusecase.NotificationCommunityContext{
			ID:   communityID.String,
			Slug: communitySlug.String,
			Name: communityName.String,
		}
	}
	notification.Context.Permalink = notificationPermalink(notification.Context)
	if readAt.Valid {
		value := readAt.Time
		notification.ReadAt = &value
	}
	return notification, nil
}

func notificationPermalink(context notificationusecase.NotificationContext) string {
	if strings.TrimSpace(context.PostID) == "" {
		return ""
	}
	path := "/posts/" + strings.TrimSpace(context.PostID)
	if context.Community != nil && strings.TrimSpace(context.Community.Slug) != "" {
		path = "/c/" + strings.TrimSpace(context.Community.Slug) + path
	}
	if strings.TrimSpace(context.CommentID) != "" {
		path += "#comment-" + strings.TrimSpace(context.CommentID)
	}
	return path
}

func truncateRunes(value string, maxRunes int) string {
	if maxRunes <= 0 || utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxRunes])
}

func normalizeAggregateCount(count int) int {
	if count <= 0 {
		return 1
	}
	return count
}

func nullableUUID(raw string) any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return strings.TrimSpace(raw)
}

func mapPostgresNotificationWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23503" {
			return apperr.New(apperr.CodeNotFound, "related user not found")
		}
		if pgErr.Code == "23505" {
			return apperr.New(apperr.CodeConflict, "notification already exists")
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

package messagerepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/message/messageusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresMessageRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresMessageRepository(pool *pgxpool.Pool) *PostgresMessageRepository {
	return &PostgresMessageRepository{pool: pool}
}

func (repo *PostgresMessageRepository) GetPrivacySettings(ctx context.Context, userID userdomain.UserID) (messageusecase.PrivacySettingsRecord, error) {
	const query = `
		SELECT allow_messages, online_status_enabled, updated_at
		FROM message_privacy_settings
		WHERE user_id = $1::uuid
	`
	var record messageusecase.PrivacySettingsRecord
	if err := repo.pool.QueryRow(ctx, query, userID.String()).Scan(&record.AllowMessages, &record.OnlineStatusEnabled, &record.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return messageusecase.PrivacySettingsRecord{AllowMessages: messageusecase.PrivacyEveryone, OnlineStatusEnabled: false, UpdatedAt: time.Time{}}, nil
		}
		return messageusecase.PrivacySettingsRecord{}, fmt.Errorf("get message privacy settings: %w", err)
	}
	return record, nil
}

func (repo *PostgresMessageRepository) UpsertPrivacySettings(ctx context.Context, userID userdomain.UserID, allowMessages string, onlineStatusEnabled bool, now time.Time) (messageusecase.PrivacySettingsRecord, error) {
	const query = `
		INSERT INTO message_privacy_settings (user_id, allow_messages, online_status_enabled, updated_at)
		VALUES ($1::uuid, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		SET allow_messages = EXCLUDED.allow_messages,
			online_status_enabled = EXCLUDED.online_status_enabled,
			updated_at = EXCLUDED.updated_at
		RETURNING allow_messages, online_status_enabled, updated_at
	`
	var record messageusecase.PrivacySettingsRecord
	if err := repo.pool.QueryRow(ctx, query, userID.String(), allowMessages, onlineStatusEnabled, now).Scan(&record.AllowMessages, &record.OnlineStatusEnabled, &record.UpdatedAt); err != nil {
		return messageusecase.PrivacySettingsRecord{}, fmt.Errorf("upsert message privacy settings: %w", err)
	}
	return record, nil
}

func (repo *PostgresMessageRepository) IsBlockedEither(ctx context.Context, a userdomain.UserID, b userdomain.UserID) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM user_blocks
			WHERE (blocker_id = $1::uuid AND blocked_id = $2::uuid)
				OR (blocker_id = $2::uuid AND blocked_id = $1::uuid)
		)
	`
	var blocked bool
	if err := repo.pool.QueryRow(ctx, query, a.String(), b.String()).Scan(&blocked); err != nil {
		return false, fmt.Errorf("check message block state: %w", err)
	}
	return blocked, nil
}

func (repo *PostgresMessageRepository) BlockUser(ctx context.Context, blockerID userdomain.UserID, blockedID userdomain.UserID, now time.Time) error {
	const query = `
		INSERT INTO user_blocks (blocker_id, blocked_id, created_at)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (blocker_id, blocked_id) DO NOTHING
	`
	if _, err := repo.pool.Exec(ctx, query, blockerID.String(), blockedID.String(), now); err != nil {
		return fmt.Errorf("block message user: %w", err)
	}
	return nil
}

func (repo *PostgresMessageRepository) UnblockUser(ctx context.Context, blockerID userdomain.UserID, blockedID userdomain.UserID) error {
	const query = `DELETE FROM user_blocks WHERE blocker_id = $1::uuid AND blocked_id = $2::uuid`
	if _, err := repo.pool.Exec(ctx, query, blockerID.String(), blockedID.String()); err != nil {
		return fmt.Errorf("unblock message user: %w", err)
	}
	return nil
}

func (repo *PostgresMessageRepository) FindDirectConversation(ctx context.Context, userID userdomain.UserID, peerID userdomain.UserID) (messageusecase.ConversationRecord, error) {
	const query = conversationSelect + `
		WHERE p.user_id = $1::uuid
			AND p.peer_user_id = $2::uuid
			AND p.deleted_at IS NULL
		LIMIT 1
	`
	return scanConversation(repo.pool.QueryRow(ctx, query, userID.String(), peerID.String()))
}

func (repo *PostgresMessageRepository) CountRecentRequests(ctx context.Context, fromUserID userdomain.UserID, since time.Time) (int, error) {
	const query = `
		SELECT COUNT(*)::int
		FROM message_requests
		WHERE from_user_id = $1::uuid
			AND created_at >= $2
	`
	var count int
	if err := repo.pool.QueryRow(ctx, query, fromUserID.String(), since).Scan(&count); err != nil {
		return 0, fmt.Errorf("count recent message requests: %w", err)
	}
	return count, nil
}

func (repo *PostgresMessageRepository) CreateConversationWithMessage(ctx context.Context, input messageusecase.CreateConversationRecord) (messageusecase.ConversationRecord, messageusecase.MessageRecord, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return messageusecase.ConversationRecord{}, messageusecase.MessageRecord{}, fmt.Errorf("begin message conversation tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO message_conversations (id, kind, status, created_by, created_at, updated_at)
		VALUES ($1::uuid, 'direct', $2, $3::uuid, $4, $4)
	`, input.ID, input.Status, input.ViewerID.String(), input.Now); err != nil {
		return messageusecase.ConversationRecord{}, messageusecase.MessageRecord{}, fmt.Errorf("insert message conversation: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO message_conversation_participants (conversation_id, user_id, peer_user_id, read_at, created_at, updated_at)
		VALUES
			($1::uuid, $2::uuid, $3::uuid, $4, $4, $4),
			($1::uuid, $3::uuid, $2::uuid, NULL, $4, $4)
	`, input.ID, input.ViewerID.String(), input.PeerID.String(), input.Now); err != nil {
		return messageusecase.ConversationRecord{}, messageusecase.MessageRecord{}, fmt.Errorf("insert message conversation participants: %w", err)
	}
	if input.Status == messageusecase.ConversationStatusPending {
		if _, err := tx.Exec(ctx, `
			INSERT INTO message_requests (id, conversation_id, from_user_id, to_user_id, status, created_at, updated_at)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'pending', $5, $5)
		`, input.RequestID, input.ID, input.ViewerID.String(), input.PeerID.String(), input.Now); err != nil {
			return messageusecase.ConversationRecord{}, messageusecase.MessageRecord{}, fmt.Errorf("insert message request: %w", err)
		}
	}
	input.Message.ConversationID = input.ID
	message, err := insertMessage(ctx, tx, input.Message)
	if err != nil {
		return messageusecase.ConversationRecord{}, messageusecase.MessageRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return messageusecase.ConversationRecord{}, messageusecase.MessageRecord{}, fmt.Errorf("commit message conversation tx: %w", err)
	}
	conversation, err := repo.GetConversation(ctx, input.ID, input.ViewerID)
	if err != nil {
		return messageusecase.ConversationRecord{}, messageusecase.MessageRecord{}, err
	}
	return conversation, message, nil
}

func (repo *PostgresMessageRepository) InsertMessage(ctx context.Context, input messageusecase.CreateMessageRecord) (messageusecase.MessageRecord, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return messageusecase.MessageRecord{}, fmt.Errorf("begin insert message tx: %w", err)
	}
	defer tx.Rollback(ctx)
	message, err := insertMessage(ctx, tx, input)
	if err != nil {
		return messageusecase.MessageRecord{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE message_conversations SET updated_at = $2 WHERE id = $1::uuid`, input.ConversationID, input.Now); err != nil {
		return messageusecase.MessageRecord{}, fmt.Errorf("touch message conversation: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE message_conversation_participants SET updated_at = $2 WHERE conversation_id = $1::uuid`, input.ConversationID, input.Now); err != nil {
		return messageusecase.MessageRecord{}, fmt.Errorf("touch message participants: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return messageusecase.MessageRecord{}, fmt.Errorf("commit insert message tx: %w", err)
	}
	return message, nil
}

func (repo *PostgresMessageRepository) ListConversations(ctx context.Context, userID userdomain.UserID, box string, limit int, offset int) ([]messageusecase.ConversationRecord, error) {
	query := conversationSelect + `
		WHERE p.user_id = $1::uuid
			AND p.deleted_at IS NULL
	`
	switch box {
	case "friends":
		query += ` AND c.status = 'accepted' AND p.archived_at IS NULL`
	case "requests":
		query += ` AND c.status = 'pending' AND c.created_by <> $1::uuid AND p.archived_at IS NULL`
	case "archived":
		query += ` AND p.archived_at IS NOT NULL`
	default:
		query += ` AND p.archived_at IS NULL`
	}
	query += `
		ORDER BY p.pinned DESC, COALESCE(last_message.created_at, c.updated_at) DESC, c.id DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := repo.pool.Query(ctx, query, userID.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list message conversations: %w", err)
	}
	defer rows.Close()
	return scanConversations(rows)
}

func (repo *PostgresMessageRepository) GetConversation(ctx context.Context, conversationID string, userID userdomain.UserID) (messageusecase.ConversationRecord, error) {
	const query = conversationSelect + `
		WHERE p.user_id = $1::uuid
			AND c.id = $2::uuid
			AND p.deleted_at IS NULL
		LIMIT 1
	`
	return scanConversation(repo.pool.QueryRow(ctx, query, userID.String(), conversationID))
}

func (repo *PostgresMessageRepository) ListMessages(ctx context.Context, conversationID string, userID userdomain.UserID, beforeMessageID string, limit int) ([]messageusecase.MessageRecord, error) {
	args := []any{conversationID, userID.String(), limit}
	query := messageSelect + `
		INNER JOIN message_conversation_participants AS p
			ON p.conversation_id = m.conversation_id
			AND p.user_id = $2::uuid
			AND p.deleted_at IS NULL
		WHERE m.conversation_id = $1::uuid
			AND NOT EXISTS (
				SELECT 1
				FROM message_user_states AS hidden
				WHERE hidden.message_id = m.id
					AND hidden.user_id = $2::uuid
			)
	`
	if beforeMessageID != "" {
		args = append(args, beforeMessageID)
		query += ` AND m.created_at < (SELECT before_message.created_at FROM messages AS before_message WHERE before_message.id = $4::uuid)`
	}
	query += ` ORDER BY m.created_at DESC, m.id DESC LIMIT $3`
	rows, err := repo.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

func (repo *PostgresMessageRepository) MarkConversationRead(ctx context.Context, conversationID string, userID userdomain.UserID, now time.Time) error {
	const query = `
		UPDATE message_conversation_participants
		SET read_at = $3,
			updated_at = $3
		WHERE conversation_id = $1::uuid
			AND user_id = $2::uuid
			AND deleted_at IS NULL
	`
	tag, err := repo.pool.Exec(ctx, query, conversationID, userID.String(), now)
	if err != nil {
		return fmt.Errorf("mark message conversation read: %w", err)
	}
	return ensureRows(tag.RowsAffected(), "message conversation not found")
}

func (repo *PostgresMessageRepository) SetConversationArchived(ctx context.Context, conversationID string, userID userdomain.UserID, archived bool, now time.Time) error {
	archivedAt := any(nil)
	if archived {
		archivedAt = now
	}
	const query = `
		UPDATE message_conversation_participants
		SET archived_at = $3,
			updated_at = $4
		WHERE conversation_id = $1::uuid
			AND user_id = $2::uuid
			AND deleted_at IS NULL
	`
	tag, err := repo.pool.Exec(ctx, query, conversationID, userID.String(), archivedAt, now)
	if err != nil {
		return fmt.Errorf("set message conversation archive: %w", err)
	}
	return ensureRows(tag.RowsAffected(), "message conversation not found")
}

func (repo *PostgresMessageRepository) SetConversationPinned(ctx context.Context, conversationID string, userID userdomain.UserID, pinned bool, now time.Time) error {
	const query = `
		UPDATE message_conversation_participants
		SET pinned = $3,
			updated_at = $4
		WHERE conversation_id = $1::uuid
			AND user_id = $2::uuid
			AND deleted_at IS NULL
	`
	tag, err := repo.pool.Exec(ctx, query, conversationID, userID.String(), pinned, now)
	if err != nil {
		return fmt.Errorf("set message conversation pin: %w", err)
	}
	return ensureRows(tag.RowsAffected(), "message conversation not found")
}

func (repo *PostgresMessageRepository) SetConversationMuted(ctx context.Context, conversationID string, userID userdomain.UserID, muted bool, now time.Time) error {
	const query = `
		UPDATE message_conversation_participants
		SET muted = $3,
			updated_at = $4
		WHERE conversation_id = $1::uuid
			AND user_id = $2::uuid
			AND deleted_at IS NULL
	`
	tag, err := repo.pool.Exec(ctx, query, conversationID, userID.String(), muted, now)
	if err != nil {
		return fmt.Errorf("set message conversation mute: %w", err)
	}
	return ensureRows(tag.RowsAffected(), "message conversation not found")
}

func (repo *PostgresMessageRepository) HideConversationForUser(ctx context.Context, conversationID string, userID userdomain.UserID, now time.Time) error {
	const query = `
		UPDATE message_conversation_participants
		SET deleted_at = $3,
			archived_at = NULL,
			pinned = false,
			muted = false,
			updated_at = $3
		WHERE conversation_id = $1::uuid
			AND user_id = $2::uuid
			AND deleted_at IS NULL
	`
	tag, err := repo.pool.Exec(ctx, query, conversationID, userID.String(), now)
	if err != nil {
		return fmt.Errorf("hide message conversation for user: %w", err)
	}
	return ensureRows(tag.RowsAffected(), "message conversation not found")
}

func (repo *PostgresMessageRepository) AcceptMessageRequest(ctx context.Context, requestID string, userID userdomain.UserID, now time.Time) (messageusecase.ConversationRecord, error) {
	return repo.updateMessageRequest(ctx, requestID, userID, "accepted", now)
}

func (repo *PostgresMessageRepository) RejectMessageRequest(ctx context.Context, requestID string, userID userdomain.UserID, now time.Time) (messageusecase.ConversationRecord, error) {
	return repo.updateMessageRequest(ctx, requestID, userID, "rejected", now)
}

func (repo *PostgresMessageRepository) updateMessageRequest(ctx context.Context, requestID string, userID userdomain.UserID, status string, now time.Time) (messageusecase.ConversationRecord, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return messageusecase.ConversationRecord{}, fmt.Errorf("begin update message request tx: %w", err)
	}
	defer tx.Rollback(ctx)
	var conversationID string
	err = tx.QueryRow(ctx, `
		UPDATE message_requests
		SET status = $3,
			updated_at = $4,
			responded_at = $4
		WHERE id = $1::uuid
			AND to_user_id = $2::uuid
			AND status = 'pending'
		RETURNING conversation_id::text
	`, requestID, userID.String(), status, now).Scan(&conversationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return messageusecase.ConversationRecord{}, apperr.New(apperr.CodeNotFound, "message request not found")
		}
		return messageusecase.ConversationRecord{}, fmt.Errorf("update message request: %w", err)
	}
	conversationStatus := status
	if status == "accepted" {
		conversationStatus = "accepted"
	}
	if _, err := tx.Exec(ctx, `
		UPDATE message_conversations
		SET status = $2,
			updated_at = $3
		WHERE id = $1::uuid
	`, conversationID, conversationStatus, now); err != nil {
		return messageusecase.ConversationRecord{}, fmt.Errorf("update message conversation request status: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE message_conversation_participants SET updated_at = $2 WHERE conversation_id = $1::uuid`, conversationID, now); err != nil {
		return messageusecase.ConversationRecord{}, fmt.Errorf("touch message request participants: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return messageusecase.ConversationRecord{}, fmt.Errorf("commit message request tx: %w", err)
	}
	return repo.GetConversation(ctx, conversationID, userID)
}

func (repo *PostgresMessageRepository) GetMessage(ctx context.Context, messageID string, userID userdomain.UserID) (messageusecase.MessageRecord, error) {
	const query = messageSelect + `
		INNER JOIN message_conversation_participants AS p
			ON p.conversation_id = m.conversation_id
			AND p.user_id = $2::uuid
			AND p.deleted_at IS NULL
		WHERE m.id = $1::uuid
		LIMIT 1
	`
	return scanMessage(repo.pool.QueryRow(ctx, query, messageID, userID.String()))
}

func (repo *PostgresMessageRepository) UpdateMessageStatus(ctx context.Context, messageID string, status string, now time.Time) (messageusecase.MessageRecord, error) {
	const query = `
		UPDATE messages
		SET status = $2,
			updated_at = $3,
			recalled_at = CASE WHEN $2 = 'recalled' THEN $3 ELSE recalled_at END
		WHERE id = $1::uuid
		RETURNING sender_id::text
	`
	var senderID string
	if err := repo.pool.QueryRow(ctx, query, messageID, status, now).Scan(&senderID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return messageusecase.MessageRecord{}, apperr.New(apperr.CodeNotFound, "message not found")
		}
		return messageusecase.MessageRecord{}, fmt.Errorf("update message status: %w", err)
	}
	parsedSenderID, err := userdomain.NewUserID(senderID)
	if err != nil {
		return messageusecase.MessageRecord{}, err
	}
	return repo.GetMessage(ctx, messageID, parsedSenderID)
}

func (repo *PostgresMessageRepository) HideMessageForUser(ctx context.Context, messageID string, userID userdomain.UserID, now time.Time) error {
	const query = `
		INSERT INTO message_user_states (message_id, user_id, hidden_at)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (message_id, user_id) DO UPDATE SET hidden_at = EXCLUDED.hidden_at
	`
	if _, err := repo.pool.Exec(ctx, query, messageID, userID.String(), now); err != nil {
		return fmt.Errorf("hide message for user: %w", err)
	}
	return nil
}

func (repo *PostgresMessageRepository) CreateMessageReport(ctx context.Context, input messageusecase.CreateReportRecord) (messageusecase.ReportRecord, error) {
	const query = `
		WITH reported AS (
			SELECT m.conversation_id, m.sender_id
			FROM messages AS m
			INNER JOIN message_conversation_participants AS p
				ON p.conversation_id = m.conversation_id
				AND p.user_id = $2::uuid
			WHERE m.id = $3::uuid
		), before_context AS (
			SELECT string_agg(body, E'\n' ORDER BY created_at DESC) AS body
			FROM (
				SELECT m.body, m.created_at
				FROM messages AS m, reported
				WHERE m.conversation_id = reported.conversation_id
					AND m.created_at < (SELECT created_at FROM messages WHERE id = $3::uuid)
				ORDER BY m.created_at DESC
				LIMIT 2
			) AS limited_before
		), after_context AS (
			SELECT string_agg(body, E'\n' ORDER BY created_at ASC) AS body
			FROM (
				SELECT m.body, m.created_at
				FROM messages AS m, reported
				WHERE m.conversation_id = reported.conversation_id
					AND m.created_at > (SELECT created_at FROM messages WHERE id = $3::uuid)
				ORDER BY m.created_at ASC
				LIMIT 2
			) AS limited_after
		)
		INSERT INTO message_reports (
			id,
			reporter_id,
			conversation_id,
			message_id,
			reported_user_id,
			reason,
			context_before,
			context_after,
			created_at
		)
		SELECT
			$1::uuid,
			$2::uuid,
			reported.conversation_id,
			$3::uuid,
			reported.sender_id,
			$4,
			COALESCE(before_context.body, ''),
			COALESCE(after_context.body, ''),
			$5
		FROM reported, before_context, after_context
		RETURNING id::text, conversation_id::text, message_id::text, reported_user_id::text, reason, context_before, context_after, created_at
	`
	var record messageusecase.ReportRecord
	var rawReportedUserID string
	err := repo.pool.QueryRow(ctx, query, input.ID, input.ReporterID.String(), input.MessageID, input.Reason, input.Now).Scan(
		&record.ID,
		&record.ConversationID,
		&record.MessageID,
		&rawReportedUserID,
		&record.Reason,
		&record.ContextBefore,
		&record.ContextAfter,
		&record.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return messageusecase.ReportRecord{}, apperr.New(apperr.CodeNotFound, "message not found")
		}
		return messageusecase.ReportRecord{}, fmt.Errorf("create message report: %w", err)
	}
	reportedUserID, err := userdomain.NewUserID(rawReportedUserID)
	if err != nil {
		return messageusecase.ReportRecord{}, err
	}
	record.ReportedUserID = reportedUserID
	return record, nil
}

func (repo *PostgresMessageRepository) CreateRealtimeEvent(ctx context.Context, input messageusecase.RealtimeEventRecord) error {
	const query = `
		INSERT INTO message_realtime_events (id, user_id, conversation_id, type, payload, created_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6)
	`
	var conversationID any
	if input.ConversationID != nil {
		conversationID = *input.ConversationID
	}
	if _, err := repo.pool.Exec(ctx, query, input.ID, input.UserID.String(), conversationID, input.Type, input.Payload, input.CreatedAt); err != nil {
		return fmt.Errorf("create message realtime event: %w", err)
	}
	return nil
}

func (repo *PostgresMessageRepository) CreateRealtimeTicket(ctx context.Context, ticket string, userID userdomain.UserID, lastEventID string, expiresAt time.Time, now time.Time) (messageusecase.RealtimeTicketRecord, error) {
	const query = `
		INSERT INTO message_realtime_tickets (ticket, user_id, last_event_id, expires_at, created_at)
		VALUES ($1, $2::uuid, NULLIF($3, '')::uuid, $4, $5)
		RETURNING ticket, user_id::text, COALESCE(last_event_id::text, ''), expires_at
	`
	return repo.scanRealtimeTicket(repo.pool.QueryRow(ctx, query, ticket, userID.String(), lastEventID, expiresAt, now))
}

func (repo *PostgresMessageRepository) ConsumeRealtimeTicket(ctx context.Context, ticket string, now time.Time) (messageusecase.RealtimeTicketRecord, error) {
	const query = `
		UPDATE message_realtime_tickets
		SET consumed_at = $2
		WHERE ticket = $1
			AND expires_at > $2
			AND consumed_at IS NULL
		RETURNING ticket, user_id::text, COALESCE(last_event_id::text, ''), expires_at
	`
	return repo.scanRealtimeTicket(repo.pool.QueryRow(ctx, query, ticket, now))
}

func (repo *PostgresMessageRepository) ListRealtimeEventsAfter(ctx context.Context, userID userdomain.UserID, afterEventID string, limit int) ([]messageusecase.RealtimeEventRecord, error) {
	args := []any{userID.String(), limit}
	query := `
		SELECT id::text, user_id::text, COALESCE(conversation_id::text, ''), type, payload, created_at
		FROM message_realtime_events
		WHERE user_id = $1::uuid
	`
	if afterEventID != "" {
		args = append(args, afterEventID)
		query += ` AND created_at > COALESCE((SELECT created_at FROM message_realtime_events WHERE id = $3::uuid AND user_id = $1::uuid), '-infinity'::timestamptz)`
	}
	query += ` ORDER BY created_at ASC, id ASC LIMIT $2`
	rows, err := repo.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list message realtime events: %w", err)
	}
	defer rows.Close()
	events := make([]messageusecase.RealtimeEventRecord, 0)
	for rows.Next() {
		var event messageusecase.RealtimeEventRecord
		var rawUserID string
		var rawConversationID string
		if err := rows.Scan(&event.ID, &rawUserID, &rawConversationID, &event.Type, &event.Payload, &event.CreatedAt); err != nil {
			return nil, err
		}
		parsedUserID, err := userdomain.NewUserID(rawUserID)
		if err != nil {
			return nil, err
		}
		event.UserID = parsedUserID
		if rawConversationID != "" {
			event.ConversationID = &rawConversationID
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate realtime events: %w", err)
	}
	return events, nil
}

const conversationSelect = `
	SELECT
		c.id::text,
		c.status,
		c.created_by::text,
		COALESCE(req.id::text, ''),
		COALESCE(req.status, ''),
		peer.id::text,
		peer.username,
		peer.password_hash,
		peer.display_name,
		peer.avatar_url,
		peer.banner_url,
		peer.headline,
		peer.bio,
		peer.status,
		peer.created_at,
		peer.updated_at,
		COALESCE(last_message.id::text, ''),
		COALESCE(last_message.conversation_id::text, ''),
		COALESCE(last_sender.id::text, ''),
		COALESCE(last_sender.username, ''),
		COALESCE(last_sender.password_hash, ''),
		COALESCE(last_sender.display_name, ''),
		COALESCE(last_sender.avatar_url, ''),
		COALESCE(last_sender.banner_url, ''),
		COALESCE(last_sender.headline, ''),
		COALESCE(last_sender.bio, ''),
		COALESCE(last_sender.status, ''),
		COALESCE(last_sender.created_at, 'epoch'::timestamptz),
		COALESCE(last_sender.updated_at, 'epoch'::timestamptz),
		COALESCE(last_message.type, ''),
		COALESCE(last_message.body, ''),
		COALESCE(last_message.image_url, ''),
		COALESCE(last_message.share_type, ''),
		COALESCE(last_message.share_id, ''),
		COALESCE(last_message.share_title, ''),
		COALESCE(last_message.share_summary, ''),
		COALESCE(last_message.share_thumbnail_url, ''),
		COALESCE(last_message.share_target_url, ''),
		last_message.share_snapshot_created_at,
		COALESCE(last_message.status, ''),
		last_message.created_at,
		last_message.updated_at,
		last_message.recalled_at,
		(
			SELECT COUNT(*)::int
			FROM messages AS unread
			WHERE unread.conversation_id = c.id
				AND unread.sender_id <> p.user_id
				AND (p.read_at IS NULL OR unread.created_at > p.read_at)
				AND NOT EXISTS (
					SELECT 1
					FROM message_user_states AS hidden
					WHERE hidden.message_id = unread.id
						AND hidden.user_id = p.user_id
				)
				AND unread.status = 'visible'
		) AS unread_count,
		COALESCE(last_message.created_at, c.updated_at) AS sort_updated_at,
		p.pinned,
		p.muted,
		p.archived_at IS NOT NULL,
		EXISTS (
			SELECT 1
			FROM user_blocks
			WHERE (blocker_id = p.user_id AND blocked_id = p.peer_user_id)
				OR (blocker_id = p.peer_user_id AND blocked_id = p.user_id)
		) AS blocked,
		COALESCE(peer_privacy.online_status_enabled, false)
	FROM message_conversation_participants AS p
	INNER JOIN message_conversations AS c ON c.id = p.conversation_id
	INNER JOIN users AS peer ON peer.id = p.peer_user_id
	LEFT JOIN message_requests AS req ON req.conversation_id = c.id
	LEFT JOIN message_privacy_settings AS peer_privacy ON peer_privacy.user_id = p.peer_user_id
	LEFT JOIN LATERAL (
		SELECT m.*
		FROM messages AS m
		WHERE m.conversation_id = c.id
			AND NOT EXISTS (
				SELECT 1
				FROM message_user_states AS hidden
				WHERE hidden.message_id = m.id
					AND hidden.user_id = p.user_id
			)
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT 1
	) AS last_message ON true
	LEFT JOIN users AS last_sender ON last_sender.id = last_message.sender_id
`

const messageSelect = `
	SELECT
		m.id::text,
		m.conversation_id::text,
		sender.id::text,
		sender.username,
		sender.password_hash,
		sender.display_name,
		sender.avatar_url,
		sender.banner_url,
		sender.headline,
		sender.bio,
		sender.status,
		sender.created_at,
		sender.updated_at,
		m.type,
		m.body,
		m.image_url,
		COALESCE(m.share_type, ''),
		COALESCE(m.share_id, ''),
		m.share_title,
		m.share_summary,
		m.share_thumbnail_url,
		m.share_target_url,
		m.share_snapshot_created_at,
		m.status,
		m.created_at,
		m.updated_at,
		m.recalled_at,
		EXISTS (
			SELECT 1
			FROM message_user_states AS hidden
			WHERE hidden.message_id = m.id
				AND hidden.user_id = p.user_id
		) AS viewer_deleted
	FROM messages AS m
	INNER JOIN users AS sender ON sender.id = m.sender_id
`

func insertMessage(ctx context.Context, tx pgx.Tx, input messageusecase.CreateMessageRecord) (messageusecase.MessageRecord, error) {
	var shareType any
	var shareID any
	var shareSnapshotCreatedAt any
	shareTitle := ""
	shareSummary := ""
	shareThumbnailURL := ""
	shareTargetURL := ""
	if input.Share != nil {
		shareType = input.Share.ShareType
		shareID = input.Share.ShareID
		shareTitle = input.Share.Title
		shareSummary = input.Share.Summary
		shareThumbnailURL = input.Share.ThumbnailURL
		shareTargetURL = input.Share.TargetURL
		shareSnapshotCreatedAt = input.Share.SnapshotCreatedAt
	}
	const query = `
		INSERT INTO messages (
			id,
			conversation_id,
			sender_id,
			type,
			body,
			image_url,
			share_type,
			share_id,
			share_title,
			share_summary,
			share_thumbnail_url,
			share_target_url,
			share_snapshot_created_at,
			status,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 'visible', $14, $14)
	`
	if _, err := tx.Exec(ctx, query, input.ID, input.ConversationID, input.SenderID.String(), input.Type, input.Body, input.ImageURL, shareType, shareID, shareTitle, shareSummary, shareThumbnailURL, shareTargetURL, shareSnapshotCreatedAt, input.Now); err != nil {
		return messageusecase.MessageRecord{}, fmt.Errorf("insert message: %w", err)
	}
	return scanMessage(tx.QueryRow(ctx, messageSelect+`
		INNER JOIN message_conversation_participants AS p
			ON p.conversation_id = m.conversation_id
			AND p.user_id = $2::uuid
		WHERE m.id = $1::uuid
	`, input.ID, input.SenderID.String()))
}

func scanConversations(rows pgx.Rows) ([]messageusecase.ConversationRecord, error) {
	records := make([]messageusecase.ConversationRecord, 0)
	for rows.Next() {
		record, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate message conversations: %w", err)
	}
	return records, nil
}

func scanConversation(row pgx.Row) (messageusecase.ConversationRecord, error) {
	var record messageusecase.ConversationRecord
	var rawCreatedBy string
	var rawRequestID string
	var rawPeer userScan
	var rawLastMessageID string
	var rawLastConversationID string
	var rawLastSender userScan
	var rawLastType string
	var rawLastBody string
	var rawLastImageURL string
	var rawLastShareType string
	var rawLastShareID string
	var rawLastShareTitle string
	var rawLastShareSummary string
	var rawLastShareThumbnailURL string
	var rawLastShareTargetURL string
	var rawLastShareSnapshotCreatedAt pgtype.Timestamptz
	var rawLastStatus string
	var rawLastCreatedAt pgtype.Timestamptz
	var rawLastUpdatedAt pgtype.Timestamptz
	var rawLastRecalledAt pgtype.Timestamptz
	if err := row.Scan(
		&record.ID,
		&record.Status,
		&rawCreatedBy,
		&rawRequestID,
		&record.RequestStatus,
		&rawPeer.id,
		&rawPeer.username,
		&rawPeer.passwordHash,
		&rawPeer.displayName,
		&rawPeer.avatarURL,
		&rawPeer.bannerURL,
		&rawPeer.headline,
		&rawPeer.bio,
		&rawPeer.status,
		&rawPeer.createdAt,
		&rawPeer.updatedAt,
		&rawLastMessageID,
		&rawLastConversationID,
		&rawLastSender.id,
		&rawLastSender.username,
		&rawLastSender.passwordHash,
		&rawLastSender.displayName,
		&rawLastSender.avatarURL,
		&rawLastSender.bannerURL,
		&rawLastSender.headline,
		&rawLastSender.bio,
		&rawLastSender.status,
		&rawLastSender.createdAt,
		&rawLastSender.updatedAt,
		&rawLastType,
		&rawLastBody,
		&rawLastImageURL,
		&rawLastShareType,
		&rawLastShareID,
		&rawLastShareTitle,
		&rawLastShareSummary,
		&rawLastShareThumbnailURL,
		&rawLastShareTargetURL,
		&rawLastShareSnapshotCreatedAt,
		&rawLastStatus,
		&rawLastCreatedAt,
		&rawLastUpdatedAt,
		&rawLastRecalledAt,
		&record.UnreadCount,
		&record.UpdatedAt,
		&record.Pinned,
		&record.Muted,
		&record.Archived,
		&record.Blocked,
		&record.PeerOnlineStatusEnabled,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return messageusecase.ConversationRecord{}, apperr.New(apperr.CodeNotFound, "message conversation not found")
		}
		return messageusecase.ConversationRecord{}, err
	}
	createdBy, err := userdomain.NewUserID(rawCreatedBy)
	if err != nil {
		return messageusecase.ConversationRecord{}, err
	}
	record.CreatedBy = createdBy
	if rawRequestID != "" {
		record.RequestID = &rawRequestID
	}
	peer, err := rawPeer.toUser()
	if err != nil {
		return messageusecase.ConversationRecord{}, err
	}
	record.Peer = *peer
	if rawLastMessageID != "" {
		lastSender, err := rawLastSender.toUser()
		if err != nil {
			return messageusecase.ConversationRecord{}, err
		}
		lastMessage := messageusecase.MessageRecord{
			ID:             rawLastMessageID,
			ConversationID: rawLastConversationID,
			Sender:         *lastSender,
			Type:           rawLastType,
			Body:           rawLastBody,
			ImageURL:       rawLastImageURL,
			Status:         rawLastStatus,
		}
		if rawLastCreatedAt.Valid {
			lastMessage.CreatedAt = rawLastCreatedAt.Time
		}
		if rawLastUpdatedAt.Valid {
			lastMessage.UpdatedAt = rawLastUpdatedAt.Time
		}
		if rawLastRecalledAt.Valid {
			lastMessage.RecalledAt = &rawLastRecalledAt.Time
		}
		lastMessage.Share = scanShare(rawLastShareType, rawLastShareID, rawLastShareTitle, rawLastShareSummary, rawLastShareThumbnailURL, rawLastShareTargetURL, rawLastShareSnapshotCreatedAt)
		record.LastMessage = &lastMessage
	}
	return record, nil
}

func scanMessages(rows pgx.Rows) ([]messageusecase.MessageRecord, error) {
	records := make([]messageusecase.MessageRecord, 0)
	for rows.Next() {
		record, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}
	return records, nil
}

func scanMessage(row pgx.Row) (messageusecase.MessageRecord, error) {
	var record messageusecase.MessageRecord
	var rawSender userScan
	var rawShareType string
	var rawShareID string
	var rawShareTitle string
	var rawShareSummary string
	var rawShareThumbnailURL string
	var rawShareTargetURL string
	var rawShareSnapshotCreatedAt pgtype.Timestamptz
	var recalledAt pgtype.Timestamptz
	if err := row.Scan(
		&record.ID,
		&record.ConversationID,
		&rawSender.id,
		&rawSender.username,
		&rawSender.passwordHash,
		&rawSender.displayName,
		&rawSender.avatarURL,
		&rawSender.bannerURL,
		&rawSender.headline,
		&rawSender.bio,
		&rawSender.status,
		&rawSender.createdAt,
		&rawSender.updatedAt,
		&record.Type,
		&record.Body,
		&record.ImageURL,
		&rawShareType,
		&rawShareID,
		&rawShareTitle,
		&rawShareSummary,
		&rawShareThumbnailURL,
		&rawShareTargetURL,
		&rawShareSnapshotCreatedAt,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
		&recalledAt,
		&record.ViewerDeleted,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return messageusecase.MessageRecord{}, apperr.New(apperr.CodeNotFound, "message not found")
		}
		return messageusecase.MessageRecord{}, err
	}
	sender, err := rawSender.toUser()
	if err != nil {
		return messageusecase.MessageRecord{}, err
	}
	record.Sender = *sender
	if recalledAt.Valid {
		record.RecalledAt = &recalledAt.Time
	}
	record.Share = scanShare(rawShareType, rawShareID, rawShareTitle, rawShareSummary, rawShareThumbnailURL, rawShareTargetURL, rawShareSnapshotCreatedAt)
	return record, nil
}

func scanShare(shareType string, shareID string, title string, summary string, thumbnailURL string, targetURL string, snapshotCreatedAt pgtype.Timestamptz) *messageusecase.ShareSnapshot {
	if shareType == "" {
		return nil
	}
	share := &messageusecase.ShareSnapshot{
		ShareType:    shareType,
		ShareID:      shareID,
		Title:        title,
		Summary:      summary,
		ThumbnailURL: thumbnailURL,
		TargetURL:    targetURL,
	}
	if snapshotCreatedAt.Valid {
		share.SnapshotCreatedAt = snapshotCreatedAt.Time
	}
	return share
}

func (repo *PostgresMessageRepository) scanRealtimeTicket(row pgx.Row) (messageusecase.RealtimeTicketRecord, error) {
	var record messageusecase.RealtimeTicketRecord
	var rawUserID string
	if err := row.Scan(&record.Ticket, &rawUserID, &record.LastEventID, &record.ExpiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return messageusecase.RealtimeTicketRecord{}, apperr.New(apperr.CodeUnauthenticated, "realtime ticket is invalid")
		}
		return messageusecase.RealtimeTicketRecord{}, fmt.Errorf("scan realtime ticket: %w", err)
	}
	userID, err := userdomain.NewUserID(rawUserID)
	if err != nil {
		return messageusecase.RealtimeTicketRecord{}, err
	}
	record.UserID = userID
	return record, nil
}

type userScan struct {
	id           string
	username     string
	passwordHash string
	displayName  string
	avatarURL    string
	bannerURL    string
	headline     string
	bio          string
	status       string
	createdAt    time.Time
	updatedAt    time.Time
}

func (raw userScan) toUser() (*userdomain.User, error) {
	id, err := userdomain.NewUserID(raw.id)
	if err != nil {
		return nil, err
	}
	username, err := userdomain.NewUsername(raw.username)
	if err != nil {
		return nil, err
	}
	passwordHash, err := userdomain.NewPasswordHash(raw.passwordHash)
	if err != nil {
		return nil, err
	}
	displayName, err := userdomain.NewDisplayName(raw.displayName)
	if err != nil {
		return nil, err
	}
	avatarURL, err := userdomain.NewAvatarURL(raw.avatarURL)
	if err != nil {
		return nil, err
	}
	bannerURL, err := userdomain.NewBannerURL(raw.bannerURL)
	if err != nil {
		return nil, err
	}
	headline, err := userdomain.NewHeadline(raw.headline)
	if err != nil {
		return nil, err
	}
	bio, err := userdomain.NewBio(raw.bio)
	if err != nil {
		return nil, err
	}
	status, err := userdomain.NewUserStatus(raw.status)
	if err != nil {
		return nil, err
	}
	return userdomain.RehydrateUserWithProfile(id, username, passwordHash, displayName, avatarURL, bannerURL, headline, bio, status, raw.createdAt, raw.updatedAt)
}

func ensureRows(rows int64, message string) error {
	if rows == 0 {
		return apperr.New(apperr.CodeNotFound, message)
	}
	return nil
}

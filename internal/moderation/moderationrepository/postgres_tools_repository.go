package moderationrepository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationusecase"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var _ moderationusecase.ToolsRepository = (*PostgresModerationRepository)(nil)

func (repo *PostgresModerationRepository) ListModQueue(ctx context.Context, input moderationusecase.ListModQueueRecordInput) ([]moderationusecase.ModQueueItem, error) {
	communityID := nullableCommunityID(input.CommunityID)
	if input.Queue == "reports" || input.Queue == "needs_review" {
		return repo.listReportQueue(ctx, communityID, input.Queue, input.Limit, input.Offset)
	}
	return repo.listContentQueue(ctx, communityID, input.Queue, input.Limit, input.Offset)
}

func (repo *PostgresModerationRepository) GetModQueueItem(ctx context.Context, input moderationusecase.GetModQueueItemRecordInput) (moderationusecase.ModQueueItemDetail, error) {
	item, preview, err := repo.findModQueueItem(ctx, nullableCommunityID(input.CommunityID), input.TargetType, input.TargetID)
	if err != nil {
		return moderationusecase.ModQueueItemDetail{}, err
	}
	reports, err := repo.listReportsForModQueueTarget(ctx, input.TargetType, input.TargetID)
	if err != nil {
		return moderationusecase.ModQueueItemDetail{}, err
	}
	actions, err := repo.listActionsForModQueueTarget(ctx, input.TargetType, input.TargetID)
	if err != nil {
		return moderationusecase.ModQueueItemDetail{}, err
	}
	return moderationusecase.ModQueueItemDetail{
		Item:          item,
		TargetPreview: preview,
		Reports:       reports,
		RecentActions: actions,
	}, nil
}

func (repo *PostgresModerationRepository) GetModQueueSummary(ctx context.Context, input moderationusecase.GetModQueueSummaryRecordInput) (moderationusecase.ModQueueSummary, error) {
	communityID := nullableCommunityID(input.CommunityID)
	counts, err := repo.countModQueues(ctx, communityID)
	if err != nil {
		return moderationusecase.ModQueueSummary{}, err
	}
	limit := input.PriorityItemLimit
	if limit <= 0 {
		limit = moderationusecase.DefaultModToolsListLimit
	}
	priorityItems, err := repo.ListModQueue(ctx, moderationusecase.ListModQueueRecordInput{
		CommunityID: input.CommunityID,
		Queue:       "reports",
		Limit:       limit,
		Offset:      input.PriorityItemOffset,
	})
	if err != nil {
		return moderationusecase.ModQueueSummary{}, err
	}
	return moderationusecase.ModQueueSummary{Queues: counts, PriorityItems: priorityItems}, nil
}

func (repo *PostgresModerationRepository) GetAutomodConfig(ctx context.Context, communityID communitydomain.CommunityID) (moderationusecase.AutomodConfig, error) {
	const query = `
		SELECT community_id::text, config_text, rules, version, updated_by::text, updated_at
		FROM community_automod_configs
		WHERE community_id = $1::uuid
	`
	config, err := scanAutomodConfig(repo.pool.QueryRow(ctx, query, communityID.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return moderationusecase.AutomodConfig{
				CommunityID: communityID.String(),
				Rules:       json.RawMessage("{}"),
			}, nil
		}
		return moderationusecase.AutomodConfig{}, err
	}
	return config, nil
}

func (repo *PostgresModerationRepository) UpsertAutomodConfig(ctx context.Context, input moderationusecase.UpsertAutomodConfigRecordInput) (moderationusecase.AutomodConfig, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return moderationusecase.AutomodConfig{}, fmt.Errorf("begin automod config transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	before, err := scanAutomodConfig(tx.QueryRow(ctx, `
		SELECT community_id::text, config_text, rules, version, updated_by::text, updated_at
		FROM community_automod_configs
		WHERE community_id = $1::uuid
		FOR UPDATE
	`, input.CommunityID.String()))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return moderationusecase.AutomodConfig{}, err
	}
	nextVersion := before.Version + 1
	if errors.Is(err, pgx.ErrNoRows) {
		nextVersion = 1
	}
	const upsert = `
		INSERT INTO community_automod_configs (community_id, config_text, rules, version, updated_by, updated_at)
		VALUES ($1::uuid, $2, $3::jsonb, $4, $5::uuid, $6)
		ON CONFLICT (community_id) DO UPDATE
		SET config_text = EXCLUDED.config_text,
			rules = EXCLUDED.rules,
			version = EXCLUDED.version,
			updated_by = EXCLUDED.updated_by,
			updated_at = EXCLUDED.updated_at
		RETURNING community_id::text, config_text, rules, version, updated_by::text, updated_at
	`
	config, err := scanAutomodConfig(tx.QueryRow(ctx, upsert, input.CommunityID.String(), input.ConfigText, string(input.Rules), nextVersion, input.ActorID.String(), input.UpdatedAt))
	if err != nil {
		return moderationusecase.AutomodConfig{}, mapPostgresWriteError("upsert automod config", err)
	}
	const versionInsert = `
		INSERT INTO community_automod_config_versions (id, community_id, version, config_text, rules, updated_by, created_at)
		VALUES (gen_random_uuid(), $1::uuid, $2, $3, $4::jsonb, $5::uuid, $6)
	`
	if _, err := tx.Exec(ctx, versionInsert, input.CommunityID.String(), config.Version, input.ConfigText, string(input.Rules), input.ActorID.String(), input.UpdatedAt); err != nil {
		return moderationusecase.AutomodConfig{}, mapPostgresWriteError("insert automod config version", err)
	}
	beforeState := map[string]any{"version": before.Version, "config_text": before.ConfigText, "rules": json.RawMessage("{}")}
	if len(before.Rules) > 0 {
		beforeState["rules"] = before.Rules
	}
	afterState := map[string]any{"version": config.Version, "config_text": config.ConfigText, "rules": config.Rules}
	if err := insertCommunityModLog(ctx, tx, input.CommunityID, moderationusecase.ApplyModerationActionRecordInput{
		ActorID:    input.ActorID,
		TargetType: moderationdomain.TargetType("automod_config"),
		TargetID:   input.CommunityID.String(),
		Action:     moderationdomain.ActionType("update_automod_config"),
		CreatedAt:  input.UpdatedAt,
	}, beforeState, afterState); err != nil {
		return moderationusecase.AutomodConfig{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return moderationusecase.AutomodConfig{}, fmt.Errorf("commit automod config transaction: %w", err)
	}
	committed = true
	return config, nil
}

func (repo *PostgresModerationRepository) ListAutomodVersions(ctx context.Context, communityID communitydomain.CommunityID, limit int, offset int) ([]moderationusecase.AutomodVersion, error) {
	const query = `
		SELECT id::text, community_id::text, version, config_text, rules, updated_by::text, created_at
		FROM community_automod_config_versions
		WHERE community_id = $1::uuid
		ORDER BY version DESC
		LIMIT $2
		OFFSET $3
	`
	rows, err := repo.pool.Query(ctx, query, communityID.String(), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	versions := make([]moderationusecase.AutomodVersion, 0)
	for rows.Next() {
		version, err := scanAutomodVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (repo *PostgresModerationRepository) GetContentControls(ctx context.Context, communityID communitydomain.CommunityID) (moderationusecase.ContentControls, error) {
	const query = `
		SELECT community_id::text, blocked_keywords, blocked_domains, min_account_age_days, post_rate_limit_per_hour, comment_rate_limit_per_hour, block_new_accounts, filter_links, updated_by::text, updated_at
		FROM community_content_controls
		WHERE community_id = $1::uuid
	`
	controls, err := scanContentControls(repo.pool.QueryRow(ctx, query, communityID.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return moderationusecase.ContentControls{CommunityID: communityID.String(), BlockedKeywords: []string{}, BlockedDomains: []string{}}, nil
		}
		return moderationusecase.ContentControls{}, err
	}
	return controls, nil
}

func (repo *PostgresModerationRepository) UpsertContentControls(ctx context.Context, input moderationusecase.UpsertContentControlsRecordInput) (moderationusecase.ContentControls, error) {
	keywordsJSON, err := json.Marshal(input.BlockedKeywords)
	if err != nil {
		return moderationusecase.ContentControls{}, err
	}
	domainsJSON, err := json.Marshal(input.BlockedDomains)
	if err != nil {
		return moderationusecase.ContentControls{}, err
	}
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return moderationusecase.ContentControls{}, fmt.Errorf("begin content controls transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	before, err := repo.GetContentControls(ctx, input.CommunityID)
	if err != nil {
		return moderationusecase.ContentControls{}, err
	}
	const query = `
		INSERT INTO community_content_controls (
			community_id, blocked_keywords, blocked_domains, min_account_age_days, post_rate_limit_per_hour,
			comment_rate_limit_per_hour, block_new_accounts, filter_links, updated_by, updated_at
		)
		VALUES ($1::uuid, $2::jsonb, $3::jsonb, $4, $5, $6, $7, $8, $9::uuid, $10)
		ON CONFLICT (community_id) DO UPDATE
		SET blocked_keywords = EXCLUDED.blocked_keywords,
			blocked_domains = EXCLUDED.blocked_domains,
			min_account_age_days = EXCLUDED.min_account_age_days,
			post_rate_limit_per_hour = EXCLUDED.post_rate_limit_per_hour,
			comment_rate_limit_per_hour = EXCLUDED.comment_rate_limit_per_hour,
			block_new_accounts = EXCLUDED.block_new_accounts,
			filter_links = EXCLUDED.filter_links,
			updated_by = EXCLUDED.updated_by,
			updated_at = EXCLUDED.updated_at
		RETURNING community_id::text, blocked_keywords, blocked_domains, min_account_age_days, post_rate_limit_per_hour, comment_rate_limit_per_hour, block_new_accounts, filter_links, updated_by::text, updated_at
	`
	controls, err := scanContentControls(tx.QueryRow(ctx, query, input.CommunityID.String(), string(keywordsJSON), string(domainsJSON), input.MinAccountAgeDays, input.PostRateLimitPerHour, input.CommentRateLimitPerHour, input.BlockNewAccounts, input.FilterLinks, input.ActorID.String(), input.UpdatedAt))
	if err != nil {
		return moderationusecase.ContentControls{}, mapPostgresWriteError("upsert content controls", err)
	}
	if err := insertCommunityModLog(ctx, tx, input.CommunityID, moderationusecase.ApplyModerationActionRecordInput{
		ActorID:    input.ActorID,
		TargetType: moderationdomain.TargetType("content_controls"),
		TargetID:   input.CommunityID.String(),
		Action:     moderationdomain.ActionType("update_content_controls"),
		CreatedAt:  input.UpdatedAt,
	}, contentControlsState(before), contentControlsState(controls)); err != nil {
		return moderationusecase.ContentControls{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return moderationusecase.ContentControls{}, fmt.Errorf("commit content controls transaction: %w", err)
	}
	committed = true
	return controls, nil
}

func scanAutomodConfig(row rowScanner) (moderationusecase.AutomodConfig, error) {
	var config moderationusecase.AutomodConfig
	var rules []byte
	if err := row.Scan(&config.CommunityID, &config.ConfigText, &rules, &config.Version, &config.UpdatedBy, &config.UpdatedAt); err != nil {
		return moderationusecase.AutomodConfig{}, err
	}
	config.Rules = json.RawMessage(rules)
	if len(config.Rules) == 0 {
		config.Rules = json.RawMessage("{}")
	}
	return config, nil
}

func scanAutomodVersion(row rowScanner) (moderationusecase.AutomodVersion, error) {
	var version moderationusecase.AutomodVersion
	var rules []byte
	if err := row.Scan(&version.ID, &version.CommunityID, &version.Version, &version.ConfigText, &rules, &version.UpdatedBy, &version.CreatedAt); err != nil {
		return moderationusecase.AutomodVersion{}, err
	}
	version.Rules = json.RawMessage(rules)
	if len(version.Rules) == 0 {
		version.Rules = json.RawMessage("{}")
	}
	return version, nil
}

func scanContentControls(row rowScanner) (moderationusecase.ContentControls, error) {
	var controls moderationusecase.ContentControls
	var keywordsJSON []byte
	var domainsJSON []byte
	if err := row.Scan(
		&controls.CommunityID,
		&keywordsJSON,
		&domainsJSON,
		&controls.MinAccountAgeDays,
		&controls.PostRateLimitPerHour,
		&controls.CommentRateLimitPerHour,
		&controls.BlockNewAccounts,
		&controls.FilterLinks,
		&controls.UpdatedBy,
		&controls.UpdatedAt,
	); err != nil {
		return moderationusecase.ContentControls{}, err
	}
	if err := json.Unmarshal(keywordsJSON, &controls.BlockedKeywords); err != nil {
		return moderationusecase.ContentControls{}, err
	}
	if err := json.Unmarshal(domainsJSON, &controls.BlockedDomains); err != nil {
		return moderationusecase.ContentControls{}, err
	}
	if controls.BlockedKeywords == nil {
		controls.BlockedKeywords = []string{}
	}
	if controls.BlockedDomains == nil {
		controls.BlockedDomains = []string{}
	}
	return controls, nil
}

func contentControlsState(controls moderationusecase.ContentControls) map[string]any {
	return map[string]any{
		"blocked_keywords":            controls.BlockedKeywords,
		"blocked_domains":             controls.BlockedDomains,
		"min_account_age_days":        controls.MinAccountAgeDays,
		"post_rate_limit_per_hour":    controls.PostRateLimitPerHour,
		"comment_rate_limit_per_hour": controls.CommentRateLimitPerHour,
		"block_new_accounts":          controls.BlockNewAccounts,
		"filter_links":                controls.FilterLinks,
	}
}

func (repo *PostgresModerationRepository) ListModmailConversations(ctx context.Context, input moderationusecase.ListModmailConversationsRecordInput) ([]moderationusecase.ModmailConversation, error) {
	const query = `
		SELECT
			c.id::text, c.community_id::text, c.subject, c.user_id::text, c.status, c.folder,
			COALESCE(c.assigned_to::text, ''), c.last_message_at,
			(
				SELECT COUNT(*)::int
				FROM community_modmail_messages m
				LEFT JOIN community_modmail_reads r ON r.conversation_id = c.id AND r.user_id = $3::uuid
				WHERE m.conversation_id = c.id
					AND m.author_id <> $3::uuid
					AND (r.read_at IS NULL OR m.created_at > r.read_at)
			) AS unread_count,
			c.created_at, c.updated_at
		FROM community_modmail_conversations c
		WHERE c.community_id = $1::uuid
			AND c.folder = $2
		ORDER BY c.last_message_at DESC, c.id DESC
		LIMIT $4
		OFFSET $5
	`
	rows, err := repo.pool.Query(ctx, query, input.CommunityID.String(), input.Folder, input.ActorID.String(), input.Limit, input.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	conversations := make([]moderationusecase.ModmailConversation, 0)
	for rows.Next() {
		conversation, err := scanModmailConversation(rows)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, conversation)
	}
	return conversations, rows.Err()
}

func (repo *PostgresModerationRepository) CreateModmailConversation(ctx context.Context, input moderationusecase.CreateModmailConversationRecordInput) (moderationusecase.ModmailConversationDetail, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return moderationusecase.ModmailConversationDetail{}, fmt.Errorf("begin modmail conversation transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	const insertConversation = `
		INSERT INTO community_modmail_conversations (
			id, community_id, user_id, subject, status, folder, created_by, last_message_at, created_at, updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, 'open', 'inbox', $5::uuid, $6, $6, $6)
	`
	if _, err := tx.Exec(ctx, insertConversation, input.ID, input.CommunityID.String(), input.UserID.String(), input.Subject, input.ActorID.String(), input.CreatedAt); err != nil {
		return moderationusecase.ModmailConversationDetail{}, mapPostgresWriteError("insert modmail conversation", err)
	}
	if err := insertModmailMessage(ctx, tx, input.MessageID, input.ID, input.ActorID.String(), input.Body, false, input.CreatedAt); err != nil {
		return moderationusecase.ModmailConversationDetail{}, err
	}
	if err := upsertModmailRead(ctx, tx, input.ID, input.ActorID.String(), input.CreatedAt); err != nil {
		return moderationusecase.ModmailConversationDetail{}, err
	}
	if err := insertCommunityModLog(ctx, tx, input.CommunityID, moderationusecase.ApplyModerationActionRecordInput{
		ActorID:    input.ActorID,
		TargetType: moderationdomain.TargetType("modmail_conversation"),
		TargetID:   input.ID,
		Action:     moderationdomain.ActionType("create_modmail_conversation"),
		CreatedAt:  input.CreatedAt,
	}, map[string]any{}, map[string]any{"subject": input.Subject, "user_id": input.UserID.String()}); err != nil {
		return moderationusecase.ModmailConversationDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return moderationusecase.ModmailConversationDetail{}, fmt.Errorf("commit modmail conversation transaction: %w", err)
	}
	committed = true
	return repo.GetModmailConversation(ctx, moderationusecase.GetModmailConversationRecordInput{
		CommunityID:    input.CommunityID,
		ActorID:        input.ActorID,
		ConversationID: input.ID,
	})
}

func (repo *PostgresModerationRepository) GetModmailConversation(ctx context.Context, input moderationusecase.GetModmailConversationRecordInput) (moderationusecase.ModmailConversationDetail, error) {
	const conversationQuery = `
		SELECT
			c.id::text, c.community_id::text, c.subject, c.user_id::text, c.status, c.folder,
			COALESCE(c.assigned_to::text, ''), c.last_message_at,
			(
				SELECT COUNT(*)::int
				FROM community_modmail_messages m
				LEFT JOIN community_modmail_reads r ON r.conversation_id = c.id AND r.user_id = $3::uuid
				WHERE m.conversation_id = c.id
					AND m.author_id <> $3::uuid
					AND (r.read_at IS NULL OR m.created_at > r.read_at)
			) AS unread_count,
			c.created_at, c.updated_at
		FROM community_modmail_conversations c
		WHERE c.community_id = $1::uuid
			AND c.id = $2::uuid
	`
	conversation, err := scanModmailConversation(repo.pool.QueryRow(ctx, conversationQuery, input.CommunityID.String(), input.ConversationID, input.ActorID.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return moderationusecase.ModmailConversationDetail{}, apperr.New(apperr.CodeNotFound, "modmail conversation not found")
		}
		return moderationusecase.ModmailConversationDetail{}, err
	}
	messages, err := repo.listModmailMessages(ctx, input.ConversationID)
	if err != nil {
		return moderationusecase.ModmailConversationDetail{}, err
	}
	return moderationusecase.ModmailConversationDetail{Conversation: conversation, Messages: messages}, nil
}

func (repo *PostgresModerationRepository) AddModmailMessage(ctx context.Context, input moderationusecase.AddModmailMessageRecordInput) (moderationusecase.ModmailConversationDetail, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return moderationusecase.ModmailConversationDetail{}, fmt.Errorf("begin modmail message transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := ensureModmailConversation(ctx, tx, input.CommunityID, input.ConversationID); err != nil {
		return moderationusecase.ModmailConversationDetail{}, err
	}
	if err := insertModmailMessage(ctx, tx, input.ID, input.ConversationID, input.ActorID.String(), input.Body, input.IsInternal, input.CreatedAt); err != nil {
		return moderationusecase.ModmailConversationDetail{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE community_modmail_conversations
		SET last_message_at = $3, updated_at = $3
		WHERE community_id = $1::uuid AND id = $2::uuid
	`, input.CommunityID.String(), input.ConversationID, input.CreatedAt); err != nil {
		return moderationusecase.ModmailConversationDetail{}, mapPostgresWriteError("update modmail conversation timestamp", err)
	}
	if err := upsertModmailRead(ctx, tx, input.ConversationID, input.ActorID.String(), input.CreatedAt); err != nil {
		return moderationusecase.ModmailConversationDetail{}, err
	}
	action := "add_modmail_message"
	if input.IsInternal {
		action = "add_modmail_internal_note"
	}
	if err := insertCommunityModLog(ctx, tx, input.CommunityID, moderationusecase.ApplyModerationActionRecordInput{
		ActorID:    input.ActorID,
		TargetType: moderationdomain.TargetType("modmail_conversation"),
		TargetID:   input.ConversationID,
		Action:     moderationdomain.ActionType(action),
		CreatedAt:  input.CreatedAt,
	}, map[string]any{}, map[string]any{"message_id": input.ID, "is_internal": input.IsInternal}); err != nil {
		return moderationusecase.ModmailConversationDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return moderationusecase.ModmailConversationDetail{}, fmt.Errorf("commit modmail message transaction: %w", err)
	}
	committed = true
	return repo.GetModmailConversation(ctx, moderationusecase.GetModmailConversationRecordInput{
		CommunityID:    input.CommunityID,
		ActorID:        input.ActorID,
		ConversationID: input.ConversationID,
	})
}

func (repo *PostgresModerationRepository) UpdateModmailConversation(ctx context.Context, input moderationusecase.UpdateModmailConversationRecordInput) (moderationusecase.ModmailConversation, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return moderationusecase.ModmailConversation{}, fmt.Errorf("begin modmail update transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	before, err := scanModmailConversation(tx.QueryRow(ctx, `
		SELECT id::text, community_id::text, subject, user_id::text, status, folder, COALESCE(assigned_to::text, ''), last_message_at, 0, created_at, updated_at
		FROM community_modmail_conversations
		WHERE community_id = $1::uuid AND id = $2::uuid
		FOR UPDATE
	`, input.CommunityID.String(), input.ConversationID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return moderationusecase.ModmailConversation{}, apperr.New(apperr.CodeNotFound, "modmail conversation not found")
		}
		return moderationusecase.ModmailConversation{}, err
	}
	assignedTo := any(nil)
	if input.AssignedTo != "" {
		assignedTo = input.AssignedTo
	}
	const query = `
		UPDATE community_modmail_conversations
		SET folder = $3,
			status = $4,
			assigned_to = $5::uuid,
			updated_at = $6
		WHERE community_id = $1::uuid AND id = $2::uuid
		RETURNING id::text, community_id::text, subject, user_id::text, status, folder, COALESCE(assigned_to::text, ''), last_message_at, 0, created_at, updated_at
	`
	conversation, err := scanModmailConversation(tx.QueryRow(ctx, query, input.CommunityID.String(), input.ConversationID, input.Folder, input.Status, assignedTo, input.UpdatedAt))
	if err != nil {
		return moderationusecase.ModmailConversation{}, mapPostgresWriteError("update modmail conversation", err)
	}
	if input.MarkRead {
		if err := upsertModmailRead(ctx, tx, input.ConversationID, input.ActorID.String(), input.UpdatedAt); err != nil {
			return moderationusecase.ModmailConversation{}, err
		}
	}
	if err := insertCommunityModLog(ctx, tx, input.CommunityID, moderationusecase.ApplyModerationActionRecordInput{
		ActorID:    input.ActorID,
		TargetType: moderationdomain.TargetType("modmail_conversation"),
		TargetID:   input.ConversationID,
		Action:     moderationdomain.ActionType("update_modmail_conversation"),
		CreatedAt:  input.UpdatedAt,
	}, modmailConversationState(before), modmailConversationState(conversation)); err != nil {
		return moderationusecase.ModmailConversation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return moderationusecase.ModmailConversation{}, fmt.Errorf("commit modmail update transaction: %w", err)
	}
	committed = true
	detail, err := repo.GetModmailConversation(ctx, moderationusecase.GetModmailConversationRecordInput{
		CommunityID:    input.CommunityID,
		ActorID:        input.ActorID,
		ConversationID: input.ConversationID,
	})
	if err != nil {
		return moderationusecase.ModmailConversation{}, err
	}
	return detail.Conversation, nil
}

func scanModmailConversation(row rowScanner) (moderationusecase.ModmailConversation, error) {
	var conversation moderationusecase.ModmailConversation
	if err := row.Scan(
		&conversation.ID,
		&conversation.CommunityID,
		&conversation.Subject,
		&conversation.UserID,
		&conversation.Status,
		&conversation.Folder,
		&conversation.AssignedTo,
		&conversation.LastMessageAt,
		&conversation.UnreadCount,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	); err != nil {
		return moderationusecase.ModmailConversation{}, err
	}
	return conversation, nil
}

func (repo *PostgresModerationRepository) listModmailMessages(ctx context.Context, conversationID string) ([]moderationusecase.ModmailMessage, error) {
	rows, err := repo.pool.Query(ctx, `
		SELECT id::text, conversation_id::text, author_id::text, body, is_internal, created_at
		FROM community_modmail_messages
		WHERE conversation_id = $1::uuid
		ORDER BY created_at ASC, id ASC
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := make([]moderationusecase.ModmailMessage, 0)
	for rows.Next() {
		var message moderationusecase.ModmailMessage
		if err := rows.Scan(&message.ID, &message.ConversationID, &message.AuthorID, &message.Body, &message.IsInternal, &message.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func ensureModmailConversation(ctx context.Context, tx pgx.Tx, communityID communitydomain.CommunityID, conversationID string) error {
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM community_modmail_conversations
			WHERE community_id = $1::uuid AND id = $2::uuid
		)
	`, communityID.String(), conversationID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return apperr.New(apperr.CodeNotFound, "modmail conversation not found")
	}
	return nil
}

func insertModmailMessage(ctx context.Context, db postgresExecutor, id string, conversationID string, authorID string, body string, isInternal bool, createdAt time.Time) error {
	_, err := db.Exec(ctx, `
		INSERT INTO community_modmail_messages (id, conversation_id, author_id, body, is_internal, created_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6)
	`, id, conversationID, authorID, body, isInternal, createdAt)
	if err != nil {
		return mapPostgresWriteError("insert modmail message", err)
	}
	return nil
}

func upsertModmailRead(ctx context.Context, db postgresExecutor, conversationID string, userID string, readAt time.Time) error {
	_, err := db.Exec(ctx, `
		INSERT INTO community_modmail_reads (conversation_id, user_id, read_at)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (conversation_id, user_id) DO UPDATE
		SET read_at = EXCLUDED.read_at
	`, conversationID, userID, readAt)
	return err
}

func modmailConversationState(conversation moderationusecase.ModmailConversation) map[string]any {
	return map[string]any{
		"status":      conversation.Status,
		"folder":      conversation.Folder,
		"assigned_to": conversation.AssignedTo,
	}
}

func (repo *PostgresModerationRepository) GetCommunityInsightsSummary(ctx context.Context, input moderationusecase.CommunityInsightsRecordInput) (moderationusecase.CommunityInsightsSummary, error) {
	const query = `
		SELECT
			(SELECT COUNT(*)::int FROM community_memberships WHERE community_id = $1::uuid AND status = 'active') AS members_total,
			(SELECT COUNT(*)::int FROM posts WHERE community_id = $1::uuid AND created_at >= $2) AS posts_created,
			(
				SELECT COUNT(*)::int
				FROM comments
				JOIN posts ON posts.id = comments.post_id
				WHERE posts.community_id = $1::uuid AND comments.created_at >= $2
			) AS comments_made,
			(
				SELECT COUNT(*)::int
				FROM (
					SELECT posts.author_id FROM posts WHERE posts.community_id = $1::uuid AND posts.created_at >= $2
					UNION
					SELECT comments.author_id
					FROM comments
					JOIN posts ON posts.id = comments.post_id
					WHERE posts.community_id = $1::uuid AND comments.created_at >= $2
				) authors
			) AS active_authors
	`
	summary := moderationusecase.CommunityInsightsSummary{
		CommunityID: input.CommunityID.String(),
		Range:       input.Range,
		Since:       input.Since,
	}
	if err := repo.pool.QueryRow(ctx, query, input.CommunityID.String(), input.Since).Scan(&summary.MembersTotal, &summary.PostsCreated, &summary.CommentsMade, &summary.ActiveAuthors); err != nil {
		return moderationusecase.CommunityInsightsSummary{}, err
	}
	return summary, nil
}

func (repo *PostgresModerationRepository) GetCommunityModerationInsights(ctx context.Context, input moderationusecase.CommunityInsightsRecordInput) (moderationusecase.CommunityModerationInsights, error) {
	const query = `
		SELECT
			(
				SELECT COUNT(*)::int
				FROM content_reports
				WHERE status = 'pending'
					AND EXISTS (
						SELECT 1 FROM posts WHERE posts.id = content_reports.post_id AND posts.community_id = $1::uuid
						UNION ALL
						SELECT 1 FROM comments JOIN posts ON posts.id = comments.post_id WHERE comments.id = content_reports.comment_id AND posts.community_id = $1::uuid
					)
			) AS pending_reports,
			(
				SELECT COUNT(*)::int
				FROM content_reports
				WHERE status <> 'pending'
					AND updated_at >= $2
					AND EXISTS (
						SELECT 1 FROM posts WHERE posts.id = content_reports.post_id AND posts.community_id = $1::uuid
						UNION ALL
						SELECT 1 FROM comments JOIN posts ON posts.id = comments.post_id WHERE comments.id = content_reports.comment_id AND posts.community_id = $1::uuid
					)
			) AS resolved_reports,
			(SELECT COUNT(*)::int FROM posts WHERE community_id = $1::uuid AND status = 'removed' AND updated_at >= $2) AS removed_posts,
			(
				SELECT COUNT(*)::int FROM comments
				JOIN posts ON posts.id = comments.post_id
				WHERE posts.community_id = $1::uuid AND comments.status = 'removed' AND comments.updated_at >= $2
			) AS removed_comments,
			(SELECT COUNT(*)::int FROM posts WHERE community_id = $1::uuid AND status = 'spam' AND updated_at >= $2) AS spam_posts,
			(
				SELECT COUNT(*)::int FROM comments
				JOIN posts ON posts.id = comments.post_id
				WHERE posts.community_id = $1::uuid AND comments.status = 'spam' AND comments.updated_at >= $2
			) AS spam_comments,
			(
				SELECT COUNT(*)::int FROM community_moderation_logs
				WHERE community_id = $1::uuid AND created_at >= $2
			) AS actions_count
	`
	insights := moderationusecase.CommunityModerationInsights{
		CommunityID: input.CommunityID.String(),
		Range:       input.Range,
		Since:       input.Since,
	}
	if err := repo.pool.QueryRow(ctx, query, input.CommunityID.String(), input.Since).Scan(
		&insights.PendingReports,
		&insights.ResolvedReports,
		&insights.RemovedPosts,
		&insights.RemovedComments,
		&insights.SpamPosts,
		&insights.SpamComments,
		&insights.ActionsCount,
	); err != nil {
		return moderationusecase.CommunityModerationInsights{}, err
	}
	return insights, nil
}

func (repo *PostgresModerationRepository) listReportQueue(ctx context.Context, communityID any, queue string, limit int, offset int) ([]moderationusecase.ModQueueItem, error) {
	const query = `
		WITH report_targets AS (
			SELECT
				'post' AS target_type,
				posts.id::text AS target_id,
				posts.id::text AS post_id,
				posts.community_id::text AS community_id,
				communities.slug AS community_slug,
				posts.author_id::text AS author_id,
				COUNT(content_reports.id)::int AS report_count,
				posts.status AS status,
				LEFT(posts.title || ' ' || posts.body, 160) AS preview,
				MIN(content_reports.created_at) AS created_at,
				MAX(content_reports.updated_at) AS updated_at
			FROM content_reports
			JOIN posts ON posts.id = content_reports.post_id
			JOIN communities ON communities.id = posts.community_id
			WHERE content_reports.status = 'pending'
				AND ($1::uuid IS NULL OR communities.id = $1::uuid)
			GROUP BY posts.id, communities.slug
			UNION ALL
			SELECT
				'comment' AS target_type,
				comments.id::text AS target_id,
				comments.post_id::text AS post_id,
				posts.community_id::text AS community_id,
				communities.slug AS community_slug,
				comments.author_id::text AS author_id,
				COUNT(content_reports.id)::int AS report_count,
				comments.status AS status,
				LEFT(comments.body, 160) AS preview,
				MIN(content_reports.created_at) AS created_at,
				MAX(content_reports.updated_at) AS updated_at
			FROM content_reports
			JOIN comments ON comments.id = content_reports.comment_id
			JOIN posts ON posts.id = comments.post_id
			JOIN communities ON communities.id = posts.community_id
			WHERE content_reports.status = 'pending'
				AND ($1::uuid IS NULL OR communities.id = $1::uuid)
			GROUP BY comments.id, posts.community_id, communities.slug
		)
		SELECT target_type, target_id, post_id, community_id, community_slug, author_id, report_count, $2, status, preview, created_at, updated_at
		FROM report_targets
		ORDER BY updated_at DESC, target_id DESC
		LIMIT $3
		OFFSET $4
	`
	return repo.queryModQueueItems(ctx, query, communityID, queue, limit, offset)
}

func (repo *PostgresModerationRepository) listContentQueue(ctx context.Context, communityID any, queue string, limit int, offset int) ([]moderationusecase.ModQueueItem, error) {
	statusPredicate := "posts.status = 'visible'"
	commentStatusPredicate := "comments.status = 'visible'"
	extraPostPredicate := ""
	extraCommentPredicate := ""
	switch queue {
	case "removed":
		statusPredicate = "posts.status = 'removed'"
		commentStatusPredicate = "comments.status = 'removed'"
	case "spam":
		statusPredicate = "posts.status = 'spam'"
		commentStatusPredicate = "comments.status = 'spam'"
	case "edited":
		extraPostPredicate = "AND posts.updated_at > posts.created_at"
		extraCommentPredicate = "AND comments.updated_at > comments.created_at"
	case "unmoderated":
		extraPostPredicate = "AND NOT EXISTS (SELECT 1 FROM moderation_actions ma WHERE ma.post_id = posts.id)"
		extraCommentPredicate = "AND NOT EXISTS (SELECT 1 FROM moderation_actions ma WHERE ma.comment_id = comments.id)"
	default:
		return nil, apperr.New(apperr.CodeInvalidArgument, "moderation queue is invalid")
	}
	query := fmt.Sprintf(`
		WITH queue_targets AS (
			SELECT
				'post' AS target_type,
				posts.id::text AS target_id,
				posts.id::text AS post_id,
				posts.community_id::text AS community_id,
				communities.slug AS community_slug,
				posts.author_id::text AS author_id,
				0::int AS report_count,
				posts.status AS status,
				LEFT(posts.title || ' ' || posts.body, 160) AS preview,
				posts.created_at,
				posts.updated_at
			FROM posts
			JOIN communities ON communities.id = posts.community_id
			WHERE %s
				AND ($1::uuid IS NULL OR communities.id = $1::uuid)
				%s
			UNION ALL
			SELECT
				'comment' AS target_type,
				comments.id::text AS target_id,
				comments.post_id::text AS post_id,
				posts.community_id::text AS community_id,
				communities.slug AS community_slug,
				comments.author_id::text AS author_id,
				0::int AS report_count,
				comments.status AS status,
				LEFT(comments.body, 160) AS preview,
				comments.created_at,
				comments.updated_at
			FROM comments
			JOIN posts ON posts.id = comments.post_id
			JOIN communities ON communities.id = posts.community_id
			WHERE %s
				AND ($1::uuid IS NULL OR communities.id = $1::uuid)
				%s
		)
		SELECT target_type, target_id, post_id, community_id, community_slug, author_id, report_count, $2, status, preview, created_at, updated_at
		FROM queue_targets
		ORDER BY updated_at DESC, target_id DESC
		LIMIT $3
		OFFSET $4
	`, statusPredicate, extraPostPredicate, commentStatusPredicate, extraCommentPredicate)
	return repo.queryModQueueItems(ctx, query, communityID, queue, limit, offset)
}

func (repo *PostgresModerationRepository) queryModQueueItems(ctx context.Context, query string, args ...any) ([]moderationusecase.ModQueueItem, error) {
	rows, err := repo.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query mod queue: %w", err)
	}
	defer rows.Close()

	items := make([]moderationusecase.ModQueueItem, 0)
	for rows.Next() {
		var item moderationusecase.ModQueueItem
		if err := rows.Scan(
			&item.TargetType,
			&item.TargetID,
			&item.PostID,
			&item.CommunityID,
			&item.CommunitySlug,
			&item.AuthorID,
			&item.ReportCount,
			&item.Queue,
			&item.Status,
			&item.Preview,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.ID = item.TargetType + ":" + item.TargetID
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mod queue: %w", err)
	}
	return items, nil
}

func (repo *PostgresModerationRepository) findModQueueItem(ctx context.Context, communityID any, targetType moderationdomain.TargetType, targetID string) (moderationusecase.ModQueueItem, moderationusecase.ReportTargetPreview, error) {
	switch targetType {
	case moderationdomain.TargetTypePost:
		const query = `
			WITH target AS (
				SELECT
					posts.id::text AS target_id,
					posts.id::text AS post_id,
					posts.community_id::text AS community_id,
					communities.slug AS community_slug,
					posts.author_id::text AS author_id,
					posts.status,
					LEFT(posts.title || ' ' || posts.body, 160) AS preview,
					posts.title,
					LEFT(posts.body, 300) AS body_excerpt,
					posts.created_at AS target_created_at,
					posts.updated_at AS target_updated_at,
					(SELECT COUNT(*)::int FROM content_reports WHERE content_reports.post_id = posts.id AND content_reports.status = 'pending') AS report_count,
					(SELECT MIN(content_reports.created_at) FROM content_reports WHERE content_reports.post_id = posts.id AND content_reports.status = 'pending') AS first_reported_at,
					(SELECT MAX(content_reports.updated_at) FROM content_reports WHERE content_reports.post_id = posts.id AND content_reports.status = 'pending') AS last_reported_at,
					EXISTS (SELECT 1 FROM moderation_actions WHERE moderation_actions.post_id = posts.id) AS has_action
				FROM posts
				JOIN communities ON communities.id = posts.community_id
				WHERE posts.id = $1::uuid
					AND ($2::uuid IS NULL OR posts.community_id = $2::uuid)
			)
			SELECT
				'post',
				target_id,
				post_id,
				community_id,
				community_slug,
				author_id,
				report_count,
				CASE
					WHEN report_count > 0 THEN 'reports'
					WHEN status = 'spam' THEN 'spam'
					WHEN status = 'removed' THEN 'removed'
					WHEN target_updated_at > target_created_at THEN 'edited'
					WHEN NOT has_action AND status = 'visible' THEN 'unmoderated'
					ELSE ''
				END,
				status,
				preview,
				COALESCE(first_reported_at, target_created_at),
				COALESCE(last_reported_at, target_updated_at),
				title,
				body_excerpt,
				target_created_at,
				target_updated_at
			FROM target
		`
		return scanModQueueDetail(repo.pool.QueryRow(ctx, query, targetID, communityID))
	case moderationdomain.TargetTypeComment:
		const query = `
			WITH target AS (
				SELECT
					comments.id::text AS target_id,
					comments.post_id::text AS post_id,
					posts.community_id::text AS community_id,
					communities.slug AS community_slug,
					comments.author_id::text AS author_id,
					comments.status,
					LEFT(comments.body, 160) AS preview,
					'' AS title,
					LEFT(comments.body, 300) AS body_excerpt,
					comments.created_at AS target_created_at,
					comments.updated_at AS target_updated_at,
					(SELECT COUNT(*)::int FROM content_reports WHERE content_reports.comment_id = comments.id AND content_reports.status = 'pending') AS report_count,
					(SELECT MIN(content_reports.created_at) FROM content_reports WHERE content_reports.comment_id = comments.id AND content_reports.status = 'pending') AS first_reported_at,
					(SELECT MAX(content_reports.updated_at) FROM content_reports WHERE content_reports.comment_id = comments.id AND content_reports.status = 'pending') AS last_reported_at,
					EXISTS (SELECT 1 FROM moderation_actions WHERE moderation_actions.comment_id = comments.id) AS has_action
				FROM comments
				JOIN posts ON posts.id = comments.post_id
				JOIN communities ON communities.id = posts.community_id
				WHERE comments.id = $1::uuid
					AND ($2::uuid IS NULL OR posts.community_id = $2::uuid)
			)
			SELECT
				'comment',
				target_id,
				post_id,
				community_id,
				community_slug,
				author_id,
				report_count,
				CASE
					WHEN report_count > 0 THEN 'reports'
					WHEN status = 'spam' THEN 'spam'
					WHEN status = 'removed' THEN 'removed'
					WHEN target_updated_at > target_created_at THEN 'edited'
					WHEN NOT has_action AND status = 'visible' THEN 'unmoderated'
					ELSE ''
				END,
				status,
				preview,
				COALESCE(first_reported_at, target_created_at),
				COALESCE(last_reported_at, target_updated_at),
				title,
				body_excerpt,
				target_created_at,
				target_updated_at
			FROM target
		`
		return scanModQueueDetail(repo.pool.QueryRow(ctx, query, targetID, communityID))
	default:
		return moderationusecase.ModQueueItem{}, moderationusecase.ReportTargetPreview{}, apperr.New(apperr.CodeInvalidArgument, "moderation target type is invalid")
	}
}

func scanModQueueDetail(row rowScanner) (moderationusecase.ModQueueItem, moderationusecase.ReportTargetPreview, error) {
	var item moderationusecase.ModQueueItem
	var preview moderationusecase.ReportTargetPreview
	if err := row.Scan(
		&item.TargetType,
		&item.TargetID,
		&item.PostID,
		&item.CommunityID,
		&item.CommunitySlug,
		&item.AuthorID,
		&item.ReportCount,
		&item.Queue,
		&item.Status,
		&item.Preview,
		&item.CreatedAt,
		&item.UpdatedAt,
		&preview.Title,
		&preview.BodyExcerpt,
		&preview.CreatedAt,
		&preview.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return moderationusecase.ModQueueItem{}, moderationusecase.ReportTargetPreview{}, apperr.New(apperr.CodeNotFound, "moderation queue item not found")
		}
		return moderationusecase.ModQueueItem{}, moderationusecase.ReportTargetPreview{}, err
	}
	if item.Queue == "" {
		return moderationusecase.ModQueueItem{}, moderationusecase.ReportTargetPreview{}, apperr.New(apperr.CodeNotFound, "moderation queue item not found")
	}
	item.ID = item.TargetType + ":" + item.TargetID
	preview.TargetType = item.TargetType
	preview.PostID = item.PostID
	if item.TargetType == moderationdomain.TargetTypeComment.String() {
		preview.CommentID = item.TargetID
	}
	preview.AuthorID = item.AuthorID
	preview.Status = item.Status
	return item, preview, nil
}

func (repo *PostgresModerationRepository) listReportsForModQueueTarget(ctx context.Context, targetType moderationdomain.TargetType, targetID string) ([]moderationusecase.ModQueueReport, error) {
	predicate := "post_id = $1::uuid"
	if targetType == moderationdomain.TargetTypeComment {
		predicate = "comment_id = $1::uuid"
	}
	query := fmt.Sprintf(`
		SELECT id::text, reporter_id::text, reason, status, created_at
		FROM content_reports
		WHERE %s
		ORDER BY created_at DESC, id DESC
		LIMIT 20
	`, predicate)
	rows, err := repo.pool.Query(ctx, query, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	reports := make([]moderationusecase.ModQueueReport, 0)
	for rows.Next() {
		var report moderationusecase.ModQueueReport
		if err := rows.Scan(&report.ID, &report.ReporterID, &report.Reason, &report.Status, &report.CreatedAt); err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

func (repo *PostgresModerationRepository) listActionsForModQueueTarget(ctx context.Context, targetType moderationdomain.TargetType, targetID string) ([]moderationusecase.ModerationAction, error) {
	predicate := "post_id = $1::uuid"
	if targetType == moderationdomain.TargetTypeComment {
		predicate = "comment_id = $1::uuid"
	}
	query := fmt.Sprintf(`
		SELECT id::text, COALESCE(post_id::text, ''), COALESCE(comment_id::text, ''), actor_id::text, action, reason, created_at
		FROM moderation_actions
		WHERE %s
		ORDER BY created_at DESC, id DESC
		LIMIT 10
	`, predicate)
	rows, err := repo.pool.Query(ctx, query, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	actions := make([]moderationusecase.ModerationAction, 0)
	for rows.Next() {
		var action moderationusecase.ModerationAction
		if err := rows.Scan(&action.ID, &action.PostID, &action.CommentID, &action.ActorID, &action.Action, &action.Reason, &action.CreatedAt); err != nil {
			return nil, err
		}
		if action.CommentID != "" {
			action.TargetType = moderationdomain.TargetTypeComment.String()
		} else {
			action.TargetType = moderationdomain.TargetTypePost.String()
		}
		actions = append(actions, action)
	}
	return actions, rows.Err()
}

func (repo *PostgresModerationRepository) countModQueues(ctx context.Context, communityID any) ([]moderationusecase.ModQueueCount, error) {
	const query = `
		WITH report_targets AS (
			SELECT 'post' AS target_type, posts.id AS target_id
			FROM content_reports
			JOIN posts ON posts.id = content_reports.post_id
			WHERE content_reports.status = 'pending'
				AND ($1::uuid IS NULL OR posts.community_id = $1::uuid)
			GROUP BY posts.id
			UNION ALL
			SELECT 'comment' AS target_type, comments.id AS target_id
			FROM content_reports
			JOIN comments ON comments.id = content_reports.comment_id
			JOIN posts ON posts.id = comments.post_id
			WHERE content_reports.status = 'pending'
				AND ($1::uuid IS NULL OR posts.community_id = $1::uuid)
			GROUP BY comments.id
		),
		content_counts AS (
			SELECT 'spam' AS queue, COUNT(*)::int AS count
			FROM (
				SELECT posts.id FROM posts WHERE posts.status = 'spam' AND ($1::uuid IS NULL OR posts.community_id = $1::uuid)
				UNION ALL
				SELECT comments.id FROM comments JOIN posts ON posts.id = comments.post_id WHERE comments.status = 'spam' AND ($1::uuid IS NULL OR posts.community_id = $1::uuid)
			) items
			UNION ALL
			SELECT 'removed', COUNT(*)::int
			FROM (
				SELECT posts.id FROM posts WHERE posts.status = 'removed' AND ($1::uuid IS NULL OR posts.community_id = $1::uuid)
				UNION ALL
				SELECT comments.id FROM comments JOIN posts ON posts.id = comments.post_id WHERE comments.status = 'removed' AND ($1::uuid IS NULL OR posts.community_id = $1::uuid)
			) items
			UNION ALL
			SELECT 'edited', COUNT(*)::int
			FROM (
				SELECT posts.id FROM posts WHERE posts.status = 'visible' AND posts.updated_at > posts.created_at AND ($1::uuid IS NULL OR posts.community_id = $1::uuid)
				UNION ALL
				SELECT comments.id FROM comments JOIN posts ON posts.id = comments.post_id WHERE comments.status = 'visible' AND comments.updated_at > comments.created_at AND ($1::uuid IS NULL OR posts.community_id = $1::uuid)
			) items
			UNION ALL
			SELECT 'unmoderated', COUNT(*)::int
			FROM (
				SELECT posts.id FROM posts WHERE posts.status = 'visible' AND ($1::uuid IS NULL OR posts.community_id = $1::uuid) AND NOT EXISTS (SELECT 1 FROM moderation_actions ma WHERE ma.post_id = posts.id)
				UNION ALL
				SELECT comments.id FROM comments JOIN posts ON posts.id = comments.post_id WHERE comments.status = 'visible' AND ($1::uuid IS NULL OR posts.community_id = $1::uuid) AND NOT EXISTS (SELECT 1 FROM moderation_actions ma WHERE ma.comment_id = comments.id)
			) items
		)
		SELECT queue, count
		FROM (
			SELECT 'reports' AS queue, COUNT(*)::int AS count FROM report_targets
			UNION ALL
			SELECT 'needs_review', COUNT(*)::int FROM report_targets
			UNION ALL
			SELECT queue, count FROM content_counts
		) counts
		ORDER BY CASE queue
			WHEN 'reports' THEN 1
			WHEN 'spam' THEN 2
			WHEN 'removed' THEN 3
			WHEN 'edited' THEN 4
			WHEN 'unmoderated' THEN 5
			WHEN 'needs_review' THEN 6
			ELSE 99
		END
	`
	rows, err := repo.pool.Query(ctx, query, communityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make([]moderationusecase.ModQueueCount, 0)
	for rows.Next() {
		var count moderationusecase.ModQueueCount
		if err := rows.Scan(&count.Queue, &count.Count); err != nil {
			return nil, err
		}
		counts = append(counts, count)
	}
	return counts, rows.Err()
}

func (repo *PostgresModerationRepository) ApplyModerationAction(ctx context.Context, input moderationusecase.ApplyModerationActionRecordInput) (result moderationusecase.ModerationAction, err error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return moderationusecase.ModerationAction{}, fmt.Errorf("begin moderation action transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	target, before, communityID, err := lockModerationToolTarget(ctx, tx, input.TargetType, input.TargetID, input.ScopeCommunityID)
	if err != nil {
		return moderationusecase.ModerationAction{}, err
	}
	after, err := applyLockedTargetAction(ctx, tx, input, before)
	if err != nil {
		return moderationusecase.ModerationAction{}, err
	}
	action, err := moderationdomain.NewModerationAction(
		moderationdomain.NewGeneratedModerationActionID(),
		target,
		input.ActorID,
		input.Action,
		moderationdomain.Reason(input.Reason),
		input.CreatedAt,
	)
	if err != nil {
		return moderationusecase.ModerationAction{}, err
	}
	if err := insertModerationAction(ctx, tx, *action); err != nil {
		return moderationusecase.ModerationAction{}, err
	}
	switch input.Action {
	case moderationdomain.ActionTypeIgnoreReports:
		if err := dismissPendingReports(ctx, tx, *action); err != nil {
			return moderationusecase.ModerationAction{}, err
		}
	case moderationdomain.ActionTypeApprove, moderationdomain.ActionTypeRemove, moderationdomain.ActionTypeSpam:
		if err := resolvePendingReports(ctx, tx, *action); err != nil {
			return moderationusecase.ModerationAction{}, err
		}
	}
	if err := insertCommunityModLog(ctx, tx, communityID, input, before, after); err != nil {
		return moderationusecase.ModerationAction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return moderationusecase.ModerationAction{}, fmt.Errorf("commit moderation action transaction: %w", err)
	}
	committed = true
	return toModerationToolActionDTO(*action), nil
}

func (repo *PostgresModerationRepository) IgnoreCommunityReport(ctx context.Context, communityID communitydomain.CommunityID, reportID moderationdomain.ContentReportID, actorID userdomain.UserID, reviewedAt time.Time) (moderationdomain.ContentReport, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return moderationdomain.ContentReport{}, fmt.Errorf("begin ignore report transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	const query = `
		UPDATE content_reports
		SET status = 'dismissed',
			reviewed_by = $3::uuid,
			reviewed_at = $4,
			updated_at = $4
		WHERE content_reports.id = $1::uuid
			AND content_reports.status = 'pending'
			AND EXISTS (
				SELECT 1
				FROM posts
				WHERE posts.id = content_reports.post_id
					AND posts.community_id = $2::uuid
				UNION ALL
				SELECT 1
				FROM comments
				JOIN posts ON posts.id = comments.post_id
				WHERE comments.id = content_reports.comment_id
					AND posts.community_id = $2::uuid
			)
		RETURNING
			id::text,
			reporter_id::text,
			post_id::text,
			comment_id::text,
			reason,
			status,
			reviewed_by::text,
			reviewed_at,
			created_at,
			updated_at
	`
	report, err := scanContentReport(tx.QueryRow(ctx, query, reportID.String(), communityID.String(), actorID.String(), reviewedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return moderationdomain.ContentReport{}, apperr.New(apperr.CodeNotFound, "content report not found")
		}
		return moderationdomain.ContentReport{}, err
	}
	target := report.Target()
	var targetType moderationdomain.TargetType
	var targetID string
	if postID, ok := target.PostID(); ok {
		targetType = moderationdomain.TargetTypePost
		targetID = postID.String()
	} else if commentID, ok := target.CommentID(); ok {
		targetType = moderationdomain.TargetTypeComment
		targetID = commentID.String()
	} else {
		return moderationdomain.ContentReport{}, apperr.New(apperr.CodeInvalidArgument, "moderation target is required")
	}
	_, before, logCommunityID, err := lockModerationToolTarget(ctx, tx, targetType, targetID, &communityID)
	if err != nil {
		return moderationdomain.ContentReport{}, err
	}
	action, err := moderationdomain.NewModerationAction(
		moderationdomain.NewGeneratedModerationActionID(),
		target,
		actorID,
		moderationdomain.ActionTypeIgnoreReports,
		moderationdomain.Reason("ignored report"),
		reviewedAt,
	)
	if err != nil {
		return moderationdomain.ContentReport{}, err
	}
	if err := insertModerationAction(ctx, tx, *action); err != nil {
		return moderationdomain.ContentReport{}, err
	}
	if err := insertCommunityModLog(ctx, tx, logCommunityID, moderationusecase.ApplyModerationActionRecordInput{
		ScopeCommunityID: &communityID,
		BatchID:          uuid.NewString(),
		ActorID:          actorID,
		TargetType:       targetType,
		TargetID:         targetID,
		Action:           moderationdomain.ActionTypeIgnoreReports,
		Reason:           "ignored report",
		CreatedAt:        reviewedAt,
	}, before, cloneMap(before)); err != nil {
		return moderationdomain.ContentReport{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return moderationdomain.ContentReport{}, fmt.Errorf("commit ignore report transaction: %w", err)
	}
	committed = true
	return *report, nil
}

func toModerationToolActionDTO(action moderationdomain.ModerationAction) moderationusecase.ModerationAction {
	target := action.Target()
	postID := ""
	if id, ok := target.PostID(); ok {
		postID = id.String()
	}
	commentID := ""
	if id, ok := target.CommentID(); ok {
		commentID = id.String()
	}
	return moderationusecase.ModerationAction{
		ID:         action.ID().String(),
		TargetType: target.Type().String(),
		PostID:     postID,
		CommentID:  commentID,
		ActorID:    action.ActorID().String(),
		Action:     action.Action().String(),
		Reason:     action.Reason().String(),
		CreatedAt:  action.CreatedAt(),
	}
}

type moderationToolTargetState struct {
	TargetType  string                      `json:"target_type"`
	TargetID    string                      `json:"target_id"`
	PostID      string                      `json:"post_id,omitempty"`
	Status      string                      `json:"status"`
	IsLocked    bool                        `json:"is_locked"`
	IsPinned    bool                        `json:"is_pinned,omitempty"`
	IsNSFW      bool                        `json:"is_nsfw,omitempty"`
	IsSpoiler   bool                        `json:"is_spoiler,omitempty"`
	FlairText   string                      `json:"flair_text,omitempty"`
	CommunityID communitydomain.CommunityID `json:"-"`
}

func lockModerationToolTarget(ctx context.Context, tx pgx.Tx, targetType moderationdomain.TargetType, targetID string, scopeCommunityID *communitydomain.CommunityID) (moderationdomain.Target, map[string]any, communitydomain.CommunityID, error) {
	scopeID := nullableCommunityID(scopeCommunityID)
	switch targetType {
	case moderationdomain.TargetTypePost:
		const query = `
			SELECT posts.id::text, posts.community_id::text, posts.status, posts.is_locked, posts.is_pinned, posts.is_nsfw, posts.is_spoiler, posts.flair_text
			FROM posts
			WHERE posts.id = $1::uuid
				AND ($2::uuid IS NULL OR posts.community_id = $2::uuid)
			FOR UPDATE
		`
		var state moderationToolTargetState
		var rawCommunityID string
		if err := tx.QueryRow(ctx, query, targetID, scopeID).Scan(&state.TargetID, &rawCommunityID, &state.Status, &state.IsLocked, &state.IsPinned, &state.IsNSFW, &state.IsSpoiler, &state.FlairText); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return moderationdomain.Target{}, nil, "", apperr.New(apperr.CodeNotFound, "post not found")
			}
			return moderationdomain.Target{}, nil, "", err
		}
		communityID, err := communitydomain.NewCommunityID(rawCommunityID)
		if err != nil {
			return moderationdomain.Target{}, nil, "", err
		}
		postID, err := postdomain.NewPostID(state.TargetID)
		if err != nil {
			return moderationdomain.Target{}, nil, "", err
		}
		target, err := moderationdomain.NewPostTarget(postID)
		if err != nil {
			return moderationdomain.Target{}, nil, "", err
		}
		state.TargetType = moderationdomain.TargetTypePost.String()
		state.PostID = state.TargetID
		state.CommunityID = communityID
		return target, state.toMap(), communityID, nil
	case moderationdomain.TargetTypeComment:
		const query = `
			SELECT comments.id::text, comments.post_id::text, posts.community_id::text, comments.status, comments.is_locked
			FROM comments
			JOIN posts ON posts.id = comments.post_id
			WHERE comments.id = $1::uuid
				AND ($2::uuid IS NULL OR posts.community_id = $2::uuid)
			FOR UPDATE OF comments
		`
		var state moderationToolTargetState
		var rawCommunityID string
		if err := tx.QueryRow(ctx, query, targetID, scopeID).Scan(&state.TargetID, &state.PostID, &rawCommunityID, &state.Status, &state.IsLocked); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return moderationdomain.Target{}, nil, "", apperr.New(apperr.CodeNotFound, "comment not found")
			}
			return moderationdomain.Target{}, nil, "", err
		}
		communityID, err := communitydomain.NewCommunityID(rawCommunityID)
		if err != nil {
			return moderationdomain.Target{}, nil, "", err
		}
		commentID, err := commentdomain.NewCommentID(state.TargetID)
		if err != nil {
			return moderationdomain.Target{}, nil, "", err
		}
		target, err := moderationdomain.NewCommentTarget(commentID)
		if err != nil {
			return moderationdomain.Target{}, nil, "", err
		}
		state.TargetType = moderationdomain.TargetTypeComment.String()
		state.CommunityID = communityID
		return target, state.toMap(), communityID, nil
	default:
		return moderationdomain.Target{}, nil, "", apperr.New(apperr.CodeInvalidArgument, "moderation target type is invalid")
	}
}

func applyLockedTargetAction(ctx context.Context, tx pgx.Tx, input moderationusecase.ApplyModerationActionRecordInput, before map[string]any) (map[string]any, error) {
	after := cloneMap(before)
	status, _ := before["status"].(string)
	targetType, _ := before["target_type"].(string)
	if status == "deleted" {
		return nil, apperr.New(apperr.CodeConflict, "deleted content cannot be moderated")
	}

	table := "posts"
	idColumn := "id"
	if targetType == moderationdomain.TargetTypeComment.String() {
		table = "comments"
	}
	switch input.Action {
	case moderationdomain.ActionTypeApprove:
		after["status"] = "visible"
		return after, execTargetUpdate(ctx, tx, table, idColumn, input.TargetID, "status = 'visible', updated_at = $2", input.CreatedAt)
	case moderationdomain.ActionTypeRemove:
		after["status"] = "removed"
		return after, execTargetUpdate(ctx, tx, table, idColumn, input.TargetID, "status = 'removed', updated_at = $2", input.CreatedAt)
	case moderationdomain.ActionTypeSpam:
		after["status"] = "spam"
		return after, execTargetUpdate(ctx, tx, table, idColumn, input.TargetID, "status = 'spam', updated_at = $2", input.CreatedAt)
	case moderationdomain.ActionTypeIgnoreReports:
		return after, nil
	case moderationdomain.ActionTypeLock:
		value := true
		if input.BoolValue != nil {
			value = *input.BoolValue
		}
		after["is_locked"] = value
		return after, execTargetUpdate(ctx, tx, table, idColumn, input.TargetID, "is_locked = $3, updated_at = $2", input.CreatedAt, value)
	case moderationdomain.ActionTypePin, moderationdomain.ActionTypeMarkNSFW, moderationdomain.ActionTypeMarkSpoiler, moderationdomain.ActionTypeSetFlair:
		if targetType != moderationdomain.TargetTypePost.String() {
			return nil, apperr.New(apperr.CodeInvalidArgument, "post moderation action requires post target")
		}
		return applyPostFlagAction(ctx, tx, input, after)
	default:
		return nil, apperr.New(apperr.CodeInvalidArgument, "moderation action type is invalid")
	}
}

func applyPostFlagAction(ctx context.Context, tx pgx.Tx, input moderationusecase.ApplyModerationActionRecordInput, after map[string]any) (map[string]any, error) {
	value := true
	if input.BoolValue != nil {
		value = *input.BoolValue
	}
	switch input.Action {
	case moderationdomain.ActionTypePin:
		after["is_pinned"] = value
		return after, execTargetUpdate(ctx, tx, "posts", "id", input.TargetID, "is_pinned = $3, updated_at = $2", input.CreatedAt, value)
	case moderationdomain.ActionTypeMarkNSFW:
		after["is_nsfw"] = value
		return after, execTargetUpdate(ctx, tx, "posts", "id", input.TargetID, "is_nsfw = $3, updated_at = $2", input.CreatedAt, value)
	case moderationdomain.ActionTypeMarkSpoiler:
		after["is_spoiler"] = value
		return after, execTargetUpdate(ctx, tx, "posts", "id", input.TargetID, "is_spoiler = $3, updated_at = $2", input.CreatedAt, value)
	case moderationdomain.ActionTypeSetFlair:
		after["flair_text"] = input.FlairText
		return after, execTargetUpdate(ctx, tx, "posts", "id", input.TargetID, "flair_text = $3, updated_at = $2", input.CreatedAt, input.FlairText)
	default:
		return nil, apperr.New(apperr.CodeInvalidArgument, "moderation action type is invalid")
	}
}

func execTargetUpdate(ctx context.Context, tx pgx.Tx, table string, idColumn string, targetID string, setClause string, updatedAt any, extra ...any) error {
	args := []any{targetID, updatedAt}
	args = append(args, extra...)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $1::uuid", table, setClause, idColumn)
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return mapPostgresWriteError("apply moderation action", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "content not found")
	}
	return nil
}

func dismissPendingReports(ctx context.Context, db postgresExecutor, action moderationdomain.ModerationAction) error {
	target := action.Target()
	postID, hasPostID := target.PostID()
	commentID, hasCommentID := target.CommentID()
	var query string
	var targetID any
	if hasPostID {
		query = `
			UPDATE content_reports
			SET status = 'dismissed',
				reviewed_by = $2::uuid,
				reviewed_at = $3,
				updated_at = $3
			WHERE post_id = $1::uuid
				AND status = 'pending'
		`
		targetID = postID.String()
	} else if hasCommentID {
		query = `
			UPDATE content_reports
			SET status = 'dismissed',
				reviewed_by = $2::uuid,
				reviewed_at = $3,
				updated_at = $3
			WHERE comment_id = $1::uuid
				AND status = 'pending'
		`
		targetID = commentID.String()
	} else {
		return apperr.New(apperr.CodeInvalidArgument, "moderation target is required")
	}
	_, err := db.Exec(ctx, query, targetID, action.ActorID().String(), action.CreatedAt())
	return err
}

func insertCommunityModLog(ctx context.Context, db postgresExecutor, communityID communitydomain.CommunityID, input moderationusecase.ApplyModerationActionRecordInput, before map[string]any, after map[string]any) error {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return err
	}
	metadata := map[string]any{
		"reason":            input.Reason,
		"removal_reason_id": input.RemovalReasonID,
		"notify_author":     input.NotifyAuthor,
	}
	if input.FlairText != "" {
		metadata["flair_text"] = input.FlairText
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	const query = `
		INSERT INTO community_moderation_logs (
			id, community_id, actor_id, action, target_type, target_id, batch_id, before_state, after_state, metadata, created_at
		)
		VALUES (gen_random_uuid(), $1::uuid, $2::uuid, $3, $4, $5, $6::uuid, $7::jsonb, $8::jsonb, $9::jsonb, $10)
	`
	_, err = db.Exec(ctx, query, communityID.String(), input.ActorID.String(), input.Action.String(), input.TargetType.String(), input.TargetID, nullableUUID(input.BatchID), string(beforeJSON), string(afterJSON), string(metadataJSON), input.CreatedAt)
	return err
}

func (repo *PostgresModerationRepository) ListCommunityModLogs(ctx context.Context, input moderationusecase.ListCommunityModLogsRecordInput) ([]moderationusecase.CommunityModLog, error) {
	args := []any{input.CommunityID.String()}
	where := []string{"community_id = $1::uuid"}
	if input.Action != "" {
		args = append(args, input.Action)
		where = append(where, fmt.Sprintf("action = $%d", len(args)))
	}
	if input.ActorID != "" {
		args = append(args, input.ActorID)
		where = append(where, fmt.Sprintf("actor_id = $%d::uuid", len(args)))
	}
	if input.TargetType != "" {
		args = append(args, input.TargetType)
		where = append(where, fmt.Sprintf("target_type = $%d", len(args)))
	}
	if input.TargetID != "" {
		args = append(args, input.TargetID)
		where = append(where, fmt.Sprintf("target_id = $%d", len(args)))
	}
	args = append(args, input.Limit, input.Offset)
	query := fmt.Sprintf(`
		SELECT id::text, community_id::text, actor_id::text, action, target_type, target_id, batch_id::text, before_state, after_state, metadata, created_at
		FROM community_moderation_logs
		WHERE %s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d
		OFFSET $%d
	`, joinWhere(where), len(args)-1, len(args))
	rows, err := repo.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	logs := make([]moderationusecase.CommunityModLog, 0)
	for rows.Next() {
		log, err := scanCommunityModLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func scanCommunityModLog(row rowScanner) (moderationusecase.CommunityModLog, error) {
	var log moderationusecase.CommunityModLog
	var rawBatchID pgtype.Text
	var beforeJSON []byte
	var afterJSON []byte
	var metadataJSON []byte
	if err := row.Scan(&log.ID, &log.CommunityID, &log.ActorID, &log.Action, &log.TargetType, &log.TargetID, &rawBatchID, &beforeJSON, &afterJSON, &metadataJSON, &log.CreatedAt); err != nil {
		return moderationusecase.CommunityModLog{}, err
	}
	if rawBatchID.Valid {
		log.BatchID = rawBatchID.String
	}
	log.Before = map[string]any{}
	log.After = map[string]any{}
	log.Metadata = map[string]any{}
	_ = json.Unmarshal(beforeJSON, &log.Before)
	_ = json.Unmarshal(afterJSON, &log.After)
	_ = json.Unmarshal(metadataJSON, &log.Metadata)
	return log, nil
}

func (repo *PostgresModerationRepository) ListRemovalReasons(ctx context.Context, communityID communitydomain.CommunityID) ([]moderationusecase.ModerationTemplate, error) {
	return repo.listTemplates(ctx, "moderation_removal_reasons", communityID)
}

func (repo *PostgresModerationRepository) CreateRemovalReason(ctx context.Context, input moderationusecase.WriteModerationTemplateRecordInput) (moderationusecase.ModerationTemplate, error) {
	return repo.createTemplate(ctx, "moderation_removal_reasons", input)
}

func (repo *PostgresModerationRepository) UpdateRemovalReason(ctx context.Context, input moderationusecase.WriteModerationTemplateRecordInput) (moderationusecase.ModerationTemplate, error) {
	return repo.updateTemplate(ctx, "moderation_removal_reasons", input)
}

func (repo *PostgresModerationRepository) DeleteRemovalReason(ctx context.Context, communityID communitydomain.CommunityID, id string, actorID userdomain.UserID, deletedAt time.Time) error {
	return repo.deactivateTemplate(ctx, "moderation_removal_reasons", communityID, id, actorID, deletedAt)
}

func (repo *PostgresModerationRepository) ListSavedResponses(ctx context.Context, communityID communitydomain.CommunityID) ([]moderationusecase.ModerationTemplate, error) {
	return repo.listTemplates(ctx, "moderation_saved_responses", communityID)
}

func (repo *PostgresModerationRepository) CreateSavedResponse(ctx context.Context, input moderationusecase.WriteModerationTemplateRecordInput) (moderationusecase.ModerationTemplate, error) {
	return repo.createTemplate(ctx, "moderation_saved_responses", input)
}

func (repo *PostgresModerationRepository) UpdateSavedResponse(ctx context.Context, input moderationusecase.WriteModerationTemplateRecordInput) (moderationusecase.ModerationTemplate, error) {
	return repo.updateTemplate(ctx, "moderation_saved_responses", input)
}

func (repo *PostgresModerationRepository) DeleteSavedResponse(ctx context.Context, communityID communitydomain.CommunityID, id string, actorID userdomain.UserID, deletedAt time.Time) error {
	return repo.deactivateTemplate(ctx, "moderation_saved_responses", communityID, id, actorID, deletedAt)
}

func (repo *PostgresModerationRepository) listTemplates(ctx context.Context, table string, communityID communitydomain.CommunityID) ([]moderationusecase.ModerationTemplate, error) {
	query := fmt.Sprintf(`
		SELECT id::text, community_id::text, title, body, COALESCE(rule_id::text, ''), is_active, position, created_by::text, updated_by::text, created_at, updated_at
		FROM %s
		WHERE community_id = $1::uuid
			AND is_active = true
		ORDER BY position ASC, created_at ASC, id ASC
	`, table)
	rows, err := repo.pool.Query(ctx, query, communityID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]moderationusecase.ModerationTemplate, 0)
	for rows.Next() {
		item, err := scanModerationTemplate(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (repo *PostgresModerationRepository) createTemplate(ctx context.Context, table string, input moderationusecase.WriteModerationTemplateRecordInput) (moderationusecase.ModerationTemplate, error) {
	query := fmt.Sprintf(`
		INSERT INTO %s (id, community_id, title, body, rule_id, position, created_by, updated_by, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::uuid, $6, $7::uuid, $7::uuid, $8, $8)
		RETURNING id::text, community_id::text, title, body, COALESCE(rule_id::text, ''), is_active, position, created_by::text, updated_by::text, created_at, updated_at
	`, table)
	item, err := scanModerationTemplate(repo.pool.QueryRow(ctx, query, input.ID, input.CommunityID.String(), input.Title, input.Body, nullableUUID(input.RuleID), input.Position, input.ActorID.String(), input.CreatedAt))
	if err != nil {
		return moderationusecase.ModerationTemplate{}, mapPostgresWriteError("create moderation template", err)
	}
	return item, nil
}

func (repo *PostgresModerationRepository) updateTemplate(ctx context.Context, table string, input moderationusecase.WriteModerationTemplateRecordInput) (moderationusecase.ModerationTemplate, error) {
	query := fmt.Sprintf(`
		UPDATE %s
		SET title = $3,
			body = $4,
			rule_id = $5::uuid,
			position = $6,
			updated_by = $7::uuid,
			updated_at = $8
		WHERE id = $1::uuid
			AND community_id = $2::uuid
			AND is_active = true
		RETURNING id::text, community_id::text, title, body, COALESCE(rule_id::text, ''), is_active, position, created_by::text, updated_by::text, created_at, updated_at
	`, table)
	item, err := scanModerationTemplate(repo.pool.QueryRow(ctx, query, input.ID, input.CommunityID.String(), input.Title, input.Body, nullableUUID(input.RuleID), input.Position, input.ActorID.String(), input.UpdatedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return moderationusecase.ModerationTemplate{}, apperr.New(apperr.CodeNotFound, "moderation template not found")
		}
		return moderationusecase.ModerationTemplate{}, mapPostgresWriteError("update moderation template", err)
	}
	return item, nil
}

func (repo *PostgresModerationRepository) deactivateTemplate(ctx context.Context, table string, communityID communitydomain.CommunityID, id string, actorID userdomain.UserID, deletedAt time.Time) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET is_active = false,
			updated_by = $3::uuid,
			updated_at = $4
		WHERE id = $1::uuid
			AND community_id = $2::uuid
			AND is_active = true
	`, table)
	tag, err := repo.pool.Exec(ctx, query, id, communityID.String(), actorID.String(), deletedAt)
	if err != nil {
		return mapPostgresWriteError("delete moderation template", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "moderation template not found")
	}
	return nil
}

func scanModerationTemplate(row rowScanner) (moderationusecase.ModerationTemplate, error) {
	var item moderationusecase.ModerationTemplate
	err := row.Scan(&item.ID, &item.CommunityID, &item.Title, &item.Body, &item.RuleID, &item.IsActive, &item.Position, &item.CreatedBy, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (repo *PostgresModerationRepository) ListUserStates(ctx context.Context, communityID communitydomain.CommunityID, kind string, limit int, offset int) ([]moderationusecase.CommunityUserState, error) {
	const query = `
		SELECT s.id::text, s.community_id::text, s.user_id::text, users.username, users.display_name, users.avatar_url, s.kind, s.reason, s.expires_at, s.created_by::text, s.updated_by::text, s.created_at, s.updated_at
		FROM community_user_moderation_states s
		JOIN users ON users.id = s.user_id
		WHERE s.community_id = $1::uuid
			AND s.kind = $2
		ORDER BY s.updated_at DESC, s.id DESC
		LIMIT $3
		OFFSET $4
	`
	rows, err := repo.pool.Query(ctx, query, communityID.String(), kind, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]moderationusecase.CommunityUserState, 0)
	for rows.Next() {
		item, err := scanCommunityUserState(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (repo *PostgresModerationRepository) UpsertUserState(ctx context.Context, input moderationusecase.UpsertUserStateRecordInput) (moderationusecase.CommunityUserState, error) {
	const query = `
		INSERT INTO community_user_moderation_states (id, community_id, user_id, kind, reason, expires_at, created_by, updated_by, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7::uuid, $7::uuid, $8, $8)
		ON CONFLICT (community_id, user_id, kind)
		DO UPDATE SET
			reason = EXCLUDED.reason,
			expires_at = EXCLUDED.expires_at,
			updated_by = EXCLUDED.updated_by,
			updated_at = EXCLUDED.updated_at
		RETURNING id::text, community_id::text, user_id::text, kind, reason, expires_at, created_by::text, updated_by::text, created_at, updated_at
	`
	row := repo.pool.QueryRow(ctx, query, input.ID, input.CommunityID.String(), input.UserID.String(), input.Kind, input.Reason, input.ExpiresAt, input.ActorID.String(), input.UpdatedAt)
	state, err := scanCommunityUserStateWithoutUser(row)
	if err != nil {
		return moderationusecase.CommunityUserState{}, mapPostgresWriteError("upsert community user state", err)
	}
	return repo.hydrateCommunityUserState(ctx, state)
}

func (repo *PostgresModerationRepository) DeleteUserState(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID, kind string, actorID userdomain.UserID, deletedAt time.Time) error {
	const query = `
		DELETE FROM community_user_moderation_states
		WHERE community_id = $1::uuid
			AND user_id = $2::uuid
			AND kind = $3
	`
	tag, err := repo.pool.Exec(ctx, query, communityID.String(), userID.String(), kind)
	if err != nil {
		return mapPostgresWriteError("delete community user state", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "community user state not found")
	}
	return nil
}

func scanCommunityUserState(row rowScanner) (moderationusecase.CommunityUserState, error) {
	var item moderationusecase.CommunityUserState
	var expiresAt pgtype.Timestamptz
	err := row.Scan(&item.ID, &item.CommunityID, &item.UserID, &item.Username, &item.DisplayName, &item.AvatarURL, &item.Kind, &item.Reason, &expiresAt, &item.CreatedBy, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt)
	if expiresAt.Valid {
		item.ExpiresAt = &expiresAt.Time
	}
	return item, err
}

func scanCommunityUserStateWithoutUser(row rowScanner) (moderationusecase.CommunityUserState, error) {
	var item moderationusecase.CommunityUserState
	var expiresAt pgtype.Timestamptz
	err := row.Scan(&item.ID, &item.CommunityID, &item.UserID, &item.Kind, &item.Reason, &expiresAt, &item.CreatedBy, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt)
	if expiresAt.Valid {
		item.ExpiresAt = &expiresAt.Time
	}
	return item, err
}

func (repo *PostgresModerationRepository) hydrateCommunityUserState(ctx context.Context, state moderationusecase.CommunityUserState) (moderationusecase.CommunityUserState, error) {
	const query = `SELECT username, display_name, avatar_url FROM users WHERE id = $1::uuid`
	if err := repo.pool.QueryRow(ctx, query, state.UserID).Scan(&state.Username, &state.DisplayName, &state.AvatarURL); err != nil {
		return moderationusecase.CommunityUserState{}, err
	}
	return state, nil
}

func (repo *PostgresModerationRepository) GetUserProfile(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID) (moderationusecase.ModerationUserProfile, error) {
	const query = `
		SELECT
			users.id::text,
			users.username,
			users.display_name,
			users.avatar_url,
			users.headline,
			users.status,
			(SELECT COUNT(*)::int FROM posts WHERE posts.community_id = $1::uuid AND posts.author_id = users.id),
			(SELECT COUNT(*)::int FROM comments JOIN posts ON posts.id = comments.post_id WHERE posts.community_id = $1::uuid AND comments.author_id = users.id),
			(SELECT COUNT(*)::int FROM content_reports WHERE content_reports.reporter_id = users.id),
			(SELECT COUNT(*)::int FROM community_moderation_logs WHERE community_id = $1::uuid AND target_id = users.id::text),
			EXISTS (SELECT 1 FROM community_user_moderation_states WHERE community_id = $1::uuid AND user_id = users.id AND kind = 'banned'),
			EXISTS (SELECT 1 FROM community_user_moderation_states WHERE community_id = $1::uuid AND user_id = users.id AND kind = 'muted'),
			EXISTS (SELECT 1 FROM community_user_moderation_states WHERE community_id = $1::uuid AND user_id = users.id AND kind = 'approved')
		FROM users
		WHERE users.id = $2::uuid
	`
	var profile moderationusecase.ModerationUserProfile
	err := repo.pool.QueryRow(ctx, query, communityID.String(), userID.String()).Scan(
		&profile.UserID,
		&profile.Username,
		&profile.DisplayName,
		&profile.AvatarURL,
		&profile.Headline,
		&profile.Status,
		&profile.PostCount,
		&profile.CommentCount,
		&profile.ReportCount,
		&profile.RemovedCount,
		&profile.IsBanned,
		&profile.IsMuted,
		&profile.IsApproved,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return moderationusecase.ModerationUserProfile{}, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return moderationusecase.ModerationUserProfile{}, err
	}
	notes, err := repo.ListModeratorNotes(ctx, communityID, userID, 5, 0)
	if err != nil {
		return moderationusecase.ModerationUserProfile{}, err
	}
	profile.RecentNotes = notes
	return profile, nil
}

func (repo *PostgresModerationRepository) ListModeratorNotes(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID, limit int, offset int) ([]moderationusecase.ModeratorNote, error) {
	const query = `
		SELECT id::text, community_id::text, user_id::text, author_id::text, body, created_at
		FROM community_moderator_notes
		WHERE community_id = $1::uuid
			AND user_id = $2::uuid
			AND deleted_at IS NULL
		ORDER BY created_at DESC, id DESC
		LIMIT $3
		OFFSET $4
	`
	rows, err := repo.pool.Query(ctx, query, communityID.String(), userID.String(), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	notes := make([]moderationusecase.ModeratorNote, 0)
	for rows.Next() {
		var note moderationusecase.ModeratorNote
		if err := rows.Scan(&note.ID, &note.CommunityID, &note.UserID, &note.AuthorID, &note.Body, &note.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

func (repo *PostgresModerationRepository) CreateModeratorNote(ctx context.Context, input moderationusecase.CreateModeratorNoteRecordInput) (moderationusecase.ModeratorNote, error) {
	const query = `
		INSERT INTO community_moderator_notes (id, community_id, user_id, author_id, body, created_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6)
		RETURNING id::text, community_id::text, user_id::text, author_id::text, body, created_at
	`
	var note moderationusecase.ModeratorNote
	err := repo.pool.QueryRow(ctx, query, input.ID, input.CommunityID.String(), input.UserID.String(), input.AuthorID.String(), input.Body, input.CreatedAt).Scan(
		&note.ID, &note.CommunityID, &note.UserID, &note.AuthorID, &note.Body, &note.CreatedAt,
	)
	if err != nil {
		return moderationusecase.ModeratorNote{}, mapPostgresWriteError("create moderator note", err)
	}
	return note, nil
}

func (repo *PostgresModerationRepository) DeleteModeratorNote(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID, noteID string, actorID userdomain.UserID, deletedAt time.Time) error {
	const query = `
		UPDATE community_moderator_notes
		SET deleted_at = $4,
			deleted_by = $5::uuid
		WHERE id = $1::uuid
			AND community_id = $2::uuid
			AND user_id = $3::uuid
			AND deleted_at IS NULL
	`
	tag, err := repo.pool.Exec(ctx, query, noteID, communityID.String(), userID.String(), deletedAt, actorID.String())
	if err != nil {
		return mapPostgresWriteError("delete moderator note", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "moderator note not found")
	}
	return nil
}

func (state moderationToolTargetState) toMap() map[string]any {
	value := map[string]any{
		"target_type": state.TargetType,
		"target_id":   state.TargetID,
		"status":      state.Status,
		"is_locked":   state.IsLocked,
	}
	if state.PostID != "" {
		value["post_id"] = state.PostID
	}
	if state.TargetType == moderationdomain.TargetTypePost.String() {
		value["is_pinned"] = state.IsPinned
		value["is_nsfw"] = state.IsNSFW
		value["is_spoiler"] = state.IsSpoiler
		value["flair_text"] = state.FlairText
	}
	return value
}

func cloneMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func nullableCommunityID(id *communitydomain.CommunityID) any {
	if id == nil {
		return nil
	}
	return id.String()
}

func nullableUUID(raw string) any {
	if raw == "" {
		return nil
	}
	return raw
}

func joinWhere(parts []string) string {
	if len(parts) == 0 {
		return "TRUE"
	}
	result := parts[0]
	for _, part := range parts[1:] {
		result += " AND " + part
	}
	return result
}

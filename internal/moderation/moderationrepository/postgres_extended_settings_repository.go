package moderationrepository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationusecase"
	"github.com/jackc/pgx/v5"
)

func (repo *PostgresModerationRepository) ListCommunityFlairs(ctx context.Context, communityID communitydomain.CommunityID, kind string) ([]moderationusecase.CommunityFlair, error) {
	table, err := flairTable(kind)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT id::text, community_id::text, $2, title, color, is_user_selectable, is_enabled, position, created_by::text, updated_by::text, created_at, updated_at
		FROM %s
		WHERE community_id = $1::uuid
		ORDER BY position ASC, created_at ASC, id ASC
	`, table)
	rows, err := repo.pool.Query(ctx, query, communityID.String(), kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]moderationusecase.CommunityFlair, 0)
	for rows.Next() {
		item, err := scanCommunityFlair(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (repo *PostgresModerationRepository) CreateCommunityFlair(ctx context.Context, input moderationusecase.CommunityFlairRecordInput) (moderationusecase.CommunityFlair, error) {
	table, err := flairTable(input.Kind)
	if err != nil {
		return moderationusecase.CommunityFlair{}, err
	}
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return moderationusecase.CommunityFlair{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	query := fmt.Sprintf(`
		INSERT INTO %s (id, community_id, title, color, is_user_selectable, is_enabled, position, created_by, updated_by, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8::uuid, $8::uuid, $9, $9)
		RETURNING id::text, community_id::text, $10, title, color, is_user_selectable, is_enabled, position, created_by::text, updated_by::text, created_at, updated_at
	`, table)
	item, err := scanCommunityFlair(tx.QueryRow(ctx, query, input.ID, input.CommunityID.String(), input.Title, input.Color, input.IsUserSelectable, input.IsEnabled, input.Position, input.ActorID.String(), input.CreatedAt, input.Kind))
	if err != nil {
		return moderationusecase.CommunityFlair{}, mapPostgresWriteError("create community flair", err)
	}
	if err := insertCommunityToolLog(ctx, tx, input.CommunityID, input.ActorID, "create_"+input.Kind+"_flair", input.Kind+"_flair", input.ID, nil, flairLogState(item), nil, input.CreatedAt); err != nil {
		return moderationusecase.CommunityFlair{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return moderationusecase.CommunityFlair{}, err
	}
	committed = true
	return item, nil
}

func (repo *PostgresModerationRepository) UpdateCommunityFlair(ctx context.Context, input moderationusecase.CommunityFlairRecordInput) (moderationusecase.CommunityFlair, error) {
	table, err := flairTable(input.Kind)
	if err != nil {
		return moderationusecase.CommunityFlair{}, err
	}
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return moderationusecase.CommunityFlair{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	before, err := repo.getCommunityFlairForUpdate(ctx, tx, table, input.CommunityID, input.Kind, input.ID)
	if err != nil {
		return moderationusecase.CommunityFlair{}, err
	}
	query := fmt.Sprintf(`
		UPDATE %s
		SET title = $3,
			color = $4,
			is_user_selectable = $5,
			is_enabled = $6,
			position = $7,
			updated_by = $8::uuid,
			updated_at = $9
		WHERE id = $1::uuid
			AND community_id = $2::uuid
		RETURNING id::text, community_id::text, $10, title, color, is_user_selectable, is_enabled, position, created_by::text, updated_by::text, created_at, updated_at
	`, table)
	item, err := scanCommunityFlair(tx.QueryRow(ctx, query, input.ID, input.CommunityID.String(), input.Title, input.Color, input.IsUserSelectable, input.IsEnabled, input.Position, input.ActorID.String(), input.UpdatedAt, input.Kind))
	if err != nil {
		return moderationusecase.CommunityFlair{}, mapPostgresWriteError("update community flair", err)
	}
	if err := insertCommunityToolLog(ctx, tx, input.CommunityID, input.ActorID, "update_"+input.Kind+"_flair", input.Kind+"_flair", input.ID, flairLogState(before), flairLogState(item), nil, input.UpdatedAt); err != nil {
		return moderationusecase.CommunityFlair{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return moderationusecase.CommunityFlair{}, err
	}
	committed = true
	return item, nil
}

func (repo *PostgresModerationRepository) DeleteCommunityFlair(ctx context.Context, input moderationusecase.DeleteCommunityFlairRecordInput) error {
	table, err := flairTable(input.Kind)
	if err != nil {
		return err
	}
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	before, err := repo.getCommunityFlairForUpdate(ctx, tx, table, input.CommunityID, input.Kind, input.ID)
	if err != nil {
		return err
	}
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1::uuid AND community_id = $2::uuid`, table)
	tag, err := tx.Exec(ctx, query, input.ID, input.CommunityID.String())
	if err != nil {
		return mapPostgresWriteError("delete community flair", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "community flair not found")
	}
	if err := insertCommunityToolLog(ctx, tx, input.CommunityID, input.ActorID, "delete_"+input.Kind+"_flair", input.Kind+"_flair", input.ID, flairLogState(before), nil, nil, input.DeletedAt); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

func (repo *PostgresModerationRepository) ReorderCommunityFlairs(ctx context.Context, input moderationusecase.ReorderCommunityFlairsRecordInput) ([]moderationusecase.CommunityFlair, error) {
	table, err := flairTable(input.Kind)
	if err != nil {
		return nil, err
	}
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	before, err := repo.listCommunityFlairsTx(ctx, tx, table, input.CommunityID, input.Kind)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`UPDATE %s SET position = $4, updated_by = $3::uuid, updated_at = $5 WHERE id = $1::uuid AND community_id = $2::uuid`, table)
	for position, id := range input.IDs {
		tag, err := tx.Exec(ctx, query, id, input.CommunityID.String(), input.ActorID.String(), position, input.UpdatedAt)
		if err != nil {
			return nil, mapPostgresWriteError("reorder community flairs", err)
		}
		if tag.RowsAffected() == 0 {
			return nil, apperr.New(apperr.CodeNotFound, "community flair not found")
		}
	}
	after, err := repo.listCommunityFlairsTx(ctx, tx, table, input.CommunityID, input.Kind)
	if err != nil {
		return nil, err
	}
	if err := insertCommunityToolLog(ctx, tx, input.CommunityID, input.ActorID, "reorder_"+input.Kind+"_flairs", input.Kind+"_flairs", input.CommunityID.String(), flairsLogState(before), flairsLogState(after), nil, input.UpdatedAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	committed = true
	return after, nil
}

func (repo *PostgresModerationRepository) ListScheduledPosts(ctx context.Context, communityID communitydomain.CommunityID, limit int, offset int) ([]moderationusecase.CommunityScheduledPost, error) {
	const query = `
		SELECT id::text, community_id::text, created_by::text, updated_by::text, title, body, scheduled_at, repeat_rule, status, created_at, updated_at
		FROM community_scheduled_posts
		WHERE community_id = $1::uuid
		ORDER BY scheduled_at ASC, id ASC
		LIMIT $2
		OFFSET $3
	`
	rows, err := repo.pool.Query(ctx, query, communityID.String(), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]moderationusecase.CommunityScheduledPost, 0)
	for rows.Next() {
		item, err := scanScheduledPost(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (repo *PostgresModerationRepository) CreateScheduledPost(ctx context.Context, input moderationusecase.WriteScheduledPostRecordInput) (moderationusecase.CommunityScheduledPost, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return moderationusecase.CommunityScheduledPost{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	const query = `
		INSERT INTO community_scheduled_posts (id, community_id, created_by, updated_by, title, body, scheduled_at, repeat_rule, status, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $9)
		RETURNING id::text, community_id::text, created_by::text, updated_by::text, title, body, scheduled_at, repeat_rule, status, created_at, updated_at
	`
	item, err := scanScheduledPost(tx.QueryRow(ctx, query, input.ID, input.CommunityID.String(), input.ActorID.String(), input.Title, input.Body, input.ScheduledAt, input.RepeatRule, input.Status, input.CreatedAt))
	if err != nil {
		return moderationusecase.CommunityScheduledPost{}, mapPostgresWriteError("create scheduled post", err)
	}
	if err := insertCommunityToolLog(ctx, tx, input.CommunityID, input.ActorID, "create_scheduled_post", "scheduled_post", input.ID, nil, scheduledPostLogState(item), nil, input.CreatedAt); err != nil {
		return moderationusecase.CommunityScheduledPost{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return moderationusecase.CommunityScheduledPost{}, err
	}
	committed = true
	return item, nil
}

func (repo *PostgresModerationRepository) UpdateScheduledPost(ctx context.Context, input moderationusecase.WriteScheduledPostRecordInput) (moderationusecase.CommunityScheduledPost, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return moderationusecase.CommunityScheduledPost{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	before, err := repo.getScheduledPostForUpdate(ctx, tx, input.CommunityID, input.ID)
	if err != nil {
		return moderationusecase.CommunityScheduledPost{}, err
	}
	const query = `
		UPDATE community_scheduled_posts
		SET title = $3,
			body = $4,
			scheduled_at = $5,
			repeat_rule = $6,
			status = $7,
			updated_by = $8::uuid,
			updated_at = $9
		WHERE id = $1::uuid
			AND community_id = $2::uuid
		RETURNING id::text, community_id::text, created_by::text, updated_by::text, title, body, scheduled_at, repeat_rule, status, created_at, updated_at
	`
	item, err := scanScheduledPost(tx.QueryRow(ctx, query, input.ID, input.CommunityID.String(), input.Title, input.Body, input.ScheduledAt, input.RepeatRule, input.Status, input.ActorID.String(), input.UpdatedAt))
	if err != nil {
		return moderationusecase.CommunityScheduledPost{}, mapPostgresWriteError("update scheduled post", err)
	}
	if err := insertCommunityToolLog(ctx, tx, input.CommunityID, input.ActorID, "update_scheduled_post", "scheduled_post", input.ID, scheduledPostLogState(before), scheduledPostLogState(item), nil, input.UpdatedAt); err != nil {
		return moderationusecase.CommunityScheduledPost{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return moderationusecase.CommunityScheduledPost{}, err
	}
	committed = true
	return item, nil
}

func (repo *PostgresModerationRepository) DeleteScheduledPost(ctx context.Context, input moderationusecase.DeleteScheduledPostRecordInput) error {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	before, err := repo.getScheduledPostForUpdate(ctx, tx, input.CommunityID, input.ID)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM community_scheduled_posts WHERE id = $1::uuid AND community_id = $2::uuid`, input.ID, input.CommunityID.String())
	if err != nil {
		return mapPostgresWriteError("delete scheduled post", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "scheduled post not found")
	}
	if err := insertCommunityToolLog(ctx, tx, input.CommunityID, input.ActorID, "delete_scheduled_post", "scheduled_post", input.ID, scheduledPostLogState(before), nil, nil, input.DeletedAt); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

func (repo *PostgresModerationRepository) ListGuides(ctx context.Context, communityID communitydomain.CommunityID, limit int, offset int) ([]moderationusecase.CommunityGuide, error) {
	const query = `
		SELECT id::text, community_id::text, created_by::text, updated_by::text, title, body, position, visibility, created_at, updated_at
		FROM community_guides
		WHERE community_id = $1::uuid
		ORDER BY position ASC, created_at ASC, id ASC
		LIMIT $2
		OFFSET $3
	`
	rows, err := repo.pool.Query(ctx, query, communityID.String(), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]moderationusecase.CommunityGuide, 0)
	for rows.Next() {
		item, err := scanGuide(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (repo *PostgresModerationRepository) CreateGuide(ctx context.Context, input moderationusecase.WriteGuideRecordInput) (moderationusecase.CommunityGuide, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return moderationusecase.CommunityGuide{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	const query = `
		INSERT INTO community_guides (id, community_id, created_by, updated_by, title, body, position, visibility, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $3::uuid, $4, $5, $6, $7, $8, $8)
		RETURNING id::text, community_id::text, created_by::text, updated_by::text, title, body, position, visibility, created_at, updated_at
	`
	item, err := scanGuide(tx.QueryRow(ctx, query, input.ID, input.CommunityID.String(), input.ActorID.String(), input.Title, input.Body, input.Position, input.Visibility, input.CreatedAt))
	if err != nil {
		return moderationusecase.CommunityGuide{}, mapPostgresWriteError("create guide", err)
	}
	if err := insertCommunityToolLog(ctx, tx, input.CommunityID, input.ActorID, "create_guide", "guide", input.ID, nil, guideLogState(item), nil, input.CreatedAt); err != nil {
		return moderationusecase.CommunityGuide{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return moderationusecase.CommunityGuide{}, err
	}
	committed = true
	return item, nil
}

func (repo *PostgresModerationRepository) UpdateGuide(ctx context.Context, input moderationusecase.WriteGuideRecordInput) (moderationusecase.CommunityGuide, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return moderationusecase.CommunityGuide{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	before, err := repo.getGuideForUpdate(ctx, tx, input.CommunityID, input.ID)
	if err != nil {
		return moderationusecase.CommunityGuide{}, err
	}
	const query = `
		UPDATE community_guides
		SET title = $3,
			body = $4,
			position = $5,
			visibility = $6,
			updated_by = $7::uuid,
			updated_at = $8
		WHERE id = $1::uuid
			AND community_id = $2::uuid
		RETURNING id::text, community_id::text, created_by::text, updated_by::text, title, body, position, visibility, created_at, updated_at
	`
	item, err := scanGuide(tx.QueryRow(ctx, query, input.ID, input.CommunityID.String(), input.Title, input.Body, input.Position, input.Visibility, input.ActorID.String(), input.UpdatedAt))
	if err != nil {
		return moderationusecase.CommunityGuide{}, mapPostgresWriteError("update guide", err)
	}
	if err := insertCommunityToolLog(ctx, tx, input.CommunityID, input.ActorID, "update_guide", "guide", input.ID, guideLogState(before), guideLogState(item), nil, input.UpdatedAt); err != nil {
		return moderationusecase.CommunityGuide{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return moderationusecase.CommunityGuide{}, err
	}
	committed = true
	return item, nil
}

func (repo *PostgresModerationRepository) DeleteGuide(ctx context.Context, input moderationusecase.DeleteGuideRecordInput) error {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	before, err := repo.getGuideForUpdate(ctx, tx, input.CommunityID, input.ID)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM community_guides WHERE id = $1::uuid AND community_id = $2::uuid`, input.ID, input.CommunityID.String())
	if err != nil {
		return mapPostgresWriteError("delete guide", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "guide not found")
	}
	if err := insertCommunityToolLog(ctx, tx, input.CommunityID, input.ActorID, "delete_guide", "guide", input.ID, guideLogState(before), nil, nil, input.DeletedAt); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

func (repo *PostgresModerationRepository) getCommunityFlairForUpdate(ctx context.Context, db postgresExecutor, table string, communityID communitydomain.CommunityID, kind string, id string) (moderationusecase.CommunityFlair, error) {
	query := fmt.Sprintf(`
		SELECT id::text, community_id::text, $3, title, color, is_user_selectable, is_enabled, position, created_by::text, updated_by::text, created_at, updated_at
		FROM %s
		WHERE id = $1::uuid AND community_id = $2::uuid
		FOR UPDATE
	`, table)
	item, err := scanCommunityFlair(db.QueryRow(ctx, query, id, communityID.String(), kind))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return moderationusecase.CommunityFlair{}, apperr.New(apperr.CodeNotFound, "community flair not found")
		}
		return moderationusecase.CommunityFlair{}, err
	}
	return item, nil
}

func (repo *PostgresModerationRepository) listCommunityFlairsTx(ctx context.Context, db postgresExecutor, table string, communityID communitydomain.CommunityID, kind string) ([]moderationusecase.CommunityFlair, error) {
	query := fmt.Sprintf(`
		SELECT id::text, community_id::text, $2, title, color, is_user_selectable, is_enabled, position, created_by::text, updated_by::text, created_at, updated_at
		FROM %s
		WHERE community_id = $1::uuid
		ORDER BY position ASC, created_at ASC, id ASC
	`, table)
	rows, err := db.Query(ctx, query, communityID.String(), kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]moderationusecase.CommunityFlair, 0)
	for rows.Next() {
		item, err := scanCommunityFlair(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (repo *PostgresModerationRepository) getScheduledPostForUpdate(ctx context.Context, db postgresExecutor, communityID communitydomain.CommunityID, id string) (moderationusecase.CommunityScheduledPost, error) {
	const query = `
		SELECT id::text, community_id::text, created_by::text, updated_by::text, title, body, scheduled_at, repeat_rule, status, created_at, updated_at
		FROM community_scheduled_posts
		WHERE id = $1::uuid AND community_id = $2::uuid
		FOR UPDATE
	`
	item, err := scanScheduledPost(db.QueryRow(ctx, query, id, communityID.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return moderationusecase.CommunityScheduledPost{}, apperr.New(apperr.CodeNotFound, "scheduled post not found")
		}
		return moderationusecase.CommunityScheduledPost{}, err
	}
	return item, nil
}

func (repo *PostgresModerationRepository) getGuideForUpdate(ctx context.Context, db postgresExecutor, communityID communitydomain.CommunityID, id string) (moderationusecase.CommunityGuide, error) {
	const query = `
		SELECT id::text, community_id::text, created_by::text, updated_by::text, title, body, position, visibility, created_at, updated_at
		FROM community_guides
		WHERE id = $1::uuid AND community_id = $2::uuid
		FOR UPDATE
	`
	item, err := scanGuide(db.QueryRow(ctx, query, id, communityID.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return moderationusecase.CommunityGuide{}, apperr.New(apperr.CodeNotFound, "guide not found")
		}
		return moderationusecase.CommunityGuide{}, err
	}
	return item, nil
}

func scanCommunityFlair(row rowScanner) (moderationusecase.CommunityFlair, error) {
	var item moderationusecase.CommunityFlair
	err := row.Scan(&item.ID, &item.CommunityID, &item.Kind, &item.Title, &item.Color, &item.IsUserSelectable, &item.IsEnabled, &item.Position, &item.CreatedBy, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanScheduledPost(row rowScanner) (moderationusecase.CommunityScheduledPost, error) {
	var item moderationusecase.CommunityScheduledPost
	err := row.Scan(&item.ID, &item.CommunityID, &item.CreatedBy, &item.UpdatedBy, &item.Title, &item.Body, &item.ScheduledAt, &item.RepeatRule, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanGuide(row rowScanner) (moderationusecase.CommunityGuide, error) {
	var item moderationusecase.CommunityGuide
	err := row.Scan(&item.ID, &item.CommunityID, &item.CreatedBy, &item.UpdatedBy, &item.Title, &item.Body, &item.Position, &item.Visibility, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func flairTable(kind string) (string, error) {
	switch kind {
	case moderationusecase.CommunityFlairKindPost:
		return "community_post_flairs", nil
	case moderationusecase.CommunityFlairKindUser:
		return "community_user_flairs", nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "flair kind is invalid")
	}
}

func insertCommunityToolLog(ctx context.Context, db postgresExecutor, communityID communitydomain.CommunityID, actorID fmt.Stringer, action string, targetType string, targetID string, before map[string]any, after map[string]any, metadata map[string]any, createdAt any) error {
	if before == nil {
		before = map[string]any{}
	}
	if after == nil {
		after = map[string]any{}
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	const query = `
		INSERT INTO community_moderation_logs (
			id, community_id, actor_id, action, target_type, target_id, batch_id, before_state, after_state, metadata, created_at
		)
		VALUES (gen_random_uuid(), $1::uuid, $2::uuid, $3, $4, $5, NULL, $6::jsonb, $7::jsonb, $8::jsonb, $9)
	`
	_, err = db.Exec(ctx, query, communityID.String(), actorID.String(), action, targetType, targetID, string(beforeJSON), string(afterJSON), string(metadataJSON), createdAt)
	return err
}

func flairLogState(item moderationusecase.CommunityFlair) map[string]any {
	return map[string]any{
		"id":                 item.ID,
		"kind":               item.Kind,
		"title":              item.Title,
		"color":              item.Color,
		"is_user_selectable": item.IsUserSelectable,
		"is_enabled":         item.IsEnabled,
		"position":           item.Position,
	}
}

func flairsLogState(items []moderationusecase.CommunityFlair) map[string]any {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return map[string]any{"ids": ids}
}

func scheduledPostLogState(item moderationusecase.CommunityScheduledPost) map[string]any {
	return map[string]any{
		"id":           item.ID,
		"title":        item.Title,
		"scheduled_at": item.ScheduledAt,
		"repeat_rule":  item.RepeatRule,
		"status":       item.Status,
	}
}

func guideLogState(item moderationusecase.CommunityGuide) map[string]any {
	return map[string]any{
		"id":         item.ID,
		"title":      item.Title,
		"position":   item.Position,
		"visibility": item.Visibility,
	}
}

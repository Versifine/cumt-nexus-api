package moderationrepository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationusecase"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ moderationusecase.ContentReportRepository = (*PostgresModerationRepository)(nil)
var _ moderationusecase.ContentReportQueryRepository = (*PostgresModerationRepository)(nil)
var _ moderationusecase.ContentReportReviewRepository = (*PostgresModerationRepository)(nil)
var _ moderationusecase.ContentRemovalRepository = (*PostgresModerationRepository)(nil)
var _ moderationusecase.ReportedTargetRemovalRepository = (*PostgresModerationRepository)(nil)

type PostgresModerationRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresModerationRepository(pool *pgxpool.Pool) *PostgresModerationRepository {
	return &PostgresModerationRepository{
		pool: pool,
	}
}

func (repo *PostgresModerationRepository) CreateReport(ctx context.Context, report moderationdomain.ContentReport) error {
	const query = `
		INSERT INTO content_reports (
			id,
			reporter_id,
			post_id,
			comment_id,
			reason,
			status,
			reviewed_by,
			reviewed_at,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7::uuid, $8, $9, $10)
	`

	target := report.Target()
	postID, hasPostID := target.PostID()
	commentID, hasCommentID := target.CommentID()
	reviewerID, hasReviewerID := report.ReviewedBy()
	reviewedAt, hasReviewedAt := report.ReviewedAt()

	_, err := repo.pool.Exec(
		ctx,
		query,
		report.ID().String(),
		report.ReporterID().String(),
		nullablePostIDValue(postID, hasPostID),
		nullableCommentIDValue(commentID, hasCommentID),
		report.Reason().String(),
		report.Status().String(),
		nullableUserIDValue(reviewerID, hasReviewerID),
		nullableTimeValue(reviewedAt, hasReviewedAt),
		report.CreatedAt(),
		report.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("create content report", err)
	}

	return nil
}

func (repo *PostgresModerationRepository) ListReports(ctx context.Context, status moderationdomain.ReportStatus, limit int, offset int) ([]moderationusecase.ContentReportRecord, error) {
	const query = `
		SELECT
			content_reports.id::text,
			content_reports.reporter_id::text,
			content_reports.post_id::text,
			content_reports.comment_id::text,
			content_reports.reason,
			content_reports.status,
			content_reports.reviewed_by::text,
			content_reports.reviewed_at,
			content_reports.created_at,
			content_reports.updated_at,
			posts.id::text,
			posts.author_id::text,
			posts.status,
			posts.title,
			posts.body,
			posts.created_at,
			posts.updated_at,
			comments.id::text,
			comments.post_id::text,
			comments.author_id::text,
			comments.status,
			comments.body,
			comments.created_at,
			comments.updated_at
		FROM content_reports
		LEFT JOIN posts ON posts.id = content_reports.post_id
		LEFT JOIN comments ON comments.id = content_reports.comment_id
		WHERE content_reports.status = $1
		ORDER BY content_reports.created_at DESC, content_reports.id DESC
		LIMIT $2
		OFFSET $3
	`

	rows, err := repo.pool.Query(ctx, query, status.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list content reports: %w", err)
	}
	defer rows.Close()

	records := make([]moderationusecase.ContentReportRecord, 0)
	for rows.Next() {
		record, err := scanContentReportRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, *record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate content reports: %w", err)
	}
	return records, nil
}

func (repo *PostgresModerationRepository) FindReportByID(ctx context.Context, id moderationdomain.ContentReportID) (*moderationusecase.ContentReportRecord, error) {
	const query = `
		SELECT
			content_reports.id::text,
			content_reports.reporter_id::text,
			content_reports.post_id::text,
			content_reports.comment_id::text,
			content_reports.reason,
			content_reports.status,
			content_reports.reviewed_by::text,
			content_reports.reviewed_at,
			content_reports.created_at,
			content_reports.updated_at,
			posts.id::text,
			posts.author_id::text,
			posts.status,
			posts.title,
			posts.body,
			posts.created_at,
			posts.updated_at,
			comments.id::text,
			comments.post_id::text,
			comments.author_id::text,
			comments.status,
			comments.body,
			comments.created_at,
			comments.updated_at
		FROM content_reports
		LEFT JOIN posts ON posts.id = content_reports.post_id
		LEFT JOIN comments ON comments.id = content_reports.comment_id
		WHERE content_reports.id = $1::uuid
		LIMIT 1
	`

	record, err := scanContentReportRecord(repo.pool.QueryRow(ctx, query, id.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeNotFound, "content report not found")
		}
		return nil, err
	}
	return record, nil
}

func (repo *PostgresModerationRepository) DismissReport(ctx context.Context, id moderationdomain.ContentReportID, reviewerID userdomain.UserID, reviewedAt time.Time) (*moderationdomain.ContentReport, error) {
	const query = `
		UPDATE content_reports
		SET status = 'dismissed',
			reviewed_by = $2::uuid,
			reviewed_at = $3,
			updated_at = $3
		WHERE id = $1::uuid
			AND status = 'pending'
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

	report, err := scanContentReport(repo.pool.QueryRow(ctx, query, id.String(), reviewerID.String(), reviewedAt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeConflict, "content report is not pending")
		}
		return nil, err
	}
	return report, nil
}

func (repo *PostgresModerationRepository) RemovePostWithAction(ctx context.Context, action moderationdomain.ModerationAction) error {
	return repo.removeWithAction(ctx, action, "posts")
}

func (repo *PostgresModerationRepository) RemoveCommentWithAction(ctx context.Context, action moderationdomain.ModerationAction) error {
	return repo.removeWithAction(ctx, action, "comments")
}

func (repo *PostgresModerationRepository) RemoveReportedTargetWithAction(ctx context.Context, reportID moderationdomain.ContentReportID, action moderationdomain.ModerationAction) (err error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reported target removal transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	reportTarget, reportStatus, err := lockContentReportTarget(ctx, tx, reportID)
	if err != nil {
		return err
	}
	if reportStatus != moderationdomain.ReportStatusPending {
		return apperr.New(apperr.CodeConflict, "content report is not pending")
	}
	if !sameModerationTarget(reportTarget, action.Target()) {
		return apperr.New(apperr.CodeInvalidArgument, "moderation action target does not match report")
	}

	if err := removeVisibleTarget(ctx, tx, action); err != nil {
		return err
	}
	if err := insertModerationAction(ctx, tx, action); err != nil {
		return err
	}
	if err := resolvePendingReports(ctx, tx, action); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit reported target removal transaction: %w", err)
	}
	committed = true
	return nil
}

func (repo *PostgresModerationRepository) removeWithAction(ctx context.Context, action moderationdomain.ModerationAction, targetTable string) (err error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin moderation transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	target := action.Target()
	_, hasPostID := target.PostID()
	_, hasCommentID := target.CommentID()

	switch targetTable {
	case "posts":
		if !hasPostID {
			return apperr.New(apperr.CodeInvalidArgument, "moderation post target is required")
		}
	case "comments":
		if !hasCommentID {
			return apperr.New(apperr.CodeInvalidArgument, "moderation comment target is required")
		}
	default:
		return apperr.New(apperr.CodeInvalidArgument, "moderation target table is invalid")
	}

	if err := removeVisibleTarget(ctx, tx, action); err != nil {
		return err
	}
	if err := insertModerationAction(ctx, tx, action); err != nil {
		return err
	}
	if err := resolvePendingReports(ctx, tx, action); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit moderation transaction: %w", err)
	}
	committed = true
	return nil
}

func lockContentReportTarget(ctx context.Context, tx pgx.Tx, reportID moderationdomain.ContentReportID) (moderationdomain.Target, moderationdomain.ReportStatus, error) {
	const query = `
		SELECT
			post_id::text,
			comment_id::text,
			status
		FROM content_reports
		WHERE id = $1::uuid
		FOR UPDATE
	`

	var rawPostID pgtype.Text
	var rawCommentID pgtype.Text
	var rawStatus string
	if err := tx.QueryRow(ctx, query, reportID.String()).Scan(&rawPostID, &rawCommentID, &rawStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return moderationdomain.Target{}, "", apperr.New(apperr.CodeNotFound, "content report not found")
		}
		return moderationdomain.Target{}, "", fmt.Errorf("lock content report: %w", err)
	}

	target, err := rehydrateReportTarget(rawPostID, rawCommentID)
	if err != nil {
		return moderationdomain.Target{}, "", err
	}
	status, err := moderationdomain.NewReportStatus(rawStatus)
	if err != nil {
		return moderationdomain.Target{}, "", err
	}
	return target, status, nil
}

func removeVisibleTarget(ctx context.Context, db postgresExecutor, action moderationdomain.ModerationAction) error {
	target := action.Target()
	postID, hasPostID := target.PostID()
	commentID, hasCommentID := target.CommentID()

	var updateQuery string
	var targetID any
	if hasPostID {
		updateQuery = `
			UPDATE posts
			SET status = 'removed',
				updated_at = $2
			WHERE id = $1::uuid
				AND status = 'visible'
		`
		targetID = postID.String()
	} else if hasCommentID {
		updateQuery = `
			UPDATE comments
			SET status = 'removed',
				updated_at = $2
			WHERE id = $1::uuid
				AND status = 'visible'
		`
		targetID = commentID.String()
	} else {
		return apperr.New(apperr.CodeInvalidArgument, "moderation target is required")
	}

	tag, err := db.Exec(ctx, updateQuery, targetID, action.CreatedAt())
	if err != nil {
		return mapPostgresWriteError("remove content", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "content not found")
	}
	return nil
}

func sameModerationTarget(left moderationdomain.Target, right moderationdomain.Target) bool {
	if left.Type() != right.Type() {
		return false
	}
	if leftPostID, ok := left.PostID(); ok {
		rightPostID, rightOK := right.PostID()
		return rightOK && rightPostID == leftPostID
	}
	if leftCommentID, ok := left.CommentID(); ok {
		rightCommentID, rightOK := right.CommentID()
		return rightOK && rightCommentID == leftCommentID
	}
	return false
}

type postgresExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanContentReport(row rowScanner) (*moderationdomain.ContentReport, error) {
	var rawID string
	var rawReporterID string
	var rawPostID pgtype.Text
	var rawCommentID pgtype.Text
	var rawReason string
	var rawStatus string
	var rawReviewedBy pgtype.Text
	var rawReviewedAt pgtype.Timestamptz
	var createdAt time.Time
	var updatedAt time.Time

	if err := row.Scan(
		&rawID,
		&rawReporterID,
		&rawPostID,
		&rawCommentID,
		&rawReason,
		&rawStatus,
		&rawReviewedBy,
		&rawReviewedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	id, err := moderationdomain.NewContentReportID(rawID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate content report id: %v", err)
	}
	reporterID, err := userdomain.NewUserID(rawReporterID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate content report reporter id: %v", err)
	}
	target, err := rehydrateReportTarget(rawPostID, rawCommentID)
	if err != nil {
		return nil, err
	}
	reason, err := moderationdomain.NewReason(rawReason)
	if err != nil {
		return nil, fmt.Errorf("rehydrate content report reason: %v", err)
	}
	status, err := moderationdomain.NewReportStatus(rawStatus)
	if err != nil {
		return nil, fmt.Errorf("rehydrate content report status: %v", err)
	}
	reviewedBy, err := rehydrateOptionalUserID(rawReviewedBy)
	if err != nil {
		return nil, err
	}
	reviewedAt := rehydrateOptionalTime(rawReviewedAt)

	report, err := moderationdomain.RehydrateContentReport(id, target, reporterID, reason, status, reviewedBy, reviewedAt, createdAt, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("rehydrate content report: %v", err)
	}
	return report, nil
}

func scanContentReportRecord(row rowScanner) (*moderationusecase.ContentReportRecord, error) {
	var rawID string
	var rawReporterID string
	var rawPostID pgtype.Text
	var rawCommentID pgtype.Text
	var rawReason string
	var rawStatus string
	var rawReviewedBy pgtype.Text
	var rawReviewedAt pgtype.Timestamptz
	var createdAt time.Time
	var updatedAt time.Time
	var rawPreviewPostID pgtype.Text
	var rawPreviewPostAuthorID pgtype.Text
	var rawPreviewPostStatus pgtype.Text
	var rawPreviewPostTitle pgtype.Text
	var rawPreviewPostBody pgtype.Text
	var rawPreviewPostCreatedAt pgtype.Timestamptz
	var rawPreviewPostUpdatedAt pgtype.Timestamptz
	var rawPreviewCommentID pgtype.Text
	var rawPreviewCommentPostID pgtype.Text
	var rawPreviewCommentAuthorID pgtype.Text
	var rawPreviewCommentStatus pgtype.Text
	var rawPreviewCommentBody pgtype.Text
	var rawPreviewCommentCreatedAt pgtype.Timestamptz
	var rawPreviewCommentUpdatedAt pgtype.Timestamptz

	if err := row.Scan(
		&rawID,
		&rawReporterID,
		&rawPostID,
		&rawCommentID,
		&rawReason,
		&rawStatus,
		&rawReviewedBy,
		&rawReviewedAt,
		&createdAt,
		&updatedAt,
		&rawPreviewPostID,
		&rawPreviewPostAuthorID,
		&rawPreviewPostStatus,
		&rawPreviewPostTitle,
		&rawPreviewPostBody,
		&rawPreviewPostCreatedAt,
		&rawPreviewPostUpdatedAt,
		&rawPreviewCommentID,
		&rawPreviewCommentPostID,
		&rawPreviewCommentAuthorID,
		&rawPreviewCommentStatus,
		&rawPreviewCommentBody,
		&rawPreviewCommentCreatedAt,
		&rawPreviewCommentUpdatedAt,
	); err != nil {
		return nil, err
	}

	id, err := moderationdomain.NewContentReportID(rawID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate content report id: %v", err)
	}
	reporterID, err := userdomain.NewUserID(rawReporterID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate content report reporter id: %v", err)
	}
	target, err := rehydrateReportTarget(rawPostID, rawCommentID)
	if err != nil {
		return nil, err
	}
	reason, err := moderationdomain.NewReason(rawReason)
	if err != nil {
		return nil, fmt.Errorf("rehydrate content report reason: %v", err)
	}
	status, err := moderationdomain.NewReportStatus(rawStatus)
	if err != nil {
		return nil, fmt.Errorf("rehydrate content report status: %v", err)
	}
	reviewedBy, err := rehydrateOptionalUserID(rawReviewedBy)
	if err != nil {
		return nil, err
	}
	reviewedAt := rehydrateOptionalTime(rawReviewedAt)

	report, err := moderationdomain.RehydrateContentReport(id, target, reporterID, reason, status, reviewedBy, reviewedAt, createdAt, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("rehydrate content report: %v", err)
	}

	preview, err := rehydrateReportTargetPreview(
		rawPreviewPostID,
		rawPreviewPostAuthorID,
		rawPreviewPostStatus,
		rawPreviewPostTitle,
		rawPreviewPostBody,
		rawPreviewPostCreatedAt,
		rawPreviewPostUpdatedAt,
		rawPreviewCommentID,
		rawPreviewCommentPostID,
		rawPreviewCommentAuthorID,
		rawPreviewCommentStatus,
		rawPreviewCommentBody,
		rawPreviewCommentCreatedAt,
		rawPreviewCommentUpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &moderationusecase.ContentReportRecord{
		Report:        *report,
		TargetPreview: preview,
	}, nil
}

func rehydrateReportTarget(rawPostID pgtype.Text, rawCommentID pgtype.Text) (moderationdomain.Target, error) {
	if rawPostID.Valid {
		postID, err := postdomain.NewPostID(rawPostID.String)
		if err != nil {
			return moderationdomain.Target{}, fmt.Errorf("rehydrate content report post id: %v", err)
		}
		target, err := moderationdomain.NewPostTarget(postID)
		if err != nil {
			return moderationdomain.Target{}, fmt.Errorf("rehydrate content report post target: %v", err)
		}
		return target, nil
	}
	if rawCommentID.Valid {
		commentID, err := commentdomain.NewCommentID(rawCommentID.String)
		if err != nil {
			return moderationdomain.Target{}, fmt.Errorf("rehydrate content report comment id: %v", err)
		}
		target, err := moderationdomain.NewCommentTarget(commentID)
		if err != nil {
			return moderationdomain.Target{}, fmt.Errorf("rehydrate content report comment target: %v", err)
		}
		return target, nil
	}
	return moderationdomain.Target{}, apperr.New(apperr.CodeInvalidArgument, "content report target is invalid")
}

func rehydrateOptionalUserID(raw pgtype.Text) (*userdomain.UserID, error) {
	if !raw.Valid {
		return nil, nil
	}
	id, err := userdomain.NewUserID(raw.String)
	if err != nil {
		return nil, fmt.Errorf("rehydrate content report reviewer id: %v", err)
	}
	return &id, nil
}

func rehydrateOptionalTime(raw pgtype.Timestamptz) *time.Time {
	if !raw.Valid {
		return nil
	}
	value := raw.Time
	return &value
}

func rehydrateReportTargetPreview(
	rawPostID pgtype.Text,
	rawPostAuthorID pgtype.Text,
	rawPostStatus pgtype.Text,
	rawPostTitle pgtype.Text,
	rawPostBody pgtype.Text,
	rawPostCreatedAt pgtype.Timestamptz,
	rawPostUpdatedAt pgtype.Timestamptz,
	rawCommentID pgtype.Text,
	rawCommentPostID pgtype.Text,
	rawCommentAuthorID pgtype.Text,
	rawCommentStatus pgtype.Text,
	rawCommentBody pgtype.Text,
	rawCommentCreatedAt pgtype.Timestamptz,
	rawCommentUpdatedAt pgtype.Timestamptz,
) (*moderationusecase.ReportTargetPreview, error) {
	if rawPostID.Valid {
		createdAt, err := requiredPreviewTime(rawPostCreatedAt, "post created_at")
		if err != nil {
			return nil, err
		}
		updatedAt, err := requiredPreviewTime(rawPostUpdatedAt, "post updated_at")
		if err != nil {
			return nil, err
		}
		return &moderationusecase.ReportTargetPreview{
			TargetType:  moderationdomain.TargetTypePost.String(),
			PostID:      rawPostID.String,
			AuthorID:    rawPostAuthorID.String,
			Status:      rawPostStatus.String,
			Title:       rawPostTitle.String,
			BodyExcerpt: excerptBody(rawPostBody.String),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}, nil
	}
	if rawCommentID.Valid {
		createdAt, err := requiredPreviewTime(rawCommentCreatedAt, "comment created_at")
		if err != nil {
			return nil, err
		}
		updatedAt, err := requiredPreviewTime(rawCommentUpdatedAt, "comment updated_at")
		if err != nil {
			return nil, err
		}
		return &moderationusecase.ReportTargetPreview{
			TargetType:  moderationdomain.TargetTypeComment.String(),
			PostID:      rawCommentPostID.String,
			CommentID:   rawCommentID.String,
			AuthorID:    rawCommentAuthorID.String,
			Status:      rawCommentStatus.String,
			BodyExcerpt: excerptBody(rawCommentBody.String),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}, nil
	}
	return nil, nil
}

func requiredPreviewTime(raw pgtype.Timestamptz, name string) (time.Time, error) {
	if !raw.Valid {
		return time.Time{}, fmt.Errorf("rehydrate report target preview %s: value is null", name)
	}
	return raw.Time, nil
}

func excerptBody(raw string) string {
	const maxRunes = 160
	trimmed := strings.TrimSpace(raw)
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return string(runes[:maxRunes]) + "..."
}

func insertModerationAction(ctx context.Context, db postgresExecutor, action moderationdomain.ModerationAction) error {
	const query = `
		INSERT INTO moderation_actions (
			id,
			actor_id,
			post_id,
			comment_id,
			action,
			reason,
			created_at
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7)
	`

	target := action.Target()
	postID, hasPostID := target.PostID()
	commentID, hasCommentID := target.CommentID()
	_, err := db.Exec(
		ctx,
		query,
		action.ID().String(),
		action.ActorID().String(),
		nullablePostIDValue(postID, hasPostID),
		nullableCommentIDValue(commentID, hasCommentID),
		action.Action().String(),
		action.Reason().String(),
		action.CreatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("insert moderation action", err)
	}
	return nil
}

func resolvePendingReports(ctx context.Context, db postgresExecutor, action moderationdomain.ModerationAction) error {
	target := action.Target()
	postID, hasPostID := target.PostID()
	commentID, hasCommentID := target.CommentID()

	var query string
	var targetID any
	if hasPostID {
		query = `
			UPDATE content_reports
			SET status = 'resolved',
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
			SET status = 'resolved',
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

	if _, err := db.Exec(ctx, query, targetID, action.ActorID().String(), action.CreatedAt()); err != nil {
		return mapPostgresWriteError("resolve pending reports", err)
	}
	return nil
}

func nullablePostIDValue(id fmt.Stringer, ok bool) any {
	if !ok {
		return nil
	}
	return id.String()
}

func nullableCommentIDValue(id fmt.Stringer, ok bool) any {
	if !ok {
		return nil
	}
	return id.String()
}

func nullableUserIDValue(id fmt.Stringer, ok bool) any {
	if !ok {
		return nil
	}
	return id.String()
}

func nullableTimeValue(value any, ok bool) any {
	if !ok {
		return nil
	}
	return value
}

func mapPostgresWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return apperr.New(apperr.CodeConflict, "content report already exists")
		}
		if pgErr.Code == "23503" {
			return apperr.New(apperr.CodeNotFound, "related record not found")
		}
		if pgErr.Code == "23514" {
			return apperr.New(apperr.CodeInvalidArgument, "content report is invalid")
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

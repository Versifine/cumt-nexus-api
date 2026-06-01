package moderationrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationusecase"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ moderationusecase.ContentReportRepository = (*PostgresModerationRepository)(nil)
var _ moderationusecase.ContentRemovalRepository = (*PostgresModerationRepository)(nil)

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

func (repo *PostgresModerationRepository) RemovePostWithAction(ctx context.Context, action moderationdomain.ModerationAction) error {
	return repo.removeWithAction(ctx, action, "posts")
}

func (repo *PostgresModerationRepository) RemoveCommentWithAction(ctx context.Context, action moderationdomain.ModerationAction) error {
	return repo.removeWithAction(ctx, action, "comments")
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
	postID, hasPostID := target.PostID()
	commentID, hasCommentID := target.CommentID()

	var updateQuery string
	var targetID any
	switch targetTable {
	case "posts":
		if !hasPostID {
			return apperr.New(apperr.CodeInvalidArgument, "moderation post target is required")
		}
		updateQuery = `
			UPDATE posts
			SET status = 'removed',
				updated_at = $2
			WHERE id = $1::uuid
				AND status = 'visible'
		`
		targetID = postID.String()
	case "comments":
		if !hasCommentID {
			return apperr.New(apperr.CodeInvalidArgument, "moderation comment target is required")
		}
		updateQuery = `
			UPDATE comments
			SET status = 'removed',
				updated_at = $2
			WHERE id = $1::uuid
				AND status = 'visible'
		`
		targetID = commentID.String()
	default:
		return apperr.New(apperr.CodeInvalidArgument, "moderation target table is invalid")
	}

	tag, err := tx.Exec(ctx, updateQuery, targetID, action.CreatedAt())
	if err != nil {
		return mapPostgresWriteError("remove content", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "content not found")
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

type postgresExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
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

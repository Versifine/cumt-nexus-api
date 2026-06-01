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

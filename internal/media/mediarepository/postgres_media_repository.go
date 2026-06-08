package mediarepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediausecase"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ mediausecase.AttachmentRepository = (*PostgresMediaRepository)(nil)

type PostgresMediaRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresMediaRepository(pool *pgxpool.Pool) *PostgresMediaRepository {
	return &PostgresMediaRepository{
		pool: pool,
	}
}

func (repo *PostgresMediaRepository) Create(ctx context.Context, attachment mediadomain.Attachment) error {
	const query = `
		INSERT INTO media_attachments (
			id,
			owner_type,
			owner_id,
			uploader_id,
			kind,
			storage_provider,
			bucket,
			object_key,
			public_url,
			thumbnail_object_key,
			width,
			height,
			size_bytes,
			mime_type,
			alt_text,
			status,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2, $3::uuid, $4::uuid, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	_, err := repo.pool.Exec(
		ctx,
		query,
		attachment.ID().String(),
		attachment.OwnerType().String(),
		nullableString(attachment.OwnerID()),
		attachment.UploaderID().String(),
		attachment.Kind().String(),
		attachment.StorageProvider().String(),
		attachment.Bucket(),
		attachment.ObjectKey(),
		attachment.PublicURL(),
		nullableString(attachment.ThumbnailObjectKey()),
		nullableInt(attachment.Width()),
		nullableInt(attachment.Height()),
		attachment.SizeBytes(),
		attachment.MimeType(),
		nullableString(attachment.AltText()),
		attachment.Status().String(),
		attachment.CreatedAt(),
		attachment.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("create media attachment", err)
	}
	return nil
}

func (repo *PostgresMediaRepository) FindByID(ctx context.Context, id mediadomain.AttachmentID) (*mediadomain.Attachment, error) {
	const query = `
		SELECT
			id::text,
			owner_type,
			owner_id::text,
			uploader_id::text,
			kind,
			storage_provider,
			bucket,
			object_key,
			public_url,
			thumbnail_object_key,
			width,
			height,
			size_bytes,
			mime_type,
			alt_text,
			status,
			created_at,
			updated_at
		FROM media_attachments
		WHERE id = $1::uuid
		LIMIT 1
	`

	attachment, err := scanAttachment(repo.pool.QueryRow(ctx, query, id.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeNotFound, "media attachment not found")
		}
		return nil, err
	}
	return attachment, nil
}

func (repo *PostgresMediaRepository) BindReadyImagesToPost(ctx context.Context, postID postdomain.PostID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error) {
	if len(attachmentIDs) == 0 {
		return []mediadomain.Attachment{}, nil
	}
	if maxCount <= 0 || len(attachmentIDs) > maxCount {
		return nil, apperr.New(apperr.CodeInvalidArgument, "post image attachment count is invalid")
	}

	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin post attachment binding transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	attachments, err := lockAttachmentsByIDs(ctx, tx, attachmentIDs)
	if err != nil {
		return nil, err
	}
	if len(attachments) != len(attachmentIDs) {
		return nil, apperr.New(apperr.CodeNotFound, "media attachment not found")
	}
	attachmentsByID := make(map[mediadomain.AttachmentID]mediadomain.Attachment, len(attachments))
	for _, attachment := range attachments {
		if attachment.UploaderID() != uploaderID {
			return nil, apperr.New(apperr.CodeNotFound, "media attachment not found")
		}
		if attachment.Kind() != mediadomain.AttachmentKindImage || attachment.Status() != mediadomain.AttachmentStatusReady {
			return nil, apperr.New(apperr.CodeConflict, "media attachment is not ready")
		}
		if attachment.OwnerType() != mediadomain.OwnerTypeNone &&
			!(attachment.OwnerType() == mediadomain.OwnerTypePost && attachment.OwnerID() == postID.String()) {
			return nil, apperr.New(apperr.CodeConflict, "media attachment is already bound")
		}
		attachmentsByID[attachment.ID()] = attachment
	}

	const updateQuery = `
		UPDATE media_attachments
		SET owner_type = 'post',
			owner_id = $2::uuid,
			updated_at = $3
		WHERE id = ANY($1::uuid[])
	`
	if _, err := tx.Exec(ctx, updateQuery, attachmentIDStrings(attachmentIDs), postID.String(), now); err != nil {
		return nil, mapPostgresWriteError("bind post media attachments", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit post attachment binding transaction: %w", err)
	}
	committed = true

	result := make([]mediadomain.Attachment, 0, len(attachmentIDs))
	for _, id := range attachmentIDs {
		attachment := attachmentsByID[id]
		updated, err := mediadomain.RehydrateAttachment(mediadomain.NewAttachmentParams{
			ID:                 attachment.ID(),
			OwnerType:          mediadomain.OwnerTypePost,
			OwnerID:            postID.String(),
			UploaderID:         attachment.UploaderID(),
			Kind:               attachment.Kind(),
			StorageProvider:    attachment.StorageProvider(),
			Bucket:             attachment.Bucket(),
			ObjectKey:          attachment.ObjectKey(),
			PublicURL:          attachment.PublicURL(),
			ThumbnailObjectKey: attachment.ThumbnailObjectKey(),
			Width:              attachment.Width(),
			Height:             attachment.Height(),
			SizeBytes:          attachment.SizeBytes(),
			MimeType:           attachment.MimeType(),
			AltText:            attachment.AltText(),
			Status:             attachment.Status(),
			CreatedAt:          attachment.CreatedAt(),
			UpdatedAt:          now,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, *updated)
	}
	return result, nil
}

func (repo *PostgresMediaRepository) ReplaceReadyImagesForPost(ctx context.Context, postID postdomain.PostID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error) {
	if maxCount <= 0 || len(attachmentIDs) > maxCount {
		return nil, apperr.New(apperr.CodeInvalidArgument, "post image attachment count is invalid")
	}

	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin post attachment replacement transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	attachmentsByID := make(map[mediadomain.AttachmentID]mediadomain.Attachment, len(attachmentIDs))
	if len(attachmentIDs) > 0 {
		attachments, err := lockAttachmentsByIDs(ctx, tx, attachmentIDs)
		if err != nil {
			return nil, err
		}
		if len(attachments) != len(attachmentIDs) {
			return nil, apperr.New(apperr.CodeNotFound, "media attachment not found")
		}
		for _, attachment := range attachments {
			if attachment.UploaderID() != uploaderID {
				return nil, apperr.New(apperr.CodeNotFound, "media attachment not found")
			}
			if attachment.Kind() != mediadomain.AttachmentKindImage || attachment.Status() != mediadomain.AttachmentStatusReady {
				return nil, apperr.New(apperr.CodeConflict, "media attachment is not ready")
			}
			if attachment.OwnerType() != mediadomain.OwnerTypeNone &&
				!(attachment.OwnerType() == mediadomain.OwnerTypePost && attachment.OwnerID() == postID.String()) {
				return nil, apperr.New(apperr.CodeConflict, "media attachment is already bound")
			}
			attachmentsByID[attachment.ID()] = attachment
		}
	}

	if len(attachmentIDs) == 0 {
		const unbindAllQuery = `
			UPDATE media_attachments
			SET owner_type = 'none',
				owner_id = NULL,
				updated_at = $2
			WHERE owner_type = 'post'
				AND owner_id = $1::uuid
				AND kind = 'image'
		`
		if _, err := tx.Exec(ctx, unbindAllQuery, postID.String(), now); err != nil {
			return nil, mapPostgresWriteError("replace post media attachments", err)
		}
	} else {
		const unbindRemovedQuery = `
			UPDATE media_attachments
			SET owner_type = 'none',
				owner_id = NULL,
				updated_at = $3
			WHERE owner_type = 'post'
				AND owner_id = $1::uuid
				AND kind = 'image'
				AND NOT (id = ANY($2::uuid[]))
		`
		if _, err := tx.Exec(ctx, unbindRemovedQuery, postID.String(), attachmentIDStrings(attachmentIDs), now); err != nil {
			return nil, mapPostgresWriteError("replace post media attachments", err)
		}

		const bindSelectedQuery = `
			UPDATE media_attachments
			SET owner_type = 'post',
				owner_id = $2::uuid,
				updated_at = $3
			WHERE id = ANY($1::uuid[])
		`
		if _, err := tx.Exec(ctx, bindSelectedQuery, attachmentIDStrings(attachmentIDs), postID.String(), now); err != nil {
			return nil, mapPostgresWriteError("replace post media attachments", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit post attachment replacement transaction: %w", err)
	}
	committed = true

	result := make([]mediadomain.Attachment, 0, len(attachmentIDs))
	for _, id := range attachmentIDs {
		attachment := attachmentsByID[id]
		updated, err := mediadomain.RehydrateAttachment(mediadomain.NewAttachmentParams{
			ID:                 attachment.ID(),
			OwnerType:          mediadomain.OwnerTypePost,
			OwnerID:            postID.String(),
			UploaderID:         attachment.UploaderID(),
			Kind:               attachment.Kind(),
			StorageProvider:    attachment.StorageProvider(),
			Bucket:             attachment.Bucket(),
			ObjectKey:          attachment.ObjectKey(),
			PublicURL:          attachment.PublicURL(),
			ThumbnailObjectKey: attachment.ThumbnailObjectKey(),
			Width:              attachment.Width(),
			Height:             attachment.Height(),
			SizeBytes:          attachment.SizeBytes(),
			MimeType:           attachment.MimeType(),
			AltText:            attachment.AltText(),
			Status:             attachment.Status(),
			CreatedAt:          attachment.CreatedAt(),
			UpdatedAt:          now,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, *updated)
	}
	return result, nil
}

func (repo *PostgresMediaRepository) ListReadyImagesByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID][]mediadomain.Attachment, error) {
	result := make(map[postdomain.PostID][]mediadomain.Attachment, len(postIDs))
	if len(postIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			id::text,
			owner_type,
			owner_id::text,
			uploader_id::text,
			kind,
			storage_provider,
			bucket,
			object_key,
			public_url,
			thumbnail_object_key,
			width,
			height,
			size_bytes,
			mime_type,
			alt_text,
			status,
			created_at,
			updated_at
		FROM media_attachments
		WHERE owner_type = 'post'
			AND owner_id = ANY($1::uuid[])
			AND kind = 'image'
			AND status = 'ready'
		ORDER BY created_at ASC, id ASC
	`
	rows, err := repo.pool.Query(ctx, query, postIDStrings(postIDs))
	if err != nil {
		return nil, fmt.Errorf("list post media attachments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		attachment, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		ownerID, err := postdomain.NewPostID(attachment.OwnerID())
		if err != nil {
			return nil, fmt.Errorf("parse attachment post owner id: %w", err)
		}
		result[ownerID] = append(result[ownerID], *attachment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate post media attachments: %w", err)
	}
	return result, nil
}

func (repo *PostgresMediaRepository) BindReadyImagesToComment(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error) {
	if len(attachmentIDs) == 0 {
		return []mediadomain.Attachment{}, nil
	}
	if maxCount <= 0 || len(attachmentIDs) > maxCount {
		return nil, apperr.New(apperr.CodeInvalidArgument, "comment image attachment count is invalid")
	}

	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin comment attachment binding transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	attachments, err := lockAttachmentsByIDs(ctx, tx, attachmentIDs)
	if err != nil {
		return nil, err
	}
	if len(attachments) != len(attachmentIDs) {
		return nil, apperr.New(apperr.CodeNotFound, "media attachment not found")
	}
	attachmentsByID := make(map[mediadomain.AttachmentID]mediadomain.Attachment, len(attachments))
	for _, attachment := range attachments {
		if attachment.UploaderID() != uploaderID {
			return nil, apperr.New(apperr.CodeNotFound, "media attachment not found")
		}
		if attachment.Kind() != mediadomain.AttachmentKindImage || attachment.Status() != mediadomain.AttachmentStatusReady {
			return nil, apperr.New(apperr.CodeConflict, "media attachment is not ready")
		}
		if attachment.OwnerType() != mediadomain.OwnerTypeNone &&
			!(attachment.OwnerType() == mediadomain.OwnerTypeComment && attachment.OwnerID() == commentID.String()) {
			return nil, apperr.New(apperr.CodeConflict, "media attachment is already bound")
		}
		attachmentsByID[attachment.ID()] = attachment
	}

	const updateQuery = `
		UPDATE media_attachments
		SET owner_type = 'comment',
			owner_id = $2::uuid,
			updated_at = $3
		WHERE id = ANY($1::uuid[])
	`
	if _, err := tx.Exec(ctx, updateQuery, attachmentIDStrings(attachmentIDs), commentID.String(), now); err != nil {
		return nil, mapPostgresWriteError("bind comment media attachments", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit comment attachment binding transaction: %w", err)
	}
	committed = true

	result := make([]mediadomain.Attachment, 0, len(attachmentIDs))
	for _, id := range attachmentIDs {
		attachment := attachmentsByID[id]
		updated, err := mediadomain.RehydrateAttachment(mediadomain.NewAttachmentParams{
			ID:                 attachment.ID(),
			OwnerType:          mediadomain.OwnerTypeComment,
			OwnerID:            commentID.String(),
			UploaderID:         attachment.UploaderID(),
			Kind:               attachment.Kind(),
			StorageProvider:    attachment.StorageProvider(),
			Bucket:             attachment.Bucket(),
			ObjectKey:          attachment.ObjectKey(),
			PublicURL:          attachment.PublicURL(),
			ThumbnailObjectKey: attachment.ThumbnailObjectKey(),
			Width:              attachment.Width(),
			Height:             attachment.Height(),
			SizeBytes:          attachment.SizeBytes(),
			MimeType:           attachment.MimeType(),
			AltText:            attachment.AltText(),
			Status:             attachment.Status(),
			CreatedAt:          attachment.CreatedAt(),
			UpdatedAt:          now,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, *updated)
	}
	return result, nil
}

func (repo *PostgresMediaRepository) ReplaceReadyImagesForComment(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error) {
	if maxCount <= 0 || len(attachmentIDs) > maxCount {
		return nil, apperr.New(apperr.CodeInvalidArgument, "comment image attachment count is invalid")
	}

	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin comment attachment replacement transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	attachmentsByID := make(map[mediadomain.AttachmentID]mediadomain.Attachment, len(attachmentIDs))
	if len(attachmentIDs) > 0 {
		attachments, err := lockAttachmentsByIDs(ctx, tx, attachmentIDs)
		if err != nil {
			return nil, err
		}
		if len(attachments) != len(attachmentIDs) {
			return nil, apperr.New(apperr.CodeNotFound, "media attachment not found")
		}
		for _, attachment := range attachments {
			if attachment.UploaderID() != uploaderID {
				return nil, apperr.New(apperr.CodeNotFound, "media attachment not found")
			}
			if attachment.Kind() != mediadomain.AttachmentKindImage || attachment.Status() != mediadomain.AttachmentStatusReady {
				return nil, apperr.New(apperr.CodeConflict, "media attachment is not ready")
			}
			if attachment.OwnerType() != mediadomain.OwnerTypeNone &&
				!(attachment.OwnerType() == mediadomain.OwnerTypeComment && attachment.OwnerID() == commentID.String()) {
				return nil, apperr.New(apperr.CodeConflict, "media attachment is already bound")
			}
			attachmentsByID[attachment.ID()] = attachment
		}
	}

	if len(attachmentIDs) == 0 {
		const unbindAllQuery = `
			UPDATE media_attachments
			SET owner_type = 'none',
				owner_id = NULL,
				updated_at = $2
			WHERE owner_type = 'comment'
				AND owner_id = $1::uuid
				AND kind = 'image'
		`
		if _, err := tx.Exec(ctx, unbindAllQuery, commentID.String(), now); err != nil {
			return nil, mapPostgresWriteError("replace comment media attachments", err)
		}
	} else {
		const unbindRemovedQuery = `
			UPDATE media_attachments
			SET owner_type = 'none',
				owner_id = NULL,
				updated_at = $3
			WHERE owner_type = 'comment'
				AND owner_id = $1::uuid
				AND kind = 'image'
				AND NOT (id = ANY($2::uuid[]))
		`
		if _, err := tx.Exec(ctx, unbindRemovedQuery, commentID.String(), attachmentIDStrings(attachmentIDs), now); err != nil {
			return nil, mapPostgresWriteError("replace comment media attachments", err)
		}

		const bindSelectedQuery = `
			UPDATE media_attachments
			SET owner_type = 'comment',
				owner_id = $2::uuid,
				updated_at = $3
			WHERE id = ANY($1::uuid[])
		`
		if _, err := tx.Exec(ctx, bindSelectedQuery, attachmentIDStrings(attachmentIDs), commentID.String(), now); err != nil {
			return nil, mapPostgresWriteError("replace comment media attachments", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit comment attachment replacement transaction: %w", err)
	}
	committed = true

	result := make([]mediadomain.Attachment, 0, len(attachmentIDs))
	for _, id := range attachmentIDs {
		attachment := attachmentsByID[id]
		updated, err := mediadomain.RehydrateAttachment(mediadomain.NewAttachmentParams{
			ID:                 attachment.ID(),
			OwnerType:          mediadomain.OwnerTypeComment,
			OwnerID:            commentID.String(),
			UploaderID:         attachment.UploaderID(),
			Kind:               attachment.Kind(),
			StorageProvider:    attachment.StorageProvider(),
			Bucket:             attachment.Bucket(),
			ObjectKey:          attachment.ObjectKey(),
			PublicURL:          attachment.PublicURL(),
			ThumbnailObjectKey: attachment.ThumbnailObjectKey(),
			Width:              attachment.Width(),
			Height:             attachment.Height(),
			SizeBytes:          attachment.SizeBytes(),
			MimeType:           attachment.MimeType(),
			AltText:            attachment.AltText(),
			Status:             attachment.Status(),
			CreatedAt:          attachment.CreatedAt(),
			UpdatedAt:          now,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, *updated)
	}
	return result, nil
}

func (repo *PostgresMediaRepository) ListReadyImagesByCommentIDs(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID][]mediadomain.Attachment, error) {
	result := make(map[commentdomain.CommentID][]mediadomain.Attachment, len(commentIDs))
	if len(commentIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			id::text,
			owner_type,
			owner_id::text,
			uploader_id::text,
			kind,
			storage_provider,
			bucket,
			object_key,
			public_url,
			thumbnail_object_key,
			width,
			height,
			size_bytes,
			mime_type,
			alt_text,
			status,
			created_at,
			updated_at
		FROM media_attachments
		WHERE owner_type = 'comment'
			AND owner_id = ANY($1::uuid[])
			AND kind = 'image'
			AND status = 'ready'
		ORDER BY created_at ASC, id ASC
	`
	rows, err := repo.pool.Query(ctx, query, commentIDStrings(commentIDs))
	if err != nil {
		return nil, fmt.Errorf("list comment media attachments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		attachment, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		ownerID, err := commentdomain.NewCommentID(attachment.OwnerID())
		if err != nil {
			return nil, fmt.Errorf("parse attachment comment owner id: %w", err)
		}
		result[ownerID] = append(result[ownerID], *attachment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comment media attachments: %w", err)
	}
	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

type txScanner interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func lockAttachmentsByIDs(ctx context.Context, tx txScanner, attachmentIDs []mediadomain.AttachmentID) ([]mediadomain.Attachment, error) {
	const query = `
		SELECT
			id::text,
			owner_type,
			owner_id::text,
			uploader_id::text,
			kind,
			storage_provider,
			bucket,
			object_key,
			public_url,
			thumbnail_object_key,
			width,
			height,
			size_bytes,
			mime_type,
			alt_text,
			status,
			created_at,
			updated_at
		FROM media_attachments
		WHERE id = ANY($1::uuid[])
		FOR UPDATE
	`
	rows, err := tx.Query(ctx, query, attachmentIDStrings(attachmentIDs))
	if err != nil {
		return nil, fmt.Errorf("lock media attachments: %w", err)
	}
	defer rows.Close()

	attachments := make([]mediadomain.Attachment, 0, len(attachmentIDs))
	for rows.Next() {
		attachment, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, *attachment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate locked media attachments: %w", err)
	}
	return attachments, nil
}

func scanAttachment(row rowScanner) (*mediadomain.Attachment, error) {
	var rawID string
	var rawOwnerType string
	var rawOwnerID pgtype.Text
	var rawUploaderID string
	var rawKind string
	var rawStorageProvider string
	var rawBucket string
	var rawObjectKey string
	var rawPublicURL string
	var rawThumbnailObjectKey pgtype.Text
	var rawWidth pgtype.Int4
	var rawHeight pgtype.Int4
	var rawSizeBytes int64
	var rawMimeType string
	var rawAltText pgtype.Text
	var rawStatus string
	var createdAt time.Time
	var updatedAt time.Time

	if err := row.Scan(
		&rawID,
		&rawOwnerType,
		&rawOwnerID,
		&rawUploaderID,
		&rawKind,
		&rawStorageProvider,
		&rawBucket,
		&rawObjectKey,
		&rawPublicURL,
		&rawThumbnailObjectKey,
		&rawWidth,
		&rawHeight,
		&rawSizeBytes,
		&rawMimeType,
		&rawAltText,
		&rawStatus,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	id, err := mediadomain.NewAttachmentID(rawID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate media attachment id: %v", err)
	}
	ownerType, err := mediadomain.NewOwnerType(rawOwnerType)
	if err != nil {
		return nil, fmt.Errorf("rehydrate media attachment owner type: %v", err)
	}
	uploaderID, err := userdomain.NewUserID(rawUploaderID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate media attachment uploader id: %v", err)
	}
	kind, err := mediadomain.NewAttachmentKind(rawKind)
	if err != nil {
		return nil, fmt.Errorf("rehydrate media attachment kind: %v", err)
	}
	provider, err := mediadomain.NewStorageProvider(rawStorageProvider)
	if err != nil {
		return nil, fmt.Errorf("rehydrate media attachment storage provider: %v", err)
	}
	status, err := mediadomain.NewAttachmentStatus(rawStatus)
	if err != nil {
		return nil, fmt.Errorf("rehydrate media attachment status: %v", err)
	}

	attachment, err := mediadomain.RehydrateAttachment(mediadomain.NewAttachmentParams{
		ID:                 id,
		OwnerType:          ownerType,
		OwnerID:            nullableTextString(rawOwnerID),
		UploaderID:         uploaderID,
		Kind:               kind,
		StorageProvider:    provider,
		Bucket:             rawBucket,
		ObjectKey:          rawObjectKey,
		PublicURL:          rawPublicURL,
		ThumbnailObjectKey: nullableTextString(rawThumbnailObjectKey),
		Width:              nullableInt32(rawWidth),
		Height:             nullableInt32(rawHeight),
		SizeBytes:          rawSizeBytes,
		MimeType:           rawMimeType,
		AltText:            nullableTextString(rawAltText),
		Status:             status,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("rehydrate media attachment: %v", err)
	}
	return attachment, nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTextString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullableInt32(value pgtype.Int4) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int32)
	return &converted
}

func attachmentIDStrings(ids []mediadomain.AttachmentID) []string {
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, id.String())
	}
	return values
}

func postIDStrings(ids []postdomain.PostID) []string {
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, id.String())
	}
	return values
}

func commentIDStrings(ids []commentdomain.CommentID) []string {
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, id.String())
	}
	return values
}

func mapPostgresWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return apperr.New(apperr.CodeConflict, "media attachment already exists")
		}
		if pgErr.Code == "23503" {
			return apperr.New(apperr.CodeNotFound, "related record not found")
		}
		if pgErr.Code == "23514" {
			return apperr.New(apperr.CodeInvalidArgument, "media attachment is invalid")
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

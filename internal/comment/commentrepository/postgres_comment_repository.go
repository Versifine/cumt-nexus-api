package commentrepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentusecase"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ commentusecase.CommentRepository = (*PostgresCommentRepository)(nil)

type PostgresCommentRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresCommentRepository(pool *pgxpool.Pool) *PostgresCommentRepository {
	return &PostgresCommentRepository{
		pool: pool,
	}
}

func (repo *PostgresCommentRepository) Create(ctx context.Context, comment commentdomain.Comment) error {
	const query = `
		INSERT INTO comments (
			id,
			post_id,
			author_id,
			parent_id,
			body,
			status,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8)
	`

	parentID, hasParentID := comment.ParentID()
	_, err := repo.pool.Exec(
		ctx,
		query,
		comment.ID().String(),
		comment.PostID().String(),
		comment.AuthorID().String(),
		nullableCommentIDValue(parentID, hasParentID),
		comment.Body().String(),
		comment.Status().String(),
		comment.CreatedAt(),
		comment.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("create comment", err)
	}

	return nil
}

func (repo *PostgresCommentRepository) FindVisibleByID(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
	const query = `
		SELECT
			id::text,
			post_id::text,
			author_id::text,
			parent_id::text,
			body,
			status,
			created_at,
			updated_at
		FROM comments
		WHERE id = $1::uuid
			AND status = 'visible'
		LIMIT 1
	`

	comment, err := scanComment(repo.pool.QueryRow(ctx, query, id.String()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeNotFound, "comment not found")
		}
		return nil, err
	}

	return comment, nil
}

func (repo *PostgresCommentRepository) UpdateContent(ctx context.Context, comment commentdomain.Comment) error {
	const query = `
		UPDATE comments
		SET
			body = $2,
			updated_at = $3
		WHERE id = $1::uuid
			AND status = 'visible'
	`

	tag, err := repo.pool.Exec(
		ctx,
		query,
		comment.ID().String(),
		comment.Body().String(),
		comment.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("update comment content", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "comment not found")
	}

	return nil
}

func (repo *PostgresCommentRepository) MarkDeleted(ctx context.Context, comment commentdomain.Comment) error {
	const query = `
		UPDATE comments
		SET
			status = 'deleted',
			updated_at = $2
		WHERE id = $1::uuid
			AND status = 'visible'
	`

	tag, err := repo.pool.Exec(ctx, query, comment.ID().String(), comment.UpdatedAt())
	if err != nil {
		return mapPostgresWriteError("delete comment", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "comment not found")
	}

	return nil
}

func (repo *PostgresCommentRepository) ListVisibleByPost(ctx context.Context, postID postdomain.PostID, limit int, offset int) ([]commentdomain.Comment, error) {
	const query = `
		SELECT
			id::text,
			post_id::text,
			author_id::text,
			parent_id::text,
			body,
			status,
			created_at,
			updated_at
		FROM comments
		WHERE post_id = $1::uuid
			AND status = 'visible'
		ORDER BY created_at DESC, id DESC
		LIMIT $2
		OFFSET $3
	`

	rows, err := repo.pool.Query(ctx, query, postID.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list visible comments by post: %w", err)
	}
	defer rows.Close()

	var comments []commentdomain.Comment
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, *comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate visible comments by post: %w", err)
	}

	return comments, nil
}

func (repo *PostgresCommentRepository) ListVisibleTreeByPost(ctx context.Context, postID postdomain.PostID) ([]commentdomain.Comment, error) {
	const query = `
		SELECT
			id::text,
			post_id::text,
			author_id::text,
			parent_id::text,
			body,
			status,
			created_at,
			updated_at
		FROM comments
		WHERE post_id = $1::uuid
			AND status = 'visible'
		ORDER BY created_at DESC, id DESC
	`

	rows, err := repo.pool.Query(ctx, query, postID.String())
	if err != nil {
		return nil, fmt.Errorf("list visible comment tree by post: %w", err)
	}
	defer rows.Close()

	var comments []commentdomain.Comment
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, *comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate visible comment tree by post: %w", err)
	}

	return comments, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanComment(row rowScanner) (*commentdomain.Comment, error) {
	var rawID string
	var rawPostID string
	var rawAuthorID string
	var rawParentID pgtype.Text
	var rawBody string
	var rawStatus string
	var createdAt time.Time
	var updatedAt time.Time

	if err := row.Scan(
		&rawID,
		&rawPostID,
		&rawAuthorID,
		&rawParentID,
		&rawBody,
		&rawStatus,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	id, err := commentdomain.NewCommentID(rawID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate comment id: %v", err)
	}
	postID, err := postdomain.NewPostID(rawPostID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate comment post id: %v", err)
	}
	authorID, err := userdomain.NewUserID(rawAuthorID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate comment author id: %v", err)
	}
	var parentID *commentdomain.CommentID
	if rawParentID.Valid {
		parsedParentID, err := commentdomain.NewCommentID(rawParentID.String)
		if err != nil {
			return nil, fmt.Errorf("rehydrate comment parent id: %v", err)
		}
		parentID = &parsedParentID
	}
	body, err := commentdomain.NewCommentBody(rawBody)
	if err != nil {
		return nil, fmt.Errorf("rehydrate comment body: %v", err)
	}
	status, err := commentdomain.NewCommentStatus(rawStatus)
	if err != nil {
		return nil, fmt.Errorf("rehydrate comment status: %v", err)
	}

	comment, err := commentdomain.RehydrateComment(id, postID, authorID, parentID, body, status, createdAt, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("rehydrate comment: %v", err)
	}

	return comment, nil
}

func nullableCommentIDValue(id commentdomain.CommentID, ok bool) any {
	if !ok {
		return nil
	}
	return id.String()
}

func mapPostgresWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return apperr.New(apperr.CodeConflict, "comment already exists")
		}
		if pgErr.Code == "23503" {
			return apperr.New(apperr.CodeNotFound, "related record not found")
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

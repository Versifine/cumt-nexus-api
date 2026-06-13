package contentrefrepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/contentref/contentrefusecase"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ contentrefusecase.EmbedRepository = (*PostgresContentRefRepository)(nil)

type PostgresContentRefRepository struct {
	db *pgxpool.Pool
}

func NewPostgresContentRefRepository(db *pgxpool.Pool) *PostgresContentRefRepository {
	return &PostgresContentRefRepository{db: db}
}

func (repo *PostgresContentRefRepository) UpsertEmbed(ctx context.Context, embed contentrefusecase.Embed, now time.Time) (contentrefusecase.Embed, error) {
	const query = `
		INSERT INTO embeds (
			id,
			provider,
			provider_ref,
			url,
			canonical_url,
			embed_url,
			iframe_allowed,
			title,
			description,
			image_url,
			author_name,
			status,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13)
		ON CONFLICT (provider, provider_ref)
		DO UPDATE SET
			url = EXCLUDED.url,
			canonical_url = EXCLUDED.canonical_url,
			embed_url = EXCLUDED.embed_url,
			iframe_allowed = EXCLUDED.iframe_allowed,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			image_url = EXCLUDED.image_url,
			author_name = EXCLUDED.author_name,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
		RETURNING
			id::text,
			provider,
			provider_ref,
			url,
			canonical_url,
			embed_url,
			iframe_allowed,
			title,
			description,
			image_url,
			author_name,
			status,
			created_at,
			updated_at
	`

	row := repo.db.QueryRow(
		ctx,
		query,
		embed.ID,
		embed.Provider,
		embed.ProviderRef,
		embed.URL,
		embed.CanonicalURL,
		embed.EmbedURL,
		embed.IframeAllowed,
		embed.Title,
		embed.Description,
		embed.ImageURL,
		embed.AuthorName,
		embed.Status,
		now,
	)
	stored, err := scanEmbed(row)
	if err != nil {
		return contentrefusecase.Embed{}, mapPostgresWriteError("upsert embed", err)
	}
	return stored, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEmbed(row rowScanner) (contentrefusecase.Embed, error) {
	var embed contentrefusecase.Embed
	if err := row.Scan(
		&embed.ID,
		&embed.Provider,
		&embed.ProviderRef,
		&embed.URL,
		&embed.CanonicalURL,
		&embed.EmbedURL,
		&embed.IframeAllowed,
		&embed.Title,
		&embed.Description,
		&embed.ImageURL,
		&embed.AuthorName,
		&embed.Status,
		&embed.CreatedAt,
		&embed.UpdatedAt,
	); err != nil {
		return contentrefusecase.Embed{}, err
	}
	return embed, nil
}

func mapPostgresWriteError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.New(apperr.CodeNotFound, "embed not found")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperr.New(apperr.CodeConflict, "embed already exists")
		case "23514", "22P02":
			return apperr.New(apperr.CodeInvalidArgument, "embed is invalid")
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

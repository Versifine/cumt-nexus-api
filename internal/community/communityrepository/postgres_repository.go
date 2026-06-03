package communityrepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communityusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ communityusecase.CommunityRepository = (*PostgresCommunityRepository)(nil)
var _ communityusecase.CommunityMembershipRepository = (*PostgresMembershipRepository)(nil)
var _ communityusecase.CommunityApplicationRepository = (*PostgresApplicationRepository)(nil)
var _ communityusecase.PlatformStaffRepository = (*PostgresPlatformStaffRepository)(nil)
var _ communityusecase.CommunityTransactionManager = (*PostgresCommunityTransactionManager)(nil)

type postgresExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresCommunityRepository struct {
	db postgresExecutor
}

func NewPostgresCommunityRepository(pool *pgxpool.Pool) *PostgresCommunityRepository {
	return &PostgresCommunityRepository{
		db: pool,
	}
}

func (repo *PostgresCommunityRepository) Create(ctx context.Context, community communitydomain.Community) error {
	const query = `
		INSERT INTO communities (
			id,
			slug,
			name,
			description,
			kind,
			status,
			visibility,
			created_by,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8::uuid, $9, $10)
	`

	createdBy, hasCreatedBy := community.CreatedBy()
	_, err := repo.db.Exec(
		ctx,
		query,
		community.ID().String(),
		community.Slug().String(),
		community.Name().String(),
		community.Description().String(),
		community.Kind().String(),
		community.Status().String(),
		community.Visibility().String(),
		nullableUserIDValue(createdBy, hasCreatedBy),
		community.CreatedAt(),
		community.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("create community", err)
	}

	return nil
}

func (repo *PostgresCommunityRepository) FindByID(ctx context.Context, id communitydomain.CommunityID) (*communitydomain.Community, error) {
	const query = `
		SELECT
			id::text,
			slug,
			name,
			description,
			kind,
			status,
			visibility,
			created_by::text,
			created_at,
			updated_at
		FROM communities
		WHERE id = $1::uuid
		LIMIT 1
	`

	row := repo.db.QueryRow(ctx, query, id.String())
	community, err := scanCommunity(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeNotFound, "community not found")
		}
		return nil, err
	}

	return community, nil
}

func (repo *PostgresCommunityRepository) FindBySlug(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
	const query = `
		SELECT
			id::text,
			slug,
			name,
			description,
			kind,
			status,
			visibility,
			created_by::text,
			created_at,
			updated_at
		FROM communities
		WHERE slug = $1
		LIMIT 1
	`

	row := repo.db.QueryRow(ctx, query, slug.String())
	community, err := scanCommunity(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeNotFound, "community not found")
		}
		return nil, err
	}

	return community, nil
}

func (repo *PostgresCommunityRepository) ListActivePublic(ctx context.Context) ([]communitydomain.Community, error) {
	const query = `
		SELECT
			id::text,
			slug,
			name,
			description,
			kind,
			status,
			visibility,
			created_by::text,
			created_at,
			updated_at
		FROM communities
		WHERE status = 'active'
			AND visibility = 'public'
		ORDER BY created_at ASC, slug ASC
	`

	rows, err := repo.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list active public communities: %w", err)
	}
	defer rows.Close()

	var communities []communitydomain.Community
	for rows.Next() {
		community, err := scanCommunity(rows)
		if err != nil {
			return nil, err
		}
		communities = append(communities, *community)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active public communities: %w", err)
	}

	return communities, nil
}

type PostgresMembershipRepository struct {
	db postgresExecutor
}

func NewPostgresMembershipRepository(pool *pgxpool.Pool) *PostgresMembershipRepository {
	return &PostgresMembershipRepository{
		db: pool,
	}
}

func (repo *PostgresMembershipRepository) Create(ctx context.Context, membership communitydomain.CommunityMembership) error {
	const query = `
		INSERT INTO community_memberships (
			community_id,
			user_id,
			role,
			status,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
	`

	_, err := repo.db.Exec(
		ctx,
		query,
		membership.CommunityID().String(),
		membership.UserID().String(),
		membership.Role().String(),
		membership.Status().String(),
		membership.CreatedAt(),
		membership.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("create community membership", err)
	}

	return nil
}

type PostgresApplicationRepository struct {
	db postgresExecutor
}

func NewPostgresApplicationRepository(pool *pgxpool.Pool) *PostgresApplicationRepository {
	return &PostgresApplicationRepository{
		db: pool,
	}
}

func (repo *PostgresApplicationRepository) Create(ctx context.Context, application communitydomain.CommunityApplication) error {
	const query = `
		INSERT INTO community_applications (
			id,
			applicant_id,
			requested_slug,
			requested_name,
			reason,
			status,
			reviewed_by,
			reviewed_at,
			reject_reason,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7::uuid, $8, $9, $10, $11)
	`

	reviewedBy, hasReviewedBy := application.ReviewedBy()
	reviewedAt, hasReviewedAt := application.ReviewedAt()
	rejectReason, hasRejectReason := application.RejectReason()

	_, err := repo.db.Exec(
		ctx,
		query,
		application.ID().String(),
		application.ApplicantID().String(),
		application.RequestedSlug().String(),
		application.RequestedName().String(),
		application.Reason().String(),
		application.Status().String(),
		nullableUserIDValue(reviewedBy, hasReviewedBy),
		nullableTimeValue(reviewedAt, hasReviewedAt),
		nullableRejectReasonValue(rejectReason, hasRejectReason),
		application.CreatedAt(),
		application.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("create community application", err)
	}

	return nil
}

func (repo *PostgresApplicationRepository) FindByID(ctx context.Context, id communitydomain.CommunityApplicationID) (*communitydomain.CommunityApplication, error) {
	return repo.findByID(ctx, id, false)
}

func (repo *PostgresApplicationRepository) FindByIDForUpdate(ctx context.Context, id communitydomain.CommunityApplicationID) (*communitydomain.CommunityApplication, error) {
	return repo.findByID(ctx, id, true)
}

func (repo *PostgresApplicationRepository) ListByStatus(ctx context.Context, status communitydomain.ApplicationStatus, limit int, offset int) ([]communitydomain.CommunityApplication, error) {
	const query = `
		SELECT
			id::text,
			applicant_id::text,
			requested_slug,
			requested_name,
			reason,
			status,
			reviewed_by::text,
			reviewed_at,
			reject_reason,
			created_at,
			updated_at
		FROM community_applications
		WHERE status = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := repo.db.Query(ctx, query, status.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list community applications by status: %w", err)
	}
	defer rows.Close()

	var applications []communitydomain.CommunityApplication
	for rows.Next() {
		application, err := scanCommunityApplication(rows)
		if err != nil {
			return nil, err
		}
		applications = append(applications, *application)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate community applications by status: %w", err)
	}

	return applications, nil
}

func (repo *PostgresApplicationRepository) findByID(ctx context.Context, id communitydomain.CommunityApplicationID, forUpdate bool) (*communitydomain.CommunityApplication, error) {
	const query = `
		SELECT
			id::text,
			applicant_id::text,
			requested_slug,
			requested_name,
			reason,
			status,
			reviewed_by::text,
			reviewed_at,
			reject_reason,
			created_at,
			updated_at
		FROM community_applications
		WHERE id = $1::uuid
		LIMIT 1
	`
	const queryForUpdate = `
		SELECT
			id::text,
			applicant_id::text,
			requested_slug,
			requested_name,
			reason,
			status,
			reviewed_by::text,
			reviewed_at,
			reject_reason,
			created_at,
			updated_at
		FROM community_applications
		WHERE id = $1::uuid
		LIMIT 1
		FOR UPDATE
	`

	queryText := query
	if forUpdate {
		queryText = queryForUpdate
	}

	row := repo.db.QueryRow(ctx, queryText, id.String())
	application, err := scanCommunityApplication(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeNotFound, "community application not found")
		}
		return nil, err
	}

	return application, nil
}

func (repo *PostgresApplicationRepository) Save(ctx context.Context, application communitydomain.CommunityApplication) error {
	const query = `
		UPDATE community_applications
		SET status = $2,
			reviewed_by = $3::uuid,
			reviewed_at = $4,
			reject_reason = $5,
			updated_at = $6
		WHERE id = $1::uuid
	`

	reviewedBy, hasReviewedBy := application.ReviewedBy()
	reviewedAt, hasReviewedAt := application.ReviewedAt()
	rejectReason, hasRejectReason := application.RejectReason()

	tag, err := repo.db.Exec(
		ctx,
		query,
		application.ID().String(),
		application.Status().String(),
		nullableUserIDValue(reviewedBy, hasReviewedBy),
		nullableTimeValue(reviewedAt, hasReviewedAt),
		nullableRejectReasonValue(rejectReason, hasRejectReason),
		application.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("save community application", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "community application not found")
	}

	return nil
}

type PostgresPlatformStaffRepository struct {
	db postgresExecutor
}

func NewPostgresPlatformStaffRepository(pool *pgxpool.Pool) *PostgresPlatformStaffRepository {
	return &PostgresPlatformStaffRepository{
		db: pool,
	}
}

func (repo *PostgresPlatformStaffRepository) IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error) {
	const query = `
		SELECT is_platform_staff
		FROM users
		WHERE id = $1::uuid
		LIMIT 1
	`

	var isPlatformStaff bool
	if err := repo.db.QueryRow(ctx, query, userID.String()).Scan(&isPlatformStaff); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, apperr.New(apperr.CodeNotFound, "user not found")
		}
		return false, fmt.Errorf("check platform staff: %w", err)
	}

	return isPlatformStaff, nil
}

type PostgresCommunityTransactionManager struct {
	pool *pgxpool.Pool
}

func NewPostgresCommunityTransactionManager(pool *pgxpool.Pool) *PostgresCommunityTransactionManager {
	return &PostgresCommunityTransactionManager{
		pool: pool,
	}
}

func (manager *PostgresCommunityTransactionManager) WithinTx(ctx context.Context, fn func(ctx context.Context, repositories communityusecase.CommunityRepositories) error) (err error) {
	tx, err := manager.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin community transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	repositories := postgresCommunityRepositories{
		communities:  &PostgresCommunityRepository{db: tx},
		memberships:  &PostgresMembershipRepository{db: tx},
		applications: &PostgresApplicationRepository{db: tx},
	}
	if err := fn(ctx, repositories); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit community transaction: %w", err)
	}
	committed = true
	return nil
}

type postgresCommunityRepositories struct {
	communities  communityusecase.CommunityRepository
	memberships  communityusecase.CommunityMembershipRepository
	applications communityusecase.CommunityApplicationRepository
}

func (repositories postgresCommunityRepositories) Communities() communityusecase.CommunityRepository {
	return repositories.communities
}

func (repositories postgresCommunityRepositories) Memberships() communityusecase.CommunityMembershipRepository {
	return repositories.memberships
}

func (repositories postgresCommunityRepositories) Applications() communityusecase.CommunityApplicationRepository {
	return repositories.applications
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCommunity(row rowScanner) (*communitydomain.Community, error) {
	var rawID string
	var rawSlug string
	var rawName string
	var rawDescription string
	var rawKind string
	var rawStatus string
	var rawVisibility string
	var rawCreatedBy pgtype.Text
	var createdAt pgtype.Timestamptz
	var updatedAt pgtype.Timestamptz

	if err := row.Scan(
		&rawID,
		&rawSlug,
		&rawName,
		&rawDescription,
		&rawKind,
		&rawStatus,
		&rawVisibility,
		&rawCreatedBy,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	id, err := communitydomain.NewCommunityID(rawID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community id: %v", err)
	}

	slug, err := communitydomain.NewCommunitySlug(rawSlug)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community slug: %v", err)
	}

	name, err := communitydomain.NewCommunityName(rawName)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community name: %v", err)
	}

	kind, err := communitydomain.NewCommunityKind(rawKind)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community kind: %v", err)
	}

	status, err := communitydomain.NewCommunityStatus(rawStatus)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community status: %v", err)
	}

	visibility, err := communitydomain.NewCommunityVisibility(rawVisibility)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community visibility: %v", err)
	}

	var createdBy *userdomain.UserID
	if rawCreatedBy.Valid {
		parsedCreatedBy, err := userdomain.NewUserID(rawCreatedBy.String)
		if err != nil {
			return nil, fmt.Errorf("rehydrate community creator: %v", err)
		}
		createdBy = &parsedCreatedBy
	}

	if !createdAt.Valid || !updatedAt.Valid {
		return nil, fmt.Errorf("rehydrate community timestamps: missing timestamp")
	}

	community, err := communitydomain.RehydrateCommunity(
		id,
		slug,
		name,
		communitydomain.NewCommunityDescription(rawDescription),
		kind,
		status,
		visibility,
		createdBy,
		createdAt.Time,
		updatedAt.Time,
	)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community: %v", err)
	}

	return community, nil
}

func scanCommunityApplication(row rowScanner) (*communitydomain.CommunityApplication, error) {
	var rawID string
	var rawApplicantID string
	var rawRequestedSlug string
	var rawRequestedName string
	var rawReason string
	var rawStatus string
	var rawReviewedBy pgtype.Text
	var rawReviewedAt pgtype.Timestamptz
	var rawRejectReason pgtype.Text
	var createdAt pgtype.Timestamptz
	var updatedAt pgtype.Timestamptz

	if err := row.Scan(
		&rawID,
		&rawApplicantID,
		&rawRequestedSlug,
		&rawRequestedName,
		&rawReason,
		&rawStatus,
		&rawReviewedBy,
		&rawReviewedAt,
		&rawRejectReason,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	id, err := communitydomain.NewCommunityApplicationID(rawID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community application id: %v", err)
	}

	applicantID, err := userdomain.NewUserID(rawApplicantID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community application applicant: %v", err)
	}

	requestedSlug, err := communitydomain.NewCommunitySlug(rawRequestedSlug)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community application requested slug: %v", err)
	}

	requestedName, err := communitydomain.NewCommunityName(rawRequestedName)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community application requested name: %v", err)
	}

	reason, err := communitydomain.NewApplicationReason(rawReason)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community application reason: %v", err)
	}

	status, err := communitydomain.NewApplicationStatus(rawStatus)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community application status: %v", err)
	}

	var reviewedBy *userdomain.UserID
	if rawReviewedBy.Valid {
		parsedReviewedBy, err := userdomain.NewUserID(rawReviewedBy.String)
		if err != nil {
			return nil, fmt.Errorf("rehydrate community application reviewer: %v", err)
		}
		reviewedBy = &parsedReviewedBy
	}

	var reviewedAt *time.Time
	if rawReviewedAt.Valid {
		reviewedAt = &rawReviewedAt.Time
	}

	var rejectReason *communitydomain.RejectReason
	if rawRejectReason.Valid {
		parsedRejectReason, err := communitydomain.NewRejectReason(rawRejectReason.String)
		if err != nil {
			return nil, fmt.Errorf("rehydrate community application reject reason: %v", err)
		}
		rejectReason = &parsedRejectReason
	}

	if !createdAt.Valid || !updatedAt.Valid {
		return nil, fmt.Errorf("rehydrate community application timestamps: missing timestamp")
	}

	application, err := communitydomain.RehydrateCommunityApplication(
		id,
		applicantID,
		requestedSlug,
		requestedName,
		reason,
		status,
		reviewedBy,
		reviewedAt,
		rejectReason,
		createdAt.Time,
		updatedAt.Time,
	)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community application: %v", err)
	}

	return application, nil
}

func nullableUserIDValue(id userdomain.UserID, ok bool) any {
	if !ok {
		return nil
	}
	return id.String()
}

func nullableTimeValue(value time.Time, ok bool) any {
	if !ok {
		return nil
	}
	return value
}

func nullableRejectReasonValue(reason communitydomain.RejectReason, ok bool) any {
	if !ok {
		return nil
	}
	return reason.String()
}

func mapPostgresWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "communities_slug_uq":
				return apperr.New(apperr.CodeConflict, "community slug already exists")
			case "community_memberships_pk":
				return apperr.New(apperr.CodeConflict, "community membership already exists")
			case "community_applications_pending_slug_uq":
				return apperr.New(apperr.CodeConflict, "pending community application slug already exists")
			}
		}
		if pgErr.Code == "23503" {
			return apperr.New(apperr.CodeNotFound, "related record not found")
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

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
var _ communityusecase.CommunitySettingsRepository = (*PostgresCommunityRepository)(nil)
var _ communityusecase.CommunityStatsRepository = (*PostgresCommunityRepository)(nil)
var _ communityusecase.CommunityFollowRepository = (*PostgresCommunityRepository)(nil)
var _ communityusecase.CommunityRuleRepository = (*PostgresCommunityRepository)(nil)
var _ communityusecase.CommunityMembershipRepository = (*PostgresMembershipRepository)(nil)
var _ communityusecase.CommunityMembershipReadRepository = (*PostgresMembershipRepository)(nil)
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

func (repo *PostgresCommunityRepository) UpdateDetails(ctx context.Context, community communitydomain.Community) error {
	const query = `
		UPDATE communities
		SET name = $2,
			description = $3,
			updated_at = $4
		WHERE id = $1::uuid
	`

	tag, err := repo.db.Exec(
		ctx,
		query,
		community.ID().String(),
		community.Name().String(),
		community.Description().String(),
		community.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("update community details", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "community not found")
	}

	return nil
}

func (repo *PostgresCommunityRepository) ListRules(ctx context.Context, communityID communitydomain.CommunityID) ([]communitydomain.CommunityRule, error) {
	const query = `
		SELECT
			id::text,
			community_id::text,
			title,
			body,
			position,
			created_by::text,
			updated_by::text,
			created_at,
			updated_at
		FROM community_rules
		WHERE community_id = $1::uuid
		ORDER BY position ASC, created_at ASC, id ASC
	`

	rows, err := repo.db.Query(ctx, query, communityID.String())
	if err != nil {
		return nil, fmt.Errorf("list community rules: %w", err)
	}
	defer rows.Close()

	rules := make([]communitydomain.CommunityRule, 0)
	for rows.Next() {
		rule, err := scanCommunityRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, *rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate community rules: %w", err)
	}

	return rules, nil
}

func (repo *PostgresCommunityRepository) FindRuleByID(ctx context.Context, id communitydomain.CommunityRuleID) (*communitydomain.CommunityRule, error) {
	const query = `
		SELECT
			id::text,
			community_id::text,
			title,
			body,
			position,
			created_by::text,
			updated_by::text,
			created_at,
			updated_at
		FROM community_rules
		WHERE id = $1::uuid
		LIMIT 1
	`

	row := repo.db.QueryRow(ctx, query, id.String())
	rule, err := scanCommunityRule(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeNotFound, "community rule not found")
		}
		return nil, err
	}

	return rule, nil
}

func (repo *PostgresCommunityRepository) CreateRule(ctx context.Context, rule communitydomain.CommunityRule) error {
	const query = `
		INSERT INTO community_rules (
			id,
			community_id,
			title,
			body,
			position,
			created_by,
			updated_by,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::uuid, $7::uuid, $8, $9)
	`

	if _, err := repo.db.Exec(
		ctx,
		query,
		rule.ID().String(),
		rule.CommunityID().String(),
		rule.Title().String(),
		rule.Body().String(),
		rule.Position().Int(),
		rule.CreatedBy().String(),
		rule.UpdatedBy().String(),
		rule.CreatedAt(),
		rule.UpdatedAt(),
	); err != nil {
		return mapPostgresWriteError("create community rule", err)
	}

	return nil
}

func (repo *PostgresCommunityRepository) UpdateRule(ctx context.Context, rule communitydomain.CommunityRule) error {
	const query = `
		UPDATE community_rules
		SET title = $3,
			body = $4,
			position = $5,
			updated_by = $6::uuid,
			updated_at = $7
		WHERE id = $1::uuid
			AND community_id = $2::uuid
	`

	tag, err := repo.db.Exec(
		ctx,
		query,
		rule.ID().String(),
		rule.CommunityID().String(),
		rule.Title().String(),
		rule.Body().String(),
		rule.Position().Int(),
		rule.UpdatedBy().String(),
		rule.UpdatedAt(),
	)
	if err != nil {
		return mapPostgresWriteError("update community rule", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "community rule not found")
	}

	return nil
}

func (repo *PostgresCommunityRepository) DeleteRule(ctx context.Context, id communitydomain.CommunityRuleID, communityID communitydomain.CommunityID) error {
	const query = `
		DELETE FROM community_rules
		WHERE id = $1::uuid
			AND community_id = $2::uuid
	`

	tag, err := repo.db.Exec(ctx, query, id.String(), communityID.String())
	if err != nil {
		return mapPostgresWriteError("delete community rule", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.CodeNotFound, "community rule not found")
	}

	return nil
}

func (repo *PostgresCommunityRepository) LoadPublicStatsByCommunityIDs(ctx context.Context, communityIDs []communitydomain.CommunityID) (map[communitydomain.CommunityID]communityusecase.CommunityStats, error) {
	result := make(map[communitydomain.CommunityID]communityusecase.CommunityStats, len(communityIDs))
	if len(communityIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			communities.id::text,
			(
				SELECT COUNT(*)::int
				FROM community_memberships
				WHERE community_memberships.community_id = communities.id
					AND community_memberships.status = 'active'
			) AS member_count,
			(
				SELECT COUNT(*)::int
				FROM posts
				WHERE posts.community_id = communities.id
					AND posts.status = 'visible'
			) AS post_count
		FROM communities
		WHERE communities.id = ANY($1::uuid[])
	`

	rows, err := repo.db.Query(ctx, query, communityIDStrings(communityIDs))
	if err != nil {
		return nil, fmt.Errorf("load public community stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rawCommunityID string
		var memberCount int
		var postCount int
		if err := rows.Scan(&rawCommunityID, &memberCount, &postCount); err != nil {
			return nil, err
		}
		communityID, err := communitydomain.NewCommunityID(rawCommunityID)
		if err != nil {
			return nil, fmt.Errorf("rehydrate community stats id: %v", err)
		}
		result[communityID] = communityusecase.CommunityStats{
			MemberCount: memberCount,
			PostCount:   postCount,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate public community stats: %w", err)
	}

	return result, nil
}

func communityIDStrings(communityIDs []communitydomain.CommunityID) []string {
	rawIDs := make([]string, 0, len(communityIDs))
	for _, communityID := range communityIDs {
		rawIDs = append(rawIDs, communityID.String())
	}
	return rawIDs
}

func (repo *PostgresCommunityRepository) FollowCommunity(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID, now time.Time) error {
	const query = `
		INSERT INTO community_follows (
			community_id,
			user_id,
			created_at
		)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (community_id, user_id) DO NOTHING
	`

	if _, err := repo.db.Exec(ctx, query, communityID.String(), userID.String(), now); err != nil {
		return mapPostgresWriteError("follow community", err)
	}
	return nil
}

func (repo *PostgresCommunityRepository) DeleteCommunityFollow(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID) error {
	const query = `
		DELETE FROM community_follows
		WHERE community_id = $1::uuid
			AND user_id = $2::uuid
	`

	if _, err := repo.db.Exec(ctx, query, communityID.String(), userID.String()); err != nil {
		return mapPostgresWriteError("delete community follow", err)
	}
	return nil
}

func (repo *PostgresCommunityRepository) ListFollowedActivePublic(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]communitydomain.Community, error) {
	const query = `
		SELECT
			communities.id::text,
			communities.slug,
			communities.name,
			communities.description,
			communities.kind,
			communities.status,
			communities.visibility,
			communities.created_by::text,
			communities.created_at,
			communities.updated_at
		FROM community_follows
		INNER JOIN communities ON communities.id = community_follows.community_id
		WHERE community_follows.user_id = $1::uuid
			AND communities.status = 'active'
			AND communities.visibility = 'public'
		ORDER BY community_follows.created_at DESC, communities.slug ASC
		LIMIT $2
		OFFSET $3
	`

	rows, err := repo.db.Query(ctx, query, userID.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list followed active public communities: %w", err)
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
		return nil, fmt.Errorf("iterate followed active public communities: %w", err)
	}

	return communities, nil
}

func (repo *PostgresCommunityRepository) FindFollowedCommunityIDsByUser(ctx context.Context, communityIDs []communitydomain.CommunityID, userID userdomain.UserID) (map[communitydomain.CommunityID]bool, error) {
	result := make(map[communitydomain.CommunityID]bool, len(communityIDs))
	if len(communityIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT community_id::text
		FROM community_follows
		WHERE community_id = ANY($1::uuid[])
			AND user_id = $2::uuid
	`

	rows, err := repo.db.Query(ctx, query, communityIDStrings(communityIDs), userID.String())
	if err != nil {
		return nil, fmt.Errorf("find followed community ids by user: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rawCommunityID string
		if err := rows.Scan(&rawCommunityID); err != nil {
			return nil, err
		}
		communityID, err := communitydomain.NewCommunityID(rawCommunityID)
		if err != nil {
			return nil, fmt.Errorf("rehydrate followed community id: %v", err)
		}
		result[communityID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate followed community ids: %w", err)
	}

	return result, nil
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

func (repo *PostgresMembershipRepository) FindActiveRolesByUser(ctx context.Context, communityIDs []communitydomain.CommunityID, userID userdomain.UserID) (map[communitydomain.CommunityID]communitydomain.MembershipRole, error) {
	result := make(map[communitydomain.CommunityID]communitydomain.MembershipRole, len(communityIDs))
	if len(communityIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			community_id::text,
			role
		FROM community_memberships
		WHERE community_id = ANY($1::uuid[])
			AND user_id = $2::uuid
			AND status = 'active'
	`
	rows, err := repo.db.Query(ctx, query, communityIDStrings(communityIDs), userID.String())
	if err != nil {
		return nil, fmt.Errorf("find active community roles by user: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rawCommunityID string
		var rawRole string
		if err := rows.Scan(&rawCommunityID, &rawRole); err != nil {
			return nil, err
		}
		communityID, err := communitydomain.NewCommunityID(rawCommunityID)
		if err != nil {
			return nil, fmt.Errorf("rehydrate community membership community id: %v", err)
		}
		role, err := communitydomain.NewMembershipRole(rawRole)
		if err != nil {
			return nil, fmt.Errorf("rehydrate community membership role: %v", err)
		}
		result[communityID] = role
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active community roles by user: %w", err)
	}

	return result, nil
}

func (repo *PostgresMembershipRepository) ListActiveMembers(ctx context.Context, communityID communitydomain.CommunityID, limit int, offset int) ([]communityusecase.CommunityMember, error) {
	const query = `
		SELECT
			users.id::text,
			users.username,
			users.display_name,
			users.avatar_url,
			users.headline,
			community_memberships.role,
			community_memberships.status,
			community_memberships.created_at,
			community_memberships.updated_at
		FROM community_memberships
		INNER JOIN users ON users.id = community_memberships.user_id
		WHERE community_memberships.community_id = $1::uuid
			AND community_memberships.status = 'active'
		ORDER BY
			CASE community_memberships.role
				WHEN 'owner' THEN 1
				WHEN 'moderator' THEN 2
				ELSE 3
			END ASC,
			community_memberships.created_at ASC,
			users.username ASC
		LIMIT $2
		OFFSET $3
	`
	rows, err := repo.db.Query(ctx, query, communityID.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list active community members: %w", err)
	}
	defer rows.Close()

	members := make([]communityusecase.CommunityMember, 0)
	for rows.Next() {
		var member communityusecase.CommunityMember
		if err := rows.Scan(
			&member.UserID,
			&member.Username,
			&member.DisplayName,
			&member.AvatarURL,
			&member.Headline,
			&member.Role,
			&member.Status,
			&member.CreatedAt,
			&member.UpdatedAt,
		); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active community members: %w", err)
	}

	return members, nil
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

func scanCommunityRule(row rowScanner) (*communitydomain.CommunityRule, error) {
	var rawID string
	var rawCommunityID string
	var rawTitle string
	var rawBody string
	var rawPosition int
	var rawCreatedBy string
	var rawUpdatedBy string
	var createdAt pgtype.Timestamptz
	var updatedAt pgtype.Timestamptz

	if err := row.Scan(
		&rawID,
		&rawCommunityID,
		&rawTitle,
		&rawBody,
		&rawPosition,
		&rawCreatedBy,
		&rawUpdatedBy,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	id, err := communitydomain.NewCommunityRuleID(rawID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community rule id: %v", err)
	}
	communityID, err := communitydomain.NewCommunityID(rawCommunityID)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community rule community id: %v", err)
	}
	title, err := communitydomain.NewCommunityRuleTitle(rawTitle)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community rule title: %v", err)
	}
	position, err := communitydomain.NewCommunityRulePosition(rawPosition)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community rule position: %v", err)
	}
	createdBy, err := userdomain.NewUserID(rawCreatedBy)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community rule creator: %v", err)
	}
	updatedBy, err := userdomain.NewUserID(rawUpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community rule updater: %v", err)
	}
	if !createdAt.Valid || !updatedAt.Valid {
		return nil, fmt.Errorf("rehydrate community rule timestamps: missing timestamp")
	}

	rule, err := communitydomain.RehydrateCommunityRule(
		id,
		communityID,
		title,
		communitydomain.NewCommunityRuleBody(rawBody),
		position,
		createdBy,
		updatedBy,
		createdAt.Time,
		updatedAt.Time,
	)
	if err != nil {
		return nil, fmt.Errorf("rehydrate community rule: %v", err)
	}

	return rule, nil
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
			case "community_rules_pkey":
				return apperr.New(apperr.CodeConflict, "community rule already exists")
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

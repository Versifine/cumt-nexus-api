package communityusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationusecase"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

const (
	PublicCommunitySlug        = "public"
	publicCommunityName        = "Public"
	publicCommunityDescription = "System public community"
	defaultCommunityListLimit  = 20
	maxCommunityListLimit      = 50
)

type PublicCommunityBootstrapUseCase struct {
	communities CommunityRepository
	now         func() time.Time
}

type CommunityReadUseCase struct {
	communities    CommunityRepository
	settings       CommunitySettingsRepository
	stats          CommunityStatsRepository
	follows        CommunityFollowRepository
	memberships    CommunityMembershipReadRepository
	managePosts    CommunityManagePostRepository
	manageComments CommunityManageCommentRepository
	manageReports  CommunityManageReportRepository
	rules          CommunityRuleRepository
	now            func() time.Time
}

type ListCommunitiesInput struct {
	ViewerID userdomain.UserID
}

type GetCommunityInput struct {
	Slug     string
	ViewerID userdomain.UserID
}

type CanPostInCommunityInput struct {
	UserID      string
	CommunityID string
}

type FollowCommunityInput struct {
	Slug   string
	UserID userdomain.UserID
}

type DeleteCommunityFollowInput struct {
	Slug   string
	UserID userdomain.UserID
}

type ListFollowedCommunitiesInput struct {
	UserID userdomain.UserID
	Limit  int
	Offset int
}

type GetCommunityManageContextInput struct {
	Slug     string
	ViewerID userdomain.UserID
}

type ListCommunityMembersInput struct {
	Slug     string
	ViewerID userdomain.UserID
	Limit    int
	Offset   int
}

type ListCommunityManagePostsInput struct {
	Slug     string
	ViewerID userdomain.UserID
	Status   string
	Limit    int
	Offset   int
}

type ListCommunityManageCommentsInput struct {
	Slug     string
	ViewerID userdomain.UserID
	Status   string
	Limit    int
	Offset   int
}

type ListCommunityManageReportsInput struct {
	Slug     string
	ViewerID userdomain.UserID
	Status   string
	Limit    int
	Offset   int
}

type GetCommunityManageSettingsInput struct {
	Slug     string
	ViewerID userdomain.UserID
}

type UpdateCommunityManageSettingsInput struct {
	Slug        string
	ViewerID    userdomain.UserID
	Name        string
	Description string
}

type ListCommunityRulesInput struct {
	Slug     string
	ViewerID userdomain.UserID
}

type CreateCommunityRuleInput struct {
	Slug     string
	ViewerID userdomain.UserID
	Title    string
	Body     string
	Position int
}

type UpdateCommunityRuleInput struct {
	Slug     string
	RuleID   string
	ViewerID userdomain.UserID
	Title    string
	Body     string
	Position int
}

type DeleteCommunityRuleInput struct {
	Slug     string
	RuleID   string
	ViewerID userdomain.UserID
}

type ListCommunitiesResult struct {
	Communities []Community
}

type ListFollowedCommunitiesResult struct {
	Communities []Community
	Limit       int
	Offset      int
}

type GetCommunityResult struct {
	Community Community
}

type FollowCommunityResult struct{}

type DeleteCommunityFollowResult struct{}

type CanPostInCommunityResult struct {
	Community Community
}

type GetCommunityManageContextResult struct {
	Community Community
}

type ListCommunityMembersResult struct {
	Community Community
	Members   []CommunityMember
	Limit     int
	Offset    int
}

type ListCommunityManagePostsResult struct {
	Community Community
	Posts     []CommunityManagePost
	Status    string
	Limit     int
	Offset    int
}

type ListCommunityManageCommentsResult struct {
	Community Community
	Comments  []CommunityManageComment
	Status    string
	Limit     int
	Offset    int
}

type ListCommunityManageReportsResult struct {
	Community Community
	Reports   []CommunityManageReport
	Status    string
	Limit     int
	Offset    int
}

type GetCommunityManageSettingsResult struct {
	Community Community
	Settings  CommunitySettings
}

type UpdateCommunityManageSettingsResult struct {
	Community Community
	Settings  CommunitySettings
}

type ListCommunityRulesResult struct {
	Community Community
	Rules     []CommunityRule
}

type CreateCommunityRuleResult struct {
	Community Community
	Rule      CommunityRule
}

type UpdateCommunityRuleResult struct {
	Community Community
	Rule      CommunityRule
}

type DeleteCommunityRuleResult struct{}

type Community struct {
	ID                string
	Slug              string
	Name              string
	Description       string
	AvatarURL         string
	BannerURL         string
	Kind              string
	Status            string
	Visibility        string
	MemberCount       int
	PostCount         int
	ViewerIsFollowing bool
	ViewerRole        string
	ViewerPermissions ViewerPermissions
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CommunitySettings struct {
	Name        string
	Description string
	AvatarURL   string
	BannerURL   string
	UpdatedAt   time.Time
}

type CommunityRule struct {
	ID          string
	CommunityID string
	Title       string
	Body        string
	Position    int
	CreatedBy   string
	UpdatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CommunityMember struct {
	UserID      string
	Username    string
	DisplayName string
	AvatarURL   string
	Headline    string
	Role        string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CommunityManagePost struct {
	ID          string
	CommunityID string
	AuthorID    string
	Title       string
	BodyExcerpt string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CommunityManageComment struct {
	ID          string
	PostID      string
	AuthorID    string
	ParentID    string
	BodyExcerpt string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CommunityManageReport struct {
	ID            string
	TargetType    string
	PostID        string
	CommentID     string
	ReporterID    string
	Reason        string
	Status        string
	ReviewedBy    string
	ReviewedAt    *time.Time
	TargetPreview *CommunityManageReportTargetPreview
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CommunityManageReportTargetPreview struct {
	TargetType  string
	PostID      string
	CommentID   string
	AuthorID    string
	Status      string
	Title       string
	BodyExcerpt string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CommunityStats struct {
	MemberCount int
	PostCount   int
}

type ViewerPermissions struct {
	CanPost     bool
	CanManage   bool
	CanModerate bool
}

func NewPublicCommunityBootstrapUseCase(communities CommunityRepository, now func() time.Time) *PublicCommunityBootstrapUseCase {
	if now == nil {
		now = time.Now
	}

	return &PublicCommunityBootstrapUseCase{
		communities: communities,
		now:         now,
	}
}

func NewCommunityReadUseCase(communities CommunityRepository) *CommunityReadUseCase {
	uc := &CommunityReadUseCase{
		communities: communities,
		now:         time.Now,
	}
	if repo, ok := communities.(CommunityStatsRepository); ok {
		uc.stats = repo
	}
	if repo, ok := communities.(CommunityFollowRepository); ok {
		uc.follows = repo
	}
	if repo, ok := communities.(CommunitySettingsRepository); ok {
		uc.settings = repo
	}
	if repo, ok := communities.(CommunityRuleRepository); ok {
		uc.rules = repo
	}
	return uc
}

func (uc *CommunityReadUseCase) SetMembershipReader(memberships CommunityMembershipReadRepository) {
	uc.memberships = memberships
}

func (uc *CommunityReadUseCase) SetManageContentReaders(posts CommunityManagePostRepository, comments CommunityManageCommentRepository, reports CommunityManageReportRepository) {
	uc.managePosts = posts
	uc.manageComments = comments
	uc.manageReports = reports
}

func (uc *CommunityReadUseCase) SetRuleRepository(rules CommunityRuleRepository) {
	uc.rules = rules
}

func (uc *CommunityReadUseCase) SetSettingsRepository(settings CommunitySettingsRepository) {
	uc.settings = settings
}

func (uc *PublicCommunityBootstrapUseCase) EnsurePublicCommunity(ctx context.Context) error {
	slug, err := communitydomain.NewCommunitySlug(PublicCommunitySlug)
	if err != nil {
		return err
	}

	existing, err := uc.communities.FindBySlug(ctx, slug)
	if err == nil {
		return validatePublicCommunity(existing)
	}
	if !apperr.IsCode(err, apperr.CodeNotFound) {
		return fmt.Errorf("find public community: %w", err)
	}

	now := uc.now().UTC()
	name, err := communitydomain.NewCommunityName(publicCommunityName)
	if err != nil {
		return err
	}
	community, err := communitydomain.NewSystemCommunity(
		communitydomain.NewGeneratedCommunityID(),
		slug,
		name,
		communitydomain.NewCommunityDescription(publicCommunityDescription),
		now,
	)
	if err != nil {
		return err
	}

	if err := uc.communities.Create(ctx, *community); err != nil {
		if apperr.IsCode(err, apperr.CodeConflict) {
			return uc.validatePublicCommunityAfterConflict(ctx, slug)
		}
		return fmt.Errorf("create public community: %w", err)
	}

	return nil
}

func (uc *PublicCommunityBootstrapUseCase) validatePublicCommunityAfterConflict(ctx context.Context, slug communitydomain.CommunitySlug) error {
	existing, err := uc.communities.FindBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("find public community after conflict: %w", err)
	}

	return validatePublicCommunity(existing)
}

func (uc *CommunityReadUseCase) ListCommunities(ctx context.Context, input ListCommunitiesInput) (ListCommunitiesResult, error) {
	communities, err := uc.communities.ListActivePublic(ctx)
	if err != nil {
		return ListCommunitiesResult{}, fmt.Errorf("list active public communities: %w", err)
	}

	result := ListCommunitiesResult{
		Communities: make([]Community, 0, len(communities)),
	}
	stats, err := uc.loadCommunityStats(ctx, communities)
	if err != nil {
		return ListCommunitiesResult{}, err
	}
	followViews, err := uc.loadCommunityFollowViews(ctx, communities, input.ViewerID)
	if err != nil {
		return ListCommunitiesResult{}, err
	}
	roleViews, err := uc.loadCommunityRoleViews(ctx, communities, input.ViewerID)
	if err != nil {
		return ListCommunitiesResult{}, err
	}
	for _, community := range communities {
		result.Communities = append(result.Communities, toCommunityDTO(community, stats[community.ID()], followViews[community.ID()], roleViews[community.ID()], input.ViewerID))
	}

	return result, nil
}

func (uc *CommunityReadUseCase) GetCommunityBySlug(ctx context.Context, input GetCommunityInput) (GetCommunityResult, error) {
	slug, err := communitydomain.NewCommunitySlug(input.Slug)
	if err != nil {
		return GetCommunityResult{}, err
	}

	community, err := uc.communities.FindBySlug(ctx, slug)
	if err != nil {
		return GetCommunityResult{}, fmt.Errorf("find community by slug: %w", err)
	}
	if !isPubliclyReadableCommunity(community) {
		return GetCommunityResult{}, apperr.New(apperr.CodeNotFound, "community not found")
	}
	stats, err := uc.loadCommunityStats(ctx, []communitydomain.Community{*community})
	if err != nil {
		return GetCommunityResult{}, err
	}
	followViews, err := uc.loadCommunityFollowViews(ctx, []communitydomain.Community{*community}, input.ViewerID)
	if err != nil {
		return GetCommunityResult{}, err
	}
	roleViews, err := uc.loadCommunityRoleViews(ctx, []communitydomain.Community{*community}, input.ViewerID)
	if err != nil {
		return GetCommunityResult{}, err
	}

	return GetCommunityResult{
		Community: toCommunityDTO(*community, stats[community.ID()], followViews[community.ID()], roleViews[community.ID()], input.ViewerID),
	}, nil
}

func (uc *CommunityReadUseCase) FollowCommunity(ctx context.Context, input FollowCommunityInput) (FollowCommunityResult, error) {
	if isBlankUserID(input.UserID) {
		return FollowCommunityResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.follows == nil {
		return FollowCommunityResult{}, apperr.New(apperr.CodeInternal, "community follows are not configured")
	}

	community, err := uc.findReadableCommunityBySlug(ctx, input.Slug)
	if err != nil {
		return FollowCommunityResult{}, err
	}
	if err := uc.follows.FollowCommunity(ctx, community.ID(), input.UserID, uc.now().UTC()); err != nil {
		return FollowCommunityResult{}, fmt.Errorf("follow community: %w", err)
	}

	return FollowCommunityResult{}, nil
}

func (uc *CommunityReadUseCase) DeleteCommunityFollow(ctx context.Context, input DeleteCommunityFollowInput) (DeleteCommunityFollowResult, error) {
	if isBlankUserID(input.UserID) {
		return DeleteCommunityFollowResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.follows == nil {
		return DeleteCommunityFollowResult{}, apperr.New(apperr.CodeInternal, "community follows are not configured")
	}

	community, err := uc.findReadableCommunityBySlug(ctx, input.Slug)
	if err != nil {
		return DeleteCommunityFollowResult{}, err
	}
	if err := uc.follows.DeleteCommunityFollow(ctx, community.ID(), input.UserID); err != nil {
		return DeleteCommunityFollowResult{}, fmt.Errorf("delete community follow: %w", err)
	}

	return DeleteCommunityFollowResult{}, nil
}

func (uc *CommunityReadUseCase) ListFollowedCommunities(ctx context.Context, input ListFollowedCommunitiesInput) (ListFollowedCommunitiesResult, error) {
	if isBlankUserID(input.UserID) {
		return ListFollowedCommunitiesResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.follows == nil {
		return ListFollowedCommunitiesResult{}, apperr.New(apperr.CodeInternal, "community follows are not configured")
	}
	limit, offset, err := normalizeCommunityListPagination(input.Limit, input.Offset)
	if err != nil {
		return ListFollowedCommunitiesResult{}, err
	}

	communities, err := uc.follows.ListFollowedActivePublic(ctx, input.UserID, limit, offset)
	if err != nil {
		return ListFollowedCommunitiesResult{}, fmt.Errorf("list followed communities: %w", err)
	}
	stats, err := uc.loadCommunityStats(ctx, communities)
	if err != nil {
		return ListFollowedCommunitiesResult{}, err
	}
	followViews, err := uc.loadCommunityFollowViews(ctx, communities, input.UserID)
	if err != nil {
		return ListFollowedCommunitiesResult{}, err
	}
	roleViews, err := uc.loadCommunityRoleViews(ctx, communities, input.UserID)
	if err != nil {
		return ListFollowedCommunitiesResult{}, err
	}

	result := ListFollowedCommunitiesResult{
		Communities: make([]Community, 0, len(communities)),
		Limit:       limit,
		Offset:      offset,
	}
	for _, community := range communities {
		result.Communities = append(result.Communities, toCommunityDTO(community, stats[community.ID()], followViews[community.ID()], roleViews[community.ID()], input.UserID))
	}

	return result, nil
}

func (uc *CommunityReadUseCase) GetCommunityManageContext(ctx context.Context, input GetCommunityManageContextInput) (GetCommunityManageContextResult, error) {
	if isBlankUserID(input.ViewerID) {
		return GetCommunityManageContextResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	community, roleView, err := uc.findManageableCommunityBySlug(ctx, input.Slug, input.ViewerID)
	if err != nil {
		return GetCommunityManageContextResult{}, err
	}
	stats, err := uc.loadCommunityStats(ctx, []communitydomain.Community{*community})
	if err != nil {
		return GetCommunityManageContextResult{}, err
	}
	followViews, err := uc.loadCommunityFollowViews(ctx, []communitydomain.Community{*community}, input.ViewerID)
	if err != nil {
		return GetCommunityManageContextResult{}, err
	}

	return GetCommunityManageContextResult{
		Community: toCommunityDTO(*community, stats[community.ID()], followViews[community.ID()], roleView, input.ViewerID),
	}, nil
}

func (uc *CommunityReadUseCase) ListCommunityMembers(ctx context.Context, input ListCommunityMembersInput) (ListCommunityMembersResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ListCommunityMembersResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.memberships == nil {
		return ListCommunityMembersResult{}, apperr.New(apperr.CodeInternal, "community memberships are not configured")
	}
	limit, offset, err := normalizeCommunityListPagination(input.Limit, input.Offset)
	if err != nil {
		return ListCommunityMembersResult{}, err
	}
	community, roleView, err := uc.findManageableCommunityBySlug(ctx, input.Slug, input.ViewerID)
	if err != nil {
		return ListCommunityMembersResult{}, err
	}
	stats, err := uc.loadCommunityStats(ctx, []communitydomain.Community{*community})
	if err != nil {
		return ListCommunityMembersResult{}, err
	}
	followViews, err := uc.loadCommunityFollowViews(ctx, []communitydomain.Community{*community}, input.ViewerID)
	if err != nil {
		return ListCommunityMembersResult{}, err
	}
	members, err := uc.memberships.ListActiveMembers(ctx, community.ID(), limit, offset)
	if err != nil {
		return ListCommunityMembersResult{}, fmt.Errorf("list community members: %w", err)
	}

	return ListCommunityMembersResult{
		Community: toCommunityDTO(*community, stats[community.ID()], followViews[community.ID()], roleView, input.ViewerID),
		Members:   members,
		Limit:     limit,
		Offset:    offset,
	}, nil
}

func (uc *CommunityReadUseCase) ListCommunityManagePosts(ctx context.Context, input ListCommunityManagePostsInput) (ListCommunityManagePostsResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ListCommunityManagePostsResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.managePosts == nil {
		return ListCommunityManagePostsResult{}, apperr.New(apperr.CodeInternal, "community manage posts are not configured")
	}
	status, statusLabel, err := parseManagePostStatus(input.Status)
	if err != nil {
		return ListCommunityManagePostsResult{}, err
	}
	limit, offset, err := normalizeCommunityListPagination(input.Limit, input.Offset)
	if err != nil {
		return ListCommunityManagePostsResult{}, err
	}
	community, roleView, err := uc.findManageableCommunityBySlug(ctx, input.Slug, input.ViewerID)
	if err != nil {
		return ListCommunityManagePostsResult{}, err
	}
	communityDTO, err := uc.buildCommunityDTOForViewer(ctx, *community, roleView, input.ViewerID)
	if err != nil {
		return ListCommunityManagePostsResult{}, err
	}
	posts, err := uc.managePosts.ListPostsByCommunityForManagement(ctx, community.ID(), status, limit, offset)
	if err != nil {
		return ListCommunityManagePostsResult{}, fmt.Errorf("list community manage posts: %w", err)
	}

	result := ListCommunityManagePostsResult{
		Community: communityDTO,
		Posts:     make([]CommunityManagePost, 0, len(posts)),
		Status:    statusLabel,
		Limit:     limit,
		Offset:    offset,
	}
	for _, post := range posts {
		result.Posts = append(result.Posts, toCommunityManagePostDTO(post))
	}
	return result, nil
}

func (uc *CommunityReadUseCase) ListCommunityManageComments(ctx context.Context, input ListCommunityManageCommentsInput) (ListCommunityManageCommentsResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ListCommunityManageCommentsResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.manageComments == nil {
		return ListCommunityManageCommentsResult{}, apperr.New(apperr.CodeInternal, "community manage comments are not configured")
	}
	status, statusLabel, err := parseManageCommentStatus(input.Status)
	if err != nil {
		return ListCommunityManageCommentsResult{}, err
	}
	limit, offset, err := normalizeCommunityListPagination(input.Limit, input.Offset)
	if err != nil {
		return ListCommunityManageCommentsResult{}, err
	}
	community, roleView, err := uc.findManageableCommunityBySlug(ctx, input.Slug, input.ViewerID)
	if err != nil {
		return ListCommunityManageCommentsResult{}, err
	}
	communityDTO, err := uc.buildCommunityDTOForViewer(ctx, *community, roleView, input.ViewerID)
	if err != nil {
		return ListCommunityManageCommentsResult{}, err
	}
	comments, err := uc.manageComments.ListCommentsByCommunityForManagement(ctx, community.ID(), status, limit, offset)
	if err != nil {
		return ListCommunityManageCommentsResult{}, fmt.Errorf("list community manage comments: %w", err)
	}

	result := ListCommunityManageCommentsResult{
		Community: communityDTO,
		Comments:  make([]CommunityManageComment, 0, len(comments)),
		Status:    statusLabel,
		Limit:     limit,
		Offset:    offset,
	}
	for _, comment := range comments {
		result.Comments = append(result.Comments, toCommunityManageCommentDTO(comment))
	}
	return result, nil
}

func (uc *CommunityReadUseCase) ListCommunityManageReports(ctx context.Context, input ListCommunityManageReportsInput) (ListCommunityManageReportsResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ListCommunityManageReportsResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.manageReports == nil {
		return ListCommunityManageReportsResult{}, apperr.New(apperr.CodeInternal, "community manage reports are not configured")
	}
	status, err := parseManageReportStatus(input.Status)
	if err != nil {
		return ListCommunityManageReportsResult{}, err
	}
	limit, offset, err := normalizeCommunityListPagination(input.Limit, input.Offset)
	if err != nil {
		return ListCommunityManageReportsResult{}, err
	}
	community, roleView, err := uc.findManageableCommunityBySlug(ctx, input.Slug, input.ViewerID)
	if err != nil {
		return ListCommunityManageReportsResult{}, err
	}
	communityDTO, err := uc.buildCommunityDTOForViewer(ctx, *community, roleView, input.ViewerID)
	if err != nil {
		return ListCommunityManageReportsResult{}, err
	}
	records, err := uc.manageReports.ListReportsByCommunityForManagement(ctx, community.ID(), status, limit, offset)
	if err != nil {
		return ListCommunityManageReportsResult{}, fmt.Errorf("list community manage reports: %w", err)
	}

	result := ListCommunityManageReportsResult{
		Community: communityDTO,
		Reports:   make([]CommunityManageReport, 0, len(records)),
		Status:    status.String(),
		Limit:     limit,
		Offset:    offset,
	}
	for _, record := range records {
		result.Reports = append(result.Reports, toCommunityManageReportDTO(record))
	}
	return result, nil
}

func (uc *CommunityReadUseCase) GetCommunityManageSettings(ctx context.Context, input GetCommunityManageSettingsInput) (GetCommunityManageSettingsResult, error) {
	if isBlankUserID(input.ViewerID) {
		return GetCommunityManageSettingsResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	community, roleView, err := uc.findManageableCommunityBySlug(ctx, input.Slug, input.ViewerID)
	if err != nil {
		return GetCommunityManageSettingsResult{}, err
	}
	communityDTO, err := uc.buildCommunityDTOForViewer(ctx, *community, roleView, input.ViewerID)
	if err != nil {
		return GetCommunityManageSettingsResult{}, err
	}
	return GetCommunityManageSettingsResult{
		Community: communityDTO,
		Settings:  toCommunitySettingsDTO(*community),
	}, nil
}

func (uc *CommunityReadUseCase) UpdateCommunityManageSettings(ctx context.Context, input UpdateCommunityManageSettingsInput) (UpdateCommunityManageSettingsResult, error) {
	if isBlankUserID(input.ViewerID) {
		return UpdateCommunityManageSettingsResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	community, roleView, err := uc.findManageableCommunityBySlug(ctx, input.Slug, input.ViewerID)
	if err != nil {
		return UpdateCommunityManageSettingsResult{}, err
	}
	if roleView.role != communitydomain.MembershipRoleOwner {
		return UpdateCommunityManageSettingsResult{}, apperr.New(apperr.CodeForbidden, "community owner required")
	}
	if uc.settings == nil {
		return UpdateCommunityManageSettingsResult{}, apperr.New(apperr.CodeInternal, "community settings are not configured")
	}
	name, err := communitydomain.NewCommunityName(input.Name)
	if err != nil {
		return UpdateCommunityManageSettingsResult{}, err
	}
	description := communitydomain.NewCommunityDescription(input.Description)
	if err := community.UpdateDetails(name, description, uc.now().UTC()); err != nil {
		return UpdateCommunityManageSettingsResult{}, err
	}
	if err := uc.settings.UpdateDetails(ctx, *community); err != nil {
		return UpdateCommunityManageSettingsResult{}, fmt.Errorf("update community settings: %w", err)
	}
	communityDTO, err := uc.buildCommunityDTOForViewer(ctx, *community, roleView, input.ViewerID)
	if err != nil {
		return UpdateCommunityManageSettingsResult{}, err
	}
	return UpdateCommunityManageSettingsResult{
		Community: communityDTO,
		Settings:  toCommunitySettingsDTO(*community),
	}, nil
}

func (uc *CommunityReadUseCase) ListCommunityRules(ctx context.Context, input ListCommunityRulesInput) (ListCommunityRulesResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ListCommunityRulesResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.rules == nil {
		return ListCommunityRulesResult{}, apperr.New(apperr.CodeInternal, "community rules are not configured")
	}
	community, roleView, err := uc.findManageableCommunityBySlug(ctx, input.Slug, input.ViewerID)
	if err != nil {
		return ListCommunityRulesResult{}, err
	}
	communityDTO, err := uc.buildCommunityDTOForViewer(ctx, *community, roleView, input.ViewerID)
	if err != nil {
		return ListCommunityRulesResult{}, err
	}
	rules, err := uc.rules.ListRules(ctx, community.ID())
	if err != nil {
		return ListCommunityRulesResult{}, fmt.Errorf("list community rules: %w", err)
	}
	result := ListCommunityRulesResult{
		Community: communityDTO,
		Rules:     make([]CommunityRule, 0, len(rules)),
	}
	for _, rule := range rules {
		result.Rules = append(result.Rules, toCommunityRuleDTO(rule))
	}
	return result, nil
}

func (uc *CommunityReadUseCase) CreateCommunityRule(ctx context.Context, input CreateCommunityRuleInput) (CreateCommunityRuleResult, error) {
	if isBlankUserID(input.ViewerID) {
		return CreateCommunityRuleResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.rules == nil {
		return CreateCommunityRuleResult{}, apperr.New(apperr.CodeInternal, "community rules are not configured")
	}
	community, roleView, err := uc.findManageableCommunityBySlug(ctx, input.Slug, input.ViewerID)
	if err != nil {
		return CreateCommunityRuleResult{}, err
	}
	title, err := communitydomain.NewCommunityRuleTitle(input.Title)
	if err != nil {
		return CreateCommunityRuleResult{}, err
	}
	position, err := communitydomain.NewCommunityRulePosition(input.Position)
	if err != nil {
		return CreateCommunityRuleResult{}, err
	}
	now := uc.now().UTC()
	rule, err := communitydomain.NewCommunityRule(communitydomain.NewGeneratedCommunityRuleID(), community.ID(), title, communitydomain.NewCommunityRuleBody(input.Body), position, input.ViewerID, now)
	if err != nil {
		return CreateCommunityRuleResult{}, err
	}
	if err := uc.rules.CreateRule(ctx, *rule); err != nil {
		return CreateCommunityRuleResult{}, fmt.Errorf("create community rule: %w", err)
	}
	communityDTO, err := uc.buildCommunityDTOForViewer(ctx, *community, roleView, input.ViewerID)
	if err != nil {
		return CreateCommunityRuleResult{}, err
	}
	return CreateCommunityRuleResult{
		Community: communityDTO,
		Rule:      toCommunityRuleDTO(*rule),
	}, nil
}

func (uc *CommunityReadUseCase) UpdateCommunityRule(ctx context.Context, input UpdateCommunityRuleInput) (UpdateCommunityRuleResult, error) {
	if isBlankUserID(input.ViewerID) {
		return UpdateCommunityRuleResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.rules == nil {
		return UpdateCommunityRuleResult{}, apperr.New(apperr.CodeInternal, "community rules are not configured")
	}
	community, roleView, err := uc.findManageableCommunityBySlug(ctx, input.Slug, input.ViewerID)
	if err != nil {
		return UpdateCommunityRuleResult{}, err
	}
	ruleID, err := communitydomain.NewCommunityRuleID(input.RuleID)
	if err != nil {
		return UpdateCommunityRuleResult{}, err
	}
	rule, err := uc.rules.FindRuleByID(ctx, ruleID)
	if err != nil {
		return UpdateCommunityRuleResult{}, fmt.Errorf("find community rule: %w", err)
	}
	if rule.CommunityID() != community.ID() {
		return UpdateCommunityRuleResult{}, apperr.New(apperr.CodeNotFound, "community rule not found")
	}
	title, err := communitydomain.NewCommunityRuleTitle(input.Title)
	if err != nil {
		return UpdateCommunityRuleResult{}, err
	}
	position, err := communitydomain.NewCommunityRulePosition(input.Position)
	if err != nil {
		return UpdateCommunityRuleResult{}, err
	}
	if err := rule.Update(title, communitydomain.NewCommunityRuleBody(input.Body), position, input.ViewerID, uc.now().UTC()); err != nil {
		return UpdateCommunityRuleResult{}, err
	}
	if err := uc.rules.UpdateRule(ctx, *rule); err != nil {
		return UpdateCommunityRuleResult{}, fmt.Errorf("update community rule: %w", err)
	}
	communityDTO, err := uc.buildCommunityDTOForViewer(ctx, *community, roleView, input.ViewerID)
	if err != nil {
		return UpdateCommunityRuleResult{}, err
	}
	return UpdateCommunityRuleResult{
		Community: communityDTO,
		Rule:      toCommunityRuleDTO(*rule),
	}, nil
}

func (uc *CommunityReadUseCase) DeleteCommunityRule(ctx context.Context, input DeleteCommunityRuleInput) (DeleteCommunityRuleResult, error) {
	if isBlankUserID(input.ViewerID) {
		return DeleteCommunityRuleResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.rules == nil {
		return DeleteCommunityRuleResult{}, apperr.New(apperr.CodeInternal, "community rules are not configured")
	}
	community, _, err := uc.findManageableCommunityBySlug(ctx, input.Slug, input.ViewerID)
	if err != nil {
		return DeleteCommunityRuleResult{}, err
	}
	ruleID, err := communitydomain.NewCommunityRuleID(input.RuleID)
	if err != nil {
		return DeleteCommunityRuleResult{}, err
	}
	if err := uc.rules.DeleteRule(ctx, ruleID, community.ID()); err != nil {
		return DeleteCommunityRuleResult{}, fmt.Errorf("delete community rule: %w", err)
	}
	return DeleteCommunityRuleResult{}, nil
}

func (uc *CommunityReadUseCase) CanPostInCommunity(ctx context.Context, input CanPostInCommunityInput) (CanPostInCommunityResult, error) {
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return CanPostInCommunityResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if _, err := userdomain.NewUserID(userID); err != nil {
		return CanPostInCommunityResult{}, err
	}

	communityID, err := communitydomain.NewCommunityID(input.CommunityID)
	if err != nil {
		return CanPostInCommunityResult{}, err
	}

	community, err := uc.communities.FindByID(ctx, communityID)
	if err != nil {
		return CanPostInCommunityResult{}, fmt.Errorf("find community by id: %w", err)
	}
	if !isPostableCommunity(community) {
		return CanPostInCommunityResult{}, apperr.New(apperr.CodeForbidden, "can't post in community")
	}

	return CanPostInCommunityResult{
		Community: toCommunityDTO(*community, CommunityStats{}, communityFollowView{}, communityRoleView{}, ""),
	}, nil
}

func validatePublicCommunity(community *communitydomain.Community) error {
	if community == nil {
		return apperr.New(apperr.CodeInternal, "public community is missing")
	}
	if community.Slug().String() != PublicCommunitySlug {
		return apperr.New(apperr.CodeInternal, "public community is misconfigured")
	}
	if community.Kind() != communitydomain.CommunityKindSystem {
		return apperr.New(apperr.CodeInternal, "public community is misconfigured")
	}
	if community.Status() != communitydomain.CommunityStatusActive {
		return apperr.New(apperr.CodeInternal, "public community is misconfigured")
	}
	if community.Visibility() != communitydomain.CommunityVisibilityPublic {
		return apperr.New(apperr.CodeInternal, "public community is misconfigured")
	}
	if _, ok := community.CreatedBy(); ok {
		return apperr.New(apperr.CodeInternal, "public community is misconfigured")
	}

	return nil
}

func isPubliclyReadableCommunity(community *communitydomain.Community) bool {
	return community != nil &&
		community.Status() == communitydomain.CommunityStatusActive &&
		community.Visibility() == communitydomain.CommunityVisibilityPublic
}

func isPostableCommunity(community *communitydomain.Community) bool {
	return isPubliclyReadableCommunity(community)
}

func (uc *CommunityReadUseCase) loadCommunityStats(ctx context.Context, communities []communitydomain.Community) (map[communitydomain.CommunityID]CommunityStats, error) {
	stats := make(map[communitydomain.CommunityID]CommunityStats, len(communities))
	if len(communities) == 0 || uc.stats == nil {
		return stats, nil
	}
	communityIDs := make([]communitydomain.CommunityID, 0, len(communities))
	for _, community := range communities {
		communityIDs = append(communityIDs, community.ID())
	}
	loaded, err := uc.stats.LoadPublicStatsByCommunityIDs(ctx, communityIDs)
	if err != nil {
		return nil, fmt.Errorf("load community stats: %w", err)
	}
	return loaded, nil
}

type communityFollowView struct {
	isFollowing bool
}

type communityRoleView struct {
	role communitydomain.MembershipRole
}

func (uc *CommunityReadUseCase) loadCommunityFollowViews(ctx context.Context, communities []communitydomain.Community, viewerID userdomain.UserID) (map[communitydomain.CommunityID]communityFollowView, error) {
	views := make(map[communitydomain.CommunityID]communityFollowView, len(communities))
	if len(communities) == 0 || uc.follows == nil || isBlankUserID(viewerID) {
		return views, nil
	}

	communityIDs := make([]communitydomain.CommunityID, 0, len(communities))
	for _, community := range communities {
		communityIDs = append(communityIDs, community.ID())
	}
	followed, err := uc.follows.FindFollowedCommunityIDsByUser(ctx, communityIDs, viewerID)
	if err != nil {
		return nil, fmt.Errorf("find followed communities by viewer: %w", err)
	}
	for _, communityID := range communityIDs {
		views[communityID] = communityFollowView{isFollowing: followed[communityID]}
	}
	return views, nil
}

func (uc *CommunityReadUseCase) loadCommunityRoleViews(ctx context.Context, communities []communitydomain.Community, viewerID userdomain.UserID) (map[communitydomain.CommunityID]communityRoleView, error) {
	views := make(map[communitydomain.CommunityID]communityRoleView, len(communities))
	if len(communities) == 0 || uc.memberships == nil || isBlankUserID(viewerID) {
		return views, nil
	}

	communityIDs := make([]communitydomain.CommunityID, 0, len(communities))
	for _, community := range communities {
		communityIDs = append(communityIDs, community.ID())
	}
	roles, err := uc.memberships.FindActiveRolesByUser(ctx, communityIDs, viewerID)
	if err != nil {
		return nil, fmt.Errorf("find community roles by viewer: %w", err)
	}
	for communityID, role := range roles {
		views[communityID] = communityRoleView{role: role}
	}
	return views, nil
}

func (uc *CommunityReadUseCase) buildCommunityDTOForViewer(ctx context.Context, community communitydomain.Community, roleView communityRoleView, viewerID userdomain.UserID) (Community, error) {
	stats, err := uc.loadCommunityStats(ctx, []communitydomain.Community{community})
	if err != nil {
		return Community{}, err
	}
	followViews, err := uc.loadCommunityFollowViews(ctx, []communitydomain.Community{community}, viewerID)
	if err != nil {
		return Community{}, err
	}
	return toCommunityDTO(community, stats[community.ID()], followViews[community.ID()], roleView, viewerID), nil
}

func (uc *CommunityReadUseCase) findReadableCommunityBySlug(ctx context.Context, rawSlug string) (*communitydomain.Community, error) {
	slug, err := communitydomain.NewCommunitySlug(rawSlug)
	if err != nil {
		return nil, err
	}
	community, err := uc.communities.FindBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("find community by slug: %w", err)
	}
	if !isPubliclyReadableCommunity(community) {
		return nil, apperr.New(apperr.CodeNotFound, "community not found")
	}
	return community, nil
}

func (uc *CommunityReadUseCase) findManageableCommunityBySlug(ctx context.Context, rawSlug string, viewerID userdomain.UserID) (*communitydomain.Community, communityRoleView, error) {
	if uc.memberships == nil {
		return nil, communityRoleView{}, apperr.New(apperr.CodeInternal, "community memberships are not configured")
	}
	community, err := uc.findReadableCommunityBySlug(ctx, rawSlug)
	if err != nil {
		return nil, communityRoleView{}, err
	}
	roleViews, err := uc.loadCommunityRoleViews(ctx, []communitydomain.Community{*community}, viewerID)
	if err != nil {
		return nil, communityRoleView{}, err
	}
	roleView := roleViews[community.ID()]
	if !canModerateCommunity(roleView.role) {
		return nil, communityRoleView{}, apperr.New(apperr.CodeForbidden, "community moderator required")
	}
	return community, roleView, nil
}

func normalizeCommunityListPagination(limit int, offset int) (int, int, error) {
	if limit < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "limit must be non-negative")
	}
	if offset < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "offset must be non-negative")
	}
	if limit == 0 {
		limit = defaultCommunityListLimit
	}
	if limit > maxCommunityListLimit {
		limit = maxCommunityListLimit
	}
	return limit, offset, nil
}

func parseManagePostStatus(raw string) (*postdomain.PostStatus, string, error) {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	if normalized == "" || normalized == "all" {
		return nil, "all", nil
	}
	status, err := postdomain.NewPostStatus(normalized)
	if err != nil {
		return nil, "", err
	}
	return &status, status.String(), nil
}

func parseManageCommentStatus(raw string) (*commentdomain.CommentStatus, string, error) {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	if normalized == "" || normalized == "all" {
		return nil, "all", nil
	}
	status, err := commentdomain.NewCommentStatus(normalized)
	if err != nil {
		return nil, "", err
	}
	return &status, status.String(), nil
}

func parseManageReportStatus(raw string) (moderationdomain.ReportStatus, error) {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	if normalized == "" {
		return moderationdomain.ReportStatusPending, nil
	}
	return moderationdomain.NewReportStatus(normalized)
}

func toCommunityDTO(community communitydomain.Community, stats CommunityStats, followView communityFollowView, roleView communityRoleView, viewerID userdomain.UserID) Community {
	return Community{
		ID:                community.ID().String(),
		Slug:              community.Slug().String(),
		Name:              community.Name().String(),
		Description:       community.Description().String(),
		Kind:              community.Kind().String(),
		Status:            community.Status().String(),
		Visibility:        community.Visibility().String(),
		MemberCount:       stats.MemberCount,
		PostCount:         stats.PostCount,
		ViewerIsFollowing: followView.isFollowing,
		ViewerRole:        roleView.role.String(),
		ViewerPermissions: communityViewerPermissions(community, roleView.role, viewerID),
		CreatedAt:         community.CreatedAt(),
		UpdatedAt:         community.UpdatedAt(),
	}
}

func toCommunitySettingsDTO(community communitydomain.Community) CommunitySettings {
	return CommunitySettings{
		Name:        community.Name().String(),
		Description: community.Description().String(),
		UpdatedAt:   community.UpdatedAt(),
	}
}

func toCommunityRuleDTO(rule communitydomain.CommunityRule) CommunityRule {
	return CommunityRule{
		ID:          rule.ID().String(),
		CommunityID: rule.CommunityID().String(),
		Title:       rule.Title().String(),
		Body:        rule.Body().String(),
		Position:    rule.Position().Int(),
		CreatedBy:   rule.CreatedBy().String(),
		UpdatedBy:   rule.UpdatedBy().String(),
		CreatedAt:   rule.CreatedAt(),
		UpdatedAt:   rule.UpdatedAt(),
	}
}

func communityViewerPermissions(community communitydomain.Community, role communitydomain.MembershipRole, viewerID userdomain.UserID) ViewerPermissions {
	if isBlankUserID(viewerID) {
		return ViewerPermissions{}
	}
	return ViewerPermissions{
		CanPost:     isPostableCommunity(&community),
		CanManage:   role == communitydomain.MembershipRoleOwner,
		CanModerate: canModerateCommunity(role),
	}
}

func canModerateCommunity(role communitydomain.MembershipRole) bool {
	return role == communitydomain.MembershipRoleOwner || role == communitydomain.MembershipRoleModerator
}

func toCommunityManagePostDTO(post postdomain.Post) CommunityManagePost {
	return CommunityManagePost{
		ID:          post.ID().String(),
		CommunityID: post.CommunityID().String(),
		AuthorID:    post.AuthorID().String(),
		Title:       post.Title().String(),
		BodyExcerpt: communityManageExcerpt(post.Body().String()),
		Status:      post.Status().String(),
		CreatedAt:   post.CreatedAt(),
		UpdatedAt:   post.UpdatedAt(),
	}
}

func toCommunityManageCommentDTO(comment commentdomain.Comment) CommunityManageComment {
	parentID := ""
	if id, ok := comment.ParentID(); ok {
		parentID = id.String()
	}
	return CommunityManageComment{
		ID:          comment.ID().String(),
		PostID:      comment.PostID().String(),
		AuthorID:    comment.AuthorID().String(),
		ParentID:    parentID,
		BodyExcerpt: communityManageExcerpt(comment.Body().String()),
		Status:      comment.Status().String(),
		CreatedAt:   comment.CreatedAt(),
		UpdatedAt:   comment.UpdatedAt(),
	}
}

func toCommunityManageReportDTO(record moderationusecase.ContentReportRecord) CommunityManageReport {
	report := record.Report
	target := report.Target()
	postID := ""
	if id, ok := target.PostID(); ok {
		postID = id.String()
	}
	commentID := ""
	if id, ok := target.CommentID(); ok {
		commentID = id.String()
	}
	reviewedBy := ""
	if id, ok := report.ReviewedBy(); ok {
		reviewedBy = id.String()
	}
	var reviewedAt *time.Time
	if value, ok := report.ReviewedAt(); ok {
		copied := value
		reviewedAt = &copied
	}

	return CommunityManageReport{
		ID:            report.ID().String(),
		TargetType:    target.Type().String(),
		PostID:        postID,
		CommentID:     commentID,
		ReporterID:    report.ReporterID().String(),
		Reason:        report.Reason().String(),
		Status:        report.Status().String(),
		ReviewedBy:    reviewedBy,
		ReviewedAt:    reviewedAt,
		TargetPreview: toCommunityManageReportPreviewDTO(record.TargetPreview),
		CreatedAt:     report.CreatedAt(),
		UpdatedAt:     report.UpdatedAt(),
	}
}

func toCommunityManageReportPreviewDTO(preview *moderationusecase.ReportTargetPreview) *CommunityManageReportTargetPreview {
	if preview == nil {
		return nil
	}
	return &CommunityManageReportTargetPreview{
		TargetType:  preview.TargetType,
		PostID:      preview.PostID,
		CommentID:   preview.CommentID,
		AuthorID:    preview.AuthorID,
		Status:      preview.Status,
		Title:       preview.Title,
		BodyExcerpt: preview.BodyExcerpt,
		CreatedAt:   preview.CreatedAt,
		UpdatedAt:   preview.UpdatedAt,
	}
}

func communityManageExcerpt(raw string) string {
	const limit = 160

	compact := strings.Join(strings.Fields(raw), " ")
	runes := []rune(compact)
	if len(runes) <= limit {
		return compact
	}
	return string(runes[:limit]) + "..."
}

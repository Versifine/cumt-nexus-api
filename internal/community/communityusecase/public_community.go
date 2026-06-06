package communityusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
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
	communities CommunityRepository
	stats       CommunityStatsRepository
	follows     CommunityFollowRepository
	now         func() time.Time
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
	return uc
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
	for _, community := range communities {
		result.Communities = append(result.Communities, toCommunityDTO(community, stats[community.ID()], followViews[community.ID()], input.ViewerID))
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

	return GetCommunityResult{
		Community: toCommunityDTO(*community, stats[community.ID()], followViews[community.ID()], input.ViewerID),
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

	result := ListFollowedCommunitiesResult{
		Communities: make([]Community, 0, len(communities)),
		Limit:       limit,
		Offset:      offset,
	}
	for _, community := range communities {
		result.Communities = append(result.Communities, toCommunityDTO(community, stats[community.ID()], followViews[community.ID()], input.UserID))
	}

	return result, nil
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
		Community: toCommunityDTO(*community, CommunityStats{}, communityFollowView{}, ""),
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

func toCommunityDTO(community communitydomain.Community, stats CommunityStats, followView communityFollowView, viewerID userdomain.UserID) Community {
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
		ViewerPermissions: communityViewerPermissions(community, viewerID),
		CreatedAt:         community.CreatedAt(),
		UpdatedAt:         community.UpdatedAt(),
	}
}

func communityViewerPermissions(community communitydomain.Community, viewerID userdomain.UserID) ViewerPermissions {
	if isBlankUserID(viewerID) {
		return ViewerPermissions{}
	}
	return ViewerPermissions{
		CanPost: isPostableCommunity(&community),
	}
}

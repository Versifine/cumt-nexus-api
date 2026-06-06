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
)

type PublicCommunityBootstrapUseCase struct {
	communities CommunityRepository
	now         func() time.Time
}

type CommunityReadUseCase struct {
	communities CommunityRepository
	stats       CommunityStatsRepository
}

type GetCommunityInput struct {
	Slug string
}

type CanPostInCommunityInput struct {
	UserID      string
	CommunityID string
}

type ListCommunitiesResult struct {
	Communities []Community
}

type GetCommunityResult struct {
	Community Community
}

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
	}
	if repo, ok := communities.(CommunityStatsRepository); ok {
		uc.stats = repo
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

func (uc *CommunityReadUseCase) ListCommunities(ctx context.Context) (ListCommunitiesResult, error) {
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
	for _, community := range communities {
		result.Communities = append(result.Communities, toCommunityDTO(community, stats[community.ID()]))
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

	return GetCommunityResult{
		Community: toCommunityDTO(*community, stats[community.ID()]),
	}, nil
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
		Community: toCommunityDTO(*community, CommunityStats{}),
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

func toCommunityDTO(community communitydomain.Community, stats CommunityStats) Community {
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
		ViewerPermissions: ViewerPermissions{},
		CreatedAt:         community.CreatedAt(),
		UpdatedAt:         community.UpdatedAt(),
	}
}

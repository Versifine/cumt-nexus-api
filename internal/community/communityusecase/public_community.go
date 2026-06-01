package communityusecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
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
}

type GetCommunityInput struct {
	Slug string
}

type ListCommunitiesResult struct {
	Communities []Community
}

type GetCommunityResult struct {
	Community Community
}

type Community struct {
	ID          string
	Slug        string
	Name        string
	Description string
	Kind        string
	Status      string
	Visibility  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
	return &CommunityReadUseCase{
		communities: communities,
	}
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
	for _, community := range communities {
		result.Communities = append(result.Communities, toCommunityDTO(community))
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

	return GetCommunityResult{
		Community: toCommunityDTO(*community),
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

func toCommunityDTO(community communitydomain.Community) Community {
	return Community{
		ID:          community.ID().String(),
		Slug:        community.Slug().String(),
		Name:        community.Name().String(),
		Description: community.Description().String(),
		Kind:        community.Kind().String(),
		Status:      community.Status().String(),
		Visibility:  community.Visibility().String(),
		CreatedAt:   community.CreatedAt(),
		UpdatedAt:   community.UpdatedAt(),
	}
}

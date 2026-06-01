package communityusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
)

func TestEnsurePublicCommunityCreatesMissingCommunity(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	var created []communitydomain.Community
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			if slug.String() != PublicCommunitySlug {
				t.Fatalf("expected public slug, got %q", slug.String())
			}
			return nil, apperr.New(apperr.CodeNotFound, "community not found")
		},
		createFunc: func(ctx context.Context, community communitydomain.Community) error {
			created = append(created, community)
			return nil
		},
	}
	uc := NewPublicCommunityBootstrapUseCase(repo, func() time.Time { return now })

	if err := uc.EnsurePublicCommunity(context.Background()); err != nil {
		t.Fatalf("EnsurePublicCommunity returned error: %v", err)
	}

	if len(created) != 1 {
		t.Fatalf("expected one community to be created, got %d", len(created))
	}
	publicCommunity := created[0]
	if publicCommunity.Slug().String() != PublicCommunitySlug {
		t.Fatalf("expected public slug, got %q", publicCommunity.Slug().String())
	}
	if publicCommunity.Kind() != communitydomain.CommunityKindSystem {
		t.Fatalf("expected system kind, got %q", publicCommunity.Kind().String())
	}
	if publicCommunity.Status() != communitydomain.CommunityStatusActive {
		t.Fatalf("expected active status, got %q", publicCommunity.Status().String())
	}
	if publicCommunity.Visibility() != communitydomain.CommunityVisibilityPublic {
		t.Fatalf("expected public visibility, got %q", publicCommunity.Visibility().String())
	}
	if _, ok := publicCommunity.CreatedBy(); ok {
		t.Fatal("public community must not have created_by")
	}
	if !publicCommunity.CreatedAt().Equal(now) || !publicCommunity.UpdatedAt().Equal(now) {
		t.Fatalf("expected timestamps %s, got created=%s updated=%s", now, publicCommunity.CreatedAt(), publicCommunity.UpdatedAt())
	}
}

func TestEnsurePublicCommunityAcceptsExistingValidCommunity(t *testing.T) {
	existing := mustSystemCommunity(t, PublicCommunitySlug, time.Now().UTC())
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return existing, nil
		},
		createFunc: func(ctx context.Context, community communitydomain.Community) error {
			t.Fatal("Create should not be called for existing public community")
			return nil
		},
	}
	uc := NewPublicCommunityBootstrapUseCase(repo, time.Now)

	if err := uc.EnsurePublicCommunity(context.Background()); err != nil {
		t.Fatalf("EnsurePublicCommunity returned error: %v", err)
	}
}

func TestEnsurePublicCommunityRejectsMisconfiguredExistingCommunity(t *testing.T) {
	existing := mustCommunity(t, PublicCommunitySlug, communitydomain.CommunityStatusSuspended, time.Now().UTC())
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return existing, nil
		},
	}
	uc := NewPublicCommunityBootstrapUseCase(repo, time.Now)

	err := uc.EnsurePublicCommunity(context.Background())
	if !hasAppCode(err, apperr.CodeInternal) {
		t.Fatalf("expected internal for misconfigured public community, got %v", err)
	}
}

func TestEnsurePublicCommunityRefetchesAfterCreateConflict(t *testing.T) {
	existing := mustSystemCommunity(t, PublicCommunitySlug, time.Now().UTC())
	findCalls := 0
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			findCalls++
			if findCalls == 1 {
				return nil, apperr.New(apperr.CodeNotFound, "community not found")
			}
			return existing, nil
		},
		createFunc: func(ctx context.Context, community communitydomain.Community) error {
			return apperr.New(apperr.CodeConflict, "community slug already exists")
		},
	}
	uc := NewPublicCommunityBootstrapUseCase(repo, time.Now)

	if err := uc.EnsurePublicCommunity(context.Background()); err != nil {
		t.Fatalf("EnsurePublicCommunity returned error: %v", err)
	}
	if findCalls != 2 {
		t.Fatalf("expected two FindBySlug calls, got %d", findCalls)
	}
}

func TestListCommunitiesMapsActivePublicCommunities(t *testing.T) {
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	repo := &fakeCommunityRepository{
		listActivePublicFunc: func(ctx context.Context) ([]communitydomain.Community, error) {
			return []communitydomain.Community{
				*mustSystemCommunity(t, "public", now),
				*mustSystemCommunity(t, "campus", now.Add(time.Minute)),
			}, nil
		},
	}
	uc := NewCommunityReadUseCase(repo)

	result, err := uc.ListCommunities(context.Background())
	if err != nil {
		t.Fatalf("ListCommunities returned error: %v", err)
	}

	if len(result.Communities) != 2 {
		t.Fatalf("expected two communities, got %d", len(result.Communities))
	}
	if result.Communities[0].Slug != "public" || result.Communities[1].Slug != "campus" {
		t.Fatalf("unexpected slugs: %#v", result.Communities)
	}
	if result.Communities[0].Kind != communitydomain.CommunityKindSystem.String() {
		t.Fatalf("expected system kind, got %q", result.Communities[0].Kind)
	}
}

func TestListCommunitiesWrapsRepositoryError(t *testing.T) {
	repoErr := errors.New("database down")
	repo := &fakeCommunityRepository{
		listActivePublicFunc: func(ctx context.Context) ([]communitydomain.Community, error) {
			return nil, repoErr
		},
	}
	uc := NewCommunityReadUseCase(repo)

	_, err := uc.ListCommunities(context.Background())
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repository error to be wrapped, got %v", err)
	}
}

func TestGetCommunityBySlugReturnsCommunity(t *testing.T) {
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	expected := mustSystemCommunity(t, "campus", now)
	var gotSlug communitydomain.CommunitySlug
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			gotSlug = slug
			return expected, nil
		},
	}
	uc := NewCommunityReadUseCase(repo)

	result, err := uc.GetCommunityBySlug(context.Background(), GetCommunityInput{Slug: "campus"})
	if err != nil {
		t.Fatalf("GetCommunityBySlug returned error: %v", err)
	}

	if gotSlug.String() != "campus" {
		t.Fatalf("expected slug campus, got %q", gotSlug.String())
	}
	if result.Community.Slug != "campus" {
		t.Fatalf("expected response slug campus, got %q", result.Community.Slug)
	}
}

func TestGetCommunityBySlugRejectsInvalidSlug(t *testing.T) {
	repo := &fakeCommunityRepository{}
	uc := NewCommunityReadUseCase(repo)

	_, err := uc.GetCommunityBySlug(context.Background(), GetCommunityInput{Slug: "no"})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid slug, got %v", err)
	}
}

func TestGetCommunityBySlugHidesNonReadableCommunity(t *testing.T) {
	suspended := mustCommunity(t, "campus", communitydomain.CommunityStatusSuspended, time.Now().UTC())
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return suspended, nil
		},
	}
	uc := NewCommunityReadUseCase(repo)

	_, err := uc.GetCommunityBySlug(context.Background(), GetCommunityInput{Slug: "campus"})
	if !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for non-readable community, got %v", err)
	}
}

type fakeCommunityRepository struct {
	createFunc           func(ctx context.Context, community communitydomain.Community) error
	findBySlugFunc       func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error)
	listActivePublicFunc func(ctx context.Context) ([]communitydomain.Community, error)
}

func (f *fakeCommunityRepository) Create(ctx context.Context, community communitydomain.Community) error {
	if f.createFunc != nil {
		return f.createFunc(ctx, community)
	}
	return nil
}

func (f *fakeCommunityRepository) FindBySlug(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
	if f.findBySlugFunc != nil {
		return f.findBySlugFunc(ctx, slug)
	}
	return nil, apperr.New(apperr.CodeNotFound, "community not found")
}

func (f *fakeCommunityRepository) ListActivePublic(ctx context.Context) ([]communitydomain.Community, error) {
	if f.listActivePublicFunc != nil {
		return f.listActivePublicFunc(ctx)
	}
	return nil, nil
}

func mustSystemCommunity(t *testing.T, rawSlug string, now time.Time) *communitydomain.Community {
	t.Helper()

	community, err := communitydomain.NewSystemCommunity(
		communitydomain.NewGeneratedCommunityID(),
		mustCommunitySlug(t, rawSlug),
		mustCommunityName(t, rawSlug),
		communitydomain.NewCommunityDescription("test community"),
		now,
	)
	if err != nil {
		t.Fatalf("NewSystemCommunity returned error: %v", err)
	}
	return community
}

func mustCommunity(t *testing.T, rawSlug string, status communitydomain.CommunityStatus, now time.Time) *communitydomain.Community {
	t.Helper()

	community, err := communitydomain.RehydrateCommunity(
		communitydomain.NewGeneratedCommunityID(),
		mustCommunitySlug(t, rawSlug),
		mustCommunityName(t, rawSlug),
		communitydomain.NewCommunityDescription("test community"),
		communitydomain.CommunityKindSystem,
		status,
		communitydomain.CommunityVisibilityPublic,
		nil,
		now,
		now,
	)
	if err != nil {
		t.Fatalf("RehydrateCommunity returned error: %v", err)
	}
	return community
}

func mustCommunitySlug(t *testing.T, raw string) communitydomain.CommunitySlug {
	t.Helper()

	slug, err := communitydomain.NewCommunitySlug(raw)
	if err != nil {
		t.Fatalf("NewCommunitySlug(%q) returned error: %v", raw, err)
	}
	return slug
}

func mustCommunityName(t *testing.T, raw string) communitydomain.CommunityName {
	t.Helper()

	name, err := communitydomain.NewCommunityName(raw)
	if err != nil {
		t.Fatalf("NewCommunityName(%q) returned error: %v", raw, err)
	}
	return name
}

func hasAppCode(err error, code apperr.Code) bool {
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Code() == code
}

package communityusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationusecase"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
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

	result, err := uc.ListCommunities(context.Background(), ListCommunitiesInput{})
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

	_, err := uc.ListCommunities(context.Background(), ListCommunitiesInput{})
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

func TestGetCommunityBySlugReturnsViewerFollowState(t *testing.T) {
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	expected := mustSystemCommunity(t, "campus", now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return expected, nil
		},
		followedCommunityIDs: map[communitydomain.CommunityID]bool{
			expected.ID(): true,
		},
	}
	uc := NewCommunityReadUseCase(repo)

	result, err := uc.GetCommunityBySlug(context.Background(), GetCommunityInput{Slug: "campus", ViewerID: viewerID})
	if err != nil {
		t.Fatalf("GetCommunityBySlug returned error: %v", err)
	}
	if !result.Community.ViewerIsFollowing {
		t.Fatal("expected viewer_is_following=true")
	}
	if !result.Community.ViewerPermissions.CanPost {
		t.Fatal("expected logged-in viewer can_post=true for active public community")
	}
	if !repo.findFollowedCalled {
		t.Fatal("expected followed community lookup")
	}
}

func TestGetCommunityBySlugReturnsViewerRoleAndPermissions(t *testing.T) {
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	expected := mustSystemCommunity(t, "campus", now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return expected, nil
		},
		activeRolesByCommunityID: map[communitydomain.CommunityID]communitydomain.MembershipRole{
			expected.ID(): communitydomain.MembershipRoleOwner,
		},
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)

	result, err := uc.GetCommunityBySlug(context.Background(), GetCommunityInput{Slug: "campus", ViewerID: viewerID})
	if err != nil {
		t.Fatalf("GetCommunityBySlug returned error: %v", err)
	}
	if result.Community.ViewerRole != communitydomain.MembershipRoleOwner.String() {
		t.Fatalf("expected owner viewer role, got %q", result.Community.ViewerRole)
	}
	if !result.Community.ViewerPermissions.CanPost || !result.Community.ViewerPermissions.CanManage || !result.Community.ViewerPermissions.CanModerate {
		t.Fatalf("unexpected viewer permissions: %#v", result.Community.ViewerPermissions)
	}
	if !repo.findActiveRolesCalled {
		t.Fatal("expected membership role lookup")
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

func TestListFollowedCommunitiesReturnsViewerRoleAndPermissions(t *testing.T) {
	now := time.Date(2026, 6, 9, 9, 0, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", now)
	repo := &fakeCommunityRepository{
		listFollowedActivePublicFunc: func(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]communitydomain.Community, error) {
			if userID != viewerID || limit != 21 || offset != 0 {
				t.Fatalf("unexpected followed args: user=%q limit=%d offset=%d", userID.String(), limit, offset)
			}
			return []communitydomain.Community{*community}, nil
		},
		followedCommunityIDs: map[communitydomain.CommunityID]bool{
			community.ID(): true,
		},
		activeRolesByCommunityID: map[communitydomain.CommunityID]communitydomain.MembershipRole{
			community.ID(): communitydomain.MembershipRoleModerator,
		},
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)

	result, err := uc.ListFollowedCommunities(context.Background(), ListFollowedCommunitiesInput{
		UserID: viewerID,
		Limit:  20,
	})
	if err != nil {
		t.Fatalf("ListFollowedCommunities returned error: %v", err)
	}
	if len(result.Communities) != 1 {
		t.Fatalf("expected one community, got %d", len(result.Communities))
	}
	got := result.Communities[0]
	if !got.ViewerIsFollowing {
		t.Fatal("expected followed community to set viewer_is_following=true")
	}
	if got.ViewerRole != communitydomain.MembershipRoleModerator.String() || !got.ViewerPermissions.CanModerate || got.ViewerPermissions.CanManage {
		t.Fatalf("unexpected viewer role or permissions: role=%q permissions=%#v", got.ViewerRole, got.ViewerPermissions)
	}
}

func TestGetCommunityManageContextRequiresModeratorRole(t *testing.T) {
	now := time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)

	_, err := uc.GetCommunityManageContext(context.Background(), GetCommunityManageContextInput{Slug: "campus", ViewerID: viewerID})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for non-moderator, got %v", err)
	}
}

func TestGetCommunityManageContextAllowsPlatformOwnerOverride(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		isPlatformOwner: true,
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)
	uc.SetPlatformOwnerRepository(repo)

	result, err := uc.GetCommunityManageContext(context.Background(), GetCommunityManageContextInput{Slug: "campus", ViewerID: viewerID})
	if err != nil {
		t.Fatalf("GetCommunityManageContext returned error: %v", err)
	}
	if result.Community.ViewerRole != "" {
		t.Fatalf("expected real viewer role to stay empty, got %q", result.Community.ViewerRole)
	}
	if !result.Community.ViewerPermissions.CanManage || !result.Community.ViewerPermissions.CanModerate || !result.Community.ViewerPermissions.PlatformOwnerOverride {
		t.Fatalf("expected platform owner override permissions, got %#v", result.Community.ViewerPermissions)
	}
}

func TestListCommunityMembersReturnsMembersForModerator(t *testing.T) {
	now := time.Date(2026, 6, 9, 11, 0, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", now)
	members := []CommunityMember{
		{
			UserID:    viewerID.String(),
			Username:  "alice",
			Role:      communitydomain.MembershipRoleOwner.String(),
			Status:    communitydomain.MembershipStatusActive.String(),
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		activeRolesByCommunityID: map[communitydomain.CommunityID]communitydomain.MembershipRole{
			community.ID(): communitydomain.MembershipRoleModerator,
		},
		activeMembers: members,
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)

	result, err := uc.ListCommunityMembers(context.Background(), ListCommunityMembersInput{
		Slug:     "campus",
		ViewerID: viewerID,
		Limit:    20,
		Offset:   5,
	})
	if err != nil {
		t.Fatalf("ListCommunityMembers returned error: %v", err)
	}
	if result.Community.ViewerRole != communitydomain.MembershipRoleModerator.String() || !result.Community.ViewerPermissions.CanModerate {
		t.Fatalf("unexpected community view: %#v", result.Community)
	}
	if len(result.Members) != 1 || result.Members[0].Username != "alice" {
		t.Fatalf("unexpected members: %#v", result.Members)
	}
	if repo.listActiveMembersCommunityID != community.ID() || repo.listActiveMembersLimit != 21 || repo.listActiveMembersOffset != 5 {
		t.Fatalf("unexpected list active members args: community=%q limit=%d offset=%d", repo.listActiveMembersCommunityID.String(), repo.listActiveMembersLimit, repo.listActiveMembersOffset)
	}
}

func TestCreateCommunityOwnerTransferRequiresRealCommunityOwner(t *testing.T) {
	now := time.Date(2026, 6, 15, 11, 0, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		isPlatformOwner: true,
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)
	uc.SetPlatformOwnerRepository(repo)

	_, err := uc.CreateCommunityOwnerTransfer(context.Background(), CreateCommunityOwnerTransferInput{
		Slug:     "campus",
		ViewerID: viewerID,
		Username: "alice",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for platform owner override owner transfer, got %v", err)
	}
}

func TestListCommunityManagePostsReturnsPostsForModerator(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	authorID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", now)
	post := mustPost(t, community.ID(), authorID, "Manage me", postdomain.PostStatusRemoved, now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		activeRolesByCommunityID: map[communitydomain.CommunityID]communitydomain.MembershipRole{
			community.ID(): communitydomain.MembershipRoleModerator,
		},
		managePosts: []postdomain.Post{*post},
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)
	uc.SetManageContentReaders(repo, repo, repo)

	result, err := uc.ListCommunityManagePosts(context.Background(), ListCommunityManagePostsInput{
		Slug:     "campus",
		ViewerID: viewerID,
		Status:   "removed",
		Limit:    20,
		Offset:   5,
	})
	if err != nil {
		t.Fatalf("ListCommunityManagePosts returned error: %v", err)
	}
	if result.Status != postdomain.PostStatusRemoved.String() || len(result.Posts) != 1 || result.Posts[0].Title != "Manage me" {
		t.Fatalf("unexpected posts result: %#v", result)
	}
	if repo.managePostsCommunityID != community.ID() || repo.managePostsStatus == nil || *repo.managePostsStatus != postdomain.PostStatusRemoved || repo.managePostsLimit != 21 || repo.managePostsOffset != 5 {
		t.Fatalf("unexpected manage posts args: community=%q status=%v limit=%d offset=%d", repo.managePostsCommunityID.String(), repo.managePostsStatus, repo.managePostsLimit, repo.managePostsOffset)
	}
}

func TestListCommunityManagePostsRequiresModeratorRole(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 30, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)
	uc.SetManageContentReaders(repo, repo, repo)

	_, err := uc.ListCommunityManagePosts(context.Background(), ListCommunityManagePostsInput{Slug: "campus", ViewerID: viewerID})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for non-moderator, got %v", err)
	}
	if repo.managePostsCalled {
		t.Fatal("manage posts repository should not be called for non-moderator")
	}
}

func TestGetCommunityManageSettingsReturnsSettingsForModerator(t *testing.T) {
	now := time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		activeRolesByCommunityID: map[communitydomain.CommunityID]communitydomain.MembershipRole{
			community.ID(): communitydomain.MembershipRoleModerator,
		},
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)

	result, err := uc.GetCommunityManageSettings(context.Background(), GetCommunityManageSettingsInput{
		Slug:     "campus",
		ViewerID: viewerID,
	})
	if err != nil {
		t.Fatalf("GetCommunityManageSettings returned error: %v", err)
	}
	if result.Settings.Name != community.Name().String() || result.Settings.Description != community.Description().String() {
		t.Fatalf("unexpected settings: %#v", result.Settings)
	}
	if result.Community.ViewerRole != communitydomain.MembershipRoleModerator.String() || !result.Community.ViewerPermissions.CanModerate {
		t.Fatalf("unexpected community view: %#v", result.Community)
	}
}

func TestUpdateCommunityManageSettingsPersistsForOwner(t *testing.T) {
	createdAt := time.Date(2026, 6, 10, 9, 30, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", createdAt)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		activeRolesByCommunityID: map[communitydomain.CommunityID]communitydomain.MembershipRole{
			community.ID(): communitydomain.MembershipRoleOwner,
		},
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)
	uc.now = func() time.Time { return updatedAt }

	result, err := uc.UpdateCommunityManageSettings(context.Background(), UpdateCommunityManageSettingsInput{
		Slug:        "campus",
		ViewerID:    viewerID,
		Name:        "Campus Hub",
		Description: "Rules and updates",
	})
	if err != nil {
		t.Fatalf("UpdateCommunityManageSettings returned error: %v", err)
	}
	if !repo.updateDetailsCalled {
		t.Fatal("expected UpdateDetails to be called")
	}
	if repo.updatedCommunity.ID() != community.ID() || repo.updatedCommunity.Name().String() != "Campus Hub" || repo.updatedCommunity.Description().String() != "Rules and updates" || !repo.updatedCommunity.UpdatedAt().Equal(updatedAt) {
		t.Fatalf("unexpected updated community: id=%q name=%q description=%q updated=%s", repo.updatedCommunity.ID().String(), repo.updatedCommunity.Name().String(), repo.updatedCommunity.Description().String(), repo.updatedCommunity.UpdatedAt())
	}
	if result.Settings.Name != "Campus Hub" || result.Settings.Description != "Rules and updates" || !result.Settings.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected settings result: %#v", result.Settings)
	}
}

func TestUpdateCommunityManageSettingsRequiresOwner(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		activeRolesByCommunityID: map[communitydomain.CommunityID]communitydomain.MembershipRole{
			community.ID(): communitydomain.MembershipRoleModerator,
		},
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)

	_, err := uc.UpdateCommunityManageSettings(context.Background(), UpdateCommunityManageSettingsInput{
		Slug:     "campus",
		ViewerID: viewerID,
		Name:     "Campus Hub",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for moderator settings update, got %v", err)
	}
	if repo.updateDetailsCalled {
		t.Fatal("settings repository should not be called for moderator")
	}
}

func TestListCommunityRulesReturnsRulesForModerator(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 30, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", now)
	rule := mustCommunityRule(t, community.ID(), viewerID, "Be kind", "Keep discussions constructive.", 1, now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		activeRolesByCommunityID: map[communitydomain.CommunityID]communitydomain.MembershipRole{
			community.ID(): communitydomain.MembershipRoleModerator,
		},
		rules: []communitydomain.CommunityRule{*rule},
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)

	result, err := uc.ListCommunityRules(context.Background(), ListCommunityRulesInput{
		Slug:     "campus",
		ViewerID: viewerID,
	})
	if err != nil {
		t.Fatalf("ListCommunityRules returned error: %v", err)
	}
	if repo.listRulesCommunityID != community.ID() {
		t.Fatalf("expected list rules community %q, got %q", community.ID().String(), repo.listRulesCommunityID.String())
	}
	if len(result.Rules) != 1 || result.Rules[0].Title != "Be kind" || result.Rules[0].Position != 1 {
		t.Fatalf("unexpected rules result: %#v", result.Rules)
	}
}

func TestCreateCommunityRuleAllowsModerator(t *testing.T) {
	now := time.Date(2026, 6, 10, 11, 0, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		activeRolesByCommunityID: map[communitydomain.CommunityID]communitydomain.MembershipRole{
			community.ID(): communitydomain.MembershipRoleModerator,
		},
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)
	uc.now = func() time.Time { return now }

	result, err := uc.CreateCommunityRule(context.Background(), CreateCommunityRuleInput{
		Slug:     "campus",
		ViewerID: viewerID,
		Title:    "Stay on topic",
		Body:     "Posts should match the community.",
		Position: 2,
	})
	if err != nil {
		t.Fatalf("CreateCommunityRule returned error: %v", err)
	}
	if !repo.createRuleCalled {
		t.Fatal("expected CreateRule to be called")
	}
	if repo.createdRule.CommunityID() != community.ID() || repo.createdRule.Title().String() != "Stay on topic" || repo.createdRule.Body().String() != "Posts should match the community." || repo.createdRule.Position().Int() != 2 || repo.createdRule.CreatedBy() != viewerID || repo.createdRule.UpdatedBy() != viewerID {
		t.Fatalf("unexpected created rule: %#v", repo.createdRule)
	}
	if result.Rule.ID == "" || result.Rule.Title != "Stay on topic" || result.Rule.Position != 2 {
		t.Fatalf("unexpected rule result: %#v", result.Rule)
	}
}

func TestUpdateCommunityRuleRejectsRuleFromAnotherCommunity(t *testing.T) {
	now := time.Date(2026, 6, 10, 11, 30, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", now)
	otherCommunity := mustSystemCommunity(t, "other-campus", now)
	rule := mustCommunityRule(t, otherCommunity.ID(), viewerID, "Other", "Other community rule.", 1, now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		activeRolesByCommunityID: map[communitydomain.CommunityID]communitydomain.MembershipRole{
			community.ID(): communitydomain.MembershipRoleModerator,
		},
		ruleByID: rule,
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)

	_, err := uc.UpdateCommunityRule(context.Background(), UpdateCommunityRuleInput{
		Slug:     "campus",
		RuleID:   rule.ID().String(),
		ViewerID: viewerID,
		Title:    "Updated",
		Position: 1,
	})
	if !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for cross-community rule, got %v", err)
	}
	if repo.updateRuleCalled {
		t.Fatal("rule repository update should not be called for cross-community rule")
	}
}

func TestUpdateCommunityRulePersistsForModerator(t *testing.T) {
	createdAt := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", createdAt)
	rule := mustCommunityRule(t, community.ID(), userdomain.NewGeneratedUserID(), "Old", "Old body.", 1, createdAt)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		activeRolesByCommunityID: map[communitydomain.CommunityID]communitydomain.MembershipRole{
			community.ID(): communitydomain.MembershipRoleModerator,
		},
		ruleByID: rule,
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)
	uc.now = func() time.Time { return updatedAt }

	result, err := uc.UpdateCommunityRule(context.Background(), UpdateCommunityRuleInput{
		Slug:     "campus",
		RuleID:   rule.ID().String(),
		ViewerID: viewerID,
		Title:    "Updated",
		Body:     "Updated body.",
		Position: 3,
	})
	if err != nil {
		t.Fatalf("UpdateCommunityRule returned error: %v", err)
	}
	if !repo.updateRuleCalled {
		t.Fatal("expected UpdateRule to be called")
	}
	if repo.updatedRule.ID() != rule.ID() || repo.updatedRule.Title().String() != "Updated" || repo.updatedRule.Body().String() != "Updated body." || repo.updatedRule.Position().Int() != 3 || repo.updatedRule.UpdatedBy() != viewerID || !repo.updatedRule.UpdatedAt().Equal(updatedAt) {
		t.Fatalf("unexpected updated rule: %#v", repo.updatedRule)
	}
	if result.Rule.Title != "Updated" || result.Rule.Position != 3 {
		t.Fatalf("unexpected rule result: %#v", result.Rule)
	}
}

func TestDeleteCommunityRuleScopesDeleteToManagedCommunity(t *testing.T) {
	now := time.Date(2026, 6, 10, 13, 0, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", now)
	ruleID := communitydomain.NewGeneratedCommunityRuleID()
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		activeRolesByCommunityID: map[communitydomain.CommunityID]communitydomain.MembershipRole{
			community.ID(): communitydomain.MembershipRoleModerator,
		},
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipReader(repo)

	if _, err := uc.DeleteCommunityRule(context.Background(), DeleteCommunityRuleInput{
		Slug:     "campus",
		RuleID:   ruleID.String(),
		ViewerID: viewerID,
	}); err != nil {
		t.Fatalf("DeleteCommunityRule returned error: %v", err)
	}
	if !repo.deleteRuleCalled {
		t.Fatal("expected DeleteRule to be called")
	}
	if repo.deletedRuleID != ruleID || repo.deletedRuleCommunityID != community.ID() {
		t.Fatalf("unexpected delete args: rule=%q community=%q", repo.deletedRuleID.String(), repo.deletedRuleCommunityID.String())
	}
}

func TestCanPostInCommunityAllowsActivePublicCommunity(t *testing.T) {
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	community := mustSystemCommunity(t, "campus", now)
	repo := &fakeCommunityRepository{
		findByIDFunc: func(ctx context.Context, id communitydomain.CommunityID) (*communitydomain.Community, error) {
			if id != community.ID() {
				t.Fatalf("expected community id %q, got %q", community.ID().String(), id.String())
			}
			return community, nil
		},
	}
	uc := NewCommunityReadUseCase(repo)

	result, err := uc.CanPostInCommunity(context.Background(), CanPostInCommunityInput{
		UserID:      userdomain.NewGeneratedUserID().String(),
		CommunityID: community.ID().String(),
	})
	if err != nil {
		t.Fatalf("CanPostInCommunity returned error: %v", err)
	}
	if result.Community.ID != community.ID().String() {
		t.Fatalf("expected community %q, got %q", community.ID().String(), result.Community.ID)
	}
}

func TestCanPostInCommunityRejectsNonPostableCommunity(t *testing.T) {
	community := mustCommunity(t, "campus", communitydomain.CommunityStatusSuspended, time.Now().UTC())
	repo := &fakeCommunityRepository{
		findByIDFunc: func(ctx context.Context, id communitydomain.CommunityID) (*communitydomain.Community, error) {
			return community, nil
		},
	}
	uc := NewCommunityReadUseCase(repo)

	_, err := uc.CanPostInCommunity(context.Background(), CanPostInCommunityInput{
		UserID:      userdomain.NewGeneratedUserID().String(),
		CommunityID: community.ID().String(),
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for non-postable community, got %v", err)
	}
}

func TestFollowCommunityPersistsReadableCommunityFollow(t *testing.T) {
	now := time.Date(2026, 6, 6, 10, 30, 0, 0, time.UTC)
	userID := userdomain.NewGeneratedUserID()
	community := mustSystemCommunity(t, "campus", now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		followCommunityFunc: func(ctx context.Context, communityID communitydomain.CommunityID, gotUserID userdomain.UserID, followedAt time.Time) error {
			if communityID != community.ID() || gotUserID != userID || !followedAt.Equal(now) {
				t.Fatalf("unexpected follow args: community=%q user=%q at=%s", communityID.String(), gotUserID.String(), followedAt)
			}
			return nil
		},
	}
	uc := NewCommunityReadUseCase(repo)
	uc.now = func() time.Time { return now }

	if _, err := uc.FollowCommunity(context.Background(), FollowCommunityInput{Slug: "campus", UserID: userID}); err != nil {
		t.Fatalf("FollowCommunity returned error: %v", err)
	}
	if !repo.followCommunityCalled {
		t.Fatal("expected follow repository call")
	}
}

func TestAddCommunityModeratorCreatesActiveModerator(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 30, 0, 0, time.UTC)
	ownerID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	community := mustCommunity(t, "campus", communitydomain.CommunityStatusActive, now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		activeRolesByCommunityID: map[communitydomain.CommunityID]communitydomain.MembershipRole{
			community.ID(): communitydomain.MembershipRoleOwner,
		},
		activeUserByUsername: map[string]CommunityMember{
			"alice": {UserID: targetID.String(), Username: "alice"},
		},
		activeUserByID: map[string]CommunityMember{
			targetID.String(): {UserID: targetID.String(), Username: "alice"},
		},
		activeMemberCount:    1,
		activeModeratorCount: 0,
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipRepository(&fakeCommunityMembershipOps{repo})
	uc.now = func() time.Time { return now }

	result, err := uc.AddCommunityModerator(context.Background(), AddCommunityModeratorInput{
		Slug:     "campus",
		ViewerID: ownerID,
		Username: "alice",
	})
	if err != nil {
		t.Fatalf("AddCommunityModerator returned error: %v", err)
	}
	if repo.upsertedMemberUserID != targetID || repo.upsertedMemberRole != communitydomain.MembershipRoleModerator {
		t.Fatalf("expected moderator upsert for target, got user=%q role=%q", repo.upsertedMemberUserID.String(), repo.upsertedMemberRole.String())
	}
	if result.Member.Role != "moderator" || result.Member.Username != "alice" {
		t.Fatalf("unexpected moderator result: %#v", result.Member)
	}
}

func TestAddCommunityModeratorRequiresOwner(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 35, 0, 0, time.UTC)
	viewerID := userdomain.NewGeneratedUserID()
	community := mustCommunity(t, "campus", communitydomain.CommunityStatusActive, now)
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		activeRolesByCommunityID: map[communitydomain.CommunityID]communitydomain.MembershipRole{
			community.ID(): communitydomain.MembershipRoleModerator,
		},
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipRepository(&fakeCommunityMembershipOps{repo})

	_, err := uc.AddCommunityModerator(context.Background(), AddCommunityModeratorInput{
		Slug:     "campus",
		ViewerID: viewerID,
		Username: "alice",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestAcceptCommunityOwnerTransferRequiresTargetUser(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 40, 0, 0, time.UTC)
	fromID := userdomain.NewGeneratedUserID()
	toID := userdomain.NewGeneratedUserID()
	viewerID := userdomain.NewGeneratedUserID()
	community := mustCommunity(t, "campus", communitydomain.CommunityStatusActive, now)
	transferID := "7b53252d-65c8-4c68-916c-255701c24b28"
	repo := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return community, nil
		},
		ownerTransfer: &CommunityOwnerTransferRecord{
			ID:          transferID,
			CommunityID: community.ID(),
			FromUserID:  fromID,
			ToUserID:    toID,
			Status:      "pending",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
	uc := NewCommunityReadUseCase(repo)
	uc.SetMembershipRepository(&fakeCommunityMembershipOps{repo})
	uc.now = func() time.Time { return now }

	_, err := uc.AcceptCommunityOwnerTransfer(context.Background(), AcceptCommunityOwnerTransferInput{
		Slug:       "campus",
		ViewerID:   viewerID,
		TransferID: transferID,
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if repo.acceptedOwnerTransferID != "" || repo.transferOwnerUserID != "" {
		t.Fatalf("transfer should not be applied: accepted=%q owner=%q", repo.acceptedOwnerTransferID, repo.transferOwnerUserID.String())
	}
}

type fakeCommunityRepository struct {
	createFunc                     func(ctx context.Context, community communitydomain.Community) error
	findByIDFunc                   func(ctx context.Context, id communitydomain.CommunityID) (*communitydomain.Community, error)
	findBySlugFunc                 func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error)
	listActivePublicFunc           func(ctx context.Context) ([]communitydomain.Community, error)
	followCommunityFunc            func(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID, now time.Time) error
	deleteCommunityFollowFunc      func(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID) error
	listFollowedActivePublicFunc   func(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]communitydomain.Community, error)
	followCommunityCalled          bool
	deleteCommunityFollowCalled    bool
	listFollowedActivePublicCalled bool
	findFollowedCalled             bool
	followedCommunityIDs           map[communitydomain.CommunityID]bool
	findActiveRolesCalled          bool
	activeRolesByCommunityID       map[communitydomain.CommunityID]communitydomain.MembershipRole
	isPlatformOwner                bool
	listActiveMembersCalled        bool
	listActiveMembersCommunityID   communitydomain.CommunityID
	listActiveMembersLimit         int
	listActiveMembersOffset        int
	activeMembers                  []CommunityMember
	activeUserByUsername           map[string]CommunityMember
	activeUserByID                 map[string]CommunityMember
	activeMemberByUserID           map[string]CommunityMember
	activeMemberCount              int
	activeModeratorCount           int
	upsertedMemberRole             communitydomain.MembershipRole
	upsertedMemberUserID           userdomain.UserID
	createdOwnerTransfer           *CommunityOwnerTransferRecord
	ownerTransfer                  *CommunityOwnerTransferRecord
	acceptedOwnerTransferID        string
	transferOwnerUserID            userdomain.UserID
	managePostsCalled              bool
	managePostsCommunityID         communitydomain.CommunityID
	managePostsStatus              *postdomain.PostStatus
	managePostsLimit               int
	managePostsOffset              int
	managePosts                    []postdomain.Post
	manageCommentsCalled           bool
	manageCommentsCommunityID      communitydomain.CommunityID
	manageCommentsStatus           *commentdomain.CommentStatus
	manageCommentsLimit            int
	manageCommentsOffset           int
	manageComments                 []commentdomain.Comment
	manageReportsCalled            bool
	manageReportsCommunityID       communitydomain.CommunityID
	manageReportsStatus            moderationdomain.ReportStatus
	manageReportsLimit             int
	manageReportsOffset            int
	manageReports                  []moderationusecase.ContentReportRecord
	updateDetailsCalled            bool
	updatedCommunity               communitydomain.Community
	listRulesCommunityID           communitydomain.CommunityID
	rules                          []communitydomain.CommunityRule
	ruleByID                       *communitydomain.CommunityRule
	createRuleCalled               bool
	createdRule                    communitydomain.CommunityRule
	updateRuleCalled               bool
	updatedRule                    communitydomain.CommunityRule
	deleteRuleCalled               bool
	deletedRuleID                  communitydomain.CommunityRuleID
	deletedRuleCommunityID         communitydomain.CommunityID
}

type fakeCommunityMembershipOps struct {
	*fakeCommunityRepository
}

func (f *fakeCommunityMembershipOps) Create(ctx context.Context, membership communitydomain.CommunityMembership) error {
	return nil
}

func (f *fakeCommunityRepository) Create(ctx context.Context, community communitydomain.Community) error {
	if f.createFunc != nil {
		return f.createFunc(ctx, community)
	}
	return nil
}

func (f *fakeCommunityRepository) FindByID(ctx context.Context, id communitydomain.CommunityID) (*communitydomain.Community, error) {
	if f.findByIDFunc != nil {
		return f.findByIDFunc(ctx, id)
	}
	return nil, apperr.New(apperr.CodeNotFound, "community not found")
}

func (f *fakeCommunityRepository) FindBySlug(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
	if f.findBySlugFunc != nil {
		return f.findBySlugFunc(ctx, slug)
	}
	return nil, apperr.New(apperr.CodeNotFound, "community not found")
}

func (f *fakeCommunityRepository) ListActivePublic(ctx context.Context, limit int, offset int) ([]communitydomain.Community, error) {
	if f.listActivePublicFunc != nil {
		return f.listActivePublicFunc(ctx)
	}
	return nil, nil
}

func (f *fakeCommunityRepository) UpdateDetails(ctx context.Context, community communitydomain.Community) error {
	f.updateDetailsCalled = true
	f.updatedCommunity = community
	return nil
}

func (f *fakeCommunityRepository) FollowCommunity(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID, now time.Time) error {
	f.followCommunityCalled = true
	if f.followCommunityFunc != nil {
		return f.followCommunityFunc(ctx, communityID, userID, now)
	}
	return nil
}

func (f *fakeCommunityRepository) DeleteCommunityFollow(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID) error {
	f.deleteCommunityFollowCalled = true
	if f.deleteCommunityFollowFunc != nil {
		return f.deleteCommunityFollowFunc(ctx, communityID, userID)
	}
	return nil
}

func (f *fakeCommunityRepository) ListFollowedActivePublic(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]communitydomain.Community, error) {
	f.listFollowedActivePublicCalled = true
	if f.listFollowedActivePublicFunc != nil {
		return f.listFollowedActivePublicFunc(ctx, userID, limit, offset)
	}
	return nil, nil
}

func (f *fakeCommunityRepository) FindFollowedCommunityIDsByUser(ctx context.Context, communityIDs []communitydomain.CommunityID, userID userdomain.UserID) (map[communitydomain.CommunityID]bool, error) {
	f.findFollowedCalled = true
	if f.followedCommunityIDs == nil {
		return map[communitydomain.CommunityID]bool{}, nil
	}
	return f.followedCommunityIDs, nil
}

func (f *fakeCommunityRepository) FindActiveRolesByUser(ctx context.Context, communityIDs []communitydomain.CommunityID, userID userdomain.UserID) (map[communitydomain.CommunityID]communitydomain.MembershipRole, error) {
	f.findActiveRolesCalled = true
	if f.activeRolesByCommunityID == nil {
		return map[communitydomain.CommunityID]communitydomain.MembershipRole{}, nil
	}
	return f.activeRolesByCommunityID, nil
}

func (f *fakeCommunityRepository) IsPlatformOwner(ctx context.Context, userID userdomain.UserID) (bool, error) {
	return f.isPlatformOwner, nil
}

func (f *fakeCommunityRepository) ListActiveMembers(ctx context.Context, communityID communitydomain.CommunityID, limit int, offset int) ([]CommunityMember, error) {
	f.listActiveMembersCalled = true
	f.listActiveMembersCommunityID = communityID
	f.listActiveMembersLimit = limit
	f.listActiveMembersOffset = offset
	return f.activeMembers, nil
}

func (f *fakeCommunityRepository) FindActiveMemberByUserID(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID) (CommunityMember, error) {
	if f.activeMemberByUserID != nil {
		if member, ok := f.activeMemberByUserID[userID.String()]; ok {
			return member, nil
		}
	}
	return CommunityMember{}, apperr.New(apperr.CodeNotFound, "community member not found")
}

func (f *fakeCommunityRepository) FindActiveMemberByUsername(ctx context.Context, communityID communitydomain.CommunityID, username string) (CommunityMember, error) {
	for _, member := range f.activeMemberByUserID {
		if member.Username == username {
			return member, nil
		}
	}
	return CommunityMember{}, apperr.New(apperr.CodeNotFound, "community member not found")
}

func (f *fakeCommunityRepository) FindActiveUserByUsername(ctx context.Context, username string) (CommunityMember, error) {
	if f.activeUserByUsername != nil {
		if user, ok := f.activeUserByUsername[username]; ok {
			return user, nil
		}
	}
	return CommunityMember{}, apperr.New(apperr.CodeNotFound, "user not found")
}

func (f *fakeCommunityRepository) FindActiveUserByID(ctx context.Context, userID userdomain.UserID) (CommunityMember, error) {
	if f.activeUserByID != nil {
		if user, ok := f.activeUserByID[userID.String()]; ok {
			return user, nil
		}
	}
	return CommunityMember{}, apperr.New(apperr.CodeNotFound, "user not found")
}

func (f *fakeCommunityRepository) CountActiveMembers(ctx context.Context, communityID communitydomain.CommunityID) (int, error) {
	return f.activeMemberCount, nil
}

func (f *fakeCommunityRepository) CountActiveModerators(ctx context.Context, communityID communitydomain.CommunityID) (int, error) {
	return f.activeModeratorCount, nil
}

func (f *fakeCommunityRepository) UpsertActiveMemberRole(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID, role communitydomain.MembershipRole, now time.Time) (CommunityMember, error) {
	f.upsertedMemberUserID = userID
	f.upsertedMemberRole = role
	member := CommunityMember{
		UserID:    userID.String(),
		Username:  "member",
		Role:      role.String(),
		Status:    communitydomain.MembershipStatusActive.String(),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if f.activeUserByID != nil {
		if user, ok := f.activeUserByID[userID.String()]; ok {
			member.Username = user.Username
			member.DisplayName = user.DisplayName
			member.AvatarURL = user.AvatarURL
			member.Headline = user.Headline
		}
	}
	return member, nil
}

func (f *fakeCommunityRepository) CreateOwnerTransfer(ctx context.Context, transfer CommunityOwnerTransferRecord) error {
	f.createdOwnerTransfer = &transfer
	return nil
}

func (f *fakeCommunityRepository) FindOwnerTransferForUpdate(ctx context.Context, transferID string) (CommunityOwnerTransferRecord, error) {
	if f.ownerTransfer == nil || f.ownerTransfer.ID != transferID {
		return CommunityOwnerTransferRecord{}, apperr.New(apperr.CodeNotFound, "community owner transfer not found")
	}
	return *f.ownerTransfer, nil
}

func (f *fakeCommunityRepository) AcceptOwnerTransfer(ctx context.Context, transferID string, acceptedAt time.Time) error {
	f.acceptedOwnerTransferID = transferID
	return nil
}

func (f *fakeCommunityRepository) TransferOwner(ctx context.Context, communityID communitydomain.CommunityID, newOwnerID userdomain.UserID, now time.Time) (CommunityOwnerChange, error) {
	f.transferOwnerUserID = newOwnerID
	return CommunityOwnerChange{
		AfterOwner: CommunityMember{UserID: newOwnerID.String(), Role: communitydomain.MembershipRoleOwner.String(), Status: communitydomain.MembershipStatusActive.String(), UpdatedAt: now},
	}, nil
}

func (f *fakeCommunityRepository) ListPostsByCommunityForManagement(ctx context.Context, communityID communitydomain.CommunityID, status *postdomain.PostStatus, limit int, offset int) ([]postdomain.Post, error) {
	f.managePostsCalled = true
	f.managePostsCommunityID = communityID
	f.managePostsStatus = status
	f.managePostsLimit = limit
	f.managePostsOffset = offset
	return f.managePosts, nil
}

func (f *fakeCommunityRepository) ListCommentsByCommunityForManagement(ctx context.Context, communityID communitydomain.CommunityID, status *commentdomain.CommentStatus, limit int, offset int) ([]commentdomain.Comment, error) {
	f.manageCommentsCalled = true
	f.manageCommentsCommunityID = communityID
	f.manageCommentsStatus = status
	f.manageCommentsLimit = limit
	f.manageCommentsOffset = offset
	return f.manageComments, nil
}

func (f *fakeCommunityRepository) ListReportsByCommunityForManagement(ctx context.Context, communityID communitydomain.CommunityID, status moderationdomain.ReportStatus, limit int, offset int) ([]moderationusecase.ContentReportRecord, error) {
	f.manageReportsCalled = true
	f.manageReportsCommunityID = communityID
	f.manageReportsStatus = status
	f.manageReportsLimit = limit
	f.manageReportsOffset = offset
	return f.manageReports, nil
}

func (f *fakeCommunityRepository) ListRules(ctx context.Context, communityID communitydomain.CommunityID) ([]communitydomain.CommunityRule, error) {
	f.listRulesCommunityID = communityID
	return f.rules, nil
}

func (f *fakeCommunityRepository) FindRuleByID(ctx context.Context, id communitydomain.CommunityRuleID) (*communitydomain.CommunityRule, error) {
	if f.ruleByID == nil {
		return nil, apperr.New(apperr.CodeNotFound, "community rule not found")
	}
	if f.ruleByID.ID() != id {
		return nil, apperr.New(apperr.CodeNotFound, "community rule not found")
	}
	return f.ruleByID, nil
}

func (f *fakeCommunityRepository) CreateRule(ctx context.Context, rule communitydomain.CommunityRule) error {
	f.createRuleCalled = true
	f.createdRule = rule
	return nil
}

func (f *fakeCommunityRepository) UpdateRule(ctx context.Context, rule communitydomain.CommunityRule) error {
	f.updateRuleCalled = true
	f.updatedRule = rule
	return nil
}

func (f *fakeCommunityRepository) DeleteRule(ctx context.Context, id communitydomain.CommunityRuleID, communityID communitydomain.CommunityID) error {
	f.deleteRuleCalled = true
	f.deletedRuleID = id
	f.deletedRuleCommunityID = communityID
	return nil
}

func mustSystemCommunity(t *testing.T, rawSlug string, now time.Time) *communitydomain.Community {
	t.Helper()

	community, err := communitydomain.NewSystemCommunity(
		communitydomain.NewGeneratedCommunityID(),
		mustCommunitySlug(t, rawSlug),
		mustCommunityName(t, rawSlug),
		mustCommunityDescription(t, "test community"),
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
		mustCommunityDescription(t, "test community"),
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

func mustCommunityDescription(t *testing.T, raw string) communitydomain.CommunityDescription {
	t.Helper()

	description, err := communitydomain.NewCommunityDescription(raw)
	if err != nil {
		t.Fatalf("NewCommunityDescription(%q) returned error: %v", raw, err)
	}
	return description
}

func mustCommunityRuleBody(t *testing.T, raw string) communitydomain.CommunityRuleBody {
	t.Helper()

	body, err := communitydomain.NewCommunityRuleBody(raw)
	if err != nil {
		t.Fatalf("NewCommunityRuleBody returned error: %v", err)
	}
	return body
}

func mustPost(t *testing.T, communityID communitydomain.CommunityID, authorID userdomain.UserID, rawTitle string, status postdomain.PostStatus, now time.Time) *postdomain.Post {
	t.Helper()

	title, err := postdomain.NewPostTitle(rawTitle)
	if err != nil {
		t.Fatalf("NewPostTitle returned error: %v", err)
	}
	body, err := postdomain.NewPostBody("body for " + rawTitle)
	if err != nil {
		t.Fatalf("NewPostBody returned error: %v", err)
	}
	post, err := postdomain.RehydratePost(postdomain.NewGeneratedPostID(), communityID, authorID, title, body, status, now, now)
	if err != nil {
		t.Fatalf("RehydratePost returned error: %v", err)
	}
	return post
}

func mustCommunityRule(t *testing.T, communityID communitydomain.CommunityID, actorID userdomain.UserID, rawTitle string, rawBody string, position int, now time.Time) *communitydomain.CommunityRule {
	t.Helper()

	title, err := communitydomain.NewCommunityRuleTitle(rawTitle)
	if err != nil {
		t.Fatalf("NewCommunityRuleTitle returned error: %v", err)
	}
	rulePosition, err := communitydomain.NewCommunityRulePosition(position)
	if err != nil {
		t.Fatalf("NewCommunityRulePosition returned error: %v", err)
	}
	rule, err := communitydomain.NewCommunityRule(
		communitydomain.NewGeneratedCommunityRuleID(),
		communityID,
		title,
		mustCommunityRuleBody(t, rawBody),
		rulePosition,
		actorID,
		now,
	)
	if err != nil {
		t.Fatalf("NewCommunityRule returned error: %v", err)
	}
	return rule
}

func hasAppCode(err error, code apperr.Code) bool {
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Code() == code
}

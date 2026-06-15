package moderationusecase

import (
	"context"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestRemovePostCreatesModerationAction(t *testing.T) {
	now := testNow()
	actorID := userdomain.NewGeneratedUserID()
	postID := postdomain.NewGeneratedPostID()
	var saved moderationdomain.ModerationAction
	removals := &fakeRemovalRepository{
		removePostFunc: func(ctx context.Context, action moderationdomain.ModerationAction) error {
			saved = action
			return nil
		},
	}
	uc := newTestRemoveUseCase(removals, &fakeStaffRepository{isStaff: true}, func() time.Time { return now })

	result, err := uc.RemovePost(context.Background(), RemovePostInput{
		PostID:  postID.String(),
		ActorID: actorID,
		Reason:  "policy violation",
	})
	if err != nil {
		t.Fatalf("RemovePost returned error: %v", err)
	}

	if !removals.removePostCalled {
		t.Fatal("expected RemovePostWithAction to be called")
	}
	if result.Action.TargetType != moderationdomain.TargetTypePost.String() || result.Action.PostID != postID.String() {
		t.Fatalf("unexpected remove post result: %#v", result.Action)
	}
	if result.Action.ActorID != actorID.String() || result.Action.Action != moderationdomain.ActionTypeRemove.String() {
		t.Fatalf("unexpected action dto: %#v", result.Action)
	}
	if saved.Reason().String() != "policy violation" {
		t.Fatalf("expected saved reason, got %q", saved.Reason().String())
	}
}

func TestRemoveCommentCreatesModerationAction(t *testing.T) {
	now := testNow()
	actorID := userdomain.NewGeneratedUserID()
	comment := mustComment(t, now)
	removals := &fakeRemovalRepository{}
	uc := newTestRemoveUseCase(removals, &fakeStaffRepository{isStaff: true}, func() time.Time { return now })

	result, err := uc.RemoveComment(context.Background(), RemoveCommentInput{
		CommentID: comment.ID().String(),
		ActorID:   actorID,
		Reason:    "abuse",
	})
	if err != nil {
		t.Fatalf("RemoveComment returned error: %v", err)
	}

	if !removals.removeCommentCalled {
		t.Fatal("expected RemoveCommentWithAction to be called")
	}
	if result.Action.TargetType != moderationdomain.TargetTypeComment.String() || result.Action.CommentID != comment.ID().String() {
		t.Fatalf("unexpected remove comment result: %#v", result.Action)
	}
}

func TestRemoveRejectsMissingActorNonStaffAndInvalidInput(t *testing.T) {
	uc := newTestRemoveUseCase(&fakeRemovalRepository{}, &fakeStaffRepository{isStaff: true}, time.Now)

	_, err := uc.RemovePost(context.Background(), RemovePostInput{
		PostID: postdomain.NewGeneratedPostID().String(),
		Reason: "spam",
	})
	if !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated for missing actor, got %v", err)
	}

	uc = newTestRemoveUseCase(&fakeRemovalRepository{}, &fakeStaffRepository{isStaff: false}, time.Now)
	_, err = uc.RemovePost(context.Background(), RemovePostInput{
		PostID:  postdomain.NewGeneratedPostID().String(),
		ActorID: userdomain.NewGeneratedUserID(),
		Reason:  "spam",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for non staff, got %v", err)
	}

	uc = newTestRemoveUseCase(&fakeRemovalRepository{}, &fakeStaffRepository{isStaff: true}, time.Now)
	_, err = uc.RemovePost(context.Background(), RemovePostInput{
		PostID:  postdomain.NewGeneratedPostID().String(),
		ActorID: userdomain.NewGeneratedUserID(),
		Reason:  " ",
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for blank reason, got %v", err)
	}

	_, err = uc.RemovePost(context.Background(), RemovePostInput{
		PostID:  "not-a-uuid",
		ActorID: userdomain.NewGeneratedUserID(),
		Reason:  "spam",
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid post id, got %v", err)
	}
}

func TestRemovePropagatesRepositoryError(t *testing.T) {
	uc := NewRemoveUseCase(
		&fakeRemovalRepository{
			removePostFunc: func(ctx context.Context, action moderationdomain.ModerationAction) error {
				return apperr.New(apperr.CodeNotFound, "post not found")
			},
		},
		&fakeStaffRepository{isStaff: true},
		nil,
		nil,
		nil,
		nil,
		time.Now,
	)

	_, err := uc.RemovePost(context.Background(), RemovePostInput{
		PostID:  postdomain.NewGeneratedPostID().String(),
		ActorID: userdomain.NewGeneratedUserID(),
		Reason:  "spam",
	})
	if !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found from repository, got %v", err)
	}
}

func TestRemoveCommunityPostAllowsModeratorInSameCommunity(t *testing.T) {
	now := testNow()
	actorID := userdomain.NewGeneratedUserID()
	community := mustModerationCommunity(t, "campus", now)
	title, _ := postdomain.NewPostTitle("Policy")
	body, _ := postdomain.NewPostBody("Body")
	post, err := postdomain.NewPost(postdomain.NewGeneratedPostID(), community.ID(), userdomain.NewGeneratedUserID(), title, body, now)
	if err != nil {
		t.Fatalf("NewPost returned error: %v", err)
	}
	removals := &fakeRemovalRepository{}
	uc := NewRemoveUseCase(
		removals,
		&fakeStaffRepository{},
		&fakeCommunityLookupRepository{community: community},
		&fakeCommunityRoleRepository{roles: map[communitydomain.CommunityID]communitydomain.MembershipRole{community.ID(): communitydomain.MembershipRoleModerator}},
		&fakePostRepository{findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		}},
		&fakeCommentRepository{},
		func() time.Time { return now },
	)

	result, err := uc.RemoveCommunityPost(context.Background(), RemoveCommunityPostInput{
		CommunitySlug: "campus",
		PostID:        post.ID().String(),
		ActorID:       actorID,
		Reason:        "community policy",
	})
	if err != nil {
		t.Fatalf("RemoveCommunityPost returned error: %v", err)
	}
	if !removals.removePostCalled {
		t.Fatal("expected RemovePostWithAction to be called")
	}
	if result.Action.PostID != post.ID().String() || result.Action.ActorID != actorID.String() {
		t.Fatalf("unexpected action: %#v", result.Action)
	}
}

func TestRemoveCommunityPostRejectsNonModerator(t *testing.T) {
	now := testNow()
	actorID := userdomain.NewGeneratedUserID()
	community := mustModerationCommunity(t, "campus", now)
	uc := NewRemoveUseCase(
		&fakeRemovalRepository{},
		&fakeStaffRepository{},
		&fakeCommunityLookupRepository{community: community},
		&fakeCommunityRoleRepository{roles: map[communitydomain.CommunityID]communitydomain.MembershipRole{community.ID(): communitydomain.MembershipRoleMember}},
		&fakePostRepository{},
		&fakeCommentRepository{},
		func() time.Time { return now },
	)

	_, err := uc.RemoveCommunityPost(context.Background(), RemoveCommunityPostInput{
		CommunitySlug: "campus",
		PostID:        postdomain.NewGeneratedPostID().String(),
		ActorID:       actorID,
		Reason:        "community policy",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestRemoveCommunityPostRejectsCrossCommunityTarget(t *testing.T) {
	now := testNow()
	actorID := userdomain.NewGeneratedUserID()
	community := mustModerationCommunity(t, "campus", now)
	otherCommunity := mustModerationCommunity(t, "other-campus", now)
	title, _ := postdomain.NewPostTitle("Policy")
	body, _ := postdomain.NewPostBody("Body")
	post, err := postdomain.NewPost(postdomain.NewGeneratedPostID(), otherCommunity.ID(), userdomain.NewGeneratedUserID(), title, body, now)
	if err != nil {
		t.Fatalf("NewPost returned error: %v", err)
	}
	removals := &fakeRemovalRepository{}
	uc := NewRemoveUseCase(
		removals,
		&fakeStaffRepository{},
		&fakeCommunityLookupRepository{community: community},
		&fakeCommunityRoleRepository{roles: map[communitydomain.CommunityID]communitydomain.MembershipRole{community.ID(): communitydomain.MembershipRoleModerator}},
		&fakePostRepository{findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		}},
		&fakeCommentRepository{},
		func() time.Time { return now },
	)

	_, err = uc.RemoveCommunityPost(context.Background(), RemoveCommunityPostInput{
		CommunitySlug: "campus",
		PostID:        post.ID().String(),
		ActorID:       actorID,
		Reason:        "community policy",
	})
	if !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found, got %v", err)
	}
	if removals.removePostCalled {
		t.Fatal("cross-community target should not be removed")
	}
}

type fakeRemovalRepository struct {
	removePostCalled    bool
	removeCommentCalled bool
	removePostFunc      func(ctx context.Context, action moderationdomain.ModerationAction) error
	removeCommentFunc   func(ctx context.Context, action moderationdomain.ModerationAction) error
}

func newTestRemoveUseCase(removals ContentRemovalRepository, staff PlatformStaffRepository, now func() time.Time) *RemoveUseCase {
	return NewRemoveUseCase(removals, staff, nil, nil, nil, nil, now)
}

func (f *fakeRemovalRepository) RemovePostWithAction(ctx context.Context, action moderationdomain.ModerationAction) error {
	f.removePostCalled = true
	if f.removePostFunc != nil {
		return f.removePostFunc(ctx, action)
	}
	return nil
}

func (f *fakeRemovalRepository) RemoveCommentWithAction(ctx context.Context, action moderationdomain.ModerationAction) error {
	f.removeCommentCalled = true
	if f.removeCommentFunc != nil {
		return f.removeCommentFunc(ctx, action)
	}
	return nil
}

type fakeStaffRepository struct {
	isStaff bool
	err     error
}

type fakeCommunityLookupRepository struct {
	community *communitydomain.Community
}

func (f *fakeCommunityLookupRepository) FindBySlug(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
	if f.community == nil || f.community.Slug() != slug {
		return nil, apperr.New(apperr.CodeNotFound, "community not found")
	}
	return f.community, nil
}

type fakeCommunityRoleRepository struct {
	roles map[communitydomain.CommunityID]communitydomain.MembershipRole
}

func (f *fakeCommunityRoleRepository) FindActiveRolesByUser(ctx context.Context, communityIDs []communitydomain.CommunityID, userID userdomain.UserID) (map[communitydomain.CommunityID]communitydomain.MembershipRole, error) {
	result := make(map[communitydomain.CommunityID]communitydomain.MembershipRole)
	for _, communityID := range communityIDs {
		if role, ok := f.roles[communityID]; ok {
			result[communityID] = role
		}
	}
	return result, nil
}

func mustModerationCommunity(t *testing.T, rawSlug string, now time.Time) *communitydomain.Community {
	t.Helper()
	slug, err := communitydomain.NewCommunitySlug(rawSlug)
	if err != nil {
		t.Fatalf("NewCommunitySlug returned error: %v", err)
	}
	name, err := communitydomain.NewCommunityName(rawSlug)
	if err != nil {
		t.Fatalf("NewCommunityName returned error: %v", err)
	}
	description, err := communitydomain.NewCommunityDescription("community")
	if err != nil {
		t.Fatalf("NewCommunityDescription returned error: %v", err)
	}
	community, err := communitydomain.NewUserCreatedCommunity(communitydomain.NewGeneratedCommunityID(), slug, name, description, userdomain.NewGeneratedUserID(), now)
	if err != nil {
		t.Fatalf("NewUserCreatedCommunity returned error: %v", err)
	}
	return community
}

func (f *fakeStaffRepository) IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.isStaff, nil
}

package moderationusecase

import (
	"context"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
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
	uc := NewRemoveUseCase(removals, &fakeStaffRepository{isStaff: true}, func() time.Time { return now })

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
	uc := NewRemoveUseCase(removals, &fakeStaffRepository{isStaff: true}, func() time.Time { return now })

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
	uc := NewRemoveUseCase(&fakeRemovalRepository{}, &fakeStaffRepository{isStaff: true}, time.Now)

	_, err := uc.RemovePost(context.Background(), RemovePostInput{
		PostID: postdomain.NewGeneratedPostID().String(),
		Reason: "spam",
	})
	if !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated for missing actor, got %v", err)
	}

	uc = NewRemoveUseCase(&fakeRemovalRepository{}, &fakeStaffRepository{isStaff: false}, time.Now)
	_, err = uc.RemovePost(context.Background(), RemovePostInput{
		PostID:  postdomain.NewGeneratedPostID().String(),
		ActorID: userdomain.NewGeneratedUserID(),
		Reason:  "spam",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for non staff, got %v", err)
	}

	uc = NewRemoveUseCase(&fakeRemovalRepository{}, &fakeStaffRepository{isStaff: true}, time.Now)
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

type fakeRemovalRepository struct {
	removePostCalled    bool
	removeCommentCalled bool
	removePostFunc      func(ctx context.Context, action moderationdomain.ModerationAction) error
	removeCommentFunc   func(ctx context.Context, action moderationdomain.ModerationAction) error
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

func (f *fakeStaffRepository) IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.isStaff, nil
}

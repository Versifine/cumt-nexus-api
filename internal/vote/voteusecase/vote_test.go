package voteusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
)

func TestSetPostVoteUpsertsVoteForVisiblePost(t *testing.T) {
	now := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	post := mustPost(t, now)
	userID := userdomain.NewGeneratedUserID()
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			if id != post.ID() {
				t.Fatalf("expected post id %q, got %q", post.ID().String(), id.String())
			}
			return post, nil
		},
	}
	votes := &fakePostVoteRepository{}
	uc := NewPostVoteUseCase(posts, votes, func() time.Time { return now })

	result, err := uc.SetPostVote(context.Background(), SetPostVoteInput{
		PostID: post.ID().String(),
		UserID: userID,
		Value:  1,
	})
	if err != nil {
		t.Fatalf("SetPostVote returned error: %v", err)
	}
	if !votes.upsertCalled {
		t.Fatal("expected vote repository Upsert to be called")
	}
	if votes.upsertedVote.PostID() != post.ID() || votes.upsertedVote.UserID() != userID || votes.upsertedVote.Value() != votedomain.VoteValueUp {
		t.Fatalf("unexpected upserted vote: %#v", votes.upsertedVote)
	}
	if result.Vote.Value != 1 || result.Vote.PostID != post.ID().String() || result.Vote.UserID != userID.String() {
		t.Fatalf("unexpected result vote: %#v", result.Vote)
	}
}

func TestSetPostVoteNotifiesPostAuthorOnFirstUpvote(t *testing.T) {
	now := time.Date(2026, 6, 2, 10, 5, 0, 0, time.UTC)
	post := mustPost(t, now)
	userID := userdomain.NewGeneratedUserID()
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	votes := &fakePostVoteRepository{}
	notifications := &fakeNotificationPublisher{}
	uc := NewPostVoteUseCase(posts, votes, func() time.Time { return now })
	uc.SetNotificationPublisher(notifications)

	_, err := uc.SetPostVote(context.Background(), SetPostVoteInput{
		PostID: post.ID().String(),
		UserID: userID,
		Value:  1,
	})
	if err != nil {
		t.Fatalf("SetPostVote returned error: %v", err)
	}
	if !notifications.postUpvotedCalled {
		t.Fatal("expected post upvote notification")
	}
	if notifications.recipientID != post.AuthorID() || notifications.actorID != userID || notifications.postID != post.ID().String() {
		t.Fatalf("unexpected notification args: %#v", notifications)
	}
}

func TestSetPostVoteDoesNotNotifyRepeatedUpvote(t *testing.T) {
	now := time.Date(2026, 6, 2, 10, 10, 0, 0, time.UTC)
	post := mustPost(t, now)
	userID := userdomain.NewGeneratedUserID()
	previousVote, err := votedomain.NewPostVote(post.ID(), userID, votedomain.VoteValueUp, now.Add(-time.Minute))
	if err != nil {
		t.Fatalf("NewPostVote returned error: %v", err)
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	votes := &fakePostVoteRepository{previousVote: previousVote}
	notifications := &fakeNotificationPublisher{}
	uc := NewPostVoteUseCase(posts, votes, func() time.Time { return now })
	uc.SetNotificationPublisher(notifications)

	_, err = uc.SetPostVote(context.Background(), SetPostVoteInput{
		PostID: post.ID().String(),
		UserID: userID,
		Value:  1,
	})
	if err != nil {
		t.Fatalf("SetPostVote returned error: %v", err)
	}
	if notifications.postUpvotedCalled {
		t.Fatal("did not expect repeated upvote notification")
	}
}

func TestSetPostVoteRejectsInvalidInput(t *testing.T) {
	uc := NewPostVoteUseCase(&fakePostRepository{}, &fakePostVoteRepository{}, time.Now)
	postID := postdomain.NewGeneratedPostID().String()
	userID := userdomain.NewGeneratedUserID()

	if _, err := uc.SetPostVote(context.Background(), SetPostVoteInput{
		PostID: postID,
		UserID: "",
		Value:  1,
	}); !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated for missing user, got %v", err)
	}
	if _, err := uc.SetPostVote(context.Background(), SetPostVoteInput{
		PostID: "not-a-uuid",
		UserID: userID,
		Value:  1,
	}); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid post id, got %v", err)
	}
	if _, err := uc.SetPostVote(context.Background(), SetPostVoteInput{
		PostID: postID,
		UserID: userID,
		Value:  0,
	}); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid vote value, got %v", err)
	}
}

func TestSetPostVotePropagatesPostNotFound(t *testing.T) {
	postID := postdomain.NewGeneratedPostID()
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return nil, apperr.New(apperr.CodeNotFound, "post not found")
		},
	}
	uc := NewPostVoteUseCase(posts, &fakePostVoteRepository{}, time.Now)

	_, err := uc.SetPostVote(context.Background(), SetPostVoteInput{
		PostID: postID.String(),
		UserID: userdomain.NewGeneratedUserID(),
		Value:  -1,
	})
	if !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing post, got %v", err)
	}
}

func TestDeletePostVoteDeletesVoteForVisiblePost(t *testing.T) {
	now := time.Date(2026, 6, 2, 10, 30, 0, 0, time.UTC)
	post := mustPost(t, now)
	userID := userdomain.NewGeneratedUserID()
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	votes := &fakePostVoteRepository{}
	uc := NewPostVoteUseCase(posts, votes, time.Now)

	err := uc.DeletePostVote(context.Background(), DeletePostVoteInput{
		PostID: post.ID().String(),
		UserID: userID,
	})
	if err != nil {
		t.Fatalf("DeletePostVote returned error: %v", err)
	}
	if !votes.deleteCalled {
		t.Fatal("expected vote repository DeleteByPostAndUser to be called")
	}
	if votes.deletedPostID != post.ID() || votes.deletedUserID != userID {
		t.Fatalf("unexpected deleted vote target: post=%q user=%q", votes.deletedPostID.String(), votes.deletedUserID.String())
	}
}

type fakePostRepository struct {
	findVisibleByIDFunc func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error)
}

func (f *fakePostRepository) FindVisibleByID(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
	if f.findVisibleByIDFunc != nil {
		return f.findVisibleByIDFunc(ctx, id)
	}
	return nil, apperr.New(apperr.CodeNotFound, "post not found")
}

type fakePostVoteRepository struct {
	upsertCalled  bool
	deleteCalled  bool
	findCalled    bool
	upsertedVote  votedomain.PostVote
	deletedPostID postdomain.PostID
	deletedUserID userdomain.UserID
	previousVote  *votedomain.PostVote
	findErr       error
	upsertErr     error
	deleteErr     error
}

func (f *fakePostVoteRepository) Upsert(ctx context.Context, vote votedomain.PostVote) error {
	f.upsertCalled = true
	f.upsertedVote = vote
	return f.upsertErr
}

func (f *fakePostVoteRepository) DeleteByPostAndUser(ctx context.Context, postID postdomain.PostID, userID userdomain.UserID) error {
	f.deleteCalled = true
	f.deletedPostID = postID
	f.deletedUserID = userID
	return f.deleteErr
}

func (f *fakePostVoteRepository) FindByPostAndUser(ctx context.Context, postID postdomain.PostID, userID userdomain.UserID) (*votedomain.PostVote, error) {
	f.findCalled = true
	if f.findErr != nil {
		return nil, f.findErr
	}
	if f.previousVote != nil {
		return f.previousVote, nil
	}
	return nil, apperr.New(apperr.CodeNotFound, "post vote not found")
}

func (f *fakePostVoteRepository) FindByPostIDsAndUser(ctx context.Context, postIDs []postdomain.PostID, userID userdomain.UserID) (map[postdomain.PostID]votedomain.VoteValue, error) {
	return map[postdomain.PostID]votedomain.VoteValue{}, nil
}

func (f *fakePostVoteRepository) SummarizeByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID]PostVoteSummary, error) {
	return map[postdomain.PostID]PostVoteSummary{}, nil
}

type fakeNotificationPublisher struct {
	postUpvotedCalled bool
	recipientID       userdomain.UserID
	actorID           userdomain.UserID
	postID            string
}

func (f *fakeNotificationPublisher) NotifyPostUpvoted(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, postID string) error {
	f.postUpvotedCalled = true
	f.recipientID = recipientID
	f.actorID = actorID
	f.postID = postID
	return nil
}

func mustPost(t *testing.T, now time.Time) *postdomain.Post {
	t.Helper()

	title, err := postdomain.NewPostTitle("Vote target")
	if err != nil {
		t.Fatalf("NewPostTitle returned error: %v", err)
	}
	body, err := postdomain.NewPostBody("Vote target body")
	if err != nil {
		t.Fatalf("NewPostBody returned error: %v", err)
	}
	post, err := postdomain.NewPost(
		postdomain.NewGeneratedPostID(),
		communitydomain.NewGeneratedCommunityID(),
		userdomain.NewGeneratedUserID(),
		title,
		body,
		now,
	)
	if err != nil {
		t.Fatalf("NewPost returned error: %v", err)
	}
	return post
}

func hasAppCode(err error, code apperr.Code) bool {
	if err == nil {
		return false
	}

	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Code() == code
}

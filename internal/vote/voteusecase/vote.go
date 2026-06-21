package voteusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/effect/effectusecase"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/progression/progressionusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
)

type PostRepository interface {
	FindVisibleByID(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error)
}

type PostVoteUseCase struct {
	posts         PostRepository
	votes         PostVoteRepository
	notifications NotificationPublisher
	progression   XPRecorder
	points        PointRecorder
	now           func() time.Time
}

func NewPostVoteUseCase(posts PostRepository, votes PostVoteRepository, now func() time.Time) *PostVoteUseCase {
	if now == nil {
		now = time.Now
	}

	return &PostVoteUseCase{
		posts: posts,
		votes: votes,
		now:   now,
	}
}

type NotificationPublisher interface {
	NotifyPostUpvoted(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, postID string) error
}

func (uc *PostVoteUseCase) SetNotificationPublisher(notifications NotificationPublisher) {
	uc.notifications = notifications
}

type XPRecorder interface {
	GrantXP(ctx context.Context, input progressionusecase.GrantXPInput) error
}

func (uc *PostVoteUseCase) SetXPRecorder(progression XPRecorder) {
	uc.progression = progression
}

type PointRecorder interface {
	GrantPoints(ctx context.Context, input effectusecase.GrantPointsInput) error
}

func (uc *PostVoteUseCase) SetPointRecorder(points PointRecorder) {
	uc.points = points
}

type SetPostVoteInput struct {
	PostID string
	UserID userdomain.UserID
	Value  int
}

type DeletePostVoteInput struct {
	PostID string
	UserID userdomain.UserID
}

type SetPostVoteResult struct {
	Vote PostVote
}

type PostVote struct {
	PostID    string
	UserID    string
	Value     int
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (uc *PostVoteUseCase) SetPostVote(ctx context.Context, input SetPostVoteInput) (SetPostVoteResult, error) {
	if strings.TrimSpace(input.UserID.String()) == "" {
		return SetPostVoteResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}

	postID, err := postdomain.NewPostID(input.PostID)
	if err != nil {
		return SetPostVoteResult{}, err
	}
	value, err := votedomain.NewVoteValue(input.Value)
	if err != nil {
		return SetPostVoteResult{}, err
	}

	post, err := uc.posts.FindVisibleByID(ctx, postID)
	if err != nil {
		return SetPostVoteResult{}, fmt.Errorf("find post for vote: %w", err)
	}
	previousVote, err := uc.votes.FindByPostAndUser(ctx, post.ID(), input.UserID)
	if err != nil && !apperr.IsCode(err, apperr.CodeNotFound) {
		return SetPostVoteResult{}, fmt.Errorf("find existing post vote: %w", err)
	}

	now := uc.now().UTC()
	vote, err := votedomain.NewPostVote(post.ID(), input.UserID, value, now)
	if err != nil {
		return SetPostVoteResult{}, err
	}

	if err := uc.votes.Upsert(ctx, *vote); err != nil {
		return SetPostVoteResult{}, fmt.Errorf("upsert post vote: %w", err)
	}
	if uc.shouldNotifyPostUpvote(post.AuthorID(), input.UserID, value, previousVote) {
		_ = uc.notifications.NotifyPostUpvoted(ctx, post.AuthorID(), input.UserID, post.ID().String())
	}
	if post.AuthorID() != input.UserID && value == votedomain.VoteValueUp && (previousVote == nil || previousVote.Value() != votedomain.VoteValueUp) {
		_ = uc.grantXP(ctx, post.AuthorID(), input.UserID, progressionusecase.XPSourcePostUpvote, post.ID().String()+":"+input.UserID.String())
		_ = uc.grantPoints(ctx, post.AuthorID(), input.UserID, effectusecase.PointSourcePostUpvote, post.ID().String()+":"+input.UserID.String())
	}

	return SetPostVoteResult{
		Vote: toPostVoteDTO(*vote),
	}, nil
}

func (uc *PostVoteUseCase) shouldNotifyPostUpvote(postAuthorID userdomain.UserID, voterID userdomain.UserID, value votedomain.VoteValue, previousVote *votedomain.PostVote) bool {
	if uc.notifications == nil || postAuthorID == voterID || value != votedomain.VoteValueUp {
		return false
	}
	if previousVote == nil {
		return true
	}
	return previousVote.Value() != votedomain.VoteValueUp
}

func (uc *PostVoteUseCase) grantXP(ctx context.Context, userID userdomain.UserID, actorID userdomain.UserID, sourceType string, sourceID string) error {
	if uc.progression == nil || strings.TrimSpace(userID.String()) == "" {
		return nil
	}
	return uc.progression.GrantXP(ctx, progressionusecase.GrantXPInput{
		UserID:     userID,
		ActorID:    actorID,
		SourceType: sourceType,
		SourceID:   sourceID,
	})
}

func (uc *PostVoteUseCase) grantPoints(ctx context.Context, userID userdomain.UserID, actorID userdomain.UserID, sourceType string, sourceID string) error {
	if uc.points == nil || strings.TrimSpace(userID.String()) == "" {
		return nil
	}
	return uc.points.GrantPoints(ctx, effectusecase.GrantPointsInput{
		UserID:     userID,
		ActorID:    actorID,
		SourceType: sourceType,
		SourceID:   sourceID,
	})
}

func (uc *PostVoteUseCase) DeletePostVote(ctx context.Context, input DeletePostVoteInput) error {
	if strings.TrimSpace(input.UserID.String()) == "" {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}

	postID, err := postdomain.NewPostID(input.PostID)
	if err != nil {
		return err
	}
	post, err := uc.posts.FindVisibleByID(ctx, postID)
	if err != nil {
		return fmt.Errorf("find post for vote deletion: %w", err)
	}

	if err := uc.votes.DeleteByPostAndUser(ctx, post.ID(), input.UserID); err != nil {
		return fmt.Errorf("delete post vote: %w", err)
	}

	return nil
}

func toPostVoteDTO(vote votedomain.PostVote) PostVote {
	return PostVote{
		PostID:    vote.PostID().String(),
		UserID:    vote.UserID().String(),
		Value:     vote.Value().Int(),
		CreatedAt: vote.CreatedAt(),
		UpdatedAt: vote.UpdatedAt(),
	}
}

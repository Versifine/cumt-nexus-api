package votedomain

import (
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type VoteValue int

const (
	VoteValueDown VoteValue = -1
	VoteValueUp   VoteValue = 1
)

func NewVoteValue(raw int) (VoteValue, error) {
	switch VoteValue(raw) {
	case VoteValueUp:
		return VoteValueUp, nil
	case VoteValueDown:
		return VoteValueDown, nil
	default:
		return 0, apperr.New(apperr.CodeInvalidArgument, "vote value is invalid")
	}
}

func (value VoteValue) Int() int {
	return int(value)
}

type PostVote struct {
	postID    postdomain.PostID
	userID    userdomain.UserID
	value     VoteValue
	createdAt time.Time
	updatedAt time.Time
}

type PostVoteSummary struct {
	PostID        postdomain.PostID
	UpvoteCount   int
	DownvoteCount int
}

func (summary PostVoteSummary) Score() int {
	return summary.UpvoteCount - summary.DownvoteCount
}

func NewPostVote(postID postdomain.PostID, userID userdomain.UserID, value VoteValue, now time.Time) (*PostVote, error) {
	return RehydratePostVote(postID, userID, value, now, now)
}

func RehydratePostVote(postID postdomain.PostID, userID userdomain.UserID, value VoteValue, createdAt time.Time, updatedAt time.Time) (*PostVote, error) {
	if strings.TrimSpace(postID.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "vote post id is required")
	}
	if strings.TrimSpace(userID.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "vote user id is required")
	}
	if _, err := NewVoteValue(value.Int()); err != nil {
		return nil, err
	}
	if createdAt.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "vote created time can't be zero")
	}
	if updatedAt.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "vote updated time can't be zero")
	}
	if updatedAt.Before(createdAt) {
		return nil, apperr.New(apperr.CodeInvalidArgument, "vote updated time can't be before created time")
	}

	return &PostVote{
		postID:    postID,
		userID:    userID,
		value:     value,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}, nil
}

func (vote *PostVote) PostID() postdomain.PostID {
	return vote.postID
}

func (vote *PostVote) UserID() userdomain.UserID {
	return vote.userID
}

func (vote *PostVote) Value() VoteValue {
	return vote.value
}

func (vote *PostVote) CreatedAt() time.Time {
	return vote.createdAt
}

func (vote *PostVote) UpdatedAt() time.Time {
	return vote.updatedAt
}

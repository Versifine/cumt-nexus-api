package votedomain

import (
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

func TestVoteValue(t *testing.T) {
	assertVoteValue(t, 1, VoteValueUp)
	assertVoteValue(t, -1, VoteValueDown)

	if _, err := NewVoteValue(0); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for zero vote value, got %v", err)
	}
	if _, err := NewVoteValue(2); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid vote value, got %v", err)
	}
}

func TestNewPostVoteCreatesVote(t *testing.T) {
	now := time.Now().UTC()
	vote, err := NewPostVote(mustPostID(t), mustUserID(t), VoteValueUp, now)
	if err != nil {
		t.Fatalf("NewPostVote returned error: %v", err)
	}
	if vote.Value() != VoteValueUp {
		t.Fatalf("expected upvote, got %d", vote.Value().Int())
	}
	if !vote.CreatedAt().Equal(now) || !vote.UpdatedAt().Equal(now) {
		t.Fatalf("expected timestamps %s, got created=%s updated=%s", now, vote.CreatedAt(), vote.UpdatedAt())
	}
}

func TestPostVoteValidation(t *testing.T) {
	now := time.Now().UTC()
	if _, err := NewPostVote("", mustUserID(t), VoteValueUp, now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for missing post id, got %v", err)
	}
	if _, err := NewPostVote(mustPostID(t), "", VoteValueUp, now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for missing user id, got %v", err)
	}
	if _, err := NewPostVote(mustPostID(t), mustUserID(t), VoteValue(0), now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid vote value, got %v", err)
	}
	if _, err := NewPostVote(mustPostID(t), mustUserID(t), VoteValueUp, time.Time{}); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for zero created time, got %v", err)
	}
	if _, err := RehydratePostVote(mustPostID(t), mustUserID(t), VoteValueUp, now, now.Add(-time.Second)); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for updated_at before created_at, got %v", err)
	}
}

func assertVoteValue(t *testing.T, raw int, want VoteValue) {
	t.Helper()

	got, err := NewVoteValue(raw)
	if err != nil {
		t.Fatalf("NewVoteValue(%d) returned error: %v", raw, err)
	}
	if got != want {
		t.Fatalf("expected vote value %d, got %d", want.Int(), got.Int())
	}
}

func mustPostID(t *testing.T) postdomain.PostID {
	t.Helper()

	id, err := postdomain.NewPostID(uuid.NewString())
	if err != nil {
		t.Fatalf("NewPostID returned error: %v", err)
	}
	return id
}

func mustUserID(t *testing.T) userdomain.UserID {
	t.Helper()

	id, err := userdomain.NewUserID(uuid.NewString())
	if err != nil {
		t.Fatalf("NewUserID returned error: %v", err)
	}
	return id
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

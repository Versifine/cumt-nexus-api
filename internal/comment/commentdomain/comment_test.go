package commentdomain

import (
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

func TestCommentValues(t *testing.T) {
	if _, err := NewCommentID(uuid.NewString()); err != nil {
		t.Fatalf("NewCommentID returned error: %v", err)
	}
	if _, err := NewCommentID("not-a-uuid"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid comment id, got %v", err)
	}

	body, err := NewCommentBody(" Body ")
	if err != nil {
		t.Fatalf("NewCommentBody returned error: %v", err)
	}
	if body.String() != "Body" {
		t.Fatalf("expected trimmed body, got %q", body.String())
	}
	if _, err := NewCommentBody(" "); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for blank body, got %v", err)
	}

	assertCommentStatus(t, "visible", CommentStatusVisible)
	assertCommentStatus(t, "removed", CommentStatusRemoved)
	assertCommentStatus(t, "deleted", CommentStatusDeleted)
	assertCommentStatus(t, "locked", CommentStatusLocked)
	assertCommentStatus(t, "hidden", CommentStatusHidden)
	if _, err := NewCommentStatus("draft"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid status, got %v", err)
	}
}

func TestNewCommentDefaultsToVisible(t *testing.T) {
	now := time.Now().UTC()
	parentID := NewGeneratedCommentID()
	comment, err := NewComment(
		NewGeneratedCommentID(),
		mustPostID(t),
		userdomain.NewGeneratedUserID(),
		&parentID,
		mustCommentBody(t, "Body"),
		now,
	)
	if err != nil {
		t.Fatalf("NewComment returned error: %v", err)
	}
	if comment.Status() != CommentStatusVisible {
		t.Fatalf("expected visible status, got %q", comment.Status().String())
	}
	if gotParentID, ok := comment.ParentID(); !ok || gotParentID != parentID {
		t.Fatalf("expected parent %q, got %q present=%t", parentID.String(), gotParentID.String(), ok)
	}
}

func TestCommentValidation(t *testing.T) {
	now := time.Now().UTC()
	commentID := NewGeneratedCommentID()

	if _, err := NewComment("", mustPostID(t), userdomain.NewGeneratedUserID(), nil, mustCommentBody(t, "Body"), now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for missing comment id, got %v", err)
	}
	if _, err := NewComment(NewGeneratedCommentID(), "", userdomain.NewGeneratedUserID(), nil, mustCommentBody(t, "Body"), now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for missing post id, got %v", err)
	}
	if _, err := NewComment(NewGeneratedCommentID(), mustPostID(t), "", nil, mustCommentBody(t, "Body"), now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for missing author id, got %v", err)
	}
	if _, err := NewComment(commentID, mustPostID(t), userdomain.NewGeneratedUserID(), &commentID, mustCommentBody(t, "Body"), now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for parent equal self, got %v", err)
	}
	if _, err := NewComment(NewGeneratedCommentID(), mustPostID(t), userdomain.NewGeneratedUserID(), nil, mustCommentBody(t, "Body"), time.Time{}); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for zero created time, got %v", err)
	}
	if _, err := RehydrateComment(NewGeneratedCommentID(), mustPostID(t), userdomain.NewGeneratedUserID(), nil, mustCommentBody(t, "Body"), CommentStatusVisible, now, now.Add(-time.Second)); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for updated_at before created_at, got %v", err)
	}
}

func TestCommentEditAndDelete(t *testing.T) {
	now := time.Now().UTC()
	comment, err := NewComment(
		NewGeneratedCommentID(),
		mustPostID(t),
		userdomain.NewGeneratedUserID(),
		nil,
		mustCommentBody(t, "Body"),
		now,
	)
	if err != nil {
		t.Fatalf("NewComment returned error: %v", err)
	}

	editedAt := now.Add(time.Minute)
	if err := comment.EditBody(mustCommentBody(t, "Updated body"), editedAt); err != nil {
		t.Fatalf("EditBody returned error: %v", err)
	}
	if comment.Body().String() != "Updated body" {
		t.Fatalf("comment was not edited: body=%q", comment.Body().String())
	}
	if !comment.UpdatedAt().Equal(editedAt) {
		t.Fatalf("expected updated_at %s, got %s", editedAt, comment.UpdatedAt())
	}

	deletedAt := editedAt.Add(time.Minute)
	if err := comment.MarkDeleted(deletedAt); err != nil {
		t.Fatalf("MarkDeleted returned error: %v", err)
	}
	if comment.Status() != CommentStatusDeleted {
		t.Fatalf("expected deleted status, got %q", comment.Status().String())
	}
	if !comment.UpdatedAt().Equal(deletedAt) {
		t.Fatalf("expected deleted updated_at %s, got %s", deletedAt, comment.UpdatedAt())
	}
	if err := comment.EditBody(mustCommentBody(t, "Again"), deletedAt.Add(time.Minute)); !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict editing deleted comment, got %v", err)
	}
}

func assertCommentStatus(t *testing.T, raw string, want CommentStatus) {
	t.Helper()

	got, err := NewCommentStatus(raw)
	if err != nil {
		t.Fatalf("NewCommentStatus(%q) returned error: %v", raw, err)
	}
	if got != want {
		t.Fatalf("expected status %q, got %q", want.String(), got.String())
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

func mustCommentBody(t *testing.T, raw string) CommentBody {
	t.Helper()

	body, err := NewCommentBody(raw)
	if err != nil {
		t.Fatalf("NewCommentBody returned error: %v", err)
	}
	return body
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

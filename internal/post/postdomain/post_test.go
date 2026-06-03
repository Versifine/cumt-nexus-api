package postdomain

import (
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

func TestPostValues(t *testing.T) {
	if _, err := NewPostID(uuid.NewString()); err != nil {
		t.Fatalf("NewPostID returned error: %v", err)
	}
	if _, err := NewPostID("not-a-uuid"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid post id, got %v", err)
	}

	title, err := NewPostTitle(" Hello ")
	if err != nil {
		t.Fatalf("NewPostTitle returned error: %v", err)
	}
	if title.String() != "Hello" {
		t.Fatalf("expected trimmed title, got %q", title.String())
	}
	if _, err := NewPostTitle(" "); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for blank title, got %v", err)
	}

	body, err := NewPostBody(" Body ")
	if err != nil {
		t.Fatalf("NewPostBody returned error: %v", err)
	}
	if body.String() != "Body" {
		t.Fatalf("expected trimmed body, got %q", body.String())
	}
	if _, err := NewPostBody(" "); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for blank body, got %v", err)
	}

	assertPostStatus(t, "visible", PostStatusVisible)
	assertPostStatus(t, "removed", PostStatusRemoved)
	assertPostStatus(t, "deleted", PostStatusDeleted)
	assertPostStatus(t, "locked", PostStatusLocked)
	assertPostStatus(t, "hidden", PostStatusHidden)
	if _, err := NewPostStatus("draft"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid status, got %v", err)
	}
}

func TestNewPostDefaultsToVisible(t *testing.T) {
	now := time.Now().UTC()
	post, err := NewPost(
		mustPostID(t),
		mustCommunityID(t),
		mustUserID(t),
		mustPostTitle(t, "Hello"),
		mustPostBody(t, "Body"),
		now,
	)
	if err != nil {
		t.Fatalf("NewPost returned error: %v", err)
	}
	if post.Status() != PostStatusVisible {
		t.Fatalf("expected visible status, got %q", post.Status().String())
	}
	if !post.CreatedAt().Equal(now) || !post.UpdatedAt().Equal(now) {
		t.Fatalf("expected timestamps %s, got created=%s updated=%s", now, post.CreatedAt(), post.UpdatedAt())
	}
}

func TestPostValidation(t *testing.T) {
	now := time.Now().UTC()
	if _, err := NewPost("", mustCommunityID(t), mustUserID(t), mustPostTitle(t, "Hello"), mustPostBody(t, "Body"), now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for missing post id, got %v", err)
	}
	if _, err := NewPost(mustPostID(t), "", mustUserID(t), mustPostTitle(t, "Hello"), mustPostBody(t, "Body"), now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for missing community id, got %v", err)
	}
	if _, err := NewPost(mustPostID(t), mustCommunityID(t), "", mustPostTitle(t, "Hello"), mustPostBody(t, "Body"), now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for missing author id, got %v", err)
	}
	if _, err := NewPost(mustPostID(t), mustCommunityID(t), mustUserID(t), mustPostTitle(t, "Hello"), mustPostBody(t, "Body"), time.Time{}); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for zero created time, got %v", err)
	}
	if _, err := RehydratePost(mustPostID(t), mustCommunityID(t), mustUserID(t), mustPostTitle(t, "Hello"), mustPostBody(t, "Body"), PostStatusVisible, now, now.Add(-time.Second)); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for updated_at before created_at, got %v", err)
	}
}

func TestPostEditAndDelete(t *testing.T) {
	now := time.Now().UTC()
	post, err := NewPost(
		mustPostID(t),
		mustCommunityID(t),
		mustUserID(t),
		mustPostTitle(t, "Hello"),
		mustPostBody(t, "Body"),
		now,
	)
	if err != nil {
		t.Fatalf("NewPost returned error: %v", err)
	}

	editedAt := now.Add(time.Minute)
	if err := post.Edit(mustPostTitle(t, "Updated"), mustPostBody(t, "Updated body"), editedAt); err != nil {
		t.Fatalf("Edit returned error: %v", err)
	}
	if post.Title().String() != "Updated" || post.Body().String() != "Updated body" {
		t.Fatalf("post was not edited: title=%q body=%q", post.Title().String(), post.Body().String())
	}
	if !post.UpdatedAt().Equal(editedAt) {
		t.Fatalf("expected updated_at %s, got %s", editedAt, post.UpdatedAt())
	}

	deletedAt := editedAt.Add(time.Minute)
	if err := post.MarkDeleted(deletedAt); err != nil {
		t.Fatalf("MarkDeleted returned error: %v", err)
	}
	if post.Status() != PostStatusDeleted {
		t.Fatalf("expected deleted status, got %q", post.Status().String())
	}
	if !post.UpdatedAt().Equal(deletedAt) {
		t.Fatalf("expected deleted updated_at %s, got %s", deletedAt, post.UpdatedAt())
	}
	if err := post.Edit(mustPostTitle(t, "Again"), mustPostBody(t, "Again body"), deletedAt.Add(time.Minute)); !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict editing deleted post, got %v", err)
	}
}

func assertPostStatus(t *testing.T, raw string, want PostStatus) {
	t.Helper()

	got, err := NewPostStatus(raw)
	if err != nil {
		t.Fatalf("NewPostStatus(%q) returned error: %v", raw, err)
	}
	if got != want {
		t.Fatalf("expected status %q, got %q", want.String(), got.String())
	}
}

func mustPostID(t *testing.T) PostID {
	t.Helper()

	id, err := NewPostID(uuid.NewString())
	if err != nil {
		t.Fatalf("NewPostID returned error: %v", err)
	}
	return id
}

func mustCommunityID(t *testing.T) communitydomain.CommunityID {
	t.Helper()

	id, err := communitydomain.NewCommunityID(uuid.NewString())
	if err != nil {
		t.Fatalf("NewCommunityID returned error: %v", err)
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

func mustPostTitle(t *testing.T, raw string) PostTitle {
	t.Helper()

	title, err := NewPostTitle(raw)
	if err != nil {
		t.Fatalf("NewPostTitle returned error: %v", err)
	}
	return title
}

func mustPostBody(t *testing.T, raw string) PostBody {
	t.Helper()

	body, err := NewPostBody(raw)
	if err != nil {
		t.Fatalf("NewPostBody returned error: %v", err)
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

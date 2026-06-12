package userusecase

import (
	"context"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestGetPublicUserReturnsStoredProfileFields(t *testing.T) {
	now := time.Date(2026, 6, 11, 9, 0, 0, 0, time.UTC)
	user := newCurrentUserTestUser(t, "alice", "active", now)
	if err := user.UpdateProfile("Alice", "https://example.com/avatar.jpg", "https://example.com/banner.jpg", "Backend builder", "Go + PostgreSQL", now.Add(time.Minute)); err != nil {
		t.Fatalf("seed UpdateProfile returned error: %v", err)
	}

	repo := &fakePublicUserRepository{
		user:         user,
		postCount:    3,
		commentCount: 4,
	}
	uc := NewPublicUserUseCase(repo)

	result, err := uc.GetPublicUser(context.Background(), GetPublicUserInput{Username: "alice"})
	if err != nil {
		t.Fatalf("GetPublicUser returned error: %v", err)
	}
	if result.User.DisplayName != "Alice" {
		t.Fatalf("expected display name Alice, got %q", result.User.DisplayName)
	}
	if result.User.AvatarURL != "https://example.com/avatar.jpg" {
		t.Fatalf("expected avatar url, got %q", result.User.AvatarURL)
	}
	if result.User.BannerURL != "https://example.com/banner.jpg" {
		t.Fatalf("expected banner url, got %q", result.User.BannerURL)
	}
	if result.User.Headline != "Backend builder" {
		t.Fatalf("expected headline, got %q", result.User.Headline)
	}
	if result.User.Bio != "Go + PostgreSQL" {
		t.Fatalf("expected bio, got %q", result.User.Bio)
	}
	if result.User.Stats.PostCount != 3 || result.User.Stats.CommentCount != 4 {
		t.Fatalf("unexpected stats: %#v", result.User.Stats)
	}
}

func TestGetPublicUserFallsBackToUsernameWhenDisplayNameIsEmpty(t *testing.T) {
	now := time.Now().UTC()
	user := newCurrentUserTestUser(t, "alice", "active", now)
	repo := &fakePublicUserRepository{user: user}
	uc := NewPublicUserUseCase(repo)

	result, err := uc.GetPublicUser(context.Background(), GetPublicUserInput{Username: "alice"})
	if err != nil {
		t.Fatalf("GetPublicUser returned error: %v", err)
	}
	if result.User.DisplayName != "alice" {
		t.Fatalf("expected username fallback, got %q", result.User.DisplayName)
	}
}

type fakePublicUserRepository struct {
	user         *userdomain.User
	postCount    int
	commentCount int
}

func (f *fakePublicUserRepository) FindByUsername(ctx context.Context, username userdomain.Username) (*userdomain.User, error) {
	return f.user, nil
}

func (f *fakePublicUserRepository) CountVisiblePostsByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID) (int, error) {
	return f.postCount, nil
}

func (f *fakePublicUserRepository) CountVisibleCommentsByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID) (int, error) {
	return f.commentCount, nil
}

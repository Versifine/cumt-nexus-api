package userusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestUpdateProfileSuccess(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	user := newCurrentUserTestUser(t, "alice", "active", now.Add(-time.Hour))
	repo := &fakeProfileRepository{
		user:         user,
		postCount:    7,
		commentCount: 9,
		platformAccess: PlatformAccess{
			IsPlatformStaff: true,
			PlatformRole:    "admin",
		},
	}
	uc := NewUpdateProfileUseCase(repo, func() time.Time { return now })

	displayName := "Alice"
	avatarURL := "https://example.com/avatar.jpg"
	bannerURL := "https://example.com/banner.jpg"
	headline := "Building the backend"
	bio := "Go + PostgreSQL"
	result, err := uc.UpdateProfile(context.Background(), UpdateProfileInput{
		UserID:      user.ID(),
		DisplayName: &displayName,
		AvatarURL:   &avatarURL,
		BannerURL:   &bannerURL,
		Headline:    &headline,
		Bio:         &bio,
	})
	if err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}

	if !repo.updateCalled {
		t.Fatal("expected UpdateProfile repository to be called")
	}
	if result.User.DisplayName != "Alice" {
		t.Fatalf("expected display name Alice, got %q", result.User.DisplayName)
	}
	if result.User.AvatarURL != avatarURL {
		t.Fatalf("expected avatar url %q, got %q", avatarURL, result.User.AvatarURL)
	}
	if result.User.BannerURL != bannerURL {
		t.Fatalf("expected banner url %q, got %q", bannerURL, result.User.BannerURL)
	}
	if result.User.Headline != headline {
		t.Fatalf("expected headline %q, got %q", headline, result.User.Headline)
	}
	if result.User.Bio != bio {
		t.Fatalf("expected bio %q, got %q", bio, result.User.Bio)
	}
	if result.User.Stats.PostCount != 7 || result.User.Stats.CommentCount != 9 {
		t.Fatalf("unexpected stats: %#v", result.User.Stats)
	}
	if !result.User.IsPlatformStaff || result.User.PlatformRole != "admin" {
		t.Fatalf("unexpected platform access: staff=%v role=%q", result.User.IsPlatformStaff, result.User.PlatformRole)
	}
	if repo.updatedUser.DisplayName().String() != "Alice" {
		t.Fatalf("expected updated user display name, got %q", repo.updatedUser.DisplayName().String())
	}
}

func TestUpdateProfileSupportsClearingFields(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	user := newCurrentUserTestUser(t, "alice", "active", now.Add(-time.Hour))
	if err := user.UpdateProfile("Alice", "https://example.com/avatar.jpg", "https://example.com/banner.jpg", "Hello", "Bio", now.Add(-time.Minute)); err != nil {
		t.Fatalf("seed UpdateProfile returned error: %v", err)
	}
	repo := &fakeProfileRepository{user: user}
	uc := NewUpdateProfileUseCase(repo, func() time.Time { return now })

	empty := ""
	result, err := uc.UpdateProfile(context.Background(), UpdateProfileInput{
		UserID:      user.ID(),
		DisplayName: &empty,
		AvatarURL:   &empty,
		BannerURL:   &empty,
		Headline:    &empty,
		Bio:         &empty,
	})
	if err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}
	if result.User.DisplayName != "alice" {
		t.Fatalf("expected username fallback after clearing display name, got %q", result.User.DisplayName)
	}
	if result.User.AvatarURL != "" || result.User.BannerURL != "" || result.User.Headline != "" || result.User.Bio != "" {
		t.Fatalf("expected cleared profile fields, got %#v", result.User)
	}
}

func TestUpdateProfileRejectsMissingPayload(t *testing.T) {
	user := newCurrentUserTestUser(t, "alice", "active", time.Now().UTC())
	repo := &fakeProfileRepository{user: user}
	uc := NewUpdateProfileUseCase(repo, time.Now)

	_, err := uc.UpdateProfile(context.Background(), UpdateProfileInput{
		UserID: user.ID(),
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument, got %v", err)
	}
	if repo.updateCalled {
		t.Fatal("UpdateProfile should not be called for empty payload")
	}
}

func TestUpdateProfileRejectsInvalidAvatarURL(t *testing.T) {
	user := newCurrentUserTestUser(t, "alice", "active", time.Now().UTC())
	repo := &fakeProfileRepository{user: user}
	uc := NewUpdateProfileUseCase(repo, time.Now)

	avatarURL := "ftp://example.com/avatar.jpg"
	_, err := uc.UpdateProfile(context.Background(), UpdateProfileInput{
		UserID:    user.ID(),
		AvatarURL: &avatarURL,
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument, got %v", err)
	}
}

func TestUpdateProfileRejectsInvalidBannerURL(t *testing.T) {
	user := newCurrentUserTestUser(t, "alice", "active", time.Now().UTC())
	repo := &fakeProfileRepository{user: user}
	uc := NewUpdateProfileUseCase(repo, time.Now)

	bannerURL := "ftp://example.com/banner.jpg"
	_, err := uc.UpdateProfile(context.Background(), UpdateProfileInput{
		UserID:    user.ID(),
		BannerURL: &bannerURL,
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument, got %v", err)
	}
}

func TestUpdateProfileNotFoundReturnsUnauthenticated(t *testing.T) {
	repo := &fakeProfileRepository{
		findErr: apperr.New(apperr.CodeNotFound, "user not found"),
	}
	uc := NewUpdateProfileUseCase(repo, time.Now)
	displayName := "Alice"

	_, err := uc.UpdateProfile(context.Background(), UpdateProfileInput{
		UserID:      userdomain.NewGeneratedUserID(),
		DisplayName: &displayName,
	})
	if !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}

func TestUpdateProfileRepositoryError(t *testing.T) {
	user := newCurrentUserTestUser(t, "alice", "active", time.Now().UTC())
	updateErr := errors.New("write failed")
	repo := &fakeProfileRepository{
		user:      user,
		updateErr: updateErr,
	}
	uc := NewUpdateProfileUseCase(repo, time.Now)
	displayName := "Alice"

	_, err := uc.UpdateProfile(context.Background(), UpdateProfileInput{
		UserID:      user.ID(),
		DisplayName: &displayName,
	})
	if !errors.Is(err, updateErr) {
		t.Fatalf("expected update error to be wrapped, got %v", err)
	}
}

type fakeProfileRepository struct {
	findCalled     bool
	updateCalled   bool
	user           *userdomain.User
	updatedUser    userdomain.User
	postCount      int
	commentCount   int
	followerCount  int
	followingCount int
	findErr        error
	updateErr      error
	platformAccess PlatformAccess
}

func (f *fakeProfileRepository) FindByID(ctx context.Context, id userdomain.UserID) (*userdomain.User, error) {
	f.findCalled = true
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.user, nil
}

func (f *fakeProfileRepository) UpdateProfile(ctx context.Context, user userdomain.User) error {
	f.updateCalled = true
	f.updatedUser = user
	return f.updateErr
}

func (f *fakeProfileRepository) CountVisiblePostsByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID) (int, error) {
	return f.postCount, nil
}

func (f *fakeProfileRepository) CountVisibleCommentsByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID) (int, error) {
	return f.commentCount, nil
}

func (f *fakeProfileRepository) CountFollowers(ctx context.Context, userID userdomain.UserID) (int, error) {
	return f.followerCount, nil
}

func (f *fakeProfileRepository) CountFollowing(ctx context.Context, userID userdomain.UserID) (int, error) {
	return f.followingCount, nil
}

func (f *fakeProfileRepository) FindPlatformAccess(ctx context.Context, userID userdomain.UserID) (PlatformAccess, error) {
	return f.platformAccess, nil
}

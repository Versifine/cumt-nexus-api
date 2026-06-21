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
		user:           user,
		postCount:      3,
		commentCount:   4,
		followerCount:  5,
		followingCount: 6,
		platformAccess: PlatformAccess{
			IsPlatformStaff: true,
			PlatformRole:    "staff",
		},
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
	if result.User.Stats.PostCount != 3 || result.User.Stats.CommentCount != 4 || result.User.Stats.FollowerCount != 5 || result.User.Stats.FollowingCount != 6 {
		t.Fatalf("unexpected stats: %#v", result.User.Stats)
	}
	if !result.User.IsPlatformStaff || result.User.PlatformRole != "staff" {
		t.Fatalf("unexpected platform access: staff=%v role=%q", result.User.IsPlatformStaff, result.User.PlatformRole)
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

func TestGetPublicUserIncludesViewerFollowState(t *testing.T) {
	now := time.Now().UTC()
	user := newCurrentUserTestUser(t, "alice", "active", now)
	viewer := newCurrentUserTestUser(t, "bob", "active", now)
	repo := &fakePublicUserRepository{user: user, isFollowing: true}
	uc := NewPublicUserUseCase(repo)

	result, err := uc.GetPublicUser(context.Background(), GetPublicUserInput{Username: "alice", ViewerID: viewer.ID()})
	if err != nil {
		t.Fatalf("GetPublicUser returned error: %v", err)
	}
	if !result.User.ViewerIsFollowing {
		t.Fatal("expected viewer_is_following=true")
	}
}

func TestFollowUserRejectsSelf(t *testing.T) {
	now := time.Now().UTC()
	user := newCurrentUserTestUser(t, "alice", "active", now)
	repo := &fakePublicUserRepository{user: user}
	uc := NewPublicUserUseCase(repo)

	if _, err := uc.FollowUser(context.Background(), FollowUserInput{Username: "alice", UserID: user.ID()}); err == nil {
		t.Fatal("expected self-follow to fail")
	}
}

func TestListFollowedUsersReturnsPagination(t *testing.T) {
	now := time.Now().UTC()
	viewer := newCurrentUserTestUser(t, "alice", "active", now)
	target := newCurrentUserTestUser(t, "bob", "active", now)
	extra := newCurrentUserTestUser(t, "carol", "active", now)
	repo := &fakePublicUserRepository{
		user:          target,
		followedUsers: []userdomain.User{*target, *extra},
		isFollowing:   true,
	}
	uc := NewPublicUserUseCase(repo)

	result, err := uc.ListFollowedUsers(context.Background(), ListFollowedUsersInput{UserID: viewer.ID(), Limit: 1})
	if err != nil {
		t.Fatalf("ListFollowedUsers returned error: %v", err)
	}
	if len(result.Users) != 1 || !result.HasMore || result.NextOffset != 1 {
		t.Fatalf("unexpected page: %#v", result)
	}
	if !result.Users[0].ViewerIsFollowing {
		t.Fatal("expected followed list items to be marked as followed")
	}
}

type fakePublicUserRepository struct {
	user           *userdomain.User
	postCount      int
	commentCount   int
	followerCount  int
	followingCount int
	isFollowing    bool
	followedUsers  []userdomain.User
	platformAccess PlatformAccess
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

func (f *fakePublicUserRepository) CountFollowers(ctx context.Context, userID userdomain.UserID) (int, error) {
	return f.followerCount, nil
}

func (f *fakePublicUserRepository) CountFollowing(ctx context.Context, userID userdomain.UserID) (int, error) {
	return f.followingCount, nil
}

func (f *fakePublicUserRepository) FindPlatformAccess(ctx context.Context, userID userdomain.UserID) (PlatformAccess, error) {
	return f.platformAccess, nil
}

func (f *fakePublicUserRepository) IsFollowing(ctx context.Context, followerID userdomain.UserID, followingID userdomain.UserID) (bool, error) {
	return f.isFollowing, nil
}

func (f *fakePublicUserRepository) FollowUser(ctx context.Context, followerID userdomain.UserID, followingID userdomain.UserID, now time.Time) error {
	return nil
}

func (f *fakePublicUserRepository) DeleteUserFollow(ctx context.Context, followerID userdomain.UserID, followingID userdomain.UserID) error {
	return nil
}

func (f *fakePublicUserRepository) ListFollowedActiveUsers(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]userdomain.User, error) {
	if offset >= len(f.followedUsers) {
		return nil, nil
	}
	end := offset + limit
	if end > len(f.followedUsers) {
		end = len(f.followedUsers)
	}
	return f.followedUsers[offset:end], nil
}

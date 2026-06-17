package userusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestGetCurrentUserSuccess(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 30, 0, 0, time.UTC)
	user := newCurrentUserTestUser(t, "alice", "active", now)
	repo := &fakeCurrentUserRepository{
		user:            user,
		isPlatformStaff: true,
		platformRole:    "owner",
	}
	uc := NewCurrentUserUseCase(repo, repo)

	result, err := uc.GetCurrentUser(context.Background(), CurrentUserInput{
		UserID: user.ID(),
	})
	if err != nil {
		t.Fatalf("GetCurrentUser returned error: %v", err)
	}

	if !repo.findByIDCalled {
		t.Fatal("expected FindByID to be called")
	}
	if repo.findByID != user.ID() {
		t.Fatalf("expected FindByID user id %q, got %q", user.ID().String(), repo.findByID.String())
	}
	if result.User.ID != user.ID().String() {
		t.Fatalf("expected user id %q, got %q", user.ID().String(), result.User.ID)
	}
	if result.User.Username != "alice" {
		t.Fatalf("expected username %q, got %q", "alice", result.User.Username)
	}
	if result.User.Status != "active" {
		t.Fatalf("expected status %q, got %q", "active", result.User.Status)
	}
	if !result.User.IsPlatformStaff {
		t.Fatal("expected current user to be platform staff")
	}
	if result.User.PlatformRole != "owner" {
		t.Fatalf("expected platform role %q, got %q", "owner", result.User.PlatformRole)
	}
	if !result.User.CreatedAt.Equal(now) {
		t.Fatalf("expected created_at %s, got %s", now, result.User.CreatedAt)
	}
}

func TestGetCurrentUserRejectsMissingUserID(t *testing.T) {
	repo := &fakeCurrentUserRepository{}
	uc := NewCurrentUserUseCase(repo, repo)

	_, err := uc.GetCurrentUser(context.Background(), CurrentUserInput{})
	if !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
	if repo.findByIDCalled {
		t.Fatal("FindByID should not be called for missing user id")
	}
	if repo.findPlatformAccessCalled {
		t.Fatal("FindPlatformAccess should not be called for missing user id")
	}
}

func TestGetCurrentUserNotFoundReturnsUnauthenticated(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	repo := &fakeCurrentUserRepository{
		err: apperr.New(apperr.CodeNotFound, "user not found"),
	}
	uc := NewCurrentUserUseCase(repo, repo)

	_, err := uc.GetCurrentUser(context.Background(), CurrentUserInput{
		UserID: userID,
	})
	if !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
	if !repo.findByIDCalled {
		t.Fatal("expected FindByID to be called")
	}
}

func TestGetCurrentUserDisabledReturnsForbidden(t *testing.T) {
	user := newCurrentUserTestUser(t, "alice", "disabled", time.Now().UTC())
	repo := &fakeCurrentUserRepository{
		user: user,
	}
	uc := NewCurrentUserUseCase(repo, repo)

	_, err := uc.GetCurrentUser(context.Background(), CurrentUserInput{
		UserID: user.ID(),
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if repo.findPlatformAccessCalled {
		t.Fatal("FindPlatformAccess should not be called for disabled user")
	}
}

func TestGetCurrentUserRepositoryError(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	repo := &fakeCurrentUserRepository{
		err: errors.New("database failed"),
	}
	uc := NewCurrentUserUseCase(repo, repo)

	_, err := uc.GetCurrentUser(context.Background(), CurrentUserInput{
		UserID: userID,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected wrapped repository error, got %v", err)
	}
}

func TestGetCurrentUserPropagatesPlatformStaffError(t *testing.T) {
	user := newCurrentUserTestUser(t, "alice", "active", time.Now().UTC())
	staffErr := errors.New("staff lookup failed")
	repo := &fakeCurrentUserRepository{
		user:     user,
		staffErr: staffErr,
	}
	uc := NewCurrentUserUseCase(repo, repo)

	_, err := uc.GetCurrentUser(context.Background(), CurrentUserInput{
		UserID: user.ID(),
	})
	if !errors.Is(err, staffErr) {
		t.Fatalf("expected staff lookup error to be wrapped, got %v", err)
	}
	if !repo.findPlatformAccessCalled {
		t.Fatal("expected FindPlatformAccess to be called")
	}
}

type fakeCurrentUserRepository struct {
	findByIDCalled           bool
	findPlatformAccessCalled bool
	findByID                 userdomain.UserID
	staffUserID              userdomain.UserID
	user                     *userdomain.User
	isPlatformStaff          bool
	platformRole             string
	err                      error
	staffErr                 error
}

func (f *fakeCurrentUserRepository) FindByID(ctx context.Context, id userdomain.UserID) (*userdomain.User, error) {
	f.findByIDCalled = true
	f.findByID = id
	return f.user, f.err
}

func (f *fakeCurrentUserRepository) FindPlatformAccess(ctx context.Context, userID userdomain.UserID) (PlatformAccess, error) {
	f.findPlatformAccessCalled = true
	f.staffUserID = userID
	if f.staffErr != nil {
		return PlatformAccess{}, f.staffErr
	}
	return PlatformAccess{IsPlatformStaff: f.isPlatformStaff, PlatformRole: f.platformRole}, nil
}

func newCurrentUserTestUser(t *testing.T, username string, status string, now time.Time) *userdomain.User {
	t.Helper()

	validUsername, err := userdomain.NewUsername(username)
	if err != nil {
		t.Fatalf("NewUsername returned error: %v", err)
	}
	passwordHash, err := userdomain.NewPasswordHash("hashed-password")
	if err != nil {
		t.Fatalf("NewPasswordHash returned error: %v", err)
	}
	userStatus, err := userdomain.NewUserStatus(status)
	if err != nil {
		t.Fatalf("NewUserStatus returned error: %v", err)
	}
	user, err := userdomain.RehydrateUser(userdomain.NewGeneratedUserID(), validUsername, passwordHash, userStatus, now, now)
	if err != nil {
		t.Fatalf("RehydrateUser returned error: %v", err)
	}
	return user
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

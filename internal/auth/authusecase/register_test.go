package authusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

const testTokenTypeBearer = "Bearer"

func TestRegisterUserSuccess(t *testing.T) {
	fixedNow := time.Date(2026, 5, 28, 10, 30, 0, 0, time.UTC)
	passwordHash := mustPasswordHash(t, "hashed-password")
	repo := &fakeUserCreator{}
	hasher := &fakePasswordHasher{
		hash: passwordHash,
	}
	issuer := &fakeAccessTokenIssuer{
		value:     "access-token",
		tokenType: testTokenTypeBearer,
		expiresIn: 24 * time.Hour,
	}
	uc := NewRegisterUserCase(repo, hasher, issuer, func() time.Time {
		return fixedNow
	})

	result, err := uc.Register(context.Background(), RegisterInput{
		Username: " Alice_123 ",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if result.AccessToken != "access-token" {
		t.Fatalf("expected access token %q, got %q", "access-token", result.AccessToken)
	}
	if result.TokenType != testTokenTypeBearer {
		t.Fatalf("expected token type %q, got %q", testTokenTypeBearer, result.TokenType)
	}
	if result.ExpiresIn != int64((24 * time.Hour).Seconds()) {
		t.Fatalf("expected expires_in %d, got %d", int64((24 * time.Hour).Seconds()), result.ExpiresIn)
	}
	if result.User.Username != "alice_123" {
		t.Fatalf("expected normalized username %q, got %q", "alice_123", result.User.Username)
	}
	if result.User.Status != "active" {
		t.Fatalf("expected user status active, got %q", result.User.Status)
	}
	if !result.User.CreatedAt.Equal(fixedNow) {
		t.Fatalf("expected created_at %s, got %s", fixedNow, result.User.CreatedAt)
	}

	if !repo.called {
		t.Fatal("expected user repository Create to be called")
	}
	if repo.user.Username().String() != "alice_123" {
		t.Fatalf("expected created user username %q, got %q", "alice_123", repo.user.Username().String())
	}
	if repo.user.PasswordHash().Raw() != passwordHash.Raw() {
		t.Fatalf("expected created user password hash %q, got %q", passwordHash.Raw(), repo.user.PasswordHash().Raw())
	}
	if repo.user.PasswordHash().Raw() == "password123" {
		t.Fatal("created user password hash must not equal plain password")
	}
	if !repo.user.CreatedAt().Equal(fixedNow) || !repo.user.UpdatedAt().Equal(fixedNow) {
		t.Fatal("created user timestamps should use injected time")
	}

	if !hasher.called {
		t.Fatal("expected password hasher to be called")
	}
	if hasher.plain.String() != "password123" {
		t.Fatalf("expected hasher plain password %q, got %q", "password123", hasher.plain.String())
	}
	if !issuer.called {
		t.Fatal("expected token issuer to be called")
	}
	if issuer.userID != repo.user.ID() {
		t.Fatalf("expected token issuer user id %q, got %q", repo.user.ID().String(), issuer.userID.String())
	}
	if !issuer.now.Equal(fixedNow) {
		t.Fatalf("expected token issuer time %s, got %s", fixedNow, issuer.now)
	}
}

func TestRegisterUserInvalidInputDoesNotCreateUser(t *testing.T) {
	tests := []struct {
		name  string
		input RegisterInput
	}{
		{
			name: "invalid username",
			input: RegisterInput{
				Username: "ab",
				Password: "password123",
			},
		},
		{
			name: "invalid password",
			input: RegisterInput{
				Username: "alice",
				Password: "short",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUserCreator{}
			hasher := &fakePasswordHasher{
				hash: mustPasswordHash(t, "hashed-password"),
			}
			issuer := &fakeAccessTokenIssuer{
				value:     "access-token",
				tokenType: testTokenTypeBearer,
				expiresIn: time.Hour,
			}
			uc := NewRegisterUserCase(repo, hasher, issuer, fixedClock())

			_, err := uc.Register(context.Background(), tt.input)
			if !hasAppCode(err, apperr.CodeInvalidArgument) {
				t.Fatalf("expected invalid_argument, got %v", err)
			}
			if repo.called {
				t.Fatal("repository Create should not be called for invalid input")
			}
			if issuer.called {
				t.Fatal("token issuer should not be called for invalid input")
			}
		})
	}
}

func TestRegisterUserHasherErrorDoesNotCreateUser(t *testing.T) {
	repo := &fakeUserCreator{}
	hasher := &fakePasswordHasher{
		err: errors.New("hash failed"),
	}
	issuer := &fakeAccessTokenIssuer{
		value:     "access-token",
		tokenType: testTokenTypeBearer,
		expiresIn: time.Hour,
	}
	uc := NewRegisterUserCase(repo, hasher, issuer, fixedClock())

	_, err := uc.Register(context.Background(), RegisterInput{
		Username: "alice",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if repo.called {
		t.Fatal("repository Create should not be called when hashing fails")
	}
	if issuer.called {
		t.Fatal("token issuer should not be called when hashing fails")
	}
}

func TestRegisterUserTokenIssuerErrorDoesNotCreateUser(t *testing.T) {
	repo := &fakeUserCreator{}
	hasher := &fakePasswordHasher{
		hash: mustPasswordHash(t, "hashed-password"),
	}
	issuer := &fakeAccessTokenIssuer{
		err: errors.New("issue token failed"),
	}
	uc := NewRegisterUserCase(repo, hasher, issuer, fixedClock())

	_, err := uc.Register(context.Background(), RegisterInput{
		Username: "alice",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !issuer.called {
		t.Fatal("expected token issuer to be called")
	}
	if repo.called {
		t.Fatal("repository Create should not be called when token issuing fails")
	}
}

func TestRegisterUserCreateConflictPreservesAppError(t *testing.T) {
	repo := &fakeUserCreator{
		err: apperr.New(apperr.CodeConflict, "username already exists"),
	}
	hasher := &fakePasswordHasher{
		hash: mustPasswordHash(t, "hashed-password"),
	}
	issuer := &fakeAccessTokenIssuer{
		value:     "access-token",
		tokenType: testTokenTypeBearer,
		expiresIn: time.Hour,
	}
	uc := NewRegisterUserCase(repo, hasher, issuer, fixedClock())

	_, err := uc.Register(context.Background(), RegisterInput{
		Username: "alice",
		Password: "password123",
	})
	if !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if !repo.called {
		t.Fatal("expected repository Create to be called")
	}
}

type fakeUserCreator struct {
	called bool
	user   userdomain.User
	err    error
}

func (f *fakeUserCreator) Create(ctx context.Context, user userdomain.User) error {
	f.called = true
	f.user = user
	return f.err
}

type fakePasswordHasher struct {
	called bool
	plain  userdomain.PlainPassword
	hash   userdomain.PasswordHash
	err    error
}

func (f *fakePasswordHasher) Hash(plain userdomain.PlainPassword) (userdomain.PasswordHash, error) {
	f.called = true
	f.plain = plain
	return f.hash, f.err
}

type fakeAccessTokenIssuer struct {
	called    bool
	userID    userdomain.UserID
	now       time.Time
	value     string
	tokenType string
	expiresIn time.Duration
	err       error
}

func (f *fakeAccessTokenIssuer) IssueAccessToken(userID userdomain.UserID, now time.Time) (string, string, time.Duration, error) {
	f.called = true
	f.userID = userID
	f.now = now
	return f.value, f.tokenType, f.expiresIn, f.err
}

func fixedClock() func() time.Time {
	return func() time.Time {
		return time.Date(2026, 5, 28, 10, 30, 0, 0, time.UTC)
	}
}

func mustPasswordHash(t *testing.T, raw string) userdomain.PasswordHash {
	t.Helper()

	hash, err := userdomain.NewPasswordHash(raw)
	if err != nil {
		t.Fatalf("NewPasswordHash returned error: %v", err)
	}
	return hash
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

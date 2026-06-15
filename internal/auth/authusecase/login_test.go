package authusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/progression/progressionusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestLoginUserSuccess(t *testing.T) {
	fixedNow := time.Date(2026, 5, 29, 10, 30, 0, 0, time.UTC)
	user := newLoginTestUser(t, "alice", "hashed-password", "active", fixedNow)
	finder := &fakeLoginUserFinder{
		user: user,
	}
	comparer := &fakeLoginPasswordComparer{}
	issuer := &fakeLoginTokenIssuer{
		value:     "access-token",
		tokenType: testTokenTypeBearer,
		expiresIn: 24 * time.Hour,
	}
	uc := NewLoginUserCase(finder, comparer, issuer, func() time.Time {
		return fixedNow
	})

	result, err := uc.Login(context.Background(), LoginInput{
		Username:  " Alice ",
		Password:  "password123",
		RequestIP: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
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
	if result.User.ID != user.ID().String() {
		t.Fatalf("expected user id %q, got %q", user.ID().String(), result.User.ID)
	}
	if result.User.Username != "alice" {
		t.Fatalf("expected username %q, got %q", "alice", result.User.Username)
	}
	if result.User.Status != "active" {
		t.Fatalf("expected status %q, got %q", "active", result.User.Status)
	}
	if !result.User.CreatedAt.Equal(fixedNow) {
		t.Fatalf("expected created_at %s, got %s", fixedNow, result.User.CreatedAt)
	}

	if !finder.called {
		t.Fatal("expected user finder to be called")
	}
	if finder.identifier != "alice" {
		t.Fatalf("expected normalized identifier %q, got %q", "alice", finder.identifier)
	}
	if !comparer.called {
		t.Fatal("expected password comparer to be called")
	}
	if comparer.hash != user.PasswordHash() {
		t.Fatalf("expected password hash %q, got %q", user.PasswordHash().Raw(), comparer.hash.Raw())
	}
	if comparer.plain.String() != "password123" {
		t.Fatalf("expected plain password %q, got %q", "password123", comparer.plain.String())
	}
	if !issuer.called {
		t.Fatal("expected token issuer to be called")
	}
	if issuer.userID != user.ID() {
		t.Fatalf("expected token user id %q, got %q", user.ID().String(), issuer.userID.String())
	}
	if !issuer.now.Equal(fixedNow) {
		t.Fatalf("expected token issue time %s, got %s", fixedNow, issuer.now)
	}
	if !finder.updateLastLoginCalled {
		t.Fatal("expected last login to be updated")
	}
	if finder.loginIP != "127.0.0.1" {
		t.Fatalf("expected login ip %q, got %q", "127.0.0.1", finder.loginIP)
	}
	if len(finder.events) != 1 {
		t.Fatalf("expected one login security event, got %d", len(finder.events))
	}
	if finder.events[0].Action != loginPasswordSuccessAction {
		t.Fatalf("expected success login event, got %q", finder.events[0].Action)
	}
}

func TestLoginUserSuccessGrantsDailyLoginXP(t *testing.T) {
	fixedNow := time.Date(2026, 6, 14, 23, 30, 0, 0, time.UTC)
	user := newLoginTestUser(t, "alice", "hashed-password", "active", fixedNow.Add(-24*time.Hour))
	finder := &fakeLoginUserFinder{user: user}
	uc := NewLoginUserCase(
		finder,
		&fakeLoginPasswordComparer{},
		&fakeLoginTokenIssuer{value: "access-token", tokenType: testTokenTypeBearer, expiresIn: time.Hour},
		func() time.Time { return fixedNow },
	)
	recorder := &fakeXPRecorder{}
	uc.SetXPRecorder(recorder)

	_, err := uc.Login(context.Background(), LoginInput{
		Username: "alice",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if len(recorder.inputs) != 1 {
		t.Fatalf("expected one xp grant, got %d", len(recorder.inputs))
	}
	input := recorder.inputs[0]
	if input.UserID != user.ID() || input.ActorID != user.ID() {
		t.Fatalf("expected daily login xp for self, got %#v", input)
	}
	if input.SourceType != progressionusecase.XPSourceDailyLogin {
		t.Fatalf("expected source type %q, got %q", progressionusecase.XPSourceDailyLogin, input.SourceType)
	}
	if input.SourceID != "2026-06-14" {
		t.Fatalf("expected source id %q, got %q", "2026-06-14", input.SourceID)
	}
}

func TestLoginUserInvalidInputDoesNotFindUser(t *testing.T) {
	tests := []struct {
		name  string
		input LoginInput
	}{
		{
			name: "invalid username",
			input: LoginInput{
				Username: "ab",
				Password: "password123",
			},
		},
		{
			name: "invalid password",
			input: LoginInput{
				Username: "alice",
				Password: "short",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finder := &fakeLoginUserFinder{}
			comparer := &fakeLoginPasswordComparer{}
			issuer := &fakeLoginTokenIssuer{}
			uc := NewLoginUserCase(finder, comparer, issuer, fixedClock())

			_, err := uc.Login(context.Background(), tt.input)
			if !hasAppCode(err, apperr.CodeInvalidArgument) {
				t.Fatalf("expected invalid_argument, got %v", err)
			}
			if finder.called {
				t.Fatal("user finder should not be called for invalid input")
			}
			if comparer.called {
				t.Fatal("password comparer should not be called for invalid input")
			}
			if issuer.called {
				t.Fatal("token issuer should not be called for invalid input")
			}
		})
	}
}

func TestLoginUserNotFoundReturnsUnauthenticated(t *testing.T) {
	finder := &fakeLoginUserFinder{
		err: apperr.New(apperr.CodeNotFound, "user not found"),
	}
	comparer := &fakeLoginPasswordComparer{}
	issuer := &fakeLoginTokenIssuer{}
	uc := NewLoginUserCase(finder, comparer, issuer, fixedClock())

	_, err := uc.Login(context.Background(), LoginInput{
		Username: "alice",
		Password: "password123",
	})
	if !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
	if !finder.called {
		t.Fatal("expected user finder to be called")
	}
	if comparer.called {
		t.Fatal("password comparer should not be called when user is missing")
	}
	if issuer.called {
		t.Fatal("token issuer should not be called when user is missing")
	}
}

func TestLoginUserPasswordCompareErrorReturnsUnauthenticated(t *testing.T) {
	user := newLoginTestUser(t, "alice", "hashed-password", "active", time.Now().UTC())
	finder := &fakeLoginUserFinder{
		user: user,
	}
	comparer := &fakeLoginPasswordComparer{
		err: errors.New("password mismatch"),
	}
	issuer := &fakeLoginTokenIssuer{}
	uc := NewLoginUserCase(finder, comparer, issuer, fixedClock())

	_, err := uc.Login(context.Background(), LoginInput{
		Username: "alice",
		Password: "password123",
	})
	if !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
	if !comparer.called {
		t.Fatal("expected password comparer to be called")
	}
	if issuer.called {
		t.Fatal("token issuer should not be called when password compare fails")
	}
	if len(finder.events) != 1 {
		t.Fatalf("expected one failed login event, got %d", len(finder.events))
	}
	if finder.events[0].Action != loginPasswordFailedAction {
		t.Fatalf("expected failed login event, got %q", finder.events[0].Action)
	}
}

func TestLoginUserBlockedByAccountFailureLimit(t *testing.T) {
	finder := &fakeLoginUserFinder{
		accountFailureCount: loginFailureAccountLimit,
	}
	comparer := &fakeLoginPasswordComparer{}
	issuer := &fakeLoginTokenIssuer{}
	uc := NewLoginUserCase(finder, comparer, issuer, fixedClock())

	_, err := uc.Login(context.Background(), LoginInput{
		Username: "alice",
		Password: "password123",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if finder.called {
		t.Fatal("user finder should not be called when account is rate limited")
	}
	if comparer.called {
		t.Fatal("password comparer should not be called when account is rate limited")
	}
}

func TestLoginUserBlockedByIPFailureLimit(t *testing.T) {
	finder := &fakeLoginUserFinder{
		ipFailureCount: loginFailureIPLimit,
	}
	comparer := &fakeLoginPasswordComparer{}
	issuer := &fakeLoginTokenIssuer{}
	uc := NewLoginUserCase(finder, comparer, issuer, fixedClock())

	_, err := uc.Login(context.Background(), LoginInput{
		Username:  "alice",
		Password:  "password123",
		RequestIP: "127.0.0.1",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if finder.called {
		t.Fatal("user finder should not be called when ip is rate limited")
	}
	if comparer.called {
		t.Fatal("password comparer should not be called when ip is rate limited")
	}
}

func TestLoginUserDisabledReturnsForbidden(t *testing.T) {
	user := newLoginTestUser(t, "alice", "hashed-password", "disabled", time.Now().UTC())
	finder := &fakeLoginUserFinder{
		user: user,
	}
	comparer := &fakeLoginPasswordComparer{}
	issuer := &fakeLoginTokenIssuer{}
	uc := NewLoginUserCase(finder, comparer, issuer, fixedClock())

	_, err := uc.Login(context.Background(), LoginInput{
		Username: "alice",
		Password: "password123",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if !comparer.called {
		t.Fatal("expected password comparer to be called")
	}
	if issuer.called {
		t.Fatal("token issuer should not be called for disabled user")
	}
}

func TestLoginUserActiveBanReturnsForbidden(t *testing.T) {
	user := newLoginTestUser(t, "alice", "hashed-password", "active", time.Now().UTC())
	finder := &fakeLoginUserFinder{
		user:      user,
		activeBan: true,
	}
	comparer := &fakeLoginPasswordComparer{}
	issuer := &fakeLoginTokenIssuer{}
	uc := NewLoginUserCase(finder, comparer, issuer, fixedClock())

	_, err := uc.Login(context.Background(), LoginInput{
		Username: "alice",
		Password: "password123",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if !comparer.called {
		t.Fatal("expected password comparer to be called")
	}
	if issuer.called {
		t.Fatal("token issuer should not be called for banned user")
	}
	if len(finder.events) != 1 || finder.events[0].Action != loginPasswordFailedAction {
		t.Fatalf("expected failed login event, got %#v", finder.events)
	}
}

func TestLoginUserTokenIssuerError(t *testing.T) {
	user := newLoginTestUser(t, "alice", "hashed-password", "active", time.Now().UTC())
	finder := &fakeLoginUserFinder{
		user: user,
	}
	comparer := &fakeLoginPasswordComparer{}
	issuer := &fakeLoginTokenIssuer{
		err: errors.New("issue token failed"),
	}
	uc := NewLoginUserCase(finder, comparer, issuer, fixedClock())

	_, err := uc.Login(context.Background(), LoginInput{
		Username: "alice",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !issuer.called {
		t.Fatal("expected token issuer to be called")
	}
}

type fakeLoginUserFinder struct {
	called                bool
	updateLastLoginCalled bool
	identifier            string
	loginIP               string
	user                  *userdomain.User
	err                   error
	accountFailureCount   int
	ipFailureCount        int
	activeBan             bool
	events                []SecurityEvent
}

func (f *fakeLoginUserFinder) FindAuthUserByIdentifier(ctx context.Context, identifier string) (AuthUserRecord, error) {
	f.called = true
	f.identifier = identifier
	if f.err != nil {
		return AuthUserRecord{}, f.err
	}
	return AuthUserRecord{User: f.user}, nil
}

func (f *fakeLoginUserFinder) UpdateLastLogin(ctx context.Context, userID userdomain.UserID, loginAt time.Time, loginIP string) error {
	f.updateLastLoginCalled = true
	f.loginIP = loginIP
	return nil
}

func (f *fakeLoginUserFinder) HasActiveAccountBan(ctx context.Context, userID userdomain.UserID, now time.Time) (bool, error) {
	return f.activeBan, nil
}

func (f *fakeLoginUserFinder) CountSecurityEventsSince(ctx context.Context, action string, identity string, requestIP string, since time.Time) (int, error) {
	if requestIP != "" {
		return f.ipFailureCount, nil
	}
	return f.accountFailureCount, nil
}

func (f *fakeLoginUserFinder) RecordSecurityEvent(ctx context.Context, event SecurityEvent) error {
	f.events = append(f.events, event)
	return nil
}

type fakeLoginPasswordComparer struct {
	called bool
	hash   userdomain.PasswordHash
	plain  userdomain.PlainPassword
	err    error
}

func (f *fakeLoginPasswordComparer) Compare(hash userdomain.PasswordHash, plain userdomain.PlainPassword) error {
	f.called = true
	f.hash = hash
	f.plain = plain
	return f.err
}

type fakeLoginTokenIssuer struct {
	called    bool
	userID    userdomain.UserID
	now       time.Time
	value     string
	tokenType string
	expiresIn time.Duration
	err       error
}

func (f *fakeLoginTokenIssuer) IssueAccessToken(userID userdomain.UserID, now time.Time) (string, string, time.Duration, error) {
	f.called = true
	f.userID = userID
	f.now = now
	return f.value, f.tokenType, f.expiresIn, f.err
}

func newLoginTestUser(t *testing.T, username string, passwordHash string, status string, now time.Time) *userdomain.User {
	t.Helper()

	userID := userdomain.NewGeneratedUserID()
	validUsername, err := userdomain.NewUsername(username)
	if err != nil {
		t.Fatalf("NewUsername returned error: %v", err)
	}
	validPasswordHash, err := userdomain.NewPasswordHash(passwordHash)
	if err != nil {
		t.Fatalf("NewPasswordHash returned error: %v", err)
	}
	validStatus, err := userdomain.NewUserStatus(status)
	if err != nil {
		t.Fatalf("NewUserStatus returned error: %v", err)
	}
	user, err := userdomain.RehydrateUser(userID, validUsername, validPasswordHash, validStatus, now, now)
	if err != nil {
		t.Fatalf("RehydrateUser returned error: %v", err)
	}
	return user
}

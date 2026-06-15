package authusecase

import (
	"context"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestSendDeleteAccountEmailCodeRequiresCurrentEmail(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	now := fixedClock()()
	repo := newFakeSecurityRepository(t, userID, "student@cumt.edu.cn", now)
	uc := newTestSecurityUseCase(t, repo, now)

	_, err := uc.SendDeleteAccountEmailCode(context.Background(), DeleteAccountCodeInput{
		UserID: userID,
		Email:  "other@cumt.edu.cn",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if repo.createdCode != nil {
		t.Fatal("delete account code should not be created for mismatched email")
	}
}

func TestLoginWithEmailCodeGrantsDailyLoginXP(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 14, 8, 0, 0, 0, time.UTC)
	repo := newFakeSecurityRepository(t, userID, "student@cumt.edu.cn", now)
	repo.authUser = newLoginTestUser(t, "student", "hashed-password", "active", now.Add(-24*time.Hour))
	repo.pendingCode = &EmailCodeRecord{
		ID:        userdomain.NewGeneratedUserID().String(),
		Email:     "student@cumt.edu.cn",
		Purpose:   EmailPurposeLogin,
		CodeHash:  "123456",
		Status:    "pending",
		ExpiresAt: now.Add(10 * time.Minute),
		CreatedAt: now,
		UpdatedAt: now,
	}
	uc := newTestSecurityUseCase(t, repo, now)
	recorder := &fakeXPRecorder{}
	uc.SetXPRecorder(recorder)

	result, err := uc.LoginWithEmailCode(context.Background(), LoginWithEmailCodeInput{
		Email: "student@cumt.edu.cn",
		Code:  "123456",
	})
	if err != nil {
		t.Fatalf("LoginWithEmailCode returned error: %v", err)
	}
	if result.AccessToken != "token" {
		t.Fatalf("expected access token %q, got %q", "token", result.AccessToken)
	}
	if !repo.codeUsed {
		t.Fatal("expected email code to be consumed")
	}
	if len(recorder.inputs) != 1 {
		t.Fatalf("expected one xp grant, got %d", len(recorder.inputs))
	}
	input := recorder.inputs[0]
	if input.UserID != repo.authUser.ID() || input.ActorID != repo.authUser.ID() {
		t.Fatalf("expected daily login xp for self, got %#v", input)
	}
	if input.SourceType != "daily_login" {
		t.Fatalf("expected daily login source, got %q", input.SourceType)
	}
	if input.SourceID != "2026-06-14" {
		t.Fatalf("expected source id %q, got %q", "2026-06-14", input.SourceID)
	}
}

func TestDeleteAccountAcceptsEmailCode(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	now := fixedClock()()
	repo := newFakeSecurityRepository(t, userID, "student@cumt.edu.cn", now)
	repo.pendingCode = &EmailCodeRecord{
		ID:        userdomain.NewGeneratedUserID().String(),
		Email:     "student@cumt.edu.cn",
		Purpose:   EmailPurposeDeleteAccount,
		CodeHash:  "123456",
		Status:    "pending",
		ExpiresAt: now.Add(10 * time.Minute),
		CreatedAt: now,
		UpdatedAt: now,
	}
	uc := newTestSecurityUseCase(t, repo, now)

	err := uc.DeleteAccount(context.Background(), DeleteAccountInput{
		UserID:       userID,
		Code:         "123456",
		Confirmation: "DELETE",
		RequestIP:    "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("DeleteAccount returned error: %v", err)
	}
	if !repo.codeUsed {
		t.Fatal("expected delete account code to be consumed")
	}
	if !repo.accountDeleted {
		t.Fatal("expected account to be soft deleted")
	}
	if len(repo.events) != 1 || repo.events[0].Action != "account_deleted" {
		t.Fatalf("expected account_deleted event, got %#v", repo.events)
	}
}

func TestDeleteAccountAcceptsCurrentPassword(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	now := fixedClock()()
	repo := newFakeSecurityRepository(t, userID, "student@cumt.edu.cn", now)
	repo.security.EmailVerifiedAt = nil
	comparer := &fakeLoginPasswordComparer{}
	uc := newTestSecurityUseCase(t, repo, now)
	uc.passwordComparer = comparer

	err := uc.DeleteAccount(context.Background(), DeleteAccountInput{
		UserID:          userID,
		CurrentPassword: "password123",
		Confirmation:    "DELETE",
		RequestIP:       "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("DeleteAccount returned error: %v", err)
	}
	if !comparer.called {
		t.Fatal("expected current password comparer to be called")
	}
	if repo.codeUsed {
		t.Fatal("email code should not be consumed for password challenge")
	}
	if !repo.accountDeleted {
		t.Fatal("expected account to be soft deleted")
	}
}

func TestDeleteAccountRequiresPasswordOrEmailCode(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	now := fixedClock()()
	repo := newFakeSecurityRepository(t, userID, "student@cumt.edu.cn", now)
	uc := newTestSecurityUseCase(t, repo, now)

	err := uc.DeleteAccount(context.Background(), DeleteAccountInput{
		UserID:       userID,
		Confirmation: "DELETE",
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument, got %v", err)
	}
	if repo.accountDeleted {
		t.Fatal("account should not be deleted without password or email code")
	}
}

func TestDeleteAccountRequiresConfirmation(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	now := fixedClock()()
	repo := newFakeSecurityRepository(t, userID, "student@cumt.edu.cn", now)
	uc := newTestSecurityUseCase(t, repo, now)

	err := uc.DeleteAccount(context.Background(), DeleteAccountInput{
		UserID:          userID,
		Code:            "123456",
		CurrentPassword: "password123",
		Confirmation:    "delete",
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument, got %v", err)
	}
	if repo.accountDeleted {
		t.Fatal("account should not be deleted when confirmation is invalid")
	}
}

func TestSendRegisterEmailCodeMarksCodeFailedWhenSenderFails(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	now := fixedClock()()
	repo := newFakeSecurityRepository(t, userID, "student@cumt.edu.cn", now)
	uc := NewSecurityUseCase(
		repo,
		&fakePasswordHasher{hash: mustPasswordHash(t, "hashed-password")},
		&fakeLoginPasswordComparer{},
		&fakeAccessTokenIssuer{value: "token", tokenType: testTokenTypeBearer, expiresIn: time.Hour},
		fakeEmailCodeGenerator{},
		fakeEmailCodeHasher{},
		failingEmailCodeSender{},
		EmailCodePolicy{
			AllowedDomains: []string{"cumt.edu.cn", "mail.cumt.edu.cn"},
			TTL:            10 * time.Minute,
			ResendInterval: time.Minute,
			MaxAttempts:    5,
			DailyLimit:     10,
			IPHourlyLimit:  30,
			CodeLength:     6,
		},
		func() time.Time { return now },
	)

	_, err := uc.SendRegisterEmailCode(context.Background(), EmailCodeDispatchInput{
		Email: "new@cumt.edu.cn",
	})
	if err == nil {
		t.Fatal("expected sender error")
	}
	if repo.createdCode == nil {
		t.Fatal("expected code to be created before sending")
	}
	if !repo.codeFailed {
		t.Fatal("expected failed send to mark code failed")
	}
	if repo.pendingCode == nil || repo.pendingCode.Status != "expired" {
		t.Fatalf("expected failed code to be expired, got %#v", repo.pendingCode)
	}
}

func newTestSecurityUseCase(t *testing.T, repo *fakeSecurityRepository, now time.Time) *SecurityUseCase {
	t.Helper()
	return NewSecurityUseCase(
		repo,
		&fakePasswordHasher{hash: mustPasswordHash(t, "hashed-password")},
		&fakeLoginPasswordComparer{},
		&fakeAccessTokenIssuer{value: "token", tokenType: testTokenTypeBearer, expiresIn: time.Hour},
		fakeEmailCodeGenerator{},
		fakeEmailCodeHasher{},
		fakeEmailCodeSender{},
		EmailCodePolicy{
			AllowedDomains: []string{"cumt.edu.cn", "mail.cumt.edu.cn"},
			TTL:            10 * time.Minute,
			ResendInterval: time.Minute,
			MaxAttempts:    5,
			DailyLimit:     10,
			IPHourlyLimit:  30,
			CodeLength:     6,
		},
		func() time.Time { return now },
	)
}

type fakeSecurityRepository struct {
	userID         userdomain.UserID
	authUser       *userdomain.User
	security       SecurityInfo
	pendingCode    *EmailCodeRecord
	createdCode    *EmailCodeRecord
	codeUsed       bool
	codeFailed     bool
	accountDeleted bool
	events         []SecurityEvent
}

func newFakeSecurityRepository(t *testing.T, userID userdomain.UserID, email string, now time.Time) *fakeSecurityRepository {
	t.Helper()
	passwordHash := mustPasswordHash(t, "hashed-password")
	verifiedAt := now.Add(-time.Hour)
	return &fakeSecurityRepository{
		userID: userID,
		security: SecurityInfo{
			UserID:          userID,
			Email:           email,
			EmailVerifiedAt: &verifiedAt,
			PasswordHash:    passwordHash,
			CreatedAt:       now.Add(-24 * time.Hour),
		},
	}
}

func (f *fakeSecurityRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	return email == f.security.Email, nil
}

func (f *fakeSecurityRepository) FindAuthUserByEmail(ctx context.Context, email string) (AuthUserRecord, error) {
	if f.authUser != nil && email == f.security.Email {
		return AuthUserRecord{
			User:            f.authUser,
			Email:           f.security.Email,
			EmailVerifiedAt: f.security.EmailVerifiedAt,
		}, nil
	}
	return AuthUserRecord{}, apperr.New(apperr.CodeNotFound, "user not found")
}

func (f *fakeSecurityRepository) CreateEmailCode(ctx context.Context, code EmailCodeRecord) error {
	f.createdCode = &code
	f.pendingCode = &code
	return nil
}

func (f *fakeSecurityRepository) LatestEmailCode(ctx context.Context, email string, purpose EmailPurpose) (EmailCodeRecord, error) {
	return EmailCodeRecord{}, apperr.New(apperr.CodeNotFound, "email code not found")
}

func (f *fakeSecurityRepository) CountEmailCodesSince(ctx context.Context, email string, purpose EmailPurpose, since time.Time) (int, error) {
	return 0, nil
}

func (f *fakeSecurityRepository) CountEmailCodesByIPSince(ctx context.Context, requestIP string, since time.Time) (int, error) {
	return 0, nil
}

func (f *fakeSecurityRepository) FindPendingEmailCode(ctx context.Context, email string, purpose EmailPurpose, now time.Time) (EmailCodeRecord, error) {
	if f.pendingCode == nil || f.pendingCode.Email != email || f.pendingCode.Purpose != purpose {
		return EmailCodeRecord{}, apperr.New(apperr.CodeNotFound, "email code not found")
	}
	return *f.pendingCode, nil
}

func (f *fakeSecurityRepository) MarkEmailCodeUsed(ctx context.Context, id string, now time.Time) error {
	f.codeUsed = true
	return nil
}

func (f *fakeSecurityRepository) MarkEmailCodeAttempt(ctx context.Context, id string, attemptCount int, status string, now time.Time) error {
	return nil
}

func (f *fakeSecurityRepository) MarkEmailCodeFailed(ctx context.Context, id string, now time.Time) error {
	f.codeFailed = true
	if f.pendingCode != nil && f.pendingCode.ID == id {
		f.pendingCode.Status = "expired"
		f.pendingCode.UpdatedAt = now
	}
	return nil
}

func (f *fakeSecurityRepository) CreateUserWithEmail(ctx context.Context, user userdomain.User, email string, verifiedAt time.Time, passwordUpdatedAt time.Time) error {
	return nil
}

func (f *fakeSecurityRepository) UpdateLastLogin(ctx context.Context, userID userdomain.UserID, loginAt time.Time, loginIP string) error {
	return nil
}

func (f *fakeSecurityRepository) UpdatePasswordByEmail(ctx context.Context, email string, passwordHash userdomain.PasswordHash, updatedAt time.Time) error {
	return nil
}

func (f *fakeSecurityRepository) GetSecurityByUserID(ctx context.Context, userID userdomain.UserID) (SecurityInfo, error) {
	if userID != f.userID {
		return SecurityInfo{}, apperr.New(apperr.CodeNotFound, "user not found")
	}
	return f.security, nil
}

func (f *fakeSecurityRepository) UpdateEmailByUserID(ctx context.Context, userID userdomain.UserID, email string, verifiedAt time.Time) (SecurityInfo, error) {
	return f.security, nil
}

func (f *fakeSecurityRepository) UpdatePasswordByUserID(ctx context.Context, userID userdomain.UserID, passwordHash userdomain.PasswordHash, updatedAt time.Time) error {
	return nil
}

func (f *fakeSecurityRepository) RevokeTokensByUserID(ctx context.Context, userID userdomain.UserID, revokedAfter time.Time) error {
	return nil
}

func (f *fakeSecurityRepository) DeleteAccountByUserID(ctx context.Context, userID userdomain.UserID, deletedAt time.Time) error {
	f.accountDeleted = true
	return nil
}

func (f *fakeSecurityRepository) RecordSecurityEvent(ctx context.Context, event SecurityEvent) error {
	f.events = append(f.events, event)
	return nil
}

type fakeEmailCodeGenerator struct{}

func (fakeEmailCodeGenerator) GenerateNumericCode(length int) (string, error) {
	return "123456", nil
}

type fakeEmailCodeHasher struct{}

func (fakeEmailCodeHasher) Hash(email string, purpose string, code string) string {
	return code
}

func (fakeEmailCodeHasher) Compare(email string, purpose string, code string, hash string) bool {
	return code == hash
}

type fakeEmailCodeSender struct{}

func (fakeEmailCodeSender) SendEmailCode(ctx context.Context, email string, purpose string, code string, ttl time.Duration) error {
	return nil
}

type failingEmailCodeSender struct{}

func (failingEmailCodeSender) SendEmailCode(ctx context.Context, email string, purpose string, code string, ttl time.Duration) error {
	return apperr.New(apperr.CodeInternal, "smtp failed")
}

package authusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	platformsettings "github.com/Versifine/cumt-nexus-api/internal/platform/settings"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type EmailPurpose string

const (
	EmailPurposeRegister      EmailPurpose = "register"
	EmailPurposeLogin         EmailPurpose = "login"
	EmailPurposePasswordReset EmailPurpose = "password_reset"
	EmailPurposeChangeEmail   EmailPurpose = "change_email"
	EmailPurposeDeleteAccount EmailPurpose = "delete_account"
)

type EmailCodePolicy struct {
	AllowedDomains []string
	TTL            time.Duration
	ResendInterval time.Duration
	MaxAttempts    int
	DailyLimit     int
	IPHourlyLimit  int
	CodeLength     int
}

type EmailCodeGenerator interface {
	GenerateNumericCode(length int) (string, error)
}

type EmailCodeHasher interface {
	Hash(email string, purpose string, code string) string
	Compare(email string, purpose string, code string, hash string) bool
}

type EmailCodeSender interface {
	SendEmailCode(ctx context.Context, email string, purpose string, code string, ttl time.Duration) error
}

type AuthSecurityRepository interface {
	EmailExists(ctx context.Context, email string) (bool, error)
	FindAuthUserByEmail(ctx context.Context, email string) (AuthUserRecord, error)
	CreateEmailCode(ctx context.Context, code EmailCodeRecord) error
	LatestEmailCode(ctx context.Context, email string, purpose EmailPurpose) (EmailCodeRecord, error)
	CountEmailCodesSince(ctx context.Context, email string, purpose EmailPurpose, since time.Time) (int, error)
	CountEmailCodesByIPSince(ctx context.Context, requestIP string, since time.Time) (int, error)
	FindPendingEmailCode(ctx context.Context, email string, purpose EmailPurpose, now time.Time) (EmailCodeRecord, error)
	MarkEmailCodeUsed(ctx context.Context, id string, now time.Time) error
	MarkEmailCodeAttempt(ctx context.Context, id string, attemptCount int, status string, now time.Time) error
	MarkEmailCodeFailed(ctx context.Context, id string, now time.Time) error
	HasActiveAccountBan(ctx context.Context, userID userdomain.UserID, now time.Time) (bool, error)
	CreateUserWithEmail(ctx context.Context, user userdomain.User, email string, verifiedAt time.Time, passwordUpdatedAt time.Time) error
	UpdateLastLogin(ctx context.Context, userID userdomain.UserID, loginAt time.Time, loginIP string) error
	UpdatePasswordByEmail(ctx context.Context, email string, passwordHash userdomain.PasswordHash, updatedAt time.Time) error
	GetSecurityByUserID(ctx context.Context, userID userdomain.UserID) (SecurityInfo, error)
	UpdateEmailByUserID(ctx context.Context, userID userdomain.UserID, email string, verifiedAt time.Time) (SecurityInfo, error)
	UpdatePasswordByUserID(ctx context.Context, userID userdomain.UserID, passwordHash userdomain.PasswordHash, updatedAt time.Time) error
	RevokeTokensByUserID(ctx context.Context, userID userdomain.UserID, revokedAfter time.Time) error
	DeleteAccountByUserID(ctx context.Context, userID userdomain.UserID, deletedAt time.Time) error
	RecordSecurityEvent(ctx context.Context, event SecurityEvent) error
}

type AuthUserRecord struct {
	User               *userdomain.User
	Email              string
	EmailVerifiedAt    *time.Time
	LastLoginAt        *time.Time
	PasswordUpdatedAt  *time.Time
	TokensRevokedAfter *time.Time
	IsPlatformStaff    bool
	PlatformRole       string
}

type EmailCodeRecord struct {
	ID           string
	Email        string
	Purpose      EmailPurpose
	CodeHash     string
	Status       string
	AttemptCount int
	SentCount    int
	LastSentAt   time.Time
	ExpiresAt    time.Time
	RequestIP    string
	UserAgent    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type SecurityEvent struct {
	ID        string
	UserID    *userdomain.UserID
	Email     string
	Action    string
	IP        string
	UserAgent string
	Metadata  map[string]any
	CreatedAt time.Time
}

type SecurityInfo struct {
	UserID             userdomain.UserID
	Email              string
	EmailVerifiedAt    *time.Time
	PasswordHash       userdomain.PasswordHash
	LastLoginAt        *time.Time
	CreatedAt          time.Time
	TokensRevokedAfter *time.Time
}

type SecurityUseCase struct {
	repository       AuthSecurityRepository
	passwordHasher   PasswordHasher
	passwordComparer PasswordComparer
	tokenIssuer      LoginTokenIssuer
	codeGenerator    EmailCodeGenerator
	codeHasher       EmailCodeHasher
	emailSender      EmailCodeSender
	settingsReader   platformsettings.Reader
	xpRecorder       XPRecorder
	policy           EmailCodePolicy
	now              func() time.Time
}

type EmailCodeDispatchInput struct {
	Email     string
	RequestIP string
	UserAgent string
}

type AuthRequestContext struct {
	RequestIP string
	UserAgent string
}

type EmailCodeDispatchResult struct {
	Email       string
	Purpose     string
	ExpiresIn   int64
	ResendAfter int64
}

type RegisterWithEmailInput struct {
	Email     string
	Code      string
	Username  string
	Password  string
	RequestIP string
	UserAgent string
}

type LoginWithEmailCodeInput struct {
	Email     string
	Code      string
	RequestIP string
	UserAgent string
}

type PasswordResetInput struct {
	Email       string
	Code        string
	NewPassword string
	RequestIP   string
	UserAgent   string
}

type ChangeEmailCodeInput struct {
	UserID    userdomain.UserID
	NewEmail  string
	RequestIP string
	UserAgent string
}

type ChangeEmailInput struct {
	UserID    userdomain.UserID
	NewEmail  string
	Code      string
	RequestIP string
	UserAgent string
}

type ChangePasswordInput struct {
	UserID          userdomain.UserID
	CurrentPassword string
	NewPassword     string
	RequestIP       string
	UserAgent       string
}

type LogoutAllInput struct {
	UserID    userdomain.UserID
	RequestIP string
	UserAgent string
}

type DeleteAccountCodeInput struct {
	UserID    userdomain.UserID
	Email     string
	RequestIP string
	UserAgent string
}

type DeleteAccountInput struct {
	UserID          userdomain.UserID
	Code            string
	CurrentPassword string
	Confirmation    string
	RequestIP       string
	UserAgent       string
}

type AccountSecurityInput struct {
	UserID userdomain.UserID
}

type AccountSecurityResult struct {
	Email           string
	EmailVerified   bool
	EmailVerifiedAt *time.Time
	PasswordSet     bool
	LastLoginAt     *time.Time
	CreatedAt       time.Time
}

type EmailChangeResult struct {
	Email           string
	EmailVerified   bool
	EmailVerifiedAt *time.Time
}

type PasswordUpdatedResult struct {
	Updated bool
}

type SecurityAuthResult struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int64
	User        AuthResultUser
}

type AuthResultUser struct {
	ID              string
	Username        string
	Status          string
	Email           string
	EmailVerified   bool
	IsPlatformStaff bool
	PlatformRole    string
	CreatedAt       time.Time
}

func NewSecurityUseCase(
	repository AuthSecurityRepository,
	passwordHasher PasswordHasher,
	passwordComparer PasswordComparer,
	tokenIssuer LoginTokenIssuer,
	codeGenerator EmailCodeGenerator,
	codeHasher EmailCodeHasher,
	emailSender EmailCodeSender,
	policy EmailCodePolicy,
	now func() time.Time,
) *SecurityUseCase {
	if now == nil {
		now = time.Now
	}
	return &SecurityUseCase{
		repository:       repository,
		passwordHasher:   passwordHasher,
		passwordComparer: passwordComparer,
		tokenIssuer:      tokenIssuer,
		codeGenerator:    codeGenerator,
		codeHasher:       codeHasher,
		emailSender:      emailSender,
		policy:           policy,
		now:              now,
	}
}

func (uc *SecurityUseCase) SetSettingsReader(settingsReader platformsettings.Reader) {
	uc.settingsReader = settingsReader
}

func (uc *SecurityUseCase) SetXPRecorder(recorder XPRecorder) {
	uc.xpRecorder = recorder
}

func (uc *SecurityUseCase) SendRegisterEmailCode(ctx context.Context, input EmailCodeDispatchInput) (EmailCodeDispatchResult, error) {
	if err := uc.ensureRegistrationEnabled(ctx); err != nil {
		return EmailCodeDispatchResult{}, err
	}
	email, err := uc.normalizeAllowedEmail(input.Email)
	if err != nil {
		return EmailCodeDispatchResult{}, err
	}
	exists, err := uc.repository.EmailExists(ctx, email)
	if err != nil {
		return EmailCodeDispatchResult{}, fmt.Errorf("check register email exists: %w", err)
	}
	if exists {
		return uc.dispatchResult(email, EmailPurposeRegister), nil
	}
	return uc.sendEmailCode(ctx, email, EmailPurposeRegister, input.RequestIP, input.UserAgent)
}

func (uc *SecurityUseCase) RegisterWithEmail(ctx context.Context, input RegisterWithEmailInput) (SecurityAuthResult, error) {
	now := uc.now().UTC()
	if err := uc.ensureRegistrationEnabled(ctx); err != nil {
		return SecurityAuthResult{}, err
	}
	email, err := uc.normalizeAllowedEmail(input.Email)
	if err != nil {
		return SecurityAuthResult{}, err
	}
	username, err := userdomain.NewUsername(input.Username)
	if err != nil {
		return SecurityAuthResult{}, err
	}
	plainPassword, err := userdomain.NewPlainPassword(input.Password)
	if err != nil {
		return SecurityAuthResult{}, err
	}
	passwordHash, err := uc.passwordHasher.Hash(plainPassword)
	if err != nil {
		return SecurityAuthResult{}, fmt.Errorf("hash register password: %w", err)
	}
	user, err := userdomain.NewUser(userdomain.NewGeneratedUserID(), username, passwordHash, now)
	if err != nil {
		return SecurityAuthResult{}, err
	}
	if err := uc.consumeEmailCode(ctx, email, EmailPurposeRegister, input.Code, now); err != nil {
		return SecurityAuthResult{}, err
	}
	if err := uc.repository.CreateUserWithEmail(ctx, *user, email, now, now); err != nil {
		return SecurityAuthResult{}, fmt.Errorf("create email registered user: %w", err)
	}
	if err := uc.recordEvent(ctx, &userIDValue{value: user.ID()}, email, "registered_with_email", input.RequestIP, input.UserAgent, now); err != nil {
		return SecurityAuthResult{}, err
	}
	return uc.issueSecurityAuthResult(user, email, true, false, "", now)
}

func (uc *SecurityUseCase) SendLoginEmailCode(ctx context.Context, input EmailCodeDispatchInput) (EmailCodeDispatchResult, error) {
	email, err := uc.normalizeAllowedEmail(input.Email)
	if err != nil {
		return EmailCodeDispatchResult{}, err
	}
	record, err := uc.repository.FindAuthUserByEmail(ctx, email)
	if err != nil {
		if apperr.IsCode(err, apperr.CodeNotFound) {
			return uc.dispatchResult(email, EmailPurposeLogin), nil
		}
		return EmailCodeDispatchResult{}, fmt.Errorf("find login email user: %w", err)
	}
	if record.User == nil || !record.User.CanLogin() || record.EmailVerifiedAt == nil {
		return uc.dispatchResult(email, EmailPurposeLogin), nil
	}
	return uc.sendEmailCode(ctx, email, EmailPurposeLogin, input.RequestIP, input.UserAgent)
}

func (uc *SecurityUseCase) LoginWithEmailCode(ctx context.Context, input LoginWithEmailCodeInput) (SecurityAuthResult, error) {
	now := uc.now().UTC()
	email, err := uc.normalizeAllowedEmail(input.Email)
	if err != nil {
		return SecurityAuthResult{}, err
	}
	if err := uc.consumeEmailCode(ctx, email, EmailPurposeLogin, input.Code, now); err != nil {
		return SecurityAuthResult{}, err
	}
	record, err := uc.repository.FindAuthUserByEmail(ctx, email)
	if err != nil {
		return SecurityAuthResult{}, apperr.New(apperr.CodeUnauthenticated, "invalid email or code")
	}
	if record.User == nil || !record.User.CanLogin() || record.EmailVerifiedAt == nil {
		if record.User != nil && record.EmailVerifiedAt != nil {
			return SecurityAuthResult{}, loginUserStatusError(record.User)
		}
		return SecurityAuthResult{}, apperr.New(apperr.CodeUnauthenticated, "invalid email or code")
	}
	banned, err := uc.repository.HasActiveAccountBan(ctx, record.User.ID(), now)
	if err != nil {
		return SecurityAuthResult{}, fmt.Errorf("check active account ban: %w", err)
	}
	if banned {
		return SecurityAuthResult{}, apperr.New(apperr.CodeAccountBanned, "account is banned")
	}
	if err := uc.repository.UpdateLastLogin(ctx, record.User.ID(), now, input.RequestIP); err != nil {
		return SecurityAuthResult{}, fmt.Errorf("update email code login time: %w", err)
	}
	if err := uc.recordEvent(ctx, &userIDValue{value: record.User.ID()}, email, "login_code_succeeded", input.RequestIP, input.UserAgent, now); err != nil {
		return SecurityAuthResult{}, err
	}
	if err := grantDailyLoginXP(ctx, uc.xpRecorder, record.User.ID(), now); err != nil {
		return SecurityAuthResult{}, err
	}
	return uc.issueSecurityAuthResult(record.User, email, true, record.IsPlatformStaff, record.PlatformRole, now)
}

func (uc *SecurityUseCase) SendPasswordResetEmailCode(ctx context.Context, input EmailCodeDispatchInput) (EmailCodeDispatchResult, error) {
	email, err := uc.normalizeAllowedEmail(input.Email)
	if err != nil {
		return EmailCodeDispatchResult{}, err
	}
	record, err := uc.repository.FindAuthUserByEmail(ctx, email)
	if err != nil {
		if apperr.IsCode(err, apperr.CodeNotFound) {
			return uc.dispatchResult(email, EmailPurposePasswordReset), nil
		}
		return EmailCodeDispatchResult{}, fmt.Errorf("find password reset email user: %w", err)
	}
	if record.User == nil || !record.User.CanLogin() || record.EmailVerifiedAt == nil {
		return uc.dispatchResult(email, EmailPurposePasswordReset), nil
	}
	return uc.sendEmailCode(ctx, email, EmailPurposePasswordReset, input.RequestIP, input.UserAgent)
}

func (uc *SecurityUseCase) ResetPassword(ctx context.Context, input PasswordResetInput) (PasswordUpdatedResult, error) {
	now := uc.now().UTC()
	email, err := uc.normalizeAllowedEmail(input.Email)
	if err != nil {
		return PasswordUpdatedResult{}, err
	}
	plainPassword, err := userdomain.NewPlainPassword(input.NewPassword)
	if err != nil {
		return PasswordUpdatedResult{}, err
	}
	passwordHash, err := uc.passwordHasher.Hash(plainPassword)
	if err != nil {
		return PasswordUpdatedResult{}, fmt.Errorf("hash reset password: %w", err)
	}
	record, err := uc.repository.FindAuthUserByEmail(ctx, email)
	if err != nil || record.User == nil || !record.User.CanLogin() || record.EmailVerifiedAt == nil {
		return PasswordUpdatedResult{}, apperr.New(apperr.CodeUnauthenticated, "invalid email or code")
	}
	if err := uc.consumeEmailCode(ctx, email, EmailPurposePasswordReset, input.Code, now); err != nil {
		return PasswordUpdatedResult{}, err
	}
	if err := uc.repository.UpdatePasswordByEmail(ctx, email, passwordHash, now); err != nil {
		return PasswordUpdatedResult{}, fmt.Errorf("reset password: %w", err)
	}
	if err := uc.recordEvent(ctx, &userIDValue{value: record.User.ID()}, email, "password_reset_succeeded", input.RequestIP, input.UserAgent, now); err != nil {
		return PasswordUpdatedResult{}, err
	}
	return PasswordUpdatedResult{Updated: true}, nil
}

func (uc *SecurityUseCase) GetSecurity(ctx context.Context, input AccountSecurityInput) (AccountSecurityResult, error) {
	info, err := uc.repository.GetSecurityByUserID(ctx, input.UserID)
	if err != nil {
		return AccountSecurityResult{}, fmt.Errorf("get account security: %w", err)
	}
	return AccountSecurityResult{
		Email:           info.Email,
		EmailVerified:   info.EmailVerifiedAt != nil,
		EmailVerifiedAt: info.EmailVerifiedAt,
		PasswordSet:     strings.TrimSpace(info.PasswordHash.Raw()) != "",
		LastLoginAt:     info.LastLoginAt,
		CreatedAt:       info.CreatedAt,
	}, nil
}

func (uc *SecurityUseCase) SendChangeEmailCode(ctx context.Context, input ChangeEmailCodeInput) (EmailCodeDispatchResult, error) {
	email, err := uc.normalizeAllowedEmail(input.NewEmail)
	if err != nil {
		return EmailCodeDispatchResult{}, err
	}
	exists, err := uc.repository.EmailExists(ctx, email)
	if err != nil {
		return EmailCodeDispatchResult{}, fmt.Errorf("check change email exists: %w", err)
	}
	if exists {
		return EmailCodeDispatchResult{}, apperr.New(apperr.CodeConflict, "email already exists")
	}
	return uc.sendEmailCode(ctx, email, EmailPurposeChangeEmail, input.RequestIP, input.UserAgent)
}

func (uc *SecurityUseCase) ChangeEmail(ctx context.Context, input ChangeEmailInput) (EmailChangeResult, error) {
	now := uc.now().UTC()
	email, err := uc.normalizeAllowedEmail(input.NewEmail)
	if err != nil {
		return EmailChangeResult{}, err
	}
	if err := uc.consumeEmailCode(ctx, email, EmailPurposeChangeEmail, input.Code, now); err != nil {
		return EmailChangeResult{}, err
	}
	info, err := uc.repository.UpdateEmailByUserID(ctx, input.UserID, email, now)
	if err != nil {
		return EmailChangeResult{}, fmt.Errorf("change email: %w", err)
	}
	if err := uc.recordEvent(ctx, &userIDValue{value: input.UserID}, email, "email_changed", input.RequestIP, input.UserAgent, now); err != nil {
		return EmailChangeResult{}, err
	}
	return EmailChangeResult{
		Email:           info.Email,
		EmailVerified:   info.EmailVerifiedAt != nil,
		EmailVerifiedAt: info.EmailVerifiedAt,
	}, nil
}

func (uc *SecurityUseCase) ChangePassword(ctx context.Context, input ChangePasswordInput) (PasswordUpdatedResult, error) {
	now := uc.now().UTC()
	info, err := uc.repository.GetSecurityByUserID(ctx, input.UserID)
	if err != nil {
		return PasswordUpdatedResult{}, fmt.Errorf("get change password user: %w", err)
	}
	currentPassword, err := userdomain.NewPlainPassword(input.CurrentPassword)
	if err != nil {
		return PasswordUpdatedResult{}, err
	}
	if err := uc.passwordComparer.Compare(info.PasswordHash, currentPassword); err != nil {
		return PasswordUpdatedResult{}, apperr.New(apperr.CodeUnauthenticated, "invalid current password")
	}
	newPassword, err := userdomain.NewPlainPassword(input.NewPassword)
	if err != nil {
		return PasswordUpdatedResult{}, err
	}
	passwordHash, err := uc.passwordHasher.Hash(newPassword)
	if err != nil {
		return PasswordUpdatedResult{}, fmt.Errorf("hash changed password: %w", err)
	}
	if err := uc.repository.UpdatePasswordByUserID(ctx, input.UserID, passwordHash, now); err != nil {
		return PasswordUpdatedResult{}, fmt.Errorf("change password: %w", err)
	}
	if err := uc.recordEvent(ctx, &userIDValue{value: input.UserID}, info.Email, "password_changed", input.RequestIP, input.UserAgent, now); err != nil {
		return PasswordUpdatedResult{}, err
	}
	return PasswordUpdatedResult{Updated: true}, nil
}

func (uc *SecurityUseCase) LogoutAll(ctx context.Context, input LogoutAllInput) error {
	now := uc.now().UTC()
	if err := uc.repository.RevokeTokensByUserID(ctx, input.UserID, now); err != nil {
		return fmt.Errorf("logout all sessions: %w", err)
	}
	if err := uc.recordEvent(ctx, &userIDValue{value: input.UserID}, "", "logout_all", input.RequestIP, input.UserAgent, now); err != nil {
		return err
	}
	return nil
}

func (uc *SecurityUseCase) SendDeleteAccountEmailCode(ctx context.Context, input DeleteAccountCodeInput) (EmailCodeDispatchResult, error) {
	email, err := uc.normalizeAllowedEmail(input.Email)
	if err != nil {
		return EmailCodeDispatchResult{}, err
	}
	info, err := uc.repository.GetSecurityByUserID(ctx, input.UserID)
	if err != nil {
		return EmailCodeDispatchResult{}, fmt.Errorf("get delete account user: %w", err)
	}
	if strings.TrimSpace(info.Email) == "" || info.EmailVerifiedAt == nil || info.Email != email {
		return EmailCodeDispatchResult{}, apperr.New(apperr.CodeForbidden, "email does not match current account")
	}
	return uc.sendEmailCode(ctx, email, EmailPurposeDeleteAccount, input.RequestIP, input.UserAgent)
}

func (uc *SecurityUseCase) DeleteAccount(ctx context.Context, input DeleteAccountInput) error {
	now := uc.now().UTC()
	if strings.TrimSpace(input.Confirmation) != "DELETE" {
		return apperr.New(apperr.CodeInvalidArgument, "delete confirmation is invalid")
	}
	hasPassword := strings.TrimSpace(input.CurrentPassword) != ""
	hasCode := strings.TrimSpace(input.Code) != ""
	if !hasPassword && !hasCode {
		return apperr.New(apperr.CodeInvalidArgument, "current password or email code is required")
	}
	info, err := uc.repository.GetSecurityByUserID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("get delete account user: %w", err)
	}
	if err := uc.verifyDeleteAccountChallenge(ctx, info, input, now); err != nil {
		return err
	}
	if err := uc.repository.DeleteAccountByUserID(ctx, input.UserID, now); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	if err := uc.recordEvent(ctx, &userIDValue{value: input.UserID}, info.Email, "account_deleted", input.RequestIP, input.UserAgent, now); err != nil {
		return err
	}
	return nil
}

func (uc *SecurityUseCase) verifyDeleteAccountChallenge(ctx context.Context, info SecurityInfo, input DeleteAccountInput, now time.Time) error {
	hasPassword := strings.TrimSpace(input.CurrentPassword) != ""
	hasCode := strings.TrimSpace(input.Code) != ""
	var passwordErr error
	var codeErr error

	if hasPassword {
		currentPassword, err := userdomain.NewPlainPassword(input.CurrentPassword)
		if err != nil {
			passwordErr = apperr.New(apperr.CodeUnauthenticated, "invalid current password")
		} else if err := uc.passwordComparer.Compare(info.PasswordHash, currentPassword); err != nil {
			passwordErr = apperr.New(apperr.CodeUnauthenticated, "invalid current password")
		} else {
			return nil
		}
	}

	if hasCode {
		if strings.TrimSpace(info.Email) == "" || info.EmailVerifiedAt == nil {
			codeErr = apperr.New(apperr.CodeForbidden, "verified email is required")
		} else if err := uc.consumeEmailCode(ctx, info.Email, EmailPurposeDeleteAccount, input.Code, now); err != nil {
			codeErr = err
		} else {
			return nil
		}
	}

	if hasPassword && !hasCode {
		return passwordErr
	}
	if hasCode && !hasPassword {
		return codeErr
	}
	return apperr.New(apperr.CodeUnauthenticated, "invalid current password or email code")
}

func (uc *SecurityUseCase) sendEmailCode(ctx context.Context, email string, purpose EmailPurpose, requestIP string, userAgent string) (EmailCodeDispatchResult, error) {
	now := uc.now().UTC()
	if latest, err := uc.repository.LatestEmailCode(ctx, email, purpose); err == nil {
		if uc.policy.ResendInterval > 0 && latest.LastSentAt.Add(uc.policy.ResendInterval).After(now) {
			return EmailCodeDispatchResult{}, apperr.New(apperr.CodeForbidden, "email code resend is too frequent")
		}
	} else if !apperr.IsCode(err, apperr.CodeNotFound) {
		return EmailCodeDispatchResult{}, fmt.Errorf("get latest email code: %w", err)
	}
	count, err := uc.repository.CountEmailCodesSince(ctx, email, purpose, now.Add(-24*time.Hour))
	if err != nil {
		return EmailCodeDispatchResult{}, fmt.Errorf("count daily email codes: %w", err)
	}
	if count >= uc.policy.DailyLimit {
		return EmailCodeDispatchResult{}, apperr.New(apperr.CodeForbidden, "email code daily limit exceeded")
	}
	if strings.TrimSpace(requestIP) != "" {
		ipCount, err := uc.repository.CountEmailCodesByIPSince(ctx, requestIP, now.Add(-time.Hour))
		if err != nil {
			return EmailCodeDispatchResult{}, fmt.Errorf("count hourly email codes by ip: %w", err)
		}
		if ipCount >= uc.policy.IPHourlyLimit {
			return EmailCodeDispatchResult{}, apperr.New(apperr.CodeForbidden, "email code ip limit exceeded")
		}
	}
	code, err := uc.codeGenerator.GenerateNumericCode(uc.policy.CodeLength)
	if err != nil {
		return EmailCodeDispatchResult{}, fmt.Errorf("generate email code: %w", err)
	}
	record := EmailCodeRecord{
		ID:           userdomain.NewGeneratedUserID().String(),
		Email:        email,
		Purpose:      purpose,
		CodeHash:     uc.codeHasher.Hash(email, purpose.String(), code),
		Status:       "pending",
		AttemptCount: 0,
		SentCount:    1,
		LastSentAt:   now,
		ExpiresAt:    now.Add(uc.policy.TTL),
		RequestIP:    requestIP,
		UserAgent:    userAgent,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := uc.repository.CreateEmailCode(ctx, record); err != nil {
		return EmailCodeDispatchResult{}, fmt.Errorf("create email code: %w", err)
	}
	if err := uc.emailSender.SendEmailCode(ctx, email, purpose.String(), code, uc.policy.TTL); err != nil {
		if markErr := uc.repository.MarkEmailCodeFailed(ctx, record.ID, now); markErr != nil {
			return EmailCodeDispatchResult{}, fmt.Errorf("send email code: %w; mark failed email code: %v", err, markErr)
		}
		return EmailCodeDispatchResult{}, fmt.Errorf("send email code: %w", err)
	}
	if err := uc.recordEvent(ctx, nil, email, purpose.String()+"_code_sent", requestIP, userAgent, now); err != nil {
		return EmailCodeDispatchResult{}, err
	}
	return uc.dispatchResult(email, purpose), nil
}

func (uc *SecurityUseCase) consumeEmailCode(ctx context.Context, email string, purpose EmailPurpose, rawCode string, now time.Time) error {
	code := strings.TrimSpace(rawCode)
	if code == "" {
		return apperr.New(apperr.CodeInvalidArgument, "email code is required")
	}
	record, err := uc.repository.FindPendingEmailCode(ctx, email, purpose, now)
	if err != nil {
		if apperr.IsCode(err, apperr.CodeNotFound) {
			return apperr.New(apperr.CodeUnauthenticated, "invalid email or code")
		}
		return fmt.Errorf("find pending email code: %w", err)
	}
	if !uc.codeHasher.Compare(email, purpose.String(), code, record.CodeHash) {
		nextAttempts := record.AttemptCount + 1
		status := "pending"
		if nextAttempts >= uc.policy.MaxAttempts {
			status = "blocked"
		}
		if err := uc.repository.MarkEmailCodeAttempt(ctx, record.ID, nextAttempts, status, now); err != nil {
			return fmt.Errorf("mark email code attempt: %w", err)
		}
		return apperr.New(apperr.CodeUnauthenticated, "invalid email or code")
	}
	if err := uc.repository.MarkEmailCodeUsed(ctx, record.ID, now); err != nil {
		return fmt.Errorf("mark email code used: %w", err)
	}
	return nil
}

func (uc *SecurityUseCase) dispatchResult(email string, purpose EmailPurpose) EmailCodeDispatchResult {
	return EmailCodeDispatchResult{
		Email:       email,
		Purpose:     purpose.String(),
		ExpiresIn:   int64(uc.policy.TTL.Seconds()),
		ResendAfter: int64(uc.policy.ResendInterval.Seconds()),
	}
}

func (uc *SecurityUseCase) issueSecurityAuthResult(user *userdomain.User, email string, emailVerified bool, isPlatformStaff bool, platformRole string, now time.Time) (SecurityAuthResult, error) {
	tokenValue, tokenType, expiresIn, err := uc.tokenIssuer.IssueAccessToken(user.ID(), now)
	if err != nil {
		return SecurityAuthResult{}, fmt.Errorf("issue access token: %w", err)
	}
	return SecurityAuthResult{
		AccessToken: tokenValue,
		TokenType:   tokenType,
		ExpiresIn:   int64(expiresIn.Seconds()),
		User: AuthResultUser{
			ID:              user.ID().String(),
			Username:        user.Username().String(),
			Status:          user.Status().String(),
			Email:           email,
			EmailVerified:   emailVerified,
			IsPlatformStaff: isPlatformStaff,
			PlatformRole:    platformRole,
			CreatedAt:       user.CreatedAt(),
		},
	}, nil
}

func (uc *SecurityUseCase) ensureRegistrationEnabled(ctx context.Context) error {
	if uc.settingsReader == nil {
		return nil
	}
	enabled, err := uc.settingsReader.IsEnabled(ctx, platformsettings.RegistrationEnabled)
	if err != nil {
		return fmt.Errorf("read registration setting: %w", err)
	}
	if !enabled {
		return apperr.New(apperr.CodeForbidden, "registration is disabled")
	}
	return nil
}

func (uc *SecurityUseCase) normalizeAllowedEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "email is required")
	}
	if len(email) > 254 || strings.Count(email, "@") != 1 {
		return "", apperr.New(apperr.CodeInvalidArgument, "email is invalid")
	}
	parts := strings.Split(email, "@")
	if parts[0] == "" || parts[1] == "" || strings.ContainsAny(email, " \t\r\n") {
		return "", apperr.New(apperr.CodeInvalidArgument, "email is invalid")
	}
	domain := parts[1]
	for _, allowed := range uc.policy.AllowedDomains {
		if domain == strings.ToLower(strings.TrimSpace(allowed)) {
			return email, nil
		}
	}
	return "", apperr.New(apperr.CodeInvalidArgument, "email domain is not allowed")
}

func (purpose EmailPurpose) String() string {
	return string(purpose)
}

type userIDValue struct {
	value userdomain.UserID
}

func (uc *SecurityUseCase) recordEvent(ctx context.Context, userID *userIDValue, email string, action string, requestIP string, userAgent string, now time.Time) error {
	var id *userdomain.UserID
	if userID != nil {
		value := userID.value
		id = &value
	}
	event := SecurityEvent{
		ID:        userdomain.NewGeneratedUserID().String(),
		UserID:    id,
		Email:     strings.ToLower(strings.TrimSpace(email)),
		Action:    action,
		IP:        requestIP,
		UserAgent: userAgent,
		Metadata:  map[string]any{},
		CreatedAt: now,
	}
	if err := uc.repository.RecordSecurityEvent(ctx, event); err != nil {
		return fmt.Errorf("record security event: %w", err)
	}
	return nil
}

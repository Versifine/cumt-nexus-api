package authusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type LoginInput struct {
	Identifier string
	Username   string
	Password   string
	RequestIP  string
	UserAgent  string
}
type LoginResult struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int64
	User        LoginUser
}
type LoginUser struct {
	ID            string
	Username      string
	Status        string
	Email         string
	EmailVerified bool
	CreatedAt     time.Time
}
type PasswordComparer interface {
	Compare(hash userdomain.PasswordHash, plain userdomain.PlainPassword) error
}
type UserFinder interface {
	FindAuthUserByIdentifier(ctx context.Context, identifier string) (AuthUserRecord, error)
	UpdateLastLogin(ctx context.Context, userID userdomain.UserID, loginAt time.Time, loginIP string) error
	HasActiveAccountBan(ctx context.Context, userID userdomain.UserID, now time.Time) (bool, error)
	CountSecurityEventsSince(ctx context.Context, action string, identity string, requestIP string, since time.Time) (int, error)
	RecordSecurityEvent(ctx context.Context, event SecurityEvent) error
}
type LoginTokenIssuer interface {
	IssueAccessToken(userID userdomain.UserID, now time.Time) (value string, tokenType string, expiresIn time.Duration, err error)
}
type LoginUserCase struct {
	userFinder       UserFinder
	passwordComparer PasswordComparer
	tokenIssuer      LoginTokenIssuer
	xpRecorder       XPRecorder
	now              func() time.Time
}

const (
	loginPasswordFailedAction  = "login_password_failed"
	loginPasswordSuccessAction = "login_password_succeeded"
	loginFailureWindow         = 15 * time.Minute
	loginFailureAccountLimit   = 5
	loginFailureIPLimit        = 20
)

func NewLoginUserCase(userFinder UserFinder, passwordComparer PasswordComparer, tokenIssuer LoginTokenIssuer, now func() time.Time) *LoginUserCase {
	if now == nil {
		now = time.Now
	}

	return &LoginUserCase{
		userFinder:       userFinder,
		passwordComparer: passwordComparer,
		tokenIssuer:      tokenIssuer,
		now:              now,
	}
}

func (uc *LoginUserCase) SetXPRecorder(recorder XPRecorder) {
	uc.xpRecorder = recorder
}

func (uc *LoginUserCase) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	now := uc.now().UTC()

	identifier, err := normalizeLoginIdentifier(input.Identifier, input.Username)
	if err != nil {
		return LoginResult{}, err
	}
	if err := uc.enforceLoginFailureLimit(ctx, identifier, input.RequestIP, now); err != nil {
		return LoginResult{}, err
	}
	plainPassword, err := userdomain.NewPlainPassword(input.Password)
	if err != nil {
		return LoginResult{}, err
	}
	record, err := uc.userFinder.FindAuthUserByIdentifier(ctx, identifier)
	if err != nil {
		if apperr.IsCode(err, apperr.CodeNotFound) {
			if recordErr := uc.recordLoginEvent(ctx, nil, identifier, loginPasswordFailedAction, input.RequestIP, input.UserAgent, now); recordErr != nil {
				return LoginResult{}, recordErr
			}
			return LoginResult{}, apperr.New(apperr.CodeUnauthenticated, "invalid username or password")
		}
		return LoginResult{}, fmt.Errorf("find login user by identifier: %w", err)
	}
	user := record.User
	if user == nil {
		if recordErr := uc.recordLoginEvent(ctx, nil, identifier, loginPasswordFailedAction, input.RequestIP, input.UserAgent, now); recordErr != nil {
			return LoginResult{}, recordErr
		}
		return LoginResult{}, apperr.New(apperr.CodeUnauthenticated, "invalid username or password")
	}
	err = uc.passwordComparer.Compare(user.PasswordHash(), plainPassword)
	if err != nil {
		if recordErr := uc.recordLoginEvent(ctx, &userIDValue{value: user.ID()}, loginEventIdentity(identifier, record.Email), loginPasswordFailedAction, input.RequestIP, input.UserAgent, now); recordErr != nil {
			return LoginResult{}, recordErr
		}
		return LoginResult{}, apperr.New(apperr.CodeUnauthenticated, "invalid username or password")
	}
	if !user.CanLogin() {
		if recordErr := uc.recordLoginEvent(ctx, &userIDValue{value: user.ID()}, loginEventIdentity(identifier, record.Email), loginPasswordFailedAction, input.RequestIP, input.UserAgent, now); recordErr != nil {
			return LoginResult{}, recordErr
		}
		return LoginResult{}, apperr.New(apperr.CodeForbidden, "user is forbidden")
	}
	banned, err := uc.userFinder.HasActiveAccountBan(ctx, user.ID(), now)
	if err != nil {
		return LoginResult{}, fmt.Errorf("check active account ban: %w", err)
	}
	if banned {
		if recordErr := uc.recordLoginEvent(ctx, &userIDValue{value: user.ID()}, loginEventIdentity(identifier, record.Email), loginPasswordFailedAction, input.RequestIP, input.UserAgent, now); recordErr != nil {
			return LoginResult{}, recordErr
		}
		return LoginResult{}, apperr.New(apperr.CodeForbidden, "user is forbidden")
	}
	tokenValue, tokenType, expiresIn, err := uc.tokenIssuer.IssueAccessToken(user.ID(), now)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue login access token: %w", err)
	}
	if err := uc.userFinder.UpdateLastLogin(ctx, user.ID(), now, input.RequestIP); err != nil {
		return LoginResult{}, fmt.Errorf("update login time: %w", err)
	}
	if err := uc.recordLoginEvent(ctx, &userIDValue{value: user.ID()}, loginEventIdentity(identifier, record.Email), loginPasswordSuccessAction, input.RequestIP, input.UserAgent, now); err != nil {
		return LoginResult{}, err
	}
	if err := grantDailyLoginXP(ctx, uc.xpRecorder, user.ID(), now); err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		AccessToken: tokenValue,
		TokenType:   tokenType,
		ExpiresIn:   int64(expiresIn.Seconds()),
		User: LoginUser{
			ID:            user.ID().String(),
			Username:      user.Username().String(),
			Status:        user.Status().String(),
			Email:         record.Email,
			EmailVerified: record.EmailVerifiedAt != nil,
			CreatedAt:     user.CreatedAt(),
		},
	}, nil
}

func (uc *LoginUserCase) enforceLoginFailureLimit(ctx context.Context, identifier string, requestIP string, now time.Time) error {
	since := now.Add(-loginFailureWindow)
	accountCount, err := uc.userFinder.CountSecurityEventsSince(ctx, loginPasswordFailedAction, identifier, "", since)
	if err != nil {
		return fmt.Errorf("count account login failures: %w", err)
	}
	if accountCount >= loginFailureAccountLimit {
		return apperr.New(apperr.CodeForbidden, "too many login attempts")
	}
	if strings.TrimSpace(requestIP) == "" {
		return nil
	}
	ipCount, err := uc.userFinder.CountSecurityEventsSince(ctx, loginPasswordFailedAction, "", requestIP, since)
	if err != nil {
		return fmt.Errorf("count ip login failures: %w", err)
	}
	if ipCount >= loginFailureIPLimit {
		return apperr.New(apperr.CodeForbidden, "too many login attempts")
	}
	return nil
}

func (uc *LoginUserCase) recordLoginEvent(ctx context.Context, userID *userIDValue, identity string, action string, requestIP string, userAgent string, now time.Time) error {
	var id *userdomain.UserID
	if userID != nil {
		value := userID.value
		id = &value
	}
	event := SecurityEvent{
		ID:        userdomain.NewGeneratedUserID().String(),
		UserID:    id,
		Email:     strings.ToLower(strings.TrimSpace(identity)),
		Action:    action,
		IP:        requestIP,
		UserAgent: userAgent,
		Metadata:  map[string]any{},
		CreatedAt: now,
	}
	if err := uc.userFinder.RecordSecurityEvent(ctx, event); err != nil {
		return fmt.Errorf("record login security event: %w", err)
	}
	return nil
}

func loginEventIdentity(identifier string, email string) string {
	if strings.TrimSpace(email) != "" {
		return email
	}
	return identifier
}

func normalizeLoginIdentifier(identifier string, username string) (string, error) {
	raw := strings.TrimSpace(identifier)
	if raw == "" {
		raw = strings.TrimSpace(username)
	}
	raw = strings.ToLower(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "identifier is required")
	}
	if strings.Contains(raw, "@") {
		if len(raw) > 254 || strings.Count(raw, "@") != 1 || strings.ContainsAny(raw, " \t\r\n") {
			return "", apperr.New(apperr.CodeInvalidArgument, "identifier is invalid")
		}
		return raw, nil
	}
	if _, err := userdomain.NewUsername(raw); err != nil {
		return "", err
	}
	return raw, nil
}

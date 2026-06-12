package authusecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	platformsettings "github.com/Versifine/cumt-nexus-api/internal/platform/settings"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type UserCreator interface {
	Create(ctx context.Context, user userdomain.User) error
}
type PasswordHasher interface {
	Hash(plain userdomain.PlainPassword) (userdomain.PasswordHash, error)
}
type AccessTokenIssuer interface {
	IssueAccessToken(userID userdomain.UserID, now time.Time) (value string, tokenType string, expiresIn time.Duration, err error)
}
type RegisterUseCase struct {
	userCreator    UserCreator
	passwordHasher PasswordHasher
	tokenIssuer    AccessTokenIssuer
	settingsReader platformsettings.Reader
	now            func() time.Time
}
type RegisterInput struct {
	Username string
	Password string
}
type RegisterResult struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int64
	User        RegisterUser
}
type RegisterUser struct {
	ID        string
	Username  string
	Status    string
	CreatedAt time.Time
}

func NewRegisterUserCase(userCreator UserCreator, passwordHasher PasswordHasher, tokenIssuer AccessTokenIssuer, now func() time.Time) *RegisterUseCase {
	if now == nil {
		now = time.Now
	}

	return &RegisterUseCase{
		userCreator:    userCreator,
		passwordHasher: passwordHasher,
		tokenIssuer:    tokenIssuer,
		now:            now,
	}
}

func (uc *RegisterUseCase) SetSettingsReader(settingsReader platformsettings.Reader) {
	uc.settingsReader = settingsReader
}

func (uc *RegisterUseCase) Register(ctx context.Context, input RegisterInput) (RegisterResult, error) {
	now := uc.now().UTC()
	if err := uc.ensureRegistrationEnabled(ctx); err != nil {
		return RegisterResult{}, err
	}

	username, err := userdomain.NewUsername(input.Username)
	if err != nil {
		return RegisterResult{}, err
	}

	plainPassword, err := userdomain.NewPlainPassword(input.Password)
	if err != nil {
		return RegisterResult{}, err
	}

	passwordHash, err := uc.passwordHasher.Hash(plainPassword)
	if err != nil {
		return RegisterResult{}, fmt.Errorf("hash register password: %w", err)
	}

	user, err := userdomain.NewUser(userdomain.NewGeneratedUserID(), username, passwordHash, now)
	if err != nil {
		return RegisterResult{}, err
	}

	tokenValue, tokenType, expiresIn, err := uc.tokenIssuer.IssueAccessToken(user.ID(), now)
	if err != nil {
		return RegisterResult{}, fmt.Errorf("issue register access token: %w", err)
	}

	if err := uc.userCreator.Create(ctx, *user); err != nil {
		return RegisterResult{}, fmt.Errorf("create registered user: %w", err)
	}

	return RegisterResult{
		AccessToken: tokenValue,
		TokenType:   tokenType,
		ExpiresIn:   int64(expiresIn.Seconds()),
		User: RegisterUser{
			ID:        user.ID().String(),
			Username:  user.Username().String(),
			Status:    user.Status().String(),
			CreatedAt: user.CreatedAt(),
		},
	}, nil
}

func (uc *RegisterUseCase) ensureRegistrationEnabled(ctx context.Context) error {
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

package authusecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type LoginInput struct {
	Username string
	Password string
}
type LoginResult struct {
	AccessToken string
	TokenType   string
	ExpiresIn   time.Duration
	User        LoginUser
}
type LoginUser struct {
	ID        string
	Username  string
	Status    string
	CreatedAt time.Time
}
type PasswordComparer interface {
	Compare(hash userdomain.PasswordHash, plain userdomain.PlainPassword) error
}
type UserFinder interface {
	FindByUsername(ctx context.Context, username userdomain.Username) (*userdomain.User, error)
}
type LoginTokenIssuer interface {
	IssueAccessToken(userID userdomain.UserID, now time.Time) (value string, tokenType string, expiresIn time.Duration, err error)
}
type LoginUserCase struct {
	userFinder       UserFinder
	passwordComparer PasswordComparer
	tokenIssuer      LoginTokenIssuer
	now              func() time.Time
}

func (uc *LoginUserCase) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	now := uc.now().UTC()

	username, err := userdomain.NewUsername(input.Username)
	if err != nil {
		return LoginResult{}, err
	}
	plainPassword, err := userdomain.NewPlainPassword(input.Password)
	if err != nil {
		return LoginResult{}, err
	}
	user, err := uc.userFinder.FindByUsername(ctx, username)
	if err != nil {
		if apperr.IsCode(err, apperr.CodeNotFound) {
			return LoginResult{}, apperr.New(apperr.CodeUnauthenticated, "invalid username or password")
		}
		return LoginResult{}, fmt.Errorf("find login user by username: %w", err)
	}
	err = uc.passwordComparer.Compare(user.PasswordHash(), plainPassword)
	if err != nil {
		return LoginResult{}, apperr.New(apperr.CodeUnauthenticated, "username or password is incorrect")
	}
	if !user.CanLogin() {
		return LoginResult{}, apperr.New(apperr.CodeForbidden, "user is forbiiden")
	}
	tokenValue, tokenType, expiresIn, err := uc.tokenIssuer.IssueAccessToken(user.ID(), now)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue register access token: %w", err)
	}

	return LoginResult{
		AccessToken: tokenValue,
		TokenType:   tokenType,
		ExpiresIn:   expiresIn,
		User: LoginUser{
			ID:        string(user.ID()),
			Username:  user.Username().String(),
			Status:    user.Status().String(),
			CreatedAt: user.CreatedAt(),
		},
	}, nil
}

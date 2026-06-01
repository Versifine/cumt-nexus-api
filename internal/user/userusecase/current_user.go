package userusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type CurrentUserUseCase struct {
	users CurrentUserFinder
}

type CurrentUserFinder interface {
	FindByID(ctx context.Context, id userdomain.UserID) (*userdomain.User, error)
}

type CurrentUserInput struct {
	UserID userdomain.UserID
}

type CurrentUserResult struct {
	User CurrentUser
}

type CurrentUser struct {
	ID        string
	Username  string
	Status    string
	CreatedAt time.Time
}

func NewCurrentUserUseCase(users CurrentUserFinder) *CurrentUserUseCase {
	return &CurrentUserUseCase{
		users: users,
	}
}

func (uc *CurrentUserUseCase) GetCurrentUser(ctx context.Context, input CurrentUserInput) (CurrentUserResult, error) {
	if strings.TrimSpace(input.UserID.String()) == "" {
		return CurrentUserResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}

	user, err := uc.users.FindByID(ctx, input.UserID)
	if err != nil {
		if apperr.IsCode(err, apperr.CodeNotFound) {
			return CurrentUserResult{}, apperr.New(apperr.CodeUnauthenticated, "invalid token")
		}
		return CurrentUserResult{}, fmt.Errorf("find current user by id: %w", err)
	}

	if !user.CanLogin() {
		return CurrentUserResult{}, apperr.New(apperr.CodeForbidden, "user is forbidden")
	}

	return CurrentUserResult{
		User: CurrentUser{
			ID:        user.ID().String(),
			Username:  user.Username().String(),
			Status:    user.Status().String(),
			CreatedAt: user.CreatedAt(),
		},
	}, nil
}

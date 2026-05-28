package usecase

import (
	"context"

	"github.com/Versifine/cumt-nexus-api/internal/user/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user domain.User)
	FindByID(ctx context.Context, id domain.UserID) (*domain.User, error)
	FindByUsername(ctx context.Context, username domain.Username) (*domain.User, error)
}

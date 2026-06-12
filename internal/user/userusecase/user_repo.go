package userusecase

import (
	"context"

	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type UserRepository interface {
	Create(ctx context.Context, user userdomain.User) error
	FindByID(ctx context.Context, id userdomain.UserID) (*userdomain.User, error)
	FindByUsername(ctx context.Context, username userdomain.Username) (*userdomain.User, error)
	UpdateProfile(ctx context.Context, user userdomain.User) error
}

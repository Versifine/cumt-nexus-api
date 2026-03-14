package service

import (
	"context"
	"cumt-nexus-api/internal/model"
	"errors"

	"gorm.io/gorm"
)

type UserRepo interface {
	FindByID(ctx context.Context, id uint64) (*model.User, error)
	FindByUserName(ctx context.Context, userName string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
}

type ProfileDTO struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Role      int8   `json:"role"`
}

type UserService struct {
	UserRepo UserRepo
}

func NewUserService(userRepo UserRepo) *UserService {
	return &UserService{
		UserRepo: userRepo,
	}
}

func (s *UserService) GetMe(ctx context.Context, userID uint64) (*ProfileDTO, error) {
	user, err := s.UserRepo.FindByID(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ProfileDTO{
		ID:        user.ID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		AvatarURL: user.AvatarURL,
		Role:      user.Role,
	}, nil
}

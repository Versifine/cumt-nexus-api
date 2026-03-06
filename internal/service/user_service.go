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

type UserService struct {
	UserRepo UserRepo
}

func NewUserService(userRepo UserRepo) *UserService {
	return &UserService{
		UserRepo: userRepo,
	}
}

func (s *UserService) GetProfile(ctx context.Context, userID uint64) (string, error) {
	user, err := s.UserRepo.FindByID(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrUserNotFound
	}
	if err != nil {
		return "", err
	}
	return user.Username, nil
}

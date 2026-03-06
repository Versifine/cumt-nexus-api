package repository

import (
	"context"
	"cumt-nexus-api/internal/model"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (ur *UserRepository) FindByID(ctx context.Context, id uint64) (*model.User, error) {
	var u model.User
	err := DB.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}
func (ur *UserRepository) FindByUserName(ctx context.Context, userName string) (*model.User, error) {
	var u model.User
	err := DB.WithContext(ctx).Where("username = ?", userName).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}
func (ur *UserRepository) Create(ctx context.Context, user *model.User) error {
	return DB.WithContext(ctx).Create(user).Error
}

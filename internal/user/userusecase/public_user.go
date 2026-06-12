package userusecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type PublicUserUseCase struct {
	users PublicUserRepository
}

type PublicUserRepository interface {
	FindByUsername(ctx context.Context, username userdomain.Username) (*userdomain.User, error)
	CountVisiblePostsByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID) (int, error)
	CountVisibleCommentsByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID) (int, error)
}

type GetPublicUserInput struct {
	Username string
}

type GetPublicUserResult struct {
	User PublicUser
}

type PublicUser struct {
	ID          string
	Username    string
	DisplayName string
	AvatarURL   string
	BannerURL   string
	Headline    string
	Bio         string
	Badges      []string
	Roles       []string
	Status      string
	Stats       PublicUserStats
	CreatedAt   time.Time
}

type PublicUserStats struct {
	PostCount    int
	CommentCount int
}

func NewPublicUserUseCase(users PublicUserRepository) *PublicUserUseCase {
	return &PublicUserUseCase{
		users: users,
	}
}

func (uc *PublicUserUseCase) GetPublicUser(ctx context.Context, input GetPublicUserInput) (GetPublicUserResult, error) {
	username, err := userdomain.NewUsername(input.Username)
	if err != nil {
		return GetPublicUserResult{}, err
	}

	user, err := uc.users.FindByUsername(ctx, username)
	if err != nil {
		return GetPublicUserResult{}, fmt.Errorf("find public user by username: %w", err)
	}
	if !user.CanLogin() {
		return GetPublicUserResult{}, apperr.New(apperr.CodeNotFound, "user not found")
	}

	postCount, err := uc.users.CountVisiblePostsByAuthorInPublicCommunities(ctx, user.ID())
	if err != nil {
		return GetPublicUserResult{}, fmt.Errorf("count public user posts: %w", err)
	}
	commentCount, err := uc.users.CountVisibleCommentsByAuthorInPublicCommunities(ctx, user.ID())
	if err != nil {
		return GetPublicUserResult{}, fmt.Errorf("count public user comments: %w", err)
	}

	publicUser := PublicUser{
		ID:          user.ID().String(),
		Username:    user.Username().String(),
		DisplayName: publicDisplayName(user),
		AvatarURL:   user.AvatarURL().String(),
		BannerURL:   user.BannerURL().String(),
		Headline:    user.Headline().String(),
		Bio:         user.Bio().String(),
		Badges:      []string{},
		Roles:       []string{},
		Status:      user.Status().String(),
		Stats: PublicUserStats{
			PostCount:    postCount,
			CommentCount: commentCount,
		},
		CreatedAt: user.CreatedAt(),
	}

	return GetPublicUserResult{User: publicUser}, nil
}

func buildPublicUser(user *userdomain.User, postCount int, commentCount int) PublicUser {
	return PublicUser{
		ID:          user.ID().String(),
		Username:    user.Username().String(),
		DisplayName: publicDisplayName(user),
		AvatarURL:   user.AvatarURL().String(),
		BannerURL:   user.BannerURL().String(),
		Headline:    user.Headline().String(),
		Bio:         user.Bio().String(),
		Badges:      []string{},
		Roles:       []string{},
		Status:      user.Status().String(),
		Stats: PublicUserStats{
			PostCount:    postCount,
			CommentCount: commentCount,
		},
		CreatedAt: user.CreatedAt(),
	}
}

func publicDisplayName(user *userdomain.User) string {
	if user.DisplayName().String() != "" {
		return user.DisplayName().String()
	}
	return user.Username().String()
}

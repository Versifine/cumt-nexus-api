package userusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/progression/progressionusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type UpdateProfileUseCase struct {
	users ProfileRepository
	now   func() time.Time
}

type ProfileRepository interface {
	FindByID(ctx context.Context, id userdomain.UserID) (*userdomain.User, error)
	UpdateProfile(ctx context.Context, user userdomain.User) error
	CountVisiblePostsByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID) (int, error)
	CountVisibleCommentsByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID) (int, error)
	CountFollowers(ctx context.Context, userID userdomain.UserID) (int, error)
	CountFollowing(ctx context.Context, userID userdomain.UserID) (int, error)
}

type UpdateProfileInput struct {
	UserID      userdomain.UserID
	DisplayName *string
	AvatarURL   *string
	BannerURL   *string
	Headline    *string
	Bio         *string
}

type UpdateProfileResult struct {
	User PublicUser
}

func NewUpdateProfileUseCase(users ProfileRepository, now func() time.Time) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{
		users: users,
		now:   now,
	}
}

func (uc *UpdateProfileUseCase) UpdateProfile(ctx context.Context, input UpdateProfileInput) (UpdateProfileResult, error) {
	if strings.TrimSpace(input.UserID.String()) == "" {
		return UpdateProfileResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if input.DisplayName == nil && input.AvatarURL == nil && input.BannerURL == nil && input.Headline == nil && input.Bio == nil {
		return UpdateProfileResult{}, apperr.New(apperr.CodeInvalidArgument, "at least one profile field is required")
	}

	user, err := uc.users.FindByID(ctx, input.UserID)
	if err != nil {
		if apperr.IsCode(err, apperr.CodeNotFound) {
			return UpdateProfileResult{}, apperr.New(apperr.CodeUnauthenticated, "invalid token")
		}
		return UpdateProfileResult{}, fmt.Errorf("find current user by id: %w", err)
	}
	if !user.CanLogin() {
		return UpdateProfileResult{}, apperr.New(apperr.CodeForbidden, "user is forbidden")
	}

	displayName := user.DisplayName()
	if input.DisplayName != nil {
		displayName, err = userdomain.NewDisplayName(*input.DisplayName)
		if err != nil {
			return UpdateProfileResult{}, err
		}
	}

	avatarURL := user.AvatarURL()
	if input.AvatarURL != nil {
		avatarURL, err = userdomain.NewAvatarURL(*input.AvatarURL)
		if err != nil {
			return UpdateProfileResult{}, err
		}
	}

	bannerURL := user.BannerURL()
	if input.BannerURL != nil {
		bannerURL, err = userdomain.NewBannerURL(*input.BannerURL)
		if err != nil {
			return UpdateProfileResult{}, err
		}
	}

	headline := user.Headline()
	if input.Headline != nil {
		headline, err = userdomain.NewHeadline(*input.Headline)
		if err != nil {
			return UpdateProfileResult{}, err
		}
	}

	bio := user.Bio()
	if input.Bio != nil {
		bio, err = userdomain.NewBio(*input.Bio)
		if err != nil {
			return UpdateProfileResult{}, err
		}
	}

	if err := user.UpdateProfile(displayName, avatarURL, bannerURL, headline, bio, uc.now()); err != nil {
		return UpdateProfileResult{}, err
	}
	if err := uc.users.UpdateProfile(ctx, *user); err != nil {
		return UpdateProfileResult{}, fmt.Errorf("update user profile: %w", err)
	}

	postCount, err := uc.users.CountVisiblePostsByAuthorInPublicCommunities(ctx, user.ID())
	if err != nil {
		return UpdateProfileResult{}, fmt.Errorf("count public user posts: %w", err)
	}
	commentCount, err := uc.users.CountVisibleCommentsByAuthorInPublicCommunities(ctx, user.ID())
	if err != nil {
		return UpdateProfileResult{}, fmt.Errorf("count public user comments: %w", err)
	}
	followerCount, err := uc.users.CountFollowers(ctx, user.ID())
	if err != nil {
		return UpdateProfileResult{}, fmt.Errorf("count user followers: %w", err)
	}
	followingCount, err := uc.users.CountFollowing(ctx, user.ID())
	if err != nil {
		return UpdateProfileResult{}, fmt.Errorf("count followed users: %w", err)
	}

	return UpdateProfileResult{
		User: buildPublicUser(
			user,
			postCount,
			commentCount,
			followerCount,
			followingCount,
			false,
			progressionusecase.BuildProgression(progressionusecase.ProgressionRecord{UserID: user.ID().String()}),
			selfDMCapability(),
		),
	}, nil
}

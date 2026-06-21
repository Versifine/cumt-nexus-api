package userusecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/progression/progressionusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type PublicUserUseCase struct {
	users        PublicUserRepository
	progression  PublicProgressionReader
	dmCapability PublicDMCapabilityReader
	now          func() time.Time
}

type PublicUserRepository interface {
	FindByUsername(ctx context.Context, username userdomain.Username) (*userdomain.User, error)
	CountVisiblePostsByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID) (int, error)
	CountVisibleCommentsByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID) (int, error)
	CountFollowers(ctx context.Context, userID userdomain.UserID) (int, error)
	CountFollowing(ctx context.Context, userID userdomain.UserID) (int, error)
	FindPlatformAccess(ctx context.Context, userID userdomain.UserID) (PlatformAccess, error)
	IsFollowing(ctx context.Context, followerID userdomain.UserID, followingID userdomain.UserID) (bool, error)
	FollowUser(ctx context.Context, followerID userdomain.UserID, followingID userdomain.UserID, now time.Time) error
	DeleteUserFollow(ctx context.Context, followerID userdomain.UserID, followingID userdomain.UserID) error
	ListFollowedActiveUsers(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]userdomain.User, error)
}

type GetPublicUserInput struct {
	Username string
	ViewerID userdomain.UserID
}

type FollowUserInput struct {
	Username string
	UserID   userdomain.UserID
}

type DeleteUserFollowInput struct {
	Username string
	UserID   userdomain.UserID
}

type ListFollowedUsersInput struct {
	UserID userdomain.UserID
	Limit  int
	Offset int
}

type PublicProgressionReader interface {
	GetPublicProgression(ctx context.Context, userID userdomain.UserID) (progressionusecase.Progression, error)
}

type PublicDMCapabilityReader interface {
	GetPublicUserDMCapability(ctx context.Context, viewerID userdomain.UserID, targetID userdomain.UserID) (PublicUserDMCapability, error)
}

type GetPublicUserResult struct {
	User PublicUser
}

type FollowUserResult struct{}

type DeleteUserFollowResult struct{}

type ListFollowedUsersResult struct {
	Users      []PublicUser
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type PublicUser struct {
	ID                string
	Username          string
	DisplayName       string
	AvatarURL         string
	BannerURL         string
	Headline          string
	Bio               string
	Badges            []string
	Roles             []string
	Status            string
	IsPlatformStaff   bool
	PlatformRole      string
	Stats             PublicUserStats
	Progression       PublicUserProgression
	DMCapability      PublicUserDMCapability
	ViewerIsFollowing bool
	CreatedAt         time.Time
}

type PublicUserDMCapability struct {
	CanStart             bool
	RequiresRequest      bool
	Reason               *string
	DirectConversationID *string
	ViewerRelation       string
}

type PublicUserProgression struct {
	Level          int
	LevelName      string
	XPTotal        int
	CurrentLevelXP int
	NextLevelXP    *int
	LevelProgress  float64
	ActiveTitle    *progressionusecase.TitleSummary
	TitlesCount    int
}

type PublicUserStats struct {
	PostCount      int
	CommentCount   int
	FollowerCount  int
	FollowingCount int
}

func NewPublicUserUseCase(users PublicUserRepository) *PublicUserUseCase {
	return &PublicUserUseCase{
		users: users,
		now:   time.Now,
	}
}

func (uc *PublicUserUseCase) SetProgressionReader(progression PublicProgressionReader) {
	uc.progression = progression
}

func (uc *PublicUserUseCase) SetDMCapabilityReader(reader PublicDMCapabilityReader) {
	uc.dmCapability = reader
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

	publicUser, err := uc.buildPublicUser(ctx, *user, input.ViewerID)
	if err != nil {
		return GetPublicUserResult{}, err
	}

	return GetPublicUserResult{User: publicUser}, nil
}

func (uc *PublicUserUseCase) FollowUser(ctx context.Context, input FollowUserInput) (FollowUserResult, error) {
	if isBlankUserID(input.UserID) {
		return FollowUserResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	target, err := uc.findActiveUserByUsername(ctx, input.Username)
	if err != nil {
		return FollowUserResult{}, err
	}
	if target.ID() == input.UserID {
		return FollowUserResult{}, apperr.New(apperr.CodeInvalidArgument, "can't follow yourself")
	}
	if err := uc.users.FollowUser(ctx, input.UserID, target.ID(), uc.now().UTC()); err != nil {
		return FollowUserResult{}, fmt.Errorf("follow user: %w", err)
	}
	return FollowUserResult{}, nil
}

func (uc *PublicUserUseCase) DeleteUserFollow(ctx context.Context, input DeleteUserFollowInput) (DeleteUserFollowResult, error) {
	if isBlankUserID(input.UserID) {
		return DeleteUserFollowResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	target, err := uc.findActiveUserByUsername(ctx, input.Username)
	if err != nil {
		return DeleteUserFollowResult{}, err
	}
	if err := uc.users.DeleteUserFollow(ctx, input.UserID, target.ID()); err != nil {
		return DeleteUserFollowResult{}, fmt.Errorf("delete user follow: %w", err)
	}
	return DeleteUserFollowResult{}, nil
}

func (uc *PublicUserUseCase) ListFollowedUsers(ctx context.Context, input ListFollowedUsersInput) (ListFollowedUsersResult, error) {
	if isBlankUserID(input.UserID) {
		return ListFollowedUsersResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	limit, offset, err := normalizePublicUserListPagination(input.Limit, input.Offset)
	if err != nil {
		return ListFollowedUsersResult{}, err
	}
	users, err := uc.users.ListFollowedActiveUsers(ctx, input.UserID, limit+1, offset)
	if err != nil {
		return ListFollowedUsersResult{}, fmt.Errorf("list followed users: %w", err)
	}
	users, hasMore := trimUsersPage(users, limit)

	result := ListFollowedUsersResult{
		Users:      make([]PublicUser, 0, len(users)),
		Limit:      limit,
		Offset:     offset,
		NextOffset: nextOffset(offset, len(users)),
		HasMore:    hasMore,
	}
	for _, user := range users {
		publicUser, err := uc.buildPublicUser(ctx, user, input.UserID)
		if err != nil {
			return ListFollowedUsersResult{}, err
		}
		result.Users = append(result.Users, publicUser)
	}
	return result, nil
}

func (uc *PublicUserUseCase) findActiveUserByUsername(ctx context.Context, rawUsername string) (*userdomain.User, error) {
	username, err := userdomain.NewUsername(rawUsername)
	if err != nil {
		return nil, err
	}
	user, err := uc.users.FindByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("find user by username: %w", err)
	}
	if !user.CanLogin() {
		return nil, apperr.New(apperr.CodeNotFound, "user not found")
	}
	return user, nil
}

func (uc *PublicUserUseCase) buildPublicUser(ctx context.Context, user userdomain.User, viewerID userdomain.UserID) (PublicUser, error) {
	postCount, err := uc.users.CountVisiblePostsByAuthorInPublicCommunities(ctx, user.ID())
	if err != nil {
		return PublicUser{}, fmt.Errorf("count public user posts: %w", err)
	}
	commentCount, err := uc.users.CountVisibleCommentsByAuthorInPublicCommunities(ctx, user.ID())
	if err != nil {
		return PublicUser{}, fmt.Errorf("count public user comments: %w", err)
	}
	followerCount, err := uc.users.CountFollowers(ctx, user.ID())
	if err != nil {
		return PublicUser{}, fmt.Errorf("count user followers: %w", err)
	}
	followingCount, err := uc.users.CountFollowing(ctx, user.ID())
	if err != nil {
		return PublicUser{}, fmt.Errorf("count followed users: %w", err)
	}
	access, err := uc.users.FindPlatformAccess(ctx, user.ID())
	if err != nil {
		return PublicUser{}, fmt.Errorf("find public user platform access: %w", err)
	}
	viewerIsFollowing := false
	if !isBlankUserID(viewerID) && viewerID != user.ID() {
		viewerIsFollowing, err = uc.users.IsFollowing(ctx, viewerID, user.ID())
		if err != nil {
			return PublicUser{}, fmt.Errorf("check viewer user follow: %w", err)
		}
	}
	progression := progressionusecase.BuildProgression(progressionusecase.ProgressionRecord{UserID: user.ID().String()})
	if uc.progression != nil {
		progression, err = uc.progression.GetPublicProgression(ctx, user.ID())
		if err != nil {
			return PublicUser{}, err
		}
	}
	dmCapability, err := uc.buildDMCapability(ctx, viewerID, user.ID())
	if err != nil {
		return PublicUser{}, err
	}
	return buildPublicUser(&user, postCount, commentCount, followerCount, followingCount, viewerIsFollowing, access, progression, dmCapability), nil
}

func buildPublicUser(user *userdomain.User, postCount int, commentCount int, followerCount int, followingCount int, viewerIsFollowing bool, access PlatformAccess, progression progressionusecase.Progression, dmCapability PublicUserDMCapability) PublicUser {
	return PublicUser{
		ID:              user.ID().String(),
		Username:        user.Username().String(),
		DisplayName:     publicDisplayName(user),
		AvatarURL:       user.AvatarURL().String(),
		BannerURL:       user.BannerURL().String(),
		Headline:        user.Headline().String(),
		Bio:             user.Bio().String(),
		Badges:          []string{},
		Roles:           []string{},
		Status:          user.Status().String(),
		IsPlatformStaff: access.IsPlatformStaff,
		PlatformRole:    access.PlatformRole,
		Stats: PublicUserStats{
			PostCount:      postCount,
			CommentCount:   commentCount,
			FollowerCount:  followerCount,
			FollowingCount: followingCount,
		},
		Progression:       toPublicUserProgression(progression),
		DMCapability:      dmCapability,
		ViewerIsFollowing: viewerIsFollowing,
		CreatedAt:         user.CreatedAt(),
	}
}

func (uc *PublicUserUseCase) buildDMCapability(ctx context.Context, viewerID userdomain.UserID, targetID userdomain.UserID) (PublicUserDMCapability, error) {
	if isBlankUserID(viewerID) {
		reason := "unauthenticated"
		return PublicUserDMCapability{CanStart: false, RequiresRequest: true, Reason: &reason, ViewerRelation: "none"}, nil
	}
	if viewerID == targetID {
		reason := "self"
		return PublicUserDMCapability{CanStart: false, Reason: &reason, ViewerRelation: "self"}, nil
	}
	if uc.dmCapability == nil {
		reason := "unavailable"
		return PublicUserDMCapability{CanStart: false, Reason: &reason, ViewerRelation: "none"}, nil
	}
	return uc.dmCapability.GetPublicUserDMCapability(ctx, viewerID, targetID)
}

func selfDMCapability() PublicUserDMCapability {
	reason := "self"
	return PublicUserDMCapability{CanStart: false, Reason: &reason, ViewerRelation: "self"}
}

func publicDisplayName(user *userdomain.User) string {
	if user.DisplayName().String() != "" {
		return user.DisplayName().String()
	}
	return user.Username().String()
}

func toPublicUserProgression(progression progressionusecase.Progression) PublicUserProgression {
	return PublicUserProgression{
		Level:          progression.Level,
		LevelName:      progression.LevelName,
		XPTotal:        progression.XPTotal,
		CurrentLevelXP: progression.CurrentLevelXP,
		NextLevelXP:    progression.NextLevelXP,
		LevelProgress:  progression.LevelProgress,
		ActiveTitle:    progression.ActiveTitle,
		TitlesCount:    progression.TitlesCount,
	}
}

func isBlankUserID(id userdomain.UserID) bool {
	return id.String() == ""
}

func normalizePublicUserListPagination(limit int, offset int) (int, int, error) {
	if limit < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "limit must be non-negative")
	}
	if offset < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "offset must be non-negative")
	}
	if limit == 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	return limit, offset, nil
}

func trimUsersPage(items []userdomain.User, limit int) ([]userdomain.User, bool) {
	if len(items) <= limit {
		return items, false
	}
	return items[:limit], true
}

func nextOffset(offset int, itemCount int) int {
	return offset + itemCount
}

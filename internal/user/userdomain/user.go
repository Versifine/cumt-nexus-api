package userdomain

import (
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
)

type User struct {
	id           UserID
	username     Username
	passwordHash PasswordHash
	displayName  DisplayName
	avatarURL    AvatarURL
	bannerURL    BannerURL
	headline     Headline
	bio          Bio
	status       UserStatus
	createdAt    time.Time
	updatedAt    time.Time
}

func NewUser(id UserID, username Username, passwordHash PasswordHash, now time.Time) (*User, error) {
	if now.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "user created time can't be zero")
	}

	return &User{
		id:           id,
		username:     username,
		passwordHash: passwordHash,
		displayName:  "",
		avatarURL:    "",
		bannerURL:    "",
		headline:     "",
		bio:          "",
		status:       active,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

func RehydrateUser(id UserID, username Username, passwordHash PasswordHash, status UserStatus, createdAt time.Time, updatedAt time.Time) (*User, error) {
	return RehydrateUserWithProfile(id, username, passwordHash, "", "", "", "", "", status, createdAt, updatedAt)
}

func RehydrateUserWithProfile(
	id UserID,
	username Username,
	passwordHash PasswordHash,
	displayName DisplayName,
	avatarURL AvatarURL,
	bannerURL BannerURL,
	headline Headline,
	bio Bio,
	status UserStatus,
	createdAt time.Time,
	updatedAt time.Time,
) (*User, error) {
	if createdAt.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "user created time can't be zero")
	}

	if updatedAt.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "user updated time can't be zero")
	}

	if updatedAt.Before(createdAt) {
		return nil, apperr.New(apperr.CodeInvalidArgument, "user updated time can't be before created time")
	}

	return &User{
		id:           id,
		username:     username,
		passwordHash: passwordHash,
		displayName:  displayName,
		avatarURL:    avatarURL,
		bannerURL:    bannerURL,
		headline:     headline,
		bio:          bio,
		status:       status,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}, nil
}
func (u *User) ID() UserID {
	return u.id
}
func (u *User) Username() Username {
	return u.username
}
func (u *User) PasswordHash() PasswordHash {
	return u.passwordHash
}
func (u *User) DisplayName() DisplayName {
	return u.displayName
}
func (u *User) AvatarURL() AvatarURL {
	return u.avatarURL
}
func (u *User) BannerURL() BannerURL {
	return u.bannerURL
}
func (u *User) Headline() Headline {
	return u.headline
}
func (u *User) Bio() Bio {
	return u.bio
}
func (u *User) Status() UserStatus {
	return u.status
}
func (u *User) CreatedAt() time.Time {
	return u.createdAt
}
func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

func (u *User) UpdateProfile(displayName DisplayName, avatarURL AvatarURL, bannerURL BannerURL, headline Headline, bio Bio, now time.Time) error {
	if now.IsZero() {
		return apperr.New(apperr.CodeInvalidArgument, "user updated time can't be zero")
	}
	if now.Before(u.createdAt) {
		return apperr.New(apperr.CodeInvalidArgument, "user updated time can't be before created time")
	}

	u.displayName = displayName
	u.avatarURL = avatarURL
	u.bannerURL = bannerURL
	u.headline = headline
	u.bio = bio
	u.updatedAt = now
	return nil
}

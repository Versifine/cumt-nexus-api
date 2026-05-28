package domain

import (
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
)

type User struct {
	id           UserID
	username     Username
	passwordHash PasswordHash
	status       UserStatus
	createdAt    time.Time
	updatedAt    time.Time
}

// 这里要不要做检验???
func NewUser(id UserID, username Username, passwordHash PasswordHash, now time.Time) (*User, error) {
	if now.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "user created time can't be zero")
	}

	return &User{
		id:           id,
		username:     username,
		passwordHash: passwordHash,
		status:       active,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

func RehydrateUser(id UserID, username Username, passwordHash PasswordHash, status UserStatus, createdAt time.Time, updatedAt time.Time) (*User, error) {
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
func (u *User) Status() UserStatus {
	return u.status
}
func (u *User) CreatedAt() time.Time {
	return u.createdAt
}
func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

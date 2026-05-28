package domain

import "github.com/Versifine/cumt-nexus-api/internal/apperr"

type UserStatus string

const (
	active   UserStatus = "active"
	disabled UserStatus = "disabled"
)

func (us UserStatus) String() string {
	return string(us)
}

func NewUserStatus(status string) (UserStatus, error) {
	switch status {
	case "active":
		return active, nil
	case "disabled":
		return disabled, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "invalid user status")
	}
}

func (u User) CanLogin() bool {
	switch u.status {
	case active:
		return true
	case disabled:
		return false
	default:
		return false
	}
}

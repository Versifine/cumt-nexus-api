package userdomain

import (
	"strings"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/google/uuid"
)

type UserID string

func NewUserID(raw string) (UserID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "user id is required")
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return "", apperr.New(apperr.CodeInvalidArgument, "user id is invalid")
	}
	return UserID(parsed.String()), nil
}
func NewUserIDFromUUID(id uuid.UUID) UserID {
	return UserID(id.String())
}
func NewGeneratedUserID() UserID {
	return UserID(uuid.NewString())
}

func (id UserID) String() string {
	return string(id)
}

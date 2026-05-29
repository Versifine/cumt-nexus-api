package userdomain

import (
	"strings"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
)

type Username string

func NewUsername(raw string) (Username, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.ToLower(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "username is required")
	}
	if len(raw) < 3 || len(raw) > 32 {
		return "", apperr.New(apperr.CodeInvalidArgument, "username is invalid")
	}
	if !isAllowedString(raw) {
		return "", apperr.New(apperr.CodeInvalidArgument, "username is invalid")
	}

	return Username(raw), nil
}
func isAllowedString(s string) bool {
	allowed := "abcdefghijklmnopqrstuvwxyz0123456789_"

	for _, ch := range s {
		if !strings.ContainsRune(allowed, ch) {
			return false
		}
	}

	return true
}
func (un Username) String() string {
	return string(un)
}

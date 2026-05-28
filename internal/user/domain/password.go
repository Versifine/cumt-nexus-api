package domain

import (
	"strings"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
)

type PlainPassword string
type PasswordHash string

func (p PlainPassword) String() string {
	return string(p)
}
func (p PasswordHash) Raw() string {
	return string(p)
}

func NewPlainPassword(raw string) (PlainPassword, error) {
	if strings.TrimSpace(raw) == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "password is required")
	}
	if len(raw) < 8 {
		return "", apperr.New(apperr.CodeInvalidArgument, "password is invalid")
	}
	return PlainPassword(raw), nil
}

func NewPasswordHash(hash string) (PasswordHash, error) {
	if strings.TrimSpace(hash) == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "password hash can't be empty")
	}

	return PasswordHash(hash), nil
}

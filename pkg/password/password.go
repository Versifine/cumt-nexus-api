package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmptyPassword    = errors.New("empty password")
	ErrPasswordTooLong  = errors.New("password too long")
	ErrInvalidHash      = errors.New("invalid hash")
	ErrPasswordMismatch = errors.New("password mismatch")
)

func HashPassword(plain string) (string, error) {
	if len(plain) == 0 {
		return "", ErrEmptyPassword
	}
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		if errors.Is(err, bcrypt.ErrPasswordTooLong) {
			return "", ErrPasswordTooLong
		}
		return "", fmt.Errorf("bcrypt generate failed: %w", err)
	}
	return string(hashBytes), nil
}

func CheckPassword(hash, plain string) error {
	if hash == "" {
		return ErrInvalidHash
	}
	if plain == "" {
		return ErrEmptyPassword
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return ErrPasswordMismatch
	case errors.Is(err, bcrypt.ErrHashTooShort):
		return ErrInvalidHash
	default:
		return fmt.Errorf("bcrypt compare failed: %w", err)
	}
}

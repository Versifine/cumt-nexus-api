package password

import (
	"fmt"

	userdomain "github.com/Versifine/cumt-nexus-api/internal/user/domain"
	"golang.org/x/crypto/bcrypt"
)

type BcryptHasher struct {
	cost int
}

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{
		cost: bcrypt.DefaultCost,
	}
}
func NewBcryptHasherWithCost(cost int) *BcryptHasher {
	return &BcryptHasher{
		cost: cost,
	}
}
func (h *BcryptHasher) Hash(plain userdomain.PlainPassword) (userdomain.PasswordHash, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(plain.String()), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	return userdomain.PasswordHash(hashedBytes), nil
}

func (h *BcryptHasher) Compare(hash userdomain.PasswordHash, plain userdomain.PlainPassword) error {
	err := bcrypt.CompareHashAndPassword(
		[]byte(hash.Raw()),
		[]byte(plain.String()),
	)
	return err
}

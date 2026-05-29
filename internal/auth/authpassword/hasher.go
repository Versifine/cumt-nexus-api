package authpassword

import (
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type Hasher interface {
	Hash(plain userdomain.PlainPassword) (userdomain.PasswordHash, error)
	Compare(hash userdomain.PasswordHash, plain userdomain.PlainPassword) error
}

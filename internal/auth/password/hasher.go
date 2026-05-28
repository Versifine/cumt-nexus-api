package password

import (
	userdomain "github.com/Versifine/cumt-nexus-api/internal/user/domain"
)

type Hasher interface {
	Hash(plain userdomain.PlainPassword) (userdomain.PasswordHash, error)
	Compare(hash userdomain.PasswordHash, plain userdomain.PlainPassword) error
}

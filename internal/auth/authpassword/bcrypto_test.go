package authpassword

import (
	"testing"

	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"golang.org/x/crypto/bcrypt"
)

func TestBcryptHasherHashAndCompare(t *testing.T) {
	hasher := NewBcryptHasherWithCost(bcrypt.MinCost)
	plain, err := userdomain.NewPlainPassword("password123")
	if err != nil {
		t.Fatalf("NewPlainPassword returned error: %v", err)
	}

	hash, err := hasher.Hash(plain)
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}
	if hash.Raw() == plain.String() {
		t.Fatal("password hash should not equal plain password")
	}

	if err := hasher.Compare(hash, plain); err != nil {
		t.Fatalf("Compare with correct password returned error: %v", err)
	}

	wrong, err := userdomain.NewPlainPassword("wrongpass")
	if err != nil {
		t.Fatalf("NewPlainPassword wrong returned error: %v", err)
	}
	if err := hasher.Compare(hash, wrong); err == nil {
		t.Fatal("Compare with wrong password should fail")
	}
}

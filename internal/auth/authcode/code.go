package authcode

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) GenerateNumericCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("code length must be positive")
	}
	var builder strings.Builder
	builder.Grow(length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("generate code digit: %w", err)
		}
		builder.WriteByte(byte('0' + n.Int64()))
	}
	return builder.String(), nil
}

type Hasher struct {
	secret string
}

func NewHasher(secret string) *Hasher {
	return &Hasher{
		secret: secret,
	}
}

func (h *Hasher) Hash(email string, purpose string, code string) string {
	mac := hmac.New(sha256.New, []byte(h.secret))
	_, _ = mac.Write([]byte(strings.ToLower(strings.TrimSpace(email))))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(strings.TrimSpace(purpose)))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(strings.TrimSpace(code)))
	return hex.EncodeToString(mac.Sum(nil))
}

func (h *Hasher) Compare(email string, purpose string, code string, hash string) bool {
	got := h.Hash(email, purpose, code)
	return hmac.Equal([]byte(got), []byte(strings.TrimSpace(hash)))
}

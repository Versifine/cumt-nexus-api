package userdomain

import (
	"strings"
	"testing"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
)

func TestProfileURLRejectsOverlongValue(t *testing.T) {
	overlongURL := "https://example.com/" + strings.Repeat("a", maxProfileURLLength)

	if _, err := NewAvatarURL(overlongURL); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for long avatar url, got %v", err)
	}
	if _, err := NewBannerURL(overlongURL); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for long banner url, got %v", err)
	}
}

func TestPlainPasswordRejectsOverlongValue(t *testing.T) {
	if _, err := NewPlainPassword(strings.Repeat("a", MaxPlainPasswordBytes+1)); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for long password, got %v", err)
	}
}

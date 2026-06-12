package settings

import (
	"context"
	"strings"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
)

const (
	RegistrationEnabled = "registration_enabled"
	PostingEnabled      = "posting_enabled"
	UploadEnabled       = "upload_enabled"
)

var KnownKeys = []string{
	RegistrationEnabled,
	PostingEnabled,
	UploadEnabled,
}

type Reader interface {
	IsEnabled(ctx context.Context, key string) (bool, error)
}

func NormalizeKey(raw string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	for _, known := range KnownKeys {
		if key == known {
			return key, nil
		}
	}
	return "", apperr.New(apperr.CodeInvalidArgument, "admin setting key is invalid")
}

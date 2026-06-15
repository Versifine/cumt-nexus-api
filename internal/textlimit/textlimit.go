package textlimit

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
)

func TrimmedRequiredMaxRunes(raw string, field string, maxRunes int) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, field+" is required")
	}
	if utf8.RuneCountInString(value) > maxRunes {
		return "", apperr.New(apperr.CodeInvalidArgument, fmt.Sprintf("%s must be at most %d characters", field, maxRunes))
	}
	return value, nil
}

func TrimmedOptionalMaxRunes(raw string, field string, maxRunes int) (string, error) {
	value := strings.TrimSpace(raw)
	if utf8.RuneCountInString(value) > maxRunes {
		return "", apperr.New(apperr.CodeInvalidArgument, fmt.Sprintf("%s must be at most %d characters", field, maxRunes))
	}
	return value, nil
}

func EnsureMaxBytes(raw string, field string, maxBytes int) error {
	if len(raw) > maxBytes {
		return apperr.New(apperr.CodeInvalidArgument, fmt.Sprintf("%s must be at most %d bytes", field, maxBytes))
	}
	return nil
}

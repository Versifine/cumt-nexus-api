package userdomain

import (
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/textlimit"
)

const (
	maxDisplayNameLength = 40
	maxHeadlineLength    = 80
	maxBioLength         = 300
	maxProfileURLLength  = 2048
)

type DisplayName string

func NewDisplayName(raw string) (DisplayName, error) {
	raw = strings.TrimSpace(raw)
	if utf8.RuneCountInString(raw) > maxDisplayNameLength {
		return "", apperr.New(apperr.CodeInvalidArgument, "display name must be at most 40 characters")
	}
	return DisplayName(raw), nil
}

func (value DisplayName) String() string {
	return string(value)
}

type Headline string

func NewHeadline(raw string) (Headline, error) {
	raw = strings.TrimSpace(raw)
	if utf8.RuneCountInString(raw) > maxHeadlineLength {
		return "", apperr.New(apperr.CodeInvalidArgument, "headline must be at most 80 characters")
	}
	return Headline(raw), nil
}

func (value Headline) String() string {
	return string(value)
}

type Bio string

func NewBio(raw string) (Bio, error) {
	raw = strings.TrimSpace(raw)
	if utf8.RuneCountInString(raw) > maxBioLength {
		return "", apperr.New(apperr.CodeInvalidArgument, "bio must be at most 300 characters")
	}
	return Bio(raw), nil
}

func (value Bio) String() string {
	return string(value)
}

type AvatarURL string

func NewAvatarURL(raw string) (AvatarURL, error) {
	raw = strings.TrimSpace(raw)
	if err := validateOptionalHTTPURL(raw); err != nil {
		return "", apperr.New(apperr.CodeInvalidArgument, "avatar url must be a valid http or https url")
	}

	return AvatarURL(raw), nil
}

func (value AvatarURL) String() string {
	return string(value)
}

type BannerURL string

func NewBannerURL(raw string) (BannerURL, error) {
	raw = strings.TrimSpace(raw)
	if err := validateOptionalHTTPURL(raw); err != nil {
		return "", apperr.New(apperr.CodeInvalidArgument, "banner url must be a valid http or https url")
	}

	return BannerURL(raw), nil
}

func (value BannerURL) String() string {
	return string(value)
}

func validateOptionalHTTPURL(raw string) error {
	if raw == "" {
		return nil
	}
	if err := textlimit.EnsureMaxBytes(raw, "url", maxProfileURLLength); err != nil {
		return err
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return apperr.New(apperr.CodeInvalidArgument, "url must be absolute")
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return nil
	default:
		return apperr.New(apperr.CodeInvalidArgument, "url must use http or https")
	}
}

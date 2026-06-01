package communitydomain

import (
	"regexp"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

var communitySlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,31}$`)

type CommunityID string

func NewCommunityID(raw string) (CommunityID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "community id is required")
	}

	parsed, err := uuid.Parse(raw)
	if err != nil {
		return "", apperr.New(apperr.CodeInvalidArgument, "community id is invalid")
	}

	return CommunityID(parsed.String()), nil
}

func NewGeneratedCommunityID() CommunityID {
	return CommunityID(uuid.NewString())
}

func (id CommunityID) String() string {
	return string(id)
}

type CommunitySlug string

func NewCommunitySlug(raw string) (CommunitySlug, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "community slug is required")
	}
	if !communitySlugPattern.MatchString(raw) {
		return "", apperr.New(apperr.CodeInvalidArgument, "community slug is invalid")
	}

	return CommunitySlug(raw), nil
}

func (slug CommunitySlug) String() string {
	return string(slug)
}

type CommunityName string

func NewCommunityName(raw string) (CommunityName, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "community name is required")
	}

	return CommunityName(raw), nil
}

func (name CommunityName) String() string {
	return string(name)
}

type CommunityDescription string

func NewCommunityDescription(raw string) CommunityDescription {
	return CommunityDescription(strings.TrimSpace(raw))
}

func (description CommunityDescription) String() string {
	return string(description)
}

type CommunityKind string

const (
	CommunityKindSystem      CommunityKind = "system"
	CommunityKindUserCreated CommunityKind = "user_created"
)

func NewCommunityKind(raw string) (CommunityKind, error) {
	switch CommunityKind(strings.TrimSpace(strings.ToLower(raw))) {
	case CommunityKindSystem:
		return CommunityKindSystem, nil
	case CommunityKindUserCreated:
		return CommunityKindUserCreated, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "community kind is invalid")
	}
}

func (kind CommunityKind) String() string {
	return string(kind)
}

type CommunityStatus string

const (
	CommunityStatusActive    CommunityStatus = "active"
	CommunityStatusSuspended CommunityStatus = "suspended"
	CommunityStatusArchived  CommunityStatus = "archived"
)

func NewCommunityStatus(raw string) (CommunityStatus, error) {
	switch CommunityStatus(strings.TrimSpace(strings.ToLower(raw))) {
	case CommunityStatusActive:
		return CommunityStatusActive, nil
	case CommunityStatusSuspended:
		return CommunityStatusSuspended, nil
	case CommunityStatusArchived:
		return CommunityStatusArchived, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "community status is invalid")
	}
}

func (status CommunityStatus) String() string {
	return string(status)
}

type CommunityVisibility string

const CommunityVisibilityPublic CommunityVisibility = "public"

func NewCommunityVisibility(raw string) (CommunityVisibility, error) {
	switch CommunityVisibility(strings.TrimSpace(strings.ToLower(raw))) {
	case CommunityVisibilityPublic:
		return CommunityVisibilityPublic, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "community visibility is invalid")
	}
}

func (visibility CommunityVisibility) String() string {
	return string(visibility)
}

type Community struct {
	id          CommunityID
	slug        CommunitySlug
	name        CommunityName
	description CommunityDescription
	kind        CommunityKind
	status      CommunityStatus
	visibility  CommunityVisibility
	createdBy   *userdomain.UserID
	createdAt   time.Time
	updatedAt   time.Time
}

func NewSystemCommunity(id CommunityID, slug CommunitySlug, name CommunityName, description CommunityDescription, now time.Time) (*Community, error) {
	return RehydrateCommunity(id, slug, name, description, CommunityKindSystem, CommunityStatusActive, CommunityVisibilityPublic, nil, now, now)
}

func NewUserCreatedCommunity(id CommunityID, slug CommunitySlug, name CommunityName, description CommunityDescription, createdBy userdomain.UserID, now time.Time) (*Community, error) {
	return RehydrateCommunity(id, slug, name, description, CommunityKindUserCreated, CommunityStatusActive, CommunityVisibilityPublic, cloneUserID(createdBy), now, now)
}

func RehydrateCommunity(
	id CommunityID,
	slug CommunitySlug,
	name CommunityName,
	description CommunityDescription,
	kind CommunityKind,
	status CommunityStatus,
	visibility CommunityVisibility,
	createdBy *userdomain.UserID,
	createdAt time.Time,
	updatedAt time.Time,
) (*Community, error) {
	if strings.TrimSpace(id.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community id is required")
	}
	if strings.TrimSpace(slug.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community slug is required")
	}
	if strings.TrimSpace(name.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community name is required")
	}
	if _, err := NewCommunityKind(kind.String()); err != nil {
		return nil, err
	}
	if _, err := NewCommunityStatus(status.String()); err != nil {
		return nil, err
	}
	if _, err := NewCommunityVisibility(visibility.String()); err != nil {
		return nil, err
	}
	if kind == CommunityKindUserCreated {
		if createdBy == nil || isZeroUserID(*createdBy) {
			return nil, apperr.New(apperr.CodeInvalidArgument, "user-created community creator is required")
		}
	}
	if err := validateCreatedUpdated("community", createdAt, updatedAt); err != nil {
		return nil, err
	}

	return &Community{
		id:          id,
		slug:        slug,
		name:        name,
		description: description,
		kind:        kind,
		status:      status,
		visibility:  visibility,
		createdBy:   cloneOptionalUserID(createdBy),
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}, nil
}

func (community *Community) ID() CommunityID {
	return community.id
}

func (community *Community) Slug() CommunitySlug {
	return community.slug
}

func (community *Community) Name() CommunityName {
	return community.name
}

func (community *Community) Description() CommunityDescription {
	return community.description
}

func (community *Community) Kind() CommunityKind {
	return community.kind
}

func (community *Community) Status() CommunityStatus {
	return community.status
}

func (community *Community) Visibility() CommunityVisibility {
	return community.visibility
}

func (community *Community) CreatedBy() (userdomain.UserID, bool) {
	if community.createdBy == nil {
		return "", false
	}

	return *community.createdBy, true
}

func (community *Community) CreatedAt() time.Time {
	return community.createdAt
}

func (community *Community) UpdatedAt() time.Time {
	return community.updatedAt
}

func validateCreatedUpdated(entity string, createdAt time.Time, updatedAt time.Time) error {
	if createdAt.IsZero() {
		return apperr.New(apperr.CodeInvalidArgument, entity+" created time can't be zero")
	}
	if updatedAt.IsZero() {
		return apperr.New(apperr.CodeInvalidArgument, entity+" updated time can't be zero")
	}
	if updatedAt.Before(createdAt) {
		return apperr.New(apperr.CodeInvalidArgument, entity+" updated time can't be before created time")
	}

	return nil
}

func cloneUserID(id userdomain.UserID) *userdomain.UserID {
	copied := id
	return &copied
}

func cloneOptionalUserID(id *userdomain.UserID) *userdomain.UserID {
	if id == nil {
		return nil
	}

	return cloneUserID(*id)
}

func isZeroUserID(id userdomain.UserID) bool {
	return strings.TrimSpace(id.String()) == ""
}

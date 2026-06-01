package communitydomain

import (
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type MembershipRole string

const (
	MembershipRoleOwner     MembershipRole = "owner"
	MembershipRoleModerator MembershipRole = "moderator"
	MembershipRoleMember    MembershipRole = "member"
)

func NewMembershipRole(raw string) (MembershipRole, error) {
	switch MembershipRole(strings.TrimSpace(strings.ToLower(raw))) {
	case MembershipRoleOwner:
		return MembershipRoleOwner, nil
	case MembershipRoleModerator:
		return MembershipRoleModerator, nil
	case MembershipRoleMember:
		return MembershipRoleMember, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "community membership role is invalid")
	}
}

func (role MembershipRole) String() string {
	return string(role)
}

type MembershipStatus string

const (
	MembershipStatusActive MembershipStatus = "active"
	MembershipStatusLeft   MembershipStatus = "left"
	MembershipStatusBanned MembershipStatus = "banned"
)

func NewMembershipStatus(raw string) (MembershipStatus, error) {
	switch MembershipStatus(strings.TrimSpace(strings.ToLower(raw))) {
	case MembershipStatusActive:
		return MembershipStatusActive, nil
	case MembershipStatusLeft:
		return MembershipStatusLeft, nil
	case MembershipStatusBanned:
		return MembershipStatusBanned, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "community membership status is invalid")
	}
}

func (status MembershipStatus) String() string {
	return string(status)
}

type CommunityMembership struct {
	communityID CommunityID
	userID      userdomain.UserID
	role        MembershipRole
	status      MembershipStatus
	createdAt   time.Time
	updatedAt   time.Time
}

func NewCommunityMembership(communityID CommunityID, userID userdomain.UserID, role MembershipRole, now time.Time) (*CommunityMembership, error) {
	return RehydrateCommunityMembership(communityID, userID, role, MembershipStatusActive, now, now)
}

func RehydrateCommunityMembership(
	communityID CommunityID,
	userID userdomain.UserID,
	role MembershipRole,
	status MembershipStatus,
	createdAt time.Time,
	updatedAt time.Time,
) (*CommunityMembership, error) {
	if strings.TrimSpace(communityID.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community id is required")
	}
	if isZeroUserID(userID) {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community membership user id is required")
	}
	if _, err := NewMembershipRole(role.String()); err != nil {
		return nil, err
	}
	if _, err := NewMembershipStatus(status.String()); err != nil {
		return nil, err
	}
	if err := validateCreatedUpdated("community membership", createdAt, updatedAt); err != nil {
		return nil, err
	}

	return &CommunityMembership{
		communityID: communityID,
		userID:      userID,
		role:        role,
		status:      status,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}, nil
}

func (membership *CommunityMembership) CommunityID() CommunityID {
	return membership.communityID
}

func (membership *CommunityMembership) UserID() userdomain.UserID {
	return membership.userID
}

func (membership *CommunityMembership) Role() MembershipRole {
	return membership.role
}

func (membership *CommunityMembership) Status() MembershipStatus {
	return membership.status
}

func (membership *CommunityMembership) CreatedAt() time.Time {
	return membership.createdAt
}

func (membership *CommunityMembership) UpdatedAt() time.Time {
	return membership.updatedAt
}

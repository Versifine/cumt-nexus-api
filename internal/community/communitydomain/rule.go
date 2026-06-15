package communitydomain

import (
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/textlimit"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

const (
	MaxCommunityRuleTitleRunes = 80
	MaxCommunityRuleBodyRunes  = 500
)

type CommunityRuleID string

func NewCommunityRuleID(raw string) (CommunityRuleID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "community rule id is required")
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return "", apperr.New(apperr.CodeInvalidArgument, "community rule id is invalid")
	}
	return CommunityRuleID(parsed.String()), nil
}

func NewGeneratedCommunityRuleID() CommunityRuleID {
	return CommunityRuleID(uuid.NewString())
}

func (id CommunityRuleID) String() string {
	return string(id)
}

type CommunityRuleTitle string

func NewCommunityRuleTitle(raw string) (CommunityRuleTitle, error) {
	value, err := textlimit.TrimmedRequiredMaxRunes(raw, "community rule title", MaxCommunityRuleTitleRunes)
	if err != nil {
		return "", err
	}
	return CommunityRuleTitle(value), nil
}

func (title CommunityRuleTitle) String() string {
	return string(title)
}

type CommunityRuleBody string

func NewCommunityRuleBody(raw string) (CommunityRuleBody, error) {
	value, err := textlimit.TrimmedOptionalMaxRunes(raw, "community rule body", MaxCommunityRuleBodyRunes)
	if err != nil {
		return "", err
	}
	return CommunityRuleBody(value), nil
}

func (body CommunityRuleBody) String() string {
	return string(body)
}

type CommunityRulePosition int

func NewCommunityRulePosition(value int) (CommunityRulePosition, error) {
	if value < 0 {
		return 0, apperr.New(apperr.CodeInvalidArgument, "community rule position must be non-negative")
	}
	return CommunityRulePosition(value), nil
}

func (position CommunityRulePosition) Int() int {
	return int(position)
}

type CommunityRule struct {
	id          CommunityRuleID
	communityID CommunityID
	title       CommunityRuleTitle
	body        CommunityRuleBody
	position    CommunityRulePosition
	createdBy   userdomain.UserID
	updatedBy   userdomain.UserID
	createdAt   time.Time
	updatedAt   time.Time
}

func NewCommunityRule(id CommunityRuleID, communityID CommunityID, title CommunityRuleTitle, body CommunityRuleBody, position CommunityRulePosition, actorID userdomain.UserID, now time.Time) (*CommunityRule, error) {
	return RehydrateCommunityRule(id, communityID, title, body, position, actorID, actorID, now, now)
}

func RehydrateCommunityRule(
	id CommunityRuleID,
	communityID CommunityID,
	title CommunityRuleTitle,
	body CommunityRuleBody,
	position CommunityRulePosition,
	createdBy userdomain.UserID,
	updatedBy userdomain.UserID,
	createdAt time.Time,
	updatedAt time.Time,
) (*CommunityRule, error) {
	if strings.TrimSpace(id.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community rule id is required")
	}
	if strings.TrimSpace(communityID.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community rule community id is required")
	}
	if strings.TrimSpace(title.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community rule title is required")
	}
	if _, err := NewCommunityRulePosition(position.Int()); err != nil {
		return nil, err
	}
	if strings.TrimSpace(createdBy.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community rule creator is required")
	}
	if strings.TrimSpace(updatedBy.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community rule updater is required")
	}
	if err := validateCreatedUpdated("community rule", createdAt, updatedAt); err != nil {
		return nil, err
	}

	return &CommunityRule{
		id:          id,
		communityID: communityID,
		title:       title,
		body:        body,
		position:    position,
		createdBy:   createdBy,
		updatedBy:   updatedBy,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}, nil
}

func (rule *CommunityRule) ID() CommunityRuleID {
	return rule.id
}

func (rule *CommunityRule) CommunityID() CommunityID {
	return rule.communityID
}

func (rule *CommunityRule) Title() CommunityRuleTitle {
	return rule.title
}

func (rule *CommunityRule) Body() CommunityRuleBody {
	return rule.body
}

func (rule *CommunityRule) Position() CommunityRulePosition {
	return rule.position
}

func (rule *CommunityRule) CreatedBy() userdomain.UserID {
	return rule.createdBy
}

func (rule *CommunityRule) UpdatedBy() userdomain.UserID {
	return rule.updatedBy
}

func (rule *CommunityRule) CreatedAt() time.Time {
	return rule.createdAt
}

func (rule *CommunityRule) UpdatedAt() time.Time {
	return rule.updatedAt
}

func (rule *CommunityRule) Update(title CommunityRuleTitle, body CommunityRuleBody, position CommunityRulePosition, actorID userdomain.UserID, now time.Time) error {
	if strings.TrimSpace(actorID.String()) == "" {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if now.IsZero() {
		return apperr.New(apperr.CodeInvalidArgument, "community rule updated time can't be zero")
	}
	if now.Before(rule.createdAt) {
		return apperr.New(apperr.CodeInvalidArgument, "community rule updated time can't be before created time")
	}

	rule.title = title
	rule.body = body
	rule.position = position
	rule.updatedBy = actorID
	rule.updatedAt = now
	return nil
}

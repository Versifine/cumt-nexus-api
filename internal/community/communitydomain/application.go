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
	MaxApplicationReasonRunes = 500
	MaxRejectReasonRunes      = 500
)

type CommunityApplicationID string

func NewCommunityApplicationID(raw string) (CommunityApplicationID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "community application id is required")
	}

	parsed, err := uuid.Parse(raw)
	if err != nil {
		return "", apperr.New(apperr.CodeInvalidArgument, "community application id is invalid")
	}

	return CommunityApplicationID(parsed.String()), nil
}

func NewGeneratedCommunityApplicationID() CommunityApplicationID {
	return CommunityApplicationID(uuid.NewString())
}

func (id CommunityApplicationID) String() string {
	return string(id)
}

type ApplicationReason string

func NewApplicationReason(raw string) (ApplicationReason, error) {
	value, err := textlimit.TrimmedRequiredMaxRunes(raw, "community application reason", MaxApplicationReasonRunes)
	if err != nil {
		return "", err
	}

	return ApplicationReason(value), nil
}

func (reason ApplicationReason) String() string {
	return string(reason)
}

type RejectReason string

func NewRejectReason(raw string) (RejectReason, error) {
	value, err := textlimit.TrimmedRequiredMaxRunes(raw, "community application reject reason", MaxRejectReasonRunes)
	if err != nil {
		return "", err
	}

	return RejectReason(value), nil
}

func (reason RejectReason) String() string {
	return string(reason)
}

type ApplicationStatus string

const (
	ApplicationStatusPending  ApplicationStatus = "pending"
	ApplicationStatusApproved ApplicationStatus = "approved"
	ApplicationStatusRejected ApplicationStatus = "rejected"
	ApplicationStatusCanceled ApplicationStatus = "canceled"
)

func NewApplicationStatus(raw string) (ApplicationStatus, error) {
	switch ApplicationStatus(strings.TrimSpace(strings.ToLower(raw))) {
	case ApplicationStatusPending:
		return ApplicationStatusPending, nil
	case ApplicationStatusApproved:
		return ApplicationStatusApproved, nil
	case ApplicationStatusRejected:
		return ApplicationStatusRejected, nil
	case ApplicationStatusCanceled:
		return ApplicationStatusCanceled, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "community application status is invalid")
	}
}

func (status ApplicationStatus) String() string {
	return string(status)
}

type CommunityApplication struct {
	id            CommunityApplicationID
	applicantID   userdomain.UserID
	requestedSlug CommunitySlug
	requestedName CommunityName
	reason        ApplicationReason
	status        ApplicationStatus
	reviewedBy    *userdomain.UserID
	reviewedAt    *time.Time
	rejectReason  *RejectReason
	createdAt     time.Time
	updatedAt     time.Time
}

func NewCommunityApplication(
	id CommunityApplicationID,
	applicantID userdomain.UserID,
	requestedSlug CommunitySlug,
	requestedName CommunityName,
	reason ApplicationReason,
	now time.Time,
) (*CommunityApplication, error) {
	return RehydrateCommunityApplication(id, applicantID, requestedSlug, requestedName, reason, ApplicationStatusPending, nil, nil, nil, now, now)
}

func RehydrateCommunityApplication(
	id CommunityApplicationID,
	applicantID userdomain.UserID,
	requestedSlug CommunitySlug,
	requestedName CommunityName,
	reason ApplicationReason,
	status ApplicationStatus,
	reviewedBy *userdomain.UserID,
	reviewedAt *time.Time,
	rejectReason *RejectReason,
	createdAt time.Time,
	updatedAt time.Time,
) (*CommunityApplication, error) {
	if strings.TrimSpace(id.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community application id is required")
	}
	if isZeroUserID(applicantID) {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community application applicant id is required")
	}
	if strings.TrimSpace(requestedSlug.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community application requested slug is required")
	}
	if strings.TrimSpace(requestedName.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community application requested name is required")
	}
	if strings.TrimSpace(reason.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "community application reason is required")
	}
	if _, err := NewApplicationStatus(status.String()); err != nil {
		return nil, err
	}
	if err := validateCreatedUpdated("community application", createdAt, updatedAt); err != nil {
		return nil, err
	}
	if err := validateReviewFields(status, reviewedBy, reviewedAt, rejectReason); err != nil {
		return nil, err
	}

	return &CommunityApplication{
		id:            id,
		applicantID:   applicantID,
		requestedSlug: requestedSlug,
		requestedName: requestedName,
		reason:        reason,
		status:        status,
		reviewedBy:    cloneOptionalUserID(reviewedBy),
		reviewedAt:    cloneOptionalTime(reviewedAt),
		rejectReason:  cloneOptionalRejectReason(rejectReason),
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}, nil
}

func (application *CommunityApplication) Approve(reviewedBy userdomain.UserID, reviewedAt time.Time) error {
	if application.status != ApplicationStatusPending {
		return apperr.New(apperr.CodeConflict, "community application is not pending")
	}
	if isZeroUserID(reviewedBy) {
		return apperr.New(apperr.CodeInvalidArgument, "community application reviewer is required")
	}
	if reviewedAt.IsZero() {
		return apperr.New(apperr.CodeInvalidArgument, "community application reviewed time can't be zero")
	}
	if reviewedAt.Before(application.createdAt) {
		return apperr.New(apperr.CodeInvalidArgument, "community application reviewed time can't be before created time")
	}

	application.status = ApplicationStatusApproved
	application.reviewedBy = cloneUserID(reviewedBy)
	application.reviewedAt = cloneTime(reviewedAt)
	application.rejectReason = nil
	application.updatedAt = reviewedAt
	return nil
}

func (application *CommunityApplication) Reject(reviewedBy userdomain.UserID, reviewedAt time.Time, rejectReason RejectReason) error {
	if application.status != ApplicationStatusPending {
		return apperr.New(apperr.CodeConflict, "community application is not pending")
	}
	if isZeroUserID(reviewedBy) {
		return apperr.New(apperr.CodeInvalidArgument, "community application reviewer is required")
	}
	if reviewedAt.IsZero() {
		return apperr.New(apperr.CodeInvalidArgument, "community application reviewed time can't be zero")
	}
	if reviewedAt.Before(application.createdAt) {
		return apperr.New(apperr.CodeInvalidArgument, "community application reviewed time can't be before created time")
	}
	if strings.TrimSpace(rejectReason.String()) == "" {
		return apperr.New(apperr.CodeInvalidArgument, "community application reject reason is required")
	}

	application.status = ApplicationStatusRejected
	application.reviewedBy = cloneUserID(reviewedBy)
	application.reviewedAt = cloneTime(reviewedAt)
	application.rejectReason = cloneRejectReason(rejectReason)
	application.updatedAt = reviewedAt
	return nil
}

func (application *CommunityApplication) Cancel(now time.Time) error {
	if application.status != ApplicationStatusPending {
		return apperr.New(apperr.CodeConflict, "community application is not pending")
	}
	if now.IsZero() {
		return apperr.New(apperr.CodeInvalidArgument, "community application updated time can't be zero")
	}
	if now.Before(application.createdAt) {
		return apperr.New(apperr.CodeInvalidArgument, "community application updated time can't be before created time")
	}

	application.status = ApplicationStatusCanceled
	application.reviewedBy = nil
	application.reviewedAt = nil
	application.rejectReason = nil
	application.updatedAt = now
	return nil
}

func (application *CommunityApplication) ID() CommunityApplicationID {
	return application.id
}

func (application *CommunityApplication) ApplicantID() userdomain.UserID {
	return application.applicantID
}

func (application *CommunityApplication) RequestedSlug() CommunitySlug {
	return application.requestedSlug
}

func (application *CommunityApplication) RequestedName() CommunityName {
	return application.requestedName
}

func (application *CommunityApplication) Reason() ApplicationReason {
	return application.reason
}

func (application *CommunityApplication) Status() ApplicationStatus {
	return application.status
}

func (application *CommunityApplication) ReviewedBy() (userdomain.UserID, bool) {
	if application.reviewedBy == nil {
		return "", false
	}

	return *application.reviewedBy, true
}

func (application *CommunityApplication) ReviewedAt() (time.Time, bool) {
	if application.reviewedAt == nil {
		return time.Time{}, false
	}

	return *application.reviewedAt, true
}

func (application *CommunityApplication) RejectReason() (RejectReason, bool) {
	if application.rejectReason == nil {
		return "", false
	}

	return *application.rejectReason, true
}

func (application *CommunityApplication) CreatedAt() time.Time {
	return application.createdAt
}

func (application *CommunityApplication) UpdatedAt() time.Time {
	return application.updatedAt
}

func validateReviewFields(status ApplicationStatus, reviewedBy *userdomain.UserID, reviewedAt *time.Time, rejectReason *RejectReason) error {
	switch status {
	case ApplicationStatusPending, ApplicationStatusCanceled:
		if reviewedBy != nil || reviewedAt != nil || rejectReason != nil {
			return apperr.New(apperr.CodeInvalidArgument, "community application review fields are invalid")
		}
	case ApplicationStatusApproved:
		if reviewedBy == nil || reviewedAt == nil || rejectReason != nil {
			return apperr.New(apperr.CodeInvalidArgument, "community application review fields are invalid")
		}
		if isZeroUserID(*reviewedBy) || reviewedAt.IsZero() {
			return apperr.New(apperr.CodeInvalidArgument, "community application review fields are invalid")
		}
	case ApplicationStatusRejected:
		if reviewedBy == nil || reviewedAt == nil || rejectReason == nil {
			return apperr.New(apperr.CodeInvalidArgument, "community application review fields are invalid")
		}
		if isZeroUserID(*reviewedBy) || reviewedAt.IsZero() || strings.TrimSpace(rejectReason.String()) == "" {
			return apperr.New(apperr.CodeInvalidArgument, "community application review fields are invalid")
		}
	default:
		return apperr.New(apperr.CodeInvalidArgument, "community application status is invalid")
	}

	return nil
}

func cloneTime(value time.Time) *time.Time {
	copied := value
	return &copied
}

func cloneOptionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	return cloneTime(*value)
}

func cloneRejectReason(reason RejectReason) *RejectReason {
	copied := reason
	return &copied
}

func cloneOptionalRejectReason(reason *RejectReason) *RejectReason {
	if reason == nil {
		return nil
	}

	return cloneRejectReason(*reason)
}

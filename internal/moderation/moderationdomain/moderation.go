package moderationdomain

import (
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/textlimit"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

const MaxReasonRunes = 500

type ContentReportID string

func NewContentReportID(raw string) (ContentReportID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "content report id is required")
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return "", apperr.New(apperr.CodeInvalidArgument, "content report id is invalid")
	}
	return ContentReportID(parsed.String()), nil
}

func NewGeneratedContentReportID() ContentReportID {
	return ContentReportID(uuid.NewString())
}

func (id ContentReportID) String() string {
	return string(id)
}

type ModerationActionID string

func NewModerationActionID(raw string) (ModerationActionID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "moderation action id is required")
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return "", apperr.New(apperr.CodeInvalidArgument, "moderation action id is invalid")
	}
	return ModerationActionID(parsed.String()), nil
}

func NewGeneratedModerationActionID() ModerationActionID {
	return ModerationActionID(uuid.NewString())
}

func (id ModerationActionID) String() string {
	return string(id)
}

type TargetType string

const (
	TargetTypePost    TargetType = "post"
	TargetTypeComment TargetType = "comment"
)

func NewTargetType(raw string) (TargetType, error) {
	switch TargetType(strings.TrimSpace(strings.ToLower(raw))) {
	case TargetTypePost:
		return TargetTypePost, nil
	case TargetTypeComment:
		return TargetTypeComment, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "moderation target type is invalid")
	}
}

func (targetType TargetType) String() string {
	return string(targetType)
}

type Target struct {
	targetType TargetType
	postID     *postdomain.PostID
	commentID  *commentdomain.CommentID
}

func NewPostTarget(postID postdomain.PostID) (Target, error) {
	if strings.TrimSpace(postID.String()) == "" {
		return Target{}, apperr.New(apperr.CodeInvalidArgument, "moderation post id is required")
	}
	return Target{
		targetType: TargetTypePost,
		postID:     clonePostID(postID),
	}, nil
}

func NewCommentTarget(commentID commentdomain.CommentID) (Target, error) {
	if strings.TrimSpace(commentID.String()) == "" {
		return Target{}, apperr.New(apperr.CodeInvalidArgument, "moderation comment id is required")
	}
	return Target{
		targetType: TargetTypeComment,
		commentID:  cloneCommentID(commentID),
	}, nil
}

func (target Target) Type() TargetType {
	return target.targetType
}

func (target Target) PostID() (postdomain.PostID, bool) {
	if target.postID == nil {
		return "", false
	}
	return *target.postID, true
}

func (target Target) CommentID() (commentdomain.CommentID, bool) {
	if target.commentID == nil {
		return "", false
	}
	return *target.commentID, true
}

func validateTarget(target Target) error {
	switch target.targetType {
	case TargetTypePost:
		if target.postID == nil || strings.TrimSpace(target.postID.String()) == "" || target.commentID != nil {
			return apperr.New(apperr.CodeInvalidArgument, "moderation post target is invalid")
		}
	case TargetTypeComment:
		if target.commentID == nil || strings.TrimSpace(target.commentID.String()) == "" || target.postID != nil {
			return apperr.New(apperr.CodeInvalidArgument, "moderation comment target is invalid")
		}
	default:
		return apperr.New(apperr.CodeInvalidArgument, "moderation target type is invalid")
	}
	return nil
}

type Reason string

func NewReason(raw string) (Reason, error) {
	value, err := textlimit.TrimmedRequiredMaxRunes(raw, "moderation reason", MaxReasonRunes)
	if err != nil {
		return "", err
	}
	return Reason(value), nil
}

func (reason Reason) String() string {
	return string(reason)
}

type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"
)

func NewReportStatus(raw string) (ReportStatus, error) {
	switch ReportStatus(strings.TrimSpace(strings.ToLower(raw))) {
	case ReportStatusPending:
		return ReportStatusPending, nil
	case ReportStatusResolved:
		return ReportStatusResolved, nil
	case ReportStatusDismissed:
		return ReportStatusDismissed, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "content report status is invalid")
	}
}

func (status ReportStatus) String() string {
	return string(status)
}

type ActionType string

const (
	ActionTypeRemove        ActionType = "remove"
	ActionTypeApprove       ActionType = "approve"
	ActionTypeSpam          ActionType = "spam"
	ActionTypeIgnoreReports ActionType = "ignore_reports"
	ActionTypeLock          ActionType = "lock"
	ActionTypePin           ActionType = "pin"
	ActionTypeMarkNSFW      ActionType = "mark_nsfw"
	ActionTypeMarkSpoiler   ActionType = "mark_spoiler"
	ActionTypeSetFlair      ActionType = "set_flair"
)

func NewActionType(raw string) (ActionType, error) {
	switch ActionType(strings.TrimSpace(strings.ToLower(raw))) {
	case ActionTypeRemove:
		return ActionTypeRemove, nil
	case ActionTypeApprove:
		return ActionTypeApprove, nil
	case ActionTypeSpam:
		return ActionTypeSpam, nil
	case ActionTypeIgnoreReports:
		return ActionTypeIgnoreReports, nil
	case ActionTypeLock:
		return ActionTypeLock, nil
	case ActionTypePin:
		return ActionTypePin, nil
	case ActionTypeMarkNSFW:
		return ActionTypeMarkNSFW, nil
	case ActionTypeMarkSpoiler:
		return ActionTypeMarkSpoiler, nil
	case ActionTypeSetFlair:
		return ActionTypeSetFlair, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "moderation action type is invalid")
	}
}

func (actionType ActionType) String() string {
	return string(actionType)
}

type ContentReport struct {
	id         ContentReportID
	target     Target
	reporterID userdomain.UserID
	reason     Reason
	status     ReportStatus
	reviewedBy *userdomain.UserID
	reviewedAt *time.Time
	createdAt  time.Time
	updatedAt  time.Time
}

func NewContentReport(id ContentReportID, target Target, reporterID userdomain.UserID, reason Reason, now time.Time) (*ContentReport, error) {
	return RehydrateContentReport(id, target, reporterID, reason, ReportStatusPending, nil, nil, now, now)
}

func RehydrateContentReport(
	id ContentReportID,
	target Target,
	reporterID userdomain.UserID,
	reason Reason,
	status ReportStatus,
	reviewedBy *userdomain.UserID,
	reviewedAt *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) (*ContentReport, error) {
	if strings.TrimSpace(id.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "content report id is required")
	}
	if err := validateTarget(target); err != nil {
		return nil, err
	}
	if strings.TrimSpace(reporterID.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "content report reporter id is required")
	}
	if strings.TrimSpace(reason.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "content report reason is required")
	}
	if _, err := NewReportStatus(status.String()); err != nil {
		return nil, err
	}
	if err := validateReportReviewFields(status, reviewedBy, reviewedAt); err != nil {
		return nil, err
	}
	if createdAt.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "content report created time can't be zero")
	}
	if updatedAt.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "content report updated time can't be zero")
	}
	if updatedAt.Before(createdAt) {
		return nil, apperr.New(apperr.CodeInvalidArgument, "content report updated time can't be before created time")
	}

	return &ContentReport{
		id:         id,
		target:     cloneTarget(target),
		reporterID: reporterID,
		reason:     reason,
		status:     status,
		reviewedBy: cloneOptionalUserID(reviewedBy),
		reviewedAt: cloneOptionalTime(reviewedAt),
		createdAt:  createdAt,
		updatedAt:  updatedAt,
	}, nil
}

func (report *ContentReport) ID() ContentReportID {
	return report.id
}

func (report *ContentReport) Target() Target {
	return cloneTarget(report.target)
}

func (report *ContentReport) ReporterID() userdomain.UserID {
	return report.reporterID
}

func (report *ContentReport) Reason() Reason {
	return report.reason
}

func (report *ContentReport) Status() ReportStatus {
	return report.status
}

func (report *ContentReport) ReviewedBy() (userdomain.UserID, bool) {
	if report.reviewedBy == nil {
		return "", false
	}
	return *report.reviewedBy, true
}

func (report *ContentReport) ReviewedAt() (time.Time, bool) {
	if report.reviewedAt == nil {
		return time.Time{}, false
	}
	return *report.reviewedAt, true
}

func (report *ContentReport) CreatedAt() time.Time {
	return report.createdAt
}

func (report *ContentReport) UpdatedAt() time.Time {
	return report.updatedAt
}

type ModerationAction struct {
	id        ModerationActionID
	target    Target
	actorID   userdomain.UserID
	action    ActionType
	reason    Reason
	createdAt time.Time
}

func NewModerationAction(id ModerationActionID, target Target, actorID userdomain.UserID, action ActionType, reason Reason, now time.Time) (*ModerationAction, error) {
	return RehydrateModerationAction(id, target, actorID, action, reason, now)
}

func RehydrateModerationAction(
	id ModerationActionID,
	target Target,
	actorID userdomain.UserID,
	action ActionType,
	reason Reason,
	createdAt time.Time,
) (*ModerationAction, error) {
	if strings.TrimSpace(id.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "moderation action id is required")
	}
	if err := validateTarget(target); err != nil {
		return nil, err
	}
	if strings.TrimSpace(actorID.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "moderation action actor id is required")
	}
	if _, err := NewActionType(action.String()); err != nil {
		return nil, err
	}
	if strings.TrimSpace(reason.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "moderation action reason is required")
	}
	if createdAt.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "moderation action created time can't be zero")
	}

	return &ModerationAction{
		id:        id,
		target:    cloneTarget(target),
		actorID:   actorID,
		action:    action,
		reason:    reason,
		createdAt: createdAt,
	}, nil
}

func (action *ModerationAction) ID() ModerationActionID {
	return action.id
}

func (action *ModerationAction) Target() Target {
	return cloneTarget(action.target)
}

func (action *ModerationAction) ActorID() userdomain.UserID {
	return action.actorID
}

func (action *ModerationAction) Action() ActionType {
	return action.action
}

func (action *ModerationAction) Reason() Reason {
	return action.reason
}

func (action *ModerationAction) CreatedAt() time.Time {
	return action.createdAt
}

func validateReportReviewFields(status ReportStatus, reviewedBy *userdomain.UserID, reviewedAt *time.Time) error {
	if status == ReportStatusPending {
		if reviewedBy != nil || reviewedAt != nil {
			return apperr.New(apperr.CodeInvalidArgument, "pending report can't have review fields")
		}
		return nil
	}

	if reviewedBy == nil || strings.TrimSpace(reviewedBy.String()) == "" {
		return apperr.New(apperr.CodeInvalidArgument, "reviewed report reviewer is required")
	}
	if reviewedAt == nil || reviewedAt.IsZero() {
		return apperr.New(apperr.CodeInvalidArgument, "reviewed report time is required")
	}
	return nil
}

func clonePostID(id postdomain.PostID) *postdomain.PostID {
	copied := id
	return &copied
}

func cloneCommentID(id commentdomain.CommentID) *commentdomain.CommentID {
	copied := id
	return &copied
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

func cloneTarget(target Target) Target {
	return Target{
		targetType: target.targetType,
		postID:     cloneOptionalPostID(target.postID),
		commentID:  cloneOptionalCommentID(target.commentID),
	}
}

func cloneOptionalPostID(id *postdomain.PostID) *postdomain.PostID {
	if id == nil {
		return nil
	}
	return clonePostID(*id)
}

func cloneOptionalCommentID(id *commentdomain.CommentID) *commentdomain.CommentID {
	if id == nil {
		return nil
	}
	return cloneCommentID(*id)
}

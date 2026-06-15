package notificationusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

const (
	DefaultNotificationListLimit = 20
	MaxNotificationListLimit     = 50
	MaxNotificationSourceIDRunes = 64
)

const (
	NotificationTypePostReply    = "post_reply"
	NotificationTypeCommentReply = "comment_reply"
	NotificationTypePostLike     = "post_like"
	NotificationTypeCommentLike  = "comment_like"
	NotificationTypeMention      = "mention"

	NotificationSourcePost                   = "post"
	NotificationSourceComment                = "comment"
	NotificationSourceCommunityOwnerTransfer = "community_owner_transfer"
)

type StatusFilter string

const (
	StatusFilterUnread StatusFilter = "unread"
	StatusFilterRead   StatusFilter = "read"
	StatusFilterAll    StatusFilter = "all"
)

type CategoryFilter string

const (
	CategoryFilterAll          CategoryFilter = "all"
	CategoryFilterInteractions CategoryFilter = "interactions"
	CategoryFilterReplies      CategoryFilter = "replies"
	CategoryFilterMentions     CategoryFilter = "mentions"
	CategoryFilterLikes        CategoryFilter = "likes"
	CategoryFilterSystem       CategoryFilter = "system"
)

type UseCase struct {
	repository Repository
	now        func() time.Time
}

type Notification struct {
	ID             string
	RecipientID    string
	Type           string
	Title          string
	Body           string
	SourceType     string
	SourceID       string
	AggregateKey   string
	AggregateCount int
	LastActorID    string
	LastActor      *NotificationActor
	Actor          *NotificationActor
	Context        NotificationContext
	ReadAt         *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type NotificationActor struct {
	ID          string
	Username    string
	DisplayName string
	AvatarURL   string
}

type NotificationCommunityContext struct {
	ID   string
	Slug string
	Name string
}

type NotificationContext struct {
	PostID         string
	CommentID      string
	Permalink      string
	PostTitle      string
	CommentExcerpt string
	CommentDepth   int
	Community      *NotificationCommunityContext
}

type ListNotificationsInput struct {
	ActorID  userdomain.UserID
	Category string
	Status   string
	Limit    int
	Offset   int
}

type ListNotificationsResult struct {
	Notifications []Notification
	Category      string
	Status        string
	Limit         int
	Offset        int
	NextOffset    int
	HasMore       bool
}

type UnreadSummaryInput struct {
	ActorID userdomain.UserID
}

type UnreadSummary struct {
	Total    int
	Replies  int
	Mentions int
	Likes    int
	System   int
}

type MarkNotificationReadInput struct {
	ActorID        userdomain.UserID
	NotificationID string
}

type MarkNotificationReadResult struct {
	Notification Notification
}

type MarkAllNotificationsReadInput struct {
	ActorID userdomain.UserID
}

type MarkAllNotificationsReadResult struct {
	UpdatedCount int
	ReadAt       time.Time
}

func NewUseCase(repository Repository, now func() time.Time) *UseCase {
	if now == nil {
		now = time.Now
	}
	return &UseCase{
		repository: repository,
		now:        now,
	}
}

func (uc *UseCase) NotifyPostCommented(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, postID string) error {
	if shouldSkipNotification(recipientID, actorID) {
		return nil
	}
	notification := uc.newNotification(recipientID, actorID, NotificationTypePostReply, "新的评论", "有人评论了你的帖子", NotificationSourcePost, postID, "")
	if err := uc.repository.Create(ctx, notification); err != nil {
		return fmt.Errorf("create post comment notification: %w", err)
	}
	return nil
}

func (uc *UseCase) NotifyCommentReplied(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, commentID string) error {
	if shouldSkipNotification(recipientID, actorID) {
		return nil
	}
	notification := uc.newNotification(recipientID, actorID, NotificationTypeCommentReply, "新的回复", "有人回复了你的评论", NotificationSourceComment, commentID, "")
	if err := uc.repository.Create(ctx, notification); err != nil {
		return fmt.Errorf("create comment reply notification: %w", err)
	}
	return nil
}

func (uc *UseCase) NotifyPostUpvoted(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, postID string) error {
	if shouldSkipNotification(recipientID, actorID) {
		return nil
	}
	notification := uc.newNotification(recipientID, actorID, NotificationTypePostLike, "新的点赞", "有人赞了你的帖子", NotificationSourcePost, postID, uc.aggregateKey(NotificationTypePostLike, postID))
	if err := uc.repository.UpsertAggregated(ctx, notification); err != nil {
		return fmt.Errorf("upsert post like notification: %w", err)
	}
	return nil
}

func (uc *UseCase) NotifyCommentUpvoted(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, commentID string) error {
	if shouldSkipNotification(recipientID, actorID) {
		return nil
	}
	notification := uc.newNotification(recipientID, actorID, NotificationTypeCommentLike, "新的点赞", "有人赞了你的评论", NotificationSourceComment, commentID, uc.aggregateKey(NotificationTypeCommentLike, commentID))
	if err := uc.repository.UpsertAggregated(ctx, notification); err != nil {
		return fmt.Errorf("upsert comment like notification: %w", err)
	}
	return nil
}

func (uc *UseCase) NotifyMentioned(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, sourceType string, sourceID string) error {
	if shouldSkipNotification(recipientID, actorID) {
		return nil
	}
	sourceType = strings.TrimSpace(sourceType)
	switch sourceType {
	case NotificationSourcePost, NotificationSourceComment:
	default:
		return apperr.New(apperr.CodeInvalidArgument, "mention notification source type is invalid")
	}
	if strings.TrimSpace(sourceID) == "" {
		return apperr.New(apperr.CodeInvalidArgument, "mention notification source id is required")
	}
	if len([]rune(strings.TrimSpace(sourceID))) > MaxNotificationSourceIDRunes {
		return apperr.New(apperr.CodeInvalidArgument, "mention notification source id is invalid")
	}
	notification := uc.newNotification(recipientID, actorID, NotificationTypeMention, "新的提及", "有人提到了你", sourceType, sourceID, "")
	if err := uc.repository.Create(ctx, notification); err != nil {
		return fmt.Errorf("create mention notification: %w", err)
	}
	return nil
}

func (uc *UseCase) NotifyCommunityOwnerTransfer(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, communitySlug string, transferID string) error {
	if shouldSkipNotification(recipientID, actorID) {
		return nil
	}
	sourceID := strings.TrimSpace(communitySlug) + ":" + strings.TrimSpace(transferID)
	if strings.TrimSpace(communitySlug) == "" || strings.TrimSpace(transferID) == "" {
		return apperr.New(apperr.CodeInvalidArgument, "community owner transfer notification source is required")
	}
	notification := uc.newNotification(
		recipientID,
		actorID,
		"system",
		"社区负责人交接邀请",
		"你收到了一条社区负责人交接邀请",
		NotificationSourceCommunityOwnerTransfer,
		sourceID,
		"",
	)
	if err := uc.repository.Create(ctx, notification); err != nil {
		return fmt.Errorf("create community owner transfer notification: %w", err)
	}
	return nil
}

func (uc *UseCase) ListNotifications(ctx context.Context, input ListNotificationsInput) (ListNotificationsResult, error) {
	if err := requireActor(input.ActorID); err != nil {
		return ListNotificationsResult{}, err
	}
	category, err := normalizeCategory(input.Category)
	if err != nil {
		return ListNotificationsResult{}, err
	}
	status, err := normalizeStatus(input.Status)
	if err != nil {
		return ListNotificationsResult{}, err
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListNotificationsResult{}, err
	}

	notifications, err := uc.repository.ListByRecipient(ctx, input.ActorID, category, status, limit+1, offset)
	if err != nil {
		return ListNotificationsResult{}, fmt.Errorf("list notifications: %w", err)
	}
	notifications, hasMore := trimNotificationsPage(notifications, limit)
	return ListNotificationsResult{
		Notifications: notifications,
		Category:      category.String(),
		Status:        status.String(),
		Limit:         limit,
		Offset:        offset,
		NextOffset:    offset + len(notifications),
		HasMore:       hasMore,
	}, nil
}

func (uc *UseCase) newNotification(recipientID userdomain.UserID, actorID userdomain.UserID, notificationType string, title string, body string, sourceType string, sourceID string, aggregateKey string) Notification {
	now := uc.now().UTC()
	sourceID = strings.TrimSpace(sourceID)
	return Notification{
		ID:             uuid.NewString(),
		RecipientID:    recipientID.String(),
		Type:           notificationType,
		Title:          title,
		Body:           body,
		SourceType:     sourceType,
		SourceID:       sourceID,
		AggregateKey:   aggregateKey,
		AggregateCount: 1,
		LastActorID:    actorID.String(),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (uc *UseCase) aggregateKey(notificationType string, sourceID string) string {
	window := uc.now().UTC().Truncate(time.Hour).Format("2006010215")
	return notificationType + ":" + strings.TrimSpace(sourceID) + ":" + window
}

func shouldSkipNotification(recipientID userdomain.UserID, actorID userdomain.UserID) bool {
	recipient := strings.TrimSpace(recipientID.String())
	actor := strings.TrimSpace(actorID.String())
	return recipient == "" || actor == "" || recipient == actor
}

func (uc *UseCase) GetUnreadSummary(ctx context.Context, input UnreadSummaryInput) (UnreadSummary, error) {
	if err := requireActor(input.ActorID); err != nil {
		return UnreadSummary{}, err
	}
	summary, err := uc.repository.CountUnreadByCategory(ctx, input.ActorID)
	if err != nil {
		return UnreadSummary{}, fmt.Errorf("count unread notifications: %w", err)
	}
	return summary, nil
}

func (uc *UseCase) MarkNotificationRead(ctx context.Context, input MarkNotificationReadInput) (MarkNotificationReadResult, error) {
	if err := requireActor(input.ActorID); err != nil {
		return MarkNotificationReadResult{}, err
	}
	if _, err := uuid.Parse(strings.TrimSpace(input.NotificationID)); err != nil {
		return MarkNotificationReadResult{}, apperr.New(apperr.CodeInvalidArgument, "notification id is invalid")
	}

	notification, err := uc.repository.MarkRead(ctx, strings.TrimSpace(input.NotificationID), input.ActorID, uc.now().UTC())
	if err != nil {
		return MarkNotificationReadResult{}, fmt.Errorf("mark notification read: %w", err)
	}
	return MarkNotificationReadResult{
		Notification: notification,
	}, nil
}

func (uc *UseCase) MarkAllNotificationsRead(ctx context.Context, input MarkAllNotificationsReadInput) (MarkAllNotificationsReadResult, error) {
	if err := requireActor(input.ActorID); err != nil {
		return MarkAllNotificationsReadResult{}, err
	}
	readAt := uc.now().UTC()
	updatedCount, err := uc.repository.MarkAllRead(ctx, input.ActorID, readAt)
	if err != nil {
		return MarkAllNotificationsReadResult{}, fmt.Errorf("mark all notifications read: %w", err)
	}
	return MarkAllNotificationsReadResult{
		UpdatedCount: updatedCount,
		ReadAt:       readAt,
	}, nil
}

func normalizeCategory(raw string) (CategoryFilter, error) {
	if strings.TrimSpace(raw) == "" {
		return CategoryFilterAll, nil
	}
	switch CategoryFilter(strings.TrimSpace(strings.ToLower(raw))) {
	case CategoryFilterAll:
		return CategoryFilterAll, nil
	case CategoryFilterInteractions:
		return CategoryFilterInteractions, nil
	case CategoryFilterReplies:
		return CategoryFilterReplies, nil
	case CategoryFilterMentions:
		return CategoryFilterMentions, nil
	case CategoryFilterLikes:
		return CategoryFilterLikes, nil
	case CategoryFilterSystem:
		return CategoryFilterSystem, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "notification category is invalid")
	}
}

func (category CategoryFilter) String() string {
	return string(category)
}

func normalizeStatus(raw string) (StatusFilter, error) {
	if strings.TrimSpace(raw) == "" {
		return StatusFilterAll, nil
	}
	switch StatusFilter(strings.TrimSpace(strings.ToLower(raw))) {
	case StatusFilterUnread:
		return StatusFilterUnread, nil
	case StatusFilterRead:
		return StatusFilterRead, nil
	case StatusFilterAll:
		return StatusFilterAll, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "notification status is invalid")
	}
}

func (status StatusFilter) String() string {
	return string(status)
}

func normalizePagination(limit int, offset int) (int, int, error) {
	if limit < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "limit must be non-negative")
	}
	if offset < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "offset must be non-negative")
	}
	if limit == 0 {
		limit = DefaultNotificationListLimit
	}
	if limit > MaxNotificationListLimit {
		limit = MaxNotificationListLimit
	}
	return limit, offset, nil
}

func trimNotificationsPage(notifications []Notification, limit int) ([]Notification, bool) {
	if len(notifications) <= limit {
		return notifications, false
	}
	return notifications[:limit], true
}

func requireActor(actorID userdomain.UserID) error {
	if strings.TrimSpace(actorID.String()) == "" {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	return nil
}

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
)

type StatusFilter string

const (
	StatusFilterUnread StatusFilter = "unread"
	StatusFilterRead   StatusFilter = "read"
	StatusFilterAll    StatusFilter = "all"
)

type CategoryFilter string

const (
	CategoryFilterAll      CategoryFilter = "all"
	CategoryFilterReplies  CategoryFilter = "replies"
	CategoryFilterMentions CategoryFilter = "mentions"
	CategoryFilterLikes    CategoryFilter = "likes"
	CategoryFilterSystem   CategoryFilter = "system"
)

type UseCase struct {
	repository Repository
	now        func() time.Time
}

type Notification struct {
	ID          string
	RecipientID string
	Type        string
	Title       string
	Body        string
	SourceType  string
	SourceID    string
	ReadAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
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

	notifications, err := uc.repository.ListByRecipient(ctx, input.ActorID, category, status, limit, offset)
	if err != nil {
		return ListNotificationsResult{}, fmt.Errorf("list notifications: %w", err)
	}
	return ListNotificationsResult{
		Notifications: notifications,
		Category:      category.String(),
		Status:        status.String(),
		Limit:         limit,
		Offset:        offset,
	}, nil
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
		return StatusFilterUnread, nil
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

func requireActor(actorID userdomain.UserID) error {
	if strings.TrimSpace(actorID.String()) == "" {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	return nil
}

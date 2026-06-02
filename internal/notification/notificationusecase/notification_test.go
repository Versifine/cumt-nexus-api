package notificationusecase

import (
	"context"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

func TestListNotificationsDefaultsUnreadAndPagination(t *testing.T) {
	now := testNow()
	actorID := userdomain.NewGeneratedUserID()
	repository := &fakeRepository{
		listFunc: func(ctx context.Context, recipientID userdomain.UserID, status StatusFilter, limit int, offset int) ([]Notification, error) {
			if recipientID != actorID {
				t.Fatalf("expected recipient %q, got %q", actorID.String(), recipientID.String())
			}
			if status != StatusFilterUnread {
				t.Fatalf("expected unread status, got %q", status)
			}
			if limit != DefaultNotificationListLimit || offset != 0 {
				t.Fatalf("expected default pagination, got %d/%d", limit, offset)
			}
			return []Notification{newNotification(actorID, now)}, nil
		},
	}
	uc := NewUseCase(repository, func() time.Time { return now })

	result, err := uc.ListNotifications(context.Background(), ListNotificationsInput{
		ActorID: actorID,
	})
	if err != nil {
		t.Fatalf("ListNotifications returned error: %v", err)
	}
	if !repository.listCalled {
		t.Fatal("expected repository list call")
	}
	if result.Status != StatusFilterUnread.String() || result.Limit != DefaultNotificationListLimit || result.Offset != 0 {
		t.Fatalf("unexpected list result: %#v", result)
	}
	if len(result.Notifications) != 1 {
		t.Fatalf("expected one notification, got %#v", result.Notifications)
	}
}

func TestListNotificationsSupportsStatusAndClampsLimit(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	repository := &fakeRepository{
		listFunc: func(ctx context.Context, recipientID userdomain.UserID, status StatusFilter, limit int, offset int) ([]Notification, error) {
			if status != StatusFilterAll {
				t.Fatalf("expected all status, got %q", status)
			}
			if limit != MaxNotificationListLimit || offset != 3 {
				t.Fatalf("expected clamped pagination, got %d/%d", limit, offset)
			}
			return nil, nil
		},
	}
	uc := NewUseCase(repository, time.Now)

	result, err := uc.ListNotifications(context.Background(), ListNotificationsInput{
		ActorID: actorID,
		Status:  "ALL",
		Limit:   99,
		Offset:  3,
	})
	if err != nil {
		t.Fatalf("ListNotifications returned error: %v", err)
	}
	if result.Status != StatusFilterAll.String() || result.Limit != MaxNotificationListLimit || result.Offset != 3 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestListNotificationsRejectsInvalidInput(t *testing.T) {
	uc := NewUseCase(&fakeRepository{}, time.Now)

	tests := []struct {
		name  string
		input ListNotificationsInput
		code  apperr.Code
	}{
		{name: "missing actor", input: ListNotificationsInput{}, code: apperr.CodeUnauthenticated},
		{name: "invalid status", input: ListNotificationsInput{ActorID: userdomain.NewGeneratedUserID(), Status: "deleted"}, code: apperr.CodeInvalidArgument},
		{name: "negative limit", input: ListNotificationsInput{ActorID: userdomain.NewGeneratedUserID(), Limit: -1}, code: apperr.CodeInvalidArgument},
		{name: "negative offset", input: ListNotificationsInput{ActorID: userdomain.NewGeneratedUserID(), Offset: -1}, code: apperr.CodeInvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.ListNotifications(context.Background(), tt.input)
			if !apperr.IsCode(err, tt.code) {
				t.Fatalf("expected %s, got %v", tt.code, err)
			}
		})
	}
}

func TestMarkNotificationRead(t *testing.T) {
	now := testNow()
	actorID := userdomain.NewGeneratedUserID()
	notificationID := uuid.NewString()
	repository := &fakeRepository{
		markReadFunc: func(ctx context.Context, id string, recipientID userdomain.UserID, readAt time.Time) (Notification, error) {
			if id != notificationID || recipientID != actorID || !readAt.Equal(now) {
				t.Fatalf("unexpected mark read args: %q %q %s", id, recipientID.String(), readAt)
			}
			notification := newNotification(actorID, now)
			notification.ID = id
			notification.ReadAt = &readAt
			return notification, nil
		},
	}
	uc := NewUseCase(repository, func() time.Time { return now })

	result, err := uc.MarkNotificationRead(context.Background(), MarkNotificationReadInput{
		ActorID:        actorID,
		NotificationID: notificationID,
	})
	if err != nil {
		t.Fatalf("MarkNotificationRead returned error: %v", err)
	}
	if !repository.markReadCalled {
		t.Fatal("expected mark read repository call")
	}
	if result.Notification.ReadAt == nil || !result.Notification.ReadAt.Equal(now) {
		t.Fatalf("expected read_at %s, got %#v", now, result.Notification.ReadAt)
	}
}

func TestMarkNotificationReadRejectsInvalidInput(t *testing.T) {
	uc := NewUseCase(&fakeRepository{}, time.Now)

	_, err := uc.MarkNotificationRead(context.Background(), MarkNotificationReadInput{
		NotificationID: uuid.NewString(),
	})
	if !apperr.IsCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated, got %v", err)
	}

	_, err = uc.MarkNotificationRead(context.Background(), MarkNotificationReadInput{
		ActorID:        userdomain.NewGeneratedUserID(),
		NotificationID: "not-a-uuid",
	})
	if !apperr.IsCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument, got %v", err)
	}
}

func TestNotificationUseCasePropagatesRepositoryError(t *testing.T) {
	repository := &fakeRepository{
		listErr: apperr.New(apperr.CodeNotFound, "notification not found"),
	}
	uc := NewUseCase(repository, time.Now)

	_, err := uc.ListNotifications(context.Background(), ListNotificationsInput{
		ActorID: userdomain.NewGeneratedUserID(),
	})
	if !apperr.IsCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found from repository, got %v", err)
	}
}

type fakeRepository struct {
	listCalled     bool
	markReadCalled bool
	listFunc       func(ctx context.Context, recipientID userdomain.UserID, status StatusFilter, limit int, offset int) ([]Notification, error)
	markReadFunc   func(ctx context.Context, id string, recipientID userdomain.UserID, readAt time.Time) (Notification, error)
	listErr        error
	markReadErr    error
}

func (f *fakeRepository) ListByRecipient(ctx context.Context, recipientID userdomain.UserID, status StatusFilter, limit int, offset int) ([]Notification, error) {
	f.listCalled = true
	if f.listFunc != nil {
		return f.listFunc(ctx, recipientID, status, limit, offset)
	}
	return nil, f.listErr
}

func (f *fakeRepository) MarkRead(ctx context.Context, id string, recipientID userdomain.UserID, readAt time.Time) (Notification, error) {
	f.markReadCalled = true
	if f.markReadFunc != nil {
		return f.markReadFunc(ctx, id, recipientID, readAt)
	}
	return Notification{}, f.markReadErr
}

func newNotification(recipientID userdomain.UserID, now time.Time) Notification {
	return Notification{
		ID:          uuid.NewString(),
		RecipientID: recipientID.String(),
		Type:        "system",
		Title:       "Title",
		Body:        "Body",
		SourceType:  "system",
		SourceID:    "source-1",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func testNow() time.Time {
	return time.Date(2026, 6, 2, 7, 0, 0, 0, time.UTC)
}

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
		listFunc: func(ctx context.Context, recipientID userdomain.UserID, category CategoryFilter, status StatusFilter, limit int, offset int) ([]Notification, error) {
			if recipientID != actorID {
				t.Fatalf("expected recipient %q, got %q", actorID.String(), recipientID.String())
			}
			if category != CategoryFilterAll {
				t.Fatalf("expected all category, got %q", category)
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
		listFunc: func(ctx context.Context, recipientID userdomain.UserID, category CategoryFilter, status StatusFilter, limit int, offset int) ([]Notification, error) {
			if category != CategoryFilterLikes {
				t.Fatalf("expected likes category, got %q", category)
			}
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
		ActorID:  actorID,
		Category: "LIKES",
		Status:   "ALL",
		Limit:    99,
		Offset:   3,
	})
	if err != nil {
		t.Fatalf("ListNotifications returned error: %v", err)
	}
	if result.Category != CategoryFilterLikes.String() || result.Status != StatusFilterAll.String() || result.Limit != MaxNotificationListLimit || result.Offset != 3 {
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
		{name: "invalid category", input: ListNotificationsInput{ActorID: userdomain.NewGeneratedUserID(), Category: "direct"}, code: apperr.CodeInvalidArgument},
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

func TestGetUnreadSummary(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	repository := &fakeRepository{
		unreadSummary: UnreadSummary{
			Total:    5,
			Replies:  1,
			Mentions: 1,
			Likes:    2,
			System:   1,
		},
	}
	uc := NewUseCase(repository, time.Now)

	result, err := uc.GetUnreadSummary(context.Background(), UnreadSummaryInput{
		ActorID: actorID,
	})
	if err != nil {
		t.Fatalf("GetUnreadSummary returned error: %v", err)
	}
	if !repository.countUnreadCalled {
		t.Fatal("expected count unread repository call")
	}
	if repository.countUnreadRecipientID != actorID {
		t.Fatalf("expected recipient %q, got %q", actorID.String(), repository.countUnreadRecipientID.String())
	}
	if result.Total != 5 || result.Likes != 2 {
		t.Fatalf("unexpected unread summary: %#v", result)
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

func TestMarkAllNotificationsRead(t *testing.T) {
	now := testNow()
	actorID := userdomain.NewGeneratedUserID()
	repository := &fakeRepository{
		markAllReadFunc: func(ctx context.Context, recipientID userdomain.UserID, readAt time.Time) (int, error) {
			if recipientID != actorID || !readAt.Equal(now) {
				t.Fatalf("unexpected mark all args: %q %s", recipientID.String(), readAt)
			}
			return 3, nil
		},
	}
	uc := NewUseCase(repository, func() time.Time { return now })

	result, err := uc.MarkAllNotificationsRead(context.Background(), MarkAllNotificationsReadInput{
		ActorID: actorID,
	})
	if err != nil {
		t.Fatalf("MarkAllNotificationsRead returned error: %v", err)
	}
	if !repository.markAllReadCalled {
		t.Fatal("expected mark all read repository call")
	}
	if result.UpdatedCount != 3 || !result.ReadAt.Equal(now) {
		t.Fatalf("unexpected mark all result: %#v", result)
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

func TestUnreadSummaryAndMarkAllRejectMissingActor(t *testing.T) {
	uc := NewUseCase(&fakeRepository{}, time.Now)

	if _, err := uc.GetUnreadSummary(context.Background(), UnreadSummaryInput{}); !apperr.IsCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated from unread summary, got %v", err)
	}
	if _, err := uc.MarkAllNotificationsRead(context.Background(), MarkAllNotificationsReadInput{}); !apperr.IsCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated from mark all, got %v", err)
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

func TestNotifyPostUpvotedUpsertsAggregate(t *testing.T) {
	now := testNow()
	recipientID := userdomain.NewGeneratedUserID()
	actorID := userdomain.NewGeneratedUserID()
	repository := &fakeRepository{}
	uc := NewUseCase(repository, func() time.Time { return now })

	err := uc.NotifyPostUpvoted(context.Background(), recipientID, actorID, "post-1")
	if err != nil {
		t.Fatalf("NotifyPostUpvoted returned error: %v", err)
	}
	if !repository.upsertAggregatedCalled {
		t.Fatal("expected aggregated notification upsert")
	}
	notification := repository.upsertAggregatedNotification
	if notification.RecipientID != recipientID.String() || notification.LastActorID != actorID.String() {
		t.Fatalf("unexpected notification actors: %#v", notification)
	}
	if notification.Type != NotificationTypePostLike || notification.SourceType != NotificationSourcePost || notification.SourceID != "post-1" {
		t.Fatalf("unexpected post like notification: %#v", notification)
	}
	if notification.AggregateKey == "" || notification.AggregateCount != 1 {
		t.Fatalf("unexpected aggregate fields: %#v", notification)
	}
}

func TestNotifyPostUpvotedSkipsSelf(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	repository := &fakeRepository{}
	uc := NewUseCase(repository, time.Now)

	if err := uc.NotifyPostUpvoted(context.Background(), actorID, actorID, "post-1"); err != nil {
		t.Fatalf("NotifyPostUpvoted returned error: %v", err)
	}
	if repository.createCalled || repository.upsertAggregatedCalled {
		t.Fatal("expected self notification to be skipped")
	}
}

func TestNotifyCommentRepliedCreatesNotification(t *testing.T) {
	recipientID := userdomain.NewGeneratedUserID()
	actorID := userdomain.NewGeneratedUserID()
	repository := &fakeRepository{}
	uc := NewUseCase(repository, time.Now)

	if err := uc.NotifyCommentReplied(context.Background(), recipientID, actorID, "comment-1"); err != nil {
		t.Fatalf("NotifyCommentReplied returned error: %v", err)
	}
	if !repository.createCalled {
		t.Fatal("expected notification create")
	}
	notification := repository.createdNotification
	if notification.Type != NotificationTypeCommentReply || notification.SourceType != NotificationSourceComment || notification.SourceID != "comment-1" {
		t.Fatalf("unexpected comment reply notification: %#v", notification)
	}
	if notification.LastActorID != actorID.String() || notification.AggregateCount != 1 {
		t.Fatalf("unexpected actor or aggregate count: %#v", notification)
	}
}

func TestNotifyMentionedCreatesNotification(t *testing.T) {
	recipientID := userdomain.NewGeneratedUserID()
	actorID := userdomain.NewGeneratedUserID()
	repository := &fakeRepository{}
	uc := NewUseCase(repository, time.Now)

	if err := uc.NotifyMentioned(context.Background(), recipientID, actorID, NotificationSourcePost, "post-1"); err != nil {
		t.Fatalf("NotifyMentioned returned error: %v", err)
	}
	if !repository.createCalled {
		t.Fatal("expected notification create")
	}
	notification := repository.createdNotification
	if notification.Type != NotificationTypeMention || notification.SourceType != NotificationSourcePost || notification.SourceID != "post-1" {
		t.Fatalf("unexpected mention notification: %#v", notification)
	}
	if notification.LastActorID != actorID.String() || notification.AggregateKey != "" {
		t.Fatalf("unexpected mention metadata: %#v", notification)
	}
}

func TestNotifyMentionedSkipsSelf(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	repository := &fakeRepository{}
	uc := NewUseCase(repository, time.Now)

	if err := uc.NotifyMentioned(context.Background(), actorID, actorID, NotificationSourcePost, "post-1"); err != nil {
		t.Fatalf("NotifyMentioned returned error: %v", err)
	}
	if repository.createCalled || repository.upsertAggregatedCalled {
		t.Fatal("expected self mention notification to be skipped")
	}
}

func TestNotifyMentionedRejectsInvalidSource(t *testing.T) {
	uc := NewUseCase(&fakeRepository{}, time.Now)

	err := uc.NotifyMentioned(context.Background(), userdomain.NewGeneratedUserID(), userdomain.NewGeneratedUserID(), "profile", "target-1")
	if !apperr.IsCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument, got %v", err)
	}
}

type fakeRepository struct {
	listCalled                   bool
	createCalled                 bool
	upsertAggregatedCalled       bool
	countUnreadCalled            bool
	markReadCalled               bool
	markAllReadCalled            bool
	countUnreadRecipientID       userdomain.UserID
	createdNotification          Notification
	upsertAggregatedNotification Notification
	listFunc                     func(ctx context.Context, recipientID userdomain.UserID, category CategoryFilter, status StatusFilter, limit int, offset int) ([]Notification, error)
	createErr                    error
	upsertAggregatedErr          error
	markReadFunc                 func(ctx context.Context, id string, recipientID userdomain.UserID, readAt time.Time) (Notification, error)
	markAllReadFunc              func(ctx context.Context, recipientID userdomain.UserID, readAt time.Time) (int, error)
	unreadSummary                UnreadSummary
	listErr                      error
	countUnreadErr               error
	markReadErr                  error
	markAllReadErr               error
}

func (f *fakeRepository) Create(ctx context.Context, notification Notification) error {
	f.createCalled = true
	f.createdNotification = notification
	return f.createErr
}

func (f *fakeRepository) UpsertAggregated(ctx context.Context, notification Notification) error {
	f.upsertAggregatedCalled = true
	f.upsertAggregatedNotification = notification
	return f.upsertAggregatedErr
}

func (f *fakeRepository) ListByRecipient(ctx context.Context, recipientID userdomain.UserID, category CategoryFilter, status StatusFilter, limit int, offset int) ([]Notification, error) {
	f.listCalled = true
	if f.listFunc != nil {
		return f.listFunc(ctx, recipientID, category, status, limit, offset)
	}
	return nil, f.listErr
}

func (f *fakeRepository) CountUnreadByCategory(ctx context.Context, recipientID userdomain.UserID) (UnreadSummary, error) {
	f.countUnreadCalled = true
	f.countUnreadRecipientID = recipientID
	return f.unreadSummary, f.countUnreadErr
}

func (f *fakeRepository) MarkRead(ctx context.Context, id string, recipientID userdomain.UserID, readAt time.Time) (Notification, error) {
	f.markReadCalled = true
	if f.markReadFunc != nil {
		return f.markReadFunc(ctx, id, recipientID, readAt)
	}
	return Notification{}, f.markReadErr
}

func (f *fakeRepository) MarkAllRead(ctx context.Context, recipientID userdomain.UserID, readAt time.Time) (int, error) {
	f.markAllReadCalled = true
	if f.markAllReadFunc != nil {
		return f.markAllReadFunc(ctx, recipientID, readAt)
	}
	return 0, f.markAllReadErr
}

func newNotification(recipientID userdomain.UserID, now time.Time) Notification {
	return Notification{
		ID:             uuid.NewString(),
		RecipientID:    recipientID.String(),
		Type:           "system",
		Title:          "Title",
		Body:           "Body",
		SourceType:     "system",
		SourceID:       "source-1",
		AggregateCount: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func testNow() time.Time {
	return time.Date(2026, 6, 2, 7, 0, 0, 0, time.UTC)
}

package notificationusecase

import (
	"context"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type Repository interface {
	ListByRecipient(ctx context.Context, recipientID userdomain.UserID, category CategoryFilter, status StatusFilter, limit int, offset int) ([]Notification, error)
	CountUnreadByCategory(ctx context.Context, recipientID userdomain.UserID) (UnreadSummary, error)
	MarkRead(ctx context.Context, id string, recipientID userdomain.UserID, readAt time.Time) (Notification, error)
	MarkAllRead(ctx context.Context, recipientID userdomain.UserID, readAt time.Time) (int, error)
}

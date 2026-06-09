package mediausecase

import (
	"context"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
)

type AttachmentRepository interface {
	Create(ctx context.Context, attachment mediadomain.Attachment) error
	FindByID(ctx context.Context, id mediadomain.AttachmentID) (*mediadomain.Attachment, error)
	ListCleanupCandidates(ctx context.Context, unboundReadyBefore time.Time, failedOrBlockedBefore time.Time, limit int) ([]mediadomain.Attachment, error)
	TakeCleanupCandidates(ctx context.Context, unboundReadyBefore time.Time, failedOrBlockedBefore time.Time, limit int) ([]mediadomain.Attachment, error)
}

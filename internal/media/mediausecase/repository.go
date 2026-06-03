package mediausecase

import (
	"context"

	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
)

type AttachmentRepository interface {
	Create(ctx context.Context, attachment mediadomain.Attachment) error
	FindByID(ctx context.Context, id mediadomain.AttachmentID) (*mediadomain.Attachment, error)
}

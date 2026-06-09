package mediausecase

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type UseCase struct {
	attachments AttachmentRepository
	storage     ObjectStorage
	limits      UploadLimits
	now         func() time.Time
}

type ObjectStorage interface {
	PutObject(ctx context.Context, input PutObjectInput) (PutObjectResult, error)
}

type PutObjectInput struct {
	ObjectKey   string
	ContentType string
	SizeBytes   int64
	Body        io.Reader
}

type PutObjectResult struct {
	StorageProvider string
	Bucket          string
	ObjectKey       string
	PublicURL       string
}

type UploadLimits struct {
	ImageMaxBytes int
}

type Attachment struct {
	ID                 string
	OwnerType          string
	OwnerID            string
	UploaderID         string
	Kind               string
	StorageProvider    string
	Bucket             string
	ObjectKey          string
	PublicURL          string
	ThumbnailObjectKey string
	Width              *int
	Height             *int
	SizeBytes          int64
	MimeType           string
	AltText            string
	Status             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type SaveImageAttachmentInput struct {
	UploaderID         userdomain.UserID
	StorageProvider    string
	Bucket             string
	ObjectKey          string
	PublicURL          string
	ThumbnailObjectKey string
	Width              *int
	Height             *int
	SizeBytes          int64
	MimeType           string
	AltText            string
}

type SaveImageAttachmentResult struct {
	Attachment Attachment
}

type UploadImageInput struct {
	UploaderID userdomain.UserID
	FileBytes  []byte
	AltText    string
}

type UploadImageResult struct {
	Attachment Attachment
}

func NewUseCase(attachments AttachmentRepository, storage ObjectStorage, limits UploadLimits, now func() time.Time) *UseCase {
	if now == nil {
		now = time.Now
	}
	if limits.ImageMaxBytes <= 0 {
		limits.ImageMaxBytes = 5 * 1024 * 1024
	}
	return &UseCase{
		attachments: attachments,
		storage:     storage,
		limits:      limits,
		now:         now,
	}
}

func (uc *UseCase) UploadImage(ctx context.Context, input UploadImageInput) (UploadImageResult, error) {
	if strings.TrimSpace(input.UploaderID.String()) == "" {
		return UploadImageResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if len(input.FileBytes) == 0 {
		return UploadImageResult{}, apperr.New(apperr.CodeInvalidArgument, "image file is required")
	}
	if len(input.FileBytes) > uc.limits.ImageMaxBytes {
		return UploadImageResult{}, apperr.New(apperr.CodeInvalidArgument, "image file is too large")
	}
	mimeType, extension, err := detectImageMimeType(input.FileBytes)
	if err != nil {
		return UploadImageResult{}, err
	}
	width, height, err := detectImageDimensions(mimeType, input.FileBytes)
	if err != nil {
		return UploadImageResult{}, err
	}
	if uc.storage == nil {
		return UploadImageResult{}, fmt.Errorf("object storage is not configured")
	}

	attachmentID := mediadomain.NewGeneratedAttachmentID()
	now := uc.now().UTC()
	objectKey := imageObjectKey(now, attachmentID, extension)
	stored, err := uc.storage.PutObject(ctx, PutObjectInput{
		ObjectKey:   objectKey,
		ContentType: mimeType,
		SizeBytes:   int64(len(input.FileBytes)),
		Body:        bytes.NewReader(input.FileBytes),
	})
	if err != nil {
		return UploadImageResult{}, fmt.Errorf("put image object: %w", err)
	}
	storageProvider, err := mediadomain.NewStorageProvider(stored.StorageProvider)
	if err != nil {
		return UploadImageResult{}, err
	}

	attachment, err := mediadomain.NewReadyImageAttachment(mediadomain.NewAttachmentParams{
		ID:              attachmentID,
		UploaderID:      input.UploaderID,
		StorageProvider: storageProvider,
		Bucket:          stored.Bucket,
		ObjectKey:       stored.ObjectKey,
		PublicURL:       stored.PublicURL,
		Width:           width,
		Height:          height,
		SizeBytes:       int64(len(input.FileBytes)),
		MimeType:        mimeType,
		AltText:         input.AltText,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		return UploadImageResult{}, err
	}
	if err := uc.attachments.Create(ctx, *attachment); err != nil {
		return UploadImageResult{}, fmt.Errorf("create media attachment: %w", err)
	}

	return UploadImageResult{
		Attachment: toAttachmentDTO(*attachment),
	}, nil
}

func (uc *UseCase) SaveImageAttachment(ctx context.Context, input SaveImageAttachmentInput) (SaveImageAttachmentResult, error) {
	if strings.TrimSpace(input.UploaderID.String()) == "" {
		return SaveImageAttachmentResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	provider, err := mediadomain.NewStorageProvider(input.StorageProvider)
	if err != nil {
		return SaveImageAttachmentResult{}, err
	}
	now := uc.now().UTC()
	attachment, err := mediadomain.NewReadyImageAttachment(mediadomain.NewAttachmentParams{
		ID:                 mediadomain.NewGeneratedAttachmentID(),
		UploaderID:         input.UploaderID,
		StorageProvider:    provider,
		Bucket:             input.Bucket,
		ObjectKey:          input.ObjectKey,
		PublicURL:          input.PublicURL,
		ThumbnailObjectKey: input.ThumbnailObjectKey,
		Width:              input.Width,
		Height:             input.Height,
		SizeBytes:          input.SizeBytes,
		MimeType:           input.MimeType,
		AltText:            input.AltText,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		return SaveImageAttachmentResult{}, err
	}

	if err := uc.attachments.Create(ctx, *attachment); err != nil {
		return SaveImageAttachmentResult{}, fmt.Errorf("create media attachment: %w", err)
	}

	return SaveImageAttachmentResult{
		Attachment: toAttachmentDTO(*attachment),
	}, nil
}

func detectImageMimeType(fileBytes []byte) (mimeType string, extension string, err error) {
	if len(fileBytes) >= 3 && fileBytes[0] == 0xff && fileBytes[1] == 0xd8 && fileBytes[2] == 0xff {
		return "image/jpeg", "jpg", nil
	}
	if len(fileBytes) >= 8 &&
		fileBytes[0] == 0x89 &&
		fileBytes[1] == 0x50 &&
		fileBytes[2] == 0x4e &&
		fileBytes[3] == 0x47 &&
		fileBytes[4] == 0x0d &&
		fileBytes[5] == 0x0a &&
		fileBytes[6] == 0x1a &&
		fileBytes[7] == 0x0a {
		return "image/png", "png", nil
	}
	if len(fileBytes) >= 12 &&
		string(fileBytes[0:4]) == "RIFF" &&
		string(fileBytes[8:12]) == "WEBP" {
		return "image/webp", "webp", nil
	}
	return "", "", apperr.New(apperr.CodeInvalidArgument, "image mime type is invalid")
}

func detectImageDimensions(mimeType string, fileBytes []byte) (*int, *int, error) {
	switch mimeType {
	case "image/jpeg", "image/png":
		config, _, err := image.DecodeConfig(bytes.NewReader(fileBytes))
		if err != nil {
			return nil, nil, apperr.New(apperr.CodeInvalidArgument, "image dimensions are invalid")
		}
		if config.Width <= 0 || config.Height <= 0 {
			return nil, nil, apperr.New(apperr.CodeInvalidArgument, "image dimensions are invalid")
		}
		width := config.Width
		height := config.Height
		return &width, &height, nil
	default:
		return nil, nil, nil
	}
}

func imageObjectKey(now time.Time, id mediadomain.AttachmentID, extension string) string {
	return fmt.Sprintf("images/%04d/%02d/%s.%s", now.Year(), int(now.Month()), id.String(), extension)
}

func toAttachmentDTO(attachment mediadomain.Attachment) Attachment {
	return Attachment{
		ID:                 attachment.ID().String(),
		OwnerType:          attachment.OwnerType().String(),
		OwnerID:            attachment.OwnerID(),
		UploaderID:         attachment.UploaderID().String(),
		Kind:               attachment.Kind().String(),
		StorageProvider:    attachment.StorageProvider().String(),
		Bucket:             attachment.Bucket(),
		ObjectKey:          attachment.ObjectKey(),
		PublicURL:          attachment.PublicURL(),
		ThumbnailObjectKey: attachment.ThumbnailObjectKey(),
		Width:              attachment.Width(),
		Height:             attachment.Height(),
		SizeBytes:          attachment.SizeBytes(),
		MimeType:           attachment.MimeType(),
		AltText:            attachment.AltText(),
		Status:             attachment.Status().String(),
		CreatedAt:          attachment.CreatedAt(),
		UpdatedAt:          attachment.UpdatedAt(),
	}
}

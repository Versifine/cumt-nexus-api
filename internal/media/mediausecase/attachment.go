package mediausecase

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"io"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
	platformsettings "github.com/Versifine/cumt-nexus-api/internal/platform/settings"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type UseCase struct {
	attachments AttachmentRepository
	storage     ObjectStorage
	limits      UploadLimits
	settings    platformsettings.Reader
	now         func() time.Time
}

type ObjectStorage interface {
	PutObject(ctx context.Context, input PutObjectInput) (PutObjectResult, error)
	DeleteObject(ctx context.Context, objectKey string) error
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

const (
	thumbnailMaxDimension = 512
	thumbnailJPEGQuality  = 82
)

type CleanupExpiredAttachmentsInput struct {
	UnboundTTL time.Duration
	FailedTTL  time.Duration
	Limit      int
	DryRun     bool
}

type CleanupExpiredAttachmentsResult struct {
	DryRun                bool      `json:"dry_run"`
	Candidates            int       `json:"candidates"`
	AttachmentsDeleted    int       `json:"attachments_deleted"`
	ObjectsDeleted        int       `json:"objects_deleted"`
	Failures              int       `json:"failures"`
	UnboundReadyBefore    time.Time `json:"unbound_ready_before"`
	FailedOrBlockedBefore time.Time `json:"failed_or_blocked_before"`
	UnboundTTL            string    `json:"unbound_ttl"`
	FailedTTL             string    `json:"failed_ttl"`
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
	ThumbnailURL       string
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

func (uc *UseCase) SetSettingsReader(settingsReader platformsettings.Reader) {
	uc.settings = settingsReader
}

func (uc *UseCase) UploadImage(ctx context.Context, input UploadImageInput) (UploadImageResult, error) {
	if strings.TrimSpace(input.UploaderID.String()) == "" {
		return UploadImageResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if err := uc.ensureUploadEnabled(ctx); err != nil {
		return UploadImageResult{}, err
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

	thumbnailObjectKey := ""
	if thumbnailBytes, ok := createJPEGThumbnail(mimeType, input.FileBytes); ok {
		thumbnailStored, err := uc.storage.PutObject(ctx, PutObjectInput{
			ObjectKey:   thumbnailImageObjectKey(now, attachmentID),
			ContentType: "image/jpeg",
			SizeBytes:   int64(len(thumbnailBytes)),
			Body:        bytes.NewReader(thumbnailBytes),
		})
		if err == nil {
			thumbnailObjectKey = thumbnailStored.ObjectKey
		}
	}

	attachment, err := mediadomain.NewReadyImageAttachment(mediadomain.NewAttachmentParams{
		ID:                 attachmentID,
		UploaderID:         input.UploaderID,
		StorageProvider:    storageProvider,
		Bucket:             stored.Bucket,
		ObjectKey:          stored.ObjectKey,
		PublicURL:          stored.PublicURL,
		ThumbnailObjectKey: thumbnailObjectKey,
		Width:              width,
		Height:             height,
		SizeBytes:          int64(len(input.FileBytes)),
		MimeType:           mimeType,
		AltText:            input.AltText,
		CreatedAt:          now,
		UpdatedAt:          now,
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

func (uc *UseCase) ensureUploadEnabled(ctx context.Context) error {
	if uc.settings == nil {
		return nil
	}
	enabled, err := uc.settings.IsEnabled(ctx, platformsettings.UploadEnabled)
	if err != nil {
		return fmt.Errorf("read upload setting: %w", err)
	}
	if !enabled {
		return apperr.New(apperr.CodeForbidden, "upload is disabled")
	}
	return nil
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

func thumbnailImageObjectKey(now time.Time, id mediadomain.AttachmentID) string {
	return fmt.Sprintf("thumbnails/%04d/%02d/%s.jpg", now.Year(), int(now.Month()), id.String())
}

func createJPEGThumbnail(mimeType string, fileBytes []byte) ([]byte, bool) {
	if mimeType != "image/jpeg" && mimeType != "image/png" {
		return nil, false
	}

	source, _, err := image.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		return nil, false
	}
	bounds := source.Bounds()
	if bounds.Dx() <= thumbnailMaxDimension && bounds.Dy() <= thumbnailMaxDimension {
		return nil, false
	}
	thumbnail := resizeForThumbnail(source, thumbnailMaxDimension)
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, thumbnail, &jpeg.Options{Quality: thumbnailJPEGQuality}); err != nil {
		return nil, false
	}
	return buffer.Bytes(), true
}

func resizeForThumbnail(source image.Image, maxDimension int) image.Image {
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	if maxDimension <= 0 {
		maxDimension = thumbnailMaxDimension
	}

	targetWidth := width
	targetHeight := height
	if width > maxDimension || height > maxDimension {
		if width >= height {
			targetWidth = maxDimension
			targetHeight = max(1, height*targetWidth/width)
		} else {
			targetHeight = maxDimension
			targetWidth = max(1, width*targetHeight/height)
		}
	}

	target := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	for y := range targetHeight {
		sourceY := bounds.Min.Y + y*height/targetHeight
		for x := range targetWidth {
			sourceX := bounds.Min.X + x*width/targetWidth
			target.Set(x, y, compositeOnWhite(source.At(sourceX, sourceY)))
		}
	}
	return target
}

func compositeOnWhite(source color.Color) color.RGBA {
	r, g, b, a := source.RGBA()
	if a == 0 {
		return color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	}
	return color.RGBA{
		R: uint8((r + (0xffff - a)) >> 8),
		G: uint8((g + (0xffff - a)) >> 8),
		B: uint8((b + (0xffff - a)) >> 8),
		A: 0xff,
	}
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
		ThumbnailURL:       attachment.ThumbnailURL(),
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

package mediausecase

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestSaveImageAttachmentCreatesReadyAttachment(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	uploaderID := userdomain.NewGeneratedUserID()
	var created mediadomain.Attachment
	repo := &fakeAttachmentRepository{
		createFunc: func(ctx context.Context, attachment mediadomain.Attachment) error {
			created = attachment
			return nil
		},
	}
	uc := NewUseCase(repo, nil, UploadLimits{}, func() time.Time { return now })

	result, err := uc.SaveImageAttachment(context.Background(), SaveImageAttachmentInput{
		UploaderID:      uploaderID,
		StorageProvider: mediadomain.StorageProviderR2.String(),
		Bucket:          "bucket",
		ObjectKey:       "images/2026/06/image.png",
		PublicURL:       "https://media.example.com/images/2026/06/image.png",
		SizeBytes:       123456,
		MimeType:        "image/png",
		AltText:         "Campus",
	})
	if err != nil {
		t.Fatalf("SaveImageAttachment returned error: %v", err)
	}
	if created.UploaderID() != uploaderID || created.Status() != mediadomain.AttachmentStatusReady {
		t.Fatalf("unexpected created attachment: %#v", created)
	}
	if result.Attachment.OwnerType != mediadomain.OwnerTypeNone.String() || result.Attachment.StorageProvider != mediadomain.StorageProviderR2.String() {
		t.Fatalf("unexpected result: %#v", result.Attachment)
	}
	if result.Attachment.ThumbnailURL != result.Attachment.PublicURL {
		t.Fatalf("expected thumbnail url fallback, got %#v", result.Attachment)
	}
}

func TestSaveImageAttachmentRejectsInvalidInput(t *testing.T) {
	uc := NewUseCase(&fakeAttachmentRepository{}, nil, UploadLimits{}, time.Now)

	tests := []struct {
		name  string
		input SaveImageAttachmentInput
		code  apperr.Code
	}{
		{name: "missing uploader", input: SaveImageAttachmentInput{}, code: apperr.CodeUnauthenticated},
		{name: "invalid provider", input: validInput(userdomain.NewGeneratedUserID()), code: apperr.CodeInvalidArgument},
	}
	tests[1].input.StorageProvider = "s3"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.SaveImageAttachment(context.Background(), tt.input)
			if !hasAppCode(err, tt.code) {
				t.Fatalf("expected %s, got %v", tt.code, err)
			}
		})
	}
}

func TestUploadImageStoresObjectAndCreatesAttachment(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	uploaderID := userdomain.NewGeneratedUserID()
	storage := &fakeObjectStorage{}
	var created mediadomain.Attachment
	repo := &fakeAttachmentRepository{
		createFunc: func(ctx context.Context, attachment mediadomain.Attachment) error {
			created = attachment
			return nil
		},
	}
	largePNG := pngBytesWithSize(800, 600)
	uc := NewUseCase(repo, storage, UploadLimits{ImageMaxBytes: len(largePNG) + 1024}, func() time.Time { return now })

	result, err := uc.UploadImage(context.Background(), UploadImageInput{
		UploaderID: uploaderID,
		FileBytes:  largePNG,
		AltText:    "Campus",
	})
	if err != nil {
		t.Fatalf("UploadImage returned error: %v", err)
	}
	if len(storage.putInputs) != 2 {
		t.Fatalf("expected original and thumbnail uploads, got %#v", storage.putInputs)
	}
	if storage.putInputs[0].ContentType != "image/png" || storage.putInputs[0].SizeBytes != int64(len(largePNG)) {
		t.Fatalf("unexpected original storage input: %#v", storage.putInputs[0])
	}
	if !strings.HasPrefix(storage.putInputs[0].ObjectKey, "images/2026/06/") || !strings.HasSuffix(storage.putInputs[0].ObjectKey, ".png") {
		t.Fatalf("unexpected original object key: %q", storage.putInputs[0].ObjectKey)
	}
	if storage.putInputs[1].ContentType != "image/jpeg" || storage.putInputs[1].SizeBytes <= 0 {
		t.Fatalf("unexpected thumbnail storage input: %#v", storage.putInputs[1])
	}
	if !strings.HasPrefix(storage.putInputs[1].ObjectKey, "thumbnails/2026/06/") || !strings.HasSuffix(storage.putInputs[1].ObjectKey, ".jpg") {
		t.Fatalf("unexpected thumbnail object key: %q", storage.putInputs[1].ObjectKey)
	}
	if _, err := jpeg.Decode(bytes.NewReader(storage.putBodies[1])); err != nil {
		t.Fatalf("expected jpeg thumbnail body: %v", err)
	}
	if created.UploaderID() != uploaderID || created.MimeType() != "image/png" || created.Status() != mediadomain.AttachmentStatusReady {
		t.Fatalf("unexpected created attachment: %#v", created)
	}
	if created.ThumbnailObjectKey() != storage.putInputs[1].ObjectKey {
		t.Fatalf("expected thumbnail object key %q, got %q", storage.putInputs[1].ObjectKey, created.ThumbnailObjectKey())
	}
	if created.Width() == nil || *created.Width() != 800 || created.Height() == nil || *created.Height() != 600 {
		t.Fatalf("expected decoded dimensions, got width=%v height=%v", created.Width(), created.Height())
	}
	if result.Attachment.PublicURL == "" || result.Attachment.ObjectKey == "" {
		t.Fatalf("expected public url and object key, got %#v", result.Attachment)
	}
	if result.Attachment.Width == nil || *result.Attachment.Width != 800 || result.Attachment.Height == nil || *result.Attachment.Height != 600 {
		t.Fatalf("expected result dimensions, got width=%v height=%v", result.Attachment.Width, result.Attachment.Height)
	}
	if result.Attachment.ThumbnailURL == result.Attachment.PublicURL || !strings.Contains(result.Attachment.ThumbnailURL, "/thumbnails/2026/06/") {
		t.Fatalf("expected independent thumbnail url, got %#v", result.Attachment)
	}
}

func TestUploadImageFallsBackToOriginalURLWhenImageIsAlreadySmall(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	storage := &fakeObjectStorage{}
	uc := NewUseCase(&fakeAttachmentRepository{}, storage, UploadLimits{ImageMaxBytes: 1024}, func() time.Time { return now })

	result, err := uc.UploadImage(context.Background(), UploadImageInput{
		UploaderID: userdomain.NewGeneratedUserID(),
		FileBytes:  pngBytes(),
	})
	if err != nil {
		t.Fatalf("UploadImage returned error: %v", err)
	}
	if len(storage.putInputs) != 1 {
		t.Fatalf("expected only original upload for small image, got %#v", storage.putInputs)
	}
	if result.Attachment.ThumbnailURL != result.Attachment.PublicURL || result.Attachment.ThumbnailObjectKey != "" {
		t.Fatalf("expected small image thumbnail fallback, got %#v", result.Attachment)
	}
}

func TestUploadImageFallsBackToOriginalURLWhenThumbnailGenerationIsUnsupported(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	storage := &fakeObjectStorage{}
	uc := NewUseCase(&fakeAttachmentRepository{}, storage, UploadLimits{ImageMaxBytes: 1024}, func() time.Time { return now })

	result, err := uc.UploadImage(context.Background(), UploadImageInput{
		UploaderID: userdomain.NewGeneratedUserID(),
		FileBytes:  webpBytes(),
	})
	if err != nil {
		t.Fatalf("UploadImage returned error: %v", err)
	}
	if len(storage.putInputs) != 1 {
		t.Fatalf("expected only original upload for unsupported thumbnail generation, got %#v", storage.putInputs)
	}
	if result.Attachment.MimeType != "image/webp" || result.Attachment.ThumbnailURL != result.Attachment.PublicURL {
		t.Fatalf("expected webp thumbnail fallback, got %#v", result.Attachment)
	}
}

func TestUploadImageFallsBackToOriginalURLWhenThumbnailUploadFails(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	storage := &fakeObjectStorage{
		putErrByIndex: map[int]error{
			1: errors.New("thumbnail storage unavailable"),
		},
	}
	largePNG := pngBytesWithSize(800, 600)
	uc := NewUseCase(&fakeAttachmentRepository{}, storage, UploadLimits{ImageMaxBytes: len(largePNG) + 1024}, func() time.Time { return now })

	result, err := uc.UploadImage(context.Background(), UploadImageInput{
		UploaderID: userdomain.NewGeneratedUserID(),
		FileBytes:  largePNG,
	})
	if err != nil {
		t.Fatalf("UploadImage returned error: %v", err)
	}
	if len(storage.putInputs) != 2 {
		t.Fatalf("expected original and attempted thumbnail uploads, got %#v", storage.putInputs)
	}
	if result.Attachment.ThumbnailURL != result.Attachment.PublicURL || result.Attachment.ThumbnailObjectKey != "" {
		t.Fatalf("expected thumbnail fallback after thumbnail upload failure, got %#v", result.Attachment)
	}
}

func TestUploadImageRejectsInvalidInput(t *testing.T) {
	uc := NewUseCase(&fakeAttachmentRepository{}, &fakeObjectStorage{}, UploadLimits{ImageMaxBytes: 4}, time.Now)

	tests := []struct {
		name  string
		input UploadImageInput
		code  apperr.Code
	}{
		{name: "missing uploader", input: UploadImageInput{FileBytes: pngBytes()}, code: apperr.CodeUnauthenticated},
		{name: "empty file", input: UploadImageInput{UploaderID: userdomain.NewGeneratedUserID()}, code: apperr.CodeInvalidArgument},
		{name: "too large", input: UploadImageInput{UploaderID: userdomain.NewGeneratedUserID(), FileBytes: pngBytes()}, code: apperr.CodeInvalidArgument},
		{name: "invalid mime", input: UploadImageInput{UploaderID: userdomain.NewGeneratedUserID(), FileBytes: []byte("text")}, code: apperr.CodeInvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.UploadImage(context.Background(), tt.input)
			if !hasAppCode(err, tt.code) {
				t.Fatalf("expected %s, got %v", tt.code, err)
			}
		})
	}
}

func validInput(uploaderID userdomain.UserID) SaveImageAttachmentInput {
	return SaveImageAttachmentInput{
		UploaderID:      uploaderID,
		StorageProvider: mediadomain.StorageProviderLocal.String(),
		Bucket:          "local",
		ObjectKey:       "images/test.png",
		PublicURL:       "http://localhost:8080/uploads/images/test.png",
		SizeBytes:       100,
		MimeType:        "image/png",
	}
}

type fakeAttachmentRepository struct {
	createFunc                func(ctx context.Context, attachment mediadomain.Attachment) error
	findByIDFunc              func(ctx context.Context, id mediadomain.AttachmentID) (*mediadomain.Attachment, error)
	listCleanupCandidatesFunc func(ctx context.Context, unboundReadyBefore time.Time, failedOrBlockedBefore time.Time, limit int) ([]mediadomain.Attachment, error)
	takeCleanupCandidatesFunc func(ctx context.Context, unboundReadyBefore time.Time, failedOrBlockedBefore time.Time, limit int) ([]mediadomain.Attachment, error)
}

type fakeObjectStorage struct {
	putInputs         []PutObjectInput
	putBodies         [][]byte
	result            PutObjectResult
	err               error
	putErrByIndex     map[int]error
	deletedObjectKeys []string
	deleteErrByKey    map[string]error
}

func (f *fakeObjectStorage) PutObject(ctx context.Context, input PutObjectInput) (PutObjectResult, error) {
	body, err := io.ReadAll(input.Body)
	if err != nil {
		return PutObjectResult{}, err
	}
	f.putInputs = append(f.putInputs, input)
	f.putBodies = append(f.putBodies, body)
	if f.err != nil {
		return PutObjectResult{}, f.err
	}
	putIndex := len(f.putInputs) - 1
	if f.putErrByIndex != nil {
		if err, ok := f.putErrByIndex[putIndex]; ok {
			return PutObjectResult{}, err
		}
	}
	result := f.result
	if result.StorageProvider == "" {
		result.StorageProvider = mediadomain.StorageProviderLocal.String()
	}
	if result.Bucket == "" {
		result.Bucket = "local"
	}
	if result.ObjectKey == "" {
		result.ObjectKey = input.ObjectKey
	}
	if result.PublicURL == "" {
		result.PublicURL = "http://localhost:8080/uploads/" + input.ObjectKey
	}
	return result, nil
}

func (f *fakeObjectStorage) DeleteObject(ctx context.Context, objectKey string) error {
	f.deletedObjectKeys = append(f.deletedObjectKeys, objectKey)
	if f.deleteErrByKey != nil {
		if err, ok := f.deleteErrByKey[objectKey]; ok {
			return err
		}
	}
	return nil
}

func (f *fakeAttachmentRepository) Create(ctx context.Context, attachment mediadomain.Attachment) error {
	if f.createFunc != nil {
		return f.createFunc(ctx, attachment)
	}
	return nil
}

func (f *fakeAttachmentRepository) FindByID(ctx context.Context, id mediadomain.AttachmentID) (*mediadomain.Attachment, error) {
	if f.findByIDFunc != nil {
		return f.findByIDFunc(ctx, id)
	}
	return nil, apperr.New(apperr.CodeNotFound, "attachment not found")
}

func (f *fakeAttachmentRepository) ListCleanupCandidates(ctx context.Context, unboundReadyBefore time.Time, failedOrBlockedBefore time.Time, limit int) ([]mediadomain.Attachment, error) {
	if f.listCleanupCandidatesFunc != nil {
		return f.listCleanupCandidatesFunc(ctx, unboundReadyBefore, failedOrBlockedBefore, limit)
	}
	return nil, nil
}

func (f *fakeAttachmentRepository) TakeCleanupCandidates(ctx context.Context, unboundReadyBefore time.Time, failedOrBlockedBefore time.Time, limit int) ([]mediadomain.Attachment, error) {
	if f.takeCleanupCandidatesFunc != nil {
		return f.takeCleanupCandidatesFunc(ctx, unboundReadyBefore, failedOrBlockedBefore, limit)
	}
	return nil, nil
}

func hasAppCode(err error, code apperr.Code) bool {
	if err == nil {
		return false
	}
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Code() == code
}

func pngBytes() []byte {
	return pngBytesWithSize(1, 1)
}

func pngBytesWithSize(width int, height int) []byte {
	imageData := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			imageData.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 0xcc, A: 0xff})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, imageData); err != nil {
		panic(err)
	}
	return buffer.Bytes()
}

func webpBytes() []byte {
	return []byte("RIFF\x00\x00\x00\x00WEBPVP8 ")
}

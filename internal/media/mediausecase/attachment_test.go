package mediausecase

import (
	"context"
	"encoding/base64"
	"errors"
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
	storage := &fakeObjectStorage{
		result: PutObjectResult{
			StorageProvider: mediadomain.StorageProviderLocal.String(),
			Bucket:          "local",
			PublicURL:       "http://localhost:8080/uploads/image.png",
		},
	}
	var created mediadomain.Attachment
	repo := &fakeAttachmentRepository{
		createFunc: func(ctx context.Context, attachment mediadomain.Attachment) error {
			created = attachment
			return nil
		},
	}
	uc := NewUseCase(repo, storage, UploadLimits{ImageMaxBytes: 1024}, func() time.Time { return now })

	result, err := uc.UploadImage(context.Background(), UploadImageInput{
		UploaderID: uploaderID,
		FileBytes:  pngBytes(),
		AltText:    "Campus",
	})
	if err != nil {
		t.Fatalf("UploadImage returned error: %v", err)
	}
	if !storage.putCalled {
		t.Fatal("expected storage PutObject to be called")
	}
	if storage.input.ContentType != "image/png" || storage.input.SizeBytes != int64(len(pngBytes())) {
		t.Fatalf("unexpected storage input: %#v", storage.input)
	}
	if created.UploaderID() != uploaderID || created.MimeType() != "image/png" || created.Status() != mediadomain.AttachmentStatusReady {
		t.Fatalf("unexpected created attachment: %#v", created)
	}
	if created.Width() == nil || *created.Width() != 1 || created.Height() == nil || *created.Height() != 1 {
		t.Fatalf("expected decoded 1x1 dimensions, got width=%v height=%v", created.Width(), created.Height())
	}
	if result.Attachment.PublicURL == "" || result.Attachment.ObjectKey == "" {
		t.Fatalf("expected public url and object key, got %#v", result.Attachment)
	}
	if result.Attachment.Width == nil || *result.Attachment.Width != 1 || result.Attachment.Height == nil || *result.Attachment.Height != 1 {
		t.Fatalf("expected result dimensions, got width=%v height=%v", result.Attachment.Width, result.Attachment.Height)
	}
	if result.Attachment.ThumbnailURL != result.Attachment.PublicURL {
		t.Fatalf("expected thumbnail url fallback, got %#v", result.Attachment)
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
	putCalled         bool
	input             PutObjectInput
	result            PutObjectResult
	err               error
	deletedObjectKeys []string
	deleteErrByKey    map[string]error
}

func (f *fakeObjectStorage) PutObject(ctx context.Context, input PutObjectInput) (PutObjectResult, error) {
	f.putCalled = true
	f.input = input
	if f.err != nil {
		return PutObjectResult{}, f.err
	}
	result := f.result
	if result.ObjectKey == "" {
		result.ObjectKey = input.ObjectKey
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
	data, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")
	if err != nil {
		panic(err)
	}
	return data
}

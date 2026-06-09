package mediausecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestCleanupExpiredAttachmentsDryRunListsCandidates(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	candidate := mustCleanupAttachment(t, "images/old.png", "", mediadomain.AttachmentStatusReady, now.Add(-3*time.Hour))
	repo := &fakeAttachmentRepository{
		listCleanupCandidatesFunc: func(ctx context.Context, unboundReadyBefore time.Time, failedOrBlockedBefore time.Time, limit int) ([]mediadomain.Attachment, error) {
			if !unboundReadyBefore.Equal(now.Add(-2 * time.Hour)) {
				t.Fatalf("unexpected unbound cutoff: %v", unboundReadyBefore)
			}
			if !failedOrBlockedBefore.Equal(now.Add(-4 * time.Hour)) {
				t.Fatalf("unexpected failed cutoff: %v", failedOrBlockedBefore)
			}
			if limit != 5 {
				t.Fatalf("unexpected limit: %d", limit)
			}
			return []mediadomain.Attachment{candidate}, nil
		},
	}
	uc := NewUseCase(repo, nil, UploadLimits{}, func() time.Time { return now })

	result, err := uc.CleanupExpiredAttachments(context.Background(), CleanupExpiredAttachmentsInput{
		UnboundTTL: 2 * time.Hour,
		FailedTTL:  4 * time.Hour,
		Limit:      5,
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("CleanupExpiredAttachments returned error: %v", err)
	}
	if !result.DryRun || result.Candidates != 1 || result.AttachmentsDeleted != 0 || result.ObjectsDeleted != 0 || result.Failures != 0 {
		t.Fatalf("unexpected dry-run result: %#v", result)
	}
}

func TestCleanupExpiredAttachmentsDeletesClaimedObjects(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	first := mustCleanupAttachment(t, "images/old.png", "thumbs/old.png", mediadomain.AttachmentStatusReady, now.Add(-48*time.Hour))
	second := mustCleanupAttachment(t, "images/failed.png", "", mediadomain.AttachmentStatusFailed, now.Add(-48*time.Hour))
	repo := &fakeAttachmentRepository{
		takeCleanupCandidatesFunc: func(ctx context.Context, unboundReadyBefore time.Time, failedOrBlockedBefore time.Time, limit int) ([]mediadomain.Attachment, error) {
			if limit != 2 {
				t.Fatalf("unexpected limit: %d", limit)
			}
			return []mediadomain.Attachment{first, second}, nil
		},
	}
	storage := &fakeObjectStorage{}
	uc := NewUseCase(repo, storage, UploadLimits{}, func() time.Time { return now })

	result, err := uc.CleanupExpiredAttachments(context.Background(), CleanupExpiredAttachmentsInput{Limit: 2})
	if err != nil {
		t.Fatalf("CleanupExpiredAttachments returned error: %v", err)
	}
	if result.Candidates != 2 || result.AttachmentsDeleted != 2 || result.ObjectsDeleted != 3 || result.Failures != 0 {
		t.Fatalf("unexpected cleanup result: %#v", result)
	}
	wantDeleted := []string{"images/old.png", "thumbs/old.png", "images/failed.png"}
	if len(storage.deletedObjectKeys) != len(wantDeleted) {
		t.Fatalf("unexpected deleted object keys: %#v", storage.deletedObjectKeys)
	}
	for i, want := range wantDeleted {
		if storage.deletedObjectKeys[i] != want {
			t.Fatalf("unexpected deleted object keys: %#v", storage.deletedObjectKeys)
		}
	}
}

func TestCleanupExpiredAttachmentsCountsObjectDeleteFailures(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	candidate := mustCleanupAttachment(t, "images/fail.png", "", mediadomain.AttachmentStatusReady, now.Add(-48*time.Hour))
	repo := &fakeAttachmentRepository{
		takeCleanupCandidatesFunc: func(ctx context.Context, unboundReadyBefore time.Time, failedOrBlockedBefore time.Time, limit int) ([]mediadomain.Attachment, error) {
			return []mediadomain.Attachment{candidate}, nil
		},
	}
	storage := &fakeObjectStorage{
		deleteErrByKey: map[string]error{
			"images/fail.png": errors.New("storage unavailable"),
		},
	}
	uc := NewUseCase(repo, storage, UploadLimits{}, func() time.Time { return now })

	result, err := uc.CleanupExpiredAttachments(context.Background(), CleanupExpiredAttachmentsInput{})
	if err != nil {
		t.Fatalf("CleanupExpiredAttachments returned error: %v", err)
	}
	if result.Candidates != 1 || result.AttachmentsDeleted != 1 || result.ObjectsDeleted != 0 || result.Failures != 1 {
		t.Fatalf("unexpected cleanup result: %#v", result)
	}
}

func TestCleanupExpiredAttachmentsRequiresStorageForDelete(t *testing.T) {
	uc := NewUseCase(&fakeAttachmentRepository{}, nil, UploadLimits{}, time.Now)

	_, err := uc.CleanupExpiredAttachments(context.Background(), CleanupExpiredAttachmentsInput{})
	if err == nil {
		t.Fatal("expected missing storage error")
	}
}

func mustCleanupAttachment(t *testing.T, objectKey string, thumbnailObjectKey string, status mediadomain.AttachmentStatus, updatedAt time.Time) mediadomain.Attachment {
	t.Helper()

	attachment, err := mediadomain.RehydrateAttachment(mediadomain.NewAttachmentParams{
		ID:                 mediadomain.NewGeneratedAttachmentID(),
		OwnerType:          mediadomain.OwnerTypeNone,
		UploaderID:         userdomain.NewGeneratedUserID(),
		Kind:               mediadomain.AttachmentKindImage,
		StorageProvider:    mediadomain.StorageProviderLocal,
		Bucket:             "local",
		ObjectKey:          objectKey,
		PublicURL:          "http://localhost:8080/uploads/" + objectKey,
		ThumbnailObjectKey: thumbnailObjectKey,
		SizeBytes:          100,
		MimeType:           "image/png",
		Status:             status,
		CreatedAt:          updatedAt.Add(-time.Hour),
		UpdatedAt:          updatedAt,
	})
	if err != nil {
		t.Fatalf("RehydrateAttachment returned error: %v", err)
	}
	return *attachment
}

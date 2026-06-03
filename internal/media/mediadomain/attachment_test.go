package mediadomain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestNewReadyImageAttachment(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	width := 1200
	height := 800

	attachment, err := NewReadyImageAttachment(NewAttachmentParams{
		ID:              NewGeneratedAttachmentID(),
		UploaderID:      userdomain.NewGeneratedUserID(),
		StorageProvider: StorageProviderR2,
		Bucket:          "bucket",
		ObjectKey:       "images/2026/06/image.png",
		PublicURL:       "https://media.example.com/images/2026/06/image.png",
		Width:           &width,
		Height:          &height,
		SizeBytes:       123456,
		MimeType:        "image/png",
		AltText:         "Campus",
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("NewReadyImageAttachment returned error: %v", err)
	}
	if attachment.OwnerType() != OwnerTypeNone || attachment.OwnerID() != "" {
		t.Fatalf("expected unbound attachment, got owner=%s owner_id=%q", attachment.OwnerType(), attachment.OwnerID())
	}
	if attachment.Kind() != AttachmentKindImage || attachment.Status() != AttachmentStatusReady {
		t.Fatalf("unexpected kind/status: %s %s", attachment.Kind(), attachment.Status())
	}
	if attachment.MimeType() != "image/png" || attachment.AltText() != "Campus" {
		t.Fatalf("unexpected mime/alt: %s %s", attachment.MimeType(), attachment.AltText())
	}
}

func TestAttachmentRejectsInvalidInput(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		mutate func(*NewAttachmentParams)
	}{
		{name: "missing uploader", mutate: func(params *NewAttachmentParams) { params.UploaderID = "" }},
		{name: "invalid owner id", mutate: func(params *NewAttachmentParams) {
			params.OwnerType = OwnerTypePost
			params.OwnerID = "not-a-uuid"
		}},
		{name: "blank object key", mutate: func(params *NewAttachmentParams) { params.ObjectKey = " " }},
		{name: "invalid mime", mutate: func(params *NewAttachmentParams) { params.MimeType = "image/svg+xml" }},
		{name: "invalid size", mutate: func(params *NewAttachmentParams) { params.SizeBytes = 0 }},
		{name: "long alt", mutate: func(params *NewAttachmentParams) { params.AltText = strings.Repeat("x", MaxAltTextLength+1) }},
		{name: "invalid status", mutate: func(params *NewAttachmentParams) { params.Status = "unknown" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := validAttachmentParams(now)
			tt.mutate(&params)

			_, err := RehydrateAttachment(params)
			if !hasAppCode(err, apperr.CodeInvalidArgument) {
				t.Fatalf("expected invalid_argument, got %v", err)
			}
		})
	}
}

func validAttachmentParams(now time.Time) NewAttachmentParams {
	return NewAttachmentParams{
		ID:              NewGeneratedAttachmentID(),
		OwnerType:       OwnerTypeNone,
		UploaderID:      userdomain.NewGeneratedUserID(),
		Kind:            AttachmentKindImage,
		StorageProvider: StorageProviderLocal,
		Bucket:          "local",
		ObjectKey:       "images/test.png",
		PublicURL:       "http://localhost:8080/uploads/images/test.png",
		SizeBytes:       100,
		MimeType:        "image/png",
		Status:          AttachmentStatusReady,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
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

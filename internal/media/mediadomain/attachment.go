package mediadomain

import (
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

const MaxAltTextLength = 200

type AttachmentID string

func NewAttachmentID(raw string) (AttachmentID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "attachment id is required")
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return "", apperr.New(apperr.CodeInvalidArgument, "attachment id is invalid")
	}
	return AttachmentID(parsed.String()), nil
}

func NewGeneratedAttachmentID() AttachmentID {
	return AttachmentID(uuid.NewString())
}

func (id AttachmentID) String() string {
	return string(id)
}

type OwnerType string

const (
	OwnerTypeNone    OwnerType = "none"
	OwnerTypePost    OwnerType = "post"
	OwnerTypeComment OwnerType = "comment"
)

func NewOwnerType(raw string) (OwnerType, error) {
	switch OwnerType(strings.ToLower(strings.TrimSpace(raw))) {
	case OwnerTypeNone:
		return OwnerTypeNone, nil
	case OwnerTypePost:
		return OwnerTypePost, nil
	case OwnerTypeComment:
		return OwnerTypeComment, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "attachment owner type is invalid")
	}
}

func (ownerType OwnerType) String() string {
	return string(ownerType)
}

type AttachmentKind string

const AttachmentKindImage AttachmentKind = "image"

func NewAttachmentKind(raw string) (AttachmentKind, error) {
	switch AttachmentKind(strings.ToLower(strings.TrimSpace(raw))) {
	case AttachmentKindImage:
		return AttachmentKindImage, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "attachment kind is invalid")
	}
}

func (kind AttachmentKind) String() string {
	return string(kind)
}

type StorageProvider string

const (
	StorageProviderR2    StorageProvider = "r2"
	StorageProviderLocal StorageProvider = "local"
)

func NewStorageProvider(raw string) (StorageProvider, error) {
	switch StorageProvider(strings.ToLower(strings.TrimSpace(raw))) {
	case StorageProviderR2:
		return StorageProviderR2, nil
	case StorageProviderLocal:
		return StorageProviderLocal, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "attachment storage provider is invalid")
	}
}

func (provider StorageProvider) String() string {
	return string(provider)
}

type AttachmentStatus string

const (
	AttachmentStatusPending AttachmentStatus = "pending"
	AttachmentStatusReady   AttachmentStatus = "ready"
	AttachmentStatusBlocked AttachmentStatus = "blocked"
	AttachmentStatusFailed  AttachmentStatus = "failed"
)

func NewAttachmentStatus(raw string) (AttachmentStatus, error) {
	switch AttachmentStatus(strings.ToLower(strings.TrimSpace(raw))) {
	case AttachmentStatusPending:
		return AttachmentStatusPending, nil
	case AttachmentStatusReady:
		return AttachmentStatusReady, nil
	case AttachmentStatusBlocked:
		return AttachmentStatusBlocked, nil
	case AttachmentStatusFailed:
		return AttachmentStatusFailed, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "attachment status is invalid")
	}
}

func (status AttachmentStatus) String() string {
	return string(status)
}

func NewImageMimeType(raw string) (string, error) {
	mimeType := strings.ToLower(strings.TrimSpace(raw))
	switch mimeType {
	case "image/jpeg", "image/png", "image/webp":
		return mimeType, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "image mime type is invalid")
	}
}

type Attachment struct {
	id                 AttachmentID
	ownerType          OwnerType
	ownerID            *uuid.UUID
	uploaderID         userdomain.UserID
	kind               AttachmentKind
	storageProvider    StorageProvider
	bucket             string
	objectKey          string
	publicURL          string
	thumbnailObjectKey string
	width              *int
	height             *int
	sizeBytes          int64
	mimeType           string
	altText            string
	status             AttachmentStatus
	createdAt          time.Time
	updatedAt          time.Time
}

type NewAttachmentParams struct {
	ID                 AttachmentID
	OwnerType          OwnerType
	OwnerID            string
	UploaderID         userdomain.UserID
	Kind               AttachmentKind
	StorageProvider    StorageProvider
	Bucket             string
	ObjectKey          string
	PublicURL          string
	ThumbnailObjectKey string
	Width              *int
	Height             *int
	SizeBytes          int64
	MimeType           string
	AltText            string
	Status             AttachmentStatus
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func NewReadyImageAttachment(params NewAttachmentParams) (*Attachment, error) {
	params.OwnerType = OwnerTypeNone
	params.Kind = AttachmentKindImage
	params.Status = AttachmentStatusReady
	return RehydrateAttachment(params)
}

func RehydrateAttachment(params NewAttachmentParams) (*Attachment, error) {
	if strings.TrimSpace(params.ID.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "attachment id is required")
	}
	if _, err := NewOwnerType(params.OwnerType.String()); err != nil {
		return nil, err
	}
	ownerID, err := parseOwnerID(params.OwnerType, params.OwnerID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.UploaderID.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "attachment uploader id is required")
	}
	if _, err := NewAttachmentKind(params.Kind.String()); err != nil {
		return nil, err
	}
	if _, err := NewStorageProvider(params.StorageProvider.String()); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.Bucket) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "attachment bucket is required")
	}
	if strings.TrimSpace(params.ObjectKey) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "attachment object key is required")
	}
	if strings.TrimSpace(params.PublicURL) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "attachment public url is required")
	}
	width, err := validateOptionalPositiveInt(params.Width, "attachment width is invalid")
	if err != nil {
		return nil, err
	}
	height, err := validateOptionalPositiveInt(params.Height, "attachment height is invalid")
	if err != nil {
		return nil, err
	}
	if params.SizeBytes <= 0 {
		return nil, apperr.New(apperr.CodeInvalidArgument, "attachment size bytes is invalid")
	}
	mimeType, err := NewImageMimeType(params.MimeType)
	if err != nil {
		return nil, err
	}
	altText := strings.TrimSpace(params.AltText)
	if len([]rune(altText)) > MaxAltTextLength {
		return nil, apperr.New(apperr.CodeInvalidArgument, "attachment alt text is too long")
	}
	if _, err := NewAttachmentStatus(params.Status.String()); err != nil {
		return nil, err
	}
	if params.CreatedAt.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "attachment created time can't be zero")
	}
	if params.UpdatedAt.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "attachment updated time can't be zero")
	}
	if params.UpdatedAt.Before(params.CreatedAt) {
		return nil, apperr.New(apperr.CodeInvalidArgument, "attachment updated time can't be before created time")
	}

	return &Attachment{
		id:                 params.ID,
		ownerType:          params.OwnerType,
		ownerID:            ownerID,
		uploaderID:         params.UploaderID,
		kind:               params.Kind,
		storageProvider:    params.StorageProvider,
		bucket:             strings.TrimSpace(params.Bucket),
		objectKey:          strings.TrimSpace(params.ObjectKey),
		publicURL:          strings.TrimSpace(params.PublicURL),
		thumbnailObjectKey: strings.TrimSpace(params.ThumbnailObjectKey),
		width:              width,
		height:             height,
		sizeBytes:          params.SizeBytes,
		mimeType:           mimeType,
		altText:            altText,
		status:             params.Status,
		createdAt:          params.CreatedAt,
		updatedAt:          params.UpdatedAt,
	}, nil
}

func (attachment *Attachment) ID() AttachmentID {
	return attachment.id
}

func (attachment *Attachment) OwnerType() OwnerType {
	return attachment.ownerType
}

func (attachment *Attachment) OwnerID() string {
	if attachment.ownerID == nil {
		return ""
	}
	return attachment.ownerID.String()
}

func (attachment *Attachment) UploaderID() userdomain.UserID {
	return attachment.uploaderID
}

func (attachment *Attachment) Kind() AttachmentKind {
	return attachment.kind
}

func (attachment *Attachment) StorageProvider() StorageProvider {
	return attachment.storageProvider
}

func (attachment *Attachment) Bucket() string {
	return attachment.bucket
}

func (attachment *Attachment) ObjectKey() string {
	return attachment.objectKey
}

func (attachment *Attachment) PublicURL() string {
	return attachment.publicURL
}

func (attachment *Attachment) ThumbnailObjectKey() string {
	return attachment.thumbnailObjectKey
}

func (attachment *Attachment) Width() *int {
	return cloneInt(attachment.width)
}

func (attachment *Attachment) Height() *int {
	return cloneInt(attachment.height)
}

func (attachment *Attachment) SizeBytes() int64 {
	return attachment.sizeBytes
}

func (attachment *Attachment) MimeType() string {
	return attachment.mimeType
}

func (attachment *Attachment) AltText() string {
	return attachment.altText
}

func (attachment *Attachment) Status() AttachmentStatus {
	return attachment.status
}

func (attachment *Attachment) CreatedAt() time.Time {
	return attachment.createdAt
}

func (attachment *Attachment) UpdatedAt() time.Time {
	return attachment.updatedAt
}

func parseOwnerID(ownerType OwnerType, rawOwnerID string) (*uuid.UUID, error) {
	rawOwnerID = strings.TrimSpace(rawOwnerID)
	if ownerType == OwnerTypeNone {
		if rawOwnerID != "" {
			return nil, apperr.New(apperr.CodeInvalidArgument, "attachment owner id must be empty for owner none")
		}
		return nil, nil
	}
	if rawOwnerID == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "attachment owner id is required")
	}
	parsed, err := uuid.Parse(rawOwnerID)
	if err != nil {
		return nil, apperr.New(apperr.CodeInvalidArgument, "attachment owner id is invalid")
	}
	return &parsed, nil
}

func validateOptionalPositiveInt(value *int, message string) (*int, error) {
	if value == nil {
		return nil, nil
	}
	if *value <= 0 {
		return nil, apperr.New(apperr.CodeInvalidArgument, message)
	}
	copied := *value
	return &copied, nil
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

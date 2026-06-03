package mediahttp

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediausecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	media MediaUseCase
}

type MediaUseCase interface {
	UploadImage(ctx context.Context, input mediausecase.UploadImageInput) (mediausecase.UploadImageResult, error)
}

type attachmentResponse struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	URL       string    `json:"url"`
	Width     *int      `json:"width"`
	Height    *int      `json:"height"`
	SizeBytes int64     `json:"size_bytes"`
	MimeType  string    `json:"mime_type"`
	AltText   string    `json:"alt_text"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type uploadImageResponse struct {
	Attachment attachmentResponse `json:"attachment"`
}

func NewHandler(media MediaUseCase) *Handler {
	return &Handler{
		media: media,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.POST("/uploads/images", handler.UploadImage)
}

func (h *Handler) UploadImage(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "image file is required"))
		c.Abort()
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "image file is invalid"))
		c.Abort()
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "image file is invalid"))
		c.Abort()
		return
	}

	result, err := h.media.UploadImage(c.Request.Context(), mediausecase.UploadImageInput{
		UploaderID: userID,
		FileBytes:  fileBytes,
		AltText:    c.PostForm("alt_text"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, uploadImageResponse{
		Attachment: toAttachmentResponse(result.Attachment),
	})
}

func toAttachmentResponse(attachment mediausecase.Attachment) attachmentResponse {
	return attachmentResponse{
		ID:        attachment.ID,
		Kind:      attachment.Kind,
		URL:       attachment.PublicURL,
		Width:     attachment.Width,
		Height:    attachment.Height,
		SizeBytes: attachment.SizeBytes,
		MimeType:  attachment.MimeType,
		AltText:   attachment.AltText,
		Status:    attachment.Status,
		CreatedAt: attachment.CreatedAt,
	}
}

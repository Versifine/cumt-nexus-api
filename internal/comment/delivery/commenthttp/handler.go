package commenthttp

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	comments CommentUseCase
}

type CommentUseCase interface {
	PublishComment(ctx context.Context, input commentusecase.PublishCommentInput) (commentusecase.PublishCommentResult, error)
	ListPostComments(ctx context.Context, input commentusecase.ListPostCommentsInput) (commentusecase.ListPostCommentsResult, error)
	UpdateComment(ctx context.Context, input commentusecase.UpdateCommentInput) (commentusecase.UpdateCommentResult, error)
	DeleteComment(ctx context.Context, input commentusecase.DeleteCommentInput) (commentusecase.DeleteCommentResult, error)
}

type publishCommentRequest struct {
	Body          string   `json:"body" binding:"required"`
	ParentID      string   `json:"parent_id"`
	AttachmentIDs []string `json:"attachment_ids"`
}

type updateCommentRequest struct {
	Body string `json:"body" binding:"required"`
}

type commentResponse struct {
	ID             string               `json:"id"`
	PostID         string               `json:"post_id"`
	AuthorID       string               `json:"author_id"`
	ParentID       *string              `json:"parent_id"`
	Body           string               `json:"body"`
	BodyFormat     string               `json:"body_format"`
	Status         string               `json:"status"`
	Depth          int                  `json:"depth"`
	ReplyCount     int                  `json:"reply_count"`
	HasMoreReplies bool                 `json:"has_more_replies"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
	Attachments    []attachmentResponse `json:"attachments"`
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

type publishCommentResponse struct {
	Comment commentResponse `json:"comment"`
}

type listPostCommentsResponse struct {
	Comments []commentResponse `json:"comments"`
	View     string            `json:"view"`
	Sort     string            `json:"sort"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
	MaxDepth int               `json:"max_depth"`
}

func NewHandler(comments CommentUseCase) *Handler {
	return &Handler{
		comments: comments,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.POST("/posts/:id/comments", handler.PublishComment)
	group.GET("/posts/:id/comments", handler.ListPostComments)
	group.PATCH("/comments/:id", handler.UpdateComment)
	group.DELETE("/comments/:id", handler.DeleteComment)
}

func (h *Handler) PublishComment(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req publishCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid publish comment request"))
		c.Abort()
		return
	}

	result, err := h.comments.PublishComment(c.Request.Context(), commentusecase.PublishCommentInput{
		PostID:        c.Param("id"),
		AuthorID:      userID,
		ParentID:      req.ParentID,
		Body:          req.Body,
		AttachmentIDs: req.AttachmentIDs,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, publishCommentResponse{
		Comment: toCommentResponse(result.Comment),
	})
}

func (h *Handler) ListPostComments(c *gin.Context) {
	limit, err := parseOptionalIntQuery(c, "limit")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	offset, err := parseOptionalIntQuery(c, "offset")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	maxDepth, err := parseOptionalIntQuery(c, "max_depth")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	result, err := h.comments.ListPostComments(c.Request.Context(), commentusecase.ListPostCommentsInput{
		PostID:   c.Param("id"),
		View:     c.Query("view"),
		Sort:     c.Query("sort"),
		Limit:    limit,
		Offset:   offset,
		MaxDepth: maxDepth,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listPostCommentsResponse{
		Comments: make([]commentResponse, 0, len(result.Comments)),
		View:     result.View,
		Sort:     result.Sort,
		Limit:    result.Limit,
		Offset:   result.Offset,
		MaxDepth: result.MaxDepth,
	}
	for _, comment := range result.Comments {
		response.Comments = append(response.Comments, toCommentResponse(comment))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) UpdateComment(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req updateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid update comment request"))
		c.Abort()
		return
	}

	result, err := h.comments.UpdateComment(c.Request.Context(), commentusecase.UpdateCommentInput{
		CommentID: c.Param("id"),
		ActorID:   userID,
		Body:      req.Body,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, publishCommentResponse{
		Comment: toCommentResponse(result.Comment),
	})
}

func (h *Handler) DeleteComment(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	if _, err := h.comments.DeleteComment(c.Request.Context(), commentusecase.DeleteCommentInput{
		CommentID: c.Param("id"),
		ActorID:   userID,
	}); err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.Status(http.StatusNoContent)
}

func parseOptionalIntQuery(c *gin.Context, key string) (int, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, apperr.New(apperr.CodeInvalidArgument, "invalid "+key+" query")
	}
	return value, nil
}

func toCommentResponse(comment commentusecase.Comment) commentResponse {
	var parentID *string
	if strings.TrimSpace(comment.ParentID) != "" {
		parentID = &comment.ParentID
	}
	return commentResponse{
		ID:             comment.ID,
		PostID:         comment.PostID,
		AuthorID:       comment.AuthorID,
		ParentID:       parentID,
		Body:           comment.Body,
		BodyFormat:     comment.BodyFormat,
		Status:         comment.Status,
		Depth:          comment.Depth,
		ReplyCount:     comment.ReplyCount,
		HasMoreReplies: comment.HasMoreReplies,
		CreatedAt:      comment.CreatedAt,
		UpdatedAt:      comment.UpdatedAt,
		Attachments:    toAttachmentResponses(comment.Attachments),
	}
}

func toAttachmentResponses(attachments []commentusecase.Attachment) []attachmentResponse {
	response := make([]attachmentResponse, 0, len(attachments))
	for _, attachment := range attachments {
		response = append(response, attachmentResponse{
			ID:        attachment.ID,
			Kind:      attachment.Kind,
			URL:       attachment.URL,
			Width:     attachment.Width,
			Height:    attachment.Height,
			SizeBytes: attachment.SizeBytes,
			MimeType:  attachment.MimeType,
			AltText:   attachment.AltText,
			Status:    attachment.Status,
			CreatedAt: attachment.CreatedAt,
		})
	}
	return response
}

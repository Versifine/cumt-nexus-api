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
}

type publishCommentRequest struct {
	Body     string `json:"body" binding:"required"`
	ParentID string `json:"parent_id"`
}

type commentResponse struct {
	ID        string    `json:"id"`
	PostID    string    `json:"post_id"`
	AuthorID  string    `json:"author_id"`
	ParentID  string    `json:"parent_id,omitempty"`
	Body      string    `json:"body"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type publishCommentResponse struct {
	Comment commentResponse `json:"comment"`
}

type listPostCommentsResponse struct {
	Comments []commentResponse `json:"comments"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

func NewHandler(comments CommentUseCase) *Handler {
	return &Handler{
		comments: comments,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.POST("/posts/:id/comments", handler.PublishComment)
	group.GET("/posts/:id/comments", handler.ListPostComments)
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
		PostID:   c.Param("id"),
		AuthorID: userID,
		ParentID: req.ParentID,
		Body:     req.Body,
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

	result, err := h.comments.ListPostComments(c.Request.Context(), commentusecase.ListPostCommentsInput{
		PostID: c.Param("id"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listPostCommentsResponse{
		Comments: make([]commentResponse, 0, len(result.Comments)),
		Limit:    result.Limit,
		Offset:   result.Offset,
	}
	for _, comment := range result.Comments {
		response.Comments = append(response.Comments, toCommentResponse(comment))
	}

	c.JSON(http.StatusOK, response)
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
	return commentResponse{
		ID:        comment.ID,
		PostID:    comment.PostID,
		AuthorID:  comment.AuthorID,
		ParentID:  comment.ParentID,
		Body:      comment.Body,
		Status:    comment.Status,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}
}

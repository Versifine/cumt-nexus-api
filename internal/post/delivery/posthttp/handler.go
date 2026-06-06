package posthttp

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	posts PostUseCase
}

type PostUseCase interface {
	PublishPost(ctx context.Context, input postusecase.PublishPostInput) (postusecase.PublishPostResult, error)
	ListCommunityPosts(ctx context.Context, input postusecase.ListCommunityPostsInput) (postusecase.ListCommunityPostsResult, error)
	ListLatestPosts(ctx context.Context, input postusecase.ListLatestPostsInput) (postusecase.ListLatestPostsResult, error)
	GetPost(ctx context.Context, input postusecase.GetPostInput) (postusecase.GetPostResult, error)
	UpdatePost(ctx context.Context, input postusecase.UpdatePostInput) (postusecase.UpdatePostResult, error)
	DeletePost(ctx context.Context, input postusecase.DeletePostInput) (postusecase.DeletePostResult, error)
}

type publishPostRequest struct {
	Title         string   `json:"title" binding:"required"`
	Body          string   `json:"body" binding:"required"`
	AttachmentIDs []string `json:"attachment_ids"`
}

type updatePostRequest struct {
	Title string `json:"title" binding:"required"`
	Body  string `json:"body" binding:"required"`
}

type postResponse struct {
	ID            string               `json:"id"`
	CommunityID   string               `json:"community_id"`
	AuthorID      string               `json:"author_id"`
	Title         string               `json:"title"`
	Body          string               `json:"body"`
	BodyFormat    string               `json:"body_format"`
	Status        string               `json:"status"`
	UpvoteCount   int                  `json:"upvote_count"`
	DownvoteCount int                  `json:"downvote_count"`
	Score         int                  `json:"score"`
	MyVote        int                  `json:"my_vote"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	Attachments   []attachmentResponse `json:"attachments"`
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

type publishPostResponse struct {
	Post postResponse `json:"post"`
}

type listCommunityPostsResponse struct {
	Posts  []postResponse `json:"posts"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

type getPostResponse struct {
	Post postResponse `json:"post"`
}

func NewHandler(posts PostUseCase) *Handler {
	return &Handler{
		posts: posts,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	RegisterReadRoutes(group, handler)
	RegisterWriteRoutes(group, handler)
}

func RegisterReadRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/communities/:slug/posts", handler.ListCommunityPosts)
	group.GET("/posts", handler.ListLatestPosts)
	group.GET("/posts/:id", handler.GetPost)
}

func RegisterWriteRoutes(group *gin.RouterGroup, handler *Handler) {
	group.POST("/communities/:slug/posts", handler.PublishPost)
	group.PATCH("/posts/:id", handler.UpdatePost)
	group.DELETE("/posts/:id", handler.DeletePost)
}

func (h *Handler) PublishPost(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req publishPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid publish post request"))
		c.Abort()
		return
	}

	result, err := h.posts.PublishPost(c.Request.Context(), postusecase.PublishPostInput{
		CommunitySlug: c.Param("slug"),
		AuthorID:      userID,
		Title:         req.Title,
		Body:          req.Body,
		AttachmentIDs: req.AttachmentIDs,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, publishPostResponse{
		Post: toPostResponse(result.Post),
	})
}

func (h *Handler) ListCommunityPosts(c *gin.Context) {
	userID, _ := authcontext.CurrentUserID(c.Request.Context())

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

	result, err := h.posts.ListCommunityPosts(c.Request.Context(), postusecase.ListCommunityPostsInput{
		CommunitySlug: c.Param("slug"),
		ViewerID:      userID,
		Sort:          c.Query("sort"),
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listCommunityPostsResponse{
		Posts:  make([]postResponse, 0, len(result.Posts)),
		Limit:  result.Limit,
		Offset: result.Offset,
	}
	for _, post := range result.Posts {
		response.Posts = append(response.Posts, toPostResponse(post))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) ListLatestPosts(c *gin.Context) {
	userID, _ := authcontext.CurrentUserID(c.Request.Context())

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

	result, err := h.posts.ListLatestPosts(c.Request.Context(), postusecase.ListLatestPostsInput{
		ViewerID: userID,
		Sort:     c.Query("sort"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listCommunityPostsResponse{
		Posts:  make([]postResponse, 0, len(result.Posts)),
		Limit:  result.Limit,
		Offset: result.Offset,
	}
	for _, post := range result.Posts {
		response.Posts = append(response.Posts, toPostResponse(post))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetPost(c *gin.Context) {
	userID, _ := authcontext.CurrentUserID(c.Request.Context())

	result, err := h.posts.GetPost(c.Request.Context(), postusecase.GetPostInput{
		PostID:   c.Param("id"),
		ViewerID: userID,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, getPostResponse{
		Post: toPostResponse(result.Post),
	})
}

func (h *Handler) UpdatePost(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req updatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid update post request"))
		c.Abort()
		return
	}

	result, err := h.posts.UpdatePost(c.Request.Context(), postusecase.UpdatePostInput{
		PostID:  c.Param("id"),
		ActorID: userID,
		Title:   req.Title,
		Body:    req.Body,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, getPostResponse{
		Post: toPostResponse(result.Post),
	})
}

func (h *Handler) DeletePost(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	if _, err := h.posts.DeletePost(c.Request.Context(), postusecase.DeletePostInput{
		PostID:  c.Param("id"),
		ActorID: userID,
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

func toPostResponse(post postusecase.Post) postResponse {
	return postResponse{
		ID:            post.ID,
		CommunityID:   post.CommunityID,
		AuthorID:      post.AuthorID,
		Title:         post.Title,
		Body:          post.Body,
		BodyFormat:    post.BodyFormat,
		Status:        post.Status,
		UpvoteCount:   post.UpvoteCount,
		DownvoteCount: post.DownvoteCount,
		Score:         post.Score,
		MyVote:        post.MyVote,
		CreatedAt:     post.CreatedAt,
		UpdatedAt:     post.UpdatedAt,
		Attachments:   toAttachmentResponses(post.Attachments),
	}
}

func toAttachmentResponses(attachments []postusecase.Attachment) []attachmentResponse {
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

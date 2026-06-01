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
	GetPost(ctx context.Context, input postusecase.GetPostInput) (postusecase.GetPostResult, error)
}

type publishPostRequest struct {
	Title string `json:"title" binding:"required"`
	Body  string `json:"body" binding:"required"`
}

type postResponse struct {
	ID          string    `json:"id"`
	CommunityID string    `json:"community_id"`
	AuthorID    string    `json:"author_id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
	group.POST("/communities/:slug/posts", handler.PublishPost)
	group.GET("/communities/:slug/posts", handler.ListCommunityPosts)
	group.GET("/posts/:id", handler.GetPost)
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

func (h *Handler) GetPost(c *gin.Context) {
	result, err := h.posts.GetPost(c.Request.Context(), postusecase.GetPostInput{
		PostID: c.Param("id"),
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
		ID:          post.ID,
		CommunityID: post.CommunityID,
		AuthorID:    post.AuthorID,
		Title:       post.Title,
		Body:        post.Body,
		Status:      post.Status,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
	}
}

package controller

import (
	"context"
	"cumt-nexus-api/internal/service"
	"cumt-nexus-api/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PostSvc interface {
	CreatePost(ctx context.Context, userID uint64, input *service.CreatePostInput) (*service.CreatePostResponse, error)
	ListPosts(ctx context.Context, page, pageSize int) (*service.PostListResponse, error)
	GetPostDetail(ctx context.Context, postID uint64) (*service.PostDetailDTO, error)
}

type PostController struct {
	postSvc PostSvc
}

func NewPostController(postSvc PostSvc) *PostController {
	return &PostController{postSvc: postSvc}
}

// CreatePost 创建帖子，作者身份必须从鉴权中间件注入的上下文中读取。
func (pc *PostController) CreatePost(c *gin.Context) {
	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		code, msg := service.MapError(service.ErrParamInvalid)
		response.Fail(c, code, msg)
		return
	}

	value, exists := c.Get("user_id")
	if !exists {
		code, msg := service.MapError(service.ErrUnauthorized)
		response.Fail(c, code, msg)
		return
	}
	userID, ok := value.(uint64)
	if !ok {
		code, msg := service.MapError(service.ErrUnauthorized)
		response.Fail(c, code, msg)
		return
	}

	out, err := pc.postSvc.CreatePost(c.Request.Context(), userID, &service.CreatePostInput{
		Title:   req.Title,
		Content: req.Content,
	})
	if err != nil {
		code, msg := service.MapError(err)
		response.Fail(c, code, msg)
		return
	}

	response.Success(c, out)
}

// ListPosts 分页查询帖子列表。
func (pc *PostController) ListPosts(c *gin.Context) {
	page, err := parsePositiveInt(c.DefaultQuery("page", "1"))
	if err != nil {
		code, msg := service.MapError(service.ErrParamInvalid)
		response.Fail(c, code, msg)
		return
	}

	pageSize, err := parsePositiveInt(c.DefaultQuery("page_size", "10"))
	if err != nil {
		code, msg := service.MapError(service.ErrParamInvalid)
		response.Fail(c, code, msg)
		return
	}

	out, err := pc.postSvc.ListPosts(c.Request.Context(), page, pageSize)
	if err != nil {
		code, msg := service.MapError(err)
		response.Fail(c, code, msg)
		return
	}

	response.Success(c, out)
}

// GetPostDetail 查询单篇帖子详情。
func (pc *PostController) GetPostDetail(c *gin.Context) {
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || postID == 0 {
		code, msg := service.MapError(service.ErrParamInvalid)
		response.Fail(c, code, msg)
		return
	}

	out, err := pc.postSvc.GetPostDetail(c.Request.Context(), postID)
	if err != nil {
		code, msg := service.MapError(err)
		response.Fail(c, code, msg)
		return
	}

	response.Success(c, out)
}

func parsePositiveInt(value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0, service.ErrParamInvalid
	}

	return n, nil
}

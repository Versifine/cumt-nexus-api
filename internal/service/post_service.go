package service

import (
	"context"
	"cumt-nexus-api/internal/model"
	"cumt-nexus-api/internal/repository"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	defaultPostListPage     = 1
	defaultPostListPageSize = 10
	maxPostListPageSize     = 20
)

// PostRepo 定义帖子模块依赖的仓储能力。
type PostRepo interface {
	Create(ctx context.Context, post *model.Post) error
	List(ctx context.Context, query repository.PostListQuery) ([]*repository.PostListRow, int64, error)
	FindDetailByID(ctx context.Context, id uint64) (*repository.PostDetailRow, error)
}

type PostService struct {
	postRepo PostRepo
}

func NewPostService(postRepo PostRepo) *PostService {
	return &PostService{postRepo: postRepo}
}

// CreatePostInput 是发帖接口的业务输入。
type CreatePostInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// CreatePostResponse 是发帖成功后的返回结构。
type CreatePostResponse struct {
	PostID uint64 `json:"post_id"`
}

// PostAuthorDTO 是帖子作者的对外展示信息。
type PostAuthorDTO struct {
	ID        uint64 `json:"id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

// PostListItemDTO 是帖子列表项的返回结构。
type PostListItemDTO struct {
	ID        uint64         `json:"id"`
	Title     string         `json:"title"`
	ViewCount int64          `json:"view_count"`
	CreatedAt time.Time      `json:"created_at"`
	Author    *PostAuthorDTO `json:"author"`
}

// PostListResponse 是分页列表的统一输出结构。
type PostListResponse struct {
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	List     []*PostListItemDTO `json:"list"`
}

// PostDetailDTO 是帖子详情页的返回结构。
type PostDetailDTO struct {
	ID        uint64         `json:"id"`
	Title     string         `json:"title"`
	Content   string         `json:"content"`
	ViewCount int64          `json:"view_count"`
	CreatedAt time.Time      `json:"created_at"`
	Author    *PostAuthorDTO `json:"author"`
}

// CreatePost 创建帖子，作者 ID 必须来自鉴权上下文，而不是前端传参。
func (s *PostService) CreatePost(ctx context.Context, userID uint64, input *CreatePostInput) (*CreatePostResponse, error) {
	if userID == 0 || input == nil {
		return nil, ErrParamInvalid
	}

	title := strings.TrimSpace(input.Title)
	content := strings.TrimSpace(input.Content)
	if title == "" || content == "" {
		return nil, ErrParamInvalid
	}

	post := &model.Post{
		UserID:  userID,
		Title:   title,
		Content: content,
	}
	if err := s.postRepo.Create(ctx, post); err != nil {
		return nil, err
	}

	return &CreatePostResponse{PostID: post.ID}, nil
}

// ListPosts 分页获取帖子列表，并把仓储层的联表结果转换成对外 DTO。
func (s *PostService) ListPosts(ctx context.Context, page, pageSize int) (*PostListResponse, error) {
	page, pageSize = normalizePostPagination(page, pageSize)
	query := repository.PostListQuery{
		Offset: (page - 1) * pageSize,
		Limit:  pageSize,
	}

	rows, total, err := s.postRepo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	list := make([]*PostListItemDTO, 0, len(rows))
	for _, row := range rows {
		list = append(list, &PostListItemDTO{
			ID:        row.ID,
			Title:     row.Title,
			ViewCount: row.ViewCount,
			CreatedAt: row.CreatedAt,
			Author: &PostAuthorDTO{
				ID:        row.AuthorID,
				Nickname:  row.AuthorNickname,
				AvatarURL: row.AuthorAvatarURL,
			},
		})
	}

	return &PostListResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     list,
	}, nil
}

// GetPostDetail 获取单篇帖子详情。
func (s *PostService) GetPostDetail(ctx context.Context, postID uint64) (*PostDetailDTO, error) {
	if postID == 0 {
		return nil, ErrParamInvalid
	}

	row, err := s.postRepo.FindDetailByID(ctx, postID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrResourceNotFound
	}
	if err != nil {
		return nil, err
	}

	return &PostDetailDTO{
		ID:        row.ID,
		Title:     row.Title,
		Content:   row.Content,
		ViewCount: row.ViewCount,
		CreatedAt: row.CreatedAt,
		Author: &PostAuthorDTO{
			ID:        row.AuthorID,
			Nickname:  row.AuthorNickname,
			AvatarURL: row.AuthorAvatarURL,
		},
	}, nil
}

func normalizePostPagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = defaultPostListPage
	}
	if pageSize < 1 {
		pageSize = defaultPostListPageSize
	}
	if pageSize > maxPostListPageSize {
		pageSize = maxPostListPageSize
	}

	return page, pageSize
}

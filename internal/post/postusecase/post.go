package postusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communityusecase"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

const (
	DefaultPostListLimit = 20
	MaxPostListLimit     = 50
)

type CommunityPolicy interface {
	GetCommunityBySlug(ctx context.Context, input communityusecase.GetCommunityInput) (communityusecase.GetCommunityResult, error)
	CanPostInCommunity(ctx context.Context, input communityusecase.CanPostInCommunityInput) (communityusecase.CanPostInCommunityResult, error)
}

type PostUseCase struct {
	posts       PostRepository
	communities CommunityPolicy
	now         func() time.Time
}

type PublishPostInput struct {
	CommunitySlug string
	AuthorID      userdomain.UserID
	Title         string
	Body          string
}

type ListCommunityPostsInput struct {
	CommunitySlug string
	Limit         int
	Offset        int
}

type GetPostInput struct {
	PostID string
}

type PublishPostResult struct {
	Post Post
}

type ListCommunityPostsResult struct {
	Posts  []Post
	Limit  int
	Offset int
}

type GetPostResult struct {
	Post Post
}

type Post struct {
	ID          string
	CommunityID string
	AuthorID    string
	Title       string
	Body        string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewPostUseCase(posts PostRepository, communities CommunityPolicy, now func() time.Time) *PostUseCase {
	if now == nil {
		now = time.Now
	}

	return &PostUseCase{
		posts:       posts,
		communities: communities,
		now:         now,
	}
}

func (uc *PostUseCase) PublishPost(ctx context.Context, input PublishPostInput) (PublishPostResult, error) {
	if strings.TrimSpace(input.AuthorID.String()) == "" {
		return PublishPostResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}

	community, err := uc.communities.GetCommunityBySlug(ctx, communityusecase.GetCommunityInput{
		Slug: input.CommunitySlug,
	})
	if err != nil {
		return PublishPostResult{}, fmt.Errorf("get community for post publishing: %w", err)
	}

	if _, err := uc.communities.CanPostInCommunity(ctx, communityusecase.CanPostInCommunityInput{
		UserID:      input.AuthorID.String(),
		CommunityID: community.Community.ID,
	}); err != nil {
		return PublishPostResult{}, fmt.Errorf("check community posting permission: %w", err)
	}

	communityID, err := communitydomain.NewCommunityID(community.Community.ID)
	if err != nil {
		return PublishPostResult{}, err
	}
	title, err := postdomain.NewPostTitle(input.Title)
	if err != nil {
		return PublishPostResult{}, err
	}
	body, err := postdomain.NewPostBody(input.Body)
	if err != nil {
		return PublishPostResult{}, err
	}

	now := uc.now().UTC()
	post, err := postdomain.NewPost(postdomain.NewGeneratedPostID(), communityID, input.AuthorID, title, body, now)
	if err != nil {
		return PublishPostResult{}, err
	}

	if err := uc.posts.Create(ctx, *post); err != nil {
		return PublishPostResult{}, fmt.Errorf("create post: %w", err)
	}

	return PublishPostResult{
		Post: toPostDTO(*post),
	}, nil
}

func (uc *PostUseCase) ListCommunityPosts(ctx context.Context, input ListCommunityPostsInput) (ListCommunityPostsResult, error) {
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListCommunityPostsResult{}, err
	}

	community, err := uc.communities.GetCommunityBySlug(ctx, communityusecase.GetCommunityInput{
		Slug: input.CommunitySlug,
	})
	if err != nil {
		return ListCommunityPostsResult{}, fmt.Errorf("get community for post list: %w", err)
	}
	communityID, err := communitydomain.NewCommunityID(community.Community.ID)
	if err != nil {
		return ListCommunityPostsResult{}, err
	}

	posts, err := uc.posts.ListVisibleByCommunity(ctx, communityID, limit, offset)
	if err != nil {
		return ListCommunityPostsResult{}, fmt.Errorf("list community posts: %w", err)
	}

	result := ListCommunityPostsResult{
		Posts:  make([]Post, 0, len(posts)),
		Limit:  limit,
		Offset: offset,
	}
	for _, post := range posts {
		result.Posts = append(result.Posts, toPostDTO(post))
	}

	return result, nil
}

func (uc *PostUseCase) GetPost(ctx context.Context, input GetPostInput) (GetPostResult, error) {
	postID, err := postdomain.NewPostID(input.PostID)
	if err != nil {
		return GetPostResult{}, err
	}

	post, err := uc.posts.FindVisibleByID(ctx, postID)
	if err != nil {
		return GetPostResult{}, fmt.Errorf("find post: %w", err)
	}

	return GetPostResult{
		Post: toPostDTO(*post),
	}, nil
}

func normalizePagination(limit int, offset int) (int, int, error) {
	if limit < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "post list limit is invalid")
	}
	if offset < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "post list offset is invalid")
	}
	if limit == 0 {
		limit = DefaultPostListLimit
	}
	if limit > MaxPostListLimit {
		limit = MaxPostListLimit
	}

	return limit, offset, nil
}

func toPostDTO(post postdomain.Post) Post {
	return Post{
		ID:          post.ID().String(),
		CommunityID: post.CommunityID().String(),
		AuthorID:    post.AuthorID().String(),
		Title:       post.Title().String(),
		Body:        post.Body().String(),
		Status:      post.Status().String(),
		CreatedAt:   post.CreatedAt(),
		UpdatedAt:   post.UpdatedAt(),
	}
}

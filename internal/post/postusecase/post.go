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
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
)

const (
	DefaultPostListLimit = 20
	MaxPostListLimit     = 50
)

type PostListSort string

const (
	PostListSortNew PostListSort = "new"
	PostListSortHot PostListSort = "hot"
)

type CommunityPolicy interface {
	GetCommunityBySlug(ctx context.Context, input communityusecase.GetCommunityInput) (communityusecase.GetCommunityResult, error)
	CanPostInCommunity(ctx context.Context, input communityusecase.CanPostInCommunityInput) (communityusecase.CanPostInCommunityResult, error)
}

type PostUseCase struct {
	posts       PostRepository
	communities CommunityPolicy
	votes       VoteRepository
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
	ViewerID      userdomain.UserID
	Sort          string
	Limit         int
	Offset        int
}

type ListLatestPostsInput struct {
	ViewerID userdomain.UserID
	Sort     string
	Limit    int
	Offset   int
}

type GetPostInput struct {
	PostID   string
	ViewerID userdomain.UserID
}

type PublishPostResult struct {
	Post Post
}

type ListCommunityPostsResult struct {
	Posts  []Post
	Limit  int
	Offset int
}

type ListLatestPostsResult struct {
	Posts  []Post
	Limit  int
	Offset int
}

type GetPostResult struct {
	Post Post
}

type Post struct {
	ID            string
	CommunityID   string
	AuthorID      string
	Title         string
	Body          string
	Status        string
	UpvoteCount   int
	DownvoteCount int
	Score         int
	MyVote        int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewPostUseCase(posts PostRepository, communities CommunityPolicy, now func() time.Time, votes ...VoteRepository) *PostUseCase {
	if now == nil {
		now = time.Now
	}

	var voteRepo VoteRepository
	if len(votes) > 0 {
		voteRepo = votes[0]
	}

	return &PostUseCase{
		posts:       posts,
		communities: communities,
		votes:       voteRepo,
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
		Post: toPostDTO(*post, postVoteView{}),
	}, nil
}

func (uc *PostUseCase) ListCommunityPosts(ctx context.Context, input ListCommunityPostsInput) (ListCommunityPostsResult, error) {
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListCommunityPostsResult{}, err
	}
	sort, err := normalizePostListSort(input.Sort)
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

	posts, err := uc.posts.ListVisibleByCommunity(ctx, communityID, sort, limit, offset)
	if err != nil {
		return ListCommunityPostsResult{}, fmt.Errorf("list community posts: %w", err)
	}

	result := ListCommunityPostsResult{
		Posts:  make([]Post, 0, len(posts)),
		Limit:  limit,
		Offset: offset,
	}
	voteViews, err := uc.loadVoteViews(ctx, posts, input.ViewerID)
	if err != nil {
		return ListCommunityPostsResult{}, err
	}

	for _, post := range posts {
		result.Posts = append(result.Posts, toPostDTO(post, voteViews[post.ID()]))
	}

	return result, nil
}

func (uc *PostUseCase) ListLatestPosts(ctx context.Context, input ListLatestPostsInput) (ListLatestPostsResult, error) {
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListLatestPostsResult{}, err
	}
	sort, err := normalizePostListSort(input.Sort)
	if err != nil {
		return ListLatestPostsResult{}, err
	}

	posts, err := uc.posts.ListVisibleInPublicCommunities(ctx, sort, limit, offset)
	if err != nil {
		return ListLatestPostsResult{}, fmt.Errorf("list latest posts: %w", err)
	}

	voteViews, err := uc.loadVoteViews(ctx, posts, input.ViewerID)
	if err != nil {
		return ListLatestPostsResult{}, err
	}

	result := ListLatestPostsResult{
		Posts:  make([]Post, 0, len(posts)),
		Limit:  limit,
		Offset: offset,
	}
	for _, post := range posts {
		result.Posts = append(result.Posts, toPostDTO(post, voteViews[post.ID()]))
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

	voteViews, err := uc.loadVoteViews(ctx, []postdomain.Post{*post}, input.ViewerID)
	if err != nil {
		return GetPostResult{}, err
	}

	return GetPostResult{
		Post: toPostDTO(*post, voteViews[post.ID()]),
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

func normalizePostListSort(raw string) (PostListSort, error) {
	sort := PostListSort(strings.ToLower(strings.TrimSpace(raw)))
	if sort == "" {
		return PostListSortNew, nil
	}
	switch sort {
	case PostListSortNew, PostListSortHot:
		return sort, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "post list sort is invalid")
	}
}

type postVoteView struct {
	upvoteCount   int
	downvoteCount int
	myVote        int
}

func (uc *PostUseCase) loadVoteViews(ctx context.Context, posts []postdomain.Post, viewerID userdomain.UserID) (map[postdomain.PostID]postVoteView, error) {
	views := make(map[postdomain.PostID]postVoteView, len(posts))
	if len(posts) == 0 || uc.votes == nil {
		return views, nil
	}

	postIDs := make([]postdomain.PostID, 0, len(posts))
	for _, post := range posts {
		postIDs = append(postIDs, post.ID())
	}

	summaries, err := uc.votes.SummarizeByPostIDs(ctx, postIDs)
	if err != nil {
		return nil, fmt.Errorf("summarize post votes: %w", err)
	}

	myVotes := map[postdomain.PostID]votedomain.VoteValue{}
	if strings.TrimSpace(viewerID.String()) != "" {
		myVotes, err = uc.votes.FindByPostIDsAndUser(ctx, postIDs, viewerID)
		if err != nil {
			return nil, fmt.Errorf("find post votes by viewer: %w", err)
		}
	}

	for _, postID := range postIDs {
		summary := summaries[postID]
		myVote := 0
		if value, ok := myVotes[postID]; ok {
			myVote = value.Int()
		}
		views[postID] = postVoteView{
			upvoteCount:   summary.UpvoteCount,
			downvoteCount: summary.DownvoteCount,
			myVote:        myVote,
		}
	}

	return views, nil
}

func toPostDTO(post postdomain.Post, voteView postVoteView) Post {
	score := voteView.upvoteCount - voteView.downvoteCount
	return Post{
		ID:            post.ID().String(),
		CommunityID:   post.CommunityID().String(),
		AuthorID:      post.AuthorID().String(),
		Title:         post.Title().String(),
		Body:          post.Body().String(),
		Status:        post.Status().String(),
		UpvoteCount:   voteView.upvoteCount,
		DownvoteCount: voteView.downvoteCount,
		Score:         score,
		MyVote:        voteView.myVote,
		CreatedAt:     post.CreatedAt(),
		UpdatedAt:     post.UpdatedAt(),
	}
}

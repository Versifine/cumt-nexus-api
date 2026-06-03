package postusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communityusecase"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
)

const (
	DefaultPostListLimit = 20
	MaxPostListLimit     = 50
	PostBodyFormat       = "markdown"
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
	posts             PostRepository
	communities       CommunityPolicy
	votes             VoteRepository
	attachments       AttachmentRepository
	postImageMaxCount int
	now               func() time.Time
}

type PublishPostInput struct {
	CommunitySlug string
	AuthorID      userdomain.UserID
	Title         string
	Body          string
	AttachmentIDs []string
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

type UpdatePostInput struct {
	PostID  string
	ActorID userdomain.UserID
	Title   string
	Body    string
}

type DeletePostInput struct {
	PostID  string
	ActorID userdomain.UserID
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

type UpdatePostResult struct {
	Post Post
}

type DeletePostResult struct{}

type Post struct {
	ID            string
	CommunityID   string
	AuthorID      string
	Title         string
	Body          string
	BodyFormat    string
	Status        string
	UpvoteCount   int
	DownvoteCount int
	Score         int
	MyVote        int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Attachments   []Attachment
}

type Attachment struct {
	ID        string
	Kind      string
	URL       string
	Width     *int
	Height    *int
	SizeBytes int64
	MimeType  string
	AltText   string
	Status    string
	CreatedAt time.Time
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
		posts:             posts,
		communities:       communities,
		votes:             voteRepo,
		postImageMaxCount: 9,
		now:               now,
	}
}

func NewPostUseCaseWithAttachments(posts PostRepository, communities CommunityPolicy, attachments AttachmentRepository, postImageMaxCount int, now func() time.Time, votes ...VoteRepository) *PostUseCase {
	uc := NewPostUseCase(posts, communities, now, votes...)
	uc.attachments = attachments
	if postImageMaxCount > 0 {
		uc.postImageMaxCount = postImageMaxCount
	}
	return uc
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
	attachmentIDs, err := parseAttachmentIDs(input.AttachmentIDs, uc.postImageMaxCount)
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
	attachments, err := uc.bindPostAttachments(ctx, post.ID(), input.AuthorID, attachmentIDs, now)
	if err != nil {
		return PublishPostResult{}, err
	}

	return PublishPostResult{
		Post: toPostDTO(*post, postVoteView{}, attachments),
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
	attachmentViews, err := uc.loadAttachmentViews(ctx, posts)
	if err != nil {
		return ListCommunityPostsResult{}, err
	}

	for _, post := range posts {
		result.Posts = append(result.Posts, toPostDTO(post, voteViews[post.ID()], attachmentViews[post.ID()]))
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
	attachmentViews, err := uc.loadAttachmentViews(ctx, posts)
	if err != nil {
		return ListLatestPostsResult{}, err
	}

	result := ListLatestPostsResult{
		Posts:  make([]Post, 0, len(posts)),
		Limit:  limit,
		Offset: offset,
	}
	for _, post := range posts {
		result.Posts = append(result.Posts, toPostDTO(post, voteViews[post.ID()], attachmentViews[post.ID()]))
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
	attachmentViews, err := uc.loadAttachmentViews(ctx, []postdomain.Post{*post})
	if err != nil {
		return GetPostResult{}, err
	}

	return GetPostResult{
		Post: toPostDTO(*post, voteViews[post.ID()], attachmentViews[post.ID()]),
	}, nil
}

func (uc *PostUseCase) UpdatePost(ctx context.Context, input UpdatePostInput) (UpdatePostResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return UpdatePostResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}

	postID, err := postdomain.NewPostID(input.PostID)
	if err != nil {
		return UpdatePostResult{}, err
	}
	post, err := uc.posts.FindVisibleByID(ctx, postID)
	if err != nil {
		return UpdatePostResult{}, fmt.Errorf("find post for update: %w", err)
	}
	if post.AuthorID() != input.ActorID {
		return UpdatePostResult{}, apperr.New(apperr.CodeForbidden, "only the post author can update post")
	}

	title, err := postdomain.NewPostTitle(input.Title)
	if err != nil {
		return UpdatePostResult{}, err
	}
	body, err := postdomain.NewPostBody(input.Body)
	if err != nil {
		return UpdatePostResult{}, err
	}

	if err := post.Edit(title, body, uc.now().UTC()); err != nil {
		return UpdatePostResult{}, err
	}
	if err := uc.posts.UpdateContent(ctx, *post); err != nil {
		return UpdatePostResult{}, fmt.Errorf("update post content: %w", err)
	}

	voteViews, err := uc.loadVoteViews(ctx, []postdomain.Post{*post}, input.ActorID)
	if err != nil {
		return UpdatePostResult{}, err
	}
	attachmentViews, err := uc.loadAttachmentViews(ctx, []postdomain.Post{*post})
	if err != nil {
		return UpdatePostResult{}, err
	}

	return UpdatePostResult{
		Post: toPostDTO(*post, voteViews[post.ID()], attachmentViews[post.ID()]),
	}, nil
}

func (uc *PostUseCase) DeletePost(ctx context.Context, input DeletePostInput) (DeletePostResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return DeletePostResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}

	postID, err := postdomain.NewPostID(input.PostID)
	if err != nil {
		return DeletePostResult{}, err
	}
	post, err := uc.posts.FindVisibleByID(ctx, postID)
	if err != nil {
		return DeletePostResult{}, fmt.Errorf("find post for delete: %w", err)
	}
	if post.AuthorID() != input.ActorID {
		return DeletePostResult{}, apperr.New(apperr.CodeForbidden, "only the post author can delete post")
	}
	if err := post.MarkDeleted(uc.now().UTC()); err != nil {
		return DeletePostResult{}, err
	}
	if err := uc.posts.MarkDeleted(ctx, *post); err != nil {
		return DeletePostResult{}, fmt.Errorf("delete post: %w", err)
	}

	return DeletePostResult{}, nil
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

func parseAttachmentIDs(rawIDs []string, maxCount int) ([]mediadomain.AttachmentID, error) {
	if len(rawIDs) == 0 {
		return []mediadomain.AttachmentID{}, nil
	}
	if maxCount <= 0 || len(rawIDs) > maxCount {
		return nil, apperr.New(apperr.CodeInvalidArgument, "post image attachment count is invalid")
	}
	ids := make([]mediadomain.AttachmentID, 0, len(rawIDs))
	seen := make(map[mediadomain.AttachmentID]bool, len(rawIDs))
	for _, rawID := range rawIDs {
		id, err := mediadomain.NewAttachmentID(rawID)
		if err != nil {
			return nil, err
		}
		if seen[id] {
			return nil, apperr.New(apperr.CodeInvalidArgument, "attachment id is duplicated")
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
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

func (uc *PostUseCase) bindPostAttachments(ctx context.Context, postID postdomain.PostID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, now time.Time) ([]mediadomain.Attachment, error) {
	if len(attachmentIDs) == 0 {
		return []mediadomain.Attachment{}, nil
	}
	if uc.attachments == nil {
		return nil, apperr.New(apperr.CodeInvalidArgument, "post image attachments are not supported")
	}
	attachments, err := uc.attachments.BindReadyImagesToPost(ctx, postID, uploaderID, attachmentIDs, uc.postImageMaxCount, now)
	if err != nil {
		return nil, fmt.Errorf("bind post image attachments: %w", err)
	}
	return attachments, nil
}

func (uc *PostUseCase) loadAttachmentViews(ctx context.Context, posts []postdomain.Post) (map[postdomain.PostID][]mediadomain.Attachment, error) {
	views := make(map[postdomain.PostID][]mediadomain.Attachment, len(posts))
	if len(posts) == 0 || uc.attachments == nil {
		return views, nil
	}
	postIDs := make([]postdomain.PostID, 0, len(posts))
	for _, post := range posts {
		postIDs = append(postIDs, post.ID())
	}
	views, err := uc.attachments.ListReadyImagesByPostIDs(ctx, postIDs)
	if err != nil {
		return nil, fmt.Errorf("list post image attachments: %w", err)
	}
	return views, nil
}

func toPostDTO(post postdomain.Post, voteView postVoteView, attachments []mediadomain.Attachment) Post {
	score := voteView.upvoteCount - voteView.downvoteCount
	return Post{
		ID:            post.ID().String(),
		CommunityID:   post.CommunityID().String(),
		AuthorID:      post.AuthorID().String(),
		Title:         post.Title().String(),
		Body:          post.Body().String(),
		BodyFormat:    PostBodyFormat,
		Status:        post.Status().String(),
		UpvoteCount:   voteView.upvoteCount,
		DownvoteCount: voteView.downvoteCount,
		Score:         score,
		MyVote:        voteView.myVote,
		CreatedAt:     post.CreatedAt(),
		UpdatedAt:     post.UpdatedAt(),
		Attachments:   toAttachmentDTOs(attachments),
	}
}

func toAttachmentDTOs(attachments []mediadomain.Attachment) []Attachment {
	result := make([]Attachment, 0, len(attachments))
	for _, attachment := range attachments {
		result = append(result, Attachment{
			ID:        attachment.ID().String(),
			Kind:      attachment.Kind().String(),
			URL:       attachment.PublicURL(),
			Width:     attachment.Width(),
			Height:    attachment.Height(),
			SizeBytes: attachment.SizeBytes(),
			MimeType:  attachment.MimeType(),
			AltText:   attachment.AltText(),
			Status:    attachment.Status().String(),
			CreatedAt: attachment.CreatedAt(),
		})
	}
	return result
}

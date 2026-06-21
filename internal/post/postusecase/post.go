package postusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communityusecase"
	"github.com/Versifine/cumt-nexus-api/internal/effect/effectusecase"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
	"github.com/Versifine/cumt-nexus-api/internal/mention"
	platformsettings "github.com/Versifine/cumt-nexus-api/internal/platform/settings"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/progression/progressionusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
	"github.com/google/uuid"
)

const (
	DefaultPostListLimit  = 20
	MaxPostListLimit      = 50
	PostFormat            = "nexus_markdown"
	DefaultPostExcerptMax = 180
	MaxContentRefCount    = 50
	MaxContentRefIDLength = 2048
	ContentRefKindImage   = "image"
	ContentRefKindLink    = "link_preview"
	ContentRefKindEmbed   = "embed"
)

type PostListSort string

const (
	PostListSortBest   PostListSort = "best"
	PostListSortHot    PostListSort = "hot"
	PostListSortNew    PostListSort = "new"
	PostListSortTop    PostListSort = "top"
	PostListSortRising PostListSort = "rising"
)

type PostFeedSource string

const (
	PostFeedSourceAll         PostFeedSource = "all"
	PostFeedSourceRecommended PostFeedSource = "recommended"
	PostFeedSourceFollowing   PostFeedSource = "following"
)

type PostListTimeRange string

const (
	PostListTimeRangeAll   PostListTimeRange = "all"
	PostListTimeRangeHour  PostListTimeRange = "hour"
	PostListTimeRangeDay   PostListTimeRange = "day"
	PostListTimeRangeWeek  PostListTimeRange = "week"
	PostListTimeRangeMonth PostListTimeRange = "month"
	PostListTimeRangeYear  PostListTimeRange = "year"
)

type CommunityPolicy interface {
	GetCommunityBySlug(ctx context.Context, input communityusecase.GetCommunityInput) (communityusecase.GetCommunityResult, error)
	CanPostInCommunity(ctx context.Context, input communityusecase.CanPostInCommunityInput) (communityusecase.CanPostInCommunityResult, error)
}

type PostUseCase struct {
	posts             PostRepository
	communities       CommunityPolicy
	votes             VoteRepository
	saves             PostSaveRepository
	attachments       AttachmentRepository
	users             PublicUserFinder
	notifications     NotificationPublisher
	progression       XPRecorder
	points            PointRecorder
	metadata          PostMetadataRepository
	contentRefs       ContentRefRepository
	effects           PostEffectRepository
	settingsReader    platformsettings.Reader
	postImageMaxCount int
	now               func() time.Time
}

type PublishPostInput struct {
	CommunitySlug string
	AuthorID      userdomain.UserID
	Title         string
	Body          string
	AttachmentIDs []string
	ContentRefs   []ContentRefInput
}

type ListCommunityPostsInput struct {
	CommunitySlug string
	ViewerID      userdomain.UserID
	Sort          string
	TimeRange     string
	Limit         int
	Offset        int
}

type ListLatestPostsInput struct {
	ViewerID  userdomain.UserID
	Source    string
	Sort      string
	TimeRange string
	Limit     int
	Offset    int
}

type ListUserPostsInput struct {
	Username  string
	ViewerID  userdomain.UserID
	Sort      string
	TimeRange string
	Limit     int
	Offset    int
}

type ListSavedPostsInput struct {
	UserID userdomain.UserID
	Limit  int
	Offset int
}

type GetPostInput struct {
	PostID   string
	ViewerID userdomain.UserID
}

type SavePostInput struct {
	PostID string
	UserID userdomain.UserID
}

type DeletePostSaveInput struct {
	PostID string
	UserID userdomain.UserID
}

type UpdatePostInput struct {
	PostID        string
	ActorID       userdomain.UserID
	Title         string
	Body          string
	AttachmentIDs *[]string
	ContentRefs   *[]ContentRefInput
}

type DeletePostInput struct {
	PostID  string
	ActorID userdomain.UserID
}

type PublishPostResult struct {
	Post Post
}

type ListCommunityPostsResult struct {
	Posts      []Post
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type ListLatestPostsResult struct {
	Posts      []Post
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type ListUserPostsResult struct {
	Posts      []Post
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type ListSavedPostsResult struct {
	Posts      []Post
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type GetPostResult struct {
	Post Post
}

type SavePostResult struct{}

type DeletePostSaveResult struct{}

type UpdatePostResult struct {
	Post Post
}

type DeletePostResult struct{}

type Post struct {
	ID                string
	CommunityID       string
	AuthorID          string
	Title             string
	Body              string
	BodyExcerpt       string
	Format            string
	ContentRefs       []ContentRef
	Status            string
	IsLocked          bool
	IsPinned          bool
	IsNSFW            bool
	IsSpoiler         bool
	FlairText         string
	Community         CommunitySummary
	Author            UserSummary
	UpvoteCount       int
	DownvoteCount     int
	CommentCount      int
	SaveCount         int
	Score             int
	MyVote            int
	IsSaved           bool
	Preview           PostPreview
	ViewerPermissions ViewerPermissions
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Attachments       []Attachment
	Effects           []PostEffectSummary
}

type ContentRefInput struct {
	Kind  string
	RefID string
}

type ContentRef struct {
	Kind  string
	RefID string
}

type UserSummary struct {
	ID              string
	Username        string
	DisplayName     string
	AvatarURL       string
	Headline        string
	CommunityRole   string
	PlatformRole    string
	IsPlatformStaff bool
	Badges          []string
}

type CommunitySummary struct {
	ID                string
	Slug              string
	Name              string
	Description       string
	AvatarURL         string
	BannerURL         string
	MemberCount       int
	PostCount         int
	ViewerIsFollowing bool
	ViewerRole        string
	ViewerPermissions CommunityViewerPermissions
}

type PostPreview struct {
	Kind  string
	Image *PostPreviewImage
}

type PostPreviewImage struct {
	URL       string
	Width     *int
	Height    *int
	MimeType  string
	AltText   string
	SizeBytes int64
}

type ViewerPermissions struct {
	CanComment  bool
	CanVote     bool
	CanReport   bool
	CanEdit     bool
	CanDelete   bool
	CanModerate bool
}

type CommunityViewerPermissions struct {
	CanPost               bool
	CanManage             bool
	CanModerate           bool
	PlatformOwnerOverride bool
}

type PostMetadata struct {
	Author       UserSummary
	Community    CommunitySummary
	CommentCount int
}

type PostEffectSummary struct {
	ID            string
	EffectID      string
	Name          string
	Emoji         string
	AssetURL      string
	AnimationKey  string
	AppliedByUser UserSummary
	PointsSpent   int
	CreatedAt     time.Time
}

type Attachment struct {
	ID           string
	Kind         string
	URL          string
	ThumbnailURL string
	Width        *int
	Height       *int
	SizeBytes    int64
	MimeType     string
	AltText      string
	Status       string
	CreatedAt    time.Time
}

func NewPostUseCase(posts PostRepository, communities CommunityPolicy, now func() time.Time, votes ...VoteRepository) *PostUseCase {
	if now == nil {
		now = time.Now
	}

	var voteRepo VoteRepository
	if len(votes) > 0 {
		voteRepo = votes[0]
	}
	var metadataRepo PostMetadataRepository
	if repo, ok := posts.(PostMetadataRepository); ok {
		metadataRepo = repo
	}
	var contentRefRepo ContentRefRepository
	if repo, ok := posts.(ContentRefRepository); ok {
		contentRefRepo = repo
	}
	var effectRepo PostEffectRepository
	if repo, ok := posts.(PostEffectRepository); ok {
		effectRepo = repo
	}
	var saveRepo PostSaveRepository
	if repo, ok := posts.(PostSaveRepository); ok {
		saveRepo = repo
	}

	return &PostUseCase{
		posts:             posts,
		communities:       communities,
		votes:             voteRepo,
		saves:             saveRepo,
		metadata:          metadataRepo,
		contentRefs:       contentRefRepo,
		effects:           effectRepo,
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

func (uc *PostUseCase) SetPublicUserFinder(users PublicUserFinder) {
	uc.users = users
}

func (uc *PostUseCase) SetNotificationPublisher(notifications NotificationPublisher) {
	uc.notifications = notifications
}

type XPRecorder interface {
	GrantXP(ctx context.Context, input progressionusecase.GrantXPInput) error
}

func (uc *PostUseCase) SetXPRecorder(progression XPRecorder) {
	uc.progression = progression
}

type PointRecorder interface {
	GrantPoints(ctx context.Context, input effectusecase.GrantPointsInput) error
}

func (uc *PostUseCase) SetPointRecorder(points PointRecorder) {
	uc.points = points
}

func (uc *PostUseCase) SetSettingsReader(settingsReader platformsettings.Reader) {
	uc.settingsReader = settingsReader
}

func (uc *PostUseCase) PublishPost(ctx context.Context, input PublishPostInput) (PublishPostResult, error) {
	if strings.TrimSpace(input.AuthorID.String()) == "" {
		return PublishPostResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if err := uc.ensurePostingEnabled(ctx); err != nil {
		return PublishPostResult{}, err
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
	contentRefs, err := ParseContentRefInputs(input.ContentRefs)
	if err != nil {
		return PublishPostResult{}, err
	}
	if err := ValidateImageContentRefIDs(contentRefs, attachmentIDs); err != nil {
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
	if err := ValidateImageContentRefs(contentRefs, attachments); err != nil {
		return PublishPostResult{}, err
	}
	if err := uc.replacePostContentRefs(ctx, post.ID(), contentRefs, now); err != nil {
		return PublishPostResult{}, err
	}

	metadataViews, err := uc.loadMetadataViews(ctx, []postdomain.Post{*post}, input.AuthorID)
	if err != nil {
		return PublishPostResult{}, err
	}
	if err := uc.notifyMentions(ctx, mention.ExtractUsernames(input.Body), input.AuthorID, "post", post.ID().String()); err != nil {
		return PublishPostResult{}, err
	}
	if err := uc.grantXP(ctx, input.AuthorID, input.AuthorID, progressionusecase.XPSourcePostPublish, post.ID().String()); err != nil {
		return PublishPostResult{}, err
	}
	_ = uc.grantPoints(ctx, input.AuthorID, input.AuthorID, effectusecase.PointSourcePostPublish, post.ID().String())

	return PublishPostResult{
		Post: toPostDTO(*post, postVoteView{}, postSaveView{}, attachments, contentRefs, metadataViews[post.ID()], nil, input.AuthorID),
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
	timeRange, err := normalizePostListTimeRange(input.TimeRange)
	if err != nil {
		return ListCommunityPostsResult{}, err
	}
	createdAfter := postListCreatedAfter(timeRange, uc.now().UTC())

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

	posts, err := uc.posts.ListVisibleByCommunity(ctx, communityID, sort, createdAfter, limit+1, offset)
	if err != nil {
		return ListCommunityPostsResult{}, fmt.Errorf("list community posts: %w", err)
	}
	posts, hasMore := trimPostPage(posts, limit)

	result := ListCommunityPostsResult{
		Posts:      make([]Post, 0, len(posts)),
		Limit:      limit,
		Offset:     offset,
		NextOffset: offset + len(posts),
		HasMore:    hasMore,
	}
	voteViews, err := uc.loadVoteViews(ctx, posts, input.ViewerID)
	if err != nil {
		return ListCommunityPostsResult{}, err
	}
	saveViews, err := uc.loadSaveViews(ctx, posts, input.ViewerID)
	if err != nil {
		return ListCommunityPostsResult{}, err
	}
	attachmentViews, err := uc.loadAttachmentViews(ctx, posts)
	if err != nil {
		return ListCommunityPostsResult{}, err
	}
	contentRefViews, err := uc.loadContentRefViews(ctx, posts)
	if err != nil {
		return ListCommunityPostsResult{}, err
	}
	metadataViews, err := uc.loadMetadataViews(ctx, posts, input.ViewerID)
	if err != nil {
		return ListCommunityPostsResult{}, err
	}
	effectViews, err := uc.loadPostEffectViews(ctx, posts)
	if err != nil {
		return ListCommunityPostsResult{}, err
	}
	for _, post := range posts {
		result.Posts = append(result.Posts, toPostDTO(post, voteViews[post.ID()], saveViews[post.ID()], attachmentViews[post.ID()], contentRefViews[post.ID()], metadataViews[post.ID()], effectViews[post.ID()], input.ViewerID))
	}

	return result, nil
}

func (uc *PostUseCase) ListLatestPosts(ctx context.Context, input ListLatestPostsInput) (ListLatestPostsResult, error) {
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListLatestPostsResult{}, err
	}
	source, err := normalizePostFeedSource(input.Source)
	if err != nil {
		return ListLatestPostsResult{}, err
	}
	defaultSort := PostListSortNew
	if source == PostFeedSourceRecommended {
		defaultSort = PostListSortHot
	}
	sort, err := normalizePostListSortWithDefault(input.Sort, defaultSort)
	if err != nil {
		return ListLatestPostsResult{}, err
	}
	timeRange, err := normalizePostListTimeRange(input.TimeRange)
	if err != nil {
		return ListLatestPostsResult{}, err
	}
	createdAfter := postListCreatedAfter(timeRange, uc.now().UTC())

	var posts []postdomain.Post
	switch source {
	case PostFeedSourceRecommended:
		posts, err = uc.posts.ListRecommendedInPublicCommunities(ctx, input.ViewerID, sort, createdAfter, limit+1, offset)
	case PostFeedSourceFollowing:
		if strings.TrimSpace(input.ViewerID.String()) == "" {
			return ListLatestPostsResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
		}
		posts, err = uc.posts.ListFollowingInPublicCommunities(ctx, input.ViewerID, sort, createdAfter, limit+1, offset)
	default:
		posts, err = uc.posts.ListVisibleInPublicCommunities(ctx, sort, createdAfter, limit+1, offset)
	}
	if err != nil {
		return ListLatestPostsResult{}, fmt.Errorf("list latest posts: %w", err)
	}
	posts, hasMore := trimPostPage(posts, limit)

	voteViews, err := uc.loadVoteViews(ctx, posts, input.ViewerID)
	if err != nil {
		return ListLatestPostsResult{}, err
	}
	saveViews, err := uc.loadSaveViews(ctx, posts, input.ViewerID)
	if err != nil {
		return ListLatestPostsResult{}, err
	}
	attachmentViews, err := uc.loadAttachmentViews(ctx, posts)
	if err != nil {
		return ListLatestPostsResult{}, err
	}
	contentRefViews, err := uc.loadContentRefViews(ctx, posts)
	if err != nil {
		return ListLatestPostsResult{}, err
	}

	result := ListLatestPostsResult{
		Posts:      make([]Post, 0, len(posts)),
		Limit:      limit,
		Offset:     offset,
		NextOffset: offset + len(posts),
		HasMore:    hasMore,
	}
	metadataViews, err := uc.loadMetadataViews(ctx, posts, input.ViewerID)
	if err != nil {
		return ListLatestPostsResult{}, err
	}
	effectViews, err := uc.loadPostEffectViews(ctx, posts)
	if err != nil {
		return ListLatestPostsResult{}, err
	}
	for _, post := range posts {
		result.Posts = append(result.Posts, toPostDTO(post, voteViews[post.ID()], saveViews[post.ID()], attachmentViews[post.ID()], contentRefViews[post.ID()], metadataViews[post.ID()], effectViews[post.ID()], input.ViewerID))
	}

	return result, nil
}

func (uc *PostUseCase) ListUserPosts(ctx context.Context, input ListUserPostsInput) (ListUserPostsResult, error) {
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListUserPostsResult{}, err
	}
	sort, err := normalizePostListSort(input.Sort)
	if err != nil {
		return ListUserPostsResult{}, err
	}
	timeRange, err := normalizePostListTimeRange(input.TimeRange)
	if err != nil {
		return ListUserPostsResult{}, err
	}
	createdAfter := postListCreatedAfter(timeRange, uc.now().UTC())
	author, err := uc.findActivePublicUser(ctx, input.Username)
	if err != nil {
		return ListUserPostsResult{}, err
	}

	posts, err := uc.posts.ListVisibleByAuthorInPublicCommunities(ctx, author.ID(), sort, createdAfter, limit+1, offset)
	if err != nil {
		return ListUserPostsResult{}, fmt.Errorf("list public user posts: %w", err)
	}
	posts, hasMore := trimPostPage(posts, limit)

	voteViews, err := uc.loadVoteViews(ctx, posts, input.ViewerID)
	if err != nil {
		return ListUserPostsResult{}, err
	}
	saveViews, err := uc.loadSaveViews(ctx, posts, input.ViewerID)
	if err != nil {
		return ListUserPostsResult{}, err
	}
	attachmentViews, err := uc.loadAttachmentViews(ctx, posts)
	if err != nil {
		return ListUserPostsResult{}, err
	}
	contentRefViews, err := uc.loadContentRefViews(ctx, posts)
	if err != nil {
		return ListUserPostsResult{}, err
	}
	metadataViews, err := uc.loadMetadataViews(ctx, posts, input.ViewerID)
	if err != nil {
		return ListUserPostsResult{}, err
	}
	effectViews, err := uc.loadPostEffectViews(ctx, posts)
	if err != nil {
		return ListUserPostsResult{}, err
	}

	result := ListUserPostsResult{
		Posts:      make([]Post, 0, len(posts)),
		Limit:      limit,
		Offset:     offset,
		NextOffset: offset + len(posts),
		HasMore:    hasMore,
	}
	for _, post := range posts {
		result.Posts = append(result.Posts, toPostDTO(post, voteViews[post.ID()], saveViews[post.ID()], attachmentViews[post.ID()], contentRefViews[post.ID()], metadataViews[post.ID()], effectViews[post.ID()], input.ViewerID))
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
	saveViews, err := uc.loadSaveViews(ctx, []postdomain.Post{*post}, input.ViewerID)
	if err != nil {
		return GetPostResult{}, err
	}
	attachmentViews, err := uc.loadAttachmentViews(ctx, []postdomain.Post{*post})
	if err != nil {
		return GetPostResult{}, err
	}
	contentRefViews, err := uc.loadContentRefViews(ctx, []postdomain.Post{*post})
	if err != nil {
		return GetPostResult{}, err
	}
	metadataViews, err := uc.loadMetadataViews(ctx, []postdomain.Post{*post}, input.ViewerID)
	if err != nil {
		return GetPostResult{}, err
	}
	effectViews, err := uc.loadPostEffectViews(ctx, []postdomain.Post{*post})
	if err != nil {
		return GetPostResult{}, err
	}

	return GetPostResult{
		Post: toPostDTO(*post, voteViews[post.ID()], saveViews[post.ID()], attachmentViews[post.ID()], contentRefViews[post.ID()], metadataViews[post.ID()], effectViews[post.ID()], input.ViewerID),
	}, nil
}

func (uc *PostUseCase) SavePost(ctx context.Context, input SavePostInput) (SavePostResult, error) {
	if strings.TrimSpace(input.UserID.String()) == "" {
		return SavePostResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.saves == nil {
		return SavePostResult{}, apperr.New(apperr.CodeInternal, "post saves are not configured")
	}

	postID, err := postdomain.NewPostID(input.PostID)
	if err != nil {
		return SavePostResult{}, err
	}
	post, err := uc.posts.FindVisibleByID(ctx, postID)
	if err != nil {
		return SavePostResult{}, fmt.Errorf("find post for save: %w", err)
	}
	if err := uc.saves.SavePost(ctx, postID, input.UserID, uc.now().UTC()); err != nil {
		return SavePostResult{}, fmt.Errorf("save post: %w", err)
	}
	if post.AuthorID() != input.UserID {
		if err := uc.grantXP(ctx, post.AuthorID(), input.UserID, progressionusecase.XPSourcePostSave, post.ID().String()+":"+input.UserID.String()); err != nil {
			return SavePostResult{}, err
		}
		_ = uc.grantPoints(ctx, post.AuthorID(), input.UserID, effectusecase.PointSourcePostSave, post.ID().String()+":"+input.UserID.String())
	}

	return SavePostResult{}, nil
}

func (uc *PostUseCase) DeletePostSave(ctx context.Context, input DeletePostSaveInput) (DeletePostSaveResult, error) {
	if strings.TrimSpace(input.UserID.String()) == "" {
		return DeletePostSaveResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.saves == nil {
		return DeletePostSaveResult{}, apperr.New(apperr.CodeInternal, "post saves are not configured")
	}

	postID, err := postdomain.NewPostID(input.PostID)
	if err != nil {
		return DeletePostSaveResult{}, err
	}
	if _, err := uc.posts.FindVisibleByID(ctx, postID); err != nil {
		return DeletePostSaveResult{}, fmt.Errorf("find post for unsave: %w", err)
	}
	if err := uc.saves.DeletePostSave(ctx, postID, input.UserID); err != nil {
		return DeletePostSaveResult{}, fmt.Errorf("delete post save: %w", err)
	}

	return DeletePostSaveResult{}, nil
}

func (uc *PostUseCase) ListSavedPosts(ctx context.Context, input ListSavedPostsInput) (ListSavedPostsResult, error) {
	if strings.TrimSpace(input.UserID.String()) == "" {
		return ListSavedPostsResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.saves == nil {
		return ListSavedPostsResult{}, apperr.New(apperr.CodeInternal, "post saves are not configured")
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListSavedPostsResult{}, err
	}

	posts, err := uc.saves.ListSavedVisiblePosts(ctx, input.UserID, limit+1, offset)
	if err != nil {
		return ListSavedPostsResult{}, fmt.Errorf("list saved posts: %w", err)
	}
	posts, hasMore := trimPostPage(posts, limit)

	voteViews, err := uc.loadVoteViews(ctx, posts, input.UserID)
	if err != nil {
		return ListSavedPostsResult{}, err
	}
	saveViews, err := uc.loadSaveViews(ctx, posts, input.UserID)
	if err != nil {
		return ListSavedPostsResult{}, err
	}
	attachmentViews, err := uc.loadAttachmentViews(ctx, posts)
	if err != nil {
		return ListSavedPostsResult{}, err
	}
	contentRefViews, err := uc.loadContentRefViews(ctx, posts)
	if err != nil {
		return ListSavedPostsResult{}, err
	}
	metadataViews, err := uc.loadMetadataViews(ctx, posts, input.UserID)
	if err != nil {
		return ListSavedPostsResult{}, err
	}
	effectViews, err := uc.loadPostEffectViews(ctx, posts)
	if err != nil {
		return ListSavedPostsResult{}, err
	}

	result := ListSavedPostsResult{
		Posts:      make([]Post, 0, len(posts)),
		Limit:      limit,
		Offset:     offset,
		NextOffset: offset + len(posts),
		HasMore:    hasMore,
	}
	for _, post := range posts {
		result.Posts = append(result.Posts, toPostDTO(post, voteViews[post.ID()], saveViews[post.ID()], attachmentViews[post.ID()], contentRefViews[post.ID()], metadataViews[post.ID()], effectViews[post.ID()], input.UserID))
	}

	return result, nil
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
	oldBody := post.Body().String()

	title, err := postdomain.NewPostTitle(input.Title)
	if err != nil {
		return UpdatePostResult{}, err
	}
	body, err := postdomain.NewPostBody(input.Body)
	if err != nil {
		return UpdatePostResult{}, err
	}

	attachmentIDs, replaceAttachments, err := parseOptionalAttachmentIDs(input.AttachmentIDs, uc.postImageMaxCount)
	if err != nil {
		return UpdatePostResult{}, err
	}
	contentRefs, replaceContentRefs, err := ParseOptionalContentRefInputs(input.ContentRefs)
	if err != nil {
		return UpdatePostResult{}, err
	}
	if replaceContentRefs && replaceAttachments {
		if err := ValidateImageContentRefIDs(contentRefs, attachmentIDs); err != nil {
			return UpdatePostResult{}, err
		}
	}

	var attachments []mediadomain.Attachment
	attachmentsLoaded := false
	if replaceContentRefs && !replaceAttachments {
		attachmentViews, err := uc.loadAttachmentViews(ctx, []postdomain.Post{*post})
		if err != nil {
			return UpdatePostResult{}, err
		}
		attachments = attachmentViews[post.ID()]
		attachmentsLoaded = true
		if err := ValidateImageContentRefs(contentRefs, attachments); err != nil {
			return UpdatePostResult{}, err
		}
	}

	now := uc.now().UTC()
	if err := post.Edit(title, body, now); err != nil {
		return UpdatePostResult{}, err
	}
	if err := uc.posts.UpdateContent(ctx, *post); err != nil {
		return UpdatePostResult{}, fmt.Errorf("update post content: %w", err)
	}

	if replaceAttachments {
		attachments, err = uc.replacePostAttachments(ctx, post.ID(), input.ActorID, attachmentIDs, now)
		if err != nil {
			return UpdatePostResult{}, err
		}
	}
	voteViews, err := uc.loadVoteViews(ctx, []postdomain.Post{*post}, input.ActorID)
	if err != nil {
		return UpdatePostResult{}, err
	}
	saveViews, err := uc.loadSaveViews(ctx, []postdomain.Post{*post}, input.ActorID)
	if err != nil {
		return UpdatePostResult{}, err
	}
	if !replaceAttachments && !attachmentsLoaded {
		attachmentViews, err := uc.loadAttachmentViews(ctx, []postdomain.Post{*post})
		if err != nil {
			return UpdatePostResult{}, err
		}
		attachments = attachmentViews[post.ID()]
	}
	if replaceContentRefs {
		if err := ValidateImageContentRefs(contentRefs, attachments); err != nil {
			return UpdatePostResult{}, err
		}
		if err := uc.replacePostContentRefs(ctx, post.ID(), contentRefs, now); err != nil {
			return UpdatePostResult{}, err
		}
	} else {
		contentRefViews, err := uc.loadContentRefViews(ctx, []postdomain.Post{*post})
		if err != nil {
			return UpdatePostResult{}, err
		}
		contentRefs = contentRefViews[post.ID()]
	}
	metadataViews, err := uc.loadMetadataViews(ctx, []postdomain.Post{*post}, input.ActorID)
	if err != nil {
		return UpdatePostResult{}, err
	}
	effectViews, err := uc.loadPostEffectViews(ctx, []postdomain.Post{*post})
	if err != nil {
		return UpdatePostResult{}, err
	}
	if err := uc.notifyMentions(ctx, mention.AddedUsernames(oldBody, input.Body), input.ActorID, "post", post.ID().String()); err != nil {
		return UpdatePostResult{}, err
	}

	return UpdatePostResult{
		Post: toPostDTO(*post, voteViews[post.ID()], saveViews[post.ID()], attachments, contentRefs, metadataViews[post.ID()], effectViews[post.ID()], input.ActorID),
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

func (uc *PostUseCase) ensurePostingEnabled(ctx context.Context) error {
	if uc.settingsReader == nil {
		return nil
	}
	enabled, err := uc.settingsReader.IsEnabled(ctx, platformsettings.PostingEnabled)
	if err != nil {
		return fmt.Errorf("read posting setting: %w", err)
	}
	if !enabled {
		return apperr.New(apperr.CodeForbidden, "posting is disabled")
	}
	return nil
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

func trimPostPage(posts []postdomain.Post, limit int) ([]postdomain.Post, bool) {
	if len(posts) <= limit {
		return posts, false
	}
	return posts[:limit], true
}

func normalizePostListSort(raw string) (PostListSort, error) {
	return normalizePostListSortWithDefault(raw, PostListSortNew)
}

func normalizePostListSortWithDefault(raw string, defaultSort PostListSort) (PostListSort, error) {
	sort := PostListSort(strings.ToLower(strings.TrimSpace(raw)))
	if sort == "" {
		return defaultSort, nil
	}
	switch sort {
	case PostListSortBest, PostListSortHot, PostListSortNew, PostListSortTop, PostListSortRising:
		return sort, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "post list sort is invalid")
	}
}

func normalizePostFeedSource(raw string) (PostFeedSource, error) {
	source := PostFeedSource(strings.ToLower(strings.TrimSpace(raw)))
	if source == "" {
		return PostFeedSourceAll, nil
	}
	switch source {
	case PostFeedSourceAll, PostFeedSourceRecommended, PostFeedSourceFollowing:
		return source, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "post feed source is invalid")
	}
}

func normalizePostListTimeRange(raw string) (PostListTimeRange, error) {
	timeRange := PostListTimeRange(strings.ToLower(strings.TrimSpace(raw)))
	if timeRange == "" {
		return PostListTimeRangeAll, nil
	}
	switch timeRange {
	case PostListTimeRangeAll, PostListTimeRangeHour, PostListTimeRangeDay, PostListTimeRangeWeek, PostListTimeRangeMonth, PostListTimeRangeYear:
		return timeRange, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "post list time range is invalid")
	}
}

func postListCreatedAfter(timeRange PostListTimeRange, now time.Time) *time.Time {
	var createdAfter time.Time
	switch timeRange {
	case PostListTimeRangeHour:
		createdAfter = now.Add(-time.Hour)
	case PostListTimeRangeDay:
		createdAfter = now.Add(-24 * time.Hour)
	case PostListTimeRangeWeek:
		createdAfter = now.Add(-7 * 24 * time.Hour)
	case PostListTimeRangeMonth:
		createdAfter = now.AddDate(0, -1, 0)
	case PostListTimeRangeYear:
		createdAfter = now.AddDate(-1, 0, 0)
	default:
		return nil
	}
	return &createdAfter
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

func parseOptionalAttachmentIDs(rawIDs *[]string, maxCount int) ([]mediadomain.AttachmentID, bool, error) {
	if rawIDs == nil {
		return nil, false, nil
	}
	attachmentIDs, err := parseAttachmentIDs(*rawIDs, maxCount)
	if err != nil {
		return nil, true, err
	}
	return attachmentIDs, true, nil
}

func ParseContentRefInputs(rawRefs []ContentRefInput) ([]ContentRef, error) {
	if len(rawRefs) == 0 {
		return []ContentRef{}, nil
	}
	if len(rawRefs) > MaxContentRefCount {
		return nil, apperr.New(apperr.CodeInvalidArgument, "content ref count is invalid")
	}

	refs := make([]ContentRef, 0, len(rawRefs))
	seen := make(map[string]bool, len(rawRefs))
	for _, rawRef := range rawRefs {
		kind := strings.ToLower(strings.TrimSpace(rawRef.Kind))
		switch kind {
		case ContentRefKindImage, ContentRefKindLink, ContentRefKindEmbed:
		default:
			return nil, apperr.New(apperr.CodeInvalidArgument, "content ref kind is invalid")
		}

		refID := strings.TrimSpace(rawRef.RefID)
		if refID == "" || len([]rune(refID)) > MaxContentRefIDLength {
			return nil, apperr.New(apperr.CodeInvalidArgument, "content ref id is invalid")
		}
		if kind == ContentRefKindEmbed {
			if _, err := uuid.Parse(refID); err != nil {
				return nil, apperr.New(apperr.CodeInvalidArgument, "embed content ref id is invalid")
			}
		}
		key := kind + "\x00" + refID
		if seen[key] {
			return nil, apperr.New(apperr.CodeInvalidArgument, "content ref is duplicated")
		}
		seen[key] = true
		refs = append(refs, ContentRef{
			Kind:  kind,
			RefID: refID,
		})
	}
	return refs, nil
}

func ParseOptionalContentRefInputs(rawRefs *[]ContentRefInput) ([]ContentRef, bool, error) {
	if rawRefs == nil {
		return nil, false, nil
	}
	refs, err := ParseContentRefInputs(*rawRefs)
	if err != nil {
		return nil, true, err
	}
	return refs, true, nil
}

type postVoteView struct {
	upvoteCount   int
	downvoteCount int
	myVote        int
}

type postSaveView struct {
	saveCount int
	isSaved   bool
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

func (uc *PostUseCase) loadSaveViews(ctx context.Context, posts []postdomain.Post, viewerID userdomain.UserID) (map[postdomain.PostID]postSaveView, error) {
	views := make(map[postdomain.PostID]postSaveView, len(posts))
	if len(posts) == 0 || uc.saves == nil {
		return views, nil
	}

	postIDs := make([]postdomain.PostID, 0, len(posts))
	for _, post := range posts {
		postIDs = append(postIDs, post.ID())
	}

	saveCounts, err := uc.saves.SummarizeSavesByPostIDs(ctx, postIDs)
	if err != nil {
		return nil, fmt.Errorf("summarize post saves: %w", err)
	}

	savedByViewer := map[postdomain.PostID]bool{}
	if strings.TrimSpace(viewerID.String()) != "" {
		savedByViewer, err = uc.saves.FindSavedPostIDsByUser(ctx, postIDs, viewerID)
		if err != nil {
			return nil, fmt.Errorf("find saved posts by viewer: %w", err)
		}
	}

	for _, postID := range postIDs {
		views[postID] = postSaveView{
			saveCount: saveCounts[postID],
			isSaved:   savedByViewer[postID],
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

func (uc *PostUseCase) replacePostAttachments(ctx context.Context, postID postdomain.PostID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, now time.Time) ([]mediadomain.Attachment, error) {
	if uc.attachments == nil {
		return nil, apperr.New(apperr.CodeInvalidArgument, "post image attachments are not supported")
	}
	attachments, err := uc.attachments.ReplaceReadyImagesForPost(ctx, postID, uploaderID, attachmentIDs, uc.postImageMaxCount, now)
	if err != nil {
		return nil, fmt.Errorf("replace post image attachments: %w", err)
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

func (uc *PostUseCase) replacePostContentRefs(ctx context.Context, postID postdomain.PostID, refs []ContentRef, now time.Time) error {
	if uc.contentRefs == nil {
		if len(refs) == 0 {
			return nil
		}
		return apperr.New(apperr.CodeInvalidArgument, "post content refs are not supported")
	}
	if err := uc.contentRefs.ReplacePostContentRefs(ctx, postID, refs, now); err != nil {
		return fmt.Errorf("replace post content refs: %w", err)
	}
	return nil
}

func (uc *PostUseCase) loadContentRefViews(ctx context.Context, posts []postdomain.Post) (map[postdomain.PostID][]ContentRef, error) {
	views := make(map[postdomain.PostID][]ContentRef, len(posts))
	if len(posts) == 0 || uc.contentRefs == nil {
		return views, nil
	}
	postIDs := make([]postdomain.PostID, 0, len(posts))
	for _, post := range posts {
		postIDs = append(postIDs, post.ID())
	}
	views, err := uc.contentRefs.ListPostContentRefsByPostIDs(ctx, postIDs)
	if err != nil {
		return nil, fmt.Errorf("list post content refs: %w", err)
	}
	return views, nil
}

func (uc *PostUseCase) loadPostEffectViews(ctx context.Context, posts []postdomain.Post) (map[postdomain.PostID][]PostEffectSummary, error) {
	views := make(map[postdomain.PostID][]PostEffectSummary, len(posts))
	if len(posts) == 0 || uc.effects == nil {
		return views, nil
	}
	postIDs := make([]postdomain.PostID, 0, len(posts))
	for _, post := range posts {
		postIDs = append(postIDs, post.ID())
	}
	views, err := uc.effects.ListPostEffectsByPostIDs(ctx, postIDs)
	if err != nil {
		return nil, fmt.Errorf("list post effects: %w", err)
	}
	return views, nil
}

func ValidateImageContentRefs(refs []ContentRef, attachments []mediadomain.Attachment) error {
	imageAttachmentIDs := make(map[string]bool, len(attachments))
	for _, attachment := range attachments {
		if attachment.Kind().String() == ContentRefKindImage {
			imageAttachmentIDs[attachment.ID().String()] = true
		}
	}
	for _, ref := range refs {
		if ref.Kind == ContentRefKindImage && !imageAttachmentIDs[ref.RefID] {
			return apperr.New(apperr.CodeInvalidArgument, "image content ref must reference a bound image attachment")
		}
	}
	return nil
}

func ValidateImageContentRefIDs(refs []ContentRef, attachmentIDs []mediadomain.AttachmentID) error {
	imageAttachmentIDs := make(map[string]bool, len(attachmentIDs))
	for _, attachmentID := range attachmentIDs {
		imageAttachmentIDs[attachmentID.String()] = true
	}
	for _, ref := range refs {
		if ref.Kind == ContentRefKindImage && !imageAttachmentIDs[ref.RefID] {
			return apperr.New(apperr.CodeInvalidArgument, "image content ref must reference a bound image attachment")
		}
	}
	return nil
}

func (uc *PostUseCase) loadMetadataViews(ctx context.Context, posts []postdomain.Post, viewerID userdomain.UserID) (map[postdomain.PostID]PostMetadata, error) {
	views := make(map[postdomain.PostID]PostMetadata, len(posts))
	if len(posts) == 0 {
		return views, nil
	}
	for _, post := range posts {
		views[post.ID()] = fallbackPostMetadata(post)
	}
	if uc.metadata == nil {
		return views, nil
	}

	postIDs := make([]postdomain.PostID, 0, len(posts))
	for _, post := range posts {
		postIDs = append(postIDs, post.ID())
	}
	loaded, err := uc.metadata.LoadMetadataByPostIDs(ctx, postIDs, viewerID)
	if err != nil {
		return nil, fmt.Errorf("load post metadata: %w", err)
	}
	for postID, metadata := range loaded {
		views[postID] = normalizePostMetadata(metadata)
	}
	return views, nil
}

func (uc *PostUseCase) findActivePublicUser(ctx context.Context, rawUsername string) (*userdomain.User, error) {
	if uc.users == nil {
		return nil, apperr.New(apperr.CodeInternal, "public user finder is not configured")
	}
	username, err := userdomain.NewUsername(rawUsername)
	if err != nil {
		return nil, err
	}
	user, err := uc.users.FindByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("find public user by username: %w", err)
	}
	if !user.CanLogin() {
		return nil, apperr.New(apperr.CodeNotFound, "user not found")
	}
	return user, nil
}

func (uc *PostUseCase) notifyMentions(ctx context.Context, usernames []userdomain.Username, actorID userdomain.UserID, sourceType string, sourceID string) error {
	if len(usernames) == 0 || uc.notifications == nil {
		return nil
	}
	if uc.users == nil {
		return apperr.New(apperr.CodeInternal, "public user finder is not configured")
	}
	for _, username := range usernames {
		user, err := uc.users.FindByUsername(ctx, username)
		if err != nil {
			if apperr.IsCode(err, apperr.CodeNotFound) {
				continue
			}
			return fmt.Errorf("find mentioned user by username: %w", err)
		}
		if !user.CanLogin() {
			continue
		}
		if err := uc.notifications.NotifyMentioned(ctx, user.ID(), actorID, sourceType, sourceID); err != nil {
			return err
		}
	}
	return nil
}

func (uc *PostUseCase) grantXP(ctx context.Context, userID userdomain.UserID, actorID userdomain.UserID, sourceType string, sourceID string) error {
	if uc.progression == nil || strings.TrimSpace(userID.String()) == "" {
		return nil
	}
	return uc.progression.GrantXP(ctx, progressionusecase.GrantXPInput{
		UserID:     userID,
		ActorID:    actorID,
		SourceType: sourceType,
		SourceID:   sourceID,
	})
}

func (uc *PostUseCase) grantPoints(ctx context.Context, userID userdomain.UserID, actorID userdomain.UserID, sourceType string, sourceID string) error {
	if uc.points == nil || strings.TrimSpace(userID.String()) == "" {
		return nil
	}
	return uc.points.GrantPoints(ctx, effectusecase.GrantPointsInput{
		UserID:     userID,
		ActorID:    actorID,
		SourceType: sourceType,
		SourceID:   sourceID,
	})
}

func toPostDTO(post postdomain.Post, voteView postVoteView, saveView postSaveView, attachments []mediadomain.Attachment, contentRefs []ContentRef, metadata PostMetadata, effects []PostEffectSummary, viewerID userdomain.UserID) Post {
	score := voteView.upvoteCount - voteView.downvoteCount
	metadata = normalizePostMetadata(metadata)
	attachmentDTOs := toAttachmentDTOs(attachments)
	return Post{
		ID:                post.ID().String(),
		CommunityID:       post.CommunityID().String(),
		AuthorID:          post.AuthorID().String(),
		Title:             post.Title().String(),
		Body:              post.Body().String(),
		BodyExcerpt:       makeExcerpt(post.Body().String(), DefaultPostExcerptMax),
		Format:            PostFormat,
		ContentRefs:       CloneContentRefs(contentRefs),
		Status:            post.Status().String(),
		IsLocked:          post.IsLocked(),
		IsPinned:          post.IsPinned(),
		IsNSFW:            post.IsNSFW(),
		IsSpoiler:         post.IsSpoiler(),
		FlairText:         post.FlairText(),
		Community:         metadata.Community,
		Author:            metadata.Author,
		UpvoteCount:       voteView.upvoteCount,
		DownvoteCount:     voteView.downvoteCount,
		CommentCount:      metadata.CommentCount,
		SaveCount:         saveView.saveCount,
		Score:             score,
		MyVote:            voteView.myVote,
		IsSaved:           saveView.isSaved,
		Preview:           buildPostPreview(attachmentDTOs),
		ViewerPermissions: postViewerPermissions(post, viewerID),
		CreatedAt:         post.CreatedAt(),
		UpdatedAt:         post.UpdatedAt(),
		Attachments:       attachmentDTOs,
		Effects:           ClonePostEffects(effects),
	}
}

func CloneContentRefs(refs []ContentRef) []ContentRef {
	if len(refs) == 0 {
		return []ContentRef{}
	}
	result := make([]ContentRef, len(refs))
	copy(result, refs)
	return result
}

func ClonePostEffects(effects []PostEffectSummary) []PostEffectSummary {
	if len(effects) == 0 {
		return []PostEffectSummary{}
	}
	result := make([]PostEffectSummary, len(effects))
	copy(result, effects)
	for index := range result {
		result[index].AppliedByUser.Badges = cloneStringSlice(result[index].AppliedByUser.Badges)
	}
	return result
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := make([]string, len(values))
	copy(result, values)
	return result
}

func fallbackPostMetadata(post postdomain.Post) PostMetadata {
	return PostMetadata{
		Author: UserSummary{
			ID:     post.AuthorID().String(),
			Badges: []string{},
		},
		Community: CommunitySummary{
			ID:                post.CommunityID().String(),
			ViewerPermissions: CommunityViewerPermissions{},
		},
	}
}

func normalizePostMetadata(metadata PostMetadata) PostMetadata {
	if metadata.Author.Badges == nil {
		metadata.Author.Badges = []string{}
	}
	if metadata.Author.DisplayName == "" {
		metadata.Author.DisplayName = metadata.Author.Username
	}
	return metadata
}

func makeExcerpt(body string, maxRunes int) string {
	body = strings.TrimSpace(body)
	if maxRunes <= 0 {
		return body
	}
	runes := []rune(body)
	if len(runes) <= maxRunes {
		return body
	}
	return strings.TrimSpace(string(runes[:maxRunes]))
}

func buildPostPreview(attachments []Attachment) PostPreview {
	for _, attachment := range attachments {
		if attachment.Kind == "image" {
			return PostPreview{
				Kind: "image",
				Image: &PostPreviewImage{
					URL:       attachment.URL,
					Width:     attachment.Width,
					Height:    attachment.Height,
					MimeType:  attachment.MimeType,
					AltText:   attachment.AltText,
					SizeBytes: attachment.SizeBytes,
				},
			}
		}
	}
	return PostPreview{Kind: "text"}
}

func postViewerPermissions(post postdomain.Post, viewerID userdomain.UserID) ViewerPermissions {
	if strings.TrimSpace(viewerID.String()) == "" {
		return ViewerPermissions{}
	}
	isAuthor := post.AuthorID() == viewerID
	return ViewerPermissions{
		CanComment: !post.IsLocked(),
		CanVote:    true,
		CanReport:  true,
		CanEdit:    isAuthor,
		CanDelete:  isAuthor,
	}
}

func toAttachmentDTOs(attachments []mediadomain.Attachment) []Attachment {
	result := make([]Attachment, 0, len(attachments))
	for _, attachment := range attachments {
		result = append(result, Attachment{
			ID:           attachment.ID().String(),
			Kind:         attachment.Kind().String(),
			URL:          attachment.PublicURL(),
			ThumbnailURL: attachment.ThumbnailURL(),
			Width:        attachment.Width(),
			Height:       attachment.Height(),
			SizeBytes:    attachment.SizeBytes(),
			MimeType:     attachment.MimeType(),
			AltText:      attachment.AltText(),
			Status:       attachment.Status().String(),
			CreatedAt:    attachment.CreatedAt(),
		})
	}
	return result
}

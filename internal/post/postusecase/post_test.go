package postusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communityusecase"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
)

func TestPublishPostCreatesVisiblePost(t *testing.T) {
	now := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	authorID := userdomain.NewGeneratedUserID()
	communityID := communitydomain.NewGeneratedCommunityID()
	posts := &fakePostRepository{
		createFunc: func(ctx context.Context, post postdomain.Post) error {
			if post.AuthorID() != authorID {
				t.Fatalf("expected author %q, got %q", authorID.String(), post.AuthorID().String())
			}
			if post.CommunityID() != communityID {
				t.Fatalf("expected community %q, got %q", communityID.String(), post.CommunityID().String())
			}
			if post.Status() != postdomain.PostStatusVisible {
				t.Fatalf("expected visible status, got %q", post.Status().String())
			}
			return nil
		},
	}
	communities := &fakeCommunityPolicy{
		getResult: communityusecase.GetCommunityResult{Community: newCommunityDTO(communityID, "campus")},
		canPostResult: communityusecase.CanPostInCommunityResult{
			Community: newCommunityDTO(communityID, "campus"),
		},
	}
	uc := NewPostUseCase(posts, communities, func() time.Time { return now })

	result, err := uc.PublishPost(context.Background(), PublishPostInput{
		CommunitySlug: "campus",
		AuthorID:      authorID,
		Title:         "Hello",
		Body:          "Post body",
	})
	if err != nil {
		t.Fatalf("PublishPost returned error: %v", err)
	}

	if result.Post.Status != postdomain.PostStatusVisible.String() {
		t.Fatalf("expected visible result, got %q", result.Post.Status)
	}
	if !communities.getCalled || !communities.canPostCalled {
		t.Fatal("expected community lookup and permission check")
	}
}

func TestPublishPostRejectsInvalidInput(t *testing.T) {
	uc := NewPostUseCase(&fakePostRepository{}, &fakeCommunityPolicy{}, time.Now)

	_, err := uc.PublishPost(context.Background(), PublishPostInput{
		CommunitySlug: "campus",
		AuthorID:      "",
		Title:         "Hello",
		Body:          "Body",
	})
	if !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated for missing author, got %v", err)
	}

	_, err = uc.PublishPost(context.Background(), PublishPostInput{
		CommunitySlug: "campus",
		AuthorID:      userdomain.NewGeneratedUserID(),
		Title:         " ",
		Body:          "Body",
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for blank title, got %v", err)
	}
}

func TestPublishPostPropagatesCommunityPermissionError(t *testing.T) {
	communityID := communitydomain.NewGeneratedCommunityID()
	communities := &fakeCommunityPolicy{
		getResult:  communityusecase.GetCommunityResult{Community: newCommunityDTO(communityID, "campus")},
		canPostErr: apperr.New(apperr.CodeForbidden, "can't post in community"),
	}
	uc := NewPostUseCase(&fakePostRepository{}, communities, time.Now)

	_, err := uc.PublishPost(context.Background(), PublishPostInput{
		CommunitySlug: "campus",
		AuthorID:      userdomain.NewGeneratedUserID(),
		Title:         "Hello",
		Body:          "Body",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden from community permission, got %v", err)
	}
}

func TestListCommunityPostsNormalizesPagination(t *testing.T) {
	communityID := communitydomain.NewGeneratedCommunityID()
	viewerID := userdomain.NewGeneratedUserID()
	authorID := userdomain.NewGeneratedUserID()
	post := mustPost(t, communityID, authorID, "Hello", time.Now().UTC())
	var gotLimit int
	var gotOffset int
	posts := &fakePostRepository{
		listVisibleByCommunityFunc: func(ctx context.Context, id communitydomain.CommunityID, sort PostListSort, limit int, offset int) ([]postdomain.Post, error) {
			if sort != PostListSortNew {
				t.Fatalf("expected default sort %q, got %q", PostListSortNew, sort)
			}
			gotLimit = limit
			gotOffset = offset
			return []postdomain.Post{*post}, nil
		},
	}
	communities := &fakeCommunityPolicy{
		getResult: communityusecase.GetCommunityResult{Community: newCommunityDTO(communityID, "campus")},
	}
	votes := &fakeVoteRepository{
		summaries: map[postdomain.PostID]votedomain.PostVoteSummary{
			post.ID(): {
				PostID:        post.ID(),
				UpvoteCount:   2,
				DownvoteCount: 1,
			},
		},
		myVotes: map[postdomain.PostID]votedomain.VoteValue{
			post.ID(): votedomain.VoteValueUp,
		},
	}
	uc := NewPostUseCase(posts, communities, time.Now, votes)

	result, err := uc.ListCommunityPosts(context.Background(), ListCommunityPostsInput{
		CommunitySlug: "campus",
		ViewerID:      viewerID,
		Limit:         100,
		Offset:        5,
	})
	if err != nil {
		t.Fatalf("ListCommunityPosts returned error: %v", err)
	}
	if gotLimit != MaxPostListLimit || result.Limit != MaxPostListLimit {
		t.Fatalf("expected clamped limit %d, got repo=%d result=%d", MaxPostListLimit, gotLimit, result.Limit)
	}
	if gotOffset != 5 || result.Offset != 5 {
		t.Fatalf("expected offset 5, got repo=%d result=%d", gotOffset, result.Offset)
	}
	if len(result.Posts) != 1 {
		t.Fatalf("expected one post, got %d", len(result.Posts))
	}
	if result.Posts[0].UpvoteCount != 2 || result.Posts[0].DownvoteCount != 1 || result.Posts[0].Score != 1 || result.Posts[0].MyVote != 1 {
		t.Fatalf("unexpected vote view: %#v", result.Posts[0])
	}
	if !votes.summarizeCalled || !votes.findByUserCalled {
		t.Fatal("expected vote summary and viewer vote lookups")
	}
}

func TestListCommunityPostsRejectsInvalidPagination(t *testing.T) {
	uc := NewPostUseCase(&fakePostRepository{}, &fakeCommunityPolicy{}, time.Now)

	_, err := uc.ListCommunityPosts(context.Background(), ListCommunityPostsInput{
		CommunitySlug: "campus",
		Limit:         -1,
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for negative limit, got %v", err)
	}
}

func TestListCommunityPostsPassesHotSort(t *testing.T) {
	communityID := communitydomain.NewGeneratedCommunityID()
	var gotSort PostListSort
	posts := &fakePostRepository{
		listVisibleByCommunityFunc: func(ctx context.Context, id communitydomain.CommunityID, sort PostListSort, limit int, offset int) ([]postdomain.Post, error) {
			gotSort = sort
			return nil, nil
		},
	}
	communities := &fakeCommunityPolicy{
		getResult: communityusecase.GetCommunityResult{Community: newCommunityDTO(communityID, "campus")},
	}
	uc := NewPostUseCase(posts, communities, time.Now)

	_, err := uc.ListCommunityPosts(context.Background(), ListCommunityPostsInput{
		CommunitySlug: "campus",
		Sort:          "HoT",
	})
	if err != nil {
		t.Fatalf("ListCommunityPosts returned error: %v", err)
	}
	if gotSort != PostListSortHot {
		t.Fatalf("expected hot sort, got %q", gotSort)
	}
}

func TestListLatestPostsReturnsVoteView(t *testing.T) {
	viewerID := userdomain.NewGeneratedUserID()
	communityID := communitydomain.NewGeneratedCommunityID()
	post := mustPost(t, communityID, userdomain.NewGeneratedUserID(), "Latest", time.Now().UTC())
	var gotLimit int
	var gotOffset int
	posts := &fakePostRepository{
		listVisibleInPublicCommunitiesFunc: func(ctx context.Context, sort PostListSort, limit int, offset int) ([]postdomain.Post, error) {
			if sort != PostListSortNew {
				t.Fatalf("expected default sort %q, got %q", PostListSortNew, sort)
			}
			gotLimit = limit
			gotOffset = offset
			return []postdomain.Post{*post}, nil
		},
	}
	votes := &fakeVoteRepository{
		summaries: map[postdomain.PostID]votedomain.PostVoteSummary{
			post.ID(): {
				PostID:        post.ID(),
				UpvoteCount:   4,
				DownvoteCount: 2,
			},
		},
		myVotes: map[postdomain.PostID]votedomain.VoteValue{
			post.ID(): votedomain.VoteValueUp,
		},
	}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, time.Now, votes)

	result, err := uc.ListLatestPosts(context.Background(), ListLatestPostsInput{
		ViewerID: viewerID,
		Limit:    100,
		Offset:   3,
	})
	if err != nil {
		t.Fatalf("ListLatestPosts returned error: %v", err)
	}
	if gotLimit != MaxPostListLimit || result.Limit != MaxPostListLimit {
		t.Fatalf("expected clamped limit %d, got repo=%d result=%d", MaxPostListLimit, gotLimit, result.Limit)
	}
	if gotOffset != 3 || result.Offset != 3 {
		t.Fatalf("expected offset 3, got repo=%d result=%d", gotOffset, result.Offset)
	}
	if len(result.Posts) != 1 {
		t.Fatalf("expected one post, got %d", len(result.Posts))
	}
	if result.Posts[0].UpvoteCount != 4 || result.Posts[0].DownvoteCount != 2 || result.Posts[0].Score != 2 || result.Posts[0].MyVote != 1 {
		t.Fatalf("unexpected vote view: %#v", result.Posts[0])
	}
}

func TestListLatestPostsPassesHotSort(t *testing.T) {
	var gotSort PostListSort
	posts := &fakePostRepository{
		listVisibleInPublicCommunitiesFunc: func(ctx context.Context, sort PostListSort, limit int, offset int) ([]postdomain.Post, error) {
			gotSort = sort
			return nil, nil
		},
	}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, time.Now)

	_, err := uc.ListLatestPosts(context.Background(), ListLatestPostsInput{Sort: "hot"})
	if err != nil {
		t.Fatalf("ListLatestPosts returned error: %v", err)
	}
	if gotSort != PostListSortHot {
		t.Fatalf("expected hot sort, got %q", gotSort)
	}
}

func TestListLatestPostsRejectsInvalidSort(t *testing.T) {
	uc := NewPostUseCase(&fakePostRepository{}, &fakeCommunityPolicy{}, time.Now)

	_, err := uc.ListLatestPosts(context.Background(), ListLatestPostsInput{Sort: "popular"})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid sort, got %v", err)
	}
}

func TestGetPostReturnsVisiblePost(t *testing.T) {
	post := mustPost(t, communitydomain.NewGeneratedCommunityID(), userdomain.NewGeneratedUserID(), "Hello", time.Now().UTC())
	viewerID := userdomain.NewGeneratedUserID()
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			if id != post.ID() {
				t.Fatalf("expected post id %q, got %q", post.ID().String(), id.String())
			}
			return post, nil
		},
	}
	votes := &fakeVoteRepository{
		summaries: map[postdomain.PostID]votedomain.PostVoteSummary{
			post.ID(): {
				PostID:        post.ID(),
				UpvoteCount:   3,
				DownvoteCount: 1,
			},
		},
		myVotes: map[postdomain.PostID]votedomain.VoteValue{
			post.ID(): votedomain.VoteValueDown,
		},
	}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, time.Now, votes)

	result, err := uc.GetPost(context.Background(), GetPostInput{PostID: post.ID().String(), ViewerID: viewerID})
	if err != nil {
		t.Fatalf("GetPost returned error: %v", err)
	}
	if result.Post.ID != post.ID().String() {
		t.Fatalf("expected post id %q, got %q", post.ID().String(), result.Post.ID)
	}
	if result.Post.UpvoteCount != 3 || result.Post.DownvoteCount != 1 || result.Post.Score != 2 || result.Post.MyVote != -1 {
		t.Fatalf("unexpected vote view: %#v", result.Post)
	}
}

func TestGetPostDefaultsEmptyVoteView(t *testing.T) {
	post := mustPost(t, communitydomain.NewGeneratedCommunityID(), userdomain.NewGeneratedUserID(), "Hello", time.Now().UTC())
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	votes := &fakeVoteRepository{}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, time.Now, votes)

	result, err := uc.GetPost(context.Background(), GetPostInput{
		PostID:   post.ID().String(),
		ViewerID: userdomain.NewGeneratedUserID(),
	})
	if err != nil {
		t.Fatalf("GetPost returned error: %v", err)
	}
	if result.Post.UpvoteCount != 0 || result.Post.DownvoteCount != 0 || result.Post.Score != 0 || result.Post.MyVote != 0 {
		t.Fatalf("expected zero vote view, got %#v", result.Post)
	}
}

type fakePostRepository struct {
	createFunc                         func(ctx context.Context, post postdomain.Post) error
	findVisibleByIDFunc                func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error)
	listVisibleByCommunityFunc         func(ctx context.Context, communityID communitydomain.CommunityID, sort PostListSort, limit int, offset int) ([]postdomain.Post, error)
	listVisibleInPublicCommunitiesFunc func(ctx context.Context, sort PostListSort, limit int, offset int) ([]postdomain.Post, error)
}

func (f *fakePostRepository) Create(ctx context.Context, post postdomain.Post) error {
	if f.createFunc != nil {
		return f.createFunc(ctx, post)
	}
	return nil
}

func (f *fakePostRepository) FindVisibleByID(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
	if f.findVisibleByIDFunc != nil {
		return f.findVisibleByIDFunc(ctx, id)
	}
	return nil, apperr.New(apperr.CodeNotFound, "post not found")
}

func (f *fakePostRepository) ListVisibleByCommunity(ctx context.Context, communityID communitydomain.CommunityID, sort PostListSort, limit int, offset int) ([]postdomain.Post, error) {
	if f.listVisibleByCommunityFunc != nil {
		return f.listVisibleByCommunityFunc(ctx, communityID, sort, limit, offset)
	}
	return nil, nil
}

func (f *fakePostRepository) ListVisibleInPublicCommunities(ctx context.Context, sort PostListSort, limit int, offset int) ([]postdomain.Post, error) {
	if f.listVisibleInPublicCommunitiesFunc != nil {
		return f.listVisibleInPublicCommunitiesFunc(ctx, sort, limit, offset)
	}
	return nil, nil
}

type fakeVoteRepository struct {
	summarizeCalled  bool
	findByUserCalled bool
	summaries        map[postdomain.PostID]votedomain.PostVoteSummary
	myVotes          map[postdomain.PostID]votedomain.VoteValue
	summarizeErr     error
	findByUserErr    error
}

func (f *fakeVoteRepository) SummarizeByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID]votedomain.PostVoteSummary, error) {
	f.summarizeCalled = true
	if f.summarizeErr != nil {
		return nil, f.summarizeErr
	}
	if f.summaries == nil {
		return map[postdomain.PostID]votedomain.PostVoteSummary{}, nil
	}
	return f.summaries, nil
}

func (f *fakeVoteRepository) FindByPostIDsAndUser(ctx context.Context, postIDs []postdomain.PostID, userID userdomain.UserID) (map[postdomain.PostID]votedomain.VoteValue, error) {
	f.findByUserCalled = true
	if f.findByUserErr != nil {
		return nil, f.findByUserErr
	}
	if f.myVotes == nil {
		return map[postdomain.PostID]votedomain.VoteValue{}, nil
	}
	return f.myVotes, nil
}

type fakeCommunityPolicy struct {
	getCalled     bool
	canPostCalled bool
	getResult     communityusecase.GetCommunityResult
	canPostResult communityusecase.CanPostInCommunityResult
	getErr        error
	canPostErr    error
}

func (f *fakeCommunityPolicy) GetCommunityBySlug(ctx context.Context, input communityusecase.GetCommunityInput) (communityusecase.GetCommunityResult, error) {
	f.getCalled = true
	return f.getResult, f.getErr
}

func (f *fakeCommunityPolicy) CanPostInCommunity(ctx context.Context, input communityusecase.CanPostInCommunityInput) (communityusecase.CanPostInCommunityResult, error) {
	f.canPostCalled = true
	return f.canPostResult, f.canPostErr
}

func newCommunityDTO(id communitydomain.CommunityID, slug string) communityusecase.Community {
	now := time.Now().UTC()
	return communityusecase.Community{
		ID:         id.String(),
		Slug:       slug,
		Name:       slug,
		Status:     communitydomain.CommunityStatusActive.String(),
		Visibility: communitydomain.CommunityVisibilityPublic.String(),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func mustPost(t *testing.T, communityID communitydomain.CommunityID, authorID userdomain.UserID, title string, now time.Time) *postdomain.Post {
	t.Helper()

	postTitle, err := postdomain.NewPostTitle(title)
	if err != nil {
		t.Fatalf("NewPostTitle returned error: %v", err)
	}
	body, err := postdomain.NewPostBody("Body for " + title)
	if err != nil {
		t.Fatalf("NewPostBody returned error: %v", err)
	}
	post, err := postdomain.NewPost(postdomain.NewGeneratedPostID(), communityID, authorID, postTitle, body, now)
	if err != nil {
		t.Fatalf("NewPost returned error: %v", err)
	}
	return post
}

func hasAppCode(err error, code apperr.Code) bool {
	if err == nil {
		return false
	}

	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Code() == code
}

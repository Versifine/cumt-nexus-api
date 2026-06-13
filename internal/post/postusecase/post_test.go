package postusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communityusecase"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
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
	if result.Post.Format != PostFormat {
		t.Fatalf("expected format %q, got %q", PostFormat, result.Post.Format)
	}
	if !communities.getCalled || !communities.canPostCalled {
		t.Fatal("expected community lookup and permission check")
	}
}

func TestPublishPostBindsImageAttachments(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	authorID := userdomain.NewGeneratedUserID()
	communityID := communitydomain.NewGeneratedCommunityID()
	attachmentID := mediadomain.NewGeneratedAttachmentID()
	posts := &fakePostRepository{
		createFunc: func(ctx context.Context, post postdomain.Post) error {
			return nil
		},
	}
	communities := &fakeCommunityPolicy{
		getResult: communityusecase.GetCommunityResult{Community: newCommunityDTO(communityID, "campus")},
		canPostResult: communityusecase.CanPostInCommunityResult{
			Community: newCommunityDTO(communityID, "campus"),
		},
	}
	attachments := &fakeAttachmentRepository{
		bindFunc: func(ctx context.Context, postID postdomain.PostID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, bindAt time.Time) ([]mediadomain.Attachment, error) {
			if uploaderID != authorID {
				t.Fatalf("expected uploader %q, got %q", authorID.String(), uploaderID.String())
			}
			if len(attachmentIDs) != 1 || attachmentIDs[0] != attachmentID {
				t.Fatalf("unexpected attachment ids: %#v", attachmentIDs)
			}
			if maxCount != 9 {
				t.Fatalf("expected max count 9, got %d", maxCount)
			}
			return []mediadomain.Attachment{*mustAttachment(t, attachmentID, authorID, postID, now)}, nil
		},
	}
	uc := NewPostUseCaseWithAttachments(posts, communities, attachments, 9, func() time.Time { return now })

	result, err := uc.PublishPost(context.Background(), PublishPostInput{
		CommunitySlug: "campus",
		AuthorID:      authorID,
		Title:         "Hello",
		Body:          "Post body",
		AttachmentIDs: []string{attachmentID.String()},
	})
	if err != nil {
		t.Fatalf("PublishPost returned error: %v", err)
	}
	if len(result.Post.Attachments) != 1 || result.Post.Attachments[0].ID != attachmentID.String() {
		t.Fatalf("expected attachment response, got %#v", result.Post.Attachments)
	}
	if result.Post.Attachments[0].ThumbnailURL != result.Post.Attachments[0].URL {
		t.Fatalf("expected thumbnail url fallback, got %#v", result.Post.Attachments[0])
	}
}

func TestPublishPostPersistsContentRefs(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 5, 0, 0, time.UTC)
	authorID := userdomain.NewGeneratedUserID()
	communityID := communitydomain.NewGeneratedCommunityID()
	attachmentID := mediadomain.NewGeneratedAttachmentID()
	var createdPostID postdomain.PostID
	posts := &fakePostRepository{
		createFunc: func(ctx context.Context, post postdomain.Post) error {
			createdPostID = post.ID()
			return nil
		},
		replaceContentRefsFunc: func(ctx context.Context, postID postdomain.PostID, refs []ContentRef, replacedAt time.Time) error {
			if postID != createdPostID {
				t.Fatalf("expected post %q, got %q", createdPostID.String(), postID.String())
			}
			if !replacedAt.Equal(now) {
				t.Fatalf("expected replace time %s, got %s", now, replacedAt)
			}
			assertPostContentRefs(t, refs, []ContentRef{
				{Kind: ContentRefKindImage, RefID: attachmentID.String()},
				{Kind: ContentRefKindLink, RefID: "https://example.com/campus"},
			})
			return nil
		},
	}
	communities := &fakeCommunityPolicy{
		getResult: communityusecase.GetCommunityResult{Community: newCommunityDTO(communityID, "campus")},
		canPostResult: communityusecase.CanPostInCommunityResult{
			Community: newCommunityDTO(communityID, "campus"),
		},
	}
	attachments := &fakeAttachmentRepository{
		bindFunc: func(ctx context.Context, postID postdomain.PostID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, bindAt time.Time) ([]mediadomain.Attachment, error) {
			return []mediadomain.Attachment{*mustAttachment(t, attachmentID, authorID, postID, now)}, nil
		},
	}
	uc := NewPostUseCaseWithAttachments(posts, communities, attachments, 9, func() time.Time { return now })

	result, err := uc.PublishPost(context.Background(), PublishPostInput{
		CommunitySlug: "campus",
		AuthorID:      authorID,
		Title:         "Hello",
		Body:          "Post body",
		AttachmentIDs: []string{attachmentID.String()},
		ContentRefs: []ContentRefInput{
			{Kind: "IMAGE", RefID: " " + attachmentID.String() + " "},
			{Kind: ContentRefKindLink, RefID: "https://example.com/campus"},
		},
	})
	if err != nil {
		t.Fatalf("PublishPost returned error: %v", err)
	}
	if !posts.replaceContentRefsCalled {
		t.Fatal("expected content refs to be persisted")
	}
	assertPostContentRefs(t, result.Post.ContentRefs, []ContentRef{
		{Kind: ContentRefKindImage, RefID: attachmentID.String()},
		{Kind: ContentRefKindLink, RefID: "https://example.com/campus"},
	})
}

func TestPublishPostRejectsImageContentRefWithoutBoundAttachment(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 10, 0, 0, time.UTC)
	authorID := userdomain.NewGeneratedUserID()
	communityID := communitydomain.NewGeneratedCommunityID()
	posts := &fakePostRepository{
		createFunc: func(ctx context.Context, post postdomain.Post) error {
			t.Fatal("Create should not be called for invalid image content ref")
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

	_, err := uc.PublishPost(context.Background(), PublishPostInput{
		CommunitySlug: "campus",
		AuthorID:      authorID,
		Title:         "Hello",
		Body:          "Post body",
		ContentRefs: []ContentRefInput{
			{Kind: ContentRefKindImage, RefID: mediadomain.NewGeneratedAttachmentID().String()},
		},
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for unbound image content ref, got %v", err)
	}
}

func TestParseContentRefInputsRejectsEmbedRefWithoutResolvedID(t *testing.T) {
	_, err := ParseContentRefInputs([]ContentRefInput{
		{Kind: ContentRefKindEmbed, RefID: "https://www.douyin.com/video/7123456789012345678"},
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for unresolved embed ref, got %v", err)
	}
}

func TestPublishPostNotifiesMentionedUsers(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 30, 0, 0, time.UTC)
	authorID := userdomain.NewGeneratedUserID()
	communityID := communitydomain.NewGeneratedCommunityID()
	mentionedAlice := mustUser(t, "alice", "active", now)
	mentionedBob := mustUser(t, "bob_123", "active", now)
	disabledUser := mustUser(t, "carol", "disabled", now)
	var createdPostID postdomain.PostID
	posts := &fakePostRepository{
		createFunc: func(ctx context.Context, post postdomain.Post) error {
			createdPostID = post.ID()
			return nil
		},
	}
	communities := &fakeCommunityPolicy{
		getResult: communityusecase.GetCommunityResult{Community: newCommunityDTO(communityID, "campus")},
		canPostResult: communityusecase.CanPostInCommunityResult{
			Community: newCommunityDTO(communityID, "campus"),
		},
	}
	users := &fakePublicUserFinder{
		users: map[string]*userdomain.User{
			"alice":   mentionedAlice,
			"bob_123": mentionedBob,
			"carol":   disabledUser,
		},
	}
	notifications := &fakeNotificationPublisher{}
	uc := NewPostUseCase(posts, communities, func() time.Time { return now })
	uc.SetPublicUserFinder(users)
	uc.SetNotificationPublisher(notifications)

	_, err := uc.PublishPost(context.Background(), PublishPostInput{
		CommunitySlug: "campus",
		AuthorID:      authorID,
		Title:         "Hello",
		Body:          "Hi @Alice, @bob_123, @missing, @carol and @Alice again.",
	})
	if err != nil {
		t.Fatalf("PublishPost returned error: %v", err)
	}
	if len(notifications.mentions) != 2 {
		t.Fatalf("expected two mention notifications, got %#v", notifications.mentions)
	}
	assertMentionNotification(t, notifications.mentions[0], mentionedAlice.ID(), authorID, "post", createdPostID.String())
	assertMentionNotification(t, notifications.mentions[1], mentionedBob.ID(), authorID, "post", createdPostID.String())
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
		listVisibleByCommunityFunc: func(ctx context.Context, id communitydomain.CommunityID, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error) {
			if sort != PostListSortNew {
				t.Fatalf("expected default sort %q, got %q", PostListSortNew, sort)
			}
			if createdAfter != nil {
				t.Fatalf("expected no time range filter, got %v", createdAfter)
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
	if result.Posts[0].Format != PostFormat {
		t.Fatalf("expected format %q, got %q", PostFormat, result.Posts[0].Format)
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
		listVisibleByCommunityFunc: func(ctx context.Context, id communitydomain.CommunityID, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error) {
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

func TestListCommunityPostsPassesTimeRange(t *testing.T) {
	communityID := communitydomain.NewGeneratedCommunityID()
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	wantCreatedAfter := now.Add(-7 * 24 * time.Hour)
	var gotCreatedAfter *time.Time
	posts := &fakePostRepository{
		listVisibleByCommunityFunc: func(ctx context.Context, id communitydomain.CommunityID, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error) {
			gotCreatedAfter = createdAfter
			return nil, nil
		},
	}
	communities := &fakeCommunityPolicy{
		getResult: communityusecase.GetCommunityResult{Community: newCommunityDTO(communityID, "campus")},
	}
	uc := NewPostUseCase(posts, communities, func() time.Time { return now })

	_, err := uc.ListCommunityPosts(context.Background(), ListCommunityPostsInput{
		CommunitySlug: "campus",
		TimeRange:     "week",
	})
	if err != nil {
		t.Fatalf("ListCommunityPosts returned error: %v", err)
	}
	if gotCreatedAfter == nil || !gotCreatedAfter.Equal(wantCreatedAfter) {
		t.Fatalf("expected created_after %v, got %v", wantCreatedAfter, gotCreatedAfter)
	}
}

func TestListLatestPostsReturnsVoteView(t *testing.T) {
	viewerID := userdomain.NewGeneratedUserID()
	communityID := communitydomain.NewGeneratedCommunityID()
	post := mustPost(t, communityID, userdomain.NewGeneratedUserID(), "Latest", time.Now().UTC())
	var gotLimit int
	var gotOffset int
	posts := &fakePostRepository{
		listVisibleInPublicCommunitiesFunc: func(ctx context.Context, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error) {
			if sort != PostListSortNew {
				t.Fatalf("expected default sort %q, got %q", PostListSortNew, sort)
			}
			if createdAfter != nil {
				t.Fatalf("expected no time range filter, got %v", createdAfter)
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

func TestListLatestPostsAnonymousViewerSkipsMyVoteLookup(t *testing.T) {
	communityID := communitydomain.NewGeneratedCommunityID()
	post := mustPost(t, communityID, userdomain.NewGeneratedUserID(), "Latest", time.Now().UTC())
	posts := &fakePostRepository{
		listVisibleInPublicCommunitiesFunc: func(ctx context.Context, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error) {
			return []postdomain.Post{*post}, nil
		},
	}
	votes := &fakeVoteRepository{
		summaries: map[postdomain.PostID]votedomain.PostVoteSummary{
			post.ID(): {
				PostID:        post.ID(),
				UpvoteCount:   5,
				DownvoteCount: 2,
			},
		},
		myVotes: map[postdomain.PostID]votedomain.VoteValue{
			post.ID(): votedomain.VoteValueUp,
		},
	}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, time.Now, votes)

	result, err := uc.ListLatestPosts(context.Background(), ListLatestPostsInput{})
	if err != nil {
		t.Fatalf("ListLatestPosts returned error: %v", err)
	}
	if len(result.Posts) != 1 {
		t.Fatalf("expected one post, got %d", len(result.Posts))
	}
	if result.Posts[0].UpvoteCount != 5 || result.Posts[0].DownvoteCount != 2 || result.Posts[0].Score != 3 || result.Posts[0].MyVote != 0 {
		t.Fatalf("unexpected anonymous vote view: %#v", result.Posts[0])
	}
	if !votes.summarizeCalled {
		t.Fatal("expected vote summary lookup")
	}
	if votes.findByUserCalled {
		t.Fatal("expected no viewer vote lookup for anonymous viewer")
	}
}

func TestListLatestPostsPassesSupportedSort(t *testing.T) {
	tests := []struct {
		raw  string
		want PostListSort
	}{
		{raw: "best", want: PostListSortBest},
		{raw: "hot", want: PostListSortHot},
		{raw: "top", want: PostListSortTop},
		{raw: "rising", want: PostListSortRising},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			var gotSort PostListSort
			posts := &fakePostRepository{
				listVisibleInPublicCommunitiesFunc: func(ctx context.Context, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error) {
					gotSort = sort
					return nil, nil
				},
			}
			uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, time.Now)

			_, err := uc.ListLatestPosts(context.Background(), ListLatestPostsInput{Sort: tt.raw})
			if err != nil {
				t.Fatalf("ListLatestPosts returned error: %v", err)
			}
			if gotSort != tt.want {
				t.Fatalf("expected %q sort, got %q", tt.want, gotSort)
			}
		})
	}
}

func TestListLatestPostsRecommendedDefaultsToHotAndPassesTimeRange(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 30, 0, 0, time.UTC)
	wantCreatedAfter := now.Add(-24 * time.Hour)
	var gotSort PostListSort
	var gotCreatedAfter *time.Time
	viewerID := userdomain.NewGeneratedUserID()
	posts := &fakePostRepository{
		listRecommendedInPublicCommunitiesFunc: func(ctx context.Context, gotViewerID userdomain.UserID, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error) {
			if gotViewerID != viewerID {
				t.Fatalf("expected viewer %q, got %q", viewerID.String(), gotViewerID.String())
			}
			gotSort = sort
			gotCreatedAfter = createdAfter
			return nil, nil
		},
		listVisibleInPublicCommunitiesFunc: func(ctx context.Context, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error) {
			t.Fatal("recommended source should not use generic public list")
			return nil, nil
		},
	}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, func() time.Time { return now })

	_, err := uc.ListLatestPosts(context.Background(), ListLatestPostsInput{
		ViewerID:  viewerID,
		Source:    "recommended",
		TimeRange: "day",
	})
	if err != nil {
		t.Fatalf("ListLatestPosts returned error: %v", err)
	}
	if gotSort != PostListSortHot {
		t.Fatalf("expected recommended default sort %q, got %q", PostListSortHot, gotSort)
	}
	if gotCreatedAfter == nil || !gotCreatedAfter.Equal(wantCreatedAfter) {
		t.Fatalf("expected created_after %v, got %v", wantCreatedAfter, gotCreatedAfter)
	}
}

func TestListLatestPostsRecommendedPassesExplicitSort(t *testing.T) {
	var gotSort PostListSort
	posts := &fakePostRepository{
		listRecommendedInPublicCommunitiesFunc: func(ctx context.Context, viewerID userdomain.UserID, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error) {
			gotSort = sort
			return nil, nil
		},
	}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, time.Now)

	_, err := uc.ListLatestPosts(context.Background(), ListLatestPostsInput{
		Source: "recommended",
		Sort:   "best",
	})
	if err != nil {
		t.Fatalf("ListLatestPosts returned error: %v", err)
	}
	if gotSort != PostListSortBest {
		t.Fatalf("expected explicit recommended sort %q, got %q", PostListSortBest, gotSort)
	}
}

func TestListLatestPostsRejectsInvalidSort(t *testing.T) {
	uc := NewPostUseCase(&fakePostRepository{}, &fakeCommunityPolicy{}, time.Now)

	_, err := uc.ListLatestPosts(context.Background(), ListLatestPostsInput{Sort: "popular"})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid sort, got %v", err)
	}
}

func TestListLatestPostsRejectsInvalidSource(t *testing.T) {
	uc := NewPostUseCase(&fakePostRepository{}, &fakeCommunityPolicy{}, time.Now)

	_, err := uc.ListLatestPosts(context.Background(), ListLatestPostsInput{Source: "friends"})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid source, got %v", err)
	}
}

func TestListLatestPostsRejectsInvalidTimeRange(t *testing.T) {
	uc := NewPostUseCase(&fakePostRepository{}, &fakeCommunityPolicy{}, time.Now)

	_, err := uc.ListLatestPosts(context.Background(), ListLatestPostsInput{TimeRange: "forever"})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid time range, got %v", err)
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
	if result.Post.Format != PostFormat {
		t.Fatalf("expected format %q, got %q", PostFormat, result.Post.Format)
	}
	if result.Post.UpvoteCount != 3 || result.Post.DownvoteCount != 1 || result.Post.Score != 2 || result.Post.MyVote != -1 {
		t.Fatalf("unexpected vote view: %#v", result.Post)
	}
}

func TestGetPostReturnsSaveView(t *testing.T) {
	post := mustPost(t, communitydomain.NewGeneratedCommunityID(), userdomain.NewGeneratedUserID(), "Saved", time.Now().UTC())
	viewerID := userdomain.NewGeneratedUserID()
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
		saveCounts: map[postdomain.PostID]int{
			post.ID(): 3,
		},
		savedPostIDs: map[postdomain.PostID]bool{
			post.ID(): true,
		},
	}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, time.Now)

	result, err := uc.GetPost(context.Background(), GetPostInput{PostID: post.ID().String(), ViewerID: viewerID})
	if err != nil {
		t.Fatalf("GetPost returned error: %v", err)
	}
	if result.Post.SaveCount != 3 || !result.Post.IsSaved {
		t.Fatalf("unexpected save view: save_count=%d is_saved=%t", result.Post.SaveCount, result.Post.IsSaved)
	}
	if !posts.summarizeSavesCalled || !posts.findSavedCalled {
		t.Fatal("expected save count and viewer save lookups")
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

func TestSavePostChecksVisiblePostAndPersistsSave(t *testing.T) {
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	post := mustPost(t, communitydomain.NewGeneratedCommunityID(), userdomain.NewGeneratedUserID(), "Save me", now)
	userID := userdomain.NewGeneratedUserID()
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			if id != post.ID() {
				t.Fatalf("expected post %q, got %q", post.ID().String(), id.String())
			}
			return post, nil
		},
		savePostFunc: func(ctx context.Context, postID postdomain.PostID, gotUserID userdomain.UserID, savedAt time.Time) error {
			if postID != post.ID() || gotUserID != userID || !savedAt.Equal(now) {
				t.Fatalf("unexpected save args: post=%q user=%q at=%s", postID.String(), gotUserID.String(), savedAt)
			}
			return nil
		},
	}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, func() time.Time { return now })

	if _, err := uc.SavePost(context.Background(), SavePostInput{PostID: post.ID().String(), UserID: userID}); err != nil {
		t.Fatalf("SavePost returned error: %v", err)
	}
	if !posts.savePostCalled {
		t.Fatal("expected SavePost repository call")
	}
}

func TestUpdatePostAllowsAuthor(t *testing.T) {
	now := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	updatedAt := now.Add(time.Minute)
	authorID := userdomain.NewGeneratedUserID()
	post := mustPost(t, communitydomain.NewGeneratedCommunityID(), authorID, "Original", now)
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			if id != post.ID() {
				t.Fatalf("expected post id %q, got %q", post.ID().String(), id.String())
			}
			return post, nil
		},
		updateContentFunc: func(ctx context.Context, updated postdomain.Post) error {
			if updated.ID() != post.ID() {
				t.Fatalf("expected updated post %q, got %q", post.ID().String(), updated.ID().String())
			}
			if updated.Title().String() != "Updated" || updated.Body().String() != "Updated body" {
				t.Fatalf("unexpected updated content: title=%q body=%q", updated.Title().String(), updated.Body().String())
			}
			if !updated.UpdatedAt().Equal(updatedAt) {
				t.Fatalf("expected updated_at %s, got %s", updatedAt, updated.UpdatedAt())
			}
			return nil
		},
	}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, func() time.Time { return updatedAt })

	result, err := uc.UpdatePost(context.Background(), UpdatePostInput{
		PostID:  post.ID().String(),
		ActorID: authorID,
		Title:   "Updated",
		Body:    "Updated body",
	})
	if err != nil {
		t.Fatalf("UpdatePost returned error: %v", err)
	}
	if result.Post.Title != "Updated" || result.Post.Body != "Updated body" {
		t.Fatalf("unexpected updated post result: %#v", result.Post)
	}
	if result.Post.Format != PostFormat {
		t.Fatalf("expected format %q, got %q", PostFormat, result.Post.Format)
	}
}

func TestUpdatePostNotifiesOnlyNewMentions(t *testing.T) {
	now := time.Date(2026, 6, 3, 10, 30, 0, 0, time.UTC)
	updatedAt := now.Add(time.Minute)
	authorID := userdomain.NewGeneratedUserID()
	mentionedAlice := mustUser(t, "alice", "active", now)
	mentionedBob := mustUser(t, "bob_123", "active", now)
	post := mustPostWithBody(t, communitydomain.NewGeneratedCommunityID(), authorID, "Original", "Hello @alice", now)
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	users := &fakePublicUserFinder{
		users: map[string]*userdomain.User{
			"alice":   mentionedAlice,
			"bob_123": mentionedBob,
		},
	}
	notifications := &fakeNotificationPublisher{}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, func() time.Time { return updatedAt })
	uc.SetPublicUserFinder(users)
	uc.SetNotificationPublisher(notifications)

	_, err := uc.UpdatePost(context.Background(), UpdatePostInput{
		PostID:  post.ID().String(),
		ActorID: authorID,
		Title:   "Updated",
		Body:    "Hello @alice and @bob_123",
	})
	if err != nil {
		t.Fatalf("UpdatePost returned error: %v", err)
	}
	if len(notifications.mentions) != 1 {
		t.Fatalf("expected one mention notification, got %#v", notifications.mentions)
	}
	assertMentionNotification(t, notifications.mentions[0], mentionedBob.ID(), authorID, "post", post.ID().String())
}

func TestUpdatePostReplacesImageAttachments(t *testing.T) {
	now := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	updatedAt := now.Add(time.Minute)
	authorID := userdomain.NewGeneratedUserID()
	attachmentID := mediadomain.NewGeneratedAttachmentID()
	post := mustPost(t, communitydomain.NewGeneratedCommunityID(), authorID, "Original", now)
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	attachments := &fakeAttachmentRepository{
		replaceFunc: func(ctx context.Context, postID postdomain.PostID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, replaceAt time.Time) ([]mediadomain.Attachment, error) {
			if postID != post.ID() {
				t.Fatalf("expected post %q, got %q", post.ID().String(), postID.String())
			}
			if uploaderID != authorID {
				t.Fatalf("expected uploader %q, got %q", authorID.String(), uploaderID.String())
			}
			if len(attachmentIDs) != 1 || attachmentIDs[0] != attachmentID {
				t.Fatalf("unexpected attachment ids: %#v", attachmentIDs)
			}
			if maxCount != 9 || !replaceAt.Equal(updatedAt) {
				t.Fatalf("unexpected replace metadata: max=%d time=%s", maxCount, replaceAt)
			}
			return []mediadomain.Attachment{*mustAttachment(t, attachmentID, authorID, postID, updatedAt)}, nil
		},
	}
	uc := NewPostUseCaseWithAttachments(posts, &fakeCommunityPolicy{}, attachments, 9, func() time.Time { return updatedAt })
	rawAttachmentIDs := []string{attachmentID.String()}

	result, err := uc.UpdatePost(context.Background(), UpdatePostInput{
		PostID:        post.ID().String(),
		ActorID:       authorID,
		Title:         "Updated",
		Body:          "Updated body",
		AttachmentIDs: &rawAttachmentIDs,
	})
	if err != nil {
		t.Fatalf("UpdatePost returned error: %v", err)
	}
	if len(result.Post.Attachments) != 1 || result.Post.Attachments[0].ID != attachmentID.String() {
		t.Fatalf("expected replacement attachment in result, got %#v", result.Post.Attachments)
	}
	if !attachments.replaceCalled {
		t.Fatal("expected replacement attachment repository call")
	}
}

func TestUpdatePostPreservesContentRefsWhenOmitted(t *testing.T) {
	now := time.Date(2026, 6, 3, 10, 10, 0, 0, time.UTC)
	updatedAt := now.Add(time.Minute)
	authorID := userdomain.NewGeneratedUserID()
	post := mustPost(t, communitydomain.NewGeneratedCommunityID(), authorID, "Original", now)
	existingRefs := []ContentRef{
		{Kind: ContentRefKindLink, RefID: "https://example.com/original"},
		{Kind: ContentRefKindEmbed, RefID: "1d2d1912-e4b6-4e0d-a7c2-5f2c57c4ce91"},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
		listContentRefsFunc: func(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID][]ContentRef, error) {
			if len(postIDs) != 1 || postIDs[0] != post.ID() {
				t.Fatalf("unexpected post ids: %#v", postIDs)
			}
			return map[postdomain.PostID][]ContentRef{post.ID(): existingRefs}, nil
		},
	}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, func() time.Time { return updatedAt })

	result, err := uc.UpdatePost(context.Background(), UpdatePostInput{
		PostID:  post.ID().String(),
		ActorID: authorID,
		Title:   "Updated",
		Body:    "Updated body",
	})
	if err != nil {
		t.Fatalf("UpdatePost returned error: %v", err)
	}
	if !posts.listContentRefsCalled {
		t.Fatal("expected existing content refs to be loaded")
	}
	if posts.replaceContentRefsCalled {
		t.Fatal("did not expect omitted content_refs to replace existing refs")
	}
	assertPostContentRefs(t, result.Post.ContentRefs, existingRefs)
}

func TestUpdatePostClearsContentRefsWithEmptyArray(t *testing.T) {
	now := time.Date(2026, 6, 3, 10, 20, 0, 0, time.UTC)
	updatedAt := now.Add(time.Minute)
	authorID := userdomain.NewGeneratedUserID()
	post := mustPost(t, communitydomain.NewGeneratedCommunityID(), authorID, "Original", now)
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
		replaceContentRefsFunc: func(ctx context.Context, postID postdomain.PostID, refs []ContentRef, replacedAt time.Time) error {
			if postID != post.ID() {
				t.Fatalf("expected post %q, got %q", post.ID().String(), postID.String())
			}
			if len(refs) != 0 {
				t.Fatalf("expected content refs to be cleared, got %#v", refs)
			}
			return nil
		},
	}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, func() time.Time { return updatedAt })
	contentRefs := []ContentRefInput{}

	result, err := uc.UpdatePost(context.Background(), UpdatePostInput{
		PostID:      post.ID().String(),
		ActorID:     authorID,
		Title:       "Updated",
		Body:        "Updated body",
		ContentRefs: &contentRefs,
	})
	if err != nil {
		t.Fatalf("UpdatePost returned error: %v", err)
	}
	if !posts.replaceContentRefsCalled {
		t.Fatal("expected empty content_refs to replace stored refs")
	}
	if len(result.Post.ContentRefs) != 0 {
		t.Fatalf("expected empty content refs in result, got %#v", result.Post.ContentRefs)
	}
}

func TestUpdatePostRejectsNonAuthor(t *testing.T) {
	post := mustPost(t, communitydomain.NewGeneratedCommunityID(), userdomain.NewGeneratedUserID(), "Original", time.Now().UTC())
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
		updateContentFunc: func(ctx context.Context, updated postdomain.Post) error {
			t.Fatal("UpdateContent should not be called for non-author")
			return nil
		},
	}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, time.Now)

	_, err := uc.UpdatePost(context.Background(), UpdatePostInput{
		PostID:  post.ID().String(),
		ActorID: userdomain.NewGeneratedUserID(),
		Title:   "Updated",
		Body:    "Updated body",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for non-author, got %v", err)
	}
}

func TestDeletePostMarksAuthorPostDeleted(t *testing.T) {
	now := time.Date(2026, 6, 3, 11, 0, 0, 0, time.UTC)
	deletedAt := now.Add(time.Minute)
	authorID := userdomain.NewGeneratedUserID()
	post := mustPost(t, communitydomain.NewGeneratedCommunityID(), authorID, "Original", now)
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
		markDeletedFunc: func(ctx context.Context, deleted postdomain.Post) error {
			if deleted.Status() != postdomain.PostStatusDeleted {
				t.Fatalf("expected deleted status, got %q", deleted.Status().String())
			}
			if !deleted.UpdatedAt().Equal(deletedAt) {
				t.Fatalf("expected deleted_at %s, got %s", deletedAt, deleted.UpdatedAt())
			}
			return nil
		},
	}
	uc := NewPostUseCase(posts, &fakeCommunityPolicy{}, func() time.Time { return deletedAt })

	if _, err := uc.DeletePost(context.Background(), DeletePostInput{PostID: post.ID().String(), ActorID: authorID}); err != nil {
		t.Fatalf("DeletePost returned error: %v", err)
	}
}

func TestDeletePostRejectsInvalidInput(t *testing.T) {
	uc := NewPostUseCase(&fakePostRepository{}, &fakeCommunityPolicy{}, time.Now)

	_, err := uc.DeletePost(context.Background(), DeletePostInput{
		PostID:  postdomain.NewGeneratedPostID().String(),
		ActorID: "",
	})
	if !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated for missing actor, got %v", err)
	}

	_, err = uc.DeletePost(context.Background(), DeletePostInput{
		PostID:  "not-a-uuid",
		ActorID: userdomain.NewGeneratedUserID(),
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid post id, got %v", err)
	}
}

type fakePostRepository struct {
	createFunc                             func(ctx context.Context, post postdomain.Post) error
	findVisibleByIDFunc                    func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error)
	updateContentFunc                      func(ctx context.Context, post postdomain.Post) error
	markDeletedFunc                        func(ctx context.Context, post postdomain.Post) error
	listVisibleByCommunityFunc             func(ctx context.Context, communityID communitydomain.CommunityID, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error)
	listVisibleInPublicCommunitiesFunc     func(ctx context.Context, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error)
	listRecommendedInPublicCommunitiesFunc func(ctx context.Context, viewerID userdomain.UserID, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error)
	listVisibleByAuthorFunc                func(ctx context.Context, authorID userdomain.UserID, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error)
	savePostFunc                           func(ctx context.Context, postID postdomain.PostID, userID userdomain.UserID, now time.Time) error
	deletePostSaveFunc                     func(ctx context.Context, postID postdomain.PostID, userID userdomain.UserID) error
	listSavedVisiblePostsFunc              func(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]postdomain.Post, error)
	replaceContentRefsFunc                 func(ctx context.Context, postID postdomain.PostID, refs []ContentRef, now time.Time) error
	listContentRefsFunc                    func(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID][]ContentRef, error)
	savePostCalled                         bool
	deletePostSaveCalled                   bool
	listSavedVisiblePostsCalled            bool
	summarizeSavesCalled                   bool
	findSavedCalled                        bool
	replaceContentRefsCalled               bool
	listContentRefsCalled                  bool
	saveCounts                             map[postdomain.PostID]int
	savedPostIDs                           map[postdomain.PostID]bool
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

func (f *fakePostRepository) UpdateContent(ctx context.Context, post postdomain.Post) error {
	if f.updateContentFunc != nil {
		return f.updateContentFunc(ctx, post)
	}
	return nil
}

func (f *fakePostRepository) MarkDeleted(ctx context.Context, post postdomain.Post) error {
	if f.markDeletedFunc != nil {
		return f.markDeletedFunc(ctx, post)
	}
	return nil
}

func (f *fakePostRepository) ListVisibleByCommunity(ctx context.Context, communityID communitydomain.CommunityID, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error) {
	if f.listVisibleByCommunityFunc != nil {
		return f.listVisibleByCommunityFunc(ctx, communityID, sort, createdAfter, limit, offset)
	}
	return nil, nil
}

func (f *fakePostRepository) ListVisibleInPublicCommunities(ctx context.Context, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error) {
	if f.listVisibleInPublicCommunitiesFunc != nil {
		return f.listVisibleInPublicCommunitiesFunc(ctx, sort, createdAfter, limit, offset)
	}
	return nil, nil
}

func (f *fakePostRepository) ListRecommendedInPublicCommunities(ctx context.Context, viewerID userdomain.UserID, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error) {
	if f.listRecommendedInPublicCommunitiesFunc != nil {
		return f.listRecommendedInPublicCommunitiesFunc(ctx, viewerID, sort, createdAfter, limit, offset)
	}
	return nil, nil
}

func (f *fakePostRepository) ListVisibleByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID, sort PostListSort, createdAfter *time.Time, limit int, offset int) ([]postdomain.Post, error) {
	if f.listVisibleByAuthorFunc != nil {
		return f.listVisibleByAuthorFunc(ctx, authorID, sort, createdAfter, limit, offset)
	}
	return nil, nil
}

func (f *fakePostRepository) SavePost(ctx context.Context, postID postdomain.PostID, userID userdomain.UserID, now time.Time) error {
	f.savePostCalled = true
	if f.savePostFunc != nil {
		return f.savePostFunc(ctx, postID, userID, now)
	}
	return nil
}

func (f *fakePostRepository) DeletePostSave(ctx context.Context, postID postdomain.PostID, userID userdomain.UserID) error {
	f.deletePostSaveCalled = true
	if f.deletePostSaveFunc != nil {
		return f.deletePostSaveFunc(ctx, postID, userID)
	}
	return nil
}

func (f *fakePostRepository) ListSavedVisiblePosts(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]postdomain.Post, error) {
	f.listSavedVisiblePostsCalled = true
	if f.listSavedVisiblePostsFunc != nil {
		return f.listSavedVisiblePostsFunc(ctx, userID, limit, offset)
	}
	return nil, nil
}

func (f *fakePostRepository) FindSavedPostIDsByUser(ctx context.Context, postIDs []postdomain.PostID, userID userdomain.UserID) (map[postdomain.PostID]bool, error) {
	f.findSavedCalled = true
	if f.savedPostIDs == nil {
		return map[postdomain.PostID]bool{}, nil
	}
	return f.savedPostIDs, nil
}

func (f *fakePostRepository) SummarizeSavesByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID]int, error) {
	f.summarizeSavesCalled = true
	if f.saveCounts == nil {
		return map[postdomain.PostID]int{}, nil
	}
	return f.saveCounts, nil
}

func (f *fakePostRepository) ReplacePostContentRefs(ctx context.Context, postID postdomain.PostID, refs []ContentRef, now time.Time) error {
	f.replaceContentRefsCalled = true
	if f.replaceContentRefsFunc != nil {
		return f.replaceContentRefsFunc(ctx, postID, refs, now)
	}
	return nil
}

func (f *fakePostRepository) ListPostContentRefsByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID][]ContentRef, error) {
	f.listContentRefsCalled = true
	if f.listContentRefsFunc != nil {
		return f.listContentRefsFunc(ctx, postIDs)
	}
	return map[postdomain.PostID][]ContentRef{}, nil
}

type fakeVoteRepository struct {
	summarizeCalled  bool
	findByUserCalled bool
	summaries        map[postdomain.PostID]votedomain.PostVoteSummary
	myVotes          map[postdomain.PostID]votedomain.VoteValue
	summarizeErr     error
	findByUserErr    error
}

type fakeAttachmentRepository struct {
	bindFunc      func(ctx context.Context, postID postdomain.PostID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error)
	replaceFunc   func(ctx context.Context, postID postdomain.PostID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error)
	listFunc      func(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID][]mediadomain.Attachment, error)
	replaceCalled bool
}

func (f *fakeAttachmentRepository) BindReadyImagesToPost(ctx context.Context, postID postdomain.PostID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error) {
	if f.bindFunc != nil {
		return f.bindFunc(ctx, postID, uploaderID, attachmentIDs, maxCount, now)
	}
	return nil, nil
}

func (f *fakeAttachmentRepository) ReplaceReadyImagesForPost(ctx context.Context, postID postdomain.PostID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error) {
	f.replaceCalled = true
	if f.replaceFunc != nil {
		return f.replaceFunc(ctx, postID, uploaderID, attachmentIDs, maxCount, now)
	}
	return nil, nil
}

func (f *fakeAttachmentRepository) ListReadyImagesByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID][]mediadomain.Attachment, error) {
	if f.listFunc != nil {
		return f.listFunc(ctx, postIDs)
	}
	return map[postdomain.PostID][]mediadomain.Attachment{}, nil
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

type fakeNotificationPublisher struct {
	mentions []fakeMentionNotification
}

type fakeMentionNotification struct {
	recipientID userdomain.UserID
	actorID     userdomain.UserID
	sourceType  string
	sourceID    string
}

func (f *fakeNotificationPublisher) NotifyMentioned(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, sourceType string, sourceID string) error {
	f.mentions = append(f.mentions, fakeMentionNotification{
		recipientID: recipientID,
		actorID:     actorID,
		sourceType:  sourceType,
		sourceID:    sourceID,
	})
	return nil
}

type fakePublicUserFinder struct {
	users map[string]*userdomain.User
}

func (f *fakePublicUserFinder) FindByUsername(ctx context.Context, username userdomain.Username) (*userdomain.User, error) {
	if f.users != nil {
		if user, ok := f.users[username.String()]; ok {
			return user, nil
		}
	}
	return nil, apperr.New(apperr.CodeNotFound, "user not found")
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

	return mustPostWithBody(t, communityID, authorID, title, "Body for "+title, now)
}

func mustPostWithBody(t *testing.T, communityID communitydomain.CommunityID, authorID userdomain.UserID, title string, body string, now time.Time) *postdomain.Post {
	t.Helper()

	postTitle, err := postdomain.NewPostTitle(title)
	if err != nil {
		t.Fatalf("NewPostTitle returned error: %v", err)
	}
	postBody, err := postdomain.NewPostBody(body)
	if err != nil {
		t.Fatalf("NewPostBody returned error: %v", err)
	}
	post, err := postdomain.NewPost(postdomain.NewGeneratedPostID(), communityID, authorID, postTitle, postBody, now)
	if err != nil {
		t.Fatalf("NewPost returned error: %v", err)
	}
	return post
}

func mustUser(t *testing.T, username string, status string, now time.Time) *userdomain.User {
	t.Helper()

	parsedUsername, err := userdomain.NewUsername(username)
	if err != nil {
		t.Fatalf("NewUsername returned error: %v", err)
	}
	passwordHash, err := userdomain.NewPasswordHash("hashed-password")
	if err != nil {
		t.Fatalf("NewPasswordHash returned error: %v", err)
	}
	userStatus, err := userdomain.NewUserStatus(status)
	if err != nil {
		t.Fatalf("NewUserStatus returned error: %v", err)
	}
	user, err := userdomain.RehydrateUser(userdomain.NewGeneratedUserID(), parsedUsername, passwordHash, userStatus, now, now)
	if err != nil {
		t.Fatalf("RehydrateUser returned error: %v", err)
	}
	return user
}

func assertMentionNotification(t *testing.T, notification fakeMentionNotification, recipientID userdomain.UserID, actorID userdomain.UserID, sourceType string, sourceID string) {
	t.Helper()

	if notification.recipientID != recipientID || notification.actorID != actorID || notification.sourceType != sourceType || notification.sourceID != sourceID {
		t.Fatalf("unexpected mention notification: %#v", notification)
	}
}

func assertPostContentRefs(t *testing.T, got []ContentRef, want []ContentRef) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d content refs, got %d: %#v", len(want), len(got), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected content ref at %d: got %#v want %#v", index, got[index], want[index])
		}
	}
}

func mustAttachment(t *testing.T, attachmentID mediadomain.AttachmentID, uploaderID userdomain.UserID, postID postdomain.PostID, now time.Time) *mediadomain.Attachment {
	t.Helper()

	attachment, err := mediadomain.RehydrateAttachment(mediadomain.NewAttachmentParams{
		ID:              attachmentID,
		OwnerType:       mediadomain.OwnerTypePost,
		OwnerID:         postID.String(),
		UploaderID:      uploaderID,
		Kind:            mediadomain.AttachmentKindImage,
		StorageProvider: mediadomain.StorageProviderLocal,
		Bucket:          "local",
		ObjectKey:       "images/test.png",
		PublicURL:       "http://localhost:8080/uploads/images/test.png",
		SizeBytes:       100,
		MimeType:        "image/png",
		Status:          mediadomain.AttachmentStatusReady,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("RehydrateAttachment returned error: %v", err)
	}
	return attachment
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

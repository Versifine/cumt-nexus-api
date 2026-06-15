package commentusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
)

func TestPublishCommentCreatesVisibleComment(t *testing.T) {
	now := time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC)
	post := mustPost(t, time.Now().UTC())
	authorID := userdomain.NewGeneratedUserID()
	parent := mustComment(t, post.ID(), authorID, nil, "Parent", now.Add(-time.Minute))
	comments := &fakeCommentRepository{
		findVisibleByIDFunc: func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
			if id != parent.ID() {
				t.Fatalf("expected parent %q, got %q", parent.ID().String(), id.String())
			}
			return parent, nil
		},
		createFunc: func(ctx context.Context, comment commentdomain.Comment) error {
			if comment.PostID() != post.ID() {
				t.Fatalf("expected post %q, got %q", post.ID().String(), comment.PostID().String())
			}
			if comment.AuthorID() != authorID {
				t.Fatalf("expected author %q, got %q", authorID.String(), comment.AuthorID().String())
			}
			if gotParentID, ok := comment.ParentID(); !ok || gotParentID != parent.ID() {
				t.Fatalf("expected parent %q, got %q present=%t", parent.ID().String(), gotParentID.String(), ok)
			}
			return nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, func() time.Time { return now })

	result, err := uc.PublishComment(context.Background(), PublishCommentInput{
		PostID:   post.ID().String(),
		AuthorID: authorID,
		ParentID: parent.ID().String(),
		Body:     "Reply",
	})
	if err != nil {
		t.Fatalf("PublishComment returned error: %v", err)
	}
	if result.Comment.Status != commentdomain.CommentStatusVisible.String() {
		t.Fatalf("expected visible status, got %q", result.Comment.Status)
	}
}

func TestPublishCommentBindsImageAttachments(t *testing.T) {
	now := time.Date(2026, 6, 2, 14, 0, 0, 0, time.UTC)
	post := mustPost(t, now)
	authorID := userdomain.NewGeneratedUserID()
	attachmentID := mediadomain.NewGeneratedAttachmentID()
	var createdCommentID commentdomain.CommentID
	comments := &fakeCommentRepository{
		createFunc: func(ctx context.Context, comment commentdomain.Comment) error {
			createdCommentID = comment.ID()
			return nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	attachments := &fakeAttachmentRepository{
		bindReadyImagesToCommentFunc: func(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, bindTime time.Time) ([]mediadomain.Attachment, error) {
			if commentID != createdCommentID {
				t.Fatalf("expected comment %q, got %q", createdCommentID.String(), commentID.String())
			}
			if uploaderID != authorID {
				t.Fatalf("expected uploader %q, got %q", authorID.String(), uploaderID.String())
			}
			if len(attachmentIDs) != 1 || attachmentIDs[0] != attachmentID {
				t.Fatalf("unexpected attachment ids: %#v", attachmentIDs)
			}
			if maxCount != 1 || !bindTime.Equal(now) {
				t.Fatalf("unexpected bind metadata: max=%d time=%s", maxCount, bindTime)
			}
			return []mediadomain.Attachment{*mustMediaAttachment(t, attachmentID, mediadomain.OwnerTypeComment, commentID.String(), authorID, now)}, nil
		},
	}
	uc := NewCommentUseCaseWithAttachments(comments, posts, attachments, 1, func() time.Time { return now })

	result, err := uc.PublishComment(context.Background(), PublishCommentInput{
		PostID:        post.ID().String(),
		AuthorID:      authorID,
		Body:          "Reply",
		AttachmentIDs: []string{attachmentID.String()},
	})
	if err != nil {
		t.Fatalf("PublishComment returned error: %v", err)
	}
	if len(result.Comment.Attachments) != 1 || result.Comment.Attachments[0].ID != attachmentID.String() {
		t.Fatalf("expected bound attachment in response, got %#v", result.Comment.Attachments)
	}
	if result.Comment.Attachments[0].ThumbnailURL != result.Comment.Attachments[0].URL {
		t.Fatalf("expected thumbnail url fallback, got %#v", result.Comment.Attachments[0])
	}
}

func TestPublishCommentPersistsContentRefs(t *testing.T) {
	now := time.Date(2026, 6, 2, 14, 5, 0, 0, time.UTC)
	post := mustPost(t, now)
	authorID := userdomain.NewGeneratedUserID()
	attachmentID := mediadomain.NewGeneratedAttachmentID()
	var createdCommentID commentdomain.CommentID
	comments := &fakeCommentRepository{
		createFunc: func(ctx context.Context, comment commentdomain.Comment) error {
			createdCommentID = comment.ID()
			return nil
		},
		replaceContentRefsFunc: func(ctx context.Context, commentID commentdomain.CommentID, refs []postusecase.ContentRef, replacedAt time.Time) error {
			if commentID != createdCommentID {
				t.Fatalf("expected comment %q, got %q", createdCommentID.String(), commentID.String())
			}
			if !replacedAt.Equal(now) {
				t.Fatalf("expected replace time %s, got %s", now, replacedAt)
			}
			assertCommentContentRefs(t, refs, []postusecase.ContentRef{
				{Kind: postusecase.ContentRefKindImage, RefID: attachmentID.String()},
				{Kind: postusecase.ContentRefKindLink, RefID: "https://example.com/comment"},
			})
			return nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	attachments := &fakeAttachmentRepository{
		bindReadyImagesToCommentFunc: func(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, bindTime time.Time) ([]mediadomain.Attachment, error) {
			return []mediadomain.Attachment{*mustMediaAttachment(t, attachmentID, mediadomain.OwnerTypeComment, commentID.String(), authorID, now)}, nil
		},
	}
	uc := NewCommentUseCaseWithAttachments(comments, posts, attachments, 1, func() time.Time { return now })

	result, err := uc.PublishComment(context.Background(), PublishCommentInput{
		PostID:        post.ID().String(),
		AuthorID:      authorID,
		Body:          "Reply",
		AttachmentIDs: []string{attachmentID.String()},
		ContentRefs: []postusecase.ContentRefInput{
			{Kind: "IMAGE", RefID: " " + attachmentID.String() + " "},
			{Kind: postusecase.ContentRefKindLink, RefID: "https://example.com/comment"},
		},
	})
	if err != nil {
		t.Fatalf("PublishComment returned error: %v", err)
	}
	if !comments.replaceContentRefsCalled {
		t.Fatal("expected comment content refs to be persisted")
	}
	assertCommentContentRefs(t, result.Comment.ContentRefs, []postusecase.ContentRef{
		{Kind: postusecase.ContentRefKindImage, RefID: attachmentID.String()},
		{Kind: postusecase.ContentRefKindLink, RefID: "https://example.com/comment"},
	})
}

func TestPublishCommentRejectsImageContentRefWithoutBoundAttachment(t *testing.T) {
	now := time.Date(2026, 6, 2, 14, 10, 0, 0, time.UTC)
	post := mustPost(t, now)
	comments := &fakeCommentRepository{
		createFunc: func(ctx context.Context, comment commentdomain.Comment) error {
			t.Fatal("Create should not be called for invalid image content ref")
			return nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, func() time.Time { return now })

	_, err := uc.PublishComment(context.Background(), PublishCommentInput{
		PostID:   post.ID().String(),
		AuthorID: userdomain.NewGeneratedUserID(),
		Body:     "Reply",
		ContentRefs: []postusecase.ContentRefInput{
			{Kind: postusecase.ContentRefKindImage, RefID: mediadomain.NewGeneratedAttachmentID().String()},
		},
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for unbound image content ref, got %v", err)
	}
}

func TestPublishCommentRejectsInvalidInput(t *testing.T) {
	uc := NewCommentUseCase(&fakeCommentRepository{}, &fakePostRepository{}, time.Now)

	_, err := uc.PublishComment(context.Background(), PublishCommentInput{
		PostID:   postdomain.NewGeneratedPostID().String(),
		AuthorID: "",
		Body:     "Body",
	})
	if !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated for missing author, got %v", err)
	}

	_, err = uc.PublishComment(context.Background(), PublishCommentInput{
		PostID:   "not-a-uuid",
		AuthorID: userdomain.NewGeneratedUserID(),
		Body:     "Body",
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid post id, got %v", err)
	}

	post := mustPost(t, time.Now().UTC())
	ucWithPost := NewCommentUseCase(&fakeCommentRepository{}, &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}, time.Now)
	_, err = ucWithPost.PublishComment(context.Background(), PublishCommentInput{
		PostID:        postdomain.NewGeneratedPostID().String(),
		AuthorID:      userdomain.NewGeneratedUserID(),
		Body:          "Body",
		AttachmentIDs: []string{"not-a-uuid"},
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid attachment id, got %v", err)
	}
}

func TestPublishCommentRejectsLockedPost(t *testing.T) {
	now := time.Now().UTC()
	basePost := mustPost(t, now)
	lockedPost, err := postdomain.RehydratePostWithModerationState(
		basePost.ID(),
		basePost.CommunityID(),
		basePost.AuthorID(),
		basePost.Title(),
		basePost.Body(),
		basePost.Status(),
		true,
		false,
		false,
		false,
		"",
		basePost.CreatedAt(),
		basePost.UpdatedAt(),
	)
	if err != nil {
		t.Fatalf("RehydratePostWithModerationState returned error: %v", err)
	}
	uc := NewCommentUseCase(&fakeCommentRepository{}, &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return lockedPost, nil
		},
	}, time.Now)

	_, err = uc.PublishComment(context.Background(), PublishCommentInput{
		PostID:   lockedPost.ID().String(),
		AuthorID: userdomain.NewGeneratedUserID(),
		Body:     "Body",
	})
	if !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for locked post, got %v", err)
	}
}

func TestPublishCommentRejectsParentFromDifferentPost(t *testing.T) {
	post := mustPost(t, time.Now().UTC())
	otherPost := mustPost(t, time.Now().UTC())
	parent := mustComment(t, otherPost.ID(), userdomain.NewGeneratedUserID(), nil, "Parent", time.Now().UTC())
	comments := &fakeCommentRepository{
		findVisibleByIDFunc: func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
			return parent, nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, time.Now)

	_, err := uc.PublishComment(context.Background(), PublishCommentInput{
		PostID:   post.ID().String(),
		AuthorID: userdomain.NewGeneratedUserID(),
		ParentID: parent.ID().String(),
		Body:     "Reply",
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for parent from another post, got %v", err)
	}
}

func TestPublishRootCommentNotifiesPostAuthor(t *testing.T) {
	now := time.Date(2026, 6, 2, 15, 0, 0, 0, time.UTC)
	post := mustPost(t, now)
	authorID := userdomain.NewGeneratedUserID()
	comments := &fakeCommentRepository{}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	notifications := &fakeNotificationPublisher{}
	uc := NewCommentUseCase(comments, posts, func() time.Time { return now })
	uc.SetNotificationPublisher(notifications)

	_, err := uc.PublishComment(context.Background(), PublishCommentInput{
		PostID:   post.ID().String(),
		AuthorID: authorID,
		Body:     "Root comment",
	})
	if err != nil {
		t.Fatalf("PublishComment returned error: %v", err)
	}
	if !notifications.postCommentedCalled {
		t.Fatal("expected post comment notification")
	}
	if notifications.recipientID != post.AuthorID() || notifications.actorID != authorID || notifications.postID != post.ID().String() {
		t.Fatalf("unexpected notification args: %#v", notifications)
	}
}

func TestPublishReplyNotifiesParentCommentAuthor(t *testing.T) {
	now := time.Date(2026, 6, 2, 15, 5, 0, 0, time.UTC)
	post := mustPost(t, now)
	authorID := userdomain.NewGeneratedUserID()
	parentAuthorID := userdomain.NewGeneratedUserID()
	parent := mustComment(t, post.ID(), parentAuthorID, nil, "Parent", now.Add(-time.Minute))
	var createdCommentID commentdomain.CommentID
	comments := &fakeCommentRepository{
		findVisibleByIDFunc: func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
			return parent, nil
		},
		createFunc: func(ctx context.Context, comment commentdomain.Comment) error {
			createdCommentID = comment.ID()
			return nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	notifications := &fakeNotificationPublisher{}
	uc := NewCommentUseCase(comments, posts, func() time.Time { return now })
	uc.SetNotificationPublisher(notifications)

	_, err := uc.PublishComment(context.Background(), PublishCommentInput{
		PostID:   post.ID().String(),
		AuthorID: authorID,
		ParentID: parent.ID().String(),
		Body:     "Reply",
	})
	if err != nil {
		t.Fatalf("PublishComment returned error: %v", err)
	}
	if !notifications.commentRepliedCalled {
		t.Fatal("expected comment reply notification")
	}
	if notifications.recipientID != parentAuthorID || notifications.actorID != authorID || notifications.commentID != createdCommentID.String() {
		t.Fatalf("unexpected notification args: %#v", notifications)
	}
}

func TestPublishCommentNotifiesMentionedUsers(t *testing.T) {
	now := time.Date(2026, 6, 2, 15, 10, 0, 0, time.UTC)
	post := mustPost(t, now)
	authorID := userdomain.NewGeneratedUserID()
	mentionedAlice := mustUser(t, "alice", "active", now)
	mentionedBob := mustUser(t, "bob_123", "active", now)
	disabledUser := mustUser(t, "carol", "disabled", now)
	var createdCommentID commentdomain.CommentID
	comments := &fakeCommentRepository{
		createFunc: func(ctx context.Context, comment commentdomain.Comment) error {
			createdCommentID = comment.ID()
			return nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
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
	uc := NewCommentUseCase(comments, posts, func() time.Time { return now })
	uc.SetPublicUserFinder(users)
	uc.SetNotificationPublisher(notifications)

	_, err := uc.PublishComment(context.Background(), PublishCommentInput{
		PostID:   post.ID().String(),
		AuthorID: authorID,
		Body:     "Hi @Alice, @bob_123, @missing, @carol and @Alice again.",
	})
	if err != nil {
		t.Fatalf("PublishComment returned error: %v", err)
	}
	if len(notifications.mentions) != 2 {
		t.Fatalf("expected two mention notifications, got %#v", notifications.mentions)
	}
	assertMentionNotification(t, notifications.mentions[0], mentionedAlice.ID(), authorID, "comment", createdCommentID.String())
	assertMentionNotification(t, notifications.mentions[1], mentionedBob.ID(), authorID, "comment", createdCommentID.String())
}

func TestListPostCommentsNormalizesPagination(t *testing.T) {
	post := mustPost(t, time.Now().UTC())
	var gotLimit int
	var gotOffset int
	comments := &fakeCommentRepository{
		listVisibleByPostFunc: func(ctx context.Context, postID postdomain.PostID, limit int, offset int) ([]commentdomain.Comment, error) {
			gotLimit = limit
			gotOffset = offset
			return []commentdomain.Comment{*mustComment(t, post.ID(), userdomain.NewGeneratedUserID(), nil, "Body", time.Now().UTC())}, nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, time.Now)

	result, err := uc.ListPostComments(context.Background(), ListPostCommentsInput{
		PostID: post.ID().String(),
		Limit:  100,
		Offset: 5,
	})
	if err != nil {
		t.Fatalf("ListPostComments returned error: %v", err)
	}
	if gotLimit != MaxCommentListLimit+1 || result.Limit != MaxCommentListLimit {
		t.Fatalf("expected clamped limit %d, got repo=%d result=%d", MaxCommentListLimit, gotLimit, result.Limit)
	}
	if gotOffset != 5 || result.Offset != 5 {
		t.Fatalf("expected offset 5, got repo=%d result=%d", gotOffset, result.Offset)
	}
	if len(result.Comments) != 1 {
		t.Fatalf("expected one comment, got %d", len(result.Comments))
	}
}

func TestListPostCommentsPassesFlatSort(t *testing.T) {
	post := mustPost(t, time.Now().UTC())
	comment := mustComment(t, post.ID(), userdomain.NewGeneratedUserID(), nil, "Body", time.Now().UTC())
	comments := &fakeCommentRepository{
		listVisibleByPostFunc: func(ctx context.Context, postID postdomain.PostID, limit int, offset int) ([]commentdomain.Comment, error) {
			return []commentdomain.Comment{*comment}, nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, time.Now)

	result, err := uc.ListPostComments(context.Background(), ListPostCommentsInput{
		PostID: post.ID().String(),
		Sort:   "top",
	})
	if err != nil {
		t.Fatalf("ListPostComments returned error: %v", err)
	}
	if comments.listVisibleSort != CommentListSortTop || result.Sort != CommentListSortTop.String() {
		t.Fatalf("expected top sort, got repo=%q result=%q", comments.listVisibleSort, result.Sort)
	}
}

func TestListPostCommentsReturnsVoteView(t *testing.T) {
	post := mustPost(t, time.Now().UTC())
	viewerID := userdomain.NewGeneratedUserID()
	comment := mustComment(t, post.ID(), userdomain.NewGeneratedUserID(), nil, "Body", time.Now().UTC())
	comments := &fakeCommentRepository{
		listVisibleByPostFunc: func(ctx context.Context, postID postdomain.PostID, limit int, offset int) ([]commentdomain.Comment, error) {
			return []commentdomain.Comment{*comment}, nil
		},
		voteSummaries: map[commentdomain.CommentID]votedomain.CommentVoteSummary{
			comment.ID(): {
				CommentID:     comment.ID(),
				UpvoteCount:   4,
				DownvoteCount: 1,
			},
		},
		myVotes: map[commentdomain.CommentID]votedomain.VoteValue{
			comment.ID(): votedomain.VoteValueUp,
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, time.Now)

	result, err := uc.ListPostComments(context.Background(), ListPostCommentsInput{
		PostID:   post.ID().String(),
		ViewerID: viewerID,
	})
	if err != nil {
		t.Fatalf("ListPostComments returned error: %v", err)
	}
	if len(result.Comments) != 1 {
		t.Fatalf("expected one comment, got %d", len(result.Comments))
	}
	got := result.Comments[0]
	if got.UpvoteCount != 4 || got.DownvoteCount != 1 || got.Score != 3 || got.MyVote != 1 {
		t.Fatalf("unexpected vote view: %#v", got)
	}
	if !comments.summarizeVotesCalled || !comments.findVotesCalled {
		t.Fatal("expected comment vote summary and viewer lookups")
	}
}

func TestListPostCommentsReturnsCommentEffects(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	post := mustPost(t, now)
	applierID := userdomain.NewGeneratedUserID()
	comment := mustComment(t, post.ID(), userdomain.NewGeneratedUserID(), nil, "Body", now)
	comments := &fakeCommentRepository{
		listVisibleByPostFunc: func(ctx context.Context, postID postdomain.PostID, limit int, offset int) ([]commentdomain.Comment, error) {
			return []commentdomain.Comment{*comment}, nil
		},
		commentEffects: map[commentdomain.CommentID][]CommentEffectSummary{
			comment.ID(): {{
				ID:           "d9f1ff4d-8a69-4f0c-8c24-8f3b0b8994b4",
				EffectID:     "sparkle",
				Name:         "Sparkle",
				AssetURL:     "https://example.com/sparkle.png",
				AnimationKey: "sparkle",
				AppliedByUser: postusecase.UserSummary{
					ID:          applierID.String(),
					Username:    "alice",
					DisplayName: "Alice",
					Badges:      []string{},
				},
				PointsSpent: 10,
				CreatedAt:   now,
			}},
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, time.Now)

	result, err := uc.ListPostComments(context.Background(), ListPostCommentsInput{
		PostID: post.ID().String(),
	})
	if err != nil {
		t.Fatalf("ListPostComments returned error: %v", err)
	}
	if !comments.listCommentEffectsCalled {
		t.Fatal("expected comment effects to be loaded")
	}
	if len(result.Comments) != 1 || len(result.Comments[0].Effects) != 1 {
		t.Fatalf("expected one comment effect, got %#v", result.Comments)
	}
	effect := result.Comments[0].Effects[0]
	if effect.EffectID != "sparkle" || effect.Name != "Sparkle" || effect.AppliedByUser.Username != "alice" || effect.PointsSpent != 10 {
		t.Fatalf("unexpected comment effect: %#v", effect)
	}
}

func TestListPostCommentsLoadsImageAttachments(t *testing.T) {
	now := time.Date(2026, 6, 2, 15, 0, 0, 0, time.UTC)
	post := mustPost(t, now)
	authorID := userdomain.NewGeneratedUserID()
	comment := mustComment(t, post.ID(), authorID, nil, "Body", now)
	attachmentID := mediadomain.NewGeneratedAttachmentID()
	comments := &fakeCommentRepository{
		listVisibleByPostFunc: func(ctx context.Context, postID postdomain.PostID, limit int, offset int) ([]commentdomain.Comment, error) {
			return []commentdomain.Comment{*comment}, nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	attachments := &fakeAttachmentRepository{
		listReadyImagesByCommentIDsFunc: func(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID][]mediadomain.Attachment, error) {
			if len(commentIDs) != 1 || commentIDs[0] != comment.ID() {
				t.Fatalf("unexpected comment ids: %#v", commentIDs)
			}
			return map[commentdomain.CommentID][]mediadomain.Attachment{
				comment.ID(): []mediadomain.Attachment{*mustMediaAttachment(t, attachmentID, mediadomain.OwnerTypeComment, comment.ID().String(), authorID, now)},
			}, nil
		},
	}
	uc := NewCommentUseCaseWithAttachments(comments, posts, attachments, 1, func() time.Time { return now })

	result, err := uc.ListPostComments(context.Background(), ListPostCommentsInput{
		PostID: post.ID().String(),
	})
	if err != nil {
		t.Fatalf("ListPostComments returned error: %v", err)
	}
	if len(result.Comments) != 1 || len(result.Comments[0].Attachments) != 1 {
		t.Fatalf("expected comment attachment, got %#v", result.Comments)
	}
	if result.Comments[0].Attachments[0].URL != "https://assets.example.com/comment.png" {
		t.Fatalf("unexpected attachment url: %#v", result.Comments[0].Attachments[0])
	}
}

func TestListPostCommentsBuildsTreePreorder(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	post := mustPost(t, now)
	authorID := userdomain.NewGeneratedUserID()
	rootOlder := mustComment(t, post.ID(), authorID, nil, "Older root", now.Add(-3*time.Minute))
	rootNewer := mustComment(t, post.ID(), authorID, nil, "Newer root", now)
	rootNewerID := rootNewer.ID()
	child := mustComment(t, post.ID(), authorID, &rootNewerID, "Child", now.Add(-time.Minute))
	childID := child.ID()
	grandchild := mustComment(t, post.ID(), authorID, &childID, "Grandchild", now.Add(-2*time.Minute))

	comments := &fakeCommentRepository{
		listVisibleTreeByPostFunc: func(ctx context.Context, postID postdomain.PostID) ([]commentdomain.Comment, error) {
			if postID != post.ID() {
				t.Fatalf("expected post %q, got %q", post.ID().String(), postID.String())
			}
			return []commentdomain.Comment{*rootOlder, *grandchild, *child, *rootNewer}, nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, time.Now)

	result, err := uc.ListPostComments(context.Background(), ListPostCommentsInput{
		PostID:   post.ID().String(),
		View:     "tree",
		Sort:     "new",
		Limit:    1,
		MaxDepth: 1,
	})
	if err != nil {
		t.Fatalf("ListPostComments returned error: %v", err)
	}
	if result.View != CommentListViewTree.String() || result.Sort != CommentListSortNew.String() || result.MaxDepth != 1 {
		t.Fatalf("unexpected result metadata: %#v", result)
	}
	if len(result.Comments) != 2 {
		t.Fatalf("expected root and child only, got %d comments: %#v", len(result.Comments), result.Comments)
	}
	if result.Comments[0].ID != rootNewer.ID().String() || result.Comments[0].Depth != 0 || result.Comments[0].ReplyCount != 1 {
		t.Fatalf("unexpected root comment: %#v", result.Comments[0])
	}
	if result.Comments[1].ID != child.ID().String() || result.Comments[1].Depth != 1 || result.Comments[1].ReplyCount != 1 || !result.Comments[1].HasMoreReplies {
		t.Fatalf("unexpected child comment: %#v", result.Comments[1])
	}
	if result.Comments[0].Format != CommentFormat || result.Comments[1].Format != CommentFormat {
		t.Fatalf("expected format %q, got %#v", CommentFormat, result.Comments)
	}
}

func TestListPostCommentsSortsTreeByTopVotes(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	post := mustPost(t, now)
	authorID := userdomain.NewGeneratedUserID()
	lowScoreRoot := mustComment(t, post.ID(), authorID, nil, "Low score", now.Add(time.Minute))
	highScoreRoot := mustComment(t, post.ID(), authorID, nil, "High score", now)

	comments := &fakeCommentRepository{
		listVisibleTreeByPostFunc: func(ctx context.Context, postID postdomain.PostID) ([]commentdomain.Comment, error) {
			return []commentdomain.Comment{*lowScoreRoot, *highScoreRoot}, nil
		},
		voteSummaries: map[commentdomain.CommentID]votedomain.CommentVoteSummary{
			lowScoreRoot.ID(): {
				CommentID:     lowScoreRoot.ID(),
				UpvoteCount:   1,
				DownvoteCount: 0,
			},
			highScoreRoot.ID(): {
				CommentID:     highScoreRoot.ID(),
				UpvoteCount:   5,
				DownvoteCount: 1,
			},
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, time.Now)

	result, err := uc.ListPostComments(context.Background(), ListPostCommentsInput{
		PostID: post.ID().String(),
		View:   "tree",
		Sort:   "top",
	})
	if err != nil {
		t.Fatalf("ListPostComments returned error: %v", err)
	}
	if len(result.Comments) != 2 {
		t.Fatalf("expected two comments, got %d", len(result.Comments))
	}
	if result.Comments[0].ID != highScoreRoot.ID().String() || result.Comments[1].ID != lowScoreRoot.ID().String() {
		t.Fatalf("expected top-vote tree order [%s %s], got %#v", highScoreRoot.ID().String(), lowScoreRoot.ID().String(), result.Comments)
	}
}

func TestListPostCommentsRejectsInvalidTreeInput(t *testing.T) {
	uc := NewCommentUseCase(&fakeCommentRepository{}, &fakePostRepository{}, time.Now)

	tests := []struct {
		name  string
		input ListPostCommentsInput
	}{
		{name: "invalid view", input: ListPostCommentsInput{PostID: postdomain.NewGeneratedPostID().String(), View: "nested"}},
		{name: "invalid sort", input: ListPostCommentsInput{PostID: postdomain.NewGeneratedPostID().String(), Sort: "popular"}},
		{name: "invalid max depth", input: ListPostCommentsInput{PostID: postdomain.NewGeneratedPostID().String(), MaxDepth: -1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.ListPostComments(context.Background(), tt.input)
			if !hasAppCode(err, apperr.CodeInvalidArgument) {
				t.Fatalf("expected invalid_argument, got %v", err)
			}
		})
	}
}

func TestUpdateCommentAllowsAuthor(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	updatedAt := now.Add(time.Minute)
	post := mustPost(t, now)
	authorID := userdomain.NewGeneratedUserID()
	comment := mustComment(t, post.ID(), authorID, nil, "Original", now)
	comments := &fakeCommentRepository{
		findVisibleByIDFunc: func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
			if id != comment.ID() {
				t.Fatalf("expected comment %q, got %q", comment.ID().String(), id.String())
			}
			return comment, nil
		},
		updateContentFunc: func(ctx context.Context, updated commentdomain.Comment) error {
			if updated.Body().String() != "Updated body" {
				t.Fatalf("expected updated body, got %q", updated.Body().String())
			}
			if !updated.UpdatedAt().Equal(updatedAt) {
				t.Fatalf("expected updated_at %s, got %s", updatedAt, updated.UpdatedAt())
			}
			return nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			if id != post.ID() {
				t.Fatalf("expected post %q, got %q", post.ID().String(), id.String())
			}
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, func() time.Time { return updatedAt })

	result, err := uc.UpdateComment(context.Background(), UpdateCommentInput{
		CommentID: comment.ID().String(),
		ActorID:   authorID,
		Body:      "Updated body",
	})
	if err != nil {
		t.Fatalf("UpdateComment returned error: %v", err)
	}
	if result.Comment.Body != "Updated body" || result.Comment.Format != CommentFormat {
		t.Fatalf("unexpected updated comment result: %#v", result.Comment)
	}
}

func TestUpdateCommentNotifiesOnlyNewMentions(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 30, 0, 0, time.UTC)
	updatedAt := now.Add(time.Minute)
	post := mustPost(t, now)
	authorID := userdomain.NewGeneratedUserID()
	mentionedAlice := mustUser(t, "alice", "active", now)
	mentionedBob := mustUser(t, "bob_123", "active", now)
	comment := mustComment(t, post.ID(), authorID, nil, "Original @alice", now)
	comments := &fakeCommentRepository{
		findVisibleByIDFunc: func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
			return comment, nil
		},
	}
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
	uc := NewCommentUseCase(comments, posts, func() time.Time { return updatedAt })
	uc.SetPublicUserFinder(users)
	uc.SetNotificationPublisher(notifications)

	_, err := uc.UpdateComment(context.Background(), UpdateCommentInput{
		CommentID: comment.ID().String(),
		ActorID:   authorID,
		Body:      "Original @alice and @bob_123",
	})
	if err != nil {
		t.Fatalf("UpdateComment returned error: %v", err)
	}
	if len(notifications.mentions) != 1 {
		t.Fatalf("expected one mention notification, got %#v", notifications.mentions)
	}
	assertMentionNotification(t, notifications.mentions[0], mentionedBob.ID(), authorID, "comment", comment.ID().String())
}

func TestUpdateCommentReplacesImageAttachments(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	updatedAt := now.Add(time.Minute)
	post := mustPost(t, now)
	authorID := userdomain.NewGeneratedUserID()
	attachmentID := mediadomain.NewGeneratedAttachmentID()
	comment := mustComment(t, post.ID(), authorID, nil, "Original", now)
	comments := &fakeCommentRepository{
		findVisibleByIDFunc: func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
			return comment, nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	attachments := &fakeAttachmentRepository{
		replaceReadyImagesForCommentFunc: func(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, replaceTime time.Time) ([]mediadomain.Attachment, error) {
			if commentID != comment.ID() {
				t.Fatalf("expected comment %q, got %q", comment.ID().String(), commentID.String())
			}
			if uploaderID != authorID {
				t.Fatalf("expected uploader %q, got %q", authorID.String(), uploaderID.String())
			}
			if len(attachmentIDs) != 1 || attachmentIDs[0] != attachmentID {
				t.Fatalf("unexpected attachment ids: %#v", attachmentIDs)
			}
			if maxCount != 1 || !replaceTime.Equal(updatedAt) {
				t.Fatalf("unexpected replace metadata: max=%d time=%s", maxCount, replaceTime)
			}
			return []mediadomain.Attachment{*mustMediaAttachment(t, attachmentID, mediadomain.OwnerTypeComment, comment.ID().String(), authorID, updatedAt)}, nil
		},
	}
	uc := NewCommentUseCaseWithAttachments(comments, posts, attachments, 1, func() time.Time { return updatedAt })
	rawAttachmentIDs := []string{attachmentID.String()}

	result, err := uc.UpdateComment(context.Background(), UpdateCommentInput{
		CommentID:     comment.ID().String(),
		ActorID:       authorID,
		Body:          "Updated body",
		AttachmentIDs: &rawAttachmentIDs,
	})
	if err != nil {
		t.Fatalf("UpdateComment returned error: %v", err)
	}
	if len(result.Comment.Attachments) != 1 || result.Comment.Attachments[0].ID != attachmentID.String() {
		t.Fatalf("expected replacement attachment in result, got %#v", result.Comment.Attachments)
	}
	if !attachments.replaceCalled {
		t.Fatal("expected replacement attachment repository call")
	}
}

func TestUpdateCommentPreservesContentRefsWhenOmitted(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 10, 0, 0, time.UTC)
	updatedAt := now.Add(time.Minute)
	post := mustPost(t, now)
	authorID := userdomain.NewGeneratedUserID()
	comment := mustComment(t, post.ID(), authorID, nil, "Original", now)
	existingRefs := []postusecase.ContentRef{
		{Kind: postusecase.ContentRefKindLink, RefID: "https://example.com/original-comment"},
		{Kind: postusecase.ContentRefKindEmbed, RefID: "2ac24321-6509-42f0-aa2a-a0d708699c95"},
	}
	comments := &fakeCommentRepository{
		findVisibleByIDFunc: func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
			return comment, nil
		},
		listContentRefsFunc: func(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID][]postusecase.ContentRef, error) {
			if len(commentIDs) != 1 || commentIDs[0] != comment.ID() {
				t.Fatalf("unexpected comment ids: %#v", commentIDs)
			}
			return map[commentdomain.CommentID][]postusecase.ContentRef{comment.ID(): existingRefs}, nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, func() time.Time { return updatedAt })

	result, err := uc.UpdateComment(context.Background(), UpdateCommentInput{
		CommentID: comment.ID().String(),
		ActorID:   authorID,
		Body:      "Updated body",
	})
	if err != nil {
		t.Fatalf("UpdateComment returned error: %v", err)
	}
	if !comments.listContentRefsCalled {
		t.Fatal("expected existing content refs to be loaded")
	}
	if comments.replaceContentRefsCalled {
		t.Fatal("did not expect omitted content_refs to replace existing refs")
	}
	assertCommentContentRefs(t, result.Comment.ContentRefs, existingRefs)
}

func TestUpdateCommentClearsContentRefsWithEmptyArray(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 20, 0, 0, time.UTC)
	updatedAt := now.Add(time.Minute)
	post := mustPost(t, now)
	authorID := userdomain.NewGeneratedUserID()
	comment := mustComment(t, post.ID(), authorID, nil, "Original", now)
	comments := &fakeCommentRepository{
		findVisibleByIDFunc: func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
			return comment, nil
		},
		replaceContentRefsFunc: func(ctx context.Context, commentID commentdomain.CommentID, refs []postusecase.ContentRef, replacedAt time.Time) error {
			if commentID != comment.ID() {
				t.Fatalf("expected comment %q, got %q", comment.ID().String(), commentID.String())
			}
			if len(refs) != 0 {
				t.Fatalf("expected content refs to be cleared, got %#v", refs)
			}
			return nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, func() time.Time { return updatedAt })
	contentRefs := []postusecase.ContentRefInput{}

	result, err := uc.UpdateComment(context.Background(), UpdateCommentInput{
		CommentID:   comment.ID().String(),
		ActorID:     authorID,
		Body:        "Updated body",
		ContentRefs: &contentRefs,
	})
	if err != nil {
		t.Fatalf("UpdateComment returned error: %v", err)
	}
	if !comments.replaceContentRefsCalled {
		t.Fatal("expected empty content_refs to replace stored refs")
	}
	if len(result.Comment.ContentRefs) != 0 {
		t.Fatalf("expected empty content refs in result, got %#v", result.Comment.ContentRefs)
	}
}

func TestUpdateCommentRejectsNonAuthor(t *testing.T) {
	now := time.Now().UTC()
	post := mustPost(t, now)
	comment := mustComment(t, post.ID(), userdomain.NewGeneratedUserID(), nil, "Original", now)
	comments := &fakeCommentRepository{
		findVisibleByIDFunc: func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
			return comment, nil
		},
		updateContentFunc: func(ctx context.Context, updated commentdomain.Comment) error {
			t.Fatal("UpdateContent should not be called for non-author")
			return nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, time.Now)

	_, err := uc.UpdateComment(context.Background(), UpdateCommentInput{
		CommentID: comment.ID().String(),
		ActorID:   userdomain.NewGeneratedUserID(),
		Body:      "Updated body",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for non-author, got %v", err)
	}
}

func TestDeleteCommentMarksAuthorCommentDeleted(t *testing.T) {
	now := time.Date(2026, 6, 3, 13, 0, 0, 0, time.UTC)
	deletedAt := now.Add(time.Minute)
	post := mustPost(t, now)
	authorID := userdomain.NewGeneratedUserID()
	comment := mustComment(t, post.ID(), authorID, nil, "Original", now)
	comments := &fakeCommentRepository{
		findVisibleByIDFunc: func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
			return comment, nil
		},
		markDeletedFunc: func(ctx context.Context, deleted commentdomain.Comment) error {
			if deleted.Status() != commentdomain.CommentStatusDeleted {
				t.Fatalf("expected deleted status, got %q", deleted.Status().String())
			}
			if !deleted.UpdatedAt().Equal(deletedAt) {
				t.Fatalf("expected deleted_at %s, got %s", deletedAt, deleted.UpdatedAt())
			}
			return nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, func() time.Time { return deletedAt })

	if _, err := uc.DeleteComment(context.Background(), DeleteCommentInput{CommentID: comment.ID().String(), ActorID: authorID}); err != nil {
		t.Fatalf("DeleteComment returned error: %v", err)
	}
}

func TestDeleteCommentRejectsInvalidInput(t *testing.T) {
	uc := NewCommentUseCase(&fakeCommentRepository{}, &fakePostRepository{}, time.Now)

	_, err := uc.DeleteComment(context.Background(), DeleteCommentInput{
		CommentID: commentdomain.NewGeneratedCommentID().String(),
		ActorID:   "",
	})
	if !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated for missing actor, got %v", err)
	}

	_, err = uc.DeleteComment(context.Background(), DeleteCommentInput{
		CommentID: "not-a-uuid",
		ActorID:   userdomain.NewGeneratedUserID(),
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid comment id, got %v", err)
	}
}

func TestSetCommentVoteChecksVisibleCommentAndPost(t *testing.T) {
	now := time.Date(2026, 6, 6, 11, 0, 0, 0, time.UTC)
	post := mustPost(t, now)
	comment := mustComment(t, post.ID(), userdomain.NewGeneratedUserID(), nil, "Body", now)
	userID := userdomain.NewGeneratedUserID()
	comments := &fakeCommentRepository{
		findVisibleByIDFunc: func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
			if id != comment.ID() {
				t.Fatalf("expected comment %q, got %q", comment.ID().String(), id.String())
			}
			return comment, nil
		},
		upsertVoteFunc: func(ctx context.Context, vote votedomain.CommentVote) error {
			if vote.CommentID() != comment.ID() || vote.UserID() != userID || vote.Value() != votedomain.VoteValueUp {
				t.Fatalf("unexpected vote: %#v", vote)
			}
			if !vote.CreatedAt().Equal(now) || !vote.UpdatedAt().Equal(now) {
				t.Fatalf("unexpected vote timestamps: created=%s updated=%s", vote.CreatedAt(), vote.UpdatedAt())
			}
			return nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			if id != post.ID() {
				t.Fatalf("expected post %q, got %q", post.ID().String(), id.String())
			}
			return post, nil
		},
	}
	uc := NewCommentUseCase(comments, posts, func() time.Time { return now })

	result, err := uc.SetCommentVote(context.Background(), SetCommentVoteInput{
		CommentID: comment.ID().String(),
		UserID:    userID,
		Value:     1,
	})
	if err != nil {
		t.Fatalf("SetCommentVote returned error: %v", err)
	}
	if result.Vote.CommentID != comment.ID().String() || result.Vote.Value != 1 {
		t.Fatalf("unexpected vote result: %#v", result.Vote)
	}
	if !comments.upsertVoteCalled {
		t.Fatal("expected upsert vote repository call")
	}
}

func TestSetCommentVoteNotifiesCommentAuthorOnFirstUpvote(t *testing.T) {
	now := time.Date(2026, 6, 6, 11, 5, 0, 0, time.UTC)
	post := mustPost(t, now)
	commentAuthorID := userdomain.NewGeneratedUserID()
	comment := mustComment(t, post.ID(), commentAuthorID, nil, "Body", now)
	userID := userdomain.NewGeneratedUserID()
	comments := &fakeCommentRepository{
		findVisibleByIDFunc: func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
			return comment, nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	notifications := &fakeNotificationPublisher{}
	uc := NewCommentUseCase(comments, posts, func() time.Time { return now })
	uc.SetNotificationPublisher(notifications)

	_, err := uc.SetCommentVote(context.Background(), SetCommentVoteInput{
		CommentID: comment.ID().String(),
		UserID:    userID,
		Value:     1,
	})
	if err != nil {
		t.Fatalf("SetCommentVote returned error: %v", err)
	}
	if !notifications.commentUpvotedCalled {
		t.Fatal("expected comment upvote notification")
	}
	if notifications.recipientID != commentAuthorID || notifications.actorID != userID || notifications.commentID != comment.ID().String() {
		t.Fatalf("unexpected notification args: %#v", notifications)
	}
}

func TestSetCommentVoteDoesNotNotifyRepeatedUpvote(t *testing.T) {
	now := time.Date(2026, 6, 6, 11, 10, 0, 0, time.UTC)
	post := mustPost(t, now)
	comment := mustComment(t, post.ID(), userdomain.NewGeneratedUserID(), nil, "Body", now)
	userID := userdomain.NewGeneratedUserID()
	comments := &fakeCommentRepository{
		findVisibleByIDFunc: func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
			return comment, nil
		},
		myVotes: map[commentdomain.CommentID]votedomain.VoteValue{
			comment.ID(): votedomain.VoteValueUp,
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			return post, nil
		},
	}
	notifications := &fakeNotificationPublisher{}
	uc := NewCommentUseCase(comments, posts, func() time.Time { return now })
	uc.SetNotificationPublisher(notifications)

	_, err := uc.SetCommentVote(context.Background(), SetCommentVoteInput{
		CommentID: comment.ID().String(),
		UserID:    userID,
		Value:     1,
	})
	if err != nil {
		t.Fatalf("SetCommentVote returned error: %v", err)
	}
	if notifications.commentUpvotedCalled {
		t.Fatal("did not expect repeated comment upvote notification")
	}
}

type fakeCommentRepository struct {
	createFunc                func(ctx context.Context, comment commentdomain.Comment) error
	findVisibleByIDFunc       func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error)
	updateContentFunc         func(ctx context.Context, comment commentdomain.Comment) error
	markDeletedFunc           func(ctx context.Context, comment commentdomain.Comment) error
	listVisibleByPostFunc     func(ctx context.Context, postID postdomain.PostID, limit int, offset int) ([]commentdomain.Comment, error)
	listVisibleTreeByPostFunc func(ctx context.Context, postID postdomain.PostID) ([]commentdomain.Comment, error)
	listVisibleByAuthorFunc   func(ctx context.Context, authorID userdomain.UserID, limit int, offset int) ([]commentdomain.Comment, error)
	listVisibleSort           CommentListSort
	upsertVoteFunc            func(ctx context.Context, vote votedomain.CommentVote) error
	deleteVoteFunc            func(ctx context.Context, commentID commentdomain.CommentID, userID userdomain.UserID) error
	replaceContentRefsFunc    func(ctx context.Context, commentID commentdomain.CommentID, refs []postusecase.ContentRef, now time.Time) error
	listContentRefsFunc       func(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID][]postusecase.ContentRef, error)
	upsertVoteCalled          bool
	deleteVoteCalled          bool
	summarizeVotesCalled      bool
	findVotesCalled           bool
	listCommentEffectsCalled  bool
	replaceContentRefsCalled  bool
	listContentRefsCalled     bool
	voteSummaries             map[commentdomain.CommentID]votedomain.CommentVoteSummary
	myVotes                   map[commentdomain.CommentID]votedomain.VoteValue
	commentEffects            map[commentdomain.CommentID][]CommentEffectSummary
}

type fakeAttachmentRepository struct {
	bindReadyImagesToCommentFunc     func(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error)
	replaceReadyImagesForCommentFunc func(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error)
	listReadyImagesByCommentIDsFunc  func(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID][]mediadomain.Attachment, error)
	replaceCalled                    bool
}

func (f *fakeAttachmentRepository) BindReadyImagesToComment(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error) {
	if f.bindReadyImagesToCommentFunc != nil {
		return f.bindReadyImagesToCommentFunc(ctx, commentID, uploaderID, attachmentIDs, maxCount, now)
	}
	return nil, nil
}

func (f *fakeAttachmentRepository) ReplaceReadyImagesForComment(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error) {
	f.replaceCalled = true
	if f.replaceReadyImagesForCommentFunc != nil {
		return f.replaceReadyImagesForCommentFunc(ctx, commentID, uploaderID, attachmentIDs, maxCount, now)
	}
	return nil, nil
}

func (f *fakeAttachmentRepository) ListReadyImagesByCommentIDs(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID][]mediadomain.Attachment, error) {
	if f.listReadyImagesByCommentIDsFunc != nil {
		return f.listReadyImagesByCommentIDsFunc(ctx, commentIDs)
	}
	return nil, nil
}

func (f *fakeCommentRepository) Create(ctx context.Context, comment commentdomain.Comment) error {
	if f.createFunc != nil {
		return f.createFunc(ctx, comment)
	}
	return nil
}

func (f *fakeCommentRepository) FindVisibleByID(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
	if f.findVisibleByIDFunc != nil {
		return f.findVisibleByIDFunc(ctx, id)
	}
	return nil, apperr.New(apperr.CodeNotFound, "comment not found")
}

func (f *fakeCommentRepository) UpdateContent(ctx context.Context, comment commentdomain.Comment) error {
	if f.updateContentFunc != nil {
		return f.updateContentFunc(ctx, comment)
	}
	return nil
}

func (f *fakeCommentRepository) MarkDeleted(ctx context.Context, comment commentdomain.Comment) error {
	if f.markDeletedFunc != nil {
		return f.markDeletedFunc(ctx, comment)
	}
	return nil
}

func (f *fakeCommentRepository) ListVisibleByPost(ctx context.Context, postID postdomain.PostID, sort CommentListSort, limit int, offset int) ([]commentdomain.Comment, error) {
	f.listVisibleSort = sort
	if f.listVisibleByPostFunc != nil {
		return f.listVisibleByPostFunc(ctx, postID, limit, offset)
	}
	return nil, nil
}

func (f *fakeCommentRepository) ListVisibleTreeByPost(ctx context.Context, postID postdomain.PostID) ([]commentdomain.Comment, error) {
	if f.listVisibleTreeByPostFunc != nil {
		return f.listVisibleTreeByPostFunc(ctx, postID)
	}
	return nil, nil
}

func (f *fakeCommentRepository) ListVisibleByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID, limit int, offset int) ([]commentdomain.Comment, error) {
	if f.listVisibleByAuthorFunc != nil {
		return f.listVisibleByAuthorFunc(ctx, authorID, limit, offset)
	}
	return nil, nil
}

func (f *fakeCommentRepository) UpsertCommentVote(ctx context.Context, vote votedomain.CommentVote) error {
	f.upsertVoteCalled = true
	if f.upsertVoteFunc != nil {
		return f.upsertVoteFunc(ctx, vote)
	}
	return nil
}

func (f *fakeCommentRepository) DeleteCommentVote(ctx context.Context, commentID commentdomain.CommentID, userID userdomain.UserID) error {
	f.deleteVoteCalled = true
	if f.deleteVoteFunc != nil {
		return f.deleteVoteFunc(ctx, commentID, userID)
	}
	return nil
}

func (f *fakeCommentRepository) FindCommentVotesByIDsAndUser(ctx context.Context, commentIDs []commentdomain.CommentID, userID userdomain.UserID) (map[commentdomain.CommentID]votedomain.VoteValue, error) {
	f.findVotesCalled = true
	if f.myVotes == nil {
		return map[commentdomain.CommentID]votedomain.VoteValue{}, nil
	}
	return f.myVotes, nil
}

func (f *fakeCommentRepository) SummarizeCommentVotesByIDs(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID]votedomain.CommentVoteSummary, error) {
	f.summarizeVotesCalled = true
	if f.voteSummaries == nil {
		return map[commentdomain.CommentID]votedomain.CommentVoteSummary{}, nil
	}
	return f.voteSummaries, nil
}

func (f *fakeCommentRepository) ListCommentEffectsByCommentIDs(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID][]CommentEffectSummary, error) {
	f.listCommentEffectsCalled = true
	if f.commentEffects == nil {
		return map[commentdomain.CommentID][]CommentEffectSummary{}, nil
	}
	return f.commentEffects, nil
}

func (f *fakeCommentRepository) ReplaceCommentContentRefs(ctx context.Context, commentID commentdomain.CommentID, refs []postusecase.ContentRef, now time.Time) error {
	f.replaceContentRefsCalled = true
	if f.replaceContentRefsFunc != nil {
		return f.replaceContentRefsFunc(ctx, commentID, refs, now)
	}
	return nil
}

func (f *fakeCommentRepository) ListCommentContentRefsByCommentIDs(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID][]postusecase.ContentRef, error) {
	f.listContentRefsCalled = true
	if f.listContentRefsFunc != nil {
		return f.listContentRefsFunc(ctx, commentIDs)
	}
	return map[commentdomain.CommentID][]postusecase.ContentRef{}, nil
}

type fakeNotificationPublisher struct {
	postCommentedCalled  bool
	commentRepliedCalled bool
	commentUpvotedCalled bool
	recipientID          userdomain.UserID
	actorID              userdomain.UserID
	postID               string
	commentID            string
	mentions             []fakeMentionNotification
}

type fakeMentionNotification struct {
	recipientID userdomain.UserID
	actorID     userdomain.UserID
	sourceType  string
	sourceID    string
}

func (f *fakeNotificationPublisher) NotifyPostCommented(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, postID string) error {
	f.postCommentedCalled = true
	f.recipientID = recipientID
	f.actorID = actorID
	f.postID = postID
	return nil
}

func (f *fakeNotificationPublisher) NotifyCommentReplied(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, commentID string) error {
	f.commentRepliedCalled = true
	f.recipientID = recipientID
	f.actorID = actorID
	f.commentID = commentID
	return nil
}

func (f *fakeNotificationPublisher) NotifyCommentUpvoted(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, commentID string) error {
	f.commentUpvotedCalled = true
	f.recipientID = recipientID
	f.actorID = actorID
	f.commentID = commentID
	return nil
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

type fakePostRepository struct {
	findVisibleByIDFunc func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error)
}

func (f *fakePostRepository) Create(ctx context.Context, post postdomain.Post) error {
	return nil
}

func (f *fakePostRepository) FindVisibleByID(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
	if f.findVisibleByIDFunc != nil {
		return f.findVisibleByIDFunc(ctx, id)
	}
	return nil, apperr.New(apperr.CodeNotFound, "post not found")
}

func (f *fakePostRepository) UpdateContent(ctx context.Context, post postdomain.Post) error {
	return nil
}

func (f *fakePostRepository) MarkDeleted(ctx context.Context, post postdomain.Post) error {
	return nil
}

func (f *fakePostRepository) ListVisibleByCommunity(ctx context.Context, communityID communitydomain.CommunityID, sort postusecase.PostListSort, limit int, offset int) ([]postdomain.Post, error) {
	return nil, nil
}

func (f *fakePostRepository) ListVisibleInPublicCommunities(ctx context.Context, sort postusecase.PostListSort, limit int, offset int) ([]postdomain.Post, error) {
	return nil, nil
}

func mustPost(t *testing.T, now time.Time) *postdomain.Post {
	t.Helper()

	title, err := postdomain.NewPostTitle("Post")
	if err != nil {
		t.Fatalf("NewPostTitle returned error: %v", err)
	}
	body, err := postdomain.NewPostBody("Post body")
	if err != nil {
		t.Fatalf("NewPostBody returned error: %v", err)
	}
	post, err := postdomain.NewPost(postdomain.NewGeneratedPostID(), communitydomain.NewGeneratedCommunityID(), userdomain.NewGeneratedUserID(), title, body, now)
	if err != nil {
		t.Fatalf("NewPost returned error: %v", err)
	}
	return post
}

func mustComment(t *testing.T, postID postdomain.PostID, authorID userdomain.UserID, parentID *commentdomain.CommentID, body string, now time.Time) *commentdomain.Comment {
	t.Helper()

	commentBody, err := commentdomain.NewCommentBody(body)
	if err != nil {
		t.Fatalf("NewCommentBody returned error: %v", err)
	}
	comment, err := commentdomain.NewComment(commentdomain.NewGeneratedCommentID(), postID, authorID, parentID, commentBody, now)
	if err != nil {
		t.Fatalf("NewComment returned error: %v", err)
	}
	return comment
}

func mustMediaAttachment(t *testing.T, id mediadomain.AttachmentID, ownerType mediadomain.OwnerType, ownerID string, uploaderID userdomain.UserID, now time.Time) *mediadomain.Attachment {
	t.Helper()

	attachment, err := mediadomain.RehydrateAttachment(mediadomain.NewAttachmentParams{
		ID:              id,
		OwnerType:       ownerType,
		OwnerID:         ownerID,
		UploaderID:      uploaderID,
		Kind:            mediadomain.AttachmentKindImage,
		StorageProvider: mediadomain.StorageProviderR2,
		Bucket:          "cumt-nexus",
		ObjectKey:       "images/comment.png",
		PublicURL:       "https://assets.example.com/comment.png",
		SizeBytes:       100,
		MimeType:        "image/png",
		AltText:         "Campus",
		Status:          mediadomain.AttachmentStatusReady,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("RehydrateAttachment returned error: %v", err)
	}
	return attachment
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

func assertCommentContentRefs(t *testing.T, got []postusecase.ContentRef, want []postusecase.ContentRef) {
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

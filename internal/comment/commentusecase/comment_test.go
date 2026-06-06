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
	if gotLimit != MaxCommentListLimit || result.Limit != MaxCommentListLimit {
		t.Fatalf("expected clamped limit %d, got repo=%d result=%d", MaxCommentListLimit, gotLimit, result.Limit)
	}
	if gotOffset != 5 || result.Offset != 5 {
		t.Fatalf("expected offset 5, got repo=%d result=%d", gotOffset, result.Offset)
	}
	if len(result.Comments) != 1 {
		t.Fatalf("expected one comment, got %d", len(result.Comments))
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

func TestListPostCommentsRejectsInvalidTreeInput(t *testing.T) {
	uc := NewCommentUseCase(&fakeCommentRepository{}, &fakePostRepository{}, time.Now)

	tests := []struct {
		name  string
		input ListPostCommentsInput
	}{
		{name: "invalid view", input: ListPostCommentsInput{PostID: postdomain.NewGeneratedPostID().String(), View: "nested"}},
		{name: "invalid sort", input: ListPostCommentsInput{PostID: postdomain.NewGeneratedPostID().String(), Sort: "old"}},
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

type fakeCommentRepository struct {
	createFunc                func(ctx context.Context, comment commentdomain.Comment) error
	findVisibleByIDFunc       func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error)
	updateContentFunc         func(ctx context.Context, comment commentdomain.Comment) error
	markDeletedFunc           func(ctx context.Context, comment commentdomain.Comment) error
	listVisibleByPostFunc     func(ctx context.Context, postID postdomain.PostID, limit int, offset int) ([]commentdomain.Comment, error)
	listVisibleTreeByPostFunc func(ctx context.Context, postID postdomain.PostID) ([]commentdomain.Comment, error)
	listVisibleByAuthorFunc   func(ctx context.Context, authorID userdomain.UserID, limit int, offset int) ([]commentdomain.Comment, error)
	upsertVoteFunc            func(ctx context.Context, vote votedomain.CommentVote) error
	deleteVoteFunc            func(ctx context.Context, commentID commentdomain.CommentID, userID userdomain.UserID) error
	upsertVoteCalled          bool
	deleteVoteCalled          bool
	summarizeVotesCalled      bool
	findVotesCalled           bool
	voteSummaries             map[commentdomain.CommentID]votedomain.CommentVoteSummary
	myVotes                   map[commentdomain.CommentID]votedomain.VoteValue
}

type fakeAttachmentRepository struct {
	bindReadyImagesToCommentFunc    func(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error)
	listReadyImagesByCommentIDsFunc func(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID][]mediadomain.Attachment, error)
}

func (f *fakeAttachmentRepository) BindReadyImagesToComment(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error) {
	if f.bindReadyImagesToCommentFunc != nil {
		return f.bindReadyImagesToCommentFunc(ctx, commentID, uploaderID, attachmentIDs, maxCount, now)
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

func (f *fakeCommentRepository) ListVisibleByPost(ctx context.Context, postID postdomain.PostID, limit int, offset int) ([]commentdomain.Comment, error) {
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

package commentusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
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

type fakeCommentRepository struct {
	createFunc            func(ctx context.Context, comment commentdomain.Comment) error
	findVisibleByIDFunc   func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error)
	listVisibleByPostFunc func(ctx context.Context, postID postdomain.PostID, limit int, offset int) ([]commentdomain.Comment, error)
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

func (f *fakeCommentRepository) ListVisibleByPost(ctx context.Context, postID postdomain.PostID, limit int, offset int) ([]commentdomain.Comment, error) {
	if f.listVisibleByPostFunc != nil {
		return f.listVisibleByPostFunc(ctx, postID, limit, offset)
	}
	return nil, nil
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

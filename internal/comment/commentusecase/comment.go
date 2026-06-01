package commentusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

const (
	DefaultCommentListLimit = 20
	MaxCommentListLimit     = 50
)

type CommentUseCase struct {
	comments CommentRepository
	posts    postusecase.PostRepository
	now      func() time.Time
}

type PublishCommentInput struct {
	PostID   string
	AuthorID userdomain.UserID
	ParentID string
	Body     string
}

type ListPostCommentsInput struct {
	PostID string
	Limit  int
	Offset int
}

type PublishCommentResult struct {
	Comment Comment
}

type ListPostCommentsResult struct {
	Comments []Comment
	Limit    int
	Offset   int
}

type Comment struct {
	ID        string
	PostID    string
	AuthorID  string
	ParentID  string
	Body      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewCommentUseCase(comments CommentRepository, posts postusecase.PostRepository, now func() time.Time) *CommentUseCase {
	if now == nil {
		now = time.Now
	}

	return &CommentUseCase{
		comments: comments,
		posts:    posts,
		now:      now,
	}
}

func (uc *CommentUseCase) PublishComment(ctx context.Context, input PublishCommentInput) (PublishCommentResult, error) {
	if strings.TrimSpace(input.AuthorID.String()) == "" {
		return PublishCommentResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}

	postID, err := postdomain.NewPostID(input.PostID)
	if err != nil {
		return PublishCommentResult{}, err
	}
	post, err := uc.posts.FindVisibleByID(ctx, postID)
	if err != nil {
		return PublishCommentResult{}, fmt.Errorf("find post for comment: %w", err)
	}

	parentID, err := uc.resolveParentComment(ctx, post.ID(), input.ParentID)
	if err != nil {
		return PublishCommentResult{}, err
	}
	body, err := commentdomain.NewCommentBody(input.Body)
	if err != nil {
		return PublishCommentResult{}, err
	}

	now := uc.now().UTC()
	comment, err := commentdomain.NewComment(
		commentdomain.NewGeneratedCommentID(),
		post.ID(),
		input.AuthorID,
		parentID,
		body,
		now,
	)
	if err != nil {
		return PublishCommentResult{}, err
	}

	if err := uc.comments.Create(ctx, *comment); err != nil {
		return PublishCommentResult{}, fmt.Errorf("create comment: %w", err)
	}

	return PublishCommentResult{
		Comment: toCommentDTO(*comment),
	}, nil
}

func (uc *CommentUseCase) ListPostComments(ctx context.Context, input ListPostCommentsInput) (ListPostCommentsResult, error) {
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListPostCommentsResult{}, err
	}

	postID, err := postdomain.NewPostID(input.PostID)
	if err != nil {
		return ListPostCommentsResult{}, err
	}
	post, err := uc.posts.FindVisibleByID(ctx, postID)
	if err != nil {
		return ListPostCommentsResult{}, fmt.Errorf("find post for comments: %w", err)
	}

	comments, err := uc.comments.ListVisibleByPost(ctx, post.ID(), limit, offset)
	if err != nil {
		return ListPostCommentsResult{}, fmt.Errorf("list post comments: %w", err)
	}

	result := ListPostCommentsResult{
		Comments: make([]Comment, 0, len(comments)),
		Limit:    limit,
		Offset:   offset,
	}
	for _, comment := range comments {
		result.Comments = append(result.Comments, toCommentDTO(comment))
	}

	return result, nil
}

func (uc *CommentUseCase) resolveParentComment(ctx context.Context, postID postdomain.PostID, rawParentID string) (*commentdomain.CommentID, error) {
	rawParentID = strings.TrimSpace(rawParentID)
	if rawParentID == "" {
		return nil, nil
	}

	parentID, err := commentdomain.NewCommentID(rawParentID)
	if err != nil {
		return nil, err
	}
	parent, err := uc.comments.FindVisibleByID(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("find parent comment: %w", err)
	}
	if parent.PostID() != postID {
		return nil, apperr.New(apperr.CodeInvalidArgument, "parent comment does not belong to post")
	}

	return &parentID, nil
}

func normalizePagination(limit int, offset int) (int, int, error) {
	if limit < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "comment list limit is invalid")
	}
	if offset < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "comment list offset is invalid")
	}
	if limit == 0 {
		limit = DefaultCommentListLimit
	}
	if limit > MaxCommentListLimit {
		limit = MaxCommentListLimit
	}

	return limit, offset, nil
}

func toCommentDTO(comment commentdomain.Comment) Comment {
	parentID, hasParentID := comment.ParentID()
	dto := Comment{
		ID:        comment.ID().String(),
		PostID:    comment.PostID().String(),
		AuthorID:  comment.AuthorID().String(),
		Body:      comment.Body().String(),
		Status:    comment.Status().String(),
		CreatedAt: comment.CreatedAt(),
		UpdatedAt: comment.UpdatedAt(),
	}
	if hasParentID {
		dto.ParentID = parentID.String()
	}
	return dto
}

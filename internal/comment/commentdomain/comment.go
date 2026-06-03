package commentdomain

import (
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

type CommentID string

func NewCommentID(raw string) (CommentID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "comment id is required")
	}

	parsed, err := uuid.Parse(raw)
	if err != nil {
		return "", apperr.New(apperr.CodeInvalidArgument, "comment id is invalid")
	}

	return CommentID(parsed.String()), nil
}

func NewGeneratedCommentID() CommentID {
	return CommentID(uuid.NewString())
}

func (id CommentID) String() string {
	return string(id)
}

type CommentBody string

func NewCommentBody(raw string) (CommentBody, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "comment body is required")
	}

	return CommentBody(raw), nil
}

func (body CommentBody) String() string {
	return string(body)
}

type CommentStatus string

const (
	CommentStatusVisible CommentStatus = "visible"
	CommentStatusRemoved CommentStatus = "removed"
	CommentStatusDeleted CommentStatus = "deleted"
	CommentStatusLocked  CommentStatus = "locked"
	CommentStatusHidden  CommentStatus = "hidden"
)

func NewCommentStatus(raw string) (CommentStatus, error) {
	switch CommentStatus(strings.TrimSpace(strings.ToLower(raw))) {
	case CommentStatusVisible:
		return CommentStatusVisible, nil
	case CommentStatusRemoved:
		return CommentStatusRemoved, nil
	case CommentStatusDeleted:
		return CommentStatusDeleted, nil
	case CommentStatusLocked:
		return CommentStatusLocked, nil
	case CommentStatusHidden:
		return CommentStatusHidden, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "comment status is invalid")
	}
}

func (status CommentStatus) String() string {
	return string(status)
}

type Comment struct {
	id        CommentID
	postID    postdomain.PostID
	authorID  userdomain.UserID
	parentID  *CommentID
	body      CommentBody
	status    CommentStatus
	createdAt time.Time
	updatedAt time.Time
}

func NewComment(id CommentID, postID postdomain.PostID, authorID userdomain.UserID, parentID *CommentID, body CommentBody, now time.Time) (*Comment, error) {
	return RehydrateComment(id, postID, authorID, parentID, body, CommentStatusVisible, now, now)
}

func RehydrateComment(
	id CommentID,
	postID postdomain.PostID,
	authorID userdomain.UserID,
	parentID *CommentID,
	body CommentBody,
	status CommentStatus,
	createdAt time.Time,
	updatedAt time.Time,
) (*Comment, error) {
	if strings.TrimSpace(id.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "comment id is required")
	}
	if strings.TrimSpace(postID.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "comment post id is required")
	}
	if strings.TrimSpace(authorID.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "comment author id is required")
	}
	if parentID != nil && *parentID == id {
		return nil, apperr.New(apperr.CodeInvalidArgument, "comment parent id can't equal comment id")
	}
	if strings.TrimSpace(body.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "comment body is required")
	}
	if _, err := NewCommentStatus(status.String()); err != nil {
		return nil, err
	}
	if createdAt.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "comment created time can't be zero")
	}
	if updatedAt.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "comment updated time can't be zero")
	}
	if updatedAt.Before(createdAt) {
		return nil, apperr.New(apperr.CodeInvalidArgument, "comment updated time can't be before created time")
	}

	return &Comment{
		id:        id,
		postID:    postID,
		authorID:  authorID,
		parentID:  cloneOptionalCommentID(parentID),
		body:      body,
		status:    status,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}, nil
}

func (comment *Comment) ID() CommentID {
	return comment.id
}

func (comment *Comment) PostID() postdomain.PostID {
	return comment.postID
}

func (comment *Comment) AuthorID() userdomain.UserID {
	return comment.authorID
}

func (comment *Comment) ParentID() (CommentID, bool) {
	if comment.parentID == nil {
		return "", false
	}
	return *comment.parentID, true
}

func (comment *Comment) Body() CommentBody {
	return comment.body
}

func (comment *Comment) Status() CommentStatus {
	return comment.status
}

func (comment *Comment) CreatedAt() time.Time {
	return comment.createdAt
}

func (comment *Comment) UpdatedAt() time.Time {
	return comment.updatedAt
}

func (comment *Comment) EditBody(body CommentBody, now time.Time) error {
	if comment.status != CommentStatusVisible {
		return apperr.New(apperr.CodeConflict, "comment is not editable")
	}
	if now.IsZero() {
		return apperr.New(apperr.CodeInvalidArgument, "comment updated time can't be zero")
	}
	if now.Before(comment.createdAt) {
		return apperr.New(apperr.CodeInvalidArgument, "comment updated time can't be before created time")
	}

	comment.body = body
	comment.updatedAt = now
	return nil
}

func (comment *Comment) MarkDeleted(now time.Time) error {
	if comment.status != CommentStatusVisible {
		return apperr.New(apperr.CodeConflict, "comment is not deletable")
	}
	if now.IsZero() {
		return apperr.New(apperr.CodeInvalidArgument, "comment updated time can't be zero")
	}
	if now.Before(comment.createdAt) {
		return apperr.New(apperr.CodeInvalidArgument, "comment updated time can't be before created time")
	}

	comment.status = CommentStatusDeleted
	comment.updatedAt = now
	return nil
}

func cloneCommentID(id CommentID) *CommentID {
	copied := id
	return &copied
}

func cloneOptionalCommentID(id *CommentID) *CommentID {
	if id == nil {
		return nil
	}
	return cloneCommentID(*id)
}

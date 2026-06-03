package postdomain

import (
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

type PostID string

func NewPostID(raw string) (PostID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "post id is required")
	}

	parsed, err := uuid.Parse(raw)
	if err != nil {
		return "", apperr.New(apperr.CodeInvalidArgument, "post id is invalid")
	}

	return PostID(parsed.String()), nil
}

func NewGeneratedPostID() PostID {
	return PostID(uuid.NewString())
}

func (id PostID) String() string {
	return string(id)
}

type PostTitle string

func NewPostTitle(raw string) (PostTitle, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "post title is required")
	}

	return PostTitle(raw), nil
}

func (title PostTitle) String() string {
	return string(title)
}

type PostBody string

func NewPostBody(raw string) (PostBody, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "post body is required")
	}

	return PostBody(raw), nil
}

func (body PostBody) String() string {
	return string(body)
}

type PostStatus string

const (
	PostStatusVisible PostStatus = "visible"
	PostStatusRemoved PostStatus = "removed"
	PostStatusDeleted PostStatus = "deleted"
	PostStatusLocked  PostStatus = "locked"
	PostStatusHidden  PostStatus = "hidden"
)

func NewPostStatus(raw string) (PostStatus, error) {
	switch PostStatus(strings.TrimSpace(strings.ToLower(raw))) {
	case PostStatusVisible:
		return PostStatusVisible, nil
	case PostStatusRemoved:
		return PostStatusRemoved, nil
	case PostStatusDeleted:
		return PostStatusDeleted, nil
	case PostStatusLocked:
		return PostStatusLocked, nil
	case PostStatusHidden:
		return PostStatusHidden, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "post status is invalid")
	}
}

func (status PostStatus) String() string {
	return string(status)
}

type Post struct {
	id          PostID
	communityID communitydomain.CommunityID
	authorID    userdomain.UserID
	title       PostTitle
	body        PostBody
	status      PostStatus
	createdAt   time.Time
	updatedAt   time.Time
}

func NewPost(id PostID, communityID communitydomain.CommunityID, authorID userdomain.UserID, title PostTitle, body PostBody, now time.Time) (*Post, error) {
	return RehydratePost(id, communityID, authorID, title, body, PostStatusVisible, now, now)
}

func RehydratePost(
	id PostID,
	communityID communitydomain.CommunityID,
	authorID userdomain.UserID,
	title PostTitle,
	body PostBody,
	status PostStatus,
	createdAt time.Time,
	updatedAt time.Time,
) (*Post, error) {
	if strings.TrimSpace(id.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "post id is required")
	}
	if strings.TrimSpace(communityID.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "post community id is required")
	}
	if strings.TrimSpace(authorID.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "post author id is required")
	}
	if strings.TrimSpace(title.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "post title is required")
	}
	if strings.TrimSpace(body.String()) == "" {
		return nil, apperr.New(apperr.CodeInvalidArgument, "post body is required")
	}
	if _, err := NewPostStatus(status.String()); err != nil {
		return nil, err
	}
	if createdAt.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "post created time can't be zero")
	}
	if updatedAt.IsZero() {
		return nil, apperr.New(apperr.CodeInvalidArgument, "post updated time can't be zero")
	}
	if updatedAt.Before(createdAt) {
		return nil, apperr.New(apperr.CodeInvalidArgument, "post updated time can't be before created time")
	}

	return &Post{
		id:          id,
		communityID: communityID,
		authorID:    authorID,
		title:       title,
		body:        body,
		status:      status,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}, nil
}

func (post *Post) ID() PostID {
	return post.id
}

func (post *Post) CommunityID() communitydomain.CommunityID {
	return post.communityID
}

func (post *Post) AuthorID() userdomain.UserID {
	return post.authorID
}

func (post *Post) Title() PostTitle {
	return post.title
}

func (post *Post) Body() PostBody {
	return post.body
}

func (post *Post) Status() PostStatus {
	return post.status
}

func (post *Post) CreatedAt() time.Time {
	return post.createdAt
}

func (post *Post) UpdatedAt() time.Time {
	return post.updatedAt
}

func (post *Post) Edit(title PostTitle, body PostBody, now time.Time) error {
	if post.status != PostStatusVisible {
		return apperr.New(apperr.CodeConflict, "post is not editable")
	}
	if now.IsZero() {
		return apperr.New(apperr.CodeInvalidArgument, "post updated time can't be zero")
	}
	if now.Before(post.createdAt) {
		return apperr.New(apperr.CodeInvalidArgument, "post updated time can't be before created time")
	}

	post.title = title
	post.body = body
	post.updatedAt = now
	return nil
}

func (post *Post) MarkDeleted(now time.Time) error {
	if post.status != PostStatusVisible {
		return apperr.New(apperr.CodeConflict, "post is not deletable")
	}
	if now.IsZero() {
		return apperr.New(apperr.CodeInvalidArgument, "post updated time can't be zero")
	}
	if now.Before(post.createdAt) {
		return apperr.New(apperr.CodeInvalidArgument, "post updated time can't be before created time")
	}

	post.status = PostStatusDeleted
	post.updatedAt = now
	return nil
}

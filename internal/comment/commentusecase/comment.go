package commentusecase

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

const (
	DefaultCommentListLimit = 20
	MaxCommentListLimit     = 50
	DefaultCommentMaxDepth  = 6
	MaxCommentMaxDepth      = 10
	CommentBodyFormat       = "markdown"
)

type CommentListView string

const (
	CommentListViewFlat CommentListView = "flat"
	CommentListViewTree CommentListView = "tree"
)

type CommentListSort string

const (
	CommentListSortNew CommentListSort = "new"
)

type CommentUseCase struct {
	comments             CommentRepository
	posts                postusecase.PostRepository
	attachments          AttachmentRepository
	commentImageMaxCount int
	now                  func() time.Time
}

type PublishCommentInput struct {
	PostID        string
	AuthorID      userdomain.UserID
	ParentID      string
	Body          string
	AttachmentIDs []string
}

type ListPostCommentsInput struct {
	PostID   string
	View     string
	Sort     string
	Limit    int
	Offset   int
	MaxDepth int
}

type UpdateCommentInput struct {
	CommentID string
	ActorID   userdomain.UserID
	Body      string
}

type DeleteCommentInput struct {
	CommentID string
	ActorID   userdomain.UserID
}

type PublishCommentResult struct {
	Comment Comment
}

type ListPostCommentsResult struct {
	Comments []Comment
	View     string
	Sort     string
	Limit    int
	Offset   int
	MaxDepth int
}

type UpdateCommentResult struct {
	Comment Comment
}

type DeleteCommentResult struct{}

type Comment struct {
	ID             string
	PostID         string
	AuthorID       string
	ParentID       string
	Body           string
	BodyFormat     string
	Status         string
	Depth          int
	ReplyCount     int
	HasMoreReplies bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Attachments    []Attachment
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

func NewCommentUseCase(comments CommentRepository, posts postusecase.PostRepository, now func() time.Time) *CommentUseCase {
	if now == nil {
		now = time.Now
	}

	return &CommentUseCase{
		comments:             comments,
		posts:                posts,
		commentImageMaxCount: 1,
		now:                  now,
	}
}

func NewCommentUseCaseWithAttachments(comments CommentRepository, posts postusecase.PostRepository, attachments AttachmentRepository, commentImageMaxCount int, now func() time.Time) *CommentUseCase {
	uc := NewCommentUseCase(comments, posts, now)
	uc.attachments = attachments
	if commentImageMaxCount > 0 {
		uc.commentImageMaxCount = commentImageMaxCount
	}
	return uc
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
	attachmentIDs, err := parseAttachmentIDs(input.AttachmentIDs, uc.commentImageMaxCount)
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
	attachments, err := uc.bindCommentAttachments(ctx, comment.ID(), input.AuthorID, attachmentIDs, now)
	if err != nil {
		return PublishCommentResult{}, err
	}

	return PublishCommentResult{
		Comment: toCommentDTO(*comment, attachments),
	}, nil
}

func (uc *CommentUseCase) ListPostComments(ctx context.Context, input ListPostCommentsInput) (ListPostCommentsResult, error) {
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListPostCommentsResult{}, err
	}
	view, err := normalizeCommentListView(input.View)
	if err != nil {
		return ListPostCommentsResult{}, err
	}
	listSort, err := normalizeCommentListSort(input.Sort)
	if err != nil {
		return ListPostCommentsResult{}, err
	}
	maxDepth, err := normalizeCommentMaxDepth(input.MaxDepth)
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

	if view == CommentListViewTree {
		return uc.listPostCommentsTree(ctx, post.ID(), listSort, limit, offset, maxDepth)
	}

	comments, err := uc.comments.ListVisibleByPost(ctx, post.ID(), limit, offset)
	if err != nil {
		return ListPostCommentsResult{}, fmt.Errorf("list post comments: %w", err)
	}

	result := ListPostCommentsResult{
		Comments: make([]Comment, 0, len(comments)),
		View:     view.String(),
		Sort:     listSort.String(),
		Limit:    limit,
		Offset:   offset,
		MaxDepth: maxDepth,
	}
	for _, comment := range comments {
		result.Comments = append(result.Comments, toCommentDTO(comment, nil))
	}
	result.Comments, err = uc.attachCommentImages(ctx, result.Comments)
	if err != nil {
		return ListPostCommentsResult{}, err
	}

	return result, nil
}

func (uc *CommentUseCase) listPostCommentsTree(ctx context.Context, postID postdomain.PostID, listSort CommentListSort, limit int, offset int, maxDepth int) (ListPostCommentsResult, error) {
	comments, err := uc.comments.ListVisibleTreeByPost(ctx, postID)
	if err != nil {
		return ListPostCommentsResult{}, fmt.Errorf("list post comment tree: %w", err)
	}

	result := ListPostCommentsResult{
		Comments: buildCommentTree(comments, listSort, limit, offset, maxDepth),
		View:     CommentListViewTree.String(),
		Sort:     listSort.String(),
		Limit:    limit,
		Offset:   offset,
		MaxDepth: maxDepth,
	}
	result.Comments, err = uc.attachCommentImages(ctx, result.Comments)
	if err != nil {
		return ListPostCommentsResult{}, err
	}
	return result, nil
}

func (uc *CommentUseCase) UpdateComment(ctx context.Context, input UpdateCommentInput) (UpdateCommentResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return UpdateCommentResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}

	commentID, err := commentdomain.NewCommentID(input.CommentID)
	if err != nil {
		return UpdateCommentResult{}, err
	}
	comment, err := uc.comments.FindVisibleByID(ctx, commentID)
	if err != nil {
		return UpdateCommentResult{}, fmt.Errorf("find comment for update: %w", err)
	}
	if _, err := uc.posts.FindVisibleByID(ctx, comment.PostID()); err != nil {
		return UpdateCommentResult{}, fmt.Errorf("find post for comment update: %w", err)
	}
	if comment.AuthorID() != input.ActorID {
		return UpdateCommentResult{}, apperr.New(apperr.CodeForbidden, "only the comment author can update comment")
	}

	body, err := commentdomain.NewCommentBody(input.Body)
	if err != nil {
		return UpdateCommentResult{}, err
	}
	if err := comment.EditBody(body, uc.now().UTC()); err != nil {
		return UpdateCommentResult{}, err
	}
	if err := uc.comments.UpdateContent(ctx, *comment); err != nil {
		return UpdateCommentResult{}, fmt.Errorf("update comment content: %w", err)
	}

	result := toCommentDTO(*comment, nil)
	comments, err := uc.attachCommentImages(ctx, []Comment{result})
	if err != nil {
		return UpdateCommentResult{}, err
	}
	if len(comments) == 1 {
		result = comments[0]
	}

	return UpdateCommentResult{Comment: result}, nil
}

func (uc *CommentUseCase) DeleteComment(ctx context.Context, input DeleteCommentInput) (DeleteCommentResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return DeleteCommentResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}

	commentID, err := commentdomain.NewCommentID(input.CommentID)
	if err != nil {
		return DeleteCommentResult{}, err
	}
	comment, err := uc.comments.FindVisibleByID(ctx, commentID)
	if err != nil {
		return DeleteCommentResult{}, fmt.Errorf("find comment for delete: %w", err)
	}
	if _, err := uc.posts.FindVisibleByID(ctx, comment.PostID()); err != nil {
		return DeleteCommentResult{}, fmt.Errorf("find post for comment delete: %w", err)
	}
	if comment.AuthorID() != input.ActorID {
		return DeleteCommentResult{}, apperr.New(apperr.CodeForbidden, "only the comment author can delete comment")
	}
	if err := comment.MarkDeleted(uc.now().UTC()); err != nil {
		return DeleteCommentResult{}, err
	}
	if err := uc.comments.MarkDeleted(ctx, *comment); err != nil {
		return DeleteCommentResult{}, fmt.Errorf("delete comment: %w", err)
	}

	return DeleteCommentResult{}, nil
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

func normalizeCommentListView(raw string) (CommentListView, error) {
	view := CommentListView(strings.ToLower(strings.TrimSpace(raw)))
	if view == "" {
		return CommentListViewFlat, nil
	}
	switch view {
	case CommentListViewFlat, CommentListViewTree:
		return view, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "comment list view is invalid")
	}
}

func (view CommentListView) String() string {
	return string(view)
}

func normalizeCommentListSort(raw string) (CommentListSort, error) {
	listSort := CommentListSort(strings.ToLower(strings.TrimSpace(raw)))
	if listSort == "" {
		return CommentListSortNew, nil
	}
	switch listSort {
	case CommentListSortNew:
		return listSort, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "comment list sort is invalid")
	}
}

func (listSort CommentListSort) String() string {
	return string(listSort)
}

func normalizeCommentMaxDepth(maxDepth int) (int, error) {
	if maxDepth < 0 {
		return 0, apperr.New(apperr.CodeInvalidArgument, "comment list max_depth is invalid")
	}
	if maxDepth == 0 {
		return DefaultCommentMaxDepth, nil
	}
	if maxDepth > MaxCommentMaxDepth {
		return MaxCommentMaxDepth, nil
	}
	return maxDepth, nil
}

func parseAttachmentIDs(rawIDs []string, maxCount int) ([]mediadomain.AttachmentID, error) {
	if len(rawIDs) == 0 {
		return []mediadomain.AttachmentID{}, nil
	}
	if maxCount <= 0 || len(rawIDs) > maxCount {
		return nil, apperr.New(apperr.CodeInvalidArgument, "comment image attachment count is invalid")
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

func (uc *CommentUseCase) bindCommentAttachments(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, now time.Time) ([]mediadomain.Attachment, error) {
	if len(attachmentIDs) == 0 {
		return []mediadomain.Attachment{}, nil
	}
	if uc.attachments == nil {
		return nil, apperr.New(apperr.CodeInvalidArgument, "comment image attachments are not supported")
	}
	attachments, err := uc.attachments.BindReadyImagesToComment(ctx, commentID, uploaderID, attachmentIDs, uc.commentImageMaxCount, now)
	if err != nil {
		return nil, fmt.Errorf("bind comment image attachments: %w", err)
	}
	return attachments, nil
}

func (uc *CommentUseCase) attachCommentImages(ctx context.Context, comments []Comment) ([]Comment, error) {
	if len(comments) == 0 || uc.attachments == nil {
		return comments, nil
	}
	commentIDs := make([]commentdomain.CommentID, 0, len(comments))
	for _, comment := range comments {
		commentID, err := commentdomain.NewCommentID(comment.ID)
		if err != nil {
			return nil, err
		}
		commentIDs = append(commentIDs, commentID)
	}
	attachmentViews, err := uc.attachments.ListReadyImagesByCommentIDs(ctx, commentIDs)
	if err != nil {
		return nil, fmt.Errorf("list comment image attachments: %w", err)
	}
	for index := range comments {
		commentID, err := commentdomain.NewCommentID(comments[index].ID)
		if err != nil {
			return nil, err
		}
		comments[index].Attachments = toAttachmentDTOs(attachmentViews[commentID])
	}
	return comments, nil
}

func toCommentDTO(comment commentdomain.Comment, attachments []mediadomain.Attachment) Comment {
	dto := toCommentTreeDTO(comment, 0, 0, false)
	dto.Attachments = toAttachmentDTOs(attachments)
	return dto
}

func toCommentTreeDTO(comment commentdomain.Comment, depth int, replyCount int, hasMoreReplies bool) Comment {
	parentID, hasParentID := comment.ParentID()
	dto := Comment{
		ID:             comment.ID().String(),
		PostID:         comment.PostID().String(),
		AuthorID:       comment.AuthorID().String(),
		Body:           comment.Body().String(),
		BodyFormat:     CommentBodyFormat,
		Status:         comment.Status().String(),
		Depth:          depth,
		ReplyCount:     replyCount,
		HasMoreReplies: hasMoreReplies,
		CreatedAt:      comment.CreatedAt(),
		UpdatedAt:      comment.UpdatedAt(),
		Attachments:    []Attachment{},
	}
	if hasParentID {
		dto.ParentID = parentID.String()
	}
	return dto
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

func buildCommentTree(comments []commentdomain.Comment, listSort CommentListSort, limit int, offset int, maxDepth int) []Comment {
	childrenByParent := make(map[commentdomain.CommentID][]commentdomain.Comment)
	byID := make(map[commentdomain.CommentID]commentdomain.Comment, len(comments))
	for _, comment := range comments {
		byID[comment.ID()] = comment
	}

	roots := make([]commentdomain.Comment, 0)
	for _, comment := range comments {
		parentID, hasParentID := comment.ParentID()
		if !hasParentID {
			roots = append(roots, comment)
			continue
		}
		if _, ok := byID[parentID]; !ok {
			roots = append(roots, comment)
			continue
		}
		childrenByParent[parentID] = append(childrenByParent[parentID], comment)
	}

	sortComments(roots, listSort)
	for parentID, children := range childrenByParent {
		sortComments(children, listSort)
		childrenByParent[parentID] = children
	}

	if offset >= len(roots) {
		return []Comment{}
	}
	end := offset + limit
	if end > len(roots) {
		end = len(roots)
	}

	result := make([]Comment, 0)
	visited := make(map[commentdomain.CommentID]bool)
	for _, root := range roots[offset:end] {
		appendCommentPreorder(&result, root, 0, maxDepth, childrenByParent, visited)
	}
	return result
}

func appendCommentPreorder(result *[]Comment, comment commentdomain.Comment, depth int, maxDepth int, childrenByParent map[commentdomain.CommentID][]commentdomain.Comment, visited map[commentdomain.CommentID]bool) {
	if visited[comment.ID()] {
		return
	}
	visited[comment.ID()] = true

	children := childrenByParent[comment.ID()]
	hasMoreReplies := depth >= maxDepth && len(children) > 0
	*result = append(*result, toCommentTreeDTO(comment, depth, len(children), hasMoreReplies))
	if depth >= maxDepth {
		return
	}

	for _, child := range children {
		appendCommentPreorder(result, child, depth+1, maxDepth, childrenByParent, visited)
	}
}

func sortComments(comments []commentdomain.Comment, listSort CommentListSort) {
	sort.SliceStable(comments, func(left int, right int) bool {
		if listSort == CommentListSortNew {
			if comments[left].CreatedAt().Equal(comments[right].CreatedAt()) {
				return comments[left].ID().String() > comments[right].ID().String()
			}
			return comments[left].CreatedAt().After(comments[right].CreatedAt())
		}
		return false
	})
}

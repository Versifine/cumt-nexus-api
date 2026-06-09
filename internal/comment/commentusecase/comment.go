package commentusecase

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
)

const (
	DefaultCommentListLimit = 20
	MaxCommentListLimit     = 50
	DefaultCommentMaxDepth  = 6
	MaxCommentMaxDepth      = 10
	CommentFormat           = postusecase.PostFormat
)

type CommentListView string

const (
	CommentListViewFlat CommentListView = "flat"
	CommentListViewTree CommentListView = "tree"
)

type CommentListSort string

const (
	CommentListSortBest          CommentListSort = "best"
	CommentListSortTop           CommentListSort = "top"
	CommentListSortNew           CommentListSort = "new"
	CommentListSortOld           CommentListSort = "old"
	CommentListSortControversial CommentListSort = "controversial"
)

type CommentUseCase struct {
	comments             CommentRepository
	posts                PostReader
	attachments          AttachmentRepository
	users                PublicUserFinder
	metadata             CommentMetadataRepository
	votes                CommentVoteRepository
	notifications        NotificationPublisher
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
	ViewerID userdomain.UserID
	View     string
	Sort     string
	Limit    int
	Offset   int
	MaxDepth int
}

type ListUserCommentsInput struct {
	Username string
	ViewerID userdomain.UserID
	Limit    int
	Offset   int
}

type UpdateCommentInput struct {
	CommentID     string
	ActorID       userdomain.UserID
	Body          string
	AttachmentIDs *[]string
}

type DeleteCommentInput struct {
	CommentID string
	ActorID   userdomain.UserID
}

type SetCommentVoteInput struct {
	CommentID string
	UserID    userdomain.UserID
	Value     int
}

type DeleteCommentVoteInput struct {
	CommentID string
	UserID    userdomain.UserID
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

type ListUserCommentsResult struct {
	Comments []Comment
	Limit    int
	Offset   int
}

type UpdateCommentResult struct {
	Comment Comment
}

type DeleteCommentResult struct{}

type SetCommentVoteResult struct {
	Vote CommentVote
}

type CommentVote struct {
	CommentID string
	UserID    string
	Value     int
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Comment struct {
	ID                string
	PostID            string
	AuthorID          string
	ParentID          string
	Body              string
	Format            string
	ContentRefs       []postusecase.ContentRef
	Author            postusecase.UserSummary
	Status            string
	Depth             int
	ReplyCount        int
	HasMoreReplies    bool
	UpvoteCount       int
	DownvoteCount     int
	Score             int
	MyVote            int
	ViewerPermissions postusecase.ViewerPermissions
	Children          []Comment
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Attachments       []Attachment
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

func NewCommentUseCase(comments CommentRepository, posts PostReader, now func() time.Time) *CommentUseCase {
	if now == nil {
		now = time.Now
	}

	uc := &CommentUseCase{
		comments:             comments,
		posts:                posts,
		commentImageMaxCount: 1,
		now:                  now,
	}
	if repo, ok := comments.(CommentMetadataRepository); ok {
		uc.metadata = repo
	}
	if repo, ok := comments.(CommentVoteRepository); ok {
		uc.votes = repo
	}
	return uc
}

func NewCommentUseCaseWithAttachments(comments CommentRepository, posts PostReader, attachments AttachmentRepository, commentImageMaxCount int, now func() time.Time) *CommentUseCase {
	uc := NewCommentUseCase(comments, posts, now)
	uc.attachments = attachments
	if commentImageMaxCount > 0 {
		uc.commentImageMaxCount = commentImageMaxCount
	}
	return uc
}

func (uc *CommentUseCase) SetPublicUserFinder(users PublicUserFinder) {
	uc.users = users
}

type NotificationPublisher interface {
	NotifyPostCommented(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, postID string) error
	NotifyCommentReplied(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, commentID string) error
	NotifyCommentUpvoted(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, commentID string) error
}

func (uc *CommentUseCase) SetNotificationPublisher(notifications NotificationPublisher) {
	uc.notifications = notifications
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

	parent, err := uc.resolveParentComment(ctx, post.ID(), input.ParentID)
	if err != nil {
		return PublishCommentResult{}, err
	}
	var parentID *commentdomain.CommentID
	if parent != nil {
		id := parent.ID()
		parentID = &id
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
	metadataViews, err := uc.loadCommentMetadataViews(ctx, []commentdomain.Comment{*comment})
	if err != nil {
		return PublishCommentResult{}, err
	}
	if err := uc.notifyCommentPublished(ctx, post, parent, *comment, input.AuthorID); err != nil {
		return PublishCommentResult{}, err
	}

	return PublishCommentResult{
		Comment: toCommentDTO(*comment, attachments, metadataViews[comment.ID()], input.AuthorID),
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
		return uc.listPostCommentsTree(ctx, post.ID(), input.ViewerID, listSort, limit, offset, maxDepth)
	}

	comments, err := uc.comments.ListVisibleByPost(ctx, post.ID(), listSort, limit, offset)
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
		result.Comments = append(result.Comments, toCommentDTO(comment, nil, CommentMetadata{}, input.ViewerID))
	}
	result.Comments, err = uc.attachCommentImages(ctx, result.Comments)
	if err != nil {
		return ListPostCommentsResult{}, err
	}
	result.Comments, err = uc.attachCommentMetadata(ctx, result.Comments, input.ViewerID)
	if err != nil {
		return ListPostCommentsResult{}, err
	}
	result.Comments, err = uc.attachCommentVotes(ctx, result.Comments, input.ViewerID)
	if err != nil {
		return ListPostCommentsResult{}, err
	}

	return result, nil
}

func (uc *CommentUseCase) listPostCommentsTree(ctx context.Context, postID postdomain.PostID, viewerID userdomain.UserID, listSort CommentListSort, limit int, offset int, maxDepth int) (ListPostCommentsResult, error) {
	comments, err := uc.comments.ListVisibleTreeByPost(ctx, postID)
	if err != nil {
		return ListPostCommentsResult{}, fmt.Errorf("list post comment tree: %w", err)
	}
	voteSummaries, err := uc.loadTreeSortVoteSummaries(ctx, comments, listSort)
	if err != nil {
		return ListPostCommentsResult{}, err
	}

	result := ListPostCommentsResult{
		Comments: buildCommentTree(comments, viewerID, listSort, voteSummaries, limit, offset, maxDepth),
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
	result.Comments, err = uc.attachCommentMetadata(ctx, result.Comments, viewerID)
	if err != nil {
		return ListPostCommentsResult{}, err
	}
	result.Comments, err = uc.attachCommentVotes(ctx, result.Comments, viewerID)
	if err != nil {
		return ListPostCommentsResult{}, err
	}
	return result, nil
}

func (uc *CommentUseCase) ListUserComments(ctx context.Context, input ListUserCommentsInput) (ListUserCommentsResult, error) {
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListUserCommentsResult{}, err
	}
	author, err := uc.findActivePublicUser(ctx, input.Username)
	if err != nil {
		return ListUserCommentsResult{}, err
	}

	comments, err := uc.comments.ListVisibleByAuthorInPublicCommunities(ctx, author.ID(), limit, offset)
	if err != nil {
		return ListUserCommentsResult{}, fmt.Errorf("list public user comments: %w", err)
	}

	result := ListUserCommentsResult{
		Comments: make([]Comment, 0, len(comments)),
		Limit:    limit,
		Offset:   offset,
	}
	for _, comment := range comments {
		result.Comments = append(result.Comments, toCommentDTO(comment, nil, CommentMetadata{}, input.ViewerID))
	}
	result.Comments, err = uc.attachCommentImages(ctx, result.Comments)
	if err != nil {
		return ListUserCommentsResult{}, err
	}
	result.Comments, err = uc.attachCommentMetadata(ctx, result.Comments, input.ViewerID)
	if err != nil {
		return ListUserCommentsResult{}, err
	}
	result.Comments, err = uc.attachCommentVotes(ctx, result.Comments, input.ViewerID)
	if err != nil {
		return ListUserCommentsResult{}, err
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
	attachmentIDs, replaceAttachments, err := parseOptionalAttachmentIDs(input.AttachmentIDs, uc.commentImageMaxCount)
	if err != nil {
		return UpdateCommentResult{}, err
	}

	now := uc.now().UTC()
	if err := comment.EditBody(body, now); err != nil {
		return UpdateCommentResult{}, err
	}
	if err := uc.comments.UpdateContent(ctx, *comment); err != nil {
		return UpdateCommentResult{}, fmt.Errorf("update comment content: %w", err)
	}
	var attachments []mediadomain.Attachment
	if replaceAttachments {
		attachments, err = uc.replaceCommentAttachments(ctx, comment.ID(), input.ActorID, attachmentIDs, now)
		if err != nil {
			return UpdateCommentResult{}, err
		}
	}

	metadataViews, err := uc.loadCommentMetadataViews(ctx, []commentdomain.Comment{*comment})
	if err != nil {
		return UpdateCommentResult{}, err
	}
	result := toCommentDTO(*comment, attachments, metadataViews[comment.ID()], input.ActorID)
	if !replaceAttachments {
		comments, err := uc.attachCommentImages(ctx, []Comment{result})
		if err != nil {
			return UpdateCommentResult{}, err
		}
		if len(comments) == 1 {
			result = comments[0]
		}
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

func (uc *CommentUseCase) SetCommentVote(ctx context.Context, input SetCommentVoteInput) (SetCommentVoteResult, error) {
	if strings.TrimSpace(input.UserID.String()) == "" {
		return SetCommentVoteResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.votes == nil {
		return SetCommentVoteResult{}, apperr.New(apperr.CodeInternal, "comment votes are not configured")
	}

	commentID, err := commentdomain.NewCommentID(input.CommentID)
	if err != nil {
		return SetCommentVoteResult{}, err
	}
	value, err := votedomain.NewVoteValue(input.Value)
	if err != nil {
		return SetCommentVoteResult{}, err
	}
	comment, err := uc.comments.FindVisibleByID(ctx, commentID)
	if err != nil {
		return SetCommentVoteResult{}, fmt.Errorf("find comment for vote: %w", err)
	}
	if _, err := uc.posts.FindVisibleByID(ctx, comment.PostID()); err != nil {
		return SetCommentVoteResult{}, fmt.Errorf("find post for comment vote: %w", err)
	}
	previousVotes, err := uc.votes.FindCommentVotesByIDsAndUser(ctx, []commentdomain.CommentID{comment.ID()}, input.UserID)
	if err != nil {
		return SetCommentVoteResult{}, fmt.Errorf("find existing comment vote: %w", err)
	}

	vote, err := votedomain.NewCommentVote(comment.ID(), input.UserID, value, uc.now().UTC())
	if err != nil {
		return SetCommentVoteResult{}, err
	}
	if err := uc.votes.UpsertCommentVote(ctx, *vote); err != nil {
		return SetCommentVoteResult{}, fmt.Errorf("upsert comment vote: %w", err)
	}
	if uc.shouldNotifyCommentUpvote(comment.AuthorID(), input.UserID, value, previousVotes[comment.ID()]) {
		if err := uc.notifications.NotifyCommentUpvoted(ctx, comment.AuthorID(), input.UserID, comment.ID().String()); err != nil {
			return SetCommentVoteResult{}, err
		}
	}

	return SetCommentVoteResult{
		Vote: toCommentVoteDTO(*vote),
	}, nil
}

func (uc *CommentUseCase) DeleteCommentVote(ctx context.Context, input DeleteCommentVoteInput) error {
	if strings.TrimSpace(input.UserID.String()) == "" {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.votes == nil {
		return apperr.New(apperr.CodeInternal, "comment votes are not configured")
	}

	commentID, err := commentdomain.NewCommentID(input.CommentID)
	if err != nil {
		return err
	}
	comment, err := uc.comments.FindVisibleByID(ctx, commentID)
	if err != nil {
		return fmt.Errorf("find comment for vote delete: %w", err)
	}
	if _, err := uc.posts.FindVisibleByID(ctx, comment.PostID()); err != nil {
		return fmt.Errorf("find post for comment vote delete: %w", err)
	}
	if err := uc.votes.DeleteCommentVote(ctx, comment.ID(), input.UserID); err != nil {
		return fmt.Errorf("delete comment vote: %w", err)
	}

	return nil
}

func (uc *CommentUseCase) notifyCommentPublished(ctx context.Context, post *postdomain.Post, parent *commentdomain.Comment, comment commentdomain.Comment, actorID userdomain.UserID) error {
	if uc.notifications == nil {
		return nil
	}
	if parent != nil {
		if err := uc.notifications.NotifyCommentReplied(ctx, parent.AuthorID(), actorID, comment.ID().String()); err != nil {
			return err
		}
		return nil
	}
	if err := uc.notifications.NotifyPostCommented(ctx, post.AuthorID(), actorID, post.ID().String()); err != nil {
		return err
	}
	return nil
}

func (uc *CommentUseCase) shouldNotifyCommentUpvote(commentAuthorID userdomain.UserID, voterID userdomain.UserID, value votedomain.VoteValue, previousVote votedomain.VoteValue) bool {
	if uc.notifications == nil || commentAuthorID == voterID || value != votedomain.VoteValueUp {
		return false
	}
	return previousVote != votedomain.VoteValueUp
}

func (uc *CommentUseCase) resolveParentComment(ctx context.Context, postID postdomain.PostID, rawParentID string) (*commentdomain.Comment, error) {
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

	return parent, nil
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
	case CommentListSortBest, CommentListSortTop, CommentListSortNew, CommentListSortOld, CommentListSortControversial:
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

func (uc *CommentUseCase) replaceCommentAttachments(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, now time.Time) ([]mediadomain.Attachment, error) {
	if uc.attachments == nil {
		return nil, apperr.New(apperr.CodeInvalidArgument, "comment image attachments are not supported")
	}
	attachments, err := uc.attachments.ReplaceReadyImagesForComment(ctx, commentID, uploaderID, attachmentIDs, uc.commentImageMaxCount, now)
	if err != nil {
		return nil, fmt.Errorf("replace comment image attachments: %w", err)
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

func (uc *CommentUseCase) attachCommentMetadata(ctx context.Context, comments []Comment, viewerID userdomain.UserID) ([]Comment, error) {
	if len(comments) == 0 {
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
	metadataViews, err := uc.loadCommentMetadataByIDs(ctx, commentIDs)
	if err != nil {
		return nil, err
	}
	for index := range comments {
		commentID, err := commentdomain.NewCommentID(comments[index].ID)
		if err != nil {
			return nil, err
		}
		metadata := normalizeCommentMetadata(metadataViews[commentID])
		comments[index].Author = metadata.Author
		comments[index].ViewerPermissions = commentViewerPermissions(comments[index].AuthorID, viewerID)
	}
	return comments, nil
}

func (uc *CommentUseCase) attachCommentVotes(ctx context.Context, comments []Comment, viewerID userdomain.UserID) ([]Comment, error) {
	if len(comments) == 0 || uc.votes == nil {
		return comments, nil
	}

	commentIDs, err := collectCommentIDs(comments)
	if err != nil {
		return nil, err
	}
	summaries, err := uc.votes.SummarizeCommentVotesByIDs(ctx, commentIDs)
	if err != nil {
		return nil, fmt.Errorf("summarize comment votes: %w", err)
	}

	myVotes := map[commentdomain.CommentID]votedomain.VoteValue{}
	if strings.TrimSpace(viewerID.String()) != "" {
		myVotes, err = uc.votes.FindCommentVotesByIDsAndUser(ctx, commentIDs, viewerID)
		if err != nil {
			return nil, fmt.Errorf("find comment votes by viewer: %w", err)
		}
	}

	applyCommentVoteViews(comments, summaries, myVotes)
	return comments, nil
}

func (uc *CommentUseCase) loadTreeSortVoteSummaries(ctx context.Context, comments []commentdomain.Comment, listSort CommentListSort) (map[commentdomain.CommentID]votedomain.CommentVoteSummary, error) {
	summaries := make(map[commentdomain.CommentID]votedomain.CommentVoteSummary, len(comments))
	if len(comments) == 0 || uc.votes == nil || !commentListSortUsesVotes(listSort) {
		return summaries, nil
	}

	commentIDs := make([]commentdomain.CommentID, 0, len(comments))
	for _, comment := range comments {
		commentIDs = append(commentIDs, comment.ID())
	}
	loaded, err := uc.votes.SummarizeCommentVotesByIDs(ctx, commentIDs)
	if err != nil {
		return nil, fmt.Errorf("summarize comment votes for tree sort: %w", err)
	}
	return loaded, nil
}

func commentListSortUsesVotes(listSort CommentListSort) bool {
	switch listSort {
	case CommentListSortBest, CommentListSortTop, CommentListSortControversial:
		return true
	default:
		return false
	}
}

func (uc *CommentUseCase) loadCommentMetadataViews(ctx context.Context, comments []commentdomain.Comment) (map[commentdomain.CommentID]CommentMetadata, error) {
	views := make(map[commentdomain.CommentID]CommentMetadata, len(comments))
	if len(comments) == 0 {
		return views, nil
	}
	commentIDs := make([]commentdomain.CommentID, 0, len(comments))
	for _, comment := range comments {
		commentIDs = append(commentIDs, comment.ID())
		views[comment.ID()] = fallbackCommentMetadata(comment)
	}
	if uc.metadata == nil {
		return views, nil
	}
	loaded, err := uc.metadata.LoadMetadataByCommentIDs(ctx, commentIDs)
	if err != nil {
		return nil, fmt.Errorf("load comment metadata: %w", err)
	}
	for commentID, metadata := range loaded {
		views[commentID] = normalizeCommentMetadata(metadata)
	}
	return views, nil
}

func (uc *CommentUseCase) loadCommentMetadataByIDs(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID]CommentMetadata, error) {
	views := make(map[commentdomain.CommentID]CommentMetadata, len(commentIDs))
	for _, commentID := range commentIDs {
		views[commentID] = CommentMetadata{
			Author: postusecase.UserSummary{Badges: []string{}},
		}
	}
	if len(commentIDs) == 0 || uc.metadata == nil {
		return views, nil
	}
	loaded, err := uc.metadata.LoadMetadataByCommentIDs(ctx, commentIDs)
	if err != nil {
		return nil, fmt.Errorf("load comment metadata: %w", err)
	}
	for commentID, metadata := range loaded {
		views[commentID] = normalizeCommentMetadata(metadata)
	}
	return views, nil
}

func collectCommentIDs(comments []Comment) ([]commentdomain.CommentID, error) {
	commentIDs := make([]commentdomain.CommentID, 0, len(comments))
	for _, comment := range comments {
		commentID, err := commentdomain.NewCommentID(comment.ID)
		if err != nil {
			return nil, err
		}
		commentIDs = append(commentIDs, commentID)
		childIDs, err := collectCommentIDs(comment.Children)
		if err != nil {
			return nil, err
		}
		commentIDs = append(commentIDs, childIDs...)
	}
	return commentIDs, nil
}

func applyCommentVoteViews(comments []Comment, summaries map[commentdomain.CommentID]votedomain.CommentVoteSummary, myVotes map[commentdomain.CommentID]votedomain.VoteValue) {
	for index := range comments {
		commentID, err := commentdomain.NewCommentID(comments[index].ID)
		if err != nil {
			continue
		}
		summary := summaries[commentID]
		comments[index].UpvoteCount = summary.UpvoteCount
		comments[index].DownvoteCount = summary.DownvoteCount
		comments[index].Score = summary.Score()
		if value, ok := myVotes[commentID]; ok {
			comments[index].MyVote = value.Int()
		} else {
			comments[index].MyVote = 0
		}
		applyCommentVoteViews(comments[index].Children, summaries, myVotes)
	}
}

func (uc *CommentUseCase) findActivePublicUser(ctx context.Context, rawUsername string) (*userdomain.User, error) {
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

func toCommentDTO(comment commentdomain.Comment, attachments []mediadomain.Attachment, metadata CommentMetadata, viewerID userdomain.UserID) Comment {
	dto := toCommentTreeDTO(comment, 0, 0, false, viewerID)
	metadata = normalizeCommentMetadata(metadata)
	dto.Author = metadata.Author
	dto.Attachments = toAttachmentDTOs(attachments)
	return dto
}

func toCommentTreeDTO(comment commentdomain.Comment, depth int, replyCount int, hasMoreReplies bool, viewerID userdomain.UserID) Comment {
	parentID, hasParentID := comment.ParentID()
	dto := Comment{
		ID:                comment.ID().String(),
		PostID:            comment.PostID().String(),
		AuthorID:          comment.AuthorID().String(),
		Body:              comment.Body().String(),
		Format:            CommentFormat,
		ContentRefs:       []postusecase.ContentRef{},
		Author:            fallbackCommentMetadata(comment).Author,
		Status:            comment.Status().String(),
		Depth:             depth,
		ReplyCount:        replyCount,
		HasMoreReplies:    hasMoreReplies,
		UpvoteCount:       0,
		DownvoteCount:     0,
		Score:             0,
		MyVote:            0,
		ViewerPermissions: commentViewerPermissions(comment.AuthorID().String(), viewerID),
		Children:          []Comment{},
		CreatedAt:         comment.CreatedAt(),
		UpdatedAt:         comment.UpdatedAt(),
		Attachments:       []Attachment{},
	}
	if hasParentID {
		dto.ParentID = parentID.String()
	}
	return dto
}

func fallbackCommentMetadata(comment commentdomain.Comment) CommentMetadata {
	return CommentMetadata{
		Author: postusecase.UserSummary{
			ID:     comment.AuthorID().String(),
			Badges: []string{},
		},
	}
}

func normalizeCommentMetadata(metadata CommentMetadata) CommentMetadata {
	if metadata.Author.Badges == nil {
		metadata.Author.Badges = []string{}
	}
	if metadata.Author.DisplayName == "" {
		metadata.Author.DisplayName = metadata.Author.Username
	}
	return metadata
}

func commentViewerPermissions(authorID string, viewerID userdomain.UserID) postusecase.ViewerPermissions {
	if strings.TrimSpace(viewerID.String()) == "" {
		return postusecase.ViewerPermissions{}
	}
	isAuthor := authorID == viewerID.String()
	return postusecase.ViewerPermissions{
		CanComment: true,
		CanVote:    true,
		CanReport:  true,
		CanEdit:    isAuthor,
		CanDelete:  isAuthor,
	}
}

func toCommentVoteDTO(vote votedomain.CommentVote) CommentVote {
	return CommentVote{
		CommentID: vote.CommentID().String(),
		UserID:    vote.UserID().String(),
		Value:     vote.Value().Int(),
		CreatedAt: vote.CreatedAt(),
		UpdatedAt: vote.UpdatedAt(),
	}
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

func buildCommentTree(comments []commentdomain.Comment, viewerID userdomain.UserID, listSort CommentListSort, voteSummaries map[commentdomain.CommentID]votedomain.CommentVoteSummary, limit int, offset int, maxDepth int) []Comment {
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

	sortComments(roots, listSort, voteSummaries)
	for parentID, children := range childrenByParent {
		sortComments(children, listSort, voteSummaries)
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
		appendCommentPreorder(&result, root, 0, maxDepth, childrenByParent, visited, viewerID)
	}
	return result
}

func appendCommentPreorder(result *[]Comment, comment commentdomain.Comment, depth int, maxDepth int, childrenByParent map[commentdomain.CommentID][]commentdomain.Comment, visited map[commentdomain.CommentID]bool, viewerID userdomain.UserID) {
	if visited[comment.ID()] {
		return
	}
	visited[comment.ID()] = true

	children := childrenByParent[comment.ID()]
	hasMoreReplies := depth >= maxDepth && len(children) > 0
	*result = append(*result, toCommentTreeDTO(comment, depth, len(children), hasMoreReplies, viewerID))
	if depth >= maxDepth {
		return
	}

	for _, child := range children {
		appendCommentPreorder(result, child, depth+1, maxDepth, childrenByParent, visited, viewerID)
	}
}

func sortComments(comments []commentdomain.Comment, listSort CommentListSort, voteSummaries map[commentdomain.CommentID]votedomain.CommentVoteSummary) {
	sort.SliceStable(comments, func(left int, right int) bool {
		switch listSort {
		case CommentListSortOld:
			if comments[left].CreatedAt().Equal(comments[right].CreatedAt()) {
				return comments[left].ID().String() < comments[right].ID().String()
			}
			return comments[left].CreatedAt().Before(comments[right].CreatedAt())
		case CommentListSortTop:
			leftSummary := voteSummaries[comments[left].ID()]
			rightSummary := voteSummaries[comments[right].ID()]
			if leftSummary.Score() != rightSummary.Score() {
				return leftSummary.Score() > rightSummary.Score()
			}
			if leftSummary.UpvoteCount != rightSummary.UpvoteCount {
				return leftSummary.UpvoteCount > rightSummary.UpvoteCount
			}
			return newerCommentFirst(comments[left], comments[right])
		case CommentListSortBest:
			leftSummary := voteSummaries[comments[left].ID()]
			rightSummary := voteSummaries[comments[right].ID()]
			leftScore := wilsonLowerBound(leftSummary.UpvoteCount, leftSummary.DownvoteCount)
			rightScore := wilsonLowerBound(rightSummary.UpvoteCount, rightSummary.DownvoteCount)
			if leftScore != rightScore {
				return leftScore > rightScore
			}
			if leftSummary.Score() != rightSummary.Score() {
				return leftSummary.Score() > rightSummary.Score()
			}
			return newerCommentFirst(comments[left], comments[right])
		case CommentListSortControversial:
			leftSummary := voteSummaries[comments[left].ID()]
			rightSummary := voteSummaries[comments[right].ID()]
			leftScore := controversyScore(leftSummary.UpvoteCount, leftSummary.DownvoteCount)
			rightScore := controversyScore(rightSummary.UpvoteCount, rightSummary.DownvoteCount)
			if leftScore != rightScore {
				return leftScore > rightScore
			}
			leftTotal := leftSummary.UpvoteCount + leftSummary.DownvoteCount
			rightTotal := rightSummary.UpvoteCount + rightSummary.DownvoteCount
			if leftTotal != rightTotal {
				return leftTotal > rightTotal
			}
			return newerCommentFirst(comments[left], comments[right])
		default:
			if comments[left].CreatedAt().Equal(comments[right].CreatedAt()) {
				return comments[left].ID().String() > comments[right].ID().String()
			}
			return comments[left].CreatedAt().After(comments[right].CreatedAt())
		}
	})
}

func newerCommentFirst(left commentdomain.Comment, right commentdomain.Comment) bool {
	if left.CreatedAt().Equal(right.CreatedAt()) {
		return left.ID().String() > right.ID().String()
	}
	return left.CreatedAt().After(right.CreatedAt())
}

func wilsonLowerBound(upvotes int, downvotes int) float64 {
	total := upvotes + downvotes
	if total <= 0 {
		return 0
	}

	z := 1.96
	positiveRatio := float64(upvotes) / float64(total)
	totalFloat := float64(total)
	numerator := positiveRatio + (z*z)/(2*totalFloat) - z*math.Sqrt((positiveRatio*(1-positiveRatio)+(z*z)/(4*totalFloat))/totalFloat)
	denominator := 1 + (z*z)/totalFloat
	return numerator / denominator
}

func controversyScore(upvotes int, downvotes int) float64 {
	total := upvotes + downvotes
	if total <= 0 {
		return 0
	}
	return float64(total) * (1 - (math.Abs(float64(upvotes-downvotes)) / float64(total)))
}

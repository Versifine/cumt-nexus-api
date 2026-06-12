package commenthttp

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentusecase"
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	comments CommentUseCase
}

type CommentUseCase interface {
	PublishComment(ctx context.Context, input commentusecase.PublishCommentInput) (commentusecase.PublishCommentResult, error)
	ListPostComments(ctx context.Context, input commentusecase.ListPostCommentsInput) (commentusecase.ListPostCommentsResult, error)
	ListUserComments(ctx context.Context, input commentusecase.ListUserCommentsInput) (commentusecase.ListUserCommentsResult, error)
	UpdateComment(ctx context.Context, input commentusecase.UpdateCommentInput) (commentusecase.UpdateCommentResult, error)
	DeleteComment(ctx context.Context, input commentusecase.DeleteCommentInput) (commentusecase.DeleteCommentResult, error)
	SetCommentVote(ctx context.Context, input commentusecase.SetCommentVoteInput) (commentusecase.SetCommentVoteResult, error)
	DeleteCommentVote(ctx context.Context, input commentusecase.DeleteCommentVoteInput) error
}

type publishCommentRequest struct {
	Body          string              `json:"body" binding:"required"`
	ParentID      string              `json:"parent_id"`
	AttachmentIDs []string            `json:"attachment_ids"`
	ContentRefs   []contentRefRequest `json:"content_refs"`
}

type updateCommentRequest struct {
	Body          string               `json:"body" binding:"required"`
	AttachmentIDs *[]string            `json:"attachment_ids"`
	ContentRefs   *[]contentRefRequest `json:"content_refs"`
}

type setCommentVoteRequest struct {
	Value int `json:"value" binding:"required"`
}

type commentResponse struct {
	ID                string                    `json:"id"`
	PostID            string                    `json:"post_id"`
	AuthorID          string                    `json:"author_id"`
	ParentID          *string                   `json:"parent_id"`
	Body              string                    `json:"body"`
	Format            string                    `json:"format"`
	ContentRefs       []contentRefResponse      `json:"content_refs"`
	Author            userSummaryResponse       `json:"author"`
	Status            string                    `json:"status"`
	Depth             int                       `json:"depth"`
	ReplyCount        int                       `json:"reply_count"`
	HasMoreReplies    bool                      `json:"has_more_replies"`
	UpvoteCount       int                       `json:"upvote_count"`
	DownvoteCount     int                       `json:"downvote_count"`
	Score             int                       `json:"score"`
	MyVote            int                       `json:"my_vote"`
	ViewerPermissions viewerPermissionsResponse `json:"viewer_permissions"`
	Children          []commentResponse         `json:"children"`
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
	Attachments       []attachmentResponse      `json:"attachments"`
}

type contentRefResponse struct {
	Kind  string `json:"kind"`
	RefID string `json:"ref_id"`
}

type contentRefRequest struct {
	Kind  string `json:"kind"`
	RefID string `json:"ref_id"`
}

type userSummaryResponse struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	AvatarURL   string   `json:"avatar_url"`
	Headline    string   `json:"headline"`
	Badges      []string `json:"badges"`
}

type viewerPermissionsResponse struct {
	CanComment  bool `json:"can_comment"`
	CanVote     bool `json:"can_vote"`
	CanReport   bool `json:"can_report"`
	CanEdit     bool `json:"can_edit"`
	CanDelete   bool `json:"can_delete"`
	CanModerate bool `json:"can_moderate"`
}

type attachmentResponse struct {
	ID           string    `json:"id"`
	Kind         string    `json:"kind"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url"`
	Width        *int      `json:"width"`
	Height       *int      `json:"height"`
	SizeBytes    int64     `json:"size_bytes"`
	MimeType     string    `json:"mime_type"`
	AltText      string    `json:"alt_text"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type publishCommentResponse struct {
	Comment commentResponse `json:"comment"`
}

type commentVoteResponse struct {
	CommentID string    `json:"comment_id"`
	UserID    string    `json:"user_id"`
	Value     int       `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type setCommentVoteResponse struct {
	Vote commentVoteResponse `json:"vote"`
}

type listPostCommentsResponse struct {
	Comments []commentResponse `json:"comments"`
	View     string            `json:"view"`
	Sort     string            `json:"sort"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
	MaxDepth int               `json:"max_depth"`
}

type listUserCommentsResponse struct {
	Comments []commentResponse `json:"comments"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

func NewHandler(comments CommentUseCase) *Handler {
	return &Handler{
		comments: comments,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	RegisterReadRoutes(group, handler)
	RegisterWriteRoutes(group, handler)
}

func RegisterReadRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/posts/:id/comments", handler.ListPostComments)
	group.GET("/users/:username/comments", handler.ListUserComments)
}

func RegisterWriteRoutes(group *gin.RouterGroup, handler *Handler) {
	group.POST("/posts/:id/comments", handler.PublishComment)
	group.PUT("/comments/:id/vote", handler.SetCommentVote)
	group.DELETE("/comments/:id/vote", handler.DeleteCommentVote)
	group.PATCH("/comments/:id", handler.UpdateComment)
	group.DELETE("/comments/:id", handler.DeleteComment)
}

func (h *Handler) PublishComment(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req publishCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid publish comment request"))
		c.Abort()
		return
	}

	result, err := h.comments.PublishComment(c.Request.Context(), commentusecase.PublishCommentInput{
		PostID:        c.Param("id"),
		AuthorID:      userID,
		ParentID:      req.ParentID,
		Body:          req.Body,
		AttachmentIDs: req.AttachmentIDs,
		ContentRefs:   toContentRefInputs(req.ContentRefs),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, publishCommentResponse{
		Comment: toCommentResponse(result.Comment),
	})
}

func (h *Handler) ListPostComments(c *gin.Context) {
	userID, _ := authcontext.CurrentUserID(c.Request.Context())

	limit, err := parseOptionalIntQuery(c, "limit")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	offset, err := parseOptionalIntQuery(c, "offset")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	maxDepth, err := parseOptionalIntQuery(c, "max_depth")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	result, err := h.comments.ListPostComments(c.Request.Context(), commentusecase.ListPostCommentsInput{
		PostID:   c.Param("id"),
		ViewerID: userID,
		View:     c.Query("view"),
		Sort:     c.Query("sort"),
		Limit:    limit,
		Offset:   offset,
		MaxDepth: maxDepth,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listPostCommentsResponse{
		Comments: make([]commentResponse, 0, len(result.Comments)),
		View:     result.View,
		Sort:     result.Sort,
		Limit:    result.Limit,
		Offset:   result.Offset,
		MaxDepth: result.MaxDepth,
	}
	for _, comment := range result.Comments {
		response.Comments = append(response.Comments, toCommentResponse(comment))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) ListUserComments(c *gin.Context) {
	userID, _ := authcontext.CurrentUserID(c.Request.Context())

	limit, err := parseOptionalIntQuery(c, "limit")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	offset, err := parseOptionalIntQuery(c, "offset")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	result, err := h.comments.ListUserComments(c.Request.Context(), commentusecase.ListUserCommentsInput{
		Username: c.Param("username"),
		ViewerID: userID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listUserCommentsResponse{
		Comments: make([]commentResponse, 0, len(result.Comments)),
		Limit:    result.Limit,
		Offset:   result.Offset,
	}
	for _, comment := range result.Comments {
		response.Comments = append(response.Comments, toCommentResponse(comment))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) UpdateComment(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req updateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid update comment request"))
		c.Abort()
		return
	}

	result, err := h.comments.UpdateComment(c.Request.Context(), commentusecase.UpdateCommentInput{
		CommentID:     c.Param("id"),
		ActorID:       userID,
		Body:          req.Body,
		AttachmentIDs: req.AttachmentIDs,
		ContentRefs:   toOptionalContentRefInputs(req.ContentRefs),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, publishCommentResponse{
		Comment: toCommentResponse(result.Comment),
	})
}

func (h *Handler) DeleteComment(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	if _, err := h.comments.DeleteComment(c.Request.Context(), commentusecase.DeleteCommentInput{
		CommentID: c.Param("id"),
		ActorID:   userID,
	}); err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) SetCommentVote(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req setCommentVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid comment vote request"))
		c.Abort()
		return
	}

	result, err := h.comments.SetCommentVote(c.Request.Context(), commentusecase.SetCommentVoteInput{
		CommentID: c.Param("id"),
		UserID:    userID,
		Value:     req.Value,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, setCommentVoteResponse{
		Vote: toCommentVoteResponse(result.Vote),
	})
}

func (h *Handler) DeleteCommentVote(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	if err := h.comments.DeleteCommentVote(c.Request.Context(), commentusecase.DeleteCommentVoteInput{
		CommentID: c.Param("id"),
		UserID:    userID,
	}); err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.Status(http.StatusNoContent)
}

func parseOptionalIntQuery(c *gin.Context, key string) (int, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, apperr.New(apperr.CodeInvalidArgument, "invalid "+key+" query")
	}
	return value, nil
}

func toCommentResponse(comment commentusecase.Comment) commentResponse {
	var parentID *string
	if strings.TrimSpace(comment.ParentID) != "" {
		parentID = &comment.ParentID
	}
	return commentResponse{
		ID:                comment.ID,
		PostID:            comment.PostID,
		AuthorID:          comment.AuthorID,
		ParentID:          parentID,
		Body:              comment.Body,
		Format:            comment.Format,
		ContentRefs:       toContentRefResponses(comment.ContentRefs),
		Author:            toUserSummaryResponse(comment.Author),
		Status:            comment.Status,
		Depth:             comment.Depth,
		ReplyCount:        comment.ReplyCount,
		HasMoreReplies:    comment.HasMoreReplies,
		UpvoteCount:       comment.UpvoteCount,
		DownvoteCount:     comment.DownvoteCount,
		Score:             comment.Score,
		MyVote:            comment.MyVote,
		ViewerPermissions: toViewerPermissionsResponse(comment.ViewerPermissions),
		Children:          toCommentResponses(comment.Children),
		CreatedAt:         comment.CreatedAt,
		UpdatedAt:         comment.UpdatedAt,
		Attachments:       toAttachmentResponses(comment.Attachments),
	}
}

func toCommentResponses(comments []commentusecase.Comment) []commentResponse {
	response := make([]commentResponse, 0, len(comments))
	for _, comment := range comments {
		response = append(response, toCommentResponse(comment))
	}
	return response
}

func toCommentVoteResponse(vote commentusecase.CommentVote) commentVoteResponse {
	return commentVoteResponse{
		CommentID: vote.CommentID,
		UserID:    vote.UserID,
		Value:     vote.Value,
		CreatedAt: vote.CreatedAt,
		UpdatedAt: vote.UpdatedAt,
	}
}

func toContentRefResponses(refs []postusecase.ContentRef) []contentRefResponse {
	response := make([]contentRefResponse, 0, len(refs))
	for _, ref := range refs {
		response = append(response, contentRefResponse{
			Kind:  ref.Kind,
			RefID: ref.RefID,
		})
	}
	return response
}

func toContentRefInputs(refs []contentRefRequest) []postusecase.ContentRefInput {
	inputs := make([]postusecase.ContentRefInput, 0, len(refs))
	for _, ref := range refs {
		inputs = append(inputs, postusecase.ContentRefInput{
			Kind:  ref.Kind,
			RefID: ref.RefID,
		})
	}
	return inputs
}

func toOptionalContentRefInputs(refs *[]contentRefRequest) *[]postusecase.ContentRefInput {
	if refs == nil {
		return nil
	}
	inputs := toContentRefInputs(*refs)
	return &inputs
}

func toUserSummaryResponse(user postusecase.UserSummary) userSummaryResponse {
	badges := user.Badges
	if badges == nil {
		badges = []string{}
	}
	return userSummaryResponse{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		Headline:    user.Headline,
		Badges:      badges,
	}
}

func toViewerPermissionsResponse(permissions postusecase.ViewerPermissions) viewerPermissionsResponse {
	return viewerPermissionsResponse{
		CanComment:  permissions.CanComment,
		CanVote:     permissions.CanVote,
		CanReport:   permissions.CanReport,
		CanEdit:     permissions.CanEdit,
		CanDelete:   permissions.CanDelete,
		CanModerate: permissions.CanModerate,
	}
}

func toAttachmentResponses(attachments []commentusecase.Attachment) []attachmentResponse {
	response := make([]attachmentResponse, 0, len(attachments))
	for _, attachment := range attachments {
		response = append(response, attachmentResponse{
			ID:           attachment.ID,
			Kind:         attachment.Kind,
			URL:          attachment.URL,
			ThumbnailURL: attachment.ThumbnailURL,
			Width:        attachment.Width,
			Height:       attachment.Height,
			SizeBytes:    attachment.SizeBytes,
			MimeType:     attachment.MimeType,
			AltText:      attachment.AltText,
			Status:       attachment.Status,
			CreatedAt:    attachment.CreatedAt,
		})
	}
	return response
}

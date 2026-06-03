package commenthttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authtoken"
	"github.com/Versifine/cumt-nexus-api/internal/auth/delivery/authhttp"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentusecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

func TestPublishCommentReturnsCreatedComment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC)
	comments := &fakeCommentUseCase{
		publishResult: commentusecase.PublishCommentResult{
			Comment: newCommentResultWithAttachment("Reply", now),
		},
	}
	router := newCommentTestRouter(comments, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/comments", bytes.NewBufferString(`{
		"body": "Reply",
		"attachment_ids": ["98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a"]
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if !comments.publishCalled {
		t.Fatal("expected PublishComment to be called")
	}
	if comments.publishInput.AuthorID != userID {
		t.Fatalf("expected author %q, got %q", userID.String(), comments.publishInput.AuthorID.String())
	}
	if comments.publishInput.Body != "Reply" {
		t.Fatalf("expected body Reply, got %q", comments.publishInput.Body)
	}
	if len(comments.publishInput.AttachmentIDs) != 1 || comments.publishInput.AttachmentIDs[0] != "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a" {
		t.Fatalf("unexpected attachment ids: %#v", comments.publishInput.AttachmentIDs)
	}

	var response publishCommentResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Comment.Status != "visible" {
		t.Fatalf("expected visible comment, got %q", response.Comment.Status)
	}
	if len(response.Comment.Attachments) != 1 || response.Comment.Attachments[0].URL != "https://assets.example.com/comment.png" {
		t.Fatalf("expected attachment in response, got %#v", response.Comment.Attachments)
	}
}

func TestListPostCommentsReturnsComments(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC)
	parentID := "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a"
	comments := &fakeCommentUseCase{
		listResult: commentusecase.ListPostCommentsResult{
			Comments: []commentusecase.Comment{
				newCommentResult("Newer", now),
				newChildCommentResultWithAttachment("Older", parentID, now.Add(-time.Minute)),
			},
			View:     "tree",
			Sort:     "new",
			Limit:    20,
			Offset:   5,
			MaxDepth: 6,
		},
	}
	router := newCommentTestRouter(comments, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/comments?view=tree&sort=new&limit=20&offset=5&max_depth=6", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !comments.listCalled {
		t.Fatal("expected ListPostComments to be called")
	}
	if comments.listInput.View != "tree" || comments.listInput.Sort != "new" || comments.listInput.Limit != 20 || comments.listInput.Offset != 5 || comments.listInput.MaxDepth != 6 {
		t.Fatalf("unexpected list input: %#v", comments.listInput)
	}

	var response listPostCommentsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Comments) != 2 {
		t.Fatalf("expected two comments, got %d", len(response.Comments))
	}
	if response.View != "tree" || response.Sort != "new" || response.MaxDepth != 6 {
		t.Fatalf("unexpected response metadata: %#v", response)
	}
	if response.Comments[0].ParentID != nil {
		t.Fatalf("expected root parent_id null, got %#v", response.Comments[0].ParentID)
	}
	if response.Comments[0].BodyFormat != "markdown" || response.Comments[1].BodyFormat != "markdown" {
		t.Fatalf("expected markdown body format, got %#v", response.Comments)
	}
	if response.Comments[1].ParentID == nil || *response.Comments[1].ParentID != parentID {
		t.Fatalf("expected child parent id %q, got %#v", parentID, response.Comments[1].ParentID)
	}
	if len(response.Comments[1].Attachments) != 1 || response.Comments[1].Attachments[0].Kind != "image" {
		t.Fatalf("expected child comment attachment, got %#v", response.Comments[1].Attachments)
	}
}

func TestUpdateCommentReturnsUpdatedComment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	comments := &fakeCommentUseCase{
		updateResult: commentusecase.UpdateCommentResult{
			Comment: newCommentResult("Updated body", now),
		},
	}
	router := newCommentTestRouter(comments, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/comments/98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a", bytes.NewBufferString(`{
		"body": "Updated body"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !comments.updateCalled {
		t.Fatal("expected UpdateComment to be called")
	}
	if comments.updateInput.ActorID != userID {
		t.Fatalf("expected actor %q, got %q", userID.String(), comments.updateInput.ActorID.String())
	}
	if comments.updateInput.Body != "Updated body" {
		t.Fatalf("unexpected update input: %#v", comments.updateInput)
	}

	var response publishCommentResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Comment.Body != "Updated body" || response.Comment.BodyFormat != "markdown" {
		t.Fatalf("unexpected update response: %#v", response.Comment)
	}
}

func TestDeleteCommentReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	comments := &fakeCommentUseCase{}
	router := newCommentTestRouter(comments, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/comments/98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, recorder.Code, recorder.Body.String())
	}
	if !comments.deleteCalled {
		t.Fatal("expected DeleteComment to be called")
	}
	if comments.deleteInput.ActorID != userID {
		t.Fatalf("expected actor %q, got %q", userID.String(), comments.deleteInput.ActorID.String())
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", recorder.Body.String())
	}
}

func TestCommentRoutesRejectInvalidAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	comments := &fakeCommentUseCase{}
	router := newCommentTestRouter(comments, &fakeAccessTokenParser{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/comments", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertCommentErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if comments.publishCalled || comments.listCalled || comments.updateCalled || comments.deleteCalled {
		t.Fatal("comment usecase should not be called for invalid auth")
	}
}

func TestListPostCommentsRejectsInvalidQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	comments := &fakeCommentUseCase{}
	router := newCommentTestRouter(comments, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/comments?offset=abc", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertCommentErrorCode(t, recorder, apperr.CodeInvalidArgument)
	if comments.listCalled {
		t.Fatal("comment usecase should not be called for invalid query")
	}
}

func TestCommentUseCaseErrorMapsToHTTPError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	comments := &fakeCommentUseCase{
		publishErr: apperr.New(apperr.CodeNotFound, "post not found"),
	}
	router := newCommentTestRouter(comments, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/comments", bytes.NewBufferString(`{
		"body": "Reply"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
	assertCommentErrorCode(t, recorder, apperr.CodeNotFound)
}

type fakeCommentUseCase struct {
	publishCalled bool
	listCalled    bool
	updateCalled  bool
	deleteCalled  bool
	publishInput  commentusecase.PublishCommentInput
	listInput     commentusecase.ListPostCommentsInput
	updateInput   commentusecase.UpdateCommentInput
	deleteInput   commentusecase.DeleteCommentInput
	publishResult commentusecase.PublishCommentResult
	listResult    commentusecase.ListPostCommentsResult
	updateResult  commentusecase.UpdateCommentResult
	deleteResult  commentusecase.DeleteCommentResult
	publishErr    error
	listErr       error
	updateErr     error
	deleteErr     error
}

func (f *fakeCommentUseCase) PublishComment(ctx context.Context, input commentusecase.PublishCommentInput) (commentusecase.PublishCommentResult, error) {
	f.publishCalled = true
	f.publishInput = input
	return f.publishResult, f.publishErr
}

func (f *fakeCommentUseCase) ListPostComments(ctx context.Context, input commentusecase.ListPostCommentsInput) (commentusecase.ListPostCommentsResult, error) {
	f.listCalled = true
	f.listInput = input
	return f.listResult, f.listErr
}

func (f *fakeCommentUseCase) UpdateComment(ctx context.Context, input commentusecase.UpdateCommentInput) (commentusecase.UpdateCommentResult, error) {
	f.updateCalled = true
	f.updateInput = input
	return f.updateResult, f.updateErr
}

func (f *fakeCommentUseCase) DeleteComment(ctx context.Context, input commentusecase.DeleteCommentInput) (commentusecase.DeleteCommentResult, error) {
	f.deleteCalled = true
	f.deleteInput = input
	return f.deleteResult, f.deleteErr
}

type fakeAccessTokenParser struct {
	claims *authtoken.AccessTokenClaims
	err    error
}

func (f *fakeAccessTokenParser) ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error) {
	return f.claims, f.err
}

func newCommentTestRouter(comments CommentUseCase, parser authhttp.AccessTokenParser) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())

	protected := router.Group("/api/v1")
	protected.Use(authhttp.RequireAuth(parser))
	RegisterRoutes(protected, NewHandler(comments))

	return router
}

func validParser() *fakeAccessTokenParser {
	return validParserWithUserID(userdomain.NewGeneratedUserID())
}

func validParserWithUserID(userID userdomain.UserID) *fakeAccessTokenParser {
	return &fakeAccessTokenParser{
		claims: &authtoken.AccessTokenClaims{
			UserID: userID,
		},
	}
}

func newCommentResult(body string, now time.Time) commentusecase.Comment {
	return commentusecase.Comment{
		ID:             "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a",
		PostID:         "8f92e975-5323-4a58-bac1-1336b668183c",
		AuthorID:       userdomain.NewGeneratedUserID().String(),
		Body:           body,
		BodyFormat:     "markdown",
		Status:         "visible",
		Depth:          0,
		ReplyCount:     0,
		HasMoreReplies: false,
		CreatedAt:      now,
		UpdatedAt:      now,
		Attachments:    []commentusecase.Attachment{},
	}
}

func newCommentResultWithAttachment(body string, now time.Time) commentusecase.Comment {
	comment := newCommentResult(body, now)
	comment.Attachments = []commentusecase.Attachment{newAttachmentResult(now)}
	return comment
}

func newChildCommentResult(body string, parentID string, now time.Time) commentusecase.Comment {
	comment := newCommentResult(body, now)
	comment.ID = "47fdf0c9-ae37-4919-982f-e88a8cb8b693"
	comment.ParentID = parentID
	comment.Depth = 1
	return comment
}

func newChildCommentResultWithAttachment(body string, parentID string, now time.Time) commentusecase.Comment {
	comment := newChildCommentResult(body, parentID, now)
	comment.Attachments = []commentusecase.Attachment{newAttachmentResult(now)}
	return comment
}

func newAttachmentResult(now time.Time) commentusecase.Attachment {
	return commentusecase.Attachment{
		ID:        "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a",
		Kind:      "image",
		URL:       "https://assets.example.com/comment.png",
		SizeBytes: 100,
		MimeType:  "image/png",
		AltText:   "Campus",
		Status:    "ready",
		CreatedAt: now,
	}
}

func assertCommentErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode apperr.Code) {
	t.Helper()

	var response httpserver.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != string(wantCode) {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
}

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
			Comment: newCommentResult("Reply", now),
		},
	}
	router := newCommentTestRouter(comments, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/comments", bytes.NewBufferString(`{
		"body": "Reply"
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

	var response publishCommentResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Comment.Status != "visible" {
		t.Fatalf("expected visible comment, got %q", response.Comment.Status)
	}
}

func TestListPostCommentsReturnsComments(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC)
	comments := &fakeCommentUseCase{
		listResult: commentusecase.ListPostCommentsResult{
			Comments: []commentusecase.Comment{newCommentResult("Newer", now), newCommentResult("Older", now.Add(-time.Minute))},
			Limit:    20,
			Offset:   5,
		},
	}
	router := newCommentTestRouter(comments, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/comments?limit=20&offset=5", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !comments.listCalled {
		t.Fatal("expected ListPostComments to be called")
	}
	if comments.listInput.Limit != 20 || comments.listInput.Offset != 5 {
		t.Fatalf("expected limit=20 offset=5, got limit=%d offset=%d", comments.listInput.Limit, comments.listInput.Offset)
	}

	var response listPostCommentsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Comments) != 2 {
		t.Fatalf("expected two comments, got %d", len(response.Comments))
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
	if comments.publishCalled || comments.listCalled {
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
	publishInput  commentusecase.PublishCommentInput
	listInput     commentusecase.ListPostCommentsInput
	publishResult commentusecase.PublishCommentResult
	listResult    commentusecase.ListPostCommentsResult
	publishErr    error
	listErr       error
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
		ID:        "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a",
		PostID:    "8f92e975-5323-4a58-bac1-1336b668183c",
		AuthorID:  userdomain.NewGeneratedUserID().String(),
		Body:      body,
		Status:    "visible",
		CreatedAt: now,
		UpdatedAt: now,
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

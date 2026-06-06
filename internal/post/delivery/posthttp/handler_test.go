package posthttp

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
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

func TestPublishPostReturnsCreatedPost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	posts := &fakePostUseCase{
		publishResult: postusecase.PublishPostResult{
			Post: newPostResultWithAttachment("Hello", now),
		},
	}
	router := newPostTestRouter(posts, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/communities/campus/posts", bytes.NewBufferString(`{
		"title": "Hello",
		"body": "Post body",
		"attachment_ids": ["98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a"]
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if !posts.publishCalled {
		t.Fatal("expected PublishPost to be called")
	}
	if posts.publishInput.AuthorID != userID {
		t.Fatalf("expected author %q, got %q", userID.String(), posts.publishInput.AuthorID.String())
	}
	if posts.publishInput.CommunitySlug != "campus" {
		t.Fatalf("expected community slug campus, got %q", posts.publishInput.CommunitySlug)
	}
	if len(posts.publishInput.AttachmentIDs) != 1 || posts.publishInput.AttachmentIDs[0] != "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a" {
		t.Fatalf("unexpected attachment ids: %#v", posts.publishInput.AttachmentIDs)
	}

	var response publishPostResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Post.Title != "Hello" {
		t.Fatalf("expected title Hello, got %q", response.Post.Title)
	}
	if response.Post.BodyFormat != "markdown" {
		t.Fatalf("expected body_format markdown, got %q", response.Post.BodyFormat)
	}
	if len(response.Post.Attachments) != 1 || response.Post.Attachments[0].URL == "" {
		t.Fatalf("expected attachment response, got %#v", response.Post.Attachments)
	}
}

func TestListCommunityPostsReturnsPosts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	posts := &fakePostUseCase{
		listResult: postusecase.ListCommunityPostsResult{
			Posts:  []postusecase.Post{newPostResult("Newer", now), newPostResult("Older", now.Add(-time.Minute))},
			Limit:  20,
			Offset: 5,
		},
	}
	router := newPostTestRouter(posts, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/campus/posts?sort=hot&limit=20&offset=5", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !posts.listCalled {
		t.Fatal("expected ListCommunityPosts to be called")
	}
	if posts.listInput.Limit != 20 || posts.listInput.Offset != 5 {
		t.Fatalf("expected limit=20 offset=5, got limit=%d offset=%d", posts.listInput.Limit, posts.listInput.Offset)
	}
	if posts.listInput.Sort != "hot" {
		t.Fatalf("expected sort hot, got %q", posts.listInput.Sort)
	}
	if posts.listInput.ViewerID != userID {
		t.Fatalf("expected viewer %q, got %q", userID.String(), posts.listInput.ViewerID.String())
	}

	var response listCommunityPostsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Posts) != 2 {
		t.Fatalf("expected two posts, got %d", len(response.Posts))
	}
	if response.Posts[0].UpvoteCount != 2 || response.Posts[0].DownvoteCount != 1 || response.Posts[0].Score != 1 || response.Posts[0].MyVote != 1 {
		t.Fatalf("unexpected vote fields: %#v", response.Posts[0])
	}
	if response.Posts[0].BodyFormat != "markdown" {
		t.Fatalf("expected body_format markdown, got %q", response.Posts[0].BodyFormat)
	}
}

func TestListCommunityPostsAllowsAnonymousViewer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	posts := &fakePostUseCase{
		listResult: postusecase.ListCommunityPostsResult{
			Posts: []postusecase.Post{newPostResult("Anonymous", now)},
			Limit: 20,
		},
	}
	router := newPostTestRouter(posts, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/campus/posts?limit=20", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !posts.listCalled {
		t.Fatal("expected ListCommunityPosts to be called")
	}
	if posts.listInput.ViewerID.String() != "" {
		t.Fatalf("expected empty anonymous viewer, got %q", posts.listInput.ViewerID.String())
	}
}

func TestGetPostReturnsPost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	posts := &fakePostUseCase{
		getResult: postusecase.GetPostResult{
			Post: newPostResult("Hello", now),
		},
	}
	router := newPostTestRouter(posts, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !posts.getCalled {
		t.Fatal("expected GetPost to be called")
	}
	if posts.getInput.PostID != "8f92e975-5323-4a58-bac1-1336b668183c" {
		t.Fatalf("unexpected post id %q", posts.getInput.PostID)
	}
	if posts.getInput.ViewerID != userID {
		t.Fatalf("expected viewer %q, got %q", userID.String(), posts.getInput.ViewerID.String())
	}
}

func TestGetPostAllowsAnonymousViewer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	posts := &fakePostUseCase{
		getResult: postusecase.GetPostResult{
			Post: newPostResult("Anonymous", now),
		},
	}
	router := newPostTestRouter(posts, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !posts.getCalled {
		t.Fatal("expected GetPost to be called")
	}
	if posts.getInput.ViewerID.String() != "" {
		t.Fatalf("expected empty anonymous viewer, got %q", posts.getInput.ViewerID.String())
	}
}

func TestUpdatePostReturnsUpdatedPost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	posts := &fakePostUseCase{
		updateResult: postusecase.UpdatePostResult{
			Post: newPostResult("Updated", now),
		},
	}
	router := newPostTestRouter(posts, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c", bytes.NewBufferString(`{
		"title": "Updated",
		"body": "Updated body"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !posts.updateCalled {
		t.Fatal("expected UpdatePost to be called")
	}
	if posts.updateInput.ActorID != userID {
		t.Fatalf("expected actor %q, got %q", userID.String(), posts.updateInput.ActorID.String())
	}
	if posts.updateInput.Title != "Updated" || posts.updateInput.Body != "Updated body" {
		t.Fatalf("unexpected update input: %#v", posts.updateInput)
	}

	var response getPostResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Post.Title != "Updated" || response.Post.BodyFormat != "markdown" {
		t.Fatalf("unexpected update response: %#v", response.Post)
	}
}

func TestDeletePostReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	posts := &fakePostUseCase{}
	router := newPostTestRouter(posts, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, recorder.Code, recorder.Body.String())
	}
	if !posts.deleteCalled {
		t.Fatal("expected DeletePost to be called")
	}
	if posts.deleteInput.ActorID != userID {
		t.Fatalf("expected actor %q, got %q", userID.String(), posts.deleteInput.ActorID.String())
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", recorder.Body.String())
	}
}

func TestListLatestPostsReturnsPosts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 1, 13, 30, 0, 0, time.UTC)
	posts := &fakePostUseCase{
		listLatestResult: postusecase.ListLatestPostsResult{
			Posts:  []postusecase.Post{newPostResult("Latest", now)},
			Limit:  20,
			Offset: 5,
		},
	}
	router := newPostTestRouter(posts, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/posts?sort=hot&limit=20&offset=5", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !posts.listLatestCalled {
		t.Fatal("expected ListLatestPosts to be called")
	}
	if posts.listLatestInput.Limit != 20 || posts.listLatestInput.Offset != 5 {
		t.Fatalf("expected limit=20 offset=5, got limit=%d offset=%d", posts.listLatestInput.Limit, posts.listLatestInput.Offset)
	}
	if posts.listLatestInput.Sort != "hot" {
		t.Fatalf("expected sort hot, got %q", posts.listLatestInput.Sort)
	}
	if posts.listLatestInput.ViewerID != userID {
		t.Fatalf("expected viewer %q, got %q", userID.String(), posts.listLatestInput.ViewerID.String())
	}

	var response listCommunityPostsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Posts) != 1 || response.Posts[0].Title != "Latest" {
		t.Fatalf("unexpected latest posts response: %#v", response.Posts)
	}
	if response.Posts[0].BodyFormat != "markdown" {
		t.Fatalf("expected body_format markdown, got %q", response.Posts[0].BodyFormat)
	}
}

func TestListLatestPostsAllowsAnonymousViewer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 1, 13, 30, 0, 0, time.UTC)
	posts := &fakePostUseCase{
		listLatestResult: postusecase.ListLatestPostsResult{
			Posts: []postusecase.Post{newPostResult("Latest anonymous", now)},
			Limit: 20,
		},
	}
	router := newPostTestRouter(posts, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/posts?sort=new&limit=20", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !posts.listLatestCalled {
		t.Fatal("expected ListLatestPosts to be called")
	}
	if posts.listLatestInput.ViewerID.String() != "" {
		t.Fatalf("expected empty anonymous viewer, got %q", posts.listLatestInput.ViewerID.String())
	}
}

func TestListLatestPostsMapsInvalidSortError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	posts := &fakePostUseCase{
		listLatestErr: apperr.New(apperr.CodeInvalidArgument, "post list sort is invalid"),
	}
	router := newPostTestRouter(posts, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/posts?sort=popular", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	if posts.listLatestInput.Sort != "popular" {
		t.Fatalf("expected sort popular, got %q", posts.listLatestInput.Sort)
	}
	assertPostErrorCode(t, recorder, apperr.CodeInvalidArgument)
}

func TestPostReadRoutesRejectInvalidAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	posts := &fakePostUseCase{}
	router := newPostTestRouter(posts, &fakeAccessTokenParser{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/campus/posts", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertPostErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if posts.publishCalled || posts.listCalled || posts.listLatestCalled || posts.getCalled || posts.updateCalled || posts.deleteCalled {
		t.Fatal("post usecase should not be called for invalid auth")
	}
}

func TestPostWriteRoutesRequireAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	posts := &fakePostUseCase{}
	router := newPostTestRouter(posts, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/communities/campus/posts", bytes.NewBufferString(`{
		"title": "Hello",
		"body": "Post body"
	}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertPostErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if posts.publishCalled || posts.listCalled || posts.listLatestCalled || posts.getCalled || posts.updateCalled || posts.deleteCalled {
		t.Fatal("post usecase should not be called without auth on write route")
	}
}

func TestListCommunityPostsRejectsInvalidQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	posts := &fakePostUseCase{}
	router := newPostTestRouter(posts, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/campus/posts?limit=abc", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertPostErrorCode(t, recorder, apperr.CodeInvalidArgument)
	if posts.listCalled {
		t.Fatal("post usecase should not be called for invalid query")
	}
}

func TestPostUseCaseErrorMapsToHTTPError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	posts := &fakePostUseCase{
		publishErr: apperr.New(apperr.CodeForbidden, "can't post in community"),
	}
	router := newPostTestRouter(posts, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/communities/campus/posts", bytes.NewBufferString(`{
		"title": "Hello",
		"body": "Post body"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, recorder.Code, recorder.Body.String())
	}
	assertPostErrorCode(t, recorder, apperr.CodeForbidden)
}

type fakePostUseCase struct {
	publishCalled    bool
	listCalled       bool
	listLatestCalled bool
	getCalled        bool
	updateCalled     bool
	deleteCalled     bool
	publishInput     postusecase.PublishPostInput
	listInput        postusecase.ListCommunityPostsInput
	listLatestInput  postusecase.ListLatestPostsInput
	getInput         postusecase.GetPostInput
	updateInput      postusecase.UpdatePostInput
	deleteInput      postusecase.DeletePostInput
	publishResult    postusecase.PublishPostResult
	listResult       postusecase.ListCommunityPostsResult
	listLatestResult postusecase.ListLatestPostsResult
	getResult        postusecase.GetPostResult
	updateResult     postusecase.UpdatePostResult
	deleteResult     postusecase.DeletePostResult
	publishErr       error
	listErr          error
	listLatestErr    error
	getErr           error
	updateErr        error
	deleteErr        error
}

func (f *fakePostUseCase) PublishPost(ctx context.Context, input postusecase.PublishPostInput) (postusecase.PublishPostResult, error) {
	f.publishCalled = true
	f.publishInput = input
	return f.publishResult, f.publishErr
}

func (f *fakePostUseCase) ListCommunityPosts(ctx context.Context, input postusecase.ListCommunityPostsInput) (postusecase.ListCommunityPostsResult, error) {
	f.listCalled = true
	f.listInput = input
	return f.listResult, f.listErr
}

func (f *fakePostUseCase) ListLatestPosts(ctx context.Context, input postusecase.ListLatestPostsInput) (postusecase.ListLatestPostsResult, error) {
	f.listLatestCalled = true
	f.listLatestInput = input
	return f.listLatestResult, f.listLatestErr
}

func (f *fakePostUseCase) GetPost(ctx context.Context, input postusecase.GetPostInput) (postusecase.GetPostResult, error) {
	f.getCalled = true
	f.getInput = input
	return f.getResult, f.getErr
}

func (f *fakePostUseCase) UpdatePost(ctx context.Context, input postusecase.UpdatePostInput) (postusecase.UpdatePostResult, error) {
	f.updateCalled = true
	f.updateInput = input
	return f.updateResult, f.updateErr
}

func (f *fakePostUseCase) DeletePost(ctx context.Context, input postusecase.DeletePostInput) (postusecase.DeletePostResult, error) {
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

func newPostTestRouter(posts PostUseCase, parser authhttp.AccessTokenParser) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())

	handler := NewHandler(posts)
	publicRead := router.Group("/api/v1")
	publicRead.Use(authhttp.OptionalAuth(parser))
	RegisterReadRoutes(publicRead, handler)

	protected := router.Group("/api/v1")
	protected.Use(authhttp.RequireAuth(parser))
	RegisterWriteRoutes(protected, handler)

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

func newPostResult(title string, now time.Time) postusecase.Post {
	return postusecase.Post{
		ID:            "8f92e975-5323-4a58-bac1-1336b668183c",
		CommunityID:   userdomain.NewGeneratedUserID().String(),
		AuthorID:      userdomain.NewGeneratedUserID().String(),
		Title:         title,
		Body:          "Post body",
		BodyFormat:    "markdown",
		Status:        "visible",
		UpvoteCount:   2,
		DownvoteCount: 1,
		Score:         1,
		MyVote:        1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func newPostResultWithAttachment(title string, now time.Time) postusecase.Post {
	post := newPostResult(title, now)
	post.Attachments = []postusecase.Attachment{
		{
			ID:        "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a",
			Kind:      "image",
			URL:       "http://localhost:8080/uploads/images/test.png",
			SizeBytes: 100,
			MimeType:  "image/png",
			Status:    "ready",
			CreatedAt: now,
		},
	}
	return post
}

func assertPostErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode apperr.Code) {
	t.Helper()

	var response httpserver.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != string(wantCode) {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
}

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
		"attachment_ids": ["98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a"],
		"content_refs": [
			{"kind": "image", "ref_id": "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a"},
			{"kind": "link_preview", "ref_id": "https://example.com/campus"}
		]
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
	if len(posts.publishInput.ContentRefs) != 2 ||
		posts.publishInput.ContentRefs[0].Kind != "image" ||
		posts.publishInput.ContentRefs[0].RefID != "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a" ||
		posts.publishInput.ContentRefs[1].Kind != "link_preview" ||
		posts.publishInput.ContentRefs[1].RefID != "https://example.com/campus" {
		t.Fatalf("unexpected content refs: %#v", posts.publishInput.ContentRefs)
	}

	var response publishPostResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Post.Title != "Hello" {
		t.Fatalf("expected title Hello, got %q", response.Post.Title)
	}
	if response.Post.Format != "nexus_markdown" {
		t.Fatalf("expected format nexus_markdown, got %q", response.Post.Format)
	}
	if len(response.Post.Attachments) != 1 || response.Post.Attachments[0].URL == "" {
		t.Fatalf("expected attachment response, got %#v", response.Post.Attachments)
	}
	if response.Post.Attachments[0].ThumbnailURL != response.Post.Attachments[0].URL {
		t.Fatalf("expected attachment thumbnail_url fallback, got %#v", response.Post.Attachments[0])
	}
	if len(response.Post.ContentRefs) != 2 || response.Post.ContentRefs[1].RefID != "https://example.com/campus" {
		t.Fatalf("expected content refs in response, got %#v", response.Post.ContentRefs)
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
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/campus/posts?sort=hot&t=week&limit=20&offset=5", nil)
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
	if posts.listInput.TimeRange != "week" {
		t.Fatalf("expected time range week, got %q", posts.listInput.TimeRange)
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
	if response.Posts[0].Format != "nexus_markdown" {
		t.Fatalf("expected format nexus_markdown, got %q", response.Posts[0].Format)
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
	if posts.updateInput.AttachmentIDs != nil {
		t.Fatalf("expected omitted attachment_ids to stay nil, got %#v", *posts.updateInput.AttachmentIDs)
	}
	if posts.updateInput.ContentRefs != nil {
		t.Fatalf("expected omitted content_refs to stay nil, got %#v", *posts.updateInput.ContentRefs)
	}

	var response getPostResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Post.Title != "Updated" || response.Post.Format != "nexus_markdown" {
		t.Fatalf("unexpected update response: %#v", response.Post)
	}
}

func TestUpdatePostAcceptsAttachmentIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	updatedPost := newPostResultWithAttachment("Updated", now)
	updatedPost.ContentRefs = []postusecase.ContentRef{
		{Kind: "image", RefID: "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a"},
	}
	posts := &fakePostUseCase{
		updateResult: postusecase.UpdatePostResult{
			Post: updatedPost,
		},
	}
	router := newPostTestRouter(posts, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c", bytes.NewBufferString(`{
		"title": "Updated",
		"body": "Updated body",
		"attachment_ids": ["98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a"],
		"content_refs": [
			{"kind": "image", "ref_id": "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a"}
		]
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if posts.updateInput.AttachmentIDs == nil || len(*posts.updateInput.AttachmentIDs) != 1 || (*posts.updateInput.AttachmentIDs)[0] != "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a" {
		t.Fatalf("unexpected attachment ids: %#v", posts.updateInput.AttachmentIDs)
	}
	if posts.updateInput.ContentRefs == nil || len(*posts.updateInput.ContentRefs) != 1 || (*posts.updateInput.ContentRefs)[0].Kind != "image" || (*posts.updateInput.ContentRefs)[0].RefID != "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a" {
		t.Fatalf("unexpected content refs: %#v", posts.updateInput.ContentRefs)
	}

	var response getPostResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Post.Attachments) != 1 || response.Post.Attachments[0].ID != "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a" {
		t.Fatalf("expected attachment response, got %#v", response.Post.Attachments)
	}
	if len(response.Post.ContentRefs) != 1 || response.Post.ContentRefs[0].Kind != "image" {
		t.Fatalf("expected content ref response, got %#v", response.Post.ContentRefs)
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

func TestSavePostReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	posts := &fakePostUseCase{}
	router := newPostTestRouter(posts, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/save", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, recorder.Code, recorder.Body.String())
	}
	if !posts.saveCalled {
		t.Fatal("expected SavePost to be called")
	}
	if posts.saveInput.PostID != "8f92e975-5323-4a58-bac1-1336b668183c" || posts.saveInput.UserID != userID {
		t.Fatalf("unexpected save input: %#v", posts.saveInput)
	}
}

func TestListSavedPostsReturnsPosts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	posts := &fakePostUseCase{
		listSavedResult: postusecase.ListSavedPostsResult{
			Posts:  []postusecase.Post{newPostResult("Saved", now)},
			Limit:  20,
			Offset: 5,
		},
	}
	router := newPostTestRouter(posts, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/saved-posts?limit=20&offset=5", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !posts.listSavedCalled {
		t.Fatal("expected ListSavedPosts to be called")
	}
	if posts.listSavedInput.UserID != userID || posts.listSavedInput.Limit != 20 || posts.listSavedInput.Offset != 5 {
		t.Fatalf("unexpected list saved input: %#v", posts.listSavedInput)
	}
	var response listCommunityPostsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Posts) != 1 || response.Posts[0].Title != "Saved" {
		t.Fatalf("unexpected saved posts response: %#v", response.Posts)
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
	request := httptest.NewRequest(http.MethodGet, "/api/v1/posts?source=recommended&sort=hot&t=day&limit=20&offset=5", nil)
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
	if posts.listLatestInput.Source != "recommended" || posts.listLatestInput.TimeRange != "day" {
		t.Fatalf("unexpected feed source/time range: %#v", posts.listLatestInput)
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
	if response.Posts[0].Format != "nexus_markdown" {
		t.Fatalf("expected format nexus_markdown, got %q", response.Posts[0].Format)
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

func TestListUserPostsAllowsAnonymousViewer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 1, 13, 45, 0, 0, time.UTC)
	posts := &fakePostUseCase{
		listUserResult: postusecase.ListUserPostsResult{
			Posts: []postusecase.Post{newPostResult("User post", now)},
			Limit: 20,
		},
	}
	router := newPostTestRouter(posts, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/alice/posts?sort=new&limit=20", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !posts.listUserCalled {
		t.Fatal("expected ListUserPosts to be called")
	}
	if posts.listUserInput.Username != "alice" || posts.listUserInput.Sort != "new" || posts.listUserInput.Limit != 20 {
		t.Fatalf("unexpected list user posts input: %#v", posts.listUserInput)
	}
	if posts.listUserInput.ViewerID.String() != "" {
		t.Fatalf("expected empty anonymous viewer, got %q", posts.listUserInput.ViewerID.String())
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
	listUserCalled   bool
	listSavedCalled  bool
	getCalled        bool
	saveCalled       bool
	deleteSaveCalled bool
	updateCalled     bool
	deleteCalled     bool
	publishInput     postusecase.PublishPostInput
	listInput        postusecase.ListCommunityPostsInput
	listLatestInput  postusecase.ListLatestPostsInput
	listUserInput    postusecase.ListUserPostsInput
	listSavedInput   postusecase.ListSavedPostsInput
	getInput         postusecase.GetPostInput
	saveInput        postusecase.SavePostInput
	deleteSaveInput  postusecase.DeletePostSaveInput
	updateInput      postusecase.UpdatePostInput
	deleteInput      postusecase.DeletePostInput
	publishResult    postusecase.PublishPostResult
	listResult       postusecase.ListCommunityPostsResult
	listLatestResult postusecase.ListLatestPostsResult
	listUserResult   postusecase.ListUserPostsResult
	listSavedResult  postusecase.ListSavedPostsResult
	getResult        postusecase.GetPostResult
	saveResult       postusecase.SavePostResult
	deleteSaveResult postusecase.DeletePostSaveResult
	updateResult     postusecase.UpdatePostResult
	deleteResult     postusecase.DeletePostResult
	publishErr       error
	listErr          error
	listLatestErr    error
	listUserErr      error
	listSavedErr     error
	getErr           error
	saveErr          error
	deleteSaveErr    error
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

func (f *fakePostUseCase) ListUserPosts(ctx context.Context, input postusecase.ListUserPostsInput) (postusecase.ListUserPostsResult, error) {
	f.listUserCalled = true
	f.listUserInput = input
	return f.listUserResult, f.listUserErr
}

func (f *fakePostUseCase) ListSavedPosts(ctx context.Context, input postusecase.ListSavedPostsInput) (postusecase.ListSavedPostsResult, error) {
	f.listSavedCalled = true
	f.listSavedInput = input
	return f.listSavedResult, f.listSavedErr
}

func (f *fakePostUseCase) GetPost(ctx context.Context, input postusecase.GetPostInput) (postusecase.GetPostResult, error) {
	f.getCalled = true
	f.getInput = input
	return f.getResult, f.getErr
}

func (f *fakePostUseCase) SavePost(ctx context.Context, input postusecase.SavePostInput) (postusecase.SavePostResult, error) {
	f.saveCalled = true
	f.saveInput = input
	return f.saveResult, f.saveErr
}

func (f *fakePostUseCase) DeletePostSave(ctx context.Context, input postusecase.DeletePostSaveInput) (postusecase.DeletePostSaveResult, error) {
	f.deleteSaveCalled = true
	f.deleteSaveInput = input
	return f.deleteSaveResult, f.deleteSaveErr
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
		Format:        "nexus_markdown",
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
			ID:           "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a",
			Kind:         "image",
			URL:          "http://localhost:8080/uploads/images/test.png",
			ThumbnailURL: "http://localhost:8080/uploads/images/test.png",
			SizeBytes:    100,
			MimeType:     "image/png",
			Status:       "ready",
			CreatedAt:    now,
		},
	}
	post.ContentRefs = []postusecase.ContentRef{
		{Kind: "image", RefID: "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a"},
		{Kind: "link_preview", RefID: "https://example.com/campus"},
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

package searchhttp

import (
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
	"github.com/Versifine/cumt-nexus-api/internal/search/searchusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

func TestSearchReturnsResults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 2, 6, 30, 0, 0, time.UTC)
	usecase := &fakeSearchUseCase{
		result: searchusecase.SearchResult{
			Query:  "campus",
			Scope:  searchusecase.ScopeAll.String(),
			Limit:  50,
			Offset: 2,
			Communities: []searchusecase.CommunityResult{{
				ID:          "community-1",
				Slug:        "campus",
				Name:        "Campus",
				Description: "Campus community",
				Kind:        "system",
				Status:      "active",
				Visibility:  "public",
				CreatedAt:   now,
				UpdatedAt:   now,
			}},
			Posts: []searchusecase.PostResult{{
				ID:            "post-1",
				CommunityID:   "community-1",
				CommunitySlug: "campus",
				AuthorID:      "author-1",
				Title:         "Campus notice",
				BodyExcerpt:   "notice body",
				Status:        "visible",
				CreatedAt:     now,
				UpdatedAt:     now,
			}},
			Users: []searchusecase.UserResult{{
				ID:          "user-1",
				Username:    "alice",
				DisplayName: "Alice",
				AvatarURL:   "https://example.com/avatar.jpg",
				Headline:    "Student",
				BioExcerpt:  "Bio",
				Status:      "active",
				CreatedAt:   now,
				UpdatedAt:   now,
			}},
		},
	}
	router := newSearchTestRouter(usecase, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=campus&scope=all&limit=50&offset=2", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !usecase.called {
		t.Fatal("expected search usecase to be called")
	}
	if usecase.input.Query != "campus" || usecase.input.Scope != "all" {
		t.Fatalf("unexpected search input: %#v", usecase.input)
	}
	if usecase.input.Limit != 50 || usecase.input.Offset != 2 {
		t.Fatalf("unexpected pagination input: %#v", usecase.input)
	}

	var response searchResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Query != "campus" || response.Scope != "all" || response.Limit != 50 || response.Offset != 2 {
		t.Fatalf("unexpected response metadata: %#v", response)
	}
	if len(response.Communities) != 1 || response.Communities[0].Slug != "campus" {
		t.Fatalf("unexpected communities response: %#v", response.Communities)
	}
	if len(response.Posts) != 1 || response.Posts[0].BodyExcerpt != "notice body" {
		t.Fatalf("unexpected posts response: %#v", response.Posts)
	}
	if len(response.Users) != 1 || response.Users[0].Username != "alice" || response.Users[0].BioExcerpt != "Bio" {
		t.Fatalf("unexpected users response: %#v", response.Users)
	}
}

func TestSearchRejectsInvalidBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &fakeSearchUseCase{}
	router := newSearchTestRouter(usecase, &fakeAccessTokenParser{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=campus", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertSearchErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if usecase.called {
		t.Fatal("search usecase should not be called for invalid auth")
	}
}

func TestSearchRejectsInvalidQueryParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &fakeSearchUseCase{}
	router := newSearchTestRouter(usecase, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=campus&limit=abc", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertSearchErrorCode(t, recorder, apperr.CodeInvalidArgument)
	if usecase.called {
		t.Fatal("search usecase should not be called for invalid query")
	}
}

func TestSearchPropagatesUseCaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &fakeSearchUseCase{
		err: apperr.New(apperr.CodeInvalidArgument, "search query is required"),
	}
	router := newSearchTestRouter(usecase, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertSearchErrorCode(t, recorder, apperr.CodeInvalidArgument)
}

type fakeSearchUseCase struct {
	called bool
	input  searchusecase.SearchInput
	result searchusecase.SearchResult
	err    error
}

func (f *fakeSearchUseCase) Search(ctx context.Context, input searchusecase.SearchInput) (searchusecase.SearchResult, error) {
	f.called = true
	f.input = input
	return f.result, f.err
}

type fakeAccessTokenParser struct {
	claims *authtoken.AccessTokenClaims
	err    error
}

func (f *fakeAccessTokenParser) ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error) {
	return f.claims, f.err
}

func newSearchTestRouter(usecase *fakeSearchUseCase, parser authhttp.AccessTokenParser) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())

	publicRead := router.Group("/api/v1")
	publicRead.Use(authhttp.OptionalAuth(parser))
	RegisterRoutes(publicRead, NewHandler(usecase))

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

func assertSearchErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode apperr.Code) {
	t.Helper()

	var response httpserver.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != string(wantCode) {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
}

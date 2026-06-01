package communityhttp

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
	"github.com/Versifine/cumt-nexus-api/internal/community/communityusecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

func TestListCommunitiesReturnsCommunities(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	communities := &fakeCommunityReadUseCase{
		listResult: communityusecase.ListCommunitiesResult{
			Communities: []communityusecase.Community{
				newCommunityResult("public", now),
				newCommunityResult("campus", now.Add(time.Minute)),
			},
		},
	}
	router := newCommunityTestRouter(communities, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.listCalled {
		t.Fatal("expected ListCommunities to be called")
	}

	var response listCommunitiesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Communities) != 2 {
		t.Fatalf("expected two communities, got %d", len(response.Communities))
	}
	if response.Communities[0].Slug != "public" || response.Communities[1].Slug != "campus" {
		t.Fatalf("unexpected communities response: %#v", response.Communities)
	}
}

func TestGetCommunityReturnsCommunity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	communities := &fakeCommunityReadUseCase{
		getResult: communityusecase.GetCommunityResult{
			Community: newCommunityResult("campus", now),
		},
	}
	router := newCommunityTestRouter(communities, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/campus", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.getCalled {
		t.Fatal("expected GetCommunityBySlug to be called")
	}
	if communities.getInput.Slug != "campus" {
		t.Fatalf("expected slug campus, got %q", communities.getInput.Slug)
	}

	var response getCommunityResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Community.Slug != "campus" {
		t.Fatalf("expected response slug campus, got %q", response.Community.Slug)
	}
}

func TestCommunityRoutesRejectInvalidAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	communities := &fakeCommunityReadUseCase{}
	router := newCommunityTestRouter(communities, &fakeAccessTokenParser{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertCommunityErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if communities.listCalled || communities.getCalled {
		t.Fatal("community usecase should not be called for invalid auth")
	}
}

func TestGetCommunityUseCaseErrorMapsToHTTPError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   apperr.Code
	}{
		{
			name:       "invalid slug",
			err:        apperr.New(apperr.CodeInvalidArgument, "community slug is invalid"),
			wantStatus: http.StatusBadRequest,
			wantCode:   apperr.CodeInvalidArgument,
		},
		{
			name:       "not found",
			err:        apperr.New(apperr.CodeNotFound, "community not found"),
			wantStatus: http.StatusNotFound,
			wantCode:   apperr.CodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			communities := &fakeCommunityReadUseCase{
				getErr: tt.err,
			}
			router := newCommunityTestRouter(communities, validParser())

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/bad-slug", nil)
			request.Header.Set("Authorization", "Bearer valid-token")

			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.wantStatus, recorder.Code, recorder.Body.String())
			}
			assertCommunityErrorCode(t, recorder, tt.wantCode)
			if !communities.getCalled {
				t.Fatal("expected GetCommunityBySlug to be called")
			}
		})
	}
}

type fakeCommunityReadUseCase struct {
	listCalled bool
	getCalled  bool
	getInput   communityusecase.GetCommunityInput
	listResult communityusecase.ListCommunitiesResult
	getResult  communityusecase.GetCommunityResult
	listErr    error
	getErr     error
}

func (f *fakeCommunityReadUseCase) ListCommunities(ctx context.Context) (communityusecase.ListCommunitiesResult, error) {
	f.listCalled = true
	return f.listResult, f.listErr
}

func (f *fakeCommunityReadUseCase) GetCommunityBySlug(ctx context.Context, input communityusecase.GetCommunityInput) (communityusecase.GetCommunityResult, error) {
	f.getCalled = true
	f.getInput = input
	return f.getResult, f.getErr
}

type fakeAccessTokenParser struct {
	claims *authtoken.AccessTokenClaims
	err    error
}

func (f *fakeAccessTokenParser) ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error) {
	return f.claims, f.err
}

func newCommunityTestRouter(communities CommunityReadUseCase, parser authhttp.AccessTokenParser) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())

	protected := router.Group("/api/v1")
	protected.Use(authhttp.RequireAuth(parser))
	RegisterRoutes(protected, NewHandler(communities))

	return router
}

func validParser() *fakeAccessTokenParser {
	return &fakeAccessTokenParser{
		claims: &authtoken.AccessTokenClaims{
			UserID: userdomain.NewGeneratedUserID(),
		},
	}
}

func newCommunityResult(slug string, now time.Time) communityusecase.Community {
	return communityusecase.Community{
		ID:          userdomain.NewGeneratedUserID().String(),
		Slug:        slug,
		Name:        slug,
		Description: "test community",
		Kind:        "system",
		Status:      "active",
		Visibility:  "public",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func assertCommunityErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode apperr.Code) {
	t.Helper()

	var response httpserver.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != string(wantCode) {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
}

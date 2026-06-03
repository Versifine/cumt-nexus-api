package userhttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authtoken"
	"github.com/Versifine/cumt-nexus-api/internal/auth/delivery/authhttp"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userusecase"
	"github.com/gin-gonic/gin"
)

func TestMeReturnsCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	createdAt := time.Date(2026, 5, 30, 10, 30, 0, 0, time.UTC)
	currentUser := &fakeCurrentUserUseCase{
		result: userusecase.CurrentUserResult{
			User: userusecase.CurrentUser{
				ID:              userID.String(),
				Username:        "alice",
				Status:          "active",
				IsPlatformStaff: true,
				CreatedAt:       createdAt,
			},
		},
	}
	parser := &fakeAccessTokenParser{
		claims: &authtoken.AccessTokenClaims{
			UserID: userID,
		},
	}
	router := newMeTestRouter(currentUser, parser)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !currentUser.called {
		t.Fatal("expected current user usecase to be called")
	}
	if currentUser.input.UserID != userID {
		t.Fatalf("expected current user id %q, got %q", userID.String(), currentUser.input.UserID.String())
	}

	var response currentUserResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.ID != userID.String() {
		t.Fatalf("expected user id %q, got %q", userID.String(), response.ID)
	}
	if response.Username != "alice" {
		t.Fatalf("expected username %q, got %q", "alice", response.Username)
	}
	if response.Status != "active" {
		t.Fatalf("expected status %q, got %q", "active", response.Status)
	}
	if !response.IsPlatformStaff {
		t.Fatal("expected is_platform_staff=true")
	}
	if !response.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected created_at %s, got %s", createdAt, response.CreatedAt)
	}
	if strings.Contains(recorder.Body.String(), "password_hash") {
		t.Fatal("response body must not contain password_hash")
	}
}

func TestMeRejectsInvalidAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		authorization string
		parserErr     error
	}{
		{
			name: "missing authorization header",
		},
		{
			name:          "invalid token",
			authorization: "Bearer invalid-token",
			parserErr:     errors.New("parse failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentUser := &fakeCurrentUserUseCase{}
			parser := &fakeAccessTokenParser{
				err: tt.parserErr,
			}
			router := newMeTestRouter(currentUser, parser)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
			if tt.authorization != "" {
				request.Header.Set("Authorization", tt.authorization)
			}

			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
			}
			assertMeErrorCode(t, recorder, apperr.CodeUnauthenticated)
			if currentUser.called {
				t.Fatal("current user usecase should not be called for invalid auth")
			}
		})
	}
}

func TestMeUseCaseErrorMapsToHTTPError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	currentUser := &fakeCurrentUserUseCase{
		err: apperr.New(apperr.CodeForbidden, "user is forbidden"),
	}
	parser := &fakeAccessTokenParser{
		claims: &authtoken.AccessTokenClaims{
			UserID: userID,
		},
	}
	router := newMeTestRouter(currentUser, parser)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, recorder.Code, recorder.Body.String())
	}
	assertMeErrorCode(t, recorder, apperr.CodeForbidden)
	if !currentUser.called {
		t.Fatal("expected current user usecase to be called")
	}
}

type fakeCurrentUserUseCase struct {
	called bool
	input  userusecase.CurrentUserInput
	result userusecase.CurrentUserResult
	err    error
}

func (f *fakeCurrentUserUseCase) GetCurrentUser(ctx context.Context, input userusecase.CurrentUserInput) (userusecase.CurrentUserResult, error) {
	f.called = true
	f.input = input
	return f.result, f.err
}

type fakeAccessTokenParser struct {
	called   bool
	rawToken string
	claims   *authtoken.AccessTokenClaims
	err      error
}

func (f *fakeAccessTokenParser) ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error) {
	f.called = true
	f.rawToken = rawToken
	return f.claims, f.err
}

func newMeTestRouter(currentUser CurrentUserUseCase, parser authhttp.AccessTokenParser) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())

	protected := router.Group("/api/v1")
	protected.Use(authhttp.RequireAuth(parser))
	RegisterRoutes(protected, NewHandler(currentUser))

	return router
}

func assertMeErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode apperr.Code) {
	t.Helper()

	var response httpserver.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != string(wantCode) {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
}

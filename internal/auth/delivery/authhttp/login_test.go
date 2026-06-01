package authhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authusecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	"github.com/gin-gonic/gin"
)

func TestLoginReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	createdAt := time.Date(2026, 5, 29, 10, 30, 0, 0, time.UTC)
	login := &fakeLoginUseCase{
		result: authusecase.LoginResult{
			AccessToken: "access-token",
			TokenType:   "Bearer",
			ExpiresIn:   86400,
			User: authusecase.LoginUser{
				ID:        "9c40e96a-7b3f-4652-a77b-d4b69bb61d2e",
				Username:  "alice",
				Status:    "active",
				CreatedAt: createdAt,
			},
		},
	}
	router := newLoginTestRouter(login)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{
		"username": " Alice ",
		"password": "password123"
	}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !login.called {
		t.Fatal("expected login usecase to be called")
	}
	if login.input.Username != " Alice " {
		t.Fatalf("expected username passed through, got %q", login.input.Username)
	}
	if login.input.Password != "password123" {
		t.Fatalf("expected password passed through, got %q", login.input.Password)
	}

	var response loginResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal login response: %v", err)
	}
	if response.AccessToken != "access-token" {
		t.Fatalf("expected access_token %q, got %q", "access-token", response.AccessToken)
	}
	if response.TokenType != "Bearer" {
		t.Fatalf("expected token_type %q, got %q", "Bearer", response.TokenType)
	}
	if response.ExpiresIn != 86400 {
		t.Fatalf("expected expires_in %d, got %d", 86400, response.ExpiresIn)
	}
	if response.User.ID != "9c40e96a-7b3f-4652-a77b-d4b69bb61d2e" {
		t.Fatalf("expected user id %q, got %q", "9c40e96a-7b3f-4652-a77b-d4b69bb61d2e", response.User.ID)
	}
	if response.User.Username != "alice" {
		t.Fatalf("expected username %q, got %q", "alice", response.User.Username)
	}
	if response.User.Status != "active" {
		t.Fatalf("expected status %q, got %q", "active", response.User.Status)
	}
	if !response.User.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected created_at %s, got %s", createdAt, response.User.CreatedAt)
	}
	if strings.Contains(recorder.Body.String(), "password_hash") {
		t.Fatal("response body must not contain password_hash")
	}
}

func TestLoginInvalidRequestReturnsInvalidArgument(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		body string
	}{
		{
			name: "malformed json",
			body: `{"username": "alice",`,
		},
		{
			name: "missing username",
			body: `{"password": "password123"}`,
		},
		{
			name: "missing password",
			body: `{"username": "alice"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			login := &fakeLoginUseCase{}
			router := newLoginTestRouter(login)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
			}
			assertErrorResponse(t, recorder, string(apperr.CodeInvalidArgument), "invalid login request")
			if login.called {
				t.Fatal("login usecase should not be called for invalid request")
			}
		})
	}
}

func TestLoginUseCaseErrorMapsToHTTPError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   apperr.Code
	}{
		{
			name:       "unauthenticated",
			err:        apperr.New(apperr.CodeUnauthenticated, "invalid username or password"),
			wantStatus: http.StatusUnauthorized,
			wantCode:   apperr.CodeUnauthenticated,
		},
		{
			name:       "forbidden",
			err:        apperr.New(apperr.CodeForbidden, "user is forbidden"),
			wantStatus: http.StatusForbidden,
			wantCode:   apperr.CodeForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			login := &fakeLoginUseCase{
				err: tt.err,
			}
			router := newLoginTestRouter(login)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{
				"username": "alice",
				"password": "password123"
			}`))
			request.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.wantStatus, recorder.Code, recorder.Body.String())
			}
			assertErrorCode(t, recorder, tt.wantCode)
			if !login.called {
				t.Fatal("expected login usecase to be called")
			}
		})
	}
}

type fakeLoginUseCase struct {
	called bool
	input  authusecase.LoginInput
	result authusecase.LoginResult
	err    error
}

func (f *fakeLoginUseCase) Login(ctx context.Context, input authusecase.LoginInput) (authusecase.LoginResult, error) {
	f.called = true
	f.input = input
	return f.result, f.err
}

func newLoginTestRouter(login LoginUseCase) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())
	RegisterRoutes(router.Group("/api/v1/auth"), NewHandler(nil, login))
	return router
}

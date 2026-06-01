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

func TestRegisterReturnsCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	createdAt := time.Date(2026, 5, 28, 10, 30, 0, 0, time.UTC)
	register := &fakeRegisterUseCase{
		result: authusecase.RegisterResult{
			AccessToken: "access-token",
			TokenType:   "Bearer",
			ExpiresIn:   86400,
			User: authusecase.RegisterUser{
				ID:        "9c40e96a-7b3f-4652-a77b-d4b69bb61d2e",
				Username:  "alice",
				Status:    "active",
				CreatedAt: createdAt,
			},
		},
	}
	router := newRegisterTestRouter(register)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{
		"username": " Alice ",
		"password": "password123"
	}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if !register.called {
		t.Fatal("expected register usecase to be called")
	}
	if register.input.Username != " Alice " {
		t.Fatalf("expected username passed through, got %q", register.input.Username)
	}
	if register.input.Password != "password123" {
		t.Fatalf("expected password passed through, got %q", register.input.Password)
	}

	var response registerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal register response: %v", err)
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

func TestRegisterInvalidRequestReturnsInvalidArgument(t *testing.T) {
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
			register := &fakeRegisterUseCase{}
			router := newRegisterTestRouter(register)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
			}
			assertErrorResponse(t, recorder, string(apperr.CodeInvalidArgument), "invalid register request")
			if register.called {
				t.Fatal("register usecase should not be called for invalid request")
			}
		})
	}
}

func TestRegisterUseCaseConflictReturnsConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)

	register := &fakeRegisterUseCase{
		err: apperr.New(apperr.CodeConflict, "username already exists"),
	}
	router := newRegisterTestRouter(register)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{
		"username": "alice",
		"password": "password123"
	}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, recorder.Code, recorder.Body.String())
	}
	assertErrorResponse(t, recorder, string(apperr.CodeConflict), "username already exists")
	if !register.called {
		t.Fatal("expected register usecase to be called")
	}
}

type fakeRegisterUseCase struct {
	called bool
	input  authusecase.RegisterInput
	result authusecase.RegisterResult
	err    error
}

func (f *fakeRegisterUseCase) Register(ctx context.Context, input authusecase.RegisterInput) (authusecase.RegisterResult, error) {
	f.called = true
	f.input = input
	return f.result, f.err
}

func newRegisterTestRouter(register RegisterUseCase) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())
	RegisterRoutes(router.Group("/api/v1/auth"), NewHandler(register, nil))
	return router
}

func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantCode string, wantMessage string) {
	t.Helper()

	var response httpserver.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != wantCode {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
	if response.Error.Message != wantMessage {
		t.Fatalf("expected error message %q, got %q", wantMessage, response.Error.Message)
	}
}

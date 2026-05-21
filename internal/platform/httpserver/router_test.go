package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/gin-gonic/gin"
)

func TestNewRouterHealthz(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := NewRouter(newDiscardLogger())
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected status body ok, got %q", body["status"])
	}
}

func TestRecoveryMiddlewareReturnsInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := newMiddlewareTestRouter()
	router.GET("/panic", func(c *gin.Context) {
		panic("secret panic detail")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	assertErrorResponse(t, recorder, string(apperr.CodeInternal), "internal server error")

	if strings.Contains(recorder.Body.String(), "secret panic detail") {
		t.Fatal("response body leaked panic detail")
	}
}

func TestErrorMiddlewareMapsAppErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		code       apperr.Code
		message    string
		wantStatus int
	}{
		{
			name:       "invalid argument",
			code:       apperr.CodeInvalidArgument,
			message:    "invalid request",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthenticated",
			code:       apperr.CodeUnauthenticated,
			message:    "login required",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "forbidden",
			code:       apperr.CodeForbidden,
			message:    "permission denied",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "not found",
			code:       apperr.CodeNotFound,
			message:    "post not found",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "conflict",
			code:       apperr.CodeConflict,
			message:    "state conflict",
			wantStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newMiddlewareTestRouter()
			router.GET("/error", func(c *gin.Context) {
				_ = c.Error(apperr.New(tt.code, tt.message))
				c.Abort()
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/error", nil)

			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, recorder.Code)
			}
			assertErrorResponse(t, recorder, string(tt.code), tt.message)
		})
	}
}

func TestErrorMiddlewareHidesUnknownError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := newMiddlewareTestRouter()
	router.GET("/unknown", func(c *gin.Context) {
		_ = c.Error(errors.New("database password leaked"))
		c.Abort()
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/unknown", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	assertErrorResponse(t, recorder, string(apperr.CodeInternal), "internal server error")

	if strings.Contains(recorder.Body.String(), "database password leaked") {
		t.Fatal("response body leaked unknown error detail")
	}
}

func newMiddlewareTestRouter() *gin.Engine {
	router := gin.New()
	router.Use(RecoveryMiddleware())
	router.Use(ErrorMiddleware())
	return router
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantCode string, wantMessage string) {
	t.Helper()

	var response ErrorResponse
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

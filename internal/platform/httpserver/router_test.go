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
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/gin-gonic/gin"
)

func TestNewRouterHealthz(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := NewRouter(newDiscardLogger(), config.HTTPConfig{})
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

	router := newMiddlewareTestRouter(newDiscardLogger())
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
			name:       "account banned",
			code:       apperr.CodeAccountBanned,
			message:    "account is banned",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "login rate limited",
			code:       apperr.CodeLoginRateLimited,
			message:    "login rate limited",
			wantStatus: http.StatusTooManyRequests,
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
			router := newMiddlewareTestRouter(newDiscardLogger())
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

	router := newMiddlewareTestRouter(newDiscardLogger())
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

func TestCORSMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := NewRouter(newDiscardLogger(), config.HTTPConfig{
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	})
	router.GET("/cors-test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/cors-test", nil)
	request.Header.Set("Origin", "http://localhost:5173")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("expected allowed origin header, got %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Expose-Headers"); got != RequestIDHeader {
		t.Fatalf("expected expose request id header, got %q", got)
	}
}

func TestCORSMiddlewareHandlesPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := NewRouter(newDiscardLogger(), config.HTTPConfig{
		CORSAllowedOrigins: []string{"*"},
	})
	router.OPTIONS("/cors-test", func(c *gin.Context) {
		t.Fatal("preflight should be handled by cors middleware")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/cors-test", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	request.Header.Set("Access-Control-Request-Method", http.MethodPut)
	request.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected wildcard origin header, got %q", got)
	}
	allowedMethods := recorder.Header().Get("Access-Control-Allow-Methods")
	for _, method := range []string{http.MethodPut, http.MethodPatch} {
		if !strings.Contains(allowedMethods, method) {
			t.Fatalf("expected %s in allow methods, got %q", method, allowedMethods)
		}
	}
}

func newMiddlewareTestRouter(log *slog.Logger) *gin.Engine {
	router := gin.New()
	router.Use(RecoveryMiddleware(log))
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

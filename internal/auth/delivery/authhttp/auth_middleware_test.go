package authhttp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authtoken"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

func TestRequireAuthRejectsInvalidAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		authorization string
		parserErr     error
		wantCalled    bool
	}{
		{
			name:       "missing authorization header",
			wantCalled: false,
		},
		{
			name:          "non bearer authorization header",
			authorization: "Basic abc",
			wantCalled:    false,
		},
		{
			name:          "empty bearer token",
			authorization: "Bearer ",
			wantCalled:    false,
		},
		{
			name:          "parser error",
			authorization: "Bearer invalid-token",
			parserErr:     errors.New("parse failed"),
			wantCalled:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &fakeAccessTokenParser{
				err: tt.parserErr,
			}
			router := newRequireAuthTestRouter(parser, func(c *gin.Context) {
				t.Fatal("next handler should not be called")
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authorization != "" {
				request.Header.Set("Authorization", tt.authorization)
			}

			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
			}
			assertErrorCode(t, recorder, apperr.CodeUnauthenticated)
			if parser.called != tt.wantCalled {
				t.Fatalf("expected parser called %v, got %v", tt.wantCalled, parser.called)
			}
		})
	}
}

func TestRequireAuthStoresCurrentUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	parser := &fakeAccessTokenParser{
		claims: &authtoken.AccessTokenClaims{
			UserID: userID,
		},
	}
	var nextCalled bool
	router := newRequireAuthTestRouter(parser, func(c *gin.Context) {
		nextCalled = true

		gotUserID, ok := CurrentUserID(c)
		if !ok {
			t.Fatal("expected current user id in context")
		}
		if gotUserID != userID {
			t.Fatalf("expected current user id %q, got %q", userID.String(), gotUserID.String())
		}

		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, recorder.Code, recorder.Body.String())
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if !parser.called {
		t.Fatal("expected parser to be called")
	}
	if parser.rawToken != "valid-token" {
		t.Fatalf("expected raw token %q, got %q", "valid-token", parser.rawToken)
	}
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

func newRequireAuthTestRouter(parser AccessTokenParser, handler gin.HandlerFunc) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())
	router.GET("/protected", RequireAuth(parser), handler)
	return router
}

func assertErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode apperr.Code) {
	t.Helper()

	var response httpserver.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != string(wantCode) {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
}

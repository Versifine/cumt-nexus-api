package votehttp

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
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/voteusecase"
	"github.com/gin-gonic/gin"
)

func TestSetPostVoteReturnsVote(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 2, 11, 0, 0, 0, time.UTC)
	votes := &fakePostVoteUseCase{
		setResult: voteusecase.SetPostVoteResult{
			Vote: newPostVoteResult(userID, 1, now),
		},
	}
	router := newVoteTestRouter(votes, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/vote", bytes.NewBufferString(`{
		"value": 1
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !votes.setCalled {
		t.Fatal("expected SetPostVote to be called")
	}
	if votes.setInput.UserID != userID {
		t.Fatalf("expected user %q, got %q", userID.String(), votes.setInput.UserID.String())
	}
	if votes.setInput.PostID != "8f92e975-5323-4a58-bac1-1336b668183c" || votes.setInput.Value != 1 {
		t.Fatalf("unexpected set input: %#v", votes.setInput)
	}

	var response setPostVoteResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Vote.Value != 1 || response.Vote.UserID != userID.String() {
		t.Fatalf("unexpected vote response: %#v", response.Vote)
	}
}

func TestDeletePostVoteReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	votes := &fakePostVoteUseCase{}
	router := newVoteTestRouter(votes, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/vote", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, recorder.Code, recorder.Body.String())
	}
	if !votes.deleteCalled {
		t.Fatal("expected DeletePostVote to be called")
	}
	if votes.deleteInput.UserID != userID || votes.deleteInput.PostID != "8f92e975-5323-4a58-bac1-1336b668183c" {
		t.Fatalf("unexpected delete input: %#v", votes.deleteInput)
	}
}

func TestVoteRoutesRejectInvalidAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	votes := &fakePostVoteUseCase{}
	router := newVoteTestRouter(votes, &fakeAccessTokenParser{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/vote", bytes.NewBufferString(`{
		"value": 1
	}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertVoteErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if votes.setCalled || votes.deleteCalled {
		t.Fatal("vote usecase should not be called for invalid auth")
	}
}

func TestSetPostVoteRejectsInvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	votes := &fakePostVoteUseCase{}
	router := newVoteTestRouter(votes, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/vote", bytes.NewBufferString(`{
		"value": 0
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertVoteErrorCode(t, recorder, apperr.CodeInvalidArgument)
	if votes.setCalled {
		t.Fatal("vote usecase should not be called for invalid body")
	}
}

func TestSetPostVoteUseCaseErrorMapsToHTTPError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	votes := &fakePostVoteUseCase{
		setErr: apperr.New(apperr.CodeNotFound, "post not found"),
	}
	router := newVoteTestRouter(votes, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/vote", bytes.NewBufferString(`{
		"value": -1
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
	assertVoteErrorCode(t, recorder, apperr.CodeNotFound)
}

type fakePostVoteUseCase struct {
	setCalled    bool
	deleteCalled bool
	setInput     voteusecase.SetPostVoteInput
	deleteInput  voteusecase.DeletePostVoteInput
	setResult    voteusecase.SetPostVoteResult
	setErr       error
	deleteErr    error
}

func (f *fakePostVoteUseCase) SetPostVote(ctx context.Context, input voteusecase.SetPostVoteInput) (voteusecase.SetPostVoteResult, error) {
	f.setCalled = true
	f.setInput = input
	return f.setResult, f.setErr
}

func (f *fakePostVoteUseCase) DeletePostVote(ctx context.Context, input voteusecase.DeletePostVoteInput) error {
	f.deleteCalled = true
	f.deleteInput = input
	return f.deleteErr
}

type fakeAccessTokenParser struct {
	claims *authtoken.AccessTokenClaims
	err    error
}

func (f *fakeAccessTokenParser) ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error) {
	return f.claims, f.err
}

func newVoteTestRouter(votes PostVoteUseCase, parser authhttp.AccessTokenParser) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())

	protected := router.Group("/api/v1")
	protected.Use(authhttp.RequireAuth(parser))
	RegisterRoutes(protected, NewHandler(votes))

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

func newPostVoteResult(userID userdomain.UserID, value int, now time.Time) voteusecase.PostVote {
	return voteusecase.PostVote{
		PostID:    "8f92e975-5323-4a58-bac1-1336b668183c",
		UserID:    userID.String(),
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func assertVoteErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode apperr.Code) {
	t.Helper()

	var response httpserver.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != string(wantCode) {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
}

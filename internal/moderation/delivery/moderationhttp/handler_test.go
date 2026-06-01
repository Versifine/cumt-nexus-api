package moderationhttp

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
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationusecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

func TestReportPostReturnsCreatedReport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	posts := &fakeReportUseCase{
		reportPostResult: moderationusecase.ReportContentResult{
			Report: newReportResult(moderationdomain.TargetTypePost.String(), "8f92e975-5323-4a58-bac1-1336b668183c", ""),
		},
	}
	router := newModerationTestRouter(posts, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/reports", bytes.NewBufferString(`{
		"reason": "spam"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if !posts.reportPostCalled {
		t.Fatal("expected ReportPost to be called")
	}
	if posts.reportPostInput.ReporterID != userID {
		t.Fatalf("expected reporter %q, got %q", userID.String(), posts.reportPostInput.ReporterID.String())
	}
	if posts.reportPostInput.Reason != "spam" {
		t.Fatalf("expected reason spam, got %q", posts.reportPostInput.Reason)
	}

	var response reportContentResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Report.TargetType != moderationdomain.TargetTypePost.String() || response.Report.PostID == "" {
		t.Fatalf("unexpected report response: %#v", response.Report)
	}
}

func TestReportCommentReturnsCreatedReport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	comments := &fakeReportUseCase{
		reportCommentResult: moderationusecase.ReportContentResult{
			Report: newReportResult(moderationdomain.TargetTypeComment.String(), "", "94509b7b-1b72-4726-ac08-819f0322d065"),
		},
	}
	router := newModerationTestRouter(comments, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/comments/94509b7b-1b72-4726-ac08-819f0322d065/reports", bytes.NewBufferString(`{
		"reason": "abuse"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if !comments.reportCommentCalled {
		t.Fatal("expected ReportComment to be called")
	}
	if comments.reportCommentInput.ReporterID != userID {
		t.Fatalf("expected reporter %q, got %q", userID.String(), comments.reportCommentInput.ReporterID.String())
	}
	if comments.reportCommentInput.Reason != "abuse" {
		t.Fatalf("expected reason abuse, got %q", comments.reportCommentInput.Reason)
	}
}

func TestReportRoutesRejectInvalidAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reports := &fakeReportUseCase{}
	router := newModerationTestRouter(reports, &fakeAccessTokenParser{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/reports", bytes.NewBufferString(`{
		"reason": "spam"
	}`))

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertModerationErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if reports.reportPostCalled || reports.reportCommentCalled {
		t.Fatal("report usecase should not be called for invalid auth")
	}
}

func TestReportPostRejectsInvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reports := &fakeReportUseCase{}
	router := newModerationTestRouter(reports, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/reports", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertModerationErrorCode(t, recorder, apperr.CodeInvalidArgument)
	if reports.reportPostCalled {
		t.Fatal("report usecase should not be called for invalid request")
	}
}

func TestReportPostPropagatesUseCaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reports := &fakeReportUseCase{
		reportPostErr: apperr.New(apperr.CodeNotFound, "post not found"),
	}
	router := newModerationTestRouter(reports, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/reports", bytes.NewBufferString(`{
		"reason": "spam"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
	assertModerationErrorCode(t, recorder, apperr.CodeNotFound)
}

type fakeReportUseCase struct {
	reportPostCalled    bool
	reportCommentCalled bool
	reportPostInput     moderationusecase.ReportPostInput
	reportCommentInput  moderationusecase.ReportCommentInput
	reportPostResult    moderationusecase.ReportContentResult
	reportCommentResult moderationusecase.ReportContentResult
	reportPostErr       error
	reportCommentErr    error
}

func (f *fakeReportUseCase) ReportPost(ctx context.Context, input moderationusecase.ReportPostInput) (moderationusecase.ReportContentResult, error) {
	f.reportPostCalled = true
	f.reportPostInput = input
	return f.reportPostResult, f.reportPostErr
}

func (f *fakeReportUseCase) ReportComment(ctx context.Context, input moderationusecase.ReportCommentInput) (moderationusecase.ReportContentResult, error) {
	f.reportCommentCalled = true
	f.reportCommentInput = input
	return f.reportCommentResult, f.reportCommentErr
}

type fakeAccessTokenParser struct {
	claims *authtoken.AccessTokenClaims
	err    error
}

func (f *fakeAccessTokenParser) ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error) {
	return f.claims, f.err
}

func newModerationTestRouter(reports ReportUseCase, parser authhttp.AccessTokenParser) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())

	protected := router.Group("/api/v1")
	protected.Use(authhttp.RequireAuth(parser))
	RegisterRoutes(protected, NewHandler(reports))

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

func newReportResult(targetType string, postID string, commentID string) moderationusecase.ContentReport {
	now := time.Date(2026, 6, 2, 4, 30, 0, 0, time.UTC)
	return moderationusecase.ContentReport{
		ID:         "2b74a80d-6d57-4d61-a3ae-db5e006766b6",
		TargetType: targetType,
		PostID:     postID,
		CommentID:  commentID,
		ReporterID: userdomain.NewGeneratedUserID().String(),
		Reason:     "spam",
		Status:     moderationdomain.ReportStatusPending.String(),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func assertModerationErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode apperr.Code) {
	t.Helper()

	var response httpserver.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != string(wantCode) {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
}

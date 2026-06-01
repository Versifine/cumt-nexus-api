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

func TestRemovePostReturnsModerationAction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	actions := &fakeReportUseCase{
		removePostResult: moderationusecase.RemoveContentResult{
			Action: newActionResult(moderationdomain.TargetTypePost.String(), "8f92e975-5323-4a58-bac1-1336b668183c", ""),
		},
	}
	router := newModerationTestRouter(actions, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/moderation/remove", bytes.NewBufferString(`{
		"reason": "policy violation"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !actions.removePostCalled {
		t.Fatal("expected RemovePost to be called")
	}
	if actions.removePostInput.ActorID != userID {
		t.Fatalf("expected actor %q, got %q", userID.String(), actions.removePostInput.ActorID.String())
	}
	if actions.removePostInput.Reason != "policy violation" {
		t.Fatalf("expected reason, got %q", actions.removePostInput.Reason)
	}

	var response removeContentResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Action.TargetType != moderationdomain.TargetTypePost.String() || response.Action.Action != moderationdomain.ActionTypeRemove.String() {
		t.Fatalf("unexpected action response: %#v", response.Action)
	}
}

func TestRemoveCommentReturnsModerationAction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	actions := &fakeReportUseCase{
		removeCommentResult: moderationusecase.RemoveContentResult{
			Action: newActionResult(moderationdomain.TargetTypeComment.String(), "", "94509b7b-1b72-4726-ac08-819f0322d065"),
		},
	}
	router := newModerationTestRouter(actions, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/comments/94509b7b-1b72-4726-ac08-819f0322d065/moderation/remove", bytes.NewBufferString(`{
		"reason": "policy violation"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !actions.removeCommentCalled {
		t.Fatal("expected RemoveComment to be called")
	}
}

func TestRemovePostRejectsInvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	actions := &fakeReportUseCase{}
	router := newModerationTestRouter(actions, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/moderation/remove", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertModerationErrorCode(t, recorder, apperr.CodeInvalidArgument)
	if actions.removePostCalled {
		t.Fatal("remove usecase should not be called for invalid request")
	}
}

func TestRemovePostPropagatesUseCaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	actions := &fakeReportUseCase{
		removePostErr: apperr.New(apperr.CodeForbidden, "platform staff required"),
	}
	router := newModerationTestRouter(actions, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/posts/8f92e975-5323-4a58-bac1-1336b668183c/moderation/remove", bytes.NewBufferString(`{
		"reason": "policy violation"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, recorder.Code, recorder.Body.String())
	}
	assertModerationErrorCode(t, recorder, apperr.CodeForbidden)
}

func TestListReportsReturnsReports(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	console := &fakeReportUseCase{
		listReportsResult: moderationusecase.ListReportsResult{
			Reports: []moderationusecase.ContentReport{
				newReportResult(moderationdomain.TargetTypePost.String(), "8f92e975-5323-4a58-bac1-1336b668183c", ""),
			},
			Limit:  50,
			Offset: 2,
		},
	}
	router := newModerationTestRouter(console, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/moderation/reports?status=resolved&limit=99&offset=2", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !console.listReportsCalled {
		t.Fatal("expected ListReports to be called")
	}
	if console.listReportsInput.ActorID != userID {
		t.Fatalf("expected actor %q, got %q", userID.String(), console.listReportsInput.ActorID.String())
	}
	if console.listReportsInput.Status != "resolved" || console.listReportsInput.Limit != 99 || console.listReportsInput.Offset != 2 {
		t.Fatalf("unexpected list input: %#v", console.listReportsInput)
	}

	var response listReportsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Limit != 50 || response.Offset != 2 || len(response.Reports) != 1 {
		t.Fatalf("unexpected list response: %#v", response)
	}
}

func TestGetReportReturnsReport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	reportID := "2b74a80d-6d57-4d61-a3ae-db5e006766b6"
	console := &fakeReportUseCase{
		getReportResult: moderationusecase.GetReportResult{
			Report: newReportResult(moderationdomain.TargetTypeComment.String(), "", "94509b7b-1b72-4726-ac08-819f0322d065"),
		},
	}
	router := newModerationTestRouter(console, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/moderation/reports/"+reportID, nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !console.getReportCalled {
		t.Fatal("expected GetReport to be called")
	}
	if console.getReportInput.ActorID != userID || console.getReportInput.ReportID != reportID {
		t.Fatalf("unexpected get input: %#v", console.getReportInput)
	}
}

func TestListReportsRejectsInvalidQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	console := &fakeReportUseCase{}
	router := newModerationTestRouter(console, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/moderation/reports?limit=abc", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertModerationErrorCode(t, recorder, apperr.CodeInvalidArgument)
	if console.listReportsCalled {
		t.Fatal("list reports usecase should not be called for invalid query")
	}
}

func TestListReportsPropagatesUseCaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	console := &fakeReportUseCase{
		listReportsErr: apperr.New(apperr.CodeForbidden, "platform staff required"),
	}
	router := newModerationTestRouter(console, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/moderation/reports", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, recorder.Code, recorder.Body.String())
	}
	assertModerationErrorCode(t, recorder, apperr.CodeForbidden)
}

func TestDismissReportReturnsDismissedReport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	reportID := "2b74a80d-6d57-4d61-a3ae-db5e006766b6"
	dismissed := newReportResult(moderationdomain.TargetTypePost.String(), "8f92e975-5323-4a58-bac1-1336b668183c", "")
	dismissed.Status = moderationdomain.ReportStatusDismissed.String()
	dismissed.ReviewedBy = userID.String()
	reviewedAt := time.Date(2026, 6, 2, 5, 0, 0, 0, time.UTC)
	dismissed.ReviewedAt = &reviewedAt
	console := &fakeReportUseCase{
		dismissReportResult: moderationusecase.DismissReportResult{
			Report: dismissed,
		},
	}
	router := newModerationTestRouter(console, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/moderation/reports/"+reportID+"/dismiss", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !console.dismissReportCalled {
		t.Fatal("expected DismissReport to be called")
	}
	if console.dismissReportInput.ActorID != userID || console.dismissReportInput.ReportID != reportID {
		t.Fatalf("unexpected dismiss input: %#v", console.dismissReportInput)
	}

	var response reportContentResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Report.Status != moderationdomain.ReportStatusDismissed.String() || response.Report.ReviewedBy != userID.String() {
		t.Fatalf("unexpected dismiss response: %#v", response.Report)
	}
}

func TestDismissReportPropagatesUseCaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	console := &fakeReportUseCase{
		dismissReportErr: apperr.New(apperr.CodeConflict, "content report is not pending"),
	}
	router := newModerationTestRouter(console, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/moderation/reports/2b74a80d-6d57-4d61-a3ae-db5e006766b6/dismiss", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, recorder.Code, recorder.Body.String())
	}
	assertModerationErrorCode(t, recorder, apperr.CodeConflict)
}

func TestRemoveReportedTargetReturnsModerationAction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	reportID := "2b74a80d-6d57-4d61-a3ae-db5e006766b6"
	console := &fakeReportUseCase{
		removeReportedTargetResult: moderationusecase.RemoveReportedTargetResult{
			Action: newActionResult(moderationdomain.TargetTypePost.String(), "8f92e975-5323-4a58-bac1-1336b668183c", ""),
		},
	}
	router := newModerationTestRouter(console, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/moderation/reports/"+reportID+"/remove-target", bytes.NewBufferString(`{
		"reason": "policy violation"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !console.removeReportedTargetCalled {
		t.Fatal("expected RemoveReportedTarget to be called")
	}
	if console.removeReportedTargetInput.ActorID != userID || console.removeReportedTargetInput.ReportID != reportID {
		t.Fatalf("unexpected remove reported target input: %#v", console.removeReportedTargetInput)
	}
	if console.removeReportedTargetInput.Reason != "policy violation" {
		t.Fatalf("expected reason, got %q", console.removeReportedTargetInput.Reason)
	}

	var response removeContentResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Action.TargetType != moderationdomain.TargetTypePost.String() || response.Action.Action != moderationdomain.ActionTypeRemove.String() {
		t.Fatalf("unexpected action response: %#v", response.Action)
	}
}

func TestRemoveReportedTargetRejectsInvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	console := &fakeReportUseCase{}
	router := newModerationTestRouter(console, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/moderation/reports/2b74a80d-6d57-4d61-a3ae-db5e006766b6/remove-target", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertModerationErrorCode(t, recorder, apperr.CodeInvalidArgument)
	if console.removeReportedTargetCalled {
		t.Fatal("remove reported target usecase should not be called for invalid request")
	}
}

func TestRemoveReportedTargetPropagatesUseCaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	console := &fakeReportUseCase{
		removeReportedTargetErr: apperr.New(apperr.CodeForbidden, "platform staff required"),
	}
	router := newModerationTestRouter(console, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/moderation/reports/2b74a80d-6d57-4d61-a3ae-db5e006766b6/remove-target", bytes.NewBufferString(`{
		"reason": "policy violation"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, recorder.Code, recorder.Body.String())
	}
	assertModerationErrorCode(t, recorder, apperr.CodeForbidden)
}

type fakeReportUseCase struct {
	reportPostCalled           bool
	reportCommentCalled        bool
	removePostCalled           bool
	removeCommentCalled        bool
	listReportsCalled          bool
	getReportCalled            bool
	dismissReportCalled        bool
	removeReportedTargetCalled bool
	reportPostInput            moderationusecase.ReportPostInput
	reportCommentInput         moderationusecase.ReportCommentInput
	removePostInput            moderationusecase.RemovePostInput
	removeCommentInput         moderationusecase.RemoveCommentInput
	listReportsInput           moderationusecase.ListReportsInput
	getReportInput             moderationusecase.GetReportInput
	dismissReportInput         moderationusecase.DismissReportInput
	removeReportedTargetInput  moderationusecase.RemoveReportedTargetInput
	reportPostResult           moderationusecase.ReportContentResult
	reportCommentResult        moderationusecase.ReportContentResult
	removePostResult           moderationusecase.RemoveContentResult
	removeCommentResult        moderationusecase.RemoveContentResult
	listReportsResult          moderationusecase.ListReportsResult
	getReportResult            moderationusecase.GetReportResult
	dismissReportResult        moderationusecase.DismissReportResult
	removeReportedTargetResult moderationusecase.RemoveReportedTargetResult
	reportPostErr              error
	reportCommentErr           error
	removePostErr              error
	removeCommentErr           error
	listReportsErr             error
	getReportErr               error
	dismissReportErr           error
	removeReportedTargetErr    error
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

func (f *fakeReportUseCase) RemovePost(ctx context.Context, input moderationusecase.RemovePostInput) (moderationusecase.RemoveContentResult, error) {
	f.removePostCalled = true
	f.removePostInput = input
	return f.removePostResult, f.removePostErr
}

func (f *fakeReportUseCase) RemoveComment(ctx context.Context, input moderationusecase.RemoveCommentInput) (moderationusecase.RemoveContentResult, error) {
	f.removeCommentCalled = true
	f.removeCommentInput = input
	return f.removeCommentResult, f.removeCommentErr
}

func (f *fakeReportUseCase) ListReports(ctx context.Context, input moderationusecase.ListReportsInput) (moderationusecase.ListReportsResult, error) {
	f.listReportsCalled = true
	f.listReportsInput = input
	return f.listReportsResult, f.listReportsErr
}

func (f *fakeReportUseCase) GetReport(ctx context.Context, input moderationusecase.GetReportInput) (moderationusecase.GetReportResult, error) {
	f.getReportCalled = true
	f.getReportInput = input
	return f.getReportResult, f.getReportErr
}

func (f *fakeReportUseCase) DismissReport(ctx context.Context, input moderationusecase.DismissReportInput) (moderationusecase.DismissReportResult, error) {
	f.dismissReportCalled = true
	f.dismissReportInput = input
	return f.dismissReportResult, f.dismissReportErr
}

func (f *fakeReportUseCase) RemoveReportedTarget(ctx context.Context, input moderationusecase.RemoveReportedTargetInput) (moderationusecase.RemoveReportedTargetResult, error) {
	f.removeReportedTargetCalled = true
	f.removeReportedTargetInput = input
	return f.removeReportedTargetResult, f.removeReportedTargetErr
}

type fakeAccessTokenParser struct {
	claims *authtoken.AccessTokenClaims
	err    error
}

func (f *fakeAccessTokenParser) ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error) {
	return f.claims, f.err
}

func newModerationTestRouter(usecase *fakeReportUseCase, parser authhttp.AccessTokenParser) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())

	protected := router.Group("/api/v1")
	protected.Use(authhttp.RequireAuth(parser))
	RegisterRoutes(protected, NewHandler(usecase, usecase, usecase))

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

func newActionResult(targetType string, postID string, commentID string) moderationusecase.ModerationAction {
	now := time.Date(2026, 6, 2, 4, 40, 0, 0, time.UTC)
	return moderationusecase.ModerationAction{
		ID:         "7dc17b1f-0743-4979-974d-78005cd0ef09",
		TargetType: targetType,
		PostID:     postID,
		CommentID:  commentID,
		ActorID:    userdomain.NewGeneratedUserID().String(),
		Action:     moderationdomain.ActionTypeRemove.String(),
		Reason:     "policy violation",
		CreatedAt:  now,
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

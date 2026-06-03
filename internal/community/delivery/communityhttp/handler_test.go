package communityhttp

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
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParser())

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
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParser())

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
	applications := &fakeCommunityApplicationUseCase{}
	router := newCommunityTestRouter(communities, applications, &fakeAccessTokenParser{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertCommunityErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if communities.listCalled || communities.getCalled || applications.submitCalled || applications.listCalled || applications.getCalled || applications.approveCalled || applications.rejectCalled {
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
			router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParser())

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

func TestSubmitCommunityApplicationReturnsCreatedApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	applications := &fakeCommunityApplicationUseCase{
		submitResult: communityusecase.SubmitCommunityApplicationResult{
			Application: newApplicationResult("campus", "pending", now),
		},
	}
	router := newCommunityTestRouter(&fakeCommunityReadUseCase{}, applications, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/community-applications", bytes.NewBufferString(`{
		"requested_slug": "campus",
		"requested_name": "Campus",
		"reason": "Need a campus board"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if !applications.submitCalled {
		t.Fatal("expected SubmitCommunityApplication to be called")
	}
	if applications.submitInput.ApplicantID != userID {
		t.Fatalf("expected applicant %q, got %q", userID.String(), applications.submitInput.ApplicantID.String())
	}
	if applications.submitInput.RequestedSlug != "campus" {
		t.Fatalf("expected requested slug campus, got %q", applications.submitInput.RequestedSlug)
	}

	var response submitCommunityApplicationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Application.Status != "pending" {
		t.Fatalf("expected pending application, got %q", response.Application.Status)
	}
}

func TestSubmitCommunityApplicationUseCaseErrorMapsToHTTPError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	applications := &fakeCommunityApplicationUseCase{
		submitErr: apperr.New(apperr.CodeConflict, "pending community application slug already exists"),
	}
	router := newCommunityTestRouter(&fakeCommunityReadUseCase{}, applications, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/community-applications", bytes.NewBufferString(`{
		"requested_slug": "campus",
		"requested_name": "Campus",
		"reason": "Need a campus board"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, recorder.Code, recorder.Body.String())
	}
	assertCommunityErrorCode(t, recorder, apperr.CodeConflict)
}

func TestListCommunityApplicationsReturnsApplications(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	applications := &fakeCommunityApplicationUseCase{
		listResult: communityusecase.ListCommunityApplicationsResult{
			Applications: []communityusecase.CommunityApplication{
				newApplicationResult("campus", "approved", now),
			},
			Limit:  10,
			Offset: 5,
		},
	}
	router := newCommunityTestRouter(&fakeCommunityReadUseCase{}, applications, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/community-applications?status=approved&limit=10&offset=5", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !applications.listCalled {
		t.Fatal("expected ListCommunityApplications to be called")
	}
	if applications.listInput.ReviewerID != userID {
		t.Fatalf("expected reviewer %q, got %q", userID.String(), applications.listInput.ReviewerID.String())
	}
	if applications.listInput.Status != "approved" || applications.listInput.Limit != 10 || applications.listInput.Offset != 5 {
		t.Fatalf("unexpected list input: %#v", applications.listInput)
	}

	var response listCommunityApplicationsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Limit != 10 || response.Offset != 5 {
		t.Fatalf("expected pagination limit=10 offset=5, got limit=%d offset=%d", response.Limit, response.Offset)
	}
	if len(response.Applications) != 1 || response.Applications[0].Status != "approved" {
		t.Fatalf("unexpected applications response: %#v", response.Applications)
	}
	if response.Applications[0].ReviewedBy != nil || response.Applications[0].ReviewedAt != nil || response.Applications[0].RejectReason != "" {
		t.Fatalf("expected empty review fields, got %#v", response.Applications[0])
	}
}

func TestListCommunityApplicationsRejectsInvalidQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	applications := &fakeCommunityApplicationUseCase{}
	router := newCommunityTestRouter(&fakeCommunityReadUseCase{}, applications, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/community-applications?limit=bad", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertCommunityErrorCode(t, recorder, apperr.CodeInvalidArgument)
	if applications.listCalled {
		t.Fatal("ListCommunityApplications should not be called for invalid query")
	}
}

func TestGetCommunityApplicationReturnsApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 3, 11, 0, 0, 0, time.UTC)
	applications := &fakeCommunityApplicationUseCase{
		getResult: communityusecase.GetCommunityApplicationResult{
			Application: newApplicationResult("campus", "pending", now),
		},
	}
	router := newCommunityTestRouter(&fakeCommunityReadUseCase{}, applications, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/community-applications/8f92e975-5323-4a58-bac1-1336b668183c", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !applications.getCalled {
		t.Fatal("expected GetCommunityApplication to be called")
	}
	if applications.getInput.ReviewerID != userID {
		t.Fatalf("expected reviewer %q, got %q", userID.String(), applications.getInput.ReviewerID.String())
	}
	if applications.getInput.ApplicationID != "8f92e975-5323-4a58-bac1-1336b668183c" {
		t.Fatalf("unexpected application id %q", applications.getInput.ApplicationID)
	}

	var response getCommunityApplicationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Application.ID != "8f92e975-5323-4a58-bac1-1336b668183c" {
		t.Fatalf("unexpected application response: %#v", response.Application)
	}
}

func TestApproveCommunityApplicationReturnsCommunity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	applications := &fakeCommunityApplicationUseCase{
		approveResult: communityusecase.ApproveCommunityApplicationResult{
			Application: newApplicationResult("campus", "approved", now),
			Community:   newCommunityResult("campus", now),
		},
	}
	router := newCommunityTestRouter(&fakeCommunityReadUseCase{}, applications, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/community-applications/8f92e975-5323-4a58-bac1-1336b668183c/approve", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !applications.approveCalled {
		t.Fatal("expected ApproveCommunityApplication to be called")
	}
	if applications.approveInput.ReviewerID != userID {
		t.Fatalf("expected reviewer %q, got %q", userID.String(), applications.approveInput.ReviewerID.String())
	}
	if applications.approveInput.ApplicationID != "8f92e975-5323-4a58-bac1-1336b668183c" {
		t.Fatalf("unexpected application id %q", applications.approveInput.ApplicationID)
	}

	var response approveCommunityApplicationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Application.Status != "approved" || response.Community.Slug != "campus" {
		t.Fatalf("unexpected approve response: %#v", response)
	}
}

func TestRejectCommunityApplicationReturnsRejectedApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 1, 12, 30, 0, 0, time.UTC)
	applications := &fakeCommunityApplicationUseCase{
		rejectResult: communityusecase.RejectCommunityApplicationResult{
			Application: newApplicationResult("campus", "rejected", now),
		},
	}
	router := newCommunityTestRouter(&fakeCommunityReadUseCase{}, applications, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/community-applications/8f92e975-5323-4a58-bac1-1336b668183c/reject", bytes.NewBufferString(`{
		"reject_reason": "duplicate slug"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !applications.rejectCalled {
		t.Fatal("expected RejectCommunityApplication to be called")
	}
	if applications.rejectInput.ReviewerID != userID {
		t.Fatalf("expected reviewer %q, got %q", userID.String(), applications.rejectInput.ReviewerID.String())
	}
	if applications.rejectInput.RejectReason != "duplicate slug" {
		t.Fatalf("expected reject reason, got %q", applications.rejectInput.RejectReason)
	}

	var response rejectCommunityApplicationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Application.Status != "rejected" {
		t.Fatalf("expected rejected application, got %q", response.Application.Status)
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

type fakeCommunityApplicationUseCase struct {
	submitCalled  bool
	listCalled    bool
	getCalled     bool
	approveCalled bool
	rejectCalled  bool
	submitInput   communityusecase.SubmitCommunityApplicationInput
	listInput     communityusecase.ListCommunityApplicationsInput
	getInput      communityusecase.GetCommunityApplicationInput
	approveInput  communityusecase.ReviewCommunityApplicationInput
	rejectInput   communityusecase.ReviewCommunityApplicationInput
	submitResult  communityusecase.SubmitCommunityApplicationResult
	listResult    communityusecase.ListCommunityApplicationsResult
	getResult     communityusecase.GetCommunityApplicationResult
	approveResult communityusecase.ApproveCommunityApplicationResult
	rejectResult  communityusecase.RejectCommunityApplicationResult
	submitErr     error
	listErr       error
	getErr        error
	approveErr    error
	rejectErr     error
}

func (f *fakeCommunityApplicationUseCase) SubmitCommunityApplication(ctx context.Context, input communityusecase.SubmitCommunityApplicationInput) (communityusecase.SubmitCommunityApplicationResult, error) {
	f.submitCalled = true
	f.submitInput = input
	return f.submitResult, f.submitErr
}

func (f *fakeCommunityApplicationUseCase) ListCommunityApplications(ctx context.Context, input communityusecase.ListCommunityApplicationsInput) (communityusecase.ListCommunityApplicationsResult, error) {
	f.listCalled = true
	f.listInput = input
	return f.listResult, f.listErr
}

func (f *fakeCommunityApplicationUseCase) GetCommunityApplication(ctx context.Context, input communityusecase.GetCommunityApplicationInput) (communityusecase.GetCommunityApplicationResult, error) {
	f.getCalled = true
	f.getInput = input
	return f.getResult, f.getErr
}

func (f *fakeCommunityApplicationUseCase) ApproveCommunityApplication(ctx context.Context, input communityusecase.ReviewCommunityApplicationInput) (communityusecase.ApproveCommunityApplicationResult, error) {
	f.approveCalled = true
	f.approveInput = input
	return f.approveResult, f.approveErr
}

func (f *fakeCommunityApplicationUseCase) RejectCommunityApplication(ctx context.Context, input communityusecase.ReviewCommunityApplicationInput) (communityusecase.RejectCommunityApplicationResult, error) {
	f.rejectCalled = true
	f.rejectInput = input
	return f.rejectResult, f.rejectErr
}

type fakeAccessTokenParser struct {
	claims *authtoken.AccessTokenClaims
	err    error
}

func (f *fakeAccessTokenParser) ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error) {
	return f.claims, f.err
}

func newCommunityTestRouter(communities CommunityReadUseCase, applications CommunityApplicationUseCase, parser authhttp.AccessTokenParser) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())

	protected := router.Group("/api/v1")
	protected.Use(authhttp.RequireAuth(parser))
	RegisterRoutes(protected, NewHandler(communities, applications))

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

func newApplicationResult(slug string, status string, now time.Time) communityusecase.CommunityApplication {
	return communityusecase.CommunityApplication{
		ID:            "8f92e975-5323-4a58-bac1-1336b668183c",
		ApplicantID:   userdomain.NewGeneratedUserID().String(),
		RequestedSlug: slug,
		RequestedName: slug,
		Reason:        "Need a community",
		Status:        status,
		CreatedAt:     now,
		UpdatedAt:     now,
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

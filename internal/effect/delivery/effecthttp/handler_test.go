package effecthttp

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
	"github.com/Versifine/cumt-nexus-api/internal/effect/effectusecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

func TestListEffectsCatalogAllowsAnonymousViewer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	usecase := &fakeUseCase{
		listResult: effectusecase.ListEffectsCatalogResult{
			Effects: []effectusecase.Effect{{
				ID:           "sparkle",
				Name:         "Sparkle",
				Description:  "Sparkle burst",
				CostPoints:   10,
				AnimationKey: "sparkle",
				IsActive:     true,
				CreatedAt:    now,
				UpdatedAt:    now,
			}},
		},
	}
	router := newEffectTestRouter(usecase, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/effects/catalog", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !usecase.listCalled {
		t.Fatal("expected ListEffectsCatalog to be called")
	}
	var response listEffectsCatalogResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Effects) != 1 || response.Effects[0].ID != "sparkle" {
		t.Fatalf("unexpected catalog response: %#v", response.Effects)
	}
}

func TestListEffectsCatalogRejectsInvalidBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &fakeUseCase{}
	router := newEffectTestRouter(usecase, &fakeAccessTokenParser{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/effects/catalog", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertEffectErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if usecase.listCalled {
		t.Fatal("usecase should not be called")
	}
}

func TestGetMyPointsReturnsPoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	usecase := &fakeUseCase{
		pointsResult: effectusecase.GetMyPointsResult{
			Points: effectusecase.PointAccount{
				UserID:         userID.String(),
				Balance:        100,
				LifetimeEarned: 100,
				UpdatedAt:      now,
			},
		},
	}
	router := newEffectTestRouter(usecase, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/points", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !usecase.pointsCalled || usecase.pointsInput.ActorID != userID {
		t.Fatalf("unexpected points input: %#v", usecase.pointsInput)
	}
	var response getMyPointsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Points.Balance != 100 || response.Points.UserID != userID.String() {
		t.Fatalf("unexpected points response: %#v", response.Points)
	}
}

func TestGetMyPointsRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &fakeUseCase{}
	router := newEffectTestRouter(usecase, &fakeAccessTokenParser{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/points", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertEffectErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if usecase.pointsCalled {
		t.Fatal("usecase should not be called")
	}
}

func TestListMyPointTransactionsReturnsTransactions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	usecase := &fakeUseCase{
		transactionsResult: effectusecase.ListMyPointTransactionsResult{
			Transactions: []effectusecase.PointTransaction{{
				ID:           "1f37d0f3-f559-4ee0-ae11-f60f258a1c9a",
				UserID:       userID.String(),
				Delta:        -10,
				BalanceAfter: 90,
				Reason:       "comment_effect",
				SourceType:   "comment_effect",
				SourceID:     "0df8a5e8-a7d5-47c5-9c24-05c9c1f4df36",
				CreatedAt:    now,
			}},
			Limit:      20,
			Offset:     5,
			NextOffset: 6,
			HasMore:    true,
		},
	}
	router := newEffectTestRouter(usecase, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/point-transactions?limit=20&offset=5", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !usecase.transactionsCalled || usecase.transactionsInput.ActorID != userID || usecase.transactionsInput.Limit != 20 || usecase.transactionsInput.Offset != 5 {
		t.Fatalf("unexpected transactions input: %#v", usecase.transactionsInput)
	}
	var response listMyPointTransactionsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Transactions) != 1 || response.Transactions[0].Delta != -10 || response.NextOffset != 6 || !response.HasMore {
		t.Fatalf("unexpected transactions response: %#v", response)
	}
}

func TestApplyCommentEffectReturnsResult(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	commentID := "6b03b945-c34f-4587-b06e-72bd91bf95c8"
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	usecase := &fakeUseCase{
		applyResult: effectusecase.ApplyCommentEffectResult{
			CommentEffect: effectusecase.CommentEffect{
				ID:          "0df8a5e8-a7d5-47c5-9c24-05c9c1f4df36",
				CommentID:   commentID,
				EffectID:    "sparkle",
				UserID:      userID.String(),
				PointsSpent: 10,
				CreatedAt:   now,
			},
			Points: effectusecase.PointAccount{
				UserID:         userID.String(),
				Balance:        90,
				LifetimeEarned: 100,
				LifetimeSpent:  10,
				UpdatedAt:      now,
			},
		},
	}
	router := newEffectTestRouter(usecase, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/comments/"+commentID+"/effects", bytes.NewBufferString(`{"effect_id":"sparkle"}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if !usecase.applyCalled || usecase.applyInput.ActorID != userID || usecase.applyInput.CommentID != commentID || usecase.applyInput.EffectID != "sparkle" {
		t.Fatalf("unexpected apply input: %#v", usecase.applyInput)
	}
	var response applyCommentEffectResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.CommentEffect.EffectID != "sparkle" || response.Points.Balance != 90 {
		t.Fatalf("unexpected apply response: %#v", response)
	}
}

func TestApplyCommentEffectRejectsInvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &fakeUseCase{}
	router := newEffectTestRouter(usecase, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/comments/6b03b945-c34f-4587-b06e-72bd91bf95c8/effects", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertEffectErrorCode(t, recorder, apperr.CodeInvalidArgument)
	if usecase.applyCalled {
		t.Fatal("usecase should not be called")
	}
}

type fakeUseCase struct {
	listCalled bool
	listResult effectusecase.ListEffectsCatalogResult
	listErr    error

	pointsCalled bool
	pointsInput  effectusecase.GetMyPointsInput
	pointsResult effectusecase.GetMyPointsResult
	pointsErr    error

	transactionsCalled bool
	transactionsInput  effectusecase.ListMyPointTransactionsInput
	transactionsResult effectusecase.ListMyPointTransactionsResult
	transactionsErr    error

	applyCalled bool
	applyInput  effectusecase.ApplyCommentEffectInput
	applyResult effectusecase.ApplyCommentEffectResult
	applyErr    error
}

func (f *fakeUseCase) ListEffectsCatalog(ctx context.Context) (effectusecase.ListEffectsCatalogResult, error) {
	_ = ctx
	f.listCalled = true
	return f.listResult, f.listErr
}

func (f *fakeUseCase) GetMyPoints(ctx context.Context, input effectusecase.GetMyPointsInput) (effectusecase.GetMyPointsResult, error) {
	_ = ctx
	f.pointsCalled = true
	f.pointsInput = input
	return f.pointsResult, f.pointsErr
}

func (f *fakeUseCase) ListMyPointTransactions(ctx context.Context, input effectusecase.ListMyPointTransactionsInput) (effectusecase.ListMyPointTransactionsResult, error) {
	_ = ctx
	f.transactionsCalled = true
	f.transactionsInput = input
	return f.transactionsResult, f.transactionsErr
}

func (f *fakeUseCase) ApplyCommentEffect(ctx context.Context, input effectusecase.ApplyCommentEffectInput) (effectusecase.ApplyCommentEffectResult, error) {
	_ = ctx
	f.applyCalled = true
	f.applyInput = input
	return f.applyResult, f.applyErr
}

type fakeAccessTokenParser struct {
	claims *authtoken.AccessTokenClaims
	err    error
}

func (f *fakeAccessTokenParser) ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error) {
	return f.claims, f.err
}

func newEffectTestRouter(usecase UseCase, parser authhttp.AccessTokenParser) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())

	publicRead := router.Group("/api/v1")
	publicRead.Use(authhttp.OptionalAuth(parser))
	RegisterPublicRoutes(publicRead, NewHandler(usecase))

	protected := router.Group("/api/v1")
	protected.Use(authhttp.RequireAuth(parser))
	RegisterRoutes(protected, NewHandler(usecase))

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

func assertEffectErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode apperr.Code) {
	t.Helper()

	var response httpserver.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != string(wantCode) {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
}

package contentrefhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authtoken"
	"github.com/Versifine/cumt-nexus-api/internal/auth/delivery/authhttp"
	"github.com/Versifine/cumt-nexus-api/internal/contentref/contentrefusecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

func TestResolveLinkPreviewReturnsPreview(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &fakeUseCase{
		linkPreview: contentrefusecase.LinkPreview{
			Provider:     contentrefusecase.ProviderGenericLink,
			URL:          "https://example.com/a",
			CanonicalURL: "https://example.com/a",
			Host:         "example.com",
			Title:        "",
			Description:  "",
			ImageURL:     "",
		},
	}
	router := newContentRefTestRouter(usecase, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/link-previews/resolve", bytes.NewBufferString(`{"url":"https://example.com/a"}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !usecase.resolveLinkPreviewCalled || usecase.linkPreviewInput.URL != "https://example.com/a" {
		t.Fatalf("unexpected link preview input: %#v", usecase.linkPreviewInput)
	}

	var response resolveLinkPreviewResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Preview.CanonicalURL != "https://example.com/a" || response.Preview.Provider != contentrefusecase.ProviderGenericLink {
		t.Fatalf("unexpected preview response: %#v", response.Preview)
	}
}

func TestResolveEmbedReturnsEmbed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &fakeUseCase{
		embed: contentrefusecase.Embed{
			ID:            "a9f42d54-9484-40da-b478-83f41c7e173b",
			Provider:      contentrefusecase.ProviderBilibili,
			ProviderRef:   "BV1xx411c7mD",
			URL:           "https://www.bilibili.com/video/BV1xx411c7mD",
			CanonicalURL:  "https://www.bilibili.com/video/BV1xx411c7mD",
			EmbedURL:      "https://player.bilibili.com/player.html?bvid=BV1xx411c7mD",
			IframeAllowed: true,
			Title:         "Campus video",
			Description:   "Campus description",
			ImageURL:      "https://example.com/cover.jpg",
			AuthorName:    "Alice",
			Status:        contentrefusecase.EmbedStatusReady,
		},
	}
	router := newContentRefTestRouter(usecase, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/embeds/resolve", bytes.NewBufferString(`{"url":"https://www.bilibili.com/video/BV1xx411c7mD"}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !usecase.resolveEmbedCalled || usecase.embedInput.URL != "https://www.bilibili.com/video/BV1xx411c7mD" {
		t.Fatalf("unexpected embed input: %#v", usecase.embedInput)
	}

	var response resolveEmbedResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Embed.ID != "a9f42d54-9484-40da-b478-83f41c7e173b" ||
		response.Embed.Provider != contentrefusecase.ProviderBilibili ||
		response.Embed.ProviderRef != "BV1xx411c7mD" ||
		!response.Embed.IframeAllowed ||
		response.Embed.Title != "Campus video" ||
		response.Embed.Status != contentrefusecase.EmbedStatusReady {
		t.Fatalf("unexpected embed response: %#v", response.Embed)
	}
}

func TestResolveLinkPreviewRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &fakeUseCase{}
	router := newContentRefTestRouter(usecase, &fakeAccessTokenParser{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/link-previews/resolve", bytes.NewBufferString(`{"url":"https://example.com"}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertContentRefErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if usecase.resolveLinkPreviewCalled || usecase.resolveEmbedCalled {
		t.Fatal("usecase should not be called")
	}
}

func TestResolveEmbedRejectsInvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &fakeUseCase{}
	router := newContentRefTestRouter(usecase, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/embeds/resolve", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertContentRefErrorCode(t, recorder, apperr.CodeInvalidArgument)
	if usecase.resolveEmbedCalled {
		t.Fatal("usecase should not be called")
	}
}

func TestResolveEmbedPropagatesUseCaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &fakeUseCase{
		embedErr: apperr.New(apperr.CodeInvalidArgument, "embed provider is unsupported"),
	}
	router := newContentRefTestRouter(usecase, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/embeds/resolve", bytes.NewBufferString(`{"url":"https://example.com"}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertContentRefErrorCode(t, recorder, apperr.CodeInvalidArgument)
}

type fakeUseCase struct {
	resolveLinkPreviewCalled bool
	linkPreviewInput         contentrefusecase.ResolveLinkPreviewInput
	linkPreview              contentrefusecase.LinkPreview
	linkPreviewErr           error

	resolveEmbedCalled bool
	embedInput         contentrefusecase.ResolveEmbedInput
	embed              contentrefusecase.Embed
	embedErr           error
}

func (f *fakeUseCase) ResolveLinkPreview(ctx context.Context, input contentrefusecase.ResolveLinkPreviewInput) (contentrefusecase.LinkPreview, error) {
	_ = ctx
	f.resolveLinkPreviewCalled = true
	f.linkPreviewInput = input
	return f.linkPreview, f.linkPreviewErr
}

func (f *fakeUseCase) ResolveEmbed(ctx context.Context, input contentrefusecase.ResolveEmbedInput) (contentrefusecase.Embed, error) {
	_ = ctx
	f.resolveEmbedCalled = true
	f.embedInput = input
	return f.embed, f.embedErr
}

type fakeAccessTokenParser struct {
	claims *authtoken.AccessTokenClaims
	err    error
}

func (f *fakeAccessTokenParser) ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error) {
	return f.claims, f.err
}

func newContentRefTestRouter(usecase UseCase, parser authhttp.AccessTokenParser) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())

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

func assertContentRefErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode apperr.Code) {
	t.Helper()

	var response httpserver.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != string(wantCode) {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
}

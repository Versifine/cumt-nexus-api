package mediahttp

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authtoken"
	"github.com/Versifine/cumt-nexus-api/internal/auth/delivery/authhttp"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediausecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

func TestUploadImageReturnsAttachment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	media := &fakeMediaUseCase{
		result: mediausecase.UploadImageResult{
			Attachment: mediausecase.Attachment{
				ID:           "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a",
				Kind:         "image",
				PublicURL:    "http://localhost:8080/uploads/images/test.png",
				ThumbnailURL: "http://localhost:8080/uploads/images/test.png",
				SizeBytes:    9,
				MimeType:     "image/png",
				AltText:      "Campus",
				Status:       "ready",
				CreatedAt:    now,
			},
		},
	}
	router := newMediaTestRouter(media, validParserWithUserID(userID))

	body, contentType := multipartBody(t, "file", "image.png", pngBytes(), "Campus")
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/images", body)
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", contentType)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if !media.uploadCalled {
		t.Fatal("expected UploadImage to be called")
	}
	if media.input.UploaderID != userID || !bytes.Equal(media.input.FileBytes, pngBytes()) || media.input.AltText != "Campus" {
		t.Fatalf("unexpected input: %#v", media.input)
	}

	var response uploadImageResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Attachment.URL == "" || response.Attachment.Status != "ready" {
		t.Fatalf("unexpected attachment response: %#v", response.Attachment)
	}
	if response.Attachment.ThumbnailURL != response.Attachment.URL {
		t.Fatalf("expected thumbnail_url fallback to url, got %#v", response.Attachment)
	}
}

func TestUploadImageRejectsMissingFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	media := &fakeMediaUseCase{}
	router := newMediaTestRouter(media, validParser())

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/images", body)
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", writer.FormDataContentType())

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertMediaErrorCode(t, recorder, apperr.CodeInvalidArgument)
	if media.uploadCalled {
		t.Fatal("usecase should not be called")
	}
}

func TestUploadImageRejectsInvalidAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	media := &fakeMediaUseCase{}
	router := newMediaTestRouter(media, &fakeAccessTokenParser{})

	body, contentType := multipartBody(t, "file", "image.png", pngBytes(), "")
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/images", body)
	request.Header.Set("Content-Type", contentType)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertMediaErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if media.uploadCalled {
		t.Fatal("usecase should not be called")
	}
}

type fakeMediaUseCase struct {
	uploadCalled bool
	input        mediausecase.UploadImageInput
	result       mediausecase.UploadImageResult
	err          error
}

func (f *fakeMediaUseCase) UploadImage(ctx context.Context, input mediausecase.UploadImageInput) (mediausecase.UploadImageResult, error) {
	f.uploadCalled = true
	f.input = input
	return f.result, f.err
}

type fakeAccessTokenParser struct {
	claims *authtoken.AccessTokenClaims
	err    error
}

func (f *fakeAccessTokenParser) ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error) {
	return f.claims, f.err
}

func newMediaTestRouter(media MediaUseCase, parser authhttp.AccessTokenParser) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())

	protected := router.Group("/api/v1")
	protected.Use(authhttp.RequireAuth(parser))
	RegisterRoutes(protected, NewHandler(media))

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

func multipartBody(t *testing.T, fieldName string, fileName string, fileBytes []byte, altText string) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(fileBytes); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if altText != "" {
		if err := writer.WriteField("alt_text", altText); err != nil {
			t.Fatalf("write alt_text: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func pngBytes() []byte {
	return []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00}
}

func assertMediaErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode apperr.Code) {
	t.Helper()

	var response httpserver.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != string(wantCode) {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
}

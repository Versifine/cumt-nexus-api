package notificationhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authtoken"
	"github.com/Versifine/cumt-nexus-api/internal/auth/delivery/authhttp"
	"github.com/Versifine/cumt-nexus-api/internal/notification/notificationusecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestListNotificationsReturnsNotifications(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := testNow()
	usecase := &fakeUseCase{
		listResult: notificationusecase.ListNotificationsResult{
			Notifications: []notificationusecase.Notification{newNotification(userID, now)},
			Category:      notificationusecase.CategoryFilterLikes.String(),
			Status:        notificationusecase.StatusFilterUnread.String(),
			Limit:         50,
			Offset:        2,
		},
	}
	router := newNotificationTestRouter(usecase, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?category=likes&status=unread&limit=50&offset=2", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !usecase.listCalled {
		t.Fatal("expected list usecase call")
	}
	if usecase.listInput.ActorID != userID || usecase.listInput.Category != "likes" || usecase.listInput.Status != "unread" || usecase.listInput.Limit != 50 || usecase.listInput.Offset != 2 {
		t.Fatalf("unexpected list input: %#v", usecase.listInput)
	}

	var response listNotificationsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Category != "likes" || response.Status != "unread" || response.Limit != 50 || response.Offset != 2 || len(response.Notifications) != 1 {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response.Notifications[0].AggregateCount != 2 || response.Notifications[0].LastActorID == "" {
		t.Fatalf("expected aggregate notification fields, got %#v", response.Notifications[0])
	}
	if response.Notifications[0].Actor == nil || response.Notifications[0].LastActor == nil {
		t.Fatalf("expected actor summaries, got %#v", response.Notifications[0])
	}
	if response.Notifications[0].Context.PostID == "" || response.Notifications[0].Context.Permalink == "" || response.Notifications[0].Context.Community == nil {
		t.Fatalf("expected notification context, got %#v", response.Notifications[0].Context)
	}
}

func TestGetUnreadSummaryReturnsCategoryCounts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	usecase := &fakeUseCase{
		summaryResult: notificationusecase.UnreadSummary{
			Total:    5,
			Replies:  1,
			Mentions: 1,
			Likes:    2,
			System:   1,
		},
	}
	router := newNotificationTestRouter(usecase, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-summary", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !usecase.summaryCalled {
		t.Fatal("expected unread summary usecase call")
	}
	if usecase.summaryInput.ActorID != userID {
		t.Fatalf("unexpected summary input: %#v", usecase.summaryInput)
	}

	var response unreadSummaryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Total != 5 || response.Likes != 2 {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestMarkNotificationReadReturnsNotification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	notificationID := uuid.NewString()
	readAt := testNow()
	notification := newNotification(userID, readAt)
	notification.ID = notificationID
	notification.ReadAt = &readAt
	usecase := &fakeUseCase{
		markResult: notificationusecase.MarkNotificationReadResult{
			Notification: notification,
		},
	}
	router := newNotificationTestRouter(usecase, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+notificationID+"/read", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !usecase.markCalled {
		t.Fatal("expected mark read usecase call")
	}
	if usecase.markInput.ActorID != userID || usecase.markInput.NotificationID != notificationID {
		t.Fatalf("unexpected mark input: %#v", usecase.markInput)
	}

	var response markNotificationReadResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Notification.ReadAt == nil {
		t.Fatalf("expected read_at, got %#v", response.Notification)
	}
}

func TestMarkAllNotificationsReadReturnsUpdatedCount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	readAt := testNow()
	usecase := &fakeUseCase{
		markAllResult: notificationusecase.MarkAllNotificationsReadResult{
			UpdatedCount: 3,
			ReadAt:       readAt,
		},
	}
	router := newNotificationTestRouter(usecase, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/read-all", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !usecase.markAllCalled {
		t.Fatal("expected mark all usecase call")
	}
	if usecase.markAllInput.ActorID != userID {
		t.Fatalf("unexpected mark all input: %#v", usecase.markAllInput)
	}

	var response markAllNotificationsReadResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.UpdatedCount != 3 || !response.ReadAt.Equal(readAt) {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestNotificationRoutesRejectInvalidAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &fakeUseCase{}
	router := newNotificationTestRouter(usecase, &fakeAccessTokenParser{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertNotificationErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if usecase.listCalled {
		t.Fatal("usecase should not be called for invalid auth")
	}
}

func TestListNotificationsRejectsInvalidQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &fakeUseCase{}
	router := newNotificationTestRouter(usecase, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?limit=abc", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertNotificationErrorCode(t, recorder, apperr.CodeInvalidArgument)
	if usecase.listCalled {
		t.Fatal("usecase should not be called for invalid query")
	}
}

func TestNotificationHandlerPropagatesUseCaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usecase := &fakeUseCase{
		listErr: apperr.New(apperr.CodeInvalidArgument, "notification status is invalid"),
	}
	router := newNotificationTestRouter(usecase, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?status=deleted", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertNotificationErrorCode(t, recorder, apperr.CodeInvalidArgument)
}

type fakeUseCase struct {
	listCalled    bool
	summaryCalled bool
	markCalled    bool
	markAllCalled bool
	listInput     notificationusecase.ListNotificationsInput
	summaryInput  notificationusecase.UnreadSummaryInput
	markInput     notificationusecase.MarkNotificationReadInput
	markAllInput  notificationusecase.MarkAllNotificationsReadInput
	listResult    notificationusecase.ListNotificationsResult
	summaryResult notificationusecase.UnreadSummary
	markResult    notificationusecase.MarkNotificationReadResult
	markAllResult notificationusecase.MarkAllNotificationsReadResult
	listErr       error
	summaryErr    error
	markErr       error
	markAllErr    error
}

func (f *fakeUseCase) ListNotifications(ctx context.Context, input notificationusecase.ListNotificationsInput) (notificationusecase.ListNotificationsResult, error) {
	f.listCalled = true
	f.listInput = input
	return f.listResult, f.listErr
}

func (f *fakeUseCase) GetUnreadSummary(ctx context.Context, input notificationusecase.UnreadSummaryInput) (notificationusecase.UnreadSummary, error) {
	f.summaryCalled = true
	f.summaryInput = input
	return f.summaryResult, f.summaryErr
}

func (f *fakeUseCase) MarkNotificationRead(ctx context.Context, input notificationusecase.MarkNotificationReadInput) (notificationusecase.MarkNotificationReadResult, error) {
	f.markCalled = true
	f.markInput = input
	return f.markResult, f.markErr
}

func (f *fakeUseCase) MarkAllNotificationsRead(ctx context.Context, input notificationusecase.MarkAllNotificationsReadInput) (notificationusecase.MarkAllNotificationsReadResult, error) {
	f.markAllCalled = true
	f.markAllInput = input
	return f.markAllResult, f.markAllErr
}

type fakeAccessTokenParser struct {
	claims *authtoken.AccessTokenClaims
	err    error
}

func (f *fakeAccessTokenParser) ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error) {
	return f.claims, f.err
}

func newNotificationTestRouter(usecase *fakeUseCase, parser authhttp.AccessTokenParser) *gin.Engine {
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

func newNotification(recipientID userdomain.UserID, now time.Time) notificationusecase.Notification {
	actorID := userdomain.NewGeneratedUserID().String()
	return notificationusecase.Notification{
		ID:             uuid.NewString(),
		RecipientID:    recipientID.String(),
		Type:           "system",
		Title:          "Title",
		Body:           "Body",
		SourceType:     "system",
		SourceID:       "source-1",
		AggregateCount: 2,
		LastActorID:    actorID,
		Actor: &notificationusecase.NotificationActor{
			ID:          actorID,
			Username:    "alice",
			DisplayName: "Alice",
			AvatarURL:   "https://example.com/avatar.jpg",
		},
		LastActor: &notificationusecase.NotificationActor{
			ID:          actorID,
			Username:    "alice",
			DisplayName: "Alice",
			AvatarURL:   "https://example.com/avatar.jpg",
		},
		Context: notificationusecase.NotificationContext{
			PostID:         uuid.NewString(),
			CommentID:      uuid.NewString(),
			Permalink:      "/c/public/posts/post-1#comment-comment-1",
			PostTitle:      "Post title",
			CommentExcerpt: "Comment excerpt",
			CommentDepth:   1,
			Community: &notificationusecase.NotificationCommunityContext{
				ID:   uuid.NewString(),
				Slug: "public",
				Name: "Public",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func testNow() time.Time {
	return time.Date(2026, 6, 2, 7, 45, 0, 0, time.UTC)
}

func assertNotificationErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode apperr.Code) {
	t.Helper()

	var response httpserver.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != string(wantCode) {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
}

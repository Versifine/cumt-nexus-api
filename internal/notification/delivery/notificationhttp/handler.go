package notificationhttp

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/notification/notificationusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	notifications UseCase
}

type UseCase interface {
	ListNotifications(ctx context.Context, input notificationusecase.ListNotificationsInput) (notificationusecase.ListNotificationsResult, error)
	MarkNotificationRead(ctx context.Context, input notificationusecase.MarkNotificationReadInput) (notificationusecase.MarkNotificationReadResult, error)
}

type notificationResponse struct {
	ID          string     `json:"id"`
	RecipientID string     `json:"recipient_id"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	SourceType  string     `json:"source_type"`
	SourceID    string     `json:"source_id"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type listNotificationsResponse struct {
	Notifications []notificationResponse `json:"notifications"`
	Status        string                 `json:"status"`
	Limit         int                    `json:"limit"`
	Offset        int                    `json:"offset"`
}

type markNotificationReadResponse struct {
	Notification notificationResponse `json:"notification"`
}

func NewHandler(notifications UseCase) *Handler {
	return &Handler{
		notifications: notifications,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/notifications", handler.ListNotifications)
	group.POST("/notifications/:id/read", handler.MarkNotificationRead)
}

func (h *Handler) ListNotifications(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	limit, err := parseOptionalIntQuery(c, "limit")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	offset, err := parseOptionalIntQuery(c, "offset")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	result, err := h.notifications.ListNotifications(c.Request.Context(), notificationusecase.ListNotificationsInput{
		ActorID: userID,
		Status:  c.Query("status"),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listNotificationsResponse{
		Notifications: make([]notificationResponse, 0, len(result.Notifications)),
		Status:        result.Status,
		Limit:         result.Limit,
		Offset:        result.Offset,
	}
	for _, notification := range result.Notifications {
		response.Notifications = append(response.Notifications, toNotificationResponse(notification))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) MarkNotificationRead(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	result, err := h.notifications.MarkNotificationRead(c.Request.Context(), notificationusecase.MarkNotificationReadInput{
		ActorID:        userID,
		NotificationID: c.Param("id"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, markNotificationReadResponse{
		Notification: toNotificationResponse(result.Notification),
	})
}

func parseOptionalIntQuery(c *gin.Context, key string) (int, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, apperr.New(apperr.CodeInvalidArgument, "invalid "+key+" query")
	}
	return value, nil
}

func toNotificationResponse(notification notificationusecase.Notification) notificationResponse {
	return notificationResponse{
		ID:          notification.ID,
		RecipientID: notification.RecipientID,
		Type:        notification.Type,
		Title:       notification.Title,
		Body:        notification.Body,
		SourceType:  notification.SourceType,
		SourceID:    notification.SourceID,
		ReadAt:      notification.ReadAt,
		CreatedAt:   notification.CreatedAt,
		UpdatedAt:   notification.UpdatedAt,
	}
}

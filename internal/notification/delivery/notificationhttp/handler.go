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
	GetUnreadSummary(ctx context.Context, input notificationusecase.UnreadSummaryInput) (notificationusecase.UnreadSummary, error)
	MarkNotificationRead(ctx context.Context, input notificationusecase.MarkNotificationReadInput) (notificationusecase.MarkNotificationReadResult, error)
	MarkAllNotificationsRead(ctx context.Context, input notificationusecase.MarkAllNotificationsReadInput) (notificationusecase.MarkAllNotificationsReadResult, error)
}

type notificationResponse struct {
	ID             string                      `json:"id"`
	RecipientID    string                      `json:"recipient_id"`
	Type           string                      `json:"type"`
	Title          string                      `json:"title"`
	Body           string                      `json:"body"`
	SourceType     string                      `json:"source_type"`
	SourceID       string                      `json:"source_id"`
	AggregateCount int                         `json:"aggregate_count"`
	LastActorID    string                      `json:"last_actor_id"`
	Actor          *actorResponse              `json:"actor,omitempty"`
	LastActor      *actorResponse              `json:"last_actor,omitempty"`
	Context        notificationContextResponse `json:"context"`
	ReadAt         *time.Time                  `json:"read_at,omitempty"`
	CreatedAt      time.Time                   `json:"created_at"`
	UpdatedAt      time.Time                   `json:"updated_at"`
}

type actorResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

type communityContextResponse struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type notificationContextResponse struct {
	PostID         string                    `json:"post_id"`
	CommentID      string                    `json:"comment_id"`
	Permalink      string                    `json:"permalink"`
	PostTitle      string                    `json:"post_title"`
	CommentExcerpt string                    `json:"comment_excerpt"`
	CommentDepth   int                       `json:"comment_depth"`
	Community      *communityContextResponse `json:"community,omitempty"`
}

type listNotificationsResponse struct {
	Notifications []notificationResponse `json:"notifications"`
	Category      string                 `json:"category"`
	Status        string                 `json:"status"`
	Limit         int                    `json:"limit"`
	Offset        int                    `json:"offset"`
	NextOffset    int                    `json:"next_offset"`
	HasMore       bool                   `json:"has_more"`
}

type unreadSummaryResponse struct {
	Total    int `json:"total"`
	Replies  int `json:"replies"`
	Mentions int `json:"mentions"`
	Likes    int `json:"likes"`
	System   int `json:"system"`
}

type markNotificationReadResponse struct {
	Notification notificationResponse `json:"notification"`
}

type markAllNotificationsReadResponse struct {
	UpdatedCount int       `json:"updated_count"`
	ReadAt       time.Time `json:"read_at"`
}

func NewHandler(notifications UseCase) *Handler {
	return &Handler{
		notifications: notifications,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/notifications/unread-summary", handler.GetUnreadSummary)
	group.GET("/notifications", handler.ListNotifications)
	group.POST("/notifications/:id/read", handler.MarkNotificationRead)
	group.POST("/notifications/read-all", handler.MarkAllNotificationsRead)
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
		ActorID:  userID,
		Category: c.Query("category"),
		Status:   c.Query("status"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listNotificationsResponse{
		Notifications: make([]notificationResponse, 0, len(result.Notifications)),
		Category:      result.Category,
		Status:        result.Status,
		Limit:         result.Limit,
		Offset:        result.Offset,
		NextOffset:    result.NextOffset,
		HasMore:       result.HasMore,
	}
	for _, notification := range result.Notifications {
		response.Notifications = append(response.Notifications, toNotificationResponse(notification))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetUnreadSummary(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	summary, err := h.notifications.GetUnreadSummary(c.Request.Context(), notificationusecase.UnreadSummaryInput{
		ActorID: userID,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, unreadSummaryResponse{
		Total:    summary.Total,
		Replies:  summary.Replies,
		Mentions: summary.Mentions,
		Likes:    summary.Likes,
		System:   summary.System,
	})
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

func (h *Handler) MarkAllNotificationsRead(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	result, err := h.notifications.MarkAllNotificationsRead(c.Request.Context(), notificationusecase.MarkAllNotificationsReadInput{
		ActorID: userID,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, markAllNotificationsReadResponse{
		UpdatedCount: result.UpdatedCount,
		ReadAt:       result.ReadAt,
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
		ID:             notification.ID,
		RecipientID:    notification.RecipientID,
		Type:           notification.Type,
		Title:          notification.Title,
		Body:           notification.Body,
		SourceType:     notification.SourceType,
		SourceID:       notification.SourceID,
		AggregateCount: notification.AggregateCount,
		LastActorID:    notification.LastActorID,
		Actor:          toActorResponse(notification.Actor),
		LastActor:      toActorResponse(notification.LastActor),
		Context:        toNotificationContextResponse(notification.Context),
		ReadAt:         notification.ReadAt,
		CreatedAt:      notification.CreatedAt,
		UpdatedAt:      notification.UpdatedAt,
	}
}

func toActorResponse(actor *notificationusecase.NotificationActor) *actorResponse {
	if actor == nil {
		return nil
	}
	return &actorResponse{
		ID:          actor.ID,
		Username:    actor.Username,
		DisplayName: actor.DisplayName,
		AvatarURL:   actor.AvatarURL,
	}
}

func toNotificationContextResponse(context notificationusecase.NotificationContext) notificationContextResponse {
	response := notificationContextResponse{
		PostID:         context.PostID,
		CommentID:      context.CommentID,
		Permalink:      context.Permalink,
		PostTitle:      context.PostTitle,
		CommentExcerpt: context.CommentExcerpt,
		CommentDepth:   context.CommentDepth,
	}
	if context.Community != nil {
		response.Community = &communityContextResponse{
			ID:   context.Community.ID,
			Slug: context.Community.Slug,
			Name: context.Community.Name,
		}
	}
	return response
}

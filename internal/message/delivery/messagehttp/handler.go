package messagehttp

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/message/messageusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

type Handler struct {
	messages MessageUseCase
}

type MessageUseCase interface {
	GetSummary(ctx context.Context, viewerID userdomain.UserID) (messageusecase.SummaryResult, error)
	ListConversations(ctx context.Context, input messageusecase.ListConversationsInput) (messageusecase.ListConversationsResult, error)
	StartConversation(ctx context.Context, input messageusecase.StartConversationInput) (messageusecase.ConversationResult, error)
	ListMessages(ctx context.Context, input messageusecase.ListMessagesInput) (messageusecase.ListMessagesResult, error)
	SendMessage(ctx context.Context, input messageusecase.SendMessageInput) (messageusecase.ConversationResult, error)
	MarkConversationRead(ctx context.Context, input messageusecase.ConversationActionInput) (messageusecase.ConversationResult, error)
	SetConversationArchived(ctx context.Context, input messageusecase.ConversationActionInput, archived bool) (messageusecase.ConversationResult, error)
	SetConversationPinned(ctx context.Context, input messageusecase.ConversationActionInput, pinned bool) (messageusecase.ConversationResult, error)
	SetConversationMuted(ctx context.Context, input messageusecase.ConversationActionInput, muted bool) (messageusecase.ConversationResult, error)
	DeleteConversation(ctx context.Context, input messageusecase.ConversationActionInput) error
	ReportConversation(ctx context.Context, input messageusecase.ConversationActionInput) (messageusecase.ReportResult, error)
	AcceptRequest(ctx context.Context, input messageusecase.RequestActionInput) (messageusecase.ConversationResult, error)
	RejectRequest(ctx context.Context, input messageusecase.RequestActionInput) (messageusecase.ConversationResult, error)
	RecallMessage(ctx context.Context, input messageusecase.MessageActionInput) (messageusecase.ConversationResult, error)
	DeleteMessage(ctx context.Context, input messageusecase.MessageActionInput) error
	ReportMessage(ctx context.Context, input messageusecase.MessageActionInput) (messageusecase.ReportResult, error)
	BlockUser(ctx context.Context, input messageusecase.BlockUserInput) error
	UnblockUser(ctx context.Context, input messageusecase.BlockUserInput) error
	GetPrivacy(ctx context.Context, viewerID userdomain.UserID) (messageusecase.PrivacyResult, error)
	UpdatePrivacy(ctx context.Context, input messageusecase.PrivacyInput) (messageusecase.PrivacyResult, error)
	CreateRealtimeTicket(ctx context.Context, input messageusecase.RealtimeTicketInput) (messageusecase.RealtimeTicketResult, error)
	ConnectRealtime(ctx context.Context, ticket string) (messageusecase.RealtimeConnectResult, error)
}

type summaryResponse struct {
	UnreadTotal         int  `json:"unread_total"`
	RequestCount        int  `json:"request_count"`
	UnreadConversations int  `json:"unread_conversations"`
	OnlineStatusEnabled bool `json:"online_status_enabled"`
}

type listConversationsResponse struct {
	Conversations []conversationResponse `json:"conversations"`
	Box           string                 `json:"box"`
	Limit         int                    `json:"limit"`
	Offset        int                    `json:"offset"`
	NextOffset    int                    `json:"next_offset"`
	HasMore       bool                   `json:"has_more"`
}

type conversationMutationResponse struct {
	Conversation conversationResponse `json:"conversation"`
	Message      *messageResponse     `json:"message,omitempty"`
}

type conversationResponse struct {
	ID                      string                  `json:"id"`
	Box                     string                  `json:"box"`
	RequestID               *string                 `json:"request_id"`
	RequestStatus           string                  `json:"request_status"`
	RequestDirection        string                  `json:"request_direction"`
	ViewerCanAcceptRequest  bool                    `json:"viewer_can_accept_request"`
	ViewerCanRejectRequest  bool                    `json:"viewer_can_reject_request"`
	ViewerCanReopen         bool                    `json:"viewer_can_reopen"`
	RequestCreatedByMe      bool                    `json:"request_created_by_me"`
	RequestToMe             bool                    `json:"request_to_me"`
	ConversationState       string                  `json:"conversation_state"`
	Participant             userSummaryResponse     `json:"participant"`
	LastMessage             *messageSummaryResponse `json:"last_message"`
	UnreadCount             int                     `json:"unread_count"`
	UpdatedAt               time.Time               `json:"updated_at"`
	Pinned                  bool                    `json:"pinned"`
	Muted                   bool                    `json:"muted"`
	Archived                bool                    `json:"archived"`
	Blocked                 bool                    `json:"blocked"`
	CanSend                 bool                    `json:"can_send"`
	DisableReason           *string                 `json:"disable_reason"`
	PeerOnlineStatusVisible bool                    `json:"peer_online_status_visible"`
	PeerOnline              bool                    `json:"peer_online"`
}

type messageSummaryResponse struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Text      string    `json:"text"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type listMessagesResponse struct {
	Messages   []messageResponse `json:"messages"`
	Limit      int               `json:"limit"`
	HasMore    bool              `json:"has_more"`
	NextBefore string            `json:"next_before"`
}

type messageResponse struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversation_id"`
	Sender         userSummaryResponse    `json:"sender"`
	Type           string                 `json:"type"`
	Body           string                 `json:"body"`
	ImageURL       string                 `json:"image_url"`
	Share          *shareSnapshotResponse `json:"share"`
	Status         string                 `json:"status"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	RecalledAt     *time.Time             `json:"recalled_at"`
	ViewerDeleted  bool                   `json:"viewer_deleted"`
}

type shareSnapshotResponse struct {
	ShareType         string    `json:"share_type"`
	ShareID           string    `json:"share_id"`
	Title             string    `json:"title"`
	Summary           string    `json:"summary"`
	ThumbnailURL      string    `json:"thumbnail_url"`
	TargetURL         string    `json:"target_url"`
	SnapshotCreatedAt time.Time `json:"snapshot_created_at"`
}

type userSummaryResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Status      string `json:"status"`
}

type startConversationRequest struct {
	TargetUsername string              `json:"target_username" binding:"required"`
	Message        messageDraftRequest `json:"message" binding:"required"`
}

type messageDraftRequest struct {
	Type     string                `json:"type"`
	Body     string                `json:"body"`
	ImageURL string                `json:"image_url"`
	Share    *shareSnapshotRequest `json:"share"`
}

type shareSnapshotRequest struct {
	ShareType         string    `json:"share_type"`
	ShareID           string    `json:"share_id"`
	Title             string    `json:"title"`
	Summary           string    `json:"summary"`
	ThumbnailURL      string    `json:"thumbnail_url"`
	TargetURL         string    `json:"target_url"`
	SnapshotCreatedAt time.Time `json:"snapshot_created_at"`
}

type sendMessageRequest struct {
	Message messageDraftRequest `json:"message" binding:"required"`
}

type reportMessageRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type privacyResponse struct {
	AllowMessages       string    `json:"allow_messages"`
	OnlineStatusEnabled bool      `json:"online_status_enabled"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type updatePrivacyRequest struct {
	AllowMessages       *string `json:"allow_messages"`
	OnlineStatusEnabled *bool   `json:"online_status_enabled"`
}

type realtimeTicketRequest struct {
	LastEventID string `json:"last_event_id"`
}

type realtimeTicketResponse struct {
	Ticket    string    `json:"ticket"`
	ExpiresAt time.Time `json:"expires_at"`
}

type realtimeEventResponse struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	ConversationID *string   `json:"conversation_id"`
	Payload        string    `json:"payload"`
	CreatedAt      time.Time `json:"created_at"`
}

type realtimeHelloResponse struct {
	Type   string                  `json:"type"`
	Events []realtimeEventResponse `json:"events"`
}

type reportResponse struct {
	Report messageReportResponse `json:"report"`
}

type messageReportResponse struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	MessageID      string    `json:"message_id"`
	ReportedUserID string    `json:"reported_user_id"`
	Reason         string    `json:"reason"`
	ContextBefore  string    `json:"context_before"`
	ContextAfter   string    `json:"context_after"`
	CreatedAt      time.Time `json:"created_at"`
}

func NewHandler(messages MessageUseCase) *Handler {
	return &Handler{messages: messages}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/messages/summary", handler.GetSummary)
	group.GET("/messages/conversations", handler.ListConversations)
	group.POST("/messages/conversations", handler.StartConversation)
	group.GET("/messages/conversations/:id/messages", handler.ListMessages)
	group.POST("/messages/conversations/:id/messages", handler.SendMessage)
	group.POST("/messages/conversations/:id/read", handler.MarkConversationRead)
	group.POST("/messages/conversations/:id/archive", handler.ArchiveConversation)
	group.DELETE("/messages/conversations/:id/archive", handler.UnarchiveConversation)
	group.POST("/messages/conversations/:id/pin", handler.PinConversation)
	group.DELETE("/messages/conversations/:id/pin", handler.UnpinConversation)
	group.POST("/messages/conversations/:id/mute", handler.MuteConversation)
	group.DELETE("/messages/conversations/:id/mute", handler.UnmuteConversation)
	group.DELETE("/messages/conversations/:id", handler.DeleteConversation)
	group.POST("/messages/conversations/:id/report", handler.ReportConversation)
	group.POST("/messages/requests/:id/accept", handler.AcceptRequest)
	group.POST("/messages/requests/:id/reject", handler.RejectRequest)
	group.POST("/messages/:id/recall", handler.RecallMessage)
	group.DELETE("/messages/:id", handler.DeleteMessage)
	group.POST("/messages/:id/report", handler.ReportMessage)
	group.POST("/users/:username/block", handler.BlockUser)
	group.DELETE("/users/:username/block", handler.UnblockUser)
	group.GET("/me/privacy/messages", handler.GetPrivacy)
	group.PATCH("/me/privacy/messages", handler.UpdatePrivacy)
	group.POST("/realtime/tickets", handler.CreateRealtimeTicket)
}

func RegisterRealtimeRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/realtime/messages", handler.RealtimeMessages)
}

func (h *Handler) GetSummary(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	result, err := h.messages.GetSummary(c.Request.Context(), userID)
	if err != nil {
		abortWithError(c, err)
		return
	}
	c.JSON(http.StatusOK, summaryResponse{
		UnreadTotal:         result.UnreadTotal,
		RequestCount:        result.RequestCount,
		UnreadConversations: result.UnreadConversations,
		OnlineStatusEnabled: result.OnlineStatusEnabled,
	})
}

func (h *Handler) ListConversations(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	limit, err := parseOptionalIntQuery(c, "limit")
	if err != nil {
		abortWithError(c, err)
		return
	}
	offset, err := parseOptionalIntQuery(c, "offset")
	if err != nil {
		abortWithError(c, err)
		return
	}
	result, err := h.messages.ListConversations(c.Request.Context(), messageusecase.ListConversationsInput{
		ViewerID: userID,
		Box:      c.Query("box"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		abortWithError(c, err)
		return
	}
	response := listConversationsResponse{
		Conversations: make([]conversationResponse, 0, len(result.Conversations)),
		Box:           result.Box,
		Limit:         result.Limit,
		Offset:        result.Offset,
		NextOffset:    result.NextOffset,
		HasMore:       result.HasMore,
	}
	for _, conversation := range result.Conversations {
		response.Conversations = append(response.Conversations, toConversationResponse(conversation))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) StartConversation(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var request startConversationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		abortWithError(c, apperr.New(apperr.CodeInvalidArgument, "invalid request body"))
		return
	}
	result, err := h.messages.StartConversation(c.Request.Context(), messageusecase.StartConversationInput{
		ViewerID:       userID,
		TargetUsername: request.TargetUsername,
		Message:        toDraft(request.Message),
	})
	if err != nil {
		abortWithError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toConversationMutationResponse(result))
}

func (h *Handler) ListMessages(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	limit, err := parseOptionalIntQuery(c, "limit")
	if err != nil {
		abortWithError(c, err)
		return
	}
	result, err := h.messages.ListMessages(c.Request.Context(), messageusecase.ListMessagesInput{
		ViewerID:        userID,
		ConversationID:  c.Param("id"),
		BeforeMessageID: c.Query("before_message_id"),
		Limit:           limit,
	})
	if err != nil {
		abortWithError(c, err)
		return
	}
	response := listMessagesResponse{Messages: make([]messageResponse, 0, len(result.Messages)), Limit: result.Limit, HasMore: result.HasMore, NextBefore: result.NextBefore}
	for _, message := range result.Messages {
		response.Messages = append(response.Messages, toMessageResponse(message))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) SendMessage(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var request sendMessageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		abortWithError(c, apperr.New(apperr.CodeInvalidArgument, "invalid request body"))
		return
	}
	result, err := h.messages.SendMessage(c.Request.Context(), messageusecase.SendMessageInput{
		ViewerID:       userID,
		ConversationID: c.Param("id"),
		Message:        toDraft(request.Message),
	})
	if err != nil {
		abortWithError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toConversationMutationResponse(result))
}

func (h *Handler) MarkConversationRead(c *gin.Context) {
	h.conversationAction(c, func(ctx context.Context, input messageusecase.ConversationActionInput) (messageusecase.ConversationResult, error) {
		return h.messages.MarkConversationRead(ctx, input)
	})
}

func (h *Handler) ArchiveConversation(c *gin.Context) {
	h.conversationAction(c, func(ctx context.Context, input messageusecase.ConversationActionInput) (messageusecase.ConversationResult, error) {
		return h.messages.SetConversationArchived(ctx, input, true)
	})
}

func (h *Handler) UnarchiveConversation(c *gin.Context) {
	h.conversationAction(c, func(ctx context.Context, input messageusecase.ConversationActionInput) (messageusecase.ConversationResult, error) {
		return h.messages.SetConversationArchived(ctx, input, false)
	})
}

func (h *Handler) PinConversation(c *gin.Context) {
	h.conversationAction(c, func(ctx context.Context, input messageusecase.ConversationActionInput) (messageusecase.ConversationResult, error) {
		return h.messages.SetConversationPinned(ctx, input, true)
	})
}

func (h *Handler) UnpinConversation(c *gin.Context) {
	h.conversationAction(c, func(ctx context.Context, input messageusecase.ConversationActionInput) (messageusecase.ConversationResult, error) {
		return h.messages.SetConversationPinned(ctx, input, false)
	})
}

func (h *Handler) MuteConversation(c *gin.Context) {
	h.conversationAction(c, func(ctx context.Context, input messageusecase.ConversationActionInput) (messageusecase.ConversationResult, error) {
		return h.messages.SetConversationMuted(ctx, input, true)
	})
}

func (h *Handler) UnmuteConversation(c *gin.Context) {
	h.conversationAction(c, func(ctx context.Context, input messageusecase.ConversationActionInput) (messageusecase.ConversationResult, error) {
		return h.messages.SetConversationMuted(ctx, input, false)
	})
}

func (h *Handler) DeleteConversation(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if err := h.messages.DeleteConversation(c.Request.Context(), messageusecase.ConversationActionInput{ViewerID: userID, ConversationID: c.Param("id")}); err != nil {
		abortWithError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ReportConversation(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var request reportMessageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		abortWithError(c, apperr.New(apperr.CodeInvalidArgument, "invalid request body"))
		return
	}
	result, err := h.messages.ReportConversation(c.Request.Context(), messageusecase.ConversationActionInput{ViewerID: userID, ConversationID: c.Param("id"), Reason: request.Reason})
	if err != nil {
		abortWithError(c, err)
		return
	}
	c.JSON(http.StatusCreated, reportResponse{Report: toReportResponse(result.Report)})
}

func (h *Handler) conversationAction(c *gin.Context, action func(context.Context, messageusecase.ConversationActionInput) (messageusecase.ConversationResult, error)) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	result, err := action(c.Request.Context(), messageusecase.ConversationActionInput{ViewerID: userID, ConversationID: c.Param("id")})
	if err != nil {
		abortWithError(c, err)
		return
	}
	c.JSON(http.StatusOK, toConversationMutationResponse(result))
}

func (h *Handler) AcceptRequest(c *gin.Context) {
	h.requestAction(c, h.messages.AcceptRequest)
}

func (h *Handler) RejectRequest(c *gin.Context) {
	h.requestAction(c, h.messages.RejectRequest)
}

func (h *Handler) requestAction(c *gin.Context, action func(context.Context, messageusecase.RequestActionInput) (messageusecase.ConversationResult, error)) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	result, err := action(c.Request.Context(), messageusecase.RequestActionInput{ViewerID: userID, RequestID: c.Param("id")})
	if err != nil {
		abortWithError(c, err)
		return
	}
	c.JSON(http.StatusOK, toConversationMutationResponse(result))
}

func (h *Handler) RecallMessage(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	result, err := h.messages.RecallMessage(c.Request.Context(), messageusecase.MessageActionInput{ViewerID: userID, MessageID: c.Param("id")})
	if err != nil {
		abortWithError(c, err)
		return
	}
	c.JSON(http.StatusOK, toConversationMutationResponse(result))
}

func (h *Handler) DeleteMessage(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if err := h.messages.DeleteMessage(c.Request.Context(), messageusecase.MessageActionInput{ViewerID: userID, MessageID: c.Param("id")}); err != nil {
		abortWithError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ReportMessage(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var request reportMessageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		abortWithError(c, apperr.New(apperr.CodeInvalidArgument, "invalid request body"))
		return
	}
	result, err := h.messages.ReportMessage(c.Request.Context(), messageusecase.MessageActionInput{ViewerID: userID, MessageID: c.Param("id"), Reason: request.Reason})
	if err != nil {
		abortWithError(c, err)
		return
	}
	c.JSON(http.StatusCreated, reportResponse{Report: toReportResponse(result.Report)})
}

func (h *Handler) BlockUser(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if err := h.messages.BlockUser(c.Request.Context(), messageusecase.BlockUserInput{ViewerID: userID, TargetUsername: c.Param("username")}); err != nil {
		abortWithError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) UnblockUser(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if err := h.messages.UnblockUser(c.Request.Context(), messageusecase.BlockUserInput{ViewerID: userID, TargetUsername: c.Param("username")}); err != nil {
		abortWithError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetPrivacy(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	result, err := h.messages.GetPrivacy(c.Request.Context(), userID)
	if err != nil {
		abortWithError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPrivacyResponse(result.Settings))
}

func (h *Handler) UpdatePrivacy(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var request updatePrivacyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		abortWithError(c, apperr.New(apperr.CodeInvalidArgument, "invalid request body"))
		return
	}
	result, err := h.messages.UpdatePrivacy(c.Request.Context(), messageusecase.PrivacyInput{ViewerID: userID, AllowMessages: request.AllowMessages, OnlineStatusEnabled: request.OnlineStatusEnabled})
	if err != nil {
		abortWithError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPrivacyResponse(result.Settings))
}

func (h *Handler) CreateRealtimeTicket(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var request realtimeTicketRequest
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			abortWithError(c, apperr.New(apperr.CodeInvalidArgument, "invalid request body"))
			return
		}
	}
	result, err := h.messages.CreateRealtimeTicket(c.Request.Context(), messageusecase.RealtimeTicketInput{ViewerID: userID, LastEventID: request.LastEventID})
	if err != nil {
		abortWithError(c, err)
		return
	}
	c.JSON(http.StatusCreated, realtimeTicketResponse{Ticket: result.Ticket, ExpiresAt: result.ExpiresAt})
}

func (h *Handler) RealtimeMessages(c *gin.Context) {
	result, err := h.messages.ConnectRealtime(c.Request.Context(), c.Query("ticket"))
	if err != nil {
		abortWithError(c, err)
		return
	}
	events := make([]realtimeEventResponse, 0, len(result.Events))
	for _, event := range result.Events {
		events = append(events, toRealtimeEventResponse(event))
	}
	websocket.Handler(func(conn *websocket.Conn) {
		defer conn.Close()
		_ = websocket.JSON.Send(conn, realtimeHelloResponse{Type: "connected", Events: events})
		var incoming string
		for {
			if err := websocket.Message.Receive(conn, &incoming); err != nil {
				return
			}
		}
	}).ServeHTTP(c.Writer, c.Request)
}

func currentUserID(c *gin.Context) (userdomain.UserID, bool) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		abortWithError(c, apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		return userdomain.UserID(""), false
	}
	return userID, true
}

func abortWithError(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}

func parseOptionalIntQuery(c *gin.Context, key string) (int, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, apperr.New(apperr.CodeInvalidArgument, key+" must be an integer")
	}
	return value, nil
}

func toDraft(request messageDraftRequest) messageusecase.MessageDraft {
	var share *messageusecase.ShareSnapshot
	if request.Share != nil {
		share = &messageusecase.ShareSnapshot{
			ShareType:         request.Share.ShareType,
			ShareID:           request.Share.ShareID,
			Title:             request.Share.Title,
			Summary:           request.Share.Summary,
			ThumbnailURL:      request.Share.ThumbnailURL,
			TargetURL:         request.Share.TargetURL,
			SnapshotCreatedAt: request.Share.SnapshotCreatedAt,
		}
	}
	return messageusecase.MessageDraft{Type: request.Type, Body: request.Body, ImageURL: request.ImageURL, Share: share}
}

func toConversationMutationResponse(result messageusecase.ConversationResult) conversationMutationResponse {
	response := conversationMutationResponse{Conversation: toConversationResponse(result.Conversation)}
	if result.Message != nil {
		message := toMessageResponse(*result.Message)
		response.Message = &message
	}
	return response
}

func toConversationResponse(conversation messageusecase.Conversation) conversationResponse {
	var lastMessage *messageSummaryResponse
	if conversation.LastMessage != nil {
		value := messageSummaryResponse{
			ID:        conversation.LastMessage.ID,
			Type:      conversation.LastMessage.Type,
			Text:      conversation.LastMessage.Text,
			Status:    conversation.LastMessage.Status,
			CreatedAt: conversation.LastMessage.CreatedAt,
		}
		lastMessage = &value
	}
	return conversationResponse{
		ID:                      conversation.ID,
		Box:                     conversation.Box,
		RequestID:               conversation.RequestID,
		RequestStatus:           conversation.RequestStatus,
		RequestDirection:        conversation.RequestDirection,
		ViewerCanAcceptRequest:  conversation.ViewerCanAcceptRequest,
		ViewerCanRejectRequest:  conversation.ViewerCanRejectRequest,
		ViewerCanReopen:         conversation.ViewerCanReopen,
		RequestCreatedByMe:      conversation.RequestCreatedByMe,
		RequestToMe:             conversation.RequestToMe,
		ConversationState:       conversation.ConversationState,
		Participant:             toUserSummaryResponse(conversation.Participant),
		LastMessage:             lastMessage,
		UnreadCount:             conversation.UnreadCount,
		UpdatedAt:               conversation.UpdatedAt,
		Pinned:                  conversation.Pinned,
		Muted:                   conversation.Muted,
		Archived:                conversation.Archived,
		Blocked:                 conversation.Blocked,
		CanSend:                 conversation.CanSend,
		DisableReason:           conversation.DisableReason,
		PeerOnlineStatusVisible: conversation.PeerOnlineStatusVisible,
		PeerOnline:              conversation.PeerOnline,
	}
}

func toMessageResponse(message messageusecase.Message) messageResponse {
	var share *shareSnapshotResponse
	if message.Share != nil {
		share = &shareSnapshotResponse{
			ShareType:         message.Share.ShareType,
			ShareID:           message.Share.ShareID,
			Title:             message.Share.Title,
			Summary:           message.Share.Summary,
			ThumbnailURL:      message.Share.ThumbnailURL,
			TargetURL:         message.Share.TargetURL,
			SnapshotCreatedAt: message.Share.SnapshotCreatedAt,
		}
	}
	return messageResponse{
		ID:             message.ID,
		ConversationID: message.ConversationID,
		Sender:         toUserSummaryResponse(message.Sender),
		Type:           message.Type,
		Body:           message.Body,
		ImageURL:       message.ImageURL,
		Share:          share,
		Status:         message.Status,
		CreatedAt:      message.CreatedAt,
		UpdatedAt:      message.UpdatedAt,
		RecalledAt:     message.RecalledAt,
		ViewerDeleted:  message.ViewerDeleted,
	}
}

func toUserSummaryResponse(user messageusecase.UserSummary) userSummaryResponse {
	return userSummaryResponse{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, AvatarURL: user.AvatarURL, Status: user.Status}
}

func toPrivacyResponse(settings messageusecase.PrivacySettings) privacyResponse {
	return privacyResponse{AllowMessages: settings.AllowMessages, OnlineStatusEnabled: settings.OnlineStatusEnabled, UpdatedAt: settings.UpdatedAt}
}

func toRealtimeEventResponse(event messageusecase.RealtimeEvent) realtimeEventResponse {
	return realtimeEventResponse{ID: event.ID, Type: event.Type, ConversationID: event.ConversationID, Payload: event.Payload, CreatedAt: event.CreatedAt}
}

func toReportResponse(report messageusecase.Report) messageReportResponse {
	return messageReportResponse{
		ID:             report.ID,
		ConversationID: report.ConversationID,
		MessageID:      report.MessageID,
		ReportedUserID: report.ReportedUserID,
		Reason:         report.Reason,
		ContextBefore:  report.ContextBefore,
		ContextAfter:   report.ContextAfter,
		CreatedAt:      report.CreatedAt,
	}
}

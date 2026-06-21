package messageusecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

const (
	MessageTypeText           = "text"
	MessageTypeImage          = "image"
	MessageTypeSharePost      = "share_post"
	MessageTypeShareComment   = "share_comment"
	MessageTypeShareUser      = "share_user"
	MessageTypeShareCommunity = "share_community"

	ConversationStatusAccepted = "accepted"
	ConversationStatusPending  = "pending"
	ConversationStatusRejected = "rejected"

	PrivacyEveryone = "everyone"
	PrivacyMutuals  = "mutuals"
	PrivacyNone     = "none"

	DefaultListLimit     = 20
	MaxListLimit         = 50
	DefaultMessagesLimit = 30
	MaxMessagesLimit     = 50
	RequestDailyLimit    = 10
	MessageRecallWindow  = 2 * time.Minute
	RealtimeTicketTTL    = 2 * time.Minute
)

type UseCase struct {
	repo  Repository
	users UserRepository
	now   func() time.Time
}

type Repository interface {
	GetPrivacySettings(ctx context.Context, userID userdomain.UserID) (PrivacySettingsRecord, error)
	UpsertPrivacySettings(ctx context.Context, userID userdomain.UserID, allowMessages string, onlineStatusEnabled bool, now time.Time) (PrivacySettingsRecord, error)
	IsBlockedEither(ctx context.Context, a userdomain.UserID, b userdomain.UserID) (bool, error)
	BlockUser(ctx context.Context, blockerID userdomain.UserID, blockedID userdomain.UserID, now time.Time) error
	UnblockUser(ctx context.Context, blockerID userdomain.UserID, blockedID userdomain.UserID) error
	FindDirectConversation(ctx context.Context, userID userdomain.UserID, peerID userdomain.UserID) (ConversationRecord, error)
	FindDirectConversationIncludingDeleted(ctx context.Context, userID userdomain.UserID, peerID userdomain.UserID) (ConversationRecord, error)
	CountRecentRequests(ctx context.Context, fromUserID userdomain.UserID, since time.Time) (int, error)
	CreateConversationWithMessage(ctx context.Context, input CreateConversationRecord) (ConversationRecord, MessageRecord, error)
	InsertMessage(ctx context.Context, input CreateMessageRecord) (MessageRecord, error)
	ReopenRejectedRequestWithMessage(ctx context.Context, input ReopenRejectedRequestRecord) (ConversationRecord, MessageRecord, error)
	ListConversations(ctx context.Context, userID userdomain.UserID, box string, limit int, offset int) ([]ConversationRecord, error)
	GetConversation(ctx context.Context, conversationID string, userID userdomain.UserID) (ConversationRecord, error)
	ListMessages(ctx context.Context, conversationID string, userID userdomain.UserID, beforeMessageID string, limit int) ([]MessageRecord, error)
	MarkConversationRead(ctx context.Context, conversationID string, userID userdomain.UserID, now time.Time) error
	SetConversationArchived(ctx context.Context, conversationID string, userID userdomain.UserID, archived bool, now time.Time) error
	SetConversationPinned(ctx context.Context, conversationID string, userID userdomain.UserID, pinned bool, now time.Time) error
	SetConversationMuted(ctx context.Context, conversationID string, userID userdomain.UserID, muted bool, now time.Time) error
	HideConversationForUser(ctx context.Context, conversationID string, userID userdomain.UserID, now time.Time) error
	AcceptMessageRequest(ctx context.Context, requestID string, userID userdomain.UserID, now time.Time) (ConversationRecord, error)
	RejectMessageRequest(ctx context.Context, requestID string, userID userdomain.UserID, now time.Time) (ConversationRecord, error)
	GetMessage(ctx context.Context, messageID string, userID userdomain.UserID) (MessageRecord, error)
	UpdateMessageStatus(ctx context.Context, messageID string, status string, now time.Time) (MessageRecord, error)
	HideMessageForUser(ctx context.Context, messageID string, userID userdomain.UserID, now time.Time) error
	CreateMessageReport(ctx context.Context, input CreateReportRecord) (ReportRecord, error)
	CreateRealtimeEvent(ctx context.Context, input RealtimeEventRecord) error
	CreateRealtimeTicket(ctx context.Context, ticket string, userID userdomain.UserID, lastEventID string, expiresAt time.Time, now time.Time) (RealtimeTicketRecord, error)
	ConsumeRealtimeTicket(ctx context.Context, ticket string, now time.Time) (RealtimeTicketRecord, error)
	ListRealtimeEventsAfter(ctx context.Context, userID userdomain.UserID, afterEventID string, limit int) ([]RealtimeEventRecord, error)
}

type UserRepository interface {
	FindByID(ctx context.Context, id userdomain.UserID) (*userdomain.User, error)
	FindByUsername(ctx context.Context, username userdomain.Username) (*userdomain.User, error)
	IsFollowing(ctx context.Context, followerID userdomain.UserID, followingID userdomain.UserID) (bool, error)
}

type StartConversationInput struct {
	ViewerID       userdomain.UserID
	TargetUsername string
	Message        MessageDraft
}

type SendMessageInput struct {
	ViewerID       userdomain.UserID
	ConversationID string
	Message        MessageDraft
}

type MessageDraft struct {
	Type         string
	Body         string
	ImageURL     string
	Share        *ShareSnapshot
	ClientNonce  string
	FallbackText string
}

type ShareSnapshot struct {
	ShareType         string
	ShareID           string
	Title             string
	Summary           string
	ThumbnailURL      string
	TargetURL         string
	SnapshotCreatedAt time.Time
}

type ListConversationsInput struct {
	ViewerID userdomain.UserID
	Box      string
	Limit    int
	Offset   int
}

type ListMessagesInput struct {
	ViewerID        userdomain.UserID
	ConversationID  string
	BeforeMessageID string
	Limit           int
}

type ConversationActionInput struct {
	ViewerID       userdomain.UserID
	ConversationID string
	Reason         string
}

type RequestActionInput struct {
	ViewerID  userdomain.UserID
	RequestID string
}

type MessageActionInput struct {
	ViewerID  userdomain.UserID
	MessageID string
	Reason    string
}

type BlockUserInput struct {
	ViewerID       userdomain.UserID
	TargetUsername string
}

type PrivacyInput struct {
	ViewerID            userdomain.UserID
	AllowMessages       *string
	OnlineStatusEnabled *bool
}

type RealtimeTicketInput struct {
	ViewerID    userdomain.UserID
	LastEventID string
}

type DMCapabilityInput struct {
	ViewerID userdomain.UserID
	TargetID userdomain.UserID
}

type SummaryResult struct {
	UnreadTotal         int
	RequestCount        int
	UnreadConversations int
	OnlineStatusEnabled bool
}

type ListConversationsResult struct {
	Conversations []Conversation
	Box           string
	Limit         int
	Offset        int
	NextOffset    int
	HasMore       bool
}

type ConversationResult struct {
	Conversation Conversation
	Message      *Message
}

type ListMessagesResult struct {
	Messages   []Message
	Limit      int
	HasMore    bool
	NextBefore string
}

type PrivacyResult struct {
	Settings PrivacySettings
}

type RealtimeTicketResult struct {
	Ticket    string
	ExpiresAt time.Time
}

type RealtimeConnectResult struct {
	UserID userdomain.UserID
	Events []RealtimeEvent
}

type ReportResult struct {
	Report Report
}

type DMCapability struct {
	CanStart             bool
	RequiresRequest      bool
	Reason               *string
	DirectConversationID *string
	ViewerRelation       string
}

type Conversation struct {
	ID                      string
	Box                     string
	RequestID               *string
	RequestStatus           string
	RequestDirection        string
	ViewerCanAcceptRequest  bool
	ViewerCanRejectRequest  bool
	ViewerCanReopen         bool
	RequestCreatedByMe      bool
	RequestToMe             bool
	ConversationState       string
	Participant             UserSummary
	LastMessage             *MessageSummary
	UnreadCount             int
	UpdatedAt               time.Time
	Pinned                  bool
	Muted                   bool
	Archived                bool
	Blocked                 bool
	CanSend                 bool
	DisableReason           *string
	PeerOnlineStatusVisible bool
	PeerOnline              bool
}

type MessageSummary struct {
	ID        string
	Type      string
	Text      string
	Status    string
	CreatedAt time.Time
}

type Message struct {
	ID             string
	ConversationID string
	Sender         UserSummary
	Type           string
	Body           string
	ImageURL       string
	Share          *ShareSnapshot
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	RecalledAt     *time.Time
	ViewerDeleted  bool
}

type UserSummary struct {
	ID          string
	Username    string
	DisplayName string
	AvatarURL   string
	Status      string
}

type PrivacySettings struct {
	AllowMessages       string
	OnlineStatusEnabled bool
	UpdatedAt           time.Time
}

type RealtimeEvent struct {
	ID             string
	Type           string
	ConversationID *string
	Payload        string
	CreatedAt      time.Time
}

type Report struct {
	ID             string
	ConversationID string
	MessageID      string
	ReportedUserID string
	Reason         string
	ContextBefore  string
	ContextAfter   string
	CreatedAt      time.Time
}

type PrivacySettingsRecord struct {
	AllowMessages       string
	OnlineStatusEnabled bool
	UpdatedAt           time.Time
}

type ConversationRecord struct {
	ID                      string
	Status                  string
	CreatedBy               userdomain.UserID
	RequestID               *string
	RequestStatus           string
	Peer                    userdomain.User
	LastMessage             *MessageRecord
	UnreadCount             int
	UpdatedAt               time.Time
	Pinned                  bool
	Muted                   bool
	Archived                bool
	Blocked                 bool
	PeerOnlineStatusEnabled bool
}

type CreateConversationRecord struct {
	ID        string
	RequestID string
	Status    string
	ViewerID  userdomain.UserID
	PeerID    userdomain.UserID
	Message   CreateMessageRecord
	Now       time.Time
}

type CreateMessageRecord struct {
	ID             string
	ConversationID string
	SenderID       userdomain.UserID
	Type           string
	Body           string
	ImageURL       string
	Share          *ShareSnapshot
	Now            time.Time
}

type ReopenRejectedRequestRecord struct {
	ConversationID string
	ViewerID       userdomain.UserID
	PeerID         userdomain.UserID
	Message        CreateMessageRecord
	Now            time.Time
}

type MessageRecord struct {
	ID             string
	ConversationID string
	Sender         userdomain.User
	Type           string
	Body           string
	ImageURL       string
	Share          *ShareSnapshot
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	RecalledAt     *time.Time
	ViewerDeleted  bool
}

type CreateReportRecord struct {
	ID             string
	ReporterID     userdomain.UserID
	ConversationID string
	MessageID      string
	ReportedUserID userdomain.UserID
	Reason         string
	Now            time.Time
}

type ReportRecord struct {
	ID             string
	ConversationID string
	MessageID      string
	ReportedUserID userdomain.UserID
	Reason         string
	ContextBefore  string
	ContextAfter   string
	CreatedAt      time.Time
}

type RealtimeEventRecord struct {
	ID             string
	UserID         userdomain.UserID
	ConversationID *string
	Type           string
	Payload        string
	CreatedAt      time.Time
}

type RealtimeTicketRecord struct {
	Ticket      string
	UserID      userdomain.UserID
	LastEventID string
	ExpiresAt   time.Time
}

func NewUseCase(repo Repository, users UserRepository, now func() time.Time) *UseCase {
	if now == nil {
		now = time.Now
	}
	return &UseCase{repo: repo, users: users, now: now}
}

func (uc *UseCase) GetSummary(ctx context.Context, viewerID userdomain.UserID) (SummaryResult, error) {
	if isBlankUserID(viewerID) {
		return SummaryResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	conversations, err := uc.repo.ListConversations(ctx, viewerID, "all", MaxListLimit, 0)
	if err != nil {
		return SummaryResult{}, fmt.Errorf("list message conversations for summary: %w", err)
	}
	var unreadTotal, unreadConversations, requestCount int
	for _, conversation := range conversations {
		if conversation.Status == ConversationStatusPending && conversation.CreatedBy != viewerID {
			requestCount++
		}
		if conversation.UnreadCount > 0 {
			unreadConversations++
			unreadTotal += conversation.UnreadCount
		}
	}
	settings, err := uc.repo.GetPrivacySettings(ctx, viewerID)
	if err != nil {
		return SummaryResult{}, err
	}
	return SummaryResult{
		UnreadTotal:         unreadTotal,
		RequestCount:        requestCount,
		UnreadConversations: unreadConversations,
		OnlineStatusEnabled: settings.OnlineStatusEnabled,
	}, nil
}

func (uc *UseCase) ListConversations(ctx context.Context, input ListConversationsInput) (ListConversationsResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ListConversationsResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	box := normalizeBox(input.Box)
	limit, offset, err := normalizePagination(input.Limit, input.Offset, DefaultListLimit, MaxListLimit)
	if err != nil {
		return ListConversationsResult{}, err
	}
	records, err := uc.repo.ListConversations(ctx, input.ViewerID, box, limit+1, offset)
	if err != nil {
		return ListConversationsResult{}, fmt.Errorf("list message conversations: %w", err)
	}
	records, hasMore := trimConversationRecords(records, limit)
	result := ListConversationsResult{
		Conversations: make([]Conversation, 0, len(records)),
		Box:           box,
		Limit:         limit,
		Offset:        offset,
		NextOffset:    offset + len(records),
		HasMore:       hasMore,
	}
	for _, record := range records {
		result.Conversations = append(result.Conversations, uc.toConversation(record, input.ViewerID))
	}
	return result, nil
}

func (uc *UseCase) StartConversation(ctx context.Context, input StartConversationInput) (ConversationResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ConversationResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	target, err := uc.findActiveUserByUsername(ctx, input.TargetUsername)
	if err != nil {
		return ConversationResult{}, err
	}
	if target.ID() == input.ViewerID {
		return ConversationResult{}, apperr.New(apperr.CodeInvalidArgument, "can't message yourself")
	}
	existing, err := uc.repo.FindDirectConversation(ctx, input.ViewerID, target.ID())
	if err == nil {
		if existing.Blocked {
			return ConversationResult{}, apperr.New(apperr.CodeForbidden, "message is blocked")
		}
		if existing.Status == ConversationStatusRejected || existing.RequestStatus == ConversationStatusRejected {
			if canReopenRejectedRequest(existing, input.ViewerID) {
				return uc.reopenRejectedRequest(ctx, existing, input.ViewerID, input.Message)
			}
			return ConversationResult{}, messageRequestRejectedError()
		}
		if existing.Status == ConversationStatusPending && existing.CreatedBy == input.ViewerID && !input.Message.isZero() {
			return ConversationResult{}, apperr.New(apperr.CodeConflict, "message request already pending")
		}
		return ConversationResult{Conversation: uc.toConversation(existing, input.ViewerID)}, nil
	}
	if !apperr.IsCode(err, apperr.CodeNotFound) {
		return ConversationResult{}, fmt.Errorf("find direct conversation: %w", err)
	}
	deletedExisting, err := uc.repo.FindDirectConversationIncludingDeleted(ctx, input.ViewerID, target.ID())
	if err == nil {
		if deletedExisting.Blocked {
			return ConversationResult{}, apperr.New(apperr.CodeForbidden, "message is blocked")
		}
		if deletedExisting.Status == ConversationStatusRejected || deletedExisting.RequestStatus == ConversationStatusRejected {
			if canReopenRejectedRequest(deletedExisting, input.ViewerID) {
				return uc.reopenRejectedRequest(ctx, deletedExisting, input.ViewerID, input.Message)
			}
			return ConversationResult{}, messageRequestRejectedError()
		}
		if deletedExisting.Status == ConversationStatusPending && deletedExisting.CreatedBy == input.ViewerID && !input.Message.isZero() {
			return ConversationResult{}, apperr.New(apperr.CodeConflict, "message request already pending")
		}
		return ConversationResult{}, apperr.New(apperr.CodeConflict, "message conversation already exists")
	}
	if !apperr.IsCode(err, apperr.CodeNotFound) {
		return ConversationResult{}, fmt.Errorf("find deleted direct conversation: %w", err)
	}
	capability, err := uc.GetDMCapability(ctx, DMCapabilityInput{ViewerID: input.ViewerID, TargetID: target.ID()})
	if err != nil {
		return ConversationResult{}, err
	}
	if !capability.CanStart {
		return ConversationResult{}, apperr.New(apperr.CodeForbidden, firstReason(capability.Reason, "message is not allowed"))
	}
	draft, err := normalizeDraft(input.Message)
	if err != nil {
		return ConversationResult{}, err
	}
	if draft.isZero() {
		return ConversationResult{}, apperr.New(apperr.CodeInvalidArgument, "initial message is required")
	}
	if capability.RequiresRequest {
		count, err := uc.repo.CountRecentRequests(ctx, input.ViewerID, uc.now().UTC().Add(-24*time.Hour))
		if err != nil {
			return ConversationResult{}, err
		}
		if count >= RequestDailyLimit {
			return ConversationResult{}, apperr.New(apperr.CodeRateLimited, "message request rate limited")
		}
	}
	now := uc.now().UTC()
	status := ConversationStatusAccepted
	requestID := ""
	if capability.RequiresRequest {
		status = ConversationStatusPending
		requestID = uuid.NewString()
	}
	conversation, message, err := uc.repo.CreateConversationWithMessage(ctx, CreateConversationRecord{
		ID:        uuid.NewString(),
		RequestID: requestID,
		Status:    status,
		ViewerID:  input.ViewerID,
		PeerID:    target.ID(),
		Message: CreateMessageRecord{
			ID:       uuid.NewString(),
			SenderID: input.ViewerID,
			Type:     draft.Type,
			Body:     draft.Body,
			ImageURL: draft.ImageURL,
			Share:    draft.Share,
			Now:      now,
		},
		Now: now,
	})
	if err != nil {
		return ConversationResult{}, fmt.Errorf("create message conversation: %w", err)
	}
	uc.emitConversationEvents(ctx, conversation.ID, []userdomain.UserID{input.ViewerID, target.ID()}, "message_created")
	return ConversationResult{Conversation: uc.toConversation(conversation, input.ViewerID), Message: ptrMessage(uc.toMessage(message))}, nil
}

func (uc *UseCase) SendMessage(ctx context.Context, input SendMessageInput) (ConversationResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ConversationResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	conversation, err := uc.repo.GetConversation(ctx, input.ConversationID, input.ViewerID)
	if err != nil {
		return ConversationResult{}, err
	}
	if conversation.Blocked {
		return ConversationResult{}, apperr.New(apperr.CodeForbidden, "message is blocked")
	}
	if conversation.Status == ConversationStatusPending && conversation.CreatedBy == input.ViewerID {
		return ConversationResult{}, apperr.New(apperr.CodeForbidden, "message request is pending")
	}
	if conversation.Status == ConversationStatusRejected || conversation.RequestStatus == ConversationStatusRejected {
		if canReopenRejectedRequest(conversation, input.ViewerID) {
			return uc.reopenRejectedRequest(ctx, conversation, input.ViewerID, input.Message)
		}
		return ConversationResult{}, messageRequestRejectedError()
	}
	if conversation.Status != ConversationStatusAccepted {
		return ConversationResult{}, apperr.New(apperr.CodeForbidden, "conversation is not active")
	}
	draft, err := normalizeDraft(input.Message)
	if err != nil {
		return ConversationResult{}, err
	}
	if draft.isZero() {
		return ConversationResult{}, apperr.New(apperr.CodeInvalidArgument, "message body is required")
	}
	now := uc.now().UTC()
	message, err := uc.repo.InsertMessage(ctx, CreateMessageRecord{
		ID:             uuid.NewString(),
		ConversationID: conversation.ID,
		SenderID:       input.ViewerID,
		Type:           draft.Type,
		Body:           draft.Body,
		ImageURL:       draft.ImageURL,
		Share:          draft.Share,
		Now:            now,
	})
	if err != nil {
		return ConversationResult{}, fmt.Errorf("insert message: %w", err)
	}
	uc.emitConversationEvents(ctx, conversation.ID, []userdomain.UserID{input.ViewerID, conversation.Peer.ID()}, "message_created")
	conversation, _ = uc.repo.GetConversation(ctx, conversation.ID, input.ViewerID)
	return ConversationResult{Conversation: uc.toConversation(conversation, input.ViewerID), Message: ptrMessage(uc.toMessage(message))}, nil
}

func (uc *UseCase) reopenRejectedRequest(ctx context.Context, conversation ConversationRecord, viewerID userdomain.UserID, messageDraft MessageDraft) (ConversationResult, error) {
	draft, err := normalizeDraft(messageDraft)
	if err != nil {
		return ConversationResult{}, err
	}
	if draft.isZero() {
		return ConversationResult{}, apperr.New(apperr.CodeInvalidArgument, "message body is required")
	}
	now := uc.now().UTC()
	updated, message, err := uc.repo.ReopenRejectedRequestWithMessage(ctx, ReopenRejectedRequestRecord{
		ConversationID: conversation.ID,
		ViewerID:       viewerID,
		PeerID:         conversation.Peer.ID(),
		Message: CreateMessageRecord{
			ID:       uuid.NewString(),
			SenderID: viewerID,
			Type:     draft.Type,
			Body:     draft.Body,
			ImageURL: draft.ImageURL,
			Share:    draft.Share,
			Now:      now,
		},
		Now: now,
	})
	if err != nil {
		return ConversationResult{}, err
	}
	uc.emitConversationEvents(ctx, updated.ID, []userdomain.UserID{viewerID, conversation.Peer.ID()}, "message_created")
	return ConversationResult{Conversation: uc.toConversation(updated, viewerID), Message: ptrMessage(uc.toMessage(message))}, nil
}

func (uc *UseCase) ListMessages(ctx context.Context, input ListMessagesInput) (ListMessagesResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ListMessagesResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	limit, _, err := normalizePagination(input.Limit, 0, DefaultMessagesLimit, MaxMessagesLimit)
	if err != nil {
		return ListMessagesResult{}, err
	}
	records, err := uc.repo.ListMessages(ctx, input.ConversationID, input.ViewerID, strings.TrimSpace(input.BeforeMessageID), limit+1)
	if err != nil {
		return ListMessagesResult{}, err
	}
	records, hasMore := trimMessageRecords(records, limit)
	result := ListMessagesResult{Messages: make([]Message, 0, len(records)), Limit: limit, HasMore: hasMore}
	for _, record := range records {
		result.Messages = append(result.Messages, uc.toMessage(record))
	}
	if len(result.Messages) > 0 {
		result.NextBefore = result.Messages[len(result.Messages)-1].ID
	}
	return result, nil
}

func (uc *UseCase) MarkConversationRead(ctx context.Context, input ConversationActionInput) (ConversationResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ConversationResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if err := uc.repo.MarkConversationRead(ctx, input.ConversationID, input.ViewerID, uc.now().UTC()); err != nil {
		return ConversationResult{}, err
	}
	conversation, err := uc.repo.GetConversation(ctx, input.ConversationID, input.ViewerID)
	if err != nil {
		return ConversationResult{}, err
	}
	uc.emitConversationEvents(ctx, conversation.ID, []userdomain.UserID{input.ViewerID}, "unread_updated")
	return ConversationResult{Conversation: uc.toConversation(conversation, input.ViewerID)}, nil
}

func (uc *UseCase) SetConversationArchived(ctx context.Context, input ConversationActionInput, archived bool) (ConversationResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ConversationResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if err := uc.repo.SetConversationArchived(ctx, input.ConversationID, input.ViewerID, archived, uc.now().UTC()); err != nil {
		return ConversationResult{}, err
	}
	conversation, err := uc.repo.GetConversation(ctx, input.ConversationID, input.ViewerID)
	if err != nil {
		return ConversationResult{}, err
	}
	return ConversationResult{Conversation: uc.toConversation(conversation, input.ViewerID)}, nil
}

func (uc *UseCase) SetConversationPinned(ctx context.Context, input ConversationActionInput, pinned bool) (ConversationResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ConversationResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if err := uc.repo.SetConversationPinned(ctx, input.ConversationID, input.ViewerID, pinned, uc.now().UTC()); err != nil {
		return ConversationResult{}, err
	}
	conversation, err := uc.repo.GetConversation(ctx, input.ConversationID, input.ViewerID)
	if err != nil {
		return ConversationResult{}, err
	}
	return ConversationResult{Conversation: uc.toConversation(conversation, input.ViewerID)}, nil
}

func (uc *UseCase) SetConversationMuted(ctx context.Context, input ConversationActionInput, muted bool) (ConversationResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ConversationResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if err := uc.repo.SetConversationMuted(ctx, input.ConversationID, input.ViewerID, muted, uc.now().UTC()); err != nil {
		return ConversationResult{}, err
	}
	conversation, err := uc.repo.GetConversation(ctx, input.ConversationID, input.ViewerID)
	if err != nil {
		return ConversationResult{}, err
	}
	return ConversationResult{Conversation: uc.toConversation(conversation, input.ViewerID)}, nil
}

func (uc *UseCase) DeleteConversation(ctx context.Context, input ConversationActionInput) error {
	if isBlankUserID(input.ViewerID) {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	return uc.repo.HideConversationForUser(ctx, input.ConversationID, input.ViewerID, uc.now().UTC())
}

func (uc *UseCase) AcceptRequest(ctx context.Context, input RequestActionInput) (ConversationResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ConversationResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	conversation, err := uc.repo.AcceptMessageRequest(ctx, input.RequestID, input.ViewerID, uc.now().UTC())
	if err != nil {
		return ConversationResult{}, err
	}
	uc.emitConversationEvents(ctx, conversation.ID, []userdomain.UserID{input.ViewerID, conversation.Peer.ID()}, "request_accepted")
	return ConversationResult{Conversation: uc.toConversation(conversation, input.ViewerID)}, nil
}

func (uc *UseCase) RejectRequest(ctx context.Context, input RequestActionInput) (ConversationResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ConversationResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	conversation, err := uc.repo.RejectMessageRequest(ctx, input.RequestID, input.ViewerID, uc.now().UTC())
	if err != nil {
		return ConversationResult{}, err
	}
	uc.emitConversationEvents(ctx, conversation.ID, []userdomain.UserID{input.ViewerID, conversation.Peer.ID()}, "request_rejected")
	return ConversationResult{Conversation: uc.toConversation(conversation, input.ViewerID)}, nil
}

func (uc *UseCase) RecallMessage(ctx context.Context, input MessageActionInput) (ConversationResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ConversationResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	message, err := uc.repo.GetMessage(ctx, input.MessageID, input.ViewerID)
	if err != nil {
		return ConversationResult{}, err
	}
	if message.Sender.ID() != input.ViewerID {
		return ConversationResult{}, apperr.New(apperr.CodeForbidden, "only sender can recall message")
	}
	if message.ViewerDeleted {
		return ConversationResult{}, apperr.New(apperr.CodeConflict, "message cannot be recalled")
	}
	if message.Status == "recalled" || message.RecalledAt != nil {
		return ConversationResult{}, apperr.New(apperr.CodeConflict, "message already recalled")
	}
	if message.Status != "visible" {
		return ConversationResult{}, apperr.New(apperr.CodeConflict, "message cannot be recalled")
	}
	now := uc.now().UTC()
	if now.After(message.CreatedAt.Add(MessageRecallWindow)) {
		return ConversationResult{}, apperr.New(apperr.CodeMessageRecallExpired, "message recall window expired")
	}
	conversation, err := uc.repo.GetConversation(ctx, message.ConversationID, input.ViewerID)
	if err != nil {
		return ConversationResult{}, err
	}
	message, err = uc.repo.UpdateMessageStatus(ctx, input.MessageID, "recalled", now)
	if err != nil {
		return ConversationResult{}, err
	}
	uc.emitConversationEvents(ctx, message.ConversationID, []userdomain.UserID{input.ViewerID, conversation.Peer.ID()}, "message_recalled")
	return ConversationResult{Conversation: uc.toConversation(conversation, input.ViewerID), Message: ptrMessage(uc.toMessage(message))}, nil
}

func (uc *UseCase) DeleteMessage(ctx context.Context, input MessageActionInput) error {
	if isBlankUserID(input.ViewerID) {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if _, err := uc.repo.GetMessage(ctx, input.MessageID, input.ViewerID); err != nil {
		return err
	}
	return uc.repo.HideMessageForUser(ctx, input.MessageID, input.ViewerID, uc.now().UTC())
}

func (uc *UseCase) ReportMessage(ctx context.Context, input MessageActionInput) (ReportResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ReportResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return ReportResult{}, apperr.New(apperr.CodeInvalidArgument, "report reason is required")
	}
	message, err := uc.repo.GetMessage(ctx, input.MessageID, input.ViewerID)
	if err != nil {
		return ReportResult{}, err
	}
	report, err := uc.repo.CreateMessageReport(ctx, CreateReportRecord{
		ID:             uuid.NewString(),
		ReporterID:     input.ViewerID,
		ConversationID: message.ConversationID,
		MessageID:      message.ID,
		ReportedUserID: message.Sender.ID(),
		Reason:         reason,
		Now:            uc.now().UTC(),
	})
	if err != nil {
		return ReportResult{}, fmt.Errorf("create message report: %w", err)
	}
	return ReportResult{Report: toReport(report)}, nil
}

func (uc *UseCase) ReportConversation(ctx context.Context, input ConversationActionInput) (ReportResult, error) {
	if isBlankUserID(input.ViewerID) {
		return ReportResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return ReportResult{}, apperr.New(apperr.CodeInvalidArgument, "report reason is required")
	}
	conversation, err := uc.repo.GetConversation(ctx, input.ConversationID, input.ViewerID)
	if err != nil {
		return ReportResult{}, err
	}
	messages, err := uc.repo.ListMessages(ctx, input.ConversationID, input.ViewerID, "", MaxMessagesLimit)
	if err != nil {
		return ReportResult{}, err
	}
	var target *MessageRecord
	for i := range messages {
		if messages[i].Sender.ID() != input.ViewerID {
			target = &messages[i]
			break
		}
	}
	if target == nil {
		return ReportResult{}, apperr.New(apperr.CodeNotFound, "reportable message not found")
	}
	report, err := uc.repo.CreateMessageReport(ctx, CreateReportRecord{
		ID:             uuid.NewString(),
		ReporterID:     input.ViewerID,
		ConversationID: conversation.ID,
		MessageID:      target.ID,
		ReportedUserID: target.Sender.ID(),
		Reason:         reason,
		Now:            uc.now().UTC(),
	})
	if err != nil {
		return ReportResult{}, fmt.Errorf("create conversation report: %w", err)
	}
	return ReportResult{Report: toReport(report)}, nil
}

func (uc *UseCase) BlockUser(ctx context.Context, input BlockUserInput) error {
	if isBlankUserID(input.ViewerID) {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	target, err := uc.findActiveUserByUsername(ctx, input.TargetUsername)
	if err != nil {
		return err
	}
	if target.ID() == input.ViewerID {
		return apperr.New(apperr.CodeInvalidArgument, "can't block yourself")
	}
	if err := uc.repo.BlockUser(ctx, input.ViewerID, target.ID(), uc.now().UTC()); err != nil {
		return err
	}
	uc.emitConversationEvents(ctx, "", []userdomain.UserID{input.ViewerID, target.ID()}, "block_changed")
	return nil
}

func (uc *UseCase) UnblockUser(ctx context.Context, input BlockUserInput) error {
	if isBlankUserID(input.ViewerID) {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	target, err := uc.findActiveUserByUsername(ctx, input.TargetUsername)
	if err != nil {
		return err
	}
	return uc.repo.UnblockUser(ctx, input.ViewerID, target.ID())
}

func (uc *UseCase) GetPrivacy(ctx context.Context, viewerID userdomain.UserID) (PrivacyResult, error) {
	if isBlankUserID(viewerID) {
		return PrivacyResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	settings, err := uc.repo.GetPrivacySettings(ctx, viewerID)
	if err != nil {
		return PrivacyResult{}, err
	}
	return PrivacyResult{Settings: toPrivacySettings(settings)}, nil
}

func (uc *UseCase) UpdatePrivacy(ctx context.Context, input PrivacyInput) (PrivacyResult, error) {
	if isBlankUserID(input.ViewerID) {
		return PrivacyResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	current, err := uc.repo.GetPrivacySettings(ctx, input.ViewerID)
	if err != nil {
		return PrivacyResult{}, err
	}
	allow := current.AllowMessages
	online := current.OnlineStatusEnabled
	if input.AllowMessages != nil {
		allow, err = normalizeAllowMessages(*input.AllowMessages)
		if err != nil {
			return PrivacyResult{}, err
		}
	}
	if input.OnlineStatusEnabled != nil {
		online = *input.OnlineStatusEnabled
	}
	updated, err := uc.repo.UpsertPrivacySettings(ctx, input.ViewerID, allow, online, uc.now().UTC())
	if err != nil {
		return PrivacyResult{}, err
	}
	return PrivacyResult{Settings: toPrivacySettings(updated)}, nil
}

func (uc *UseCase) CreateRealtimeTicket(ctx context.Context, input RealtimeTicketInput) (RealtimeTicketResult, error) {
	if isBlankUserID(input.ViewerID) {
		return RealtimeTicketResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	ticket, err := randomTicket()
	if err != nil {
		return RealtimeTicketResult{}, err
	}
	now := uc.now().UTC()
	record, err := uc.repo.CreateRealtimeTicket(ctx, ticket, input.ViewerID, strings.TrimSpace(input.LastEventID), now.Add(RealtimeTicketTTL), now)
	if err != nil {
		return RealtimeTicketResult{}, err
	}
	return RealtimeTicketResult{Ticket: record.Ticket, ExpiresAt: record.ExpiresAt}, nil
}

func (uc *UseCase) ConnectRealtime(ctx context.Context, ticket string) (RealtimeConnectResult, error) {
	record, err := uc.repo.ConsumeRealtimeTicket(ctx, strings.TrimSpace(ticket), uc.now().UTC())
	if err != nil {
		return RealtimeConnectResult{}, err
	}
	events, err := uc.repo.ListRealtimeEventsAfter(ctx, record.UserID, record.LastEventID, 100)
	if err != nil {
		return RealtimeConnectResult{}, err
	}
	result := RealtimeConnectResult{UserID: record.UserID, Events: make([]RealtimeEvent, 0, len(events))}
	for _, event := range events {
		result.Events = append(result.Events, toRealtimeEvent(event))
	}
	return result, nil
}

func (uc *UseCase) GetDMCapability(ctx context.Context, input DMCapabilityInput) (DMCapability, error) {
	if isBlankUserID(input.ViewerID) {
		reason := "unauthenticated"
		return DMCapability{CanStart: false, RequiresRequest: true, Reason: &reason, ViewerRelation: "none"}, nil
	}
	if input.ViewerID == input.TargetID {
		reason := "self"
		return DMCapability{CanStart: false, Reason: &reason, ViewerRelation: "self"}, nil
	}
	target, err := uc.users.FindByID(ctx, input.TargetID)
	if err != nil {
		return DMCapability{}, err
	}
	if !target.CanLogin() {
		reason := "unavailable"
		return DMCapability{CanStart: false, Reason: &reason, ViewerRelation: "none"}, nil
	}
	viewerFollows, err := uc.users.IsFollowing(ctx, input.ViewerID, input.TargetID)
	if err != nil {
		return DMCapability{}, err
	}
	targetFollows, err := uc.users.IsFollowing(ctx, input.TargetID, input.ViewerID)
	if err != nil {
		return DMCapability{}, err
	}
	relation := viewerRelation(viewerFollows, targetFollows)
	blocked, err := uc.repo.IsBlockedEither(ctx, input.ViewerID, input.TargetID)
	if err != nil {
		return DMCapability{}, err
	}
	if blocked {
		reason := "blocked"
		return DMCapability{CanStart: false, Reason: &reason, ViewerRelation: relation}, nil
	}
	settings, err := uc.repo.GetPrivacySettings(ctx, input.TargetID)
	if err != nil {
		return DMCapability{}, err
	}
	var directID *string
	if conversation, err := uc.repo.FindDirectConversation(ctx, input.ViewerID, input.TargetID); err == nil {
		directID = &conversation.ID
	}
	if settings.AllowMessages == PrivacyNone {
		reason := "privacy"
		return DMCapability{CanStart: false, Reason: &reason, DirectConversationID: directID, ViewerRelation: relation}, nil
	}
	if settings.AllowMessages == PrivacyMutuals && relation != "mutual" {
		reason := "privacy"
		return DMCapability{CanStart: false, Reason: &reason, DirectConversationID: directID, ViewerRelation: relation}, nil
	}
	return DMCapability{
		CanStart:             true,
		RequiresRequest:      relation != "mutual",
		DirectConversationID: directID,
		ViewerRelation:       relation,
	}, nil
}

func (uc *UseCase) findActiveUserByUsername(ctx context.Context, raw string) (*userdomain.User, error) {
	username, err := userdomain.NewUsername(raw)
	if err != nil {
		return nil, err
	}
	user, err := uc.users.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if !user.CanLogin() {
		return nil, apperr.New(apperr.CodeNotFound, "user not found")
	}
	return user, nil
}

func (uc *UseCase) emitConversationEvents(ctx context.Context, conversationID string, users []userdomain.UserID, eventType string) {
	for _, userID := range users {
		if isBlankUserID(userID) {
			continue
		}
		var conversationIDPtr *string
		if strings.TrimSpace(conversationID) != "" {
			value := conversationID
			conversationIDPtr = &value
		}
		_ = uc.repo.CreateRealtimeEvent(ctx, RealtimeEventRecord{
			ID:             uuid.NewString(),
			UserID:         userID,
			ConversationID: conversationIDPtr,
			Type:           eventType,
			Payload:        "{}",
			CreatedAt:      uc.now().UTC(),
		})
	}
}

func (uc *UseCase) toConversation(record ConversationRecord, viewerID userdomain.UserID) Conversation {
	disableReason := (*string)(nil)
	canSend := record.Status == ConversationStatusAccepted && !record.Blocked
	if record.Blocked {
		reason := "blocked"
		disableReason = &reason
	} else if record.Status == ConversationStatusPending {
		reason := "request_pending"
		disableReason = &reason
	} else if record.Status != ConversationStatusAccepted {
		reason := "inactive"
		disableReason = &reason
	}
	var requestID *string
	if record.RequestID != nil {
		value := *record.RequestID
		requestID = &value
	}
	requestDirection := "none"
	requestCreatedByMe := false
	requestToMe := false
	viewerCanAcceptRequest := false
	viewerCanRejectRequest := false
	viewerCanReopen := canReopenRejectedRequest(record, viewerID)
	conversationState := "normal"
	if record.RequestID != nil && record.RequestStatus == ConversationStatusPending {
		if record.CreatedBy == viewerID {
			requestDirection = "outgoing"
			requestCreatedByMe = true
			conversationState = "outgoing_request"
		} else {
			requestDirection = "incoming"
			requestToMe = true
			viewerCanAcceptRequest = true
			viewerCanRejectRequest = true
			conversationState = "incoming_request"
		}
	}
	if record.Blocked {
		conversationState = "blocked"
	} else if record.Status != ConversationStatusAccepted && record.Status != ConversationStatusPending {
		conversationState = "disabled"
	}
	conversation := Conversation{
		ID:                      record.ID,
		Box:                     conversationBox(record, viewerID),
		RequestID:               requestID,
		RequestStatus:           record.RequestStatus,
		RequestDirection:        requestDirection,
		ViewerCanAcceptRequest:  viewerCanAcceptRequest,
		ViewerCanRejectRequest:  viewerCanRejectRequest,
		ViewerCanReopen:         viewerCanReopen,
		RequestCreatedByMe:      requestCreatedByMe,
		RequestToMe:             requestToMe,
		ConversationState:       conversationState,
		Participant:             toUserSummary(record.Peer),
		UnreadCount:             record.UnreadCount,
		UpdatedAt:               record.UpdatedAt,
		Pinned:                  record.Pinned,
		Muted:                   record.Muted,
		Archived:                record.Archived,
		Blocked:                 record.Blocked,
		CanSend:                 canSend,
		DisableReason:           disableReason,
		PeerOnlineStatusVisible: record.PeerOnlineStatusEnabled && record.Status == ConversationStatusAccepted,
		PeerOnline:              false,
	}
	if record.LastMessage != nil {
		conversation.LastMessage = toMessageSummary(*record.LastMessage)
	}
	return conversation
}

func canReopenRejectedRequest(record ConversationRecord, viewerID userdomain.UserID) bool {
	if isBlankUserID(viewerID) || record.Blocked || record.RequestID == nil {
		return false
	}
	if record.Status != ConversationStatusRejected && record.RequestStatus != ConversationStatusRejected {
		return false
	}
	return record.CreatedBy != viewerID && record.Peer.ID() == record.CreatedBy
}

func messageRequestRejectedError() error {
	return apperr.New(apperr.CodeMessageRequestRejected, "message request was ignored")
}

func (uc *UseCase) toMessage(record MessageRecord) Message {
	return Message{
		ID:             record.ID,
		ConversationID: record.ConversationID,
		Sender:         toUserSummary(record.Sender),
		Type:           record.Type,
		Body:           record.Body,
		ImageURL:       record.ImageURL,
		Share:          cloneShare(record.Share),
		Status:         record.Status,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
		RecalledAt:     record.RecalledAt,
		ViewerDeleted:  record.ViewerDeleted,
	}
}

func normalizeDraft(input MessageDraft) (MessageDraft, error) {
	draft := MessageDraft{
		Type:     strings.TrimSpace(input.Type),
		Body:     strings.TrimSpace(input.Body),
		ImageURL: strings.TrimSpace(input.ImageURL),
		Share:    cloneShare(input.Share),
	}
	if draft.Type == "" {
		draft.Type = MessageTypeText
	}
	switch draft.Type {
	case MessageTypeText:
		if draft.Body == "" {
			return MessageDraft{}, apperr.New(apperr.CodeInvalidArgument, "message body is required")
		}
	case MessageTypeImage:
		if draft.ImageURL == "" {
			return MessageDraft{}, apperr.New(apperr.CodeInvalidArgument, "image url is required")
		}
	case MessageTypeSharePost, MessageTypeShareComment, MessageTypeShareUser, MessageTypeShareCommunity:
		if draft.Share == nil {
			return MessageDraft{}, apperr.New(apperr.CodeInvalidArgument, "share snapshot is required")
		}
		expected := strings.TrimPrefix(draft.Type, "share_")
		draft.Share.ShareType = expected
		draft.Share.ShareID = strings.TrimSpace(draft.Share.ShareID)
		draft.Share.Title = strings.TrimSpace(draft.Share.Title)
		draft.Share.Summary = strings.TrimSpace(draft.Share.Summary)
		draft.Share.ThumbnailURL = strings.TrimSpace(draft.Share.ThumbnailURL)
		draft.Share.TargetURL = strings.TrimSpace(draft.Share.TargetURL)
		if draft.Share.ShareID == "" {
			return MessageDraft{}, apperr.New(apperr.CodeInvalidArgument, "share id is required")
		}
		if draft.Share.Title == "" {
			draft.Share.Title = "内容暂不可查看"
		}
		if draft.Share.SnapshotCreatedAt.IsZero() {
			draft.Share.SnapshotCreatedAt = time.Now().UTC()
		}
	default:
		return MessageDraft{}, apperr.New(apperr.CodeInvalidArgument, "message type is invalid")
	}
	return draft, nil
}

func (draft MessageDraft) isZero() bool {
	return strings.TrimSpace(draft.Body) == "" && strings.TrimSpace(draft.ImageURL) == "" && draft.Share == nil
}

func normalizePagination(limit int, offset int, defaultLimit int, maxLimit int) (int, int, error) {
	if limit < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "limit must be non-negative")
	}
	if offset < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "offset must be non-negative")
	}
	if limit == 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return limit, offset, nil
}

func normalizeBox(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "friends":
		return "friends"
	case "requests":
		return "requests"
	case "archived":
		return "archived"
	default:
		return "all"
	}
}

func normalizeAllowMessages(raw string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", PrivacyEveryone:
		return PrivacyEveryone, nil
	case PrivacyMutuals:
		return PrivacyMutuals, nil
	case PrivacyNone:
		return PrivacyNone, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "message privacy setting is invalid")
	}
}

func viewerRelation(viewerFollows bool, targetFollows bool) string {
	if viewerFollows && targetFollows {
		return "mutual"
	}
	if viewerFollows {
		return "following"
	}
	if targetFollows {
		return "followed_by"
	}
	return "none"
}

func conversationBox(record ConversationRecord, viewerID userdomain.UserID) string {
	if record.Archived {
		return "archived"
	}
	if record.Status == ConversationStatusPending && record.CreatedBy != viewerID {
		return "requests"
	}
	return "friends"
}

func toUserSummary(user userdomain.User) UserSummary {
	displayName := user.DisplayName().String()
	if displayName == "" {
		displayName = user.Username().String()
	}
	return UserSummary{
		ID:          user.ID().String(),
		Username:    user.Username().String(),
		DisplayName: displayName,
		AvatarURL:   user.AvatarURL().String(),
		Status:      user.Status().String(),
	}
}

func toMessageSummary(record MessageRecord) *MessageSummary {
	text := record.Body
	if record.Status == "recalled" {
		text = "消息已撤回"
	} else if strings.HasPrefix(record.Type, "share_") && record.Share != nil {
		text = record.Share.Title
	} else if record.Type == MessageTypeImage {
		text = "[图片]"
	}
	return &MessageSummary{ID: record.ID, Type: record.Type, Text: text, Status: record.Status, CreatedAt: record.CreatedAt}
}

func toPrivacySettings(record PrivacySettingsRecord) PrivacySettings {
	allow := record.AllowMessages
	if allow == "" {
		allow = PrivacyEveryone
	}
	return PrivacySettings{AllowMessages: allow, OnlineStatusEnabled: record.OnlineStatusEnabled, UpdatedAt: record.UpdatedAt}
}

func toRealtimeEvent(record RealtimeEventRecord) RealtimeEvent {
	var conversationID *string
	if record.ConversationID != nil {
		value := *record.ConversationID
		conversationID = &value
	}
	return RealtimeEvent{ID: record.ID, Type: record.Type, ConversationID: conversationID, Payload: record.Payload, CreatedAt: record.CreatedAt}
}

func toReport(record ReportRecord) Report {
	return Report{
		ID:             record.ID,
		ConversationID: record.ConversationID,
		MessageID:      record.MessageID,
		ReportedUserID: record.ReportedUserID.String(),
		Reason:         record.Reason,
		ContextBefore:  record.ContextBefore,
		ContextAfter:   record.ContextAfter,
		CreatedAt:      record.CreatedAt,
	}
}

func cloneShare(input *ShareSnapshot) *ShareSnapshot {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func ptrMessage(message Message) *Message {
	return &message
}

func trimConversationRecords(items []ConversationRecord, limit int) ([]ConversationRecord, bool) {
	if len(items) <= limit {
		return items, false
	}
	return items[:limit], true
}

func trimMessageRecords(items []MessageRecord, limit int) ([]MessageRecord, bool) {
	if len(items) <= limit {
		return items, false
	}
	return items[:limit], true
}

func firstReason(reason *string, fallback string) string {
	if reason != nil && strings.TrimSpace(*reason) != "" {
		return *reason
	}
	return fallback
}

func randomTicket() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate realtime ticket: %w", err)
	}
	return hex.EncodeToString(bytes[:]), nil
}

func isBlankUserID(id userdomain.UserID) bool {
	return id.String() == ""
}

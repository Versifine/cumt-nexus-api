package messageusecase

import (
	"context"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestStartConversationMutualCreatesAcceptedConversation(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	viewer := mustUser(t, "alice")
	target := mustUser(t, "bob")
	repo := newFakeRepo()
	users := newFakeUsers(viewer, target)
	users.follow(viewer.ID(), target.ID())
	users.follow(target.ID(), viewer.ID())
	uc := NewUseCase(repo, users, func() time.Time { return now })

	result, err := uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       viewer.ID(),
		TargetUsername: target.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "hello"},
	})
	if err != nil {
		t.Fatalf("StartConversation returned error: %v", err)
	}
	if result.Conversation.RequestStatus != "" {
		t.Fatalf("expected no request status, got %q", result.Conversation.RequestStatus)
	}
	if !result.Conversation.CanSend {
		t.Fatalf("expected accepted mutual conversation to be sendable")
	}
	if result.Message == nil || result.Message.Body != "hello" {
		t.Fatalf("expected initial message")
	}
}

func TestStartConversationNonMutualCreatesSinglePendingRequest(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	viewer := mustUser(t, "alice")
	target := mustUser(t, "bob")
	repo := newFakeRepo()
	users := newFakeUsers(viewer, target)
	uc := NewUseCase(repo, users, func() time.Time { return now })

	result, err := uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       viewer.ID(),
		TargetUsername: target.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "request"},
	})
	if err != nil {
		t.Fatalf("StartConversation returned error: %v", err)
	}
	if result.Conversation.RequestID == nil || result.Conversation.RequestStatus != "pending" {
		t.Fatalf("expected pending request conversation, got %#v", result.Conversation)
	}
	if result.Conversation.RequestDirection != "outgoing" || result.Conversation.ViewerCanAcceptRequest || result.Conversation.ConversationState != "outgoing_request" {
		t.Fatalf("expected outgoing request state for sender, got %#v", result.Conversation)
	}
	incoming := uc.toConversation(repo.conversations[result.Conversation.ID], target.ID())
	if incoming.RequestDirection != "incoming" || !incoming.ViewerCanAcceptRequest || !incoming.ViewerCanRejectRequest || incoming.ConversationState != "incoming_request" {
		t.Fatalf("expected incoming request state for recipient, got %#v", incoming)
	}

	_, err = uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       viewer.ID(),
		TargetUsername: target.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "again"},
	})
	if !apperr.IsCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for second pending request message, got %v", err)
	}
}

func TestRejectedRequestCanOnlyBeReopenedByOriginalRecipient(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	viewer := mustUser(t, "alice")
	target := mustUser(t, "bob")
	repo := newFakeRepo()
	users := newFakeUsers(viewer, target)
	uc := NewUseCase(repo, users, func() time.Time { return now })
	created, err := uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       viewer.ID(),
		TargetUsername: target.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "initial"},
	})
	if err != nil {
		t.Fatalf("StartConversation returned error: %v", err)
	}
	rejected, err := uc.RejectRequest(context.Background(), RequestActionInput{
		ViewerID:  target.ID(),
		RequestID: *created.Conversation.RequestID,
	})
	if err != nil {
		t.Fatalf("RejectRequest returned error: %v", err)
	}
	if !rejected.Conversation.ViewerCanReopen {
		t.Fatalf("expected original recipient to be able to reopen, got %#v", rejected.Conversation)
	}
	originalSender := uc.toConversation(repo.conversations[created.Conversation.ID], viewer.ID())
	if originalSender.ViewerCanReopen {
		t.Fatalf("original requester must not be able to reopen: %#v", originalSender)
	}

	_, err = uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       viewer.ID(),
		TargetUsername: target.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "bypass"},
	})
	if !apperr.IsCode(err, apperr.CodeMessageRequestRejected) {
		t.Fatalf("expected message_request_rejected for original requester, got %v", err)
	}
	_, err = uc.SendMessage(context.Background(), SendMessageInput{
		ViewerID:       viewer.ID(),
		ConversationID: created.Conversation.ID,
		Message:        MessageDraft{Type: MessageTypeText, Body: "still bypass"},
	})
	if !apperr.IsCode(err, apperr.CodeMessageRequestRejected) {
		t.Fatalf("expected message_request_rejected for direct send by original requester, got %v", err)
	}

	reopened, err := uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       target.ID(),
		TargetUsername: viewer.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "receiver starts after ignore"},
	})
	if err != nil {
		t.Fatalf("recipient reopen StartConversation returned error: %v", err)
	}
	if reopened.Conversation.ID != created.Conversation.ID || reopened.Conversation.RequestStatus != ConversationStatusAccepted || reopened.Conversation.ConversationState != "normal" || !reopened.Conversation.CanSend {
		t.Fatalf("expected old conversation to become normal accepted conversation, got %#v", reopened.Conversation)
	}
	if reopened.Message == nil || reopened.Message.Body != "receiver starts after ignore" {
		t.Fatalf("expected reopened message, got %#v", reopened.Message)
	}
	if reopened.Conversation.ViewerCanReopen {
		t.Fatalf("accepted conversation should not remain reopenable: %#v", reopened.Conversation)
	}
}

func TestRejectedRequestRecipientCanReopenFromConversationMessageInput(t *testing.T) {
	now := time.Date(2026, 6, 16, 11, 0, 0, 0, time.UTC)
	viewer := mustUser(t, "alice")
	target := mustUser(t, "bob")
	repo := newFakeRepo()
	users := newFakeUsers(viewer, target)
	uc := NewUseCase(repo, users, func() time.Time { return now })
	created, err := uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       viewer.ID(),
		TargetUsername: target.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "initial"},
	})
	if err != nil {
		t.Fatalf("StartConversation returned error: %v", err)
	}
	if _, err := uc.RejectRequest(context.Background(), RequestActionInput{ViewerID: target.ID(), RequestID: *created.Conversation.RequestID}); err != nil {
		t.Fatalf("RejectRequest returned error: %v", err)
	}

	reopened, err := uc.SendMessage(context.Background(), SendMessageInput{
		ViewerID:       target.ID(),
		ConversationID: created.Conversation.ID,
		Message:        MessageDraft{Type: MessageTypeText, Body: "inline reopen"},
	})
	if err != nil {
		t.Fatalf("recipient SendMessage reopen returned error: %v", err)
	}
	if reopened.Conversation.RequestStatus != ConversationStatusAccepted || reopened.Message == nil || reopened.Message.Body != "inline reopen" {
		t.Fatalf("expected send to reopen rejected request, got %#v", reopened)
	}
}

func TestRejectedRequestRecipientCanReopenAfterLocalDelete(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	viewer := mustUser(t, "alice")
	target := mustUser(t, "bob")
	repo := newFakeRepo()
	users := newFakeUsers(viewer, target)
	uc := NewUseCase(repo, users, func() time.Time { return now })
	created, err := uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       viewer.ID(),
		TargetUsername: target.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "initial"},
	})
	if err != nil {
		t.Fatalf("StartConversation returned error: %v", err)
	}
	if _, err := uc.RejectRequest(context.Background(), RequestActionInput{ViewerID: target.ID(), RequestID: *created.Conversation.RequestID}); err != nil {
		t.Fatalf("RejectRequest returned error: %v", err)
	}
	if err := uc.DeleteConversation(context.Background(), ConversationActionInput{ViewerID: target.ID(), ConversationID: created.Conversation.ID}); err != nil {
		t.Fatalf("DeleteConversation returned error: %v", err)
	}
	if _, err := uc.SendMessage(context.Background(), SendMessageInput{
		ViewerID:       target.ID(),
		ConversationID: created.Conversation.ID,
		Message:        MessageDraft{Type: MessageTypeText, Body: "hidden direct send"},
	}); !apperr.IsCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected hidden old conversation to be inaccessible by direct send, got %v", err)
	}

	reopened, err := uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       target.ID(),
		TargetUsername: viewer.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "after local delete retry"},
	})
	if err != nil {
		t.Fatalf("recipient reopen after local delete returned error: %v", err)
	}
	if reopened.Conversation.ID != created.Conversation.ID {
		t.Fatalf("expected reopened old conversation %s, got %s", created.Conversation.ID, reopened.Conversation.ID)
	}
	if reopened.Conversation.RequestStatus != ConversationStatusAccepted || reopened.Conversation.ConversationState != "normal" || !reopened.Conversation.CanSend {
		t.Fatalf("expected restored normal conversation, got %#v", reopened.Conversation)
	}
	if reopened.Message == nil || reopened.Message.Body != "after local delete retry" {
		t.Fatalf("expected retry message to be inserted, got %#v", reopened.Message)
	}
	if repo.deletedConversations[conversationKey(target.ID(), created.Conversation.ID)] {
		t.Fatalf("expected reopen to clear local delete marker")
	}
}

func TestSendMessageBlockedForbidden(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	viewer := mustUser(t, "alice")
	target := mustUser(t, "bob")
	repo := newFakeRepo()
	users := newFakeUsers(viewer, target)
	users.follow(viewer.ID(), target.ID())
	users.follow(target.ID(), viewer.ID())
	uc := NewUseCase(repo, users, func() time.Time { return now })
	created, err := uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       viewer.ID(),
		TargetUsername: target.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "hello"},
	})
	if err != nil {
		t.Fatalf("StartConversation returned error: %v", err)
	}
	repo.blocked[key(viewer.ID(), target.ID())] = true

	_, err = uc.SendMessage(context.Background(), SendMessageInput{
		ViewerID:       viewer.ID(),
		ConversationID: created.Conversation.ID,
		Message:        MessageDraft{Type: MessageTypeText, Body: "blocked"},
	})
	if !apperr.IsCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden after block, got %v", err)
	}
}

func TestRecallMessageWithinWindowSucceedsAndRepeatConflicts(t *testing.T) {
	sentAt := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	now := sentAt
	viewer := mustUser(t, "alice")
	target := mustUser(t, "bob")
	repo := newFakeRepo()
	users := newFakeUsers(viewer, target)
	users.follow(viewer.ID(), target.ID())
	users.follow(target.ID(), viewer.ID())
	uc := NewUseCase(repo, users, func() time.Time { return now })
	created, err := uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       viewer.ID(),
		TargetUsername: target.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "recall me"},
	})
	if err != nil {
		t.Fatalf("StartConversation returned error: %v", err)
	}

	now = sentAt.Add(MessageRecallWindow)
	recalled, err := uc.RecallMessage(context.Background(), MessageActionInput{
		ViewerID:  viewer.ID(),
		MessageID: created.Message.ID,
	})
	if err != nil {
		t.Fatalf("RecallMessage returned error: %v", err)
	}
	if recalled.Message == nil || recalled.Message.Status != "recalled" || recalled.Message.RecalledAt == nil {
		t.Fatalf("expected recalled message, got %#v", recalled.Message)
	}
	eventCount := len(repo.events)

	_, err = uc.RecallMessage(context.Background(), MessageActionInput{
		ViewerID:  viewer.ID(),
		MessageID: created.Message.ID,
	})
	if !apperr.IsCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for repeated recall, got %v", err)
	}
	if len(repo.events) != eventCount {
		t.Fatalf("expected repeated recall not to emit event, before=%d after=%d", eventCount, len(repo.events))
	}
}

func TestRecallMessageExpiredDoesNotMutateMessage(t *testing.T) {
	sentAt := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	now := sentAt
	viewer := mustUser(t, "alice")
	target := mustUser(t, "bob")
	repo := newFakeRepo()
	users := newFakeUsers(viewer, target)
	users.follow(viewer.ID(), target.ID())
	users.follow(target.ID(), viewer.ID())
	uc := NewUseCase(repo, users, func() time.Time { return now })
	created, err := uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       viewer.ID(),
		TargetUsername: target.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "too late"},
	})
	if err != nil {
		t.Fatalf("StartConversation returned error: %v", err)
	}
	eventCount := len(repo.events)

	now = sentAt.Add(MessageRecallWindow + time.Nanosecond)
	_, err = uc.RecallMessage(context.Background(), MessageActionInput{
		ViewerID:  viewer.ID(),
		MessageID: created.Message.ID,
	})
	if !apperr.IsCode(err, apperr.CodeMessageRecallExpired) {
		t.Fatalf("expected message_recall_expired, got %v", err)
	}
	message := repo.messages[created.Message.ID]
	if message.Status != "visible" || message.RecalledAt != nil {
		t.Fatalf("expected expired recall not to mutate message, got %#v", message)
	}
	if len(repo.events) != eventCount {
		t.Fatalf("expected expired recall not to emit event, before=%d after=%d", eventCount, len(repo.events))
	}
}

func TestRecallMessageRequiresSenderAndVisibleMessage(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	viewer := mustUser(t, "alice")
	target := mustUser(t, "bob")
	repo := newFakeRepo()
	users := newFakeUsers(viewer, target)
	users.follow(viewer.ID(), target.ID())
	users.follow(target.ID(), viewer.ID())
	uc := NewUseCase(repo, users, func() time.Time { return now })
	created, err := uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       viewer.ID(),
		TargetUsername: target.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "private"},
	})
	if err != nil {
		t.Fatalf("StartConversation returned error: %v", err)
	}

	_, err = uc.RecallMessage(context.Background(), MessageActionInput{
		ViewerID:  target.ID(),
		MessageID: created.Message.ID,
	})
	if !apperr.IsCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for non-sender recall, got %v", err)
	}
	if err := uc.DeleteMessage(context.Background(), MessageActionInput{ViewerID: viewer.ID(), MessageID: created.Message.ID}); err != nil {
		t.Fatalf("DeleteMessage returned error: %v", err)
	}
	eventCount := len(repo.events)

	_, err = uc.RecallMessage(context.Background(), MessageActionInput{
		ViewerID:  viewer.ID(),
		MessageID: created.Message.ID,
	})
	if !apperr.IsCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for viewer-deleted message recall, got %v", err)
	}
	if len(repo.events) != eventCount {
		t.Fatalf("expected viewer-deleted recall not to emit event, before=%d after=%d", eventCount, len(repo.events))
	}
}

func TestConversationManagementActionsAreViewerLocal(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	viewer := mustUser(t, "alice")
	target := mustUser(t, "bob")
	repo := newFakeRepo()
	users := newFakeUsers(viewer, target)
	users.follow(viewer.ID(), target.ID())
	users.follow(target.ID(), viewer.ID())
	uc := NewUseCase(repo, users, func() time.Time { return now })
	created, err := uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       viewer.ID(),
		TargetUsername: target.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "hello"},
	})
	if err != nil {
		t.Fatalf("StartConversation returned error: %v", err)
	}

	pinned, err := uc.SetConversationPinned(context.Background(), ConversationActionInput{ViewerID: viewer.ID(), ConversationID: created.Conversation.ID}, true)
	if err != nil {
		t.Fatalf("SetConversationPinned returned error: %v", err)
	}
	if !pinned.Conversation.Pinned {
		t.Fatalf("expected pinned conversation")
	}
	muted, err := uc.SetConversationMuted(context.Background(), ConversationActionInput{ViewerID: viewer.ID(), ConversationID: created.Conversation.ID}, true)
	if err != nil {
		t.Fatalf("SetConversationMuted returned error: %v", err)
	}
	if !muted.Conversation.Muted {
		t.Fatalf("expected muted conversation")
	}
	if err := uc.DeleteConversation(context.Background(), ConversationActionInput{ViewerID: viewer.ID(), ConversationID: created.Conversation.ID}); err != nil {
		t.Fatalf("DeleteConversation returned error: %v", err)
	}
	if !repo.deletedConversations[conversationKey(viewer.ID(), created.Conversation.ID)] {
		t.Fatalf("expected local conversation delete marker")
	}
}

func TestCreateRealtimeTicketConnectReturnsBacklog(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	viewer := mustUser(t, "alice")
	repo := newFakeRepo()
	users := newFakeUsers(viewer)
	uc := NewUseCase(repo, users, func() time.Time { return now })
	repo.events = append(repo.events, RealtimeEventRecord{ID: "11111111-1111-1111-1111-111111111111", UserID: viewer.ID(), Type: "conversation_updated", Payload: "{}", CreatedAt: now})

	ticket, err := uc.CreateRealtimeTicket(context.Background(), RealtimeTicketInput{ViewerID: viewer.ID()})
	if err != nil {
		t.Fatalf("CreateRealtimeTicket returned error: %v", err)
	}
	result, err := uc.ConnectRealtime(context.Background(), ticket.Ticket)
	if err != nil {
		t.Fatalf("ConnectRealtime returned error: %v", err)
	}
	if len(result.Events) != 1 || result.Events[0].ID != repo.events[0].ID {
		t.Fatalf("expected one replayed event, got %#v", result.Events)
	}
}

func TestReportMessageDelegatesLimitedReportCreation(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	viewer := mustUser(t, "alice")
	target := mustUser(t, "bob")
	repo := newFakeRepo()
	users := newFakeUsers(viewer, target)
	users.follow(viewer.ID(), target.ID())
	users.follow(target.ID(), viewer.ID())
	uc := NewUseCase(repo, users, func() time.Time { return now })
	created, err := uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       viewer.ID(),
		TargetUsername: target.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "spam"},
	})
	if err != nil {
		t.Fatalf("StartConversation returned error: %v", err)
	}

	report, err := uc.ReportMessage(context.Background(), MessageActionInput{ViewerID: target.ID(), MessageID: created.Message.ID, Reason: "spam"})
	if err != nil {
		t.Fatalf("ReportMessage returned error: %v", err)
	}
	if report.Report.MessageID != created.Message.ID || report.Report.ReportedUserID != viewer.ID().String() {
		t.Fatalf("unexpected report: %#v", report.Report)
	}
}

func TestReportConversationUsesPeerMessageAsLimitedContextAnchor(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	viewer := mustUser(t, "alice")
	target := mustUser(t, "bob")
	repo := newFakeRepo()
	users := newFakeUsers(viewer, target)
	users.follow(viewer.ID(), target.ID())
	users.follow(target.ID(), viewer.ID())
	uc := NewUseCase(repo, users, func() time.Time { return now })
	created, err := uc.StartConversation(context.Background(), StartConversationInput{
		ViewerID:       viewer.ID(),
		TargetUsername: target.Username().String(),
		Message:        MessageDraft{Type: MessageTypeText, Body: "hello"},
	})
	if err != nil {
		t.Fatalf("StartConversation returned error: %v", err)
	}
	peerMessage, err := uc.SendMessage(context.Background(), SendMessageInput{
		ViewerID:       target.ID(),
		ConversationID: created.Conversation.ID,
		Message:        MessageDraft{Type: MessageTypeText, Body: "abuse"},
	})
	if err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}

	report, err := uc.ReportConversation(context.Background(), ConversationActionInput{ViewerID: viewer.ID(), ConversationID: created.Conversation.ID, Reason: "abuse"})
	if err != nil {
		t.Fatalf("ReportConversation returned error: %v", err)
	}
	if report.Report.MessageID != peerMessage.Message.ID || report.Report.ReportedUserID != target.ID().String() {
		t.Fatalf("unexpected conversation report: %#v", report.Report)
	}
}

type fakeRepo struct {
	privacy              map[userdomain.UserID]PrivacySettingsRecord
	blocked              map[string]bool
	conversations        map[string]ConversationRecord
	deletedConversations map[string]bool
	pairIndex            map[string]string
	messages             map[string]MessageRecord
	reports              map[string]ReportRecord
	events               []RealtimeEventRecord
	tickets              map[string]RealtimeTicketRecord
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		privacy:              map[userdomain.UserID]PrivacySettingsRecord{},
		blocked:              map[string]bool{},
		conversations:        map[string]ConversationRecord{},
		deletedConversations: map[string]bool{},
		pairIndex:            map[string]string{},
		messages:             map[string]MessageRecord{},
		reports:              map[string]ReportRecord{},
		tickets:              map[string]RealtimeTicketRecord{},
	}
}

func (f *fakeRepo) GetPrivacySettings(ctx context.Context, userID userdomain.UserID) (PrivacySettingsRecord, error) {
	if record, ok := f.privacy[userID]; ok {
		return record, nil
	}
	return PrivacySettingsRecord{AllowMessages: PrivacyEveryone}, nil
}

func (f *fakeRepo) UpsertPrivacySettings(ctx context.Context, userID userdomain.UserID, allowMessages string, onlineStatusEnabled bool, now time.Time) (PrivacySettingsRecord, error) {
	record := PrivacySettingsRecord{AllowMessages: allowMessages, OnlineStatusEnabled: onlineStatusEnabled, UpdatedAt: now}
	f.privacy[userID] = record
	return record, nil
}

func (f *fakeRepo) IsBlockedEither(ctx context.Context, a userdomain.UserID, b userdomain.UserID) (bool, error) {
	return f.blocked[key(a, b)] || f.blocked[key(b, a)], nil
}

func (f *fakeRepo) BlockUser(ctx context.Context, blockerID userdomain.UserID, blockedID userdomain.UserID, now time.Time) error {
	f.blocked[key(blockerID, blockedID)] = true
	return nil
}

func (f *fakeRepo) UnblockUser(ctx context.Context, blockerID userdomain.UserID, blockedID userdomain.UserID) error {
	delete(f.blocked, key(blockerID, blockedID))
	return nil
}

func (f *fakeRepo) FindDirectConversation(ctx context.Context, userID userdomain.UserID, peerID userdomain.UserID) (ConversationRecord, error) {
	return f.findDirectConversation(ctx, userID, peerID, false)
}

func (f *fakeRepo) FindDirectConversationIncludingDeleted(ctx context.Context, userID userdomain.UserID, peerID userdomain.UserID) (ConversationRecord, error) {
	return f.findDirectConversation(ctx, userID, peerID, true)
}

func (f *fakeRepo) findDirectConversation(ctx context.Context, userID userdomain.UserID, peerID userdomain.UserID, includeDeleted bool) (ConversationRecord, error) {
	id, ok := f.pairIndex[pairKey(userID, peerID)]
	if !ok {
		return ConversationRecord{}, apperr.New(apperr.CodeNotFound, "message conversation not found")
	}
	if !includeDeleted && f.deletedConversations[conversationKey(userID, id)] {
		return ConversationRecord{}, apperr.New(apperr.CodeNotFound, "message conversation not found")
	}
	record := f.conversations[id]
	record.Peer = fakeUserByID(peerID)
	return record, nil
}

func (f *fakeRepo) CountRecentRequests(ctx context.Context, fromUserID userdomain.UserID, since time.Time) (int, error) {
	count := 0
	for _, conversation := range f.conversations {
		if conversation.CreatedBy == fromUserID && conversation.Status == ConversationStatusPending {
			count++
		}
	}
	return count, nil
}

func (f *fakeRepo) CreateConversationWithMessage(ctx context.Context, input CreateConversationRecord) (ConversationRecord, MessageRecord, error) {
	peer := fakeUserByID(input.PeerID)
	requestID := (*string)(nil)
	requestStatus := ""
	if input.RequestID != "" {
		requestID = &input.RequestID
		requestStatus = "pending"
	}
	message := f.makeMessage(input.Message, fakeUserByID(input.ViewerID))
	message.ConversationID = input.ID
	conversation := ConversationRecord{
		ID:            input.ID,
		Status:        input.Status,
		CreatedBy:     input.ViewerID,
		RequestID:     requestID,
		RequestStatus: requestStatus,
		Peer:          peer,
		LastMessage:   &message,
		UpdatedAt:     input.Now,
	}
	f.conversations[input.ID] = conversation
	f.pairIndex[pairKey(input.ViewerID, input.PeerID)] = input.ID
	f.messages[message.ID] = message
	return conversation, message, nil
}

func (f *fakeRepo) InsertMessage(ctx context.Context, input CreateMessageRecord) (MessageRecord, error) {
	message := f.makeMessage(input, fakeUserByID(input.SenderID))
	f.messages[message.ID] = message
	conversation := f.conversations[input.ConversationID]
	conversation.LastMessage = &message
	f.conversations[input.ConversationID] = conversation
	return message, nil
}

func (f *fakeRepo) ReopenRejectedRequestWithMessage(ctx context.Context, input ReopenRejectedRequestRecord) (ConversationRecord, MessageRecord, error) {
	conversation := f.conversations[input.ConversationID]
	if conversation.RequestID == nil || conversation.Status != ConversationStatusRejected || conversation.RequestStatus != ConversationStatusRejected || conversation.CreatedBy != input.PeerID {
		return ConversationRecord{}, MessageRecord{}, messageRequestRejectedError()
	}
	conversation.Status = ConversationStatusAccepted
	conversation.RequestStatus = ConversationStatusAccepted
	message := f.makeMessage(input.Message, fakeUserByID(input.ViewerID))
	message.ConversationID = input.ConversationID
	conversation.LastMessage = &message
	conversation.UpdatedAt = input.Now
	f.messages[message.ID] = message
	f.conversations[input.ConversationID] = conversation
	delete(f.deletedConversations, conversationKey(input.ViewerID, input.ConversationID))
	updated, err := f.GetConversation(ctx, input.ConversationID, input.ViewerID)
	return updated, message, err
}

func (f *fakeRepo) ListConversations(ctx context.Context, userID userdomain.UserID, box string, limit int, offset int) ([]ConversationRecord, error) {
	result := make([]ConversationRecord, 0, len(f.conversations))
	for _, conversation := range f.conversations {
		if f.deletedConversations[conversationKey(userID, conversation.ID)] {
			continue
		}
		conversation.Peer = f.peerForViewer(conversation, userID)
		result = append(result, conversation)
	}
	return result, nil
}

func (f *fakeRepo) GetConversation(ctx context.Context, conversationID string, userID userdomain.UserID) (ConversationRecord, error) {
	record, ok := f.conversations[conversationID]
	if !ok {
		return ConversationRecord{}, apperr.New(apperr.CodeNotFound, "message conversation not found")
	}
	if f.deletedConversations[conversationKey(userID, conversationID)] {
		return ConversationRecord{}, apperr.New(apperr.CodeNotFound, "message conversation not found")
	}
	record.Peer = f.peerForViewer(record, userID)
	if f.blocked[key(userID, record.Peer.ID())] || f.blocked[key(record.Peer.ID(), userID)] {
		record.Blocked = true
	}
	return record, nil
}

func (f *fakeRepo) ListMessages(ctx context.Context, conversationID string, userID userdomain.UserID, beforeMessageID string, limit int) ([]MessageRecord, error) {
	result := []MessageRecord{}
	for _, message := range f.messages {
		if message.ConversationID == conversationID {
			result = append(result, message)
		}
	}
	return result, nil
}

func (f *fakeRepo) MarkConversationRead(ctx context.Context, conversationID string, userID userdomain.UserID, now time.Time) error {
	return nil
}

func (f *fakeRepo) SetConversationArchived(ctx context.Context, conversationID string, userID userdomain.UserID, archived bool, now time.Time) error {
	conversation := f.conversations[conversationID]
	conversation.Archived = archived
	f.conversations[conversationID] = conversation
	return nil
}

func (f *fakeRepo) SetConversationPinned(ctx context.Context, conversationID string, userID userdomain.UserID, pinned bool, now time.Time) error {
	conversation := f.conversations[conversationID]
	conversation.Pinned = pinned
	f.conversations[conversationID] = conversation
	return nil
}

func (f *fakeRepo) SetConversationMuted(ctx context.Context, conversationID string, userID userdomain.UserID, muted bool, now time.Time) error {
	conversation := f.conversations[conversationID]
	conversation.Muted = muted
	f.conversations[conversationID] = conversation
	return nil
}

func (f *fakeRepo) HideConversationForUser(ctx context.Context, conversationID string, userID userdomain.UserID, now time.Time) error {
	if _, ok := f.conversations[conversationID]; !ok {
		return apperr.New(apperr.CodeNotFound, "message conversation not found")
	}
	f.deletedConversations[conversationKey(userID, conversationID)] = true
	return nil
}

func (f *fakeRepo) AcceptMessageRequest(ctx context.Context, requestID string, userID userdomain.UserID, now time.Time) (ConversationRecord, error) {
	for id, conversation := range f.conversations {
		if conversation.RequestID != nil && *conversation.RequestID == requestID {
			conversation.Status = ConversationStatusAccepted
			conversation.RequestStatus = "accepted"
			f.conversations[id] = conversation
			return f.GetConversation(ctx, id, userID)
		}
	}
	return ConversationRecord{}, apperr.New(apperr.CodeNotFound, "message request not found")
}

func (f *fakeRepo) RejectMessageRequest(ctx context.Context, requestID string, userID userdomain.UserID, now time.Time) (ConversationRecord, error) {
	for id, conversation := range f.conversations {
		if conversation.RequestID != nil && *conversation.RequestID == requestID {
			conversation.Status = ConversationStatusRejected
			conversation.RequestStatus = "rejected"
			f.conversations[id] = conversation
			return f.GetConversation(ctx, id, userID)
		}
	}
	return ConversationRecord{}, apperr.New(apperr.CodeNotFound, "message request not found")
}

func (f *fakeRepo) GetMessage(ctx context.Context, messageID string, userID userdomain.UserID) (MessageRecord, error) {
	message, ok := f.messages[messageID]
	if !ok {
		return MessageRecord{}, apperr.New(apperr.CodeNotFound, "message not found")
	}
	return message, nil
}

func (f *fakeRepo) UpdateMessageStatus(ctx context.Context, messageID string, status string, now time.Time) (MessageRecord, error) {
	message := f.messages[messageID]
	message.Status = status
	message.UpdatedAt = now
	if status == "recalled" {
		message.RecalledAt = &now
	}
	f.messages[messageID] = message
	return message, nil
}

func (f *fakeRepo) HideMessageForUser(ctx context.Context, messageID string, userID userdomain.UserID, now time.Time) error {
	message := f.messages[messageID]
	message.ViewerDeleted = true
	f.messages[messageID] = message
	return nil
}

func (f *fakeRepo) CreateMessageReport(ctx context.Context, input CreateReportRecord) (ReportRecord, error) {
	message := f.messages[input.MessageID]
	report := ReportRecord{ID: input.ID, ConversationID: message.ConversationID, MessageID: message.ID, ReportedUserID: message.Sender.ID(), Reason: input.Reason, ContextBefore: "before", ContextAfter: "after", CreatedAt: input.Now}
	f.reports[report.ID] = report
	return report, nil
}

func (f *fakeRepo) CreateRealtimeEvent(ctx context.Context, input RealtimeEventRecord) error {
	f.events = append(f.events, input)
	return nil
}

func (f *fakeRepo) CreateRealtimeTicket(ctx context.Context, ticket string, userID userdomain.UserID, lastEventID string, expiresAt time.Time, now time.Time) (RealtimeTicketRecord, error) {
	record := RealtimeTicketRecord{Ticket: ticket, UserID: userID, LastEventID: lastEventID, ExpiresAt: expiresAt}
	f.tickets[ticket] = record
	return record, nil
}

func (f *fakeRepo) ConsumeRealtimeTicket(ctx context.Context, ticket string, now time.Time) (RealtimeTicketRecord, error) {
	record, ok := f.tickets[ticket]
	if !ok {
		return RealtimeTicketRecord{}, apperr.New(apperr.CodeUnauthenticated, "realtime ticket is invalid")
	}
	return record, nil
}

func (f *fakeRepo) ListRealtimeEventsAfter(ctx context.Context, userID userdomain.UserID, afterEventID string, limit int) ([]RealtimeEventRecord, error) {
	result := []RealtimeEventRecord{}
	for _, event := range f.events {
		if event.UserID == userID {
			result = append(result, event)
		}
	}
	return result, nil
}

func (f *fakeRepo) makeMessage(input CreateMessageRecord, sender userdomain.User) MessageRecord {
	return MessageRecord{ID: input.ID, ConversationID: input.ConversationID, Sender: sender, Type: input.Type, Body: input.Body, ImageURL: input.ImageURL, Share: input.Share, Status: "visible", CreatedAt: input.Now, UpdatedAt: input.Now}
}

func (f *fakeRepo) peerForViewer(record ConversationRecord, viewerID userdomain.UserID) userdomain.User {
	if record.CreatedBy != viewerID && record.Peer.ID() == viewerID {
		return fakeUserByID(record.CreatedBy)
	}
	return record.Peer
}

type fakeUsers struct {
	users   map[userdomain.UserID]userdomain.User
	byName  map[string]userdomain.UserID
	follows map[string]bool
}

func newFakeUsers(users ...userdomain.User) *fakeUsers {
	fake := &fakeUsers{users: map[userdomain.UserID]userdomain.User{}, byName: map[string]userdomain.UserID{}, follows: map[string]bool{}}
	for _, user := range users {
		fake.users[user.ID()] = user
		fake.byName[user.Username().String()] = user.ID()
	}
	return fake
}

func (f *fakeUsers) FindByID(ctx context.Context, id userdomain.UserID) (*userdomain.User, error) {
	user, ok := f.users[id]
	if !ok {
		return nil, apperr.New(apperr.CodeNotFound, "user not found")
	}
	return &user, nil
}

func (f *fakeUsers) FindByUsername(ctx context.Context, username userdomain.Username) (*userdomain.User, error) {
	id, ok := f.byName[username.String()]
	if !ok {
		return nil, apperr.New(apperr.CodeNotFound, "user not found")
	}
	return f.FindByID(ctx, id)
}

func (f *fakeUsers) IsFollowing(ctx context.Context, followerID userdomain.UserID, followingID userdomain.UserID) (bool, error) {
	return f.follows[key(followerID, followingID)], nil
}

func (f *fakeUsers) follow(followerID userdomain.UserID, followingID userdomain.UserID) {
	f.follows[key(followerID, followingID)] = true
}

var fakeUserStore = map[userdomain.UserID]userdomain.User{}

func mustUser(t *testing.T, username string) userdomain.User {
	t.Helper()
	id, err := userdomain.NewUserID(testUUID(username))
	if err != nil {
		t.Fatalf("NewUserID returned error: %v", err)
	}
	name, err := userdomain.NewUsername(username)
	if err != nil {
		t.Fatalf("NewUsername returned error: %v", err)
	}
	hash, err := userdomain.NewPasswordHash("hash-" + username)
	if err != nil {
		t.Fatalf("NewPasswordHash returned error: %v", err)
	}
	user, err := userdomain.NewUser(id, name, hash, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewUser returned error: %v", err)
	}
	fakeUserStore[id] = *user
	return *user
}

func fakeUserByID(id userdomain.UserID) userdomain.User {
	return fakeUserStore[id]
}

func key(a userdomain.UserID, b userdomain.UserID) string {
	return a.String() + ":" + b.String()
}

func conversationKey(userID userdomain.UserID, conversationID string) string {
	return userID.String() + ":" + conversationID
}

func pairKey(a userdomain.UserID, b userdomain.UserID) string {
	if a.String() < b.String() {
		return a.String() + ":" + b.String()
	}
	return b.String() + ":" + a.String()
}

func testUUID(seed string) string {
	switch seed {
	case "alice":
		return "11111111-1111-1111-1111-111111111111"
	case "bob":
		return "22222222-2222-2222-2222-222222222222"
	default:
		return "33333333-3333-3333-3333-333333333333"
	}
}

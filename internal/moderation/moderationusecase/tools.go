package moderationusecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/textlimit"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

const (
	DefaultModToolsListLimit = 20
	MaxModToolsListLimit     = 50
	MaxToolTitleRunes        = 80
	MaxRemovalReasonBody     = 1000
	MaxSavedResponseBody     = 2000
	MaxModNoteBody           = 1000
	MaxModFlairRunes         = 64
	MaxAutomodConfigRunes    = 20000
	MaxContentControlItems   = 100
	MaxModmailSubjectRunes   = 160
	MaxModmailBodyRunes      = 4000
	UserStateBanned          = "banned"
	UserStateMuted           = "muted"
	UserStateApproved        = "approved"
)

type ToolsUseCase struct {
	tools       ToolsRepository
	staff       PlatformStaffRepository
	owners      PlatformOwnerRepository
	communities CommunityRepository
	roles       CommunityRoleRepository
	now         func() time.Time
}

func NewToolsUseCase(
	tools ToolsRepository,
	staff PlatformStaffRepository,
	communities CommunityRepository,
	roles CommunityRoleRepository,
	now func() time.Time,
) *ToolsUseCase {
	if now == nil {
		now = time.Now
	}
	return &ToolsUseCase{
		tools:       tools,
		staff:       staff,
		owners:      platformOwnerRepository(staff),
		communities: communities,
		roles:       roles,
		now:         now,
	}
}

type ToolsRepository interface {
	ListModQueue(ctx context.Context, input ListModQueueRecordInput) ([]ModQueueItem, error)
	GetModQueueItem(ctx context.Context, input GetModQueueItemRecordInput) (ModQueueItemDetail, error)
	GetModQueueSummary(ctx context.Context, input GetModQueueSummaryRecordInput) (ModQueueSummary, error)
	ApplyModerationAction(ctx context.Context, input ApplyModerationActionRecordInput) (ModerationAction, error)
	IgnoreCommunityReport(ctx context.Context, communityID communitydomain.CommunityID, reportID moderationdomain.ContentReportID, actorID userdomain.UserID, reviewedAt time.Time) (moderationdomain.ContentReport, error)
	ListCommunityModLogs(ctx context.Context, input ListCommunityModLogsRecordInput) ([]CommunityModLog, error)
	ListRemovalReasons(ctx context.Context, communityID communitydomain.CommunityID) ([]ModerationTemplate, error)
	CreateRemovalReason(ctx context.Context, input WriteModerationTemplateRecordInput) (ModerationTemplate, error)
	UpdateRemovalReason(ctx context.Context, input WriteModerationTemplateRecordInput) (ModerationTemplate, error)
	DeleteRemovalReason(ctx context.Context, communityID communitydomain.CommunityID, id string, actorID userdomain.UserID, deletedAt time.Time) error
	ListSavedResponses(ctx context.Context, communityID communitydomain.CommunityID) ([]ModerationTemplate, error)
	CreateSavedResponse(ctx context.Context, input WriteModerationTemplateRecordInput) (ModerationTemplate, error)
	UpdateSavedResponse(ctx context.Context, input WriteModerationTemplateRecordInput) (ModerationTemplate, error)
	DeleteSavedResponse(ctx context.Context, communityID communitydomain.CommunityID, id string, actorID userdomain.UserID, deletedAt time.Time) error
	ListUserStates(ctx context.Context, communityID communitydomain.CommunityID, kind string, limit int, offset int) ([]CommunityUserState, error)
	UpsertUserState(ctx context.Context, input UpsertUserStateRecordInput) (CommunityUserState, error)
	DeleteUserState(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID, kind string, actorID userdomain.UserID, deletedAt time.Time) error
	GetUserProfile(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID) (ModerationUserProfile, error)
	ListModeratorNotes(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID, limit int, offset int) ([]ModeratorNote, error)
	CreateModeratorNote(ctx context.Context, input CreateModeratorNoteRecordInput) (ModeratorNote, error)
	DeleteModeratorNote(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID, noteID string, actorID userdomain.UserID, deletedAt time.Time) error
	GetAutomodConfig(ctx context.Context, communityID communitydomain.CommunityID) (AutomodConfig, error)
	UpsertAutomodConfig(ctx context.Context, input UpsertAutomodConfigRecordInput) (AutomodConfig, error)
	ListAutomodVersions(ctx context.Context, communityID communitydomain.CommunityID, limit int, offset int) ([]AutomodVersion, error)
	GetContentControls(ctx context.Context, communityID communitydomain.CommunityID) (ContentControls, error)
	UpsertContentControls(ctx context.Context, input UpsertContentControlsRecordInput) (ContentControls, error)
	ListModmailConversations(ctx context.Context, input ListModmailConversationsRecordInput) ([]ModmailConversation, error)
	CreateModmailConversation(ctx context.Context, input CreateModmailConversationRecordInput) (ModmailConversationDetail, error)
	GetModmailConversation(ctx context.Context, input GetModmailConversationRecordInput) (ModmailConversationDetail, error)
	AddModmailMessage(ctx context.Context, input AddModmailMessageRecordInput) (ModmailConversationDetail, error)
	UpdateModmailConversation(ctx context.Context, input UpdateModmailConversationRecordInput) (ModmailConversation, error)
	GetCommunityInsightsSummary(ctx context.Context, input CommunityInsightsRecordInput) (CommunityInsightsSummary, error)
	GetCommunityModerationInsights(ctx context.Context, input CommunityInsightsRecordInput) (CommunityModerationInsights, error)
}

type PlatformOwnerRepository interface {
	IsPlatformOwner(ctx context.Context, userID userdomain.UserID) (bool, error)
}

type ListModQueueRecordInput struct {
	CommunityID *communitydomain.CommunityID
	Queue       string
	Limit       int
	Offset      int
}

type GetModQueueItemRecordInput struct {
	CommunityID *communitydomain.CommunityID
	TargetType  moderationdomain.TargetType
	TargetID    string
}

type GetModQueueSummaryRecordInput struct {
	CommunityID        *communitydomain.CommunityID
	PriorityItemLimit  int
	PriorityItemOffset int
}

type ApplyModerationActionRecordInput struct {
	ScopeCommunityID *communitydomain.CommunityID
	BatchID          string
	ActorID          userdomain.UserID
	TargetType       moderationdomain.TargetType
	TargetID         string
	Action           moderationdomain.ActionType
	Reason           string
	RemovalReasonID  string
	NotifyAuthor     bool
	BoolValue        *bool
	FlairText        string
	CreatedAt        time.Time
}

type ListCommunityModLogsRecordInput struct {
	CommunityID communitydomain.CommunityID
	Action      string
	ActorID     string
	TargetType  string
	TargetID    string
	Limit       int
	Offset      int
}

type WriteModerationTemplateRecordInput struct {
	ID          string
	CommunityID communitydomain.CommunityID
	ActorID     userdomain.UserID
	Title       string
	Body        string
	RuleID      string
	Position    int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type UpsertUserStateRecordInput struct {
	ID          string
	CommunityID communitydomain.CommunityID
	UserID      userdomain.UserID
	Kind        string
	Reason      string
	ExpiresAt   *time.Time
	ActorID     userdomain.UserID
	UpdatedAt   time.Time
}

type CreateModeratorNoteRecordInput struct {
	ID          string
	CommunityID communitydomain.CommunityID
	UserID      userdomain.UserID
	AuthorID    userdomain.UserID
	Body        string
	CreatedAt   time.Time
}

type UpsertAutomodConfigRecordInput struct {
	CommunityID communitydomain.CommunityID
	ActorID     userdomain.UserID
	ConfigText  string
	Rules       json.RawMessage
	UpdatedAt   time.Time
}

type UpsertContentControlsRecordInput struct {
	CommunityID             communitydomain.CommunityID
	ActorID                 userdomain.UserID
	BlockedKeywords         []string
	BlockedDomains          []string
	MinAccountAgeDays       int
	PostRateLimitPerHour    int
	CommentRateLimitPerHour int
	BlockNewAccounts        bool
	FilterLinks             bool
	UpdatedAt               time.Time
}

type ListModmailConversationsRecordInput struct {
	CommunityID communitydomain.CommunityID
	ActorID     userdomain.UserID
	Folder      string
	Limit       int
	Offset      int
}

type CreateModmailConversationRecordInput struct {
	ID          string
	MessageID   string
	CommunityID communitydomain.CommunityID
	ActorID     userdomain.UserID
	UserID      userdomain.UserID
	Subject     string
	Body        string
	CreatedAt   time.Time
}

type GetModmailConversationRecordInput struct {
	CommunityID    communitydomain.CommunityID
	ActorID        userdomain.UserID
	ConversationID string
}

type AddModmailMessageRecordInput struct {
	ID             string
	CommunityID    communitydomain.CommunityID
	ActorID        userdomain.UserID
	ConversationID string
	Body           string
	IsInternal     bool
	CreatedAt      time.Time
}

type UpdateModmailConversationRecordInput struct {
	CommunityID    communitydomain.CommunityID
	ActorID        userdomain.UserID
	ConversationID string
	Folder         string
	Status         string
	AssignedTo     string
	MarkRead       bool
	UpdatedAt      time.Time
}

type CommunityInsightsRecordInput struct {
	CommunityID communitydomain.CommunityID
	Since       time.Time
	Range       string
}

type ListModQueueInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	Queue         string
	Limit         int
	Offset        int
}

type ListModQueueResult struct {
	Items      []ModQueueItem
	Queue      string
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type GetModQueueItemInput struct {
	ActorID userdomain.UserID
	ItemID  string
}

type GetModQueueItemResult struct {
	Detail ModQueueItemDetail
}

type GetModQueueSummaryInput struct {
	ActorID userdomain.UserID
}

type GetModQueueSummaryResult struct {
	Summary ModQueueSummary
}

type BulkActionInput struct {
	ActorID         userdomain.UserID
	CommunitySlug   string
	Action          string
	TargetType      string
	TargetIDs       []string
	Targets         []ModerationTargetInput
	Reason          string
	RemovalReasonID string
	NotifyAuthor    bool
	Confirm         bool
	Value           *bool
	FlairText       string
}

type ModerationTargetInput struct {
	TargetType string
	TargetID   string
}

type BulkActionResult struct {
	Results []BulkActionItemResult
}

type BulkActionItemResult struct {
	TargetType   string
	TargetID     string
	OK           bool
	Action       *ModerationAction
	ErrorCode    string
	ErrorMessage string
}

type ListCommunityModLogsInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	Action        string
	ActorFilterID string
	TargetType    string
	TargetID      string
	Limit         int
	Offset        int
}

type ListCommunityModLogsResult struct {
	Logs       []CommunityModLog
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type ModerationTemplateInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	ID            string
	Title         string
	Body          string
	RuleID        string
	Position      int
}

type ListModerationTemplatesResult struct {
	Templates []ModerationTemplate
}

type ModerationTemplateResult struct {
	Template ModerationTemplate
}

type DeleteModerationTemplateInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	ID            string
}

type IgnoreCommunityReportInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	ReportID      string
}

type IgnoreCommunityReportResult struct {
	Report ContentReport
}

type ListUserStatesInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	Kind          string
	Limit         int
	Offset        int
}

type ListUserStatesResult struct {
	Users      []CommunityUserState
	Kind       string
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type WriteUserStateInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	Kind          string
	UserID        string
	Reason        string
	ExpiresAt     *time.Time
}

type UserStateResult struct {
	User CommunityUserState
}

type DeleteUserStateInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	Kind          string
	UserID        string
}

type GetUserProfileInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	UserID        string
}

type GetUserProfileResult struct {
	Profile ModerationUserProfile
}

type ListModeratorNotesInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	UserID        string
	Limit         int
	Offset        int
}

type ListModeratorNotesResult struct {
	Notes      []ModeratorNote
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type CreateModeratorNoteInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	UserID        string
	Body          string
}

type ModeratorNoteResult struct {
	Note ModeratorNote
}

type DeleteModeratorNoteInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	UserID        string
	NoteID        string
}

type AutomodConfigInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	ConfigText    string
	Rules         json.RawMessage
}

type AutomodConfigResult struct {
	Config AutomodConfig
}

type AutomodVersionsInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	Limit         int
	Offset        int
}

type AutomodVersionsResult struct {
	Versions   []AutomodVersion
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type AutomodDryRunInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	TargetType    string
	Title         string
	Body          string
	AuthorID      string
	Links         []string
}

type AutomodDryRunResult struct {
	Matches         []AutomodDryRunMatch
	SuggestedAction string
	Reasons         []string
}

type ContentControlsInput struct {
	ActorID                 userdomain.UserID
	CommunitySlug           string
	BlockedKeywords         []string
	BlockedDomains          []string
	MinAccountAgeDays       int
	PostRateLimitPerHour    int
	CommentRateLimitPerHour int
	BlockNewAccounts        bool
	FilterLinks             bool
}

type ContentControlsResult struct {
	Controls ContentControls
}

type ListModmailConversationsInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	Folder        string
	Limit         int
	Offset        int
}

type ListModmailConversationsResult struct {
	Conversations []ModmailConversation
	Folder        string
	Limit         int
	Offset        int
	NextOffset    int
	HasMore       bool
}

type CreateModmailConversationInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	UserID        string
	Subject       string
	Body          string
}

type ModmailConversationResult struct {
	Detail ModmailConversationDetail
}

type GetModmailConversationInput struct {
	ActorID        userdomain.UserID
	CommunitySlug  string
	ConversationID string
}

type ModmailMessageInput struct {
	ActorID        userdomain.UserID
	CommunitySlug  string
	ConversationID string
	Body           string
	IsInternal     bool
}

type UpdateModmailConversationInput struct {
	ActorID        userdomain.UserID
	CommunitySlug  string
	ConversationID string
	Folder         string
	Status         string
	AssignedTo     string
	MarkRead       bool
}

type UpdateModmailConversationResult struct {
	Conversation ModmailConversation
}

type CommunityInsightsInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	Range         string
	Limit         int
	Offset        int
}

type CommunityInsightsSummaryResult struct {
	Summary CommunityInsightsSummary
}

type CommunityModerationInsightsResult struct {
	Moderation CommunityModerationInsights
}

type CommunityTrainingQueueResult struct {
	Items      []CommunityTrainingQueueItem
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type ModQueueItem struct {
	ID            string
	TargetType    string
	TargetID      string
	PostID        string
	CommunityID   string
	CommunitySlug string
	AuthorID      string
	ReportCount   int
	Queue         string
	Status        string
	Preview       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ModQueueItemDetail struct {
	Item          ModQueueItem
	TargetPreview ReportTargetPreview
	Reports       []ModQueueReport
	RecentActions []ModerationAction
}

type ModQueueReport struct {
	ID         string
	ReporterID string
	Reason     string
	Status     string
	CreatedAt  time.Time
}

type ModQueueCount struct {
	Queue string
	Count int
}

type ModQueueSummary struct {
	Queues        []ModQueueCount
	PriorityItems []ModQueueItem
}

type CommunityModLog struct {
	ID          string
	CommunityID string
	ActorID     string
	Action      string
	TargetType  string
	TargetID    string
	BatchID     string
	Before      map[string]any
	After       map[string]any
	Metadata    map[string]any
	CreatedAt   time.Time
}

type ModerationTemplate struct {
	ID          string
	CommunityID string
	Title       string
	Body        string
	RuleID      string
	IsActive    bool
	Position    int
	CreatedBy   string
	UpdatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CommunityUserState struct {
	ID          string
	CommunityID string
	UserID      string
	Username    string
	DisplayName string
	AvatarURL   string
	Kind        string
	Reason      string
	ExpiresAt   *time.Time
	CreatedBy   string
	UpdatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ModerationUserProfile struct {
	UserID       string
	Username     string
	DisplayName  string
	AvatarURL    string
	Headline     string
	Status       string
	PostCount    int
	CommentCount int
	ReportCount  int
	RemovedCount int
	IsBanned     bool
	IsMuted      bool
	IsApproved   bool
	RecentNotes  []ModeratorNote
}

type ModeratorNote struct {
	ID          string
	CommunityID string
	UserID      string
	AuthorID    string
	Body        string
	CreatedAt   time.Time
}

type AutomodConfig struct {
	CommunityID string
	ConfigText  string
	Rules       json.RawMessage
	Version     int
	UpdatedBy   string
	UpdatedAt   time.Time
}

type AutomodVersion struct {
	ID          string
	CommunityID string
	Version     int
	ConfigText  string
	Rules       json.RawMessage
	UpdatedBy   string
	CreatedAt   time.Time
}

type AutomodDryRunMatch struct {
	Rule   string
	Action string
	Reason string
}

type ContentControls struct {
	CommunityID             string
	BlockedKeywords         []string
	BlockedDomains          []string
	MinAccountAgeDays       int
	PostRateLimitPerHour    int
	CommentRateLimitPerHour int
	BlockNewAccounts        bool
	FilterLinks             bool
	UpdatedBy               string
	UpdatedAt               time.Time
}

type ModmailConversation struct {
	ID            string
	CommunityID   string
	Subject       string
	UserID        string
	Status        string
	Folder        string
	AssignedTo    string
	LastMessageAt time.Time
	UnreadCount   int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ModmailMessage struct {
	ID             string
	ConversationID string
	AuthorID       string
	Body           string
	IsInternal     bool
	CreatedAt      time.Time
}

type ModmailConversationDetail struct {
	Conversation ModmailConversation
	Messages     []ModmailMessage
}

type CommunityInsightsSummary struct {
	CommunityID   string
	Range         string
	Since         time.Time
	MembersTotal  int
	PostsCreated  int
	CommentsMade  int
	ActiveAuthors int
}

type CommunityModerationInsights struct {
	CommunityID     string
	Range           string
	Since           time.Time
	PendingReports  int
	ResolvedReports int
	RemovedPosts    int
	RemovedComments int
	SpamPosts       int
	SpamComments    int
	ActionsCount    int
}

type CommunityTrainingQueueItem struct {
	ID              string
	TargetType      string
	TargetID        string
	CommunityID     string
	Preview         string
	SuggestedAction string
	Reason          string
	CreatedAt       time.Time
}

func (uc *ToolsUseCase) ListModQueue(ctx context.Context, input ListModQueueInput) (ListModQueueResult, error) {
	communityID, err := uc.ensureScope(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ListModQueueResult{}, err
	}
	queue, err := normalizeModQueue(input.Queue)
	if err != nil {
		return ListModQueueResult{}, err
	}
	limit, offset, err := normalizeModToolsPagination(input.Limit, input.Offset)
	if err != nil {
		return ListModQueueResult{}, err
	}
	items, err := uc.tools.ListModQueue(ctx, ListModQueueRecordInput{
		CommunityID: communityID,
		Queue:       queue,
		Limit:       limit + 1,
		Offset:      offset,
	})
	if err != nil {
		return ListModQueueResult{}, fmt.Errorf("list moderation queue: %w", err)
	}
	items, hasMore := trimToolsPage(items, limit)
	return ListModQueueResult{Items: items, Queue: queue, Limit: limit, Offset: offset, NextOffset: offset + len(items), HasMore: hasMore}, nil
}

func (uc *ToolsUseCase) GetAdminModQueueItem(ctx context.Context, input GetModQueueItemInput) (GetModQueueItemResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return GetModQueueItemResult{}, err
	}
	targetType, targetID, err := parseModQueueItemID(input.ItemID)
	if err != nil {
		return GetModQueueItemResult{}, err
	}
	detail, err := uc.tools.GetModQueueItem(ctx, GetModQueueItemRecordInput{
		TargetType: targetType,
		TargetID:   targetID,
	})
	if err != nil {
		return GetModQueueItemResult{}, fmt.Errorf("get moderation queue item: %w", err)
	}
	return GetModQueueItemResult{Detail: detail}, nil
}

func (uc *ToolsUseCase) GetAdminModQueueSummary(ctx context.Context, input GetModQueueSummaryInput) (GetModQueueSummaryResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return GetModQueueSummaryResult{}, err
	}
	summary, err := uc.tools.GetModQueueSummary(ctx, GetModQueueSummaryRecordInput{
		PriorityItemLimit: DefaultModToolsListLimit,
	})
	if err != nil {
		return GetModQueueSummaryResult{}, fmt.Errorf("get moderation queue summary: %w", err)
	}
	return GetModQueueSummaryResult{Summary: summary}, nil
}

func (uc *ToolsUseCase) ApplyBulkAction(ctx context.Context, input BulkActionInput) (BulkActionResult, error) {
	communityID, err := uc.ensureScope(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return BulkActionResult{}, err
	}
	action, err := moderationdomain.NewActionType(input.Action)
	if err != nil {
		return BulkActionResult{}, err
	}
	targets, err := normalizeModerationTargets(input.TargetType, input.TargetIDs, input.Targets)
	if err != nil {
		return BulkActionResult{}, err
	}
	reason, err := normalizeActionReason(input.Reason, input.Confirm)
	if err != nil {
		return BulkActionResult{}, err
	}
	flair, err := textlimit.TrimmedOptionalMaxRunes(input.FlairText, "post flair", MaxModFlairRunes)
	if err != nil {
		return BulkActionResult{}, err
	}
	if action == moderationdomain.ActionTypeSetFlair && flair == "" && !input.Confirm {
		return BulkActionResult{}, apperr.New(apperr.CodeInvalidArgument, "post flair is required")
	}

	batchID := uuid.NewString()
	results := make([]BulkActionItemResult, 0, len(targets))
	for _, target := range targets {
		result := BulkActionItemResult{
			TargetType: target.TargetType.String(),
			TargetID:   target.TargetID,
		}
		applied, err := uc.tools.ApplyModerationAction(ctx, ApplyModerationActionRecordInput{
			ScopeCommunityID: communityID,
			BatchID:          batchID,
			ActorID:          input.ActorID,
			TargetType:       target.TargetType,
			TargetID:         target.TargetID,
			Action:           action,
			Reason:           reason,
			RemovalReasonID:  normalizeOptionalUUID(input.RemovalReasonID),
			NotifyAuthor:     input.NotifyAuthor,
			BoolValue:        input.Value,
			FlairText:        flair,
			CreatedAt:        uc.now().UTC(),
		})
		if err != nil {
			result.ErrorCode, result.ErrorMessage = errorResult(err)
			results = append(results, result)
			continue
		}
		result.OK = true
		result.Action = &applied
		results = append(results, result)
	}
	return BulkActionResult{Results: results}, nil
}

func (uc *ToolsUseCase) IgnoreCommunityReport(ctx context.Context, input IgnoreCommunityReportInput) (IgnoreCommunityReportResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return IgnoreCommunityReportResult{}, err
	}
	reportID, err := moderationdomain.NewContentReportID(input.ReportID)
	if err != nil {
		return IgnoreCommunityReportResult{}, err
	}
	report, err := uc.tools.IgnoreCommunityReport(ctx, communityID, reportID, input.ActorID, uc.now().UTC())
	if err != nil {
		return IgnoreCommunityReportResult{}, fmt.Errorf("ignore community report: %w", err)
	}
	return IgnoreCommunityReportResult{Report: toContentReportDTO(report)}, nil
}

func (uc *ToolsUseCase) ListCommunityModLogs(ctx context.Context, input ListCommunityModLogsInput) (ListCommunityModLogsResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ListCommunityModLogsResult{}, err
	}
	limit, offset, err := normalizeModToolsPagination(input.Limit, input.Offset)
	if err != nil {
		return ListCommunityModLogsResult{}, err
	}
	logs, err := uc.tools.ListCommunityModLogs(ctx, ListCommunityModLogsRecordInput{
		CommunityID: communityID,
		Action:      strings.TrimSpace(input.Action),
		ActorID:     strings.TrimSpace(input.ActorFilterID),
		TargetType:  strings.TrimSpace(input.TargetType),
		TargetID:    strings.TrimSpace(input.TargetID),
		Limit:       limit + 1,
		Offset:      offset,
	})
	if err != nil {
		return ListCommunityModLogsResult{}, fmt.Errorf("list community moderation logs: %w", err)
	}
	logs, hasMore := trimToolsPage(logs, limit)
	return ListCommunityModLogsResult{Logs: logs, Limit: limit, Offset: offset, NextOffset: offset + len(logs), HasMore: hasMore}, nil
}

func (uc *ToolsUseCase) ListRemovalReasons(ctx context.Context, actorID userdomain.UserID, slug string) (ListModerationTemplatesResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, actorID, slug)
	if err != nil {
		return ListModerationTemplatesResult{}, err
	}
	templates, err := uc.tools.ListRemovalReasons(ctx, communityID)
	if err != nil {
		return ListModerationTemplatesResult{}, fmt.Errorf("list removal reasons: %w", err)
	}
	return ListModerationTemplatesResult{Templates: templates}, nil
}

func (uc *ToolsUseCase) CreateRemovalReason(ctx context.Context, input ModerationTemplateInput) (ModerationTemplateResult, error) {
	return uc.writeTemplate(ctx, input, "removal_reason", true)
}

func (uc *ToolsUseCase) UpdateRemovalReason(ctx context.Context, input ModerationTemplateInput) (ModerationTemplateResult, error) {
	return uc.writeTemplate(ctx, input, "removal_reason", false)
}

func (uc *ToolsUseCase) DeleteRemovalReason(ctx context.Context, input DeleteModerationTemplateInput) error {
	return uc.deleteTemplate(ctx, input, "removal_reason")
}

func (uc *ToolsUseCase) ListSavedResponses(ctx context.Context, actorID userdomain.UserID, slug string) (ListModerationTemplatesResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, actorID, slug)
	if err != nil {
		return ListModerationTemplatesResult{}, err
	}
	templates, err := uc.tools.ListSavedResponses(ctx, communityID)
	if err != nil {
		return ListModerationTemplatesResult{}, fmt.Errorf("list saved responses: %w", err)
	}
	return ListModerationTemplatesResult{Templates: templates}, nil
}

func (uc *ToolsUseCase) CreateSavedResponse(ctx context.Context, input ModerationTemplateInput) (ModerationTemplateResult, error) {
	return uc.writeTemplate(ctx, input, "saved_response", true)
}

func (uc *ToolsUseCase) UpdateSavedResponse(ctx context.Context, input ModerationTemplateInput) (ModerationTemplateResult, error) {
	return uc.writeTemplate(ctx, input, "saved_response", false)
}

func (uc *ToolsUseCase) DeleteSavedResponse(ctx context.Context, input DeleteModerationTemplateInput) error {
	return uc.deleteTemplate(ctx, input, "saved_response")
}

func (uc *ToolsUseCase) ListUserStates(ctx context.Context, input ListUserStatesInput) (ListUserStatesResult, error) {
	kind, err := normalizeUserStateKind(input.Kind)
	if err != nil {
		return ListUserStatesResult{}, err
	}
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ListUserStatesResult{}, err
	}
	limit, offset, err := normalizeModToolsPagination(input.Limit, input.Offset)
	if err != nil {
		return ListUserStatesResult{}, err
	}
	users, err := uc.tools.ListUserStates(ctx, communityID, kind, limit+1, offset)
	if err != nil {
		return ListUserStatesResult{}, fmt.Errorf("list community user states: %w", err)
	}
	users, hasMore := trimToolsPage(users, limit)
	return ListUserStatesResult{Users: users, Kind: kind, Limit: limit, Offset: offset, NextOffset: offset + len(users), HasMore: hasMore}, nil
}

func (uc *ToolsUseCase) UpsertUserState(ctx context.Context, input WriteUserStateInput) (UserStateResult, error) {
	kind, err := normalizeUserStateKind(input.Kind)
	if err != nil {
		return UserStateResult{}, err
	}
	userID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return UserStateResult{}, err
	}
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return UserStateResult{}, err
	}
	reason, err := textlimit.TrimmedOptionalMaxRunes(input.Reason, "community user moderation reason", moderationdomain.MaxReasonRunes)
	if err != nil {
		return UserStateResult{}, err
	}
	state, err := uc.tools.UpsertUserState(ctx, UpsertUserStateRecordInput{
		ID:          uuid.NewString(),
		CommunityID: communityID,
		UserID:      userID,
		Kind:        kind,
		Reason:      reason,
		ExpiresAt:   input.ExpiresAt,
		ActorID:     input.ActorID,
		UpdatedAt:   uc.now().UTC(),
	})
	if err != nil {
		return UserStateResult{}, fmt.Errorf("upsert community user state: %w", err)
	}
	return UserStateResult{User: state}, nil
}

func (uc *ToolsUseCase) DeleteUserState(ctx context.Context, input DeleteUserStateInput) error {
	kind, err := normalizeUserStateKind(input.Kind)
	if err != nil {
		return err
	}
	userID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return err
	}
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return err
	}
	return uc.tools.DeleteUserState(ctx, communityID, userID, kind, input.ActorID, uc.now().UTC())
}

func (uc *ToolsUseCase) GetUserProfile(ctx context.Context, input GetUserProfileInput) (GetUserProfileResult, error) {
	userID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return GetUserProfileResult{}, err
	}
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return GetUserProfileResult{}, err
	}
	profile, err := uc.tools.GetUserProfile(ctx, communityID, userID)
	if err != nil {
		return GetUserProfileResult{}, fmt.Errorf("get moderation user profile: %w", err)
	}
	return GetUserProfileResult{Profile: profile}, nil
}

func (uc *ToolsUseCase) ListModeratorNotes(ctx context.Context, input ListModeratorNotesInput) (ListModeratorNotesResult, error) {
	userID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return ListModeratorNotesResult{}, err
	}
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ListModeratorNotesResult{}, err
	}
	limit, offset, err := normalizeModToolsPagination(input.Limit, input.Offset)
	if err != nil {
		return ListModeratorNotesResult{}, err
	}
	notes, err := uc.tools.ListModeratorNotes(ctx, communityID, userID, limit+1, offset)
	if err != nil {
		return ListModeratorNotesResult{}, fmt.Errorf("list moderator notes: %w", err)
	}
	notes, hasMore := trimToolsPage(notes, limit)
	return ListModeratorNotesResult{Notes: notes, Limit: limit, Offset: offset, NextOffset: offset + len(notes), HasMore: hasMore}, nil
}

func (uc *ToolsUseCase) CreateModeratorNote(ctx context.Context, input CreateModeratorNoteInput) (ModeratorNoteResult, error) {
	userID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return ModeratorNoteResult{}, err
	}
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ModeratorNoteResult{}, err
	}
	body, err := textlimit.TrimmedRequiredMaxRunes(input.Body, "moderator note body", MaxModNoteBody)
	if err != nil {
		return ModeratorNoteResult{}, err
	}
	note, err := uc.tools.CreateModeratorNote(ctx, CreateModeratorNoteRecordInput{
		ID:          uuid.NewString(),
		CommunityID: communityID,
		UserID:      userID,
		AuthorID:    input.ActorID,
		Body:        body,
		CreatedAt:   uc.now().UTC(),
	})
	if err != nil {
		return ModeratorNoteResult{}, fmt.Errorf("create moderator note: %w", err)
	}
	return ModeratorNoteResult{Note: note}, nil
}

func (uc *ToolsUseCase) DeleteModeratorNote(ctx context.Context, input DeleteModeratorNoteInput) error {
	userID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return err
	}
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return err
	}
	noteID := normalizeOptionalUUID(input.NoteID)
	if noteID == "" {
		return apperr.New(apperr.CodeInvalidArgument, "moderator note id is invalid")
	}
	return uc.tools.DeleteModeratorNote(ctx, communityID, userID, noteID, input.ActorID, uc.now().UTC())
}

func (uc *ToolsUseCase) GetAutomodConfig(ctx context.Context, actorID userdomain.UserID, slug string) (AutomodConfigResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, actorID, slug)
	if err != nil {
		return AutomodConfigResult{}, err
	}
	config, err := uc.tools.GetAutomodConfig(ctx, communityID)
	if err != nil {
		return AutomodConfigResult{}, fmt.Errorf("get automod config: %w", err)
	}
	return AutomodConfigResult{Config: config}, nil
}

func (uc *ToolsUseCase) UpdateAutomodConfig(ctx context.Context, input AutomodConfigInput) (AutomodConfigResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return AutomodConfigResult{}, err
	}
	configText, err := textlimit.TrimmedOptionalMaxRunes(input.ConfigText, "automod config", MaxAutomodConfigRunes)
	if err != nil {
		return AutomodConfigResult{}, err
	}
	rules, err := normalizeJSON(input.Rules, "{}")
	if err != nil {
		return AutomodConfigResult{}, err
	}
	config, err := uc.tools.UpsertAutomodConfig(ctx, UpsertAutomodConfigRecordInput{
		CommunityID: communityID,
		ActorID:     input.ActorID,
		ConfigText:  configText,
		Rules:       rules,
		UpdatedAt:   uc.now().UTC(),
	})
	if err != nil {
		return AutomodConfigResult{}, fmt.Errorf("update automod config: %w", err)
	}
	return AutomodConfigResult{Config: config}, nil
}

func (uc *ToolsUseCase) ListAutomodVersions(ctx context.Context, input AutomodVersionsInput) (AutomodVersionsResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return AutomodVersionsResult{}, err
	}
	limit, offset, err := normalizeModToolsPagination(input.Limit, input.Offset)
	if err != nil {
		return AutomodVersionsResult{}, err
	}
	versions, err := uc.tools.ListAutomodVersions(ctx, communityID, limit+1, offset)
	if err != nil {
		return AutomodVersionsResult{}, fmt.Errorf("list automod versions: %w", err)
	}
	versions, hasMore := trimToolsPage(versions, limit)
	return AutomodVersionsResult{Versions: versions, Limit: limit, Offset: offset, NextOffset: offset + len(versions), HasMore: hasMore}, nil
}

func (uc *ToolsUseCase) DryRunAutomod(ctx context.Context, input AutomodDryRunInput) (AutomodDryRunResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return AutomodDryRunResult{}, err
	}
	targetType, err := moderationdomain.NewTargetType(input.TargetType)
	if err != nil {
		return AutomodDryRunResult{}, err
	}
	controls, err := uc.tools.GetContentControls(ctx, communityID)
	if err != nil {
		return AutomodDryRunResult{}, fmt.Errorf("get content controls: %w", err)
	}
	content := strings.ToLower(strings.TrimSpace(input.Title + "\n" + input.Body))
	matches := make([]AutomodDryRunMatch, 0)
	for _, keyword := range controls.BlockedKeywords {
		if keyword != "" && strings.Contains(content, strings.ToLower(keyword)) {
			matches = append(matches, AutomodDryRunMatch{Rule: "blocked_keyword", Action: "filter", Reason: "matched keyword: " + keyword})
		}
	}
	for _, domain := range controls.BlockedDomains {
		needle := strings.ToLower(domain)
		for _, link := range input.Links {
			if needle != "" && strings.Contains(strings.ToLower(link), needle) {
				matches = append(matches, AutomodDryRunMatch{Rule: "blocked_domain", Action: "filter", Reason: "matched domain: " + domain})
				break
			}
		}
	}
	if controls.FilterLinks && len(input.Links) > 0 {
		matches = append(matches, AutomodDryRunMatch{Rule: "link_filter", Action: "review", Reason: "links require moderator review"})
	}
	reasons := make([]string, 0, len(matches))
	for _, match := range matches {
		reasons = append(reasons, match.Reason)
	}
	action := "allow"
	if len(matches) > 0 {
		action = "filter"
	}
	_ = targetType
	return AutomodDryRunResult{Matches: matches, SuggestedAction: action, Reasons: reasons}, nil
}

func (uc *ToolsUseCase) GetContentControls(ctx context.Context, actorID userdomain.UserID, slug string) (ContentControlsResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, actorID, slug)
	if err != nil {
		return ContentControlsResult{}, err
	}
	controls, err := uc.tools.GetContentControls(ctx, communityID)
	if err != nil {
		return ContentControlsResult{}, fmt.Errorf("get content controls: %w", err)
	}
	return ContentControlsResult{Controls: controls}, nil
}

func (uc *ToolsUseCase) UpdateContentControls(ctx context.Context, input ContentControlsInput) (ContentControlsResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ContentControlsResult{}, err
	}
	keywords, err := normalizeStringList(input.BlockedKeywords, "blocked keyword")
	if err != nil {
		return ContentControlsResult{}, err
	}
	domains, err := normalizeStringList(input.BlockedDomains, "blocked domain")
	if err != nil {
		return ContentControlsResult{}, err
	}
	if input.MinAccountAgeDays < 0 || input.PostRateLimitPerHour < 0 || input.CommentRateLimitPerHour < 0 {
		return ContentControlsResult{}, apperr.New(apperr.CodeInvalidArgument, "content control limits must be non-negative")
	}
	controls, err := uc.tools.UpsertContentControls(ctx, UpsertContentControlsRecordInput{
		CommunityID:             communityID,
		ActorID:                 input.ActorID,
		BlockedKeywords:         keywords,
		BlockedDomains:          domains,
		MinAccountAgeDays:       input.MinAccountAgeDays,
		PostRateLimitPerHour:    input.PostRateLimitPerHour,
		CommentRateLimitPerHour: input.CommentRateLimitPerHour,
		BlockNewAccounts:        input.BlockNewAccounts,
		FilterLinks:             input.FilterLinks,
		UpdatedAt:               uc.now().UTC(),
	})
	if err != nil {
		return ContentControlsResult{}, fmt.Errorf("update content controls: %w", err)
	}
	return ContentControlsResult{Controls: controls}, nil
}

func (uc *ToolsUseCase) ListModmailConversations(ctx context.Context, input ListModmailConversationsInput) (ListModmailConversationsResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ListModmailConversationsResult{}, err
	}
	folder, err := normalizeModmailFolder(input.Folder)
	if err != nil {
		return ListModmailConversationsResult{}, err
	}
	limit, offset, err := normalizeModToolsPagination(input.Limit, input.Offset)
	if err != nil {
		return ListModmailConversationsResult{}, err
	}
	conversations, err := uc.tools.ListModmailConversations(ctx, ListModmailConversationsRecordInput{
		CommunityID: communityID,
		ActorID:     input.ActorID,
		Folder:      folder,
		Limit:       limit + 1,
		Offset:      offset,
	})
	if err != nil {
		return ListModmailConversationsResult{}, fmt.Errorf("list modmail conversations: %w", err)
	}
	conversations, hasMore := trimToolsPage(conversations, limit)
	return ListModmailConversationsResult{Conversations: conversations, Folder: folder, Limit: limit, Offset: offset, NextOffset: offset + len(conversations), HasMore: hasMore}, nil
}

func (uc *ToolsUseCase) CreateModmailConversation(ctx context.Context, input CreateModmailConversationInput) (ModmailConversationResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ModmailConversationResult{}, err
	}
	userID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return ModmailConversationResult{}, err
	}
	subject, err := textlimit.TrimmedRequiredMaxRunes(input.Subject, "modmail subject", MaxModmailSubjectRunes)
	if err != nil {
		return ModmailConversationResult{}, err
	}
	body, err := textlimit.TrimmedRequiredMaxRunes(input.Body, "modmail body", MaxModmailBodyRunes)
	if err != nil {
		return ModmailConversationResult{}, err
	}
	now := uc.now().UTC()
	detail, err := uc.tools.CreateModmailConversation(ctx, CreateModmailConversationRecordInput{
		ID:          uuid.NewString(),
		MessageID:   uuid.NewString(),
		CommunityID: communityID,
		ActorID:     input.ActorID,
		UserID:      userID,
		Subject:     subject,
		Body:        body,
		CreatedAt:   now,
	})
	if err != nil {
		return ModmailConversationResult{}, fmt.Errorf("create modmail conversation: %w", err)
	}
	return ModmailConversationResult{Detail: detail}, nil
}

func (uc *ToolsUseCase) GetModmailConversation(ctx context.Context, input GetModmailConversationInput) (ModmailConversationResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ModmailConversationResult{}, err
	}
	conversationID := normalizeOptionalUUID(input.ConversationID)
	if conversationID == "" {
		return ModmailConversationResult{}, apperr.New(apperr.CodeInvalidArgument, "modmail conversation id is invalid")
	}
	detail, err := uc.tools.GetModmailConversation(ctx, GetModmailConversationRecordInput{
		CommunityID:    communityID,
		ActorID:        input.ActorID,
		ConversationID: conversationID,
	})
	if err != nil {
		return ModmailConversationResult{}, fmt.Errorf("get modmail conversation: %w", err)
	}
	return ModmailConversationResult{Detail: detail}, nil
}

func (uc *ToolsUseCase) AddModmailMessage(ctx context.Context, input ModmailMessageInput) (ModmailConversationResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ModmailConversationResult{}, err
	}
	conversationID := normalizeOptionalUUID(input.ConversationID)
	if conversationID == "" {
		return ModmailConversationResult{}, apperr.New(apperr.CodeInvalidArgument, "modmail conversation id is invalid")
	}
	body, err := textlimit.TrimmedRequiredMaxRunes(input.Body, "modmail body", MaxModmailBodyRunes)
	if err != nil {
		return ModmailConversationResult{}, err
	}
	detail, err := uc.tools.AddModmailMessage(ctx, AddModmailMessageRecordInput{
		ID:             uuid.NewString(),
		CommunityID:    communityID,
		ActorID:        input.ActorID,
		ConversationID: conversationID,
		Body:           body,
		IsInternal:     input.IsInternal,
		CreatedAt:      uc.now().UTC(),
	})
	if err != nil {
		return ModmailConversationResult{}, fmt.Errorf("add modmail message: %w", err)
	}
	return ModmailConversationResult{Detail: detail}, nil
}

func (uc *ToolsUseCase) UpdateModmailConversation(ctx context.Context, input UpdateModmailConversationInput) (UpdateModmailConversationResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return UpdateModmailConversationResult{}, err
	}
	conversationID := normalizeOptionalUUID(input.ConversationID)
	if conversationID == "" {
		return UpdateModmailConversationResult{}, apperr.New(apperr.CodeInvalidArgument, "modmail conversation id is invalid")
	}
	folder, err := normalizeModmailFolder(input.Folder)
	if err != nil {
		return UpdateModmailConversationResult{}, err
	}
	status, err := normalizeModmailStatus(input.Status, folder)
	if err != nil {
		return UpdateModmailConversationResult{}, err
	}
	assignedTo := ""
	if strings.TrimSpace(input.AssignedTo) != "" {
		userID, err := userdomain.NewUserID(input.AssignedTo)
		if err != nil {
			return UpdateModmailConversationResult{}, err
		}
		assignedTo = userID.String()
	}
	conversation, err := uc.tools.UpdateModmailConversation(ctx, UpdateModmailConversationRecordInput{
		CommunityID:    communityID,
		ActorID:        input.ActorID,
		ConversationID: conversationID,
		Folder:         folder,
		Status:         status,
		AssignedTo:     assignedTo,
		MarkRead:       input.MarkRead,
		UpdatedAt:      uc.now().UTC(),
	})
	if err != nil {
		return UpdateModmailConversationResult{}, fmt.Errorf("update modmail conversation: %w", err)
	}
	return UpdateModmailConversationResult{Conversation: conversation}, nil
}

func (uc *ToolsUseCase) GetCommunityInsightsSummary(ctx context.Context, input CommunityInsightsInput) (CommunityInsightsSummaryResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return CommunityInsightsSummaryResult{}, err
	}
	rangeValue, since, err := normalizeInsightsRange(input.Range, uc.now().UTC())
	if err != nil {
		return CommunityInsightsSummaryResult{}, err
	}
	summary, err := uc.tools.GetCommunityInsightsSummary(ctx, CommunityInsightsRecordInput{
		CommunityID: communityID,
		Since:       since,
		Range:       rangeValue,
	})
	if err != nil {
		return CommunityInsightsSummaryResult{}, fmt.Errorf("get community insights summary: %w", err)
	}
	return CommunityInsightsSummaryResult{Summary: summary}, nil
}

func (uc *ToolsUseCase) GetCommunityModerationInsights(ctx context.Context, input CommunityInsightsInput) (CommunityModerationInsightsResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return CommunityModerationInsightsResult{}, err
	}
	rangeValue, since, err := normalizeInsightsRange(input.Range, uc.now().UTC())
	if err != nil {
		return CommunityModerationInsightsResult{}, err
	}
	insights, err := uc.tools.GetCommunityModerationInsights(ctx, CommunityInsightsRecordInput{
		CommunityID: communityID,
		Since:       since,
		Range:       rangeValue,
	})
	if err != nil {
		return CommunityModerationInsightsResult{}, fmt.Errorf("get community moderation insights: %w", err)
	}
	return CommunityModerationInsightsResult{Moderation: insights}, nil
}

func (uc *ToolsUseCase) ListCommunityTrainingQueue(ctx context.Context, input CommunityInsightsInput) (CommunityTrainingQueueResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return CommunityTrainingQueueResult{}, err
	}
	limit, offset, err := normalizeModToolsPagination(input.Limit, input.Offset)
	if err != nil {
		return CommunityTrainingQueueResult{}, err
	}
	items, err := uc.tools.ListModQueue(ctx, ListModQueueRecordInput{
		CommunityID: &communityID,
		Queue:       "needs_review",
		Limit:       limit + 1,
		Offset:      offset,
	})
	if err != nil {
		return CommunityTrainingQueueResult{}, fmt.Errorf("list training queue: %w", err)
	}
	items, hasMore := trimToolsPage(items, limit)
	trainingItems := make([]CommunityTrainingQueueItem, 0, len(items))
	for _, item := range items {
		trainingItems = append(trainingItems, CommunityTrainingQueueItem{
			ID:              item.ID,
			TargetType:      item.TargetType,
			TargetID:        item.TargetID,
			CommunityID:     item.CommunityID,
			Preview:         item.Preview,
			SuggestedAction: "review",
			Reason:          item.Queue,
			CreatedAt:       item.CreatedAt,
		})
	}
	return CommunityTrainingQueueResult{Items: trainingItems, Limit: limit, Offset: offset, NextOffset: offset + len(trainingItems), HasMore: hasMore}, nil
}

func (uc *ToolsUseCase) writeTemplate(ctx context.Context, input ModerationTemplateInput, kind string, create bool) (ModerationTemplateResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ModerationTemplateResult{}, err
	}
	title, err := textlimit.TrimmedRequiredMaxRunes(input.Title, "moderation template title", MaxToolTitleRunes)
	if err != nil {
		return ModerationTemplateResult{}, err
	}
	bodyLimit := MaxRemovalReasonBody
	if kind == "saved_response" {
		bodyLimit = MaxSavedResponseBody
	}
	var body string
	if kind == "saved_response" {
		body, err = textlimit.TrimmedRequiredMaxRunes(input.Body, "saved response body", bodyLimit)
	} else {
		body, err = textlimit.TrimmedOptionalMaxRunes(input.Body, "removal reason body", bodyLimit)
	}
	if err != nil {
		return ModerationTemplateResult{}, err
	}
	if input.Position < 0 {
		return ModerationTemplateResult{}, apperr.New(apperr.CodeInvalidArgument, "position must be non-negative")
	}
	now := uc.now().UTC()
	record := WriteModerationTemplateRecordInput{
		ID:          input.ID,
		CommunityID: communityID,
		ActorID:     input.ActorID,
		Title:       title,
		Body:        body,
		RuleID:      normalizeOptionalUUID(input.RuleID),
		Position:    input.Position,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if create {
		record.ID = uuid.NewString()
	}
	if record.ID = normalizeOptionalUUID(record.ID); record.ID == "" {
		return ModerationTemplateResult{}, apperr.New(apperr.CodeInvalidArgument, "moderation template id is invalid")
	}
	var template ModerationTemplate
	if kind == "saved_response" && create {
		template, err = uc.tools.CreateSavedResponse(ctx, record)
	} else if kind == "saved_response" {
		template, err = uc.tools.UpdateSavedResponse(ctx, record)
	} else if create {
		template, err = uc.tools.CreateRemovalReason(ctx, record)
	} else {
		template, err = uc.tools.UpdateRemovalReason(ctx, record)
	}
	if err != nil {
		return ModerationTemplateResult{}, fmt.Errorf("write moderation template: %w", err)
	}
	return ModerationTemplateResult{Template: template}, nil
}

func (uc *ToolsUseCase) deleteTemplate(ctx context.Context, input DeleteModerationTemplateInput, kind string) error {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return err
	}
	id := normalizeOptionalUUID(input.ID)
	if id == "" {
		return apperr.New(apperr.CodeInvalidArgument, "moderation template id is invalid")
	}
	if kind == "saved_response" {
		return uc.tools.DeleteSavedResponse(ctx, communityID, id, input.ActorID, uc.now().UTC())
	}
	return uc.tools.DeleteRemovalReason(ctx, communityID, id, input.ActorID, uc.now().UTC())
}

func (uc *ToolsUseCase) ensureScope(ctx context.Context, actorID userdomain.UserID, slug string) (*communitydomain.CommunityID, error) {
	if strings.TrimSpace(slug) == "" {
		if err := uc.ensurePlatformStaff(ctx, actorID); err != nil {
			return nil, err
		}
		return nil, nil
	}
	communityID, err := uc.ensureCommunityModerator(ctx, actorID, slug)
	if err != nil {
		return nil, err
	}
	return &communityID, nil
}

func (uc *ToolsUseCase) ensurePlatformStaff(ctx context.Context, actorID userdomain.UserID) error {
	if strings.TrimSpace(actorID.String()) == "" {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	ok, err := uc.staff.IsPlatformStaff(ctx, actorID)
	if err != nil {
		return fmt.Errorf("check platform staff: %w", err)
	}
	if !ok {
		return apperr.New(apperr.CodeForbidden, "platform staff required")
	}
	return nil
}

func (uc *ToolsUseCase) ensureCommunityModerator(ctx context.Context, actorID userdomain.UserID, rawSlug string) (communitydomain.CommunityID, error) {
	if strings.TrimSpace(actorID.String()) == "" {
		return "", apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	slug, err := communitydomain.NewCommunitySlug(rawSlug)
	if err != nil {
		return "", err
	}
	community, err := uc.communities.FindBySlug(ctx, slug)
	if err != nil {
		return "", fmt.Errorf("find community: %w", err)
	}
	if community.Status() != communitydomain.CommunityStatusActive {
		return "", apperr.New(apperr.CodeNotFound, "community not found")
	}
	if uc.owners != nil {
		isOwner, err := uc.owners.IsPlatformOwner(ctx, actorID)
		if err != nil {
			return "", fmt.Errorf("check platform owner override: %w", err)
		}
		if isOwner {
			return community.ID(), nil
		}
	}
	roles, err := uc.roles.FindActiveRolesByUser(ctx, []communitydomain.CommunityID{community.ID()}, actorID)
	if err != nil {
		return "", fmt.Errorf("find community role: %w", err)
	}
	role, ok := roles[community.ID()]
	if !ok || (role != communitydomain.MembershipRoleOwner && role != communitydomain.MembershipRoleModerator) {
		return "", apperr.New(apperr.CodeForbidden, "community moderator required")
	}
	return community.ID(), nil
}

func platformOwnerRepository(staff PlatformStaffRepository) PlatformOwnerRepository {
	owners, ok := staff.(PlatformOwnerRepository)
	if !ok {
		return nil
	}
	return owners
}

func normalizeModQueue(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		value = "reports"
	}
	switch value {
	case "reports", "spam", "removed", "edited", "unmoderated", "needs_review":
		return value, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "moderation queue is invalid")
	}
}

func normalizeJSON(raw json.RawMessage, fallback string) (json.RawMessage, error) {
	value := strings.TrimSpace(string(raw))
	if value == "" {
		value = fallback
	}
	if !json.Valid([]byte(value)) {
		return nil, apperr.New(apperr.CodeInvalidArgument, "json payload is invalid")
	}
	return json.RawMessage(value), nil
}

func normalizeStringList(values []string, label string) ([]string, error) {
	if len(values) > MaxContentControlItems {
		return nil, apperr.New(apperr.CodeInvalidArgument, label+" list is too large")
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		item, err := textlimit.TrimmedRequiredMaxRunes(value, label, MaxToolTitleRunes)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result, nil
}

func normalizeModmailFolder(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		value = "inbox"
	}
	switch value {
	case "inbox", "needs_reply", "in_progress", "archived":
		return value, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "modmail folder is invalid")
	}
}

func normalizeModmailStatus(raw string, folder string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		switch folder {
		case "archived":
			value = "archived"
		case "needs_reply":
			value = "needs_reply"
		case "in_progress":
			value = "in_progress"
		default:
			value = "open"
		}
	}
	switch value {
	case "open", "needs_reply", "in_progress", "archived", "closed":
		return value, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "modmail status is invalid")
	}
}

func normalizeInsightsRange(raw string, now time.Time) (string, time.Time, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		value = "7d"
	}
	var duration time.Duration
	switch value {
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	case "90d":
		duration = 90 * 24 * time.Hour
	default:
		return "", time.Time{}, apperr.New(apperr.CodeInvalidArgument, "insights range is invalid")
	}
	return value, now.Add(-duration), nil
}

func parseModQueueItemID(raw string) (moderationdomain.TargetType, string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", "", apperr.New(apperr.CodeInvalidArgument, "moderation queue item id is required")
	}
	parts := strings.Split(value, ":")
	if len(parts) == 3 {
		if _, err := normalizeModQueue(parts[0]); err != nil {
			return "", "", err
		}
		parts = parts[1:]
	}
	if len(parts) != 2 {
		return "", "", apperr.New(apperr.CodeInvalidArgument, "moderation queue item id is invalid")
	}
	targetType, err := moderationdomain.NewTargetType(parts[0])
	if err != nil {
		return "", "", err
	}
	targetID, err := normalizeTargetID(targetType, parts[1])
	if err != nil {
		return "", "", err
	}
	return targetType, targetID, nil
}

type normalizedModerationTarget struct {
	TargetType moderationdomain.TargetType
	TargetID   string
}

func normalizeModerationTargets(rawType string, targetIDs []string, targets []ModerationTargetInput) ([]normalizedModerationTarget, error) {
	normalized := make([]normalizedModerationTarget, 0, len(targets)+len(targetIDs))
	if len(targetIDs) > 0 {
		targetType, err := moderationdomain.NewTargetType(rawType)
		if err != nil {
			return nil, err
		}
		for _, rawID := range targetIDs {
			targetID, err := normalizeTargetID(targetType, rawID)
			if err != nil {
				return nil, err
			}
			normalized = append(normalized, normalizedModerationTarget{TargetType: targetType, TargetID: targetID})
		}
	}
	for _, target := range targets {
		targetType, err := moderationdomain.NewTargetType(target.TargetType)
		if err != nil {
			return nil, err
		}
		targetID, err := normalizeTargetID(targetType, target.TargetID)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, normalizedModerationTarget{TargetType: targetType, TargetID: targetID})
	}
	if len(normalized) == 0 {
		return nil, apperr.New(apperr.CodeInvalidArgument, "moderation action targets are required")
	}
	return normalized, nil
}

func normalizeTargetID(targetType moderationdomain.TargetType, rawID string) (string, error) {
	switch targetType {
	case moderationdomain.TargetTypePost:
		id, err := postdomain.NewPostID(rawID)
		if err != nil {
			return "", err
		}
		return id.String(), nil
	case moderationdomain.TargetTypeComment:
		id, err := commentdomain.NewCommentID(rawID)
		if err != nil {
			return "", err
		}
		return id.String(), nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "moderation target type is invalid")
	}
}

func normalizeActionReason(raw string, confirm bool) (string, error) {
	reason, err := textlimit.TrimmedOptionalMaxRunes(raw, "moderation reason", moderationdomain.MaxReasonRunes)
	if err != nil {
		return "", err
	}
	if reason == "" {
		if !confirm {
			return "", apperr.New(apperr.CodeInvalidArgument, "moderation reason or confirm is required")
		}
		reason = "confirmed without reason"
	}
	return reason, nil
}

func normalizeOptionalUUID(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return ""
	}
	return parsed.String()
}

func normalizeUserStateKind(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case UserStateBanned:
		return UserStateBanned, nil
	case UserStateMuted:
		return UserStateMuted, nil
	case UserStateApproved:
		return UserStateApproved, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "community user state kind is invalid")
	}
}

func normalizeModToolsPagination(limit int, offset int) (int, int, error) {
	if limit < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "limit must be non-negative")
	}
	if offset < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "offset must be non-negative")
	}
	if limit == 0 {
		limit = DefaultModToolsListLimit
	}
	if limit > MaxModToolsListLimit {
		limit = MaxModToolsListLimit
	}
	return limit, offset, nil
}

func trimToolsPage[T any](items []T, limit int) ([]T, bool) {
	if len(items) <= limit {
		return items, false
	}
	return items[:limit], true
}

func errorResult(err error) (string, string) {
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return string(appErr.Code()), appErr.Message()
	}
	return string(apperr.CodeInternal), "moderation action failed"
}

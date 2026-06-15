package progressionusecase

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/textlimit"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

const (
	DefaultProgressionListLimit = 20
	MaxProgressionListLimit     = 50
	MaxTitleNameRunes           = 20
	MaxTitleDescriptionRunes    = 120
	MaxTitleGrantReasonRunes    = 500
)

const (
	TitleScopePlatform  = "platform"
	TitleScopeSystem    = "system"
	TitleScopeCommunity = "community"
)

const (
	XPSourcePostPublish    = "post_publish"
	XPSourceCommentPublish = "comment_publish"
	XPSourcePostUpvote     = "post_upvote_received"
	XPSourceCommentUpvote  = "comment_upvote_received"
	XPSourcePostSave       = "post_saved_received"
	XPSourceDailyLogin     = "daily_login"
	XPReasonPostPublish    = "post_publish"
	XPReasonCommentPublish = "comment_publish"
	XPReasonPostUpvote     = "post_upvote_received"
	XPReasonCommentUpvote  = "comment_upvote_received"
	XPReasonPostSave       = "post_saved_received"
	XPReasonDailyLogin     = "daily_login"
)

var levelThresholds = []int{
	0, 100, 260, 520, 900, 1450, 2200, 3200, 4500, 6200,
	8400, 11200, 14800, 19400, 25200, 32500, 41600, 52900, 66800, 83800,
	104500, 129600, 159900, 196400, 240000, 292000, 354000, 428000, 516000, 620000,
}

var xpPolicies = map[string]XPPolicy{
	XPSourcePostPublish:    {Delta: 20, DailyCap: 100, Reason: XPReasonPostPublish},
	XPSourceCommentPublish: {Delta: 5, DailyCap: 80, Reason: XPReasonCommentPublish},
	XPSourcePostUpvote:     {Delta: 3, DailyCap: 150, Reason: XPReasonPostUpvote},
	XPSourceCommentUpvote:  {Delta: 2, DailyCap: 150, Reason: XPReasonCommentUpvote},
	XPSourcePostSave:       {Delta: 8, DailyCap: 120, Reason: XPReasonPostSave},
	XPSourceDailyLogin:     {Delta: 5, DailyCap: 5, Reason: XPReasonDailyLogin},
}

type UseCase struct {
	repository   Repository
	transactions TransactionManager
	now          func() time.Time
}

type Repository interface {
	IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error)
	GetOrCreateProgression(ctx context.Context, userID userdomain.UserID, now time.Time) (ProgressionRecord, error)
	GetPublicProgression(ctx context.Context, userID userdomain.UserID, now time.Time) (ProgressionRecord, error)
	ListXPEvents(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]XPEvent, error)
	GrantXP(ctx context.Context, input GrantXPRecordInput) (GrantXPRecordResult, error)
	ListTitles(ctx context.Context, filter TitleFilter, limit int, offset int) ([]Title, error)
	CreateTitle(ctx context.Context, input CreateTitleRecordInput) (Title, error)
	UpdateTitle(ctx context.Context, input UpdateTitleRecordInput) (Title, error)
	FindTitleByID(ctx context.Context, titleID string) (Title, error)
	ListUserTitleGrants(ctx context.Context, userID userdomain.UserID, now time.Time, limit int, offset int) ([]TitleGrant, error)
	GrantTitle(ctx context.Context, input GrantTitleRecordInput) (TitleGrant, error)
	RevokeTitle(ctx context.Context, grantID string, revokedAt time.Time) (TitleGrant, error)
	SetActiveTitle(ctx context.Context, userID userdomain.UserID, grantID *string, now time.Time) (ProgressionRecord, error)
	FindUserByID(ctx context.Context, userID userdomain.UserID) (User, error)
	CreateAdminAuditLog(ctx context.Context, log AdminAuditLog) error
}

type TransactionManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, repository Repository) error) error
}

type XPPolicy struct {
	Delta    int
	DailyCap int
	Reason   string
}

type ProgressionRecord struct {
	UserID      string
	XPTotal     int
	ActiveTitle *TitleSummary
	TitlesCount int
	UpdatedAt   time.Time
}

type Progression struct {
	UserID         string
	XPTotal        int
	Level          int
	LevelName      string
	CurrentLevelXP int
	NextLevelXP    *int
	LevelProgress  float64
	ActiveTitle    *TitleSummary
	TitlesCount    int
	UpdatedAt      time.Time
}

type XPEvent struct {
	ID           string
	UserID       string
	Delta        int
	XPTotalAfter int
	Reason       string
	SourceType   string
	SourceID     string
	ActorID      string
	CreatedAt    time.Time
}

type Title struct {
	ID          string
	Name        string
	Description string
	ScopeType   string
	ScopeID     string
	IsActive    bool
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TitleGrant struct {
	ID        string
	UserID    string
	Title     Title
	GrantedBy string
	Reason    string
	ExpiresAt *time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type TitleSummary struct {
	GrantID   string
	TitleID   string
	Name      string
	ScopeType string
	ScopeID   string
}

type User struct {
	ID       string
	Username string
	Status   string
}

type AdminAuditLog struct {
	ID         string
	ActorID    string
	Action     string
	TargetType string
	TargetID   string
	Before     map[string]any
	After      map[string]any
	CreatedAt  time.Time
}

type GrantXPRecordInput struct {
	EventID    string
	UserID     userdomain.UserID
	ActorID    userdomain.UserID
	Delta      int
	DailyCap   int
	Reason     string
	SourceType string
	SourceID   string
	CreatedAt  time.Time
}

type GrantXPRecordResult struct {
	Event       *XPEvent
	Progression ProgressionRecord
	Granted     bool
}

type TitleFilter struct {
	ScopeType string
	Active    *bool
}

type CreateTitleRecordInput struct {
	ID          string
	Name        string
	Description string
	ScopeType   string
	ScopeID     string
	CreatedBy   userdomain.UserID
	CreatedAt   time.Time
}

type UpdateTitleRecordInput struct {
	TitleID     string
	Name        string
	Description string
	IsActive    bool
	UpdatedAt   time.Time
}

type GrantTitleRecordInput struct {
	ID        string
	UserID    userdomain.UserID
	TitleID   string
	GrantedBy userdomain.UserID
	Reason    string
	ExpiresAt *time.Time
	CreatedAt time.Time
}

type GetMyProgressionInput struct {
	UserID userdomain.UserID
}

type GetMyProgressionResult struct {
	Progression Progression
}

type ListMyXPEventsInput struct {
	UserID userdomain.UserID
	Limit  int
	Offset int
}

type ListMyXPEventsResult struct {
	Events     []XPEvent
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type GrantXPInput struct {
	UserID     userdomain.UserID
	ActorID    userdomain.UserID
	SourceType string
	SourceID   string
}

type ListMyTitlesInput struct {
	UserID userdomain.UserID
	Limit  int
	Offset int
}

type ListMyTitlesResult struct {
	Titles     []TitleGrant
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type SetActiveTitleInput struct {
	UserID       userdomain.UserID
	TitleGrantID *string
}

type SetActiveTitleResult struct {
	Progression Progression
}

type ListTitlesInput struct {
	ActorID   userdomain.UserID
	ScopeType string
	Active    string
	Limit     int
	Offset    int
}

type ListTitlesResult struct {
	Titles     []Title
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type CreateTitleInput struct {
	ActorID     userdomain.UserID
	Name        string
	Description string
	ScopeType   string
	ScopeID     string
}

type CreateTitleResult struct {
	Title Title
}

type UpdateTitleInput struct {
	ActorID     userdomain.UserID
	TitleID     string
	Name        *string
	Description *string
	IsActive    *bool
}

type UpdateTitleResult struct {
	Title Title
}

type ListUserTitleGrantsInput struct {
	ActorID userdomain.UserID
	UserID  string
	Limit   int
	Offset  int
}

type ListUserTitleGrantsResult struct {
	Titles     []TitleGrant
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type GrantTitleInput struct {
	ActorID   userdomain.UserID
	UserID    string
	TitleID   string
	Reason    string
	ExpiresAt *time.Time
}

type GrantTitleResult struct {
	Grant TitleGrant
}

type RevokeTitleInput struct {
	ActorID userdomain.UserID
	UserID  string
	GrantID string
}

type RevokeTitleResult struct {
	Grant TitleGrant
}

func NewUseCase(repository Repository, now func() time.Time) *UseCase {
	if now == nil {
		now = time.Now
	}
	uc := &UseCase{repository: repository, now: now}
	if transactions, ok := repository.(TransactionManager); ok {
		uc.transactions = transactions
	}
	return uc
}

func (uc *UseCase) GetMyProgression(ctx context.Context, input GetMyProgressionInput) (GetMyProgressionResult, error) {
	if err := requireAuthenticated(input.UserID); err != nil {
		return GetMyProgressionResult{}, err
	}
	record, err := uc.repository.GetOrCreateProgression(ctx, input.UserID, uc.now().UTC())
	if err != nil {
		return GetMyProgressionResult{}, fmt.Errorf("get user progression: %w", err)
	}
	return GetMyProgressionResult{Progression: BuildProgression(record)}, nil
}

func (uc *UseCase) GetPublicProgression(ctx context.Context, userID userdomain.UserID) (Progression, error) {
	record, err := uc.repository.GetPublicProgression(ctx, userID, uc.now().UTC())
	if err != nil {
		return Progression{}, fmt.Errorf("get public progression: %w", err)
	}
	return BuildProgression(record), nil
}

func (uc *UseCase) ListMyXPEvents(ctx context.Context, input ListMyXPEventsInput) (ListMyXPEventsResult, error) {
	if err := requireAuthenticated(input.UserID); err != nil {
		return ListMyXPEventsResult{}, err
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListMyXPEventsResult{}, err
	}
	events, err := uc.repository.ListXPEvents(ctx, input.UserID, limit+1, offset)
	if err != nil {
		return ListMyXPEventsResult{}, fmt.Errorf("list xp events: %w", err)
	}
	events, hasMore := trimPage(events, limit)
	return ListMyXPEventsResult{Events: events, Limit: limit, Offset: offset, NextOffset: offset + len(events), HasMore: hasMore}, nil
}

func (uc *UseCase) GrantXP(ctx context.Context, input GrantXPInput) error {
	policy, ok := xpPolicies[strings.TrimSpace(input.SourceType)]
	if !ok {
		return apperr.New(apperr.CodeInvalidArgument, "xp source type is invalid")
	}
	if strings.TrimSpace(input.UserID.String()) == "" {
		return apperr.New(apperr.CodeInvalidArgument, "xp user is required")
	}
	sourceID := strings.TrimSpace(input.SourceID)
	if sourceID == "" {
		return apperr.New(apperr.CodeInvalidArgument, "xp source id is required")
	}
	_, err := uc.repository.GrantXP(ctx, GrantXPRecordInput{
		EventID:    uuid.NewString(),
		UserID:     input.UserID,
		ActorID:    input.ActorID,
		Delta:      policy.Delta,
		DailyCap:   policy.DailyCap,
		Reason:     policy.Reason,
		SourceType: input.SourceType,
		SourceID:   sourceID,
		CreatedAt:  uc.now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("grant xp: %w", err)
	}
	return nil
}

func (uc *UseCase) ListMyTitles(ctx context.Context, input ListMyTitlesInput) (ListMyTitlesResult, error) {
	if err := requireAuthenticated(input.UserID); err != nil {
		return ListMyTitlesResult{}, err
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListMyTitlesResult{}, err
	}
	titles, err := uc.repository.ListUserTitleGrants(ctx, input.UserID, uc.now().UTC(), limit+1, offset)
	if err != nil {
		return ListMyTitlesResult{}, fmt.Errorf("list my titles: %w", err)
	}
	titles, hasMore := trimPage(titles, limit)
	return ListMyTitlesResult{Titles: titles, Limit: limit, Offset: offset, NextOffset: offset + len(titles), HasMore: hasMore}, nil
}

func (uc *UseCase) SetActiveTitle(ctx context.Context, input SetActiveTitleInput) (SetActiveTitleResult, error) {
	if err := requireAuthenticated(input.UserID); err != nil {
		return SetActiveTitleResult{}, err
	}
	var grantID *string
	if input.TitleGrantID != nil {
		normalized, err := normalizeUUIDString(*input.TitleGrantID, "title grant id")
		if err != nil {
			return SetActiveTitleResult{}, err
		}
		grantID = &normalized
	}
	record, err := uc.repository.SetActiveTitle(ctx, input.UserID, grantID, uc.now().UTC())
	if err != nil {
		return SetActiveTitleResult{}, fmt.Errorf("set active title: %w", err)
	}
	return SetActiveTitleResult{Progression: BuildProgression(record)}, nil
}

func (uc *UseCase) ListTitles(ctx context.Context, input ListTitlesInput) (ListTitlesResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return ListTitlesResult{}, err
	}
	scopeType, err := normalizeOptionalTitleScope(input.ScopeType)
	if err != nil {
		return ListTitlesResult{}, err
	}
	active, err := normalizeOptionalBool(input.Active, "active")
	if err != nil {
		return ListTitlesResult{}, err
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListTitlesResult{}, err
	}
	titles, err := uc.repository.ListTitles(ctx, TitleFilter{ScopeType: scopeType, Active: active}, limit+1, offset)
	if err != nil {
		return ListTitlesResult{}, fmt.Errorf("list titles: %w", err)
	}
	titles, hasMore := trimPage(titles, limit)
	return ListTitlesResult{Titles: titles, Limit: limit, Offset: offset, NextOffset: offset + len(titles), HasMore: hasMore}, nil
}

func (uc *UseCase) CreateTitle(ctx context.Context, input CreateTitleInput) (CreateTitleResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return CreateTitleResult{}, err
	}
	name, err := normalizeTitleName(input.Name)
	if err != nil {
		return CreateTitleResult{}, err
	}
	description, err := textlimit.TrimmedOptionalMaxRunes(input.Description, "title description", MaxTitleDescriptionRunes)
	if err != nil {
		return CreateTitleResult{}, err
	}
	scopeType, err := normalizeTitleScope(input.ScopeType)
	if err != nil {
		return CreateTitleResult{}, err
	}
	scopeID := strings.TrimSpace(input.ScopeID)
	if scopeType == TitleScopeCommunity && scopeID == "" {
		return CreateTitleResult{}, apperr.New(apperr.CodeInvalidArgument, "title community scope id is required")
	}
	if scopeType != TitleScopeCommunity {
		scopeID = ""
	}
	now := uc.now().UTC()
	var title Title
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		created, err := repository.CreateTitle(ctx, CreateTitleRecordInput{
			ID:          uuid.NewString(),
			Name:        name,
			Description: description,
			ScopeType:   scopeType,
			ScopeID:     scopeID,
			CreatedBy:   input.ActorID,
			CreatedAt:   now,
		})
		if err != nil {
			return fmt.Errorf("create title: %w", err)
		}
		if err := repository.CreateAdminAuditLog(ctx, newAudit(input.ActorID, "admin.titles.create", "title", created.ID, map[string]any{}, titleAuditState(created), now)); err != nil {
			return fmt.Errorf("create title audit log: %w", err)
		}
		title = created
		return nil
	}); err != nil {
		return CreateTitleResult{}, err
	}
	return CreateTitleResult{Title: title}, nil
}

func (uc *UseCase) UpdateTitle(ctx context.Context, input UpdateTitleInput) (UpdateTitleResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return UpdateTitleResult{}, err
	}
	titleID, err := normalizeUUIDString(input.TitleID, "title id")
	if err != nil {
		return UpdateTitleResult{}, err
	}
	before, err := uc.repository.FindTitleByID(ctx, titleID)
	if err != nil {
		return UpdateTitleResult{}, fmt.Errorf("find title: %w", err)
	}
	name := before.Name
	if input.Name != nil {
		name, err = normalizeTitleName(*input.Name)
		if err != nil {
			return UpdateTitleResult{}, err
		}
	}
	description := before.Description
	if input.Description != nil {
		description, err = textlimit.TrimmedOptionalMaxRunes(*input.Description, "title description", MaxTitleDescriptionRunes)
		if err != nil {
			return UpdateTitleResult{}, err
		}
	}
	active := before.IsActive
	if input.IsActive != nil {
		active = *input.IsActive
	}
	now := uc.now().UTC()
	var after Title
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		updated, err := repository.UpdateTitle(ctx, UpdateTitleRecordInput{
			TitleID:     titleID,
			Name:        name,
			Description: description,
			IsActive:    active,
			UpdatedAt:   now,
		})
		if err != nil {
			return fmt.Errorf("update title: %w", err)
		}
		if err := repository.CreateAdminAuditLog(ctx, newAudit(input.ActorID, "admin.titles.update", "title", titleID, titleAuditState(before), titleAuditState(updated), now)); err != nil {
			return fmt.Errorf("create title audit log: %w", err)
		}
		after = updated
		return nil
	}); err != nil {
		return UpdateTitleResult{}, err
	}
	return UpdateTitleResult{Title: after}, nil
}

func (uc *UseCase) ListUserTitleGrants(ctx context.Context, input ListUserTitleGrantsInput) (ListUserTitleGrantsResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return ListUserTitleGrantsResult{}, err
	}
	targetID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return ListUserTitleGrantsResult{}, err
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListUserTitleGrantsResult{}, err
	}
	grants, err := uc.repository.ListUserTitleGrants(ctx, targetID, uc.now().UTC(), limit+1, offset)
	if err != nil {
		return ListUserTitleGrantsResult{}, fmt.Errorf("list user title grants: %w", err)
	}
	grants, hasMore := trimPage(grants, limit)
	return ListUserTitleGrantsResult{Titles: grants, Limit: limit, Offset: offset, NextOffset: offset + len(grants), HasMore: hasMore}, nil
}

func (uc *UseCase) GrantTitle(ctx context.Context, input GrantTitleInput) (GrantTitleResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return GrantTitleResult{}, err
	}
	targetID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return GrantTitleResult{}, err
	}
	user, err := uc.repository.FindUserByID(ctx, targetID)
	if err != nil {
		return GrantTitleResult{}, fmt.Errorf("find title target user: %w", err)
	}
	if user.Status == "deleted" {
		return GrantTitleResult{}, apperr.New(apperr.CodeNotFound, "user not found")
	}
	titleID, err := normalizeUUIDString(input.TitleID, "title id")
	if err != nil {
		return GrantTitleResult{}, err
	}
	title, err := uc.repository.FindTitleByID(ctx, titleID)
	if err != nil {
		return GrantTitleResult{}, fmt.Errorf("find title: %w", err)
	}
	if !title.IsActive {
		return GrantTitleResult{}, apperr.New(apperr.CodeInvalidArgument, "title is inactive")
	}
	reason, err := textlimit.TrimmedOptionalMaxRunes(input.Reason, "title grant reason", MaxTitleGrantReasonRunes)
	if err != nil {
		return GrantTitleResult{}, err
	}
	now := uc.now().UTC()
	if input.ExpiresAt != nil && !input.ExpiresAt.After(now) {
		return GrantTitleResult{}, apperr.New(apperr.CodeInvalidArgument, "title grant expiry is invalid")
	}
	var grant TitleGrant
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		created, err := repository.GrantTitle(ctx, GrantTitleRecordInput{
			ID:        uuid.NewString(),
			UserID:    targetID,
			TitleID:   titleID,
			GrantedBy: input.ActorID,
			Reason:    reason,
			ExpiresAt: input.ExpiresAt,
			CreatedAt: now,
		})
		if err != nil {
			return fmt.Errorf("grant title: %w", err)
		}
		if err := repository.CreateAdminAuditLog(ctx, newAudit(input.ActorID, "admin.titles.grant", "user", targetID.String(), map[string]any{}, titleGrantAuditState(created), now)); err != nil {
			return fmt.Errorf("create title grant audit log: %w", err)
		}
		grant = created
		return nil
	}); err != nil {
		return GrantTitleResult{}, err
	}
	return GrantTitleResult{Grant: grant}, nil
}

func (uc *UseCase) RevokeTitle(ctx context.Context, input RevokeTitleInput) (RevokeTitleResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return RevokeTitleResult{}, err
	}
	targetID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return RevokeTitleResult{}, err
	}
	grantID, err := normalizeUUIDString(input.GrantID, "title grant id")
	if err != nil {
		return RevokeTitleResult{}, err
	}
	now := uc.now().UTC()
	var grant TitleGrant
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		revoked, err := repository.RevokeTitle(ctx, grantID, now)
		if err != nil {
			return fmt.Errorf("revoke title: %w", err)
		}
		if revoked.UserID != targetID.String() {
			return apperr.New(apperr.CodeNotFound, "title grant not found")
		}
		if err := repository.CreateAdminAuditLog(ctx, newAudit(input.ActorID, "admin.titles.revoke", "user", targetID.String(), titleGrantAuditState(revoked), map[string]any{"grant_id": revoked.ID, "revoked_at": now}, now)); err != nil {
			return fmt.Errorf("create title revoke audit log: %w", err)
		}
		grant = revoked
		return nil
	}); err != nil {
		return RevokeTitleResult{}, err
	}
	return RevokeTitleResult{Grant: grant}, nil
}

func BuildProgression(record ProgressionRecord) Progression {
	level, current, next := ResolveLevel(record.XPTotal)
	return Progression{
		UserID:         record.UserID,
		XPTotal:        record.XPTotal,
		Level:          level,
		LevelName:      LevelName(level),
		CurrentLevelXP: current,
		NextLevelXP:    next,
		LevelProgress:  LevelProgress(record.XPTotal, current, next),
		ActiveTitle:    record.ActiveTitle,
		TitlesCount:    record.TitlesCount,
		UpdatedAt:      record.UpdatedAt,
	}
}

func ResolveLevel(xpTotal int) (int, int, *int) {
	if xpTotal < 0 {
		xpTotal = 0
	}
	level := 1
	for index, threshold := range levelThresholds {
		if xpTotal >= threshold {
			level = index + 1
		}
	}
	current := levelThresholds[level-1]
	if level >= len(levelThresholds) {
		return level, current, nil
	}
	next := levelThresholds[level]
	return level, current, &next
}

func LevelName(level int) string {
	switch {
	case level <= 5:
		return "初来乍到"
	case level <= 10:
		return "熟悉校园"
	case level <= 15:
		return "活跃同学"
	case level <= 20:
		return "资深贡献者"
	case level <= 25:
		return "社区骨干"
	default:
		return "Nexus 老用户"
	}
}

func LevelProgress(xpTotal int, current int, next *int) float64 {
	if next == nil {
		return 1
	}
	width := *next - current
	if width <= 0 {
		return 1
	}
	progress := float64(xpTotal-current) / float64(width)
	if progress < 0 {
		return 0
	}
	if progress > 1 {
		return 1
	}
	return progress
}

func (uc *UseCase) ensurePlatformStaff(ctx context.Context, actorID userdomain.UserID) error {
	if err := requireAuthenticated(actorID); err != nil {
		return err
	}
	isStaff, err := uc.repository.IsPlatformStaff(ctx, actorID)
	if err != nil {
		return fmt.Errorf("check platform staff: %w", err)
	}
	if !isStaff {
		return apperr.New(apperr.CodeForbidden, "platform staff required")
	}
	return nil
}

func (uc *UseCase) withWriteRepository(ctx context.Context, fn func(ctx context.Context, repository Repository) error) error {
	if uc.transactions == nil {
		return apperr.New(apperr.CodeInternal, "progression transaction support is not configured")
	}
	return uc.transactions.WithinTx(ctx, fn)
}

func requireAuthenticated(userID userdomain.UserID) error {
	if strings.TrimSpace(userID.String()) == "" {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	return nil
}

func normalizePagination(limit int, offset int) (int, int, error) {
	if limit < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "limit must be non-negative")
	}
	if offset < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "offset must be non-negative")
	}
	if limit == 0 {
		limit = DefaultProgressionListLimit
	}
	if limit > MaxProgressionListLimit {
		limit = MaxProgressionListLimit
	}
	return limit, offset, nil
}

func trimPage[T any](items []T, limit int) ([]T, bool) {
	if len(items) <= limit {
		return items, false
	}
	return items[:limit], true
}

func normalizeOptionalBool(raw string, field string) (*bool, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" || value == "all" {
		return nil, nil
	}
	switch value {
	case "true":
		parsed := true
		return &parsed, nil
	case "false":
		parsed := false
		return &parsed, nil
	default:
		return nil, apperr.New(apperr.CodeInvalidArgument, field+" query is invalid")
	}
}

func normalizeTitleScope(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		value = TitleScopePlatform
	}
	switch value {
	case TitleScopePlatform, TitleScopeSystem, TitleScopeCommunity:
		return value, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "title scope type is invalid")
	}
}

func normalizeOptionalTitleScope(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" || value == "all" {
		return "all", nil
	}
	return normalizeTitleScope(value)
}

func normalizeTitleName(raw string) (string, error) {
	value, err := textlimit.TrimmedRequiredMaxRunes(raw, "title name", MaxTitleNameRunes)
	if err != nil {
		return "", err
	}
	lower := strings.ToLower(value)
	for _, reserved := range []string{"官方", "管理员", "认证", "平台", "系统", "教务处", "学生会", "版主", "owner", "admin", "official", "verified"} {
		if strings.Contains(lower, strings.ToLower(reserved)) {
			return "", apperr.New(apperr.CodeInvalidArgument, "title name uses reserved words")
		}
	}
	return value, nil
}

func normalizeUUIDString(raw string, field string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, field+" is required")
	}
	if _, err := uuid.Parse(value); err != nil {
		return "", apperr.New(apperr.CodeInvalidArgument, field+" is invalid")
	}
	return value, nil
}

func newAudit(actorID userdomain.UserID, action string, targetType string, targetID string, before map[string]any, after map[string]any, createdAt time.Time) AdminAuditLog {
	return AdminAuditLog{
		ID:         uuid.NewString(),
		ActorID:    actorID.String(),
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Before:     before,
		After:      after,
		CreatedAt:  createdAt,
	}
}

func titleAuditState(title Title) map[string]any {
	return map[string]any{
		"id":          title.ID,
		"name":        title.Name,
		"scope_type":  title.ScopeType,
		"scope_id":    title.ScopeID,
		"is_active":   title.IsActive,
		"description": title.Description,
	}
}

func titleGrantAuditState(grant TitleGrant) map[string]any {
	return map[string]any{
		"id":         grant.ID,
		"user_id":    grant.UserID,
		"title_id":   grant.Title.ID,
		"title_name": grant.Title.Name,
		"expires_at": grant.ExpiresAt,
		"revoked_at": grant.RevokedAt,
		"reason":     grant.Reason,
	}
}

func RuneCount(value string) int {
	return utf8.RuneCountInString(value)
}

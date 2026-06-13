package adminusecase

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/platform/settings"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

const (
	DefaultAdminListLimit = 20
	MaxAdminListLimit     = 50
)

type UseCase struct {
	repository   Repository
	transactions TransactionManager
	now          func() time.Time
}

type Repository interface {
	IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error)
	ListUsers(ctx context.Context, status string, limit int, offset int) ([]User, error)
	FindUserByID(ctx context.Context, userID userdomain.UserID) (User, error)
	UpdateUser(ctx context.Context, userID userdomain.UserID, input UpdateUserRecordInput) (User, error)
	ListCommunities(ctx context.Context, status string, limit int, offset int) ([]Community, error)
	FindCommunityByID(ctx context.Context, communityID communitydomain.CommunityID) (Community, error)
	UpdateCommunityStatus(ctx context.Context, communityID communitydomain.CommunityID, status communitydomain.CommunityStatus, updatedAt time.Time) (Community, error)
	ListEffects(ctx context.Context, active *bool, limit int, offset int) ([]Effect, error)
	FindEffectByID(ctx context.Context, effectID string) (Effect, error)
	UpdateEffectActive(ctx context.Context, effectID string, active bool, updatedAt time.Time) (Effect, error)
	ListSettings(ctx context.Context) ([]Setting, error)
	FindSettingByKey(ctx context.Context, key string) (Setting, error)
	SetSetting(ctx context.Context, key string, enabled bool, updatedBy userdomain.UserID, updatedAt time.Time) (Setting, error)
	CreateAuditLog(ctx context.Context, log AuditLog) error
	ListAuditLogs(ctx context.Context, targetType string, targetID string, limit int, offset int) ([]AuditLog, error)
}

type TransactionManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, repository Repository) error) error
}

type UpdateUserRecordInput struct {
	Status          string
	IsPlatformStaff bool
	UpdatedAt       time.Time
}

type User struct {
	ID              string
	Username        string
	Status          string
	IsPlatformStaff bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Community struct {
	ID          string
	Slug        string
	Name        string
	Description string
	Kind        string
	Status      string
	Visibility  string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Effect struct {
	ID           string
	Name         string
	Description  string
	CostPoints   int
	AssetURL     string
	AnimationKey string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Setting struct {
	Key       string
	Enabled   bool
	UpdatedBy string
	UpdatedAt time.Time
}

type AuditLog struct {
	ID         string
	ActorID    string
	Action     string
	TargetType string
	TargetID   string
	Before     map[string]any
	After      map[string]any
	CreatedAt  time.Time
}

type ListUsersInput struct {
	ActorID userdomain.UserID
	Status  string
	Limit   int
	Offset  int
}

type ListUsersResult struct {
	Users  []User
	Status string
	Limit  int
	Offset int
}

type UpdateUserInput struct {
	ActorID         userdomain.UserID
	UserID          string
	Status          *string
	IsPlatformStaff *bool
}

type UpdateUserResult struct {
	User User
}

type ListCommunitiesInput struct {
	ActorID userdomain.UserID
	Status  string
	Limit   int
	Offset  int
}

type ListCommunitiesResult struct {
	Communities []Community
	Status      string
	Limit       int
	Offset      int
}

type UpdateCommunityStatusInput struct {
	ActorID     userdomain.UserID
	CommunityID string
	Status      string
}

type UpdateCommunityStatusResult struct {
	Community Community
}

type ListEffectsInput struct {
	ActorID userdomain.UserID
	Active  string
	Limit   int
	Offset  int
}

type ListEffectsResult struct {
	Effects []Effect
	Active  string
	Limit   int
	Offset  int
}

type UpdateEffectActiveInput struct {
	ActorID  userdomain.UserID
	EffectID string
	IsActive bool
}

type UpdateEffectActiveResult struct {
	Effect Effect
}

type ListSettingsInput struct {
	ActorID userdomain.UserID
}

type ListSettingsResult struct {
	Settings []Setting
}

type UpdateSettingInput struct {
	ActorID userdomain.UserID
	Key     string
	Enabled bool
}

type UpdateSettingResult struct {
	Setting Setting
}

type ListAuditLogsInput struct {
	ActorID    userdomain.UserID
	TargetType string
	TargetID   string
	Limit      int
	Offset     int
}

type ListAuditLogsResult struct {
	AuditLogs []AuditLog
	Limit     int
	Offset    int
}

func NewUseCase(repository Repository, now func() time.Time) *UseCase {
	if now == nil {
		now = time.Now
	}
	uc := &UseCase{
		repository: repository,
		now:        now,
	}
	if transactions, ok := repository.(TransactionManager); ok {
		uc.transactions = transactions
	}
	return uc
}

func (uc *UseCase) ListUsers(ctx context.Context, input ListUsersInput) (ListUsersResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return ListUsersResult{}, err
	}
	status, err := normalizeUserStatusFilter(input.Status)
	if err != nil {
		return ListUsersResult{}, err
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListUsersResult{}, err
	}
	users, err := uc.repository.ListUsers(ctx, status, limit, offset)
	if err != nil {
		return ListUsersResult{}, fmt.Errorf("list admin users: %w", err)
	}
	return ListUsersResult{Users: users, Status: status, Limit: limit, Offset: offset}, nil
}

func (uc *UseCase) UpdateUser(ctx context.Context, input UpdateUserInput) (UpdateUserResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return UpdateUserResult{}, err
	}
	targetID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return UpdateUserResult{}, err
	}
	if input.Status == nil && input.IsPlatformStaff == nil {
		return UpdateUserResult{}, apperr.New(apperr.CodeInvalidArgument, "admin user update is empty")
	}

	var requestedStatus string
	var hasStatus bool
	if input.Status != nil {
		status, err := normalizeUserStatus(*input.Status)
		if err != nil {
			return UpdateUserResult{}, err
		}
		requestedStatus = status
		hasStatus = true
	}

	var updated User
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		before, err := repository.FindUserByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("find admin user: %w", err)
		}
		status := before.Status
		if hasStatus {
			status = requestedStatus
		}
		isPlatformStaff := before.IsPlatformStaff
		if input.IsPlatformStaff != nil {
			isPlatformStaff = *input.IsPlatformStaff
		}
		updatedAt := uc.now().UTC()
		after, err := repository.UpdateUser(ctx, targetID, UpdateUserRecordInput{
			Status:          status,
			IsPlatformStaff: isPlatformStaff,
			UpdatedAt:       updatedAt,
		})
		if err != nil {
			return fmt.Errorf("update admin user: %w", err)
		}
		if err := repository.CreateAuditLog(ctx, newAuditLog(input.ActorID, "admin.users.update", "user", targetID.String(), userAuditState(before), userAuditState(after), updatedAt)); err != nil {
			return fmt.Errorf("create admin user audit log: %w", err)
		}
		updated = after
		return nil
	}); err != nil {
		return UpdateUserResult{}, err
	}

	return UpdateUserResult{User: updated}, nil
}

func (uc *UseCase) ListCommunities(ctx context.Context, input ListCommunitiesInput) (ListCommunitiesResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return ListCommunitiesResult{}, err
	}
	status, err := normalizeCommunityStatusFilter(input.Status)
	if err != nil {
		return ListCommunitiesResult{}, err
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListCommunitiesResult{}, err
	}
	communities, err := uc.repository.ListCommunities(ctx, status, limit, offset)
	if err != nil {
		return ListCommunitiesResult{}, fmt.Errorf("list admin communities: %w", err)
	}
	return ListCommunitiesResult{Communities: communities, Status: status, Limit: limit, Offset: offset}, nil
}

func (uc *UseCase) UpdateCommunityStatus(ctx context.Context, input UpdateCommunityStatusInput) (UpdateCommunityStatusResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return UpdateCommunityStatusResult{}, err
	}
	communityID, err := communitydomain.NewCommunityID(input.CommunityID)
	if err != nil {
		return UpdateCommunityStatusResult{}, err
	}
	status, err := communitydomain.NewCommunityStatus(strings.ToLower(strings.TrimSpace(input.Status)))
	if err != nil {
		return UpdateCommunityStatusResult{}, err
	}

	var updated Community
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		before, err := repository.FindCommunityByID(ctx, communityID)
		if err != nil {
			return fmt.Errorf("find admin community: %w", err)
		}
		updatedAt := uc.now().UTC()
		after, err := repository.UpdateCommunityStatus(ctx, communityID, status, updatedAt)
		if err != nil {
			return fmt.Errorf("update admin community status: %w", err)
		}
		if err := repository.CreateAuditLog(ctx, newAuditLog(input.ActorID, "admin.communities.update_status", "community", communityID.String(), communityAuditState(before), communityAuditState(after), updatedAt)); err != nil {
			return fmt.Errorf("create admin community audit log: %w", err)
		}
		updated = after
		return nil
	}); err != nil {
		return UpdateCommunityStatusResult{}, err
	}

	return UpdateCommunityStatusResult{Community: updated}, nil
}

func (uc *UseCase) ListEffects(ctx context.Context, input ListEffectsInput) (ListEffectsResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return ListEffectsResult{}, err
	}
	active, label, err := normalizeActiveFilter(input.Active)
	if err != nil {
		return ListEffectsResult{}, err
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListEffectsResult{}, err
	}
	effects, err := uc.repository.ListEffects(ctx, active, limit, offset)
	if err != nil {
		return ListEffectsResult{}, fmt.Errorf("list admin effects: %w", err)
	}
	return ListEffectsResult{Effects: effects, Active: label, Limit: limit, Offset: offset}, nil
}

func (uc *UseCase) UpdateEffectActive(ctx context.Context, input UpdateEffectActiveInput) (UpdateEffectActiveResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return UpdateEffectActiveResult{}, err
	}
	effectID, err := normalizeEffectID(input.EffectID)
	if err != nil {
		return UpdateEffectActiveResult{}, err
	}

	var updated Effect
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		before, err := repository.FindEffectByID(ctx, effectID)
		if err != nil {
			return fmt.Errorf("find admin effect: %w", err)
		}
		updatedAt := uc.now().UTC()
		after, err := repository.UpdateEffectActive(ctx, effectID, input.IsActive, updatedAt)
		if err != nil {
			return fmt.Errorf("update admin effect active state: %w", err)
		}
		if err := repository.CreateAuditLog(ctx, newAuditLog(input.ActorID, "admin.effects.update_active", "effect", effectID, effectAuditState(before), effectAuditState(after), updatedAt)); err != nil {
			return fmt.Errorf("create admin effect audit log: %w", err)
		}
		updated = after
		return nil
	}); err != nil {
		return UpdateEffectActiveResult{}, err
	}

	return UpdateEffectActiveResult{Effect: updated}, nil
}

func (uc *UseCase) ListSettings(ctx context.Context, input ListSettingsInput) (ListSettingsResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return ListSettingsResult{}, err
	}
	settingsRows, err := uc.repository.ListSettings(ctx)
	if err != nil {
		return ListSettingsResult{}, fmt.Errorf("list admin settings: %w", err)
	}
	return ListSettingsResult{Settings: settingsRows}, nil
}

func (uc *UseCase) UpdateSetting(ctx context.Context, input UpdateSettingInput) (UpdateSettingResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return UpdateSettingResult{}, err
	}
	key, err := settings.NormalizeKey(input.Key)
	if err != nil {
		return UpdateSettingResult{}, err
	}

	var updated Setting
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		before, err := repository.FindSettingByKey(ctx, key)
		if err != nil {
			return fmt.Errorf("find admin setting: %w", err)
		}
		updatedAt := uc.now().UTC()
		after, err := repository.SetSetting(ctx, key, input.Enabled, input.ActorID, updatedAt)
		if err != nil {
			return fmt.Errorf("update admin setting: %w", err)
		}
		if err := repository.CreateAuditLog(ctx, newAuditLog(input.ActorID, "admin.settings.update", "setting", key, settingAuditState(before), settingAuditState(after), updatedAt)); err != nil {
			return fmt.Errorf("create admin setting audit log: %w", err)
		}
		updated = after
		return nil
	}); err != nil {
		return UpdateSettingResult{}, err
	}

	return UpdateSettingResult{Setting: updated}, nil
}

func (uc *UseCase) ListAuditLogs(ctx context.Context, input ListAuditLogsInput) (ListAuditLogsResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return ListAuditLogsResult{}, err
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListAuditLogsResult{}, err
	}
	auditLogs, err := uc.repository.ListAuditLogs(ctx, strings.TrimSpace(input.TargetType), strings.TrimSpace(input.TargetID), limit, offset)
	if err != nil {
		return ListAuditLogsResult{}, fmt.Errorf("list admin audit logs: %w", err)
	}
	return ListAuditLogsResult{AuditLogs: auditLogs, Limit: limit, Offset: offset}, nil
}

func (uc *UseCase) ensurePlatformStaff(ctx context.Context, actorID userdomain.UserID) error {
	if strings.TrimSpace(actorID.String()) == "" {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
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
	if uc.transactions != nil {
		return uc.transactions.WithinTx(ctx, fn)
	}
	return apperr.New(apperr.CodeInternal, "admin transaction support is not configured")
}

func normalizePagination(limit int, offset int) (int, int, error) {
	if limit < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "limit must be non-negative")
	}
	if offset < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "offset must be non-negative")
	}
	if limit == 0 {
		limit = DefaultAdminListLimit
	}
	if limit > MaxAdminListLimit {
		limit = MaxAdminListLimit
	}
	return limit, offset, nil
}

func normalizeUserStatusFilter(raw string) (string, error) {
	status := strings.ToLower(strings.TrimSpace(raw))
	if status == "" || status == "all" {
		return "all", nil
	}
	return normalizeUserStatus(status)
}

func normalizeUserStatus(raw string) (string, error) {
	status := strings.ToLower(strings.TrimSpace(raw))
	if _, err := userdomain.NewUserStatus(status); err != nil {
		return "", err
	}
	return status, nil
}

func normalizeCommunityStatusFilter(raw string) (string, error) {
	status := strings.ToLower(strings.TrimSpace(raw))
	if status == "" || status == "all" {
		return "all", nil
	}
	if _, err := communitydomain.NewCommunityStatus(status); err != nil {
		return "", err
	}
	return status, nil
}

func normalizeActiveFilter(raw string) (*bool, string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" || value == "all" {
		return nil, "all", nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, "", apperr.New(apperr.CodeInvalidArgument, "active query is invalid")
	}
	return &parsed, strconv.FormatBool(parsed), nil
}

func normalizeEffectID(raw string) (string, error) {
	effectID := strings.ToLower(strings.TrimSpace(raw))
	if effectID == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "effect id is required")
	}
	for index, char := range effectID {
		if index == 0 && !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
			return "", apperr.New(apperr.CodeInvalidArgument, "effect id is invalid")
		}
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' || char == '-' {
			continue
		}
		return "", apperr.New(apperr.CodeInvalidArgument, "effect id is invalid")
	}
	if len(effectID) > 64 {
		return "", apperr.New(apperr.CodeInvalidArgument, "effect id is invalid")
	}
	return effectID, nil
}

func newAuditLog(actorID userdomain.UserID, action string, targetType string, targetID string, before map[string]any, after map[string]any, createdAt time.Time) AuditLog {
	return AuditLog{
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

func userAuditState(user User) map[string]any {
	return map[string]any{
		"id":                user.ID,
		"username":          user.Username,
		"status":            user.Status,
		"is_platform_staff": user.IsPlatformStaff,
	}
}

func communityAuditState(community Community) map[string]any {
	return map[string]any{
		"id":     community.ID,
		"slug":   community.Slug,
		"status": community.Status,
	}
}

func effectAuditState(effect Effect) map[string]any {
	return map[string]any{
		"id":        effect.ID,
		"is_active": effect.IsActive,
	}
}

func settingAuditState(setting Setting) map[string]any {
	return map[string]any{
		"key":     setting.Key,
		"enabled": setting.Enabled,
	}
}

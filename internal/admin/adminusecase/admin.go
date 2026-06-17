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
	"github.com/Versifine/cumt-nexus-api/internal/textlimit"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

const (
	DefaultAdminListLimit               = 20
	MaxAdminListLimit                   = 50
	MaxPointAdjustReasonRunes           = 500
	MaxAdminSearchQueryRunes            = 80
	MaxUserSanctionReasonRunes          = 500
	MaxOwnerTransferReasonRunes         = 500
	PlatformOwnerTransferTTL            = 48 * time.Hour
	PlatformRoleOwner                   = "owner"
	PlatformRoleAdmin                   = "admin"
	PlatformRoleStaff                   = "staff"
	OwnerTransferStatusPending          = "pending"
	OwnerTransferStatusAccepted         = "accepted"
	OwnerTransferStatusCancelled        = "cancelled"
	OwnerTransferStatusExpired          = "expired"
	UserSanctionTypeAccountBan          = "account_ban"
	UserSanctionStatusActive            = "active"
	UserSanctionStatusRevoked           = "revoked"
	UserSanctionStatusExpired           = "expired"
	platformOwnerTransferRequiredReason = "platform owner changes require owner transfer or recovery"
)

type UseCase struct {
	repository                 Repository
	transactions               TransactionManager
	passwordComparer           PasswordComparer
	ownerTransferNotifications OwnerTransferNotificationPublisher
	now                        func() time.Time
}

type Repository interface {
	IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error)
	ListUsers(ctx context.Context, status string, query string, limit int, offset int) ([]User, error)
	FindUserByID(ctx context.Context, userID userdomain.UserID) (User, error)
	FindUserPasswordHash(ctx context.Context, userID userdomain.UserID) (userdomain.PasswordHash, error)
	UpdateUser(ctx context.Context, userID userdomain.UserID, input UpdateUserRecordInput) (User, error)
	UpdateUserPlatformRole(ctx context.Context, userID userdomain.UserID, role string, updatedAt time.Time) (User, error)
	CountPlatformOwners(ctx context.Context) (int, error)
	FindCurrentOwnerTransfer(ctx context.Context, now time.Time) (OwnerTransfer, error)
	FindOwnerTransferByID(ctx context.Context, transferID string, now time.Time) (OwnerTransfer, error)
	ListOwnerTransfersByTarget(ctx context.Context, targetUserID userdomain.UserID, status string, now time.Time, limit int, offset int) ([]OwnerTransfer, error)
	CreateOwnerTransfer(ctx context.Context, input CreateOwnerTransferRecordInput) (OwnerTransfer, error)
	CancelOwnerTransfer(ctx context.Context, transferID string, cancelledAt time.Time) (OwnerTransfer, error)
	AcceptOwnerTransfer(ctx context.Context, transferID string, acceptedAt time.Time) (OwnerTransfer, error)
	BootstrapOwner(ctx context.Context, input BootstrapOwnerRecordInput) (User, error)
	RecoverOwner(ctx context.Context, input RecoverOwnerRecordInput) (OwnerRecoveryRecordResult, error)
	ListCommunities(ctx context.Context, status string, query string, limit int, offset int) ([]Community, error)
	FindCommunityByID(ctx context.Context, communityID communitydomain.CommunityID) (Community, error)
	UpdateCommunityStatus(ctx context.Context, communityID communitydomain.CommunityID, status communitydomain.CommunityStatus, updatedAt time.Time) (Community, error)
	TransferCommunityOwner(ctx context.Context, communityID communitydomain.CommunityID, newOwnerID userdomain.UserID, updatedAt time.Time) (CommunityOwnerChange, error)
	ListEffects(ctx context.Context, active *bool, limit int, offset int) ([]Effect, error)
	FindEffectByID(ctx context.Context, effectID string) (Effect, error)
	UpdateEffectActive(ctx context.Context, effectID string, active bool, updatedAt time.Time) (Effect, error)
	ListSettings(ctx context.Context) ([]Setting, error)
	FindSettingByKey(ctx context.Context, key string) (Setting, error)
	SetSetting(ctx context.Context, key string, enabled bool, updatedBy userdomain.UserID, updatedAt time.Time) (Setting, error)
	ListPointTransactions(ctx context.Context, userID *userdomain.UserID, limit int, offset int) ([]PointTransaction, error)
	AdjustUserPoints(ctx context.Context, input AdjustUserPointsRecordInput) (AdjustUserPointsRecordResult, error)
	CreateUserSanction(ctx context.Context, input CreateUserSanctionRecordInput) (UserSanction, error)
	ListUserSanctions(ctx context.Context, userID userdomain.UserID, limit int, offset int, now time.Time) ([]UserSanction, error)
	FindUserSanctionByID(ctx context.Context, sanctionID string, now time.Time) (UserSanction, error)
	RevokeUserSanction(ctx context.Context, sanctionID string, actorID userdomain.UserID, revokedAt time.Time) (UserSanction, error)
	CreateAuditLog(ctx context.Context, log AuditLog) error
	ListAuditLogs(ctx context.Context, targetType string, targetID string, query string, limit int, offset int) ([]AuditLog, error)
}

type TransactionManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, repository Repository) error) error
}

type PasswordComparer interface {
	Compare(hash userdomain.PasswordHash, plain userdomain.PlainPassword) error
}

type OwnerTransferNotificationPublisher interface {
	NotifyPlatformOwnerTransfer(ctx context.Context, recipientID userdomain.UserID, actorID userdomain.UserID, transferID string) error
}

type UpdateUserRecordInput struct {
	Status          string
	IsPlatformStaff bool
	UpdatedAt       time.Time
}

type User struct {
	ID              string
	Username        string
	DisplayName     string
	AvatarURL       string
	Headline        string
	Status          string
	IsPlatformStaff bool
	PlatformRole    string
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

type CommunityOwnerMember struct {
	UserID    string
	Username  string
	Role      string
	Status    string
	UpdatedAt time.Time
}

type CommunityOwnerChange struct {
	BeforeOwner *CommunityOwnerMember
	AfterOwner  CommunityOwnerMember
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

type PointAccount struct {
	UserID         string
	Balance        int
	LifetimeEarned int
	LifetimeSpent  int
	UpdatedAt      time.Time
}

type PointTransaction struct {
	ID           string
	UserID       string
	Delta        int
	BalanceAfter int
	Reason       string
	SourceType   string
	SourceID     string
	CreatedAt    time.Time
}

type AdjustUserPointsRecordInput struct {
	TransactionID string
	UserID        userdomain.UserID
	ActorID       userdomain.UserID
	Delta         int
	Reason        string
	CreatedAt     time.Time
}

type AdjustUserPointsRecordResult struct {
	Account     PointAccount
	Transaction PointTransaction
}

type UserSanction struct {
	ID        string
	UserID    string
	Type      string
	Status    string
	Reason    string
	CreatedBy string
	StartsAt  time.Time
	ExpiresAt *time.Time
	RevokedBy string
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OwnerTransfer struct {
	ID                     string
	Status                 string
	InitiatedByID          string
	InitiatedByUsername    string
	InitiatedByDisplayName string
	TargetUserID           string
	TargetUsername         string
	TargetDisplayName      string
	PreviousOwnerRole      string
	Reason                 string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	ExpiresAt              time.Time
	AcceptedAt             *time.Time
	CancelledAt            *time.Time
}

type CreateUserSanctionRecordInput struct {
	ID        string
	UserID    userdomain.UserID
	Type      string
	Reason    string
	CreatedBy userdomain.UserID
	StartsAt  time.Time
	ExpiresAt *time.Time
	CreatedAt time.Time
}

type CreateOwnerTransferRecordInput struct {
	ID                string
	InitiatedByID     userdomain.UserID
	TargetUserID      userdomain.UserID
	PreviousOwnerRole string
	Reason            string
	CreatedAt         time.Time
	ExpiresAt         time.Time
}

type BootstrapOwnerRecordInput struct {
	UserID    userdomain.UserID
	UpdatedAt time.Time
}

type RecoverOwnerRecordInput struct {
	NewOwnerID         userdomain.UserID
	CompromisedUserID  userdomain.UserID
	UpdatedAt          time.Time
	RevokeSessions     bool
	DisableCompromised bool
}

type OwnerRecoveryRecordResult struct {
	NewOwner        User
	CompromisedUser User
	PreviousOwners  []User
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
	Query   string
	Limit   int
	Offset  int
}

type ListUsersResult struct {
	Users      []User
	Status     string
	Query      string
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
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

type UpdateUserPlatformRoleInput struct {
	ActorID userdomain.UserID
	UserID  string
	Role    *string
}

type UpdateUserPlatformRoleResult struct {
	User User
}

type GetCurrentOwnerTransferInput struct {
	ActorID userdomain.UserID
}

type GetCurrentOwnerTransferResult struct {
	Transfer *OwnerTransfer
}

type CreateOwnerTransferInput struct {
	ActorID           userdomain.UserID
	TargetUserID      string
	PreviousOwnerRole *string
	Reason            string
	CurrentPassword   string
}

type CreateOwnerTransferResult struct {
	Transfer OwnerTransfer
}

type CancelOwnerTransferInput struct {
	ActorID    userdomain.UserID
	TransferID string
}

type CancelOwnerTransferResult struct {
	Transfer OwnerTransfer
}

type GetOwnerTransferInput struct {
	ActorID    userdomain.UserID
	TransferID string
}

type GetOwnerTransferResult struct {
	Transfer        OwnerTransfer
	ViewerIsTarget  bool
	ViewerCanAccept bool
}

type AcceptOwnerTransferInput struct {
	ActorID         userdomain.UserID
	TransferID      string
	CurrentPassword string
}

type AcceptOwnerTransferResult struct {
	Transfer OwnerTransfer
}

type ListOwnerTransfersInput struct {
	ActorID userdomain.UserID
	Status  string
	Limit   int
	Offset  int
}

type ListOwnerTransfersResult struct {
	Transfers  []OwnerTransfer
	Status     string
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type BootstrapOwnerInput struct {
	UserID  string
	Reason  string
	Confirm bool
}

type BootstrapOwnerResult struct {
	User User
}

type RecoverOwnerInput struct {
	NewOwnerUserID     string
	CompromisedUserID  string
	Reason             string
	RevokeSessions     bool
	DisableCompromised bool
	Confirm            bool
}

type RecoverOwnerResult struct {
	NewOwner        User
	CompromisedUser User
}

type ListCommunitiesInput struct {
	ActorID userdomain.UserID
	Status  string
	Query   string
	Limit   int
	Offset  int
}

type ListCommunitiesResult struct {
	Communities []Community
	Status      string
	Query       string
	Limit       int
	Offset      int
	NextOffset  int
	HasMore     bool
}

type UpdateCommunityStatusInput struct {
	ActorID     userdomain.UserID
	CommunityID string
	Status      string
}

type UpdateCommunityStatusResult struct {
	Community Community
}

type UpdateCommunityOwnerInput struct {
	ActorID     userdomain.UserID
	CommunityID string
	UserID      string
	Reason      string
}

type UpdateCommunityOwnerResult struct {
	Community Community
	Owner     CommunityOwnerMember
}

type ListEffectsInput struct {
	ActorID userdomain.UserID
	Active  string
	Limit   int
	Offset  int
}

type ListEffectsResult struct {
	Effects    []Effect
	Active     string
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
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
	Query      string
	Limit      int
	Offset     int
}

type ListAuditLogsResult struct {
	AuditLogs  []AuditLog
	Query      string
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type ListPointTransactionsInput struct {
	ActorID userdomain.UserID
	UserID  string
	Limit   int
	Offset  int
}

type ListPointTransactionsResult struct {
	Transactions []PointTransaction
	Limit        int
	Offset       int
	NextOffset   int
	HasMore      bool
}

type AdjustUserPointsInput struct {
	ActorID userdomain.UserID
	UserID  string
	Delta   int
	Reason  string
}

type AdjustUserPointsResult struct {
	Account     PointAccount
	Transaction PointTransaction
}

type CreateUserSanctionInput struct {
	ActorID  userdomain.UserID
	UserID   string
	Type     string
	Duration string
	Reason   string
}

type CreateUserSanctionResult struct {
	Sanction UserSanction
}

type ListUserSanctionsInput struct {
	ActorID userdomain.UserID
	UserID  string
	Limit   int
	Offset  int
}

type ListUserSanctionsResult struct {
	Sanctions  []UserSanction
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type RevokeUserSanctionInput struct {
	ActorID    userdomain.UserID
	SanctionID string
}

type RevokeUserSanctionResult struct {
	Sanction UserSanction
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

func (uc *UseCase) SetPasswordComparer(comparer PasswordComparer) {
	uc.passwordComparer = comparer
}

func (uc *UseCase) SetOwnerTransferNotificationPublisher(publisher OwnerTransferNotificationPublisher) {
	uc.ownerTransferNotifications = publisher
}

func (uc *UseCase) ListUsers(ctx context.Context, input ListUsersInput) (ListUsersResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return ListUsersResult{}, err
	}
	status, err := normalizeUserStatusFilter(input.Status)
	if err != nil {
		return ListUsersResult{}, err
	}
	query, err := normalizeAdminSearchQuery(input.Query)
	if err != nil {
		return ListUsersResult{}, err
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListUsersResult{}, err
	}
	users, err := uc.repository.ListUsers(ctx, status, query, limit+1, offset)
	if err != nil {
		return ListUsersResult{}, fmt.Errorf("list admin users: %w", err)
	}
	users, hasMore := trimPage(users, limit)
	return ListUsersResult{Users: users, Status: status, Query: query, Limit: limit, Offset: offset, NextOffset: offset + len(users), HasMore: hasMore}, nil
}

func (uc *UseCase) UpdateUser(ctx context.Context, input UpdateUserInput) (UpdateUserResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return UpdateUserResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
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
		actor, err := repository.FindUserByID(ctx, input.ActorID)
		if err != nil {
			return fmt.Errorf("find admin user actor: %w", err)
		}
		before, err := repository.FindUserByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("find admin user: %w", err)
		}
		if err := authorizeUserWrite(actor, before); err != nil {
			return err
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

func (uc *UseCase) UpdateUserPlatformRole(ctx context.Context, input UpdateUserPlatformRoleInput) (UpdateUserPlatformRoleResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return UpdateUserPlatformRoleResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	targetID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return UpdateUserPlatformRoleResult{}, err
	}
	requestedRole, err := normalizePlatformRole(input.Role)
	if err != nil {
		return UpdateUserPlatformRoleResult{}, err
	}
	if requestedRole == PlatformRoleOwner {
		return UpdateUserPlatformRoleResult{}, apperr.New(apperr.CodeForbidden, platformOwnerTransferRequiredReason)
	}

	var updated User
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		actor, err := repository.FindUserByID(ctx, input.ActorID)
		if err != nil {
			return fmt.Errorf("find platform role actor: %w", err)
		}
		actorRole := effectivePlatformRole(actor)
		if actor.Status != "active" || actorRole == "" {
			return apperr.New(apperr.CodeForbidden, "platform staff required")
		}
		before, err := repository.FindUserByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("find platform role user: %w", err)
		}
		targetRole := effectivePlatformRole(before)
		if targetRole == PlatformRoleOwner {
			return apperr.New(apperr.CodeForbidden, platformOwnerTransferRequiredReason)
		}
		if err := authorizePlatformRoleChange(actorRole, targetRole, requestedRole); err != nil {
			return err
		}
		updatedAt := uc.now().UTC()
		after, err := repository.UpdateUserPlatformRole(ctx, targetID, requestedRole, updatedAt)
		if err != nil {
			return fmt.Errorf("update platform role: %w", err)
		}
		if err := repository.CreateAuditLog(ctx, newAuditLog(input.ActorID, "admin.users.update_platform_role", "user", targetID.String(), userAuditState(before), userAuditState(after), updatedAt)); err != nil {
			return fmt.Errorf("create platform role audit log: %w", err)
		}
		updated = after
		return nil
	}); err != nil {
		return UpdateUserPlatformRoleResult{}, err
	}

	return UpdateUserPlatformRoleResult{User: updated}, nil
}

func (uc *UseCase) ListCommunities(ctx context.Context, input ListCommunitiesInput) (ListCommunitiesResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return ListCommunitiesResult{}, err
	}
	status, err := normalizeCommunityStatusFilter(input.Status)
	if err != nil {
		return ListCommunitiesResult{}, err
	}
	query, err := normalizeAdminSearchQuery(input.Query)
	if err != nil {
		return ListCommunitiesResult{}, err
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListCommunitiesResult{}, err
	}
	communities, err := uc.repository.ListCommunities(ctx, status, query, limit+1, offset)
	if err != nil {
		return ListCommunitiesResult{}, fmt.Errorf("list admin communities: %w", err)
	}
	communities, hasMore := trimPage(communities, limit)
	return ListCommunitiesResult{Communities: communities, Status: status, Query: query, Limit: limit, Offset: offset, NextOffset: offset + len(communities), HasMore: hasMore}, nil
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

func (uc *UseCase) UpdateCommunityOwner(ctx context.Context, input UpdateCommunityOwnerInput) (UpdateCommunityOwnerResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return UpdateCommunityOwnerResult{}, err
	}
	communityID, err := communitydomain.NewCommunityID(input.CommunityID)
	if err != nil {
		return UpdateCommunityOwnerResult{}, err
	}
	newOwnerID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return UpdateCommunityOwnerResult{}, err
	}
	reason, err := textlimit.TrimmedOptionalMaxRunes(input.Reason, "community owner transfer reason", MaxOwnerTransferReasonRunes)
	if err != nil {
		return UpdateCommunityOwnerResult{}, err
	}

	var community Community
	var owner CommunityOwnerMember
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		currentCommunity, err := repository.FindCommunityByID(ctx, communityID)
		if err != nil {
			return fmt.Errorf("find admin community: %w", err)
		}
		updatedAt := uc.now().UTC()
		change, err := repository.TransferCommunityOwner(ctx, communityID, newOwnerID, updatedAt)
		if err != nil {
			return fmt.Errorf("transfer admin community owner: %w", err)
		}
		beforeState := map[string]any{"owner": nil}
		if change.BeforeOwner != nil {
			beforeState["owner"] = communityOwnerAuditState(*change.BeforeOwner)
		}
		afterState := communityOwnerAuditState(change.AfterOwner)
		if reason != "" {
			afterState["reason"] = reason
		}
		if err := repository.CreateAuditLog(ctx, newAuditLog(input.ActorID, "admin.communities.update_owner", "community", communityID.String(), beforeState, afterState, updatedAt)); err != nil {
			return fmt.Errorf("create admin community owner audit log: %w", err)
		}
		community = currentCommunity
		owner = change.AfterOwner
		return nil
	}); err != nil {
		return UpdateCommunityOwnerResult{}, err
	}
	return UpdateCommunityOwnerResult{Community: community, Owner: owner}, nil
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
	effects, err := uc.repository.ListEffects(ctx, active, limit+1, offset)
	if err != nil {
		return ListEffectsResult{}, fmt.Errorf("list admin effects: %w", err)
	}
	effects, hasMore := trimPage(effects, limit)
	return ListEffectsResult{Effects: effects, Active: label, Limit: limit, Offset: offset, NextOffset: offset + len(effects), HasMore: hasMore}, nil
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
	query, err := normalizeAdminSearchQuery(input.Query)
	if err != nil {
		return ListAuditLogsResult{}, err
	}
	auditLogs, err := uc.repository.ListAuditLogs(ctx, strings.TrimSpace(input.TargetType), strings.TrimSpace(input.TargetID), query, limit+1, offset)
	if err != nil {
		return ListAuditLogsResult{}, fmt.Errorf("list admin audit logs: %w", err)
	}
	auditLogs, hasMore := trimPage(auditLogs, limit)
	return ListAuditLogsResult{AuditLogs: auditLogs, Query: query, Limit: limit, Offset: offset, NextOffset: offset + len(auditLogs), HasMore: hasMore}, nil
}

func (uc *UseCase) ListPointTransactions(ctx context.Context, input ListPointTransactionsInput) (ListPointTransactionsResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return ListPointTransactionsResult{}, err
	}
	var targetID *userdomain.UserID
	if strings.TrimSpace(input.UserID) != "" {
		parsed, err := userdomain.NewUserID(input.UserID)
		if err != nil {
			return ListPointTransactionsResult{}, err
		}
		targetID = &parsed
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListPointTransactionsResult{}, err
	}
	transactions, err := uc.repository.ListPointTransactions(ctx, targetID, limit+1, offset)
	if err != nil {
		return ListPointTransactionsResult{}, fmt.Errorf("list admin point transactions: %w", err)
	}
	transactions, hasMore := trimPage(transactions, limit)
	return ListPointTransactionsResult{Transactions: transactions, Limit: limit, Offset: offset, NextOffset: offset + len(transactions), HasMore: hasMore}, nil
}

func (uc *UseCase) AdjustUserPoints(ctx context.Context, input AdjustUserPointsInput) (AdjustUserPointsResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return AdjustUserPointsResult{}, err
	}
	targetID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return AdjustUserPointsResult{}, err
	}
	if input.Delta == 0 {
		return AdjustUserPointsResult{}, apperr.New(apperr.CodeInvalidArgument, "point adjustment delta must not be zero")
	}
	reason, err := textlimit.TrimmedRequiredMaxRunes(input.Reason, "point adjustment reason", MaxPointAdjustReasonRunes)
	if err != nil {
		return AdjustUserPointsResult{}, err
	}

	var result AdjustUserPointsRecordResult
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		user, err := repository.FindUserByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("find point adjustment user: %w", err)
		}
		if user.Status == "deleted" {
			return apperr.New(apperr.CodeNotFound, "user not found")
		}
		createdAt := uc.now().UTC()
		adjusted, err := repository.AdjustUserPoints(ctx, AdjustUserPointsRecordInput{
			TransactionID: uuid.NewString(),
			UserID:        targetID,
			ActorID:       input.ActorID,
			Delta:         input.Delta,
			Reason:        reason,
			CreatedAt:     createdAt,
		})
		if err != nil {
			return fmt.Errorf("adjust user points: %w", err)
		}
		if err := repository.CreateAuditLog(ctx, newAuditLog(input.ActorID, "admin.points.adjust", "user", targetID.String(), map[string]any{
			"id":      targetID.String(),
			"balance": adjusted.Transaction.BalanceAfter - input.Delta,
		}, pointAdjustmentAuditState(adjusted), createdAt)); err != nil {
			return fmt.Errorf("create point adjustment audit log: %w", err)
		}
		result = adjusted
		return nil
	}); err != nil {
		return AdjustUserPointsResult{}, err
	}

	return AdjustUserPointsResult{Account: result.Account, Transaction: result.Transaction}, nil
}

func (uc *UseCase) CreateUserSanction(ctx context.Context, input CreateUserSanctionInput) (CreateUserSanctionResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return CreateUserSanctionResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	targetID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return CreateUserSanctionResult{}, err
	}
	sanctionType, err := normalizeUserSanctionType(input.Type)
	if err != nil {
		return CreateUserSanctionResult{}, err
	}
	reason, err := textlimit.TrimmedRequiredMaxRunes(input.Reason, "user sanction reason", MaxUserSanctionReasonRunes)
	if err != nil {
		return CreateUserSanctionResult{}, err
	}
	var duration *time.Duration
	if sanctionType == UserSanctionTypeAccountBan {
		duration, err = normalizeUserSanctionDuration(input.Duration)
		if err != nil {
			return CreateUserSanctionResult{}, err
		}
	}

	var sanction UserSanction
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		actor, err := repository.FindUserByID(ctx, input.ActorID)
		if err != nil {
			return fmt.Errorf("find sanction actor: %w", err)
		}
		if actor.Status != "active" {
			return apperr.New(apperr.CodeForbidden, "platform admin required")
		}
		target, err := repository.FindUserByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("find sanction user: %w", err)
		}
		if target.Status == "deleted" {
			return apperr.New(apperr.CodeNotFound, "user not found")
		}
		if err := authorizeUserSanction(actor, target); err != nil {
			return err
		}
		startsAt := uc.now().UTC()
		var expiresAt *time.Time
		if duration != nil {
			value := startsAt.Add(*duration)
			expiresAt = &value
		}
		created, err := repository.CreateUserSanction(ctx, CreateUserSanctionRecordInput{
			ID:        uuid.NewString(),
			UserID:    targetID,
			Type:      sanctionType,
			Reason:    reason,
			CreatedBy: input.ActorID,
			StartsAt:  startsAt,
			ExpiresAt: expiresAt,
			CreatedAt: startsAt,
		})
		if err != nil {
			return fmt.Errorf("create user sanction: %w", err)
		}
		if err := repository.CreateAuditLog(ctx, newAuditLog(input.ActorID, "admin.users.create_sanction", "user", targetID.String(), map[string]any{}, userSanctionAuditState(created), startsAt)); err != nil {
			return fmt.Errorf("create user sanction audit log: %w", err)
		}
		sanction = created
		return nil
	}); err != nil {
		return CreateUserSanctionResult{}, err
	}

	return CreateUserSanctionResult{Sanction: sanction}, nil
}

func (uc *UseCase) ListUserSanctions(ctx context.Context, input ListUserSanctionsInput) (ListUserSanctionsResult, error) {
	if err := uc.ensurePlatformStaff(ctx, input.ActorID); err != nil {
		return ListUserSanctionsResult{}, err
	}
	targetID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return ListUserSanctionsResult{}, err
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListUserSanctionsResult{}, err
	}
	if _, err := uc.repository.FindUserByID(ctx, targetID); err != nil {
		return ListUserSanctionsResult{}, fmt.Errorf("find sanction list user: %w", err)
	}
	sanctions, err := uc.repository.ListUserSanctions(ctx, targetID, limit+1, offset, uc.now().UTC())
	if err != nil {
		return ListUserSanctionsResult{}, fmt.Errorf("list user sanctions: %w", err)
	}
	sanctions, hasMore := trimPage(sanctions, limit)
	return ListUserSanctionsResult{Sanctions: sanctions, Limit: limit, Offset: offset, NextOffset: offset + len(sanctions), HasMore: hasMore}, nil
}

func (uc *UseCase) RevokeUserSanction(ctx context.Context, input RevokeUserSanctionInput) (RevokeUserSanctionResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return RevokeUserSanctionResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	sanctionID, err := normalizeUserSanctionID(input.SanctionID)
	if err != nil {
		return RevokeUserSanctionResult{}, err
	}
	var sanction UserSanction
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		now := uc.now().UTC()
		actor, err := repository.FindUserByID(ctx, input.ActorID)
		if err != nil {
			return fmt.Errorf("find sanction revoke actor: %w", err)
		}
		if actor.Status != "active" {
			return apperr.New(apperr.CodeForbidden, "platform admin required")
		}
		before, err := repository.FindUserSanctionByID(ctx, sanctionID, now)
		if err != nil {
			return fmt.Errorf("find user sanction: %w", err)
		}
		if before.Status != UserSanctionStatusActive {
			return apperr.New(apperr.CodeConflict, "user sanction is not active")
		}
		targetID, err := userdomain.NewUserID(before.UserID)
		if err != nil {
			return err
		}
		target, err := repository.FindUserByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("find sanctioned user: %w", err)
		}
		if err := authorizeUserSanction(actor, target); err != nil {
			return err
		}
		after, err := repository.RevokeUserSanction(ctx, sanctionID, input.ActorID, now)
		if err != nil {
			return fmt.Errorf("revoke user sanction: %w", err)
		}
		if err := repository.CreateAuditLog(ctx, newAuditLog(input.ActorID, "admin.users.revoke_sanction", "user", targetID.String(), userSanctionAuditState(before), userSanctionAuditState(after), now)); err != nil {
			return fmt.Errorf("create user sanction revoke audit log: %w", err)
		}
		sanction = after
		return nil
	}); err != nil {
		return RevokeUserSanctionResult{}, err
	}
	return RevokeUserSanctionResult{Sanction: sanction}, nil
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

func normalizePlatformRole(role *string) (string, error) {
	if role == nil {
		return "", nil
	}
	value := strings.ToLower(strings.TrimSpace(*role))
	switch value {
	case "", PlatformRoleOwner, PlatformRoleAdmin, PlatformRoleStaff:
		return value, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "invalid platform role")
	}
}

func effectivePlatformRole(user User) string {
	role, err := normalizePlatformRole(&user.PlatformRole)
	if err == nil && role != "" {
		return role
	}
	if user.IsPlatformStaff {
		return PlatformRoleStaff
	}
	return ""
}

func authorizeUserWrite(actor User, target User) error {
	actorRole := effectivePlatformRole(actor)
	targetRole := effectivePlatformRole(target)
	if actor.Status != "active" || actorRole == "" {
		return apperr.New(apperr.CodeForbidden, "platform staff required")
	}
	if targetRole == PlatformRoleOwner {
		return apperr.New(apperr.CodeForbidden, platformOwnerTransferRequiredReason)
	}
	switch actorRole {
	case PlatformRoleOwner:
		return nil
	case PlatformRoleAdmin:
		if targetRole != "" {
			return apperr.New(apperr.CodeForbidden, "platform owner required")
		}
		return nil
	default:
		return apperr.New(apperr.CodeForbidden, "platform owner required")
	}
}

func authorizePlatformRoleChange(actorRole string, targetCurrentRole string, requestedRole string) error {
	switch actorRole {
	case PlatformRoleOwner:
		return nil
	default:
		return apperr.New(apperr.CodeForbidden, "platform owner required")
	}
}

func authorizeUserSanction(actor User, target User) error {
	if strings.TrimSpace(actor.ID) == "" {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if actor.ID == target.ID {
		return apperr.New(apperr.CodeForbidden, "cannot sanction yourself")
	}
	actorRole := effectivePlatformRole(actor)
	targetRole := effectivePlatformRole(target)
	switch actorRole {
	case PlatformRoleOwner:
		if targetRole == PlatformRoleOwner {
			return apperr.New(apperr.CodeForbidden, "cannot sanction platform owner")
		}
		return nil
	case PlatformRoleAdmin:
		if targetRole != "" {
			return apperr.New(apperr.CodeForbidden, "platform owner required")
		}
		return nil
	default:
		return apperr.New(apperr.CodeForbidden, "platform admin required")
	}
}

func normalizeUserSanctionType(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		value = UserSanctionTypeAccountBan
	}
	if value != UserSanctionTypeAccountBan {
		return "", apperr.New(apperr.CodeInvalidArgument, "invalid user sanction type")
	}
	return value, nil
}

func normalizeUserSanctionDuration(raw string) (*time.Duration, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1d":
		value := 24 * time.Hour
		return &value, nil
	case "3d":
		value := 3 * 24 * time.Hour
		return &value, nil
	case "7d":
		value := 7 * 24 * time.Hour
		return &value, nil
	case "30d":
		value := 30 * 24 * time.Hour
		return &value, nil
	case "permanent":
		return nil, nil
	default:
		return nil, apperr.New(apperr.CodeInvalidArgument, "invalid user sanction duration")
	}
}

func normalizeUserSanctionID(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "user sanction id is required")
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return "", apperr.New(apperr.CodeInvalidArgument, "user sanction id is invalid")
	}
	return parsed.String(), nil
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

func normalizeAdminSearchQuery(raw string) (string, error) {
	return textlimit.TrimmedOptionalMaxRunes(raw, "admin search query", MaxAdminSearchQueryRunes)
}

func trimPage[T any](items []T, limit int) ([]T, bool) {
	if len(items) <= limit {
		return items, false
	}
	return items[:limit], true
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
		"display_name":      user.DisplayName,
		"avatar_url":        user.AvatarURL,
		"headline":          user.Headline,
		"status":            user.Status,
		"is_platform_staff": user.IsPlatformStaff,
		"platform_role":     user.PlatformRole,
	}
}

func communityAuditState(community Community) map[string]any {
	return map[string]any{
		"id":     community.ID,
		"slug":   community.Slug,
		"status": community.Status,
	}
}

func communityOwnerAuditState(owner CommunityOwnerMember) map[string]any {
	return map[string]any{
		"user_id":  owner.UserID,
		"username": owner.Username,
		"role":     owner.Role,
		"status":   owner.Status,
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

func pointAdjustmentAuditState(result AdjustUserPointsRecordResult) map[string]any {
	return map[string]any{
		"id":             result.Account.UserID,
		"balance":        result.Account.Balance,
		"transaction_id": result.Transaction.ID,
		"delta":          result.Transaction.Delta,
		"reason":         result.Transaction.Reason,
	}
}

func userSanctionAuditState(sanction UserSanction) map[string]any {
	return map[string]any{
		"id":         sanction.ID,
		"user_id":    sanction.UserID,
		"type":       sanction.Type,
		"status":     sanction.Status,
		"reason":     sanction.Reason,
		"created_by": sanction.CreatedBy,
		"starts_at":  sanction.StartsAt,
		"expires_at": sanction.ExpiresAt,
		"revoked_by": sanction.RevokedBy,
		"revoked_at": sanction.RevokedAt,
	}
}

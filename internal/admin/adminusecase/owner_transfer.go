package adminusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/textlimit"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

func (uc *UseCase) GetCurrentOwnerTransfer(ctx context.Context, input GetCurrentOwnerTransferInput) (GetCurrentOwnerTransferResult, error) {
	actor, err := uc.findActiveOwnerTransferActor(ctx, uc.repository, input.ActorID)
	if err != nil {
		return GetCurrentOwnerTransferResult{}, err
	}
	role := effectivePlatformRole(actor)
	if role != PlatformRoleOwner && role != PlatformRoleAdmin {
		return GetCurrentOwnerTransferResult{}, apperr.New(apperr.CodeForbidden, "platform owner required")
	}
	transfer, err := uc.repository.FindCurrentOwnerTransfer(ctx, uc.now().UTC())
	if err != nil {
		if apperr.IsCode(err, apperr.CodeNotFound) {
			return GetCurrentOwnerTransferResult{}, nil
		}
		return GetCurrentOwnerTransferResult{}, fmt.Errorf("find current owner transfer: %w", err)
	}
	return GetCurrentOwnerTransferResult{Transfer: &transfer}, nil
}

func (uc *UseCase) CreateOwnerTransfer(ctx context.Context, input CreateOwnerTransferInput) (CreateOwnerTransferResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return CreateOwnerTransferResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	targetID, err := userdomain.NewUserID(input.TargetUserID)
	if err != nil {
		return CreateOwnerTransferResult{}, err
	}
	previousOwnerRole, err := normalizePreviousOwnerRole(input.PreviousOwnerRole)
	if err != nil {
		return CreateOwnerTransferResult{}, err
	}
	reason, err := textlimit.TrimmedRequiredMaxRunes(input.Reason, "owner transfer reason", MaxOwnerTransferReasonRunes)
	if err != nil {
		return CreateOwnerTransferResult{}, err
	}
	currentPassword, err := normalizeCurrentPassword(input.CurrentPassword)
	if err != nil {
		return CreateOwnerTransferResult{}, err
	}

	var transfer OwnerTransfer
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		actor, err := repository.FindUserByID(ctx, input.ActorID)
		if err != nil {
			return fmt.Errorf("find owner transfer actor: %w", err)
		}
		if actor.Status != "active" || effectivePlatformRole(actor) != PlatformRoleOwner {
			return apperr.New(apperr.CodeForbidden, "platform owner required")
		}
		if input.ActorID == targetID {
			return apperr.New(apperr.CodeInvalidArgument, "target user must be different from current owner")
		}
		target, err := repository.FindUserByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("find owner transfer target: %w", err)
		}
		if target.Status != "active" {
			return apperr.New(apperr.CodeInvalidArgument, "target user must be active")
		}
		if effectivePlatformRole(target) == PlatformRoleOwner {
			return apperr.New(apperr.CodeInvalidArgument, "target user is already platform owner")
		}
		if err := uc.verifyCurrentPassword(ctx, repository, input.ActorID, currentPassword); err != nil {
			return err
		}
		now := uc.now().UTC()
		created, err := repository.CreateOwnerTransfer(ctx, CreateOwnerTransferRecordInput{
			ID:                uuid.NewString(),
			InitiatedByID:     input.ActorID,
			TargetUserID:      targetID,
			PreviousOwnerRole: previousOwnerRole,
			Reason:            reason,
			CreatedAt:         now,
			ExpiresAt:         now.Add(PlatformOwnerTransferTTL),
		})
		if err != nil {
			if apperr.IsCode(err, apperr.CodeConflict) {
				return apperr.New(apperr.CodeConflict, "owner transfer already pending")
			}
			return fmt.Errorf("create owner transfer: %w", err)
		}
		if err := repository.CreateAuditLog(ctx, newAuditLog(input.ActorID, "admin.owner_transfer.create", "owner_transfer", created.ID, map[string]any{}, ownerTransferAuditState(created), now)); err != nil {
			return fmt.Errorf("create owner transfer audit log: %w", err)
		}
		transfer = created
		return nil
	}); err != nil {
		return CreateOwnerTransferResult{}, err
	}
	return CreateOwnerTransferResult{Transfer: transfer}, nil
}

func (uc *UseCase) CancelOwnerTransfer(ctx context.Context, input CancelOwnerTransferInput) (CancelOwnerTransferResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return CancelOwnerTransferResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	transferID, err := normalizeOwnerTransferID(input.TransferID)
	if err != nil {
		return CancelOwnerTransferResult{}, err
	}
	var transfer OwnerTransfer
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		now := uc.now().UTC()
		actor, err := uc.findActiveOwnerTransferActor(ctx, repository, input.ActorID)
		if err != nil {
			return err
		}
		before, err := repository.FindOwnerTransferByID(ctx, transferID, now)
		if err != nil {
			return fmt.Errorf("find owner transfer: %w", err)
		}
		if before.Status != OwnerTransferStatusPending {
			return apperr.New(apperr.CodeConflict, "owner transfer is not pending")
		}
		if before.InitiatedByID != input.ActorID.String() && effectivePlatformRole(actor) != PlatformRoleOwner {
			return apperr.New(apperr.CodeForbidden, "platform owner required")
		}
		cancelled, err := repository.CancelOwnerTransfer(ctx, transferID, now)
		if err != nil {
			return fmt.Errorf("cancel owner transfer: %w", err)
		}
		if err := repository.CreateAuditLog(ctx, newAuditLog(input.ActorID, "admin.owner_transfer.cancel", "owner_transfer", transferID, ownerTransferAuditState(before), ownerTransferAuditState(cancelled), now)); err != nil {
			return fmt.Errorf("create owner transfer cancel audit log: %w", err)
		}
		transfer = cancelled
		return nil
	}); err != nil {
		return CancelOwnerTransferResult{}, err
	}
	return CancelOwnerTransferResult{Transfer: transfer}, nil
}

func (uc *UseCase) GetOwnerTransfer(ctx context.Context, input GetOwnerTransferInput) (GetOwnerTransferResult, error) {
	actor, err := uc.findActiveOwnerTransferActor(ctx, uc.repository, input.ActorID)
	if err != nil {
		return GetOwnerTransferResult{}, err
	}
	transferID, err := normalizeOwnerTransferID(input.TransferID)
	if err != nil {
		return GetOwnerTransferResult{}, err
	}
	transfer, err := uc.repository.FindOwnerTransferByID(ctx, transferID, uc.now().UTC())
	if err != nil {
		return GetOwnerTransferResult{}, fmt.Errorf("find owner transfer: %w", err)
	}
	if transfer.TargetUserID != input.ActorID.String() && effectivePlatformRole(actor) != PlatformRoleOwner {
		return GetOwnerTransferResult{}, apperr.New(apperr.CodeForbidden, "owner transfer is not visible")
	}
	return GetOwnerTransferResult{Transfer: transfer}, nil
}

func (uc *UseCase) AcceptOwnerTransfer(ctx context.Context, input AcceptOwnerTransferInput) (AcceptOwnerTransferResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return AcceptOwnerTransferResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	transferID, err := normalizeOwnerTransferID(input.TransferID)
	if err != nil {
		return AcceptOwnerTransferResult{}, err
	}
	currentPassword, err := normalizeCurrentPassword(input.CurrentPassword)
	if err != nil {
		return AcceptOwnerTransferResult{}, err
	}

	var transfer OwnerTransfer
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		now := uc.now().UTC()
		actor, err := uc.findActiveOwnerTransferActor(ctx, repository, input.ActorID)
		if err != nil {
			return err
		}
		before, err := repository.FindOwnerTransferByID(ctx, transferID, now)
		if err != nil {
			return fmt.Errorf("find owner transfer: %w", err)
		}
		if before.Status != OwnerTransferStatusPending {
			return apperr.New(apperr.CodeConflict, "owner transfer is not pending")
		}
		if before.TargetUserID != input.ActorID.String() {
			return apperr.New(apperr.CodeForbidden, "only target user can accept owner transfer")
		}
		if actor.Status != "active" {
			return apperr.New(apperr.CodeInvalidArgument, "target user must be active")
		}
		if err := uc.verifyCurrentPassword(ctx, repository, input.ActorID, currentPassword); err != nil {
			return err
		}
		accepted, err := repository.AcceptOwnerTransfer(ctx, transferID, now)
		if err != nil {
			return fmt.Errorf("accept owner transfer: %w", err)
		}
		if err := repository.CreateAuditLog(ctx, newAuditLog(input.ActorID, "admin.owner_transfer.accept", "owner_transfer", transferID, ownerTransferAuditState(before), ownerTransferAuditState(accepted), now)); err != nil {
			return fmt.Errorf("create owner transfer accept audit log: %w", err)
		}
		transfer = accepted
		return nil
	}); err != nil {
		return AcceptOwnerTransferResult{}, err
	}
	return AcceptOwnerTransferResult{Transfer: transfer}, nil
}

func (uc *UseCase) BootstrapOwner(ctx context.Context, input BootstrapOwnerInput) (BootstrapOwnerResult, error) {
	if !input.Confirm {
		return BootstrapOwnerResult{}, apperr.New(apperr.CodeInvalidArgument, "bootstrap owner requires confirmation")
	}
	targetID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return BootstrapOwnerResult{}, err
	}
	reason, err := textlimit.TrimmedRequiredMaxRunes(input.Reason, "owner bootstrap reason", MaxOwnerTransferReasonRunes)
	if err != nil {
		return BootstrapOwnerResult{}, err
	}
	var user User
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		target, err := repository.FindUserByID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("find bootstrap owner user: %w", err)
		}
		if target.Status != "active" {
			return apperr.New(apperr.CodeInvalidArgument, "new owner must be active")
		}
		ownerCount, err := repository.CountPlatformOwners(ctx)
		if err != nil {
			return fmt.Errorf("count platform owners: %w", err)
		}
		if ownerCount > 0 {
			return apperr.New(apperr.CodeConflict, "platform owner already exists")
		}
		now := uc.now().UTC()
		updated, err := repository.BootstrapOwner(ctx, BootstrapOwnerRecordInput{UserID: targetID, UpdatedAt: now})
		if err != nil {
			return fmt.Errorf("bootstrap owner: %w", err)
		}
		if err := repository.CreateAuditLog(ctx, newSystemAuditLog("system:bootstrap-owner", "admin.owner.bootstrap", "user", targetID.String(), userAuditState(target), map[string]any{
			"id":                updated.ID,
			"username":          updated.Username,
			"status":            updated.Status,
			"is_platform_staff": updated.IsPlatformStaff,
			"platform_role":     updated.PlatformRole,
			"reason":            reason,
		}, now)); err != nil {
			return fmt.Errorf("create bootstrap owner audit log: %w", err)
		}
		user = updated
		return nil
	}); err != nil {
		return BootstrapOwnerResult{}, err
	}
	return BootstrapOwnerResult{User: user}, nil
}

func (uc *UseCase) RecoverOwner(ctx context.Context, input RecoverOwnerInput) (RecoverOwnerResult, error) {
	if !input.Confirm {
		return RecoverOwnerResult{}, apperr.New(apperr.CodeInvalidArgument, "recover owner requires confirmation")
	}
	newOwnerID, err := userdomain.NewUserID(input.NewOwnerUserID)
	if err != nil {
		return RecoverOwnerResult{}, err
	}
	compromisedID, err := userdomain.NewUserID(input.CompromisedUserID)
	if err != nil {
		return RecoverOwnerResult{}, err
	}
	if newOwnerID == compromisedID {
		return RecoverOwnerResult{}, apperr.New(apperr.CodeInvalidArgument, "new owner must differ from compromised owner")
	}
	reason, err := textlimit.TrimmedRequiredMaxRunes(input.Reason, "owner recovery reason", MaxOwnerTransferReasonRunes)
	if err != nil {
		return RecoverOwnerResult{}, err
	}
	var result OwnerRecoveryRecordResult
	if err := uc.withWriteRepository(ctx, func(ctx context.Context, repository Repository) error {
		newOwner, err := repository.FindUserByID(ctx, newOwnerID)
		if err != nil {
			return fmt.Errorf("find new owner user: %w", err)
		}
		if newOwner.Status != "active" {
			return apperr.New(apperr.CodeInvalidArgument, "new owner must be active")
		}
		compromised, err := repository.FindUserByID(ctx, compromisedID)
		if err != nil {
			return fmt.Errorf("find compromised owner user: %w", err)
		}
		if effectivePlatformRole(compromised) != PlatformRoleOwner {
			return apperr.New(apperr.CodeInvalidArgument, "compromised user is not platform owner")
		}
		now := uc.now().UTC()
		recovered, err := repository.RecoverOwner(ctx, RecoverOwnerRecordInput{
			NewOwnerID:         newOwnerID,
			CompromisedUserID:  compromisedID,
			UpdatedAt:          now,
			RevokeSessions:     input.RevokeSessions,
			DisableCompromised: input.DisableCompromised,
		})
		if err != nil {
			return fmt.Errorf("recover owner: %w", err)
		}
		if err := repository.CreateAuditLog(ctx, newSystemAuditLog("system:recover-owner", "admin.owner.recover", "user", newOwnerID.String(), map[string]any{
			"old_owner": compromised.ID,
			"new_owner": newOwner.ID,
		}, map[string]any{
			"old_owner":           recovered.CompromisedUser.ID,
			"old_owner_role":      recovered.CompromisedUser.PlatformRole,
			"old_owner_status":    recovered.CompromisedUser.Status,
			"new_owner":           recovered.NewOwner.ID,
			"new_owner_role":      recovered.NewOwner.PlatformRole,
			"reason":              reason,
			"revoke_sessions":     input.RevokeSessions,
			"disable_compromised": input.DisableCompromised,
		}, now)); err != nil {
			return fmt.Errorf("create recover owner audit log: %w", err)
		}
		result = recovered
		return nil
	}); err != nil {
		return RecoverOwnerResult{}, err
	}
	return RecoverOwnerResult{NewOwner: result.NewOwner, CompromisedUser: result.CompromisedUser}, nil
}

func (uc *UseCase) findActiveOwnerTransferActor(ctx context.Context, repository Repository, actorID userdomain.UserID) (User, error) {
	if strings.TrimSpace(actorID.String()) == "" {
		return User{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	actor, err := repository.FindUserByID(ctx, actorID)
	if err != nil {
		return User{}, fmt.Errorf("find owner transfer actor: %w", err)
	}
	if actor.Status != "active" {
		return User{}, apperr.New(apperr.CodeForbidden, "active platform user required")
	}
	return actor, nil
}

func (uc *UseCase) verifyCurrentPassword(ctx context.Context, repository Repository, userID userdomain.UserID, password userdomain.PlainPassword) error {
	if uc.passwordComparer == nil {
		return apperr.New(apperr.CodeInternal, "owner transfer password verification is not configured")
	}
	hash, err := repository.FindUserPasswordHash(ctx, userID)
	if err != nil {
		return fmt.Errorf("find current password hash: %w", err)
	}
	if err := uc.passwordComparer.Compare(hash, password); err != nil {
		return apperr.New(apperr.CodeForbidden, "current password is invalid")
	}
	return nil
}

func normalizePreviousOwnerRole(role *string) (string, error) {
	if role == nil {
		return "", nil
	}
	value := strings.ToLower(strings.TrimSpace(*role))
	switch value {
	case "":
		return "", nil
	case PlatformRoleAdmin:
		return value, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "previous owner role is invalid")
	}
}

func normalizeCurrentPassword(raw string) (userdomain.PlainPassword, error) {
	if strings.TrimSpace(raw) == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "current password is required")
	}
	password, err := userdomain.NewPlainPassword(raw)
	if err != nil {
		return "", apperr.New(apperr.CodeForbidden, "current password is invalid")
	}
	return password, nil
}

func normalizeOwnerTransferID(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "owner transfer id is required")
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return "", apperr.New(apperr.CodeInvalidArgument, "owner transfer id is invalid")
	}
	return parsed.String(), nil
}

func newSystemAuditLog(actorRef string, action string, targetType string, targetID string, before map[string]any, after map[string]any, createdAt time.Time) AuditLog {
	return AuditLog{
		ID:         uuid.NewString(),
		ActorID:    actorRef,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Before:     before,
		After:      after,
		CreatedAt:  createdAt,
	}
}

func ownerTransferAuditState(transfer OwnerTransfer) map[string]any {
	return map[string]any{
		"id":                    transfer.ID,
		"status":                transfer.Status,
		"initiated_by_id":       transfer.InitiatedByID,
		"initiated_by_username": transfer.InitiatedByUsername,
		"target_user_id":        transfer.TargetUserID,
		"target_username":       transfer.TargetUsername,
		"previous_owner_role":   transfer.PreviousOwnerRole,
		"reason":                transfer.Reason,
		"created_at":            transfer.CreatedAt,
		"updated_at":            transfer.UpdatedAt,
		"expires_at":            transfer.ExpiresAt,
		"accepted_at":           transfer.AcceptedAt,
		"cancelled_at":          transfer.CancelledAt,
	}
}

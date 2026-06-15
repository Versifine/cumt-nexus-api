package communityusecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

type AddCommunityModeratorInput struct {
	Slug     string
	ViewerID userdomain.UserID
	Username string
}

type RemoveCommunityModeratorInput struct {
	Slug     string
	ViewerID userdomain.UserID
	UserID   string
}

type CreateCommunityOwnerTransferInput struct {
	Slug     string
	ViewerID userdomain.UserID
	Username string
}

type AcceptCommunityOwnerTransferInput struct {
	Slug       string
	ViewerID   userdomain.UserID
	TransferID string
}

type CommunityMemberMutationResult struct {
	Community Community
	Member    CommunityMember
}

type CommunityOwnerTransferResult struct {
	Community Community
	Transfer  CommunityOwnerTransfer
}

type CommunityOwnerTransfer struct {
	ID          string
	CommunityID string
	FromUserID  string
	ToUserID    string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	AcceptedAt  *time.Time
}

type CommunityOwnerTransferRecord struct {
	ID          string
	CommunityID communitydomain.CommunityID
	FromUserID  userdomain.UserID
	ToUserID    userdomain.UserID
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	AcceptedAt  *time.Time
}

type CommunityOwnerChange struct {
	BeforeOwner CommunityMember
	AfterOwner  CommunityMember
}

func (uc *CommunityReadUseCase) AddCommunityModerator(ctx context.Context, input AddCommunityModeratorInput) (CommunityMemberMutationResult, error) {
	if isBlankUserID(input.ViewerID) {
		return CommunityMemberMutationResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	username, err := userdomain.NewUsername(input.Username)
	if err != nil {
		return CommunityMemberMutationResult{}, err
	}
	community, roleView, err := uc.findManageableCommunityBySlug(ctx, input.Slug, input.ViewerID)
	if err != nil {
		return CommunityMemberMutationResult{}, err
	}
	if !canManageCommunity(roleView) {
		return CommunityMemberMutationResult{}, apperr.New(apperr.CodeForbidden, "community owner required")
	}

	var member CommunityMember
	if err := uc.withMembershipWrite(ctx, func(txCtx context.Context, repo CommunityMembershipRepository) error {
		target, err := repo.FindActiveUserByUsername(txCtx, username.String())
		if err != nil {
			return fmt.Errorf("find moderator user: %w", err)
		}
		targetID, err := userdomain.NewUserID(target.UserID)
		if err != nil {
			return err
		}
		existing, existingErr := repo.FindActiveMemberByUserID(txCtx, community.ID(), targetID)
		if existingErr == nil {
			switch existing.Role {
			case communitydomain.MembershipRoleOwner.String():
				return apperr.New(apperr.CodeInvalidArgument, "community owner cannot be assigned as moderator")
			case communitydomain.MembershipRoleModerator.String():
				member = existing
				return nil
			}
		} else if !apperr.IsCode(existingErr, apperr.CodeNotFound) {
			return fmt.Errorf("find existing community member: %w", existingErr)
		}

		memberCount, err := repo.CountActiveMembers(txCtx, community.ID())
		if err != nil {
			return fmt.Errorf("count community members: %w", err)
		}
		if apperr.IsCode(existingErr, apperr.CodeNotFound) {
			memberCount++
		}
		moderatorCount, err := repo.CountActiveModerators(txCtx, community.ID())
		if err != nil {
			return fmt.Errorf("count community moderators: %w", err)
		}
		if moderatorCount >= moderatorLimit(memberCount) {
			return apperr.New(apperr.CodeForbidden, "community moderator limit reached")
		}
		updated, err := repo.UpsertActiveMemberRole(txCtx, community.ID(), targetID, communitydomain.MembershipRoleModerator, uc.now().UTC())
		if err != nil {
			return fmt.Errorf("assign community moderator: %w", err)
		}
		member = updated
		return nil
	}); err != nil {
		return CommunityMemberMutationResult{}, err
	}
	communityDTO, err := uc.buildCommunityDTOForViewer(ctx, *community, roleView, input.ViewerID)
	if err != nil {
		return CommunityMemberMutationResult{}, err
	}
	return CommunityMemberMutationResult{Community: communityDTO, Member: member}, nil
}

func (uc *CommunityReadUseCase) RemoveCommunityModerator(ctx context.Context, input RemoveCommunityModeratorInput) (CommunityMemberMutationResult, error) {
	if isBlankUserID(input.ViewerID) {
		return CommunityMemberMutationResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	targetID, err := userdomain.NewUserID(input.UserID)
	if err != nil {
		return CommunityMemberMutationResult{}, err
	}
	community, roleView, err := uc.findManageableCommunityBySlug(ctx, input.Slug, input.ViewerID)
	if err != nil {
		return CommunityMemberMutationResult{}, err
	}
	if !canManageCommunity(roleView) {
		return CommunityMemberMutationResult{}, apperr.New(apperr.CodeForbidden, "community owner required")
	}
	if targetID == input.ViewerID {
		return CommunityMemberMutationResult{}, apperr.New(apperr.CodeInvalidArgument, "community owner cannot remove self as moderator")
	}

	var member CommunityMember
	if err := uc.withMembershipWrite(ctx, func(txCtx context.Context, repo CommunityMembershipRepository) error {
		existing, err := repo.FindActiveMemberByUserID(txCtx, community.ID(), targetID)
		if err != nil {
			return fmt.Errorf("find community member: %w", err)
		}
		switch existing.Role {
		case communitydomain.MembershipRoleOwner.String():
			return apperr.New(apperr.CodeForbidden, "community owner cannot be removed as moderator")
		case communitydomain.MembershipRoleMember.String():
			member = existing
			return nil
		}
		updated, err := repo.UpsertActiveMemberRole(txCtx, community.ID(), targetID, communitydomain.MembershipRoleMember, uc.now().UTC())
		if err != nil {
			return fmt.Errorf("remove community moderator: %w", err)
		}
		member = updated
		return nil
	}); err != nil {
		return CommunityMemberMutationResult{}, err
	}
	communityDTO, err := uc.buildCommunityDTOForViewer(ctx, *community, roleView, input.ViewerID)
	if err != nil {
		return CommunityMemberMutationResult{}, err
	}
	return CommunityMemberMutationResult{Community: communityDTO, Member: member}, nil
}

func (uc *CommunityReadUseCase) CreateCommunityOwnerTransfer(ctx context.Context, input CreateCommunityOwnerTransferInput) (CommunityOwnerTransferResult, error) {
	if isBlankUserID(input.ViewerID) {
		return CommunityOwnerTransferResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	username, err := userdomain.NewUsername(input.Username)
	if err != nil {
		return CommunityOwnerTransferResult{}, err
	}
	community, roleView, err := uc.findManageableCommunityBySlug(ctx, input.Slug, input.ViewerID)
	if err != nil {
		return CommunityOwnerTransferResult{}, err
	}
	if roleView.role != communitydomain.MembershipRoleOwner {
		return CommunityOwnerTransferResult{}, apperr.New(apperr.CodeForbidden, "community owner required")
	}

	var transfer CommunityOwnerTransferRecord
	if err := uc.withMembershipWrite(ctx, func(txCtx context.Context, repo CommunityMembershipRepository) error {
		target, err := repo.FindActiveUserByUsername(txCtx, username.String())
		if err != nil {
			return fmt.Errorf("find owner transfer target: %w", err)
		}
		targetID, err := userdomain.NewUserID(target.UserID)
		if err != nil {
			return err
		}
		if targetID == input.ViewerID {
			return apperr.New(apperr.CodeInvalidArgument, "community owner transfer target cannot be self")
		}
		now := uc.now().UTC()
		transfer = CommunityOwnerTransferRecord{
			ID:          uuid.NewString(),
			CommunityID: community.ID(),
			FromUserID:  input.ViewerID,
			ToUserID:    targetID,
			Status:      "pending",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := repo.CreateOwnerTransfer(txCtx, transfer); err != nil {
			return fmt.Errorf("create community owner transfer: %w", err)
		}
		return nil
	}); err != nil {
		return CommunityOwnerTransferResult{}, err
	}
	communityDTO, err := uc.buildCommunityDTOForViewer(ctx, *community, roleView, input.ViewerID)
	if err != nil {
		return CommunityOwnerTransferResult{}, err
	}
	return CommunityOwnerTransferResult{Community: communityDTO, Transfer: toCommunityOwnerTransfer(transfer)}, nil
}

func (uc *CommunityReadUseCase) AcceptCommunityOwnerTransfer(ctx context.Context, input AcceptCommunityOwnerTransferInput) (CommunityOwnerTransferResult, error) {
	if isBlankUserID(input.ViewerID) {
		return CommunityOwnerTransferResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	communitySlug, err := communitydomain.NewCommunitySlug(input.Slug)
	if err != nil {
		return CommunityOwnerTransferResult{}, err
	}
	if _, err := uuid.Parse(input.TransferID); err != nil {
		return CommunityOwnerTransferResult{}, apperr.New(apperr.CodeInvalidArgument, "owner transfer id is invalid")
	}
	community, err := uc.communities.FindBySlug(ctx, communitySlug)
	if err != nil {
		return CommunityOwnerTransferResult{}, fmt.Errorf("find community: %w", err)
	}
	if community.Status() != communitydomain.CommunityStatusActive {
		return CommunityOwnerTransferResult{}, apperr.New(apperr.CodeNotFound, "community not found")
	}

	var transfer CommunityOwnerTransferRecord
	if err := uc.withMembershipWrite(ctx, func(txCtx context.Context, repo CommunityMembershipRepository) error {
		record, err := repo.FindOwnerTransferForUpdate(txCtx, input.TransferID)
		if err != nil {
			return fmt.Errorf("find community owner transfer: %w", err)
		}
		if record.CommunityID != community.ID() {
			return apperr.New(apperr.CodeNotFound, "community owner transfer not found")
		}
		if record.Status != "pending" {
			return apperr.New(apperr.CodeConflict, "community owner transfer is not pending")
		}
		if record.ToUserID != input.ViewerID {
			return apperr.New(apperr.CodeForbidden, "community owner transfer target required")
		}
		now := uc.now().UTC()
		if _, err := repo.TransferOwner(txCtx, community.ID(), input.ViewerID, now); err != nil {
			return fmt.Errorf("transfer community owner: %w", err)
		}
		if err := repo.AcceptOwnerTransfer(txCtx, input.TransferID, now); err != nil {
			return fmt.Errorf("accept community owner transfer: %w", err)
		}
		record.Status = "accepted"
		record.AcceptedAt = &now
		record.UpdatedAt = now
		transfer = record
		return nil
	}); err != nil {
		return CommunityOwnerTransferResult{}, err
	}
	roleView := communityRoleView{role: communitydomain.MembershipRoleOwner}
	communityDTO, err := uc.buildCommunityDTOForViewer(ctx, *community, roleView, input.ViewerID)
	if err != nil {
		return CommunityOwnerTransferResult{}, err
	}
	return CommunityOwnerTransferResult{Community: communityDTO, Transfer: toCommunityOwnerTransfer(transfer)}, nil
}

func (uc *CommunityReadUseCase) withMembershipWrite(ctx context.Context, fn func(ctx context.Context, repo CommunityMembershipRepository) error) error {
	if uc.transactions != nil {
		return uc.transactions.WithinTx(ctx, func(txCtx context.Context, repositories CommunityRepositories) error {
			return fn(txCtx, repositories.Memberships())
		})
	}
	if uc.membershipOps == nil {
		return apperr.New(apperr.CodeInternal, "community membership writes are not configured")
	}
	return fn(ctx, uc.membershipOps)
}

func moderatorLimit(memberCount int) int {
	switch {
	case memberCount >= 2000:
		return 20
	case memberCount >= 500:
		return 10
	default:
		return 5
	}
}

func toCommunityOwnerTransfer(record CommunityOwnerTransferRecord) CommunityOwnerTransfer {
	return CommunityOwnerTransfer{
		ID:          record.ID,
		CommunityID: record.CommunityID.String(),
		FromUserID:  record.FromUserID.String(),
		ToUserID:    record.ToUserID.String(),
		Status:      record.Status,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
		AcceptedAt:  record.AcceptedAt,
	}
}

package adminusecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/platform/settings"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestListUsersRequiresPlatformStaff(t *testing.T) {
	uc := NewUseCase(&fakeRepository{}, time.Now)
	if _, err := uc.ListUsers(context.Background(), ListUsersInput{}); !apperr.IsCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated for missing actor, got %v", err)
	}

	uc = NewUseCase(&fakeRepository{isStaff: false}, time.Now)
	if _, err := uc.ListUsers(context.Background(), ListUsersInput{ActorID: userdomain.NewGeneratedUserID()}); !apperr.IsCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for non-staff actor, got %v", err)
	}
}

func TestListUsersSupportsSearchQuery(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{
		isStaff: true,
		users: map[string]User{
			"1": {ID: "1", Username: "alice", Status: "active"},
			"2": {ID: "2", Username: "bob", Status: "disabled"},
			"3": {ID: "3", Username: "campus-admin", Status: "active"},
		},
	}
	uc := NewUseCase(repo, time.Now)

	result, err := uc.ListUsers(context.Background(), ListUsersInput{
		ActorID: actorID,
		Status:  "all",
		Query:   " admin ",
	})
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}
	if result.Query != "admin" {
		t.Fatalf("expected normalized query admin, got %q", result.Query)
	}
	if len(result.Users) != 1 || result.Users[0].Username != "campus-admin" {
		t.Fatalf("unexpected search result: %#v", result.Users)
	}
}

func TestListCommunitiesSupportsSearchQuery(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{
		isStaff: true,
		communities: map[string]Community{
			"1": {ID: "1", Slug: "campus-life", Name: "Campus Life", Description: "events", Status: "active"},
			"2": {ID: "2", Slug: "lost-found", Name: "Lost", Description: "finder board", Status: "active"},
		},
	}
	uc := NewUseCase(repo, time.Now)

	result, err := uc.ListCommunities(context.Background(), ListCommunitiesInput{
		ActorID: actorID,
		Query:   "found",
	})
	if err != nil {
		t.Fatalf("ListCommunities returned error: %v", err)
	}
	if result.Query != "found" {
		t.Fatalf("expected normalized query found, got %q", result.Query)
	}
	if len(result.Communities) != 1 || result.Communities[0].Slug != "lost-found" {
		t.Fatalf("unexpected search result: %#v", result.Communities)
	}
}

func TestUpdateCommunityOwnerWritesReasonToAudit(t *testing.T) {
	now := time.Date(2026, 6, 10, 9, 30, 0, 0, time.UTC)
	actorID := userdomain.NewGeneratedUserID()
	communityID := communitydomain.NewGeneratedCommunityID()
	targetID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{
		isStaff: true,
		communities: map[string]Community{
			communityID.String(): {ID: communityID.String(), Slug: "campus", Name: "Campus", Status: "active"},
		},
		users: map[string]User{
			targetID.String(): {ID: targetID.String(), Username: "alice", Status: "active"},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return now })

	result, err := uc.UpdateCommunityOwner(context.Background(), UpdateCommunityOwnerInput{
		ActorID:     actorID,
		CommunityID: communityID.String(),
		UserID:      targetID.String(),
		Reason:      " owner left campus ",
	})
	if err != nil {
		t.Fatalf("UpdateCommunityOwner returned error: %v", err)
	}
	if result.Owner.UserID != targetID.String() {
		t.Fatalf("unexpected owner result: %#v", result.Owner)
	}
	if len(repo.auditLogs) != 1 {
		t.Fatalf("expected one audit log, got %d", len(repo.auditLogs))
	}
	log := repo.auditLogs[0]
	if log.Action != "admin.communities.update_owner" || log.After["reason"] != "owner left campus" {
		t.Fatalf("unexpected audit log: %#v", log)
	}
}

func TestUpdateCommunityOwnerAllowsMissingActiveOwner(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	actorID := userdomain.NewGeneratedUserID()
	communityID := communitydomain.NewGeneratedCommunityID()
	targetID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{
		isStaff:                true,
		noActiveCommunityOwner: true,
		communities: map[string]Community{
			communityID.String(): {ID: communityID.String(), Slug: "campus", Name: "Campus", Status: "active"},
		},
		users: map[string]User{
			targetID.String(): {ID: targetID.String(), Username: "alice", Status: "active"},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return now })

	result, err := uc.UpdateCommunityOwner(context.Background(), UpdateCommunityOwnerInput{
		ActorID:     actorID,
		CommunityID: communityID.String(),
		UserID:      targetID.String(),
		Reason:      "no active owner",
	})
	if err != nil {
		t.Fatalf("UpdateCommunityOwner returned error: %v", err)
	}
	if result.Owner.UserID != targetID.String() {
		t.Fatalf("unexpected owner result: %#v", result.Owner)
	}
	if len(repo.auditLogs) != 1 {
		t.Fatalf("expected one audit log, got %d", len(repo.auditLogs))
	}
	if repo.auditLogs[0].Before["owner"] != nil {
		t.Fatalf("expected nil before owner, got %#v", repo.auditLogs[0].Before)
	}
	if repo.auditLogs[0].After["reason"] != "no active owner" {
		t.Fatalf("expected reason in audit after state, got %#v", repo.auditLogs[0].After)
	}
}

func TestUpdateSettingWritesAuditLogInTransaction(t *testing.T) {
	now := time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)
	actorID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{
		isStaff: true,
		settings: map[string]Setting{
			settings.RegistrationEnabled: {
				Key:       settings.RegistrationEnabled,
				Enabled:   true,
				UpdatedAt: now.Add(-time.Hour),
			},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return now })

	result, err := uc.UpdateSetting(context.Background(), UpdateSettingInput{
		ActorID: actorID,
		Key:     " registration_enabled ",
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("UpdateSetting returned error: %v", err)
	}
	if result.Setting.Enabled {
		t.Fatal("expected setting to be disabled")
	}
	if result.Setting.UpdatedBy != actorID.String() {
		t.Fatalf("expected updated_by %q, got %q", actorID.String(), result.Setting.UpdatedBy)
	}
	if repo.txCount != 1 {
		t.Fatalf("expected transaction to be used once, got %d", repo.txCount)
	}
	if len(repo.auditLogs) != 1 {
		t.Fatalf("expected one audit log, got %d", len(repo.auditLogs))
	}
	log := repo.auditLogs[0]
	if log.Action != "admin.settings.update" || log.TargetType != "setting" || log.TargetID != settings.RegistrationEnabled {
		t.Fatalf("unexpected audit log identity: %#v", log)
	}
	if log.Before["enabled"] != true || log.After["enabled"] != false {
		t.Fatalf("unexpected audit before/after: %#v -> %#v", log.Before, log.After)
	}
}

func TestUpdateUserPreservesOmittedFieldsAndAudits(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	staff := true
	repo := &fakeRepository{
		isStaff: true,
		users: map[string]User{
			actorID.String(): {
				ID:              actorID.String(),
				Username:        "owner",
				Status:          "active",
				IsPlatformStaff: true,
				PlatformRole:    PlatformRoleOwner,
			},
			targetID.String(): {
				ID:              targetID.String(),
				Username:        "alice",
				Status:          "active",
				IsPlatformStaff: false,
				CreatedAt:       now.Add(-24 * time.Hour),
				UpdatedAt:       now.Add(-time.Hour),
			},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return now })

	result, err := uc.UpdateUser(context.Background(), UpdateUserInput{
		ActorID:         actorID,
		UserID:          targetID.String(),
		IsPlatformStaff: &staff,
	})
	if err != nil {
		t.Fatalf("UpdateUser returned error: %v", err)
	}
	if result.User.Status != "active" {
		t.Fatalf("expected status to be preserved, got %q", result.User.Status)
	}
	if !result.User.IsPlatformStaff {
		t.Fatal("expected user to become platform staff")
	}
	if len(repo.auditLogs) != 1 {
		t.Fatalf("expected one audit log, got %d", len(repo.auditLogs))
	}
	log := repo.auditLogs[0]
	if log.Before["is_platform_staff"] != false || log.After["is_platform_staff"] != true {
		t.Fatalf("unexpected staff audit before/after: %#v -> %#v", log.Before, log.After)
	}
	if log.After["status"] != "active" {
		t.Fatalf("expected audit after status active, got %#v", log.After["status"])
	}
}

func TestUpdateUserPlatformRoleAllowsOwnerToAssignAdmin(t *testing.T) {
	now := time.Date(2026, 6, 14, 13, 0, 0, 0, time.UTC)
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	role := PlatformRoleAdmin
	repo := &fakeRepository{
		users: map[string]User{
			actorID.String(): {
				ID:              actorID.String(),
				Username:        "owner",
				Status:          "active",
				IsPlatformStaff: true,
				PlatformRole:    PlatformRoleOwner,
			},
			targetID.String(): {
				ID:       targetID.String(),
				Username: "alice",
				Status:   "active",
			},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return now })

	result, err := uc.UpdateUserPlatformRole(context.Background(), UpdateUserPlatformRoleInput{
		ActorID: actorID,
		UserID:  targetID.String(),
		Role:    &role,
	})
	if err != nil {
		t.Fatalf("UpdateUserPlatformRole returned error: %v", err)
	}
	if result.User.PlatformRole != PlatformRoleAdmin || !result.User.IsPlatformStaff {
		t.Fatalf("unexpected platform role result: %#v", result.User)
	}
	if len(repo.auditLogs) != 1 {
		t.Fatalf("expected one audit log, got %d", len(repo.auditLogs))
	}
	if repo.auditLogs[0].Action != "admin.users.update_platform_role" {
		t.Fatalf("unexpected audit action: %q", repo.auditLogs[0].Action)
	}
}

func TestUpdateUserPlatformRoleRejectsAdminAssigningAdmin(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	role := PlatformRoleAdmin
	repo := &fakeRepository{
		users: map[string]User{
			actorID.String(): {
				ID:              actorID.String(),
				Status:          "active",
				IsPlatformStaff: true,
				PlatformRole:    PlatformRoleAdmin,
			},
			targetID.String(): {
				ID:       targetID.String(),
				Username: "alice",
				Status:   "active",
			},
		},
	}
	uc := NewUseCase(repo, time.Now)

	_, err := uc.UpdateUserPlatformRole(context.Background(), UpdateUserPlatformRoleInput{
		ActorID: actorID,
		UserID:  targetID.String(),
		Role:    &role,
	})
	if !apperr.IsCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for admin assigning admin, got %v", err)
	}
}

func TestUpdateUserPlatformRoleRejectsRemovingLastOwner(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	role := ""
	repo := &fakeRepository{
		users: map[string]User{
			actorID.String(): {
				ID:              actorID.String(),
				Status:          "active",
				IsPlatformStaff: true,
				PlatformRole:    PlatformRoleOwner,
			},
		},
	}
	uc := NewUseCase(repo, time.Now)

	_, err := uc.UpdateUserPlatformRole(context.Background(), UpdateUserPlatformRoleInput{
		ActorID: actorID,
		UserID:  actorID.String(),
		Role:    &role,
	})
	if !apperr.IsCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for removing last owner, got %v", err)
	}
}

func TestUpdateUserPlatformRoleRejectsSettingOwner(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	role := PlatformRoleOwner
	repo := &fakeRepository{
		users: map[string]User{
			actorID.String():  {ID: actorID.String(), Status: "active", IsPlatformStaff: true, PlatformRole: PlatformRoleOwner},
			targetID.String(): {ID: targetID.String(), Status: "active"},
		},
	}
	uc := NewUseCase(repo, time.Now)

	_, err := uc.UpdateUserPlatformRole(context.Background(), UpdateUserPlatformRoleInput{
		ActorID: actorID,
		UserID:  targetID.String(),
		Role:    &role,
	})
	if !apperr.IsCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for setting owner, got %v", err)
	}
}

func TestUpdateUserRejectsOwnerStatusChange(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	status := "disabled"
	repo := &fakeRepository{
		users: map[string]User{
			actorID.String():  {ID: actorID.String(), Status: "active", IsPlatformStaff: true, PlatformRole: PlatformRoleOwner},
			targetID.String(): {ID: targetID.String(), Status: "active", IsPlatformStaff: true, PlatformRole: PlatformRoleOwner},
		},
	}
	uc := NewUseCase(repo, time.Now)

	_, err := uc.UpdateUser(context.Background(), UpdateUserInput{
		ActorID: actorID,
		UserID:  targetID.String(),
		Status:  &status,
	})
	if !apperr.IsCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for owner status update, got %v", err)
	}
}

func TestUpdateUserAdminCannotUpdatePlatformRoleAccount(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	status := "disabled"
	repo := &fakeRepository{
		users: map[string]User{
			actorID.String():  {ID: actorID.String(), Status: "active", IsPlatformStaff: true, PlatformRole: PlatformRoleAdmin},
			targetID.String(): {ID: targetID.String(), Status: "active", IsPlatformStaff: true, PlatformRole: PlatformRoleStaff},
		},
	}
	uc := NewUseCase(repo, time.Now)

	_, err := uc.UpdateUser(context.Background(), UpdateUserInput{
		ActorID: actorID,
		UserID:  targetID.String(),
		Status:  &status,
	})
	if !apperr.IsCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for admin updating platform role account, got %v", err)
	}
}

func TestOwnerTransferCreateAndAccept(t *testing.T) {
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)
	ownerID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	previousRole := PlatformRoleAdmin
	repo := &fakeRepository{
		users: map[string]User{
			ownerID.String():  {ID: ownerID.String(), Username: "owner", Status: "active", IsPlatformStaff: true, PlatformRole: PlatformRoleOwner},
			targetID.String(): {ID: targetID.String(), Username: "alice", Status: "active"},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return now })
	uc.SetPasswordComparer(fakePasswordComparer{})

	created, err := uc.CreateOwnerTransfer(context.Background(), CreateOwnerTransferInput{
		ActorID:           ownerID,
		TargetUserID:      targetID.String(),
		PreviousOwnerRole: &previousRole,
		Reason:            " graduation handoff ",
		CurrentPassword:   "correct-password",
	})
	if err != nil {
		t.Fatalf("CreateOwnerTransfer returned error: %v", err)
	}
	if created.Transfer.Status != OwnerTransferStatusPending ||
		created.Transfer.Reason != "graduation handoff" ||
		!created.Transfer.ExpiresAt.Equal(now.Add(PlatformOwnerTransferTTL)) {
		t.Fatalf("unexpected created transfer: %#v", created.Transfer)
	}

	accepted, err := uc.AcceptOwnerTransfer(context.Background(), AcceptOwnerTransferInput{
		ActorID:         targetID,
		TransferID:      created.Transfer.ID,
		CurrentPassword: "correct-password",
	})
	if err != nil {
		t.Fatalf("AcceptOwnerTransfer returned error: %v", err)
	}
	if accepted.Transfer.Status != OwnerTransferStatusAccepted || accepted.Transfer.AcceptedAt == nil {
		t.Fatalf("unexpected accepted transfer: %#v", accepted.Transfer)
	}
	if repo.users[targetID.String()].PlatformRole != PlatformRoleOwner {
		t.Fatalf("expected target to become owner, got %#v", repo.users[targetID.String()])
	}
	if repo.users[ownerID.String()].PlatformRole != PlatformRoleAdmin {
		t.Fatalf("expected previous owner to become admin, got %#v", repo.users[ownerID.String()])
	}
	if len(repo.auditLogs) != 2 {
		t.Fatalf("expected create and accept audit logs, got %#v", repo.auditLogs)
	}
}

func TestCreateOwnerTransferRejectsDuplicatePending(t *testing.T) {
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)
	ownerID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	otherTargetID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{
		users: map[string]User{
			ownerID.String():       {ID: ownerID.String(), Username: "owner", Status: "active", IsPlatformStaff: true, PlatformRole: PlatformRoleOwner},
			targetID.String():      {ID: targetID.String(), Username: "alice", Status: "active"},
			otherTargetID.String(): {ID: otherTargetID.String(), Username: "bob", Status: "active"},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return now })
	uc.SetPasswordComparer(fakePasswordComparer{})

	if _, err := uc.CreateOwnerTransfer(context.Background(), CreateOwnerTransferInput{
		ActorID:         ownerID,
		TargetUserID:    targetID.String(),
		Reason:          "first",
		CurrentPassword: "correct-password",
	}); err != nil {
		t.Fatalf("first CreateOwnerTransfer returned error: %v", err)
	}
	_, err := uc.CreateOwnerTransfer(context.Background(), CreateOwnerTransferInput{
		ActorID:         ownerID,
		TargetUserID:    otherTargetID.String(),
		Reason:          "second",
		CurrentPassword: "correct-password",
	})
	if !apperr.IsCode(err, apperr.CodeConflict) {
		t.Fatalf("expected duplicate pending conflict, got %v", err)
	}
}

func TestAcceptOwnerTransferRejectsNonTarget(t *testing.T) {
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)
	ownerID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	otherID := userdomain.NewGeneratedUserID()
	transferID := userdomain.NewGeneratedUserID().String()
	repo := &fakeRepository{
		users: map[string]User{
			ownerID.String():  {ID: ownerID.String(), Username: "owner", Status: "active", IsPlatformStaff: true, PlatformRole: PlatformRoleOwner},
			targetID.String(): {ID: targetID.String(), Username: "alice", Status: "active"},
			otherID.String():  {ID: otherID.String(), Username: "bob", Status: "active"},
		},
		ownerTransfers: map[string]OwnerTransfer{
			transferID: {ID: transferID, Status: OwnerTransferStatusPending, InitiatedByID: ownerID.String(), TargetUserID: targetID.String(), ExpiresAt: now.Add(time.Hour)},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return now })
	uc.SetPasswordComparer(fakePasswordComparer{})

	_, err := uc.AcceptOwnerTransfer(context.Background(), AcceptOwnerTransferInput{
		ActorID:         otherID,
		TransferID:      transferID,
		CurrentPassword: "correct-password",
	})
	if !apperr.IsCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for non-target accept, got %v", err)
	}
}

func TestAcceptOwnerTransferRejectsExpiredTransfer(t *testing.T) {
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)
	ownerID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	transferID := userdomain.NewGeneratedUserID().String()
	repo := &fakeRepository{
		users: map[string]User{
			ownerID.String():  {ID: ownerID.String(), Username: "owner", Status: "active", IsPlatformStaff: true, PlatformRole: PlatformRoleOwner},
			targetID.String(): {ID: targetID.String(), Username: "alice", Status: "active"},
		},
		ownerTransfers: map[string]OwnerTransfer{
			transferID: {ID: transferID, Status: OwnerTransferStatusPending, InitiatedByID: ownerID.String(), TargetUserID: targetID.String(), ExpiresAt: now.Add(-time.Minute)},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return now })
	uc.SetPasswordComparer(fakePasswordComparer{})

	_, err := uc.AcceptOwnerTransfer(context.Background(), AcceptOwnerTransferInput{
		ActorID:         targetID,
		TransferID:      transferID,
		CurrentPassword: "correct-password",
	})
	if !apperr.IsCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for expired transfer, got %v", err)
	}
}

func TestBootstrapOwnerFailsWhenOwnerExists(t *testing.T) {
	ownerID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{
		users: map[string]User{
			ownerID.String():  {ID: ownerID.String(), Username: "owner", Status: "active", IsPlatformStaff: true, PlatformRole: PlatformRoleOwner},
			targetID.String(): {ID: targetID.String(), Username: "alice", Status: "active"},
		},
	}
	uc := NewUseCase(repo, time.Now)

	_, err := uc.BootstrapOwner(context.Background(), BootstrapOwnerInput{
		UserID:  targetID.String(),
		Reason:  "initial recovery",
		Confirm: true,
	})
	if !apperr.IsCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict when owner exists, got %v", err)
	}
}

func TestUpdateEffectRejectsDatabaseInvalidID(t *testing.T) {
	uc := NewUseCase(&fakeRepository{isStaff: true}, time.Now)

	_, err := uc.UpdateEffectActive(context.Background(), UpdateEffectActiveInput{
		ActorID:  userdomain.NewGeneratedUserID(),
		EffectID: "_sparkle",
		IsActive: false,
	})
	if !apperr.IsCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for effect id that fails database format, got %v", err)
	}
}

func TestAdjustUserPointsWritesTransactionAndAuditLog(t *testing.T) {
	now := time.Date(2026, 6, 10, 11, 0, 0, 0, time.UTC)
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{
		isStaff: true,
		users: map[string]User{
			targetID.String(): {
				ID:       targetID.String(),
				Username: "bob",
				Status:   "active",
			},
		},
		pointAccounts: map[string]PointAccount{
			targetID.String(): {
				UserID:         targetID.String(),
				Balance:        20,
				LifetimeEarned: 20,
				UpdatedAt:      now.Add(-time.Hour),
			},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return now })

	result, err := uc.AdjustUserPoints(context.Background(), AdjustUserPointsInput{
		ActorID: actorID,
		UserID:  targetID.String(),
		Delta:   15,
		Reason:  " manual bonus ",
	})
	if err != nil {
		t.Fatalf("AdjustUserPoints returned error: %v", err)
	}
	if result.Account.Balance != 35 || result.Account.LifetimeEarned != 35 {
		t.Fatalf("unexpected account after adjustment: %#v", result.Account)
	}
	if result.Transaction.Delta != 15 || result.Transaction.Reason != "manual bonus" || result.Transaction.SourceType != "admin_adjustment" || result.Transaction.SourceID != actorID.String() {
		t.Fatalf("unexpected point transaction: %#v", result.Transaction)
	}
	if repo.txCount != 1 {
		t.Fatalf("expected transaction to be used once, got %d", repo.txCount)
	}
	if len(repo.auditLogs) != 1 {
		t.Fatalf("expected one audit log, got %d", len(repo.auditLogs))
	}
	log := repo.auditLogs[0]
	if log.Action != "admin.points.adjust" || log.TargetType != "user" || log.TargetID != targetID.String() {
		t.Fatalf("unexpected audit log identity: %#v", log)
	}
	if log.Before["balance"] != 20 || log.After["balance"] != 35 || log.After["delta"] != 15 {
		t.Fatalf("unexpected audit before/after: %#v -> %#v", log.Before, log.After)
	}
}

func TestAdjustUserPointsRejectsInsufficientBalance(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{
		isStaff: true,
		users: map[string]User{
			targetID.String(): {
				ID:       targetID.String(),
				Username: "bob",
				Status:   "active",
			},
		},
		pointAccounts: map[string]PointAccount{
			targetID.String(): {
				UserID:  targetID.String(),
				Balance: 5,
			},
		},
	}
	uc := NewUseCase(repo, time.Now)

	_, err := uc.AdjustUserPoints(context.Background(), AdjustUserPointsInput{
		ActorID: actorID,
		UserID:  targetID.String(),
		Delta:   -10,
		Reason:  "penalty",
	})
	if !apperr.IsCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for insufficient balance, got %v", err)
	}
	if len(repo.pointTransactions) != 0 {
		t.Fatalf("expected no point transactions, got %d", len(repo.pointTransactions))
	}
	if len(repo.auditLogs) != 0 {
		t.Fatalf("expected no audit logs, got %d", len(repo.auditLogs))
	}
}

func TestListPointTransactionsSupportsUserFilterAndPagination(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	otherID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{
		isStaff: true,
		pointTransactions: []PointTransaction{
			{ID: "1", UserID: targetID.String(), Delta: 1},
			{ID: "2", UserID: targetID.String(), Delta: 2},
			{ID: "3", UserID: otherID.String(), Delta: 3},
		},
	}
	uc := NewUseCase(repo, time.Now)

	result, err := uc.ListPointTransactions(context.Background(), ListPointTransactionsInput{
		ActorID: actorID,
		UserID:  targetID.String(),
		Limit:   1,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("ListPointTransactions returned error: %v", err)
	}
	if len(result.Transactions) != 1 || result.Transactions[0].ID != "1" {
		t.Fatalf("unexpected filtered transactions: %#v", result.Transactions)
	}
	if !result.HasMore || result.NextOffset != 1 {
		t.Fatalf("unexpected pagination: has_more=%t next_offset=%d", result.HasMore, result.NextOffset)
	}
}

func TestListAuditLogsSupportsSearchQuery(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{
		isStaff: true,
		auditLogs: []AuditLog{
			{ID: "1", Action: "admin.users.update", TargetType: "user", TargetID: "user-1"},
			{ID: "2", Action: "admin.communities.update_status", TargetType: "community", TargetID: "community-1"},
		},
	}
	uc := NewUseCase(repo, time.Now)

	result, err := uc.ListAuditLogs(context.Background(), ListAuditLogsInput{
		ActorID: actorID,
		Query:   "communities",
	})
	if err != nil {
		t.Fatalf("ListAuditLogs returned error: %v", err)
	}
	if result.Query != "communities" {
		t.Fatalf("expected normalized query communities, got %q", result.Query)
	}
	if len(result.AuditLogs) != 1 || result.AuditLogs[0].ID != "2" {
		t.Fatalf("unexpected audit search result: %#v", result.AuditLogs)
	}
}

func TestCreateUserSanctionCreatesTimedBanAndAudit(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{
		users: map[string]User{
			actorID.String():  {ID: actorID.String(), Username: "owner", Status: "active", IsPlatformStaff: true, PlatformRole: PlatformRoleOwner},
			targetID.String(): {ID: targetID.String(), Username: "alice", Status: "active"},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return now })

	result, err := uc.CreateUserSanction(context.Background(), CreateUserSanctionInput{
		ActorID:  actorID,
		UserID:   targetID.String(),
		Type:     "account_ban",
		Duration: "7d",
		Reason:   " repeated spam ",
	})
	if err != nil {
		t.Fatalf("CreateUserSanction returned error: %v", err)
	}
	if result.Sanction.Type != UserSanctionTypeAccountBan ||
		result.Sanction.Status != UserSanctionStatusActive ||
		result.Sanction.Reason != "repeated spam" ||
		result.Sanction.ExpiresAt == nil ||
		!result.Sanction.ExpiresAt.Equal(now.Add(7*24*time.Hour)) {
		t.Fatalf("unexpected sanction: %#v", result.Sanction)
	}
	if repo.createdSanction.UserID != targetID || repo.createdSanction.CreatedBy != actorID {
		t.Fatalf("unexpected created sanction input: %#v", repo.createdSanction)
	}
	if len(repo.auditLogs) != 1 || repo.auditLogs[0].Action != "admin.users.create_sanction" {
		t.Fatalf("expected sanction audit log, got %#v", repo.auditLogs)
	}
}

func TestCreateUserSanctionRequiresActor(t *testing.T) {
	targetID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{
		users: map[string]User{
			targetID.String(): {ID: targetID.String(), Username: "alice", Status: "active"},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC) })

	_, err := uc.CreateUserSanction(context.Background(), CreateUserSanctionInput{
		UserID:   targetID.String(),
		Type:     "account_ban",
		Duration: "1d",
		Reason:   "abuse",
	})
	if !apperr.IsCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
	if repo.createdSanction.ID != "" {
		t.Fatalf("sanction should not be created without actor, got %#v", repo.createdSanction)
	}
}

func TestCreateUserSanctionAdminCannotSanctionPlatformStaff(t *testing.T) {
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{
		users: map[string]User{
			actorID.String():  {ID: actorID.String(), Username: "admin", Status: "active", IsPlatformStaff: true, PlatformRole: PlatformRoleAdmin},
			targetID.String(): {ID: targetID.String(), Username: "staff", Status: "active", IsPlatformStaff: true, PlatformRole: PlatformRoleStaff},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC) })

	_, err := uc.CreateUserSanction(context.Background(), CreateUserSanctionInput{
		ActorID:  actorID,
		UserID:   targetID.String(),
		Type:     "account_ban",
		Duration: "1d",
		Reason:   "abuse",
	})
	if !apperr.IsCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestRevokeUserSanctionRejectsInactiveActor(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	sanctionID := userdomain.NewGeneratedUserID().String()
	repo := &fakeRepository{
		users: map[string]User{
			actorID.String():  {ID: actorID.String(), Username: "owner", Status: "disabled", IsPlatformStaff: true, PlatformRole: PlatformRoleOwner},
			targetID.String(): {ID: targetID.String(), Username: "alice", Status: "active"},
		},
		sanctions: map[string]UserSanction{
			sanctionID: {
				ID:        sanctionID,
				UserID:    targetID.String(),
				Type:      UserSanctionTypeAccountBan,
				Status:    UserSanctionStatusActive,
				Reason:    "abuse",
				CreatedBy: actorID.String(),
				StartsAt:  now.Add(-time.Hour),
				CreatedAt: now.Add(-time.Hour),
				UpdatedAt: now.Add(-time.Hour),
			},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return now })

	_, err := uc.RevokeUserSanction(context.Background(), RevokeUserSanctionInput{
		ActorID:    actorID,
		SanctionID: sanctionID,
	})
	if !apperr.IsCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if repo.revokedSanctionID != "" {
		t.Fatalf("sanction should not be revoked by inactive actor, got %q", repo.revokedSanctionID)
	}
}

func TestRevokeUserSanctionWritesAudit(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	sanctionID := userdomain.NewGeneratedUserID().String()
	repo := &fakeRepository{
		users: map[string]User{
			actorID.String():  {ID: actorID.String(), Username: "owner", Status: "active", IsPlatformStaff: true, PlatformRole: PlatformRoleOwner},
			targetID.String(): {ID: targetID.String(), Username: "alice", Status: "active"},
		},
		sanctions: map[string]UserSanction{
			sanctionID: {
				ID:        sanctionID,
				UserID:    targetID.String(),
				Type:      UserSanctionTypeAccountBan,
				Status:    UserSanctionStatusActive,
				Reason:    "abuse",
				CreatedBy: actorID.String(),
				StartsAt:  now.Add(-time.Hour),
				CreatedAt: now.Add(-time.Hour),
				UpdatedAt: now.Add(-time.Hour),
			},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return now })

	result, err := uc.RevokeUserSanction(context.Background(), RevokeUserSanctionInput{
		ActorID:    actorID,
		SanctionID: sanctionID,
	})
	if err != nil {
		t.Fatalf("RevokeUserSanction returned error: %v", err)
	}
	if result.Sanction.Status != UserSanctionStatusRevoked || result.Sanction.RevokedBy != actorID.String() || result.Sanction.RevokedAt == nil {
		t.Fatalf("unexpected revoked sanction: %#v", result.Sanction)
	}
	if len(repo.auditLogs) != 1 || repo.auditLogs[0].Action != "admin.users.revoke_sanction" {
		t.Fatalf("expected revoke audit log, got %#v", repo.auditLogs)
	}
}

type fakeRepository struct {
	isStaff                bool
	noActiveCommunityOwner bool
	users                  map[string]User
	passwordHashes         map[string]userdomain.PasswordHash
	ownerTransfers         map[string]OwnerTransfer
	communities            map[string]Community
	effects                map[string]Effect
	settings               map[string]Setting
	pointAccounts          map[string]PointAccount
	pointTransactions      []PointTransaction
	createdSanction        CreateUserSanctionRecordInput
	sanctions              map[string]UserSanction
	revokedSanctionID      string
	auditLogs              []AuditLog
	txCount                int
}

func (f *fakeRepository) WithinTx(ctx context.Context, fn func(ctx context.Context, repository Repository) error) error {
	f.txCount++
	return fn(ctx, f)
}

func (f *fakeRepository) IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error) {
	return f.isStaff, nil
}

func (f *fakeRepository) ListUsers(ctx context.Context, status string, query string, limit int, offset int) ([]User, error) {
	users := make([]User, 0, len(f.users))
	for _, user := range f.users {
		if (status == "all" || user.Status == status) && matchesAdminUserQuery(user, query) {
			users = append(users, user)
		}
	}
	return paginateFake(users, limit, offset), nil
}

func (f *fakeRepository) FindUserByID(ctx context.Context, userID userdomain.UserID) (User, error) {
	user, ok := f.users[userID.String()]
	if !ok {
		return User{}, apperr.New(apperr.CodeNotFound, "user not found")
	}
	return user, nil
}

func (f *fakeRepository) FindUserPasswordHash(ctx context.Context, userID userdomain.UserID) (userdomain.PasswordHash, error) {
	if f.passwordHashes != nil {
		if hash, ok := f.passwordHashes[userID.String()]; ok {
			return hash, nil
		}
	}
	if _, ok := f.users[userID.String()]; !ok {
		return "", apperr.New(apperr.CodeNotFound, "user not found")
	}
	return userdomain.PasswordHash("correct-password"), nil
}

func (f *fakeRepository) UpdateUser(ctx context.Context, userID userdomain.UserID, input UpdateUserRecordInput) (User, error) {
	user, ok := f.users[userID.String()]
	if !ok {
		return User{}, apperr.New(apperr.CodeNotFound, "user not found")
	}
	user.Status = input.Status
	user.IsPlatformStaff = input.IsPlatformStaff
	if input.IsPlatformStaff && user.PlatformRole == "" {
		user.PlatformRole = PlatformRoleStaff
	}
	if !input.IsPlatformStaff {
		user.PlatformRole = ""
	}
	user.UpdatedAt = input.UpdatedAt
	f.users[userID.String()] = user
	return user, nil
}

func (f *fakeRepository) UpdateUserPlatformRole(ctx context.Context, userID userdomain.UserID, role string, updatedAt time.Time) (User, error) {
	user, ok := f.users[userID.String()]
	if !ok {
		return User{}, apperr.New(apperr.CodeNotFound, "user not found")
	}
	user.PlatformRole = role
	user.IsPlatformStaff = role != ""
	user.UpdatedAt = updatedAt
	f.users[userID.String()] = user
	return user, nil
}

func (f *fakeRepository) CountPlatformOwners(ctx context.Context) (int, error) {
	count := 0
	for _, user := range f.users {
		if user.Status == "active" && user.PlatformRole == PlatformRoleOwner {
			count++
		}
	}
	return count, nil
}

func (f *fakeRepository) FindCurrentOwnerTransfer(ctx context.Context, now time.Time) (OwnerTransfer, error) {
	for id, transfer := range f.ownerTransfers {
		if transfer.Status == OwnerTransferStatusPending && !transfer.ExpiresAt.After(now) {
			transfer.Status = OwnerTransferStatusExpired
			transfer.UpdatedAt = now
			f.ownerTransfers[id] = transfer
		}
		if transfer.Status == OwnerTransferStatusPending {
			return transfer, nil
		}
	}
	return OwnerTransfer{}, apperr.New(apperr.CodeNotFound, "owner transfer not found")
}

func (f *fakeRepository) FindOwnerTransferByID(ctx context.Context, transferID string, now time.Time) (OwnerTransfer, error) {
	transfer, ok := f.ownerTransfers[transferID]
	if !ok {
		return OwnerTransfer{}, apperr.New(apperr.CodeNotFound, "owner transfer not found")
	}
	if transfer.Status == OwnerTransferStatusPending && !transfer.ExpiresAt.After(now) {
		transfer.Status = OwnerTransferStatusExpired
		transfer.UpdatedAt = now
		f.ownerTransfers[transferID] = transfer
	}
	return transfer, nil
}

func (f *fakeRepository) CreateOwnerTransfer(ctx context.Context, input CreateOwnerTransferRecordInput) (OwnerTransfer, error) {
	if f.ownerTransfers == nil {
		f.ownerTransfers = make(map[string]OwnerTransfer)
	}
	for _, transfer := range f.ownerTransfers {
		if transfer.Status == OwnerTransferStatusPending && transfer.ExpiresAt.After(input.CreatedAt) {
			return OwnerTransfer{}, apperr.New(apperr.CodeConflict, "owner transfer already pending")
		}
	}
	initiator := f.users[input.InitiatedByID.String()]
	target := f.users[input.TargetUserID.String()]
	transfer := OwnerTransfer{
		ID:                  input.ID,
		Status:              OwnerTransferStatusPending,
		InitiatedByID:       input.InitiatedByID.String(),
		InitiatedByUsername: initiator.Username,
		TargetUserID:        input.TargetUserID.String(),
		TargetUsername:      target.Username,
		PreviousOwnerRole:   input.PreviousOwnerRole,
		Reason:              input.Reason,
		CreatedAt:           input.CreatedAt,
		UpdatedAt:           input.CreatedAt,
		ExpiresAt:           input.ExpiresAt,
	}
	f.ownerTransfers[input.ID] = transfer
	return transfer, nil
}

func (f *fakeRepository) CancelOwnerTransfer(ctx context.Context, transferID string, cancelledAt time.Time) (OwnerTransfer, error) {
	transfer, ok := f.ownerTransfers[transferID]
	if !ok {
		return OwnerTransfer{}, apperr.New(apperr.CodeNotFound, "owner transfer not found")
	}
	if transfer.Status != OwnerTransferStatusPending || !transfer.ExpiresAt.After(cancelledAt) {
		return OwnerTransfer{}, apperr.New(apperr.CodeConflict, "owner transfer is not pending")
	}
	transfer.Status = OwnerTransferStatusCancelled
	transfer.CancelledAt = &cancelledAt
	transfer.UpdatedAt = cancelledAt
	f.ownerTransfers[transferID] = transfer
	return transfer, nil
}

func (f *fakeRepository) AcceptOwnerTransfer(ctx context.Context, transferID string, acceptedAt time.Time) (OwnerTransfer, error) {
	transfer, ok := f.ownerTransfers[transferID]
	if !ok {
		return OwnerTransfer{}, apperr.New(apperr.CodeNotFound, "owner transfer not found")
	}
	if transfer.Status != OwnerTransferStatusPending || !transfer.ExpiresAt.After(acceptedAt) {
		return OwnerTransfer{}, apperr.New(apperr.CodeConflict, "owner transfer is not pending")
	}
	for id, user := range f.users {
		if user.Status == "active" && user.PlatformRole == PlatformRoleOwner && id != transfer.TargetUserID && id != transfer.InitiatedByID {
			user.PlatformRole = ""
			user.IsPlatformStaff = false
			user.UpdatedAt = acceptedAt
			f.users[id] = user
		}
	}
	initiator := f.users[transfer.InitiatedByID]
	initiator.PlatformRole = transfer.PreviousOwnerRole
	initiator.IsPlatformStaff = transfer.PreviousOwnerRole != ""
	initiator.UpdatedAt = acceptedAt
	f.users[transfer.InitiatedByID] = initiator

	target := f.users[transfer.TargetUserID]
	target.PlatformRole = PlatformRoleOwner
	target.IsPlatformStaff = true
	target.UpdatedAt = acceptedAt
	f.users[transfer.TargetUserID] = target

	transfer.Status = OwnerTransferStatusAccepted
	transfer.AcceptedAt = &acceptedAt
	transfer.UpdatedAt = acceptedAt
	f.ownerTransfers[transferID] = transfer
	return transfer, nil
}

func (f *fakeRepository) BootstrapOwner(ctx context.Context, input BootstrapOwnerRecordInput) (User, error) {
	user, ok := f.users[input.UserID.String()]
	if !ok {
		return User{}, apperr.New(apperr.CodeNotFound, "user not found")
	}
	user.PlatformRole = PlatformRoleOwner
	user.IsPlatformStaff = true
	user.UpdatedAt = input.UpdatedAt
	f.users[input.UserID.String()] = user
	return user, nil
}

func (f *fakeRepository) RecoverOwner(ctx context.Context, input RecoverOwnerRecordInput) (OwnerRecoveryRecordResult, error) {
	compromised := f.users[input.CompromisedUserID.String()]
	compromised.PlatformRole = ""
	compromised.IsPlatformStaff = false
	if input.DisableCompromised {
		compromised.Status = "disabled"
	}
	compromised.UpdatedAt = input.UpdatedAt
	f.users[input.CompromisedUserID.String()] = compromised

	newOwner := f.users[input.NewOwnerID.String()]
	newOwner.PlatformRole = PlatformRoleOwner
	newOwner.IsPlatformStaff = true
	newOwner.UpdatedAt = input.UpdatedAt
	f.users[input.NewOwnerID.String()] = newOwner
	return OwnerRecoveryRecordResult{NewOwner: newOwner, CompromisedUser: compromised}, nil
}

func (f *fakeRepository) ListCommunities(ctx context.Context, status string, query string, limit int, offset int) ([]Community, error) {
	communities := make([]Community, 0, len(f.communities))
	for _, community := range f.communities {
		if (status == "all" || community.Status == status) && matchesAdminCommunityQuery(community, query) {
			communities = append(communities, community)
		}
	}
	return paginateFake(communities, limit, offset), nil
}

func (f *fakeRepository) FindCommunityByID(ctx context.Context, communityID communitydomain.CommunityID) (Community, error) {
	community, ok := f.communities[communityID.String()]
	if !ok {
		return Community{}, apperr.New(apperr.CodeNotFound, "community not found")
	}
	return community, nil
}

func (f *fakeRepository) UpdateCommunityStatus(ctx context.Context, communityID communitydomain.CommunityID, status communitydomain.CommunityStatus, updatedAt time.Time) (Community, error) {
	community, ok := f.communities[communityID.String()]
	if !ok {
		return Community{}, apperr.New(apperr.CodeNotFound, "community not found")
	}
	community.Status = status.String()
	community.UpdatedAt = updatedAt
	f.communities[communityID.String()] = community
	return community, nil
}

func (f *fakeRepository) TransferCommunityOwner(ctx context.Context, communityID communitydomain.CommunityID, newOwnerID userdomain.UserID, updatedAt time.Time) (CommunityOwnerChange, error) {
	user, ok := f.users[newOwnerID.String()]
	if !ok || user.Status != "active" {
		return CommunityOwnerChange{}, apperr.New(apperr.CodeNotFound, "user not found")
	}
	var beforeOwner *CommunityOwnerMember
	if !f.noActiveCommunityOwner {
		beforeOwner = &CommunityOwnerMember{
			UserID:    "previous-owner",
			Username:  "previous",
			Role:      "owner",
			Status:    "active",
			UpdatedAt: updatedAt.Add(-time.Minute),
		}
	}
	return CommunityOwnerChange{
		BeforeOwner: beforeOwner,
		AfterOwner: CommunityOwnerMember{
			UserID:    user.ID,
			Username:  user.Username,
			Role:      "owner",
			Status:    "active",
			UpdatedAt: updatedAt,
		},
	}, nil
}

func (f *fakeRepository) ListEffects(ctx context.Context, active *bool, limit int, offset int) ([]Effect, error) {
	effects := make([]Effect, 0, len(f.effects))
	for _, effect := range f.effects {
		if active == nil || effect.IsActive == *active {
			effects = append(effects, effect)
		}
	}
	return effects, nil
}

func (f *fakeRepository) FindEffectByID(ctx context.Context, effectID string) (Effect, error) {
	effect, ok := f.effects[effectID]
	if !ok {
		return Effect{}, apperr.New(apperr.CodeNotFound, "effect not found")
	}
	return effect, nil
}

func (f *fakeRepository) UpdateEffectActive(ctx context.Context, effectID string, active bool, updatedAt time.Time) (Effect, error) {
	effect, ok := f.effects[effectID]
	if !ok {
		return Effect{}, apperr.New(apperr.CodeNotFound, "effect not found")
	}
	effect.IsActive = active
	effect.UpdatedAt = updatedAt
	f.effects[effectID] = effect
	return effect, nil
}

func (f *fakeRepository) ListSettings(ctx context.Context) ([]Setting, error) {
	settingsRows := make([]Setting, 0, len(f.settings))
	for _, setting := range f.settings {
		settingsRows = append(settingsRows, setting)
	}
	return settingsRows, nil
}

func (f *fakeRepository) FindSettingByKey(ctx context.Context, key string) (Setting, error) {
	setting, ok := f.settings[key]
	if !ok {
		return Setting{}, apperr.New(apperr.CodeNotFound, "admin setting not found")
	}
	return setting, nil
}

func (f *fakeRepository) SetSetting(ctx context.Context, key string, enabled bool, updatedBy userdomain.UserID, updatedAt time.Time) (Setting, error) {
	if _, ok := f.settings[key]; !ok {
		return Setting{}, apperr.New(apperr.CodeNotFound, "admin setting not found")
	}
	setting := Setting{
		Key:       key,
		Enabled:   enabled,
		UpdatedBy: updatedBy.String(),
		UpdatedAt: updatedAt,
	}
	f.settings[key] = setting
	return setting, nil
}

func (f *fakeRepository) ListPointTransactions(ctx context.Context, userID *userdomain.UserID, limit int, offset int) ([]PointTransaction, error) {
	filtered := make([]PointTransaction, 0, len(f.pointTransactions))
	for _, transaction := range f.pointTransactions {
		if userID == nil || transaction.UserID == userID.String() {
			filtered = append(filtered, transaction)
		}
	}
	if offset >= len(filtered) {
		return []PointTransaction{}, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], nil
}

func (f *fakeRepository) AdjustUserPoints(ctx context.Context, input AdjustUserPointsRecordInput) (AdjustUserPointsRecordResult, error) {
	if f.pointAccounts == nil {
		f.pointAccounts = map[string]PointAccount{}
	}
	account, ok := f.pointAccounts[input.UserID.String()]
	if !ok {
		account = PointAccount{
			UserID: input.UserID.String(),
		}
	}
	if account.Balance+input.Delta < 0 {
		return AdjustUserPointsRecordResult{}, apperr.New(apperr.CodeForbidden, "insufficient points")
	}
	account.Balance += input.Delta
	if input.Delta > 0 {
		account.LifetimeEarned += input.Delta
	} else {
		account.LifetimeSpent += -input.Delta
	}
	account.UpdatedAt = input.CreatedAt
	f.pointAccounts[input.UserID.String()] = account

	transaction := PointTransaction{
		ID:           input.TransactionID,
		UserID:       input.UserID.String(),
		Delta:        input.Delta,
		BalanceAfter: account.Balance,
		Reason:       input.Reason,
		SourceType:   "admin_adjustment",
		SourceID:     input.ActorID.String(),
		CreatedAt:    input.CreatedAt,
	}
	f.pointTransactions = append(f.pointTransactions, transaction)
	return AdjustUserPointsRecordResult{Account: account, Transaction: transaction}, nil
}

func (f *fakeRepository) CreateUserSanction(ctx context.Context, input CreateUserSanctionRecordInput) (UserSanction, error) {
	f.createdSanction = input
	return UserSanction{
		ID:        input.ID,
		UserID:    input.UserID.String(),
		Type:      input.Type,
		Status:    UserSanctionStatusActive,
		Reason:    input.Reason,
		CreatedBy: input.CreatedBy.String(),
		StartsAt:  input.StartsAt,
		ExpiresAt: input.ExpiresAt,
		CreatedAt: input.CreatedAt,
		UpdatedAt: input.CreatedAt,
	}, nil
}

func (f *fakeRepository) ListUserSanctions(ctx context.Context, userID userdomain.UserID, limit int, offset int, now time.Time) ([]UserSanction, error) {
	return []UserSanction{}, nil
}

func (f *fakeRepository) FindUserSanctionByID(ctx context.Context, sanctionID string, now time.Time) (UserSanction, error) {
	if f.sanctions != nil {
		if sanction, ok := f.sanctions[sanctionID]; ok {
			return sanction, nil
		}
	}
	return UserSanction{}, apperr.New(apperr.CodeNotFound, "user sanction not found")
}

func (f *fakeRepository) RevokeUserSanction(ctx context.Context, sanctionID string, actorID userdomain.UserID, revokedAt time.Time) (UserSanction, error) {
	sanction, ok := f.sanctions[sanctionID]
	if !ok {
		return UserSanction{}, apperr.New(apperr.CodeConflict, "user sanction is not active")
	}
	sanction.Status = UserSanctionStatusRevoked
	sanction.RevokedBy = actorID.String()
	sanction.RevokedAt = &revokedAt
	sanction.UpdatedAt = revokedAt
	f.sanctions[sanctionID] = sanction
	f.revokedSanctionID = sanctionID
	return sanction, nil
}

func (f *fakeRepository) CreateAuditLog(ctx context.Context, log AuditLog) error {
	if log.ID == "" {
		return errors.New("audit log id is required")
	}
	f.auditLogs = append(f.auditLogs, log)
	return nil
}

func (f *fakeRepository) ListAuditLogs(ctx context.Context, targetType string, targetID string, query string, limit int, offset int) ([]AuditLog, error) {
	logs := make([]AuditLog, 0, len(f.auditLogs))
	for _, log := range f.auditLogs {
		if targetType != "" && log.TargetType != targetType {
			continue
		}
		if targetID != "" && log.TargetID != targetID {
			continue
		}
		if !matchesAdminAuditLogQuery(log, query) {
			continue
		}
		logs = append(logs, log)
	}
	return paginateFake(logs, limit, offset), nil
}

func matchesAdminUserQuery(user User, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(user.ID), query) ||
		strings.Contains(strings.ToLower(user.Username), query)
}

func matchesAdminCommunityQuery(community Community, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(community.ID), query) ||
		strings.Contains(strings.ToLower(community.Slug), query) ||
		strings.Contains(strings.ToLower(community.Name), query) ||
		strings.Contains(strings.ToLower(community.Description), query) ||
		strings.Contains(strings.ToLower(community.CreatedBy), query)
}

func matchesAdminAuditLogQuery(log AuditLog, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(log.ID), query) ||
		strings.Contains(strings.ToLower(log.ActorID), query) ||
		strings.Contains(strings.ToLower(log.Action), query) ||
		strings.Contains(strings.ToLower(log.TargetType), query) ||
		strings.Contains(strings.ToLower(log.TargetID), query)
}

func paginateFake[T any](items []T, limit int, offset int) []T {
	if offset >= len(items) {
		return []T{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

type fakePasswordComparer struct{}

func (fakePasswordComparer) Compare(hash userdomain.PasswordHash, plain userdomain.PlainPassword) error {
	if hash.Raw() != plain.String() {
		return errors.New("password mismatch")
	}
	return nil
}

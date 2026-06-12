package communityrepository

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communityusecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCommunityRepositoryCreateFindListAndConflict(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresCommunityRepository(pool)
	now := testNow()

	activeSlug := mustCommunitySlug(t, "repo-"+randomSuffix())
	activeCommunity := mustSystemCommunity(t, activeSlug, now)
	if err := repo.Create(ctx, *activeCommunity); err != nil {
		t.Fatalf("Create active community returned error: %v", err)
	}
	cleanupCommunity(ctx, t, pool, activeCommunity.ID())

	got, err := repo.FindBySlug(ctx, activeSlug)
	if err != nil {
		t.Fatalf("FindBySlug returned error: %v", err)
	}
	assertSameCommunity(t, got, activeCommunity)

	if _, err := repo.FindBySlug(ctx, mustCommunitySlug(t, "missing-"+randomSuffix())); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing community, got %v", err)
	}

	duplicateCommunity := mustSystemCommunity(t, activeSlug, now)
	if err := repo.Create(ctx, *duplicateCommunity); !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for duplicate community slug, got %v", err)
	}

	suspendedSlug := mustCommunitySlug(t, "repo-"+randomSuffix())
	suspendedCommunity := mustCommunity(t, suspendedSlug, communitydomain.CommunityStatusSuspended, now)
	if err := repo.Create(ctx, *suspendedCommunity); err != nil {
		t.Fatalf("Create suspended community returned error: %v", err)
	}
	cleanupCommunity(ctx, t, pool, suspendedCommunity.ID())

	communities, err := repo.ListActivePublic(ctx)
	if err != nil {
		t.Fatalf("ListActivePublic returned error: %v", err)
	}
	if !containsCommunitySlug(communities, activeSlug) {
		t.Fatalf("expected active community %q in list", activeSlug.String())
	}
	if containsCommunitySlug(communities, suspendedSlug) {
		t.Fatalf("did not expect suspended community %q in list", suspendedSlug.String())
	}
}

func TestPostgresMembershipRepositoryCreateAndConflict(t *testing.T) {
	ctx, pool := newTestPool(t)
	communityRepo := NewPostgresCommunityRepository(pool)
	membershipRepo := NewPostgresMembershipRepository(pool)
	now := testNow()

	ownerID := insertTestUser(ctx, t, pool, false)
	community := mustUserCreatedCommunity(t, mustCommunitySlug(t, "repo-"+randomSuffix()), ownerID, now)
	if err := communityRepo.Create(ctx, *community); err != nil {
		t.Fatalf("Create community returned error: %v", err)
	}
	cleanupCommunity(ctx, t, pool, community.ID())

	membership, err := communitydomain.NewCommunityMembership(community.ID(), ownerID, communitydomain.MembershipRoleOwner, now)
	if err != nil {
		t.Fatalf("NewCommunityMembership returned error: %v", err)
	}

	if err := membershipRepo.Create(ctx, *membership); err != nil {
		t.Fatalf("Create membership returned error: %v", err)
	}
	cleanupMembership(ctx, t, pool, community.ID(), ownerID)

	if err := membershipRepo.Create(ctx, *membership); !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for duplicate membership, got %v", err)
	}
}

func TestPostgresMembershipRepositoryFindActiveRolesByUser(t *testing.T) {
	ctx, pool := newTestPool(t)
	communityRepo := NewPostgresCommunityRepository(pool)
	membershipRepo := NewPostgresMembershipRepository(pool)
	now := testNow()

	viewerID := insertTestUser(ctx, t, pool, false)
	otherID := insertTestUser(ctx, t, pool, false)

	ownerCommunity := mustUserCreatedCommunity(t, mustCommunitySlug(t, "repo-"+randomSuffix()), viewerID, now)
	moderatorCommunity := mustUserCreatedCommunity(t, mustCommunitySlug(t, "repo-"+randomSuffix()), otherID, now)
	inactiveCommunity := mustUserCreatedCommunity(t, mustCommunitySlug(t, "repo-"+randomSuffix()), otherID, now)
	for _, community := range []*communitydomain.Community{ownerCommunity, moderatorCommunity, inactiveCommunity} {
		if err := communityRepo.Create(ctx, *community); err != nil {
			t.Fatalf("Create community returned error: %v", err)
		}
		cleanupCommunity(ctx, t, pool, community.ID())
	}

	memberships := []*communitydomain.CommunityMembership{
		mustMembership(t, ownerCommunity.ID(), viewerID, communitydomain.MembershipRoleOwner, communitydomain.MembershipStatusActive, now),
		mustMembership(t, moderatorCommunity.ID(), viewerID, communitydomain.MembershipRoleModerator, communitydomain.MembershipStatusActive, now),
		mustMembership(t, inactiveCommunity.ID(), viewerID, communitydomain.MembershipRoleMember, communitydomain.MembershipStatusLeft, now),
		mustMembership(t, inactiveCommunity.ID(), otherID, communitydomain.MembershipRoleOwner, communitydomain.MembershipStatusActive, now),
	}
	for _, membership := range memberships {
		if err := membershipRepo.Create(ctx, *membership); err != nil {
			t.Fatalf("Create membership returned error: %v", err)
		}
		cleanupMembership(ctx, t, pool, membership.CommunityID(), membership.UserID())
	}

	roles, err := membershipRepo.FindActiveRolesByUser(
		ctx,
		[]communitydomain.CommunityID{ownerCommunity.ID(), moderatorCommunity.ID(), inactiveCommunity.ID()},
		viewerID,
	)
	if err != nil {
		t.Fatalf("FindActiveRolesByUser returned error: %v", err)
	}
	if got := roles[ownerCommunity.ID()]; got != communitydomain.MembershipRoleOwner {
		t.Fatalf("expected owner role, got %q", got.String())
	}
	if got := roles[moderatorCommunity.ID()]; got != communitydomain.MembershipRoleModerator {
		t.Fatalf("expected moderator role, got %q", got.String())
	}
	if _, ok := roles[inactiveCommunity.ID()]; ok {
		t.Fatal("did not expect inactive membership role to be returned")
	}
}

func TestPostgresMembershipRepositoryListActiveMembers(t *testing.T) {
	ctx, pool := newTestPool(t)
	communityRepo := NewPostgresCommunityRepository(pool)
	membershipRepo := NewPostgresMembershipRepository(pool)
	now := testNow()

	ownerID := insertTestUser(ctx, t, pool, false)
	moderatorID := insertTestUser(ctx, t, pool, false)
	memberID := insertTestUser(ctx, t, pool, false)
	leftID := insertTestUser(ctx, t, pool, false)
	updateTestUserProfile(ctx, t, pool, ownerID, "Owner Alice", "https://example.com/owner.jpg", "Owner headline")

	community := mustUserCreatedCommunity(t, mustCommunitySlug(t, "repo-"+randomSuffix()), ownerID, now)
	if err := communityRepo.Create(ctx, *community); err != nil {
		t.Fatalf("Create community returned error: %v", err)
	}
	cleanupCommunity(ctx, t, pool, community.ID())

	memberships := []*communitydomain.CommunityMembership{
		mustMembership(t, community.ID(), memberID, communitydomain.MembershipRoleMember, communitydomain.MembershipStatusActive, now.Add(time.Minute)),
		mustMembership(t, community.ID(), moderatorID, communitydomain.MembershipRoleModerator, communitydomain.MembershipStatusActive, now.Add(2*time.Minute)),
		mustMembership(t, community.ID(), ownerID, communitydomain.MembershipRoleOwner, communitydomain.MembershipStatusActive, now.Add(3*time.Minute)),
		mustMembership(t, community.ID(), leftID, communitydomain.MembershipRoleMember, communitydomain.MembershipStatusLeft, now.Add(4*time.Minute)),
	}
	for _, membership := range memberships {
		if err := membershipRepo.Create(ctx, *membership); err != nil {
			t.Fatalf("Create membership returned error: %v", err)
		}
		cleanupMembership(ctx, t, pool, membership.CommunityID(), membership.UserID())
	}

	members, err := membershipRepo.ListActiveMembers(ctx, community.ID(), 20, 0)
	if err != nil {
		t.Fatalf("ListActiveMembers returned error: %v", err)
	}
	if len(members) != 3 {
		t.Fatalf("expected 3 active members, got %d", len(members))
	}

	expected := []struct {
		userID userdomain.UserID
		role   string
	}{
		{userID: ownerID, role: communitydomain.MembershipRoleOwner.String()},
		{userID: moderatorID, role: communitydomain.MembershipRoleModerator.String()},
		{userID: memberID, role: communitydomain.MembershipRoleMember.String()},
	}
	for index, want := range expected {
		if members[index].UserID != want.userID.String() {
			t.Fatalf("member %d user id = %q, want %q", index, members[index].UserID, want.userID.String())
		}
		if members[index].Role != want.role {
			t.Fatalf("member %d role = %q, want %q", index, members[index].Role, want.role)
		}
		if members[index].Status != communitydomain.MembershipStatusActive.String() {
			t.Fatalf("member %d status = %q, want active", index, members[index].Status)
		}
		if strings.TrimSpace(members[index].Username) == "" {
			t.Fatalf("member %d username is empty", index)
		}
	}
	if members[0].DisplayName != "Owner Alice" || members[0].AvatarURL != "https://example.com/owner.jpg" || members[0].Headline != "Owner headline" {
		t.Fatalf("expected owner profile fields, got %#v", members[0])
	}
}

func TestPostgresCommunityRepositoryUpdateDetails(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresCommunityRepository(pool)
	now := testNow()

	community := mustSystemCommunity(t, mustCommunitySlug(t, "repo-"+randomSuffix()), now)
	if err := repo.Create(ctx, *community); err != nil {
		t.Fatalf("Create community returned error: %v", err)
	}
	cleanupCommunity(ctx, t, pool, community.ID())

	updatedAt := now.Add(time.Hour)
	if err := community.UpdateDetails(mustCommunityName(t, "Updated Community"), communitydomain.NewCommunityDescription("updated description"), updatedAt); err != nil {
		t.Fatalf("UpdateDetails domain returned error: %v", err)
	}
	if err := repo.UpdateDetails(ctx, *community); err != nil {
		t.Fatalf("UpdateDetails returned error: %v", err)
	}

	got, err := repo.FindByID(ctx, community.ID())
	if err != nil {
		t.Fatalf("FindByID after update returned error: %v", err)
	}
	if got.Name().String() != "Updated Community" || got.Description().String() != "updated description" || !got.UpdatedAt().Equal(updatedAt) {
		t.Fatalf("unexpected updated community: name=%q description=%q updated=%s", got.Name().String(), got.Description().String(), got.UpdatedAt())
	}

	missing := mustSystemCommunity(t, mustCommunitySlug(t, "repo-"+randomSuffix()), now)
	if err := repo.UpdateDetails(ctx, *missing); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing community update, got %v", err)
	}
}

func TestPostgresCommunityRepositoryRuleCRUD(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresCommunityRepository(pool)
	now := testNow()

	ownerID := insertTestUser(ctx, t, pool, false)
	updaterID := insertTestUser(ctx, t, pool, false)
	community := mustUserCreatedCommunity(t, mustCommunitySlug(t, "repo-"+randomSuffix()), ownerID, now)
	if err := repo.Create(ctx, *community); err != nil {
		t.Fatalf("Create community returned error: %v", err)
	}
	cleanupCommunity(ctx, t, pool, community.ID())

	firstRule := mustCommunityRule(t, community.ID(), ownerID, "Second rule", "created later but sorted second", 2, now.Add(time.Minute))
	secondRule := mustCommunityRule(t, community.ID(), ownerID, "First rule", "created earlier but sorted first", 1, now)
	for _, rule := range []*communitydomain.CommunityRule{firstRule, secondRule} {
		if err := repo.CreateRule(ctx, *rule); err != nil {
			t.Fatalf("CreateRule returned error: %v", err)
		}
		cleanupRule(ctx, t, pool, rule.ID())
	}

	rules, err := repo.ListRules(ctx, community.ID())
	if err != nil {
		t.Fatalf("ListRules returned error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected two rules, got %d", len(rules))
	}
	if rules[0].ID() != secondRule.ID() || rules[1].ID() != firstRule.ID() {
		t.Fatalf("rules not sorted by position: got [%q, %q]", rules[0].Title().String(), rules[1].Title().String())
	}

	got, err := repo.FindRuleByID(ctx, firstRule.ID())
	if err != nil {
		t.Fatalf("FindRuleByID returned error: %v", err)
	}
	if got.ID() != firstRule.ID() || got.Title().String() != "Second rule" {
		t.Fatalf("unexpected rule lookup result: id=%q title=%q", got.ID().String(), got.Title().String())
	}

	updatedTitle, err := communitydomain.NewCommunityRuleTitle("Updated rule")
	if err != nil {
		t.Fatalf("NewCommunityRuleTitle returned error: %v", err)
	}
	updatedPosition, err := communitydomain.NewCommunityRulePosition(0)
	if err != nil {
		t.Fatalf("NewCommunityRulePosition returned error: %v", err)
	}
	updatedAt := now.Add(2 * time.Hour)
	if err := firstRule.Update(updatedTitle, communitydomain.NewCommunityRuleBody("updated body"), updatedPosition, updaterID, updatedAt); err != nil {
		t.Fatalf("Update domain returned error: %v", err)
	}
	if err := repo.UpdateRule(ctx, *firstRule); err != nil {
		t.Fatalf("UpdateRule returned error: %v", err)
	}

	updated, err := repo.FindRuleByID(ctx, firstRule.ID())
	if err != nil {
		t.Fatalf("FindRuleByID updated returned error: %v", err)
	}
	if updated.Title().String() != "Updated rule" || updated.Body().String() != "updated body" || updated.Position().Int() != 0 || updated.UpdatedBy() != updaterID || !updated.UpdatedAt().Equal(updatedAt) {
		t.Fatalf("unexpected updated rule: title=%q body=%q position=%d updater=%q updated=%s", updated.Title().String(), updated.Body().String(), updated.Position().Int(), updated.UpdatedBy().String(), updated.UpdatedAt())
	}

	otherCommunity := mustUserCreatedCommunity(t, mustCommunitySlug(t, "repo-"+randomSuffix()), ownerID, now)
	if err := repo.Create(ctx, *otherCommunity); err != nil {
		t.Fatalf("Create other community returned error: %v", err)
	}
	cleanupCommunity(ctx, t, pool, otherCommunity.ID())
	if err := repo.DeleteRule(ctx, firstRule.ID(), otherCommunity.ID()); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for cross-community delete, got %v", err)
	}

	if err := repo.DeleteRule(ctx, firstRule.ID(), community.ID()); err != nil {
		t.Fatalf("DeleteRule returned error: %v", err)
	}
	if _, err := repo.FindRuleByID(ctx, firstRule.ID()); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found after delete, got %v", err)
	}

	if err := repo.CreateRule(ctx, *secondRule); !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for duplicate rule id, got %v", err)
	}
}

func TestPostgresApplicationRepositoryCreateFindSaveAndConflict(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresApplicationRepository(pool)
	now := testNow()

	applicantID := insertTestUser(ctx, t, pool, false)
	reviewerID := insertTestUser(ctx, t, pool, true)
	requestedSlug := mustCommunitySlug(t, "repo-"+randomSuffix())

	application := mustApplication(t, applicantID, requestedSlug, now)
	if err := repo.Create(ctx, *application); err != nil {
		t.Fatalf("Create application returned error: %v", err)
	}
	cleanupApplication(ctx, t, pool, application.ID())

	got, err := repo.FindByID(ctx, application.ID())
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if got.ID() != application.ID() || got.Status() != communitydomain.ApplicationStatusPending {
		t.Fatalf("expected pending application %q, got id=%q status=%q", application.ID().String(), got.ID().String(), got.Status().String())
	}

	duplicate := mustApplication(t, applicantID, requestedSlug, now)
	if err := repo.Create(ctx, *duplicate); !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for duplicate pending application slug, got %v", err)
	}

	reviewedAt := now.Add(time.Minute)
	if err := application.Approve(reviewerID, reviewedAt); err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}
	if err := repo.Save(ctx, *application); err != nil {
		t.Fatalf("Save approved application returned error: %v", err)
	}

	approved, err := repo.FindByID(ctx, application.ID())
	if err != nil {
		t.Fatalf("FindByID approved returned error: %v", err)
	}
	if approved.Status() != communitydomain.ApplicationStatusApproved {
		t.Fatalf("expected approved status, got %q", approved.Status().String())
	}
	if gotReviewerID, ok := approved.ReviewedBy(); !ok || gotReviewerID != reviewerID {
		t.Fatalf("expected reviewer %q, got %q present=%t", reviewerID.String(), gotReviewerID.String(), ok)
	}

	pendingApplication := mustApplication(t, applicantID, mustCommunitySlug(t, "repo-"+randomSuffix()), now.Add(2*time.Minute))
	if err := repo.Create(ctx, *pendingApplication); err != nil {
		t.Fatalf("Create second pending application returned error: %v", err)
	}
	cleanupApplication(ctx, t, pool, pendingApplication.ID())

	pendingApplications, err := repo.ListByStatus(ctx, communitydomain.ApplicationStatusPending, 20, 0)
	if err != nil {
		t.Fatalf("ListByStatus pending returned error: %v", err)
	}
	if !containsApplicationID(pendingApplications, pendingApplication.ID()) {
		t.Fatalf("expected pending application %q in pending list", pendingApplication.ID().String())
	}
	if containsApplicationID(pendingApplications, application.ID()) {
		t.Fatalf("did not expect approved application %q in pending list", application.ID().String())
	}

	approvedApplications, err := repo.ListByStatus(ctx, communitydomain.ApplicationStatusApproved, 20, 0)
	if err != nil {
		t.Fatalf("ListByStatus approved returned error: %v", err)
	}
	if !containsApplicationID(approvedApplications, application.ID()) {
		t.Fatalf("expected approved application %q in approved list", application.ID().String())
	}

	if _, err := repo.FindByID(ctx, communitydomain.NewGeneratedCommunityApplicationID()); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing application, got %v", err)
	}

	missing := mustApplication(t, applicantID, mustCommunitySlug(t, "repo-"+randomSuffix()), now)
	if err := missing.Approve(reviewerID, reviewedAt); err != nil {
		t.Fatalf("Approve missing application returned error: %v", err)
	}
	if err := repo.Save(ctx, *missing); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for saving missing application, got %v", err)
	}
}

func TestPostgresPlatformStaffRepository(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPlatformStaffRepository(pool)

	staffID := insertTestUser(ctx, t, pool, true)
	normalID := insertTestUser(ctx, t, pool, false)

	isStaff, err := repo.IsPlatformStaff(ctx, staffID)
	if err != nil {
		t.Fatalf("IsPlatformStaff staff returned error: %v", err)
	}
	if !isStaff {
		t.Fatal("expected staff user to be platform staff")
	}

	isStaff, err = repo.IsPlatformStaff(ctx, normalID)
	if err != nil {
		t.Fatalf("IsPlatformStaff normal user returned error: %v", err)
	}
	if isStaff {
		t.Fatal("expected normal user not to be platform staff")
	}

	if _, err := repo.IsPlatformStaff(ctx, userdomain.NewGeneratedUserID()); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing user, got %v", err)
	}
}

func TestPostgresCommunityTransactionManagerApprovesApplicationWithCommunityAndOwner(t *testing.T) {
	ctx, pool := newTestPool(t)
	manager := NewPostgresCommunityTransactionManager(pool)
	applicationRepo := NewPostgresApplicationRepository(pool)
	now := testNow()

	applicantID := insertTestUser(ctx, t, pool, false)
	reviewerID := insertTestUser(ctx, t, pool, true)
	application := mustApplication(t, applicantID, mustCommunitySlug(t, "repo-"+randomSuffix()), now)
	if err := applicationRepo.Create(ctx, *application); err != nil {
		t.Fatalf("Create application returned error: %v", err)
	}
	cleanupApplication(ctx, t, pool, application.ID())

	var createdCommunityID communitydomain.CommunityID
	if err := manager.WithinTx(ctx, func(txCtx context.Context, repositories communityusecase.CommunityRepositories) error {
		lockedApplication, err := repositories.Applications().FindByIDForUpdate(txCtx, application.ID())
		if err != nil {
			return err
		}
		reviewedAt := now.Add(time.Minute)
		if err := lockedApplication.Approve(reviewerID, reviewedAt); err != nil {
			return err
		}
		community, err := communitydomain.NewUserCreatedCommunity(
			communitydomain.NewGeneratedCommunityID(),
			lockedApplication.RequestedSlug(),
			lockedApplication.RequestedName(),
			communitydomain.NewCommunityDescription(""),
			lockedApplication.ApplicantID(),
			reviewedAt,
		)
		if err != nil {
			return err
		}
		membership, err := communitydomain.NewCommunityMembership(
			community.ID(),
			lockedApplication.ApplicantID(),
			communitydomain.MembershipRoleOwner,
			reviewedAt,
		)
		if err != nil {
			return err
		}
		if err := repositories.Applications().Save(txCtx, *lockedApplication); err != nil {
			return err
		}
		if err := repositories.Communities().Create(txCtx, *community); err != nil {
			return err
		}
		if err := repositories.Memberships().Create(txCtx, *membership); err != nil {
			return err
		}
		createdCommunityID = community.ID()
		return nil
	}); err != nil {
		t.Fatalf("WithinTx returned error: %v", err)
	}
	cleanupCommunity(ctx, t, pool, createdCommunityID)

	approved, err := applicationRepo.FindByID(ctx, application.ID())
	if err != nil {
		t.Fatalf("FindByID approved returned error: %v", err)
	}
	if approved.Status() != communitydomain.ApplicationStatusApproved {
		t.Fatalf("expected approved application, got %q", approved.Status().String())
	}
	if !membershipExists(ctx, t, pool, createdCommunityID, applicantID) {
		t.Fatal("expected owner membership to be created")
	}
}

func TestPostgresCommunityTransactionManagerRollsBackApprovalWhenCommunityCreateFails(t *testing.T) {
	ctx, pool := newTestPool(t)
	manager := NewPostgresCommunityTransactionManager(pool)
	communityRepo := NewPostgresCommunityRepository(pool)
	applicationRepo := NewPostgresApplicationRepository(pool)
	now := testNow()

	applicantID := insertTestUser(ctx, t, pool, false)
	reviewerID := insertTestUser(ctx, t, pool, true)
	requestedSlug := mustCommunitySlug(t, "repo-"+randomSuffix())
	existingCommunity := mustSystemCommunity(t, requestedSlug, now)
	if err := communityRepo.Create(ctx, *existingCommunity); err != nil {
		t.Fatalf("Create existing community returned error: %v", err)
	}
	cleanupCommunity(ctx, t, pool, existingCommunity.ID())

	application := mustApplication(t, applicantID, requestedSlug, now)
	if err := applicationRepo.Create(ctx, *application); err != nil {
		t.Fatalf("Create application returned error: %v", err)
	}
	cleanupApplication(ctx, t, pool, application.ID())

	err := manager.WithinTx(ctx, func(txCtx context.Context, repositories communityusecase.CommunityRepositories) error {
		lockedApplication, err := repositories.Applications().FindByIDForUpdate(txCtx, application.ID())
		if err != nil {
			return err
		}
		reviewedAt := now.Add(time.Minute)
		if err := lockedApplication.Approve(reviewerID, reviewedAt); err != nil {
			return err
		}
		conflictingCommunity, err := communitydomain.NewUserCreatedCommunity(
			communitydomain.NewGeneratedCommunityID(),
			lockedApplication.RequestedSlug(),
			lockedApplication.RequestedName(),
			communitydomain.NewCommunityDescription(""),
			lockedApplication.ApplicantID(),
			reviewedAt,
		)
		if err != nil {
			return err
		}
		if err := repositories.Applications().Save(txCtx, *lockedApplication); err != nil {
			return err
		}
		return repositories.Communities().Create(txCtx, *conflictingCommunity)
	})
	if !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict from duplicate community slug, got %v", err)
	}

	pending, err := applicationRepo.FindByID(ctx, application.ID())
	if err != nil {
		t.Fatalf("FindByID pending returned error: %v", err)
	}
	if pending.Status() != communitydomain.ApplicationStatusPending {
		t.Fatalf("expected transaction rollback to keep pending status, got %q", pending.Status().String())
	}
}

func newTestPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	pool, err := db.Open(ctx, testPostgresConfig())
	if err != nil {
		t.Skipf("skip repository integration test: open postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.Ping(ctx, pool); err != nil {
		t.Skipf("skip repository integration test: ping postgres: %v", err)
	}

	requireCommunitySchema(ctx, t, pool)

	return ctx, pool
}

func requireCommunitySchema(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	for _, table := range []string{"users", "communities", "community_memberships", "community_applications", "community_rules"} {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'public'
					AND table_name = $1
			)
		`, table).Scan(&exists); err != nil {
			t.Fatalf("check table %s exists: %v", table, err)
		}
		if !exists {
			t.Skipf("%s table does not exist; run go run ./cmd/migrate up before repository tests", table)
		}
	}

	var staffColumnExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
				AND table_name = 'users'
				AND column_name = 'is_platform_staff'
		)
	`).Scan(&staffColumnExists); err != nil {
		t.Fatalf("check users.is_platform_staff exists: %v", err)
	}
	if !staffColumnExists {
		t.Skip("users.is_platform_staff column does not exist; run go run ./cmd/migrate up before repository tests")
	}
}

func testPostgresConfig() config.PostgresConfig {
	return config.PostgresConfig{
		Host:            envString("POSTGRES_HOST", "localhost"),
		Port:            envInt("POSTGRES_PORT", 5432),
		User:            envString("POSTGRES_USER", "postgres"),
		Password:        envString("POSTGRES_PASSWORD", "postgres"),
		Database:        envString("POSTGRES_DATABASE", "cumt_nexus"),
		SSLMode:         envString("POSTGRES_SSL_MODE", "disable"),
		MaxConns:        5,
		MaxConnLifetime: time.Minute,
		MaxConnIdleTime: time.Minute,
	}
}

func envString(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func insertTestUser(ctx context.Context, t *testing.T, pool *pgxpool.Pool, isPlatformStaff bool) userdomain.UserID {
	t.Helper()

	id := userdomain.NewGeneratedUserID()
	username := "comm_repo_" + randomSuffix()

	_, err := pool.Exec(ctx, `
		INSERT INTO users (
			id,
			username,
			password_hash,
			status,
			is_platform_staff,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2, $3, 'active', $4, $5, $5)
	`, id.String(), username, "hashed-password-"+username, isPlatformStaff, testNow())
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup test user %q: %v", id.String(), err)
		}
	})

	return id
}

func updateTestUserProfile(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userID userdomain.UserID, displayName string, avatarURL string, headline string) {
	t.Helper()

	if _, err := pool.Exec(ctx, `
		UPDATE users
		SET display_name = $2, avatar_url = $3, headline = $4
		WHERE id = $1::uuid
	`, userID.String(), displayName, avatarURL, headline); err != nil {
		t.Fatalf("update test user profile: %v", err)
	}
}

func cleanupCommunity(ctx context.Context, t *testing.T, pool *pgxpool.Pool, id communitydomain.CommunityID) {
	t.Helper()

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM communities WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup community %q: %v", id.String(), err)
		}
	})
}

func cleanupMembership(ctx context.Context, t *testing.T, pool *pgxpool.Pool, communityID communitydomain.CommunityID, userID userdomain.UserID) {
	t.Helper()

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `
			DELETE FROM community_memberships
			WHERE community_id = $1::uuid
				AND user_id = $2::uuid
		`, communityID.String(), userID.String()); err != nil {
			t.Fatalf("cleanup membership community=%q user=%q: %v", communityID.String(), userID.String(), err)
		}
	})
}

func cleanupApplication(ctx context.Context, t *testing.T, pool *pgxpool.Pool, id communitydomain.CommunityApplicationID) {
	t.Helper()

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM community_applications WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup application %q: %v", id.String(), err)
		}
	})
}

func cleanupRule(ctx context.Context, t *testing.T, pool *pgxpool.Pool, id communitydomain.CommunityRuleID) {
	t.Helper()

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM community_rules WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup community rule %q: %v", id.String(), err)
		}
	})
}

func membershipExists(ctx context.Context, t *testing.T, pool *pgxpool.Pool, communityID communitydomain.CommunityID, userID userdomain.UserID) bool {
	t.Helper()

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM community_memberships
			WHERE community_id = $1::uuid
				AND user_id = $2::uuid
				AND role = 'owner'
				AND status = 'active'
		)
	`, communityID.String(), userID.String()).Scan(&exists); err != nil {
		t.Fatalf("check membership exists: %v", err)
	}
	return exists
}

func testNow() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

func randomSuffix() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
}

func mustCommunitySlug(t *testing.T, raw string) communitydomain.CommunitySlug {
	t.Helper()

	slug, err := communitydomain.NewCommunitySlug(raw)
	if err != nil {
		t.Fatalf("NewCommunitySlug returned error: %v", err)
	}
	return slug
}

func mustCommunityName(t *testing.T, raw string) communitydomain.CommunityName {
	t.Helper()

	name, err := communitydomain.NewCommunityName(raw)
	if err != nil {
		t.Fatalf("NewCommunityName returned error: %v", err)
	}
	return name
}

func mustSystemCommunity(t *testing.T, slug communitydomain.CommunitySlug, now time.Time) *communitydomain.Community {
	t.Helper()

	community, err := communitydomain.NewSystemCommunity(
		communitydomain.NewGeneratedCommunityID(),
		slug,
		mustCommunityName(t, "Test Community "+slug.String()),
		communitydomain.NewCommunityDescription("test community"),
		now,
	)
	if err != nil {
		t.Fatalf("NewSystemCommunity returned error: %v", err)
	}
	return community
}

func mustCommunity(t *testing.T, slug communitydomain.CommunitySlug, status communitydomain.CommunityStatus, now time.Time) *communitydomain.Community {
	t.Helper()

	community, err := communitydomain.RehydrateCommunity(
		communitydomain.NewGeneratedCommunityID(),
		slug,
		mustCommunityName(t, "Test Community "+slug.String()),
		communitydomain.NewCommunityDescription("test community"),
		communitydomain.CommunityKindSystem,
		status,
		communitydomain.CommunityVisibilityPublic,
		nil,
		now,
		now,
	)
	if err != nil {
		t.Fatalf("RehydrateCommunity returned error: %v", err)
	}
	return community
}

func mustUserCreatedCommunity(t *testing.T, slug communitydomain.CommunitySlug, createdBy userdomain.UserID, now time.Time) *communitydomain.Community {
	t.Helper()

	community, err := communitydomain.NewUserCreatedCommunity(
		communitydomain.NewGeneratedCommunityID(),
		slug,
		mustCommunityName(t, "Test Community "+slug.String()),
		communitydomain.NewCommunityDescription("test community"),
		createdBy,
		now,
	)
	if err != nil {
		t.Fatalf("NewUserCreatedCommunity returned error: %v", err)
	}
	return community
}

func mustMembership(
	t *testing.T,
	communityID communitydomain.CommunityID,
	userID userdomain.UserID,
	role communitydomain.MembershipRole,
	status communitydomain.MembershipStatus,
	now time.Time,
) *communitydomain.CommunityMembership {
	t.Helper()

	membership, err := communitydomain.RehydrateCommunityMembership(communityID, userID, role, status, now, now)
	if err != nil {
		t.Fatalf("RehydrateCommunityMembership returned error: %v", err)
	}
	return membership
}

func mustApplication(t *testing.T, applicantID userdomain.UserID, slug communitydomain.CommunitySlug, now time.Time) *communitydomain.CommunityApplication {
	t.Helper()

	reason, err := communitydomain.NewApplicationReason("Need a community for " + slug.String())
	if err != nil {
		t.Fatalf("NewApplicationReason returned error: %v", err)
	}

	application, err := communitydomain.NewCommunityApplication(
		communitydomain.NewGeneratedCommunityApplicationID(),
		applicantID,
		slug,
		mustCommunityName(t, "Application "+slug.String()),
		reason,
		now,
	)
	if err != nil {
		t.Fatalf("NewCommunityApplication returned error: %v", err)
	}
	return application
}

func mustCommunityRule(t *testing.T, communityID communitydomain.CommunityID, actorID userdomain.UserID, rawTitle string, rawBody string, position int, now time.Time) *communitydomain.CommunityRule {
	t.Helper()

	title, err := communitydomain.NewCommunityRuleTitle(rawTitle)
	if err != nil {
		t.Fatalf("NewCommunityRuleTitle returned error: %v", err)
	}
	rulePosition, err := communitydomain.NewCommunityRulePosition(position)
	if err != nil {
		t.Fatalf("NewCommunityRulePosition returned error: %v", err)
	}
	rule, err := communitydomain.NewCommunityRule(
		communitydomain.NewGeneratedCommunityRuleID(),
		communityID,
		title,
		communitydomain.NewCommunityRuleBody(rawBody),
		rulePosition,
		actorID,
		now,
	)
	if err != nil {
		t.Fatalf("NewCommunityRule returned error: %v", err)
	}
	return rule
}

func assertSameCommunity(t *testing.T, got *communitydomain.Community, want *communitydomain.Community) {
	t.Helper()

	if got.ID() != want.ID() {
		t.Fatalf("expected community id %q, got %q", want.ID().String(), got.ID().String())
	}
	if got.Slug() != want.Slug() {
		t.Fatalf("expected community slug %q, got %q", want.Slug().String(), got.Slug().String())
	}
	if got.Kind() != want.Kind() {
		t.Fatalf("expected community kind %q, got %q", want.Kind().String(), got.Kind().String())
	}
	if got.Status() != want.Status() {
		t.Fatalf("expected community status %q, got %q", want.Status().String(), got.Status().String())
	}
}

func containsCommunitySlug(communities []communitydomain.Community, slug communitydomain.CommunitySlug) bool {
	for _, community := range communities {
		if community.Slug() == slug {
			return true
		}
	}
	return false
}

func containsApplicationID(applications []communitydomain.CommunityApplication, id communitydomain.CommunityApplicationID) bool {
	for _, application := range applications {
		if application.ID() == id {
			return true
		}
	}
	return false
}

func hasAppCode(err error, code apperr.Code) bool {
	if err == nil {
		return false
	}

	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Code() == code
}

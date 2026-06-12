package communitydomain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

func TestCommunitySlug(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "normalizes trim and case", raw: " Go-Backend ", want: "go-backend"},
		{name: "allows max length", raw: strings.Repeat("a", 32), want: strings.Repeat("a", 32)},
		{name: "rejects empty", raw: "   ", wantErr: true},
		{name: "rejects too short", raw: "ab", wantErr: true},
		{name: "rejects too long", raw: strings.Repeat("a", 33), wantErr: true},
		{name: "rejects underscore", raw: "go_backend", wantErr: true},
		{name: "rejects leading hyphen", raw: "-backend", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug, err := NewCommunitySlug(tt.raw)
			if tt.wantErr {
				if !hasAppCode(err, apperr.CodeInvalidArgument) {
					t.Fatalf("expected invalid_argument, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewCommunitySlug returned error: %v", err)
			}
			if slug.String() != tt.want {
				t.Fatalf("expected slug %q, got %q", tt.want, slug.String())
			}
		})
	}
}

func TestCommunityValues(t *testing.T) {
	if _, err := NewCommunityID(uuid.NewString()); err != nil {
		t.Fatalf("NewCommunityID returned error: %v", err)
	}
	if _, err := NewCommunityID("not-a-uuid"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid community id, got %v", err)
	}

	name, err := NewCommunityName(" Campus Life ")
	if err != nil {
		t.Fatalf("NewCommunityName returned error: %v", err)
	}
	if name.String() != "Campus Life" {
		t.Fatalf("expected trimmed community name, got %q", name.String())
	}
	if _, err := NewCommunityName(" "); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for blank community name, got %v", err)
	}

	if description := NewCommunityDescription(" description "); description.String() != "description" {
		t.Fatalf("expected trimmed description, got %q", description.String())
	}

	assertCommunityKind(t, "system", CommunityKindSystem)
	assertCommunityKind(t, "user_created", CommunityKindUserCreated)
	if _, err := NewCommunityKind("private"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid community kind, got %v", err)
	}

	assertCommunityStatus(t, "active", CommunityStatusActive)
	assertCommunityStatus(t, "suspended", CommunityStatusSuspended)
	assertCommunityStatus(t, "archived", CommunityStatusArchived)
	if _, err := NewCommunityStatus("pending"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid community status, got %v", err)
	}

	visibility, err := NewCommunityVisibility(" public ")
	if err != nil {
		t.Fatalf("NewCommunityVisibility returned error: %v", err)
	}
	if visibility != CommunityVisibilityPublic {
		t.Fatalf("expected public visibility, got %q", visibility.String())
	}
	if _, err := NewCommunityVisibility("restricted"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid visibility, got %v", err)
	}
}

func TestCommunityCreationInvariants(t *testing.T) {
	now := time.Now().UTC()

	systemCommunity, err := NewSystemCommunity(mustCommunityID(t), mustCommunitySlug(t, "public"), mustCommunityName(t, "Public"), NewCommunityDescription(""), now)
	if err != nil {
		t.Fatalf("NewSystemCommunity returned error: %v", err)
	}
	if systemCommunity.Kind() != CommunityKindSystem {
		t.Fatalf("expected system community kind, got %q", systemCommunity.Kind().String())
	}
	if _, ok := systemCommunity.CreatedBy(); ok {
		t.Fatal("system community should not require created_by")
	}

	creatorID := mustUserID(t)
	userCommunity, err := NewUserCreatedCommunity(mustCommunityID(t), mustCommunitySlug(t, "campus-life"), mustCommunityName(t, "Campus Life"), NewCommunityDescription(""), creatorID, now)
	if err != nil {
		t.Fatalf("NewUserCreatedCommunity returned error: %v", err)
	}
	if gotCreatorID, ok := userCommunity.CreatedBy(); !ok || gotCreatorID != creatorID {
		t.Fatalf("expected user-created community creator %q, got %q present=%t", creatorID.String(), gotCreatorID.String(), ok)
	}

	if _, err := RehydrateCommunity(mustCommunityID(t), mustCommunitySlug(t, "go-backend"), mustCommunityName(t, "Go Backend"), NewCommunityDescription(""), CommunityKindUserCreated, CommunityStatusActive, CommunityVisibilityPublic, nil, now, now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for missing user-created community creator, got %v", err)
	}
	if _, err := NewSystemCommunity(mustCommunityID(t), mustCommunitySlug(t, "public"), mustCommunityName(t, "Public"), NewCommunityDescription(""), time.Time{}); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for zero community creation time, got %v", err)
	}
	if _, err := RehydrateCommunity(mustCommunityID(t), mustCommunitySlug(t, "public"), mustCommunityName(t, "Public"), NewCommunityDescription(""), CommunityKindSystem, CommunityStatusActive, CommunityVisibilityPublic, nil, now, now.Add(-time.Second)); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for updated_at before created_at, got %v", err)
	}

	updatedAt := now.Add(time.Minute)
	if err := systemCommunity.UpdateDetails(mustCommunityName(t, "Public Square"), NewCommunityDescription("general discussion"), updatedAt); err != nil {
		t.Fatalf("UpdateDetails returned error: %v", err)
	}
	if systemCommunity.Name().String() != "Public Square" || systemCommunity.Description().String() != "general discussion" || !systemCommunity.UpdatedAt().Equal(updatedAt) {
		t.Fatalf("unexpected updated community details: name=%q description=%q updated=%s", systemCommunity.Name().String(), systemCommunity.Description().String(), systemCommunity.UpdatedAt())
	}
	if err := systemCommunity.UpdateDetails(mustCommunityName(t, "Invalid"), NewCommunityDescription(""), now.Add(-time.Second)); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for update before created time, got %v", err)
	}
}

func TestCommunityRuleValuesAndLifecycle(t *testing.T) {
	if _, err := NewCommunityRuleID(uuid.NewString()); err != nil {
		t.Fatalf("NewCommunityRuleID returned error: %v", err)
	}
	if _, err := NewCommunityRuleID("not-a-uuid"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid rule id, got %v", err)
	}

	title, err := NewCommunityRuleTitle(" Be kind ")
	if err != nil {
		t.Fatalf("NewCommunityRuleTitle returned error: %v", err)
	}
	if title.String() != "Be kind" {
		t.Fatalf("expected trimmed title, got %q", title.String())
	}
	if _, err := NewCommunityRuleTitle(" "); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for blank title, got %v", err)
	}
	if body := NewCommunityRuleBody(" body "); body.String() != "body" {
		t.Fatalf("expected trimmed body, got %q", body.String())
	}
	if _, err := NewCommunityRulePosition(-1); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for negative position, got %v", err)
	}

	now := time.Now().UTC()
	actorID := mustUserID(t)
	position, err := NewCommunityRulePosition(1)
	if err != nil {
		t.Fatalf("NewCommunityRulePosition returned error: %v", err)
	}
	rule, err := NewCommunityRule(mustCommunityRuleID(t), mustCommunityID(t), title, NewCommunityRuleBody("Initial body"), position, actorID, now)
	if err != nil {
		t.Fatalf("NewCommunityRule returned error: %v", err)
	}
	if rule.CreatedBy() != actorID || rule.UpdatedBy() != actorID {
		t.Fatalf("expected actor to be creator/updater")
	}

	updaterID := mustUserID(t)
	updatedAt := now.Add(time.Minute)
	updatedTitle, err := NewCommunityRuleTitle("Stay on topic")
	if err != nil {
		t.Fatalf("NewCommunityRuleTitle returned error: %v", err)
	}
	updatedPosition, err := NewCommunityRulePosition(2)
	if err != nil {
		t.Fatalf("NewCommunityRulePosition returned error: %v", err)
	}
	if err := rule.Update(updatedTitle, NewCommunityRuleBody("Updated body"), updatedPosition, updaterID, updatedAt); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if rule.Title().String() != "Stay on topic" || rule.Body().String() != "Updated body" || rule.Position().Int() != 2 || rule.UpdatedBy() != updaterID || !rule.UpdatedAt().Equal(updatedAt) {
		t.Fatalf("unexpected updated rule: title=%q body=%q position=%d updater=%q updated=%s", rule.Title().String(), rule.Body().String(), rule.Position().Int(), rule.UpdatedBy().String(), rule.UpdatedAt())
	}
	if err := rule.Update(updatedTitle, NewCommunityRuleBody(""), updatedPosition, updaterID, now.Add(-time.Second)); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for update before created time, got %v", err)
	}
}

func TestMembershipValuesAndCreation(t *testing.T) {
	assertMembershipRole(t, "owner", MembershipRoleOwner)
	assertMembershipRole(t, "moderator", MembershipRoleModerator)
	assertMembershipRole(t, "member", MembershipRoleMember)
	if _, err := NewMembershipRole("admin"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid membership role, got %v", err)
	}

	assertMembershipStatus(t, "active", MembershipStatusActive)
	assertMembershipStatus(t, "left", MembershipStatusLeft)
	assertMembershipStatus(t, "banned", MembershipStatusBanned)
	if _, err := NewMembershipStatus("disabled"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid membership status, got %v", err)
	}

	now := time.Now().UTC()
	membership, err := NewCommunityMembership(mustCommunityID(t), mustUserID(t), MembershipRoleOwner, now)
	if err != nil {
		t.Fatalf("NewCommunityMembership returned error: %v", err)
	}
	if membership.Role() != MembershipRoleOwner {
		t.Fatalf("expected owner role, got %q", membership.Role().String())
	}
	if membership.Status() != MembershipStatusActive {
		t.Fatalf("new membership should default to active, got %q", membership.Status().String())
	}

	if _, err := NewCommunityMembership("", mustUserID(t), MembershipRoleMember, now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for missing community id, got %v", err)
	}
	if _, err := NewCommunityMembership(mustCommunityID(t), "", MembershipRoleMember, now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for missing user id, got %v", err)
	}
}

func TestCommunityApplicationValuesAndLifecycle(t *testing.T) {
	now := time.Now().UTC()
	application, err := NewCommunityApplication(mustCommunityApplicationID(t), mustUserID(t), mustCommunitySlug(t, "campus-life"), mustCommunityName(t, "Campus Life"), mustApplicationReason(t, "Need a campus board"), now)
	if err != nil {
		t.Fatalf("NewCommunityApplication returned error: %v", err)
	}
	if application.Status() != ApplicationStatusPending {
		t.Fatalf("new application should default to pending, got %q", application.Status().String())
	}
	if _, ok := application.ReviewedBy(); ok {
		t.Fatal("pending application should not have reviewer")
	}

	reviewerID := mustUserID(t)
	reviewedAt := now.Add(time.Minute)
	if err := application.Approve(reviewerID, reviewedAt); err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}
	if application.Status() != ApplicationStatusApproved {
		t.Fatalf("expected approved status, got %q", application.Status().String())
	}
	if gotReviewerID, ok := application.ReviewedBy(); !ok || gotReviewerID != reviewerID {
		t.Fatalf("expected reviewer %q, got %q present=%t", reviewerID.String(), gotReviewerID.String(), ok)
	}
	if _, ok := application.RejectReason(); ok {
		t.Fatal("approved application should not have reject reason")
	}
	if err := application.Approve(reviewerID, reviewedAt.Add(time.Minute)); !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict when approving non-pending application, got %v", err)
	}

	rejected := mustApplication(t, now)
	rejectReason, err := NewRejectReason("duplicate slug")
	if err != nil {
		t.Fatalf("NewRejectReason returned error: %v", err)
	}
	if err := rejected.Reject(reviewerID, reviewedAt, rejectReason); err != nil {
		t.Fatalf("Reject returned error: %v", err)
	}
	if gotReason, ok := rejected.RejectReason(); !ok || gotReason != rejectReason {
		t.Fatalf("expected reject reason %q, got %q present=%t", rejectReason.String(), gotReason.String(), ok)
	}

	canceled := mustApplication(t, now)
	if err := canceled.Cancel(reviewedAt); err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}
	if canceled.Status() != ApplicationStatusCanceled {
		t.Fatalf("expected canceled status, got %q", canceled.Status().String())
	}
}

func TestCommunityApplicationValidation(t *testing.T) {
	if _, err := NewCommunityApplicationID("not-a-uuid"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid application id, got %v", err)
	}
	if _, err := NewApplicationReason(" "); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for blank application reason, got %v", err)
	}
	if _, err := NewRejectReason(" "); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for blank reject reason, got %v", err)
	}
	if _, err := NewApplicationStatus("done"); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid application status, got %v", err)
	}

	now := time.Now().UTC()
	reviewerID := mustUserID(t)
	reviewedAt := now.Add(time.Minute)
	rejectReason := mustRejectReason(t, "duplicate")

	if _, err := RehydrateCommunityApplication(mustCommunityApplicationID(t), mustUserID(t), mustCommunitySlug(t, "campus-life"), mustCommunityName(t, "Campus Life"), mustApplicationReason(t, "Need a campus board"), ApplicationStatusPending, &reviewerID, nil, nil, now, now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for pending application with review fields, got %v", err)
	}
	if _, err := RehydrateCommunityApplication(mustCommunityApplicationID(t), mustUserID(t), mustCommunitySlug(t, "campus-life"), mustCommunityName(t, "Campus Life"), mustApplicationReason(t, "Need a campus board"), ApplicationStatusApproved, nil, &reviewedAt, nil, now, now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for approved application without reviewer, got %v", err)
	}
	if _, err := RehydrateCommunityApplication(mustCommunityApplicationID(t), mustUserID(t), mustCommunitySlug(t, "campus-life"), mustCommunityName(t, "Campus Life"), mustApplicationReason(t, "Need a campus board"), ApplicationStatusRejected, &reviewerID, &reviewedAt, nil, now, now); !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for rejected application without reject reason, got %v", err)
	}
	if _, err := RehydrateCommunityApplication(mustCommunityApplicationID(t), mustUserID(t), mustCommunitySlug(t, "campus-life"), mustCommunityName(t, "Campus Life"), mustApplicationReason(t, "Need a campus board"), ApplicationStatusRejected, &reviewerID, &reviewedAt, &rejectReason, now, now); err != nil {
		t.Fatalf("RehydrateCommunityApplication rejected returned error: %v", err)
	}
}

func assertCommunityKind(t *testing.T, raw string, want CommunityKind) {
	t.Helper()

	got, err := NewCommunityKind(raw)
	if err != nil {
		t.Fatalf("NewCommunityKind(%q) returned error: %v", raw, err)
	}
	if got != want {
		t.Fatalf("expected community kind %q, got %q", want.String(), got.String())
	}
}

func assertCommunityStatus(t *testing.T, raw string, want CommunityStatus) {
	t.Helper()

	got, err := NewCommunityStatus(raw)
	if err != nil {
		t.Fatalf("NewCommunityStatus(%q) returned error: %v", raw, err)
	}
	if got != want {
		t.Fatalf("expected community status %q, got %q", want.String(), got.String())
	}
}

func assertMembershipRole(t *testing.T, raw string, want MembershipRole) {
	t.Helper()

	got, err := NewMembershipRole(raw)
	if err != nil {
		t.Fatalf("NewMembershipRole(%q) returned error: %v", raw, err)
	}
	if got != want {
		t.Fatalf("expected membership role %q, got %q", want.String(), got.String())
	}
}

func assertMembershipStatus(t *testing.T, raw string, want MembershipStatus) {
	t.Helper()

	got, err := NewMembershipStatus(raw)
	if err != nil {
		t.Fatalf("NewMembershipStatus(%q) returned error: %v", raw, err)
	}
	if got != want {
		t.Fatalf("expected membership status %q, got %q", want.String(), got.String())
	}
}

func mustCommunityID(t *testing.T) CommunityID {
	t.Helper()

	id, err := NewCommunityID(uuid.NewString())
	if err != nil {
		t.Fatalf("NewCommunityID returned error: %v", err)
	}
	return id
}

func mustCommunityApplicationID(t *testing.T) CommunityApplicationID {
	t.Helper()

	id, err := NewCommunityApplicationID(uuid.NewString())
	if err != nil {
		t.Fatalf("NewCommunityApplicationID returned error: %v", err)
	}
	return id
}

func mustCommunityRuleID(t *testing.T) CommunityRuleID {
	t.Helper()

	id, err := NewCommunityRuleID(uuid.NewString())
	if err != nil {
		t.Fatalf("NewCommunityRuleID returned error: %v", err)
	}
	return id
}

func mustCommunitySlug(t *testing.T, raw string) CommunitySlug {
	t.Helper()

	slug, err := NewCommunitySlug(raw)
	if err != nil {
		t.Fatalf("NewCommunitySlug returned error: %v", err)
	}
	return slug
}

func mustCommunityName(t *testing.T, raw string) CommunityName {
	t.Helper()

	name, err := NewCommunityName(raw)
	if err != nil {
		t.Fatalf("NewCommunityName returned error: %v", err)
	}
	return name
}

func mustApplicationReason(t *testing.T, raw string) ApplicationReason {
	t.Helper()

	reason, err := NewApplicationReason(raw)
	if err != nil {
		t.Fatalf("NewApplicationReason returned error: %v", err)
	}
	return reason
}

func mustRejectReason(t *testing.T, raw string) RejectReason {
	t.Helper()

	reason, err := NewRejectReason(raw)
	if err != nil {
		t.Fatalf("NewRejectReason returned error: %v", err)
	}
	return reason
}

func mustUserID(t *testing.T) userdomain.UserID {
	t.Helper()

	id, err := userdomain.NewUserID(uuid.NewString())
	if err != nil {
		t.Fatalf("NewUserID returned error: %v", err)
	}
	return id
}

func mustApplication(t *testing.T, now time.Time) *CommunityApplication {
	t.Helper()

	application, err := NewCommunityApplication(mustCommunityApplicationID(t), mustUserID(t), mustCommunitySlug(t, "campus-life"), mustCommunityName(t, "Campus Life"), mustApplicationReason(t, "Need a campus board"), now)
	if err != nil {
		t.Fatalf("NewCommunityApplication returned error: %v", err)
	}
	return application
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

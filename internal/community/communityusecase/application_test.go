package communityusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestSubmitCommunityApplicationCreatesPendingApplication(t *testing.T) {
	now := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	applicantID := userdomain.NewGeneratedUserID()
	var created *communitydomain.CommunityApplication
	communities := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return nil, apperr.New(apperr.CodeNotFound, "community not found")
		},
	}
	applications := &fakeApplicationRepository{
		createFunc: func(ctx context.Context, application communitydomain.CommunityApplication) error {
			created = &application
			return nil
		},
	}
	uc := NewCommunityApplicationUseCase(communities, applications, nil, nil, func() time.Time { return now })

	result, err := uc.SubmitCommunityApplication(context.Background(), SubmitCommunityApplicationInput{
		ApplicantID:   applicantID,
		RequestedSlug: "Campus-Life",
		RequestedName: "Campus Life",
		Reason:        "Need a campus board",
	})
	if err != nil {
		t.Fatalf("SubmitCommunityApplication returned error: %v", err)
	}

	if created == nil {
		t.Fatal("expected application to be created")
	}
	if created.ApplicantID() != applicantID {
		t.Fatalf("expected applicant %q, got %q", applicantID.String(), created.ApplicantID().String())
	}
	if created.RequestedSlug().String() != "campus-life" {
		t.Fatalf("expected normalized slug, got %q", created.RequestedSlug().String())
	}
	if created.Status() != communitydomain.ApplicationStatusPending {
		t.Fatalf("expected pending status, got %q", created.Status().String())
	}
	if result.Application.Status != communitydomain.ApplicationStatusPending.String() {
		t.Fatalf("expected result pending status, got %q", result.Application.Status)
	}
}

func TestSubmitCommunityApplicationRejectsExistingCommunitySlug(t *testing.T) {
	now := time.Now().UTC()
	existing := mustSystemCommunity(t, "campus", now)
	communities := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return existing, nil
		},
	}
	applications := &fakeApplicationRepository{
		createFunc: func(ctx context.Context, application communitydomain.CommunityApplication) error {
			t.Fatal("Create should not be called when community slug exists")
			return nil
		},
	}
	uc := NewCommunityApplicationUseCase(communities, applications, nil, nil, time.Now)

	_, err := uc.SubmitCommunityApplication(context.Background(), SubmitCommunityApplicationInput{
		ApplicantID:   userdomain.NewGeneratedUserID(),
		RequestedSlug: "campus",
		RequestedName: "Campus",
		Reason:        "Need a campus board",
	})
	if !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for existing community slug, got %v", err)
	}
}

func TestSubmitCommunityApplicationMapsPendingSlugConflict(t *testing.T) {
	communities := &fakeCommunityRepository{
		findBySlugFunc: func(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error) {
			return nil, apperr.New(apperr.CodeNotFound, "community not found")
		},
	}
	applications := &fakeApplicationRepository{
		createFunc: func(ctx context.Context, application communitydomain.CommunityApplication) error {
			return apperr.New(apperr.CodeConflict, "pending community application slug already exists")
		},
	}
	uc := NewCommunityApplicationUseCase(communities, applications, nil, nil, time.Now)

	_, err := uc.SubmitCommunityApplication(context.Background(), SubmitCommunityApplicationInput{
		ApplicantID:   userdomain.NewGeneratedUserID(),
		RequestedSlug: "campus",
		RequestedName: "Campus",
		Reason:        "Need a campus board",
	})
	if !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict for pending slug conflict, got %v", err)
	}
}

func TestSubmitCommunityApplicationRejectsInvalidInput(t *testing.T) {
	uc := NewCommunityApplicationUseCase(&fakeCommunityRepository{}, &fakeApplicationRepository{}, nil, nil, time.Now)

	_, err := uc.SubmitCommunityApplication(context.Background(), SubmitCommunityApplicationInput{
		ApplicantID:   userdomain.NewGeneratedUserID(),
		RequestedSlug: "no",
		RequestedName: "Campus",
		Reason:        "Need a campus board",
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid slug, got %v", err)
	}

	_, err = uc.SubmitCommunityApplication(context.Background(), SubmitCommunityApplicationInput{
		ApplicantID:   "",
		RequestedSlug: "campus",
		RequestedName: "Campus",
		Reason:        "Need a campus board",
	})
	if !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated for missing applicant, got %v", err)
	}
}

func TestListCommunityApplicationsChecksStaffAndNormalizesInput(t *testing.T) {
	now := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	reviewerID := userdomain.NewGeneratedUserID()
	applicantID := userdomain.NewGeneratedUserID()
	pending := mustApplicationWithSlug(t, applicantID, "campus", now)

	var gotStatus communitydomain.ApplicationStatus
	var gotLimit int
	var gotOffset int
	applications := &fakeApplicationRepository{
		listByStatusFunc: func(ctx context.Context, status communitydomain.ApplicationStatus, limit int, offset int) ([]communitydomain.CommunityApplication, error) {
			gotStatus = status
			gotLimit = limit
			gotOffset = offset
			return []communitydomain.CommunityApplication{*pending}, nil
		},
	}
	uc := NewCommunityApplicationUseCase(
		&fakeCommunityRepository{},
		applications,
		&fakePlatformStaffRepository{isStaff: true},
		&fakeCommunityTransactionManager{},
		time.Now,
	)

	result, err := uc.ListCommunityApplications(context.Background(), ListCommunityApplicationsInput{
		ReviewerID: reviewerID,
		Status:     "",
		Limit:      100,
		Offset:     5,
	})
	if err != nil {
		t.Fatalf("ListCommunityApplications returned error: %v", err)
	}

	if gotStatus != communitydomain.ApplicationStatusPending {
		t.Fatalf("expected default pending status, got %q", gotStatus.String())
	}
	if gotLimit != maxCommunityApplicationListLimit+1 || result.Limit != maxCommunityApplicationListLimit {
		t.Fatalf("expected capped limit %d, got repo=%d result=%d", maxCommunityApplicationListLimit, gotLimit, result.Limit)
	}
	if gotOffset != 5 || result.Offset != 5 {
		t.Fatalf("expected offset 5, got repo=%d result=%d", gotOffset, result.Offset)
	}
	if len(result.Applications) != 1 || result.Applications[0].ID != pending.ID().String() {
		t.Fatalf("unexpected applications result: %#v", result.Applications)
	}
}

func TestListCommunityApplicationsRejectsNonStaffAndInvalidInput(t *testing.T) {
	applications := &fakeApplicationRepository{
		listByStatusFunc: func(ctx context.Context, status communitydomain.ApplicationStatus, limit int, offset int) ([]communitydomain.CommunityApplication, error) {
			t.Fatal("ListByStatus should not be called")
			return nil, nil
		},
	}
	uc := NewCommunityApplicationUseCase(
		&fakeCommunityRepository{},
		applications,
		&fakePlatformStaffRepository{isStaff: false},
		&fakeCommunityTransactionManager{},
		time.Now,
	)

	_, err := uc.ListCommunityApplications(context.Background(), ListCommunityApplicationsInput{
		ReviewerID: userdomain.NewGeneratedUserID(),
		Status:     "pending",
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for non-staff reviewer, got %v", err)
	}

	uc = NewCommunityApplicationUseCase(
		&fakeCommunityRepository{},
		applications,
		&fakePlatformStaffRepository{isStaff: true},
		&fakeCommunityTransactionManager{},
		time.Now,
	)

	_, err = uc.ListCommunityApplications(context.Background(), ListCommunityApplicationsInput{
		ReviewerID: userdomain.NewGeneratedUserID(),
		Status:     "canceled",
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for unsupported status, got %v", err)
	}

	_, err = uc.ListCommunityApplications(context.Background(), ListCommunityApplicationsInput{
		ReviewerID: userdomain.NewGeneratedUserID(),
		Status:     "pending",
		Limit:      -1,
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for negative limit, got %v", err)
	}
}

func TestGetCommunityApplicationChecksStaffAndReturnsApplication(t *testing.T) {
	now := time.Date(2026, 6, 3, 11, 0, 0, 0, time.UTC)
	reviewerID := userdomain.NewGeneratedUserID()
	application := mustApplicationWithSlug(t, userdomain.NewGeneratedUserID(), "campus", now)

	var gotID communitydomain.CommunityApplicationID
	applications := &fakeApplicationRepository{
		findByIDFunc: func(ctx context.Context, id communitydomain.CommunityApplicationID) (*communitydomain.CommunityApplication, error) {
			gotID = id
			return application, nil
		},
	}
	uc := NewCommunityApplicationUseCase(
		&fakeCommunityRepository{},
		applications,
		&fakePlatformStaffRepository{isStaff: true},
		&fakeCommunityTransactionManager{},
		time.Now,
	)

	result, err := uc.GetCommunityApplication(context.Background(), GetCommunityApplicationInput{
		ReviewerID:    reviewerID,
		ApplicationID: application.ID().String(),
	})
	if err != nil {
		t.Fatalf("GetCommunityApplication returned error: %v", err)
	}
	if gotID != application.ID() {
		t.Fatalf("expected application id %q, got %q", application.ID().String(), gotID.String())
	}
	if result.Application.ID != application.ID().String() {
		t.Fatalf("expected application result %q, got %q", application.ID().String(), result.Application.ID)
	}
}

func TestApproveCommunityApplicationCreatesCommunityAndOwnerInTransaction(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	applicantID := userdomain.NewGeneratedUserID()
	reviewerID := userdomain.NewGeneratedUserID()
	application := mustApplicationWithSlug(t, applicantID, "campus", now.Add(-time.Minute))

	applications := &fakeApplicationRepository{
		findByIDForUpdateFunc: func(ctx context.Context, id communitydomain.CommunityApplicationID) (*communitydomain.CommunityApplication, error) {
			return application, nil
		},
		saveFunc: func(ctx context.Context, saved communitydomain.CommunityApplication) error {
			if saved.Status() != communitydomain.ApplicationStatusApproved {
				t.Fatalf("expected approved application, got %q", saved.Status().String())
			}
			return nil
		},
	}
	communities := &fakeCommunityRepository{
		createFunc: func(ctx context.Context, community communitydomain.Community) error {
			if community.Slug().String() != "campus" {
				t.Fatalf("expected campus community, got %q", community.Slug().String())
			}
			if creatorID, ok := community.CreatedBy(); !ok || creatorID != applicantID {
				t.Fatalf("expected creator %q, got %q present=%t", applicantID.String(), creatorID.String(), ok)
			}
			return nil
		},
	}
	memberships := &fakeMembershipRepository{
		createFunc: func(ctx context.Context, membership communitydomain.CommunityMembership) error {
			if membership.UserID() != applicantID {
				t.Fatalf("expected owner user %q, got %q", applicantID.String(), membership.UserID().String())
			}
			if membership.Role() != communitydomain.MembershipRoleOwner {
				t.Fatalf("expected owner role, got %q", membership.Role().String())
			}
			return nil
		},
	}
	transactions := &fakeCommunityTransactionManager{
		repositories: fakeCommunityRepositories{
			communities:  communities,
			memberships:  memberships,
			applications: applications,
		},
	}
	uc := NewCommunityApplicationUseCase(communities, applications, &fakePlatformStaffRepository{isStaff: true}, transactions, func() time.Time { return now })

	result, err := uc.ApproveCommunityApplication(context.Background(), ReviewCommunityApplicationInput{
		ApplicationID: application.ID().String(),
		ReviewerID:    reviewerID,
	})
	if err != nil {
		t.Fatalf("ApproveCommunityApplication returned error: %v", err)
	}

	if !transactions.called {
		t.Fatal("expected transaction manager to be called")
	}
	if result.Application.Status != communitydomain.ApplicationStatusApproved.String() {
		t.Fatalf("expected approved result, got %q", result.Application.Status)
	}
	if result.Community.Slug != "campus" {
		t.Fatalf("expected campus community result, got %q", result.Community.Slug)
	}
}

func TestApproveCommunityApplicationRejectsNonStaffReviewer(t *testing.T) {
	uc := NewCommunityApplicationUseCase(
		&fakeCommunityRepository{},
		&fakeApplicationRepository{},
		&fakePlatformStaffRepository{isStaff: false},
		&fakeCommunityTransactionManager{},
		time.Now,
	)

	_, err := uc.ApproveCommunityApplication(context.Background(), ReviewCommunityApplicationInput{
		ApplicationID: communitydomain.NewGeneratedCommunityApplicationID().String(),
		ReviewerID:    userdomain.NewGeneratedUserID(),
	})
	if !hasAppCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden for non-staff reviewer, got %v", err)
	}
}

func TestRejectCommunityApplicationSavesRejectedApplication(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 30, 0, 0, time.UTC)
	applicantID := userdomain.NewGeneratedUserID()
	reviewerID := userdomain.NewGeneratedUserID()
	application := mustApplicationWithSlug(t, applicantID, "campus", now.Add(-time.Minute))
	applications := &fakeApplicationRepository{
		findByIDForUpdateFunc: func(ctx context.Context, id communitydomain.CommunityApplicationID) (*communitydomain.CommunityApplication, error) {
			return application, nil
		},
		saveFunc: func(ctx context.Context, saved communitydomain.CommunityApplication) error {
			if saved.Status() != communitydomain.ApplicationStatusRejected {
				t.Fatalf("expected rejected status, got %q", saved.Status().String())
			}
			if reason, ok := saved.RejectReason(); !ok || reason.String() != "duplicate slug" {
				t.Fatalf("expected reject reason, got %q present=%t", reason.String(), ok)
			}
			return nil
		},
	}
	transactions := &fakeCommunityTransactionManager{
		repositories: fakeCommunityRepositories{
			communities:  &fakeCommunityRepository{},
			memberships:  &fakeMembershipRepository{},
			applications: applications,
		},
	}
	uc := NewCommunityApplicationUseCase(
		&fakeCommunityRepository{},
		applications,
		&fakePlatformStaffRepository{isStaff: true},
		transactions,
		func() time.Time { return now },
	)

	result, err := uc.RejectCommunityApplication(context.Background(), ReviewCommunityApplicationInput{
		ApplicationID: application.ID().String(),
		ReviewerID:    reviewerID,
		RejectReason:  "duplicate slug",
	})
	if err != nil {
		t.Fatalf("RejectCommunityApplication returned error: %v", err)
	}
	if result.Application.Status != communitydomain.ApplicationStatusRejected.String() {
		t.Fatalf("expected rejected result, got %q", result.Application.Status)
	}
}

type fakeApplicationRepository struct {
	createFunc            func(ctx context.Context, application communitydomain.CommunityApplication) error
	findByIDFunc          func(ctx context.Context, id communitydomain.CommunityApplicationID) (*communitydomain.CommunityApplication, error)
	findByIDForUpdateFunc func(ctx context.Context, id communitydomain.CommunityApplicationID) (*communitydomain.CommunityApplication, error)
	listByStatusFunc      func(ctx context.Context, status communitydomain.ApplicationStatus, limit int, offset int) ([]communitydomain.CommunityApplication, error)
	saveFunc              func(ctx context.Context, application communitydomain.CommunityApplication) error
}

func (f *fakeApplicationRepository) Create(ctx context.Context, application communitydomain.CommunityApplication) error {
	if f.createFunc != nil {
		return f.createFunc(ctx, application)
	}
	return nil
}

func (f *fakeApplicationRepository) FindByID(ctx context.Context, id communitydomain.CommunityApplicationID) (*communitydomain.CommunityApplication, error) {
	if f.findByIDFunc != nil {
		return f.findByIDFunc(ctx, id)
	}
	return nil, apperr.New(apperr.CodeNotFound, "community application not found")
}

func (f *fakeApplicationRepository) FindByIDForUpdate(ctx context.Context, id communitydomain.CommunityApplicationID) (*communitydomain.CommunityApplication, error) {
	if f.findByIDForUpdateFunc != nil {
		return f.findByIDForUpdateFunc(ctx, id)
	}
	return f.FindByID(ctx, id)
}

func (f *fakeApplicationRepository) ListByStatus(ctx context.Context, status communitydomain.ApplicationStatus, limit int, offset int) ([]communitydomain.CommunityApplication, error) {
	if f.listByStatusFunc != nil {
		return f.listByStatusFunc(ctx, status, limit, offset)
	}
	return nil, nil
}

func (f *fakeApplicationRepository) Save(ctx context.Context, application communitydomain.CommunityApplication) error {
	if f.saveFunc != nil {
		return f.saveFunc(ctx, application)
	}
	return nil
}

type fakeMembershipRepository struct {
	createFunc func(ctx context.Context, membership communitydomain.CommunityMembership) error
}

func (f *fakeMembershipRepository) Create(ctx context.Context, membership communitydomain.CommunityMembership) error {
	if f.createFunc != nil {
		return f.createFunc(ctx, membership)
	}
	return nil
}

func (f *fakeMembershipRepository) FindActiveRolesByUser(ctx context.Context, communityIDs []communitydomain.CommunityID, userID userdomain.UserID) (map[communitydomain.CommunityID]communitydomain.MembershipRole, error) {
	return map[communitydomain.CommunityID]communitydomain.MembershipRole{}, nil
}

func (f *fakeMembershipRepository) ListActiveMembers(ctx context.Context, communityID communitydomain.CommunityID, limit int, offset int) ([]CommunityMember, error) {
	return nil, nil
}

func (f *fakeMembershipRepository) FindActiveMemberByUserID(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID) (CommunityMember, error) {
	return CommunityMember{}, apperr.New(apperr.CodeNotFound, "community member not found")
}

func (f *fakeMembershipRepository) FindActiveMemberByUsername(ctx context.Context, communityID communitydomain.CommunityID, username string) (CommunityMember, error) {
	return CommunityMember{}, apperr.New(apperr.CodeNotFound, "community member not found")
}

func (f *fakeMembershipRepository) FindActiveUserByUsername(ctx context.Context, username string) (CommunityMember, error) {
	return CommunityMember{}, apperr.New(apperr.CodeNotFound, "user not found")
}

func (f *fakeMembershipRepository) FindActiveUserByID(ctx context.Context, userID userdomain.UserID) (CommunityMember, error) {
	return CommunityMember{}, apperr.New(apperr.CodeNotFound, "user not found")
}

func (f *fakeMembershipRepository) CountActiveMembers(ctx context.Context, communityID communitydomain.CommunityID) (int, error) {
	return 0, nil
}

func (f *fakeMembershipRepository) CountActiveModerators(ctx context.Context, communityID communitydomain.CommunityID) (int, error) {
	return 0, nil
}

func (f *fakeMembershipRepository) UpsertActiveMemberRole(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID, role communitydomain.MembershipRole, now time.Time) (CommunityMember, error) {
	return CommunityMember{}, nil
}

func (f *fakeMembershipRepository) CreateOwnerTransfer(ctx context.Context, transfer CommunityOwnerTransferRecord) error {
	return nil
}

func (f *fakeMembershipRepository) FindOwnerTransferForUpdate(ctx context.Context, transferID string) (CommunityOwnerTransferRecord, error) {
	return CommunityOwnerTransferRecord{}, apperr.New(apperr.CodeNotFound, "community owner transfer not found")
}

func (f *fakeMembershipRepository) AcceptOwnerTransfer(ctx context.Context, transferID string, acceptedAt time.Time) error {
	return nil
}

func (f *fakeMembershipRepository) TransferOwner(ctx context.Context, communityID communitydomain.CommunityID, newOwnerID userdomain.UserID, now time.Time) (CommunityOwnerChange, error) {
	return CommunityOwnerChange{}, nil
}

type fakePlatformStaffRepository struct {
	isStaff bool
	err     error
}

func (f *fakePlatformStaffRepository) IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.isStaff, nil
}

type fakeCommunityTransactionManager struct {
	called       bool
	repositories fakeCommunityRepositories
	err          error
}

func (f *fakeCommunityTransactionManager) WithinTx(ctx context.Context, fn func(ctx context.Context, repositories CommunityRepositories) error) error {
	f.called = true
	if f.err != nil {
		return f.err
	}
	return fn(ctx, f.repositories)
}

type fakeCommunityRepositories struct {
	communities  CommunityRepository
	memberships  CommunityMembershipRepository
	applications CommunityApplicationRepository
}

func (repositories fakeCommunityRepositories) Communities() CommunityRepository {
	return repositories.communities
}

func (repositories fakeCommunityRepositories) Memberships() CommunityMembershipRepository {
	return repositories.memberships
}

func (repositories fakeCommunityRepositories) Applications() CommunityApplicationRepository {
	return repositories.applications
}

func mustApplicationWithSlug(t *testing.T, applicantID userdomain.UserID, rawSlug string, now time.Time) *communitydomain.CommunityApplication {
	t.Helper()

	application, err := communitydomain.NewCommunityApplication(
		communitydomain.NewGeneratedCommunityApplicationID(),
		applicantID,
		mustCommunitySlug(t, rawSlug),
		mustCommunityName(t, "Application "+rawSlug),
		mustApplicationReason(t, "Need a community for "+rawSlug),
		now,
	)
	if err != nil {
		t.Fatalf("NewCommunityApplication returned error: %v", err)
	}
	return application
}

func mustApplicationReason(t *testing.T, raw string) communitydomain.ApplicationReason {
	t.Helper()

	reason, err := communitydomain.NewApplicationReason(raw)
	if err != nil {
		t.Fatalf("NewApplicationReason returned error: %v", err)
	}
	return reason
}

func TestApproveCommunityApplicationPropagatesTransactionError(t *testing.T) {
	expectedErr := errors.New("transaction failed")
	uc := NewCommunityApplicationUseCase(
		&fakeCommunityRepository{},
		&fakeApplicationRepository{},
		&fakePlatformStaffRepository{isStaff: true},
		&fakeCommunityTransactionManager{err: expectedErr},
		time.Now,
	)

	_, err := uc.ApproveCommunityApplication(context.Background(), ReviewCommunityApplicationInput{
		ApplicationID: communitydomain.NewGeneratedCommunityApplicationID().String(),
		ReviewerID:    userdomain.NewGeneratedUserID(),
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected transaction error to be propagated, got %v", err)
	}
}

package communityusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

const (
	defaultCommunityApplicationListLimit = 20
	maxCommunityApplicationListLimit     = 50
)

type CommunityApplicationUseCase struct {
	communities  CommunityRepository
	applications CommunityApplicationRepository
	staff        PlatformStaffRepository
	transactions CommunityTransactionManager
	now          func() time.Time
}

type SubmitCommunityApplicationInput struct {
	ApplicantID   userdomain.UserID
	RequestedSlug string
	RequestedName string
	Reason        string
}

type ReviewCommunityApplicationInput struct {
	ApplicationID string
	ReviewerID    userdomain.UserID
	RejectReason  string
}

type ListCommunityApplicationsInput struct {
	ReviewerID userdomain.UserID
	Status     string
	Limit      int
	Offset     int
}

type GetCommunityApplicationInput struct {
	ReviewerID    userdomain.UserID
	ApplicationID string
}

type SubmitCommunityApplicationResult struct {
	Application CommunityApplication
}

type ListCommunityApplicationsResult struct {
	Applications []CommunityApplication
	Limit        int
	Offset       int
}

type GetCommunityApplicationResult struct {
	Application CommunityApplication
}

type ApproveCommunityApplicationResult struct {
	Application CommunityApplication
	Community   Community
}

type RejectCommunityApplicationResult struct {
	Application CommunityApplication
}

type CommunityApplication struct {
	ID            string
	ApplicantID   string
	RequestedSlug string
	RequestedName string
	Reason        string
	Status        string
	ReviewedBy    string
	ReviewedAt    *time.Time
	RejectReason  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewCommunityApplicationUseCase(
	communities CommunityRepository,
	applications CommunityApplicationRepository,
	staff PlatformStaffRepository,
	transactions CommunityTransactionManager,
	now func() time.Time,
) *CommunityApplicationUseCase {
	if now == nil {
		now = time.Now
	}

	return &CommunityApplicationUseCase{
		communities:  communities,
		applications: applications,
		staff:        staff,
		transactions: transactions,
		now:          now,
	}
}

func (uc *CommunityApplicationUseCase) SubmitCommunityApplication(ctx context.Context, input SubmitCommunityApplicationInput) (SubmitCommunityApplicationResult, error) {
	if isBlankUserID(input.ApplicantID) {
		return SubmitCommunityApplicationResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}

	slug, err := communitydomain.NewCommunitySlug(input.RequestedSlug)
	if err != nil {
		return SubmitCommunityApplicationResult{}, err
	}
	name, err := communitydomain.NewCommunityName(input.RequestedName)
	if err != nil {
		return SubmitCommunityApplicationResult{}, err
	}
	reason, err := communitydomain.NewApplicationReason(input.Reason)
	if err != nil {
		return SubmitCommunityApplicationResult{}, err
	}

	if err := uc.ensureRequestedSlugAvailable(ctx, slug); err != nil {
		return SubmitCommunityApplicationResult{}, err
	}

	now := uc.now().UTC()
	application, err := communitydomain.NewCommunityApplication(
		communitydomain.NewGeneratedCommunityApplicationID(),
		input.ApplicantID,
		slug,
		name,
		reason,
		now,
	)
	if err != nil {
		return SubmitCommunityApplicationResult{}, err
	}

	if err := uc.applications.Create(ctx, *application); err != nil {
		return SubmitCommunityApplicationResult{}, fmt.Errorf("create community application: %w", err)
	}

	return SubmitCommunityApplicationResult{
		Application: toCommunityApplicationDTO(*application),
	}, nil
}

func (uc *CommunityApplicationUseCase) ListCommunityApplications(ctx context.Context, input ListCommunityApplicationsInput) (ListCommunityApplicationsResult, error) {
	if err := uc.ensureReviewerIsPlatformStaff(ctx, input.ReviewerID); err != nil {
		return ListCommunityApplicationsResult{}, err
	}

	status, err := parseApplicationListStatusDefaultPending(input.Status)
	if err != nil {
		return ListCommunityApplicationsResult{}, err
	}
	limit, offset, err := normalizeCommunityApplicationPagination(input.Limit, input.Offset)
	if err != nil {
		return ListCommunityApplicationsResult{}, err
	}

	applications, err := uc.applications.ListByStatus(ctx, status, limit, offset)
	if err != nil {
		return ListCommunityApplicationsResult{}, fmt.Errorf("list community applications: %w", err)
	}

	result := ListCommunityApplicationsResult{
		Applications: make([]CommunityApplication, 0, len(applications)),
		Limit:        limit,
		Offset:       offset,
	}
	for _, application := range applications {
		result.Applications = append(result.Applications, toCommunityApplicationDTO(application))
	}

	return result, nil
}

func (uc *CommunityApplicationUseCase) GetCommunityApplication(ctx context.Context, input GetCommunityApplicationInput) (GetCommunityApplicationResult, error) {
	if err := uc.ensureReviewerIsPlatformStaff(ctx, input.ReviewerID); err != nil {
		return GetCommunityApplicationResult{}, err
	}

	applicationID, err := communitydomain.NewCommunityApplicationID(input.ApplicationID)
	if err != nil {
		return GetCommunityApplicationResult{}, err
	}

	application, err := uc.applications.FindByID(ctx, applicationID)
	if err != nil {
		return GetCommunityApplicationResult{}, fmt.Errorf("find community application: %w", err)
	}

	return GetCommunityApplicationResult{
		Application: toCommunityApplicationDTO(*application),
	}, nil
}

func (uc *CommunityApplicationUseCase) ApproveCommunityApplication(ctx context.Context, input ReviewCommunityApplicationInput) (ApproveCommunityApplicationResult, error) {
	if err := uc.ensureReviewerIsPlatformStaff(ctx, input.ReviewerID); err != nil {
		return ApproveCommunityApplicationResult{}, err
	}

	applicationID, err := communitydomain.NewCommunityApplicationID(input.ApplicationID)
	if err != nil {
		return ApproveCommunityApplicationResult{}, err
	}

	var result ApproveCommunityApplicationResult
	if err := uc.transactions.WithinTx(ctx, func(txCtx context.Context, repositories CommunityRepositories) error {
		application, err := repositories.Applications().FindByIDForUpdate(txCtx, applicationID)
		if err != nil {
			return fmt.Errorf("find community application for review: %w", err)
		}

		reviewedAt := uc.now().UTC()
		if err := application.Approve(input.ReviewerID, reviewedAt); err != nil {
			return err
		}

		community, err := communitydomain.NewUserCreatedCommunity(
			communitydomain.NewGeneratedCommunityID(),
			application.RequestedSlug(),
			application.RequestedName(),
			communitydomain.NewCommunityDescription(""),
			application.ApplicantID(),
			reviewedAt,
		)
		if err != nil {
			return err
		}

		membership, err := communitydomain.NewCommunityMembership(
			community.ID(),
			application.ApplicantID(),
			communitydomain.MembershipRoleOwner,
			reviewedAt,
		)
		if err != nil {
			return err
		}

		if err := repositories.Applications().Save(txCtx, *application); err != nil {
			return fmt.Errorf("approve community application: %w", err)
		}
		if err := repositories.Communities().Create(txCtx, *community); err != nil {
			return fmt.Errorf("create approved community: %w", err)
		}
		if err := repositories.Memberships().Create(txCtx, *membership); err != nil {
			return fmt.Errorf("create community owner membership: %w", err)
		}

		result = ApproveCommunityApplicationResult{
			Application: toCommunityApplicationDTO(*application),
			Community:   toCommunityDTO(*community),
		}
		return nil
	}); err != nil {
		return ApproveCommunityApplicationResult{}, err
	}

	return result, nil
}

func (uc *CommunityApplicationUseCase) RejectCommunityApplication(ctx context.Context, input ReviewCommunityApplicationInput) (RejectCommunityApplicationResult, error) {
	if err := uc.ensureReviewerIsPlatformStaff(ctx, input.ReviewerID); err != nil {
		return RejectCommunityApplicationResult{}, err
	}

	applicationID, err := communitydomain.NewCommunityApplicationID(input.ApplicationID)
	if err != nil {
		return RejectCommunityApplicationResult{}, err
	}
	rejectReason, err := communitydomain.NewRejectReason(input.RejectReason)
	if err != nil {
		return RejectCommunityApplicationResult{}, err
	}

	var result RejectCommunityApplicationResult
	if err := uc.transactions.WithinTx(ctx, func(txCtx context.Context, repositories CommunityRepositories) error {
		application, err := repositories.Applications().FindByIDForUpdate(txCtx, applicationID)
		if err != nil {
			return fmt.Errorf("find community application for review: %w", err)
		}

		reviewedAt := uc.now().UTC()
		if err := application.Reject(input.ReviewerID, reviewedAt, rejectReason); err != nil {
			return err
		}
		if err := repositories.Applications().Save(txCtx, *application); err != nil {
			return fmt.Errorf("reject community application: %w", err)
		}

		result = RejectCommunityApplicationResult{
			Application: toCommunityApplicationDTO(*application),
		}
		return nil
	}); err != nil {
		return RejectCommunityApplicationResult{}, err
	}

	return result, nil
}

func (uc *CommunityApplicationUseCase) ensureRequestedSlugAvailable(ctx context.Context, slug communitydomain.CommunitySlug) error {
	_, err := uc.communities.FindBySlug(ctx, slug)
	if err == nil {
		return apperr.New(apperr.CodeConflict, "community slug already exists")
	}
	if apperr.IsCode(err, apperr.CodeNotFound) {
		return nil
	}

	return fmt.Errorf("check community slug availability: %w", err)
}

func (uc *CommunityApplicationUseCase) ensureReviewerIsPlatformStaff(ctx context.Context, reviewerID userdomain.UserID) error {
	if isBlankUserID(reviewerID) {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}

	isStaff, err := uc.staff.IsPlatformStaff(ctx, reviewerID)
	if err != nil {
		return fmt.Errorf("check platform staff: %w", err)
	}
	if !isStaff {
		return apperr.New(apperr.CodeForbidden, "platform staff required")
	}

	return nil
}

func parseApplicationListStatusDefaultPending(raw string) (communitydomain.ApplicationStatus, error) {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	switch normalized {
	case "":
		return communitydomain.ApplicationStatusPending, nil
	case communitydomain.ApplicationStatusPending.String(),
		communitydomain.ApplicationStatusApproved.String(),
		communitydomain.ApplicationStatusRejected.String():
		return communitydomain.NewApplicationStatus(normalized)
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "community application status is invalid")
	}
}

func normalizeCommunityApplicationPagination(limit int, offset int) (int, int, error) {
	if limit < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "limit must be non-negative")
	}
	if offset < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "offset must be non-negative")
	}
	if limit == 0 {
		limit = defaultCommunityApplicationListLimit
	}
	if limit > maxCommunityApplicationListLimit {
		limit = maxCommunityApplicationListLimit
	}
	return limit, offset, nil
}

func toCommunityApplicationDTO(application communitydomain.CommunityApplication) CommunityApplication {
	reviewedBy, hasReviewedBy := application.ReviewedBy()
	reviewedAt, hasReviewedAt := application.ReviewedAt()
	rejectReason, hasRejectReason := application.RejectReason()

	dto := CommunityApplication{
		ID:            application.ID().String(),
		ApplicantID:   application.ApplicantID().String(),
		RequestedSlug: application.RequestedSlug().String(),
		RequestedName: application.RequestedName().String(),
		Reason:        application.Reason().String(),
		Status:        application.Status().String(),
		CreatedAt:     application.CreatedAt(),
		UpdatedAt:     application.UpdatedAt(),
	}
	if hasReviewedBy {
		dto.ReviewedBy = reviewedBy.String()
	}
	if hasReviewedAt {
		copiedReviewedAt := reviewedAt
		dto.ReviewedAt = &copiedReviewedAt
	}
	if hasRejectReason {
		dto.RejectReason = rejectReason.String()
	}

	return dto
}

func isBlankUserID(userID userdomain.UserID) bool {
	return strings.TrimSpace(userID.String()) == ""
}

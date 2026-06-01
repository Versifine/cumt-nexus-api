package moderationusecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

const (
	defaultReportListLimit = 20
	maxReportListLimit     = 50
)

type ConsoleUseCase struct {
	reports ContentReportQueryRepository
	staff   PlatformStaffRepository
}

func NewConsoleUseCase(reports ContentReportQueryRepository, staff PlatformStaffRepository) *ConsoleUseCase {
	return &ConsoleUseCase{
		reports: reports,
		staff:   staff,
	}
}

type ListReportsInput struct {
	ActorID userdomain.UserID
	Status  string
	Limit   int
	Offset  int
}

type ListReportsResult struct {
	Reports []ContentReport
	Limit   int
	Offset  int
}

type GetReportInput struct {
	ActorID  userdomain.UserID
	ReportID string
}

type GetReportResult struct {
	Report ContentReport
}

func (uc *ConsoleUseCase) ListReports(ctx context.Context, input ListReportsInput) (ListReportsResult, error) {
	if err := uc.ensureActorCanUseConsole(ctx, input.ActorID); err != nil {
		return ListReportsResult{}, err
	}
	status, err := parseReportStatusDefaultPending(input.Status)
	if err != nil {
		return ListReportsResult{}, err
	}
	limit, offset, err := normalizeReportPagination(input.Limit, input.Offset)
	if err != nil {
		return ListReportsResult{}, err
	}

	reports, err := uc.reports.ListReports(ctx, status, limit, offset)
	if err != nil {
		return ListReportsResult{}, fmt.Errorf("list moderation reports: %w", err)
	}

	result := ListReportsResult{
		Reports: make([]ContentReport, 0, len(reports)),
		Limit:   limit,
		Offset:  offset,
	}
	for _, report := range reports {
		result.Reports = append(result.Reports, toContentReportDTO(report))
	}
	return result, nil
}

func (uc *ConsoleUseCase) GetReport(ctx context.Context, input GetReportInput) (GetReportResult, error) {
	if err := uc.ensureActorCanUseConsole(ctx, input.ActorID); err != nil {
		return GetReportResult{}, err
	}
	reportID, err := moderationdomain.NewContentReportID(input.ReportID)
	if err != nil {
		return GetReportResult{}, err
	}

	report, err := uc.reports.FindReportByID(ctx, reportID)
	if err != nil {
		return GetReportResult{}, fmt.Errorf("find moderation report: %w", err)
	}
	return GetReportResult{
		Report: toContentReportDTO(*report),
	}, nil
}

func (uc *ConsoleUseCase) ensureActorCanUseConsole(ctx context.Context, actorID userdomain.UserID) error {
	if strings.TrimSpace(actorID.String()) == "" {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}

	isStaff, err := uc.staff.IsPlatformStaff(ctx, actorID)
	if err != nil {
		return fmt.Errorf("check platform staff: %w", err)
	}
	if !isStaff {
		return apperr.New(apperr.CodeForbidden, "platform staff required")
	}
	return nil
}

func parseReportStatusDefaultPending(raw string) (moderationdomain.ReportStatus, error) {
	if strings.TrimSpace(raw) == "" {
		return moderationdomain.ReportStatusPending, nil
	}
	return moderationdomain.NewReportStatus(raw)
}

func normalizeReportPagination(limit int, offset int) (int, int, error) {
	if limit < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "limit must be non-negative")
	}
	if offset < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "offset must be non-negative")
	}
	if limit == 0 {
		limit = defaultReportListLimit
	}
	if limit > maxReportListLimit {
		limit = maxReportListLimit
	}
	return limit, offset, nil
}

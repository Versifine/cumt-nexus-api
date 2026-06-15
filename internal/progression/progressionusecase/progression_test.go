package progressionusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestBuildProgressionResolvesLevelCurve(t *testing.T) {
	progress := BuildProgression(ProgressionRecord{UserID: "u1", XPTotal: 6200})
	if progress.Level != 10 {
		t.Fatalf("expected level 10, got %d", progress.Level)
	}
	if progress.LevelName != "熟悉校园" {
		t.Fatalf("expected level name 熟悉校园, got %q", progress.LevelName)
	}
	if progress.CurrentLevelXP != 6200 {
		t.Fatalf("expected current threshold 6200, got %d", progress.CurrentLevelXP)
	}
	if progress.NextLevelXP == nil || *progress.NextLevelXP != 8400 {
		t.Fatalf("expected next threshold 8400, got %#v", progress.NextLevelXP)
	}
}

func TestGrantXPSupportsDailyLoginPolicy(t *testing.T) {
	now := time.Date(2026, 6, 14, 9, 0, 0, 0, time.UTC)
	userID := userdomain.NewGeneratedUserID()
	repo := &fakeRepository{}
	uc := NewUseCase(repo, func() time.Time { return now })

	err := uc.GrantXP(context.Background(), GrantXPInput{
		UserID:     userID,
		ActorID:    userID,
		SourceType: XPSourceDailyLogin,
		SourceID:   "2026-06-14",
	})
	if err != nil {
		t.Fatalf("GrantXP returned error: %v", err)
	}
	if len(repo.xpGrants) != 1 {
		t.Fatalf("expected one xp grant, got %d", len(repo.xpGrants))
	}
	grant := repo.xpGrants[0]
	if grant.Delta != 5 || grant.DailyCap != 5 || grant.Reason != XPReasonDailyLogin {
		t.Fatalf("unexpected daily login policy: %#v", grant)
	}
}

func TestCreateTitleRejectsReservedWords(t *testing.T) {
	uc := NewUseCase(&fakeRepository{isStaff: true}, time.Now)

	_, err := uc.CreateTitle(context.Background(), CreateTitleInput{
		ActorID: userdomain.NewGeneratedUserID(),
		Name:    "官方认证",
	})
	if !apperr.IsCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for reserved title, got %v", err)
	}
}

func TestGrantTitleWritesAuditInTransaction(t *testing.T) {
	now := time.Date(2026, 6, 14, 9, 0, 0, 0, time.UTC)
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	titleID := userdomain.NewGeneratedUserID().String()
	repo := &fakeRepository{
		isStaff: true,
		users: map[string]User{
			targetID.String(): {ID: targetID.String(), Username: "alice", Status: "active"},
		},
		titles: map[string]Title{
			titleID: {ID: titleID, Name: "热心同学", ScopeType: TitleScopePlatform, IsActive: true},
		},
	}
	uc := NewUseCase(repo, func() time.Time { return now })

	result, err := uc.GrantTitle(context.Background(), GrantTitleInput{
		ActorID: actorID,
		UserID:  targetID.String(),
		TitleID: titleID,
		Reason:  "helpful",
	})
	if err != nil {
		t.Fatalf("GrantTitle returned error: %v", err)
	}
	if result.Grant.Title.Name != "热心同学" {
		t.Fatalf("unexpected grant: %#v", result.Grant)
	}
	if repo.txCount != 1 {
		t.Fatalf("expected transaction once, got %d", repo.txCount)
	}
	if len(repo.auditLogs) != 1 {
		t.Fatalf("expected one audit log, got %d", len(repo.auditLogs))
	}
	if repo.auditLogs[0].Action != "admin.titles.grant" || repo.auditLogs[0].TargetID != targetID.String() {
		t.Fatalf("unexpected audit log: %#v", repo.auditLogs[0])
	}
}

type fakeRepository struct {
	isStaff   bool
	users     map[string]User
	titles    map[string]Title
	grants    map[string]TitleGrant
	auditLogs []AdminAuditLog
	xpGrants  []GrantXPRecordInput
	txCount   int
}

func (f *fakeRepository) WithinTx(ctx context.Context, fn func(ctx context.Context, repository Repository) error) error {
	f.txCount++
	return fn(ctx, f)
}

func (f *fakeRepository) IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error) {
	return f.isStaff, nil
}

func (f *fakeRepository) GetOrCreateProgression(ctx context.Context, userID userdomain.UserID, now time.Time) (ProgressionRecord, error) {
	return ProgressionRecord{UserID: userID.String(), UpdatedAt: now}, nil
}

func (f *fakeRepository) GetPublicProgression(ctx context.Context, userID userdomain.UserID, now time.Time) (ProgressionRecord, error) {
	return ProgressionRecord{UserID: userID.String(), UpdatedAt: now}, nil
}

func (f *fakeRepository) ListXPEvents(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]XPEvent, error) {
	return nil, nil
}

func (f *fakeRepository) GrantXP(ctx context.Context, input GrantXPRecordInput) (GrantXPRecordResult, error) {
	f.xpGrants = append(f.xpGrants, input)
	return GrantXPRecordResult{Granted: true}, nil
}

func (f *fakeRepository) ListTitles(ctx context.Context, filter TitleFilter, limit int, offset int) ([]Title, error) {
	return nil, nil
}

func (f *fakeRepository) CreateTitle(ctx context.Context, input CreateTitleRecordInput) (Title, error) {
	title := Title{ID: input.ID, Name: input.Name, Description: input.Description, ScopeType: input.ScopeType, ScopeID: input.ScopeID, IsActive: true, CreatedBy: input.CreatedBy.String(), CreatedAt: input.CreatedAt, UpdatedAt: input.CreatedAt}
	if f.titles == nil {
		f.titles = map[string]Title{}
	}
	f.titles[title.ID] = title
	return title, nil
}

func (f *fakeRepository) UpdateTitle(ctx context.Context, input UpdateTitleRecordInput) (Title, error) {
	title, ok := f.titles[input.TitleID]
	if !ok {
		return Title{}, apperr.New(apperr.CodeNotFound, "title not found")
	}
	title.Name = input.Name
	title.Description = input.Description
	title.IsActive = input.IsActive
	title.UpdatedAt = input.UpdatedAt
	f.titles[input.TitleID] = title
	return title, nil
}

func (f *fakeRepository) FindTitleByID(ctx context.Context, titleID string) (Title, error) {
	title, ok := f.titles[titleID]
	if !ok {
		return Title{}, apperr.New(apperr.CodeNotFound, "title not found")
	}
	return title, nil
}

func (f *fakeRepository) ListUserTitleGrants(ctx context.Context, userID userdomain.UserID, now time.Time, limit int, offset int) ([]TitleGrant, error) {
	return nil, nil
}

func (f *fakeRepository) GrantTitle(ctx context.Context, input GrantTitleRecordInput) (TitleGrant, error) {
	title, ok := f.titles[input.TitleID]
	if !ok {
		return TitleGrant{}, apperr.New(apperr.CodeNotFound, "title not found")
	}
	grant := TitleGrant{ID: input.ID, UserID: input.UserID.String(), Title: title, GrantedBy: input.GrantedBy.String(), Reason: input.Reason, ExpiresAt: input.ExpiresAt, CreatedAt: input.CreatedAt}
	if f.grants == nil {
		f.grants = map[string]TitleGrant{}
	}
	f.grants[grant.ID] = grant
	return grant, nil
}

func (f *fakeRepository) RevokeTitle(ctx context.Context, grantID string, revokedAt time.Time) (TitleGrant, error) {
	grant, ok := f.grants[grantID]
	if !ok {
		return TitleGrant{}, apperr.New(apperr.CodeNotFound, "title grant not found")
	}
	grant.RevokedAt = &revokedAt
	f.grants[grantID] = grant
	return grant, nil
}

func (f *fakeRepository) SetActiveTitle(ctx context.Context, userID userdomain.UserID, grantID *string, now time.Time) (ProgressionRecord, error) {
	return ProgressionRecord{UserID: userID.String(), UpdatedAt: now}, nil
}

func (f *fakeRepository) FindUserByID(ctx context.Context, userID userdomain.UserID) (User, error) {
	user, ok := f.users[userID.String()]
	if !ok {
		return User{}, apperr.New(apperr.CodeNotFound, "user not found")
	}
	return user, nil
}

func (f *fakeRepository) CreateAdminAuditLog(ctx context.Context, log AdminAuditLog) error {
	if log.ID == "" {
		return errors.New("audit id is required")
	}
	f.auditLogs = append(f.auditLogs, log)
	return nil
}

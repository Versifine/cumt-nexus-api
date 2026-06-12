package adminusecase

import (
	"context"
	"errors"
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

type fakeRepository struct {
	isStaff     bool
	users       map[string]User
	communities map[string]Community
	effects     map[string]Effect
	settings    map[string]Setting
	auditLogs   []AuditLog
	txCount     int
}

func (f *fakeRepository) WithinTx(ctx context.Context, fn func(ctx context.Context, repository Repository) error) error {
	f.txCount++
	return fn(ctx, f)
}

func (f *fakeRepository) IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error) {
	return f.isStaff, nil
}

func (f *fakeRepository) ListUsers(ctx context.Context, status string, limit int, offset int) ([]User, error) {
	users := make([]User, 0, len(f.users))
	for _, user := range f.users {
		if status == "all" || user.Status == status {
			users = append(users, user)
		}
	}
	return users, nil
}

func (f *fakeRepository) FindUserByID(ctx context.Context, userID userdomain.UserID) (User, error) {
	user, ok := f.users[userID.String()]
	if !ok {
		return User{}, apperr.New(apperr.CodeNotFound, "user not found")
	}
	return user, nil
}

func (f *fakeRepository) UpdateUser(ctx context.Context, userID userdomain.UserID, input UpdateUserRecordInput) (User, error) {
	user, ok := f.users[userID.String()]
	if !ok {
		return User{}, apperr.New(apperr.CodeNotFound, "user not found")
	}
	user.Status = input.Status
	user.IsPlatformStaff = input.IsPlatformStaff
	user.UpdatedAt = input.UpdatedAt
	f.users[userID.String()] = user
	return user, nil
}

func (f *fakeRepository) ListCommunities(ctx context.Context, status string, limit int, offset int) ([]Community, error) {
	communities := make([]Community, 0, len(f.communities))
	for _, community := range f.communities {
		if status == "all" || community.Status == status {
			communities = append(communities, community)
		}
	}
	return communities, nil
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

func (f *fakeRepository) CreateAuditLog(ctx context.Context, log AuditLog) error {
	if log.ID == "" {
		return errors.New("audit log id is required")
	}
	f.auditLogs = append(f.auditLogs, log)
	return nil
}

func (f *fakeRepository) ListAuditLogs(ctx context.Context, targetType string, targetID string, limit int, offset int) ([]AuditLog, error) {
	return f.auditLogs, nil
}

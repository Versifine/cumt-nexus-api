package adminhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/admin/adminusecase"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

func TestUpdateCommunityOwnerParsesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	admin := &fakeAdminUseCase{
		updateCommunityOwnerResult: adminusecase.UpdateCommunityOwnerResult{
			Community: adminusecase.Community{
				ID:          "8f92e975-5323-4a58-bac1-1336b668183c",
				Slug:        "campus",
				Name:        "Campus",
				Description: "Campus community",
				Kind:        "user",
				Status:      "active",
				Visibility:  "public",
				CreatedBy:   actorID.String(),
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			Owner: adminusecase.CommunityOwnerMember{
				UserID:    targetID.String(),
				Username:  "alice",
				Role:      "owner",
				Status:    "active",
				UpdatedAt: now,
			},
		},
	}
	router := newAdminTestRouter(admin, actorID)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/communities/8f92e975-5323-4a58-bac1-1336b668183c/owner", bytes.NewBufferString(`{"user_id":"`+targetID.String()+`","reason":"owner left campus"}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !admin.updateCommunityOwnerCalled {
		t.Fatal("expected UpdateCommunityOwner to be called")
	}
	if admin.updateCommunityOwnerInput.ActorID != actorID ||
		admin.updateCommunityOwnerInput.CommunityID != "8f92e975-5323-4a58-bac1-1336b668183c" ||
		admin.updateCommunityOwnerInput.UserID != targetID.String() ||
		admin.updateCommunityOwnerInput.Reason != "owner left campus" {
		t.Fatalf("unexpected update community owner input: %#v", admin.updateCommunityOwnerInput)
	}

	var response updateAdminCommunityOwnerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Owner.UserID != targetID.String() || response.Owner.Role != "owner" {
		t.Fatalf("unexpected owner response: %#v", response.Owner)
	}
}

func TestListUsersParsesSearchQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	actorID := userdomain.NewGeneratedUserID()
	admin := &fakeAdminUseCase{}
	router := newAdminTestRouter(admin, actorID)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?status=active&q=alice&limit=10&offset=5", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !admin.listUsersCalled {
		t.Fatal("expected ListUsers to be called")
	}
	if admin.listUsersInput.ActorID != actorID ||
		admin.listUsersInput.Status != "active" ||
		admin.listUsersInput.Query != "alice" ||
		admin.listUsersInput.Limit != 10 ||
		admin.listUsersInput.Offset != 5 {
		t.Fatalf("unexpected list users input: %#v", admin.listUsersInput)
	}
}

func TestListCommunitiesParsesSearchQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	actorID := userdomain.NewGeneratedUserID()
	admin := &fakeAdminUseCase{}
	router := newAdminTestRouter(admin, actorID)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/communities?status=active&q=campus&limit=10&offset=5", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !admin.listCommunitiesCalled {
		t.Fatal("expected ListCommunities to be called")
	}
	if admin.listCommunitiesInput.ActorID != actorID ||
		admin.listCommunitiesInput.Status != "active" ||
		admin.listCommunitiesInput.Query != "campus" ||
		admin.listCommunitiesInput.Limit != 10 ||
		admin.listCommunitiesInput.Offset != 5 {
		t.Fatalf("unexpected list communities input: %#v", admin.listCommunitiesInput)
	}
}

func TestUpdateUserPlatformRoleParsesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 14, 13, 0, 0, 0, time.UTC)
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	admin := &fakeAdminUseCase{
		updateUserPlatformRoleResult: adminusecase.UpdateUserPlatformRoleResult{
			User: adminusecase.User{
				ID:              targetID.String(),
				Username:        "alice",
				Status:          "active",
				IsPlatformStaff: true,
				PlatformRole:    "admin",
				CreatedAt:       now,
				UpdatedAt:       now,
			},
		},
	}
	router := newAdminTestRouter(admin, actorID)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/"+targetID.String()+"/platform-role", bytes.NewBufferString(`{"role":"admin"}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !admin.updateUserPlatformRoleCalled {
		t.Fatal("expected UpdateUserPlatformRole to be called")
	}
	if admin.updateUserPlatformRoleInput.ActorID != actorID ||
		admin.updateUserPlatformRoleInput.UserID != targetID.String() ||
		admin.updateUserPlatformRoleInput.Role == nil ||
		*admin.updateUserPlatformRoleInput.Role != "admin" {
		t.Fatalf("unexpected update platform role input: %#v", admin.updateUserPlatformRoleInput)
	}

	var response updateAdminUserPlatformRoleResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.User.PlatformRole != "admin" || !response.User.IsPlatformStaff {
		t.Fatalf("unexpected user response: %#v", response.User)
	}
}

func TestCreateOwnerTransferParsesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)
	actorID := userdomain.NewGeneratedUserID()
	targetID := userdomain.NewGeneratedUserID()
	previousRole := "admin"
	admin := &fakeAdminUseCase{
		createOwnerTransferResult: adminusecase.CreateOwnerTransferResult{
			Transfer: adminusecase.OwnerTransfer{
				ID:                  "8f92e975-5323-4a58-bac1-1336b668183c",
				Status:              adminusecase.OwnerTransferStatusPending,
				InitiatedByID:       actorID.String(),
				InitiatedByUsername: "owner",
				TargetUserID:        targetID.String(),
				TargetUsername:      "alice",
				PreviousOwnerRole:   "admin",
				Reason:              "graduation",
				CreatedAt:           now,
				UpdatedAt:           now,
				ExpiresAt:           now.Add(48 * time.Hour),
			},
		},
	}
	router := newAdminTestRouter(admin, actorID)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/owner-transfer", bytes.NewBufferString(`{"target_user_id":"`+targetID.String()+`","previous_owner_role":"admin","reason":"graduation","current_password":"correct-password"}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if !admin.createOwnerTransferCalled {
		t.Fatal("expected CreateOwnerTransfer to be called")
	}
	if admin.createOwnerTransferInput.ActorID != actorID ||
		admin.createOwnerTransferInput.TargetUserID != targetID.String() ||
		admin.createOwnerTransferInput.PreviousOwnerRole == nil ||
		*admin.createOwnerTransferInput.PreviousOwnerRole != previousRole ||
		admin.createOwnerTransferInput.CurrentPassword != "correct-password" {
		t.Fatalf("unexpected create owner transfer input: %#v", admin.createOwnerTransferInput)
	}
	var response ownerTransferResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Transfer == nil || response.Transfer.TargetUserID != targetID.String() {
		t.Fatalf("unexpected owner transfer response: %#v", response.Transfer)
	}
}

func TestListOwnerTransfersParsesQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 16, 9, 0, 0, 0, time.UTC)
	actorID := userdomain.NewGeneratedUserID()
	admin := &fakeAdminUseCase{
		listOwnerTransfersResult: adminusecase.ListOwnerTransfersResult{
			Transfers: []adminusecase.OwnerTransfer{
				{
					ID:                  "8f92e975-5323-4a58-bac1-1336b668183c",
					Status:              adminusecase.OwnerTransferStatusPending,
					InitiatedByID:       userdomain.NewGeneratedUserID().String(),
					InitiatedByUsername: "owner",
					TargetUserID:        actorID.String(),
					TargetUsername:      "alice",
					CreatedAt:           now,
					UpdatedAt:           now,
					ExpiresAt:           now.Add(48 * time.Hour),
				},
			},
			Status:     adminusecase.OwnerTransferStatusPending,
			Limit:      10,
			Offset:     5,
			NextOffset: 6,
		},
	}
	router := newAdminTestRouter(admin, actorID)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/owner-transfers?status=pending&limit=10&offset=5", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !admin.listOwnerTransfersCalled {
		t.Fatal("expected ListOwnerTransfers to be called")
	}
	if admin.listOwnerTransfersInput.ActorID != actorID ||
		admin.listOwnerTransfersInput.Status != "pending" ||
		admin.listOwnerTransfersInput.Limit != 10 ||
		admin.listOwnerTransfersInput.Offset != 5 {
		t.Fatalf("unexpected list owner transfers input: %#v", admin.listOwnerTransfersInput)
	}

	var response listOwnerTransfersResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Transfers) != 1 || !response.Transfers[0].ViewerIsTarget || !response.Transfers[0].ViewerCanAccept {
		t.Fatalf("unexpected owner transfer list response: %#v", response)
	}
}

func newAdminTestRouter(admin UseCase, userID userdomain.UserID) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authcontext.WithCurrentUserID(c.Request.Context(), userID))
		c.Next()
	})
	RegisterRoutes(router.Group("/api/v1"), NewHandler(admin))
	return router
}

type fakeAdminUseCase struct {
	listUsersCalled              bool
	listUsersInput               adminusecase.ListUsersInput
	listCommunitiesCalled        bool
	listCommunitiesInput         adminusecase.ListCommunitiesInput
	updateCommunityOwnerCalled   bool
	updateCommunityOwnerInput    adminusecase.UpdateCommunityOwnerInput
	updateCommunityOwnerResult   adminusecase.UpdateCommunityOwnerResult
	updateCommunityOwnerErr      error
	updateUserPlatformRoleCalled bool
	updateUserPlatformRoleInput  adminusecase.UpdateUserPlatformRoleInput
	updateUserPlatformRoleResult adminusecase.UpdateUserPlatformRoleResult
	updateUserPlatformRoleErr    error
	createOwnerTransferCalled    bool
	createOwnerTransferInput     adminusecase.CreateOwnerTransferInput
	createOwnerTransferResult    adminusecase.CreateOwnerTransferResult
	createOwnerTransferErr       error
	listOwnerTransfersCalled     bool
	listOwnerTransfersInput      adminusecase.ListOwnerTransfersInput
	listOwnerTransfersResult     adminusecase.ListOwnerTransfersResult
	listOwnerTransfersErr        error
}

func (f *fakeAdminUseCase) ListUsers(ctx context.Context, input adminusecase.ListUsersInput) (adminusecase.ListUsersResult, error) {
	f.listUsersCalled = true
	f.listUsersInput = input
	return adminusecase.ListUsersResult{}, nil
}

func (f *fakeAdminUseCase) UpdateUser(ctx context.Context, input adminusecase.UpdateUserInput) (adminusecase.UpdateUserResult, error) {
	return adminusecase.UpdateUserResult{}, nil
}

func (f *fakeAdminUseCase) UpdateUserPlatformRole(ctx context.Context, input adminusecase.UpdateUserPlatformRoleInput) (adminusecase.UpdateUserPlatformRoleResult, error) {
	f.updateUserPlatformRoleCalled = true
	f.updateUserPlatformRoleInput = input
	return f.updateUserPlatformRoleResult, f.updateUserPlatformRoleErr
}

func (f *fakeAdminUseCase) GetCurrentOwnerTransfer(ctx context.Context, input adminusecase.GetCurrentOwnerTransferInput) (adminusecase.GetCurrentOwnerTransferResult, error) {
	return adminusecase.GetCurrentOwnerTransferResult{}, nil
}

func (f *fakeAdminUseCase) CreateOwnerTransfer(ctx context.Context, input adminusecase.CreateOwnerTransferInput) (adminusecase.CreateOwnerTransferResult, error) {
	f.createOwnerTransferCalled = true
	f.createOwnerTransferInput = input
	return f.createOwnerTransferResult, f.createOwnerTransferErr
}

func (f *fakeAdminUseCase) CancelOwnerTransfer(ctx context.Context, input adminusecase.CancelOwnerTransferInput) (adminusecase.CancelOwnerTransferResult, error) {
	return adminusecase.CancelOwnerTransferResult{}, nil
}

func (f *fakeAdminUseCase) GetOwnerTransfer(ctx context.Context, input adminusecase.GetOwnerTransferInput) (adminusecase.GetOwnerTransferResult, error) {
	return adminusecase.GetOwnerTransferResult{}, nil
}

func (f *fakeAdminUseCase) AcceptOwnerTransfer(ctx context.Context, input adminusecase.AcceptOwnerTransferInput) (adminusecase.AcceptOwnerTransferResult, error) {
	return adminusecase.AcceptOwnerTransferResult{}, nil
}

func (f *fakeAdminUseCase) ListOwnerTransfers(ctx context.Context, input adminusecase.ListOwnerTransfersInput) (adminusecase.ListOwnerTransfersResult, error) {
	f.listOwnerTransfersCalled = true
	f.listOwnerTransfersInput = input
	return f.listOwnerTransfersResult, f.listOwnerTransfersErr
}

func (f *fakeAdminUseCase) ListCommunities(ctx context.Context, input adminusecase.ListCommunitiesInput) (adminusecase.ListCommunitiesResult, error) {
	f.listCommunitiesCalled = true
	f.listCommunitiesInput = input
	return adminusecase.ListCommunitiesResult{}, nil
}

func (f *fakeAdminUseCase) UpdateCommunityStatus(ctx context.Context, input adminusecase.UpdateCommunityStatusInput) (adminusecase.UpdateCommunityStatusResult, error) {
	return adminusecase.UpdateCommunityStatusResult{}, nil
}

func (f *fakeAdminUseCase) UpdateCommunityOwner(ctx context.Context, input adminusecase.UpdateCommunityOwnerInput) (adminusecase.UpdateCommunityOwnerResult, error) {
	f.updateCommunityOwnerCalled = true
	f.updateCommunityOwnerInput = input
	return f.updateCommunityOwnerResult, f.updateCommunityOwnerErr
}

func (f *fakeAdminUseCase) ListEffects(ctx context.Context, input adminusecase.ListEffectsInput) (adminusecase.ListEffectsResult, error) {
	return adminusecase.ListEffectsResult{}, nil
}

func (f *fakeAdminUseCase) UpdateEffectActive(ctx context.Context, input adminusecase.UpdateEffectActiveInput) (adminusecase.UpdateEffectActiveResult, error) {
	return adminusecase.UpdateEffectActiveResult{}, nil
}

func (f *fakeAdminUseCase) ListSettings(ctx context.Context, input adminusecase.ListSettingsInput) (adminusecase.ListSettingsResult, error) {
	return adminusecase.ListSettingsResult{}, nil
}

func (f *fakeAdminUseCase) UpdateSetting(ctx context.Context, input adminusecase.UpdateSettingInput) (adminusecase.UpdateSettingResult, error) {
	return adminusecase.UpdateSettingResult{}, nil
}

func (f *fakeAdminUseCase) ListAuditLogs(ctx context.Context, input adminusecase.ListAuditLogsInput) (adminusecase.ListAuditLogsResult, error) {
	return adminusecase.ListAuditLogsResult{}, nil
}

func (f *fakeAdminUseCase) ListPointTransactions(ctx context.Context, input adminusecase.ListPointTransactionsInput) (adminusecase.ListPointTransactionsResult, error) {
	return adminusecase.ListPointTransactionsResult{}, nil
}

func (f *fakeAdminUseCase) AdjustUserPoints(ctx context.Context, input adminusecase.AdjustUserPointsInput) (adminusecase.AdjustUserPointsResult, error) {
	return adminusecase.AdjustUserPointsResult{}, nil
}

func (f *fakeAdminUseCase) CreateUserSanction(ctx context.Context, input adminusecase.CreateUserSanctionInput) (adminusecase.CreateUserSanctionResult, error) {
	return adminusecase.CreateUserSanctionResult{}, nil
}

func (f *fakeAdminUseCase) ListUserSanctions(ctx context.Context, input adminusecase.ListUserSanctionsInput) (adminusecase.ListUserSanctionsResult, error) {
	return adminusecase.ListUserSanctionsResult{}, nil
}

func (f *fakeAdminUseCase) RevokeUserSanction(ctx context.Context, input adminusecase.RevokeUserSanctionInput) (adminusecase.RevokeUserSanctionResult, error) {
	return adminusecase.RevokeUserSanctionResult{}, nil
}

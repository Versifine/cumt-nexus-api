package communityhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authtoken"
	"github.com/Versifine/cumt-nexus-api/internal/auth/delivery/authhttp"
	"github.com/Versifine/cumt-nexus-api/internal/community/communityusecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

func TestListCommunitiesReturnsCommunities(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	communities := &fakeCommunityReadUseCase{
		listResult: communityusecase.ListCommunitiesResult{
			Communities: []communityusecase.Community{
				newCommunityResult("public", now),
				newCommunityResult("campus", now.Add(time.Minute)),
			},
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.listCalled {
		t.Fatal("expected ListCommunities to be called")
	}

	var response listCommunitiesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Communities) != 2 {
		t.Fatalf("expected two communities, got %d", len(response.Communities))
	}
	if response.Communities[0].Slug != "public" || response.Communities[1].Slug != "campus" {
		t.Fatalf("unexpected communities response: %#v", response.Communities)
	}
}

func TestListCommunitiesAllowsAnonymousViewer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	communities := &fakeCommunityReadUseCase{
		listResult: communityusecase.ListCommunitiesResult{
			Communities: []communityusecase.Community{newCommunityResult("public", now)},
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.listCalled {
		t.Fatal("expected ListCommunities to be called")
	}
}

func TestGetCommunityReturnsCommunity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	communities := &fakeCommunityReadUseCase{
		getResult: communityusecase.GetCommunityResult{
			Community: newCommunityResult("campus", now),
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/campus", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.getCalled {
		t.Fatal("expected GetCommunityBySlug to be called")
	}
	if communities.getInput.Slug != "campus" {
		t.Fatalf("expected slug campus, got %q", communities.getInput.Slug)
	}

	var response getCommunityResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Community.Slug != "campus" {
		t.Fatalf("expected response slug campus, got %q", response.Community.Slug)
	}
}

func TestFollowCommunityReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	communities := &fakeCommunityReadUseCase{}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/communities/campus/follow", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, recorder.Code, recorder.Body.String())
	}
	if !communities.followCalled {
		t.Fatal("expected FollowCommunity to be called")
	}
	if communities.followInput.Slug != "campus" || communities.followInput.UserID != userID {
		t.Fatalf("unexpected follow input: %#v", communities.followInput)
	}
}

func TestListFollowedCommunitiesReturnsCommunities(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	communities := &fakeCommunityReadUseCase{
		listFollowedResult: communityusecase.ListFollowedCommunitiesResult{
			Communities: []communityusecase.Community{newCommunityResult("campus", now)},
			Limit:       20,
			Offset:      5,
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/followed-communities?limit=20&offset=5", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.listFollowedCalled {
		t.Fatal("expected ListFollowedCommunities to be called")
	}
	if communities.listFollowedInput.UserID != userID || communities.listFollowedInput.Limit != 20 || communities.listFollowedInput.Offset != 5 {
		t.Fatalf("unexpected list followed input: %#v", communities.listFollowedInput)
	}
	var response listFollowedCommunitiesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Communities) != 1 || response.Communities[0].Slug != "campus" {
		t.Fatalf("unexpected followed communities response: %#v", response.Communities)
	}
}

func TestGetCommunityManageContextReturnsCommunity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC)
	community := newCommunityResult("campus", now)
	community.ViewerRole = "owner"
	community.ViewerPermissions.CanManage = true
	communities := &fakeCommunityReadUseCase{
		manageResult: communityusecase.GetCommunityManageContextResult{
			Community: community,
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/campus/manage", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.manageCalled {
		t.Fatal("expected GetCommunityManageContext to be called")
	}
	if communities.manageInput.Slug != "campus" || communities.manageInput.ViewerID != userID {
		t.Fatalf("unexpected manage input: %#v", communities.manageInput)
	}
	var response getCommunityManageContextResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Community.ViewerRole != "owner" || !response.Community.ViewerPermissions.CanManage {
		t.Fatalf("unexpected manage community response: %#v", response.Community)
	}
}

func TestListCommunityMembersReturnsMembers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 9, 10, 30, 0, 0, time.UTC)
	communities := &fakeCommunityReadUseCase{
		listMembersResult: communityusecase.ListCommunityMembersResult{
			Community: newCommunityResult("campus", now),
			Members: []communityusecase.CommunityMember{
				{
					UserID:    userdomain.NewGeneratedUserID().String(),
					Username:  "alice",
					Role:      "owner",
					Status:    "active",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			Limit:  20,
			Offset: 5,
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/campus/manage/members?limit=20&offset=5", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.listMembersCalled {
		t.Fatal("expected ListCommunityMembers to be called")
	}
	if communities.listMembersInput.Slug != "campus" || communities.listMembersInput.ViewerID != userID || communities.listMembersInput.Limit != 20 || communities.listMembersInput.Offset != 5 {
		t.Fatalf("unexpected list members input: %#v", communities.listMembersInput)
	}
	var response listCommunityMembersResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Members) != 1 || response.Members[0].User.Username != "alice" || response.Members[0].Role != "owner" {
		t.Fatalf("unexpected members response: %#v", response.Members)
	}
}

func TestListCommunityManagePostsReturnsPosts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 9, 13, 0, 0, 0, time.UTC)
	communities := &fakeCommunityReadUseCase{
		listManagePostsResult: communityusecase.ListCommunityManagePostsResult{
			Community: newCommunityResult("campus", now),
			Posts: []communityusecase.CommunityManagePost{
				{
					ID:          "post-1",
					CommunityID: "community-1",
					AuthorID:    "author-1",
					Title:       "Pinned topic",
					BodyExcerpt: "body preview",
					Status:      "visible",
					CreatedAt:   now,
					UpdatedAt:   now,
				},
			},
			Status: "visible",
			Limit:  20,
			Offset: 5,
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/campus/manage/posts?status=visible&limit=20&offset=5", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.listManagePostsCalled {
		t.Fatal("expected ListCommunityManagePosts to be called")
	}
	if communities.listManagePostsInput.Slug != "campus" || communities.listManagePostsInput.ViewerID != userID || communities.listManagePostsInput.Status != "visible" || communities.listManagePostsInput.Limit != 20 || communities.listManagePostsInput.Offset != 5 {
		t.Fatalf("unexpected list manage posts input: %#v", communities.listManagePostsInput)
	}
	var response listCommunityManagePostsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Status != "visible" || len(response.Posts) != 1 || response.Posts[0].Title != "Pinned topic" {
		t.Fatalf("unexpected manage posts response: %#v", response)
	}
}

func TestGetCommunityManageSettingsReturnsSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)
	communities := &fakeCommunityReadUseCase{
		getSettingsResult: communityusecase.GetCommunityManageSettingsResult{
			Community: newCommunityResult("campus", now),
			Settings: communityusecase.CommunitySettings{
				Name:        "Campus",
				Description: "Updated community",
				UpdatedAt:   now,
			},
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/campus/manage/settings", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.getSettingsCalled {
		t.Fatal("expected GetCommunityManageSettings to be called")
	}
	if communities.getSettingsInput.Slug != "campus" || communities.getSettingsInput.ViewerID != userID {
		t.Fatalf("unexpected get settings input: %#v", communities.getSettingsInput)
	}
	var response getCommunityManageSettingsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Settings.Name != "Campus" || response.Settings.Description != "Updated community" {
		t.Fatalf("unexpected settings response: %#v", response.Settings)
	}
}

func TestUpdateCommunityManageSettingsParsesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 10, 9, 30, 0, 0, time.UTC)
	communities := &fakeCommunityReadUseCase{
		updateSettingsResult: communityusecase.UpdateCommunityManageSettingsResult{
			Community: newCommunityResult("campus", now),
			Settings: communityusecase.CommunitySettings{
				Name:        "Campus Hub",
				Description: "Rules and updates",
				UpdatedAt:   now,
			},
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/communities/campus/manage/settings", bytes.NewBufferString(`{
		"name": "Campus Hub",
		"description": "Rules and updates"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.updateSettingsCalled {
		t.Fatal("expected UpdateCommunityManageSettings to be called")
	}
	if communities.updateSettingsInput.Slug != "campus" || communities.updateSettingsInput.ViewerID != userID || communities.updateSettingsInput.Name != "Campus Hub" || communities.updateSettingsInput.Description != "Rules and updates" {
		t.Fatalf("unexpected update settings input: %#v", communities.updateSettingsInput)
	}
	var response updateCommunityManageSettingsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Settings.Name != "Campus Hub" {
		t.Fatalf("unexpected settings response: %#v", response.Settings)
	}
}

func TestListCommunityRulesReturnsRules(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	actorID := userdomain.NewGeneratedUserID().String()
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	communities := &fakeCommunityReadUseCase{
		listRulesResult: communityusecase.ListCommunityRulesResult{
			Community: newCommunityResult("campus", now),
			Rules: []communityusecase.CommunityRule{
				{
					ID:          "8f92e975-5323-4a58-bac1-1336b668183c",
					CommunityID: "community-1",
					Title:       "Be kind",
					Body:        "Keep discussions constructive.",
					Position:    1,
					CreatedBy:   actorID,
					UpdatedBy:   actorID,
					CreatedAt:   now,
					UpdatedAt:   now,
				},
			},
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/campus/manage/rules", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.listRulesCalled {
		t.Fatal("expected ListCommunityRules to be called")
	}
	if communities.listRulesInput.Slug != "campus" || communities.listRulesInput.ViewerID != userID {
		t.Fatalf("unexpected list rules input: %#v", communities.listRulesInput)
	}
	var response listCommunityRulesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.Rules) != 1 || response.Rules[0].Title != "Be kind" || response.Rules[0].Position != 1 {
		t.Fatalf("unexpected rules response: %#v", response)
	}
}

func TestCreateCommunityRuleParsesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	actorID := userdomain.NewGeneratedUserID().String()
	now := time.Date(2026, 6, 10, 10, 30, 0, 0, time.UTC)
	communities := &fakeCommunityReadUseCase{
		createRuleResult: communityusecase.CreateCommunityRuleResult{
			Community: newCommunityResult("campus", now),
			Rule: communityusecase.CommunityRule{
				ID:          "8f92e975-5323-4a58-bac1-1336b668183c",
				CommunityID: "community-1",
				Title:       "Stay on topic",
				Body:        "Posts should match the community.",
				Position:    2,
				CreatedBy:   actorID,
				UpdatedBy:   actorID,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/communities/campus/manage/rules", bytes.NewBufferString(`{
		"title": "Stay on topic",
		"body": "Posts should match the community.",
		"position": 2
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if !communities.createRuleCalled {
		t.Fatal("expected CreateCommunityRule to be called")
	}
	if communities.createRuleInput.Slug != "campus" || communities.createRuleInput.ViewerID != userID || communities.createRuleInput.Title != "Stay on topic" || communities.createRuleInput.Body != "Posts should match the community." || communities.createRuleInput.Position != 2 {
		t.Fatalf("unexpected create rule input: %#v", communities.createRuleInput)
	}
	var response createCommunityRuleResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Rule.Title != "Stay on topic" || response.Rule.Position != 2 {
		t.Fatalf("unexpected rule response: %#v", response.Rule)
	}
}

func TestUpdateCommunityRuleParsesBodyAndRuleID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	actorID := userdomain.NewGeneratedUserID().String()
	ruleID := "8f92e975-5323-4a58-bac1-1336b668183c"
	now := time.Date(2026, 6, 10, 11, 0, 0, 0, time.UTC)
	communities := &fakeCommunityReadUseCase{
		updateRuleResult: communityusecase.UpdateCommunityRuleResult{
			Community: newCommunityResult("campus", now),
			Rule: communityusecase.CommunityRule{
				ID:          ruleID,
				CommunityID: "community-1",
				Title:       "Updated title",
				Body:        "Updated body.",
				Position:    3,
				CreatedBy:   actorID,
				UpdatedBy:   actorID,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/communities/campus/manage/rules/"+ruleID, bytes.NewBufferString(`{
		"title": "Updated title",
		"body": "Updated body.",
		"position": 3
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.updateRuleCalled {
		t.Fatal("expected UpdateCommunityRule to be called")
	}
	if communities.updateRuleInput.Slug != "campus" || communities.updateRuleInput.RuleID != ruleID || communities.updateRuleInput.ViewerID != userID || communities.updateRuleInput.Title != "Updated title" || communities.updateRuleInput.Body != "Updated body." || communities.updateRuleInput.Position != 3 {
		t.Fatalf("unexpected update rule input: %#v", communities.updateRuleInput)
	}
	var response updateCommunityRuleResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Rule.ID != ruleID || response.Rule.Title != "Updated title" {
		t.Fatalf("unexpected rule response: %#v", response.Rule)
	}
}

func TestDeleteCommunityRuleReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	ruleID := "8f92e975-5323-4a58-bac1-1336b668183c"
	communities := &fakeCommunityReadUseCase{}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/communities/campus/manage/rules/"+ruleID, nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, recorder.Code, recorder.Body.String())
	}
	if !communities.deleteRuleCalled {
		t.Fatal("expected DeleteCommunityRule to be called")
	}
	if communities.deleteRuleInput.Slug != "campus" || communities.deleteRuleInput.RuleID != ruleID || communities.deleteRuleInput.ViewerID != userID {
		t.Fatalf("unexpected delete rule input: %#v", communities.deleteRuleInput)
	}
}

func TestAddCommunityModeratorParsesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	userID := userdomain.NewGeneratedUserID()
	memberID := userdomain.NewGeneratedUserID().String()
	communities := &fakeCommunityReadUseCase{
		addModeratorResult: communityusecase.CommunityMemberMutationResult{
			Community: newCommunityResult("campus", now),
			Member: communityusecase.CommunityMember{
				UserID:    memberID,
				Username:  "alice",
				Role:      "moderator",
				Status:    "active",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/communities/campus/manage/moderators", bytes.NewBufferString(`{"username":"alice"}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.addModeratorCalled {
		t.Fatal("expected AddCommunityModerator to be called")
	}
	if communities.addModeratorInput.Slug != "campus" || communities.addModeratorInput.ViewerID != userID || communities.addModeratorInput.Username != "alice" {
		t.Fatalf("unexpected add moderator input: %#v", communities.addModeratorInput)
	}

	var response communityMemberMutationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Member.User.ID != memberID || response.Member.Role != "moderator" {
		t.Fatalf("unexpected member response: %#v", response.Member)
	}
}

func TestRemoveCommunityModeratorPassesPathParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	userID := userdomain.NewGeneratedUserID()
	memberID := userdomain.NewGeneratedUserID().String()
	communities := &fakeCommunityReadUseCase{
		removeModeratorResult: communityusecase.CommunityMemberMutationResult{
			Community: newCommunityResult("campus", now),
			Member: communityusecase.CommunityMember{
				UserID:    memberID,
				Username:  "alice",
				Role:      "member",
				Status:    "active",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/communities/campus/manage/moderators/"+memberID, nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !communities.removeModeratorCalled {
		t.Fatal("expected RemoveCommunityModerator to be called")
	}
	if communities.removeModeratorInput.Slug != "campus" || communities.removeModeratorInput.ViewerID != userID || communities.removeModeratorInput.UserID != memberID {
		t.Fatalf("unexpected remove moderator input: %#v", communities.removeModeratorInput)
	}
}

func TestCommunityOwnerTransferRoutesPassInputs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	userID := userdomain.NewGeneratedUserID()
	transferID := "8f92e975-5323-4a58-bac1-1336b668183c"
	communities := &fakeCommunityReadUseCase{
		createTransferResult: communityusecase.CommunityOwnerTransferResult{
			Community: newCommunityResult("campus", now),
			Transfer: communityusecase.CommunityOwnerTransfer{
				ID:          transferID,
				CommunityID: userdomain.NewGeneratedUserID().String(),
				FromUserID:  userID.String(),
				ToUserID:    userdomain.NewGeneratedUserID().String(),
				Status:      "pending",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		acceptTransferResult: communityusecase.CommunityOwnerTransferResult{
			Community: newCommunityResult("campus", now),
			Transfer: communityusecase.CommunityOwnerTransfer{
				ID:          transferID,
				CommunityID: userdomain.NewGeneratedUserID().String(),
				FromUserID:  userdomain.NewGeneratedUserID().String(),
				ToUserID:    userID.String(),
				Status:      "accepted",
				CreatedAt:   now,
				UpdatedAt:   now,
				AcceptedAt:  &now,
			},
		},
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParserWithUserID(userID))

	createRecorder := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/communities/campus/manage/owner-transfer", bytes.NewBufferString(`{"username":"bob"}`))
	createRequest.Header.Set("Authorization", "Bearer valid-token")
	createRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(createRecorder, createRequest)

	if createRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, createRecorder.Code, createRecorder.Body.String())
	}
	if !communities.createTransferCalled {
		t.Fatal("expected CreateCommunityOwnerTransfer to be called")
	}
	if communities.createTransferInput.Slug != "campus" || communities.createTransferInput.ViewerID != userID || communities.createTransferInput.Username != "bob" {
		t.Fatalf("unexpected create transfer input: %#v", communities.createTransferInput)
	}

	acceptRecorder := httptest.NewRecorder()
	acceptRequest := httptest.NewRequest(http.MethodPost, "/api/v1/communities/campus/manage/owner-transfer/"+transferID+"/accept", nil)
	acceptRequest.Header.Set("Authorization", "Bearer valid-token")
	router.ServeHTTP(acceptRecorder, acceptRequest)

	if acceptRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, acceptRecorder.Code, acceptRecorder.Body.String())
	}
	if !communities.acceptTransferCalled {
		t.Fatal("expected AcceptCommunityOwnerTransfer to be called")
	}
	if communities.acceptTransferInput.Slug != "campus" || communities.acceptTransferInput.ViewerID != userID || communities.acceptTransferInput.TransferID != transferID {
		t.Fatalf("unexpected accept transfer input: %#v", communities.acceptTransferInput)
	}
}

func TestCommunityManageRoutesRequireAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	communities := &fakeCommunityReadUseCase{}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/campus/manage", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertCommunityErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if communities.manageCalled {
		t.Fatal("GetCommunityManageContext should not be called without auth")
	}
}

func TestCommunityManageUseCaseErrorMapsToHTTPError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	communities := &fakeCommunityReadUseCase{
		manageErr: apperr.New(apperr.CodeForbidden, "community moderator required"),
	}
	router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/campus/manage", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, recorder.Code, recorder.Body.String())
	}
	assertCommunityErrorCode(t, recorder, apperr.CodeForbidden)
}

func TestCommunityRoutesRejectInvalidAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	communities := &fakeCommunityReadUseCase{}
	applications := &fakeCommunityApplicationUseCase{}
	router := newCommunityTestRouter(communities, applications, &fakeAccessTokenParser{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/communities", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertCommunityErrorCode(t, recorder, apperr.CodeUnauthenticated)
	if communities.listCalled || communities.getCalled || communities.manageCalled || communities.listMembersCalled || communities.addModeratorCalled || communities.removeModeratorCalled || communities.createTransferCalled || communities.acceptTransferCalled || communities.listManagePostsCalled || communities.listManageCommentsCalled || communities.listManageReportsCalled || communities.getSettingsCalled || communities.updateSettingsCalled || communities.listRulesCalled || communities.createRuleCalled || communities.updateRuleCalled || communities.deleteRuleCalled || applications.submitCalled || applications.listCalled || applications.getCalled || applications.approveCalled || applications.rejectCalled {
		t.Fatal("community usecase should not be called for invalid auth")
	}
}

func TestGetCommunityUseCaseErrorMapsToHTTPError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   apperr.Code
	}{
		{
			name:       "invalid slug",
			err:        apperr.New(apperr.CodeInvalidArgument, "community slug is invalid"),
			wantStatus: http.StatusBadRequest,
			wantCode:   apperr.CodeInvalidArgument,
		},
		{
			name:       "not found",
			err:        apperr.New(apperr.CodeNotFound, "community not found"),
			wantStatus: http.StatusNotFound,
			wantCode:   apperr.CodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			communities := &fakeCommunityReadUseCase{
				getErr: tt.err,
			}
			router := newCommunityTestRouter(communities, &fakeCommunityApplicationUseCase{}, validParser())

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/communities/bad-slug", nil)
			request.Header.Set("Authorization", "Bearer valid-token")

			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.wantStatus, recorder.Code, recorder.Body.String())
			}
			assertCommunityErrorCode(t, recorder, tt.wantCode)
			if !communities.getCalled {
				t.Fatal("expected GetCommunityBySlug to be called")
			}
		})
	}
}

func TestSubmitCommunityApplicationReturnsCreatedApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC)
	applications := &fakeCommunityApplicationUseCase{
		submitResult: communityusecase.SubmitCommunityApplicationResult{
			Application: newApplicationResult("campus", "pending", now),
		},
	}
	router := newCommunityTestRouter(&fakeCommunityReadUseCase{}, applications, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/community-applications", bytes.NewBufferString(`{
		"requested_slug": "campus",
		"requested_name": "Campus",
		"reason": "Need a campus board"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if !applications.submitCalled {
		t.Fatal("expected SubmitCommunityApplication to be called")
	}
	if applications.submitInput.ApplicantID != userID {
		t.Fatalf("expected applicant %q, got %q", userID.String(), applications.submitInput.ApplicantID.String())
	}
	if applications.submitInput.RequestedSlug != "campus" {
		t.Fatalf("expected requested slug campus, got %q", applications.submitInput.RequestedSlug)
	}

	var response submitCommunityApplicationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Application.Status != "pending" {
		t.Fatalf("expected pending application, got %q", response.Application.Status)
	}
}

func TestSubmitCommunityApplicationUseCaseErrorMapsToHTTPError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	applications := &fakeCommunityApplicationUseCase{
		submitErr: apperr.New(apperr.CodeConflict, "pending community application slug already exists"),
	}
	router := newCommunityTestRouter(&fakeCommunityReadUseCase{}, applications, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/community-applications", bytes.NewBufferString(`{
		"requested_slug": "campus",
		"requested_name": "Campus",
		"reason": "Need a campus board"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, recorder.Code, recorder.Body.String())
	}
	assertCommunityErrorCode(t, recorder, apperr.CodeConflict)
}

func TestListCommunityApplicationsReturnsApplications(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
	applications := &fakeCommunityApplicationUseCase{
		listResult: communityusecase.ListCommunityApplicationsResult{
			Applications: []communityusecase.CommunityApplication{
				newApplicationResult("campus", "approved", now),
			},
			Limit:  10,
			Offset: 5,
		},
	}
	router := newCommunityTestRouter(&fakeCommunityReadUseCase{}, applications, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/community-applications?status=approved&limit=10&offset=5", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !applications.listCalled {
		t.Fatal("expected ListCommunityApplications to be called")
	}
	if applications.listInput.ReviewerID != userID {
		t.Fatalf("expected reviewer %q, got %q", userID.String(), applications.listInput.ReviewerID.String())
	}
	if applications.listInput.Status != "approved" || applications.listInput.Limit != 10 || applications.listInput.Offset != 5 {
		t.Fatalf("unexpected list input: %#v", applications.listInput)
	}

	var response listCommunityApplicationsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Limit != 10 || response.Offset != 5 {
		t.Fatalf("expected pagination limit=10 offset=5, got limit=%d offset=%d", response.Limit, response.Offset)
	}
	if len(response.Applications) != 1 || response.Applications[0].Status != "approved" {
		t.Fatalf("unexpected applications response: %#v", response.Applications)
	}
	if response.Applications[0].ReviewedBy != nil || response.Applications[0].ReviewedAt != nil || response.Applications[0].RejectReason != "" {
		t.Fatalf("expected empty review fields, got %#v", response.Applications[0])
	}
}

func TestListCommunityApplicationsRejectsInvalidQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	applications := &fakeCommunityApplicationUseCase{}
	router := newCommunityTestRouter(&fakeCommunityReadUseCase{}, applications, validParser())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/community-applications?limit=bad", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertCommunityErrorCode(t, recorder, apperr.CodeInvalidArgument)
	if applications.listCalled {
		t.Fatal("ListCommunityApplications should not be called for invalid query")
	}
}

func TestGetCommunityApplicationReturnsApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 3, 11, 0, 0, 0, time.UTC)
	applications := &fakeCommunityApplicationUseCase{
		getResult: communityusecase.GetCommunityApplicationResult{
			Application: newApplicationResult("campus", "pending", now),
		},
	}
	router := newCommunityTestRouter(&fakeCommunityReadUseCase{}, applications, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/community-applications/8f92e975-5323-4a58-bac1-1336b668183c", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !applications.getCalled {
		t.Fatal("expected GetCommunityApplication to be called")
	}
	if applications.getInput.ReviewerID != userID {
		t.Fatalf("expected reviewer %q, got %q", userID.String(), applications.getInput.ReviewerID.String())
	}
	if applications.getInput.ApplicationID != "8f92e975-5323-4a58-bac1-1336b668183c" {
		t.Fatalf("unexpected application id %q", applications.getInput.ApplicationID)
	}

	var response getCommunityApplicationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Application.ID != "8f92e975-5323-4a58-bac1-1336b668183c" {
		t.Fatalf("unexpected application response: %#v", response.Application)
	}
}

func TestApproveCommunityApplicationReturnsCommunity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	applications := &fakeCommunityApplicationUseCase{
		approveResult: communityusecase.ApproveCommunityApplicationResult{
			Application: newApplicationResult("campus", "approved", now),
			Community:   newCommunityResult("campus", now),
		},
	}
	router := newCommunityTestRouter(&fakeCommunityReadUseCase{}, applications, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/community-applications/8f92e975-5323-4a58-bac1-1336b668183c/approve", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !applications.approveCalled {
		t.Fatal("expected ApproveCommunityApplication to be called")
	}
	if applications.approveInput.ReviewerID != userID {
		t.Fatalf("expected reviewer %q, got %q", userID.String(), applications.approveInput.ReviewerID.String())
	}
	if applications.approveInput.ApplicationID != "8f92e975-5323-4a58-bac1-1336b668183c" {
		t.Fatalf("unexpected application id %q", applications.approveInput.ApplicationID)
	}

	var response approveCommunityApplicationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Application.Status != "approved" || response.Community.Slug != "campus" {
		t.Fatalf("unexpected approve response: %#v", response)
	}
}

func TestRejectCommunityApplicationReturnsRejectedApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 1, 12, 30, 0, 0, time.UTC)
	applications := &fakeCommunityApplicationUseCase{
		rejectResult: communityusecase.RejectCommunityApplicationResult{
			Application: newApplicationResult("campus", "rejected", now),
		},
	}
	router := newCommunityTestRouter(&fakeCommunityReadUseCase{}, applications, validParserWithUserID(userID))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/community-applications/8f92e975-5323-4a58-bac1-1336b668183c/reject", bytes.NewBufferString(`{
		"reject_reason": "duplicate slug"
	}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !applications.rejectCalled {
		t.Fatal("expected RejectCommunityApplication to be called")
	}
	if applications.rejectInput.ReviewerID != userID {
		t.Fatalf("expected reviewer %q, got %q", userID.String(), applications.rejectInput.ReviewerID.String())
	}
	if applications.rejectInput.RejectReason != "duplicate slug" {
		t.Fatalf("expected reject reason, got %q", applications.rejectInput.RejectReason)
	}

	var response rejectCommunityApplicationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Application.Status != "rejected" {
		t.Fatalf("expected rejected application, got %q", response.Application.Status)
	}
}

type fakeCommunityReadUseCase struct {
	listCalled               bool
	getCalled                bool
	followCalled             bool
	deleteFollowCalled       bool
	listFollowedCalled       bool
	manageCalled             bool
	listMembersCalled        bool
	listManagePostsCalled    bool
	listManageCommentsCalled bool
	listManageReportsCalled  bool
	getSettingsCalled        bool
	updateSettingsCalled     bool
	listRulesCalled          bool
	createRuleCalled         bool
	updateRuleCalled         bool
	deleteRuleCalled         bool
	addModeratorCalled       bool
	removeModeratorCalled    bool
	createTransferCalled     bool
	acceptTransferCalled     bool
	listInput                communityusecase.ListCommunitiesInput
	getInput                 communityusecase.GetCommunityInput
	followInput              communityusecase.FollowCommunityInput
	deleteFollowInput        communityusecase.DeleteCommunityFollowInput
	listFollowedInput        communityusecase.ListFollowedCommunitiesInput
	manageInput              communityusecase.GetCommunityManageContextInput
	listMembersInput         communityusecase.ListCommunityMembersInput
	listManagePostsInput     communityusecase.ListCommunityManagePostsInput
	listManageCommentsInput  communityusecase.ListCommunityManageCommentsInput
	listManageReportsInput   communityusecase.ListCommunityManageReportsInput
	getSettingsInput         communityusecase.GetCommunityManageSettingsInput
	updateSettingsInput      communityusecase.UpdateCommunityManageSettingsInput
	listRulesInput           communityusecase.ListCommunityRulesInput
	createRuleInput          communityusecase.CreateCommunityRuleInput
	updateRuleInput          communityusecase.UpdateCommunityRuleInput
	deleteRuleInput          communityusecase.DeleteCommunityRuleInput
	addModeratorInput        communityusecase.AddCommunityModeratorInput
	removeModeratorInput     communityusecase.RemoveCommunityModeratorInput
	createTransferInput      communityusecase.CreateCommunityOwnerTransferInput
	acceptTransferInput      communityusecase.AcceptCommunityOwnerTransferInput
	listResult               communityusecase.ListCommunitiesResult
	getResult                communityusecase.GetCommunityResult
	followResult             communityusecase.FollowCommunityResult
	deleteFollowResult       communityusecase.DeleteCommunityFollowResult
	listFollowedResult       communityusecase.ListFollowedCommunitiesResult
	manageResult             communityusecase.GetCommunityManageContextResult
	listMembersResult        communityusecase.ListCommunityMembersResult
	listManagePostsResult    communityusecase.ListCommunityManagePostsResult
	listManageCommentsResult communityusecase.ListCommunityManageCommentsResult
	listManageReportsResult  communityusecase.ListCommunityManageReportsResult
	getSettingsResult        communityusecase.GetCommunityManageSettingsResult
	updateSettingsResult     communityusecase.UpdateCommunityManageSettingsResult
	listRulesResult          communityusecase.ListCommunityRulesResult
	createRuleResult         communityusecase.CreateCommunityRuleResult
	updateRuleResult         communityusecase.UpdateCommunityRuleResult
	deleteRuleResult         communityusecase.DeleteCommunityRuleResult
	addModeratorResult       communityusecase.CommunityMemberMutationResult
	removeModeratorResult    communityusecase.CommunityMemberMutationResult
	createTransferResult     communityusecase.CommunityOwnerTransferResult
	acceptTransferResult     communityusecase.CommunityOwnerTransferResult
	listErr                  error
	getErr                   error
	followErr                error
	deleteFollowErr          error
	listFollowedErr          error
	manageErr                error
	listMembersErr           error
	listManagePostsErr       error
	listManageCommentsErr    error
	listManageReportsErr     error
	getSettingsErr           error
	updateSettingsErr        error
	listRulesErr             error
	createRuleErr            error
	updateRuleErr            error
	deleteRuleErr            error
	addModeratorErr          error
	removeModeratorErr       error
	createTransferErr        error
	acceptTransferErr        error
}

func (f *fakeCommunityReadUseCase) ListCommunities(ctx context.Context, input communityusecase.ListCommunitiesInput) (communityusecase.ListCommunitiesResult, error) {
	f.listCalled = true
	f.listInput = input
	return f.listResult, f.listErr
}

func (f *fakeCommunityReadUseCase) GetCommunityBySlug(ctx context.Context, input communityusecase.GetCommunityInput) (communityusecase.GetCommunityResult, error) {
	f.getCalled = true
	f.getInput = input
	return f.getResult, f.getErr
}

func (f *fakeCommunityReadUseCase) FollowCommunity(ctx context.Context, input communityusecase.FollowCommunityInput) (communityusecase.FollowCommunityResult, error) {
	f.followCalled = true
	f.followInput = input
	return f.followResult, f.followErr
}

func (f *fakeCommunityReadUseCase) DeleteCommunityFollow(ctx context.Context, input communityusecase.DeleteCommunityFollowInput) (communityusecase.DeleteCommunityFollowResult, error) {
	f.deleteFollowCalled = true
	f.deleteFollowInput = input
	return f.deleteFollowResult, f.deleteFollowErr
}

func (f *fakeCommunityReadUseCase) ListFollowedCommunities(ctx context.Context, input communityusecase.ListFollowedCommunitiesInput) (communityusecase.ListFollowedCommunitiesResult, error) {
	f.listFollowedCalled = true
	f.listFollowedInput = input
	return f.listFollowedResult, f.listFollowedErr
}

func (f *fakeCommunityReadUseCase) GetCommunityManageContext(ctx context.Context, input communityusecase.GetCommunityManageContextInput) (communityusecase.GetCommunityManageContextResult, error) {
	f.manageCalled = true
	f.manageInput = input
	return f.manageResult, f.manageErr
}

func (f *fakeCommunityReadUseCase) ListCommunityMembers(ctx context.Context, input communityusecase.ListCommunityMembersInput) (communityusecase.ListCommunityMembersResult, error) {
	f.listMembersCalled = true
	f.listMembersInput = input
	return f.listMembersResult, f.listMembersErr
}

func (f *fakeCommunityReadUseCase) ListCommunityManagePosts(ctx context.Context, input communityusecase.ListCommunityManagePostsInput) (communityusecase.ListCommunityManagePostsResult, error) {
	f.listManagePostsCalled = true
	f.listManagePostsInput = input
	return f.listManagePostsResult, f.listManagePostsErr
}

func (f *fakeCommunityReadUseCase) ListCommunityManageComments(ctx context.Context, input communityusecase.ListCommunityManageCommentsInput) (communityusecase.ListCommunityManageCommentsResult, error) {
	f.listManageCommentsCalled = true
	f.listManageCommentsInput = input
	return f.listManageCommentsResult, f.listManageCommentsErr
}

func (f *fakeCommunityReadUseCase) ListCommunityManageReports(ctx context.Context, input communityusecase.ListCommunityManageReportsInput) (communityusecase.ListCommunityManageReportsResult, error) {
	f.listManageReportsCalled = true
	f.listManageReportsInput = input
	return f.listManageReportsResult, f.listManageReportsErr
}

func (f *fakeCommunityReadUseCase) GetCommunityManageSettings(ctx context.Context, input communityusecase.GetCommunityManageSettingsInput) (communityusecase.GetCommunityManageSettingsResult, error) {
	f.getSettingsCalled = true
	f.getSettingsInput = input
	return f.getSettingsResult, f.getSettingsErr
}

func (f *fakeCommunityReadUseCase) UpdateCommunityManageSettings(ctx context.Context, input communityusecase.UpdateCommunityManageSettingsInput) (communityusecase.UpdateCommunityManageSettingsResult, error) {
	f.updateSettingsCalled = true
	f.updateSettingsInput = input
	return f.updateSettingsResult, f.updateSettingsErr
}

func (f *fakeCommunityReadUseCase) ListCommunityRules(ctx context.Context, input communityusecase.ListCommunityRulesInput) (communityusecase.ListCommunityRulesResult, error) {
	f.listRulesCalled = true
	f.listRulesInput = input
	return f.listRulesResult, f.listRulesErr
}

func (f *fakeCommunityReadUseCase) CreateCommunityRule(ctx context.Context, input communityusecase.CreateCommunityRuleInput) (communityusecase.CreateCommunityRuleResult, error) {
	f.createRuleCalled = true
	f.createRuleInput = input
	return f.createRuleResult, f.createRuleErr
}

func (f *fakeCommunityReadUseCase) UpdateCommunityRule(ctx context.Context, input communityusecase.UpdateCommunityRuleInput) (communityusecase.UpdateCommunityRuleResult, error) {
	f.updateRuleCalled = true
	f.updateRuleInput = input
	return f.updateRuleResult, f.updateRuleErr
}

func (f *fakeCommunityReadUseCase) DeleteCommunityRule(ctx context.Context, input communityusecase.DeleteCommunityRuleInput) (communityusecase.DeleteCommunityRuleResult, error) {
	f.deleteRuleCalled = true
	f.deleteRuleInput = input
	return f.deleteRuleResult, f.deleteRuleErr
}

func (f *fakeCommunityReadUseCase) AddCommunityModerator(ctx context.Context, input communityusecase.AddCommunityModeratorInput) (communityusecase.CommunityMemberMutationResult, error) {
	f.addModeratorCalled = true
	f.addModeratorInput = input
	return f.addModeratorResult, f.addModeratorErr
}

func (f *fakeCommunityReadUseCase) RemoveCommunityModerator(ctx context.Context, input communityusecase.RemoveCommunityModeratorInput) (communityusecase.CommunityMemberMutationResult, error) {
	f.removeModeratorCalled = true
	f.removeModeratorInput = input
	return f.removeModeratorResult, f.removeModeratorErr
}

func (f *fakeCommunityReadUseCase) CreateCommunityOwnerTransfer(ctx context.Context, input communityusecase.CreateCommunityOwnerTransferInput) (communityusecase.CommunityOwnerTransferResult, error) {
	f.createTransferCalled = true
	f.createTransferInput = input
	return f.createTransferResult, f.createTransferErr
}

func (f *fakeCommunityReadUseCase) AcceptCommunityOwnerTransfer(ctx context.Context, input communityusecase.AcceptCommunityOwnerTransferInput) (communityusecase.CommunityOwnerTransferResult, error) {
	f.acceptTransferCalled = true
	f.acceptTransferInput = input
	return f.acceptTransferResult, f.acceptTransferErr
}

type fakeCommunityApplicationUseCase struct {
	submitCalled  bool
	listCalled    bool
	getCalled     bool
	approveCalled bool
	rejectCalled  bool
	submitInput   communityusecase.SubmitCommunityApplicationInput
	listInput     communityusecase.ListCommunityApplicationsInput
	getInput      communityusecase.GetCommunityApplicationInput
	approveInput  communityusecase.ReviewCommunityApplicationInput
	rejectInput   communityusecase.ReviewCommunityApplicationInput
	submitResult  communityusecase.SubmitCommunityApplicationResult
	listResult    communityusecase.ListCommunityApplicationsResult
	getResult     communityusecase.GetCommunityApplicationResult
	approveResult communityusecase.ApproveCommunityApplicationResult
	rejectResult  communityusecase.RejectCommunityApplicationResult
	submitErr     error
	listErr       error
	getErr        error
	approveErr    error
	rejectErr     error
}

func (f *fakeCommunityApplicationUseCase) SubmitCommunityApplication(ctx context.Context, input communityusecase.SubmitCommunityApplicationInput) (communityusecase.SubmitCommunityApplicationResult, error) {
	f.submitCalled = true
	f.submitInput = input
	return f.submitResult, f.submitErr
}

func (f *fakeCommunityApplicationUseCase) ListCommunityApplications(ctx context.Context, input communityusecase.ListCommunityApplicationsInput) (communityusecase.ListCommunityApplicationsResult, error) {
	f.listCalled = true
	f.listInput = input
	return f.listResult, f.listErr
}

func (f *fakeCommunityApplicationUseCase) GetCommunityApplication(ctx context.Context, input communityusecase.GetCommunityApplicationInput) (communityusecase.GetCommunityApplicationResult, error) {
	f.getCalled = true
	f.getInput = input
	return f.getResult, f.getErr
}

func (f *fakeCommunityApplicationUseCase) ApproveCommunityApplication(ctx context.Context, input communityusecase.ReviewCommunityApplicationInput) (communityusecase.ApproveCommunityApplicationResult, error) {
	f.approveCalled = true
	f.approveInput = input
	return f.approveResult, f.approveErr
}

func (f *fakeCommunityApplicationUseCase) RejectCommunityApplication(ctx context.Context, input communityusecase.ReviewCommunityApplicationInput) (communityusecase.RejectCommunityApplicationResult, error) {
	f.rejectCalled = true
	f.rejectInput = input
	return f.rejectResult, f.rejectErr
}

type fakeAccessTokenParser struct {
	claims *authtoken.AccessTokenClaims
	err    error
}

func (f *fakeAccessTokenParser) ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error) {
	return f.claims, f.err
}

func newCommunityTestRouter(communities CommunityReadUseCase, applications CommunityApplicationUseCase, parser authhttp.AccessTokenParser) *gin.Engine {
	router := gin.New()
	router.Use(httpserver.ErrorMiddleware())

	publicRead := router.Group("/api/v1")
	publicRead.Use(authhttp.OptionalAuth(parser))
	RegisterReadRoutes(publicRead, NewHandler(communities, applications))

	protected := router.Group("/api/v1")
	protected.Use(authhttp.RequireAuth(parser))
	RegisterApplicationRoutes(protected, NewHandler(communities, applications))
	RegisterFollowRoutes(protected, NewHandler(communities, applications))
	RegisterManageRoutes(protected, NewHandler(communities, applications))

	return router
}

func validParser() *fakeAccessTokenParser {
	return validParserWithUserID(userdomain.NewGeneratedUserID())
}

func validParserWithUserID(userID userdomain.UserID) *fakeAccessTokenParser {
	return &fakeAccessTokenParser{
		claims: &authtoken.AccessTokenClaims{
			UserID: userID,
		},
	}
}

func newCommunityResult(slug string, now time.Time) communityusecase.Community {
	return communityusecase.Community{
		ID:          userdomain.NewGeneratedUserID().String(),
		Slug:        slug,
		Name:        slug,
		Description: "test community",
		Kind:        "system",
		Status:      "active",
		Visibility:  "public",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func newApplicationResult(slug string, status string, now time.Time) communityusecase.CommunityApplication {
	return communityusecase.CommunityApplication{
		ID:            "8f92e975-5323-4a58-bac1-1336b668183c",
		ApplicantID:   userdomain.NewGeneratedUserID().String(),
		RequestedSlug: slug,
		RequestedName: slug,
		Reason:        "Need a community",
		Status:        status,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func assertCommunityErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, wantCode apperr.Code) {
	t.Helper()

	var response httpserver.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != string(wantCode) {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
}

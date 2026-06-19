package moderationusecase

import (
	"context"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/textlimit"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

const (
	CommunityFlairKindPost = "post"
	CommunityFlairKindUser = "user"

	MaxScheduledPostTitleRunes = 160
	MaxScheduledPostBodyRunes  = 20000
	MaxRepeatRuleRunes         = 160
	MaxGuideTitleRunes         = 160
	MaxGuideBodyRunes          = 20000
	MaxFlairColorRunes         = 32
)

type CommunityFlairInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	Kind          string
}

type WriteCommunityFlairInput struct {
	ActorID          userdomain.UserID
	CommunitySlug    string
	Kind             string
	ID               string
	Title            string
	Color            string
	IsUserSelectable bool
	IsEnabled        *bool
	Position         int
}

type DeleteCommunityFlairInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	Kind          string
	ID            string
}

type ReorderCommunityFlairsInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	Kind          string
	IDs           []string
}

type CommunityFlairRecordInput struct {
	ID               string
	CommunityID      communitydomain.CommunityID
	ActorID          userdomain.UserID
	Kind             string
	Title            string
	Color            string
	IsUserSelectable bool
	IsEnabled        bool
	Position         int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type DeleteCommunityFlairRecordInput struct {
	ID          string
	CommunityID communitydomain.CommunityID
	ActorID     userdomain.UserID
	Kind        string
	DeletedAt   time.Time
}

type ReorderCommunityFlairsRecordInput struct {
	CommunityID communitydomain.CommunityID
	ActorID     userdomain.UserID
	Kind        string
	IDs         []string
	UpdatedAt   time.Time
}

type ListCommunityFlairsResult struct {
	Items []CommunityFlair
}

type CommunityFlairResult struct {
	Item CommunityFlair
}

type CommunityFlair struct {
	ID               string
	CommunityID      string
	Kind             string
	Title            string
	Color            string
	IsUserSelectable bool
	IsEnabled        bool
	Position         int
	CreatedBy        string
	UpdatedBy        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ListScheduledPostsInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	Limit         int
	Offset        int
}

type WriteScheduledPostInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	ID            string
	Title         string
	Body          string
	ScheduledAt   time.Time
	RepeatRule    string
	Status        string
}

type DeleteScheduledPostInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	ID            string
}

type WriteScheduledPostRecordInput struct {
	ID          string
	CommunityID communitydomain.CommunityID
	ActorID     userdomain.UserID
	Title       string
	Body        string
	ScheduledAt time.Time
	RepeatRule  string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type DeleteScheduledPostRecordInput struct {
	ID          string
	CommunityID communitydomain.CommunityID
	ActorID     userdomain.UserID
	DeletedAt   time.Time
}

type ListScheduledPostsResult struct {
	Items      []CommunityScheduledPost
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type CommunityScheduledPostResult struct {
	Item CommunityScheduledPost
}

type CommunityScheduledPost struct {
	ID          string
	CommunityID string
	CreatedBy   string
	UpdatedBy   string
	Title       string
	Body        string
	ScheduledAt time.Time
	RepeatRule  string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ListGuidesInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	Limit         int
	Offset        int
}

type WriteGuideInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	ID            string
	Title         string
	Body          string
	Position      int
	Visibility    string
}

type DeleteGuideInput struct {
	ActorID       userdomain.UserID
	CommunitySlug string
	ID            string
}

type WriteGuideRecordInput struct {
	ID          string
	CommunityID communitydomain.CommunityID
	ActorID     userdomain.UserID
	Title       string
	Body        string
	Position    int
	Visibility  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type DeleteGuideRecordInput struct {
	ID          string
	CommunityID communitydomain.CommunityID
	ActorID     userdomain.UserID
	DeletedAt   time.Time
}

type ListGuidesResult struct {
	Items      []CommunityGuide
	Limit      int
	Offset     int
	NextOffset int
	HasMore    bool
}

type CommunityGuideResult struct {
	Item CommunityGuide
}

type CommunityGuide struct {
	ID          string
	CommunityID string
	CreatedBy   string
	UpdatedBy   string
	Title       string
	Body        string
	Position    int
	Visibility  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (uc *ToolsUseCase) ListCommunityFlairs(ctx context.Context, input CommunityFlairInput) (ListCommunityFlairsResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ListCommunityFlairsResult{}, err
	}
	kind, err := normalizeFlairKind(input.Kind)
	if err != nil {
		return ListCommunityFlairsResult{}, err
	}
	items, err := uc.tools.ListCommunityFlairs(ctx, communityID, kind)
	if err != nil {
		return ListCommunityFlairsResult{}, err
	}
	return ListCommunityFlairsResult{Items: items}, nil
}

func (uc *ToolsUseCase) CreateCommunityFlair(ctx context.Context, input WriteCommunityFlairInput) (CommunityFlairResult, error) {
	communityID, record, err := uc.communityFlairRecord(ctx, input, uuid.NewString())
	if err != nil {
		return CommunityFlairResult{}, err
	}
	record.CommunityID = communityID
	item, err := uc.tools.CreateCommunityFlair(ctx, record)
	if err != nil {
		return CommunityFlairResult{}, err
	}
	return CommunityFlairResult{Item: item}, nil
}

func (uc *ToolsUseCase) UpdateCommunityFlair(ctx context.Context, input WriteCommunityFlairInput) (CommunityFlairResult, error) {
	id := normalizeOptionalUUID(input.ID)
	if id == "" {
		return CommunityFlairResult{}, apperr.New(apperr.CodeInvalidArgument, "flair id is invalid")
	}
	communityID, record, err := uc.communityFlairRecord(ctx, input, id)
	if err != nil {
		return CommunityFlairResult{}, err
	}
	record.CommunityID = communityID
	item, err := uc.tools.UpdateCommunityFlair(ctx, record)
	if err != nil {
		return CommunityFlairResult{}, err
	}
	return CommunityFlairResult{Item: item}, nil
}

func (uc *ToolsUseCase) DeleteCommunityFlair(ctx context.Context, input DeleteCommunityFlairInput) error {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return err
	}
	kind, err := normalizeFlairKind(input.Kind)
	if err != nil {
		return err
	}
	id := normalizeOptionalUUID(input.ID)
	if id == "" {
		return apperr.New(apperr.CodeInvalidArgument, "flair id is invalid")
	}
	return uc.tools.DeleteCommunityFlair(ctx, DeleteCommunityFlairRecordInput{
		ID:          id,
		CommunityID: communityID,
		ActorID:     input.ActorID,
		Kind:        kind,
		DeletedAt:   uc.now().UTC(),
	})
}

func (uc *ToolsUseCase) ReorderCommunityFlairs(ctx context.Context, input ReorderCommunityFlairsInput) (ListCommunityFlairsResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ListCommunityFlairsResult{}, err
	}
	kind, err := normalizeFlairKind(input.Kind)
	if err != nil {
		return ListCommunityFlairsResult{}, err
	}
	ids := make([]string, 0, len(input.IDs))
	seen := map[string]struct{}{}
	for _, raw := range input.IDs {
		id := normalizeOptionalUUID(raw)
		if id == "" {
			return ListCommunityFlairsResult{}, apperr.New(apperr.CodeInvalidArgument, "flair id is invalid")
		}
		if _, ok := seen[id]; ok {
			return ListCommunityFlairsResult{}, apperr.New(apperr.CodeInvalidArgument, "flair ids must be unique")
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return ListCommunityFlairsResult{}, apperr.New(apperr.CodeInvalidArgument, "flair ids are required")
	}
	items, err := uc.tools.ReorderCommunityFlairs(ctx, ReorderCommunityFlairsRecordInput{
		CommunityID: communityID,
		ActorID:     input.ActorID,
		Kind:        kind,
		IDs:         ids,
		UpdatedAt:   uc.now().UTC(),
	})
	if err != nil {
		return ListCommunityFlairsResult{}, err
	}
	return ListCommunityFlairsResult{Items: items}, nil
}

func (uc *ToolsUseCase) communityFlairRecord(ctx context.Context, input WriteCommunityFlairInput, id string) (communitydomain.CommunityID, CommunityFlairRecordInput, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return zeroCommunityID(), CommunityFlairRecordInput{}, err
	}
	kind, err := normalizeFlairKind(input.Kind)
	if err != nil {
		return zeroCommunityID(), CommunityFlairRecordInput{}, err
	}
	title, err := textlimit.TrimmedRequiredMaxRunes(input.Title, "flair title", MaxToolTitleRunes)
	if err != nil {
		return zeroCommunityID(), CommunityFlairRecordInput{}, err
	}
	color, err := textlimit.TrimmedOptionalMaxRunes(input.Color, "flair color", MaxFlairColorRunes)
	if err != nil {
		return zeroCommunityID(), CommunityFlairRecordInput{}, err
	}
	if input.Position < 0 {
		return zeroCommunityID(), CommunityFlairRecordInput{}, apperr.New(apperr.CodeInvalidArgument, "flair position is invalid")
	}
	isEnabled := true
	if input.IsEnabled != nil {
		isEnabled = *input.IsEnabled
	}
	now := uc.now().UTC()
	return communityID, CommunityFlairRecordInput{
		ID:               id,
		CommunityID:      communityID,
		ActorID:          input.ActorID,
		Kind:             kind,
		Title:            title,
		Color:            color,
		IsUserSelectable: input.IsUserSelectable,
		IsEnabled:        isEnabled,
		Position:         input.Position,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func (uc *ToolsUseCase) ListScheduledPosts(ctx context.Context, input ListScheduledPostsInput) (ListScheduledPostsResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ListScheduledPostsResult{}, err
	}
	limit, offset, err := normalizeModToolsPagination(input.Limit, input.Offset)
	if err != nil {
		return ListScheduledPostsResult{}, err
	}
	items, err := uc.tools.ListScheduledPosts(ctx, communityID, limit+1, offset)
	if err != nil {
		return ListScheduledPostsResult{}, err
	}
	items, hasMore := trimToolsPage(items, limit)
	return ListScheduledPostsResult{Items: items, Limit: limit, Offset: offset, NextOffset: offset + len(items), HasMore: hasMore}, nil
}

func (uc *ToolsUseCase) CreateScheduledPost(ctx context.Context, input WriteScheduledPostInput) (CommunityScheduledPostResult, error) {
	communityID, record, err := uc.scheduledPostRecord(ctx, input, uuid.NewString())
	if err != nil {
		return CommunityScheduledPostResult{}, err
	}
	record.CommunityID = communityID
	item, err := uc.tools.CreateScheduledPost(ctx, record)
	if err != nil {
		return CommunityScheduledPostResult{}, err
	}
	return CommunityScheduledPostResult{Item: item}, nil
}

func (uc *ToolsUseCase) UpdateScheduledPost(ctx context.Context, input WriteScheduledPostInput) (CommunityScheduledPostResult, error) {
	id := normalizeOptionalUUID(input.ID)
	if id == "" {
		return CommunityScheduledPostResult{}, apperr.New(apperr.CodeInvalidArgument, "scheduled post id is invalid")
	}
	communityID, record, err := uc.scheduledPostRecord(ctx, input, id)
	if err != nil {
		return CommunityScheduledPostResult{}, err
	}
	record.CommunityID = communityID
	item, err := uc.tools.UpdateScheduledPost(ctx, record)
	if err != nil {
		return CommunityScheduledPostResult{}, err
	}
	return CommunityScheduledPostResult{Item: item}, nil
}

func (uc *ToolsUseCase) DeleteScheduledPost(ctx context.Context, input DeleteScheduledPostInput) error {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return err
	}
	id := normalizeOptionalUUID(input.ID)
	if id == "" {
		return apperr.New(apperr.CodeInvalidArgument, "scheduled post id is invalid")
	}
	return uc.tools.DeleteScheduledPost(ctx, DeleteScheduledPostRecordInput{
		ID:          id,
		CommunityID: communityID,
		ActorID:     input.ActorID,
		DeletedAt:   uc.now().UTC(),
	})
}

func (uc *ToolsUseCase) scheduledPostRecord(ctx context.Context, input WriteScheduledPostInput, id string) (communitydomain.CommunityID, WriteScheduledPostRecordInput, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return zeroCommunityID(), WriteScheduledPostRecordInput{}, err
	}
	title, err := textlimit.TrimmedRequiredMaxRunes(input.Title, "scheduled post title", MaxScheduledPostTitleRunes)
	if err != nil {
		return zeroCommunityID(), WriteScheduledPostRecordInput{}, err
	}
	body, err := textlimit.TrimmedOptionalMaxRunes(input.Body, "scheduled post body", MaxScheduledPostBodyRunes)
	if err != nil {
		return zeroCommunityID(), WriteScheduledPostRecordInput{}, err
	}
	repeatRule, err := textlimit.TrimmedOptionalMaxRunes(input.RepeatRule, "repeat rule", MaxRepeatRuleRunes)
	if err != nil {
		return zeroCommunityID(), WriteScheduledPostRecordInput{}, err
	}
	status, err := normalizeScheduledPostStatus(input.Status)
	if err != nil {
		return zeroCommunityID(), WriteScheduledPostRecordInput{}, err
	}
	if input.ScheduledAt.IsZero() {
		return zeroCommunityID(), WriteScheduledPostRecordInput{}, apperr.New(apperr.CodeInvalidArgument, "scheduled_at is required")
	}
	now := uc.now().UTC()
	return communityID, WriteScheduledPostRecordInput{
		ID:          id,
		CommunityID: communityID,
		ActorID:     input.ActorID,
		Title:       title,
		Body:        body,
		ScheduledAt: input.ScheduledAt.UTC(),
		RepeatRule:  repeatRule,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (uc *ToolsUseCase) ListGuides(ctx context.Context, input ListGuidesInput) (ListGuidesResult, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return ListGuidesResult{}, err
	}
	limit, offset, err := normalizeModToolsPagination(input.Limit, input.Offset)
	if err != nil {
		return ListGuidesResult{}, err
	}
	items, err := uc.tools.ListGuides(ctx, communityID, limit+1, offset)
	if err != nil {
		return ListGuidesResult{}, err
	}
	items, hasMore := trimToolsPage(items, limit)
	return ListGuidesResult{Items: items, Limit: limit, Offset: offset, NextOffset: offset + len(items), HasMore: hasMore}, nil
}

func (uc *ToolsUseCase) CreateGuide(ctx context.Context, input WriteGuideInput) (CommunityGuideResult, error) {
	communityID, record, err := uc.guideRecord(ctx, input, uuid.NewString())
	if err != nil {
		return CommunityGuideResult{}, err
	}
	record.CommunityID = communityID
	item, err := uc.tools.CreateGuide(ctx, record)
	if err != nil {
		return CommunityGuideResult{}, err
	}
	return CommunityGuideResult{Item: item}, nil
}

func (uc *ToolsUseCase) UpdateGuide(ctx context.Context, input WriteGuideInput) (CommunityGuideResult, error) {
	id := normalizeOptionalUUID(input.ID)
	if id == "" {
		return CommunityGuideResult{}, apperr.New(apperr.CodeInvalidArgument, "guide id is invalid")
	}
	communityID, record, err := uc.guideRecord(ctx, input, id)
	if err != nil {
		return CommunityGuideResult{}, err
	}
	record.CommunityID = communityID
	item, err := uc.tools.UpdateGuide(ctx, record)
	if err != nil {
		return CommunityGuideResult{}, err
	}
	return CommunityGuideResult{Item: item}, nil
}

func (uc *ToolsUseCase) DeleteGuide(ctx context.Context, input DeleteGuideInput) error {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return err
	}
	id := normalizeOptionalUUID(input.ID)
	if id == "" {
		return apperr.New(apperr.CodeInvalidArgument, "guide id is invalid")
	}
	return uc.tools.DeleteGuide(ctx, DeleteGuideRecordInput{
		ID:          id,
		CommunityID: communityID,
		ActorID:     input.ActorID,
		DeletedAt:   uc.now().UTC(),
	})
}

func (uc *ToolsUseCase) guideRecord(ctx context.Context, input WriteGuideInput, id string) (communitydomain.CommunityID, WriteGuideRecordInput, error) {
	communityID, err := uc.ensureCommunityModerator(ctx, input.ActorID, input.CommunitySlug)
	if err != nil {
		return zeroCommunityID(), WriteGuideRecordInput{}, err
	}
	title, err := textlimit.TrimmedRequiredMaxRunes(input.Title, "guide title", MaxGuideTitleRunes)
	if err != nil {
		return zeroCommunityID(), WriteGuideRecordInput{}, err
	}
	body, err := textlimit.TrimmedOptionalMaxRunes(input.Body, "guide body", MaxGuideBodyRunes)
	if err != nil {
		return zeroCommunityID(), WriteGuideRecordInput{}, err
	}
	if input.Position < 0 {
		return zeroCommunityID(), WriteGuideRecordInput{}, apperr.New(apperr.CodeInvalidArgument, "guide position is invalid")
	}
	visibility, err := normalizeGuideVisibility(input.Visibility)
	if err != nil {
		return zeroCommunityID(), WriteGuideRecordInput{}, err
	}
	now := uc.now().UTC()
	return communityID, WriteGuideRecordInput{
		ID:          id,
		CommunityID: communityID,
		ActorID:     input.ActorID,
		Title:       title,
		Body:        body,
		Position:    input.Position,
		Visibility:  visibility,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func normalizeFlairKind(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case CommunityFlairKindPost, CommunityFlairKindUser:
		return value, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "flair kind is invalid")
	}
}

func zeroCommunityID() communitydomain.CommunityID {
	var id communitydomain.CommunityID
	return id
}

func normalizeScheduledPostStatus(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		value = "scheduled"
	}
	switch value {
	case "scheduled", "paused", "published", "cancelled":
		return value, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "scheduled post status is invalid")
	}
}

func normalizeGuideVisibility(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		value = "public"
	}
	switch value {
	case "public", "members", "mods":
		return value, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "guide visibility is invalid")
	}
}

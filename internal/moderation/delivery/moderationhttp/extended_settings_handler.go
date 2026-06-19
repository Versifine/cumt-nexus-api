package moderationhttp

import (
	"net/http"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationusecase"
	"github.com/gin-gonic/gin"
)

type communityFlairListResponse struct {
	Items []communityFlairView `json:"items"`
}

type communityFlairResponse struct {
	Item communityFlairView `json:"item"`
}

type communityFlairView struct {
	ID               string    `json:"id"`
	CommunityID      string    `json:"community_id"`
	Kind             string    `json:"kind"`
	Title            string    `json:"title"`
	Color            string    `json:"color"`
	IsUserSelectable bool      `json:"is_user_selectable"`
	IsEnabled        bool      `json:"is_enabled"`
	Position         int       `json:"position"`
	CreatedBy        string    `json:"created_by"`
	UpdatedBy        string    `json:"updated_by"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type communityFlairRequest struct {
	Title            string `json:"title" binding:"required"`
	Color            string `json:"color"`
	IsUserSelectable bool   `json:"is_user_selectable"`
	IsEnabled        *bool  `json:"is_enabled"`
	Position         int    `json:"position"`
}

type reorderCommunityFlairsRequest struct {
	Kind string   `json:"kind" binding:"required"`
	IDs  []string `json:"ids" binding:"required"`
}

type scheduledPostListResponse struct {
	Items      []scheduledPostView `json:"items"`
	Limit      int                 `json:"limit"`
	Offset     int                 `json:"offset"`
	NextOffset int                 `json:"next_offset"`
	HasMore    bool                `json:"has_more"`
}

type scheduledPostResponse struct {
	Item scheduledPostView `json:"item"`
}

type scheduledPostView struct {
	ID          string    `json:"id"`
	CommunityID string    `json:"community_id"`
	CreatedBy   string    `json:"created_by"`
	UpdatedBy   string    `json:"updated_by"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	ScheduledAt time.Time `json:"scheduled_at"`
	RepeatRule  string    `json:"repeat_rule"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type scheduledPostRequest struct {
	Title       string    `json:"title" binding:"required"`
	Body        string    `json:"body"`
	ScheduledAt time.Time `json:"scheduled_at" binding:"required"`
	RepeatRule  string    `json:"repeat_rule"`
	Status      string    `json:"status"`
}

type guideListResponse struct {
	Items      []guideView `json:"items"`
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
	NextOffset int         `json:"next_offset"`
	HasMore    bool        `json:"has_more"`
}

type guideResponse struct {
	Item guideView `json:"item"`
}

type guideView struct {
	ID          string    `json:"id"`
	CommunityID string    `json:"community_id"`
	CreatedBy   string    `json:"created_by"`
	UpdatedBy   string    `json:"updated_by"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	Position    int       `json:"position"`
	Visibility  string    `json:"visibility"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type guideRequest struct {
	Title      string `json:"title" binding:"required"`
	Body       string `json:"body"`
	Position   int    `json:"position"`
	Visibility string `json:"visibility"`
}

func (h *Handler) ListCommunityPostFlairs(c *gin.Context) {
	h.listCommunityFlairs(c, moderationusecase.CommunityFlairKindPost)
}

func (h *Handler) CreateCommunityPostFlair(c *gin.Context) {
	h.createCommunityFlair(c, moderationusecase.CommunityFlairKindPost)
}

func (h *Handler) UpdateCommunityPostFlair(c *gin.Context) {
	h.updateCommunityFlair(c, moderationusecase.CommunityFlairKindPost)
}

func (h *Handler) DeleteCommunityPostFlair(c *gin.Context) {
	h.deleteCommunityFlair(c, moderationusecase.CommunityFlairKindPost)
}

func (h *Handler) ListCommunityUserFlairs(c *gin.Context) {
	h.listCommunityFlairs(c, moderationusecase.CommunityFlairKindUser)
}

func (h *Handler) CreateCommunityUserFlair(c *gin.Context) {
	h.createCommunityFlair(c, moderationusecase.CommunityFlairKindUser)
}

func (h *Handler) UpdateCommunityUserFlair(c *gin.Context) {
	h.updateCommunityFlair(c, moderationusecase.CommunityFlairKindUser)
}

func (h *Handler) DeleteCommunityUserFlair(c *gin.Context) {
	h.deleteCommunityFlair(c, moderationusecase.CommunityFlairKindUser)
}

func (h *Handler) listCommunityFlairs(c *gin.Context, kind string) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	result, err := h.tools.ListCommunityFlairs(c.Request.Context(), moderationusecase.CommunityFlairInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		Kind:          kind,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := communityFlairListResponse{Items: make([]communityFlairView, 0, len(result.Items))}
	for _, item := range result.Items {
		response.Items = append(response.Items, toCommunityFlairView(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) createCommunityFlair(c *gin.Context, kind string) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req communityFlairRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid community flair request"))
		c.Abort()
		return
	}
	result, err := h.tools.CreateCommunityFlair(c.Request.Context(), moderationusecase.WriteCommunityFlairInput{
		ActorID:          userID,
		CommunitySlug:    c.Param("slug"),
		Kind:             kind,
		Title:            req.Title,
		Color:            req.Color,
		IsUserSelectable: req.IsUserSelectable,
		IsEnabled:        req.IsEnabled,
		Position:         req.Position,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusCreated, communityFlairResponse{Item: toCommunityFlairView(result.Item)})
}

func (h *Handler) updateCommunityFlair(c *gin.Context, kind string) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req communityFlairRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid community flair request"))
		c.Abort()
		return
	}
	result, err := h.tools.UpdateCommunityFlair(c.Request.Context(), moderationusecase.WriteCommunityFlairInput{
		ActorID:          userID,
		CommunitySlug:    c.Param("slug"),
		Kind:             kind,
		ID:               c.Param("flair_id"),
		Title:            req.Title,
		Color:            req.Color,
		IsUserSelectable: req.IsUserSelectable,
		IsEnabled:        req.IsEnabled,
		Position:         req.Position,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, communityFlairResponse{Item: toCommunityFlairView(result.Item)})
}

func (h *Handler) deleteCommunityFlair(c *gin.Context, kind string) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	err := h.tools.DeleteCommunityFlair(c.Request.Context(), moderationusecase.DeleteCommunityFlairInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		Kind:          kind,
		ID:            c.Param("flair_id"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ReorderCommunityFlairs(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req reorderCommunityFlairsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid community flair reorder request"))
		c.Abort()
		return
	}
	result, err := h.tools.ReorderCommunityFlairs(c.Request.Context(), moderationusecase.ReorderCommunityFlairsInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		Kind:          req.Kind,
		IDs:           req.IDs,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := communityFlairListResponse{Items: make([]communityFlairView, 0, len(result.Items))}
	for _, item := range result.Items {
		response.Items = append(response.Items, toCommunityFlairView(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) ListScheduledPosts(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	limit, err := parseOptionalIntQuery(c, "limit")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	offset, err := parseOptionalIntQuery(c, "offset")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	result, err := h.tools.ListScheduledPosts(c.Request.Context(), moderationusecase.ListScheduledPostsInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := scheduledPostListResponse{Items: make([]scheduledPostView, 0, len(result.Items)), Limit: result.Limit, Offset: result.Offset, NextOffset: result.NextOffset, HasMore: result.HasMore}
	for _, item := range result.Items {
		response.Items = append(response.Items, toScheduledPostView(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) CreateScheduledPost(c *gin.Context) {
	h.writeScheduledPost(c, "")
}

func (h *Handler) UpdateScheduledPost(c *gin.Context) {
	h.writeScheduledPost(c, c.Param("scheduled_post_id"))
}

func (h *Handler) writeScheduledPost(c *gin.Context, id string) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req scheduledPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid scheduled post request"))
		c.Abort()
		return
	}
	input := moderationusecase.WriteScheduledPostInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		ID:            id,
		Title:         req.Title,
		Body:          req.Body,
		ScheduledAt:   req.ScheduledAt,
		RepeatRule:    req.RepeatRule,
		Status:        req.Status,
	}
	var result moderationusecase.CommunityScheduledPostResult
	var err error
	status := http.StatusCreated
	if id == "" {
		result, err = h.tools.CreateScheduledPost(c.Request.Context(), input)
	} else {
		status = http.StatusOK
		result, err = h.tools.UpdateScheduledPost(c.Request.Context(), input)
	}
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(status, scheduledPostResponse{Item: toScheduledPostView(result.Item)})
}

func (h *Handler) DeleteScheduledPost(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	err := h.tools.DeleteScheduledPost(c.Request.Context(), moderationusecase.DeleteScheduledPostInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		ID:            c.Param("scheduled_post_id"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListGuides(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	limit, err := parseOptionalIntQuery(c, "limit")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	offset, err := parseOptionalIntQuery(c, "offset")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	result, err := h.tools.ListGuides(c.Request.Context(), moderationusecase.ListGuidesInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	response := guideListResponse{Items: make([]guideView, 0, len(result.Items)), Limit: result.Limit, Offset: result.Offset, NextOffset: result.NextOffset, HasMore: result.HasMore}
	for _, item := range result.Items {
		response.Items = append(response.Items, toGuideView(item))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) CreateGuide(c *gin.Context) {
	h.writeGuide(c, "")
}

func (h *Handler) UpdateGuide(c *gin.Context) {
	h.writeGuide(c, c.Param("guide_id"))
}

func (h *Handler) writeGuide(c *gin.Context, id string) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	var req guideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid guide request"))
		c.Abort()
		return
	}
	input := moderationusecase.WriteGuideInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		ID:            id,
		Title:         req.Title,
		Body:          req.Body,
		Position:      req.Position,
		Visibility:    req.Visibility,
	}
	var result moderationusecase.CommunityGuideResult
	var err error
	status := http.StatusCreated
	if id == "" {
		result, err = h.tools.CreateGuide(c.Request.Context(), input)
	} else {
		status = http.StatusOK
		result, err = h.tools.UpdateGuide(c.Request.Context(), input)
	}
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(status, guideResponse{Item: toGuideView(result.Item)})
}

func (h *Handler) DeleteGuide(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		return
	}
	err := h.tools.DeleteGuide(c.Request.Context(), moderationusecase.DeleteGuideInput{
		ActorID:       userID,
		CommunitySlug: c.Param("slug"),
		ID:            c.Param("guide_id"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.Status(http.StatusNoContent)
}

func toCommunityFlairView(item moderationusecase.CommunityFlair) communityFlairView {
	return communityFlairView{
		ID:               item.ID,
		CommunityID:      item.CommunityID,
		Kind:             item.Kind,
		Title:            item.Title,
		Color:            item.Color,
		IsUserSelectable: item.IsUserSelectable,
		IsEnabled:        item.IsEnabled,
		Position:         item.Position,
		CreatedBy:        item.CreatedBy,
		UpdatedBy:        item.UpdatedBy,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

func toScheduledPostView(item moderationusecase.CommunityScheduledPost) scheduledPostView {
	return scheduledPostView{
		ID:          item.ID,
		CommunityID: item.CommunityID,
		CreatedBy:   item.CreatedBy,
		UpdatedBy:   item.UpdatedBy,
		Title:       item.Title,
		Body:        item.Body,
		ScheduledAt: item.ScheduledAt,
		RepeatRule:  item.RepeatRule,
		Status:      item.Status,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func toGuideView(item moderationusecase.CommunityGuide) guideView {
	return guideView{
		ID:          item.ID,
		CommunityID: item.CommunityID,
		CreatedBy:   item.CreatedBy,
		UpdatedBy:   item.UpdatedBy,
		Title:       item.Title,
		Body:        item.Body,
		Position:    item.Position,
		Visibility:  item.Visibility,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

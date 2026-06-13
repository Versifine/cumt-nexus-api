package communityhttp

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/community/communityusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	communities  CommunityReadUseCase
	applications CommunityApplicationUseCase
}

type CommunityReadUseCase interface {
	ListCommunities(ctx context.Context, input communityusecase.ListCommunitiesInput) (communityusecase.ListCommunitiesResult, error)
	GetCommunityBySlug(ctx context.Context, input communityusecase.GetCommunityInput) (communityusecase.GetCommunityResult, error)
	FollowCommunity(ctx context.Context, input communityusecase.FollowCommunityInput) (communityusecase.FollowCommunityResult, error)
	DeleteCommunityFollow(ctx context.Context, input communityusecase.DeleteCommunityFollowInput) (communityusecase.DeleteCommunityFollowResult, error)
	ListFollowedCommunities(ctx context.Context, input communityusecase.ListFollowedCommunitiesInput) (communityusecase.ListFollowedCommunitiesResult, error)
	GetCommunityManageContext(ctx context.Context, input communityusecase.GetCommunityManageContextInput) (communityusecase.GetCommunityManageContextResult, error)
	ListCommunityMembers(ctx context.Context, input communityusecase.ListCommunityMembersInput) (communityusecase.ListCommunityMembersResult, error)
	ListCommunityManagePosts(ctx context.Context, input communityusecase.ListCommunityManagePostsInput) (communityusecase.ListCommunityManagePostsResult, error)
	ListCommunityManageComments(ctx context.Context, input communityusecase.ListCommunityManageCommentsInput) (communityusecase.ListCommunityManageCommentsResult, error)
	ListCommunityManageReports(ctx context.Context, input communityusecase.ListCommunityManageReportsInput) (communityusecase.ListCommunityManageReportsResult, error)
	GetCommunityManageSettings(ctx context.Context, input communityusecase.GetCommunityManageSettingsInput) (communityusecase.GetCommunityManageSettingsResult, error)
	UpdateCommunityManageSettings(ctx context.Context, input communityusecase.UpdateCommunityManageSettingsInput) (communityusecase.UpdateCommunityManageSettingsResult, error)
	ListCommunityRules(ctx context.Context, input communityusecase.ListCommunityRulesInput) (communityusecase.ListCommunityRulesResult, error)
	CreateCommunityRule(ctx context.Context, input communityusecase.CreateCommunityRuleInput) (communityusecase.CreateCommunityRuleResult, error)
	UpdateCommunityRule(ctx context.Context, input communityusecase.UpdateCommunityRuleInput) (communityusecase.UpdateCommunityRuleResult, error)
	DeleteCommunityRule(ctx context.Context, input communityusecase.DeleteCommunityRuleInput) (communityusecase.DeleteCommunityRuleResult, error)
}

type CommunityApplicationUseCase interface {
	SubmitCommunityApplication(ctx context.Context, input communityusecase.SubmitCommunityApplicationInput) (communityusecase.SubmitCommunityApplicationResult, error)
	ListCommunityApplications(ctx context.Context, input communityusecase.ListCommunityApplicationsInput) (communityusecase.ListCommunityApplicationsResult, error)
	GetCommunityApplication(ctx context.Context, input communityusecase.GetCommunityApplicationInput) (communityusecase.GetCommunityApplicationResult, error)
	ApproveCommunityApplication(ctx context.Context, input communityusecase.ReviewCommunityApplicationInput) (communityusecase.ApproveCommunityApplicationResult, error)
	RejectCommunityApplication(ctx context.Context, input communityusecase.ReviewCommunityApplicationInput) (communityusecase.RejectCommunityApplicationResult, error)
}

type listCommunitiesResponse struct {
	Communities []communityResponse `json:"communities"`
}

type getCommunityResponse struct {
	Community communityResponse `json:"community"`
}

type getCommunityManageContextResponse struct {
	Community communityResponse `json:"community"`
}

type getCommunityManageSettingsResponse struct {
	Community communityResponse         `json:"community"`
	Settings  communitySettingsResponse `json:"settings"`
}

type updateCommunityManageSettingsResponse struct {
	Community communityResponse         `json:"community"`
	Settings  communitySettingsResponse `json:"settings"`
}

type listFollowedCommunitiesResponse struct {
	Communities []communityResponse `json:"communities"`
	Limit       int                 `json:"limit"`
	Offset      int                 `json:"offset"`
}

type listCommunityMembersResponse struct {
	Community communityResponse         `json:"community"`
	Members   []communityMemberResponse `json:"members"`
	Limit     int                       `json:"limit"`
	Offset    int                       `json:"offset"`
}

type communityMemberResponse struct {
	User      communityMemberUserResponse `json:"user"`
	Role      string                      `json:"role"`
	Status    string                      `json:"status"`
	CreatedAt time.Time                   `json:"created_at"`
	UpdatedAt time.Time                   `json:"updated_at"`
}

type communityMemberUserResponse struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	AvatarURL   string   `json:"avatar_url"`
	Headline    string   `json:"headline"`
	Badges      []string `json:"badges"`
}

type listCommunityManagePostsResponse struct {
	Community communityResponse             `json:"community"`
	Posts     []communityManagePostResponse `json:"posts"`
	Status    string                        `json:"status"`
	Limit     int                           `json:"limit"`
	Offset    int                           `json:"offset"`
}

type communityManagePostResponse struct {
	ID          string    `json:"id"`
	CommunityID string    `json:"community_id"`
	AuthorID    string    `json:"author_id"`
	Title       string    `json:"title"`
	BodyExcerpt string    `json:"body_excerpt"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type listCommunityManageCommentsResponse struct {
	Community communityResponse                `json:"community"`
	Comments  []communityManageCommentResponse `json:"comments"`
	Status    string                           `json:"status"`
	Limit     int                              `json:"limit"`
	Offset    int                              `json:"offset"`
}

type communityManageCommentResponse struct {
	ID          string    `json:"id"`
	PostID      string    `json:"post_id"`
	AuthorID    string    `json:"author_id"`
	ParentID    string    `json:"parent_id"`
	BodyExcerpt string    `json:"body_excerpt"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type listCommunityManageReportsResponse struct {
	Community communityResponse               `json:"community"`
	Reports   []communityManageReportResponse `json:"reports"`
	Status    string                          `json:"status"`
	Limit     int                             `json:"limit"`
	Offset    int                             `json:"offset"`
}

type communityManageReportResponse struct {
	ID            string                                      `json:"id"`
	TargetType    string                                      `json:"target_type"`
	PostID        string                                      `json:"post_id"`
	CommentID     string                                      `json:"comment_id"`
	ReporterID    string                                      `json:"reporter_id"`
	Reason        string                                      `json:"reason"`
	Status        string                                      `json:"status"`
	ReviewedBy    string                                      `json:"reviewed_by"`
	ReviewedAt    *time.Time                                  `json:"reviewed_at"`
	TargetPreview *communityManageReportTargetPreviewResponse `json:"target_preview"`
	CreatedAt     time.Time                                   `json:"created_at"`
	UpdatedAt     time.Time                                   `json:"updated_at"`
}

type communityManageReportTargetPreviewResponse struct {
	TargetType  string    `json:"target_type"`
	PostID      string    `json:"post_id"`
	CommentID   string    `json:"comment_id"`
	AuthorID    string    `json:"author_id"`
	Status      string    `json:"status"`
	Title       string    `json:"title"`
	BodyExcerpt string    `json:"body_excerpt"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type listCommunityRulesResponse struct {
	Community communityResponse       `json:"community"`
	Rules     []communityRuleResponse `json:"rules"`
}

type createCommunityRuleResponse struct {
	Community communityResponse     `json:"community"`
	Rule      communityRuleResponse `json:"rule"`
}

type updateCommunityRuleResponse struct {
	Community communityResponse     `json:"community"`
	Rule      communityRuleResponse `json:"rule"`
}

type communitySettingsResponse struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type communityRuleResponse struct {
	ID          string    `json:"id"`
	CommunityID string    `json:"community_id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	Position    int       `json:"position"`
	CreatedBy   string    `json:"created_by"`
	UpdatedBy   string    `json:"updated_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type updateCommunityManageSettingsRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type writeCommunityRuleRequest struct {
	Title    string `json:"title" binding:"required"`
	Body     string `json:"body"`
	Position int    `json:"position"`
}

type submitCommunityApplicationRequest struct {
	RequestedSlug string `json:"requested_slug" binding:"required"`
	RequestedName string `json:"requested_name" binding:"required"`
	Reason        string `json:"reason" binding:"required"`
}

type rejectCommunityApplicationRequest struct {
	RejectReason string `json:"reject_reason" binding:"required"`
}

type communityApplicationResponse struct {
	ID            string     `json:"id"`
	ApplicantID   string     `json:"applicant_id"`
	RequestedSlug string     `json:"requested_slug"`
	RequestedName string     `json:"requested_name"`
	Reason        string     `json:"reason"`
	Status        string     `json:"status"`
	ReviewedBy    *string    `json:"reviewed_by"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
	RejectReason  string     `json:"reject_reason"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type listCommunityApplicationsResponse struct {
	Applications []communityApplicationResponse `json:"applications"`
	Limit        int                            `json:"limit"`
	Offset       int                            `json:"offset"`
}

type getCommunityApplicationResponse struct {
	Application communityApplicationResponse `json:"application"`
}

type submitCommunityApplicationResponse struct {
	Application communityApplicationResponse `json:"application"`
}

type approveCommunityApplicationResponse struct {
	Application communityApplicationResponse `json:"application"`
	Community   communityResponse            `json:"community"`
}

type rejectCommunityApplicationResponse struct {
	Application communityApplicationResponse `json:"application"`
}

type communityResponse struct {
	ID                string                             `json:"id"`
	Slug              string                             `json:"slug"`
	Name              string                             `json:"name"`
	Description       string                             `json:"description"`
	AvatarURL         string                             `json:"avatar_url"`
	BannerURL         string                             `json:"banner_url"`
	Kind              string                             `json:"kind"`
	Status            string                             `json:"status"`
	Visibility        string                             `json:"visibility"`
	MemberCount       int                                `json:"member_count"`
	PostCount         int                                `json:"post_count"`
	ViewerIsFollowing bool                               `json:"viewer_is_following"`
	ViewerRole        string                             `json:"viewer_role"`
	ViewerPermissions communityViewerPermissionsResponse `json:"viewer_permissions"`
	CreatedAt         time.Time                          `json:"created_at"`
	UpdatedAt         time.Time                          `json:"updated_at"`
}

type communityViewerPermissionsResponse struct {
	CanPost     bool `json:"can_post"`
	CanManage   bool `json:"can_manage"`
	CanModerate bool `json:"can_moderate"`
}

func NewHandler(communities CommunityReadUseCase, applications CommunityApplicationUseCase) *Handler {
	return &Handler{
		communities:  communities,
		applications: applications,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	RegisterReadRoutes(group, handler)
	RegisterApplicationRoutes(group, handler)
	RegisterFollowRoutes(group, handler)
}

func RegisterReadRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/communities", handler.ListCommunities)
	group.GET("/communities/:slug", handler.GetCommunity)
}

func RegisterApplicationRoutes(group *gin.RouterGroup, handler *Handler) {
	group.POST("/community-applications", handler.SubmitCommunityApplication)
	group.GET("/community-applications", handler.ListCommunityApplications)
	group.GET("/community-applications/:id", handler.GetCommunityApplication)
	group.POST("/community-applications/:id/approve", handler.ApproveCommunityApplication)
	group.POST("/community-applications/:id/reject", handler.RejectCommunityApplication)
}

func RegisterFollowRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/me/followed-communities", handler.ListFollowedCommunities)
	group.POST("/communities/:slug/follow", handler.FollowCommunity)
	group.DELETE("/communities/:slug/follow", handler.DeleteCommunityFollow)
}

func RegisterManageRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/communities/:slug/manage", handler.GetCommunityManageContext)
	group.GET("/communities/:slug/manage/posts", handler.ListCommunityManagePosts)
	group.GET("/communities/:slug/manage/comments", handler.ListCommunityManageComments)
	group.GET("/communities/:slug/manage/reports", handler.ListCommunityManageReports)
	group.GET("/communities/:slug/manage/members", handler.ListCommunityMembers)
	group.GET("/communities/:slug/manage/settings", handler.GetCommunityManageSettings)
	group.PATCH("/communities/:slug/manage/settings", handler.UpdateCommunityManageSettings)
	group.GET("/communities/:slug/manage/rules", handler.ListCommunityRules)
	group.POST("/communities/:slug/manage/rules", handler.CreateCommunityRule)
	group.PATCH("/communities/:slug/manage/rules/:rule_id", handler.UpdateCommunityRule)
	group.DELETE("/communities/:slug/manage/rules/:rule_id", handler.DeleteCommunityRule)
}

func (h *Handler) ListCommunities(c *gin.Context) {
	userID, _ := authcontext.CurrentUserID(c.Request.Context())

	result, err := h.communities.ListCommunities(c.Request.Context(), communityusecase.ListCommunitiesInput{
		ViewerID: userID,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listCommunitiesResponse{
		Communities: make([]communityResponse, 0, len(result.Communities)),
	}
	for _, community := range result.Communities {
		response.Communities = append(response.Communities, toCommunityResponse(community))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetCommunity(c *gin.Context) {
	userID, _ := authcontext.CurrentUserID(c.Request.Context())

	result, err := h.communities.GetCommunityBySlug(c.Request.Context(), communityusecase.GetCommunityInput{
		Slug:     c.Param("slug"),
		ViewerID: userID,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, getCommunityResponse{
		Community: toCommunityResponse(result.Community),
	})
}

func (h *Handler) FollowCommunity(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	if _, err := h.communities.FollowCommunity(c.Request.Context(), communityusecase.FollowCommunityInput{
		Slug:   c.Param("slug"),
		UserID: userID,
	}); err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteCommunityFollow(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	if _, err := h.communities.DeleteCommunityFollow(c.Request.Context(), communityusecase.DeleteCommunityFollowInput{
		Slug:   c.Param("slug"),
		UserID: userID,
	}); err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) ListFollowedCommunities(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
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

	result, err := h.communities.ListFollowedCommunities(c.Request.Context(), communityusecase.ListFollowedCommunitiesInput{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listFollowedCommunitiesResponse{
		Communities: make([]communityResponse, 0, len(result.Communities)),
		Limit:       result.Limit,
		Offset:      result.Offset,
	}
	for _, community := range result.Communities {
		response.Communities = append(response.Communities, toCommunityResponse(community))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetCommunityManageContext(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	result, err := h.communities.GetCommunityManageContext(c.Request.Context(), communityusecase.GetCommunityManageContextInput{
		Slug:     c.Param("slug"),
		ViewerID: userID,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, getCommunityManageContextResponse{
		Community: toCommunityResponse(result.Community),
	})
}

func (h *Handler) ListCommunityMembers(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
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

	result, err := h.communities.ListCommunityMembers(c.Request.Context(), communityusecase.ListCommunityMembersInput{
		Slug:     c.Param("slug"),
		ViewerID: userID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listCommunityMembersResponse{
		Community: toCommunityResponse(result.Community),
		Members:   make([]communityMemberResponse, 0, len(result.Members)),
		Limit:     result.Limit,
		Offset:    result.Offset,
	}
	for _, member := range result.Members {
		response.Members = append(response.Members, toCommunityMemberResponse(member))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) ListCommunityManagePosts(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
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

	result, err := h.communities.ListCommunityManagePosts(c.Request.Context(), communityusecase.ListCommunityManagePostsInput{
		Slug:     c.Param("slug"),
		ViewerID: userID,
		Status:   c.Query("status"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listCommunityManagePostsResponse{
		Community: toCommunityResponse(result.Community),
		Posts:     make([]communityManagePostResponse, 0, len(result.Posts)),
		Status:    result.Status,
		Limit:     result.Limit,
		Offset:    result.Offset,
	}
	for _, post := range result.Posts {
		response.Posts = append(response.Posts, toCommunityManagePostResponse(post))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) ListCommunityManageComments(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
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

	result, err := h.communities.ListCommunityManageComments(c.Request.Context(), communityusecase.ListCommunityManageCommentsInput{
		Slug:     c.Param("slug"),
		ViewerID: userID,
		Status:   c.Query("status"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listCommunityManageCommentsResponse{
		Community: toCommunityResponse(result.Community),
		Comments:  make([]communityManageCommentResponse, 0, len(result.Comments)),
		Status:    result.Status,
		Limit:     result.Limit,
		Offset:    result.Offset,
	}
	for _, comment := range result.Comments {
		response.Comments = append(response.Comments, toCommunityManageCommentResponse(comment))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) ListCommunityManageReports(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
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

	result, err := h.communities.ListCommunityManageReports(c.Request.Context(), communityusecase.ListCommunityManageReportsInput{
		Slug:     c.Param("slug"),
		ViewerID: userID,
		Status:   c.Query("status"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listCommunityManageReportsResponse{
		Community: toCommunityResponse(result.Community),
		Reports:   make([]communityManageReportResponse, 0, len(result.Reports)),
		Status:    result.Status,
		Limit:     result.Limit,
		Offset:    result.Offset,
	}
	for _, report := range result.Reports {
		response.Reports = append(response.Reports, toCommunityManageReportResponse(report))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetCommunityManageSettings(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	result, err := h.communities.GetCommunityManageSettings(c.Request.Context(), communityusecase.GetCommunityManageSettingsInput{
		Slug:     c.Param("slug"),
		ViewerID: userID,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, getCommunityManageSettingsResponse{
		Community: toCommunityResponse(result.Community),
		Settings:  toCommunitySettingsResponse(result.Settings),
	})
}

func (h *Handler) UpdateCommunityManageSettings(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req updateCommunityManageSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid community settings request"))
		c.Abort()
		return
	}

	result, err := h.communities.UpdateCommunityManageSettings(c.Request.Context(), communityusecase.UpdateCommunityManageSettingsInput{
		Slug:        c.Param("slug"),
		ViewerID:    userID,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, updateCommunityManageSettingsResponse{
		Community: toCommunityResponse(result.Community),
		Settings:  toCommunitySettingsResponse(result.Settings),
	})
}

func (h *Handler) ListCommunityRules(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	result, err := h.communities.ListCommunityRules(c.Request.Context(), communityusecase.ListCommunityRulesInput{
		Slug:     c.Param("slug"),
		ViewerID: userID,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listCommunityRulesResponse{
		Community: toCommunityResponse(result.Community),
		Rules:     make([]communityRuleResponse, 0, len(result.Rules)),
	}
	for _, rule := range result.Rules {
		response.Rules = append(response.Rules, toCommunityRuleResponse(rule))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) CreateCommunityRule(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req writeCommunityRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid community rule request"))
		c.Abort()
		return
	}

	result, err := h.communities.CreateCommunityRule(c.Request.Context(), communityusecase.CreateCommunityRuleInput{
		Slug:     c.Param("slug"),
		ViewerID: userID,
		Title:    req.Title,
		Body:     req.Body,
		Position: req.Position,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, createCommunityRuleResponse{
		Community: toCommunityResponse(result.Community),
		Rule:      toCommunityRuleResponse(result.Rule),
	})
}

func (h *Handler) UpdateCommunityRule(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req writeCommunityRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid community rule request"))
		c.Abort()
		return
	}

	result, err := h.communities.UpdateCommunityRule(c.Request.Context(), communityusecase.UpdateCommunityRuleInput{
		Slug:     c.Param("slug"),
		RuleID:   c.Param("rule_id"),
		ViewerID: userID,
		Title:    req.Title,
		Body:     req.Body,
		Position: req.Position,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, updateCommunityRuleResponse{
		Community: toCommunityResponse(result.Community),
		Rule:      toCommunityRuleResponse(result.Rule),
	})
}

func (h *Handler) DeleteCommunityRule(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	if _, err := h.communities.DeleteCommunityRule(c.Request.Context(), communityusecase.DeleteCommunityRuleInput{
		Slug:     c.Param("slug"),
		RuleID:   c.Param("rule_id"),
		ViewerID: userID,
	}); err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) SubmitCommunityApplication(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req submitCommunityApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid community application request"))
		c.Abort()
		return
	}

	result, err := h.applications.SubmitCommunityApplication(c.Request.Context(), communityusecase.SubmitCommunityApplicationInput{
		ApplicantID:   userID,
		RequestedSlug: req.RequestedSlug,
		RequestedName: req.RequestedName,
		Reason:        req.Reason,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, submitCommunityApplicationResponse{
		Application: toCommunityApplicationResponse(result.Application),
	})
}

func (h *Handler) ListCommunityApplications(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
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

	result, err := h.applications.ListCommunityApplications(c.Request.Context(), communityusecase.ListCommunityApplicationsInput{
		ReviewerID: userID,
		Status:     c.Query("status"),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listCommunityApplicationsResponse{
		Applications: make([]communityApplicationResponse, 0, len(result.Applications)),
		Limit:        result.Limit,
		Offset:       result.Offset,
	}
	for _, application := range result.Applications {
		response.Applications = append(response.Applications, toCommunityApplicationResponse(application))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetCommunityApplication(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	result, err := h.applications.GetCommunityApplication(c.Request.Context(), communityusecase.GetCommunityApplicationInput{
		ReviewerID:    userID,
		ApplicationID: c.Param("id"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, getCommunityApplicationResponse{
		Application: toCommunityApplicationResponse(result.Application),
	})
}

func (h *Handler) ApproveCommunityApplication(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	result, err := h.applications.ApproveCommunityApplication(c.Request.Context(), communityusecase.ReviewCommunityApplicationInput{
		ApplicationID: c.Param("id"),
		ReviewerID:    userID,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, approveCommunityApplicationResponse{
		Application: toCommunityApplicationResponse(result.Application),
		Community:   toCommunityResponse(result.Community),
	})
}

func (h *Handler) RejectCommunityApplication(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req rejectCommunityApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid community application review request"))
		c.Abort()
		return
	}

	result, err := h.applications.RejectCommunityApplication(c.Request.Context(), communityusecase.ReviewCommunityApplicationInput{
		ApplicationID: c.Param("id"),
		ReviewerID:    userID,
		RejectReason:  req.RejectReason,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, rejectCommunityApplicationResponse{
		Application: toCommunityApplicationResponse(result.Application),
	})
}

func parseOptionalIntQuery(c *gin.Context, key string) (int, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, apperr.New(apperr.CodeInvalidArgument, "invalid "+key+" query")
	}
	return value, nil
}

func toCommunityResponse(community communityusecase.Community) communityResponse {
	return communityResponse{
		ID:                community.ID,
		Slug:              community.Slug,
		Name:              community.Name,
		Description:       community.Description,
		AvatarURL:         community.AvatarURL,
		BannerURL:         community.BannerURL,
		Kind:              community.Kind,
		Status:            community.Status,
		Visibility:        community.Visibility,
		MemberCount:       community.MemberCount,
		PostCount:         community.PostCount,
		ViewerIsFollowing: community.ViewerIsFollowing,
		ViewerRole:        community.ViewerRole,
		ViewerPermissions: communityViewerPermissionsResponse{
			CanPost:     community.ViewerPermissions.CanPost,
			CanManage:   community.ViewerPermissions.CanManage,
			CanModerate: community.ViewerPermissions.CanModerate,
		},
		CreatedAt: community.CreatedAt,
		UpdatedAt: community.UpdatedAt,
	}
}

func toCommunityApplicationResponse(application communityusecase.CommunityApplication) communityApplicationResponse {
	response := communityApplicationResponse{
		ID:            application.ID,
		ApplicantID:   application.ApplicantID,
		RequestedSlug: application.RequestedSlug,
		RequestedName: application.RequestedName,
		Reason:        application.Reason,
		Status:        application.Status,
		ReviewedAt:    application.ReviewedAt,
		RejectReason:  application.RejectReason,
		CreatedAt:     application.CreatedAt,
		UpdatedAt:     application.UpdatedAt,
	}
	if application.ReviewedBy != "" {
		reviewedBy := application.ReviewedBy
		response.ReviewedBy = &reviewedBy
	}
	return response
}

func toCommunitySettingsResponse(settings communityusecase.CommunitySettings) communitySettingsResponse {
	return communitySettingsResponse{
		Name:        settings.Name,
		Description: settings.Description,
		UpdatedAt:   settings.UpdatedAt,
	}
}

func toCommunityRuleResponse(rule communityusecase.CommunityRule) communityRuleResponse {
	return communityRuleResponse{
		ID:          rule.ID,
		CommunityID: rule.CommunityID,
		Title:       rule.Title,
		Body:        rule.Body,
		Position:    rule.Position,
		CreatedBy:   rule.CreatedBy,
		UpdatedBy:   rule.UpdatedBy,
		CreatedAt:   rule.CreatedAt,
		UpdatedAt:   rule.UpdatedAt,
	}
}

func toCommunityMemberResponse(member communityusecase.CommunityMember) communityMemberResponse {
	displayName := member.DisplayName
	if displayName == "" {
		displayName = member.Username
	}
	return communityMemberResponse{
		User: communityMemberUserResponse{
			ID:          member.UserID,
			Username:    member.Username,
			DisplayName: displayName,
			AvatarURL:   member.AvatarURL,
			Headline:    member.Headline,
			Badges:      []string{},
		},
		Role:      member.Role,
		Status:    member.Status,
		CreatedAt: member.CreatedAt,
		UpdatedAt: member.UpdatedAt,
	}
}

func toCommunityManagePostResponse(post communityusecase.CommunityManagePost) communityManagePostResponse {
	return communityManagePostResponse{
		ID:          post.ID,
		CommunityID: post.CommunityID,
		AuthorID:    post.AuthorID,
		Title:       post.Title,
		BodyExcerpt: post.BodyExcerpt,
		Status:      post.Status,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
	}
}

func toCommunityManageCommentResponse(comment communityusecase.CommunityManageComment) communityManageCommentResponse {
	return communityManageCommentResponse{
		ID:          comment.ID,
		PostID:      comment.PostID,
		AuthorID:    comment.AuthorID,
		ParentID:    comment.ParentID,
		BodyExcerpt: comment.BodyExcerpt,
		Status:      comment.Status,
		CreatedAt:   comment.CreatedAt,
		UpdatedAt:   comment.UpdatedAt,
	}
}

func toCommunityManageReportResponse(report communityusecase.CommunityManageReport) communityManageReportResponse {
	return communityManageReportResponse{
		ID:            report.ID,
		TargetType:    report.TargetType,
		PostID:        report.PostID,
		CommentID:     report.CommentID,
		ReporterID:    report.ReporterID,
		Reason:        report.Reason,
		Status:        report.Status,
		ReviewedBy:    report.ReviewedBy,
		ReviewedAt:    report.ReviewedAt,
		TargetPreview: toCommunityManageReportTargetPreviewResponse(report.TargetPreview),
		CreatedAt:     report.CreatedAt,
		UpdatedAt:     report.UpdatedAt,
	}
}

func toCommunityManageReportTargetPreviewResponse(preview *communityusecase.CommunityManageReportTargetPreview) *communityManageReportTargetPreviewResponse {
	if preview == nil {
		return nil
	}
	return &communityManageReportTargetPreviewResponse{
		TargetType:  preview.TargetType,
		PostID:      preview.PostID,
		CommentID:   preview.CommentID,
		AuthorID:    preview.AuthorID,
		Status:      preview.Status,
		Title:       preview.Title,
		BodyExcerpt: preview.BodyExcerpt,
		CreatedAt:   preview.CreatedAt,
		UpdatedAt:   preview.UpdatedAt,
	}
}

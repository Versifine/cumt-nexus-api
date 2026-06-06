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
	ListCommunities(ctx context.Context) (communityusecase.ListCommunitiesResult, error)
	GetCommunityBySlug(ctx context.Context, input communityusecase.GetCommunityInput) (communityusecase.GetCommunityResult, error)
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

func (h *Handler) ListCommunities(c *gin.Context) {
	result, err := h.communities.ListCommunities(c.Request.Context())
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
	result, err := h.communities.GetCommunityBySlug(c.Request.Context(), communityusecase.GetCommunityInput{
		Slug: c.Param("slug"),
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

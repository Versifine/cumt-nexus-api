package communityhttp

import (
	"context"
	"net/http"
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
	ReviewedBy    string     `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	RejectReason  string     `json:"reject_reason,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
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
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"`
	Visibility  string    `json:"visibility"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewHandler(communities CommunityReadUseCase, applications CommunityApplicationUseCase) *Handler {
	return &Handler{
		communities:  communities,
		applications: applications,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/communities", handler.ListCommunities)
	group.GET("/communities/:slug", handler.GetCommunity)
	group.POST("/community-applications", handler.SubmitCommunityApplication)
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

func toCommunityResponse(community communityusecase.Community) communityResponse {
	return communityResponse{
		ID:          community.ID,
		Slug:        community.Slug,
		Name:        community.Name,
		Description: community.Description,
		Kind:        community.Kind,
		Status:      community.Status,
		Visibility:  community.Visibility,
		CreatedAt:   community.CreatedAt,
		UpdatedAt:   community.UpdatedAt,
	}
}

func toCommunityApplicationResponse(application communityusecase.CommunityApplication) communityApplicationResponse {
	return communityApplicationResponse{
		ID:            application.ID,
		ApplicantID:   application.ApplicantID,
		RequestedSlug: application.RequestedSlug,
		RequestedName: application.RequestedName,
		Reason:        application.Reason,
		Status:        application.Status,
		ReviewedBy:    application.ReviewedBy,
		ReviewedAt:    application.ReviewedAt,
		RejectReason:  application.RejectReason,
		CreatedAt:     application.CreatedAt,
		UpdatedAt:     application.UpdatedAt,
	}
}

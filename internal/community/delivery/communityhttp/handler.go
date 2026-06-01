package communityhttp

import (
	"context"
	"net/http"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/community/communityusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	communities CommunityReadUseCase
}

type CommunityReadUseCase interface {
	ListCommunities(ctx context.Context) (communityusecase.ListCommunitiesResult, error)
	GetCommunityBySlug(ctx context.Context, input communityusecase.GetCommunityInput) (communityusecase.GetCommunityResult, error)
}

type listCommunitiesResponse struct {
	Communities []communityResponse `json:"communities"`
}

type getCommunityResponse struct {
	Community communityResponse `json:"community"`
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

func NewHandler(communities CommunityReadUseCase) *Handler {
	return &Handler{
		communities: communities,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/communities", handler.ListCommunities)
	group.GET("/communities/:slug", handler.GetCommunity)
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

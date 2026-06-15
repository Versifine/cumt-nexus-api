package searchhttp

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/search/searchusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	search SearchUseCase
}

type SearchUseCase interface {
	Search(ctx context.Context, input searchusecase.SearchInput) (searchusecase.SearchResult, error)
}

type searchResponse struct {
	Query       string                    `json:"query"`
	Scope       string                    `json:"scope"`
	Limit       int                       `json:"limit"`
	Offset      int                       `json:"offset"`
	NextOffset  int                       `json:"next_offset"`
	HasMore     bool                      `json:"has_more"`
	Communities []searchCommunityResponse `json:"communities"`
	Posts       []searchPostResponse      `json:"posts"`
	Users       []searchUserResponse      `json:"users"`
}

type searchCommunityResponse struct {
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

type searchPostResponse struct {
	ID            string    `json:"id"`
	CommunityID   string    `json:"community_id"`
	CommunitySlug string    `json:"community_slug"`
	AuthorID      string    `json:"author_id"`
	Title         string    `json:"title"`
	BodyExcerpt   string    `json:"body_excerpt"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type searchUserResponse struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	Headline    string    `json:"headline"`
	BioExcerpt  string    `json:"bio_excerpt"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewHandler(search SearchUseCase) *Handler {
	return &Handler{
		search: search,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/search", handler.Search)
}

func (h *Handler) Search(c *gin.Context) {
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

	result, err := h.search.Search(c.Request.Context(), searchusecase.SearchInput{
		Query:  c.Query("q"),
		Scope:  c.Query("scope"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, toSearchResponse(result))
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

func toSearchResponse(result searchusecase.SearchResult) searchResponse {
	response := searchResponse{
		Query:       result.Query,
		Scope:       result.Scope,
		Limit:       result.Limit,
		Offset:      result.Offset,
		NextOffset:  result.NextOffset,
		HasMore:     result.HasMore,
		Communities: make([]searchCommunityResponse, 0, len(result.Communities)),
		Posts:       make([]searchPostResponse, 0, len(result.Posts)),
		Users:       make([]searchUserResponse, 0, len(result.Users)),
	}
	for _, community := range result.Communities {
		response.Communities = append(response.Communities, searchCommunityResponse{
			ID:          community.ID,
			Slug:        community.Slug,
			Name:        community.Name,
			Description: community.Description,
			Kind:        community.Kind,
			Status:      community.Status,
			Visibility:  community.Visibility,
			CreatedAt:   community.CreatedAt,
			UpdatedAt:   community.UpdatedAt,
		})
	}
	for _, post := range result.Posts {
		response.Posts = append(response.Posts, searchPostResponse{
			ID:            post.ID,
			CommunityID:   post.CommunityID,
			CommunitySlug: post.CommunitySlug,
			AuthorID:      post.AuthorID,
			Title:         post.Title,
			BodyExcerpt:   post.BodyExcerpt,
			Status:        post.Status,
			CreatedAt:     post.CreatedAt,
			UpdatedAt:     post.UpdatedAt,
		})
	}
	for _, user := range result.Users {
		response.Users = append(response.Users, searchUserResponse{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			AvatarURL:   user.AvatarURL,
			Headline:    user.Headline,
			BioExcerpt:  user.BioExcerpt,
			Status:      user.Status,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
		})
	}
	return response
}

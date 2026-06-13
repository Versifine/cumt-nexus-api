package contentrefhttp

import (
	"context"
	"net/http"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/contentref/contentrefusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	contentRefs UseCase
}

type UseCase interface {
	ResolveLinkPreview(ctx context.Context, input contentrefusecase.ResolveLinkPreviewInput) (contentrefusecase.LinkPreview, error)
	ResolveEmbed(ctx context.Context, input contentrefusecase.ResolveEmbedInput) (contentrefusecase.Embed, error)
}

type resolveLinkPreviewRequest struct {
	URL string `json:"url" binding:"required"`
}

type resolveEmbedRequest struct {
	URL string `json:"url" binding:"required"`
}

type linkPreviewResponse struct {
	Provider     string `json:"provider"`
	URL          string `json:"url"`
	CanonicalURL string `json:"canonical_url"`
	Host         string `json:"host"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	ImageURL     string `json:"image_url"`
}

type resolveLinkPreviewResponse struct {
	Preview linkPreviewResponse `json:"preview"`
}

type embedResponse struct {
	ID            string `json:"id"`
	Provider      string `json:"provider"`
	ProviderRef   string `json:"provider_ref"`
	URL           string `json:"url"`
	CanonicalURL  string `json:"canonical_url"`
	EmbedURL      string `json:"embed_url"`
	IframeAllowed bool   `json:"iframe_allowed"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	ImageURL      string `json:"image_url"`
	AuthorName    string `json:"author_name"`
	Status        string `json:"status"`
}

type resolveEmbedResponse struct {
	Embed embedResponse `json:"embed"`
}

func NewHandler(contentRefs UseCase) *Handler {
	return &Handler{
		contentRefs: contentRefs,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.POST("/link-previews/resolve", handler.ResolveLinkPreview)
	group.POST("/embeds/resolve", handler.ResolveEmbed)
}

func (h *Handler) ResolveLinkPreview(c *gin.Context) {
	if _, ok := authcontext.CurrentUserID(c.Request.Context()); !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req resolveLinkPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid link preview resolve request"))
		c.Abort()
		return
	}

	result, err := h.contentRefs.ResolveLinkPreview(c.Request.Context(), contentrefusecase.ResolveLinkPreviewInput{
		URL: req.URL,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, resolveLinkPreviewResponse{
		Preview: toLinkPreviewResponse(result),
	})
}

func (h *Handler) ResolveEmbed(c *gin.Context) {
	if _, ok := authcontext.CurrentUserID(c.Request.Context()); !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req resolveEmbedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid embed resolve request"))
		c.Abort()
		return
	}

	result, err := h.contentRefs.ResolveEmbed(c.Request.Context(), contentrefusecase.ResolveEmbedInput{
		URL: req.URL,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, resolveEmbedResponse{
		Embed: toEmbedResponse(result),
	})
}

func toLinkPreviewResponse(preview contentrefusecase.LinkPreview) linkPreviewResponse {
	return linkPreviewResponse{
		Provider:     preview.Provider,
		URL:          preview.URL,
		CanonicalURL: preview.CanonicalURL,
		Host:         preview.Host,
		Title:        preview.Title,
		Description:  preview.Description,
		ImageURL:     preview.ImageURL,
	}
}

func toEmbedResponse(embed contentrefusecase.Embed) embedResponse {
	return embedResponse{
		ID:            embed.ID,
		Provider:      embed.Provider,
		ProviderRef:   embed.ProviderRef,
		URL:           embed.URL,
		CanonicalURL:  embed.CanonicalURL,
		EmbedURL:      embed.EmbedURL,
		IframeAllowed: embed.IframeAllowed,
		Title:         embed.Title,
		Description:   embed.Description,
		ImageURL:      embed.ImageURL,
		AuthorName:    embed.AuthorName,
		Status:        embed.Status,
	}
}

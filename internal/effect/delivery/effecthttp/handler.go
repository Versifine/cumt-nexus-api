package effecthttp

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/effect/effectusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	effects UseCase
}

type UseCase interface {
	ListEffectsCatalog(ctx context.Context) (effectusecase.ListEffectsCatalogResult, error)
	GetMyPoints(ctx context.Context, input effectusecase.GetMyPointsInput) (effectusecase.GetMyPointsResult, error)
	ListMyPointTransactions(ctx context.Context, input effectusecase.ListMyPointTransactionsInput) (effectusecase.ListMyPointTransactionsResult, error)
	ApplyCommentEffect(ctx context.Context, input effectusecase.ApplyCommentEffectInput) (effectusecase.ApplyCommentEffectResult, error)
}

type effectResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	CostPoints   int       `json:"cost_points"`
	AssetURL     string    `json:"asset_url"`
	AnimationKey string    `json:"animation_key"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type listEffectsCatalogResponse struct {
	Effects []effectResponse `json:"effects"`
}

type pointAccountResponse struct {
	UserID         string    `json:"user_id"`
	Balance        int       `json:"balance"`
	LifetimeEarned int       `json:"lifetime_earned"`
	LifetimeSpent  int       `json:"lifetime_spent"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type getMyPointsResponse struct {
	Points pointAccountResponse `json:"points"`
}

type pointTransactionResponse struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Delta        int       `json:"delta"`
	BalanceAfter int       `json:"balance_after"`
	Reason       string    `json:"reason"`
	SourceType   string    `json:"source_type"`
	SourceID     string    `json:"source_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type listMyPointTransactionsResponse struct {
	Transactions []pointTransactionResponse `json:"transactions"`
	Limit        int                        `json:"limit"`
	Offset       int                        `json:"offset"`
	NextOffset   int                        `json:"next_offset"`
	HasMore      bool                       `json:"has_more"`
}

type applyCommentEffectRequest struct {
	EffectID string `json:"effect_id" binding:"required"`
}

type commentEffectResponse struct {
	ID          string    `json:"id"`
	CommentID   string    `json:"comment_id"`
	EffectID    string    `json:"effect_id"`
	UserID      string    `json:"user_id"`
	PointsSpent int       `json:"points_spent"`
	CreatedAt   time.Time `json:"created_at"`
}

type applyCommentEffectResponse struct {
	CommentEffect commentEffectResponse `json:"comment_effect"`
	Points        pointAccountResponse  `json:"points"`
}

func NewHandler(effects UseCase) *Handler {
	return &Handler{
		effects: effects,
	}
}

func RegisterPublicRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/effects/catalog", handler.ListEffectsCatalog)
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/me/points", handler.GetMyPoints)
	group.GET("/me/point-transactions", handler.ListMyPointTransactions)
	group.POST("/comments/:id/effects", handler.ApplyCommentEffect)
}

func (h *Handler) ListEffectsCatalog(c *gin.Context) {
	result, err := h.effects.ListEffectsCatalog(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listEffectsCatalogResponse{
		Effects: make([]effectResponse, 0, len(result.Effects)),
	}
	for _, effect := range result.Effects {
		response.Effects = append(response.Effects, toEffectResponse(effect))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetMyPoints(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	result, err := h.effects.GetMyPoints(c.Request.Context(), effectusecase.GetMyPointsInput{
		ActorID: userID,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, getMyPointsResponse{
		Points: toPointAccountResponse(result.Points),
	})
}

func (h *Handler) ListMyPointTransactions(c *gin.Context) {
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

	result, err := h.effects.ListMyPointTransactions(c.Request.Context(), effectusecase.ListMyPointTransactionsInput{
		ActorID: userID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response := listMyPointTransactionsResponse{
		Transactions: make([]pointTransactionResponse, 0, len(result.Transactions)),
		Limit:        result.Limit,
		Offset:       result.Offset,
		NextOffset:   result.NextOffset,
		HasMore:      result.HasMore,
	}
	for _, transaction := range result.Transactions {
		response.Transactions = append(response.Transactions, toPointTransactionResponse(transaction))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) ApplyCommentEffect(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req applyCommentEffectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid comment effect request"))
		c.Abort()
		return
	}

	result, err := h.effects.ApplyCommentEffect(c.Request.Context(), effectusecase.ApplyCommentEffectInput{
		ActorID:   userID,
		CommentID: c.Param("id"),
		EffectID:  req.EffectID,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, applyCommentEffectResponse{
		CommentEffect: toCommentEffectResponse(result.CommentEffect),
		Points:        toPointAccountResponse(result.Points),
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

func toEffectResponse(effect effectusecase.Effect) effectResponse {
	return effectResponse{
		ID:           effect.ID,
		Name:         effect.Name,
		Description:  effect.Description,
		CostPoints:   effect.CostPoints,
		AssetURL:     effect.AssetURL,
		AnimationKey: effect.AnimationKey,
		IsActive:     effect.IsActive,
		CreatedAt:    effect.CreatedAt,
		UpdatedAt:    effect.UpdatedAt,
	}
}

func toPointAccountResponse(points effectusecase.PointAccount) pointAccountResponse {
	return pointAccountResponse{
		UserID:         points.UserID,
		Balance:        points.Balance,
		LifetimeEarned: points.LifetimeEarned,
		LifetimeSpent:  points.LifetimeSpent,
		UpdatedAt:      points.UpdatedAt,
	}
}

func toPointTransactionResponse(transaction effectusecase.PointTransaction) pointTransactionResponse {
	return pointTransactionResponse{
		ID:           transaction.ID,
		UserID:       transaction.UserID,
		Delta:        transaction.Delta,
		BalanceAfter: transaction.BalanceAfter,
		Reason:       transaction.Reason,
		SourceType:   transaction.SourceType,
		SourceID:     transaction.SourceID,
		CreatedAt:    transaction.CreatedAt,
	}
}

func toCommentEffectResponse(commentEffect effectusecase.CommentEffect) commentEffectResponse {
	return commentEffectResponse{
		ID:          commentEffect.ID,
		CommentID:   commentEffect.CommentID,
		EffectID:    commentEffect.EffectID,
		UserID:      commentEffect.UserID,
		PointsSpent: commentEffect.PointsSpent,
		CreatedAt:   commentEffect.CreatedAt,
	}
}

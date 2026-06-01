package votehttp

import (
	"context"
	"net/http"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/vote/voteusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	votes PostVoteUseCase
}

type PostVoteUseCase interface {
	SetPostVote(ctx context.Context, input voteusecase.SetPostVoteInput) (voteusecase.SetPostVoteResult, error)
	DeletePostVote(ctx context.Context, input voteusecase.DeletePostVoteInput) error
}

type setPostVoteRequest struct {
	Value int `json:"value" binding:"required"`
}

type postVoteResponse struct {
	PostID    string    `json:"post_id"`
	UserID    string    `json:"user_id"`
	Value     int       `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type setPostVoteResponse struct {
	Vote postVoteResponse `json:"vote"`
}

func NewHandler(votes PostVoteUseCase) *Handler {
	return &Handler{
		votes: votes,
	}
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.PUT("/posts/:id/vote", handler.SetPostVote)
	group.DELETE("/posts/:id/vote", handler.DeletePostVote)
}

func (h *Handler) SetPostVote(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	var req setPostVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid post vote request"))
		c.Abort()
		return
	}

	result, err := h.votes.SetPostVote(c.Request.Context(), voteusecase.SetPostVoteInput{
		PostID: c.Param("id"),
		UserID: userID,
		Value:  req.Value,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, setPostVoteResponse{
		Vote: toPostVoteResponse(result.Vote),
	})
}

func (h *Handler) DeletePostVote(c *gin.Context) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return
	}

	if err := h.votes.DeletePostVote(c.Request.Context(), voteusecase.DeletePostVoteInput{
		PostID: c.Param("id"),
		UserID: userID,
	}); err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.Status(http.StatusNoContent)
}

func toPostVoteResponse(vote voteusecase.PostVote) postVoteResponse {
	return postVoteResponse{
		PostID:    vote.PostID,
		UserID:    vote.UserID,
		Value:     vote.Value,
		CreatedAt: vote.CreatedAt,
		UpdatedAt: vote.UpdatedAt,
	}
}

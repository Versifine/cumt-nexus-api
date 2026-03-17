package controller

import (
	"cumt-nexus-api/internal/service"
	"cumt-nexus-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type CommentSvc interface {
}

type CommentController struct {
	commentSvc CommentSvc
}

func NewCommentController(commentSvc CommentSvc) *CommentController {
	return &CommentController{commentSvc: commentSvc}
}

func (cc *CommentController) CreateComment(c *gin.Context) {
	var req struct {
		Content  string  `json:"content" binding:"required"`
		ParentID *uint64 `json:"parent_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		code, msg := service.MapError(service.ErrParamInvalid)
		response.Fail(c, code, msg)
		return
	}

}
func (cc *CommentController) ListComment(c *gin.Context) {
	// Implement the logic to get comments for a post

}

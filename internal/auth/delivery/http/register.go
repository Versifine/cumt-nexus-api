package authhttp

import (
	"time"

	"github.com/gin-gonic/gin"
)

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type registerResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        userResponse `json:"user"`
}
type userResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.POST("/register", handler.Register)
}

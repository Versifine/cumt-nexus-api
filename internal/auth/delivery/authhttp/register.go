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
	ID              string    `json:"id"`
	Username        string    `json:"username"`
	Status          string    `json:"status"`
	Email           string    `json:"email"`
	EmailVerified   bool      `json:"email_verified"`
	IsPlatformStaff bool      `json:"is_platform_staff"`
	PlatformRole    string    `json:"platform_role"`
	CreatedAt       time.Time `json:"created_at"`
}

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.POST("/register", handler.Register)
	group.POST("/email-codes/register", handler.SendRegisterEmailCode)
	group.POST("/register-with-email", handler.RegisterWithEmail)
	group.POST("/login", handler.Login)
	group.POST("/email-codes/login", handler.SendLoginEmailCode)
	group.POST("/login-with-email-code", handler.LoginWithEmailCode)
	group.POST("/email-codes/password-reset", handler.SendPasswordResetEmailCode)
	group.POST("/password-reset", handler.ResetPassword)
}

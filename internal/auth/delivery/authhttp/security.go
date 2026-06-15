package authhttp

import (
	"net/http"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

type emailCodeRequest struct {
	Email string `json:"email"`
}

type emailCodeResponse struct {
	Email       string `json:"email"`
	Purpose     string `json:"purpose"`
	ExpiresIn   int64  `json:"expires_in"`
	ResendAfter int64  `json:"resend_after"`
}

type registerWithEmailRequest struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginWithEmailCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type passwordResetRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

type accountSecurityResponse struct {
	Email           string     `json:"email"`
	EmailVerified   bool       `json:"email_verified"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	PasswordSet     bool       `json:"password_set"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type changeEmailCodeRequest struct {
	NewEmail string `json:"new_email"`
}

type changeEmailRequest struct {
	NewEmail string `json:"new_email"`
	Code     string `json:"code"`
}

type changeEmailResponse struct {
	Email           string     `json:"email"`
	EmailVerified   bool       `json:"email_verified"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type passwordUpdatedResponse struct {
	Updated bool `json:"updated"`
}

type deleteAccountCodeRequest struct {
	Email string `json:"email"`
}

type deleteAccountRequest struct {
	Code            string `json:"code"`
	CurrentPassword string `json:"current_password"`
	Confirmation    string `json:"confirmation"`
}

func RegisterSecurityRoutes(group *gin.RouterGroup, handler *Handler) {
	group.GET("/me/security", handler.GetSecurity)
	group.POST("/me/security/email-codes/change-email", handler.SendChangeEmailCode)
	group.POST("/me/security/email-codes/delete-account", handler.SendDeleteAccountEmailCode)
	group.PATCH("/me/security/email", handler.ChangeEmail)
	group.PATCH("/me/security/password", handler.ChangePassword)
	group.DELETE("/me/account", handler.DeleteAccount)
	group.POST("/auth/logout-all", handler.LogoutAll)
}

func (h *Handler) GetSecurity(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}
	result, err := h.security.GetSecurity(c.Request.Context(), authusecase.AccountSecurityInput{UserID: userID})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, accountSecurityResponse{
		Email:           result.Email,
		EmailVerified:   result.EmailVerified,
		EmailVerifiedAt: result.EmailVerifiedAt,
		PasswordSet:     result.PasswordSet,
		LastLoginAt:     result.LastLoginAt,
		CreatedAt:       result.CreatedAt,
	})
}

func (h *Handler) SendChangeEmailCode(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}
	var req changeEmailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid change email code request"))
		c.Abort()
		return
	}
	result, err := h.security.SendChangeEmailCode(c.Request.Context(), authusecase.ChangeEmailCodeInput{
		UserID:    userID,
		NewEmail:  req.NewEmail,
		RequestIP: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	})
	respondEmailCode(c, result, err)
}

func (h *Handler) ChangeEmail(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}
	var req changeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid change email request"))
		c.Abort()
		return
	}
	result, err := h.security.ChangeEmail(c.Request.Context(), authusecase.ChangeEmailInput{
		UserID:    userID,
		NewEmail:  req.NewEmail,
		Code:      req.Code,
		RequestIP: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, changeEmailResponse{
		Email:           result.Email,
		EmailVerified:   result.EmailVerified,
		EmailVerifiedAt: result.EmailVerifiedAt,
	})
}

func (h *Handler) ChangePassword(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid change password request"))
		c.Abort()
		return
	}
	result, err := h.security.ChangePassword(c.Request.Context(), authusecase.ChangePasswordInput{
		UserID:          userID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
		RequestIP:       c.ClientIP(),
		UserAgent:       c.GetHeader("User-Agent"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, passwordUpdatedResponse{Updated: result.Updated})
}

func (h *Handler) LogoutAll(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}
	if err := h.security.LogoutAll(c.Request.Context(), authusecase.LogoutAllInput{
		UserID:    userID,
		RequestIP: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}); err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) SendDeleteAccountEmailCode(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}
	var req deleteAccountCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid delete account code request"))
		c.Abort()
		return
	}
	result, err := h.security.SendDeleteAccountEmailCode(c.Request.Context(), authusecase.DeleteAccountCodeInput{
		UserID:    userID,
		Email:     req.Email,
		RequestIP: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	})
	respondEmailCode(c, result, err)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}
	var req deleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid delete account request"))
		c.Abort()
		return
	}
	if err := h.security.DeleteAccount(c.Request.Context(), authusecase.DeleteAccountInput{
		UserID:          userID,
		Code:            req.Code,
		CurrentPassword: req.CurrentPassword,
		Confirmation:    req.Confirmation,
		RequestIP:       c.ClientIP(),
		UserAgent:       c.GetHeader("User-Agent"),
	}); err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.Status(http.StatusNoContent)
}

func currentUserIDOrAbort(c *gin.Context) (userdomain.UserID, bool) {
	userID, ok := authcontext.CurrentUserID(c.Request.Context())
	if !ok {
		_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "authentication required"))
		c.Abort()
		return "", false
	}
	return userID, true
}

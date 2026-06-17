package authhttp

import (
	"context"
	"net/http"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authusecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	register RegisterUseCase
	login    LoginUseCase
	security SecurityUseCase
}

func NewHandler(register RegisterUseCase, login LoginUseCase) *Handler {
	return &Handler{
		register: register,
		login:    login,
	}
}

type RegisterUseCase interface {
	Register(ctx context.Context, input authusecase.RegisterInput) (authusecase.RegisterResult, error)
}

type LoginUseCase interface {
	Login(ctx context.Context, input authusecase.LoginInput) (authusecase.LoginResult, error)
}

type SecurityUseCase interface {
	SendRegisterEmailCode(ctx context.Context, input authusecase.EmailCodeDispatchInput) (authusecase.EmailCodeDispatchResult, error)
	RegisterWithEmail(ctx context.Context, input authusecase.RegisterWithEmailInput) (authusecase.SecurityAuthResult, error)
	SendLoginEmailCode(ctx context.Context, input authusecase.EmailCodeDispatchInput) (authusecase.EmailCodeDispatchResult, error)
	LoginWithEmailCode(ctx context.Context, input authusecase.LoginWithEmailCodeInput) (authusecase.SecurityAuthResult, error)
	SendPasswordResetEmailCode(ctx context.Context, input authusecase.EmailCodeDispatchInput) (authusecase.EmailCodeDispatchResult, error)
	ResetPassword(ctx context.Context, input authusecase.PasswordResetInput) (authusecase.PasswordUpdatedResult, error)
	GetSecurity(ctx context.Context, input authusecase.AccountSecurityInput) (authusecase.AccountSecurityResult, error)
	SendChangeEmailCode(ctx context.Context, input authusecase.ChangeEmailCodeInput) (authusecase.EmailCodeDispatchResult, error)
	ChangeEmail(ctx context.Context, input authusecase.ChangeEmailInput) (authusecase.EmailChangeResult, error)
	ChangePassword(ctx context.Context, input authusecase.ChangePasswordInput) (authusecase.PasswordUpdatedResult, error)
	LogoutAll(ctx context.Context, input authusecase.LogoutAllInput) error
	SendDeleteAccountEmailCode(ctx context.Context, input authusecase.DeleteAccountCodeInput) (authusecase.EmailCodeDispatchResult, error)
	DeleteAccount(ctx context.Context, input authusecase.DeleteAccountInput) error
}

func (h *Handler) SetSecurityUseCase(security SecurityUseCase) {
	h.security = security
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid register request"))
		c.Abort()
		return
	}

	result, err := h.register.Register(c.Request.Context(), authusecase.RegisterInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusCreated, registerResponse{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresIn:   result.ExpiresIn,
		User: userResponse{
			ID:        result.User.ID,
			Username:  result.User.Username,
			Status:    result.User.Status,
			CreatedAt: result.User.CreatedAt,
		},
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid login request"))
		c.Abort()
		return
	}

	result, err := h.login.Login(c.Request.Context(), authusecase.LoginInput{
		Identifier: req.Identifier,
		Username:   req.Username,
		Password:   req.Password,
		RequestIP:  c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, loginResponse{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresIn:   result.ExpiresIn,
		User: userResponse{
			ID:              result.User.ID,
			Username:        result.User.Username,
			Status:          result.User.Status,
			Email:           result.User.Email,
			EmailVerified:   result.User.EmailVerified,
			IsPlatformStaff: result.User.IsPlatformStaff,
			PlatformRole:    result.User.PlatformRole,
			CreatedAt:       result.User.CreatedAt,
		},
	})
}

func (h *Handler) SendRegisterEmailCode(c *gin.Context) {
	var req emailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid email code request"))
		c.Abort()
		return
	}
	result, err := h.security.SendRegisterEmailCode(c.Request.Context(), authusecase.EmailCodeDispatchInput{
		Email:     req.Email,
		RequestIP: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	})
	respondEmailCode(c, result, err)
}

func (h *Handler) RegisterWithEmail(c *gin.Context) {
	var req registerWithEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid email register request"))
		c.Abort()
		return
	}
	result, err := h.security.RegisterWithEmail(c.Request.Context(), authusecase.RegisterWithEmailInput{
		Email:     req.Email,
		Code:      req.Code,
		Username:  req.Username,
		Password:  req.Password,
		RequestIP: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusCreated, toSecurityAuthResponse(result))
}

func (h *Handler) SendLoginEmailCode(c *gin.Context) {
	var req emailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid email code request"))
		c.Abort()
		return
	}
	result, err := h.security.SendLoginEmailCode(c.Request.Context(), authusecase.EmailCodeDispatchInput{
		Email:     req.Email,
		RequestIP: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	})
	respondEmailCode(c, result, err)
}

func (h *Handler) LoginWithEmailCode(c *gin.Context) {
	var req loginWithEmailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid email code login request"))
		c.Abort()
		return
	}
	result, err := h.security.LoginWithEmailCode(c.Request.Context(), authusecase.LoginWithEmailCodeInput{
		Email:     req.Email,
		Code:      req.Code,
		RequestIP: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, toSecurityAuthResponse(result))
}

func (h *Handler) SendPasswordResetEmailCode(c *gin.Context) {
	var req emailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid email code request"))
		c.Abort()
		return
	}
	result, err := h.security.SendPasswordResetEmailCode(c.Request.Context(), authusecase.EmailCodeDispatchInput{
		Email:     req.Email,
		RequestIP: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	})
	respondEmailCode(c, result, err)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req passwordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperr.New(apperr.CodeInvalidArgument, "invalid password reset request"))
		c.Abort()
		return
	}
	result, err := h.security.ResetPassword(c.Request.Context(), authusecase.PasswordResetInput{
		Email:       req.Email,
		Code:        req.Code,
		NewPassword: req.NewPassword,
		RequestIP:   c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
	})
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, passwordUpdatedResponse{Updated: result.Updated})
}

func respondEmailCode(c *gin.Context, result authusecase.EmailCodeDispatchResult, err error) {
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, emailCodeResponse{
		Email:       result.Email,
		Purpose:     result.Purpose,
		ExpiresIn:   result.ExpiresIn,
		ResendAfter: result.ResendAfter,
	})
}

func toSecurityAuthResponse(result authusecase.SecurityAuthResult) loginResponse {
	return loginResponse{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresIn:   result.ExpiresIn,
		User: userResponse{
			ID:              result.User.ID,
			Username:        result.User.Username,
			Status:          result.User.Status,
			Email:           result.User.Email,
			EmailVerified:   result.User.EmailVerified,
			IsPlatformStaff: result.User.IsPlatformStaff,
			PlatformRole:    result.User.PlatformRole,
			CreatedAt:       result.User.CreatedAt,
		},
	}
}

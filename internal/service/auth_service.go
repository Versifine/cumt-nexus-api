package service

import (
	"context"
	"cumt-nexus-api/internal/config"
	"cumt-nexus-api/internal/model"
	"cumt-nexus-api/pkg/jwtx"
	passwordPkg "cumt-nexus-api/pkg/password"
	"errors"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type AuthService struct {
	UserRepo UserRepo
}

func NewAuthService(userRepo UserRepo) *AuthService {
	return &AuthService{
		UserRepo: userRepo,
	}
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type RegisterResponse struct {
	UserID uint64 `json:"user_id"`
}

func (s *AuthService) Register(ctx context.Context, username, password, nickname string) (*RegisterResponse, error) {
	// 先检查用户名是否已存在；未查到记录说明可以继续注册。
	existingUser, err := s.UserRepo.FindByUserName(ctx, username)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			// 用户不存在，可以继续注册。
		default:
			return nil, err
		}
	}
	if existingUser != nil {
		return nil, ErrUsernameExists
	}

	// 密码工具层返回的是底层错误，这里需要翻译成业务错误。
	passwordHash, err := passwordPkg.HashPassword(password)
	if err != nil {
		switch {
		case errors.Is(err, passwordPkg.ErrEmptyPassword), errors.Is(err, passwordPkg.ErrPasswordTooLong):
			return nil, ErrParamInvalid
		default:
			return nil, err
		}
	}

	// 创建用户时仍要兜底唯一索引冲突，避免并发注册绕过前面的查重。
	user := &model.User{
		Username:     username,
		PasswordHash: passwordHash,
		Nickname:     nickname,
	}
	err = s.UserRepo.Create(ctx, user)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrDuplicatedKey), isMySQLDuplicateEntry(err):
			return nil, ErrUsernameExists
		default:
			return nil, err
		}
	}
	return &RegisterResponse{UserID: user.ID}, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	user, err := s.UserRepo.FindByUserName(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	// 在这里生成访问令牌
	if user.Status == 2 {
		return nil, ErrUserBanned
	}
	err = passwordPkg.CheckPassword(user.PasswordHash, password)
	if err != nil {
		switch {
		case errors.Is(err, passwordPkg.ErrPasswordMismatch):
			return nil, ErrPasswordWrong
		case errors.Is(err, passwordPkg.ErrEmptyPassword):
			return nil, ErrParamInvalid
		default:
			return nil, err
		}
	}
	accessToken, err := jwtx.GenerateToken(user.ID, user.Role, config.Conf.JWT.Secret, config.Conf.JWT.ExpireHours)
	if err != nil {
		return nil, err
	}
	expireIn := int64(config.Conf.JWT.ExpireHours * 3600)
	return &LoginResponse{
		AccessToken: accessToken,
		ExpiresIn:   expireIn,
		TokenType:   "Bearer",
	}, nil
}

func isMySQLDuplicateEntry(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}

	return mysqlErr.Number == 1062
}

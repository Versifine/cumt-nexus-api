package service

import "errors"

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrParamInvalid = errors.New("invalid param")
)

func MapError(err error) (code int, msg string) {
	switch {
	case errors.Is(err, ErrParamInvalid):
		return 10001, "参数错误"
	case errors.Is(err, ErrUserNotFound):
		return 20001, "用户不存在"
	case errors.Is(err, ErrUnauthorized):
		return 30001, "未登录"
	case errors.Is(err, ErrForbidden):
		return 30002, "权限不足"
	default:
		return 50000, "系统繁忙,请稍候再试"
	}
}

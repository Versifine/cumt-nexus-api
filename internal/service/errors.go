package service

import "errors"

var (
	ErrParamInvalid         = errors.New("invalid param")
	ErrParamMissing         = errors.New("missing param")
	ErrUserNotFound         = errors.New("user not found")
	ErrPasswordWrong        = errors.New("password wrong")
	ErrUsernameExists       = errors.New("username exists")
	ErrCommunityExists      = errors.New("community exists")
	ErrUserBanned           = errors.New("user banned")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrResourceNotFound     = errors.New("resource not found")
	ErrPostNotFound         = errors.New("post not found")
	ErrCommentNotFound      = errors.New("comment not found")
	ErrCommunityNotFound    = errors.New("community not found")
	ErrParentCommentInvalid = errors.New("parent comment invalid")
)

func MapError(err error) (int, string) {
	switch {
	case errors.Is(err, ErrParamMissing):
		return 10002, "缺少必填参数"
	case errors.Is(err, ErrParamInvalid):
		return 10001, "参数错误"
	case errors.Is(err, ErrUserNotFound):
		return 20001, "用户不存在"
	case errors.Is(err, ErrPasswordWrong):
		return 20002, "密码错误"
	case errors.Is(err, ErrUsernameExists):
		return 20003, "用户名已存在"
	case errors.Is(err, ErrCommunityExists):
		return 20005, "社区名称已存在"
	case errors.Is(err, ErrUserBanned):
		return 20004, "用户被封禁"
	case errors.Is(err, ErrUnauthorized):
		return 30001, "未登录或 token 无效"
	case errors.Is(err, ErrForbidden):
		return 30002, "权限不足"
	case errors.Is(err, ErrPostNotFound):
		return 40002, "帖子不存在"
	case errors.Is(err, ErrCommentNotFound):
		return 40003, "评论不存在"
	case errors.Is(err, ErrCommunityNotFound):
		return 40004, "社区不存在"
	case errors.Is(err, ErrParentCommentInvalid):
		return 40005, "父评论不合法"
	case errors.Is(err, ErrResourceNotFound):
		return 40001, "资源不存在"

	default:
		return 50000, "系统繁忙，请稍后再试"
	}
}

package apperr

import (
	"errors"
)

type Code string

const (
	CodeInvalidArgument Code = "invalid_argument"
	CodeUnauthenticated Code = "unauthenticated"
	CodeForbidden       Code = "forbidden"
	CodeNotFound        Code = "not_found"
	CodeConflict        Code = "conflict"
	CodeInternal        Code = "internal"
)

type Error struct {
	code    Code
	message string
}

func (e *Error) Error() string {
	return e.message
}

func (e *Error) Message() string {
	return e.message
}

func (e *Error) Code() Code {
	return e.code
}

func New(code Code, message string) error {
	return &Error{
		code:    code,
		message: message,
	}
}
func IsCode(err error, code Code) bool {
	var appErr *Error
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Code() == code
}

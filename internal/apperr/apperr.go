package apperr

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

func (e *Error) Code() Code {
	return e.code
}

func (e *Error) New(code Code, message string) error {
	return &Error{
		code:    code,
		message: message,
	}
}

package utils

import "net/http"

type Error struct {
	error
	Mess string
	Code int
}

func (e *Error) With(err error) *Error {
	e.error = err
	return e
}

const (
	StatusOK               = 2000
	ErrorInternalCode      = 4000
	ErrorObjectExist       = 4001
	ErrorLoginFail         = 4002
	ErrorInvalidCredential = 4003
	ErrorBodyRequited      = 4004
	ErrorBadRequest        = 4010
	ErrorUnauthorized      = 4011
	ErrorNotFound          = 4040
	ErrorForbidden         = 4030
	ErrorSendMailFailed    = 5001
)

func (e *Error) HttpStatus() int {
	switch e.Code {
	case ErrorInternalCode:
		return http.StatusInternalServerError
	case ErrorBadRequest, ErrorObjectExist, ErrorLoginFail, ErrorInvalidCredential, ErrorBodyRequited:
		return http.StatusBadRequest
	case ErrorNotFound:
		return http.StatusNotFound
	case ErrorForbidden:
		return http.StatusForbidden
	case ErrorSendMailFailed:
		return http.StatusBadGateway
	case ErrorUnauthorized:
		return http.StatusUnauthorized
	default:
		return http.StatusOK
	}
}

var SendMailFailed = &Error{
	Mess: "send mail failed",
	Code: ErrorSendMailFailed,
}

var ForbiddenError = &Error{
	Mess: "not allowed",
	Code: ErrorForbidden,
}

var NotFoundError = &Error{
	Mess: "not found",
	Code: ErrorNotFound,
}

var InternalError = Error{
	Mess: "Something went wrong. please contact admin",
	Code: ErrorInternalCode,
}

var LoginFail = &Error{
	Mess: "Your username or password is incorrect",
	Code: ErrorLoginFail,
}

var InvalidCredential = &Error{
	Mess: "your credential is invalid",
	Code: ErrorLoginFail,
}

func (e *Error) Error() string {
	if e.Mess != "" {
		return e.Mess
	}
	return e.error.Error()
}

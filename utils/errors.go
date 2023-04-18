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
	ErrorloginFail         = 4002
	ErrorInvalidCredential = 4003
	ErrorBodyRequited      = 4004
	ErrorBadRequest        = 4010
	ErrorNotFound          = 4040
	ErrorForbidden         = 4030
	ErrorSendMailFailed    = 5001
)

func (e *Error) HttpStatus() int {
	switch e.Code {
	case ErrorInternalCode:
		return http.StatusInternalServerError
	case ErrorBadRequest:
		return http.StatusBadRequest
	case ErrorNotFound:
		return http.StatusNotFound
	case ErrorForbidden:
		return http.StatusForbidden
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
	Code: ErrorloginFail,
}

var InvalidCredential = &Error{
	Mess: "your credential is invalid",
	Code: ErrorloginFail,
}

func (e *Error) Error() string {
	if e.Mess != "" {
		return e.Mess
	}
	return e.error.Error()
}

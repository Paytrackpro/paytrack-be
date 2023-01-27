package utils

type Error struct {
	error
	Mess string
	Code int
}

const (
	StatusOK               = 2000
	ErrorInternalCode      = 4000
	ErrorObjectExist       = 4001
	ErrorloginFail         = 4002
	ErrorInvalidCredential = 4003
	ErrorBodyRequited      = 4004
)

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

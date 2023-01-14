package utils

type Error struct {
	error
	Mess string
	Code int
}

const (
	ErrorInternalCode      = 400
	ErrorObjectExist       = 401
	ErrorloginFail         = 402
	ErrorInvalidCredential = 403
	ErrorBodyRequited      = 403
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

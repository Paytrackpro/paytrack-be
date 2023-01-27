package utils

import (
	"encoding/json"
	"net/http"
)

type response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
}

func NewError(mess string, code int) *Error {
	return &Error{
		Mess: mess,
		Code: code,
	}
}
func ResponseOK(w http.ResponseWriter, err error, data interface{}) {
	Response(w, http.StatusOK, err, data)
}

func Response(w http.ResponseWriter, httpStatus int, err error, data interface{}) {
	w.WriteHeader(httpStatus)
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	res := response{
		Success: err == nil,
		Code:    StatusOK,
		Message: "ok",
		Data:    data,
	}

	//TODO: Save error to files if need in here

	if err != nil {
		switch er := err.(type) {
		case *Error:
			res.Code = er.Code
			res.Message = er.Error()
		default:
			res.Message = err.Error()
			res.Code = ErrorInternalCode
		}
	}
	enc.Encode(res)
}

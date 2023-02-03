package utils

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/gorilla/schema"
)

var decoder = schema.NewDecoder()

type response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
}

func NewError(err error, code int) *Error {
	return &Error{
		error: err,
		Code:  code,
	}
}
func ResponseOK(w http.ResponseWriter, data interface{}, errs ...*Error) {
	if len(errs) > 0 {
		Response(w, http.StatusOK, errs[0], data)
		return
	}
	Response(w, http.StatusOK, nil, data)
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

func IsEmpty(x interface{}) bool {
	switch value := x.(type) {
	case string:
		return value == ""
	case int32:
		return value == 0
	case int:
		return value == 0
	case uint32:
		return value == 0
	case int64:
		return value == 0
	case float64:
		return value == 0
	default:
		return true
	}
}

func DecodeQuery(object interface{}, query url.Values) error {
	err := decoder.Decode(object, query)
	if err != nil {
		return err
	}

	return nil
}

func SetValue[T any](source *T, value T) {
	if !IsEmpty(value) && source != &value {
		*source = value
	}
}

package utils

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
	"unicode/utf8"

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
	if len(errs) > 0 && errs[0] != nil {
		Response(w, http.StatusOK, errs[0], data)
		return
	}
	Response(w, http.StatusOK, nil, data)
}

func Response(w http.ResponseWriter, httpStatus int, err error, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	res := response{
		Success: err == nil,
		Code:    StatusOK,
		Message: "ok",
		Data:    data,
	}

	if err != nil {
		switch er := err.(type) {
		case *Error:
			res.Code = er.Code
			res.Message = er.Error()
			w.WriteHeader(er.HttpStatus())
		default:
			res.Message = err.Error()
			res.Code = ErrorInternalCode
			w.WriteHeader(httpStatus)
		}
	} else {
		w.WriteHeader(httpStatus)
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
	case uint64:
		return value == 0
	case int64:
		return value == 0
	case float64:
		return value == 0
	case bool:
		return false
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

func ImageToBase64(img image.Image) (string, error) {
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, img, nil)
	if err != nil {
		return "", err
	}

	qrBytesString := buf.Bytes()
	imgBase64Str := "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrBytesString)

	return imgBase64Str, nil
}

var imagePath = getBinPath() + "\\upload\\images"

func getBinPath() string {
	e, err := os.Executable()
	if err != nil {
		panic(err)
	}
	path := path.Dir(e)
	return path
}

func GetImagePath() string {
	return imagePath
}

func ConvertImageToBase64(fileName string) string {
	imgFile, err := os.Open(imagePath + "\\" + fileName) //Image file

	if err != nil {
		fmt.Println(err)
		return ""
	}

	defer imgFile.Close()

	// create a new buffer base on file size
	fInfo, _ := imgFile.Stat()
	var size int64 = fInfo.Size()
	buf := make([]byte, size)

	// read file content into buffer
	fReader := bufio.NewReader(imgFile)
	fReader.Read(buf)

	// convert the buffer bytes to base64 string - use buf.Bytes() for new image
	imgBase64Str := base64.StdEncoding.EncodeToString(buf)
	return imgBase64Str
}

func ContainsUint64(inputArr []uint64, value uint64) bool {
	if inputArr == nil {
		return false
	}
	for _, v := range inputArr {
		if v == value {
			return true
		}
	}
	return false
}

// convert time to string, format yyyy-mm-dd hh:MM:ss
func TimeToStringWithoutTimeZone(date time.Time) string {
	return date.Format("2006-01-02 15:04:05")
}

func GetUserDisplayName(userName string, displayName string) string {
	if IsEmpty(displayName) {
		return userName
	}
	return displayName
}

// handler date format (YYYY/MM/DD)
func HandlerDateFormat(date string) string {
	dateArr := strings.Split(date, "/")
	if len(dateArr) < 3 {
		return date
	}

	if utf8.RuneCountInString(dateArr[1]) == 1 {
		dateArr[1] = fmt.Sprintf("0%s", dateArr[1])
	}

	if utf8.RuneCountInString(dateArr[2]) == 1 {
		dateArr[2] = fmt.Sprintf("0%s", dateArr[2])
	}
	return strings.Join(dateArr, "/")
}

func GetSecondDurationFromStartEnd(startTime, endTime time.Time) uint64 {
	return uint64(math.Floor(endTime.Sub(startTime).Seconds()))
}

func CatchObject(from interface{}, to interface{}) error {
	jsonBytes, err := json.Marshal(from)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonBytes, &to)
	if err != nil {
		return err
	}
	return nil
}

package response

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

const (
	StatusOK    = "OK"
	StatusError = "Error"
)

func OK() Response {
	return Response{
		Status: StatusOK,
	}
}

func Error(msg string) Response {
	return Response{
		Status: StatusError,
		Error:  msg,
	}
}

func ValidationError(err validator.ValidationErrors) Response {

	var errorMsgs []string

	for _, err := range err {
		switch err.ActualTag() {
		case "required":
			errorMsgs = append(errorMsgs, fmt.Sprintf("поле %s является обязательным", err.Field()))
		case "url":
			errorMsgs = append(errorMsgs, fmt.Sprintf("поле %s должно быть валидным URL", err.Field()))
		default:
			errorMsgs = append(errorMsgs, fmt.Sprintf("поле %s является невалидным", err.Field()))
		}
	}

	return Response{
		Status: StatusError,
		Error:  strings.Join(errorMsgs, ", "),
	}
}

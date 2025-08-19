package apperror

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	ErrNotFound     = NewAppError("not found")
	ErrUnauthorized = NewAppError("unauthorized")
	ErrForbidden    = NewAppError("forbidden")
	ErrDecodeBody   = NewAppError("failed to decode request body")
)

type AppError struct {
	Message string `json:"message"`
}

func NewAppError(message string) *AppError {
	return &AppError{
		Message: message,
	}
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Marshal() []byte {
	marshal, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	return marshal
}

func NewValidationErr(errs validator.ValidationErrors) *AppError {
	var errMsgs []string

	for _, err := range errs {
		switch err.ActualTag() {
		case "required":
			errMsgs = append(errMsgs, fmt.Sprintf("field %s is a required field", err.Field()))
		case "email":
			errMsgs = append(errMsgs, fmt.Sprintf("field %s is not a valid email", err.Field()))
		case "min":
			errMsgs = append(errMsgs, fmt.Sprintf("the minimum length of the %s field is %s characters", err.Field(), err.Param()))
		default:
			errMsgs = append(errMsgs, fmt.Sprintf("field %s is not valid", err.Field()))
		}
	}

	return NewAppError(strings.Join(errMsgs, ", "))
}

func internalError() *AppError {
	return NewAppError("internal error")
}

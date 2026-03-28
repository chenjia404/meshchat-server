package apperrors

import (
	"errors"
	"net/http"
)

// AppError carries a stable HTTP status and message for API responses.
type AppError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func (e *AppError) Error() string {
	return e.Message
}

func New(status int, code, message string) error {
	return &AppError{
		StatusCode: status,
		Code:       code,
		Message:    message,
	}
}

func Is(err error, code string) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Code == code
}

func HTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}

func Public(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return &AppError{
		StatusCode: http.StatusInternalServerError,
		Code:       "internal_error",
		Message:    "internal server error",
	}
}

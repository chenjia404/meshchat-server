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
	// Detail is set only when ExposeInternalErrorDetail is enabled and the error was not already an AppError.
	Detail string `json:"detail,omitempty"`
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
	return PublicWithDetail(err, false)
}

// PublicWithDetail mirrors [Public] but may attach err.Error() as Detail when exposeDetail is true
// and err is not an *AppError (e.g. database/driver errors), to aid debugging 5xx responses.
func PublicWithDetail(err error, exposeDetail bool) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		out := *appErr
		return &out
	}
	e := &AppError{
		StatusCode: http.StatusInternalServerError,
		Code:       "internal_error",
		Message:    "internal server error",
	}
	if exposeDetail && err != nil {
		e.Detail = err.Error()
	}
	return e
}

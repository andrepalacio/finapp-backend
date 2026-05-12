package apperror

import "fmt"

type AppError struct {
	Code       string
	Message    string
	StatusCode int
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Is allows errors.Is to match by Code, so Wrap(base, cause) still matches base.
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// WithMessage returns a new AppError with the same Code/StatusCode but a custom message.
func WithMessage(base *AppError, msg string) *AppError {
	return &AppError{
		Code:       base.Code,
		Message:    msg,
		StatusCode: base.StatusCode,
	}
}

// Wrap returns a new AppError with the same Code/StatusCode/Message but with a wrapped cause.
func Wrap(base *AppError, err error) *AppError {
	return &AppError{
		Code:       base.Code,
		Message:    base.Message,
		StatusCode: base.StatusCode,
		Err:        err,
	}
}

var (
	ErrNotFound     = &AppError{Code: "NOT_FOUND",      StatusCode: 404, Message: "resource not found"}
	ErrUnauthorized = &AppError{Code: "UNAUTHORIZED",   StatusCode: 401, Message: "unauthorized"}
	ErrForbidden    = &AppError{Code: "FORBIDDEN",      StatusCode: 403, Message: "forbidden"}
	ErrInvalidInput = &AppError{Code: "INVALID_INPUT",  StatusCode: 400, Message: "invalid input"}
	ErrConflict     = &AppError{Code: "CONFLICT",       StatusCode: 409, Message: "already exists"}
	ErrInternal     = &AppError{Code: "INTERNAL_ERROR", StatusCode: 500, Message: "internal server error"}
)

package apperror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		want string
	}{
		{
			name: "without wrapped err",
			err:  &AppError{Message: "invalid input"},
			want: "invalid input",
		},
		{
			name: "with wrapped err",
			err:  &AppError{Message: "internal server error", Err: errors.New("db timeout")},
			want: "internal server error: db timeout",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("boom")
	err := &AppError{Message: "x", Err: cause}
	assert.Equal(t, cause, err.Unwrap())

	err2 := &AppError{Message: "x"}
	assert.Nil(t, err2.Unwrap())
}

func TestAppError_Is(t *testing.T) {
	a := &AppError{Code: "NOT_FOUND", StatusCode: 404, Message: "resource not found"}
	b := &AppError{Code: "NOT_FOUND", StatusCode: 404, Message: "different message"}
	c := &AppError{Code: "FORBIDDEN", StatusCode: 403, Message: "forbidden"}

	assert.True(t, errors.Is(a, b))
	assert.False(t, errors.Is(a, c))
	assert.False(t, a.Is(errors.New("plain error")))
}

func TestWithMessage(t *testing.T) {
	got := WithMessage(ErrInvalidInput, "custom message")
	assert.Equal(t, ErrInvalidInput.Code, got.Code)
	assert.Equal(t, ErrInvalidInput.StatusCode, got.StatusCode)
	assert.Equal(t, "custom message", got.Message)
	assert.Nil(t, got.Err)
	assert.True(t, errors.Is(got, ErrInvalidInput))
}

func TestWrap(t *testing.T) {
	cause := errors.New("connection refused")
	got := Wrap(ErrInternal, cause)
	assert.Equal(t, ErrInternal.Code, got.Code)
	assert.Equal(t, ErrInternal.StatusCode, got.StatusCode)
	assert.Equal(t, ErrInternal.Message, got.Message)
	assert.Equal(t, cause, got.Err)
	assert.True(t, errors.Is(got, ErrInternal))
	assert.Equal(t, fmt.Sprintf("%s: %v", ErrInternal.Message, cause), got.Error())
}

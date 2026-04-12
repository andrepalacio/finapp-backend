package response

import (
	"errors"
	"net/http"

	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/gin-gonic/gin"
)

type errorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

func BadRequest(c *gin.Context, code, message string) {
	c.JSON(http.StatusBadRequest, errorBody{Error: message, Code: code})
}

func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, errorBody{Error: message, Code: code})
}

func HandleError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.StatusCode, errorBody{Error: appErr.Message, Code: appErr.Code})
		return
	}
	c.JSON(http.StatusInternalServerError, errorBody{
		Error: apperror.ErrInternal.Message,
		Code:  apperror.ErrInternal.Code,
	})
}

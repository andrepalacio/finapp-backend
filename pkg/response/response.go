package response

import "github.com/gin-gonic/gin"

type errorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func OK(c *gin.Context, data any) {
	c.JSON(200, data)
}

func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, errorBody{Error: message, Code: code})
}

package response

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type body struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func TestOK(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	OK(c, gin.H{"foo": "bar"})
	assert.Equal(t, 200, w.Code)
	assert.JSONEq(t, `{"foo":"bar"}`, w.Body.String())
}

func TestCreated(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Created(c, gin.H{"id": "1"})
	assert.Equal(t, 201, w.Code)
}

func TestNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	NoContent(c)
	assert.Equal(t, 204, c.Writer.Status())
	assert.Empty(t, w.Body.String())
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	BadRequest(c, "INVALID_INPUT", "bad input")
	assert.Equal(t, 400, w.Code)
	var b body
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))
	assert.Equal(t, "bad input", b.Error)
	assert.Equal(t, "INVALID_INPUT", b.Code)
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Error(c, 418, "TEAPOT", "i am a teapot")
	assert.Equal(t, 418, w.Code)
	var b body
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))
	assert.Equal(t, "i am a teapot", b.Error)
	assert.Equal(t, "TEAPOT", b.Code)
}

func TestHandleError(t *testing.T) {
	t.Run("app error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		HandleError(c, apperror.ErrForbidden)
		assert.Equal(t, apperror.ErrForbidden.StatusCode, w.Code)
		var b body
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))
		assert.Equal(t, apperror.ErrForbidden.Message, b.Error)
		assert.Equal(t, apperror.ErrForbidden.Code, b.Code)
	})

	t.Run("plain error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		HandleError(c, errors.New("something broke"))
		assert.Equal(t, 500, w.Code)
		var b body
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))
		assert.Equal(t, apperror.ErrInternal.Message, b.Error)
		assert.Equal(t, apperror.ErrInternal.Code, b.Code)
	})
}

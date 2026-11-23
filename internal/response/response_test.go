package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func run(t *testing.T, h gin.HandlerFunc) (*httptest.ResponseRecorder, map[string]any) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", h)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return w, body
}

func TestSuccess(t *testing.T) {
	w, b := run(t, func(c *gin.Context) { Success(c, 200, "ok", gin.H{"a": 1}) })
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "success", b["status"])
	assert.Equal(t, "ok", b["message"])
	assert.NotNil(t, b["data"])
}

func TestFail(t *testing.T) {
	w, b := run(t, func(c *gin.Context) { Fail(c, 404, "nope") })
	assert.Equal(t, 404, w.Code)
	assert.Equal(t, "fail", b["status"])
	assert.NotContains(t, b, "data")
}

func TestError(t *testing.T) {
	w, b := run(t, func(c *gin.Context) { Error(c, 500, "boom") })
	assert.Equal(t, 500, w.Code)
	assert.Equal(t, "error", b["status"])
}

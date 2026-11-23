package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

const secret = "test-secret"

func do(r *gin.Engine, method, path, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func engine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/me", Auth(secret), func(c *gin.Context) {
		c.JSON(200, gin.H{"id": c.GetUint(CtxUserID), "role": c.GetString(CtxRole)})
	})
	r.GET("/admin", Auth(secret), RequireRole("admin"), func(c *gin.Context) { c.Status(200) })
	return r
}

func TestCORS(t *testing.T) {
	r := engine()
	w := do(r, http.MethodOptions, "/me", "")
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	w = do(r, http.MethodGet, "/me", "")
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestAuth(t *testing.T) {
	r := engine()
	assert.Equal(t, 401, do(r, "GET", "/me", "").Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer sampah").Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer "+GenerateToken("secret-lain", 1, "member", time.Hour)).Code)
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer "+GenerateToken(secret, 1, "member", -time.Hour)).Code)

	// algoritma selain HS256 ditolak
	other, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{UserID: 1}).SignedString([]byte(secret))
	assert.Equal(t, 401, do(r, "GET", "/me", "Bearer "+other).Code)

	w := do(r, "GET", "/me", "Bearer "+GenerateToken(secret, 7, "member", time.Hour))
	assert.Equal(t, 200, w.Code)
	assert.JSONEq(t, `{"id":7,"role":"member"}`, w.Body.String())
}

func TestRequireRole(t *testing.T) {
	r := engine()
	assert.Equal(t, 403, do(r, "GET", "/admin", "Bearer "+GenerateToken(secret, 1, "member", time.Hour)).Code)
	assert.Equal(t, 200, do(r, "GET", "/admin", "Bearer "+GenerateToken(secret, 1, "admin", time.Hour)).Code)
}

func TestRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", RateLimiter(2, 50*time.Millisecond), func(c *gin.Context) { c.Status(200) })

	assert.Equal(t, 200, do(r, "GET", "/", "").Code)
	assert.Equal(t, 200, do(r, "GET", "/", "").Code)
	assert.Equal(t, 429, do(r, "GET", "/", "").Code)

	time.Sleep(80 * time.Millisecond) // jendela habis -> reset
	assert.Equal(t, 200, do(r, "GET", "/", "").Code)
}

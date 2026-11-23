package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"app/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	CtxUserID = "user_id"
	CtxRole   = "role"
)

// Claims adalah payload JWT.
type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken menerbitkan JWT HS256. HMAC dengan kunci []byte tidak pernah gagal menandatangani.
func GenerateToken(secret string, userID uint, role string, ttl time.Duration) string {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	signed, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return signed
}

// CORS mengizinkan akses lintas origin (frontend dan backend dideploy terpisah).
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Auth memvalidasi Bearer token JWT dan menyimpan user_id & role ke context.
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Fail(c, http.StatusUnauthorized, "Token autentikasi tidak ditemukan")
			c.Abort()
			return
		}
		token, err := jwt.ParseWithClaims(
			strings.TrimPrefix(header, "Bearer "),
			&Claims{},
			func(*jwt.Token) (any, error) { return []byte(secret), nil },
			jwt.WithValidMethods([]string{"HS256"}),
		)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "Token tidak valid atau sudah kedaluwarsa")
			c.Abort()
			return
		}
		claims := token.Claims.(*Claims)
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxRole, claims.Role)
		c.Next()
	}
}

// RequireRole hanya meloloskan pengguna dengan role tertentu.
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString(CtxRole) != role {
			response.Fail(c, http.StatusForbidden, "Akses ditolak: hak akses tidak mencukupi")
			c.Abort()
			return
		}
		c.Next()
	}
}

type bucket struct {
	count int
	reset time.Time
}

// RateLimiter membatasi jumlah request per IP dalam satu jendela waktu.
func RateLimiter(max int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	hits := map[string]bucket{}
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		b, ok := hits[ip]
		if !ok || now.After(b.reset) {
			b = bucket{reset: now.Add(window)}
		}
		b.count++
		hits[ip] = b
		blocked := b.count > max
		mu.Unlock()

		if blocked {
			response.Fail(c, http.StatusTooManyRequests, "Terlalu banyak permintaan, coba lagi nanti")
			c.Abort()
			return
		}
		c.Next()
	}
}

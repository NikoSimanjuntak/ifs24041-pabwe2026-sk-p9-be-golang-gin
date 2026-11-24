package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"app/internal/database"
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const secret = "unit-test-secret"

func setup(t *testing.T) (*gorm.DB, *gin.Engine, *Service) {
	gin.SetMode(gin.TestMode)
	db := database.OpenSQLiteMemory()
	require.NoError(t, database.Migrate(db))
	svc := NewService(NewRepository(db), secret, time.Hour)
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	a := r.Group("/", middleware.Auth(secret))
	a.POST("/logout", h.Logout)
	a.GET("/profile", h.Profile)
	a.PUT("/profile", h.UpdateProfile)
	return db, r, svc
}

func do(r *gin.Engine, method, path string, body any, token string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	switch v := body.(type) {
	case string:
		buf.WriteString(v)
	case nil:
	default:
		_ = json.NewEncoder(&buf).Encode(v)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func closeDB(db *gorm.DB) {
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
}

func failOn(db *gorm.DB, op string) {
	cb := func(tx *gorm.DB) { _ = tx.AddError(errors.New("boom")) }
	switch op {
	case "create":
		_ = db.Callback().Create().Before("gorm:create").Register("fail", cb)
	case "update":
		_ = db.Callback().Update().Before("gorm:update").Register("fail", cb)
	}
}

func failQueryNth(db *gorm.DB, nth int) {
	n := 0
	_ = db.Callback().Query().Before("gorm:query").Register("failnth", func(tx *gorm.DB) {
		n++
		if n == nth {
			_ = tx.AddError(errors.New("boom"))
		}
	})
}

func registerAndLogin(t *testing.T, r *gin.Engine) string {
	w := do(r, "POST", "/register", RegisterRequest{Username: "budi", Email: "budi@mail.com", Password: "rahasia1"}, "")
	require.Equal(t, 201, w.Code)
	var reg struct {
		Data AuthResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &reg))
	require.Equal(t, 3, len(strings.Split(reg.Data.Token, "."))) // JWT
	w = do(r, "POST", "/login", LoginRequest{Username: "budi", Password: "rahasia1"}, "")
	require.Equal(t, 200, w.Code)
	var body struct {
		Data AuthResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotEmpty(t, body.Data.Token)
	return body.Data.Token
}

func TestRegisterLoginProfileFlow(t *testing.T) {
	_, r, _ := setup(t)
	token := registerAndLogin(t, r)

	// login dengan email juga berhasil
	assert.Equal(t, 200, do(r, "POST", "/login", LoginRequest{Username: "budi@mail.com", Password: "rahasia1"}, "").Code)

	w := do(r, "GET", "/profile", nil, token)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"username":"budi"`)
	assert.NotContains(t, w.Body.String(), "rahasia1")

	assert.Equal(t, 200, do(r, "POST", "/logout", nil, token).Code)

	w = do(r, "PUT", "/profile", UpdateProfileRequest{Username: "budi2", Email: "b2@mail.com", Password: "barubaru"}, token)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"username":"budi2"`)
	assert.Equal(t, 200, do(r, "POST", "/login", LoginRequest{Username: "budi2", Password: "barubaru"}, "").Code)

	// update kosong tetap valid
	assert.Equal(t, 200, do(r, "PUT", "/profile", UpdateProfileRequest{}, token).Code)
}

func TestRegisterErrors(t *testing.T) {
	db, r, _ := setup(t)
	registerAndLogin(t, r)

	assert.Equal(t, 400, do(r, "POST", "/register", "{bukan json", "").Code)
	assert.Equal(t, 400, do(r, "POST", "/register", RegisterRequest{Username: "x", Email: "salah", Password: "1"}, "").Code)
	assert.Equal(t, 400, do(r, "POST", "/register", RegisterRequest{Username: "budi", Email: "lain@mail.com", Password: "rahasia1"}, "").Code)
	assert.Equal(t, 400, do(r, "POST", "/register", RegisterRequest{Username: "lain", Email: "budi@mail.com", Password: "rahasia1"}, "").Code)

	failOn(db, "create")
	assert.Equal(t, 500, do(r, "POST", "/register", RegisterRequest{Username: "baru", Email: "baru@mail.com", Password: "rahasia1"}, "").Code)
}

func TestRegisterTakenQueryError(t *testing.T) {
	db, r, _ := setup(t)
	closeDB(db)
	assert.Equal(t, 500, do(r, "POST", "/register", RegisterRequest{Username: "baru", Email: "baru@mail.com", Password: "rahasia1"}, "").Code)
}

func TestRegisterHashErrorAtService(t *testing.T) {
	_, _, svc := setup(t)
	_, err := svc.Register(RegisterRequest{Username: "panjang", Email: "p@mail.com", Password: strings.Repeat("x", 100)})
	assert.Error(t, err)
}

func TestLoginErrors(t *testing.T) {
	db, r, _ := setup(t)
	registerAndLogin(t, r)

	assert.Equal(t, 400, do(r, "POST", "/login", "{}", "").Code)
	assert.Equal(t, 401, do(r, "POST", "/login", LoginRequest{Username: "budi", Password: "salah"}, "").Code)
	assert.Equal(t, 401, do(r, "POST", "/login", LoginRequest{Username: "tidakada", Password: "x"}, "").Code)

	closeDB(db)
	assert.Equal(t, 500, do(r, "POST", "/login", LoginRequest{Username: "budi", Password: "rahasia1"}, "").Code)
}

func TestProfileErrors(t *testing.T) {
	db, r, svc := setup(t)
	token := registerAndLogin(t, r)

	assert.Equal(t, 401, do(r, "GET", "/profile", nil, "").Code)
	assert.Equal(t, 400, do(r, "PUT", "/profile", "{bukan json", token).Code)

	// token valid tetapi pengguna tidak ada
	ghost := middleware.GenerateToken(secret, 999, "member", time.Hour)
	assert.Equal(t, 404, do(r, "GET", "/profile", nil, ghost).Code)
	assert.Equal(t, 404, do(r, "PUT", "/profile", UpdateProfileRequest{Username: "zzz"}, ghost).Code)

	// username milik pengguna lain
	require.Equal(t, 201, do(r, "POST", "/register", RegisterRequest{Username: "sari", Email: "sari@mail.com", Password: "rahasia1"}, "").Code)
	assert.Equal(t, 400, do(r, "PUT", "/profile", UpdateProfileRequest{Username: "sari"}, token).Code)

	// password terlalu panjang untuk bcrypt (dilewatkan langsung ke service)
	_, err := svc.UpdateProfile(1, UpdateProfileRequest{Password: strings.Repeat("x", 100)})
	assert.Error(t, err)

	// kegagalan simpan
	failOn(db, "update")
	assert.Equal(t, 500, do(r, "PUT", "/profile", UpdateProfileRequest{Email: "baru@mail.com"}, token).Code)
}

func TestProfileDatabaseErrors(t *testing.T) {
	db, r, _ := setup(t)
	token := registerAndLogin(t, r)

	// query ke-2 pada UpdateProfile adalah pemeriksaan Taken
	failQueryNth(db, 2)
	assert.Equal(t, 500, do(r, "PUT", "/profile", UpdateProfileRequest{Username: "x123"}, token).Code)

	closeDB(db)
	assert.Equal(t, 500, do(r, "GET", "/profile", nil, token).Code)
	assert.Equal(t, 500, do(r, "PUT", "/profile", UpdateProfileRequest{Username: "x123"}, token).Code)
}

func TestFailDefaultMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) { fail(c, errors.New("lain")) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, 500, w.Code)
}

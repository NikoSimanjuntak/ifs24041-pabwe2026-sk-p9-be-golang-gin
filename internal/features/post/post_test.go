package post

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"app/internal/database"
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setup(t *testing.T) (*gorm.DB, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	db := database.OpenSQLiteMemory()
	require.NoError(t, database.Migrate(db))
	require.NoError(t, db.Create(&database.User{Username: "ani", Email: "ani@m.com", Password: "x", Role: "member"}).Error)
	require.NoError(t, db.Create(&database.User{Username: "boss", Email: "boss@m.com", Password: "x", Role: "admin"}).Error)

	h := NewHandler(NewService(NewRepository(db)))
	stub := func(c *gin.Context) {
		id, _ := strconv.Atoi(c.GetHeader("X-User"))
		c.Set(middleware.CtxUserID, uint(id))
	}
	r := gin.New()
	r.GET("/public/posts", h.ListPublic)
	r.GET("/public/posts/:slug", h.GetPublic)
	p := r.Group("/posts", stub)
	p.GET("", h.ListMine)
	p.POST("", h.Create)
	p.GET("/stats", h.Stats)
	p.GET("/:id", h.GetMine)
	p.PUT("/:id", h.Update)
	p.PATCH("/:id/toggle-publish", h.TogglePublish)
	p.DELETE("/:id", h.Delete)
	a := r.Group("/admin")
	a.GET("/users", h.AdminUsers)
	a.GET("/posts", h.AdminPosts)
	a.GET("/users/:id/posts", h.AdminUserPosts)
	return db, r
}

func do(r *gin.Engine, method, path string, body any, user int) *httptest.ResponseRecorder {
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
	req.Header.Set("X-User", strconv.Itoa(user))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func data(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	var b struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))
	return b.Data
}

func list(t *testing.T, w *httptest.ResponseRecorder) []any {
	var b struct {
		Data []any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))
	return b.Data
}

func bptr(b bool) *bool { return &b }

func closeDB(db *gorm.DB) {
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
}

func failOp(db *gorm.DB, op string) {
	cb := func(tx *gorm.DB) { _ = tx.AddError(errors.New("boom")) }
	switch op {
	case "create":
		_ = db.Callback().Create().Before("gorm:create").Register("fail", cb)
	case "update":
		_ = db.Callback().Update().Before("gorm:update").Register("fail", cb)
	case "delete":
		_ = db.Callback().Delete().Before("gorm:delete").Register("fail", cb)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Halo Dunia!":          "halo-dunia",
		"  Golang  &  Gin  ":   "golang-gin",
		"Belajar 100% Go":      "belajar-100-go",
		"---":                  "post",
		"":                     "post",
		"日本語":                  "post",
		"Sudah-ada-tanda-hubung": "sudah-ada-tanda-hubung",
	}
	for in, want := range cases {
		assert.Equal(t, want, Slugify(in), in)
	}
}

func TestReadingTime(t *testing.T) {
	assert.Equal(t, 1, ReadingTime(""))
	assert.Equal(t, 1, ReadingTime(strings.Repeat("kata ", 200)))
	assert.Equal(t, 2, ReadingTime(strings.Repeat("kata ", 201)))
	assert.Equal(t, 3, ReadingTime(strings.Repeat("kata ", 600)))
}

func TestCreate(t *testing.T) {
	_, r := setup(t)
	body := PostRequest{Title: "Halo Dunia!", Category: "Teknologi", Content: "isi artikel"}

	w := do(r, "POST", "/posts", body, 1)
	require.Equal(t, 201, w.Code)
	d := data(t, w)
	assert.Equal(t, "halo-dunia", d["slug"])
	assert.Equal(t, false, d["is_published"])
	assert.EqualValues(t, 1, d["reading_time"])

	w = do(r, "POST", "/posts", body, 1)
	assert.Equal(t, "halo-dunia-2", data(t, w)["slug"])

	body.IsPublished = bptr(true)
	body.Category = ""
	w = do(r, "POST", "/posts", body, 1)
	d = data(t, w)
	assert.Equal(t, "Umum", d["category"])
	assert.NotNil(t, d["published_at"])
	assert.Equal(t, "halo-dunia-3", d["slug"])

	assert.Equal(t, 400, do(r, "POST", "/posts", PostRequest{Title: "x", Content: "y", Category: "Olahraga"}, 1).Code)
	assert.Equal(t, 400, do(r, "POST", "/posts", PostRequest{Content: "y"}, 1).Code)
	assert.Equal(t, 400, do(r, "POST", "/posts", "{bukan json", 1).Code)
}

func TestPublicEndpoints(t *testing.T) {
	_, r := setup(t)
	do(r, "POST", "/posts", PostRequest{Title: "Golang Gin", Category: "Teknologi", Content: "belajar backend", IsPublished: bptr(true)}, 1)
	do(r, "POST", "/posts", PostRequest{Title: "Startup Baru", Category: "Bisnis", Excerpt: "modal ventura", Content: "isi", IsPublished: bptr(true)}, 1)
	do(r, "POST", "/posts", PostRequest{Title: "Draf Rahasia", Content: "isi"}, 1)

	assert.Len(t, list(t, do(r, "GET", "/public/posts", nil, 0)), 2)
	assert.NotContains(t, do(r, "GET", "/public/posts", nil, 0).Body.String(), "user_id")
	assert.Len(t, list(t, do(r, "GET", "/public/posts?q=GOLANG", nil, 0)), 1)
	assert.Len(t, list(t, do(r, "GET", "/public/posts?q=ventura", nil, 0)), 1)
	assert.Len(t, list(t, do(r, "GET", "/public/posts?category=Bisnis", nil, 0)), 1)
	assert.Len(t, list(t, do(r, "GET", "/public/posts?q=zzz", nil, 0)), 0)
	assert.Contains(t, do(r, "GET", "/public/posts?q=zzz", nil, 0).Body.String(), `"data":[]`)

	w := do(r, "GET", "/public/posts/golang-gin", nil, 0)
	require.Equal(t, 200, w.Code)
	author := data(t, w)["author"].(map[string]any)
	assert.Equal(t, "ani", author["username"])
	assert.Equal(t, "ani", data(t, w)["author_name"])
	assert.NotContains(t, w.Body.String(), "ani@m.com")
	assert.NotContains(t, w.Body.String(), "user_id")

	assert.Equal(t, 404, do(r, "GET", "/public/posts/draf-rahasia", nil, 0).Code)
	assert.Equal(t, 404, do(r, "GET", "/public/posts/tidak-ada", nil, 0).Code)
}

func TestAuthorCRUD(t *testing.T) {
	_, r := setup(t)
	do(r, "POST", "/posts", PostRequest{Title: "Satu", Content: "a b c", IsPublished: bptr(true)}, 1)
	do(r, "POST", "/posts", PostRequest{Title: "Dua", Content: "d e f"}, 1)
	do(r, "POST", "/posts", PostRequest{Title: "Milik Boss", Content: "g"}, 2)

	assert.Len(t, list(t, do(r, "GET", "/posts", nil, 1)), 2)

	d := data(t, do(r, "GET", "/posts/stats", nil, 1))
	assert.EqualValues(t, 2, d["total_articles"])
	assert.EqualValues(t, 1, d["total_selesai"])
	assert.EqualValues(t, 1, d["belum_selesai"])

	assert.Equal(t, 200, do(r, "GET", "/posts/2", nil, 1).Code)
	assert.Equal(t, 403, do(r, "GET", "/posts/3", nil, 1).Code) // milik user lain
	assert.Equal(t, 404, do(r, "GET", "/posts/999", nil, 1).Code)
	assert.Equal(t, 404, do(r, "GET", "/posts/99999999999999999999", nil, 1).Code) // terlalu besar
	assert.Equal(t, 400, do(r, "GET", "/posts/abc", nil, 1).Code)

	// update: is_published nil = tidak berubah
	w := do(r, "PUT", "/posts/2", PostRequest{Title: " Dua Baru ", Category: "Opini", Content: "isi baru"}, 1)
	require.Equal(t, 200, w.Code)
	d = data(t, w)
	assert.Equal(t, "Dua Baru", d["title"])
	assert.Equal(t, "dua", d["slug"]) // slug stabil
	assert.Equal(t, false, d["is_published"])

	// update sambil menerbitkan
	w = do(r, "PUT", "/posts/2", PostRequest{Title: "Dua", Content: "isi", IsPublished: bptr(true)}, 1)
	assert.NotNil(t, data(t, w)["published_at"])

	assert.Equal(t, 400, do(r, "PUT", "/posts/2", PostRequest{Title: "x", Content: "y", Category: "Olahraga"}, 1).Code)
	assert.Equal(t, 400, do(r, "PUT", "/posts/2", "{bukan json", 1).Code)
	assert.Equal(t, 400, do(r, "PUT", "/posts/abc", PostRequest{Title: "x", Content: "y"}, 1).Code)
	assert.Equal(t, 403, do(r, "PUT", "/posts/3", PostRequest{Title: "x", Content: "y"}, 1).Code)
	assert.Equal(t, 404, do(r, "PUT", "/posts/999", PostRequest{Title: "x", Content: "y"}, 1).Code)
	// kepemilikan diperiksa sebelum validasi body
	assert.Equal(t, 403, do(r, "PUT", "/posts/3", "{}", 1).Code)
	assert.Equal(t, 404, do(r, "PUT", "/posts/999", "{}", 1).Code)

	// toggle bolak-balik
	d = data(t, do(r, "PATCH", "/posts/2/toggle-publish", nil, 1))
	assert.Equal(t, false, d["is_published"])
	assert.Nil(t, d["published_at"])
	d = data(t, do(r, "PATCH", "/posts/2/toggle-publish", nil, 1))
	assert.Equal(t, true, d["is_published"])
	assert.NotNil(t, d["published_at"])
	assert.Equal(t, 403, do(r, "PATCH", "/posts/3/toggle-publish", nil, 1).Code)
	assert.Equal(t, 404, do(r, "PATCH", "/posts/999/toggle-publish", nil, 1).Code)
	assert.Equal(t, 400, do(r, "PATCH", "/posts/abc/toggle-publish", nil, 1).Code)

	// hapus
	assert.Equal(t, 403, do(r, "DELETE", "/posts/3", nil, 1).Code)
	assert.Equal(t, 404, do(r, "DELETE", "/posts/999", nil, 1).Code)
	assert.Equal(t, 400, do(r, "DELETE", "/posts/abc", nil, 1).Code)
	assert.Equal(t, 200, do(r, "DELETE", "/posts/2", nil, 1).Code)
	assert.Equal(t, 404, do(r, "GET", "/posts/2", nil, 1).Code)
}

func TestAdmin(t *testing.T) {
	_, r := setup(t)
	do(r, "POST", "/posts", PostRequest{Title: "Satu", Content: "a", IsPublished: bptr(true)}, 1)
	do(r, "POST", "/posts", PostRequest{Title: "Dua", Content: "b"}, 1)

	users := list(t, do(r, "GET", "/admin/users", nil, 2))
	require.Len(t, users, 2)
	first := users[0].(map[string]any)
	assert.Equal(t, "ani", first["username"])
	assert.EqualValues(t, 2, first["total_posts"])
	assert.EqualValues(t, 1, first["total_published"])
	assert.EqualValues(t, 2, first["total_articles"])
	assert.EqualValues(t, 1, first["total_selesai"])
	assert.NotContains(t, do(r, "GET", "/admin/users", nil, 2).Body.String(), "password")

	assert.Len(t, list(t, do(r, "GET", "/admin/posts", nil, 2)), 2)
	assert.Len(t, list(t, do(r, "GET", "/admin/users/1/posts", nil, 2)), 2)
	assert.Equal(t, 200, do(r, "GET", "/admin/users/99/posts", nil, 2).Code) // user tidak ada -> daftar kosong
	assert.Len(t, list(t, do(r, "GET", "/admin/users/99/posts", nil, 2)), 0)
	assert.Equal(t, 400, do(r, "GET", "/admin/users/abc/posts", nil, 2).Code)
}

func TestDatabaseUnavailable(t *testing.T) {
	db, r := setup(t)
	closeDB(db)
	valid := PostRequest{Title: "x", Content: "y"}
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{"GET", "/public/posts", nil},
		{"GET", "/public/posts/x", nil},
		{"GET", "/posts", nil},
		{"GET", "/posts/stats", nil},
		{"GET", "/posts/1", nil},
		{"POST", "/posts", valid},
		{"PUT", "/posts/1", valid},
		{"PATCH", "/posts/1/toggle-publish", nil},
		{"DELETE", "/posts/1", nil},
		{"GET", "/admin/users", nil},
		{"GET", "/admin/posts", nil},
		{"GET", "/admin/users/1/posts", nil},
	} {
		assert.Equal(t, 500, do(r, c.method, c.path, c.body, 1).Code, c.method+" "+c.path)
	}
}

func TestWriteFailures(t *testing.T) {
	db, r := setup(t)
	require.Equal(t, 201, do(r, "POST", "/posts", PostRequest{Title: "Satu", Content: "a"}, 1).Code)

	failOp(db, "create")
	assert.Equal(t, 500, do(r, "POST", "/posts", PostRequest{Title: "Dua", Content: "b"}, 1).Code)

	failOp(db, "update")
	assert.Equal(t, 500, do(r, "PUT", "/posts/1", PostRequest{Title: "Satu", Content: "a"}, 1).Code)
	assert.Equal(t, 500, do(r, "PATCH", "/posts/1/toggle-publish", nil, 1).Code)

	failOp(db, "delete")
	assert.Equal(t, 500, do(r, "DELETE", "/posts/1", nil, 1).Code)
}

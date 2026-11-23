package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/config"
	"app/internal/database"
	"app/internal/features/ai"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type env struct {
	r     *gin.Engine
	close func()
}

func newEnv(t *testing.T) env {
	gin.SetMode(gin.TestMode)
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"title":"Berita","url":"http://berita","snippet":"fakta"}]}`))
	}))
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, _ := json.Marshal(`{"title":"Artikel AI","category":"Teknologi","excerpt":"ringkas","content":"# Isi"}`)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":` + string(content) + `}}]}`))
	}))

	db := database.OpenSQLiteMemory()
	require.NoError(t, database.Migrate(db))
	require.NoError(t, database.Seed(db, "admin", "admin"))

	cfg := &config.Config{AppKey: "router-test-secret"}
	aiSvc := ai.NewService(&ai.SearchClient{Endpoint: search.URL}, &ai.ChatClient{Endpoint: chat.URL, Key: "k", Model: "m"})
	return env{r: New(cfg, db, aiSvc), close: func() { search.Close(); chat.Close() }}
}

func call(r *gin.Engine, method, path, token string, body any) (*httptest.ResponseRecorder, map[string]any) {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var parsed map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &parsed)
	return w, parsed
}

func login(t *testing.T, r *gin.Engine, user, pass string) string {
	w, b := call(r, "POST", "/api/auth/login", "", map[string]string{"username": user, "password": pass})
	require.Equal(t, 200, w.Code)
	return b["data"].(map[string]any)["token"].(string)
}

func TestDocsAndNoRoute(t *testing.T) {
	e := newEnv(t)
	defer e.close()

	w, _ := call(e.r, "GET", "/swagger/index.html", "", nil)
	assert.Equal(t, 200, w.Code)
	w, _ = call(e.r, "GET", "/swagger/doc.json", "", nil)
	assert.Equal(t, 200, w.Code)
	w, spec := call(e.r, "GET", "/api.json", "", nil)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "2.0", spec["swagger"])

	w, b := call(e.r, "GET", "/api/tidak-ada", "", nil)
	assert.Equal(t, 404, w.Code)
	assert.Equal(t, "Endpoint not found", b["message"])
	assert.Equal(t, "fail", b["status"])
}

func TestCategories(t *testing.T) {
	e := newEnv(t)
	defer e.close()
	w, b := call(e.r, "GET", "/api/public/categories", "", nil)
	assert.Equal(t, 200, w.Code)
	assert.Len(t, b["data"], 6)
}

func TestEndToEnd(t *testing.T) {
	e := newEnv(t)
	defer e.close()
	r := e.r

	// registrasi & login penulis
	w, _ := call(r, "POST", "/api/auth/register", "", map[string]string{"username": "budi", "email": "budi@mail.com", "password": "rahasia1"})
	require.Equal(t, 201, w.Code)
	token := login(t, r, "budi", "rahasia1")

	assert.Equal(t, 200, func() int { w, _ := call(r, "GET", "/api/auth/profile", token, nil); return w.Code }())
	assert.Equal(t, 401, func() int { w, _ := call(r, "GET", "/api/posts", "", nil); return w.Code }())

	// AI generate dengan search engine
	w, b := call(r, "POST", "/api/posts/ai-generate", token, map[string]any{"topic": "Go", "include_web_search": true})
	require.Equal(t, 200, w.Code)
	draft := b["data"].(map[string]any)
	assert.Equal(t, "Artikel AI", draft["title"])
	assert.Len(t, draft["sources"], 1)

	// simpan draf sebagai artikel terbit
	w, b = call(r, "POST", "/api/posts", token, map[string]any{
		"title": draft["title"], "category": draft["category"], "excerpt": draft["excerpt"], "content": draft["content"], "is_published": true,
	})
	require.Equal(t, 201, w.Code)
	slug := b["data"].(map[string]any)["slug"].(string)

	w, _ = call(r, "GET", "/api/public/posts/"+slug, "", nil)
	assert.Equal(t, 200, w.Code)
	w, b = call(r, "GET", "/api/public/posts?q=artikel", "", nil)
	assert.Len(t, b["data"], 1)

	w, b = call(r, "GET", "/api/posts/stats", token, nil)
	assert.EqualValues(t, 1, b["data"].(map[string]any)["total_published"])

	w, _ = call(r, "PATCH", "/api/posts/1/toggle-publish", token, nil)
	assert.Equal(t, 200, w.Code)
	w, _ = call(r, "PUT", "/api/posts/1", token, map[string]any{"title": "Baru", "content": "isi"})
	assert.Equal(t, 200, w.Code)
	w, _ = call(r, "DELETE", "/api/posts/1", token, nil)
	assert.Equal(t, 200, w.Code)

	// RBAC: penulis tidak boleh ke admin, admin boleh
	w, _ = call(r, "GET", "/api/admin/users", token, nil)
	assert.Equal(t, 403, w.Code)
	admin := login(t, r, "admin", "admin")
	w, b = call(r, "GET", "/api/admin/users", admin, nil)
	assert.Equal(t, 200, w.Code)
	assert.Len(t, b["data"], 2)
	w, _ = call(r, "GET", "/api/admin/posts", admin, nil)
	assert.Equal(t, 200, w.Code)
	w, _ = call(r, "GET", "/api/admin/users/2/posts", admin, nil)
	assert.Equal(t, 200, w.Code)
}

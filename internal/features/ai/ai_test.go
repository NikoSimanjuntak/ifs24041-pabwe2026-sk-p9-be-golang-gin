package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jsonServer(status int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

// ---------- SearchClient ----------

func TestSearchFormats(t *testing.T) {
	cases := map[string]string{
		"results": `{"results":[{"title":"A","url":"http://a","snippet":"sa"}]}`,
		"data":    `{"data":[{"title":"A","link":"http://a","content":"sa"}]}`,
		"items":   `{"items":[{"title":"A","url":"http://a","description":"sa"}]}`,
	}
	for name, body := range cases {
		srv := jsonServer(200, body)
		got, err := (&SearchClient{Endpoint: srv.URL, HTTP: srv.Client()}).Search(context.Background(), "go lang")
		srv.Close()
		require.NoError(t, err, name)
		assert.Equal(t, []Source{{Title: "A", URL: "http://a", Snippet: "sa"}}, got, name)
	}
}

func TestSearchSendsQueryAndLimitsResults(t *testing.T) {
	var q string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q = r.URL.Query().Get("q")
		items := make([]string, 8)
		for i := range items {
			items[i] = fmt.Sprintf(`{"title":"T%d","url":"u"}`, i)
		}
		_, _ = w.Write([]byte(`{"results":[` + strings.Join(items, ",") + `]}`))
	}))
	defer srv.Close()

	got, err := (&SearchClient{Endpoint: srv.URL}).Search(context.Background(), "berita & teknologi") // HTTP nil -> klien bawaan
	require.NoError(t, err)
	assert.Equal(t, "berita & teknologi", q)
	assert.Len(t, got, maxSources)
}

func TestSearchErrors(t *testing.T) {
	_, err := (&SearchClient{Endpoint: "http://127.0.0.1:1/\x7f"}).Search(context.Background(), "x")
	assert.Error(t, err) // URL tidak valid

	_, err = (&SearchClient{Endpoint: "http://127.0.0.1:1"}).Search(context.Background(), "x")
	assert.Error(t, err) // koneksi ditolak

	srv := jsonServer(500, `{}`)
	_, err = (&SearchClient{Endpoint: srv.URL}).Search(context.Background(), "x")
	srv.Close()
	assert.ErrorContains(t, err, "status 500")

	srv = jsonServer(200, `bukan json`)
	_, err = (&SearchClient{Endpoint: srv.URL}).Search(context.Background(), "x")
	srv.Close()
	assert.Error(t, err)
}

// ---------- ChatClient ----------

func chatBody(content string) string {
	b, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": content}}}})
	return string(b)
}

func TestChatComplete(t *testing.T) {
	var auth, path string
	var sent chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&sent)
		_, _ = w.Write([]byte(chatBody("halo")))
	}))
	defer srv.Close()

	out, err := (&ChatClient{Endpoint: srv.URL + "/", Key: "k", Model: "m"}).Complete(context.Background(), "sys", "usr")
	require.NoError(t, err)
	assert.Equal(t, "halo", out)
	assert.Equal(t, "Bearer k", auth)
	assert.Equal(t, "/chat/completions", path)
	assert.Equal(t, "m", sent.Model)
	require.Len(t, sent.Messages, 2)
	assert.Equal(t, "usr", sent.Messages[1].Content)
}

func TestChatErrors(t *testing.T) {
	_, err := (&ChatClient{Endpoint: "http://127.0.0.1:1/\x7f"}).Complete(context.Background(), "s", "u")
	assert.Error(t, err)

	_, err = (&ChatClient{Endpoint: "http://127.0.0.1:1", HTTP: &http.Client{}}).Complete(context.Background(), "s", "u")
	assert.Error(t, err)

	srv := jsonServer(401, `{}`)
	_, err = (&ChatClient{Endpoint: srv.URL}).Complete(context.Background(), "s", "u")
	srv.Close()
	assert.ErrorContains(t, err, "status 401")

	srv = jsonServer(200, `bukan json`)
	_, err = (&ChatClient{Endpoint: srv.URL}).Complete(context.Background(), "s", "u")
	srv.Close()
	assert.Error(t, err)

	srv = jsonServer(200, `{"choices":[]}`)
	_, err = (&ChatClient{Endpoint: srv.URL}).Complete(context.Background(), "s", "u")
	srv.Close()
	assert.ErrorContains(t, err, "pilihan")
}

// ---------- Service ----------

type fakeSearch struct {
	sources []Source
	err     error
	called  bool
}

func (f *fakeSearch) Search(context.Context, string) ([]Source, error) {
	f.called = true
	return f.sources, f.err
}

type fakeChat struct {
	out    string
	err    error
	prompt string
}

func (f *fakeChat) Complete(_ context.Context, _, user string) (string, error) {
	f.prompt = user
	return f.out, f.err
}

const goodJSON = `{"title":" Judul AI ","category":"Teknologi","excerpt":" ringkas ","content":"# Isi"}`

func TestGenerateWithSearch(t *testing.T) {
	s := &fakeSearch{sources: []Source{{Title: "Sumber 1", URL: "http://s1", Snippet: "fakta"}}}
	c := &fakeChat{out: "```json\n" + goodJSON + "\n```"}
	res, err := NewService(s, c).Generate(context.Background(), GenerateRequest{Topic: "Go", Category: "Bisnis", IncludeWebSearch: true})
	require.NoError(t, err)
	assert.True(t, s.called)
	assert.Equal(t, "Judul AI", res.Title)
	assert.Equal(t, "ringkas", res.Excerpt)
	assert.Equal(t, "Teknologi", res.Category)
	assert.Equal(t, s.sources, res.Sources)
	assert.Contains(t, c.prompt, "[1] Sumber 1 (http://s1): fakta")
	assert.Contains(t, c.prompt, "## Sumber")
}

func TestGenerateWithoutSearch(t *testing.T) {
	s := &fakeSearch{}
	c := &fakeChat{out: `{"title":"J","category":"Tidak Ada","content":"isi"}`}
	res, err := NewService(s, c).Generate(context.Background(), GenerateRequest{Topic: "Go"})
	require.NoError(t, err)
	assert.False(t, s.called)
	assert.Equal(t, "Umum", res.Category) // fallback ke kategori permintaan (default Umum)
	assert.NotNil(t, res.Sources)
	assert.Empty(t, res.Sources)
	assert.NotContains(t, c.prompt, "## Sumber")
}

func TestGenerateErrors(t *testing.T) {
	ctx := context.Background()

	_, err := NewService(&fakeSearch{}, &fakeChat{}).Generate(ctx, GenerateRequest{Topic: "x", Category: "Olahraga"})
	assert.ErrorIs(t, err, ErrInvalidCategory)

	_, err = NewService(&fakeSearch{}, &fakeChat{err: errors.New("down")}).Generate(ctx, GenerateRequest{Topic: "x"})
	assert.ErrorContains(t, err, "pemanggilan AI")

	for _, bad := range []string{"bukan json sama sekali", `{rusak`, `{"title":"","content":"isi"}`, `{"title":"J","content":"  "}`} {
		_, err = NewService(&fakeSearch{}, &fakeChat{out: bad}).Generate(ctx, GenerateRequest{Topic: "x"})
		assert.ErrorIs(t, err, ErrBadOutput, bad)
	}
}

func TestGenerateSearchFailureDoesNotBlock(t *testing.T) {
	c := &fakeChat{out: goodJSON}
	res, err := NewService(&fakeSearch{err: errors.New("down")}, c).Generate(context.Background(), GenerateRequest{Topic: "x", IncludeWebSearch: true})
	require.NoError(t, err)
	assert.Empty(t, res.Sources)
	assert.Contains(t, res.SearchWarning, "down")
	assert.NotContains(t, c.prompt, "## Sumber")
}

func TestExtractJSON(t *testing.T) {
	assert.Equal(t, `{"a":1}`, extractJSON("teks {\"a\":1} teks"))
	assert.Equal(t, "tanpa kurung", extractJSON("tanpa kurung"))
	assert.Equal(t, "} {", extractJSON("} {"))
}

// ---------- Handler ----------

func post(h *Handler, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/gen", h.Generate)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/gen", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestHandler(t *testing.T) {
	ok := NewHandler(NewService(&fakeSearch{sources: []Source{{Title: "S"}}}, &fakeChat{out: goodJSON}))
	w := post(ok, `{"topic":"Go","include_web_search":true}`)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"sources"`)

	assert.Equal(t, 400, post(ok, `{}`).Code)
	assert.Equal(t, 400, post(ok, `{"topic":"Go","category":"Olahraga"}`).Code)

	bad := NewHandler(NewService(&fakeSearch{}, &fakeChat{err: errors.New("down")}))
	assert.Equal(t, 502, post(bad, `{"topic":"Go"}`).Code)
}

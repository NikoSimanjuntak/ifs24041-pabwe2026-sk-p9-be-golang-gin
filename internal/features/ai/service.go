package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"app/internal/database"
)

var (
	ErrInvalidCategory = errors.New("kategori tidak valid")
	ErrBadOutput       = errors.New("keluaran AI tidak berformat JSON artikel yang valid")
)

type Searcher interface {
	Search(ctx context.Context, query string) ([]Source, error)
}

type Completer interface {
	Complete(ctx context.Context, system, user string) (string, error)
}

type Service struct {
	search Searcher
	chat   Completer
}

func NewService(search Searcher, chat Completer) *Service {
	return &Service{search: search, chat: chat}
}

const systemPrompt = `Anda adalah penulis blog profesional berbahasa Indonesia.
Balas HANYA dengan satu objek JSON valid tanpa teks lain dan tanpa blok kode, dengan kunci:
"title" (judul menarik), "category" (salah satu: Teknologi, Pendidikan, Gaya Hidup, Bisnis, Opini, Umum),
"excerpt" (ringkasan pemikat 1-2 kalimat), dan "content" (isi lengkap artikel dalam format Markdown
dengan heading, paragraf, dan daftar bila perlu).`

func buildPrompt(topic, category string, sources []Source) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Tulis artikel blog tentang: %s\nKategori yang diinginkan: %s\n", topic, category)
	if len(sources) > 0 {
		b.WriteString("\nGunakan fakta terkini dari hasil penelusuran web berikut. Sitasi dengan format [n] dan akhiri artikel dengan bagian \"## Sumber\" yang memuat daftar rujukan:\n")
		for i, s := range sources {
			fmt.Fprintf(&b, "[%d] %s (%s): %s\n", i+1, s.Title, s.URL, s.Snippet)
		}
	}
	return b.String()
}

// extractJSON mengambil objek JSON terluar (mengabaikan pagar kode ```json bila ada).
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end < start {
		return s
	}
	return s[start : end+1]
}

func (s *Service) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "Umum"
	}
	if !database.IsValidCategory(category) {
		return nil, ErrInvalidCategory
	}

	sources := []Source{}
	if req.IncludeWebSearch {
		found, err := s.search.Search(ctx, req.Topic)
		if err != nil {
			return nil, fmt.Errorf("penelusuran search engine gagal: %w", err)
		}
		sources = found
	}

	raw, err := s.chat.Complete(ctx, systemPrompt, buildPrompt(req.Topic, category, sources))
	if err != nil {
		return nil, fmt.Errorf("pemanggilan AI gagal: %w", err)
	}

	var out struct {
		Title    string `json:"title"`
		Category string `json:"category"`
		Excerpt  string `json:"excerpt"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal([]byte(extractJSON(raw)), &out); err != nil {
		return nil, ErrBadOutput
	}
	if strings.TrimSpace(out.Title) == "" || strings.TrimSpace(out.Content) == "" {
		return nil, ErrBadOutput
	}
	if !database.IsValidCategory(out.Category) {
		out.Category = category
	}
	return &GenerateResponse{
		Title:    strings.TrimSpace(out.Title),
		Category: out.Category,
		Excerpt:  strings.TrimSpace(out.Excerpt),
		Content:  out.Content,
		Sources:  sources,
	}, nil
}

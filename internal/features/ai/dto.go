package ai

// GenerateRequest adalah body POST /api/posts/ai-generate.
type GenerateRequest struct {
	Topic            string `json:"topic" binding:"required,max=300"`
	Category         string `json:"category"`
	IncludeWebSearch bool   `json:"include_web_search"`
}

// Source adalah satu rujukan hasil penelusuran Search Engine.
type Source struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// GenerateResponse adalah draf artikel utuh beserta sumber rujukan web.
type GenerateResponse struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Excerpt  string   `json:"excerpt"`
	Content  string   `json:"content"`
	Sources  []Source `json:"sources"`
}

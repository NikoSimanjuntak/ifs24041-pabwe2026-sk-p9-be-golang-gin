package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const maxSources = 5

// SearchClient memanggil Search Engine: GET {Endpoint}?q={query}.
// Respons yang didukung: {"results":[...]}, {"data":[...]}, atau {"items":[...]},
// dengan tiap item memuat title, url/link, dan snippet/content/description.
type SearchClient struct {
	Endpoint string
	HTTP     *http.Client
}

type rawItem struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Link        string `json:"link"`
	Snippet     string `json:"snippet"`
	Content     string `json:"content"`
	Description string `json:"description"`
}

type rawSearchResponse struct {
	Results []rawItem `json:"results"`
	Data    []rawItem `json:"data"`
	Items   []rawItem `json:"items"`
}

func httpClient(c *http.Client, timeout time.Duration) *http.Client {
	if c != nil {
		return c
	}
	return &http.Client{Timeout: timeout}
}

func first(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (s *SearchClient) Search(ctx context.Context, query string) ([]Source, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.Endpoint+"?q="+url.QueryEscape(query), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient(s.HTTP, 20*time.Second).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("search engine mengembalikan status %d", resp.StatusCode)
	}

	var raw rawSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	items := raw.Results
	if len(items) == 0 {
		items = raw.Data
	}
	if len(items) == 0 {
		items = raw.Items
	}

	sources := make([]Source, 0, maxSources)
	for _, it := range items {
		if len(sources) == maxSources {
			break
		}
		sources = append(sources, Source{
			Title:   it.Title,
			URL:     first(it.URL, it.Link),
			Snippet: first(it.Snippet, it.Content, it.Description),
		})
	}
	return sources, nil
}

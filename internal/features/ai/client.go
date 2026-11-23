package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ChatClient memanggil endpoint OpenAI-compatible: POST {Endpoint}/chat/completions.
type ChatClient struct {
	Endpoint string
	Key      string
	Model    string
	HTTP     *http.Client
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func (c *ChatClient) Complete(ctx context.Context, system, user string) (string, error) {
	payload, _ := json.Marshal(chatRequest{
		Model:    c.Model,
		Messages: []chatMessage{{Role: "system", Content: system}, {Role: "user", Content: user}},
	})
	url := strings.TrimRight(c.Endpoint, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Key)

	resp, err := httpClient(c.HTTP, 120*time.Second).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("openai mengembalikan status %d", resp.StatusCode)
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("openai tidak mengembalikan pilihan jawaban")
	}
	return out.Choices[0].Message.Content, nil
}

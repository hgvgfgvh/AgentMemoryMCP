package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Client OpenAI 兼容 Chat Completions 客户端。
type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

// ConfigFromEnv 读取环境变量；未配置 API 则 ok=false。
func ConfigFromEnv() (Client, bool) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("MEMORY_MCP_LLM_API_BASE")), "/")
	if base == "" {
		return Client{}, false
	}
	key := os.Getenv("MEMORY_MCP_LLM_API_KEY")
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	model := strings.TrimSpace(os.Getenv("MEMORY_MCP_LLM_MODEL"))
	if model == "" {
		model = "gpt-4o-mini"
	}
	timeout := 45 * time.Second
	if v := os.Getenv("MEMORY_MCP_LLM_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Second
		}
	}
	return Client{
		BaseURL: base,
		APIKey:  key,
		Model:   model,
		HTTP:    &http.Client{Timeout: timeout},
	}, true
}

type chatReq struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	ResponseFormat *struct {
		Type string `json:"type"`
	} `json:"response_format,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ChatJSON 单次对话，要求模型返回 JSON 文本。
func (c *Client) ChatJSON(ctx context.Context, system, user string) (string, error) {
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 45 * time.Second}
	}
	url := chatCompletionsURL(c.BaseURL)
	body := chatReq{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		ResponseFormat: &struct {
			Type string `json:"type"`
		}{Type: "json_object"},
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm http %d: %s", resp.StatusCode, trim(string(raw), 400))
	}
	var out chatResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Error != nil && out.Error.Message != "" {
		return "", fmt.Errorf("llm api: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("llm: empty choices")
	}
	content := strings.TrimSpace(out.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("llm: empty content")
	}
	return content, nil
}

// chatCompletionsURL 兼容 BASE 为根路径或已含 /chat/completions 的配置。
func chatCompletionsURL(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	return base + "/chat/completions"
}

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

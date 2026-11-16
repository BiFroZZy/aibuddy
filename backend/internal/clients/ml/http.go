package ml

import (
	"aibuddy/internal/domain"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultChatPath   = "/chat"
	DefaultStreamPath = "/chat/stream"
)

type HTTPClient struct {
	BaseURL    string
	HTTP       *http.Client
	ChatPath   string
	StreamPath string
}

var _ Client = (*HTTPClient)(nil)

func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTP:       &http.Client{Timeout: timeout},
		ChatPath:   DefaultChatPath,
		StreamPath: DefaultStreamPath,
	}
}

func unavailable(format string, a ...any) error {
	return fmt.Errorf("%w: %s", domain.ErrLLMUnavailable, fmt.Sprintf(format, a...))
}

func (c *HTTPClient) post(ctx context.Context, path, accept string, req ChatRequest) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ml: encode request: %w", err)
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ml: build request: %w", err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", accept)
	resp, err := c.HTTP.Do(r)
	if err != nil {
		return nil, unavailable("%v", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, unavailable("status %d", resp.StatusCode)
	}
	return resp, nil
}

func (c *HTTPClient) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	resp, err := c.post(ctx, c.ChatPath, "application/json", req)
	if err != nil {
		return ChatResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	var out ChatResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
		return ChatResponse{}, unavailable("unexpected response format")
	}
	return out, nil
}

type streamEvent struct {
	Token    string    `json:"token"`
	Done     bool      `json:"done"`
	Text     string    `json:"text"`
	ToolCall *ToolCall `json:"tool_call"`
}

func (c *HTTPClient) Stream(ctx context.Context, req ChatRequest, onToken func(string)) (ChatResponse, error) {
	resp, err := c.post(ctx, c.StreamPath, "text/event-stream", req)
	if err != nil {
		return ChatResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		data, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue
		}
		var ev streamEvent
		if err := json.Unmarshal([]byte(strings.TrimSpace(data)), &ev); err != nil {
			return ChatResponse{}, unavailable("malformed stream event")
		}
		if ev.Done {
			return ChatResponse{Text: ev.Text, ToolCall: ev.ToolCall}, nil
		}
		if ev.Token != "" && onToken != nil {
			onToken(ev.Token)
		}
	}
	if err := sc.Err(); err != nil {
		return ChatResponse{}, unavailable("stream interrupted: %v", err)
	}
	return ChatResponse{}, unavailable("stream ended without a final event")
}

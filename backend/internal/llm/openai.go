package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"aibuddy/internal/domain"
)

type OpenAI struct {
	BaseURL    string
	Model      string
	APIKey     string
	HTTP       *http.Client
	RetryDelay time.Duration
}

func NewOpenAI(baseURL, model, apiKey string, timeout time.Duration) *OpenAI {
	return &OpenAI{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Model:      model,
		APIKey:     apiKey,
		HTTP:       &http.Client{Timeout: timeout},
		RetryDelay: 500 * time.Millisecond,
	}
}

type oaToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type oaMessage struct {
	Role       string       `json:"role"`
	Content    string       `json:"content,omitempty"`
	Name       string       `json:"name,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
	ToolCalls  []oaToolCall `json:"tool_calls,omitempty"`
}

type oaTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

type oaRequest struct {
	Model       string      `json:"model"`
	Messages    []oaMessage `json:"messages"`
	Tools       []oaTool    `json:"tools,omitempty"`
	Temperature float64     `json:"temperature"`
}

type oaResponse struct {
	Choices []struct {
		Message oaMessage `json:"message"`
	} `json:"choices"`
}

func (c *OpenAI) Chat(ctx context.Context, msgs []Message, tools []ToolSpec) (Message, error) {
	body, err := json.Marshal(c.buildRequest(msgs, tools))
	if err != nil {
		return Message{}, fmt.Errorf("llm: encode request: %w", err)
	}

	const attempts = 2
	var lastErr error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return Message{}, fmt.Errorf("%w: %v", domain.ErrLLMUnavailable, ctx.Err())
			case <-time.After(c.RetryDelay):
			}
		}
		msg, retry, err := c.do(ctx, body)
		if err == nil {
			return msg, nil
		}
		lastErr = err
		if !retry {
			break
		}
	}
	return Message{}, lastErr
}

func (c *OpenAI) buildRequest(msgs []Message, tools []ToolSpec) oaRequest {
	req := oaRequest{Model: c.Model, Temperature: 0.2}
	for _, m := range msgs {
		om := oaMessage{Role: m.Role, Content: m.Content, Name: m.Name, ToolCallID: m.ToolCallID}
		for _, tc := range m.ToolCalls {
			o := oaToolCall{ID: tc.ID, Type: "function"}
			o.Function.Name = tc.Name
			o.Function.Arguments = tc.Arguments
			om.ToolCalls = append(om.ToolCalls, o)
		}
		req.Messages = append(req.Messages, om)
	}
	for _, t := range tools {
		o := oaTool{Type: "function"}
		o.Function.Name = t.Name
		o.Function.Description = t.Description
		o.Function.Parameters = t.Parameters
		req.Tools = append(req.Tools, o)
	}
	return req
}

func (c *OpenAI) do(ctx context.Context, body []byte) (msg Message, retry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Message{}, false, fmt.Errorf("llm: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Message{}, true, fmt.Errorf("%w: %v", domain.ErrLLMUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Message{}, true, fmt.Errorf("%w: read body: %v", domain.ErrLLMUnavailable, err)
	}
	if resp.StatusCode != http.StatusOK {
		retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		return Message{}, retryable, fmt.Errorf("%w: status %d", domain.ErrLLMUnavailable, resp.StatusCode)
	}

	var parsed oaResponse
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Choices) == 0 {
		return Message{}, false, fmt.Errorf("%w: unexpected response format", domain.ErrLLMUnavailable)
	}
	m := parsed.Choices[0].Message
	out := Message{Role: RoleAssistant, Content: m.Content}
	for _, tc := range m.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: tc.Function.Arguments})
	}
	return out, false, nil
}

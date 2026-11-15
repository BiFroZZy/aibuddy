package ml

import (
	"context"
	"encoding/json"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ToolSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"`
}

type KBChunk struct {
	Source  string `json:"source"`
	Content string `json:"content"`
}

type UserContext struct {
	Role  string `json:"role"`
	Team  string `json:"team"`
	Stage string `json:"stage"`
}

type ChatRequest struct {
	User    UserContext `json:"user"`
	History []Message   `json:"history"`
	Tools   []ToolSpec  `json:"tools"`
	KB      []KBChunk   `json:"kb"`
}

type ToolCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

type ChatResponse struct {
	Text     string    `json:"text"`
	ToolCall *ToolCall `json:"tool_call,omitempty"`
}

type Client interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
	Stream(ctx context.Context, req ChatRequest, onToken func(string)) (ChatResponse, error)
}

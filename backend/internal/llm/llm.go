package llm

import (
	"context"
	"encoding/json"
)

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type Message struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
	Name       string
}

type ToolSpec struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

type Client interface {
	Chat(ctx context.Context, msgs []Message, tools []ToolSpec) (Message, error)
}

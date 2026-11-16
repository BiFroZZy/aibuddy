package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"aibuddy/internal/auth"
	"aibuddy/internal/clients/ml"
	"aibuddy/internal/domain"
)

type KBProvider interface {
	Search(ctx context.Context, id domain.Identity, query string, limit int) ([]domain.KBHit, error)
}

type StageProvider func(ctx context.Context, id domain.Identity) string

type ML struct {
	Client ml.Client
	KB     KBProvider
	Stage  StageProvider
}

var _ Client = (*ML)(nil)

const (
	kbToolName = "search_kb"
	kbLimit    = 5
)

func (a *ML) Chat(ctx context.Context, msgs []Message, specs []ToolSpec) (Message, error) {
	id, ok := auth.IdentityFrom(ctx)
	if !ok {
		return Message{}, errors.New("llm: caller identity is missing in the context")
	}

	req := ml.ChatRequest{
		User:    ml.UserContext{Role: string(id.Role), Team: id.Team},
		History: []ml.Message{},
		Tools:   []ml.ToolSpec{},
		KB:      []ml.KBChunk{},
	}
	if a.Stage != nil {
		req.User.Stage = a.Stage(ctx, id)
	}

	for _, m := range msgs {
		if h, ok := historyEntry(m); ok {
			req.History = append(req.History, h)
		}
	}
	for _, s := range specs {
		if s.Name == kbToolName {
			continue
		}
		req.Tools = append(req.Tools, ml.ToolSpec{Name: s.Name, Description: s.Description, Schema: s.Parameters})
	}

	if a.KB != nil {
		if q := lastUserText(msgs); q != "" {
			hits, err := a.KB.Search(ctx, id, q, kbLimit)
			if err != nil {
				return Message{}, fmt.Errorf("llm: retrieve knowledge base: %w", err)
			}
			for _, h := range hits {
				req.KB = append(req.KB, ml.KBChunk{Source: h.Source, Content: h.Title + ": " + h.Snippet})
			}
		}
	}

	resp, err := a.Client.Chat(ctx, req)
	if err != nil {
		return Message{}, err
	}

	out := Message{Role: RoleAssistant, Content: resp.Text}
	if tc := resp.ToolCall; tc != nil && tc.Name != "" {
		args := strings.TrimSpace(string(tc.Args))
		if args == "" {
			args = "{}"
		}
		out.ToolCalls = []ToolCall{{ID: fmt.Sprintf("ml_call_%d", countTool(msgs)+1), Name: tc.Name, Arguments: args}}
	}
	return out, nil
}

func historyEntry(m Message) (ml.Message, bool) {
	switch m.Role {
	case RoleUser:
		return ml.Message{Role: RoleUser, Content: m.Content}, true
	case RoleAssistant:
		if len(m.ToolCalls) > 0 {
			tc := m.ToolCalls[0]
			b, _ := json.Marshal(map[string]any{"tool_call": map[string]any{"name": tc.Name, "args": json.RawMessage(validJSON(tc.Arguments))}})
			return ml.Message{Role: RoleAssistant, Content: string(b)}, true
		}
		return ml.Message{Role: RoleAssistant, Content: m.Content}, true
	case RoleTool:
		b, _ := json.Marshal(map[string]any{"tool": m.Name, "result": json.RawMessage(validJSON(m.Content))})
		return ml.Message{Role: RoleTool, Content: string(b)}, true
	}
	return ml.Message{}, false
}

func validJSON(s string) string {
	if json.Valid([]byte(s)) {
		return s
	}
	b, _ := json.Marshal(s)
	return string(b)
}

func lastUserText(msgs []Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleUser {
			return strings.TrimSpace(msgs[i].Content)
		}
	}
	return ""
}

func countTool(msgs []Message) int {
	n := 0
	for _, m := range msgs {
		if m.Role == RoleTool {
			n++
		}
	}
	return n
}

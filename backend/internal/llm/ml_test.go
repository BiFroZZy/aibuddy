package llm

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"aibuddy/internal/auth"
	"aibuddy/internal/clients/ml"
	"aibuddy/internal/domain"

	"github.com/google/uuid"
)

type fakeML struct {
	req  ml.ChatRequest
	resp ml.ChatResponse
	err  error
}

func (f *fakeML) Chat(_ context.Context, r ml.ChatRequest) (ml.ChatResponse, error) {
	f.req = r
	return f.resp, f.err
}

func (f *fakeML) Stream(context.Context, ml.ChatRequest, func(string)) (ml.ChatResponse, error) {
	return ml.ChatResponse{}, errors.New("not used")
}

type fakeKB struct {
	hits  []domain.KBHit
	err   error
	query string
	id    domain.Identity
}

func (f *fakeKB) Search(_ context.Context, id domain.Identity, q string, _ int) ([]domain.KBHit, error) {
	f.query, f.id = q, id
	return f.hits, f.err
}

var caller = domain.Identity{UserID: uuid.New(), Role: domain.RoleIntern, Team: "Backend"}

func withCaller() context.Context { return auth.WithIdentity(context.Background(), caller) }

func TestML_RequestMapping(t *testing.T) {
	client := &fakeML{resp: ml.ChatResponse{Text: "готово"}}
	kb := &fakeKB{hits: []domain.KBHit{{Title: "Отгул", Source: "kb/absence.md", Snippet: "через сервис"}}}
	a := &ML{Client: client, KB: kb, Stage: func(context.Context, domain.Identity) string { return "Доступы" }}

	msgs := []Message{
		{Role: RoleSystem, Content: "секретный системный промпт"},
		{Role: RoleUser, Content: "раньше"},
		{Role: RoleAssistant, Content: "ответ"},
		{Role: RoleUser, Content: "  оформи отгул  "},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "ml_call_1", Name: "create_absence_request", Arguments: `{"date_from":"2030-01-10"}`}}},
		{Role: RoleTool, Name: "create_absence_request", ToolCallID: "ml_call_1", Content: `{"ok":true}`},
	}
	specs := []ToolSpec{
		{Name: "search_kb", Description: "kb"},
		{Name: "create_absence_request", Description: "d", Parameters: json.RawMessage(`{"type":"object"}`)},
	}
	out, err := a.Chat(withCaller(), msgs, specs)
	if err != nil || out.Content != "готово" || len(out.ToolCalls) != 0 || out.Role != RoleAssistant {
		t.Fatalf("reply: %+v %v", out, err)
	}

	r := client.req
	if r.User.Role != "intern" || r.User.Team != "Backend" || r.User.Stage != "Доступы" {
		t.Fatalf("user context: %+v", r.User)
	}
	if len(r.Tools) != 1 || r.Tools[0].Name != "create_absence_request" || string(r.Tools[0].Schema) != `{"type":"object"}` {
		t.Fatalf("search_kb must not be offered to the ML service: %+v", r.Tools)
	}
	if len(r.KB) != 1 || r.KB[0].Source != "kb/absence.md" || !strings.Contains(r.KB[0].Content, "через сервис") || !strings.Contains(r.KB[0].Content, "Отгул") {
		t.Fatalf("kb: %+v", r.KB)
	}
	if kb.query != "оформи отгул" || kb.id != caller {
		t.Fatalf("retrieval must use the last user message and the caller's rights: %q %+v", kb.query, kb.id)
	}
	if len(r.History) != 5 {
		t.Fatalf("history: %+v", r.History)
	}
	for _, h := range r.History {
		if strings.Contains(h.Content, "секретный") {
			t.Fatal("the system prompt must not be sent to the ML service")
		}
	}
	if r.History[3].Role != RoleAssistant || !strings.Contains(r.History[3].Content, `"tool_call":{"args":{"date_from":"2030-01-10"},"name":"create_absence_request"}`) {
		t.Fatalf("tool call entry: %+v", r.History[3])
	}
	if r.History[4].Role != RoleTool || !strings.Contains(r.History[4].Content, `"tool":"create_absence_request"`) || !strings.Contains(r.History[4].Content, `"result":{"ok":true}`) {
		t.Fatalf("tool result entry: %+v", r.History[4])
	}
}

func TestML_ResponseMapping(t *testing.T) {
	client := &fakeML{resp: ml.ChatResponse{ToolCall: &ml.ToolCall{Name: "get_my_progress"}}}
	a := &ML{Client: client}

	out, err := a.Chat(withCaller(), []Message{{Role: RoleUser, Content: "прогресс"}}, nil)
	if err != nil || len(out.ToolCalls) != 1 || out.ToolCalls[0].Name != "get_my_progress" || out.ToolCalls[0].Arguments != "{}" || out.ToolCalls[0].ID != "ml_call_1" {
		t.Fatalf("a tool call without args: %+v %v", out, err)
	}

	msgs := []Message{{Role: RoleUser, Content: "q"}, {Role: RoleTool, Content: `{}`}, {Role: RoleTool, Content: `{}`}}
	out, _ = a.Chat(withCaller(), msgs, nil)
	if out.ToolCalls[0].ID != "ml_call_3" {
		t.Fatalf("call id: %s", out.ToolCalls[0].ID)
	}

	client.resp = ml.ChatResponse{Text: "ответ", ToolCall: &ml.ToolCall{Name: "request_access", Args: json.RawMessage(`{"project":"x"}`)}}
	out, _ = a.Chat(withCaller(), []Message{{Role: RoleUser, Content: "q"}}, nil)
	if out.Content != "ответ" || out.ToolCalls[0].Arguments != `{"project":"x"}` {
		t.Fatalf("text and tool call together: %+v", out)
	}
	client.resp = ml.ChatResponse{ToolCall: &ml.ToolCall{}}
	if out, _ := a.Chat(withCaller(), []Message{{Role: RoleUser, Content: "q"}}, nil); len(out.ToolCalls) != 0 {
		t.Fatalf("nameless tool call: %+v", out)
	}
}

func TestML_NonJSONToolContentIsEscaped(t *testing.T) {
	client := &fakeML{}
	a := &ML{Client: client}
	msgs := []Message{{Role: RoleUser, Content: "q"}, {Role: RoleTool, Name: "t", Content: "plain text"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{Name: "t", Arguments: "not json"}}}}
	if _, err := a.Chat(withCaller(), msgs, nil); err != nil {
		t.Fatal(err)
	}
	for _, h := range client.req.History {
		if !json.Valid([]byte(h.Content)) && h.Role != RoleUser {
			t.Fatalf("content must stay valid JSON: %q", h.Content)
		}
	}
}

func TestML_Failures(t *testing.T) {
	if _, err := (&ML{Client: &fakeML{}}).Chat(context.Background(), nil, nil); err == nil {
		t.Fatal("a missing identity is a programming error")
	}
	if _, err := (&ML{Client: &fakeML{}, KB: &fakeKB{err: errors.New("db down")}}).
		Chat(withCaller(), []Message{{Role: RoleUser, Content: "q"}}, nil); err == nil || errors.Is(err, domain.ErrLLMUnavailable) {
		t.Fatalf("a knowledge-base failure is an infrastructure error, not an LLM outage: %v", err)
	}
	if _, err := (&ML{Client: &fakeML{err: domain.ErrLLMUnavailable}}).
		Chat(withCaller(), []Message{{Role: RoleUser, Content: "q"}}, nil); !errors.Is(err, domain.ErrLLMUnavailable) {
		t.Fatalf("ml failure: %v", err)
	}
	kb := &fakeKB{}
	if _, err := (&ML{Client: &fakeML{}, KB: kb}).Chat(withCaller(), []Message{{Role: RoleSystem, Content: "s"}}, nil); err != nil || kb.query != "" {
		t.Fatalf("no user text: %v %q", err, kb.query)
	}
}

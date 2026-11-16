package chat

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"aibuddy/internal/domain"
	"aibuddy/internal/llm"
	"aibuddy/internal/tools"

	"github.com/google/uuid"
)

type memStore struct {
	mu    sync.Mutex
	convs map[uuid.UUID]uuid.UUID
	msgs  map[uuid.UUID][]domain.ChatMessage
	err   error
}

func newStore() *memStore {
	return &memStore{convs: map[uuid.UUID]uuid.UUID{}, msgs: map[uuid.UUID][]domain.ChatMessage{}}
}

func (s *memStore) CreateConversation(_ context.Context, u uuid.UUID) (uuid.UUID, error) {
	if s.err != nil {
		return uuid.Nil, s.err
	}
	id := uuid.New()
	s.convs[id] = u
	return id, nil
}

func (s *memStore) Owns(_ context.Context, u, c uuid.UUID) (bool, error) {
	return s.convs[c] == u && u != uuid.Nil, s.err
}

func (s *memStore) AddMessage(_ context.Context, c uuid.UUID, role, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs[c] = append(s.msgs[c], domain.ChatMessage{ID: uuid.New(), Role: role, Content: content, CreatedAt: time.Now()})
	return s.err
}

func (s *memStore) Messages(_ context.Context, c uuid.UUID, limit int) ([]domain.ChatMessage, error) {
	m := s.msgs[c]
	if len(m) > limit {
		m = m[len(m)-limit:]
	}
	return m, nil
}

type scriptedLLM struct {
	replies []llm.Message
	err     error
	calls   [][]llm.Message
	specs   [][]llm.ToolSpec
}

func (l *scriptedLLM) Chat(_ context.Context, msgs []llm.Message, specs []llm.ToolSpec) (llm.Message, error) {
	l.calls = append(l.calls, append([]llm.Message(nil), msgs...))
	l.specs = append(l.specs, specs)
	if l.err != nil {
		return llm.Message{}, l.err
	}
	i := len(l.calls) - 1
	if i >= len(l.replies) {
		i = len(l.replies) - 1
	}
	return l.replies[i], nil
}

type fakeTools struct {
	envs    map[string]tools.Envelope
	err     error
	calls   []string
	keys    []string
	actions map[uuid.UUID]domain.Action
}

func (f *fakeTools) Specs(domain.Role) []llm.ToolSpec { return []llm.ToolSpec{{Name: "search_kb"}} }

func (f *fakeTools) Call(_ context.Context, _ domain.Identity, name string, _ json.RawMessage, key string) (tools.Envelope, error) {
	f.calls = append(f.calls, name)
	f.keys = append(f.keys, key)
	if f.err != nil {
		return tools.Envelope{}, f.err
	}
	return f.envs[name], nil
}

func (f *fakeTools) GetAction(_ context.Context, _, id uuid.UUID) (domain.Action, error) {
	a, ok := f.actions[id]
	if !ok {
		return domain.Action{}, domain.ErrNotFound
	}
	return a, nil
}

var user = domain.Identity{UserID: uuid.New(), Role: domain.RoleIntern}

func text(s string) llm.Message { return llm.Message{Role: llm.RoleAssistant, Content: s} }

func call(id, name string) llm.Message {
	return llm.Message{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: id, Name: name, Arguments: `{}`}}}
}

func newService(model *scriptedLLM, ft *fakeTools, st *memStore) *Service {
	return &Service{LLM: model, Tools: ft, Store: st,
		Clock: func() time.Time { return time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC) }}
}

func TestHandle_PlainReplyIsStoredAndHistoryIsReused(t *testing.T) {
	model, st := &scriptedLLM{replies: []llm.Message{text("привет"), text("ещё раз")}}, newStore()
	svc := newService(model, &fakeTools{}, st)
	ctx := context.Background()

	out, err := svc.Handle(ctx, user, Input{Message: "  здравствуйте  "})
	if err != nil || out.Reply != "привет" || out.ConversationID == uuid.Nil || len(out.PendingActions) != 0 || len(out.ToolsUsed) != 0 {
		t.Fatalf("first turn: %+v %v", out, err)
	}
	first := model.calls[0]
	if first[0].Role != llm.RoleSystem || !strings.Contains(first[0].Content, "2026-11-14") || !strings.Contains(first[0].Content, "intern") {
		t.Fatalf("system prompt: %q", first[0].Content)
	}
	if last := first[len(first)-1]; last.Role != llm.RoleUser || last.Content != "здравствуйте" {
		t.Fatalf("user text must be trimmed: %+v", last)
	}

	if _, err := svc.Handle(ctx, user, Input{ConversationID: out.ConversationID, Message: "а дальше?"}); err != nil {
		t.Fatal(err)
	}
	second := model.calls[1] 
	if len(second) != 4 || second[1].Content != "здравствуйте" || second[2].Content != "привет" {
		t.Fatalf("history must be passed to the model: %+v", second)
	}
	if got, _ := svc.History(ctx, user.UserID, out.ConversationID, 10); len(got) != 4 {
		t.Fatalf("history: %d messages", len(got))
	}
}

func TestHandle_ToolLoopAndPendingAction(t *testing.T) {
	actionID := uuid.New()
	ft := &fakeTools{
		envs: map[string]tools.Envelope{
			"create_absence_request": {OK: true, Pending: &tools.PendingInfo{ID: actionID, Summary: "отгул"}},
		},
		actions: map[uuid.UUID]domain.Action{actionID: {ID: actionID, Tool: "create_absence_request", Status: domain.ActionPending}},
	}
	model := &scriptedLLM{replies: []llm.Message{call("c1", "create_absence_request"), text("Подтвердите заявку")}}
	svc := newService(model, ft, newStore())

	out, err := svc.Handle(context.Background(), user, Input{Message: "отгул", RequestID: "req-7"})
	if err != nil || out.Reply != "Подтвердите заявку" {
		t.Fatalf("handle: %+v %v", out, err)
	}
	if len(out.PendingActions) != 1 || out.PendingActions[0].ID != actionID || len(out.ToolsUsed) != 1 {
		t.Fatalf("the pending action must be returned to the client: %+v", out)
	}
	if ft.keys[0] != "req-7:c1" {
		t.Fatalf("idempotency key: %s", ft.keys[0])
	}
	second := model.calls[1]
	n := len(second)
	if second[n-2].ToolCalls[0].ID != "c1" || second[n-1].Role != llm.RoleTool || second[n-1].Name != "create_absence_request" || !strings.Contains(second[n-1].Content, "pending_action") {
		t.Fatalf("tool messages: %+v", second[n-2:])
	}
}

func TestHandle_GeneratedKeysDifferPerRequest(t *testing.T) {
	ft := &fakeTools{envs: map[string]tools.Envelope{"search_kb": {OK: true}}}
	model := &scriptedLLM{replies: []llm.Message{call("c1", "search_kb"), text("a"), call("c1", "search_kb"), text("b")}}
	svc := newService(model, ft, newStore())
	for i := 0; i < 2; i++ {
		if _, err := svc.Handle(context.Background(), user, Input{Message: "q"}); err != nil {
			t.Fatal(err)
		}
	}
	if ft.keys[0] == ft.keys[1] {
		t.Fatalf("without a client request id every message needs a fresh key: %v", ft.keys)
	}
}

func TestHandle_ToolLoopIsBounded(t *testing.T) {
	ft := &fakeTools{envs: map[string]tools.Envelope{"search_kb": {OK: true}}}
	model := &scriptedLLM{replies: []llm.Message{call("c", "search_kb")}}
	out, err := newService(model, ft, newStore()).Handle(context.Background(), user, Input{Message: "q"})
	if err != nil || out.Reply != fallbackReply || len(ft.calls) != maxToolRounds {
		t.Fatalf("loop guard: %+v %v calls=%d", out, err, len(ft.calls))
	}
}

func TestHandle_EmptyModelReplyFallsBack(t *testing.T) {
	out, err := newService(&scriptedLLM{replies: []llm.Message{text("   ")}}, &fakeTools{}, newStore()).
		Handle(context.Background(), user, Input{Message: "q"})
	if err != nil || out.Reply != fallbackReply {
		t.Fatalf("%+v %v", out, err)
	}
}

func TestHandle_Validation(t *testing.T) {
	svc := newService(&scriptedLLM{replies: []llm.Message{text("x")}}, &fakeTools{}, newStore())
	ctx := context.Background()
	for _, msg := range []string{"", "   ", strings.Repeat("я", maxMessageRunes+1)} {
		if _, err := svc.Handle(ctx, user, Input{Message: msg}); !errors.Is(err, domain.ErrInvalidArgs) {
			t.Fatalf("message of %d chars: %v", len(msg), err)
		}
	}
	if _, err := svc.Handle(ctx, user, Input{Message: strings.Repeat("я", maxMessageRunes)}); err != nil {
		t.Fatalf("a message at the limit is fine: %v", err)
	}
}

func TestHandle_ConversationOwnership(t *testing.T) {
	st := newStore()
	svc := newService(&scriptedLLM{replies: []llm.Message{text("x")}}, &fakeTools{}, st)
	ctx := context.Background()
	out, _ := svc.Handle(ctx, user, Input{Message: "q"})

	stranger := domain.Identity{UserID: uuid.New(), Role: domain.RoleIntern}
	if _, err := svc.Handle(ctx, stranger, Input{ConversationID: out.ConversationID, Message: "q"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("foreign conversation: %v", err)
	}
	if _, err := svc.History(ctx, stranger.UserID, out.ConversationID, 10); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("foreign history: %v", err)
	}
	if _, err := svc.Handle(ctx, user, Input{ConversationID: uuid.New(), Message: "q"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown conversation: %v", err)
	}
}

func TestHandle_Failures(t *testing.T) {
	ctx := context.Background()

	model := &scriptedLLM{err: domain.ErrLLMUnavailable}
	st := newStore()
	if _, err := newService(model, &fakeTools{}, st).Handle(ctx, user, Input{Message: "q"}); !errors.Is(err, domain.ErrLLMUnavailable) {
		t.Fatalf("llm failure: %v", err)
	}
	for _, msgs := range st.msgs {
		if len(msgs) != 1 || msgs[0].Role != llm.RoleUser {
			t.Fatalf("the user's message must be kept even when the model fails: %+v", msgs)
		}
	}

	ft := &fakeTools{err: errors.New("db down")}
	m2 := &scriptedLLM{replies: []llm.Message{call("c", "search_kb")}}
	if _, err := newService(m2, ft, newStore()).Handle(ctx, user, Input{Message: "q"}); err == nil {
		t.Fatal("an infrastructure error in a tool must abort the turn")
	}

	bad := newStore()
	bad.err = errors.New("store down")
	if _, err := newService(&scriptedLLM{replies: []llm.Message{text("x")}}, &fakeTools{}, bad).Handle(ctx, user, Input{Message: "q"}); err == nil {
		t.Fatal("store errors must propagate")
	}
}

type countingEvents struct{ n int }

func (c *countingEvents) Record(context.Context, uuid.UUID, string, any) error { c.n++; return nil }

func TestHandle_RecordsEventAndUsesRealClockByDefault(t *testing.T) {
	ev := &countingEvents{}
	svc := &Service{LLM: &scriptedLLM{replies: []llm.Message{text("ok")}}, Tools: &fakeTools{}, Store: newStore(), Events: ev}
	if _, err := svc.Handle(context.Background(), user, Input{Message: "q"}); err != nil || ev.n != 1 {
		t.Fatalf("events=%d err=%v", ev.n, err)
	}
	if time.Since(svc.now()) > time.Minute {
		t.Fatal("without a clock the service uses the current time")
	}
}

func TestHandle_HistoryIgnoresToolRoleMessages(t *testing.T) {
	st := newStore()
	model := &scriptedLLM{replies: []llm.Message{text("ok")}}
	svc := newService(model, &fakeTools{}, st)
	cid, _ := st.CreateConversation(context.Background(), user.UserID)
	_ = st.AddMessage(context.Background(), cid, "tool", `{"leak":"internal"}`)
	_ = st.AddMessage(context.Background(), cid, "user", "раньше")

	if _, err := svc.Handle(context.Background(), user, Input{ConversationID: cid, Message: "теперь"}); err != nil {
		t.Fatal(err)
	}
	for _, m := range model.calls[0] {
		if strings.Contains(m.Content, "leak") {
			t.Fatal("stored tool messages must not be replayed to the model")
		}
	}
}

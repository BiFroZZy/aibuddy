package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"aibuddy/internal/auth"
	"aibuddy/internal/domain"

	"github.com/google/uuid"
)

type memActions struct {
	mu   sync.Mutex
	list []*domain.Action
	err  error
}

func (m *memActions) Create(_ context.Context, a domain.Action) (domain.Action, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return domain.Action{}, false, m.err
	}
	for _, x := range m.list {
		if x.UserID == a.UserID && x.IdempotencyKey == a.IdempotencyKey {
			return *x, false, nil
		}
	}
	a.CreatedAt = time.Now()
	m.list = append(m.list, &a)
	return a, true, nil
}

func (m *memActions) Get(_ context.Context, userID, id uuid.UUID) (domain.Action, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, x := range m.list {
		if x.ID == id && x.UserID == userID {
			return *x, nil
		}
	}
	return domain.Action{}, domain.ErrNotFound
}

func (m *memActions) Transition(_ context.Context, id uuid.UUID, from, to domain.ActionStatus, result json.RawMessage, ext *uuid.UUID, errMsg string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, x := range m.list {
		if x.ID == id && x.Status == from {
			x.Status, x.Error = to, errMsg
			if result != nil {
				x.Result = result
			}
			if ext != nil {
				x.ExternalID = ext
			}
			return true, nil
		}
	}
	return false, nil
}

func (m *memActions) List(_ context.Context, userID uuid.UUID, limit int) ([]domain.Action, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Action
	for _, x := range m.list {
		if x.UserID == userID {
			out = append(out, *x)
		}
	}
	return out, nil
}

func (m *memActions) byStatus(s domain.ActionStatus) int {
	n := 0
	for _, x := range m.list {
		if x.Status == s {
			n++
		}
	}
	return n
}

type fakeKB struct {
	hits []domain.KBHit
	err  error
	last string
}

func (f *fakeKB) Search(_ context.Context, _ domain.Identity, q string, _ int) ([]domain.KBHit, error) {
	f.last = q
	return f.hits, f.err
}

type fakeSteps struct{ steps []domain.Step }

func (f fakeSteps) StepsFor(context.Context, uuid.UUID) ([]domain.Step, error) { return f.steps, nil }

type fakePeople struct{}

func (fakePeople) WhoToAsk(context.Context, uuid.UUID, string) ([]domain.Person, error) {
	return []domain.Person{{FullName: "Наталья", Role: "mentor"}}, nil
}

type fakeRequests struct {
	keys    []string
	created []domain.Request
	err     error
}

func (f *fakeRequests) Create(ctx context.Context, userID uuid.UUID, kind string, payload json.RawMessage, approver, key string) (uuid.UUID, error) {
	if f.err != nil {
		return uuid.Nil, f.err
	}
	f.keys = append(f.keys, key)
	q := domain.Request{ID: uuid.New(), Kind: kind, UserID: userID, Payload: payload, ApproverRole: approver, Status: domain.RequestPending}
	f.created = append(f.created, q)
	return q.ID, nil
}

func (f *fakeRequests) ListForUser(context.Context, uuid.UUID, string, int) ([]domain.Request, error) {
	return f.created, nil
}

type fakeEvents struct {
	mu    sync.Mutex
	names []string
	err   error
}

func (f *fakeEvents) Record(_ context.Context, _ uuid.UUID, name string, _ any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.names = append(f.names, name)
	return f.err
}

func (f *fakeEvents) count(name string) int {
	n := 0
	for _, x := range f.names {
		if x == name {
			n++
		}
	}
	return n
}

type env struct {
	reg      *Registry
	actions  *memActions
	kb       *fakeKB
	requests *fakeRequests
	events   *fakeEvents
	now      time.Time
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		actions:  &memActions{},
		kb:       &fakeKB{},
		requests: &fakeRequests{},
		events:   &fakeEvents{},
		now:      time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC),
	}
	e.reg = NewRegistry(Deps{
		Policy: auth.DefaultPolicy(), Actions: e.actions, KB: e.kb,
		Steps: fakeSteps{steps: []domain.Step{
			{ID: uuid.New(), Title: "A", Status: domain.StepDone, ConfirmBy: domain.ConfirmSelf},
			{ID: uuid.New(), Title: "B", Description: "do b", Status: domain.StepNotStarted, ConfirmBy: domain.ConfirmSelf},
			{ID: uuid.New(), Title: "C", Status: domain.StepWaiting, ConfirmBy: domain.ConfirmMentor},
		}},
		People: fakePeople{}, Requests: e.requests, Events: e.events,
		Clock: func() time.Time { return e.now },
	})
	return e
}

var (
	intern = domain.Identity{UserID: uuid.New(), Role: domain.RoleIntern, TeamID: uuid.New()}
	hr     = domain.Identity{UserID: uuid.New(), Role: domain.RoleHRBP}
)

func raw(s string) json.RawMessage { return json.RawMessage(s) }

func TestSpecsAreFilteredByRole(t *testing.T) {
	e := newEnv(t)
	names := func(role domain.Role) string {
		var n []string
		for _, s := range e.reg.Specs(role) {
			n = append(n, s.Name)
		}
		return strings.Join(n, ",")
	}
	if got := names(domain.RoleIntern); !strings.Contains(got, "create_absence_request") || !strings.Contains(got, "request_access") {
		t.Fatalf("intern tools: %s", got)
	}
	if got := names(domain.RoleHRBP); got != "search_kb,get_who_to_ask" {
		t.Fatalf("hr tools must be limited to reading: %s", got)
	}
	for _, s := range e.reg.Specs(domain.RoleIntern) {
		if !json.Valid(s.Parameters) {
			t.Fatalf("tool %s has invalid JSON schema", s.Name)
		}
	}
}

func TestReadTools(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	e.kb.hits = []domain.KBHit{{Title: "T", Snippet: "S"}}

	env, err := e.reg.Call(ctx, intern, "search_kb", raw(`{"query":"  отгул  "}`), "k1")
	if err != nil || !env.OK || e.kb.last != "отгул" {
		t.Fatalf("search: %+v %v (query %q)", env, err, e.kb.last)
	}
	if e.events.count("tool_called") != 1 || e.events.count("kb_miss") != 0 {
		t.Fatalf("events: %v", e.events.names)
	}

	e.kb.hits = nil
	env, _ = e.reg.Call(ctx, intern, "search_kb", raw(`{"query":"что-то"}`), "k2")
	if !env.OK || e.events.count("kb_miss") != 1 {
		t.Fatalf("a miss must be counted: %+v %v", env, e.events.names)
	}

	long := strings.Repeat("я", 400)
	if _, err := e.reg.Call(ctx, intern, "search_kb", raw(`{"query":"`+long+`"}`), "k3"); err != nil || len([]rune(e.kb.last)) != 300 {
		t.Fatalf("long queries are truncated: %v %d", err, len([]rune(e.kb.last)))
	}

	for _, bad := range []string{`{"query":""}`, `{}`, `[]`, `not json`} {
		env, err := e.reg.Call(ctx, intern, "search_kb", raw(bad), "kb"+bad)
		if err != nil || env.OK || env.Error == "" {
			t.Fatalf("args %s must produce a model-visible error: %+v %v", bad, env, err)
		}
	}

	env, _ = e.reg.Call(ctx, intern, "get_my_progress", nil, "p")
	data := env.Data.(map[string]any)
	if !env.OK || data["done"] != 1 || data["total"] != 3 || data["percent"] != 33 {
		t.Fatalf("progress: %+v", data)
	}
	if next := data["next"].(map[string]any); next["title"] != "B" {
		t.Fatalf("next step: %+v", next)
	}
	if w := data["waiting"].([]string); len(w) != 1 || w[0] != "C" {
		t.Fatalf("waiting: %+v", w)
	}

	env, _ = e.reg.Call(ctx, intern, "get_who_to_ask", raw(`{"topic":"доступ"}`), "w")
	if !env.OK {
		t.Fatalf("who: %+v", env)
	}
	env, _ = e.reg.Call(ctx, intern, "get_absence_status", raw(`null`), "s")
	if !env.OK || len(env.Data.(map[string]any)["requests"].([]map[string]any)) != 0 {
		t.Fatalf("status: %+v", env)
	}
}

func TestUnknownToolAndForbidden(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	env, err := e.reg.Call(ctx, intern, "drop_database", raw(`{}`), "k")
	if err != nil || env.OK || !strings.Contains(env.Error, "unknown tool") {
		t.Fatalf("unknown tool: %+v %v", env, err)
	}

	env, err = e.reg.Call(ctx, hr, "create_absence_request", raw(`{"date_from":"2026-12-01"}`), "k2")
	if err != nil || env.OK || !strings.Contains(env.Error, "forbidden") {
		t.Fatalf("forbidden: %+v %v", env, err)
	}
	if len(e.requests.created) != 0 || e.actions.byStatus(domain.ActionDenied) != 1 || e.events.count("tool_denied") != 1 {
		t.Fatalf("denied call must be logged and must not run: created=%d denied=%d", len(e.requests.created), e.actions.byStatus(domain.ActionDenied))
	}

	if _, err := e.reg.Call(ctx, hr, "request_access", raw(`{oops`), "k3"); err != nil {
		t.Fatalf("denied call with broken JSON: %v", err)
	}
	if e.actions.byStatus(domain.ActionDenied) != 2 || !json.Valid(e.actions.list[1].Args) {
		t.Fatal("denied action args must be valid JSON")
	}

	e.actions.err = errors.New("db down")
	if _, err := e.reg.Call(ctx, hr, "request_access", raw(`{}`), "k4"); err == nil {
		t.Fatal("audit failures must not be swallowed")
	}
}

func TestConfirmFlow_Absence(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	env, err := e.reg.Call(ctx, intern, "create_absence_request",
		raw(`{"date_from":"2026-11-20","reason":"  семья  "}`), "key-1")
	if err != nil || !env.OK || env.Pending == nil {
		t.Fatalf("propose: %+v %v", env, err)
	}
	if !strings.Contains(env.Pending.Summary, "отгул с 2026-11-20 по 2026-11-20") || !strings.Contains(env.Pending.Summary, "семья") {
		t.Fatalf("summary: %s", env.Pending.Summary)
	}
	if len(e.requests.created) != 0 {
		t.Fatal("nothing may be sent before the user confirms")
	}

	again, _ := e.reg.Call(ctx, intern, "create_absence_request", raw(`{"date_from":"2026-11-20"}`), "key-1")
	if again.Pending == nil || again.Pending.ID != env.Pending.ID || len(e.actions.list) != 1 {
		t.Fatalf("idempotency broken: %+v actions=%d", again, len(e.actions.list))
	}

	other := domain.Identity{UserID: uuid.New(), Role: domain.RoleIntern}
	if _, err := e.reg.Confirm(ctx, other, env.Pending.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("foreign confirm: %v", err)
	}

	a, err := e.reg.Confirm(ctx, intern, env.Pending.ID)
	if err != nil || a.Status != domain.ActionDone || a.ExternalID == nil {
		t.Fatalf("confirm: %+v %v", a, err)
	}
	if len(e.requests.created) != 1 || e.requests.created[0].Kind != domain.KindAbsence || e.requests.created[0].ApproverRole != "manager" {
		t.Fatalf("request not filed correctly: %+v", e.requests.created)
	}
	var result map[string]any
	_ = json.Unmarshal(a.Result, &result)
	if result["status"] != domain.RequestPending || result["request_id"] != a.ExternalID.String() {
		t.Fatalf("result is stored with the external id: %v", result)
	}

	if _, err := e.reg.Confirm(ctx, intern, a.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("double confirm: %v", err)
	}
	if _, err := e.reg.Cancel(ctx, intern, a.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("cancel after confirm: %v", err)
	}
	if len(e.requests.created) != 1 {
		t.Fatal("the request must be filed exactly once")
	}
	if e.events.count("action_confirmed") != 1 || e.events.count("action_proposed") != 2 {
		t.Fatalf("events: %v", e.events.names)
	}
}

func TestCancelFlow(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	env, _ := e.reg.Call(ctx, intern, "request_access",
		raw(`{"project":"billing","level":"admin","justification":"нужно для первой задачи"}`), "k")
	if env.Pending == nil || !strings.Contains(env.Pending.Summary, "DevOps") {
		t.Fatalf("propose: %+v", env)
	}
	a, err := e.reg.Cancel(ctx, intern, env.Pending.ID)
	if err != nil || a.Status != domain.ActionCancelled {
		t.Fatalf("cancel: %+v %v", a, err)
	}
	if _, err := e.reg.Confirm(ctx, intern, a.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("confirm after cancel: %v", err)
	}
	if _, err := e.reg.Cancel(ctx, intern, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cancel unknown: %v", err)
	}
	if len(e.requests.created) != 0 {
		t.Fatal("a cancelled action must never run")
	}
}

func TestConfirmRevalidatesAndHandlesFailures(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	env, _ := e.reg.Call(ctx, intern, "create_absence_request", raw(`{"date_from":"2026-11-15"}`), "k1")
	e.now = e.now.Add(48 * time.Hour)
	a, err := e.reg.Confirm(ctx, intern, env.Pending.ID)
	if err != nil || a.Status != domain.ActionFailed || !strings.Contains(a.Error, "past") || len(e.requests.created) != 0 {
		t.Fatalf("stale confirm: %+v %v", a, err)
	}

	e.now = e.now.Add(-48 * time.Hour)
	e.requests.err = errors.New("service unavailable")
	env, _ = e.reg.Call(ctx, intern, "create_absence_request", raw(`{"date_from":"2026-11-20"}`), "k2")
	a, err = e.reg.Confirm(ctx, intern, env.Pending.ID)
	if err != nil || a.Status != domain.ActionFailed || a.Error == "" || e.events.count("action_failed") != 2 {
		t.Fatalf("service failure: %+v %v", a, err)
	}
}

func TestConfirmGuards(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	env, _ := e.reg.Call(ctx, intern, "create_absence_request", raw(`{"date_from":"2026-11-20"}`), "k")

	demoted := intern
	demoted.Role = domain.RoleHRBP
	if _, err := e.reg.Confirm(ctx, demoted, env.Pending.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("role change: %v", err)
	}

	e.actions.list = append(e.actions.list, &domain.Action{ID: uuid.New(), UserID: intern.UserID, Tool: "search_kb", Status: domain.ActionPending})
	if _, err := e.reg.Confirm(ctx, intern, e.actions.list[1].ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("non-mutating tool: %v", err)
	}
}

func TestValidation(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	tests := []struct {
		name, tool, args, wantErr string
	}{
		{"absence: no date", "create_absence_request", `{}`, "date_from is required"},
		{"absence: bad date", "create_absence_request", `{"date_from":"20.11"}`, "date_from must look like"},
		{"absence: bad end date", "create_absence_request", `{"date_from":"2026-11-20","date_to":"soon"}`, "date_to must look like"},
		{"absence: reversed", "create_absence_request", `{"date_from":"2026-11-21","date_to":"2026-11-20"}`, "before date_from"},
		{"absence: past", "create_absence_request", `{"date_from":"2026-11-13"}`, "in the past"},
		{"absence: too long", "create_absence_request", `{"date_from":"2026-11-20","date_to":"2026-12-25"}`, "30 days"},
		{"absence: bad kind", "create_absence_request", `{"kind":"sabbatical","date_from":"2026-11-20"}`, "kind must be"},
		{"absence: long reason", "create_absence_request", `{"date_from":"2026-11-20","reason":"` + strings.Repeat("я", 501) + `"}`, "500"},
		{"absence: not an object", "create_absence_request", `"2026-11-20"`, "JSON object"},
		{"access: no project", "request_access", `{"justification":"нужно для задач"}`, "project is required"},
		{"access: bad level", "request_access", `{"project":"x","level":"root","justification":"нужно для задач"}`, "level must be"},
		{"access: short justification", "request_access", `{"project":"x","justification":"надо"}`, "justification"},
	}
	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env, err := e.reg.Call(ctx, intern, tc.tool, raw(tc.args), "v"+string(rune('a'+i)))
			if err != nil || env.OK || !strings.Contains(env.Error, tc.wantErr) {
				t.Fatalf("want error containing %q, got %+v %v", tc.wantErr, env, err)
			}
		})
	}
	if len(e.actions.list) != 0 {
		t.Fatal("invalid calls must not create actions")
	}
}

func TestValidation_Accepts(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	for i, args := range []string{
		`{"date_from":"2026-11-14"}`,
		`{"kind":"vacation","date_from":"2026-12-01","date_to":"2026-12-14","reason":"отдых"}`,
		`{"kind":"sick","date_from":"2026-11-20","date_to":"2026-12-20"}`,
	} {
		env, err := e.reg.Call(ctx, intern, "create_absence_request", raw(args), "ok"+string(rune('a'+i)))
		if err != nil || env.Pending == nil {
			t.Fatalf("%s: %+v %v", args, env, err)
		}
	}
	a := e.actions.list[0]
	var norm absenceArgs
	_ = json.Unmarshal(a.Args, &norm)
	if norm.Kind != "day_off" || norm.DateTo != "2026-11-14" {
		t.Fatalf("defaults must be filled in: %+v", norm)
	}
}

func TestAccessApproverByLevel(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	want := map[string]string{"read": "project_owner", "write": "manager", "admin": "devops"}
	for level, approver := range want {
		env, _ := e.reg.Call(ctx, intern, "request_access",
			raw(`{"project":"svc","level":"`+level+`","justification":"нужно для первой задачи"}`), "lvl-"+level)
		if env.Pending == nil {
			t.Fatalf("%s: %+v", level, env)
		}
		if _, err := e.reg.Confirm(ctx, intern, env.Pending.ID); err != nil {
			t.Fatal(err)
		}
		got := e.requests.created[len(e.requests.created)-1]
		if got.Kind != domain.KindAccess || got.ApproverRole != approver {
			t.Fatalf("%s: approver %s, want %s", level, got.ApproverRole, approver)
		}
	}
	env, _ := e.reg.Call(ctx, intern, "request_access", raw(`{"project":"svc","justification":"нужно для первой задачи"}`), "dflt")
	if !strings.Contains(env.Pending.Summary, "чтение") {
		t.Fatalf("default level: %s", env.Pending.Summary)
	}
}

func TestActionListingAndSummaries(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	env, _ := e.reg.Call(ctx, intern, "create_absence_request", raw(`{"date_from":"2026-11-20"}`), "k")
	_, _ = e.reg.Call(ctx, intern, "search_kb", raw(`{"query":"x"}`), "k2")

	a, err := e.reg.GetAction(ctx, intern.UserID, env.Pending.ID)
	if err != nil || !strings.Contains(a.Summary, "2026-11-20") {
		t.Fatalf("get action: %+v %v", a, err)
	}
	if _, err := e.reg.GetAction(ctx, uuid.New(), env.Pending.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("foreign get: %v", err)
	}

	if _, err := e.reg.Confirm(ctx, intern, a.ID); err != nil {
		t.Fatal(err)
	}
	e.now = e.now.AddDate(0, 1, 0)
	list, err := e.reg.ListActions(ctx, intern.UserID, 10)
	if err != nil || len(list) != 1 || !strings.Contains(list[0].Summary, "2026-11-20") {
		t.Fatalf("list: %+v %v", list, err)
	}
}

func TestEventFailuresAreNotFatal(t *testing.T) {
	e := newEnv(t)
	e.events.err = errors.New("analytics down")
	env, err := e.reg.Call(context.Background(), intern, "get_my_progress", nil, "k")
	if err != nil || !env.OK {
		t.Fatalf("analytics must never break a tool call: %+v %v", env, err)
	}
	reg := NewRegistry(Deps{Policy: auth.DefaultPolicy(), Actions: e.actions, KB: e.kb, Steps: fakeSteps{}, People: fakePeople{}, Requests: e.requests})
	if env, err := reg.Call(context.Background(), intern, "get_my_progress", nil, "k"); err != nil || !env.OK {
		t.Fatalf("a registry without events/log/clock must work: %+v %v", env, err)
	}
}

func TestConfirmUsesActionIDAsIdempotencyKey(t *testing.T) {
	e := newEnv(t)
	env, _ := e.reg.Call(context.Background(), intern, "create_absence_request", raw(`{"date_from":"2026-11-20"}`), "k")
	if _, err := e.reg.Confirm(context.Background(), intern, env.Pending.ID); err != nil {
		t.Fatal(err)
	}
	if len(e.requests.keys) != 1 || e.requests.keys[0] != env.Pending.ID.String() {
		t.Fatalf("the corporate service must get the action id as the idempotency key: %v", e.requests.keys)
	}
}

type slowRequests struct{ fakeRequests }

func (s *slowRequests) Create(ctx context.Context, _ uuid.UUID, _ string, _ json.RawMessage, _, _ string) (uuid.UUID, error) {
	<-ctx.Done()
	return uuid.Nil, ctx.Err()
}

func TestConfirmTimesOutOnHungService(t *testing.T) {
	e := newEnv(t)
	reg := NewRegistry(Deps{Policy: auth.DefaultPolicy(), Actions: e.actions, KB: e.kb, Steps: fakeSteps{}, People: fakePeople{},
		Requests: &slowRequests{}, Clock: func() time.Time { return e.now }, ToolTimeout: 20 * time.Millisecond})
	env, _ := reg.Call(context.Background(), intern, "create_absence_request", raw(`{"date_from":"2026-11-20"}`), "k")
	a, err := reg.Confirm(context.Background(), intern, env.Pending.ID)
	if err != nil || a.Status != domain.ActionFailed {
		t.Fatalf("a hung service must fail the action, not hang the request: %+v %v", a, err)
	}
}

package tools_test

import (
	"aibuddy/internal/auth"
	"aibuddy/internal/clients/hr"
	"aibuddy/internal/domain"
	"aibuddy/internal/tools"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"testing"
	"time"
)

type hrFake struct {
	req hr.CreateAbsenceReq
	key string
	res hr.Absence
	err error
}

func (f *hrFake) CreateAbsence(_ context.Context, r hr.CreateAbsenceReq, k string) (hr.Absence, error) {
	f.req = r
	f.key = k
	return f.res, f.err
}
func (f *hrFake) GetAbsence(context.Context, string) (hr.Absence, error) { return f.res, f.err }

type repoFake struct {
	items               map[uuid.UUID]domain.Action
	claimed             bool
	finished            domain.ActionStatus
	createErr, claimErr error
}

func newRepo() *repoFake { return &repoFake{items: map[uuid.UUID]domain.Action{}} }
func (f *repoFake) Create(_ context.Context, a domain.Action) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.items[a.ID] = a
	return nil
}
func (f *repoFake) Get(_ context.Context, id uuid.UUID) (domain.Action, error) {
	a, ok := f.items[id]
	if !ok {
		return domain.Action{}, domain.ErrNotFound
	}
	return a, nil
}
func (f *repoFake) Claim(_ context.Context, id uuid.UUID) (bool, error) {
	if f.claimErr != nil {
		return false, f.claimErr
	}
	a, ok := f.items[id]
	if f.claimed || !ok || a.Status != domain.ActionPending {
		return false, nil
	}
	f.claimed = true
	a.Status = domain.ActionRunning
	f.items[id] = a
	return true, nil
}
func (f *repoFake) Finish(_ context.Context, id uuid.UUID, s domain.ActionStatus, _ json.RawMessage, _ uuid.UUID, _ string) error {
	f.finished = s
	a := f.items[id]
	a.Status = s
	f.items[id] = a
	return nil
}

type eventsFake struct{ names []string }

func (f *eventsFake) Emit(_ context.Context, _ uuid.UUID, n string, _ any) {
	f.names = append(f.names, n)
}

type toolFake struct {
	name                string
	confirm             bool
	validation, execErr error
	calls               int
}

func (f *toolFake) Name() string                   { return f.name }
func (f *toolFake) Description() string            { return "test" }
func (f *toolFake) Schema() json.RawMessage        { return json.RawMessage(`{"type":"object"}`) }
func (f *toolFake) RequiresConfirmation() bool     { return f.confirm }
func (f *toolFake) Validate(json.RawMessage) error { return f.validation }
func (f *toolFake) Execute(context.Context, domain.Identity, json.RawMessage, string) (tools.Output, error) {
	f.calls++
	if f.execErr != nil {
		return tools.Output{}, f.execErr
	}
	return tools.Output{Data: json.RawMessage(`{"ok":true}`)}, nil
}

var good = json.RawMessage(`{"type":"day_off","date_from":"2026-12-01","date_to":"2026-12-02"}`)

func makeExec(ft *toolFake, r *repoFake, ev *eventsFake) *tools.Executor {
	return &tools.Executor{Reg: tools.NewRegistry(ft), Policy: auth.DefaultPolicy(), Repo: r, Events: ev, Timeout: time.Second}
}
func userctx(id uuid.UUID) context.Context {
	return auth.WithIdentity(context.Background(), domain.Identity{UserID: id, Role: domain.RoleIntern})
}
func TestRegistryFiltersAndSorts(t *testing.T) {
	r := tools.NewRegistry(&toolFake{name: "search_kb"}, &toolFake{name: "get_my_progress"}, &toolFake{name: "create_absence_request"})
	s := r.Specs(domain.RoleMentor, auth.DefaultPolicy())
	if len(s) != 2 || s[0].Name != "get_my_progress" || s[1].Name != "search_kb" {
		t.Fatalf("%#v", s)
	}
	if _, ok := r.Get("unknown"); ok {
		t.Fatal("unexpected tool")
	}
}
func TestAbsenceValidation(t *testing.T) {
	cases := []struct {
		raw string
		bad bool
	}{{string(good), false}, {`{"type":"sick","date_from":"2026-12-01","date_to":"2026-12-02"}`, true}, {`{"type":"day_off","date_from":"bad","date_to":"2026-12-02"}`, true}, {`{"type":"day_off","date_from":"2026-12-03","date_to":"2026-12-02"}`, true}, {`{"type":"day_off","date_from":"2026-12-01","date_to":"2026-12-02","extra":1}`, true}, {`{} {}`, true}}
	for _, c := range cases {
		err := (tools.CreateAbsence{}).Validate(json.RawMessage(c.raw))
		if (err != nil) != c.bad {
			t.Errorf("Validate(%s): %v", c.raw, err)
		}
	}
}
func TestAbsenceExecute(t *testing.T) {
	uid, xid := uuid.New(), uuid.New()
	f := &hrFake{res: hr.Absence{ID: xid}}
	out, err := (tools.CreateAbsence{HR: f}).Execute(context.Background(), domain.Identity{UserID: uid}, good, "key")
	if err != nil || f.req.UserID != uid || f.key != "key" || out.ExternalID != xid || !json.Valid(out.Data) {
		t.Fatalf("out=%#v err=%v req=%#v", out, err, f.req)
	}
}
func TestExecutorConfirmationFlowAndDuplicate(t *testing.T) {
	uid := uuid.New()
	ft := &toolFake{name: "create_absence_request", confirm: true}
	r := newRepo()
	ev := &eventsFake{}
	e := makeExec(ft, r, ev)
	c := userctx(uid)
	pending, err := e.Run(c, ft.name, good)
	if err != nil || pending.Status != domain.ActionPending || ft.calls != 0 {
		t.Fatalf("%#v %v", pending, err)
	}
	done, err := e.Confirm(c, pending.ActionID)
	if err != nil || done.Status != domain.ActionDone || ft.calls != 1 || r.finished != domain.ActionDone {
		t.Fatalf("%#v %v", done, err)
	}
	if _, err = e.Confirm(c, pending.ActionID); !errors.Is(err, domain.ErrAlreadyHandled) {
		t.Fatalf("repeat: %v", err)
	}
}
func TestExecutorDenialsAndErrors(t *testing.T) {
	ft := &toolFake{name: "search_kb"}
	e := makeExec(ft, newRepo(), &eventsFake{})
	if _, err := e.Run(context.Background(), ft.name, good); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatal(err)
	}
	if _, err := e.Run(userctx(uuid.New()), "missing", good); !errors.Is(err, domain.ErrUnknownTool) {
		t.Fatal(err)
	}
	restricted := &toolFake{name: "create_absence_request"}
	ctx := auth.WithIdentity(context.Background(), domain.Identity{UserID: uuid.New(), Role: domain.RoleHRBP})
	if _, err := makeExec(restricted, newRepo(), &eventsFake{}).Run(ctx, restricted.name, good); !errors.Is(err, domain.ErrForbidden) || restricted.calls > 0 {
		t.Fatalf("%v", err)
	}
	invalid := &toolFake{name: "search_kb", validation: errors.New("invalid")}
	if _, err := makeExec(invalid, newRepo(), &eventsFake{}).Run(userctx(uuid.New()), invalid.name, good); !errors.Is(err, domain.ErrInvalidArgs) {
		t.Fatal(err)
	}
	broken := &toolFake{name: "search_kb", execErr: errors.New("down")}
	repo := newRepo()
	res, err := makeExec(broken, repo, &eventsFake{}).Run(userctx(uuid.New()), broken.name, good)
	if err == nil || res.Status != domain.ActionFailed || repo.finished != domain.ActionFailed {
		t.Fatalf("%#v %v", res, err)
	}
}
func TestExecutorConfirmOwnerAndClaimFailure(t *testing.T) {
	uid := uuid.New()
	ft := &toolFake{name: "create_absence_request", confirm: true}
	r := newRepo()
	e := makeExec(ft, r, &eventsFake{})
	pending, err := e.Run(userctx(uid), ft.name, good)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.Confirm(userctx(uuid.New()), pending.ActionID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	r.claimErr = errors.New("db")
	if _, err = e.Confirm(userctx(uid), pending.ActionID); err == nil {
		t.Fatal("expected repo error")
	}
}

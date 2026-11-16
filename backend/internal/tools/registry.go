package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"aibuddy/internal/auth"
	"aibuddy/internal/domain"
	"aibuddy/internal/llm"

	"github.com/google/uuid"
)

type Envelope struct {
	OK      bool         `json:"ok"`
	Data    any          `json:"data,omitempty"`
	Pending *PendingInfo `json:"pending_action,omitempty"`
	Error   string       `json:"error,omitempty"`
}

type PendingInfo struct {
	ID      uuid.UUID `json:"id"`
	Summary string    `json:"summary"`
}

type ActionStore interface {
	Create(ctx context.Context, a domain.Action) (stored domain.Action, created bool, err error)
	Get(ctx context.Context, userID, id uuid.UUID) (domain.Action, error)
	Transition(ctx context.Context, id uuid.UUID, from, to domain.ActionStatus, result json.RawMessage, externalID *uuid.UUID, errMsg string) (ok bool, err error)
	List(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Action, error)
}

type KBSearcher interface {
	Search(ctx context.Context, id domain.Identity, query string, limit int) ([]domain.KBHit, error)
}

type StepReader interface {
	StepsFor(ctx context.Context, userID uuid.UUID) ([]domain.Step, error)
}

type PeopleFinder interface {
	WhoToAsk(ctx context.Context, userID uuid.UUID, topic string) ([]domain.Person, error)
}

type RequestService interface {
	Create(ctx context.Context, userID uuid.UUID, kind string, payload json.RawMessage, approverRole, idempotencyKey string) (uuid.UUID, error)
	ListForUser(ctx context.Context, userID uuid.UUID, kind string, limit int) ([]domain.Request, error)
}

type EventRecorder interface {
	Record(ctx context.Context, userID uuid.UUID, name string, payload any) error
}

type Deps struct {
	Policy      auth.Policy
	Actions     ActionStore
	KB          KBSearcher
	Steps       StepReader
	People      PeopleFinder
	Requests    RequestService
	Events      EventRecorder
	Log         *slog.Logger
	Clock       func() time.Time
	ToolTimeout time.Duration
}

type idemKey struct{}

func idempotencyKeyFrom(ctx context.Context) string {
	k, _ := ctx.Value(idemKey{}).(string)
	return k
}

type tool struct {
	spec     llm.ToolSpec
	mutating bool
	prepare  func(args json.RawMessage, now time.Time) (json.RawMessage, string, error)
	run      func(ctx context.Context, id domain.Identity, args json.RawMessage) (data any, externalID *uuid.UUID, err error)
}

type Registry struct {
	d     Deps
	tools map[string]*tool
	order []string
}

func NewRegistry(d Deps) *Registry {
	if d.Clock == nil {
		d.Clock = time.Now
	}
	if d.ToolTimeout <= 0 {
		d.ToolTimeout = 10 * time.Second
	}
	if d.Log == nil {
		d.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	r := &Registry{d: d, tools: map[string]*tool{}}
	for _, t := range r.definitions() {
		r.tools[t.spec.Name] = t
		r.order = append(r.order, t.spec.Name)
	}
	return r
}

func (r *Registry) Specs(role domain.Role) []llm.ToolSpec {
	var out []llm.ToolSpec
	for _, name := range r.order {
		if r.d.Policy.Allowed(role, name) {
			out = append(out, r.tools[name].spec)
		}
	}
	return out
}

func (r *Registry) Call(ctx context.Context, id domain.Identity, name string, args json.RawMessage, key string) (Envelope, error) {
	t, ok := r.tools[name]
	if !ok {
		return Envelope{Error: "unknown tool: " + name}, nil
	}
	args = normalizeArgs(args)

	if !r.d.Policy.Allowed(id.Role, name) {
		_, _, err := r.d.Actions.Create(ctx, domain.Action{
			ID: uuid.New(), UserID: id.UserID, Tool: name, Args: asJSONObject(args),
			Status: domain.ActionDenied, IdempotencyKey: "denied:" + key,
			Error: "role " + string(id.Role) + " may not use this tool",
		})
		if err != nil {
			return Envelope{}, fmt.Errorf("log denied action: %w", err)
		}
		r.event(ctx, id.UserID, "tool_denied", map[string]any{"tool": name})
		return Envelope{Error: "forbidden: this operation is not available for your role"}, nil
	}

	if !t.mutating {
		data, _, err := t.run(ctx, id, args)
		if err != nil {
			return r.domainOrFail(err)
		}
		r.event(ctx, id.UserID, "tool_called", map[string]any{"tool": name})
		if name == auth.PermSearchKB && hitsCount(data) == 0 {
			r.event(ctx, id.UserID, "kb_miss", map[string]any{})
		}
		return Envelope{OK: true, Data: data}, nil
	}

	norm, summary, err := t.prepare(args, r.d.Clock())
	if err != nil {
		return r.domainOrFail(err)
	}
	stored, _, err := r.d.Actions.Create(ctx, domain.Action{
		ID: uuid.New(), UserID: id.UserID, Tool: name, Args: norm,
		Status: domain.ActionPending, IdempotencyKey: key,
	})
	if err != nil {
		return Envelope{}, fmt.Errorf("store action: %w", err)
	}
	r.event(ctx, id.UserID, "action_proposed", map[string]any{"tool": name})
	if stored.Status != domain.ActionPending {
		return Envelope{OK: true, Data: map[string]any{"action_id": stored.ID, "status": stored.Status}}, nil
	}
	return Envelope{OK: true, Pending: &PendingInfo{ID: stored.ID, Summary: summary}}, nil
}

func (r *Registry) Confirm(ctx context.Context, id domain.Identity, actionID uuid.UUID) (domain.Action, error) {
	a, err := r.d.Actions.Get(ctx, id.UserID, actionID)
	if err != nil {
		return domain.Action{}, err
	}
	t, ok := r.tools[a.Tool]
	if !ok || !t.mutating {
		return domain.Action{}, domain.ErrNotFound
	}
	if !r.d.Policy.Allowed(id.Role, a.Tool) {
		return domain.Action{}, domain.ErrForbidden
	}
	if a.Status != domain.ActionPending {
		return domain.Action{}, fmt.Errorf("%w: action is %s", domain.ErrConflict, a.Status)
	}
	moved, err := r.d.Actions.Transition(ctx, a.ID, domain.ActionPending, domain.ActionRunning, nil, nil, "")
	if err != nil {
		return domain.Action{}, err
	}
	if !moved {
		return domain.Action{}, fmt.Errorf("%w: action was already handled", domain.ErrConflict)
	}

	args, _, err := t.prepare(a.Args, r.d.Clock())
	var data any
	var ext *uuid.UUID
	if err == nil {
		runCtx, cancel := context.WithTimeout(context.WithValue(ctx, idemKey{}, a.ID.String()), r.d.ToolTimeout)
		data, ext, err = t.run(runCtx, id, args)
		cancel()
	}
	if err != nil {
		if _, terr := r.d.Actions.Transition(ctx, a.ID, domain.ActionRunning, domain.ActionFailed, nil, nil, err.Error()); terr != nil {
			return domain.Action{}, terr
		}
		r.event(ctx, id.UserID, "action_failed", map[string]any{"tool": a.Tool})
		return r.d.Actions.Get(ctx, id.UserID, a.ID)
	}
	result, _ := json.Marshal(data)
	if _, err := r.d.Actions.Transition(ctx, a.ID, domain.ActionRunning, domain.ActionDone, result, ext, ""); err != nil {
		return domain.Action{}, err
	}
	r.event(ctx, id.UserID, "action_confirmed", map[string]any{"tool": a.Tool})
	return r.d.Actions.Get(ctx, id.UserID, a.ID)
}

func (r *Registry) Cancel(ctx context.Context, id domain.Identity, actionID uuid.UUID) (domain.Action, error) {
	a, err := r.d.Actions.Get(ctx, id.UserID, actionID)
	if err != nil {
		return domain.Action{}, err
	}
	moved, err := r.d.Actions.Transition(ctx, a.ID, domain.ActionPending, domain.ActionCancelled, nil, nil, "")
	if err != nil {
		return domain.Action{}, err
	}
	if !moved {
		return domain.Action{}, fmt.Errorf("%w: action is %s", domain.ErrConflict, a.Status)
	}
	r.event(ctx, id.UserID, "action_cancelled", map[string]any{"tool": a.Tool})
	return r.d.Actions.Get(ctx, id.UserID, a.ID)
}

func (r *Registry) GetAction(ctx context.Context, userID, actionID uuid.UUID) (domain.Action, error) {
	a, err := r.d.Actions.Get(ctx, userID, actionID)
	if err != nil {
		return domain.Action{}, err
	}
	r.fillSummary(&a)
	return a, nil
}

func (r *Registry) ListActions(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Action, error) {
	list, err := r.d.Actions.List(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	for i := range list {
		r.fillSummary(&list[i])
	}
	return list, nil
}

func (r *Registry) fillSummary(a *domain.Action) {
	if t, ok := r.tools[a.Tool]; ok && t.mutating {
		if _, s, err := t.prepare(a.Args, r.d.Clock().AddDate(-1, 0, 0)); err == nil {
			a.Summary = s
		}
	}
}

func (r *Registry) domainOrFail(err error) (Envelope, error) {
	if errors.Is(err, domain.ErrInvalidArgs) {
		return Envelope{Error: err.Error()}, nil
	}
	return Envelope{}, err
}

func (r *Registry) event(ctx context.Context, userID uuid.UUID, name string, payload any) {
	if r.d.Events == nil {
		return
	}
	if err := r.d.Events.Record(ctx, userID, name, payload); err != nil {
		r.d.Log.Warn("record event failed", "event", name, "err", err)
	}
}

func normalizeArgs(raw json.RawMessage) json.RawMessage {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return json.RawMessage("{}")
	}
	return raw
}

func asJSONObject(raw json.RawMessage) json.RawMessage {
	if json.Valid(raw) {
		return raw
	}
	b, _ := json.Marshal(map[string]string{"raw": string(raw)})
	return b
}

func hitsCount(data any) int {
	m, ok := data.(map[string]any)
	if !ok {
		return -1
	}
	if h, ok := m["hits"].([]domain.KBHit); ok {
		return len(h)
	}
	return -1
}

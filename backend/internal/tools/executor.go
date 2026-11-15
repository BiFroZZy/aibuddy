package tools

import (
	"aibuddy/internal/auth"
	"aibuddy/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ActionRepo interface {
	Create(ctx context.Context, a domain.Action) error
	Get(ctx context.Context, id uuid.UUID) (domain.Action, error)
	Claim(ctx context.Context, id uuid.UUID) (bool, error)
	Finish(ctx context.Context, id uuid.UUID, st domain.ActionStatus, result json.RawMessage, externalID uuid.UUID, errText string) error
}

type Events interface {
	Emit(ctx context.Context, userID uuid.UUID, name string, payload any)
}

type Result struct {
	Status     domain.ActionStatus `json:"status"`
	ActionID   uuid.UUID           `json:"action_id"`
	Data       json.RawMessage     `json:"data,omitempty"`
	ExternalID uuid.UUID           `json:"external_id,omitempty"`
}

type Executor struct {
	Reg     *Registry
	Policy  auth.Policy
	Repo    ActionRepo
	Events  Events
	Timeout time.Duration
}

func (e *Executor) Run(ctx context.Context, toolName string, args json.RawMessage) (Result, error) {
	id, ok := auth.IdentityFrom(ctx)
	if !ok {
		return Result{}, domain.ErrUnauthorized
	}
	t, ok := e.Reg.Get(toolName)
	if !ok {
		return Result{}, domain.ErrUnknownTool
	}

	actID, err := uuid.NewUUID()
	if err != nil {
		return Result{}, err
	}
	act := domain.Action{
		ID: actID, UserID: id.UserID, Tool: toolName, Args: args,
		IdempotencyKey: actID.String(), CreatedAt: time.Now(),
	}

	if !e.Policy.Allowed(id.Role, toolName) {
		act.Status = domain.ActionDenied
		_ = e.Repo.Create(ctx, act)
		e.Events.Emit(ctx, id.UserID, "tool_denied", map[string]string{"tool": toolName})
		return Result{Status: domain.ActionDenied, ActionID: actID}, domain.ErrForbidden
	}

	if err := t.Validate(args); err != nil {
		return Result{}, fmt.Errorf("%w: %v", domain.ErrInvalidArgs, err)
	}

	if t.RequiresConfirmation() {
		act.Status = domain.ActionPending
		if err := e.Repo.Create(ctx, act); err != nil {
			return Result{}, err
		}
		e.Events.Emit(ctx, id.UserID, "action_proposed", map[string]string{"tool": toolName})
		return Result{Status: domain.ActionPending, ActionID: actID, Data: args}, nil
	}

	act.Status = domain.ActionRunning
	if err := e.Repo.Create(ctx, act); err != nil {
		return Result{}, err
	}
	return e.execute(ctx, t, id, act)
}

func (e *Executor) Confirm(ctx context.Context, actionID uuid.UUID) (Result, error) {
	id, ok := auth.IdentityFrom(ctx)
	if !ok {
		return Result{}, domain.ErrUnauthorized
	}
	act, err := e.Repo.Get(ctx, actionID)
	if err != nil {
		return Result{}, err
	}

	if act.UserID != id.UserID {
		return Result{}, domain.ErrNotFound
	}
	t, ok := e.Reg.Get(act.Tool)
	if !ok {
		return Result{}, domain.ErrUnknownTool
	}

	if !e.Policy.Allowed(id.Role, act.Tool) {
		return Result{Status: domain.ActionDenied, ActionID: act.ID}, domain.ErrForbidden
	}
	if err := t.Validate(act.Args); err != nil {
		return Result{}, fmt.Errorf("%w: %v", domain.ErrInvalidArgs, err)
	}
	claimed, err := e.Repo.Claim(ctx, act.ID)
	if err != nil {
		return Result{}, err
	}
	if !claimed {
		return Result{}, domain.ErrAlreadyHandled
	}
	return e.execute(ctx, t, id, act)
}

func (e *Executor) execute(ctx context.Context, t Tool, id domain.Identity, act domain.Action) (Result, error) {
	cctx, cancel := context.WithTimeout(ctx, e.Timeout)
	defer cancel()

	out, err := t.Execute(cctx, id, act.Args, act.IdempotencyKey)
	if err != nil {
		_ = e.Repo.Finish(ctx, act.ID, domain.ActionFailed, nil, uuid.Nil, err.Error())
		e.Events.Emit(ctx, id.UserID, "action_failed", map[string]string{"tool": act.Tool})
		return Result{Status: domain.ActionFailed, ActionID: act.ID}, err
	}

	if err := e.Repo.Finish(ctx, act.ID, domain.ActionDone, out.Data, out.ExternalID, ""); err != nil {
		return Result{}, err
	}
	e.Events.Emit(ctx, id.UserID, "action_done", map[string]string{"tool": act.Tool})
	return Result{Status: domain.ActionDone, ActionID: act.ID, Data: out.Data, ExternalID: out.ExternalID}, nil
}

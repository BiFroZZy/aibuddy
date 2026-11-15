package tools

import (
	"aibuddy/internal/auth"
	"aibuddy/internal/clients/ml"
	"aibuddy/internal/domain"
	"context"
	"encoding/json"
	"sort"

	"github.com/google/uuid"
)

type Output struct {
	Data       json.RawMessage
	ExternalID uuid.UUID
}

type Tool interface {
	Name() string
	Description() string 
	Schema() json.RawMessage
	RequiresConfirmation() bool
	Validate(args json.RawMessage) error
	Execute(ctx context.Context, id domain.Identity, args json.RawMessage, idemKey string) (Output, error)
}

type Registry struct {
	byName map[string]Tool
}

func NewRegistry(ts ...Tool) *Registry {
	r := &Registry{byName: make(map[string]Tool, len(ts))}
	for _, t := range ts {
		r.byName[t.Name()] = t
	}
	return r
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.byName[name]
	return t, ok
}

func (r *Registry) Specs(role domain.Role, p auth.Policy) []ml.ToolSpec {
	var out []ml.ToolSpec
	for name, t := range r.byName {
		if p.Allowed(role, name) {
			out = append(out, ml.ToolSpec{Name: name, Description: t.Description(), Schema: t.Schema()})
		}
	}
	sort.Slice(out, func(i, j int) bool {return out[i].Name < out[j].Name})
	return out
}
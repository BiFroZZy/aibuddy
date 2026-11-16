package httptransport

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"aibuddy/internal/auth"
	"aibuddy/internal/chat"
	"aibuddy/internal/domain"
	"aibuddy/internal/progress"

	"github.com/google/uuid"
)

type ChatService interface {
	Handle(ctx context.Context, id domain.Identity, in chat.Input) (chat.Output, error)
	History(ctx context.Context, userID, convID uuid.UUID, limit int) ([]domain.ChatMessage, error)
}

type ActionService interface {
	Confirm(ctx context.Context, id domain.Identity, actionID uuid.UUID) (domain.Action, error)
	Cancel(ctx context.Context, id domain.Identity, actionID uuid.UUID) (domain.Action, error)
	ListActions(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Action, error)
}

type ProgressService interface {
	Get(ctx context.Context, userID uuid.UUID) (progress.Progress, error)
	SetStatus(ctx context.Context, userID, stepID uuid.UUID, to domain.StepStatus) (progress.Progress, error)
	Confirmations(ctx context.Context, mentorID uuid.UUID) ([]domain.Confirmation, error)
	Decide(ctx context.Context, mentorID, internID, stepID uuid.UUID, approve bool) error
}

type RequestStore interface {
	ListForUser(ctx context.Context, userID uuid.UUID, kind string, limit int) ([]domain.Request, error)
	PendingForManager(ctx context.Context, managerID uuid.UUID) ([]domain.Request, error)
	Decide(ctx context.Context, managerID, id uuid.UUID, approve bool, comment string) (domain.Request, error)
}

type MetricsStore interface {
	Metrics(ctx context.Context, since time.Time) (domain.Metrics, error)
}

type Deps struct {
	Log        *slog.Logger
	DB         Pinger
	Auth       auth.Authenticator
	Policy     auth.Policy
	Chat       ChatService
	Actions    ActionService
	Progress   ProgressService
	Requests   RequestStore
	Metrics    MetricsStore
	CORSOrigin string
	Clock      func() time.Time
}

func NewRouter(d Deps) http.Handler {
	if d.Clock == nil {
		d.Clock = time.Now
	}
	h := &handlers{d: d}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", Healthz)
	mux.Handle("GET /readyz", Readyz(d.DB, d.Log))

	authed := RequireAuth(d.Auth, d.Log)
	route := func(pattern, perm string, fn http.HandlerFunc) {
		var handler http.Handler = fn
		if perm != "" {
			handler = RequirePermission(d.Policy, perm)(handler)
		}
		mux.Handle(pattern, authed(handler))
	}

	route("GET /me", "", Me)
	route("GET /api/me", "", Me)

	route("POST /api/chat", "", h.chat)
	route("GET /api/conversations/{id}/messages", "", h.history)

	route("GET /api/progress", auth.PermManageOwnSteps, h.progress)
	route("POST /api/progress/steps/{id}/status", auth.PermManageOwnSteps, h.setStepStatus)

	route("GET /api/actions", "", h.listActions)
	route("POST /api/actions/{id}/confirm", "", h.confirmAction)
	route("POST /api/actions/{id}/cancel", "", h.cancelAction)
	route("GET /api/requests", "", h.myRequests)

	route("GET /api/mentor/confirmations", auth.PermConfirmStep, h.confirmations)
	route("POST /api/mentor/interns/{internID}/steps/{stepID}/decision", auth.PermConfirmStep, h.decideStep)

	route("GET /api/manager/requests", auth.PermDecideRequest, h.pendingRequests)
	route("POST /api/manager/requests/{id}/decision", auth.PermDecideRequest, h.decideRequest)

	route("GET /api/metrics", auth.PermViewMetrics, h.metrics)

	return Chain(mux, CORS(d.CORSOrigin), Logging(d.Log), Recover(d.Log))
}

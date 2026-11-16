package app

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"aibuddy/internal/auth"
	"aibuddy/internal/chat"
	"aibuddy/internal/clients/hr"
	"aibuddy/internal/clients/ml"
	"aibuddy/internal/config"
	"aibuddy/internal/domain"
	"aibuddy/internal/llm"
	"aibuddy/internal/migrate"
	"aibuddy/internal/progress"
	"aibuddy/internal/repository"
	"aibuddy/internal/tools"
	httptransport "aibuddy/internal/transport/http"
	"aibuddy/migrations"
)

var (
	_ tools.ActionStore          = (*repository.Actions)(nil)
	_ tools.KBSearcher           = (*repository.KB)(nil)
	_ tools.StepReader           = (*repository.Steps)(nil)
	_ tools.PeopleFinder         = (*repository.People)(nil)
	_ tools.RequestService       = (*repository.Requests)(nil)
	_ tools.EventRecorder        = (*repository.Events)(nil)
	_ progress.Store             = (*repository.Steps)(nil)
	_ chat.Store                 = (*repository.Conversations)(nil)
	_ requestBackend             = (*repository.Requests)(nil)
	_ requestBackend             = (*hr.HTTPClient)(nil)
	_ httptransport.MetricsStore = (*repository.Events)(nil)
)

func Migrate(ctx context.Context, db *sql.DB, log *slog.Logger) error {
	applied, err := migrate.Up(ctx, db, migrations.FS)
	if err != nil {
		return err
	}
	if len(applied) > 0 {
		log.Info("migrations applied", "versions", applied)
	}
	return nil
}

func NewLLM(cfg config.LLMConfig, db *sql.DB, log *slog.Logger) llm.Client {
	if cfg.Mode == config.LLMModeML {
		log.Info("llm configured", "mode", cfg.Mode, "url", cfg.MLURL)
		steps := &repository.Steps{DB: db}
		return &llm.ML{
			Client: ml.NewHTTPClient(cfg.MLURL, cfg.Timeout),
			KB:     &repository.KB{DB: db},
			Stage: func(ctx context.Context, id domain.Identity) string {
				list, err := steps.StepsFor(ctx, id.UserID)
				if err != nil {
					return ""
				}
				if next := progress.Summarize(list).Next; next != nil {
					return next.Title
				}
				return ""
			},
		}
	}
	if cfg.Mode == config.LLMModeOpenAI {
		log.Info("llm configured", "mode", cfg.Mode, "model", cfg.Model, "base_url", cfg.BaseURL)
		return llm.NewOpenAI(cfg.BaseURL, cfg.Model, cfg.APIKey, cfg.Timeout)
	}
	log.Warn("llm runs in fake mode: answers are rule-based, set LLM_API_KEY and LLM_MODEL to use a real model")
	return llm.Fake{}
}

type requestBackend interface {
	tools.RequestService
	httptransport.RequestStore
}

func newRequests(cfg config.Config, db *sql.DB, log *slog.Logger) requestBackend {
	if cfg.HRURL == "" {
		log.Info("corporate request services run in-process (HR_URL is empty)")
		return &repository.Requests{DB: db}
	}
	log.Info("corporate request services via HTTP", "url", cfg.HRURL)
	return hr.NewHTTPClient(cfg.HRURL, 5*time.Second)
}

func NewHandler(cfg config.Config, db *sql.DB, log *slog.Logger, model llm.Client) http.Handler {
	policy := auth.DefaultPolicy()

	events := &repository.Events{DB: db}
	actions := &repository.Actions{DB: db}
	requests := newRequests(cfg, db, log)
	steps := &repository.Steps{DB: db}

	registry := tools.NewRegistry(tools.Deps{
		Policy:   policy,
		Actions:  actions,
		KB:       &repository.KB{DB: db},
		Steps:    steps,
		People:   &repository.People{DB: db},
		Requests: requests,
		Events:   events,
		Log:      log,
	})
	chatSvc := &chat.Service{
		LLM:    model,
		Tools:  registry,
		Store:  &repository.Conversations{DB: db},
		Events: events,
	}
	progressSvc := &progress.Service{Store: steps, Events: events}

	return httptransport.NewRouter(httptransport.Deps{
		Log:        log,
		DB:         db,
		Auth:       &repository.MockTokenAuth{DB: db},
		Policy:     policy,
		Chat:       chatSvc,
		Actions:    registry,
		Progress:   progressSvc,
		Requests:   requests,
		Metrics:    events,
		CORSOrigin: cfg.CORSOrigin,
	})
}

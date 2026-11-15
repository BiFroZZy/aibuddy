package repository

import (
	"aibuddy/internal/tools"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

var _ tools.Events = (*EventsRepo)(nil)

type EventsRepo struct {
	DB *sql.DB
	Log *slog.Logger
}

func (e *EventsRepo) Emit(ctx context.Context, userID uuid.UUID, name string, payload any) {
	b, err := json.Marshal(payload)
	if err != nil {
		b = []byte("{}")
	}
	wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	var userIDArg any
	if userID != uuid.Nil {
		userIDArg = userID
	}
	if _, err := e.DB.ExecContext(wctx,
		`INSERT INTO events (user_id, name, payload) VALUES (NULLIF($1, ''), $2, $3::jsonb)`,
		userIDArg, name, string(b)); err != nil && e.Log != nil {
		e.Log.Warn("emit event failed", "event", name, "err", err)
	}
}
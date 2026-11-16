package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"aibuddy/internal/domain"

	"github.com/google/uuid"
)

type Events struct{ DB *sql.DB }

func (r *Events) Record(ctx context.Context, userID uuid.UUID, name string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode event payload: %w", err)
	}
	var uid any
	if userID != uuid.Nil {
		uid = userID
	}
	if _, err := r.DB.ExecContext(ctx,
		`INSERT INTO events (id, user_id, name, payload) VALUES ($1, $2, $3, $4::jsonb)`,
		uuid.New(), uid, name, string(b)); err != nil {
		return fmt.Errorf("record event: %w", err)
	}
	return nil
}

func (r *Events) Metrics(ctx context.Context, since time.Time) (domain.Metrics, error) {
	m := domain.Metrics{Since: since, Events: map[string]int{}, Tools: map[string]int{}}

	rows, err := r.DB.QueryContext(ctx,
		`SELECT name, count(*) FROM events WHERE created_at >= $1 GROUP BY name`, since)
	if err != nil {
		return m, fmt.Errorf("metrics events: %w", err)
	}
	if err := fillCounts(rows, m.Events); err != nil {
		return m, err
	}

	rows, err = r.DB.QueryContext(ctx, `
		SELECT payload->>'tool', count(*) FROM events
		WHERE created_at >= $1 AND name = 'tool_called' AND payload->>'tool' IS NOT NULL
		GROUP BY 1`, since)
	if err != nil {
		return m, fmt.Errorf("metrics tools: %w", err)
	}
	if err := fillCounts(rows, m.Tools); err != nil {
		return m, err
	}

	if err := r.DB.QueryRowContext(ctx,
		`SELECT count(DISTINCT user_id) FROM events WHERE created_at >= $1 AND user_id IS NOT NULL`, since).Scan(&m.ActiveUsers); err != nil {
		return m, fmt.Errorf("metrics active users: %w", err)
	}
	return m, nil
}

func fillCounts(rows *sql.Rows, dst map[string]int) error {
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		var n int
		if err := rows.Scan(&name, &n); err != nil {
			return fmt.Errorf("scan count: %w", err)
		}
		dst[name] = n
	}
	return rows.Err()
}

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"aibuddy/internal/domain"

	"github.com/google/uuid"
)

type Actions struct{ DB *sql.DB }

const actionCols = `id, user_id, tool, args, result, status, idempotency_key, external_id, error, created_at`

type rowScanner interface{ Scan(dest ...any) error }

func scanAction(s rowScanner) (domain.Action, error) {
	var (
		a      domain.Action
		args   []byte
		result []byte
		status string
		ext    uuid.NullUUID
	)
	if err := s.Scan(&a.ID, &a.UserID, &a.Tool, &args, &result, &status, &a.IdempotencyKey, &ext, &a.Error, &a.CreatedAt); err != nil {
		return domain.Action{}, err
	}
	a.Args, a.Status = json.RawMessage(args), domain.ActionStatus(status)
	if len(result) > 0 {
		a.Result = json.RawMessage(result)
	}
	if ext.Valid {
		id := ext.UUID
		a.ExternalID = &id
	}
	return a, nil
}

func (r *Actions) Create(ctx context.Context, a domain.Action) (domain.Action, bool, error) {
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO action_log (id, user_id, tool, args, status, idempotency_key, error)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7)
		ON CONFLICT (user_id, idempotency_key) DO NOTHING`,
		a.ID, a.UserID, a.Tool, string(a.Args), string(a.Status), a.IdempotencyKey, a.Error)
	if err != nil {
		return domain.Action{}, false, fmt.Errorf("insert action: %w", err)
	}
	created := false
	if n, _ := res.RowsAffected(); n > 0 {
		created = true
	}
	stored, err := scanAction(r.DB.QueryRowContext(ctx,
		`SELECT `+actionCols+` FROM action_log WHERE user_id = $1 AND idempotency_key = $2`, a.UserID, a.IdempotencyKey))
	if err != nil {
		return domain.Action{}, false, fmt.Errorf("read action: %w", err)
	}
	return stored, created, nil
}

func (r *Actions) Get(ctx context.Context, userID, id uuid.UUID) (domain.Action, error) {
	a, err := scanAction(r.DB.QueryRowContext(ctx,
		`SELECT `+actionCols+` FROM action_log WHERE id = $1 AND user_id = $2`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Action{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Action{}, fmt.Errorf("get action: %w", err)
	}
	return a, nil
}

func (r *Actions) Transition(ctx context.Context, id uuid.UUID, from, to domain.ActionStatus, result json.RawMessage, externalID *uuid.UUID, errMsg string) (bool, error) {
	var res any
	if len(result) > 0 {
		res = string(result)
	}
	var ext any
	if externalID != nil {
		ext = *externalID
	}
	out, err := r.DB.ExecContext(ctx, `
		UPDATE action_log
		SET status = $3, result = COALESCE($4::jsonb, result),
		    external_id = COALESCE($5::uuid, external_id), error = $6
		WHERE id = $1 AND status = $2`, id, string(from), string(to), res, ext, errMsg)
	if err != nil {
		return false, fmt.Errorf("transition action: %w", err)
	}
	n, _ := out.RowsAffected()
	return n > 0, nil
}

func (r *Actions) List(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Action, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT `+actionCols+` FROM action_log WHERE user_id = $1 ORDER BY created_at DESC, id LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list actions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	list := []domain.Action{}
	for rows.Next() {
		a, err := scanAction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan action: %w", err)
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

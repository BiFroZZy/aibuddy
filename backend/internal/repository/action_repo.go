package repository

import (
	"aibuddy/internal/domain"
	"aibuddy/internal/tools"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

var _ tools.ActionRepo = (*ActionRepo)(nil)

type ActionRepo struct {
	DB *sql.DB
}

func (r *ActionRepo) Create(ctx context.Context, a domain.Action) error {
	args := string(a.Args)
	if args == "" {
		args = "{}"
	}
	created := a.CreatedAt
	if created.IsZero() {
		created = time.Now()
	}
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO action_log (id, user_id, tool, args, status, idempotency_key, created_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7)`,
		a.ID, a.UserID, a.Tool, args, string(a.Status), a.IdempotencyKey, created)
	if err != nil {
		return fmt.Errorf("insert action_log: %w", err)
	}
	return nil
}

const selectAction = `
	SELECT id, user_id, tool, args, result, status, idempotency_key,
		COALESCE(external_id, ''), error, created_at
	FROM action_log`

type rowScanner interface{ Scan(dest ...any) error }

func scanAction(s rowScanner) (domain.Action, error) {
	var (
		a            domain.Action
		args, result []byte
		status       string
	)
	err := s.Scan(&a.ID, &a.UserID, &a.Tool, &args, &result, &status,
		&a.IdempotencyKey, &a.ExternalID, &a.Error, &a.CreatedAt)
	if err != nil {
		return domain.Action{}, err
	}
	a.Args = json.RawMessage(args)
	if result != nil {
		a.Result = json.RawMessage(result)
	}
	a.Status = domain.ActionStatus(status)
	return a, nil
}

func (r *ActionRepo) Get(ctx context.Context, id uuid.UUID) (domain.Action, error) {
	a, err := scanAction(r.DB.QueryRowContext(ctx, selectAction+` WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Action{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Action{}, fmt.Errorf("get action: %w", err)
	}
	return a, nil
}

func (r *ActionRepo) Claim(ctx context.Context, id uuid.UUID) (bool, error) {
	res, err := r.DB.ExecContext(ctx, `
		UPDATE action_log SET status = 'running'
		WHERE id = $1 AND status = 'pending_confirmation'`, id)
	if err != nil {
		return false, fmt.Errorf("claim action: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func (r *ActionRepo) Finish(ctx context.Context, id uuid.UUID, st domain.ActionStatus,
	result json.RawMessage, externalID uuid.UUID, errText string) error {

	var (
		res           any
		externalIDArg any
	)
	if len(result) > 0 {
		res = string(result)
	}
	if externalID != uuid.Nil {
		externalIDArg = externalID
	}
	out, err := r.DB.ExecContext(ctx, `
		UPDATE action_log
		SET status = $2, result = $3::jsonb, external_id = NULLIF($4, ''), error = $5
		WHERE id = $1 AND status = 'running'`,
		id, string(st), res, externalIDArg, errText)
	if err != nil {
		return fmt.Errorf("finish action: %w", err)
	}
	n, err := out.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("finish action %s: not in running state", id)
	}
	return nil
}

func (r *ActionRepo) ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Action, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.DB.QueryContext(ctx,
		selectAction+` WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`, userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list actions: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	var out []domain.Action
	for rows.Next() {
		a, err := scanAction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

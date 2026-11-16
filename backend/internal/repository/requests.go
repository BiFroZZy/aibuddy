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

type Requests struct{ DB *sql.DB }

const requestCols = `r.id, r.kind, r.user_id, COALESCE(u.full_name, ''), r.payload, r.status,
	r.approver_role, r.decision_comment, r.created_at, r.decided_at`

func scanRequest(s rowScanner) (domain.Request, error) {
	var (
		q       domain.Request
		payload []byte
		decided sql.NullTime
	)
	if err := s.Scan(&q.ID, &q.Kind, &q.UserID, &q.UserName, &payload, &q.Status,
		&q.ApproverRole, &q.DecisionComment, &q.CreatedAt, &decided); err != nil {
		return domain.Request{}, err
	}
	q.Payload = json.RawMessage(payload)
	if decided.Valid {
		t := decided.Time
		q.DecidedAt = &t
	}
	return q, nil
}

func (r *Requests) Create(ctx context.Context, userID uuid.UUID, kind string, payload json.RawMessage, approverRole, idempotencyKey string) (uuid.UUID, error) {
	if idempotencyKey == "" {
		idempotencyKey = uuid.NewString()
	}
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO mock_requests (id, idempotency_key, kind, user_id, payload, approver_role)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
		ON CONFLICT (idempotency_key) DO NOTHING`, uuid.New(), idempotencyKey, kind, userID, string(payload), approverRole)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create request: %w", err)
	}
	var id uuid.UUID
	if err := r.DB.QueryRowContext(ctx, `SELECT id FROM mock_requests WHERE idempotency_key = $1`, idempotencyKey).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("read request: %w", err)
	}
	return id, nil
}

func (r *Requests) Get(ctx context.Context, id uuid.UUID) (domain.Request, error) {
	q, err := scanRequest(r.DB.QueryRowContext(ctx, `
		SELECT `+requestCols+`
		FROM mock_requests r JOIN users u ON u.id = r.user_id WHERE r.id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Request{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Request{}, fmt.Errorf("get request: %w", err)
	}
	return q, nil
}

func (r *Requests) ListForUser(ctx context.Context, userID uuid.UUID, kind string, limit int) ([]domain.Request, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT `+requestCols+`
		FROM mock_requests r JOIN users u ON u.id = r.user_id
		WHERE r.user_id = $1 AND ($2 = '' OR r.kind = $2)
		ORDER BY r.created_at DESC, r.id LIMIT $3`, userID, kind, limit)
	if err != nil {
		return nil, fmt.Errorf("list requests: %w", err)
	}
	return collectRequests(rows)
}

func (r *Requests) PendingForManager(ctx context.Context, managerID uuid.UUID) ([]domain.Request, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT `+requestCols+`
		FROM mock_requests r JOIN users u ON u.id = r.user_id
		WHERE u.manager_id = $1 AND r.status = 'pending_approval'
		ORDER BY r.created_at, r.id`, managerID)
	if err != nil {
		return nil, fmt.Errorf("pending requests: %w", err)
	}
	return collectRequests(rows)
}

func (r *Requests) Decide(ctx context.Context, managerID, id uuid.UUID, approve bool, comment string) (domain.Request, error) {
	status := domain.RequestRejected
	if approve {
		status = domain.RequestApproved
	}
	row := r.DB.QueryRowContext(ctx, `
		WITH upd AS (
		  UPDATE mock_requests r
		  SET status = $3, decided_by = $2, decision_comment = $4, decided_at = now()
		  FROM users u
		  WHERE r.id = $1 AND u.id = r.user_id AND u.manager_id = $2 AND r.status = 'pending_approval'
		  RETURNING r.*)
		SELECT `+requestCols+` FROM upd r JOIN users u ON u.id = r.user_id`, id, managerID, status, comment)
	q, err := scanRequest(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Request{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Request{}, fmt.Errorf("decide request: %w", err)
	}
	return q, nil
}

func collectRequests(rows *sql.Rows) ([]domain.Request, error) {
	defer func() { _ = rows.Close() }()
	list := []domain.Request{}
	for rows.Next() {
		q, err := scanRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("scan request: %w", err)
		}
		list = append(list, q)
	}
	return list, rows.Err()
}

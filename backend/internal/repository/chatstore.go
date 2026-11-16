package repository

import (
	"context"
	"database/sql"
	"fmt"

	"aibuddy/internal/domain"

	"github.com/google/uuid"
)

type Conversations struct{ DB *sql.DB }

func (r *Conversations) CreateConversation(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	id := uuid.New()
	if _, err := r.DB.ExecContext(ctx, `INSERT INTO conversations (id, user_id) VALUES ($1, $2)`, id, userID); err != nil {
		return uuid.Nil, fmt.Errorf("create conversation: %w", err)
	}
	return id, nil
}

func (r *Conversations) Owns(ctx context.Context, userID, convID uuid.UUID) (bool, error) {
	var ok bool
	err := r.DB.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM conversations WHERE id = $1 AND user_id = $2)`, convID, userID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("conversation owner: %w", err)
	}
	return ok, nil
}

func (r *Conversations) AddMessage(ctx context.Context, convID uuid.UUID, role, content string) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO messages (id, conversation_id, role, content) VALUES ($1, $2, $3, $4)`,
		uuid.New(), convID, role, content)
	if err != nil {
		return fmt.Errorf("add message: %w", err)
	}
	return nil
}

func (r *Conversations) Messages(ctx context.Context, convID uuid.UUID, limit int) ([]domain.ChatMessage, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, role, content, created_at FROM (
		  SELECT id, role, content, created_at FROM messages
		  WHERE conversation_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2) t
		ORDER BY created_at, id`, convID, limit)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	list := []domain.ChatMessage{}
	for rows.Next() {
		var m domain.ChatMessage
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

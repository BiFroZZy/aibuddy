package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"aibuddy/internal/domain"

	"github.com/google/uuid"
)

type People struct{ DB *sql.DB }

var hrTopics = []string{"бенефит", "документ", "оформлен", "отпуск", "справк", "трудов", "больнич", "увольн"}

var roleNotes = map[string]string{
	"mentor":  "наставник: задачи, инструменты, первые шаги",
	"manager": "руководитель: цели, согласования, отсутствия",
	"hr_bp":   "HR BP: оформление, бенефиты, организационные вопросы",
}

func (r *People) WhoToAsk(ctx context.Context, userID uuid.UUID, topic string) ([]domain.Person, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT u.full_name, u.role
		FROM users me JOIN users u ON u.id IN (me.mentor_id, me.manager_id)
		WHERE me.id = $1
		ORDER BY (u.role = 'mentor') DESC, u.full_name`, userID)
	if err != nil {
		return nil, fmt.Errorf("who to ask: %w", err)
	}
	defer func() { _ = rows.Close() }()

	people := []domain.Person{}
	for rows.Next() {
		var p domain.Person
		if err := rows.Scan(&p.FullName, &p.Role); err != nil {
			return nil, fmt.Errorf("scan person: %w", err)
		}
		p.Note = roleNotes[p.Role]
		people = append(people, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if isHRTopic(topic) {
		var p domain.Person
		err := r.DB.QueryRowContext(ctx,
			`SELECT full_name, role FROM users WHERE role = 'hr_bp' ORDER BY login LIMIT 1`).Scan(&p.FullName, &p.Role)
		switch err {
		case nil:
			p.Note = roleNotes[p.Role]
			people = append(people, p)
		case sql.ErrNoRows:
		default:
			return nil, fmt.Errorf("find hr bp: %w", err)
		}
	}
	return people, nil
}

func isHRTopic(topic string) bool {
	low := strings.ToLower(topic)
	for _, k := range hrTopics {
		if strings.Contains(low, k) {
			return true
		}
	}
	return false
}

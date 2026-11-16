package repository

import (
	"context"
	"database/sql"
	"fmt"

	"aibuddy/internal/domain"

	"github.com/google/uuid"
)

type Steps struct{ DB *sql.DB }

func (r *Steps) StepsFor(ctx context.Context, userID uuid.UUID) ([]domain.Step, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT ss.id, s.code, s.title, ss.position, ss.title, ss.description, ss.confirm_by,
		       COALESCE(i.status, 'not_started'), i.updated_at
		FROM scenario_steps ss
		JOIN scenarios s ON s.id = ss.scenario_id
		LEFT JOIN intern_steps i ON i.step_id = ss.id AND i.user_id = $1
		ORDER BY s.code, ss.position`, userID)
	if err != nil {
		return nil, fmt.Errorf("steps for user: %w", err)
	}
	defer func() { _ = rows.Close() }()

	steps := []domain.Step{}
	for rows.Next() {
		var (
			s       domain.Step
			confirm string
			status  string
			updated sql.NullTime
		)
		if err := rows.Scan(&s.ID, &s.ScenarioCode, &s.ScenarioTitle, &s.Position, &s.Title, &s.Description, &confirm, &status, &updated); err != nil {
			return nil, fmt.Errorf("scan step: %w", err)
		}
		s.ConfirmBy, s.Status = domain.ConfirmBy(confirm), domain.StepStatus(status)
		if updated.Valid {
			t := updated.Time
			s.UpdatedAt = &t
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

func (r *Steps) SetStatus(ctx context.Context, userID, stepID uuid.UUID, status domain.StepStatus) error {
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO intern_steps (user_id, step_id, status) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, step_id)
		DO UPDATE SET status = EXCLUDED.status, updated_at = now()
		WHERE intern_steps.status <> 'done'`, userID, stepID, string(status))
	if err != nil {
		return fmt.Errorf("set step status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: step is already done", domain.ErrConflict)
	}
	return nil
}

func (r *Steps) WaitingForMentor(ctx context.Context, mentorID uuid.UUID) ([]domain.Confirmation, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT u.id, u.full_name, ss.id, ss.title, i.updated_at
		FROM intern_steps i
		JOIN users u ON u.id = i.user_id
		JOIN scenario_steps ss ON ss.id = i.step_id
		WHERE u.mentor_id = $1 AND i.status = 'waiting'
		ORDER BY i.updated_at, ss.position`, mentorID)
	if err != nil {
		return nil, fmt.Errorf("waiting for mentor: %w", err)
	}
	defer func() { _ = rows.Close() }()

	list := []domain.Confirmation{}
	for rows.Next() {
		var c domain.Confirmation
		if err := rows.Scan(&c.InternID, &c.InternName, &c.StepID, &c.StepTitle, &c.Since); err != nil {
			return nil, fmt.Errorf("scan confirmation: %w", err)
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *Steps) MentorDecide(ctx context.Context, mentorID, internID, stepID uuid.UUID, approve bool) error {
	to := domain.StepInProgress
	if approve {
		to = domain.StepDone
	}
	res, err := r.DB.ExecContext(ctx, `
		UPDATE intern_steps i SET status = $4, updated_at = now()
		FROM users u, scenario_steps s
		WHERE i.user_id = $2 AND i.step_id = $3
		  AND u.id = i.user_id AND u.mentor_id = $1
		  AND s.id = i.step_id AND s.confirm_by = 'mentor'
		  AND i.status = 'waiting'`, mentorID, internID, stepID, string(to))
	if err != nil {
		return fmt.Errorf("mentor decide: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

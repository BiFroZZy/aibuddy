package progress

import (
	"context"
	"fmt"

	"aibuddy/internal/domain"

	"github.com/google/uuid"
)

type Progress struct {
	Steps   []domain.Step `json:"steps"`
	Done    int           `json:"done"`
	Total   int           `json:"total"`
	Percent int           `json:"percent"`
	Next    *domain.Step  `json:"next"`
	Waiting []domain.Step `json:"waiting"`
}

func Summarize(steps []domain.Step) Progress {
	p := Progress{Steps: steps, Total: len(steps), Waiting: []domain.Step{}}
	for i := range steps {
		s := steps[i]
		switch s.Status {
		case domain.StepDone:
			p.Done++
		case domain.StepWaiting:
			p.Waiting = append(p.Waiting, s)
		default:
			if p.Next == nil {
				next := s
				p.Next = &next
			}
		}
	}
	if p.Total > 0 {
		p.Percent = p.Done * 100 / p.Total
	}
	return p
}

func CheckTransition(step domain.Step, to domain.StepStatus) error {
	if !to.Valid() {
		return fmt.Errorf("%w: unknown status %q", domain.ErrInvalidArgs, to)
	}
	if step.Status == domain.StepDone {
		return fmt.Errorf("%w: step is already done", domain.ErrConflict)
	}
	switch to {
	case domain.StepDone:
		if step.ConfirmBy != domain.ConfirmSelf {
			return fmt.Errorf("%w: this step must be confirmed by the mentor", domain.ErrForbidden)
		}
	case domain.StepWaiting:
		if step.ConfirmBy != domain.ConfirmMentor {
			return fmt.Errorf("%w: this step does not need mentor confirmation", domain.ErrInvalidArgs)
		}
	}
	return nil
}

type Store interface {
	StepsFor(ctx context.Context, userID uuid.UUID) ([]domain.Step, error)
	SetStatus(ctx context.Context, userID, stepID uuid.UUID, status domain.StepStatus) error
	WaitingForMentor(ctx context.Context, mentorID uuid.UUID) ([]domain.Confirmation, error)
	MentorDecide(ctx context.Context, mentorID, internID, stepID uuid.UUID, approve bool) error
}

type Recorder interface {
	Record(ctx context.Context, userID uuid.UUID, name string, payload any) error
}

type Service struct {
	Store  Store
	Events Recorder
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (Progress, error) {
	steps, err := s.Store.StepsFor(ctx, userID)
	if err != nil {
		return Progress{}, err
	}
	return Summarize(steps), nil
}

func (s *Service) SetStatus(ctx context.Context, userID, stepID uuid.UUID, to domain.StepStatus) (Progress, error) {
	steps, err := s.Store.StepsFor(ctx, userID)
	if err != nil {
		return Progress{}, err
	}
	var step *domain.Step
	for i := range steps {
		if steps[i].ID == stepID {
			step = &steps[i]
			break
		}
	}
	if step == nil {
		return Progress{}, domain.ErrNotFound
	}
	if err := CheckTransition(*step, to); err != nil {
		return Progress{}, err
	}
	if err := s.Store.SetStatus(ctx, userID, stepID, to); err != nil {
		return Progress{}, err
	}
	s.record(ctx, userID, "step_status_changed", map[string]any{"step_id": stepID, "status": to})
	return s.Get(ctx, userID)
}

func (s *Service) Confirmations(ctx context.Context, mentorID uuid.UUID) ([]domain.Confirmation, error) {
	return s.Store.WaitingForMentor(ctx, mentorID)
}

func (s *Service) Decide(ctx context.Context, mentorID, internID, stepID uuid.UUID, approve bool) error {
	if err := s.Store.MentorDecide(ctx, mentorID, internID, stepID, approve); err != nil {
		return err
	}
	s.record(ctx, mentorID, "step_decided", map[string]any{"intern_id": internID, "step_id": stepID, "approved": approve})
	return nil
}

func (s *Service) record(ctx context.Context, userID uuid.UUID, name string, payload any) {
	if s.Events != nil {
		_ = s.Events.Record(ctx, userID, name, payload)
	}
}

package progress

import (
	"context"
	"errors"
	"testing"

	"aibuddy/internal/domain"

	"github.com/google/uuid"
)

func step(status domain.StepStatus, by domain.ConfirmBy) domain.Step {
	return domain.Step{ID: uuid.New(), Title: string(status), Status: status, ConfirmBy: by}
}

func TestSummarize(t *testing.T) {
	steps := []domain.Step{
		step(domain.StepDone, domain.ConfirmSelf),
		step(domain.StepWaiting, domain.ConfirmMentor),
		step(domain.StepInProgress, domain.ConfirmSelf),
		step(domain.StepNotStarted, domain.ConfirmSelf),
	}
	p := Summarize(steps)
	if p.Done != 1 || p.Total != 4 || p.Percent != 25 || len(p.Waiting) != 1 {
		t.Fatalf("counters: %+v", p)
	}
	if p.Next == nil || p.Next.Status != domain.StepInProgress {
		t.Fatalf("next must skip done and waiting steps: %+v", p.Next)
	}
}

func TestSummarize_EdgeCases(t *testing.T) {
	if p := Summarize(nil); p.Percent != 0 || p.Next != nil || p.Waiting == nil {
		t.Fatalf("empty checklist: %+v", p)
	}
	all := Summarize([]domain.Step{step(domain.StepDone, domain.ConfirmSelf)})
	if all.Percent != 100 || all.Next != nil {
		t.Fatalf("everything done: %+v", all)
	}
}

func TestCheckTransition(t *testing.T) {
	tests := []struct {
		name string
		step domain.Step
		to   domain.StepStatus
		want error
	}{
		{"self step to done", step(domain.StepInProgress, domain.ConfirmSelf), domain.StepDone, nil},
		{"mentor step cannot be self-closed", step(domain.StepInProgress, domain.ConfirmMentor), domain.StepDone, domain.ErrForbidden},
		{"mentor step to waiting", step(domain.StepInProgress, domain.ConfirmMentor), domain.StepWaiting, nil},
		{"self step cannot wait for mentor", step(domain.StepInProgress, domain.ConfirmSelf), domain.StepWaiting, domain.ErrInvalidArgs},
		{"done is final", step(domain.StepDone, domain.ConfirmSelf), domain.StepInProgress, domain.ErrConflict},
		{"unknown status", step(domain.StepNotStarted, domain.ConfirmSelf), domain.StepStatus("fly"), domain.ErrInvalidArgs},
		{"withdraw from waiting", step(domain.StepWaiting, domain.ConfirmMentor), domain.StepInProgress, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckTransition(tc.step, tc.to)
			if !errors.Is(err, tc.want) || (tc.want == nil && err != nil) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

type fakeStore struct {
	steps    []domain.Step
	setErr   error
	set      map[uuid.UUID]domain.StepStatus
	decided  []bool
	confirms []domain.Confirmation
}

func (f *fakeStore) StepsFor(context.Context, uuid.UUID) ([]domain.Step, error) {
	out := make([]domain.Step, len(f.steps))
	copy(out, f.steps)
	for i := range out {
		if s, ok := f.set[out[i].ID]; ok {
			out[i].Status = s
		}
	}
	return out, nil
}

func (f *fakeStore) SetStatus(_ context.Context, _, stepID uuid.UUID, s domain.StepStatus) error {
	if f.setErr != nil {
		return f.setErr
	}
	if f.set == nil {
		f.set = map[uuid.UUID]domain.StepStatus{}
	}
	f.set[stepID] = s
	return nil
}

func (f *fakeStore) WaitingForMentor(context.Context, uuid.UUID) ([]domain.Confirmation, error) {
	return f.confirms, nil
}

func (f *fakeStore) MentorDecide(_ context.Context, _, _, _ uuid.UUID, approve bool) error {
	f.decided = append(f.decided, approve)
	return nil
}

type recorder struct{ names []string }

func (r *recorder) Record(_ context.Context, _ uuid.UUID, name string, _ any) error {
	r.names = append(r.names, name)
	return nil
}

func TestService(t *testing.T) {
	self, mentor := step(domain.StepNotStarted, domain.ConfirmSelf), step(domain.StepNotStarted, domain.ConfirmMentor)
	store := &fakeStore{steps: []domain.Step{self, mentor}, confirms: []domain.Confirmation{{StepTitle: "x"}}}
	rec := &recorder{}
	svc := &Service{Store: store, Events: rec}
	ctx := context.Background()
	user := uuid.New()

	p, err := svc.SetStatus(ctx, user, self.ID, domain.StepDone)
	if err != nil || p.Done != 1 {
		t.Fatalf("self done: %+v %v", p, err)
	}
	if _, err := svc.SetStatus(ctx, user, mentor.ID, domain.StepDone); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("intern cannot close a mentor step: %v", err)
	}
	if _, err := svc.SetStatus(ctx, user, uuid.New(), domain.StepDone); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown step: %v", err)
	}
	p, err = svc.SetStatus(ctx, user, mentor.ID, domain.StepWaiting)
	if err != nil || len(p.Waiting) != 1 {
		t.Fatalf("waiting: %+v %v", p, err)
	}

	if list, _ := svc.Confirmations(ctx, user); len(list) != 1 {
		t.Fatal("confirmations must be passed through")
	}
	if err := svc.Decide(ctx, user, uuid.New(), mentor.ID, true); err != nil || len(store.decided) != 1 || !store.decided[0] {
		t.Fatalf("decide: %v %v", store.decided, err)
	}
	if len(rec.names) != 3 {
		t.Fatalf("events: %v", rec.names)
	}

	store.setErr = errors.New("db down")
	if _, err := svc.SetStatus(ctx, user, self.ID, domain.StepInProgress); err == nil {
		t.Fatal("store errors must propagate")
	}
	if _, err := (&Service{Store: store}).Get(ctx, user); err != nil {
		t.Fatalf("service must work without an event recorder: %v", err)
	}
}

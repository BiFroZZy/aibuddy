package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"aibuddy/internal/domain"
	"aibuddy/internal/repository"
	"aibuddy/internal/testdb"

	"github.com/google/uuid"
)

var ctx = context.Background()

func TestMockTokenAuth(t *testing.T) {
	a := &repository.MockTokenAuth{DB: testdb.New(t)}

	id, err := a.Authenticate(ctx, testdb.InternToken)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if id.UserID != testdb.InternID || id.Role != domain.RoleIntern || id.TeamID != testdb.Team1ID || id.Team == "" {
		t.Fatalf("unexpected identity: %+v", id)
	}
	for _, tok := range []string{"", "nope"} {
		if _, err := a.Authenticate(ctx, tok); !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("token %q: want ErrUnauthorized, got %v", tok, err)
		}
	}
}

func TestMockTokenAuth_UserWithoutTeam(t *testing.T) {
	db := testdb.New(t)
	if _, err := db.Exec(`UPDATE users SET team_id = NULL WHERE id = $1`, testdb.InternID); err != nil {
		t.Fatal(err)
	}
	id, err := (&repository.MockTokenAuth{DB: db}).Authenticate(ctx, testdb.InternToken)
	if err != nil {
		t.Fatalf("a user without a team must still authenticate: %v", err)
	}
	if id.TeamID != uuid.Nil || id.Team != "" {
		t.Fatalf("expected empty team, got %+v", id)
	}
}

func TestMockTokenAuth_BackendFailure(t *testing.T) {
	db := testdb.New(t)
	a := &repository.MockTokenAuth{DB: db}
	_ = db.Close()
	if _, err := a.Authenticate(ctx, testdb.InternToken); err == nil || errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("want infrastructure error, got %v", err)
	}
}

func TestStepsLifecycle(t *testing.T) {
	r := &repository.Steps{DB: testdb.New(t)}

	steps, err := r.StepsFor(ctx, testdb.InternID)
	if err != nil || len(steps) != 7 {
		t.Fatalf("steps: %d, %v", len(steps), err)
	}
	byID := map[uuid.UUID]domain.Step{}
	for _, s := range steps {
		byID[s.ID] = s
	}
	if byID[testdb.Step1].Status != domain.StepDone || byID[testdb.Step2].Status != domain.StepInProgress || byID[testdb.Step3].Status != domain.StepNotStarted {
		t.Fatalf("unexpected seeded statuses: %+v", steps)
	}
	if byID[testdb.Step2].ConfirmBy != domain.ConfirmMentor || byID[testdb.Step1].UpdatedAt == nil {
		t.Fatalf("unexpected step metadata: %+v", byID[testdb.Step2])
	}

	other, _ := r.StepsFor(ctx, testdb.Intern2ID)
	for _, s := range other {
		if s.Status != domain.StepNotStarted {
			t.Fatalf("intern2 must start clean, got %s for %s", s.Status, s.Title)
		}
	}

	if err := r.SetStatus(ctx, testdb.InternID, testdb.Step3, domain.StepInProgress); err != nil {
		t.Fatal(err)
	}
	if err := r.SetStatus(ctx, testdb.InternID, testdb.Step3, domain.StepDone); err != nil {
		t.Fatal(err)
	}
	if err := r.SetStatus(ctx, testdb.InternID, testdb.Step3, domain.StepInProgress); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("a finished step must not be overwritten, got %v", err)
	}
}

func TestMentorFlow(t *testing.T) {
	r := &repository.Steps{DB: testdb.New(t)}
	if err := r.SetStatus(ctx, testdb.InternID, testdb.Step2, domain.StepWaiting); err != nil {
		t.Fatal(err)
	}

	list, err := r.WaitingForMentor(ctx, testdb.MentorID)
	if err != nil || len(list) != 1 || list[0].StepID != testdb.Step2 || list[0].InternID != testdb.InternID {
		t.Fatalf("waiting: %+v, %v", list, err)
	}
	if none, _ := r.WaitingForMentor(ctx, testdb.ManagerID); len(none) != 0 {
		t.Fatalf("only the intern's mentor sees confirmations, got %+v", none)
	}

	if err := r.MentorDecide(ctx, testdb.ManagerID, testdb.InternID, testdb.Step2, true); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("foreign mentor: %v", err)
	}
	if err := r.MentorDecide(ctx, testdb.MentorID, testdb.Intern2ID, testdb.Step2, true); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("intern without waiting step: %v", err)
	}

	if err := r.MentorDecide(ctx, testdb.MentorID, testdb.InternID, testdb.Step2, false); err != nil {
		t.Fatal(err)
	}
	steps, _ := r.StepsFor(ctx, testdb.InternID)
	for _, s := range steps {
		if s.ID == testdb.Step2 && s.Status != domain.StepInProgress {
			t.Fatalf("rejected step must go back to in_progress, got %s", s.Status)
		}
	}

	_ = r.SetStatus(ctx, testdb.InternID, testdb.Step2, domain.StepWaiting)
	if err := r.MentorDecide(ctx, testdb.MentorID, testdb.InternID, testdb.Step2, true); err != nil {
		t.Fatal(err)
	}
	steps, _ = r.StepsFor(ctx, testdb.InternID)
	for _, s := range steps {
		if s.ID == testdb.Step2 && s.Status != domain.StepDone {
			t.Fatalf("approved step must be done, got %s", s.Status)
		}
	}
	if err := r.MentorDecide(ctx, testdb.MentorID, testdb.InternID, testdb.Step2, true); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("repeat decision: %v", err)
	}
}

func TestKBSearch_AccessRules(t *testing.T) {
	r := &repository.KB{DB: testdb.New(t)}
	intern := domain.Identity{UserID: testdb.InternID, Role: domain.RoleIntern, TeamID: testdb.Team1ID}

	hits, err := r.Search(ctx, intern, "как оформить отгул?", 5)
	if err != nil || len(hits) == 0 {
		t.Fatalf("search: %v, %v", hits, err)
	}
	if hits[0].Source != "test-data/absence.md" || hits[0].Snippet == "" {
		t.Fatalf("unexpected top hit: %+v", hits[0])
	}
	for _, h := range hits {
		if containsTag(h.Snippet) {
			t.Fatalf("snippet must not contain markup: %q", h.Snippet)
		}
	}

	memo := func(id domain.Identity) bool {
		hs, err := r.Search(ctx, id, "памятка руководителя согласование отсутствий", 10)
		if err != nil {
			t.Fatal(err)
		}
		for _, h := range hs {
			if h.Source == "test-data/manager-absence.md" {
				return true
			}
		}
		return false
	}
	if memo(intern) {
		t.Fatal("intern must not see the manager memo")
	}
	if !memo(domain.Identity{UserID: testdb.ManagerID, Role: domain.RoleManager, TeamID: testdb.Team1ID}) {
		t.Fatal("manager must see the manager memo")
	}

	if hs, _ := r.Search(ctx, intern, "расписание дежурств", 5); len(hs) != 0 {
		t.Fatalf("other team's document leaked: %+v", hs)
	}
	hasSource := func(hs []domain.KBHit, src string) bool {
		for _, h := range hs {
			if h.Source == src {
				return true
			}
		}
		return false
	}
	noTeam := domain.Identity{UserID: testdb.InternID, Role: domain.RoleIntern}
	hs, _ := r.Search(ctx, noTeam, "Git Docker PostgreSQL локальное окружение", 5)
	if hasSource(hs, "test-data/tools.md") {
		t.Fatalf("team document must not be visible without a team: %+v", hs)
	}
	hs, _ = r.Search(ctx, intern, "Git Docker PostgreSQL локальное окружение", 5)
	if !hasSource(hs, "test-data/tools.md") {
		t.Fatalf("a team member must see the team document: %+v", hs)
	}
	if hs, _ := r.Search(ctx, intern, "абракадабра", 5); len(hs) != 0 {
		t.Fatalf("unexpected hits: %+v", hs)
	}
}

func containsTag(s string) bool {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '<' && (s[i+1] == 'b' || s[i+1] == '/') {
			return true
		}
	}
	return false
}

func TestActions(t *testing.T) {
	r := &repository.Actions{DB: testdb.New(t)}
	a := domain.Action{ID: uuid.New(), UserID: testdb.InternID, Tool: "create_absence_request",
		Args: json.RawMessage(`{"x":1}`), Status: domain.ActionPending, IdempotencyKey: "k1"}

	stored, created, err := r.Create(ctx, a)
	if err != nil || !created || stored.ID != a.ID {
		t.Fatalf("create: %+v %v %v", stored, created, err)
	}
	dup := a
	dup.ID = uuid.New()
	again, created, err := r.Create(ctx, dup)
	if err != nil || created || again.ID != a.ID {
		t.Fatalf("idempotent create: %+v %v %v", again, created, err)
	}
	other := a
	other.ID, other.UserID = uuid.New(), testdb.Intern2ID
	if _, created, err := r.Create(ctx, other); err != nil || !created {
		t.Fatalf("keys must be scoped per user: %v %v", created, err)
	}

	if _, err := r.Get(ctx, testdb.Intern2ID, a.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("another user's action must be invisible, got %v", err)
	}

	ok, err := r.Transition(ctx, a.ID, domain.ActionPending, domain.ActionRunning, nil, nil, "")
	if err != nil || !ok {
		t.Fatalf("first transition: %v %v", ok, err)
	}
	if ok, _ := r.Transition(ctx, a.ID, domain.ActionPending, domain.ActionRunning, nil, nil, ""); ok {
		t.Fatal("second pending->running must lose the race")
	}
	ext := uuid.New()
	if ok, err := r.Transition(ctx, a.ID, domain.ActionRunning, domain.ActionDone, json.RawMessage(`{"ok":true}`), &ext, ""); err != nil || !ok {
		t.Fatalf("finish: %v %v", ok, err)
	}
	got, err := r.Get(ctx, testdb.InternID, a.ID)
	if err != nil || got.Status != domain.ActionDone || got.ExternalID == nil || *got.ExternalID != ext || string(got.Result) == "" {
		t.Fatalf("final state: %+v %v", got, err)
	}

	list, err := r.List(ctx, testdb.InternID, 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %+v %v", list, err)
	}
}

func TestRequests(t *testing.T) {
	r := &repository.Requests{DB: testdb.New(t)}
	payload := json.RawMessage(`{"kind":"day_off","date_from":"2026-12-01","date_to":"2026-12-01"}`)

	id, err := r.Create(ctx, testdb.InternID, domain.KindAbsence, payload, "manager", "key-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Create(ctx, testdb.InternID, domain.KindAccess, json.RawMessage(`{"project":"x"}`), "devops", ""); err != nil {
		t.Fatal(err)
	}

	abs, err := r.ListForUser(ctx, testdb.InternID, domain.KindAbsence, 10)
	if err != nil || len(abs) != 1 || abs[0].ID != id || abs[0].Status != domain.RequestPending {
		t.Fatalf("absences: %+v %v", abs, err)
	}
	if all, _ := r.ListForUser(ctx, testdb.InternID, "", 10); len(all) != 2 {
		t.Fatalf("all kinds: %d", len(all))
	}
	if none, _ := r.ListForUser(ctx, testdb.Intern2ID, "", 10); len(none) != 0 {
		t.Fatal("requests are per user")
	}

	pending, err := r.PendingForManager(ctx, testdb.ManagerID)
	if err != nil || len(pending) != 2 || pending[0].UserName == "" {
		t.Fatalf("pending: %+v %v", pending, err)
	}
	if other, _ := r.PendingForManager(ctx, testdb.MentorID); len(other) != 0 {
		t.Fatal("only the direct manager sees pending requests")
	}

	if _, err := r.Decide(ctx, testdb.MentorID, id, true, ""); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("a non-manager must not decide: %v", err)
	}
	got, err := r.Decide(ctx, testdb.ManagerID, id, false, "в эти дни релиз")
	if err != nil || got.Status != domain.RequestRejected || got.DecisionComment != "в эти дни релиз" || got.DecidedAt == nil {
		t.Fatalf("decide: %+v %v", got, err)
	}
	if _, err := r.Decide(ctx, testdb.ManagerID, id, true, ""); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("a decided request cannot be decided again: %v", err)
	}
}

func TestConversations(t *testing.T) {
	r := &repository.Conversations{DB: testdb.New(t)}
	cid, err := r.CreateConversation(ctx, testdb.InternID)
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := r.Owns(ctx, testdb.InternID, cid); !ok {
		t.Fatal("owner must own the conversation")
	}
	if ok, _ := r.Owns(ctx, testdb.Intern2ID, cid); ok {
		t.Fatal("another user must not own it")
	}
	if ok, _ := r.Owns(ctx, testdb.InternID, uuid.New()); ok {
		t.Fatal("unknown conversation")
	}

	for i, text := range []string{"один", "два", "три", "четыре"} {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		if err := r.AddMessage(ctx, cid, role, text); err != nil {
			t.Fatal(err)
		}
	}
	msgs, err := r.Messages(ctx, cid, 3)
	if err != nil || len(msgs) != 3 {
		t.Fatalf("messages: %+v %v", msgs, err)
	}
	if msgs[0].Content != "два" || msgs[2].Content != "четыре" {
		t.Fatalf("the last messages must come back in chronological order: %+v", msgs)
	}
}

func TestPeople(t *testing.T) {
	r := &repository.People{DB: testdb.New(t)}
	p, err := r.WhoToAsk(ctx, testdb.InternID, "доступ к git")
	if err != nil || len(p) != 2 || p[0].Role != "mentor" || p[1].Role != "manager" || p[0].Note == "" {
		t.Fatalf("who to ask: %+v %v", p, err)
	}
	p, err = r.WhoToAsk(ctx, testdb.InternID, "Оформление ДМС и бенефиты")
	if err != nil || len(p) != 3 || p[2].Role != "hr_bp" {
		t.Fatalf("HR topic must add the HR BP: %+v %v", p, err)
	}
	if p, _ := r.WhoToAsk(ctx, testdb.HRID, ""); len(p) != 0 {
		t.Fatalf("a user without mentor/manager: %+v", p)
	}
}

func TestEventsAndMetrics(t *testing.T) {
	r := &repository.Events{DB: testdb.New(t)}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(r.Record(ctx, testdb.InternID, "chat_message", map[string]any{}))
	must(r.Record(ctx, testdb.InternID, "tool_called", map[string]any{"tool": "search_kb"}))
	must(r.Record(ctx, testdb.Intern2ID, "tool_called", map[string]any{"tool": "search_kb"}))
	must(r.Record(ctx, uuid.Nil, "system_event", map[string]any{}))

	m, err := r.Metrics(ctx, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if m.Events["tool_called"] != 2 || m.Events["chat_message"] != 1 || m.Tools["search_kb"] != 2 || m.ActiveUsers != 2 {
		t.Fatalf("metrics: %+v", m)
	}
	if old, _ := r.Metrics(ctx, time.Now().Add(time.Hour)); len(old.Events) != 0 || old.ActiveUsers != 0 {
		t.Fatalf("the window must exclude events before it: %+v", old)
	}
	if err := r.Record(ctx, uuid.Nil, "bad", make(chan int)); err == nil {
		t.Fatal("an unencodable payload must fail")
	}
}

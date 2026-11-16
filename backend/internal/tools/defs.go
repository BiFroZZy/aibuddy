package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"aibuddy/internal/auth"
	"aibuddy/internal/domain"
	"aibuddy/internal/llm"
	"aibuddy/internal/progress"

	"github.com/google/uuid"
)

const dateLayout = "2006-01-02"

var kindLabels = map[string]string{"day_off": "отгул", "vacation": "отпуск", "sick": "больничный"}

var levelLabels = map[string]string{"read": "чтение", "write": "редактирование", "admin": "администрирование"}

var approverByLevel = map[string]string{"read": "project_owner", "write": "manager", "admin": "devops"}

var approverLabels = map[string]string{"project_owner": "владелец проекта", "manager": "руководитель", "devops": "DevOps-инженер"}

func badArgs(format string, a ...any) error {
	return fmt.Errorf("%w: %s", domain.ErrInvalidArgs, fmt.Sprintf(format, a...))
}

func decode(raw json.RawMessage, dst any) error {
	if err := json.Unmarshal(raw, dst); err != nil {
		return badArgs("arguments must be a JSON object")
	}
	return nil
}

func (r *Registry) definitions() []*tool {
	return []*tool{
		{
			spec: llm.ToolSpec{
				Name:        auth.PermSearchKB,
				Description: "Поиск по базе знаний: инструкции, процессы, инструменты команды. Используй для любых вопросов о том, как что-то сделать.",
				Parameters: json.RawMessage(`{"type":"object","properties":{
					"query":{"type":"string","description":"Вопрос или ключевые слова"}},"required":["query"]}`),
			},
			run: r.runSearchKB,
		},
		{
			spec: llm.ToolSpec{
				Name:        auth.PermProgress,
				Description: "Прогресс адаптации пользователя: шаги чек-листа, их статусы и следующий шаг.",
				Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
			},
			run: r.runProgress,
		},
		{
			spec: llm.ToolSpec{
				Name:        auth.PermWhoToAsk,
				Description: "Подсказать, к кому обратиться по вопросу: наставник, руководитель или HR BP.",
				Parameters: json.RawMessage(`{"type":"object","properties":{
					"topic":{"type":"string","description":"Тема вопроса"}}}`),
			},
			run: r.runWhoToAsk,
		},
		{
			spec: llm.ToolSpec{
				Name: auth.PermAbsenceCreate,
				Description: "Подготовить заявку на отсутствие (отгул, отпуск, больничный). Заявка уйдёт в сервис только после подтверждения пользователем. " +
					"Не придумывай даты: если их нет, спроси у пользователя.",
				Parameters: json.RawMessage(`{"type":"object","properties":{
					"kind":{"type":"string","enum":["day_off","vacation","sick"],"description":"Тип отсутствия, по умолчанию day_off"},
					"date_from":{"type":"string","description":"Первый день, YYYY-MM-DD"},
					"date_to":{"type":"string","description":"Последний день, YYYY-MM-DD; по умолчанию равен date_from"},
					"reason":{"type":"string","description":"Причина, коротко"}},"required":["date_from"]}`),
			},
			mutating: true,
			prepare:  r.prepareAbsence,
			run:      r.runCreateRequest(domain.KindAbsence),
		},
		{
			spec: llm.ToolSpec{
				Name:        auth.PermAbsenceStatus,
				Description: "Статусы заявок пользователя на отсутствие.",
				Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
			},
			run: r.runAbsenceStatus,
		},
		{
			spec: llm.ToolSpec{
				Name: auth.PermAccessRequest,
				Description: "Подготовить заявку на доступ к проекту или инструменту. Уйдёт в сервис только после подтверждения пользователем. " +
					"Перед заявкой полезно сверить с наставником, нужен ли доступ.",
				Parameters: json.RawMessage(`{"type":"object","properties":{
					"project":{"type":"string","description":"Проект или инструмент"},
					"level":{"type":"string","enum":["read","write","admin"],"description":"Уровень прав, по умолчанию read"},
					"justification":{"type":"string","description":"Обоснование, не короче 10 символов"}},"required":["project","justification"]}`),
			},
			mutating: true,
			prepare:  r.prepareAccess,
			run:      r.runCreateRequest(domain.KindAccess),
		},
	}
}

func (r *Registry) runSearchKB(ctx context.Context, id domain.Identity, args json.RawMessage) (any, *uuid.UUID, error) {
	var in struct {
		Query string `json:"query"`
	}
	if err := decode(args, &in); err != nil {
		return nil, nil, err
	}
	in.Query = strings.TrimSpace(in.Query)
	if in.Query == "" {
		return nil, nil, badArgs("query is required")
	}
	if utf8.RuneCountInString(in.Query) > 300 {
		in.Query = string([]rune(in.Query)[:300])
	}
	hits, err := r.d.KB.Search(ctx, id, in.Query, 5)
	if err != nil {
		return nil, nil, err
	}
	if hits == nil {
		hits = []domain.KBHit{}
	}
	return map[string]any{"hits": hits}, nil, nil
}

func (r *Registry) runProgress(ctx context.Context, id domain.Identity, _ json.RawMessage) (any, *uuid.UUID, error) {
	steps, err := r.d.Steps.StepsFor(ctx, id.UserID)
	if err != nil {
		return nil, nil, err
	}
	p := progress.Summarize(steps)
	waiting := make([]string, 0, len(p.Waiting))
	for _, s := range p.Waiting {
		waiting = append(waiting, s.Title)
	}
	compact := make([]map[string]any, 0, len(steps))
	for _, s := range steps {
		compact = append(compact, map[string]any{"title": s.Title, "status": s.Status, "confirm_by": s.ConfirmBy})
	}
	data := map[string]any{
		"done": p.Done, "total": p.Total, "percent": p.Percent,
		"waiting": waiting, "steps": compact, "next": nil,
	}
	if p.Next != nil {
		data["next"] = map[string]any{"title": p.Next.Title, "description": p.Next.Description, "status": p.Next.Status}
	}
	return data, nil, nil
}

func (r *Registry) runWhoToAsk(ctx context.Context, id domain.Identity, args json.RawMessage) (any, *uuid.UUID, error) {
	var in struct {
		Topic string `json:"topic"`
	}
	if err := decode(args, &in); err != nil {
		return nil, nil, err
	}
	people, err := r.d.People.WhoToAsk(ctx, id.UserID, in.Topic)
	if err != nil {
		return nil, nil, err
	}
	if people == nil {
		people = []domain.Person{}
	}
	return map[string]any{"people": people}, nil, nil
}

func (r *Registry) runAbsenceStatus(ctx context.Context, id domain.Identity, _ json.RawMessage) (any, *uuid.UUID, error) {
	list, err := r.d.Requests.ListForUser(ctx, id.UserID, domain.KindAbsence, 5)
	if err != nil {
		return nil, nil, err
	}
	out := make([]map[string]any, 0, len(list))
	for _, q := range list {
		out = append(out, map[string]any{
			"id": q.ID, "status": q.Status, "payload": json.RawMessage(q.Payload),
			"decision_comment": q.DecisionComment,
		})
	}
	return map[string]any{"requests": out}, nil, nil
}

type absenceArgs struct {
	Kind     string `json:"kind"`
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
	Reason   string `json:"reason"`
}

func (r *Registry) prepareAbsence(raw json.RawMessage, now time.Time) (json.RawMessage, string, error) {
	var a absenceArgs
	if err := decode(raw, &a); err != nil {
		return nil, "", err
	}
	a.Kind = strings.TrimSpace(a.Kind)
	if a.Kind == "" {
		a.Kind = "day_off"
	}
	if _, ok := kindLabels[a.Kind]; !ok {
		return nil, "", badArgs("kind must be one of day_off, vacation, sick")
	}
	if strings.TrimSpace(a.DateFrom) == "" {
		return nil, "", badArgs("date_from is required (YYYY-MM-DD); ask the user for the dates")
	}
	from, err := time.Parse(dateLayout, strings.TrimSpace(a.DateFrom))
	if err != nil {
		return nil, "", badArgs("date_from must look like 2026-11-20")
	}
	to := from
	if strings.TrimSpace(a.DateTo) != "" {
		if to, err = time.Parse(dateLayout, strings.TrimSpace(a.DateTo)); err != nil {
			return nil, "", badArgs("date_to must look like 2026-11-20")
		}
	}
	today, _ := time.Parse(dateLayout, now.Format(dateLayout))
	switch {
	case to.Before(from):
		return nil, "", badArgs("date_to is before date_from")
	case from.Before(today):
		return nil, "", badArgs("dates are in the past")
	case to.Sub(from) > 30*24*time.Hour:
		return nil, "", badArgs("absence longer than 30 days is not supported")
	}
	a.Reason = strings.TrimSpace(a.Reason)
	if utf8.RuneCountInString(a.Reason) > 500 {
		return nil, "", badArgs("reason is longer than 500 characters")
	}
	a.DateFrom, a.DateTo = from.Format(dateLayout), to.Format(dateLayout)

	norm, _ := json.Marshal(a)
	summary := fmt.Sprintf("%s с %s по %s", kindLabels[a.Kind], a.DateFrom, a.DateTo)
	if a.Reason != "" {
		summary += ". Причина: " + a.Reason
	}
	summary += ". Согласует: " + approverLabels["manager"] + "."
	return norm, summary, nil
}

type accessArgs struct {
	Project       string `json:"project"`
	Level         string `json:"level"`
	Justification string `json:"justification"`
}

func (r *Registry) prepareAccess(raw json.RawMessage, _ time.Time) (json.RawMessage, string, error) {
	var a accessArgs
	if err := decode(raw, &a); err != nil {
		return nil, "", err
	}
	a.Project = strings.TrimSpace(a.Project)
	a.Level = strings.TrimSpace(a.Level)
	a.Justification = strings.TrimSpace(a.Justification)
	if a.Level == "" {
		a.Level = "read"
	}
	if _, ok := levelLabels[a.Level]; !ok {
		return nil, "", badArgs("level must be one of read, write, admin")
	}
	if n := utf8.RuneCountInString(a.Project); n == 0 || n > 100 {
		return nil, "", badArgs("project is required (up to 100 characters)")
	}
	if n := utf8.RuneCountInString(a.Justification); n < 10 || n > 500 {
		return nil, "", badArgs("justification must be 10-500 characters; ask the user why the access is needed")
	}
	norm, _ := json.Marshal(a)
	summary := fmt.Sprintf("доступ к «%s», уровень: %s. Согласует: %s. Перед отправкой сверьтесь с наставником: нужен ли этот доступ.",
		a.Project, levelLabels[a.Level], approverLabels[approverByLevel[a.Level]])
	return norm, summary, nil
}

func (r *Registry) runCreateRequest(kind string) func(context.Context, domain.Identity, json.RawMessage) (any, *uuid.UUID, error) {
	return func(ctx context.Context, id domain.Identity, args json.RawMessage) (any, *uuid.UUID, error) {
		approver := "manager"
		if kind == domain.KindAccess {
			var a accessArgs
			if err := json.Unmarshal(args, &a); err != nil {
				return nil, nil, badArgs("stored arguments are corrupted")
			}
			approver = approverByLevel[a.Level]
		}
		reqID, err := r.d.Requests.Create(ctx, id.UserID, kind, args, approver, idempotencyKeyFrom(ctx))
		if err != nil {
			return nil, nil, err
		}
		return map[string]any{"request_id": reqID, "status": domain.RequestPending, "approver_role": approver}, &reqID, nil
	}
}

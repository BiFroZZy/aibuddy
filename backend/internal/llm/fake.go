package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type Fake struct{}

var (
	isoDateRe    = regexp.MustCompile(`\b(\d{4}-\d{2}-\d{2})\b`)
	ruDateRe     = regexp.MustCompile(`\b(\d{2})\.(\d{2})\.(\d{4})\b`)
	accessProjRe = regexp.MustCompile(`(?i)доступ\w*\s+(?:к|в|на)\s+([\p{L}\p{N}_\-./]+)`)
)

func (Fake) Chat(_ context.Context, msgs []Message, tools []ToolSpec) (Message, error) {
	if len(msgs) == 0 {
		return Message{Role: RoleAssistant, Content: "Чем могу помочь?"}, nil
	}
	if msgs[len(msgs)-1].Role == RoleTool {
		return Message{Role: RoleAssistant, Content: renderToolResults(msgs)}, nil
	}

	text := msgs[len(msgs)-1].Content
	name, args := route(text)
	if !offered(tools, name) {
		return Message{Role: RoleAssistant, Content: "Эта операция вам недоступна. Если она нужна для работы, уточните у руководителя или наставника."}, nil
	}
	raw, _ := json.Marshal(args)
	return Message{
		Role:      RoleAssistant,
		ToolCalls: []ToolCall{{ID: "fake_call_1", Name: name, Arguments: string(raw)}},
	}, nil
}

func offered(tools []ToolSpec, name string) bool {
	for _, t := range tools {
		if t.Name == name {
			return true
		}
	}
	return false
}

func containsAny(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func route(text string) (string, map[string]any) {
	low := strings.ToLower(text)

	if containsAny(low, "отгул", "отпуск", "больнич", "отсутств", "заявк") {
		if containsAny(low, "статус", "согласова", "одобр", "что с мо") {
			return "get_absence_status", map[string]any{}
		}
		if dates := extractDates(text); len(dates) > 0 {
			args := map[string]any{
				"kind":      absenceKind(low),
				"date_from": dates[0],
				"date_to":   dates[len(dates)-1],
				"reason":    "Заявка создана через AI-buddy",
			}
			return "create_absence_request", args
		}
		return "search_kb", map[string]any{"query": text}
	}
	if strings.Contains(low, "доступ") {
		if m := accessProjRe.FindStringSubmatch(text); m != nil {
			return "request_access", map[string]any{
				"project":       strings.TrimRight(m[1], ".,;:!?"),
				"level":         accessLevel(low),
				"justification": "Нужен для выполнения задач адаптации (запрос через AI-buddy)",
			}
		}
		return "search_kb", map[string]any{"query": text}
	}
	if containsAny(low, "к кому", "кого спросить", "кому задать", "кто поможет") {
		return "get_who_to_ask", map[string]any{"topic": text}
	}
	if containsAny(low, "прогресс", "чек-лист", "чеклист", "следующ", "что дальше", "этап", "мои шаги") {
		return "get_my_progress", map[string]any{}
	}
	return "search_kb", map[string]any{"query": text}
}

func absenceKind(low string) string {
	switch {
	case strings.Contains(low, "отпуск"):
		return "vacation"
	case strings.Contains(low, "больнич"):
		return "sick"
	}
	return "day_off"
}

func accessLevel(low string) string {
	switch {
	case containsAny(low, "админ", "admin"):
		return "admin"
	case containsAny(low, "запис", "редакт", "write"):
		return "write"
	}
	return "read"
}

func extractDates(text string) []string {
	var out []string
	out = append(out, isoDateRe.FindAllString(text, -1)...)
	for _, m := range ruDateRe.FindAllStringSubmatch(text, -1) {
		out = append(out, fmt.Sprintf("%s-%s-%s", m[3], m[2], m[1]))
	}
	return out
}

type envelope struct {
	OK      bool            `json:"ok"`
	Data    json.RawMessage `json:"data"`
	Pending *struct {
		ID      string `json:"id"`
		Summary string `json:"summary"`
	} `json:"pending_action"`
	Error string `json:"error"`
}

func renderToolResults(msgs []Message) string {
	var tail []Message
	for i := len(msgs) - 1; i >= 0 && msgs[i].Role == RoleTool; i-- {
		tail = append([]Message{msgs[i]}, tail...)
	}
	parts := make([]string, 0, len(tail))
	for _, m := range tail {
		parts = append(parts, renderOne(m))
	}
	return strings.Join(parts, "\n\n")
}

var statusRU = map[string]string{
	"pending_approval": "на согласовании",
	"approved":         "одобрена",
	"rejected":         "отклонена",
}

var kindRU = map[string]string{"day_off": "отгул", "vacation": "отпуск", "sick": "больничный"}

func renderOne(m Message) string {
	var env envelope
	if err := json.Unmarshal([]byte(m.Content), &env); err != nil {
		return "Не удалось разобрать ответ инструмента."
	}
	if !env.OK {
		return "Не получилось выполнить действие: " + env.Error
	}
	if env.Pending != nil {
		return "Подготовил заявку: " + env.Pending.Summary +
			"\nПодтвердите или отмените её в карточке ниже: пока вы не подтвердите, ничего не отправляется."
	}

	switch m.Name {
	case "search_kb":
		var d struct {
			Hits []struct{ Title, Snippet string } `json:"hits"`
		}
		_ = json.Unmarshal(env.Data, &d)
		if len(d.Hits) == 0 {
			return "В базе знаний по этому вопросу ничего не нашёл. Лучше уточнить у наставника или руководителя; могу подсказать, к кому обратиться."
		}
		var b strings.Builder
		b.WriteString("Нашёл в базе знаний:")
		for _, h := range d.Hits {
			fmt.Fprintf(&b, "\n• %s — %s", h.Title, h.Snippet)
		}
		return b.String()
	case "get_my_progress":
		var d struct {
			Done, Total, Percent int
			Next                 *struct{ Title, Description string }
			Waiting              []string
		}
		_ = json.Unmarshal(env.Data, &d)
		s := fmt.Sprintf("Выполнено %d из %d шагов (%d%%).", d.Done, d.Total, d.Percent)
		if d.Next != nil {
			s += fmt.Sprintf(" Следующий шаг: «%s» — %s", d.Next.Title, d.Next.Description)
		} else {
			s += " Свободных шагов нет: остальные выполнены или ждут подтверждения."
		}
		if len(d.Waiting) > 0 {
			s += "\nЖдут подтверждения наставника: " + strings.Join(d.Waiting, "; ")
		}
		return s
	case "get_who_to_ask":
		var d struct {
			People []struct {
				FullName string `json:"full_name"`
				Note     string `json:"note"`
			} `json:"people"`
		}
		_ = json.Unmarshal(env.Data, &d)
		if len(d.People) == 0 {
			return "Не нашёл, к кому обратиться. Напишите руководителю."
		}
		var b strings.Builder
		b.WriteString("К кому обратиться:")
		for _, p := range d.People {
			fmt.Fprintf(&b, "\n• %s — %s", p.FullName, p.Note)
		}
		return b.String()
	case "get_absence_status":
		var d struct {
			Requests []struct {
				Status  string
				Payload struct {
					Kind     string `json:"kind"`
					DateFrom string `json:"date_from"`
					DateTo   string `json:"date_to"`
				}
			} `json:"requests"`
		}
		_ = json.Unmarshal(env.Data, &d)
		if len(d.Requests) == 0 {
			return "У вас пока нет заявок на отсутствие."
		}
		var b strings.Builder
		b.WriteString("Ваши заявки на отсутствие:")
		for _, r := range d.Requests {
			fmt.Fprintf(&b, "\n• %s %s — %s: %s", kindRU[r.Payload.Kind], r.Payload.DateFrom, r.Payload.DateTo, statusRU[r.Status])
		}
		return b.String()
	}
	return "Готово."
}

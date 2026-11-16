package llm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func specs(names ...string) []ToolSpec {
	var out []ToolSpec
	for _, n := range names {
		out = append(out, ToolSpec{Name: n})
	}
	return out
}

var allTools = specs("search_kb", "get_my_progress", "get_who_to_ask", "create_absence_request", "get_absence_status", "request_access")

func TestFake_Routing(t *testing.T) {
	tests := []struct {
		text     string
		wantTool string
		check    func(t *testing.T, args map[string]any)
	}{
		{"Как оформить отгул?", "search_kb", nil},
		{"хочу отгул 2026-11-20", "create_absence_request", func(t *testing.T, a map[string]any) {
			if a["date_from"] != "2026-11-20" || a["date_to"] != "2026-11-20" || a["kind"] != "day_off" {
				t.Fatalf("args: %v", a)
			}
		}},
		{"отпуск с 01.12.2026 по 14.12.2026", "create_absence_request", func(t *testing.T, a map[string]any) {
			if a["date_from"] != "2026-12-01" || a["date_to"] != "2026-12-14" || a["kind"] != "vacation" {
				t.Fatalf("args: %v", a)
			}
		}},
		{"больничный 2026-11-20", "create_absence_request", func(t *testing.T, a map[string]any) {
			if a["kind"] != "sick" {
				t.Fatalf("args: %v", a)
			}
		}},
		{"Какой статус моей заявки на отгул?", "get_absence_status", nil},
		{"Нужен доступ к billing-service на запись", "request_access", func(t *testing.T, a map[string]any) {
			if a["project"] != "billing-service" || a["level"] != "write" {
				t.Fatalf("args: %v", a)
			}
		}},
		{"дай доступ в repo-x админский", "request_access", func(t *testing.T, a map[string]any) {
			if a["level"] != "admin" {
				t.Fatalf("args: %v", a)
			}
		}},
		{"как получить доступы?", "search_kb", nil},
		{"К кому обратиться по бенефитам?", "get_who_to_ask", nil},
		{"Покажи мой прогресс", "get_my_progress", nil},
		{"что дальше?", "get_my_progress", nil},
		{"что такое Kafka?", "search_kb", nil},
	}
	for _, tc := range tests {
		t.Run(tc.text, func(t *testing.T) {
			out, err := Fake{}.Chat(context.Background(), []Message{{Role: RoleUser, Content: tc.text}}, allTools)
			if err != nil || len(out.ToolCalls) != 1 || out.ToolCalls[0].Name != tc.wantTool {
				t.Fatalf("got %+v %v, want tool %s", out, err, tc.wantTool)
			}
			if tc.check != nil {
				var args map[string]any
				if err := json.Unmarshal([]byte(out.ToolCalls[0].Arguments), &args); err != nil {
					t.Fatal(err)
				}
				tc.check(t, args)
			}
		})
	}
}

func TestFake_OnlyOfferedTools(t *testing.T) {
	out, err := Fake{}.Chat(context.Background(),
		[]Message{{Role: RoleUser, Content: "оформи отгул 2026-11-20"}}, specs("search_kb", "get_who_to_ask"))
	if err != nil || len(out.ToolCalls) != 0 || !strings.Contains(out.Content, "недоступна") {
		t.Fatalf("a tool outside the role must not be called: %+v %v", out, err)
	}
	if out, _ := (Fake{}).Chat(context.Background(), nil, nil); out.Content == "" {
		t.Fatal("empty input still gets a reply")
	}
}

func toolMsg(name, content string) []Message {
	return []Message{{Role: RoleUser, Content: "q"}, {Role: RoleTool, Name: name, Content: content}}
}

func TestFake_RenderToolResults(t *testing.T) {
	tests := []struct {
		name, tool, content string
		want                []string
	}{
		{"kb hits", "search_kb", `{"ok":true,"data":{"hits":[{"title":"Как оформить отгул","snippet":"через сервис"}]}}`, []string{"Как оформить отгул", "через сервис"}},
		{"kb miss", "search_kb", `{"ok":true,"data":{"hits":[]}}`, []string{"ничего не нашёл"}},
		{"progress", "get_my_progress", `{"ok":true,"data":{"done":2,"total":7,"percent":28,"next":{"title":"Доступы","description":"Git"},"waiting":["Первая задача"]}}`, []string{"2 из 7", "28%", "«Доступы»", "Первая задача"}},
		{"progress without next", "get_my_progress", `{"ok":true,"data":{"done":7,"total":7,"percent":100,"next":null}}`, []string{"Свободных шагов нет"}},
		{"who", "get_who_to_ask", `{"ok":true,"data":{"people":[{"full_name":"Наталья","note":"наставник"}]}}`, []string{"Наталья", "наставник"}},
		{"who empty", "get_who_to_ask", `{"ok":true,"data":{"people":[]}}`, []string{"Напишите руководителю"}},
		{"absence status", "get_absence_status", `{"ok":true,"data":{"requests":[{"status":"pending_approval","payload":{"kind":"day_off","date_from":"2026-11-20","date_to":"2026-11-20"}}]}}`, []string{"отгул", "2026-11-20", "на согласовании"}},
		{"no absences", "get_absence_status", `{"ok":true,"data":{"requests":[]}}`, []string{"нет заявок"}},
		{"pending", "create_absence_request", `{"ok":true,"pending_action":{"id":"x","summary":"отгул 2026-11-20"}}`, []string{"Подготовил заявку", "отгул 2026-11-20", "Подтвердите"}},
		{"error", "request_access", `{"ok":false,"error":"invalid arguments: project is required"}`, []string{"Не получилось", "project is required"}},
		{"garbage", "search_kb", `<<<`, []string{"Не удалось разобрать"}},
		{"unknown tool", "other", `{"ok":true}`, []string{"Готово"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Fake{}.Chat(context.Background(), toolMsg(tc.tool, tc.content), allTools)
			if err != nil || len(out.ToolCalls) != 0 {
				t.Fatalf("after a tool result the fake must answer in text: %+v %v", out, err)
			}
			for _, w := range tc.want {
				if !strings.Contains(out.Content, w) {
					t.Fatalf("reply %q does not contain %q", out.Content, w)
				}
			}
		})
	}
}

func TestFake_SeveralToolResultsAreCombined(t *testing.T) {
	msgs := []Message{
		{Role: RoleUser, Content: "q"},
		{Role: RoleTool, Name: "get_my_progress", Content: `{"ok":true,"data":{"done":1,"total":2,"percent":50}}`},
		{Role: RoleTool, Name: "get_who_to_ask", Content: `{"ok":true,"data":{"people":[{"full_name":"Н","note":"n"}]}}`},
	}
	out, _ := Fake{}.Chat(context.Background(), msgs, allTools)
	if !strings.Contains(out.Content, "1 из 2") || !strings.Contains(out.Content, "К кому обратиться") {
		t.Fatalf("both results must be rendered: %q", out.Content)
	}
}

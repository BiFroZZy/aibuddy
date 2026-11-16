package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"aibuddy/internal/app"
	"aibuddy/internal/clients/ml"
	"aibuddy/internal/config"
	"aibuddy/internal/domain"
	"aibuddy/internal/llm"
	"aibuddy/internal/mockhr"
	"aibuddy/internal/repository"
	"aibuddy/internal/testdb"

	"github.com/google/uuid"
)

func logger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

type client struct {
	t   *testing.T
	srv *httptest.Server
}

func newClient(t *testing.T, model llm.Client) *client {
	t.Helper()
	db := testdb.New(t)
	srv := httptest.NewServer(app.NewHandler(config.Config{}, db, logger(), model))
	t.Cleanup(srv.Close)
	return &client{t: t, srv: srv}
}

func newClientViaHR(t *testing.T) (*client, func()) {
	t.Helper()
	db := testdb.New(t)
	hrSrv := httptest.NewServer(mockhr.NewHandler(&repository.Requests{DB: db}, logger()))
	t.Cleanup(hrSrv.Close)
	srv := httptest.NewServer(app.NewHandler(config.Config{HRURL: hrSrv.URL}, db, logger(), llm.Fake{}))
	t.Cleanup(srv.Close)
	return &client{t: t, srv: srv}, hrSrv.Close
}

func (c *client) call(method, path, token string, body any, out any) int {
	c.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.srv.URL+path, rd)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			c.t.Fatalf("%s %s: bad JSON %q", method, path, raw)
		}
	}
	return resp.StatusCode
}

type chatResp struct {
	ConversationID string   `json:"conversation_id"`
	Reply          string   `json:"reply"`
	ToolsUsed      []string `json:"tools_used"`
	PendingActions []struct {
		ID      string `json:"id"`
		Tool    string `json:"tool"`
		Status  string `json:"status"`
		Summary string `json:"summary"`
	} `json:"pending_actions"`
}

func (c *client) chat(token, msg string) chatResp {
	c.t.Helper()
	var out chatResp
	if code := c.call("POST", "/api/chat", token, map[string]string{"message": msg}, &out); code != 200 {
		c.t.Fatalf("chat %q: status %d", msg, code)
	}
	return out
}

func TestScenario_AbsenceRequestEndToEnd(t *testing.T) {
	runAbsenceScenario(t, newClient(t, llm.Fake{}))
}

func TestScenario_AbsenceRequestEndToEnd_ViaMockHR(t *testing.T) {
	c, _ := newClientViaHR(t)
	runAbsenceScenario(t, c)
}

func TestScenario_CorporateServiceOutage(t *testing.T) {
	c, stopHR := newClientViaHR(t)

	r := c.chat(testdb.InternToken, "Оформи отгул на 2030-01-10")
	if len(r.PendingActions) != 1 {
		t.Fatalf("proposal must not need the corporate service: %+v", r)
	}
	stopHR()

	var failed struct{ Status, Error string }
	if code := c.call("POST", "/api/actions/"+r.PendingActions[0].ID+"/confirm", testdb.InternToken, nil, &failed); code != 200 ||
		failed.Status != "failed" || !strings.Contains(failed.Error, "temporarily unavailable") {
		t.Fatalf("a failed action must be reported, without transport details: %d %+v", code, failed)
	}
	if strings.Contains(failed.Error, "127.0.0.1") || strings.Contains(failed.Error, "connect") {
		t.Fatalf("transport details leaked to the user: %q", failed.Error)
	}

	var body struct{ Error string }
	if code := c.call("GET", "/api/manager/requests", testdb.ManagerToken, nil, &body); code != 502 {
		t.Fatalf("manager queue during an outage: %d", code)
	}
	var p struct{ Total int }
	if code := c.call("GET", "/api/progress", testdb.InternToken, nil, &p); code != 200 || p.Total != 7 {
		t.Fatalf("progress: %d %+v", code, p)
	}
}

func runAbsenceScenario(t *testing.T, c *client) {

	r := c.chat(testdb.InternToken, "Как оформить отгул?")
	if !strings.Contains(r.Reply, "сервис отсутствий") || len(r.ToolsUsed) != 1 || r.ToolsUsed[0] != "search_kb" {
		t.Fatalf("kb answer: %+v", r)
	}

	r = c.chat(testdb.InternToken, "Оформи мне отгул на 2030-01-10")
	if len(r.PendingActions) != 1 || r.PendingActions[0].Status != "pending_confirmation" || !strings.Contains(r.PendingActions[0].Summary, "2030-01-10") {
		t.Fatalf("pending action: %+v", r)
	}
	var mine struct{ Requests []map[string]any }
	c.call("GET", "/api/requests", testdb.InternToken, nil, &mine)
	if len(mine.Requests) != 0 {
		t.Fatal("no request may exist before confirmation")
	}
	actionID := r.PendingActions[0].ID

	if code := c.call("POST", "/api/actions/"+actionID+"/confirm", testdb.Intern2Token, nil, nil); code != 404 {
		t.Fatalf("foreign confirm: %d", code)
	}

	var done struct {
		Status     string `json:"status"`
		ExternalID string `json:"external_id"`
	}
	if code := c.call("POST", "/api/actions/"+actionID+"/confirm", testdb.InternToken, nil, &done); code != 200 || done.Status != "done" || done.ExternalID == "" {
		t.Fatalf("confirm: %d %+v", code, done)
	}
	if code := c.call("POST", "/api/actions/"+actionID+"/confirm", testdb.InternToken, nil, nil); code != 409 {
		t.Fatalf("double confirm must be a conflict: %d", code)
	}
	c.call("GET", "/api/requests", testdb.InternToken, nil, &mine)
	if len(mine.Requests) != 1 || mine.Requests[0]["status"] != "pending_approval" {
		t.Fatalf("filed request: %+v", mine)
	}

	var pending struct {
		Requests []struct {
			ID       string `json:"id"`
			UserName string `json:"user_name"`
		}
	}
	c.call("GET", "/api/manager/requests", testdb.ManagerToken, nil, &pending)
	if len(pending.Requests) != 1 || pending.Requests[0].ID != mine.Requests[0]["id"] {
		t.Fatalf("manager queue: %+v", pending)
	}
	if code := c.call("GET", "/api/manager/requests", testdb.MentorToken, nil, nil); code != 403 {
		t.Fatalf("a mentor must not see the manager queue: %d", code)
	}
	if code := c.call("POST", "/api/manager/requests/"+pending.Requests[0].ID+"/decision", testdb.ManagerToken,
		map[string]string{"decision": "approve", "comment": "хорошего отдыха"}, nil); code != 200 {
		t.Fatalf("approve: %d", code)
	}

	r = c.chat(testdb.InternToken, "Какой статус моей заявки на отгул?")
	if !strings.Contains(r.Reply, "одобрена") || !strings.Contains(r.Reply, "2030-01-10") {
		t.Fatalf("status answer: %q", r.Reply)
	}

	var hist struct {
		Messages []struct{ Role, Content string }
	}
	if code := c.call("GET", "/api/conversations/"+r.ConversationID+"/messages", testdb.InternToken, nil, &hist); code != 200 || len(hist.Messages) != 2 {
		t.Fatalf("history: %d %+v", code, hist)
	}
	if code := c.call("GET", "/api/conversations/"+r.ConversationID+"/messages", testdb.Intern2Token, nil, nil); code != 404 {
		t.Fatalf("foreign history: %d", code)
	}

	var acts struct {
		Actions []struct{ Status, Tool string }
	}
	c.call("GET", "/api/actions", testdb.InternToken, nil, &acts)
	if len(acts.Actions) != 1 || acts.Actions[0].Status != "done" {
		t.Fatalf("actions: %+v", acts)
	}
	var m domain.Metrics
	if code := c.call("GET", "/api/metrics", testdb.HRToken, nil, &m); code != 200 {
		t.Fatalf("metrics: %d", code)
	}
	if m.Events["chat_message"] != 3 || m.Events["action_confirmed"] != 1 || m.Tools["search_kb"] != 1 || m.ActiveUsers < 1 {
		t.Fatalf("metrics: %+v", m)
	}
}

func TestScenario_RightsAreEnforced(t *testing.T) {
	c := newClient(t, llm.Fake{})

	r := c.chat(testdb.HRToken, "оформи отгул 2030-01-10")
	if len(r.PendingActions) != 0 || !strings.Contains(r.Reply, "недоступна") {
		t.Fatalf("hr: %+v", r)
	}
	var acts struct{ Actions []any }
	c.call("GET", "/api/actions", testdb.HRToken, nil, &acts)
	if len(acts.Actions) != 0 {
		t.Fatal("nothing may be recorded for a tool the role cannot see")
	}

	const memoQuestion = "памятка руководителя: как проверять даты у стажёров"
	r = c.chat(testdb.InternToken, memoQuestion)
	if !strings.Contains(r.Reply, "Нашёл в базе знаний") || strings.Contains(r.Reply, "Памятка руководителя") {
		t.Fatalf("intern must get an answer without the manager document: %q", r.Reply)
	}
	if r := c.chat(testdb.ManagerToken, memoQuestion); !strings.Contains(r.Reply, "Памятка руководителя") {
		t.Fatalf("manager must see it: %q", r.Reply)
	}

	if r := c.chat(testdb.InternToken, "расписание дежурств"); strings.Contains(r.Reply, "дежурств другой") {
		t.Fatalf("cross-team leak: %q", r.Reply)
	}

	if code := c.call("GET", "/api/metrics", testdb.InternToken, nil, nil); code != 403 {
		t.Fatalf("metrics for an intern: %d", code)
	}
	if code := c.call("GET", "/api/me", "bad-token", nil, nil); code != 401 {
		t.Fatalf("bad token: %d", code)
	}
}

func TestScenario_AccessRequestAndWhoToAsk(t *testing.T) {
	c := newClient(t, llm.Fake{})

	r := c.chat(testdb.InternToken, "Нужен доступ к billing-service на запись")
	if len(r.PendingActions) != 1 || !strings.Contains(r.PendingActions[0].Summary, "руководитель") || !strings.Contains(r.PendingActions[0].Summary, "наставником") {
		t.Fatalf("access proposal: %+v", r)
	}
	var cancelled struct{ Status string }
	c.call("POST", "/api/actions/"+r.PendingActions[0].ID+"/cancel", testdb.InternToken, nil, &cancelled)
	if cancelled.Status != "cancelled" {
		t.Fatalf("cancel: %+v", cancelled)
	}
	var mine struct{ Requests []any }
	c.call("GET", "/api/requests", testdb.InternToken, nil, &mine)
	if len(mine.Requests) != 0 {
		t.Fatal("a cancelled request must not be filed")
	}

	r = c.chat(testdb.InternToken, "К кому обратиться по бенефитам?")
	for _, want := range []string{"Михаил Наставник", "Мария Руководитель", "Ольга HR"} {
		if !strings.Contains(r.Reply, want) {
			t.Fatalf("who to ask must include %s: %q", want, r.Reply)
		}
	}
}

func TestScenario_ChecklistProgressAndMentorConfirmation(t *testing.T) {
	c := newClient(t, llm.Fake{})
	type steps struct {
		Done, Total, Percent int
		Next                 *struct{ Title string }
		Waiting              []struct{ Title string }
	}
	var p steps
	c.call("GET", "/api/progress", testdb.InternToken, nil, &p)
	if p.Total != 7 || p.Done != 1 || p.Next == nil || !strings.Contains(p.Next.Title, "цели") {
		t.Fatalf("seeded progress: %+v", p)
	}
	if r := c.chat(testdb.InternToken, "Покажи мой прогресс"); !strings.Contains(r.Reply, "1 из 7") {
		t.Fatalf("chat progress: %q", r.Reply)
	}

	step := func(id uuid.UUID, status string) int {
		return c.call("POST", "/api/progress/steps/"+id.String()+"/status", testdb.InternToken, map[string]string{"status": status}, &p)
	}
	if code := step(testdb.Step3, "done"); code != 200 || p.Done != 2 {
		t.Fatalf("self-confirmed step: %d %+v", code, p)
	}
	if code := step(testdb.Step2, "done"); code != 403 {
		t.Fatalf("an intern must not close a mentor step: %d", code)
	}
	if code := step(testdb.Step3, "in_progress"); code != 409 {
		t.Fatalf("a done step is final: %d", code)
	}
	if code := step(testdb.Step7, "waiting"); code != 200 || len(p.Waiting) != 1 {
		t.Fatalf("asking the mentor: %d %+v", code, p)
	}

	var conf struct {
		Confirmations []struct {
			InternID string `json:"intern_id"`
			StepID   string `json:"step_id"`
		}
	}
	c.call("GET", "/api/mentor/confirmations", testdb.MentorToken, nil, &conf)
	if len(conf.Confirmations) != 1 || conf.Confirmations[0].StepID != testdb.Step7.String() {
		t.Fatalf("mentor queue: %+v", conf)
	}
	path := "/api/mentor/interns/" + conf.Confirmations[0].InternID + "/steps/" + conf.Confirmations[0].StepID + "/decision"
	if code := c.call("POST", path, testdb.ManagerToken, map[string]string{"decision": "approve"}, nil); code != 403 {
		t.Fatalf("a manager cannot confirm steps: %d", code)
	}
	if code := c.call("POST", path, testdb.MentorToken, map[string]string{"decision": "approve"}, nil); code != 200 {
		t.Fatalf("mentor approve: %d", code)
	}
	c.call("GET", "/api/progress", testdb.InternToken, nil, &p)
	if p.Done != 3 || len(p.Waiting) != 0 {
		t.Fatalf("after confirmation: %+v", p)
	}
}

type brokenLLM struct{}

func (brokenLLM) Chat(context.Context, []llm.Message, []llm.ToolSpec) (llm.Message, error) {
	return llm.Message{}, domain.ErrLLMUnavailable
}

func TestScenario_LLMOutageIsReportedCleanly(t *testing.T) {
	c := newClient(t, brokenLLM{})
	var body struct{ Error string }
	if code := c.call("POST", "/api/chat", testdb.InternToken, map[string]string{"message": "привет"}, &body); code != 502 || !strings.Contains(body.Error, "temporarily unavailable") {
		t.Fatalf("outage: %d %+v", code, body)
	}
	var p struct{ Total int }
	if code := c.call("GET", "/api/progress", testdb.InternToken, nil, &p); code != 200 || p.Total != 7 {
		t.Fatalf("progress during an outage: %d %+v", code, p)
	}
	if code := c.call("GET", "/readyz", "", nil, nil); code != 200 {
		t.Fatalf("readyz: %d", code)
	}
}

func TestNewLLMAndMigrate(t *testing.T) {
	if _, ok := app.NewLLM(config.LLMConfig{Mode: config.LLMModeFake}, nil, logger()).(llm.Fake); !ok {
		t.Fatal("fake mode must return the fake model")
	}
	real := app.NewLLM(config.LLMConfig{Mode: config.LLMModeOpenAI, BaseURL: "http://x", Model: "m", APIKey: "k"}, nil, logger())
	if _, ok := real.(*llm.OpenAI); !ok {
		t.Fatalf("openai mode: %T", real)
	}

	db := testdb.Empty(t)
	if err := app.Migrate(context.Background(), db, logger()); err != nil {
		t.Fatal(err)
	}
	if err := app.Migrate(context.Background(), db, logger()); err != nil {
		t.Fatalf("migrations must be idempotent: %v", err)
	}
	_ = db.Close()
	if err := app.Migrate(context.Background(), db, logger()); err == nil {
		t.Fatal("expected an error on a closed database")
	}
}

type fakeMLService struct {
	mu       sync.Mutex
	requests []ml.ChatRequest
}

func (f *fakeMLService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req ml.ChatRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.mu.Unlock()

	last := req.History[len(req.History)-1]
	var resp ml.ChatResponse
	switch {
	case last.Role == "tool" && strings.Contains(last.Content, "pending_action"):
		resp.Text = "Заявка подготовлена, подтвердите её в карточке."
	case last.Role == "tool":
		resp.Text = "Действие не выполнено: " + last.Content
	case strings.Contains(last.Content, "2030-01-10"):
		resp.ToolCall = &ml.ToolCall{Name: "create_absence_request", Args: json.RawMessage(`{"date_from":"2030-01-10"}`)}
	case len(req.KB) > 0:
		resp.Text = "По базе знаний: " + req.KB[0].Content
	default:
		resp.Text = "Не нашёл в базе знаний."
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (f *fakeMLService) last() ml.ChatRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[len(f.requests)-1]
}

func newClientViaML(t *testing.T) (*client, *fakeMLService) {
	t.Helper()
	db := testdb.New(t)
	svc := &fakeMLService{}
	mlSrv := httptest.NewServer(svc)
	t.Cleanup(mlSrv.Close)
	cfg := config.Config{LLM: config.LLMConfig{Mode: config.LLMModeML, MLURL: mlSrv.URL, Timeout: 5 * time.Second}}
	srv := httptest.NewServer(app.NewHandler(cfg, db, logger(), app.NewLLM(cfg.LLM, db, logger())))
	t.Cleanup(srv.Close)
	return &client{t: t, srv: srv}, svc
}

func TestScenario_MLServiceMode(t *testing.T) {
	c, svc := newClientViaML(t)

	r := c.chat(testdb.InternToken, "Как оформить отгул?")
	if !strings.Contains(r.Reply, "сервис отсутствий") {
		t.Fatalf("answer built from the retrieved fragment: %q", r.Reply)
	}
	req := svc.last()
	if req.User.Role != "intern" || req.User.Team != "Backend (тестовая команда)" || !strings.Contains(req.User.Stage, "цели") {
		t.Fatalf("user context: %+v", req.User)
	}
	for _, tl := range req.Tools {
		if tl.Name == "search_kb" {
			t.Fatal("search_kb must not be offered: retrieval is the backend's job")
		}
	}
	names := []string{}
	for _, tl := range req.Tools {
		names = append(names, tl.Name)
	}
	if len(names) != 5 {
		t.Fatalf("tools offered to an intern: %v", names)
	}

	r = c.chat(testdb.InternToken, "Оформи отгул на 2030-01-10")
	if len(r.PendingActions) != 1 || !strings.Contains(r.Reply, "подтвердите") {
		t.Fatalf("proposal: %+v", r)
	}
	last := svc.last().History[len(svc.last().History)-1]
	if last.Role != "tool" || !strings.Contains(last.Content, "pending_action") {
		t.Fatalf("the tool result must reach ML for the final answer: %+v", last)
	}
	var done struct{ Status string }
	if code := c.call("POST", "/api/actions/"+r.PendingActions[0].ID+"/confirm", testdb.InternToken, nil, &done); code != 200 || done.Status != "done" {
		t.Fatalf("confirm: %d %+v", code, done)
	}
}

func TestScenario_MLServiceCannotExceedRights(t *testing.T) {
	c, svc := newClientViaML(t)

	r := c.chat(testdb.HRToken, "Оформи отгул на 2030-01-10")
	for _, tl := range svc.requests[0].Tools {
		if tl.Name == "create_absence_request" {
			t.Fatal("the ML service must only be offered the tools of the caller's role")
		}
	}
	if len(r.PendingActions) != 0 || !strings.Contains(r.Reply, "forbidden") {
		t.Fatalf("forbidden call must not produce an action: %+v", r)
	}
	var acts struct{ Actions []struct{ Status string } }
	c.call("GET", "/api/actions", testdb.HRToken, nil, &acts)
	if len(acts.Actions) != 1 || acts.Actions[0].Status != "denied" {
		t.Fatalf("the attempt must be in the audit log: %+v", acts)
	}
	var m domain.Metrics
	c.call("GET", "/api/metrics", testdb.HRToken, nil, &m)
	if m.Events["tool_denied"] != 1 {
		t.Fatalf("metrics: %+v", m.Events)
	}
}

func TestScenario_MLServiceDown(t *testing.T) {
	db := testdb.New(t)
	dead := httptest.NewServer(http.NotFoundHandler())
	dead.Close()
	cfg := config.Config{LLM: config.LLMConfig{Mode: config.LLMModeML, MLURL: dead.URL, Timeout: time.Second}}
	srv := httptest.NewServer(app.NewHandler(cfg, db, logger(), app.NewLLM(cfg.LLM, db, logger())))
	defer srv.Close()
	c := &client{t: t, srv: srv}

	var body struct{ Error string }
	if code := c.call("POST", "/api/chat", testdb.InternToken, map[string]string{"message": "привет"}, &body); code != 502 {
		t.Fatalf("ML outage: %d", code)
	}
	var p struct{ Total int }
	if code := c.call("GET", "/api/progress", testdb.InternToken, nil, &p); code != 200 || p.Total != 7 {
		t.Fatalf("progress: %d", code)
	}
}

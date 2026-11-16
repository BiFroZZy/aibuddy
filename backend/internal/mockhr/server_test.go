package mockhr_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aibuddy/internal/domain"
	"aibuddy/internal/mockhr"
	"aibuddy/internal/repository"
	"aibuddy/internal/testdb"

	"github.com/google/uuid"
)

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	db := testdb.New(t)
	srv := httptest.NewServer(mockhr.NewHandler(&repository.Requests{DB: db}, slog.New(slog.NewTextHandler(io.Discard, nil))))
	t.Cleanup(srv.Close)
	return srv
}

func call(t *testing.T, srv *httptest.Server, method, path, idem, body string, out any) int {
	t.Helper()
	req, _ := http.NewRequest(method, srv.URL+path, bytes.NewBufferString(body))
	if idem != "" {
		req.Header.Set("Idempotency-Key", idem)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if out != nil {
		_ = json.NewDecoder(resp.Body).Decode(out)
	}
	return resp.StatusCode
}

func createBody(user uuid.UUID, kind string) string {
	return `{"user_id":"` + user.String() + `","kind":"` + kind + `","payload":{"date_from":"2030-01-10"},"approver_role":"manager"}`
}

func TestCreateIsIdempotent(t *testing.T) {
	srv := newServer(t)
	var a, b, c domain.Request
	if code := call(t, srv, "POST", "/requests", "key-1", createBody(testdb.InternID, "absence"), &a); code != 200 || a.ID == uuid.Nil || a.Status != domain.RequestPending {
		t.Fatalf("create: %d %+v", code, a)
	}
	call(t, srv, "POST", "/requests", "key-1", createBody(testdb.InternID, "absence"), &b)
	call(t, srv, "POST", "/requests", "key-2", createBody(testdb.InternID, "absence"), &c)
	if b.ID != a.ID || c.ID == a.ID {
		t.Fatalf("idempotency: a=%s b=%s c=%s", a.ID, b.ID, c.ID)
	}
	var list struct{ Requests []domain.Request }
	call(t, srv, "GET", "/requests?user_id="+testdb.InternID.String(), "", "", &list)
	if len(list.Requests) != 2 {
		t.Fatalf("exactly two requests expected, got %d", len(list.Requests))
	}
}

func TestListGetAndDecide(t *testing.T) {
	srv := newServer(t)
	var made domain.Request
	call(t, srv, "POST", "/requests", "k", createBody(testdb.InternID, "access"), &made)
	call(t, srv, "POST", "/requests", "k2", createBody(testdb.InternID, "absence"), nil)

	var got domain.Request
	if code := call(t, srv, "GET", "/requests/"+made.ID.String(), "", "", &got); code != 200 || got.UserName != "Анна Стажёр" {
		t.Fatalf("get: %d %+v", code, got)
	}
	var byKind struct{ Requests []domain.Request }
	call(t, srv, "GET", "/requests?user_id="+testdb.InternID.String()+"&kind=access&limit=5", "", "", &byKind)
	if len(byKind.Requests) != 1 || byKind.Requests[0].Kind != "access" {
		t.Fatalf("filter by kind: %+v", byKind)
	}

	var queue struct{ Requests []domain.Request }
	call(t, srv, "GET", "/requests?manager_id="+testdb.ManagerID.String(), "", "", &queue)
	if len(queue.Requests) != 2 {
		t.Fatalf("manager queue: %d", len(queue.Requests))
	}
	call(t, srv, "GET", "/requests?manager_id="+testdb.MentorID.String(), "", "", &queue)
	if len(queue.Requests) != 0 {
		t.Fatal("only the direct manager has a queue")
	}

	body := func(m uuid.UUID) string {
		return `{"manager_id":"` + m.String() + `","approve":true,"comment":"ок"}`
	}
	if code := call(t, srv, "POST", "/requests/"+made.ID.String()+"/decision", "", body(testdb.MentorID), nil); code != 404 {
		t.Fatalf("a foreign manager must not decide: %d", code)
	}
	var decided domain.Request
	if code := call(t, srv, "POST", "/requests/"+made.ID.String()+"/decision", "", body(testdb.ManagerID), &decided); code != 200 || decided.Status != domain.RequestApproved || decided.DecisionComment != "ок" {
		t.Fatalf("decide: %d %+v", code, decided)
	}
	if code := call(t, srv, "POST", "/requests/"+made.ID.String()+"/decision", "", body(testdb.ManagerID), nil); code != 404 {
		t.Fatalf("a decided request cannot be decided again: %d", code)
	}
}

func TestValidationAndErrors(t *testing.T) {
	srv := newServer(t)
	uid := testdb.InternID
	bad := []struct{ method, path, body string }{
		{"POST", "/requests", `not json`},
		{"POST", "/requests", `{"user_id":"` + uid.String() + `","kind":"hack","payload":{},"approver_role":"manager"}`},
		{"POST", "/requests", `{"kind":"absence","payload":{},"approver_role":"manager"}`},
		{"POST", "/requests", `{"user_id":"` + uid.String() + `","kind":"absence","payload":{},"approver_role":"manager","x":1}`},
		{"GET", "/requests", ""},
		{"GET", "/requests?user_id=nope", ""},
		{"GET", "/requests?manager_id=nope", ""},
		{"GET", "/requests?user_id=" + uid.String() + "&limit=0", ""},
		{"GET", "/requests/nope", ""},
		{"POST", "/requests/nope/decision", `{}`},
		{"POST", "/requests/" + uuid.NewString() + "/decision", `{"x":1}`},
	}
	for _, tc := range bad {
		if code := call(t, srv, tc.method, tc.path, "", tc.body, nil); code != 400 {
			t.Fatalf("%s %s %.30s: want 400, got %d", tc.method, tc.path, tc.body, code)
		}
	}
	if code := call(t, srv, "GET", "/requests/"+uuid.NewString(), "", "", nil); code != 404 {
		t.Fatalf("unknown request: %d", code)
	}
	if code := call(t, srv, "GET", "/healthz", "", "", nil); code != 200 {
		t.Fatalf("healthz: %d", code)
	}
	if code := call(t, srv, "POST", "/requests", "", createBody(uuid.New(), "absence"), nil); code != 500 {
		t.Fatalf("unknown user must be reported as an internal error: %d", code)
	}
	_ = strings.TrimSpace
}

package hr_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aibuddy/internal/clients/hr"
	"aibuddy/internal/domain"
	"aibuddy/internal/mockhr"
	"aibuddy/internal/repository"
	"aibuddy/internal/testdb"

	"github.com/google/uuid"
)

var ctx = context.Background()

func TestClientAgainstMockHR(t *testing.T) {
	db := testdb.New(t)
	srv := httptest.NewServer(mockhr.NewHandler(&repository.Requests{DB: db}, slog.New(slog.NewTextHandler(io.Discard, nil))))
	defer srv.Close()
	c := hr.NewHTTPClient(srv.URL, 2*time.Second)

	payload := json.RawMessage(`{"kind":"day_off","date_from":"2030-01-10","date_to":"2030-01-10"}`)
	id, err := c.Create(ctx, testdb.InternID, domain.KindAbsence, payload, "manager", "action-1")
	if err != nil || id == uuid.Nil {
		t.Fatalf("create: %v %v", id, err)
	}
	again, err := c.Create(ctx, testdb.InternID, domain.KindAbsence, payload, "manager", "action-1")
	if err != nil || again != id {
		t.Fatalf("the same idempotency key must return the same request: %v %v", again, err)
	}

	list, err := c.ListForUser(ctx, testdb.InternID, domain.KindAbsence, 10)
	if err != nil || len(list) != 1 || list[0].ID != id {
		t.Fatalf("list: %+v %v", list, err)
	}
	queue, err := c.PendingForManager(ctx, testdb.ManagerID)
	if err != nil || len(queue) != 1 || queue[0].UserName == "" {
		t.Fatalf("queue: %+v %v", queue, err)
	}

	if _, err := c.Decide(ctx, testdb.MentorID, id, true, ""); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("foreign manager: %v", err)
	}
	got, err := c.Decide(ctx, testdb.ManagerID, id, false, "занято")
	if err != nil || got.Status != domain.RequestRejected {
		t.Fatalf("decide: %+v %v", got, err)
	}
}

func TestClientErrorMapping(t *testing.T) {
	status := http.StatusOK
	body := `{}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	c := hr.NewHTTPClient(srv.URL, time.Second)
	uid := uuid.New()

	tests := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"not found", 404, `{}`, domain.ErrNotFound},
		{"bad request", 400, `{}`, domain.ErrInvalidArgs},
		{"server error", 500, `{}`, domain.ErrUnavailable},
		{"gateway", 502, `{}`, domain.ErrUnavailable},
		{"broken json", 200, `<html>`, domain.ErrUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, body = tc.status, tc.body
			_, err := c.Create(ctx, uid, domain.KindAbsence, json.RawMessage(`{}`), "manager", "k")
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}

	srv.Close()
	_, err := c.ListForUser(ctx, uid, "", 5)
	if !errors.Is(err, domain.ErrUnavailable) || err.Error() != domain.ErrUnavailable.Error() {
		t.Fatalf("unreachable: %v", err)
	}
	if _, err := c.PendingForManager(ctx, uid); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("pending: %v", err)
	}
	if _, err := c.Decide(ctx, uid, uuid.New(), true, ""); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("decide: %v", err)
	}
}

func TestClientBadURL(t *testing.T) {
	c := hr.NewHTTPClient("http://bad host", time.Second)
	if _, err := c.ListForUser(ctx, uuid.New(), "", 5); err == nil {
		t.Fatal("an invalid URL must fail")
	}
}

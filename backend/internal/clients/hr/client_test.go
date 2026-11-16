package hr

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
func TestCreateAbsenceRequestAndDecode(t *testing.T) {
	uid, xid := uuid.New(), uuid.New()
	c := NewHTTPClient("https://hr.example")
	c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/absences" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Idempotency-Key") != "idem-1" {
			t.Errorf("headers %v", r.Header)
		}
		var req CreateAbsenceReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.UserID != uid || req.Type != "day_off" {
			t.Errorf("body %#v", req)
		}
		return response(201, `{"id":"`+xid.String()+`","status":"pending"}`), nil
	})}
	got, err := c.CreateAbsence(context.Background(), CreateAbsenceReq{UserID: uid, Type: "day_off"}, "idem-1")
	if err != nil || got.ID != xid || got.Status != "pending" {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}
func TestGetAbsence(t *testing.T) {
	id := uuid.New()
	c := NewHTTPClient("https://hr.example")
	c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet || r.URL.Path != "/absences/abc" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		return response(200, `{"id":"`+id.String()+`"}`), nil
	})}
	got, err := c.GetAbsence(context.Background(), "abc")
	if err != nil || got.ID != id {
		t.Fatalf("%#v %v", got, err)
	}
}
func TestClientErrors(t *testing.T) {
	for _, tc := range []struct {
		name         string
		resp         *http.Response
		transportErr error
	}{{"status", response(503, "down"), nil}, {"malformed json", response(200, "not-json"), nil}, {"transport", nil, errors.New("network")}} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewHTTPClient("https://hr.example")
			c.HTTP = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return tc.resp, tc.transportErr })}
			if _, err := c.GetAbsence(context.Background(), "x"); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
func TestRequestContextCancellation(t *testing.T) {
	c := NewHTTPClient("https://hr.example")
	c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Context().Err() == nil {
			t.Fatal("expected canceled request context")
		}
		return nil, r.Context().Err()
	})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.GetAbsence(ctx, "x"); err == nil {
		t.Fatal("expected context error")
	}
}

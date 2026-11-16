package httptransport

import (
	"aibuddy/internal/auth"
	"aibuddy/internal/domain"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type authFake struct {
	id    domain.Identity
	err   error
	token string
}

func (f *authFake) Authenticate(_ context.Context, t string) (domain.Identity, error) {
	f.token = t
	return f.id, f.err
}

type pingFake struct{ err error }

func (f pingFake) PingContext(context.Context) error { return f.err }
func TestRequireAuth(t *testing.T) {
	id := domain.Identity{UserID: uuid.New(), Role: domain.RoleIntern, Team: "be"}
	cases := []struct {
		name, header string
		authErr      error
		status       int
		wantToken    string
	}{{"missing", "", nil, 401, ""}, {"bad scheme", "Basic abc", nil, 401, ""}, {"valid", "Bearer token", nil, 200, "token"}, {"unauthorized", "Bearer bad", domain.ErrUnauthorized, 401, "bad"}, {"backend failure", "Bearer token", errors.New("db down"), 503, "token"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &authFake{id: id, err: tc.authErr}
			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			h := RequireAuth(a, log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got, ok := auth.IdentityFrom(r.Context())
				if !ok || got != id {
					t.Error("identity not injected")
				}
				w.WriteHeader(200)
			}))
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Authorization", tc.header)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status || a.token != tc.wantToken {
				t.Fatalf("status=%d token=%q", w.Code, a.token)
			}
		})
	}
}
func TestHealthReadyAndMe(t *testing.T) {
	w := httptest.NewRecorder()
	Healthz(w, httptest.NewRequest("GET", "/healthz", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"ok"`) {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	for _, tc := range []struct {
		err    error
		status int
	}{{nil, 200}, {errors.New("down"), 503}} {
		w = httptest.NewRecorder()
		Readyz(pingFake{tc.err}, slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(w, httptest.NewRequest("GET", "/readyz", nil))
		if w.Code != tc.status {
			t.Fatalf("ready status %d", w.Code)
		}
	}
	id := domain.Identity{UserID: uuid.New(), Role: domain.RoleMentor, Team: "team"}
	req := httptest.NewRequest("GET", "/me", nil).WithContext(auth.WithIdentity(context.Background(), id))
	w = httptest.NewRecorder()
	Me(w, req)
	var got meResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got.UserID != id.UserID || got.Team != id.Team {
		t.Fatalf("%#v err=%v", got, err)
	}
}
func TestRecoverAndLogging(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }), Recover(log))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 500 || !strings.Contains(w.Body.String(), "internal error") {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	h = Logging(log)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(202); w.WriteHeader(500) }))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	if w.Code != 202 {
		t.Fatalf("status=%d", w.Code)
	}
}

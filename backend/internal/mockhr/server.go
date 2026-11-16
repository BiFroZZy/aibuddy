package mockhr

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"aibuddy/internal/domain"

	"github.com/google/uuid"
)

type Store interface {
	Create(ctx context.Context, userID uuid.UUID, kind string, payload json.RawMessage, approverRole, idempotencyKey string) (uuid.UUID, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Request, error)
	ListForUser(ctx context.Context, userID uuid.UUID, kind string, limit int) ([]domain.Request, error)
	PendingForManager(ctx context.Context, managerID uuid.UUID) ([]domain.Request, error)
	Decide(ctx context.Context, managerID, id uuid.UUID, approve bool, comment string) (domain.Request, error)
}

type server struct {
	store Store
	log   *slog.Logger
}

func NewHandler(store Store, log *slog.Logger) http.Handler {
	s := &server{store: store, log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /requests", s.create)
	mux.HandleFunc("GET /requests", s.list)
	mux.HandleFunc("GET /requests/{id}", s.get)
	mux.HandleFunc("POST /requests/{id}/decision", s.decide)
	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *server) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	default:
		s.log.Error("mock-hr request failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

func bad(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		bad(w, "invalid JSON body")
		return false
	}
	return true
}

func (s *server) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		UserID       uuid.UUID       `json:"user_id"`
		Kind         string          `json:"kind"`
		Payload      json.RawMessage `json:"payload"`
		ApproverRole string          `json:"approver_role"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.UserID == uuid.Nil || (in.Kind != domain.KindAbsence && in.Kind != domain.KindAccess) || !json.Valid(in.Payload) || in.ApproverRole == "" {
		bad(w, "user_id, kind (absence|access), payload and approver_role are required")
		return
	}
	id, err := s.store.Create(r.Context(), in.UserID, in.Kind, in.Payload, in.ApproverRole, r.Header.Get("Idempotency-Key"))
	if err != nil {
		s.fail(w, err)
		return
	}
	req, err := s.store.Get(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (s *server) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		bad(w, "invalid id")
		return
	}
	req, err := s.store.Get(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (s *server) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if v := q.Get("manager_id"); v != "" {
		mid, err := uuid.Parse(v)
		if err != nil {
			bad(w, "invalid manager_id")
			return
		}
		list, err := s.store.PendingForManager(r.Context(), mid)
		if err != nil {
			s.fail(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"requests": list})
		return
	}
	uid, err := uuid.Parse(q.Get("user_id"))
	if err != nil {
		bad(w, "user_id or manager_id is required")
		return
	}
	limit := 50
	if v := q.Get("limit"); v != "" {
		if limit, err = strconv.Atoi(v); err != nil || limit < 1 || limit > 200 {
			bad(w, "limit must be between 1 and 200")
			return
		}
	}
	list, err := s.store.ListForUser(r.Context(), uid, q.Get("kind"), limit)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": list})
}

func (s *server) decide(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		bad(w, "invalid id")
		return
	}
	var in struct {
		ManagerID uuid.UUID `json:"manager_id"`
		Approve   bool      `json:"approve"`
		Comment   string    `json:"comment"`
	}
	if !decode(w, r, &in) {
		return
	}
	req, err := s.store.Decide(r.Context(), in.ManagerID, id, in.Approve, in.Comment)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

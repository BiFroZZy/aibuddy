package httptransport

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"aibuddy/internal/domain"

	"github.com/google/uuid"
)

const maxBodyBytes = 64 << 10

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type errorBody struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorBody{Error: msg})
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func pathUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+name)
		return uuid.Nil, false
	}
	return id, true
}

func fail(w http.ResponseWriter, log *slog.Logger, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidArgs):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		unauthorized(w)
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrUnavailable):
		log.Warn("corporate service unavailable", "err", err)
		writeError(w, http.StatusBadGateway, err.Error())
	case errors.Is(err, domain.ErrLLMUnavailable):
		log.Warn("llm unavailable", "err", err)
		writeError(w, http.StatusBadGateway, "the assistant is temporarily unavailable, please try again")
	default:
		log.Error("request failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

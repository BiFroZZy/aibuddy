package httptransport

import (
	"net/http"
	"strconv"
	"time"

	"aibuddy/internal/auth"
	"aibuddy/internal/chat"
	"aibuddy/internal/domain"

	"github.com/google/uuid"
)

type handlers struct{ d Deps }

func (h *handlers) identity(w http.ResponseWriter, r *http.Request) (domain.Identity, bool) {
	id, ok := auth.IdentityFrom(r.Context())
	if !ok {
		unauthorized(w)
	}
	return id, ok
}

func (h *handlers) chat(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(w, r)
	if !ok {
		return
	}
	var body struct {
		ConversationID string `json:"conversation_id"`
		Message        string `json:"message"`
		RequestID      string `json:"request_id"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	in := chat.Input{Message: body.Message, RequestID: body.RequestID}
	if body.ConversationID != "" {
		cid, err := uuid.Parse(body.ConversationID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid conversation_id")
			return
		}
		in.ConversationID = cid
	}
	out, err := h.d.Chat.Handle(r.Context(), id, in)
	if err != nil {
		fail(w, h.d.Log, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) history(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(w, r)
	if !ok {
		return
	}
	cid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	msgs, err := h.d.Chat.History(r.Context(), id.UserID, cid, 100)
	if err != nil {
		fail(w, h.d.Log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": msgs})
}

func (h *handlers) progress(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(w, r)
	if !ok {
		return
	}
	p, err := h.d.Progress.Get(r.Context(), id.UserID)
	if err != nil {
		fail(w, h.d.Log, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *handlers) setStepStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(w, r)
	if !ok {
		return
	}
	stepID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	p, err := h.d.Progress.SetStatus(r.Context(), id.UserID, stepID, domain.StepStatus(body.Status))
	if err != nil {
		fail(w, h.d.Log, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *handlers) listActions(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(w, r)
	if !ok {
		return
	}
	list, err := h.d.Actions.ListActions(r.Context(), id.UserID, 50)
	if err != nil {
		fail(w, h.d.Log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"actions": list})
}

func (h *handlers) confirmAction(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(w, r)
	if !ok {
		return
	}
	aid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	a, err := h.d.Actions.Confirm(r.Context(), id, aid)
	if err != nil {
		fail(w, h.d.Log, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *handlers) cancelAction(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(w, r)
	if !ok {
		return
	}
	aid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	a, err := h.d.Actions.Cancel(r.Context(), id, aid)
	if err != nil {
		fail(w, h.d.Log, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *handlers) myRequests(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(w, r)
	if !ok {
		return
	}
	list, err := h.d.Requests.ListForUser(r.Context(), id.UserID, "", 50)
	if err != nil {
		fail(w, h.d.Log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": list})
}

func (h *handlers) confirmations(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(w, r)
	if !ok {
		return
	}
	list, err := h.d.Progress.Confirmations(r.Context(), id.UserID)
	if err != nil {
		fail(w, h.d.Log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"confirmations": list})
}

type decisionBody struct {
	Decision string `json:"decision"`
	Comment  string `json:"comment"`
}

func (b decisionBody) approve() (bool, error) {
	switch b.Decision {
	case "approve":
		return true, nil
	case "reject":
		return false, nil
	}
	return false, domain.ErrInvalidArgs
}

func (h *handlers) decideStep(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(w, r)
	if !ok {
		return
	}
	internID, ok := pathUUID(w, r, "internID")
	if !ok {
		return
	}
	stepID, ok := pathUUID(w, r, "stepID")
	if !ok {
		return
	}
	var body decisionBody
	if !decodeBody(w, r, &body) {
		return
	}
	approve, err := body.approve()
	if err != nil {
		writeError(w, http.StatusBadRequest, `decision must be "approve" or "reject"`)
		return
	}
	if err := h.d.Progress.Decide(r.Context(), id.UserID, internID, stepID, approve); err != nil {
		fail(w, h.d.Log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) pendingRequests(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(w, r)
	if !ok {
		return
	}
	list, err := h.d.Requests.PendingForManager(r.Context(), id.UserID)
	if err != nil {
		fail(w, h.d.Log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": list})
}

func (h *handlers) decideRequest(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(w, r)
	if !ok {
		return
	}
	rid, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var body decisionBody
	if !decodeBody(w, r, &body) {
		return
	}
	approve, err := body.approve()
	if err != nil {
		writeError(w, http.StatusBadRequest, `decision must be "approve" or "reject"`)
		return
	}
	req, err := h.d.Requests.Decide(r.Context(), id.UserID, rid, approve, body.Comment)
	if err != nil {
		fail(w, h.d.Log, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (h *handlers) metrics(w http.ResponseWriter, r *http.Request) {
	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 365 {
			writeError(w, http.StatusBadRequest, "days must be between 1 and 365")
			return
		}
		days = n
	}
	m, err := h.d.Metrics.Metrics(r.Context(), h.d.Clock().Add(-time.Duration(days)*24*time.Hour))
	if err != nil {
		fail(w, h.d.Log, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

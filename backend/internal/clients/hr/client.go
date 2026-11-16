package hr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"aibuddy/internal/domain"

	"github.com/google/uuid"
)

type HTTPClient struct {
	BaseURL string
	HTTP    *http.Client
}

func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{BaseURL: baseURL, HTTP: &http.Client{Timeout: timeout}}
}

type createBody struct {
	UserID       uuid.UUID       `json:"user_id"`
	Kind         string          `json:"kind"`
	Payload      json.RawMessage `json:"payload"`
	ApproverRole string          `json:"approver_role"`
}

func (c *HTTPClient) Create(ctx context.Context, userID uuid.UUID, kind string, payload json.RawMessage, approverRole, idempotencyKey string) (uuid.UUID, error) {
	var out domain.Request
	err := c.do(ctx, http.MethodPost, "/requests", idempotencyKey,
		createBody{UserID: userID, Kind: kind, Payload: payload, ApproverRole: approverRole}, &out)
	return out.ID, err
}

func (c *HTTPClient) ListForUser(ctx context.Context, userID uuid.UUID, kind string, limit int) ([]domain.Request, error) {
	q := url.Values{"user_id": {userID.String()}, "kind": {kind}, "limit": {strconv.Itoa(limit)}}
	var out struct {
		Requests []domain.Request `json:"requests"`
	}
	err := c.do(ctx, http.MethodGet, "/requests?"+q.Encode(), "", nil, &out)
	return out.Requests, err
}

func (c *HTTPClient) PendingForManager(ctx context.Context, managerID uuid.UUID) ([]domain.Request, error) {
	q := url.Values{"manager_id": {managerID.String()}, "status": {domain.RequestPending}}
	var out struct {
		Requests []domain.Request `json:"requests"`
	}
	err := c.do(ctx, http.MethodGet, "/requests?"+q.Encode(), "", nil, &out)
	return out.Requests, err
}

type decisionBody struct {
	ManagerID uuid.UUID `json:"manager_id"`
	Approve   bool      `json:"approve"`
	Comment   string    `json:"comment"`
}

func (c *HTTPClient) Decide(ctx context.Context, managerID, id uuid.UUID, approve bool, comment string) (domain.Request, error) {
	var out domain.Request
	err := c.do(ctx, http.MethodPost, "/requests/"+id.String()+"/decision", "",
		decisionBody{ManagerID: managerID, Approve: approve, Comment: comment}, &out)
	return out, err
}

func (c *HTTPClient) do(ctx context.Context, method, path, idemKey string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return domain.ErrUnavailable
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return domain.ErrNotFound
	case resp.StatusCode == http.StatusBadRequest:
		return fmt.Errorf("%w: rejected by the corporate service", domain.ErrInvalidArgs)
	case resp.StatusCode >= 300:
		return domain.ErrUnavailable
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out); err != nil {
		return errors.Join(domain.ErrUnavailable, err)
	}
	return nil
}

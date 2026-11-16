package hr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type CreateAbsenceReq struct {
	UserID   uuid.UUID `json:"user_id"`
	Type     string    `json:"type"`
	DateFrom string    `json:"date_from"`
	DateTo   string    `json:"date_to"`
	Comment  string    `json:"comment,omitempty"`
}

type Absence struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Type      string    `json:"type"`
	DateFrom  string    `json:"date_from"`
	DateTo    string    `json:"date_to"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Client interface {
	CreateAbsence(ctx context.Context, req CreateAbsenceReq, idemKey string) (Absence, error)
	GetAbsence(ctx context.Context, id string) (Absence, error)
}

type HTTPClient struct {
	BaseURL string
	HTTP    *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{BaseURL: baseURL, HTTP: &http.Client{Timeout: 5 * time.Second}}
}

func (c *HTTPClient) CreateAbsence(ctx context.Context, req CreateAbsenceReq, idemKey string) (Absence, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return Absence{}, err
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/absences", bytes.NewReader(body))
	if err != nil {
		return Absence{}, err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", idemKey)
	return c.do(r)
}

func (c *HTTPClient) GetAbsence(ctx context.Context, id string) (Absence, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/absences/"+id, nil)
	if err != nil {
		return Absence{}, err
	}
	return c.do(r)
}

func (c *HTTPClient) do(r *http.Request) (Absence, error) {
	resp, err := c.HTTP.Do(r)
	if err != nil {
		return Absence{}, fmt.Errorf("hr service unavailable: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		return Absence{}, fmt.Errorf("hr service: unexpected status %d", resp.StatusCode)
	}
	var a Absence
	if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
		return Absence{}, err
	}
	return a, nil
}

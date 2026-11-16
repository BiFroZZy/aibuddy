package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"aibuddy/internal/domain"
)

func newClient(url string) *OpenAI {
	c := NewOpenAI(url+"/", "test-model", "secret", 2*time.Second)
	c.RetryDelay = time.Millisecond
	return c
}

func TestOpenAI_TextReplyAndRequestShape(t *testing.T) {
	var got oaRequest
	var auth, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"привет"}}]}`))
	}))
	defer srv.Close()

	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "q"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Name: "search_kb", Arguments: `{"query":"x"}`}}},
		{Role: RoleTool, ToolCallID: "c1", Name: "search_kb", Content: `{"ok":true}`},
	}
	tools := []ToolSpec{{Name: "search_kb", Description: "d", Parameters: json.RawMessage(`{"type":"object"}`)}}
	out, err := newClient(srv.URL).Chat(context.Background(), msgs, tools)
	if err != nil || out.Content != "привет" || out.Role != RoleAssistant {
		t.Fatalf("reply: %+v %v", out, err)
	}
	if path != "/chat/completions" || auth != "Bearer secret" || got.Model != "test-model" {
		t.Fatalf("request: path=%s auth=%s model=%s", path, auth, got.Model)
	}
	if len(got.Messages) != 4 || got.Messages[2].ToolCalls[0].Function.Name != "search_kb" || got.Messages[3].ToolCallID != "c1" {
		t.Fatalf("messages were not mapped correctly: %+v", got.Messages)
	}
	if len(got.Tools) != 1 || got.Tools[0].Type != "function" || got.Tools[0].Function.Name != "search_kb" {
		t.Fatalf("tools: %+v", got.Tools)
	}
}

func TestOpenAI_ToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":null,"tool_calls":[
			{"id":"c9","type":"function","function":{"name":"get_my_progress","arguments":"{}"}}]}}]}`))
	}))
	defer srv.Close()
	out, err := newClient(srv.URL).Chat(context.Background(), []Message{{Role: RoleUser, Content: "x"}}, nil)
	if err != nil || len(out.ToolCalls) != 1 || out.ToolCalls[0].ID != "c9" || out.ToolCalls[0].Name != "get_my_progress" {
		t.Fatalf("tool calls: %+v %v", out, err)
	}
}

func TestOpenAI_RetriesServerErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()
	out, err := newClient(srv.URL).Chat(context.Background(), nil, nil)
	if err != nil || out.Content != "ok" || calls.Load() != 2 {
		t.Fatalf("retry: %+v %v calls=%d", out, err, calls.Load())
	}
}

func TestOpenAI_Failures(t *testing.T) {
	tests := []struct {
		name      string
		handler   http.HandlerFunc
		wantCalls int32
	}{
		{"client error is not retried", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }, 1},
		{"persistent 500 gives up", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }, 2},
		{"rate limit is retried", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTooManyRequests) }, 2},
		{"garbage body", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`not json`)) }, 1},
		{"no choices", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"choices":[]}`)) }, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				tc.handler(w, r)
			}))
			defer srv.Close()
			_, err := newClient(srv.URL).Chat(context.Background(), nil, nil)
			if !errors.Is(err, domain.ErrLLMUnavailable) || calls.Load() != tc.wantCalls {
				t.Fatalf("err=%v calls=%d want %d", err, calls.Load(), tc.wantCalls)
			}
		})
	}
}

func TestOpenAI_NetworkErrorAndCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()
	if _, err := newClient(url).Chat(context.Background(), nil, nil); !errors.Is(err, domain.ErrLLMUnavailable) {
		t.Fatalf("network error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := newClient(url)
	c.RetryDelay = time.Hour
	if _, err := c.Chat(ctx, nil, nil); !errors.Is(err, domain.ErrLLMUnavailable) {
		t.Fatalf("cancelled context: %v", err)
	}
}

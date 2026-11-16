package ml_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aibuddy/internal/clients/ml"
	"aibuddy/internal/domain"
)

var ctx = context.Background()

func sampleRequest() ml.ChatRequest {
	return ml.ChatRequest{
		User:    ml.UserContext{Role: "intern", Team: "Backend", Stage: "Доступы"},
		History: []ml.Message{{Role: "user", Content: "привет"}},
		Tools:   []ml.ToolSpec{{Name: "request_access", Description: "d", Schema: json.RawMessage(`{"type":"object"}`)}},
		KB:      []ml.KBChunk{{Source: "kb/a.md", Content: "текст"}},
	}
}

func TestChat(t *testing.T) {
	var got ml.ChatRequest
	var path, ct, accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, ct, accept = r.URL.Path, r.Header.Get("Content-Type"), r.Header.Get("Accept")
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"text":"ответ","tool_call":{"name":"request_access","args":{"project":"x"}}}`))
	}))
	defer srv.Close()

	c := ml.NewHTTPClient(srv.URL+"/", 2*time.Second)
	out, err := c.Chat(ctx, sampleRequest())
	if err != nil || out.Text != "ответ" || out.ToolCall == nil || out.ToolCall.Name != "request_access" || string(out.ToolCall.Args) != `{"project":"x"}` {
		t.Fatalf("response: %+v %v", out, err)
	}
	if path != "/chat" || ct != "application/json" || accept != "application/json" {
		t.Fatalf("request: %s %s %s", path, ct, accept)
	}
	if got.User.Stage != "Доступы" || got.KB[0].Source != "kb/a.md" || got.Tools[0].Name != "request_access" || len(got.History) != 1 {
		t.Fatalf("the request body must carry the whole contract: %+v", got)
	}

	c.ChatPath = "/v2/generate"
	_, _ = c.Chat(ctx, sampleRequest())
	if path != "/v2/generate" {
		t.Fatalf("custom path: %s", path)
	}
}

func TestChatFailures(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"500":      func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) },
		"404":      func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(404) },
		"bad json": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) },
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(h)
			defer srv.Close()
			_, err := ml.NewHTTPClient(srv.URL, time.Second).Chat(ctx, sampleRequest())
			if !errors.Is(err, domain.ErrLLMUnavailable) {
				t.Fatalf("got %v", err)
			}
		})
	}

	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()
	if _, err := ml.NewHTTPClient(url, time.Second).Chat(ctx, sampleRequest()); !errors.Is(err, domain.ErrLLMUnavailable) {
		t.Fatalf("unreachable: %v", err)
	}
	if _, err := ml.NewHTTPClient("http://bad host", time.Second).Chat(ctx, sampleRequest()); err == nil {
		t.Fatal("an invalid URL must fail")
	}
}

func sse(lines ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, l := range lines {
			_, _ = w.Write([]byte(l + "\n"))
		}
	}
}

func TestStream(t *testing.T) {
	var accept, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept, path = r.Header.Get("Accept"), r.URL.Path
		sse(": keep-alive", "event: token", `data: {"token":"При"}`, "", `data: {"token":"вет"}`, "",
			`data: {"done":true,"text":"Привет","tool_call":{"name":"get_my_progress","args":{}}}`, "")(w, r)
	}))
	defer srv.Close()

	var tokens []string
	out, err := ml.NewHTTPClient(srv.URL, time.Second).Stream(ctx, sampleRequest(), func(s string) { tokens = append(tokens, s) })
	if err != nil || out.Text != "Привет" || out.ToolCall == nil || out.ToolCall.Name != "get_my_progress" {
		t.Fatalf("stream result: %+v %v", out, err)
	}
	if strings.Join(tokens, "") != "Привет" || accept != "text/event-stream" || path != "/chat/stream" {
		t.Fatalf("tokens %v accept %s path %s", tokens, accept, path)
	}
	if _, err := ml.NewHTTPClient(srv.URL, time.Second).Stream(ctx, sampleRequest(), nil); err != nil {
		t.Fatal(err)
	}
}

func TestStreamFailures(t *testing.T) {
	tests := map[string]http.HandlerFunc{
		"no final event":  sse(`data: {"token":"a"}`, ""),
		"malformed event": sse(`data: not-json`, ""),
		"http error":      func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(503) },
	}
	for name, h := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(h)
			defer srv.Close()
			if _, err := ml.NewHTTPClient(srv.URL, time.Second).Stream(ctx, sampleRequest(), nil); !errors.Is(err, domain.ErrLLMUnavailable) {
				t.Fatalf("got %v", err)
			}
		})
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "1000")
		_, _ = w.Write([]byte("data: {\"token\":\"a\"}\n"))
	}))
	defer srv.Close()
	if _, err := ml.NewHTTPClient(srv.URL, time.Second).Stream(ctx, sampleRequest(), nil); !errors.Is(err, domain.ErrLLMUnavailable) {
		t.Fatalf("interrupted stream: %v", err)
	}
	if _, err := ml.NewHTTPClient("http://bad host", time.Second).Stream(ctx, sampleRequest(), nil); err == nil {
		t.Fatal("an invalid URL must fail")
	}
}

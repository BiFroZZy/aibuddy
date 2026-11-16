package config

import (
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestLoad_Defaults(t *testing.T) {
	c, err := Load(env(map[string]string{"DATABASE_URL": "postgres://x"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.HTTPAddr != ":8080" || !c.AutoMigrate || c.LLM.Mode != LLMModeFake || c.LLM.Timeout != 60*time.Second {
		t.Fatalf("unexpected defaults: %+v", c)
	}
}

func TestLoad_LLMModes(t *testing.T) {
	base := map[string]string{"DATABASE_URL": "x"}
	with := func(kv ...string) map[string]string {
		m := map[string]string{}
		for k, v := range base {
			m[k] = v
		}
		for i := 0; i < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		return m
	}
	tests := []struct {
		name    string
		env     map[string]string
		mode    string
		wantErr string
	}{
		{"key switches to openai", with("LLM_API_KEY", "k", "LLM_MODEL", "m"), LLMModeOpenAI, ""},
		{"explicit fake wins over a key", with("LLM_MODE", "FAKE", "LLM_API_KEY", "k"), LLMModeFake, ""},
		{"openai needs key and model", with("LLM_MODE", "openai"), "", "LLM_API_KEY"},
		{"openai needs model", with("LLM_MODE", "openai", "LLM_API_KEY", "k"), "", "LLM_MODEL"},
		{"ml mode with url", with("LLM_MODE", "ml", "ML_URL", "http://ml:8000/"), LLMModeML, ""},
		{"ml mode needs url", with("LLM_MODE", "ml"), "", "ML_URL"},
		{"unknown mode", with("LLM_MODE", "gpt"), "", "unknown mode"},
		{"bad timeout", with("LLM_TIMEOUT", "soon"), "", "LLM_TIMEOUT"},
		{"non-positive timeout", with("LLM_TIMEOUT", "0s"), "", "LLM_TIMEOUT"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, err := Load(env(tc.env))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil || c.LLM.Mode != tc.mode {
				t.Fatalf("mode %q, err %v", c.LLM.Mode, err)
			}
		})
	}
}

func TestLoad_BaseURLTrimmedAndTimeout(t *testing.T) {
	c, err := Load(env(map[string]string{
		"DATABASE_URL": "x", "LLM_BASE_URL": "http://localhost:11434/v1/", "LLM_TIMEOUT": "5s",
		"LOG_LEVEL": "debug", "HR_URL": " http://mock-hr:8081/ ", "AUTO_MIGRATE": "false", "CORS_ALLOWED_ORIGIN": " http://localhost:5173 ",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if c.LLM.BaseURL != "http://localhost:11434/v1" || c.LLM.Timeout != 5*time.Second || c.AutoMigrate || c.CORSOrigin != "http://localhost:5173" || c.HRURL != "http://mock-hr:8081" || c.LogLevel.String() != "DEBUG" {
		t.Fatalf("unexpected config: %+v", c)
	}
}

func TestLoad_Errors(t *testing.T) {
	_, err := Load(env(map[string]string{"LOG_LEVEL": "loud", "AUTO_MIGRATE": "maybe"}))
	if err == nil {
		t.Fatal("expected errors")
	}
	for _, want := range []string{"DATABASE_URL", "LOG_LEVEL", "AUTO_MIGRATE"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not mention %s", err, want)
		}
	}
}

func TestLoad_MLURLIsTrimmed(t *testing.T) {
	c, err := Load(env(map[string]string{"DATABASE_URL": "x", "LLM_MODE": "ml", "ML_URL": " http://ml:8000/ "}))
	if err != nil || c.LLM.MLURL != "http://ml:8000" {
		t.Fatalf("%+v %v", c.LLM, err)
	}
}

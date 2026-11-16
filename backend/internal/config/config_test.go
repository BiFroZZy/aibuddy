package config

import (
	"log/slog"
	"testing"
)

func TestLoadDefaultsAndEnvironment(t *testing.T) {
	env := map[string]string{"POSTGRES_USER": "app", "POSTGRES_PASSWORD": "secret", "POSTGRES_PORT": "5432", "POSTGRES_DB": "demo"}
	get := func(k string) string { return env[k] }
	c, err := Load(get)
	if err != nil {
		t.Fatal(err)
	}
	if c.HTTPAddr != ":8080" || c.HRURL != "http://mock-hr:8081" || c.MLURL != "http://ml:8000" {
		t.Fatalf("defaults: %#v", c)
	}
	env["HTTP_ADDR"] = " 127.0.0.1:9090 "
	env["HR_URL"] = " http://hr "
	env["ML_URL"] = "http://ml.local"
	env["LOG_LEVEL"] = "debug"
	c, err = Load(get)
	if err != nil {
		t.Fatal(err)
	}
	if c.HTTPAddr != "127.0.0.1:9090" || c.HRURL != "http://hr" || c.MLURL != "http://ml.local" || c.LogLevel != slog.LevelDebug {
		t.Fatalf("overrides: %#v", c)
	}
}
func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	env := map[string]string{"POSTGRES_USER": "u", "POSTGRES_PASSWORD": "p", "POSTGRES_PORT": "1", "POSTGRES_DB": "d", "LOG_LEVEL": "invalid"}
	if _, err := Load(func(k string) string { return env[k] }); err == nil {
		t.Fatal("expected invalid log level error")
	}
}
func TestOrDefaultTrims(t *testing.T) {
	if got := orDefault("  ", "fallback"); got != "fallback" {
		t.Fatal(got)
	}
	if got := orDefault(" value ", "fallback"); got != "value" {
		t.Fatal(got)
	}
}

package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

const (
	LLMModeFake   = "fake"
	LLMModeOpenAI = "openai"
	LLMModeML     = "ml"
)

type LLMConfig struct {
	Mode    string
	BaseURL string
	Model   string
	APIKey  string
	Timeout time.Duration
	MLURL   string
}

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	LogLevel    slog.Level
	AutoMigrate bool
	CORSOrigin  string
	HRURL       string // mock-hr (корпоративный сервис отсутсвий)
	LLM         LLMConfig
}

func Load(getenv func(string) string) (Config, error) {
	var DB_URL = "postgres://" + getenv("POSTGRES_USER") + ":" + getenv("POSTGRES_PASSWORD") + "@" + "localhost:" + getenv("POSTGRES_PORT") + "/" + getenv("POSTGRES_DB") + "?sslmode=disable"
	c := Config{
		HTTPAddr:    orDefault(getenv("HTTP_ADDR"), ":8080"),
		DatabaseURL: strings.TrimSpace(DB_URL),
		CORSOrigin: strings.TrimSpace(getenv("CORS_ALLOWED_ORIGIN")),
		HRURL:       strings.TrimRight(strings.TrimSpace(getenv("HR_URL")), "/"),
		AutoMigrate: true,
	}

	var errs []error
	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if lvl := strings.TrimSpace(getenv("LOG_LEVEL")); lvl != "" {
		if err := c.LogLevel.UnmarshalText([]byte(lvl)); err != nil {
			errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
		}
	}
	if v := strings.TrimSpace(getenv("AUTO_MIGRATE")); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("AUTO_MIGRATE: %w", err))
		}
		c.AutoMigrate = b
	}
	llmCfg, err := loadLLM(getenv)
	if err != nil {
		errs = append(errs, err)
	}
	c.LLM = llmCfg

	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return c, nil
}

func loadLLM(getenv func(string) string) (LLMConfig, error) {
	l := LLMConfig{
		Mode: strings.ToLower(strings.TrimSpace(getenv("LLM_MODE"))),
		BaseURL: strings.TrimRight(orDefault(getenv("LLM_BASE_URL"), "https://api.openai.com/v1"), "/"),
		Model: strings.TrimSpace(getenv("LLM_MODEL")),
		APIKey: strings.TrimSpace(getenv("LLM_API_KEY")),
		Timeout: 60 * time.Second,
		MLURL: strings.TrimRight(strings.TrimSpace(getenv("ML_URL")), "/"),
	}
	if v := strings.TrimSpace(getenv("LLM_TIMEOUT")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return LLMConfig{}, fmt.Errorf("LLM_TIMEOUT: invalid duration %q", v)
		}
		l.Timeout = d
	}
	if l.Mode == "" {
		l.Mode = LLMModeFake
		if l.APIKey != "" {
			l.Mode = LLMModeOpenAI
		}
	}
	switch l.Mode {
	case LLMModeFake:
	case LLMModeOpenAI:
		var errs []error
		if l.APIKey == "" {
			errs = append(errs, errors.New("LLM_API_KEY is required when LLM_MODE = openai"))
		}
		if l.Model == "" {
			errs = append(errs, errors.New("LLM_MODEL is required when LLM_MODE = openai"))
		}
		if err := errors.Join(errs...); err != nil {
			return LLMConfig{}, err
		}
	case LLMModeML:
		if l.MLURL == "" {
			return LLMConfig{}, errors.New("ML_URL is required when LLM_MODE = ml")
		}
	default:
		return LLMConfig{}, fmt.Errorf("LLM_MODE: unknown mode %q (use fake, openai or ml)", l.Mode)
	}
	return l, nil
}

func orDefault(v, def string) string {
	if v = strings.TrimSpace(v); v != "" {
		return v
	}
	return def
}

package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	HRURL       string // mock-hr (корпоративный сервис отсутсвий)
	MLURL       string
	LogLevel    slog.Level
}

func Load(getenv func(string) string) (Config, error) {
	var DB_URL = "postgres://"+getenv("POSTGRES_USER")+":"+getenv("POSTGRES_PASSWORD")+"@"+"localhost:"+getenv("POSTGRES_PORT")+"/"+getenv("POSTGRES_DB")+"?sslmode=disable"
	c := Config{
		HTTPAddr:    orDefault(getenv("HTTP_ADDR"), ":8080"),
		DatabaseURL: strings.TrimSpace(DB_URL),
		HRURL:       orDefault(getenv("HR_URL"), "http://mock-hr:8081"),
		MLURL:       orDefault(getenv("ML_URL"), "http://ml:8000"),
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

	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return c, nil
}

func orDefault(v, def string) string {
	if v = strings.TrimSpace(v); v != "" {
		return v
	}
	return def
}

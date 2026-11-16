package server

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aibuddy/internal/app"
	"aibuddy/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Server struct {
	cfg config.Config
	log *slog.Logger
}

func New() (*Server, error) {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return nil, err
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	return &Server{cfg: cfg, log: log}, nil
}

func (s *Server) Start() error {
	db, err := sql.Open("pgx", s.cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			s.log.Error("failed to close database", "error", err)
		}
	}()

	pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelPing()
	if err := db.PingContext(pingCtx); err != nil {
		s.log.Error("no response from database", "error", err)
		return err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if s.cfg.AutoMigrate {
		migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), time.Minute)
		defer cancelMigrate()
		if err := app.Migrate(migrateCtx, db, s.log); err != nil {
			s.log.Error("migrations failed", "error", err)
			return err
		}
	}

	handler := app.NewHandler(s.cfg, db, s.log, app.NewLLM(s.cfg.LLM, db, s.log))

	srv := &http.Server{
		Addr:              s.cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      s.cfg.LLM.Timeout*2 + 30*time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(s.log.Handler(), slog.LevelError),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		s.log.Info("api started", "addr", s.cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.log.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		s.log.Error("shutdown failed", "error", err)
		return err
	}
	return <-errCh
}

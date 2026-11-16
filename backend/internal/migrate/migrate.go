package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

const lockID = 727274

func Up(ctx context.Context, db *sql.DB, fsys fs.FS) ([]string, error) {
	files, err := fs.Glob(fsys, "*.up.sql")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(files)

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate: acquire connection: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID); err != nil {
		return nil, fmt.Errorf("migrate: lock: %w", err)
	}
	defer func() { _, _ = conn.ExecContext(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", lockID) }()

	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS app_schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return nil, fmt.Errorf("migrate: create app_schema_migrations: %w", err)
	}

	applied := map[string]bool{}
	rows, err := conn.QueryContext(ctx, "SELECT version FROM app_schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("migrate: read versions: %w", err)
	}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("migrate: scan version: %w", err)
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("migrate: iterate versions: %w", err)
	}
	_ = rows.Close()

	var done []string
	for _, f := range files {
		version := strings.TrimSuffix(f, ".up.sql")
		if applied[version] {
			continue
		}
		body, err := fs.ReadFile(fsys, f)
		if err != nil {
			return done, fmt.Errorf("migrate: read %s: %w", f, err)
		}
		if err := apply(ctx, conn, version, string(body)); err != nil {
			return done, err
		}
		done = append(done, version)
	}
	return done, nil
}

func apply(ctx context.Context, conn *sql.Conn, version, body string) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrate %s: begin: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx, body); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("migrate %s: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO app_schema_migrations (version) VALUES ($1)", version); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("migrate %s: record: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate %s: commit: %w", version, err)
	}
	return nil
}

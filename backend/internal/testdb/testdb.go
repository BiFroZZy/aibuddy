package testdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/url"
	"os"
	"testing"
	"time"

	"aibuddy/internal/migrate"
	"aibuddy/migrations"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const Driver = "pgx"

var (
	HRID      = uuid.MustParse("fb8a5a61-d41c-464d-b9ee-fe07b3745136")
	ManagerID = uuid.MustParse("c390da66-1700-4705-b5b3-d24c40914e2d")
	MentorID  = uuid.MustParse("abff9c90-827e-4e33-b9ec-8c1082a75f63")
	InternID  = uuid.MustParse("a2be6219-ca2c-4f3b-bcb1-1e6c07622624")
	Intern2ID = uuid.MustParse("28a82a11-925c-4783-a32e-426278bfec83")

	Team1ID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

	Step1 = uuid.MustParse("cccccccc-0000-0000-0000-000000000001") 
	Step2 = uuid.MustParse("cccccccc-0000-0000-0000-000000000002") 
	Step3 = uuid.MustParse("cccccccc-0000-0000-0000-000000000003") 
	Step7 = uuid.MustParse("cccccccc-0000-0000-0000-000000000007") 
)

const (
	InternToken  = "dev-intern-1"
	Intern2Token = "dev-intern-2"
	MentorToken  = "dev-mentor"
	ManagerToken = "dev-manager"
	HRToken      = "dev-hr"
)

func Empty(t testing.TB) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping database test")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}

	admin, err := sql.Open(Driver, dsn)
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var suffix [6]byte
	_, _ = rand.Read(suffix[:])
	name := "aibuddy_test_" + hex.EncodeToString(suffix[:])
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		_ = admin.Close()
		t.Fatalf("create test database: %v", err)
	}

	u.Path = "/" + name
	db, err := sql.Open(Driver, u.String())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		_, _ = admin.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		_ = admin.Close()
	})
	return db
}

func New(t testing.TB) *sql.DB {
	t.Helper()
	db := Empty(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := migrate.Up(ctx, db, migrations.FS); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}

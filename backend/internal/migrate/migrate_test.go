package migrate_test

import (
	"context"
	"testing"
	"testing/fstest"

	"aibuddy/internal/migrate"
	"aibuddy/internal/testdb"
)

func TestUp_AppliesOnceAndInOrder(t *testing.T) {
	db := testdb.Empty(t)
	fsys := fstest.MapFS{
		"0002_second.up.sql":  {Data: []byte(`INSERT INTO t VALUES (2);`)},
		"0001_first.up.sql":   {Data: []byte(`CREATE TABLE t (n INT); INSERT INTO t VALUES (1);`)},
		"0001_first.down.sql": {Data: []byte(`DROP TABLE t;`)},
	}
	ctx := context.Background()

	done, err := migrate.Up(ctx, db, fsys)
	if err != nil || len(done) != 2 || done[0] != "0001_first" || done[1] != "0002_second" {
		t.Fatalf("first run: %v %v", done, err)
	}
	done, err = migrate.Up(ctx, db, fsys)
	if err != nil || len(done) != 0 {
		t.Fatalf("second run must be a no-op: %v %v", done, err)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM t`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("rows: %d %v", n, err)
	}
}

func TestUp_FailedMigrationRollsBack(t *testing.T) {
	db := testdb.Empty(t)
	fsys := fstest.MapFS{
		"0001_ok.up.sql":  {Data: []byte(`CREATE TABLE ok (n INT);`)},
		"0002_bad.up.sql": {Data: []byte(`CREATE TABLE half (n INT); SELECT * FROM missing_table;`)},
	}
	done, err := migrate.Up(context.Background(), db, fsys)
	if err == nil || len(done) != 1 || done[0] != "0001_ok" {
		t.Fatalf("want failure after the first migration, got %v %v", done, err)
	}
	var exists bool
	_ = db.QueryRow(`SELECT to_regclass('half') IS NOT NULL`).Scan(&exists)
	if exists {
		t.Fatal("a failed migration must leave no partial changes")
	}
	var applied int
	_ = db.QueryRow(`SELECT count(*) FROM app_schema_migrations`).Scan(&applied)
	if applied != 1 {
		t.Fatalf("only the good migration is recorded, got %d", applied)
	}
}

func TestUp_ClosedDatabase(t *testing.T) {
	db := testdb.Empty(t)
	_ = db.Close()
	if _, err := migrate.Up(context.Background(), db, fstest.MapFS{}); err == nil {
		t.Fatal("expected an error for a closed database")
	}
}

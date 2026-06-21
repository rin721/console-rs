package migrator

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/rei0721/go-scaffold/pkg/database"
)

func TestRunnerUpStatusDown(t *testing.T) {
	dir := t.TempDir()
	migration := `-- +goose Up
CREATE TABLE migrator_smoke (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
-- +goose Down
DROP TABLE migrator_smoke;
`
	if err := os.WriteFile(filepath.Join(dir, "20260531000100_create_smoke.sql"), []byte(migration), 0644); err != nil {
		t.Fatalf("write migration: %v", err)
	}

	db, err := database.New(&database.Config{Driver: database.DriverSQLite, DBName: filepath.Join(t.TempDir(), "test.db")})
	if err != nil {
		t.Fatalf("database.New() failed: %v", err)
	}
	defer db.Close()

	runner, err := New(db, Config{Driver: string(database.DriverSQLite), Dir: dir})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if err := runner.Up(context.Background()); err != nil {
		t.Fatalf("Up() failed: %v", err)
	}
	if ok, err := db.HasTable(context.Background(), struct{}{}); err != nil || ok {
		t.Fatalf("anonymous HasTable sanity check = %v, %v", ok, err)
	}
	var count int
	if _, err := db.Raw(context.Background(), &count, "SELECT COUNT(*) FROM migrator_smoke"); err != nil {
		t.Fatalf("query migrated table: %v", err)
	}
	var status bytes.Buffer
	if err := runner.Status(context.Background(), &status); err != nil {
		t.Fatalf("Status() failed: %v", err)
	}
	if status.Len() == 0 {
		t.Fatal("expected status output")
	}
	if err := runner.Down(context.Background()); err != nil {
		t.Fatalf("Down() failed: %v", err)
	}
}

func TestNewRequiresSQLProvider(t *testing.T) {
	_, err := New(nil, Config{Driver: "sqlite", Dir: t.TempDir()})
	if err == nil {
		t.Fatal("expected nil provider error")
	}
}

var _ SQLProvider = sqlProviderFunc(nil)

type sqlProviderFunc func() (*sql.DB, error)

func (f sqlProviderFunc) SQLDB() (*sql.DB, error) {
	return f()
}

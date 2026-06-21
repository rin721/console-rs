package initservice

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rei0721/go-scaffold/pkg/database"
)

type bootstrapCasbinRule struct{}

func (bootstrapCasbinRule) TableName() string { return "iam_casbin_rules" }

func TestInspectInitializationStatusUsesBootstrapOnly(t *testing.T) {
	t.Parallel()

	configPath, dbPath := copyTempConfig(t)
	status, err := InspectInitializationStatus(context.Background(), configPath)
	if err != nil {
		t.Fatalf("InspectInitializationStatus() error = %v", err)
	}
	if !status.Required {
		t.Fatal("status.Required = false, want true for empty bootstrap database")
	}

	db, err := database.New(&database.Config{
		Driver: database.DriverSQLite,
		DBName: dbPath,
		Silent: true,
	})
	if err != nil {
		t.Fatalf("reopen sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	hasCasbinRules, err := db.HasTable(context.Background(), bootstrapCasbinRule{})
	if err != nil {
		t.Fatalf("HasTable(iam_casbin_rules) error = %v", err)
	}
	if hasCasbinRules {
		t.Fatal("InspectInitializationStatus created or touched iam_casbin_rules; bootstrap status must not load IAM policies")
	}
}

func copyTempConfig(t *testing.T) (string, string) {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, "configs", "config.example.yaml"))
	if err != nil {
		t.Fatalf("read config example: %v", err)
	}
	dir := t.TempDir()
	dbPath := filepath.ToSlash(filepath.Join(dir, "app.db"))
	content := strings.ReplaceAll(string(raw), "  dbname: ./data/app.db", "  dbname: \""+dbPath+"\"")
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path, dbPath
}

package initcenter

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rei0721/go-scaffold/internal/app/initapp"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/configloader"
)

func TestConfigStoreSaveForceWritesEnvManagedPlaceholder(t *testing.T) {
	configPath := copyExampleConfig(t)
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	raw = []byte(strings.Replace(string(raw), "productName: Aoi Admin", "productName: ${BRAND_PRODUCT_NAME:Aoi Admin}", 1))
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatalf("write config placeholder: %v", err)
	}
	manager := appconfig.NewManager()
	if err := manager.Load(configPath); err != nil {
		t.Fatalf("load config: %v", err)
	}

	store := NewInitConfigStore(initapp.Core{
		Config:        manager.Get(),
		ConfigManager: manager,
	}, configPath)
	saved, err := store.Save(context.Background(), "site.configure", map[string]any{
		"brand.productName":     "Custom Product",
		"brand.versionName":     "Custom Version",
		"webui.public_base_url": "https://admin.example.invalid",
	}, true)
	if err != nil {
		t.Fatalf("save site.configure: %v", err)
	}
	if saved.EnvManagedPersistence != envManagedPersistenceForceFile {
		t.Fatalf("EnvManagedPersistence = %q, want %q", saved.EnvManagedPersistence, envManagedPersistenceForceFile)
	}
	if !containsString(saved.EnvManagedPathsOverwritten, "brand.productName") {
		t.Fatalf("overwritten paths = %v, want brand.productName", saved.EnvManagedPathsOverwritten)
	}

	reloaded := appconfig.NewManager()
	if err := reloaded.Load(configPath); err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if got := reloaded.Get().Brand.ProductName; got != "Custom Product" {
		t.Fatalf("productName = %q, want Custom Product", got)
	}
	disabled, err := configloader.YAMLStringSlice(configPath, "env_override.disabled_paths")
	if err != nil {
		t.Fatalf("read disabled paths: %v", err)
	}
	if !containsString(disabled, "brand.productName") {
		t.Fatalf("disabled paths = %v, want brand.productName", disabled)
	}
	raw, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(raw), "productName: ${BRAND_PRODUCT_NAME") {
		t.Fatalf("brand productName still contains placeholder after force file save")
	}
}

func TestConfigTestDoesNotPersistEnvManagedPlaceholder(t *testing.T) {
	configPath := copyExampleConfig(t)
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	service := New(initapp.Core{Config: &appconfig.Config{}}, initapp.Infrastructure{}, initapp.Modules{}, configPath, nil)
	result, err := service.TestConfig(context.Background(), "site.configure", Input{Source: SourceCLI}, map[string]any{
		"brand.productName": "Custom Product",
		"brand.versionName": "Custom Version",
	})
	if err != nil {
		t.Fatalf("test config: %v", err)
	}
	if result.Status != "succeeded" {
		t.Fatalf("test status = %s error=%s", result.Status, result.Error)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config after test: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("TestConfig changed config file")
	}
}

func copyExampleConfig(t *testing.T) string {
	t.Helper()
	source := filepath.Join("..", "..", "..", "configs", "config.example.yaml")
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read example config: %v", err)
	}
	target := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(target, raw, 0600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return target
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

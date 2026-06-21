package initapp

import (
	"context"
	"errors"
	"testing"

	"github.com/rei0721/go-scaffold/internal/config"
	projectplugin "github.com/rei0721/go-scaffold/internal/plugin"
	pluginpkg "github.com/rei0721/go-scaffold/pkg/plugin"
)

func TestNewPluginsModuleDisabledKeepsHostButNoHandler(t *testing.T) {
	module, err := NewPluginsModule(Core{Config: &config.Config{Plugins: config.PluginsConfig{Enabled: false}}}, Infrastructure{}, IAMModule{})
	if err != nil {
		t.Fatalf("NewPluginsModule() error = %v", err)
	}
	if module.Host == nil {
		t.Fatal("plugin host is nil")
	}
	if module.Handler != nil {
		t.Fatal("disabled plugins module must not expose HTTP handler")
	}
	_, err = module.Host.ListPlugins(context.Background())
	if !errors.Is(err, pluginpkg.ErrDisabled) {
		t.Fatalf("ListPlugins() error = %v, want %v", err, pluginpkg.ErrDisabled)
	}
}

func TestNewPluginsModuleEnabledRegistersDisabledAPITokenProvider(t *testing.T) {
	module, err := NewPluginsModule(Core{Config: &config.Config{Plugins: config.PluginsConfig{Enabled: true, RegistryBackend: "memory"}}}, Infrastructure{}, IAMModule{})
	if err != nil {
		t.Fatalf("NewPluginsModule() error = %v", err)
	}
	if module.Host == nil {
		t.Fatal("plugin host is nil")
	}
	if module.Handler == nil {
		t.Fatal("enabled plugins module should expose management handler")
	}
	providers, err := module.Host.ListProviders(context.Background())
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}
	if len(providers) != 1 || providers[0].Name != projectplugin.CapabilityIAMAPITokenIssue {
		t.Fatalf("providers = %#v, want disabled API token provider", providers)
	}
}

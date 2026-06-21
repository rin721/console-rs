package plugin

import (
	"context"
	"errors"
	"testing"
	"time"

	pluginpkg "github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

func TestServiceListsRegisteredPlugins(t *testing.T) {
	host := pluginpkg.New(pluginpkg.Config{Enabled: true})
	_, err := host.Register(context.Background(), protocol.RegisterRequest{Plugin: protocol.PluginMetadata{
		PluginID:   "demo",
		InstanceID: "demo-1",
		Name:       "Demo",
		Version:    "0.1.0",
		Protocol:   protocol.TransportHTTP,
		Endpoint:   "http://127.0.0.1:10098",
	}})
	if err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	items, err := NewService(host).List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 || items[0].PluginID != "demo" {
		t.Fatalf("List() = %#v, want demo plugin", items)
	}
}

func TestServiceReportsDisabledWithoutHost(t *testing.T) {
	_, err := NewService(nil).List(context.Background())
	if !errors.Is(err, ErrDisabled) {
		t.Fatalf("List() error = %v, want %v", err, ErrDisabled)
	}
}

func TestServiceRejectsCapabilitiesForOfflinePlugin(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	host := pluginpkg.New(pluginpkg.Config{
		Enabled:          true,
		HeartbeatTimeout: time.Second,
		Now: func() time.Time {
			return now
		},
	})
	_, err := host.Register(context.Background(), protocol.RegisterRequest{Plugin: protocol.PluginMetadata{
		PluginID:   "demo",
		InstanceID: "demo-1",
		Name:       "Demo",
		Version:    "0.1.0",
		Protocol:   protocol.TransportHTTP,
		Endpoint:   "http://127.0.0.1:10098",
	}})
	if err != nil {
		t.Fatalf("register plugin: %v", err)
	}
	now = now.Add(2 * time.Second)

	_, err = NewService(host).Capabilities(context.Background(), "demo")
	if !errors.Is(err, ErrPluginOffline) {
		t.Fatalf("Capabilities() error = %v, want %v", err, ErrPluginOffline)
	}
}

func TestDisabledAPITokenProviderReturnsUnavailable(t *testing.T) {
	host := pluginpkg.New(pluginpkg.Config{Enabled: true})
	if err := host.RegisterProvider(DisabledAPITokenProvider()); err != nil {
		t.Fatalf("RegisterProvider() error = %v", err)
	}
	_, err := host.Invoke(context.Background(), protocol.InvokeRequest{Capability: CapabilityIAMAPITokenIssue})
	if !errors.Is(err, pluginpkg.ErrProviderUnavailable) {
		t.Fatalf("Invoke(api token) error = %v, want ErrProviderUnavailable", err)
	}
}

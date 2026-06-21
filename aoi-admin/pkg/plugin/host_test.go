package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/rei0721/go-scaffold/pkg/plugin/injection"
	"github.com/rei0721/go-scaffold/pkg/plugin/observability"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
	"github.com/rei0721/go-scaffold/pkg/plugin/security"
)

func TestHostRegistersHeartbeatAndExpiresPlugin(t *testing.T) {
	now := time.Date(2026, 6, 15, 1, 0, 0, 0, time.UTC)
	host := New(Config{
		Enabled:          true,
		HeartbeatTimeout: time.Minute,
		Now: func() time.Time {
			return now
		},
	})

	registered, err := host.Register(context.Background(), protocol.RegisterRequest{Plugin: protocol.PluginMetadata{
		PluginID:   "demo",
		InstanceID: "demo-1",
		Name:       "Demo",
		Version:    "0.1.0",
		Protocol:   protocol.TransportHTTP,
		Endpoint:   "http://127.0.0.1:10098",
		Capabilities: []protocol.Capability{
			{Name: "demo.echo"},
		},
	}})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if registered.Plugin.Status != protocol.StatusOnline {
		t.Fatalf("registered status = %q", registered.Plugin.Status)
	}

	now = now.Add(30 * time.Second)
	heartbeat, err := host.Heartbeat(context.Background(), protocol.HeartbeatRequest{PluginID: "demo", InstanceID: "demo-1"})
	if err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
	}
	if !heartbeat.Plugin.LastHeartbeatAt.Equal(now) {
		t.Fatalf("heartbeat time = %s, want %s", heartbeat.Plugin.LastHeartbeatAt, now)
	}

	now = now.Add(2 * time.Minute)
	plugin, err := host.GetPlugin(context.Background(), "demo")
	if err != nil {
		t.Fatalf("GetPlugin() error = %v", err)
	}
	if plugin.Status != protocol.StatusOffline {
		t.Fatalf("expired status = %q, want offline", plugin.Status)
	}
}

func TestCreatePluginAppNormalizesRemoteInstance(t *testing.T) {
	now := time.Date(2026, 6, 15, 1, 0, 0, 0, time.UTC)
	instance, err := CreatePluginApp(protocol.RegisterRequest{Plugin: protocol.PluginMetadata{
		PluginID:   " demo ",
		InstanceID: " demo-1 ",
		Name:       " Demo ",
		Version:    "0.1.0",
		Protocol:   protocol.TransportHTTP,
		Endpoint:   "http://127.0.0.1:10098",
		Capabilities: []protocol.Capability{
			{Name: " demo.echo ", Permissions: []string{" plugin:demo "}},
		},
	}}, InstanceContext{
		Now:                    now,
		DefaultProtocolVersion: protocol.ProtocolVersionV1,
		Source:                 "test-host",
	})
	if err != nil {
		t.Fatalf("CreatePluginApp() error = %v", err)
	}
	if instance.Snapshot.PluginID != "demo" || instance.Snapshot.InstanceID != "demo-1" || instance.Transport != protocol.TransportHTTP {
		t.Fatalf("instance snapshot = %#v", instance.Snapshot)
	}
	if instance.Snapshot.Protocol != protocol.PluginProtocolJSON || instance.Snapshot.Transport != protocol.TransportHTTP {
		t.Fatalf("protocol/transport = %q/%q", instance.Snapshot.Protocol, instance.Snapshot.Transport)
	}
	if instance.ProtocolVersion != protocol.ProtocolVersionV1 {
		t.Fatalf("protocol version = %q", instance.ProtocolVersion)
	}
	if got := instance.Snapshot.Capabilities[0].Name; got != "demo.echo" {
		t.Fatalf("capability name = %q", got)
	}
	if !instance.Audit.GeneratedAt.Equal(now) || instance.Audit.Source != "test-host" {
		t.Fatalf("audit = %#v", instance.Audit)
	}
}

func TestHostUnregisterRemovesPlugin(t *testing.T) {
	host := New(Config{Enabled: true})
	_, err := host.Register(context.Background(), protocol.RegisterRequest{Plugin: protocol.PluginMetadata{
		PluginID:   "demo",
		InstanceID: "demo-1",
		Name:       "Demo",
		Version:    "0.1.0",
		Protocol:   protocol.TransportHTTP,
		Endpoint:   "http://127.0.0.1:10098",
	}})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if _, err := host.Unregister(context.Background(), protocol.UnregisterRequest{PluginID: "demo", InstanceID: "demo-1"}); err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}
	if response, err := host.Unregister(context.Background(), protocol.UnregisterRequest{PluginID: "demo", InstanceID: "demo-1"}); err != nil || response.Status != protocol.StatusOffline {
		t.Fatalf("duplicate Unregister() response = %#v, error = %v", response, err)
	}
	plugin, err := host.GetPlugin(context.Background(), "demo")
	if err != nil {
		t.Fatalf("GetPlugin() error = %v", err)
	}
	if plugin.Status != protocol.StatusOffline {
		t.Fatalf("GetPlugin() status = %q, want offline", plugin.Status)
	}
	if _, err := host.Invoke(context.Background(), protocol.InvokeRequest{
		PluginID:   "demo",
		InstanceID: "demo-1",
		Capability: "demo.echo",
	}); !errors.Is(err, ErrPluginOffline) {
		t.Fatalf("Invoke() error = %v, want ErrPluginOffline", err)
	}
}

func TestHostProvidersInvokeAndRejectUnknownCapability(t *testing.T) {
	host := New(Config{Enabled: true})
	if err := host.RegisterProvider(ProviderFunc{
		Definition: protocol.Capability{Name: "demo.echo"},
		Handler: func(_ context.Context, req protocol.InvokeRequest) (json.RawMessage, error) {
			return append(json.RawMessage(nil), req.Payload...), nil
		},
	}); err != nil {
		t.Fatalf("RegisterProvider() error = %v", err)
	}

	response, err := host.Invoke(context.Background(), protocol.InvokeRequest{
		Capability: "demo.echo",
		Payload:    json.RawMessage(`{"ok":true}`),
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if string(response.Result) != `{"ok":true}` {
		t.Fatalf("Invoke() result = %s", response.Result)
	}

	if _, err := host.Invoke(context.Background(), protocol.InvokeRequest{Capability: "missing"}); !errors.Is(err, ErrCapabilityNotFound) {
		t.Fatalf("Invoke(missing) error = %v, want ErrCapabilityNotFound", err)
	}
}

func TestHostRecordsOperationEvents(t *testing.T) {
	now := time.Date(2026, 6, 15, 2, 0, 0, 0, time.UTC)
	var events []observability.Event
	host := New(Config{
		Enabled: true,
		Now: func() time.Time {
			return now
		},
		Recorder: observability.RecorderFunc(func(_ context.Context, event observability.Event) {
			events = append(events, event)
		}),
	})
	if err := host.RegisterProvider(ProviderFunc{
		Definition: protocol.Capability{Name: "demo.echo"},
		Handler: func(_ context.Context, req protocol.InvokeRequest) (json.RawMessage, error) {
			return append(json.RawMessage(nil), req.Payload...), nil
		},
	}); err != nil {
		t.Fatalf("RegisterProvider() error = %v", err)
	}

	_, err := host.Invoke(context.Background(), protocol.InvokeRequest{
		RequestMeta: protocol.RequestMeta{
			RequestID:      "req-1",
			TraceID:        "trace-1",
			IdempotencyKey: "idem-1",
			Metadata:       map[string]string{"source": "test"},
		},
		Capability: "demo.echo",
		Payload:    json.RawMessage(`{"ok":true}`),
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	_, _ = host.Invoke(context.Background(), protocol.InvokeRequest{Capability: "missing"})

	if len(events) != 2 {
		t.Fatalf("recorded events = %d, want 2", len(events))
	}
	if events[0].Operation != protocol.OperationInvoke || events[0].Status != observability.StatusOK || events[0].Capability != "demo.echo" {
		t.Fatalf("success event = %#v", events[0])
	}
	if events[0].RequestID != "req-1" || events[0].TraceID != "trace-1" || events[0].IdempotencyKey != "idem-1" {
		t.Fatalf("success event meta = %#v", events[0])
	}
	if events[0].Metadata["source"] != "test" {
		t.Fatalf("success event metadata = %#v", events[0].Metadata)
	}
	if events[1].Operation != protocol.OperationInvoke || events[1].Status != observability.StatusError || events[1].Error == "" {
		t.Fatalf("error event = %#v", events[1])
	}
}

func TestHostRejectsDisallowedCapabilityPermissionOnRegister(t *testing.T) {
	host := New(Config{
		Enabled:    true,
		Authorizer: securityAuthorizer([]string{"plugin:demo"}),
	})
	_, err := host.Register(context.Background(), protocol.RegisterRequest{Plugin: protocol.PluginMetadata{
		PluginID:   "demo",
		InstanceID: "demo-1",
		Name:       "Demo",
		Version:    "0.1.0",
		Protocol:   protocol.TransportHTTP,
		Endpoint:   "http://127.0.0.1:10098",
		Capabilities: []protocol.Capability{
			{Name: "demo.admin", Permissions: []string{"admin:root"}},
		},
		Permissions: []string{"plugin:demo"},
	}})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Register() error = %v, want ErrUnauthorized", err)
	}
}

func TestHostRejectsProviderInvokeWithoutAllowedPermission(t *testing.T) {
	host := New(Config{
		Enabled:    true,
		Authorizer: securityAuthorizer([]string{"system:read"}),
	})
	if err := host.RegisterProvider(ProviderFunc{
		Definition: protocol.Capability{Name: "iam.apiToken.issue", Permissions: []string{"api_token:create"}},
		Handler: func(context.Context, protocol.InvokeRequest) (json.RawMessage, error) {
			return json.RawMessage(`{"ok":true}`), nil
		},
	}); err != nil {
		t.Fatalf("RegisterProvider() error = %v", err)
	}

	_, err := host.Invoke(context.Background(), protocol.InvokeRequest{Capability: "iam.apiToken.issue"})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Invoke() error = %v, want ErrUnauthorized", err)
	}
}

func TestHostPassesInvokeOperationToAuthorizer(t *testing.T) {
	host := New(Config{
		Enabled: true,
		Authorizer: security.AuthorizerFunc(func(_ context.Context, _ security.Principal, req security.PermissionRequest) (security.Decision, error) {
			return security.Decision{Allowed: req.Operation.Name != protocol.OperationInvoke}, nil
		}),
	})
	if err := host.RegisterProvider(ProviderFunc{
		Definition: protocol.Capability{Name: "demo.echo"},
		Handler: func(context.Context, protocol.InvokeRequest) (json.RawMessage, error) {
			return json.RawMessage(`{"ok":true}`), nil
		},
	}); err != nil {
		t.Fatalf("RegisterProvider() error = %v", err)
	}

	_, err := host.Invoke(context.Background(), protocol.InvokeRequest{Capability: "demo.echo"})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Invoke() error = %v, want ErrUnauthorized", err)
	}
}

func TestHostAuthorizesEventSubscription(t *testing.T) {
	host := New(Config{
		Enabled: true,
		Authorizer: security.AuthorizerFunc(func(_ context.Context, _ security.Principal, req security.PermissionRequest) (security.Decision, error) {
			return security.Decision{Allowed: req.Operation.Name != protocol.OperationSubscribeEvent}, nil
		}),
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
		t.Fatalf("Register() error = %v", err)
	}

	_, err = host.SubscribeEvent(context.Background(), protocol.SubscribeEventRequest{
		PluginID:   "demo",
		InstanceID: "demo-1",
		Events:     []string{"demo.event"},
	})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("SubscribeEvent() error = %v, want ErrUnauthorized", err)
	}
}

func TestHostAuthorizesInjectedContext(t *testing.T) {
	host := New(Config{
		Enabled:          true,
		InjectionEnabled: true,
		Authorizer: security.AuthorizerFunc(func(_ context.Context, _ security.Principal, req security.PermissionRequest) (security.Decision, error) {
			return security.Decision{Allowed: req.Operation.Name != protocol.OperationInjectContext}, nil
		}),
	})
	if err := host.RegisterInjectionProvider(testInjectionProvider()); err != nil {
		t.Fatalf("RegisterInjectionProvider() error = %v", err)
	}

	_, err := host.InjectContext(context.Background(), protocol.InjectContextRequest{Capabilities: []string{"demo.context"}})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("InjectContext() error = %v, want ErrUnauthorized", err)
	}
}

func TestHostRejectsProviderInvokeFromOfflinePlugin(t *testing.T) {
	now := time.Date(2026, 6, 15, 1, 0, 0, 0, time.UTC)
	host := New(Config{
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
		t.Fatalf("Register() error = %v", err)
	}
	if err := host.RegisterProvider(ProviderFunc{
		Definition: protocol.Capability{Name: "demo.echo"},
		Handler: func(_ context.Context, req protocol.InvokeRequest) (json.RawMessage, error) {
			return append(json.RawMessage(nil), req.Payload...), nil
		},
	}); err != nil {
		t.Fatalf("RegisterProvider() error = %v", err)
	}

	now = now.Add(2 * time.Second)
	_, err = host.Invoke(context.Background(), protocol.InvokeRequest{PluginID: "demo", InstanceID: "demo-1", Capability: "demo.echo"})
	if !errors.Is(err, ErrPluginOffline) {
		t.Fatalf("Invoke() error = %v, want ErrPluginOffline", err)
	}
}

func TestHostExpireLeasesIsPublicLifecycleOperation(t *testing.T) {
	now := time.Date(2026, 6, 15, 1, 0, 0, 0, time.UTC)
	host := New(Config{
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
		t.Fatalf("Register() error = %v", err)
	}

	now = now.Add(2 * time.Second)
	expired, err := host.ExpireLeases(context.Background())
	if err != nil {
		t.Fatalf("ExpireLeases() error = %v", err)
	}
	if len(expired) != 1 || expired[0].PluginID != "demo" {
		t.Fatalf("expired = %#v, want demo", expired)
	}
	plugin, err := host.GetPlugin(context.Background(), "demo")
	if err != nil {
		t.Fatalf("GetPlugin() error = %v", err)
	}
	if plugin.Status != protocol.StatusOffline {
		t.Fatalf("status = %q, want offline", plugin.Status)
	}
}

func TestHostNegotiateProtocol(t *testing.T) {
	host := New(Config{Enabled: true, AllowedTransports: []string{protocol.TransportHTTP}})
	response, err := host.NegotiateProtocol(context.Background(), protocol.NegotiateProtocolRequest{
		ProtocolVersions: []string{protocol.ProtocolVersionV1},
		Transports:       []string{protocol.TransportWebSocket, protocol.TransportHTTP},
	})
	if err != nil {
		t.Fatalf("NegotiateProtocol() error = %v", err)
	}
	if !response.Accepted || response.Transport != protocol.TransportHTTP {
		t.Fatalf("NegotiateProtocol() = %#v, want accepted http", response)
	}
}

func securityAuthorizer(permissions []string) security.ScopeAuthorizer {
	return security.ScopeAuthorizer{AllowedPermissions: permissions}
}

func testInjectionProvider() injection.Provider {
	return injection.ProviderFunc{
		Definition: injection.Capability{
			Name:        "demo.context",
			Permissions: []string{"demo:context"},
		},
		Builder: func(context.Context, injection.Request) (json.RawMessage, error) {
			return json.RawMessage(`{"ok":true}`), nil
		},
	}
}

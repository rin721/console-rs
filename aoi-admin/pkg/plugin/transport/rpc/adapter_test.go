package rpctransport

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

func TestRegisterAddsPluginProtocolMethods(t *testing.T) {
	host := plugin.New(plugin.Config{Enabled: true, AllowedTransports: []string{protocol.TransportRPC}})
	registry := newTestRegistry()

	if err := Register(registry, host); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	want := []string{
		MethodDrain,
		MethodGetInjectedSchema,
		MethodHealthCheck,
		MethodHeartbeat,
		MethodInjectContext,
		MethodInvoke,
		MethodListCapabilities,
		MethodNegotiateProtocol,
		MethodPushEvent,
		MethodRegister,
		MethodRenewLease,
		MethodReportStatus,
		MethodSubscribeEvent,
		MethodSyncMetadata,
		MethodUnregister,
	}
	if got := registry.Methods(); !equalStrings(got, want) {
		t.Fatalf("methods = %v, want %v", got, want)
	}
}

func TestRegisterRPCPluginFlow(t *testing.T) {
	host := plugin.New(plugin.Config{Enabled: true, AllowedTransports: []string{protocol.TransportRPC}})
	registry := newTestRegistry()
	if err := Register(registry, host); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	raw, err := json.Marshal(protocol.RegisterRequest{Plugin: protocol.PluginMetadata{
		PluginID:   "demo",
		InstanceID: "demo-1",
		Name:       "Demo",
		Version:    "0.1.0",
		Protocol:   protocol.TransportRPC,
		Endpoint:   "http://127.0.0.1:10099/rpc",
	}})
	if err != nil {
		t.Fatalf("marshal register request: %v", err)
	}
	result, err := registry.call(context.Background(), MethodRegister, raw)
	if err != nil {
		t.Fatalf("call register: %v", err)
	}
	response, ok := result.(protocol.RegisterResponse)
	if !ok {
		t.Fatalf("response type = %T", result)
	}
	if response.Plugin.PluginID != "demo" || response.Plugin.Status != protocol.StatusOnline {
		t.Fatalf("response = %#v", response)
	}
}

type testRegistry struct {
	handlers map[string]HandlerFunc
}

func newTestRegistry() *testRegistry {
	return &testRegistry{handlers: map[string]HandlerFunc{}}
}

func (r *testRegistry) Register(method string, handler HandlerFunc) error {
	r.handlers[method] = handler
	return nil
}

func (r *testRegistry) Methods() []string {
	methods := make([]string, 0, len(r.handlers))
	for method := range r.handlers {
		methods = append(methods, method)
	}
	sort.Strings(methods)
	return methods
}

func (r *testRegistry) call(ctx context.Context, method string, params json.RawMessage) (any, error) {
	return r.handlers[method](ctx, params)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

package rpctransport

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"github.com/rei0721/go-scaffold/internal/ports"
)

func TestRegisterAddsOnlySystemMethods(t *testing.T) {
	registry := newTestRegistry()

	if err := Register(registry); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	for _, method := range []string{"system.ping", "system.methods"} {
		if _, ok := registry.handlers[method]; !ok {
			t.Fatalf("method %s not registered; methods=%v", method, registry.Methods())
		}
	}
}

func TestRegisterDoesNotExposePluginRPCMethods(t *testing.T) {
	registry := newTestRegistry()

	if err := Register(registry); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	for _, method := range []string{"plugin.register", "plugin.heartbeat", "plugin.unregister", "plugin.listCapabilities", "plugin.invoke"} {
		if _, ok := registry.handlers[method]; ok {
			t.Fatalf("plugin method %s registered; methods=%v", method, registry.Methods())
		}
	}
}

type testRegistry struct {
	handlers map[string]ports.RPCHandlerFunc
}

func newTestRegistry() *testRegistry {
	return &testRegistry{handlers: map[string]ports.RPCHandlerFunc{}}
}

func (r *testRegistry) Register(method string, handler ports.RPCHandlerFunc) error {
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

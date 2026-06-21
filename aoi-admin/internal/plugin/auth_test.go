package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/rei0721/go-scaffold/internal/ports"
	pluginpkg "github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

func TestAuthenticatorAcceptsSharedSecretHeader(t *testing.T) {
	t.Setenv("PLUGIN_SECRET", "secret")
	auth := newAuthenticator(Config{RegistrationAuthMode: "shared_secret", SharedSecretEnv: "PLUGIN_SECRET"})
	ctx := &testHTTPContext{header: http.Header{pluginSecretHeader: []string{"secret"}}}

	if err := auth.HTTP(ctx, protocol.OperationRegister, nil); err != nil {
		t.Fatalf("HTTP auth error = %v", err)
	}
}

func TestAuthenticatorRejectsMissingSharedSecret(t *testing.T) {
	t.Setenv("PLUGIN_SECRET", "secret")
	auth := newAuthenticator(Config{RegistrationAuthMode: "shared_secret", SharedSecretEnv: "PLUGIN_SECRET"})

	err := auth.RPC(context.Background(), protocol.OperationRegister, nil)
	if !errors.Is(err, pluginpkg.ErrUnauthorized) {
		t.Fatalf("auth error = %v, want ErrUnauthorized", err)
	}
}

func TestProtocolHandlerRegistersRPCWithAuth(t *testing.T) {
	t.Setenv("PLUGIN_SECRET", "secret")
	host := pluginpkg.New(pluginpkg.Config{Enabled: true, AllowedTransports: []string{protocol.TransportRPC}})
	handler := NewProtocolHandler(host, Config{RPCEnabled: true, RegistrationAuthMode: "shared_secret", SharedSecretEnv: "PLUGIN_SECRET"}, newAuthenticator(Config{
		RegistrationAuthMode: "shared_secret",
		SharedSecretEnv:      "PLUGIN_SECRET",
	}))
	registry := &testRPCRegistry{handlers: map[string]ports.RPCHandlerFunc{}}
	if err := handler.RegisterRPC(registry); err != nil {
		t.Fatalf("RegisterRPC() error = %v", err)
	}

	raw, err := json.Marshal(protocol.NegotiateProtocolRequest{
		RequestMeta:      protocol.RequestMeta{Auth: &protocol.Auth{Mode: "shared_secret", Token: "secret"}},
		ProtocolVersions: []string{protocol.ProtocolVersionV1},
		Transports:       []string{protocol.TransportRPC},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	result, err := registry.handlers["plugin.negotiate"](context.Background(), raw)
	if err != nil {
		t.Fatalf("plugin.negotiate error = %v", err)
	}
	response, ok := result.(protocol.NegotiateProtocolResponse)
	if !ok || !response.Accepted || response.Transport != protocol.TransportRPC {
		t.Fatalf("response = %#v", result)
	}
}

type testHTTPContext struct {
	header http.Header
}

func (c *testHTTPContext) RequestContext() context.Context { return context.Background() }
func (c *testHTTPContext) BindJSON(any) error              { return nil }
func (c *testHTTPContext) JSON(int, any)                   {}
func (c *testHTTPContext) GetHeader(name string) string    { return c.header.Get(name) }

type testRPCRegistry struct {
	handlers map[string]ports.RPCHandlerFunc
}

func (r *testRPCRegistry) Register(method string, handler ports.RPCHandlerFunc) error {
	r.handlers[method] = handler
	return nil
}

func (r *testRPCRegistry) Methods() []string {
	methods := make([]string, 0, len(r.handlers))
	for method := range r.handlers {
		methods = append(methods, method)
	}
	return methods
}

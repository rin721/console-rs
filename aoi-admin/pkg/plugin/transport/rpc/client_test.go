package rpctransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

func TestRemoteInvokerCallsJSONRPCEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode rpc request: %v", err)
		}
		if req.Method != MethodInvoke {
			t.Fatalf("method = %q", req.Method)
		}
		_ = json.NewEncoder(w).Encode(response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: mustJSON(t, protocol.InvokeResponse{
				Capability: "demo.echo",
				Result:     json.RawMessage(`{"ok":true}`),
			}),
		})
	}))
	defer server.Close()

	invoker := NewRemoteInvoker(server.Client())
	result, err := invoker.Invoke(context.Background(), protocol.PluginSnapshot{PluginMetadata: protocol.PluginMetadata{
		PluginID: "demo",
		Protocol: protocol.TransportRPC,
		Endpoint: server.URL,
	}}, protocol.InvokeRequest{Capability: "demo.echo"})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if string(result) != `{"ok":true}` {
		t.Fatalf("result = %s", result)
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return raw
}

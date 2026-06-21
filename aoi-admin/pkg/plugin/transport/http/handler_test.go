package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

func TestHandlerRegistersPlugin(t *testing.T) {
	host := plugin.New(plugin.Config{Enabled: true})
	body, err := json.Marshal(protocol.RegisterRequest{Plugin: protocol.PluginMetadata{
		PluginID:   "demo",
		InstanceID: "demo-1",
		Name:       "Demo",
		Version:    "0.1.0",
		Protocol:   protocol.TransportHTTP,
		Endpoint:   "http://127.0.0.1:10098",
	}})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	ctx := newFakeContext(body)

	New(host).Register(ctx)

	if ctx.status != http.StatusOK {
		t.Fatalf("status = %d, want %d: %#v", ctx.status, http.StatusOK, ctx.response)
	}
	response, ok := ctx.response.(Response)
	if !ok {
		t.Fatalf("response type = %T, want Response", ctx.response)
	}
	raw, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var registered protocol.RegisterResponse
	if err := json.Unmarshal(raw, &registered); err != nil {
		t.Fatalf("decode registered response: %v", err)
	}
	if registered.Plugin.PluginID != "demo" || registered.Plugin.Status != protocol.StatusOnline {
		t.Fatalf("registered = %#v", registered)
	}
}

func TestRemoteInvokerCallsPluginEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/invoke":
			var req protocol.InvokeRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode invoke request: %v", err)
			}
			if req.Capability != "demo.echo" {
				t.Fatalf("capability = %q", req.Capability)
			}
			_ = json.NewEncoder(w).Encode(Response{Data: protocol.InvokeResponse{
				Capability: req.Capability,
				Result:     json.RawMessage(`{"ok":true}`),
			}})
		case "/events":
			_ = json.NewEncoder(w).Encode(Response{Data: protocol.PushEventResponse{Accepted: true, Event: "demo.event"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	invoker := NewRemoteInvoker(server.Client())
	remote := protocol.PluginSnapshot{PluginMetadata: protocol.PluginMetadata{
		PluginID:   "demo",
		InstanceID: "demo-1",
		Protocol:   protocol.TransportHTTP,
		Endpoint:   server.URL,
	}}
	result, err := invoker.Invoke(context.Background(), remote, protocol.InvokeRequest{Capability: "demo.echo"})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if string(result) != `{"ok":true}` {
		t.Fatalf("result = %s", result)
	}
	if err := invoker.PushEvent(context.Background(), remote, protocol.PushEventRequest{Event: "demo.event"}); err != nil {
		t.Fatalf("PushEvent() error = %v", err)
	}
}

type fakeContext struct {
	request  *http.Request
	response any
	status   int
}

func newFakeContext(body []byte) *fakeContext {
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/plugin-api/v1/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return &fakeContext{request: req}
}

func (c *fakeContext) RequestContext() context.Context {
	return c.request.Context()
}

func (c *fakeContext) BindJSON(target any) error {
	return json.NewDecoder(c.request.Body).Decode(target)
}

func (c *fakeContext) JSON(status int, value any) {
	c.status = status
	c.response = value
}

func (c *fakeContext) GetHeader(name string) string {
	return c.request.Header.Get(name)
}

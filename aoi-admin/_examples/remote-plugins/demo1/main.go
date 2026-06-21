package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"demo1/protocol"
)

type client struct {
	baseURL string
	http    *http.Client
	secret  string
}

type hostResponse struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error *protocol.Error `json:"error,omitempty"`
}

func main() {
	var host string
	var id string
	var instanceID string
	var endpoint string
	var secret string
	var transports string
	var heartbeatInterval time.Duration
	flag.StringVar(&host, "host", "http://127.0.0.1:9999/plugin-api/v1", "plugin host HTTP protocol base URL")
	flag.StringVar(&id, "id", "demo1", "plugin id")
	flag.StringVar(&instanceID, "instance", "", "plugin instance id; defaults to plugin id plus hostname")
	flag.StringVar(&endpoint, "endpoint", "http://127.0.0.1:10098", "public endpoint of this remote plugin")
	flag.StringVar(&secret, "secret", "", "optional shared secret sent as X-Plugin-Secret")
	flag.StringVar(&transports, "transports", protocol.TransportHTTP, "comma-separated transports to offer: http,websocket,rpc")
	flag.DurationVar(&heartbeatInterval, "heartbeat-interval", 10*time.Second, "heartbeat interval")
	flag.Parse()
	if instanceID == "" {
		instanceID = defaultInstanceID(id)
	}

	callbackServer := newCallbackServer(endpoint)
	go func() {
		if err := callbackServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	defer callbackServer.Shutdown(context.Background())

	api := client{
		baseURL: strings.TrimRight(host, "/"),
		http:    &http.Client{Timeout: 5 * time.Second},
		secret:  secret,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	negotiated := protocol.NegotiateProtocolResponse{}
	if err := api.post(ctx, "/negotiate", protocol.NegotiateProtocolRequest{
		PluginID:         id,
		InstanceID:       instanceID,
		ProtocolVersions: []string{protocol.ProtocolVersionV1},
		Transports:       splitCSV(transports),
	}, &negotiated); err != nil {
		panic(err)
	}
	if !negotiated.Accepted {
		panic("protocol negotiation rejected")
	}

	metadata := protocol.PluginMetadata{
		PluginID:      id,
		InstanceID:    instanceID,
		Name:          "Demo Remote Plugin",
		Version:       "0.1.0",
		Protocol:      protocol.PluginProtocolJSON,
		Transport:     negotiated.Transport,
		Endpoint:      endpointForTransport(endpoint, negotiated.Transport),
		SchemaVersion: protocol.ProtocolVersionV1,
		Capabilities: []protocol.Capability{
			{
				Name:        "demo.echo",
				Version:     "v1",
				Scope:       protocol.CapabilityScopePlugin,
				Description: "Echo payloads handled by the remote demo plugin process.",
			},
		},
		Permissions: []string{"plugin:demo"},
		Hooks:       []string{"on_start", "on_stop"},
		Metadata:    map[string]string{"example": "true"},
	}
	if err := api.post(ctx, "/register", protocol.RegisterRequest{Plugin: metadata}, nil); err != nil {
		panic(err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = api.post(shutdownCtx, "/unregister", protocol.UnregisterRequest{PluginID: id, InstanceID: instanceID}, nil)
	}()
	fmt.Printf("registered %s/%s with %s\n", id, instanceID, api.baseURL)

	if err := api.post(ctx, "/subscriptions", protocol.SubscribeEventRequest{
		PluginID:   id,
		InstanceID: instanceID,
		Events:     []string{"demo.event"},
	}, nil); err != nil {
		fmt.Printf("subscribe event failed: %v\n", err)
	}
	if err := api.post(ctx, "/status", protocol.ReportStatusRequest{
		PluginID:      id,
		InstanceID:    instanceID,
		Status:        protocol.StatusOnline,
		RuntimeStatus: protocol.RuntimeStatusReady,
	}, nil); err != nil {
		fmt.Printf("report status failed: %v\n", err)
	}

	var schema protocol.GetInjectedSchemaResponse
	if err := api.post(ctx, "/injected-schema", protocol.GetInjectedSchemaRequest{}, &schema); err == nil {
		fmt.Printf("injected schema capabilities: %d\n", len(schema.Capabilities))
	}
	if err := api.post(ctx, "/invoke", protocol.InvokeRequest{
		Capability: "iam.apiToken.issue",
		Payload:    json.RawMessage(`{}`),
	}, nil); err != nil {
		fmt.Printf("invoke iam.apiToken.issue rejected as expected: %v\n", err)
	}

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("shutting down %s/%s\n", id, instanceID)
			return
		case <-ticker.C:
			if err := api.post(ctx, "/lease", protocol.RenewLeaseRequest{PluginID: id, InstanceID: instanceID}, nil); err != nil {
				fmt.Printf("lease renewal failed: %v\n", err)
				continue
			}
			fmt.Printf("lease renewed for %s/%s\n", id, instanceID)
		}
	}
}

func (c client) post(ctx context.Context, path string, payload any, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.secret != "" {
		req.Header.Set("X-Plugin-Secret", c.secret)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var decoded hostResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("decode host response: %w: %s", err, strings.TrimSpace(string(raw)))
	}
	if decoded.Error != nil {
		return fmt.Errorf("%s: %s", decoded.Error.Code, decoded.Error.Message)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("host http status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if result != nil && len(decoded.Data) > 0 {
		return json.Unmarshal(decoded.Data, result)
	}
	return nil
}

func newCallbackServer(endpoint string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/invoke", handleInvoke)
	mux.HandleFunc("/events", handleEvent)
	mux.HandleFunc("/drain", handleDrain)
	mux.HandleFunc("/rpc", handleRPC)
	return &http.Server{
		Addr:         listenAddress(endpoint),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
}

func handleDrain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, hostResponse{Error: &protocol.Error{Code: protocol.ErrorCodeInvalidPlugin, Message: "method must be POST"}})
		return
	}
	var req protocol.DrainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, hostResponse{Error: &protocol.Error{Code: protocol.ErrorCodeInvalidPlugin, Message: "invalid drain payload"}})
		return
	}
	fmt.Printf("drain requested for %s/%s: %s\n", req.PluginID, req.InstanceID, req.Reason)
	writeJSON(w, http.StatusOK, hostResponse{Data: mustMarshal(protocol.DrainResponse{})})
}

func handleInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, hostResponse{Error: &protocol.Error{Code: protocol.ErrorCodeInvalidPlugin, Message: "method must be POST"}})
		return
	}
	var req protocol.InvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, hostResponse{Error: &protocol.Error{Code: protocol.ErrorCodeInvalidPlugin, Message: "invalid invoke payload"}})
		return
	}
	if req.Capability != "demo.echo" {
		writeJSON(w, http.StatusNotFound, hostResponse{Error: &protocol.Error{Code: protocol.ErrorCodeCapabilityNotFound, Message: "capability not found"}})
		return
	}
	writeJSON(w, http.StatusOK, hostResponse{Data: mustMarshal(protocol.InvokeResponse{
		Capability: req.Capability,
		Result:     req.Payload,
	})})
}

func handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, hostResponse{Error: &protocol.Error{Code: protocol.ErrorCodeInvalidPlugin, Message: "method must be POST"}})
		return
	}
	var req protocol.PushEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, hostResponse{Error: &protocol.Error{Code: protocol.ErrorCodeInvalidPlugin, Message: "invalid event payload"}})
		return
	}
	fmt.Printf("received event %s: %s\n", req.Event, req.Payload)
	writeJSON(w, http.StatusOK, hostResponse{Data: mustMarshal(protocol.PushEventResponse{Accepted: true, Event: req.Event})})
}

func handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"jsonrpc": "2.0", "error": map[string]any{"code": -32600, "message": "method must be POST"}})
		return
	}
	var req struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"jsonrpc": "2.0", "error": map[string]any{"code": -32700, "message": "parse error"}})
		return
	}
	switch req.Method {
	case "plugin.invoke":
		var invoke protocol.InvokeRequest
		if err := json.Unmarshal(req.Params, &invoke); err != nil {
			writeJSON(w, http.StatusOK, rpcError(req.ID, -32602, "invalid invoke payload"))
			return
		}
		writeJSON(w, http.StatusOK, rpcResult(req.ID, protocol.InvokeResponse{Capability: invoke.Capability, Result: invoke.Payload}))
	case "plugin.pushEvent":
		var event protocol.PushEventRequest
		if err := json.Unmarshal(req.Params, &event); err != nil {
			writeJSON(w, http.StatusOK, rpcError(req.ID, -32602, "invalid event payload"))
			return
		}
		fmt.Printf("received rpc event %s: %s\n", event.Event, event.Payload)
		writeJSON(w, http.StatusOK, rpcResult(req.ID, protocol.PushEventResponse{Accepted: true, Event: event.Event}))
	case "plugin.drain":
		var drain protocol.DrainRequest
		if err := json.Unmarshal(req.Params, &drain); err != nil {
			writeJSON(w, http.StatusOK, rpcError(req.ID, -32602, "invalid drain payload"))
			return
		}
		fmt.Printf("received rpc drain for %s/%s: %s\n", drain.PluginID, drain.InstanceID, drain.Reason)
		writeJSON(w, http.StatusOK, rpcResult(req.ID, protocol.DrainResponse{}))
	default:
		writeJSON(w, http.StatusOK, rpcError(req.ID, -32601, "method not found"))
	}
}

func endpointForTransport(endpoint string, transport string) string {
	endpoint = strings.TrimRight(endpoint, "/")
	if transport == protocol.TransportRPC {
		return endpoint + "/rpc"
	}
	return endpoint
}

func listenAddress(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return "127.0.0.1:10098"
	}
	return parsed.Host
}

func defaultInstanceID(pluginID string) string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = "host"
	}
	return pluginID + "-" + hostname
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return []string{protocol.TransportHTTP}
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func mustMarshal(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`null`)
	}
	return raw
}

func rpcResult(id json.RawMessage, result any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
}

func rpcError(id json.RawMessage, code int, message string) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": message}}
}

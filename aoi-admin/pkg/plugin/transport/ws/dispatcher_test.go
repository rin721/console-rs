package ws

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

func TestDispatcherNegotiatesProtocolEnvelope(t *testing.T) {
	host := plugin.New(plugin.Config{Enabled: true})
	payload, err := json.Marshal(protocol.NegotiateProtocolRequest{
		ProtocolVersions: []string{protocol.ProtocolVersionV1},
		Transports:       []string{protocol.TransportHTTP},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	response := NewDispatcher(host).Dispatch(context.Background(), protocol.Envelope{
		ID:        "1",
		Operation: protocol.OperationNegotiateProtocol,
		Version:   protocol.ProtocolVersionV1,
		Payload:   payload,
	})
	if response.Error != nil {
		t.Fatalf("Dispatch() error = %#v", response.Error)
	}
	var negotiated protocol.NegotiateProtocolResponse
	if err := json.Unmarshal(response.Payload, &negotiated); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !negotiated.Accepted || negotiated.Transport != protocol.TransportHTTP {
		t.Fatalf("negotiated = %#v, want http", negotiated)
	}
}

func TestRemoteInvokerUsesBoundWebSocketSession(t *testing.T) {
	registry := NewSessionRegistry()
	session := NewSession(bufio.NewWriter(&bytes.Buffer{}))
	registry.Bind("demo", "demo-1", session)

	go func() {
		time.Sleep(10 * time.Millisecond)
		session.Complete(protocol.Envelope{
			ID:        "1",
			Type:      "response",
			Operation: protocol.OperationInvoke,
			Payload: mustJSON(t, protocol.InvokeResponse{
				Capability: "demo.echo",
				Result:     json.RawMessage(`{"ok":true}`),
			}),
		})
	}()

	result, err := NewRemoteInvoker(registry).Invoke(context.Background(), protocol.PluginSnapshot{PluginMetadata: protocol.PluginMetadata{
		PluginID:   "demo",
		InstanceID: "demo-1",
		Protocol:   protocol.TransportWebSocket,
	}}, protocol.InvokeRequest{Capability: "demo.echo"})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if string(result) != `{"ok":true}` {
		t.Fatalf("result = %s", result)
	}
}

func TestDispatcherServesWebSocketEnvelope(t *testing.T) {
	host := plugin.New(plugin.Config{Enabled: true})
	dispatcher := NewDispatcher(host)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := dispatcher.Serve(r.Context(), w, r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}))
	defer server.Close()

	addr := strings.TrimPrefix(server.URL, "http://")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial test server: %v", err)
	}
	defer conn.Close()

	key := "dGhlIHNhbXBsZSBub25jZQ=="
	if _, err := conn.Write([]byte("GET / HTTP/1.1\r\nHost: " + addr + "\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: " + key + "\r\nSec-WebSocket-Version: 13\r\n\r\n")); err != nil {
		t.Fatalf("write handshake: %v", err)
	}
	reader := bufio.NewReader(conn)
	status, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read handshake status: %v", err)
	}
	if !strings.Contains(status, "101") {
		t.Fatalf("handshake status = %q", status)
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read handshake header: %v", err)
		}
		if line == "\r\n" {
			break
		}
	}

	payload, err := json.Marshal(protocol.Envelope{
		ID:        "1",
		Operation: protocol.OperationNegotiateProtocol,
		Payload: mustJSON(t, protocol.NegotiateProtocolRequest{
			ProtocolVersions: []string{protocol.ProtocolVersionV1},
			Transports:       []string{protocol.TransportHTTP},
		}),
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	if _, err := conn.Write(maskedTextFrame(t, payload)); err != nil {
		t.Fatalf("write envelope frame: %v", err)
	}
	opcode, responsePayload, err := readFrame(reader)
	if err != nil {
		t.Fatalf("read response frame: %v", err)
	}
	if opcode != websocketOpcodeText {
		t.Fatalf("opcode = %d, want text", opcode)
	}
	var response protocol.Envelope
	if err := json.Unmarshal(responsePayload, &response); err != nil {
		t.Fatalf("unmarshal response envelope: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("response error = %#v", response.Error)
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return raw
}

func maskedTextFrame(t *testing.T, payload []byte) []byte {
	t.Helper()
	if len(payload) >= 126 {
		t.Fatalf("test payload too large: %d", len(payload))
	}
	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		t.Fatalf("read mask: %v", err)
	}
	frame := []byte{0x81, 0x80 | byte(len(payload))}
	frame = append(frame, mask...)
	for i, b := range payload {
		frame = append(frame, b^mask[i%4])
	}
	return frame
}

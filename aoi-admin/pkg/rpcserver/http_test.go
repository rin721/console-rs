package rpcserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRPCHandlerCallsRegisteredMethod(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t)
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"system.ping","params":{"echo":"hi"}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(body))

	handler.ServeHTTP(rec, req)

	resp := decodeRPCResponse(t, rec)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if string(resp.ID) != "1" {
		t.Fatalf("id = %s, want 1", resp.ID)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T, want map", resp.Result)
	}
	if result["ok"] != true || result["echo"] != "hi" {
		t.Fatalf("result = %#v, want ok and echo", result)
	}
}

func TestRPCHandlerReturnsMethodRegistry(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t)
	body := []byte(`{"jsonrpc":"2.0","id":"methods","method":"system.methods"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(body))

	handler.ServeHTTP(rec, req)

	resp := decodeRPCResponse(t, rec)
	gotAny, ok := resp.Result.([]any)
	if !ok {
		t.Fatalf("result type = %T, want []any", resp.Result)
	}
	got := make([]string, 0, len(gotAny))
	for _, value := range gotAny {
		got = append(got, value.(string))
	}
	want := []string{"system.methods", "system.ping"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("methods = %#v, want %#v", got, want)
	}
}

func TestRPCHandlerRejectsUnknownMethod(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t)
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"missing.method"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(body))

	handler.ServeHTTP(rec, req)

	resp := decodeRPCResponse(t, rec)
	if resp.Error == nil || resp.Error.Code != CodeMethodNotFound {
		t.Fatalf("error = %#v, want method not found", resp.Error)
	}
}

func TestRPCHandlerRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader([]byte(`{`)))

	handler.ServeHTTP(rec, req)

	resp := decodeRPCResponse(t, rec)
	if resp.Error == nil || resp.Error.Code != CodeParseError {
		t.Fatalf("error = %#v, want parse error", resp.Error)
	}
}

func TestRPCHandlerRejectsInvalidRequest(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t)
	body := []byte(`{"jsonrpc":"2.0","method":"system.ping"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(body))

	handler.ServeHTTP(rec, req)

	resp := decodeRPCResponse(t, rec)
	if resp.Error == nil || resp.Error.Code != CodeInvalidRequest {
		t.Fatalf("error = %#v, want invalid request", resp.Error)
	}
}

func TestRPCHandlerRejectsNonPost(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rpc", nil)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	resp := decodeRPCResponse(t, rec)
	if resp.Error == nil || resp.Error.Code != CodeInvalidRequest {
		t.Fatalf("error = %#v, want invalid request", resp.Error)
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()

	registry := NewRegistry()
	if err := registry.Register("system.ping", func(_ context.Context, params json.RawMessage) (any, error) {
		result := map[string]any{"ok": true}
		var values map[string]any
		if len(params) > 0 && string(params) != "null" {
			if err := json.Unmarshal(params, &values); err != nil {
				return nil, InvalidParams("params must be an object")
			}
			if echo, ok := values["echo"]; ok {
				result["echo"] = echo
			}
		}
		return result, nil
	}); err != nil {
		t.Fatalf("register ping: %v", err)
	}
	if err := registry.Register("system.methods", func(context.Context, json.RawMessage) (any, error) {
		return registry.Methods(), nil
	}); err != nil {
		t.Fatalf("register methods: %v", err)
	}
	return NewHandler(registry)
}

func decodeRPCResponse(t *testing.T, rec *httptest.ResponseRecorder) Response {
	t.Helper()

	var raw struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  any             `json:"result"`
		Error   *Error          `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return Response{
		JSONRPC: raw.JSONRPC,
		ID:      raw.ID,
		Result:  raw.Result,
		Error:   raw.Error,
	}
}

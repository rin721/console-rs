package rpcserver

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestRegistryRegisterAndMethods(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	handler := func(context.Context, json.RawMessage) (any, error) {
		return map[string]bool{"ok": true}, nil
	}

	if err := registry.Register("z.method", handler); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := registry.Register("a.method", handler); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	got := registry.Methods()
	want := []string{"a.method", "z.method"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Methods() = %#v, want %#v", got, want)
	}
}

func TestRegistryRejectsInvalidRegistration(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	handler := func(context.Context, json.RawMessage) (any, error) {
		return nil, nil
	}
	if err := registry.Register("", handler); err == nil {
		t.Fatal("Register(empty) error = nil, want error")
	}
	if err := registry.Register("system.ping", nil); err == nil {
		t.Fatal("Register(nil handler) error = nil, want error")
	}
	if err := registry.Register("system.ping", handler); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := registry.Register("system.ping", handler); err == nil {
		t.Fatal("Register(duplicate) error = nil, want error")
	}
}

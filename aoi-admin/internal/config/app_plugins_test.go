package config

import (
	"reflect"
	"testing"
)

func TestPluginsConfigAllowsRPCOnlyTransport(t *testing.T) {
	cfg := PluginsConfig{
		Enabled:                 true,
		AllowedTransports:       []string{"rpc"},
		RPCEnabled:              true,
		RequestTimeoutSeconds:   10,
		HeartbeatTimeoutSeconds: 30,
		RegistrationAuthMode:    "none",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if cfg.HTTPEnabled || cfg.WSEnabled {
		t.Fatalf("rpc-only config should not enable http/ws: %#v", cfg)
	}
}

func TestPluginsConfigRejectsUnknownTransport(t *testing.T) {
	cfg := PluginsConfig{
		Enabled:                 true,
		AllowedTransports:       []string{"nats"},
		HTTPEnabled:             true,
		RequestTimeoutSeconds:   10,
		HeartbeatTimeoutSeconds: 30,
		RegistrationAuthMode:    "none",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want unsupported transport")
	}
}

func TestPluginsConfigDoesNotModelPrivatePluginConfig(t *testing.T) {
	typ := reflect.TypeOf(PluginsConfig{})
	for _, forbidden := range []string{"items", "manifests", "configs", "plugins", "private_config"} {
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if field.Tag.Get("mapstructure") == forbidden || field.Tag.Get("yaml") == forbidden {
				t.Fatalf("PluginsConfig must not model private plugin config field %q", forbidden)
			}
		}
	}
}

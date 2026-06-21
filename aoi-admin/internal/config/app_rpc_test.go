package config

import "testing"

func TestRPCConfigDisabledZeroValueIsValid(t *testing.T) {
	t.Parallel()

	var cfg RPCConfig
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestRPCConfigEnabledAppliesDefaults(t *testing.T) {
	t.Parallel()

	cfg := RPCConfig{Enabled: true}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if cfg.Host != "127.0.0.1" || cfg.Port != DefaultRPCPort ||
		cfg.ReadTimeout != DefaultRPCReadTimeout ||
		cfg.WriteTimeout != DefaultRPCWriteTimeout ||
		cfg.IdleTimeout != DefaultRPCIdleTimeout {
		t.Fatalf("defaults not applied: %#v", cfg)
	}
}

func TestRPCConfigEnabledRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  RPCConfig
	}{
		{name: "invalid port", cfg: RPCConfig{Enabled: true, Host: "127.0.0.1", Port: 70000, ReadTimeout: 1, WriteTimeout: 1, IdleTimeout: 1}},
		{name: "invalid read timeout", cfg: RPCConfig{Enabled: true, Host: "127.0.0.1", Port: 10099, ReadTimeout: -1, WriteTimeout: 1, IdleTimeout: 1}},
		{name: "invalid write timeout", cfg: RPCConfig{Enabled: true, Host: "127.0.0.1", Port: 10099, ReadTimeout: 1, WriteTimeout: -1, IdleTimeout: 1}},
		{name: "invalid idle timeout", cfg: RPCConfig{Enabled: true, Host: "127.0.0.1", Port: 10099, ReadTimeout: 1, WriteTimeout: 1, IdleTimeout: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
		})
	}
}

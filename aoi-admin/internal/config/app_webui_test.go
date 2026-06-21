package config

import "testing"

// TestWebUIConfigDefaultsAndValidation 固定 WebUI 静态托管配置的默认值和保留路径边界。
func TestWebUIConfigDefaultsAndValidation(t *testing.T) {
	cfg := WebUIConfig{}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() with defaults error = %v", err)
	}
	if !cfg.EnabledValue() || cfg.MountPath != DefaultWebUIMountPath || cfg.DistDir != DefaultWebUIDistDir {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}

	rootMounted := WebUIConfig{Enabled: boolPtr(true), MountPath: "/", DistDir: "./dist"}
	if err := rootMounted.Validate(); err != nil {
		t.Fatalf("expected root mount_path to be allowed, got %v", err)
	}

	for _, mountPath := range []string{"/api", "/api/v1", "/api/v1/admin", "/health", "/ready"} {
		cfg := WebUIConfig{Enabled: boolPtr(true), MountPath: mountPath, DistDir: "./dist"}
		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected mount_path %q to be rejected", mountPath)
		}
	}
}

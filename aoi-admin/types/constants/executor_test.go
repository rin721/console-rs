package constants

// 本测试文件固定跨包公共类型的导入边界和响应契约，防止注释补全和后续重构改变外部可观察行为。

import "testing"

// TestPoolNameConstants 验证池名常量的值。
func TestPoolNameConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"AppPoolHTTP", AppPoolHTTP, "http"},
		{"AppPoolDatabase", AppPoolDatabase, "database"},
		{"AppPoolCache", AppPoolCache, "cache"},
		{"AppPoolLogger", AppPoolLogger, "logger"},
		{"AppPoolBackground", AppPoolBackground, "background"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestPoolNameUniqueness 验证池名常量的唯一性。
func TestPoolNameUniqueness(t *testing.T) {
	pools := []string{
		AppPoolHTTP,
		AppPoolDatabase,
		AppPoolCache,
		AppPoolLogger,
		AppPoolBackground,
	}

	seen := make(map[string]bool)
	for _, pool := range pools {
		if seen[pool] {
			t.Errorf("Duplicate pool name found: %s", pool)
		}
		seen[pool] = true
	}

	// 验证定义了5个池。
	if len(pools) != 5 {
		t.Errorf("Expected 5 pools, got %d", len(pools))
	}
}

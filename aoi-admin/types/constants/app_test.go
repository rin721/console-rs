package constants

// 本测试文件固定跨包公共类型的导入边界和响应契约，防止注释补全和后续重构改变外部可观察行为。

import "testing"

// TestApplicationConstants 固定跨包公共类型的导入边界和响应契约，确保后续注释补全或结构调整不改变该场景。
func TestApplicationConstants(t *testing.T) {
	if AppPrefix != "Rin" {
		t.Fatalf("AppPrefix = %q, want %q", AppPrefix, "Rin")
	}

	if AppServerCommandName != "server" {
		t.Fatalf("AppServerCommandName = %q, want %q", AppServerCommandName, "server")
	}

	if EnvConfigPathName != "RIN_CONFIG_PATH" {
		t.Fatalf("EnvConfigPathName = %q, want %q", EnvConfigPathName, "RIN_CONFIG_PATH")
	}
}

func TestHTTPAPIPathContracts(t *testing.T) {
	if APIPathRoot != "/api" {
		t.Fatalf("APIPathRoot = %q, want %q", APIPathRoot, "/api")
	}
	if APIBasePath != "/api/v1" {
		t.Fatalf("APIBasePath = %q, want %q", APIBasePath, "/api/v1")
	}
	if APIBasePrefix != "/api/v1/" {
		t.Fatalf("APIBasePrefix = %q, want %q", APIBasePrefix, "/api/v1/")
	}
	if got := APIPath("system", "media", "assets", "42", "download"); got != "/api/v1/system/media/assets/42/download" {
		t.Fatalf("APIPath(media download) = %q", got)
	}
	if got := APIPath("/system/", "/apis/"); got != "/api/v1/system/apis" {
		t.Fatalf("APIPath(trimmed parts) = %q", got)
	}
	if !IsAPIPath("/api/v1/system/apis") {
		t.Fatal("expected /api/v1/system/apis to be recognized as an API path")
	}
	if IsAPIPath("/api/v1") {
		t.Fatal("expected bare /api/v1 to stay outside concrete API path checks")
	}
	if got := TrimAPIPathPrefix("/api/v1/system/apis"); got != "system/apis" {
		t.Fatalf("TrimAPIPathPrefix = %q, want %q", got, "system/apis")
	}
	if got := MediaAssetDownloadPath(42); got != "/api/v1/system/media/assets/42/download" {
		t.Fatalf("MediaAssetDownloadPath = %q", got)
	}
}

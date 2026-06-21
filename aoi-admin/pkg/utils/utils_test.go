package utils

import (
	"encoding/hex"
	"net"
	"strconv"
	"testing"

	"github.com/rei0721/go-scaffold/pkg/i18n"
)

type fakeI18n struct {
	lang         string
	namespace    string
	key          string
	templateData map[string]any
	missing      []i18n.MissingKey
}

func (f *fakeI18n) Localize(lang string, namespace string, key string, templateData map[string]any) string {
	f.lang = lang
	f.namespace = namespace
	f.key = key
	f.templateData = templateData
	return lang + ":" + namespace + ":" + key
}

func (f *fakeI18n) ResolveLocale(candidates ...string) string {
	for _, candidate := range candidates {
		if candidate != "" {
			return candidate
		}
	}
	return "zh-CN"
}

func (f *fakeI18n) ValidateResources() error {
	return nil
}

func (f *fakeI18n) IsSupported(string) bool {
	return true
}

func (f *fakeI18n) DefaultLocale() string {
	return "zh-CN"
}

func (f *fakeI18n) FallbackLocale() string {
	return "zh-CN"
}

func (f *fakeI18n) LoadNamespace(string, string) error {
	return nil
}

func (f *fakeI18n) MissingKeys() []i18n.MissingKey {
	return f.missing
}

func TestSnowflakeGeneratesIDsAndRejectsInvalidNode(t *testing.T) {
	gen, err := NewSnowflake(1)
	if err != nil {
		t.Fatalf("NewSnowflake() error = %v", err)
	}

	first := gen.NextID()
	second := gen.NextID()
	if first <= 0 || second <= 0 {
		t.Fatalf("NextID() generated non-positive IDs: %d, %d", first, second)
	}
	if first == second {
		t.Fatalf("NextID() generated duplicate IDs: %d", first)
	}

	asString := gen.NextIDString()
	parsed, err := strconv.ParseInt(asString, 10, 64)
	if err != nil {
		t.Fatalf("NextIDString() = %q, parse error = %v", asString, err)
	}
	if parsed <= 0 {
		t.Fatalf("NextIDString() parsed to non-positive ID: %d", parsed)
	}

	defaultGen := DefaultSnowflake()
	if defaultGen.NextID() <= 0 {
		t.Fatal("DefaultSnowflake().NextID() generated a non-positive ID")
	}

	if _, err := NewSnowflake(1024); err == nil {
		t.Fatal("NewSnowflake(1024) error = nil, want invalid node error")
	}
}

func TestListenAddrValidation(t *testing.T) {
	valid := []string{
		":8080",
		"0.0.0.0:8080",
		"127.0.0.1:8080",
		"localhost:8080",
		"[::]:8080",
	}
	for _, addr := range valid {
		if err := IsValidListenAddr(addr); err != nil {
			t.Fatalf("IsValidListenAddr(%q) error = %v", addr, err)
		}
	}

	invalid := []string{
		"",
		"invalid",
		":0",
		":65536",
		"not-a-host:8080",
		"203.0.113.1:8080",
	}
	for _, addr := range invalid {
		if err := IsValidListenAddr(addr); err == nil {
			t.Fatalf("IsValidListenAddr(%q) error = nil, want error", addr)
		}
	}
}

func TestHTTPListenAddrValidation(t *testing.T) {
	if err := IsValidHTTPListenAddr("127.0.0.1:0"); err != nil {
		t.Fatalf("IsValidHTTPListenAddr() with ephemeral loopback error = %v", err)
	}
	if err := IsValidHTTPListenAddr(""); err == nil {
		t.Fatal("IsValidHTTPListenAddr(\"\") error = nil, want error")
	}
	if err := IsValidHTTPListenAddr("203.0.113.1:0"); err == nil {
		t.Fatal("IsValidHTTPListenAddr(\"203.0.113.1:0\") error = nil, want bind error")
	}
}

func TestGetAvailablePortHonorsRangeAndExclude(t *testing.T) {
	invalidRanges := [][2]int{
		{0, 1},
		{10002, 10001},
		{1, 65536},
	}
	for _, ports := range invalidRanges {
		if _, err := GetAvailablePort(ports[0], ports[1]); err == nil {
			t.Fatalf("GetAvailablePort(%d, %d) error = nil, want error", ports[0], ports[1])
		}
	}

	port := freeLoopbackPort(t)
	got, err := GetAvailablePort(port, port)
	if err != nil {
		t.Fatalf("GetAvailablePort(%d, %d) error = %v", port, port, err)
	}
	if got != port {
		t.Fatalf("GetAvailablePort(%d, %d) = %d, want %d", port, port, got, port)
	}

	if _, err := GetAvailablePort(port, port, port); err == nil {
		t.Fatalf("GetAvailablePort(%d, %d, %d) error = nil, want no available port", port, port, port)
	}
}

func TestGenerateDeviceIDIsStableAndSalted(t *testing.T) {
	first := GenerateDeviceID("app-a")
	second := GenerateDeviceID("app-a")
	third := GenerateDeviceID("app-b")

	if first != second {
		t.Fatalf("GenerateDeviceID() is not stable for the same salt: %q != %q", first, second)
	}
	if first == third {
		t.Fatal("GenerateDeviceID() returned the same value for different salts")
	}
	if len(first) != 64 {
		t.Fatalf("GenerateDeviceID() length = %d, want 64", len(first))
	}
	if _, err := hex.DecodeString(first); err != nil {
		t.Fatalf("GenerateDeviceID() = %q, invalid hex: %v", first, err)
	}
}

func TestI18nUtilsUsesDefaultLocaleAndNamespace(t *testing.T) {
	backend := &fakeI18n{}
	utils := NewI18nUtils(backend, "zh-CN")

	template := map[string]any{"Name": "Codex"}
	got := utils.UI("welcome", template)
	if got != "zh-CN:ui:welcome" {
		t.Fatalf("UI() = %q, want %q", got, "zh-CN:ui:welcome")
	}
	if backend.lang != "zh-CN" {
		t.Fatalf("backend lang = %q, want zh-CN", backend.lang)
	}
	if backend.namespace != "ui" {
		t.Fatalf("backend namespace = %q, want ui", backend.namespace)
	}
	if backend.key != "welcome" {
		t.Fatalf("backend key = %q, want welcome", backend.key)
	}
	if backend.templateData["Name"] != "Codex" {
		t.Fatalf("backend template Name = %v, want Codex", backend.templateData["Name"])
	}
}

func freeLoopbackPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	port := portOf(t, listener.Addr())
	if err := listener.Close(); err != nil {
		t.Fatalf("listener.Close() error = %v", err)
	}
	return port
}

func portOf(t *testing.T, addr net.Addr) int {
	t.Helper()

	_, portString, err := net.SplitHostPort(addr.String())
	if err != nil {
		t.Fatalf("net.SplitHostPort(%q) error = %v", addr.String(), err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		t.Fatalf("strconv.Atoi(%q) error = %v", portString, err)
	}
	return port
}

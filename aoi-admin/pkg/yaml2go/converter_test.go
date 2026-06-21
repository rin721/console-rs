package yaml2go

// 本测试文件固定 YAML 到 Go 配置代码生成链路，防止注释补全和后续重构改变外部可观察行为。

import (
	"errors"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestConvertGeneratesMainAndSubConfigFiles 固定 YAML 到 Go 配置代码生成链路，确保后续注释补全或结构调整不改变该场景。
func TestConvertGeneratesMainAndSubConfigFiles(t *testing.T) {
	converter := New(&Config{
		PackageName: "config",
		Tags:        []string{"json", "yaml", "mapstructure"},
	})

	result, err := converter.Convert(`
server:
  host: localhost
  port: 8080
  enabled: true
`)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if result.PackageName != "config" {
		t.Fatalf("PackageName = %q, want %q", result.PackageName, "config")
	}
	if result.MainConfig == nil {
		t.Fatal("MainConfig = nil")
	}
	if result.MainConfig.FileName != "config.go" || result.MainConfig.StructName != "Config" {
		t.Fatalf("MainConfig = %#v, want config.go Config", result.MainConfig)
	}
	assertContainsAll(t, result.MainConfig.Content,
		"package config",
		"type Config struct",
		"Server *ServerConfig",
		"`json:\"server\" mapstructure:\"server\" yaml:\"server\"`",
	)
	assertValidGoFile(t, result.MainConfig.Content)

	if len(result.SubConfigs) != 1 {
		t.Fatalf("len(SubConfigs) = %d, want 1", len(result.SubConfigs))
	}
	sub := result.SubConfigs[0]
	if sub.FileName != "server_config.go" || sub.ConfigName != "server" || sub.StructName != "ServerConfig" {
		t.Fatalf("SubConfig = %#v, want server_config.go server ServerConfig", sub)
	}
	assertContainsAll(t, sub.Content,
		"package config",
		"type ServerConfig struct",
		"Host    string",
		"Port    int64",
		"Enabled bool",
		"`json:\"host\" mapstructure:\"host\" yaml:\"host\"`",
		"func (c *ServerConfig) ValidateName() string",
		"return \"server\"",
		"func (c *ServerConfig) Validate() error",
		"func (c *ServerConfig) DefaultConfig() *ServerConfig",
		"func (cfg *ServerConfig) OverrideConfig(prefix string)",
	)
	assertValidGoFile(t, sub.Content)
}

// TestConvertRejectsEmptyAndInvalidYAML 固定 YAML 到 Go 配置代码生成链路，确保后续注释补全或结构调整不改变该场景。
func TestConvertRejectsEmptyAndInvalidYAML(t *testing.T) {
	converter := New(nil)

	if _, err := converter.Convert("   \n\t"); !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("Convert(empty) error = %v, want ErrEmptyInput", err)
	}
	if _, err := converter.Convert("server: [unterminated"); !errors.Is(err, ErrInvalidYAML) {
		t.Fatalf("Convert(invalid) error = %v, want ErrInvalidYAML", err)
	}
}

// TestSetConfigValidatesConfig 固定 YAML 到 Go 配置代码生成链路，确保后续注释补全或结构调整不改变该场景。
func TestSetConfigValidatesConfig(t *testing.T) {
	converter := New(nil)

	if err := converter.SetConfig(nil); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("SetConfig(nil) error = %v, want ErrInvalidConfig", err)
	}
	if err := converter.SetConfig(&Config{IndentStyle: "invalid"}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("SetConfig(invalid indent) error = %v, want ErrInvalidConfig", err)
	}
	if err := converter.SetConfig(&Config{PackageName: "custom", IndentStyle: IndentStyleSpace}); err != nil {
		t.Fatalf("SetConfig(valid) error = %v", err)
	}

	result, err := converter.Convert("server:\n  host: localhost\n")
	if err != nil {
		t.Fatalf("Convert() after SetConfig error = %v", err)
	}
	if result.PackageName != "custom" {
		t.Fatalf("PackageName after SetConfig = %q, want %q", result.PackageName, "custom")
	}
}

// assertContainsAll 校验测试响应或状态中的关键字段，使测试断言聚焦在对外契约而非重复解析细节。
func assertContainsAll(t *testing.T, text string, substrings ...string) {
	t.Helper()
	for _, substring := range substrings {
		if !strings.Contains(text, substring) {
			t.Fatalf("generated content does not contain %q:\n%s", substring, text)
		}
	}
}

// assertValidGoFile 校验测试响应或状态中的关键字段，使测试断言聚焦在对外契约而非重复解析细节。
func assertValidGoFile(t *testing.T, content string) {
	t.Helper()
	if _, err := parser.ParseFile(token.NewFileSet(), "generated.go", content, parser.AllErrors); err != nil {
		t.Fatalf("generated content is not valid Go: %v\n%s", err, content)
	}
}

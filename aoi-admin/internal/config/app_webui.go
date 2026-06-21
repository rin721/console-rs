package config

import (
	"fmt"
	"strings"

	appconstants "github.com/rei0721/go-scaffold/types/constants"
)

const (
	DefaultWebUIMountPath = "/"
	DefaultWebUIDistDir   = "./web/app/build/client"
)

// WebUIConfig 控制内置 WebUI 静态产物的托管行为。
type WebUIConfig struct {
	Enabled       *bool  `mapstructure:"enabled" envname:"WEBUI_ENABLED" json:"enabled" yaml:"enabled" toml:"enabled"`
	MountPath     string `mapstructure:"mount_path" envname:"WEBUI_MOUNT_PATH" json:"mount_path" yaml:"mount_path" toml:"mount_path"`
	DistDir       string `mapstructure:"dist_dir" envname:"WEBUI_DIST_DIR" json:"dist_dir" yaml:"dist_dir" toml:"dist_dir"`
	PublicBaseURL string `mapstructure:"public_base_url" envname:"WEBUI_PUBLIC_BASE_URL" json:"public_base_url" yaml:"public_base_url" toml:"public_base_url"`
}

// ValidateName 返回 WebUI 配置分区名称。
func (c *WebUIConfig) ValidateName() string {
	return AppWebUIName
}

// ValidateRequired 声明 WebUI 配置为可选分区。
func (c *WebUIConfig) ValidateRequired() bool {
	return false
}

// Validate 校验 WebUI 静态挂载配置，文件存在性留给 HTTP 装配阶段降级处理。
func (c *WebUIConfig) Validate() error {
	c.ApplyDefaults()
	if !c.EnabledValue() {
		return nil
	}
	if c.MountPath == "" || !strings.HasPrefix(c.MountPath, "/") {
		return fmt.Errorf("mount_path must start with /")
	}
	if webUIReservedPath(c.MountPath) {
		return fmt.Errorf("mount_path conflicts with reserved API or probe path")
	}
	if strings.TrimSpace(c.DistDir) == "" {
		return fmt.Errorf("dist_dir is required")
	}
	c.PublicBaseURL = strings.TrimRight(strings.TrimSpace(c.PublicBaseURL), "/")
	return nil
}

// ApplyDefaults 补齐 WebUI 默认挂载位置和静态产物目录。
func (c *WebUIConfig) ApplyDefaults() {
	if c.MountPath == "" {
		c.MountPath = DefaultWebUIMountPath
	}
	c.MountPath = normalizeWebUIMountPath(c.MountPath)
	if c.DistDir == "" {
		c.DistDir = DefaultWebUIDistDir
	}
}

// EnabledValue 解析 WebUI 开关，nil 表示默认启用。
func (c WebUIConfig) EnabledValue() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// copyWebUIConfig 深拷贝 WebUIConfig 的指针字段，避免配置快照共享可变布尔值。
func copyWebUIConfig(src WebUIConfig) WebUIConfig {
	dst := src
	if src.Enabled != nil {
		value := *src.Enabled
		dst.Enabled = &value
	}
	return dst
}

func normalizeWebUIMountPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "/") {
		return value
	}
	return "/" + strings.Trim(strings.TrimRight(value, "/"), "/")
}

func webUIReservedPath(value string) bool {
	for _, reserved := range []string{
		appconstants.APIPathRoot,
		appconstants.APIBasePath,
		appconstants.HTTPHealthPath,
		appconstants.HTTPReadyPath,
	} {
		if value == reserved || strings.HasPrefix(value, reserved+"/") {
			return true
		}
	}
	return false
}

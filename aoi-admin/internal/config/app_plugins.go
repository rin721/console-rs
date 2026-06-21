package config

import (
	"fmt"
	"strings"
)

const (
	DefaultPluginBasePath                 = "/plugin-api/v1"
	DefaultPluginProtocolVersion          = "v1"
	DefaultPluginRequestTimeoutSeconds    = 10
	DefaultPluginHeartbeatTimeoutSeconds  = 30
	DefaultPluginLeaseTTLSeconds          = 30
	DefaultPluginLeaseScanIntervalSeconds = 15
	DefaultPluginRegistrationAuthMode     = "none"
	DefaultPluginRegistryBackend          = "db"
	DefaultPluginRouterStrategy           = "round_robin"
)

// PluginsConfig 定义远程插件宿主、协议入口和注册表相关配置。
//
// HTTPEnabled、WSEnabled、RPCEnabled 控制实际暴露的 transport；AllowedTransports 控制协议协商中允许的
// transport 名称，两者会在装配层共同决定插件可用入口。
type PluginsConfig struct {
	Enabled                  bool     `mapstructure:"enabled" envname:"PLUGINS_ENABLED" json:"enabled" yaml:"enabled" toml:"enabled"`
	BasePath                 string   `mapstructure:"base_path" envname:"PLUGINS_BASE_PATH" json:"base_path" yaml:"base_path" toml:"base_path"`
	DefaultProtocolVersion   string   `mapstructure:"default_protocol_version" envname:"PLUGINS_DEFAULT_PROTOCOL_VERSION" json:"default_protocol_version" yaml:"default_protocol_version" toml:"default_protocol_version"`
	AllowedTransports        []string `mapstructure:"allowed_transports" envname:"PLUGINS_ALLOWED_TRANSPORTS" json:"allowed_transports" yaml:"allowed_transports" toml:"allowed_transports"`
	NodeID                   string   `mapstructure:"node_id" envname:"PLUGINS_NODE_ID" json:"node_id" yaml:"node_id" toml:"node_id"`
	NodeAddress              string   `mapstructure:"node_address" envname:"PLUGINS_NODE_ADDRESS" json:"node_address" yaml:"node_address" toml:"node_address"`
	RegistryBackend          string   `mapstructure:"registry_backend" envname:"PLUGINS_REGISTRY_BACKEND" json:"registry_backend" yaml:"registry_backend" toml:"registry_backend"`
	RequestTimeoutSeconds    int      `mapstructure:"request_timeout_seconds" envname:"PLUGINS_REQUEST_TIMEOUT_SECONDS" json:"request_timeout_seconds" yaml:"request_timeout_seconds" toml:"request_timeout_seconds"`
	HeartbeatTimeoutSeconds  int      `mapstructure:"heartbeat_timeout_seconds" envname:"PLUGINS_HEARTBEAT_TIMEOUT_SECONDS" json:"heartbeat_timeout_seconds" yaml:"heartbeat_timeout_seconds" toml:"heartbeat_timeout_seconds"`
	LeaseTTLSeconds          int      `mapstructure:"lease_ttl_seconds" envname:"PLUGINS_LEASE_TTL_SECONDS" json:"lease_ttl_seconds" yaml:"lease_ttl_seconds" toml:"lease_ttl_seconds"`
	LeaseScanIntervalSeconds int      `mapstructure:"lease_scan_interval_seconds" envname:"PLUGINS_LEASE_SCAN_INTERVAL_SECONDS" json:"lease_scan_interval_seconds" yaml:"lease_scan_interval_seconds" toml:"lease_scan_interval_seconds"`
	RetryCount               int      `mapstructure:"retry_count" envname:"PLUGINS_RETRY_COUNT" json:"retry_count" yaml:"retry_count" toml:"retry_count"`
	RouterStrategy           string   `mapstructure:"router_strategy" envname:"PLUGINS_ROUTER_STRATEGY" json:"router_strategy" yaml:"router_strategy" toml:"router_strategy"`
	AllowedPermissions       []string `mapstructure:"allowed_permissions" envname:"PLUGINS_ALLOWED_PERMISSIONS" json:"allowed_permissions" yaml:"allowed_permissions" toml:"allowed_permissions"`
	RegistrationAuthMode     string   `mapstructure:"registration_auth_mode" envname:"PLUGINS_REGISTRATION_AUTH_MODE" json:"registration_auth_mode" yaml:"registration_auth_mode" toml:"registration_auth_mode"`
	SharedSecretEnv          string   `mapstructure:"shared_secret_env" envname:"PLUGINS_SHARED_SECRET_ENV" json:"shared_secret_env" yaml:"shared_secret_env" toml:"shared_secret_env"`
	WSEnabled                bool     `mapstructure:"ws_enabled" envname:"PLUGINS_WS_ENABLED" json:"ws_enabled" yaml:"ws_enabled" toml:"ws_enabled"`
	HTTPEnabled              bool     `mapstructure:"http_enabled" envname:"PLUGINS_HTTP_ENABLED" json:"http_enabled" yaml:"http_enabled" toml:"http_enabled"`
	RPCEnabled               bool     `mapstructure:"rpc_enabled" envname:"PLUGINS_RPC_ENABLED" json:"rpc_enabled" yaml:"rpc_enabled" toml:"rpc_enabled"`
	InjectionEnabled         bool     `mapstructure:"injection_enabled" envname:"PLUGINS_INJECTION_ENABLED" json:"injection_enabled" yaml:"injection_enabled" toml:"injection_enabled"`
}

func (c *PluginsConfig) ValidateName() string {
	return AppPluginsName
}

func (c *PluginsConfig) ValidateRequired() bool {
	return false
}

// Validate 校验插件配置在启用状态下是否足以启动宿主。
//
// 插件关闭时零值允许通过；插件启用时必须至少暴露一种 transport，并确保注册表、路由和鉴权模式可识别。
func (c *PluginsConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	c.ApplyDefaults()
	if !strings.HasPrefix(c.BasePath, "/") || c.BasePath == "/" {
		return fmt.Errorf("base_path must be an absolute non-root path")
	}
	if strings.TrimSpace(c.DefaultProtocolVersion) == "" {
		return fmt.Errorf("default_protocol_version must not be empty")
	}
	if c.RequestTimeoutSeconds <= 0 {
		return fmt.Errorf("request_timeout_seconds must be positive")
	}
	if c.HeartbeatTimeoutSeconds <= 0 {
		return fmt.Errorf("heartbeat_timeout_seconds must be positive")
	}
	if c.LeaseTTLSeconds <= 0 {
		return fmt.Errorf("lease_ttl_seconds must be positive")
	}
	if c.LeaseScanIntervalSeconds <= 0 {
		return fmt.Errorf("lease_scan_interval_seconds must be positive")
	}
	if c.RetryCount < 0 {
		return fmt.Errorf("retry_count must not be negative")
	}
	switch strings.ToLower(strings.TrimSpace(c.RegistryBackend)) {
	case "db", "memory":
	default:
		return fmt.Errorf("unsupported plugin registry_backend %q", c.RegistryBackend)
	}
	switch strings.ToLower(strings.TrimSpace(c.RouterStrategy)) {
	case "round_robin":
	default:
		return fmt.Errorf("unsupported plugin router_strategy %q", c.RouterStrategy)
	}
	if len(c.AllowedTransports) == 0 {
		return fmt.Errorf("allowed_transports must not be empty")
	}
	for _, transport := range c.AllowedTransports {
		switch strings.ToLower(strings.TrimSpace(transport)) {
		case "http", "websocket", "rpc":
		default:
			return fmt.Errorf("unsupported plugin transport %q", transport)
		}
	}
	switch strings.ToLower(strings.TrimSpace(c.RegistrationAuthMode)) {
	case "none", "shared_secret", "signature":
	default:
		return fmt.Errorf("unsupported registration_auth_mode %q", c.RegistrationAuthMode)
	}
	if strings.EqualFold(strings.TrimSpace(c.RegistrationAuthMode), "shared_secret") && strings.TrimSpace(c.SharedSecretEnv) == "" {
		return fmt.Errorf("shared_secret_env must be set when registration_auth_mode is shared_secret")
	}
	if !c.HTTPEnabled && !c.WSEnabled && !c.RPCEnabled {
		return fmt.Errorf("at least one plugin transport endpoint must be enabled")
	}
	return nil
}

// ApplyDefaults 补齐插件配置默认值并做基础归一化。
//
// 当插件启用但没有显式开启任何 transport 时，默认打开 HTTP 和 WebSocket，保持向后兼容；注入能力默认
// 跟随插件启用，避免宿主已启用但上下文 schema 缺失。
func (c *PluginsConfig) ApplyDefaults() {
	c.BasePath = strings.TrimRight(strings.TrimSpace(c.BasePath), "/")
	if c.BasePath == "" {
		c.BasePath = DefaultPluginBasePath
	}
	c.DefaultProtocolVersion = strings.TrimSpace(c.DefaultProtocolVersion)
	if c.DefaultProtocolVersion == "" {
		c.DefaultProtocolVersion = DefaultPluginProtocolVersion
	}
	if len(c.AllowedTransports) == 0 {
		c.AllowedTransports = []string{"http", "websocket"}
	}
	for i := range c.AllowedTransports {
		c.AllowedTransports[i] = strings.ToLower(strings.TrimSpace(c.AllowedTransports[i]))
	}
	if c.RequestTimeoutSeconds == 0 {
		c.RequestTimeoutSeconds = DefaultPluginRequestTimeoutSeconds
	}
	if c.HeartbeatTimeoutSeconds == 0 {
		c.HeartbeatTimeoutSeconds = DefaultPluginHeartbeatTimeoutSeconds
	}
	c.NodeID = strings.TrimSpace(c.NodeID)
	c.NodeAddress = strings.TrimSpace(c.NodeAddress)
	c.RegistryBackend = strings.ToLower(strings.TrimSpace(c.RegistryBackend))
	if c.RegistryBackend == "" && c.Enabled {
		c.RegistryBackend = DefaultPluginRegistryBackend
	}
	if c.RegistryBackend == "" {
		c.RegistryBackend = "memory"
	}
	if c.LeaseTTLSeconds == 0 {
		c.LeaseTTLSeconds = DefaultPluginLeaseTTLSeconds
	}
	if c.LeaseScanIntervalSeconds == 0 {
		c.LeaseScanIntervalSeconds = DefaultPluginLeaseScanIntervalSeconds
	}
	c.RouterStrategy = strings.ToLower(strings.TrimSpace(c.RouterStrategy))
	if c.RouterStrategy == "" {
		c.RouterStrategy = DefaultPluginRouterStrategy
	}
	for i := range c.AllowedPermissions {
		c.AllowedPermissions[i] = strings.TrimSpace(c.AllowedPermissions[i])
	}
	c.RegistrationAuthMode = strings.ToLower(strings.TrimSpace(c.RegistrationAuthMode))
	if c.RegistrationAuthMode == "" {
		c.RegistrationAuthMode = DefaultPluginRegistrationAuthMode
	}
	if c.Enabled && !c.HTTPEnabled && !c.WSEnabled && !c.RPCEnabled {
		c.HTTPEnabled = true
		c.WSEnabled = true
	}
	if c.Enabled && !c.InjectionEnabled {
		c.InjectionEnabled = true
	}
}

// copyPluginsConfig 深拷贝包含切片字段的插件配置，避免运行时更新污染旧快照。
func copyPluginsConfig(src PluginsConfig) PluginsConfig {
	dst := src
	dst.AllowedTransports = append([]string(nil), src.AllowedTransports...)
	dst.AllowedPermissions = append([]string(nil), src.AllowedPermissions...)
	return dst
}

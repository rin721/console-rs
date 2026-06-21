package plugin

import "github.com/rei0721/go-scaffold/internal/config"

// ConfigFromApp 将应用插件配置转换为项目插件模块内部配置。
//
// 这里会同时应用默认值并按 HTTP/WS/RPC 开关过滤 allowed transports，避免已关闭的 transport
// 继续出现在协议协商结果中。切片字段会复制，防止后续修改互相影响。
func ConfigFromApp(cfg config.PluginsConfig) Config {
	cfg.ApplyDefaults()
	transports := make([]string, 0, len(cfg.AllowedTransports))
	for _, transport := range cfg.AllowedTransports {
		switch transport {
		case "http":
			if cfg.HTTPEnabled {
				transports = append(transports, transport)
			}
		case "websocket":
			if cfg.WSEnabled {
				transports = append(transports, transport)
			}
		case "rpc":
			if cfg.RPCEnabled {
				transports = append(transports, transport)
			}
		default:
			transports = append(transports, transport)
		}
	}
	return Config{
		Enabled:                  cfg.Enabled,
		BasePath:                 cfg.BasePath,
		DefaultProtocolVersion:   cfg.DefaultProtocolVersion,
		AllowedTransports:        transports,
		NodeID:                   cfg.NodeID,
		NodeAddress:              cfg.NodeAddress,
		RegistryBackend:          cfg.RegistryBackend,
		RequestTimeoutSeconds:    cfg.RequestTimeoutSeconds,
		HeartbeatTimeoutSeconds:  cfg.HeartbeatTimeoutSeconds,
		LeaseTTLSeconds:          cfg.LeaseTTLSeconds,
		LeaseScanIntervalSeconds: cfg.LeaseScanIntervalSeconds,
		RetryCount:               cfg.RetryCount,
		RouterStrategy:           cfg.RouterStrategy,
		AllowedPermissions:       append([]string(nil), cfg.AllowedPermissions...),
		RegistrationAuthMode:     cfg.RegistrationAuthMode,
		SharedSecretEnv:          cfg.SharedSecretEnv,
		HTTPEnabled:              cfg.HTTPEnabled,
		WSEnabled:                cfg.WSEnabled,
		RPCEnabled:               cfg.RPCEnabled,
		InjectionEnabled:         cfg.InjectionEnabled,
	}
}

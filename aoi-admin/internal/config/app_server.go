package config

// 本文件定义一个配置分区及其校验规则，是外部配置进入运行时基础设施前的类型化边界。

import "errors"

// ServerConfig HTTP 服务器配置
// 控制 HTTP 服务的行为
type ServerConfig struct {
	// Host HTTP服务器地址
	// 例如: localhost, 127.0.0.1, db.example.com
	// SQLite 不需要此字段
	Host string `mapstructure:"host" envname:"SERVER_HOST"`

	// Port 监听端口
	// 有效范围: 1-65535
	// 常用端口: 8080, 3000, 80(需要 root)
	Port int `mapstructure:"port" envname:"SERVER_PORT"`

	// Mode 运行模式
	// 可选值:
	// - debug: 开发模式,详细日志,panic 堆栈
	// - release: 生产模式,性能优化,简化日志
	// - test: 测试模式
	// 影响:
	// - Gin 的日志详细程度
	// - 性能优化级别
	// - panic 恢复行为
	Mode string `mapstructure:"mode" envname:"SERVER_MODE"`

	// ReadTimeout 读取请求的超时时间(秒)
	// 从连接建立到读取完整请求体的最大时间
	// 防止慢速客户端占用连接
	// 推荐: 5-60 秒
	ReadTimeout int `mapstructure:"read_timeout" envname:"SERVER_READ_TIMEOUT"`

	// WriteTimeout 写入响应的超时时间(秒)
	// 从请求处理完成到写入完整响应的最大时间
	// 防止慢速客户端占用连接
	// 推荐: 10-120 秒(取决于响应大小)
	WriteTimeout int `mapstructure:"write_timeout" envname:"SERVER_WRITE_TIMEOUT"`

	// IdleTimeout 空闲连接的超时时间(秒)
	// 从连接建立到空闲的最大时间
	// 防止慢速客户端占用连接
	// 推荐: 60-300 秒
	IdleTimeout int `mapstructure:"idle_timeout" envname:"SERVER_IDLE_TIMEOUT"`
}

// ValidateName 返回当前配置分区在聚合校验错误中的稳定名称。
func (c *ServerConfig) ValidateName() string {
	return AppServerName
}

// ValidateRequired 声明当前配置分区是否必须出现在完整应用配置中。
func (c *ServerConfig) ValidateRequired() bool {
	return true
}

// Validate 验证服务器配置
// 实现 Configurable 接口
func (c *ServerConfig) Validate() error {
	// 验证端口范围
	if c.Port <= 0 || c.Port > 65535 {
		// 端口必须在 1-65535 之间
		// 1-1023 是保留端口,需要 root 权限
		// 1024-65535 是用户端口
		return errors.New("port must be between 1 and 65535")
	}

	// 验证运行模式
	if c.Mode != "debug" && c.Mode != "release" && c.Mode != "test" {
		// 只允许这三种模式
		return errors.New("mode must be debug, release, or test")
	}

	// 验证读取超时
	if c.ReadTimeout <= 0 {
		// 必须大于 0
		// 0 表示无超时,可能导致连接泄漏
		return errors.New("readTimeout must be positive")
	}

	// 验证写入超时
	if c.WriteTimeout <= 0 {
		// 必须大于 0
		return errors.New("writeTimeout must be positive")
	}

	return nil
}

// overrideServerConfig 使用环境变量覆盖服务器配置
func overrideServerConfig(cfg *ServerConfig) {
	overrideConfigFromEnv(cfg)
}

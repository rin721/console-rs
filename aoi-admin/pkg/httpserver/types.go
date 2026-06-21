package httpserver

// 本文件属于 HTTP 服务封装层，管理 net/http Server 的启动、关闭、地址选择和配置重载。

import (
	"context"
	"net/http"
	"time"
)

// HTTPServer HTTP 服务器接口
// 提供统一的 HTTP 服务器抽象，支持启动、关闭和配置热更新
type HTTPServer interface {
	// Start 启动 HTTP 服务器（非阻塞）
	// 在新的 goroutine 中启动服务器，立即返回
	// 如果服务器已在运行，返回错误
	// 参数:
	//   ctx: 上下文，用于控制启动过程
	// 返回:
	//   error: 启动失败时的错误
	Start(ctx context.Context) error

	// Wait 等待 HTTP 服务器运行期结果
	// Start 成功后调用会阻塞，直到异步 Serve 返回运行期错误、服务器优雅关闭或 context 取消。
	// 参数:
	//   ctx: 上下文，用于控制等待时间
	// 返回:
	//   error: 运行期错误或 context 错误；优雅关闭时返回 nil
	Wait(ctx context.Context) error

	// Shutdown 优雅关闭服务器
	// 停止接收新连接，等待现有请求处理完成
	// 或者直到 context 超时
	// 参数:
	//   ctx: 上下文，用于控制关闭超时
	// 返回:
	//   error: 关闭失败时的错误
	Shutdown(ctx context.Context) error

	// Reload 热重载配置（原子操作）
	// 当前实现会在必要时关闭旧监听再绑定新地址，调用方不应把它理解为严格零停机切换。
	// 参数:
	//   ctx: 上下文，用于控制重载过程
	//   cfg: 新的服务器配置
	// 返回:
	//   error: 重载失败时的错误
	Reload(ctx context.Context, cfg *Config) error
}

// Config HTTP 服务器配置
type Config struct {
	// Host 监听地址
	// 例如: "localhost", "0.0.0.0", "127.0.0.1"
	// 空字符串表示监听所有网络接口
	Host string

	// Port 监听端口
	// 范围: 1-65535
	// 0 表示随机分配一个可用端口
	Port int

	// ReadTimeout 读取请求的最大时间
	// 包括读取请求头和请求体
	// 防止慢速客户端长时间占用连接
	ReadTimeout time.Duration

	// WriteTimeout 写入响应的最大时间
	// 从请求处理完成到写入完整响应
	// 防止慢速客户端长时间占用连接
	WriteTimeout time.Duration

	// IdleTimeout 空闲连接的超时时间
	// Keep-Alive 连接的最大空闲时间
	// 超时后连接将被关闭
	IdleTimeout time.Duration
}

// Validate 验证配置是否有效
// 返回:
//
//	error: 配置无效时的错误信息
func (c *Config) Validate() error {
	// 端口范围验证
	if c.Port < 0 || c.Port > 65535 {
		return &ConfigError{
			Field:   "Port",
			Value:   c.Port,
			Message: "port must be between 0 and 65535",
		}
	}

	// 超时时间验证
	if c.ReadTimeout < 0 {
		return &ConfigError{
			Field:   "ReadTimeout",
			Value:   c.ReadTimeout,
			Message: "read timeout must be non-negative",
		}
	}

	if c.WriteTimeout < 0 {
		return &ConfigError{
			Field:   "WriteTimeout",
			Value:   c.WriteTimeout,
			Message: "write timeout must be non-negative",
		}
	}

	if c.IdleTimeout < 0 {
		return &ConfigError{
			Field:   "IdleTimeout",
			Value:   c.IdleTimeout,
			Message: "idle timeout must be non-negative",
		}
	}

	return nil
}

// ApplyDefaults 应用默认值到未设置的配置项
func (c *Config) ApplyDefaults() {
	if c.Host == "" {
		c.Host = DefaultHost
	}
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = DefaultReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = DefaultWriteTimeout
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = DefaultIdleTimeout
	}
}

// ConfigError 配置错误
type ConfigError struct {
	Field   string      // 错误的字段名
	Value   interface{} // 错误的值
	Message string      // 错误信息
}

// Error 实现 error 接口
func (e *ConfigError) Error() string {
	return "config error: " + e.Field + " = " + e.Message
}

// ServerError HTTP 服务器错误
type ServerError struct {
	Op      string // 操作名称 (start, shutdown, reload)
	Message string // 错误信息
	Err     error  // 底层错误
}

// Error 实现 error 接口
func (e *ServerError) Error() string {
	if e.Err != nil {
		return "httpserver: " + e.Op + ": " + e.Message + ": " + e.Err.Error()
	}
	return "httpserver: " + e.Op + ": " + e.Message
}

// Unwrap 返回底层错误
func (e *ServerError) Unwrap() error {
	return e.Err
}

// listenerInfo 监听器信息
// 用于追踪服务器监听的地址和端口
type listenerInfo struct {
	Addr string // 监听地址，格式: "host:port"
	// 未来可以扩展更多信息
}

// serverState 服务器状态
type serverState int

const (
	// stateStopped 服务器已停止
	stateStopped serverState = iota

	// stateStarting 服务器正在启动
	stateStarting

	// stateRunning 服务器正在运行
	stateRunning

	// stateStopping 服务器正在停止
	stateStopping
)

// String 返回状态的字符串表示
func (s serverState) String() string {
	switch s {
	case stateStopped:
		return "stopped"
	case stateStarting:
		return "starting"
	case stateRunning:
		return "running"
	case stateStopping:
		return "stopping"
	default:
		return "unknown"
	}
}

// Handler 定义了一个 HTTP 请求处理器类型
// 这是 net/http 包中 Handler 接口的别名
type Handler = http.Handler

// Option 定义配置选项函数类型
// 用于在创建 HTTPServer 时应用可选配置
// 采用函数选项模式(Functional Options Pattern)提高API扩展性
type Option func(*httpServer)

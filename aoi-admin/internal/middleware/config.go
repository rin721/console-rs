// Package middleware 提供应用程序的 HTTP 中间件组件
// 中间件在请求处理链中起到过滤、增强和监控的作用
package middleware

// 本文件定义 Gin 中间件能力，约束请求进入业务 handler 前后的链路上下文、副作用和错误输出。

// MiddlewareConfig 包含所有中间件组件的配置
// 这是一个聚合配置,统一管理所有中间件
// 好处:
// - 集中管理:所有中间件配置在一起
// - 便于传递:只需传递一个配置对象
// - 类型安全:使用结构体而不是 map
type MiddlewareConfig struct {
	// Recovery panic 恢复中间件配置
	// 负责捕获 panic 并返回 500 错误
	Recovery RecoveryConfig `mapstructure:"recovery"`

	// Logger 日志记录中间件配置
	// 负责记录所有 HTTP 请求
	Logger LoggerConfig `mapstructure:"logger"`

	// TraceID 请求追踪 ID 中间件配置
	// 负责为每个请求生成或提取 TraceID
	TraceID TraceIDConfig `mapstructure:"traceId"`

	// CORS 跨域资源共享中间件配置
	// 负责处理浏览器跨域请求
	CORS CORSConfig `mapstructure:"cors"`
}

// RecoveryConfig panic 恢复中间件的配置
// 这个中间件防止 panic 导致服务崩溃
type RecoveryConfig struct {
	// Enabled 是否启用 recovery 中间件
	// true: 启用,捕获 panic 并返回 500
	// false: 禁用,panic 会导致程序崩溃
	// 生产环境必须设置为 true
	// 测试环境可以设置为 false 以便 panic 直接暴露
	Enabled bool `mapstructure:"enabled"`
}

// LoggerConfig 日志记录中间件的配置
// 这个中间件记录所有 HTTP 请求的详细信息
type LoggerConfig struct {
	// Enabled 是否启用 logger 中间件
	// true: 记录所有请求日志
	// false: 不记录日志(测试环境可能禁用)
	Enabled bool `mapstructure:"enabled"`

	// SkipPaths 跳过日志记录的路径列表
	// 对于某些路径(如健康检查),记录日志没有意义
	// 跳过这些路径可以:
	// - 减少日志量
	// - 降低 I/O 压力
	// - 提高性能
	// 例如: []string{"/health", "/metrics", "/favicon.ico"}
	SkipPaths []string `mapstructure:"skipPaths"`
}

// TraceIDConfig 请求追踪 ID 中间件的配置
// 这个中间件为每个请求生成唯一的追踪 ID
type TraceIDConfig struct {
	// Enabled 是否启用 trace ID 中间件
	// true: 为每个请求生成/提取 TraceID
	// false: 不处理 TraceID
	// 强烈建议启用,TraceID 对问题追踪非常重要
	Enabled bool `mapstructure:"enabled"`

	// HeaderName TraceID 的 HTTP header 名称
	// 默认: "X-Request-ID"
	// 这个 header 用于:
	// - 从客户端接收现有的 TraceID(如果有)
	// - 在响应中返回 TraceID 给客户端
	// 可以自定义为其他名称,如:
	// - "X-Trace-ID"
	// - "X-Correlation-ID"
	// - "Request-ID"
	HeaderName string `mapstructure:"headerName"`
}

// DefaultMiddlewareConfig 返回一个使用合理默认值的中间件配置
// 这些默认值适合大多数应用场景
// 返回:
//
//	MiddlewareConfig: 默认配置
//
// 默认行为:
//   - Recovery: 启用(生产环境必需)
//   - Logger: 启用,跳过 /health(减少日志量)
//   - TraceID: 启用,使用 X-Request-ID header
func DefaultMiddlewareConfig() MiddlewareConfig {
	return MiddlewareConfig{
		Recovery: RecoveryConfig{
			Enabled: true, // 必须启用,防止 panic 导致服务崩溃
		},
		Logger: LoggerConfig{
			Enabled: true, // 启用请求日志
			// 跳过健康检查路径
			// 健康检查频率很高(每秒多次),记录日志意义不大
			SkipPaths: []string{"/health"},
		},
		TraceID: TraceIDConfig{
			Enabled:    true,           // 启用 TraceID
			HeaderName: "X-Request-ID", // 使用标准的 header 名称
		},
	}
}

// CORSConfig 跨域资源共享中间件的配置
// 这个中间件处理浏览器的跨域请求
type CORSConfig struct {
	// Enabled 是否启用 CORS 中间件
	// true: 启用跨域支持
	// false: 禁用(所有跨域请求将被浏览器阻止)
	// 开发环境通常启用,生产环境根据需求决定
	Enabled bool `mapstructure:"enabled"`

	// AllowOrigins 允许的源列表
	// 指定哪些域名可以跨域访问
	// 格式:
	//   - 精确匹配: "http://localhost:3000"
	//   - 通配符: "*" (允许所有源,不安全,仅开发环境使用)
	// 示例: []string{"http://localhost:3000", "https://example.com"}
	AllowOrigins []string `mapstructure:"allowOrigins"`

	// AllowMethods 允许的 HTTP 方法
	// 指定跨域请求允许使用的 HTTP 方法
	// 常用方法: GET, POST, PUT, DELETE, PATCH, OPTIONS
	AllowMethods []string `mapstructure:"allowMethods"`

	// AllowHeaders 允许的请求头
	// 指定跨域请求允许携带的自定义请求头
	AllowHeaders []string `mapstructure:"allowHeaders"`

	// ExposeHeaders 暴露给浏览器的响应头
	// 默认情况下浏览器只能访问简单响应头
	// 通过此配置可以让浏览器访问自定义响应头
	ExposeHeaders []string `mapstructure:"exposeHeaders"`

	// AllowCredentials 是否允许携带凭证
	// true: 允许跨域请求携带 Cookie、HTTP Auth 等
	// false: 不允许携带凭证
	AllowCredentials bool `mapstructure:"allowCredentials"`

	// MaxAge 预检请求缓存时间(秒)
	// 浏览器会缓存 OPTIONS 预检请求的结果
	MaxAge int `mapstructure:"maxAge"`
}

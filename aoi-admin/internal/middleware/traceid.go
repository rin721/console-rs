package middleware

// 本文件定义 Gin 中间件能力，约束请求进入业务 handler 前后的链路上下文、副作用和错误输出。

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/rei0721/go-scaffold/internal/ports"
)

// TraceIDKey 是存储追踪 ID 的上下文键
// 用于在中间件和处理器之间传递 TraceID
const TraceIDKey = "traceId"

// DefaultHeaderName 是 TraceID 的默认 HTTP header 名称
// 使用 X- 前缀表示这是自定义 header
// X-Request-ID 是业界常用的请求 ID header 名称
const DefaultHeaderName = "X-Request-ID"

// traceIDGenerator TraceID 生成器
// 使用包级别变量,在整个应用中复用同一个生成器
// 这样可以确保生成的 ID 在单个实例中是唯一的
// TraceID 返回一个生成或提取 TraceID 的中间件
// TraceID 用于请求链路追踪,是分布式系统可观测性的重要组成部分
// 功能:
// 1. 从请求 header 中提取现有的 TraceID(如果有)
// 2. 如果没有,生成新的 TraceID
// 3. 将 TraceID 存储在上下文中,供后续使用
// 4. 将 TraceID 添加到响应 header 中返回给客户端
// 使用场景:
// - 微服务架构中跨服务追踪请求
// - 关联同一请求在不同组件中的日志
// - 客户端报告问题时提供追踪标识
// 注意:这个中间件应该最先注册,确保所有后续中间件都能访问 TraceID
func TraceID(cfg TraceIDConfig, generator ports.IDGenerator) ports.HTTPHandlerFunc {
	// 确定 header 名称
	// 允许通过配置自定义,提供灵活性
	headerName := cfg.HeaderName
	if headerName == "" {
		// 如果未配置,使用默认值
		headerName = DefaultHeaderName
	}

	return func(c ports.HTTPContext) {
		// 检查中间件是否启用
		// 在某些场景下可能需要禁用(如简单的健康检查)
		if !cfg.Enabled {
			c.Next()
			return
		}

		// 1. 尝试从请求 header 中提取 TraceID
		// 客户端可能已经提供了 TraceID(例如从上游服务传递下来)
		// 保留现有的 TraceID 可以实现端到端的请求追踪
		traceID := c.GetHeader(headerName)

		// 2. 如果 header 中没有 TraceID,生成新的
		// 使用 Snowflake 算法生成,保证:
		// - 全局唯一性(在当前节点)
		// - 时间有序性(可以通过 ID 判断请求时间)
		// - 高性能(无需访问数据库或外部服务)
		if traceID == "" {
			traceID = nextTraceID(generator)
		}

		// 3. 将 TraceID 存储在 Gin 上下文中
		// 后续的中间件和处理器可以通过 c.Get(TraceIDKey) 获取
		// 这样就不需要在每个函数中传递 TraceID 参数
		c.Set(TraceIDKey, traceID)

		// 4. 将 TraceID 添加到响应 header 中
		// 好处:
		// - 客户端可以获取 TraceID,用于问题报告
		// - 前端可以将 TraceID 显示在错误页面上
		// - 调试时可以在浏览器开发工具中看到
		c.Header(headerName, traceID)

		// 继续执行后续中间件和处理器
		// 它们都可以通过 GetTraceID(c) 获取 TraceID
		c.Next()
	}
}

func nextTraceID(generator ports.IDGenerator) string {
	if generator != nil {
		return generator.NextIDString()
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "trace-unavailable"
	}
	return hex.EncodeToString(raw[:])
}

// GetTraceID 从 Gin 上下文中获取 TraceID
// 这是一个便捷函数,封装了类型断言的细节
// 参数:
//
//	c: Gin 上下文
//
// 返回:
//
//	string: TraceID,如果不存在或类型错误返回空字符串
//
// 使用场景:
// - 在处理器中获取 TraceID 并添加到日志
// - 在中间件中获取 TraceID 并传递给下游服务
// - 在错误处理中包含 TraceID
func GetTraceID(c ports.HTTPContext) string {
	// 从上下文中获取值
	// c.Get() 返回 (interface{}, bool)
	if traceID, exists := c.Get(TraceIDKey); exists {
		// 类型断言,将 interface{} 转换为 string
		// 使用 ok 模式避免 panic
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	// 如果不存在或类型不匹配,返回空字符串
	// 这是一个安全的默认值
	return ""
}

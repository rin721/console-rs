package middleware

// 本文件定义 Gin 中间件能力，约束请求进入业务 handler 前后的链路上下文、副作用和错误输出。

import (
	"net/http"

	"github.com/rei0721/go-scaffold/internal/ports"
	"github.com/rei0721/go-scaffold/types/errors"
	"github.com/rei0721/go-scaffold/types/result"
)

// Recovery 返回一个从 panic 中恢复的中间件
// 当处理器发生 panic 时,捕获错误并返回 500 状态码
// 这是防止服务崩溃的最后一道防线
// 设计考虑:
// - 捕获所有未处理的 panic,保证服务持续运行
// - 记录详细的错误日志,包含 TraceID 便于问题追踪
// - 返回统一的错误格式,避免暴露内部实现细节
// 使用场景:
// - 处理意外的运行时错误(nil 指针、数组越界等)
// - 防止第三方库的 panic 导致整个服务崩溃
// - 在生产环境中必须使用,确保服务的高可用性
func Recovery(cfg RecoveryConfig, log ports.Logger) ports.HTTPHandlerFunc {
	return func(c ports.HTTPContext) {
		// 检查中间件是否启用
		// 在测试环境可能需要禁用以便 panic 能够直接暴露
		if !cfg.Enabled {
			c.Next()
			return
		}

		// 使用 defer + recover 捕获 panic
		// defer 确保即使发生 panic 也会执行这段代码
		// recover() 只在 defer 函数中有效,用于捕获 panic
		defer func() {
			if err := recover(); err != nil {
				// 从上下文中获取 TraceID
				// TraceID 对于追踪分布式系统中的错误至关重要
				// 可以将同一个请求在不同组件中的日志关联起来
				traceID := GetTraceID(c)

				// 记录 panic 详情到日志
				// 这是排查问题的关键信息:
				// - error: panic 的原因(可能是字符串或 error 类型)
				// - path: 发生错误的请求路径
				// - method: HTTP 方法
				// - traceId: 请求追踪 ID
				if log != nil {
					log.Error("panic recovered",
						"error", err,
						"path", c.Path(),
						"method", c.Method(),
						"traceId", traceID,
					)
				}

				// 返回 500 错误给客户端
				// AbortWithStatusJSON 会立即返回响应并停止后续中间件执行
				// 使用统一的错误格式,包含:
				// - HTTP 状态码: 500 (Internal Server Error)
				// - 错误码: errors.ErrInternalServer
				// - 错误消息: "internal server error"(不暴露内部细节)
				// - TraceID: 便于客户端报告问题时提供追踪信息
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					result.LocalizedError(c, errors.ErrInternalServer, result.MessageKeyInternalError, nil, traceID),
				)
			}
		}()

		// 继续执行后续的中间件和处理器
		// 如果这里发生 panic,会被上面的 defer 捕获
		c.Next()
	}
}

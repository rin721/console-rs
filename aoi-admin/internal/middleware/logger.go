package middleware

// 本文件定义 Gin 中间件能力，约束请求进入业务 handler 前后的链路上下文、副作用和错误输出。

import (
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/internal/ports"
)

// Logger 返回一个记录请求信息的中间件
// 它会记录每个 HTTP 请求的方法、路径、状态码和处理时长
// 这对于监控应用性能和排查问题至关重要
func Logger(cfg LoggerConfig, log ports.Logger) ports.HTTPHandlerFunc {
	// 构建跳过路径的映射表,实现 O(1) 时间复杂度的查找
	// 使用 map 而不是遍历切片,可以显著提高性能
	// 特别是当跳过路径列表较长时(例如健康检查、监控端点等)
	skipPaths := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		// 将每个跳过的路径标记为 true
		skipPaths[path] = true
	}

	// 返回 Gin 中间件处理函数
	// 这个函数会在每个请求处理前后被调用
	return func(c ports.HTTPContext) {
		// 检查是否启用了日志中间件
		// 如果未启用,直接调用 c.Next() 跳过日志记录
		// 这个设计允许在配置中动态开关日志功能
		if !cfg.Enabled {
			c.Next()
			return
		}

		// 检查当前请求路径是否应该跳过日志记录
		path := c.Path()
		if skipPaths[path] {
			// 对于健康检查等频繁请求,跳过日志可以:
			// - 减少日志文件大小
			// - 降低 I/O 压力
			// - 提高整体性能
			c.Next()
			return
		}

		// 记录请求开始时间
		// 用于后续计算请求处理耗时
		start := time.Now()

		// 调用链中的下一个中间件或处理函数
		// 这是 Gin 中间件链的核心机制
		// c.Next() 之前的代码在请求处理前执行
		// c.Next() 之后的代码在请求处理后执行
		c.Next()

		// 计算请求处理的总耗时
		// time.Since(start) 返回从 start 到现在的时间差
		// 这个指标对于性能监控和优化非常重要
		duration := time.Since(start)

		// 从上下文中获取 TraceID (追踪ID)
		// TraceID 用于关联同一个请求在不同服务/组件中的日志
		// 这在微服务架构和问题排查时特别有用
		traceID := GetTraceID(c)

		// 记录请求详细信息
		// 使用结构化日志,便于日志分析和监控系统解析
		log.Info("request completed",
			"method", c.Method(), // HTTP 方法(GET/POST/PUT等)
			"path", path, // 请求路径
			"status", c.Status(), // HTTP 响应状态码
			"duration", duration.String(), // 耗时的字符串表示(如 "123ms")
			"durationMs", duration.Milliseconds(), // 耗时的毫秒数,便于监控系统计算
			"clientIP", c.ClientIP(), // 路由框架识别的客户端 IP
			"realIP", ClientIPRealIP(c), // 代理头解析后的真实客户端 IP
			"traceId", traceID, // 追踪 ID,用于请求链路追踪
		)
	}
}

func ClientIPRealIP(c ports.HTTPContext) string {
	for _, candidate := range []string{
		forwardedForHeader(c.GetHeader("Forwarded")),
		firstForwardedIP(c.GetHeader("X-Forwarded-For")),
		c.GetHeader("X-Real-IP"),
		c.GetHeader("CF-Connecting-IP"),
		c.GetHeader("True-Client-IP"),
		c.ClientIP(),
		remoteAddrIP(c),
	} {
		if ip := normalizeIP(candidate); ip != "" {
			return ip
		}
	}
	return ""
}

func firstForwardedIP(value string) string {
	for _, part := range strings.Split(value, ",") {
		if normalizeIP(part) != "" {
			return part
		}
	}
	return ""
}

func forwardedForHeader(value string) string {
	for _, element := range strings.Split(value, ",") {
		for _, part := range strings.Split(element, ";") {
			key, val, ok := strings.Cut(strings.TrimSpace(part), "=")
			if !ok || !strings.EqualFold(key, "for") {
				continue
			}
			if normalizeIP(val) != "" {
				return val
			}
		}
	}
	return ""
}

func remoteAddrIP(c ports.HTTPContext) string {
	req := c.Request()
	if req == nil {
		return ""
	}
	return req.RemoteAddr
}

func normalizeIP(value string) string {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if value == "" || strings.EqualFold(value, "unknown") {
		return ""
	}
	if ip, err := netip.ParseAddr(value); err == nil {
		return ip.String()
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		if ip, err := netip.ParseAddr(strings.Trim(host, "[]")); err == nil {
			return ip.String()
		}
	}
	if ip, err := netip.ParseAddr(strings.Trim(value, "[]")); err == nil {
		return ip.String()
	}
	return ""
}

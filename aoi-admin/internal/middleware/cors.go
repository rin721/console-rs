package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/rei0721/go-scaffold/internal/ports"
)

// CORSMiddleware 返回基于内部 HTTP 端口的 CORS 中间件。
// 只有白名单命中的 Origin 会被回显，避免在允许凭证时错误返回通配符。
func CORSMiddleware(cfg CORSConfig) ports.HTTPHandlerFunc {
	return func(c ports.HTTPContext) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		if origin != "" && originAllowed(origin, cfg.AllowOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			if cfg.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		}
		if len(cfg.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(cfg.ExposeHeaders, ", "))
		}

		if c.Method() == http.MethodOptions {
			if len(cfg.AllowMethods) > 0 {
				c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))
			}
			if len(cfg.AllowHeaders) > 0 {
				c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))
			}
			if cfg.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
			}
			c.Data(http.StatusNoContent, "text/plain; charset=utf-8", nil)
			return
		}

		c.Next()
	}
}

// originAllowed 判断请求 Origin 是否命中允许列表。
// "*" 只作为显式配置的开发/测试通配符处理。
func originAllowed(origin string, allowed []string) bool {
	for _, item := range allowed {
		item = strings.TrimSpace(item)
		if item == "*" || item == origin {
			return true
		}
	}
	return false
}

package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/rei0721/go-scaffold/internal/ports"
	apperrors "github.com/rei0721/go-scaffold/types/errors"
	"github.com/rei0721/go-scaffold/types/result"
)

// RateLimitConfig 控制进程内固定窗口限流，适合保护公开认证入口。
type RateLimitConfig struct {
	Enabled bool
	Limit   int
	Window  time.Duration
}

type rateLimitWindow struct {
	Count   int
	ResetAt time.Time
}

// RateLimit 使用客户端 IP、方法和路径作为限流键。
// 这是进程内固定窗口实现，适合保护登录、注册等公开入口；多实例部署需要在网关或共享存储层补充限流。
func RateLimit(cfg RateLimitConfig) ports.HTTPHandlerFunc {
	if !cfg.Enabled {
		return func(c ports.HTTPContext) {
			c.Next()
		}
	}
	if cfg.Limit <= 0 {
		cfg.Limit = 20
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}

	var mu sync.Mutex
	windows := map[string]rateLimitWindow{}

	return func(c ports.HTTPContext) {
		now := time.Now()
		key := c.ClientIP() + "|" + c.Method() + "|" + c.Path()

		mu.Lock()
		window := windows[key]
		if window.ResetAt.IsZero() || now.After(window.ResetAt) {
			window = rateLimitWindow{ResetAt: now.Add(cfg.Window)}
		}
		window.Count++
		windows[key] = window
		allowed := window.Count <= cfg.Limit
		retryAfter := int(time.Until(window.ResetAt).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		mu.Unlock()

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, result.LocalizedError(c, apperrors.ErrBusinessLogic, result.MessageKeyRateLimited, nil, GetTraceID(c)))
			return
		}
		c.Next()
	}
}

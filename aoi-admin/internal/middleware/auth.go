package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
	"github.com/rei0721/go-scaffold/internal/ports"
	apperrors "github.com/rei0721/go-scaffold/types/errors"
	"github.com/rei0721/go-scaffold/types/result"
)

const PrincipalKey = "principal"

// Authenticator 抽象 token 到 Principal 的认证能力。
// 中间件只依赖该端口，不关心 token 是 JWT、API token 还是其他实现。
type Authenticator interface {
	AuthenticateToken(context.Context, string) (iamservice.Principal, error)
}

// Authorizer 抽象 Principal 对 object/action 的授权判断。
type Authorizer interface {
	Authorize(context.Context, iamservice.Principal, iamservice.PermissionContext) (bool, error)
}

// Auth 从 Bearer header 或配置化访问 Cookie 中解析 token，并把认证后的 Principal 写入请求上下文。
// 后续权限中间件和 handler 都通过 PrincipalKey 读取当前调用主体。
type AuthConfig struct {
	AccessCookieName string
}

type CSRFConfig struct {
	Enabled    bool
	CookieName string
	HeaderName string
}

func Auth(authenticator Authenticator, configs ...AuthConfig) ports.HTTPHandlerFunc {
	cfg := AuthConfig{}
	if len(configs) > 0 {
		cfg = configs[0]
	}
	return func(c ports.HTTPContext) {
		if authenticator == nil {
			abort(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "api.auth.unavailable")
			return
		}
		token := requestToken(c, cfg)
		if token == "" {
			abort(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "api.auth.missingBearerToken")
			return
		}
		principal, err := authenticator.AuthenticateToken(c.RequestContext(), token)
		if err != nil {
			abort(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, result.MessageKeyUnauthorized)
			return
		}
		c.Set(PrincipalKey, principal)
		c.Next()
	}
}

// RequirePermission 在进入业务 handler 前校验当前主体是否拥有指定权限。
// obj 和 act 对应服务层权限模型中的 object/action，next 只会在授权成功后执行。
func CSRF(cfg CSRFConfig) ports.HTTPHandlerFunc {
	return func(c ports.HTTPContext) {
		if !cfg.Enabled || isSafeHTTPMethod(c.Method()) || bearerToken(c.GetHeader("Authorization")) != "" {
			return
		}
		cookieName := strings.TrimSpace(cfg.CookieName)
		headerName := strings.TrimSpace(cfg.HeaderName)
		if cookieName == "" || headerName == "" {
			abort(c, http.StatusForbidden, apperrors.ErrPermissionDenied, result.MessageKeyForbidden)
			return
		}
		cookieValue, err := c.Cookie(cookieName)
		if err != nil || strings.TrimSpace(cookieValue) == "" {
			abort(c, http.StatusForbidden, apperrors.ErrPermissionDenied, result.MessageKeyForbidden)
			return
		}
		if constantTimeEqual(cookieValue, c.GetHeader(headerName)) {
			return
		}
		abort(c, http.StatusForbidden, apperrors.ErrPermissionDenied, result.MessageKeyForbidden)
	}
}

func RequirePermission(authorizer Authorizer, permission iamservice.PermissionContext, next ports.HTTPHandlerFunc) ports.HTTPHandlerFunc {
	return func(c ports.HTTPContext) {
		if authorizer == nil {
			abort(c, http.StatusForbidden, apperrors.ErrPermissionDenied, "api.auth.authorizationUnavailable")
			return
		}
		principal, ok := GetPrincipal(c)
		if !ok {
			abort(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "api.auth.missingPrincipal")
			return
		}
		allowed, err := authorizer.Authorize(c.RequestContext(), principal, permission)
		if err != nil || !allowed {
			abort(c, http.StatusForbidden, apperrors.ErrPermissionDenied, result.MessageKeyForbidden)
			return
		}
		next(c)
	}
}

// RequireOrgParam 校验路径参数中的组织 ID 与 Principal.OrgID 一致。
// 该检查用于组织作用域路由，防止用户修改 URL 访问其他租户资源。
func RequireOrgParam(param string, next ports.HTTPHandlerFunc) ports.HTTPHandlerFunc {
	return func(c ports.HTTPContext) {
		principal, ok := GetPrincipal(c)
		if !ok {
			abort(c, http.StatusUnauthorized, apperrors.ErrUnauthorized, "api.auth.missingPrincipal")
			return
		}
		orgID, err := strconv.ParseInt(c.Param(param), 10, 64)
		if err != nil || orgID != principal.OrgID {
			abort(c, http.StatusForbidden, apperrors.ErrPermissionDenied, result.MessageKeyForbidden)
			return
		}
		next(c)
	}
}

// GetPrincipal 从请求上下文读取认证主体。
func GetPrincipal(c ports.HTTPContext) (iamservice.Principal, bool) {
	value, ok := c.Get(PrincipalKey)
	if !ok {
		return iamservice.Principal{}, false
	}
	principal, ok := value.(iamservice.Principal)
	return principal, ok
}

// bearerToken 从 Authorization header 提取 Bearer token。
func requestToken(c ports.HTTPContext, cfg AuthConfig) string {
	if token := bearerToken(c.GetHeader("Authorization")); token != "" {
		return token
	}
	cookieName := strings.TrimSpace(cfg.AccessCookieName)
	if cookieName == "" {
		return ""
	}
	token, err := c.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(token)
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func isSafeHTTPMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET", "HEAD", "OPTIONS", "TRACE":
		return true
	default:
		return false
	}
}

func constantTimeEqual(left string, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || len(left) != len(right) {
		return false
	}
	var diff byte
	for i := 0; i < len(left); i++ {
		diff |= left[i] ^ right[i]
	}
	return diff == 0
}

// abort 按统一结果格式终止请求，并附带当前 TraceID。
func abort(c ports.HTTPContext, status int, code int, message string) {
	c.AbortWithStatusJSON(status, result.LocalizedError(c, code, message, nil, GetTraceID(c)))
}

// Package security 定义远程插件认证和 scope 授权的通用抽象。
package security

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

var (
	// ErrUnauthorized 表示认证或授权失败；ErrUnsupportedAuth 表示当前认证模式尚未实现。
	ErrUnauthorized    = errors.New("plugin request unauthorized")
	ErrUnsupportedAuth = errors.New("unsupported plugin authentication")
)

// Operation 描述一次需要认证或授权的插件协议操作。
type Operation struct {
	Name       string
	PluginID   string
	InstanceID string
}

// Principal 表示认证后的调用主体。
//
// Scopes 来自插件注册或认证信息，AuthMode 记录认证方式，Metadata 预留给审计和扩展策略使用。
type Principal struct {
	PluginID   string
	InstanceID string
	Scopes     []string
	AuthMode   string
	Metadata   map[string]string
}

// PermissionRequest 描述一次授权裁决所需的上下文。
//
// Permissions 是本次操作声明需要的 scope；Capability 和 Event 用于让实现按能力或事件进一步收敛权限。
type PermissionRequest struct {
	Operation   Operation
	Permissions []string
	Capability  string
	Event       string
	Context     map[string]string
}

// Decision 是 Authorizer 的裁决结果。
//
// Audit 用于把裁决来源和时间传给上层观测系统，而不是在 security 包内直接写日志或数据库。
type Decision struct {
	Allowed bool
	Reason  string
	Audit   protocol.AuditInfo
}

// Authenticator 负责把协议认证信息转换为 Principal。
type Authenticator interface {
	Authenticate(context.Context, Operation, *protocol.Auth) (Principal, error)
}

// Authorizer 负责判断 Principal 是否允许执行某次插件操作。
type Authorizer interface {
	Authorize(context.Context, Principal, PermissionRequest) (Decision, error)
}

// AuthenticatorFunc 让普通函数可以作为 Authenticator 使用。
type AuthenticatorFunc func(context.Context, Operation, *protocol.Auth) (Principal, error)

// Authenticate 调用底层函数实现认证逻辑。
func (f AuthenticatorFunc) Authenticate(ctx context.Context, op Operation, auth *protocol.Auth) (Principal, error) {
	return f(ctx, op, auth)
}

// AuthorizerFunc 让普通函数可以作为 Authorizer 使用。
type AuthorizerFunc func(context.Context, Principal, PermissionRequest) (Decision, error)

// Authorize 调用底层函数实现授权逻辑。
func (f AuthorizerFunc) Authorize(ctx context.Context, principal Principal, req PermissionRequest) (Decision, error) {
	return f(ctx, principal, req)
}

// SharedSecretAuthenticator 实现当前内置的插件共享密钥认证。
//
// Mode 为 none 时用于受信任环境或测试；signature 会显式返回未支持，避免调用方误以为签名已生效。
type SharedSecretAuthenticator struct {
	Secret string
	Mode   string
}

// Authenticate 校验 protocol.Auth 并返回插件主体。
//
// op 提供 plugin_id/instance_id 回填 Principal；auth 为空时只有 none 模式会放行。
func (a SharedSecretAuthenticator) Authenticate(_ context.Context, op Operation, auth *protocol.Auth) (Principal, error) {
	mode := strings.ToLower(strings.TrimSpace(a.Mode))
	switch mode {
	case "", "none":
		return Principal{PluginID: op.PluginID, InstanceID: op.InstanceID, AuthMode: "none"}, nil
	case "shared_secret":
		if strings.TrimSpace(a.Secret) == "" {
			return Principal{}, ErrUnauthorized
		}
		if auth == nil || strings.TrimSpace(auth.Token) != strings.TrimSpace(a.Secret) {
			return Principal{}, ErrUnauthorized
		}
		if authMode := strings.ToLower(strings.TrimSpace(auth.Mode)); authMode != "" && authMode != "shared_secret" {
			return Principal{}, ErrUnauthorized
		}
		return Principal{PluginID: op.PluginID, InstanceID: op.InstanceID, AuthMode: "shared_secret"}, nil
	case "signature":
		return Principal{}, ErrUnsupportedAuth
	default:
		return Principal{}, ErrUnsupportedAuth
	}
}

// ScopeAuthorizer 基于允许列表裁决插件声明的权限 scope。
//
// AllowedPermissions 为空表示不限制 scope；非空时，请求中的每个权限都必须在允许列表中。
type ScopeAuthorizer struct {
	AllowedPermissions []string
	Source             string
	Now                func() time.Time
}

// Authorize 校验请求声明的权限是否落在允许列表内。
//
// 返回的 Decision 始终包含审计时间和来源，方便 Host 记录统一的插件安全事件。
func (a ScopeAuthorizer) Authorize(_ context.Context, principal Principal, req PermissionRequest) (Decision, error) {
	now := time.Now().UTC()
	if a.Now != nil {
		now = a.Now().UTC()
	}
	allowed := map[string]bool{}
	for _, permission := range a.AllowedPermissions {
		allowed[strings.TrimSpace(permission)] = true
	}
	for _, permission := range req.Permissions {
		permission = strings.TrimSpace(permission)
		if permission == "" {
			continue
		}
		if len(allowed) > 0 && !allowed[permission] {
			return Decision{
				Allowed: false,
				Reason:  "permission not allowed",
				Audit:   protocol.AuditInfo{GeneratedAt: now, Source: firstNonEmpty(a.Source, "plugin-security")},
			}, ErrUnauthorized
		}
	}
	return Decision{
		Allowed: true,
		Audit:   protocol.AuditInfo{GeneratedAt: now, Source: firstNonEmpty(a.Source, "plugin-security")},
	}, nil
}

// AllowAll 返回一个无条件放行的 Authorizer。
//
// 该实现适合开发、测试或已由外层网关保证安全的部署，不应误认为生产默认策略。
func AllowAll() Authorizer {
	return AuthorizerFunc(func(_ context.Context, _ Principal, _ PermissionRequest) (Decision, error) {
		return Decision{Allowed: true, Audit: protocol.AuditInfo{GeneratedAt: time.Now().UTC(), Source: "plugin-security"}}, nil
	})
}

// firstNonEmpty 返回第一个非空字符串，用于补齐审计来源等可选字段。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

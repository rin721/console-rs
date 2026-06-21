package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/rei0721/go-scaffold/internal/ports"
	pluginpkg "github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
	"github.com/rei0721/go-scaffold/pkg/plugin/security"
	pluginhttp "github.com/rei0721/go-scaffold/pkg/plugin/transport/http"
	pluginrpc "github.com/rei0721/go-scaffold/pkg/plugin/transport/rpc"
)

const pluginSecretHeader = "X-Plugin-Secret"

// authenticator 将项目配置中的插件注册鉴权策略适配到 HTTP、WebSocket 和 RPC 三种传输。
type authenticator struct {
	verifier security.Authenticator
}

// rpcRegistry 将应用层 RPC 注册表适配为 pkg/plugin RPC transport 需要的注册接口。
type rpcRegistry struct {
	inner ports.RPCRegistry
}

// newAuthenticator 从插件配置中构造共享密钥鉴权器。
//
// SharedSecretEnv 只保存环境变量名，真实 secret 在启动时读取，避免配置文件直接携带注册密钥。
func newAuthenticator(cfg Config) authenticator {
	mode := strings.ToLower(strings.TrimSpace(cfg.RegistrationAuthMode))
	secret := ""
	if strings.TrimSpace(cfg.SharedSecretEnv) != "" {
		secret = os.Getenv(strings.TrimSpace(cfg.SharedSecretEnv))
	}
	return authenticator{
		verifier: security.SharedSecretAuthenticator{
			Mode:   mode,
			Secret: secret,
		},
	}
}

// HTTP 处理插件 HTTP 协议的鉴权。
//
// 为兼容简单客户端，缺少协议体 auth 时会读取 X-Plugin-Secret 请求头作为 shared_secret token。
func (a authenticator) HTTP(c pluginhttp.Context, operation string, auth *protocol.Auth) error {
	if auth == nil && c != nil {
		if token := strings.TrimSpace(c.GetHeader(pluginSecretHeader)); token != "" {
			auth = &protocol.Auth{Mode: "shared_secret", Token: token}
		}
	}
	return a.authorize(c.RequestContext(), operation, auth)
}

// WS 处理插件 WebSocket envelope 的鉴权。
func (a authenticator) WS(ctx context.Context, operation string, auth *protocol.Auth) error {
	return a.authorize(ctx, operation, auth)
}

// RPC 处理插件 JSON-RPC 协议的鉴权。
func (a authenticator) RPC(ctx context.Context, operation string, auth *protocol.Auth) error {
	return a.authorize(ctx, operation, auth)
}

// authorize 统一执行插件操作鉴权并转换为插件协议错误。
//
// pkg/security 的错误被映射成 pkg/plugin 错误，保证不同 transport 返回一致的协议错误码。
func (a authenticator) authorize(ctx context.Context, operation string, auth *protocol.Auth) error {
	if a.verifier == nil {
		return nil
	}
	_, err := a.verifier.Authenticate(ctx, security.Operation{Name: operation}, auth)
	if errors.Is(err, security.ErrUnauthorized) {
		return pluginpkg.ErrUnauthorized
	}
	if errors.Is(err, security.ErrUnsupportedAuth) {
		return pluginpkg.ErrUnsupportedAuth
	}
	return err
}

// Register 注册 JSON-RPC 方法，并把插件 transport 错误转换为应用 RPC 错误。
func (r rpcRegistry) Register(method string, handler pluginrpc.HandlerFunc) error {
	return r.inner.Register(method, func(ctx context.Context, params json.RawMessage) (any, error) {
		result, err := handler(ctx, params)
		if err != nil {
			return nil, pluginRPCError(err)
		}
		return result, nil
	})
}

// pluginRPCError 将插件协议错误映射到 JSON-RPC code 和可机器读取的插件错误码。
//
// 映射保持保守：参数/协议/鉴权类错误使用更具体 code，其余错误落到 server error 范围。
func pluginRPCError(err error) error {
	code := -32000
	switch {
	case errors.Is(err, pluginpkg.ErrInvalidPlugin), errors.Is(err, pluginpkg.ErrInvalidCapability):
		code = -32602
	case errors.Is(err, pluginpkg.ErrUnsupportedProtocol), errors.Is(err, pluginpkg.ErrUnsupportedAuth):
		code = -32601
	case errors.Is(err, pluginpkg.ErrUnauthorized):
		code = -32001
	}
	return &ports.RPCError{
		Code:    code,
		Message: err.Error(),
		Data: map[string]string{
			"code": pluginrpc.ErrorCode(err),
		},
	}
}

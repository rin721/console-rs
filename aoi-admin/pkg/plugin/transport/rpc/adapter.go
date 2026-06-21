// Package rpctransport 实现远程插件协议的 JSON-RPC transport adapter。
package rpctransport

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

// JSON-RPC 方法名与插件协议操作一一对应，调用端和服务端必须保持稳定。
const (
	MethodNegotiateProtocol = "plugin.negotiate"
	MethodRegister          = "plugin.register"
	MethodHeartbeat         = "plugin.heartbeat"
	MethodRenewLease        = "plugin.renewLease"
	MethodUnregister        = "plugin.unregister"
	MethodHealthCheck       = "plugin.healthCheck"
	MethodListCapabilities  = "plugin.listCapabilities"
	MethodInvoke            = "plugin.invoke"
	MethodPushEvent         = "plugin.pushEvent"
	MethodSubscribeEvent    = "plugin.subscribeEvent"
	MethodInjectContext     = "plugin.injectContext"
	MethodGetInjectedSchema = "plugin.getInjectedSchema"
	MethodReportStatus      = "plugin.reportStatus"
	MethodSyncMetadata      = "plugin.syncMetadata"
	MethodDrain             = "plugin.drain"
)

// HandlerFunc 是 JSON-RPC registry 接收的通用处理函数签名。
type HandlerFunc func(context.Context, json.RawMessage) (any, error)

// Registry 是 JSON-RPC 方法注册表的最小抽象。
type Registry interface {
	Register(string, HandlerFunc) error
}

// Host 是 RPC adapter 调用插件宿主所需的协议方法集合。
type Host interface {
	Register(context.Context, protocol.RegisterRequest) (protocol.RegisterResponse, error)
	Heartbeat(context.Context, protocol.HeartbeatRequest) (protocol.HeartbeatResponse, error)
	RenewLease(context.Context, protocol.RenewLeaseRequest) (protocol.RenewLeaseResponse, error)
	Unregister(context.Context, protocol.UnregisterRequest) (protocol.UnregisterResponse, error)
	Health(context.Context, string) (protocol.HealthStatus, error)
	ListCapabilities(context.Context, protocol.ListCapabilitiesRequest) (protocol.ListCapabilitiesResponse, error)
	Invoke(context.Context, protocol.InvokeRequest) (protocol.InvokeResponse, error)
	PushEvent(context.Context, protocol.PushEventRequest) (protocol.PushEventResponse, error)
	SubscribeEvent(context.Context, protocol.SubscribeEventRequest) (protocol.SubscribeEventResponse, error)
	InjectContext(context.Context, protocol.InjectContextRequest) (protocol.InjectContextResponse, error)
	GetInjectedSchema(context.Context, protocol.GetInjectedSchemaRequest) (protocol.GetInjectedSchemaResponse, error)
	NegotiateProtocol(context.Context, protocol.NegotiateProtocolRequest) (protocol.NegotiateProtocolResponse, error)
	ReportStatus(context.Context, protocol.ReportStatusRequest) (protocol.ReportStatusResponse, error)
	SyncMetadata(context.Context, protocol.SyncMetadataRequest) (protocol.SyncMetadataResponse, error)
	Drain(context.Context, protocol.DrainRequest) (protocol.DrainResponse, error)
}

// AuthFunc 在 RPC 方法分发到 Host 前执行认证。
type AuthFunc func(context.Context, string, *protocol.Auth) error

// Option 配置 RPC adapter。
type Option func(*adapter)

// adapter 持有 Host 和可选认证函数。
type adapter struct {
	host Host
	auth AuthFunc
}

// Register 将插件协议方法批量注册到 JSON-RPC registry。
// registry 负责真正的 JSON-RPC server 生命周期；host 负责执行业务协议。
func Register(registry Registry, host Host, options ...Option) error {
	a := &adapter{host: host}
	for _, option := range options {
		if option != nil {
			option(a)
		}
	}
	for _, route := range []struct {
		method  string
		handler HandlerFunc
	}{
		{MethodNegotiateProtocol, a.negotiate},
		{MethodRegister, a.register},
		{MethodHeartbeat, a.heartbeat},
		{MethodRenewLease, a.renewLease},
		{MethodUnregister, a.unregister},
		{MethodHealthCheck, a.healthCheck},
		{MethodListCapabilities, a.listCapabilities},
		{MethodInvoke, a.invoke},
		{MethodPushEvent, a.pushEvent},
		{MethodSubscribeEvent, a.subscribeEvent},
		{MethodInjectContext, a.injectContext},
		{MethodGetInjectedSchema, a.getInjectedSchema},
		{MethodReportStatus, a.reportStatus},
		{MethodSyncMetadata, a.syncMetadata},
		{MethodDrain, a.drain},
	} {
		if err := registry.Register(route.method, route.handler); err != nil {
			return err
		}
	}
	return nil
}

// WithAuth 注入 RPC transport 认证钩子。
func WithAuth(fn AuthFunc) Option {
	return func(a *adapter) {
		a.auth = fn
	}
}

func (a *adapter) negotiate(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.NegotiateProtocolRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationNegotiateProtocol, req.Auth); err != nil {
		return nil, err
	}
	return a.host.NegotiateProtocol(ctx, req)
}

func (a *adapter) register(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.RegisterRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationRegister, req.Auth); err != nil {
		return nil, err
	}
	return a.host.Register(ctx, req)
}

func (a *adapter) heartbeat(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.HeartbeatRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationHeartbeat, req.Auth); err != nil {
		return nil, err
	}
	return a.host.Heartbeat(ctx, req)
}

func (a *adapter) renewLease(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.RenewLeaseRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationRenewLease, req.Auth); err != nil {
		return nil, err
	}
	return a.host.RenewLease(ctx, req)
}

func (a *adapter) unregister(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.UnregisterRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationUnregister, req.Auth); err != nil {
		return nil, err
	}
	return a.host.Unregister(ctx, req)
}

// healthCheck 支持实例级健康查询的可选扩展。
// Host 没有实现 HealthInstance 时回退到插件级健康状态，保持旧实现兼容。
func (a *adapter) healthCheck(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.HealthCheckRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationHealthCheck, req.Auth); err != nil {
		return nil, err
	}
	if req.InstanceID != "" {
		if exact, ok := a.host.(interface {
			HealthInstance(context.Context, string, string) (protocol.HealthStatus, error)
		}); ok {
			return exact.HealthInstance(ctx, req.PluginID, req.InstanceID)
		}
	}
	return a.host.Health(ctx, req.PluginID)
}

func (a *adapter) listCapabilities(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.ListCapabilitiesRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationListCapabilities, req.Auth); err != nil {
		return nil, err
	}
	return a.host.ListCapabilities(ctx, req)
}

func (a *adapter) invoke(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.InvokeRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationInvoke, req.Auth); err != nil {
		return nil, err
	}
	return a.host.Invoke(ctx, req)
}

func (a *adapter) pushEvent(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.PushEventRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationPushEvent, req.Auth); err != nil {
		return nil, err
	}
	return a.host.PushEvent(ctx, req)
}

func (a *adapter) subscribeEvent(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.SubscribeEventRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationSubscribeEvent, req.Auth); err != nil {
		return nil, err
	}
	return a.host.SubscribeEvent(ctx, req)
}

func (a *adapter) injectContext(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.InjectContextRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationInjectContext, req.Auth); err != nil {
		return nil, err
	}
	return a.host.InjectContext(ctx, req)
}

func (a *adapter) getInjectedSchema(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.GetInjectedSchemaRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationGetInjectedSchema, req.Auth); err != nil {
		return nil, err
	}
	return a.host.GetInjectedSchema(ctx, req)
}

func (a *adapter) reportStatus(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.ReportStatusRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationReportStatus, req.Auth); err != nil {
		return nil, err
	}
	return a.host.ReportStatus(ctx, req)
}

func (a *adapter) syncMetadata(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.SyncMetadataRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationSyncMetadata, req.Auth); err != nil {
		return nil, err
	}
	return a.host.SyncMetadata(ctx, req)
}

func (a *adapter) drain(ctx context.Context, params json.RawMessage) (any, error) {
	var req protocol.DrainRequest
	if err := decode(params, &req); err != nil {
		return nil, err
	}
	if err := a.authorize(ctx, protocol.OperationDrain, req.Auth); err != nil {
		return nil, err
	}
	return a.host.Drain(ctx, req)
}

// authorize 执行可选 RPC transport 认证。
// 没有配置认证函数时，信任边界由部署环境或上层 JSON-RPC server 承担。
func (a *adapter) authorize(ctx context.Context, operation string, auth *protocol.Auth) error {
	if a == nil || a.auth == nil {
		return nil
	}
	return a.auth(ctx, operation, auth)
}

// decode 将 JSON-RPC params 解码为具体协议请求。
// 空 params 和 null 被视为 {}，方便无额外参数的协议方法保持兼容。
func decode(params json.RawMessage, target any) error {
	if len(params) == 0 || string(params) == "null" {
		params = []byte(`{}`)
	}
	if err := json.Unmarshal(params, target); err != nil {
		return plugin.ErrInvalidPlugin
	}
	return nil
}

// ErrorCode 将插件域错误映射为 JSON-RPC 响应中的协议错误码。
// 未识别错误统一归入 internal，避免向远端泄漏内部错误分类细节。
func ErrorCode(err error) string {
	code := protocol.ErrorCodeInternal
	switch {
	case errors.Is(err, plugin.ErrDisabled):
		code = protocol.ErrorCodeDisabled
	case errors.Is(err, plugin.ErrPluginNotFound):
		code = protocol.ErrorCodePluginNotFound
	case errors.Is(err, plugin.ErrCapabilityNotFound):
		code = protocol.ErrorCodeCapabilityNotFound
	case errors.Is(err, plugin.ErrPluginOffline):
		code = protocol.ErrorCodePluginOffline
	case errors.Is(err, plugin.ErrProviderUnavailable):
		code = protocol.ErrorCodeProviderUnavailable
	case errors.Is(err, plugin.ErrTransportUnavailable):
		code = protocol.ErrorCodeTransportUnavailable
	case errors.Is(err, plugin.ErrInvalidPlugin):
		code = protocol.ErrorCodeInvalidPlugin
	case errors.Is(err, plugin.ErrInvalidCapability):
		code = protocol.ErrorCodeInvalidCapability
	case errors.Is(err, plugin.ErrUnsupportedProtocol):
		code = protocol.ErrorCodeUnsupportedProtocol
	case errors.Is(err, plugin.ErrUnauthorized):
		code = protocol.ErrorCodeUnauthorized
	case errors.Is(err, plugin.ErrUnsupportedAuth):
		code = protocol.ErrorCodeUnsupportedAuth
	}
	return code
}

// Package ws 实现远程插件协议的 WebSocket transport adapter。
package ws

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

// Host 是 WebSocket dispatcher 依赖的插件宿主能力集合。
//
// 该接口只描述协议操作需要调用的用例，避免 transport 层直接依赖 pkg/plugin.Host 的具体结构。
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

// Dispatcher 将 WebSocket envelope 分发到 Host，并把执行结果编码回协议响应。
//
// Dispatcher 同时持有会话注册表，使主系统可以通过已绑定的 WebSocket 连接反向调用远程插件。
type Dispatcher struct {
	host     Host
	auth     AuthFunc
	sessions *SessionRegistry
}

// AuthFunc 在 envelope 分发前执行 transport 侧认证。
//
// operation 用于让调用方按协议动作区分权限；auth 来自请求 payload，允许插件通过共享密钥等方式证明身份。
type AuthFunc func(context.Context, string, *protocol.Auth) error

// Option 配置 Dispatcher 的可选行为。
type Option func(*Dispatcher)

// NewDispatcher 创建 WebSocket 协议分发器。
//
// host 为 nil 时后续分发会在调用具体方法时失败；调用方通常在组合根保证 Host 已启用并完成装配。
func NewDispatcher(host Host, options ...Option) *Dispatcher {
	dispatcher := &Dispatcher{
		host:     host,
		sessions: NewSessionRegistry(),
	}
	for _, option := range options {
		if option != nil {
			option(dispatcher)
		}
	}
	return dispatcher
}

// WithAuth 注入 envelope 级认证函数。
func WithAuth(fn AuthFunc) Option {
	return func(dispatcher *Dispatcher) {
		dispatcher.auth = fn
	}
}

// RemoteInvoker 返回基于当前 WebSocket 会话注册表的反向调用器。
//
// 返回值为 nil 表示 Dispatcher 本身不可用；调用器只负责 transport 发送，不重新选择插件实例。
func (d *Dispatcher) RemoteInvoker() *RemoteInvoker {
	if d == nil {
		return nil
	}
	return NewRemoteInvoker(d.sessions)
}

// Dispatch 处理单个协议 envelope 并构造响应 envelope。
//
// 返回值始终是 response 类型；业务错误会被转换为 protocol.Error，避免 transport 层泄漏 Go error 结构。
func (d *Dispatcher) Dispatch(ctx context.Context, envelope protocol.Envelope) protocol.Envelope {
	out := protocol.Envelope{
		ID:        envelope.ID,
		Type:      "response",
		Operation: envelope.Operation,
		Version:   envelope.Version,
	}
	payload, err := d.dispatch(ctx, envelope)
	if err != nil {
		out.Error = protocolError(err)
		return out
	}
	out.Payload = payload
	return out
}

// dispatch 将 envelope payload 解码为具体请求，并调用对应 Host 方法。
//
// 每个分支先 decode 再 authorize，确保认证逻辑拿到结构化 auth 字段，同时把 payload 格式错误统一映射为协议错误。
func (d *Dispatcher) dispatch(ctx context.Context, envelope protocol.Envelope) (json.RawMessage, error) {
	switch envelope.Operation {
	case protocol.OperationNegotiateProtocol:
		var req protocol.NegotiateProtocolRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		return encode(d.host.NegotiateProtocol(ctx, req))
	case protocol.OperationRegister:
		var req protocol.RegisterRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		return encode(d.host.Register(ctx, req))
	case protocol.OperationHeartbeat:
		var req protocol.HeartbeatRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		return encode(d.host.Heartbeat(ctx, req))
	case protocol.OperationRenewLease:
		var req protocol.RenewLeaseRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		return encode(d.host.RenewLease(ctx, req))
	case protocol.OperationUnregister:
		var req protocol.UnregisterRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		return encode(d.host.Unregister(ctx, req))
	case protocol.OperationHealthCheck:
		var req protocol.HealthCheckRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		// 兼容 Host 尚未实现实例级健康检查的部署；没有 instance_id 时仍使用插件级汇总健康状态。
		if req.InstanceID != "" {
			if exact, ok := d.host.(interface {
				HealthInstance(context.Context, string, string) (protocol.HealthStatus, error)
			}); ok {
				return encode(exact.HealthInstance(ctx, req.PluginID, req.InstanceID))
			}
		}
		return encode(d.host.Health(ctx, req.PluginID))
	case protocol.OperationListCapabilities:
		var req protocol.ListCapabilitiesRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		return encode(d.host.ListCapabilities(ctx, req))
	case protocol.OperationInvoke:
		var req protocol.InvokeRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		return encode(d.host.Invoke(ctx, req))
	case protocol.OperationPushEvent:
		var req protocol.PushEventRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		return encode(d.host.PushEvent(ctx, req))
	case protocol.OperationSubscribeEvent:
		var req protocol.SubscribeEventRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		return encode(d.host.SubscribeEvent(ctx, req))
	case protocol.OperationInjectContext:
		var req protocol.InjectContextRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		return encode(d.host.InjectContext(ctx, req))
	case protocol.OperationGetInjectedSchema:
		var req protocol.GetInjectedSchemaRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		return encode(d.host.GetInjectedSchema(ctx, req))
	case protocol.OperationReportStatus:
		var req protocol.ReportStatusRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		return encode(d.host.ReportStatus(ctx, req))
	case protocol.OperationSyncMetadata:
		var req protocol.SyncMetadataRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		return encode(d.host.SyncMetadata(ctx, req))
	case protocol.OperationDrain:
		var req protocol.DrainRequest
		if err := decode(envelope.Payload, &req); err != nil {
			return nil, err
		}
		if err := d.authorize(ctx, envelope.Operation, req.Auth); err != nil {
			return nil, err
		}
		return encode(d.host.Drain(ctx, req))
	default:
		return nil, plugin.ErrUnsupportedProtocol
	}
}

// authorize 执行可选认证钩子。
//
// 未配置 AuthFunc 时直接放行，便于测试和受信任内网部署复用同一 dispatcher。
func (d *Dispatcher) authorize(ctx context.Context, operation string, auth *protocol.Auth) error {
	if d == nil || d.auth == nil {
		return nil
	}
	return d.auth(ctx, operation, auth)
}

// decode 将可空 payload 解码为目标请求结构。
//
// 空 payload 被视为 `{}`，让不带参数的协议操作可以复用同一 envelope 编码。
func decode(payload json.RawMessage, target any) error {
	if len(payload) == 0 || string(payload) == "null" {
		payload = []byte(`{}`)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return plugin.ErrInvalidPlugin
	}
	return nil
}

// encode 将 Host 返回值序列化为 envelope payload。
//
// Host 错误原样返回给上层转换为 protocol.Error，序列化错误保留为内部错误。
func encode(value any, err error) (json.RawMessage, error) {
	if err != nil {
		return nil, err
	}
	raw, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		return nil, marshalErr
	}
	return raw, nil
}

// protocolError 将内部错误归一化为跨语言协议错误码。
//
// 这里保留原始错误消息用于排查，但错误码必须来自 protocol 包，方便外部插件稳定处理。
func protocolError(err error) *protocol.Error {
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
	return &protocol.Error{Code: code, Message: err.Error()}
}

// Package httptransport 实现远程插件协议的 HTTP transport adapter。
package httptransport

import (
	"context"
	"errors"
	"net/http"

	"github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

// Context 抽象 HTTP 框架上下文需要提供的最小能力。
// 这样 adapter 可以接入 Gin 等框架，同时在测试中使用轻量 fake。
type Context interface {
	RequestContext() context.Context
	BindJSON(any) error
	JSON(int, any)
	GetHeader(string) string
}

// Host 是 HTTP adapter 调用插件宿主所需的协议方法集合。
// 该接口刻意只描述 transport 入口需要的行为，避免 handler 依赖完整 Host 结构。
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

// Handler 将 HTTP 请求转换为插件协议请求并委托给 Host。
// auth 为可选钩子，会在业务方法执行前根据 operation 和 payload 中的 Auth 做认证。
type Handler struct {
	host Host
	auth AuthFunc
}

// AuthFunc 是 HTTP transport 层的认证钩子。
type AuthFunc func(Context, string, *protocol.Auth) error

// Option 配置 Handler 的可选依赖。
type Option func(*Handler)

// Response 是 HTTP transport 的统一响应 envelope。
// 成功时写入 Data，失败时写入协议错误，保持跨语言插件端解析稳定。
type Response struct {
	Data  any             `json:"data,omitempty"`
	Error *protocol.Error `json:"error,omitempty"`
}

// New 创建 HTTP transport handler。
func New(host Host, options ...Option) *Handler {
	handler := &Handler{host: host}
	for _, option := range options {
		if option != nil {
			option(handler)
		}
	}
	return handler
}

// WithAuth 注入认证钩子。
// fn 为空时 handler 会跳过 transport 层认证，由上层部署环境承担信任边界。
func WithAuth(fn AuthFunc) Option {
	return func(handler *Handler) {
		handler.auth = fn
	}
}

// Negotiate 处理插件注册前的协议版本和 transport 协商请求。
func (h *Handler) Negotiate(c Context) {
	var req protocol.NegotiateProtocolRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationNegotiateProtocol, req.Auth) {
		return
	}
	data, err := h.host.NegotiateProtocol(c.RequestContext(), req)
	h.write(c, data, err)
}

// Register 处理远程插件实例注册请求。
func (h *Handler) Register(c Context) {
	var req protocol.RegisterRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationRegister, req.Auth) {
		return
	}
	data, err := h.host.Register(c.RequestContext(), req)
	h.write(c, data, err)
}

// Heartbeat 处理旧版心跳请求，并由 Host 映射为租约续期。
func (h *Handler) Heartbeat(c Context) {
	var req protocol.HeartbeatRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationHeartbeat, req.Auth) {
		return
	}
	data, err := h.host.Heartbeat(c.RequestContext(), req)
	h.write(c, data, err)
}

// RenewLease 处理插件实例租约续期请求。
func (h *Handler) RenewLease(c Context) {
	var req protocol.RenewLeaseRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationRenewLease, req.Auth) {
		return
	}
	data, err := h.host.RenewLease(c.RequestContext(), req)
	h.write(c, data, err)
}

// Unregister 处理插件实例注销请求。
func (h *Handler) Unregister(c Context) {
	var req protocol.UnregisterRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationUnregister, req.Auth) {
		return
	}
	data, err := h.host.Unregister(c.RequestContext(), req)
	h.write(c, data, err)
}

// HealthCheck 返回插件或具体实例的健康状态。
// Host 支持实例级健康检查时优先调用精确接口，否则回退到插件级状态。
func (h *Handler) HealthCheck(c Context) {
	var req protocol.HealthCheckRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationHealthCheck, req.Auth) {
		return
	}
	var data protocol.HealthStatus
	var err error
	if req.InstanceID != "" {
		if exact, ok := h.host.(interface {
			HealthInstance(context.Context, string, string) (protocol.HealthStatus, error)
		}); ok {
			data, err = exact.HealthInstance(c.RequestContext(), req.PluginID, req.InstanceID)
		} else {
			data, err = h.host.Health(c.RequestContext(), req.PluginID)
		}
	} else {
		data, err = h.host.Health(c.RequestContext(), req.PluginID)
	}
	h.write(c, data, err)
}

// ListCapabilities 返回本地或远程插件暴露的能力列表。
func (h *Handler) ListCapabilities(c Context) {
	var req protocol.ListCapabilitiesRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationListCapabilities, req.Auth) {
		return
	}
	data, err := h.host.ListCapabilities(c.RequestContext(), req)
	h.write(c, data, err)
}

// Invoke 处理能力调用请求，并把 Host 返回的 result 包在统一响应 envelope 中。
func (h *Handler) Invoke(c Context) {
	var req protocol.InvokeRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationInvoke, req.Auth) {
		return
	}
	data, err := h.host.Invoke(c.RequestContext(), req)
	h.write(c, data, err)
}

// PushEvent 处理事件推送请求。
func (h *Handler) PushEvent(c Context) {
	var req protocol.PushEventRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationPushEvent, req.Auth) {
		return
	}
	data, err := h.host.PushEvent(c.RequestContext(), req)
	h.write(c, data, err)
}

// SubscribeEvent 处理远程插件事件订阅请求。
func (h *Handler) SubscribeEvent(c Context) {
	var req protocol.SubscribeEventRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationSubscribeEvent, req.Auth) {
		return
	}
	data, err := h.host.SubscribeEvent(c.RequestContext(), req)
	h.write(c, data, err)
}

// InjectContext 处理插件上下文注入请求。
func (h *Handler) InjectContext(c Context) {
	var req protocol.InjectContextRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationInjectContext, req.Auth) {
		return
	}
	data, err := h.host.InjectContext(c.RequestContext(), req)
	h.write(c, data, err)
}

// ReportStatus 接收插件运行状态上报。
func (h *Handler) ReportStatus(c Context) {
	var req protocol.ReportStatusRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationReportStatus, req.Auth) {
		return
	}
	data, err := h.host.ReportStatus(c.RequestContext(), req)
	h.write(c, data, err)
}

// SyncMetadata 处理插件元数据同步请求。
func (h *Handler) SyncMetadata(c Context) {
	var req protocol.SyncMetadataRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationSyncMetadata, req.Auth) {
		return
	}
	data, err := h.host.SyncMetadata(c.RequestContext(), req)
	h.write(c, data, err)
}

// Drain 将插件实例切换到 draining 状态。
func (h *Handler) Drain(c Context) {
	var req protocol.DrainRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationDrain, req.Auth) {
		return
	}
	data, err := h.host.Drain(c.RequestContext(), req)
	h.write(c, data, err)
}

// GetInjectedSchema 返回当前可注入上下文的 schema。
func (h *Handler) GetInjectedSchema(c Context) {
	var req protocol.GetInjectedSchemaRequest
	if !bind(c, &req) {
		return
	}
	if !h.authorize(c, protocol.OperationGetInjectedSchema, req.Auth) {
		return
	}
	data, err := h.host.GetInjectedSchema(c.RequestContext(), req)
	h.write(c, data, err)
}

// authorize 执行可选认证钩子并在失败时直接写 HTTP 响应。
// 返回 false 表示 handler 已完成响应，调用方不应继续执行 Host 方法。
func (h *Handler) authorize(c Context, operation string, auth *protocol.Auth) bool {
	if h == nil || h.auth == nil {
		return true
	}
	if err := h.auth(c, operation, auth); err != nil {
		c.JSON(statusCode(err), Response{Error: protocolError(err)})
		return false
	}
	return true
}

// write 将 Host 返回值转换为 HTTP transport 的统一响应格式。
func (h *Handler) write(c Context, data any, err error) {
	if err != nil {
		c.JSON(statusCode(err), Response{Error: protocolError(err)})
		return
	}
	c.JSON(http.StatusOK, Response{Data: data})
}

// bind 解码请求 JSON；失败时统一返回 invalid plugin 错误码。
func bind(c Context, target any) bool {
	if err := c.BindJSON(target); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: &protocol.Error{
			Code:    protocol.ErrorCodeInvalidPlugin,
			Message: "invalid json payload",
		}})
		return false
	}
	return true
}

// statusCode 将插件域错误映射为 HTTP 状态码。
// 不可用、离线等可恢复错误使用 409，未知错误按上游网关失败处理。
func statusCode(err error) int {
	switch {
	case errors.Is(err, plugin.ErrDisabled):
		return http.StatusNotFound
	case errors.Is(err, plugin.ErrPluginNotFound), errors.Is(err, plugin.ErrCapabilityNotFound):
		return http.StatusNotFound
	case errors.Is(err, plugin.ErrPluginOffline), errors.Is(err, plugin.ErrProviderUnavailable), errors.Is(err, plugin.ErrTransportUnavailable):
		return http.StatusConflict
	case errors.Is(err, plugin.ErrInvalidPlugin), errors.Is(err, plugin.ErrInvalidCapability), errors.Is(err, plugin.ErrUnsupportedProtocol):
		return http.StatusBadRequest
	case errors.Is(err, plugin.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, plugin.ErrUnsupportedAuth):
		return http.StatusBadRequest
	default:
		return http.StatusBadGateway
	}
}

// protocolError 将 Go 错误转换为跨 transport 的协议错误码。
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

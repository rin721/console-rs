package plugin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/rei0721/go-scaffold/internal/ports"
	pluginpkg "github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
	pluginhttp "github.com/rei0721/go-scaffold/pkg/plugin/transport/http"
	pluginrpc "github.com/rei0721/go-scaffold/pkg/plugin/transport/rpc"
	pluginws "github.com/rei0721/go-scaffold/pkg/plugin/transport/ws"
)

// Config 是项目插件模块使用的运行期配置快照。
//
// 该结构由 internal/config 转换而来，避免协议层直接依赖应用完整配置对象。
type Config struct {
	Enabled                  bool
	BasePath                 string
	DefaultProtocolVersion   string
	AllowedTransports        []string
	RequestTimeoutSeconds    int
	HeartbeatTimeoutSeconds  int
	RegistrationAuthMode     string
	SharedSecretEnv          string
	HTTPEnabled              bool
	WSEnabled                bool
	RPCEnabled               bool
	InjectionEnabled         bool
	NodeID                   string
	NodeAddress              string
	RegistryBackend          string
	LeaseTTLSeconds          int
	LeaseScanIntervalSeconds int
	RetryCount               int
	RouterStrategy           string
	AllowedPermissions       []string
}

// Module 汇总插件宿主、后台管理服务、管理 handler 和协议 handler。
//
// 插件功能关闭时 Service 仍可创建，但 Handler 和 Protocol 会保持 nil，路由装配需要按 nil 处理。
type Module struct {
	Host     *pluginpkg.Host
	Service  Service
	Handler  *Handler
	Protocol *ProtocolHandler
}

// NewModule 创建项目插件模块。
//
// cfg.Enabled 为 false 时返回可查询 disabled 状态的最小模块；启用时必须提供 host，并创建后台管理与
// 插件协议入口。返回错误表示配置与依赖不满足启动条件。
func NewModule(cfg Config, host *pluginpkg.Host, logger ports.Logger) (Module, error) {
	service := NewService(host)
	module := Module{
		Host:    host,
		Service: service,
	}
	if !cfg.Enabled {
		if logger != nil {
			logger.Info("plugins module disabled")
		}
		return module, nil
	}
	if host == nil {
		return Module{}, pluginpkg.ErrDisabled
	}
	authenticator := newAuthenticator(cfg)
	module.Handler = NewHandler(service, logger)
	module.Protocol = NewProtocolHandler(host, cfg, authenticator)
	return module, nil
}

// ProtocolHandler 暴露插件协议的 HTTP、WebSocket 和 RPC 入口。
//
// 各 transport 的启用状态在构造时固化，后续请求通过 ensureHTTP 或对应开关返回协议级不可用错误。
type ProtocolHandler struct {
	host        *pluginpkg.Host
	http        *pluginhttp.Handler
	ws          *pluginws.Dispatcher
	auth        authenticator
	httpEnabled bool
	wsEnabled   bool
	rpcEnabled  bool
}

// NewProtocolHandler 创建插件协议处理器，并把远程调用器注册到宿主。
//
// 注册 remote invoker 的副作用让宿主在调用插件能力时能按 transport 选择对应客户端。
func NewProtocolHandler(host *pluginpkg.Host, cfg Config, auth authenticator) *ProtocolHandler {
	wsDispatcher := pluginws.NewDispatcher(host, pluginws.WithAuth(auth.WS))
	if cfg.HTTPEnabled {
		_ = host.RegisterRemoteInvoker(protocol.TransportHTTP, pluginhttp.NewRemoteInvoker(nil))
	}
	if cfg.WSEnabled {
		_ = host.RegisterRemoteInvoker(protocol.TransportWebSocket, wsDispatcher.RemoteInvoker())
	}
	if cfg.RPCEnabled {
		_ = host.RegisterRemoteInvoker(protocol.TransportRPC, pluginrpc.NewRemoteInvoker(nil))
	}
	return &ProtocolHandler{
		host:        host,
		http:        pluginhttp.New(host, pluginhttp.WithAuth(auth.HTTP)),
		ws:          wsDispatcher,
		auth:        auth,
		httpEnabled: cfg.HTTPEnabled,
		wsEnabled:   cfg.WSEnabled,
		rpcEnabled:  cfg.RPCEnabled,
	}
}

// RegisterRPC 将插件协议方法注册到应用 JSON-RPC 入口。
//
// 当 RPC transport 未启用时保持空操作，便于调用方统一执行注册流程。
func (h *ProtocolHandler) RegisterRPC(registry ports.RPCRegistry) error {
	if h == nil || !h.rpcEnabled {
		return nil
	}
	return pluginrpc.Register(rpcRegistry{inner: registry}, h.host, pluginrpc.WithAuth(h.auth.RPC))
}

func (h *ProtocolHandler) Negotiate(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.Negotiate(c)
}

func (h *ProtocolHandler) Register(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.Register(c)
}

func (h *ProtocolHandler) Heartbeat(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.Heartbeat(c)
}

func (h *ProtocolHandler) RenewLease(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.RenewLease(c)
}

func (h *ProtocolHandler) Unregister(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.Unregister(c)
}

func (h *ProtocolHandler) HealthCheck(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.HealthCheck(c)
}

func (h *ProtocolHandler) ListCapabilities(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.ListCapabilities(c)
}

func (h *ProtocolHandler) Invoke(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.Invoke(c)
}

func (h *ProtocolHandler) PushEvent(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.PushEvent(c)
}

func (h *ProtocolHandler) SubscribeEvent(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.SubscribeEvent(c)
}

func (h *ProtocolHandler) InjectContext(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.InjectContext(c)
}

func (h *ProtocolHandler) ReportStatus(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.ReportStatus(c)
}

func (h *ProtocolHandler) SyncMetadata(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.SyncMetadata(c)
}

func (h *ProtocolHandler) Drain(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.Drain(c)
}

func (h *ProtocolHandler) GetInjectedSchema(c ports.HTTPContext) {
	if !h.ensureHTTP(c) {
		return
	}
	h.http.GetInjectedSchema(c)
}

// WSEnvelope 处理 WebSocket transport 的真实升级请求或 JSON envelope 兼容请求。
//
// 浏览器/SDK 发起 Upgrade 时走长连接；普通 HTTP POST 发送 envelope 时走一次性 dispatch，
// 便于测试或不支持 WebSocket 的客户端复用同一协议语义。
func (h *ProtocolHandler) WSEnvelope(c ports.HTTPContext) {
	if !h.wsEnabled {
		c.JSON(404, pluginhttp.Response{Error: &protocol.Error{
			Code:    protocol.ErrorCodeTransportUnavailable,
			Message: pluginpkg.ErrTransportUnavailable.Error(),
		}})
		return
	}
	if isWebSocketUpgrade(c.Request()) {
		writerContext, ok := c.(interface {
			ResponseWriter() http.ResponseWriter
		})
		if !ok {
			c.JSON(500, pluginhttp.Response{Error: &protocol.Error{
				Code:    protocol.ErrorCodeTransportUnavailable,
				Message: pluginpkg.ErrTransportUnavailable.Error(),
			}})
			return
		}
		if err := h.ws.Serve(c.RequestContext(), writerContext.ResponseWriter(), c.Request()); errors.Is(err, pluginws.ErrInvalidWebSocketHandshake) {
			c.JSON(400, pluginhttp.Response{Error: &protocol.Error{
				Code:    protocol.ErrorCodeTransportUnavailable,
				Message: err.Error(),
			}})
		}
		return
	}
	var envelope protocol.Envelope
	if err := c.BindJSON(&envelope); err != nil {
		c.JSON(400, pluginhttp.Response{Error: &protocol.Error{
			Code:    protocol.ErrorCodeInvalidPlugin,
			Message: "invalid json payload",
		}})
		return
	}
	c.JSON(200, h.ws.Dispatch(c.RequestContext(), envelope))
}

// isWebSocketUpgrade 判断请求是否为标准 WebSocket 升级。
func isWebSocketUpgrade(req *http.Request) bool {
	if req == nil {
		return false
	}
	return strings.EqualFold(req.Header.Get("Upgrade"), "websocket") && strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade")
}

// ensureHTTP 确保 HTTP transport 可用，否则写入插件协议格式的 404 响应。
func (h *ProtocolHandler) ensureHTTP(c ports.HTTPContext) bool {
	if h != nil && h.httpEnabled {
		return true
	}
	c.JSON(404, pluginhttp.Response{Error: &protocol.Error{
		Code:    protocol.ErrorCodeTransportUnavailable,
		Message: pluginpkg.ErrTransportUnavailable.Error(),
	}})
	return false
}

// jsonObject 将结构化对象编码为 RawMessage。
//
// schema 构建阶段的编码失败会退回空对象，避免插件注入声明因为局部字段异常导致启动失败。
func jsonObject(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

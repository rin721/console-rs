package protocol

import (
	"encoding/json"
	"strings"
	"time"
)

const (
	// Operation* 常量定义插件宿主与插件之间支持的协议操作名。
	OperationRegister          = "register"
	OperationHeartbeat         = "heartbeat"
	OperationRenewLease        = "renew_lease"
	OperationUnregister        = "unregister"
	OperationHealthCheck       = "health_check"
	OperationListCapabilities  = "list_capabilities"
	OperationInvoke            = "invoke"
	OperationPushEvent         = "push_event"
	OperationSubscribeEvent    = "subscribe_event"
	OperationInjectContext     = "inject_context"
	OperationGetInjectedSchema = "get_injected_schema"
	OperationNegotiateProtocol = "negotiate_protocol"
	OperationReportStatus      = "report_status"
	OperationSyncMetadata      = "sync_metadata"
	OperationDrain             = "drain"

	// Transport* 常量定义内置支持的插件传输方式。
	TransportHTTP      = "http"
	TransportWebSocket = "websocket"
	TransportRPC       = "rpc"

	// PluginProtocolJSON 是 JSON 版本插件协议的标识。
	PluginProtocolJSON = "aoi-plugin-json"

	// ProtocolVersionV1 是当前插件协议版本。
	ProtocolVersionV1 = "v1"

	// Status* 常量描述插件实例在注册表中的生命周期状态。
	StatusOnline   = "online"
	StatusOffline  = "offline"
	StatusDraining = "draining"

	// RuntimeStatus* 常量描述插件自身报告的运行状态。
	RuntimeStatusReady    = "ready"
	RuntimeStatusBusy     = "busy"
	RuntimeStatusDegraded = "degraded"
	RuntimeStatusDraining = "draining"

	// HealthStatus* 常量描述健康检查结果。
	HealthStatusOK      = "ok"
	HealthStatusUnknown = "unknown"

	// CapabilityScope* 常量区分能力作用域。
	CapabilityScopeSystem = "system"
	CapabilityScopePlugin = "plugin"

	// SecretPolicy* 常量描述能力调用时 secret 的处理策略。
	SecretPolicyNone     = "none"
	SecretPolicyOneTime  = "one_time"
	SecretPolicyExternal = "external"

	// ErrorCode* 常量是插件协议层对外暴露的稳定错误码。
	ErrorCodeDisabled             = "plugin_host_disabled"
	ErrorCodeInvalidPlugin        = "invalid_plugin"
	ErrorCodeInvalidCapability    = "invalid_capability"
	ErrorCodePluginNotFound       = "plugin_not_found"
	ErrorCodePluginOffline        = "plugin_offline"
	ErrorCodeCapabilityNotFound   = "capability_not_found"
	ErrorCodeProviderUnavailable  = "provider_unavailable"
	ErrorCodeTransportUnavailable = "transport_unavailable"
	ErrorCodeUnsupportedProtocol  = "unsupported_protocol"
	ErrorCodeUnauthorized         = "unauthorized"
	ErrorCodeUnsupportedAuth      = "unsupported_auth"
	ErrorCodeInternal             = "internal_error"
)

// Envelope 是插件协议的外层消息信封，统一承载操作名、版本、payload 和错误。
type Envelope struct {
	ID        string          `json:"id,omitempty"`
	Type      string          `json:"type,omitempty"`
	Operation string          `json:"operation"`
	Version   string          `json:"version,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Error     *Error          `json:"error,omitempty"`
}

// Error 表示插件协议中的稳定错误对象。
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Auth 描述插件请求携带的认证材料，具体校验策略由宿主侧安全层决定。
type Auth struct {
	Mode      string `json:"mode,omitempty"`
	Token     string `json:"token,omitempty"`
	Signature string `json:"signature,omitempty"`
	KeyID     string `json:"key_id,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
}

// RequestMeta 是所有协议请求可复用的元数据块。
type RequestMeta struct {
	RequestID string `json:"request_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	// IdempotencyKey 用于标识可重试写操作，避免 transport 重试造成重复执行。
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Auth           *Auth             `json:"auth,omitempty"`
}

// Capability 描述插件向宿主暴露的一项可调用能力。
type Capability struct {
	Name         string          `json:"name"`
	Version      string          `json:"version,omitempty"`
	Scope        string          `json:"scope,omitempty"`
	Permissions  []string        `json:"permissions,omitempty"`
	InputSchema  json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`
	SecretPolicy string          `json:"secret_policy,omitempty"`
	Description  string          `json:"description,omitempty"`
}

// PluginMetadata 是插件注册和同步时提交的静态描述信息。
type PluginMetadata struct {
	PluginID      string            `json:"plugin_id"`
	InstanceID    string            `json:"instance_id"`
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Protocol      string            `json:"protocol"`
	Transport     string            `json:"transport,omitempty"`
	Endpoint      string            `json:"endpoint,omitempty"`
	Capabilities  []Capability      `json:"capabilities,omitempty"`
	Permissions   []string          `json:"permissions,omitempty"`
	Hooks         []string          `json:"hooks,omitempty"`
	SchemaVersion string            `json:"schema_version,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// NormalizeTransport 返回插件传输层的规范名称。
func NormalizeTransport(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case TransportHTTP, TransportWebSocket, TransportRPC:
		return value
	case "ws":
		return TransportWebSocket
	default:
		return value
	}
}

// IsTransport 判断 value 是否是当前内置支持的 transport 名称。
func IsTransport(value string) bool {
	switch NormalizeTransport(value) {
	case TransportHTTP, TransportWebSocket, TransportRPC:
		return true
	default:
		return false
	}
}

// EffectiveTransport 从显式 transport 字段读取传输类型，并兼容旧版 protocol 字段。
func EffectiveTransport(metadata PluginMetadata) string {
	if IsTransport(metadata.Transport) {
		return NormalizeTransport(metadata.Transport)
	}
	if IsTransport(metadata.Protocol) {
		return NormalizeTransport(metadata.Protocol)
	}
	return ""
}

// PluginSnapshot 是注册表中的插件实例快照，包含元数据、状态和租约时间。
type PluginSnapshot struct {
	PluginMetadata
	Status          string    `json:"status"`
	RuntimeStatus   string    `json:"runtime_status,omitempty"`
	OwnerHost       string    `json:"owner_host,omitempty"`
	LeaseTTLSeconds int       `json:"lease_ttl_seconds,omitempty"`
	LeaseExpiresAt  time.Time `json:"lease_expires_at,omitempty"`
	RegisteredAt    time.Time `json:"registered_at"`
	LastHeartbeatAt time.Time `json:"last_heartbeat_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// RegisterRequest 是插件注册请求。
type RegisterRequest struct {
	RequestMeta
	Plugin PluginMetadata `json:"plugin"`
}

// RegisterResponse 返回注册后的插件快照。
type RegisterResponse struct {
	Plugin PluginSnapshot `json:"plugin"`
}

// HeartbeatRequest 是插件续活心跳请求。
type HeartbeatRequest struct {
	RequestMeta
	PluginID   string `json:"plugin_id"`
	InstanceID string `json:"instance_id"`
}

// HeartbeatResponse 返回心跳处理后的插件快照。
type HeartbeatResponse struct {
	Plugin PluginSnapshot `json:"plugin"`
}

// RenewLeaseRequest 请求延长插件租约。
type RenewLeaseRequest struct {
	RequestMeta
	PluginID        string `json:"plugin_id"`
	InstanceID      string `json:"instance_id"`
	LeaseTTLSeconds int    `json:"lease_ttl_seconds,omitempty"`
}

// RenewLeaseResponse 返回续约后的插件快照。
type RenewLeaseResponse struct {
	Plugin PluginSnapshot `json:"plugin"`
}

// UnregisterRequest 请求注销指定插件实例。
type UnregisterRequest struct {
	RequestMeta
	PluginID   string `json:"plugin_id"`
	InstanceID string `json:"instance_id"`
}

// UnregisterResponse 返回注销结果。
type UnregisterResponse struct {
	PluginID   string `json:"plugin_id"`
	InstanceID string `json:"instance_id"`
	Status     string `json:"status"`
}

// HealthCheckRequest 请求检查插件实例健康状态。
type HealthCheckRequest struct {
	RequestMeta
	PluginID   string `json:"plugin_id"`
	InstanceID string `json:"instance_id,omitempty"`
}

// HealthStatus 描述一个插件实例的健康和租约状态。
type HealthStatus struct {
	PluginID        string    `json:"plugin_id"`
	InstanceID      string    `json:"instance_id,omitempty"`
	Status          string    `json:"status"`
	RuntimeStatus   string    `json:"runtime_status,omitempty"`
	LastHeartbeatAt time.Time `json:"last_heartbeat_at"`
	LeaseExpiresAt  time.Time `json:"lease_expires_at,omitempty"`
	Error           string    `json:"error,omitempty"`
}

// ListCapabilitiesRequest 请求列出插件能力。
type ListCapabilitiesRequest struct {
	RequestMeta
	PluginID   string `json:"plugin_id,omitempty"`
	InstanceID string `json:"instance_id,omitempty"`
}

// ListCapabilitiesResponse 返回插件能力列表。
type ListCapabilitiesResponse struct {
	Capabilities []Capability `json:"capabilities"`
}

// InvokeRequest 请求调用插件能力。
type InvokeRequest struct {
	RequestMeta
	PluginID          string            `json:"plugin_id,omitempty"`
	InstanceID        string            `json:"instance_id,omitempty"`
	Capability        string            `json:"capability"`
	TimeoutMillis     int               `json:"timeout_millis,omitempty"`
	PermissionContext map[string]string `json:"permission_context,omitempty"`
	Payload           json.RawMessage   `json:"payload,omitempty"`
}

// InvokeResponse 返回插件能力调用结果。
type InvokeResponse struct {
	Capability string          `json:"capability"`
	Result     json.RawMessage `json:"result,omitempty"`
}

// PushEventRequest 请求宿主向插件推送事件。
type PushEventRequest struct {
	RequestMeta
	PluginID          string            `json:"plugin_id,omitempty"`
	InstanceID        string            `json:"instance_id,omitempty"`
	Event             string            `json:"event"`
	PermissionContext map[string]string `json:"permission_context,omitempty"`
	Payload           json.RawMessage   `json:"payload,omitempty"`
}

// PushEventResponse 返回事件推送接收结果。
type PushEventResponse struct {
	Accepted  bool   `json:"accepted"`
	Event     string `json:"event"`
	Delivered int    `json:"delivered,omitempty"`
}

// SubscribeEventRequest 请求插件订阅宿主事件。
type SubscribeEventRequest struct {
	RequestMeta
	PluginID   string            `json:"plugin_id"`
	InstanceID string            `json:"instance_id"`
	Events     []string          `json:"events"`
	Filters    map[string]string `json:"filters,omitempty"`
}

// SubscribeEventResponse 返回事件订阅结果。
type SubscribeEventResponse struct {
	PluginID   string   `json:"plugin_id"`
	InstanceID string   `json:"instance_id"`
	Events     []string `json:"events"`
	Accepted   bool     `json:"accepted"`
}

// InjectContextRequest 请求宿主生成可注入到插件的上下文。
type InjectContextRequest struct {
	RequestMeta
	PluginID      string   `json:"plugin_id,omitempty"`
	InstanceID    string   `json:"instance_id,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"`
	SchemaVersion string   `json:"schema_version,omitempty"`
}

// InjectContextResponse 返回注入上下文及其审计信息。
type InjectContextResponse struct {
	SchemaVersion string                     `json:"schema_version"`
	Context       map[string]json.RawMessage `json:"context"`
	Audit         AuditInfo                  `json:"audit"`
}

// GetInjectedSchemaRequest 请求注入上下文的 schema 描述。
type GetInjectedSchemaRequest struct {
	RequestMeta
	Capabilities []string `json:"capabilities,omitempty"`
}

// GetInjectedSchemaResponse 返回可注入能力 schema 和审计信息。
type GetInjectedSchemaResponse struct {
	SchemaVersion string               `json:"schema_version"`
	Capabilities  []InjectedCapability `json:"capabilities"`
	Audit         AuditInfo            `json:"audit"`
}

// InjectedCapability 描述可注入上下文中的单个能力 schema。
type InjectedCapability struct {
	Name        string          `json:"name"`
	Version     string          `json:"version,omitempty"`
	Kind        string          `json:"kind,omitempty"`
	Permissions []string        `json:"permissions,omitempty"`
	Schema      json.RawMessage `json:"schema,omitempty"`
	Description string          `json:"description,omitempty"`
}

// AuditInfo 描述宿主生成上下文或 schema 时的审计元数据。
type AuditInfo struct {
	GeneratedAt time.Time `json:"generated_at"`
	Source      string    `json:"source,omitempty"`
}

// NegotiateProtocolRequest 请求协商插件协议版本和传输方式。
type NegotiateProtocolRequest struct {
	RequestMeta
	PluginID          string   `json:"plugin_id,omitempty"`
	InstanceID        string   `json:"instance_id,omitempty"`
	ProtocolVersions  []string `json:"protocol_versions,omitempty"`
	Transports        []string `json:"transports,omitempty"`
	SchemaVersion     string   `json:"schema_version,omitempty"`
	PreferredProtocol string   `json:"preferred_protocol,omitempty"`
}

// NegotiateProtocolResponse 返回协商结果和可选替代方案。
type NegotiateProtocolResponse struct {
	ProtocolVersion string   `json:"protocol_version"`
	Transport       string   `json:"transport"`
	Accepted        bool     `json:"accepted"`
	Alternatives    []string `json:"alternatives,omitempty"`
}

// ReportStatusRequest 请求插件主动报告运行状态。
type ReportStatusRequest struct {
	RequestMeta
	PluginID      string            `json:"plugin_id"`
	InstanceID    string            `json:"instance_id"`
	Status        string            `json:"status,omitempty"`
	RuntimeStatus string            `json:"runtime_status,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// ReportStatusResponse 返回状态同步后的插件快照。
type ReportStatusResponse struct {
	Plugin PluginSnapshot `json:"plugin"`
}

// SyncMetadataRequest 请求同步插件元数据。
type SyncMetadataRequest struct {
	RequestMeta
	Plugin PluginMetadata `json:"plugin"`
}

// SyncMetadataResponse 返回同步后的插件快照。
type SyncMetadataResponse struct {
	Plugin PluginSnapshot `json:"plugin"`
}

// DrainRequest 请求插件进入 draining 状态。
type DrainRequest struct {
	RequestMeta
	PluginID   string `json:"plugin_id"`
	InstanceID string `json:"instance_id"`
	Reason     string `json:"reason,omitempty"`
}

// DrainResponse 返回 draining 状态切换后的插件快照。
type DrainResponse struct {
	Plugin PluginSnapshot `json:"plugin"`
}

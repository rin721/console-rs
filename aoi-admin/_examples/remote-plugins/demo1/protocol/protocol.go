// Package protocol 定义 demo 远程插件本地使用的 JSON wire DTO。
//
// 远程插件不依赖主系统 Go module；生产插件应根据公开 OpenAPI /
// JSON Schema / JSON-RPC 契约自行定义 DTO，或从契约生成客户端。
package protocol

import (
	"encoding/json"
	"time"
)

const (
	TransportHTTP      = "http"
	TransportWebSocket = "websocket"
	TransportRPC       = "rpc"

	PluginProtocolJSON = "aoi-plugin-json"

	ProtocolVersionV1 = "v1"

	StatusOnline = "online"

	RuntimeStatusReady = "ready"

	CapabilityScopePlugin = "plugin"

	ErrorCodeInvalidPlugin      = "invalid_plugin"
	ErrorCodeCapabilityNotFound = "capability_not_found"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Auth struct {
	Mode      string `json:"mode,omitempty"`
	Token     string `json:"token,omitempty"`
	Signature string `json:"signature,omitempty"`
	KeyID     string `json:"key_id,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
}

type RequestMeta struct {
	RequestID      string            `json:"request_id,omitempty"`
	TraceID        string            `json:"trace_id,omitempty"`
	Timestamp      string            `json:"timestamp,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Auth           *Auth             `json:"auth,omitempty"`
}

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

type RegisterRequest struct {
	RequestMeta
	Plugin PluginMetadata `json:"plugin"`
}

type UnregisterRequest struct {
	RequestMeta
	PluginID   string `json:"plugin_id"`
	InstanceID string `json:"instance_id"`
}

type RenewLeaseRequest struct {
	RequestMeta
	PluginID        string `json:"plugin_id"`
	InstanceID      string `json:"instance_id"`
	LeaseTTLSeconds int    `json:"lease_ttl_seconds,omitempty"`
}

type SubscribeEventRequest struct {
	RequestMeta
	PluginID   string            `json:"plugin_id"`
	InstanceID string            `json:"instance_id"`
	Events     []string          `json:"events"`
	Filters    map[string]string `json:"filters,omitempty"`
}

type ReportStatusRequest struct {
	RequestMeta
	PluginID      string            `json:"plugin_id"`
	InstanceID    string            `json:"instance_id"`
	Status        string            `json:"status,omitempty"`
	RuntimeStatus string            `json:"runtime_status,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type GetInjectedSchemaRequest struct {
	RequestMeta
	Capabilities []string `json:"capabilities,omitempty"`
}

type GetInjectedSchemaResponse struct {
	SchemaVersion string               `json:"schema_version"`
	Capabilities  []InjectedCapability `json:"capabilities"`
	Audit         AuditInfo            `json:"audit"`
}

type InjectedCapability struct {
	Name        string          `json:"name"`
	Version     string          `json:"version,omitempty"`
	Kind        string          `json:"kind,omitempty"`
	Permissions []string        `json:"permissions,omitempty"`
	Schema      json.RawMessage `json:"schema,omitempty"`
	Description string          `json:"description,omitempty"`
}

type AuditInfo struct {
	GeneratedAt time.Time `json:"generated_at"`
	Source      string    `json:"source,omitempty"`
}

type InvokeRequest struct {
	RequestMeta
	PluginID          string            `json:"plugin_id,omitempty"`
	InstanceID        string            `json:"instance_id,omitempty"`
	Capability        string            `json:"capability"`
	TimeoutMillis     int               `json:"timeout_millis,omitempty"`
	PermissionContext map[string]string `json:"permission_context,omitempty"`
	Payload           json.RawMessage   `json:"payload,omitempty"`
}

type InvokeResponse struct {
	Capability string          `json:"capability"`
	Result     json.RawMessage `json:"result,omitempty"`
}

type PushEventRequest struct {
	RequestMeta
	PluginID          string            `json:"plugin_id,omitempty"`
	InstanceID        string            `json:"instance_id,omitempty"`
	Event             string            `json:"event"`
	PermissionContext map[string]string `json:"permission_context,omitempty"`
	Payload           json.RawMessage   `json:"payload,omitempty"`
}

type PushEventResponse struct {
	Accepted  bool   `json:"accepted"`
	Event     string `json:"event"`
	Delivered int    `json:"delivered,omitempty"`
}

type NegotiateProtocolRequest struct {
	RequestMeta
	PluginID          string   `json:"plugin_id,omitempty"`
	InstanceID        string   `json:"instance_id,omitempty"`
	ProtocolVersions  []string `json:"protocol_versions,omitempty"`
	Transports        []string `json:"transports,omitempty"`
	SchemaVersion     string   `json:"schema_version,omitempty"`
	PreferredProtocol string   `json:"preferred_protocol,omitempty"`
}

type NegotiateProtocolResponse struct {
	ProtocolVersion string   `json:"protocol_version"`
	Transport       string   `json:"transport"`
	Accepted        bool     `json:"accepted"`
	Alternatives    []string `json:"alternatives,omitempty"`
}

type DrainRequest struct {
	RequestMeta
	PluginID   string `json:"plugin_id"`
	InstanceID string `json:"instance_id"`
	Reason     string `json:"reason,omitempty"`
}

type DrainResponse struct {
	Plugin PluginSnapshot `json:"plugin"`
}

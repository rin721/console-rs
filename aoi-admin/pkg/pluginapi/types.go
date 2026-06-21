// Package pluginapi 是已弃用的兼容层，用于保持旧插件 API 的导入路径可用。
//
// 新代码应直接导入 github.com/rei0721/go-scaffold/pkg/plugin/protocol。
package pluginapi

import "github.com/rei0721/go-scaffold/pkg/plugin/protocol"

const (
	MethodRegister         = protocol.OperationRegister
	MethodHeartbeat        = protocol.OperationHeartbeat
	MethodRenewLease       = protocol.OperationRenewLease
	MethodUnregister       = protocol.OperationUnregister
	MethodInvoke           = protocol.OperationInvoke
	MethodListCapabilities = protocol.OperationListCapabilities

	ProtocolJSONRPC   = "json-rpc"
	ProtocolHTTP      = protocol.TransportHTTP
	ProtocolWebSocket = protocol.TransportWebSocket
	ProtocolRPC       = protocol.TransportRPC

	StatusOnline   = protocol.StatusOnline
	StatusOffline  = protocol.StatusOffline
	StatusDraining = protocol.StatusDraining

	HealthStatusOK      = protocol.HealthStatusOK
	HealthStatusUnknown = protocol.HealthStatusUnknown

	CapabilityScopeSystem = protocol.CapabilityScopeSystem
	CapabilityScopePlugin = protocol.CapabilityScopePlugin

	SecretPolicyNone     = protocol.SecretPolicyNone
	SecretPolicyOneTime  = protocol.SecretPolicyOneTime
	SecretPolicyExternal = protocol.SecretPolicyExternal
)

type PluginMetadata = protocol.PluginMetadata
type Capability = protocol.Capability
type PluginSnapshot = protocol.PluginSnapshot
type RegisterRequest = protocol.RegisterRequest
type RegisterResponse = protocol.RegisterResponse
type HeartbeatRequest = protocol.HeartbeatRequest
type HeartbeatResponse = protocol.HeartbeatResponse
type RenewLeaseRequest = protocol.RenewLeaseRequest
type RenewLeaseResponse = protocol.RenewLeaseResponse
type UnregisterRequest = protocol.UnregisterRequest
type UnregisterResponse = protocol.UnregisterResponse
type ListCapabilitiesRequest = protocol.ListCapabilitiesRequest
type ListCapabilitiesResponse = protocol.ListCapabilitiesResponse
type InvokeRequest = protocol.InvokeRequest
type InvokeResponse = protocol.InvokeResponse
type PushEventRequest = protocol.PushEventRequest
type PushEventResponse = protocol.PushEventResponse
type SubscribeEventRequest = protocol.SubscribeEventRequest
type SubscribeEventResponse = protocol.SubscribeEventResponse
type ReportStatusRequest = protocol.ReportStatusRequest
type ReportStatusResponse = protocol.ReportStatusResponse
type SyncMetadataRequest = protocol.SyncMetadataRequest
type SyncMetadataResponse = protocol.SyncMetadataResponse
type DrainRequest = protocol.DrainRequest
type DrainResponse = protocol.DrainResponse
type HealthStatus = protocol.HealthStatus

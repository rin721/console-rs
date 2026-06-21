// Package registry 定义远程插件实例、租约、订阅和状态变化的存储抽象。
package registry

import (
	"context"
	"errors"
	"time"

	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

var (
	// ErrNotFound 和 ErrInvalid 是 registry 实现对外暴露的稳定错误哨兵。
	ErrNotFound = errors.New("plugin registry instance not found")
	ErrInvalid  = errors.New("invalid plugin registry record")
)

// InstanceKey 唯一标识一个远程插件运行实例。
type InstanceKey struct {
	PluginID   string
	InstanceID string
}

// InstanceFilter 描述查询插件实例时可叠加的过滤条件。
//
// IncludeExpired 控制在线但租约已过期的实例是否仍返回；Now 允许测试或调用方固定过期判断时间。
type InstanceFilter struct {
	PluginID       string
	InstanceID     string
	Capability     string
	Status         string
	IncludeExpired bool
	Now            time.Time
}

// Lease 描述插件实例续租时写入的租约信息。
//
// LastHeartbeatAt 是远端最近一次心跳时间，ExpiresAt 是主系统计算后的租约截止时间。
type Lease struct {
	PluginID        string
	InstanceID      string
	LeaseTTL        time.Duration
	LastHeartbeatAt time.Time
	ExpiresAt       time.Time
}

// Subscription 表示插件实例对某类事件的订阅记录。
type Subscription struct {
	PluginID   string
	InstanceID string
	Event      string
	Filters    map[string]string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// SubscriptionFilter 描述事件订阅查询条件。
type SubscriptionFilter struct {
	PluginID   string
	InstanceID string
	Event      string
}

// Change* 常量是 watcher 对外发送的状态变化类型，消费者应按类型判断 Plugin、Previous 或 Subscription 是否有值。
const (
	ChangeRegistered          = "registered"
	ChangeLeaseRenewed        = "lease_renewed"
	ChangeUnregistered        = "unregistered"
	ChangeStatusReported      = "status_reported"
	ChangeMetadataSynced      = "metadata_synced"
	ChangeLeaseExpired        = "lease_expired"
	ChangeEventSubscribed     = "event_subscribed"
	ChangeSnapshotObserved    = "snapshot_observed"
	ChangeSnapshotDisappeared = "snapshot_disappeared"
)

// Change 表示 registry 中一次可观察状态变化。
//
// Plugin 保存变化后的快照，Previous 保存变化前快照；事件订阅变化会通过 Subscription 表达。
type Change struct {
	Type         string
	PluginID     string
	InstanceID   string
	Plugin       protocol.PluginSnapshot
	Previous     *protocol.PluginSnapshot
	Subscription *Subscription
	ObservedAt   time.Time
}

// WatchOptions 控制 watcher 的缓冲、初始快照和轮询行为。
//
// InitialSnapshot 为 true 时会先输出当前状态，随后再输出增量变化。
type WatchOptions struct {
	Buffer          int
	InitialSnapshot bool
	PollInterval    time.Duration
	Filter          InstanceFilter
}

// Watcher 统一抽象 registry 状态变化订阅能力。
type Watcher interface {
	Watch(context.Context, WatchOptions) (<-chan Change, error)
}

// Registry 定义插件实例注册、租约、查询、事件订阅和过期处理的持久化契约。
//
// 实现可以是内存、SQL 或其他共享存储，但必须返回 clone 后的数据，避免调用方修改内部状态。
type Registry interface {
	RegisterInstance(context.Context, protocol.PluginSnapshot) (protocol.PluginSnapshot, error)
	RenewLease(context.Context, Lease) (protocol.PluginSnapshot, error)
	UnregisterInstance(context.Context, string, string, time.Time) error
	ReportStatus(context.Context, string, string, string, string, map[string]string, time.Time) (protocol.PluginSnapshot, error)
	SyncMetadata(context.Context, protocol.PluginSnapshot) (protocol.PluginSnapshot, error)
	GetInstance(context.Context, string, string) (protocol.PluginSnapshot, error)
	ListInstances(context.Context, InstanceFilter) ([]protocol.PluginSnapshot, error)
	ListByCapability(context.Context, string, InstanceFilter) ([]protocol.PluginSnapshot, error)
	ListExpired(context.Context, time.Time) ([]protocol.PluginSnapshot, error)
	ExpireLeases(context.Context, time.Time) ([]protocol.PluginSnapshot, error)
	SubscribeEvent(context.Context, Subscription) (Subscription, error)
	ListSubscriptions(context.Context, SubscriptionFilter) ([]Subscription, error)
}

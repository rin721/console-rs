package registry

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

// Memory 是进程内插件注册中心实现。
// 它适用于单进程运行和测试场景；所有对外返回的快照都会深拷贝，避免调用方修改内部状态。
type Memory struct {
	mu            sync.RWMutex
	instances     map[string]protocol.PluginSnapshot
	subscriptions map[string]Subscription
	watchers      map[int]memoryWatcher
	nextWatcherID int
}

// memoryWatcher 保存单个订阅者的通道和过滤条件。
// notify 使用非阻塞发送，慢消费者不会拖住 registry 主流程。
type memoryWatcher struct {
	ch     chan Change
	filter InstanceFilter
}

// NewMemory 创建空的内存注册中心。
func NewMemory() *Memory {
	return &Memory{
		instances:     map[string]protocol.PluginSnapshot{},
		subscriptions: map[string]Subscription{},
		watchers:      map[int]memoryWatcher{},
	}
}

// RegisterInstance 新增或替换一个插件实例快照。
// snapshot 必须包含 plugin_id 和 instance_id；如果实例已存在，则保留原 CreatedAt。
// 副作用是更新内存状态并向 watcher 发送 registered 变化。
func (m *Memory) RegisterInstance(_ context.Context, snapshot protocol.PluginSnapshot) (protocol.PluginSnapshot, error) {
	if err := validateSnapshot(snapshot); err != nil {
		return protocol.PluginSnapshot{}, err
	}
	m.mu.Lock()
	var previous *protocol.PluginSnapshot
	if existing, ok := m.instances[key(snapshot.PluginID, snapshot.InstanceID)]; ok && !existing.CreatedAt.IsZero() {
		existing := cloneSnapshot(existing)
		previous = &existing
		snapshot.CreatedAt = existing.CreatedAt
	}
	m.instances[key(snapshot.PluginID, snapshot.InstanceID)] = cloneSnapshot(snapshot)
	out := cloneSnapshot(snapshot)
	m.mu.Unlock()
	m.notify(Change{Type: ChangeRegistered, PluginID: out.PluginID, InstanceID: out.InstanceID, Plugin: out, Previous: previous})
	return out, nil
}

// RenewLease 刷新插件实例租约并把实例恢复为 online。
// lease 标识目标实例和新的心跳/过期时间；返回更新后的快照，不存在时返回 ErrNotFound。
func (m *Memory) RenewLease(_ context.Context, lease Lease) (protocol.PluginSnapshot, error) {
	m.mu.Lock()
	item, ok := m.instances[key(lease.PluginID, lease.InstanceID)]
	if !ok {
		m.mu.Unlock()
		return protocol.PluginSnapshot{}, ErrNotFound
	}
	previous := cloneSnapshot(item)
	item.Status = protocol.StatusOnline
	item.LastHeartbeatAt = utcOrNow(lease.LastHeartbeatAt)
	item.LeaseExpiresAt = utcOrNow(lease.ExpiresAt)
	if lease.LeaseTTL > 0 {
		item.LeaseTTLSeconds = int(lease.LeaseTTL.Seconds())
	}
	item.UpdatedAt = item.LastHeartbeatAt
	m.instances[key(lease.PluginID, lease.InstanceID)] = item
	out := cloneSnapshot(item)
	m.mu.Unlock()
	m.notify(Change{Type: ChangeLeaseRenewed, PluginID: out.PluginID, InstanceID: out.InstanceID, Plugin: out, Previous: &previous})
	return out, nil
}

// UnregisterInstance 将插件实例标记为 offline。
// 这里不删除快照，保留最后状态用于管理端观察和 watcher 对比。
func (m *Memory) UnregisterInstance(_ context.Context, pluginID string, instanceID string, now time.Time) error {
	m.mu.Lock()
	k := key(pluginID, instanceID)
	item, ok := m.instances[k]
	if !ok {
		m.mu.Unlock()
		return ErrNotFound
	}
	previous := cloneSnapshot(item)
	now = utcOrNow(now)
	item.Status = protocol.StatusOffline
	item.RuntimeStatus = ""
	item.LeaseExpiresAt = now
	item.UpdatedAt = now
	m.instances[k] = item
	out := cloneSnapshot(item)
	m.mu.Unlock()
	m.notify(Change{Type: ChangeUnregistered, PluginID: out.PluginID, InstanceID: out.InstanceID, Plugin: out, Previous: &previous})
	return nil
}

// ReportStatus 合并插件主动上报的运行状态和 metadata。
// 空 status/runtimeStatus 表示不覆盖已有值；metadata 按 key 增量合并。
func (m *Memory) ReportStatus(_ context.Context, pluginID string, instanceID string, status string, runtimeStatus string, metadata map[string]string, now time.Time) (protocol.PluginSnapshot, error) {
	m.mu.Lock()
	item, ok := m.instances[key(pluginID, instanceID)]
	if !ok {
		m.mu.Unlock()
		return protocol.PluginSnapshot{}, ErrNotFound
	}
	previous := cloneSnapshot(item)
	if strings.TrimSpace(status) != "" {
		item.Status = strings.TrimSpace(status)
	}
	if strings.TrimSpace(runtimeStatus) != "" {
		item.RuntimeStatus = strings.TrimSpace(runtimeStatus)
	}
	if len(metadata) > 0 {
		item.Metadata = mergeMetadata(item.Metadata, metadata)
	}
	item.UpdatedAt = utcOrNow(now)
	m.instances[key(pluginID, instanceID)] = item
	out := cloneSnapshot(item)
	m.mu.Unlock()
	m.notify(Change{Type: ChangeStatusReported, PluginID: out.PluginID, InstanceID: out.InstanceID, Plugin: out, Previous: &previous})
	return out, nil
}

// SyncMetadata 用插件提交的完整元数据替换已有快照。
// 调用方可以只提交业务元数据，缺失的生命周期字段会从已有快照回填。
func (m *Memory) SyncMetadata(_ context.Context, snapshot protocol.PluginSnapshot) (protocol.PluginSnapshot, error) {
	if err := validateSnapshot(snapshot); err != nil {
		return protocol.PluginSnapshot{}, err
	}
	m.mu.Lock()
	k := key(snapshot.PluginID, snapshot.InstanceID)
	existing, ok := m.instances[k]
	if !ok {
		m.mu.Unlock()
		return protocol.PluginSnapshot{}, ErrNotFound
	}
	previous := cloneSnapshot(existing)
	snapshot.Status = firstNonEmpty(snapshot.Status, existing.Status)
	snapshot.RuntimeStatus = firstNonEmpty(snapshot.RuntimeStatus, existing.RuntimeStatus)
	snapshot.OwnerHost = firstNonEmpty(snapshot.OwnerHost, existing.OwnerHost)
	if snapshot.LeaseTTLSeconds == 0 {
		snapshot.LeaseTTLSeconds = existing.LeaseTTLSeconds
	}
	if snapshot.LeaseExpiresAt.IsZero() {
		snapshot.LeaseExpiresAt = existing.LeaseExpiresAt
	}
	if snapshot.RegisteredAt.IsZero() {
		snapshot.RegisteredAt = existing.RegisteredAt
	}
	if snapshot.LastHeartbeatAt.IsZero() {
		snapshot.LastHeartbeatAt = existing.LastHeartbeatAt
	}
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = existing.CreatedAt
	}
	if snapshot.UpdatedAt.IsZero() {
		snapshot.UpdatedAt = existing.UpdatedAt
	}
	m.instances[k] = cloneSnapshot(snapshot)
	out := cloneSnapshot(snapshot)
	m.mu.Unlock()
	m.notify(Change{Type: ChangeMetadataSynced, PluginID: out.PluginID, InstanceID: out.InstanceID, Plugin: out, Previous: &previous})
	return out, nil
}

// GetInstance 按 plugin_id 和 instance_id 读取插件实例快照。
// 返回值是内部状态的深拷贝，调用方修改不会影响 registry。
func (m *Memory) GetInstance(_ context.Context, pluginID string, instanceID string) (protocol.PluginSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.instances[key(pluginID, instanceID)]
	if !ok {
		return protocol.PluginSnapshot{}, ErrNotFound
	}
	return cloneSnapshot(item), nil
}

// ListInstances 按过滤条件列出插件实例。
// 过期在线实例默认会被过滤，除非 filter.IncludeExpired 显式放开。
func (m *Memory) ListInstances(_ context.Context, filter InstanceFilter) ([]protocol.PluginSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := make([]protocol.PluginSnapshot, 0, len(m.instances))
	for _, item := range m.instances {
		if matches(item, filter) {
			items = append(items, cloneSnapshot(item))
		}
	}
	sortSnapshots(items)
	return items, nil
}

// ListByCapability 是按能力查询实例的便捷入口。
func (m *Memory) ListByCapability(ctx context.Context, capability string, filter InstanceFilter) ([]protocol.PluginSnapshot, error) {
	filter.Capability = capability
	return m.ListInstances(ctx, filter)
}

// ListExpired 返回租约已到期但状态仍为 online 的实例。
// now 为空时使用当前 UTC 时间，方便测试传入固定时间。
func (m *Memory) ListExpired(_ context.Context, now time.Time) ([]protocol.PluginSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	now = utcOrNow(now)
	var items []protocol.PluginSnapshot
	for _, item := range m.instances {
		if item.Status == protocol.StatusOnline && !item.LeaseExpiresAt.IsZero() && !item.LeaseExpiresAt.After(now) {
			items = append(items, cloneSnapshot(item))
		}
	}
	sortSnapshots(items)
	return items, nil
}

// ExpireLeases 将已过期的 online 实例标记为 offline。
// 状态修改在锁内完成，watcher 通知在解锁后发送，避免回调路径阻塞写锁。
func (m *Memory) ExpireLeases(_ context.Context, now time.Time) ([]protocol.PluginSnapshot, error) {
	m.mu.Lock()
	now = utcOrNow(now)
	var expired []protocol.PluginSnapshot
	var changes []Change
	for k, item := range m.instances {
		if item.Status == protocol.StatusOnline && !item.LeaseExpiresAt.IsZero() && !item.LeaseExpiresAt.After(now) {
			previous := cloneSnapshot(item)
			item.Status = protocol.StatusOffline
			item.UpdatedAt = now
			m.instances[k] = item
			out := cloneSnapshot(item)
			expired = append(expired, out)
			changes = append(changes, Change{Type: ChangeLeaseExpired, PluginID: out.PluginID, InstanceID: out.InstanceID, Plugin: out, Previous: &previous})
		}
	}
	sortSnapshots(expired)
	m.mu.Unlock()
	for _, change := range changes {
		m.notify(change)
	}
	return expired, nil
}

// SubscribeEvent 记录插件实例对事件的订阅。
// 同一 plugin_id、instance_id、event 会覆盖旧订阅，便于插件重复上报时保持幂等。
func (m *Memory) SubscribeEvent(_ context.Context, sub Subscription) (Subscription, error) {
	if strings.TrimSpace(sub.PluginID) == "" || strings.TrimSpace(sub.InstanceID) == "" || strings.TrimSpace(sub.Event) == "" {
		return Subscription{}, ErrInvalid
	}
	now := utcOrNow(sub.UpdatedAt)
	sub.PluginID = strings.TrimSpace(sub.PluginID)
	sub.InstanceID = strings.TrimSpace(sub.InstanceID)
	sub.Event = strings.TrimSpace(sub.Event)
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	sub.UpdatedAt = now
	m.mu.Lock()
	m.subscriptions[subscriptionKey(sub.PluginID, sub.InstanceID, sub.Event)] = cloneSubscription(sub)
	out := cloneSubscription(sub)
	m.mu.Unlock()
	m.notify(Change{Type: ChangeEventSubscribed, PluginID: out.PluginID, InstanceID: out.InstanceID, Subscription: &out})
	return out, nil
}

// ListSubscriptions 按插件、实例或事件过滤订阅记录。
// 返回结果按插件、实例、事件排序，保证事件推送目标稳定。
func (m *Memory) ListSubscriptions(_ context.Context, filter SubscriptionFilter) ([]Subscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var items []Subscription
	for _, sub := range m.subscriptions {
		if filter.PluginID != "" && sub.PluginID != filter.PluginID {
			continue
		}
		if filter.InstanceID != "" && sub.InstanceID != filter.InstanceID {
			continue
		}
		if filter.Event != "" && sub.Event != filter.Event {
			continue
		}
		items = append(items, cloneSubscription(sub))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].PluginID != items[j].PluginID {
			return items[i].PluginID < items[j].PluginID
		}
		if items[i].InstanceID != items[j].InstanceID {
			return items[i].InstanceID < items[j].InstanceID
		}
		return items[i].Event < items[j].Event
	})
	return items, nil
}

// Watch 注册一个内存 watcher 并返回变化通道。
// InitialSnapshot 会先发送当前匹配实例；通道随 context 结束关闭。
func (m *Memory) Watch(ctx context.Context, options WatchOptions) (<-chan Change, error) {
	if m == nil {
		return nil, ErrInvalid
	}
	if options.Buffer <= 0 {
		options.Buffer = 32
	}
	if options.Filter.Now.IsZero() && options.Filter.Status == "" {
		options.Filter.IncludeExpired = true
	}
	ch := make(chan Change, options.Buffer)
	m.mu.Lock()
	id := m.nextWatcherID
	m.nextWatcherID++
	m.watchers[id] = memoryWatcher{ch: ch, filter: options.Filter}
	var initial []protocol.PluginSnapshot
	if options.InitialSnapshot {
		for _, item := range m.instances {
			if matches(item, options.Filter) {
				initial = append(initial, cloneSnapshot(item))
			}
		}
		sortSnapshots(initial)
	}
	m.mu.Unlock()

	go func() {
		for _, item := range initial {
			change := Change{Type: ChangeSnapshotObserved, PluginID: item.PluginID, InstanceID: item.InstanceID, Plugin: item, ObservedAt: time.Now().UTC()}
			select {
			case ch <- cloneChange(change):
			case <-ctx.Done():
				m.closeWatcher(id)
				return
			default:
			}
		}
		<-ctx.Done()
		m.closeWatcher(id)
	}()
	return ch, nil
}

// closeWatcher 从 watcher 表移除订阅并关闭其通道。
func (m *Memory) closeWatcher(id int) {
	m.mu.Lock()
	watcher, ok := m.watchers[id]
	if ok {
		delete(m.watchers, id)
	}
	m.mu.Unlock()
	if ok {
		close(watcher.ch)
	}
}

// notify 将变化广播给匹配的 watcher。
// 发送使用 default 分支丢弃满通道事件，避免慢消费者影响插件注册和续租路径。
func (m *Memory) notify(change Change) {
	if m == nil {
		return
	}
	change = cloneChange(change)
	if change.ObservedAt.IsZero() {
		change.ObservedAt = time.Now().UTC()
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, watcher := range m.watchers {
		if !matchesChange(change, watcher.filter) {
			continue
		}
		select {
		case watcher.ch <- change:
		default:
		}
	}
}

// matchesChange 将变化事件补齐为快照语义后复用实例过滤逻辑。
func matchesChange(change Change, filter InstanceFilter) bool {
	item := change.Plugin
	if item.PluginID == "" {
		item.PluginID = change.PluginID
	}
	if item.InstanceID == "" {
		item.InstanceID = change.InstanceID
	}
	return matches(item, filter)
}

// matches 判断实例是否满足查询或 watcher 过滤条件。
// IncludeExpired 为 false 时，租约已过期的 online 实例会被隐藏。
func matches(item protocol.PluginSnapshot, filter InstanceFilter) bool {
	if filter.PluginID != "" && item.PluginID != filter.PluginID {
		return false
	}
	if filter.InstanceID != "" && item.InstanceID != filter.InstanceID {
		return false
	}
	if filter.Status != "" && item.Status != filter.Status {
		return false
	}
	if filter.Capability != "" && !hasCapability(item, filter.Capability) {
		return false
	}
	now := filter.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if !filter.IncludeExpired && item.Status == protocol.StatusOnline && !item.LeaseExpiresAt.IsZero() && !item.LeaseExpiresAt.After(now) {
		return false
	}
	return true
}

// hasCapability 判断实例是否声明了指定能力。
func hasCapability(item protocol.PluginSnapshot, name string) bool {
	for _, capability := range item.Capabilities {
		if capability.Name == name {
			return true
		}
	}
	return false
}

// validateSnapshot 校验快照是否具备 registry 的最小身份字段。
func validateSnapshot(snapshot protocol.PluginSnapshot) error {
	if strings.TrimSpace(snapshot.PluginID) == "" || strings.TrimSpace(snapshot.InstanceID) == "" {
		return ErrInvalid
	}
	return nil
}

// key 使用 NUL 分隔 plugin_id 和 instance_id，避免普通字符串拼接出现歧义。
func key(pluginID string, instanceID string) string {
	return strings.TrimSpace(pluginID) + "\x00" + strings.TrimSpace(instanceID)
}

// subscriptionKey 在实例 key 后追加事件名，唯一标识一条订阅。
func subscriptionKey(pluginID string, instanceID string, event string) string {
	return key(pluginID, instanceID) + "\x00" + strings.TrimSpace(event)
}

// utcOrNow 将时间统一为 UTC；零值表示使用当前时间。
func utcOrNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

// firstNonEmpty 返回第一个去空白后非空的字符串，用于元数据回填。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// mergeMetadata 合并 metadata，并忽略空 key。
// 返回新 map，避免调用方持有的 map 与 registry 内部状态共享。
func mergeMetadata(base map[string]string, update map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range update {
		if strings.TrimSpace(key) != "" {
			out[strings.TrimSpace(key)] = value
		}
	}
	return out
}

// cloneChange 深拷贝变化事件中的可变字段。
func cloneChange(src Change) Change {
	dst := src
	dst.Plugin = cloneSnapshot(src.Plugin)
	if src.Previous != nil {
		previous := cloneSnapshot(*src.Previous)
		dst.Previous = &previous
	}
	if src.Subscription != nil {
		sub := cloneSubscription(*src.Subscription)
		dst.Subscription = &sub
	}
	return dst
}

// cloneSnapshot 深拷贝插件快照，保护 registry 内部状态不被外部修改。
func cloneSnapshot(src protocol.PluginSnapshot) protocol.PluginSnapshot {
	dst := src
	dst.Capabilities = make([]protocol.Capability, 0, len(src.Capabilities))
	for _, capability := range src.Capabilities {
		dst.Capabilities = append(dst.Capabilities, cloneCapability(capability))
	}
	dst.Permissions = append([]string(nil), src.Permissions...)
	dst.Hooks = append([]string(nil), src.Hooks...)
	dst.Metadata = cloneStringMap(src.Metadata)
	return dst
}

// cloneCapability 深拷贝 capability 中的权限和 schema 字段。
func cloneCapability(src protocol.Capability) protocol.Capability {
	dst := src
	dst.Permissions = append([]string(nil), src.Permissions...)
	dst.InputSchema = append(json.RawMessage(nil), src.InputSchema...)
	dst.OutputSchema = append(json.RawMessage(nil), src.OutputSchema...)
	return dst
}

// cloneSubscription 深拷贝订阅过滤条件。
func cloneSubscription(src Subscription) Subscription {
	dst := src
	dst.Filters = cloneStringMap(src.Filters)
	return dst
}

// cloneStringMap 复制字符串 map；空 map 统一返回 nil 以减少无意义分配。
func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

// sortSnapshots 提供稳定的插件实例输出顺序。
func sortSnapshots(items []protocol.PluginSnapshot) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].PluginID != items[j].PluginID {
			return items[i].PluginID < items[j].PluginID
		}
		return items[i].InstanceID < items[j].InstanceID
	})
}

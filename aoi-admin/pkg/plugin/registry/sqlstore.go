package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

// SQLStore 是基于关系型数据库的插件注册中心实现。
// 它保存插件实例、能力索引和事件订阅，适合多进程或需要持久化恢复的部署。
type SQLStore struct {
	db      *sql.DB
	dialect string
}

// SQLOption 配置 SQLStore 的方言等可选行为。
type SQLOption func(*SQLStore)

// NewSQLStore 创建 SQL registry 实例。
// db 由组合根管理生命周期；options 目前主要用于指定占位符方言。
func NewSQLStore(db *sql.DB, options ...SQLOption) *SQLStore {
	store := &SQLStore{db: db}
	for _, option := range options {
		if option != nil {
			option(store)
		}
	}
	store.dialect = strings.ToLower(strings.TrimSpace(store.dialect))
	return store
}

// WithDialect 设置 SQL 方言。
// dialect 为 postgres 时会把 ? 占位符转换为 $1、$2 等 PostgreSQL 格式。
func WithDialect(dialect string) SQLOption {
	return func(store *SQLStore) {
		store.dialect = dialect
	}
}

// RegisterInstance 在数据库中登记或替换插件实例快照。
// snapshot 必须包含插件和实例标识；副作用是在事务内重写实例主表和能力索引表。
func (s *SQLStore) RegisterInstance(ctx context.Context, snapshot protocol.PluginSnapshot) (protocol.PluginSnapshot, error) {
	if err := validateSnapshot(snapshot); err != nil {
		return protocol.PluginSnapshot{}, err
	}
	if s == nil || s.db == nil {
		return protocol.PluginSnapshot{}, ErrInvalid
	}
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = snapshot.RegisteredAt
	}
	if snapshot.UpdatedAt.IsZero() {
		snapshot.UpdatedAt = snapshot.LastHeartbeatAt
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return protocol.PluginSnapshot{}, err
	}
	if err := s.replaceInstance(ctx, tx, snapshot); err != nil {
		_ = tx.Rollback()
		return protocol.PluginSnapshot{}, err
	}
	if err := tx.Commit(); err != nil {
		return protocol.PluginSnapshot{}, err
	}
	return s.GetInstance(ctx, snapshot.PluginID, snapshot.InstanceID)
}

// RenewLease 刷新插件实例租约并返回最新数据库快照。
// 只更新租约相关字段，避免心跳请求覆盖插件注册时提交的能力和 metadata。
func (s *SQLStore) RenewLease(ctx context.Context, lease Lease) (protocol.PluginSnapshot, error) {
	if s == nil || s.db == nil {
		return protocol.PluginSnapshot{}, ErrInvalid
	}
	lease.LastHeartbeatAt = utcOrNow(lease.LastHeartbeatAt)
	lease.ExpiresAt = utcOrNow(lease.ExpiresAt)
	args := []any{protocol.StatusOnline, lease.LastHeartbeatAt, lease.ExpiresAt, int(lease.LeaseTTL.Seconds()), lease.LastHeartbeatAt, lease.PluginID, lease.InstanceID}
	result, err := s.db.ExecContext(ctx, s.bind("UPDATE plugin_instances SET status = ?, last_heartbeat_at = ?, lease_expires_at = ?, lease_ttl_seconds = ?, updated_at = ? WHERE plugin_id = ? AND instance_id = ?", len(args)), args...)
	if err != nil {
		return protocol.PluginSnapshot{}, err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return protocol.PluginSnapshot{}, ErrNotFound
	}
	return s.GetInstance(ctx, lease.PluginID, lease.InstanceID)
}

// UnregisterInstance 将插件实例标记为 offline。
// 记录会保留在数据库中，用于后续审计、健康展示和 stale 状态判断。
func (s *SQLStore) UnregisterInstance(ctx context.Context, pluginID string, instanceID string, now time.Time) error {
	if s == nil || s.db == nil {
		return ErrInvalid
	}
	now = utcOrNow(now)
	args := []any{protocol.StatusOffline, "", now, now, pluginID, instanceID}
	result, err := s.db.ExecContext(ctx, s.bind("UPDATE plugin_instances SET status = ?, runtime_status = ?, lease_expires_at = ?, updated_at = ? WHERE plugin_id = ? AND instance_id = ?", len(args)), args...)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ReportStatus 合并插件主动上报的状态和 metadata。
// 该方法先读取当前快照再走 SyncMetadata，保证未上报字段不会被清空。
func (s *SQLStore) ReportStatus(ctx context.Context, pluginID string, instanceID string, status string, runtimeStatus string, metadata map[string]string, now time.Time) (protocol.PluginSnapshot, error) {
	if s == nil || s.db == nil {
		return protocol.PluginSnapshot{}, ErrInvalid
	}
	item, err := s.GetInstance(ctx, pluginID, instanceID)
	if err != nil {
		return protocol.PluginSnapshot{}, err
	}
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
	return s.SyncMetadata(ctx, item)
}

// SyncMetadata 用新快照替换数据库中的插件元数据。
// 缺失的生命周期字段会从现有记录回填，避免插件只同步 metadata 时丢失租约状态。
func (s *SQLStore) SyncMetadata(ctx context.Context, snapshot protocol.PluginSnapshot) (protocol.PluginSnapshot, error) {
	if err := validateSnapshot(snapshot); err != nil {
		return protocol.PluginSnapshot{}, err
	}
	if s == nil || s.db == nil {
		return protocol.PluginSnapshot{}, ErrInvalid
	}
	existing, err := s.GetInstance(ctx, snapshot.PluginID, snapshot.InstanceID)
	if err != nil {
		return protocol.PluginSnapshot{}, err
	}
	if snapshot.Status == "" {
		snapshot.Status = existing.Status
	}
	if snapshot.RuntimeStatus == "" {
		snapshot.RuntimeStatus = existing.RuntimeStatus
	}
	if snapshot.OwnerHost == "" {
		snapshot.OwnerHost = existing.OwnerHost
	}
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
		snapshot.UpdatedAt = time.Now().UTC()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return protocol.PluginSnapshot{}, err
	}
	if err := s.replaceInstance(ctx, tx, snapshot); err != nil {
		_ = tx.Rollback()
		return protocol.PluginSnapshot{}, err
	}
	if err := tx.Commit(); err != nil {
		return protocol.PluginSnapshot{}, err
	}
	return s.GetInstance(ctx, snapshot.PluginID, snapshot.InstanceID)
}

// GetInstance 按插件实例标识读取单条快照。
// 当前实现复用 listInstances 的解码逻辑，确保单查和列表返回结构完全一致。
func (s *SQLStore) GetInstance(ctx context.Context, pluginID string, instanceID string) (protocol.PluginSnapshot, error) {
	items, err := s.listInstances(ctx)
	if err != nil {
		return protocol.PluginSnapshot{}, err
	}
	for _, item := range items {
		if item.PluginID == pluginID && item.InstanceID == instanceID {
			return item, nil
		}
	}
	return protocol.PluginSnapshot{}, ErrNotFound
}

// ListInstances 读取全部实例后在服务层应用统一过滤规则。
// 这样内存实现和 SQL 实现共享过期、状态、能力等过滤语义。
func (s *SQLStore) ListInstances(ctx context.Context, filter InstanceFilter) ([]protocol.PluginSnapshot, error) {
	items, err := s.listInstances(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]protocol.PluginSnapshot, 0, len(items))
	for _, item := range items {
		if matches(item, filter) {
			out = append(out, item)
		}
	}
	sortSnapshots(out)
	return out, nil
}

// ListByCapability 是能力过滤的便捷入口。
func (s *SQLStore) ListByCapability(ctx context.Context, capability string, filter InstanceFilter) ([]protocol.PluginSnapshot, error) {
	filter.Capability = capability
	return s.ListInstances(ctx, filter)
}

// ListExpired 返回租约已经到期但仍标记为 online 的实例。
func (s *SQLStore) ListExpired(ctx context.Context, now time.Time) ([]protocol.PluginSnapshot, error) {
	now = utcOrNow(now)
	items, err := s.ListInstances(ctx, InstanceFilter{IncludeExpired: true})
	if err != nil {
		return nil, err
	}
	var out []protocol.PluginSnapshot
	for _, item := range items {
		if item.Status == protocol.StatusOnline && !item.LeaseExpiresAt.IsZero() && !item.LeaseExpiresAt.After(now) {
			out = append(out, item)
		}
	}
	sortSnapshots(out)
	return out, nil
}

// ExpireLeases 将过期实例批量标记为 offline。
// 返回值保留过期前读取到的快照，方便上层记录或广播过期事件。
func (s *SQLStore) ExpireLeases(ctx context.Context, now time.Time) ([]protocol.PluginSnapshot, error) {
	expired, err := s.ListExpired(ctx, now)
	if err != nil {
		return nil, err
	}
	now = utcOrNow(now)
	for _, item := range expired {
		args := []any{protocol.StatusOffline, now, item.PluginID, item.InstanceID}
		if _, err := s.db.ExecContext(ctx, s.bind("UPDATE plugin_instances SET status = ?, updated_at = ? WHERE plugin_id = ? AND instance_id = ?", len(args)), args...); err != nil {
			return nil, err
		}
	}
	return expired, nil
}

// SubscribeEvent 持久化插件实例的事件订阅。
// 为保持幂等，事务中先删除同 key 订阅再插入新记录。
func (s *SQLStore) SubscribeEvent(ctx context.Context, sub Subscription) (Subscription, error) {
	if s == nil || s.db == nil {
		return Subscription{}, ErrInvalid
	}
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
	filters, err := marshalStringMap(sub.Filters)
	if err != nil {
		return Subscription{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Subscription{}, err
	}
	args := []any{sub.PluginID, sub.InstanceID, sub.Event}
	if _, err := tx.ExecContext(ctx, s.bind("DELETE FROM plugin_event_subscriptions WHERE plugin_id = ? AND instance_id = ? AND event = ?", len(args)), args...); err != nil {
		_ = tx.Rollback()
		return Subscription{}, err
	}
	args = []any{sub.PluginID, sub.InstanceID, sub.Event, filters, sub.CreatedAt, sub.UpdatedAt}
	if _, err := tx.ExecContext(ctx, s.bind("INSERT INTO plugin_event_subscriptions (plugin_id, instance_id, event, filters_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)", len(args)), args...); err != nil {
		_ = tx.Rollback()
		return Subscription{}, err
	}
	if err := tx.Commit(); err != nil {
		return Subscription{}, err
	}
	return sub, nil
}

// ListSubscriptions 读取事件订阅并在内存中应用过滤条件。
// filters_json 会被解码为 map；解码失败时按空过滤处理，避免单条坏数据拖垮列表。
func (s *SQLStore) ListSubscriptions(ctx context.Context, filter SubscriptionFilter) ([]Subscription, error) {
	if s == nil || s.db == nil {
		return nil, ErrInvalid
	}
	rows, err := s.db.QueryContext(ctx, "SELECT plugin_id, instance_id, event, filters_json, created_at, updated_at FROM plugin_event_subscriptions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Subscription
	for rows.Next() {
		var sub Subscription
		var filters string
		if err := rows.Scan(&sub.PluginID, &sub.InstanceID, &sub.Event, &filters, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			return nil, err
		}
		sub.Filters = unmarshalStringMap(filters)
		if filter.PluginID != "" && sub.PluginID != filter.PluginID {
			continue
		}
		if filter.InstanceID != "" && sub.InstanceID != filter.InstanceID {
			continue
		}
		if filter.Event != "" && sub.Event != filter.Event {
			continue
		}
		out = append(out, cloneSubscription(sub))
	}
	return out, rows.Err()
}

// Watch 使用 PollWatcher 为 SQLStore 提供变化订阅能力。
// SQLStore 本身不维护推送通道，轮询实现可以兼容不同数据库。
func (s *SQLStore) Watch(ctx context.Context, options WatchOptions) (<-chan Change, error) {
	return NewPollWatcher(s).Watch(ctx, options)
}

// replaceInstance 在事务中重写实例快照及其能力索引。
// 采用先删后插是为了让 capability 列表与当前快照完全一致，避免旧能力残留。
func (s *SQLStore) replaceInstance(ctx context.Context, tx *sql.Tx, snapshot protocol.PluginSnapshot) error {
	args := []any{snapshot.PluginID, snapshot.InstanceID}
	if _, err := tx.ExecContext(ctx, s.bind("DELETE FROM plugin_instance_capabilities WHERE plugin_id = ? AND instance_id = ?", len(args)), args...); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, s.bind("DELETE FROM plugin_instances WHERE plugin_id = ? AND instance_id = ?", len(args)), args...); err != nil {
		return err
	}
	capabilities, err := json.Marshal(snapshot.Capabilities)
	if err != nil {
		return err
	}
	permissions, err := marshalStrings(snapshot.Permissions)
	if err != nil {
		return err
	}
	hooks, err := marshalStrings(snapshot.Hooks)
	if err != nil {
		return err
	}
	metadata, err := marshalStringMap(snapshot.Metadata)
	if err != nil {
		return err
	}
	transport := protocol.EffectiveTransport(snapshot.PluginMetadata)
	args = []any{
		snapshot.PluginID, snapshot.InstanceID, snapshot.Name, snapshot.Version, snapshot.Protocol, transport, snapshot.Endpoint,
		snapshot.Status, snapshot.RuntimeStatus, snapshot.SchemaVersion, snapshot.OwnerHost, snapshot.LeaseTTLSeconds,
		snapshot.LeaseExpiresAt, snapshot.RegisteredAt, snapshot.LastHeartbeatAt, snapshot.CreatedAt, snapshot.UpdatedAt,
		string(permissions), string(hooks), string(metadata), string(capabilities),
	}
	if _, err := tx.ExecContext(ctx, s.bind(`INSERT INTO plugin_instances (
plugin_id, instance_id, name, version, protocol, transport, endpoint, status, runtime_status, schema_version, owner_host,
lease_ttl_seconds, lease_expires_at, registered_at, last_heartbeat_at, created_at, updated_at,
permissions_json, hooks_json, metadata_json, capabilities_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, len(args)), args...); err != nil {
		return err
	}
	for _, capability := range snapshot.Capabilities {
		capPerms, err := marshalStrings(capability.Permissions)
		if err != nil {
			return err
		}
		args = []any{
			snapshot.PluginID, snapshot.InstanceID, capability.Name, capability.Version, capability.Scope,
			string(capPerms), string(capability.InputSchema), string(capability.OutputSchema),
			capability.SecretPolicy, capability.Description, snapshot.UpdatedAt,
		}
		if _, err := tx.ExecContext(ctx, s.bind(`INSERT INTO plugin_instance_capabilities (
plugin_id, instance_id, capability, capability_version, scope, permissions_json,
input_schema, output_schema, secret_policy, description, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, len(args)), args...); err != nil {
			return err
		}
	}
	return nil
}

// listInstances 从数据库读取实例主表并还原协议快照。
// permissions、hooks、metadata 和 capabilities 以 JSON 存储，以兼容不同 SQL 方言的简单 schema。
func (s *SQLStore) listInstances(ctx context.Context) ([]protocol.PluginSnapshot, error) {
	if s == nil || s.db == nil {
		return nil, ErrInvalid
	}
	rows, err := s.db.QueryContext(ctx, `SELECT plugin_id, instance_id, name, version, protocol, transport, endpoint,
status, runtime_status, schema_version, owner_host, lease_ttl_seconds, lease_expires_at,
registered_at, last_heartbeat_at, created_at, updated_at, permissions_json, hooks_json, metadata_json, capabilities_json
FROM plugin_instances`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []protocol.PluginSnapshot
	for rows.Next() {
		var item protocol.PluginSnapshot
		var permissions, hooks, metadata, capabilities string
		if err := rows.Scan(
			&item.PluginID, &item.InstanceID, &item.Name, &item.Version, &item.Protocol, &item.Transport, &item.Endpoint,
			&item.Status, &item.RuntimeStatus, &item.SchemaVersion, &item.OwnerHost, &item.LeaseTTLSeconds, &item.LeaseExpiresAt,
			&item.RegisteredAt, &item.LastHeartbeatAt, &item.CreatedAt, &item.UpdatedAt,
			&permissions, &hooks, &metadata, &capabilities,
		); err != nil {
			return nil, err
		}
		item.Permissions = unmarshalStrings(permissions)
		item.Hooks = unmarshalStrings(hooks)
		item.Metadata = unmarshalStringMap(metadata)
		if strings.TrimSpace(capabilities) != "" {
			_ = json.Unmarshal([]byte(capabilities), &item.Capabilities)
		}
		out = append(out, cloneSnapshot(item))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sortSnapshots(out)
	return out, nil
}

// bind 根据方言转换 SQL 占位符。
// 默认保留 ?，PostgreSQL 下按参数顺序替换为 $1、$2，避免每条语句维护两份文本。
func (s *SQLStore) bind(query string, count int) string {
	if s == nil || s.dialect != "postgres" {
		return query
	}
	out := strings.Builder{}
	index := 1
	for _, r := range query {
		if r == '?' && index <= count {
			out.WriteString(fmt.Sprintf("$%d", index))
			index++
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

// marshalStrings 将字符串列表编码为 JSON；nil 会落为 []，便于数据库字段保持非空语义。
func marshalStrings(values []string) ([]byte, error) {
	if values == nil {
		values = []string{}
	}
	return json.Marshal(values)
}

// unmarshalStrings 将 JSON 字符串列表还原为切片。
// 解析失败时返回零值，避免历史脏数据影响主查询流程。
func unmarshalStrings(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

// marshalStringMap 将字符串 map 编码为 JSON；nil 会落为 {}。
func marshalStringMap(values map[string]string) ([]byte, error) {
	if values == nil {
		values = map[string]string{}
	}
	return json.Marshal(values)
}

// unmarshalStringMap 将 JSON map 还原为 Go map。
// 空对象统一返回 nil，减少调用方对空 map 和 nil 的双重处理。
func unmarshalStringMap(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out map[string]string
	_ = json.Unmarshal([]byte(raw), &out)
	if len(out) == 0 {
		return nil
	}
	return out
}

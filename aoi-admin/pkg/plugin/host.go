// Package plugin 实现远程插件 Host，统一编排注册、租约、路由、事件、注入和观测。
package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rei0721/go-scaffold/pkg/plugin/event"
	"github.com/rei0721/go-scaffold/pkg/plugin/injection"
	"github.com/rei0721/go-scaffold/pkg/plugin/observability"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
	"github.com/rei0721/go-scaffold/pkg/plugin/registry"
	pluginrouter "github.com/rei0721/go-scaffold/pkg/plugin/router"
	"github.com/rei0721/go-scaffold/pkg/plugin/security"
)

// Config 描述插件 Host 的运行依赖和策略。
//
// Registry、EventBus、Authorizer、Recorder 等字段允许组合根替换存储、事件、安全和观测实现；
// LeaseTTL 与 LeaseScanInterval 控制远程实例生命周期；AllowedTransports 限制插件可注册的 transport。
type Config struct {
	Enabled                bool
	HeartbeatTimeout       time.Duration
	RequestTimeout         time.Duration
	DefaultProtocolVersion string
	AllowedTransports      []string
	InjectionEnabled       bool
	Injection              *injection.Registry
	Registry               registry.Registry
	EventBus               event.Bus
	Authorizer             security.Authorizer
	Recorder               observability.Recorder
	NodeID                 string
	NodeAddress            string
	LeaseTTL               time.Duration
	LeaseScanInterval      time.Duration
	RetryCount             int
	RouterStrategy         string
	Now                    func() time.Time
}

// Provider 是主系统本地暴露给插件调用的能力实现。
//
// Capability 声明能力元数据，Invoke 执行实际逻辑；Host 会在调用前统一做能力校验和授权。
type Provider interface {
	Capability() protocol.Capability
	Invoke(context.Context, protocol.InvokeRequest) (json.RawMessage, error)
}

// ProviderFunc 让普通函数可以注册为本地插件能力 provider。
type ProviderFunc struct {
	Definition protocol.Capability
	Handler    func(context.Context, protocol.InvokeRequest) (json.RawMessage, error)
}

// Capability 返回 provider 声明的能力定义。
func (p ProviderFunc) Capability() protocol.Capability {
	return p.Definition
}

// Invoke 调用底层处理函数执行能力。
func (p ProviderFunc) Invoke(ctx context.Context, req protocol.InvokeRequest) (json.RawMessage, error) {
	if p.Handler == nil {
		return nil, ErrProviderUnavailable
	}
	return p.Handler(ctx, req)
}

// RemoteInvoker 是 router 侧 transport 调用器的类型别名，供 Host 装配时直接暴露。
type RemoteInvoker = pluginrouter.RemoteInvoker

// RemoteInvokerFunc 让普通函数可以作为远程 transport invoker 注册。
type RemoteInvokerFunc struct {
	InvokeFunc    func(context.Context, protocol.PluginSnapshot, protocol.InvokeRequest) (json.RawMessage, error)
	PushEventFunc func(context.Context, protocol.PluginSnapshot, protocol.PushEventRequest) error
}

// Invoke 通过底层函数调用远程插件能力。
func (f RemoteInvokerFunc) Invoke(ctx context.Context, plugin protocol.PluginSnapshot, req protocol.InvokeRequest) (json.RawMessage, error) {
	if f.InvokeFunc == nil {
		return nil, ErrTransportUnavailable
	}
	return f.InvokeFunc(ctx, plugin, req)
}

// PushEvent 通过底层函数向远程插件推送事件。
func (f RemoteInvokerFunc) PushEvent(ctx context.Context, plugin protocol.PluginSnapshot, req protocol.PushEventRequest) error {
	if f.PushEventFunc == nil {
		return ErrTransportUnavailable
	}
	return f.PushEventFunc(ctx, plugin, req)
}

// Host 是远程插件系统的进程内编排器。
//
// 它不直接实现业务能力，而是协调 registry、router、event bus、security、injection 和 observability，
// 并把跨 transport 的协议请求转换为稳定的插件生命周期操作。
type Host struct {
	mu                     sync.RWMutex
	enabled                bool
	heartbeatTimeout       time.Duration
	requestTimeout         time.Duration
	defaultProtocolVersion string
	allowedTransports      []string
	injectionEnabled       bool
	injection              *injection.Registry
	registry               registry.Registry
	router                 *pluginrouter.Router
	eventBus               event.Bus
	authorizer             security.Authorizer
	recorder               observability.Recorder
	nodeID                 string
	nodeAddress            string
	leaseTTL               time.Duration
	leaseScanInterval      time.Duration
	now                    func() time.Time
	providers              map[string]Provider
}

// New 创建插件 Host 并补齐默认依赖。
//
// 该函数只完成内存对象装配，不启动后台 goroutine；租约清理需要调用 RunLeaseReaper 或由外部调度触发。
func New(cfg Config) *Host {
	if cfg.HeartbeatTimeout <= 0 {
		cfg.HeartbeatTimeout = 30 * time.Second
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 10 * time.Second
	}
	if cfg.LeaseTTL <= 0 {
		cfg.LeaseTTL = cfg.HeartbeatTimeout
	}
	if cfg.LeaseScanInterval <= 0 {
		cfg.LeaseScanInterval = cfg.LeaseTTL / 2
	}
	if cfg.LeaseScanInterval <= 0 {
		cfg.LeaseScanInterval = 15 * time.Second
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if strings.TrimSpace(cfg.DefaultProtocolVersion) == "" {
		cfg.DefaultProtocolVersion = protocol.ProtocolVersionV1
	}
	if len(cfg.AllowedTransports) == 0 {
		cfg.AllowedTransports = []string{protocol.TransportHTTP, protocol.TransportWebSocket}
	}
	reg := cfg.Registry
	if reg == nil {
		reg = registry.NewMemory()
	}
	r := pluginrouter.New(pluginrouter.Config{
		Registry:       reg,
		RequestTimeout: cfg.RequestTimeout,
		RetryCount:     cfg.RetryCount,
		Strategy:       cfg.RouterStrategy,
		Now:            cfg.Now,
	})
	bus := cfg.EventBus
	if bus == nil {
		bus = event.NewDirectBus(reg, r)
	}
	recorder := cfg.Recorder
	if recorder == nil {
		recorder = observability.NopRecorder{}
	}
	return &Host{
		enabled:                cfg.Enabled,
		heartbeatTimeout:       cfg.HeartbeatTimeout,
		requestTimeout:         cfg.RequestTimeout,
		defaultProtocolVersion: strings.TrimSpace(cfg.DefaultProtocolVersion),
		allowedTransports:      normalizedStrings(cfg.AllowedTransports),
		injectionEnabled:       cfg.InjectionEnabled,
		injection:              cfg.Injection,
		registry:               reg,
		router:                 r,
		eventBus:               bus,
		authorizer:             cfg.Authorizer,
		recorder:               recorder,
		nodeID:                 firstNonEmpty(cfg.NodeID, cfg.NodeAddress, "plugin-host"),
		nodeAddress:            strings.TrimSpace(cfg.NodeAddress),
		leaseTTL:               cfg.LeaseTTL,
		leaseScanInterval:      cfg.LeaseScanInterval,
		now:                    func() time.Time { return cfg.Now().UTC() },
		providers:              map[string]Provider{},
	}
}

// Enabled 返回当前 Host 是否允许处理插件协议请求。
func (h *Host) Enabled() bool {
	if h == nil {
		return false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.enabled
}

// Register 校验并登记一个远程插件实例。
//
// req.Plugin 提供插件元数据和能力声明；返回值包含规范化后的实例快照。副作用是写入 registry，
// 并通过 deferred recorder 记录一次注册观测事件。
func (h *Host) Register(ctx context.Context, req protocol.RegisterRequest) (resp protocol.RegisterResponse, err error) {
	started := h.observationStart()
	defer func() {
		metadata := req.Plugin
		if resp.Plugin.PluginID != "" {
			metadata = resp.Plugin.PluginMetadata
		}
		h.recordOperation(ctx, protocol.OperationRegister, req.RequestMeta, metadata, "", "", started, err)
	}()
	if h == nil || !h.Enabled() {
		return protocol.RegisterResponse{}, ErrDisabled
	}
	instance, err := CreatePluginApp(req, InstanceContext{
		Now:                    h.now(),
		DefaultProtocolVersion: h.defaultProtocolVersion,
		Source:                 "plugin-host",
		OwnerHost:              h.nodeID,
		LeaseTTL:               h.leaseTTL,
	})
	if err != nil {
		return protocol.RegisterResponse{}, err
	}
	snapshot := instance.Snapshot
	// transport 白名单在写入 registry 前检查，避免不可调用实例进入共享状态层。
	if !contains(h.allowedTransports, protocol.EffectiveTransport(snapshot.PluginMetadata)) {
		return protocol.RegisterResponse{}, ErrUnsupportedProtocol
	}
	if h.authorizer != nil {
		if err := h.authorize(ctx, protocol.OperationRegister, snapshot, registrationPermissions(snapshot), "", ""); err != nil {
			return protocol.RegisterResponse{}, err
		}
	}
	registered, err := h.registry.RegisterInstance(ctx, snapshot)
	if err != nil {
		return protocol.RegisterResponse{}, mapRegistryError(err)
	}
	return protocol.RegisterResponse{Plugin: cloneSnapshot(registered)}, nil
}

// Heartbeat 将旧版心跳语义复用为租约续期。
//
// 返回值只保留插件快照，方便协议层保持 heartbeat 与 renew_lease 两个操作的兼容性。
func (h *Host) Heartbeat(ctx context.Context, req protocol.HeartbeatRequest) (protocol.HeartbeatResponse, error) {
	response, err := h.RenewLease(ctx, protocol.RenewLeaseRequest{
		RequestMeta: req.RequestMeta,
		PluginID:    req.PluginID,
		InstanceID:  req.InstanceID,
	})
	if err != nil {
		return protocol.HeartbeatResponse{}, err
	}
	return protocol.HeartbeatResponse{Plugin: response.Plugin}, nil
}

// RenewLease 刷新插件实例租约并返回最新快照。
//
// req.PluginID 和 req.InstanceID 标识目标实例；LeaseTTLSeconds 为空时使用 Host 默认 TTL。
// 副作用是更新 registry 中的心跳时间、过期时间和在线状态。
func (h *Host) RenewLease(ctx context.Context, req protocol.RenewLeaseRequest) (resp protocol.RenewLeaseResponse, err error) {
	started := h.observationStart()
	defer func() {
		metadata := protocol.PluginMetadata{PluginID: req.PluginID, InstanceID: req.InstanceID}
		if resp.Plugin.PluginID != "" {
			metadata = resp.Plugin.PluginMetadata
		}
		h.recordOperation(ctx, protocol.OperationRenewLease, req.RequestMeta, metadata, "", "", started, err)
	}()
	if h == nil || !h.Enabled() {
		return protocol.RenewLeaseResponse{}, ErrDisabled
	}
	if strings.TrimSpace(req.PluginID) == "" || strings.TrimSpace(req.InstanceID) == "" {
		return protocol.RenewLeaseResponse{}, ErrInvalidPlugin
	}
	ttl := h.leaseTTL
	if req.LeaseTTLSeconds > 0 {
		ttl = time.Duration(req.LeaseTTLSeconds) * time.Second
	}
	now := h.now()
	plugin, err := h.registry.RenewLease(ctx, registry.Lease{
		PluginID:        req.PluginID,
		InstanceID:      req.InstanceID,
		LeaseTTL:        ttl,
		LastHeartbeatAt: now,
		ExpiresAt:       now.Add(ttl),
	})
	if err != nil {
		return protocol.RenewLeaseResponse{}, mapRegistryError(err)
	}
	return protocol.RenewLeaseResponse{Plugin: cloneSnapshot(plugin)}, nil
}

// Unregister 将插件实例标记为离线。
//
// 注销按幂等语义处理：registry 中不存在同一实例时仍返回 offline 结果，避免远程插件重复注销被迫重试。
func (h *Host) Unregister(ctx context.Context, req protocol.UnregisterRequest) (resp protocol.UnregisterResponse, err error) {
	started := h.observationStart()
	defer func() {
		h.recordOperation(ctx, protocol.OperationUnregister, req.RequestMeta, protocol.PluginMetadata{PluginID: req.PluginID, InstanceID: req.InstanceID}, "", "", started, err)
	}()
	if h == nil || !h.Enabled() {
		return protocol.UnregisterResponse{}, ErrDisabled
	}
	if strings.TrimSpace(req.PluginID) == "" || strings.TrimSpace(req.InstanceID) == "" {
		return protocol.UnregisterResponse{}, ErrInvalidPlugin
	}
	if err := h.registry.UnregisterInstance(ctx, req.PluginID, req.InstanceID, h.now()); err != nil && !errors.Is(err, registry.ErrNotFound) {
		return protocol.UnregisterResponse{}, mapRegistryError(err)
	}
	return protocol.UnregisterResponse{PluginID: req.PluginID, InstanceID: req.InstanceID, Status: protocol.StatusOffline}, nil
}

// ListPlugins 返回当前 registry 中所有插件实例快照。
//
// 查询前会尽力清理过期租约；结果包含已过期或离线实例，便于管理端展示完整状态。
func (h *Host) ListPlugins(ctx context.Context) ([]protocol.PluginSnapshot, error) {
	if h == nil || !h.Enabled() {
		return nil, ErrDisabled
	}
	h.expireLeases(ctx)
	items, err := h.registry.ListInstances(ctx, registry.InstanceFilter{IncludeExpired: true, Now: h.now()})
	if err != nil {
		return nil, mapRegistryError(err)
	}
	sortSnapshots(items)
	return items, nil
}

// GetPlugin 返回某个 plugin_id 下的代表性实例快照。
//
// 如果存在在线实例优先返回在线实例，否则返回排序后的第一个历史实例，方便管理端查看离线插件信息。
func (h *Host) GetPlugin(ctx context.Context, id string) (protocol.PluginSnapshot, error) {
	if h == nil || !h.Enabled() {
		return protocol.PluginSnapshot{}, ErrDisabled
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return protocol.PluginSnapshot{}, ErrInvalidPlugin
	}
	h.expireLeases(ctx)
	items, err := h.registry.ListInstances(ctx, registry.InstanceFilter{PluginID: id, IncludeExpired: true, Now: h.now()})
	if err != nil {
		return protocol.PluginSnapshot{}, mapRegistryError(err)
	}
	if len(items) == 0 {
		return protocol.PluginSnapshot{}, ErrPluginNotFound
	}
	sortSnapshots(items)
	for _, item := range items {
		if item.Status == protocol.StatusOnline {
			return cloneSnapshot(item), nil
		}
	}
	return cloneSnapshot(items[0]), nil
}

// GetPluginInstance 返回指定插件实例快照。
func (h *Host) GetPluginInstance(ctx context.Context, pluginID string, instanceID string) (protocol.PluginSnapshot, error) {
	if h == nil || !h.Enabled() {
		return protocol.PluginSnapshot{}, ErrDisabled
	}
	h.expireLeases(ctx)
	item, err := h.registry.GetInstance(ctx, pluginID, instanceID)
	if err != nil {
		return protocol.PluginSnapshot{}, mapRegistryError(err)
	}
	return cloneSnapshot(item), nil
}

// ExpireLeases 将已过期的远程插件实例标记为离线。
func (h *Host) ExpireLeases(ctx context.Context) ([]protocol.PluginSnapshot, error) {
	if h == nil || h.registry == nil {
		return nil, ErrDisabled
	}
	expired, err := h.registry.ExpireLeases(ctx, h.now())
	if err != nil {
		return nil, mapRegistryError(err)
	}
	return expired, nil
}

// WatchRegistry 监听插件注册中心状态变化；底层 registry 没有原生 watcher 时使用轮询实现。
func (h *Host) WatchRegistry(ctx context.Context, options registry.WatchOptions) (<-chan registry.Change, error) {
	if h == nil || h.registry == nil {
		return nil, ErrDisabled
	}
	if watcher, ok := h.registry.(registry.Watcher); ok {
		return watcher.Watch(ctx, options)
	}
	return registry.NewPollWatcher(h.registry).Watch(ctx, options)
}

// RunLeaseReaper 按固定间隔清理过期租约，调用方负责用 context 控制生命周期。
func (h *Host) RunLeaseReaper(ctx context.Context, interval time.Duration) {
	if h == nil || !h.Enabled() {
		return
	}
	if interval <= 0 {
		interval = h.leaseScanInterval
	}
	if interval <= 0 {
		interval = 15 * time.Second
	}
	_, _ = h.ExpireLeases(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = h.ExpireLeases(ctx)
		}
	}
}

// Health 返回插件维度的健康状态。
//
// id 是 plugin_id；实现会复用 GetPlugin 的代表性实例选择逻辑，因此多实例插件优先反映在线实例状态。
func (h *Host) Health(ctx context.Context, id string) (protocol.HealthStatus, error) {
	plugin, err := h.GetPlugin(ctx, id)
	if err != nil {
		return protocol.HealthStatus{}, err
	}
	return healthStatus(plugin, h.now()), nil
}

// HealthInstance 返回指定插件实例的健康状态。
func (h *Host) HealthInstance(ctx context.Context, pluginID string, instanceID string) (protocol.HealthStatus, error) {
	plugin, err := h.GetPluginInstance(ctx, pluginID, instanceID)
	if err != nil {
		return protocol.HealthStatus{}, err
	}
	return healthStatus(plugin, h.now()), nil
}

// RegisterProvider 注册主系统本地能力 provider。
//
// provider.Capability 会被规范化并检查重名；副作用是写入 Host 内存 provider 表，不影响远程 registry。
func (h *Host) RegisterProvider(provider Provider) error {
	if h == nil {
		return ErrDisabled
	}
	if provider == nil {
		return ErrProviderUnavailable
	}
	capability, err := normalizeCapability(provider.Capability())
	if err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.providers[capability.Name]; ok {
		return fmt.Errorf("%w: %s", ErrInvalidCapability, capability.Name)
	}
	h.providers[capability.Name] = ProviderFunc{
		Definition: capability,
		Handler:    provider.Invoke,
	}
	return nil
}

// RegisterRemoteInvoker 为某种 transport 注册远程调用器。
//
// transport 名称由 protocol.EffectiveTransport 产生；注册后 router 才能把实例路由到该 transport。
func (h *Host) RegisterRemoteInvoker(transport string, invoker RemoteInvoker) error {
	if h == nil {
		return ErrDisabled
	}
	if h.router == nil {
		return ErrTransportUnavailable
	}
	return mapRouterError(h.router.RegisterRemoteInvoker(transport, invoker))
}

// RegisterInjectionProvider 注册一个上下文注入 provider。
//
// 如果 Host 构造时没有传入 injection registry，会按默认协议版本延迟创建一个。
func (h *Host) RegisterInjectionProvider(provider injection.Provider) error {
	if h == nil {
		return ErrDisabled
	}
	if h.injection == nil {
		h.injection = injection.NewRegistry(injection.Config{SchemaVersion: h.defaultProtocolVersion, Source: "plugin-host"})
	}
	return h.injection.Register(provider)
}

// ListProviders 返回主系统本地 provider 暴露的能力列表。
func (h *Host) ListProviders(context.Context) ([]protocol.Capability, error) {
	if h == nil || !h.Enabled() {
		return nil, ErrDisabled
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	items := make([]protocol.Capability, 0, len(h.providers))
	for _, provider := range h.providers {
		items = append(items, cloneCapability(provider.Capability()))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items, nil
}

// ListCapabilities 返回插件或本地 provider 的能力集合。
//
// req.PluginID 为空时列出主系统本地能力；指定 PluginID 时从 registry 聚合远程插件实例能力。
func (h *Host) ListCapabilities(ctx context.Context, req protocol.ListCapabilitiesRequest) (protocol.ListCapabilitiesResponse, error) {
	if strings.TrimSpace(req.PluginID) != "" {
		h.expireLeases(ctx)
		var items []protocol.PluginSnapshot
		var err error
		if strings.TrimSpace(req.InstanceID) != "" {
			var item protocol.PluginSnapshot
			item, err = h.registry.GetInstance(ctx, req.PluginID, req.InstanceID)
			items = []protocol.PluginSnapshot{item}
		} else {
			items, err = h.registry.ListInstances(ctx, registry.InstanceFilter{PluginID: req.PluginID, IncludeExpired: true, Now: h.now()})
		}
		if err != nil {
			return protocol.ListCapabilitiesResponse{}, mapRegistryError(err)
		}
		return protocol.ListCapabilitiesResponse{Capabilities: aggregateCapabilities(items)}, nil
	}
	providers, err := h.ListProviders(ctx)
	if err != nil {
		return protocol.ListCapabilitiesResponse{}, err
	}
	return protocol.ListCapabilitiesResponse{Capabilities: providers}, nil
}

// Invoke 执行插件能力调用。
//
// 未指定 PluginID 时会优先尝试本地 provider，再尝试远程 router，最后回退本地 provider；
// 指定 PluginID 时会把错误限定在远程调用路径，避免误调用同名本地能力。
func (h *Host) Invoke(ctx context.Context, req protocol.InvokeRequest) (resp protocol.InvokeResponse, err error) {
	started := h.observationStart()
	defer func() {
		h.recordOperation(ctx, protocol.OperationInvoke, req.RequestMeta, protocol.PluginMetadata{PluginID: req.PluginID, InstanceID: req.InstanceID}, req.Capability, "", started, err)
	}()
	if h == nil || !h.Enabled() {
		return protocol.InvokeResponse{}, ErrDisabled
	}
	capability := strings.TrimSpace(req.Capability)
	if capability == "" {
		return protocol.InvokeResponse{}, ErrInvalidCapability
	}
	if strings.TrimSpace(req.PluginID) == "" {
		if response, ok, err := h.invokeProvider(ctx, req); ok || err != nil {
			return response, err
		}
	}
	if h.router != nil {
		response, err := h.router.Invoke(ctx, req)
		if err == nil {
			return response, nil
		}
		// 明确指定远程插件时不再回退本地 provider，避免把远程离线误解释为本地能力成功。
		if strings.TrimSpace(req.PluginID) != "" {
			return protocol.InvokeResponse{}, mapRouterError(err)
		}
	}
	response, ok, err := h.invokeProvider(ctx, req)
	if ok || err != nil {
		return response, err
	}
	return protocol.InvokeResponse{}, ErrCapabilityNotFound
}

// PushEvent 发布远程插件事件。
//
// 有 EventBus 时委托事件总线处理订阅和投递；无 EventBus 且无 router 时返回 accepted，表示事件被 Host 接收但未投递。
func (h *Host) PushEvent(ctx context.Context, req protocol.PushEventRequest) (resp protocol.PushEventResponse, err error) {
	started := h.observationStart()
	defer func() {
		h.recordOperation(ctx, protocol.OperationPushEvent, req.RequestMeta, protocol.PluginMetadata{PluginID: req.PluginID, InstanceID: req.InstanceID}, "", req.Event, started, err)
	}()
	if h == nil || !h.Enabled() {
		return protocol.PushEventResponse{}, ErrDisabled
	}
	if strings.TrimSpace(req.Event) == "" {
		return protocol.PushEventResponse{}, ErrInvalidCapability
	}
	if h.eventBus != nil {
		response, err := h.eventBus.Publish(ctx, req)
		return response, mapRouterError(err)
	}
	if h.router == nil {
		return protocol.PushEventResponse{Accepted: true, Event: req.Event}, nil
	}
	response, err := h.router.PushEvent(ctx, req)
	return response, mapRouterError(err)
}

// SubscribeEvent 登记插件实例的事件订阅。
//
// 每个事件都会先通过 authorizer 校验插件权限，成功后交给 EventBus/registry 保存订阅关系。
func (h *Host) SubscribeEvent(ctx context.Context, req protocol.SubscribeEventRequest) (resp protocol.SubscribeEventResponse, err error) {
	started := h.observationStart()
	defer func() {
		h.recordOperation(ctx, protocol.OperationSubscribeEvent, req.RequestMeta, protocol.PluginMetadata{PluginID: req.PluginID, InstanceID: req.InstanceID}, "", strings.Join(req.Events, ","), started, err)
	}()
	if h == nil || !h.Enabled() {
		return protocol.SubscribeEventResponse{}, ErrDisabled
	}
	if strings.TrimSpace(req.PluginID) == "" || strings.TrimSpace(req.InstanceID) == "" || len(req.Events) == 0 {
		return protocol.SubscribeEventResponse{}, ErrInvalidPlugin
	}
	snapshot, err := h.registry.GetInstance(ctx, req.PluginID, req.InstanceID)
	if err != nil {
		return protocol.SubscribeEventResponse{}, mapRegistryError(err)
	}
	for _, eventName := range req.Events {
		if err := h.authorize(ctx, protocol.OperationSubscribeEvent, snapshot, snapshot.Permissions, "", eventName); err != nil {
			return protocol.SubscribeEventResponse{}, err
		}
	}
	if h.eventBus == nil {
		return protocol.SubscribeEventResponse{}, ErrTransportUnavailable
	}
	response, err := h.eventBus.Subscribe(ctx, req)
	return response, mapRegistryError(err)
}

// ReportStatus 接收插件实例主动上报的运行状态和附加 metadata。
//
// 副作用是更新 registry 中的 status/runtime_status/metadata/updated_at。
func (h *Host) ReportStatus(ctx context.Context, req protocol.ReportStatusRequest) (resp protocol.ReportStatusResponse, err error) {
	started := h.observationStart()
	defer func() {
		metadata := protocol.PluginMetadata{PluginID: req.PluginID, InstanceID: req.InstanceID}
		if resp.Plugin.PluginID != "" {
			metadata = resp.Plugin.PluginMetadata
		}
		h.recordOperation(ctx, protocol.OperationReportStatus, req.RequestMeta, metadata, "", "", started, err)
	}()
	if h == nil || !h.Enabled() {
		return protocol.ReportStatusResponse{}, ErrDisabled
	}
	plugin, err := h.registry.ReportStatus(ctx, req.PluginID, req.InstanceID, req.Status, req.RuntimeStatus, req.Metadata, h.now())
	if err != nil {
		return protocol.ReportStatusResponse{}, mapRegistryError(err)
	}
	return protocol.ReportStatusResponse{Plugin: cloneSnapshot(plugin)}, nil
}

// SyncMetadata 用插件提交的新元数据更新 registry 中已有实例。
//
// 该方法保留租约和运行状态，只替换可同步的 PluginMetadata，用于插件运行中更新能力或描述信息。
func (h *Host) SyncMetadata(ctx context.Context, req protocol.SyncMetadataRequest) (resp protocol.SyncMetadataResponse, err error) {
	started := h.observationStart()
	defer func() {
		metadata := req.Plugin
		if resp.Plugin.PluginID != "" {
			metadata = resp.Plugin.PluginMetadata
		}
		h.recordOperation(ctx, protocol.OperationSyncMetadata, req.RequestMeta, metadata, "", "", started, err)
	}()
	if h == nil || !h.Enabled() {
		return protocol.SyncMetadataResponse{}, ErrDisabled
	}
	metadata, err := normalizeMetadata(req.Plugin)
	if err != nil {
		return protocol.SyncMetadataResponse{}, err
	}
	existing, err := h.registry.GetInstance(ctx, metadata.PluginID, metadata.InstanceID)
	if err != nil {
		return protocol.SyncMetadataResponse{}, mapRegistryError(err)
	}
	// 先读取已有快照再替换元数据，避免同步请求覆盖 lease、status 等 Host 维护的运行态字段。
	existing.PluginMetadata = metadata
	existing.UpdatedAt = h.now()
	plugin, err := h.registry.SyncMetadata(ctx, existing)
	if err != nil {
		return protocol.SyncMetadataResponse{}, mapRegistryError(err)
	}
	return protocol.SyncMetadataResponse{Plugin: cloneSnapshot(plugin)}, nil
}

// Drain 将插件实例标记为 draining 状态。
//
// 该状态用于通知路由和管理端实例正在下线或维护，实际连接关闭仍由插件或运维流程完成。
func (h *Host) Drain(ctx context.Context, req protocol.DrainRequest) (resp protocol.DrainResponse, err error) {
	started := h.observationStart()
	defer func() {
		metadata := protocol.PluginMetadata{PluginID: req.PluginID, InstanceID: req.InstanceID}
		if resp.Plugin.PluginID != "" {
			metadata = resp.Plugin.PluginMetadata
		}
		h.recordOperation(ctx, protocol.OperationDrain, req.RequestMeta, metadata, "", "", started, err)
	}()
	if h == nil || !h.Enabled() {
		return protocol.DrainResponse{}, ErrDisabled
	}
	plugin, err := h.registry.ReportStatus(ctx, req.PluginID, req.InstanceID, protocol.StatusDraining, protocol.RuntimeStatusDraining, nil, h.now())
	if err != nil {
		return protocol.DrainResponse{}, mapRegistryError(err)
	}
	return protocol.DrainResponse{Plugin: cloneSnapshot(plugin)}, nil
}

// InjectContext 构造插件请求的上下文注入 payload。
//
// 注入未启用时返回空 context 而不是错误，方便插件在不同部署环境中使用同一协议流程。
func (h *Host) InjectContext(ctx context.Context, req protocol.InjectContextRequest) (resp protocol.InjectContextResponse, err error) {
	started := h.observationStart()
	defer func() {
		h.recordOperation(ctx, protocol.OperationInjectContext, req.RequestMeta, protocol.PluginMetadata{PluginID: req.PluginID, InstanceID: req.InstanceID}, strings.Join(req.Capabilities, ","), "", started, err)
	}()
	if h == nil || !h.Enabled() {
		return protocol.InjectContextResponse{}, ErrDisabled
	}
	if !h.injectionEnabled || h.injection == nil {
		return protocol.InjectContextResponse{
			SchemaVersion: h.defaultProtocolVersion,
			Context:       map[string]json.RawMessage{},
			Audit: protocol.AuditInfo{
				GeneratedAt: h.now(),
				Source:      "plugin-host",
			},
		}, nil
	}
	if err := h.authorizeInjection(ctx, protocol.OperationInjectContext, req.PluginID, req.InstanceID, req.Capabilities); err != nil {
		return protocol.InjectContextResponse{}, err
	}
	return h.injection.BuildContext(ctx, injection.Request{
		PluginID:      req.PluginID,
		Capabilities:  req.Capabilities,
		SchemaVersion: req.SchemaVersion,
	})
}

// GetInjectedSchema 返回当前 Host 暴露的上下文注入能力 schema。
//
// 注入未启用时返回空能力列表，插件可据此降级而不必把该情况视为协议失败。
func (h *Host) GetInjectedSchema(ctx context.Context, req protocol.GetInjectedSchemaRequest) (resp protocol.GetInjectedSchemaResponse, err error) {
	started := h.observationStart()
	defer func() {
		h.recordOperation(ctx, protocol.OperationGetInjectedSchema, req.RequestMeta, protocol.PluginMetadata{}, strings.Join(req.Capabilities, ","), "", started, err)
	}()
	if h == nil || !h.Enabled() {
		return protocol.GetInjectedSchemaResponse{}, ErrDisabled
	}
	if !h.injectionEnabled || h.injection == nil {
		return protocol.GetInjectedSchemaResponse{
			SchemaVersion: h.defaultProtocolVersion,
			Capabilities:  []protocol.InjectedCapability{},
			Audit: protocol.AuditInfo{
				GeneratedAt: h.now(),
				Source:      "plugin-host",
			},
		}, nil
	}
	if err := h.authorizeInjection(ctx, protocol.OperationGetInjectedSchema, "", "", req.Capabilities); err != nil {
		return protocol.GetInjectedSchemaResponse{}, err
	}
	return h.injection.Schema(req.Capabilities), nil
}

// NegotiateProtocol 在插件注册前协商协议版本和 transport。
//
// req.ProtocolVersions 与 req.Transports 是插件可支持列表；返回值选择 Host 支持的第一个匹配项。
func (h *Host) NegotiateProtocol(ctx context.Context, req protocol.NegotiateProtocolRequest) (resp protocol.NegotiateProtocolResponse, err error) {
	started := h.observationStart()
	defer func() {
		metadata := protocol.PluginMetadata{PluginID: req.PluginID, InstanceID: req.InstanceID, Protocol: protocol.PluginProtocolJSON, Transport: resp.Transport}
		h.recordOperation(ctx, protocol.OperationNegotiateProtocol, req.RequestMeta, metadata, "", "", started, err)
	}()
	if h == nil || !h.Enabled() {
		return protocol.NegotiateProtocolResponse{}, ErrDisabled
	}
	h.mu.RLock()
	defaultVersion := h.defaultProtocolVersion
	transports := append([]string(nil), h.allowedTransports...)
	h.mu.RUnlock()
	version := choose(req.ProtocolVersions, []string{defaultVersion})
	transport := choose(req.Transports, transports)
	if version == "" || transport == "" {
		return protocol.NegotiateProtocolResponse{
			Accepted:     false,
			Alternatives: transports,
		}, ErrUnsupportedProtocol
	}
	return protocol.NegotiateProtocolResponse{
		ProtocolVersion: version,
		Transport:       transport,
		Accepted:        true,
		Alternatives:    transports,
	}, nil
}

// invokeProvider 调用 Host 本地 provider。
//
// 第二个返回值表示是否命中了 provider；未命中不算错误，调用方可继续尝试远程 router。
func (h *Host) invokeProvider(ctx context.Context, req protocol.InvokeRequest) (protocol.InvokeResponse, bool, error) {
	h.mu.RLock()
	provider, ok := h.providers[strings.TrimSpace(req.Capability)]
	h.mu.RUnlock()
	if !ok {
		return protocol.InvokeResponse{}, false, nil
	}
	capability := provider.Capability()
	if h.authorizer != nil {
		if err := h.authorize(ctx, protocol.OperationInvoke, protocol.PluginSnapshot{PluginMetadata: protocol.PluginMetadata{
			PluginID:   req.PluginID,
			InstanceID: req.InstanceID,
		}}, capability.Permissions, capability.Name, ""); err != nil {
			return protocol.InvokeResponse{}, true, err
		}
	}
	result, err := provider.Invoke(ctx, req)
	if err != nil {
		return protocol.InvokeResponse{}, true, err
	}
	return protocol.InvokeResponse{Capability: req.Capability, Result: result}, true, nil
}

// expireLeases 尽力清理过期租约。
//
// 该辅助函数用于查询前状态收敛，失败会被忽略，避免只读管理接口因为清理任务失败而不可用。
func (h *Host) expireLeases(ctx context.Context) {
	if h == nil || h.registry == nil {
		return
	}
	_, _ = h.ExpireLeases(ctx)
}

// authorize 统一执行插件操作授权。
//
// permissions 是本次操作需要校验的 scope；capability 和 eventName 用于增强授权上下文和审计可读性。
func (h *Host) authorize(ctx context.Context, operation string, snapshot protocol.PluginSnapshot, permissions []string, capability string, eventName string) error {
	if h.authorizer == nil {
		return nil
	}
	decision, err := h.authorizer.Authorize(ctx, security.Principal{
		PluginID:   snapshot.PluginID,
		InstanceID: snapshot.InstanceID,
		Scopes:     snapshot.Permissions,
	}, security.PermissionRequest{
		Operation:   security.Operation{Name: operation, PluginID: snapshot.PluginID, InstanceID: snapshot.InstanceID},
		Permissions: permissions,
		Capability:  capability,
		Event:       eventName,
	})
	if err != nil {
		return ErrUnauthorized
	}
	if !decision.Allowed {
		return ErrUnauthorized
	}
	return nil
}

// authorizeInjection 校验上下文注入能力的权限。
//
// 当请求携带 plugin_id 和 instance_id 时会加载真实插件快照；匿名 schema 查询则只校验注入能力本身的权限声明。
func (h *Host) authorizeInjection(ctx context.Context, operation string, pluginID string, instanceID string, capabilityNames []string) error {
	if h == nil || h.authorizer == nil || h.injection == nil {
		return nil
	}
	var snapshot protocol.PluginSnapshot
	if strings.TrimSpace(pluginID) != "" && strings.TrimSpace(instanceID) != "" {
		item, err := h.registry.GetInstance(ctx, pluginID, instanceID)
		if err != nil {
			return mapRegistryError(err)
		}
		snapshot = item
	} else {
		snapshot.PluginID = strings.TrimSpace(pluginID)
		snapshot.InstanceID = strings.TrimSpace(instanceID)
	}
	return h.authorize(ctx, operation, snapshot, injectedCapabilityPermissions(h.injection.Capabilities(capabilityNames)), "", "")
}

// observationStart 返回观测事件使用的开始时间。
func (h *Host) observationStart() time.Time {
	if h == nil || h.now == nil {
		return time.Now().UTC()
	}
	return h.now()
}

// recordOperation 记录一次插件协议操作的观测事件。
//
// Recorder 是可选依赖；该方法不能影响主流程，因此仅组装事件并交给 recorder，不在这里处理存储或日志失败。
func (h *Host) recordOperation(ctx context.Context, operation string, meta protocol.RequestMeta, plugin protocol.PluginMetadata, capability string, eventName string, started time.Time, err error) {
	if h == nil || h.recorder == nil {
		return
	}
	ended := h.observationStart()
	if started.IsZero() {
		started = ended
	}
	status := observability.StatusOK
	errMessage := ""
	if err != nil {
		status = observability.StatusError
		errMessage = err.Error()
	}
	h.recorder.Record(ctx, observability.Event{
		Operation:      operation,
		PluginID:       strings.TrimSpace(plugin.PluginID),
		InstanceID:     strings.TrimSpace(plugin.InstanceID),
		Capability:     strings.TrimSpace(capability),
		Event:          strings.TrimSpace(eventName),
		Protocol:       strings.TrimSpace(plugin.Protocol),
		Transport:      protocol.EffectiveTransport(plugin),
		RequestID:      strings.TrimSpace(meta.RequestID),
		TraceID:        strings.TrimSpace(meta.TraceID),
		IdempotencyKey: strings.TrimSpace(meta.IdempotencyKey),
		Status:         status,
		Error:          errMessage,
		Source:         h.nodeID,
		StartedAt:      started,
		EndedAt:        ended,
		Duration:       ended.Sub(started),
		Metadata:       cloneStringMap(meta.Metadata),
	})
}

// normalizeMetadata 规范化插件注册或同步提交的元数据。
//
// 它兼容旧版把 transport 写在 protocol 字段中的格式，并复制可变切片与 map，避免外部请求对象被后续修改污染 registry。
func normalizeMetadata(metadata protocol.PluginMetadata) (protocol.PluginMetadata, error) {
	metadata.PluginID = strings.TrimSpace(metadata.PluginID)
	metadata.InstanceID = strings.TrimSpace(metadata.InstanceID)
	metadata.Name = strings.TrimSpace(metadata.Name)
	metadata.Version = strings.TrimSpace(metadata.Version)
	metadata.Protocol = strings.TrimSpace(metadata.Protocol)
	metadata.Transport = protocol.NormalizeTransport(metadata.Transport)
	metadata.Endpoint = strings.TrimSpace(metadata.Endpoint)
	metadata.SchemaVersion = strings.TrimSpace(metadata.SchemaVersion)
	if metadata.Transport == "" && protocol.IsTransport(metadata.Protocol) {
		metadata.Transport = protocol.NormalizeTransport(metadata.Protocol)
		metadata.Protocol = ""
	}
	if metadata.Protocol == "" {
		metadata.Protocol = protocol.PluginProtocolJSON
	}
	if metadata.PluginID == "" || metadata.InstanceID == "" || metadata.Name == "" || metadata.Version == "" || metadata.Transport == "" || metadata.Endpoint == "" {
		return protocol.PluginMetadata{}, ErrInvalidPlugin
	}
	metadata.Capabilities = cloneCapabilities(metadata.Capabilities)
	for i := range metadata.Capabilities {
		capability, err := normalizeCapability(metadata.Capabilities[i])
		if err != nil {
			return protocol.PluginMetadata{}, err
		}
		metadata.Capabilities[i] = capability
	}
	metadata.Permissions = normalizedStrings(metadata.Permissions)
	metadata.Hooks = normalizedStrings(metadata.Hooks)
	if len(metadata.Metadata) > 0 {
		cloned := make(map[string]string, len(metadata.Metadata))
		for key, value := range metadata.Metadata {
			key = strings.TrimSpace(key)
			if key != "" {
				cloned[key] = value
			}
		}
		metadata.Metadata = cloned
	}
	return metadata, nil
}

// normalizeCapability 校验并补齐插件能力声明的默认值。
func normalizeCapability(capability protocol.Capability) (protocol.Capability, error) {
	capability.Name = strings.TrimSpace(capability.Name)
	capability.Version = strings.TrimSpace(capability.Version)
	capability.Scope = strings.TrimSpace(capability.Scope)
	capability.SecretPolicy = strings.TrimSpace(capability.SecretPolicy)
	capability.Description = strings.TrimSpace(capability.Description)
	if capability.Name == "" {
		return protocol.Capability{}, ErrInvalidCapability
	}
	capability.Permissions = normalizedStrings(capability.Permissions)
	if capability.Scope == "" {
		capability.Scope = protocol.CapabilityScopePlugin
	}
	if capability.SecretPolicy == "" {
		capability.SecretPolicy = protocol.SecretPolicyNone
	}
	return capability, nil
}

// normalizedStrings 清理字符串列表中的空值和多余空白。
func normalizedStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

// cloneSnapshot 深拷贝插件快照中的可变字段。
func cloneSnapshot(src protocol.PluginSnapshot) protocol.PluginSnapshot {
	dst := src
	dst.Capabilities = cloneCapabilities(src.Capabilities)
	dst.Permissions = append([]string(nil), src.Permissions...)
	dst.Hooks = append([]string(nil), src.Hooks...)
	if len(src.Metadata) > 0 {
		dst.Metadata = make(map[string]string, len(src.Metadata))
		for key, value := range src.Metadata {
			dst.Metadata[key] = value
		}
	}
	return dst
}

// cloneCapabilities 深拷贝 capability 列表。
func cloneCapabilities(src []protocol.Capability) []protocol.Capability {
	out := make([]protocol.Capability, 0, len(src))
	for _, capability := range src {
		out = append(out, cloneCapability(capability))
	}
	return out
}

// cloneCapability 深拷贝单个 capability 的可变字段。
func cloneCapability(src protocol.Capability) protocol.Capability {
	dst := src
	dst.Permissions = append([]string(nil), src.Permissions...)
	dst.InputSchema = append(json.RawMessage(nil), src.InputSchema...)
	dst.OutputSchema = append(json.RawMessage(nil), src.OutputSchema...)
	return dst
}

// cloneStringMap 深拷贝字符串 map。
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

// contains 判断字符串列表中是否包含指定值。
func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

// choose 从插件提供列表和 Host 支持列表中选择第一个共同值。
//
// offered 为空时使用 supported 的第一个值，表示插件接受 Host 默认选择。
func choose(offered []string, supported []string) string {
	if len(offered) == 0 {
		if len(supported) == 0 {
			return ""
		}
		return supported[0]
	}
	for _, value := range offered {
		value = strings.TrimSpace(value)
		if contains(supported, value) {
			return value
		}
	}
	return ""
}

// firstNonEmpty 返回第一个非空字符串，用于默认值回退。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// healthStatus 根据快照和当前时间计算协议健康状态。
func healthStatus(plugin protocol.PluginSnapshot, now time.Time) protocol.HealthStatus {
	status := protocol.HealthStatus{
		PluginID:        plugin.PluginID,
		InstanceID:      plugin.InstanceID,
		Status:          protocol.HealthStatusOK,
		RuntimeStatus:   plugin.RuntimeStatus,
		LastHeartbeatAt: plugin.LastHeartbeatAt,
		LeaseExpiresAt:  plugin.LeaseExpiresAt,
	}
	if plugin.Status != protocol.StatusOnline || (!plugin.LeaseExpiresAt.IsZero() && !plugin.LeaseExpiresAt.After(now)) {
		status.Status = protocol.HealthStatusUnknown
		status.Error = "plugin lease expired or offline"
	}
	return status
}

// aggregateCapabilities 汇总多实例插件暴露的能力，并按能力名称去重。
func aggregateCapabilities(items []protocol.PluginSnapshot) []protocol.Capability {
	seen := map[string]protocol.Capability{}
	for _, item := range items {
		for _, capability := range item.Capabilities {
			if _, ok := seen[capability.Name]; !ok {
				seen[capability.Name] = cloneCapability(capability)
			}
		}
	}
	out := make([]protocol.Capability, 0, len(seen))
	for _, capability := range seen {
		out = append(out, capability)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

// registrationPermissions 汇总插件注册时需要授权的全部权限声明。
func registrationPermissions(snapshot protocol.PluginSnapshot) []string {
	values := append([]string(nil), snapshot.Permissions...)
	for _, capability := range snapshot.Capabilities {
		values = append(values, capability.Permissions...)
	}
	return normalizedStrings(values)
}

// injectedCapabilityPermissions 汇总注入能力声明的权限。
func injectedCapabilityPermissions(capabilities []protocol.InjectedCapability) []string {
	var values []string
	for _, capability := range capabilities {
		values = append(values, capability.Permissions...)
	}
	return normalizedStrings(values)
}

// sortSnapshots 按 plugin_id 和 instance_id 排序，保证管理接口输出稳定。
func sortSnapshots(items []protocol.PluginSnapshot) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].PluginID != items[j].PluginID {
			return items[i].PluginID < items[j].PluginID
		}
		return items[i].InstanceID < items[j].InstanceID
	})
}

// mapRegistryError 将 registry 包错误映射为 Host 对外错误。
func mapRegistryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, registry.ErrNotFound) {
		return ErrPluginNotFound
	}
	if errors.Is(err, registry.ErrInvalid) {
		return ErrInvalidPlugin
	}
	return err
}

// mapRouterError 将 router 包错误映射为 Host 对外错误。
func mapRouterError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, pluginrouter.ErrNoRoute):
		return ErrCapabilityNotFound
	case errors.Is(err, pluginrouter.ErrPluginOffline):
		return ErrPluginOffline
	case errors.Is(err, pluginrouter.ErrTransportUnavailable):
		return ErrTransportUnavailable
	case errors.Is(err, pluginrouter.ErrInvalidRequest):
		return ErrInvalidCapability
	default:
		return err
	}
}

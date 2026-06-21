// Package router 负责从插件 registry 中选择可用远程实例，并通过对应 transport invoker 发起调用。
package router

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
	"github.com/rei0721/go-scaffold/pkg/plugin/registry"
)

var (
	// Err* 是 router 对上层 Host 暴露的稳定错误哨兵，上层会再映射为插件协议错误码。
	ErrNoRoute              = errors.New("plugin route not found")
	ErrPluginOffline        = errors.New("plugin route offline")
	ErrTransportUnavailable = errors.New("plugin route transport unavailable")
	ErrInvalidRequest       = errors.New("invalid plugin route request")
)

// RemoteInvoker 是具体 transport adapter 需要实现的最小调用接口。
//
// Router 只依赖这个接口，从而保持 HTTP、WebSocket、RPC 等 transport 互不耦合。
type RemoteInvoker interface {
	Invoke(context.Context, protocol.PluginSnapshot, protocol.InvokeRequest) (json.RawMessage, error)
	PushEvent(context.Context, protocol.PluginSnapshot, protocol.PushEventRequest) error
}

// Config 描述 Router 的依赖和调用策略。
//
// Registry 提供实例与订阅查询；RemoteInvokers 按 transport 名称索引；RequestTimeout 和 RetryCount
// 控制远程调用边界；Now 用于测试固定租约过期时间。
type Config struct {
	Registry       registry.Registry
	RemoteInvokers map[string]RemoteInvoker
	RequestTimeout time.Duration
	RetryCount     int
	Strategy       string
	Now            func() time.Time
}

// Router 将能力调用和事件推送路由到远程插件实例。
//
// counters 保存按 route key 的轮询位置，保证同一能力的多个健康实例可以简单分摊请求。
type Router struct {
	registry       registry.Registry
	remoteInvokers map[string]RemoteInvoker
	requestTimeout time.Duration
	retryCount     int
	now            func() time.Time

	mu       sync.Mutex
	counters map[string]int
}

// New 创建远程插件路由器，并补齐默认超时和时钟。
func New(cfg Config) *Router {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 10 * time.Second
	}
	return &Router{
		registry:       cfg.Registry,
		remoteInvokers: cloneInvokers(cfg.RemoteInvokers),
		requestTimeout: cfg.RequestTimeout,
		retryCount:     cfg.RetryCount,
		now:            func() time.Time { return cfg.Now().UTC() },
		counters:       map[string]int{},
	}
}

// RegisterRemoteInvoker 为 transport 名称注册调用适配器。
//
// 该方法可在 Host 装配阶段或 transport 初始化完成后调用；同名 transport 会被后注册的 invoker 覆盖。
func (r *Router) RegisterRemoteInvoker(transport string, invoker RemoteInvoker) error {
	transport = strings.TrimSpace(transport)
	if r == nil || transport == "" || invoker == nil {
		return ErrTransportUnavailable
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.remoteInvokers == nil {
		r.remoteInvokers = map[string]RemoteInvoker{}
	}
	r.remoteInvokers[transport] = invoker
	return nil
}

// Invoke 根据 capability 和可选插件实例标识选择远程实例并执行能力调用。
//
// req.Capability 是必需参数；PluginID/InstanceID 为空时按能力选择健康实例。返回值中的 Result
// 来自远端 InvokeResponse，调用失败会按 RetryCount 尝试重新选择实例。
func (r *Router) Invoke(ctx context.Context, req protocol.InvokeRequest) (protocol.InvokeResponse, error) {
	if r == nil || r.registry == nil {
		return protocol.InvokeResponse{}, ErrNoRoute
	}
	if strings.TrimSpace(req.Capability) == "" {
		return protocol.InvokeResponse{}, ErrInvalidRequest
	}
	instances, err := r.candidates(ctx, req.PluginID, req.InstanceID, req.Capability)
	if err != nil {
		return protocol.InvokeResponse{}, err
	}
	attempts := r.retryCount + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		plugin, err := r.pick(req.Capability, instances)
		if err != nil {
			return protocol.InvokeResponse{}, err
		}
		invoker := r.invoker(protocol.EffectiveTransport(plugin.PluginMetadata))
		if invoker == nil {
			return protocol.InvokeResponse{}, ErrTransportUnavailable
		}
		callReq := req
		// 选定实例后把路由结果写回请求，确保远端能看到最终被调用的 plugin_id 和 instance_id。
		callReq.PluginID = plugin.PluginID
		callReq.InstanceID = plugin.InstanceID
		callCtx, cancel := r.callContext(ctx, req.TimeoutMillis)
		result, err := invoker.Invoke(callCtx, plugin, callReq)
		cancel()
		if err == nil {
			return protocol.InvokeResponse{Capability: req.Capability, Result: result}, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return protocol.InvokeResponse{}, lastErr
	}
	return protocol.InvokeResponse{}, ErrNoRoute
}

// PushEvent 将事件推送给显式目标或订阅了该事件的健康插件实例。
//
// 单个目标失败不会阻断其他目标；只有一个实例都未成功投递且存在错误时才返回失败。
func (r *Router) PushEvent(ctx context.Context, req protocol.PushEventRequest) (protocol.PushEventResponse, error) {
	if r == nil || r.registry == nil {
		return protocol.PushEventResponse{}, ErrNoRoute
	}
	if strings.TrimSpace(req.Event) == "" {
		return protocol.PushEventResponse{}, ErrInvalidRequest
	}
	instances, err := r.eventTargets(ctx, req)
	if err != nil {
		return protocol.PushEventResponse{}, err
	}
	delivered := 0
	var lastErr error
	for _, plugin := range instances {
		invoker := r.invoker(protocol.EffectiveTransport(plugin.PluginMetadata))
		if invoker == nil {
			lastErr = ErrTransportUnavailable
			continue
		}
		callReq := req
		callReq.PluginID = plugin.PluginID
		callReq.InstanceID = plugin.InstanceID
		callCtx, cancel := r.callContext(ctx, 0)
		err := invoker.PushEvent(callCtx, plugin, callReq)
		cancel()
		if err != nil {
			lastErr = err
			continue
		}
		delivered++
	}
	if delivered == 0 && lastErr != nil {
		return protocol.PushEventResponse{}, lastErr
	}
	return protocol.PushEventResponse{Accepted: true, Event: req.Event, Delivered: delivered}, nil
}

// candidates 返回满足目标条件、在线租约有效且具备 transport invoker 的实例。
//
// 查询前会先触发一次过期租约清理，让路由选择尽量基于最新可用状态。
func (r *Router) candidates(ctx context.Context, pluginID string, instanceID string, capability string) ([]protocol.PluginSnapshot, error) {
	now := r.now()
	_, _ = r.registry.ExpireLeases(ctx, now)
	filter := registry.InstanceFilter{PluginID: strings.TrimSpace(pluginID), InstanceID: strings.TrimSpace(instanceID), Status: protocol.StatusOnline, Now: now}
	var items []protocol.PluginSnapshot
	var err error
	if filter.InstanceID != "" {
		var item protocol.PluginSnapshot
		item, err = r.registry.GetInstance(ctx, filter.PluginID, filter.InstanceID)
		if err == nil {
			items = []protocol.PluginSnapshot{item}
		}
	} else if strings.TrimSpace(capability) != "" {
		items, err = r.registry.ListByCapability(ctx, capability, filter)
	} else {
		items, err = r.registry.ListInstances(ctx, filter)
	}
	if err != nil {
		return nil, err
	}
	items = healthy(items, now, r.remoteInvokers)
	if len(items) == 0 {
		return nil, ErrPluginOffline
	}
	return items, nil
}

// eventTargets 根据 PushEvent 请求解析投递目标。
//
// 显式 plugin_id/instance_id 优先；否则按订阅表查找事件订阅者，并去重同一插件实例。
func (r *Router) eventTargets(ctx context.Context, req protocol.PushEventRequest) ([]protocol.PluginSnapshot, error) {
	if strings.TrimSpace(req.PluginID) != "" || strings.TrimSpace(req.InstanceID) != "" {
		return r.candidates(ctx, req.PluginID, req.InstanceID, "")
	}
	subs, err := r.registry.ListSubscriptions(ctx, registry.SubscriptionFilter{Event: req.Event})
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []protocol.PluginSnapshot
	now := r.now()
	_, _ = r.registry.ExpireLeases(ctx, now)
	for _, sub := range subs {
		k := sub.PluginID + "\x00" + sub.InstanceID
		if seen[k] {
			continue
		}
		seen[k] = true
		item, err := r.registry.GetInstance(ctx, sub.PluginID, sub.InstanceID)
		if err != nil {
			continue
		}
		out = append(out, item)
	}
	out = healthy(out, now, r.remoteInvokers)
	return out, nil
}

// pick 使用简单轮询从候选实例中选择一个目标。
//
// routeKey 通常是 capability 名称，用于让不同能力维护独立轮询计数。
func (r *Router) pick(routeKey string, items []protocol.PluginSnapshot) (protocol.PluginSnapshot, error) {
	if len(items) == 0 {
		return protocol.PluginSnapshot{}, ErrNoRoute
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	index := r.counters[routeKey] % len(items)
	r.counters[routeKey] = r.counters[routeKey] + 1
	return items[index], nil
}

// invoker 按 transport 名称查找已注册的远程调用适配器。
func (r *Router) invoker(transport string) RemoteInvoker {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.remoteInvokers[transport]
}

// callContext 为单次远程调用创建带超时的子 context。
//
// req.TimeoutMillis 优先于 Router 默认超时；非正超时会保留取消能力但不设置 deadline。
func (r *Router) callContext(ctx context.Context, timeoutMillis int) (context.Context, context.CancelFunc) {
	timeout := r.requestTimeout
	if timeoutMillis > 0 {
		timeout = time.Duration(timeoutMillis) * time.Millisecond
	}
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

// healthy 过滤掉离线、租约过期或缺少 transport invoker 的实例。
func healthy(items []protocol.PluginSnapshot, now time.Time, invokers map[string]RemoteInvoker) []protocol.PluginSnapshot {
	out := make([]protocol.PluginSnapshot, 0, len(items))
	for _, item := range items {
		if item.Status != protocol.StatusOnline {
			continue
		}
		if !item.LeaseExpiresAt.IsZero() && !item.LeaseExpiresAt.After(now) {
			continue
		}
		if invokers[protocol.EffectiveTransport(item.PluginMetadata)] == nil {
			continue
		}
		out = append(out, item)
	}
	return out
}

// cloneInvokers 复制并规范化 transport invoker 表，避免外部 map 后续修改影响 Router。
func cloneInvokers(src map[string]RemoteInvoker) map[string]RemoteInvoker {
	dst := map[string]RemoteInvoker{}
	for key, value := range src {
		if strings.TrimSpace(key) != "" && value != nil {
			dst[strings.TrimSpace(key)] = value
		}
	}
	return dst
}

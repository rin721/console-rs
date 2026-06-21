package registry

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

// PollWatcher 通过定时读取 Registry 来模拟状态变化订阅。
//
// 它用于 SQLStore 等没有原生推送能力的后端，代价是变化最多延迟一个 PollInterval。
type PollWatcher struct {
	Registry Registry
	Now      func() time.Time
}

// NewPollWatcher 创建基于轮询的 watcher。
func NewPollWatcher(registry Registry) *PollWatcher {
	return &PollWatcher{Registry: registry}
}

// Watch 启动轮询并返回只读变化通道。
//
// options 会补齐默认缓冲和轮询间隔；当过滤条件没有指定状态或时间时，会包含已过期实例，
// 这样状态同步消费者可以观察到离线或过期边界，而不是被默认在线过滤误伤。
func (w *PollWatcher) Watch(ctx context.Context, options WatchOptions) (<-chan Change, error) {
	if w == nil || w.Registry == nil {
		return nil, ErrInvalid
	}
	if options.Buffer <= 0 {
		options.Buffer = 32
	}
	if options.PollInterval <= 0 {
		options.PollInterval = 5 * time.Second
	}
	if options.Filter.Now.IsZero() && options.Filter.Status == "" {
		options.Filter.IncludeExpired = true
	}
	out := make(chan Change, options.Buffer)
	go w.run(ctx, out, options)
	return out, nil
}

// run 执行 watcher 主循环，并在 context 结束时关闭输出通道。
func (w *PollWatcher) run(ctx context.Context, out chan<- Change, options WatchOptions) {
	defer close(out)
	previous := map[InstanceKey]protocol.PluginSnapshot{}
	_ = w.poll(ctx, out, options, previous, options.InitialSnapshot)

	ticker := time.NewTicker(options.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = w.poll(ctx, out, options, previous, true)
		}
	}
}

// poll 对比当前 registry 快照和上一次快照，输出新增、变化和消失事件。
//
// previous 由调用方复用并在函数末尾原地刷新，避免每轮循环额外返回大对象。
func (w *PollWatcher) poll(ctx context.Context, out chan<- Change, options WatchOptions, previous map[InstanceKey]protocol.PluginSnapshot, emit bool) error {
	filter := options.Filter
	if filter.Now.IsZero() {
		filter.Now = w.now()
	}
	items, err := w.Registry.ListInstances(ctx, filter)
	if err != nil {
		return err
	}
	current := make(map[InstanceKey]protocol.PluginSnapshot, len(items))
	for _, item := range items {
		key := InstanceKey{PluginID: item.PluginID, InstanceID: item.InstanceID}
		current[key] = item
		old, existed := previous[key]
		// fingerprint 允许 Registry 追加字段后 watcher 仍能检测结构化快照的任意变化。
		if emit && (!existed || snapshotFingerprint(old) != snapshotFingerprint(item)) {
			change := Change{
				Type:       ChangeSnapshotObserved,
				PluginID:   item.PluginID,
				InstanceID: item.InstanceID,
				Plugin:     item,
				ObservedAt: w.now(),
			}
			if existed {
				old := old
				change.Previous = &old
			}
			sendChange(ctx, out, change)
		}
	}
	if emit {
		for key, old := range previous {
			if _, ok := current[key]; ok {
				continue
			}
			// 当前轮查询不到上一轮存在的实例时，显式发送 disappeared 事件供缓存清理。
			old := old
			sendChange(ctx, out, Change{
				Type:       ChangeSnapshotDisappeared,
				PluginID:   key.PluginID,
				InstanceID: key.InstanceID,
				Previous:   &old,
				ObservedAt: w.now(),
			})
		}
	}
	for key := range previous {
		delete(previous, key)
	}
	for key, item := range current {
		previous[key] = item
	}
	return nil
}

// now 返回当前 UTC 时间，并允许测试注入稳定时钟。
func (w *PollWatcher) now() time.Time {
	if w != nil && w.Now != nil {
		return w.Now().UTC()
	}
	return time.Now().UTC()
}

// StatusHandler 消费 watcher 输出的状态变化。
type StatusHandler func(context.Context, Change) error

// StatusSynchronizer 将 Watcher 的变化通道桥接到调用方提供的处理函数。
//
// 它不吞掉 Handler 错误，确保同步下游失败时调用方可以重启或报警。
type StatusSynchronizer struct {
	Watcher Watcher
	Options WatchOptions
	Handler StatusHandler
}

// Run 阻塞消费 watcher 变化，直到 context 结束、通道关闭或 Handler 返回错误。
func (s StatusSynchronizer) Run(ctx context.Context) error {
	if s.Watcher == nil || s.Handler == nil {
		return ErrInvalid
	}
	changes, err := s.Watcher.Watch(ctx, s.Options)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case change, ok := <-changes:
			if !ok {
				return nil
			}
			if err := s.Handler(ctx, change); err != nil {
				return err
			}
		}
	}
}

// sendChange 尝试发送变化事件。
//
// 输出通道已满时会丢弃事件，避免轮询 goroutine 被慢消费者永久阻塞；下一轮快照仍能重新收敛状态。
func sendChange(ctx context.Context, out chan<- Change, change Change) {
	select {
	case out <- cloneChange(change):
	case <-ctx.Done():
	default:
	}
}

// snapshotFingerprint 为快照比较生成稳定字符串。
//
// PluginSnapshot 当前由 JSON 友好的字段组成，使用 JSON 可以覆盖嵌套 capability、metadata 和时间字段。
func snapshotFingerprint(snapshot protocol.PluginSnapshot) string {
	raw, _ := json.Marshal(snapshot)
	return string(raw)
}

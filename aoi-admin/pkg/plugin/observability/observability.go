package observability

import (
	"context"
	"time"
)

const (
	StatusOK    = "ok"
	StatusError = "error"
)

// Event 描述一次远程插件系统操作的审计与观测事件。
type Event struct {
	Operation      string
	PluginID       string
	InstanceID     string
	Capability     string
	Event          string
	Protocol       string
	Transport      string
	RequestID      string
	TraceID        string
	IdempotencyKey string
	Status         string
	Error          string
	Source         string
	StartedAt      time.Time
	EndedAt        time.Time
	Duration       time.Duration
	Metadata       map[string]string
}

// Recorder 接收插件系统产生的审计与观测事件。
type Recorder interface {
	Record(context.Context, Event)
}

// RecorderFunc 让普通函数可以作为 Recorder 使用。
type RecorderFunc func(context.Context, Event)

func (f RecorderFunc) Record(ctx context.Context, event Event) {
	if f != nil {
		f(ctx, event)
	}
}

// NopRecorder 是默认空实现，避免观测能力影响插件主流程。
type NopRecorder struct{}

func (NopRecorder) Record(context.Context, Event) {}

package adapters

import (
	"context"
	"sync"
	"time"

	"github.com/rei0721/go-scaffold/internal/ports"
)

const DefaultTrafficProbeScheduleInterval = 5 * time.Second

type TrafficProbeSchedulerService interface {
	RunDueTrafficProbes(context.Context) (int, error)
}

type TrafficProbeScheduler struct {
	service  TrafficProbeSchedulerService
	logger   ports.Logger
	interval time.Duration

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	started bool
}

func NewTrafficProbeScheduler(service TrafficProbeSchedulerService, logger ports.Logger, interval time.Duration) *TrafficProbeScheduler {
	if interval <= 0 {
		interval = DefaultTrafficProbeScheduleInterval
	}
	return &TrafficProbeScheduler{service: service, logger: logger, interval: interval}
}

func (s *TrafficProbeScheduler) Start(ctx context.Context) error {
	if s == nil || s.service == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	s.started = true
	go s.run(runCtx, done)
	return nil
}

func (s *TrafficProbeScheduler) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	s.cancel = nil
	s.done = nil
	s.started = false
	s.mu.Unlock()
	if cancel == nil || done == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *TrafficProbeScheduler) run(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	s.tick(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *TrafficProbeScheduler) tick(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, s.interval)
	defer cancel()
	count, err := s.service.RunDueTrafficProbes(runCtx)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("traffic probe scheduler failed", "error", err)
		}
		return
	}
	if count > 0 && s.logger != nil {
		s.logger.Debug("traffic probe scheduler completed", "count", count)
	}
}

package adapters

import (
	"context"
	"math"
	"runtime"
	"sync"
	"time"

	systemservice "github.com/rei0721/go-scaffold/internal/modules/system/service"
)

const (
	DefaultServerMetricsInterval   = 5 * time.Second
	DefaultServerMetricsMaxSamples = 60
)

// ServerMetricsSampler 维护服务器运行指标的短窗口内存历史。
type ServerMetricsSampler struct {
	collector  systemservice.HostMetricsCollector
	interval   time.Duration
	maxSamples int
	now        func() time.Time

	mu              sync.RWMutex
	samples         []systemservice.MetricsSample
	previousAt      time.Time
	previousDiskIO  map[string]systemservice.DiskIOInfo
	previousNetwork systemservice.NetworkInfo
	cancel          context.CancelFunc
	done            chan struct{}
}

func NewServerMetricsSampler(
	collector systemservice.HostMetricsCollector,
	interval time.Duration,
	maxSamples int,
) *ServerMetricsSampler {
	if interval <= 0 {
		interval = DefaultServerMetricsInterval
	}
	if maxSamples <= 0 {
		maxSamples = DefaultServerMetricsMaxSamples
	}
	return &ServerMetricsSampler{
		collector:  collector,
		interval:   interval,
		maxSamples: maxSamples,
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (s *ServerMetricsSampler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	s.mu.Unlock()

	s.sample(runCtx)
	go func() {
		defer close(done)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				s.sample(runCtx)
			}
		}
	}()
	return nil
}

func (s *ServerMetricsSampler) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	s.cancel = nil
	s.done = nil
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

func (s *ServerMetricsSampler) History(context.Context) systemservice.MetricsHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	samples := make([]systemservice.MetricsSample, len(s.samples))
	for i, sample := range s.samples {
		samples[i] = sample
		samples[i].DiskIO = append([]systemservice.DiskIOSample(nil), sample.DiskIO...)
	}
	return systemservice.MetricsHistory{
		IntervalSeconds: int(s.interval.Seconds()),
		Samples:         samples,
		WindowSeconds:   int(s.interval.Seconds()) * s.maxSamples,
	}
}

func (s *ServerMetricsSampler) sample(ctx context.Context) {
	now := s.now().UTC()
	host := systemservice.HostMetrics{}
	if s.collector != nil {
		host = s.collector.Collect(ctx)
	}

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	sample := systemservice.MetricsSample{
		SampledAt:          now,
		CPUUsedPercent:     average(host.CPU.Percent),
		RAMUsedPercent:     sanitizePercent(host.RAM.UsedPercent),
		DiskMaxUsedPercent: maxDiskUsedPercent(host.Disk),
		HeapAllocMB:        bytesToMB(stats.HeapAlloc),
		Goroutines:         runtime.NumGoroutine(),
	}

	s.mu.Lock()
	if !s.previousAt.IsZero() {
		elapsed := now.Sub(s.previousAt).Seconds()
		diskIO, diskSummary := diskIOSamples(host.DiskIO, s.previousDiskIO, elapsed)
		sample.DiskReadMBPerSecond = diskSummary.ReadMBPerSecond
		sample.DiskWriteMBPerSecond = diskSummary.WriteMBPerSecond
		sample.DiskReadOpsPerSecond = diskSummary.ReadOpsPerSecond
		sample.DiskWriteOpsPerSecond = diskSummary.WriteOpsPerSecond
		sample.DiskIOLatencyMs = diskSummary.IOLatencyMs
		sample.DiskIO = diskIO
		sample.NetworkReceiveKBPerSecond = bytesPerSecondToKBPerSecond(
			host.Network.ReceiveBytes,
			s.previousNetwork.ReceiveBytes,
			elapsed,
		)
		sample.NetworkTransmitKBPerSecond = bytesPerSecondToKBPerSecond(
			host.Network.TransmitBytes,
			s.previousNetwork.TransmitBytes,
			elapsed,
		)
	}
	s.previousAt = now
	s.previousDiskIO = diskIOCounterMap(host.DiskIO)
	s.previousNetwork = host.Network
	s.samples = append(s.samples, sample)
	if len(s.samples) > s.maxSamples {
		s.samples = append([]systemservice.MetricsSample(nil), s.samples[len(s.samples)-s.maxSamples:]...)
	}
	s.mu.Unlock()
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += sanitizePercent(value)
	}
	return roundMetric(sum / float64(len(values)))
}

func maxDiskUsedPercent(disks []systemservice.DiskInfo) float64 {
	out := 0.0
	for _, disk := range disks {
		out = math.Max(out, sanitizePercent(disk.UsedPercent))
	}
	return roundMetric(out)
}

func diskIOSamples(current []systemservice.DiskIOInfo, previous map[string]systemservice.DiskIOInfo, elapsedSeconds float64) ([]systemservice.DiskIOSample, systemservice.DiskIOSample) {
	if elapsedSeconds <= 0 || len(current) == 0 {
		return []systemservice.DiskIOSample{}, systemservice.DiskIOSample{}
	}
	out := make([]systemservice.DiskIOSample, 0, len(current))
	var totalReadMBPerSecond float64
	var totalWriteMBPerSecond float64
	var totalReadOpsPerSecond float64
	var totalWriteOpsPerSecond float64
	var totalTimeDelta uint64
	var totalOpsDelta uint64
	for _, counter := range current {
		prev, ok := previous[counter.Name]
		sample := systemservice.DiskIOSample{Name: counter.Name}
		if ok {
			readBytesDelta := counterDelta(counter.ReadBytes, prev.ReadBytes)
			writeBytesDelta := counterDelta(counter.WriteBytes, prev.WriteBytes)
			readCountDelta := counterDelta(counter.ReadCount, prev.ReadCount)
			writeCountDelta := counterDelta(counter.WriteCount, prev.WriteCount)
			readTimeDelta := counterDelta(counter.ReadTimeMs, prev.ReadTimeMs)
			writeTimeDelta := counterDelta(counter.WriteTimeMs, prev.WriteTimeMs)
			opsDelta := readCountDelta + writeCountDelta
			timeDelta := readTimeDelta + writeTimeDelta

			sample.ReadMBPerSecond = bytesPerSecondToMBPerSecond(readBytesDelta, elapsedSeconds)
			sample.WriteMBPerSecond = bytesPerSecondToMBPerSecond(writeBytesDelta, elapsedSeconds)
			sample.ReadOpsPerSecond = countPerSecond(readCountDelta, elapsedSeconds)
			sample.WriteOpsPerSecond = countPerSecond(writeCountDelta, elapsedSeconds)
			sample.IOLatencyMs = latencyMs(timeDelta, opsDelta)

			totalReadMBPerSecond += sample.ReadMBPerSecond
			totalWriteMBPerSecond += sample.WriteMBPerSecond
			totalReadOpsPerSecond += sample.ReadOpsPerSecond
			totalWriteOpsPerSecond += sample.WriteOpsPerSecond
			totalTimeDelta += timeDelta
			totalOpsDelta += opsDelta
		}
		out = append(out, sample)
	}
	return out, systemservice.DiskIOSample{
		Name:              "all",
		ReadMBPerSecond:   roundMetric(totalReadMBPerSecond),
		WriteMBPerSecond:  roundMetric(totalWriteMBPerSecond),
		ReadOpsPerSecond:  roundMetric(totalReadOpsPerSecond),
		WriteOpsPerSecond: roundMetric(totalWriteOpsPerSecond),
		IOLatencyMs:       latencyMs(totalTimeDelta, totalOpsDelta),
	}
}

func diskIOCounterMap(counters []systemservice.DiskIOInfo) map[string]systemservice.DiskIOInfo {
	out := make(map[string]systemservice.DiskIOInfo, len(counters))
	for _, counter := range counters {
		if counter.Name == "" {
			continue
		}
		out[counter.Name] = counter
	}
	return out
}

func bytesPerSecondToKBPerSecond(current, previous uint64, elapsedSeconds float64) float64 {
	if elapsedSeconds <= 0 || current < previous {
		return 0
	}
	return roundMetric((float64(current-previous) / 1024) / elapsedSeconds)
}

func bytesPerSecondToMBPerSecond(delta uint64, elapsedSeconds float64) float64 {
	if elapsedSeconds <= 0 {
		return 0
	}
	return roundMetric((float64(delta) / (1024 * 1024)) / elapsedSeconds)
}

func countPerSecond(delta uint64, elapsedSeconds float64) float64 {
	if elapsedSeconds <= 0 {
		return 0
	}
	return roundMetric(float64(delta) / elapsedSeconds)
}

func latencyMs(timeDelta uint64, opDelta uint64) float64 {
	if opDelta == 0 {
		return 0
	}
	return roundMetric(float64(timeDelta) / float64(opDelta))
}

func counterDelta(current, previous uint64) uint64 {
	if current < previous {
		return 0
	}
	return current - previous
}

func sanitizePercent(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func roundMetric(value float64) float64 {
	return math.Round(value*10) / 10
}

func bytesToMB(value uint64) uint64 {
	const bytesPerMB = 1024 * 1024
	return value / bytesPerMB
}

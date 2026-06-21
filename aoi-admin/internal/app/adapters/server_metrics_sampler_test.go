package adapters

import (
	"context"
	"testing"
	"time"

	systemservice "github.com/rei0721/go-scaffold/internal/modules/system/service"
)

func TestServerMetricsSamplerCalculatesDiskIODeltas(t *testing.T) {
	start := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	collector := &sequenceHostMetricsCollector{
		items: []systemservice.HostMetrics{
			{
				DiskIO: []systemservice.DiskIOInfo{
					{
						Name:        "disk0",
						ReadBytes:   1 * 1024 * 1024,
						WriteBytes:  2 * 1024 * 1024,
						ReadCount:   10,
						WriteCount:  4,
						ReadTimeMs:  20,
						WriteTimeMs: 8,
					},
					{
						Name:        "disk1",
						ReadBytes:   4 * 1024 * 1024,
						WriteBytes:  1 * 1024 * 1024,
						ReadCount:   8,
						WriteCount:  2,
						ReadTimeMs:  16,
						WriteTimeMs: 4,
					},
				},
			},
			{
				DiskIO: []systemservice.DiskIOInfo{
					{
						Name:        "disk0",
						ReadBytes:   6 * 1024 * 1024,
						WriteBytes:  4 * 1024 * 1024,
						ReadCount:   20,
						WriteCount:  8,
						ReadTimeMs:  70,
						WriteTimeMs: 28,
					},
					{
						Name:        "disk1",
						ReadBytes:   5 * 1024 * 1024,
						WriteBytes:  6 * 1024 * 1024,
						ReadCount:   13,
						WriteCount:  12,
						ReadTimeMs:  31,
						WriteTimeMs: 44,
					},
				},
			},
		},
	}
	sampler := NewServerMetricsSampler(collector, 5*time.Second, 10)
	tick := 0
	sampler.now = func() time.Time {
		value := start.Add(time.Duration(tick) * 5 * time.Second)
		tick++
		return value
	}

	sampler.sample(context.Background())
	sampler.sample(context.Background())

	history := sampler.History(context.Background())
	if len(history.Samples) != 2 {
		t.Fatalf("sample count = %d, want 2", len(history.Samples))
	}
	first := history.Samples[0]
	if first.DiskReadMBPerSecond != 0 || first.DiskWriteMBPerSecond != 0 || len(first.DiskIO) != 0 {
		t.Fatalf("first sample should not invent disk IO rates: %#v", first)
	}
	second := history.Samples[1]
	if second.DiskReadMBPerSecond != 1.2 ||
		second.DiskWriteMBPerSecond != 1.4 ||
		second.DiskReadOpsPerSecond != 3 ||
		second.DiskWriteOpsPerSecond != 2.8 ||
		second.DiskIOLatencyMs != 4.3 {
		t.Fatalf("unexpected aggregate disk IO sample: %#v", second)
	}
	if len(second.DiskIO) != 2 {
		t.Fatalf("disk IO sample count = %d, want 2", len(second.DiskIO))
	}
	if second.DiskIO[0].Name != "disk0" ||
		second.DiskIO[0].ReadMBPerSecond != 1 ||
		second.DiskIO[0].WriteMBPerSecond != 0.4 ||
		second.DiskIO[0].ReadOpsPerSecond != 2 ||
		second.DiskIO[0].WriteOpsPerSecond != 0.8 ||
		second.DiskIO[0].IOLatencyMs != 5 {
		t.Fatalf("unexpected disk0 sample: %#v", second.DiskIO[0])
	}
	if second.DiskIO[1].Name != "disk1" ||
		second.DiskIO[1].ReadMBPerSecond != 0.2 ||
		second.DiskIO[1].WriteMBPerSecond != 1 ||
		second.DiskIO[1].ReadOpsPerSecond != 1 ||
		second.DiskIO[1].WriteOpsPerSecond != 2 ||
		second.DiskIO[1].IOLatencyMs != 3.7 {
		t.Fatalf("unexpected disk1 sample: %#v", second.DiskIO[1])
	}
}

func TestServerMetricsSamplerTreatsDiskCounterRollbackAsZeroDelta(t *testing.T) {
	start := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	collector := &sequenceHostMetricsCollector{
		items: []systemservice.HostMetrics{
			{DiskIO: []systemservice.DiskIOInfo{{
				Name:        "disk0",
				ReadBytes:   10 * 1024 * 1024,
				WriteBytes:  8 * 1024 * 1024,
				ReadCount:   20,
				WriteCount:  10,
				ReadTimeMs:  80,
				WriteTimeMs: 40,
			}}},
			{DiskIO: []systemservice.DiskIOInfo{{
				Name:        "disk0",
				ReadBytes:   1 * 1024 * 1024,
				WriteBytes:  1 * 1024 * 1024,
				ReadCount:   2,
				WriteCount:  1,
				ReadTimeMs:  8,
				WriteTimeMs: 4,
			}}},
		},
	}
	sampler := NewServerMetricsSampler(collector, 5*time.Second, 10)
	tick := 0
	sampler.now = func() time.Time {
		value := start.Add(time.Duration(tick) * 5 * time.Second)
		tick++
		return value
	}

	sampler.sample(context.Background())
	sampler.sample(context.Background())

	second := sampler.History(context.Background()).Samples[1]
	if second.DiskReadMBPerSecond != 0 ||
		second.DiskWriteMBPerSecond != 0 ||
		second.DiskReadOpsPerSecond != 0 ||
		second.DiskWriteOpsPerSecond != 0 ||
		second.DiskIOLatencyMs != 0 ||
		second.DiskIO[0].IOLatencyMs != 0 {
		t.Fatalf("rollback should produce zero deltas: %#v", second)
	}
}

type sequenceHostMetricsCollector struct {
	index int
	items []systemservice.HostMetrics
}

func (c *sequenceHostMetricsCollector) Collect(context.Context) systemservice.HostMetrics {
	if len(c.items) == 0 {
		return systemservice.HostMetrics{}
	}
	if c.index >= len(c.items) {
		return c.items[len(c.items)-1]
	}
	item := c.items[c.index]
	c.index++
	return item
}

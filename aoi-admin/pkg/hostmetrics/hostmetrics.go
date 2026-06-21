package hostmetrics

import (
	"context"
	"math"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	gopsnet "github.com/shirou/gopsutil/v3/net"
)

const cpuSampleInterval = 200 * time.Millisecond

// Snapshot 是当前主机资源采样快照。
type Snapshot struct {
	CPU     CPUInfo
	Disk    []DiskInfo
	DiskIO  []DiskIOInfo
	Network NetworkInfo
	RAM     RAMInfo
}

// CPUInfo 描述主机 CPU 核心数和按逻辑核心采样的使用率。
type CPUInfo struct {
	Cores   int
	Percent []float64
}

// RAMInfo 描述主机内存使用情况。
type RAMInfo struct {
	TotalMB     uint64
	UsedMB      uint64
	UsedPercent float64
}

// NetworkInfo 描述主机网络接口累计收发字节数。
type NetworkInfo struct {
	ReceiveBytes  uint64
	TransmitBytes uint64
}

// DiskInfo 描述一个可读挂载点的磁盘使用情况。
type DiskInfo struct {
	FSType      string
	MountPoint  string
	TotalGB     uint64
	TotalMB     uint64
	UsedGB      uint64
	UsedMB      uint64
	UsedPercent float64
}

// DiskIOInfo describes cumulative IO counters for one operating-system disk.
type DiskIOInfo struct {
	Name        string
	ReadBytes   uint64
	WriteBytes  uint64
	ReadCount   uint64
	WriteCount  uint64
	ReadTimeMs  uint64
	WriteTimeMs uint64
}

// Collect 采集当前主机的 CPU、内存和磁盘信息。
func Collect(ctx context.Context) Snapshot {
	return Snapshot{
		CPU:     collectCPU(ctx),
		Disk:    collectDisks(),
		DiskIO:  collectDiskIO(),
		Network: collectNetwork(),
		RAM:     collectRAM(),
	}
}

// collectCPU 采用短时间采样获取每核占用率；采样失败时仍返回可用核心数作为降级信息。
func collectCPU(ctx context.Context) CPUInfo {
	cores, err := cpu.Counts(false)
	if err != nil || cores < 1 {
		cores = runtime.NumCPU()
	}

	select {
	case <-ctx.Done():
		return CPUInfo{Cores: cores}
	default:
	}

	percentages, err := cpu.Percent(cpuSampleInterval, true)
	if err != nil {
		return CPUInfo{Cores: cores}
	}
	out := make([]float64, 0, len(percentages))
	for _, value := range percentages {
		out = append(out, roundPercent(value))
	}
	return CPUInfo{
		Cores:   cores,
		Percent: out,
	}
}

// collectRAM 读取虚拟内存快照；底层平台不支持时返回零值，避免监控接口整体失败。
func collectRAM() RAMInfo {
	stats, err := mem.VirtualMemory()
	if err != nil || stats == nil {
		return RAMInfo{}
	}
	return RAMInfo{
		TotalMB:     bytesToMB(stats.Total),
		UsedMB:      bytesToMB(stats.Used),
		UsedPercent: roundPercent(stats.UsedPercent),
	}
}

// collectDisks 汇总可读挂载点并按挂载路径排序，去重可避免同一挂载点被重复展示。
func collectDisks() []DiskInfo {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return []DiskInfo{}
	}

	out := make([]DiskInfo, 0, len(partitions))
	seen := make(map[string]struct{}, len(partitions))
	for _, partition := range partitions {
		mountPoint := strings.TrimSpace(partition.Mountpoint)
		if mountPoint == "" {
			continue
		}
		key := strings.ToLower(mountPoint)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		usage, err := disk.Usage(mountPoint)
		if err != nil || usage == nil || usage.Total == 0 {
			continue
		}
		fsType := strings.TrimSpace(usage.Fstype)
		if fsType == "" {
			fsType = strings.TrimSpace(partition.Fstype)
		}
		out = append(out, DiskInfo{
			FSType:      fsType,
			MountPoint:  mountPoint,
			TotalGB:     bytesToGB(usage.Total),
			TotalMB:     bytesToMB(usage.Total),
			UsedGB:      bytesToGB(usage.Used),
			UsedMB:      bytesToMB(usage.Used),
			UsedPercent: roundPercent(usage.UsedPercent),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].MountPoint < out[j].MountPoint
	})
	return out
}

func collectDiskIO() []DiskIOInfo {
	counters, err := disk.IOCounters()
	if err != nil || len(counters) == 0 {
		return []DiskIOInfo{}
	}
	names := make([]string, 0, len(counters))
	for name := range counters {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]DiskIOInfo, 0, len(names))
	for _, name := range names {
		counter := counters[name]
		out = append(out, DiskIOInfo{
			Name:        name,
			ReadBytes:   counter.ReadBytes,
			WriteBytes:  counter.WriteBytes,
			ReadCount:   counter.ReadCount,
			WriteCount:  counter.WriteCount,
			ReadTimeMs:  counter.ReadTime,
			WriteTimeMs: counter.WriteTime,
		})
	}
	return out
}

// collectNetwork 读取主机网络累计收发字节数；平台不支持时返回零值。
func collectNetwork() NetworkInfo {
	counters, err := gopsnet.IOCounters(false)
	if err != nil || len(counters) == 0 {
		return NetworkInfo{}
	}
	return NetworkInfo{
		ReceiveBytes:  counters[0].BytesRecv,
		TransmitBytes: counters[0].BytesSent,
	}
}

// bytesToMB 使用二进制单位换算，和多数操作系统资源统计保持一致。
func bytesToMB(value uint64) uint64 {
	const bytesPerMB = 1024 * 1024
	return value / bytesPerMB
}

// bytesToGB 使用二进制单位换算，便于与 MB 字段互相校验。
func bytesToGB(value uint64) uint64 {
	const bytesPerGB = 1024 * 1024 * 1024
	return value / bytesPerGB
}

// roundPercent 统一资源百分比精度，并把异常浮点值收敛为可展示的零值。
func roundPercent(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	return math.Round(value*10) / 10
}

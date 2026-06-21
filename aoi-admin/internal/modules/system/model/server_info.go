package model

import "time"

type ServerInfo struct {
	Build       ServerBuildInfo   `json:"build"`
	CPU         ServerCPUInfo     `json:"cpu"`
	Disk        []ServerDiskInfo  `json:"disk"`
	GC          ServerGCInfo      `json:"gc"`
	Memory      ServerMemoryInfo  `json:"memory"`
	OS          ServerOSInfo      `json:"os"`
	RAM         ServerRAMInfo     `json:"ram"`
	RefreshedAt time.Time         `json:"refreshedAt"`
	Runtime     ServerRuntimeInfo `json:"runtime"`
}

type ServerMetricsHistory struct {
	IntervalSeconds int                   `json:"intervalSeconds"`
	Samples         []ServerMetricsSample `json:"samples"`
	WindowSeconds   int                   `json:"windowSeconds"`
}

type ServerMetricsSample struct {
	CPUUsedPercent             float64              `json:"cpuUsedPercent"`
	DiskMaxUsedPercent         float64              `json:"diskMaxUsedPercent"`
	DiskReadMBPerSecond        float64              `json:"diskReadMbPerSecond"`
	DiskWriteMBPerSecond       float64              `json:"diskWriteMbPerSecond"`
	DiskReadOpsPerSecond       float64              `json:"diskReadOpsPerSecond"`
	DiskWriteOpsPerSecond      float64              `json:"diskWriteOpsPerSecond"`
	DiskIOLatencyMs            float64              `json:"diskIoLatencyMs"`
	DiskIO                     []ServerDiskIOSample `json:"diskIo"`
	Goroutines                 int                  `json:"goroutines"`
	HeapAllocMB                uint64               `json:"heapAllocMb"`
	NetworkReceiveKBPerSecond  float64              `json:"networkReceiveKbPerSecond"`
	NetworkTransmitKBPerSecond float64              `json:"networkTransmitKbPerSecond"`
	RAMUsedPercent             float64              `json:"ramUsedPercent"`
	SampledAt                  time.Time            `json:"sampledAt"`
}

type ServerDiskIOSample struct {
	Name              string  `json:"name"`
	ReadMBPerSecond   float64 `json:"readMbPerSecond"`
	WriteMBPerSecond  float64 `json:"writeMbPerSecond"`
	ReadOpsPerSecond  float64 `json:"readOpsPerSecond"`
	WriteOpsPerSecond float64 `json:"writeOpsPerSecond"`
	IOLatencyMs       float64 `json:"ioLatencyMs"`
}

type ServerOSInfo struct {
	Compiler     string `json:"compiler"`
	GoArch       string `json:"goarch"`
	GoOS         string `json:"goos"`
	GoVersion    string `json:"goVersion"`
	NumCPU       int    `json:"numCpu"`
	NumGoroutine int    `json:"numGoroutine"`
}

type ServerRuntimeInfo struct {
	StartTime     time.Time `json:"startTime"`
	Uptime        string    `json:"uptime"`
	UptimeSeconds int64     `json:"uptimeSeconds"`
}

type ServerCPUInfo struct {
	Cores   int       `json:"cores"`
	Percent []float64 `json:"percent"`
}

type ServerRAMInfo struct {
	TotalMB     uint64  `json:"totalMb"`
	UsedMB      uint64  `json:"usedMb"`
	UsedPercent float64 `json:"usedPercent"`
}

type ServerDiskInfo struct {
	FSType      string  `json:"fsType"`
	MountPoint  string  `json:"mountPoint"`
	TotalGB     uint64  `json:"totalGb"`
	TotalMB     uint64  `json:"totalMb"`
	UsedGB      uint64  `json:"usedGb"`
	UsedMB      uint64  `json:"usedMb"`
	UsedPercent float64 `json:"usedPercent"`
}

type ServerMemoryInfo struct {
	AllocMB        uint64 `json:"allocMb"`
	HeapAllocMB    uint64 `json:"heapAllocMb"`
	HeapIdleMB     uint64 `json:"heapIdleMb"`
	HeapInuseMB    uint64 `json:"heapInuseMb"`
	HeapObjects    uint64 `json:"heapObjects"`
	HeapReleasedMB uint64 `json:"heapReleasedMb"`
	HeapSysMB      uint64 `json:"heapSysMb"`
	StackInuseMB   uint64 `json:"stackInuseMb"`
	StackSysMB     uint64 `json:"stackSysMb"`
	SysMB          uint64 `json:"sysMb"`
	TotalAllocMB   uint64 `json:"totalAllocMb"`
}

type ServerGCInfo struct {
	LastGCAt     *time.Time `json:"lastGcAt,omitempty"`
	NextGCMB     uint64     `json:"nextGcMb"`
	NumGC        uint32     `json:"numGc"`
	PauseTotalNs uint64     `json:"pauseTotalNs"`
}

type ServerBuildInfo struct {
	GoVersion string               `json:"goVersion"`
	Module    string               `json:"module"`
	Path      string               `json:"path"`
	Settings  []ServerBuildSetting `json:"settings"`
	Version   string               `json:"version"`
}

type ServerBuildSetting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

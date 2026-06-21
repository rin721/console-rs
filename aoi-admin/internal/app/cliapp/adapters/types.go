package adapters

import "time"

const (
	RuntimeDirEnvName        = "RIN_CLI_RUNTIME_DIR"
	ManagedServiceEnvName    = "RIN_CLI_MANAGED"
	ManagedServiceNameEnvKey = "RIN_CLI_SERVICE"

	ControlActionStop = "stop"
)

// ControlRequest 是 CLI 写给托管服务进程的控制消息。
type ControlRequest struct {
	Service          string    `json:"service"`
	Action           string    `json:"action"`
	PID              int       `json:"pid"`
	ProcessStartTime int64     `json:"processStartTime"`
	RequestedAt      time.Time `json:"requestedAt"`
}

// ProcessInfo 描述一个已启动或已探测进程。
type ProcessInfo struct {
	PID              int
	ProcessStartTime int64
}

// ProcessDetails 描述从操作系统发现的进程信息。
type ProcessDetails struct {
	ProcessInfo
	Executable  string
	CommandLine string
}

// ProcessStartRequest 描述后台进程启动请求。
type ProcessStartRequest struct {
	Executable string
	Args       []string
	WorkDir    string
	Env        []string
	StdoutPath string
	StderrPath string
}

// ProcessRunner 隔离真实操作系统进程操作，便于测试服务状态机。
type ProcessRunner interface {
	StartProcess(ProcessStartRequest) (ProcessInfo, error)
	IsProcessRunning(ProcessInfo) (bool, error)
	KillProcess(ProcessInfo) error
	FindTCPListener(addr string) (ProcessDetails, bool, error)
}

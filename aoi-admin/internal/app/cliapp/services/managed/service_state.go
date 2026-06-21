package managed

import "time"

const (
	ServiceServer = "server"

	StatusStarting   = "starting"
	StatusRunning    = "running"
	StatusStopped    = "stopped"
	StatusFailed     = "failed"
	StatusRestarting = "restarting"
	StatusUnmanaged  = "unmanaged"
)

// ServiceState 是 CLI 后台服务管理的持久化状态。
type ServiceState struct {
	Service          string     `json:"service"`
	Status           string     `json:"status"`
	PID              int        `json:"pid"`
	ProcessStartTime int64      `json:"processStartTime"`
	StartedAt        *time.Time `json:"startedAt,omitempty"`
	StoppedAt        *time.Time `json:"stoppedAt,omitempty"`
	ConfigPath       string     `json:"configPath"`
	ListenAddr       string     `json:"listenAddr"`
	ExecutablePath   string     `json:"executablePath,omitempty"`
	StdoutLogPath    string     `json:"stdoutLogPath"`
	StderrLogPath    string     `json:"stderrLogPath"`
	AppLogPath       string     `json:"appLogPath"`
	LastError        string     `json:"lastError,omitempty"`
	Unmanaged        bool       `json:"unmanaged,omitempty"`
}

package managed

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	cliappadapters "github.com/rei0721/go-scaffold/internal/app/cliapp/adapters"
)

const defaultStopTimeout = 30 * time.Second

// Manager 管理 CLI 托管的后台服务进程。
type Manager struct {
	RuntimeDir string
	Executable string
	WorkDir    string
	Runner     cliappadapters.ProcessRunner
	Now        func() time.Time
}

// NewManager 创建默认服务管理器。
func NewManager() *Manager {
	executable, _ := os.Executable()
	workDir, _ := os.Getwd()
	runtimeDir := strings.TrimSpace(os.Getenv(cliappadapters.RuntimeDirEnvName))
	if runtimeDir == "" {
		runtimeDir = filepath.Join("data", "cli-runtime")
	}
	return &Manager{
		RuntimeDir: runtimeDir,
		Executable: executable,
		WorkDir:    workDir,
		Runner:     cliappadapters.NewOSProcessRunner(),
		Now:        time.Now,
	}
}

// MarkManagedServiceStopped 供托管 server 进程在优雅退出后更新状态文件。
func MarkManagedServiceStopped(service string, lastError string) {
	if os.Getenv(cliappadapters.ManagedServiceEnvName) == "" {
		return
	}
	manager := NewManager()
	state, err := manager.readState(NormalizeServiceName(service))
	if err != nil {
		return
	}
	stoppedAt := manager.now()
	state.Status = StatusStopped
	if strings.TrimSpace(lastError) != "" {
		state.Status = StatusFailed
		state.LastError = lastError
	}
	state.PID = 0
	state.ProcessStartTime = 0
	state.StoppedAt = &stoppedAt
	_ = manager.writeState(state)
}

func (m *Manager) workDir() string {
	if strings.TrimSpace(m.WorkDir) != "" {
		return m.WorkDir
	}
	workDir, _ := os.Getwd()
	return workDir
}

func (m *Manager) runner() cliappadapters.ProcessRunner {
	if m.Runner != nil {
		return m.Runner
	}
	return cliappadapters.NewOSProcessRunner()
}

func (m *Manager) now() time.Time {
	if m.Now != nil {
		return m.Now().UTC()
	}
	return time.Now().UTC()
}

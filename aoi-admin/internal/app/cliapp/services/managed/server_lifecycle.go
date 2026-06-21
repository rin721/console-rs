package managed

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cliappadapters "github.com/rei0721/go-scaffold/internal/app/cliapp/adapters"
	cliconfig "github.com/rei0721/go-scaffold/internal/app/cliapp/config"
	"github.com/rei0721/go-scaffold/types/constants"
)

// StartServer 后台启动 server 服务。
func (m *Manager) StartServer(ctx context.Context, configPath string) (ServiceState, error) {
	if err := ctx.Err(); err != nil {
		return ServiceState{}, err
	}
	if configPath == "" {
		configPath = constants.AppDefaultConfigPath
	}
	configPath = filepath.Clean(configPath)

	current, err := m.Status(ctx, ServiceServer)
	if err != nil {
		return current, err
	}
	if current.Status == StatusRunning || current.Status == StatusStarting || current.Status == StatusRestarting || current.Status == StatusUnmanaged {
		return current, fmt.Errorf("%s service is already %s", ServiceServer, current.Status)
	}

	cfg, err := cliconfig.LoadConfig(configPath)
	if err != nil {
		err = cliconfig.ActionableConfigLoadError(configPath, err)
		state := m.baseState(ServiceServer, configPath, nil)
		state.Status = StatusFailed
		state.LastError = err.Error()
		_ = m.writeState(state)
		return state, err
	}

	runtimeDir, err := filepath.Abs(m.runtimeDir())
	if err != nil {
		runtimeDir = m.runtimeDir()
	}
	state := m.baseState(ServiceServer, configPath, cfg)
	executablePath, err := m.managedExecutable(runtimeDir)
	if err != nil {
		failedAt := m.now()
		state.Status = StatusFailed
		state.LastError = err.Error()
		state.StoppedAt = &failedAt
		_ = m.writeState(state)
		return state, err
	}
	state.ExecutablePath = executablePath
	startedAt := m.now()
	state.Status = StatusStarting
	state.StartedAt = &startedAt
	state.StoppedAt = nil
	state.LastError = ""
	if err := m.writeState(state); err != nil {
		return state, err
	}
	_ = os.Remove(m.controlPath(ServiceServer))

	info, err := m.runner().StartProcess(cliappadapters.ProcessStartRequest{
		Executable: executablePath,
		Args:       []string{constants.AppServerCommandName, "--config", configPath},
		WorkDir:    m.workDir(),
		Env: []string{
			cliappadapters.ManagedServiceEnvName + "=1",
			cliappadapters.ManagedServiceNameEnvKey + "=" + ServiceServer,
			cliappadapters.RuntimeDirEnvName + "=" + runtimeDir,
		},
		StdoutPath: state.StdoutLogPath,
		StderrPath: state.StderrLogPath,
	})
	if err != nil {
		failedAt := m.now()
		state.Status = StatusFailed
		state.LastError = err.Error()
		state.StoppedAt = &failedAt
		_ = m.writeState(state)
		return state, err
	}

	state.PID = info.PID
	state.ProcessStartTime = info.ProcessStartTime
	state.Status = StatusRunning
	if alive, _ := m.runner().IsProcessRunning(info); !alive {
		stoppedAt := m.now()
		state.Status = StatusFailed
		state.LastError = "process exited during startup"
		state.StoppedAt = &stoppedAt
	}
	if err := m.writeState(state); err != nil {
		return state, err
	}
	return state, nil
}

// Status 返回服务状态，并刷新已退出的后台进程。
func (m *Manager) Status(ctx context.Context, service string) (ServiceState, error) {
	if err := ctx.Err(); err != nil {
		return ServiceState{}, err
	}
	service = NormalizeServiceName(service)
	state, err := m.readState(service)
	if err != nil {
		return ServiceState{}, err
	}
	if state.Service == "" {
		state = ServiceState{Service: service, Status: StatusStopped}
	}
	if state.PID <= 0 || !ActiveStatus(state.Status) {
		return m.withUnmanagedServerStatus(state), nil
	}
	running, err := m.runner().IsProcessRunning(cliappadapters.ProcessInfo{PID: state.PID, ProcessStartTime: state.ProcessStartTime})
	if err != nil {
		return state, err
	}
	if running {
		if state.Status == StatusStarting || state.Status == StatusRestarting {
			state.Status = StatusRunning
			_ = m.writeState(state)
		}
		return state, nil
	}
	stoppedAt := m.now()
	if state.Status != StatusStopped {
		state.Status = StatusFailed
		state.LastError = "process is not running"
	}
	state.StoppedAt = &stoppedAt
	state.PID = 0
	state.ProcessStartTime = 0
	_ = m.writeState(state)
	return m.withUnmanagedServerStatus(state), nil
}

// StopServer 优雅停止 server 服务，超时后强制结束。
func (m *Manager) StopServer(ctx context.Context) (ServiceState, error) {
	state, err := m.Status(ctx, ServiceServer)
	if err != nil {
		return state, err
	}
	if state.Status == StatusUnmanaged {
		return m.stopProcessByInfo(ctx, state, cliappadapters.ProcessInfo{PID: state.PID, ProcessStartTime: state.ProcessStartTime}, false)
	}
	if state.Status != StatusRunning && state.Status != StatusStarting && state.Status != StatusRestarting {
		state.Status = StatusStopped
		_ = m.writeState(state)
		return state, nil
	}

	info := cliappadapters.ProcessInfo{PID: state.PID, ProcessStartTime: state.ProcessStartTime}
	return m.stopProcessByInfo(ctx, state, info, true)
}

func (m *Manager) stopProcessByInfo(ctx context.Context, state ServiceState, info cliappadapters.ProcessInfo, graceful bool) (ServiceState, error) {
	if info.PID <= 0 {
		state.Status = StatusStopped
		state.PID = 0
		state.ProcessStartTime = 0
		state.Unmanaged = false
		_ = m.writeState(state)
		return state, nil
	}
	if graceful {
		if err := m.writeControl(cliappadapters.ControlRequest{
			Service:          ServiceServer,
			Action:           cliappadapters.ControlActionStop,
			PID:              info.PID,
			ProcessStartTime: info.ProcessStartTime,
			RequestedAt:      m.now(),
		}); err != nil {
			return state, err
		}
	}
	if !graceful {
		if err := m.runner().KillProcess(info); err != nil {
			return state, err
		}
		stoppedAt := m.now()
		state.Status = StatusStopped
		state.PID = 0
		state.ProcessStartTime = 0
		state.Unmanaged = false
		state.StoppedAt = &stoppedAt
		state.LastError = ""
		_ = m.writeState(state)
		_ = os.Remove(m.controlPath(ServiceServer))
		return state, nil
	}

	deadline := m.now().Add(defaultStopTimeout)
	for m.now().Before(deadline) {
		running, err := m.runner().IsProcessRunning(info)
		if err != nil {
			return state, err
		}
		if !running {
			stoppedAt := m.now()
			state.Status = StatusStopped
			state.PID = 0
			state.ProcessStartTime = 0
			state.Unmanaged = false
			state.StoppedAt = &stoppedAt
			state.LastError = ""
			_ = m.writeState(state)
			_ = os.Remove(m.controlPath(ServiceServer))
			return state, nil
		}
		select {
		case <-ctx.Done():
			return state, ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}

	if err := m.runner().KillProcess(info); err != nil {
		return state, err
	}
	stoppedAt := m.now()
	state.Status = StatusStopped
	state.PID = 0
	state.ProcessStartTime = 0
	state.Unmanaged = false
	state.StoppedAt = &stoppedAt
	state.LastError = "forced stop after graceful timeout"
	_ = m.writeState(state)
	_ = os.Remove(m.controlPath(ServiceServer))
	return state, nil
}

func (m *Manager) withUnmanagedServerStatus(state ServiceState) ServiceState {
	if state.Service == "" {
		state.Service = ServiceServer
	}
	if NormalizeServiceName(state.Service) != ServiceServer || strings.TrimSpace(state.ListenAddr) == "" {
		return state
	}
	details, ok, err := m.runner().FindTCPListener(state.ListenAddr)
	if err != nil || !ok || details.PID <= 0 {
		return state
	}
	if state.PID == details.PID && state.ProcessStartTime == details.ProcessStartTime {
		return state
	}
	if !looksLikeAoiServerProcess(details, state.ConfigPath) {
		if state.Status == "" || state.Status == StatusStopped {
			state.Status = StatusFailed
		}
		state.LastError = fmt.Sprintf("listen address is occupied by unmanaged process pid %d", details.PID)
		return state
	}
	state.Status = StatusUnmanaged
	state.PID = details.PID
	state.ProcessStartTime = details.ProcessStartTime
	state.ExecutablePath = details.Executable
	state.Unmanaged = true
	state.LastError = "server is running outside the CLI service manager"
	return state
}

func looksLikeAoiServerProcess(details cliappadapters.ProcessDetails, configPath string) bool {
	command := normalizeProcessText(details.CommandLine)
	executable := normalizeProcessText(details.Executable)
	if !strings.Contains(command, " server") && !strings.Contains(command, "server ") {
		return false
	}
	if strings.Contains(executable, "aoi") || strings.Contains(command, "cmd/aoi") || strings.Contains(command, "cmd\\aoi") || strings.Contains(command, " aoi") {
		return true
	}
	configPath = normalizeProcessText(configPath)
	return configPath != "" && strings.Contains(command, configPath)
}

func normalizeProcessText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "\\", "/")
	return value
}

// RestartServer 重启 server 服务，沿用上次配置路径。
func (m *Manager) RestartServer(ctx context.Context) (ServiceState, error) {
	state, err := m.Status(ctx, ServiceServer)
	if err != nil {
		return state, err
	}
	configPath := state.ConfigPath
	if configPath == "" {
		configPath = constants.AppDefaultConfigPath
	}
	state.Status = StatusRestarting
	_ = m.writeState(state)
	if _, err := m.StopServer(ctx); err != nil {
		return state, err
	}
	return m.StartServer(ctx, configPath)
}

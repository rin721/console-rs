package managed

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	cliappadapters "github.com/rei0721/go-scaffold/internal/app/cliapp/adapters"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
)

// baseState 构造服务状态文件的初始内容。
//
// cfg 为空时仍会返回可写状态，便于配置加载失败时记录失败原因；日志路径始终基于服务运行目录生成。
func (m *Manager) baseState(service string, configPath string, cfg *appconfig.Config) ServiceState {
	service = NormalizeServiceName(service)
	serviceDir := m.serviceDir(service)
	state := ServiceState{
		Service:       service,
		Status:        StatusStopped,
		ConfigPath:    configPath,
		StdoutLogPath: filepath.Join(serviceDir, "stdout.log"),
		StderrLogPath: filepath.Join(serviceDir, "stderr.log"),
	}
	if cfg != nil {
		state.ListenAddr = net.JoinHostPort(cfg.Server.Host, fmt.Sprint(cfg.Server.Port))
		state.AppLogPath = cfg.Logger.FilePath
	}
	return state
}

// readState 读取持久化服务状态。
//
// 状态文件不存在时视为服务已停止；缺失的 service/status 字段会被补齐以兼容旧状态文件。
func (m *Manager) readState(service string) (ServiceState, error) {
	service = NormalizeServiceName(service)
	raw, err := os.ReadFile(m.statePath(service))
	if errors.Is(err, os.ErrNotExist) {
		return ServiceState{Service: service, Status: StatusStopped}, nil
	}
	if err != nil {
		return ServiceState{}, err
	}
	var state ServiceState
	if err := json.Unmarshal(raw, &state); err != nil {
		return ServiceState{}, err
	}
	if state.Service == "" {
		state.Service = service
	}
	if state.Status == "" {
		state.Status = StatusStopped
	}
	return state, nil
}

// writeState 原子写入服务状态文件。
//
// 先写临时文件再 rename，避免 CLI 进程或托管服务同时读取时看到半截 JSON。
func (m *Manager) writeState(state ServiceState) error {
	if state.Service == "" {
		state.Service = ServiceServer
	}
	if state.Status == "" {
		state.Status = StatusStopped
	}
	path := m.statePath(state.Service)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	_ = os.Remove(path)
	return os.Rename(tmp, path)
}

// writeControl 写入托管服务会轮询读取的控制请求。
//
// 控制文件是一次性信号，服务进程匹配并消费后会删除它。
func (m *Manager) writeControl(req cliappadapters.ControlRequest) error {
	path := m.controlPath(req.Service)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func (m *Manager) statePath(service string) string {
	return filepath.Join(m.serviceDir(service), "state.json")
}

func (m *Manager) controlPath(service string) string {
	return filepath.Join(m.serviceDir(service), "control.json")
}

// ControlPath 返回托管服务控制文件路径。
func (m *Manager) ControlPath(service string) string {
	return m.controlPath(service)
}

func (m *Manager) serviceDir(service string) string {
	return filepath.Join(m.runtimeDir(), NormalizeServiceName(service))
}

func (m *Manager) runtimeDir() string {
	if strings.TrimSpace(m.RuntimeDir) != "" {
		return filepath.Clean(m.RuntimeDir)
	}
	return filepath.Join("data", "cli-runtime")
}

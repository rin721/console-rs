package adapters

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rei0721/go-scaffold/pkg/processx"
)

type osProcessRunner struct{}

// NewOSProcessRunner 创建默认的系统进程运行器。
func NewOSProcessRunner() ProcessRunner {
	return osProcessRunner{}
}

// StartProcess 启动脱离当前 CLI 的后台进程，并把 stdout/stderr 追加写入指定日志文件。
func (osProcessRunner) StartProcess(req ProcessStartRequest) (ProcessInfo, error) {
	if req.Executable == "" {
		return ProcessInfo{}, errors.New("executable is required")
	}
	if err := os.MkdirAll(filepath.Dir(req.StdoutPath), 0o755); err != nil {
		return ProcessInfo{}, err
	}
	if err := os.MkdirAll(filepath.Dir(req.StderrPath), 0o755); err != nil {
		return ProcessInfo{}, err
	}
	stdout, err := os.OpenFile(req.StdoutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return ProcessInfo{}, err
	}
	defer stdout.Close()
	stderr, err := os.OpenFile(req.StderrPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return ProcessInfo{}, err
	}
	defer stderr.Close()

	cmd := exec.Command(req.Executable, req.Args...)
	cmd.Dir = req.WorkDir
	cmd.Env = append(os.Environ(), req.Env...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	configureDetachedProcess(cmd)

	if err := cmd.Start(); err != nil {
		return ProcessInfo{}, err
	}
	info := ProcessInfo{PID: cmd.Process.Pid}
	info.ProcessStartTime, _ = processCreateTime(info.PID)
	_ = cmd.Process.Release()
	return info, nil
}

// IsProcessRunning 根据 PID 和创建时间判断目标进程是否仍是同一个进程。
func (osProcessRunner) IsProcessRunning(info ProcessInfo) (bool, error) {
	return processx.IsRunning(info.PID, info.ProcessStartTime)
}

// KillProcess 强制结束目标进程。
func (osProcessRunner) KillProcess(info ProcessInfo) error {
	if info.PID <= 0 {
		return nil
	}
	proc, err := os.FindProcess(info.PID)
	if err != nil {
		return nil
	}
	return proc.Kill()
}

func (osProcessRunner) FindTCPListener(addr string) (ProcessDetails, bool, error) {
	pid, ok, err := processx.TCPListenerPID(addr)
	if err != nil || !ok {
		return ProcessDetails{}, ok, err
	}
	details := ProcessDetails{ProcessInfo: ProcessInfo{PID: pid}}
	details.ProcessStartTime, _ = processCreateTime(pid)
	details.Executable, _ = processx.Exe(pid)
	details.CommandLine, _ = processx.CommandLine(pid)
	return details, true, nil
}

func processCreateTime(pid int) (int64, error) {
	var lastErr error
	for attempt := 0; attempt < 10; attempt++ {
		createTime, err := processx.CreateTime(pid)
		if err == nil && createTime > 0 {
			return createTime, nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return 0, lastErr
}

// CurrentProcessInfo 返回当前进程标识，用于托管服务匹配 CLI 控制请求。
func CurrentProcessInfo() ProcessInfo {
	pid := os.Getpid()
	createTime, _ := processCreateTime(pid)
	return ProcessInfo{PID: pid, ProcessStartTime: createTime}
}

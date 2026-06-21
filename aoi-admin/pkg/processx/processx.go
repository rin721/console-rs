package processx

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	gopsnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// CreateTime 返回指定 PID 的进程创建时间，单位沿用 gopsutil 的毫秒时间戳。
func CreateTime(pid int) (int64, error) {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return 0, err
	}
	return proc.CreateTime()
}

// IsRunning 校验指定 PID 是否仍在运行；createTime 大于 0 时会同时比对创建时间，避免 PID 复用误判。
func IsRunning(pid int, createTime int64) (bool, error) {
	if pid <= 0 {
		return false, nil
	}
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return false, nil
	}
	running, err := proc.IsRunning()
	if err != nil || !running {
		return false, err
	}
	if createTime <= 0 {
		return true, nil
	}
	actualCreateTime, err := proc.CreateTime()
	if err != nil {
		return false, err
	}
	return actualCreateTime == createTime, nil
}

// TCPListenerPID 返回监听指定地址的进程 PID。
func TCPListenerPID(addr string) (int, bool, error) {
	host, portText, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return 0, false, err
	}
	port, err := strconv.ParseUint(portText, 10, 32)
	if err != nil {
		return 0, false, err
	}
	connections, err := gopsnet.Connections("tcp")
	if err != nil {
		return 0, false, err
	}
	for _, conn := range connections {
		if !strings.EqualFold(conn.Status, "LISTEN") {
			continue
		}
		if uint64(conn.Laddr.Port) != port {
			continue
		}
		if !sameListenHost(host, conn.Laddr.IP) {
			continue
		}
		if conn.Pid <= 0 {
			return 0, false, fmt.Errorf("listener on %s has no process id", addr)
		}
		return int(conn.Pid), true, nil
	}
	return 0, false, nil
}

// Exe 返回指定 PID 的可执行文件路径。
func Exe(pid int) (string, error) {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return "", err
	}
	return proc.Exe()
}

// CommandLine 返回指定 PID 的命令行。
func CommandLine(pid int) (string, error) {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return "", err
	}
	return proc.Cmdline()
}

func sameListenHost(want string, got string) bool {
	want = strings.TrimSpace(want)
	got = strings.TrimSpace(got)
	if want == "" || want == "0.0.0.0" || want == "::" || want == "[::]" {
		return true
	}
	if got == "" || got == "0.0.0.0" || got == "::" || got == "[::]" {
		return true
	}
	wantIP := net.ParseIP(want)
	gotIP := net.ParseIP(got)
	if wantIP != nil && gotIP != nil {
		return wantIP.Equal(gotIP)
	}
	return strings.EqualFold(want, got)
}

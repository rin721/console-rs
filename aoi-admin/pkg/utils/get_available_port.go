package utils

// 本文件属于通用工具层，提供无业务状态的地址校验、端口选择、设备标识或国际化代理能力。

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

var portScanLock sync.Mutex // 防止并发抢同一端口

// GetAvailablePort 在指定范围内获取一个可用 TCP 端口
//
// start: 起始端口，例如 30000
// end:   结束端口，例如 40000
// exclude: 需要排除的端口
//
// 返回：可用端口 or error
func GetAvailablePort(start, end int, exclude ...int) (int, error) {
	if start <= 0 || end > 65535 || start > end {
		return 0, errors.New("invalid port range")
	}

	excludeSet := make(map[int]struct{}, len(exclude))
	for _, p := range exclude {
		excludeSet[p] = struct{}{}
	}

	portScanLock.Lock()
	defer portScanLock.Unlock()

	for port := start; port <= end; port++ {
		if _, skip := excludeSet[port]; skip {
			continue
		}

		if isPortAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available port in range %d-%d", start, end)
}

// isPortAvailable 尝试真实 bind TCP 端口
func isPortAvailable(port int) bool {
	addr := fmt.Sprintf("0.0.0.0:%d", port)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}

	_ = l.Close()
	return true
}

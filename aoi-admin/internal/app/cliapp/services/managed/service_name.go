package managed

import (
	"strings"
)

// normalizeServiceName 统一 CLI 服务名，空值默认指向 server。
func NormalizeServiceName(service string) string {
	service = strings.ToLower(strings.TrimSpace(service))
	if service == "" {
		return ServiceServer
	}
	return service
}

// activeStatus 判断状态是否仍代表可能存在的后台进程。
func ActiveStatus(status string) bool {
	switch status {
	case StatusStarting, StatusRunning, StatusRestarting:
		return true
	default:
		return false
	}
}

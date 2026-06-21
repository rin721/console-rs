package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"
)

// WatchManagedServiceControl 监听 CLI 写入的控制文件。
func WatchManagedServiceControl(ctx context.Context, service string, controlPath string) <-chan ControlRequest {
	if os.Getenv(ManagedServiceEnvName) == "" {
		return nil
	}
	out := make(chan ControlRequest, 1)
	service = normalizeServiceName(service)
	self := CurrentProcessInfo()
	go func() {
		defer close(out)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				req, ok := readControlRequest(controlPath)
				if !ok || !matchesCurrentProcess(req, service, self) {
					continue
				}
				_ = os.Remove(controlPath)
				out <- req
				return
			}
		}
	}()
	return out
}

func readControlRequest(path string) (ControlRequest, bool) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) || err != nil {
		return ControlRequest{}, false
	}
	var req ControlRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return ControlRequest{}, false
	}
	return req, true
}

func matchesCurrentProcess(req ControlRequest, service string, self ProcessInfo) bool {
	if normalizeServiceName(req.Service) != service {
		return false
	}
	if req.Action != ControlActionStop {
		return false
	}
	if req.PID != self.PID {
		return false
	}
	if req.ProcessStartTime > 0 && self.ProcessStartTime > 0 && req.ProcessStartTime != self.ProcessStartTime {
		return false
	}
	return true
}

func normalizeServiceName(service string) string {
	return strings.ToLower(strings.TrimSpace(service))
}

package rpctransport

import (
	"context"
	"encoding/json"

	"github.com/rei0721/go-scaffold/internal/ports"
)

// Register 注册内置 RPC 方法，供外部调用方做连通性检查和方法发现。
func Register(registry ports.RPCRegistry) error {
	if err := registry.Register("system.ping", ping); err != nil {
		return err
	}
	if err := registry.Register("system.methods", func(context.Context, json.RawMessage) (any, error) {
		return registry.Methods(), nil
	}); err != nil {
		return err
	}
	return nil
}

// ping 返回最小健康响应；当 params 是对象且包含 echo 时会回显该值，便于客户端调试链路。
func ping(_ context.Context, params json.RawMessage) (any, error) {
	response := map[string]any{"ok": true}
	if len(params) == 0 || string(params) == "null" {
		return response, nil
	}

	var values map[string]any
	if err := json.Unmarshal(params, &values); err != nil {
		return nil, ports.InvalidRPCParams("params must be an object")
	}
	if echo, ok := values["echo"]; ok {
		response["echo"] = echo
	}
	return response, nil
}

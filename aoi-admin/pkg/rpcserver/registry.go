package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Registry 保存可被 RPC 入口调用的方法集合。
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]HandlerFunc
}

// NewRegistry 创建空方法注册表。
func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]HandlerFunc)}
}

// Register 注册一个 RPC 方法。
func (r *Registry) Register(method string, handler HandlerFunc) error {
	if r == nil {
		return errors.New("registry cannot be nil")
	}
	method = strings.TrimSpace(method)
	if method == "" {
		return errors.New("method cannot be empty")
	}
	if handler == nil {
		return fmt.Errorf("handler for %s cannot be nil", method)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.handlers[method]; exists {
		return fmt.Errorf("method %s already registered", method)
	}
	r.handlers[method] = handler
	return nil
}

// Methods 返回已注册方法名的稳定排序列表。
func (r *Registry) Methods() []string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	methods := make([]string, 0, len(r.handlers))
	for method := range r.handlers {
		methods = append(methods, method)
	}
	sort.Strings(methods)
	return methods
}

// Call 调用一个已注册 RPC 方法。
func (r *Registry) Call(ctx context.Context, method string, params json.RawMessage) (any, error) {
	if r == nil {
		return nil, &RPCError{Code: CodeInternalError, Message: "registry is not initialized"}
	}
	r.mu.RLock()
	handler, ok := r.handlers[method]
	r.mu.RUnlock()
	if !ok {
		return nil, &RPCError{Code: CodeMethodNotFound, Message: "method not found"}
	}
	return handler(ctx, params)
}

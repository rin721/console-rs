package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// HandlerFunc 是单个 JSON-RPC 方法的处理函数。
type HandlerFunc func(ctx context.Context, params json.RawMessage) (any, error)

// Server 管理 RPC 服务的生命周期。
type Server interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
	Reload(ctx context.Context, cfg *Config) error
}

// Config 描述 RPC 服务监听和超时配置。
type Config struct {
	Enabled      bool
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// ApplyDefaults 补齐 RPC 服务默认配置。
func (c *Config) ApplyDefaults() {
	if c.Host == "" {
		c.Host = DefaultHost
	}
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = DefaultReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = DefaultWriteTimeout
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = DefaultIdleTimeout
	}
}

// Validate 校验 RPC 服务配置。
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config cannot be nil")
	}
	if !c.Enabled {
		return nil
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("host is required")
	}
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive")
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive")
	}
	if c.IdleTimeout < 0 {
		return fmt.Errorf("idle timeout must be non-negative")
	}
	return nil
}

// Request 是单个 JSON-RPC 2.0 请求。
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response 是单个 JSON-RPC 2.0 响应。
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error 是 JSON-RPC 标准错误对象。
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// RPCError 表示方法处理阶段可控的 JSON-RPC 错误。
type RPCError struct {
	Code    int
	Message string
	Data    any
}

func (e *RPCError) Error() string {
	return e.Message
}

// InvalidParams 构造参数错误。
func InvalidParams(message string) *RPCError {
	return &RPCError{Code: CodeInvalidParams, Message: message}
}

// InternalError 构造内部错误。
func InternalError(message string) *RPCError {
	return &RPCError{Code: CodeInternalError, Message: message}
}

package rpcserver

import "time"

const (
	// DefaultHost 是 RPC 服务默认监听地址。
	DefaultHost = "127.0.0.1"
	// DefaultPort 是 RPC 服务默认监听端口。
	DefaultPort = 10099

	// DefaultReadTimeout 是读取 RPC 请求的默认超时。
	DefaultReadTimeout = 10 * time.Second
	// DefaultWriteTimeout 是写入 RPC 响应的默认超时。
	DefaultWriteTimeout = 10 * time.Second
	// DefaultIdleTimeout 是 RPC keep-alive 连接的默认空闲超时。
	DefaultIdleTimeout = 30 * time.Second
)

const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

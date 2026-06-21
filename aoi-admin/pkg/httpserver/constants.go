package httpserver

// 本文件属于 HTTP 服务封装层，管理 net/http Server 的启动、关闭、地址选择和配置重载。

import "time"

// 默认配置常量
const (
	// DefaultHost 默认监听地址
	DefaultHost = "localhost"

	// DefaultPort 默认监听端口
	DefaultPort = 8080

	// DefaultReadTimeout 默认读取超时时间
	// 包括读取请求头和请求体
	DefaultReadTimeout = 15 * time.Second

	// DefaultWriteTimeout 默认写入超时时间
	// 从请求处理完成到写入完整响应
	DefaultWriteTimeout = 15 * time.Second

	// DefaultIdleTimeout 默认空闲连接超时时间
	// Keep-Alive 连接的最大空闲时间
	DefaultIdleTimeout = 60 * time.Second
)

// 错误消息常量
const (
	// ErrMsgInvalidAddress 无效的监听地址
	ErrMsgInvalidAddress = "invalid listen address"

	// ErrMsgServerStartFailed 服务器启动失败
	ErrMsgServerStartFailed = "failed to start server"

	// ErrMsgServerShutdownFailed 服务器关闭失败
	ErrMsgServerShutdownFailed = "failed to shutdown server"

	// ErrMsgPortUnavailable 端口不可用
	ErrMsgPortUnavailable = "port is not available"

	// ErrMsgServerAlreadyRunning 服务器已经在运行
	ErrMsgServerAlreadyRunning = "server is already running"

	// ErrMsgServerNotRunning 服务器未运行
	ErrMsgServerNotRunning = "server is not running"

	// ErrMsgInvalidConfig 无效的配置
	ErrMsgInvalidConfig = "invalid server config"

	// ErrMsgReloadFailed 配置重载失败
	ErrMsgReloadFailed = "failed to reload server config"
)

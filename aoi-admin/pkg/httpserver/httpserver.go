package httpserver

// 本文件属于 HTTP 服务封装层，管理 net/http Server 的启动、关闭、地址选择和配置重载。

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/rei0721/go-scaffold/pkg/logger"
	"github.com/rei0721/go-scaffold/pkg/utils"
)

// httpServer HTTP 服务器实现
type httpServer struct {
	// server 标准库 http.Server 实例
	server *http.Server

	// handler HTTP 请求处理器 (Gin Router)
	handler Handler

	// config 当前配置
	config *Config

	// logger 日志记录器
	logger logger.Logger

	// mu 保护并发访问（热重载使用）
	mu sync.RWMutex

	// state 服务器运行状态
	state atomic.Int32

	// errChan 服务器错误通道
	errChan chan error

	// doneChan 服务器优雅关闭完成信号
	doneChan chan struct{}

	// doneOnce 确保关闭完成信号只触发一次
	doneOnce sync.Once

	// lastRuntimeErr 记录最近一次异步 Serve 运行期错误
	lastRuntimeErr error
}

// New 创建新的 HTTP Server 实例
// 参数:
//
//	handler: HTTP 请求处理器，通常是 Gin Router
//	cfg: 服务器配置
//	log: 日志记录器
//
// 返回:
//
//	HTTPServer: 服务器实例
//	error: 创建失败时的错误
func New(handler Handler, cfg *Config, log logger.Logger) (HTTPServer, error) {
	if handler == nil {
		return nil, &ServerError{
			Op:      "new",
			Message: "handler cannot be nil",
		}
	}

	if cfg == nil {
		cfg = &Config{}
	}

	// 应用默认值
	cfg.ApplyDefaults()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, &ServerError{
			Op:      "new",
			Message: ErrMsgInvalidConfig,
			Err:     err,
		}
	}

	s := &httpServer{
		handler:  handler,
		config:   cfg,
		logger:   log,
		errChan:  make(chan error, 1),
		doneChan: make(chan struct{}),
	}

	// 设置初始状态为已停止
	s.state.Store(int32(stateStopped))

	return s, nil
}

// Start 启动 HTTP 服务器（非阻塞）
func (s *httpServer) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return &ServerError{
			Op:      "start",
			Message: ErrMsgServerStartFailed,
			Err:     err,
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查服务器是否已在运行
	currentState := serverState(s.state.Load())
	if currentState == stateRunning || currentState == stateStarting {
		return &ServerError{
			Op:      "start",
			Message: ErrMsgServerAlreadyRunning,
		}
	}

	// 设置状态为启动中
	s.state.Store(int32(stateStarting))

	// 如果端口为 0，自动分配可用端口
	if s.config.Port == 0 {
		port, err := utils.GetAvailablePort(9000, 30000)
		if err != nil {
			s.state.Store(int32(stateStopped))
			return &ServerError{
				Op:      "start",
				Message: ErrMsgPortUnavailable,
				Err:     err,
			}
		}
		s.config.Port = port
	}

	// 构造监听地址
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// 校验地址是否合法
	if err := utils.IsValidHTTPListenAddr(addr); err != nil {
		// 如果地址不合法，使用默认 host
		s.logger.Warn("invalid listen address, using default host", "addr", addr, "error", err)
		addr = fmt.Sprintf("%s:%d", DefaultHost, s.config.Port)
	}

	// 创建 HTTP 服务器实例
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.state.Store(int32(stateStopped))
		return &ServerError{
			Op:      "start",
			Message: ErrMsgServerStartFailed,
			Err:     err,
		}
	}

	s.resetRuntimeSignalsLocked()

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.handler,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	// 记录启动信息
	s.logger.Info(fmt.Sprintf("starting HTTP server on http://%s", addr), "addr", addr)
	s.state.Store(int32(stateRunning))

	// 在新的 goroutine 中启动服务器
	go s.serve(listener, s.server, "start", ErrMsgServerStartFailed, "HTTP server error")

	return nil
}

// Wait 等待 HTTP 服务器运行期结果。
func (s *httpServer) Wait(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.RLock()
	lastRuntimeErr := s.lastRuntimeErr
	errChan := s.errChan
	doneChan := s.doneChan
	s.mu.RUnlock()

	if lastRuntimeErr != nil {
		return lastRuntimeErr
	}

	select {
	case err := <-errChan:
		return err
	case <-doneChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Shutdown 优雅关闭服务器
func (s *httpServer) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查服务器是否在运行
	currentState := serverState(s.state.Load())
	if currentState != stateRunning {
		s.logger.Warn("attempting to shutdown a non-running server", "state", currentState.String())
		if s.lastRuntimeErr == nil {
			s.signalDoneLocked()
		}
		return nil
	}

	// 设置状态为停止中
	s.state.Store(int32(stateStopping))

	s.logger.Info("shutting down HTTP server...")

	// 执行优雅关闭
	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			s.state.Store(int32(stateStopped))
			return &ServerError{
				Op:      "shutdown",
				Message: ErrMsgServerShutdownFailed,
				Err:     err,
			}
		}
	}

	// 设置状态为已停止
	s.state.Store(int32(stateStopped))
	s.signalDoneLocked()
	s.logger.Info("HTTP server stopped gracefully")

	return nil
}

// Reload 使用新配置替换 HTTP 服务器运行参数。
//
// 当服务尚未运行时，该方法只更新内存配置；当服务已运行且监听地址变化时，
// 当前实现会先优雅关闭旧 server，再创建新监听并启动 Serve goroutine。
// 因此它保证的是失败时返回明确错误和状态可见，不声明严格的零停机切换。
func (s *httpServer) Reload(ctx context.Context, cfg *Config) error {
	if cfg == nil {
		return &ServerError{
			Op:      "reload",
			Message: ErrMsgInvalidConfig,
		}
	}

	// 应用默认值
	cfg.ApplyDefaults()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return &ServerError{
			Op:      "reload",
			Message: ErrMsgInvalidConfig,
			Err:     err,
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	currentState := serverState(s.state.Load())

	// 如果服务器未运行，直接更新配置
	if currentState != stateRunning {
		s.config = cfg
		s.logger.Info("HTTP server config updated (server not running)")
		return nil
	}

	s.logger.Info("reloading HTTP server with new config...")

	// 保存旧服务器实例
	oldServer := s.server

	// 构造新的监听地址
	newAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	// 检查端口是否变化
	oldAddr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	portChanged := newAddr != oldAddr

	if portChanged {
		s.logger.Info("address changed, starting replacement server", "old", oldAddr, "new", newAddr)
		s.config = cfg
		if err := utils.IsValidHTTPListenAddr(newAddr); err != nil {
			s.logger.Warn("invalid listen address, using default host", "addr", newAddr, "error", err)
			newAddr = fmt.Sprintf("%s:%d", DefaultHost, cfg.Port)
		}
		listener, err := net.Listen("tcp", newAddr)
		if err != nil {
			return &ServerError{
				Op:      "reload",
				Message: ErrMsgReloadFailed,
				Err:     err,
			}
		}
		s.server = &http.Server{
			Addr:         newAddr,
			Handler:      s.handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		}
		go func(server *http.Server) {
			s.logger.Info(fmt.Sprintf("restarting HTTP server on http://%s", newAddr), "addr", newAddr)
			s.serve(listener, server, "reload", ErrMsgReloadFailed, "reloaded HTTP server error")
		}(s.server)
		if oldServer != nil {
			go func(server *http.Server) {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), DefaultWriteTimeout)
				defer cancel()
				if err := server.Shutdown(shutdownCtx); err != nil {
					s.logger.Error("failed to shutdown old server during reload", "error", err)
				}
			}(oldServer)
		}
		s.logger.Info("HTTP server reloaded successfully")
		return nil
	}

	// 地址变化时必须先关闭旧监听，否则同端口重绑会被操作系统拒绝。
	// 这段路径存在短暂不可接收新连接的窗口，文档层不能将其描述为零停机。
	if portChanged {
		s.logger.Info("port changed, shutting down old server first", "old", oldAddr, "new", newAddr)

		// 创建一个临时上下文用于关闭旧服务器
		shutdownCtx, cancel := context.WithTimeout(context.Background(), DefaultWriteTimeout)
		defer cancel()

		if err := oldServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("failed to shutdown old server during reload", "error", err)
			return &ServerError{
				Op:      "reload",
				Message: ErrMsgReloadFailed,
				Err:     err,
			}
		}
	}

	// 更新配置
	s.config = cfg

	// 校验地址是否合法
	if err := utils.IsValidHTTPListenAddr(newAddr); err != nil {
		s.logger.Warn("invalid listen address, using default host", "addr", newAddr, "error", err)
		newAddr = fmt.Sprintf("%s:%d", DefaultHost, cfg.Port)
	}

	if !portChanged {
		if oldServer != nil {
			oldServer.ReadTimeout = cfg.ReadTimeout
			oldServer.WriteTimeout = cfg.WriteTimeout
			oldServer.IdleTimeout = cfg.IdleTimeout
		}
		s.logger.Info("HTTP server config updated in place")
		return nil
	}

	listener, err := net.Listen("tcp", newAddr)
	if err != nil {
		return &ServerError{
			Op:      "reload",
			Message: ErrMsgReloadFailed,
			Err:     err,
		}
	}

	// 创建新的服务器实例
	s.server = &http.Server{
		Addr:         newAddr,
		Handler:      s.handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// 启动新服务器
	go func(server *http.Server) {
		s.logger.Info(fmt.Sprintf("restarting HTTP server on http://%s", newAddr), "addr", newAddr)
		s.serve(listener, server, "reload", ErrMsgReloadFailed, "reloaded HTTP server error")
	}(s.server)

	// 如果端口未变化，现在关闭旧服务器
	s.logger.Info("HTTP server reloaded successfully")
	return nil
}

// serve 在后台运行 net/http Server，并把非正常退出转换为可等待的运行时错误。
func (s *httpServer) serve(listener net.Listener, server *http.Server, op, message, logMessage string) {
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		if s.logger != nil {
			s.logger.Error(logMessage, "error", err)
		}
		s.reportRuntimeError(&ServerError{
			Op:      op,
			Message: message,
			Err:     err,
		})
	}
}

// resetRuntimeSignalsLocked 重置 Start/Reload 后 Wait 需要监听的运行时信号。
//
// 调用方必须持有写锁；重置会清空上一轮 Serve 错误，避免新监听周期被旧错误污染。
func (s *httpServer) resetRuntimeSignalsLocked() {
	s.errChan = make(chan error, 1)
	s.doneChan = make(chan struct{})
	s.doneOnce = sync.Once{}
	s.lastRuntimeErr = nil
}

// reportRuntimeError 记录后台 Serve 错误并尽力通知 Wait。
//
// errChan 带缓冲且非阻塞发送，避免没有等待者时后台 goroutine 卡死。
func (s *httpServer) reportRuntimeError(err error) {
	s.mu.Lock()
	s.lastRuntimeErr = err
	s.state.Store(int32(stateStopped))
	errChan := s.errChan
	s.mu.Unlock()

	select {
	case errChan <- err:
	default:
	}
}

// signalDoneLocked 关闭 doneChan 以唤醒 Wait。
//
// doneOnce 防止 Shutdown 和异常路径重复关闭 channel；调用方必须持有锁。
func (s *httpServer) signalDoneLocked() {
	if s.doneChan == nil {
		return
	}
	s.doneOnce.Do(func() {
		close(s.doneChan)
	})
}

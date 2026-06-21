package rpcserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/rei0721/go-scaffold/pkg/logger"
)

// serverState 仅记录监听生命周期，具体 HTTP server 指针仍由互斥锁保护。
type serverState int32

const (
	stateStopped serverState = iota
	stateRunning
)

// server 管理 RPC HTTP server 的生命周期；handler 在初始化后保持稳定，配置可通过 Reload 替换。
type server struct {
	handler http.Handler
	config  *Config
	logger  logger.Logger

	mu      sync.Mutex
	httpSrv *http.Server
	state   atomic.Int32
}

// New 创建 RPC Server。
func New(registry *Registry, cfg *Config, log logger.Logger) (Server, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry cannot be nil")
	}
	if cfg == nil {
		cfg = &Config{}
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	s := &server{
		handler: NewHandler(registry),
		config:  cfg,
		logger:  log,
	}
	s.state.Store(int32(stateStopped))
	return s, nil
}

func (s *server) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("start rpc server: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Disabled 配置被视为正常状态，方便复用同一生命周期入口管理可选 RPC 服务。
	if !s.config.Enabled {
		if s.logger != nil {
			s.logger.Info("RPC server disabled")
		}
		return nil
	}
	if serverState(s.state.Load()) == stateRunning {
		return fmt.Errorf("rpc server is already running")
	}
	return s.startLocked()
}

// Shutdown 幂等停止 RPC server；调用方传入的 context 控制优雅关闭等待时间。
func (s *server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shutdownLocked(ctx)
}

// Reload 校验并替换运行时配置；只有监听地址或超时配置变化时才重启底层 HTTP server。
func (s *server) Reload(ctx context.Context, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("rpc config cannot be nil")
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	running := serverState(s.state.Load()) == stateRunning
	if !cfg.Enabled {
		// 关闭配置需要先停止现有监听，再保存新配置，避免后续 Start 使用旧状态。
		if running {
			if err := s.shutdownLocked(ctx); err != nil {
				return err
			}
		}
		s.config = cfg
		if s.logger != nil {
			s.logger.Info("RPC server disabled")
		}
		return nil
	}

	if running {
		if sameServerConfig(s.config, cfg) {
			s.config = cfg
			if s.logger != nil {
				s.logger.Info("RPC server config unchanged")
			}
			return nil
		}
		if err := s.shutdownLocked(ctx); err != nil {
			return err
		}
	}

	s.config = cfg
	return s.startLocked()
}

// startLocked 在持锁状态下创建监听器并启动 Serve 协程；调用前必须确认服务未运行且配置已校验。
func (s *server) startLocked() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.state.Store(int32(stateStopped))
		return fmt.Errorf("start rpc server: %w", err)
	}

	s.httpSrv = &http.Server{
		Addr:         addr,
		Handler:      s.handler,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}
	s.state.Store(int32(stateRunning))
	if s.logger != nil {
		s.logger.Info(fmt.Sprintf("starting RPC server on http://%s", addr), "addr", addr)
	}

	httpSrv := s.httpSrv
	go func() {
		// Serve 在 Shutdown 时会返回 http.ErrServerClosed，这是预期路径，不应记录为错误。
		if err := httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
			if s.logger != nil {
				s.logger.Error("RPC server error", "error", err)
			}
			s.state.Store(int32(stateStopped))
		}
	}()
	return nil
}

// shutdownLocked 在持锁状态下执行优雅关闭；未运行时直接返回以支持重复调用。
func (s *server) shutdownLocked(ctx context.Context) error {
	if serverState(s.state.Load()) != stateRunning || s.httpSrv == nil {
		return nil
	}
	if s.logger != nil {
		s.logger.Info("shutting down RPC server...")
	}
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown rpc server: %w", err)
	}
	s.httpSrv = nil
	s.state.Store(int32(stateStopped))
	if s.logger != nil {
		s.logger.Info("RPC server stopped gracefully")
	}
	return nil
}

// sameServerConfig 只比较会影响 HTTP server 运行形态的配置字段。
func sameServerConfig(a, b *Config) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Enabled == b.Enabled &&
		a.Host == b.Host &&
		a.Port == b.Port &&
		a.ReadTimeout == b.ReadTimeout &&
		a.WriteTimeout == b.WriteTimeout &&
		a.IdleTimeout == b.IdleTimeout
}

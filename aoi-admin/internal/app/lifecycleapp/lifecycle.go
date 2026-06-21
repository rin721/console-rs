// Package lifecycleapp 管理应用传输层启动和资源关闭顺序。
package lifecycleapp

// 本文件定义应用运行期生命周期编排，统一处理 HTTP 服务启动、资源关闭和日志同步。

import (
	"context"
	"fmt"
	"time"

	"github.com/rei0721/go-scaffold/internal/app/initapp"
)

// Start 启动 HTTP server。
//
// 传输层必须已完成初始化；nil HTTPServer 表示装配阶段失败或被错误跳过，应立即返回错误。
func Start(ctx context.Context, transport initapp.Transport) error {
	if transport.HTTPServer == nil {
		return fmt.Errorf("http server is not initialized")
	}
	if err := startBackground(ctx, transport.Background); err != nil {
		return err
	}
	if err := transport.HTTPServer.Start(ctx); err != nil {
		shutdownBackground(context.Background(), transport.Background)
		return fmt.Errorf("start HTTP server: %w", err)
	}
	if transport.RPCServer != nil {
		if err := transport.RPCServer.Start(ctx); err != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = transport.HTTPServer.Shutdown(shutdownCtx)
			shutdownBackground(shutdownCtx, transport.Background)
			return fmt.Errorf("start RPC server: %w", err)
		}
	}
	return nil
}

// Run 启动传输层并等待 HTTP server 的运行期结果。
//
// Start 保持非阻塞启动语义；命令行主进程使用 Run，让异步 Serve 错误能够回到 cmd/aoi。
func Run(ctx context.Context, transport initapp.Transport) error {
	if err := Start(ctx, transport); err != nil {
		return err
	}
	if err := transport.HTTPServer.Wait(ctx); err != nil {
		return fmt.Errorf("wait HTTP server: %w", err)
	}
	return nil
}

// Shutdown 按固定顺序释放应用资源。
//
// 关闭过程采用最佳努力策略：某个资源关闭失败不会阻断后续资源释放，最终返回错误数量汇总。
func Shutdown(ctx context.Context, core initapp.Core, infra initapp.Infrastructure, transport initapp.Transport) error {
	log := core.Logger
	if log != nil {
		log.Info("shutting down application...")
	}

	var errs []error

	if transport.HTTPServer != nil {
		if err := transport.HTTPServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("http server shutdown: %w", err))
			if log != nil {
				log.Error("failed to shutdown HTTP server", "error", err)
			}
		} else if log != nil {
			log.Info("HTTP server stopped")
		}
	}

	if transport.RPCServer != nil {
		if err := transport.RPCServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("rpc server shutdown: %w", err))
			if log != nil {
				log.Error("failed to shutdown RPC server", "error", err)
			}
		} else if log != nil {
			log.Info("RPC server stopped")
		}
	}

	for i := len(transport.Background) - 1; i >= 0; i-- {
		if transport.Background[i] == nil {
			continue
		}
		if err := transport.Background[i].Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("background shutdown: %w", err))
			if log != nil {
				log.Error("failed to shutdown background service", "error", err)
			}
		}
	}

	if infra.Storage != nil {
		if err := infra.Storage.Close(); err != nil {
			errs = append(errs, fmt.Errorf("storage close: %w", err))
			if log != nil {
				log.Error("failed to close storage", "error", err)
			}
		} else if log != nil {
			log.Info("storage closed")
		}
	}

	if infra.Executor != nil {
		infra.Executor.Shutdown()
		if log != nil {
			log.Info("executor stopped")
		}
	}

	if infra.Cache != nil {
		if err := infra.Cache.Close(); err != nil {
			errs = append(errs, fmt.Errorf("cache close: %w", err))
			if log != nil {
				log.Error("failed to close cache", "error", err)
			}
		} else if log != nil {
			log.Info("cache closed")
		}
	}

	if infra.Database != nil {
		if err := infra.Database.Close(); err != nil {
			errs = append(errs, fmt.Errorf("database close: %w", err))
			if log != nil {
				log.Error("failed to close database", "error", err)
			}
		} else if log != nil {
			log.Info("database connection closed")
		}
	}

	if log != nil {
		_ = log.Sync()
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown completed with %d errors", len(errs))
	}
	if log != nil {
		log.Info("application shutdown complete")
	}
	return nil
}

// startBackground 按声明顺序启动后台任务。
//
// 任一任务启动失败时，会反向关闭已经启动成功的任务，避免半启动状态遗留 goroutine 或外部租约。
func startBackground(ctx context.Context, services []initapp.BackgroundService) error {
	for i, service := range services {
		if service == nil {
			continue
		}
		if err := service.Start(ctx); err != nil {
			shutdownBackground(context.Background(), services[:i])
			return fmt.Errorf("start background service: %w", err)
		}
	}
	return nil
}

// shutdownBackground 以 best-effort 方式反向关闭后台任务。
//
// 该 helper 只用于启动失败回滚，错误会被忽略；正式关闭路径由 Shutdown 收集并记录错误。
func shutdownBackground(ctx context.Context, services []initapp.BackgroundService) {
	for i := len(services) - 1; i >= 0; i-- {
		if services[i] != nil {
			_ = services[i].Shutdown(ctx)
		}
	}
}

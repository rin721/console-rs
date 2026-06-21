package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rei0721/go-scaffold/internal/app"
	cliappadapters "github.com/rei0721/go-scaffold/internal/app/cliapp/adapters"
	"github.com/rei0721/go-scaffold/internal/app/cliapp/services/managed"
	"github.com/rei0721/go-scaffold/types/constants"
)

// Run 装配应用、启动 HTTP 服务并等待系统信号、托管控制请求或启动错误。
func Run(configPath string) error {
	application, err := app.New(app.Options{
		ConfigPath: configPath,
	})
	if err != nil {
		managed.MarkManagedServiceStopped(managed.ServiceServer, err.Error())
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	manager := managed.NewManager()
	controlCtx, stopControlWatcher := context.WithCancel(context.Background())
	defer stopControlWatcher()
	control := cliappadapters.WatchManagedServiceControl(controlCtx, managed.ServiceServer, manager.ControlPath(managed.ServiceServer))

	errChan := make(chan error, 1)
	go func() {
		if err := application.Run(); err != nil {
			errChan <- err
		}
	}()

	var finalError string
	select {
	case sig := <-quit:
		application.Core.Logger.Info("received shutdown signal", "signal", sig.String())
	case req, ok := <-control:
		if ok {
			application.Core.Logger.Info("received CLI service control request", "action", req.Action, "pid", req.PID)
		}
	case err := <-errChan:
		application.Core.Logger.Error("server error", "error", err)
		finalError = err.Error()
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.AppShutdownTimeout)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		application.Core.Logger.Error("shutdown error", "error", err)
		managed.MarkManagedServiceStopped(managed.ServiceServer, err.Error())
		return fmt.Errorf("shutdown error: %w", err)
	}

	managed.MarkManagedServiceStopped(managed.ServiceServer, finalError)
	application.Core.Logger.Info("application exited gracefully")
	return nil
}

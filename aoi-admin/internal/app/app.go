// Package app 是服务的组合根。
//
// 本包遵循洋葱式依赖方向：
//
//	Core -> Infrastructure -> Modules -> Transport
//
// 内层不感知外层；只有本包负责把配置、基础设施、业务模块和传输层装配成可运行应用。
package app

// 本文件属于应用组合根，描述 Core、Infrastructure、Modules 与 Transport 在进程内的持有关系。

import (
	"context"
	"fmt"

	"github.com/rei0721/go-scaffold/internal/app/initapp"
	"github.com/rei0721/go-scaffold/internal/app/lifecycleapp"
	"github.com/rei0721/go-scaffold/internal/app/mainapp"
	"github.com/rei0721/go-scaffold/internal/app/reloadapp"
	"github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/logger"
)

// App 是服务启动后的运行态装配结果。
//
// 它只保存各层依赖的引用，不承载具体业务逻辑；业务能力应通过 Modules 或 Transport 暴露。
type App struct {
	Options Options

	Core      CoreLayer
	Infra     InfrastructureLayer
	Modules   ModulesLayer
	Transport TransportLayer
}

// Options 描述应用构建入口参数。
//
// ConfigPath 交给配置管理器解析；Mode 为空时会在 New 中回落到 ModeServer，
// 以保持命令行入口不需要理解内部运行模式枚举。
type Options struct {
	ConfigPath string
	Mode       AppMode
}

// New 创建完整应用实例，并按 Core、Infrastructure、Modules、Transport 的顺序完成装配。
//
// 这里是依赖注入边界：外部只传入启动选项，内部负责创建核心服务、注册配置变更回调，
// 再交给 mainapp 构建具体运行模式。
func New(opts Options) (*App, error) {
	if opts.Mode == "" {
		opts.Mode = ModeServer
	}

	app := &App{Options: opts}
	core, err := initapp.NewCore(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("initialize core: %w", err)
	}
	app.Core = core

	app.Core.ConfigManager.RegisterLogger(func() logger.Logger {
		return app.Core.Logger
	})

	initapp.DebugConfig(app.Core, opts.ConfigPath)

	result, err := mainapp.Build(opts.Mode, app.Core, opts.ConfigPath, app.handleConfigChange)
	if err != nil {
		return nil, err
	}
	app.Infra = result.Infra
	app.Modules = result.Modules
	app.Transport = result.Transport

	return app, nil
}

// Run 使用后台上下文启动应用并等待 HTTP 运行期结果，适合命令行主进程的常规入口。
func (a *App) Run() error {
	return lifecycleapp.Run(context.Background(), a.Transport)
}

// Start 启动传输层服务；资源启动顺序由 lifecycleapp 统一维护。
func (a *App) Start(ctx context.Context) error {
	return lifecycleapp.Start(ctx, a.Transport)
}

// Shutdown 按生命周期约定释放传输层和基础设施资源。
func (a *App) Shutdown(ctx context.Context) error {
	return lifecycleapp.Shutdown(ctx, a.Core, a.Infra, a.Transport)
}

// handleConfigChange 接收配置管理器发布的新旧快照，并把组件级热加载委托给 reloadapp 执行。
func (a *App) handleConfigChange(old, new *config.Config) {
	a.Core.Logger.Info("configuration file changed, processing updates")
	reloadapp.Reload(&a.Core, &a.Infra, &a.Transport, old, new, &a.Modules)
	// reloadapp 会原地更新可重载组件，这里同步 Core.Config 指针，避免后续读取旧配置快照。
	a.Core.Config = new
	a.Core.Logger.Info("configuration update completed")
}

// Package mainapp 负责把已初始化的核心服务装配成主应用运行模式。
//
// 当前脚手架只有 server 真实模式；内部包名调整不改变 CLI 命令、配置字段或运行模式字符串。
package mainapp

// 本文件定义主运行模式的组装入口，是当前 server 模式从分层构建到可运行应用的收敛点。

import (
	"github.com/rei0721/go-scaffold/internal/app/initapp"
	"github.com/rei0721/go-scaffold/internal/app/initcenter"
	"github.com/rei0721/go-scaffold/internal/config"
)

// Mode 表示内部应用运行模式。
//
// 它是应用装配选择，不等同于 Gin 的 server.mode，也不应直接扩展为用户权限或业务状态。
type Mode string

const (
	// ModeServer 是当前唯一稳定模式；其字符串值必须保持为 "server" 以兼容 CLI 和文档。
	ModeServer Mode = "server"
)

// ConfigChangeHandler 是配置管理器变更回调的别名。
type ConfigChangeHandler = config.HookHandler

// BuildResult 汇总主应用模式构建后的三层外部依赖。
//
// Core 已由 app.New 创建，因此结果只返回依赖 Core 的基础设施、模块和传输层。
type BuildResult struct {
	Infra     initapp.Infrastructure
	Modules   initapp.Modules
	Transport initapp.Transport
}

// Build 根据 mode 构建应用层。
//
// 当前只有 server 模式；未知值会回落到 BuildServer，避免内部模式重命名或空值误传破坏启动。
func Build(mode Mode, core initapp.Core, configPath string, onConfigChange ConfigChangeHandler) (BuildResult, error) {
	switch mode {
	case ModeServer:
		return BuildServer(core, configPath, onConfigChange)
	default:
		return BuildServer(core, configPath, onConfigChange)
	}
}

// BuildServer 按基础设施、模块、传输层的顺序构建 server 模式。
//
// 配置监听最后启动，确保 reload hook 只能观察到已经装配完成的运行态组件。
func BuildServer(core initapp.Core, configPath string, onConfigChange ConfigChangeHandler) (BuildResult, error) {
	infra, err := initapp.NewInfrastructure(core)
	if err != nil {
		return BuildResult{}, err
	}

	modules, err := initapp.NewModules(core, infra)
	if err != nil {
		return BuildResult{}, err
	}

	setupCenter := initcenter.New(core, infra, modules, configPath, nil)
	if modules.IAM.Handler != nil {
		modules.IAM.Handler.UseSetupService(setupCenter.IAMSetupService())
	}
	setupHandler := initcenter.NewHandler(setupCenter, core.Logger, initcenter.HandlerConfigFromAppConfig(core.Config))
	transport, err := initapp.NewTransport(core, infra, modules, setupHandler)
	if err != nil {
		return BuildResult{}, err
	}

	WatchConfig(core, onConfigChange)
	core.Logger.Info("application initialized successfully")

	return BuildResult{
		Infra:     infra,
		Modules:   modules,
		Transport: transport,
	}, nil
}

// WatchConfig 启动配置文件监听，并在监听成功后注册变更回调。
//
// watcher 启动失败只记录告警，不阻断服务启动；脚手架允许在无热更新能力时继续运行。
func WatchConfig(core initapp.Core, onConfigChange ConfigChangeHandler) {
	if err := core.ConfigManager.Watch(); err != nil {
		core.Logger.Warn("failed to start config watcher", "error", err)
		return
	}
	core.Logger.Debug("config watcher started")

	if onConfigChange != nil {
		core.ConfigManager.RegisterHook(onConfigChange)
	}
}

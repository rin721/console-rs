// Package reloadapp 将配置变化应用到已装配的可重载组件。
//
// reload 是运行态最佳努力流程：单个组件失败只记录日志，不回滚其他已处理组件。
package reloadapp

// 本文件实现配置热加载时的组件级替换策略，强调失败保留旧实例和可选能力降级。

import (
	"context"
	"time"

	"github.com/rei0721/go-scaffold/internal/app/initapp"
	"github.com/rei0721/go-scaffold/internal/config"
	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
	"github.com/rei0721/go-scaffold/pkg/executor"
	"github.com/rei0721/go-scaffold/pkg/storage"
)

// Reload 比较新旧配置并原地更新发生变化的组件。
//
// 调用方传入指针是有意设计：缓存、数据库、执行器、HTTP server 和存储可能被替换或置空。
func Reload(core *initapp.Core, infra *initapp.Infrastructure, transport *initapp.Transport, old, new *config.Config, modules ...*initapp.Modules) {
	if initapp.IsCacheConfigChanged(old, new) {
		reloadCache(core, infra, new)
	}
	if initapp.IsDatabaseConfigChanged(old, new) {
		reloadDatabase(core, infra, new)
	}
	if initapp.IsLoggerConfigChanged(old, new) {
		reloadLogger(core, new)
	}
	if initapp.IsExecutorConfigChanged(old, new) {
		reloadExecutor(core, infra, new)
	}
	if initapp.IsServerConfigChanged(old, new) {
		reloadHTTPServer(core, transport, new)
	}
	if initapp.IsRPCConfigChanged(old, new) {
		reloadRPCServer(core, transport, new)
	}
	if initapp.IsStorageConfigChanged(old, new) {
		reloadStorage(core, infra, new)
	}
	if initapp.IsIAMRuntimeConfigChanged(old, new) && len(modules) > 0 {
		reloadIAMRuntime(core, modules[0], new)
	}
}

// reloadIAMRuntime 在邮件或注册策略配置变化后替换 IAM 运行时依赖。
func reloadIAMRuntime(core *initapp.Core, modules *initapp.Modules, cfg *config.Config) {
	if modules == nil || modules.IAM.Notifier == nil || modules.IAM.Service == nil {
		if core.Logger != nil {
			core.Logger.Warn("iam runtime reload skipped: iam module is not available")
		}
		return
	}

	authCfg := cfg.Auth
	authCfg.ApplyDefaults()
	nextCore := *core
	nextCore.Config = cfg
	notifier, err := initapp.NewIAMNotifier(nextCore, authCfg)
	if err != nil {
		if core.Logger != nil {
			core.Logger.Error("failed to reload iam notification provider", "error", err)
		}
		return
	}
	modules.IAM.Notifier.Replace(notifier)
	if reloader, ok := modules.IAM.Service.(iamservice.NotificationRuntimeReloader); ok {
		reloader.ReloadNotificationRuntime(initapp.IAMNotificationRuntimeConfig(cfg))
	}
	if reloader, ok := modules.IAM.Service.(iamservice.RegistrationRuntimeReloader); ok {
		reloader.ReloadRegistrationRuntime(initapp.IAMRegistrationRuntimeConfig(cfg))
	}
	if core.Logger != nil {
		core.Logger.Info("iam runtime reloaded", "driver", authCfg.NotificationDriver, "registrationMode", authCfg.RegistrationMode)
	}
}

// reloadCache 对比缓存配置并按模式决定创建、关闭或替换缓存实例，失败时记录错误后继续处理其他组件。
func reloadCache(core *initapp.Core, infra *initapp.Infrastructure, cfg *config.Config) {
	if cfg.Cache.Driver == config.CacheDriverDisabled {
		if infra.Cache != nil {
			_ = infra.Cache.Close()
			infra.Cache = nil
		}
		core.Logger.Info("cache disabled")
		return
	}

	if infra.Cache != nil && (cfg.Cache.Driver == config.CacheDriverRedis || cfg.Cache.Driver == config.CacheDriverHybrid) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := infra.Cache.Reload(ctx, initapp.RedisConfig(cfg.Cache.Redis)); err != nil {
			core.Logger.Error("failed to reload cache", "error", err)
			return
		}
		core.Logger.Info("cache reloaded", "driver", cfg.Cache.Driver)
		return
	}

	cacheClient, err := initapp.NewCache(cfg, core.Logger)
	if err != nil {
		core.Logger.Error("failed to initialize cache", "error", err)
		return
	}
	old := infra.Cache
	infra.Cache = cacheClient
	if old != nil {
		_ = old.Close()
	}
	core.Logger.Info("cache reloaded", "driver", cfg.Cache.Driver)
}

// reloadDatabase 在数据库配置变化时重载硬依赖连接，避免热更新流程修改业务表结构。
func reloadDatabase(core *initapp.Core, infra *initapp.Infrastructure, cfg *config.Config) {
	if infra.Database == nil {
		db, err := initapp.NewDatabase(cfg)
		if err != nil {
			core.Logger.Error("failed to initialize database", "error", err)
			return
		}
		infra.Database = db
		core.Logger.Info("database initialized")
		return
	}

	if err := infra.Database.Reload(initapp.DatabaseConfig(cfg)); err != nil {
		core.Logger.Error("failed to reload database", "error", err)
		return
	}
	core.Logger.Info("database reloaded")
}

// reloadLogger 在日志配置变化时替换 zap 配置，使后续日志输出使用新级别和目的地。
func reloadLogger(core *initapp.Core, cfg *config.Config) {
	if core.Logger == nil {
		return
	}
	if err := core.Logger.Reload(initapp.LoggerConfig(cfg)); err != nil {
		core.Logger.Error("failed to reload logger", "error", err)
		return
	}
	core.Logger.Info("logger reloaded")
}

// reloadExecutor 根据 executor 配置的启用状态创建、关闭或重载协程池管理器。
func reloadExecutor(core *initapp.Core, infra *initapp.Infrastructure, cfg *config.Config) {
	if !cfg.Executor.Enabled {
		if infra.Executor != nil {
			infra.Executor.Shutdown()
			infra.Executor = nil
		}
		core.Logger.Info("executor disabled")
		return
	}

	executorConfigs := initapp.ExecutorConfigs(cfg)
	if infra.Executor == nil {
		mgr, err := executor.NewManager(executorConfigs)
		if err != nil {
			core.Logger.Error("failed to initialize executor", "error", err)
			return
		}
		infra.Executor = mgr
		core.Logger.Info("executor initialized", "pools", len(executorConfigs))
		return
	}

	if err := infra.Executor.Reload(executorConfigs); err != nil {
		core.Logger.Error("failed to reload executor", "error", err)
		return
	}
	core.Logger.Info("executor reloaded", "pools", len(executorConfigs))
}

// reloadHTTPServer 在监听配置变化时把新配置交给 HTTPServer，具体停启语义由 httpserver 包负责。
func reloadHTTPServer(core *initapp.Core, transport *initapp.Transport, cfg *config.Config) {
	if transport.HTTPServer == nil {
		core.Logger.Warn("http server is nil, cannot reload configuration")
		return
	}

	// HTTP reload 可能等待监听器或连接状态切换，因此给出比缓存和存储更宽的窗口。
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := transport.HTTPServer.Reload(ctx, initapp.HTTPServerConfig(cfg)); err != nil {
		core.Logger.Error("failed to reload HTTP server", "error", err)
		return
	}
	core.Logger.Info("HTTP server reloaded")
}

// reloadRPCServer 在独立 RPC 监听配置变化时执行启停或重载。
func reloadRPCServer(core *initapp.Core, transport *initapp.Transport, cfg *config.Config) {
	if transport.RPCServer == nil {
		core.Logger.Warn("rpc server is nil, cannot reload configuration")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := transport.RPCServer.Reload(ctx, initapp.RPCServerConfig(cfg)); err != nil {
		core.Logger.Error("failed to reload RPC server", "error", err)
		return
	}
	core.Logger.Info("RPC server reloaded")
}

// reloadStorage 根据存储配置变化创建、关闭或重载存储实例，并用有限超时保护热更新链路。
func reloadStorage(core *initapp.Core, infra *initapp.Infrastructure, cfg *config.Config) {
	storageCfg := initapp.StorageConfig(cfg)
	if cfg.Storage.Driver == config.StorageDriverDisabled || cfg.Storage.Driver == config.StorageDriverS3 || cfg.Storage.Driver == config.StorageDriverMinIO {
		if infra.Storage != nil {
			_ = infra.Storage.Close()
			infra.Storage = nil
		}
		core.Logger.Info("storage disabled")
		return
	}

	if infra.Storage == nil {
		storageService, err := storage.New(storageCfg)
		if err != nil {
			core.Logger.Error("failed to initialize storage", "error", err)
			return
		}
		infra.Storage = storageService
		core.Logger.Info("storage initialized")
		return
	}

	// 存储 reload 涉及 watcher 或文件系统句柄，使用有限超时避免热更新卡死。
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := infra.Storage.Reload(ctx, storageCfg); err != nil {
		core.Logger.Error("failed to reload storage", "error", err)
		return
	}
	core.Logger.Info("storage reloaded")
}

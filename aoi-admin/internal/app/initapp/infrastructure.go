package initapp

// 本文件属于应用初始化装配层，负责把配置、基础设施、业务模块或传输层拼接为可运行的分层对象。

import (
	"fmt"

	"github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cache"
	"github.com/rei0721/go-scaffold/pkg/database"
	"github.com/rei0721/go-scaffold/pkg/executor"
	"github.com/rei0721/go-scaffold/pkg/logger"
	"github.com/rei0721/go-scaffold/pkg/storage"
)

// NewInfrastructure 创建应用基础设施层。
//
// 数据库连接失败会阻断启动；缓存、执行器和存储会根据各自配置决定是否返回 nil。
func NewInfrastructure(core Core) (Infrastructure, error) {
	db, err := NewDatabase(core.Config)
	if err != nil {
		return Infrastructure{}, err
	}

	cacheClient, err := NewCache(core.Config, core.Logger)
	if err != nil {
		return Infrastructure{}, err
	}

	executorManager, err := NewExecutor(core.Config, core.Logger)
	if err != nil {
		return Infrastructure{}, err
	}

	storageService, err := NewStorage(core.Config, core.Logger)
	if err != nil {
		return Infrastructure{}, err
	}

	return Infrastructure{
		Database: db,
		Cache:    cacheClient,
		Executor: executorManager,
		Storage:  storageService,
	}, nil
}

// NewDatabase 创建数据库抽象。
//
// 数据库是运行时健康检查和持久化能力的硬依赖，因此连接失败会作为启动错误返回。
func NewDatabase(cfg *config.Config) (database.Database, error) {
	db, err := database.New(DatabaseConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}

// NewCache 创建运行时缓存客户端。
//
// 缓存是可选增强能力：配置关闭或连接失败时返回 nil 或降级实例，让服务保持可用。
func NewCache(cfg *config.Config, log logger.Logger) (cache.Cache, error) {
	driver := cfg.Cache.Driver
	if driver == config.CacheDriverDisabled {
		log.Info("cache disabled")
		return nil, nil
	}

	cacheClient, err := cache.New(CacheRuntimeConfig(cfg), log)
	if err != nil {
		log.Warn("failed to initialize cache, running without cache", "driver", driver, "error", err)
		return nil, nil
	}
	if cacheClient == nil {
		log.Info("cache disabled")
		return nil, nil
	}

	log.Info("cache initialized", "driver", driver)
	return cacheClient, nil
}

// NewExecutor 创建后台执行器管理器。
//
// 执行器是可选组件；禁用时返回 nil，但配置错误会阻断启动以避免任务调度处于半初始化状态。
func NewExecutor(cfg *config.Config, log logger.Logger) (executor.Manager, error) {
	if !cfg.Executor.Enabled {
		log.Info("executor disabled")
		return nil, nil
	}

	executorConfigs := ExecutorConfigs(cfg)
	mgr, err := executor.NewManager(executorConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor manager: %w", err)
	}

	log.Info("executor initialized", "pools", len(executorConfigs))
	return mgr, nil
}

// NewStorage 创建文件存储服务。
//
// 存储配置会先补默认值并应用环境覆盖；只有校验通过后才根据 enabled 决定是否启动。
func NewStorage(cfg *config.Config, log logger.Logger) (storage.Storage, error) {
	storageCfg := cfg.Storage
	if err := storageCfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid storage config: %w", err)
	}

	driver := storageCfg.Driver
	if driver == config.StorageDriverDisabled || driver == config.StorageDriverS3 || driver == config.StorageDriverMinIO {
		log.Info("local file storage disabled", "driver", driver)
		return nil, nil
	}

	storageService, err := storage.New(storageCfg.ToPkgConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	log.Info(
		"storage initialized",
		"driver", driver,
		"fs_type", storageCfg.Local.FSType,
		"base_path", storageCfg.Local.BasePath,
		"enable_watch", storageCfg.Local.EnableWatch,
	)
	return storageService, nil
}

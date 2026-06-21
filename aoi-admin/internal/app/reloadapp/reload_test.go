package reloadapp

// 本测试文件固定配置热加载的组件替换边界，防止注释补全和后续重构改变外部可观察行为。

import (
	"context"
	"testing"

	"github.com/rei0721/go-scaffold/internal/app/initapp"
	"github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cache"
	"github.com/rei0721/go-scaffold/pkg/database"
	"github.com/rei0721/go-scaffold/pkg/executor"
	"github.com/rei0721/go-scaffold/pkg/httpserver"
	"github.com/rei0721/go-scaffold/pkg/logger"
	"github.com/rei0721/go-scaffold/pkg/rpcserver"
	storagepkg "github.com/rei0721/go-scaffold/pkg/storage"
)

// TestReloadSkipsUnchangedComponents 固定配置热加载的组件替换边界，确保后续注释补全或结构调整不改变该场景。
func TestReloadSkipsUnchangedComponents(t *testing.T) {
	clearReloadEnv(t)

	oldCfg := baseReloadConfig()
	newCfg := cloneReloadConfig(oldCfg)
	core, infra, transport, fakes := newReloadFixture(t)

	Reload(&core, &infra, &transport, oldCfg, newCfg)

	assertReloadCounts(t, fakes, reloadCounts{})
	if fakes.cache.closes != 0 {
		t.Fatalf("expected cache not to close, got %d", fakes.cache.closes)
	}
	if fakes.executor.shutdowns != 0 {
		t.Fatalf("expected executor not to shutdown, got %d", fakes.executor.shutdowns)
	}
	if fakes.storage.closes != 0 {
		t.Fatalf("expected storage not to close, got %d", fakes.storage.closes)
	}
	if fakes.database.dbCalls != 0 {
		t.Fatalf("expected unchanged database config not to inspect gorm DB, got %d calls", fakes.database.dbCalls)
	}
}

// TestReloadReloadsOnlyChangedComponent 固定配置热加载的组件替换边界，确保后续注释补全或结构调整不改变该场景。
func TestReloadReloadsOnlyChangedComponent(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.Config)
		want   reloadCounts
	}{
		{
			name: "cache",
			mutate: func(cfg *config.Config) {
				cfg.Cache.Redis.Addr = "redis.changed.local:6379"
			},
			want: reloadCounts{cache: 1},
		},
		{
			name: "database",
			mutate: func(cfg *config.Config) {
				cfg.Database.SQLite.Path = "file:changed-database?mode=memory&cache=shared"
			},
			want: reloadCounts{database: 1},
		},
		{
			name: "logger",
			mutate: func(cfg *config.Config) {
				cfg.Logger.Level = "debug"
			},
			want: reloadCounts{logger: 1},
		},
		{
			name: "executor",
			mutate: func(cfg *config.Config) {
				cfg.Executor.Pools[0].Size = 3
			},
			want: reloadCounts{executor: 1},
		},
		{
			name: "http-server",
			mutate: func(cfg *config.Config) {
				cfg.Server.Port = 18082
			},
			want: reloadCounts{httpServer: 1},
		},
		{
			name: "rpc-server",
			mutate: func(cfg *config.Config) {
				cfg.RPC.Enabled = true
				cfg.RPC.Port = 11099
			},
			want: reloadCounts{rpcServer: 1},
		},
		{
			name: "storage",
			mutate: func(cfg *config.Config) {
				cfg.Storage.Local.BasePath = "./changed-storage"
			},
			want: reloadCounts{storage: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearReloadEnv(t)

			oldCfg := baseReloadConfig()
			newCfg := cloneReloadConfig(oldCfg)
			tt.mutate(newCfg)
			core, infra, transport, fakes := newReloadFixture(t)

			Reload(&core, &infra, &transport, oldCfg, newCfg)

			assertReloadCounts(t, fakes, tt.want)
			if tt.name == "database" && fakes.database.dbCalls != 0 {
				t.Fatalf("expected database reload to avoid migration DB() calls, got %d calls", fakes.database.dbCalls)
			}
		})
	}
}

// TestReloadDisablesOptionalComponents 固定配置热加载的组件替换边界，确保后续注释补全或结构调整不改变该场景。
func TestReloadDisablesOptionalComponents(t *testing.T) {
	clearReloadEnv(t)

	oldCfg := baseReloadConfig()
	newCfg := cloneReloadConfig(oldCfg)
	newCfg.Cache.Driver = config.CacheDriverDisabled
	newCfg.Executor.Enabled = false
	newCfg.Storage.Driver = config.StorageDriverDisabled
	core, infra, transport, fakes := newReloadFixture(t)

	Reload(&core, &infra, &transport, oldCfg, newCfg)

	assertReloadCounts(t, fakes, reloadCounts{})
	if fakes.cache.closes != 1 {
		t.Fatalf("expected cache close once, got %d", fakes.cache.closes)
	}
	if infra.Cache != nil {
		t.Fatal("expected disabled cache to be nil")
	}
	if fakes.executor.shutdowns != 1 {
		t.Fatalf("expected executor shutdown once, got %d", fakes.executor.shutdowns)
	}
	if infra.Executor != nil {
		t.Fatal("expected disabled executor to be nil")
	}
	if fakes.storage.closes != 1 {
		t.Fatalf("expected storage close once, got %d", fakes.storage.closes)
	}
	if infra.Storage != nil {
		t.Fatal("expected disabled storage to be nil")
	}
}

func TestReloadDisablesRPCServer(t *testing.T) {
	clearReloadEnv(t)

	oldCfg := baseReloadConfig()
	oldCfg.RPC.Enabled = true
	newCfg := cloneReloadConfig(oldCfg)
	newCfg.RPC.Enabled = false
	core, infra, transport, fakes := newReloadFixture(t)

	Reload(&core, &infra, &transport, oldCfg, newCfg)

	assertReloadCounts(t, fakes, reloadCounts{rpcServer: 1})
	if fakes.rpcServer.lastConfig == nil || fakes.rpcServer.lastConfig.Enabled {
		t.Fatalf("expected RPC reload with disabled config, got %#v", fakes.rpcServer.lastConfig)
	}
}

// baseReloadConfig 是当前测试文件的辅助函数，用于复用夹具、断言或输入构造逻辑。
func baseReloadConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         18081,
			Mode:         "test",
			ReadTimeout:  1,
			WriteTimeout: 1,
			IdleTimeout:  1,
		},
		Database: config.DatabaseConfig{
			Driver: config.DatabaseDriverSQLite,
			SQLite: config.DatabaseSQLiteConfig{
				Path: "file:reload-old?mode=memory&cache=shared",
			},
			Pool: config.DatabasePoolConfig{
				MaxOpenConns: 1,
				MaxIdleConns: 1,
			},
		},
		Cache: config.CacheConfig{
			Driver: config.CacheDriverRedis,
			Local: config.LocalCacheConfig{
				MaxCost:     67108864,
				NumCounters: 1000000,
				BufferItems: 64,
			},
			Redis: config.RedisCacheConfig{
				Addr:         "127.0.0.1:6379",
				DB:           0,
				PoolSize:     1,
				MinIdleConns: 0,
				MaxRetries:   0,
				DialTimeout:  1,
				ReadTimeout:  1,
				WriteTimeout: 1,
			},
		},
		Logger: config.LoggerConfig{
			Level:  "info",
			Format: "console",
			Output: "stdout",
		},
		I18n: config.I18nConfig{
			DefaultLocale:  "zh-CN",
			FallbackLocale: "zh-CN",
			Supported:      []string{"zh-CN", "en-US"},
			Resources: map[string]string{
				"ui":         "./configs/locales/ui",
				"api":        "./configs/locales/api",
				"validation": "./configs/locales/validation",
				"system":     "./configs/locales/system",
			},
		},
		Executor: config.ExecutorConfig{
			Enabled: true,
			Pools: []config.ExecutorPoolConfig{
				{Name: "background", Size: 1, Expiry: 1, NonBlocking: true},
			},
		},
		Storage: config.StorageConfig{
			Driver: config.StorageDriverLocal,
			Local: config.StorageLocalConfig{
				FSType:          "memory",
				BasePath:        "./storage",
				EnableWatch:     false,
				WatchBufferSize: 1,
			},
		},
		CORS: config.CORSConfig{
			Enabled:      true,
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "OPTIONS"},
			AllowHeaders: []string{"Origin", "Content-Type", "X-Request-ID"},
			ExposeHeaders: []string{
				"X-Request-ID",
			},
			MaxAge: 60,
		},
		RPC: config.RPCConfig{
			Enabled:      false,
			Host:         "127.0.0.1",
			Port:         10099,
			ReadTimeout:  10,
			WriteTimeout: 10,
			IdleTimeout:  30,
		},
	}
}

// cloneReloadConfig 是当前测试文件的辅助函数，用于复用夹具、断言或输入构造逻辑。
func cloneReloadConfig(src *config.Config) *config.Config {
	dst := *src
	dst.I18n.Supported = append([]string(nil), src.I18n.Supported...)
	dst.I18n.Resources = cloneStringMap(src.I18n.Resources)
	dst.Executor.Pools = append([]config.ExecutorPoolConfig(nil), src.Executor.Pools...)
	dst.CORS.AllowOrigins = append([]string(nil), src.CORS.AllowOrigins...)
	dst.CORS.AllowMethods = append([]string(nil), src.CORS.AllowMethods...)
	dst.CORS.AllowHeaders = append([]string(nil), src.CORS.AllowHeaders...)
	dst.CORS.ExposeHeaders = append([]string(nil), src.CORS.ExposeHeaders...)
	return &dst
}

func cloneStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

type reloadFixture struct {
	cache      *fakeCache
	database   *fakeDatabase
	logger     *fakeLogger
	executor   *fakeExecutor
	httpServer *fakeHTTPServer
	rpcServer  *fakeRPCServer
	storage    *fakeStorage
}

// newReloadFixture 构造当前测试场景所需的最小依赖集合，避免测试直接耦合生产装配流程。
func newReloadFixture(t *testing.T) (initapp.Core, initapp.Infrastructure, initapp.Transport, reloadFixture) {
	t.Helper()

	fakes := reloadFixture{
		cache:      &fakeCache{},
		database:   newFakeDatabase(t),
		logger:     &fakeLogger{},
		executor:   &fakeExecutor{},
		httpServer: &fakeHTTPServer{},
		rpcServer:  &fakeRPCServer{},
		storage:    &fakeStorage{},
	}

	return initapp.Core{Logger: fakes.logger},
		initapp.Infrastructure{
			Cache:    fakes.cache,
			Database: fakes.database,
			Executor: fakes.executor,
			Storage:  fakes.storage,
		},
		initapp.Transport{HTTPServer: fakes.httpServer, RPCServer: fakes.rpcServer},
		fakes
}

type reloadCounts struct {
	cache      int
	database   int
	logger     int
	executor   int
	httpServer int
	rpcServer  int
	storage    int
}

// assertReloadCounts 校验测试响应或状态中的关键字段，使测试断言聚焦在对外契约而非重复解析细节。
func assertReloadCounts(t *testing.T, fakes reloadFixture, want reloadCounts) {
	t.Helper()

	if fakes.cache.reloads != want.cache {
		t.Fatalf("expected cache reloads %d, got %d", want.cache, fakes.cache.reloads)
	}
	if fakes.database.reloads != want.database {
		t.Fatalf("expected database reloads %d, got %d", want.database, fakes.database.reloads)
	}
	if fakes.logger.reloads != want.logger {
		t.Fatalf("expected logger reloads %d, got %d", want.logger, fakes.logger.reloads)
	}
	if fakes.executor.reloads != want.executor {
		t.Fatalf("expected executor reloads %d, got %d", want.executor, fakes.executor.reloads)
	}
	if fakes.httpServer.reloads != want.httpServer {
		t.Fatalf("expected http server reloads %d, got %d", want.httpServer, fakes.httpServer.reloads)
	}
	if fakes.rpcServer.reloads != want.rpcServer {
		t.Fatalf("expected rpc server reloads %d, got %d", want.rpcServer, fakes.rpcServer.reloads)
	}
	if fakes.storage.reloads != want.storage {
		t.Fatalf("expected storage reloads %d, got %d", want.storage, fakes.storage.reloads)
	}
}

type fakeCache struct {
	cache.Cache
	reloads    int
	closes     int
	lastConfig *cache.Config
}

// Reload 实现测试桩的配置重载入口，用于验证调用路径而不触发真实资源替换。
func (c *fakeCache) Reload(_ context.Context, cfg *cache.Config) error {
	c.reloads++
	c.lastConfig = cfg
	return nil
}

// Close 实现测试桩的资源关闭入口，用于验证生命周期调用而不释放外部资源。
func (c *fakeCache) Close() error {
	c.closes++
	return nil
}

type fakeDatabase struct {
	database.Database
	reloads    int
	dbCalls    int
	lastConfig *database.Config
}

// newFakeDatabase 构造当前测试场景所需的最小依赖集合，避免测试直接耦合生产装配流程。
func newFakeDatabase(t *testing.T) *fakeDatabase {
	t.Helper()
	return &fakeDatabase{}
}

// Reload 实现测试桩的配置重载入口，用于验证调用路径而不触发真实资源替换。
func (d *fakeDatabase) Reload(cfg *database.Config) error {
	d.reloads++
	d.lastConfig = cfg
	return nil
}

// Close 实现测试桩的资源关闭入口，用于验证生命周期调用而不释放外部资源。
func (d *fakeDatabase) Close() error {
	return nil
}

// Ping 实现数据库测试桩的健康检查入口，按测试需要返回成功或预设错误。
func (d *fakeDatabase) Ping(context.Context) error {
	return nil
}

func (d *fakeDatabase) Exec(context.Context, string, ...any) (database.Result, error) {
	d.dbCalls++
	return database.Result{}, nil
}

func (d *fakeDatabase) Raw(context.Context, any, string, ...any) (database.Result, error) {
	d.dbCalls++
	return database.Result{}, nil
}

func (d *fakeDatabase) HasTable(context.Context, any) (bool, error) {
	d.dbCalls++
	return false, nil
}

type fakeLogger struct {
	reloads    int
	lastConfig *logger.Config
}

// Debug 实现测试日志桩的同名输出入口，当前测试只关心接口满足而不采集日志内容。
func (l *fakeLogger) Debug(string, ...interface{}) {}

// Info 实现测试日志桩的同名输出入口，当前测试只关心接口满足而不采集日志内容。
func (l *fakeLogger) Info(string, ...interface{}) {}

// Warn 实现测试日志桩的同名输出入口，当前测试只关心接口满足而不采集日志内容。
func (l *fakeLogger) Warn(string, ...interface{}) {}

// Error 实现测试日志桩的同名输出入口，当前测试只关心接口满足而不采集日志内容。
func (l *fakeLogger) Error(string, ...interface{}) {}

// Fatal 实现测试日志桩的同名输出入口，当前测试只关心接口满足而不采集日志内容。
func (l *fakeLogger) Fatal(string, ...interface{}) {}

// With 实现测试日志桩的字段绑定入口，返回自身以保持 logger.Logger 链式调用契约。
func (l *fakeLogger) With(...interface{}) logger.Logger {
	return l
}

// Sync 实现测试日志桩的刷新入口，测试环境不持有真实缓冲区。
func (l *fakeLogger) Sync() error {
	return nil
}

// Reload 实现测试桩的配置重载入口，用于验证调用路径而不触发真实资源替换。
func (l *fakeLogger) Reload(cfg *logger.Config) error {
	l.reloads++
	l.lastConfig = cfg
	return nil
}

type fakeExecutor struct {
	executor.Manager
	reloads     int
	shutdowns   int
	lastConfigs []executor.Config
}

// Reload 实现测试桩的配置重载入口，用于验证调用路径而不触发真实资源替换。
func (e *fakeExecutor) Reload(configs []executor.Config) error {
	e.reloads++
	e.lastConfigs = configs
	return nil
}

// Shutdown 实现测试桩的关闭入口，用于覆盖热加载或生命周期编排中的分支。
func (e *fakeExecutor) Shutdown() {
	e.shutdowns++
}

type fakeHTTPServer struct {
	httpserver.HTTPServer
	reloads    int
	lastConfig *httpserver.Config
}

// Reload 实现测试桩的配置重载入口，用于验证调用路径而不触发真实资源替换。
func (s *fakeHTTPServer) Reload(_ context.Context, cfg *httpserver.Config) error {
	s.reloads++
	s.lastConfig = cfg
	return nil
}

type fakeRPCServer struct {
	rpcserver.Server
	reloads    int
	lastConfig *rpcserver.Config
}

// Reload 实现测试桩的 RPC 配置重载入口。
func (s *fakeRPCServer) Reload(_ context.Context, cfg *rpcserver.Config) error {
	s.reloads++
	s.lastConfig = cfg
	return nil
}

type fakeStorage struct {
	storagepkg.Storage
	reloads    int
	closes     int
	lastConfig *storagepkg.Config
}

// Reload 实现测试桩的配置重载入口，用于验证调用路径而不触发真实资源替换。
func (s *fakeStorage) Reload(_ context.Context, cfg *storagepkg.Config) error {
	s.reloads++
	s.lastConfig = cfg
	return nil
}

// Close 实现测试桩的资源关闭入口，用于验证生命周期调用而不释放外部资源。
func (s *fakeStorage) Close() error {
	s.closes++
	return nil
}

// clearReloadEnv 清理测试期间设置的环境变量或全局状态，避免用例之间互相污染。
func clearReloadEnv(t *testing.T) {
	t.Helper()

	keys := []string{
		"STORAGE_DRIVER",
		"STORAGE_LOCAL_FS_TYPE",
		"STORAGE_LOCAL_BASE_PATH",
		"STORAGE_LOCAL_PUBLIC_URL",
		"STORAGE_LOCAL_ENABLE_WATCH",
		"STORAGE_LOCAL_WATCH_BUFFER_SIZE",
	}
	for _, key := range keys {
		t.Setenv(key, "")
		t.Setenv(config.EnvPrefixJoin(key), "")
	}
}

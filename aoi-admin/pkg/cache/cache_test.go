package cache

// 本测试文件固定 Redis 缓存适配器的配置、操作和热重载契约，防止注释补全和后续重构改变外部可观察行为。

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

type testLogger struct {
	infos  []string
	errors []string
}

// Info 实现测试日志桩的同名输出入口，当前测试只关心接口满足而不采集日志内容。
func (l *testLogger) Info(msg string, keysAndValues ...interface{}) {
	l.infos = append(l.infos, msg)
}

// Error 实现测试日志桩的同名输出入口，当前测试只关心接口满足而不采集日志内容。
func (l *testLogger) Error(msg string, keysAndValues ...interface{}) {
	l.errors = append(l.errors, msg)
}

// runRedis 启动测试依赖服务或场景，并返回清理函数以保证资源可回收。
func runRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()

	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	t.Cleanup(server.Close)

	return server
}

// redisConfig 是当前测试文件的辅助函数，用于复用夹具、断言或输入构造逻辑。
func redisConfig(t *testing.T, server *miniredis.Miniredis) *Config {
	t.Helper()

	host, portText, ok := strings.Cut(server.Addr(), ":")
	if !ok {
		t.Fatalf("unexpected redis addr %q", server.Addr())
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("redis port %q is not numeric: %v", portText, err)
	}

	cfg := DefaultConfig()
	cfg.Host = host
	cfg.Port = port
	cfg.PoolSize = 2
	cfg.MinIdleConns = 0
	cfg.MaxRetries = 0
	cfg.DialTimeout = 100 * time.Millisecond
	cfg.ReadTimeout = 100 * time.Millisecond
	cfg.WriteTimeout = 100 * time.Millisecond
	return cfg
}

// newTestRedisCache 构造当前测试场景所需的最小依赖集合，避免测试直接耦合生产装配流程。
func newTestRedisCache(t *testing.T, server *miniredis.Miniredis) Cache {
	t.Helper()

	cache, err := NewRedis(redisConfig(t, server), &testLogger{})
	if err != nil {
		t.Fatalf("NewRedis() error = %v", err)
	}
	t.Cleanup(func() {
		if err := cache.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	return cache
}

// TestDefaultConfigAndValidate 固定 Redis 缓存适配器的配置、操作和热重载契约，确保后续注释补全或结构调整不改变该场景。
func TestDefaultConfigAndValidate(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Host != DefaultHost || cfg.Port != DefaultPort || cfg.DB != DefaultDB {
		t.Fatalf("DefaultConfig() = host %q port %d db %d, want defaults", cfg.Host, cfg.Port, cfg.DB)
	}
	if cfg.PoolSize != DefaultPoolSize || cfg.MinIdleConns != DefaultMinIdleConns || cfg.MaxRetries != DefaultMaxRetries {
		t.Fatalf("DefaultConfig() pool settings = %d/%d retries %d, want defaults", cfg.PoolSize, cfg.MinIdleConns, cfg.MaxRetries)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("DefaultConfig().Validate() error = %v", err)
	}

	tests := []struct {
		name string
		edit func(*Config)
		want string
	}{
		{
			name: "empty host",
			edit: func(c *Config) { c.Host = "" },
			want: "redis host cannot be empty",
		},
		{
			name: "invalid port",
			edit: func(c *Config) { c.Port = 0 },
			want: "invalid redis port",
		},
		{
			name: "invalid db",
			edit: func(c *Config) { c.DB = 16 },
			want: "invalid redis db",
		},
		{
			name: "invalid pool size",
			edit: func(c *Config) { c.PoolSize = 0 },
			want: "redis pool size must be greater than 0",
		},
		{
			name: "negative min idle",
			edit: func(c *Config) { c.MinIdleConns = -1 },
			want: "redis min idle conns cannot be negative",
		},
		{
			name: "min idle greater than pool",
			edit: func(c *Config) { c.MinIdleConns = c.PoolSize + 1 },
			want: "cannot be greater than pool size",
		},
		{
			name: "invalid dial timeout",
			edit: func(c *Config) { c.DialTimeout = 0 },
			want: "redis dial timeout must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := *DefaultConfig()
			tt.edit(&cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("Validate() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() error = %q, want to contain %q", err.Error(), tt.want)
			}
		})
	}
}

// TestRedisCacheBasicOperations 固定 Redis 缓存适配器的配置、操作和热重载契约，确保后续注释补全或结构调整不改变该场景。
func TestRedisCacheBasicOperations(t *testing.T) {
	ctx := context.Background()
	server := runRedis(t)
	cache := newTestRedisCache(t, server)

	if err := cache.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	if _, err := cache.Get(ctx, "missing"); err == nil || !strings.Contains(err.Error(), "cache key not found: missing") {
		t.Fatalf("Get(missing) error = %v, want key-not-found error", err)
	}

	if err := cache.Set(ctx, "name", "alice", 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	got, err := cache.Get(ctx, "name")
	if err != nil {
		t.Fatalf("Get(name) error = %v", err)
	}
	if got != "alice" {
		t.Fatalf("Get(name) = %q, want alice", got)
	}

	count, err := cache.Exists(ctx, "name", "missing")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("Exists() = %d, want 1", count)
	}

	if err := cache.Expire(ctx, "name", time.Minute); err != nil {
		t.Fatalf("Expire(name) error = %v", err)
	}
	ttl, err := cache.TTL(ctx, "name")
	if err != nil {
		t.Fatalf("TTL(name) error = %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("TTL(name) = %v, want positive duration", ttl)
	}

	if err := cache.Expire(ctx, "missing", time.Minute); err == nil || !strings.Contains(err.Error(), "cache key not found: missing") {
		t.Fatalf("Expire(missing) error = %v, want key-not-found error", err)
	}

	if err := cache.Set(ctx, "short-lived", "value", time.Second); err != nil {
		t.Fatalf("Set(short-lived) error = %v", err)
	}
	server.FastForward(2 * time.Second)
	if _, err := cache.Get(ctx, "short-lived"); err == nil || !strings.Contains(err.Error(), "cache key not found: short-lived") {
		t.Fatalf("Get(short-lived) error = %v, want expired key-not-found error", err)
	}

	if err := cache.Delete(ctx, "name"); err != nil {
		t.Fatalf("Delete(name) error = %v", err)
	}
	count, err = cache.Exists(ctx, "name")
	if err != nil {
		t.Fatalf("Exists(name after delete) error = %v", err)
	}
	if count != 0 {
		t.Fatalf("Exists(name after delete) = %d, want 0", count)
	}
}

// TestRedisCacheBatchAndCounterOperations 固定 Redis 缓存适配器的配置、操作和热重载契约，确保后续注释补全或结构调整不改变该场景。
func TestRedisCacheBatchAndCounterOperations(t *testing.T) {
	ctx := context.Background()
	cache := newTestRedisCache(t, runRedis(t))

	if err := cache.Delete(ctx); err != nil {
		t.Fatalf("Delete() with no keys error = %v", err)
	}

	values, err := cache.MGet(ctx)
	if err != nil {
		t.Fatalf("MGet() with no keys error = %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("MGet() with no keys len = %d, want 0", len(values))
	}

	if err := cache.MSet(ctx); err != nil {
		t.Fatalf("MSet() with no pairs error = %v", err)
	}
	if err := cache.MSet(ctx, "one"); err == nil || !strings.Contains(err.Error(), "even number") {
		t.Fatalf("MSet() odd pairs error = %v, want even number error", err)
	}

	if err := cache.MSet(ctx, "one", "1", "two", "2"); err != nil {
		t.Fatalf("MSet() error = %v", err)
	}
	values, err = cache.MGet(ctx, "one", "missing", "two")
	if err != nil {
		t.Fatalf("MGet() error = %v", err)
	}
	if len(values) != 3 || values[0] != "1" || values[1] != nil || values[2] != "2" {
		t.Fatalf("MGet() = %#v, want [1 nil 2]", values)
	}

	got, err := cache.Incr(ctx, "counter")
	if err != nil {
		t.Fatalf("Incr() error = %v", err)
	}
	if got != 1 {
		t.Fatalf("Incr() = %d, want 1", got)
	}

	got, err = cache.IncrBy(ctx, "counter", 4)
	if err != nil {
		t.Fatalf("IncrBy() error = %v", err)
	}
	if got != 5 {
		t.Fatalf("IncrBy() = %d, want 5", got)
	}

	got, err = cache.Decr(ctx, "counter")
	if err != nil {
		t.Fatalf("Decr() error = %v", err)
	}
	if got != 4 {
		t.Fatalf("Decr() = %d, want 4", got)
	}
}

// TestRedisCacheReloadKeepsOldClientOnFailureAndSwitchesOnSuccess 固定 Redis 缓存适配器的配置、操作和热重载契约，确保后续注释补全或结构调整不改变该场景。
func TestRedisCacheReloadKeepsOldClientOnFailureAndSwitchesOnSuccess(t *testing.T) {
	ctx := context.Background()
	oldServer := runRedis(t)
	cache := newTestRedisCache(t, oldServer)

	if err := cache.Set(ctx, "stable", "old", 0); err != nil {
		t.Fatalf("Set(stable) error = %v", err)
	}

	badConfig := *redisConfig(t, oldServer)
	badConfig.Port = 1
	badConfig.DialTimeout = time.Millisecond
	badConfig.ReadTimeout = time.Millisecond
	badConfig.WriteTimeout = time.Millisecond
	if err := cache.Reload(ctx, &badConfig); err == nil {
		t.Fatal("Reload() with unreachable redis error = nil, want failure")
	}

	got, err := cache.Get(ctx, "stable")
	if err != nil {
		t.Fatalf("Get(stable) after failed reload error = %v", err)
	}
	if got != "old" {
		t.Fatalf("Get(stable) after failed reload = %q, want old", got)
	}

	newServer := runRedis(t)
	if err := cache.Reload(ctx, redisConfig(t, newServer)); err != nil {
		t.Fatalf("Reload() with new redis error = %v", err)
	}

	if _, err := cache.Get(ctx, "stable"); err == nil || !strings.Contains(err.Error(), "cache key not found: stable") {
		t.Fatalf("Get(stable) after successful reload error = %v, want key-not-found from new redis", err)
	}
	if err := cache.Set(ctx, "fresh", "new", 0); err != nil {
		t.Fatalf("Set(fresh) after reload error = %v", err)
	}
	got, err = newServer.Get("fresh")
	if err != nil {
		t.Fatalf("new redis Get(fresh) error = %v", err)
	}
	if got != "new" {
		t.Fatalf("new redis fresh value = %q, want new", got)
	}
}

func TestLocalCacheOperations(t *testing.T) {
	ctx := context.Background()
	cache, err := NewLocal(LocalConfig{MaxCost: 1 << 20, NumCounters: 1000, BufferItems: 8})
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}
	t.Cleanup(func() {
		if err := cache.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	if err := cache.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	if err := cache.Set(ctx, "name", "local", time.Minute); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	got, err := cache.Get(ctx, "name")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "local" {
		t.Fatalf("Get() = %q, want local", got)
	}
	if count, err := cache.Exists(ctx, "name", "missing"); err != nil || count != 1 {
		t.Fatalf("Exists() = %d, %v; want 1 nil", count, err)
	}
	if ttl, err := cache.TTL(ctx, "name"); err != nil || ttl <= 0 {
		t.Fatalf("TTL() = %v, %v; want positive ttl", ttl, err)
	}
	if next, err := cache.IncrBy(ctx, "counter", 5); err != nil || next != 5 {
		t.Fatalf("IncrBy() = %d, %v; want 5 nil", next, err)
	}
	if err := cache.Delete(ctx, "name"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := cache.Get(ctx, "name"); err == nil || !IsKeyNotFound(err) {
		t.Fatalf("Get(deleted) error = %v, want key-not-found", err)
	}
}

func TestHybridCacheReadThroughAndLocalDegrade(t *testing.T) {
	ctx := context.Background()
	server := runRedis(t)
	redisCfg := redisConfig(t, server)
	cache, err := NewHybrid(LocalConfig{MaxCost: 1 << 20, NumCounters: 1000, BufferItems: 8}, redisCfg, &testLogger{})
	if err != nil {
		t.Fatalf("NewHybrid() error = %v", err)
	}
	t.Cleanup(func() {
		if err := cache.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	if err := cache.Set(ctx, "shared", "value", 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	got, err := server.Get("shared")
	if err != nil || got != "value" {
		t.Fatalf("redis Get(shared) = %q, %v; want value nil", got, err)
	}
	if err := cache.Delete(ctx, "shared"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err := server.Set("remote", "redis-only"); err != nil {
		t.Fatalf("redis Set(remote) error = %v", err)
	}
	got, err = cache.Get(ctx, "remote")
	if err != nil || got != "redis-only" {
		t.Fatalf("Get(remote) = %q, %v; want redis-only nil", got, err)
	}

	badRedis := *redisCfg
	badRedis.Port = 1
	badRedis.DialTimeout = time.Millisecond
	badRedis.ReadTimeout = time.Millisecond
	badRedis.WriteTimeout = time.Millisecond
	degraded, err := NewHybrid(LocalConfig{MaxCost: 1 << 20, NumCounters: 1000, BufferItems: 8}, &badRedis, &testLogger{})
	if err != nil {
		t.Fatalf("NewHybrid(degraded) error = %v", err)
	}
	defer degraded.Close()
	if err := degraded.Set(ctx, "local", "ok", 0); err != nil {
		t.Fatalf("degraded Set() error = %v", err)
	}
	got, err = degraded.Get(ctx, "local")
	if err != nil || got != "ok" {
		t.Fatalf("degraded Get() = %q, %v; want ok nil", got, err)
	}
}

package initcenter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/internal/app/initapp"
	appconfig "github.com/rei0721/go-scaffold/internal/config"
	"github.com/rei0721/go-scaffold/pkg/cache"
	"github.com/rei0721/go-scaffold/pkg/database"
	"github.com/rei0721/go-scaffold/pkg/storage"
)

type InitValidator struct {
	service *Service
}

func NewInitValidator(service *Service) InitValidator {
	return InitValidator{service: service}
}

func (v InitValidator) Test(ctx context.Context, stepKey string, values map[string]any) TestResult {
	now := time.Now().UTC()
	result := TestResult{StepKey: stepKey, Status: "succeeded", TestedAt: now}
	cfg, err := NewInitConfigStore(v.service.core, v.service.configPath).candidate(stepKey, values)
	if err != nil {
		return failedTest(stepKey, now, err.Error(), "检查当前步骤输入是否完整且类型正确。")
	}
	switch stepKey {
	case "database.configure":
		return v.testDatabase(ctx, cfg, now)
	case "cache.configure":
		return v.testCache(ctx, cfg, now)
	case "storage.configure":
		return v.testStorage(ctx, cfg, now)
	case "system.configure":
		return v.testSystemConfig(cfg, now)
	case "site.configure":
		return v.testSiteConfig(cfg, now)
	default:
		result.Summary = "no active test for this step"
		return result
	}
}

func (v InitValidator) testDatabase(ctx context.Context, cfg *appconfig.Config, now time.Time) TestResult {
	result := TestResult{StepKey: "database.configure", TestedAt: now}
	if err := cfg.Database.Validate(); err != nil {
		return failedTest("database.configure", now, err.Error(), "修正数据库驱动及对应分支配置后重试。")
	}
	if cfg.Database.Driver == appconfig.DatabaseDriverSQLite {
		dir := filepath.Dir(cfg.Database.SQLite.Path)
		if dir != "." {
			if err := os.MkdirAll(dir, 0700); err != nil {
				return failedTest("database.configure", now, err.Error(), "确认 SQLite 数据目录可创建且当前进程有写入权限。")
			}
		}
	}
	db, err := database.New(initapp.DatabaseConfig(cfg))
	if err != nil {
		return failedTest("database.configure", now, err.Error(), "确认数据库已启动、账号密码正确，并具备连接权限。")
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		return failedTest("database.configure", now, err.Error(), "确认数据库网络连通、库名存在，并具备访问权限。")
	}
	result.Status = "succeeded"
	result.Summary = fmt.Sprintf("%s 数据库连接测试通过", cfg.Database.Driver)
	if v.service != nil && databaseFingerprintFor(cfg) != v.service.databaseFingerprint() {
		result.RestartRequired = true
		result.RepairHint = "当前输入可以连接，但保存数据库配置后需要重启服务，重启后系统会继续初始化。"
	}
	return result
}

func (v InitValidator) testCache(ctx context.Context, cfg *appconfig.Config, now time.Time) TestResult {
	driver := cfg.Cache.Driver
	if driver == appconfig.CacheDriverDisabled {
		return TestResult{StepKey: "cache.configure", Status: "succeeded", Summary: "cache=disabled", TestedAt: now}
	}
	if driver == appconfig.CacheDriverLocal {
		client, err := cache.NewLocal(initapp.CacheRuntimeConfig(cfg).Local)
		if err != nil {
			return failedTest("cache.configure", now, "local cache init failed: "+err.Error(), "调整本地缓存容量、计数器或写入缓冲后重试。")
		}
		defer client.Close()
		if err := exerciseCache(ctx, client); err != nil {
			return failedTest("cache.configure", now, "local cache read/write failed: "+err.Error(), "调整本地缓存容量后重试。")
		}
		return TestResult{StepKey: "cache.configure", Status: "succeeded", Summary: "cache=local ok", TestedAt: now}
	}

	redisClient, err := cache.NewRedis(initapp.RedisConfig(cfg.Cache.Redis), v.service.core.Logger)
	if err != nil {
		if driver == appconfig.CacheDriverHybrid {
			return v.testHybridLocalFallback(ctx, cfg, now, err)
		}
		return failedTest("cache.configure", now, "redis connection failed: "+err.Error(), "确认 Redis 地址、账号、密码和 DB 编号后重试。")
	}
	defer redisClient.Close()
	if err := redisClient.Ping(ctx); err != nil {
		return failedTest("cache.configure", now, "redis ping failed: "+err.Error(), "确认 Redis 服务可用并允许当前账号访问。")
	}
	if err := exerciseCache(ctx, redisClient); err != nil {
		return failedTest("cache.configure", now, "redis read/write failed: "+err.Error(), "确认 Redis 账号具备读写权限。")
	}
	if driver == appconfig.CacheDriverHybrid {
		localClient, err := cache.NewLocal(initapp.CacheRuntimeConfig(cfg).Local)
		if err != nil {
			return failedTest("cache.configure", now, "hybrid local cache init failed: "+err.Error(), "调整本地缓存容量、计数器或写入缓冲后重试。")
		}
		defer localClient.Close()
		if err := exerciseCache(ctx, localClient); err != nil {
			return failedTest("cache.configure", now, "hybrid local cache read/write failed: "+err.Error(), "确认本地缓存容量足够。")
		}
		return TestResult{StepKey: "cache.configure", Status: "succeeded", Summary: "cache=hybrid ok", TestedAt: now}
	}
	return TestResult{StepKey: "cache.configure", Status: "succeeded", Summary: "cache=redis ok", TestedAt: now}
}

func (v InitValidator) testHybridLocalFallback(ctx context.Context, cfg *appconfig.Config, now time.Time, redisErr error) TestResult {
	localClient, localErr := cache.NewLocal(initapp.CacheRuntimeConfig(cfg).Local)
	if localErr != nil {
		return failedTest("cache.configure", now, "hybrid local cache init failed: "+localErr.Error(), "先修复本地缓存容量配置，再重试。")
	}
	defer localClient.Close()
	if localErr := exerciseCache(ctx, localClient); localErr != nil {
		return failedTest("cache.configure", now, "hybrid local cache read/write failed: "+localErr.Error(), "先修复本地缓存配置，再重试。")
	}
	return TestResult{
		StepKey:    "cache.configure",
		Status:     "succeeded",
		Summary:    "cache=hybrid degraded; local ok; redis failed: " + redisErr.Error(),
		RepairHint: "Redis 不可用时 Hybrid 会临时降级为本地缓存；如需跨进程共享，请修复 Redis 地址、账号、密码和 DB 编号。",
		TestedAt:   now,
	}
}

func (v InitValidator) testStorage(ctx context.Context, cfg *appconfig.Config, now time.Time) TestResult {
	if cfg.Storage.Driver == appconfig.StorageDriverDisabled {
		return TestResult{StepKey: "storage.configure", Status: "succeeded", Summary: "storage=disabled", TestedAt: now}
	}
	if err := cfg.Storage.Validate(); err != nil {
		return failedTest("storage.configure", now, "storage config invalid: "+err.Error(), "确认存储驱动、本地目录、Endpoint、Bucket 和密钥配置完整。")
	}
	manager, err := storage.NewManager(ctx, cfg.Storage.ToPkgConfig())
	if err != nil {
		return failedTest("storage.configure", now, "storage init failed: "+err.Error(), "确认存储驱动对应的本地目录或对象存储连接参数可用。")
	}
	defer manager.Close()
	parts := []string{}
	if manager.Local != nil {
		if err := storage.ExerciseClient(ctx, manager.Local); err != nil {
			return failedTest("storage.configure", now, "local storage read/write failed: "+err.Error(), "确认本地存储目录可创建、可写入、可删除。")
		}
		parts = append(parts, "local=ok")
	}
	if manager.Object != nil {
		if err := storage.ExerciseClient(ctx, manager.Object); err != nil {
			return failedTest("storage.configure", now, "object storage read/write failed: "+err.Error(), "确认 Endpoint、Bucket、Access Key、Secret、Path-style 策略和网络连通性。")
		}
		parts = append(parts, "object=ok")
	}
	if len(parts) == 0 {
		parts = append(parts, "storage=disabled")
	}
	return TestResult{StepKey: "storage.configure", Status: "succeeded", Summary: "storage " + strings.Join(parts, ", "), TestedAt: now}
}

func (v InitValidator) testSystemConfig(cfg *appconfig.Config, now time.Time) TestResult {
	if err := cfg.I18n.Validate(); err != nil {
		return failedTest("system.configure", now, err.Error(), "确认默认语言位于支持语言列表中。")
	}
	if err := cfg.Auth.Validate(); err != nil {
		return failedTest("system.configure", now, err.Error(), "确认 IAM issuer、密钥和密码策略配置完整。")
	}
	return TestResult{StepKey: "system.configure", Status: "succeeded", Summary: "system configuration ok", TestedAt: now}
}

func (v InitValidator) testSiteConfig(cfg *appconfig.Config, now time.Time) TestResult {
	if strings.TrimSpace(cfg.Brand.ProductName) == "" {
		return failedTest("site.configure", now, "productName is required", "确认产品名称不为空。")
	}
	if strings.TrimSpace(cfg.Brand.VersionName) == "" {
		return failedTest("site.configure", now, "versionName is required", "确认版本展示名不为空。")
	}
	return TestResult{StepKey: "site.configure", Status: "succeeded", Summary: "site configuration ok", TestedAt: now}
}

func failedTest(stepKey string, testedAt time.Time, message string, hint string) TestResult {
	return TestResult{StepKey: stepKey, Status: "failed", Error: message, RepairHint: hint, TestedAt: testedAt}
}

func exerciseCache(ctx context.Context, client cache.Cache) error {
	const key = "setup:cache:healthcheck"
	if err := client.Set(ctx, key, "ok", time.Minute); err != nil {
		return err
	}
	value, err := client.Get(ctx, key)
	if err != nil {
		return err
	}
	if value != "ok" {
		return fmt.Errorf("unexpected cache value %q", value)
	}
	return client.Delete(ctx, key)
}

package config

// 本文件属于配置子系统，处理配置加载、环境变量覆盖、运行时快照或跨分区校验。

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/rei0721/go-scaffold/pkg/configloader"
	"github.com/rei0721/go-scaffold/pkg/logger"
)

// HookHandler 配置变更时调用的回调函数
// 参数:
//
//	old: 旧配置
//	new: 新配置
//
// 使用场景:
//   - 配置热更新时需要重新初始化某些组件
//   - 记录配置变更日志
//   - 通知其他模块配置已更新
type HookHandler func(old, new *Config)

// LoggerHandler 返回日志记录器实例的函数
// 为什么使用函数而不是直接传递 Logger:
//   - 延迟初始化:日志器可能在配置管理器之后初始化
//   - 动态获取:每次需要时获取最新的日志器实例
type LoggerHandler func() logger.Logger

// Manager 定义配置管理器接口
// 提供配置加载、更新、监听等功能
// 为什么使用接口:
//   - 便于测试(可以 mock)
//   - 支持不同的实现
//   - 符合依赖倒置原则
type Manager interface {
	// Load 从指定路径加载配置
	// 参数:
	//   configPath: 配置文件路径(支持 YAML、JSON 等)
	// 返回:
	//   error: 加载或验证失败时的错误
	Load(configPath string) error

	// Get 返回只读的配置快照
	// 返回:
	//   *Config: 当前配置的副本
	// 注意:
	//   这是线程安全的,可以并发调用
	Get() *Config

	// Update 原子地更新配置
	// 参数:
	//   fn: 更新函数,接收配置副本并修改
	// 返回:
	//   error: 验证失败时的错误
	// 使用示例:
	//   manager.Update(func(cfg *Config) {
	//       cfg.Server.Port = 9090
	//   })
	Update(fn func(*Config), options ...UpdateOption) error

	// RegisterHook 注册配置变更钩子
	// 参数:
	//   h: 钩子处理函数
	// 用途:
	//   当配置重新加载时,所有注册的钩子都会被调用
	RegisterHook(h HookHandler)

	// RegisterLogger 注册日志处理器并返回日志器
	// 参数:
	//   h: 日志处理器函数
	// 返回:
	//   logger.Logger: 日志器实例
	RegisterLogger(h LoggerHandler) logger.Logger

	// Watch 开始监听配置文件变化
	// 返回:
	//   error: 启动监听失败时的错误
	// 功能:
	//   自动检测配置文件变化并重新加载
	Watch() error
}

// manager 实现 Manager 接口
// 使用 viper 进行配置管理,支持配置热重载
// 线程安全设计:
//   - 使用 atomic.Pointer 存储配置(无锁读取)
//   - 使用 RWMutex 保护 hooks 列表
type manager struct {
	// v viper 实例,用于读取配置文件
	// viper 支持多种格式:YAML、JSON、TOML 等
	v *configloader.Loader

	// config 当前配置
	// 使用 atomic.Pointer 实现无锁并发读取
	// 写入时原子替换整个配置对象
	config atomic.Pointer[Config]

	// configPath 配置文件路径
	// 用于监听文件变化
	configPath string

	// hooks 配置变更钩子列表
	// 当配置重新加载时,按注册顺序调用
	hooks []HookHandler

	// hooksMu 保护 hooks 列表的读写锁
	// 读锁:通知钩子时
	// 写锁:注册新钩子时
	hooksMu sync.RWMutex

	updateMu sync.Mutex

	// loggerHandler 日志处理器
	// 延迟获取日志器,因为日志器可能在配置管理器之后初始化
	loggerHandler LoggerHandler

	// log 日志记录器实例
	// 用于记录配置加载、更新等事件
	log logger.Logger
}

// NewManager 创建一个新的配置管理器
// 返回:
//
//	Manager: 配置管理器接口
//
// 使用示例:
//
//	mgr := config.NewManager()
//	mgr.Load("config.yaml")
func NewManager() Manager {
	return &manager{
		v:     configloader.New(),     // 创建新的配置加载器实例
		hooks: make([]HookHandler, 0), // 初始化空的钩子列表
	}
}

// Load 从指定路径加载配置
// 加载流程:
//  1. 设置配置文件路径
//  2. 读取配置文件
//  3. 处理环境变量替换(${VAR:default})
//  4. 反序列化到 Config 结构体
//  5. 验证配置
//  6. 原子存储配置
//
// 参数:
//
//	configPath: 配置文件路径
//
// 返回:
//
//	error: 加载失败时的错误
func (m *manager) Load(configPath string) error {
	// 保存配置文件路径,用于 Watch
	m.configPath = configPath

	// 1. 加载 .env 文件(如果存在)
	// 这应该在读取 config.yaml 之前完成
	// .env 文件中的变量会被加载到进程环境变量中
	// 已存在的系统环境变量不会被覆盖
	LoadEnv()

	// 2. 设置配置文件
	// viper 会根据文件扩展名自动检测格式
	m.v.SetConfigFile(configPath)

	// 3. 读取配置文件
	// 这会解析文件内容到 viper 内部结构
	if err := m.v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// 4. 处理环境变量替换
	// 将配置中的 ${VAR_NAME:default} 替换为环境变量值
	// 例如: port: ${PORT:8080} -> port: 8080(如果 PORT 未设置)
	if err := m.processEnvSubstitution(); err != nil {
		return fmt.Errorf("failed to process env substitution: %w", err)
	}

	// 5. 反序列化为 Config 结构体
	// viper 会根据 mapstructure tag 映射字段
	cfg := &Config{}
	if err := m.v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 6. 使用环境变量覆盖配置
	// 优先级: 环境变量 > config.yaml
	// 这允许通过环境变量覆盖配置文件中的任何值
	// 特别适合容器环境和CI/CD流程
	OverrideWithEnvExcept(cfg, cfg.EnvOverride.DisabledPaths)

	// 7. 验证配置
	// 确保所有必需的字段都有有效值
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// 8. 原子存储配置
	// 使用 atomic.Pointer.Store 确保并发安全
	m.config.Store(cfg)

	return nil
}

// processEnvSubstitution 处理配置值中的环境变量替换
// 支持的语法:
//
//	${VAR_NAME}          - 环境变量值,如果不存在则为空字符串
//	${VAR_NAME:default}  - 环境变量值,如果不存在则使用默认值
//
// 示例:
//
//	port: ${PORT:8080}
//	host: ${HOST:localhost}
//
// 返回:
//
//	error: 处理失败时的错误
func (m *manager) processEnvSubstitution() error {
	// 编译正则表达式匹配 ${VAR:default} 格式
	// 捕获组:
	//   1: 变量名
	//   2: 默认值(可选)
	envPattern := regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)

	// 获取所有配置项
	// 返回 map[string]any,包含所有配置的键值对
	settings := m.v.AllSettings()
	disabledPaths := disabledEnvOverridePathsFromSettings(settings)

	// 递归处理所有配置值
	processed := m.processMap(settings, envPattern, "", disabledPaths)

	// 将处理后的值设置回 viper
	for key, value := range processed {
		m.v.Set(key, value)
	}

	return nil
}

// processMap 递归处理 map 中的环境变量替换
// 参数:
//
//	data: 要处理的 map
//	pattern: 环境变量匹配正则表达式
//
// 返回:
//
//	map[string]any: 处理后的 map
func (m *manager) processMap(data map[string]any, pattern *regexp.Regexp, basePath string, disabledPaths map[string]struct{}) map[string]any {
	result := make(map[string]any)
	for key, value := range data {
		// 递归处理每个值
		path := configSettingPath(basePath, key)
		result[key] = m.processValue(value, pattern, path, disabledPaths)
	}
	return result
}

func configSettingPath(basePath string, key string) string {
	key = strings.TrimSpace(key)
	if basePath == "" {
		return key
	}
	if key == "" {
		return basePath
	}
	return basePath + "." + key
}

func disabledEnvOverridePathsFromSettings(settings map[string]any) map[string]struct{} {
	disabled := map[string]struct{}{}
	envOverride, ok := settings["env_override"].(map[string]any)
	if !ok {
		return disabled
	}
	for _, path := range stringSliceFromSetting(envOverride["disabled_paths"]) {
		disabled[path] = struct{}{}
	}
	return disabled
}

func stringSliceFromSetting(value any) []string {
	switch typed := value.(type) {
	case []string:
		return normalizeConfigPaths(typed)
	case []any:
		paths := make([]string, 0, len(typed))
		for _, item := range typed {
			if path, ok := item.(string); ok {
				paths = append(paths, path)
			}
		}
		return normalizeConfigPaths(paths)
	default:
		return nil
	}
}

// processValue 处理单个值的环境变量替换
// 支持的类型:
//   - string: 执行环境变量替换
//   - map: 递归处理
//   - slice: 递归处理每个元素
//   - 其他: 原样返回
//
// 参数:
//
//	value: 要处理的值
//	pattern: 环境变量匹配正则表达式
//
// 返回:
//
//	any: 处理后的值
func (m *manager) processValue(value any, pattern *regexp.Regexp, path string, disabledPaths map[string]struct{}) any {
	switch v := value.(type) {
	case string:
		// 字符串类型,执行环境变量替换
		_, disabled := disabledPaths[path]
		return m.substituteEnv(v, pattern, !disabled)

	case map[string]any:
		// 嵌套 map,递归处理
		return m.processMap(v, pattern, path, disabledPaths)

	case []any:
		// 数组,处理每个元素
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = m.processValue(item, pattern, path, disabledPaths)
		}
		return result

	default:
		// 其他类型(int、bool 等),原样返回
		return value
	}
}

// substituteEnv 替换字符串中的环境变量
// 处理逻辑:
//  1. 查找所有匹配 ${VAR:default} 的部分
//  2. 对每个匹配,尝试获取环境变量
//  3. 如果环境变量存在,使用其值
//  4. 如果不存在,使用默认值
//
// 参数:
//
//	s: 要处理的字符串
//	pattern: 环境变量匹配正则表达式
//
// 返回:
//
//	string: 替换后的字符串
func (m *manager) substituteEnv(s string, pattern *regexp.Regexp, useEnvironment bool) string {
	return pattern.ReplaceAllStringFunc(s, func(match string) string {
		// 提取变量名和默认值
		submatches := pattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			// 匹配失败,返回原字符串
			return match
		}

		// submatches[0]: 完整匹配 "${VAR:default}"
		// submatches[1]: 变量名 "VAR"
		// submatches[2]: 默认值 "default"(可选)
		envName := submatches[1]
		defaultValue := ""
		if len(submatches) >= 3 {
			defaultValue = submatches[2]
		}

		// 尝试获取环境变量
		if useEnvironment {
			if envValue := os.Getenv(envName); envValue != "" {
				// 环境变量存在,使用其值
				return envValue
			}
		}
		// 环境变量不存在,使用默认值
		return defaultValue
	})
}

// Get 返回只读的配置快照
// 实现:
//
//	使用 atomic.Pointer.Load 进行无锁读取
//	并发安全,多个 goroutine 可以同时调用
//
// 返回:
//
//	*Config: 当前配置(只读)
func (m *manager) Get() *Config {
	return m.config.Load()
}

// Update 原子地更新配置
// 更新流程:
//  1. 获取当前配置
//  2. 创建配置副本
//  3. 应用更新函数
//  4. 验证新配置
//  5. 原子替换配置
//  6. 通知钩子
//
// 参数:
//
//	fn: 更新函数,接收配置副本并修改
//
// 返回:
//
//	error: 验证失败时的错误
//
// 线程安全:
//
//	使用原子操作确保并发安全
func (m *manager) Update(fn func(*Config), options ...UpdateOption) error {
	updateOptions := collectUpdateOptions(options)
	m.updateMu.Lock()
	defer m.updateMu.Unlock()

	// 获取当前配置
	oldCfg := m.Get()
	if oldCfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// 创建配置副本
	// 避免直接修改原配置
	newCfg := m.copyConfig(oldCfg)

	// 应用更新函数
	// 用户在这里修改配置
	fn(newCfg)

	persistPaths := updateOptions.persistPaths
	persistMetadata := false
	if updateOptions.envManagedPersistMode == EnvManagedPersistRuntimeEnvOnly && len(persistPaths) > 0 {
		runtimeOnlyPaths, metadataChanged, err := m.applyRuntimeEnvOnlyPersistPaths(newCfg, persistPaths)
		if err != nil {
			return err
		}
		persistPaths = removeConfigPaths(persistPaths, runtimeOnlyPaths)
		persistMetadata = metadataChanged
	}
	if updateOptions.envManagedPersistMode == EnvManagedPersistForceFile && len(persistPaths) > 0 {
		persistMetadata = addDisabledEnvOverridePaths(newCfg, persistPaths)
	}

	// 验证新配置
	// 确保修改后的配置仍然有效
	if err := newCfg.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	if persistMetadata {
		persistPaths = append(persistPaths, envOverrideDisabledPathsConfigPath)
	}
	if len(persistPaths) > 0 {
		if err := m.persistConfigUpdate(newCfg, persistPaths, updateOptions); err != nil {
			return err
		}
	}

	// 原子替换配置
	// 使用 atomic.Pointer.Store 确保并发安全
	m.config.Store(newCfg)

	// 通知所有注册的钩子
	// 让其他组件知道配置已更新
	m.notifyHooks(oldCfg, newCfg)

	return nil
}

// copyConfig 创建配置的深拷贝
// 为什么需要深拷贝:
//   - 避免修改原配置
//   - 支持回滚(如果验证失败)
//   - 线程安全
//
// 参数:
//
//	src: 源配置
//
// 返回:
//
//	*Config: 配置副本
func (m *manager) copyConfig(src *Config) *Config {
	dst := *src
	dst.I18n.Supported = append([]string(nil), src.I18n.Supported...)
	dst.I18n.Resources = copyStringMap(src.I18n.Resources)
	dst.Executor.Pools = append([]ExecutorPoolConfig(nil), src.Executor.Pools...)
	dst.CORS.AllowOrigins = append([]string(nil), src.CORS.AllowOrigins...)
	dst.CORS.AllowMethods = append([]string(nil), src.CORS.AllowMethods...)
	dst.CORS.AllowHeaders = append([]string(nil), src.CORS.AllowHeaders...)
	dst.CORS.ExposeHeaders = append([]string(nil), src.CORS.ExposeHeaders...)
	dst.Auth.Audience = append([]string(nil), src.Auth.Audience...)
	dst.System = copySystemConfig(src.System)
	dst.WebUI = copyWebUIConfig(src.WebUI)
	dst.Plugins = copyPluginsConfig(src.Plugins)
	dst.EnvOverride.DisabledPaths = append([]string(nil), src.EnvOverride.DisabledPaths...)
	return &dst
}

func copyStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

// RegisterHook 注册配置变更钩子
// 钩子会在配置重新加载时被调用
// 参数:
//
//	h: 钩子处理函数
//
// 线程安全:
//
//	使用写锁保护 hooks 列表
func (m *manager) RegisterHook(h HookHandler) {
	m.hooksMu.Lock()
	defer m.hooksMu.Unlock()
	m.hooks = append(m.hooks, h)
}

// notifyHooks 通知所有注册的钩子配置已变更
// 参数:
//
//	old: 旧配置
//	new: 新配置
//
// 线程安全:
//
//	使用读锁保护 hooks 列表
func (m *manager) notifyHooks(old, new *Config) {
	m.hooksMu.RLock()
	defer m.hooksMu.RUnlock()
	for _, h := range m.hooks {
		// 依次调用每个钩子
		h(old, new)
	}
}

// RegisterLogger 注册日志处理器并返回日志器
// 参数:
//
//	h: 日志处理器函数
//
// 返回:
//
//	logger.Logger: 日志器实例
func (m *manager) RegisterLogger(h LoggerHandler) logger.Logger {
	m.loggerHandler = h
	m.log = h()
	return m.log
}

// Watch 开始监听配置文件变化
// 使用 fsnotify 监听文件系统事件
// 当配置文件变化时,自动重新加载
// 返回:
//
//	error: 启动监听失败时的错误
//
// 注意:
//   - 必须先调用 Load
//   - 在后台运行,不会阻塞
func (m *manager) Watch() error {
	if m.configPath == "" {
		return fmt.Errorf("configuration not loaded, call Load first")
	}

	// 注册配置变更回调
	// 当文件变化时,viper 会调用这个函数
	m.v.OnConfigChange(func(e configloader.Event) {
		m.handleConfigChange(e)
	})

	// 开始监听配置文件
	// 这是一个非阻塞操作,在后台运行
	m.v.WatchConfig()
	return nil
}

// handleConfigChange 处理配置文件变化事件
// Shadow Loading 模式:
//  1. 使用临时 viper 实例加载新配置
//  2. 验证新配置
//  3. 如果验证通过,替换当前配置
//  4. 如果验证失败,保持当前配置不变
//
// 好处:
//   - 避免加载无效配置导致应用崩溃
//   - 原子替换,确保一致性
//
// 参数:
//
//	e: 文件系统事件
func (m *manager) handleConfigChange(e configloader.Event) {
	m.updateMu.Lock()
	defer m.updateMu.Unlock()

	if m.log != nil {
		m.log.Info("config file changed", "file", e.Name, "op", e.Op)
	}

	// Shadow loading: 加载到临时配置中
	// 使用新的 viper 实例,避免影响当前配置
	tempViper := configloader.New()
	tempViper.SetConfigFile(m.configPath)

	// 读取变更后的配置文件
	if err := tempViper.ReadInConfig(); err != nil {
		if m.log != nil {
			m.log.Error("failed to read changed config", "error", err)
		}
		return
	}

	// 处理环境变量替换
	m.processEnvSubstitutionForViper(tempViper)

	// 反序列化到临时配置
	newCfg := &Config{}
	if err := tempViper.Unmarshal(newCfg); err != nil {
		if m.log != nil {
			m.log.Error("failed to unmarshal changed config", "error", err)
		}
		return
	}

	LoadEnv()
	OverrideWithEnvExcept(newCfg, newCfg.EnvOverride.DisabledPaths)

	// 验证新配置
	// 如果验证失败,保持当前配置不变
	if err := newCfg.Validate(); err != nil {
		if m.log != nil {
			m.log.Error("changed config validation failed, keeping current config", "error", err)
		}
		return
	}

	// 获取旧配置用于钩子通知
	oldCfg := m.Get()
	if reflect.DeepEqual(oldCfg, newCfg) {
		if m.log != nil {
			m.log.Info("config file change produced no effective config change")
		}
		return
	}

	// 原子切换配置
	// 从这一刻起,Get() 会返回新配置
	m.config.Store(newCfg)

	// 通知所有钩子配置已更新
	m.notifyHooks(oldCfg, newCfg)

	if m.log != nil {
		m.log.Info("config reloaded successfully")
	}
}

// processEnvSubstitutionForViper 为指定的 viper 实例处理环境变量替换
// 参数:
//
//	v: viper 实例
func (m *manager) processEnvSubstitutionForViper(v *configloader.Loader) {
	envPattern := regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)
	settings := v.AllSettings()
	disabledPaths := disabledEnvOverridePathsFromSettings(settings)
	processed := m.processMap(settings, envPattern, "", disabledPaths)
	for key, value := range processed {
		v.Set(key, value)
	}
}

// GetConfigDir 返回配置文件所在的目录
// 用途:
//   - 加载相对于配置文件的其他文件
//   - 日志文件路径等
//
// 返回:
//
//	string: 配置文件目录路径
func (m *manager) GetConfigDir() string {
	if m.configPath == "" {
		return ""
	}
	return filepath.Dir(m.configPath)
}

// ResolveEnvValue 解析字符串中的环境变量语法
// 支持格式:
//
//	${VAR_NAME}          - 环境变量值
//	${VAR_NAME:default}  - 带默认值的环境变量
//
// 参数:
//
//	value: 要解析的字符串
//
// 返回:
//
//	string: 解析后的字符串
//
// 使用场景:
//   - 手动解析配置值
//   - 在代码中使用环境变量
func ResolveEnvValue(value string) string {
	envPattern := regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)
	return envPattern.ReplaceAllStringFunc(value, func(match string) string {
		submatches := envPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		envName := submatches[1]
		defaultValue := ""
		if len(submatches) >= 3 {
			defaultValue = submatches[2]
		}

		if envValue := os.Getenv(envName); envValue != "" {
			return envValue
		}
		return defaultValue
	})
}

// ParsePort 解析可能包含环境变量语法的端口值
// 参数:
//
//	value: 端口字符串(如 "8080" 或 "${PORT:8080}")
//
// 返回:
//
//	int: 解析后的端口号
//
// 使用示例:
//
//	port := config.ParsePort("${PORT:8080}")
func ParsePort(value string) int {
	resolved := ResolveEnvValue(value)
	resolved = strings.TrimSpace(resolved)
	var port int
	fmt.Sscanf(resolved, "%d", &port)
	return port
}

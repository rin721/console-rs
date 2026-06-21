// Package logger 提供统一的日志接口
// 封装底层日志实现(如 zap),提供一致的日志 API
// 设计目标:
// - 定义统一接口,屏蔽具体日志库的差异
// - 支持结构化日志,便于日志分析
// - 便于切换日志实现,无需修改业务代码
package logger

// 本文件属于日志基础设施，封装 zap 配置、输出目的地、字段绑定和运行时重载。

// Logger 定义统一的日志接口
// 这是一个抽象接口,具体实现在 zap.go 中
// 设计考虑:
// - 定义最常用的日志方法,保持简洁
// - 使用可变参数支持结构化日志
// - 支持日志上下文(With 方法)
// 使用示例:
//
//	log.Info("user created", "userId", 123, "username", "alice")
//	log.Error("database error", "error", err, "query", sql)
type Logger interface {
	// Debug 记录调试级别的日志
	// 用途:
	// - 详细的程序执行信息
	// - 变量值、函数调用等
	// - 仅在开发和调试时启用
	// 参数:
	//   msg: 日志消息
	//   keysAndValues: 可选的键值对,必须成对出现
	// 示例:
	//   log.Debug("processing request", "path", "/api/users", "method", "GET")
	Debug(msg string, keysAndValues ...interface{})

	// Info 记录信息级别的日志
	// 用途:
	// - 重要的程序事件
	// - 正常的业务流程
	// - 这是生产环境的默认日志级别
	// 参数:
	//   msg: 日志消息
	//   keysAndValues: 可选的键值对
	// 示例:
	//   log.Info("server started", "port", 8080, "mode", "production")
	//   log.Info("user logged in", "userId", 123, "ip", "192.168.1.1")
	Info(msg string, keysAndValues ...interface{})

	// Warn 记录警告级别的日志
	// 用途:
	// - 潜在的问题
	// - 不影响功能但需要注意的情况
	// - 即将发生的错误
	// 参数:
	//   msg: 日志消息
	//   keysAndValues: 可选的键值对
	// 示例:
	//   log.Warn("cache miss", "key", "user:123")
	//   log.Warn("API rate limit approaching", "current", 950, "limit", 1000)
	Warn(msg string, keysAndValues ...interface{})

	// Error 记录错误级别的日志
	// 用途:
	// - 错误情况,但程序可以继续运行
	// - 需要人工介入的问题
	// - 应该告警的错误
	// 参数:
	//   msg: 日志消息
	//   keysAndValues: 可选的键值对
	// 示例:
	//   log.Error("database query failed", "error", err, "sql", query)
	//   log.Error("external API error", "service", "payment", "statusCode", 500)
	Error(msg string, keysAndValues ...interface{})

	// Fatal 记录致命错误并退出程序
	// 用途:
	// - 无法恢复的严重错误
	// - 程序无法继续运行的情况
	// 警告:
	//   会调用 os.Exit(1) 终止程序
	//   应该谨慎使用,仅用于初始化等场景
	// 参数:
	//   msg: 日志消息
	//   keysAndValues: 可选的键值对
	// 示例:
	//   log.Fatal("failed to load config", "error", err)
	//   log.Fatal("required service unavailable", "service", "database")
	Fatal(msg string, keysAndValues ...interface{})

	// With 返回一个新的 Logger,添加了给定的键值对到上下文
	// 用途:
	// - 为一组日志添加公共字段
	// - 避免重复传递相同的键值对
	// 参数:
	//   keysAndValues: 要添加到上下文的键值对
	// 返回:
	//   Logger: 新的 Logger 实例,包含上下文信息
	// 使用示例:
	//   userLog := log.With("userId", 123, "username", "alice")
	//   userLog.Info("login successful")        // 自动包含 userId 和 username
	//   userLog.Error("update failed", "error", err) // 也包含 userId 和 username
	// 好处:
	//   - 代码更简洁
	//   - 确保相关日志都包含上下文信息
	//   - 便于日志检索和分析
	With(keysAndValues ...interface{}) Logger

	// Sync 刷新缓冲的日志条目
	// 用途:
	// - 确保所有日志都写入磁盘
	// - 应该在程序退出前调用
	// 返回:
	//   error: 刷新失败时的错误
	// 使用场景:
	//   defer log.Sync() // 在 main 函数中
	//   log.Sync()       // 在优雅关闭时
	// 为什么需要:
	//   日志可能被缓冲以提高性能
	//   如果程序突然退出,缓冲的日志可能丢失
	//   Sync 确保所有日志都被写入
	Sync() error

	// Reloader 嵌入重载接口
	// 使 Logger 支持运行时配置热更新
	Reloader
}

// Reloader 定义日志配置重载接口
// 允许在运行时热更新日志配置,无需重启应用
// 使用场景:
//   - 配置文件变更时自动重载
//   - 动态调整日志级别
//   - 切换日志输出目标
type Reloader interface {
	// Reload 使用新配置重新加载日志系统
	// 这是一个原子操作,失败时保持原配置不变
	// 参数:
	//   cfg: 新的日志配置
	// 返回:
	//   error: 重载失败时的错误
	// 并发安全:
	//   - 使用读写锁保护,确保重载过程原子性
	//   - 失败时保持原有 logger 不变
	//   - 新 logger 创建成功后才替换旧 logger
	Reload(cfg *Config) error
}

// Config 保存日志配置
// 包含日志库初始化所需的所有参数
// 这些配置通常从配置文件加载
type Config struct {
	// Level 最低日志级别
	// 可选值: debug, info, warn, error
	// 只有 >= 此级别的日志会被记录
	// 例如:如果设置为 info,debug 日志不会输出
	// 开发环境推荐: debug
	// 生产环境推荐: info 或 warn
	Level string

	// Format 默认输出格式(用于所有输出)
	// 可选值:
	// - json: 结构化 JSON 格式,便于日志系统解析
	// - console: 人类可读的控制台格式,便于开发调试
	// 如果设置了 ConsoleFormat 或 FileFormat,则此字段作为后备默认值
	// 生产环境推荐: json(便于 ELK、Splunk 等系统分析)
	// 开发环境推荐: console(易读)
	Format string

	// ConsoleFormat 控制台输出专用格式
	// 可选值: json, console
	// 如果为空,则使用 Format 的值
	// 适用场景: 想要控制台用 console 格式,文件用 json 格式
	ConsoleFormat string

	// FileFormat 文件输出专用格式
	// 可选值: json, console
	// 如果为空,则使用 Format 的值
	// 适用场景: 想要控制台用 console 格式,文件用 json 格式
	FileFormat string

	// Output 输出目标
	// 可选值:
	// - stdout: 仅标准输出,适合容器环境(日志收集器会捕获)
	// - file: 仅文件输出,需要配合 FilePath,适合传统部署
	// - both: 同时输出到文件和标准输出,适合开发环境
	// 推荐:
	// - 容器/K8s 环境: stdout
	// - 传统部署: file
	// - 开发环境: both
	Output string

	// FilePath 日志文件路径
	// 仅当 Output="file" 或 Output="both" 时有效
	// 例如: /var/log/app/app.log
	// 注意:
	// - 确保目录存在且有写权限
	// - 建议使用绝对路径
	FilePath string

	// MaxSize 单个日志文件的最大大小(MB)
	// 超过此大小会触发日志轮转
	// 推荐值: 100-500 MB
	// 设置过大:单个文件难以处理
	// 设置过小:文件过多
	MaxSize int

	// MaxBackups 保留的旧日志文件最大数量
	// 超过此数量的旧文件会被删除
	// 推荐值: 3-10
	// 用途:
	// - 防止日志占满磁盘
	// - 保留足够的历史日志用于问题排查
	MaxBackups int

	// MaxAge 保留旧日志文件的最大天数
	// 超过此天数的日志文件会被删除
	// 推荐值: 7-30 天
	// 考虑因素:
	// - 法规要求(某些行业要求保留审计日志)
	// - 磁盘空间
	// - 问题排查需求
	MaxAge int
}

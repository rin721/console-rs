package logger

// 本文件属于日志基础设施，封装 zap 配置、输出目的地、字段绑定和运行时重载。

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 编译时检查 zapLogger 是否实现了 Logger 接口
// 这是 Go 中的一个常见模式,确保类型安全
// 如果 zapLogger 没有实现 Logger 的所有方法,编译会失败
// _ 表示我们不使用这个变量,只是做类型检查
var _ Logger = (*zapLogger)(nil)

// zapLogger 封装 zap.SugaredLogger 实现 Logger 接口
// 为什么选择 zap:
// - 性能优秀:比标准库 log 快 10 倍以上
// - 零内存分配:减少 GC 压力
// - 结构化日志:支持键值对,便于日志分析
// - 功能完善:支持日志级别、格式化、轮转等
// 为什么使用 SugaredLogger:
// - 更友好的 API,支持可变参数
// - 性能略低于 Logger,但仍然很快
// - 适合大多数场景
type zapLogger struct {
	// mu 读写锁,保护并发访问
	// 读操作(日志记录)使用读锁,允许并发
	// 写操作(Reload)使用写锁,独占访问
	mu sync.RWMutex

	// sugar zap 的 SugaredLogger
	// SugaredLogger 提供了类似 fmt.Printf 的 API
	// 相比原始的 Logger,牺牲一点性能换取更好的易用性
	sugar *zap.SugaredLogger

	// config 保存配置用于 Reload 时对比
	// 也用于确保重载时使用正确的配置
	config *Config
}

// New 基于提供的配置创建一个新的 Logger 实例
// 这是工厂函数,根据配置初始化日志系统
// 参数:
//
//	cfg: 日志配置,包含级别、格式、输出目标等
//
// 返回:
//
//	Logger: 日志接口
//	error: 创建失败时的错误(当前实现总是成功)
//
// 配置过程:
//  1. 解析日志级别(debug/info/warn/error)
//  2. 根据输出模式构建 Core:
//     - stdout: 使用控制台格式
//     - file: 使用文件格式
//     - both: 分别为控制台和文件创建 Core,然后合并
//  3. 创建 zap Logger
//  4. 包装为 SugaredLogger
func New(cfg *Config) (Logger, error) {
	// 1. 解析日志级别
	level := zapParseLevel(parseLevel(cfg.Level))

	output := strings.ToLower(cfg.Output)

	// 2. 根据输出模式构建 Core
	var core zapcore.Core

	switch output {
	case OutputStdout:
		// 仅控制台输出
		consoleFormat := getConsoleFormat(cfg)
		encoder := buildEncoder(consoleFormat)
		writer := zapcore.AddSync(os.Stdout)
		core = zapcore.NewCore(encoder, writer, level)

	case OutputFile:
		// 仅文件输出
		fileFormat := getFileFormat(cfg)
		encoder := buildEncoder(fileFormat)
		writer := buildFileWriter(cfg)
		core = zapcore.NewCore(encoder, writer, level)

	case OutputBoth:
		// 同时输出到控制台和文件,可以使用不同格式
		consoleFormat := getConsoleFormat(cfg)
		fileFormat := getFileFormat(cfg)

		// 为控制台创建 Core
		consoleEncoder := buildEncoder(consoleFormat)
		consoleWriter := zapcore.AddSync(os.Stdout)
		consoleCore := zapcore.NewCore(consoleEncoder, consoleWriter, level)

		// 为文件创建 Core
		fileEncoder := buildEncoder(fileFormat)
		fileWriter := buildFileWriter(cfg)
		fileCore := zapcore.NewCore(fileEncoder, fileWriter, level)

		// 使用 Tee 合并多个 Core
		core = zapcore.NewTee(consoleCore, fileCore)

	default:
		// 未知模式,降级到控制台输出
		consoleFormat := getConsoleFormat(cfg)
		encoder := buildEncoder(consoleFormat)
		writer := zapcore.AddSync(os.Stdout)
		core = zapcore.NewCore(encoder, writer, level)
	}

	// 3. 创建 Logger
	// zap.AddCaller(): 记录调用者信息(文件名和行号)
	// zap.AddCallerSkip(1): 跳过 1 层调用栈
	//   因为我们封装了一层,需要跳过才能显示真实调用者
	zapLog := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	// 4. 返回 SugaredLogger
	return &zapLogger{
		sugar:  zapLog.Sugar(),
		config: cfg,
	}, nil
}

// Default 返回一个默认的 Logger
// 默认配置:
// - 级别: info
// - 格式: console(易读)
// - 输出: stdout
// 使用场景:
// - 快速开始,不需要配置
// - 测试环境
// - 简单应用
func Default() Logger {
	cfg := &Config{
		Level:  DefaultLevel,  // 默认 debug 级别
		Format: DefaultFormat, // 控制台格式,易读
		Output: DefaultOutput, // 输出到标准输出
	}
	// 忽略错误,因为默认配置不会失败
	log, _ := New(cfg)
	return log
}

// parseLevel 解析日志级别字符串
// 将人类可读的字符串转换为 zapcore.Level
// 参数:
//
//	level: 日志级别字符串(不区分大小写)
//
// 返回:
//
//	zapcore.Level: zap 的日志级别
//
// 支持的级别:
//
//	debug: 最详细,包含调试信息
//	info: 正常信息,生产环境默认
//	warn/warning: 警告信息
//	error: 错误信息
//	默认: info(如果输入无效)
func zapParseLevel(level Level) zapcore.Level {
	// ToLower 确保不区分大小写
	switch level {
	case LevelDebug:
		// 调试级别,最详细
		// 包含所有日志(debug, info, warn, error)
		return zapcore.DebugLevel
	case LevelInfo:
		// 信息级别,生产环境默认
		// 包含 info, warn, error
		return zapcore.InfoLevel
	case LevelWarn:
		// 警告级别
		// 包含 warn, error
		// 支持两种拼写
		return zapcore.WarnLevel
	case LevelError:
		// 错误级别,只记录错误
		return zapcore.ErrorLevel
	default:
		// 默认使用 info 级别
		// 这是一个安全的默认值
		return zapcore.InfoLevel
	}
}

// buildEncoder 构建日志编码器
// 编码器决定了日志的输出格式
// 参数:
//
//	format: 格式类型("json" 或 "console")
//
// 返回:
//
//	zapcore.Encoder: zap 编码器
//
// 两种格式的选择:
//   - JSON: 结构化,便于机器解析和日志系统(ELK)
//   - Console: 易读,便于人类阅读和开发调试
func buildEncoder(format string) zapcore.Encoder {
	// 配置编码器
	// 定义日志中各个字段的键名和格式
	encoderConfig := zapcore.EncoderConfig{
		// TimeKey: 时间字段的键名
		TimeKey: "time",

		// LevelKey: 日志级别字段的键名
		LevelKey: "level",

		// NameKey: logger 名称字段的键名
		NameKey: "logger",

		// CallerKey: 调用者信息字段的键名
		// 包含文件名和行号,如 "main.go:42"
		CallerKey: "caller",

		// MessageKey: 日志消息字段的键名
		MessageKey: "message",

		// StacktraceKey: 堆栈跟踪字段的键名
		// 只在 error 级别及以上时添加
		StacktraceKey: "stacktrace",

		// LineEnding: 每条日志的结尾字符
		// 默认是 "\n"
		LineEnding: zapcore.DefaultLineEnding,

		// EncodeLevel: 日志级别的编码方式
		// LowercaseLevelEncoder: 小写(info, warn, error)
		// 其他选项: UppercaseLevelEncoder, CapitalLevelEncoder
		EncodeLevel: zapcore.LowercaseLevelEncoder,

		// EncodeTime: 时间的编码方式
		// ISO8601TimeEncoder: ISO 8601 格式 "2006-01-02T15:04:05.000Z0700"
		// 其他选项: RFC3339TimeEncoder, EpochTimeEncoder
		EncodeTime: zapcore.ISO8601TimeEncoder,

		// EncodeDuration: 时间间隔的编码方式
		// SecondsDurationEncoder: 以秒为单位
		EncodeDuration: zapcore.SecondsDurationEncoder,

		// EncodeCaller: 调用者信息的编码方式
		// ShortCallerEncoder: 短格式 "main.go:42"
		// FullCallerEncoder: 完整路径 "/path/to/main.go:42"
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	// 根据配置选择编码器类型
	if strings.ToLower(format) == "json" {
		// JSON 编码器
		// 每条日志是一个 JSON 对象
		// 例如: {"level":"info","time":"2024-01-01T12:00:00Z","msg":"hello"}
		// 优点: 结构化,易于解析和搜索
		// 适用: 生产环境,日志收集系统
		return zapcore.NewJSONEncoder(encoderConfig)
	}
	// Console 编码器
	// 更易读的格式
	// 例如: 2024-01-01T12:00:00Z	info	hello
	// 优点: 人类易读
	// 适用: 开发环境,控制台输出
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// getConsoleFormat 获取控制台输出的格式
// 优先使用 ConsoleFormat,如果未设置则使用 Format
// 参数:
//
//	cfg: 日志配置
//
// 返回:
//
//	string: 格式名称
func getConsoleFormat(cfg *Config) string {
	if cfg.ConsoleFormat != "" {
		return cfg.ConsoleFormat
	}
	if cfg.Format != "" {
		return cfg.Format
	}
	return DefaultFormat
}

// getFileFormat 获取文件输出的格式
// 优先使用 FileFormat,如果未设置则使用 Format
// 参数:
//
//	cfg: 日志配置
//
// 返回:
//
//	string: 格式名称
func getFileFormat(cfg *Config) string {
	if cfg.FileFormat != "" {
		return cfg.FileFormat
	}
	if cfg.Format != "" {
		return cfg.Format
	}
	return DefaultFormat
}

// buildFileWriter 构建文件输出写入器
// 使用 lumberjack 进行日志轮转
// 参数:
//
//	cfg: 日志配置
//
// 返回:
//
//	zapcore.WriteSyncer: 文件写入同步器
func buildFileWriter(cfg *Config) zapcore.WriteSyncer {
	if cfg.FilePath == "" {
		// 如果没有配置文件路径,降级到 stdout
		return zapcore.AddSync(os.Stdout)
	}

	lj := &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   true,
	}
	return zapcore.AddSync(lj)
}

// buildWriteSyncer 构建日志输出目标
// 决定日志写入到哪里(文件或标准输出或两者)
// 参数:
//
//	cfg: 日志配置
//
// 返回:
//
//	zapcore.WriteSyncer: zap 输出同步器
//
// 三种输出方式:
//   - stdout: 仅标准输出,实时查看,容器友好
//   - file: 仅文件,持久化,支持轮转
//   - both: 同时输出到文件和标准输出
func buildWriteSyncer(cfg *Config) zapcore.WriteSyncer {
	output := strings.ToLower(cfg.Output)

	// 创建文件输出同步器(如果需要)
	var fileWriter zapcore.WriteSyncer
	if (output == OutputFile || output == OutputBoth) && cfg.FilePath != "" {
		// 使用 lumberjack 进行日志轮转
		// lumberjack 是一个流行的 Go 日志轮转库
		// 功能:
		// - 按文件大小轮转
		// - 保留指定数量的旧文件
		// - 保留指定天数的旧文件
		// - 自动压缩旧文件
		lj := &lumberjack.Logger{
			// Filename: 日志文件路径
			Filename: cfg.FilePath,

			// MaxSize: 单个文件最大大小(MB)
			// 超过此大小会创建新文件
			MaxSize: cfg.MaxSize,

			// MaxBackups: 保留的旧文件数量
			// 超过会删除最旧的
			MaxBackups: cfg.MaxBackups,

			// MaxAge: 保留旧文件的最大天数
			// 超过会删除
			MaxAge: cfg.MaxAge,

			// Compress: 是否压缩旧文件
			// true: 使用 gzip 压缩,节省磁盘空间
			// false: 不压缩,便于直接查看
			Compress: true,
		}
		fileWriter = zapcore.AddSync(lj)
	}

	// 根据输出模式返回相应的 WriteSyncer
	switch output {
	case OutputFile:
		// 仅输出到文件
		if fileWriter != nil {
			return fileWriter
		}
		// 如果文件配置无效,降级到 stdout
		return zapcore.AddSync(os.Stdout)

	case OutputBoth:
		// 同时输出到文件和控制台
		if fileWriter != nil {
			// 使用 NewMultiWriteSyncer 同时写入多个目标
			return zapcore.NewMultiWriteSyncer(
				fileWriter,
				zapcore.AddSync(os.Stdout),
			)
		}
		// 如果文件配置无效,降级到 stdout
		return zapcore.AddSync(os.Stdout)

	default:
		// stdout 或其他未知值,默认输出到标准输出
		// 适合:
		// - 开发环境(直接在终端查看)
		// - 容器环境(日志收集器会捕获 stdout)
		// - K8s 环境(kubectl logs)
		return zapcore.AddSync(os.Stdout)
	}
}

// Debug 记录调试级别的日志
// 实现 Logger 接口
// 使用读锁保护,允许并发日志记录
func (l *zapLogger) Debug(msg string, keysAndValues ...interface{}) {
	// Debugw: "w" 表示 "with",支持键值对参数
	// 例如: Debug("processing", "userId", 123, "action", "login")
	l.mu.RLock()
	sugar := l.sugar
	l.mu.RUnlock()
	sugar.Debugw(msg, keysAndValues...)
}

// Info 记录信息级别的日志
// 实现 Logger 接口
// 使用读锁保护,允许并发日志记录
func (l *zapLogger) Info(msg string, keysAndValues ...interface{}) {
	l.mu.RLock()
	sugar := l.sugar
	l.mu.RUnlock()
	sugar.Infow(msg, keysAndValues...)
}

// Warn 记录警告级别的日志
// 实现 Logger 接口
// 使用读锁保护,允许并发日志记录
func (l *zapLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.mu.RLock()
	sugar := l.sugar
	l.mu.RUnlock()
	sugar.Warnw(msg, keysAndValues...)
}

// Error 记录错误级别的日志
// 实现 Logger 接口
// 使用读锁保护,允许并发日志记录
func (l *zapLogger) Error(msg string, keysAndValues ...interface{}) {
	l.mu.RLock()
	sugar := l.sugar
	l.mu.RUnlock()
	sugar.Errorw(msg, keysAndValues...)
}

// Fatal 记录致命错误并退出程序
// 实现 Logger 接口
// 警告: 会调用 os.Exit(1),终止程序
// 使用读锁保护,允许并发日志记录
func (l *zapLogger) Fatal(msg string, keysAndValues ...interface{}) {
	// Fatalw 会先记录日志,然后调用 os.Exit(1)
	// 使用场景: 无法恢复的严重错误
	l.mu.RLock()
	sugar := l.sugar
	l.mu.RUnlock()
	sugar.Fatalw(msg, keysAndValues...)
}

// With 返回一个新的 Logger,添加了给定的键值对到上下文
// 实现 Logger 接口
// 这是一个非常有用的功能,可以创建带上下文的子 logger
// 参数:
//
//	keysAndValues: 要添加到所有日志的键值对
//
// 返回:
//
//	Logger: 新的 Logger 实例
//
// 使用示例:
//
//	requestLog := logger.With("requestId", "abc123", "userId", 456)
//	requestLog.Info("processing request")  // 自动包含 requestId 和 userId
//	requestLog.Error("request failed")     // 也包含 requestId 和 userId
//
// 好处:
//   - 避免在每条日志中重复传递相同的字段
//   - 自动为一组相关日志添加上下文
//   - 便于日志检索和分析
func (l *zapLogger) With(keysAndValues ...interface{}) Logger {
	// sugar.With 创建一个新的 SugaredLogger
	// 不会修改原始 logger,而是返回新实例
	l.mu.RLock()
	sugar := l.sugar
	config := l.config
	l.mu.RUnlock()
	return &zapLogger{
		sugar:  sugar.With(keysAndValues...),
		config: config,
	}
}

// Sync 刷新缓冲的日志条目
// 实现 Logger 接口
// 支持异步模式：如果设置了executor，使用协程池异步刷新
// 为什么需要 Sync:
// - zap 为了性能会缓冲日志
// - 如果程序突然退出,缓冲的日志可能丢失
// - Sync 确保所有日志都写入磁盘
// 使用场景:
//
//	defer logger.Sync()  // 在 main 函数中
//	logger.Sync()        // 在优雅关闭时
//
// 返回:
//
//	error: 刷新失败时的错误（同步模式）
//	       nil（异步模式，错误会输出到stderr）
func (l *zapLogger) Sync() error {
	l.mu.RLock()
	sugar := l.sugar
	l.mu.RUnlock()

	// 没有executor时同步执行
	return sugar.Sync()
}

// Reload 使用新配置重新加载日志系统
// 实现 Reloader 接口
// 这个方法允许在运行时热更新日志配置,无需重启应用
// 使用场景:
//   - 配置文件变更时自动重载
//   - 动态调整日志级别
//   - 切换日志输出目标
//
// 参数:
//
//	cfg: 新的日志配置
//
// 返回:
//
//	error: 重载失败时的错误
//
// 并发安全:
//   - 使用写锁保护,确保重载过程原子性
//   - 失败时保持原有 logger 不变
//   - 新 logger 创建成功后才替换旧 logger
func (l *zapLogger) Reload(cfg *Config) error {
	// 1. 在锁外创建新的 logger 实例
	// 避免长时间持有锁,提高并发性能
	newLogger, err := New(cfg)
	if err != nil {
		return fmt.Errorf(ErrMsgReloadFailed, err)
	}

	// 2. 获取写锁,开始原子替换操作
	// 写锁确保:
	// - 没有其他 goroutine 正在读取 sugar
	// - 没有其他 goroutine 正在执行 Reload
	l.mu.Lock()

	// 保存旧 sugar 的引用,用于后续同步
	oldSugar := l.sugar

	// 3. 原子地替换 logger 实例
	// 将新 logger 的内部字段复制到当前实例
	// 这样外部持有的 Logger 接口引用仍然有效
	newZapLogger := newLogger.(*zapLogger)
	l.sugar = newZapLogger.sugar
	l.config = cfg

	// 4. 释放写锁
	// 新 logger 已替换完成,其他 goroutine 可以使用新 logger
	l.mu.Unlock()

	// 5. 同步旧 logger
	// 在锁外执行,避免长时间持有锁
	// 确保旧 logger 的缓冲日志被刷新到磁盘
	if oldSugar != nil {
		// 忽略 Sync 错误,因为旧 logger 已被替换
		// 某些平台上 Sync 可能返回无害的错误
		_ = oldSugar.Sync()
	}

	return nil
}

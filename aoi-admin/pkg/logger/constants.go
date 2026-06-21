package logger

// 本文件属于日志基础设施，集中封装日志级别、输出模式和重载日志消息常量。

// Level 表示日志严重级别的内部枚举，与 zapcore.Level 在构建配置时完成映射。
type Level int8

// Level 定义日志级别
const (
	LevelDebug Level = iota - 1
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// LevelNames 将配置文件中的字符串级别映射为内部 Level，用于校验和默认化。
var LevelNames = map[string]Level{
	"debug": LevelDebug,
	"info":  LevelInfo,
	"warn":  LevelWarn,
	"error": LevelError,
	"fatal": LevelFatal,
}

// parseLevel 将字符串日志级别解析为内部枚举，未知值回退到 info 以保持日志可用。
func parseLevel(l string) Level {
	for k, v := range LevelNames {
		if k == l {
			return v
		}
	}
	return LevelInfo
}

const (
	// DefaultLevel 定义未显式配置时的日志级别，适合开发期观察脚手架行为。
	DefaultLevel = "debug"
	// DefaultFormat 定义未显式配置时的日志编码格式。
	DefaultFormat = "console"
	// DefaultOutput 定义未显式配置时的日志输出目的地。
	DefaultOutput = "stdout"

	// 日志输出模式用于控制日志写入控制台、文件或两者同时写入。
	OutputStdout = "stdout" // 仅输出到标准输出（控制台）
	// OutputFile 表示日志只写入滚动文件。
	OutputFile = "file" // 仅输出到文件
	// OutputBoth 表示日志同时写入控制台和滚动文件。
	OutputBoth = "both" // 同时输出到文件和控制台

	// MsgLoggerReloading 日志重载中消息
	MsgLoggerReloading = "reloading logger configuration"

	// MsgLoggerReloaded 日志重载成功消息
	MsgLoggerReloaded = "logger configuration reloaded successfully"

	// ErrMsgReloadFailed 重载失败的错误消息
	ErrMsgReloadFailed = "failed to reload logger: %w"
)

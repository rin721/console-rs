package config

// 本文件定义一个配置分区及其校验规则，是外部配置进入运行时基础设施前的类型化边界。

import "errors"

// Config 保存日志配置
// 包含日志库初始化所需的所有参数
// 这些配置通常从配置文件加载
type LoggerConfig struct {
	// Level 最低日志级别
	// 可选值: debug, info, warn, error
	// 只有 >= 此级别的日志会被记录
	// 例如:如果设置为 info,debug 日志不会输出
	// 开发环境推荐: debug
	// 生产环境推荐: info 或 warn
	Level string `mapstructure:"level" envname:"LOG_LEVEL"`

	// Format 默认输出格式(用于所有输出)
	// 可选值:
	// - json: 结构化 JSON 格式,便于日志系统解析
	// - console: 人类可读的控制台格式,便于开发调试
	// 如果设置了 ConsoleFormat 或 FileFormat,则此字段作为后备默认值
	// 生产环境推荐: json(便于 ELK、Splunk 等系统分析)
	// 开发环境推荐: console(易读)
	Format string `mapstructure:"format" envname:"LOG_FORMAT"`

	// ConsoleFormat 控制台输出专用格式(可选)
	// 可选值: json, console
	// 如果为空,则使用 Format 的值
	// 使用场景: 希望控制台用易读的 console 格式,文件用 json 格式
	ConsoleFormat string `mapstructure:"console_format" envname:"LOG_CONSOLE_FORMAT"`

	// FileFormat 文件输出专用格式(可选)
	// 可选值: json, console
	// 如果为空,则使用 Format 的值
	// 使用场景: 希望控制台用 console 格式,文件用 json 格式
	FileFormat string `mapstructure:"file_format" envname:"LOG_FILE_FORMAT"`

	// Output 输出目标
	// 可选值:
	// - stdout: 仅标准输出,适合容器环境(日志收集器会捕获)
	// - file: 仅文件输出,需要配合 FilePath,适合传统部署
	// - both: 同时输出到文件和标准输出,适合开发环境
	// 推荐:
	// - 容器/K8s 环境: stdout
	// - 传统部署: file
	// - 开发环境: both
	Output string `mapstructure:"output" envname:"LOG_OUTPUT"`

	// FilePath 日志文件路径
	// 仅当 Output="file" 或 Output="both" 时有效
	// 例如: /var/log/app/app.log
	// 注意:
	// - 确保目录存在且有写权限
	// - 建议使用绝对路径
	FilePath string `mapstructure:"file_path" envname:"LOG_FILE_PATH"`

	// MaxSize 单个日志文件的最大大小(MB)
	// 超过此大小会触发日志轮转
	// 推荐值: 100-500 MB
	// 设置过大:单个文件难以处理
	// 设置过小:文件过多
	MaxSize int `mapstructure:"max_size" envname:"LOG_MAX_SIZE"`

	// MaxBackups 保留的旧日志文件最大数量
	// 超过此数量的旧文件会被删除
	// 推荐值: 3-10
	// 用途:
	// - 防止日志占满磁盘
	// - 保留足够的历史日志用于问题排查
	MaxBackups int `mapstructure:"max_backups" envname:"LOG_MAX_BACKUPS"`

	// MaxAge 保留旧日志文件的最大天数
	// 超过此天数的日志文件会被删除
	// 推荐值: 7-30 天
	// 考虑因素:
	// - 法规要求(某些行业要求保留审计日志)
	// - 磁盘空间
	// - 问题排查需求
	MaxAge int `mapstructure:"max_age" envname:"LOG_MAX_AGE"`
}

// ValidateName 返回当前配置分区在聚合校验错误中的稳定名称。
func (c *LoggerConfig) ValidateName() string {
	return AppLoggerName
}

// ValidateRequired 声明当前配置分区是否必须出现在完整应用配置中。
func (c *LoggerConfig) ValidateRequired() bool {
	return true
}

// Validate 验证日志配置
// 实现 Configurable 接口
func (c *LoggerConfig) Validate() error {
	// 验证日志级别
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Level] {
		return errors.New("level must be debug, info, warn, or error")
	}

	// 验证日志格式
	validFormats := map[string]bool{"json": true, "console": true}
	if !validFormats[c.Format] {
		return errors.New("format must be json or console")
	}

	// 验证输出目标
	validOutputs := map[string]bool{"stdout": true, "file": true, "both": true}
	if !validOutputs[c.Output] {
		return errors.New("output must be stdout, file, or both")
	}

	return nil
}

// overrideLoggerConfig 使用环境变量覆盖日志配置
func overrideLoggerConfig(cfg *LoggerConfig) {
	overrideConfigFromEnv(cfg)
}

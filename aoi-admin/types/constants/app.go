package constants

// 本文件定义跨层共享常量，避免内部基础设施包反向污染公共 types 边界。

import "time"

const (
	// AppDefaultConfigPath 定义未显式传参时应用读取的默认配置文件路径。
	AppDefaultConfigPath = "configs/config.yaml"
	// AppShutdownTimeout 定义进程收到退出信号后的优雅关闭最长等待时间。
	AppShutdownTimeout = 30 * time.Second
	// EnvConfigPathName 定义覆盖配置文件路径的环境变量名称。
	EnvConfigPathName = "RIN_CONFIG_PATH"

	// AppPrefix 定义运行时环境变量前缀，配置包会基于它构造动态覆盖键。
	AppPrefix = "Rin"
	// AppServerCommandName 定义启动服务子命令的稳定名称。
	AppServerCommandName = "server"
)

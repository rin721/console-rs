package config

// 本文件属于配置子系统，处理配置加载、环境变量覆盖、运行时快照或跨分区校验。

import "github.com/rei0721/go-scaffold/pkg/configloader"

// LoadEnv 加载 .env 文件
// .env 文件是可选的,如果不存在不会报错
// 这个函数应该在加载 config.yaml 之前调用
//
// 工作流程:
//  1. 尝试加载项目根目录的 .env 文件
//  2. 如果文件不存在,静默跳过
//  3. 如果文件存在但格式错误,记录错误但不中断
//
// 使用场景:
//   - 本地开发: 创建 .env 文件存放敏感配置
//   - 生产环境: 不使用 .env 文件,直接使用系统环境变量
//
// 注意事项:
//   - .env 文件不应该提交到 Git
//   - .env 文件中的变量会被加载到进程环境变量中
//   - 如果系统环境变量已存在同名变量,.env 文件的值会被忽略
func LoadEnv() {
	configloader.LoadEnv(EnvFilePath)
}

// OverrideWithEnv 使用环境变量覆盖配置
// 优先级: 环境变量 > config.yaml
//
// 参数:
//
//	cfg: 从 config.yaml 加载的配置
//
// 工作流程:
//  1. 检查每个支持的环境变量
//  2. 如果环境变量存在,使用其值覆盖配置
//  3. 如果环境变量不存在,保持 config.yaml 的值
//
// 使用示例:
//
//	config := loadFromYaml()
//	OverrideWithEnv(config)
//	// 此时 config 中的值可能已被环境变量覆盖
func OverrideWithEnv(cfg *Config) {
	OverrideWithEnvExcept(cfg, nil)
}

// OverrideWithEnvExcept 使用环境变量覆盖配置，但跳过指定的 mapstructure 路径。
func OverrideWithEnvExcept(cfg *Config, disabledPaths []string) {
	if cfg == nil {
		return
	}
	overrideConfigFromEnvExcept(cfg, disabledPaths)
}

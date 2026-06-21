package yaml2go

// 本文件属于 YAML 配置代码生成器，把配置样例解析为可编译的 Go 结构体、标签和辅助方法。

const (
	// DefaultPackageName 默认包名
	DefaultPackageName = "main"

	// DefaultStructName 默认根结构体名
	DefaultStructName = "Config"

	// DefaultIndentStyle 默认缩进风格（使用 tab）
	DefaultIndentStyle = "tab"

	// IndentStyleTab tab 缩进
	IndentStyleTab = "tab"

	// IndentStyleSpace 空格缩进
	IndentStyleSpace = "space"

	// ConfigBlockFilenameSuffix 配置块文件名后缀
	ConfigBlockFilenameSuffix = "_config.go"
)

var (
	// DefaultTags 默认生成的标签列表
	// - json: JSON 序列化
	// - yaml: YAML 序列化
	// - mapstructure: Viper 配置读取
	// - toml: TOML 序列化
	DefaultTags = []string{"json", "yaml", "mapstructure", "toml"}
)

package yaml2go

// 本文件属于 YAML 配置代码生成器，把配置样例解析为可编译的 Go 结构体、标签和辅助方法。

// Converter YAML 转 Go 结构体转换器接口
// 提供 YAML 字符串到 Go 结构体代码的转换功能
//
// 为什么使用接口:
//   - 定义契约: 明确转换器提供的能力
//   - 依赖倒置: 使用方依赖接口而非实现
//   - 便于测试: 可以创建 mock 实现进行单元测试
//   - 解耦: 可以轻松替换不同的实现方式
//
// 注意: 此工具只做转换，不做文件操作
//   - 用户拿到 GenerateResult 后自行决定如何使用（打印、写文件等）
type Converter interface {
	// Convert 转换 YAML 字符串为多个配置代码
	// 参数:
	//   yamlStr: YAML 格式的字符串
	// 返回:
	//   *GenerateResult: 生成结果，包含主配置和所有子配置的代码
	//   error: 转换失败时的错误
	// 业务流程:
	//   1. 解析 YAML 字符串
	//   2. 拆分顶级配置节点
	//   3. 为每个节点生成独立的配置代码
	//   4. 生成主配置代码组合所有子配置
	//   5. 为每个配置生成接口方法（ValidateName, Validate, DefaultConfig, OverrideConfig）
	Convert(yamlStr string) (*GenerateResult, error)

	// SetConfig 更新配置（支持热更新）
	// 参数:
	//   config: 新的配置对象
	// 返回:
	//   error: 配置无效时返回错误
	// 注意:
	//   - 配置会立即生效
	//   - nil 配置会使用默认值
	SetConfig(config *Config) error
}

// Config 转换器配置
// 用于自定义代码生成行为
type Config struct {
	// PackageName 包名
	// 默认: "main"
	// 示例: "config", "model"
	PackageName string

	// StructName 根结构体名称
	// 默认: "Config"
	// 示例: "AppConfig", "Settings"
	StructName string

	// Tags 需要生成的标签列表
	// 默认: ["json", "yaml", "mapstructure", "toml"]
	// 可选值: "json", "yaml", "xml", "mapstructure", "toml", "validate"
	// 注意: mapstructure 是 viper 使用的标签
	Tags []string

	// UsePointer 字段是否使用指针类型
	// true: 字段类型为 *string, *int64 等
	// false: 字段类型为 string, int64 等
	// 默认: false
	// 指针的优势:
	//   - 可以区分零值和未设置
	//   - 节省内存（对于大型结构体）
	// 指针的劣势:
	//   - 使用时需要判空
	//   - 代码略显繁琐
	UsePointer bool

	// OmitEmpty 是否在标签中添加 omitempty 选项
	// true 时生成 `json:"field,omitempty"`。
	// false 时生成 `json:"field"`。
	// 默认: false
	// 作用: 序列化时忽略零值字段
	OmitEmpty bool

	// IndentStyle 缩进风格
	// "tab": 使用 tab 缩进（推荐）
	// "space": 使用空格缩进
	// 默认: "tab"
	IndentStyle string

	// AddComments 是否添加字段注释
	// true: 为每个字段生成注释
	// false: 不生成注释
	// 默认: false
	AddComments bool

	// EnvPrefix 环境变量前缀
	// 例如: "APP_" 则生成的环境变量为 APP_SERVER_HOST
	// 默认: ""（空字符串，直接使用配置名作为前缀，如 SERVER_HOST）
	EnvPrefix string

	// GenerateMethods 是否生成接口方法
	// true: 为每个配置生成 ValidateName, Validate, DefaultConfig, OverrideConfig 方法
	// false: 只生成结构体定义
	// 默认: true
	GenerateMethods bool

	// SplitFiles 是否分离文件
	// true: 为每个顶级配置生成独立文件（新模式）
	// false: 生成单个文件（兼容模式）
	// 默认: true
	SplitFiles bool
}

// New 创建一个新的 Converter 实例
// 参数:
//
//	config: 配置对象，nil 时使用默认配置
//
// 返回:
//
//	Converter: 转换器实例
//
// 使用示例:
//
//	// 使用默认配置
//	converter := togo.New(nil)
//
//	// 自定义配置
//	converter := togo.New(&togo.Config{
//	    PackageName: "config",
//	    StructName:  "AppConfig",
//	})
func New(config *Config) Converter {
	return newConverter(config)
}

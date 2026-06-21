package yaml2go

// 本文件属于 YAML 配置代码生成器，把配置样例解析为可编译的 Go 结构体、标签和辅助方法。

// FieldInfo 字段信息
// 描述结构体中的一个字段
type FieldInfo struct {
	// Name 字段名（Go 格式，驼峰命名）
	Name string

	// OriginalName 原始名称（YAML 中的名称）
	OriginalName string

	// Type 字段类型
	Type FieldType

	// IsPointer 是否为指针类型
	IsPointer bool

	// Comment 字段注释
	Comment string

	// Tags 标签映射 {tagName: tagValue}
	// 例如: {"json": "field_name", "yaml": "field_name"}
	Tags map[string]string

	// Children 子字段（用于嵌套结构体）
	Children []*FieldInfo

	// ElementType 数组元素类型（用于数组类型）
	ElementType *FieldInfo
}

// FieldType 字段类型枚举
type FieldType int

const (
	// TypeUnknown 未知类型
	TypeUnknown FieldType = iota

	// TypeString 字符串类型
	TypeString

	// TypeInt 整数类型
	TypeInt

	// TypeFloat 浮点数类型
	TypeFloat

	// TypeBool 布尔类型
	TypeBool

	// TypeStruct 结构体类型
	TypeStruct

	// TypeSlice 数组类型
	TypeSlice

	// TypeMap Map 类型
	TypeMap

	// TypeInterface 接口类型（用于无法推断的类型）
	TypeInterface
)

// String 返回类型的字符串表示
func (t FieldType) String() string {
	switch t {
	case TypeString:
		return "string"
	case TypeInt:
		return "int64"
	case TypeFloat:
		return "float64"
	case TypeBool:
		return "bool"
	case TypeStruct:
		return "struct"
	case TypeSlice:
		return "slice"
	case TypeMap:
		return "map[string]interface{}"
	case TypeInterface:
		return "interface{}"
	default:
		return "unknown"
	}
}

// StructInfo 结构体信息
// 描述整个结构体的元数据
type StructInfo struct {
	// Name 结构体名称
	Name string

	// PackageName 包名
	PackageName string

	// Fields 字段列表
	Fields []*FieldInfo

	// Comment 结构体注释
	Comment string

	// ConfigName 配置名称（对应 YAML 中的顶级键名）
	// 例如: "server", "database", "redis"
	ConfigName string

	// DefaultValues 默认值映射 {字段名: 默认值}
	// 用于生成 DefaultConfig 方法
	DefaultValues map[string]interface{}
}

// ConfigNode 配置节点接口
// 每个生成的配置结构体都应实现此接口
// 用于统一配置管理、验证和环境变量覆盖
type ConfigNode interface {
	// ValidateName 返回配置名称（对应YAML中的键名）
	// 例如: "server", "database", "redis"
	ValidateName() string

	// Validate 验证配置的有效性
	// 返回 nil 表示验证通过，否则返回错误信息
	// 开发者可在生成代码后自行添加验证逻辑
	Validate() error

	// DefaultConfig 返回默认配置
	// 返回一个包含默认值的配置实例
	DefaultConfig() interface{}

	// OverrideConfig 使用环境变量覆盖配置
	// prefix: 环境变量前缀，如 "APP_" 则最终变量名为 APP_SERVER_HOST
	// 支持的类型转换:
	//   - string: 直接赋值
	//   - int64 使用 strconv.ParseInt。
	//   - float64 使用 strconv.ParseFloat。
	//   - bool 使用 strconv.ParseBool。
	OverrideConfig(prefix string)
}

// FileContent 文件内容
// 表示生成的单个 Go 源文件
type FileContent struct {
	// FileName 文件名（不含路径，如 "server_config.go"）
	FileName string

	// Content 文件内容（完整的 Go 代码）
	Content string

	// ConfigName 配置名称（如 "server", "database"）
	// 空字符串表示主配置文件
	ConfigName string

	// StructName 结构体名称（如 "ServerConfig", "DatabaseConfig"）
	StructName string
}

// GenerateResult 代码生成结果
// 包含主配置文件和所有子配置文件
type GenerateResult struct {
	// MainConfig 主配置文件内容（config.go）
	// 包含所有子配置的组合结构体
	MainConfig *FileContent

	// SubConfigs 子配置文件列表
	// 每个元素对应一个顶级配置（如 server_config.go, database_config.go）
	SubConfigs []*FileContent

	// PackageName 包名
	PackageName string
}

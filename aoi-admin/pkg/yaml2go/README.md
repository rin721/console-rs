# pkg/yaml2go - YAML 到 Go 结构体生成工具

YAML 转 Go 结构体代码生成工具库

## 📖 简介

`pkg/yaml2go` 是一个**纯转换工具**，可以将 YAML 格式的配置自动转换为 Go 结构体代码。

## API 分类

- 定位：[CONFIRMED] 公共工具 API。
- 稳定边界：`Converter`、`Config`、`New`、`Convert` 返回的代码内容结构。
- 当前风险：[CONFIRMED] 包自身已有最小行为测试；原命令演示测试依赖已在 TASK-P1-006 移除。
- 非目标：[CONFIRMED] 本包不做文件写入、不管理 CLI 命令、不修改项目配置。

> [!IMPORTANT]
> **纯转换，不做文件操作**
>
> 本工具只负责代码生成，不包含任何文件写入功能。用户完全控制如何使用生成的代码。

### 核心特性

- ✅ **配置分离**: 每个顶级配置生成独立的结构体（如 `ServerConfig`、`DatabaseConfig`）
- ✅ **统一接口**: 每个配置自动实现 `ValidateName()`、`Validate()`、`DefaultConfig()`、`OverrideConfig()`
- ✅ **环境变量覆盖**: 自动生成环境变量读取和类型转换逻辑
- ✅ **智能类型推断**: 自动识别 string、int、float、bool、数组、嵌套结构等类型
- ✅ **多标签支持**: 自动生成 `json`、`yaml`、`mapstructure`（Viper）、`toml` 等标签
- ✅ **命名转换**: 自动将 snake_case 转换为 PascalCase
- ✅ **纯转换工具**: 只返回代码，用户自己决定如何使用
- ✅ **线程安全**: 所有方法都是并发安全的

## 🚀 快速开始

### 基本用法

```go
package main

import (
    "fmt"
    "log"
    "github.com/rei0721/go-scaffold/pkg/yaml2go"
)

func main() {
    yamlStr := `
server:
  host: localhost
  port: 8080
  required: true

database:
  host: localhost
  port: 5432
  username: admin
`

    // 创建转换器
    converter := yaml2go.New(nil)

    // 转换 YAML
    result, err := converter.Convert(yamlStr)
    if err != nil {
        log.Fatal(err)
    }

    // 查看主配置
    fmt.Println("=== config.go ===")
    fmt.Println(result.MainConfig.Content)

    // 查看子配置
    for _, subConfig := range result.SubConfigs {
        fmt.Printf("\n=== %s ===\n", subConfig.FileName)
        fmt.Println(subConfig.Content)
    }
}
```

### 写入文件（用户自己控制）

```go
import (
    "os"
    "path/filepath"
)

// 用户自己决定如何处理生成的代码
outputDir := "./internal/config"
os.MkdirAll(outputDir, 0755)

// 写入主配置
mainPath := filepath.Join(outputDir, result.MainConfig.FileName)
os.WriteFile(mainPath, []byte(result.MainConfig.Content), 0644)

// 写入子配置
for _, subConfig := range result.SubConfigs {
    subPath := filepath.Join(outputDir, subConfig.FileName)
    os.WriteFile(subPath, []byte(subConfig.Content), 0644)
}
```

### 自定义配置

```go
converter := yaml2go.New(&yaml2go.Config{
    PackageName:     "config",                              // 包名
    EnvPrefix:       "APP_",                                // 环境变量前缀
    GenerateMethods: true,                                  // 生成接口方法
    SplitFiles:      true,                                  // 分离文件（新模式）
    Tags:            []string{"json", "yaml", "mapstructure"}, // 自定义标签
    OmitEmpty:       true,                                  // 添加 omitempty 选项
    AddComments:     true,                                  // 添加字段注释
})

result, err := converter.Convert(yamlStr)
```

## 📚 API 文档

### Converter 接口

```go
type Converter interface {
    // Convert 转换 YAML 字符串为多个配置代码
    // 返回 GenerateResult，包含主配置和所有子配置
    Convert(yamlStr string) (*GenerateResult, error)

    // SetConfig 更新配置（支持热更新）
    SetConfig(config *Config) error
}
```

### GenerateResult 结构

```go
type GenerateResult struct {
    // MainConfig 主配置文件（config.go）
    MainConfig *FileContent

    // SubConfigs 子配置文件列表
    SubConfigs []*FileContent

    // PackageName 包名
    PackageName string
}

type FileContent struct {
    // FileName 文件名（如 "server_config.go"）
    FileName string

    // Content 文件内容（完整的 Go 代码）
    Content string

    // ConfigName 配置名称（如 "server"）
    ConfigName string

    // StructName 结构体名称（如 "ServerConfig"）
    StructName string
}
```

### Config 配置

| 字段              | 类型     | 默认值                                   | 说明                         |
| ----------------- | -------- | ---------------------------------------- | ---------------------------- |
| `PackageName`     | string   | "main"                                   | 生成代码的包名               |
| `StructName`      | string   | "Config"                                 | 根结构体名称                 |
| `Tags`            | []string | ["json", "yaml", "mapstructure", "toml"] | 生成的标签列表               |
| `UsePointer`      | bool     | false                                    | 字段是否使用指针类型         |
| `OmitEmpty`       | bool     | false                                    | 是否添加 omitempty 选项      |
| `IndentStyle`     | string   | "tab"                                    | 缩进风格（"tab" 或 "space"） |
| `AddComments`     | bool     | false                                    | 是否添加字段注释             |
| `EnvPrefix`       | string   | ""                                       | 环境变量前缀（如 "APP\_"）   |
| `GenerateMethods` | bool     | true                                     | 是否生成接口方法             |
| `SplitFiles`      | bool     | true                                     | 是否分离文件（新模式）       |

### 构造函数

```go
// New 创建转换器实例
// config 为 nil 时使用默认配置
func New(config *Config) Converter
```

## 🎯 使用场景

### 1. 配合 Viper 使用

**步骤 1: 生成结构体（开发阶段）**

```go
// tools/gen_config.go
package main

import (
    "log"
    "os"
    "github.com/rei0721/go-scaffold/pkg/yaml2go"
)

func main() {
    yamlBytes, _ := os.ReadFile("config.yaml")

    converter := yaml2go.New(&yaml2go.Config{
        PackageName: "config",
        StructName:  "AppConfig",
    })

    err := converter.ConvertToFile(string(yamlBytes), "internal/config/types.go")
    if err != nil {
        log.Fatal(err)
    }
}
```

**步骤 2: 使用生成的结构体（运行时）**

```go
package main

import (
    "github.com/spf13/viper"
    "yourapp/internal/config"
)

func main() {
    var cfg config.AppConfig

    viper.SetConfigFile("config.yaml")
    if err := viper.ReadInConfig(); err != nil {
        panic(err)
    }

    if err := viper.Unmarshal(&cfg); err != nil {
        panic(err)
    }

    // 使用配置
    fmt.Println(cfg.Database.Host)
}
```

### 2. 多环境配置

```go
environments := []string{"dev", "staging", "prod"}

for _, env := range environments {
    yamlBytes, _ := os.ReadFile(fmt.Sprintf("config.%s.yaml", env))

    converter := yaml2go.New(&yaml2go.Config{
        PackageName: "config",
        StructName:  fmt.Sprintf("%sConfig", strings.Title(env)),
    })

    converter.ConvertToFile(
        string(yamlBytes),
        fmt.Sprintf("internal/config/%s.go", env),
    )
}
```

### 3. API 模型生成

```go
// 从 OpenAPI/Swagger YAML 生成请求/响应结构体
converter := yaml2go.New(&yaml2go.Config{
    PackageName: "models",
    StructName:  "UserRequest",
    Tags:        []string{"json", "validate"},
    OmitEmpty:   true,
})

code, _ := converter.Convert(apiSchemaYaml)
```

## 🔧 类型映射

| YAML 类型 | Go 类型     | 示例                                    |
| --------- | ----------- | --------------------------------------- |
| 字符串    | string      | `name: "John"` → `Name string`          |
| 整数      | int64       | `port: 8080` → `Port int64`             |
| 浮点数    | float64     | `price: 19.99` → `Price float64`        |
| 布尔值    | bool        | `debug: true` → `Debug bool`            |
| 数组      | []T         | `tags: ["a", "b"]` → `Tags []string`    |
| 对象      | struct      | `user: {name: ""}` → `User struct{...}` |
| null      | interface{} | `data: null` → `Data interface{}`       |

## 🎨 命名规则

### 字段名转换

- YAML: `my_field` → Go: `MyField`
- YAML: `database_host` → Go: `DatabaseHost`
- YAML: `api_key` → Go: `ApiKey`

### 标签保留原名

```go
type Config struct {
    MyField      string `json:"my_field" yaml:"my_field"`
    DatabaseHost string `json:"database_host" yaml:"database_host"`
}
```

### Go 关键字处理

如果字段名是 Go 关键字，会自动添加前缀：

- YAML: `type` → Go: `FieldType`
- YAML: `interface` → Go: `FieldInterface`

## ⚠️ 注意事项

### 1. 数组类型推断

数组类型基于**第一个元素**推断：

```yaml
items:
  - name: "A"
    value: 1
  - name: "B"
    value: 2
```

生成：

```go
Items []struct {
    Name  string `json:"name"`
    Value int64  `json:"value"`
}
```

**空数组无法推断类型**：

```yaml
empty_list: []
```

生成：

```go
EmptyList []interface{} `json:"empty_list"`
```

### 2. 指针类型选择

**不使用指针（默认）:**

```go
type Config struct {
    Port int64  `json:"port"`  // 零值为 0
}
```

**使用指针:**

```go
type Config struct {
    Port *int64  `json:"port"`  // 零值为 nil，可区分未设置和设为 0
}
```

### 3. OmitEmpty 选项

**不使用 omitempty:**

```go
Port int64 `json:"port"`  // 即使为 0 也会序列化
```

**使用 omitempty:**

```go
Port int64 `json:"port,omitempty"`  // 为 0 时不序列化
```

## 🔍 故障排查

### 错误: ErrInvalidYAML

**原因:** YAML 格式不正确

**解决:**

- 检查 YAML 缩进（必须使用空格，不能用 Tab）
- 验证 YAML 语法：https://www.yamllint.com/
- 确保键值对格式正确

### 错误: ErrEmptyInput

**原因:** 输入字符串为空

**解决:**

- 检查读取文件是否成功
- 确保 YAML 字符串不为空

### 生成的代码无法编译

**原因:** 可能是字段名冲突或类型推断错误

**解决:**

- 检查生成的字段名是否重复
- 手动调整复杂类型的定义
- 启用 `AddComments` 帮助识别问题字段

## 📦 依赖

- `github.com/dave/jennifer/jen` - Go 代码生成
- `gopkg.in/yaml.v3` - YAML 解析
- `github.com/iancoleman/strcase` - 字符串格式转换

## 🤝 最佳实践

1. **开发阶段使用**
   - 在开发时生成结构体代码
   - 不要在运行时动态生成（性能开销）

2. **版本控制**
   - 将生成的代码提交到版本控制
   - 便于 code review 和追踪变更

3. **代码组织**
   - 将生成的结构体放在独立的文件（如 `types.go`）
   - 不要与业务逻辑混在一起

4. **配置验证**
   - 生成后运行 `go fmt` 格式化
   - 使用 `go build` 验证能否编译
   - 添加单元测试验证序列化/反序列化

## 📄 许可证

与主项目保持一致

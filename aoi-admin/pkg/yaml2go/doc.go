/*
Package yaml2go 提供 YAML 字符串到 Go 结构体代码的转换功能

# 设计目标

- 简单易用: 提供简洁的 API,一行代码完成转换
- 智能推断: 自动推断字段类型,支持嵌套结构和数组
- 多标签支持: 自动生成 json、yaml、mapstructure（viper）、toml 等标签
- 配置驱动: 支持自定义包名、结构体名、命名风格等
- 线程安全: 所有方法都是并发安全的

# 核心价值

在开发配置驱动的应用时,常常需要根据 YAML 配置文件定义对应的 Go 结构体。
手动编写这些结构体不仅繁琐,还容易出错。本包可以自动完成这个过程:

1. 粘贴 YAML 配置
2. 调用转换器
3. 获得格式规范的 Go 代码

特别适合与 Viper、Cobra 等配置管理库配合使用。

# 核心概念

## YAML 到 Go 的类型映射

本包使用以下规则推断类型:
  - YAML 字符串 -> Go string
  - YAML 整数 -> Go int64
  - YAML 浮点数 -> Go float64
  - YAML 布尔值 -> Go bool
  - YAML 数组 -> Go []T
  - YAML 对象 -> Go struct
  - YAML null -> Go interface{}

## 标签生成

默认为每个字段生成以下标签:
  - json: 用于 encoding/json
  - yaml: 用于 gopkg.in/yaml.v3
  - mapstructure: 用于 Viper 配置绑定
  - toml: 用于 TOML 序列化

可通过 Config.Tags 自定义需要的标签。

## 命名转换

YAML 字段名通常使用 snake_case,Go 字段名使用 PascalCase:
  - my_field -> MyField
  - database_host -> DatabaseHost
  - api_key -> ApiKey

标签中保留原始的 YAML 字段名,确保序列化兼容性。

# 使用示例

基本用法:

	import (
		"fmt"
		"log"
		"github.com/rei0721/go-scaffold/pkg/yaml2go"
	)

	func main() {
		yamlStr := `

database:

	host: localhost
	port: 5432
	username: admin
	password: secret

server:

	port: 8080
	timeout: 30
	debug: true

`

		// 使用默认配置
		converter := togo.New(nil)
		code, err := converter.Convert(yamlStr)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(code)
	}

输出结果:

	package main

	// Config 配置结构
	type Config struct {
		Database struct {
			Host     string `json:"host" yaml:"host" mapstructure:"host" toml:"host"`
			Password string `json:"password" yaml:"password" mapstructure:"password" toml:"password"`
			Port     int64  `json:"port" yaml:"port" mapstructure:"port" toml:"port"`
			Username string `json:"username" yaml:"username" mapstructure:"username" toml:"username"`
		} `json:"database" yaml:"database" mapstructure:"database" toml:"database"`
		Server struct {
			Debug   bool  `json:"debug" yaml:"debug" mapstructure:"debug" toml:"debug"`
			Port    int64 `json:"port" yaml:"port" mapstructure:"port" toml:"port"`
			Timeout int64 `json:"timeout" yaml:"timeout" mapstructure:"timeout" toml:"timeout"`
		} `json:"server" yaml:"server" mapstructure:"server" toml:"server"`
	}

自定义配置:

	converter := togo.New(&togo.Config{
		PackageName: "config",
		StructName:  "AppConfig",
		Tags:        []string{"json", "yaml"},  // 只生成这两个标签
		OmitEmpty:   true,                       // 添加 omitempty
		AddComments: true,                       // 添加字段注释
	})

	code, err := converter.Convert(yamlStr)

转换并保存到文件:

	err := converter.ConvertToFile(yamlStr, "config/models.go")
	if err != nil {
		log.Fatal(err)
	}

配合 Viper 使用:

	import (
		"github.com/spf13/viper"
		"github.com/rei0721/go-scaffold/pkg/yaml2go"
	)

	// 1. 使用 yaml2go 生成配置结构体（开发阶段）
	converter := togo.New(&togo.Config{
		PackageName: "config",
		StructName:  "AppConfig",
	})
	// ... 转换 YAML 为 Go 代码并保存

	// 2. 使用生成的结构体加载配置（运行时）
	var cfg AppConfig
	viper.SetConfigFile("config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(err)
	}

# 最佳实践

1. 开发流程
  - 先设计 YAML 配置文件
  - 使用 yaml2go 生成对应的 Go 结构体
  - 将生成的代码保存到 config 或 model 包
  - 在应用中使用结构体加载配置

2. 标签选择
  - 如果只用于内部配置,生成 json 和 yaml 标签即可
  - 如果使用 Viper,必须包含 mapstructure 标签
  - 如果需要多种配置格式,包含所有标签

3. 命名规范
  - 使用有意义的结构体名,如 DatabaseConfig、ServerConfig
  - 避免使用通用名称如 Config、Settings（容易冲突）
  - 包名建议使用 config、model 等

4. 类型选择
  - 默认不使用指针类型(UsePointer: false)
  - 如需区分零值和未设置,启用指针类型
  - 对于必填字段,不使用指针;对于可选字段,使用指针

5. 错误处理
  - 生成阶段的错误应该在开发时修复
  - 验证生成的代码能够编译通过
  - 使用 go fmt 格式化生成的代码

# 线程安全

所有公开方法都是线程安全的,可以在并发环境下安全使用。
内部使用 sync.RWMutex 保护配置的读写。

可以安全地在多个 goroutine 中共享同一个 Converter 实例:

	converter := togo.New(nil)

	go func() {
		code, _ := converter.Convert(yaml1)
		// ...
	}()

	go func() {
		code, _ := converter.Convert(yaml2)
		// ...
	}()

# 性能考虑

- YAML 解析是 CPU 密集型操作,大文件可能耗时较长
- 建议在开发阶段使用,而非运行时动态生成
- 如需运行时使用,考虑缓存生成结果

# 限制

1. 类型推断基于第一个元素
  - 对于数组,只检查第一个元素的类型
  - 如果数组为空,元素类型为 interface{}

2. Map 类型
  - 当前版本对复杂 Map 的支持有限
  - 推荐使用嵌套对象代替 Map

3. 自定义类型
  - 不支持生成自定义类型（如 time.Time）
  - 复杂类型会被推断为 interface{}

# 依赖

- github.com/dave/jennifer/jen: Go 代码生成库
- gopkg.in/yaml.v3: YAML 解析库
- github.com/iancoleman/strcase: 字符串格式转换库
*/
package yaml2go

// 本文件承载包级 Godoc 入口，集中说明该包在脚手架架构中的定位、使用边界和非目标能力。

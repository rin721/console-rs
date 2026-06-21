package yaml2go

// 本文件属于 YAML 配置代码生成器，把配置样例解析为可编译的 Go 结构体、标签和辅助方法。

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
)

// generateMethods 为配置结构体生成所有接口方法
func (c *converter) generateMethods(structInfo *StructInfo, cfg *Config) (string, error) {
	if !cfg.GenerateMethods {
		return "", nil
	}

	f := jen.NewFile(structInfo.PackageName)
	c.appendMethods(f, structInfo, cfg)

	buf := &bytes.Buffer{}
	if err := f.Render(buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// appendMethods 按配置为生成文件追加辅助方法，保证主配置和子配置拥有一致的运行时契约。
func (c *converter) appendMethods(f *jen.File, structInfo *StructInfo, cfg *Config) {
	// 需要导入的包
	f.ImportName("os", "os")

	// 1. 生成 ValidateName 方法
	c.generateValidateNameMethod(f, structInfo)

	// 2. 生成 Validate 方法
	c.generateValidateMethod(f, structInfo)

	// 3. 生成 DefaultConfig 方法
	c.generateDefaultConfigMethod(f, structInfo, cfg)

	// 4. 生成 OverrideConfig 方法
	c.generateOverrideConfigMethod(f, structInfo, cfg)
}

// generateValidateNameMethod 生成 ValidateName 方法
func (c *converter) generateValidateNameMethod(f *jen.File, structInfo *StructInfo) {
	f.Comment("ValidateName 返回配置名称")
	f.Func().Params(
		jen.Id("c").Op("*").Id(structInfo.Name),
	).Id("ValidateName").Params().String().Block(
		jen.Return(jen.Lit(structInfo.ConfigName)),
	)
	f.Line()
}

// generateValidateMethod 生成 Validate 方法
func (c *converter) generateValidateMethod(f *jen.File, structInfo *StructInfo) {
	f.Comment("Validate 验证配置")
	f.Comment("开发者可在此添加自定义验证逻辑。")
	f.Func().Params(
		jen.Id("c").Op("*").Id(structInfo.Name),
	).Id("Validate").Params().Error().Block(
		jen.Return(jen.Nil()),
	)
	f.Line()
}

// generateDefaultConfigMethod 生成 DefaultConfig 方法
func (c *converter) generateDefaultConfigMethod(f *jen.File, structInfo *StructInfo, cfg *Config) {
	f.Comment("DefaultConfig 返回默认配置")
	f.Func().Params(
		jen.Id("c").Op("*").Id(structInfo.Name),
	).Id("DefaultConfig").Params().Op("*").Id(structInfo.Name).Block(
		jen.Return(c.buildDefaultConfigStruct(structInfo, cfg)),
	)
	f.Line()
}

// buildDefaultConfigStruct 构建默认配置结构体
func (c *converter) buildDefaultConfigStruct(structInfo *StructInfo, cfg *Config) *jen.Statement {
	// 创建结构体字面量
	fields := []jen.Code{}

	for _, field := range structInfo.Fields {
		// 从 DefaultValues 中获取默认值
		if defaultValue, ok := structInfo.DefaultValues[field.OriginalName]; ok {
			fieldAssign := jen.Id(field.Name).Op(":").Lit(defaultValue)
			fields = append(fields, fieldAssign)
		}
	}

	return jen.Op("&").Id(structInfo.Name).Values(fields...)
}

// generateOverrideConfigMethod 生成 OverrideConfig 方法
func (c *converter) generateOverrideConfigMethod(f *jen.File, structInfo *StructInfo, cfg *Config) {
	// 需要导入 strconv
	f.ImportName("strconv", "strconv")

	f.Comment("OverrideConfig 使用环境变量覆盖配置")
	f.Func().Params(
		jen.Id("cfg").Op("*").Id(structInfo.Name),
	).Id("OverrideConfig").Params(jen.Id("prefix").String()).Block(
		c.buildOverrideConfigBody(structInfo, cfg)...,
	)
	f.Line()
}

// buildOverrideConfigBody 构建 OverrideConfig 方法体
func (c *converter) buildOverrideConfigBody(structInfo *StructInfo, cfg *Config) []jen.Code {
	var statements []jen.Code

	// 为每个字段生成环境变量覆盖逻辑
	for _, field := range structInfo.Fields {
		envKey := c.buildEnvKey(structInfo.ConfigName, field.OriginalName)
		statement := c.buildFieldOverrideStatement(field, envKey)
		if statement != nil {
			statements = append(statements, statement...)
		}
	}

	return statements
}

// buildEnvKey 构建环境变量键名
// 例如: server + host -> "SERVER_HOST"
func (c *converter) buildEnvKey(configName, fieldName string) string {
	// 转换为大写 + 下划线分隔
	parts := []string{
		strings.ToUpper(toSnakeCase(configName)),
		strings.ToUpper(toSnakeCase(fieldName)),
	}
	return strings.Join(parts, "_")
}

// buildFieldOverrideStatement 为单个字段生成环境变量覆盖语句
func (c *converter) buildFieldOverrideStatement(field *FieldInfo, envKey string) []jen.Code {
	var statements []jen.Code

	// 添加注释
	statements = append(statements, jen.Comment(field.Name))

	// 根据字段类型生成不同的转换逻辑
	switch field.Type {
	case TypeString:
		// 字符串类型：直接赋值
		statements = append(statements,
			jen.If(
				jen.List(jen.Id("val")).Op(":=").Qual("os", "Getenv").Call(
					jen.Id("prefix").Op("+").Lit(envKey),
				),
				jen.Id("val").Op("!=").Lit(""),
			).Block(
				jen.Id("cfg").Dot(field.Name).Op("=").Id("val"),
			),
		)

	case TypeInt:
		// 整数类型：使用 strconv.Atoi
		statements = append(statements,
			jen.If(
				jen.List(jen.Id("val")).Op(":=").Qual("os", "Getenv").Call(
					jen.Id("prefix").Op("+").Lit(envKey),
				),
				jen.Id("val").Op("!=").Lit(""),
			).Block(
				jen.If(
					jen.List(jen.Id("parsed"), jen.Id("err")).Op(":=").Qual("strconv", "Atoi").Call(jen.Id("val")),
					jen.Id("err").Op("==").Nil(),
				).Block(
					jen.Id("cfg").Dot(field.Name).Op("=").Int64().Call(jen.Id("parsed")),
				),
			),
		)

	case TypeFloat:
		// 浮点数类型：使用 strconv.ParseFloat
		statements = append(statements,
			jen.If(
				jen.List(jen.Id("val")).Op(":=").Qual("os", "Getenv").Call(
					jen.Id("prefix").Op("+").Lit(envKey),
				),
				jen.Id("val").Op("!=").Lit(""),
			).Block(
				jen.If(
					jen.List(jen.Id("parsed"), jen.Id("err")).Op(":=").Qual("strconv", "ParseFloat").Call(
						jen.Id("val"),
						jen.Lit(64),
					),
					jen.Id("err").Op("==").Nil(),
				).Block(
					jen.Id("cfg").Dot(field.Name).Op("=").Id("parsed"),
				),
			),
		)

	case TypeBool:
		// 布尔类型：使用 strconv.ParseBool
		statements = append(statements,
			jen.If(
				jen.List(jen.Id("val")).Op(":=").Qual("os", "Getenv").Call(
					jen.Id("prefix").Op("+").Lit(envKey),
				),
				jen.Id("val").Op("!=").Lit(""),
			).Block(
				jen.If(
					jen.List(jen.Id("parsed"), jen.Id("err")).Op(":=").Qual("strconv", "ParseBool").Call(jen.Id("val")),
					jen.Id("err").Op("==").Nil(),
				).Block(
					jen.Id("cfg").Dot(field.Name).Op("=").Id("parsed"),
				),
			),
		)

	case TypeStruct:
		// 嵌套结构体：递归处理子字段
		for _, child := range field.Children {
			childEnvKey := c.buildEnvKey(envKey, child.OriginalName)
			childStatements := c.buildFieldOverrideStatement(child, childEnvKey)
			statements = append(statements, childStatements...)
		}

	case TypeSlice:
		// 数组类型：使用注释说明
		statements = append(statements,
			jen.Comment(fmt.Sprintf("Slice type %s: use JSON format in env var", field.Name)),
		)

	default:
		// 其他类型跳过
		statements = append(statements,
			jen.Comment(fmt.Sprintf("Unsupported type for %s", field.Name)),
		)
	}

	statements = append(statements, jen.Line())
	return statements
}

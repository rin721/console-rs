package yaml2go

// 本文件属于 YAML 配置代码生成器，把配置样例解析为可编译的 Go 结构体、标签和辅助方法。

import "errors"

var (
	// ErrInvalidYAML YAML 格式无效
	// 当输入的 YAML 字符串无法解析时返回
	ErrInvalidYAML = errors.New("invalid YAML format")

	// ErrEmptyInput 输入为空
	// 当传入空字符串时返回
	ErrEmptyInput = errors.New("empty input string")

	// ErrTypeInference 类型推断失败
	// 当无法确定字段类型时返回
	ErrTypeInference = errors.New("failed to infer type")

	// ErrCodeGeneration 代码生成失败
	// 当生成 Go 代码时发生错误
	ErrCodeGeneration = errors.New("failed to generate code")

	// ErrInvalidConfig 配置无效
	// 当提供的配置参数不合法时返回
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrFileWrite 文件写入失败
	// 当无法写入输出文件时返回
	ErrFileWrite = errors.New("failed to write file")
)

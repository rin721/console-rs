package yaml2go

// 本文件属于 YAML 配置代码生成器，把配置样例解析为可编译的 Go 结构体、标签和辅助方法。

import (
	"strings"
	"unicode"

	"github.com/iancoleman/strcase"
)

// toGoFieldName 将 YAML 字段名转换为 Go 字段名（大写开头的驼峰命名）
// 例如: "my_field" -> "MyField", "myField" -> "MyField"
func toGoFieldName(yamlName string) string {
	// 使用 strcase 库转换为驼峰命名
	return strcase.ToCamel(yamlName)
}

// toSnakeCase 将字符串转换为 snake_case
// 例如: "MyField" -> "my_field"
func toSnakeCase(s string) string {
	return strcase.ToSnake(s)
}

// buildTag 构建单个标签字符串
// 例如: buildTag("json", "field_name", true) -> "json:\"field_name,omitempty\""
func buildTag(tagName, tagValue string, omitEmpty bool) string {
	if omitEmpty {
		return tagName + ":\"" + tagValue + ",omitempty\""
	}
	return tagName + ":\"" + tagValue + "\""
}

// buildTags 构建完整的标签字符串
// 例如: buildTags(map{"json":"field","yaml":"field"}, true) -> "`json:\"field,omitempty\" yaml:\"field,omitempty\"`"
func buildTags(tags map[string]string, omitEmpty bool) string {
	if len(tags) == 0 {
		return ""
	}

	var parts []string
	// 按固定顺序生成标签: json, yaml, mapstructure, toml, xml, validate
	order := []string{"json", "yaml", "mapstructure", "toml", "xml", "validate"}

	for _, key := range order {
		if value, ok := tags[key]; ok {
			parts = append(parts, buildTag(key, value, omitEmpty))
		}
	}

	// 添加其他未在顺序中的标签
	for key, value := range tags {
		if !contains(order, key) {
			parts = append(parts, buildTag(key, value, omitEmpty))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "`" + strings.Join(parts, " ") + "`"
}

// buildTagValues 构建 Jennifer 使用的 tag 值，并保留 omitempty 等 tag option。
func buildTagValues(tags map[string]string, omitEmpty bool) map[string]string {
	if len(tags) == 0 {
		return nil
	}

	values := make(map[string]string, len(tags))
	for key, value := range tags {
		if omitEmpty {
			value += ",omitempty"
		}
		values[key] = value
	}
	return values
}

// contains 检查字符串切片是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isGoKeyword 检查是否为 Go 语言关键字
func isGoKeyword(name string) bool {
	keywords := []string{
		"break", "case", "chan", "const", "continue",
		"default", "defer", "else", "fallthrough", "for",
		"func", "go", "goto", "if", "import",
		"interface", "map", "package", "range", "return",
		"select", "struct", "switch", "type", "var",
	}
	return contains(keywords, name)
}

// sanitizeFieldName 规范化字段名
// 如果字段名是 Go 关键字，添加下划线前缀
func sanitizeFieldName(name string) string {
	goName := toGoFieldName(name)
	if isGoKeyword(strings.ToLower(goName)) {
		return "Field" + goName
	}
	return goName
}

// isNumeric 检查字符串是否为数字
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) && r != '.' && r != '-' && r != '+' && r != 'e' && r != 'E' {
			return false
		}
	}
	return true
}

// hasDecimalPoint 检查字符串是否包含小数点
func hasDecimalPoint(s string) bool {
	return strings.Contains(s, ".")
}

// copyStringSlice 复制字符串切片
func copyStringSlice(src []string) []string {
	if src == nil {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

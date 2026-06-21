package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

import (
	"reflect"
	"strconv"
	"strings"
)

// ============================================================================
// GORM Tag 解析
// ============================================================================

// ParsedTag 解析后的 GORM Tag
type ParsedTag struct {
	Column        string
	Type          string
	Size          int
	PrimaryKey    bool
	AutoIncrement bool
	NotNull       bool
	Default       string
	Index         string
	UniqueIndex   string
	Comment       string
	Ignore        bool // 对应 gorm:"-"，表示忽略该字段。
}

// parseGormTag 解析 gorm struct tag
func parseGormTag(tag string) *ParsedTag {
	result := &ParsedTag{}

	if tag == "" || tag == "-" {
		result.Ignore = tag == "-"
		return result
	}

	parts := strings.Split(tag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 处理键值对
		if idx := strings.Index(part, ":"); idx > 0 {
			key := strings.TrimSpace(part[:idx])
			value := strings.TrimSpace(part[idx+1:])

			switch strings.ToLower(key) {
			case "column":
				result.Column = value
			case "type":
				result.Type = value
			case "size":
				if size, err := strconv.Atoi(value); err == nil {
					result.Size = size
				}
			case "default":
				result.Default = value
			case "index":
				result.Index = value
			case "uniqueindex":
				result.UniqueIndex = value
			case "comment":
				result.Comment = value
			}
		} else {
			// 处理单独的标志
			switch strings.ToLower(part) {
			case "primarykey", "primary_key":
				result.PrimaryKey = true
			case "autoincrement", "auto_increment":
				result.AutoIncrement = true
			case "not null", "notnull":
				result.NotNull = true
			case "index":
				result.Index = "true"
			case "uniqueindex", "unique_index":
				result.UniqueIndex = "true"
			case "-":
				result.Ignore = true
			}
		}
	}

	return result
}

// ============================================================================
// 结构体字段解析
// ============================================================================

// FieldInfo 字段解析结果
type FieldInfo struct {
	// Name Go 字段名
	Name string

	// ColumnName 数据库列名
	ColumnName string

	// Type Go 类型
	Type string

	// SQLType SQL 类型
	SQLType string

	// Tag 解析后的 GORM Tag
	Tag *ParsedTag

	// Value 字段值 (如果有)
	Value interface{}

	// IsZero 是否为零值
	IsZero bool

	// Index 字段在结构体中的索引
	Index int
}

// parseStructFields 解析结构体字段
func parseStructFields(t reflect.Type, v reflect.Value, dialect DialectHandler) []FieldInfo {
	var fields []FieldInfo

	// 确保是结构体类型
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if t.Kind() != reflect.Struct {
		return fields
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 跳过非导出字段
		if !field.IsExported() {
			continue
		}

		// 处理嵌入字段
		if field.Anonymous {
			embeddedType := field.Type
			var embeddedValue reflect.Value
			if v.IsValid() {
				embeddedValue = v.Field(i)
			}
			embeddedFields := parseStructFields(embeddedType, embeddedValue, dialect)
			fields = append(fields, embeddedFields...)
			continue
		}

		// 解析 gorm tag
		gormTag := field.Tag.Get("gorm")
		parsedTag := parseGormTag(gormTag)

		// 跳过忽略的字段
		if parsedTag.Ignore {
			continue
		}

		// 确定列名
		columnName := parsedTag.Column
		if columnName == "" {
			columnName = toSnakeCase(field.Name)
		}

		// 确定类型
		goType := getGoTypeName(field.Type)

		// 确定 SQL 类型
		sqlType := parsedTag.Type
		if sqlType == "" {
			sqlType = dialect.TypeMapping(goType, parsedTag.Size)
		}

		// 获取字段值
		var fieldValue interface{}
		var isZero bool
		if v.IsValid() && v.Field(i).IsValid() {
			fieldValue = v.Field(i).Interface()
			isZero = isZeroValue(v.Field(i))
		} else {
			isZero = true
		}

		fields = append(fields, FieldInfo{
			Name:       field.Name,
			ColumnName: columnName,
			Type:       goType,
			SQLType:    sqlType,
			Tag:        parsedTag,
			Value:      fieldValue,
			IsZero:     isZero,
			Index:      i,
		})
	}

	return fields
}

// getGoTypeName 获取 Go 类型名称的字符串表示
func getGoTypeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Ptr:
		return "*" + getGoTypeName(t.Elem())
	case reflect.Slice:
		return "[]" + getGoTypeName(t.Elem())
	case reflect.Map:
		return "map[" + getGoTypeName(t.Key()) + "]" + getGoTypeName(t.Elem())
	default:
		if t.PkgPath() != "" {
			// 外部包类型，返回完整路径
			return t.PkgPath() + "." + t.Name()
		}
		return t.String()
	}
}

// isZeroValue 判断值是否为零值
func isZeroValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Struct:
		// 对于 time.Time 等结构体，使用 IsZero 方法
		if v.Type().String() == "time.Time" {
			return v.MethodByName("IsZero").Call(nil)[0].Bool()
		}
		return false
	default:
		return false
	}
}

// ============================================================================
// 检测特殊字段
// ============================================================================

// hasSoftDelete 检测模型是否包含软删除字段
func hasSoftDelete(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 检查字段名
		if field.Name == "DeletedAt" {
			return true
		}

		// 检查 gorm tag 中的 column
		gormTag := field.Tag.Get("gorm")
		if strings.Contains(gormTag, "column:deleted_at") {
			return true
		}

		// 检查嵌入的 gorm.DeletedAt
		if field.Type.String() == "gorm.DeletedAt" {
			return true
		}
	}

	return false
}

// getPrimaryKeyField 获取主键字段
func getPrimaryKeyField(fields []FieldInfo) *FieldInfo {
	for i := range fields {
		if fields[i].Tag.PrimaryKey {
			return &fields[i]
		}
	}

	// 默认查找 ID 字段
	for i := range fields {
		if strings.ToLower(fields[i].Name) == "id" {
			return &fields[i]
		}
	}

	return nil
}

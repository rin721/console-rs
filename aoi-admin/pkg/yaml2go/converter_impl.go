package yaml2go

// 本文件属于 YAML 配置代码生成器，把配置样例解析为可编译的 Go 结构体、标签和辅助方法。

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/dave/jennifer/jen"
	"gopkg.in/yaml.v3"
)

// converter YAML 转 Go 结构体转换器的实现
type converter struct {
	config *Config
	mu     sync.RWMutex
}

// newConverter 创建转换器实例（内部使用）
func newConverter(config *Config) *converter {
	c := &converter{
		config: normalizeConfig(config),
	}
	return c
}

// normalizeConfig 规范化配置，填充默认值
func normalizeConfig(config *Config) *Config {
	if config == nil {
		config = &Config{}
	}

	// 设置默认包名
	if config.PackageName == "" {
		config.PackageName = DefaultPackageName
	}

	// 设置默认结构体名
	if config.StructName == "" {
		config.StructName = DefaultStructName
	}

	// 设置默认标签
	if len(config.Tags) == 0 {
		config.Tags = copyStringSlice(DefaultTags)
	}

	// 设置默认缩进风格
	if config.IndentStyle == "" {
		config.IndentStyle = DefaultIndentStyle
	}

	// 该版本默认生成辅助方法，以保证拆分后的配置结构仍然具备加载、校验和环境变量绑定入口。
	if !config.GenerateMethods {
		config.GenerateMethods = true
	}

	// 当前生成链路以多文件输出为主路径，主配置和子配置分离可以降低单文件体积并保留模块边界。
	// 如果未来恢复旧版单文件输出，需要同步补齐模板、测试和调用侧对 GenerateResult 的解释。
	config.SplitFiles = true // 强制启用新模式

	return config
}

// Convert 实现 Converter.Convert
func (c *converter) Convert(yamlStr string) (*GenerateResult, error) {
	// 1. 验证输入
	if strings.TrimSpace(yamlStr) == "" {
		return nil, ErrEmptyInput
	}

	// 2. 解析 YAML
	var data interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &data); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidYAML, err)
	}

	// 3. 获取配置
	c.mu.RLock()
	cfg := c.config
	c.mu.RUnlock()

	// 4. 检查是否分离文件
	if !cfg.SplitFiles {
		// 兼容模式：生成单个文件
		return c.convertLegacy(data, cfg)
	}

	// 5. 新模式：生成多个文件
	return c.convertMultiFile(data, cfg)
}

// convertLegacy 兼容模式：生成单个文件（保持向后兼容）
func (c *converter) convertLegacy(data interface{}, cfg *Config) (*GenerateResult, error) {
	// 构建结构体信息
	structInfo, err := c.buildStructInfo(data, cfg.StructName)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTypeInference, err)
	}
	structInfo.PackageName = cfg.PackageName

	// 生成代码
	code, err := c.generateCode(structInfo)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCodeGeneration, err)
	}

	// 返回单文件结果
	result := &GenerateResult{
		PackageName: cfg.PackageName,
		MainConfig: &FileContent{
			FileName:   "config.go",
			Content:    code,
			ConfigName: "",
			StructName: cfg.StructName,
		},
		SubConfigs: nil,
	}

	return result, nil
}

// convertMultiFile 新模式：为每个顶级配置生成独立文件
func (c *converter) convertMultiFile(data interface{}, cfg *Config) (*GenerateResult, error) {
	// 确保根数据是 map
	rootMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("root element must be an object, got %T", data)
	}

	result := &GenerateResult{
		PackageName: cfg.PackageName,
		SubConfigs:  make([]*FileContent, 0, len(rootMap)),
	}

	// 为每个顶级配置生成子配置文件
	for configName, configValue := range rootMap {
		subConfig, err := c.generateSubConfig(configName, configValue, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to generate config for %s: %w", configName, err)
		}
		result.SubConfigs = append(result.SubConfigs, subConfig)
	}

	// 生成主配置文件
	mainConfig, err := c.generateMainConfig(rootMap, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to generate main config: %w", err)
	}
	result.MainConfig = mainConfig

	return result, nil
}

// SetConfig 实现 Converter.SetConfig
func (c *converter) SetConfig(config *Config) error {
	if config == nil {
		return ErrInvalidConfig
	}

	// 验证配置
	if config.IndentStyle != "" && config.IndentStyle != IndentStyleTab && config.IndentStyle != IndentStyleSpace {
		return fmt.Errorf("%w: invalid indent style: %s", ErrInvalidConfig, config.IndentStyle)
	}

	c.mu.Lock()
	c.config = normalizeConfig(config)
	c.mu.Unlock()

	return nil
}

// buildStructInfo 从 YAML 数据构建结构体信息
func (c *converter) buildStructInfo(data interface{}, structName string) (*StructInfo, error) {
	structInfo := &StructInfo{
		Name:    structName,
		Comment: structName + " 配置结构",
	}

	// 确保根数据是 map
	rootMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("root element must be an object, got %T", data)
	}

	// 构建字段
	for key, value := range rootMap {
		field, err := c.buildFieldInfo(key, value)
		if err != nil {
			return nil, err
		}
		structInfo.Fields = append(structInfo.Fields, field)
	}

	return structInfo, nil
}

// buildFieldInfo 从键值对构建字段信息
func (c *converter) buildFieldInfo(key string, value interface{}) (*FieldInfo, error) {
	c.mu.RLock()
	cfg := c.config
	c.mu.RUnlock()

	field := &FieldInfo{
		Name:         sanitizeFieldName(key),
		OriginalName: key,
		Tags:         make(map[string]string),
		IsPointer:    cfg.UsePointer,
	}

	// 生成标签
	for _, tagName := range cfg.Tags {
		field.Tags[tagName] = key
	}

	// 推断类型
	fieldType, elementType, children, err := c.inferType(value)
	if err != nil {
		return nil, err
	}

	field.Type = fieldType
	field.ElementType = elementType
	field.Children = children

	// 添加注释
	if cfg.AddComments {
		field.Comment = key + " 字段"
	}

	return field, nil
}

// inferType 推断值的类型
func (c *converter) inferType(value interface{}) (FieldType, *FieldInfo, []*FieldInfo, error) {
	if value == nil {
		return TypeInterface, nil, nil, nil
	}

	switch v := value.(type) {
	case string:
		return TypeString, nil, nil, nil

	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return TypeInt, nil, nil, nil

	case float32, float64:
		return TypeFloat, nil, nil, nil

	case bool:
		return TypeBool, nil, nil, nil

	case []interface{}:
		if len(v) == 0 {
			// 空数组，无法推断元素类型
			return TypeSlice, &FieldInfo{Type: TypeInterface}, nil, nil
		}

		// 推断第一个元素的类型
		elemType, elemElementType, elemChildren, err := c.inferType(v[0])
		if err != nil {
			return TypeUnknown, nil, nil, err
		}

		elementInfo := &FieldInfo{
			Type:        elemType,
			ElementType: elemElementType,
			Children:    elemChildren,
		}

		return TypeSlice, elementInfo, nil, nil

	case map[string]interface{}:
		// 嵌套对象
		var children []*FieldInfo
		for key, val := range v {
			child, err := c.buildFieldInfo(key, val)
			if err != nil {
				return TypeUnknown, nil, nil, err
			}
			children = append(children, child)
		}
		return TypeStruct, nil, children, nil

	default:
		// 未知类型，使用 interface{}
		return TypeInterface, nil, nil, nil
	}
}

// generateCode 生成 Go 代码
func (c *converter) generateCode(structInfo *StructInfo) (string, error) {
	f := jen.NewFile(structInfo.PackageName)

	// 生成根结构体
	c.generateStruct(f, structInfo.Name, structInfo.Fields, structInfo.Comment)

	// 生成嵌套结构体
	c.generateNestedStructs(f, structInfo.Fields)

	// 渲染代码
	buf := &bytes.Buffer{}
	if err := f.Render(buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateStruct 生成单个结构体定义
func (c *converter) generateStruct(f *jen.File, name string, fields []*FieldInfo, comment string) {
	c.mu.RLock()
	cfg := c.config
	c.mu.RUnlock()

	// 添加注释
	if comment != "" {
		f.Comment(comment)
	}

	// 构建结构体
	structFields := []jen.Code{}

	for _, field := range fields {
		// 字段注释
		var fieldCode *jen.Statement
		if cfg.AddComments && field.Comment != "" {
			fieldCode = jen.Comment(field.Comment).Line()
		} else {
			fieldCode = jen.Null()
		}

		// 字段定义
		fieldType := c.buildFieldType(field)
		fieldCode = fieldCode.Id(field.Name).Add(fieldType)
		if len(field.Tags) > 0 {
			fieldCode = fieldCode.Tag(buildTagValues(field.Tags, cfg.OmitEmpty))
		}

		structFields = append(structFields, fieldCode)
	}

	f.Type().Id(name).Struct(structFields...)
}

// buildFieldType 构建字段类型
func (c *converter) buildFieldType(field *FieldInfo) jen.Code {
	c.mu.RLock()
	cfg := c.config
	c.mu.RUnlock()

	var typeCode jen.Code

	switch field.Type {
	case TypeString:
		typeCode = jen.String()
	case TypeInt:
		typeCode = jen.Int64()
	case TypeFloat:
		typeCode = jen.Float64()
	case TypeBool:
		typeCode = jen.Bool()
	case TypeInterface:
		typeCode = jen.Interface()
	case TypeSlice:
		elemType := c.buildFieldType(field.ElementType)
		typeCode = jen.Index().Add(elemType)
	case TypeStruct:
		// 内联结构体
		structFields := []jen.Code{}
		for _, child := range field.Children {
			childType := c.buildFieldType(child)

			childCode := jen.Id(child.Name).Add(childType)
			if len(child.Tags) > 0 {
				childCode = childCode.Tag(buildTagValues(child.Tags, cfg.OmitEmpty))
			}
			structFields = append(structFields, childCode)
		}
		typeCode = jen.Struct(structFields...)
	default:
		typeCode = jen.Interface()
	}

	// 添加指针
	if cfg.UsePointer && field.Type != TypeInterface {
		typeCode = jen.Op("*").Add(typeCode)
	}

	return typeCode
}

// generateNestedStructs 生成嵌套结构体（当前实现使用内联结构体，此方法预留）
func (c *converter) generateNestedStructs(f *jen.File, fields []*FieldInfo) {
	// 当前使用内联结构体，不需要额外生成
	// 如果未来需要提取嵌套结构体为独立类型，在此实现
}

// generateMainConfig 生成主配置文件（config.go）
func (c *converter) generateMainConfig(rootMap map[string]interface{}, cfg *Config) (*FileContent, error) {
	f := jen.NewFile(cfg.PackageName)

	// 添加注释
	f.Comment("Config 应用配置")
	f.Comment("此文件由 yaml2go 自动生成，请勿手动修改")

	// 构建主结构体字段
	structFields := []jen.Code{}
	for configName := range rootMap {
		// 生成结构体名称 (如 "server" -> "ServerConfig")
		structName := sanitizeFieldName(configName) + "Config"

		// 创建字段
		fieldCode := jen.Id(sanitizeFieldName(configName)).
			Op("*").Id(structName).
			Tag(buildTagValues(map[string]string{
				"mapstructure": configName,
				"json":         configName,
				"yaml":         configName,
			}, false))

		structFields = append(structFields, fieldCode)
	}

	// 生成主 Config 结构体
	f.Type().Id("Config").Struct(structFields...)

	// 渲染代码
	buf := &bytes.Buffer{}
	if err := f.Render(buf); err != nil {
		return nil, err
	}

	return &FileContent{
		FileName:   "config.go",
		Content:    buf.String(),
		ConfigName: "",
		StructName: "Config",
	}, nil
}

// generateSubConfig 为单个顶级配置生成独立文件
func (c *converter) generateSubConfig(configName string, configValue interface{}, cfg *Config) (*FileContent, error) {
	// 1. 构建结构体信息
	structName := sanitizeFieldName(configName) + "Config"

	// 确保是 map 类型
	configMap, ok := configValue.(map[string]interface{})
	if !ok {
		// 如果不是 map，创建一个简单的值字段
		return c.generateSimpleConfig(configName, configValue, cfg)
	}

	// 2. 构建字段
	var fields []*FieldInfo
	for key, value := range configMap {
		field, err := c.buildFieldInfo(key, value)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}

	// 3. 创建 StructInfo
	structInfo := &StructInfo{
		Name:          structName,
		PackageName:   cfg.PackageName,
		Fields:        fields,
		Comment:       structName + " " + configName + " 配置",
		ConfigName:    configName,
		DefaultValues: extractDefaultValues(configMap),
	}

	// 4. 生成文件（暂时不包含方法，留待阶段3实现）
	code, err := c.generateSubConfigCode(structInfo, cfg)
	if err != nil {
		return nil, err
	}

	// 5. 生成文件名
	fileName := toSnakeCase(configName) + "_config.go"

	return &FileContent{
		FileName:   fileName,
		Content:    code,
		ConfigName: configName,
		StructName: structName,
	}, nil
}

// generateSimpleConfig 为简单类型配置生成代码
func (c *converter) generateSimpleConfig(configName string, configValue interface{}, cfg *Config) (*FileContent, error) {
	structName := sanitizeFieldName(configName) + "Config"
	f := jen.NewFile(cfg.PackageName)

	// 推断类型
	fieldType, _, _, err := c.inferType(configValue)
	if err != nil {
		return nil, err
	}

	// 简单的值包装结构体
	f.Type().Id(structName).Struct(
		jen.Id("Value").Add(jen.Id(fieldType.String())).Tag(buildTagValues(map[string]string{
			"json":         "value",
			"yaml":         "value",
			"mapstructure": "value",
		}, false)),
	)

	buf := &bytes.Buffer{}
	if err := f.Render(buf); err != nil {
		return nil, err
	}

	// ConfigBlockFilenameSuffix 配置块文件名后缀
	fileName := toSnakeCase(configName) + ConfigBlockFilenameSuffix
	return &FileContent{
		FileName:   fileName,
		Content:    buf.String(),
		ConfigName: configName,
		StructName: structName,
	}, nil
}

// generateSubConfigCode 生成子配置的结构体代码（包含方法）
func (c *converter) generateSubConfigCode(structInfo *StructInfo, cfg *Config) (string, error) {
	f := jen.NewFile(structInfo.PackageName)

	// 添加文件注释
	f.Comment("此文件由 yaml2go 自动生成，请勿手动修改")
	f.Line()

	// 生成结构体
	c.generateStruct(f, structInfo.Name, structInfo.Fields, structInfo.Comment)
	if cfg.GenerateMethods {
		c.appendMethods(f, structInfo, cfg)
	}

	// 渲染完整文件代码
	buf := &bytes.Buffer{}
	if err := f.Render(buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// extractDefaultValues 从配置 map 中提取默认值
func extractDefaultValues(configMap map[string]interface{}) map[string]interface{} {
	defaults := make(map[string]interface{})
	for key, value := range configMap {
		// 只保存基础类型的默认值
		switch value.(type) {
		case string, int, int64, float64, bool:
			defaults[key] = value
		}
	}
	return defaults
}

package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

import (
	"fmt"
	"strings"
)

// ============================================================================
// 代码生成器
// ============================================================================

// CodeGenerator Go 代码生成器
type CodeGenerator struct {
	options *ReverseOptions
}

// NewCodeGenerator 创建新的代码生成器
func NewCodeGenerator(opts *ReverseOptions) *CodeGenerator {
	return &CodeGenerator{
		options: opts,
	}
}

// Generate 生成 Go Struct 代码
func (c *CodeGenerator) Generate(schema *Schema) string {
	var sb strings.Builder

	// 包声明
	sb.WriteString(fmt.Sprintf("package %s\n\n", schema.Package))

	// 导入
	if len(schema.Imports) > 0 {
		sb.WriteString("import (\n")
		for _, imp := range schema.Imports {
			sb.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
		}
		sb.WriteString(")\n\n")
	}

	// 结构体注释
	if c.options.WithComments && schema.Comment != "" {
		sb.WriteString(fmt.Sprintf("// %s %s\n", schema.Name, schema.Comment))
	}

	// 结构体定义
	sb.WriteString(fmt.Sprintf("type %s struct {\n", schema.Name))

	// 字段
	for _, field := range schema.Fields {
		c.writeField(&sb, field)
	}

	sb.WriteString("}\n")

	// TableName 方法
	if c.options.WithTableName {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("// TableName overrides the table name\n"))
		sb.WriteString(fmt.Sprintf("func (%s) TableName() string {\n", schema.Name))
		sb.WriteString(fmt.Sprintf("\treturn \"%s\"\n", schema.TableName))
		sb.WriteString("}\n")
	}

	return sb.String()
}

// writeField 写入字段定义
func (c *CodeGenerator) writeField(sb *strings.Builder, field Field) {
	// 字段注释
	if c.options.WithComments && field.Comment != "" {
		sb.WriteString(fmt.Sprintf("\t// %s %s\n", field.Name, field.Comment))
	}

	// 字段名和类型
	sb.WriteString(fmt.Sprintf("\t%s %s", field.Name, field.Type))

	// 构建 struct tag 字符串。
	tags := c.buildTags(field)
	if tags != "" {
		sb.WriteString(fmt.Sprintf(" `%s`", tags))
	}

	sb.WriteString("\n")
}

// buildTags 构建 struct tags
func (c *CodeGenerator) buildTags(field Field) string {
	var tags []string

	// 生成 GORM tag。
	if c.options.Tags&TagGorm != 0 {
		gormTag := c.buildGormTag(field)
		if gormTag != "" {
			tags = append(tags, fmt.Sprintf("gorm:\"%s\"", gormTag))
		}
	}

	// 生成 JSON tag。
	if c.options.Tags&TagJson != 0 {
		jsonName := convertNaming(field.Column.Name, c.options.JSONNaming)
		tags = append(tags, fmt.Sprintf("json:\"%s\"", jsonName))
	}

	// 生成 XML tag。
	if c.options.Tags&TagXml != 0 {
		xmlName := convertNaming(field.Column.Name, c.options.JSONNaming)
		tags = append(tags, fmt.Sprintf("xml:\"%s\"", xmlName))
	}

	// 生成 YAML tag。
	if c.options.Tags&TagYaml != 0 {
		yamlName := convertNaming(field.Column.Name, c.options.JSONNaming)
		tags = append(tags, fmt.Sprintf("yaml:\"%s\"", yamlName))
	}

	// 非空且非自增字段才需要生成 validate required。
	if c.options.Tags&TagValidate != 0 {
		if field.Column.NotNull && !field.Column.AutoIncrement {
			tags = append(tags, "validate:\"required\"")
		}
	}

	return strings.Join(tags, " ")
}

// buildGormTag 构建 GORM tag
func (c *CodeGenerator) buildGormTag(field Field) string {
	var parts []string

	// column 指定数据库列名。
	parts = append(parts, fmt.Sprintf("column:%s", field.Column.Name))

	// type 保留数据库原始列类型，便于 GORM 迁移时复用。
	if field.Column.Type != "" {
		parts = append(parts, fmt.Sprintf("type:%s", field.Column.Type))
	}

	// primaryKey 标记主键字段。
	if field.Column.PrimaryKey {
		parts = append(parts, "primaryKey")
	}

	// autoIncrement 标记自增字段。
	if field.Column.AutoIncrement {
		parts = append(parts, "autoIncrement")
	}

	// not null 只在非主键字段上补充，避免与主键约束重复表达。
	if field.Column.NotNull && !field.Column.PrimaryKey {
		parts = append(parts, "not null")
	}

	// default 保留数据库默认值表达式。
	if field.Column.Default != "" {
		parts = append(parts, fmt.Sprintf("default:%s", field.Column.Default))
	}

	// size 只对 VARCHAR 类型有意义。
	if field.Column.Size > 0 && strings.Contains(strings.ToUpper(field.Column.Type), "VARCHAR") {
		parts = append(parts, fmt.Sprintf("size:%d", field.Column.Size))
	}

	// comment 在启用注释生成时写入 GORM tag。
	if field.Column.Comment != "" && c.options.WithComments {
		parts = append(parts, fmt.Sprintf("comment:%s", field.Column.Comment))
	}

	return strings.Join(parts, ";")
}

// ============================================================================
// DAO 代码生成
// ============================================================================

// GenerateDAO 生成 DAO 层代码
func (c *CodeGenerator) GenerateDAO(schema *Schema, methods []string) string {
	var sb strings.Builder

	daoName := schema.Name + "DAO"

	// 包声明
	sb.WriteString(fmt.Sprintf("package %s\n\n", schema.Package))

	// 导入
	sb.WriteString("import (\n")
	sb.WriteString("\t\"gorm.io/gorm\"\n")
	sb.WriteString(")\n\n")

	// DAO 结构体
	sb.WriteString(fmt.Sprintf("// %s 数据访问对象\n", daoName))
	sb.WriteString(fmt.Sprintf("type %s struct {\n", daoName))
	sb.WriteString("\tdb *gorm.DB\n")
	sb.WriteString("}\n\n")

	// 构造函数
	sb.WriteString(fmt.Sprintf("// New%s 创建新的 DAO 实例\n", daoName))
	sb.WriteString(fmt.Sprintf("func New%s(db *gorm.DB) *%s {\n", daoName, daoName))
	sb.WriteString(fmt.Sprintf("\treturn &%s{db: db}\n", daoName))
	sb.WriteString("}\n\n")

	// 生成方法
	for _, method := range methods {
		switch method {
		case "Create":
			c.writeCreateMethod(&sb, schema, daoName)
		case "Update":
			c.writeUpdateMethod(&sb, schema, daoName)
		case "Delete":
			c.writeDeleteMethod(&sb, schema, daoName)
		case "FindByID":
			c.writeFindByIDMethod(&sb, schema, daoName)
		case "FindAll":
			c.writeFindAllMethod(&sb, schema, daoName)
		}
	}

	return sb.String()
}

// writeCreateMethod 向代码生成缓冲区写入 DAO 方法模板，保持模板顺序和缩进可预测。
func (c *CodeGenerator) writeCreateMethod(sb *strings.Builder, schema *Schema, daoName string) {
	sb.WriteString(fmt.Sprintf("// Create 创建记录\n"))
	sb.WriteString(fmt.Sprintf("func (d *%s) Create(entity *%s) error {\n", daoName, schema.Name))
	sb.WriteString("\treturn d.db.Create(entity).Error\n")
	sb.WriteString("}\n\n")
}

// writeUpdateMethod 向代码生成缓冲区写入 DAO 方法模板，保持模板顺序和缩进可预测。
func (c *CodeGenerator) writeUpdateMethod(sb *strings.Builder, schema *Schema, daoName string) {
	sb.WriteString(fmt.Sprintf("// Update 更新记录\n"))
	sb.WriteString(fmt.Sprintf("func (d *%s) Update(entity *%s) error {\n", daoName, schema.Name))
	sb.WriteString("\treturn d.db.Save(entity).Error\n")
	sb.WriteString("}\n\n")
}

// writeDeleteMethod 向代码生成缓冲区写入 DAO 方法模板，保持模板顺序和缩进可预测。
func (c *CodeGenerator) writeDeleteMethod(sb *strings.Builder, schema *Schema, daoName string) {
	// 查找主键字段
	var pkField *Field
	for i := range schema.Fields {
		if schema.Fields[i].Column.PrimaryKey {
			pkField = &schema.Fields[i]
			break
		}
	}

	pkType := "uint64"
	if pkField != nil {
		pkType = pkField.Type
	}

	sb.WriteString(fmt.Sprintf("// Delete 删除记录\n"))
	sb.WriteString(fmt.Sprintf("func (d *%s) Delete(id %s) error {\n", daoName, pkType))
	sb.WriteString(fmt.Sprintf("\treturn d.db.Delete(&%s{}, id).Error\n", schema.Name))
	sb.WriteString("}\n\n")
}

// writeFindByIDMethod 向代码生成缓冲区写入 DAO 方法模板，保持模板顺序和缩进可预测。
func (c *CodeGenerator) writeFindByIDMethod(sb *strings.Builder, schema *Schema, daoName string) {
	// 查找主键字段
	var pkField *Field
	for i := range schema.Fields {
		if schema.Fields[i].Column.PrimaryKey {
			pkField = &schema.Fields[i]
			break
		}
	}

	pkType := "uint64"
	if pkField != nil {
		pkType = pkField.Type
	}

	sb.WriteString(fmt.Sprintf("// FindByID 根据 ID 查找记录\n"))
	sb.WriteString(fmt.Sprintf("func (d *%s) FindByID(id %s) (*%s, error) {\n", daoName, pkType, schema.Name))
	sb.WriteString(fmt.Sprintf("\tvar entity %s\n", schema.Name))
	sb.WriteString("\tif err := d.db.First(&entity, id).Error; err != nil {\n")
	sb.WriteString("\t\treturn nil, err\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn &entity, nil\n")
	sb.WriteString("}\n\n")
}

// writeFindAllMethod 向代码生成缓冲区写入 DAO 方法模板，保持模板顺序和缩进可预测。
func (c *CodeGenerator) writeFindAllMethod(sb *strings.Builder, schema *Schema, daoName string) {
	sb.WriteString(fmt.Sprintf("// FindAll 查找所有记录\n"))
	sb.WriteString(fmt.Sprintf("func (d *%s) FindAll() ([]*%s, error) {\n", daoName, schema.Name))
	sb.WriteString(fmt.Sprintf("\tvar entities []*%s\n", schema.Name))
	sb.WriteString("\tif err := d.db.Find(&entities).Error; err != nil {\n")
	sb.WriteString("\t\treturn nil, err\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn entities, nil\n")
	sb.WriteString("}\n\n")
}

package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

import (
	"os"
	"path/filepath"
	"strings"
)

// ============================================================================
// 逆向生成入口
// ============================================================================

// ParseSQL 从 SQL DDL 字符串解析表结构
func (g *Generator) ParseSQL(ddl string) *ReverseBuilder {
	parser := NewParser(g.config.Dialect)
	schemas, _ := parser.Parse(ddl)

	return &ReverseBuilder{
		generator: g,
		schemas:   schemas,
		options:   DefaultReverseOptions(),
	}
}

// ParseSQLFile 从 SQL 文件解析表结构
func (g *Generator) ParseSQLFile(path string) *ReverseBuilder {
	content, err := os.ReadFile(path)
	if err != nil {
		return &ReverseBuilder{
			generator: g,
			err:       WrapError(ErrCodeFileIO, "failed to read file", err),
			options:   DefaultReverseOptions(),
		}
	}

	return g.ParseSQL(string(content))
}

// ============================================================================
// ReverseBuilder 逆向生成构建器
// ============================================================================

// ReverseBuilder 逆向生成构建器 (类 GORM 风格链式调用)
type ReverseBuilder struct {
	generator     *Generator
	schemas       []*Schema
	options       *ReverseOptions
	err           error
	daoMethods    []string // DAO 方法列表
	mergeFilePath string   // 增量更新文件路径
}

// Name 设置生成的结构体名称
func (r *ReverseBuilder) Name(name string) *ReverseBuilder {
	r.options.StructName = name
	return r
}

// Package 设置包名
func (r *ReverseBuilder) Package(pkg string) *ReverseBuilder {
	r.options.Package = pkg
	return r
}

// Tags 设置要生成的 Tag 类型
func (r *ReverseBuilder) Tags(flags TagType) *ReverseBuilder {
	r.options.Tags = flags
	return r
}

// JSONTagNaming 设置 JSON Tag 命名策略
func (r *ReverseBuilder) JSONTagNaming(strategy NamingStrategy) *ReverseBuilder {
	r.options.JSONNaming = strategy
	return r
}

// FieldNaming 设置字段命名策略
func (r *ReverseBuilder) FieldNaming(strategy NamingStrategy) *ReverseBuilder {
	r.options.FieldNaming = strategy
	return r
}

// TypeMapping 添加类型映射
func (r *ReverseBuilder) TypeMapping(sqlType, goType string) *ReverseBuilder {
	r.options.TypeMappings[sqlType] = goType
	return r
}

// TypeMappings 批量添加类型映射
func (r *ReverseBuilder) TypeMappings(mappings map[string]string) *ReverseBuilder {
	for k, v := range mappings {
		r.options.TypeMappings[k] = v
	}
	return r
}

// WithComments 是否生成注释
func (r *ReverseBuilder) WithComments(enabled bool) *ReverseBuilder {
	r.options.WithComments = enabled
	return r
}

// WithTableName 是否生成 TableName() 方法
func (r *ReverseBuilder) WithTableName(enabled bool) *ReverseBuilder {
	r.options.WithTableName = enabled
	return r
}

// WithSoftDelete 是否识别软删除字段
func (r *ReverseBuilder) WithSoftDelete(enabled bool) *ReverseBuilder {
	r.options.WithSoftDelete = enabled
	return r
}

// Import 添加额外导入的包
func (r *ReverseBuilder) Import(packages ...string) *ReverseBuilder {
	r.options.Imports = append(r.options.Imports, packages...)
	return r
}

// Template 设置自定义模板
func (r *ReverseBuilder) Template(tmpl string) *ReverseBuilder {
	r.options.Template = tmpl
	return r
}

// FieldConverter 设置自定义字段转换器
func (r *ReverseBuilder) FieldConverter(fn func(col Column) Field) *ReverseBuilder {
	r.options.FieldConverter = fn
	return r
}

// BeforeGenerate 设置生成前钩子
func (r *ReverseBuilder) BeforeGenerate(fn func(schema *Schema)) *ReverseBuilder {
	r.options.BeforeGenerate = fn
	return r
}

// AfterGenerate 设置生成后钩子
func (r *ReverseBuilder) AfterGenerate(fn func(code string) string) *ReverseBuilder {
	r.options.AfterGenerate = fn
	return r
}

// FileNaming 设置文件命名策略
func (r *ReverseBuilder) FileNaming(strategy NamingStrategy) *ReverseBuilder {
	r.options.FileNaming = strategy
	return r
}

// Overwrite 是否覆盖已存在的文件
func (r *ReverseBuilder) Overwrite(enabled bool) *ReverseBuilder {
	r.options.Overwrite = enabled
	return r
}

// Dialect 设置方言
func (r *ReverseBuilder) Dialect(d Dialect) *ReverseBuilder {
	r.generator.config.Dialect = d
	return r
}

// ============================================================================
// 生成方法
// ============================================================================

// Generate 生成单个 Go Struct 代码
func (r *ReverseBuilder) Generate() (string, error) {
	if r.err != nil {
		return "", r.err
	}

	if len(r.schemas) == 0 {
		return "", ErrParseFailed
	}

	// 使用第一个 Schema
	schema := r.schemas[0]

	// 应用自定义名称
	if r.options.StructName != "" {
		schema.Name = r.options.StructName
	}

	return r.generateCode(schema)
}

// GenerateAll 生成所有表的 Go Struct 代码
func (r *ReverseBuilder) GenerateAll() (map[string]string, error) {
	if r.err != nil {
		return nil, r.err
	}

	result := make(map[string]string)

	for _, schema := range r.schemas {
		code, err := r.generateCode(schema)
		if err != nil {
			continue
		}
		result[schema.TableName] = code
	}

	return result, nil
}

// GenerateToFile 生成代码到单个文件
func (r *ReverseBuilder) GenerateToFile(path string) error {
	code, err := r.Generate()
	if err != nil {
		return err
	}

	// 检查文件是否存在
	if !r.options.Overwrite {
		if _, err := os.Stat(path); err == nil {
			return WrapError(ErrCodeFileIO, "file already exists", nil)
		}
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return WrapError(ErrCodeFileIO, "failed to create directory", err)
	}

	return os.WriteFile(path, []byte(code), 0644)
}

// GenerateToDir 生成代码到目录 (每个表一个文件)
func (r *ReverseBuilder) GenerateToDir(dir string) error {
	if r.err != nil {
		return r.err
	}

	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return WrapError(ErrCodeFileIO, "failed to create directory", err)
	}

	for _, schema := range r.schemas {
		code, err := r.generateCode(schema)
		if err != nil {
			continue
		}

		// 生成文件名
		filename := convertNaming(schema.TableName, r.options.FileNaming) + ".go"
		filepath := filepath.Join(dir, filename)

		// 检查文件是否存在
		if !r.options.Overwrite {
			if _, err := os.Stat(filepath); err == nil {
				continue
			}
		}

		if err := os.WriteFile(filepath, []byte(code), 0644); err != nil {
			return WrapError(ErrCodeFileIO, "failed to write file", err)
		}
	}

	return nil
}

// ============================================================================
// 内部方法
// ============================================================================

// generateCode 根据 schema 和 ReverseOptions 渲染 Go 结构体源码，是逆向生成的最终出口。
func (r *ReverseBuilder) generateCode(schema *Schema) (string, error) {
	// 应用类型映射
	for i := range schema.Fields {
		if mappedType, ok := r.options.TypeMappings[schema.Fields[i].Column.Type]; ok {
			schema.Fields[i].Type = mappedType
		}
	}

	// 应用字段转换器
	if r.options.FieldConverter != nil {
		for i := range schema.Fields {
			schema.Fields[i] = r.options.FieldConverter(schema.Fields[i].Column)
		}
	}

	// 调用 BeforeGenerate 钩子
	if r.options.BeforeGenerate != nil {
		r.options.BeforeGenerate(schema)
	}

	// 合并导入
	allImports := make(map[string]bool)
	for _, imp := range schema.Imports {
		allImports[imp] = true
	}
	for _, imp := range r.options.Imports {
		allImports[imp] = true
	}

	var imports []string
	for imp := range allImports {
		imports = append(imports, imp)
	}
	schema.Imports = imports

	// 设置包名
	schema.Package = r.options.Package

	// 生成代码
	codegen := NewCodeGenerator(r.options)
	code := codegen.Generate(schema)

	// 调用 AfterGenerate 钩子
	if r.options.AfterGenerate != nil {
		code = r.options.AfterGenerate(code)
	}

	return code, nil
}

// ============================================================================
// 数据库逆向 (可选功能)
// ============================================================================

// ReverseDB 从数据库连接读取 Schema
// 注意: 需要传入 *sql.DB 连接
func (g *Generator) ReverseDB(db interface{}) *DBReverseBuilder {
	return &DBReverseBuilder{
		generator: g,
		db:        db,
		options:   DefaultReverseOptions(),
	}
}

// DBReverseBuilder 数据库逆向构建器
type DBReverseBuilder struct {
	generator *Generator
	db        interface{}
	tables    []string
	excludes  []string
	includes  []string
	allTables bool
	options   *ReverseOptions
}

// Table 指定表名
func (d *DBReverseBuilder) Table(name string) *DBReverseBuilder {
	d.tables = []string{name}
	return d
}

// Tables 指定多个表名
func (d *DBReverseBuilder) Tables(names ...string) *DBReverseBuilder {
	d.tables = names
	return d
}

// AllTables 处理所有表
func (d *DBReverseBuilder) AllTables() *DBReverseBuilder {
	d.allTables = true
	return d
}

// Exclude 排除匹配的表
func (d *DBReverseBuilder) Exclude(patterns ...string) *DBReverseBuilder {
	d.excludes = patterns
	return d
}

// Include 仅包含匹配的表
func (d *DBReverseBuilder) Include(patterns ...string) *DBReverseBuilder {
	d.includes = patterns
	return d
}

// Package 设置包名
func (d *DBReverseBuilder) Package(pkg string) *DBReverseBuilder {
	d.options.Package = pkg
	return d
}

// FileNaming 设置文件命名策略
func (d *DBReverseBuilder) FileNaming(strategy NamingStrategy) *DBReverseBuilder {
	d.options.FileNaming = strategy
	return d
}

// Generate 生成代码
func (d *DBReverseBuilder) Generate() (string, error) {
	return "", NewUnsupportedError("ReverseDB.Generate")
}

// GenerateAll 生成所有表的代码
func (d *DBReverseBuilder) GenerateAll() (map[string]string, error) {
	return nil, NewUnsupportedError("ReverseDB.GenerateAll")
}

// GenerateToDir 生成代码到目录
func (d *DBReverseBuilder) GenerateToDir(dir string) error {
	return NewUnsupportedError("ReverseDB.GenerateToDir")
}

// matchPattern 匹配通配符模式
func matchPattern(name, pattern string) bool {
	// 简单的通配符匹配
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(name, prefix)
	}
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(name, suffix)
	}
	return name == pattern
}

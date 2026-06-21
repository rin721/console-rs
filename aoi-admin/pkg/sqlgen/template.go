package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

import (
	"bytes"
	"text/template"
)

// ============================================================================
// 模板引擎
// ============================================================================

// DefaultStructTemplate 默认的结构体生成模板
const DefaultStructTemplate = `package {{.Package}}
{{if .Imports}}
import (
{{range .Imports}}	"{{.}}"
{{end}})
{{end}}
{{if and .Comment $.WithComments}}// {{.Name}} {{.Comment}}
{{end}}type {{.Name}} struct {
{{range .Fields}}{{if and .Comment $.WithComments}}	// {{.Name}} {{.Comment}}
{{end}}	{{.Name}} {{.Type}}{{if .Tags}} ` + "`{{.Tags}}`" + `{{end}}
{{end}}}
{{if $.WithTableName}}
// TableName overrides the table name
func ({{.Name}}) TableName() string {
	return "{{.TableName}}"
}
{{end}}`

// DefaultDAOTemplate 默认的 DAO 层生成模板
const DefaultDAOTemplate = `package {{.Package}}

import (
	"gorm.io/gorm"
)

// {{.Name}}DAO 数据访问对象
type {{.Name}}DAO struct {
	db *gorm.DB
}

// New{{.Name}}DAO 创建新的 DAO 实例
func New{{.Name}}DAO(db *gorm.DB) *{{.Name}}DAO {
	return &{{.Name}}DAO{db: db}
}

{{range .Methods}}{{.}}
{{end}}`

// ============================================================================
// TemplateData 模板数据
// ============================================================================

// TemplateData 传递给模板的数据结构
type TemplateData struct {
	*Schema
	*ReverseOptions
	Methods []string
}

// ============================================================================
// 模板渲染
// ============================================================================

// RenderTemplate 使用模板渲染代码
func RenderTemplate(tmplStr string, data *TemplateData) (string, error) {
	tmpl, err := template.New("sqlgen").Parse(tmplStr)
	if err != nil {
		return "", WrapError(ErrCodeGenerateFailed, "failed to parse template", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", WrapError(ErrCodeGenerateFailed, "failed to execute template", err)
	}

	return buf.String(), nil
}

// ============================================================================
// ReverseBuilder 模板支持增强
// ============================================================================

// WithDAO 启用 DAO 层生成
func (r *ReverseBuilder) WithDAO(enabled bool) *ReverseBuilder {
	if enabled {
		r.options.Template = DefaultDAOTemplate
	}
	return r
}

// DAOMethods 设置要生成的 DAO 方法
func (r *ReverseBuilder) DAOMethods(methods ...string) *ReverseBuilder {
	// 存储方法名，在生成时使用
	r.daoMethods = methods
	return r
}

// GenerateWithDAO 生成结构体和 DAO 代码
func (r *ReverseBuilder) GenerateWithDAO() (structCode string, daoCode string, err error) {
	if r.err != nil {
		return "", "", r.err
	}

	if len(r.schemas) == 0 {
		return "", "", ErrParseFailed
	}

	schema := r.schemas[0]

	// 生成结构体
	structCode, err = r.generateCode(schema)
	if err != nil {
		return "", "", err
	}

	// 生成 DAO
	codegen := NewCodeGenerator(r.options)
	daoCode = codegen.GenerateDAO(schema, r.daoMethods)

	return structCode, daoCode, nil
}

// ============================================================================
// 增量更新支持
// ============================================================================

// MergeWithFile 读取现有文件并合并
func (r *ReverseBuilder) MergeWithFile(path string) *ReverseBuilder {
	r.mergeFilePath = path
	return r
}

// ============================================================================
// ReverseBuilder 扩展字段
// ============================================================================

// 在 ReverseBuilder 结构体中添加字段 (需要更新 reverse.go)
// daoMethods []string：DAO 方法名列表。
// mergeFilePath string：待合并的既有文件路径。

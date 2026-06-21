package sqlgen

// 本文件属于 SQL 生成器，负责把结构体、schema 或解析结果转换为特定方言的 SQL 文本。

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ============================================================================
// SQL DDL 解析器
// ============================================================================

// Parser SQL 解析器
type Parser struct {
	dialect Dialect
	input   string
	pos     int
}

// NewParser 创建新的解析器
func NewParser(dialect Dialect) *Parser {
	return &Parser{
		dialect: dialect,
	}
}

// Parse 解析 SQL DDL 脚本
func (p *Parser) Parse(sql string) ([]*Schema, error) {
	p.input = sql
	p.pos = 0

	var schemas []*Schema

	// 查找所有 CREATE TABLE 语句
	tables := p.findCreateTableStatements()

	for _, tableSQL := range tables {
		schema, err := p.parseCreateTable(tableSQL)
		if err != nil {
			continue // 跳过解析失败的表
		}
		schemas = append(schemas, schema)
	}

	return schemas, nil
}

// ParseSingle 解析单个 CREATE TABLE 语句
func (p *Parser) ParseSingle(sql string) (*Schema, error) {
	return p.parseCreateTable(sql)
}

// ============================================================================
// CREATE TABLE 解析
// ============================================================================

// 正则表达式
var (
	// 匹配 CREATE TABLE 语句头。列定义主体使用括号深度扫描解析，避免被 varchar(64) 之类的类型截断。
	createTableStartRegex = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?[` + "`" + `"'\[]?(\w+)[` + "`" + `"'\]]?\s*\(`)

	// 匹配列定义
	columnDefRegex = regexp.MustCompile(`(?i)^[` + "`" + `"'\[]?(\w+)[` + "`" + `"'\]]?\s+(\w+(?:\([^)]+\))?(?:\s+\w+)*)\s*(.*)$`)

	// 匹配数据类型
	dataTypeRegex = regexp.MustCompile(`(?i)^(\w+)(?:\(([^)]+)\))?`)

	// 匹配 COMMENT
	commentRegex = regexp.MustCompile(`(?i)COMMENT\s+['"]([^'"]+)['"]`)

	// 匹配 DEFAULT
	defaultRegex = regexp.MustCompile(`(?i)DEFAULT\s+(['"]?[^,'"]+['"]?)`)

	// 匹配 PRIMARY KEY 约束
	pkConstraintRegex = regexp.MustCompile(`(?i)(?:CONSTRAINT\s+\w+\s+)?PRIMARY\s+KEY\s*\(([^)]+)\)`)
)

// findCreateTableStatements 在 SQL 文本中定位语法边界，为后续解析步骤提供稳定切片范围。
func (p *Parser) findCreateTableStatements() []string {
	var results []string

	offset := 0
	for offset < len(p.input) {
		loc := createTableStartRegex.FindStringIndex(p.input[offset:])
		if loc == nil {
			break
		}

		start := offset + loc[0]
		openParen := offset + loc[1] - 1
		closeParen := findMatchingParen(p.input, openParen)
		if closeParen < 0 {
			break
		}

		end := closeParen + 1
		for end < len(p.input) && p.input[end] != ';' {
			end++
		}
		if end < len(p.input) && p.input[end] == ';' {
			end++
		}

		results = append(results, p.input[start:end])
		offset = end
	}

	return results
}

// parseCreateTable 从 SQL 或标签文本中提取结构化信息，无法识别时返回错误或保守默认值。
func (p *Parser) parseCreateTable(sql string) (*Schema, error) {
	matches := createTableStartRegex.FindStringSubmatchIndex(sql)
	if len(matches) < 4 {
		return nil, ErrParseFailed
	}

	tableName := sql[matches[2]:matches[3]]
	openParen := matches[1] - 1
	closeParen := findMatchingParen(sql, openParen)
	if closeParen < 0 {
		return nil, ErrParseFailed
	}
	columnsBody := sql[openParen+1 : closeParen]

	schema := &Schema{
		Name:      toPascalCase(singularizeTableName(tableName)),
		TableName: tableName,
	}

	// 解析列定义
	columns := p.splitColumns(columnsBody)

	var primaryKeys []string

	for _, colDef := range columns {
		colDef = strings.TrimSpace(colDef)
		if colDef == "" {
			continue
		}

		// 检查是否是约束定义
		if strings.HasPrefix(strings.ToUpper(colDef), "PRIMARY KEY") ||
			strings.HasPrefix(strings.ToUpper(colDef), "CONSTRAINT") {
			// 提取主键列
			if pkMatch := pkConstraintRegex.FindStringSubmatch(colDef); len(pkMatch) > 1 {
				pkCols := strings.Split(pkMatch[1], ",")
				for _, pk := range pkCols {
					primaryKeys = append(primaryKeys, strings.Trim(strings.TrimSpace(pk), "`\"'[]"))
				}
			}
			continue
		}

		// 跳过其他约束
		upper := strings.ToUpper(colDef)
		if strings.HasPrefix(upper, "INDEX") ||
			strings.HasPrefix(upper, "KEY") ||
			strings.HasPrefix(upper, "UNIQUE") ||
			strings.HasPrefix(upper, "FOREIGN") ||
			strings.HasPrefix(upper, "CHECK") {
			continue
		}

		// 解析列
		col, err := p.parseColumnDef(colDef)
		if err != nil {
			continue
		}

		schema.Fields = append(schema.Fields, *col)
	}

	// 标记主键
	for i := range schema.Fields {
		for _, pk := range primaryKeys {
			if strings.EqualFold(schema.Fields[i].Column.Name, pk) {
				schema.Fields[i].Column.PrimaryKey = true
			}
		}
	}

	// 检查需要导入的包
	p.analyzeImports(schema)

	return schema, nil
}

// findMatchingParen 在 SQL 文本中定位语法边界，为后续解析步骤提供稳定切片范围。
func findMatchingParen(input string, openIndex int) int {
	if openIndex < 0 || openIndex >= len(input) || input[openIndex] != '(' {
		return -1
	}

	depth := 0
	inQuote := rune(0)
	escaped := false
	for i, ch := range input[openIndex:] {
		idx := openIndex + i
		if inQuote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}

		switch ch {
		case '\'', '"':
			inQuote = ch
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return idx
			}
		}
	}
	return -1
}

// singularizeTableName 将常见复数表名转换为结构体候选名，复杂命名仍允许调用方通过选项覆盖。
func singularizeTableName(name string) string {
	if strings.HasSuffix(name, "ies") && len(name) > 3 {
		return strings.TrimSuffix(name, "ies") + "y"
	}
	if strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss") && len(name) > 1 {
		return strings.TrimSuffix(name, "s")
	}
	return name
}

// splitColumns 分割列定义 (处理嵌套括号)
func (p *Parser) splitColumns(body string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, ch := range body {
		switch ch {
		case '(':
			depth++
			current.WriteRune(ch)
		case ')':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// parseColumnDef 解析列定义
func (p *Parser) parseColumnDef(def string) (*Field, error) {
	def = strings.TrimSpace(def)

	// 提取列名和其余部分
	parts := strings.Fields(def)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid column definition: %s", def)
	}

	columnName := strings.Trim(parts[0], "`\"'[]")
	restDef := strings.Join(parts[1:], " ")

	// 解析数据类型
	typeMatch := dataTypeRegex.FindStringSubmatch(restDef)
	if len(typeMatch) < 2 {
		return nil, fmt.Errorf("cannot parse data type: %s", def)
	}

	sqlType := parts[1]
	baseType := strings.ToUpper(typeMatch[1])

	// 解析类型参数
	var size, precision, scale int
	if len(typeMatch) > 2 && typeMatch[2] != "" {
		params := strings.Split(typeMatch[2], ",")
		if len(params) >= 1 {
			size, _ = strconv.Atoi(strings.TrimSpace(params[0]))
			precision = size
		}
		if len(params) >= 2 {
			scale, _ = strconv.Atoi(strings.TrimSpace(params[1]))
		}
	}

	// 解析修饰符
	upper := strings.ToUpper(restDef)
	isPrimaryKey := strings.Contains(upper, "PRIMARY KEY")
	isAutoIncrement := strings.Contains(upper, "AUTO_INCREMENT") ||
		strings.Contains(upper, "AUTOINCREMENT") ||
		strings.Contains(upper, "SERIAL") ||
		strings.Contains(upper, "IDENTITY")
	isNotNull := strings.Contains(upper, "NOT NULL")

	// 解析默认值
	var defaultValue string
	if match := defaultRegex.FindStringSubmatch(def); len(match) > 1 {
		defaultValue = strings.Trim(match[1], "'\"")
	}

	// 解析注释
	var comment string
	if match := commentRegex.FindStringSubmatch(def); len(match) > 1 {
		comment = match[1]
	}

	// 获取 Go 类型
	dialect := getDialect(p.dialect)
	goType := dialect.ReverseTypeMapping(sqlType)

	// 特殊处理 UNSIGNED
	if strings.Contains(upper, "UNSIGNED") && !strings.HasPrefix(goType, "u") {
		switch goType {
		case "int8":
			goType = "uint8"
		case "int16":
			goType = "uint16"
		case "int32":
			goType = "uint32"
		case "int64":
			goType = "uint64"
		}
	}

	// 处理布尔类型
	if (baseType == "TINYINT" || baseType == "BIT") && size == 1 {
		goType = "bool"
	}

	col := Column{
		Name:          columnName,
		Type:          sqlType,
		GoType:        goType,
		PrimaryKey:    isPrimaryKey,
		AutoIncrement: isAutoIncrement,
		NotNull:       isNotNull,
		Default:       defaultValue,
		Comment:       comment,
		Size:          size,
		Precision:     precision,
		Scale:         scale,
	}

	field := &Field{
		Name:    toPascalCase(columnName),
		Type:    goType,
		Column:  col,
		Comment: comment,
	}

	return field, nil
}

// analyzeImports 分析需要导入的包
func (p *Parser) analyzeImports(schema *Schema) {
	imports := make(map[string]bool)

	for _, field := range schema.Fields {
		switch {
		case strings.Contains(field.Type, "time.Time"):
			imports["time"] = true
		case strings.Contains(field.Type, "json.RawMessage"):
			imports["encoding/json"] = true
		}
	}

	for pkg := range imports {
		schema.Imports = append(schema.Imports, pkg)
	}
}

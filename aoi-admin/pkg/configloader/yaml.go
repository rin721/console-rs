package configloader

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLScalarKind 限定可持久化的 YAML 标量类型，避免把任意结构写回配置文件。
type YAMLScalarKind string

const (
	YAMLScalarBool        YAMLScalarKind = "bool"
	YAMLScalarInt         YAMLScalarKind = "int"
	YAMLScalarString      YAMLScalarKind = "string"
	YAMLScalarStringSlice YAMLScalarKind = "string_slice"
)

// YAMLScalarUpdate 描述一次 YAML 节点更新；Path 使用点分路径，CreateMissing 只为显式允许的键补齐父级映射。
type YAMLScalarUpdate struct {
	Kind          YAMLScalarKind
	Path          string
	Value         string
	Values        []string
	CreateMissing bool
}

// YAMLUpdateOption 调整 YAML 标量持久化行为。
type YAMLUpdateOption func(*yamlUpdateOptions)

type yamlUpdateOptions struct {
	allowEnvPlaceholderOverwrite bool
}

var envPlaceholderPattern = regexp.MustCompile(`\$\{[^}]+\}`)

// WithEnvPlaceholderOverwrite 允许显式覆盖包含 ${...} 环境变量占位符的 YAML 节点。
func WithEnvPlaceholderOverwrite() YAMLUpdateOption {
	return func(options *yamlUpdateOptions) {
		options.allowEnvPlaceholderOverwrite = true
	}
}

// UpdateYAMLScalars 在保持文件身份和权限的前提下批量更新 YAML 标量节点。
// 它只处理 YAML 文件，并默认拒绝覆盖含 ${...} 的节点，避免把由环境变量托管的配置意外固化到文件中。
func UpdateYAMLScalars(path string, updates []YAMLScalarUpdate, options ...YAMLUpdateOption) error {
	if !isYAMLFile(path) {
		return fmt.Errorf("persistent config update only supports YAML files")
	}
	updateOptions := collectYAMLUpdateOptions(options)
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat config file: %w", err)
	}

	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}
	root, err := yamlDocumentRoot(&document)
	if err != nil {
		return err
	}

	for _, update := range updates {
		node, err := yamlValueNodeForPath(root, update.Path)
		if err != nil {
			if !update.CreateMissing {
				return fmt.Errorf("%s: %w", update.Path, err)
			}
			node, err = yamlEnsureValueNodeForPath(root, update.Path, update.Kind)
			if err != nil {
				return fmt.Errorf("%s: %w", update.Path, err)
			}
		}
		if yamlNodeContainsEnvPlaceholder(node) && !updateOptions.allowEnvPlaceholderOverwrite {
			return fmt.Errorf("%s is managed by environment placeholder", update.Path)
		}
		if err := setYAMLScalarValue(node, update); err != nil {
			return err
		}
	}

	nextContent, err := yaml.Marshal(&document)
	if err != nil {
		return fmt.Errorf("marshal config file: %w", err)
	}
	if bytes.Equal(content, nextContent) {
		return nil
	}
	if err := writeFilePreservingIdentity(path, content, nextContent, stat.Mode().Perm()); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

// yamlEnsureValueNodeForPath 沿点分路径创建缺失的 mapping 链，叶子节点类型由目标更新类型决定。
func yamlEnsureValueNodeForPath(root *yaml.Node, path string, kind YAMLScalarKind) (*yaml.Node, error) {
	current := root
	segments := strings.Split(path, ".")
	for index, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return nil, fmt.Errorf("invalid config key")
		}
		if current.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("config key parent is not a mapping")
		}
		if next, ok := yamlMappingValue(current, segment); ok {
			current = next
			continue
		}

		nextKind := yaml.MappingNode
		nextTag := "!!map"
		if index == len(segments)-1 {
			nextKind, nextTag = yamlNodeKindForScalarUpdate(kind)
		}
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: segment}
		valueNode := &yaml.Node{Kind: nextKind, Tag: nextTag}
		current.Content = append(current.Content, keyNode, valueNode)
		current = valueNode
	}
	return current, nil
}

// yamlNodeKindForScalarUpdate 将业务层标量类型映射到 yaml.v3 的节点形态。
func yamlNodeKindForScalarUpdate(kind YAMLScalarKind) (yaml.Kind, string) {
	if kind == YAMLScalarStringSlice {
		return yaml.SequenceNode, "!!seq"
	}
	return yaml.ScalarNode, "!!str"
}

// collectYAMLUpdateOptions 合并可选行为，nil option 会被忽略以便调用方条件式传参。
func collectYAMLUpdateOptions(options []YAMLUpdateOption) yamlUpdateOptions {
	var collected yamlUpdateOptions
	for _, option := range options {
		if option != nil {
			option(&collected)
		}
	}
	return collected
}

// YAMLPathContainsEnvPlaceholder 判断 YAML 文件中指定路径是否包含 ${...} 环境变量占位符。
func YAMLPathContainsEnvPlaceholder(path string, valuePath string) (bool, error) {
	if !isYAMLFile(path) {
		return false, fmt.Errorf("persistent config update only supports YAML files")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read config file: %w", err)
	}

	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return false, fmt.Errorf("parse config file: %w", err)
	}
	root, err := yamlDocumentRoot(&document)
	if err != nil {
		return false, err
	}
	node, err := yamlValueNodeForPath(root, valuePath)
	if err != nil {
		return false, fmt.Errorf("%s: %w", valuePath, err)
	}
	return yamlNodeContainsEnvPlaceholder(node), nil
}

// YAMLStringSlice 读取指定路径上的字符串序列；缺失路径返回 nil，便于调用方按“未配置”处理。
func YAMLStringSlice(path string, valuePath string) ([]string, error) {
	if !isYAMLFile(path) {
		return nil, fmt.Errorf("persistent config update only supports YAML files")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}
	root, err := yamlDocumentRoot(&document)
	if err != nil {
		return nil, err
	}
	node, err := yamlValueNodeForPath(root, valuePath)
	if err != nil {
		if strings.Contains(err.Error(), "config key does not exist in file") {
			return nil, nil
		}
		return nil, fmt.Errorf("%s: %w", valuePath, err)
	}
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("%s is not a string sequence", valuePath)
	}

	values := make([]string, 0, len(node.Content))
	for _, child := range node.Content {
		if child == nil || child.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("%s contains a non-scalar sequence item", valuePath)
		}
		values = append(values, child.Value)
	}
	return normalizeYAMLStringValues(values), nil
}

func isYAMLFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return true
	default:
		return false
	}
}

// yamlDocumentRoot 约束配置文件必须是 mapping 根节点，避免后续点分路径解析落到非对象结构上。
func yamlDocumentRoot(document *yaml.Node) (*yaml.Node, error) {
	if document == nil || document.Kind != yaml.DocumentNode || len(document.Content) == 0 {
		return nil, fmt.Errorf("config file must contain a YAML document")
	}
	if document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("config file root must be a YAML mapping")
	}
	return document.Content[0], nil
}

// yamlValueNodeForPath 按点分路径查找节点；中间节点允许 mapping 或 sequence 以支持数组下标访问。
func yamlValueNodeForPath(root *yaml.Node, path string) (*yaml.Node, error) {
	current := root
	segments := strings.Split(path, ".")
	for index, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return nil, fmt.Errorf("invalid config key")
		}
		next, err := yamlChildValue(current, segment)
		if err != nil {
			return nil, err
		}
		if index == len(segments)-1 {
			return next, nil
		}
		if next.Kind != yaml.MappingNode && next.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("config key parent is not a mapping or sequence")
		}
		current = next
	}
	return nil, fmt.Errorf("invalid config key")
}

// yamlChildValue 根据父节点类型解析子节点：mapping 使用键名，sequence 使用整数下标。
func yamlChildValue(parent *yaml.Node, segment string) (*yaml.Node, error) {
	switch parent.Kind {
	case yaml.MappingNode:
		next, ok := yamlMappingValue(parent, segment)
		if !ok {
			return nil, fmt.Errorf("config key does not exist in file")
		}
		return next, nil
	case yaml.SequenceNode:
		index, err := strconv.Atoi(segment)
		if err != nil {
			return nil, fmt.Errorf("config sequence index must be an integer")
		}
		if index < 0 || index >= len(parent.Content) {
			return nil, fmt.Errorf("config sequence index is out of range")
		}
		return parent.Content[index], nil
	default:
		return nil, fmt.Errorf("config key parent is not a mapping or sequence")
	}
}

// yamlMappingValue 按 yaml.v3 的 key/value 交错存储方式查找 mapping 值节点。
func yamlMappingValue(mapping *yaml.Node, key string) (*yaml.Node, bool) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil, false
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		keyNode := mapping.Content[index]
		if keyNode.Kind == yaml.ScalarNode && keyNode.Value == key {
			return mapping.Content[index+1], true
		}
	}
	return nil, false
}

// yamlNodeContainsEnvPlaceholder 递归检查节点树，保护由环境变量占位符托管的配置片段。
func yamlNodeContainsEnvPlaceholder(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	if node.Kind == yaml.ScalarNode {
		return envPlaceholderPattern.MatchString(node.Value)
	}
	for _, child := range node.Content {
		if yamlNodeContainsEnvPlaceholder(child) {
			return true
		}
	}
	return false
}

// setYAMLScalarValue 按声明的业务类型写入 YAML 节点，同时设置 tag/style 让输出保持可读且类型明确。
func setYAMLScalarValue(node *yaml.Node, update YAMLScalarUpdate) error {
	switch update.Kind {
	case YAMLScalarString:
		if node.Kind != yaml.ScalarNode {
			return fmt.Errorf("%s is not a scalar value", update.Path)
		}
		node.Tag = "!!str"
		node.Value = update.Value
		node.Style = yaml.DoubleQuotedStyle
	case YAMLScalarBool:
		if node.Kind != yaml.ScalarNode {
			return fmt.Errorf("%s is not a scalar value", update.Path)
		}
		node.Tag = "!!bool"
		node.Value = update.Value
		node.Style = 0
	case YAMLScalarInt:
		if node.Kind != yaml.ScalarNode {
			return fmt.Errorf("%s is not a scalar value", update.Path)
		}
		node.Tag = "!!int"
		node.Value = update.Value
		node.Style = 0
	case YAMLScalarStringSlice:
		if node.Kind != yaml.SequenceNode {
			return fmt.Errorf("%s is not a string sequence", update.Path)
		}
		node.Tag = "!!seq"
		node.Style = 0
		values := normalizeYAMLStringValues(update.Values)
		node.Content = make([]*yaml.Node, 0, len(values))
		for _, value := range values {
			node.Content = append(node.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: value,
				Style: yaml.DoubleQuotedStyle,
			})
		}
	default:
		return fmt.Errorf("%s has unsupported scalar kind %q", update.Path, update.Kind)
	}
	return nil
}

// normalizeYAMLStringValues 清理空白项并去重，避免多次保存后配置列表无限膨胀。
func normalizeYAMLStringValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

// writeFilePreservingIdentity 通过原地截断写入保留目标文件身份，便于配置监听器继续观察同一个 inode/句柄。
func writeFilePreservingIdentity(path string, oldContent, newContent []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tempFile, err := os.CreateTemp(dir, "."+base+".*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() {
		if tempPath != "" {
			_ = os.Remove(tempPath)
		}
	}()

	if err := tempFile.Chmod(mode); err != nil {
		_ = tempFile.Close()
		return err
	}
	if _, err := tempFile.Write(newContent); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := targetFile.Write(newContent); err != nil {
		_ = targetFile.Close()
		_ = os.WriteFile(path, oldContent, mode)
		return err
	}
	if err := targetFile.Sync(); err != nil {
		_ = targetFile.Close()
		_ = os.WriteFile(path, oldContent, mode)
		return err
	}
	if err := targetFile.Close(); err != nil {
		_ = os.WriteFile(path, oldContent, mode)
		return err
	}
	tempPath = ""
	_ = os.Remove(tempFile.Name())
	return nil
}

// Package injection 管理插件可请求的上下文注入能力和对应 JSON payload 构造器。
package injection

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

// DefaultSchemaVersion 是注入能力默认对外声明的 schema 版本。
const DefaultSchemaVersion = "v1"

// Capability 描述一个可注入上下文能力。
//
// Schema 是暴露给插件的 JSON Schema；Permissions 表示请求该能力前需要通过的授权 scope。
type Capability struct {
	Name        string
	Version     string
	Kind        string
	Permissions []string
	Schema      json.RawMessage
	Description string
}

// Provider 负责声明注入能力，并按请求构造实际上下文 payload。
type Provider interface {
	Capability() Capability
	Build(context.Context, Request) (json.RawMessage, error)
}

// ProviderFunc 让内联函数可以注册为注入 provider。
type ProviderFunc struct {
	Definition Capability
	Builder    func(context.Context, Request) (json.RawMessage, error)
}

// Capability 返回 provider 声明的能力定义。
func (p ProviderFunc) Capability() Capability {
	return p.Definition
}

// Build 调用底层构造函数生成注入 payload。
//
// Builder 为空时返回 nil，Registry 会把它规范化为 JSON null。
func (p ProviderFunc) Build(ctx context.Context, req Request) (json.RawMessage, error) {
	if p.Builder == nil {
		return nil, nil
	}
	return p.Builder(ctx, req)
}

// Request 描述插件请求上下文注入时的入参。
//
// Capabilities 为空表示请求全部已注册能力；SchemaVersion 由调用方透传，便于 provider 做版本兼容。
type Request struct {
	PluginID      string
	Capabilities  []string
	SchemaVersion string
}

// Registry 保存注入 provider，并负责输出 schema 与上下文 payload。
//
// Registry 不持久化插件私有状态；每次 BuildContext 都即时调用 provider，确保上下文反映当前运行态。
type Registry struct {
	mu            sync.RWMutex
	schemaVersion string
	source        string
	now           func() time.Time
	providers     map[string]Provider
}

// Config 描述注入 registry 的版本、审计来源和时钟。
type Config struct {
	SchemaVersion string
	Source        string
	Now           func() time.Time
}

// NewRegistry 创建注入能力注册表，并补齐默认 schema 版本和 UTC 时钟。
func NewRegistry(cfg Config) *Registry {
	if strings.TrimSpace(cfg.SchemaVersion) == "" {
		cfg.SchemaVersion = DefaultSchemaVersion
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Registry{
		schemaVersion: strings.TrimSpace(cfg.SchemaVersion),
		source:        strings.TrimSpace(cfg.Source),
		now:           func() time.Time { return cfg.Now().UTC() },
		providers:     map[string]Provider{},
	}
}

// Register 注册或替换一个注入 provider。
//
// 无效 provider 或空名称会被忽略，避免可选能力未启用时影响 Host 启动流程。
func (r *Registry) Register(provider Provider) error {
	if r == nil || provider == nil {
		return nil
	}
	capability := normalizeCapability(provider.Capability())
	if capability.Name == "" {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[capability.Name] = ProviderFunc{
		Definition: capability,
		Builder:    provider.Build,
	}
	return nil
}

// Capabilities 返回已注册注入能力的协议表示。
//
// names 为空时返回全部能力；结果按名称排序，保证 schema 响应和测试输出稳定。
func (r *Registry) Capabilities(names []string) []protocol.InjectedCapability {
	if r == nil {
		return nil
	}
	want := wanted(names)
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]protocol.InjectedCapability, 0, len(r.providers))
	for name, provider := range r.providers {
		if len(want) > 0 && !want[name] {
			continue
		}
		out = append(out, toProtocolCapability(provider.Capability()))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

// Schema 返回插件可见的注入能力 schema 响应。
func (r *Registry) Schema(names []string) protocol.GetInjectedSchemaResponse {
	if r == nil {
		return protocol.GetInjectedSchemaResponse{}
	}
	return protocol.GetInjectedSchemaResponse{
		SchemaVersion: r.schemaVersion,
		Capabilities:  r.Capabilities(names),
		Audit:         r.audit(),
	}
}

// BuildContext 根据请求能力构造注入上下文。
//
// 先复制 provider 表再释放读锁，避免 provider 构造 payload 时阻塞其他注册或查询操作。
func (r *Registry) BuildContext(ctx context.Context, req Request) (protocol.InjectContextResponse, error) {
	if r == nil {
		return protocol.InjectContextResponse{}, nil
	}
	want := wanted(req.Capabilities)
	r.mu.RLock()
	providers := make(map[string]Provider, len(r.providers))
	for name, provider := range r.providers {
		if len(want) > 0 && !want[name] {
			continue
		}
		providers[name] = provider
	}
	r.mu.RUnlock()

	payload := make(map[string]json.RawMessage, len(providers))
	for name, provider := range providers {
		value, err := provider.Build(ctx, req)
		if err != nil {
			return protocol.InjectContextResponse{}, err
		}
		if len(value) == 0 {
			value = json.RawMessage(`null`)
		}
		payload[name] = append(json.RawMessage(nil), value...)
	}
	return protocol.InjectContextResponse{
		SchemaVersion: r.schemaVersion,
		Context:       payload,
		Audit:         r.audit(),
	}, nil
}

// audit 生成注入响应使用的审计元数据。
func (r *Registry) audit() protocol.AuditInfo {
	return protocol.AuditInfo{
		GeneratedAt: r.now(),
		Source:      r.source,
	}
}

// normalizeCapability 清理能力定义并复制可变字段，避免调用方后续修改影响 registry 内部状态。
func normalizeCapability(capability Capability) Capability {
	capability.Name = strings.TrimSpace(capability.Name)
	capability.Version = strings.TrimSpace(capability.Version)
	capability.Kind = strings.TrimSpace(capability.Kind)
	capability.Description = strings.TrimSpace(capability.Description)
	capability.Permissions = normalizedStrings(capability.Permissions)
	capability.Schema = append(json.RawMessage(nil), capability.Schema...)
	return capability
}

// normalizedStrings 去除空白字符串并统一 trim，用于权限列表和能力名称过滤。
func normalizedStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

// wanted 将请求能力列表转换为集合；空集合表示不做过滤。
func wanted(names []string) map[string]bool {
	out := map[string]bool{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			out[name] = true
		}
	}
	return out
}

// toProtocolCapability 将内部能力定义转换为跨 transport 的协议模型。
func toProtocolCapability(capability Capability) protocol.InjectedCapability {
	capability = normalizeCapability(capability)
	return protocol.InjectedCapability{
		Name:        capability.Name,
		Version:     capability.Version,
		Kind:        capability.Kind,
		Permissions: append([]string(nil), capability.Permissions...),
		Schema:      append(json.RawMessage(nil), capability.Schema...),
		Description: capability.Description,
	}
}

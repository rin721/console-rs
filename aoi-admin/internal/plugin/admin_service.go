package plugin

import (
	"context"

	pluginpkg "github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

var (
	ErrDisabled       = pluginpkg.ErrDisabled
	ErrPluginNotFound = pluginpkg.ErrPluginNotFound
	ErrPluginOffline  = pluginpkg.ErrPluginOffline
)

// Host 定义后台管理服务需要的插件宿主最小能力集合。
//
// 通过窄接口隔离 pkg/plugin 的完整实现，便于在服务测试中替换宿主。
type Host interface {
	ListPlugins(context.Context) ([]protocol.PluginSnapshot, error)
	GetPlugin(context.Context, string) (protocol.PluginSnapshot, error)
	Health(context.Context, string) (protocol.HealthStatus, error)
	ListCapabilities(context.Context, protocol.ListCapabilitiesRequest) (protocol.ListCapabilitiesResponse, error)
}

// Service 定义后台管理侧可调用的插件查询能力。
type Service interface {
	List(context.Context) ([]protocol.PluginSnapshot, error)
	Get(context.Context, string) (protocol.PluginSnapshot, error)
	Health(context.Context, string) (protocol.HealthStatus, error)
	Capabilities(context.Context, string) (protocol.ListCapabilitiesResponse, error)
}

// service 是插件后台管理服务的默认实现。
type service struct {
	host Host
}

// NewService 创建插件后台管理服务。
//
// host 允许为 nil；这种情况下所有查询返回 ErrDisabled，便于上层在插件关闭时复用同一 handler 结构。
func NewService(host Host) Service {
	return &service{host: host}
}

func (s *service) List(ctx context.Context) ([]protocol.PluginSnapshot, error) {
	if s.host == nil {
		return nil, ErrDisabled
	}
	return s.host.ListPlugins(ctx)
}

func (s *service) Get(ctx context.Context, id string) (protocol.PluginSnapshot, error) {
	if s.host == nil {
		return protocol.PluginSnapshot{}, ErrDisabled
	}
	return s.host.GetPlugin(ctx, id)
}

func (s *service) Health(ctx context.Context, id string) (protocol.HealthStatus, error) {
	if s.host == nil {
		return protocol.HealthStatus{}, ErrDisabled
	}
	return s.host.Health(ctx, id)
}

// Capabilities 查询在线插件声明的能力。
//
// 能力查询会先确认插件存在且在线，避免对离线插件发起可能阻塞或失真的远程调用。
func (s *service) Capabilities(ctx context.Context, id string) (protocol.ListCapabilitiesResponse, error) {
	if s.host == nil {
		return protocol.ListCapabilitiesResponse{}, ErrDisabled
	}
	plugin, err := s.host.GetPlugin(ctx, id)
	if err != nil {
		return protocol.ListCapabilitiesResponse{}, err
	}
	if plugin.Status != protocol.StatusOnline {
		return protocol.ListCapabilitiesResponse{}, ErrPluginOffline
	}
	return s.host.ListCapabilities(ctx, protocol.ListCapabilitiesRequest{PluginID: id})
}

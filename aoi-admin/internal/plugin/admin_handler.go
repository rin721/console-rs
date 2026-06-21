package plugin

import (
	"errors"
	"net/http"

	"github.com/rei0721/go-scaffold/internal/ports"
	"github.com/rei0721/go-scaffold/types/result"
)

// Handler 提供后台管理侧的插件查询接口。
//
// 它只负责 HTTP 参数提取和错误映射，插件状态、健康检查和能力查询逻辑由 Service 承担。
type Handler struct {
	service Service
	logger  ports.Logger
}

// NewHandler 创建插件后台管理 HTTP handler。
func NewHandler(service Service, logger ports.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) List(c ports.HTTPContext) {
	plugins, err := h.service.List(c.RequestContext())
	if err != nil {
		h.writeError(c, err)
		return
	}
	result.OK(c, plugins)
}

func (h *Handler) Get(c ports.HTTPContext) {
	plugin, err := h.service.Get(c.RequestContext(), c.Param("pluginId"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	result.OK(c, plugin)
}

func (h *Handler) Health(c ports.HTTPContext) {
	status, err := h.service.Health(c.RequestContext(), c.Param("pluginId"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	result.OK(c, status)
}

func (h *Handler) Capabilities(c ports.HTTPContext) {
	capabilities, err := h.service.Capabilities(c.RequestContext(), c.Param("pluginId"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	result.OK(c, capabilities)
}

// writeError 将插件领域错误转换为后台管理 API 的 HTTP 响应。
//
// 未知错误会记录日志并返回 Bad Gateway，表示请求已到达管理端但下游插件或宿主处理失败。
func (h *Handler) writeError(c ports.HTTPContext, err error) {
	switch {
	case errors.Is(err, ErrDisabled):
		result.NotFound(c, "plugins disabled")
	case errors.Is(err, ErrPluginNotFound):
		result.NotFound(c, "plugin not found")
	case errors.Is(err, ErrPluginOffline):
		result.Fail(c, http.StatusConflict, "plugin offline")
	default:
		if h.logger != nil {
			h.logger.Error("plugin request failed", "error", err)
		}
		result.Fail(c, http.StatusBadGateway, "plugin request failed")
	}
}

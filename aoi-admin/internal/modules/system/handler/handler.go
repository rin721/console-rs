// Package handler 将系统管理 HTTP 请求转换为服务层输入，并统一处理响应和错误映射。
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/internal/middleware"
	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
	"github.com/rei0721/go-scaffold/internal/modules/system/model"
	"github.com/rei0721/go-scaffold/internal/modules/system/service"
	"github.com/rei0721/go-scaffold/internal/ports"
	"github.com/rei0721/go-scaffold/types/result"
)

// Handler 是系统模块的 HTTP 适配器。
// authorizer 用于在返回菜单前过滤当前用户不可访问的菜单项。
type Handler struct {
	service    service.Service
	authorizer middleware.Authorizer
	logger     ports.Logger
}

type CreateDictionaryRequest struct {
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
	Name        string `json:"name" binding:"required"`
	Status      string `json:"status"`
}

type UpdateDictionaryRequest struct {
	Description *string `json:"description"`
	Name        *string `json:"name"`
	Status      *string `json:"status"`
}

type CreateDictionaryItemRequest struct {
	Extra  string `json:"extra"`
	Label  string `json:"label" binding:"required"`
	Sort   int    `json:"sort"`
	Status string `json:"status"`
	Value  string `json:"value" binding:"required"`
}

type UpdateDictionaryItemRequest struct {
	Extra  *string `json:"extra"`
	Label  *string `json:"label"`
	Sort   *int    `json:"sort"`
	Status *string `json:"status"`
	Value  *string `json:"value"`
}

type CreateParameterRequest struct {
	Description string `json:"description"`
	Key         string `json:"key" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Value       string `json:"value" binding:"required"`
}

type UpdateParameterRequest struct {
	Description *string `json:"description"`
	Key         *string `json:"key"`
	Name        *string `json:"name"`
	Value       *string `json:"value"`
}

type CreateTrafficProbeTargetRequest struct {
	AlertChannels          []string `json:"alertChannels"`
	AllowPrivateNetwork    bool     `json:"allowPrivateNetwork"`
	EmailRecipients        []string `json:"emailRecipients"`
	Enabled                *bool    `json:"enabled"`
	ExpectedContentKeyword string   `json:"expectedContentKeyword"`
	ExpectedFinalHost      string   `json:"expectedFinalHost"`
	ExpectedIPCIDRs        []string `json:"expectedIpCidrs"`
	ExpectedStatusCodes    string   `json:"expectedStatusCodes"`
	ExpectedTLSFingerprint string   `json:"expectedTlsFingerprint"`
	IntervalSeconds        int      `json:"intervalSeconds"`
	Method                 string   `json:"method"`
	Name                   string   `json:"name" binding:"required"`
	TimeoutSeconds         int      `json:"timeoutSeconds"`
	URL                    string   `json:"url" binding:"required"`
}

type UpdateTrafficProbeTargetRequest struct {
	AlertChannels          *[]string `json:"alertChannels"`
	AllowPrivateNetwork    *bool     `json:"allowPrivateNetwork"`
	EmailRecipients        *[]string `json:"emailRecipients"`
	Enabled                *bool     `json:"enabled"`
	ExpectedContentKeyword *string   `json:"expectedContentKeyword"`
	ExpectedFinalHost      *string   `json:"expectedFinalHost"`
	ExpectedIPCIDRs        *[]string `json:"expectedIpCidrs"`
	ExpectedStatusCodes    *string   `json:"expectedStatusCodes"`
	ExpectedTLSFingerprint *string   `json:"expectedTlsFingerprint"`
	IntervalSeconds        *int      `json:"intervalSeconds"`
	Method                 *string   `json:"method"`
	Name                   *string   `json:"name"`
	TimeoutSeconds         *int      `json:"timeoutSeconds"`
	URL                    *string   `json:"url"`
}

// DeleteOperationRecordsRequest 承载批量删除操作记录的 ID 列表。
type DeleteOperationRecordsRequest struct {
	IDs []systemID `json:"ids"`
}

// DeleteParametersRequest 承载批量删除参数的 ID 列表。
type DeleteParametersRequest struct {
	IDs []systemID `json:"ids"`
}

// DeleteVersionsRequest 承载批量删除版本记录的 ID 列表。
type DeleteVersionsRequest struct {
	IDs []systemID `json:"ids"`
}

// UpdateConfigRequest 描述运行时配置批量更新请求。
type UpdateConfigRequest struct {
	Items   []UpdateConfigItemRequest `json:"items" binding:"required"`
	Persist bool                      `json:"persist"`
}

type UpdateConfigItemRequest struct {
	Key   string `json:"key" binding:"required"`
	Value any    `json:"value"`
}

func (h *Handler) PublicSettings(c ports.HTTPContext) {
	settings, err := h.service.GetPublicSettings(c.RequestContext())
	writeOK(c, settings, err, h.writeError)
}

// ExportVersionRequest 描述版本导出时选择的菜单、API 和字典范围。
type ExportVersionRequest struct {
	APICodes        []string `json:"apiCodes"`
	Description     string   `json:"description"`
	DictionaryCodes []string `json:"dictionaryCodes"`
	MenuCodes       []string `json:"menuCodes"`
	VersionCode     string   `json:"versionCode" binding:"required"`
	VersionName     string   `json:"versionName" binding:"required"`
}

// ImportVersionRequest 承载导入版本包的 JSON 文本。
type ImportVersionRequest struct {
	VersionData string `json:"versionData" binding:"required"`
}

type UpsertMediaCategoryRequest struct {
	ID       systemID `json:"id"`
	ParentID systemID `json:"parentId"`
	Name     string   `json:"name" binding:"required"`
	Sort     int      `json:"sort"`
}

// ImportMediaURLsRequest 支持结构化列表或多行文本两种 URL 导入格式。
type ImportMediaURLsRequest struct {
	CategoryID systemID             `json:"categoryId"`
	Items      []ImportMediaURLItem `json:"items"`
	Text       string               `json:"text"`
}

type ImportMediaURLItem struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type UpdateMediaAssetRequest struct {
	DisplayName string `json:"displayName" binding:"required"`
}

// CheckMediaResumableUploadRequest 描述断点续传建会话前的文件摘要信息。
type CheckMediaResumableUploadRequest struct {
	CategoryID systemID `json:"categoryId"`
	ChunkSize  int64    `json:"chunkSize"`
	ChunkTotal int      `json:"chunkTotal"`
	FileHash   string   `json:"fileHash" binding:"required"`
	FileName   string   `json:"fileName" binding:"required"`
	SizeBytes  int64    `json:"sizeBytes" binding:"required"`
}

// MediaResumableSessionRequest 是完成或终止断点续传会话的公共请求体。
type MediaResumableSessionRequest struct {
	FileHash  string   `json:"fileHash" binding:"required"`
	SessionID systemID `json:"sessionId" binding:"required"`
}

// systemID 兼容前端以字符串或数字传递 int64 ID。
type systemID int64

// New 创建系统模块 HTTP handler。
func New(service service.Service, authorizer middleware.Authorizer, logger ports.Logger) *Handler {
	return &Handler{service: service, authorizer: authorizer, logger: logger}
}

func (h *Handler) ListMenus(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	groups, err := h.service.ListMenus(c.RequestContext())
	if err != nil {
		h.writeError(c, err)
		return
	}
	result.OK(c, h.localizeMenus(c, h.filterMenus(c.RequestContext(), principal, groups)))
}

func (h *Handler) ListAPIs(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	groups, err := h.service.ListAPIs(c.RequestContext())
	if err != nil {
		h.writeError(c, err)
		return
	}
	result.OK(c, groups)
}

func (h *Handler) ListConfig(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	snapshot, err := h.service.ListConfig(c.RequestContext())
	if err != nil {
		h.writeError(c, err)
		return
	}
	result.OK(c, h.localizeConfigSnapshot(c, snapshot))
}

func (h *Handler) UpdateConfig(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	var req UpdateConfigRequest
	if !bind(c, &req) {
		return
	}
	input := service.UpdateConfigInput{Items: make([]service.UpdateConfigItem, 0, len(req.Items)), Persist: req.Persist}
	for _, item := range req.Items {
		input.Items = append(input.Items, service.UpdateConfigItem{
			Key:   item.Key,
			Value: item.Value,
		})
	}
	snapshot, err := h.service.UpdateConfig(c.RequestContext(), input)
	if err != nil {
		h.writeError(c, err)
		return
	}
	result.OK(c, h.localizeConfigSnapshot(c, snapshot))
}

func (h *Handler) GetServerInfo(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	info, err := h.service.GetServerInfo(c.RequestContext())
	writeOK(c, info, err, h.writeError)
}

func (h *Handler) GetServerMetricsHistory(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	history, err := h.service.GetServerMetricsHistory(c.RequestContext())
	writeOK(c, history, err, h.writeError)
}

func (h *Handler) GetTrafficHijackOverview(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	overview, err := h.service.GetTrafficHijackOverview(c.RequestContext())
	writeOK(c, overview, err, h.writeError)
}

func (h *Handler) ListTrafficProbeTargets(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	targets, err := h.service.ListTrafficProbeTargets(c.RequestContext())
	writeOK(c, targets, err, h.writeError)
}

func (h *Handler) CreateTrafficProbeTarget(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	var req CreateTrafficProbeTargetRequest
	if !bind(c, &req) {
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	target, err := h.service.CreateTrafficProbeTarget(c.RequestContext(), service.CreateTrafficProbeTargetInput{
		AlertChannels:          req.AlertChannels,
		AllowPrivateNetwork:    req.AllowPrivateNetwork,
		EmailRecipients:        req.EmailRecipients,
		Enabled:                enabled,
		ExpectedContentKeyword: req.ExpectedContentKeyword,
		ExpectedFinalHost:      req.ExpectedFinalHost,
		ExpectedIPCIDRs:        req.ExpectedIPCIDRs,
		ExpectedStatusCodes:    req.ExpectedStatusCodes,
		ExpectedTLSFingerprint: req.ExpectedTLSFingerprint,
		IntervalSeconds:        req.IntervalSeconds,
		Method:                 req.Method,
		Name:                   req.Name,
		TimeoutSeconds:         req.TimeoutSeconds,
		URL:                    req.URL,
	})
	writeCreated(c, target, err, h.writeError)
}

func (h *Handler) UpdateTrafficProbeTarget(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "targetId")
	if !ok {
		return
	}
	var req UpdateTrafficProbeTargetRequest
	if !bind(c, &req) {
		return
	}
	target, err := h.service.UpdateTrafficProbeTarget(c.RequestContext(), id, service.UpdateTrafficProbeTargetInput{
		AlertChannels:          req.AlertChannels,
		AllowPrivateNetwork:    req.AllowPrivateNetwork,
		EmailRecipients:        req.EmailRecipients,
		Enabled:                req.Enabled,
		ExpectedContentKeyword: req.ExpectedContentKeyword,
		ExpectedFinalHost:      req.ExpectedFinalHost,
		ExpectedIPCIDRs:        req.ExpectedIPCIDRs,
		ExpectedStatusCodes:    req.ExpectedStatusCodes,
		ExpectedTLSFingerprint: req.ExpectedTLSFingerprint,
		IntervalSeconds:        req.IntervalSeconds,
		Method:                 req.Method,
		Name:                   req.Name,
		TimeoutSeconds:         req.TimeoutSeconds,
		URL:                    req.URL,
	})
	writeOK(c, target, err, h.writeError)
}

func (h *Handler) DeleteTrafficProbeTarget(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "targetId")
	if !ok {
		return
	}
	writeOK(c, map[string]bool{"deleted": true}, h.service.DeleteTrafficProbeTarget(c.RequestContext(), id), h.writeError)
}

func (h *Handler) RunTrafficProbe(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "targetId")
	if !ok {
		return
	}
	probeResult, err := h.service.RunTrafficProbe(c.RequestContext(), id)
	writeOK(c, probeResult, err, h.writeError)
}

func (h *Handler) ListTrafficProbeResults(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	targetID, ok := parseInt64Query(c, "targetId", 0)
	if !ok {
		return
	}
	limit, ok := parseIntQuery(c, "limit", 50)
	if !ok {
		return
	}
	cursor, ok := parseInt64Query(c, "cursor", 0)
	if !ok {
		return
	}
	page, err := h.service.ListTrafficProbeResults(c.RequestContext(), service.TrafficProbeResultFilter{TargetID: targetID, Limit: limit, Cursor: cursor})
	writeOK(c, page, err, h.writeError)
}

func (h *Handler) ListTrafficHijackEvents(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	values := c.Request().URL.Query()
	targetID, ok := parseInt64Query(c, "targetId", 0)
	if !ok {
		return
	}
	page, ok := parseIntQuery(c, "page", 1)
	if !ok {
		return
	}
	pageSize, ok := parseIntQuery(c, "pageSize", 10)
	if !ok {
		return
	}
	events, err := h.service.ListTrafficHijackEvents(c.RequestContext(), service.TrafficHijackEventFilter{
		TargetID: targetID,
		Severity: values.Get("severity"),
		State:    values.Get("state"),
		Page:     page,
		PageSize: pageSize,
	})
	writeOK(c, events, err, h.writeError)
}

func (h *Handler) ResolveTrafficHijackEvent(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "eventId")
	if !ok {
		return
	}
	event, err := h.service.ResolveTrafficHijackEvent(c.RequestContext(), id)
	writeOK(c, event, err, h.writeError)
}

func (h *Handler) StreamTrafficHijack(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	writerContext, ok := c.(interface{ ResponseWriter() http.ResponseWriter })
	if !ok {
		h.writeError(c, service.ErrConfigUnavailable)
		return
	}
	writer := writerContext.ResponseWriter()
	flusher, ok := writer.(http.Flusher)
	if !ok {
		h.writeError(c, service.ErrConfigUnavailable)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	events, cancel := h.service.SubscribeTrafficHijack(c.RequestContext())
	defer cancel()
	writeSSE(writer, flusher, "ready", map[string]string{"status": "ok"})
	for {
		select {
		case <-c.RequestContext().Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			writeSSE(writer, flusher, event.Type, event.Payload)
		}
	}
}

func (h *Handler) SyncAPIs(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	syncResult, err := h.service.SyncAPIs(c.RequestContext())
	if err != nil {
		h.writeError(c, err)
		return
	}
	result.OK(c, syncResult)
}

func (h *Handler) SyncPermissions(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	syncResult, err := h.service.SyncPermissions(c.RequestContext())
	if err != nil {
		h.writeError(c, err)
		return
	}
	result.OK(c, syncResult)
}

func (h *Handler) ListDictionaries(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	catalog, err := h.service.ListDictionaries(c.RequestContext())
	writeOK(c, catalog, err, h.writeError)
}

func (h *Handler) CreateDictionary(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	var req CreateDictionaryRequest
	if !bind(c, &req) {
		return
	}
	dictionary, err := h.service.CreateDictionary(c.RequestContext(), service.CreateDictionaryInput{
		Code:        req.Code,
		Description: req.Description,
		Name:        req.Name,
		Status:      req.Status,
	})
	writeCreated(c, dictionary, err, h.writeError)
}

func (h *Handler) UpdateDictionary(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "dictionaryId")
	if !ok {
		return
	}
	var req UpdateDictionaryRequest
	if !bind(c, &req) {
		return
	}
	dictionary, err := h.service.UpdateDictionary(c.RequestContext(), id, service.UpdateDictionaryInput{
		Description: req.Description,
		Name:        req.Name,
		Status:      req.Status,
	})
	writeOK(c, dictionary, err, h.writeError)
}

func (h *Handler) DeleteDictionary(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "dictionaryId")
	if !ok {
		return
	}
	writeOK(c, map[string]bool{"deleted": true}, h.service.DeleteDictionary(c.RequestContext(), id), h.writeError)
}

func (h *Handler) CreateDictionaryItem(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	dictionaryID, ok := parseInt64Param(c, "dictionaryId")
	if !ok {
		return
	}
	var req CreateDictionaryItemRequest
	if !bind(c, &req) {
		return
	}
	item, err := h.service.CreateDictionaryItem(c.RequestContext(), dictionaryID, service.CreateDictionaryItemInput{
		Extra:  req.Extra,
		Label:  req.Label,
		Sort:   req.Sort,
		Status: req.Status,
		Value:  req.Value,
	})
	writeCreated(c, item, err, h.writeError)
}

func (h *Handler) UpdateDictionaryItem(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "itemId")
	if !ok {
		return
	}
	var req UpdateDictionaryItemRequest
	if !bind(c, &req) {
		return
	}
	item, err := h.service.UpdateDictionaryItem(c.RequestContext(), id, service.UpdateDictionaryItemInput{
		Extra:  req.Extra,
		Label:  req.Label,
		Sort:   req.Sort,
		Status: req.Status,
		Value:  req.Value,
	})
	writeOK(c, item, err, h.writeError)
}

func (h *Handler) DeleteDictionaryItem(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "itemId")
	if !ok {
		return
	}
	writeOK(c, map[string]bool{"deleted": true}, h.service.DeleteDictionaryItem(c.RequestContext(), id), h.writeError)
}

func (h *Handler) ListOperationRecords(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	values := c.Request().URL.Query()
	page, ok := parseIntQuery(c, "page", 1)
	if !ok {
		return
	}
	pageSize, ok := parseIntQuery(c, "pageSize", 10)
	if !ok {
		return
	}
	status, ok := parseIntQuery(c, "status", 0)
	if !ok {
		return
	}
	records, err := h.service.ListOperationRecords(c.RequestContext(), service.OperationRecordFilter{
		Method:      values.Get("method"),
		Page:        page,
		PageSize:    pageSize,
		Path:        values.Get("path"),
		Status:      status,
		StatusClass: values.Get("statusClass"),
	})
	writeOK(c, records, err, h.writeError)
}

func (h *Handler) DeleteOperationRecords(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	var req DeleteOperationRecordsRequest
	if !bind(c, &req) {
		return
	}
	writeOK(c, map[string]bool{"deleted": true}, h.service.DeleteOperationRecords(c.RequestContext(), req.int64s()), h.writeError)
}

func (h *Handler) ListParameters(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	values := c.Request().URL.Query()
	page, ok := parseIntQuery(c, "page", 1)
	if !ok {
		return
	}
	pageSize, ok := parseIntQuery(c, "pageSize", 10)
	if !ok {
		return
	}
	startCreatedAt, ok := parseTimeQuery(c, "startCreatedAt", false)
	if !ok {
		return
	}
	endCreatedAt, ok := parseTimeQuery(c, "endCreatedAt", true)
	if !ok {
		return
	}
	parameters, err := h.service.ListParameters(c.RequestContext(), service.ParameterFilter{
		EndCreatedAt:   endCreatedAt,
		Key:            values.Get("key"),
		Name:           values.Get("name"),
		Page:           page,
		PageSize:       pageSize,
		StartCreatedAt: startCreatedAt,
	})
	writeOK(c, parameters, err, h.writeError)
}

func (h *Handler) CreateParameter(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	var req CreateParameterRequest
	if !bind(c, &req) {
		return
	}
	parameter, err := h.service.CreateParameter(c.RequestContext(), service.CreateParameterInput{
		Description: req.Description,
		Key:         req.Key,
		Name:        req.Name,
		Value:       req.Value,
	})
	writeCreated(c, parameter, err, h.writeError)
}

func (h *Handler) GetParameter(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "parameterId")
	if !ok {
		return
	}
	parameter, err := h.service.FindParameter(c.RequestContext(), id)
	writeOK(c, parameter, err, h.writeError)
}

func (h *Handler) GetParameterByKey(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	parameter, err := h.service.FindParameterByKey(c.RequestContext(), c.Request().URL.Query().Get("key"))
	writeOK(c, parameter, err, h.writeError)
}

func (h *Handler) UpdateParameter(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "parameterId")
	if !ok {
		return
	}
	var req UpdateParameterRequest
	if !bind(c, &req) {
		return
	}
	parameter, err := h.service.UpdateParameter(c.RequestContext(), id, service.UpdateParameterInput{
		Description: req.Description,
		Key:         req.Key,
		Name:        req.Name,
		Value:       req.Value,
	})
	writeOK(c, parameter, err, h.writeError)
}

func (h *Handler) DeleteParameter(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "parameterId")
	if !ok {
		return
	}
	writeOK(c, map[string]bool{"deleted": true}, h.service.DeleteParameter(c.RequestContext(), id), h.writeError)
}

func (h *Handler) DeleteParameters(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	var req DeleteParametersRequest
	if !bind(c, &req) {
		return
	}
	writeOK(c, map[string]bool{"deleted": true}, h.service.DeleteParameters(c.RequestContext(), req.int64s()), h.writeError)
}

func (h *Handler) ListVersionSources(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	sources, err := h.service.ListVersionSources(c.RequestContext())
	writeOK(c, sources, err, h.writeError)
}

func (h *Handler) ListVersions(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	values := c.Request().URL.Query()
	page, ok := parseIntQuery(c, "page", 1)
	if !ok {
		return
	}
	pageSize, ok := parseIntQuery(c, "pageSize", 10)
	if !ok {
		return
	}
	startCreatedAt, ok := parseTimeQuery(c, "startCreatedAt", false)
	if !ok {
		return
	}
	endCreatedAt, ok := parseTimeQuery(c, "endCreatedAt", true)
	if !ok {
		return
	}
	versions, err := h.service.ListVersions(c.RequestContext(), service.VersionFilter{
		EndCreatedAt:   endCreatedAt,
		Page:           page,
		PageSize:       pageSize,
		StartCreatedAt: startCreatedAt,
		VersionCode:    values.Get("versionCode"),
		VersionName:    values.Get("versionName"),
	})
	writeOK(c, versions, err, h.writeError)
}

func (h *Handler) GetVersion(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "versionId")
	if !ok {
		return
	}
	version, err := h.service.FindVersion(c.RequestContext(), id)
	writeOK(c, version, err, h.writeError)
}

func (h *Handler) ExportVersion(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	var req ExportVersionRequest
	if !bind(c, &req) {
		return
	}
	version, err := h.service.ExportVersion(c.RequestContext(), service.ExportVersionInput{
		APICodes:        req.APICodes,
		CreatedBy:       principal.UserID,
		CreatorUsername: principal.Username,
		Description:     req.Description,
		DictionaryCodes: req.DictionaryCodes,
		MenuCodes:       req.MenuCodes,
		VersionCode:     req.VersionCode,
		VersionName:     req.VersionName,
	})
	writeCreated(c, version, err, h.writeError)
}

func (h *Handler) ImportVersion(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	var req ImportVersionRequest
	if !bind(c, &req) {
		return
	}
	importResult, err := h.service.ImportVersion(c.RequestContext(), service.ImportVersionInput{
		CreatedBy:       principal.UserID,
		CreatorUsername: principal.Username,
		VersionData:     req.VersionData,
	})
	writeCreated(c, importResult, err, h.writeError)
}

func (h *Handler) DownloadVersion(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "versionId")
	if !ok {
		return
	}
	pkg, err := h.service.GetVersionPackage(c.RequestContext(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="system-version-`+strconv.FormatInt(id, 10)+`.json"`)
	result.OK(c, pkg)
}

func (h *Handler) DeleteVersion(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "versionId")
	if !ok {
		return
	}
	writeOK(c, map[string]bool{"deleted": true}, h.service.DeleteVersion(c.RequestContext(), id), h.writeError)
}

func (h *Handler) DeleteVersions(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	var req DeleteVersionsRequest
	if !bind(c, &req) {
		return
	}
	writeOK(c, map[string]bool{"deleted": true}, h.service.DeleteVersions(c.RequestContext(), req.int64s()), h.writeError)
}

func (h *Handler) ListMediaCategories(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	catalog, err := h.service.ListMediaCategories(c.RequestContext())
	writeOK(c, catalog, err, h.writeError)
}

func (h *Handler) UpsertMediaCategory(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	var req UpsertMediaCategoryRequest
	if !bind(c, &req) {
		return
	}
	category, err := h.service.UpsertMediaCategory(c.RequestContext(), service.UpsertMediaCategoryInput{
		ID:       int64(req.ID),
		Name:     req.Name,
		ParentID: int64(req.ParentID),
		Sort:     req.Sort,
	})
	if int64(req.ID) > 0 {
		writeOK(c, category, err, h.writeError)
		return
	}
	writeCreated(c, category, err, h.writeError)
}

func (h *Handler) DeleteMediaCategory(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "categoryId")
	if !ok {
		return
	}
	writeOK(c, map[string]bool{"deleted": true}, h.service.DeleteMediaCategory(c.RequestContext(), id), h.writeError)
}

func (h *Handler) ListMediaAssets(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	values := c.Request().URL.Query()
	page, ok := parseIntQuery(c, "page", 1)
	if !ok {
		return
	}
	pageSize, ok := parseIntQuery(c, "pageSize", 10)
	if !ok {
		return
	}
	categoryID, ok := parseInt64Query(c, "categoryId", 0)
	if !ok {
		return
	}
	assets, err := h.service.ListMediaAssets(c.RequestContext(), service.MediaAssetFilter{
		CategoryID: categoryID,
		Keyword:    values.Get("keyword"),
		Page:       page,
		PageSize:   pageSize,
	})
	writeOK(c, assets, err, h.writeError)
}

func (h *Handler) UploadMediaAsset(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	req := c.Request()
	if err := req.ParseMultipartForm(32 << 20); err != nil {
		result.BadRequest(c, result.MessageKeyInvalidRequest)
		return
	}
	file, header, err := req.FormFile("file")
	if err != nil {
		result.BadRequest(c, "validation.common.required", map[string]any{"field": "file"})
		return
	}
	defer file.Close()
	categoryID, ok := parseInt64Form(c, "categoryId", 0)
	if !ok {
		return
	}
	asset, err := h.service.UploadMediaAsset(c.RequestContext(), service.UploadMediaAssetInput{
		CategoryID:         categoryID,
		Filename:           header.Filename,
		Reader:             file,
		Size:               header.Size,
		UploadedBy:         principal.UserID,
		UploadedByUsername: principal.Username,
	})
	writeCreated(c, asset, err, h.writeError)
}

func (h *Handler) CheckMediaResumableUpload(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	var req CheckMediaResumableUploadRequest
	if !bind(c, &req) {
		return
	}
	check, err := h.service.CheckMediaResumableUpload(c.RequestContext(), service.CheckMediaResumableUploadInput{
		CategoryID:         int64(req.CategoryID),
		ChunkSize:          req.ChunkSize,
		ChunkTotal:         req.ChunkTotal,
		FileHash:           req.FileHash,
		Filename:           req.FileName,
		SizeBytes:          req.SizeBytes,
		UploadedBy:         principal.UserID,
		UploadedByUsername: principal.Username,
	})
	writeOK(c, check, err, h.writeError)
}

func (h *Handler) UploadMediaChunk(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	req := c.Request()
	if err := req.ParseMultipartForm(32 << 20); err != nil {
		result.BadRequest(c, result.MessageKeyInvalidRequest)
		return
	}
	file, header, err := req.FormFile("file")
	if err != nil {
		result.BadRequest(c, "validation.common.required", map[string]any{"field": "file"})
		return
	}
	defer file.Close()
	sessionID, ok := parseInt64Form(c, "sessionId", 0)
	if !ok || sessionID <= 0 {
		result.BadRequest(c, "validation.common.invalid", map[string]any{"field": "sessionId"})
		return
	}
	chunkIndex, ok := parseIntForm(c, "chunkIndex", -1)
	if !ok || chunkIndex < 0 {
		result.BadRequest(c, "validation.common.invalid", map[string]any{"field": "chunkIndex"})
		return
	}
	chunkTotal, ok := parseIntForm(c, "chunkTotal", 0)
	if !ok {
		return
	}
	chunk, err := h.service.UploadMediaChunk(c.RequestContext(), service.UploadMediaChunkInput{
		ChunkHash:          req.FormValue("chunkHash"),
		ChunkIndex:         chunkIndex,
		ChunkTotal:         chunkTotal,
		FileHash:           req.FormValue("fileHash"),
		Filename:           req.FormValue("fileName"),
		Reader:             file,
		SessionID:          sessionID,
		Size:               header.Size,
		UploadedBy:         principal.UserID,
		UploadedByUsername: principal.Username,
	})
	writeCreated(c, chunk, err, h.writeError)
}

func (h *Handler) CompleteMediaResumableUpload(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	var req MediaResumableSessionRequest
	if !bind(c, &req) {
		return
	}
	complete, err := h.service.CompleteMediaResumableUpload(c.RequestContext(), service.CompleteMediaResumableUploadInput{
		FileHash:   req.FileHash,
		SessionID:  int64(req.SessionID),
		UploadedBy: principal.UserID,
	})
	writeCreated(c, complete, err, h.writeError)
}

func (h *Handler) AbortMediaResumableUpload(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	var req MediaResumableSessionRequest
	if !bind(c, &req) {
		return
	}
	abort, err := h.service.AbortMediaResumableUpload(c.RequestContext(), service.AbortMediaResumableUploadInput{
		FileHash:   req.FileHash,
		SessionID:  int64(req.SessionID),
		UploadedBy: principal.UserID,
	})
	writeOK(c, abort, err, h.writeError)
}

func (h *Handler) ImportMediaURLs(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	var req ImportMediaURLsRequest
	if !bind(c, &req) {
		return
	}
	importResult, err := h.service.ImportMediaURLs(c.RequestContext(), service.ImportMediaURLsInput{
		CategoryID:         int64(req.CategoryID),
		Items:              req.items(),
		UploadedBy:         principal.UserID,
		UploadedByUsername: principal.Username,
	})
	writeCreated(c, importResult, err, h.writeError)
}

func (h *Handler) UpdateMediaAsset(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "assetId")
	if !ok {
		return
	}
	var req UpdateMediaAssetRequest
	if !bind(c, &req) {
		return
	}
	asset, err := h.service.UpdateMediaAsset(c.RequestContext(), id, service.UpdateMediaAssetInput{DisplayName: req.DisplayName})
	writeOK(c, asset, err, h.writeError)
}

func (h *Handler) DownloadMediaAsset(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "assetId")
	if !ok {
		return
	}
	download, err := h.service.DownloadMediaAsset(c.RequestContext(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": download.Filename}))
	c.Header("Content-Length", strconv.Itoa(len(download.Data)))
	c.Data(http.StatusOK, download.ContentType, download.Data)
}

func (h *Handler) DeleteMediaAsset(c ports.HTTPContext) {
	if _, ok := requirePrincipal(c); !ok {
		return
	}
	id, ok := parseInt64Param(c, "assetId")
	if !ok {
		return
	}
	writeOK(c, map[string]bool{"deleted": true}, h.service.DeleteMediaAsset(c.RequestContext(), id), h.writeError)
}

// RegisterAPIs 接收路由装配阶段发现的 API 清单，并转交系统服务保存。
func (h *Handler) RegisterAPIs(entries []model.APIEntry) {
	h.service.RegisterAPIs(entries)
}

// RecordOperation 让 HTTP 传输层的操作记录中间件复用系统服务审计能力。
func (h *Handler) RecordOperation(ctx context.Context, input service.OperationRecordInput) error {
	return h.service.RecordOperation(ctx, input)
}

// filterMenus 根据当前主体权限过滤菜单项。
// 整个分组没有可见菜单项时会被移除，避免前端展示空栏目。
func (h *Handler) filterMenus(ctx context.Context, principal iamservice.Principal, groups []model.MenuGroup) []model.MenuGroup {
	filtered := make([]model.MenuGroup, 0, len(groups))
	for _, group := range groups {
		items := make([]model.MenuItem, 0, len(group.Items))
		for _, item := range group.Items {
			if item.Permission == "" || h.allowed(ctx, principal, item) {
				items = append(items, item)
			}
		}
		if len(items) == 0 {
			continue
		}
		group.Items = items
		filtered = append(filtered, group)
	}
	return filtered
}

func (h *Handler) localizeMenus(c ports.HTTPContext, groups []model.MenuGroup) []model.MenuGroup {
	out := make([]model.MenuGroup, 0, len(groups))
	for _, group := range groups {
		group.Label = localizeSystemText(c, group.LabelKey, group.Label)
		group.Description = localizeSystemText(c, group.DescriptionKey, group.Description)
		for itemIndex := range group.Items {
			item := &group.Items[itemIndex]
			item.Label = localizeSystemText(c, item.LabelKey, item.Label)
			item.Description = localizeSystemText(c, item.DescriptionKey, item.Description)
		}
		out = append(out, group)
	}
	return out
}

func (h *Handler) localizeConfigSnapshot(c ports.HTTPContext, snapshot model.ConfigSnapshot) model.ConfigSnapshot {
	for sectionIndex := range snapshot.Sections {
		section := &snapshot.Sections[sectionIndex]
		section.Label = localizeSystemText(c, section.LabelKey, section.Label)
		section.Description = localizeSystemText(c, section.DescriptionKey, section.Description)
		for itemIndex := range section.Items {
			item := &section.Items[itemIndex]
			localizeConfigItem(c, item)
		}
		for groupIndex := range section.Groups {
			group := &section.Groups[groupIndex]
			group.Label = localizeSystemText(c, group.LabelKey, group.Label)
			group.Description = localizeSystemText(c, group.DescriptionKey, group.Description)
			for itemIndex := range group.Items {
				item := &group.Items[itemIndex]
				localizeConfigItem(c, item)
			}
		}
	}
	return snapshot
}

func localizeConfigItem(c ports.HTTPContext, item *model.ConfigItem) {
	item.Label = localizeSystemText(c, item.LabelKey, item.Label)
	item.Description = localizeSystemText(c, item.DescriptionKey, item.Description)
	if item.Secret {
		if value, ok := item.Value.(string); ok {
			item.Value = localizeSystemText(c, "system.config.secret."+value, value)
		}
	}
	for optionIndex := range item.Options {
		option := &item.Options[optionIndex]
		option.Label = localizeSystemText(c, option.LabelKey, option.Label)
		option.Description = localizeSystemText(c, option.DescriptionKey, option.Description)
	}
}

func localizeSystemText(c ports.HTTPContext, fullKey string, fallback string) string {
	fullKey = strings.TrimSpace(fullKey)
	if fullKey == "" {
		return fallback
	}
	namespace, key := splitMessageKey(fullKey)
	if value, ok := c.Get("i18n"); ok {
		if localizer, ok := value.(ports.I18n); ok {
			locale := localizer.DefaultLocale()
			if raw, ok := c.Get("locale"); ok {
				if text, ok := raw.(string); ok && strings.TrimSpace(text) != "" {
					locale = text
				}
			}
			resolved := localizer.Localize(locale, namespace, key, nil)
			if resolved != key && resolved != fullKey {
				return resolved
			}
		}
	}
	return fallback
}

func splitMessageKey(fullKey string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(fullKey), ".", 2)
	if len(parts) != 2 {
		return "system", fullKey
	}
	return parts[0], parts[1]
}

// allowed 校验单个菜单权限编码。
// 权限编码必须是 object:action 形式；授权器缺失时默认拒绝。
func (h *Handler) allowed(ctx context.Context, principal iamservice.Principal, item model.MenuItem) bool {
	if h.authorizer == nil {
		return false
	}
	obj, act := permissionObjectAction(item.Permission)
	if obj == "" || act == "" {
		return false
	}
	allowed, err := h.authorizer.Authorize(ctx, principal, iamservice.PermissionContext{
		ProductCode: item.ProductCode,
		Scope:       item.Scope,
		Object:      obj,
		Action:      act,
	})
	return err == nil && allowed
}

// writeError 将系统服务层错误映射为 HTTP 响应。
// 未知错误只记录日志并返回通用 500，避免暴露内部细节。
func (h *Handler) writeError(c ports.HTTPContext, err error) {
	switch {
	case errors.Is(err, context.Canceled):
		result.Fail(c, http.StatusRequestTimeout, "api.common.requestCanceled")
	case errors.Is(err, service.ErrInvalidInput), errors.Is(err, service.ErrDuplicate), errors.Is(err, service.ErrExternalMedia):
		result.BadRequest(c, result.MessageKeyInvalidRequest)
	case errors.Is(err, service.ErrNotFound):
		result.NotFound(c, result.MessageKeyNotFound)
	case errors.Is(err, service.ErrConfigUnavailable), errors.Is(err, service.ErrStorageUnavailable):
		result.Fail(c, http.StatusServiceUnavailable, "api.common.notReady")
	default:
		if h.logger != nil {
			h.logger.Error("system request failed", "error", err)
		}
		result.InternalError(c, result.MessageKeyInternalError)
	}
}

// requirePrincipal 从认证中间件结果中读取 Principal。
func requirePrincipal(c ports.HTTPContext) (iamservice.Principal, bool) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		result.Unauthorized(c, "api.auth.missingPrincipal")
		return iamservice.Principal{}, false
	}
	return principal, true
}

// bind 绑定 JSON 请求体；失败时统一写 400 响应。
func bind(c ports.HTTPContext, dest any) bool {
	if err := c.BindJSON(dest); err != nil {
		result.BadRequest(c, result.MessageKeyInvalidRequest)
		return false
	}
	return true
}

// parseInt64Param 解析正整数路径参数。
func parseInt64Param(c ports.HTTPContext, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		result.BadRequest(c, "validation.common.invalid", map[string]any{"field": name})
		return 0, false
	}
	return id, true
}

// parseIntQuery 解析整数查询参数，空值使用 fallback。
func parseIntQuery(c ports.HTTPContext, name string, fallback int) (int, bool) {
	raw := strings.TrimSpace(c.Request().URL.Query().Get(name))
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		result.BadRequest(c, "validation.common.invalidNumber", map[string]any{"field": name})
		return 0, false
	}
	return value, true
}

// parseInt64Query 解析 int64 查询参数，空值使用 fallback。
func parseInt64Query(c ports.HTTPContext, name string, fallback int64) (int64, bool) {
	raw := strings.TrimSpace(c.Request().URL.Query().Get(name))
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		result.BadRequest(c, "validation.common.invalidNumber", map[string]any{"field": name})
		return 0, false
	}
	return value, true
}

// parseInt64Form 解析 multipart form 中的 int64 字段。
func parseInt64Form(c ports.HTTPContext, name string, fallback int64) (int64, bool) {
	raw := strings.TrimSpace(c.Request().FormValue(name))
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		result.BadRequest(c, "validation.common.invalidNumber", map[string]any{"field": name})
		return 0, false
	}
	return value, true
}

// parseIntForm 解析 multipart form 中的整数字段。
func parseIntForm(c ports.HTTPContext, name string, fallback int) (int, bool) {
	raw := strings.TrimSpace(c.Request().FormValue(name))
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		result.BadRequest(c, "validation.common.invalidNumber", map[string]any{"field": name})
		return 0, false
	}
	return value, true
}

// parseTimeQuery 解析时间查询参数。
// 支持 RFC3339、常见日期时间和日期格式；日期上界会转换为次日零点以形成半开区间。
func parseTimeQuery(c ports.HTTPContext, name string, endExclusive bool) (*time.Time, bool) {
	raw := strings.TrimSpace(c.Request().URL.Query().Get(name))
	if raw == "" {
		return nil, true
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		value, err := time.Parse(layout, raw)
		if err != nil {
			continue
		}
		if layout == "2006-01-02" && endExclusive {
			value = value.AddDate(0, 0, 1)
		}
		return &value, true
	}
	result.BadRequest(c, "validation.common.invalid", map[string]any{"field": name})
	return nil, false
}

// int64s 将请求中的 systemID 切片转换为服务层使用的 int64。
func (r DeleteOperationRecordsRequest) int64s() []int64 {
	ids := make([]int64, 0, len(r.IDs))
	for _, id := range r.IDs {
		ids = append(ids, int64(id))
	}
	return ids
}

// int64s 将批量参数删除 ID 转换为 int64。
func (r DeleteParametersRequest) int64s() []int64 {
	ids := make([]int64, 0, len(r.IDs))
	for _, id := range r.IDs {
		ids = append(ids, int64(id))
	}
	return ids
}

// int64s 将批量版本删除 ID 转换为 int64。
func (r DeleteVersionsRequest) int64s() []int64 {
	ids := make([]int64, 0, len(r.IDs))
	for _, id := range r.IDs {
		ids = append(ids, int64(id))
	}
	return ids
}

// items 将结构化列表或多行文本转换为媒体 URL 导入项。
// 文本模式支持 “名称|URL” 和纯 URL 两种行格式。
func (r ImportMediaURLsRequest) items() []service.MediaURLImportItem {
	items := make([]service.MediaURLImportItem, 0, len(r.Items))
	for _, item := range r.Items {
		items = append(items, service.MediaURLImportItem{Name: item.Name, URL: item.URL})
	}
	if len(items) > 0 {
		return items
	}
	for _, line := range strings.Split(r.Text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name := ""
		rawURL := line
		if left, right, ok := strings.Cut(line, "|"); ok {
			name = strings.TrimSpace(left)
			rawURL = strings.TrimSpace(right)
		}
		items = append(items, service.MediaURLImportItem{Name: name, URL: rawURL})
	}
	return items
}

// UnmarshalJSON 将 JSON 字符串或数字解析为系统资源 ID。
func (id *systemID) UnmarshalJSON(raw []byte) error {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		return service.ErrInvalidInput
	}
	if strings.HasPrefix(value, `"`) {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return err
		}
		value = strings.TrimSpace(unquoted)
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	*id = systemID(parsed)
	return nil
}

// writeOK 根据服务层结果写入 200 或统一错误响应。
func writeOK(c ports.HTTPContext, data any, err error, writeError func(ports.HTTPContext, error)) {
	if err != nil {
		writeError(c, err)
		return
	}
	result.OK(c, data)
}

func writeSSE(writer http.ResponseWriter, flusher http.Flusher, event string, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		raw = []byte(`{"error":"encode failed"}`)
	}
	_, _ = fmt.Fprintf(writer, "event: %s\n", event)
	_, _ = fmt.Fprintf(writer, "data: %s\n\n", raw)
	flusher.Flush()
}

// writeCreated 根据服务层结果写入 201 或统一错误响应。
func writeCreated(c ports.HTTPContext, data any, err error, writeError func(ports.HTTPContext, error)) {
	if err != nil {
		writeError(c, err)
		return
	}
	result.Created(c, data)
}

// permissionObjectAction 将 object:action 权限编码拆成授权器需要的两个维度。
func permissionObjectAction(code string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(code), ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	return parts[0], parts[1]
}

// Package httptransport 实现远程插件协议的 HTTP transport adapter。
package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

// RemoteInvoker 通过 HTTP POST 调用远程插件实例。
// client 由调用方注入以复用超时、代理和连接池配置；为空时使用安全的默认超时。
type RemoteInvoker struct {
	client *http.Client
}

// remoteResponse 兼容 HTTP transport 的统一响应 envelope。
// Data 保存业务响应，Error 保存远端协议错误。
type remoteResponse struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error *protocol.Error `json:"error,omitempty"`
}

// NewRemoteInvoker 创建 HTTP 远程调用器。
func NewRemoteInvoker(client *http.Client) *RemoteInvoker {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &RemoteInvoker{client: client}
}

// Invoke 调用远程插件能力并返回 result 原始 JSON。
// remote.Endpoint 是插件 HTTP 基地址；req 会作为 /invoke 的 JSON 请求体发送。
func (i *RemoteInvoker) Invoke(ctx context.Context, remote protocol.PluginSnapshot, req protocol.InvokeRequest) (json.RawMessage, error) {
	var response protocol.InvokeResponse
	if err := i.post(ctx, remote.Endpoint, "/invoke", req, &response); err != nil {
		return nil, err
	}
	return append(json.RawMessage(nil), response.Result...), nil
}

// PushEvent 通过 HTTP transport 向远程插件推送事件。
func (i *RemoteInvoker) PushEvent(ctx context.Context, remote protocol.PluginSnapshot, req protocol.PushEventRequest) error {
	return i.post(ctx, remote.Endpoint, "/events", req, nil)
}

// post 执行 HTTP transport 的公共请求流程。
// out 非 nil 时会优先解析统一 envelope 中的 Data；若远端直接返回裸 JSON，也会兼容解码。
func (i *RemoteInvoker) post(ctx context.Context, endpoint string, path string, payload any, out any) error {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return plugin.ErrTransportUnavailable
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := i.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: http status %d: %s", plugin.ErrTransportUnavailable, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var decoded remoteResponse
	if err := json.Unmarshal(raw, &decoded); err == nil && (decoded.Error != nil || len(decoded.Data) > 0) {
		if decoded.Error != nil {
			return fmt.Errorf("%w: %s: %s", plugin.ErrTransportUnavailable, decoded.Error.Code, decoded.Error.Message)
		}
		if out != nil {
			return json.Unmarshal(decoded.Data, out)
		}
		return nil
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

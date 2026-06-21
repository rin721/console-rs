// Package rpctransport 实现远程插件协议的 JSON-RPC transport adapter。
package rpctransport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

// RemoteInvoker 通过 JSON-RPC over HTTP 调用远程插件实例。
// ids 为每次请求生成递增 ID，便于远端日志和协议调试关联请求响应。
type RemoteInvoker struct {
	client *http.Client
	ids    atomic.Uint64
}

// request 是发送给远程插件的 JSON-RPC 2.0 请求 envelope。
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// response 是远程插件返回的 JSON-RPC 2.0 响应 envelope。
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// rpcError 描述 JSON-RPC 错误对象。
type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// NewRemoteInvoker 创建 JSON-RPC 远程调用器。
func NewRemoteInvoker(client *http.Client) *RemoteInvoker {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &RemoteInvoker{client: client}
}

// Invoke 调用远程插件能力并返回 result 原始 JSON。
// 如果远端返回完整 InvokeResponse，会剥离外层并只返回 Result，保持 Router 调用语义一致。
func (i *RemoteInvoker) Invoke(ctx context.Context, remote protocol.PluginSnapshot, req protocol.InvokeRequest) (json.RawMessage, error) {
	raw, err := i.call(ctx, remote.Endpoint, MethodInvoke, req)
	if err != nil {
		return nil, err
	}
	var response protocol.InvokeResponse
	if err := json.Unmarshal(raw, &response); err == nil && response.Capability != "" {
		return append(json.RawMessage(nil), response.Result...), nil
	}
	return append(json.RawMessage(nil), raw...), nil
}

// PushEvent 通过 JSON-RPC transport 推送事件。
func (i *RemoteInvoker) PushEvent(ctx context.Context, remote protocol.PluginSnapshot, req protocol.PushEventRequest) error {
	_, err := i.call(ctx, remote.Endpoint, MethodPushEvent, req)
	return err
}

// call 执行单次 JSON-RPC 请求并返回 result 原始 JSON。
// endpoint 是完整 RPC 地址；HTTP 或 JSON-RPC 错误都会映射为 transport unavailable，
// 让上层路由可以按 transport 失败语义重试其他实例。
func (i *RemoteInvoker) call(ctx context.Context, endpoint string, method string, params any) (json.RawMessage, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, plugin.ErrTransportUnavailable
	}
	rawParams, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	id := fmt.Sprintf("%d", i.ids.Add(1))
	body, err := json.Marshal(request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  rawParams,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := i.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%w: rpc http status %d: %s", plugin.ErrTransportUnavailable, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var decoded response
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	if decoded.Error != nil {
		return nil, fmt.Errorf("%w: rpc error %d: %s", plugin.ErrTransportUnavailable, decoded.Error.Code, decoded.Error.Message)
	}
	return append(json.RawMessage(nil), decoded.Result...), nil
}

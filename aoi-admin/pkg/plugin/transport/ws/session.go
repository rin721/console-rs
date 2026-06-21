package ws

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/rei0721/go-scaffold/pkg/plugin"
	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

// SessionRegistry 维护远程插件实例到 WebSocket 会话的映射。
//
// key 使用 plugin_id 与 instance_id 共同组成，因为同一插件可以有多个运行实例。
type SessionRegistry struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// Session 表示一条已升级的 WebSocket 连接。
//
// writer 由 HTTP hijack 后的 bufio.Writer 提供；pending 用 envelope ID 关联主系统发出的请求和插件返回的响应。
type Session struct {
	writer *bufio.Writer

	// WebSocket 连接上的写操作必须串行化，避免响应帧、pong 帧和反向调用帧交错写入。
	writeMu sync.Mutex

	pendingMu sync.Mutex
	pending   map[string]chan protocol.Envelope
	ids       atomic.Uint64

	mu         sync.RWMutex
	pluginID   string
	instanceID string
}

// RemoteInvoker 通过已绑定的 WebSocket 会话向远程插件发起反向调用。
type RemoteInvoker struct {
	sessions *SessionRegistry
}

// NewSessionRegistry 创建空会话注册表。
func NewSessionRegistry() *SessionRegistry {
	return &SessionRegistry{sessions: map[string]*Session{}}
}

// NewSession 创建 WebSocket 会话对象。
//
// 调用方需要在连接生命周期内负责读取 frame，并在收到 response envelope 时调用 Complete。
func NewSession(writer *bufio.Writer) *Session {
	return &Session{
		writer:  writer,
		pending: map[string]chan protocol.Envelope{},
	}
}

// Bind 将插件实例绑定到一条 WebSocket 会话。
//
// 注册成功后才绑定，确保主系统只会向已经通过协议注册的实例发起反向调用。
func (r *SessionRegistry) Bind(pluginID string, instanceID string, session *Session) {
	if r == nil || session == nil || pluginID == "" || instanceID == "" {
		return
	}
	session.setPlugin(pluginID, instanceID)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[sessionKey(pluginID, instanceID)] = session
}

// Get 根据插件实例标识查找当前可用会话。
func (r *SessionRegistry) Get(pluginID string, instanceID string) (*Session, bool) {
	if r == nil || pluginID == "" || instanceID == "" {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[sessionKey(pluginID, instanceID)]
	return session, ok
}

// Unbind 移除指定插件实例的会话绑定。
func (r *SessionRegistry) Unbind(pluginID string, instanceID string) {
	if r == nil || pluginID == "" || instanceID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, sessionKey(pluginID, instanceID))
}

// UnbindSession 按会话对象移除所有绑定。
//
// 连接关闭时使用该方法兜底清理，覆盖插件未显式 unregister 或同一连接被重新绑定的情况。
func (r *SessionRegistry) UnbindSession(session *Session) {
	if r == nil || session == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for pluginID, candidate := range r.sessions {
		if candidate == session {
			delete(r.sessions, pluginID)
		}
	}
}

// Write 将协议 envelope 作为单个 text frame 写回 WebSocket。
//
// 该方法内部加锁并 flush，因此调用方不需要关心控制帧和业务帧的写入竞争。
func (s *Session) Write(envelope protocol.Envelope) error {
	if s == nil || s.writer == nil {
		return plugin.ErrTransportUnavailable
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := writeFrame(s.writer, websocketOpcodeText, raw); err != nil {
		return err
	}
	return s.writer.Flush()
}

// Request 通过 WebSocket 向远程插件发送请求，并等待同 ID 的 response envelope。
//
// operation 是协议操作名，payload 会被编码进 envelope。ctx 负责控制等待响应的最长生命周期；
// 方法返回后会清理 pending 表项，避免超时或写入失败后泄漏等待通道。
func (s *Session) Request(ctx context.Context, operation string, payload any) (protocol.Envelope, error) {
	if s == nil {
		return protocol.Envelope{}, plugin.ErrTransportUnavailable
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return protocol.Envelope{}, err
	}
	id := strconv.FormatUint(s.ids.Add(1), 10)
	wait := make(chan protocol.Envelope, 1)
	s.pendingMu.Lock()
	s.pending[id] = wait
	s.pendingMu.Unlock()
	defer func() {
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
	}()

	if err := s.Write(protocol.Envelope{
		ID:        id,
		Type:      "request",
		Operation: operation,
		Version:   protocol.ProtocolVersionV1,
		Payload:   raw,
	}); err != nil {
		return protocol.Envelope{}, err
	}

	select {
	case <-ctx.Done():
		return protocol.Envelope{}, ctx.Err()
	case envelope := <-wait:
		if envelope.Error != nil {
			// WebSocket response 中的协议错误表示远端未完成请求，统一映射为 transport 层错误供 router 重试或降级。
			return envelope, fmt.Errorf("%w: %s: %s", plugin.ErrTransportUnavailable, envelope.Error.Code, envelope.Error.Message)
		}
		return envelope, nil
	}
}

// Complete 用 response envelope 唤醒等待相同 ID 的 Request。
//
// 返回 false 表示该响应不是当前会话发出的反向调用结果，调用方应继续按普通插件请求处理。
func (s *Session) Complete(envelope protocol.Envelope) bool {
	if s == nil || envelope.ID == "" {
		return false
	}
	s.pendingMu.Lock()
	wait, ok := s.pending[envelope.ID]
	s.pendingMu.Unlock()
	if !ok {
		return false
	}
	wait <- envelope
	return true
}

// setPlugin 记录当前会话已绑定的插件实例标识，供后续调试或扩展会话级状态使用。
func (s *Session) setPlugin(pluginID string, instanceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pluginID = pluginID
	s.instanceID = instanceID
}

// NewRemoteInvoker 创建 WebSocket 反向调用器。
func NewRemoteInvoker(registry *SessionRegistry) *RemoteInvoker {
	return &RemoteInvoker{sessions: registry}
}

// Invoke 通过绑定会话调用远程插件能力，并返回远端 result payload。
//
// remote 必须包含 plugin_id 和 instance_id；缺少绑定时返回 transport unavailable，让上层路由尝试其他实例。
func (i *RemoteInvoker) Invoke(ctx context.Context, remote protocol.PluginSnapshot, req protocol.InvokeRequest) (json.RawMessage, error) {
	if i == nil || i.sessions == nil {
		return nil, plugin.ErrTransportUnavailable
	}
	session, ok := i.sessions.Get(remote.PluginID, remote.InstanceID)
	if !ok {
		return nil, plugin.ErrTransportUnavailable
	}
	envelope, err := session.Request(ctx, protocol.OperationInvoke, req)
	if err != nil {
		return nil, err
	}
	var response protocol.InvokeResponse
	if err := json.Unmarshal(envelope.Payload, &response); err != nil {
		return nil, err
	}
	return append(json.RawMessage(nil), response.Result...), nil
}

// PushEvent 通过绑定会话向远程插件推送事件。
func (i *RemoteInvoker) PushEvent(ctx context.Context, remote protocol.PluginSnapshot, req protocol.PushEventRequest) error {
	if i == nil || i.sessions == nil {
		return plugin.ErrTransportUnavailable
	}
	session, ok := i.sessions.Get(remote.PluginID, remote.InstanceID)
	if !ok {
		return plugin.ErrTransportUnavailable
	}
	_, err := session.Request(ctx, protocol.OperationPushEvent, req)
	return err
}

// sessionKey 使用 NUL 分隔 plugin_id 和 instance_id，避免普通标识符拼接时出现歧义。
func sessionKey(pluginID string, instanceID string) string {
	return pluginID + "\x00" + instanceID
}

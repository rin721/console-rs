package ws

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/rei0721/go-scaffold/pkg/plugin/protocol"
)

// WebSocket 协议常量集中在这里，maxWebSocketFrame 用于限制远端插件单帧占用的内存上限。
const (
	websocketGUID        = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	maxWebSocketFrame    = 1 << 20
	websocketOpcodeText  = 0x1
	websocketOpcodeClose = 0x8
	websocketOpcodePing  = 0x9
	websocketOpcodePong  = 0xA
)

// ErrInvalidWebSocketHandshake 表示请求不满足 WebSocket upgrade 的最小握手条件。
var ErrInvalidWebSocketHandshake = errors.New("invalid websocket handshake")

// Serve 完成 HTTP 到 WebSocket 的升级，并在同一连接上处理插件协议 envelope。
//
// 该实现只依赖 net/http 的 Hijacker，不引入额外 WebSocket 库；调用方通过 ctx 控制连接处理生命周期。
func (d *Dispatcher) Serve(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if !isWebSocketRequest(r) {
		return ErrInvalidWebSocketHandshake
	}
	key := strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key"))
	if key == "" {
		return ErrInvalidWebSocketHandshake
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return fmt.Errorf("%w: response writer does not support hijacking", ErrInvalidWebSocketHandshake)
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := rw.WriteString("HTTP/1.1 101 Switching Protocols\r\n"); err != nil {
		return err
	}
	if _, err := rw.WriteString("Upgrade: websocket\r\n"); err != nil {
		return err
	}
	if _, err := rw.WriteString("Connection: Upgrade\r\n"); err != nil {
		return err
	}
	if _, err := rw.WriteString("Sec-WebSocket-Accept: " + acceptKey(key) + "\r\n\r\n"); err != nil {
		return err
	}
	if err := rw.Flush(); err != nil {
		return err
	}
	// 会话在 register 响应后才会绑定到插件实例；连接退出时无论是否 unregister 都会清理绑定。
	session := NewSession(rw.Writer)
	defer d.sessions.UnbindSession(session)
	return d.serveFrames(ctx, conn, rw.Reader, session)
}

// serveFrames 循环读取 WebSocket frame，并在 text frame 中执行插件协议分发。
//
// response envelope 会优先用于唤醒本端发起的 pending 请求；其他 envelope 才进入 Host 分发流程。
func (d *Dispatcher) serveFrames(ctx context.Context, conn net.Conn, reader *bufio.Reader, session *Session) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		opcode, payload, err := readFrame(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		switch opcode {
		case websocketOpcodeClose:
			// close 帧按协议回写 close，随后退出循环，让 defer 负责连接和会话清理。
			_ = writeControlFrame(session.writer, websocketOpcodeClose, nil, &session.writeMu)
			return nil
		case websocketOpcodePing:
			if err := writeControlFrame(session.writer, websocketOpcodePong, payload, &session.writeMu); err != nil {
				return err
			}
		case websocketOpcodeText:
			var envelope protocol.Envelope
			if err := json.Unmarshal(payload, &envelope); err != nil {
				envelope = protocol.Envelope{
					Type:  "response",
					Error: &protocol.Error{Code: protocol.ErrorCodeInvalidPlugin, Message: "invalid websocket envelope"},
				}
			} else {
				// 远程插件对反向调用的响应不会再进入 Host，否则会被误当成新的协议请求。
				if strings.EqualFold(envelope.Type, "response") && session.Complete(envelope) {
					continue
				}
				envelope = d.Dispatch(ctx, envelope)
				d.bindSessionAfterDispatch(session, envelope)
			}
			if err := session.Write(envelope); err != nil {
				return err
			}
		default:
			_ = writeControlFrame(session.writer, websocketOpcodeClose, nil, &session.writeMu)
			_ = conn.Close()
			return nil
		}
	}
}

// bindSessionAfterDispatch 根据注册或注销响应维护会话绑定。
//
// 只在 Host 成功处理后更新绑定，避免认证失败或协议错误时把未授权连接纳入反向调用池。
func (d *Dispatcher) bindSessionAfterDispatch(session *Session, envelope protocol.Envelope) {
	if d == nil || d.sessions == nil || session == nil || envelope.Error != nil {
		return
	}
	switch envelope.Operation {
	case protocol.OperationRegister:
		var response protocol.RegisterResponse
		if err := json.Unmarshal(envelope.Payload, &response); err == nil && response.Plugin.PluginID != "" && response.Plugin.InstanceID != "" {
			d.sessions.Bind(response.Plugin.PluginID, response.Plugin.InstanceID, session)
		}
	case protocol.OperationUnregister:
		var response protocol.UnregisterResponse
		if err := json.Unmarshal(envelope.Payload, &response); err == nil && response.PluginID != "" && response.InstanceID != "" {
			d.sessions.Unbind(response.PluginID, response.InstanceID)
		}
	}
}

// isWebSocketRequest 判断请求是否声明了 WebSocket upgrade。
func isWebSocketRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") && strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// acceptKey 按 RFC 6455 计算 Sec-WebSocket-Accept 响应值。
func acceptKey(key string) string {
	sum := sha1.Sum([]byte(key + websocketGUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// readFrame 读取一个完整 WebSocket frame 并返回 opcode 与解码后的 payload。
//
// 客户端到服务端的 frame 通常带 mask；这里在读取 payload 后就地解 mask，并对扩展长度做上限保护。
func readFrame(reader *bufio.Reader) (byte, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return 0, nil, err
	}
	opcode := header[0] & 0x0F
	masked := header[1]&0x80 != 0
	length := uint64(header[1] & 0x7F)
	switch length {
	case 126:
		extended := make([]byte, 2)
		if _, err := io.ReadFull(reader, extended); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(extended))
	case 127:
		extended := make([]byte, 8)
		if _, err := io.ReadFull(reader, extended); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(extended)
	}
	if length > maxWebSocketFrame {
		return 0, nil, fmt.Errorf("websocket frame too large: %d", length)
	}
	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(reader, mask[:]); err != nil {
			return 0, nil, err
		}
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return opcode, payload, nil
}

// writeFrame 写入单个服务端到客户端的 WebSocket frame。
//
// 服务端发送帧按协议不使用 mask；调用方负责在并发场景下持有写锁并在需要时 flush。
func writeFrame(writer *bufio.Writer, opcode byte, payload []byte) error {
	header := []byte{0x80 | opcode}
	length := len(payload)
	switch {
	case length < 126:
		header = append(header, byte(length))
	case length <= 0xFFFF:
		header = append(header, 126, byte(length>>8), byte(length))
	default:
		header = append(header, 127)
		var extended [8]byte
		binary.BigEndian.PutUint64(extended[:], uint64(length))
		header = append(header, extended[:]...)
	}
	if _, err := writer.Write(header); err != nil {
		return err
	}
	if len(payload) == 0 {
		return nil
	}
	_, err := writer.Write(payload)
	return err
}

// writeControlFrame 写入 close/pong 等控制帧。
//
// 控制帧可能和业务响应来自不同路径，因此允许传入写锁来复用 Session 的串行化策略。
func writeControlFrame(writer *bufio.Writer, opcode byte, payload []byte, mu *sync.Mutex) error {
	if writer == nil {
		return nil
	}
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	if err := writeFrame(writer, opcode, payload); err != nil {
		return err
	}
	return writer.Flush()
}

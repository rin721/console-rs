package rpcserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// NewHandler 创建包含 /rpc 和 /health 的 HTTP handler。
func NewHandler(registry *Registry) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", handleRPC(registry))
	mux.HandleFunc("/health", handleHealth(registry))
	return mux
}

// handleHealth 返回轻量健康信息，并暴露已注册方法数量用于部署侧探活和排障。
func handleHealth(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"methods": len(registry.Methods()),
		})
	}
}

// handleRPC 实现最小 JSON-RPC 2.0 HTTP 入口：只接受单个 POST 请求，并用标准响应体表达业务错误。
func handleRPC(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse(nil, CodeInvalidRequest, "method must be POST", nil))
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse(nil, CodeParseError, "parse error", nil))
			return
		}
		defer r.Body.Close()

		var raw any
		if err := json.Unmarshal(body, &raw); err != nil {
			writeJSON(w, http.StatusOK, errorResponse(nil, CodeParseError, "parse error", nil))
			return
		}
		// 暂不支持 batch，提前在原始 JSON 层拒绝数组可避免后续请求结构被误解析。
		if _, ok := raw.([]any); ok {
			writeJSON(w, http.StatusOK, errorResponse(nil, CodeInvalidRequest, "batch requests are not supported", nil))
			return
		}

		var req Request
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusOK, errorResponse(nil, CodeInvalidRequest, "invalid request", nil))
			return
		}
		if !validRequest(req) {
			writeJSON(w, http.StatusOK, errorResponse(req.ID, CodeInvalidRequest, "invalid request", nil))
			return
		}

		result, err := registry.Call(r.Context(), req.Method, req.Params)
		if err != nil {
			var rpcErr *RPCError
			if errors.As(err, &rpcErr) {
				writeJSON(w, http.StatusOK, errorResponse(req.ID, rpcErr.Code, rpcErr.Message, rpcErr.Data))
				return
			}
			writeJSON(w, http.StatusOK, errorResponse(req.ID, CodeInternalError, "internal error", nil))
			return
		}

		writeJSON(w, http.StatusOK, Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		})
	}
}

// validRequest 保留最小协议校验，具体参数语义由注册方法自行判断。
func validRequest(req Request) bool {
	return req.JSONRPC == "2.0" && len(req.ID) > 0 && req.Method != ""
}

// errorResponse 统一 JSON-RPC 错误响应形态，避免不同分支返回不一致的 error 对象。
func errorResponse(id json.RawMessage, code int, message string, data any) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// writeJSON 写出 JSON 响应；编码错误只能被忽略，因为此时 HTTP 头和状态码已经发出。
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

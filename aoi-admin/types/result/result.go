package result

import "time"

const (
	MessageKeySuccess        = "api.common.success"
	MessageKeyInvalidRequest = "api.common.invalidRequest"
	MessageKeyUnauthorized   = "api.auth.unauthorized"
	MessageKeyForbidden      = "api.auth.forbidden"
	MessageKeyNotFound       = "api.common.notFound"
	MessageKeyInternalError  = "api.common.internalError"
	MessageKeyRateLimited    = "api.common.rateLimited"
)

type Result[T any] struct {
	Code        int            `json:"code"`
	MessageKey  string         `json:"messageKey"`
	Message     string         `json:"message"`
	MessageArgs map[string]any `json:"messageArgs,omitempty"`
	Data        T              `json:"data,omitempty"`
	TraceID     string         `json:"traceId,omitempty"`
	ServerTime  int64          `json:"serverTime"`
}

func Success[T any](data T) *Result[T] {
	return SuccessWithMessage(data, MessageKeySuccess, MessageKeySuccess, nil)
}

func SuccessWithMessage[T any](data T, messageKey string, message string, args map[string]any) *Result[T] {
	return &Result[T]{
		Code:        0,
		MessageKey:  messageKey,
		Message:     message,
		MessageArgs: cloneArgs(args),
		Data:        data,
		ServerTime:  time.Now().Unix(),
	}
}

func Error(code int, messageKey string) *Result[any] {
	return ErrorWithTraceArgs(code, messageKey, messageKey, nil, "")
}

func ErrorWithTrace(code int, messageKey string, traceID string) *Result[any] {
	return ErrorWithTraceArgs(code, messageKey, messageKey, nil, traceID)
}

func ErrorWithTraceArgs(code int, messageKey string, message string, args map[string]any, traceID string) *Result[any] {
	return &Result[any]{
		Code:        code,
		MessageKey:  messageKey,
		Message:     message,
		MessageArgs: cloneArgs(args),
		TraceID:     traceID,
		ServerTime:  time.Now().Unix(),
	}
}

func cloneArgs(args map[string]any) map[string]any {
	if len(args) == 0 {
		return nil
	}
	out := make(map[string]any, len(args))
	for key, value := range args {
		out[key] = value
	}
	return out
}

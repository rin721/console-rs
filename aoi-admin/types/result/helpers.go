package result

import (
	"net/http"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/types/errors"
)

const (
	localeContextKey = "locale"
	i18nContextKey   = "i18n"
	defaultLocale    = "zh-CN"
)

type HTTPContext interface {
	JSON(status int, body any)
	Get(key any) (any, bool)
}

func nowUnix() int64 {
	return time.Now().Unix()
}

type Localizer interface {
	Localize(locale string, namespace string, key string, data map[string]any) string
	DefaultLocale() string
}

func Unauthorized(c HTTPContext, messageKey string, args ...map[string]any) {
	writeError(c, http.StatusUnauthorized, errors.ErrUnauthorized, messageKeyOrDefault(messageKey, MessageKeyUnauthorized), firstArgs(args))
}

func BadRequest(c HTTPContext, messageKey string, args ...map[string]any) {
	writeError(c, http.StatusBadRequest, errors.ErrInvalidParams, messageKeyOrDefault(messageKey, MessageKeyInvalidRequest), firstArgs(args))
}

func NotFound(c HTTPContext, messageKey string, args ...map[string]any) {
	writeError(c, http.StatusNotFound, errors.ErrResourceNotFound, messageKeyOrDefault(messageKey, MessageKeyNotFound), firstArgs(args))
}

func InternalError(c HTTPContext, messageKey string, args ...map[string]any) {
	writeError(c, http.StatusInternalServerError, errors.ErrInternalServer, messageKeyOrDefault(messageKey, MessageKeyInternalError), firstArgs(args))
}

func Forbidden(c HTTPContext, messageKey string, args ...map[string]any) {
	writeError(c, http.StatusForbidden, errors.ErrPermissionDenied, messageKeyOrDefault(messageKey, MessageKeyForbidden), firstArgs(args))
}

func Fail(c HTTPContext, httpStatus int, messageKey string, args ...map[string]any) {
	code := errors.ErrInternalServer
	defaultKey := MessageKeyInternalError
	switch httpStatus {
	case http.StatusBadRequest:
		code = errors.ErrInvalidParams
		defaultKey = MessageKeyInvalidRequest
	case http.StatusUnauthorized:
		code = errors.ErrUnauthorized
		defaultKey = MessageKeyUnauthorized
	case http.StatusForbidden:
		code = errors.ErrPermissionDenied
		defaultKey = MessageKeyForbidden
	case http.StatusNotFound:
		code = errors.ErrResourceNotFound
		defaultKey = MessageKeyNotFound
	case http.StatusTooManyRequests:
		code = errors.ErrBusinessLogic
		defaultKey = MessageKeyRateLimited
	}
	writeError(c, httpStatus, code, messageKeyOrDefault(messageKey, defaultKey), firstArgs(args))
}

func LocalizedError(c HTTPContext, code int, messageKey string, args map[string]any, traceID string) *Result[any] {
	return ErrorWithTraceArgs(code, messageKey, localize(c, messageKey, args), args, traceID)
}

func LocalizedErrorWithData[T any](c HTTPContext, code int, messageKey string, args map[string]any, traceID string, data T) *Result[T] {
	return &Result[T]{
		Code:        code,
		MessageKey:  messageKey,
		Message:     localize(c, messageKey, args),
		MessageArgs: cloneArgs(args),
		Data:        data,
		TraceID:     traceID,
		ServerTime:  nowUnix(),
	}
}

func OK[T any](c HTTPContext, data T) {
	key := MessageKeySuccess
	c.JSON(http.StatusOK, SuccessWithMessage(data, key, localize(c, key, nil), nil))
}

func Created[T any](c HTTPContext, data T) {
	key := MessageKeySuccess
	c.JSON(http.StatusCreated, SuccessWithMessage(data, key, localize(c, key, nil), nil))
}

func Page[T any](c HTTPContext, list []T, total int64, page, pageSize int) {
	OK(c, NewPageResult(list, page, pageSize, total))
}

func GetTraceID(c HTTPContext) string {
	for _, key := range []string{"traceId", "trace_id"} {
		traceID, exists := c.Get(key)
		if !exists {
			continue
		}
		id, ok := traceID.(string)
		if !ok {
			return ""
		}
		return id
	}
	return ""
}

func writeError(c HTTPContext, httpStatus int, code int, messageKey string, args map[string]any) {
	c.JSON(httpStatus, ErrorWithTraceArgs(
		code,
		messageKey,
		localize(c, messageKey, args),
		args,
		GetTraceID(c),
	))
}

func localize(c HTTPContext, fullKey string, args map[string]any) string {
	namespace, key := splitMessageKey(fullKey)
	locale := localeFromContext(c)
	if value, ok := c.Get(i18nContextKey); ok {
		if localizer, ok := value.(Localizer); ok {
			return localizer.Localize(locale, namespace, key, args)
		}
	}
	return fullKey
}

func localeFromContext(c HTTPContext) string {
	if value, ok := c.Get(localeContextKey); ok {
		if locale, ok := value.(string); ok && strings.TrimSpace(locale) != "" {
			return locale
		}
	}
	if value, ok := c.Get(i18nContextKey); ok {
		if localizer, ok := value.(Localizer); ok {
			return localizer.DefaultLocale()
		}
	}
	return defaultLocale
}

func splitMessageKey(fullKey string) (string, string) {
	fullKey = strings.TrimSpace(fullKey)
	if fullKey == "" {
		return "api", "common.internalError"
	}
	parts := strings.SplitN(fullKey, ".", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "api", fullKey
	}
	return parts[0], parts[1]
}

func messageKeyOrDefault(messageKey string, fallback string) string {
	messageKey = strings.TrimSpace(messageKey)
	if messageKey == "" || !strings.Contains(messageKey, ".") {
		return fallback
	}
	return messageKey
}

func firstArgs(args []map[string]any) map[string]any {
	if len(args) == 0 {
		return nil
	}
	return args[0]
}

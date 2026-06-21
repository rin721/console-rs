package result

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	apperrors "github.com/rei0721/go-scaffold/types/errors"
)

type responseBody struct {
	Code       int             `json:"code"`
	MessageKey string          `json:"messageKey"`
	Message    string          `json:"message"`
	Data       json.RawMessage `json:"data,omitempty"`
	TraceID    string          `json:"traceId,omitempty"`
	ServerTime int64           `json:"serverTime"`
}

func TestSuccessErrorAndPaginationContracts(t *testing.T) {
	success := Success("ok")
	if success.Code != 0 {
		t.Fatalf("Success code = %d, want 0", success.Code)
	}
	if success.MessageKey != MessageKeySuccess || success.Message != MessageKeySuccess {
		t.Fatalf("Success message = (%q, %q), want key fallback", success.MessageKey, success.Message)
	}
	if success.Data != "ok" {
		t.Fatalf("Success data = %q, want ok", success.Data)
	}
	if success.ServerTime == 0 {
		t.Fatal("Success server time should be set")
	}

	errResult := Error(apperrors.ErrInvalidParams, MessageKeyInvalidRequest)
	if errResult.Code != apperrors.ErrInvalidParams {
		t.Fatalf("Error code = %d, want %d", errResult.Code, apperrors.ErrInvalidParams)
	}
	if errResult.MessageKey != MessageKeyInvalidRequest {
		t.Fatalf("Error messageKey = %q, want %q", errResult.MessageKey, MessageKeyInvalidRequest)
	}
	if errResult.TraceID != "" {
		t.Fatalf("Error trace id = %q, want empty", errResult.TraceID)
	}

	traceResult := ErrorWithTrace(apperrors.ErrInternalServer, MessageKeyInternalError, "trace-1")
	if traceResult.TraceID != "trace-1" {
		t.Fatalf("ErrorWithTrace trace id = %q, want trace-1", traceResult.TraceID)
	}

	page := NewPageResult([]string{"a", "b"}, 2, 10, 21)
	if page.Pagination.TotalPages != 3 {
		t.Fatalf("TotalPages = %d, want 3", page.Pagination.TotalPages)
	}
	if page.Pagination.Page != 2 || page.Pagination.PageSize != 10 || page.Pagination.Total != 21 {
		t.Fatalf("unexpected pagination metadata: %+v", page.Pagination)
	}
}

func TestGinResponseHelpersContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		call       func(*gin.Context)
		wantStatus int
		wantCode   int
		wantKey    string
	}{
		{
			name:       "bad request",
			call:       func(c *gin.Context) { BadRequest(c, MessageKeyInvalidRequest) },
			wantStatus: http.StatusBadRequest,
			wantCode:   apperrors.ErrInvalidParams,
			wantKey:    MessageKeyInvalidRequest,
		},
		{
			name:       "unauthorized",
			call:       func(c *gin.Context) { Unauthorized(c, MessageKeyUnauthorized) },
			wantStatus: http.StatusUnauthorized,
			wantCode:   apperrors.ErrUnauthorized,
			wantKey:    MessageKeyUnauthorized,
		},
		{
			name:       "forbidden",
			call:       func(c *gin.Context) { Forbidden(c, MessageKeyForbidden) },
			wantStatus: http.StatusForbidden,
			wantCode:   apperrors.ErrPermissionDenied,
			wantKey:    MessageKeyForbidden,
		},
		{
			name:       "not found",
			call:       func(c *gin.Context) { NotFound(c, MessageKeyNotFound) },
			wantStatus: http.StatusNotFound,
			wantCode:   apperrors.ErrResourceNotFound,
			wantKey:    MessageKeyNotFound,
		},
		{
			name:       "internal error",
			call:       func(c *gin.Context) { InternalError(c, MessageKeyInternalError) },
			wantStatus: http.StatusInternalServerError,
			wantCode:   apperrors.ErrInternalServer,
			wantKey:    MessageKeyInternalError,
		},
		{
			name:       "generic fail",
			call:       func(c *gin.Context) { Fail(c, http.StatusBadRequest, MessageKeyInvalidRequest) },
			wantStatus: http.StatusBadRequest,
			wantCode:   apperrors.ErrInvalidParams,
			wantKey:    MessageKeyInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Set("traceId", "trace-1")

			tt.call(ctx)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("HTTP status = %d, want %d", recorder.Code, tt.wantStatus)
			}

			var body responseBody
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Code != tt.wantCode {
				t.Fatalf("body code = %d, want %d", body.Code, tt.wantCode)
			}
			if body.MessageKey != tt.wantKey {
				t.Fatalf("messageKey = %q, want %q", body.MessageKey, tt.wantKey)
			}
			if body.TraceID != "trace-1" {
				t.Fatalf("trace id = %q, want trace-1", body.TraceID)
			}
			if body.ServerTime == 0 {
				t.Fatal("server time should be set")
			}
		})
	}
}

func TestGetTraceIDContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	if got := GetTraceID(ctx); got != "" {
		t.Fatalf("GetTraceID without value = %q, want empty", got)
	}

	ctx.Set("traceId", 123)
	if got := GetTraceID(ctx); got != "" {
		t.Fatalf("GetTraceID with non-string value = %q, want empty", got)
	}

	ctx.Set("traceId", "trace-1")
	if got := GetTraceID(ctx); got != "trace-1" {
		t.Fatalf("GetTraceID = %q, want trace-1", got)
	}

	legacyCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	legacyCtx.Set("trace_id", "legacy-trace")
	if got := GetTraceID(legacyCtx); got != "legacy-trace" {
		t.Fatalf("GetTraceID legacy key = %q, want legacy-trace", got)
	}
}

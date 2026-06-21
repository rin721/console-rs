package errors

import (
	stderrors "errors"
	"strings"
	"testing"
)

func TestBizErrorContract(t *testing.T) {
	cause := stderrors.New("database offline")
	args := map[string]any{"resource": "database"}
	err := NewBizError(ErrDatabaseError, "api.database.failed", args).WithCause(cause)

	if err.Code != ErrDatabaseError {
		t.Fatalf("Code = %d, want %d", err.Code, ErrDatabaseError)
	}
	if err.MessageKey != "api.database.failed" {
		t.Fatalf("MessageKey = %q, want api.database.failed", err.MessageKey)
	}
	if err.Args["resource"] != "database" {
		t.Fatalf("Args = %#v, want resource database", err.Args)
	}
	if !stderrors.Is(err, cause) {
		t.Fatal("BizError should unwrap the cause error")
	}
	if got := err.Error(); !strings.Contains(got, "[5001] api.database.failed: database offline") {
		t.Fatalf("Error() = %q, want code, message key, and cause", got)
	}
}

func TestErrorCodeRangesContract(t *testing.T) {
	tests := []struct {
		name string
		code int
		min  int
		max  int
	}{
		{"invalid params", ErrInvalidParams, 1000, 1999},
		{"business logic", ErrBusinessLogic, 2000, 2999},
		{"unauthorized", ErrUnauthorized, 3000, 3999},
		{"invalid token", ErrInvalidToken, 3000, 3999},
		{"token expired", ErrTokenExpired, 3000, 3999},
		{"permission denied", ErrPermissionDenied, 3000, 3999},
		{"resource not found", ErrResourceNotFound, 4000, 4999},
		{"internal server", ErrInternalServer, 5000, 5999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code < tt.min || tt.code > tt.max {
				t.Fatalf("code %d outside range %d-%d", tt.code, tt.min, tt.max)
			}
		})
	}
}

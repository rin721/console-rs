package middleware

import (
	"context"
	"net/http"
	"testing"

	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
	"github.com/rei0721/go-scaffold/internal/ports"
	"github.com/rei0721/go-scaffold/types/result"
)

func TestRequireOrgParam(t *testing.T) {
	tests := []struct {
		name       string
		principal  iamservice.Principal
		param      string
		wantCalled bool
		wantStatus int
	}{
		{
			name:       "same organization",
			principal:  iamservice.Principal{OrgID: 42},
			param:      "42",
			wantCalled: true,
		},
		{
			name:       "cross organization",
			principal:  iamservice.Principal{OrgID: 42},
			param:      "43",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "invalid organization",
			principal:  iamservice.Principal{OrgID: 42},
			param:      "bad",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newAuthTestContext()
			ctx.values[PrincipalKey] = tt.principal
			ctx.params["orgId"] = tt.param

			called := false
			RequireOrgParam("orgId", func(c ports.HTTPContext) {
				called = true
			})(ctx)

			if called != tt.wantCalled {
				t.Fatalf("next called = %v, want %v", called, tt.wantCalled)
			}
			if tt.wantStatus != 0 && ctx.abortStatus != tt.wantStatus {
				t.Fatalf("abort status = %d, want %d", ctx.abortStatus, tt.wantStatus)
			}
			if tt.wantStatus == 0 && ctx.abortStatus != 0 {
				t.Fatalf("unexpected abort status %d", ctx.abortStatus)
			}
		})
	}
}

func TestRequirePermissionWithoutAuthorizer(t *testing.T) {
	ctx := newAuthTestContext()
	ctx.values[PrincipalKey] = iamservice.Principal{OrgID: 42}

	RequirePermission(nil, iamservice.PermissionContext{
		ProductCode: "core",
		Scope:       "tenant",
		Object:      "user",
		Action:      "read",
	}, func(c ports.HTTPContext) {
		t.Fatal("next should not be called")
	})(ctx)

	if ctx.abortStatus != http.StatusForbidden {
		t.Fatalf("abort status = %d, want %d", ctx.abortStatus, http.StatusForbidden)
	}
	body, ok := ctx.abortBody.(*result.Result[any])
	if !ok {
		t.Fatalf("abort body type = %T, want *result.Result[any]", ctx.abortBody)
	}
	if body.TraceID != "trace-test" {
		t.Fatalf("trace id = %q, want trace-test", body.TraceID)
	}
}

type authTestContext struct {
	values      map[string]any
	params      map[string]string
	headers     map[string]string
	abortStatus int
	abortBody   any
}

func newAuthTestContext() *authTestContext {
	return &authTestContext{
		values:  map[string]any{TraceIDKey: "trace-test"},
		params:  map[string]string{},
		headers: map[string]string{},
	}
}

func (c *authTestContext) Request() *http.Request {
	return (&http.Request{}).WithContext(context.Background())
}

func (c *authTestContext) RequestContext() context.Context { return context.Background() }
func (c *authTestContext) GetHeader(name string) string    { return c.headers[name] }
func (c *authTestContext) Header(name, value string)       { c.headers[name] = value }
func (c *authTestContext) Cookie(name string) (string, error) {
	value, ok := c.headers["cookie:"+name]
	if !ok {
		return "", http.ErrNoCookie
	}
	return value, nil
}
func (c *authTestContext) SetCookie(cookie *http.Cookie) {
	if cookie != nil {
		c.headers["cookie:"+cookie.Name] = cookie.Value
	}
}
func (c *authTestContext) Set(key string, value any) { c.values[key] = value }
func (c *authTestContext) Get(key any) (any, bool) {
	value, ok := c.values[key.(string)]
	return value, ok
}
func (c *authTestContext) Param(name string) string  { return c.params[name] }
func (c *authTestContext) BindJSON(dest any) error   { return nil }
func (c *authTestContext) JSON(status int, body any) {}
func (c *authTestContext) Data(int, string, []byte)  {}
func (c *authTestContext) AbortWithStatusJSON(status int, body any) {
	c.abortStatus = status
	c.abortBody = body
}
func (c *authTestContext) Next()            {}
func (c *authTestContext) Path() string     { return "" }
func (c *authTestContext) Method() string   { return "" }
func (c *authTestContext) ClientIP() string { return "" }
func (c *authTestContext) Status() int      { return c.abortStatus }

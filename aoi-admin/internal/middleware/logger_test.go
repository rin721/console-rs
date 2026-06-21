package middleware

import (
	"context"
	"net/http"
	"testing"
)

func TestRealIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		clientIP string
		remote   string
		want     string
	}{
		{
			name: "forwarded header",
			headers: map[string]string{
				"Forwarded":       `for=203.0.113.10;proto=https`,
				"X-Forwarded-For": "198.51.100.10",
			},
			clientIP: "10.0.0.1",
			remote:   "10.0.0.2:1234",
			want:     "203.0.113.10",
		},
		{
			name: "x forwarded for first address",
			headers: map[string]string{
				"X-Forwarded-For": "198.51.100.10, 10.0.0.1",
			},
			clientIP: "10.0.0.1",
			remote:   "10.0.0.2:1234",
			want:     "198.51.100.10",
		},
		{
			name: "x forwarded for skips unknown address",
			headers: map[string]string{
				"X-Forwarded-For": "unknown, 198.51.100.11, 10.0.0.1",
			},
			clientIP: "10.0.0.1",
			remote:   "10.0.0.2:1234",
			want:     "198.51.100.11",
		},
		{
			name: "x real ip fallback",
			headers: map[string]string{
				"X-Forwarded-For": "unknown",
				"X-Real-IP":       "198.51.100.20",
			},
			clientIP: "10.0.0.1",
			remote:   "10.0.0.2:1234",
			want:     "198.51.100.20",
		},
		{
			name:     "request remote address fallback",
			clientIP: "",
			remote:   "192.0.2.30:4321",
			want:     "192.0.2.30",
		},
		{
			name: "forwarded ipv6 with port",
			headers: map[string]string{
				"Forwarded": `for="[2001:db8::1]:443";proto=https`,
			},
			clientIP: "10.0.0.1",
			remote:   "10.0.0.2:1234",
			want:     "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &loggerTestContext{
				request:  &http.Request{RemoteAddr: tt.remote},
				headers:  tt.headers,
				clientIP: tt.clientIP,
			}

			if got := ClientIPRealIP(ctx); got != tt.want {
				t.Fatalf("realIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoggerRecordsRealIP(t *testing.T) {
	log := &capturingLogger{}
	ctx := &loggerTestContext{
		request: &http.Request{RemoteAddr: "10.0.0.2:1234"},
		headers: map[string]string{
			"X-Forwarded-For": "198.51.100.10, 10.0.0.1",
		},
		values:   map[string]any{TraceIDKey: "trace-1"},
		path:     "/api/v1/system/menus",
		method:   http.MethodGet,
		clientIP: "10.0.0.1",
		status:   http.StatusOK,
	}

	Logger(LoggerConfig{Enabled: true}, log)(ctx)

	if log.msg != "request completed" {
		t.Fatalf("log message = %q, want request completed", log.msg)
	}
	if got := log.fields["clientIP"]; got != "10.0.0.1" {
		t.Fatalf("clientIP field = %v, want 10.0.0.1", got)
	}
	if got := log.fields["realIP"]; got != "198.51.100.10" {
		t.Fatalf("realIP field = %v, want 198.51.100.10", got)
	}
}

type loggerTestContext struct {
	request  *http.Request
	values   map[string]any
	headers  map[string]string
	path     string
	method   string
	clientIP string
	status   int
}

func (c *loggerTestContext) Request() *http.Request {
	if c.request == nil {
		return (&http.Request{}).WithContext(context.Background())
	}
	if c.request.Context() == nil {
		return c.request.WithContext(context.Background())
	}
	return c.request
}

func (c *loggerTestContext) RequestContext() context.Context { return c.Request().Context() }
func (c *loggerTestContext) GetHeader(name string) string    { return c.headers[name] }
func (c *loggerTestContext) Header(name, value string)       { c.headers[name] = value }
func (c *loggerTestContext) Cookie(name string) (string, error) {
	if c.headers == nil {
		return "", http.ErrNoCookie
	}
	value, ok := c.headers["cookie:"+name]
	if !ok {
		return "", http.ErrNoCookie
	}
	return value, nil
}
func (c *loggerTestContext) SetCookie(cookie *http.Cookie) {
	if cookie != nil {
		if c.headers == nil {
			c.headers = map[string]string{}
		}
		c.headers["cookie:"+cookie.Name] = cookie.Value
	}
}
func (c *loggerTestContext) Set(key string, value any) {
	if c.values == nil {
		c.values = map[string]any{}
	}
	c.values[key] = value
}
func (c *loggerTestContext) Get(key any) (any, bool) {
	if c.values == nil {
		return nil, false
	}
	value, ok := c.values[key.(string)]
	return value, ok
}
func (c *loggerTestContext) Param(string) string          { return "" }
func (c *loggerTestContext) BindJSON(any) error           { return nil }
func (c *loggerTestContext) JSON(int, any)                {}
func (c *loggerTestContext) Data(int, string, []byte)     {}
func (c *loggerTestContext) AbortWithStatusJSON(int, any) {}
func (c *loggerTestContext) Next()                        {}
func (c *loggerTestContext) Path() string                 { return c.path }
func (c *loggerTestContext) Method() string               { return c.method }
func (c *loggerTestContext) ClientIP() string             { return c.clientIP }
func (c *loggerTestContext) Status() int                  { return c.status }

type capturingLogger struct {
	msg    string
	fields map[string]any
}

func (l *capturingLogger) Debug(string, ...interface{}) {}
func (l *capturingLogger) Info(msg string, keysAndValues ...interface{}) {
	l.msg = msg
	l.fields = map[string]any{}
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		l.fields[key] = keysAndValues[i+1]
	}
}
func (l *capturingLogger) Warn(string, ...interface{})  {}
func (l *capturingLogger) Error(string, ...interface{}) {}
func (l *capturingLogger) Fatal(string, ...interface{}) {}
func (l *capturingLogger) Sync() error                  { return nil }

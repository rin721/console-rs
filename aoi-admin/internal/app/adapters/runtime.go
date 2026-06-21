package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
	systemservice "github.com/rei0721/go-scaffold/internal/modules/system/service"
	"github.com/rei0721/go-scaffold/internal/ports"
	"github.com/rei0721/go-scaffold/pkg/authorization"
	"github.com/rei0721/go-scaffold/pkg/hostmetrics"
	"github.com/rei0721/go-scaffold/pkg/mfa"
	"github.com/rei0721/go-scaffold/pkg/rpcserver"
	"github.com/rei0721/go-scaffold/pkg/token"
	"github.com/rei0721/go-scaffold/pkg/web"
)

// HTTPRouter 将 pkg/web 路由适配为 internal/ports.HTTPRouter。
//
// 该适配器让 transport 层只依赖端口接口，不直接绑定具体 Web 框架实现。
type HTTPRouter struct {
	inner web.Router
}

// NewHTTPRouter 创建端口化 HTTP 路由适配器；nil 输入表示该能力未启用。
func NewHTTPRouter(router web.Router) ports.HTTPRouter {
	if router == nil {
		return nil
	}
	return HTTPRouter{inner: router}
}

// NewHTTPEngine 创建带路由枚举和 SPA 挂载能力的 HTTP engine 适配器。
func NewHTTPEngine(engine *web.Engine) *HTTPEngine {
	if engine == nil {
		return nil
	}
	return &HTTPEngine{Engine: engine}
}

// HTTPEngine 暴露应用路由装配所需的完整 HTTP engine 能力。
type HTTPEngine struct {
	Engine *web.Engine
}

func (e *HTTPEngine) Use(handlers ...ports.HTTPHandlerFunc) {
	e.Engine.Use(wrapHTTPHandlers(handlers)...)
}

func (e *HTTPEngine) GET(path string, handler ports.HTTPHandlerFunc) {
	e.Engine.GET(path, wrapHTTPHandler(handler))
}

func (e *HTTPEngine) POST(path string, handler ports.HTTPHandlerFunc) {
	e.Engine.POST(path, wrapHTTPHandler(handler))
}

func (e *HTTPEngine) PATCH(path string, handler ports.HTTPHandlerFunc) {
	e.Engine.PATCH(path, wrapHTTPHandler(handler))
}

func (e *HTTPEngine) PUT(path string, handler ports.HTTPHandlerFunc) {
	e.Engine.PUT(path, wrapHTTPHandler(handler))
}

func (e *HTTPEngine) DELETE(path string, handler ports.HTTPHandlerFunc) {
	e.Engine.DELETE(path, wrapHTTPHandler(handler))
}

func (e *HTTPEngine) ANY(path string, handler ports.HTTPHandlerFunc) {
	e.Engine.ANY(path, wrapHTTPHandler(handler))
}

func (e *HTTPEngine) Group(path string) ports.HTTPRouter {
	return NewHTTPRouter(e.Engine.Group(path))
}

func (e *HTTPEngine) Routes() []ports.RouteInfo {
	return routeInfos(e.Engine.Routes())
}

func (e *HTTPEngine) MountStaticSPA(cfg ports.StaticSPAConfig) error {
	err := e.Engine.MountStaticSPA(web.StaticSPAConfig{
		MountPath:            cfg.MountPath,
		DistDir:              cfg.DistDir,
		ExcludedPathPrefixes: cfg.ExcludedPathPrefixes,
	})
	if errors.Is(err, web.ErrStaticSPAIndexMissing) {
		return ports.ErrStaticSPAIndexMissing
	}
	return err
}

func (r HTTPRouter) Use(handlers ...ports.HTTPHandlerFunc) {
	r.inner.Use(wrapHTTPHandlers(handlers)...)
}

func (r HTTPRouter) GET(path string, handler ports.HTTPHandlerFunc) {
	r.inner.GET(path, wrapHTTPHandler(handler))
}

func (r HTTPRouter) POST(path string, handler ports.HTTPHandlerFunc) {
	r.inner.POST(path, wrapHTTPHandler(handler))
}

func (r HTTPRouter) PATCH(path string, handler ports.HTTPHandlerFunc) {
	r.inner.PATCH(path, wrapHTTPHandler(handler))
}

func (r HTTPRouter) PUT(path string, handler ports.HTTPHandlerFunc) {
	r.inner.PUT(path, wrapHTTPHandler(handler))
}

func (r HTTPRouter) DELETE(path string, handler ports.HTTPHandlerFunc) {
	r.inner.DELETE(path, wrapHTTPHandler(handler))
}

func (r HTTPRouter) ANY(path string, handler ports.HTTPHandlerFunc) {
	r.inner.ANY(path, wrapHTTPHandler(handler))
}

func (r HTTPRouter) Group(path string) ports.HTTPRouter {
	return NewHTTPRouter(r.inner.Group(path))
}

// CORS 将端口层 CORS 配置转换为 pkg/web 中间件。
func CORS(cfg ports.CORSConfig) ports.HTTPHandlerFunc {
	return func(c ports.HTTPContext) {
		web.CORS(web.CORSConfig{
			Enabled:          cfg.Enabled,
			AllowOrigins:     cfg.AllowOrigins,
			AllowMethods:     cfg.AllowMethods,
			AllowHeaders:     cfg.AllowHeaders,
			ExposeHeaders:    cfg.ExposeHeaders,
			AllowCredentials: cfg.AllowCredentials,
			MaxAge:           cfg.MaxAge,
		})(unwrapHTTPContext(c))
	}
}

// Recovery 将 pkg/web 的 panic 恢复中间件暴露为端口层处理器。
func Recovery() ports.HTTPHandlerFunc {
	return func(c ports.HTTPContext) {
		web.Recovery()(unwrapHTTPContext(c))
	}
}

// wrapHTTPHandlers 批量把端口层 handler 转换为 pkg/web handler。
func wrapHTTPHandlers(handlers []ports.HTTPHandlerFunc) []web.HandlerFunc {
	wrapped := make([]web.HandlerFunc, 0, len(handlers))
	for _, handler := range handlers {
		wrapped = append(wrapped, wrapHTTPHandler(handler))
	}
	return wrapped
}

// wrapHTTPHandler 保持请求上下文原样传递，避免适配层改变 handler 执行语义。
func wrapHTTPHandler(handler ports.HTTPHandlerFunc) web.HandlerFunc {
	return func(c web.Context) {
		handler(c)
	}
}

// unwrapHTTPContext 在可能时恢复底层 web.Context。
//
// 端口层 context 不是 pkg/web 实现时退回到 noop 包装，保证中间件适配不会 panic。
func unwrapHTTPContext(c ports.HTTPContext) web.Context {
	ctx, ok := c.(web.Context)
	if !ok {
		return noopHTTPContext{HTTPContext: c}
	}
	return ctx
}

// noopHTTPContext 为非 pkg/web 的测试或替代 context 提供最小 web.Context 包装。
type noopHTTPContext struct {
	ports.HTTPContext
}

// routeInfos 将底层路由表转换为系统模块使用的端口类型。
func routeInfos(routes []web.RouteInfo) []ports.RouteInfo {
	out := make([]ports.RouteInfo, 0, len(routes))
	for _, route := range routes {
		out = append(out, ports.RouteInfo{
			Method:  route.Method,
			Path:    route.Path,
			Handler: route.Handler,
		})
	}
	return out
}

// TokenManager 将 pkg/token.Manager 适配为 IAM service 所需接口。
type TokenManager struct {
	inner token.Manager
}

// NewTokenManager 创建 IAM service 的 token 管理适配器。
func NewTokenManager(manager token.Manager) iamservice.TokenManager {
	if manager == nil {
		return nil
	}
	return TokenManager{inner: manager}
}

func (m TokenManager) IssueAccess(ctx context.Context, subject iamservice.TokenSubject) (string, time.Time, error) {
	return m.inner.IssueAccess(ctx, tokenSubject(subject))
}

func (m TokenManager) IssueRefresh(ctx context.Context) (string, string, time.Time, error) {
	return m.inner.IssueRefresh(ctx)
}

func (m TokenManager) IssuePair(ctx context.Context, subject iamservice.TokenSubject) (iamservice.IssuedTokenPair, error) {
	pair, err := m.inner.IssuePair(ctx, tokenSubject(subject))
	return iamservice.IssuedTokenPair{
		AccessToken:      pair.AccessToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshToken:     pair.RefreshToken,
		RefreshTokenHash: pair.RefreshTokenHash,
		RefreshExpiresAt: pair.RefreshExpiresAt,
	}, err
}

// Parse 将 pkg/token 的 claims 转换为 IAM service 的 claims 类型，隔离跨包模型。
func (m TokenManager) Parse(ctx context.Context, raw string, expectedType string) (*iamservice.TokenClaims, error) {
	claims, err := m.inner.Parse(ctx, raw, expectedType)
	if err != nil {
		return nil, err
	}
	return &iamservice.TokenClaims{
		UserID:      claims.UserID,
		OrgID:       claims.OrgID,
		SessionID:   claims.SessionID,
		ProductCode: claims.ProductCode,
		TokenType:   claims.TokenType,
	}, nil
}

func (m TokenManager) HashRefreshToken(raw string) string {
	return m.inner.HashRefreshToken(raw)
}

// tokenSubject 将 IAM service 的 token subject 转换为基础设施 token 包的 subject。
func tokenSubject(subject iamservice.TokenSubject) token.Subject {
	return token.Subject{
		UserID:      subject.UserID,
		OrgID:       subject.OrgID,
		SessionID:   subject.SessionID,
		ProductCode: subject.ProductCode,
	}
}

// AuthorizerEnforcer 将授权引擎适配为 IAM service 所需的最小授权端口。
type AuthorizerEnforcer struct {
	inner authorization.Enforcer
}

// NewAuthorizerEnforcer 创建 IAM service 的授权执行器适配器。
func NewAuthorizerEnforcer(enforcer authorization.Enforcer) iamservice.AuthorizerEnforcer {
	if enforcer == nil {
		return nil
	}
	return AuthorizerEnforcer{inner: enforcer}
}

func (e AuthorizerEnforcer) Enforce(ctx context.Context, sub, org, product, scope, obj, act string) (bool, error) {
	return e.inner.Enforce(ctx, sub, org, product, scope, obj, act)
}

func (e AuthorizerEnforcer) AddPolicy(ctx context.Context, role, org, product, scope, obj, act string) (bool, error) {
	return e.inner.AddPolicy(ctx, role, org, product, scope, obj, act)
}

func (e AuthorizerEnforcer) AddRoleForUser(ctx context.Context, user, role, org string) (bool, error) {
	return e.inner.AddRoleForUser(ctx, user, role, org)
}

func (e AuthorizerEnforcer) DeleteRoleForUser(ctx context.Context, user, role, org string) (bool, error) {
	return e.inner.DeleteRoleForUser(ctx, user, role, org)
}

func (e AuthorizerEnforcer) GetRolesForUser(ctx context.Context, user, org string) ([]string, error) {
	return e.inner.GetRolesForUser(ctx, user, org)
}

// LoadRules 将 IAM service 规则批量转换为 authorization 包的规则模型。
func (e AuthorizerEnforcer) LoadRules(ctx context.Context, rules []iamservice.AuthorizationRule) error {
	out := make([]authorization.Rule, 0, len(rules))
	for _, rule := range rules {
		out = append(out, authorization.Rule{PType: rule.PType, Values: rule.Values})
	}
	return e.inner.LoadRules(ctx, out)
}

// TOTPProvider 将 pkg/mfa 的 TOTP 能力适配到 IAM service。
type TOTPProvider struct{}

// GenerateTOTP 生成 TOTP 密钥并转换为 IAM service 的返回模型。
func (TOTPProvider) GenerateTOTP(issuer, accountName string) (iamservice.TOTPKey, error) {
	key, err := mfa.GenerateTOTP(issuer, accountName)
	return iamservice.TOTPKey{Secret: key.Secret, URL: key.URL}, err
}

// ValidateTOTP 校验一次性验证码。
func (TOTPProvider) ValidateTOTP(code, secret string) bool {
	return mfa.ValidateTOTP(code, secret)
}

// HostMetricsCollector 将主机指标采集结果转换为系统模块模型。
type HostMetricsCollector struct{}

// Collect 收集主机指标，并复制切片字段以避免跨包共享底层数组。
func (HostMetricsCollector) Collect(ctx context.Context) systemservice.HostMetrics {
	metrics := hostmetrics.Collect(ctx)
	disks := make([]systemservice.DiskInfo, 0, len(metrics.Disk))
	for _, disk := range metrics.Disk {
		disks = append(disks, systemservice.DiskInfo{
			FSType:      disk.FSType,
			MountPoint:  disk.MountPoint,
			TotalGB:     disk.TotalGB,
			TotalMB:     disk.TotalMB,
			UsedGB:      disk.UsedGB,
			UsedMB:      disk.UsedMB,
			UsedPercent: disk.UsedPercent,
		})
	}
	diskIO := make([]systemservice.DiskIOInfo, 0, len(metrics.DiskIO))
	for _, counter := range metrics.DiskIO {
		diskIO = append(diskIO, systemservice.DiskIOInfo{
			Name:        counter.Name,
			ReadBytes:   counter.ReadBytes,
			WriteBytes:  counter.WriteBytes,
			ReadCount:   counter.ReadCount,
			WriteCount:  counter.WriteCount,
			ReadTimeMs:  counter.ReadTimeMs,
			WriteTimeMs: counter.WriteTimeMs,
		})
	}
	return systemservice.HostMetrics{
		CPU: systemservice.CPUInfo{
			Cores:   metrics.CPU.Cores,
			Percent: append([]float64(nil), metrics.CPU.Percent...),
		},
		RAM: systemservice.RAMInfo{
			TotalMB:     metrics.RAM.TotalMB,
			UsedMB:      metrics.RAM.UsedMB,
			UsedPercent: metrics.RAM.UsedPercent,
		},
		Disk:   disks,
		DiskIO: diskIO,
		Network: systemservice.NetworkInfo{
			ReceiveBytes:  metrics.Network.ReceiveBytes,
			TransmitBytes: metrics.Network.TransmitBytes,
		},
	}
}

// RPCRegistry 将 pkg/rpcserver.Registry 适配为端口层 RPC 注册表。
type RPCRegistry struct {
	inner *rpcserver.Registry
}

// NewRPCRegistry 创建端口层 RPC 注册表适配器。
func NewRPCRegistry(registry *rpcserver.Registry) ports.RPCRegistry {
	if registry == nil {
		return nil
	}
	return RPCRegistry{inner: registry}
}

// Register 注册 RPC 方法，并把端口层 RPCError 转换为底层 rpcserver 可识别的错误。
func (r RPCRegistry) Register(method string, handler ports.RPCHandlerFunc) error {
	return r.inner.Register(method, func(ctx context.Context, params json.RawMessage) (any, error) {
		result, err := handler(ctx, params)
		if err == nil {
			return result, nil
		}
		var rpcErr *ports.RPCError
		if errors.As(err, &rpcErr) {
			return nil, &rpcserver.RPCError{Code: rpcErr.Code, Message: rpcErr.Message, Data: rpcErr.Data}
		}
		return nil, err
	})
}

func (r RPCRegistry) Methods() []string {
	return r.inner.Methods()
}

// UnwrapRPCRegistry 在需要底层实现时解包 RPC 注册表适配器。
func UnwrapRPCRegistry(registry ports.RPCRegistry) *rpcserver.Registry {
	if wrapped, ok := registry.(RPCRegistry); ok {
		return wrapped.inner
	}
	if wrapped, ok := registry.(*RPCRegistry); ok && wrapped != nil {
		return wrapped.inner
	}
	return nil
}

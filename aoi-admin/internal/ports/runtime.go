package ports

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"
)

// Logger 是运行时层依赖的最小日志端口，避免内部模块直接绑定具体日志实现。
type Logger interface {
	Debug(string, ...interface{})
	Info(string, ...interface{})
	Warn(string, ...interface{})
	Error(string, ...interface{})
	Fatal(string, ...interface{})
	Sync() error
}

// IDGenerator 抽象全局 ID 生成能力，调用方按需选择数值或字符串形式。
type IDGenerator interface {
	NextID() int64
	NextIDString() string
}

// I18n 抽象国际化翻译能力，供中间件和业务层统一读取语言资源。
type I18n interface {
	Localize(string, string, string, map[string]any) string
	ResolveLocale(...string) string
	IsSupported(string) bool
	DefaultLocale() string
	FallbackLocale() string
	ValidateResources() error
}

// HTTPContext 是 HTTP 处理链可使用的最小请求上下文端口。
type HTTPContext interface {
	Request() *http.Request
	RequestContext() context.Context
	GetHeader(string) string
	Header(string, string)
	Cookie(string) (string, error)
	SetCookie(*http.Cookie)
	Set(string, any)
	Get(any) (any, bool)
	Param(string) string
	BindJSON(any) error
	JSON(int, any)
	Data(int, string, []byte)
	AbortWithStatusJSON(int, any)
	Next()
	Path() string
	Method() string
	ClientIP() string
	Status() int
}

// HTTPHandlerFunc 表示与具体 Web 框架解耦的 HTTP handler。
type HTTPHandlerFunc func(HTTPContext)

// HTTPRouter 是路由注册端口，供 transport 层组装路由而不暴露 Gin 等实现。
type HTTPRouter interface {
	Use(...HTTPHandlerFunc)
	GET(string, HTTPHandlerFunc)
	POST(string, HTTPHandlerFunc)
	PATCH(string, HTTPHandlerFunc)
	PUT(string, HTTPHandlerFunc)
	DELETE(string, HTTPHandlerFunc)
	ANY(string, HTTPHandlerFunc)
	Group(string) HTTPRouter
}

// RouteInfo 描述已注册路由的只读元数据，主要用于启动日志和诊断。
type RouteInfo struct {
	Method  string
	Path    string
	Handler string
}

// StaticSPAConfig 描述静态单页应用的挂载路径和构建产物目录。
type StaticSPAConfig struct {
	MountPath            string
	DistDir              string
	ExcludedPathPrefixes []string
}

// ErrStaticSPAIndexMissing 表示静态 SPA 目录缺少可回退的入口文件。
var ErrStaticSPAIndexMissing = errors.New("static spa index.html missing")

// StaticSPAMounter 抽象静态 SPA 挂载能力，供 WebUI 模块按配置接入。
type StaticSPAMounter interface {
	MountStaticSPA(StaticSPAConfig) error
}

// CORSConfig 是 HTTP CORS 中间件的跨层配置快照。
type CORSConfig struct {
	Enabled          bool
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// CORSFactory 根据配置创建 HTTP CORS 中间件。
type CORSFactory func(CORSConfig) HTTPHandlerFunc

// MediaObjectStorage 抽象媒体服务需要的文件存储能力，避免系统模块依赖完整 storage 实现。
type MediaObjectStorage interface {
	ReadFile(string) ([]byte, error)
	WriteFile(string, []byte, os.FileMode) error
	Remove(string) error
	RemoveAll(string) error
	MkdirAll(string, os.FileMode) error
	DetectMIMEFromBytes([]byte) (string, error)
}

// HostMetricsCollector 抽象主机资源采集能力。
type HostMetricsCollector interface {
	Collect(context.Context) HostMetrics
}

// HostMetrics 是一次主机资源采集的聚合结果。
type HostMetrics struct {
	CPU  CPUInfo
	RAM  RAMInfo
	Disk []DiskInfo
}

// CPUInfo 描述 CPU 核心数和每核使用率。
type CPUInfo struct {
	Cores   int
	Percent []float64
}

// RAMInfo 描述内存总量、已用量和使用率。
type RAMInfo struct {
	TotalMB     uint64
	UsedMB      uint64
	UsedPercent float64
}

// DiskInfo 描述一个挂载点的磁盘容量和使用率。
type DiskInfo struct {
	FSType      string
	MountPoint  string
	TotalGB     uint64
	TotalMB     uint64
	UsedGB      uint64
	UsedMB      uint64
	UsedPercent float64
}

// PasswordCrypto 抽象密码哈希与校验能力。
type PasswordCrypto interface {
	HashPassword(string) (string, error)
	VerifyPassword(string, string) error
}

// TokenSubject 是签发访问令牌所需的身份上下文。
type TokenSubject struct {
	UserID    int64
	OrgID     int64
	SessionID int64
}

const (
	// TokenTypeAccess 表示访问令牌。
	TokenTypeAccess = "access"
	// TokenTypeRefresh 表示刷新令牌。
	TokenTypeRefresh = "refresh"
)

// TokenClaims 是业务层从已校验 token 中读取的身份声明。
type TokenClaims struct {
	UserID    int64
	OrgID     int64
	SessionID int64
	TokenType string
}

// TokenPair 表示访问令牌和刷新令牌的签发结果。
type TokenPair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshTokenHash string
	RefreshExpiresAt time.Time
}

// TokenManager 抽象 token 签发、解析和 refresh token 哈希能力。
type TokenManager interface {
	IssueAccess(context.Context, TokenSubject) (string, time.Time, error)
	IssueRefresh(context.Context) (string, string, time.Time, error)
	IssuePair(context.Context, TokenSubject) (TokenPair, error)
	Parse(context.Context, string, string) (*TokenClaims, error)
	HashRefreshToken(string) string
}

// AuthorizationRule 表示可加载到授权引擎的一条策略或角色规则。
type AuthorizationRule struct {
	PType  string
	Values []string
}

// AuthorizerEnforcer 抽象带组织域的 RBAC 授权能力。
type AuthorizerEnforcer interface {
	Enforce(context.Context, string, string, string, string, string, string) (bool, error)
	AddPolicy(context.Context, string, string, string, string, string, string) (bool, error)
	AddRoleForUser(context.Context, string, string, string) (bool, error)
	DeleteRoleForUser(context.Context, string, string, string) (bool, error)
	GetRolesForUser(context.Context, string, string) ([]string, error)
	LoadRules(context.Context, []AuthorizationRule) error
}

// TOTPKey 是创建 TOTP 因子后返回的密钥和绑定 URL。
type TOTPKey struct {
	Secret string
	URL    string
}

// TOTPProvider 抽象 TOTP 生成和校验能力。
type TOTPProvider interface {
	GenerateTOTP(string, string) (TOTPKey, error)
	ValidateTOTP(string, string) bool
}

// RPCHandlerFunc 是内部 RPC 方法的处理函数签名。
type RPCHandlerFunc func(context.Context, json.RawMessage) (any, error)

// RPCRegistry 是 RPC 方法注册端口。
type RPCRegistry interface {
	Register(string, RPCHandlerFunc) error
	Methods() []string
}

// RPCError 表示可映射为 JSON-RPC 错误响应的业务错误。
type RPCError struct {
	Code    int
	Message string
	Data    any
}

// Error 实现 error 接口，返回可展示的 RPC 错误消息。
func (e *RPCError) Error() string {
	return e.Message
}

// InvalidRPCParams 构造 JSON-RPC invalid params 错误。
func InvalidRPCParams(message string) *RPCError {
	return &RPCError{Code: -32602, Message: message}
}

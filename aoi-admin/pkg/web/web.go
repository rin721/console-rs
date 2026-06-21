package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	urlpath "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var ErrStaticSPAIndexMissing = errors.New("static spa index.html missing")

// Context 是暴露给内部层的 HTTP 请求边界，隐藏 gin.Context 的具体实现。
type Context interface {
	Request() *http.Request
	RequestContext() context.Context
	GetHeader(name string) string
	Header(name, value string)
	Cookie(name string) (string, error)
	SetCookie(cookie *http.Cookie)
	Set(key string, value any)
	Get(key any) (any, bool)
	Param(name string) string
	BindJSON(dest any) error
	JSON(status int, body any)
	Data(status int, contentType string, data []byte)
	AbortWithStatusJSON(status int, body any)
	Next()
	Path() string
	Method() string
	ClientIP() string
	Status() int
}

// HandlerFunc 是与底层路由框架解耦的传输层处理函数。
type HandlerFunc func(Context)

// Router 是内部传输层使用的路由注册面，避免业务代码直接依赖 Gin。
type Router interface {
	Use(...HandlerFunc)
	GET(string, HandlerFunc)
	POST(string, HandlerFunc)
	PATCH(string, HandlerFunc)
	PUT(string, HandlerFunc)
	DELETE(string, HandlerFunc)
	ANY(string, HandlerFunc)
	Group(string) Router
}

// Engine 封装底层 HTTP router，并只向应用层暴露项目自有类型。
type Engine struct {
	engine *gin.Engine
}

// RouteInfo 暴露已注册路由的元数据，同时避免泄露 Gin 的路由类型。
type RouteInfo struct {
	Method  string
	Path    string
	Handler string
}

type group struct {
	group *gin.RouterGroup
}

type contextAdapter struct {
	ctx *gin.Context
}

// CORSConfig 描述 CORS 中间件配置，调用方不需要直接导入 gin-contrib/cors。
type CORSConfig struct {
	Enabled          bool
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// StaticSPAConfig 描述一个静态单页应用的挂载点和构建产物目录。
type StaticSPAConfig struct {
	MountPath            string
	DistDir              string
	ExcludedPathPrefixes []string
}

// New 按指定 Gin 模式创建路由引擎；空模式会沿用 Gin 当前全局模式。
func New(mode string) *Engine {
	if mode != "" {
		gin.SetMode(mode)
	}
	return &Engine{engine: gin.New()}
}

// WithSilentGlobals runs fn while Gin global debug output is disabled.
func WithSilentGlobals(fn func() error) error {
	previousMode := gin.Mode()
	previousWriter := gin.DefaultWriter
	previousErrorWriter := gin.DefaultErrorWriter
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard
	defer func() {
		gin.SetMode(previousMode)
		gin.DefaultWriter = previousWriter
		gin.DefaultErrorWriter = previousErrorWriter
	}()
	return fn()
}

// Recovery 返回默认恢复中间件，并保持对外 HandlerFunc 抽象不变。
func Recovery() HandlerFunc {
	return wrapGinHandler(gin.Recovery())
}

// CORS 返回按配置创建的 CORS 中间件；禁用时只继续后续处理链。
func CORS(cfg CORSConfig) HandlerFunc {
	if !cfg.Enabled {
		return func(c Context) {
			c.Next()
		}
	}
	return wrapGinHandler(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowOrigins,
		AllowMethods:     cfg.AllowMethods,
		AllowHeaders:     cfg.AllowHeaders,
		ExposeHeaders:    cfg.ExposeHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           time.Duration(cfg.MaxAge) * time.Second,
	}))
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.engine.ServeHTTP(w, r)
}

// Routes 返回当前已注册路由快照，主要用于启动日志和测试断言。
func (e *Engine) Routes() []RouteInfo {
	routes := e.engine.Routes()
	out := make([]RouteInfo, 0, len(routes))
	for _, route := range routes {
		out = append(out, RouteInfo{
			Method:  route.Method,
			Path:    route.Path,
			Handler: route.Handler,
		})
	}
	return out
}

// MountStaticSPA 在指定前缀托管静态单页应用，并把非资源路由回退到 SPA 入口文件。
func (e *Engine) MountStaticSPA(cfg StaticSPAConfig) error {
	mountPath := normalizeMountPath(cfg.MountPath)
	if mountPath == "" || !strings.HasPrefix(mountPath, "/") {
		return errors.New("mount path must be an absolute path")
	}
	if mountPath == "/" {
		handler := func(c *gin.Context) {
			serveRootStaticSPA(c, cfg.DistDir, cfg.ExcludedPathPrefixes)
		}
		e.engine.GET("/", handler)
		e.engine.NoRoute(handler)
		if _, ok := staticSPAIndexPath(cfg.DistDir); !ok {
			return ErrStaticSPAIndexMissing
		}
		return nil
	}

	handler := func(c *gin.Context) {
		serveStaticSPAPath(c, cfg.DistDir, c.Param("filepath"))
	}
	e.engine.GET(mountPath, handler)
	e.engine.GET(mountPath+"/*filepath", handler)

	if _, ok := staticSPAIndexPath(cfg.DistDir); !ok {
		return ErrStaticSPAIndexMissing
	}
	return nil
}

func (e *Engine) Use(handlers ...HandlerFunc) {
	e.engine.Use(wrapHandlers(handlers)...)
}

func (e *Engine) GET(path string, handler HandlerFunc) {
	e.engine.GET(path, wrapHandler(handler))
}

func (e *Engine) POST(path string, handler HandlerFunc) {
	e.engine.POST(path, wrapHandler(handler))
}

func (e *Engine) PATCH(path string, handler HandlerFunc) {
	e.engine.PATCH(path, wrapHandler(handler))
}

func (e *Engine) PUT(path string, handler HandlerFunc) {
	e.engine.PUT(path, wrapHandler(handler))
}

func (e *Engine) DELETE(path string, handler HandlerFunc) {
	e.engine.DELETE(path, wrapHandler(handler))
}

func (e *Engine) ANY(path string, handler HandlerFunc) {
	e.engine.Any(path, wrapHandler(handler))
}

func (e *Engine) Group(path string) Router {
	return &group{group: e.engine.Group(path)}
}

func (g *group) Use(handlers ...HandlerFunc) {
	g.group.Use(wrapHandlers(handlers)...)
}

func (g *group) GET(path string, handler HandlerFunc) {
	g.group.GET(path, wrapHandler(handler))
}

func (g *group) POST(path string, handler HandlerFunc) {
	g.group.POST(path, wrapHandler(handler))
}

func (g *group) PATCH(path string, handler HandlerFunc) {
	g.group.PATCH(path, wrapHandler(handler))
}

func (g *group) PUT(path string, handler HandlerFunc) {
	g.group.PUT(path, wrapHandler(handler))
}

func (g *group) DELETE(path string, handler HandlerFunc) {
	g.group.DELETE(path, wrapHandler(handler))
}

func (g *group) ANY(path string, handler HandlerFunc) {
	g.group.Any(path, wrapHandler(handler))
}

func (g *group) Group(path string) Router {
	return &group{group: g.group.Group(path)}
}

// serveStaticSPA 优先返回真实静态资源；非资源路径回退到入口文件以支持前端路由。
func serveRootStaticSPA(c *gin.Context, distDir string, excludedPathPrefixes []string) {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		c.Status(http.StatusNotFound)
		return
	}
	if rootSPAPathExcluded(c.Request.URL.Path, excludedPathPrefixes) {
		c.Status(http.StatusNotFound)
		return
	}
	serveStaticSPAPath(c, distDir, c.Request.URL.Path)
}

func serveStaticSPAPath(c *gin.Context, distDir string, requestPath string) {
	cleanPath := cleanSPARequestPath(requestPath)
	if cleanPath != "" {
		filePath, ok := safeJoin(distDir, cleanPath)
		if !ok {
			c.Status(http.StatusNotFound)
			return
		}
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			c.File(filePath)
			return
		}
		if isStaticAssetPath(cleanPath) {
			c.Status(http.StatusNotFound)
			return
		}
	}

	indexPath, ok := staticSPAIndexPath(distDir)
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	c.File(indexPath)
}

func rootSPAPathExcluded(requestPath string, prefixes []string) bool {
	path := normalizeMountPath(requestPath)
	if path == "" {
		path = "/"
	}
	for _, prefix := range prefixes {
		prefix = normalizeMountPath(prefix)
		if prefix == "" || prefix == "/" {
			continue
		}
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

// staticSPAIndexPath 兼容常见 SPA 构建产物，优先 index.html，随后回退到 200.html。
func staticSPAIndexPath(distDir string) (string, bool) {
	for _, name := range []string{"index.html", "200.html"} {
		path := filepath.Join(distDir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

// cleanSPARequestPath 规范化通配路由参数，保证后续路径拼接只处理相对路径片段。
func cleanSPARequestPath(value string) string {
	value = strings.TrimPrefix(value, "/")
	if value == "" {
		return ""
	}
	cleaned := urlpath.Clean("/" + value)
	if cleaned == "/" || cleaned == "." {
		return ""
	}
	return strings.TrimPrefix(cleaned, "/")
}

// safeJoin 将请求路径限制在静态目录内，避免通过 .. 或绝对路径读取目录外文件。
func safeJoin(root string, cleanPath string) (string, bool) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	filePath := filepath.Join(absRoot, filepath.FromSlash(cleanPath))
	rel, err := filepath.Rel(absRoot, filePath)
	if err != nil {
		return "", false
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", false
	}
	return filePath, true
}

// isStaticAssetPath 用于区分资源缺失和前端路由回退，避免把 JS/CSS 404 伪装成 index.html。
func isStaticAssetPath(value string) bool {
	if strings.HasPrefix(value, "assets/") {
		return true
	}
	return urlpath.Ext(value) != ""
}

// normalizeMountPath 将挂载点收敛为非根绝对路径格式，由调用方继续校验是否允许。
func normalizeMountPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "/") {
		return value
	}
	if value == "/" {
		return "/"
	}
	return "/" + strings.Trim(strings.TrimRight(value, "/"), "/")
}

func (c contextAdapter) Request() *http.Request {
	return c.ctx.Request
}

func (c contextAdapter) ResponseWriter() http.ResponseWriter {
	return c.ctx.Writer
}

func (c contextAdapter) RequestContext() context.Context {
	return c.ctx.Request.Context()
}

func (c contextAdapter) GetHeader(name string) string {
	return c.ctx.GetHeader(name)
}

func (c contextAdapter) Header(name, value string) {
	c.ctx.Header(name, value)
}

func (c contextAdapter) Cookie(name string) (string, error) {
	return c.ctx.Cookie(name)
}

func (c contextAdapter) SetCookie(cookie *http.Cookie) {
	if cookie == nil {
		return
	}
	http.SetCookie(c.ctx.Writer, cookie)
}

func (c contextAdapter) Set(key string, value any) {
	c.ctx.Set(key, value)
}

func (c contextAdapter) Get(key any) (any, bool) {
	return c.ctx.Get(key)
}

func (c contextAdapter) Param(name string) string {
	return c.ctx.Param(name)
}

func (c contextAdapter) BindJSON(dest any) error {
	return c.ctx.ShouldBindJSON(dest)
}

func (c contextAdapter) JSON(status int, body any) {
	c.ctx.JSON(status, body)
}

func (c contextAdapter) Data(status int, contentType string, data []byte) {
	c.ctx.Data(status, contentType, data)
}

func (c contextAdapter) AbortWithStatusJSON(status int, body any) {
	c.ctx.AbortWithStatusJSON(status, body)
}

func (c contextAdapter) Next() {
	c.ctx.Next()
}

func (c contextAdapter) Path() string {
	return c.ctx.Request.URL.Path
}

func (c contextAdapter) Method() string {
	return c.ctx.Request.Method
}

func (c contextAdapter) ClientIP() string {
	return c.ctx.ClientIP()
}

func (c contextAdapter) Status() int {
	return c.ctx.Writer.Status()
}

func wrapHandlers(handlers []HandlerFunc) []gin.HandlerFunc {
	wrapped := make([]gin.HandlerFunc, 0, len(handlers))
	for _, handler := range handlers {
		wrapped = append(wrapped, wrapHandler(handler))
	}
	return wrapped
}

func wrapHandler(handler HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(contextAdapter{ctx: c})
	}
}

// wrapGinHandler 只在收到本包创建的 contextAdapter 时桥接到 Gin，避免误用外部 Context 实现。
func wrapGinHandler(handler gin.HandlerFunc) HandlerFunc {
	return func(c Context) {
		adapter, ok := c.(contextAdapter)
		if !ok {
			c.Next()
			return
		}
		handler(adapter.ctx)
	}
}

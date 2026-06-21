// Package httptransport 负责装配 HTTP 中间件、业务路由、插件协议入口和 WebUI。
package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/internal/middleware"
	iamhandler "github.com/rei0721/go-scaffold/internal/modules/iam/handler"
	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
	systemhandler "github.com/rei0721/go-scaffold/internal/modules/system/handler"
	systemmodel "github.com/rei0721/go-scaffold/internal/modules/system/model"
	systemservice "github.com/rei0721/go-scaffold/internal/modules/system/service"
	projectplugin "github.com/rei0721/go-scaffold/internal/plugin"
	"github.com/rei0721/go-scaffold/internal/ports"
	appconstants "github.com/rei0721/go-scaffold/types/constants"
	apperrors "github.com/rei0721/go-scaffold/types/errors"
	"github.com/rei0721/go-scaffold/types/result"
)

// RouterDeps 聚合 HTTP 路由装配所需依赖。
type RouterDeps struct {
	Router           ports.HTTPRouter
	StaticSPA        ports.StaticSPAMounter
	Logger           ports.Logger
	I18n             ports.I18n
	Database         ports.Database
	TraceIDGenerator ports.IDGenerator
	Middleware       middleware.MiddlewareConfig
	IAMHandler       *iamhandler.Handler
	PluginHandler    *projectplugin.Handler
	PluginProtocol   *projectplugin.ProtocolHandler
	PluginBasePath   string
	SystemHandler    *systemhandler.Handler
	SetupHandler     SetupHandler
	IAMAuth          middleware.Authenticator
	IAMAuthz         middleware.Authorizer
	WebUI            WebUIDeps
}

type SetupHandler interface {
	Status(ports.HTTPContext)
	Schema(ports.HTTPContext)
	CreateRun(ports.HTTPContext)
	RetryRun(ports.HTTPContext)
	Logs(ports.HTTPContext)
	SaveConfig(ports.HTTPContext)
	TestConfig(ports.HTTPContext)
	SkipStep(ports.HTTPContext)
	Complete(ports.HTTPContext)
}

// WebUIDeps 描述 WebUI 静态产物挂载所需配置。
type WebUIDeps struct {
	Enabled   bool
	MountPath string
	DistDir   string
}

// NewRouter 把中间件和业务路由注册到传入的 router。
func NewRouter(deps RouterDeps) ports.HTTPRouter {
	r := deps.Router
	if r == nil {
		return nil
	}

	if deps.I18n != nil {
		r.Use(middleware.I18n(deps.I18n))
	}
	r.Use(middleware.TraceID(deps.Middleware.TraceID, deps.TraceIDGenerator))
	r.Use(middleware.CORSMiddleware(deps.Middleware.CORS))
	if deps.Logger != nil {
		r.Use(middleware.Logger(deps.Middleware.Logger, deps.Logger))
		r.Use(middleware.Recovery(deps.Middleware.Recovery, deps.Logger))
	} else {
		r.Use(middleware.Recovery(deps.Middleware.Recovery, nil))
	}

	r.GET(appconstants.HTTPHealthPath, health)
	r.GET(appconstants.HTTPReadyPath, ready(deps.Database))
	r.GET(OpenAPIPath, openAPIYAML)
	if deps.PluginProtocol != nil {
		registerPluginProtocolRoutes(r, deps)
	}

	registeredContracts := []RouteContract{
		mustRouteContract("probe.health"),
		mustRouteContract("probe.ready"),
		mustRouteContract("openapi.yaml"),
	}
	v1 := r.Group(appconstants.APIBasePath)
	if deps.SetupHandler != nil {
		registeredContracts = append(registeredContracts, registerSetupRoutes(v1, deps)...)
	}
	if deps.IAMHandler != nil {
		registeredContracts = append(registeredContracts, registerIAMRoutes(v1, deps)...)
	}
	if deps.PluginHandler != nil {
		registeredContracts = append(registeredContracts, registerPluginRoutes(v1, deps)...)
	}
	if deps.SystemHandler != nil {
		registeredContracts = append(registeredContracts, registerSystemRoutes(v1, deps)...)
		deps.SystemHandler.RegisterAPIs(catalogAPIContracts(registeredContracts))
	}
	registerWebUI(deps)

	return r
}

func registerSetupRoutes(v1 ports.HTTPRouter, deps RouterDeps) []RouteContract {
	setup := v1.Group("/setup")
	setup.Use(middleware.RateLimit(middleware.RateLimitConfig{Enabled: true, Limit: 120, Window: time.Minute}))
	specs := []routeSpec{
		routeSpecFor("setup.status", deps.SetupHandler.Status),
		routeSpecFor("setup.schema", deps.SetupHandler.Schema),
		routeSpecFor("setup.config.save", deps.SetupHandler.SaveConfig),
		routeSpecFor("setup.config.test", deps.SetupHandler.TestConfig),
		routeSpecFor("setup.run.create", deps.SetupHandler.CreateRun),
		routeSpecFor("setup.run.retry", deps.SetupHandler.RetryRun),
		routeSpecFor("setup.step.skip", deps.SetupHandler.SkipStep),
		routeSpecFor("setup.run.logs", deps.SetupHandler.Logs),
		routeSpecFor("setup.complete", deps.SetupHandler.Complete),
	}
	registerRouteSpecs(setup, appconstants.APIPath("setup"), specs)
	return routeContractsFromSpecs(specs)
}

// registerWebUI 挂载 WebUI 静态产物。
// 缺少 mounter 或静态入口时只记录告警，不阻断 API 路由启动。
func registerWebUI(deps RouterDeps) {
	if !deps.WebUI.Enabled {
		return
	}
	mounter := deps.StaticSPA
	if mounter == nil {
		if candidate, ok := deps.Router.(ports.StaticSPAMounter); ok {
			mounter = candidate
		}
	}
	if mounter == nil {
		if deps.Logger != nil {
			deps.Logger.Warn("webui mount skipped", "mount_path", deps.WebUI.MountPath, "dist_dir", deps.WebUI.DistDir, "error", "static spa mounter missing")
		}
		return
	}
	err := mounter.MountStaticSPA(ports.StaticSPAConfig{
		MountPath:            deps.WebUI.MountPath,
		DistDir:              deps.WebUI.DistDir,
		ExcludedPathPrefixes: staticSPAExcludedPathPrefixes(deps),
	})
	if err == nil {
		if deps.Logger != nil {
			deps.Logger.Info("webui mounted", "mount_path", deps.WebUI.MountPath, "dist_dir", deps.WebUI.DistDir)
		}
		return
	}
	if deps.Logger == nil {
		return
	}
	if errors.Is(err, ports.ErrStaticSPAIndexMissing) {
		deps.Logger.Warn("webui static files missing", "mount_path", deps.WebUI.MountPath, "dist_dir", deps.WebUI.DistDir)
		return
	}
	deps.Logger.Warn("webui mount skipped", "mount_path", deps.WebUI.MountPath, "dist_dir", deps.WebUI.DistDir, "error", err)
}

func staticSPAExcludedPathPrefixes(deps RouterDeps) []string {
	prefixes := []string{
		appconstants.APIPathRoot,
		appconstants.APIBasePath,
		appconstants.HTTPHealthPath,
		appconstants.HTTPReadyPath,
		OpenAPIPath,
	}
	if deps.PluginProtocol != nil {
		basePath := strings.TrimRight(strings.TrimSpace(deps.PluginBasePath), "/")
		if basePath == "" {
			basePath = "/plugin-api/v1"
		}
		prefixes = append(prefixes, basePath)
	}
	return prefixes
}

// registerIAMRoutes 注册 IAM 公开、受保护和组织作用域路由。
// 登录、邀请接受等公开入口使用独立限流；组织路由统一叠加 orgId 校验和权限校验。
func registerIAMRoutes(v1 ports.HTTPRouter, deps RouterDeps) []RouteContract {
	registered := make([]RouteContract, 0, 32)
	auth := v1.Group("/auth")
	auth.Use(middleware.RateLimit(middleware.RateLimitConfig{Enabled: true, Limit: 20, Window: time.Minute}))
	authSpecs := []routeSpec{
		routeSpecFor("iam.setup.status", deps.IAMHandler.SetupStatus),
		routeSpecFor("iam.setup.initial-admin", deps.IAMHandler.InitialAdminSetup),
		routeSpecFor("iam.signup", deps.IAMHandler.Signup),
		routeSpecFor("iam.email-verification.confirm", deps.IAMHandler.ConfirmEmailVerification),
		routeSpecFor("iam.captcha", deps.IAMHandler.Captcha),
		routeSpecFor("iam.login", deps.IAMHandler.Login),
		routeSpecFor("iam.refresh", deps.IAMHandler.Refresh),
		routeSpecFor("iam.password.forgot", deps.IAMHandler.ForgotPassword),
		routeSpecFor("iam.password.reset", deps.IAMHandler.ResetPassword),
	}
	registerRouteSpecs(auth, appconstants.APIPath("auth"), authSpecs)
	registered = append(registered, routeContractsFromSpecs(authSpecs)...)

	invitations := v1.Group("/invitations")
	invitations.Use(middleware.RateLimit(middleware.RateLimitConfig{Enabled: true, Limit: 20, Window: time.Minute}))
	invitationSpecs := []routeSpec{
		routeSpecFor("iam.invitation.accept", deps.IAMHandler.AcceptInvitation),
	}
	registerRouteSpecs(invitations, appconstants.APIPath("invitations"), invitationSpecs)
	registered = append(registered, routeContractsFromSpecs(invitationSpecs)...)

	protected := v1.Group("")
	protected.Use(middleware.Auth(deps.IAMAuth, iamAuthMiddlewareConfig(deps)))
	protected.Use(middleware.CSRF(iamCSRFMiddlewareConfig(deps)))
	protected.Use(OperationRecorder(deps.SystemHandler))
	protectedSpecs := []routeSpec{
		routeSpecFor("iam.logout", deps.IAMHandler.Logout),
		routeSpecFor("iam.switch-org", deps.IAMHandler.SwitchOrg),
		routeSpecFor("iam.mfa.setup", deps.IAMHandler.SetupMFA),
		routeSpecFor("iam.mfa.verify", deps.IAMHandler.VerifyMFA),
		routeSpecFor("iam.me", deps.IAMHandler.Me),
		routeSpecFor("iam.me.session", deps.IAMHandler.Session),
		routeSpecFor("iam.me.orgs", deps.IAMHandler.MyOrganizations),
	}
	registerRouteSpecs(protected, appconstants.APIBasePath, protectedSpecs)
	registered = append(registered, routeContractsFromSpecs(protectedSpecs)...)

	orgs := protected.Group("/orgs")
	orgSpecs := []routeSpec{
		routeSpecFor("iam.orgs.list", deps.IAMHandler.ListOrganizations),
		routeSpecFor("iam.orgs.create", deps.IAMHandler.CreateOrganization),
		routeSpecFor("iam.orgs.update", deps.IAMHandler.UpdateOrganization),
		routeSpecFor("iam.users.list", deps.IAMHandler.ListUsers),
		routeSpecFor("iam.users.update", deps.IAMHandler.UpdateUser),
		routeSpecFor("iam.users.invite", deps.IAMHandler.InviteUser),
		routeSpecFor("iam.invitations.list", deps.IAMHandler.ListInvitations),
		routeSpecFor("iam.invitations.revoke", deps.IAMHandler.RevokeInvitation),
		routeSpecFor("iam.api-tokens.list", deps.IAMHandler.ListAPITokens),
		routeSpecFor("iam.api-tokens.create", deps.IAMHandler.CreateAPIToken),
		routeSpecFor("iam.api-tokens.revoke", deps.IAMHandler.RevokeAPIToken),
		routeSpecFor("iam.roles.list", deps.IAMHandler.ListRoles),
		routeSpecFor("iam.roles.create", deps.IAMHandler.CreateRole),
		routeSpecFor("iam.roles.update", deps.IAMHandler.UpdateRole),
		routeSpecFor("iam.permissions.list", deps.IAMHandler.ListPermissions),
		routeSpecFor("iam.sessions.list", deps.IAMHandler.ListSessions),
		routeSpecFor("iam.sessions.revoke", deps.IAMHandler.RevokeSession),
		routeSpecFor("iam.audit-logs.list", deps.IAMHandler.ListAuditLogs),
	}
	registerProtectedRouteSpecs(orgs, appconstants.APIPath("orgs"), deps, orgSpecs)
	registered = append(registered, routeContractsFromSpecs(orgSpecs)...)
	return registered
}

// registerPluginRoutes 注册管理端插件查询路由。
func iamAuthMiddlewareConfig(deps RouterDeps) middleware.AuthConfig {
	if deps.IAMHandler == nil {
		return middleware.AuthConfig{}
	}
	return deps.IAMHandler.AuthMiddlewareConfig()
}

func iamCSRFMiddlewareConfig(deps RouterDeps) middleware.CSRFConfig {
	if deps.IAMHandler == nil {
		return middleware.CSRFConfig{}
	}
	return deps.IAMHandler.CSRFMiddlewareConfig()
}

func registerPluginRoutes(v1 ports.HTTPRouter, deps RouterDeps) []RouteContract {
	plugins := v1.Group("/plugins")
	plugins.Use(middleware.Auth(deps.IAMAuth, iamAuthMiddlewareConfig(deps)))
	plugins.Use(middleware.CSRF(iamCSRFMiddlewareConfig(deps)))
	plugins.Use(OperationRecorder(deps.SystemHandler))
	specs := []routeSpec{
		routeSpecFor("plugins.list", deps.PluginHandler.List),
		routeSpecFor("plugins.get", deps.PluginHandler.Get),
		routeSpecFor("plugins.health", deps.PluginHandler.Health),
		routeSpecFor("plugins.capabilities", deps.PluginHandler.Capabilities),
	}
	registerProtectedRouteSpecs(plugins, appconstants.APIPath("plugins"), deps, specs)
	return routeContractsFromSpecs(specs)
}

// registerPluginProtocolRoutes 注册远程插件协议路由。
// PluginBasePath 为空时使用默认 /plugin-api/v1，避免和管理端 API 混在同一鉴权体系中。
func registerPluginProtocolRoutes(r ports.HTTPRouter, deps RouterDeps) {
	basePath := strings.TrimRight(strings.TrimSpace(deps.PluginBasePath), "/")
	if basePath == "" {
		basePath = "/plugin-api/v1"
	}
	plugins := r.Group(basePath)
	plugins.POST("/negotiate", deps.PluginProtocol.Negotiate)
	plugins.POST("/register", deps.PluginProtocol.Register)
	plugins.POST("/heartbeat", deps.PluginProtocol.Heartbeat)
	plugins.POST("/lease", deps.PluginProtocol.RenewLease)
	plugins.POST("/unregister", deps.PluginProtocol.Unregister)
	plugins.POST("/health-check", deps.PluginProtocol.HealthCheck)
	plugins.POST("/capabilities", deps.PluginProtocol.ListCapabilities)
	plugins.POST("/invoke", deps.PluginProtocol.Invoke)
	plugins.POST("/events", deps.PluginProtocol.PushEvent)
	plugins.POST("/subscriptions", deps.PluginProtocol.SubscribeEvent)
	plugins.POST("/context", deps.PluginProtocol.InjectContext)
	plugins.POST("/injected-schema", deps.PluginProtocol.GetInjectedSchema)
	plugins.POST("/status", deps.PluginProtocol.ReportStatus)
	plugins.POST("/metadata", deps.PluginProtocol.SyncMetadata)
	plugins.POST("/drain", deps.PluginProtocol.Drain)
	plugins.POST("/ws", deps.PluginProtocol.WSEnvelope)
}

// registerSystemRoutes 注册系统管理路由。
// 每个写入或敏感读取入口都在这里绑定 IAM 权限，服务层只负责业务规则。
func registerSystemRoutes(v1 ports.HTTPRouter, deps RouterDeps) []RouteContract {
	registered := make([]RouteContract, 0, len(systemRouteContracts()))
	publicSystem := v1.Group("/system")
	publicSpecs := []routeSpec{
		routeSpecFor("system.public-settings", deps.SystemHandler.PublicSettings),
	}
	registerRouteSpecs(publicSystem, appconstants.APIPath("system"), publicSpecs)
	registered = append(registered, routeContractsFromSpecs(publicSpecs)...)

	system := v1.Group("/system")
	system.Use(middleware.Auth(deps.IAMAuth, iamAuthMiddlewareConfig(deps)))
	system.Use(middleware.CSRF(iamCSRFMiddlewareConfig(deps)))
	system.Use(OperationRecorder(deps.SystemHandler))
	protectedSpecs := []routeSpec{
		routeSpecFor("system.menus", deps.SystemHandler.ListMenus),
		routeSpecFor("system.config.get", deps.SystemHandler.ListConfig),
		routeSpecFor("system.config.update", deps.SystemHandler.UpdateConfig),
		routeSpecFor("system.server-info", deps.SystemHandler.GetServerInfo),
		routeSpecFor("system.server-metrics.history", deps.SystemHandler.GetServerMetricsHistory),
		routeSpecFor("system.traffic-hijack.overview", deps.SystemHandler.GetTrafficHijackOverview),
		routeSpecFor("system.traffic-hijack.targets.list", deps.SystemHandler.ListTrafficProbeTargets),
		routeSpecFor("system.traffic-hijack.targets.create", deps.SystemHandler.CreateTrafficProbeTarget),
		routeSpecFor("system.traffic-hijack.targets.update", deps.SystemHandler.UpdateTrafficProbeTarget),
		routeSpecFor("system.traffic-hijack.targets.delete", deps.SystemHandler.DeleteTrafficProbeTarget),
		routeSpecFor("system.traffic-hijack.targets.probe", deps.SystemHandler.RunTrafficProbe),
		routeSpecFor("system.traffic-hijack.results", deps.SystemHandler.ListTrafficProbeResults),
		routeSpecFor("system.traffic-hijack.events", deps.SystemHandler.ListTrafficHijackEvents),
		routeSpecFor("system.traffic-hijack.events.resolve", deps.SystemHandler.ResolveTrafficHijackEvent),
		routeSpecFor("system.traffic-hijack.stream", deps.SystemHandler.StreamTrafficHijack),
		routeSpecFor("system.apis", deps.SystemHandler.ListAPIs),
		routeSpecFor("system.apis.sync", deps.SystemHandler.SyncAPIs),
		routeSpecFor("system.apis.permissions.sync", deps.SystemHandler.SyncPermissions),
		routeSpecFor("system.operation-records.list", deps.SystemHandler.ListOperationRecords),
		routeSpecFor("system.operation-records.delete", deps.SystemHandler.DeleteOperationRecords),
		routeSpecFor("system.versions.list", deps.SystemHandler.ListVersions),
		routeSpecFor("system.versions.export", deps.SystemHandler.ExportVersion),
		routeSpecFor("system.versions.import", deps.SystemHandler.ImportVersion),
		routeSpecFor("system.versions.delete-batch", deps.SystemHandler.DeleteVersions),
		routeSpecFor("system.versions.sources", deps.SystemHandler.ListVersionSources),
		routeSpecFor("system.versions.get", deps.SystemHandler.GetVersion),
		routeSpecFor("system.versions.download", deps.SystemHandler.DownloadVersion),
		routeSpecFor("system.versions.delete", deps.SystemHandler.DeleteVersion),
		routeSpecFor("system.media.categories", deps.SystemHandler.ListMediaCategories),
		routeSpecFor("system.media.categories.upsert", deps.SystemHandler.UpsertMediaCategory),
		routeSpecFor("system.media.categories.delete", deps.SystemHandler.DeleteMediaCategory),
		routeSpecFor("system.media.assets", deps.SystemHandler.ListMediaAssets),
		routeSpecFor("system.media.assets.upload", deps.SystemHandler.UploadMediaAsset),
		routeSpecFor("system.media.assets.resumable.check", deps.SystemHandler.CheckMediaResumableUpload),
		routeSpecFor("system.media.assets.resumable.chunks", deps.SystemHandler.UploadMediaChunk),
		routeSpecFor("system.media.assets.resumable.complete", deps.SystemHandler.CompleteMediaResumableUpload),
		routeSpecFor("system.media.assets.resumable.abort", deps.SystemHandler.AbortMediaResumableUpload),
		routeSpecFor("system.media.assets.import-url", deps.SystemHandler.ImportMediaURLs),
		routeSpecFor("system.media.assets.update", deps.SystemHandler.UpdateMediaAsset),
		routeSpecFor("system.media.assets.download", deps.SystemHandler.DownloadMediaAsset),
		routeSpecFor("system.media.assets.delete", deps.SystemHandler.DeleteMediaAsset),
		routeSpecFor("system.parameters.list", deps.SystemHandler.ListParameters),
		routeSpecFor("system.parameters.create", deps.SystemHandler.CreateParameter),
		routeSpecFor("system.parameters.delete-batch", deps.SystemHandler.DeleteParameters),
		routeSpecFor("system.parameters.value", deps.SystemHandler.GetParameterByKey),
		routeSpecFor("system.parameters.get", deps.SystemHandler.GetParameter),
		routeSpecFor("system.parameters.update", deps.SystemHandler.UpdateParameter),
		routeSpecFor("system.parameters.delete", deps.SystemHandler.DeleteParameter),
		routeSpecFor("system.dictionaries.list", deps.SystemHandler.ListDictionaries),
		routeSpecFor("system.dictionaries.create", deps.SystemHandler.CreateDictionary),
		routeSpecFor("system.dictionaries.update", deps.SystemHandler.UpdateDictionary),
		routeSpecFor("system.dictionaries.delete", deps.SystemHandler.DeleteDictionary),
		routeSpecFor("system.dictionary-items.create", deps.SystemHandler.CreateDictionaryItem),
		routeSpecFor("system.dictionary-items.update", deps.SystemHandler.UpdateDictionaryItem),
		routeSpecFor("system.dictionary-items.delete", deps.SystemHandler.DeleteDictionaryItem),
	}
	registerProtectedRouteSpecs(system, appconstants.APIPath("system"), deps, protectedSpecs)
	registered = append(registered, routeContractsFromSpecs(protectedSpecs)...)
	return registered
}

func registerRouteSpecs(router ports.HTTPRouter, basePath string, specs []routeSpec) {
	for _, spec := range specs {
		registerRoute(router, spec.Contract.Method, routeRelativePath(basePath, spec.Contract.Path), spec.Handler)
	}
}

func registerProtectedRouteSpecs(router ports.HTTPRouter, basePath string, deps RouterDeps, specs []routeSpec) {
	for _, spec := range specs {
		registerRoute(router, spec.Contract.Method, routeRelativePath(basePath, spec.Contract.Path), protectedRouteHandler(deps, spec.Contract, spec.Handler))
	}
}

func protectedRouteHandler(deps RouterDeps, contract RouteContract, handler ports.HTTPHandlerFunc) ports.HTTPHandlerFunc {
	if contract.Permission == "" {
		return handler
	}
	object, action, ok := strings.Cut(contract.Permission, ":")
	if !ok || object == "" || action == "" {
		panic("invalid route permission: " + contract.ID + " " + contract.Permission)
	}
	handler = middleware.RequirePermission(deps.IAMAuthz, iamservice.PermissionContext{
		ProductCode: contract.ProductCode,
		Scope:       contract.Scope,
		Object:      object,
		Action:      action,
	}, handler)
	if strings.Contains(contract.Path, ":orgId") {
		handler = middleware.RequireOrgParam("orgId", handler)
	}
	return handler
}

func registerRoute(router ports.HTTPRouter, method string, path string, handler ports.HTTPHandlerFunc) {
	switch method {
	case http.MethodGet:
		router.GET(path, handler)
	case http.MethodPost:
		router.POST(path, handler)
	case http.MethodPatch:
		router.PATCH(path, handler)
	case http.MethodPut:
		router.PUT(path, handler)
	case http.MethodDelete:
		router.DELETE(path, handler)
	default:
		panic("unsupported route method: " + method)
	}
}

func routeRelativePath(basePath string, fullPath string) string {
	basePath = strings.TrimRight(basePath, "/")
	if fullPath == basePath {
		return ""
	}
	return strings.TrimPrefix(fullPath, basePath)
}

// operationRecorder 是操作记录中间件依赖的最小系统服务端口。
type operationRecorder interface {
	RecordOperation(context.Context, systemservice.OperationRecordInput) error
}

// OperationRecorder 记录 API 请求的基本审计信息。
// 中间件会恢复请求体供后续 handler 继续读取，并对系统配置更新内容做脱敏。
func OperationRecorder(recorder operationRecorder) ports.HTTPHandlerFunc {
	return func(c ports.HTTPContext) {
		if isNilOperationRecorder(recorder) || !appconstants.IsAPIPath(c.Path()) {
			c.Next()
			return
		}
		body := sanitizeOperationRequestBody(c.Method(), c.Path(), readRequestBody(c.Request()))
		start := time.Now()
		c.Next()

		status := c.Status()
		if status == 0 {
			status = http.StatusOK
		}
		principal, _ := middleware.GetPrincipal(c)
		input := systemservice.OperationRecordInput{
			Body:      body,
			IPAddress: middleware.ClientIPRealIP(c),
			LatencyMs: time.Since(start).Milliseconds(),
			Method:    c.Method(),
			Path:      c.Path(),
			Status:    status,
			TraceID:   middleware.GetTraceID(c),
			UserAgent: c.GetHeader("User-Agent"),
			UserID:    principal.UserID,
			Username:  principal.Username,
		}
		if status >= http.StatusBadRequest {
			input.ErrorMessage = http.StatusText(status)
		}
		_ = recorder.RecordOperation(context.Background(), input)
	}
}

// isNilOperationRecorder 识别接口中包裹的 nil 指针。
// handler 以接口传入时，直接与 nil 比较无法覆盖 typed nil。
func isNilOperationRecorder(recorder operationRecorder) bool {
	if recorder == nil {
		return true
	}
	value := reflect.ValueOf(recorder)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// readRequestBody 读取并恢复请求体。
// 操作记录需要读取 body，但后续 handler 仍要能正常 BindJSON 或解析表单。
func readRequestBody(req *http.Request) string {
	if req == nil || req.Body == nil {
		return ""
	}
	raw, err := io.ReadAll(req.Body)
	if err != nil {
		req.Body = io.NopCloser(bytes.NewReader(nil))
		return ""
	}
	req.Body = io.NopCloser(bytes.NewReader(raw))
	return string(raw)
}

// sanitizeOperationRequestBody 对敏感操作记录做脱敏。
// 系统配置更新可能包含密钥值，只保留 key 和 persist 标记用于审计。
func sanitizeOperationRequestBody(method string, path string, body string) string {
	if method != http.MethodPatch || path != appconstants.APIPath("system", "config") {
		return body
	}
	var payload struct {
		Items []struct {
			Key string `json:"key"`
		} `json:"items"`
		Persist bool `json:"persist"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return `{"items":"[redacted]"}`
	}
	out := struct {
		Items   []map[string]string `json:"items"`
		Persist bool                `json:"persist"`
	}{
		Items:   make([]map[string]string, 0, len(payload.Items)),
		Persist: payload.Persist,
	}
	for _, item := range payload.Items {
		out.Items = append(out.Items, map[string]string{
			"key":   item.Key,
			"value": "[redacted]",
		})
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return `{"items":"[redacted]"}`
	}
	return string(raw)
}

// catalogAPIContracts 将路由契约转换为系统模块可展示和同步的 API 清单。
// 非业务 API 会被跳过；权限、分组和展示描述由 route contract 派生。
func catalogAPIContracts(contracts []RouteContract) []systemmodel.APIEntry {
	entries := make([]systemmodel.APIEntry, 0, len(contracts))
	for _, contract := range contracts {
		if !contract.IncludeCatalog || !appconstants.IsAPIPath(contract.Path) {
			continue
		}
		entries = append(entries, systemmodel.APIEntry{
			Access:      contract.Access,
			Code:        strings.ToLower(contract.Method + " " + contract.Path),
			Group:       apiRouteGroup(contract.Path),
			Method:      contract.Method,
			Path:        contract.Path,
			Description: contract.Summary,
			Permission:  contract.Permission,
			ProductCode: contract.ProductCode,
			Scope:       contract.Scope,
			Order:       apiRouteMethodOrder(contract.Method),
		})
	}
	return entries
}

// apiRouteGroup 使用 API 基路径后的第一段作为分组编码。
func apiRouteGroup(path string) string {
	path = appconstants.TrimAPIPathPrefix(path)
	segment, _, _ := strings.Cut(path, "/")
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return "other"
	}
	return segment
}

// apiRouteMethodOrder 为同一路径下不同 method 提供稳定展示顺序。
func apiRouteMethodOrder(method string) int {
	switch method {
	case http.MethodGet:
		return 10
	case http.MethodPost:
		return 20
	case http.MethodPatch, http.MethodPut:
		return 30
	case http.MethodDelete:
		return 40
	default:
		return 50
	}
}

// health 返回轻量存活探针响应。
func openAPIYAML(c ports.HTTPContext) {
	raw, err := GenerateOpenAPIYAML()
	if err != nil {
		result.InternalError(c, "api.common.internalError")
		return
	}
	c.Data(http.StatusOK, ContentYAML, raw)
}

func health(c ports.HTTPContext) {
	result.OK(c, map[string]any{"status": "ok"})
}

// ready 执行数据库就绪检查。
func ready(db ports.Database) ports.HTTPHandlerFunc {
	return func(c ports.HTTPContext) {
		if db == nil {
			c.JSON(http.StatusServiceUnavailable, result.LocalizedErrorWithData(c, apperrors.ErrDatabaseError, "api.common.notReady", nil, "", map[string]any{
				"status": "not_ready",
				"checks": map[string]any{"database": "missing"},
			}))
			return
		}
		if err := db.Ping(c.RequestContext()); err != nil {
			c.JSON(http.StatusServiceUnavailable, result.LocalizedErrorWithData(c, apperrors.ErrDatabaseError, "api.common.notReady", nil, "", map[string]any{
				"status": "not_ready",
				"checks": map[string]any{"database": err.Error()},
			}))
			return
		}
		result.OK(c, map[string]any{
			"status": "ready",
			"checks": map[string]any{"database": "ok"},
		})
	}
}

// ReadyCheck 构造就绪探针回调。
func ReadyCheck(db ports.Database) func(context.Context) error {
	return func(ctx context.Context) error {
		if db == nil {
			return http.ErrServerClosed
		}
		return db.Ping(ctx)
	}
}

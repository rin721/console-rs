package httptransport

import (
	"net/http"
	"reflect"

	initdto "github.com/rei0721/go-scaffold/internal/app/initcenter/dto"
	iamhandler "github.com/rei0721/go-scaffold/internal/modules/iam/handler"
	iammodel "github.com/rei0721/go-scaffold/internal/modules/iam/model"
	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
	systemhandler "github.com/rei0721/go-scaffold/internal/modules/system/handler"
	systemmodel "github.com/rei0721/go-scaffold/internal/modules/system/model"
	"github.com/rei0721/go-scaffold/internal/ports"
	appconstants "github.com/rei0721/go-scaffold/types/constants"
)

const OpenAPIPath = "/openapi.yaml"

const (
	ParamInPath      = "path"
	ParamInQuery     = "query"
	ParamInHeader    = "header"
	ParamInMultipart = "multipart"

	ContentJSON      = "application/json"
	ContentYAML      = "application/yaml"
	ContentMultipart = "multipart/form-data"
	ContentOctet     = "application/octet-stream"
)

type RouteParam struct {
	Name        string
	In          string
	Type        string
	Format      string
	Required    bool
	Description string
}

type RouteContract struct {
	ID              string
	Method          string
	Path            string
	Tag             string
	Summary         string
	Description     string
	Access          string
	Permission      string
	ProductCode     string
	Scope           string
	RequestType     reflect.Type
	ResponseType    reflect.Type
	RequestContent  string
	ResponseContent string
	Status          int
	Params          []RouteParam
	RawResponse     bool
	IncludeCatalog  bool
	IncludeOpenAPI  bool
}

type routeSpec struct {
	Contract RouteContract
	Handler  ports.HTTPHandlerFunc
}

type BooleanResult struct {
	Deleted   bool `json:"deleted,omitempty"`
	LoggedOut bool `json:"loggedOut,omitempty"`
	Reset     bool `json:"reset,omitempty"`
	Revoked   bool `json:"revoked,omitempty"`
	Verified  bool `json:"verified,omitempty"`
}

type MFASetupResult struct {
	Secret     string `json:"secret"`
	OtpauthURL string `json:"otpauthUrl"`
}

func typeOf[T any]() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

func jsonType[T any]() reflect.Type {
	return typeOf[T]()
}

func routeContract(id string, method string, path string, tag string, summary string, access string, permission string, request reflect.Type, response reflect.Type, params ...RouteParam) RouteContract {
	status := http.StatusOK
	if method == http.MethodPost && request != nil {
		status = http.StatusOK
	}
	return RouteContract{
		ID:              id,
		Method:          method,
		Path:            path,
		Tag:             tag,
		Summary:         summary,
		Access:          access,
		Permission:      permission,
		RequestType:     request,
		ResponseType:    response,
		RequestContent:  ContentJSON,
		ResponseContent: ContentJSON,
		Status:          status,
		Params:          params,
		IncludeCatalog:  true,
		IncludeOpenAPI:  true,
	}
}

func publicRoute(id string, method string, path string, tag string, summary string, request reflect.Type, response reflect.Type, params ...RouteParam) RouteContract {
	contract := routeContract(id, method, path, tag, summary, systemmodel.APIAccessPublic, "", request, response, params...)
	contract.Scope = iammodel.PermissionScopePlatform
	return contract
}

func authRoute(id string, method string, path string, tag string, summary string, request reflect.Type, response reflect.Type, params ...RouteParam) RouteContract {
	contract := routeContract(id, method, path, tag, summary, systemmodel.APIAccessAuthenticated, "", request, response, params...)
	contract.Scope = iammodel.PermissionScopeTenant
	return contract
}

func permissionRoute(id string, method string, path string, tag string, summary string, permission string, request reflect.Type, response reflect.Type, params ...RouteParam) RouteContract {
	contract := routeContract(id, method, path, tag, summary, systemmodel.APIAccessPermission, permission, request, response, params...)
	contract.Scope = iammodel.PermissionScopeTenant
	return contract
}

func platformPermissionRoute(id string, method string, path string, tag string, summary string, permission string, request reflect.Type, response reflect.Type, params ...RouteParam) RouteContract {
	contract := permissionRoute(id, method, path, tag, summary, permission, request, response, params...)
	contract.Scope = iammodel.PermissionScopePlatform
	return contract
}

func pathID(name string) RouteParam {
	return RouteParam{Name: name, In: ParamInPath, Type: "integer", Format: "int64", Required: true}
}

func pathString(name string) RouteParam {
	return RouteParam{Name: name, In: ParamInPath, Type: "string", Required: true}
}

func queryParam(name string, typ string) RouteParam {
	return RouteParam{Name: name, In: ParamInQuery, Type: typ}
}

func queryInt(name string) RouteParam {
	return RouteParam{Name: name, In: ParamInQuery, Type: "integer", Format: "int32"}
}

func queryInt64(name string) RouteParam {
	return RouteParam{Name: name, In: ParamInQuery, Type: "integer", Format: "int64"}
}

func queryBool(name string) RouteParam {
	return RouteParam{Name: name, In: ParamInQuery, Type: "boolean"}
}

func queryDateTime(name string) RouteParam {
	return RouteParam{Name: name, In: ParamInQuery, Type: "string", Format: "date-time"}
}

func multipartParam(name string, typ string, required bool) RouteParam {
	return RouteParam{Name: name, In: ParamInMultipart, Type: typ, Required: required}
}

func multipartPermissionRoute(id string, method string, path string, tag string, summary string, permission string, response reflect.Type, params ...RouteParam) RouteContract {
	contract := permissionRoute(id, method, path, tag, summary, permission, nil, response, params...)
	contract.RequestContent = ContentMultipart
	return contract
}

func multipartPlatformPermissionRoute(id string, method string, path string, tag string, summary string, permission string, response reflect.Type, params ...RouteParam) RouteContract {
	contract := platformPermissionRoute(id, method, path, tag, summary, permission, nil, response, params...)
	contract.RequestContent = ContentMultipart
	return contract
}

func binaryPermissionRoute(id string, method string, path string, tag string, summary string, permission string, params ...RouteParam) RouteContract {
	contract := permissionRoute(id, method, path, tag, summary, permission, nil, nil, params...)
	contract.ResponseContent = ContentOctet
	contract.RawResponse = true
	return contract
}

func binaryPlatformPermissionRoute(id string, method string, path string, tag string, summary string, permission string, params ...RouteParam) RouteContract {
	contract := platformPermissionRoute(id, method, path, tag, summary, permission, nil, nil, params...)
	contract.ResponseContent = ContentOctet
	contract.RawResponse = true
	return contract
}

func openAPIContract() RouteContract {
	contract := publicRoute("openapi.yaml", http.MethodGet, OpenAPIPath, "OpenAPI", "读取主系统 OpenAPI 契约", nil, nil)
	contract.ResponseContent = ContentYAML
	contract.RawResponse = true
	contract.IncludeCatalog = false
	return contract
}

func mainHTTPContracts() []RouteContract {
	contracts := []RouteContract{
		publicRoute("probe.health", http.MethodGet, appconstants.HTTPHealthPath, "探针", "存活探针", nil, jsonType[map[string]string]()),
		publicRoute("probe.ready", http.MethodGet, appconstants.HTTPReadyPath, "探针", "就绪探针", nil, jsonType[map[string]any]()),
		openAPIContract(),

		publicRoute("setup.status", http.MethodGet, appconstants.APIPath("setup", "status"), "Setup", "查询统一初始化状态", nil, jsonType[initdto.Status]()),
		publicRoute("setup.schema", http.MethodGet, appconstants.APIPath("setup", "schema"), "Setup", "查询统一初始化 schema", nil, jsonType[initdto.SetupSchema]()),
		publicRoute("setup.config.save", http.MethodPatch, appconstants.APIPath("setup", "configs", ":stepKey"), "Setup", "保存初始化步骤配置", jsonType[initdto.ConfigRequest](), jsonType[initdto.ConfigSaveResult](), pathString("stepKey")),
		publicRoute("setup.config.test", http.MethodPost, appconstants.APIPath("setup", "configs", ":stepKey", "test"), "Setup", "测试初始化步骤配置", jsonType[initdto.TestRequest](), jsonType[initdto.TestResult](), pathString("stepKey")),
		publicRoute("setup.run.create", http.MethodPost, appconstants.APIPath("setup", "runs"), "Setup", "执行统一初始化流程", jsonType[initdto.RunRequest](), jsonType[initdto.RunResult]()),
		publicRoute("setup.run.retry", http.MethodPost, appconstants.APIPath("setup", "runs", ":id", "retry"), "Setup", "重试初始化运行", jsonType[initdto.RunRequest](), jsonType[initdto.RunResult](), pathString("id")),
		publicRoute("setup.step.skip", http.MethodPost, appconstants.APIPath("setup", "runs", ":id", "steps", ":stepKey", "skip"), "Setup", "跳过初始化步骤", jsonType[initdto.SkipRequest](), jsonType[initdto.RunResult](), pathString("id"), pathString("stepKey")),
		publicRoute("setup.run.logs", http.MethodGet, appconstants.APIPath("setup", "runs", ":id", "logs"), "Setup", "查询初始化运行日志", nil, jsonType[[]initdto.StepReport](), pathString("id"), queryParam("setupToken", "string")),
		publicRoute("setup.complete", http.MethodPost, appconstants.APIPath("setup", "complete"), "Setup", "完成初始化流程", jsonType[initdto.CompleteRequest](), jsonType[initdto.CompleteResult]()),

		publicRoute("iam.setup.status", http.MethodGet, appconstants.APIPath("auth", "setup", "status"), "IAM 认证", "查询首次管理员初始化状态", nil, jsonType[iamservice.SetupStatus]()),
		publicRoute("iam.setup.initial-admin", http.MethodPost, appconstants.APIPath("auth", "setup", "initial-admin"), "IAM 认证", "创建首个组织 owner", jsonType[iamhandler.InitialAdminSetupRequest](), jsonType[iamservice.TokenPair]()),
		publicRoute("iam.signup", http.MethodPost, appconstants.APIPath("auth", "signup"), "IAM 认证", "自助注册并创建首个组织 owner", jsonType[iamhandler.SignupRequest](), jsonType[iamservice.SignupResult]()),
		publicRoute("iam.email-verification.confirm", http.MethodPost, appconstants.APIPath("auth", "email-verifications", ":token", "confirm"), "IAM 认证", "确认邮箱验证并登录", nil, jsonType[iamservice.TokenPair](), pathString("token")),
		publicRoute("iam.captcha", http.MethodGet, appconstants.APIPath("auth", "captcha"), "IAM 认证", "获取登录验证码", nil, jsonType[iamservice.CaptchaChallenge]()),
		publicRoute("iam.login", http.MethodPost, appconstants.APIPath("auth", "login"), "IAM 认证", "登录并签发令牌", jsonType[iamhandler.LoginRequest](), jsonType[iamservice.TokenPair]()),
		publicRoute("iam.refresh", http.MethodPost, appconstants.APIPath("auth", "refresh"), "IAM 认证", "刷新 access token 和 refresh token", jsonType[iamhandler.RefreshRequest](), jsonType[iamservice.TokenPair]()),
		publicRoute("iam.password.forgot", http.MethodPost, appconstants.APIPath("auth", "password", "forgot"), "IAM 认证", "创建密码重置令牌", jsonType[iamhandler.ForgotPasswordRequest](), jsonType[iamservice.NotificationDelivery]()),
		publicRoute("iam.password.reset", http.MethodPost, appconstants.APIPath("auth", "password", "reset"), "IAM 认证", "使用重置令牌重置密码", jsonType[iamhandler.ResetPasswordRequest](), jsonType[BooleanResult]()),
		publicRoute("iam.invitation.accept", http.MethodPost, appconstants.APIPath("invitations", ":token", "accept"), "IAM 认证", "接受组织邀请", jsonType[iamhandler.AcceptInvitationRequest](), jsonType[iamservice.Principal](), pathString("token")),

		authRoute("iam.logout", http.MethodPost, appconstants.APIPath("auth", "logout"), "IAM 账号", "撤销当前会话", nil, jsonType[BooleanResult]()),
		authRoute("iam.switch-org", http.MethodPost, appconstants.APIPath("auth", "switch-org"), "IAM 账号", "切换当前组织并签发新令牌", jsonType[iamhandler.SwitchOrgRequest](), jsonType[iamservice.TokenPair]()),
		authRoute("iam.mfa.setup", http.MethodPost, appconstants.APIPath("auth", "mfa", "setup"), "IAM 账号", "创建或轮换 TOTP MFA 密钥", nil, jsonType[MFASetupResult]()),
		authRoute("iam.mfa.verify", http.MethodPost, appconstants.APIPath("auth", "mfa", "verify"), "IAM 账号", "验证并启用 TOTP MFA", jsonType[iamhandler.VerifyMFARequest](), jsonType[BooleanResult]()),
		authRoute("iam.me", http.MethodGet, appconstants.APIPath("me"), "IAM 账号", "获取当前用户资料", nil, jsonType[iammodel.User]()),
		authRoute("iam.me.session", http.MethodGet, appconstants.APIPath("me", "session"), "IAM 账号", "获取当前会话快照", nil, jsonType[iamservice.SessionSnapshot]()),
		authRoute("iam.me.orgs", http.MethodGet, appconstants.APIPath("me", "orgs"), "IAM 账号", "查询当前用户所属组织", nil, jsonType[[]iammodel.Organization]()),

		platformPermissionRoute("iam.orgs.list", http.MethodGet, appconstants.APIPath("orgs"), "IAM 组织", "查询组织列表", "org:read", nil, jsonType[iamservice.OrganizationPage](), organizationListParams()...),
		platformPermissionRoute("iam.orgs.create", http.MethodPost, appconstants.APIPath("orgs"), "IAM 组织", "创建组织", "org:create", jsonType[iamhandler.CreateOrgRequest](), jsonType[iammodel.Organization]()),
		permissionRoute("iam.orgs.update", http.MethodPatch, appconstants.APIPath("orgs", ":orgId"), "IAM 组织", "更新当前组织信息", "org:update", jsonType[iamhandler.UpdateOrgRequest](), jsonType[iammodel.Organization](), pathID("orgId")),
		permissionRoute("iam.users.list", http.MethodGet, appconstants.APIPath("orgs", ":orgId", "users"), "IAM 组织", "查询当前组织用户", "user:read", nil, jsonType[iamservice.OrganizationUserPage](), append([]RouteParam{pathID("orgId")}, userListParams()...)...),
		permissionRoute("iam.users.update", http.MethodPatch, appconstants.APIPath("orgs", ":orgId", "users", ":userId"), "IAM 组织", "更新成员状态或角色", "user:update", jsonType[iamhandler.UpdateUserRequest](), jsonType[iamservice.OrganizationUser](), pathID("orgId"), pathID("userId")),
		permissionRoute("iam.users.invite", http.MethodPost, appconstants.APIPath("orgs", ":orgId", "users", "invitations"), "IAM 组织", "邀请用户加入当前组织", "user:invite", jsonType[iamhandler.InviteUserRequest](), jsonType[iamservice.NotificationDelivery](), pathID("orgId")),
		permissionRoute("iam.invitations.list", http.MethodGet, appconstants.APIPath("orgs", ":orgId", "invitations"), "IAM 组织", "查询当前组织邀请", "user:invite", nil, jsonType[[]iammodel.Invitation](), pathID("orgId")),
		permissionRoute("iam.invitations.revoke", http.MethodDelete, appconstants.APIPath("orgs", ":orgId", "invitations", ":invitationId"), "IAM 组织", "撤销待处理邀请", "user:invite", nil, jsonType[BooleanResult](), pathID("orgId"), pathID("invitationId")),
		permissionRoute("iam.roles.list", http.MethodGet, appconstants.APIPath("orgs", ":orgId", "roles"), "IAM 组织", "查询当前组织角色", "role:read", nil, jsonType[[]iammodel.Role](), pathID("orgId")),
		permissionRoute("iam.roles.create", http.MethodPost, appconstants.APIPath("orgs", ":orgId", "roles"), "IAM 组织", "在当前组织创建角色", "role:create", jsonType[iamhandler.CreateRoleRequest](), jsonType[iammodel.Role](), pathID("orgId")),
		permissionRoute("iam.roles.update", http.MethodPatch, appconstants.APIPath("orgs", ":orgId", "roles", ":roleId"), "IAM 组织", "更新自定义角色", "role:update", jsonType[iamhandler.UpdateRoleRequest](), jsonType[iammodel.Role](), pathID("orgId"), pathID("roleId")),
		permissionRoute("iam.permissions.list", http.MethodGet, appconstants.APIPath("orgs", ":orgId", "permissions"), "IAM 组织", "查询可用权限", "permission:read", nil, jsonType[[]iammodel.Permission](), pathID("orgId")),
		permissionRoute("iam.api-tokens.list", http.MethodGet, appconstants.APIPath("orgs", ":orgId", "api-tokens"), "IAM API Token", "分页查询 IAM API Token", "api_token:read", nil, jsonType[iamservice.APITokenPage](), pathID("orgId"), queryParam("status", "string"), queryInt64("userId"), queryInt("page"), queryInt("pageSize")),
		permissionRoute("iam.api-tokens.create", http.MethodPost, appconstants.APIPath("orgs", ":orgId", "api-tokens"), "IAM API Token", "签发 IAM API Token", "api_token:create", jsonType[iamhandler.CreateAPITokenRequest](), jsonType[iamservice.CreateAPITokenResult](), pathID("orgId")),
		permissionRoute("iam.api-tokens.revoke", http.MethodDelete, appconstants.APIPath("orgs", ":orgId", "api-tokens", ":tokenId"), "IAM API Token", "撤销 IAM API Token", "api_token:revoke", nil, jsonType[BooleanResult](), pathID("orgId"), pathID("tokenId")),
		permissionRoute("iam.sessions.list", http.MethodGet, appconstants.APIPath("orgs", ":orgId", "sessions"), "IAM 会话", "分页查询会话", "session:read", nil, jsonType[iamservice.SessionPage](), append([]RouteParam{pathID("orgId")}, sessionListParams()...)...),
		permissionRoute("iam.sessions.revoke", http.MethodDelete, appconstants.APIPath("orgs", ":orgId", "sessions", ":sessionId"), "IAM 会话", "撤销当前组织中的会话", "session:revoke", nil, jsonType[BooleanResult](), pathID("orgId"), pathID("sessionId")),
		permissionRoute("iam.audit-logs.list", http.MethodGet, appconstants.APIPath("orgs", ":orgId", "audit-logs"), "IAM 审计", "查询当前组织审计日志", "audit:read", nil, jsonType[[]iammodel.AuditLog](), pathID("orgId"), queryParam("action", "string"), queryInt64("userId"), queryInt("limit"), queryInt64("cursor"), queryDateTime("from"), queryDateTime("to")),

		platformPermissionRoute("plugins.list", http.MethodGet, appconstants.APIPath("plugins"), "插件", "列出插件", "plugin:read", nil, nil),
		platformPermissionRoute("plugins.get", http.MethodGet, appconstants.APIPath("plugins", ":pluginId"), "插件", "读取插件", "plugin:read", nil, nil, pathString("pluginId")),
		platformPermissionRoute("plugins.health", http.MethodGet, appconstants.APIPath("plugins", ":pluginId", "health"), "插件", "检查插件健康状态", "plugin:read", nil, nil, pathString("pluginId")),
		platformPermissionRoute("plugins.capabilities", http.MethodGet, appconstants.APIPath("plugins", ":pluginId", "capabilities"), "插件", "查看远程插件能力", "plugin:read", nil, nil, pathString("pluginId")),
	}
	contracts = append(contracts, systemRouteContracts()...)
	return contracts
}

func MainHTTPContracts() []RouteContract {
	contracts := mainHTTPContracts()
	out := make([]RouteContract, len(contracts))
	copy(out, contracts)
	return out
}

func routeContractByID(id string) (RouteContract, bool) {
	for _, contract := range mainHTTPContracts() {
		if contract.ID == id {
			return contract, true
		}
	}
	return RouteContract{}, false
}

func mustRouteContract(id string) RouteContract {
	contract, ok := routeContractByID(id)
	if !ok {
		panic("missing route contract: " + id)
	}
	return contract
}

func routeSpecFor(id string, handler ports.HTTPHandlerFunc) routeSpec {
	return routeSpec{Contract: mustRouteContract(id), Handler: handler}
}

func routeContractsFromSpecs(specs []routeSpec) []RouteContract {
	contracts := make([]RouteContract, 0, len(specs))
	for _, spec := range specs {
		contracts = append(contracts, spec.Contract)
	}
	return contracts
}

func organizationListParams() []RouteParam {
	return []RouteParam{queryParam("keyword", "string"), queryParam("code", "string"), queryParam("name", "string"), queryParam("status", "string"), queryInt("page"), queryInt("pageSize"), queryParam("orderKey", "string"), queryBool("desc")}
}

func userListParams() []RouteParam {
	return []RouteParam{queryParam("keyword", "string"), queryParam("username", "string"), queryParam("displayName", "string"), queryParam("email", "string"), queryParam("roleCode", "string"), queryParam("status", "string"), queryInt("page"), queryInt("pageSize"), queryParam("orderKey", "string"), queryBool("desc")}
}

func sessionListParams() []RouteParam {
	return []RouteParam{queryParam("scope", "string"), queryParam("keyword", "string"), queryInt64("userId"), queryParam("ipAddress", "string"), queryParam("status", "string"), queryInt("page"), queryInt("pageSize"), queryParam("orderKey", "string"), queryBool("desc")}
}

func systemRouteContracts() []RouteContract {
	return []RouteContract{
		publicRoute("system.public-settings", http.MethodGet, appconstants.APIPath("system", "public-settings"), "System", "查询公开运行设置", nil, jsonType[systemmodel.PublicSettings]()),
		authRoute("system.menus", http.MethodGet, appconstants.APIPath("system", "menus"), "System", "查询当前用户可见菜单", nil, jsonType[[]systemmodel.MenuGroup]()),
		platformPermissionRoute("system.config.get", http.MethodGet, appconstants.APIPath("system", "config"), "System", "查询脱敏运行配置快照", "config:read", nil, jsonType[systemmodel.ConfigSnapshot]()),
		platformPermissionRoute("system.config.update", http.MethodPatch, appconstants.APIPath("system", "config"), "System", "更新运行配置快照", "config:update", jsonType[systemhandler.UpdateConfigRequest](), jsonType[systemmodel.ConfigSnapshot]()),
		platformPermissionRoute("system.server-info", http.MethodGet, appconstants.APIPath("system", "server-info"), "System", "查询服务运行状态", "server:read", nil, jsonType[systemmodel.ServerInfo]()),
		platformPermissionRoute("system.server-metrics.history", http.MethodGet, appconstants.APIPath("system", "server-metrics", "history"), "System", "查询服务运行指标短窗口历史", "server:read", nil, jsonType[systemmodel.ServerMetricsHistory]()),
		platformPermissionRoute("system.traffic-hijack.overview", http.MethodGet, appconstants.APIPath("system", "traffic-hijack", "overview"), "System", "查询流量劫持监控概览", "traffic_hijack:read", nil, jsonType[systemmodel.TrafficHijackOverview]()),
		platformPermissionRoute("system.traffic-hijack.targets.list", http.MethodGet, appconstants.APIPath("system", "traffic-hijack", "targets"), "System", "查询流量探针目标", "traffic_hijack:read", nil, jsonType[[]systemmodel.TrafficProbeTarget]()),
		platformPermissionRoute("system.traffic-hijack.targets.create", http.MethodPost, appconstants.APIPath("system", "traffic-hijack", "targets"), "System", "创建流量探针目标", "traffic_hijack:update", jsonType[systemhandler.CreateTrafficProbeTargetRequest](), jsonType[systemmodel.TrafficProbeTarget]()),
		platformPermissionRoute("system.traffic-hijack.targets.update", http.MethodPatch, appconstants.APIPath("system", "traffic-hijack", "targets", ":targetId"), "System", "更新流量探针目标", "traffic_hijack:update", jsonType[systemhandler.UpdateTrafficProbeTargetRequest](), jsonType[systemmodel.TrafficProbeTarget](), pathID("targetId")),
		platformPermissionRoute("system.traffic-hijack.targets.delete", http.MethodDelete, appconstants.APIPath("system", "traffic-hijack", "targets", ":targetId"), "System", "删除流量探针目标", "traffic_hijack:delete", nil, jsonType[BooleanResult](), pathID("targetId")),
		platformPermissionRoute("system.traffic-hijack.targets.probe", http.MethodPost, appconstants.APIPath("system", "traffic-hijack", "targets", ":targetId", "probe"), "System", "立即执行流量探针", "traffic_hijack:update", nil, jsonType[systemmodel.TrafficProbeResult](), pathID("targetId")),
		platformPermissionRoute("system.traffic-hijack.results", http.MethodGet, appconstants.APIPath("system", "traffic-hijack", "results"), "System", "查询流量探针结果", "traffic_hijack:read", nil, jsonType[systemmodel.TrafficProbeResultPage](), queryInt64("targetId"), queryInt("limit"), queryInt64("cursor")),
		platformPermissionRoute("system.traffic-hijack.events", http.MethodGet, appconstants.APIPath("system", "traffic-hijack", "events"), "System", "查询流量劫持事件", "traffic_hijack:read", nil, jsonType[systemmodel.TrafficHijackEventPage](), queryInt64("targetId"), queryParam("severity", "string"), queryParam("state", "string"), queryInt("page"), queryInt("pageSize")),
		platformPermissionRoute("system.traffic-hijack.events.resolve", http.MethodPost, appconstants.APIPath("system", "traffic-hijack", "events", ":eventId", "resolve"), "System", "确认流量劫持事件已恢复", "traffic_hijack:update", nil, jsonType[systemmodel.TrafficHijackEvent](), pathID("eventId")),
		platformPermissionRoute("system.traffic-hijack.stream", http.MethodGet, appconstants.APIPath("system", "traffic-hijack", "stream"), "System", "订阅流量劫持监控事件流", "traffic_hijack:read", nil, nil),
		platformPermissionRoute("system.apis", http.MethodGet, appconstants.APIPath("system", "apis"), "System", "查询已注册 HTTP API 目录", "permission:read", nil, jsonType[[]systemmodel.APIGroup]()),
		platformPermissionRoute("system.apis.sync", http.MethodPost, appconstants.APIPath("system", "apis", "sync"), "System", "同步 HTTP API 目录", "permission:read", nil, jsonType[systemmodel.APISyncResult]()),
		platformPermissionRoute("system.apis.permissions.sync", http.MethodPost, appconstants.APIPath("system", "apis", "permissions", "sync"), "System", "同步路由权限到 IAM 权限字典", "permission:sync", nil, jsonType[systemmodel.PermissionSyncResult]()),
		platformPermissionRoute("system.operation-records.list", http.MethodGet, appconstants.APIPath("system", "operation-records"), "System", "分页查询后台操作记录", "operation:read", nil, jsonType[systemmodel.OperationRecordPage](), operationRecordParams()...),
		platformPermissionRoute("system.operation-records.delete", http.MethodDelete, appconstants.APIPath("system", "operation-records"), "System", "按 ID 删除操作记录", "operation:delete", jsonType[systemhandler.DeleteOperationRecordsRequest](), jsonType[BooleanResult]()),
		platformPermissionRoute("system.versions.list", http.MethodGet, appconstants.APIPath("system", "versions"), "System", "分页查询系统版本发布包", "version:read", nil, jsonType[systemmodel.VersionPage](), versionListParams()...),
		platformPermissionRoute("system.versions.delete-batch", http.MethodDelete, appconstants.APIPath("system", "versions"), "System", "按 ID 批量删除系统版本发布包", "version:delete", jsonType[systemhandler.DeleteVersionsRequest](), jsonType[BooleanResult]()),
		platformPermissionRoute("system.versions.sources", http.MethodGet, appconstants.APIPath("system", "versions", "sources"), "System", "查询版本发布包可选来源", "version:read", nil, jsonType[systemmodel.VersionSourceCatalog]()),
		platformPermissionRoute("system.versions.export", http.MethodPost, appconstants.APIPath("system", "versions", "export"), "System", "创建系统版本发布包", "version:create", jsonType[systemhandler.ExportVersionRequest](), jsonType[systemmodel.VersionDetail]()),
		platformPermissionRoute("system.versions.import", http.MethodPost, appconstants.APIPath("system", "versions", "import"), "System", "导入系统版本发布包", "version:import", jsonType[systemhandler.ImportVersionRequest](), jsonType[systemmodel.VersionImportResult]()),
		platformPermissionRoute("system.versions.get", http.MethodGet, appconstants.APIPath("system", "versions", ":versionId"), "System", "查询系统版本发布包详情", "version:read", nil, jsonType[systemmodel.VersionDetail](), pathID("versionId")),
		platformPermissionRoute("system.versions.delete", http.MethodDelete, appconstants.APIPath("system", "versions", ":versionId"), "System", "删除系统版本发布包", "version:delete", nil, jsonType[BooleanResult](), pathID("versionId")),
		binaryPlatformPermissionRoute("system.versions.download", http.MethodGet, appconstants.APIPath("system", "versions", ":versionId", "download"), "System", "下载系统版本发布包 JSON", "version:download", pathID("versionId")),
		platformPermissionRoute("system.media.categories", http.MethodGet, appconstants.APIPath("system", "media", "categories"), "System", "查询媒体分类树", "media:read", nil, jsonType[systemmodel.MediaCategoryCatalog]()),
		platformPermissionRoute("system.media.categories.upsert", http.MethodPost, appconstants.APIPath("system", "media", "categories"), "System", "创建或更新媒体分类", "media:update", jsonType[systemhandler.UpsertMediaCategoryRequest](), jsonType[systemmodel.MediaCategory]()),
		platformPermissionRoute("system.media.categories.delete", http.MethodDelete, appconstants.APIPath("system", "media", "categories", ":categoryId"), "System", "删除空媒体分类", "media:update", nil, jsonType[BooleanResult](), pathID("categoryId")),
		platformPermissionRoute("system.media.assets", http.MethodGet, appconstants.SystemMediaAssetsAPIPath, "System", "分页查询媒体资源", "media:read", nil, jsonType[systemmodel.MediaAssetPage](), queryInt64("categoryId"), queryParam("keyword", "string"), queryInt("page"), queryInt("pageSize")),
		multipartPlatformPermissionRoute("system.media.assets.upload", http.MethodPost, appconstants.APIPath("system", "media", "assets", "upload"), "System", "上传本地媒体资源", "media:upload", jsonType[systemmodel.MediaAsset](), multipartParam("file", "string", true), multipartParam("categoryId", "integer", false)),
		platformPermissionRoute("system.media.assets.resumable.check", http.MethodPost, appconstants.APIPath("system", "media", "assets", "resumable", "check"), "System", "检查或创建媒体断点上传会话", "media:upload", jsonType[systemhandler.CheckMediaResumableUploadRequest](), jsonType[systemmodel.MediaResumableCheckResult]()),
		multipartPlatformPermissionRoute("system.media.assets.resumable.chunks", http.MethodPost, appconstants.APIPath("system", "media", "assets", "resumable", "chunks"), "System", "上传媒体断点分片", "media:upload", jsonType[systemmodel.MediaResumableChunkResult](), multipartParam("file", "string", true), multipartParam("sessionId", "integer", true), multipartParam("fileHash", "string", true), multipartParam("fileName", "string", true), multipartParam("chunkIndex", "integer", true), multipartParam("chunkTotal", "integer", true), multipartParam("chunkHash", "string", true)),
		platformPermissionRoute("system.media.assets.resumable.complete", http.MethodPost, appconstants.APIPath("system", "media", "assets", "resumable", "complete"), "System", "完成媒体断点上传", "media:upload", jsonType[systemhandler.MediaResumableSessionRequest](), jsonType[systemmodel.MediaResumableCompleteResult]()),
		platformPermissionRoute("system.media.assets.resumable.abort", http.MethodPost, appconstants.APIPath("system", "media", "assets", "resumable", "abort"), "System", "中止媒体断点上传", "media:upload", jsonType[systemhandler.MediaResumableSessionRequest](), jsonType[systemmodel.MediaResumableAbortResult]()),
		platformPermissionRoute("system.media.assets.import-url", http.MethodPost, appconstants.APIPath("system", "media", "assets", "import-url"), "System", "导入外链媒体资源", "media:import", jsonType[systemhandler.ImportMediaURLsRequest](), jsonType[systemmodel.MediaURLImportResult]()),
		platformPermissionRoute("system.media.assets.update", http.MethodPatch, appconstants.APIPath("system", "media", "assets", ":assetId"), "System", "更新媒体显示名称", "media:update", jsonType[systemhandler.UpdateMediaAssetRequest](), jsonType[systemmodel.MediaAsset](), pathID("assetId")),
		platformPermissionRoute("system.media.assets.delete", http.MethodDelete, appconstants.APIPath("system", "media", "assets", ":assetId"), "System", "删除媒体资源", "media:delete", nil, jsonType[BooleanResult](), pathID("assetId")),
		binaryPlatformPermissionRoute("system.media.assets.download", http.MethodGet, appconstants.APIPath("system", "media", "assets", ":assetId", "download"), "System", "下载本地媒体资源", "media:download", pathID("assetId")),
		platformPermissionRoute("system.parameters.list", http.MethodGet, appconstants.APIPath("system", "parameters"), "System", "分页查询系统参数", "parameter:read", nil, jsonType[systemmodel.ParameterPage](), parameterListParams()...),
		platformPermissionRoute("system.parameters.create", http.MethodPost, appconstants.APIPath("system", "parameters"), "System", "创建系统参数", "parameter:create", jsonType[systemhandler.CreateParameterRequest](), jsonType[systemmodel.Parameter]()),
		platformPermissionRoute("system.parameters.delete-batch", http.MethodDelete, appconstants.APIPath("system", "parameters"), "System", "按 ID 批量删除系统参数", "parameter:delete", jsonType[systemhandler.DeleteParametersRequest](), jsonType[BooleanResult]()),
		platformPermissionRoute("system.parameters.value", http.MethodGet, appconstants.APIPath("system", "parameters", "value"), "System", "按参数键查询系统参数", "parameter:read", nil, jsonType[systemmodel.Parameter](), queryParam("key", "string")),
		platformPermissionRoute("system.parameters.get", http.MethodGet, appconstants.APIPath("system", "parameters", ":parameterId"), "System", "按 ID 查询系统参数", "parameter:read", nil, jsonType[systemmodel.Parameter](), pathID("parameterId")),
		platformPermissionRoute("system.parameters.update", http.MethodPatch, appconstants.APIPath("system", "parameters", ":parameterId"), "System", "更新系统参数", "parameter:update", jsonType[systemhandler.UpdateParameterRequest](), jsonType[systemmodel.Parameter](), pathID("parameterId")),
		platformPermissionRoute("system.parameters.delete", http.MethodDelete, appconstants.APIPath("system", "parameters", ":parameterId"), "System", "删除系统参数", "parameter:delete", nil, jsonType[BooleanResult](), pathID("parameterId")),
		platformPermissionRoute("system.dictionaries.list", http.MethodGet, appconstants.APIPath("system", "dictionaries"), "System", "查询系统字典", "dictionary:read", nil, jsonType[systemmodel.DictionaryCatalog]()),
		platformPermissionRoute("system.dictionaries.create", http.MethodPost, appconstants.APIPath("system", "dictionaries"), "System", "创建系统字典", "dictionary:create", jsonType[systemhandler.CreateDictionaryRequest](), jsonType[systemmodel.Dictionary]()),
		platformPermissionRoute("system.dictionaries.update", http.MethodPatch, appconstants.APIPath("system", "dictionaries", ":dictionaryId"), "System", "更新系统字典", "dictionary:update", jsonType[systemhandler.UpdateDictionaryRequest](), jsonType[systemmodel.Dictionary](), pathID("dictionaryId")),
		platformPermissionRoute("system.dictionaries.delete", http.MethodDelete, appconstants.APIPath("system", "dictionaries", ":dictionaryId"), "System", "删除系统字典", "dictionary:delete", nil, jsonType[BooleanResult](), pathID("dictionaryId")),
		platformPermissionRoute("system.dictionary-items.create", http.MethodPost, appconstants.APIPath("system", "dictionaries", ":dictionaryId", "items"), "System", "创建系统字典项", "dictionary:update", jsonType[systemhandler.CreateDictionaryItemRequest](), jsonType[systemmodel.DictionaryItem](), pathID("dictionaryId")),
		platformPermissionRoute("system.dictionary-items.update", http.MethodPatch, appconstants.APIPath("system", "dictionary-items", ":itemId"), "System", "更新系统字典项", "dictionary:update", jsonType[systemhandler.UpdateDictionaryItemRequest](), jsonType[systemmodel.DictionaryItem](), pathID("itemId")),
		platformPermissionRoute("system.dictionary-items.delete", http.MethodDelete, appconstants.APIPath("system", "dictionary-items", ":itemId"), "System", "删除系统字典项", "dictionary:delete", nil, jsonType[BooleanResult](), pathID("itemId")),
	}
}

func operationRecordParams() []RouteParam {
	return []RouteParam{queryParam("method", "string"), queryParam("path", "string"), queryInt("status"), queryInt("page"), queryInt("pageSize")}
}

func versionListParams() []RouteParam {
	return []RouteParam{queryParam("versionName", "string"), queryParam("versionCode", "string"), queryDateTime("startCreatedAt"), queryDateTime("endCreatedAt"), queryInt("page"), queryInt("pageSize")}
}

func parameterListParams() []RouteParam {
	return []RouteParam{queryParam("keyword", "string"), queryParam("name", "string"), queryParam("key", "string"), queryDateTime("startCreatedAt"), queryDateTime("endCreatedAt"), queryInt("page"), queryInt("pageSize")}
}

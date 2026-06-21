use std::collections::{BTreeMap, BTreeSet};

use serde::{Deserialize, Serialize};
use serde_json::{Value, json};

use crate::config::Settings;

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct RouteContract {
    pub id: String,
    pub method: String,
    pub path: String,
    pub tag: String,
    pub summary: String,
    pub access: String,
    pub permission: Option<String>,
    pub scope: String,
    pub product_code: String,
    pub request_schema: Option<String>,
    pub response_schema: Option<String>,
    pub query_parameters: Vec<QueryParameterContract>,
    pub include_catalog: bool,
    pub include_openapi: bool,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct QueryParameterContract {
    pub name: String,
    pub schema_type: String,
    pub required: bool,
    pub description: String,
}

impl RouteContract {
    pub fn axum_path(&self) -> String {
        self.path.clone()
    }

    fn with_query_parameters(mut self, parameters: Vec<QueryParameterContract>) -> Self {
        self.query_parameters = parameters;
        self
    }
}

pub fn should_record_operation(path: &str) -> bool {
    path.starts_with("/api/v1/") && path != "/api/v1/system/public-settings"
}

pub fn contracts(settings: &Settings) -> Vec<RouteContract> {
    let product = settings.app.product_code.clone();
    vec![
        public(
            "probe.health",
            "GET",
            "/health",
            "Probe",
            "存活探针",
            &product,
            None,
            None,
            false,
        ),
        public(
            "probe.ready",
            "GET",
            "/ready",
            "Probe",
            "就绪探针",
            &product,
            None,
            None,
            false,
        ),
        public(
            "openapi.yaml",
            "GET",
            "/openapi.yaml",
            "OpenAPI",
            "读取 OpenAPI 契约",
            &product,
            None,
            None,
            false,
        ),
        public(
            "setup.status",
            "GET",
            "/api/v1/setup/status",
            "Setup",
            "查询首次安装状态",
            &product,
            None,
            Some("SetupStatus"),
            true,
        ),
        public(
            "setup.schema",
            "GET",
            "/api/v1/setup/schema",
            "Setup",
            "查询首次安装 schema",
            &product,
            None,
            Some("SetupSchema"),
            true,
        ),
        public(
            "setup.config-checks",
            "GET",
            "/api/v1/setup/config-checks",
            "Setup",
            "查询初始化配置检测结果",
            &product,
            None,
            Some("SetupConfigCheckSummary"),
            true,
        ),
        public(
            "setup.run.list",
            "GET",
            "/api/v1/setup/runs",
            "Setup",
            "列出初始化运行",
            &product,
            None,
            Some("SetupRun[]"),
            true,
        ),
        public(
            "setup.run.create",
            "POST",
            "/api/v1/setup/runs",
            "Setup",
            "创建初始化运行",
            &product,
            Some("CreateSetupRunRequest"),
            Some("SetupRun"),
            true,
        ),
        public(
            "setup.run.logs",
            "GET",
            "/api/v1/setup/runs/{runId}/logs",
            "Setup",
            "查询初始化步骤日志",
            &product,
            None,
            Some("SetupStepLog[]"),
            true,
        ),
        public(
            "setup.complete",
            "POST",
            "/api/v1/setup/complete",
            "Setup",
            "完成初始化",
            &product,
            Some("CompleteSetupRequest"),
            Some("CompleteSetupResult"),
            true,
        ),
        public(
            "iam.setup.status",
            "GET",
            "/api/v1/auth/setup/status",
            "IAM",
            "查询首个管理员状态",
            &product,
            None,
            Some("IamSetupStatus"),
            true,
        ),
        public(
            "iam.initial-admin",
            "POST",
            "/api/v1/auth/setup/initial-admin",
            "IAM",
            "创建首个管理员并建立会话",
            &product,
            Some("InitialAdminRequest"),
            Some("SessionSnapshot"),
            true,
        ),
        public(
            "iam.login",
            "POST",
            "/api/v1/auth/login",
            "IAM",
            "登录并建立 HttpOnly Cookie 会话",
            &product,
            Some("LoginRequest"),
            Some("SessionSnapshot"),
            true,
        ),
        public(
            "iam.register",
            "POST",
            "/api/v1/auth/register",
            "IAM",
            "自助注册租户账号并创建邮箱验证通知",
            &product,
            Some("RegisterRequest"),
            Some("NotificationDelivery"),
            true,
        ),
        auth(
            "iam.refresh",
            "POST",
            "/api/v1/auth/refresh",
            "IAM",
            "使用 refresh cookie 轮换会话",
            &product,
            None,
            Some("SessionSnapshot"),
        ),
        public(
            "iam.password.forgot",
            "POST",
            "/api/v1/auth/password/forgot",
            "IAM",
            "创建密码重置通知",
            &product,
            Some("ForgotPasswordRequest"),
            Some("NotificationDelivery"),
            true,
        ),
        public(
            "iam.password.reset",
            "POST",
            "/api/v1/auth/password/reset",
            "IAM",
            "使用重置令牌更新密码",
            &product,
            Some("ResetPasswordRequest"),
            Some("BooleanResult"),
            true,
        ),
        public(
            "iam.email-verification.request",
            "POST",
            "/api/v1/auth/email-verifications",
            "IAM",
            "创建邮箱验证通知",
            &product,
            Some("RequestEmailVerification"),
            Some("NotificationDelivery"),
            true,
        ),
        public(
            "iam.email-verification.confirm",
            "POST",
            "/api/v1/auth/email-verifications/confirm",
            "IAM",
            "确认邮箱验证令牌",
            &product,
            Some("ConfirmEmailVerificationRequest"),
            Some("BooleanResult"),
            true,
        ),
        auth(
            "iam.mfa.setup",
            "POST",
            "/api/v1/auth/mfa/setup",
            "IAM MFA",
            "创建或轮换 TOTP MFA 密钥",
            &product,
            None,
            Some("MfaSetupResult"),
        ),
        auth(
            "iam.mfa.factors.list",
            "GET",
            "/api/v1/auth/mfa/factors",
            "IAM MFA",
            "列出当前账号 TOTP MFA 因子元数据",
            &product,
            None,
            Some("MfaFactorSummary[]"),
        ),
        auth(
            "iam.mfa.verify",
            "POST",
            "/api/v1/auth/mfa/verify",
            "IAM MFA",
            "验证并启用 TOTP MFA",
            &product,
            Some("VerifyMfaRequest"),
            Some("MfaVerifyResult"),
        ),
        auth(
            "iam.mfa.recovery-codes.list",
            "GET",
            "/api/v1/auth/mfa/recovery-codes",
            "IAM MFA",
            "列出当前账号 MFA 恢复码元数据",
            &product,
            None,
            Some("MfaRecoveryCodeSummary[]"),
        ),
        auth(
            "iam.mfa.recovery-codes.rotate",
            "POST",
            "/api/v1/auth/mfa/recovery-codes",
            "IAM MFA",
            "轮换当前账号 MFA 恢复码",
            &product,
            None,
            Some("MfaRecoveryCodesResult"),
        ),
        auth(
            "iam.mfa.revoke",
            "DELETE",
            "/api/v1/auth/mfa/factors/{factorId}",
            "IAM MFA",
            "撤销当前账号 TOTP MFA 因子",
            &product,
            None,
            Some("BooleanResult"),
        ),
        auth(
            "iam.logout",
            "POST",
            "/api/v1/auth/logout",
            "IAM",
            "撤销当前会话",
            &product,
            None,
            Some("LogoutResult"),
        ),
        auth(
            "iam.me.session",
            "GET",
            "/api/v1/me/session",
            "IAM",
            "读取当前会话快照",
            &product,
            None,
            Some("SessionSnapshot"),
        ),
        permission(
            "iam.organizations.list",
            "GET",
            "/api/v1/iam/orgs",
            "IAM Organization",
            "列出组织",
            &product,
            "org:read",
            "platform",
            None,
            Some("OrganizationSummary[]"),
        ),
        permission(
            "iam.org-users.list",
            "GET",
            "/api/v1/iam/orgs/{orgId}/users",
            "IAM User",
            "列出组织用户",
            &product,
            "user:read",
            "tenant",
            None,
            Some("OrganizationUserSummary[]"),
        ),
        permission(
            "iam.org-users.update",
            "PUT",
            "/api/v1/iam/orgs/{orgId}/users/{userId}",
            "IAM User",
            "更新组织用户资料、状态和角色",
            &product,
            "user:write",
            "tenant",
            Some("UpdateOrgUserRequest"),
            Some("OrganizationUserSummary"),
        ),
        permission(
            "iam.org-roles.list",
            "GET",
            "/api/v1/iam/orgs/{orgId}/roles",
            "IAM Role",
            "列出组织角色",
            &product,
            "role:read",
            "tenant",
            None,
            Some("RoleSummary[]"),
        ),
        permission(
            "iam.org-roles.create",
            "POST",
            "/api/v1/iam/orgs/{orgId}/roles",
            "IAM Role",
            "创建组织角色",
            &product,
            "role:write",
            "tenant",
            Some("CreateRoleRequest"),
            Some("RoleSummary"),
        ),
        permission(
            "iam.org-roles.update",
            "PUT",
            "/api/v1/iam/orgs/{orgId}/roles/{roleId}",
            "IAM Role",
            "更新组织角色",
            &product,
            "role:write",
            "tenant",
            Some("UpdateRoleRequest"),
            Some("RoleSummary"),
        ),
        permission(
            "iam.org-roles.delete",
            "DELETE",
            "/api/v1/iam/orgs/{orgId}/roles/{roleId}",
            "IAM Role",
            "删除组织角色",
            &product,
            "role:write",
            "tenant",
            None,
            Some("BooleanResult"),
        ),
        permission(
            "iam.permissions.list",
            "GET",
            "/api/v1/iam/permissions",
            "IAM Permission",
            "列出 IAM 权限",
            &product,
            "permission:read",
            "platform",
            None,
            Some("PermissionSummary[]"),
        ),
        permission(
            "iam.api-tokens.list",
            "GET",
            "/api/v1/orgs/{orgId}/api-tokens",
            "IAM API Token",
            "列出组织 API Token",
            &product,
            "api_token:read",
            "tenant",
            None,
            Some("APITokenSummary[]"),
        ),
        permission(
            "iam.api-tokens.create",
            "POST",
            "/api/v1/orgs/{orgId}/api-tokens",
            "IAM API Token",
            "签发组织 API Token",
            &product,
            "api_token:create",
            "tenant",
            Some("CreateAPITokenRequest"),
            Some("CreateAPITokenResult"),
        ),
        permission(
            "iam.api-tokens.revoke",
            "DELETE",
            "/api/v1/orgs/{orgId}/api-tokens/{tokenId}",
            "IAM API Token",
            "撤销组织 API Token",
            &product,
            "api_token:revoke",
            "tenant",
            None,
            Some("BooleanResult"),
        ),
        permission(
            "iam.invitations.list",
            "GET",
            "/api/v1/orgs/{orgId}/invitations",
            "IAM Invitation",
            "列出组织邀请",
            &product,
            "user:invite",
            "tenant",
            None,
            Some("InvitationSummary[]"),
        ),
        permission(
            "iam.invitations.create",
            "POST",
            "/api/v1/orgs/{orgId}/users/invitations",
            "IAM Invitation",
            "创建组织邀请",
            &product,
            "user:invite",
            "tenant",
            Some("InviteUserRequest"),
            Some("InviteUserResult"),
        ),
        permission(
            "iam.invitations.revoke",
            "DELETE",
            "/api/v1/orgs/{orgId}/invitations/{invitationId}",
            "IAM Invitation",
            "撤销组织邀请",
            &product,
            "user:invite",
            "tenant",
            None,
            Some("BooleanResult"),
        ),
        public(
            "iam.invitations.accept",
            "POST",
            "/api/v1/auth/invitations/accept",
            "IAM Invitation",
            "接受组织邀请并建立会话",
            &product,
            Some("AcceptInvitationRequest"),
            Some("SessionSnapshot"),
            true,
        ),
        public(
            "system.public-settings",
            "GET",
            "/api/v1/system/public-settings",
            "System",
            "读取前端公开运行设置",
            &product,
            None,
            Some("PublicSettings"),
            true,
        ),
        permission(
            "system.menus",
            "GET",
            "/api/v1/system/menus",
            "System",
            "读取系统菜单",
            &product,
            "menu:read",
            "platform",
            None,
            Some("SystemMenuEntry[]"),
        ),
        permission(
            "system.apis",
            "GET",
            "/api/v1/system/apis",
            "System",
            "读取 API 目录",
            &product,
            "permission:read",
            "platform",
            None,
            Some("ApiCatalogGroup[]"),
        ),
        permission(
            "system.operation-records",
            "GET",
            "/api/v1/system/operation-records",
            "System",
            "读取操作记录",
            &product,
            "operation_record:read",
            "platform",
            None,
            Some("OperationRecord[]"),
        )
        .with_query_parameters(vec![
            query_parameter("method", "string", "按 HTTP method 精确过滤"),
            query_parameter(
                "path",
                "string",
                "按 route 模板精确过滤，不包含 query/fragment",
            ),
            query_parameter("status", "integer", "按 HTTP 状态码过滤"),
            query_parameter("actor_user_id", "integer", "按操作者用户 ID 过滤"),
            query_parameter("created_from", "string", "按创建时间起点过滤，RFC3339"),
            query_parameter("created_to", "string", "按创建时间终点过滤，RFC3339"),
            query_parameter("limit", "integer", "返回条数，范围 1..200"),
            query_parameter("offset", "integer", "分页偏移，范围 0..10000"),
        ]),
        permission(
            "system.operation-records.export",
            "GET",
            "/api/v1/system/operation-records/export.csv",
            "System",
            "导出操作记录 CSV",
            &product,
            "operation_record:read",
            "platform",
            None,
            Some("OperationRecordCsv"),
        )
        .with_query_parameters(vec![
            query_parameter("method", "string", "按 HTTP method 精确过滤"),
            query_parameter(
                "path",
                "string",
                "按 route 模板精确过滤，不包含 query/fragment",
            ),
            query_parameter("status", "integer", "按 HTTP 状态码过滤"),
            query_parameter("actor_user_id", "integer", "按操作者用户 ID 过滤"),
            query_parameter("created_from", "string", "按创建时间起点过滤，RFC3339"),
            query_parameter("created_to", "string", "按创建时间终点过滤，RFC3339"),
            query_parameter("limit", "integer", "返回条数，范围 1..200"),
            query_parameter("offset", "integer", "分页偏移，范围 0..10000"),
        ]),
        permission(
            "system.operation-records.summary",
            "GET",
            "/api/v1/system/operation-records/summary",
            "System",
            "读取操作记录审计汇总",
            &product,
            "operation_record:read",
            "platform",
            None,
            Some("OperationRecordSummary"),
        )
        .with_query_parameters(vec![
            query_parameter("method", "string", "按 HTTP method 精确过滤"),
            query_parameter(
                "path",
                "string",
                "按 route 模板精确过滤，不包含 query/fragment",
            ),
            query_parameter("status", "integer", "按 HTTP 状态码过滤"),
            query_parameter("actor_user_id", "integer", "按操作者用户 ID 过滤"),
            query_parameter("created_from", "string", "按创建时间起点过滤，RFC3339"),
            query_parameter("created_to", "string", "按创建时间终点过滤，RFC3339"),
            query_parameter("top_limit", "integer", "top route 返回条数，范围 1..50"),
        ]),
        permission(
            "system.operation-records.prune",
            "POST",
            "/api/v1/system/operation-records/prune",
            "System",
            "按留存策略清理操作记录",
            &product,
            "operation_record:write",
            "platform",
            None,
            Some("OperationRecordRetentionReport"),
        ),
        permission(
            "system.server-status",
            "GET",
            "/api/v1/system/server-status",
            "System",
            "读取服务器状态",
            &product,
            "server:read",
            "platform",
            None,
            Some("ServerStatus"),
        ),
        permission(
            "system.metrics.prometheus",
            "GET",
            "/api/v1/system/metrics/prometheus",
            "System",
            "导出 Prometheus 文本指标",
            &product,
            "server:read",
            "platform",
            None,
            Some("PrometheusMetrics"),
        ),
        permission(
            "system.configs.list",
            "GET",
            "/api/v1/system/configs",
            "System Config",
            "列出系统配置",
            &product,
            "config:read",
            "platform",
            None,
            Some("SystemConfigEntry[]"),
        ),
        permission(
            "system.configs.upsert",
            "PUT",
            "/api/v1/system/configs/{key}",
            "System Config",
            "写入系统配置",
            &product,
            "config:write",
            "platform",
            Some("UpsertSystemConfigRequest"),
            Some("SystemConfigEntry"),
        ),
        permission(
            "system.configs.delete",
            "DELETE",
            "/api/v1/system/configs/{key}",
            "System Config",
            "删除系统配置",
            &product,
            "config:write",
            "platform",
            None,
            Some("DeleteResult"),
        ),
        permission(
            "system.dictionaries.list",
            "GET",
            "/api/v1/system/dictionaries",
            "System Dictionary",
            "列出系统字典",
            &product,
            "dictionary:read",
            "platform",
            None,
            Some("SystemDictionaryEntry[]"),
        ),
        permission(
            "system.dictionaries.upsert",
            "PUT",
            "/api/v1/system/dictionaries/{code}",
            "System Dictionary",
            "写入系统字典",
            &product,
            "dictionary:write",
            "platform",
            Some("UpsertSystemDictionaryRequest"),
            Some("SystemDictionaryEntry"),
        ),
        permission(
            "system.dictionaries.delete",
            "DELETE",
            "/api/v1/system/dictionaries/{code}",
            "System Dictionary",
            "删除系统字典",
            &product,
            "dictionary:write",
            "platform",
            None,
            Some("DeleteResult"),
        ),
        permission(
            "system.parameters.list",
            "GET",
            "/api/v1/system/parameters",
            "System Parameter",
            "列出系统参数",
            &product,
            "parameter:read",
            "platform",
            None,
            Some("SystemParameterEntry[]"),
        ),
        permission(
            "system.parameters.upsert",
            "PUT",
            "/api/v1/system/parameters/{key}",
            "System Parameter",
            "写入系统参数",
            &product,
            "parameter:write",
            "platform",
            Some("UpsertSystemParameterRequest"),
            Some("SystemParameterEntry"),
        ),
        permission(
            "system.parameters.delete",
            "DELETE",
            "/api/v1/system/parameters/{key}",
            "System Parameter",
            "删除系统参数",
            &product,
            "parameter:write",
            "platform",
            None,
            Some("DeleteResult"),
        ),
        permission(
            "system.version-packages.list",
            "GET",
            "/api/v1/system/version-packages",
            "System Version",
            "列出版本包",
            &product,
            "version_package:read",
            "platform",
            None,
            Some("VersionPackageEntry[]"),
        ),
        permission(
            "system.version-packages.create",
            "POST",
            "/api/v1/system/version-packages",
            "System Version",
            "创建版本包",
            &product,
            "version_package:write",
            "platform",
            Some("CreateVersionPackageRequest"),
            Some("VersionPackageEntry"),
        ),
        permission(
            "system.version-packages.releases",
            "GET",
            "/api/v1/system/version-packages/releases",
            "System Version",
            "列出版本发布事件",
            &product,
            "version_package:read",
            "platform",
            None,
            Some("VersionReleaseEventEntry[]"),
        ),
        permission(
            "system.version-packages.publish",
            "POST",
            "/api/v1/system/version-packages/{id}/publish",
            "System Version",
            "发布版本包为 active",
            &product,
            "version_package:write",
            "platform",
            Some("VersionPackageActionRequest"),
            Some("VersionPackageActionResult"),
        ),
        permission(
            "system.version-packages.rollback",
            "POST",
            "/api/v1/system/version-packages/{id}/rollback",
            "System Version",
            "回滚到指定版本包",
            &product,
            "version_package:write",
            "platform",
            Some("VersionPackageActionRequest"),
            Some("VersionPackageActionResult"),
        ),
        permission(
            "system.version-packages.delete",
            "DELETE",
            "/api/v1/system/version-packages/{id}",
            "System Version",
            "删除版本包",
            &product,
            "version_package:write",
            "platform",
            None,
            Some("DeleteResult"),
        ),
        permission(
            "system.media-assets.list",
            "GET",
            "/api/v1/system/media-assets",
            "System Media",
            "列出媒体资产",
            &product,
            "media:read",
            "platform",
            None,
            Some("MediaAssetEntry[]"),
        ),
        permission(
            "system.media-assets.create",
            "POST",
            "/api/v1/system/media-assets",
            "System Media",
            "登记媒体资产",
            &product,
            "media:write",
            "platform",
            Some("CreateMediaAssetRequest"),
            Some("MediaAssetEntry"),
        ),
        permission(
            "system.media-assets.upload",
            "POST",
            "/api/v1/system/media-assets/upload",
            "System Media",
            "上传媒体文件",
            &product,
            "media:write",
            "platform",
            Some("MediaUploadMultipart"),
            Some("MediaAssetEntry"),
        ),
        permission(
            "system.media-assets.delete",
            "DELETE",
            "/api/v1/system/media-assets/{id}",
            "System Media",
            "删除媒体资产",
            &product,
            "media:write",
            "platform",
            None,
            Some("DeleteResult"),
        ),
        permission(
            "system.storage-objects.list",
            "GET",
            "/api/v1/system/storage-objects",
            "System Storage",
            "列出对象存储对象",
            &product,
            "storage_object:read",
            "platform",
            None,
            Some("StorageObjectEntry[]"),
        ),
        permission(
            "system.storage-objects.delete",
            "DELETE",
            "/api/v1/system/storage-objects",
            "System Storage",
            "删除对象存储对象",
            &product,
            "storage_object:delete",
            "platform",
            Some("DeleteStorageObjectRequest"),
            Some("DeleteResult"),
        ),
        permission(
            "system.traffic-probes.targets.list",
            "GET",
            "/api/v1/system/traffic-probes/targets",
            "System Traffic Probe",
            "列出流量探针目标",
            &product,
            "traffic_probe:read",
            "platform",
            None,
            Some("TrafficProbeTargetEntry[]"),
        ),
        permission(
            "system.traffic-probes.targets.create",
            "POST",
            "/api/v1/system/traffic-probes/targets",
            "System Traffic Probe",
            "创建流量探针目标",
            &product,
            "traffic_probe:write",
            "platform",
            Some("CreateTrafficProbeTargetRequest"),
            Some("TrafficProbeTargetEntry"),
        ),
        permission(
            "system.traffic-probes.targets.delete",
            "DELETE",
            "/api/v1/system/traffic-probes/targets/{id}",
            "System Traffic Probe",
            "删除流量探针目标",
            &product,
            "traffic_probe:write",
            "platform",
            None,
            Some("DeleteResult"),
        ),
        permission(
            "system.traffic-probes.targets.run",
            "POST",
            "/api/v1/system/traffic-probes/targets/{id}/run",
            "System Traffic Probe",
            "立即执行流量探针",
            &product,
            "traffic_probe:write",
            "platform",
            None,
            Some("TrafficProbeResultEntry"),
        ),
        permission(
            "system.traffic-probes.results",
            "GET",
            "/api/v1/system/traffic-probes/results",
            "System Traffic Probe",
            "列出流量探针结果",
            &product,
            "traffic_probe:read",
            "platform",
            None,
            Some("TrafficProbeResultEntry[]"),
        ),
        permission(
            "system.traffic-probes.alerts",
            "GET",
            "/api/v1/system/traffic-probes/alerts",
            "System Traffic Probe",
            "列出流量探针告警",
            &product,
            "traffic_probe:read",
            "platform",
            None,
            Some("TrafficProbeAlertEntry[]"),
        ),
        permission(
            "system.traffic-probes.events",
            "GET",
            "/api/v1/system/traffic-probes/events",
            "System Traffic Probe",
            "订阅流量探针告警事件流",
            &product,
            "traffic_probe:read",
            "platform",
            None,
            Some("TrafficProbeEventStream"),
        ),
        permission(
            "system.traffic-probes.alerts.ack",
            "POST",
            "/api/v1/system/traffic-probes/alerts/{id}/ack",
            "System Traffic Probe",
            "确认流量探针告警",
            &product,
            "traffic_probe:write",
            "platform",
            None,
            Some("TrafficProbeAlertActionResult"),
        ),
        permission(
            "system.traffic-probes.alerts.resolve",
            "POST",
            "/api/v1/system/traffic-probes/alerts/{id}/resolve",
            "System Traffic Probe",
            "恢复流量探针告警",
            &product,
            "traffic_probe:write",
            "platform",
            None,
            Some("TrafficProbeAlertActionResult"),
        ),
    ]
}

pub fn openapi_yaml(settings: &Settings) -> serde_yaml::Result<String> {
    serde_yaml::to_string(&openapi_json(settings))
}

pub fn required_permission(settings: &Settings, route_id: &str) -> Option<String> {
    contracts(settings)
        .into_iter()
        .find(|contract| contract.id == route_id)
        .and_then(|contract| contract.permission)
}

pub fn openapi_json(settings: &Settings) -> Value {
    let contracts = contracts(settings);
    let mut paths = serde_json::Map::new();
    for contract in contracts.iter().filter(|contract| contract.include_openapi) {
        let method = contract.method.to_lowercase();
        let mut operation = json!({
            "operationId": contract.id.clone(),
            "tags": [contract.tag.clone()],
            "summary": contract.summary.clone(),
            "x-console-access": contract.access.clone(),
            "x-console-scope": contract.scope.clone(),
            "x-console-permission": contract.permission.clone(),
            "x-console-product-code": contract.product_code.clone(),
            "responses": {
                "200": {
                    "description": "成功",
                    "content": response_content(contract.response_schema.as_deref())
                }
            }
        });
        if let Some(request_schema) = contract.request_schema.as_deref() {
            operation["requestBody"] = request_body(request_schema);
        }
        let mut parameters = path_parameters(&contract.path);
        parameters.extend(query_parameters(&contract.query_parameters));
        if !parameters.is_empty() {
            operation["parameters"] = Value::Array(parameters);
        }
        let path = paths
            .entry(contract.path.clone())
            .or_insert_with(|| json!({}));
        path[method] = operation;
    }

    json!({
        "openapi": "3.1.0",
        "info": {
            "title": "Aoi[葵] API",
            "version": settings.app.version,
            "description": "由 Rust route registry 生成的当前有效 API 契约，禁止手写漂移。"
        },
        "paths": paths,
        "components": {
            "schemas": openapi_components(&contracts)
        }
    })
}

fn request_content_type(request_schema: &str) -> &'static str {
    if request_schema.contains("Multipart") {
        "multipart/form-data"
    } else {
        "application/json"
    }
}

fn request_body(request_schema: &str) -> Value {
    let mut content = serde_json::Map::new();
    content.insert(
        request_content_type(request_schema).into(),
        json!({
            "schema": schema_ref(request_schema)
        }),
    );
    json!({
        "required": true,
        "content": content
    })
}

fn response_content(response_schema_name: Option<&str>) -> Value {
    let mut content = serde_json::Map::new();
    content.insert(
        response_content_type(response_schema_name).into(),
        json!({
            "schema": response_schema(response_schema_name)
        }),
    );
    Value::Object(content)
}

fn response_content_type(response_schema_name: Option<&str>) -> &'static str {
    match response_schema_name {
        Some("TrafficProbeEventStream") => "text/event-stream",
        Some("PrometheusMetrics") => "text/plain; version=0.0.4",
        Some("OperationRecordCsv") => "text/csv; charset=utf-8",
        _ => "application/json",
    }
}

fn response_schema(response_schema: Option<&str>) -> Value {
    response_schema
        .map(schema_ref)
        .unwrap_or_else(|| json!({ "type": "object", "additionalProperties": true }))
}

fn schema_ref(name: &str) -> Value {
    if let Some(item) = name.strip_suffix("[]") {
        json!({
            "type": "array",
            "items": schema_ref(item)
        })
    } else {
        json!({ "$ref": format!("#/components/schemas/{name}") })
    }
}

fn openapi_components(contracts: &[RouteContract]) -> Value {
    let catalog = schema_catalog();
    let mut schemas = serde_json::Map::new();
    for (name, schema) in catalog {
        schemas.insert(name, schema);
    }
    for name in referenced_schema_names(contracts) {
        debug_assert!(
            schemas.contains_key(name.as_str()),
            "route registry references an OpenAPI schema missing from schema_catalog: {name}"
        );
    }
    Value::Object(schemas)
}

fn referenced_schema_names(contracts: &[RouteContract]) -> BTreeSet<String> {
    let mut names = BTreeSet::new();
    for contract in contracts {
        if let Some(name) = &contract.request_schema {
            names.insert(base_schema_name(name).to_owned());
        }
        if let Some(name) = &contract.response_schema {
            names.insert(base_schema_name(name).to_owned());
        }
    }
    names
}

fn base_schema_name(name: &str) -> &str {
    name.strip_suffix("[]").unwrap_or(name)
}

fn schema_catalog() -> BTreeMap<String, Value> {
    let mut schemas = BTreeMap::new();

    insert_schema(
        &mut schemas,
        "AcceptInvitationRequest",
        object(
            [
                ("token", string_schema()),
                ("password", secret_string_schema("邀请接受密码")),
                ("display_name", string_schema()),
            ],
            ["token", "password", "display_name"],
        ),
    );
    insert_schema(
        &mut schemas,
        "APITokenSummary",
        object(
            [
                ("id", int64_schema()),
                ("org_id", int64_schema()),
                ("user_id", int64_schema()),
                ("token_prefix", string_schema()),
                ("status", string_schema()),
                ("expires_at", nullable_string_schema()),
                ("created_at", string_schema()),
                ("revoked_at", nullable_string_schema()),
            ],
            [
                "id",
                "org_id",
                "user_id",
                "token_prefix",
                "status",
                "created_at",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "ApiCatalogGroup",
        object(
            [
                ("tag", string_schema()),
                ("items", array_schema(schema_ref("ApiCatalogEntry"))),
            ],
            ["tag", "items"],
        ),
    );
    insert_schema(
        &mut schemas,
        "ApiCatalogEntry",
        object(
            [
                ("id", string_schema()),
                ("method", string_schema()),
                ("path", string_schema()),
                ("tag", string_schema()),
                ("summary", string_schema()),
                ("access", string_schema()),
                ("permission", nullable_string_schema()),
                ("scope", string_schema()),
                ("product_code", string_schema()),
            ],
            [
                "id",
                "method",
                "path",
                "tag",
                "summary",
                "access",
                "scope",
                "product_code",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "BooleanResult",
        object(
            [
                ("revoked", nullable_bool_schema()),
                ("accepted", nullable_bool_schema()),
                ("reset", nullable_bool_schema()),
                ("verified", nullable_bool_schema()),
                ("deleted", nullable_bool_schema()),
            ],
            [],
        ),
    );
    insert_schema(
        &mut schemas,
        "CompleteSetupRequest",
        object(
            [
                ("confirm", bool_schema()),
                ("run_id", nullable_string_schema()),
            ],
            ["confirm"],
        ),
    );
    insert_schema(
        &mut schemas,
        "CompleteSetupResult",
        object([("completed", bool_schema())], ["completed"]),
    );
    insert_schema(
        &mut schemas,
        "CreateAPITokenRequest",
        object(
            [
                ("expires_in_days", nullable_int64_schema()),
                ("remark", nullable_string_schema()),
            ],
            [],
        ),
    );
    insert_schema(
        &mut schemas,
        "CreateAPITokenResult",
        object(
            [
                ("item", schema_ref("APITokenSummary")),
                (
                    "token",
                    sensitive_string_schema("仅创建时返回一次的 API Token 明文"),
                ),
            ],
            ["item", "token"],
        ),
    );
    insert_schema(
        &mut schemas,
        "CreateMediaAssetRequest",
        object(
            [
                ("category", nullable_string_schema()),
                ("display_name", string_schema()),
                ("storage_key", string_schema()),
                ("mime_type", string_schema()),
                ("size_bytes", int64_schema()),
            ],
            ["display_name", "storage_key", "mime_type", "size_bytes"],
        ),
    );
    insert_schema(
        &mut schemas,
        "DeleteStorageObjectRequest",
        object([("storage_key", string_schema())], ["storage_key"]),
    );
    insert_schema(
        &mut schemas,
        "CreateRoleRequest",
        object(
            [
                ("code", string_schema()),
                ("name", string_schema()),
                ("permission_codes", array_schema(string_schema())),
            ],
            ["code", "name", "permission_codes"],
        ),
    );
    insert_schema(
        &mut schemas,
        "CreateSetupRunRequest",
        object([("reason", nullable_string_schema())], []),
    );
    insert_schema(
        &mut schemas,
        "CreateTrafficProbeTargetRequest",
        object(
            [
                ("name", string_schema()),
                ("url", string_schema()),
                ("expected_status", nullable_int64_schema()),
            ],
            ["name", "url"],
        ),
    );
    insert_schema(
        &mut schemas,
        "CreateVersionPackageRequest",
        object(
            [
                ("version_name", string_schema()),
                ("version_code", string_schema()),
                ("manifest", value_schema()),
            ],
            ["version_name", "version_code", "manifest"],
        ),
    );
    insert_schema(
        &mut schemas,
        "VersionPackageActionRequest",
        object([("reason", nullable_string_schema())], []),
    );
    insert_schema(
        &mut schemas,
        "DeleteResult",
        object([("deleted", bool_schema())], ["deleted"]),
    );
    insert_schema(
        &mut schemas,
        "ForgotPasswordRequest",
        object([("email", string_schema())], ["email"]),
    );
    insert_schema(
        &mut schemas,
        "IamSetupStatus",
        object([("initialized", bool_schema())], ["initialized"]),
    );
    insert_schema(
        &mut schemas,
        "InitialAdminRequest",
        object(
            [
                ("email", string_schema()),
                ("password", secret_string_schema("初始管理员密码")),
                ("display_name", string_schema()),
                ("organization_code", string_schema()),
                ("organization_name", string_schema()),
            ],
            [
                "email",
                "password",
                "display_name",
                "organization_code",
                "organization_name",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "InvitationSummary",
        object(
            [
                ("id", int64_schema()),
                ("org_id", int64_schema()),
                ("email", string_schema()),
                ("role_code", string_schema()),
                ("status", string_schema()),
                ("expires_at", string_schema()),
                ("created_at", string_schema()),
                ("accepted_at", nullable_string_schema()),
                ("revoked_at", nullable_string_schema()),
            ],
            [
                "id",
                "org_id",
                "email",
                "role_code",
                "status",
                "expires_at",
                "created_at",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "InviteUserRequest",
        object(
            [
                ("email", string_schema()),
                ("role_code", nullable_string_schema()),
            ],
            ["email"],
        ),
    );
    insert_schema(
        &mut schemas,
        "InviteUserResult",
        object(
            [
                ("item", schema_ref("InvitationSummary")),
                ("delivery", schema_ref("NotificationDelivery")),
            ],
            ["item", "delivery"],
        ),
    );
    insert_schema(
        &mut schemas,
        "LoginRequest",
        object(
            [
                ("identifier", string_schema()),
                ("password", secret_string_schema("登录密码")),
                ("mfa_code", nullable_string_schema()),
            ],
            ["identifier", "password"],
        ),
    );
    insert_schema(
        &mut schemas,
        "LogoutResult",
        object([("logged_out", bool_schema())], ["logged_out"]),
    );
    insert_schema(
        &mut schemas,
        "MediaAssetEntry",
        object(
            [
                ("id", int64_schema()),
                ("category", nullable_string_schema()),
                ("display_name", string_schema()),
                ("storage_key", string_schema()),
                ("mime_type", string_schema()),
                ("size_bytes", int64_schema()),
                ("created_at", string_schema()),
            ],
            [
                "id",
                "display_name",
                "storage_key",
                "mime_type",
                "size_bytes",
                "created_at",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "MediaUploadMultipart",
        object(
            [
                ("category", nullable_string_schema()),
                ("display_name", string_schema()),
                ("file", binary_schema()),
            ],
            ["file"],
        ),
    );
    insert_schema(
        &mut schemas,
        "StorageObjectEntry",
        object(
            [
                ("storage_key", string_schema()),
                ("size_bytes", int64_schema()),
                ("updated_at", nullable_string_schema()),
                ("e_tag", nullable_string_schema()),
            ],
            ["storage_key", "size_bytes", "updated_at", "e_tag"],
        ),
    );
    insert_schema(
        &mut schemas,
        "MfaFactorSummary",
        object(
            [
                ("id", int64_schema()),
                ("kind", string_schema()),
                ("status", string_schema()),
                ("created_at", string_schema()),
                ("verified_at", nullable_string_schema()),
                ("revoked_at", nullable_string_schema()),
            ],
            ["id", "kind", "status", "created_at"],
        ),
    );
    insert_schema(
        &mut schemas,
        "MfaRecoveryCodesResult",
        object(
            [
                ("items", array_schema(schema_ref("MfaRecoveryCodeSummary"))),
                (
                    "recovery_codes",
                    array_schema(sensitive_string_schema("仅轮换时返回一次的 MFA 恢复码")),
                ),
            ],
            ["items", "recovery_codes"],
        ),
    );
    insert_schema(
        &mut schemas,
        "MfaRecoveryCodeSummary",
        object(
            [
                ("id", int64_schema()),
                ("code_prefix", string_schema()),
                ("status", string_schema()),
                ("created_at", string_schema()),
                ("used_at", nullable_string_schema()),
                ("revoked_at", nullable_string_schema()),
            ],
            ["id", "code_prefix", "status", "created_at"],
        ),
    );
    insert_schema(
        &mut schemas,
        "MfaSetupResult",
        object(
            [
                ("factor", schema_ref("MfaFactorSummary")),
                (
                    "secret",
                    sensitive_string_schema("仅设置时返回一次的 TOTP secret"),
                ),
                (
                    "otpauth_url",
                    sensitive_string_schema("包含 TOTP secret 的一次性 URI"),
                ),
            ],
            ["factor", "secret", "otpauth_url"],
        ),
    );
    insert_schema(
        &mut schemas,
        "MfaVerifyResult",
        object(
            [
                ("verified", bool_schema()),
                (
                    "recovery_codes",
                    array_schema(sensitive_string_schema("仅启用时返回一次的 MFA 恢复码")),
                ),
            ],
            ["verified", "recovery_codes"],
        ),
    );
    insert_schema(
        &mut schemas,
        "NotificationDelivery",
        object(
            [("accepted", bool_schema()), ("channel", string_schema())],
            ["accepted", "channel"],
        ),
    );
    insert_schema(
        &mut schemas,
        "OperationRecord",
        object(
            [
                ("id", int64_schema()),
                ("actor_user_id", nullable_int64_schema()),
                ("method", string_schema()),
                ("path", string_schema()),
                ("status", int64_schema()),
                ("created_at", string_schema()),
            ],
            ["id", "method", "path", "status", "created_at"],
        ),
    );
    insert_schema(
        &mut schemas,
        "OperationRecordCsv",
        json!({
            "type": "string",
            "description": "UTF-8 CSV 文本，列为 id、actor_user_id、method、path、status、created_at"
        }),
    );
    insert_schema(
        &mut schemas,
        "OperationRecordCountBucket",
        object(
            [("key", string_schema()), ("count", int64_schema())],
            ["key", "count"],
        ),
    );
    insert_schema(
        &mut schemas,
        "OperationRecordPathBucket",
        object(
            [
                ("path", string_schema()),
                ("count", int64_schema()),
                ("error_count", int64_schema()),
                ("last_seen_at", nullable_string_schema()),
            ],
            ["path", "count", "error_count"],
        ),
    );
    insert_schema(
        &mut schemas,
        "OperationRecordSummary",
        object(
            [
                ("generated_at", string_schema()),
                ("total_count", int64_schema()),
                ("success_count", int64_schema()),
                ("redirect_count", int64_schema()),
                ("client_error_count", int64_schema()),
                ("server_error_count", int64_schema()),
                ("other_count", int64_schema()),
                ("top_limit", int64_schema()),
                (
                    "by_method",
                    array_schema(schema_ref("OperationRecordCountBucket")),
                ),
                (
                    "by_status_class",
                    array_schema(schema_ref("OperationRecordCountBucket")),
                ),
                (
                    "top_paths",
                    array_schema(schema_ref("OperationRecordPathBucket")),
                ),
            ],
            [
                "generated_at",
                "total_count",
                "success_count",
                "redirect_count",
                "client_error_count",
                "server_error_count",
                "other_count",
                "top_limit",
                "by_method",
                "by_status_class",
                "top_paths",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "OperationRecordRetentionReport",
        object(
            [
                ("retention_days", int64_schema()),
                ("cutoff", string_schema()),
                ("prune_batch_size", int64_schema()),
                ("deleted", int64_schema()),
            ],
            ["retention_days", "cutoff", "prune_batch_size", "deleted"],
        ),
    );
    insert_schema(
        &mut schemas,
        "Organization",
        object(
            [
                ("id", int64_schema()),
                ("code", string_schema()),
                ("name", string_schema()),
                ("scope", string_schema()),
            ],
            ["id", "code", "name", "scope"],
        ),
    );
    insert_schema(
        &mut schemas,
        "OrganizationSummary",
        object(
            [
                ("id", int64_schema()),
                ("code", string_schema()),
                ("name", string_schema()),
                ("scope", string_schema()),
                ("status", string_schema()),
                ("created_at", string_schema()),
            ],
            ["id", "code", "name", "scope", "status", "created_at"],
        ),
    );
    insert_schema(
        &mut schemas,
        "OrganizationUserSummary",
        object(
            [
                ("id", int64_schema()),
                ("email", string_schema()),
                ("display_name", string_schema()),
                ("status", string_schema()),
                ("email_verified_at", nullable_string_schema()),
                ("role_codes", array_schema(string_schema())),
            ],
            ["id", "email", "display_name", "status", "role_codes"],
        ),
    );
    insert_schema(
        &mut schemas,
        "PermissionSummary",
        object(
            [
                ("id", int64_schema()),
                ("product_code", string_schema()),
                ("scope", string_schema()),
                ("code", string_schema()),
                ("name", string_schema()),
            ],
            ["id", "product_code", "scope", "code", "name"],
        ),
    );
    insert_schema(
        &mut schemas,
        "PublicAuthSettings",
        object(
            [
                ("self_signup_enabled", bool_schema()),
                ("session_cookie_name", string_schema()),
                ("refresh_cookie_name", string_schema()),
                ("product_header", string_schema()),
                ("client_type_header", string_schema()),
                ("default_client_type", string_schema()),
                ("csrf_enabled", bool_schema()),
                ("csrf_cookie_name", string_schema()),
                ("csrf_header_name", string_schema()),
            ],
            [
                "self_signup_enabled",
                "session_cookie_name",
                "refresh_cookie_name",
                "product_header",
                "client_type_header",
                "default_client_type",
                "csrf_enabled",
                "csrf_cookie_name",
                "csrf_header_name",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "PublicSettings",
        object(
            [
                ("product_name", string_schema()),
                ("product_code", string_schema()),
                ("default_locale", string_schema()),
                ("supported_locales", array_schema(string_schema())),
                ("auth", schema_ref("PublicAuthSettings")),
            ],
            [
                "product_name",
                "product_code",
                "default_locale",
                "supported_locales",
                "auth",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "RequestEmailVerification",
        object([("email", string_schema())], ["email"]),
    );
    insert_schema(
        &mut schemas,
        "ConfirmEmailVerificationRequest",
        object(
            [("token", secret_string_schema("邮箱验证一次性令牌"))],
            ["token"],
        ),
    );
    insert_schema(
        &mut schemas,
        "RegisterRequest",
        object(
            [
                ("email", string_schema()),
                ("password", secret_string_schema("注册密码")),
                ("display_name", string_schema()),
                ("organization_code", string_schema()),
                ("organization_name", string_schema()),
            ],
            [
                "email",
                "password",
                "display_name",
                "organization_code",
                "organization_name",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "ResetPasswordRequest",
        object(
            [
                ("token", string_schema()),
                ("password", secret_string_schema("新密码")),
            ],
            ["token", "password"],
        ),
    );
    insert_schema(
        &mut schemas,
        "RoleSummary",
        object(
            [
                ("id", int64_schema()),
                ("org_id", nullable_int64_schema()),
                ("code", string_schema()),
                ("name", string_schema()),
                ("scope", string_schema()),
                ("system_builtin", bool_schema()),
                ("permissions", array_schema(string_schema())),
            ],
            [
                "id",
                "code",
                "name",
                "scope",
                "system_builtin",
                "permissions",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "ServerResourceMetrics",
        object(
            [
                ("source", string_schema()),
                ("cpu_usage_percent", number_schema()),
                ("process_cpu_usage_percent", number_schema()),
                ("total_memory_bytes", uint64_schema()),
                ("used_memory_bytes", uint64_schema()),
                ("available_memory_bytes", uint64_schema()),
                ("process_memory_bytes", uint64_schema()),
                ("process_virtual_memory_bytes", uint64_schema()),
                ("total_swap_bytes", uint64_schema()),
                ("used_swap_bytes", uint64_schema()),
                ("total_disk_bytes", uint64_schema()),
                ("used_disk_bytes", uint64_schema()),
                ("available_disk_bytes", uint64_schema()),
                ("disk_count", uint64_schema()),
                ("network_interface_count", uint64_schema()),
                ("network_received_bytes", uint64_schema()),
                ("network_transmitted_bytes", uint64_schema()),
                ("system_uptime_seconds", uint64_schema()),
                ("system_boot_time_seconds", uint64_schema()),
                ("load_average_one", number_schema()),
                ("load_average_five", number_schema()),
                ("load_average_fifteen", number_schema()),
            ],
            [
                "source",
                "cpu_usage_percent",
                "process_cpu_usage_percent",
                "total_memory_bytes",
                "used_memory_bytes",
                "available_memory_bytes",
                "process_memory_bytes",
                "process_virtual_memory_bytes",
                "total_swap_bytes",
                "used_swap_bytes",
                "total_disk_bytes",
                "used_disk_bytes",
                "available_disk_bytes",
                "disk_count",
                "network_interface_count",
                "network_received_bytes",
                "network_transmitted_bytes",
                "system_uptime_seconds",
                "system_boot_time_seconds",
                "load_average_one",
                "load_average_five",
                "load_average_fifteen",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "ServerStatus",
        object(
            [
                ("source", string_schema()),
                ("collected_at", string_schema()),
                ("started_at", string_schema()),
                ("uptime_seconds", int64_schema()),
                ("process_id", int64_schema()),
                ("os", string_schema()),
                ("arch", string_schema()),
                ("available_parallelism", int64_schema()),
                ("product_code", string_schema()),
                ("version", string_schema()),
                ("database_driver", string_schema()),
                ("metrics", schema_ref("ServerResourceMetrics")),
            ],
            [
                "source",
                "collected_at",
                "started_at",
                "uptime_seconds",
                "process_id",
                "os",
                "arch",
                "available_parallelism",
                "product_code",
                "version",
                "database_driver",
                "metrics",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "PrometheusMetrics",
        json!({
            "type": "string",
            "description": "Prometheus text exposition format 0.0.4，内容只来自后端真实采集的 ServerStatus 指标。"
        }),
    );
    insert_schema(
        &mut schemas,
        "SessionSnapshot",
        object(
            [
                ("authenticated", bool_schema()),
                ("user", nullable_schema(schema_ref("User"))),
                ("organization", nullable_schema(schema_ref("Organization"))),
                ("product_code", string_schema()),
                ("client_type", string_schema()),
                ("permissions", array_schema(string_schema())),
                ("mfa_enabled", bool_schema()),
                ("expires_at", nullable_string_schema()),
                ("refresh_expires_at", nullable_string_schema()),
            ],
            [
                "authenticated",
                "product_code",
                "client_type",
                "permissions",
                "mfa_enabled",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "SetupConfigCheck",
        object(
            [
                ("key", string_schema()),
                ("title", string_schema()),
                ("status", string_schema()),
                ("severity", string_schema()),
                ("message", string_schema()),
            ],
            ["key", "title", "status", "severity", "message"],
        ),
    );
    insert_schema(
        &mut schemas,
        "SetupConfigCheckSummary",
        object(
            [
                ("ready", bool_schema()),
                ("checks", array_schema(schema_ref("SetupConfigCheck"))),
            ],
            ["ready", "checks"],
        ),
    );
    insert_schema(
        &mut schemas,
        "SetupFieldSchema",
        object(
            [
                ("key", string_schema()),
                ("label", string_schema()),
                ("kind", string_schema()),
                ("required", bool_schema()),
                ("sensitive", bool_schema()),
            ],
            ["key", "label", "kind", "required", "sensitive"],
        ),
    );
    insert_schema(
        &mut schemas,
        "SetupRun",
        object(
            [
                ("id", string_schema()),
                ("status", string_schema()),
                ("reason", nullable_string_schema()),
                ("created_at", string_schema()),
                ("updated_at", string_schema()),
            ],
            ["id", "status", "created_at", "updated_at"],
        ),
    );
    insert_schema(
        &mut schemas,
        "SetupSchema",
        object(
            [
                ("locale", string_schema()),
                ("steps", array_schema(schema_ref("SetupStepSchema"))),
            ],
            ["locale", "steps"],
        ),
    );
    insert_schema(
        &mut schemas,
        "SetupStatus",
        object(
            [
                ("completed", bool_schema()),
                ("has_initial_admin", bool_schema()),
                (
                    "required_steps",
                    array_schema(schema_ref("SetupStepStatus")),
                ),
            ],
            ["completed", "has_initial_admin", "required_steps"],
        ),
    );
    insert_schema(
        &mut schemas,
        "SetupStepLog",
        object(
            [
                ("step_key", string_schema()),
                ("status", string_schema()),
                ("message", string_schema()),
                ("created_at", string_schema()),
            ],
            ["step_key", "status", "message", "created_at"],
        ),
    );
    insert_schema(
        &mut schemas,
        "SetupStepSchema",
        object(
            [
                ("key", string_schema()),
                ("title", string_schema()),
                ("fields", array_schema(schema_ref("SetupFieldSchema"))),
            ],
            ["key", "title", "fields"],
        ),
    );
    insert_schema(
        &mut schemas,
        "SetupStepStatus",
        object(
            [
                ("key", string_schema()),
                ("title", string_schema()),
                ("status", string_schema()),
            ],
            ["key", "title", "status"],
        ),
    );
    insert_schema(
        &mut schemas,
        "SystemConfigEntry",
        object(
            [
                ("key", string_schema()),
                ("value", value_schema()),
                ("updated_at", string_schema()),
            ],
            ["key", "value", "updated_at"],
        ),
    );
    insert_schema(
        &mut schemas,
        "SystemDictionaryEntry",
        object(
            [
                ("id", int64_schema()),
                ("code", string_schema()),
                ("name", string_schema()),
                ("created_at", string_schema()),
            ],
            ["id", "code", "name", "created_at"],
        ),
    );
    insert_schema(
        &mut schemas,
        "SystemMenuEntry",
        object(
            [
                ("code", string_schema()),
                ("title", string_schema()),
                ("path", string_schema()),
                ("permission", nullable_string_schema()),
                ("scope", string_schema()),
                ("sort_order", int64_schema()),
            ],
            ["code", "title", "path", "scope", "sort_order"],
        ),
    );
    insert_schema(
        &mut schemas,
        "SystemParameterEntry",
        object(
            [
                ("id", int64_schema()),
                ("key", string_schema()),
                ("name", string_schema()),
                ("value", string_schema()),
                ("created_at", string_schema()),
                ("updated_at", string_schema()),
            ],
            ["id", "key", "name", "value", "created_at", "updated_at"],
        ),
    );
    insert_schema(
        &mut schemas,
        "TrafficProbeAlertActionResult",
        object([("updated", bool_schema())], ["updated"]),
    );
    insert_schema(
        &mut schemas,
        "TrafficProbeAlertEntry",
        object(
            [
                ("id", int64_schema()),
                ("target_id", int64_schema()),
                ("result_id", int64_schema()),
                ("severity", string_schema()),
                ("status", string_schema()),
                ("reason", string_schema()),
                ("detail", value_schema()),
                ("opened_at", string_schema()),
                ("acknowledged_at", nullable_string_schema()),
                ("resolved_at", nullable_string_schema()),
            ],
            [
                "id",
                "target_id",
                "result_id",
                "severity",
                "status",
                "reason",
                "detail",
                "opened_at",
                "acknowledged_at",
                "resolved_at",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "TrafficProbeEventError",
        object(
            [
                ("event_type", string_schema()),
                ("generated_at", string_schema()),
                ("message", string_schema()),
            ],
            ["event_type", "generated_at", "message"],
        ),
    );
    insert_schema(
        &mut schemas,
        "TrafficProbeEventSnapshot",
        object(
            [
                ("event_type", string_schema()),
                ("generated_at", string_schema()),
                ("reconnect_after_millis", uint64_schema()),
                ("alerts", array_schema(schema_ref("TrafficProbeAlertEntry"))),
            ],
            [
                "event_type",
                "generated_at",
                "reconnect_after_millis",
                "alerts",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "TrafficProbeEventStream",
        object(
            [
                ("content_type", string_schema()),
                ("snapshot_event", schema_ref("TrafficProbeEventSnapshot")),
                ("error_event", schema_ref("TrafficProbeEventError")),
            ],
            ["content_type", "snapshot_event", "error_event"],
        ),
    );
    insert_schema(
        &mut schemas,
        "TrafficProbeResultEntry",
        object(
            [
                ("id", int64_schema()),
                ("target_id", int64_schema()),
                ("status", string_schema()),
                ("detail", value_schema()),
                ("probed_at", string_schema()),
            ],
            ["id", "target_id", "status", "detail", "probed_at"],
        ),
    );
    insert_schema(
        &mut schemas,
        "TrafficProbeTargetEntry",
        object(
            [
                ("id", int64_schema()),
                ("name", string_schema()),
                ("url", string_schema()),
                ("expected_status", int64_schema()),
                ("status", string_schema()),
                ("created_at", string_schema()),
            ],
            [
                "id",
                "name",
                "url",
                "expected_status",
                "status",
                "created_at",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "UpdateOrgUserRequest",
        object(
            [
                ("display_name", string_schema()),
                ("status", string_schema()),
                ("role_codes", array_schema(string_schema())),
            ],
            ["display_name", "status", "role_codes"],
        ),
    );
    insert_schema(
        &mut schemas,
        "UpdateRoleRequest",
        object(
            [
                ("name", string_schema()),
                ("permission_codes", array_schema(string_schema())),
            ],
            ["name", "permission_codes"],
        ),
    );
    insert_schema(
        &mut schemas,
        "UpsertSystemConfigRequest",
        object([("value", value_schema())], ["value"]),
    );
    insert_schema(
        &mut schemas,
        "UpsertSystemDictionaryRequest",
        object([("name", string_schema())], ["name"]),
    );
    insert_schema(
        &mut schemas,
        "UpsertSystemParameterRequest",
        object(
            [("name", string_schema()), ("value", string_schema())],
            ["name", "value"],
        ),
    );
    insert_schema(
        &mut schemas,
        "User",
        object(
            [
                ("id", int64_schema()),
                ("email", string_schema()),
                ("display_name", string_schema()),
                ("status", string_schema()),
            ],
            ["id", "email", "display_name", "status"],
        ),
    );
    insert_schema(
        &mut schemas,
        "VerifyMfaRequest",
        object([("code", string_schema())], ["code"]),
    );
    insert_schema(
        &mut schemas,
        "VersionPackageEntry",
        object(
            [
                ("id", int64_schema()),
                ("version_name", string_schema()),
                ("version_code", string_schema()),
                ("manifest", value_schema()),
                ("status", string_schema()),
                ("created_at", string_schema()),
                ("published_at", nullable_string_schema()),
                ("retired_at", nullable_string_schema()),
            ],
            [
                "id",
                "version_name",
                "version_code",
                "manifest",
                "status",
                "created_at",
            ],
        ),
    );
    insert_schema(
        &mut schemas,
        "VersionPackageActionResult",
        object(
            [
                ("event_id", int64_schema()),
                ("previous_active_id", nullable_int64_schema()),
                ("package", schema_ref("VersionPackageEntry")),
            ],
            ["event_id", "package"],
        ),
    );
    insert_schema(
        &mut schemas,
        "VersionReleaseEventEntry",
        object(
            [
                ("id", int64_schema()),
                ("package_id", int64_schema()),
                ("previous_active_id", nullable_int64_schema()),
                ("action", string_schema()),
                ("status", string_schema()),
                ("reason", nullable_string_schema()),
                ("created_at", string_schema()),
            ],
            ["id", "package_id", "action", "status", "created_at"],
        ),
    );

    schemas
}

fn insert_schema(schemas: &mut BTreeMap<String, Value>, name: &str, schema: Value) {
    schemas.insert(name.to_owned(), schema);
}

fn object(
    properties: impl IntoIterator<Item = (&'static str, Value)>,
    required: impl IntoIterator<Item = &'static str>,
) -> Value {
    let mut property_map = serde_json::Map::new();
    for (name, schema) in properties {
        property_map.insert(name.to_owned(), schema);
    }
    json!({
        "type": "object",
        "additionalProperties": false,
        "properties": property_map,
        "required": required.into_iter().collect::<Vec<_>>()
    })
}

fn string_schema() -> Value {
    json!({ "type": "string" })
}

fn secret_string_schema(description: &str) -> Value {
    json!({
        "type": "string",
        "format": "password",
        "writeOnly": true,
        "description": description
    })
}

fn sensitive_string_schema(description: &str) -> Value {
    json!({
        "type": "string",
        "x-console-sensitive": true,
        "description": description
    })
}

fn binary_schema() -> Value {
    json!({ "type": "string", "format": "binary" })
}

fn bool_schema() -> Value {
    json!({ "type": "boolean" })
}

fn int64_schema() -> Value {
    json!({ "type": "integer", "format": "int64" })
}

fn uint64_schema() -> Value {
    json!({ "type": "integer", "format": "uint64", "minimum": 0 })
}

fn number_schema() -> Value {
    json!({ "type": "number", "format": "float" })
}

fn array_schema(items: Value) -> Value {
    json!({ "type": "array", "items": items })
}

fn nullable_schema(schema: Value) -> Value {
    json!({ "anyOf": [schema, { "type": "null" }] })
}

fn nullable_string_schema() -> Value {
    nullable_schema(string_schema())
}

fn nullable_int64_schema() -> Value {
    nullable_schema(int64_schema())
}

fn nullable_bool_schema() -> Value {
    nullable_schema(bool_schema())
}

fn value_schema() -> Value {
    json!({
        "description": "任意 JSON 值",
        "additionalProperties": true
    })
}

fn query_parameters(parameters: &[QueryParameterContract]) -> Vec<Value> {
    parameters
        .iter()
        .map(|parameter| {
            json!({
                "name": parameter.name.clone(),
                "in": "query",
                "required": parameter.required,
                "description": parameter.description.clone(),
                "schema": {
                    "type": parameter.schema_type.clone()
                }
            })
        })
        .collect()
}

fn path_parameters(path: &str) -> Vec<Value> {
    let mut parameters = Vec::new();
    let mut remaining = path;
    while let Some(start) = remaining.find('{') {
        let after_start = &remaining[start + 1..];
        let Some(end) = after_start.find('}') else {
            break;
        };
        let name = &after_start[..end];
        if !name.is_empty() {
            parameters.push(json!({
                "name": name,
                "in": "path",
                "required": true,
                "schema": {
                    "type": "string"
                }
            }));
        }
        remaining = &after_start[end + 1..];
    }
    parameters
}

fn query_parameter(name: &str, schema_type: &str, description: &str) -> QueryParameterContract {
    QueryParameterContract {
        name: name.into(),
        schema_type: schema_type.into(),
        required: false,
        description: description.into(),
    }
}

// route registry 的私有构造函数需要完整描述契约元数据；保持显式参数能避免字段被隐式默认吞掉。
#[allow(clippy::too_many_arguments)]
fn public(
    id: &str,
    method: &str,
    path: &str,
    tag: &str,
    summary: &str,
    product_code: &str,
    request_schema: Option<&str>,
    response_schema: Option<&str>,
    include_catalog: bool,
) -> RouteContract {
    route(
        id,
        method,
        path,
        tag,
        summary,
        "public",
        None,
        "platform",
        product_code,
        request_schema,
        response_schema,
        include_catalog,
    )
}

#[allow(clippy::too_many_arguments)]
fn auth(
    id: &str,
    method: &str,
    path: &str,
    tag: &str,
    summary: &str,
    product_code: &str,
    request_schema: Option<&str>,
    response_schema: Option<&str>,
) -> RouteContract {
    route(
        id,
        method,
        path,
        tag,
        summary,
        "authenticated",
        None,
        "tenant",
        product_code,
        request_schema,
        response_schema,
        true,
    )
}

#[allow(clippy::too_many_arguments)]
fn permission(
    id: &str,
    method: &str,
    path: &str,
    tag: &str,
    summary: &str,
    product_code: &str,
    permission: &str,
    scope: &str,
    request_schema: Option<&str>,
    response_schema: Option<&str>,
) -> RouteContract {
    route(
        id,
        method,
        path,
        tag,
        summary,
        "permission",
        Some(permission),
        scope,
        product_code,
        request_schema,
        response_schema,
        true,
    )
}

#[allow(clippy::too_many_arguments)]
fn route(
    id: &str,
    method: &str,
    path: &str,
    tag: &str,
    summary: &str,
    access: &str,
    permission: Option<&str>,
    scope: &str,
    product_code: &str,
    request_schema: Option<&str>,
    response_schema: Option<&str>,
    include_catalog: bool,
) -> RouteContract {
    RouteContract {
        id: id.into(),
        method: method.into(),
        path: path.into(),
        tag: tag.into(),
        summary: summary.into(),
        access: access.into(),
        permission: permission.map(ToOwned::to_owned),
        scope: scope.into(),
        product_code: product_code.into(),
        request_schema: request_schema.map(ToOwned::to_owned),
        response_schema: response_schema.map(ToOwned::to_owned),
        query_parameters: Vec::new(),
        include_catalog,
        include_openapi: true,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn registry_does_not_expose_plugins_or_old_identity() {
        let settings = Settings::default();
        let serialized = serde_json::to_string(&contracts(&settings)).expect("serialize contracts");
        assert!(!serialized.contains("plugin"));
        assert!(!serialized.contains("go-scaffold"));
        assert!(!serialized.contains("aoi-admin"));
    }

    #[test]
    fn openapi_yaml_is_generated_from_route_registry() {
        let settings = Settings::default();
        let yaml = openapi_yaml(&settings).expect("openapi yaml");

        assert!(yaml.contains("title: Aoi[葵] API"));
        assert!(yaml.contains("operationId: system.public-settings"));
        assert!(yaml.contains("/api/v1/setup/status"));
        assert!(yaml.contains("x-console-product-code: console"));
        assert!(yaml.contains("#/components/schemas/InitialAdminRequest"));
        assert!(yaml.contains("InitialAdminRequest:"));
        assert!(yaml.contains("multipart/form-data:"));
        assert!(yaml.contains("MediaUploadMultipart:"));
        assert!(yaml.contains("/api/v1/system/metrics/prometheus"));
        assert!(yaml.contains("text/plain; version=0.0.4:"));
        assert!(yaml.contains("PrometheusMetrics:"));
        assert!(yaml.contains("/api/v1/system/operation-records/export.csv"));
        assert!(yaml.contains("text/csv; charset=utf-8:"));
        assert!(yaml.contains("OperationRecordCsv:"));
        assert!(yaml.contains("/api/v1/system/operation-records/summary"));
        assert!(yaml.contains("operationId: system.operation-records.summary"));
        assert!(yaml.contains("OperationRecordSummary:"));
        assert!(yaml.contains("OperationRecordPathBucket:"));
        assert!(yaml.contains("/api/v1/system/operation-records/prune"));
        assert!(yaml.contains("operationId: system.operation-records.prune"));
        assert!(yaml.contains("OperationRecordRetentionReport:"));
        assert!(yaml.contains("name: orgId"));
        assert!(yaml.contains("in: path"));
        assert!(yaml.contains("x-console-sensitive: true"));
        assert!(!yaml.contains("go-scaffold"));
        assert!(!yaml.contains("aoi-admin"));
    }

    #[test]
    fn docs_openapi_snapshot_matches_route_registry() {
        let settings = Settings::default();
        let generated = openapi_yaml(&settings).expect("openapi yaml");
        let snapshot = include_str!("../../../../../../docs/api/openapi.yaml");

        assert_eq!(
            normalize_snapshot(snapshot),
            normalize_snapshot(&generated),
            "docs/api/openapi.yaml 已与 route registry 生成结果漂移，请运行 \
             cargo run -p app -- openapi --config configs/console.example.yaml --output docs/api/openapi.yaml"
        );
    }

    #[test]
    fn route_schema_references_have_components() {
        let settings = Settings::default();
        let contracts = contracts(&settings);
        let openapi = openapi_json(&settings);
        let schemas = openapi["components"]["schemas"]
            .as_object()
            .expect("components schemas");

        for name in referenced_schema_names(&contracts) {
            assert!(
                schemas.contains_key(name.as_str()),
                "missing OpenAPI component schema for {name}"
            );
        }
        assert_eq!(
            schemas["MediaUploadMultipart"]["properties"]["file"]["format"],
            "binary"
        );
        assert_eq!(
            schemas["CreateAPITokenResult"]["properties"]["token"]["x-console-sensitive"],
            true
        );
    }

    fn normalize_snapshot(value: &str) -> String {
        value.replace("\r\n", "\n").trim_end().to_owned()
    }
}

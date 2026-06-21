# IAM 模块

`internal/modules/iam` 提供本地账号、组织租户、角色权限、会话、API Token、邀请、密码重置、TOTP MFA 和审计能力。IAM service 定义自己的密码、token、授权、TOTP、通知和 repository 接口，具体实现由 `internal/app`、`repository` 和 `infrastructure` 注入。

## 能力

| 能力 | 说明 |
| --- | --- |
| 本地账号 | 用户名和邮箱全局唯一，密码由注入的 `PasswordCrypto` 处理 |
| 首次初始化 | 空用户表时创建平台组织和首个 platform owner |
| 自助注册 | 可选公开注册入口，创建租户组织、tenant owner 用户和登录会话 |
| 组织租户 | access token 绑定单个 `orgId`，切换组织会重新签发 token |
| JWT 会话 | access/refresh token 由 app adapter 注入的 token manager 签发 |
| API Token | 按用户和角色签发 Bearer token，服务端只保存 hash 和显示前缀 |
| 权限 | app adapter 注入授权执行器，service 只依赖 `AuthorizerEnforcer` contract |
| 邀请和重置密码 | service 依赖 `Notifier`；debug/noop 返回调试 token/link，SMTP 由 `internal/app` 适配 `pkg/mail` 投递，投递失败会撤销一次性 token 并返回错误 |
| MFA | service 依赖 TOTP contract，密钥加密后存储 |
| 会话撤销 | 登出、refresh 轮换、密码重置和管理员撤销都会更新会话状态 |
| 审计 | 关键 IAM 动作写入 `iam_audit_logs` |

## 依赖边界

- `service` 定义 `Repository`、`TokenManager`、`AuthorizerEnforcer`、`TOTPProvider`、`Notifier` 等最小接口。
- `repository` 实现 IAM repository contract，持有数据库 executor、事务和 not-found 映射。
- `infrastructure/smtp_notifier.go` 实现 IAM 通知适配，只依赖本地 `MailSender` contract；`internal/app` 将 `pkg/mail` sender 适配进来。
- `internal/app/adapters` 把 token、authorization、TOTP、host 等 `pkg` 实现适配成 IAM service 接口。
- IAM service 不直接导入 `pkg/token`、`pkg/authorization`、`pkg/mfa`、`pkg/crypto`、`pkg/database` 或 `net/smtp`。

## 表结构

goose 迁移位于 `internal/migrations`，主要表包括：

`iam_organizations`、`iam_users`、`iam_memberships`、`iam_roles`、`iam_permissions`、`iam_sessions`、`iam_api_tokens`、`iam_invitations`、`iam_password_resets`、`iam_mfa_factors`、`iam_audit_logs`、`iam_casbin_rules`。

本地默认可以自动迁移；生产应显式执行：

```powershell
go run ./cmd/aoi db migrate status --config=configs/config.yaml
go run ./cmd/aoi db migrate up --config=configs/config.yaml
```

## 初始管理员

```powershell
"change-this-local-password" | go run ./cmd/aoi iam bootstrap-admin --config=configs/config.yaml --org-code=acme --org-name="Acme Corp" --username=admin --email=admin@example.com --password-stdin
```

该命令会初始化平台组织、内置权限、`platform_owner/owner/admin/member` 角色、组织成员关系和 Casbin policy。首次初始化管理员只获得平台组织内的 `platform_owner`；公开注册和普通组织创建只会创建租户组织并授予 tenant `owner`。浏览器首次初始化优先走统一初始化中心 `/api/v1/setup/status` 和 `/api/v1/setup/runs`；旧的 `/api/v1/auth/setup/status` 和 `/api/v1/auth/setup/initial-admin` 保持兼容，并在内部复用同一套初始化编排。

## 路由和权限

| 路由 | 认证 | 用途 |
| --- | --- | --- |
| `GET /api/v1/auth/setup/status` | 否 | 查询是否需要首次初始化 |
| `POST /api/v1/auth/setup/initial-admin` | 否 | 创建首个平台组织 owner |
| `GET /api/v1/setup/status` | 否 | 查询统一初始化状态、步骤和诊断 |
| `POST /api/v1/setup/runs` | 否 | 执行统一初始化流程并在首次 setup 时返回登录令牌 |
| `POST /api/v1/setup/runs/{id}/retry` | 条件 | 重试初始化运行；初始化完成后需 setup token |
| `GET /api/v1/setup/runs/{id}/logs` | 条件 | 查询脱敏后的初始化步骤日志摘要；初始化完成后需 setup token |
| `POST /api/v1/auth/signup` | 否 | 自助注册 |
| `POST /api/v1/auth/email-verifications/{token}/confirm` | 否 | 确认邮箱验证并签发会话 |
| `GET /api/v1/auth/captcha` | 否 | 获取登录验证码 |
| `POST /api/v1/auth/login` | 否 | 登录并签发 token |
| `POST /api/v1/auth/refresh` | 否 | 轮换 refresh token |
| `POST /api/v1/auth/password/forgot` | 否 | 创建重置密码 token |
| `POST /api/v1/auth/password/reset` | 否 | 重置密码并撤销旧会话 |
| `POST /api/v1/invitations/{token}/accept` | 否 | 接受邀请 |
| `POST /api/v1/auth/logout` | 是 | 撤销当前会话 |
| `POST /api/v1/auth/switch-org` | 是 | 切换组织 |
| `POST /api/v1/auth/mfa/setup` | 是 | 创建或轮换 TOTP secret |
| `POST /api/v1/auth/mfa/verify` | 是 | 校验并启用 TOTP |
| `GET /api/v1/me`、`GET /api/v1/me/orgs` | 是 | 当前身份和组织 |
| `/api/v1/orgs/*` | 是 | 组织、用户、邀请、角色、权限、API Token、会话、审计管理 |

组织管理接口按 `productCode + scope + obj:act` 权限保护。平台能力使用 `platform` scope，例如系统配置、插件、API catalog、权限同步和组织列表；租户能力使用 `tenant` scope，例如 `user:update`、`role:create`、`api_token:revoke`、`session:revoke`、`audit:read`。当前预留 `product` scope，但尚未提供产品线业务 API。

## API Token

API Token 用于脚本和外部系统调用受保护接口。创建成功后完整 token 只显示一次；列表只保留 `tokenPrefix`，数据库只保存 hash。

Token 绑定签发时的用户、组织和角色。请求仍走 `Authorization: Bearer <token>`，但不会创建 refresh 会话。如果用户被禁用、成员关系失效、角色移除、token 过期或撤销，请求会被拒绝。

`auth.refresh_token_pepper` 同时参与 refresh token 和 API Token hash；轮换该值会让两类 token 一起失效。

## 配置

关键字段：

- `auth.enabled`
- `auth.registration_mode`
- `auth.email_verification_ttl_seconds`
- `auth.invitation_ttl_seconds`
- `auth.signing_key`
- `auth.refresh_token_pepper`
- `auth.mfa_secret_key`
- `auth.access_token_ttl_seconds`
- `auth.refresh_token_ttl_seconds`
- `auth.login_captcha_enabled`
- `auth.login_max_failures`
- `auth.notification_driver`
- `auth.smtp.*`
- `auth.password_policy.*`
- `auth.casbin_reload_interval_seconds`
- `migration.auto_apply`

生产环境必须通过 secrets 注入 `signing_key`、`refresh_token_pepper`、`mfa_secret_key` 和 SMTP 密码。`notification_driver=debug/noop/local` 会把调试 token/link 暴露在 API 响应中，只适合本地；`notification_driver=smtp` 不在响应中暴露 token，邮件投递失败时会撤销刚创建的邀请或密码重置 token，并返回通知投递失败错误。

`auth.registration_mode` 支持 `disabled`、`direct`、`email_verification` 和 `invite_only`。`direct` 保持注册后立即登录；`email_verification` 会先创建 pending 组织、用户、成员关系和邮箱验证 token，确认链接后才激活并签发会话；`disabled` 与 `invite_only` 会拒绝公开注册，邀请链接流程仍然可用。

邮箱验证注册投递失败时也会清理本次创建的 pending 组织、用户、成员关系和验证 token，避免唯一索引阻塞用户重试。

## 测试入口

```powershell
go test ./internal/modules/iam/... -count=1 -mod=readonly
go test ./internal/app/initapp -count=1 -mod=readonly
```

IAM service 测试通过 `internal/app/testsupport.IAMSQLiteDatabase` 和 `NewIAMDeps` 获取真实 SQLite、迁移、token、RBAC、TOTP 和密码实现。

## 非目标

- 当前不提供 SSO/OIDC/SAML。
- 当前不提供短信 MFA、邮件验证码 MFA 或企业消息网关。
- 当前内置通知只有 debug/noop/local 和 SMTP；外部通知系统应新增 `Notifier` 实现并在 `internal/app` 装配。

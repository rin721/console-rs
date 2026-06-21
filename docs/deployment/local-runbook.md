# 本地运行手册

## 启动

```powershell
cargo run -p app -- serve --config configs/console.example.yaml
```

## 启动新 WebUI

构建后由 Rust 服务直接托管：

```powershell
npm --prefix web/app install
npm --prefix web/app run build
cargo run -p app -- serve --config configs/console.example.yaml
```

浏览器打开 `http://127.0.0.1:8080`。`webui.enabled` 与 `webui.dist_dir` 控制静态托管，默认读取 `web/app/dist`。前端 history 路由会返回 `index.html`，但 `/api/*`、`/health`、`/ready` 和 `/openapi.yaml` 是后端保留路径，不进入 SPA fallback。

## 本地部署 smoke

提交或发布前可以运行本地部署 smoke，证明构建后的 WebUI 能由 Rust 二进制托管，且健康探针、就绪探针、OpenAPI、Setup API、公开运行设置、受保护 Prometheus 指标和 WebUI fallback 都能从同一个真实服务进程返回：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/deployment-smoke.ps1
```

脚本会使用环境变量把服务端口覆盖为 `18080`，并把 SQLite、通知投递和媒体目录放到 `target/deployment-smoke`。启动服务前，它会先对同一临时 SQLite 执行 `database-deploy-preflight.ps1 -ApplyMigrations`，证明部署前显式迁移和 `serve_ready=true` 成立；服务启动后还会运行 `target-environment-acceptance.ps1` 生成 `target/deployment-smoke/target-acceptance-local.json`，再用 `validate-target-acceptance-report.ps1 -AllowLocalHttp` 校验本地报告结构。它不会修改 `configs/console.example.yaml` 或默认 `data/console.sqlite`；结束后会自动停止服务。该 smoke 证明本地部署和目标验收脚本链路，不代表 PostgreSQL/MySQL、外部对象存储、外部版本部署执行器、非本地 HTTPS 入口或生产网络环境已经完成。

## 配置与 secrets 分层

配置加载顺序是主 YAML、可选 secrets YAML、环境变量三层覆盖：

```text
configs/console*.yaml < --secrets/CONSOLE_SECRETS < CONSOLE__... 环境变量
```

本地开发可以只使用 `configs/console.example.yaml`。生产环境请从 `configs/console.production.example.yaml` 起步，把真实密钥放入未提交的 secrets 文件或平台 secret manager 注入的环境变量中：

```powershell
Copy-Item configs/console.secrets.example.yaml configs/console.secrets.yaml
# 编辑 configs/console.secrets.yaml，替换所有 replace-with-* 占位符

cargo run -p app -- check-config `
  --config configs/console.production.example.yaml `
  --secrets configs/console.secrets.yaml
```

`configs/*.secrets.yaml` 已被 `.gitignore` 忽略。生产校验会拒绝 `dev-`、`${...}`、`change-me`、`replace-with-*`、`example` 和过短密钥；因此 secrets 模板不能直接用于部署。环境变量仍然拥有最高优先级，适合由容器平台或 secret manager 注入：

```powershell
$env:CONSOLE_SECRETS = "configs/console.secrets.yaml"
$env:CONSOLE__AUTH__SESSION_SECRET = "<平台注入的强随机会话密钥>"
cargo run -p app -- serve --config configs/console.production.example.yaml
```

服务启动后可查看 Setup 配置检测结果。响应只包含状态和说明，不返回密钥、SMTP 密码或 token 明文：

```powershell
Invoke-RestMethod `
  -Uri http://127.0.0.1:8080/api/v1/setup/config-checks
```

当前本地闭环默认使用 SQLite。配置层也识别 `postgres` 和 `mysql` URL，`serve` 运行时会装配对应数据库连接与 repository set；未声明的外部数据库运行时会显式失败，不会 fallback 到 SQLite。`check-config` 的脱敏输出包含 `database_runtime`，SQLite、PostgreSQL/MySQL 均为 `ready`。`database-plan` 会输出当前 driver 的 runtime 状态、迁移目录、迁移文件大小和 SHA-256 摘要，可用于部署前确认将要使用的迁移方言：

```powershell
cargo run -p app -- database-plan --config configs/console.example.yaml
```

`database-ping` 会使用当前 driver 对应的 sqlx pool 执行 `select 1`，可用于验证 SQLite 文件、PostgreSQL/MySQL 网络和凭据。它只是连接探针；进入 `serve` 前仍必须继续通过显式迁移、schema-check、repository probe 和 `database-preflight`，确保 `serve_ready=true`：

```powershell
cargo run -p app -- database-ping --config configs/console.example.yaml
```

`database-migrate` 会显式执行当前 driver 的迁移脚本，并输出已执行和已跳过文件清单。SQLite 使用 `migrations/*.sql` 与 sqlx migrator；PostgreSQL/MySQL 使用对应方言目录和带 SHA-256 摘要的 `schema_migrations` 记录表，同版本摘要不一致时会失败并要求新建增量迁移。生产部署可关闭启动自动迁移，改由部署流水线先执行该命令或 `database-deploy-preflight.ps1 -ApplyMigrations`：

```powershell
cargo run -p app -- database-migrate --config configs/console.example.yaml
```

迁移后可以用 `database-migration-history` 反查当前数据库已应用迁移记录。SQLite 历史来自 sqlx `_sqlx_migrations`；PostgreSQL/MySQL 历史来自 `schema_migrations`，checksum 为迁移文件 SHA-256：

```powershell
cargo run -p app -- database-migration-history --config configs/console.example.yaml
```

可以用 `database-insert-id-probe` 验证当前 driver 的插入后 ID 获取策略。SQLite/PostgreSQL 应通过 `insert ... returning id`，MySQL 必须在同一连接内通过 `select last_insert_id()`；该探针只验证方言读 ID 语义，仍需配合 Setup/IAM/Notification/System repository probes 证明业务仓储读写路径：

```powershell
cargo run -p app -- database-insert-id-probe --config configs/console.example.yaml
```

迁移后可以用 `database-setup-repository-probe` 验证当前 driver 的 SetupRepository 读写路径。该命令会写入一次性 setup run/log，适合临时 SQLite 或真实 PostgreSQL/MySQL smoke 数据库；它不应被当作生产只读检查使用：

```powershell
cargo run -p app -- database-setup-repository-probe --config configs/console.example.yaml
```

迁移后可以用 `database-iam-repository-probe` 验证当前 driver 的 IamRepository 读写路径。该命令会写入一次性管理员、权限、会话、API Token、pending 通知、邀请、密码重置、邮箱验证、MFA 与审计记录，适合空的 smoke 数据库；它不应被当作生产只读检查使用：

```powershell
cargo run -p app -- database-iam-repository-probe --config configs/console.example.yaml
```

迁移后可以用 `database-notification-repository-probe` 验证当前 driver 的 NotificationRepository 读写路径。该命令会写入一次性 notification outbox/secret 记录，并验证 claim、delivered、retry、final failed、投递 secret 清理和安全 requeue，适合 smoke 数据库；它不应被当作生产只读检查使用：

```powershell
cargo run -p app -- database-notification-repository-probe --config configs/console.example.yaml
```

迁移后可以用 `database-system-repository-probe` 验证当前 driver 的 SystemRepository 读写路径。该命令会写入一次性 catalog、menu、config、dictionary、parameter、operation、version、media 和 traffic probe 记录，适合 smoke 数据库；它不应被当作生产只读检查使用：

```powershell
cargo run -p app -- database-system-repository-probe --config configs/console.example.yaml
```

迁移后可以用 `database-schema-check` 反查当前数据库是否包含平台核心表。报告会同时输出 `repository_runtime` 能力矩阵；SQLite、PostgreSQL/MySQL 表齐备且 Setup/IAM/Notification/System trait 均已接入时 `repository_ready=true`：

```powershell
cargo run -p app -- database-schema-check --config configs/console.example.yaml
```

`database-preflight` 会把迁移计划、连接探针、迁移历史、核心表和 repository readiness 汇总为一个只读报告。SQLite、PostgreSQL/MySQL 迁移完成后都应得到 `serve_ready=true`：

```powershell
cargo run -p app -- database-preflight --config configs/console.example.yaml
```

部署前应使用 `database-deploy-preflight.ps1` 固化同一组数据库门禁。该脚本会执行迁移计划、连接探针、插入 ID 探针，可选择先执行显式迁移，然后强制校验迁移历史、核心表和 `serve_ready`；所有 driver 都必须通过 `serve_ready=true`：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-deploy-preflight.ps1 -ApplyMigrations
```

提交前可以运行数据库 runtime smoke。它会把数据库 URL 覆盖到 `target/database-runtime-smoke/runtime.sqlite`，依次验证迁移计划、连接探针、插入 ID 探针、显式迁移、SetupRepository 探针、IamRepository 探针、NotificationRepository 探针、SystemRepository 探针、迁移历史、核心表反查、运行前聚合预检和第二次迁移 skipped 报告，不会修改默认 `data/console.sqlite`：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-runtime-smoke.ps1
```

有真实 PostgreSQL/MySQL 服务时，也可以运行外部数据库 smoke。该脚本不会启动数据库，只使用传入 URL 验证 plan、ping、insert-id-probe、migrate、setup-repository-probe、iam-repository-probe、notification-repository-probe、system-repository-probe、migration-history、schema-check、preflight 和二次迁移 skipped 报告，并确认外部库得到 `serve_ready=true`；MySQL 会额外确认插入后 ID 读取来自同一连接内 `select last_insert_id()`：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-external-smoke.ps1 `
  -Driver postgres `
  -Url "postgres://postgres:postgres@127.0.0.1:5432/console_ci"

powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-external-smoke.ps1 `
  -Driver mysql `
  -Url "mysql://root:mysql@127.0.0.1:3306/console_ci"
```

外部数据库部署门禁可以在同一真实服务上验证。该命令证明 schema、repository traits、迁移历史和 ID 策略可用于部署前检查，并要求 PostgreSQL/MySQL 作为 `serve` 运行态通过：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-deploy-preflight.ps1 `
  -Driver postgres `
  -Url "postgres://postgres:postgres@127.0.0.1:5432/console_ci" `
  -ApplyMigrations

powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-deploy-preflight.ps1 `
  -Driver mysql `
  -Url "mysql://root:mysql@127.0.0.1:3306/console_ci" `
  -ApplyMigrations
```

仓库已提供 `migrations/postgres`、`migrations/mysql` bootstrap schema、PostgreSQL/MySQL sqlx 连接探针、插入 ID 探针、显式迁移执行器、外部 SetupRepository、外部 IamRepository、外部 NotificationRepository、外部 SystemRepository、迁移历史反查、运行前聚合预检、真实服务 smoke 脚本和 `scripts/database-schema-dialect-scan.ps1`，并在数据库报告中用 `repository_runtime` 逐项暴露 Setup/IAM/Notification/System trait 覆盖状态。`/ready` 通过数据库连接句柄执行 ping，避免 handler 直接依赖具体 pool。

开发联调时可使用 Vite：

```powershell
cd web/app
npm install
npm run dev
```

浏览器打开 `http://127.0.0.1:3002`。Vite 会把 `/api`、`/health`、`/ready` 和 `/openapi.yaml` 代理到 `127.0.0.1:8080`。WebUI 只接当前 Rust 后端已有能力：公开入口读取 runtime/public settings/setup 状态，Setup 状态/schema/配置检测、登录、可选自助注册、账号邀请/找回/邮箱验证流程、会话刷新、IAM 组织/用户/权限基础视图、租户用户资料/状态/角色管理、租户自定义角色管理、API Token 元数据、邀请、TOTP MFA 因子列表/setup/verify/revoke、MFA 恢复码元数据/轮换、System 配置/字典/参数创建与删除、版本包元数据创建/发布/回滚/删除和发布事件查看、媒体上传/元数据登记与删除、流量探针目标创建/运行/删除和结果查看。

## 初始化首个管理员

```powershell
$body = @{
  email = "owner@example.com"
  password = "change-me-123"
  display_name = "平台所有者"
  organization_code = "main"
  organization_name = "主组织"
} | ConvertTo-Json

Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/auth/setup/initial-admin `
  -ContentType "application/json" `
  -Body $body `
  -SessionVariable aoi

Invoke-RestMethod -Uri http://127.0.0.1:8080/api/v1/me/session -WebSession $aoi
```

首个管理员创建后才能显式完成初始化。完成接口会再次读取 Setup 配置检测结果；检测项存在 `error` 时会返回业务冲突，不会写入完成状态。推荐先创建一次初始化运行，再携带 `run_id` 完成，这样后端会在同一事务内更新 run 状态并追加完成日志：

```powershell
$run = Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/setup/runs `
  -ContentType "application/json" `
  -Body (@{ reason = "local-runbook" } | ConvertTo-Json)

Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/setup/complete `
  -ContentType "application/json" `
  -Body (@{ confirm = $true; run_id = $run.id } | ConvertTo-Json)

Invoke-RestMethod -Uri http://127.0.0.1:8080/api/v1/setup/status
Invoke-RestMethod -Uri http://127.0.0.1:8080/api/v1/setup/runs
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/v1/setup/runs/$($run.id)/logs"
```

## 轮换 Cookie 会话

```powershell
Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/auth/refresh `
  -WebSession $aoi

Invoke-RestMethod -Uri http://127.0.0.1:8080/api/v1/me/session -WebSession $aoi
```

`/api/v1/auth/refresh` 只读取 HttpOnly refresh cookie，不接受 URL 或 JSON body 中的 refresh token。成功后服务端会同时下发新的 session cookie 与 refresh cookie，旧 refresh 立即失效；数据库只保存 hash。

## 启用 CSRF 双提交保护

本地示例配置默认关闭 CSRF，方便直接调用初始化和账号接口。生产环境配置校验会强制要求 `auth.csrf.enabled=true` 且 `auth.csrf.secure=true`，避免公网 Cookie 会话缺少跨站请求保护。

本地联调可以临时启用：

```powershell
$env:CONSOLE__AUTH__CSRF__ENABLED = "true"
cargo run -p app -- serve --config configs/console.example.yaml
```

启用后，客户端需要先请求 public settings，服务端会同时下发可读的 CSRF cookie：

```powershell
Invoke-WebRequest `
  -Uri http://127.0.0.1:8080/api/v1/system/public-settings `
  -SessionVariable aoi

$csrf = $aoi.Cookies.GetCookies("http://127.0.0.1:8080")["console_csrf"].Value
```

之后所有非 GET/HEAD/OPTIONS 的 `/api/*` 请求都必须同时带上该 cookie 和配置化 header：

```powershell
Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/auth/refresh `
  -Headers @{ "X-CSRF-Token" = $csrf } `
  -WebSession $aoi
```

WebUI 会自动读取 `/api/v1/system/public-settings`，并在后续非 GET API 请求中发送配置化 CSRF header。CSRF cookie 不加 `HttpOnly` 是为了让前端完成双提交；它不承载身份、权限或服务端会话。

## 签发 API Token

```powershell
$tokenBody = @{ expires_in_days = 7 } | ConvertTo-Json
Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/orgs/1/api-tokens `
  -ContentType "application/json" `
  -Body $tokenBody `
  -WebSession $aoi
```

响应中的 `token` 只显示一次；后续列表只返回 `token_prefix`。

使用 Bearer token 访问租户级接口：

```powershell
$apiToken = "<上一条响应中的 token>"
Invoke-RestMethod `
  -Uri http://127.0.0.1:8080/api/v1/orgs/1/api-tokens `
  -Headers @{ Authorization = "Bearer $apiToken" }
```

Bearer API Token 不继承平台权限，例如 `/api/v1/system/apis` 仍需要 Cookie 会话中的平台级 `permission:read`。

## 查看 IAM 组织、用户、角色和权限

```powershell
Invoke-RestMethod `
  -Uri http://127.0.0.1:8080/api/v1/iam/orgs `
  -WebSession $aoi

Invoke-RestMethod `
  -Uri http://127.0.0.1:8080/api/v1/iam/orgs/1/users `
  -WebSession $aoi

Invoke-RestMethod `
  -Uri http://127.0.0.1:8080/api/v1/iam/orgs/1/roles `
  -WebSession $aoi

Invoke-RestMethod `
  -Uri http://127.0.0.1:8080/api/v1/iam/permissions `
  -WebSession $aoi
```

`/api/v1/iam/orgs` 与 `/api/v1/iam/permissions` 是平台级只读管理面；组织用户和组织角色列表必须使用当前会话所属组织。响应不会返回密码哈希、token、MFA secret 或 pending 通知令牌。

更新组织用户只作用于当前租户，不会修改平台角色：

```powershell
Invoke-RestMethod -Method Put `
  -Uri http://127.0.0.1:8080/api/v1/iam/orgs/$orgId/users/2 `
  -WebSession $session `
  -ContentType application/json `
  -Body (@{
    display_name = "运营成员"
    status = "active"
    role_codes = @("member")
  } | ConvertTo-Json)
```

`status` 只能是 `active` 或 `disabled`；禁用用户后登录、refresh 和旧 Cookie 会话都会失效。服务端会阻止移除或禁用组织最后一个 active owner。

## 管理租户角色

```powershell
$roleBody = @{
  code = "operator"
  name = "运营角色"
  permission_codes = @("user:read", "role:read", "user:invite")
} | ConvertTo-Json

$role = Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/iam/orgs/1/roles `
  -ContentType "application/json" `
  -Body $roleBody `
  -WebSession $aoi

$updateRoleBody = @{
  name = "只读运营"
  permission_codes = @("user:read")
} | ConvertTo-Json

Invoke-RestMethod -Method Put `
  -Uri "http://127.0.0.1:8080/api/v1/iam/orgs/1/roles/$($role.id)" `
  -ContentType "application/json" `
  -Body $updateRoleBody `
  -WebSession $aoi

Invoke-RestMethod -Method Delete `
  -Uri "http://127.0.0.1:8080/api/v1/iam/orgs/1/roles/$($role.id)" `
  -WebSession $aoi
```

租户自定义角色只能绑定当前产品的 `tenant`/`product` 权限，不能绑定 `permission:read` 等平台级权限。`owner` 等内置角色不可修改或删除；仍被成员或待处理邀请引用的角色也不可删除。

## 启用当前账号 TOTP MFA

```powershell
$mfa = Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/auth/mfa/setup `
  -WebSession $aoi

$mfa.secret
$mfa.otpauth_url

$code = Read-Host "输入认证器应用中的 6 位验证码"
$verified = Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/auth/mfa/verify `
  -ContentType "application/json" `
  -Body (@{ code = $code } | ConvertTo-Json) `
  -WebSession $aoi

$verified.recovery_codes
```

`secret` 和 `otpauth_url` 只在 setup 响应中用于绑定认证器应用；恢复码明文只在 verify 或 rotate 响应中出现一次。不要把这些值写入日志、URL、localStorage 或测试快照。持久化表 `iam_mfa_factors.secret_ciphertext` 只保存加密密文，`iam_mfa_recovery_codes` 只保存 hash、prefix 和状态。MFA 启用后，登录请求需要提交 `mfa_code` 或 `mfaCode`，可使用认证器 TOTP 或未消费恢复码；缺失时接口返回 `MFA_REQUIRED`。

查看或撤销当前账号 MFA 因子只返回元数据，不返回 secret 或密文；撤销 MFA 会同时吊销剩余 active 恢复码：

```powershell
Invoke-RestMethod -Uri http://127.0.0.1:8080/api/v1/auth/mfa/factors -WebSession $session

Invoke-RestMethod -Method Delete `
  -Uri http://127.0.0.1:8080/api/v1/auth/mfa/factors/1 `
  -WebSession $session
```

查看恢复码元数据或轮换恢复码：

```powershell
Invoke-RestMethod -Uri http://127.0.0.1:8080/api/v1/auth/mfa/recovery-codes -WebSession $aoi

$rotated = Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/auth/mfa/recovery-codes `
  -WebSession $aoi

$rotated.recovery_codes
```

## 启用自助注册

自助注册默认关闭，避免未规划的租户组织在公网环境被创建。本地联调可以显式打开：

```powershell
$env:CONSOLE__AUTH__SELF_SIGNUP_ENABLED = "true"
cargo run -p app -- serve --config configs/console.example.yaml
```

注册请求会在一个事务内创建 `pending_verification` 用户、tenant 组织、owner 成员关系、邮箱验证 pending 数据、通知 outbox 和加密投递 secret。任何一步失败都会回滚，不留下污染唯一索引的半成品账号或组织：

```powershell
$registerBody = @{
  email = "signup@example.com"
  password = "change-me-123"
  display_name = "注册用户"
  organization_code = "signup-main"
  organization_name = "注册组织"
} | ConvertTo-Json

Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/auth/register `
  -ContentType "application/json" `
  -Body $registerBody
```

注册响应只返回 accepted 语义，不返回邮箱验证 raw token。使用 `notification-drain` 领取本地 file driver、SMTP driver 或 queue driver 投递结果后，在 WebUI `/account` 的邮箱验证表单中粘贴一次性令牌完成验证；验证前该用户不能登录。

## 创建邀请和通知 pending 数据

```powershell
$inviteBody = @{ email = "invitee@example.com"; role_code = "owner" } | ConvertTo-Json
Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/orgs/1/users/invitations `
  -ContentType "application/json" `
  -Body $inviteBody `
  -WebSession $aoi

Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/auth/password/forgot `
  -ContentType "application/json" `
  -Body (@{ email = "owner@example.com" } | ConvertTo-Json)
```

本地可以先执行一次通知 outbox drain，确认 pending 通知被领取并标记投递结果。默认 file driver 会把一次性令牌写入 `data/notifications` 下的本地投递文件；raw token 不进入 outbox、日志、URL 或响应：

```powershell
cargo run -p app -- notification-drain --config configs/console.example.yaml --limit 20
```

如果投递 driver 临时失败，outbox 会增加 `attempt_count` 并按 `notification.retry_backoff_seconds` 延迟下一轮领取；达到 `notification.max_attempts` 后才标记最终失败并清空投递密文。`notification-drain` 输出中的 `retried` 表示已重新排队，`failed` 表示最终失败。

生产或联调真实邮箱时，把通知 driver 切到 SMTP，并通过 secrets/env 注入账号密码：

```powershell
$env:CONSOLE__NOTIFICATION__DRIVER="smtp"
$env:CONSOLE__NOTIFICATION__SMTP__HOST="smtp.example.com"
$env:CONSOLE__NOTIFICATION__SMTP__PORT="587"
$env:CONSOLE__NOTIFICATION__SMTP__USERNAME="<smtp-user>"
$env:CONSOLE__NOTIFICATION__SMTP__PASSWORD="<smtp-password>"
$env:CONSOLE__NOTIFICATION__SMTP__FROM="Aoi Console <noreply@example.com>"
$env:CONSOLE__NOTIFICATION__SMTP__TLS="start_tls"
cargo run -p app -- notification-drain --config configs/console.example.yaml --limit 20
```

SMTP driver 只在基础设施层使用邮件 SDK，IAM service 仍只写 pending 数据、outbox 和加密投递 secret。生产环境禁止 `file`/`log` driver，也禁止 `notification.smtp.tls=none`。

也可以切到本地加密 queue driver，给外部 worker 消费结构化 JSON envelope。queue driver 不会写 raw token，会把一次性令牌用 `notification.queue.secret_key` 重新加密为 `secret_ciphertext`：

```powershell
$env:CONSOLE__NOTIFICATION__DRIVER="queue"
$env:CONSOLE__NOTIFICATION__QUEUE__DIR="data/notification-queue"
$env:CONSOLE__NOTIFICATION__QUEUE__SECRET_KEY="<queue-secret-at-least-32-bytes>"
cargo run -p app -- notification-drain --config configs/console.example.yaml --limit 20
Get-ChildItem data/notification-queue
```

最终失败的通知可以通过脱敏 dead-letter 报表观察，不会返回 payload、raw token 或密文：

```powershell
cargo run -p app -- notification-dead-letters --config configs/console.example.yaml --limit 20
```

报表中的 `recipient_hint` 只保留收件人提示，`secret_state=purged` 表示投递密文已清空。若失败记录仍显示 `secret_state=pending_secret_present`，可以把它安全恢复为 `pending`，等待下一轮 drain；该命令不会重建已清空的一次性令牌，也不会输出密文：

```powershell
cargo run -p app -- notification-requeue-failed --config configs/console.example.yaml --limit 20
cargo run -p app -- notification-drain --config configs/console.example.yaml --limit 20
```

`notification-requeue-failed` 的报表会对每条记录给出 `requeued` 或 `skipped`，已 `purged`、缺失或并发状态变化的记录只会跳过。当前 queue driver 是本地持久化队列 envelope，已经通过 `NotificationSender` 注入；RabbitMQ/Kafka/SQS 等托管 MQ、外部队列 redrive 和告警策略仍需按部署环境接入。

接受邀请时，token 必须来自通知 driver 的安全投递结果，不放在 URL 中：

```powershell
$acceptBody = @{
  token = "<notification-worker-delivered-token>"
  password = "change-me-123"
  display_name = "受邀用户"
} | ConvertTo-Json

Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/auth/invitations/accept `
  -ContentType "application/json" `
  -Body $acceptBody `
  -SessionVariable invited
```

当前阶段不在 HTTP 响应中返回自助注册、邀请、重置、邮箱验证 raw token。已创建的通知会写入 `iam_notification_outbox`，一次性令牌只以密文写入 `iam_notification_delivery_secrets`，并可由 `notification-drain` 领取后交给本地 file、SMTP 或 queue driver；投递完成或达到最终失败后密文会被清空。未知邮箱的密码重置请求返回相同 accepted 语义，但不会创建 pending 数据、outbox 或投递 secret。接受邀请会在一个事务内创建用户、组织成员关系并标记邀请已接受。

WebUI 的 `/account` 页面会按 public settings 决定是否展示自助注册表单，并提供邀请接受、请求密码重置、执行密码重置、请求邮箱验证和确认邮箱验证表单。页面会清理历史 URL 中的查询参数，但不会从 `/account?invite=...`、`/account?reset=...` 或 `/account?verify=...` 读取 pending token；这些一次性令牌必须来自通知 driver 的安全投递结果，并手动填入对应表单。

## 查看 API 目录

```powershell
Invoke-RestMethod `
  -Uri http://127.0.0.1:8080/api/v1/system/apis `
  -WebSession $aoi
```

API 目录、权限元数据和 OpenAPI 都来自 Rust route registry。未登录请求会返回 `401`；只有拥有 `permission:read` 的平台会话可以读取该目录。生成仓库内契约快照时使用：

```powershell
cargo run -p app -- openapi --config configs/console.example.yaml --output docs/api/openapi.yaml
```

`docs/api/openapi.yaml` 是生成文件，不要手写维护。修改 `crates/core/app/src/transport/http/route_registry.rs` 后，应重新运行上面的命令，再执行 Rust 测试和 route inspection；`cargo test --workspace` 会检查该快照是否仍与当前 route registry 生成结果一致。

## 查看系统菜单、服务器状态和操作记录

```powershell
Invoke-RestMethod `
  -Uri http://127.0.0.1:8080/api/v1/system/menus `
  -WebSession $aoi

Invoke-RestMethod `
  -Uri http://127.0.0.1:8080/api/v1/system/server-status `
  -WebSession $aoi

Invoke-RestMethod `
  -Uri http://127.0.0.1:8080/api/v1/system/metrics/prometheus `
  -WebSession $aoi

$env:CONSOLE_OBSERVABILITY_SCRAPE_TOKEN = "local-prometheus-scrape-token"
cargo run -p app -- observability-token-hash --config configs/console.example.yaml
Remove-Item Env:\CONSOLE_OBSERVABILITY_SCRAPE_TOKEN

# 将输出的哈希写入 observability.prometheus_scrape_token_hash 并重启服务后，可用专用 Bearer token 验证外部监控路径：
Invoke-RestMethod `
  -Uri http://127.0.0.1:8080/api/v1/system/metrics/prometheus `
  -Headers @{ Authorization = "Bearer local-prometheus-scrape-token" }

Invoke-RestMethod `
  -Uri http://127.0.0.1:8080/api/v1/system/operation-records `
  -WebSession $aoi

Invoke-RestMethod `
  -Uri "http://127.0.0.1:8080/api/v1/system/operation-records/summary?top_limit=5" `
  -WebSession $aoi

Invoke-WebRequest `
  -Uri "http://127.0.0.1:8080/api/v1/system/operation-records/export.csv?limit=100" `
  -WebSession $aoi `
  -OutFile target/operation-records.csv

Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/system/operation-records/prune `
  -WebSession $aoi

cargo run -p app -- operation-records-summary --config configs/console.example.yaml --top-limit 5
cargo run -p app -- operation-records-prune --config configs/console.example.yaml
```

`server-status` 只返回后端真实采集字段：进程、运行时、产品版本、数据库驱动，以及 `sysinfo` 采集的全局 CPU、进程 CPU、内存、进程内存、交换空间、磁盘容量/使用量、磁盘数量、网络接口数量、网络累计接收/发送字节、系统 uptime/boot time 和 1/5/15 分钟 load average。`metrics/prometheus` 复用同一组真实字段输出 Prometheus text format；平台会话要求 `server:read` 平台权限，外部监控建议使用 `observability.prometheus_scrape_token_hash` 配置的专用 Bearer scrape token。该哈希与 `auth.session_secret` 绑定，轮换 session secret 后需要重新生成；raw scrape token 只交给监控系统和验收脚本，不写入 YAML、System 配置表、日志、报告或前端产物。操作记录不保存请求体、Cookie、Authorization 或 raw token；summary 只基于已落库的 route 模板、method、status、actor 和时间聚合，不反查请求正文；留存清理由 `audit.operation_record_retention_days` 和 `audit.operation_record_prune_batch_size` 控制，HTTP prune、CLI prune 与 scheduler 均复用同一后端 usecase。

## 管理系统配置、字典和参数

```powershell
Invoke-RestMethod -Method Put `
  -Uri http://127.0.0.1:8080/api/v1/system/configs/feature_flags `
  -ContentType "application/json" `
  -Body (@{ value = @{ setup_v2 = $true } } | ConvertTo-Json -Depth 5) `
  -WebSession $aoi

Invoke-RestMethod -Method Put `
  -Uri http://127.0.0.1:8080/api/v1/system/dictionaries/locales `
  -ContentType "application/json" `
  -Body (@{ name = "语言" } | ConvertTo-Json) `
  -WebSession $aoi

Invoke-RestMethod -Method Put `
  -Uri http://127.0.0.1:8080/api/v1/system/parameters/page_size `
  -ContentType "application/json" `
  -Body (@{ name = "默认分页大小"; value = "20" } | ConvertTo-Json) `
  -WebSession $aoi
```

配置和参数 key 不允许包含 `secret`、`token`、`password`、`private`、`credential`；真实密钥必须继续通过 YAML/env/secrets 管理。

## 登记版本包和媒体资产

```powershell
$versionPackage = Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/system/version-packages `
  -ContentType "application/json" `
  -Body (@{
    version_name = "Aoi 0.1.0"
    version_code = "0.1.0"
    manifest = @{ channel = "local"; notes = @("init") }
  } | ConvertTo-Json -Depth 5) `
  -WebSession $aoi

Invoke-RestMethod -Method Post `
  -Uri "http://127.0.0.1:8080/api/v1/system/version-packages/$($versionPackage.id)/publish" `
  -ContentType "application/json" `
  -Body (@{ reason = "本地验收发布" } | ConvertTo-Json) `
  -WebSession $aoi

Invoke-RestMethod `
  -Uri http://127.0.0.1:8080/api/v1/system/version-packages/releases `
  -WebSession $aoi

需要回滚时，目标版本必须不是当前 `active` 版本；接口路径为 `/api/v1/system/version-packages/{id}/rollback`，请求体同样只接收可选 `reason`，用于事件审计。

Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/system/media-assets `
  -ContentType "application/json" `
  -Body (@{
    category = "avatars"
    display_name = "Logo"
    storage_key = "media/logos/aoi.png"
    mime_type = "image/png"
    size_bytes = 1024
  } | ConvertTo-Json) `
  -WebSession $aoi
```

使用默认本地 storage 上传媒体文件：

```powershell
# PowerShell 7+
$uploadResult = Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/system/media-assets/upload `
  -Form @{
    category = "logos"
    display_name = "Aoi Logo"
    file = Get-Item ".\README.md"
  } `
  -WebSession $aoi
```

查看并删除当前 storage driver 下的对象：

```powershell
$objects = Invoke-RestMethod `
  -Uri "http://127.0.0.1:8080/api/v1/system/storage-objects?prefix=local/&limit=20" `
  -WebSession $aoi

$objects | Format-Table storage_key,size_bytes,updated_at

if ($objects.Count -gt 0) {
  Invoke-RestMethod -Method Delete `
    -Uri http://127.0.0.1:8080/api/v1/system/storage-objects `
    -ContentType "application/json" `
    -Body (@{ storage_key = $objects[0].storage_key } | ConvertTo-Json) `
    -WebSession $aoi
}
```

版本包发布/回滚只维护单 active 状态并写入 `system_version_release_events`；当前不会执行外部部署、机器回滚或制品分发。媒体上传默认写入 `storage.local_dir`，只保存 storage key 和元数据。也可以把上传后端切换到 S3 兼容对象存储，例如本地 MinIO：

```powershell
$env:CONSOLE__STORAGE__DRIVER = "s3"
$env:CONSOLE__STORAGE__S3__ENDPOINT = "http://127.0.0.1:9000"
$env:CONSOLE__STORAGE__S3__BUCKET = "console-media"
$env:CONSOLE__STORAGE__S3__REGION = "local"
$env:CONSOLE__STORAGE__S3__ACCESS_KEY_ID = "<minio-access-key>"
$env:CONSOLE__STORAGE__S3__SECRET_ACCESS_KEY = "<minio-secret-key>"
$env:CONSOLE__STORAGE__S3__ALLOW_HTTP = "true"
$env:CONSOLE__STORAGE__S3__FORCE_PATH_STYLE = "true"
$env:CONSOLE__STORAGE__S3__PREFIX = "media"
cargo run -p app -- check-config --config configs/console.example.yaml
```

S3 driver 只作为媒体上传和当前前缀内对象浏览/删除的 storage 后端，不管理 bucket 生命周期或权限策略。生产环境必须使用 HTTPS，`storage.s3.allow_http` 必须为 `false`，访问密钥应通过 secrets 文件或环境变量注入，不能写入 System 配置表。

## 创建并执行流量探针

```powershell
$probeBody = @{
  name = "官网健康检查"
  url = "https://example.com/"
  expected_status = 200
} | ConvertTo-Json

$probeTarget = Invoke-RestMethod -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/system/traffic-probes/targets `
  -ContentType "application/json" `
  -Body $probeBody `
  -WebSession $aoi

Invoke-RestMethod -Method Post `
  -Uri "http://127.0.0.1:8080/api/v1/system/traffic-probes/targets/$($probeTarget.id)/run" `
  -WebSession $aoi

Invoke-RestMethod `
  -Uri "http://127.0.0.1:8080/api/v1/system/traffic-probes/results?target_id=$($probeTarget.id)&limit=10" `
  -WebSession $aoi

$probeAlerts = Invoke-RestMethod `
  -Uri "http://127.0.0.1:8080/api/v1/system/traffic-probes/alerts?target_id=$($probeTarget.id)&status=open&limit=10" `
  -WebSession $aoi

if ($probeAlerts.Count -gt 0) {
  Invoke-RestMethod -Method Post `
    -Uri "http://127.0.0.1:8080/api/v1/system/traffic-probes/alerts/$($probeAlerts[0].id)/ack" `
    -WebSession $aoi

  Invoke-RestMethod -Method Post `
    -Uri "http://127.0.0.1:8080/api/v1/system/traffic-probes/alerts/$($probeAlerts[0].id)/resolve" `
    -WebSession $aoi
}
```

也可以不经过 HTTP 管理端，直接让 scheduler 执行一轮已登记目标的真实采集：

```powershell
cargo run -p app -- scheduler-run-once --config configs/console.example.yaml
```

后台 scheduler 默认关闭，避免本地服务启动后主动探测外部地址。需要周期采集时再显式开启：

```powershell
$env:CONSOLE__SCHEDULER__ENABLED = "true"
$env:CONSOLE__SCHEDULER__TRAFFIC_PROBE_INTERVAL_SECONDS = "300"
cargo run -p app -- serve --config configs/console.example.yaml
```

流量探针只保存真实 HTTP GET 采集得到的状态码、耗时、最终 URL 和分类原因。非 `healthy` 结果会写入持久化告警，后续同目标 `healthy` 结果会自动把未关闭告警标记为 `resolved`；WebUI 当前只调用后端已暴露的告警查询、确认和恢复 API。SSE 订阅端点为 `/api/v1/system/traffic-probes/events`，响应 `text/event-stream`，复用 `traffic_probe:read` 权限、当前告警查询和 `scheduler.event_stream_heartbeat_seconds` 重连提示；后续 WebUI 实时订阅只能接入该真实端点。DNS/TLS 细节采集尚未接入，不得由前端 mock 成生产能力。

## 停止

在启动服务的终端按 `Ctrl+C`。


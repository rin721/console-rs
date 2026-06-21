# 目标环境验收清单

本文用于记录 Aoi[葵] 在真实部署环境中的最终验收结果。它不是本地 smoke 的替代品；只有目标环境证据填完整，才能把外部数据库、网络、反向代理、TLS、通知、存储和运行手册验收视为完成。

## 基本信息

| 项目 | 记录 |
| --- | --- |
| 验收日期 | 待填写 |
| 验收人 | 待填写 |
| Git commit / tag | 待填写 |
| 部署环境 | 待填写 |
| 应用入口 URL | 待填写 |
| 数据库 driver | `postgres` / `mysql` / `sqlite` |
| WebUI 托管方式 | Rust binary / 反向代理静态托管 |
| secrets 来源 | 环境变量 / secrets 文件 / secret manager |

## 必须通过的命令

以下命令应在目标环境或同等网络边界的发布流水线中运行。输出日志需要保存到发布记录或 CI artifact，敏感 URL 可脱敏，但不得删除 `driver`、`serve_ready`、迁移文件名、checksum 和失败原因。推荐先使用汇总脚本生成机器可读报告：

```powershell
$env:CONSOLE_ACCEPTANCE_METRICS_SCRAPE_TOKEN = "<raw-prometheus-scrape-token>"

powershell -NoProfile -ExecutionPolicy Bypass -File scripts/target-environment-acceptance.ps1 `
  -Driver postgres `
  -Url "<postgres-url>" `
  -BaseUrl "https://console.example.com" `
  -ApplyMigrations

powershell -NoProfile -ExecutionPolicy Bypass -File scripts/validate-target-acceptance-report.ps1 `
  -ReportPath "target/target-environment-acceptance/<report>.json"

powershell -NoProfile -ExecutionPolicy Bypass -File scripts/goal-completion-audit.ps1 `
  -TargetAcceptanceReport "target/target-environment-acceptance/<report>.json" `
  -RequireReady
```

目标服务的 `observability.prometheus_scrape_token_hash` 必须提前配置为同一个 raw scrape token 的哈希，且生成哈希时要使用目标服务实际加载的 `auth.session_secret`。示例：

```powershell
$env:CONSOLE_OBSERVABILITY_SCRAPE_TOKEN = "<raw-prometheus-scrape-token>"
cargo run -p app -- observability-token-hash --config "<target-config.yaml>" --secrets "<target-secrets.yaml>"
Remove-Item Env:\CONSOLE_OBSERVABILITY_SCRAPE_TOKEN
```

将输出的 64 位 SHA-256 hex 写入目标配置或 secrets 覆盖层；不要把 raw scrape token 写入 YAML、System 配置表、日志、报告或前端产物。`CONSOLE_ACCEPTANCE_METRICS_SCRAPE_TOKEN` 只由验收脚本进程读取，用于证明目标环境授权 scrape 可用，报告中会脱敏为 `<metrics-scrape-token>`。

当修改目标验收报告校验器或发布证据规则时，还必须运行 `scripts/target-acceptance-report-validator-smoke.ps1`。该 smoke 只证明校验器规则能接受或拒绝合成报告样例，不能替代上面的真实目标环境报告。

验收脚本会在 PostgreSQL/MySQL 时串联 `database-external-smoke.ps1` 与 `database-deploy-preflight.ps1 -ApplyMigrations`，并探测目标入口 HTTPS 策略、目标服务 `/health`、`/ready`、`/openapi.yaml`、Setup status、public settings、Cookie/CSRF 生产策略、受保护指标策略、WebUI fallback 和 `/api/*` fallback 边界。默认 full scope 必须同时包含数据库和 HTTP/WebUI 探针，缺 `-BaseUrl` 或使用 `-SkipDatabase`/`-SkipHttp` 会失败；非本地入口必须使用 `https://`，本地 loopback 的 `http://` 只允许做本地 smoke；只有显式传入 `-AllowPartial` 时才允许生成 `result=partial` 的诊断报告，且 partial 报告不能填写为最终通过证据。JSON 报告默认写入 `target/target-environment-acceptance/`，数据库 URL 会脱敏为 `<database-url>`。报告校验脚本会拒绝 partial、failed、本地 HTTP final、非本地 HTTP、缺数据库 preflight、缺 HTTP/WebUI 探针、缺 `entrypoint-security`、缺 `csrf-policy` 或缺 `metrics-policy` 的报告；`metrics-policy` 必须同时证明匿名请求 `/api/v1/system/metrics/prometheus` 被 401/403 拒绝且响应体没有 Prometheus 指标正文，并证明配置化 Bearer scrape token 可返回 200 与真实指标正文，且不会泄露 raw scrape token、API token、session token 或 secret；`-AllowLocalHttp` 只用于本地 smoke 报告结构校验，不得用于最终发布。目标报告通过校验并回填本清单后，再运行 `scripts/goal-completion-audit.ps1 -TargetAcceptanceReport <report> -RequireReady`，作为是否允许声明 `/goal` 完成的总闸。下面的单项命令仍可用于人工复核和故障定位。

```powershell
cargo fmt --all --check
cargo check --workspace
cargo clippy --workspace --all-targets -- -D warnings
cargo test --workspace
cargo build --workspace
npm --prefix web/app run typecheck
npm --prefix web/app run lint:i18n
npm --prefix web/app run build
npm --prefix web/app run test:e2e
```

真实 PostgreSQL：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-external-smoke.ps1 `
  -Driver postgres `
  -Url "<postgres-url>"

powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-deploy-preflight.ps1 `
  -Driver postgres `
  -Url "<postgres-url>" `
  -ApplyMigrations
```

真实 MySQL：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-external-smoke.ps1 `
  -Driver mysql `
  -Url "<mysql-url>"

powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-deploy-preflight.ps1 `
  -Driver mysql `
  -Url "<mysql-url>" `
  -ApplyMigrations
```

目标服务启动后：

```powershell
Invoke-RestMethod -Uri "<base-url>/health"
Invoke-RestMethod -Uri "<base-url>/ready"
Invoke-RestMethod -Uri "<base-url>/openapi.yaml"
Invoke-RestMethod -Uri "<base-url>/api/v1/setup/status"
Invoke-RestMethod -Uri "<base-url>/api/v1/system/public-settings"
```

## 验收矩阵

| 门禁 | 通过条件 | 结果 | 证据位置 |
| --- | --- | --- | --- |
| 外部数据库 smoke | `database-external-smoke.ps1` 成功；plan、ping、insert-id、migrate、Setup/IAM/Notification/System repository probes、history、schema-check、preflight、二次 migrate 均通过 | 待填写 | 待填写 |
| 部署前数据库门禁 | `database-deploy-preflight.ps1 -ApplyMigrations` 成功；`schema_ready=true`、`repository_ready=true`、`serve_ready=true` | 待填写 | 待填写 |
| 目标验收 scope | `target-environment-acceptance.ps1` 报告 `scope=full` 且 `result=passed`；`partial` 只允许作为诊断材料 | 待填写 | 待填写 |
| 目标验收报告校验 | `validate-target-acceptance-report.ps1 -ReportPath <report>` 成功；不得使用 `-AllowLocalHttp` 作为最终证据 | 待填写 | 待填写 |
| 目标入口 HTTPS | `target-environment-acceptance.ps1` 的 `entrypoint-security` 通过；非本地 `BaseUrl` 必须是 `https://`，本地 loopback `http://` 只用于 smoke | 待填写 | 待填写 |
| secrets 分层 | 生产配置拒绝占位密钥；真实密钥来自 env/secrets/secret manager；未写入 System 配置表或前端产物 | 待填写 | 待填写 |
| HTTP 探针 | `/health`、`/ready`、`/openapi.yaml`、Setup status、public settings 均由目标服务返回 | 待填写 | 待填写 |
| WebUI 托管 | `/`、`/setup/*`、`/admin/*` 能加载构建产物；`/api/*` 不被 SPA fallback 吞掉 | 待填写 | 待填写 |
| Cookie/CSRF | `target-environment-acceptance.ps1` 的 `csrf-policy` 通过；public settings 暴露 `csrf_enabled=true`，CSRF cookie 非 HttpOnly 且 Secure，非 GET `/api/*` 缺 CSRF 时被 403 拒绝 | 待填写 | 待填写 |
| 指标访问策略 | `target-environment-acceptance.ps1` 的 `metrics-policy` 通过；匿名访问 `/api/v1/system/metrics/prometheus` 返回 401/403 且不泄露 Prometheus 指标正文；配置化 Bearer scrape token 返回 200、包含真实指标正文且不泄露 raw token、API token、session token 或 secret | 待填写 | 待填写 |
| IAM 安全路径 | 首个管理员、登录、refresh、API Token、邀请、密码重置、邮箱验证、TOTP MFA 在目标环境可运行且不泄露 raw token | 待填写 | 待填写 |
| System 真实能力 | server-status 返回真实采集字段，包括 CPU、内存、磁盘、网络接口与累计收发字节；metrics/prometheus 受 `server:read` 或专用 scrape token 哈希保护并导出同一组真实字段，且匿名访问由 `metrics-policy` 拒绝；版本包、媒体、对象浏览/删除、流量探针调用真实后端 API | 待填写 | 待填写 |
| 通知投递 | SMTP 或目标队列 driver 可投递；outbox 不保存 raw token；投递完成或最终失败会清空密文 | 待填写 | 待填写 |
| 存储后端 | `local` 或 S3 兼容 storage 可上传、列出、删除对象；生产 S3 禁止明文 HTTP | 待填写 | 待填写 |
| 反向代理/TLS | 外部入口启用 TLS；转发保留 Cookie、CSRF header、Authorization header 和 SSE 连接；`entrypoint-security` 不能是 partial | 待填写 | 待填写 |
| 备份/回滚 | 数据库备份、迁移回滚策略、版本包状态回滚流程已演练或记录 | 待填写 | 待填写 |
| 日志与审计 | 日志不含 session/API/refresh/MFA/pending raw token；操作记录只保存 route 模板和状态；审计 summary 可基于真实后端记录输出总量、状态段、method 分布和 top path；按 `audit.operation_record_retention_days` 执行留存清理并归档清理报告 | 待填写 | 待填写 |

## 暂缓能力确认

这些能力当前不得用前端 mock 或文案宣称完成：

| 能力 | 当前处理 | 进入完成条件 |
| --- | --- | --- |
| 外部版本部署执行器 | 版本包只维护元数据、单 active 状态和发布事件 | 接入真实制品分发、发布、回滚执行器及权限/审计闭环 |
| bucket 生命周期/权限策略控制面 | Storage API 只做当前 driver 对象上传、浏览和删除 | 接入真实 bucket 生命周期、对象策略管理、权限和审计 |
| 外部指标导出/深度主机指标 | 已有受 `server:read` 或专用 scrape token 哈希保护的 Prometheus 文本导出，WebUI 只展示 `server-status` 已返回的 CPU、内存、磁盘、网络和 load 字段 | 在目标监控系统中配置 raw scrape token，运行 `metrics-policy` 证明授权 scrape；接入更深主机采集器 |
| 托管外部 MQ/告警策略 | 当前 queue driver 是本地持久化 envelope；最终失败通知已有脱敏 dead-letter 报表，且仍有 pending secret 的 failed 记录可安全重排队 | 接入 RabbitMQ/Kafka/SQS 等真实队列、外部队列 redrive 和告警策略 |

## 最终判定

| 项目 | 结果 |
| --- | --- |
| 是否可对外发布 | 待填写 |
| 阻断项 | 待填写 |
| 残留风险 | 待填写 |
| 下一步负责人 | 待填写 |

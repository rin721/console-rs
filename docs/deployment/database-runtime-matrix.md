# 数据库运行矩阵

本文面向部署者和后续维护者，说明 Aoi[葵] 当前数据库能力的真实运行边界。配置层识别 `sqlite`、`postgres`、`mysql`，运行时通过 `DatabaseConnection` 装配对应 pool 和 repository set；未声明的 driver 必须显式失败，禁止静默回退到 SQLite。

## 当前状态

| Driver | 配置识别 | URL 校验 | Schema/迁移目录 | 运行时支持 | 当前用途 |
| --- | --- | --- | --- | --- | --- |
| `sqlite` | 是 | `sqlite:` | `migrations/*.sql` | `ready` | 默认本地与轻量部署数据库 |
| `postgres` | 是 | `postgres://` 或 `postgresql://` | `migrations/postgres/*.sql` | `ready` | 外部数据库部署与真实服务 smoke |
| `mysql` | 是 | `mysql://` | `migrations/mysql/*.sql` | `ready` | 外部数据库部署与真实服务 smoke |

`cargo run -p app -- check-config --config configs/console.example.yaml` 会在输出中包含 `database_runtime`。该字段用于向 WebUI、Setup 配置检测和部署脚本暴露当前 driver 的运行能力，而不是让前端或脚本猜测数据库状态。

`cargo run -p app -- database-plan --config configs/console.example.yaml` 会输出同一配置下的 runtime 状态、自动迁移策略、迁移目录、迁移文件大小和 SHA-256 摘要。SQLite、PostgreSQL 和 MySQL 都应报告 `runtime_status=ready`。

`cargo run -p app -- database-ping --config configs/console.example.yaml` 会使用当前 driver 对应的 sqlx pool 执行连接探针，并输出 `repository_runtime`。PostgreSQL/MySQL 使用外部 pool，SQLite 使用本地 pool。

`cargo run -p app -- database-insert-id-probe --config configs/console.example.yaml` 会创建临时表并插入一行，用当前方言的插入后 ID 策略读取生成主键。SQLite/PostgreSQL 使用 `insert ... returning id`；MySQL 必须在同一连接内通过 `select last_insert_id()` 读取。

`cargo run -p app -- database-migrate --config configs/console.example.yaml` 会显式执行当前 driver 的迁移脚本。SQLite 使用 sqlx migrator 和 `_sqlx_migrations`；PostgreSQL/MySQL 使用方言化 SQL 文件和带 SHA-256 摘要的 `schema_migrations` 记录表。如果同一版本已应用但当前文件摘要不同，命令会失败并要求新建增量迁移。

`cargo run -p app -- database-migration-history --config configs/console.example.yaml` 会反查当前数据库已应用迁移记录。SQLite 历史来自 sqlx `_sqlx_migrations`；PostgreSQL/MySQL 历史来自 `schema_migrations`，checksum 为迁移文件 SHA-256。

`cargo run -p app -- database-setup-repository-probe --config configs/console.example.yaml` 会在已迁移数据库上写入一次性 setup run/log，并读取 setup 完成状态、run 列表和 log 列表，证明 SetupRepository trait 的读写路径可以在当前 driver 上运行。

`cargo run -p app -- database-iam-repository-probe --config configs/console.example.yaml` 会在空的 smoke 数据库上写入一次性管理员、权限、会话、API Token、pending 通知、邀请、密码重置、邮箱验证、MFA 与审计记录，证明 IamRepository trait 的安全关键读写路径可以在当前 driver 上运行。

`cargo run -p app -- database-notification-repository-probe --config configs/console.example.yaml` 会在已迁移数据库上写入一次性 notification outbox/secret 记录，并验证 claim、delivered、retry、final failed、投递 secret 清理、已清除 secret 的 requeue 拒绝和仍有 pending secret 的安全重排队，证明 NotificationRepository trait 的核心投递状态机可以在当前 driver 上运行。

`cargo run -p app -- database-system-repository-probe --config configs/console.example.yaml` 会在已迁移数据库上写入一次性 catalog/menu/config/dictionary/parameter/operation/version/media/traffic probe 记录，并校验操作记录汇总报表，证明 SystemRepository trait 的核心读写和聚合路径可以在当前 driver 上运行。

`cargo run -p app -- database-schema-check --config configs/console.example.yaml` 会反查当前数据库是否包含平台核心表，并输出 `repository_runtime` 能力矩阵。schema-ready 且 repository-ready 是进入 serve 的必要条件。

`cargo run -p app -- database-preflight --config configs/console.example.yaml` 会聚合迁移计划、连接探针、迁移历史、核心表和 repository readiness，输出 `repository_runtime`、`repository_ready` 与 `serve_ready`。SQLite、PostgreSQL 和 MySQL 在迁移完成后都应得到 `serve_ready=true`。

`scripts/database-deploy-preflight.ps1` 是部署流水线入口，按目标配置执行 plan、ping、insert-id-probe、可选显式迁移、migration-history、schema-check 和 preflight。默认策略会拒绝任何没有通过 `serve_ready` 的部署。

`scripts/repository-dialect-audit.ps1` 会扫描 `SqliteRepository` 的方言隔离点，确认 `database.rs` 暴露类型化 `repository_runtime` 能力矩阵，并确认 PostgreSQL/MySQL 的 `DatabaseRuntimeSupport` 保持 `ready`。当前隔离基线为 `Pool<Sqlite>=2`、`SqliteConnection=5`、`SqlDialect::Sqlite` 常量=1、生产 SQL literal 中 SQLite bind placeholder `?=0`、`insert_returning_id=0`、`sqlite_row_type=0`、`last_insert_rowid=0`，且所有已登记 scattered SQL 指标均为 0。审计保留这些数字是为了防止 SQLite-only 代码重新扩散到业务 repository，而不是阻止外部 runtime。

SQLite upsert 语义、`limit ?` 查询、Setup 完成状态/run/log SQL、会话 SQL、API Token 读取/列表/撤销 SQL、System 配置/字典/参数读删 SQL、API catalog/menu 读取、操作记录写入/查询/汇总/留存删除 SQL、流量探针目标读删/status 更新、告警状态更新、媒体库列表/软删除、版本包列表/按 ID 查询/active 查询、版本包退役/激活更新、版本包/媒体库/流量探针目标/结果/告警插入、IAM 组织/用户创建、成员关系写入、审计日志写入、组织用户管理、IAM 权限列表、角色权限关系、租户角色读取/更新/删除、IAM pending 状态读写、用户密码/邮箱验证状态更新、IAM 角色/API Token/邀请/密码重置/邮箱验证/MFA factor 插入、通知 outbox/投递 secret 插入与状态读写、MFA 恢复码插入、MFA factor 状态读写、MFA 恢复码状态读写、版本发布事件插入、版本发布事件列表/版本包软删除和删除前状态查询 SQL 已集中到 `sql_templates.rs`。SQLite/PostgreSQL 插入后取 ID 使用显式 `returning id` 模板，MySQL 使用 `InsertIdStrategy::DialectSpecificPostInsertRead` 与 `InsertIdRead::PostInsertQuery("select last_insert_id()")` 固化后续读取策略；MySQL 操作记录留存删除使用 derived table，避免 `delete` 目标表直接出现在子查询里；MySQL 操作记录汇总会显式 cast 聚合值，避免 unsigned/decimal 解码差异。

`scripts/database-external-smoke.ps1` 会在真实 PostgreSQL/MySQL 服务上验证 `database-plan`、`database-ping`、`database-insert-id-probe`、`database-migrate`、`database-setup-repository-probe`、`database-iam-repository-probe`、`database-notification-repository-probe`、`database-system-repository-probe`、`database-migration-history`、`database-schema-check`、`database-preflight` 和二次迁移 skipped 报告，并逐项比对 plan SHA-256 与 `schema_migrations` checksum；NotificationRepository probe 会覆盖 dead-letter、secret purge 和安全 requeue；SystemRepository probe 会覆盖操作记录汇总报表；MySQL 会额外确认 ID 读取策略来自同连接 `select last_insert_id()`。CI 的 `external-database-smoke` job 使用服务容器运行该脚本，并对同一 PostgreSQL/MySQL 服务继续执行 `database-deploy-preflight.ps1 -ApplyMigrations`。

目标环境的最终验收记录应填写到 `docs/deployment/target-environment-acceptance.md`。`scripts/target-environment-acceptance.ps1` 可在目标网络边界内串联外部数据库 smoke、部署前数据库门禁、目标入口 HTTPS 策略、目标 HTTP/WebUI 探针和 Cookie/CSRF 生产策略探针，并把结果写成 JSON artifact。最终发布证据必须是 `scope=full` 且 `result=passed`；非本地 `BaseUrl` 必须是 `https://`，显式 `-AllowPartial` 生成的 `result=partial` 只用于故障定位。本地 SQLite smoke 和 CI 服务容器 smoke 只能证明代码与流水线门禁，不能替代目标网络、反向代理、TLS、SMTP/S3、备份/回滚和实际流量验收。

## SQLite

SQLite 是默认本地数据库：

- `crates/core/app/src/infrastructure/database.rs` 创建 `SqlitePool`，启用 WAL 和外键，并按配置决定是否自动执行迁移。
- `migrations/*.sql` 下的当前 schema 面向 SQLite 方言维护，不包含旧插件表。
- `database-plan`、`database-migrate`、`database-migration-history` 和 `database-preflight` 覆盖迁移计划、显式迁移、历史反查和运行前门禁。
- app 生命周期层通过中性的 `DatabaseConnection` 与 `RepositorySet` 注入 repository，handler 和 service 不直接依赖具体连接池。

## PostgreSQL/MySQL

PostgreSQL 和 MySQL 已接入 app runtime：

- 配置层校验 driver 与 URL scheme 必须匹配。
- `migrations/postgres/20260621000100_init_core.sql` 与 `migrations/mysql/20260621000100_init_core.sql` 是方言化 bootstrap schema，覆盖当前核心表、身份、通知、System、版本包、媒体和流量探针 schema。
- `connect()` 会为外部 driver 创建对应 sqlx pool，并通过 `RepositorySet` 装配 `ExternalSetupRepository`、`ExternalIamRepository`、`ExternalNotificationRepository` 与 `ExternalSystemRepository`。
- PostgreSQL/MySQL 在 `migration.auto_apply=true` 时会先执行显式迁移，再启动服务；生产环境可关闭自动迁移，由部署流水线先运行 `database-deploy-preflight.ps1 -ApplyMigrations`。
- MySQL 的插入后 ID 读取必须在同一连接或事务内执行 `select last_insert_id()`，不能复用 SQLite/PostgreSQL 的 `returning id` 假设。
- 外部数据库的最终验收必须在真实 PostgreSQL/MySQL 服务上运行 `database-external-smoke.ps1`，不能只依赖 SQLite 本地 smoke。

## 验证命令

```powershell
cargo test -p config@0.1.0 database_runtime_support_reports_ready_external_drivers
cargo test -p config@0.1.0 database_driver_accepts_matching_url_schemes
cargo test -p config@0.1.0 database_driver_rejects_mismatched_url_scheme
cargo test -p app migration_plan
cargo test -p app sqlite_ping_reports_runtime_ready_without_running_migrations
cargo test -p app sqlite_insert_id_probe_reports_returning_strategy
cargo test -p app sqlite_migrate_applies_runtime_migrations
cargo test -p app config_checks_report_external_database_runtime_ready
cargo run -p app -- database-plan --config configs/console.example.yaml
cargo run -p app -- database-ping --config configs/console.example.yaml
cargo run -p app -- database-insert-id-probe --config configs/console.example.yaml
cargo run -p app -- database-migrate --config configs/console.example.yaml
cargo run -p app -- database-setup-repository-probe --config configs/console.example.yaml
cargo run -p app -- database-iam-repository-probe --config configs/console.example.yaml
cargo run -p app -- database-notification-repository-probe --config configs/console.example.yaml
cargo run -p app -- database-system-repository-probe --config configs/console.example.yaml
cargo run -p app -- database-migration-history --config configs/console.example.yaml
cargo run -p app -- database-schema-check --config configs/console.example.yaml
cargo run -p app -- database-preflight --config configs/console.example.yaml
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-deploy-preflight.ps1 -ApplyMigrations
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-runtime-smoke.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/repository-dialect-audit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-external-smoke.ps1 -Driver postgres -Url "<postgres-url>"
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-external-smoke.ps1 -Driver mysql -Url "<mysql-url>"
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-deploy-preflight.ps1 -Driver postgres -Url "<postgres-url>" -ApplyMigrations
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-deploy-preflight.ps1 -Driver mysql -Url "<mysql-url>" -ApplyMigrations
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/database-schema-dialect-scan.ps1
```

这些命令分别证明配置识别、协议校验、运行能力报告、迁移计划输出、连接探针、插入 ID 策略、显式迁移执行、Setup/IAM/Notification/System repository 读写和操作记录聚合路径、迁移历史反查、核心表反查、运行前聚合预检、迁移报告幂等性、真实 PostgreSQL/MySQL 方言执行、部署门禁、以及三类 schema 都包含必需表且不含插件运行时表。

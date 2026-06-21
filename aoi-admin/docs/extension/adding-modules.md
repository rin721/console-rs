# 新增模块

新增应用能力优先放在 `internal/modules/<name>`，再由 `internal/app/initapp` 完成装配。模块可以使用现有包名 `model`、`service`、`repository`、`handler` 和可选 `infrastructure`，但依赖方向必须保持清楚：业务核心定义接口，外层实现接口。

## 推荐形态

```text
model <- service -> handler
model <- service <- repository
service <- infrastructure
internal/app -> repository/infrastructure/service/handler
```

很小的模块不需要一次创建所有包。只要出现数据库、缓存、HTTP client、SMTP、TOTP、RBAC、host metrics、secret resolver 等基础设施细节，就应放到 repository、infrastructure 或 app adapter，不放进 service。

## 接线步骤

1. 在 `internal/modules/<name>` 下创建 `model` 和 `service`。service 先定义输入输出、业务错误和需要的最小接口。
2. 如需持久化，在 `repository` 实现 service 中定义的 repository contract；事务、SQL/ORM 条件和底层 not-found 映射留在 repository。
3. 如需外部系统调用或技术实现，在 `infrastructure` 实现 service 中定义的接口，例如 notifier、proxy、secret resolver。
4. 如需 HTTP 暴露，在 `handler` 绑定请求、读取 principal、调用 service 并转换响应。
5. 在 `internal/app/initapp` 创建配置、基础设施实现、repository/infrastructure、service 和 handler。
6. 在 `internal/transport/http` 或 `internal/transport/rpc` 注册路由或方法。
7. 补 service、handler、路由和边界测试；真实 SQLite、迁移、token、RBAC、TOTP 优先用 `internal/app/testsupport`。

## 边界规则

- module service 不导入 `pkg/*`、`internal/app`、同模块 `repository` 或 `internal/ports`。
- module service 不创建数据库连接、HTTP client、SMTP client、logger、config loader、token manager、RBAC enforcer、TOTP provider 或 host metrics collector。
- `internal/modules`、`internal/middleware`、`internal/transport` 的生产代码不直接导入 `pkg/*`。
- `pkg` 不导入任何 `internal/*`。
- 新增基础能力如果只服务一个模块，优先放模块 `infrastructure`；如果是可复用基础设施，再放 `pkg` 并在 `internal/app` 适配。

## 身份和权限

受保护 HTTP 接口应通过 IAM middleware 读取 principal，并通过权限中间件声明权限码，例如 `report:read`、`report:export`。权限码需要进入 IAM 权限目录，System API/权限同步会根据路由目录标注权限注册状态。

业务 service 可以接收 IAM principal 这类业务身份类型，但不应直接依赖 JWT、Casbin 或 HTTP context。

## 配置和迁移

- 新增配置字段时同步 `internal/config`、`configs/config.example.yaml`、`.env.example`、`deploy/config.production.example.yaml` 和 `docs/environment/configuration.md`。
- 新增表结构时追加 goose migration；共享后的迁移视为 append-only。

## 参考模块

- IAM：service-local token/authz/TOTP/notifier/repository contract 示例。
- Plugins：service-local HTTP proxy、health checker、secret resolver contract 示例。
- System：service-local repository、host metrics 和 storage error 映射示例。

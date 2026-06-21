# 新人接手指南

本文面向刚接手项目的维护者。第一目标不是读完所有代码，而是先跑起来，再沿一条最小链路理解启动、路由、service、repository 和 app 装配。

## 先记住这张地图

| 层 | 路径 | 第一眼怎么理解 |
| --- | --- | --- |
| 进程入口 | `cmd/aoi` | 声明命令、解析参数、启动服务或执行 CLI 任务 |
| 装配根 | `internal/app` | 创建基础设施，装配模块，管理生命周期 |
| 业务模块 | `internal/modules` | `service` 写业务 contract，`repository/infrastructure` 实现技术细节 |
| 传输层 | `internal/transport`、`internal/middleware` | 注册路由、中间件、认证、权限和响应转换 |
| 基础设施 | `pkg` | 数据库、日志、HTTP/RPC server、缓存、迁移、token、RBAC、TOTP 等实现 |

第一遍不要从 `pkg` 开始。`pkg` 细节很多，但业务主线在 `internal/app` 和 `internal/modules`。

## 第一天阅读路线

1. `docs/README.md`
2. `docs/architecture/layers.md`
3. `cmd/aoi/main.go`
4. `cmd/aoi/app.go`
5. `internal/app/initapp/modules.go`
6. `internal/transport/http/router.go`
11. `internal/modules/iam/service/service.go`
12. `internal/plugin/admin_service.go`
13. `internal/modules/system/service/service.go`

这条路线会带过命令入口、应用装配、路由注册、HTTP handler、业务 service、本地接口 contract 和具体实现。

## 本地跑起来

```powershell
go run ./cmd/aoi server --config=configs/config.yaml
curl http://127.0.0.1:9999/health
curl http://127.0.0.1:9999/ready
```

打开 `http://127.0.0.1:9999/admin`。如果 IAM 还没有用户，后台会进入首次初始化页面。

也可以通过 CLI 初始化管理员：

```powershell
"change-this-local-password" | go run ./cmd/aoi iam bootstrap-admin --config=configs/config.yaml --org-code=acme --org-name="Acme Corp" --username=admin --email=admin@example.com --password-stdin
```

本地默认使用 SQLite `./data/app.db`。IAM/System 迁移通常随本地服务自动应用；生产或共享环境应通过 `db migrate status/up` 显式处理。

## 试一条系统功能链路

系统菜单链路：

```text
HTTP request
  -> router.go
  -> system/handler
  -> system/service
  -> service-local repository/permission contract
  -> system/repository 或内置菜单快照
```

读代码时先回答：

1. 路由在哪里注册？
2. handler 从请求和 context 里取了什么？
3. service 如何过滤权限和组织运行时上下文？
4. repository 或基础设施负责哪些查询、事务或错误映射？
## 后台功能试用路径

| 功能 | 入口 | 注意 |
| --- | --- | --- |
| 用户管理 | 左侧 `用户` | 当前新增成员走邀请流程，不直接创建账号 |
| 组织管理 | 左侧 `组织` | 新建组织会把当前用户加入并授予 owner |
| 安全/MFA | 左侧 `安全` | TOTP secret 由 IAM service 加密存储 |
| 会话 | 左侧 `会话` | 撤销后对应 refresh token 失效 |
| API Token | 左侧 `API Token` | 完整 token 只在创建成功时显示一次 |
| 媒体库 | 左侧 `媒体库` | 外链导入不需要对象存储，本地上传需要 Storage |
| 断点上传 | 左侧 `断点上传` | 需要 media 相关迁移和 Storage |
| 版本管理 | 左侧 `版本管理` | 发布包记录菜单/API/字典快照，不是 Go 构建版本 |

## 新功能应该改哪里

| 任务 | 优先看哪里 |
| --- | --- |
| 新增 HTTP 接口 | `internal/transport/http/router.go`、模块 `handler` |
| 修改业务规则 | 模块 `service` |
| 修改数据库读写 | 模块 `repository` |
| 接入外部系统 | 模块 `infrastructure` 或 `internal/app/adapters` |
| 新增表结构 | `internal/migrations` |
| 修改配置字段 | `internal/config`、示例配置、配置文档 |
| 修改通用基础能力 | `pkg`，并确认不依赖 `internal` |

## 接手维护检查表

- 先确认需求属于哪个模块或能力域。
- 先看对应模块已有测试和文档。
- service 层保持业务 contract，不直接创建基础设施。
- 数据库结构变更追加迁移，不改已共享迁移。
- 配置变更同步示例配置、`.env.example`、生产配置示例和文档。
- API 变更同步 HTTP API 文档和 OpenAPI。

常用验证：

```powershell
go test ./... -count=1 -mod=readonly
go vet ./...
go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi
```

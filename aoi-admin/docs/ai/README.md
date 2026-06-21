# AI 协作入口

本目录是给 Codex 和其他代码代理使用的运行态说明。它只补充协作流程、历史证据和长任务上下文；面向开发者的当前事实以 `docs/README.md`、`AGENTS.md` 和各主题文档为准。

## 当前入口

| 文件 | 状态 | 用途 |
| --- | --- | --- |
| `project-map.md` | 当前事实 | 代码地图、分层边界、扩展路径和高风险区。 |
| `tooling.md` | 当前事实 | 本地工具、验证命令、GitHub/安全/浏览器工作流说明。 |
| `prompts.md` | 当前事实 | 可复用任务提示词，已按当前洋葱边界更新。 |
| `handoff-template.md` | 当前模板 | 长任务交接模板。 |
| `react-frontend-migration-plan.md` | 当前任务书 | React 官网 + SaaS 后台一体化前端迁移阶段计划、事实证据和完成状态。 |
| `progressive-project-audit.md` | 历史任务书 | 渐进式审计记录。先看开头状态说明，不要把全文当作当前事实。 |
| `admin-template-parity.md` | 历史任务书 | 外部后台能力对齐记录。只作为证据和决策来源。 |
| `server-status-dashboard-refactor-plan.md` | 历史任务书 | Server Status Dashboard 重构记录。当前维护入口见 `docs/maintenance/server-status-dashboard.md`。 |
| `generator-product-spec.md` | 规格草案 | 模板/代码/表单/导出生成器的产品与安全边界；当前仍非运行时能力。 |

## 使用顺序

1. 先读仓库根目录 `AGENTS.md`。
2. 再读 `docs/README.md` 和本目录的 `project-map.md`。
3. 如果任务跨配置、部署、API、WebUI 或安全边界，读取对应主题文档。
4. 只有在处理历史切片或需要证据链时，才进入历史任务书。
5. 临时分析报告写入 `tmp/ai`；确认要长期保留的事实再提升到 `docs`。

## 当前架构提醒

- `pkg` 是基础设施实现层，不依赖 `internal/app` 或业务模块。
- `internal/app` 是装配根，负责初始化基础设施、模块装配和生命周期管理。
- 命令行主进程使用 `App.Run -> lifecycleapp.Run -> HTTPServer.Wait` 等待 HTTP 运行期错误；需要非阻塞启动时使用 `App.Start(ctx)`。
- 业务模块保留现有 `model/repository/service/handler` 包名：`model` 视为 domain，`service` 视为 application，`handler` 视为 adapter，`repository` 和 `infrastructure` 视为 infrastructure implementation。
- service 层依赖本包定义的最小接口，通过构造函数注入；生产 service 不直接导入同模块 repository、`pkg/*` 或 `internal/ports`。
- 历史文档里出现的旧依赖图、旧任务状态或未实现设想，仅作为上下文，不应覆盖当前代码事实。

## 常用验证

```powershell
go test ./internal/... -count=1 -mod=readonly
go test ./... -count=1 -mod=readonly
go vet ./...
go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi
rg -n "github\\.com/rei0721/go-scaffold/pkg/" internal/modules internal/middleware internal/transport --glob "*.go" --glob "!**/*_test.go"
rg -n "http\\.Client|smtp\\.|os\\.Getenv|database\\.New|WithExecutor|internal/modules/.*/repository" internal/modules/*/service --glob "*.go"
```

# 已知缺口

本文记录已知缺口，避免未来工作被遗漏，也避免把未完成能力描述成已完成能力。

## 实现漂移

当前没有待修复的实现漂移项。

## 文档债务

公共包 README、AI 入口和渐进式审计仍按任务触达范围持续维护。

## 已修复记录

| 日期 | 缺口 | 修复与验证 |
| --- | --- | --- |
| 2026-06-15 | HTTP 服务启动后的异步 `Serve` 错误无法传播到 `cmd/aoi` | `pkg/httpserver.HTTPServer` 新增 `Wait(ctx)`；`App.Run()` 通过 `lifecycleapp.Run` 等待 HTTP 运行期结果，`cmd/aoi` 既有错误通道可接收后续服务错误。聚焦验证：`go test ./pkg/httpserver ./internal/app/lifecycleapp ./cmd/aoi -count=1 -mod=readonly`。 |
| 2026-06-15 | `pkg/httpserver` README/Godoc 与导出 API 漂移 | README 和 Godoc 已补充 `Wait` 用法，并移除当前不存在的 `SetExecutor`/`WithExecutor` 示例。 |
| 2026-06-15 | `docs/ai` 历史任务书和渐进式审计入口需要本轮状态记录 | `docs/ai/README.md`、`project-map.md` 和 `progressive-project-audit.md` 已记录 HTTP `Run/Start/Wait` 当前事实和本轮阶段记录；历史正文继续只作证据来源。 |

## 产品规格缺口

| 缺口 | 影响 | 建议动作 |
| --- | --- | --- |
| 模板配置、代码生成、表单生成和导出模板仍只有离线工具能力，没有后台产品规格 | 直接实现后台页面会绕过写入目录、覆盖策略、字段映射、权限、审计和回滚设计 | 先从 `docs/ai/generator-product-spec.md` 回答产品问题、统一配置方案和安全门禁，再完成外部工作流/源码研究，最后决定是否扩展 `pkg/sqlgen`、`pkg/yaml2go`、新增后端 API、统一配置和 WebUI 页面。 |

## 生产就绪缺口

| 缺口 | 影响 |
| --- | --- |
| 已有 goose 封装迁移执行器，但生产发布/回滚治理仍需流程化 | 迁移能力可执行，不等于发布窗口、回滚演练和审计证据完备。 |
| 没有内置插件进程编排、插件市场和在线安装打包系统 | 现有 Plugins 模块只负责远程插件宿主管理视图，插件进程仍需外部系统管理。 |
| 插件 registry watcher 已有 memory 即时通知和 SQL polling 实现，但尚未接入 Redis/etcd/Consul/NATS 等生产级外部推送后端 | 当前多 Host 可以通过共享 DB polling 感知状态；低延迟、大规模集群和跨机房一致性仍需后续后端实现。 |
| 插件审计/观测已有统一 recorder 抽象，但尚未接入持久审计表、指标后端或链路追踪 | 当前 Host 会产生操作事件；生产环境仍需要在组合根接入具体 sink、采样策略、脱敏策略和告警规则。 |
| 未来所有业务通过远程插件实现尚未设计迁移路线 | 当前只建立 `pkg/plugin`、`pkg/plugin/injection`、`internal/plugin` 和 `docs/api/plugin-protocol` 契约边界；`pkg/plugin/protocol` 仅为主系统内部模型，IAM、System 等现有业务仍是主系统模块。 |
| 发布/回滚证据还不是稳定 v1 流程 | v1 发布仍需要独立验收工作。 |

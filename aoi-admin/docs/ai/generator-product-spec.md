# 生成器产品规格门禁

> 规格草案：本文只用于约束后续可能的生成器产品化工作。当前系统没有后台代码生成、表单生成或导出模板运行时能力。

状态：`draft/gated`，2026-06-13 本地审计后建立。本文不是实现说明，
也不表示后台生成器已经启用；它用于阻止后续工作在没有产品边界和安全边界
时直接添加页面、接口、配置或文件写入能力。

关联任务书：`docs/ai/admin-template-parity.md`。

## 当前事实

- 本地已存在 `pkg/sqlgen` 和 `pkg/yaml2go`，定位是离线/开发期工具。
- `pkg/sqlgen` 被 `db` CLI 和 Demo schema 预览流程使用，不是用户可见后台
  工作流。
- `pkg/yaml2go` 只返回生成代码，不负责写文件。
- 当前没有生成器后台路由、后端 API、菜单、权限、运行时配置、Nuxt runtime
  config 或静态 WebUI 产物路径。
- 后台前端 API 入口应继续集中在 `useAdminApi()` 和
  `web/admin/app/config/admin-api.ts`；静态托管路径仍由 Go `webui.mount_path`
  与 Nuxt `NUXT_APP_BASE_URL` 对齐。

## 本阶段不实现

- 不新增 `internal/modules` 运行时模块。
- 不新增数据库迁移、权限码、菜单项、API catalog 或后台页面。
- 不新增 `configs/*.example.yaml`、`.env.example`、Nuxt runtime config 或构建
  参数。
- 不允许后台用户输入直接决定源代码写入路径。
- 不把生成产物写入 `cmd`、`internal`、`pkg`、`types`、`web/admin` 或
  `configs`，除非未来切片先批准明确的写入策略、回滚策略和代码审查流程。

## 必答产品问题

后续任何实现前，必须先在任务书中回答这些问题：

1. 生成器只做预览/导出，还是允许应用到仓库文件？
2. 支持哪些产物类型：Go model、repository、service、handler、SQL migration、
   Nuxt page、表单 schema、导出模板，还是只支持其中一部分？
3. 字段映射的来源是什么：数据库表、DDL、YAML、JSON schema、人工表单，还是
   多来源合并？
4. 字段类型、校验规则、状态枚举、默认值、列表筛选、表单布局和文案如何统一
   表达，避免页面内散落硬编码？
5. 生成结果的输出形式是什么：一次性 zip、可下载 diff、临时预览目录，还是
   受控写入工作区？
6. 覆盖已存在文件时如何处理：拒绝、生成冲突报告、显式覆盖，还是只允许追加？
7. 谁可以执行预览、导出和应用写入？权限码、审计日志和失败响应如何设计？
8. 生成产物如何回滚，尤其是迁移文件、菜单/API 权限和前端路由的组合变更？

## 统一配置原则

当前没有运行时能力，因此不新增配置项。若未来批准后台生成器，配置必须先
进入 `internal/config`，再同步示例配置、环境变量、配置文档和测试。

候选配置需要覆盖下列维度，最终名称以实现切片的配置评审为准：

- `generator.enabled`：总开关，默认关闭。
- `generator.workspace_root`：允许生成器读取/写入的工作区根目录。
- `generator.allowed_output_roots`：写入目录白名单，必须在服务端校验。
- `generator.max_input_bytes`、`generator.max_generated_files`：输入和产物数量
  阈值，禁止写死在 handler 或页面中。
- `generator.preview_retention_seconds`：预览产物保留时间。
- `generator.overwrite_policy`：覆盖策略，例如 `reject`、`diff-only`、
  `explicit-apply`。
- `generator.allowed_artifact_types`：允许生成的产物类型。
- `generator.export_enabled`、`generator.export_retention_seconds`：导出能力与
  清理策略。
- `generator.audit_enabled`：是否记录预览、导出和应用写入的审计日志。

接口路径、状态枚举、用户可见文案、字段映射、刷新策略和布局参数也必须进入
统一位置：后端使用配置或领域常量，前端使用共享 endpoint 配置、schema、Aoi
语义 token 和 `--aoi-admin-*` 变量，不能散落在单个页面组件里。

## 安全门禁

- 所有路径必须服务端 `clean` 后再与白名单根目录比较，拒绝绝对路径、目录穿越
  和符号链接逃逸。
- 默认只允许 dry-run 预览和导出，应用写入必须有单独权限、二次确认、审计记录
  和可复核 diff。
- 生成器服务层必须拥有最终决策权；handler 只负责解析请求、调用服务和返回
  统一响应。
- 前端只能提交结构化意图，不能提交任意本地路径或可执行脚本。
- 输入 schema、DDL、YAML 和模板内容都要限制大小、字段数量和递归深度。
- 生成结果不得包含本地密钥、`.env` 内容、访问令牌或用户上传的未脱敏样本。
- 迁移文件一旦共享必须 append-only；回滚能力需要独立设计和发布流程支持。
- 任何写入源码树的能力都必须在实现前说明如何与 git diff、代码审查和失败回滚
  配合。

## 架构草案

如果产品问题和安全门禁通过，推荐分层如下：

- `pkg/sqlgen`、`pkg/yaml2go` 继续保持纯工具库，不依赖 `internal`。
- 新增运行时能力时再考虑 `internal/modules/generator`，并沿用现有
  `model/repository/service/handler` 包名；service 定义本地接口，由 app 注入
  repository/infrastructure 具体实现。
- 生成预览、导出、应用写入和清理由 service 统一编排。
- 路由注册进入 `internal/transport/http`，权限进入 IAM/API catalog/menu 的同一
  切片。
- WebUI 页面只在后端 API 和权限模型稳定后添加，接口集中到
  `web/admin/app/config/admin-api.ts`。
- 样式使用现有 Aoi token 和 `--aoi-admin-*` 变量；新增状态色、密度、间距和
  响应式断点必须先进入共享样式层。

## 验证门禁

- 后端：覆盖配置校验、路径白名单、大小阈值、权限判断、审计记录、预览导出和
  失败响应的单元/集成测试。
- 前端：`pnpm typecheck`，可见页面还要用 Browser 检查 `1440x900` 和
  `390x844`。
- 构建：如果新增静态产物或 Nuxt baseURL 相关行为，必须验证 `pnpm generate`
  的 `.output/public` 与 Go `webui.dist_dir`、`webui.mount_path` 对齐。
- 文档：同步更新任务书、配置说明、示例配置、用户说明和变更记录。

## 变更记录

- 2026-06-13：建立规格门禁。结论是当前只完成本地审计，不实现运行时后台生成器；
  后续必须先补外部工作流/源码证据、产品问题答案和安全评审。

# 渐进式项目审计任务书

> 历史记录：本文保留长期审计过程和证据索引。执行新任务时先读当前主线文档，再按需参考本文，不要把历史状态或旧依赖图当作当前事实。

状态：`active`，2026-06-13 建立。本文是全项目长期任务书，用来约束后续每
一个实现切片：先读证据、再写计划、再小步修改、最后补齐文档和验证记录。

关联文档：

- `docs/ai/project-map.md`：架构和扩展边界。
- `docs/structure/directory-map.md`：目录职责。
- `docs/environment/configuration.md`：配置字段、环境变量和 WebUI 挂载说明。
- `docs/runtime/config-flow.md`、`docs/runtime/http-flow.md`：运行时配置和 HTTP
  调用链。
- `docs/build/docker-and-ci.md`：构建、CI 和静态产物生成。
- `web/admin/AGENTS.md`、`web/admin/design/rules.md`：后台前端实现规则。
- `docs/ai/admin-template-parity.md`：外部后台平替切片任务书。

## 每轮必查证据

开始任何实现前，先确认下面证据是否仍然成立。若切片不涉及某一项，也要在本
任务书或切片计划中说明为什么不涉及。

| 主题 | 必查入口 | 维护原因 |
| --- | --- | --- |
| 项目结构 | `AGENTS.md`、`docs/ai/project-map.md`、`docs/structure/directory-map.md` | 防止把 AI 产物、业务逻辑或可复用基础设施放错层。 |
| 配置体系 | `docs/environment/configuration.md`、`docs/runtime/config-flow.md`、`internal/config/*` | 所有阈值、路径、状态、刷新策略和运行时开关必须先进入统一配置或共享常量。 |
| 后端调用链 | `docs/runtime/http-flow.md`、`internal/transport/http/router.go`、目标模块的 handler/service/repository | handler 只处理请求和响应，业务规则、事务、权限和状态判断落在 service。 |
| 前端调用链 | `web/admin/AGENTS.md`、`web/admin/app/composables/useAdminApi.ts`、`web/admin/app/config/admin-api.ts` | 后台 API 统一走 `useAdminApi()`，endpoint 集中维护，不在页面内散落路径。 |
| 构建流程 | `docs/build/docker-and-ci.md`、`Dockerfile`、`web/admin/package.json` | Go 构建、Nuxt typecheck/generate 和 Docker build 的职责不同，不能互相替代。 |
| 静态资源加载 | `internal/config/app_webui.go`、`internal/transport/http/router.go`、`pkg/web/web.go`、`web/admin/nuxt.config.ts` | `webui.mount_path`、`webui.dist_dir` 和 `NUXT_APP_BASE_URL` 必须保持同一部署语义。 |
| 样式体系 | `web/admin/design/rules.md`、`web/admin/app/assets/css/tokens.css`、`web/admin/app/assets/css/main.css` | 新颜色、间距、布局尺寸、状态样式和响应式参数优先进入共享 token 或 `--aoi-admin-*`。 |
| 相关文档 | `docs/README.md`、目标模块文档、`docs/backlog/known-gaps.md`、本任务书 | 防止实现和文档漂移，保留下一轮可续接证据。 |

## 实现前计划模板

每个切片开工前，必须在对应任务书或本文中记录：

1. 本轮要分析什么：列出已读文件、命令和当前事实。
2. 本轮要修改什么：列出文件范围、配置范围、文档范围和验证范围。
3. 本轮不修改什么：明确不碰的运行时代码、配置、示例配置、生成物或 UI。
4. 统一配置落点：说明新增或调整的阈值、状态、文案、颜色、接口、路径、刷新
   策略、布局参数、字段映射和构建产物路径在哪里维护。
5. 风险：说明安全、权限、迁移、静态资源路径、前后端契约、响应式和兼容性风险。
6. 验证：先列最近的 focused check，再列需要扩展到全量检查或 Browser 检查的条件。

## 统一配置规则

- 运行时配置新增时，先改 `internal/config` 结构、标签、默认值和校验，再同步
  `configs/config.example.yaml`、`.env.example`、生产示例和配置文档。
- 前端运行时配置只能放在 Nuxt public runtime config；构建期路径必须说明是否
  需要重新 `pnpm generate` 或重新构建 Docker 镜像。
- 后台 endpoint 集中放在 `web/admin/app/config/admin-api.ts` 或目标共享配置，不在
  Vue 页面内写散落路径。
- 前端文案如果是共享用户可见文案，按 `web/admin/AGENTS.md` 同步 locale；窄改
  保留内联文案时要在切片计划说明。
- 样式颜色、间距、布局尺寸、滚动高度、状态色和响应式参数优先进入
  `tokens.css`、`main.css` 或目标共享 UI 配置。
- 没有运行时能力的草案只能进入 `docs/ai` 或 `docs/backlog/known-gaps.md`，不能
  预先添加无效 YAML/env/Nuxt 配置。

## 当前审计状态

| 区域 | 状态 | 最近证据 | 下一步 |
| --- | --- | --- | --- |
| 总流程任务书 | `[done]` | 2026-06-13 读取结构、配置、HTTP、构建、静态托管、样式和相关文档后建立本文 | 后续每轮维护本表或对应切片任务书。 |
| 外部后台平替 | `[active]` | `docs/ai/admin-template-parity.md` 已记录多个已完成切片和生成器门禁 | 继续按切片做外部工作流/源码研究和本地实现。 |
| 生成器/模板配置 | `[gated]` | `docs/ai/generator-product-spec.md` 明确当前无运行时能力 | 先完成产品问题、安全门禁和外部工作流研究。 |
| 配置治理 | `[audit]` | 2026-06-13 对照 `internal/config` 的 99 个 `envname` 标签、`.env.example`、配置说明和生产 Compose | 下一次触碰配置时补测试、示例和文档；生产 Compose 是否全量暴露变量需单独部署审计。 |
| WebUI 静态托管 | `[audit]` | 2026-06-13 对照 `internal/config/app_webui.go`、`router.go`、`pkg/web/web.go`、`nuxt.config.ts`、`Dockerfile` 和 `deploy.sh`；部署脚本已与 Go 配置统一拒绝根挂载路径 `/` | 任何挂载路径或构建产物变化都要重新验证 `pnpm generate` 与 Go 挂载。 |
| 前端 API endpoint | `[audit]` | 2026-06-13 对照 `web/admin/app/composables/useAdminApi.ts` 和 `web/admin/app/config/admin-api.ts`；API 调用路径已集中到 `ADMIN_API_ENDPOINTS` | 新增后台 API 时先补 endpoint 配置，再由 `useAdminApi()` 调用。 |
| 后端 API 路径契约 | `[audit]` | 2026-06-13 新增 `types/constants/http.go`，并让 `router.go`、`app_webui.go`、媒体上传和断点上传 URL 生成复用同一 API 路径契约 | 后续新增后端生成 URL 或 API catalog 路径判断时先扩展共享契约，再同步前端 endpoint 文档。 |
| HTTP 服务运行期错误传播 | `[done]` | 2026-06-15 新增 `HTTPServer.Wait`，`App.Run` 通过 `lifecycleapp.Run` 等待 HTTP 异步 `Serve` 错误；`pkg/httpserver` README/Godoc 已移除过期 Executor 示例 | 后续如需同类能力，再单独评估 RPC server 的异步错误传播契约。 |
| 前端自动刷新策略 | `[audit]` | 2026-06-13 新增 `admin-auto-refresh.ts`，`useAdminAutoRefresh()`、`AdminAutoRefreshControls.vue`、`server-info.vue` 与 `server-status-dashboard.ts` 已改用共享配置/页面配置 | 后续新增自动刷新页面时先复用通用配置；页面级默认开关或间隔必须显式来自页面配置。 |
| 手动刷新冷却策略 | `[audit]` | 2026-06-13 `manualCooldownMs` 已进入通用配置类型并由 `useAdminAutoRefresh()` 对真实点击刷新生效；服务器状态页显式传入页面级冷却值 | 后续调整冷却策略时保持“点击冷却不阻塞程序化/静默刷新”的契约。 |
| 自动刷新控件响应式样式 | `[audit]` | 2026-06-13 已移除 `AdminAutoRefreshControls.vue` 局部 `680px` 断点；控件继续依赖共享 gap/height token 与 flex wrapping | 后续如需全局断点 token，需要样式体系切片统一设计，不在单个控件内临时新增。 |
| 前端页面语法恢复 | `[audit]` | 2026-06-13 修复 `apis.vue`、`menus.vue`、`organizations.vue`、`system.vue`、`operation-records.vue` 中截断字符串/标签，并收敛事件类型 | 后续若终端显示 mojibake，先做 UTF-8/编译检查，不要大段重写页面文案。 |
| 样式体系 | `[audit]` | `web/admin/design/rules.md` 与 `main.css` 已集中 `--aoi-admin-*` 参数 | 可见 UI 切片必须先声明 token/变量落点并做 Browser 检查。 |

## 2026-06-13 阶段记录

本轮分析：

- 读取 `AGENTS.md`、`docs/ai/project-map.md`、`docs/structure/directory-map.md`。
- 读取 `docs/runtime/config-flow.md`、`docs/runtime/http-flow.md`、
  `docs/environment/configuration.md`。
- 读取 `docs/build/docker-and-ci.md`、`web/admin/AGENTS.md`、
  `web/admin/design/rules.md`。
- 读取静态托管实现：`internal/config/app_webui.go`、
  `internal/transport/http/router.go`、`pkg/web/web.go`、`web/admin/nuxt.config.ts`。
- 读取前端构建和 endpoint 入口：`web/admin/package.json`、
  `web/admin/app/config/admin-api.ts`。

本轮修改：

- 新增本文作为全项目渐进式审计任务书。
- 更新 AI 索引、配置说明和已知缺口，让下一轮能从同一入口继续。

本轮不修改：

- 不修改 Go 运行时代码、Nuxt 代码、`configs/config.yaml`、示例配置、迁移、
  `.output`、`.nuxt`、`tmp` 或其他生成物。
- 不新增运行时配置、后台路由、前端页面、权限码、菜单或构建参数。

风险记录：

- 本文是流程和证据索引，不是功能完成证明；全项目审计仍然是长期 active 状态。
- 后续切片如果涉及 UI 或运行时行为，必须补 focused test、必要全量检查和 Browser
  桌面/移动端证据。

验证记录：

- 本轮为文档任务书更新，运行文档空白和引用检索即可。
- Go/Nuxt/Browser 验证留给实际运行时或可见 UI 变更切片。

## 2026-06-13 后端 API 路径契约计划

本轮要分析：

- 已读取结构与边界：`AGENTS.md`、`docs/ai/project-map.md`、`docs/structure/directory-map.md`。
- 已读取配置与运行链路：`docs/environment/configuration.md`、`docs/runtime/config-flow.md`、
  `docs/runtime/http-flow.md`、`internal/config/app_webui.go`。
- 已读取前端与构建/静态约束：`web/admin/AGENTS.md`、`web/admin/design/rules.md`、
  `web/admin/package.json`、`web/admin/nuxt.config.ts`、`docs/build/docker-and-ci.md`、
  `pkg/web/web.go`。
- 已读取目标代码：`internal/transport/http/router.go`、
  `internal/modules/system/service/media.go`、
  `internal/modules/system/service/media_resumable.go`、`types/constants/*`。
- 当前事实：前端 endpoint 已集中在 `web/admin/app/config/admin-api.ts`，但后端
  route catalog、操作记录过滤和媒体资产本地下载 URL 仍直接引用或拼接 `/api/v1`。

本轮要修改：

- 在共享常量层新增稳定 HTTP API 路径契约，提供公共前缀、前缀判定、路径分组
  截取和媒体下载 URL 构造。
- 让 `internal/transport/http/router.go` 使用共享契约处理 API 前缀、公共路由、
  API catalog 分组/描述/权限映射和操作记录判定。
- 让媒体库普通上传与断点上传完成后的本地下载 URL 由同一共享构造函数生成。
- 同步更新本任务书、配置说明或模块文档，说明这些路径不是运行时 YAML/env，
  而是前后端共享的外部 API 契约常量。

本轮不修改：

- 不改变任何现有 HTTP 路由、OpenAPI 路径、权限码、菜单、数据库迁移或前端页面。
- 不修改 `configs/config.yaml`、示例配置、Nuxt runtime config、WebUI mount path、
  `.nuxt`、`.output` 或其他生成物。
- 不把 API 前缀改成运行时可配置项；当前 `/api/v1` 是公开契约，若未来要可配置，
  需先设计 OpenAPI、前端 endpoint、API catalog、权限同步和部署兼容策略。

统一配置落点：

- 后端公共 API 前缀和服务端生成的媒体下载 URL 放入 `types/constants` 作为共享契约。
- 前端调用路径仍由 `web/admin/app/config/admin-api.ts` 维护；本轮不会让前端直接
  依赖 Go 常量，也不会新增跨语言生成流程。

风险：

- API catalog 权限识别依赖路径字符串匹配；集中前缀时必须保持结果完全不变。
- 媒体下载 URL 已持久化到媒体资产记录，新构造函数必须生成相同字符串，避免新旧
  记录展示不一致。
- 路由前缀不是运行时配置，本轮若误改外部路径会破坏文档、OpenAPI、前端和测试。

验证：

- 聚焦运行 `go test ./types/constants ./internal/transport/http ./internal/modules/system/service -count=1 -mod=readonly`。
- 若后续实际改变路由、响应形态或前端工作流，再扩展到 `go test ./...`、OpenAPI
  对照和 Browser 桌面/移动端检查；本轮不产生可见 UI 变更，暂不运行 Browser。

完成记录：

- 新增 `types/constants/http.go`，集中 `HTTPHealthPath`、`HTTPReadyPath`、
  `APIPathRoot`、`APIBasePath`、`APIBasePrefix`、`APIPath()`、
  `IsAPIPath()`、`TrimAPIPathPrefix()` 和 `MediaAssetDownloadPath()`。
- `internal/transport/http/router.go` 改用共享契约注册探针/API 前缀，并用于操作
  记录、API catalog 过滤、分组、描述和权限映射。
- `internal/config/app_webui.go` 的保留路径使用同一 API/probe 常量，保持静态
  WebUI 挂载校验与路由契约一致。
- `internal/modules/system/service/media.go` 与 `media_resumable.go` 用
  `MediaAssetDownloadPath()` 生成本地媒体下载 URL，普通上传和断点上传不再各自
  拼接路径。
- 同步更新 `docs/environment/configuration.md`、`docs/runtime/http-flow.md` 和
  `docs/modules/system.md`，说明 `/api/v1` 是公开 API 契约常量，不是运行时
  YAML/env/Nuxt 配置。

验证记录：

- `go test ./types/constants ./internal/transport/http ./internal/modules/system/service -count=1 -mod=readonly` 通过。
- `go test ./internal/config -count=1 -mod=readonly` 通过。
- `go test ./types/... -count=1 -mod=readonly` 通过，确认新增共享常量没有破坏 `types`
  导入边界。
- `rg -n --fixed-strings "/api/v1" internal/transport/http/router.go internal/config/app_webui.go internal/modules/system/service/media.go internal/modules/system/service/media_resumable.go types/constants` 结果只剩共享常量注释和测试断言。
- `git diff --check` 退出成功；仍提示既有 `web/admin/design/rules.md` CRLF 将被 Git
  转为 LF。

## 2026-06-13 前端自动刷新策略计划

本轮要分析：

- 已读取结构与边界：`AGENTS.md`、`docs/ai/progressive-project-audit.md`、`docs/ai/project-map.md`、
  `docs/structure/directory-map.md`、`web/admin/AGENTS.md`、`web/admin/design/rules.md`。
- 已读取配置、运行链路、构建和静态资源：`docs/environment/configuration.md`、
  `docs/runtime/config-flow.md`、`docs/runtime/http-flow.md`、`docs/build/docker-and-ci.md`、
  `internal/config/app_webui.go`、`pkg/web/web.go`、`web/admin/package.json`、`web/admin/nuxt.config.ts`。
- 已读取目标前端链路：`web/admin/app/composables/useAdminAutoRefresh.ts`、
  `web/admin/app/components/AdminAutoRefreshControls.vue`、`web/admin/app/config/server-status-dashboard.ts`、
  `web/admin/app/utils/serverStatusDashboard.ts` 以及已接入自动刷新的后台页面。
- 当前事实：自动刷新已经被多处页面复用，但通用间隔、最小间隔、计时 tick、状态文案、时间单位和
  控件 label 仍散落在 composable/组件中；服务器状态页有 `refresh.autoEnabled` 页面配置，
  但当前页面没有把它传给 `useAdminAutoRefresh()`。

本轮要修改：

- 新增 `web/admin/app/config/admin-auto-refresh.ts`，集中通用自动刷新默认启用状态、间隔、最小间隔、
  tick 间隔、单位换算、时间 locale 和共享控件/状态文案。
- 调整 `useAdminAutoRefresh()` 从共享配置读取默认值和文案，并保留页面传入 `defaultEnabled`、`intervalMs`
  的覆盖能力。
- 调整 `AdminAutoRefreshControls.vue` 从共享配置读取默认 label，并复用现有尺寸/间距 token。
- 调整 `server-info.vue` 显式传入 `dashboardConfig.refresh.autoEnabled`；为避免改变当前实际运行行为，
  同步让 `server-status-dashboard.ts` 的 `refresh.autoEnabled` 与现有默认开启行为一致。
- 同步更新本任务书、配置说明、系统模块说明、服务器状态使用说明和前端设计规则。

本轮不修改：

- 不批量改动已接入自动刷新的业务页面、接口路径、后端 DTO、菜单、权限、迁移、`configs/config.yaml`、
  Nuxt runtime config、`.nuxt`、`.output` 或其他生成物。
- 不把自动刷新策略提升为后端 YAML/env 配置；它只影响当前静态 Admin WebUI 交互。
- 不迁移全站 inline 中文到 i18n locale；本轮仅把自动刷新共享文案集中到前端配置，后续如做完整 i18n
  再统一迁移。

统一配置落点：

- 通用刷新策略、刷新状态文案、时间单位和时间 locale 落在 `web/admin/app/config/admin-auto-refresh.ts`。
- 服务器状态页自己的刷新开关和间隔仍落在 `web/admin/app/config/server-status-dashboard.ts`，页面只传入
  覆盖值，不直接写数字。
- 控件间距和高度分别落在 `--aoi-admin-auto-refresh-gap` 与 `--aoi-control-height-sm` token。

风险：

- 自动刷新已经在多个后台页面中使用，共享默认值变化会影响多个页面；本轮保持现有默认开启行为，只修正配置来源。
- `server-status-dashboard.ts` 的页面级 `autoEnabled` 之前是 false 但实际页面默认开启；同步为 true 是为了让配置
  反映当前行为，仍需在文档中说明需要调整默认开关时改配置。
- 部分前端文件在终端显示 mojibake，本轮避免大段重写页面文案，防止误伤编码。

验证：

- 运行 `pnpm typecheck` 验证 Vue/TypeScript 变更。
- 用 `rg` 检查 `useAdminAutoRefresh.ts` 和 `AdminAutoRefreshControls.vue` 中不再保留自动刷新相关硬编码文案和时间数字。
- 运行 `git diff --check`；如仅出现既有 CRLF 提示，则记录为残余风险。

完成记录：

- 新增 `web/admin/app/config/admin-auto-refresh.ts`，集中通用自动刷新默认开启状态、间隔、最小间隔、
  tick 间隔、秒/分钟单位、时间 locale 和共享控件/状态文案。
- `useAdminAutoRefresh()` 改为读取共享配置，并通过 `resolveAutoRefreshLoadOptions()` 忽略 Vue 事件、
  watch 新值等非配置参数，只从显式 `{ silent: true }` 进入静默刷新。
- `AdminAutoRefreshControls.vue` 的默认 label 来自共享配置；控件 gap 由
  `--aoi-admin-auto-refresh-gap` 控制，高度复用 `--aoi-control-height-sm`。
- `server-info.vue` 显式传入 `dashboardConfig.refresh.autoEnabled` 和 `refresh.intervalMs`；
  `server-status-dashboard.ts` 的 `refresh.autoEnabled` 调整为 true，以反映本轮前页面实际默认开启的行为。
- `AoiTextField` 的 `enter` 明确为无参数命令事件，需要原始键盘事件时继续使用 `keydown`。
- 同步更新 `docs/environment/configuration.md`、`docs/modules/system.md`、
  `docs/onboarding/server-status-dashboard.md` 和 `web/admin/design/rules.md`。

验证记录：

- `pnpm typecheck` 通过。
- `rg` 检查确认 `useAdminAutoRefresh.ts` 与 `AdminAutoRefreshControls.vue` 不再保留自动刷新文案、时间数字、
  `label="自动刷新"`、`gap: 8px` 或 `min-height: 32px` 等散落硬编码；这些值已进入配置或 token。
- Browser 桌面 `1440x900` 打开 `/admin/operation-records`，未登录状态按预期重定向到
  `/admin/login?redirect=/operation-records`，页面正文无 Vite 编译错误。
- Browser 移动 `390x844` 打开 `/admin/apis`，未登录状态按预期重定向到
  `/admin/login?redirect=/apis`，页面正文无 Vite 编译错误。
- 浏览器控制台仍保留一条修复前的旧动态导入错误日志；检查依据以修复后的页面正文、typecheck 和重新导航结果为准。

## 2026-06-13 前端页面语法恢复计划

本轮要分析：

- 已读取 `pnpm typecheck` 输出，当前阻塞点集中在 `web/admin/app/pages/apis.vue`、
  `menus.vue`、`organizations.vue`、`system.vue` 的字符串语法错误。
- 已读取目标文件报错行和 `git diff --`，确认这些页面已有并行改动引入自动刷新逻辑，同时部分用户可见中文文案
  被截断，导致缺少结束引号或结束标签。
- 已沿用本轮已读结构、配置、前端调用链路、构建流程、静态资源加载方式、样式体系和文档证据；该修复不涉及
  后端配置、静态托管或构建产物路径。

本轮要修改：

- 只修复 `apis.vue`、`menus.vue`、`organizations.vue`、`system.vue` 中被截断的字符串字面量和模板文本标签。
- 保留这些页面现有自动刷新逻辑、接口调用、过滤条件、分页和页面结构。
- 在本任务书记录 typecheck 阻塞来源和修复范围。

本轮不修改：

- 不恢复或重排这些页面的自动刷新接线，不改接口 endpoint、DTO、后端、权限、菜单、样式 token 或配置示例。
- 不把整页 mojibake 文案做大规模重写；仅处理造成语法损坏的截断点。

统一配置落点：

- 本轮不新增配置。自动刷新相关默认值仍由 `web/admin/app/config/admin-auto-refresh.ts` 维护。

风险：

- 这些页面在本轮之前已经处于脏工作区，修复时必须避免覆盖其逻辑改动。
- 终端可能显示 mojibake，修复截断文案时以 TypeScript/Vue 语法完整性和页面语义为准。

验证：

- 重新运行 `pnpm typecheck`。
- 如仍有新暴露的同类截断点，继续按同一范围小步修复；若出现非本轮范围的类型错误，则记录为残余风险。

完成记录：

- 修复 `apis.vue`、`menus.vue`、`organizations.vue`、`system.vue` 中被截断的字符串字面量、结束标签和模板文案，
  恢复 TypeScript/Vue 语法完整性。
- Browser 检查继续暴露 `operation-records.vue` 中同类截断点后，按同一范围恢复标题描述、筛选 label、
  翻页文案、表头、空状态和结束标签。
- 为避免批量页面 wrapper，调整 `AoiTextField` 的 `enter` 类型为无参数命令事件；保留 `keydown` 继续暴露
  `KeyboardEvent`。
- 保留这些页面既有自动刷新接线、接口调用、筛选、分页、删除等业务逻辑，不做结构性改写。

验证记录：

- `rg -n -F -e '�' -e '?/' -e '?>' web/admin/app/pages` 无结果。
- `pnpm typecheck` 通过。
- Browser 桌面/移动重定向检查未再出现 Vite 编译错误正文；受登录态限制，未进入受保护业务页做数据态截图。

## 2026-06-13 手动刷新冷却策略计划

本轮要分析：

- 已读取结构与边界：`AGENTS.md`、`docs/ai/project-map.md`、`docs/structure/directory-map.md`、
  `web/admin/AGENTS.md`、`web/admin/design/rules.md`。
- 已读取配置、构建、静态托管和运行链路：`docs/environment/configuration.md`、
  `docs/runtime/config-flow.md`、`docs/runtime/http-flow.md`、`docs/build/docker-and-ci.md`、
  `internal/config/app_webui.go`、`pkg/web/web.go`、`web/admin/nuxt.config.ts`。
- 已读取目标前端链路：`web/admin/app/config/admin-auto-refresh.ts`、
  `web/admin/app/composables/useAdminAutoRefresh.ts`、`AdminAutoRefreshControls.vue`、
  `web/admin/app/config/server-status-dashboard.ts` 和 `web/admin/app/pages/server-info.vue`。
- 当前事实：`server-status-dashboard.ts` 暴露 `refresh.manualCooldownMs=1_000`，文档也声明手动刷新冷却由
  Dashboard 配置控制；但 `useAdminAutoRefresh()` 当前只按 `refreshing` 和 `blocked` 禁用刷新按钮。

本轮要修改：

- 将通用 `manualCooldownMs` 放入 `admin-auto-refresh.ts`，默认 0 以保持其他页面现有行为。
- 扩展 `useAdminAutoRefresh()` 支持 `manualCooldownMs` 覆盖；仅当 `refreshNow` 收到真实浏览器事件时应用手动冷却。
- 让 `server-info.vue` 传入 `dashboardConfig.refresh.manualCooldownMs`，使服务器状态页页面配置真正生效。
- 同步更新任务书、配置说明、System 模块说明、服务器状态使用说明和前端设计规则。

本轮不修改：

- 不批量改动其他已接入自动刷新的页面，不改变接口路径、后端 DTO、菜单、权限、迁移、Nuxt runtime config、
  WebUI 静态托管路径或生成物。
- 不让冷却阻塞自动静默刷新、页面筛选/分页后的程序化刷新或数据变更后的主动 reload。
- 不新增用户可见倒计时文案；本轮只实现禁用策略，避免扩大共享文案和 i18n 范围。

统一配置落点：

- 通用默认冷却落在 `web/admin/app/config/admin-auto-refresh.ts`。
- 服务器状态页覆盖值落在 `web/admin/app/config/server-status-dashboard.ts` 的 `refresh.manualCooldownMs`。
- 按钮禁用状态继续由 `useAdminAutoRefresh().refreshDisabled` 暴露给页面，不在页面内写冷却判断。

风险：

- `refreshNow()` 被 Vue click、mounted、watch、筛选、分页和 mutation 后 reload 共用；冷却必须只识别真实用户点击事件，
  否则会误跳过程序化刷新。
- 冷却依赖 `now` 的定时 tick 更新禁用状态；当前 tick 已集中在自动刷新配置，冷却显示最多有一个 tick 的禁用解除延迟。
- 受登录态限制，本轮 Browser 只能验证受保护路由未出现编译错误，不能直接点击服务器状态页刷新按钮。

验证：

- 运行 `pnpm typecheck`。
- 用 `rg` 确认 `manualCooldownMs` 不再只停在页面配置/文档，且没有页面内新增冷却硬编码。
- Browser 桌面/移动重新打开受影响后台路由，确认没有 Vite 编译错误；如无法进入受保护页，在记录中说明登录态限制。

完成记录：

- `web/admin/app/config/admin-auto-refresh.ts` 增加通用 `manualCooldownMs`，默认值为 0，保持非页面覆盖场景的现有行为。
- `useAdminAutoRefresh()` 支持 `manualCooldownMs` 覆盖，并通过 `resolveAutoRefreshRequest()` 区分真实浏览器点击、
  静默自动刷新和程序化刷新；只有真实点击会进入手动冷却。
- `useAdminAutoRefresh()` 增加冷却边界定时刷新，避免 1000ms 这类短冷却被 `clockTickMs` 粒度拉长。
- `server-info.vue` 显式传入 `dashboardConfig.refresh.manualCooldownMs`，使服务器状态页配置字段驱动按钮禁用状态。
- 同步更新配置说明、System 模块说明、服务器状态使用说明和前端设计规则。

验证记录：

- `pnpm typecheck` 通过。
- `rg -n -F -e 'manualCooldownMs' web/admin/app docs web/admin/design/rules.md` 确认冷却字段已进入
  `admin-auto-refresh.ts`、`useAdminAutoRefresh.ts` 和 `server-info.vue`，不再只停留在页面配置/文档。
- `rg -n -F -e 'cooldown' -e '冷却' web/admin/app/pages web/admin/app/components web/admin/app/composables web/admin/app/config`
  未发现业务页面内新增冷却判断；命中仅为 composable 注释和既有 `AoiScrollScene` 局部滚动冷却。
- Browser 桌面 `1440x900` 全新标签打开 `/admin/server-info`，未登录状态按预期重定向到
  `/admin/login?redirect=/server-info`，页面正文无 Vite 编译错误。
- Browser 移动 `390x844` 打开 `/admin/server-info`，未登录状态按预期重定向到
  `/admin/login?redirect=/server-info`，页面正文无 Vite 编译错误。
- 受登录态限制，未能直接点击服务器状态页刷新按钮验证 1000ms 禁用；本轮以 typecheck、配置接线和受保护路由编译检查作为证据。

## 2026-06-13 自动刷新控件响应式样式计划

本轮要分析：

- 已读取项目结构和边界：根目录、`docs/structure/directory-map.md`、`web/admin/AGENTS.md`
  以及 `web/admin/design/rules.md`。
- 已读取配置、运行链路、构建和静态资源：`docs/environment/configuration.md`、
  `docs/runtime/config-flow.md`、`docs/runtime/http-flow.md`、`docs/build/docker-and-ci.md`、
  `internal/config/app_webui.go`、`internal/transport/http/router.go`、`pkg/web/web.go`、
  `web/admin/package.json` 和 `web/admin/nuxt.config.ts`。
- 已读取目标前端链路：`web/admin/app/config/admin-auto-refresh.ts`、
  `web/admin/app/composables/useAdminAutoRefresh.ts`、`AdminAutoRefreshControls.vue`、
  `server-info.vue`、`AoiAdminCard.vue`、`main.css` 中的 `--aoi-admin-*` token。
- 当前事实：自动刷新控件的 gap 和高度已经使用 `--aoi-admin-auto-refresh-gap` 与
  `--aoi-control-height-sm`，但组件内还保留 `@media (max-width: 680px)` 断点。

本轮要修改：

- 移除 `AdminAutoRefreshControls.vue` 中仅用于把控件改为纵向排列的局部断点，改为依赖
  现有 `flex-wrap`、`min-width: 0` 和共享 gap token 自适应换行。
- 同步更新配置说明、服务器状态使用说明、前端设计规则和本任务书，说明自动刷新控件响应式策略不再依赖页面或组件内额外断点。

本轮不修改：

- 不批量调整 `AoiAdminCard.vue`、`server-info.vue` 或其他后台页面现存 `680px`、`760px`、
  `1080px` 断点；这些属于更大的样式体系审计。
- 不改变自动刷新间隔、手动冷却、文案、接口路径、权限、后端静态托管、Nuxt runtime config、
  `.nuxt`、`.output` 或其他生成物。

统一配置落点：

- 自动刷新控件间距继续由 `web/admin/app/assets/css/main.css` 的
  `--aoi-admin-auto-refresh-gap` 控制。
- 控件高度继续复用全局 `--aoi-control-height-sm`。
- 本轮不新增 CSS 断点 token。CSS 自定义属性不能直接用于 media query 条件；该控件可通过
  wrapping 消除局部断点，未来如需全局断点策略，应作为样式体系切片统一设计。

风险：

- 可见布局会从窄屏强制纵向改为按可用宽度自动换行，可能影响受保护页面头部动作区的视觉密度。
- 受登录态限制，Browser 可能只能验证受保护路由未出现编译错误，不能直接查看服务器状态页头部控件。
- 其他组件仍有局部断点，本轮不会消除全部响应式硬编码，只记录后续审计边界。

验证：

- 运行 `pnpm typecheck`。
- 用 `rg` 确认 `AdminAutoRefreshControls.vue` 不再包含 `max-width: 680px` 或局部 media query。
- 运行 `git diff --check`。
- Browser 桌面 `1440x900` 与移动 `390x844` 打开 `/admin/server-info`，至少确认未登录重定向页无 Vite 编译错误；如无法进入受保护页，在验证记录中说明。

完成记录：

- `AdminAutoRefreshControls.vue` 移除组件局部 `@media (max-width: 680px)`，保留 `flex-wrap`、
  `min-width: 0`、`--aoi-admin-auto-refresh-gap` 和 `--aoi-control-height-sm`。
- 配置说明、服务器状态入门文档、System 模块文档和前端设计规则已同步说明：自动刷新控件布局参数使用共享 token，窄屏优先依赖自然换行。
- 本轮不新增 CSS 断点配置；原因是 CSS 自定义属性不能直接用于 media query 条件，而该控件可以通过 wrapping 消除局部断点。

验证记录：

- `rg -n -F -e "max-width: 680px" -e "@media" web/admin/app/components/AdminAutoRefreshControls.vue`
  无命中。
- `pnpm typecheck` 通过。
- `git diff --check` 退出成功；仍提示若干既有 CRLF 将被 Git 转为 LF。
- Browser 桌面 `1440x900` 打开 `/admin/server-info`，最终重定向到
  `/admin/login?redirect=/server-info`，页面正文无 Vite 编译错误，console error 为空。
- Browser 移动 `390x844` 打开 `/admin/server-info`，重定向到
  `/admin/login?redirect=/server-info`，页面正文无 Vite 编译错误，console error 为空。
- 受登录态限制，本轮未能直接查看受保护服务器状态页头部里的自动刷新控件；残余视觉风险已记录。

## 2026-06-15 HTTP 服务运行期错误传播记录

本轮分析：
- 已读 `AGENTS.md`、`docs/backlog/known-gaps.md`、`pkg/httpserver`、`internal/app/lifecycleapp`、`internal/app/app.go`、`cmd/aoi/run.go`、`docs/runtime/startup-flow.md` 和 `docs/runtime/http-flow.md`。
- 当前事实：`cmd/aoi/run.go` 已等待 `application.Run()` 返回值，但 `App.Run()` 之前只做非阻塞启动；`pkg/httpserver` 已有内部错误通道，却没有公开等待运行期错误的接口。

本轮修改：
- `pkg/httpserver.HTTPServer` 新增 `Wait(ctx)`，异步 `Serve` 错误非阻塞投递并记录为最近运行期错误；优雅 `Shutdown` 让 `Wait` 返回 `nil`。
- `lifecycleapp.Run` 启动传输层后等待 HTTP 运行期结果；`App.Run()` 改用该入口，`App.Start(ctx)` 继续保持非阻塞启动。
- 同步更新 `pkg/httpserver` README/Godoc、运行时文档、AI 入口、项目地图和已知缺口记录。

本轮不修改：
- 不新增 YAML/env/Nuxt 配置、HTTP 路由、OpenAPI、权限、数据库迁移或 WebUI 页面。
- 不实现产品规格缺口、生产就绪缺口，也不扩展 RPC server 的异步错误传播接口。

验证记录：
- `go test ./pkg/httpserver ./internal/app/lifecycleapp ./cmd/aoi -count=1 -mod=readonly` 通过。
- 扩展验证记录以本轮最终验证命令为准。

## 变更记录

- 2026-06-15：完成 HTTP 服务运行期错误传播修复。`HTTPServer` 新增 `Wait(ctx)`，`pkg/httpserver` 会把异步 `Serve` 错误传回等待方，优雅关闭返回 `nil`；`App.Run()` 改为通过 `lifecycleapp.Run` 启动并等待 HTTP 运行期结果，`cmd/aoi` 既有错误通道可以接收后续服务错误。同步清理 `pkg/httpserver` README/Godoc 中不存在的 Executor 示例，并更新运行时文档、AI 入口、项目地图和已知缺口记录。
- 2026-06-13：完成自动刷新控件响应式样式收敛。`AdminAutoRefreshControls.vue` 不再保留组件局部
  `680px` 断点，继续通过共享 gap/height token 与 `flex-wrap` 自适应换行；相关配置说明、模块说明、
  入门文档和前端设计规则已同步。验证：`pnpm typecheck`、目标硬编码检索、`git diff --check`
  和 Browser 桌面/移动未登录重定向检查通过。
- 2026-06-13：完成手动刷新冷却策略落地。`manualCooldownMs` 从文档/页面配置扩展为
  `useAdminAutoRefresh()` 的真实点击冷却策略，服务器状态页显式传入页面级冷却值；程序化刷新和静默自动刷新
  不受点击冷却影响。验证记录见本节计划下方。
- 2026-06-13：完成前端自动刷新策略集中化审计。新增 `web/admin/app/config/admin-auto-refresh.ts`
  统一自动刷新默认值、时间单位、状态文案和控件文案；`useAdminAutoRefresh()` 读取共享配置并能安全忽略
  Vue 事件/watch 参数；`AdminAutoRefreshControls.vue` 读取共享 label，布局间距落入
  `--aoi-admin-auto-refresh-gap`；服务器状态页显式接入 `refresh.autoEnabled` 和 `refresh.intervalMs`。
  验证：`pnpm typecheck`、硬编码检索和 Browser 桌面/移动未登录重定向检查通过。
- 2026-06-13：修复前端页面语法恢复阻塞。`apis.vue`、`menus.vue`、`organizations.vue`、
  `system.vue` 和 `operation-records.vue` 中被截断的字符串/标签已恢复；`AoiTextField` 的 `enter`
  明确为无参数命令事件，避免键盘事件误传给 `{ silent }` 加载函数。验证：截断字符检索和
  `pnpm typecheck` 通过。
- 2026-06-13：完成后端 API 路径契约集中化审计。`types/constants` 新增稳定
  HTTP API 路径契约，后端 route catalog、操作记录判定、WebUI 保留路径和媒体
  本地下载 URL 改为复用同一 `/api/v1` 来源；本轮不改变外部路由、OpenAPI、权限码、
  前端 endpoint 或运行时配置。验证：聚焦 Go 测试与精确路径检索通过。
- 2026-06-13：完成前端 API endpoint 集中化审计。`web/admin/app/composables/useAdminApi.ts`
  仍是后台 API 的统一调用入口，但不再散落维护 `/api/v1` 路径；现有认证、组织、
  Demo、插件、System、媒体和版本接口路径集中到
  `web/admin/app/config/admin-api.ts` 的 `ADMIN_API_ENDPOINTS`。页面中的
  `/api/v1` 仅作为用户可见示例或筛选占位文案保留。验证：`pnpm typecheck` 通过。
- 2026-06-13：完成 WebUI 静态托管与构建路径审计。确认链路为
  `webui.mount_path`/`webui.dist_dir` -> `WebUIDeps` -> `pkg/web.MountStaticSPA`
  -> Nuxt `NUXT_APP_BASE_URL`/`.output/public`，并修正 `deploy.sh` 曾允许
  `--webui-mount-path /` 的不一致；根挂载会被 Go 配置拒绝，且容易让 SPA fallback
  覆盖 API、health、ready，因此部署脚本现在同步要求非根绝对路径。同步更新配置和
  发布文档。
- 2026-06-13：完成配置治理文档对照。`internal/config` 当前提取到 99 个
  `envname` 标签，`.env.example` 已覆盖这些前缀变量；配置说明补齐 Redis 连接池/
  超时、日志分流格式、IAM MFA/锁定/邀请/重置/SMTP/Casbin 等漏列变量。生产
  Compose 暴露的是部署模板变量子集，未在本轮扩展，后续如需全量化应先评估部署
  语义、端口映射和 secret 注入边界。
- 2026-06-13：建立全项目渐进式审计任务书，记录每轮必查证据、计划模板、
  统一配置规则、当前审计状态和本阶段记录。

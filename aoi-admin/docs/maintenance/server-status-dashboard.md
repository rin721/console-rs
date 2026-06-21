# 工作台服务器可视化维护指南

本文面向维护者，记录 `/admin` 工作台服务器可视化的治理入口、排查方式和发布检查。历史任务书位于 `docs/ai/server-status-dashboard-refactor-plan.md`。

## 阶段状态

| 阶段 | 状态 | 说明 |
| --- | --- | --- |
| 任务书与现状审计 | DONE | 已识别 Go 静态托管链路、React 前端入口和当前 API 数据边界。 |
| React 迁移 | DONE | 服务器状态已合并到 `/admin` 工作台概览，旧 `/admin/server-info` 后台入口已删除。 |
| 状态与格式化治理 | DONE | 容量换算、空值 fallback、图表 option 和展示结构收敛在 React 工作台与 Aoi patterns。 |
| 视觉基础治理 | DONE | 工作台服务器可视化使用 Aoi React 组件、ECharts 本体、后台 tokens 和统一数据状态组件。 |
| 服务明细 / GPU / CI | NEXT | 当前接口没有真实数据，不在前端伪造。 |

## 配置治理范围

- API endpoint：`web/app/app/lib/api/endpoints.ts` 和 `web/app/app/lib/api/system.ts`。
- 页面入口：`web/app/app/routes/admin/dashboard.tsx`。
- 图表基础组件：`web/app/app/components/aoi/patterns/EChart.tsx`。
- Aoi 展示模式：`web/app/app/components/aoi/patterns`。
- UI tokens：`web/app/app/styles/app.css` 中的 Aoi CSS 变量。

不要在页面中新增后端没有返回的指标、阈值或状态文案。新增指标必须先扩展后端采集 DTO 和短窗口采样器，再扩展 React API 类型、i18n 文案、图表 option、页面展示和测试。

## 常见问题

| 问题 | 排查入口 |
| --- | --- |
| 页面显示 `-` 或空状态 | 检查后端 DTO 或历史采样器是否返回字段；初次启动历史样本不足属于正常状态。 |
| 状态颜色不符合预期 | 检查 React 路由中的状态派生逻辑和 i18n 状态文案，不要只改页面 CSS。 |
| 图表尺寸异常 | 检查 `EChart` 的 `ResizeObserver`、容器尺寸和移动端媒体查询。 |
| 构建信息挤成细列 | 检查 Aoi key-value 展示模式和长文本换行。 |
| 静态资源 404 | 确认先在 `web/app` 执行 `pnpm build`，再检查 `webui.mount_path` 与 `webui.dist_dir`。 |
| API 不通 | `VITE_PUBLIC_API_BASE_URL` 为空时表示同源请求 Go API；非空时检查部署网关配置。 |

## 验证清单

1. 在 `web/app` 执行 `pnpm typecheck`。
2. 执行 `pnpm lint:i18n` 和 `pnpm lint`。
3. 执行 `pnpm build`，确认 `build/client/index.html` 存在。
4. 在仓库根目录执行 `go test ./... -count=1 -mod=readonly`。
5. 启动 Go 服务后访问 `/`、`/setup` 和 `/admin`，确认 SPA fallback 与静态资源正常，且 `/api/v1`、`/health`、`/ready` 不被前端 fallback 吞掉。
6. 在 1440x900、1280x720、390x844 检查无横向溢出、无 `undefined`、`null`、`NaN` 文本。

## Git 收口

每个可验证阶段都应提交并合并到 `main`。合并优先使用 fast-forward；如 `main` 前进，先只读检查差异，再用普通 merge 解决冲突，禁止强制覆盖或清理他人改动。

# Aoi IAM 产品维度、配置优先与缓存会话实现记录

## 背景

本次任务把 IAM 认证会话从“浏览器前端保存 token pair”的旧路径，收敛为服务端 HttpOnly Cookie 会话与 session snapshot；同时把产品码、客户端类型、会话并发策略、Cookie/CSRF 名称和 IAM 缓存策略接入配置，作为“共享主平台 + 多独立产品线”的后端扩展钩子。

## 实现边界

- 不新增完整产品线管理 UI。
- 不要求开发环境具备 Redis；IAM 只依赖本包 `CacheStore` contract，由应用装配层把现有本地/Redis/Hybrid 缓存实现注入。
- 浏览器认证响应不再暴露 access token 或 refresh token；前端只保存服务端 session snapshot。
- OpenAPI、system API catalog 和权限同步继续从 route contract 派生。

## 关键结果

- `auth.cookie`、`auth.csrf`、`auth.session`、`auth.cache` 进入配置结构、示例配置、生产示例和系统配置快照。
- `iam_sessions` 增加 `product_code` 与 `client_type`，支持同一 `user + org + product + client_type` 的单会话策略。
- `iam_permissions` 与 `system_apis` 增加 `product_code`，为未来产品线隔离权限和 API catalog 预留维度。
- IAM service 缓存 key 包含产品、组织、用户、epoch 和过滤条件，TTL 与开关来自配置。
- React API client 使用 cookie credentials、CSRF 双提交和 `/me/session` 快照；Zustand auth store 不再持久化 token。

## 验证

- 后端聚焦测试覆盖 config、IAM、System、middleware、transport 和 app。
- React 验证覆盖 i18n、typecheck、unit test 和 build。
- OpenAPI 由 `go run ./cmd/aoi api openapi --output docs/api/openapi.yaml` 重新生成。

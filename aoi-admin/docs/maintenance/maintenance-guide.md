# 维护指南

维护工作应保持代码、测试、文档和 AI 运行态一致。

## 常规变更流程

1. 阅读 `AGENTS.md` 和相关 `docs/ai` 运行态。
2. 识别本次变更真实影响的代码边界。
3. 更新代码和测试。
4. 更新 `docs` 下的结构化人类文档。
5. 如果运行态变化，更新或修复 `docs/ai` artifact。
6. 根据影响范围运行定向测试和更广测试。
7. 将剩余风险记录到 `docs/backlog/known-gaps.md` 或运行态证据中。

## 文档卫生

人类文档应解释当前代码，而不是把未来想法写成既成事实。未来工作或缺失能力应放入 backlog/known gaps，除非明确标记为计划变更。

优先在结构化目录中增加文档，避免继续增加顶层零散文件。旧链接必须迁移到当前结构化入口并删除旧入口；无法立即迁移的外部约束必须记录为阻塞点，不得写成长期兼容规则。

## 架构边界维护

- 基础设施初始化、关闭、reload 和跨模块装配保持在 `internal/app`。
- 模块 service 只依赖本包最小接口，不直接导入 `pkg/*`、同模块 repository 或 `internal/ports`。
- repository/infrastructure 可以实现 service 定义的接口，并负责数据库、外部 HTTP、SMTP、Storage、host metrics 等技术细节。
- 变更边界时同步 `docs/architecture/layers.md`、`docs/structure/directory-map.md`、模块文档和 `internal/import_boundary_test.go`。

## 运行态卫生

`docs/ai` 是运行态系统。当某个 task 或 slice 成为当前工作，它应能通过 current status、task tree、requirement ledger、evidence index
和 handoff 被发现。如果某个 artifact 缺失或太薄，应修复物理 artifact，而不是依赖聊天历史。

## API Token 维护

- 发布包含 API Token 的版本前，确认 `internal/migrations/20260612000400_create_iam_api_tokens.sql` 已在目标环境执行。
- `auth.refresh_token_pepper` 同时保护 refresh token 和 API Token hash；轮换该值会让既有 refresh token 与 API Token 全部失效，发布说明里必须提前告知调用方。
- API Token 明文只在签发成功时出现一次。排障时不要要求用户把完整 token 贴到 issue、日志或聊天记录中，优先使用 `tokenPrefix`、`tokenId` 和审计日志定位。
- 调用方泄漏 token、用户被禁用、角色权限收缩或自动化任务下线时，应在 `/admin/api-tokens` 或 `DELETE /api/v1/orgs/{orgId}/api-tokens/{tokenId}` 立即撤销。

## 组织管理维护

- `/admin/organizations` 读取 `GET /api/v1/orgs`。该接口返回分页对象，二次开发调用方应读取 `data.items`；`GET /api/v1/me/orgs` 仍返回当前用户所属组织数组，用于会话和顶部切换器。
- 组织列表筛选在 service 层按 `keyword`、`code`、`name`、`status` 处理。排查“看不到组织”时，先点击页面 `重置` 回到第一页，再检查调用方是否把 `page` 设置到超出总页数。
- 更新组织名称时必须满足请求路径 `orgId` 与 access token 中的 `orgId` 一致。需要维护其他组织时，先切换组织重新签发 token，再提交修改。
- 创建组织会把当前用户加入新组织并授予 owner 角色；如果创建成功但切换失败，优先检查新组织的内置角色和 Casbin policy 是否初始化完成。

## 用户管理维护

- `/admin/users` 与 API Token 签发页都读取 `GET /api/v1/orgs/{orgId}/users`。该接口现在返回分页对象，不再是裸数组；二次开发调用方应读取 `data.items`。
- 用户列表筛选在 service 层先限定当前组织成员，再按 `keyword`、`username`、`displayName`、`email`、`roleCode`、`status` 过滤。排查“看不到用户”时先确认当前 token 的 `orgId`，再检查角色和成员状态筛选。
- 本地 IAM 用户模型暂不包含手机号和头像字段。需要这些字段时，应先设计迁移和资料编辑权限，不要只为前端表格临时拼接伪字段。
- 新增成员仍走邀请流程。生产环境必须使用 SMTP 或外部通知驱动，避免 debug/no-op 响应把邀请 token 暴露给不该看到的人。

## 会话与 MFA 维护

- `/admin/sessions` 读取 `GET /api/v1/orgs/{orgId}/sessions?scope=org`。该接口返回分页对象，二次开发调用方应读取 `data.items`。
- 会话列表在 service 层始终限定当前 `orgId`。即使调用方传入 `userId`，也不会返回其他组织的会话；排查缺失会话时先确认 token 所属组织和页面筛选范围。
- 会话状态由 `revokedAt` 和 `expiresAt` 计算：`active` 表示未撤销且未过期，`revoked` 表示已撤销，`expired` 表示 refresh 会话已过期。过期或已撤销会话不应再次执行撤销操作。
- `/admin/security` 只处理当前账号的 MFA 设置、当前会话信息和退出登录。密码修改、邮箱、手机号、头像等资料编辑需要单独设计验证、通知和审计流程。

## Review 清单

- 变更是否保持目录边界？
- 配置示例和 env 文档是否同步？
- 启动、reload、shutdown 影响是否记录？
- 测试是否靠近它保护的行为？
- 生产风险是否清楚标记？
- AI 运行态 artifact 是否与当前工作状态一致？
## 版本发布包维护

- 发布包含版本管理的代码前，确认 `internal/migrations/20260612000500_create_system_versions.sql` 已在目标环境执行。
- `system_versions.version_data` 是完整 JSON 包，可能包含菜单路径、API 路径和字典内容；生产排障可以下载比对，但不要把敏感业务字典直接贴到公开 issue。
- 当前导入只会幂等创建缺失字典和字典项。菜单和 API 来自代码/路由目录，导入结果中的 `menusSkipped` 与 `apisSkipped` 属于预期行为。
- 如果后续把菜单或 API 改成数据库可编辑，必须同步更新版本导入冲突策略、回滚说明、OpenAPI 和 `docs/modules/system.md`。

## 媒体库维护

- 发布包含媒体库的代码前，确认 `internal/migrations/20260612000600_create_system_media.sql` 已在目标环境执行，并同步 `media:*` IAM 权限。
- 发布包含断点上传的代码前，还要确认 `internal/migrations/20260612000700_create_system_media_resumable_uploads.sql` 已执行。
- 普通上传和断点上传依赖 `storage.driver` 选择可写方案。本地建议使用 `storage.driver=local`、`storage.local.fsType=basepath` 和 `storage.local.basePath=./data/uploads`，避免把对象写到进程工作目录的非预期位置。
- 断点上传临时分片位于 `media/chunks/<session-id>/`，完成或中止时会尽力清理；排障时可以用 `system_media_upload_sessions.status`、`expires_at`、`final_asset_id` 和 `system_media_upload_chunks.chunk_index` 判断会话进度。
- URL 导入只保存外链，不下载远程文件。排障时优先检查 `system_media_assets.external`、`url`、`storage_key` 和 storage 配置。

- 删除本地媒体资源会先尝试删除对象，再软删除数据库记录；如果 storage 暂不可用，先恢复 storage，再执行删除，避免形成孤儿对象。

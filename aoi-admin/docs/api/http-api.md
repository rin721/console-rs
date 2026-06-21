# HTTP API 文档

本文档面向开发者阅读，概述当前服务暴露的 HTTP API。机器可读的完整契约见
`docs/api/openapi.yaml`。

## 通用约定

默认本地服务地址：

```text
http://127.0.0.1:9999
```

除特别说明外，响应统一包裹在 `Result` 结构中：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | integer | 业务错误码，`0` 表示成功。 |
| `message` | string | 响应消息。 |
| `data` | any | 响应数据，错误时通常为空。 |
| `traceId` | string | 请求追踪 ID，错误响应中常见。 |
| `serverTime` | integer | 服务端 Unix 秒级时间戳。 |

受保护的 IAM 接口需要请求头：

```http
Authorization: Bearer <accessToken>
```

自动化调用也可以使用后台签发的 API Token 作为同一个 Bearer 值。API Token 固定到签发组织，并按签发时绑定的角色权限授权；完整 token 只在创建响应中返回一次。

## 探针接口

| 方法 | 路径 | 认证 | 说明 |
| --- | --- | --- | --- |
| GET | `/health` | 否 | 存活检查，只说明进程和路由可响应。 |
| GET | `/ready` | 否 | 就绪检查，会检查数据库依赖。 |

## 插件接口

插件 HTTP 管理接口只提供远程插件宿主的管理视图，需要 IAM access token，并通过 Casbin 的 `plugin:read` 权限控制。插件注册、心跳、租约续期、注销、能力调用、事件订阅/推送和注入上下文走独立的 HTTP/WS/RPC JSON 协议端点；HTTP/WS 默认挂载在 `/plugin-api/v1/*`。

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/api/v1/plugins` | `plugin:read` | 列出已注册远程插件。 |
| GET | `/api/v1/plugins/{id}` | `plugin:read` | 读取单个远程插件快照。 |
| GET | `/api/v1/plugins/{id}/health` | `plugin:read` | 查看心跳健康状态。 |
| GET | `/api/v1/plugins/{id}/capabilities` | `plugin:read` | 查看插件注册时声明的能力。 |

## 系统接口

系统接口需要 IAM access token。菜单接口按当前用户权限过滤，API 目录接口需要 `permission:read`。

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | `/api/v1/system/menus` | 登录可见 | 返回当前用户可见的后台菜单分组。 |
| GET | `/api/v1/system/config` | `config:read` | 返回后端配置管理器当前脱敏运行配置快照，敏感字段只显示是否已配置。 |
| PATCH | `/api/v1/system/config` | `config:update` | 更新后端配置管理器当前快照；请求体 `persist=true` 时将支持的标量字段和字符串列表写回当前 YAML 配置文件，环境变量管理项拒绝持久化，保存后返回脱敏快照。 |
| GET | `/api/v1/system/server-info` | `server:read` | 返回主机 CPU/RAM/磁盘以及当前后端进程的运行时、内存、GC 和构建信息快照。 |
| GET | `/api/v1/system/server-metrics/history` | `server:read` | 返回应用启动后的短窗口真实采样，包含 CPU、RAM、最高磁盘使用率、Go heap、goroutine 数、网络收发 KB/s 和磁盘 IO 读写速率/次数/延迟。 |
| GET | `/api/v1/system/traffic-hijack/overview` | `traffic_hijack:read` | 返回流量劫持监控目标计数、风险分布、最近异常和最近结果。 |
| GET/POST | `/api/v1/system/traffic-hijack/targets` | `traffic_hijack:read/update` | 查询或创建 HTTP(S) 主动探针目标。 |
| PATCH/DELETE | `/api/v1/system/traffic-hijack/targets/{targetId}` | `traffic_hijack:update/delete` | 更新或删除单个探针目标。 |
| POST | `/api/v1/system/traffic-hijack/targets/{targetId}/probe` | `traffic_hijack:update` | 立即执行一次探测并返回真实结果。 |
| GET | `/api/v1/system/traffic-hijack/results` | `traffic_hijack:read` | 查询最近探测结果，支持 `targetId`、`limit` 和 `cursor`。 |
| GET | `/api/v1/system/traffic-hijack/events` | `traffic_hijack:read` | 查询按目标、原因和证据聚合的劫持事件，支持目标、风险、状态和分页筛选。 |
| POST | `/api/v1/system/traffic-hijack/events/{eventId}/resolve` | `traffic_hijack:update` | 手动确认事件已恢复。 |
| GET | `/api/v1/system/traffic-hijack/stream` | `traffic_hijack:read` | 以 `text/event-stream` 推送目标状态、探测结果和事件变化。 |
| GET | `/api/v1/system/apis` | `permission:read` | 返回当前进程真实注册的 HTTP API 目录。 |
| POST | `/api/v1/system/apis/sync` | `permission:read` | 同步当前进程 HTTP API 目录到 `system_apis` 表；表未迁移时只刷新目录并返回未持久化状态。 |
| POST | `/api/v1/system/apis/permissions/sync` | `permission:sync` | 从当前 API 目录提取权限码并补齐 `iam_permissions` 字典，便于角色授权页直接勾选。 |
| GET | `/api/v1/system/operation-records` | `operation:read` | 分页查询后台受保护接口的操作记录，支持请求方法、路径和状态码筛选。 |
| DELETE | `/api/v1/system/operation-records` | `operation:delete` | 按 ID 批量删除操作记录。 |
| GET | `/api/v1/system/versions` | `version:read` | 分页查询系统版本发布包，支持创建日期、版本名称和版本号筛选。 |
| GET | `/api/v1/system/versions/sources` | `version:read` | 返回可打包的菜单、API 和字典来源目录。 |
| POST | `/api/v1/system/versions/export` | `version:create` | 按所选菜单、API 和字典创建版本发布包。 |
| POST | `/api/v1/system/versions/import` | `version:import` | 导入版本发布包 JSON；字典会幂等补齐，菜单和 API 会记录在包内并报告跳过。 |
| GET | `/api/v1/system/versions/{versionId}` | `version:read` | 读取版本发布包详情和完整包内容。 |
| GET | `/api/v1/system/versions/{versionId}/download` | `version:download` | 返回版本发布包 JSON，前端会保存为文件。 |
| DELETE | `/api/v1/system/versions/{versionId}` | `version:delete` | 软删除单个版本发布包。 |
| DELETE | `/api/v1/system/versions` | `version:delete` | 按 ID 批量软删除版本发布包。 |
| GET | `/api/v1/system/media/categories` | `media:read` | 返回媒体库分类树。 |
| POST | `/api/v1/system/media/categories` | `media:update` | 创建或更新媒体分类。 |
| DELETE | `/api/v1/system/media/categories/{categoryId}` | `media:update` | 删除空媒体分类；存在子分类或文件时会拒绝。 |
| GET | `/api/v1/system/media/assets` | `media:read` | 分页查询媒体资源，支持 `categoryId`、`keyword`、`page`、`pageSize`。 |
| POST | `/api/v1/system/media/assets/upload` | `media:upload` | multipart 普通上传，字段名为 `file`，可选 `categoryId`。 |
| POST | `/api/v1/system/media/assets/resumable/check` | `media:upload` | 检查或创建断点上传会话，返回已上传分片和缺失分片。 |
| POST | `/api/v1/system/media/assets/resumable/chunks` | `media:upload` | multipart 上传单个分片，字段名为 `file`。 |
| POST | `/api/v1/system/media/assets/resumable/complete` | `media:upload` | 校验并合并全部分片，生成媒体库资产。 |
| POST | `/api/v1/system/media/assets/resumable/abort` | `media:upload` | 中止断点上传会话并清理临时分片。 |
| POST | `/api/v1/system/media/assets/import-url` | `media:import` | 导入外链媒体记录；只保存 URL，不抓取远程内容。 |
| PATCH | `/api/v1/system/media/assets/{assetId}` | `media:update` | 更新媒体显示名称。 |
| GET | `/api/v1/system/media/assets/{assetId}/download` | `media:download` | 鉴权下载本地媒体对象；外链资源应直接打开其 URL。 |
| DELETE | `/api/v1/system/media/assets/{assetId}` | `media:delete` | 删除媒体资源；本地资源会尝试移除对象后软删除元数据。 |
| GET | `/api/v1/system/parameters` | `parameter:read` | 分页查询系统参数，支持创建日期、参数名称和参数键筛选。 |
| POST | `/api/v1/system/parameters` | `parameter:create` | 创建系统参数。 |
| DELETE | `/api/v1/system/parameters` | `parameter:delete` | 按 ID 批量软删除系统参数。 |
| GET | `/api/v1/system/parameters/value` | `parameter:read` | 按参数键读取系统参数。 |
| GET | `/api/v1/system/parameters/{parameterId}` | `parameter:read` | 按 ID 读取系统参数。 |
| PATCH | `/api/v1/system/parameters/{parameterId}` | `parameter:update` | 更新系统参数名称、键、值或说明。 |
| DELETE | `/api/v1/system/parameters/{parameterId}` | `parameter:delete` | 软删除系统参数。 |
| GET | `/api/v1/system/dictionaries` | `dictionary:read` | 返回系统字典目录和字典项，表未就绪时返回不可用状态。 |
| POST | `/api/v1/system/dictionaries` | `dictionary:create` | 创建系统字典。 |
| PATCH | `/api/v1/system/dictionaries/{dictionaryId}` | `dictionary:update` | 更新系统字典编码、名称、说明或状态。 |
| DELETE | `/api/v1/system/dictionaries/{dictionaryId}` | `dictionary:delete` | 软删除系统字典及其字典项。 |
| POST | `/api/v1/system/dictionaries/{dictionaryId}/items` | `dictionary:update` | 创建字典项。 |
| PATCH | `/api/v1/system/dictionary-items/{itemId}` | `dictionary:update` | 更新字典项标签、值、扩展信息、排序或状态。 |
| DELETE | `/api/v1/system/dictionary-items/{itemId}` | `dictionary:delete` | 软删除字典项。 |

### 创建版本发布包

```http
POST /api/v1/system/versions/export
Authorization: Bearer <accessToken>
Content-Type: application/json
```

```json
{
  "versionName": "June Release",
  "versionCode": "v2026.06.12",
  "description": "menus, APIs and dictionaries for release window",
  "menuCodes": ["system:menus", "system:apis"],
  "apiCodes": ["get /api/v1/system/menus", "get /api/v1/system/apis"],
  "dictionaryCodes": ["system.status", "http.method"]
}
```

`menuCodes` 使用 `menuGroupCode:menuItemCode`，`apiCodes` 使用 API 目录中的 `code` 字段，`dictionaryCodes` 使用字典编码。至少选择一种资源。

### 导入版本发布包

```http
POST /api/v1/system/versions/import
Authorization: Bearer <accessToken>
Content-Type: application/json
```

```json
{
  "versionData": "{\"version\":{\"name\":\"June Release\",\"code\":\"v2026.06.12\",\"description\":\"sample\",\"exportTime\":\"2026-06-12T08:00:00Z\"},\"menus\":[],\"apis\":[],\"dictionaries\":[]}"
}
```

导入会写入一条 `system_versions` 记录。当前系统的菜单来自代码内置目录，API 来自路由同步目录，因此导入时不会修改菜单和 API；字典和字典项会按编码和值幂等创建，响应中的 `menusSkipped`、`apisSkipped`、`dictionariesCreated` 和 `dictionaryItemsCreated` 可用于发布记录。

### 媒体库普通上传

```http
POST /api/v1/system/media/assets/upload
Authorization: Bearer <accessToken>
Content-Type: multipart/form-data
```

表单字段：

| 字段 | 说明 |
| --- | --- |
| `file` | 必填，上传文件。服务端会生成 `media/YYYY/MM/<id>.<ext>` 存储 key。 |
| `categoryId` | 可选，媒体分类 ID；不传或 `0` 表示全部/根分类。 |

普通上传需要 `storage.driver` 选择 `local`、`s3`、`minio`、`local+s3` 或 `local+minio`。如果 storage 未启用，接口返回 503；外链导入不依赖对象存储。

### 媒体库断点上传

断点上传复用 `media:upload` 权限和 Storage 配置。前端先计算整文件 SHA-256，再按分片计算 SHA-256；服务端会把临时分片写入 `media/chunks/<session-id>/`，完成后校验整文件哈希并合并为普通媒体资产。

1. 检查或创建会话：

```http
POST /api/v1/system/media/assets/resumable/check
Authorization: Bearer <accessToken>
Content-Type: application/json
```

```json
{
  "categoryId": 0,
  "fileName": "sample.zip",
  "fileHash": "<sha256>",
  "sizeBytes": 2097152,
  "chunkSize": 1048576,
  "chunkTotal": 2
}
```

响应中的 `uploadedChunks` 和 `missingChunks` 使用从 `0` 开始的分片序号。`status=completed` 且带有 `asset` 时，说明相同文件已经完成，可直接使用该媒体资产。

2. 上传单个分片：

```http
POST /api/v1/system/media/assets/resumable/chunks
Authorization: Bearer <accessToken>
Content-Type: multipart/form-data
```

表单字段：

| 字段 | 说明 |
| --- | --- |
| `file` | 必填，当前分片二进制。 |
| `sessionId` | 必填，会话 ID。 |
| `fileHash` | 必填，整文件 SHA-256。 |
| `fileName` | 必填，原始文件名，只作展示和会话匹配。 |
| `chunkIndex` | 必填，从 `0` 开始。 |
| `chunkTotal` | 必填，总分片数。 |
| `chunkHash` | 必填，当前分片 SHA-256。 |

3. 合并分片：

```http
POST /api/v1/system/media/assets/resumable/complete
Authorization: Bearer <accessToken>
Content-Type: application/json
```

```json
{
  "sessionId": "93001",
  "fileHash": "<sha256>"
}
```

4. 中止会话：

```http
POST /api/v1/system/media/assets/resumable/abort
Authorization: Bearer <accessToken>
Content-Type: application/json
```

```json
{
  "sessionId": "93001",
  "fileHash": "<sha256>"
}
```

会话默认 24 小时过期。过期后不能继续写入分片，需要重新 `check` 创建会话。断点上传需要 `storage.driver` 选择可写方案；如果 storage 未启用，接口返回 503。

### 导入媒体外链

```http
POST /api/v1/system/media/assets/import-url
Authorization: Bearer <accessToken>
Content-Type: application/json
```

```json
{
  "categoryId": 0,
  "text": "我的图片|https://example.com/my.png\nhttps://example.com/other.png"
}
```

也可以传 `items: [{ "name": "我的图片", "url": "https://example.com/my.png" }]`。导入只保存 URL 元数据，不会下载远程内容。

## IAM 公开接口

这些接口不要求 access token。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/auth/setup/status` | 查询是否需要首次初始化管理员。 |
| POST | `/api/v1/auth/setup/initial-admin` | 空 IAM 用户表时创建首个组织 owner，并返回登录令牌。 |
| GET | `/api/v1/setup/status` | 查询统一初始化状态、步骤、诊断和最近一次运行。 |
| POST | `/api/v1/setup/runs` | 执行统一初始化流程，包含迁移、系统数据、权限同步和首个 owner。 |
| POST | `/api/v1/setup/runs/{id}/retry` | 按初始化运行记录发起重试；初始化完成后需提供 setup token。 |
| GET | `/api/v1/setup/runs/{id}/logs` | 查询脱敏后的初始化步骤日志摘要；初始化完成后需通过 `X-Setup-Token` 或 `setupToken` query 提供 setup token。 |
| POST | `/api/v1/auth/signup` | 按注册策略自助注册；direct 返回认证会话，email_verification 返回待验证状态。 |
| POST | `/api/v1/auth/email-verifications/{token}/confirm` | 确认邮箱验证，激活 pending 账号和组织并签发会话。 |
| GET | `/api/v1/auth/captcha` | 获取登录验证码开关、图片和 `captchaId`；验证码关闭时返回 `enabled=false`。 |
| POST | `/api/v1/auth/login` | 登录并签发 access token 与 refresh token。 |
| POST | `/api/v1/auth/refresh` | 使用 refresh token 刷新令牌。 |
| POST | `/api/v1/auth/password/forgot` | 创建密码重置通知；debug/noop/local 通知驱动会返回调试 token/link，smtp 不返回 token；smtp 投递失败会撤销重置 token 并返回 503。 |
| POST | `/api/v1/auth/password/reset` | 使用重置令牌重置密码。 |
| POST | `/api/v1/invitations/{token}/accept` | 接受组织邀请。 |

### 登录

```json
{
  "identifier": "admin@example.com",
  "password": "secret",
  "orgCode": "acme",
  "captchaId": "captcha-id",
  "captchaCode": "7",
  "mfaCode": "123456"
}
```

必填字段：`identifier`、`password`。

`orgCode` 可用于指定登录组织；开启登录验证码后，先调用 `GET /api/v1/auth/captcha` 并在登录时提交 `captchaId` 与 `captchaCode`；开启 MFA 后需要 `mfaCode`。

成功响应的 `data` 为：

| 字段 | 说明 |
| --- | --- |
| `accessToken` | access token。 |
| `accessExpiresAt` | access token 过期时间。 |
| `refreshToken` | refresh token。 |
| `refreshExpiresAt` | refresh token 过期时间。 |

### 刷新令牌

```json
{
  "refreshToken": "<refreshToken>"
}
```

### 找回密码

```json
{
  "email": "admin@example.com"
}
```

`auth.notification_driver=debug/noop/local` 时响应会包含调试 `token` 和 `url`；`smtp` 或外部通知模式不会在响应中暴露一次性 token。SMTP 投递失败会撤销本次创建的密码重置 token，并以 `api.auth.notificationDeliveryFailed` 返回 503。

### 重置密码

```json
{
  "token": "<resetToken>",
  "newPassword": "new-secret"
}
```

### 接受邀请

```http
POST /api/v1/invitations/<token>/accept
Content-Type: application/json
```

```json
{
  "username": "member",
  "displayName": "Member",
  "password": "secret"
}
```

必填字段：`username`、`password`。

## IAM 账号接口

以下接口都需要 `Authorization: Bearer <accessToken>`。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/v1/auth/logout` | 撤销当前会话。 |
| POST | `/api/v1/auth/switch-org` | 切换当前组织并签发新令牌。 |
| POST | `/api/v1/auth/mfa/setup` | 创建或轮换 TOTP MFA 密钥。 |
| POST | `/api/v1/auth/mfa/verify` | 验证并启用 TOTP MFA。 |
| GET | `/api/v1/me` | 查询当前用户资料。 |
| GET | `/api/v1/me/orgs` | 查询当前用户所属组织。 |

### 切换组织

```json
{
  "orgId": 10001
}
```

### 验证 MFA

```json
{
  "code": "123456"
}
```

## IAM 组织管理接口

以下接口都需要认证，并根据路由要求检查 Casbin 权限。

| 方法 | 路径 | 权限对象/动作 | 说明 |
| --- | --- | --- | --- |
| GET | `/api/v1/orgs` | `org:read` | 分页查询组织列表，支持 `keyword`、`code`、`name`、`status`、`page`、`pageSize`。 |
| POST | `/api/v1/orgs` | `org:create` | 创建组织，并把当前用户设为新组织 owner。 |
| PATCH | `/api/v1/orgs/{orgId}` | `org:update` | 更新当前组织信息。 |
| GET | `/api/v1/orgs/{orgId}/users` | `user:read` | 分页查询当前组织用户，支持 `keyword`、`username`、`displayName`/`nickName`、`email`、`roleCode`、`status`、`page`、`pageSize`。 |
| PATCH | `/api/v1/orgs/{orgId}/users/{userId}` | `user:update` | 更新成员状态或角色。 |
| POST | `/api/v1/orgs/{orgId}/users/invitations` | `user:invite` | 邀请用户加入当前组织；SMTP 投递失败会撤销邀请 token 并返回 503。 |
| GET | `/api/v1/orgs/{orgId}/invitations` | `user:invite` | 查询当前组织邀请。 |
| DELETE | `/api/v1/orgs/{orgId}/invitations/{invitationId}` | `user:invite` | 撤销待处理邀请。 |
| GET | `/api/v1/orgs/{orgId}/roles` | `role:read` | 查询当前组织角色。 |
| POST | `/api/v1/orgs/{orgId}/roles` | `role:create` | 在当前组织创建角色。 |
| PATCH | `/api/v1/orgs/{orgId}/roles/{roleId}` | `role:update` | 更新自定义角色。 |
| GET | `/api/v1/orgs/{orgId}/permissions` | `permission:read` | 查询可用权限。 |
| GET | `/api/v1/orgs/{orgId}/api-tokens` | `api_token:read` | 分页查询当前组织 API Token，支持 `userId`、`status`、`page`、`pageSize`。 |
| POST | `/api/v1/orgs/{orgId}/api-tokens` | `api_token:create` | 为组织成员和角色签发 API Token，完整 token 只在响应中返回一次。 |
| DELETE | `/api/v1/orgs/{orgId}/api-tokens/{tokenId}` | `api_token:revoke` | 撤销 API Token。 |

路径中的 `{orgId}` 必须与 access token 中的 `orgId` 一致。

### 查询组织列表

组织列表返回分页对象，用于后台 `组织管理` 的筛选表格。`keyword` 会匹配
组织 Code、组织名称和状态；`status` 可传 `active` 或 `disabled`。
二次开发调用方应读取 `data.items`，不要再按裸数组解析响应。

```http
GET /api/v1/orgs?keyword=team&status=active&page=1&pageSize=10
```

### 查询组织用户

用户列表返回分页对象，用于后台 `用户管理` 的筛选表格。`keyword`
会匹配用户名、显示名、邮箱、成员状态和角色；`status` 可传 `active` 或
`disabled`；`roleCode` 使用当前组织角色编码，例如 `owner`、`admin` 或
`member`。

```http
GET /api/v1/orgs/{orgId}/users?keyword=alice&roleCode=admin&page=1&pageSize=10
```

### 创建组织

```json
{
  "code": "acme",
  "name": "Acme Corp"
}
```

### 邀请用户

```json
{
  "email": "member@example.com",
  "roleCode": "member"
}
```

当前 no-op 通知器会在响应中直接返回邀请 `token`。

### 创建角色

```json
{
  "code": "operator",
  "name": "Operator",
  "description": "Daily operator",
  "permissions": ["user:read", "session:read"]
}
```

## IAM API Token 接口

API Token 接口用于管理服务到服务调用凭据。创建时必须选择当前组织中的用户和该用户已经拥有的角色，服务端会把 token 权限限制在这个角色上。

### 查询 API Token

```http
GET /api/v1/orgs/10001/api-tokens?status=active&userId=10002&page=1&pageSize=10
Authorization: Bearer <accessToken>
```

`status` 可选：`active`、`expired`、`revoked`。不传时返回全部状态。响应中的 `tokenPrefix` 只用于识别，不可用于认证。

### 签发 API Token

```http
POST /api/v1/orgs/10001/api-tokens
Authorization: Bearer <accessToken>
Content-Type: application/json
```

```json
{
  "userId": 10002,
  "roleCode": "member",
  "days": 30,
  "remark": "CI deploy job"
}
```

`days` 省略或为 `0` 时默认 30 天，`-1` 表示长期有效，最大支持 3650 天。响应：

```json
{
  "item": {
    "id": "93001",
    "userId": "10002",
    "roleCode": "member",
    "tokenPrefix": "aoi_xxxxxxxx",
    "status": "active"
  },
  "token": "aoi_full_token_only_returned_once"
}
```

调用业务接口时把 `token` 放入 Bearer 头：

```http
GET /api/v1/me
Authorization: Bearer aoi_full_token_only_returned_once
```

### 撤销 API Token

```http
DELETE /api/v1/orgs/10001/api-tokens/93001
Authorization: Bearer <accessToken>
```

## IAM 会话接口

| 方法 | 路径 | 权限对象/动作 | 说明 |
| --- | --- | --- | --- |
| GET | `/api/v1/orgs/{orgId}/sessions` | `session:read` | 分页查询会话，支持 `scope`、`keyword`、`userId`、`ipAddress`、`status`、`page`、`pageSize`。 |
| DELETE | `/api/v1/orgs/{orgId}/sessions/{sessionId}` | `session:revoke` | 撤销当前组织中的会话。 |

会话列表返回分页对象。未传 `scope` 和 `userId` 时查询当前用户会话；
`scope=org` 查询当前组织范围，指定 `userId` 时仍会限定当前组织。`status`
可传 `active`、`revoked` 或 `expired`。

```http
GET /api/v1/orgs/10001/sessions?scope=org&status=active&page=1&pageSize=10
```

## IAM 审计接口

| 方法 | 路径 | 权限对象/动作 | 说明 |
| --- | --- | --- | --- |
| GET | `/api/v1/orgs/{orgId}/audit-logs` | `audit:read` | 查询当前组织审计日志。 |

可选查询参数：

```http
GET /api/v1/orgs/10001/audit-logs?action=auth.login&userId=10002&limit=100&cursor=90001
```

`limit` 默认值为 `100`。

## 常见错误

| HTTP 状态码 | 错误码 | 说明 |
| --- | --- | --- |
| 400 | `1000` | 请求参数无效。 |
| 401 | `3000` | 未认证、登录失败或令牌无效。 |
| 403 | `3003` | 权限不足或组织不匹配。 |
| 404 | `4000` | 资源不存在。 |
| 500 | `5000` | 服务端内部错误。 |
| 503 | `5001` | 服务未就绪，常见于数据库不可用。 |

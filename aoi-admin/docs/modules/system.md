# System 模块

`internal/modules/system` 承载后台系统管理能力。System service 定义 repository、权限同步、host metrics 和 storage 相关 contract；具体数据库 executor、主机指标采集和对象存储由 repository、app adapter 和 app 装配注入。

## 能力

| 能力 | 说明 |
| --- | --- |
| 菜单目录 | `GET /api/v1/system/menus` 返回当前后台菜单目录 |
| API 目录 | 从当前进程已注册路由生成 API catalog，并可同步到数据库 |
| 权限同步 | 将带权限的 API 目录同步到 IAM 权限目录 |
| 字典管理 | 维护 `system_dictionaries` 和 `system_dictionary_items` |
| 参数管理 | 维护 `system_parameters` |
| 系统配置 | 读取脱敏配置快照，受控更新运行时配置并可持久化到 YAML |
| 操作记录 | 记录受保护后台 API 请求 |
| 服务器状态 | 返回运行时、构建信息、CPU、内存、磁盘容量和短窗口网络/磁盘 IO 趋势指标 |
| 流量劫持监控 | 以 HTTP(S) 主动探针检测 DNS、TLS、跳转、状态码、内容关键字和耗时异常 |
| 版本发布包 | 保存菜单、API、字典快照 JSON，用于留痕、下载和跨环境导入 |
| 媒体库 | 分类、普通上传、断点上传、外链导入、重命名、下载和删除 |

## 依赖边界

- `service` 定义 `Repository`、`PermissionStore`、`HostMetricsCollector`、`MetricsHistoryProvider`、`IDGenerator` 和 `MediaObjectStorage` contract。
- `repository` 实现 System repository contract，持有数据库 executor，并把底层 not-found 映射为 `ErrNotFound`，把缺表类错误包装为 `ErrStorageUnavailable`。
- host metrics 由 `internal/app/adapters.HostMetricsCollector` 适配 `pkg/hostmetrics`；短窗口历史由 `internal/app/adapters.ServerMetricsSampler` 在应用生命周期中采样并注入。
- traffic hijack 由 `internal/app/adapters.TrafficProbeRunner` 使用标准库 HTTP client、DNS resolver、`httptrace` 和 SSE 事件流实现；调度器随应用生命周期启动，service 只依赖本包定义的 runner、alert sink 和 repository contract。
- 媒体对象存储由 `internal/app` 从 Storage 基础设施注入；service 不直接依赖 `pkg/storage`。

## 路由和权限

| 路由 | 权限 | 用途 |
| --- | --- | --- |
| `GET /api/v1/system/menus` | 认证 | 菜单目录 |
| `GET/PATCH /api/v1/system/config` | `config:read/update` | 运行时配置 |
| `GET /api/v1/system/server-info` | `server:read` | 服务器状态 |
| `GET /api/v1/system/server-metrics/history` | `server:read` | 服务器指标短窗口历史 |
| `/api/v1/system/traffic-hijack*` | `traffic_hijack:*` | 流量劫持监控目标、结果、事件和 SSE 流 |
| `GET /api/v1/system/apis` | `permission:read` | API 目录 |
| `POST /api/v1/system/apis/sync` | `permission:read` | 同步 API 目录 |
| `POST /api/v1/system/apis/permissions/sync` | `permission:sync` | 同步权限目录 |
| `GET/DELETE /api/v1/system/operation-records` | `operation:read/delete` | 操作记录 |
| `/api/v1/system/versions*` | `version:*` | 版本发布包 |
| `/api/v1/system/media*` | `media:*` | 媒体库 |
| `/api/v1/system/parameters*` | `parameter:*` | 参数管理 |
| `/api/v1/system/dictionaries*`、`/dictionary-items*` | `dictionary:*` | 字典管理 |

## 版本发布包

版本发布包存储菜单、API 和字典配置快照：

- 菜单来自 service 内置目录；
- API 来自当前进程路由目录；
- 字典来自数据库；
- 导入时只幂等补齐字典，菜单和 API 保留在包记录中并报告跳过。

它不是 Go 构建版本，也不是 goose 迁移版本。

## 媒体库

- 外链导入只保存 URL，不下载远程文件。
- 普通上传和断点上传需要 `storage.driver` 选择 `local`、`s3`、`minio`、`local+s3` 或 `local+minio`。
- 本地对象 key 由服务端生成，原始文件名只用于展示。
- 下载本地对象需要 IAM 鉴权，不提供匿名静态下载。
- 断点上传临时分片位于 `media/chunks/<session-id>/`，完成或中止后清理。

Storage 不可用时，列表和外链导入仍可工作；普通上传、断点上传、本地下载和本地删除会返回 storage unavailable。

## 服务器状态

`GET /api/v1/system/server-info` 返回 Go runtime、构建信息、CPU、内存和磁盘挂载点容量采样。`GET /api/v1/system/server-metrics/history` 返回应用启动后的短窗口真实采样，默认每 5 秒采样、保留 60 个点，包含 CPU、RAM、最高磁盘使用率、Go heap、goroutine 数、网络收发 KB/s，以及聚合和单磁盘 IO 的读写 MB/s、读写次数/s 和平均 IO 延迟。磁盘 IO 名称来自操作系统 disk counter；容量/使用率仍来自挂载点数据，不做 device 到 mount point 的虚假映射。初次启动样本不足或首个样本速率为 0 是正常状态。

后端 DTO 当前不包含 GPU、CI/CD、后台任务或服务进程明细；前端不能 mock 不存在的指标。

前端治理入口：

- `web/app/app/lib/api/endpoints.ts`
- `web/app/app/lib/api/system.ts`
- `web/app/app/routes/admin/dashboard.tsx`
- `web/app/app/components/aoi/patterns/EChart.tsx`
- `web/app/app/components/aoi/patterns`
- `web/app/app/i18n/locales/{zh-CN,en}.json`

新增指标必须先扩展后端采集和 DTO，再扩展前端配置、图表 option 和派生模型。服务器状态不再有独立 `/admin/server-info` 后台页面，工作台入口统一为 `/admin`。

## 流量劫持监控

流量劫持监控的 V1 定义是“外部访问路径异常”，不做抓包、旁路代理或真实 MITM 检测。后端主动探测用户保存的 HTTP(S) 目标，异常来源包括：

- DNS 解析 IP 偏离期望 IP/CIDR；
- loopback、link-local、multicast、private 或 reserved 地址默认被阻断，只有目标显式开启 `allowPrivateNetwork` 时才允许；
- TLS 证书异常或 SHA256 指纹不匹配；
- 跳转超过 5 次或最终 Host 与期望不一致；
- 状态码不在期望范围；
- 响应体缺少期望关键字；
- DNS、连接、TLS、TTFB 或总耗时探测失败。

目标数据写入 `system_traffic_probe_targets`，每次探测结果写入 `system_traffic_probe_results`，同一目标只保留最近 500 条结果。异常事件写入 `system_traffic_hijack_events`，按 `targetId + reason + evidenceHash` 聚合 open/update/resolved 状态。告警通道为目标级配置：`event` 写站内事件，`debug` 写后端日志，`email` 复用 SMTP sender；邮件不可用不会阻塞探针，只更新通知状态。

后台入口为 `/admin/traffic-hijack`，工作台 `/admin` 同步展示概览卡片。实时展示使用 `GET /api/v1/system/traffic-hijack/stream` 的 `text/event-stream`，前端因认证需要通过带 Bearer header 的 fetch stream 消费 SSE；断开后按 30 秒轮询 overview/results/events。

## 配置

System 本身只有：

```yaml
system:
  seed_defaults_on_start: true
```

媒体库复用 `storage.*`。版本发布包不新增 YAML 或环境变量配置。运行时配置 API 由 `config:*` 权限保护。

## 测试入口

```powershell
go test ./internal/modules/system/... -count=1 -mod=readonly
go test ./internal/transport/http -count=1 -mod=readonly
```

## 非目标

- 版本导入不改写代码菜单和 HTTP 路由。
- 媒体外链导入不下载远程资源。
- 服务器状态不展示后端未采集的指标。

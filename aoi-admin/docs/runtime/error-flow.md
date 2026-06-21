# 错误流

错误从 repository/infrastructure 进入 service，再由 handler 转换为 HTTP 响应。业务层应暴露模块语义错误，不把 GORM、SMTP、HTTP client、storage 或底层端口错误泄漏给调用者。

## 模块错误

| 层 | 责任 |
| --- | --- |
| repository | 捕获数据库 not-found、缺表、约束或执行错误，并按 service contract 做必要映射 |
| infrastructure | 返回 service 接口定义的错误语义，保留技术细节给日志或包装错误 |
| service | 做业务校验、权限/可见范围判断、事务意图和模块错误归一化 |
| handler | 根据模块错误选择 HTTP status 和统一响应 envelope |

示例：

- IAM repository 将数据库 not-found 映射为 `iamservice.ErrNotFound`。
- System repository 将 not-found 映射为 `systemservice.ErrNotFound`，将缺表类错误包装为 `systemservice.ErrStorageUnavailable`。

## 响应辅助

`types/result` 负责通用 JSON 响应辅助。模块 handler 使用它保持响应 envelope、trace ID 和错误格式一致。

## Panic Recovery

HTTP recovery middleware 捕获 panic，带 trace context 记录日志，并返回错误响应，避免进程崩溃。

## Trace ID

trace middleware 使用 `traceId` 保存 request trace ID。`types/result` 优先读取同一个键，并兼容历史键 `trace_id`。panic recovery 和普通 result helper 都会把同一 trace ID 写入错误响应，便于日志、审计和客户端错误上报关联。

## 就绪检查错误

`GET /ready` 检查 database manager 是否存在以及能否 ping。它用于区分进程存活和服务就绪，应被本地验证、Docker healthcheck 和部署脚本使用。

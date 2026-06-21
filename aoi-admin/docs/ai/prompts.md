# 可复用提示词

这些提示词是起点。使用前补充具体文件、失败命令、范围限制和验收标准。

## 理解一条调用链

```text
请只读追踪 <feature> 的调用链。从 CLI 或 HTTP 路由开始，经过 internal/app 装配，
再进入 handler、service、本地接口和 repository/infrastructure 实现。请指出 pkg
基础设施在哪里被 app 装配，确认 service 是否直接依赖 pkg、internal/ports 或同模块
repository。不要编辑文件，输出主要文件和简短数据流。
```

## 增加模块

```text
请按当前模块约定增加 <module-name>。保持现有 model/repository/service/handler
包名：model 是 domain，service 是 application，handler 是 adapter，
repository/infrastructure 是接口实现。service 包先定义最小接口，通过构造函数注入；
具体基础设施由 internal/app 装配。添加聚焦测试并同步 docs/modules、docs/api。
```

## 架构边界审查

```text
请审查本分支是否违反洋葱模型。重点检查 internal/modules/*/service 是否导入 pkg、
internal/ports 或同模块 repository，是否出现 http.Client、smtp、os.Getenv、
database.New、WithExecutor 等基础设施细节。发现问题先列文件和规则，再做最小重构。
完成后运行 go test ./internal/... -count=1 -mod=readonly 和依赖扫描。
```

## Debug CI

```text
请使用 GitHub tooling 检查当前 PR 的失败检查。先获取失败 job 日志，定位第一个可执行
失败点，再提出聚焦修复。若 gh 未登录，请停止并说明需要用户运行 gh auth login。
```

## 安全审查

```text
请审查 IAM/auth/security-sensitive 变更。优先列出认证绕过、权限缺失、secret 暴露、
不安全文件/SQL 处理、token/MFA/API Token 风险和缺失测试。 findings first，带文件和行号。
```

## 文档刷新

```text
请对照当前代码刷新 <area> 文档。只写已验证的当前行为；不确定或未实现能力写入
docs/backlog/known-gaps.md 或 docs/ai 历史记录，不要写成已交付事实。同步涉及的
docs/api/http-api.md、docs/api/openapi.yaml 和配置示例说明。
```

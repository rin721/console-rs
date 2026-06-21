# DB CLI 工作流

`db` 命令提供基于 sqlgen 的数据库 DDL 预览/应用，以及 goose 迁移执行。
命令由 `cmd/aoi` 声明为 `cli.CommandSpec`，通过 `pkg/cli` 的 Cobra 封装解析参数。

## 示例

```bash
go run ./cmd/aoi db --operation=database --print-sql
go run ./cmd/aoi db --operation=database --apply
go run ./cmd/aoi db migrate status --config=configs/config.yaml
go run ./cmd/aoi db migrate up --config=configs/config.yaml
go run ./cmd/aoi db migrate down --config=configs/config.yaml
```

## 范围

`db --operation=database` 聚焦数据库级 DDL 预览和显式应用。`db migrate *` 通过
`pkg/migrator` 执行 `internal/migrations` 中的 goose 迁移，用于创建 IAM、System、
媒体库等版本化数据库变更。

## 维护提示

- 命令声明保持在 `cmd/aoi`，使用 `cli.CommandSpec`/`cli.FlagSpec` 描述命令和 flag。
- CLI 路由、flag 解析、help 和无参数 TUI 首页由 `pkg/cli` 统一封装。
- SQL 生成和执行行为保持在 `internal/app/cliapp/services/db`。
- 迁移执行行为保持在 `pkg/migrator`，通过 `pkg/database.SQLDB()` 获取标准 SQL 连接。
- flag 行为的测试放在命令层附近，SQL 行为的测试放在 `internal/app/cliapp/services/db` 附近。

IAM 初始化命令见 [IAM CLI 工作流](iam-cli.md)。

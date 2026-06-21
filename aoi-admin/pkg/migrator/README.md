# pkg/migrator - 数据库迁移封装

`pkg/migrator` 是项目的 goose 防腐层。它通过项目自有的 `Runner` 和 `Config` 暴露迁移能力，避免命令层或应用层直接依赖 goose API。

## API 分类

- 定位：[CONFIRMED] 公共基础设施 API。
- 稳定边界：`Runner`、`Config`、`SQLProvider`、`DefaultDir`、`New`。
- 当前风险：[RISK] goose 的 dialect/logger 是全局状态，本包内部用互斥锁串行化迁移执行。
- 非目标：[CONFIRMED] 本包不决定生产发布窗口、不生成迁移文件、不做备份和回滚演练。

## 设计约束

- 默认迁移目录是 `internal/migrations`。
- 调用方通过 `SQLProvider` 提供标准 `*sql.DB`，当前由 `pkg/database.SQLDB()` 实现。
- `sqlite` 会映射为 goose 的 `sqlite3` dialect，`postgres` 会映射为 `postgres`。
- 状态输出通过 `io.Writer` 注入，便于 CLI 和测试复用。

## 基本用法

```go
runner, err := migrator.New(db, migrator.Config{
    Driver: string(database.DriverSQLite),
    Dir:    migrator.DefaultDir,
})
if err != nil {
    return err
}

if err := runner.Up(ctx); err != nil {
    return err
}

if err := runner.Status(ctx, os.Stdout); err != nil {
    return err
}
```

## CLI 集成

`cmd/aoi` 通过 `db migrate up|down|status` 暴露迁移命令。生产默认不自动迁移，应在维护窗口显式执行，并保留数据库备份和迁移输出证据。

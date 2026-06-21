# 数据库迁移目录

`migrations/*.sql` 是当前运行时使用的 SQLite 迁移，Rust 服务只会自动应用这一组文件。

`migrations/postgres/*.sql` 和 `migrations/mysql/*.sql` 是 PostgreSQL/MySQL 的方言化 bootstrap schema 草案，用于证明 schema 扩展路径和禁止旧插件表回流。它们可以通过 `cargo run -p app -- database-plan --config <config>` 作为迁移计划清单输出，也可以通过 `cargo run -p app -- database-migrate --config <config>` 在对应 driver 下显式执行。

`database-migrate` 不等于 `serve` 已支持外部数据库：启用 PostgreSQL/MySQL app runtime 前，还必须补齐 repository 实现、真实数据库集成测试矩阵、启动或部署流水线接入和生产部署说明。

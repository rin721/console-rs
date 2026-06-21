# pkg/sqlgen - SQL 生成工具

双向 SQL 生成库 - Go 语言实现

## 简介

`sqlgen` 是一个"离线版"的 GORM，提供两种核心功能:

- **正向生成 (Model → SQL)**: 接收 Go Struct 和链式条件，返回 SQL 字符串
- **逆向生成 (SQL → Model)**: 接收 SQL DDL 脚本，返回 Go Struct 代码

## API 分类

- 定位：[CONFIRMED] 公共工具 API。
- 稳定边界：当前测试覆盖的 SQL 构建、解析、事务和模板能力。
- 当前风险：[CONFIRMED] TODO/未实现能力已在下方“不支持能力”边界中标注。
- 非目标：[CONFIRMED] 本包不替代运行时 ORM，也不直接执行数据库写入。

## 安装

```go
import "github.com/rei0721/go-scaffold/pkg/sqlgen"
```

## 快速开始

### 正向生成 (Model → SQL)

```go
// 定义模型
type User struct {
    ID        uint64    `gorm:"column:id;primaryKey;autoIncrement"`
    Username  string    `gorm:"column:username;size:64;not null"`
    Email     string    `gorm:"column:email;size:128"`
    Status    int       `gorm:"column:status;default:1"`
    CreatedAt time.Time `gorm:"column:created_at"`
    DeletedAt *time.Time `gorm:"column:deleted_at"`
}

// 初始化生成器
gen := sqlgen.New(&sqlgen.Config{
    Dialect: sqlgen.MySQL,
    Pretty:  false,
})

// CREATE TABLE
sql, _ := gen.Table(&User{})
// Output: CREATE TABLE `users` (...)

// INSERT
user := User{Username: "admin", Email: "admin@test.com"}
sql, _ := gen.Create(&user)
// Output: INSERT INTO `users` (...) VALUES (...)

// SELECT
sql, _ := gen.Where("status = ?", 1).Find(&[]User{})
// Output: SELECT * FROM `users` WHERE status = 1 ...

// UPDATE
sql, _ := gen.Model(&user).Updates(map[string]interface{}{"status": 2})
// Output: UPDATE `users` SET `status`=2 WHERE ...

// DELETE (软删除)
sql, _ := gen.Delete(&User{}, 1)
// Output: UPDATE `users` SET `deleted_at`='...' WHERE `id` = 1
```

### 逆向生成 (SQL → Model)

```go
ddl := `
CREATE TABLE sys_users (
    id bigint unsigned AUTO_INCREMENT PRIMARY KEY,
    username varchar(64) NOT NULL COMMENT '用户名',
    is_active tinyint(1) DEFAULT 1,
    created_at datetime
);`

// 解析并生成
goCode, _ := gen.ParseSQL(ddl).
    Package("models").
    Tags(sqlgen.TagGorm | sqlgen.TagJson).
    WithTableName(true).
    Generate()

// Output:
// package models
//
// type SysUser struct {
//     ID        uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
//     Username  string    `gorm:"column:username;not null" json:"username"`
//     ...
// }
```

## API 参考

### 配置

```go
type Config struct {
    Dialect             Dialect // MySQL, PostgreSQL, SQLite, SQLServer
    Pretty              bool    // 格式化输出
    SkipZeroValue       bool    // 跳过零值 (UPDATE)
    SoftDelete          bool    // 启用软删除
    AllowEmptyCondition bool    // 允许无条件 UPDATE/DELETE
}
```

### 正向生成 API

| 方法                      | 说明           | 示例                            |
| ------------------------- | -------------- | ------------------------------- |
| `Table(model)`            | CREATE TABLE   | `gen.Table(&User{})`            |
| `Drop(model)`             | DROP TABLE     | `gen.Drop(&User{})`             |
| `Create(data)`            | INSERT         | `gen.Create(&user)`             |
| `First(dest, conds...)`   | SELECT LIMIT 1 | `gen.First(&user, 1)`           |
| `Find(dest, conds...)`    | SELECT         | `gen.Find(&users)`              |
| `Updates(values)`         | UPDATE         | `gen.Model(&user).Updates(...)` |
| `Delete(model, conds...)` | DELETE/软删除  | `gen.Delete(&User{}, 1)`        |

### 链式方法

| 方法                    | 说明       |
| ----------------------- | ---------- |
| `Model(model)`          | 设置模型   |
| `Select(columns...)`    | 选择列     |
| `Omit(columns...)`      | 忽略列     |
| `Where(query, args...)` | WHERE 条件 |
| `Order(value)`          | ORDER BY   |
| `Limit(n)`              | LIMIT      |
| `Offset(n)`             | OFFSET     |
| `Unscoped()`            | 忽略软删除 |

### 逆向生成 API

| 方法                   | 说明            |
| ---------------------- | --------------- |
| `ParseSQL(ddl)`        | 解析 DDL 字符串 |
| `ParseSQLFile(path)`   | 解析 DDL 文件   |
| `Generate()`           | 生成单个 Struct |
| `GenerateAll()`        | 生成所有表      |
| `GenerateToFile(path)` | 生成到文件      |
| `GenerateToDir(dir)`   | 生成到目录      |

## 不支持能力 / 部分能力边界

以下能力当前不属于稳定可用 API。使用时不得假设其已经按 GORM 或数据库连接语义完整实现。

| 能力 | 当前行为 | 状态 | 建议 |
|---|---|---|---|
| `Or` / `Not` / `Group` / `Having` / `Distinct` / `Joins` | 链式调用会记录 unsupported 状态；后续 `Find`、`First`、`Count`、`Pluck`、`Updates` 或 `Delete` 返回 `ErrCodeUnsupportedOperation` | [CONFIRMED] unsupported | 使用 `Where` 写完整 SQL 片段，或单独提升需求 |
| `DeleteInBatches` | 直接返回 `ErrCodeUnsupportedOperation`，不再退化为普通删除 | [CONFIRMED] unsupported | 使用调用方自行分页生成条件删除 SQL |
| `ReverseDB(...).Generate` / `GenerateAll` / `GenerateToDir` | 直接返回 `ErrCodeUnsupportedOperation` | [CONFIRMED] unsupported | 使用 `ParseSQL` / `ParseSQLFile` 从 DDL 逆向生成 |
| `MigrationBuilder.BuildRollback` 中的 `ModifyColumn` 回滚 | 输出 `-- TODO: Restore original type...` 注释，不生成可直接执行的回滚 SQL | [CONFIRMED] partial | 手工补齐原始字段类型后再执行 |
| `Config.Pretty` | 当前不保证所有 SQL 输出全局格式化；部分 DDL 本身包含换行 | [CONFIRMED] partial | 不要把它作为稳定格式化契约 |

## 支持的方言

- MySQL
- PostgreSQL
- SQLite
- SQL Server

## 许可证

MIT

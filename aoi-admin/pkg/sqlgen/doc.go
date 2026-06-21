// Package sqlgen 提供双向 SQL 生成功能
//
// sqlgen 是一个"离线版"的 GORM，支持两种核心功能：
//
// 1. 正向生成 (Forward): 接收 Go Struct 和链式条件，返回 SQL 字符串
//   - 提供 GORM-like API 风格，但不承诺完整兼容 GORM 查询能力
//   - 支持已测试的 CRUD、DDL、事务、迁移脚本生成
//
// 2. 逆向生成 (Reverse): 接收 SQL DDL 脚本，返回 Go Struct 代码
//   - 支持 CREATE TABLE 解析
//   - 可配置 Tag、命名策略、类型映射
//
// # 快速开始
//
// 正向生成示例:
//
//	gen := sqlgen.New(&sqlgen.Config{
//	    Dialect: sqlgen.MySQL,
//	    Pretty:  false,
//	})
//
//	// 生成 INSERT 语句
//	sql, _ := gen.Create(&user)
//
//	// 生成 SELECT 语句
//	sql, _ := gen.Where("status = ?", 1).Find(&users)
//
// 逆向生成示例:
//
//	ddl := "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(64));"
//	goCode, _ := gen.ParseSQL(ddl).Generate()
//
// # 设计哲学
//
//   - 纯文本工具: 不依赖数据库连接，可在任何环境运行
//   - GORM-like: 利用开发者对 GORM API 的肌肉记忆，unsupported 能力会显式返回错误
//   - 可扩展性: 支持 Template、Hooks、自定义转换器
//   - 多方言: 支持 MySQL, PostgreSQL, SQLite, SQL Server
package sqlgen

// 本文件承载包级 Godoc 入口，集中说明该包在脚手架架构中的定位、使用边界和非目标能力。

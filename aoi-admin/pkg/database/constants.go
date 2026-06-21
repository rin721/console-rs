package database

// 本文件属于数据库基础设施层，定义连接工厂、池参数、事务上下文和热重载资源替换边界。

import "time"

const (
	// DefaultReloadTimeout 默认重载超时时间
	// 用于控制重载操作的最大等待时间
	// 包括创建新连接和关闭旧连接的时间
	DefaultReloadTimeout = 30 * time.Second

	// DefaultConnMaxLifetime 默认连接最大生命周期
	// 如果配置中未指定,使用此默认值
	DefaultConnMaxLifetime = time.Hour
)

// 错误消息常量
const (
	// ErrMsgFailedToCreateConnection 创建数据库连接失败的错误消息
	ErrMsgFailedToCreateConnection = "failed to create new database connection"

	// ErrMsgConnectionPingFailed 连接 Ping 测试失败的错误消息
	ErrMsgConnectionPingFailed = "database connection ping failed"

	// ErrMsgFailedToCloseConnection 关闭数据库连接失败的错误消息
	ErrMsgFailedToCloseConnection = "failed to close database connection"

	// ErrMsgUnsupportedDriver 不支持的数据库驱动错误消息
	ErrMsgUnsupportedDriver = "unsupported database driver"
)

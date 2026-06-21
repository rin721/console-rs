package cache

// 本文件属于 Redis 缓存适配器，说明连接生命周期、键值操作、批处理或热重载边界。

// 默认配置常量
// 这些值是经过生产环境验证的合理默认值
const (
	// DefaultHost 默认 Redis 主机地址
	// 通常 Redis 部署在本地或内网
	DefaultHost = "localhost"

	// DefaultPort 默认 Redis 端口
	// Redis 的标准端口是 6379
	DefaultPort = 6379

	// DefaultDB 默认数据库索引
	// Redis 支持 0-15 共 16 个数据库
	// 默认使用 0 号数据库
	DefaultDB = 0

	// DefaultPoolSize 默认连接池大小
	// 连接池大小决定了最大并发连接数
	// 10 个连接适合中小型应用
	DefaultPoolSize = 10

	// DefaultMinIdleConns 默认最小空闲连接数
	// 保持一定数量的空闲连接可以减少连接建立的延迟
	// 通常设置为连接池大小的一半
	DefaultMinIdleConns = 5

	// DefaultMaxRetries 默认最大重试次数
	// 当命令执行失败时,自动重试的次数
	// 3 次重试可以应对大多数临时性网络问题
	DefaultMaxRetries = 3

	// DefaultDialTimeout 默认连接超时时间(秒)
	// 建立 TCP 连接的最大等待时间
	DefaultDialTimeout = 5

	// DefaultReadTimeout 默认读取超时时间(秒)
	// 从 Redis 读取响应的最大等待时间
	DefaultReadTimeout = 3

	// DefaultWriteTimeout 默认写入超时时间(秒)
	// 向 Redis 写入命令的最大等待时间
	DefaultWriteTimeout = 3
)

// 日志消息常量
// 避免在代码中使用魔法字符串,便于统一管理和修改
const (
	// MsgCacheConnecting 缓存连接中消息
	MsgCacheConnecting = "connecting to redis"

	// MsgCacheConnected 缓存连接成功消息
	MsgCacheConnected = "redis connected successfully"

	// MsgCacheConnectionFailed 缓存连接失败消息
	MsgCacheConnectionFailed = "failed to connect to redis"

	// MsgCachePingSuccess 缓存 Ping 成功消息
	MsgCachePingSuccess = "redis ping successful"

	// MsgCachePingFailed 缓存 Ping 失败消息
	MsgCachePingFailed = "redis ping failed"

	// MsgCacheReloading 缓存重载中消息
	MsgCacheReloading = "reloading redis configuration"

	// MsgCacheReloaded 缓存重载成功消息
	MsgCacheReloaded = "redis configuration reloaded successfully"

	// MsgCacheReloadFailed 缓存重载失败消息
	MsgCacheReloadFailed = "failed to reload redis configuration"

	// MsgCacheClosing 缓存关闭中消息
	MsgCacheClosing = "closing redis connection"

	// MsgCacheClosed 缓存关闭成功消息
	MsgCacheClosed = "redis connection closed"
)

// 错误消息常量
// 用于创建错误时的统一消息格式
const (
	// ErrMsgNilValue 值为 nil 的错误消息
	ErrMsgNilValue = "redis: nil"

	// ErrMsgKeyNotFound 键不存在的错误消息
	ErrMsgKeyNotFound = "cache key not found: %s"

	// ErrMsgConnectionFailed 连接失败的错误消息
	ErrMsgConnectionFailed = "failed to connect to redis: %w"

	// ErrMsgPingFailed Ping 失败的错误消息
	ErrMsgPingFailed = "redis ping failed: %w"

	// ErrMsgOperationFailed 操作失败的错误消息格式
	// 使用 fmt.Sprintf(ErrMsgOperationFailed, operation, err)
	ErrMsgOperationFailed = "redis %s failed: %w"

	// ErrMsgReloadFailed 重载失败的错误消息
	ErrMsgReloadFailed = "failed to reload redis: %w"
)

// 键前缀常量
// 用于不同类型数据的键命名空间隔离
// 防止不同模块的键冲突
const (
	// KeyPrefixUser 用户相关数据的键前缀
	// 例如: user:profile:123
	KeyPrefixUser = "user:"

	// KeyPrefixSession 会话数据的键前缀
	// 例如: session:abc123
	KeyPrefixSession = "session:"

	// KeyPrefixCache 通用缓存数据的键前缀
	// 例如: cache:article:456
	KeyPrefixCache = "cache:"

	// KeyPrefixLock 分布式锁的键前缀
	// 例如: lock:resource:789
	KeyPrefixLock = "lock:"

	// KeyPrefixCounter 计数器的键前缀
	// 例如: counter:page_views
	KeyPrefixCounter = "counter:"
)

// 过期时间常量
// 常用的缓存过期时间预设
const (
	// ExpirationShort 短期过期时间: 5 分钟
	// 适用于频繁变化的数据
	ExpirationShort = 5 * 60 // 秒

	// ExpirationMedium 中期过期时间: 1 小时
	// 适用于一般缓存数据
	ExpirationMedium = 60 * 60 // 秒

	// ExpirationLong 长期过期时间: 24 小时
	// 适用于相对稳定的数据
	ExpirationLong = 24 * 60 * 60 // 秒

	// ExpirationNever 永不过期
	// 0 表示键不会自动过期
	ExpirationNever = 0
)

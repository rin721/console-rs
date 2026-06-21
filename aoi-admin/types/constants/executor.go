package constants

// 本文件定义跨层共享常量，避免内部基础设施包反向污染公共 types 边界。

// 协程池名称常量
// 用于引用配置文件中定义的池。
// 所有使用 executor 的应用层代码应该引用这些常量而非硬编码字符串。
//
// 设计原则:
// - 保持为应用层常量,不直接依赖 pkg/executor
// - 常量值必须与 config.yaml 中的池名称一致
// - 每个池都应该有明确的用途说明
const (
	// AppPoolHTTP HTTP请求处理池
	// 用途: 处理HTTP请求相关的异步任务（如请求日志记录、统计上报等）
	// 特性: 非阻塞模式，池满时立即返回错误，避免阻塞请求处理
	// 配置: size=200, expiry=10s, nonBlocking=true
	AppPoolHTTP = "http"

	// AppPoolDatabase 数据库操作池
	// 用途: 执行数据库相关的异步任务（如后台统计、批量更新、数据同步等）
	// 特性: 阻塞模式，确保数据库任务最终完成，避免数据丢失
	// 配置: size=50, expiry=30s, nonBlocking=false
	AppPoolDatabase = "database"

	// AppPoolCache 缓存操作池
	// 用途: 异步缓存更新、缓存预热、缓存失效等操作
	// 特性: 非阻塞模式，缓存操作失败可降级，不影响业务主流程
	// 配置: size=30, expiry=15s, nonBlocking=true
	AppPoolCache = "cache"

	// AppPoolLogger 日志处理池
	// 用途: 异步日志刷新、日志上传、日志归档等操作
	// 特性: 阻塞模式，确保日志不丢失，保证审计和追踪的完整性
	// 配置: size=10, expiry=60s, nonBlocking=false
	AppPoolLogger = "logger"

	// AppPoolBackground 后台任务池
	// 用途: 通用后台任务（邮件发送、消息推送、文件处理、定时任务等）
	// 特性: 非阻塞模式，允许任务溢出，避免影响核心业务
	// 配置: size=30, expiry=60s, nonBlocking=true
	AppPoolBackground = "background"
)

package constants

// 本文件定义跨层共享常量，避免内部基础设施包反向污染公共 types 边界。

import "time"

const (
	// CacheKeyPrefixUser 是用户缓存的前缀
	CacheKeyPrefixUser = "user:"

	// CacheTTLUser 是用户缓存的过期时间
	CacheTTLUser = 30 * time.Minute
)

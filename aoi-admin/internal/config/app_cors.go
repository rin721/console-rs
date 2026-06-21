package config

// 本文件定义一个配置分区及其校验规则，是外部配置进入运行时基础设施前的类型化边界。

import (
	"fmt"
	"strings"
)

var defaultCORSAllowHeaders = []string{"Origin", "Content-Type", "X-Request-ID", "Authorization", "X-Locale", "X-Setup-Token"}

// CORSConfig 跨域资源共享(CORS)配置
// 控制浏览器跨域访问策略
type CORSConfig struct {
	// Enabled 是否启用 CORS 中间件
	// true: 启用跨域支持
	// false: 禁用(所有跨域请求将被浏览器阻止)
	// 开发环境通常启用,生产环境根据需求决定
	Enabled bool `mapstructure:"enabled" envname:"CORS_ENABLED" json:"enabled" yaml:"enabled" toml:"enabled"`

	// AllowOrigins 允许的源列表
	// 指定哪些域名可以跨域访问
	// 格式:
	//   - 精确匹配: "http://localhost:3000"
	//   - 通配符: "*" (允许所有源,不安全,仅开发环境使用)
	// 示例: ["http://localhost:3000", "https://example.com"]
	// 安全建议: 生产环境必须明确列出允许的域名,禁止使用通配符
	AllowOrigins []string `mapstructure:"allow_origins" envname:"CORS_ALLOW_ORIGINS" json:"allow_origins" yaml:"allow_origins" toml:"allow_origins"`

	// AllowMethods 允许的 HTTP 方法
	// 指定跨域请求允许使用的 HTTP 方法
	// 常用方法: GET, POST, PUT, DELETE, PATCH, OPTIONS
	// OPTIONS 用于预检请求,通常需要包含
	// 示例: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
	AllowMethods []string `mapstructure:"allow_methods" envname:"CORS_ALLOW_METHODS" json:"allow_methods" yaml:"allow_methods" toml:"allow_methods"`

	// AllowHeaders 允许的请求头
	// 指定跨域请求允许携带的自定义请求头
	// 常用头:
	//   - Origin: 必需,标识请求来源
	//   - Content-Type: 必需,指定请求体类型
	//   - X-Request-ID: 用于请求追踪
	// 示例: ["Origin", "Content-Type", "X-Request-ID"]
	AllowHeaders []string `mapstructure:"allow_headers" envname:"CORS_ALLOW_HEADERS" json:"allow_headers" yaml:"allow_headers" toml:"allow_headers"`

	// ExposeHeaders 暴露给浏览器的响应头
	// 默认情况下浏览器只能访问简单响应头(如 Content-Type)
	// 通过此配置可以让浏览器访问自定义响应头
	// 常用场景:
	//   - X-Request-ID: 让前端获取请求追踪ID
	//   - X-Total-Count: 分页总数
	// 示例: ["X-Request-ID", "X-Total-Count"]
	ExposeHeaders []string `mapstructure:"expose_headers" envname:"CORS_EXPOSE_HEADERS" json:"expose_headers" yaml:"expose_headers" toml:"expose_headers"`

	// AllowCredentials 是否允许携带凭证
	// true: 允许跨域请求携带 Cookie、HTTP Auth 等凭证
	// false: 不允许携带凭证
	// 安全警告:
	//   - 设置为 true 时,AllowOrigins 不能使用通配符 "*"
	//   - 必须明确指定允许的域名
	// 使用场景: 需要在跨域请求中保持用户会话
	AllowCredentials bool `mapstructure:"allow_credentials" envname:"CORS_ALLOW_CREDENTIALS" json:"allow_credentials" yaml:"allow_credentials" toml:"allow_credentials"`

	// MaxAge 预检请求缓存时间(秒)
	// 浏览器会缓存 OPTIONS 预检请求的结果
	// 在缓存有效期内,相同的跨域请求不会再发送预检请求
	// 推荐值:
	//   - 开发环境: 600-3600 (10分钟-1小时)
	//   - 生产环境: 3600-86400 (1小时-24小时)
	// 作用: 减少网络开销,提升性能
	MaxAge int `mapstructure:"max_age" envname:"CORS_MAX_AGE" json:"max_age" yaml:"max_age" toml:"max_age"`
}

// ValidateName 返回配置名称
// 实现 Validator 接口
func (c *CORSConfig) ValidateName() string {
	return AppCORSName
}

// ValidateRequired 返回是否为必需配置
// CORS 配置是可选的,通过 Enabled 字段控制
func (c *CORSConfig) ValidateRequired() bool {
	return false
}

// Validate 验证 CORS 配置有效性
// 实现 Validator 接口
// 验证规则:
//  1. 如果未启用,跳过验证
//  2. 如果启用了 AllowCredentials,AllowOrigins 不能包含通配符 "*"
//  3. MaxAge 不能为负数
func (c *CORSConfig) Validate() error {
	// 如果未启用,跳过验证
	if !c.Enabled {
		return nil
	}

	// 验证 AllowCredentials 和 AllowOrigins 的组合
	// 这是 CORS 规范的安全要求
	if c.AllowCredentials {
		for _, origin := range c.AllowOrigins {
			if origin == "*" {
				return fmt.Errorf("allow_origins cannot contain wildcard \"*\" when allow_credentials is true")
			}
		}
	}

	// 验证 MaxAge
	if c.MaxAge < 0 {
		return fmt.Errorf("max_age must be non-negative, got %d", c.MaxAge)
	}

	return nil
}

// DefaultConfig 设置默认配置
// 提供开发环境友好的默认值
// 生产环境应该通过配置文件或环境变量覆盖
func (c *CORSConfig) DefaultConfig() {
	// 默认启用 CORS
	if !c.Enabled {
		c.Enabled = true
	}

	// 如果未配置允许的源,默认允许所有源(仅适用开发环境)
	if len(c.AllowOrigins) == 0 {
		c.AllowOrigins = []string{"*"}
	}

	// 如果未配置允许的方法,使用常用 HTTP 方法
	if len(c.AllowMethods) == 0 {
		c.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	}

	// 如果未配置允许的请求头,使用常用请求头
	if len(c.AllowHeaders) == 0 {
		c.AllowHeaders = append([]string(nil), defaultCORSAllowHeaders...)
	} else {
		c.AllowHeaders = ensureCORSHeaders(c.AllowHeaders, defaultCORSAllowHeaders...)
	}

	// 如果未配置暴露的响应头,默认暴露请求追踪ID
	if len(c.ExposeHeaders) == 0 {
		c.ExposeHeaders = []string{"X-Request-ID"}
	}

	// 如果未配置 MaxAge,默认 1 小时
	if c.MaxAge == 0 {
		c.MaxAge = 3600
	}
}

// OverrideConfig 从环境变量覆盖配置。
// 环境变量命名规则: <动态应用前缀>_CORS_<字段名>,全大写,单词间用下划线。
// 当前 AppPrefix=Rin 时主变量形如 RIN_APP_CORS_ENABLED；未加前缀变量保留为兼容 fallback。
// 支持的环境变量:
//   - CORS_ENABLED: 是否启用(true/false)
//   - CORS_ALLOW_ORIGINS: 允许的源(逗号分隔)
//   - CORS_ALLOW_METHODS: 允许的方法(逗号分隔)
//   - CORS_ALLOW_HEADERS: 允许的请求头(逗号分隔)
//   - CORS_EXPOSE_HEADERS: 暴露的响应头(逗号分隔)
//   - CORS_ALLOW_CREDENTIALS: 是否允许凭证(true/false)
//   - CORS_MAX_AGE: 预检缓存时间(秒)
func (c *CORSConfig) OverrideConfig() {
	overrideConfigFromEnv(c)
}

func ensureCORSHeaders(headers []string, required ...string) []string {
	out := append([]string(nil), headers...)
	for _, header := range required {
		if containsCORSHeader(out, header) {
			continue
		}
		out = append(out, header)
	}
	return out
}

func containsCORSHeader(headers []string, target string) bool {
	for _, header := range headers {
		if strings.EqualFold(strings.TrimSpace(header), target) {
			return true
		}
	}
	return false
}

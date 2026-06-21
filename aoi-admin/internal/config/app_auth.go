package config

import (
	"fmt"
	"strings"
)

const (
	DefaultAccessTokenTTLSeconds         = 900
	DefaultRefreshTokenTTLSeconds        = 604800
	DefaultInvitationTTLSeconds          = 86400
	DefaultEmailVerificationTTLSeconds   = 86400
	DefaultPasswordResetTTLSeconds       = 1800
	DefaultCasbinReloadIntervalSeconds   = 300
	DefaultLoginMaxFailures              = 5
	DefaultLoginLockMinutes              = 15
	DefaultCaptchaTTLSeconds             = 120
	DefaultMFAIssuer                     = "aoi-admin"
	DefaultNotificationDriver            = "debug"
	DefaultAuthCookieNamePrefix          = "aoi"
	DefaultAuthCookiePath                = "/"
	DefaultAuthCookieSameSite            = "lax"
	DefaultAuthCSRFCookieName            = "aoi_csrf"
	DefaultAuthCSRFHeaderName            = "X-CSRF-Token"
	DefaultAuthProductHeader             = "X-Aoi-Product-Code"
	DefaultAuthClientTypeHeader          = "X-Aoi-Client-Type"
	DefaultAuthClientType                = "pc_web"
	DefaultAuthCacheUserTTLSeconds       = 120
	DefaultAuthCacheOrgTTLSeconds        = 120
	DefaultAuthCacheRoleTTLSeconds       = 120
	DefaultAuthCachePermissionTTLSeconds = 300
	SMTPSecurityNone                     = "none"
	SMTPSecurityStartTLS                 = "starttls"
	SMTPSecurityTLS                      = "tls"
	DefaultPasswordMinLength             = 8
	RegistrationModeDisabled             = "disabled"
	RegistrationModeDirect               = "direct"
	RegistrationModeEmailVerification    = "email_verification"
	RegistrationModeInviteOnly           = "invite_only"
	DefaultRegistrationMode              = RegistrationModeDisabled
)

// AuthConfig 控制本地账号 IAM 模块的认证、安全策略和通知配置。
type AuthConfig struct {
	Enabled                     bool                 `mapstructure:"enabled" envname:"AUTH_ENABLED" json:"enabled" yaml:"enabled" toml:"enabled"`
	RegistrationMode            string               `mapstructure:"registration_mode" envname:"AUTH_REGISTRATION_MODE" json:"registration_mode" yaml:"registration_mode" toml:"registration_mode"`
	Issuer                      string               `mapstructure:"issuer" envname:"AUTH_ISSUER" json:"issuer" yaml:"issuer" toml:"issuer"`
	Audience                    []string             `mapstructure:"audience" envname:"AUTH_AUDIENCE" json:"audience" yaml:"audience" toml:"audience"`
	SigningKey                  string               `mapstructure:"signing_key" envname:"AUTH_SIGNING_KEY" json:"signing_key" yaml:"signing_key" toml:"signing_key"`
	AccessTokenTTLSeconds       int                  `mapstructure:"access_token_ttl_seconds" envname:"AUTH_ACCESS_TOKEN_TTL_SECONDS" json:"access_token_ttl_seconds" yaml:"access_token_ttl_seconds" toml:"access_token_ttl_seconds"`
	RefreshTokenTTLSeconds      int                  `mapstructure:"refresh_token_ttl_seconds" envname:"AUTH_REFRESH_TOKEN_TTL_SECONDS" json:"refresh_token_ttl_seconds" yaml:"refresh_token_ttl_seconds" toml:"refresh_token_ttl_seconds"`
	RefreshTokenPepper          string               `mapstructure:"refresh_token_pepper" envname:"AUTH_REFRESH_TOKEN_PEPPER" json:"refresh_token_pepper" yaml:"refresh_token_pepper" toml:"refresh_token_pepper"`
	MFAIssuer                   string               `mapstructure:"mfa_issuer" envname:"AUTH_MFA_ISSUER" json:"mfa_issuer" yaml:"mfa_issuer" toml:"mfa_issuer"`
	MFASecretKey                string               `mapstructure:"mfa_secret_key" envname:"AUTH_MFA_SECRET_KEY" json:"mfa_secret_key" yaml:"mfa_secret_key" toml:"mfa_secret_key"`
	LoginMaxFailures            int                  `mapstructure:"login_max_failures" envname:"AUTH_LOGIN_MAX_FAILURES" json:"login_max_failures" yaml:"login_max_failures" toml:"login_max_failures"`
	LoginLockMinutes            int                  `mapstructure:"login_lock_minutes" envname:"AUTH_LOGIN_LOCK_MINUTES" json:"login_lock_minutes" yaml:"login_lock_minutes" toml:"login_lock_minutes"`
	LoginCaptchaEnabled         bool                 `mapstructure:"login_captcha_enabled" envname:"AUTH_LOGIN_CAPTCHA_ENABLED" json:"login_captcha_enabled" yaml:"login_captcha_enabled" toml:"login_captcha_enabled"`
	CaptchaTTLSeconds           int                  `mapstructure:"captcha_ttl_seconds" envname:"AUTH_CAPTCHA_TTL_SECONDS" json:"captcha_ttl_seconds" yaml:"captcha_ttl_seconds" toml:"captcha_ttl_seconds"`
	InvitationTTLSeconds        int                  `mapstructure:"invitation_ttl_seconds" envname:"AUTH_INVITATION_TTL_SECONDS" json:"invitation_ttl_seconds" yaml:"invitation_ttl_seconds" toml:"invitation_ttl_seconds"`
	EmailVerificationTTLSeconds int                  `mapstructure:"email_verification_ttl_seconds" envname:"AUTH_EMAIL_VERIFICATION_TTL_SECONDS" json:"email_verification_ttl_seconds" yaml:"email_verification_ttl_seconds" toml:"email_verification_ttl_seconds"`
	PasswordResetTTLSeconds     int                  `mapstructure:"password_reset_ttl_seconds" envname:"AUTH_PASSWORD_RESET_TTL_SECONDS" json:"password_reset_ttl_seconds" yaml:"password_reset_ttl_seconds" toml:"password_reset_ttl_seconds"`
	NotificationDriver          string               `mapstructure:"notification_driver" envname:"AUTH_NOTIFICATION_DRIVER" json:"notification_driver" yaml:"notification_driver" toml:"notification_driver"`
	Cookie                      AuthCookieConfig     `mapstructure:"cookie" json:"cookie" yaml:"cookie" toml:"cookie"`
	CSRF                        AuthCSRFConfig       `mapstructure:"csrf" json:"csrf" yaml:"csrf" toml:"csrf"`
	Session                     AuthSessionConfig    `mapstructure:"session" json:"session" yaml:"session" toml:"session"`
	Cache                       AuthCacheConfig      `mapstructure:"cache" json:"cache" yaml:"cache" toml:"cache"`
	SMTP                        SMTPConfig           `mapstructure:"smtp" json:"smtp" yaml:"smtp" toml:"smtp"`
	PasswordPolicy              PasswordPolicyConfig `mapstructure:"password_policy" json:"password_policy" yaml:"password_policy" toml:"password_policy"`
	CasbinReloadIntervalSeconds int                  `mapstructure:"casbin_reload_interval_seconds" envname:"AUTH_CASBIN_RELOAD_INTERVAL_SECONDS" json:"casbin_reload_interval_seconds" yaml:"casbin_reload_interval_seconds" toml:"casbin_reload_interval_seconds"`
}

type AuthCookieConfig struct {
	NamePrefix string `mapstructure:"name_prefix" envname:"AUTH_COOKIE_NAME_PREFIX" json:"name_prefix" yaml:"name_prefix" toml:"name_prefix"`
	Domain     string `mapstructure:"domain" envname:"AUTH_COOKIE_DOMAIN" json:"domain" yaml:"domain" toml:"domain"`
	Path       string `mapstructure:"path" envname:"AUTH_COOKIE_PATH" json:"path" yaml:"path" toml:"path"`
	SameSite   string `mapstructure:"same_site" envname:"AUTH_COOKIE_SAME_SITE" json:"same_site" yaml:"same_site" toml:"same_site"`
	Secure     bool   `mapstructure:"secure" envname:"AUTH_COOKIE_SECURE" json:"secure" yaml:"secure" toml:"secure"`
}

type AuthCSRFConfig struct {
	Enabled    *bool  `mapstructure:"enabled" envname:"AUTH_CSRF_ENABLED" json:"enabled" yaml:"enabled" toml:"enabled"`
	CookieName string `mapstructure:"cookie_name" envname:"AUTH_CSRF_COOKIE_NAME" json:"cookie_name" yaml:"cookie_name" toml:"cookie_name"`
	HeaderName string `mapstructure:"header_name" envname:"AUTH_CSRF_HEADER_NAME" json:"header_name" yaml:"header_name" toml:"header_name"`
}

type AuthSessionConfig struct {
	SinglePerProductPlatform *bool    `mapstructure:"single_per_product_platform" envname:"AUTH_SESSION_SINGLE_PER_PRODUCT_PLATFORM" json:"single_per_product_platform" yaml:"single_per_product_platform" toml:"single_per_product_platform"`
	ProductHeader            string   `mapstructure:"product_header" envname:"AUTH_SESSION_PRODUCT_HEADER" json:"product_header" yaml:"product_header" toml:"product_header"`
	ClientTypeHeader         string   `mapstructure:"client_type_header" envname:"AUTH_SESSION_CLIENT_TYPE_HEADER" json:"client_type_header" yaml:"client_type_header" toml:"client_type_header"`
	DefaultClientType        string   `mapstructure:"default_client_type" envname:"AUTH_SESSION_DEFAULT_CLIENT_TYPE" json:"default_client_type" yaml:"default_client_type" toml:"default_client_type"`
	MobileUserAgentHints     []string `mapstructure:"mobile_user_agent_hints" envname:"AUTH_SESSION_MOBILE_USER_AGENT_HINTS" json:"mobile_user_agent_hints" yaml:"mobile_user_agent_hints" toml:"mobile_user_agent_hints"`
}

type AuthCacheConfig struct {
	Enabled              *bool `mapstructure:"enabled" envname:"AUTH_CACHE_ENABLED" json:"enabled" yaml:"enabled" toml:"enabled"`
	UserTTLSeconds       int   `mapstructure:"user_ttl_seconds" envname:"AUTH_CACHE_USER_TTL_SECONDS" json:"user_ttl_seconds" yaml:"user_ttl_seconds" toml:"user_ttl_seconds"`
	OrgTTLSeconds        int   `mapstructure:"org_ttl_seconds" envname:"AUTH_CACHE_ORG_TTL_SECONDS" json:"org_ttl_seconds" yaml:"org_ttl_seconds" toml:"org_ttl_seconds"`
	RoleTTLSeconds       int   `mapstructure:"role_ttl_seconds" envname:"AUTH_CACHE_ROLE_TTL_SECONDS" json:"role_ttl_seconds" yaml:"role_ttl_seconds" toml:"role_ttl_seconds"`
	PermissionTTLSeconds int   `mapstructure:"permission_ttl_seconds" envname:"AUTH_CACHE_PERMISSION_TTL_SECONDS" json:"permission_ttl_seconds" yaml:"permission_ttl_seconds" toml:"permission_ttl_seconds"`
}

func (c AuthCSRFConfig) EnabledValue() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

func (c AuthSessionConfig) SinglePerProductPlatformValue() bool {
	if c.SinglePerProductPlatform == nil {
		return true
	}
	return *c.SinglePerProductPlatform
}

func (c AuthCacheConfig) EnabledValue() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// SMTPConfig 定义 IAM 邀请和找回密码邮件通知的 SMTP 连接参数。
type SMTPConfig struct {
	Host     string `mapstructure:"host" envname:"AUTH_SMTP_HOST" json:"host" yaml:"host" toml:"host"`
	Port     int    `mapstructure:"port" envname:"AUTH_SMTP_PORT" json:"port" yaml:"port" toml:"port"`
	Username string `mapstructure:"username" envname:"AUTH_SMTP_USERNAME" json:"username" yaml:"username" toml:"username"`
	Password string `mapstructure:"password" envname:"AUTH_SMTP_PASSWORD" json:"password" yaml:"password" toml:"password"`
	From     string `mapstructure:"from" envname:"AUTH_SMTP_FROM" json:"from" yaml:"from" toml:"from"`
	FromName string `mapstructure:"from_name" envname:"AUTH_SMTP_FROM_NAME" json:"from_name" yaml:"from_name" toml:"from_name"`
	Security string `mapstructure:"security" envname:"AUTH_SMTP_SECURITY" json:"security" yaml:"security" toml:"security"`
}

// PasswordPolicyConfig 定义账号创建和密码重置时的最低密码要求。
type PasswordPolicyConfig struct {
	MinLength     int  `mapstructure:"min_length" envname:"AUTH_PASSWORD_MIN_LENGTH" json:"min_length" yaml:"min_length" toml:"min_length"`
	RequireLower  bool `mapstructure:"require_lower" envname:"AUTH_PASSWORD_REQUIRE_LOWER" json:"require_lower" yaml:"require_lower" toml:"require_lower"`
	RequireUpper  bool `mapstructure:"require_upper" envname:"AUTH_PASSWORD_REQUIRE_UPPER" json:"require_upper" yaml:"require_upper" toml:"require_upper"`
	RequireNumber bool `mapstructure:"require_number" envname:"AUTH_PASSWORD_REQUIRE_NUMBER" json:"require_number" yaml:"require_number" toml:"require_number"`
	RequireSymbol bool `mapstructure:"require_symbol" envname:"AUTH_PASSWORD_REQUIRE_SYMBOL" json:"require_symbol" yaml:"require_symbol" toml:"require_symbol"`
}

func (c *AuthConfig) ValidateName() string {
	return AppAuthName
}

func (c *AuthConfig) ValidateRequired() bool {
	return false
}

func (c *AuthConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	c.ApplyDefaults()
	if c.Issuer == "" {
		return fmt.Errorf("issuer is required")
	}
	if len(c.SigningKey) < 32 {
		return fmt.Errorf("signing_key must be at least 32 bytes")
	}
	if c.RefreshTokenPepper == "" {
		return fmt.Errorf("refresh_token_pepper is required")
	}
	if len(c.MFASecretKey) < 32 {
		return fmt.Errorf("mfa_secret_key must be at least 32 bytes")
	}
	if c.AccessTokenTTLSeconds <= 0 || c.RefreshTokenTTLSeconds <= 0 {
		return fmt.Errorf("token ttl values must be positive")
	}
	if !validRegistrationMode(c.RegistrationMode) {
		return fmt.Errorf("registration_mode must be one of disabled, direct, email_verification, invite_only")
	}
	if c.InvitationTTLSeconds <= 0 || c.EmailVerificationTTLSeconds <= 0 || c.PasswordResetTTLSeconds <= 0 {
		return fmt.Errorf("invitation, email verification, and password reset ttl values must be positive")
	}
	if c.LoginMaxFailures <= 0 || c.LoginLockMinutes <= 0 {
		return fmt.Errorf("login lock policy values must be positive")
	}
	if c.LoginCaptchaEnabled && c.CaptchaTTLSeconds <= 0 {
		return fmt.Errorf("captcha_ttl_seconds must be positive when login captcha is enabled")
	}
	if strings.TrimSpace(c.Cookie.NamePrefix) == "" {
		return fmt.Errorf("cookie.name_prefix is required")
	}
	if strings.TrimSpace(c.Cookie.Path) == "" {
		return fmt.Errorf("cookie.path is required")
	}
	if !validCookieSameSite(c.Cookie.SameSite) {
		return fmt.Errorf("cookie.same_site must be one of lax, strict, none")
	}
	if c.CSRF.EnabledValue() {
		if strings.TrimSpace(c.CSRF.CookieName) == "" || strings.TrimSpace(c.CSRF.HeaderName) == "" {
			return fmt.Errorf("csrf cookie_name and header_name are required when csrf is enabled")
		}
	}
	if strings.TrimSpace(c.Session.ProductHeader) == "" {
		return fmt.Errorf("session.product_header is required")
	}
	if strings.TrimSpace(c.Session.ClientTypeHeader) == "" {
		return fmt.Errorf("session.client_type_header is required")
	}
	if strings.TrimSpace(c.Session.DefaultClientType) == "" {
		return fmt.Errorf("session.default_client_type is required")
	}
	if c.Cache.EnabledValue() {
		if c.Cache.UserTTLSeconds <= 0 || c.Cache.OrgTTLSeconds <= 0 || c.Cache.RoleTTLSeconds <= 0 || c.Cache.PermissionTTLSeconds <= 0 {
			return fmt.Errorf("auth cache ttl values must be positive when auth cache is enabled")
		}
	}
	if c.PasswordPolicy.MinLength <= 0 {
		return fmt.Errorf("password policy min_length must be positive")
	}
	if strings.EqualFold(c.NotificationDriver, "smtp") {
		if c.SMTP.Host == "" || c.SMTP.Port <= 0 || c.SMTP.From == "" {
			return fmt.Errorf("smtp host, port, and from are required when notification_driver is smtp")
		}
		if c.SMTP.Security == "" {
			return fmt.Errorf("smtp security is required when notification_driver is smtp")
		}
		if !validSMTPSecurity(c.SMTP.Security) {
			return fmt.Errorf("smtp security must be one of none, starttls, tls")
		}
	}
	return nil
}

func (c *AuthConfig) ApplyDefaults() {
	c.RegistrationMode = strings.ToLower(strings.TrimSpace(c.RegistrationMode))
	if c.RegistrationMode == "" {
		c.RegistrationMode = DefaultRegistrationMode
	}
	if c.Issuer == "" {
		c.Issuer = "aoi-admin"
	}
	if len(c.Audience) == 0 {
		c.Audience = []string{"aoi-admin-api"}
	}
	if c.AccessTokenTTLSeconds == 0 {
		c.AccessTokenTTLSeconds = DefaultAccessTokenTTLSeconds
	}
	if c.RefreshTokenTTLSeconds == 0 {
		c.RefreshTokenTTLSeconds = DefaultRefreshTokenTTLSeconds
	}
	if c.MFAIssuer == "" {
		c.MFAIssuer = DefaultMFAIssuer
	}
	if c.LoginMaxFailures == 0 {
		c.LoginMaxFailures = DefaultLoginMaxFailures
	}
	if c.LoginLockMinutes == 0 {
		c.LoginLockMinutes = DefaultLoginLockMinutes
	}
	if c.CaptchaTTLSeconds == 0 {
		c.CaptchaTTLSeconds = DefaultCaptchaTTLSeconds
	}
	if c.InvitationTTLSeconds == 0 {
		c.InvitationTTLSeconds = DefaultInvitationTTLSeconds
	}
	if c.EmailVerificationTTLSeconds == 0 {
		c.EmailVerificationTTLSeconds = DefaultEmailVerificationTTLSeconds
	}
	if c.PasswordResetTTLSeconds == 0 {
		c.PasswordResetTTLSeconds = DefaultPasswordResetTTLSeconds
	}
	if c.CasbinReloadIntervalSeconds == 0 {
		c.CasbinReloadIntervalSeconds = DefaultCasbinReloadIntervalSeconds
	}
	if c.NotificationDriver == "" {
		c.NotificationDriver = DefaultNotificationDriver
	}
	if strings.TrimSpace(c.Cookie.NamePrefix) == "" {
		c.Cookie.NamePrefix = DefaultAuthCookieNamePrefix
	}
	if strings.TrimSpace(c.Cookie.Path) == "" {
		c.Cookie.Path = DefaultAuthCookiePath
	}
	if strings.TrimSpace(c.Cookie.SameSite) == "" {
		c.Cookie.SameSite = DefaultAuthCookieSameSite
	}
	c.Cookie.SameSite = strings.ToLower(strings.TrimSpace(c.Cookie.SameSite))
	if strings.TrimSpace(c.CSRF.CookieName) == "" {
		c.CSRF.CookieName = DefaultAuthCSRFCookieName
	}
	if strings.TrimSpace(c.CSRF.HeaderName) == "" {
		c.CSRF.HeaderName = DefaultAuthCSRFHeaderName
	}
	if strings.TrimSpace(c.Session.ProductHeader) == "" {
		c.Session.ProductHeader = DefaultAuthProductHeader
	}
	if strings.TrimSpace(c.Session.ClientTypeHeader) == "" {
		c.Session.ClientTypeHeader = DefaultAuthClientTypeHeader
	}
	if strings.TrimSpace(c.Session.DefaultClientType) == "" {
		c.Session.DefaultClientType = DefaultAuthClientType
	}
	c.Session.DefaultClientType = normalizeAuthClientType(c.Session.DefaultClientType)
	if len(c.Session.MobileUserAgentHints) == 0 {
		c.Session.MobileUserAgentHints = []string{"mobile", "android", "iphone", "ipad"}
	}
	if c.Cache.UserTTLSeconds == 0 {
		c.Cache.UserTTLSeconds = DefaultAuthCacheUserTTLSeconds
	}
	if c.Cache.OrgTTLSeconds == 0 {
		c.Cache.OrgTTLSeconds = DefaultAuthCacheOrgTTLSeconds
	}
	if c.Cache.RoleTTLSeconds == 0 {
		c.Cache.RoleTTLSeconds = DefaultAuthCacheRoleTTLSeconds
	}
	if c.Cache.PermissionTTLSeconds == 0 {
		c.Cache.PermissionTTLSeconds = DefaultAuthCachePermissionTTLSeconds
	}
	c.SMTP.Security = strings.ToLower(strings.TrimSpace(c.SMTP.Security))
	if c.SMTP.Port == 0 {
		switch c.SMTP.Security {
		case SMTPSecurityTLS:
			c.SMTP.Port = 465
		case SMTPSecurityNone:
			c.SMTP.Port = 25
		default:
			c.SMTP.Port = 587
		}
	}
	if c.PasswordPolicy.MinLength == 0 {
		c.PasswordPolicy.MinLength = DefaultPasswordMinLength
	}
}

func validSMTPSecurity(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case SMTPSecurityNone, SMTPSecurityStartTLS, SMTPSecurityTLS:
		return true
	default:
		return false
	}
}

func validRegistrationMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case RegistrationModeDisabled, RegistrationModeDirect, RegistrationModeEmailVerification, RegistrationModeInviteOnly:
		return true
	default:
		return false
	}
}

func validCookieSameSite(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "lax", "strict", "none":
		return true
	default:
		return false
	}
}

func normalizeAuthClientType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

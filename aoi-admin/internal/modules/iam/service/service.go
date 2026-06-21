// Package service 承载 IAM 模块的认证、授权、组织和审计应用层规则。
package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/rei0721/go-scaffold/internal/modules/iam/model"
)

var (
	ErrInvalidInput         = errors.New("invalid iam input")
	ErrUnauthorized         = errors.New("invalid credentials")
	ErrForbidden            = errors.New("permission denied")
	ErrNotFound             = errors.New("iam resource not found")
	ErrDuplicate            = errors.New("iam resource already exists")
	ErrMFARequired          = errors.New("mfa code required")
	ErrInvalidToken         = errors.New("invalid iam token")
	ErrAccountLocked        = errors.New("account locked")
	ErrAccountDisabled      = errors.New("account disabled")
	ErrCaptchaRequired      = errors.New("captcha code required")
	ErrCaptchaInvalid       = errors.New("invalid captcha code")
	ErrSessionRevoked       = errors.New("session revoked")
	ErrInvitationClosed     = errors.New("invitation is not available")
	ErrSignupDisabled       = errors.New("signup is disabled")
	ErrSetupCompleted       = errors.New("setup already completed")
	ErrNotificationDelivery = errors.New("notification delivery failed")
)

// Config 描述 IAM 服务的安全策略和外部可配置行为。
// 登录锁定、验证码、邀请、重置密码和密码策略都在服务层统一归一化默认值。
const (
	cacheScopeIAM         = "iam"
	cacheScopeEpoch       = "epoch"
	cacheScopeUserOrgs    = "user_orgs"
	cacheScopeOrgUsers    = "org_users"
	cacheScopeOrgRoles    = "org_roles"
	cacheScopePermissions = "permissions"
)

const (
	RegistrationModeDisabled          = "disabled"
	RegistrationModeDirect            = "direct"
	RegistrationModeEmailVerification = "email_verification"
	RegistrationModeInviteOnly        = "invite_only"

	SignupStatusAuthenticated       = "authenticated"
	SignupStatusVerificationPending = "verification_pending"
)

// Config 描述 IAM 服务的安全策略和外部可配置行为。
type Config struct {
	RegistrationMode        string
	MFAIssuer               string
	MFASecretKey            string
	LoginMaxFailures        int
	LoginLockDuration       time.Duration
	CaptchaEnabled          bool
	CaptchaTTL              time.Duration
	InvitationTTL           time.Duration
	EmailVerificationTTL    time.Duration
	PasswordResetTTL        time.Duration
	NotificationDriver      string
	PublicBaseURL           string
	DefaultProductCode      string
	DefaultClientType       string
	SingleSessionPerContext bool
	UserCacheTTL            time.Duration
	OrgCacheTTL             time.Duration
	RoleCacheTTL            time.Duration
	PermissionCacheTTL      time.Duration
	PasswordPolicy          PasswordPolicy
	Now                     func() time.Time
}

// PasswordPolicy 描述密码复杂度要求。
// 字段带 JSON tag 是为了直接暴露给初始化或前端配置接口。
type PasswordPolicy struct {
	MinLength     int  `json:"minLength"`
	RequireLower  bool `json:"requireLower"`
	RequireUpper  bool `json:"requireUpper"`
	RequireNumber bool `json:"requireNumber"`
	RequireSymbol bool `json:"requireSymbol"`
}

// Principal 表示通过认证后的调用主体。
// OrgID 和 SessionID 共同限定当前请求的租户上下文与会话边界。
type Principal struct {
	UserID      int64  `json:"userId,string"`
	OrgID       int64  `json:"orgId,string"`
	SessionID   int64  `json:"sessionId,string"`
	ProductCode string `json:"productCode"`
	ClientType  string `json:"clientType"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	RoleCode    string `json:"roleCode,omitempty"`
}

type PermissionContext struct {
	ProductCode string
	Scope       string
	Object      string
	Action      string
}

// TokenPair 是发给客户端的 access/refresh token 组合。
type TokenPair struct {
	AccessToken      string    `json:"accessToken"`
	AccessExpiresAt  time.Time `json:"accessExpiresAt"`
	RefreshToken     string    `json:"refreshToken"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
	UserID           int64     `json:"userId,string,omitempty"`
	OrgID            int64     `json:"orgId,string,omitempty"`
	SessionID        int64     `json:"sessionId,string,omitempty"`
	ProductCode      string    `json:"productCode,omitempty"`
	ClientType       string    `json:"clientType,omitempty"`
}

type SessionSnapshot struct {
	UserID           int64     `json:"userId,string"`
	OrgID            int64     `json:"orgId,string"`
	SessionID        int64     `json:"sessionId,string"`
	ProductCode      string    `json:"productCode"`
	ClientType       string    `json:"clientType"`
	AccessExpiresAt  time.Time `json:"accessExpiresAt"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
}

type AuthResult struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
	Snapshot         SessionSnapshot
}

func (p TokenPair) SessionSnapshot() SessionSnapshot {
	return SessionSnapshot{
		UserID:           p.UserID,
		OrgID:            p.OrgID,
		SessionID:        p.SessionID,
		ProductCode:      p.ProductCode,
		ClientType:       p.ClientType,
		AccessExpiresAt:  p.AccessExpiresAt,
		RefreshExpiresAt: p.RefreshExpiresAt,
	}
}

// LoginInput 描述用户名密码登录请求。
// Captcha 和 MFA 字段按配置与用户状态条件性生效。
type LoginInput struct {
	CaptchaCode string
	CaptchaID   string
	Identifier  string
	Password    string
	OrgCode     string
	MFACode     string
	UserAgent   string
	IPAddress   string
	ProductCode string
	ClientType  string
}

// CaptchaChallenge 表示一次登录验证码挑战。
// CaptchaEnabled 关闭时只返回 Enabled=false，不产生挑战状态。
type CaptchaChallenge struct {
	CaptchaID string    `json:"captchaId,omitempty"`
	Enabled   bool      `json:"enabled"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
	Image     string    `json:"image,omitempty"`
}

// SignupInput 描述自助注册组织和首个 owner 用户的请求。
type SignupInput struct {
	OrgCode     string
	OrgName     string
	Username    string
	Email       string
	DisplayName string
	Password    string
	UserAgent   string
	IPAddress   string
	ProductCode string
	ClientType  string
}

type SignupResult struct {
	Status   string                `json:"status"`
	Session  *SessionSnapshot      `json:"session,omitempty"`
	Delivery *NotificationDelivery `json:"delivery,omitempty"`
	Tokens   TokenPair             `json:"-"`
}

type ConfirmEmailVerificationInput struct {
	Token       string
	UserAgent   string
	IPAddress   string
	ProductCode string
	ClientType  string
}

// InitialAdminSetupInput 描述系统首次初始化管理员的请求。
// 该流程要求当前用户表为空，避免重复初始化覆盖已有租户。
type InitialAdminSetupInput struct {
	OrgCode     string
	OrgName     string
	Username    string
	Email       string
	DisplayName string
	Password    string
	UserAgent   string
	IPAddress   string
	ProductCode string
	ClientType  string
}

// RefreshInput 描述 refresh token 换发请求。
type RefreshInput struct {
	RefreshToken string
	UserAgent    string
	IPAddress    string
}

type SessionContext struct {
	ProductCode string
	ClientType  string
}

// BootstrapAdminInput 描述内部引导管理员的请求。
// 与首次初始化不同，它允许复用已存在组织或用户，用于部署脚本的幂等引导。
type BootstrapAdminInput struct {
	OrgCode     string
	OrgName     string
	Username    string
	Email       string
	DisplayName string
	Password    string
}

// InviteUserInput 描述管理员邀请用户加入当前组织的请求。
type InviteUserInput struct {
	Principal Principal
	Email     string
	RoleCode  string
	UserAgent string
	IPAddress string
}

// AcceptInvitationInput 描述受邀用户接受邀请并设置账号信息的请求。
type AcceptInvitationInput struct {
	Token       string
	Username    string
	DisplayName string
	Password    string
	UserAgent   string
	IPAddress   string
}

// ForgotPasswordInput 描述发起密码重置的一次性 token 请求。
type ForgotPasswordInput struct {
	Email     string
	UserAgent string
	IPAddress string
}

// ResetPasswordInput 描述使用一次性 token 重置密码的请求。
type ResetPasswordInput struct {
	Token       string
	NewPassword string
	UserAgent   string
	IPAddress   string
}

// CreateRoleInput 描述创建组织角色及其权限集合的请求。
type CreateRoleInput struct {
	Principal   Principal
	Code        string
	Name        string
	Description string
	Permissions []string
}

// UpdateUserInput 描述管理员更新组织成员状态或角色的请求。
// HasRoles 用于区分“不改角色”和“把角色更新为空列表”。
type UpdateUserInput struct {
	Principal Principal
	UserID    int64
	Status    *string
	Roles     []string
	HasRoles  bool
	UserAgent string
	IPAddress string
}

// UpdateRoleInput 描述更新角色基本信息和权限集合的请求。
// HasPermissions 用于区分“不改权限”和“清空权限”。
type UpdateRoleInput struct {
	Principal      Principal
	RoleID         int64
	Name           string
	Description    string
	Permissions    []string
	HasPermissions bool
	UserAgent      string
	IPAddress      string
}

// UpdateOrganizationInput 描述组织名称更新请求。
type UpdateOrganizationInput struct {
	Principal Principal
	OrgID     int64
	Name      string
	UserAgent string
	IPAddress string
}

// OrganizationListFilter 描述组织列表的内存过滤、排序和分页条件。
type OrganizationListFilter struct {
	Keyword  string
	Code     string
	Name     string
	Status   string
	OrderKey string
	Desc     bool
	Page     int
	PageSize int
}

// OrganizationPage 是组织列表分页响应。
type OrganizationPage struct {
	Items         []model.Organization `json:"items"`
	Page          int                  `json:"page"`
	PageSize      int                  `json:"pageSize"`
	Total         int64                `json:"total"`
	StorageStatus string               `json:"storageStatus"`
}

// CreateAPITokenInput 描述为用户创建 API token 的请求。
// Days 会被归一化到允许范围，RoleCode 约束 token 能代表的角色。
type CreateAPITokenInput struct {
	Principal Principal
	UserID    int64
	RoleCode  string
	Days      int
	Remark    string
	UserAgent string
	IPAddress string
}

// RevokeAPITokenInput 描述撤销 API token 的请求。
type RevokeAPITokenInput struct {
	Principal Principal
	TokenID   int64
	UserAgent string
	IPAddress string
}

// AuditLogFilter 描述审计日志查询条件。
type AuditLogFilter struct {
	Action string
	UserID int64
	From   time.Time
	To     time.Time
	Limit  int
	Cursor int64
}

// APITokenFilter 描述 API token 列表查询条件。
type APITokenFilter struct {
	Page     int
	PageSize int
	Status   string
	UserID   int64
	Now      time.Time
}

// UserListFilter 描述组织成员列表的过滤、排序和分页条件。
type UserListFilter struct {
	Keyword     string
	Username    string
	DisplayName string
	Email       string
	RoleCode    string
	Status      string
	OrderKey    string
	Desc        bool
	Page        int
	PageSize    int
}

// OrganizationUser 组合用户、成员状态和组织内角色。
type OrganizationUser struct {
	User             model.User `json:"user"`
	MembershipStatus string     `json:"membershipStatus"`
	Roles            []string   `json:"roles"`
}

// OrganizationUserPage 是组织成员列表分页响应。
type OrganizationUserPage struct {
	Items         []OrganizationUser `json:"items"`
	Page          int                `json:"page"`
	PageSize      int                `json:"pageSize"`
	Total         int64              `json:"total"`
	StorageStatus string             `json:"storageStatus"`
}

// SessionListFilter 描述会话列表过滤、排序和分页条件。
type SessionListFilter struct {
	Keyword     string
	UserID      int64
	IPAddress   string
	Status      string
	Scope       string
	ProductCode string
	ClientType  string
	OrderKey    string
	Desc        bool
	Page        int
	PageSize    int
}

// SessionPage 是会话列表分页响应。
type SessionPage struct {
	Items         []model.Session `json:"items"`
	Page          int             `json:"page"`
	PageSize      int             `json:"pageSize"`
	Total         int64           `json:"total"`
	StorageStatus string          `json:"storageStatus"`
}

// APITokenView 是 API token 的安全展示模型。
// 明文 token 只在创建时返回，列表中只展示前缀和状态信息。
type APITokenView struct {
	ID                int64      `json:"id,string"`
	OrgID             int64      `json:"orgId,string"`
	UserID            int64      `json:"userId,string"`
	Username          string     `json:"username"`
	UserDisplayName   string     `json:"userDisplayName"`
	RoleCode          string     `json:"roleCode"`
	TokenPrefix       string     `json:"tokenPrefix"`
	Status            string     `json:"status"`
	ExpiresAt         *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt        *time.Time `json:"lastUsedAt,omitempty"`
	LastUsedIPAddress string     `json:"lastUsedIpAddress"`
	RevokedAt         *time.Time `json:"revokedAt,omitempty"`
	RevokedBy         *int64     `json:"revokedBy,omitempty,string"`
	Remark            string     `json:"remark"`
	CreatedBy         int64      `json:"createdBy,string"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

// APITokenPage 是 API token 列表分页响应。
type APITokenPage struct {
	Items         []APITokenView `json:"items"`
	Page          int            `json:"page"`
	PageSize      int            `json:"pageSize"`
	Total         int64          `json:"total"`
	StorageStatus string         `json:"storageStatus"`
}

// CreateAPITokenResult 返回新建 token 记录和一次性明文 token。
type CreateAPITokenResult struct {
	Item  APITokenView `json:"item"`
	Token string       `json:"token"`
}

// NotificationDelivery 描述邀请或重置密码通知的调试返回。
// debug 模式会返回 token 或 URL，真实通知驱动通常只发送给用户邮箱。
type NotificationDelivery struct {
	Debug bool   `json:"debug"`
	Token string `json:"token,omitempty"`
	URL   string `json:"url,omitempty"`
}

// NotificationRuntimeConfig 描述运行期可热更新的通知策略。
type NotificationRuntimeConfig struct {
	NotificationDriver string
	PublicBaseURL      string
}

type RegistrationRuntimeConfig struct {
	RegistrationMode     string
	EmailVerificationTTL time.Duration
}

// NotificationRuntimeReloader 表示服务支持热更新通知策略。
type NotificationRuntimeReloader interface {
	ReloadNotificationRuntime(NotificationRuntimeConfig)
}

type RegistrationRuntimeReloader interface {
	ReloadRegistrationRuntime(RegistrationRuntimeConfig)
}

// ReloadableNotifier 允许组合根在配置变更后替换通知实现，而无需重建 IAM service。
type ReloadableNotifier struct {
	mu    sync.RWMutex
	inner Notifier
}

// NewReloadableNotifier 创建可热替换的通知器包装器。
func NewReloadableNotifier(inner Notifier) *ReloadableNotifier {
	if inner == nil {
		inner = NoopNotifier{}
	}
	return &ReloadableNotifier{inner: inner}
}

// Replace 原子替换后续通知请求使用的实现。
func (n *ReloadableNotifier) Replace(inner Notifier) {
	if inner == nil {
		inner = NoopNotifier{}
	}
	n.mu.Lock()
	n.inner = inner
	n.mu.Unlock()
}

func (n *ReloadableNotifier) SendInvitation(ctx context.Context, notice InvitationNotice) error {
	n.mu.RLock()
	inner := n.inner
	n.mu.RUnlock()
	return inner.SendInvitation(ctx, notice)
}

func (n *ReloadableNotifier) SendPasswordReset(ctx context.Context, notice PasswordResetNotice) error {
	n.mu.RLock()
	inner := n.inner
	n.mu.RUnlock()
	return inner.SendPasswordReset(ctx, notice)
}

func (n *ReloadableNotifier) SendEmailVerification(ctx context.Context, notice EmailVerificationNotice) error {
	n.mu.RLock()
	inner := n.inner
	n.mu.RUnlock()
	return inner.SendEmailVerification(ctx, notice)
}

// SetupStatus 描述系统是否仍需要首次管理员初始化。
type SetupStatus struct {
	Required       bool           `json:"required"`
	PasswordPolicy PasswordPolicy `json:"passwordPolicy"`
}

// Notifier 抽象邀请和密码重置通知发送能力。
type Notifier interface {
	SendInvitation(context.Context, InvitationNotice) error
	SendPasswordReset(context.Context, PasswordResetNotice) error
	SendEmailVerification(context.Context, EmailVerificationNotice) error
}

// InvitationNotice 是邀请邮件通知内容。
type InvitationNotice struct {
	Email string
	Token string
	URL   string
}

// PasswordResetNotice 是密码重置通知内容。
type PasswordResetNotice struct {
	Email string
	Token string
	URL   string
}

type EmailVerificationNotice struct {
	Email string
	Token string
	URL   string
}

// NoopNotifier 是默认空通知实现，避免未配置通知驱动时阻断主流程。
type NoopNotifier struct{}

func (NoopNotifier) SendInvitation(context.Context, InvitationNotice) error       { return nil }
func (NoopNotifier) SendPasswordReset(context.Context, PasswordResetNotice) error { return nil }
func (NoopNotifier) SendEmailVerification(context.Context, EmailVerificationNotice) error {
	return nil
}

// PasswordCrypto 抽象密码哈希和校验能力。
type PasswordCrypto interface {
	HashPassword(string) (string, error)
	VerifyPassword(string, string) error
}

// IDGenerator 为 IAM 资源生成数字和字符串 ID。
type IDGenerator interface {
	NextID() int64
	NextIDString() string
}

// TokenSubject 是签发 token 时写入声明的主体信息。
type TokenSubject struct {
	UserID      int64
	OrgID       int64
	SessionID   int64
	ProductCode string
}

// Token 类型用于区分 access token 与 refresh token 的解析语义。
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// TokenClaims 是 token 解析后的声明内容。
type TokenClaims struct {
	UserID      int64
	OrgID       int64
	SessionID   int64
	ProductCode string
	TokenType   string
}

// IssuedTokenPair 是 TokenManager 返回的签发结果。
// RefreshTokenHash 用于会话表持久化，避免明文 refresh token 落库。
type IssuedTokenPair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshTokenHash string
	RefreshExpiresAt time.Time
}

// TokenManager 抽象 access/refresh token 的签发、解析和哈希能力。
type TokenManager interface {
	IssueAccess(context.Context, TokenSubject) (string, time.Time, error)
	IssueRefresh(context.Context) (string, string, time.Time, error)
	IssuePair(context.Context, TokenSubject) (IssuedTokenPair, error)
	Parse(context.Context, string, string) (*TokenClaims, error)
	HashRefreshToken(string) string
}

type CacheStore interface {
	GetJSON(context.Context, string, any) (bool, error)
	SetJSON(context.Context, string, any, time.Duration) error
	Delete(context.Context, ...string) error
	Incr(context.Context, string, time.Duration) (int64, error)
}

// AuthorizationRule 表示一条可加载到授权引擎的策略规则。
type AuthorizationRule struct {
	PType  string
	Values []string
}

type RolePermission struct {
	ProductCode string
	Scope       string
	Code        string
}

// AuthorizerEnforcer 抽象 Casbin 等授权引擎的最小能力。
// 服务层通过它执行权限判断、角色绑定和策略加载。
type AuthorizerEnforcer interface {
	Enforce(context.Context, string, string, string, string, string, string) (bool, error)
	AddPolicy(context.Context, string, string, string, string, string, string) (bool, error)
	AddRoleForUser(context.Context, string, string, string) (bool, error)
	DeleteRoleForUser(context.Context, string, string, string) (bool, error)
	GetRolesForUser(context.Context, string, string) ([]string, error)
	LoadRules(context.Context, []AuthorizationRule) error
}

// TOTPKey 是创建 MFA 因子时返回的密钥和绑定 URL。
type TOTPKey struct {
	Secret string
	URL    string
}

// TOTPProvider 抽象 TOTP 密钥生成和验证码校验。
type TOTPProvider interface {
	GenerateTOTP(string, string) (TOTPKey, error)
	ValidateTOTP(string, string) bool
}

// Repository 是 IAM 服务需要的持久化端口。
// 它把组织、用户、成员、角色、会话、token、邀请和审计等读写收拢到服务层可替换接口。
type Repository interface {
	WithTx(context.Context, func(context.Context, Repository) error) error
	HasUsersTable(context.Context) (bool, error)
	CreateOrganization(context.Context, *model.Organization) error
	FindOrganizationByID(context.Context, int64) (*model.Organization, error)
	FindOrganizationByCode(context.Context, string) (*model.Organization, error)
	ListOrganizations(context.Context) ([]model.Organization, error)
	SaveOrganization(context.Context, *model.Organization) error
	CreateUser(context.Context, *model.User) error
	CountUsers(context.Context) (int64, error)
	FindUserByID(context.Context, int64) (*model.User, error)
	FindUserByIdentifier(context.Context, string) (*model.User, error)
	SaveUser(context.Context, *model.User) error
	CreateMembership(context.Context, *model.Membership) error
	FindMembership(context.Context, int64, int64) (*model.Membership, error)
	FindMembershipAnyStatus(context.Context, int64, int64) (*model.Membership, error)
	ListMembershipsByUser(context.Context, int64) ([]model.Membership, error)
	ListMembershipsByOrg(context.Context, int64) ([]model.Membership, error)
	SaveMembership(context.Context, *model.Membership) error
	ListUsersByOrg(context.Context, int64) ([]model.User, error)
	CreateRole(context.Context, *model.Role) error
	FindRoleByID(context.Context, int64) (*model.Role, error)
	FindRole(context.Context, int64, string) (*model.Role, error)
	ListRoles(context.Context, int64) ([]model.Role, error)
	ListRolePermissions(context.Context, int64, string) ([]RolePermission, error)
	SaveRole(context.Context, *model.Role) error
	CreatePermission(context.Context, *model.Permission) error
	FindPermission(context.Context, string, string, string) (*model.Permission, error)
	ListPermissions(context.Context) ([]model.Permission, error)
	CreateSession(context.Context, *model.Session) error
	FindSessionByID(context.Context, int64) (*model.Session, error)
	FindSessionByRefreshHash(context.Context, string) (*model.Session, error)
	ListSessionsByOrg(context.Context, int64) ([]model.Session, error)
	ListSessionsByUser(context.Context, int64) ([]model.Session, error)
	SaveSession(context.Context, *model.Session) error
	CreateAPIToken(context.Context, *model.APIToken) error
	FindAPITokenByHash(context.Context, string) (*model.APIToken, error)
	FindAPITokenByID(context.Context, int64) (*model.APIToken, error)
	ListAPITokens(context.Context, int64, APITokenFilter) ([]model.APIToken, int64, error)
	SaveAPIToken(context.Context, *model.APIToken) error
	CreateInvitation(context.Context, *model.Invitation) error
	FindInvitationByID(context.Context, int64) (*model.Invitation, error)
	FindInvitationByTokenHash(context.Context, string) (*model.Invitation, error)
	ListInvitationsByOrg(context.Context, int64) ([]model.Invitation, error)
	SaveInvitation(context.Context, *model.Invitation) error
	CreatePasswordReset(context.Context, *model.PasswordReset) error
	FindPasswordResetByTokenHash(context.Context, string) (*model.PasswordReset, error)
	SavePasswordReset(context.Context, *model.PasswordReset) error
	CreateEmailVerification(context.Context, *model.EmailVerification) error
	FindEmailVerificationByTokenHash(context.Context, string) (*model.EmailVerification, error)
	SaveEmailVerification(context.Context, *model.EmailVerification) error
	DeletePendingSignup(context.Context, int64, int64, int64) error
	CreateMFAFactor(context.Context, *model.MFAFactor) error
	FindActiveMFAFactor(context.Context, int64) (*model.MFAFactor, error)
	SaveMFAFactor(context.Context, *model.MFAFactor) error
	CreateAuditLog(context.Context, *model.AuditLog) error
	ListAuditLogs(context.Context, int64, AuditLogFilter) ([]model.AuditLog, error)
	AddCasbinRule(context.Context, *model.CasbinRule) error
	DeleteCasbinRules(context.Context, string, ...string) error
	ListCasbinRules(context.Context) ([]AuthorizationRule, error)
}

// Service 定义 IAM 模块对 handler 暴露的应用层能力。
// 这些方法覆盖认证生命周期、组织成员管理、角色权限、API token、会话和审计。
type Service interface {
	BootstrapAdmin(context.Context, BootstrapAdminInput) (*Principal, error)
	SetupStatus(context.Context) (SetupStatus, error)
	InitialAdminSetup(context.Context, InitialAdminSetupInput) (TokenPair, error)
	Signup(context.Context, SignupInput) (SignupResult, error)
	ConfirmEmailVerification(context.Context, ConfirmEmailVerificationInput) (TokenPair, error)
	Captcha(context.Context) (CaptchaChallenge, error)
	Login(context.Context, LoginInput) (TokenPair, error)
	Refresh(context.Context, RefreshInput) (TokenPair, error)
	Logout(context.Context, Principal) error
	SwitchOrg(context.Context, Principal, int64, string, string) (TokenPair, error)
	AuthenticateToken(context.Context, string) (Principal, error)
	Authorize(context.Context, Principal, PermissionContext) (bool, error)
	CurrentSession(context.Context, Principal) (SessionSnapshot, error)
	Me(context.Context, Principal) (*model.User, error)
	ListMyOrganizations(context.Context, Principal) ([]model.Organization, error)
	ListOrganizations(context.Context, Principal, OrganizationListFilter) (OrganizationPage, error)
	CreateOrganization(context.Context, Principal, string, string) (*model.Organization, error)
	UpdateOrganization(context.Context, UpdateOrganizationInput) (*model.Organization, error)
	InviteUser(context.Context, InviteUserInput) (NotificationDelivery, error)
	ListInvitations(context.Context, Principal) ([]model.Invitation, error)
	RevokeInvitation(context.Context, Principal, int64, string, string) error
	AcceptInvitation(context.Context, AcceptInvitationInput) (*Principal, error)
	ForgotPassword(context.Context, ForgotPasswordInput) (NotificationDelivery, error)
	ResetPassword(context.Context, ResetPasswordInput) error
	SetupMFA(context.Context, Principal) (string, string, error)
	VerifyMFA(context.Context, Principal, string) error
	ListUsers(context.Context, Principal, UserListFilter) (OrganizationUserPage, error)
	UpdateUser(context.Context, UpdateUserInput) (*OrganizationUser, error)
	CreateAPIToken(context.Context, CreateAPITokenInput) (CreateAPITokenResult, error)
	ListAPITokens(context.Context, Principal, APITokenFilter) (APITokenPage, error)
	RevokeAPIToken(context.Context, RevokeAPITokenInput) error
	ListRoles(context.Context, Principal) ([]model.Role, error)
	CreateRole(context.Context, CreateRoleInput) (*model.Role, error)
	UpdateRole(context.Context, UpdateRoleInput) (*model.Role, error)
	ListPermissions(context.Context, Principal) ([]model.Permission, error)
	ListSessions(context.Context, Principal, SessionListFilter) (SessionPage, error)
	RevokeSession(context.Context, Principal, int64) error
	ListAuditLogs(context.Context, Principal, AuditLogFilter) ([]model.AuditLog, error)
	RecordAudit(context.Context, Principal, string, string, string, string, string, map[string]any) error
	LoadPolicies(context.Context) error
}

// service 是 IAM 模块的应用层实现。
// captchaChallenges 是进程内临时状态，需要通过 captchaMu 保护。
type service struct {
	repo     Repository
	crypto   PasswordCrypto
	tokens   TokenManager
	authz    AuthorizerEnforcer
	ids      IDGenerator
	totp     TOTPProvider
	cache    CacheStore
	cfg      Config
	notifier Notifier

	notificationMu        sync.RWMutex
	notificationDriver    string
	notificationPublicURL string

	registrationMu       sync.RWMutex
	registrationMode     string
	emailVerificationTTL time.Duration

	captchaMu         sync.Mutex
	captchaChallenges map[string]captchaState
}

type Option func(*service)

func WithCacheStore(cache CacheStore) Option {
	return func(s *service) {
		s.cache = cache
	}
}

// New 创建 IAM 服务并补齐安全相关默认值。
// 依赖由组合根注入；未配置 notifier 或 totp 时使用空实现，避免可选能力阻断核心认证流程。
func New(repo Repository, crypto PasswordCrypto, tokens TokenManager, authz AuthorizerEnforcer, ids IDGenerator, totp TOTPProvider, cfg Config, notifier Notifier, options ...Option) Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.LoginMaxFailures <= 0 {
		cfg.LoginMaxFailures = 5
	}
	if cfg.LoginLockDuration <= 0 {
		cfg.LoginLockDuration = 15 * time.Minute
	}
	if cfg.CaptchaTTL <= 0 {
		cfg.CaptchaTTL = 2 * time.Minute
	}
	cfg.RegistrationMode = normalizeRegistrationMode(cfg.RegistrationMode)
	if cfg.RegistrationMode == "" {
		cfg.RegistrationMode = RegistrationModeDisabled
	}
	if cfg.InvitationTTL <= 0 {
		cfg.InvitationTTL = 24 * time.Hour
	}
	if cfg.EmailVerificationTTL <= 0 {
		cfg.EmailVerificationTTL = 24 * time.Hour
	}
	if cfg.PasswordResetTTL <= 0 {
		cfg.PasswordResetTTL = 30 * time.Minute
	}
	if cfg.NotificationDriver == "" {
		cfg.NotificationDriver = "debug"
	}
	if strings.TrimSpace(cfg.DefaultProductCode) == "" {
		cfg.DefaultProductCode = "platform"
	}
	if strings.TrimSpace(cfg.DefaultClientType) == "" {
		cfg.DefaultClientType = "pc_web"
	}
	cfg.DefaultProductCode = normalizeProductCode(cfg.DefaultProductCode)
	cfg.DefaultClientType = normalizeClientType(cfg.DefaultClientType)
	if cfg.UserCacheTTL <= 0 {
		cfg.UserCacheTTL = 2 * time.Minute
	}
	if cfg.OrgCacheTTL <= 0 {
		cfg.OrgCacheTTL = 2 * time.Minute
	}
	if cfg.RoleCacheTTL <= 0 {
		cfg.RoleCacheTTL = 2 * time.Minute
	}
	if cfg.PermissionCacheTTL <= 0 {
		cfg.PermissionCacheTTL = 5 * time.Minute
	}
	if cfg.PasswordPolicy.MinLength <= 0 {
		cfg.PasswordPolicy.MinLength = 8
	}
	if cfg.MFAIssuer == "" {
		cfg.MFAIssuer = "mfa"
	}
	if notifier == nil {
		notifier = NoopNotifier{}
	}
	if totp == nil {
		totp = noopTOTPProvider{}
	}
	svc := &service{
		repo:                  repo,
		crypto:                crypto,
		tokens:                tokens,
		authz:                 authz,
		ids:                   ids,
		totp:                  totp,
		cfg:                   cfg,
		notifier:              notifier,
		notificationDriver:    cfg.NotificationDriver,
		notificationPublicURL: cfg.PublicBaseURL,
		registrationMode:      cfg.RegistrationMode,
		emailVerificationTTL:  cfg.EmailVerificationTTL,
		captchaChallenges:     make(map[string]captchaState),
	}
	for _, option := range options {
		if option != nil {
			option(svc)
		}
	}
	return svc
}

// ReloadNotificationRuntime 更新通知驱动和链接基地址，供配置热加载后立即生效。
func (s *service) ReloadNotificationRuntime(cfg NotificationRuntimeConfig) {
	driver := strings.TrimSpace(cfg.NotificationDriver)
	if driver == "" {
		driver = "debug"
	}
	s.notificationMu.Lock()
	s.notificationDriver = driver
	s.notificationPublicURL = cfg.PublicBaseURL
	s.notificationMu.Unlock()
}

func (s *service) ReloadRegistrationRuntime(cfg RegistrationRuntimeConfig) {
	mode := normalizeRegistrationMode(cfg.RegistrationMode)
	if mode == "" {
		mode = RegistrationModeDisabled
	}
	ttl := cfg.EmailVerificationTTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	s.registrationMu.Lock()
	s.registrationMode = mode
	s.emailVerificationTTL = ttl
	s.registrationMu.Unlock()
}

func (s *service) currentRegistrationMode() string {
	s.registrationMu.RLock()
	mode := s.registrationMode
	s.registrationMu.RUnlock()
	return normalizeRegistrationMode(mode)
}

func (s *service) currentEmailVerificationTTL() time.Duration {
	s.registrationMu.RLock()
	ttl := s.emailVerificationTTL
	s.registrationMu.RUnlock()
	if ttl <= 0 {
		return 24 * time.Hour
	}
	return ttl
}

// noopTOTPProvider 是默认 MFA 空实现。
type noopTOTPProvider struct{}

// GenerateTOTP 返回 ErrInvalidInput，提示调用方需要配置真实 TOTP provider。
func (noopTOTPProvider) GenerateTOTP(string, string) (TOTPKey, error) {
	return TOTPKey{}, ErrInvalidInput
}

// ValidateTOTP 默认拒绝所有验证码。
func (noopTOTPProvider) ValidateTOTP(string, string) bool {
	return false
}

// BootstrapAdmin 幂等引导内置管理员和 owner 角色。
// 该入口允许复用已有组织或用户，适合部署脚本反复执行；成功后会尽力重载授权策略。
func (s *service) BootstrapAdmin(ctx context.Context, input BootstrapAdminInput) (*Principal, error) {
	result, err := s.bootstrapOwner(ctx, ownerBootstrapInput{
		OrgCode:       input.OrgCode,
		OrgName:       input.OrgName,
		Username:      input.Username,
		Email:         input.Email,
		DisplayName:   input.DisplayName,
		Password:      input.Password,
		AuditAction:   "iam.bootstrap_admin",
		AllowExisting: true,
	})
	if err != nil {
		return nil, err
	}
	_ = s.LoadPolicies(ctx)
	return result.Principal, nil
}

// SetupStatus 判断系统是否仍需要首次管理员初始化。
// 用户表缺失或用户数为零都会视为需要 setup，用于新库和未迁移环境的引导页展示。
func (s *service) SetupStatus(ctx context.Context) (SetupStatus, error) {
	if s.repo != nil {
		hasUsersTable, err := s.repo.HasUsersTable(ctx)
		if err != nil {
			return SetupStatus{}, err
		}
		if !hasUsersTable {
			return s.setupStatus(true), nil
		}
	}
	count, err := s.repo.CountUsers(ctx)
	if err != nil {
		if isMissingTableError(err) {
			return s.setupStatus(true), nil
		}
		return SetupStatus{}, err
	}
	return s.setupStatus(count == 0), nil
}

// InitialAdminSetup 创建首次管理员并签发登录 token。
// 该流程要求系统还没有用户，防止初始化接口在已上线环境中被重复调用。
func (s *service) InitialAdminSetup(ctx context.Context, input InitialAdminSetupInput) (TokenPair, error) {
	result, err := s.bootstrapOwner(ctx, ownerBootstrapInput{
		OrgCode:          input.OrgCode,
		OrgName:          input.OrgName,
		Username:         input.Username,
		Email:            input.Email,
		DisplayName:      input.DisplayName,
		Password:         input.Password,
		UserAgent:        input.UserAgent,
		IPAddress:        input.IPAddress,
		ProductCode:      input.ProductCode,
		ClientType:       input.ClientType,
		AuditAction:      "iam.initial_setup",
		RequireEmpty:     true,
		ValidatePassword: true,
		IssueTokens:      true,
	})
	if err != nil {
		return TokenPair{}, err
	}
	_ = s.LoadPolicies(ctx)
	return result.Tokens, nil
}

// ownerBootstrapInput 统一描述管理员引导流程的可变策略。
type ownerBootstrapInput struct {
	OrgCode          string
	OrgName          string
	Username         string
	Email            string
	DisplayName      string
	Password         string
	UserAgent        string
	IPAddress        string
	ProductCode      string
	ClientType       string
	AuditAction      string
	AllowExisting    bool
	RequireEmpty     bool
	ValidatePassword bool
	IssueTokens      bool
}

// ownerBootstrapResult 保存引导后可能返回的 principal 和 token。
type ownerBootstrapResult struct {
	Principal *Principal
	Tokens    TokenPair
}

// bootstrapOwner 在事务中创建或复用组织、用户、成员关系和内置 owner 角色。
// AllowExisting/RequireEmpty/IssueTokens 等开关让部署引导和首次 setup 复用同一套一致性逻辑。
func (s *service) bootstrapOwner(ctx context.Context, input ownerBootstrapInput) (ownerBootstrapResult, error) {
	input.OrgCode = normalizeCode(input.OrgCode)
	input.OrgName = strings.TrimSpace(input.OrgName)
	input.Username = normalizeCode(input.Username)
	input.Email = normalizeEmail(input.Email)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.OrgName == "" && input.AllowExisting {
		input.OrgName = input.OrgCode
	}
	if input.DisplayName == "" {
		input.DisplayName = input.Username
	}
	if input.OrgCode == "" || input.OrgName == "" || input.Username == "" || input.Email == "" || input.Password == "" {
		return ownerBootstrapResult{}, ErrInvalidInput
	}
	if input.ValidatePassword {
		if err := s.validatePassword(input.Password); err != nil {
			return ownerBootstrapResult{}, err
		}
	}
	if input.AuditAction == "" {
		input.AuditAction = "iam.bootstrap_owner"
	}

	var result ownerBootstrapResult
	err := s.repo.WithTx(ctx, func(txCtx context.Context, repo Repository) error {
		if input.RequireEmpty {
			count, err := repo.CountUsers(txCtx)
			if err != nil {
				return err
			}
			if count > 0 {
				return ErrSetupCompleted
			}
		}

		org, err := repo.FindOrganizationByCode(txCtx, input.OrgCode)
		if err != nil {
			if !errors.Is(err, ErrNotFound) {
				return err
			}
			now := s.now()
			org = &model.Organization{ID: s.ids.NextID(), Code: input.OrgCode, Name: input.OrgName, Kind: model.OrgKindPlatform, Status: model.StatusActive, CreatedAt: now, UpdatedAt: now}
			if err := repo.CreateOrganization(txCtx, org); err != nil {
				return err
			}
		} else if !input.AllowExisting {
			return ErrDuplicate
		}
		if org.Kind != model.OrgKindPlatform {
			org.Kind = model.OrgKindPlatform
			if err := repo.SaveOrganization(txCtx, org); err != nil {
				return err
			}
		}

		user, err := repo.FindUserByIdentifier(txCtx, input.Email)
		if err != nil {
			if !errors.Is(err, ErrNotFound) {
				return err
			}
			hash, err := s.crypto.HashPassword(input.Password)
			if err != nil {
				return err
			}
			now := s.now()
			user = &model.User{ID: s.ids.NextID(), Username: input.Username, Email: input.Email, PasswordHash: hash, DisplayName: input.DisplayName, Status: model.StatusActive, CreatedAt: now, UpdatedAt: now}
			if err := repo.CreateUser(txCtx, user); err != nil {
				return err
			}
		} else if !input.AllowExisting {
			return ErrDuplicate
		}

		if err := s.ensureMembership(txCtx, repo, org.ID, user.ID); err != nil {
			return err
		}
		if err := s.ensureBuiltins(txCtx, repo, org.ID, org.Kind); err != nil {
			return err
		}
		if err := s.addUserRole(txCtx, repo, user.ID, org.ID, model.RolePlatformOwner); err != nil {
			return err
		}
		if input.IssueTokens {
			issued, err := s.createSessionAndTokensWithRepo(txCtx, repo, user, org.ID, input.UserAgent, input.IPAddress, input.ProductCode, input.ClientType)
			if err != nil {
				return err
			}
			result.Tokens = issued
		}
		if err := s.audit(txCtx, repo, &org.ID, &user.ID, input.AuditAction, "organization", strconv.FormatInt(org.ID, 10), input.IPAddress, input.UserAgent, map[string]any{"orgCode": org.Code, "email": user.Email}, SessionContext{ProductCode: input.ProductCode, ClientType: input.ClientType}); err != nil {
			return err
		}
		sessionCtx := s.resolveSessionContext(input.ProductCode, input.ClientType)
		result.Principal = &Principal{UserID: user.ID, OrgID: org.ID, ProductCode: sessionCtx.ProductCode, ClientType: sessionCtx.ClientType, Username: user.Username, Email: user.Email}
		return nil
	})
	if err != nil {
		return ownerBootstrapResult{}, err
	}
	return result, nil
}

// isMissingTableError 识别不同数据库驱动返回的“表不存在”错误文本。
// setup 状态接口需要在迁移前也能工作，因此这里做保守的字符串兼容。
func isMissingTableError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "no such table") ||
		strings.Contains(text, "doesn't exist") ||
		strings.Contains(text, "undefined_table") ||
		strings.Contains(text, "unknown table")
}

// Signup 根据注册策略创建公开注册流程。
func (s *service) Signup(ctx context.Context, input SignupInput) (SignupResult, error) {
	switch s.currentRegistrationMode() {
	case RegistrationModeDirect:
		return s.signupDirect(ctx, input)
	case RegistrationModeEmailVerification:
		return s.signupWithEmailVerification(ctx, input)
	case RegistrationModeDisabled, RegistrationModeInviteOnly:
		return SignupResult{}, ErrSignupDisabled
	default:
		return SignupResult{}, ErrSignupDisabled
	}
}

func (s *service) signupDirect(ctx context.Context, input SignupInput) (SignupResult, error) {
	normalized, err := s.normalizeSignupInput(input)
	if err != nil {
		return SignupResult{}, err
	}
	input = normalized

	var pair TokenPair
	err = s.repo.WithTx(ctx, func(txCtx context.Context, repo Repository) error {
		if err := s.ensureSignupUnique(txCtx, repo, input); err != nil {
			return err
		}
		hash, err := s.crypto.HashPassword(input.Password)
		if err != nil {
			return err
		}
		now := s.now()
		org := &model.Organization{ID: s.ids.NextID(), Code: input.OrgCode, Name: input.OrgName, Kind: model.OrgKindTenant, Status: model.StatusActive, CreatedAt: now, UpdatedAt: now}
		if err := repo.CreateOrganization(txCtx, org); err != nil {
			return err
		}
		user := &model.User{ID: s.ids.NextID(), Username: input.Username, Email: input.Email, PasswordHash: hash, DisplayName: input.DisplayName, Status: model.StatusActive, CreatedAt: now, UpdatedAt: now}
		if err := repo.CreateUser(txCtx, user); err != nil {
			return err
		}
		if err := s.ensureMembership(txCtx, repo, org.ID, user.ID); err != nil {
			return err
		}
		if err := s.ensureBuiltins(txCtx, repo, org.ID, org.Kind); err != nil {
			return err
		}
		if err := s.addUserRole(txCtx, repo, user.ID, org.ID, model.RoleOwner); err != nil {
			return err
		}
		issued, err := s.createSessionAndTokensWithRepo(txCtx, repo, user, org.ID, input.UserAgent, input.IPAddress, input.ProductCode, input.ClientType)
		if err != nil {
			return err
		}
		pair = issued
		return s.audit(txCtx, repo, &org.ID, &user.ID, "auth.signup", "organization", strconv.FormatInt(org.ID, 10), input.IPAddress, input.UserAgent, map[string]any{"orgCode": org.Code, "email": user.Email}, SessionContext{ProductCode: input.ProductCode, ClientType: input.ClientType})
	})
	if err != nil {
		return SignupResult{}, err
	}
	_ = s.LoadPolicies(ctx)
	snapshot := pair.SessionSnapshot()
	return SignupResult{Status: SignupStatusAuthenticated, Session: &snapshot, Tokens: pair}, nil
}

func (s *service) signupWithEmailVerification(ctx context.Context, input SignupInput) (SignupResult, error) {
	normalized, err := s.normalizeSignupInput(input)
	if err != nil {
		return SignupResult{}, err
	}
	input = normalized

	var rawToken string
	var orgID, userID, verificationID int64
	err = s.repo.WithTx(ctx, func(txCtx context.Context, repo Repository) error {
		if err := s.ensureSignupUnique(txCtx, repo, input); err != nil {
			return err
		}
		hash, err := s.crypto.HashPassword(input.Password)
		if err != nil {
			return err
		}
		rawToken = s.ids.NextIDString()
		tokenHash := s.tokens.HashRefreshToken(rawToken)
		now := s.now()
		org := &model.Organization{ID: s.ids.NextID(), Code: input.OrgCode, Name: input.OrgName, Kind: model.OrgKindTenant, Status: model.StatusPending, CreatedAt: now, UpdatedAt: now}
		if err := repo.CreateOrganization(txCtx, org); err != nil {
			return err
		}
		user := &model.User{ID: s.ids.NextID(), Username: input.Username, Email: input.Email, PasswordHash: hash, DisplayName: input.DisplayName, Status: model.StatusPending, CreatedAt: now, UpdatedAt: now}
		if err := repo.CreateUser(txCtx, user); err != nil {
			return err
		}
		membership := &model.Membership{ID: s.ids.NextID(), OrgID: org.ID, UserID: user.ID, Status: model.StatusPending, CreatedAt: now, UpdatedAt: now}
		if err := repo.CreateMembership(txCtx, membership); err != nil {
			return err
		}
		verification := &model.EmailVerification{ID: s.ids.NextID(), UserID: user.ID, OrgID: org.ID, TokenHash: tokenHash, Status: model.StatusPending, ExpiresAt: now.Add(s.currentEmailVerificationTTL()), CreatedAt: now, UpdatedAt: now}
		if err := repo.CreateEmailVerification(txCtx, verification); err != nil {
			return err
		}
		orgID = org.ID
		userID = user.ID
		verificationID = verification.ID
		return nil
	})
	if err != nil {
		return SignupResult{}, err
	}

	adminPath := "signup/verify/" + url.PathEscape(rawToken)
	if err := s.notifier.SendEmailVerification(ctx, EmailVerificationNotice{Email: input.Email, Token: rawToken, URL: s.notificationURL(adminPath)}); err != nil {
		if cleanupErr := s.repo.DeletePendingSignup(ctx, orgID, userID, verificationID); cleanupErr != nil {
			return SignupResult{}, fmt.Errorf("%w: %v; cleanup pending signup: %v", ErrNotificationDelivery, err, cleanupErr)
		}
		return SignupResult{}, fmt.Errorf("%w: %v", ErrNotificationDelivery, err)
	}

	result := SignupResult{Status: SignupStatusVerificationPending}
	if delivery := s.debugDelivery(rawToken, adminPath); delivery.Debug {
		result.Delivery = &delivery
	}
	return result, nil
}

func (s *service) ConfirmEmailVerification(ctx context.Context, input ConfirmEmailVerificationInput) (TokenPair, error) {
	rawToken := strings.TrimSpace(input.Token)
	if rawToken == "" {
		return TokenPair{}, ErrInvalidToken
	}
	tokenHash := s.tokens.HashRefreshToken(rawToken)
	var pair TokenPair
	err := s.repo.WithTx(ctx, func(txCtx context.Context, repo Repository) error {
		verification, err := repo.FindEmailVerificationByTokenHash(txCtx, tokenHash)
		if err != nil {
			return ErrInvalidToken
		}
		now := s.now()
		if verification.Status != model.StatusPending {
			return ErrInvalidToken
		}
		if verification.ExpiresAt.Before(now) {
			verification.Status = model.StatusExpired
			verification.UpdatedAt = now
			_ = repo.SaveEmailVerification(txCtx, verification)
			return ErrInvalidToken
		}
		user, err := repo.FindUserByID(txCtx, verification.UserID)
		if err != nil {
			return ErrInvalidToken
		}
		org, err := repo.FindOrganizationByID(txCtx, verification.OrgID)
		if err != nil {
			return ErrInvalidToken
		}
		membership, err := repo.FindMembershipAnyStatus(txCtx, org.ID, user.ID)
		if err != nil {
			return ErrInvalidToken
		}
		if user.Status != model.StatusPending || org.Status != model.StatusPending || membership.Status != model.StatusPending {
			return ErrInvalidToken
		}
		user.Status = model.StatusActive
		user.UpdatedAt = now
		org.Status = model.StatusActive
		org.UpdatedAt = now
		membership.Status = model.StatusActive
		membership.UpdatedAt = now
		verification.Status = model.StatusUsed
		verification.VerifiedAt = &now
		verification.UpdatedAt = now
		if err := repo.SaveOrganization(txCtx, org); err != nil {
			return err
		}
		if err := repo.SaveUser(txCtx, user); err != nil {
			return err
		}
		if err := repo.SaveMembership(txCtx, membership); err != nil {
			return err
		}
		if err := repo.SaveEmailVerification(txCtx, verification); err != nil {
			return err
		}
		if err := s.ensureBuiltins(txCtx, repo, org.ID, org.Kind); err != nil {
			return err
		}
		if err := s.addUserRole(txCtx, repo, user.ID, org.ID, model.RoleOwner); err != nil {
			return err
		}
		issued, err := s.createSessionAndTokensWithRepo(txCtx, repo, user, org.ID, input.UserAgent, input.IPAddress, input.ProductCode, input.ClientType)
		if err != nil {
			return err
		}
		pair = issued
		return s.audit(txCtx, repo, &org.ID, &user.ID, "auth.email_verification.confirm", "email_verification", strconv.FormatInt(verification.ID, 10), input.IPAddress, input.UserAgent, map[string]any{"orgCode": org.Code, "email": user.Email}, SessionContext{ProductCode: input.ProductCode, ClientType: input.ClientType})
	})
	if err != nil {
		return TokenPair{}, err
	}
	_ = s.LoadPolicies(ctx)
	s.bumpUserOrganizationsCache(ctx, pair.ProductCode, pair.UserID)
	s.bumpOrganizationCaches(ctx, pair.ProductCode, pair.OrgID)
	return pair, nil
}

func (s *service) normalizeSignupInput(input SignupInput) (SignupInput, error) {
	input.OrgCode = normalizeCode(input.OrgCode)
	input.OrgName = strings.TrimSpace(input.OrgName)
	input.Username = normalizeCode(input.Username)
	input.Email = normalizeEmail(input.Email)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.OrgCode == "" || input.OrgName == "" || input.Username == "" || input.Email == "" || input.Password == "" {
		return SignupInput{}, ErrInvalidInput
	}
	if input.DisplayName == "" {
		input.DisplayName = input.Username
	}
	if err := s.validatePassword(input.Password); err != nil {
		return SignupInput{}, err
	}
	return input, nil
}

func (s *service) ensureSignupUnique(ctx context.Context, repo Repository, input SignupInput) error {
	if _, err := repo.FindOrganizationByCode(ctx, input.OrgCode); err == nil {
		return ErrDuplicate
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	if _, err := repo.FindUserByIdentifier(ctx, input.Username); err == nil {
		return ErrDuplicate
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	if input.Email != input.Username {
		if _, err := repo.FindUserByIdentifier(ctx, input.Email); err == nil {
			return ErrDuplicate
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return nil
}

// Login 校验账号、密码、验证码和 MFA 后创建会话。
// 密码错误会记录失败次数并可能锁定账号；成功登录会清空失败计数并写入审计记录。
func (s *service) Login(ctx context.Context, input LoginInput) (TokenPair, error) {
	identifier := strings.TrimSpace(strings.ToLower(input.Identifier))
	if identifier == "" || input.Password == "" {
		return TokenPair{}, ErrUnauthorized
	}
	if err := s.validateLoginCaptcha(input.CaptchaID, input.CaptchaCode); err != nil {
		return TokenPair{}, err
	}
	user, err := s.repo.FindUserByIdentifier(ctx, identifier)
	if err != nil {
		return TokenPair{}, ErrUnauthorized
	}
	if err := s.ensureUserCanLogin(user); err != nil {
		return TokenPair{}, err
	}
	if err := s.crypto.VerifyPassword(user.PasswordHash, input.Password); err != nil {
		_ = s.recordFailedLogin(ctx, user)
		return TokenPair{}, ErrUnauthorized
	}
	if user.MFAEnabled {
		if input.MFACode == "" {
			return TokenPair{}, ErrMFARequired
		}
		if err := s.verifyUserMFA(ctx, user.ID, input.MFACode); err != nil {
			return TokenPair{}, ErrUnauthorized
		}
	}
	org, err := s.loginOrg(ctx, user.ID, input.OrgCode)
	if err != nil {
		return TokenPair{}, err
	}
	pair, err := s.createSessionAndTokens(ctx, user, org.ID, input.UserAgent, input.IPAddress, input.ProductCode, input.ClientType)
	if err != nil {
		return TokenPair{}, err
	}
	now := s.now()
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	user.LastLoginAt = &now
	_ = s.repo.SaveUser(ctx, user)
	_ = s.audit(ctx, s.repo, &org.ID, &user.ID, "auth.login", "session", "", input.IPAddress, input.UserAgent, nil, SessionContext{ProductCode: input.ProductCode, ClientType: input.ClientType})
	return pair, nil
}

// Refresh 使用 refresh token 换发新的 token pair。
// 成功后会轮换 refresh token hash 并更新会话最后使用信息，降低旧 refresh token 泄漏风险。
func (s *service) Refresh(ctx context.Context, input RefreshInput) (TokenPair, error) {
	hash := s.tokens.HashRefreshToken(strings.TrimSpace(input.RefreshToken))
	session, err := s.repo.FindSessionByRefreshHash(ctx, hash)
	if err != nil {
		return TokenPair{}, ErrInvalidToken
	}
	if err := s.ensureSessionActive(session); err != nil {
		return TokenPair{}, err
	}
	user, err := s.repo.FindUserByID(ctx, session.UserID)
	if err != nil {
		return TokenPair{}, ErrInvalidToken
	}
	if err := s.ensureUserCanLogin(user); err != nil {
		return TokenPair{}, err
	}
	if _, err := s.repo.FindMembership(ctx, session.OrgID, session.UserID); err != nil {
		return TokenPair{}, ErrForbidden
	}
	sessionCtx := s.resolveSessionContext(session.ProductCode, session.ClientType)
	pair, err := s.tokens.IssuePair(ctx, TokenSubject{UserID: user.ID, OrgID: session.OrgID, SessionID: session.ID, ProductCode: sessionCtx.ProductCode})
	if err != nil {
		return TokenPair{}, err
	}
	now := s.now()
	session.RefreshTokenHash = pair.RefreshTokenHash
	session.ExpiresAt = pair.RefreshExpiresAt
	session.LastUsedAt = &now
	session.UserAgent = input.UserAgent
	session.IPAddress = input.IPAddress
	session.ProductCode = sessionCtx.ProductCode
	session.ClientType = sessionCtx.ClientType
	if err := s.repo.SaveSession(ctx, session); err != nil {
		return TokenPair{}, err
	}
	return tokenPair(pair, session), nil
}

// Logout 撤销当前会话。
func (s *service) Logout(ctx context.Context, principal Principal) error {
	session, err := s.repo.FindSessionByID(ctx, principal.SessionID)
	if err != nil {
		return ErrInvalidToken
	}
	now := s.now()
	session.RevokedAt = &now
	return s.repo.SaveSession(ctx, session)
}

// SwitchOrg 为同一用户切换组织上下文并创建新会话。
// 目标组织必须已有成员关系，避免用户伪造 OrgID 获取跨租户 token。
func (s *service) SwitchOrg(ctx context.Context, principal Principal, orgID int64, userAgent, ip string) (TokenPair, error) {
	if _, err := s.repo.FindMembership(ctx, orgID, principal.UserID); err != nil {
		return TokenPair{}, ErrForbidden
	}
	user, err := s.repo.FindUserByID(ctx, principal.UserID)
	if err != nil {
		return TokenPair{}, ErrInvalidToken
	}
	return s.createSessionAndTokens(ctx, user, orgID, userAgent, ip, principal.ProductCode, principal.ClientType)
}

// AuthenticateToken 解析 access token 并构造请求主体。
// access token 解析失败时会尝试 API token 认证，支持浏览器会话和机器访问共用入口。
func (s *service) AuthenticateToken(ctx context.Context, raw string) (Principal, error) {
	claims, err := s.tokens.Parse(ctx, raw, TokenTypeAccess)
	if err != nil {
		return s.authenticateAPIToken(ctx, raw)
	}
	session, err := s.repo.FindSessionByID(ctx, claims.SessionID)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}
	if err := s.ensureSessionActive(session); err != nil {
		return Principal{}, err
	}
	user, err := s.repo.FindUserByID(ctx, claims.UserID)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}
	if err := s.ensureUserCanLogin(user); err != nil {
		return Principal{}, err
	}
	if _, err := s.repo.FindMembership(ctx, claims.OrgID, claims.UserID); err != nil {
		return Principal{}, ErrForbidden
	}
	sessionCtx := s.resolveSessionContext(claims.ProductCode, session.ClientType)
	if session.ProductCode != sessionCtx.ProductCode || session.ClientType != sessionCtx.ClientType {
		return Principal{}, ErrInvalidToken
	}
	return Principal{UserID: user.ID, OrgID: claims.OrgID, SessionID: session.ID, ProductCode: sessionCtx.ProductCode, ClientType: sessionCtx.ClientType, Username: user.Username, Email: user.Email}, nil
}

// Authorize 校验主体是否允许访问指定对象动作。
// API token 会携带 RoleCode 并走角色内权限匹配，普通会话走授权引擎。
func (s *service) Authorize(ctx context.Context, p Principal, permission PermissionContext) (bool, error) {
	permission = s.normalizePermissionContext(permission)
	if permission.Scope == "" || permission.Object == "" || permission.Action == "" {
		return false, nil
	}
	if p.RoleCode != "" {
		return s.authorizeRole(ctx, p, permission)
	}
	return s.authz.Enforce(ctx, userSubject(p.UserID), strconv.FormatInt(p.OrgID, 10), permission.ProductCode, permission.Scope, permission.Object, permission.Action)
}

func (s *service) CurrentSession(ctx context.Context, p Principal) (SessionSnapshot, error) {
	session, err := s.repo.FindSessionByID(ctx, p.SessionID)
	if err != nil {
		return SessionSnapshot{}, ErrInvalidToken
	}
	if err := s.ensureSessionActive(session); err != nil {
		return SessionSnapshot{}, err
	}
	sessionCtx := s.resolveSessionContext(session.ProductCode, session.ClientType)
	if session.UserID != p.UserID || session.OrgID != p.OrgID || sessionCtx.ProductCode != p.ProductCode || sessionCtx.ClientType != p.ClientType {
		return SessionSnapshot{}, ErrInvalidToken
	}
	return SessionSnapshot{
		UserID:           session.UserID,
		OrgID:            session.OrgID,
		SessionID:        session.ID,
		ProductCode:      sessionCtx.ProductCode,
		ClientType:       sessionCtx.ClientType,
		RefreshExpiresAt: session.ExpiresAt,
	}, nil
}

// Me 返回当前主体对应的用户资料。
func (s *service) Me(ctx context.Context, p Principal) (*model.User, error) {
	return s.repo.FindUserByID(ctx, p.UserID)
}

// ListMyOrganizations 返回当前用户已加入的组织。
// 单个组织读取失败会被跳过，避免历史脏成员关系拖垮整个列表。
func (s *service) ListMyOrganizations(ctx context.Context, p Principal) ([]model.Organization, error) {
	cacheKey := s.userOrganizationsCacheKey(ctx, p)
	var cached []model.Organization
	if s.cacheGet(ctx, cacheKey, s.cfg.OrgCacheTTL, &cached) {
		return cached, nil
	}
	memberships, err := s.repo.ListMembershipsByUser(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	orgs := make([]model.Organization, 0, len(memberships))
	for _, membership := range memberships {
		org, err := s.repo.FindOrganizationByID(ctx, membership.OrgID)
		if err == nil {
			orgs = append(orgs, *org)
		}
	}
	s.cacheSet(ctx, cacheKey, s.cfg.OrgCacheTTL, orgs)
	return orgs, nil
}

// ListOrganizations 返回组织列表，并在服务层执行过滤、排序和分页。
// 当前仓储端口返回全量组织，因此这里统一应用管理端查询语义。
func (s *service) ListOrganizations(ctx context.Context, _ Principal, filter OrganizationListFilter) (OrganizationPage, error) {
	orgs, err := s.repo.ListOrganizations(ctx)
	if err != nil {
		return OrganizationPage{}, err
	}
	out := make([]model.Organization, 0, len(orgs))
	for _, org := range orgs {
		if organizationMatches(org, filter) {
			out = append(out, org)
		}
	}
	sortOrganizations(out, filter)
	page, pageSize := normalizeListPage(filter.Page, filter.PageSize)
	total := int64(len(out))
	start := (page - 1) * pageSize
	if start > len(out) {
		start = len(out)
	}
	end := start + pageSize
	if end > len(out) {
		end = len(out)
	}
	return OrganizationPage{
		Items:         out[start:end],
		Page:          page,
		PageSize:      pageSize,
		Total:         total,
		StorageStatus: "persisted",
	}, nil
}

// CreateOrganization 创建新组织并把创建者设为 owner。
// 组织、成员关系、内置角色权限和审计记录在同一事务中写入，成功后尽力重载授权策略。
func (s *service) CreateOrganization(ctx context.Context, p Principal, code, name string) (*model.Organization, error) {
	code = normalizeCode(code)
	name = strings.TrimSpace(name)
	if code == "" || name == "" {
		return nil, ErrInvalidInput
	}
	var org *model.Organization
	err := s.repo.WithTx(ctx, func(txCtx context.Context, repo Repository) error {
		if _, err := repo.FindOrganizationByCode(txCtx, code); err == nil {
			return ErrDuplicate
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		now := s.now()
		org = &model.Organization{ID: s.ids.NextID(), Code: code, Name: name, Kind: model.OrgKindTenant, Status: model.StatusActive, CreatedAt: now, UpdatedAt: now}
		if err := repo.CreateOrganization(txCtx, org); err != nil {
			return err
		}
		if err := s.ensureMembership(txCtx, repo, org.ID, p.UserID); err != nil {
			return err
		}
		if err := s.ensureBuiltins(txCtx, repo, org.ID, org.Kind); err != nil {
			return err
		}
		if err := s.addUserRole(txCtx, repo, p.UserID, org.ID, model.RoleOwner); err != nil {
			return err
		}
		return s.audit(txCtx, repo, &org.ID, &p.UserID, "org.create", "organization", strconv.FormatInt(org.ID, 10), "", "", nil, SessionContext{ProductCode: p.ProductCode, ClientType: p.ClientType})
	})
	if err != nil {
		return nil, err
	}
	_ = s.LoadPolicies(ctx)
	s.bumpUserOrganizationsCache(ctx, p.ProductCode, p.UserID)
	s.bumpOrganizationCaches(ctx, p.ProductCode, org.ID)
	return org, nil
}

// UpdateOrganization 更新当前组织的名称。
// 只允许修改 Principal 所在组织，避免跨租户更新。
func (s *service) UpdateOrganization(ctx context.Context, input UpdateOrganizationInput) (*model.Organization, error) {
	if input.OrgID != input.Principal.OrgID {
		return nil, ErrForbidden
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	org, err := s.repo.FindOrganizationByID(ctx, input.OrgID)
	if err != nil {
		return nil, ErrNotFound
	}
	org.Name = name
	if err := s.repo.SaveOrganization(ctx, org); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, s.repo, &input.Principal.OrgID, &input.Principal.UserID, "org.update", "organization", strconv.FormatInt(org.ID, 10), input.IPAddress, input.UserAgent, map[string]any{"name": name}, SessionContext{ProductCode: input.Principal.ProductCode, ClientType: input.Principal.ClientType})
	s.bumpOrganizationCaches(ctx, input.Principal.ProductCode, input.Principal.OrgID)
	return org, nil
}

// InviteUser 创建加入组织的一次性邀请。
// token 明文只用于通知和 debug 返回，仓储中保存 hash；通知失败时会撤销邀请并返回错误。
func (s *service) InviteUser(ctx context.Context, input InviteUserInput) (NotificationDelivery, error) {
	email := normalizeEmail(input.Email)
	roleCode := normalizeCode(input.RoleCode)
	if email == "" || roleCode == "" {
		return NotificationDelivery{}, ErrInvalidInput
	}
	if roleCode == model.RolePlatformOwner {
		return NotificationDelivery{}, ErrForbidden
	}
	if _, err := s.repo.FindRole(ctx, input.Principal.OrgID, roleCode); err != nil {
		return NotificationDelivery{}, ErrNotFound
	}
	raw, hash, err := s.oneTimeToken()
	if err != nil {
		return NotificationDelivery{}, err
	}
	now := s.now()
	invitation := &model.Invitation{
		ID: s.ids.NextID(), OrgID: input.Principal.OrgID, Email: email, RoleCode: roleCode, TokenHash: hash,
		Status: model.StatusPending, InvitedBy: input.Principal.UserID, ExpiresAt: now.Add(s.cfg.InvitationTTL), CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.CreateInvitation(ctx, invitation); err != nil {
		return NotificationDelivery{}, err
	}
	adminPath := "invitations/" + url.PathEscape(raw)
	if err := s.notifier.SendInvitation(ctx, InvitationNotice{Email: email, Token: raw, URL: s.notificationURL(adminPath)}); err != nil {
		if revokeErr := s.revokeInvitationAfterDeliveryFailure(ctx, invitation); revokeErr != nil {
			return NotificationDelivery{}, fmt.Errorf("%w: %v; revoke invitation: %v", ErrNotificationDelivery, err, revokeErr)
		}
		return NotificationDelivery{}, fmt.Errorf("%w: %v", ErrNotificationDelivery, err)
	}
	_ = s.audit(ctx, s.repo, &input.Principal.OrgID, &input.Principal.UserID, "user.invite", "invitation", strconv.FormatInt(invitation.ID, 10), input.IPAddress, input.UserAgent, map[string]any{"email": email}, SessionContext{ProductCode: input.Principal.ProductCode, ClientType: input.Principal.ClientType})
	return s.debugDelivery(raw, adminPath), nil
}

// ListInvitations 返回当前组织的邀请记录。
func (s *service) ListInvitations(ctx context.Context, p Principal) ([]model.Invitation, error) {
	return s.repo.ListInvitationsByOrg(ctx, p.OrgID)
}

// RevokeInvitation 撤销当前组织内仍处于 pending 的邀请。
func (s *service) RevokeInvitation(ctx context.Context, p Principal, invitationID int64, userAgent, ip string) error {
	invitation, err := s.repo.FindInvitationByID(ctx, invitationID)
	if err != nil {
		return ErrNotFound
	}
	if invitation.OrgID != p.OrgID {
		return ErrForbidden
	}
	if invitation.Status != model.StatusPending {
		return ErrInvitationClosed
	}
	invitation.Status = model.StatusRevoked
	if err := s.repo.SaveInvitation(ctx, invitation); err != nil {
		return err
	}
	return s.audit(ctx, s.repo, &p.OrgID, &p.UserID, "invite.revoke", "invitation", strconv.FormatInt(invitation.ID, 10), ip, userAgent, map[string]any{"email": invitation.Email}, SessionContext{ProductCode: p.ProductCode, ClientType: p.ClientType})
}

// AcceptInvitation 使用一次性 token 加入组织。
// 接受过程会创建缺失用户、补齐成员关系、绑定邀请角色并标记邀请已使用。
func (s *service) AcceptInvitation(ctx context.Context, input AcceptInvitationInput) (*Principal, error) {
	hash := s.tokens.HashRefreshToken(strings.TrimSpace(input.Token))
	invitation, err := s.repo.FindInvitationByTokenHash(ctx, hash)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if invitation.Status != model.StatusPending || invitation.ExpiresAt.Before(s.now()) {
		return nil, ErrInvitationClosed
	}
	if normalizeCode(invitation.RoleCode) == model.RolePlatformOwner {
		return nil, ErrForbidden
	}
	username := normalizeCode(input.Username)
	email := normalizeEmail(invitation.Email)
	displayName := strings.TrimSpace(input.DisplayName)
	if username == "" || input.Password == "" {
		return nil, ErrInvalidInput
	}
	if err := s.validatePassword(input.Password); err != nil {
		return nil, err
	}
	if displayName == "" {
		displayName = username
	}
	var principal *Principal
	err = s.repo.WithTx(ctx, func(txCtx context.Context, repo Repository) error {
		user, err := repo.FindUserByIdentifier(txCtx, email)
		if err != nil {
			if !errors.Is(err, ErrNotFound) {
				return err
			}
			hash, err := s.crypto.HashPassword(input.Password)
			if err != nil {
				return err
			}
			now := s.now()
			user = &model.User{ID: s.ids.NextID(), Username: username, Email: email, PasswordHash: hash, DisplayName: displayName, Status: model.StatusActive, CreatedAt: now, UpdatedAt: now}
			if err := repo.CreateUser(txCtx, user); err != nil {
				return err
			}
		}
		if err := s.ensureMembership(txCtx, repo, invitation.OrgID, user.ID); err != nil {
			return err
		}
		if err := s.addUserRole(txCtx, repo, user.ID, invitation.OrgID, invitation.RoleCode); err != nil {
			return err
		}
		now := s.now()
		invitation.Status = model.StatusUsed
		invitation.AcceptedBy = &user.ID
		invitation.UpdatedAt = now
		if err := repo.SaveInvitation(txCtx, invitation); err != nil {
			return err
		}
		sessionCtx := s.resolveSessionContext("", "")
		principal = &Principal{UserID: user.ID, OrgID: invitation.OrgID, ProductCode: sessionCtx.ProductCode, ClientType: sessionCtx.ClientType, Username: user.Username, Email: user.Email}
		return s.audit(txCtx, repo, &invitation.OrgID, &user.ID, "invitation.accept", "invitation", strconv.FormatInt(invitation.ID, 10), input.IPAddress, input.UserAgent, nil, sessionCtx)
	})
	if err != nil {
		return nil, err
	}
	_ = s.LoadPolicies(ctx)
	if principal != nil {
		s.bumpUserOrganizationsCache(ctx, principal.ProductCode, principal.UserID)
		s.bumpOrganizationCaches(ctx, principal.ProductCode, principal.OrgID)
	}
	return principal, nil
}

// ForgotPassword 创建密码重置 token。
// 邮箱不存在时返回空成功，避免泄露账号是否存在。
func (s *service) ForgotPassword(ctx context.Context, input ForgotPasswordInput) (NotificationDelivery, error) {
	user, err := s.repo.FindUserByIdentifier(ctx, normalizeEmail(input.Email))
	if err != nil {
		return NotificationDelivery{}, nil
	}
	raw, hash, err := s.oneTimeToken()
	if err != nil {
		return NotificationDelivery{}, err
	}
	now := s.now()
	reset := &model.PasswordReset{ID: s.ids.NextID(), UserID: user.ID, TokenHash: hash, Status: model.StatusPending, ExpiresAt: now.Add(s.cfg.PasswordResetTTL), CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreatePasswordReset(ctx, reset); err != nil {
		return NotificationDelivery{}, err
	}
	adminPath := "password/reset?token=" + url.QueryEscape(raw)
	if err := s.notifier.SendPasswordReset(ctx, PasswordResetNotice{Email: user.Email, Token: raw, URL: s.notificationURL(adminPath)}); err != nil {
		if revokeErr := s.revokePasswordResetAfterDeliveryFailure(ctx, reset); revokeErr != nil {
			return NotificationDelivery{}, fmt.Errorf("%w: %v; revoke password reset: %v", ErrNotificationDelivery, err, revokeErr)
		}
		return NotificationDelivery{}, fmt.Errorf("%w: %v", ErrNotificationDelivery, err)
	}
	_ = s.audit(ctx, s.repo, nil, &user.ID, "password.forgot", "password_reset", strconv.FormatInt(reset.ID, 10), input.IPAddress, input.UserAgent, nil)
	return s.debugDelivery(raw, adminPath), nil
}

// ResetPassword 使用一次性 token 更新密码。
// 成功后会清空登录失败锁定状态，并撤销该用户的所有现有会话。
func (s *service) ResetPassword(ctx context.Context, input ResetPasswordInput) error {
	reset, err := s.repo.FindPasswordResetByTokenHash(ctx, s.tokens.HashRefreshToken(strings.TrimSpace(input.Token)))
	if err != nil {
		return ErrInvalidToken
	}
	if reset.Status != model.StatusPending || reset.ExpiresAt.Before(s.now()) {
		return ErrInvalidToken
	}
	if err := s.validatePassword(input.NewPassword); err != nil {
		return err
	}
	user, err := s.repo.FindUserByID(ctx, reset.UserID)
	if err != nil {
		return ErrInvalidToken
	}
	hash, err := s.crypto.HashPassword(input.NewPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	if err := s.repo.SaveUser(ctx, user); err != nil {
		return err
	}
	now := s.now()
	reset.Status = model.StatusUsed
	reset.UsedAt = &now
	if err := s.repo.SavePasswordReset(ctx, reset); err != nil {
		return err
	}
	sessions, err := s.repo.ListSessionsByUser(ctx, user.ID)
	if err != nil {
		return err
	}
	for i := range sessions {
		if sessions[i].RevokedAt != nil {
			continue
		}
		sessions[i].RevokedAt = &now
		if err := s.repo.SaveSession(ctx, &sessions[i]); err != nil {
			return err
		}
	}
	return s.audit(ctx, s.repo, nil, &user.ID, "password.reset", "password_reset", strconv.FormatInt(reset.ID, 10), input.IPAddress, input.UserAgent, nil)
}

// SetupMFA 生成或刷新当前用户的 TOTP 因子。
// 返回明文 secret 和绑定 URL 仅供本次展示，持久化时会加密保存。
func (s *service) SetupMFA(ctx context.Context, p Principal) (string, string, error) {
	user, err := s.repo.FindUserByID(ctx, p.UserID)
	if err != nil {
		return "", "", ErrInvalidToken
	}
	key, err := s.totp.GenerateTOTP(s.cfg.MFAIssuer, user.Email)
	if err != nil {
		return "", "", err
	}
	encrypted, err := s.encryptSecret(key.Secret)
	if err != nil {
		return "", "", err
	}
	now := s.now()
	factor, err := s.repo.FindActiveMFAFactor(ctx, user.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return "", "", err
	}
	if err == nil {
		factor.Secret = encrypted
		factor.ConfirmedAt = nil
		factor.UpdatedAt = now
		if err := s.repo.SaveMFAFactor(ctx, factor); err != nil {
			return "", "", err
		}
		if err := s.audit(ctx, s.repo, &p.OrgID, &p.UserID, "mfa.setup", "mfa_factor", strconv.FormatInt(factor.ID, 10), "", "", nil, SessionContext{ProductCode: p.ProductCode, ClientType: p.ClientType}); err != nil {
			return "", "", err
		}
		return key.Secret, key.URL, nil
	}
	factor = &model.MFAFactor{ID: s.ids.NextID(), UserID: user.ID, Type: "totp", Secret: encrypted, Status: model.StatusActive, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreateMFAFactor(ctx, factor); err != nil {
		return "", "", err
	}
	if err := s.audit(ctx, s.repo, &p.OrgID, &p.UserID, "mfa.setup", "mfa_factor", strconv.FormatInt(factor.ID, 10), "", "", nil, SessionContext{ProductCode: p.ProductCode, ClientType: p.ClientType}); err != nil {
		return "", "", err
	}
	return key.Secret, key.URL, nil
}

// VerifyMFA 校验 TOTP 并启用用户 MFA。
// 成功后会标记因子已确认，并写入审计日志。
func (s *service) VerifyMFA(ctx context.Context, p Principal, code string) error {
	if err := s.verifyUserMFA(ctx, p.UserID, code); err != nil {
		return err
	}
	user, err := s.repo.FindUserByID(ctx, p.UserID)
	if err != nil {
		return err
	}
	now := s.now()
	user.MFAEnabled = true
	if err := s.repo.SaveUser(ctx, user); err != nil {
		return err
	}
	factor, err := s.repo.FindActiveMFAFactor(ctx, p.UserID)
	if err == nil {
		factor.ConfirmedAt = &now
		_ = s.repo.SaveMFAFactor(ctx, factor)
	}
	return s.audit(ctx, s.repo, &p.OrgID, &p.UserID, "mfa.verify", "mfa_factor", "", "", "", nil, SessionContext{ProductCode: p.ProductCode, ClientType: p.ClientType})
}

// ListUsers 返回当前组织成员列表。
// 成员、用户和角色信息分步拼装；无法读取的用户会被跳过，避免脏成员关系影响管理页。
func (s *service) ListUsers(ctx context.Context, p Principal, filter UserListFilter) (OrganizationUserPage, error) {
	cacheKey := s.organizationUsersCacheKey(ctx, p, filter)
	var cached OrganizationUserPage
	if s.cacheGet(ctx, cacheKey, s.cfg.UserCacheTTL, &cached) {
		return cached, nil
	}
	memberships, err := s.repo.ListMembershipsByOrg(ctx, p.OrgID)
	if err != nil {
		return OrganizationUserPage{}, err
	}
	out := make([]OrganizationUser, 0, len(memberships))
	for _, membership := range memberships {
		user, err := s.repo.FindUserByID(ctx, membership.UserID)
		if err != nil {
			continue
		}
		roles, _ := s.authz.GetRolesForUser(ctx, userSubject(user.ID), strconv.FormatInt(p.OrgID, 10))
		item := OrganizationUser{User: *user, MembershipStatus: membership.Status, Roles: roles}
		if organizationUserMatches(item, filter) {
			out = append(out, item)
		}
	}
	sortOrganizationUsers(out, filter)
	page, pageSize := normalizeListPage(filter.Page, filter.PageSize)
	total := int64(len(out))
	start := (page - 1) * pageSize
	if start > len(out) {
		start = len(out)
	}
	end := start + pageSize
	if end > len(out) {
		end = len(out)
	}
	pageResult := OrganizationUserPage{
		Items:         out[start:end],
		Page:          page,
		PageSize:      pageSize,
		Total:         total,
		StorageStatus: "persisted",
	}
	s.cacheSet(ctx, cacheKey, s.cfg.UserCacheTTL, pageResult)
	return pageResult, nil
}

// normalizeListPage 统一管理列表分页参数。
// pageSize 上限为 100，避免全量内存过滤后再返回过大的响应。
func normalizeListPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

// organizationMatches 判断组织是否命中管理端过滤条件。
// Keyword 会同时匹配 code、name 和 status，便于前端做单框搜索。
func organizationMatches(org model.Organization, filter OrganizationListFilter) bool {
	keyword := normalizeCode(filter.Keyword)
	if keyword != "" && !containsAnyFold(keyword, org.Code, org.Name, org.Status) {
		return false
	}
	if code := normalizeCode(filter.Code); code != "" && !strings.Contains(strings.ToLower(org.Code), code) {
		return false
	}
	if name := normalizeCode(filter.Name); name != "" && !strings.Contains(strings.ToLower(org.Name), name) {
		return false
	}
	if status := normalizeCode(filter.Status); status != "" && normalizeCode(org.Status) != status {
		return false
	}
	return true
}

// organizationUserMatches 判断组织成员是否命中用户列表过滤条件。
// 角色过滤会去掉 role: 前缀再比较，兼容 Casbin subject 与业务 role code。
func organizationUserMatches(item OrganizationUser, filter UserListFilter) bool {
	user := item.User
	keyword := normalizeCode(filter.Keyword)
	if keyword != "" && !containsAnyFold(keyword, user.Username, user.DisplayName, user.Email, item.MembershipStatus, strings.Join(item.Roles, " ")) {
		return false
	}
	if username := normalizeCode(filter.Username); username != "" && !strings.Contains(strings.ToLower(user.Username), username) {
		return false
	}
	if displayName := normalizeCode(filter.DisplayName); displayName != "" && !strings.Contains(strings.ToLower(user.DisplayName), displayName) {
		return false
	}
	if email := normalizeCode(filter.Email); email != "" && !strings.Contains(strings.ToLower(user.Email), email) {
		return false
	}
	if status := normalizeCode(filter.Status); status != "" && normalizeCode(item.MembershipStatus) != status {
		return false
	}
	if roleCode := normalizeCode(filter.RoleCode); roleCode != "" && !rolesContain(item.Roles, roleCode) {
		return false
	}
	return true
}

// containsAnyFold 对多个字段做大小写不敏感的包含匹配。
func containsAnyFold(needle string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), needle) {
			return true
		}
	}
	return false
}

// rolesContain 判断角色列表是否包含指定业务角色编码。
func rolesContain(roles []string, roleCode string) bool {
	for _, role := range roles {
		normalized := strings.TrimPrefix(normalizeCode(role), "role:")
		if normalized == roleCode {
			return true
		}
	}
	return false
}

// sortOrganizations 根据管理端排序字段稳定排列组织列表。
func sortOrganizations(items []model.Organization, filter OrganizationListFilter) {
	orderKey := normalizeCode(filter.OrderKey)
	if orderKey == "" {
		orderKey = "id"
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		compare := int64(0)
		switch orderKey {
		case "code":
			compare = int64(strings.Compare(strings.ToLower(left.Code), strings.ToLower(right.Code)))
		case "name":
			compare = int64(strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name)))
		case "status":
			compare = int64(strings.Compare(strings.ToLower(left.Status), strings.ToLower(right.Status)))
		default:
			compare = left.ID - right.ID
		}
		if compare == 0 {
			return false
		}
		if filter.Desc {
			return compare > 0
		}
		return compare < 0
	})
}

// sortOrganizationUsers 根据管理端排序字段稳定排列组织成员列表。
func sortOrganizationUsers(items []OrganizationUser, filter UserListFilter) {
	orderKey := normalizeCode(filter.OrderKey)
	if orderKey == "" {
		orderKey = "id"
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		compare := int64(0)
		switch orderKey {
		case "username":
			compare = int64(strings.Compare(strings.ToLower(left.User.Username), strings.ToLower(right.User.Username)))
		case "displayname", "display_name", "nickname", "nick_name":
			compare = int64(strings.Compare(strings.ToLower(left.User.DisplayName), strings.ToLower(right.User.DisplayName)))
		case "email":
			compare = int64(strings.Compare(strings.ToLower(left.User.Email), strings.ToLower(right.User.Email)))
		case "status":
			compare = int64(strings.Compare(strings.ToLower(left.MembershipStatus), strings.ToLower(right.MembershipStatus)))
		default:
			compare = left.User.ID - right.User.ID
		}
		if compare == 0 {
			return false
		}
		if filter.Desc {
			return compare > 0
		}
		return compare < 0
	})
}

// sessionMatches 判断会话是否命中列表过滤条件。
// 会话状态由 revoked、expired、active 派生，不直接依赖持久化状态字段。
func sessionMatches(session model.Session, filter SessionListFilter, now time.Time) bool {
	status := sessionStatus(session, now)
	keyword := normalizeCode(filter.Keyword)
	if keyword != "" && !containsAnyFold(
		keyword,
		strconv.FormatInt(session.ID, 10),
		strconv.FormatInt(session.UserID, 10),
		session.IPAddress,
		session.UserAgent,
		status,
		session.ProductCode,
		session.ClientType,
	) {
		return false
	}
	if ipAddress := normalizeCode(filter.IPAddress); ipAddress != "" && !strings.Contains(strings.ToLower(session.IPAddress), ipAddress) {
		return false
	}
	if filterStatus := normalizeCode(filter.Status); filterStatus != "" && status != filterStatus {
		return false
	}
	if productCode := normalizeProductCode(filter.ProductCode); productCode != "" && normalizeProductCode(session.ProductCode) != productCode {
		return false
	}
	if clientType := normalizeClientType(filter.ClientType); clientType != "" && normalizeClientType(session.ClientType) != clientType {
		return false
	}
	return true
}

// sessionStatus 根据撤销时间和过期时间派生会话状态。
func sessionStatus(session model.Session, now time.Time) string {
	if session.RevokedAt != nil {
		return "revoked"
	}
	if !session.ExpiresAt.IsZero() && !session.ExpiresAt.After(now) {
		return "expired"
	}
	return "active"
}

// sortSessions 根据管理端排序字段稳定排列会话列表。
func sortSessions(items []model.Session, filter SessionListFilter) {
	orderKey := normalizeCode(filter.OrderKey)
	if orderKey == "" {
		orderKey = "created_at"
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		compare := int64(0)
		switch orderKey {
		case "user_id", "userid":
			compare = left.UserID - right.UserID
		case "ip", "ip_address", "ipaddress":
			compare = int64(strings.Compare(strings.ToLower(left.IPAddress), strings.ToLower(right.IPAddress)))
		case "product_code", "productcode":
			compare = int64(strings.Compare(normalizeProductCode(left.ProductCode), normalizeProductCode(right.ProductCode)))
		case "client_type", "clienttype", "platform":
			compare = int64(strings.Compare(normalizeClientType(left.ClientType), normalizeClientType(right.ClientType)))
		case "expires_at", "expiresat":
			compare = compareTime(left.ExpiresAt, right.ExpiresAt)
		case "last_used_at", "lastusedat":
			compare = compareTime(sessionLastUsedAt(left), sessionLastUsedAt(right))
		default:
			compare = compareTime(left.CreatedAt, right.CreatedAt)
		}
		if compare == 0 {
			return false
		}
		if filter.Desc {
			return compare > 0
		}
		return compare < 0
	})
}

// sessionLastUsedAt 返回会话最后使用时间；未使用过则回退到创建时间。
func sessionLastUsedAt(session model.Session) time.Time {
	if session.LastUsedAt != nil {
		return *session.LastUsedAt
	}
	return session.CreatedAt
}

// compareTime 将时间比较结果转换为排序用的整数。
func compareTime(left, right time.Time) int64 {
	if left.Equal(right) {
		return 0
	}
	if left.After(right) {
		return 1
	}
	return -1
}

// UpdateUser 更新当前组织内成员状态或角色集合。
// 角色更新采用先删后加，确保提交后的角色集合与输入完全一致。
func (s *service) UpdateUser(ctx context.Context, input UpdateUserInput) (*OrganizationUser, error) {
	membership, err := s.repo.FindMembershipAnyStatus(ctx, input.Principal.OrgID, input.UserID)
	if err != nil {
		return nil, ErrNotFound
	}
	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return nil, ErrNotFound
	}
	if input.Status != nil {
		status := normalizeCode(*input.Status)
		switch status {
		case model.StatusActive, model.StatusDisabled:
			membership.Status = status
			if err := s.repo.SaveMembership(ctx, membership); err != nil {
				return nil, err
			}
			_ = s.audit(ctx, s.repo, &input.Principal.OrgID, &input.Principal.UserID, "user.update", "membership", strconv.FormatInt(membership.ID, 10), input.IPAddress, input.UserAgent, map[string]any{"userId": input.UserID, "status": status}, SessionContext{ProductCode: input.Principal.ProductCode, ClientType: input.Principal.ClientType})
		default:
			return nil, ErrInvalidInput
		}
	}
	if input.HasRoles {
		for _, roleCode := range input.Roles {
			if normalizeCode(roleCode) == model.RolePlatformOwner {
				return nil, ErrForbidden
			}
			if _, err := s.repo.FindRole(ctx, input.Principal.OrgID, normalizeCode(roleCode)); err != nil {
				return nil, ErrNotFound
			}
		}
		if err := s.repo.DeleteCasbinRules(ctx, "g", userSubject(input.UserID), "", strconv.FormatInt(input.Principal.OrgID, 10)); err != nil {
			return nil, err
		}
		for _, roleCode := range input.Roles {
			if err := s.addUserRole(ctx, s.repo, input.UserID, input.Principal.OrgID, roleCode); err != nil {
				return nil, err
			}
		}
		_ = s.LoadPolicies(ctx)
		_ = s.audit(ctx, s.repo, &input.Principal.OrgID, &input.Principal.UserID, "user.roles.update", "user", strconv.FormatInt(input.UserID, 10), input.IPAddress, input.UserAgent, map[string]any{"roles": input.Roles}, SessionContext{ProductCode: input.Principal.ProductCode, ClientType: input.Principal.ClientType})
	}
	roles, _ := s.authz.GetRolesForUser(ctx, userSubject(user.ID), strconv.FormatInt(input.Principal.OrgID, 10))
	s.bumpOrganizationCaches(ctx, input.Principal.ProductCode, input.Principal.OrgID)
	s.bumpUserOrganizationsCache(ctx, input.Principal.ProductCode, input.UserID)
	return &OrganizationUser{User: *user, MembershipStatus: membership.Status, Roles: roles}, nil
}

// CreateAPIToken 为指定用户创建机器访问 token。
// 明文 token 只在返回值中出现一次，数据库只保存 hash 和展示前缀；RoleCode 限制该 token 的权限范围。
func (s *service) CreateAPIToken(ctx context.Context, input CreateAPITokenInput) (CreateAPITokenResult, error) {
	roleCode := normalizeCode(input.RoleCode)
	if input.UserID <= 0 || roleCode == "" {
		return CreateAPITokenResult{}, ErrInvalidInput
	}
	if roleCode == model.RolePlatformOwner {
		if input.UserID != input.Principal.UserID {
			return CreateAPITokenResult{}, ErrForbidden
		}
		ok, err := s.userHasRole(ctx, input.Principal.UserID, input.Principal.OrgID, model.RolePlatformOwner)
		if err != nil || !ok {
			return CreateAPITokenResult{}, ErrForbidden
		}
	}
	days, err := normalizeAPITokenDays(input.Days)
	if err != nil {
		return CreateAPITokenResult{}, err
	}
	if _, err := s.repo.FindMembership(ctx, input.Principal.OrgID, input.UserID); err != nil {
		return CreateAPITokenResult{}, ErrNotFound
	}
	user, err := s.repo.FindUserByID(ctx, input.UserID)
	if err != nil {
		return CreateAPITokenResult{}, ErrNotFound
	}
	if err := s.ensureUserCanLogin(user); err != nil {
		return CreateAPITokenResult{}, err
	}
	if _, err := s.repo.FindRole(ctx, input.Principal.OrgID, roleCode); err != nil {
		return CreateAPITokenResult{}, ErrNotFound
	}
	if ok, err := s.userHasRole(ctx, input.UserID, input.Principal.OrgID, roleCode); err != nil || !ok {
		return CreateAPITokenResult{}, ErrForbidden
	}

	raw, hash, prefix, err := s.issueAPITokenSecret()
	if err != nil {
		return CreateAPITokenResult{}, err
	}
	now := s.now()
	var expiresAt *time.Time
	if days > 0 {
		expires := now.Add(time.Duration(days) * 24 * time.Hour)
		expiresAt = &expires
	}
	apiToken := &model.APIToken{
		ID:          s.ids.NextID(),
		OrgID:       input.Principal.OrgID,
		UserID:      input.UserID,
		RoleCode:    roleCode,
		TokenPrefix: prefix,
		TokenHash:   hash,
		Status:      model.StatusActive,
		ExpiresAt:   expiresAt,
		Remark:      strings.TrimSpace(input.Remark),
		CreatedBy:   input.Principal.UserID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.CreateAPIToken(ctx, apiToken); err != nil {
		return CreateAPITokenResult{}, err
	}
	_ = s.audit(ctx, s.repo, &input.Principal.OrgID, &input.Principal.UserID, "api_token.create", "api_token", strconv.FormatInt(apiToken.ID, 10), input.IPAddress, input.UserAgent, map[string]any{"userId": input.UserID, "roleCode": roleCode, "expiresAt": expiresAt}, SessionContext{ProductCode: input.Principal.ProductCode, ClientType: input.Principal.ClientType})
	return CreateAPITokenResult{Item: s.apiTokenView(ctx, *apiToken, user), Token: raw}, nil
}

// ListAPITokens 分页列出当前组织的 API token。
// 过期状态由仓储过滤和视图映射共同处理，避免把已过期 token 误展示为 active。
func (s *service) ListAPITokens(ctx context.Context, p Principal, filter APITokenFilter) (APITokenPage, error) {
	filter.Status = normalizeCode(filter.Status)
	switch filter.Status {
	case "", model.StatusActive, model.StatusExpired, model.StatusRevoked:
	default:
		return APITokenPage{}, ErrInvalidInput
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 10
	}
	filter.Now = s.now()
	items, total, err := s.repo.ListAPITokens(ctx, p.OrgID, filter)
	if err != nil {
		return APITokenPage{}, err
	}
	views := make([]APITokenView, 0, len(items))
	for _, item := range items {
		var user *model.User
		if found, err := s.repo.FindUserByID(ctx, item.UserID); err == nil {
			user = found
		}
		views = append(views, s.apiTokenView(ctx, item, user))
	}
	return APITokenPage{Items: views, Page: filter.Page, PageSize: filter.PageSize, Total: total, StorageStatus: "persisted"}, nil
}

// RevokeAPIToken 撤销当前组织内的 API token。
// 重复撤销会直接成功，保证管理端重试操作具备幂等性。
func (s *service) RevokeAPIToken(ctx context.Context, input RevokeAPITokenInput) error {
	if input.TokenID <= 0 {
		return ErrInvalidInput
	}
	apiToken, err := s.repo.FindAPITokenByID(ctx, input.TokenID)
	if err != nil {
		return ErrNotFound
	}
	if apiToken.OrgID != input.Principal.OrgID {
		return ErrForbidden
	}
	if apiToken.Status == model.StatusRevoked {
		return nil
	}
	now := s.now()
	apiToken.Status = model.StatusRevoked
	apiToken.RevokedAt = &now
	apiToken.RevokedBy = &input.Principal.UserID
	if err := s.repo.SaveAPIToken(ctx, apiToken); err != nil {
		return err
	}
	return s.audit(ctx, s.repo, &input.Principal.OrgID, &input.Principal.UserID, "api_token.revoke", "api_token", strconv.FormatInt(apiToken.ID, 10), input.IPAddress, input.UserAgent, map[string]any{"userId": apiToken.UserID, "roleCode": apiToken.RoleCode}, SessionContext{ProductCode: input.Principal.ProductCode, ClientType: input.Principal.ClientType})
}

// ListRoles 返回当前组织角色，并尽力补齐每个角色的权限列表。
func (s *service) ListRoles(ctx context.Context, p Principal) ([]model.Role, error) {
	cacheKey := s.organizationRolesCacheKey(ctx, p)
	var cached []model.Role
	if s.cacheGet(ctx, cacheKey, s.cfg.RoleCacheTTL, &cached) {
		return cached, nil
	}
	roles, err := s.repo.ListRoles(ctx, p.OrgID)
	if err != nil {
		return nil, err
	}
	for i := range roles {
		_ = s.hydrateRolePermissions(ctx, &roles[i])
	}
	s.cacheSet(ctx, cacheKey, s.cfg.RoleCacheTTL, roles)
	return roles, nil
}

// CreateRole 创建组织自定义角色并写入权限策略。
// 无效权限编码会被跳过，避免单个坏权限阻断角色基础信息创建。
func (s *service) CreateRole(ctx context.Context, input CreateRoleInput) (*model.Role, error) {
	code := normalizeCode(input.Code)
	name := strings.TrimSpace(input.Name)
	if code == "" || name == "" {
		return nil, ErrInvalidInput
	}
	if isBuiltinRoleCode(code) {
		return nil, ErrForbidden
	}
	now := s.now()
	role := &model.Role{ID: s.ids.NextID(), OrgID: input.Principal.OrgID, Code: code, Name: name, Description: strings.TrimSpace(input.Description), CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, err
	}
	for _, permission := range input.Permissions {
		obj, act := permissionObjectAction(permission)
		if obj == "" || act == "" {
			continue
		}
		if err := s.addPolicy(ctx, s.repo, input.Principal.OrgID, code, input.Principal.ProductCode, model.PermissionScopeTenant, obj, act); err != nil {
			return nil, err
		}
	}
	_ = s.LoadPolicies(ctx)
	_ = s.hydrateRolePermissions(ctx, role)
	s.bumpOrganizationCaches(ctx, input.Principal.ProductCode, input.Principal.OrgID)
	return role, nil
}

// UpdateRole 更新组织自定义角色。
// 系统内置角色禁止修改；权限更新采用替换语义，确保策略表与输入保持一致。
func (s *service) UpdateRole(ctx context.Context, input UpdateRoleInput) (*model.Role, error) {
	role, err := s.repo.FindRoleByID(ctx, input.RoleID)
	if err != nil {
		return nil, ErrNotFound
	}
	if role.OrgID != input.Principal.OrgID {
		return nil, ErrForbidden
	}
	if role.System {
		return nil, ErrForbidden
	}
	name := strings.TrimSpace(input.Name)
	if name != "" {
		role.Name = name
	}
	role.Description = strings.TrimSpace(input.Description)
	if err := s.repo.SaveRole(ctx, role); err != nil {
		return nil, err
	}
	if input.HasPermissions {
		if err := s.repo.DeleteCasbinRules(ctx, "p", roleSubject(role.Code), strconv.FormatInt(input.Principal.OrgID, 10)); err != nil {
			return nil, err
		}
		for _, permission := range input.Permissions {
			obj, act := permissionObjectAction(permission)
			if obj == "" || act == "" {
				continue
			}
			if err := s.addPolicy(ctx, s.repo, input.Principal.OrgID, role.Code, input.Principal.ProductCode, model.PermissionScopeTenant, obj, act); err != nil {
				return nil, err
			}
		}
		_ = s.LoadPolicies(ctx)
	}
	_ = s.hydrateRolePermissions(ctx, role)
	_ = s.audit(ctx, s.repo, &input.Principal.OrgID, &input.Principal.UserID, "role.update", "role", strconv.FormatInt(role.ID, 10), input.IPAddress, input.UserAgent, map[string]any{"permissions": input.Permissions}, SessionContext{ProductCode: input.Principal.ProductCode, ClientType: input.Principal.ClientType})
	s.bumpOrganizationCaches(ctx, input.Principal.ProductCode, input.Principal.OrgID)
	return role, nil
}

// ListPermissions 返回系统已登记的权限清单。
func (s *service) ListPermissions(ctx context.Context, p Principal) ([]model.Permission, error) {
	sessionCtx := s.resolveSessionContext(p.ProductCode, p.ClientType)
	includePlatform, err := s.userHasRole(ctx, p.UserID, p.OrgID, model.RolePlatformOwner)
	if err != nil {
		return nil, err
	}
	cacheKey := s.productPermissionsCacheKey(ctx, sessionCtx.ProductCode, includePlatform)
	var cached []model.Permission
	if s.cacheGet(ctx, cacheKey, s.cfg.PermissionCacheTTL, &cached) {
		return cached, nil
	}
	permissions, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.Permission, 0, len(permissions))
	for _, permission := range permissions {
		permissionProductCode := s.resolveSessionContext(permission.ProductCode, "").ProductCode
		if permissionProductCode != sessionCtx.ProductCode {
			continue
		}
		if permission.Scope != model.PermissionScopeTenant && !includePlatform {
			continue
		}
		permission.ProductCode = permissionProductCode
		out = append(out, permission)
	}
	s.cacheSet(ctx, cacheKey, s.cfg.PermissionCacheTTL, out)
	return out, nil
}

// ListSessions 返回当前组织内可见的会话列表。
// 默认只看当前用户；Scope 为 org 且未指定 UserID 时切换为组织视角。
func (s *service) ListSessions(ctx context.Context, p Principal, filter SessionListFilter) (SessionPage, error) {
	var (
		sessions []model.Session
		err      error
	)
	if normalizeCode(filter.Scope) == "org" && filter.UserID == 0 {
		sessions, err = s.repo.ListSessionsByOrg(ctx, p.OrgID)
	} else {
		userID := filter.UserID
		if userID == 0 {
			userID = p.UserID
		}
		sessions, err = s.repo.ListSessionsByUser(ctx, userID)
	}
	if err != nil {
		return SessionPage{}, err
	}
	now := s.now()
	out := make([]model.Session, 0, len(sessions))
	for _, session := range sessions {
		if session.OrgID != p.OrgID {
			continue
		}
		sessionCtx := s.resolveSessionContext(session.ProductCode, session.ClientType)
		session.ProductCode = sessionCtx.ProductCode
		session.ClientType = sessionCtx.ClientType
		if sessionMatches(session, filter, now) {
			out = append(out, session)
		}
	}
	sortSessions(out, filter)
	page, pageSize := normalizeListPage(filter.Page, filter.PageSize)
	total := int64(len(out))
	start := (page - 1) * pageSize
	if start > len(out) {
		start = len(out)
	}
	end := start + pageSize
	if end > len(out) {
		end = len(out)
	}
	return SessionPage{
		Items:         out[start:end],
		Page:          page,
		PageSize:      pageSize,
		Total:         total,
		StorageStatus: "persisted",
	}, nil
}

// RevokeSession 撤销当前组织内的指定会话。
func (s *service) RevokeSession(ctx context.Context, p Principal, sessionID int64) error {
	session, err := s.repo.FindSessionByID(ctx, sessionID)
	if err != nil {
		return ErrNotFound
	}
	if session.OrgID != p.OrgID {
		return ErrForbidden
	}
	now := s.now()
	session.RevokedAt = &now
	return s.repo.SaveSession(ctx, session)
}

// ListAuditLogs 返回当前组织的审计日志。
func (s *service) ListAuditLogs(ctx context.Context, p Principal, filter AuditLogFilter) ([]model.AuditLog, error) {
	logs, err := s.repo.ListAuditLogs(ctx, p.OrgID, filter)
	if err != nil {
		return nil, err
	}
	for i := range logs {
		sessionCtx := s.resolveSessionContext(logs[i].ProductCode, logs[i].ClientType)
		logs[i].ProductCode = sessionCtx.ProductCode
		logs[i].ClientType = sessionCtx.ClientType
	}
	return logs, nil
}

// RecordAudit 为外部调用方写入一条 IAM 审计日志。
func (s *service) RecordAudit(ctx context.Context, p Principal, action, resource, resourceID, ip, userAgent string, metadata map[string]any) error {
	return s.audit(ctx, s.repo, &p.OrgID, &p.UserID, action, resource, resourceID, ip, userAgent, metadata, SessionContext{ProductCode: p.ProductCode, ClientType: p.ClientType})
}

// LoadPolicies 从仓储加载 Casbin 规则到授权引擎。
// 角色或权限变更后调用该方法，保证后续 Authorize 使用最新策略。
func (s *service) LoadPolicies(ctx context.Context) error {
	rules, err := s.repo.ListCasbinRules(ctx)
	if err != nil {
		return err
	}
	return s.authz.LoadRules(ctx, rules)
}

// createSessionAndTokens 使用主仓储创建会话并签发 token。
func (s *service) createSessionAndTokens(ctx context.Context, user *model.User, orgID int64, userAgent, ip, productCode, clientType string) (TokenPair, error) {
	return s.createSessionAndTokensWithRepo(ctx, s.repo, user, orgID, userAgent, ip, productCode, clientType)
}

// createSessionAndTokensWithRepo 在指定仓储上下文中创建会话并保存 refresh token hash。
// 该方法可在事务内复用，确保注册或初始化流程与首个会话保持原子性。
func (s *service) createSessionAndTokensWithRepo(ctx context.Context, repo Repository, user *model.User, orgID int64, userAgent, ip, productCode, clientType string) (TokenPair, error) {
	now := s.now()
	sessionCtx := s.resolveSessionContext(productCode, clientType)
	if s.cfg.SingleSessionPerContext {
		if err := s.revokeActiveSessionsForContext(ctx, repo, user.ID, orgID, sessionCtx, now); err != nil {
			return TokenPair{}, err
		}
	}
	sessionID := s.ids.NextID()
	pair, err := s.tokens.IssuePair(ctx, TokenSubject{UserID: user.ID, OrgID: orgID, SessionID: sessionID, ProductCode: sessionCtx.ProductCode})
	if err != nil {
		return TokenPair{}, err
	}
	session := &model.Session{ID: sessionID, UserID: user.ID, OrgID: orgID, ProductCode: sessionCtx.ProductCode, ClientType: sessionCtx.ClientType, RefreshTokenHash: pair.RefreshTokenHash, UserAgent: userAgent, IPAddress: ip, ExpiresAt: pair.RefreshExpiresAt, CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateSession(ctx, session); err != nil {
		return TokenPair{}, err
	}
	return tokenPair(pair, session), nil
}

// loginOrg 解析登录目标组织。
// 未指定 org code 时使用用户的第一个成员组织；指定时必须验证成员关系。
func (s *service) loginOrg(ctx context.Context, userID int64, code string) (*model.Organization, error) {
	if code != "" {
		org, err := s.repo.FindOrganizationByCode(ctx, normalizeCode(code))
		if err != nil {
			return nil, ErrNotFound
		}
		if _, err := s.repo.FindMembership(ctx, org.ID, userID); err != nil {
			return nil, ErrForbidden
		}
		return org, nil
	}
	memberships, err := s.repo.ListMembershipsByUser(ctx, userID)
	if err != nil || len(memberships) == 0 {
		return nil, ErrForbidden
	}
	return s.repo.FindOrganizationByID(ctx, memberships[0].OrgID)
}

// ensureUserCanLogin 校验用户状态和锁定时间。
func (s *service) ensureUserCanLogin(user *model.User) error {
	if user.Status != model.StatusActive {
		return ErrAccountDisabled
	}
	if user.LockedUntil != nil && user.LockedUntil.After(s.now()) {
		return ErrAccountLocked
	}
	return nil
}

// ensureSessionActive 校验会话未撤销且未过期。
func (s *service) ensureSessionActive(session *model.Session) error {
	if session.RevokedAt != nil {
		return ErrSessionRevoked
	}
	if session.ExpiresAt.Before(s.now()) {
		return ErrInvalidToken
	}
	return nil
}

func (s *service) resolveSessionContext(productCode string, clientType string) SessionContext {
	productCode = normalizeProductCode(firstNonEmpty(productCode, s.cfg.DefaultProductCode))
	clientType = normalizeClientType(firstNonEmpty(clientType, s.cfg.DefaultClientType))
	return SessionContext{ProductCode: productCode, ClientType: clientType}
}

func (s *service) normalizePermissionContext(permission PermissionContext) PermissionContext {
	sessionCtx := s.resolveSessionContext(permission.ProductCode, "")
	permission.ProductCode = sessionCtx.ProductCode
	permission.Scope = normalizePermissionScope(permission.Scope)
	permission.Object = strings.TrimSpace(permission.Object)
	permission.Action = strings.TrimSpace(permission.Action)
	if !validPermissionScope(permission.Scope) {
		permission.Scope = ""
	}
	return permission
}

func (s *service) revokeActiveSessionsForContext(ctx context.Context, repo Repository, userID int64, orgID int64, sessionCtx SessionContext, now time.Time) error {
	sessions, err := repo.ListSessionsByUser(ctx, userID)
	if err != nil {
		return err
	}
	for index := range sessions {
		session := sessions[index]
		if session.OrgID != orgID || session.ProductCode != sessionCtx.ProductCode || session.ClientType != sessionCtx.ClientType {
			continue
		}
		if session.RevokedAt != nil || session.ExpiresAt.Before(now) {
			continue
		}
		session.RevokedAt = &now
		if err := repo.SaveSession(ctx, &session); err != nil {
			return err
		}
	}
	return nil
}

// authenticateAPIToken 使用机器 token 构造 Principal。
// token 必须 active、未过期，且用户仍拥有 token 声明的角色，防止角色移除后旧 token 继续越权。
func (s *service) authenticateAPIToken(ctx context.Context, raw string) (Principal, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Principal{}, ErrInvalidToken
	}
	apiToken, err := s.repo.FindAPITokenByHash(ctx, s.tokens.HashRefreshToken(raw))
	if err != nil {
		return Principal{}, ErrInvalidToken
	}
	if apiToken.Status != model.StatusActive {
		return Principal{}, ErrSessionRevoked
	}
	if apiToken.ExpiresAt != nil && apiToken.ExpiresAt.Before(s.now()) {
		return Principal{}, ErrInvalidToken
	}
	user, err := s.repo.FindUserByID(ctx, apiToken.UserID)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}
	if err := s.ensureUserCanLogin(user); err != nil {
		return Principal{}, err
	}
	if _, err := s.repo.FindMembership(ctx, apiToken.OrgID, apiToken.UserID); err != nil {
		return Principal{}, ErrForbidden
	}
	if _, err := s.repo.FindRole(ctx, apiToken.OrgID, apiToken.RoleCode); err != nil {
		return Principal{}, ErrForbidden
	}
	if ok, err := s.userHasRole(ctx, apiToken.UserID, apiToken.OrgID, apiToken.RoleCode); err != nil || !ok {
		return Principal{}, ErrForbidden
	}
	now := s.now()
	apiToken.LastUsedAt = &now
	_ = s.repo.SaveAPIToken(ctx, apiToken)
	sessionCtx := s.resolveSessionContext("", "api_token")
	return Principal{UserID: user.ID, OrgID: apiToken.OrgID, ProductCode: sessionCtx.ProductCode, ClientType: sessionCtx.ClientType, Username: user.Username, Email: user.Email, RoleCode: apiToken.RoleCode}, nil
}

// authorizeRole 按 Principal 中的 RoleCode 校验权限。
// 该路径主要服务 API token，因为机器 token 不绑定普通会话的 Casbin subject。
func (s *service) authorizeRole(ctx context.Context, p Principal, permission PermissionContext) (bool, error) {
	permissions, err := s.repo.ListRolePermissions(ctx, p.OrgID, roleSubject(p.RoleCode))
	if err != nil {
		return false, err
	}
	for _, grant := range permissions {
		if permissionAllows(grant, permission) {
			return true, nil
		}
	}
	return false, nil
}

// recordFailedLogin 递增失败登录次数，并在达到阈值时写入锁定时间。
func (s *service) recordFailedLogin(ctx context.Context, user *model.User) error {
	user.FailedLoginAttempts++
	if user.FailedLoginAttempts >= s.cfg.LoginMaxFailures {
		lockedUntil := s.now().Add(s.cfg.LoginLockDuration)
		user.LockedUntil = &lockedUntil
	}
	return s.repo.SaveUser(ctx, user)
}

// userHasRole 校验用户在组织内是否仍拥有指定角色。
func (s *service) userHasRole(ctx context.Context, userID, orgID int64, roleCode string) (bool, error) {
	roles, err := s.authz.GetRolesForUser(ctx, userSubject(userID), strconv.FormatInt(orgID, 10))
	if err != nil {
		return false, err
	}
	want := roleSubject(roleCode)
	for _, role := range roles {
		if roleSubject(role) == want {
			return true, nil
		}
	}
	return false, nil
}

// verifyUserMFA 解密用户 TOTP secret 并校验一次性验证码。
func (s *service) verifyUserMFA(ctx context.Context, userID int64, code string) error {
	factor, err := s.repo.FindActiveMFAFactor(ctx, userID)
	if err != nil {
		return ErrUnauthorized
	}
	secret, err := s.decryptSecret(factor.Secret)
	if err != nil {
		return err
	}
	if !s.totp.ValidateTOTP(code, secret) {
		return ErrUnauthorized
	}
	return nil
}

// ensureMembership 幂等创建用户与组织的成员关系。
func (s *service) ensureMembership(ctx context.Context, repo Repository, orgID, userID int64) error {
	if _, err := repo.FindMembership(ctx, orgID, userID); err == nil {
		return nil
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	now := s.now()
	return repo.CreateMembership(ctx, &model.Membership{ID: s.ids.NextID(), OrgID: orgID, UserID: userID, Status: model.StatusActive, CreatedAt: now, UpdatedAt: now})
}

// ensureBuiltins 为组织补齐内置权限、系统角色和默认策略。
// 该方法按缺失项创建，允许初始化流程重复执行而不破坏已有配置。
func (s *service) ensureBuiltins(ctx context.Context, repo Repository, orgID int64, orgKind string) error {
	for _, permission := range builtinPermissions {
		scope := normalizePermissionScope(permission.Scope)
		if scope == "" {
			scope = model.PermissionScopeTenant
		}
		if _, err := repo.FindPermission(ctx, s.cfg.DefaultProductCode, scope, permission.Code); err == nil {
			continue
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		now := s.now()
		if err := repo.CreatePermission(ctx, &model.Permission{ID: s.ids.NextID(), ProductCode: s.cfg.DefaultProductCode, Scope: scope, Code: permission.Code, Name: permission.Name, Description: permission.Description, CreatedAt: now, UpdatedAt: now}); err != nil {
			return err
		}
	}
	type builtinRoleSeed struct {
		code string
		name string
	}
	roles := []builtinRoleSeed{}
	if orgKind == model.OrgKindPlatform {
		roles = append(roles, builtinRoleSeed{model.RolePlatformOwner, "Platform Owner"})
	}
	roles = append(roles,
		builtinRoleSeed{model.RoleOwner, "Owner"},
		builtinRoleSeed{model.RoleAdmin, "Admin"},
		builtinRoleSeed{model.RoleMember, "Member"},
	)
	for _, role := range roles {
		if _, err := repo.FindRole(ctx, orgID, role.code); err == nil {
			continue
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		now := s.now()
		if err := repo.CreateRole(ctx, &model.Role{ID: s.ids.NextID(), OrgID: orgID, Code: role.code, Name: role.name, Description: role.name, System: true, CreatedAt: now, UpdatedAt: now}); err != nil {
			return err
		}
	}
	for _, permission := range builtinPermissions {
		obj, act := permissionObjectAction(permission.Code)
		scope := normalizePermissionScope(permission.Scope)
		if obj == "" || act == "" || scope == "" {
			continue
		}
		if orgKind == model.OrgKindPlatform {
			if err := s.addPolicy(ctx, repo, orgID, model.RolePlatformOwner, s.cfg.DefaultProductCode, scope, obj, act); err != nil {
				return err
			}
		}
		if scope == model.PermissionScopeTenant {
			if err := s.addPolicy(ctx, repo, orgID, model.RoleOwner, s.cfg.DefaultProductCode, scope, obj, act); err != nil {
				return err
			}
			if err := s.addPolicy(ctx, repo, orgID, model.RoleAdmin, s.cfg.DefaultProductCode, scope, obj, act); err != nil {
				return err
			}
		}
	}
	return s.addPolicy(ctx, repo, orgID, model.RoleMember, s.cfg.DefaultProductCode, model.PermissionScopeTenant, "me", "read")
}

// addUserRole 写入用户到角色的授权规则。
func (s *service) cacheGet(ctx context.Context, key string, ttl time.Duration, out any) bool {
	if s.cache == nil || ttl <= 0 {
		return false
	}
	ok, err := s.cache.GetJSON(ctx, key, out)
	return err == nil && ok
}

func (s *service) cacheSet(ctx context.Context, key string, ttl time.Duration, value any) {
	if s.cache == nil || ttl <= 0 {
		return
	}
	_ = s.cache.SetJSON(ctx, key, value, ttl)
}

func (s *service) cacheEpoch(ctx context.Context, key string) string {
	if s.cache == nil {
		return "0"
	}
	var epoch int64
	ok, err := s.cache.GetJSON(ctx, key, &epoch)
	if err != nil || !ok {
		return "0"
	}
	return strconv.FormatInt(epoch, 10)
}

func (s *service) bumpCacheEpoch(ctx context.Context, key string) {
	if s.cache == nil {
		return
	}
	_, _ = s.cache.Incr(ctx, key, 0)
}

func (s *service) userOrganizationsCacheKey(ctx context.Context, p Principal) string {
	sessionCtx := s.resolveSessionContext(p.ProductCode, p.ClientType)
	epochKey := s.cacheKey(cacheScopeEpoch, cacheScopeUserOrgs, sessionCtx.ProductCode, strconv.FormatInt(p.UserID, 10))
	return s.cacheKey(cacheScopeUserOrgs, sessionCtx.ProductCode, strconv.FormatInt(p.UserID, 10), s.cacheEpoch(ctx, epochKey))
}

func (s *service) organizationUsersCacheKey(ctx context.Context, p Principal, filter UserListFilter) string {
	sessionCtx := s.resolveSessionContext(p.ProductCode, p.ClientType)
	orgID := strconv.FormatInt(p.OrgID, 10)
	epochKey := s.cacheKey(cacheScopeEpoch, cacheScopeOrgUsers, sessionCtx.ProductCode, orgID)
	return s.cacheKey(cacheScopeOrgUsers, sessionCtx.ProductCode, orgID, s.cacheEpoch(ctx, epochKey), hashCacheValue(filter))
}

func (s *service) organizationRolesCacheKey(ctx context.Context, p Principal) string {
	sessionCtx := s.resolveSessionContext(p.ProductCode, p.ClientType)
	orgID := strconv.FormatInt(p.OrgID, 10)
	epochKey := s.cacheKey(cacheScopeEpoch, cacheScopeOrgRoles, sessionCtx.ProductCode, orgID)
	return s.cacheKey(cacheScopeOrgRoles, sessionCtx.ProductCode, orgID, s.cacheEpoch(ctx, epochKey))
}

func (s *service) productPermissionsCacheKey(ctx context.Context, productCode string, includePlatform bool) string {
	productCode = normalizeProductCode(firstNonEmpty(productCode, s.cfg.DefaultProductCode))
	epochKey := s.cacheKey(cacheScopeEpoch, cacheScopePermissions, productCode)
	scopeKey := model.PermissionScopeTenant
	if includePlatform {
		scopeKey = "all"
	}
	return s.cacheKey(cacheScopePermissions, productCode, scopeKey, s.cacheEpoch(ctx, epochKey))
}

func (s *service) bumpUserOrganizationsCache(ctx context.Context, productCode string, userID int64) {
	productCode = normalizeProductCode(firstNonEmpty(productCode, s.cfg.DefaultProductCode))
	s.bumpCacheEpoch(ctx, s.cacheKey(cacheScopeEpoch, cacheScopeUserOrgs, productCode, strconv.FormatInt(userID, 10)))
}

func (s *service) bumpOrganizationCaches(ctx context.Context, productCode string, orgID int64) {
	productCode = normalizeProductCode(firstNonEmpty(productCode, s.cfg.DefaultProductCode))
	orgIDText := strconv.FormatInt(orgID, 10)
	s.bumpCacheEpoch(ctx, s.cacheKey(cacheScopeEpoch, cacheScopeOrgUsers, productCode, orgIDText))
	s.bumpCacheEpoch(ctx, s.cacheKey(cacheScopeEpoch, cacheScopeOrgRoles, productCode, orgIDText))
}

func (s *service) cacheKey(parts ...string) string {
	normalized := make([]string, 0, len(parts)+1)
	normalized = append(normalized, cacheScopeIAM)
	for _, part := range parts {
		normalized = append(normalized, cacheKeyPart(part))
	}
	return strings.Join(normalized, ":")
}

func cacheKeyPart(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ":", "_")
	if value == "" {
		return "_"
	}
	return value
}

func hashCacheValue(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		raw = []byte(fmt.Sprintf("%v", value))
	}
	sum := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (s *service) addUserRole(ctx context.Context, repo Repository, userID, orgID int64, roleCode string) error {
	now := s.now()
	rule := &model.CasbinRule{ID: s.ids.NextID(), PType: "g", V0: userSubject(userID), V1: roleSubject(roleCode), V2: strconv.FormatInt(orgID, 10), CreatedAt: now}
	return repo.AddCasbinRule(ctx, rule)
}

// addPolicy 写入角色到对象动作的授权策略。
func (s *service) addPolicy(ctx context.Context, repo Repository, orgID int64, roleCode, productCode, scope, obj, act string) error {
	productCode = normalizeProductCode(firstNonEmpty(productCode, s.cfg.DefaultProductCode))
	scope = normalizePermissionScope(scope)
	if productCode == "" || !validPermissionScope(scope) || strings.TrimSpace(obj) == "" || strings.TrimSpace(act) == "" {
		return ErrInvalidInput
	}
	now := s.now()
	rule := &model.CasbinRule{ID: s.ids.NextID(), PType: "p", V0: roleSubject(roleCode), V1: strconv.FormatInt(orgID, 10), V2: productCode, V3: scope, V4: strings.TrimSpace(obj), V5: strings.TrimSpace(act), CreatedAt: now}
	return repo.AddCasbinRule(ctx, rule)
}

// audit 写入 IAM 审计日志。
// metadata 序列化失败时会被忽略为零值，避免审计附加信息阻断主业务流程。
func (s *service) audit(ctx context.Context, repo Repository, orgID, userID *int64, action, resource, resourceID, ip, userAgent string, metadata map[string]any, contexts ...SessionContext) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	sessionCtx := s.resolveSessionContext("", "")
	if len(contexts) > 0 {
		sessionCtx = s.resolveSessionContext(contexts[0].ProductCode, contexts[0].ClientType)
	}
	raw, _ := json.Marshal(metadata)
	return repo.CreateAuditLog(ctx, &model.AuditLog{ID: s.ids.NextID(), OrgID: orgID, UserID: userID, ProductCode: sessionCtx.ProductCode, ClientType: sessionCtx.ClientType, Action: action, Resource: resource, ResourceID: resourceID, IPAddress: ip, UserAgent: userAgent, Metadata: string(raw), CreatedAt: s.now()})
}

// oneTimeToken 生成一次性明文 token 及其 hash。
// 明文用于通知用户，hash 用于持久化查找。
func (s *service) oneTimeToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", "", err
	}
	value := base64.RawURLEncoding.EncodeToString(raw)
	return value, s.tokens.HashRefreshToken(value), nil
}

// issueAPITokenSecret 生成 API token 明文、hash 和展示前缀。
func (s *service) issueAPITokenSecret() (string, string, string, error) {
	raw := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", "", "", err
	}
	value := "aoi_" + base64.RawURLEncoding.EncodeToString(raw)
	return value, s.tokens.HashRefreshToken(value), apiTokenDisplayPrefix(value), nil
}

// apiTokenDisplayPrefix 返回可展示的 token 前缀。
func apiTokenDisplayPrefix(value string) string {
	if len(value) <= 16 {
		return value
	}
	return value[:16]
}

// normalizeAPITokenDays 归一化 API token 有效期。
// 0 表示默认 30 天，-1 表示不过期，其余值限制在 1 到 3650 天。
func normalizeAPITokenDays(value int) (int, error) {
	if value == 0 {
		return 30, nil
	}
	if value == -1 {
		return value, nil
	}
	if value < 1 || value > 3650 {
		return 0, ErrInvalidInput
	}
	return value, nil
}

// apiTokenView 将持久化 token 转换为安全展示模型。
// 已过期但仍标记 active 的 token 会在视图层显示为 expired。
func (s *service) apiTokenView(_ context.Context, apiToken model.APIToken, user *model.User) APITokenView {
	status := apiToken.Status
	if status == model.StatusActive && apiToken.ExpiresAt != nil && apiToken.ExpiresAt.Before(s.now()) {
		status = model.StatusExpired
	}
	view := APITokenView{
		ID:                apiToken.ID,
		OrgID:             apiToken.OrgID,
		UserID:            apiToken.UserID,
		RoleCode:          apiToken.RoleCode,
		TokenPrefix:       apiToken.TokenPrefix,
		Status:            status,
		ExpiresAt:         apiToken.ExpiresAt,
		LastUsedAt:        apiToken.LastUsedAt,
		LastUsedIPAddress: apiToken.LastUsedIPAddress,
		RevokedAt:         apiToken.RevokedAt,
		RevokedBy:         apiToken.RevokedBy,
		Remark:            apiToken.Remark,
		CreatedBy:         apiToken.CreatedBy,
		CreatedAt:         apiToken.CreatedAt,
		UpdatedAt:         apiToken.UpdatedAt,
	}
	if user != nil {
		view.Username = user.Username
		view.UserDisplayName = user.DisplayName
	}
	return view
}

func (s *service) debugDelivery(token string, adminPath string) NotificationDelivery {
	if !s.debugNotificationsEnabled() {
		return NotificationDelivery{}
	}
	return NotificationDelivery{Debug: true, Token: token, URL: s.notificationURL(adminPath)}
}

func (s *service) revokeInvitationAfterDeliveryFailure(ctx context.Context, invitation *model.Invitation) error {
	now := s.now()
	invitation.Status = model.StatusRevoked
	invitation.UpdatedAt = now
	return s.repo.SaveInvitation(ctx, invitation)
}

func (s *service) revokePasswordResetAfterDeliveryFailure(ctx context.Context, reset *model.PasswordReset) error {
	now := s.now()
	reset.Status = model.StatusRevoked
	reset.UpdatedAt = now
	return s.repo.SavePasswordReset(ctx, reset)
}

func (s *service) notificationURL(adminPath string) string {
	adminPath = strings.TrimLeft(adminPath, "/")
	base := s.publicBaseURL()
	if base == "" {
		return "/admin/" + adminPath
	}
	return strings.TrimRight(base, "/") + "/" + adminPath
}

func (s *service) debugNotificationsEnabled() bool {
	s.notificationMu.RLock()
	driver := s.notificationDriver
	s.notificationMu.RUnlock()
	switch normalizeCode(driver) {
	case "", "debug", "noop", "local":
		return true
	default:
		return false
	}
}

func (s *service) publicBaseURL() string {
	s.notificationMu.RLock()
	base := s.notificationPublicURL
	s.notificationMu.RUnlock()
	return strings.TrimRight(strings.TrimSpace(base), "/")
}

func (s *service) setupStatus(required bool) SetupStatus {
	return SetupStatus{Required: required, PasswordPolicy: s.passwordPolicy()}
}

func (s *service) passwordPolicy() PasswordPolicy {
	policy := s.cfg.PasswordPolicy
	if policy.MinLength <= 0 {
		policy.MinLength = 8
	}
	return policy
}

// validatePassword 根据当前密码策略校验明文密码。
// 使用 rune 长度处理非 ASCII 密码，避免多字节字符被错误计长。
func (s *service) validatePassword(value string) error {
	policy := s.passwordPolicy()
	if len([]rune(value)) < policy.MinLength {
		return passwordPolicyError(policy)
	}
	var hasLower, hasUpper, hasNumber, hasSymbol bool
	for _, r := range value {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasNumber = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}
	if policy.RequireLower && !hasLower {
		return passwordPolicyError(policy)
	}
	if policy.RequireUpper && !hasUpper {
		return passwordPolicyError(policy)
	}
	if policy.RequireNumber && !hasNumber {
		return passwordPolicyError(policy)
	}
	if policy.RequireSymbol && !hasSymbol {
		return passwordPolicyError(policy)
	}
	return nil
}

// passwordPolicyError 构造包含策略要求的人类可读错误。
func passwordPolicyError(policy PasswordPolicy) error {
	return fmt.Errorf("%w: 密码必须%s", ErrInvalidInput, strings.Join(passwordPolicyRequirements(policy), "、"))
}

// passwordPolicyRequirements 生成当前密码策略的要求列表。
func passwordPolicyRequirements(policy PasswordPolicy) []string {
	if policy.MinLength <= 0 {
		policy.MinLength = 8
	}
	requirements := []string{"至少 " + strconv.Itoa(policy.MinLength) + " 位"}
	if policy.RequireLower {
		requirements = append(requirements, "包含小写字母")
	}
	if policy.RequireUpper {
		requirements = append(requirements, "包含大写字母")
	}
	if policy.RequireNumber {
		requirements = append(requirements, "包含数字")
	}
	if policy.RequireSymbol {
		requirements = append(requirements, "包含符号")
	}
	return requirements
}

// encryptSecret 使用 AES-GCM 加密 MFA secret。
// nonce 会附加在密文前并一起编码，解密时按 GCM nonce 长度拆分。
func (s *service) encryptSecret(secret string) (string, error) {
	block, err := aes.NewCipher(secretKey(s.cfg.MFASecretKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(secret), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

// decryptSecret 解密持久化的 MFA secret。
// 密文长度不足以包含 nonce 时返回 ErrInvalidToken，避免 GCM 打开短数据时报错不清晰。
func (s *service) decryptSecret(value string) (string, error) {
	raw, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(secretKey(s.cfg.MFASecretKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", ErrInvalidToken
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s *service) now() time.Time {
	return s.cfg.Now().UTC()
}

func tokenPair(pair IssuedTokenPair, session *model.Session) TokenPair {
	out := TokenPair{AccessToken: pair.AccessToken, AccessExpiresAt: pair.AccessExpiresAt, RefreshToken: pair.RefreshToken, RefreshExpiresAt: pair.RefreshExpiresAt}
	if session != nil {
		out.UserID = session.UserID
		out.OrgID = session.OrgID
		out.SessionID = session.ID
		out.ProductCode = session.ProductCode
		out.ClientType = session.ClientType
	}
	return out
}

func normalizeCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeProductCode(value string) string {
	return normalizeCode(value)
}

func normalizeClientType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func normalizePermissionScope(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validPermissionScope(value string) bool {
	switch value {
	case model.PermissionScopePlatform, model.PermissionScopeTenant, model.PermissionScopeProduct:
		return true
	default:
		return false
	}
}

func isBuiltinRoleCode(code string) bool {
	switch normalizeCode(code) {
	case model.RolePlatformOwner, model.RoleOwner, model.RoleAdmin, model.RoleMember:
		return true
	default:
		return false
	}
}

func normalizeRegistrationMode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func userSubject(id int64) string {
	return "user:" + strconv.FormatInt(id, 10)
}

func roleSubject(code string) string {
	if strings.HasPrefix(code, "role:") {
		return code
	}
	return "role:" + normalizeCode(code)
}

func permissionObjectAction(code string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(code), ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	return parts[0], parts[1]
}

func permissionAllows(grant RolePermission, required PermissionContext) bool {
	if grant.ProductCode != required.ProductCode {
		return false
	}
	if grant.Scope != required.Scope {
		return false
	}
	permissionObj, permissionAct := permissionObjectAction(grant.Code)
	if permissionObj == "" || permissionAct == "" {
		return false
	}
	return (permissionObj == "*" || permissionObj == required.Object) && (permissionAct == "*" || permissionAct == required.Action)
}

func (s *service) hydrateRolePermissions(ctx context.Context, role *model.Role) error {
	if role == nil {
		return nil
	}
	permissions, err := s.repo.ListRolePermissions(ctx, role.OrgID, roleSubject(role.Code))
	if err != nil {
		return err
	}
	role.Permissions = make([]string, 0, len(permissions))
	for _, permission := range permissions {
		role.Permissions = append(role.Permissions, permission.Code)
	}
	return nil
}

// secretKey 将配置密钥派生为 AES-256 长度。
func secretKey(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}

// permissionSeed 描述内置权限种子。
type permissionSeed struct {
	Code        string
	Scope       string
	Name        string
	Description string
}

// builtinPermissions 是新组织初始化时写入的基础权限集合。
var builtinPermissions = []permissionSeed{
	{Code: "org:create", Scope: model.PermissionScopePlatform, Name: "Create organizations", Description: "Create organizations"},
	{Code: "org:read", Scope: model.PermissionScopePlatform, Name: "Read organizations", Description: "Read organizations"},
	{Code: "org:update", Scope: model.PermissionScopeTenant, Name: "Update organizations", Description: "Update organization settings"},
	{Code: "user:read", Scope: model.PermissionScopeTenant, Name: "Read users", Description: "Read organization users"},
	{Code: "user:invite", Scope: model.PermissionScopeTenant, Name: "Invite users", Description: "Invite users into organization"},
	{Code: "user:update", Scope: model.PermissionScopeTenant, Name: "Update users", Description: "Update organization users"},
	{Code: "user:disable", Scope: model.PermissionScopeTenant, Name: "Disable users", Description: "Disable organization users"},
	{Code: "role:read", Scope: model.PermissionScopeTenant, Name: "Read roles", Description: "Read roles"},
	{Code: "role:create", Scope: model.PermissionScopeTenant, Name: "Create roles", Description: "Create roles"},
	{Code: "role:update", Scope: model.PermissionScopeTenant, Name: "Update roles", Description: "Update roles"},
	{Code: "config:read", Scope: model.PermissionScopePlatform, Name: "Read runtime config", Description: "Read sanitized system runtime configuration"},
	{Code: "config:update", Scope: model.PermissionScopePlatform, Name: "Update runtime config", Description: "Update current runtime configuration snapshot"},
	{Code: "server:read", Scope: model.PermissionScopePlatform, Name: "Read server info", Description: "Read runtime server information"},
	{Code: "dictionary:read", Scope: model.PermissionScopePlatform, Name: "Read dictionaries", Description: "Read system dictionaries"},
	{Code: "dictionary:create", Scope: model.PermissionScopePlatform, Name: "Create dictionaries", Description: "Create system dictionaries"},
	{Code: "dictionary:update", Scope: model.PermissionScopePlatform, Name: "Update dictionaries", Description: "Update system dictionaries and items"},
	{Code: "dictionary:delete", Scope: model.PermissionScopePlatform, Name: "Delete dictionaries", Description: "Delete system dictionaries and items"},
	{Code: "operation:read", Scope: model.PermissionScopePlatform, Name: "Read operation records", Description: "Read HTTP operation records"},
	{Code: "operation:delete", Scope: model.PermissionScopePlatform, Name: "Delete operation records", Description: "Delete HTTP operation records"},
	{Code: "traffic_hijack:read", Scope: model.PermissionScopePlatform, Name: "Read traffic hijack monitoring", Description: "Read traffic hijack probe targets, results, and events"},
	{Code: "traffic_hijack:update", Scope: model.PermissionScopePlatform, Name: "Update traffic hijack monitoring", Description: "Create, update, probe, and resolve traffic hijack monitoring records"},
	{Code: "traffic_hijack:delete", Scope: model.PermissionScopePlatform, Name: "Delete traffic hijack monitoring", Description: "Delete traffic hijack probe targets"},
	{Code: "parameter:read", Scope: model.PermissionScopePlatform, Name: "Read parameters", Description: "Read system parameters"},
	{Code: "parameter:create", Scope: model.PermissionScopePlatform, Name: "Create parameters", Description: "Create system parameters"},
	{Code: "parameter:update", Scope: model.PermissionScopePlatform, Name: "Update parameters", Description: "Update system parameters"},
	{Code: "parameter:delete", Scope: model.PermissionScopePlatform, Name: "Delete parameters", Description: "Delete system parameters"},
	{Code: "version:read", Scope: model.PermissionScopePlatform, Name: "Read versions", Description: "Read system release packages"},
	{Code: "version:create", Scope: model.PermissionScopePlatform, Name: "Create versions", Description: "Create system release packages"},
	{Code: "version:import", Scope: model.PermissionScopePlatform, Name: "Import versions", Description: "Import system release packages"},
	{Code: "version:download", Scope: model.PermissionScopePlatform, Name: "Download versions", Description: "Download system release packages"},
	{Code: "version:delete", Scope: model.PermissionScopePlatform, Name: "Delete versions", Description: "Delete system release packages"},
	{Code: "media:read", Scope: model.PermissionScopePlatform, Name: "Read media", Description: "Read media assets and categories"},
	{Code: "media:upload", Scope: model.PermissionScopePlatform, Name: "Upload media", Description: "Upload local media assets"},
	{Code: "media:import", Scope: model.PermissionScopePlatform, Name: "Import media URLs", Description: "Import external media URL records"},
	{Code: "media:update", Scope: model.PermissionScopePlatform, Name: "Update media", Description: "Update media assets and categories"},
	{Code: "media:download", Scope: model.PermissionScopePlatform, Name: "Download media", Description: "Download local media assets"},
	{Code: "media:delete", Scope: model.PermissionScopePlatform, Name: "Delete media", Description: "Delete media assets"},
	{Code: "permission:read", Scope: model.PermissionScopeTenant, Name: "Read tenant permissions", Description: "Read tenant permissions"},
	{Code: "permission:read", Scope: model.PermissionScopePlatform, Name: "Read platform permissions", Description: "Read platform API and permission catalog"},
	{Code: "permission:sync", Scope: model.PermissionScopePlatform, Name: "Sync permissions", Description: "Sync permissions from registered APIs"},
	{Code: "session:read", Scope: model.PermissionScopeTenant, Name: "Read sessions", Description: "Read sessions"},
	{Code: "session:revoke", Scope: model.PermissionScopeTenant, Name: "Revoke sessions", Description: "Revoke sessions"},
	{Code: "api_token:read", Scope: model.PermissionScopeTenant, Name: "Read API tokens", Description: "Read API token records"},
	{Code: "api_token:create", Scope: model.PermissionScopeTenant, Name: "Create API tokens", Description: "Issue API tokens"},
	{Code: "api_token:revoke", Scope: model.PermissionScopeTenant, Name: "Revoke API tokens", Description: "Revoke API tokens"},
	{Code: "audit:read", Scope: model.PermissionScopeTenant, Name: "Read audit logs", Description: "Read audit logs"},
	{Code: "plugin:read", Scope: model.PermissionScopePlatform, Name: "Read plugins", Description: "Read registered remote plugins"},
}

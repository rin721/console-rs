package model

import "time"

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusPending  = "pending"
	StatusUsed     = "used"
	StatusRevoked  = "revoked"
	StatusExpired  = "expired"

	RolePlatformOwner = "platform_owner"
	RoleOwner         = "owner"
	RoleAdmin         = "admin"
	RoleMember        = "member"

	OrgKindPlatform = "platform"
	OrgKindTenant   = "tenant"

	PermissionScopePlatform = "platform"
	PermissionScopeTenant   = "tenant"
	PermissionScopeProduct  = "product"
)

type Organization struct {
	ID        int64      `gorm:"column:id;primaryKey" json:"id,string"`
	Code      string     `gorm:"column:code;size:64;not null;uniqueIndex" json:"code"`
	Name      string     `gorm:"column:name;size:128;not null" json:"name"`
	Kind      string     `gorm:"column:kind;size:32;not null;default:tenant;index" json:"kind"`
	Status    string     `gorm:"column:status;size:32;not null" json:"status"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt time.Time  `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt *time.Time `gorm:"column:deleted_at" json:"-"`
}

func (Organization) TableName() string { return "iam_organizations" }

type User struct {
	ID                  int64      `gorm:"column:id;primaryKey" json:"id,string"`
	Username            string     `gorm:"column:username;size:64;not null;uniqueIndex" json:"username"`
	Email               string     `gorm:"column:email;size:255;not null;uniqueIndex" json:"email"`
	PasswordHash        string     `gorm:"column:password_hash;size:255;not null" json:"-"`
	DisplayName         string     `gorm:"column:display_name;size:128;not null" json:"displayName"`
	Status              string     `gorm:"column:status;size:32;not null" json:"status"`
	MFAEnabled          bool       `gorm:"column:mfa_enabled;not null;default:false" json:"mfaEnabled"`
	FailedLoginAttempts int        `gorm:"column:failed_login_attempts;not null;default:0" json:"-"`
	LockedUntil         *time.Time `gorm:"column:locked_until" json:"lockedUntil,omitempty"`
	LastLoginAt         *time.Time `gorm:"column:last_login_at" json:"lastLoginAt,omitempty"`
	CreatedAt           time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt           time.Time  `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt           *time.Time `gorm:"column:deleted_at" json:"-"`
}

func (User) TableName() string { return "iam_users" }

type Membership struct {
	ID        int64      `gorm:"column:id;primaryKey" json:"id,string"`
	OrgID     int64      `gorm:"column:org_id;not null;uniqueIndex:uk_iam_membership_org_user" json:"orgId,string"`
	UserID    int64      `gorm:"column:user_id;not null;uniqueIndex:uk_iam_membership_org_user" json:"userId,string"`
	Status    string     `gorm:"column:status;size:32;not null" json:"status"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt time.Time  `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt *time.Time `gorm:"column:deleted_at" json:"-"`
}

func (Membership) TableName() string { return "iam_memberships" }

type Role struct {
	ID          int64      `gorm:"column:id;primaryKey" json:"id,string"`
	OrgID       int64      `gorm:"column:org_id;not null;uniqueIndex:uk_iam_role_org_code" json:"orgId,string"`
	Code        string     `gorm:"column:code;size:64;not null;uniqueIndex:uk_iam_role_org_code" json:"code"`
	Name        string     `gorm:"column:name;size:128;not null" json:"name"`
	Description string     `gorm:"column:description;not null" json:"description"`
	System      bool       `gorm:"column:system;not null;default:false" json:"system"`
	Permissions []string   `gorm:"-" json:"permissions,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   time.Time  `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt   *time.Time `gorm:"column:deleted_at" json:"-"`
}

func (Role) TableName() string { return "iam_roles" }

type Permission struct {
	ID          int64     `gorm:"column:id;primaryKey" json:"id,string"`
	ProductCode string    `gorm:"column:product_code;size:64;not null;uniqueIndex:uk_iam_permission_product_scope_code" json:"productCode"`
	Scope       string    `gorm:"column:scope;size:32;not null;uniqueIndex:uk_iam_permission_product_scope_code;index" json:"scope"`
	Code        string    `gorm:"column:code;size:128;not null;uniqueIndex:uk_iam_permission_product_scope_code" json:"code"`
	Name        string    `gorm:"column:name;size:128;not null" json:"name"`
	Description string    `gorm:"column:description;not null" json:"description"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (Permission) TableName() string { return "iam_permissions" }

type Session struct {
	ID               int64      `gorm:"column:id;primaryKey" json:"id,string"`
	UserID           int64      `gorm:"column:user_id;not null" json:"userId,string"`
	OrgID            int64      `gorm:"column:org_id;not null" json:"orgId,string"`
	ProductCode      string     `gorm:"column:product_code;size:64;not null;index:idx_iam_sessions_scope" json:"productCode"`
	ClientType       string     `gorm:"column:client_type;size:32;not null;index:idx_iam_sessions_scope" json:"clientType"`
	RefreshTokenHash string     `gorm:"column:refresh_token_hash;size:128;not null;uniqueIndex" json:"-"`
	UserAgent        string     `gorm:"column:user_agent;not null" json:"userAgent"`
	IPAddress        string     `gorm:"column:ip_address;size:64;not null" json:"ipAddress"`
	ExpiresAt        time.Time  `gorm:"column:expires_at" json:"expiresAt"`
	RevokedAt        *time.Time `gorm:"column:revoked_at" json:"revokedAt,omitempty"`
	LastUsedAt       *time.Time `gorm:"column:last_used_at" json:"lastUsedAt,omitempty"`
	CreatedAt        time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt        time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}

func (Session) TableName() string { return "iam_sessions" }

type APIToken struct {
	ID                int64      `gorm:"column:id;primaryKey" json:"id,string"`
	OrgID             int64      `gorm:"column:org_id;not null" json:"orgId,string"`
	UserID            int64      `gorm:"column:user_id;not null" json:"userId,string"`
	RoleCode          string     `gorm:"column:role_code;size:64;not null" json:"roleCode"`
	TokenPrefix       string     `gorm:"column:token_prefix;size:32;not null" json:"tokenPrefix"`
	TokenHash         string     `gorm:"column:token_hash;size:128;not null;uniqueIndex" json:"-"`
	Status            string     `gorm:"column:status;size:32;not null" json:"status"`
	ExpiresAt         *time.Time `gorm:"column:expires_at" json:"expiresAt,omitempty"`
	LastUsedAt        *time.Time `gorm:"column:last_used_at" json:"lastUsedAt,omitempty"`
	LastUsedIPAddress string     `gorm:"column:last_used_ip_address;size:64;not null" json:"lastUsedIpAddress"`
	RevokedAt         *time.Time `gorm:"column:revoked_at" json:"revokedAt,omitempty"`
	RevokedBy         *int64     `gorm:"column:revoked_by" json:"revokedBy,omitempty,string"`
	Remark            string     `gorm:"column:remark;not null" json:"remark"`
	CreatedBy         int64      `gorm:"column:created_by;not null" json:"createdBy,string"`
	CreatedAt         time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt         time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}

func (APIToken) TableName() string { return "iam_api_tokens" }

type Invitation struct {
	ID         int64     `gorm:"column:id;primaryKey" json:"id,string"`
	OrgID      int64     `gorm:"column:org_id;not null" json:"orgId,string"`
	Email      string    `gorm:"column:email;size:255;not null" json:"email"`
	RoleCode   string    `gorm:"column:role_code;size:64;not null" json:"roleCode"`
	TokenHash  string    `gorm:"column:token_hash;size:128;not null;uniqueIndex" json:"-"`
	Status     string    `gorm:"column:status;size:32;not null" json:"status"`
	InvitedBy  int64     `gorm:"column:invited_by;not null" json:"invitedBy,string"`
	AcceptedBy *int64    `gorm:"column:accepted_by" json:"acceptedBy,omitempty,string"`
	ExpiresAt  time.Time `gorm:"column:expires_at" json:"expiresAt"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (Invitation) TableName() string { return "iam_invitations" }

type PasswordReset struct {
	ID        int64      `gorm:"column:id;primaryKey" json:"id,string"`
	UserID    int64      `gorm:"column:user_id;not null" json:"userId,string"`
	TokenHash string     `gorm:"column:token_hash;size:128;not null;uniqueIndex" json:"-"`
	Status    string     `gorm:"column:status;size:32;not null" json:"status"`
	ExpiresAt time.Time  `gorm:"column:expires_at" json:"expiresAt"`
	UsedAt    *time.Time `gorm:"column:used_at" json:"usedAt,omitempty"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}

func (PasswordReset) TableName() string { return "iam_password_resets" }

type EmailVerification struct {
	ID         int64      `gorm:"column:id;primaryKey" json:"id,string"`
	UserID     int64      `gorm:"column:user_id;not null;index" json:"userId,string"`
	OrgID      int64      `gorm:"column:org_id;not null;index" json:"orgId,string"`
	TokenHash  string     `gorm:"column:token_hash;size:128;not null;uniqueIndex" json:"-"`
	Status     string     `gorm:"column:status;size:32;not null;index" json:"status"`
	ExpiresAt  time.Time  `gorm:"column:expires_at;not null" json:"expiresAt"`
	VerifiedAt *time.Time `gorm:"column:verified_at" json:"verifiedAt,omitempty"`
	CreatedAt  time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt  time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}

func (EmailVerification) TableName() string { return "iam_email_verifications" }

type MFAFactor struct {
	ID          int64      `gorm:"column:id;primaryKey" json:"id,string"`
	UserID      int64      `gorm:"column:user_id;not null" json:"userId,string"`
	Type        string     `gorm:"column:type;size:32;not null" json:"type"`
	Secret      string     `gorm:"column:secret;not null" json:"-"`
	Status      string     `gorm:"column:status;size:32;not null" json:"status"`
	ConfirmedAt *time.Time `gorm:"column:confirmed_at" json:"confirmedAt,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}

func (MFAFactor) TableName() string { return "iam_mfa_factors" }

type AuditLog struct {
	ID          int64     `gorm:"column:id;primaryKey" json:"id,string"`
	OrgID       *int64    `gorm:"column:org_id" json:"orgId,omitempty,string"`
	UserID      *int64    `gorm:"column:user_id" json:"userId,omitempty,string"`
	ProductCode string    `gorm:"column:product_code;size:64;not null;index" json:"productCode"`
	ClientType  string    `gorm:"column:client_type;size:32;not null;index" json:"clientType"`
	Action      string    `gorm:"column:action;size:128;not null" json:"action"`
	Resource    string    `gorm:"column:resource;size:128;not null" json:"resource"`
	ResourceID  string    `gorm:"column:resource_id;size:128;not null" json:"resourceId"`
	IPAddress   string    `gorm:"column:ip_address;size:64;not null" json:"ipAddress"`
	UserAgent   string    `gorm:"column:user_agent;not null" json:"userAgent"`
	Metadata    string    `gorm:"column:metadata;not null" json:"metadata"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"createdAt"`
}

func (AuditLog) TableName() string { return "iam_audit_logs" }

type CasbinRule struct {
	ID        int64     `gorm:"column:id;primaryKey" json:"id,string"`
	PType     string    `gorm:"column:ptype;size:8;not null;uniqueIndex:uk_iam_casbin_rule" json:"ptype"`
	V0        string    `gorm:"column:v0;size:128;not null;uniqueIndex:uk_iam_casbin_rule" json:"v0"`
	V1        string    `gorm:"column:v1;size:128;not null;uniqueIndex:uk_iam_casbin_rule" json:"v1"`
	V2        string    `gorm:"column:v2;size:256;not null;uniqueIndex:uk_iam_casbin_rule" json:"v2"`
	V3        string    `gorm:"column:v3;size:256;not null;uniqueIndex:uk_iam_casbin_rule" json:"v3"`
	V4        string    `gorm:"column:v4;size:256;not null;uniqueIndex:uk_iam_casbin_rule" json:"v4"`
	V5        string    `gorm:"column:v5;size:256;not null;uniqueIndex:uk_iam_casbin_rule" json:"v5"`
	CreatedAt time.Time `gorm:"column:created_at" json:"createdAt"`
}

func (CasbinRule) TableName() string { return "iam_casbin_rules" }

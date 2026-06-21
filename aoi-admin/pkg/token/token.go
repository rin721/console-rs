// Package token 将 JWT 签发校验与 refresh token 哈希细节封装在项目自有 API 后面。
package token

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// TokenTypeAccess 和 TokenTypeRefresh 写入 claims，用于解析阶段区分令牌用途。
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var (
	// ErrInvalidConfig 表示创建 Manager 时发现必要配置缺失或不满足安全下限。
	ErrInvalidConfig = errors.New("invalid token config")
	// ErrInvalidToken 表示 JWT 签名、registered claims 或业务 claims 校验失败。
	ErrInvalidToken = errors.New("invalid token")
	// ErrWrongType 表示令牌有效但不符合调用方期望的 token type。
	ErrWrongType = errors.New("wrong token type")
)

// Config 描述 token 管理器运行所需的签发者、受众、密钥和过期策略。
type Config struct {
	// Issuer 写入 JWT registered claims，并在解析时作为可信签发者校验。
	Issuer string
	// Audience 非空时会同时写入和校验 JWT audience。
	Audience []string
	// SigningKey 用于 HS256 签名，长度限制用于避免过短密钥降低签名强度。
	SigningKey string
	// AccessTTL 和 RefreshTTL 分别控制访问令牌与刷新令牌的有效期。
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	// RefreshPepper 参与 refresh token HMAC，避免数据库中的哈希值可被离线直接验证。
	RefreshPepper string
	// Now 允许测试注入稳定时间；为空时使用 time.Now。
	Now func() time.Time
}

// Subject 是访问令牌内承载的身份上下文。
type Subject struct {
	UserID      int64
	OrgID       int64
	SessionID   int64
	ProductCode string
}

// Claims 扩展标准 JWT claims，附加项目需要的用户、组织、会话和令牌类型。
type Claims struct {
	UserID      int64  `json:"userId,string"`
	OrgID       int64  `json:"orgId,string"`
	SessionID   int64  `json:"sessionId,string"`
	ProductCode string `json:"productCode"`
	TokenType   string `json:"tokenType"`
	jwt.RegisteredClaims
}

// Pair 是一次登录或刷新流程返回给调用方的访问/刷新令牌组合及其过期时间。
type Pair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshTokenHash string
	RefreshExpiresAt time.Time
}

// Manager 定义令牌签发、解析和 refresh token 哈希能力。
type Manager interface {
	// IssueAccess 为指定主体签发短期访问令牌，并返回令牌过期时间。
	IssueAccess(context.Context, Subject) (string, time.Time, error)
	// IssueRefresh 生成不可预测的 refresh token，并返回可持久化的 HMAC 哈希。
	IssueRefresh(context.Context) (string, string, time.Time, error)
	// IssuePair 一次性签发访问令牌和刷新令牌，保证两者使用同一份配置策略。
	IssuePair(context.Context, Subject) (Pair, error)
	// Parse 校验 JWT 签名、registered claims 和业务 claims，可按 expectedType 限定令牌类型。
	Parse(context.Context, string, string) (*Claims, error)
	// HashRefreshToken 将 refresh token 转换为数据库可保存的不可逆摘要。
	HashRefreshToken(string) string
}

type manager struct {
	cfg Config
}

func New(cfg Config) (Manager, error) {
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("%w: issuer is required", ErrInvalidConfig)
	}
	if cfg.SigningKey == "" {
		return nil, fmt.Errorf("%w: signing key is required", ErrInvalidConfig)
	}
	if len(cfg.SigningKey) < 32 {
		return nil, fmt.Errorf("%w: signing key must be at least 32 bytes", ErrInvalidConfig)
	}
	if cfg.AccessTTL <= 0 {
		return nil, fmt.Errorf("%w: access ttl must be positive", ErrInvalidConfig)
	}
	if cfg.RefreshTTL <= 0 {
		return nil, fmt.Errorf("%w: refresh ttl must be positive", ErrInvalidConfig)
	}
	if cfg.RefreshPepper == "" {
		return nil, fmt.Errorf("%w: refresh pepper is required", ErrInvalidConfig)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &manager{cfg: cfg}, nil
}

// IssueAccess 使用 HS256 签发访问令牌；调用方负责保证 subject 已通过业务层校验。
func (m *manager) IssueAccess(_ context.Context, subject Subject) (string, time.Time, error) {
	now := m.cfg.Now().UTC()
	expiresAt := now.Add(m.cfg.AccessTTL)
	claims := Claims{
		UserID:      subject.UserID,
		OrgID:       subject.OrgID,
		SessionID:   subject.SessionID,
		ProductCode: subject.ProductCode,
		TokenType:   TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.cfg.Issuer,
			Audience:  jwt.ClaimStrings(m.cfg.Audience),
			Subject:   fmt.Sprintf("%d", subject.UserID),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(m.cfg.SigningKey))
	if err != nil {
		return "", time.Time{}, err
	}
	return raw, expiresAt, nil
}

// IssueRefresh 使用 crypto/rand 生成随机令牌，明文只返回给客户端，服务端保存哈希值。
func (m *manager) IssueRefresh(_ context.Context) (string, string, time.Time, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, m.HashRefreshToken(token), m.cfg.Now().UTC().Add(m.cfg.RefreshTTL), nil
}

// IssuePair 组合签发流程，避免调用方遗漏 refresh token 的哈希和过期时间。
func (m *manager) IssuePair(ctx context.Context, subject Subject) (Pair, error) {
	access, accessExpiresAt, err := m.IssueAccess(ctx, subject)
	if err != nil {
		return Pair{}, err
	}
	refresh, refreshHash, refreshExpiresAt, err := m.IssueRefresh(ctx)
	if err != nil {
		return Pair{}, err
	}
	return Pair{
		AccessToken:      access,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refresh,
		RefreshTokenHash: refreshHash,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

// Parse 解析并校验访问令牌；expectedType 为空时只校验签名和基础业务 claims。
func (m *manager) Parse(_ context.Context, raw string, expectedType string) (*Claims, error) {
	claims := &Claims{}
	opts := []jwt.ParserOption{
		jwt.WithIssuer(m.cfg.Issuer),
	}
	if len(m.cfg.Audience) > 0 {
		opts = append(opts, jwt.WithAudience(m.cfg.Audience...))
	}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		// 固定 HS256 可避免攻击者通过 alg header 切换到非预期签名算法。
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return []byte(m.cfg.SigningKey), nil
	}, opts...)
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if expectedType != "" && claims.TokenType != expectedType {
		return nil, ErrWrongType
	}
	if claims.UserID <= 0 || claims.SessionID <= 0 || claims.ProductCode == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// HashRefreshToken 使用带 pepper 的 HMAC 保存 refresh token，避免持久化明文凭据。
func (m *manager) HashRefreshToken(raw string) string {
	mac := hmac.New(sha256.New, []byte(m.cfg.RefreshPepper))
	_, _ = mac.Write([]byte(raw))
	return hex.EncodeToString(mac.Sum(nil))
}

// Package handler 将 IAM HTTP 请求转换为服务层输入，并统一处理响应和错误映射。
package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rei0721/go-scaffold/internal/middleware"
	"github.com/rei0721/go-scaffold/internal/modules/iam/service"
	"github.com/rei0721/go-scaffold/internal/ports"
	"github.com/rei0721/go-scaffold/types/result"
)

// Handler 是 IAM 模块的 HTTP 适配器。
// service 处理普通 IAM API；setupService 可单独替换首次初始化后端，便于部署场景隔离 setup 权限。
type Handler struct {
	service      service.Service
	setupService setupService
	logger       ports.Logger
	config       RuntimeConfig
}

// setupService 抽象首次初始化需要的最小服务能力。
// 它让初始化接口可以使用专门的服务实例，而不影响普通 IAM API。
type setupService interface {
	SetupStatus(context.Context) (service.SetupStatus, error)
	InitialAdminSetup(context.Context, service.InitialAdminSetupInput) (service.TokenPair, error)
}

// New 创建 IAM HTTP handler。
type RuntimeConfig struct {
	CookieNamePrefix     string
	CookieDomain         string
	CookiePath           string
	CookieSameSite       string
	CookieSecure         bool
	CSRFEnabled          bool
	CSRFCookieName       string
	CSRFHeaderName       string
	ProductHeader        string
	ClientTypeHeader     string
	DefaultProductCode   string
	DefaultClientType    string
	MobileUserAgentHints []string
}

func New(service service.Service, logger ports.Logger, configs ...RuntimeConfig) *Handler {
	cfg := RuntimeConfig{}
	if len(configs) > 0 {
		cfg = configs[0]
	}
	cfg.applyDefaults()
	return &Handler{service: service, setupService: iamSetupService{service: service}, logger: logger, config: cfg}
}

// UseSetupService 替换首次初始化专用后端，普通 IAM API 仍保持原服务实例。
func (cfg *RuntimeConfig) applyDefaults() {
	if strings.TrimSpace(cfg.CookieNamePrefix) == "" {
		cfg.CookieNamePrefix = "aoi"
	}
	if strings.TrimSpace(cfg.CookiePath) == "" {
		cfg.CookiePath = "/"
	}
	if strings.TrimSpace(cfg.CookieSameSite) == "" {
		cfg.CookieSameSite = "lax"
	}
	if strings.TrimSpace(cfg.CSRFCookieName) == "" {
		cfg.CSRFCookieName = cfg.CookieNamePrefix + "_csrf"
	}
	if strings.TrimSpace(cfg.CSRFHeaderName) == "" {
		cfg.CSRFHeaderName = "X-CSRF-Token"
	}
	if strings.TrimSpace(cfg.ProductHeader) == "" {
		cfg.ProductHeader = "X-Aoi-Product-Code"
	}
	if strings.TrimSpace(cfg.ClientTypeHeader) == "" {
		cfg.ClientTypeHeader = "X-Aoi-Client-Type"
	}
	if strings.TrimSpace(cfg.DefaultProductCode) == "" {
		cfg.DefaultProductCode = "platform"
	}
	if strings.TrimSpace(cfg.DefaultClientType) == "" {
		cfg.DefaultClientType = "pc_web"
	}
	if len(cfg.MobileUserAgentHints) == 0 {
		cfg.MobileUserAgentHints = []string{"mobile", "android", "iphone", "ipad"}
	}
}

func (cfg RuntimeConfig) AccessCookieName() string {
	return strings.TrimSpace(cfg.CookieNamePrefix) + "_access"
}

func (cfg RuntimeConfig) RefreshCookieName() string {
	return strings.TrimSpace(cfg.CookieNamePrefix) + "_refresh"
}

func (cfg RuntimeConfig) CSRFMiddlewareConfig() middleware.CSRFConfig {
	return middleware.CSRFConfig{Enabled: cfg.CSRFEnabled, CookieName: cfg.CSRFCookieName, HeaderName: cfg.CSRFHeaderName}
}

func (cfg RuntimeConfig) AuthMiddlewareConfig() middleware.AuthConfig {
	return middleware.AuthConfig{AccessCookieName: cfg.AccessCookieName()}
}

func (h *Handler) AuthMiddlewareConfig() middleware.AuthConfig {
	return h.config.AuthMiddlewareConfig()
}

func (h *Handler) CSRFMiddlewareConfig() middleware.CSRFConfig {
	return h.config.CSRFMiddlewareConfig()
}

func (h *Handler) requestSessionContext(c ports.HTTPContext) (string, string) {
	productCode := strings.TrimSpace(c.GetHeader(h.config.ProductHeader))
	if productCode == "" {
		productCode = h.config.DefaultProductCode
	}
	clientType := strings.TrimSpace(c.GetHeader(h.config.ClientTypeHeader))
	if clientType == "" {
		clientType = h.clientTypeFromUserAgent(c.GetHeader("User-Agent"))
	}
	return productCode, clientType
}

func (h *Handler) clientTypeFromUserAgent(userAgent string) string {
	ua := strings.ToLower(userAgent)
	for _, hint := range h.config.MobileUserAgentHints {
		if hint = strings.ToLower(strings.TrimSpace(hint)); hint != "" && strings.Contains(ua, hint) {
			return "mobile_web"
		}
	}
	return h.config.DefaultClientType
}

func (h *Handler) UseSetupService(setup setupService) {
	if setup != nil {
		h.setupService = setup
	}
}

// iamSetupService 把普通 IAM service 适配为 setupService。
type iamSetupService struct {
	service service.Service
}

// SetupStatus 转发首次初始化状态查询。
func (s iamSetupService) SetupStatus(ctx context.Context) (service.SetupStatus, error) {
	return s.service.SetupStatus(ctx)
}

// InitialAdminSetup 转发首次管理员初始化请求。
func (s iamSetupService) InitialAdminSetup(ctx context.Context, input service.InitialAdminSetupInput) (service.TokenPair, error) {
	return s.service.InitialAdminSetup(ctx, input)
}

// LoginRequest 是登录接口的 JSON 请求体。
// Captcha 和 MFA 字段由服务层按配置和用户状态决定是否必需。
type LoginRequest struct {
	CaptchaCode string `json:"captchaCode"`
	CaptchaID   string `json:"captchaId"`
	Identifier  string `json:"identifier" binding:"required"`
	Password    string `json:"password" binding:"required"`
	OrgCode     string `json:"orgCode"`
	MFACode     string `json:"mfaCode"`
}

// SignupRequest 是自助注册和首次管理员初始化共用的请求体。
type SignupRequest struct {
	OrgCode     string `json:"orgCode" binding:"required"`
	OrgName     string `json:"orgName" binding:"required"`
	Username    string `json:"username" binding:"required"`
	Email       string `json:"email" binding:"required"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password" binding:"required"`
}

// InitialAdminSetupRequest 复用注册字段，保持 setup 接口与注册接口的输入一致。
type InitialAdminSetupRequest = SignupRequest

// RefreshRequest 是 refresh token 换发请求体。
type RefreshRequest struct {
}

// SwitchOrgRequest 是切换组织请求体。
type SwitchOrgRequest struct {
	OrgID int64String `json:"orgId" binding:"required"`
}

// int64String 兼容前端以字符串或数字传递 int64 ID。
// Go 的 JSON number 对大整数不友好，字符串形式可以避免精度丢失。
type int64String int64

// UnmarshalJSON 将 JSON 字符串或数字解析为 int64。
func (v *int64String) UnmarshalJSON(raw []byte) error {
	text := string(raw)
	if unquoted, err := strconv.Unquote(text); err == nil {
		text = unquoted
	}
	id, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return err
	}
	*v = int64String(id)
	return nil
}

type CreateOrgRequest struct {
	Code string `json:"code" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type UpdateOrgRequest struct {
	Name string `json:"name" binding:"required"`
}

type InviteUserRequest struct {
	Email    string `json:"email" binding:"required"`
	RoleCode string `json:"roleCode" binding:"required"`
}

type AcceptInvitationRequest struct {
	Username    string `json:"username" binding:"required"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

type VerifyMFARequest struct {
	Code string `json:"code" binding:"required"`
}

type CreateRoleRequest struct {
	Code        string   `json:"code" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// UpdateUserRequest 使用指针区分“字段未提交”和“提交空值”。
type UpdateUserRequest struct {
	Status *string   `json:"status"`
	Roles  *[]string `json:"roles"`
}

type CreateAPITokenRequest struct {
	UserID   int64String `json:"userId" binding:"required"`
	RoleCode string      `json:"roleCode" binding:"required"`
	Days     int         `json:"days"`
	Remark   string      `json:"remark"`
}

// UpdateRoleRequest 使用指针区分“不更新权限”和“清空权限集合”。
type UpdateRoleRequest struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions *[]string `json:"permissions"`
}

func (h *Handler) writeAuthResult(c ports.HTTPContext, pair service.TokenPair, err error) {
	if err != nil {
		h.write(c, nil, err)
		return
	}
	h.setAuthCookies(c, pair)
	result.OK(c, pair.SessionSnapshot())
}

func (h *Handler) setAuthCookies(c ports.HTTPContext, pair service.TokenPair) {
	h.setCookie(c, h.config.AccessCookieName(), pair.AccessToken, pair.AccessExpiresAt, true)
	h.setCookie(c, h.config.RefreshCookieName(), pair.RefreshToken, pair.RefreshExpiresAt, true)
	if h.config.CSRFEnabled {
		h.setCookie(c, h.config.CSRFCookieName, newCSRFToken(), pair.RefreshExpiresAt, false)
	}
}

func (h *Handler) clearAuthCookies(c ports.HTTPContext) {
	expired := time.Unix(0, 0).UTC()
	h.setCookie(c, h.config.AccessCookieName(), "", expired, true)
	h.setCookie(c, h.config.RefreshCookieName(), "", expired, true)
	if h.config.CSRFEnabled {
		h.setCookie(c, h.config.CSRFCookieName, "", expired, false)
	}
}

func (h *Handler) setCookie(c ports.HTTPContext, name string, value string, expires time.Time, httpOnly bool) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     h.config.CookiePath,
		Domain:   strings.TrimSpace(h.config.CookieDomain),
		Expires:  expires,
		HttpOnly: httpOnly,
		Secure:   h.config.CookieSecure,
		SameSite: cookieSameSite(h.config.CookieSameSite),
	}
	if value == "" || expires.Before(time.Now()) {
		cookie.MaxAge = -1
	} else if seconds := int(time.Until(expires).Seconds()); seconds > 0 {
		cookie.MaxAge = seconds
	}
	c.SetCookie(cookie)
}

func (h *Handler) refreshTokenFromCookie(c ports.HTTPContext) (string, bool) {
	token, err := c.Cookie(h.config.RefreshCookieName())
	if err != nil || strings.TrimSpace(token) == "" {
		result.Unauthorized(c, result.MessageKeyUnauthorized)
		return "", false
	}
	return strings.TrimSpace(token), true
}

func (h *Handler) validateCSRF(c ports.HTTPContext) bool {
	if !h.config.CSRFEnabled {
		return true
	}
	cookieValue, err := c.Cookie(h.config.CSRFCookieName)
	if err != nil || strings.TrimSpace(cookieValue) == "" {
		result.Forbidden(c, result.MessageKeyForbidden)
		return false
	}
	if constantTimeEqual(cookieValue, c.GetHeader(h.config.CSRFHeaderName)) {
		return true
	}
	result.Forbidden(c, result.MessageKeyForbidden)
	return false
}

func cookieSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func newCSRFToken() string {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func constantTimeEqual(left string, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || len(left) != len(right) {
		return false
	}
	var diff byte
	for i := 0; i < len(left); i++ {
		diff |= left[i] ^ right[i]
	}
	return diff == 0
}

func (h *Handler) Signup(c ports.HTTPContext) {
	var req SignupRequest
	if !bind(c, &req) {
		return
	}
	productCode, clientType := h.requestSessionContext(c)
	signupResult, err := h.service.Signup(c.RequestContext(), service.SignupInput{
		OrgCode:     req.OrgCode,
		OrgName:     req.OrgName,
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Password:    req.Password,
		ProductCode: productCode,
		ClientType:  clientType,
		UserAgent:   c.GetHeader("User-Agent"),
		IPAddress:   c.ClientIP(),
	})
	if err != nil {
		h.write(c, nil, err)
		return
	}
	if signupResult.Tokens.AccessToken != "" {
		h.setAuthCookies(c, signupResult.Tokens)
	}
	result.OK(c, signupResult)
}

func (h *Handler) ConfirmEmailVerification(c ports.HTTPContext) {
	productCode, clientType := h.requestSessionContext(c)
	pair, err := h.service.ConfirmEmailVerification(c.RequestContext(), service.ConfirmEmailVerificationInput{
		Token:       c.Param("token"),
		ProductCode: productCode,
		ClientType:  clientType,
		UserAgent:   c.GetHeader("User-Agent"),
		IPAddress:   c.ClientIP(),
	})
	h.writeAuthResult(c, pair, err)
}

func (h *Handler) SetupStatus(c ports.HTTPContext) {
	status, err := h.setupService.SetupStatus(c.RequestContext())
	h.write(c, status, err)
}

func (h *Handler) InitialAdminSetup(c ports.HTTPContext) {
	var req InitialAdminSetupRequest
	if !bind(c, &req) {
		return
	}
	productCode, clientType := h.requestSessionContext(c)
	pair, err := h.setupService.InitialAdminSetup(c.RequestContext(), service.InitialAdminSetupInput{
		OrgCode:     req.OrgCode,
		OrgName:     req.OrgName,
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Password:    req.Password,
		ProductCode: productCode,
		ClientType:  clientType,
		UserAgent:   c.GetHeader("User-Agent"),
		IPAddress:   c.ClientIP(),
	})
	if err != nil {
		h.writeSetupError(c, err)
		return
	}
	h.writeAuthResult(c, pair, nil)
}

func (h *Handler) Login(c ports.HTTPContext) {
	var req LoginRequest
	if !bind(c, &req) {
		return
	}
	productCode, clientType := h.requestSessionContext(c)
	pair, err := h.service.Login(c.RequestContext(), service.LoginInput{
		CaptchaCode: req.CaptchaCode,
		CaptchaID:   req.CaptchaID,
		Identifier:  req.Identifier,
		Password:    req.Password,
		OrgCode:     req.OrgCode,
		MFACode:     req.MFACode,
		ProductCode: productCode,
		ClientType:  clientType,
		UserAgent:   c.GetHeader("User-Agent"),
		IPAddress:   c.ClientIP(),
	})
	h.writeAuthResult(c, pair, err)
}

func (h *Handler) Captcha(c ports.HTTPContext) {
	challenge, err := h.service.Captcha(c.RequestContext())
	h.write(c, challenge, err)
}

func (h *Handler) Refresh(c ports.HTTPContext) {
	if !h.validateCSRF(c) {
		return
	}
	refreshToken, ok := h.refreshTokenFromCookie(c)
	if !ok {
		return
	}
	pair, err := h.service.Refresh(c.RequestContext(), service.RefreshInput{
		RefreshToken: refreshToken,
		UserAgent:    c.GetHeader("User-Agent"),
		IPAddress:    c.ClientIP(),
	})
	h.writeAuthResult(c, pair, err)
}

func (h *Handler) Logout(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	err := h.service.Logout(c.RequestContext(), principal)
	if err == nil {
		h.clearAuthCookies(c)
	}
	h.write(c, map[string]bool{"loggedOut": true}, err)
}

func (h *Handler) SwitchOrg(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	var req SwitchOrgRequest
	if !bind(c, &req) {
		return
	}
	pair, err := h.service.SwitchOrg(c.RequestContext(), principal, int64(req.OrgID), c.GetHeader("User-Agent"), c.ClientIP())
	h.writeAuthResult(c, pair, err)
}

func (h *Handler) Session(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	session, err := h.service.CurrentSession(c.RequestContext(), principal)
	h.write(c, session, err)
}

func (h *Handler) Me(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	user, err := h.service.Me(c.RequestContext(), principal)
	h.write(c, user, err)
}

func (h *Handler) MyOrganizations(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	orgs, err := h.service.ListMyOrganizations(c.RequestContext(), principal)
	h.write(c, orgs, err)
}

func (h *Handler) ListOrganizations(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	filter, ok := parseOrganizationListFilter(c)
	if !ok {
		return
	}
	orgs, err := h.service.ListOrganizations(c.RequestContext(), principal, filter)
	h.write(c, orgs, err)
}

func (h *Handler) CreateOrganization(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	var req CreateOrgRequest
	if !bind(c, &req) {
		return
	}
	org, err := h.service.CreateOrganization(c.RequestContext(), principal, req.Code, req.Name)
	h.writeCreated(c, org, err)
}

func (h *Handler) UpdateOrganization(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	orgID, ok := parseInt64Param(c, "orgId")
	if !ok {
		return
	}
	var req UpdateOrgRequest
	if !bind(c, &req) {
		return
	}
	org, err := h.service.UpdateOrganization(c.RequestContext(), service.UpdateOrganizationInput{
		Principal: principal,
		OrgID:     orgID,
		Name:      req.Name,
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	})
	h.write(c, org, err)
}

func (h *Handler) InviteUser(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	var req InviteUserRequest
	if !bind(c, &req) {
		return
	}
	delivery, err := h.service.InviteUser(c.RequestContext(), service.InviteUserInput{
		Principal: principal,
		Email:     req.Email,
		RoleCode:  req.RoleCode,
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	})
	h.writeCreated(c, delivery, err)
}

func (h *Handler) ListInvitations(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	invitations, err := h.service.ListInvitations(c.RequestContext(), principal)
	h.write(c, invitations, err)
}

func (h *Handler) RevokeInvitation(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	id, ok := parseInt64Param(c, "invitationId")
	if !ok {
		return
	}
	h.write(c, map[string]bool{"revoked": true}, h.service.RevokeInvitation(c.RequestContext(), principal, id, c.GetHeader("User-Agent"), c.ClientIP()))
}

func (h *Handler) AcceptInvitation(c ports.HTTPContext) {
	var req AcceptInvitationRequest
	if !bind(c, &req) {
		return
	}
	principal, err := h.service.AcceptInvitation(c.RequestContext(), service.AcceptInvitationInput{
		Token:       c.Param("token"),
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Password:    req.Password,
		UserAgent:   c.GetHeader("User-Agent"),
		IPAddress:   c.ClientIP(),
	})
	h.writeCreated(c, principal, err)
}

func (h *Handler) ForgotPassword(c ports.HTTPContext) {
	var req ForgotPasswordRequest
	if !bind(c, &req) {
		return
	}
	delivery, err := h.service.ForgotPassword(c.RequestContext(), service.ForgotPasswordInput{
		Email:     req.Email,
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	})
	h.write(c, delivery, err)
}

func (h *Handler) ResetPassword(c ports.HTTPContext) {
	var req ResetPasswordRequest
	if !bind(c, &req) {
		return
	}
	err := h.service.ResetPassword(c.RequestContext(), service.ResetPasswordInput{
		Token:       req.Token,
		NewPassword: req.NewPassword,
		UserAgent:   c.GetHeader("User-Agent"),
		IPAddress:   c.ClientIP(),
	})
	h.write(c, map[string]bool{"reset": true}, err)
}

func (h *Handler) SetupMFA(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	secret, url, err := h.service.SetupMFA(c.RequestContext(), principal)
	h.write(c, map[string]string{"secret": secret, "otpauthUrl": url}, err)
}

func (h *Handler) VerifyMFA(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	var req VerifyMFARequest
	if !bind(c, &req) {
		return
	}
	h.write(c, map[string]bool{"verified": true}, h.service.VerifyMFA(c.RequestContext(), principal, req.Code))
}

func (h *Handler) ListUsers(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	filter, ok := parseUserListFilter(c)
	if !ok {
		return
	}
	users, err := h.service.ListUsers(c.RequestContext(), principal, filter)
	h.write(c, users, err)
}

func (h *Handler) UpdateUser(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	userID, ok := parseInt64Param(c, "userId")
	if !ok {
		return
	}
	var req UpdateUserRequest
	if !bind(c, &req) {
		return
	}
	input := service.UpdateUserInput{
		Principal: principal,
		UserID:    userID,
		Status:    req.Status,
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	}
	if req.Roles != nil {
		input.Roles = *req.Roles
		input.HasRoles = true
	}
	user, err := h.service.UpdateUser(c.RequestContext(), input)
	h.write(c, user, err)
}

func (h *Handler) ListAPITokens(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	filter, ok := parseAPITokenFilter(c)
	if !ok {
		return
	}
	page, err := h.service.ListAPITokens(c.RequestContext(), principal, filter)
	h.write(c, page, err)
}

func (h *Handler) CreateAPIToken(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	var req CreateAPITokenRequest
	if !bind(c, &req) {
		return
	}
	created, err := h.service.CreateAPIToken(c.RequestContext(), service.CreateAPITokenInput{
		Principal: principal,
		UserID:    int64(req.UserID),
		RoleCode:  req.RoleCode,
		Days:      req.Days,
		Remark:    req.Remark,
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	})
	h.writeCreated(c, created, err)
}

func (h *Handler) RevokeAPIToken(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	id, ok := parseInt64Param(c, "tokenId")
	if !ok {
		return
	}
	h.write(c, map[string]bool{"revoked": true}, h.service.RevokeAPIToken(c.RequestContext(), service.RevokeAPITokenInput{
		Principal: principal,
		TokenID:   id,
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	}))
}

func (h *Handler) ListRoles(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	roles, err := h.service.ListRoles(c.RequestContext(), principal)
	h.write(c, roles, err)
}

func (h *Handler) CreateRole(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	var req CreateRoleRequest
	if !bind(c, &req) {
		return
	}
	role, err := h.service.CreateRole(c.RequestContext(), service.CreateRoleInput{
		Principal:   principal,
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
	})
	h.writeCreated(c, role, err)
}

func (h *Handler) UpdateRole(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	roleID, ok := parseInt64Param(c, "roleId")
	if !ok {
		return
	}
	var req UpdateRoleRequest
	if !bind(c, &req) {
		return
	}
	input := service.UpdateRoleInput{
		Principal:   principal,
		RoleID:      roleID,
		Name:        req.Name,
		Description: req.Description,
		UserAgent:   c.GetHeader("User-Agent"),
		IPAddress:   c.ClientIP(),
	}
	if req.Permissions != nil {
		input.Permissions = *req.Permissions
		input.HasPermissions = true
	}
	role, err := h.service.UpdateRole(c.RequestContext(), input)
	h.write(c, role, err)
}

func (h *Handler) ListPermissions(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	permissions, err := h.service.ListPermissions(c.RequestContext(), principal)
	h.write(c, permissions, err)
}

func (h *Handler) ListSessions(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	filter, ok := parseSessionListFilter(c)
	if !ok {
		return
	}
	sessions, err := h.service.ListSessions(c.RequestContext(), principal, filter)
	h.write(c, sessions, err)
}

func (h *Handler) RevokeSession(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	id, ok := parseInt64Param(c, "sessionId")
	if !ok {
		return
	}
	h.write(c, map[string]bool{"revoked": true}, h.service.RevokeSession(c.RequestContext(), principal, id))
}

func (h *Handler) ListAuditLogs(c ports.HTTPContext) {
	principal, ok := requirePrincipal(c)
	if !ok {
		return
	}
	filter, ok := parseAuditLogFilter(c)
	if !ok {
		return
	}
	logs, err := h.service.ListAuditLogs(c.RequestContext(), principal, filter)
	h.write(c, logs, err)
}

// parseAuditLogFilter 从查询参数构造审计日志过滤条件。
// 时间参数要求 RFC3339，解析失败会直接写入 400 响应并终止 handler。
func parseAuditLogFilter(c ports.HTTPContext) (service.AuditLogFilter, bool) {
	query := c.Request().URL.Query()
	filter := service.AuditLogFilter{Action: query.Get("action")}
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			result.BadRequest(c, "invalid limit")
			return service.AuditLogFilter{}, false
		}
		filter.Limit = parsed
	}
	if raw := query.Get("userId"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			result.BadRequest(c, "invalid userId")
			return service.AuditLogFilter{}, false
		}
		filter.UserID = parsed
	}
	if raw := query.Get("cursor"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			result.BadRequest(c, "invalid cursor")
			return service.AuditLogFilter{}, false
		}
		filter.Cursor = parsed
	}
	if raw := query.Get("from"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			result.BadRequest(c, "invalid from")
			return service.AuditLogFilter{}, false
		}
		filter.From = parsed
	}
	if raw := query.Get("to"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			result.BadRequest(c, "invalid to")
			return service.AuditLogFilter{}, false
		}
		filter.To = parsed
	}
	return filter, true
}

// parseAPITokenFilter 从查询参数构造 API token 列表过滤条件。
func parseAPITokenFilter(c ports.HTTPContext) (service.APITokenFilter, bool) {
	query := c.Request().URL.Query()
	filter := service.APITokenFilter{Status: query.Get("status")}
	if raw := query.Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			result.BadRequest(c, "invalid page")
			return service.APITokenFilter{}, false
		}
		filter.Page = parsed
	}
	if raw := query.Get("pageSize"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			result.BadRequest(c, "invalid pageSize")
			return service.APITokenFilter{}, false
		}
		filter.PageSize = parsed
	}
	if raw := query.Get("userId"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			result.BadRequest(c, "invalid userId")
			return service.APITokenFilter{}, false
		}
		filter.UserID = parsed
	}
	return filter, true
}

// parseUserListFilter 从查询参数构造组织成员列表过滤条件。
// displayName 兼容 nickName/nickname，避免前端历史字段名变化导致过滤失效。
func parseUserListFilter(c ports.HTTPContext) (service.UserListFilter, bool) {
	query := c.Request().URL.Query()
	filter := service.UserListFilter{
		Keyword:     query.Get("keyword"),
		Username:    query.Get("username"),
		DisplayName: firstNonEmpty(query.Get("displayName"), query.Get("nickName"), query.Get("nickname")),
		Email:       query.Get("email"),
		RoleCode:    query.Get("roleCode"),
		Status:      query.Get("status"),
		OrderKey:    query.Get("orderKey"),
		Desc:        query.Get("desc") == "true" || query.Get("desc") == "1",
	}
	if raw := query.Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			result.BadRequest(c, "invalid page")
			return service.UserListFilter{}, false
		}
		filter.Page = parsed
	}
	if raw := query.Get("pageSize"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			result.BadRequest(c, "invalid pageSize")
			return service.UserListFilter{}, false
		}
		filter.PageSize = parsed
	}
	return filter, true
}

// parseOrganizationListFilter 从查询参数构造组织列表过滤条件。
func parseOrganizationListFilter(c ports.HTTPContext) (service.OrganizationListFilter, bool) {
	query := c.Request().URL.Query()
	filter := service.OrganizationListFilter{
		Keyword:  query.Get("keyword"),
		Code:     query.Get("code"),
		Name:     query.Get("name"),
		Status:   query.Get("status"),
		OrderKey: query.Get("orderKey"),
		Desc:     query.Get("desc") == "true" || query.Get("desc") == "1",
	}
	if raw := query.Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			result.BadRequest(c, "invalid page")
			return service.OrganizationListFilter{}, false
		}
		filter.Page = parsed
	}
	if raw := query.Get("pageSize"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			result.BadRequest(c, "invalid pageSize")
			return service.OrganizationListFilter{}, false
		}
		filter.PageSize = parsed
	}
	return filter, true
}

// parseSessionListFilter 从查询参数构造会话列表过滤条件。
// ipAddress 兼容 ip 简写，scope 控制服务层按个人或组织视角查询。
func parseSessionListFilter(c ports.HTTPContext) (service.SessionListFilter, bool) {
	query := c.Request().URL.Query()
	filter := service.SessionListFilter{
		Keyword:     query.Get("keyword"),
		IPAddress:   firstNonEmpty(query.Get("ipAddress"), query.Get("ip")),
		Status:      query.Get("status"),
		Scope:       query.Get("scope"),
		ProductCode: firstNonEmpty(query.Get("productCode"), query.Get("product_code")),
		ClientType:  firstNonEmpty(query.Get("clientType"), query.Get("client_type"), query.Get("platform")),
		OrderKey:    query.Get("orderKey"),
		Desc:        query.Get("desc") == "true" || query.Get("desc") == "1",
	}
	if raw := query.Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			result.BadRequest(c, "invalid page")
			return service.SessionListFilter{}, false
		}
		filter.Page = parsed
	}
	if raw := query.Get("pageSize"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			result.BadRequest(c, "invalid pageSize")
			return service.SessionListFilter{}, false
		}
		filter.PageSize = parsed
	}
	if raw := query.Get("userId"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			result.BadRequest(c, "invalid userId")
			return service.SessionListFilter{}, false
		}
		filter.UserID = parsed
	}
	return filter, true
}

// firstNonEmpty 返回第一个非空字符串，用于兼容多个查询参数别名。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// bind 绑定 JSON 请求体；失败时统一写 400 响应。
func bind(c ports.HTTPContext, dest any) bool {
	if err := c.BindJSON(dest); err != nil {
		result.BadRequest(c, result.MessageKeyInvalidRequest)
		return false
	}
	return true
}

// requirePrincipal 从认证中间件结果中读取 Principal。
// 返回 false 表示已经写入 401 响应，调用方应立即结束。
func requirePrincipal(c ports.HTTPContext) (service.Principal, bool) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		result.Unauthorized(c, "api.auth.missingPrincipal")
		return service.Principal{}, false
	}
	return principal, true
}

// parseInt64Param 解析正整数路径参数。
func parseInt64Param(c ports.HTTPContext, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		result.BadRequest(c, "validation.common.invalid", map[string]any{"field": name})
		return 0, false
	}
	return id, true
}

// write 将服务层返回值写为普通成功响应或统一错误响应。
func (h *Handler) write(c ports.HTTPContext, data any, err error) {
	if err != nil {
		h.writeError(c, err)
		return
	}
	result.OK(c, data)
}

// writeCreated 将创建成功响应写为 201。
func (h *Handler) writeCreated(c ports.HTTPContext, data any, err error) {
	if err != nil {
		h.writeError(c, err)
		return
	}
	result.Created(c, data)
}

// writeSetupError 为首次初始化接口提供更明确的错误映射。
// 未知错误会透出初始化失败信息，便于部署阶段排查。
func (h *Handler) writeSetupError(c ports.HTTPContext, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		result.BadRequest(c, result.MessageKeyInvalidRequest)
	case errors.Is(err, service.ErrSetupCompleted):
		result.Forbidden(c, "api.setup.forbidden")
	case errors.Is(err, service.ErrDuplicate):
		result.BadRequest(c, "api.common.duplicate")
	default:
		if h.logger != nil {
			h.logger.Error("iam setup failed", "error", err)
		}
		result.InternalError(c, result.MessageKeyInternalError)
	}
}

// writeError 将 IAM 服务层错误映射为 HTTP 响应。
// 未知错误只记录日志并返回通用 500，避免泄露内部实现细节。
func (h *Handler) writeError(c ports.HTTPContext, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		result.BadRequest(c, result.MessageKeyInvalidRequest)
	case errors.Is(err, service.ErrCaptchaRequired):
		result.BadRequest(c, "api.auth.captchaRequired")
	case errors.Is(err, service.ErrCaptchaInvalid):
		result.BadRequest(c, "api.auth.captchaInvalid")
	case errors.Is(err, service.ErrUnauthorized):
		result.Unauthorized(c, result.MessageKeyUnauthorized)
	case errors.Is(err, service.ErrMFARequired):
		result.Unauthorized(c, "api.auth.mfaRequired")
	case errors.Is(err, service.ErrInvalidToken):
		result.Unauthorized(c, "api.auth.invalidToken")
	case errors.Is(err, service.ErrAccountLocked):
		result.Unauthorized(c, "api.auth.accountLocked")
	case errors.Is(err, service.ErrAccountDisabled):
		result.Unauthorized(c, "api.auth.accountDisabled")
	case errors.Is(err, service.ErrSessionRevoked):
		result.Unauthorized(c, "api.auth.sessionRevoked")
	case errors.Is(err, service.ErrForbidden):
		result.Forbidden(c, result.MessageKeyForbidden)
	case errors.Is(err, service.ErrSignupDisabled):
		result.Forbidden(c, "api.auth.signupDisabled")
	case errors.Is(err, service.ErrSetupCompleted):
		result.Forbidden(c, "api.setup.forbidden")
	case errors.Is(err, service.ErrNotFound):
		result.NotFound(c, result.MessageKeyNotFound)
	case errors.Is(err, service.ErrDuplicate):
		result.BadRequest(c, "api.common.duplicate")
	case errors.Is(err, service.ErrInvitationClosed):
		result.BadRequest(c, "api.auth.invitationClosed")
	case errors.Is(err, service.ErrNotificationDelivery):
		result.Fail(c, http.StatusServiceUnavailable, "api.auth.notificationDeliveryFailed")
	default:
		if h.logger != nil {
			h.logger.Error("iam request failed", "error", err)
		}
		result.InternalError(c, result.MessageKeyInternalError)
	}
}

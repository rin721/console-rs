package service_test

import (
	"context"
	"encoding/base64"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rei0721/go-scaffold/internal/app/testsupport"
	"github.com/rei0721/go-scaffold/internal/modules/iam/model"
	"github.com/rei0721/go-scaffold/internal/modules/iam/repository"

	. "github.com/rei0721/go-scaffold/internal/modules/iam/service"
)

func permissionContext(scope string, object string, action string) PermissionContext {
	return PermissionContext{Scope: scope, Object: object, Action: action}
}

func TestIAMLifecycle(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	admin, err := svc.BootstrapAdmin(ctx, BootstrapAdminInput{
		OrgCode:  "acme",
		OrgName:  "Acme",
		Username: "admin",
		Email:    "admin@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("BootstrapAdmin() failed: %v", err)
	}
	if admin.OrgID == 0 || admin.UserID == 0 {
		t.Fatalf("unexpected principal: %#v", admin)
	}

	allowed, err := svc.Authorize(ctx, *admin, permissionContext(model.PermissionScopeTenant, "audit", "read"))
	if err != nil || !allowed {
		t.Fatalf("owner should read audit logs, allowed=%v err=%v", allowed, err)
	}

	login, err := svc.Login(ctx, LoginInput{Identifier: "admin@example.com", Password: "password123", OrgCode: "acme"})
	if err != nil {
		t.Fatalf("Login() failed: %v", err)
	}
	principal, err := svc.AuthenticateToken(ctx, login.AccessToken)
	if err != nil {
		t.Fatalf("AuthenticateToken() failed: %v", err)
	}
	if principal.UserID != admin.UserID || principal.OrgID != admin.OrgID {
		t.Fatalf("unexpected authenticated principal: %#v", principal)
	}

	refreshed, err := svc.Refresh(ctx, RefreshInput{RefreshToken: login.RefreshToken})
	if err != nil {
		t.Fatalf("Refresh() failed: %v", err)
	}
	if refreshed.AccessToken == "" || refreshed.RefreshToken == "" || refreshed.RefreshToken == login.RefreshToken {
		t.Fatalf("refresh rotation failed: %#v", refreshed)
	}

	inviteDelivery, err := svc.InviteUser(ctx, InviteUserInput{Principal: principal, Email: "member@example.com", RoleCode: model.RoleMember})
	if err != nil {
		t.Fatalf("InviteUser() failed: %v", err)
	}
	if inviteDelivery.Token == "" || inviteDelivery.URL == "" {
		t.Fatalf("expected debug invitation delivery, got %#v", inviteDelivery)
	}
	if !inviteDelivery.Debug {
		t.Fatalf("expected debug invitation delivery flag, got %#v", inviteDelivery)
	}
	member, err := svc.AcceptInvitation(ctx, AcceptInvitationInput{Token: inviteDelivery.Token, Username: "member", Password: "password123"})
	if err != nil {
		t.Fatalf("AcceptInvitation() failed: %v", err)
	}
	memberAllowed, err := svc.Authorize(ctx, *member, permissionContext(model.PermissionScopeTenant, "audit", "read"))
	if err != nil {
		t.Fatalf("Authorize(member) failed: %v", err)
	}
	if memberAllowed {
		t.Fatal("member should not read audit logs")
	}
	memberLogin, err := svc.Login(ctx, LoginInput{Identifier: "member@example.com", Password: "password123", OrgCode: "acme"})
	if err != nil {
		t.Fatalf("member Login() before reset failed: %v", err)
	}

	resetDelivery, err := svc.ForgotPassword(ctx, ForgotPasswordInput{Email: "member@example.com"})
	if err != nil {
		t.Fatalf("ForgotPassword() failed: %v", err)
	}
	if resetDelivery.Token == "" || resetDelivery.URL == "" {
		t.Fatalf("expected debug password reset delivery, got %#v", resetDelivery)
	}
	if !resetDelivery.Debug {
		t.Fatalf("expected debug password reset delivery flag, got %#v", resetDelivery)
	}
	if err := svc.ResetPassword(ctx, ResetPasswordInput{Token: resetDelivery.Token, NewPassword: "newpassword123"}); err != nil {
		t.Fatalf("ResetPassword() failed: %v", err)
	}
	if _, err := svc.Refresh(ctx, RefreshInput{RefreshToken: memberLogin.RefreshToken}); err != ErrSessionRevoked {
		t.Fatalf("old refresh after password reset = %v, want ErrSessionRevoked", err)
	}
	if _, err := svc.Login(ctx, LoginInput{Identifier: "member@example.com", Password: "newpassword123", OrgCode: "acme"}); err != nil {
		t.Fatalf("member login after reset failed: %v", err)
	}
}

func TestDirectSignupCreatesOwnerSession(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	signup, err := svc.Signup(ctx, SignupInput{
		OrgCode:  "acme",
		OrgName:  "Acme",
		Username: "owner",
		Email:    "owner@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Signup() failed: %v", err)
	}
	if signup.Status != SignupStatusAuthenticated || signup.Session == nil || signup.Tokens.AccessToken == "" {
		t.Fatalf("unexpected signup result: %#v", signup)
	}
	principal, err := svc.AuthenticateToken(ctx, signup.Tokens.AccessToken)
	if err != nil {
		t.Fatalf("AuthenticateToken(signup token) failed: %v", err)
	}
	allowed, err := svc.Authorize(ctx, principal, permissionContext(model.PermissionScopeTenant, "audit", "read"))
	if err != nil || !allowed {
		t.Fatalf("signup owner should read audit logs, allowed=%v err=%v", allowed, err)
	}
	platformAllowed, err := svc.Authorize(ctx, principal, permissionContext(model.PermissionScopePlatform, "config", "read"))
	if err != nil {
		t.Fatalf("Authorize(platform config) failed: %v", err)
	}
	if platformAllowed {
		t.Fatal("signup owner should not read platform config")
	}
	if _, err := svc.CreateAPIToken(ctx, CreateAPITokenInput{
		Principal: principal,
		UserID:    principal.UserID,
		RoleCode:  model.RolePlatformOwner,
		Days:      1,
	}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("tenant owner CreateAPIToken(platform_owner) error = %v, want ErrForbidden", err)
	}
	orgs, err := svc.ListMyOrganizations(ctx, principal)
	if err != nil || len(orgs) != 1 || orgs[0].Code != "acme" {
		t.Fatalf("unexpected signup organizations: %#v err=%v", orgs, err)
	}
	if _, err := svc.CreateRole(ctx, CreateRoleInput{
		Principal:   principal,
		Code:        "operator",
		Name:        "Operator",
		Permissions: []string{"audit:read", "user:read"},
	}); err != nil {
		t.Fatalf("CreateRole() failed: %v", err)
	}
	roles, err := svc.ListRoles(ctx, principal)
	if err != nil {
		t.Fatalf("ListRoles() failed: %v", err)
	}
	var operator *model.Role
	for i := range roles {
		if roles[i].Code == model.RolePlatformOwner {
			t.Fatal("tenant signup organization should not expose platform_owner role")
		}
		if roles[i].Code == "operator" {
			operator = &roles[i]
		}
	}
	if operator == nil || !containsString(operator.Permissions, "audit:read") || !containsString(operator.Permissions, "user:read") {
		t.Fatalf("operator permissions not hydrated: %#v", operator)
	}

	if _, err := svc.Signup(ctx, SignupInput{OrgCode: "acme", OrgName: "Other", Username: "other", Email: "other@example.com", Password: "password123"}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate org signup error = %v, want ErrDuplicate", err)
	}
	if _, err := svc.Signup(ctx, SignupInput{OrgCode: "other", OrgName: "Other", Username: "owner", Email: "other@example.com", Password: "password123"}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate username signup error = %v, want ErrDuplicate", err)
	}
	if _, err := svc.Signup(ctx, SignupInput{OrgCode: "other", OrgName: "Other", Username: "other", Email: "owner@example.com", Password: "password123"}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate email signup error = %v, want ErrDuplicate", err)
	}
}

func TestEmailVerificationSignupConfirmsPendingAccount(t *testing.T) {
	ctx := context.Background()
	notifier := &recordingNotifier{}
	svc, repo, deps, cleanup := newTestServiceWithNotifier(t, RegistrationModeEmailVerification, notifier)
	defer cleanup()

	signup, err := svc.Signup(ctx, SignupInput{
		OrgCode:  "acme",
		OrgName:  "Acme",
		Username: "owner",
		Email:    "owner@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Signup() failed: %v", err)
	}
	if signup.Status != SignupStatusVerificationPending || signup.Session != nil || signup.Tokens.AccessToken != "" {
		t.Fatalf("unexpected email verification signup result: %#v", signup)
	}
	if signup.Delivery == nil || !signup.Delivery.Debug || signup.Delivery.Token == "" || signup.Delivery.URL == "" {
		t.Fatalf("expected debug verification delivery, got %#v", signup.Delivery)
	}
	if notifier.emailVerification == nil || notifier.emailVerification.Token != signup.Delivery.Token {
		t.Fatalf("notifier did not receive verification notice: %#v", notifier.emailVerification)
	}
	if _, err := svc.Login(ctx, LoginInput{Identifier: "owner@example.com", Password: "password123", OrgCode: "acme"}); !errors.Is(err, ErrAccountDisabled) {
		t.Fatalf("pending account login error = %v, want ErrAccountDisabled", err)
	}
	verification, err := repo.FindEmailVerificationByTokenHash(ctx, deps.Tokens.HashRefreshToken(signup.Delivery.Token))
	if err != nil {
		t.Fatalf("FindEmailVerificationByTokenHash() failed: %v", err)
	}
	if verification.Status != model.StatusPending {
		t.Fatalf("verification status = %q, want pending", verification.Status)
	}

	pair, err := svc.ConfirmEmailVerification(ctx, ConfirmEmailVerificationInput{Token: signup.Delivery.Token})
	if err != nil {
		t.Fatalf("ConfirmEmailVerification() failed: %v", err)
	}
	principal, err := svc.AuthenticateToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("AuthenticateToken(verification token) failed: %v", err)
	}
	if principal.Email != "owner@example.com" || principal.OrgID == 0 {
		t.Fatalf("unexpected principal after verification: %#v", principal)
	}
	platformAllowed, err := svc.Authorize(ctx, principal, permissionContext(model.PermissionScopePlatform, "config", "read"))
	if err != nil {
		t.Fatalf("Authorize(platform config) failed: %v", err)
	}
	if platformAllowed {
		t.Fatal("email verification owner should not read platform config")
	}
	verification, err = repo.FindEmailVerificationByTokenHash(ctx, deps.Tokens.HashRefreshToken(signup.Delivery.Token))
	if err != nil {
		t.Fatalf("FindEmailVerificationByTokenHash() after confirm failed: %v", err)
	}
	if verification.Status != model.StatusUsed || verification.VerifiedAt == nil {
		t.Fatalf("verification after confirm = %#v, want used with verifiedAt", verification)
	}
}

func TestEmailVerificationSignupCleansPendingRowsWhenNotificationFails(t *testing.T) {
	ctx := context.Background()
	notifier := &failingNotifier{err: errors.New("smtp down")}
	svc, repo, deps, cleanup := newTestServiceWithNotifier(t, RegistrationModeEmailVerification, notifier)
	defer cleanup()

	input := SignupInput{OrgCode: "acme", OrgName: "Acme", Username: "owner", Email: "owner@example.com", Password: "password123"}
	if _, err := svc.Signup(ctx, input); !errors.Is(err, ErrNotificationDelivery) {
		t.Fatalf("Signup() error = %v, want ErrNotificationDelivery", err)
	}
	if notifier.emailVerification == nil || notifier.emailVerification.Token == "" {
		t.Fatalf("notifier did not receive email verification notice: %#v", notifier.emailVerification)
	}
	if _, err := repo.FindEmailVerificationByTokenHash(ctx, deps.Tokens.HashRefreshToken(notifier.emailVerification.Token)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("verification after delivery failure error = %v, want ErrNotFound", err)
	}
	if _, err := svc.Signup(ctx, input); !errors.Is(err, ErrNotificationDelivery) {
		t.Fatalf("retry after cleanup error = %v, want ErrNotificationDelivery", err)
	}
}

func TestInviteUserRevokesInvitationWhenNotificationFails(t *testing.T) {
	ctx := context.Background()
	notifier := &failingNotifier{err: errors.New("smtp down")}
	svc, repo, deps, cleanup := newTestServiceWithNotifier(t, RegistrationModeDirect, notifier)
	defer cleanup()

	admin, err := svc.BootstrapAdmin(ctx, BootstrapAdminInput{
		OrgCode:  "acme",
		OrgName:  "Acme",
		Username: "admin",
		Email:    "admin@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("BootstrapAdmin() failed: %v", err)
	}
	delivery, err := svc.InviteUser(ctx, InviteUserInput{Principal: *admin, Email: "member@example.com", RoleCode: model.RoleMember})
	if !errors.Is(err, ErrNotificationDelivery) {
		t.Fatalf("InviteUser() error = %v, want ErrNotificationDelivery", err)
	}
	if delivery.Token != "" || delivery.URL != "" {
		t.Fatalf("smtp failure should not expose debug delivery: %#v", delivery)
	}
	if notifier.invitation == nil || notifier.invitation.Token == "" {
		t.Fatalf("notifier did not receive invitation notice: %#v", notifier.invitation)
	}
	invitation, err := repo.FindInvitationByTokenHash(ctx, deps.Tokens.HashRefreshToken(notifier.invitation.Token))
	if err != nil {
		t.Fatalf("FindInvitationByTokenHash() failed: %v", err)
	}
	if invitation.Status != model.StatusRevoked {
		t.Fatalf("invitation status = %q, want %q", invitation.Status, model.StatusRevoked)
	}
}

func TestInviteUserRejectsPlatformOwnerRole(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	admin, err := svc.BootstrapAdmin(ctx, BootstrapAdminInput{OrgCode: "acme", Username: "admin", Email: "admin@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("BootstrapAdmin() failed: %v", err)
	}
	_, err = svc.InviteUser(ctx, InviteUserInput{Principal: *admin, Email: "owner@example.com", RoleCode: model.RolePlatformOwner})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("InviteUser(platform_owner) error = %v, want ErrForbidden", err)
	}
}

func TestSMTPNotificationDriverDoesNotExposeDebugDelivery(t *testing.T) {
	ctx := context.Background()
	notifier := &recordingNotifier{}
	svc, _, _, cleanup := newTestServiceWithNotifier(t, RegistrationModeDirect, notifier, func(cfg *Config) {
		cfg.NotificationDriver = "smtp"
	})
	defer cleanup()

	admin, err := svc.BootstrapAdmin(ctx, BootstrapAdminInput{
		OrgCode:  "acme",
		OrgName:  "Acme",
		Username: "admin",
		Email:    "admin@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("BootstrapAdmin() failed: %v", err)
	}
	invite, err := svc.InviteUser(ctx, InviteUserInput{Principal: *admin, Email: "member@example.com", RoleCode: model.RoleMember})
	if err != nil {
		t.Fatalf("InviteUser() failed: %v", err)
	}
	if invite.Debug || invite.Token != "" || invite.URL != "" {
		t.Fatalf("smtp invitation should not expose debug delivery: %#v", invite)
	}
	if notifier.invitation == nil || notifier.invitation.Token == "" {
		t.Fatalf("notifier did not receive invitation notice: %#v", notifier.invitation)
	}

	reset, err := svc.ForgotPassword(ctx, ForgotPasswordInput{Email: "admin@example.com"})
	if err != nil {
		t.Fatalf("ForgotPassword() failed: %v", err)
	}
	if reset.Debug || reset.Token != "" || reset.URL != "" {
		t.Fatalf("smtp password reset should not expose debug delivery: %#v", reset)
	}
	if notifier.passwordReset == nil || notifier.passwordReset.Token == "" {
		t.Fatalf("notifier did not receive password reset notice: %#v", notifier.passwordReset)
	}
}

func TestReloadNotificationRuntimeStopsDebugDelivery(t *testing.T) {
	ctx := context.Background()
	reloadable := NewReloadableNotifier(NoopNotifier{})
	svc, _, _, cleanup := newTestServiceWithNotifier(t, RegistrationModeDirect, reloadable)
	defer cleanup()

	admin, err := svc.BootstrapAdmin(ctx, BootstrapAdminInput{
		OrgCode:  "acme",
		OrgName:  "Acme",
		Username: "admin",
		Email:    "admin@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("BootstrapAdmin() failed: %v", err)
	}
	debugInvite, err := svc.InviteUser(ctx, InviteUserInput{Principal: *admin, Email: "debug@example.com", RoleCode: model.RoleMember})
	if err != nil {
		t.Fatalf("InviteUser(debug) failed: %v", err)
	}
	if !debugInvite.Debug || debugInvite.Token == "" || debugInvite.URL == "" {
		t.Fatalf("expected debug delivery before reload, got %#v", debugInvite)
	}

	notifier := &recordingNotifier{}
	reloadable.Replace(notifier)
	reloader, ok := svc.(NotificationRuntimeReloader)
	if !ok {
		t.Fatal("service does not implement NotificationRuntimeReloader")
	}
	reloader.ReloadNotificationRuntime(NotificationRuntimeConfig{NotificationDriver: "smtp", PublicBaseURL: "/admin"})

	smtpInvite, err := svc.InviteUser(ctx, InviteUserInput{Principal: *admin, Email: "smtp@example.com", RoleCode: model.RoleMember})
	if err != nil {
		t.Fatalf("InviteUser(smtp) failed: %v", err)
	}
	if smtpInvite.Debug || smtpInvite.Token != "" || smtpInvite.URL != "" {
		t.Fatalf("expected no debug delivery after smtp reload, got %#v", smtpInvite)
	}
	if notifier.invitation == nil || notifier.invitation.Email != "smtp@example.com" {
		t.Fatalf("reloaded notifier did not receive smtp invitation: %#v", notifier.invitation)
	}
}

func TestForgotPasswordRevokesResetWhenNotificationFails(t *testing.T) {
	ctx := context.Background()
	notifier := &failingNotifier{err: errors.New("smtp down")}
	svc, repo, deps, cleanup := newTestServiceWithNotifier(t, RegistrationModeDirect, notifier)
	defer cleanup()

	if _, err := svc.BootstrapAdmin(ctx, BootstrapAdminInput{
		OrgCode:  "acme",
		OrgName:  "Acme",
		Username: "admin",
		Email:    "admin@example.com",
		Password: "password123",
	}); err != nil {
		t.Fatalf("BootstrapAdmin() failed: %v", err)
	}
	delivery, err := svc.ForgotPassword(ctx, ForgotPasswordInput{Email: "admin@example.com"})
	if !errors.Is(err, ErrNotificationDelivery) {
		t.Fatalf("ForgotPassword() error = %v, want ErrNotificationDelivery", err)
	}
	if delivery.Token != "" || delivery.URL != "" {
		t.Fatalf("smtp failure should not expose debug delivery: %#v", delivery)
	}
	if notifier.passwordReset == nil || notifier.passwordReset.Token == "" {
		t.Fatalf("notifier did not receive password reset notice: %#v", notifier.passwordReset)
	}
	reset, err := repo.FindPasswordResetByTokenHash(ctx, deps.Tokens.HashRefreshToken(notifier.passwordReset.Token))
	if err != nil {
		t.Fatalf("FindPasswordResetByTokenHash() failed: %v", err)
	}
	if reset.Status != model.StatusRevoked {
		t.Fatalf("password reset status = %q, want %q", reset.Status, model.StatusRevoked)
	}
	if err := svc.ResetPassword(ctx, ResetPasswordInput{Token: notifier.passwordReset.Token, NewPassword: "newpassword123"}); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("ResetPassword(revoked token) error = %v, want ErrInvalidToken", err)
	}
}

func TestForgotPasswordKeepsEmptySuccessForUnknownEmail(t *testing.T) {
	ctx := context.Background()
	notifier := &failingNotifier{err: errors.New("smtp down")}
	svc, cleanup := newTestServiceWithCustomNotifier(t, notifier)
	defer cleanup()

	delivery, err := svc.ForgotPassword(ctx, ForgotPasswordInput{Email: "missing@example.com"})
	if err != nil {
		t.Fatalf("ForgotPassword(unknown email) failed: %v", err)
	}
	if delivery.Token != "" || delivery.URL != "" {
		t.Fatalf("unknown email should return empty delivery: %#v", delivery)
	}
	if notifier.invitation != nil || notifier.passwordReset != nil {
		t.Fatalf("notifier should not be called for unknown email: %#v", notifier)
	}
}

func TestListOrganizationsFiltersAndPaginates(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	admin, err := svc.BootstrapAdmin(ctx, BootstrapAdminInput{
		OrgCode:  "core",
		OrgName:  "Core Org",
		Username: "admin",
		Email:    "admin@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("BootstrapAdmin() failed: %v", err)
	}
	if _, err := svc.CreateOrganization(ctx, *admin, "alpha", "Alpha Team"); err != nil {
		t.Fatalf("CreateOrganization(alpha) failed: %v", err)
	}
	if _, err := svc.CreateOrganization(ctx, *admin, "beta", "Beta Team"); err != nil {
		t.Fatalf("CreateOrganization(beta) failed: %v", err)
	}
	if _, err := svc.CreateOrganization(ctx, *admin, "support", "Support Desk"); err != nil {
		t.Fatalf("CreateOrganization(support) failed: %v", err)
	}

	firstPage, err := svc.ListOrganizations(ctx, *admin, OrganizationListFilter{
		Keyword:  "team",
		OrderKey: "code",
		Page:     1,
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("ListOrganizations(page 1) failed: %v", err)
	}
	if firstPage.Total != 2 || firstPage.Page != 1 || firstPage.PageSize != 1 || len(firstPage.Items) != 1 || firstPage.Items[0].Code != "alpha" || firstPage.StorageStatus != "persisted" {
		t.Fatalf("unexpected first page: %#v", firstPage)
	}

	secondPage, err := svc.ListOrganizations(ctx, *admin, OrganizationListFilter{
		Keyword:  "team",
		OrderKey: "code",
		Page:     2,
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("ListOrganizations(page 2) failed: %v", err)
	}
	if len(secondPage.Items) != 1 || secondPage.Items[0].Code != "beta" {
		t.Fatalf("unexpected second page: %#v", secondPage)
	}

	filtered, err := svc.ListOrganizations(ctx, *admin, OrganizationListFilter{
		Code:     "sup",
		Name:     "desk",
		Status:   model.StatusActive,
		OrderKey: "name",
		Desc:     true,
	})
	if err != nil {
		t.Fatalf("ListOrganizations(filtered) failed: %v", err)
	}
	if filtered.Total != 1 || len(filtered.Items) != 1 || filtered.Items[0].Code != "support" {
		t.Fatalf("unexpected filtered organizations: %#v", filtered)
	}
}

func TestInitialAdminSetupCreatesPlatformOwnerAndClosesSetup(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestServiceWithRegistrationMode(t, RegistrationModeDisabled)
	defer cleanup()

	status, err := svc.SetupStatus(ctx)
	if err != nil {
		t.Fatalf("SetupStatus() failed: %v", err)
	}
	if !status.Required {
		t.Fatal("SetupStatus().Required = false, want true for empty IAM users")
	}

	pair, err := svc.InitialAdminSetup(ctx, InitialAdminSetupInput{
		OrgCode:  "acme",
		OrgName:  "Acme",
		Username: "admin",
		Email:    "admin@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("InitialAdminSetup() failed: %v", err)
	}
	principal, err := svc.AuthenticateToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("AuthenticateToken(initial setup token) failed: %v", err)
	}
	for _, tc := range []struct {
		scope string
		obj   string
		act   string
	}{
		{scope: model.PermissionScopeTenant, obj: "audit", act: "read"},
		{scope: model.PermissionScopeTenant, obj: "session", act: "read"},
		{scope: model.PermissionScopePlatform, obj: "org", act: "read"},
		{scope: model.PermissionScopeTenant, obj: "user", act: "read"},
	} {
		allowed, err := svc.Authorize(ctx, principal, permissionContext(tc.scope, tc.obj, tc.act))
		if err != nil || !allowed {
			t.Fatalf("initial platform owner should %s:%s, allowed=%v err=%v", tc.obj, tc.act, allowed, err)
		}
	}
	token, err := svc.CreateAPIToken(ctx, CreateAPITokenInput{
		Principal: principal,
		UserID:    principal.UserID,
		RoleCode:  model.RolePlatformOwner,
		Days:      1,
	})
	if err != nil {
		t.Fatalf("CreateAPIToken(platform_owner) failed: %v", err)
	}
	if token.Token == "" || token.Item.RoleCode != model.RolePlatformOwner {
		t.Fatalf("unexpected platform owner token: %#v", token)
	}
	orgs, err := svc.ListMyOrganizations(ctx, principal)
	if err != nil || len(orgs) != 1 || orgs[0].Code != "acme" {
		t.Fatalf("unexpected setup organizations: %#v err=%v", orgs, err)
	}
	status, err = svc.SetupStatus(ctx)
	if err != nil {
		t.Fatalf("SetupStatus(after setup) failed: %v", err)
	}
	if status.Required {
		t.Fatal("SetupStatus().Required = true after initial setup, want false")
	}
	if _, err := svc.InitialAdminSetup(ctx, InitialAdminSetupInput{OrgCode: "other", OrgName: "Other", Username: "other", Email: "other@example.com", Password: "password123"}); !errors.Is(err, ErrSetupCompleted) {
		t.Fatalf("second InitialAdminSetup() error = %v, want ErrSetupCompleted", err)
	}
}

func TestSetupStatusIncludesPasswordPolicyAndPasswordErrorExplainsRules(t *testing.T) {
	ctx := context.Background()
	policy := PasswordPolicy{
		MinLength:     10,
		RequireLower:  true,
		RequireUpper:  true,
		RequireNumber: true,
	}
	svc, cleanup := newTestServiceWithRegistrationMode(t, RegistrationModeDisabled, func(cfg *Config) {
		cfg.PasswordPolicy = policy
	})
	defer cleanup()

	status, err := svc.SetupStatus(ctx)
	if err != nil {
		t.Fatalf("SetupStatus() failed: %v", err)
	}
	if status.PasswordPolicy.MinLength != 10 || !status.PasswordPolicy.RequireLower || !status.PasswordPolicy.RequireUpper || !status.PasswordPolicy.RequireNumber || status.PasswordPolicy.RequireSymbol {
		t.Fatalf("unexpected setup password policy: %#v", status.PasswordPolicy)
	}

	_, err = svc.InitialAdminSetup(ctx, InitialAdminSetupInput{
		OrgCode:  "acme",
		OrgName:  "Acme",
		Username: "admin",
		Email:    "admin@example.com",
		Password: "password123",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("InitialAdminSetup() error = %v, want ErrInvalidInput", err)
	}
	for _, want := range []string{"密码必须", "至少 10 位", "包含小写字母", "包含大写字母", "包含数字"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("password policy error missing %q: %v", want, err)
		}
	}
}

func TestSignupDisabled(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestServiceWithRegistrationMode(t, RegistrationModeDisabled)
	defer cleanup()

	_, err := svc.Signup(ctx, SignupInput{OrgCode: "acme", OrgName: "Acme", Username: "owner", Email: "owner@example.com", Password: "password123"})
	if err != ErrSignupDisabled {
		t.Fatalf("Signup() error = %v, want ErrSignupDisabled", err)
	}
}

func TestCreateOrganizationAddsCurrentUserAsOwner(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	admin, err := svc.BootstrapAdmin(ctx, BootstrapAdminInput{OrgCode: "acme", Username: "admin", Email: "admin@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("BootstrapAdmin() failed: %v", err)
	}
	login, err := svc.Login(ctx, LoginInput{Identifier: "admin@example.com", Password: "password123", OrgCode: "acme"})
	if err != nil {
		t.Fatalf("Login() failed: %v", err)
	}
	principal, err := svc.AuthenticateToken(ctx, login.AccessToken)
	if err != nil {
		t.Fatalf("AuthenticateToken() failed: %v", err)
	}
	org, err := svc.CreateOrganization(ctx, principal, "beta", "Beta")
	if err != nil {
		t.Fatalf("CreateOrganization() failed: %v", err)
	}
	switched, err := svc.SwitchOrg(ctx, principal, org.ID, "", "")
	if err != nil {
		t.Fatalf("SwitchOrg(created org) failed: %v", err)
	}
	newPrincipal, err := svc.AuthenticateToken(ctx, switched.AccessToken)
	if err != nil {
		t.Fatalf("AuthenticateToken(new org) failed: %v", err)
	}
	if newPrincipal.UserID != admin.UserID || newPrincipal.OrgID != org.ID {
		t.Fatalf("unexpected switched principal: %#v", newPrincipal)
	}
	allowed, err := svc.Authorize(ctx, newPrincipal, permissionContext(model.PermissionScopeTenant, "role", "create"))
	if err != nil || !allowed {
		t.Fatalf("created org owner should create roles, allowed=%v err=%v", allowed, err)
	}
}

func TestListUsersFiltersAndPaginates(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	admin, err := svc.BootstrapAdmin(ctx, BootstrapAdminInput{OrgCode: "acme", Username: "admin", Email: "admin@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("BootstrapAdmin() failed: %v", err)
	}
	for _, input := range []struct {
		email    string
		username string
		roleCode string
	}{
		{email: "alice@example.com", username: "alice", roleCode: model.RoleMember},
		{email: "bob@example.com", username: "bob", roleCode: model.RoleAdmin},
	} {
		invite, err := svc.InviteUser(ctx, InviteUserInput{Principal: *admin, Email: input.email, RoleCode: input.roleCode})
		if err != nil {
			t.Fatalf("InviteUser(%s) failed: %v", input.email, err)
		}
		if _, err := svc.AcceptInvitation(ctx, AcceptInvitationInput{Token: invite.Token, Username: input.username, Password: "password123"}); err != nil {
			t.Fatalf("AcceptInvitation(%s) failed: %v", input.email, err)
		}
	}

	all, err := svc.ListUsers(ctx, *admin, UserListFilter{Page: 1, PageSize: 2, Desc: true})
	if err != nil {
		t.Fatalf("ListUsers() failed: %v", err)
	}
	if all.Total != 3 || len(all.Items) != 2 || all.Page != 1 || all.PageSize != 2 {
		t.Fatalf("unexpected first page: %#v", all)
	}

	memberPage, err := svc.ListUsers(ctx, *admin, UserListFilter{RoleCode: model.RoleMember})
	if err != nil {
		t.Fatalf("ListUsers(member) failed: %v", err)
	}
	if memberPage.Total != 1 || memberPage.Items[0].User.Username != "alice" {
		t.Fatalf("unexpected member filter page: %#v", memberPage)
	}

	keywordPage, err := svc.ListUsers(ctx, *admin, UserListFilter{Keyword: "bob"})
	if err != nil {
		t.Fatalf("ListUsers(keyword) failed: %v", err)
	}
	if keywordPage.Total != 1 || keywordPage.Items[0].User.Email != "bob@example.com" {
		t.Fatalf("unexpected keyword filter page: %#v", keywordPage)
	}

	if _, err := svc.UpdateUser(ctx, UpdateUserInput{Principal: *admin, UserID: keywordPage.Items[0].User.ID, Status: ptrString(model.StatusDisabled)}); err != nil {
		t.Fatalf("UpdateUser(disable bob) failed: %v", err)
	}
	disabledPage, err := svc.ListUsers(ctx, *admin, UserListFilter{Status: model.StatusDisabled})
	if err != nil {
		t.Fatalf("ListUsers(disabled) failed: %v", err)
	}
	if disabledPage.Total != 1 || disabledPage.Items[0].User.Username != "bob" {
		t.Fatalf("unexpected disabled filter page: %#v", disabledPage)
	}
}

func TestListSessionsFiltersPaginatesAndScopesOrganization(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	admin, err := svc.BootstrapAdmin(ctx, BootstrapAdminInput{OrgCode: "acme", Username: "admin", Email: "admin@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("BootstrapAdmin() failed: %v", err)
	}
	adminLogin, err := svc.Login(ctx, LoginInput{Identifier: "admin@example.com", Password: "password123", OrgCode: "acme", IPAddress: "127.0.0.1", UserAgent: "Edge"})
	if err != nil {
		t.Fatalf("Login(admin) failed: %v", err)
	}
	principal, err := svc.AuthenticateToken(ctx, adminLogin.AccessToken)
	if err != nil {
		t.Fatalf("AuthenticateToken(admin) failed: %v", err)
	}
	invite, err := svc.InviteUser(ctx, InviteUserInput{Principal: principal, Email: "member@example.com", RoleCode: model.RoleMember})
	if err != nil {
		t.Fatalf("InviteUser() failed: %v", err)
	}
	if _, err := svc.AcceptInvitation(ctx, AcceptInvitationInput{Token: invite.Token, Username: "member", Password: "password123"}); err != nil {
		t.Fatalf("AcceptInvitation() failed: %v", err)
	}
	memberLogin, err := svc.Login(ctx, LoginInput{Identifier: "member@example.com", Password: "password123", OrgCode: "acme", IPAddress: "10.0.0.2", UserAgent: "Firefox", ProductCode: principal.ProductCode, ClientType: "mobile_web"})
	if err != nil {
		t.Fatalf("Login(member) failed: %v", err)
	}
	memberPrincipal, err := svc.AuthenticateToken(ctx, memberLogin.AccessToken)
	if err != nil {
		t.Fatalf("AuthenticateToken(member) failed: %v", err)
	}
	loginLogs, err := svc.ListAuditLogs(ctx, principal, AuditLogFilter{Action: "auth.login", Limit: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs(auth.login) failed: %v", err)
	}
	foundMemberLoginAudit := false
	for _, log := range loginLogs {
		if log.UserID != nil && *log.UserID == memberPrincipal.UserID {
			foundMemberLoginAudit = true
			if log.ProductCode != principal.ProductCode || log.ClientType != "mobile_web" {
				t.Fatalf("expected login audit to carry product and client type: %#v", log)
			}
		}
	}
	if !foundMemberLoginAudit {
		t.Fatalf("expected member login audit in logs: %#v", loginLogs)
	}
	beta, err := svc.CreateOrganization(ctx, principal, "beta", "Beta")
	if err != nil {
		t.Fatalf("CreateOrganization(beta) failed: %v", err)
	}
	if _, err := svc.SwitchOrg(ctx, principal, beta.ID, "Safari", "172.16.0.1"); err != nil {
		t.Fatalf("SwitchOrg(beta) failed: %v", err)
	}

	ownPage, err := svc.ListSessions(ctx, principal, SessionListFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListSessions(own) failed: %v", err)
	}
	if ownPage.Total != 1 || len(ownPage.Items) != 1 || ownPage.Items[0].UserID != admin.UserID || ownPage.Items[0].OrgID != principal.OrgID {
		t.Fatalf("unexpected own sessions: %#v", ownPage)
	}

	orgPage, err := svc.ListSessions(ctx, principal, SessionListFilter{Scope: "org", Keyword: "fire", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListSessions(org keyword) failed: %v", err)
	}
	if orgPage.Total != 1 || len(orgPage.Items) != 1 || orgPage.Items[0].UserID != memberPrincipal.UserID || orgPage.Items[0].OrgID != principal.OrgID {
		t.Fatalf("unexpected org keyword sessions: %#v", orgPage)
	}
	if orgPage.Items[0].ProductCode != principal.ProductCode || orgPage.Items[0].ClientType != "mobile_web" {
		t.Fatalf("expected member session to carry product and client type: %#v", orgPage.Items[0])
	}

	mobilePage, err := svc.ListSessions(ctx, principal, SessionListFilter{Scope: "org", ProductCode: principal.ProductCode, ClientType: "mobile_web", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListSessions(org platform) failed: %v", err)
	}
	if mobilePage.Total != 1 || len(mobilePage.Items) != 1 || mobilePage.Items[0].UserID != memberPrincipal.UserID {
		t.Fatalf("unexpected platform-filtered sessions: %#v", mobilePage)
	}

	platformKeywordPage, err := svc.ListSessions(ctx, principal, SessionListFilter{Scope: "org", Keyword: "mobile_web", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListSessions(org platform keyword) failed: %v", err)
	}
	if platformKeywordPage.Total != 1 || len(platformKeywordPage.Items) != 1 || platformKeywordPage.Items[0].UserID != memberPrincipal.UserID {
		t.Fatalf("unexpected platform keyword sessions: %#v", platformKeywordPage)
	}

	adminOrgSessions, err := svc.ListSessions(ctx, principal, SessionListFilter{UserID: admin.UserID, Scope: "org", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListSessions(admin user) failed: %v", err)
	}
	if adminOrgSessions.Total != 1 || len(adminOrgSessions.Items) != 1 || adminOrgSessions.Items[0].OrgID != principal.OrgID {
		t.Fatalf("expected beta session to be filtered out: %#v", adminOrgSessions)
	}

	if err := svc.RevokeSession(ctx, principal, adminOrgSessions.Items[0].ID); err != nil {
		t.Fatalf("RevokeSession() failed: %v", err)
	}
	revokedPage, err := svc.ListSessions(ctx, principal, SessionListFilter{Scope: "org", Status: "revoked"})
	if err != nil {
		t.Fatalf("ListSessions(revoked) failed: %v", err)
	}
	if revokedPage.Total != 1 || len(revokedPage.Items) != 1 || revokedPage.Items[0].ID != adminOrgSessions.Items[0].ID {
		t.Fatalf("unexpected revoked sessions: %#v", revokedPage)
	}
}

func TestMFASetupAndLogin(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()
	admin, err := svc.BootstrapAdmin(ctx, BootstrapAdminInput{OrgCode: "acme", Username: "admin", Email: "admin@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("BootstrapAdmin() failed: %v", err)
	}
	secret, _, err := svc.SetupMFA(ctx, *admin)
	if err != nil {
		t.Fatalf("SetupMFA() failed: %v", err)
	}
	oldCode := testsupport.IAMTOTPCode(t, secret, time.Now())
	secret, _, err = svc.SetupMFA(ctx, *admin)
	if err != nil {
		t.Fatalf("second SetupMFA() failed: %v", err)
	}
	if err := svc.VerifyMFA(ctx, *admin, oldCode); err == nil {
		t.Fatal("VerifyMFA() should reject code from replaced setup secret")
	}
	code := testsupport.IAMTOTPCode(t, secret, time.Now())
	if err := svc.VerifyMFA(ctx, *admin, code); err != nil {
		t.Fatalf("VerifyMFA() failed: %v", err)
	}
	if _, err := svc.Login(ctx, LoginInput{Identifier: "admin@example.com", Password: "password123", OrgCode: "acme"}); err != ErrMFARequired {
		t.Fatalf("expected ErrMFARequired, got %v", err)
	}
	code = testsupport.IAMTOTPCode(t, secret, time.Now())
	if _, err := svc.Login(ctx, LoginInput{Identifier: "admin@example.com", Password: "password123", OrgCode: "acme", MFACode: code}); err != nil {
		t.Fatalf("MFA login failed: %v", err)
	}
}

func TestLoginCaptchaWhenEnabled(t *testing.T) {
	svc, cleanup := newTestServiceWithRegistrationMode(t, RegistrationModeDirect, func(cfg *Config) {
		cfg.CaptchaEnabled = true
		cfg.CaptchaTTL = time.Minute
	})
	defer cleanup()
	ctx := context.Background()

	if _, err := svc.BootstrapAdmin(ctx, BootstrapAdminInput{OrgCode: "acme", Username: "admin", Email: "admin@example.com", Password: "password123"}); err != nil {
		t.Fatalf("BootstrapAdmin() failed: %v", err)
	}

	challenge, err := svc.Captcha(ctx)
	if err != nil {
		t.Fatalf("Captcha() error = %v", err)
	}
	if !challenge.Enabled || challenge.CaptchaID == "" || !strings.HasPrefix(challenge.Image, "data:image/svg+xml;base64,") {
		t.Fatalf("unexpected captcha challenge: %#v", challenge)
	}
	if _, err := svc.Login(ctx, LoginInput{Identifier: "admin@example.com", Password: "password123", OrgCode: "acme"}); !errors.Is(err, ErrCaptchaRequired) {
		t.Fatalf("expected captcha required error, got %v", err)
	}
	if _, err := svc.Login(ctx, LoginInput{CaptchaID: challenge.CaptchaID, CaptchaCode: "bad", Identifier: "admin@example.com", Password: "password123", OrgCode: "acme"}); !errors.Is(err, ErrCaptchaInvalid) {
		t.Fatalf("expected captcha invalid error, got %v", err)
	}

	challenge, err = svc.Captcha(ctx)
	if err != nil {
		t.Fatalf("Captcha() second error = %v", err)
	}
	answer := captchaAnswer(t, challenge.Image)
	if _, err := svc.Login(ctx, LoginInput{CaptchaID: challenge.CaptchaID, CaptchaCode: answer, Identifier: "admin@example.com", Password: "password123", OrgCode: "acme"}); err != nil {
		t.Fatalf("Login() with captcha failed: %v", err)
	}
}

func newTestService(t *testing.T) (Service, func()) {
	return newTestServiceWithRegistrationMode(t, RegistrationModeDirect)
}

type testServiceOption func(*Config)

func newTestServiceWithRegistrationMode(t *testing.T, registrationMode string, options ...testServiceOption) (Service, func()) {
	svc, _, _, cleanup := newTestServiceWithNotifier(t, registrationMode, NoopNotifier{}, options...)
	return svc, cleanup
}

func newTestServiceWithCustomNotifier(t *testing.T, notifier Notifier, options ...testServiceOption) (Service, func()) {
	svc, _, _, cleanup := newTestServiceWithNotifier(t, RegistrationModeDirect, notifier, options...)
	return svc, cleanup
}

func newTestServiceWithNotifier(t *testing.T, registrationMode string, notifier Notifier, options ...testServiceOption) (Service, Repository, testsupport.IAMDeps, func()) {
	t.Helper()
	moduleDB := testsupport.IAMSQLiteDatabase(t, "iam.db")
	deps := testsupport.NewIAMDeps(t)
	repo := repository.New(moduleDB)
	cfg := Config{
		RegistrationMode:     registrationMode,
		MFAIssuer:            "go-scaffold-test",
		MFASecretKey:         "01234567890123456789012345678901",
		LoginMaxFailures:     3,
		LoginLockDuration:    time.Minute,
		InvitationTTL:        time.Hour,
		EmailVerificationTTL: time.Hour,
		PasswordResetTTL:     time.Hour,
		NotificationDriver:   "debug",
		PublicBaseURL:        "/admin",
	}
	for _, option := range options {
		option(&cfg)
	}
	svc := New(repo, deps.Passwords, deps.Tokens, deps.Authz, deps.IDs, deps.TOTP, cfg, notifier)
	return svc, repo, deps, func() {}
}

type failingNotifier struct {
	err               error
	invitation        *InvitationNotice
	passwordReset     *PasswordResetNotice
	emailVerification *EmailVerificationNotice
}

type recordingNotifier struct {
	invitation        *InvitationNotice
	passwordReset     *PasswordResetNotice
	emailVerification *EmailVerificationNotice
}

func (n *recordingNotifier) SendInvitation(_ context.Context, notice InvitationNotice) error {
	noticeCopy := notice
	n.invitation = &noticeCopy
	return nil
}

func (n *recordingNotifier) SendPasswordReset(_ context.Context, notice PasswordResetNotice) error {
	noticeCopy := notice
	n.passwordReset = &noticeCopy
	return nil
}

func (n *recordingNotifier) SendEmailVerification(_ context.Context, notice EmailVerificationNotice) error {
	noticeCopy := notice
	n.emailVerification = &noticeCopy
	return nil
}

func (n *failingNotifier) SendInvitation(_ context.Context, notice InvitationNotice) error {
	noticeCopy := notice
	n.invitation = &noticeCopy
	return n.deliveryError()
}

func (n *failingNotifier) SendPasswordReset(_ context.Context, notice PasswordResetNotice) error {
	noticeCopy := notice
	n.passwordReset = &noticeCopy
	return n.deliveryError()
}

func (n *failingNotifier) SendEmailVerification(_ context.Context, notice EmailVerificationNotice) error {
	noticeCopy := notice
	n.emailVerification = &noticeCopy
	return n.deliveryError()
}

func (n *failingNotifier) deliveryError() error {
	if n.err != nil {
		return n.err
	}
	return errors.New("notification failed")
}

func captchaAnswer(t *testing.T, dataURL string) string {
	t.Helper()
	const prefix = "data:image/svg+xml;base64,"
	if !strings.HasPrefix(dataURL, prefix) {
		t.Fatalf("unexpected captcha data URL: %s", dataURL)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(dataURL, prefix))
	if err != nil {
		t.Fatalf("decode captcha SVG: %v", err)
	}
	matches := regexp.MustCompile(`([1-9]) \+ ([1-9])`).FindStringSubmatch(string(raw))
	if len(matches) != 3 {
		t.Fatalf("captcha SVG missing addition question: %s", string(raw))
	}
	left, _ := strconv.Atoi(matches[1])
	right, _ := strconv.Atoi(matches[2])
	return strconv.Itoa(left + right)
}

func ptrString(value string) *string {
	return &value
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

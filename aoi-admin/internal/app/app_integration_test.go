package app

// 本测试文件固定应用组装根的最小可启动契约，防止注释补全和后续重构改变外部可观察行为。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rei0721/go-scaffold/internal/config"
	iammodel "github.com/rei0721/go-scaffold/internal/modules/iam/model"
	iamservice "github.com/rei0721/go-scaffold/internal/modules/iam/service"
	systemmodel "github.com/rei0721/go-scaffold/internal/modules/system/model"
	systemservice "github.com/rei0721/go-scaffold/internal/modules/system/service"
)

// TestNewServerModeBuildsMinimalApplication 固定应用组装根的最小可启动契约，确保后续注释补全或结构调整不改变该场景。
func TestNewServerModeBuildsMinimalApplication(t *testing.T) {
	clearAppIntegrationEnv(t)

	configPath := writeAppIntegrationConfig(t, filepath.Join(t.TempDir(), "server-mode.db"))

	application, err := New(Options{ConfigPath: configPath, Mode: ModeServer})
	if err != nil {
		t.Fatalf("new server app: %v", err)
	}
	defer shutdownApp(t, application)

	if application.Core.Config == nil {
		t.Fatal("expected core config")
	}
	if application.Core.ConfigManager == nil {
		t.Fatal("expected core config manager")
	}
	if application.Core.Logger == nil {
		t.Fatal("expected core logger")
	}
	if application.Core.I18n == nil {
		t.Fatal("expected core i18n")
	}
	if application.Core.I18nUtils == nil {
		t.Fatal("expected core i18n utils")
	}
	if application.Core.IDGenerator == nil {
		t.Fatal("expected core id generator")
	}

	if application.Infra.Database == nil {
		t.Fatal("expected database infrastructure")
	}
	if application.Infra.Cache != nil {
		t.Fatal("expected redis cache to be disabled")
	}
	if application.Infra.Executor != nil {
		t.Fatal("expected executor to be disabled")
	}
	if application.Infra.Storage != nil {
		t.Fatal("expected storage to be disabled")
	}

	if application.Transport.Router == nil {
		t.Fatal("expected HTTP router")
	}
	if application.Transport.HTTPServer == nil {
		t.Fatal("expected HTTP server wrapper")
	}
	if application.Transport.RPCServer == nil {
		t.Fatal("expected RPC server wrapper")
	}

	if err := application.Core.ConfigManager.Update(func(cfg *config.Config) {
		cfg.Server.Port = 18082
		cfg.Server.ReadTimeout = 2
	}); err != nil {
		t.Fatalf("update config through manager: %v", err)
	}
	if got := application.Core.Config.Server.Port; got != 18082 {
		t.Fatalf("expected app hook to update core config port to 18082, got %d", got)
	}
	if got := application.Core.ConfigManager.Get().Server.Port; got != 18082 {
		t.Fatalf("expected manager config port to be 18082, got %d", got)
	}

	snapshot, err := application.Modules.System.Service.ListConfig(context.Background())
	if err != nil {
		t.Fatalf("list system config: %v", err)
	}
	value, ok := systemConfigValue(snapshot, "server.port")
	if !ok {
		t.Fatalf("expected system config to include server.port, got %#v", snapshot)
	}
	if value != 18082 {
		t.Fatalf("expected system config port to follow manager update, got %#v", value)
	}
	corsOriginsItem, ok := systemConfigItem(snapshot, "cors.allow_origins")
	if !ok || !corsOriginsItem.Editable || corsOriginsItem.ValueType != systemmodel.ConfigValueTypeArray {
		t.Fatalf("expected cors.allow_origins to be editable string array, got %#v ok=%v", corsOriginsItem, ok)
	}
	executorPoolSizeItem, ok := systemConfigItem(snapshot, "executor.pools.0.size")
	if !ok || !executorPoolSizeItem.Editable || executorPoolSizeItem.ValueType != systemmodel.ConfigValueTypeNumber {
		t.Fatalf("expected executor.pools.0.size to be editable number, got %#v ok=%v", executorPoolSizeItem, ok)
	}

	updated, err := application.Modules.System.Service.UpdateConfig(context.Background(), systemservice.UpdateConfigInput{
		Items: []systemservice.UpdateConfigItem{
			{Key: "server.port", Value: 18083},
			{Key: "database.mysql.password", Value: "runtime-secret"},
		},
	})
	if err != nil {
		t.Fatalf("update system config through service: %v", err)
	}
	if got := application.Core.ConfigManager.Get().Server.Port; got != 18083 {
		t.Fatalf("expected manager config port to be 18083, got %d", got)
	}
	value, ok = systemConfigValue(updated, "server.port")
	if !ok || value != 18083 {
		t.Fatalf("expected updated snapshot port 18083, got %#v ok=%v", value, ok)
	}
	value, ok = systemConfigValue(updated, "database.mysql.password")
	if !ok || value != "configured" {
		t.Fatalf("expected database.mysql.password to remain redacted, got %#v ok=%v", value, ok)
	}
	if got := application.Core.ConfigManager.Get().Database.MySQL.Password; got != "runtime-secret" {
		t.Fatalf("expected manager database password to be updated, got %q", got)
	}
	fileManager := config.NewManager()
	if err := fileManager.Load(configPath); err != nil {
		t.Fatalf("reload config file after runtime update: %v", err)
	}
	if got := fileManager.Get().Server.Port; got != 18081 {
		t.Fatalf("expected non-persist update to leave config file port 18081, got %d", got)
	}

	unchangedSecret, err := application.Modules.System.Service.UpdateConfig(context.Background(), systemservice.UpdateConfigInput{
		Items: []systemservice.UpdateConfigItem{
			{Key: "database.mysql.password", Value: ""},
		},
	})
	if err != nil {
		t.Fatalf("update empty secret through service: %v", err)
	}
	if got := application.Core.ConfigManager.Get().Database.MySQL.Password; got != "runtime-secret" {
		t.Fatalf("expected empty secret value to leave current password unchanged, got %q", got)
	}
	value, ok = systemConfigValue(unchangedSecret, "database.mysql.password")
	if !ok || value != "configured" {
		t.Fatalf("expected unchanged database.mysql.password to remain redacted, got %#v ok=%v", value, ok)
	}

	persisted, err := application.Modules.System.Service.UpdateConfig(context.Background(), systemservice.UpdateConfigInput{
		Items: []systemservice.UpdateConfigItem{
			{Key: "server.port", Value: 18084},
			{Key: "database.mysql.password", Value: "persistent-secret"},
			{Key: "cors.allow_origins", Value: []any{"https://admin.example.com", "https://app.example.com"}},
			{Key: "executor.pools.0.size", Value: 24},
		},
		Persist: true,
	})
	if err != nil {
		t.Fatalf("persist system config through service: %v", err)
	}
	value, ok = systemConfigValue(persisted, "server.port")
	if !ok || value != 18084 {
		t.Fatalf("expected persisted snapshot port 18084, got %#v ok=%v", value, ok)
	}
	value, ok = systemConfigValue(persisted, "database.mysql.password")
	if !ok || value != "configured" {
		t.Fatalf("expected persisted database.mysql.password to remain redacted, got %#v ok=%v", value, ok)
	}
	value, ok = systemConfigValue(persisted, "cors.allow_origins")
	if !ok || !reflect.DeepEqual(value, []string{"https://admin.example.com", "https://app.example.com"}) {
		t.Fatalf("expected persisted CORS origins in snapshot, got %#v ok=%v", value, ok)
	}
	value, ok = systemConfigValue(persisted, "executor.pools.0.size")
	if !ok || value != 24 {
		t.Fatalf("expected persisted executor pool size in snapshot, got %#v ok=%v", value, ok)
	}
	persistedContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read persisted config file: %v", err)
	}
	if text := string(persistedContent); !strings.Contains(text, "port: 18084") || !strings.Contains(text, `password: "persistent-secret"`) ||
		!strings.Contains(text, `- "https://admin.example.com"`) || !strings.Contains(text, `size: 24`) {
		t.Fatalf("expected persisted config file to include updated port and password, got:\n%s", text)
	}
	fileManager = config.NewManager()
	if err := fileManager.Load(configPath); err != nil {
		t.Fatalf("reload persisted config file: %v", err)
	}
	if got := fileManager.Get().Server.Port; got != 18084 {
		t.Fatalf("expected config file port 18084, got %d", got)
	}
	if got := fileManager.Get().Database.MySQL.Password; got != "persistent-secret" {
		t.Fatalf("expected config file password to be persistent-secret, got %q", got)
	}
	if !reflect.DeepEqual(fileManager.Get().CORS.AllowOrigins, []string{"https://admin.example.com", "https://app.example.com"}) {
		t.Fatalf("expected config file CORS origins to persist, got %#v", fileManager.Get().CORS.AllowOrigins)
	}
	if got := fileManager.Get().Executor.Pools[0].Size; got != 24 {
		t.Fatalf("expected config file executor pool size 24, got %d", got)
	}
}

func TestWebInitialSetupRunsSharedInitialization(t *testing.T) {
	clearAppIntegrationEnv(t)
	clearInitialSetupEnv(t)

	configPath := writeInitialSetupIntegrationConfig(t, filepath.Join(t.TempDir(), "web-setup.db"))
	application, err := New(Options{ConfigPath: configPath, Mode: ModeServer})
	if err != nil {
		t.Fatalf("new server app: %v", err)
	}
	defer shutdownApp(t, application)

	statusRecorder := performAppJSONRequest(application, http.MethodGet, "/api/v1/auth/setup/status", "", "")
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("setup status HTTP status = %d, body=%s", statusRecorder.Code, statusRecorder.Body.String())
	}
	var statusBody appAPIResponse[struct {
		Required       bool                      `json:"required"`
		PasswordPolicy iamservice.PasswordPolicy `json:"passwordPolicy"`
	}]
	if err := json.Unmarshal(statusRecorder.Body.Bytes(), &statusBody); err != nil {
		t.Fatalf("decode setup status: %v", err)
	}
	if !statusBody.Data.Required {
		t.Fatalf("setup status required = false, want true: %#v", statusBody)
	}
	if statusBody.Data.PasswordPolicy.MinLength != 8 {
		t.Fatalf("setup status password min length = %d, want 8: %#v", statusBody.Data.PasswordPolicy.MinLength, statusBody.Data.PasswordPolicy)
	}

	setupBody := `{"orgCode":"acme","orgName":"Acme Corp","username":"admin","displayName":"Admin","email":"admin@example.com","password":"password123"}`
	setupRecorder := performAppJSONRequest(application, http.MethodPost, "/api/v1/auth/setup/initial-admin", setupBody, "")
	if setupRecorder.Code != http.StatusOK {
		t.Fatalf("initial setup HTTP status = %d, body=%s", setupRecorder.Code, setupRecorder.Body.String())
	}
	var sessionBody appAPIResponse[iamservice.SessionSnapshot]
	if err := json.Unmarshal(setupRecorder.Body.Bytes(), &sessionBody); err != nil {
		t.Fatalf("decode setup session response: %v", err)
	}
	if sessionBody.Data.SessionID == 0 || sessionBody.Data.UserID == 0 || sessionBody.Data.ProductCode == "" || sessionBody.Data.ClientType == "" {
		t.Fatalf("setup did not return session snapshot: %#v", sessionBody.Data)
	}
	accessCookie := requireResponseCookie(t, setupRecorder, "aoi_access")
	requireResponseCookie(t, setupRecorder, "aoi_refresh")
	requireResponseCookie(t, setupRecorder, "aoi_csrf")

	principal, err := application.Modules.IAM.Service.AuthenticateToken(context.Background(), accessCookie.Value)
	if err != nil {
		t.Fatalf("authenticate setup token: %v", err)
	}
	for _, permission := range []struct {
		scope string
		obj   string
		act   string
	}{
		{scope: iammodel.PermissionScopeTenant, obj: "audit", act: "read"},
		{scope: iammodel.PermissionScopeTenant, obj: "session", act: "read"},
		{scope: iammodel.PermissionScopePlatform, obj: "server", act: "read"},
	} {
		allowed, err := application.Modules.IAM.Service.Authorize(context.Background(), principal, iamservice.PermissionContext{Scope: permission.scope, Object: permission.obj, Action: permission.act})
		if err != nil || !allowed {
			t.Fatalf("owner permission %s:%s allowed=%v err=%v", permission.obj, permission.act, allowed, err)
		}
	}

	for _, endpoint := range []string{
		fmt.Sprintf("/api/v1/orgs/%d/sessions?pageSize=6", principal.OrgID),
		fmt.Sprintf("/api/v1/orgs/%d/audit-logs?limit=6", principal.OrgID),
		"/api/v1/system/menus",
	} {
		recorder := performAppJSONRequest(application, http.MethodGet, endpoint, "", accessCookie.Value)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s HTTP status = %d, body=%s", endpoint, recorder.Code, recorder.Body.String())
		}
	}
}

func TestSetupCenterRunEndpointInitializesSystem(t *testing.T) {
	clearAppIntegrationEnv(t)
	clearInitialSetupEnv(t)

	configPath := writeInitialSetupIntegrationConfig(t, filepath.Join(t.TempDir(), "setup-center.db"))
	application, err := New(Options{ConfigPath: configPath, Mode: ModeServer})
	if err != nil {
		t.Fatalf("new server app: %v", err)
	}
	defer shutdownApp(t, application)

	statusRecorder := performAppJSONRequest(application, http.MethodGet, "/api/v1/setup/status", "", "")
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("setup center status HTTP status = %d, body=%s", statusRecorder.Code, statusRecorder.Body.String())
	}
	var statusBody appAPIResponse[struct {
		Required bool `json:"required"`
		Steps    []struct {
			Key    string `json:"key"`
			Status string `json:"status"`
		} `json:"steps"`
	}]
	if err := json.Unmarshal(statusRecorder.Body.Bytes(), &statusBody); err != nil {
		t.Fatalf("decode setup center status: %v", err)
	}
	if !statusBody.Data.Required {
		t.Fatalf("setup center status required = false, want true")
	}
	if !setupStepsContain(statusBody.Data.Steps, "database.migrate") || !setupStepsContain(statusBody.Data.Steps, "iam.owner") {
		t.Fatalf("setup center steps missing migration or iam owner: %#v", statusBody.Data.Steps)
	}

	setupBody := `{"orgCode":"acme","orgName":"Acme Corp","username":"admin","displayName":"Admin","email":"admin@example.com","password":"password123"}`
	setupRecorder := performAppJSONRequest(application, http.MethodPost, "/api/v1/setup/runs", setupBody, "")
	if setupRecorder.Code != http.StatusCreated {
		t.Fatalf("setup center run HTTP status = %d, body=%s", setupRecorder.Code, setupRecorder.Body.String())
	}
	var runBody appAPIResponse[struct {
		LoginTokensIssued bool                       `json:"loginTokensIssued"`
		LoginTokens       iamservice.SessionSnapshot `json:"loginTokens"`
		Run               struct {
			Status string `json:"status"`
		} `json:"run"`
		Steps []struct {
			Key    string `json:"key"`
			Status string `json:"status"`
		} `json:"steps"`
	}]
	if err := json.Unmarshal(setupRecorder.Body.Bytes(), &runBody); err != nil {
		t.Fatalf("decode setup center run: %v", err)
	}
	if runBody.Data.Run.Status != "succeeded" || !runBody.Data.LoginTokensIssued {
		t.Fatalf("setup center run did not succeed or issue tokens: %#v", runBody.Data)
	}
	if runBody.Data.LoginTokens.SessionID == 0 || runBody.Data.LoginTokens.ProductCode == "" || runBody.Data.LoginTokens.ClientType == "" {
		t.Fatalf("setup center run returned empty session snapshot: %#v", runBody.Data.LoginTokens)
	}
	requireResponseCookie(t, setupRecorder, "aoi_access")
	requireResponseCookie(t, setupRecorder, "aoi_refresh")
	requireResponseCookie(t, setupRecorder, "aoi_csrf")
	for _, key := range []string{"database.migrate", "system.seed", "catalog.sync", "iam.owner", "verify.finish"} {
		if !setupStepsContainStatus(runBody.Data.Steps, key, "succeeded") {
			t.Fatalf("setup center run step %s not succeeded: %#v", key, runBody.Data.Steps)
		}
	}
}

func TestAuthHTTPFlowMatchesReactContracts(t *testing.T) {
	clearAppIntegrationEnv(t)
	clearInitialSetupEnv(t)

	configPath := writeInitialSetupIntegrationConfig(t, filepath.Join(t.TempDir(), "auth-http-flow.db"))
	application, err := New(Options{ConfigPath: configPath, Mode: ModeServer})
	if err != nil {
		t.Fatalf("new server app: %v", err)
	}
	defer shutdownApp(t, application)

	localeHeader := map[string]string{"X-Locale": "en-US"}
	setupBody := appJSONBody(t, map[string]any{
		"orgCode":     "acme",
		"orgName":     "Acme Corp",
		"username":    "admin",
		"displayName": "Admin",
		"email":       "admin@example.com",
		"password":    "password123",
	})
	setupRecorder := performAppJSONRequestWithHeaders(application, http.MethodPost, "/api/v1/setup/runs", setupBody, "", localeHeader)
	if setupRecorder.Code != http.StatusCreated {
		t.Fatalf("setup center run HTTP status = %d, body=%s", setupRecorder.Code, setupRecorder.Body.String())
	}
	var setupRun appAPIResponse[struct {
		LoginTokensIssued bool                       `json:"loginTokensIssued"`
		LoginTokens       iamservice.SessionSnapshot `json:"loginTokens"`
	}]
	if err := json.Unmarshal(setupRecorder.Body.Bytes(), &setupRun); err != nil {
		t.Fatalf("decode setup run: %v", err)
	}
	if !setupRun.Data.LoginTokensIssued || setupRun.Data.LoginTokens.SessionID == 0 {
		t.Fatalf("setup run did not issue session snapshot: %#v", setupRun.Data)
	}

	loginBody := appJSONBody(t, map[string]any{
		"identifier": "admin@example.com",
		"password":   "password123",
		"orgCode":    "acme",
	})
	loginRecorder := performAppJSONRequestWithHeaders(application, http.MethodPost, "/api/v1/auth/login", loginBody, "", localeHeader)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("login HTTP status = %d, body=%s", loginRecorder.Code, loginRecorder.Body.String())
	}
	if strings.Contains(loginRecorder.Body.String(), "password123") {
		t.Fatalf("login response leaked submitted password: %s", loginRecorder.Body.String())
	}
	var loginResponse appAPIResponse[iamservice.SessionSnapshot]
	if err := json.Unmarshal(loginRecorder.Body.Bytes(), &loginResponse); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginResponse.Data.SessionID == 0 || loginResponse.Data.ProductCode == "" || loginResponse.Data.ClientType == "" {
		t.Fatalf("login response missing session snapshot: %#v", loginResponse.Data)
	}
	requireResponseCookie(t, loginRecorder, "aoi_access")
	loginRefreshCookie := requireResponseCookie(t, loginRecorder, "aoi_refresh")
	loginCSRFCookie := requireResponseCookie(t, loginRecorder, "aoi_csrf")

	refreshHeaders := map[string]string{
		"X-Locale":     "en-US",
		"X-CSRF-Token": loginCSRFCookie.Value,
	}
	refreshRecorder := performAppJSONRequestWithHeadersAndCookies(
		application,
		http.MethodPost,
		"/api/v1/auth/refresh",
		"",
		"",
		refreshHeaders,
		[]*http.Cookie{loginRefreshCookie, loginCSRFCookie},
	)
	if refreshRecorder.Code != http.StatusOK {
		t.Fatalf("refresh HTTP status = %d, body=%s", refreshRecorder.Code, refreshRecorder.Body.String())
	}
	var refreshResponse appAPIResponse[iamservice.SessionSnapshot]
	if err := json.Unmarshal(refreshRecorder.Body.Bytes(), &refreshResponse); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if refreshResponse.Data.SessionID == 0 || refreshResponse.Data.SessionID != loginResponse.Data.SessionID {
		t.Fatalf("refresh response returned unexpected session snapshot: %#v", refreshResponse.Data)
	}
	refreshAccessCookie := requireResponseCookie(t, refreshRecorder, "aoi_access")
	refreshRefreshCookie := requireResponseCookie(t, refreshRecorder, "aoi_refresh")
	if refreshAccessCookie.Value == "" || refreshRefreshCookie.Value == "" {
		t.Fatalf("refresh did not set auth cookies")
	}

	principal, err := application.Modules.IAM.Service.AuthenticateToken(context.Background(), refreshAccessCookie.Value)
	if err != nil {
		t.Fatalf("authenticate refreshed owner token: %v", err)
	}
	inviteBody := appJSONBody(t, map[string]any{
		"email":    "member@example.com",
		"roleCode": iammodel.RoleMember,
	})
	invitePath := fmt.Sprintf("/api/v1/orgs/%d/users/invitations", principal.OrgID)
	inviteRecorder := performAppJSONRequestWithHeaders(application, http.MethodPost, invitePath, inviteBody, refreshAccessCookie.Value, localeHeader)
	if inviteRecorder.Code != http.StatusCreated {
		t.Fatalf("invite HTTP status = %d, body=%s", inviteRecorder.Code, inviteRecorder.Body.String())
	}
	var inviteResponse appAPIResponse[iamservice.NotificationDelivery]
	if err := json.Unmarshal(inviteRecorder.Body.Bytes(), &inviteResponse); err != nil {
		t.Fatalf("decode invite response: %v", err)
	}
	if !inviteResponse.Data.Debug || inviteResponse.Data.Token == "" || inviteResponse.Data.URL == "" {
		t.Fatalf("expected debug invitation delivery, got %#v", inviteResponse.Data)
	}

	legacyInviteRecorder := performAppJSONRequestWithHeaders(application, http.MethodPost, fmt.Sprintf("/api/v1/orgs/%d/invitations", principal.OrgID), inviteBody, refreshAccessCookie.Value, localeHeader)
	if legacyInviteRecorder.Code == http.StatusOK || legacyInviteRecorder.Code == http.StatusCreated {
		t.Fatalf("legacy invitation create path unexpectedly succeeded: status=%d body=%s", legacyInviteRecorder.Code, legacyInviteRecorder.Body.String())
	}

	acceptBody := appJSONBody(t, map[string]any{
		"username":    "member",
		"displayName": "Member User",
		"password":    "password123",
	})
	acceptPath := fmt.Sprintf("/api/v1/invitations/%s/accept", inviteResponse.Data.Token)
	acceptRecorder := performAppJSONRequestWithHeaders(application, http.MethodPost, acceptPath, acceptBody, "", localeHeader)
	if acceptRecorder.Code != http.StatusCreated {
		t.Fatalf("accept invitation HTTP status = %d, body=%s", acceptRecorder.Code, acceptRecorder.Body.String())
	}
	if strings.Contains(acceptRecorder.Body.String(), "password123") {
		t.Fatalf("accept invitation response leaked submitted password: %s", acceptRecorder.Body.String())
	}
	var accepted appAPIResponse[iamservice.Principal]
	if err := json.Unmarshal(acceptRecorder.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode accepted invitation response: %v", err)
	}
	if accepted.Data.UserID == 0 || accepted.Data.OrgID != principal.OrgID || accepted.Data.Email != "member@example.com" {
		t.Fatalf("unexpected accepted principal: %#v", accepted.Data)
	}

	forgotBody := appJSONBody(t, map[string]any{"email": "member@example.com"})
	forgotRecorder := performAppJSONRequestWithHeaders(application, http.MethodPost, "/api/v1/auth/password/forgot", forgotBody, "", localeHeader)
	if forgotRecorder.Code != http.StatusOK {
		t.Fatalf("forgot password HTTP status = %d, body=%s", forgotRecorder.Code, forgotRecorder.Body.String())
	}
	var forgotResponse appAPIResponse[iamservice.NotificationDelivery]
	if err := json.Unmarshal(forgotRecorder.Body.Bytes(), &forgotResponse); err != nil {
		t.Fatalf("decode forgot password response: %v", err)
	}
	if !forgotResponse.Data.Debug || forgotResponse.Data.Token == "" || forgotResponse.Data.URL == "" {
		t.Fatalf("expected debug password reset delivery, got %#v", forgotResponse.Data)
	}

	resetBody := appJSONBody(t, map[string]any{
		"token":       forgotResponse.Data.Token,
		"newPassword": "newpassword123",
	})
	resetRecorder := performAppJSONRequestWithHeaders(application, http.MethodPost, "/api/v1/auth/password/reset", resetBody, "", localeHeader)
	if resetRecorder.Code != http.StatusOK {
		t.Fatalf("reset password HTTP status = %d, body=%s", resetRecorder.Code, resetRecorder.Body.String())
	}
	var resetResponse appAPIResponse[struct {
		Reset bool `json:"reset"`
	}]
	if err := json.Unmarshal(resetRecorder.Body.Bytes(), &resetResponse); err != nil {
		t.Fatalf("decode reset password response: %v", err)
	}
	if !resetResponse.Data.Reset {
		t.Fatalf("reset password response missing reset=true: %#v", resetResponse.Data)
	}

	oldPasswordRecorder := performAppJSONRequestWithHeaders(application, http.MethodPost, "/api/v1/auth/login", appJSONBody(t, map[string]any{
		"identifier": "member@example.com",
		"password":   "password123",
		"orgCode":    "acme",
	}), "", localeHeader)
	if oldPasswordRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("member old password login HTTP status = %d, want 401 body=%s", oldPasswordRecorder.Code, oldPasswordRecorder.Body.String())
	}
	memberLoginRecorder := performAppJSONRequestWithHeaders(application, http.MethodPost, "/api/v1/auth/login", appJSONBody(t, map[string]any{
		"identifier": "member@example.com",
		"password":   "newpassword123",
		"orgCode":    "acme",
	}), "", localeHeader)
	if memberLoginRecorder.Code != http.StatusOK {
		t.Fatalf("member new password login HTTP status = %d, body=%s", memberLoginRecorder.Code, memberLoginRecorder.Body.String())
	}
	var memberLogin appAPIResponse[iamservice.SessionSnapshot]
	if err := json.Unmarshal(memberLoginRecorder.Body.Bytes(), &memberLogin); err != nil {
		t.Fatalf("decode member login response: %v", err)
	}
	if memberLogin.Data.SessionID == 0 || memberLogin.Data.ProductCode == "" || memberLogin.Data.ClientType == "" {
		t.Fatalf("member login response missing session snapshot: %#v", memberLogin.Data)
	}
	requireResponseCookie(t, memberLoginRecorder, "aoi_access")
	requireResponseCookie(t, memberLoginRecorder, "aoi_refresh")
}

func TestSetupCenterSchemaExposesReactWizardContract(t *testing.T) {
	clearAppIntegrationEnv(t)
	clearInitialSetupEnv(t)

	configPath := writeInitialSetupIntegrationConfig(t, filepath.Join(t.TempDir(), "setup-schema.db"))
	application, err := New(Options{ConfigPath: configPath, Mode: ModeServer})
	if err != nil {
		t.Fatalf("new server app: %v", err)
	}
	defer shutdownApp(t, application)

	recorder := performAppJSONRequestWithHeaders(application, http.MethodGet, "/api/v1/setup/schema", "", "", map[string]string{
		"X-Locale": "en-US",
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("setup schema HTTP status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var schemaBody appAPIResponse[setupSchemaHTTPBody]
	if err := json.Unmarshal(recorder.Body.Bytes(), &schemaBody); err != nil {
		t.Fatalf("decode setup schema: %v", err)
	}
	steps := schemaBody.Data.Steps
	if len(steps) == 0 {
		t.Fatal("setup schema returned no steps")
	}

	assertSetupStepBefore(t, steps, "storage.configure", "database.configure")
	assertSetupStepBefore(t, steps, "database.configure", "cache.configure")
	assertSetupStepBefore(t, steps, "cache.configure", "iam.owner")
	assertSetupStepBefore(t, steps, "iam.owner", "site.configure")

	storage := requireSetupStep(t, steps, "storage.configure")
	if storage.RouteSlug != "storage" || storage.Phase != "storage" || storage.Required || !storage.Skippable || !storage.Testable {
		t.Fatalf("storage.configure contract drifted: %#v", storage)
	}
	storageDriver := requireSetupField(t, storage, "storage.driver")
	if storageDriver.Label != "Storage driver" || storageDriver.ConfigPath != "storage.driver" || storageDriver.Type != "select" {
		t.Fatalf("storage.driver field contract drifted: %#v", storageDriver)
	}
	if got := setupOptionLabel(storageDriver.Options, "local"); got != "Local storage" {
		t.Fatalf("storage.driver local option label = %q, want Local storage", got)
	}

	database := requireSetupStep(t, steps, "database.configure")
	if database.RouteSlug != "database" || database.Phase != "database" || !database.Required || database.Skippable || !database.Testable {
		t.Fatalf("database.configure contract drifted: %#v", database)
	}
	sqlitePath := requireSetupField(t, database, "database.sqlite.path")
	if sqlitePath.ConfigPath != "database.sqlite.path" || sqlitePath.Type != "text" || !sqlitePath.Required {
		t.Fatalf("database.sqlite.path field contract drifted: %#v", sqlitePath)
	}

	cache := requireSetupStep(t, steps, "cache.configure")
	if cache.RouteSlug != "cache" || cache.Phase != "cache" || cache.Required || !cache.Skippable || !cache.Testable {
		t.Fatalf("cache.configure contract drifted: %#v", cache)
	}
	if !setupStringSliceContains(cache.Dependencies, "database.configure") {
		t.Fatalf("cache.configure dependencies = %v, want database.configure", cache.Dependencies)
	}

	owner := requireSetupStep(t, steps, "iam.owner")
	if owner.RouteSlug != "owner" || owner.Phase != "iam" || !owner.Required || owner.Skippable || owner.Testable {
		t.Fatalf("iam.owner contract drifted: %#v", owner)
	}
	if !setupStringSliceContains(owner.Dependencies, "catalog.sync") {
		t.Fatalf("iam.owner dependencies = %v, want catalog.sync", owner.Dependencies)
	}
	password := requireSetupField(t, owner, "password")
	if password.Label != "Admin password" || password.ConfigPath != "" || password.Type != "password" || !password.Sensitive || !password.Required {
		t.Fatalf("owner password field contract drifted: %#v", password)
	}

	site := requireSetupStep(t, steps, "site.configure")
	if site.RouteSlug != "site" || site.Phase != "site" || !site.Required || site.Skippable || !site.Testable {
		t.Fatalf("site.configure contract drifted: %#v", site)
	}
	if !setupStringSliceContains(site.Dependencies, "iam.owner") {
		t.Fatalf("site.configure dependencies = %v, want iam.owner", site.Dependencies)
	}
	productName := requireSetupField(t, site, "brand.productName")
	if productName.Label != "Product name" || productName.ConfigPath != "brand.productName" || productName.Type != "text" || !productName.Required {
		t.Fatalf("brand.productName field contract drifted: %#v", productName)
	}
	publicBaseURL := requireSetupField(t, site, "webui.public_base_url")
	if publicBaseURL.ConfigPath != "webui.public_base_url" || publicBaseURL.Required {
		t.Fatalf("webui.public_base_url field contract drifted: %#v", publicBaseURL)
	}
}

func systemConfigValue(snapshot systemmodel.ConfigSnapshot, key string) (any, bool) {
	item, ok := systemConfigItem(snapshot, key)
	if !ok {
		return nil, false
	}
	return item.Value, true
}

func setupStepsContain(steps []struct {
	Key    string `json:"key"`
	Status string `json:"status"`
}, key string) bool {
	for _, step := range steps {
		if step.Key == key {
			return true
		}
	}
	return false
}

func setupStepsContainStatus(steps []struct {
	Key    string `json:"key"`
	Status string `json:"status"`
}, key string, status string) bool {
	for _, step := range steps {
		if step.Key == key && step.Status == status {
			return true
		}
	}
	return false
}

type setupSchemaHTTPBody struct {
	Steps []setupStepSchemaHTTP `json:"steps"`
}

type setupStepSchemaHTTP struct {
	Key          string                 `json:"key"`
	RouteSlug    string                 `json:"routeSlug"`
	Phase        string                 `json:"phase"`
	Title        string                 `json:"title"`
	Order        int                    `json:"order"`
	Required     bool                   `json:"required"`
	Skippable    bool                   `json:"skippable"`
	Testable     bool                   `json:"testable"`
	Dependencies []string               `json:"dependencies"`
	Fields       []setupFieldSchemaHTTP `json:"fields"`
	Groups       []setupFieldGroupHTTP  `json:"groups"`
}

type setupFieldGroupHTTP struct {
	Key    string                 `json:"key"`
	Fields []setupFieldSchemaHTTP `json:"fields"`
}

type setupFieldSchemaHTTP struct {
	Key        string            `json:"key"`
	Label      string            `json:"label"`
	Type       string            `json:"type"`
	Required   bool              `json:"required"`
	Sensitive  bool              `json:"sensitive"`
	Options    []setupOptionHTTP `json:"options"`
	ConfigPath string            `json:"configPath"`
}

type setupOptionHTTP struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func assertSetupStepBefore(t *testing.T, steps []setupStepSchemaHTTP, left string, right string) {
	t.Helper()
	leftIndex := setupStepIndex(steps, left)
	rightIndex := setupStepIndex(steps, right)
	if leftIndex < 0 || rightIndex < 0 {
		t.Fatalf("missing setup step %s=%d %s=%d", left, leftIndex, right, rightIndex)
	}
	if leftIndex >= rightIndex {
		t.Fatalf("setup step %s index=%d must be before %s index=%d; steps=%v", left, leftIndex, right, rightIndex, setupHTTPStepKeys(steps))
	}
}

func requireSetupStep(t *testing.T, steps []setupStepSchemaHTTP, key string) setupStepSchemaHTTP {
	t.Helper()
	for _, step := range steps {
		if step.Key == key {
			return step
		}
	}
	t.Fatalf("setup schema missing step %s; steps=%v", key, setupHTTPStepKeys(steps))
	return setupStepSchemaHTTP{}
}

func requireSetupField(t *testing.T, step setupStepSchemaHTTP, key string) setupFieldSchemaHTTP {
	t.Helper()
	for _, field := range step.Fields {
		if field.Key == key {
			return field
		}
	}
	for _, group := range step.Groups {
		for _, field := range group.Fields {
			if field.Key == key {
				return field
			}
		}
	}
	t.Fatalf("setup step %s missing field %s", step.Key, key)
	return setupFieldSchemaHTTP{}
}

func setupStepIndex(steps []setupStepSchemaHTTP, key string) int {
	for index, step := range steps {
		if step.Key == key {
			return index
		}
	}
	return -1
}

func setupHTTPStepKeys(steps []setupStepSchemaHTTP) []string {
	out := make([]string, 0, len(steps))
	for _, step := range steps {
		out = append(out, step.Key)
	}
	return out
}

func setupOptionLabel(options []setupOptionHTTP, value string) string {
	for _, option := range options {
		if option.Value == value {
			return option.Label
		}
	}
	return ""
}

func setupStringSliceContains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func systemConfigItem(snapshot systemmodel.ConfigSnapshot, key string) (systemmodel.ConfigItem, bool) {
	for _, section := range snapshot.Sections {
		for _, item := range section.Items {
			if item.Key == key {
				return item, true
			}
		}
	}
	return systemmodel.ConfigItem{}, false
}

// writeAppIntegrationConfig 写入测试夹具文件，并把文件系统准备细节限制在测试辅助层。
func writeAppIntegrationConfig(t *testing.T, dbPath string) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	localesDir := filepath.Join(repoRoot, "configs", "locales")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	storagePath := filepath.Join(t.TempDir(), "storage")

	content := fmt.Sprintf(`server:
  host: "127.0.0.1"
  port: 18081
  mode: "test"
  read_timeout: 1
  write_timeout: 1
  idle_timeout: 1
database:
  driver: "sqlite"
  sqlite:
    path: %s
  mysql:
    host: "127.0.0.1"
    port: 3306
    username: "root"
    password: ""
    database: "aoi_admin"
    charset: "utf8mb4"
  postgres:
    host: "127.0.0.1"
    port: 5432
    username: "postgres"
    password: ""
    database: "aoi_admin"
    sslMode: "disable"
  pool:
    maxOpenConns: 1
    maxIdleConns: 1
cache:
  driver: "disabled"
  local:
    maxCost: 0
    numCounters: 0
    bufferItems: 0
    defaultTtlSeconds: 0
  redis:
    addr: "127.0.0.1:6379"
    username: ""
    password: ""
    db: 0
    poolSize: 1
    minIdleConns: 0
    maxRetries: 0
    dialTimeout: 1
    readTimeout: 1
    writeTimeout: 1
logger:
  level: "error"
  format: "console"
  output: "stdout"
  file_path: ""
  max_size: 1
  max_backups: 1
  max_age: 1
i18n:
  defaultLocale: "zh-CN"
  fallbackLocale: "zh-CN"
  supportedLocales:
    - "zh-CN"
    - "en-US"
  resources:
    ui: %s
    api: %s
    validation: %s
    system: %s
brand:
  productName: "Aoi Admin"
  productCode: "aoi-admin"
  versionName: "Community"
executor:
  enabled: false
  pools:
    - name: "default"
      size: 8
      expiry: 30
      non_blocking: true
storage:
  driver: "disabled"
  local:
    fsType: "memory"
    basePath: %s
    publicUrl: "/uploads"
    enableWatch: false
    watchBufferSize: 1
  s3:
    endpoint: "https://s3.example.com"
    region: "us-east-1"
    bucket: "aoi-admin"
    accessKeyId: "example"
    secretAccessKey: "example"
    usePathStyle: true
    publicBaseUrl: ""
  minio:
    endpoint: "http://127.0.0.1:9000"
    region: "us-east-1"
    bucket: "aoi-admin"
    accessKeyId: "example"
    secretAccessKey: "example"
    usePathStyle: true
    publicBaseUrl: ""
cors:
  enabled: true
  allow_origins:
    - "*"
  allow_methods:
    - "GET"
    - "POST"
    - "PUT"
    - "DELETE"
    - "PATCH"
    - "OPTIONS"
  allow_headers:
    - "Origin"
    - "Content-Type"
    - "X-Request-ID"
  expose_headers:
    - "X-Request-ID"
  allow_credentials: false
  max_age: 60
rpc:
  enabled: false
  host: "127.0.0.1"
  port: 10099
  read_timeout: 1
  write_timeout: 1
  idle_timeout: 1
`, yamlString(dbPath),
		yamlString(filepath.Join(localesDir, "ui")),
		yamlString(filepath.Join(localesDir, "api")),
		yamlString(filepath.Join(localesDir, "validation")),
		yamlString(filepath.Join(localesDir, "system")),
		yamlString(storagePath))

	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("write test config: %v", err)
	}
	return configPath
}

func writeInitialSetupIntegrationConfig(t *testing.T, dbPath string) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	localesDir := filepath.Join(repoRoot, "configs", "locales")
	migrationsDir := filepath.Join(repoRoot, "internal", "migrations")
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	content := fmt.Sprintf(`server:
  host: "127.0.0.1"
  port: 18081
  mode: "test"
  read_timeout: 1
  write_timeout: 1
  idle_timeout: 1
webui:
  enabled: false
database:
  driver: "sqlite"
  sqlite:
    path: %s
  mysql:
    host: "127.0.0.1"
    port: 3306
    username: "root"
    password: ""
    database: "aoi_admin"
    charset: "utf8mb4"
  postgres:
    host: "127.0.0.1"
    port: 5432
    username: "postgres"
    password: ""
    database: "aoi_admin"
    sslMode: "disable"
  pool:
    maxOpenConns: 1
    maxIdleConns: 1
cache:
  driver: "disabled"
logger:
  level: "error"
  format: "console"
  output: "stdout"
  file_path: ""
  max_size: 1
  max_backups: 1
  max_age: 1
i18n:
  defaultLocale: "zh-CN"
  fallbackLocale: "zh-CN"
  supportedLocales:
    - "zh-CN"
    - "en-US"
  resources:
    ui: %s
    api: %s
    validation: %s
    system: %s
brand:
  productName: "Aoi Admin"
  productCode: "aoi-admin"
  versionName: "Community"
executor:
  enabled: false
storage:
  driver: "disabled"
  local:
    fsType: "memory"
    basePath: "./data/uploads"
    publicUrl: "/uploads"
    enableWatch: false
    watchBufferSize: 1
cors:
  enabled: true
  allow_origins:
    - "*"
  allow_methods:
    - "GET"
    - "POST"
    - "PATCH"
    - "OPTIONS"
  allow_headers:
    - "Origin"
    - "Content-Type"
    - "Authorization"
    - "X-Request-ID"
  expose_headers:
    - "X-Request-ID"
  allow_credentials: false
  max_age: 60
rpc:
  enabled: false
  host: "127.0.0.1"
  port: 10099
  read_timeout: 1
  write_timeout: 1
  idle_timeout: 1
plugins:
  enabled: false
system:
  seed_defaults_on_start: false
auth:
  enabled: true
  registration_mode: disabled
  issuer: "aoi-admin"
  audience:
    - "aoi-admin-api"
  signing_key: "12345678901234567890123456789012"
  refresh_token_pepper: "pepper"
  mfa_secret_key: "12345678901234567890123456789012"
  notification_driver: "debug"
  password_policy:
    min_length: 8
migration:
  auto_apply: false
  dir: %s
`, yamlString(dbPath),
		yamlString(filepath.Join(localesDir, "ui")),
		yamlString(filepath.Join(localesDir, "api")),
		yamlString(filepath.Join(localesDir, "validation")),
		yamlString(filepath.Join(localesDir, "system")),
		yamlString(migrationsDir))

	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("write setup test config: %v", err)
	}
	return configPath
}

type appAPIResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func appJSONBody(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	return string(raw)
}

func performAppJSONRequest(application *App, method string, path string, body string, accessToken string) *httptest.ResponseRecorder {
	return performAppJSONRequestWithHeaders(application, method, path, body, accessToken, nil)
}

func performAppJSONRequestWithHeaders(application *App, method string, path string, body string, accessToken string, headers map[string]string) *httptest.ResponseRecorder {
	return performAppJSONRequestWithHeadersAndCookies(application, method, path, body, accessToken, headers, nil)
}

func performAppJSONRequestWithHeadersAndCookies(application *App, method string, path string, body string, accessToken string, headers map[string]string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	request := httptest.NewRequest(method, path, reader)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		request.Header.Set("Authorization", "Bearer "+accessToken)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	for _, cookie := range cookies {
		if cookie != nil {
			request.AddCookie(cookie)
		}
	}
	recorder := httptest.NewRecorder()
	application.Transport.Router.ServeHTTP(recorder, request)
	return recorder
}

func requireResponseCookie(t *testing.T, recorder *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == name && cookie.Value != "" {
			return cookie
		}
	}
	t.Fatalf("response missing cookie %q; set-cookie=%v body=%s", name, recorder.Result().Header.Values("Set-Cookie"), recorder.Body.String())
	return nil
}

// shutdownApp 是当前测试文件的辅助函数，用于复用夹具、断言或输入构造逻辑。
func shutdownApp(t *testing.T, application *App) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := application.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown app: %v", err)
	}
}

// yamlString 是当前测试文件的辅助函数，用于复用夹具、断言或输入构造逻辑。
func yamlString(value string) string {
	return strconv.Quote(filepath.ToSlash(value))
}

// clearAppIntegrationEnv 清理测试期间设置的环境变量或全局状态，避免用例之间互相污染。
func clearAppIntegrationEnv(t *testing.T) {
	t.Helper()

	keys := []string{
		"DB_DRIVER",
		"DB_SQLITE_PATH",
		"DB_MYSQL_HOST",
		"DB_MYSQL_PORT",
		"DB_MYSQL_USERNAME",
		"DB_MYSQL_PASSWORD",
		"DB_MYSQL_DATABASE",
		"DB_MYSQL_CHARSET",
		"DB_POSTGRES_HOST",
		"DB_POSTGRES_PORT",
		"DB_POSTGRES_USERNAME",
		"DB_POSTGRES_PASSWORD",
		"DB_POSTGRES_DATABASE",
		"DB_POSTGRES_SSL_MODE",
		"DB_POOL_MAX_OPEN_CONNS",
		"DB_POOL_MAX_IDLE_CONNS",
		"REI_APP_DB_DRIVER",
		"REI_APP_DB_SQLITE_PATH",
		"REI_APP_DB_MYSQL_HOST",
		"REI_APP_DB_MYSQL_PORT",
		"REI_APP_DB_MYSQL_USERNAME",
		"REI_APP_DB_MYSQL_PASSWORD",
		"REI_APP_DB_MYSQL_DATABASE",
		"REI_APP_DB_MYSQL_CHARSET",
		"REI_APP_DB_POSTGRES_HOST",
		"REI_APP_DB_POSTGRES_PORT",
		"REI_APP_DB_POSTGRES_USERNAME",
		"REI_APP_DB_POSTGRES_PASSWORD",
		"REI_APP_DB_POSTGRES_DATABASE",
		"REI_APP_DB_POSTGRES_SSL_MODE",
		"REI_APP_DB_POOL_MAX_OPEN_CONNS",
		"REI_APP_DB_POOL_MAX_IDLE_CONNS",
		"CACHE_DRIVER",
		"CACHE_REDIS_ADDR",
		"CACHE_REDIS_USERNAME",
		"CACHE_REDIS_PASSWORD",
		"CACHE_REDIS_DB",
		"CACHE_REDIS_POOL_SIZE",
		"CACHE_REDIS_MIN_IDLE_CONNS",
		"CACHE_REDIS_MAX_RETRIES",
		"CACHE_REDIS_DIAL_TIMEOUT",
		"CACHE_REDIS_READ_TIMEOUT",
		"CACHE_REDIS_WRITE_TIMEOUT",
		"SERVER_PORT",
		"SERVER_MODE",
		"SERVER_READ_TIMEOUT",
		"SERVER_WRITE_TIMEOUT",
		"LOG_LEVEL",
		"LOG_FORMAT",
		"LOG_OUTPUT",
		"I18N_DEFAULT",
		"I18N_SUPPORTED",
		"STORAGE_DRIVER",
		"STORAGE_LOCAL_FS_TYPE",
		"STORAGE_LOCAL_BASE_PATH",
		"STORAGE_LOCAL_PUBLIC_URL",
		"STORAGE_LOCAL_ENABLE_WATCH",
		"STORAGE_LOCAL_WATCH_BUFFER_SIZE",
		"STORAGE_S3_ENDPOINT",
		"STORAGE_S3_REGION",
		"STORAGE_S3_BUCKET",
		"STORAGE_S3_ACCESS_KEY_ID",
		"STORAGE_S3_SECRET_ACCESS_KEY",
		"STORAGE_S3_USE_PATH_STYLE",
		"STORAGE_S3_PUBLIC_BASE_URL",
		"STORAGE_MINIO_ENDPOINT",
		"STORAGE_MINIO_REGION",
		"STORAGE_MINIO_BUCKET",
		"STORAGE_MINIO_ACCESS_KEY_ID",
		"STORAGE_MINIO_SECRET_ACCESS_KEY",
		"STORAGE_MINIO_USE_PATH_STYLE",
		"STORAGE_MINIO_PUBLIC_BASE_URL",
		"CORS_ENABLED",
		"CORS_ALLOW_ORIGINS",
		"CORS_ALLOW_METHODS",
		"CORS_ALLOW_HEADERS",
		"CORS_EXPOSE_HEADERS",
		"CORS_ALLOW_CREDENTIALS",
		"CORS_MAX_AGE",
		"RPC_ENABLED",
		"RPC_HOST",
		"RPC_PORT",
		"RPC_READ_TIMEOUT",
		"RPC_WRITE_TIMEOUT",
		"RPC_IDLE_TIMEOUT",
	}
	for _, key := range keys {
		t.Setenv(key, "")
		t.Setenv(config.EnvPrefixJoin(key), "")
	}
}

func clearInitialSetupEnv(t *testing.T) {
	t.Helper()

	for _, path := range []string{
		"auth.enabled",
		"auth.signing_key",
		"auth.refresh_token_pepper",
		"auth.mfa_secret_key",
		"auth.notification_driver",
		"database.driver",
		"database.sqlite.path",
		"migration.auto_apply",
		"migration.dir",
	} {
		for _, key := range config.EnvNamesForPath(path) {
			unsetAppIntegrationEnvForTest(t, key)
		}
	}
}

func unsetAppIntegrationEnvForTest(t *testing.T, key string) {
	t.Helper()

	oldValue, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
	t.Cleanup(func() {
		if existed {
			if err := os.Setenv(key, oldValue); err != nil {
				t.Errorf("restore %s: %v", key, err)
			}
			return
		}
		if err := os.Unsetenv(key); err != nil {
			t.Errorf("restore unset %s: %v", key, err)
		}
	})
}

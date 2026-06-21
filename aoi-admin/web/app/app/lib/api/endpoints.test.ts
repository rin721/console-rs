import { describe, expect, it } from "vitest";

import { API_ENDPOINTS } from "./endpoints";

describe("API endpoint registry", () => {
  it("matches the current Go health, readiness, setup, auth, and plugin route paths", () => {
    expect(API_ENDPOINTS.health).toBe("/health");
    expect(API_ENDPOINTS.ready).toBe("/ready");

    expect(API_ENDPOINTS.setup.status).toBe("/api/v1/setup/status");
    expect(API_ENDPOINTS.setup.schema).toBe("/api/v1/setup/schema");
    expect(API_ENDPOINTS.setup.config("database.configure")).toBe(
      "/api/v1/setup/configs/database.configure",
    );
    expect(API_ENDPOINTS.setup.configTest("database.configure")).toBe(
      "/api/v1/setup/configs/database.configure/test",
    );
    expect(API_ENDPOINTS.setup.runs).toBe("/api/v1/setup/runs");
    expect(API_ENDPOINTS.setup.runRetry("run-1")).toBe("/api/v1/setup/runs/run-1/retry");
    expect(API_ENDPOINTS.setup.runStepSkip("run-1", "cache.configure")).toBe(
      "/api/v1/setup/runs/run-1/steps/cache.configure/skip",
    );
    expect(API_ENDPOINTS.setup.runLogs("run-1")).toBe("/api/v1/setup/runs/run-1/logs");
    expect(API_ENDPOINTS.setup.complete).toBe("/api/v1/setup/complete");

    expect(API_ENDPOINTS.auth.setupStatus).toBe("/api/v1/auth/setup/status");
    expect(API_ENDPOINTS.auth.initialAdminSetup).toBe("/api/v1/auth/setup/initial-admin");
    expect(API_ENDPOINTS.auth.signup).toBe("/api/v1/auth/signup");
    expect(API_ENDPOINTS.auth.emailVerificationConfirm("verify token")).toBe(
      "/api/v1/auth/email-verifications/verify%20token/confirm",
    );
    expect(API_ENDPOINTS.auth.captcha).toBe("/api/v1/auth/captcha");
    expect(API_ENDPOINTS.auth.login).toBe("/api/v1/auth/login");
    expect(API_ENDPOINTS.auth.refresh).toBe("/api/v1/auth/refresh");
    expect(API_ENDPOINTS.auth.forgotPassword).toBe("/api/v1/auth/password/forgot");
    expect(API_ENDPOINTS.auth.passwordReset).toBe("/api/v1/auth/password/reset");
    expect(API_ENDPOINTS.auth.logout).toBe("/api/v1/auth/logout");
    expect(API_ENDPOINTS.auth.switchOrg).toBe("/api/v1/auth/switch-org");
    expect(API_ENDPOINTS.auth.mfaSetup).toBe("/api/v1/auth/mfa/setup");
    expect(API_ENDPOINTS.auth.mfaVerify).toBe("/api/v1/auth/mfa/verify");
    expect(API_ENDPOINTS.me.profile).toBe("/api/v1/me");
    expect(API_ENDPOINTS.me.organizations).toBe("/api/v1/me/orgs");
    expect(API_ENDPOINTS.me.session).toBe("/api/v1/me/session");
    expect(API_ENDPOINTS.invitations.accept("invite token")).toBe(
      "/api/v1/invitations/invite%20token/accept",
    );

    expect(API_ENDPOINTS.plugins.collection).toBe("/api/v1/plugins");
    expect(API_ENDPOINTS.plugins.item("plugin/one")).toBe("/api/v1/plugins/plugin%2Fone");
    expect(API_ENDPOINTS.plugins.health("plugin/one")).toBe("/api/v1/plugins/plugin%2Fone/health");
    expect(API_ENDPOINTS.plugins.capabilities("plugin/one")).toBe(
      "/api/v1/plugins/plugin%2Fone/capabilities",
    );
  });

  it("matches the current Go IAM organization route paths", () => {
    expect(API_ENDPOINTS.orgs.collection).toBe("/api/v1/orgs");
    expect(API_ENDPOINTS.orgs.item(42)).toBe("/api/v1/orgs/42");
    expect(API_ENDPOINTS.orgs.users(42)).toBe("/api/v1/orgs/42/users");
    expect(API_ENDPOINTS.orgs.user(42, "user/7")).toBe("/api/v1/orgs/42/users/user%2F7");
    expect(API_ENDPOINTS.orgs.userInvitations(42)).toBe("/api/v1/orgs/42/users/invitations");
    expect(API_ENDPOINTS.orgs.invitations(42)).toBe("/api/v1/orgs/42/invitations");
    expect(API_ENDPOINTS.orgs.invitation(42, "invite/7")).toBe(
      "/api/v1/orgs/42/invitations/invite%2F7",
    );
    expect(API_ENDPOINTS.orgs.apiTokens(42)).toBe("/api/v1/orgs/42/api-tokens");
    expect(API_ENDPOINTS.orgs.apiToken(42, "token/7")).toBe("/api/v1/orgs/42/api-tokens/token%2F7");
    expect(API_ENDPOINTS.orgs.roles(42)).toBe("/api/v1/orgs/42/roles");
    expect(API_ENDPOINTS.orgs.role(42, "role/7")).toBe("/api/v1/orgs/42/roles/role%2F7");
    expect(API_ENDPOINTS.orgs.permissions(42)).toBe("/api/v1/orgs/42/permissions");
    expect(API_ENDPOINTS.orgs.sessions(42)).toBe("/api/v1/orgs/42/sessions");
    expect(API_ENDPOINTS.orgs.session(42, "session/7")).toBe(
      "/api/v1/orgs/42/sessions/session%2F7",
    );
    expect(API_ENDPOINTS.orgs.auditLogs(42)).toBe("/api/v1/orgs/42/audit-logs");
  });

  it("matches the current Go System route paths", () => {
    expect(API_ENDPOINTS.system.publicSettings).toBe("/api/v1/system/public-settings");
    expect(API_ENDPOINTS.system.menus).toBe("/api/v1/system/menus");
    expect(API_ENDPOINTS.system.config).toBe("/api/v1/system/config");
    expect(API_ENDPOINTS.system.serverInfo).toBe("/api/v1/system/server-info");
    expect(API_ENDPOINTS.system.serverMetricsHistory).toBe("/api/v1/system/server-metrics/history");
    expect(API_ENDPOINTS.system.trafficHijack.overview).toBe(
      "/api/v1/system/traffic-hijack/overview",
    );
    expect(API_ENDPOINTS.system.trafficHijack.targets).toBe(
      "/api/v1/system/traffic-hijack/targets",
    );
    expect(API_ENDPOINTS.system.trafficHijack.target("target/7")).toBe(
      "/api/v1/system/traffic-hijack/targets/target%2F7",
    );
    expect(API_ENDPOINTS.system.trafficHijack.probe("target/7")).toBe(
      "/api/v1/system/traffic-hijack/targets/target%2F7/probe",
    );
    expect(API_ENDPOINTS.system.trafficHijack.results).toBe(
      "/api/v1/system/traffic-hijack/results",
    );
    expect(API_ENDPOINTS.system.trafficHijack.events).toBe("/api/v1/system/traffic-hijack/events");
    expect(API_ENDPOINTS.system.trafficHijack.eventResolve("event/7")).toBe(
      "/api/v1/system/traffic-hijack/events/event%2F7/resolve",
    );
    expect(API_ENDPOINTS.system.trafficHijack.stream).toBe("/api/v1/system/traffic-hijack/stream");
    expect(API_ENDPOINTS.system.apis).toBe("/api/v1/system/apis");
    expect(API_ENDPOINTS.system.apiSync).toBe("/api/v1/system/apis/sync");
    expect(API_ENDPOINTS.system.apiPermissionsSync).toBe("/api/v1/system/apis/permissions/sync");
    expect(API_ENDPOINTS.system.operationRecords).toBe("/api/v1/system/operation-records");

    expect(API_ENDPOINTS.system.versions).toBe("/api/v1/system/versions");
    expect(API_ENDPOINTS.system.versionExport).toBe("/api/v1/system/versions/export");
    expect(API_ENDPOINTS.system.versionImport).toBe("/api/v1/system/versions/import");
    expect(API_ENDPOINTS.system.versionSources).toBe("/api/v1/system/versions/sources");
    expect(API_ENDPOINTS.system.version("version/7")).toBe("/api/v1/system/versions/version%2F7");
    expect(API_ENDPOINTS.system.versionDownload("version/7")).toBe(
      "/api/v1/system/versions/version%2F7/download",
    );

    expect(API_ENDPOINTS.system.media.categories).toBe("/api/v1/system/media/categories");
    expect(API_ENDPOINTS.system.media.category("category/7")).toBe(
      "/api/v1/system/media/categories/category%2F7",
    );
    expect(API_ENDPOINTS.system.media.assets).toBe("/api/v1/system/media/assets");
    expect(API_ENDPOINTS.system.media.assetUpload).toBe("/api/v1/system/media/assets/upload");
    expect(API_ENDPOINTS.system.media.resumableCheck).toBe(
      "/api/v1/system/media/assets/resumable/check",
    );
    expect(API_ENDPOINTS.system.media.resumableChunks).toBe(
      "/api/v1/system/media/assets/resumable/chunks",
    );
    expect(API_ENDPOINTS.system.media.resumableComplete).toBe(
      "/api/v1/system/media/assets/resumable/complete",
    );
    expect(API_ENDPOINTS.system.media.resumableAbort).toBe(
      "/api/v1/system/media/assets/resumable/abort",
    );
    expect(API_ENDPOINTS.system.media.importURL).toBe("/api/v1/system/media/assets/import-url");
    expect(API_ENDPOINTS.system.media.asset("asset/7")).toBe(
      "/api/v1/system/media/assets/asset%2F7",
    );
    expect(API_ENDPOINTS.system.media.assetDownload("asset/7")).toBe(
      "/api/v1/system/media/assets/asset%2F7/download",
    );

    expect(API_ENDPOINTS.system.parameters).toBe("/api/v1/system/parameters");
    expect(API_ENDPOINTS.system.parameterValue).toBe("/api/v1/system/parameters/value");
    expect(API_ENDPOINTS.system.parameter("parameter/7")).toBe(
      "/api/v1/system/parameters/parameter%2F7",
    );
    expect(API_ENDPOINTS.system.dictionaries).toBe("/api/v1/system/dictionaries");
    expect(API_ENDPOINTS.system.dictionary("dictionary/7")).toBe(
      "/api/v1/system/dictionaries/dictionary%2F7",
    );
    expect(API_ENDPOINTS.system.dictionaryItems("dictionary/7")).toBe(
      "/api/v1/system/dictionaries/dictionary%2F7/items",
    );
    expect(API_ENDPOINTS.system.dictionaryItem("item/7")).toBe(
      "/api/v1/system/dictionary-items/item%2F7",
    );
  });
});

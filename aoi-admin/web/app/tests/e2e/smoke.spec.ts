import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { Buffer } from "node:buffer";

import { expect, test, type Locator, type Page, type Request, type Route } from "@playwright/test";

import type {
  IAMAPIToken,
  IAMInvitation,
  IAMOrganization,
  IAMOrganizationUser,
  IAMPermission,
  IAMRole,
  PluginCapabilitiesResponse,
  PluginHealthStatus,
  PluginSnapshot,
  SystemAPIGroup,
  SystemConfigSnapshot,
  SystemDictionary,
  SystemDictionaryItem,
  SystemMediaCategory,
  SystemOperationRecord,
  SystemParameter,
  SessionSnapshot,
  SystemTrafficHijackEvent,
  SystemTrafficHijackOverview,
  SystemTrafficProbeResult,
  SystemTrafficProbeTarget,
  SystemVersionPackage,
} from "~/lib/api/types";
import { setupLanguageConfirmedStorageKey } from "~/features/setup/setup-progress";

const testDir = fileURLToPath(new URL(".", import.meta.url));
const zhCN = JSON.parse(
  readFileSync(resolve(testDir, "../../app/i18n/locales/zh-CN.json"), "utf8"),
) as {
  auth: {
    login: {
      captchaTitle: string;
      mfaTitle: string;
      submit: string;
      title: string;
    };
    invitation: {
      accepted: string;
      submit: string;
      title: string;
    };
    passwordForgot: {
      debugTokenLabel: string;
      submit: string;
      title: string;
      tokenCreated: string;
    };
    passwordReset: {
      resetComplete: string;
      submit: string;
      title: string;
    };
  };
  admin: {
    apis: {
      actions: {
        reset: string;
        syncPermissions: string;
        syncRoutes: string;
      };
      filters: {
        keyword: string;
      };
      status: {
        unregistered: string;
        unsynced: string;
      };
      sync: {
        confirm: string;
        permissionsConfirmTitle: string;
        permissionsSuccessTitle: string;
        routesConfirmTitle: string;
        routesSuccessTitle: string;
      };
      title: string;
    };
    plugins: {
      actions: {
        reset: string;
        selected: string;
        view: string;
      };
      filters: {
        keyword: string;
      };
      status: {
        offline: string;
        online: string;
      };
      title: string;
    };
    iam: {
      status: {
        active: string;
      };
      title: string;
      values: {
        systemRole: string;
      };
    };
    roles: {
      actions: {
        create: string;
        save: string;
        submitCreate: string;
      };
      create: {
        successTitle: string;
        title: string;
      };
      edit: {
        role: string;
        successTitle: string;
        title: string;
      };
      fields: {
        code: string;
        description: string;
        name: string;
      };
      permissions: {
        createTitle: string;
        editTitle: string;
        toggle: string;
      };
      title: string;
    };
    organizations: {
      actions: {
        create: string;
        save: string;
        search: string;
        switch: string;
      };
      create: {
        successTitle: string;
        title: string;
      };
      current: {
        successTitle: string;
        title: string;
      };
      fields: {
        code: string;
        name: string;
        status: string;
      };
      filters: {
        keyword: string;
        pageSize: string;
        title: string;
      };
      switch: {
        successTitle: string;
      };
      title: string;
    };
    users: {
      actions: {
        confirmRevoke: string;
        disable: string;
        invite: string;
        roleSelect: string;
        saveRole: string;
        saveRoleFor: string;
        search: string;
        sendInvitation: string;
        toggleStatusFor: string;
      };
      filters: {
        displayName: string;
        email: string;
        keyword: string;
        pageSize: string;
        role: string;
        status: string;
        username: string;
      };
      invite: {
        debugToken: string;
        email: string;
        role: string;
        successTitle: string;
      };
      invitation: {
        confirmRevokeTitle: string;
        revokeInvitation: string;
        revokeSuccessTitle: string;
      };
      member: {
        roleSuccessTitle: string;
        statusSuccessTitle: string;
      };
      status: {
        active: string;
        disabled: string;
        pending: string;
        revoked: string;
      };
      title: string;
    };
    sessions: {
      actions: {
        confirmRevoke: string;
        currentSession: string;
        revokeSession: string;
        search: string;
      };
      filters: {
        ipAddress: string;
        keyword: string;
        scope: string;
        status: string;
        userId: string;
      };
      status: {
        active: string;
        expired: string;
        revoked: string;
      };
      revoke: {
        confirmTitle: string;
        successDescription: string;
      };
      title: string;
    };
    security: {
      actions: {
        generateSecret: string;
        logout: string;
        verifyAndEnable: string;
      };
      fields: {
        mfaCode: string;
      };
      messages: {
        enabledTitle: string;
        secretGeneratedTitle: string;
      };
      mfa: {
        disabledBadge: string;
        enabledBadge: string;
        otpauthLabel: string;
        secretLabel: string;
      };
      title: string;
    };
    apiTokens: {
      actions: {
        confirmRevoke: string;
        copyToken: string;
        create: string;
        issue: string;
        revokeToken: string;
        search: string;
      };
      filters: {
        status: string;
        userId: string;
      };
      issue: {
        remark: string;
        role: string;
        successDescription: string;
        title: string;
        user: string;
        validity: string;
      };
      issued: {
        fullToken: string;
        title: string;
        warningTitle: string;
      };
      revoke: {
        confirmTitle: string;
        successDescription: string;
      };
      status: {
        active: string;
        expired: string;
        revoked: string;
      };
      title: string;
    };
    auditLogs: {
      actions: {
        search: string;
      };
      filters: {
        action: string;
        cursor: string;
        from: string;
        limit: string;
        to: string;
        userId: string;
      };
      title: string;
    };
    loginLogs: {
      actions: {
        search: string;
      };
      filters: {
        from: string;
        ipAddress: string;
        limit: string;
        to: string;
        userId: string;
      };
      title: string;
    };
    dictionaries: {
      actions: {
        addItem: string;
        confirmDelete: string;
        create: string;
        delete: string;
        edit: string;
        reset: string;
        save: string;
        saveItem: string;
        submitCreate: string;
        submitCreateItem: string;
      };
      delete: {
        dictionaryTitle: string;
        itemTitle: string;
      };
      filters: {
        keyword: string;
        status: string;
        title: string;
      };
      form: {
        code: string;
        descriptionField: string;
        itemExtra: string;
        itemLabel: string;
        itemSort: string;
        itemStatus: string;
        itemValue: string;
        name: string;
        status: string;
      };
      messages: {
        createdTitle: string;
        deletedTitle: string;
        itemCreatedTitle: string;
        itemDeletedTitle: string;
        itemUpdatedTitle: string;
        updatedTitle: string;
      };
      title: string;
    };
    menus: {
      actions: {
        reset: string;
      };
      filters: {
        keyword: string;
      };
      title: string;
    };
    media: {
      actions: {
        confirmDeleteCategory: string;
        confirmDelete: string;
        createCategory: string;
        deleteAsset: string;
        deleteCategory: string;
        downloadAsset: string;
        editCategory: string;
        importUrls: string;
        renameAsset: string;
        saveCategory: string;
        saveRename: string;
        search: string;
      };
      a11y: {
        fileInput: string;
      };
      delete: {
        confirmTitle: string;
      };
      categories: {
        deleteTitle: string;
        nameField: string;
        sortField: string;
      };
      filters: {
        keyword: string;
      };
      messages: {
        categoryCreatedTitle: string;
        categoryDeletedTitle: string;
        categoryUpdatedTitle: string;
        deletedTitle: string;
        downloadedTitle: string;
        importedTitle: string;
        renamedTitle: string;
        uploadedTitle: string;
      };
      rename: {
        field: string;
        title: string;
      };
      title: string;
      write: {
        import: {
          label: string;
        };
      };
    };
    mediaResumable: {
      actions: {
        abort: string;
        reset: string;
        upload: string;
      };
      messages: {
        abortCompletedTitle: string;
        readyTitle: string;
        uploadCompletedTitle: string;
      };
      title: string;
      upload: {
        emptyFileTitle: string;
      };
    };
    parameters: {
      actions: {
        confirmDelete: string;
        create: string;
        deleteFor: string;
        deleteSelected: string;
        editFor: string;
        save: string;
        search: string;
      };
      form: {
        descriptionField: string;
        key: string;
        name: string;
        value: string;
      };
      filters: {
        key: string;
        name: string;
      };
      messages: {
        bulkDeletedTitle: string;
        createdTitle: string;
        deletedTitle: string;
        updatedTitle: string;
      };
      selection: {
        rowAria: string;
      };
      title: string;
    };
    operationRecords: {
      actions: {
        confirmDelete: string;
        deleteSelected: string;
        search: string;
      };
      delete: {
        bulkTitle: string;
      };
      filters: {
        method: string;
        path: string;
        status: string;
        statusClass: string;
      };
      messages: {
        deletedSelectedTitle: string;
      };
      selection: {
        rowAria: string;
      };
      title: string;
    };
    errorLogs: {
      actions: {
        search: string;
      };
      filters: {
        method: string;
        path: string;
        status: string;
        statusClass: string;
      };
      title: string;
    };
    probes: {
      status: {
        missing: string;
        notReady: string;
        ok: string;
        ready: string;
      };
      title: string;
    };
    dashboard: {
      apiCatalog: {
        title: string;
      };
      serverInfo: {
        title: string;
      };
      serverMetrics: {
        chartAria: string;
        diskChartAria: string;
        filters: {
          disk: string;
        };
        modes: {
          disk: string;
        };
        title: string;
      };
      title: string;
      trafficHijack: {
        chartAria: string;
        title: string;
      };
      versions: {
        title: string;
      };
    };
    notFound: {
      title: string;
    };
    trafficHijack: {
      form: {
        title: string;
      };
      results: {
        chartAria: string;
        title: string;
      };
      targets: {
        title: string;
      };
      title: string;
    };
    system: {
      actions: {
        editGroup: string;
        saveChanges: string;
      };
      editor: {
        newSecretValue: string;
        secretChanged: string;
      };
      messages: {
        updateSuccessTitle: string;
      };
      title: string;
      values: {
        secret: string;
      };
    };
    versions: {
      actions: {
        confirmDelete: string;
        createRelease: string;
        delete: string;
        deleteSelected: string;
        download: string;
        importVersion: string;
        search: string;
        selectAllSources: string;
        submitImport: string;
        view: string;
      };
      a11y: {
        selectVersion: string;
      };
      detail: {
        title: string;
      };
      export: {
        title: string;
        versionCode: string;
        versionName: string;
      };
      filters: {
        versionName: string;
      };
      import: {
        json: string;
      };
      messages: {
        deletedTitle: string;
        downloadedTitle: string;
        exportedTitle: string;
        importedTitle: string;
      };
      title: string;
    };
  };
  forms: {
    auth: {
      captchaCode: {
        label: string;
      };
      displayName: {
        label: string;
      };
      email: {
        label: string;
      };
      identifier: {
        label: string;
      };
      newPassword: {
        label: string;
      };
      password: {
        label: string;
      };
      mfaCode: {
        label: string;
      };
      resetToken: {
        label: string;
      };
      username: {
        label: string;
      };
    };
  };
  seo: {
    home: {
      description: string;
      title: string;
    };
  };
  setup: {
    actions: {
      continue: string;
      run: string;
      save: string;
      test: string;
    };
    confirm: {
      title: string;
    };
    errors: {
      passwordConfirmRequired: string;
      passwordMismatch: string;
      stepBlocked: string;
    };
    language: {
      field: string;
      title: string;
    };
    messages: {
      savedEnvManagedOverwritten: string;
    };
    owner: {
      passwordConfirm: {
        label: string;
      };
    };
    security: {
      title: string;
    };
    stepStatus: {
      blocked: string;
    };
    test: {
      failedTitle: string;
      repairHint: string;
      stale: string;
    };
  };
};

const en = JSON.parse(
  readFileSync(resolve(testDir, "../../app/i18n/locales/en.json"), "utf8"),
) as typeof zhCN;

type DesignSystemTestMessages = {
  actions: {
    applyImport: string;
    backendDisabled: string;
    saveDraft: string;
    sourcePackage: string;
  };
  fields: {
    mode: {
      label: string;
    };
    primaryColor: {
      label: string;
    };
  };
  import: {
    textarea: string;
  };
  messages: {
    importedTitle: string;
    savedTitle: string;
  };
  package: {
    title: string;
  };
  title: string;
};

const zhDesignSystem = (
  zhCN as typeof zhCN & {
    admin: {
      designSystem: DesignSystemTestMessages;
    };
  }
).admin.designSystem;

type SystemParameterInputBody = {
  description: string;
  key: string;
  name: string;
  value: string;
};

type SystemDictionaryInputBody = {
  code: string;
  description: string;
  name: string;
  status: string;
};

type SystemDictionaryItemInputBody = {
  extra: string;
  label: string;
  sort: number;
  status: string;
  value: string;
};

async function preferZhCN(page: Page) {
  await page.addInitScript(() => {
    window.localStorage.setItem("aoi-locale", "zh-CN");
  });
}

async function confirmSetupLanguagePreference(page: Page) {
  await page.addInitScript((storageKey) => {
    window.localStorage.setItem(storageKey, "true");
  }, setupLanguageConfirmedStorageKey);
}

function accessTokenWithOrg(orgId: string, extraClaims: Record<string, unknown> = {}) {
  const header = Buffer.from(JSON.stringify({ alg: "none", typ: "JWT" })).toString("base64url");
  const payload = Buffer.from(JSON.stringify({ orgId, ...extraClaims })).toString("base64url");
  return `${header}.${payload}.signature`;
}

async function setAuthenticatedSession(page: Page, accessToken: string) {
  await page.context().addCookies([
    {
      expires: Math.floor(new Date("2099-01-01T00:00:00Z").getTime() / 1000),
      httpOnly: true,
      name: "aoi_access",
      sameSite: "Lax",
      url: "http://127.0.0.1:3002",
      value: accessToken,
    },
    {
      expires: Math.floor(new Date("2099-01-02T00:00:00Z").getTime() / 1000),
      httpOnly: true,
      name: "aoi_refresh",
      sameSite: "Lax",
      url: "http://127.0.0.1:3002",
      value: "refresh-token",
    },
    {
      expires: Math.floor(new Date("2099-01-01T00:00:00Z").getTime() / 1000),
      httpOnly: false,
      name: "aoi_csrf",
      sameSite: "Lax",
      url: "http://127.0.0.1:3002",
      value: "csrf-token",
    },
  ]);
  await page.route(/\/api\/v1\/me\/session$/, async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: sessionSnapshot(accessToken),
      }),
    });
  });
  await page.route(/\/api\/v1\/me\/orgs$/, async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: [{ code: "root", id: "1", name: "Root Org" }],
      }),
    });
  });
  await page.route(/\/api\/v1\/me$/, async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          displayName: "Owner User",
          email: "owner@example.com",
          id: "1",
          mfaEnabled: false,
          username: "owner",
        },
      }),
    });
  });
  await page.addInitScript(() => {
    window.localStorage.setItem("aoi-locale", "zh-CN");
  });
}

function sessionSnapshot(
  accessToken: string,
  overrides: Partial<SessionSnapshot> = {},
): SessionSnapshot {
  const claims = accessTokenClaims(accessToken);
  const orgId = String(overrides.orgId ?? claims.orgId ?? "1");
  return {
    accessExpiresAt: "2099-01-01T00:00:00Z",
    clientType: "pc_web",
    orgId,
    productCode: "aoi-admin",
    refreshExpiresAt: "2099-01-02T00:00:00Z",
    sessionId: String(overrides.sessionId ?? claims.sessionId ?? `sess-${orgId}`),
    userId: String(overrides.userId ?? claims.userId ?? "1"),
    ...overrides,
  };
}

function accessTokenClaims(accessToken: string): Record<string, unknown> {
  const payload = accessToken.split(".")[1];
  if (!payload) {
    return {};
  }
  try {
    return JSON.parse(Buffer.from(payload, "base64url").toString("utf8")) as Record<
      string,
      unknown
    >;
  } catch {
    return {};
  }
}

function requestAuth(request: Request) {
  return authFromHeaders(request.headers());
}

function authFromHeaders(headers: Record<string, string>) {
  if (headers.authorization) {
    return headers.authorization;
  }
  const accessToken = cookieValue(headers.cookie ?? "", "aoi_access");
  return accessToken ? `Bearer ${accessToken}` : null;
}

function cookieValue(cookieHeader: string, name: string) {
  return (
    cookieHeader
      .split(";")
      .map((part) => part.trim())
      .find((part) => part.startsWith(`${name}=`))
      ?.slice(name.length + 1) ?? ""
  );
}

function responseCookies(accessToken: string, refreshToken = "refresh-token") {
  return {
    "Set-Cookie": [
      `aoi_access=${accessToken}; Path=/; HttpOnly; SameSite=Lax; Expires=Thu, 01 Jan 2099 00:00:00 GMT`,
      `aoi_refresh=${refreshToken}; Path=/; HttpOnly; SameSite=Lax; Expires=Fri, 02 Jan 2099 00:00:00 GMT`,
      `aoi_csrf=csrf-token; Path=/; SameSite=Lax; Expires=Thu, 01 Jan 2099 00:00:00 GMT`,
    ].join("\n"),
  };
}

async function routePublicSettings(page: Page, registrationMode = "direct") {
  await page.route("**/api/v1/system/public-settings", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          auth: {
            clientTypeHeader: "X-Aoi-Client-Type",
            csrfCookieName: "aoi_csrf",
            csrfEnabled: true,
            csrfHeaderName: "X-CSRF-Token",
            defaultClientType: "pc_web",
            defaultProductCode: "aoi-admin",
            productHeader: "X-Aoi-Product-Code",
            registrationMode,
          },
          brand: {
            productCode: "aoi-admin",
            productName: "Aoi Admin",
            versionName: "Community",
          },
          defaultLocale: "zh-CN",
          fallbackLocale: "zh-CN",
          supportedLocales: ["zh-CN", "en-US"],
        },
      }),
    });
  });
}

function interpolate(template: string, values: Record<string, number | string>) {
  return Object.entries(values).reduce(
    (result, [key, value]) => result.replaceAll(`{{${key}}}`, String(value)),
    template,
  );
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

async function activateButton(button: Locator, page: Page) {
  await button.focus();
  await page.keyboard.press("Enter");
}

async function scrollTableActionsIntoView(row: Locator) {
  await row.locator(".aoi-user-actions").evaluate((element) => {
    element.scrollIntoView({ block: "center", inline: "end" });
    const scroller = element.closest(".aoi-data-table-wrap");
    if (scroller instanceof HTMLElement) {
      scroller.scrollLeft = scroller.scrollWidth;
    }
  });
}

function organizationMatchesQuery(organization: IAMOrganization, params: URLSearchParams) {
  const keyword = params.get("keyword")?.trim().toLowerCase();
  const code = params.get("code")?.trim().toLowerCase();
  const name = params.get("name")?.trim().toLowerCase();
  const status = params.get("status")?.trim();

  if (code && !organization.code.toLowerCase().includes(code)) {
    return false;
  }
  if (name && !organization.name.toLowerCase().includes(name)) {
    return false;
  }
  if (status && organization.status !== status) {
    return false;
  }
  if (!keyword) {
    return true;
  }

  return [organization.code, organization.name, organization.status].some((value) =>
    value.toLowerCase().includes(keyword),
  );
}

test("public home renders", async ({ page }) => {
  const publicSettingsRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
  }> = [];

  await page.route("**/api/v1/system/public-settings", async (route) => {
    const request = route.request();
    const requestUrl = new URL(request.url());
    publicSettingsRequests.push({
      authorization: requestAuth(request),
      locale: request.headers()["x-locale"] ?? null,
      path: requestUrl.pathname,
    });

    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          auth: {
            clientTypeHeader: "X-Aoi-Client-Type",
            csrfCookieName: "aoi_csrf",
            csrfEnabled: true,
            csrfHeaderName: "X-CSRF-Token",
            defaultClientType: "pc_web",
            defaultProductCode: "aoi-suite",
            productHeader: "X-Aoi-Product-Code",
            registrationMode: "direct",
          },
          brand: {
            productCode: "aoi-suite",
            productName: "Aoi Suite",
            versionName: "Community",
          },
          defaultLocale: "zh-CN",
          fallbackLocale: "zh-CN",
          supportedLocales: ["zh-CN", "en-US"],
        },
      }),
    });
  });

  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
  await expect(page.getByRole("heading", { level: 1 })).toHaveCount(1);
  await expect(page.locator(".aoi-nav")).toBeVisible();
  await expect(page.locator(".aoi-brand")).toContainText("Aoi Suite");
  await expect(page).toHaveTitle(new RegExp(escapeRegExp(en.seo.home.title)));
  await expect(page.locator('meta[name="description"]')).toHaveAttribute(
    "content",
    en.seo.home.description,
  );
  await expect(page.locator('meta[property="og:title"]')).toHaveAttribute("content", /Aoi/);
  await expect(page.locator('link[rel="canonical"]')).toHaveAttribute(
    "href",
    /http:\/\/127\.0\.0\.1:3002\/$/,
  );
  const homeJsonLd = await page
    .locator('script[data-aoi-jsonld="home"]')
    .evaluate((script) => script.textContent ?? "");
  expect(JSON.parse(homeJsonLd)).toMatchObject({
    "@type": "SoftwareApplication",
    name: "Aoi Suite",
  });
  expect(publicSettingsRequests.length).toBeGreaterThan(0);
  for (const request of publicSettingsRequests) {
    expect(request).toEqual({
      authorization: null,
      locale: "en-US",
      path: "/api/v1/system/public-settings",
    });
  }
});

test("public support routes keep page structure and metadata", async ({ page }) => {
  await routePublicSettings(page);
  for (const path of [
    "/about",
    "/terms",
    "/privacy",
    "/login",
    "/signup",
    "/password/forgot",
    "/password/reset",
  ]) {
    await page.goto(path);
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
    await expect(page.getByRole("heading", { level: 1 })).toHaveCount(1);
    await expect(page.locator('meta[name="description"]')).toHaveAttribute("content", /.+/);
    await expect(page.locator('meta[property="og:description"]')).toHaveAttribute("content", /.+/);
    await expect(page.locator('link[rel="canonical"]')).toHaveAttribute(
      "href",
      new RegExp(`http://127\\.0\\.0\\.1:3002${path}$`),
    );
  }
});

test("login route submits backend captcha challenge fields", async ({ page }) => {
  await preferZhCN(page);
  const accessToken = accessTokenWithOrg("1");
  const loginBodies: unknown[] = [];

  await page.route("**/api/v1/auth/captcha", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          captchaId: "captcha-1",
          enabled: true,
          expiresAt: "2026-06-18T09:05:00Z",
          image: `data:image/svg+xml;base64,${Buffer.from(
            '<svg xmlns="http://www.w3.org/2000/svg" width="120" height="44"><text x="12" y="28">A1B2</text></svg>',
          ).toString("base64")}`,
        },
      }),
    });
  });
  await page.route("**/api/v1/auth/login", async (route) => {
    const body = route.request().postDataJSON() as unknown;
    loginBodies.push(body);
    if (loginBodies.length === 1) {
      await route.fulfill({
        status: 400,
        contentType: "application/json",
        body: JSON.stringify({
          code: "CAPTCHA_REQUIRED",
          message: "Captcha required",
          messageKey: "api.auth.captchaRequired",
        }),
      });
      return;
    }
    await route.fulfill({
      contentType: "application/json",
      headers: responseCookies(accessToken),
      body: JSON.stringify({
        code: 0,
        data: sessionSnapshot(accessToken),
      }),
    });
  });
  await page.route("**/api/v1/me/orgs", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: [{ code: "default", id: "1", name: "Default Org" }],
      }),
    });
  });
  await page.route("**/api/v1/me", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          displayName: "Owner User",
          email: "owner@example.com",
          id: "1",
          username: "owner",
        },
      }),
    });
  });

  await page.goto("/login");
  await expect(page.getByRole("heading", { level: 1, name: zhCN.auth.login.title })).toBeVisible();
  await page.getByLabel(zhCN.forms.auth.identifier.label).fill("owner@example.com");
  await page.getByLabel(zhCN.forms.auth.password.label).fill("password123");
  await page.getByRole("button", { name: zhCN.auth.login.submit }).click();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.auth.login.captchaTitle }),
  ).toBeVisible();
  await page.getByLabel(zhCN.forms.auth.captchaCode.label).fill("A1B2");
  await page.getByRole("button", { name: zhCN.auth.login.submit }).click();
  await expect(page).toHaveURL(/\/admin$/);

  expect(loginBodies).toEqual([
    { identifier: "owner@example.com", password: "password123" },
    {
      captchaCode: "A1B2",
      captchaId: "captcha-1",
      identifier: "owner@example.com",
      password: "password123",
    },
  ]);
});

test("login route submits backend MFA challenge fields", async ({ page }) => {
  await preferZhCN(page);
  const accessToken = accessTokenWithOrg("1");
  const loginBodies: unknown[] = [];

  await page.route("**/api/v1/auth/login", async (route) => {
    const body = route.request().postDataJSON() as unknown;
    loginBodies.push(body);
    if (loginBodies.length === 1) {
      await route.fulfill({
        status: 401,
        contentType: "application/json",
        body: JSON.stringify({
          code: "MFA_REQUIRED",
          message: "MFA required",
          messageKey: "api.auth.mfaRequired",
        }),
      });
      return;
    }
    await route.fulfill({
      contentType: "application/json",
      headers: responseCookies(accessToken),
      body: JSON.stringify({
        code: 0,
        data: sessionSnapshot(accessToken),
      }),
    });
  });
  await page.route("**/api/v1/me/orgs", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: [{ code: "default", id: "1", name: "Default Org" }],
      }),
    });
  });
  await page.route("**/api/v1/me", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          displayName: "Owner User",
          email: "owner@example.com",
          id: "1",
          username: "owner",
        },
      }),
    });
  });

  await page.goto("/login");
  await expect(page.getByRole("heading", { level: 1, name: zhCN.auth.login.title })).toBeVisible();
  await page.getByLabel(zhCN.forms.auth.identifier.label).fill("owner@example.com");
  await page.getByLabel(zhCN.forms.auth.password.label).fill("password123");
  await page.getByRole("button", { name: zhCN.auth.login.submit }).click();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.auth.login.mfaTitle }),
  ).toBeVisible();
  await page.getByLabel(zhCN.forms.auth.mfaCode.label).fill("123456");
  await page.getByRole("button", { name: zhCN.auth.login.submit }).click();
  await expect(page).toHaveURL(/\/admin$/);

  expect(loginBodies).toEqual([
    { identifier: "owner@example.com", password: "password123" },
    {
      identifier: "owner@example.com",
      mfaCode: "123456",
      password: "password123",
    },
  ]);
});

test("password recovery routes submit backend-supported payloads", async ({ page }) => {
  await preferZhCN(page);
  let forgotBody: unknown;
  await page.route("**/api/v1/auth/password/forgot", async (route) => {
    forgotBody = route.request().postDataJSON();
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          debug: true,
          token: "reset-token",
          url: "/admin/password/reset?token=reset-token",
        },
      }),
    });
  });

  await page.goto("/password/forgot");
  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.auth.passwordForgot.title }),
  ).toBeVisible();
  await page.getByLabel(zhCN.forms.auth.email.label).fill("owner@example.com");
  await page.getByRole("button", { name: zhCN.auth.passwordForgot.submit }).click();
  await expect(page.getByText(zhCN.auth.passwordForgot.tokenCreated)).toBeVisible();
  await expect(
    page
      .locator(".aoi-auth-debug")
      .getByText(zhCN.auth.passwordForgot.debugTokenLabel, { exact: true }),
  ).toBeVisible();
  expect(forgotBody).toEqual({ email: "owner@example.com" });

  let resetBody: unknown;
  await page.route("**/api/v1/auth/password/reset", async (route) => {
    resetBody = route.request().postDataJSON();
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: { reset: true } }),
    });
  });

  await page.goto("/password/reset?token=reset-token");
  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.auth.passwordReset.title }),
  ).toBeVisible();
  await expect(page.getByLabel(zhCN.forms.auth.resetToken.label)).toHaveValue("reset-token");
  await page.getByLabel(zhCN.forms.auth.newPassword.label).fill("newpassword123");
  await page.getByRole("button", { name: zhCN.auth.passwordReset.submit }).click();
  await expect(page.getByText(zhCN.auth.passwordReset.resetComplete)).toBeVisible();
  expect(resetBody).toEqual({ newPassword: "newpassword123", token: "reset-token" });
});

test("invitation acceptance submits backend-supported payload", async ({ page }) => {
  await preferZhCN(page);
  let invitationBody: unknown;
  await page.route("**/api/v1/invitations/invite-token/accept", async (route) => {
    invitationBody = route.request().postDataJSON();
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          email: "member@example.com",
          orgId: "1",
          userId: "2",
        },
      }),
    });
  });

  await page.goto("/invitations/invite-token");
  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.auth.invitation.title }),
  ).toBeVisible();
  await page.getByLabel(zhCN.forms.auth.username.label).fill("member");
  await page.getByLabel(zhCN.forms.auth.displayName.label).fill("Member User");
  await page.getByLabel(zhCN.forms.auth.password.label).fill("password123");
  await page.getByRole("button", { name: zhCN.auth.invitation.submit }).click();
  await expect(page.getByText(zhCN.auth.invitation.accepted)).toBeVisible();
  expect(invitationBody).toEqual({
    displayName: "Member User",
    password: "password123",
    username: "member",
  });
});

test("blog detail uses article metadata from front matter", async ({ page }) => {
  await page.goto("/blog/react-frontend-migration");
  await expect(page.getByRole("heading", { level: 1 })).toContainText(/React/);
  await expect(page.locator(".aoi-article-cover")).toBeVisible();
  await expect(page.locator('meta[property="og:type"]')).toHaveAttribute("content", "article");
  await expect(page.locator('meta[property="article:published_time"]')).toHaveAttribute(
    "content",
    "2026-06-18",
  );
  const articleJsonLd = await page
    .locator('script[data-aoi-jsonld="blog-article"]')
    .evaluate((script) => script.textContent ?? "");
  expect(articleJsonLd).toContain("Article");
});

test("admin route requires an authenticated session", async ({ page }) => {
  await page.addInitScript(() => {
    window.sessionStorage.removeItem("aoi-admin-session");
  });
  await page.goto("/admin");
  await expect(page).toHaveURL(/\/login\?next=%2Fadmin/);
  await expect(page.locator(".aoi-admin-sidebar")).toHaveCount(0);
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
});

test("authenticated admin dashboard reads backend-supported overview APIs", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
  }> = [];
  const trafficTarget: SystemTrafficProbeTarget = {
    alertChannels: "event,debug",
    allowPrivateNetwork: false,
    createdAt: "2026-06-18T09:58:00Z",
    emailRecipients: "",
    enabled: true,
    expectedContentKeyword: "Aoi",
    expectedFinalHost: "example.com",
    expectedIpCidrs: "93.184.216.34/32",
    expectedStatusCodes: "200-399",
    expectedTlsFingerprint: "",
    id: "target-1",
    intervalSeconds: 30,
    lastCheckedAt: "2026-06-18T10:00:00Z",
    lastError: "",
    lastReason: "probe healthy",
    lastSeverity: "ok",
    lastStatus: "healthy",
    method: "GET",
    name: "Public edge",
    nextProbeAt: "2026-06-18T10:00:30Z",
    timeoutSeconds: 5,
    updatedAt: "2026-06-18T10:00:00Z",
    url: "https://example.com",
  };
  const trafficResult: SystemTrafficProbeResult = {
    connectDurationMs: 8,
    createdAt: "2026-06-18T10:00:00Z",
    dnsDurationMs: 5,
    dnsIps: "93.184.216.34",
    errorMessage: "",
    evidenceJson: "{}",
    finalUrl: "https://example.com/",
    id: "result-1",
    method: "GET",
    reason: "probe healthy",
    severity: "ok",
    stage: "complete",
    status: "healthy",
    statusCode: 200,
    targetId: "target-1",
    targetName: "Public edge",
    tlsDurationMs: 9,
    tlsFingerprintSha256: "ABCD",
    tlsIssuer: "CN=Example CA",
    tlsNotAfter: "2027-01-01T00:00:00Z",
    tlsSubject: "CN=example.com",
    totalDurationMs: 48,
    ttfbMs: 20,
    url: "https://example.com",
  };
  const trafficOverview: SystemTrafficHijackOverview = {
    criticalTargets: 0,
    enabledTargets: 1,
    healthyTargets: 1,
    recentEvents: [],
    recentResults: [trafficResult],
    severityCounts: { ok: 1 },
    storageStatus: "available",
    targets: [trafficTarget],
    totalTargets: 1,
    warningTargets: 0,
  };

  await page.route("**/health", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: { status: "ok" } }),
    });
  });
  await page.route("**/ready", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: { checks: { database: "ok" }, status: "ready" } }),
    });
  });
  await page.route("**/api/v1/system/server-info", async (route) => {
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          build: {
            goVersion: "go1.25.7",
            module: "github.com/rei0721/go-scaffold",
            path: "github.com/rei0721/go-scaffold/cmd/aoi",
            settings: [],
            version: "dev",
          },
          cpu: { cores: 8, percent: [12.5, 8.2] },
          disk: [
            {
              device: "C:",
              fsType: "NTFS",
              mountPoint: "C:\\",
              totalMb: 512000,
              usedMb: 256000,
              usedPercent: 50,
            },
          ],
          gc: { nextGcMb: 64, numGc: 3, pauseTotalNs: 1000 },
          memory: {
            allocMb: 32,
            heapAllocMb: 28,
            heapIdleMb: 12,
            heapInuseMb: 32,
            heapObjects: 1200,
            heapReleasedMb: 4,
            heapSysMb: 44,
            stackInuseMb: 2,
            stackSysMb: 3,
            sysMb: 90,
            totalAllocMb: 128,
          },
          os: {
            compiler: "gc",
            goarch: "amd64",
            goos: "windows",
            goVersion: "go1.25.7",
            numCpu: 8,
            numGoroutine: 27,
          },
          ram: { totalMb: 32768, usedMb: 8192, usedPercent: 25 },
          refreshedAt: "2026-06-18T10:00:00Z",
          runtime: {
            startTime: "2026-06-18T08:00:00Z",
            uptime: "2h0m0s",
            uptimeSeconds: 7200,
          },
        },
      }),
    });
  });
  let serverMetricsCalls = 0;
  await page.route("**/api/v1/system/server-metrics/history", async (route) => {
    serverMetricsCalls += 1;
    const latestDiskRead = serverMetricsCalls > 1 ? 1.6 : 1.2;
    const latestDiskWrite = serverMetricsCalls > 1 ? 0.7 : 0.4;
    const latestReceive = serverMetricsCalls > 1 ? 9.6 : 7.2;
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          intervalSeconds: 5,
          samples: [
            {
              cpuUsedPercent: 12.4,
              diskIo: [
                {
                  ioLatencyMs: 3.2,
                  name: "PhysicalDrive0",
                  readMbPerSecond: 0.8,
                  readOpsPerSecond: 1.6,
                  writeMbPerSecond: 0.2,
                  writeOpsPerSecond: 0.6,
                },
              ],
              diskIoLatencyMs: 3.2,
              diskMaxUsedPercent: 50,
              diskReadMbPerSecond: 0.8,
              diskReadOpsPerSecond: 1.6,
              diskWriteMbPerSecond: 0.2,
              diskWriteOpsPerSecond: 0.6,
              goroutines: 27,
              heapAllocMb: 28,
              networkReceiveKbPerSecond: 4.8,
              networkTransmitKbPerSecond: 2.4,
              ramUsedPercent: 25,
              sampledAt: "2026-06-18T09:59:55Z",
            },
            {
              cpuUsedPercent: 10.2,
              diskIo: [
                {
                  ioLatencyMs: 4.3,
                  name: "PhysicalDrive0",
                  readMbPerSecond: latestDiskRead,
                  readOpsPerSecond: 2.4,
                  writeMbPerSecond: latestDiskWrite,
                  writeOpsPerSecond: 1.1,
                },
              ],
              diskIoLatencyMs: 4.3,
              diskMaxUsedPercent: 50,
              diskReadMbPerSecond: latestDiskRead,
              diskReadOpsPerSecond: 2.4,
              diskWriteMbPerSecond: latestDiskWrite,
              diskWriteOpsPerSecond: 1.1,
              goroutines: 28,
              heapAllocMb: 29,
              networkReceiveKbPerSecond: latestReceive,
              networkTransmitKbPerSecond: 3.1,
              ramUsedPercent: 25.5,
              sampledAt: "2026-06-18T10:00:00Z",
            },
          ],
          windowSeconds: 300,
        },
      }),
    });
  });
  await page.route("**/api/v1/system/traffic-hijack/overview", async (route) => {
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: trafficOverview,
      }),
    });
  });
  await page.route("**/api/v1/system/apis", async (route) => {
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: [
          {
            code: "system",
            count: 2,
            label: "system",
            items: [
              {
                access: "permission",
                code: "get /api/v1/system/server-info",
                description: "server info",
                group: "system",
                method: "GET",
                order: 10,
                path: "/api/v1/system/server-info",
                permission: "server:read",
                permissionRegistered: true,
                synced: true,
              },
              {
                access: "permission",
                code: "get /api/v1/system/server-metrics/history",
                description: "server metrics history",
                group: "system",
                method: "GET",
                order: 11,
                path: "/api/v1/system/server-metrics/history",
                permission: "server:read",
                permissionRegistered: true,
                synced: true,
              },
              {
                access: "public",
                code: "get /api/v1/system/public-settings",
                description: "public settings",
                group: "system",
                method: "GET",
                order: 10,
                path: "/api/v1/system/public-settings",
                permissionRegistered: true,
                synced: true,
              },
            ],
          },
        ],
      }),
    });
  });
  await page.route("**/api/v1/system/versions**", async (route) => {
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: [
            {
              apiCount: 2,
              createdAt: "2026-06-18T09:00:00Z",
              createdBy: "1",
              createdByUsername: "owner",
              description: "Initial export",
              dictionaryCount: 1,
              id: "100",
              menuCount: 1,
              source: "export",
              updatedAt: "2026-06-18T09:00:00Z",
              versionCode: "v2026.06",
              versionName: "June Release",
            },
          ],
          page: 1,
          pageSize: 5,
          storageStatus: "available",
          total: 1,
        },
      }),
    });
  });

  await page.goto("/admin");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.dashboard.title }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.dashboard.serverInfo.title }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.dashboard.serverMetrics.title }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.dashboard.trafficHijack.title }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.dashboard.apiCatalog.title }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.dashboard.versions.title }),
  ).toBeVisible();
  await expect(page.getByText("go1.25.7").first()).toBeVisible();
  await expect(
    page.getByRole("img", { name: zhCN.admin.dashboard.serverMetrics.chartAria }),
  ).toBeVisible();
  await page.getByRole("tab", { name: zhCN.admin.dashboard.serverMetrics.modes.disk }).click();
  await expect(
    page.getByRole("img", { name: zhCN.admin.dashboard.serverMetrics.diskChartAria }),
  ).toBeVisible();
  await expect(
    page.getByLabel(zhCN.admin.dashboard.serverMetrics.filters.disk, { exact: true }),
  ).toBeVisible();
  await expect.poll(() => serverMetricsCalls, { timeout: 7_000 }).toBeGreaterThan(1);
  await expect(page.getByText(/1\.6 MB\/s/)).toBeVisible();
  await expect(
    page.getByRole("img", { name: zhCN.admin.dashboard.trafficHijack.chartAria }),
  ).toBeVisible();
  await expect(page.getByText("system").first()).toBeVisible();
  await expect(page.getByRole("cell", { name: "June Release" })).toBeVisible();
  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set([
      "/api/v1/system/apis",
      "/api/v1/system/server-info",
      "/api/v1/system/server-metrics/history",
      "/api/v1/system/traffic-hijack/overview",
      "/api/v1/system/versions",
    ]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      path: request.path,
    })),
  );
});

test("admin IAM route renders backend IAM overview read-only", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
    query: Record<string, string>;
  }> = [];

  function recordRequest(routeUrl: string, headers: Record<string, string>) {
    const url = new URL(routeUrl);
    protectedRequests.push({
      authorization: authFromHeaders(headers),
      locale: headers["x-locale"] ?? null,
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
  }

  await page.route(/\/api\/v1\/orgs\/1\/users(?:\?.*)?$/, async (route) => {
    recordRequest(route.request().url(), route.request().headers());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: [
            {
              membershipStatus: "active",
              roles: ["owner"],
              user: {
                createdAt: "2026-06-18T08:00:00Z",
                displayName: "Owner User",
                email: "owner@example.com",
                id: "1",
                lastLoginAt: "2026-06-18T09:00:00Z",
                mfaEnabled: true,
                status: "active",
                updatedAt: "2026-06-18T09:00:00Z",
                username: "owner",
              },
            },
          ],
          page: 1,
          pageSize: 5,
          storageStatus: "persisted",
          total: 1,
        },
      }),
    });
  });
  await page.route(/\/api\/v1\/orgs\/1\/roles(?:\?.*)?$/, async (route) => {
    recordRequest(route.request().url(), route.request().headers());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: [
          {
            code: "owner",
            createdAt: "2026-06-18T08:00:00Z",
            description: "Full organization access",
            id: "10",
            name: "Owner Role",
            orgId: "1",
            permissions: ["org:read", "user:read", "role:read"],
            system: true,
            updatedAt: "2026-06-18T09:00:00Z",
          },
        ],
      }),
    });
  });
  await page.route(/\/api\/v1\/orgs\/1\/permissions(?:\?.*)?$/, async (route) => {
    recordRequest(route.request().url(), route.request().headers());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: [
          {
            code: "user:read",
            createdAt: "2026-06-18T08:00:00Z",
            description: "Read users",
            id: "100",
            name: "Read users",
            updatedAt: "2026-06-18T09:00:00Z",
          },
          {
            code: "role:read",
            createdAt: "2026-06-18T08:00:00Z",
            description: "Read roles",
            id: "101",
            name: "Read roles",
            updatedAt: "2026-06-18T09:00:00Z",
          },
        ],
      }),
    });
  });
  await page.route(/\/api\/v1\/orgs\/1\/sessions(?:\?.*)?$/, async (route) => {
    recordRequest(route.request().url(), route.request().headers());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: [
            {
              createdAt: "2026-06-18T08:00:00Z",
              expiresAt: "2099-01-01T00:00:00Z",
              id: "200",
              ipAddress: "127.0.0.1",
              lastUsedAt: "2026-06-18T09:00:00Z",
              orgId: "1",
              updatedAt: "2026-06-18T09:00:00Z",
              userAgent: "Playwright",
              userId: "1",
            },
          ],
          page: 1,
          pageSize: 5,
          storageStatus: "persisted",
          total: 1,
        },
      }),
    });
  });
  await page.route(/\/api\/v1\/orgs\/1\/audit-logs(?:\?.*)?$/, async (route) => {
    recordRequest(route.request().url(), route.request().headers());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: [
          {
            action: "user.update",
            createdAt: "2026-06-18T09:30:00Z",
            id: "300",
            ipAddress: "127.0.0.1",
            metadata: "{}",
            orgId: "1",
            resource: "user",
            resourceId: "1",
            userAgent: "Playwright",
            userId: "1",
          },
        ],
      }),
    });
  });
  await page.route(/\/api\/v1\/orgs(?:\?.*)?$/, async (route) => {
    recordRequest(route.request().url(), route.request().headers());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: [
            {
              code: "main",
              createdAt: "2026-06-18T08:00:00Z",
              id: "1",
              name: "Main Organization",
              status: "active",
              updatedAt: "2026-06-18T09:00:00Z",
            },
          ],
          page: 1,
          pageSize: 5,
          storageStatus: "persisted",
          total: 1,
        },
      }),
    });
  });

  await page.goto("/admin/iam");

  await expect(page.getByRole("heading", { level: 1, name: zhCN.admin.iam.title })).toBeVisible();
  await expect(page.getByText("Main Organization", { exact: true })).toBeVisible();
  await expect(page.getByRole("row", { name: /Owner User owner@example\.com/ })).toBeVisible();
  await expect(page.getByText("Owner Role", { exact: true })).toBeVisible();
  await expect(
    page
      .locator(".aoi-iam-table--roles")
      .getByText(zhCN.admin.iam.values.systemRole, { exact: true }),
  ).toBeVisible();
  await expect(page.getByText("127.0.0.1", { exact: true }).first()).toBeVisible();
  await expect(page.getByText("user.update", { exact: true })).toBeVisible();
  await expect(page.getByText(zhCN.admin.iam.status.active, { exact: true }).first()).toBeVisible();

  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set([
      "/api/v1/orgs",
      "/api/v1/orgs/1/audit-logs",
      "/api/v1/orgs/1/permissions",
      "/api/v1/orgs/1/roles",
      "/api/v1/orgs/1/sessions",
      "/api/v1/orgs/1/users",
    ]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      path: request.path,
      query: request.query,
    })),
  );
  expect(protectedRequests.find((request) => request.path.endsWith("/users"))?.query).toMatchObject(
    { page: "1", pageSize: "5" },
  );
  expect(
    protectedRequests.find((request) => request.path.endsWith("/sessions"))?.query,
  ).toMatchObject({ page: "1", pageSize: "5", scope: "org" });
  expect(
    protectedRequests.find((request) => request.path.endsWith("/audit-logs"))?.query,
  ).toMatchObject({ limit: "6" });
});

test("admin organizations route manages backend-supported organizations", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  const switchedAccessToken = accessTokenWithOrg("2");
  await setAuthenticatedSession(page, accessToken);
  const listRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
    query: Record<string, string>;
  }> = [];
  const identityRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
  }> = [];
  const mutationRequests: Array<{
    authorization: string | null;
    body?: unknown;
    locale: string | null;
    method: string;
    path: string;
  }> = [];
  let organizationRecords: IAMOrganization[] = [
    {
      code: "root",
      createdAt: "2026-06-18T07:00:00Z",
      id: "1",
      name: "Root Org",
      status: "active",
      updatedAt: "2026-06-18T07:00:00Z",
    },
    {
      code: "beta",
      createdAt: "2026-06-18T08:00:00Z",
      id: "2",
      name: "Beta Org",
      status: "active",
      updatedAt: "2026-06-18T08:00:00Z",
    },
    {
      code: "archive",
      createdAt: "2026-06-18T09:00:00Z",
      id: "3",
      name: "Archive Org",
      status: "disabled",
      updatedAt: "2026-06-18T09:00:00Z",
    },
  ];

  await page.route("**/api/v1/me/orgs", async (route) => {
    identityRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: organizationRecords.map((organization) => ({
          code: organization.code,
          id: organization.id,
          name: organization.name,
        })),
      }),
    });
  });

  await page.route("**/api/v1/me", async (route) => {
    identityRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          displayName: "Owner User",
          email: "owner@example.com",
          id: "1",
          username: "owner",
        },
      }),
    });
  });

  await page.route("**/api/v1/auth/switch-org", async (route) => {
    const body = route.request().postDataJSON() as { orgId: number };
    mutationRequests.push({
      authorization: requestAuth(route.request()),
      body,
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      headers: responseCookies(switchedAccessToken, "refresh-token-switched"),
      body: JSON.stringify({
        code: 0,
        data: sessionSnapshot(switchedAccessToken),
      }),
    });
  });

  await page.route(/\/api\/v1\/orgs\/1$/, async (route) => {
    const body = route.request().postDataJSON() as { name: string };
    mutationRequests.push({
      authorization: requestAuth(route.request()),
      body,
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: new URL(route.request().url()).pathname,
    });
    organizationRecords = organizationRecords.map((organization) =>
      organization.id === "1"
        ? {
            ...organization,
            name: body.name,
            updatedAt: "2026-06-18T10:00:00Z",
          }
        : organization,
    );
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: organizationRecords.find((organization) => organization.id === "1"),
      }),
    });
  });

  await page.route(/\/api\/v1\/orgs(?:\?.*)?$/, async (route) => {
    const url = new URL(route.request().url());
    if (route.request().method() === "POST") {
      const body = route.request().postDataJSON() as { code: string; name: string };
      mutationRequests.push({
        authorization: requestAuth(route.request()),
        body,
        locale: route.request().headers()["x-locale"] ?? null,
        method: route.request().method(),
        path: url.pathname,
      });
      const createdOrganization: IAMOrganization = {
        code: body.code,
        createdAt: "2026-06-18T10:30:00Z",
        id: "4",
        name: body.name,
        status: "active",
        updatedAt: "2026-06-18T10:30:00Z",
      };
      organizationRecords = [...organizationRecords, createdOrganization];
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: createdOrganization }),
      });
      return;
    }

    listRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
    const filteredOrganizations = organizationRecords.filter((organization) =>
      organizationMatchesQuery(organization, url.searchParams),
    );
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: filteredOrganizations,
          page: Number(url.searchParams.get("page") ?? "1"),
          pageSize: Number(url.searchParams.get("pageSize") ?? "10"),
          storageStatus: "persisted",
          total: filteredOrganizations.length,
        },
      }),
    });
  });

  await page.goto("/admin/organizations");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.organizations.title }),
  ).toBeVisible();
  await expect(page.getByText("Root Org", { exact: true })).toBeVisible();
  await expect(page.getByText("Beta Org", { exact: true })).toBeVisible();

  const createPanel = page.locator(".aoi-admin-panel").filter({
    has: page.getByRole("heading", { name: zhCN.admin.organizations.create.title }),
  });
  await createPanel.getByLabel(zhCN.admin.organizations.fields.code).fill("alpha");
  await createPanel.getByLabel(zhCN.admin.organizations.fields.name).fill("Alpha Org");
  await createPanel.getByRole("button", { name: zhCN.admin.organizations.actions.create }).click();
  await expect(page.getByText(zhCN.admin.organizations.create.successTitle)).toBeVisible();
  await expect(page.getByText("Alpha Org", { exact: true })).toBeVisible();

  const currentPanel = page.locator(".aoi-admin-panel").filter({
    has: page.getByRole("heading", { name: zhCN.admin.organizations.current.title }),
  });
  await currentPanel.getByLabel(zhCN.admin.organizations.fields.name).fill("Root Operations");
  await currentPanel.getByRole("button", { name: zhCN.admin.organizations.actions.save }).click();
  await expect(page.getByText(zhCN.admin.organizations.current.successTitle)).toBeVisible();
  await expect(page.getByText("Root Operations", { exact: true })).toBeVisible();

  await page
    .getByRole("row", { name: /Beta Org/ })
    .getByRole("button", { name: zhCN.admin.organizations.actions.switch })
    .click();
  await expect(page.getByText(zhCN.admin.organizations.switch.successTitle)).toBeVisible();
  await expect
    .poll(
      async () =>
        (await page.context().cookies()).find((cookie) => cookie.name === "aoi_access")?.value ??
        "",
    )
    .toBe(switchedAccessToken);

  const filterPanel = page.locator(".aoi-admin-panel").filter({
    has: page.getByRole("heading", { name: zhCN.admin.organizations.filters.title }),
  });
  await filterPanel.getByLabel(zhCN.admin.organizations.filters.keyword).fill("beta");
  await filterPanel.getByLabel(zhCN.admin.organizations.fields.code).fill("beta");
  await filterPanel.getByLabel(zhCN.admin.organizations.fields.name).fill("Beta");
  await filterPanel.getByLabel(zhCN.admin.organizations.fields.status).selectOption("active");
  await filterPanel.getByLabel(zhCN.admin.organizations.filters.pageSize).fill("25");
  await filterPanel.getByRole("button", { name: zhCN.admin.organizations.actions.search }).click();
  await expect
    .poll(() =>
      listRequests.some(
        (request) =>
          request.query.code === "beta" &&
          request.query.keyword === "beta" &&
          request.query.name === "Beta" &&
          request.query.pageSize === "25" &&
          request.query.status === "active",
      ),
    )
    .toBeTruthy();

  const filteredRequest = listRequests.find(
    (request) => request.query.code === "beta" && request.query.pageSize === "25",
  );
  expect(new Set(listRequests.map((request) => request.path))).toEqual(new Set(["/api/v1/orgs"]));
  expect(new Set(listRequests.map((request) => request.locale))).toEqual(new Set(["zh-CN"]));
  expect(
    listRequests.every((request) =>
      [`Bearer ${accessToken}`, `Bearer ${switchedAccessToken}`].includes(
        request.authorization ?? "",
      ),
    ),
  ).toBeTruthy();
  expect(
    listRequests.every((request) => request.path === "/api/v1/orgs" && request.query.page),
  ).toBeTruthy();
  expect(filteredRequest?.query).toMatchObject({
    code: "beta",
    desc: "true",
    keyword: "beta",
    name: "Beta",
    orderKey: "id",
    page: "1",
    pageSize: "25",
    status: "active",
  });
  expect(mutationRequests).toEqual([
    {
      authorization: `Bearer ${accessToken}`,
      body: { code: "alpha", name: "Alpha Org" },
      locale: "zh-CN",
      method: "POST",
      path: "/api/v1/orgs",
    },
    {
      authorization: `Bearer ${accessToken}`,
      body: { name: "Root Operations" },
      locale: "zh-CN",
      method: "PATCH",
      path: "/api/v1/orgs/1",
    },
    {
      authorization: `Bearer ${accessToken}`,
      body: { orgId: 2 },
      locale: "zh-CN",
      method: "POST",
      path: "/api/v1/auth/switch-org",
    },
  ]);
  expect(new Set(identityRequests.map((request) => request.path))).toEqual(
    new Set(["/api/v1/me", "/api/v1/me/orgs"]),
  );
  expect(new Set(identityRequests.map((request) => request.locale))).toEqual(new Set(["zh-CN"]));
  expect(
    identityRequests.every((request) =>
      [`Bearer ${accessToken}`, `Bearer ${switchedAccessToken}`].includes(
        request.authorization ?? "",
      ),
    ),
  ).toBeTruthy();
});

test("admin users route manages backend-supported users and invitations", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const userListRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
    query: Record<string, string>;
  }> = [];
  const roleRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
  }> = [];
  const invitationListRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
  }> = [];
  const mutationRequests: Array<{
    authorization: string | null;
    body?: unknown;
    locale: string | null;
    method: string;
    path: string;
  }> = [];
  let userRecords: IAMOrganizationUser[] = [
    {
      membershipStatus: "active",
      roles: ["role:owner"],
      user: {
        createdAt: "2026-06-18T07:00:00Z",
        displayName: "Owner User",
        email: "owner@example.com",
        id: "1",
        lastLoginAt: "2026-06-18T09:00:00Z",
        mfaEnabled: true,
        status: "active",
        updatedAt: "2026-06-18T09:00:00Z",
        username: "owner",
      },
    },
    {
      membershipStatus: "active",
      roles: ["role:member"],
      user: {
        createdAt: "2026-06-18T07:30:00Z",
        displayName: "Member User",
        email: "member@example.com",
        id: "2",
        lastLoginAt: null,
        mfaEnabled: false,
        status: "active",
        updatedAt: "2026-06-18T07:30:00Z",
        username: "member",
      },
    },
  ];
  let invitationRecords: IAMInvitation[] = [
    {
      createdAt: "2026-06-18T08:00:00Z",
      email: "pending@example.com",
      expiresAt: "2099-01-01T00:00:00Z",
      id: "800",
      invitedBy: "1",
      orgId: "1",
      roleCode: "member",
      status: "pending",
      updatedAt: "2026-06-18T08:00:00Z",
    },
  ];

  await page.route(/\/api\/v1\/orgs\/1\/roles$/, async (route) => {
    const url = new URL(route.request().url());
    roleRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: url.pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: [
          {
            code: "owner",
            createdAt: "2026-06-18T07:00:00Z",
            description: "Full access",
            id: "10",
            name: "Owner Role",
            orgId: "1",
            permissions: ["*"],
            system: true,
            updatedAt: "2026-06-18T07:00:00Z",
          },
          {
            code: "admin",
            createdAt: "2026-06-18T07:00:00Z",
            description: "Admin access",
            id: "11",
            name: "Admin Role",
            orgId: "1",
            permissions: ["user:read", "user:update"],
            system: true,
            updatedAt: "2026-06-18T07:00:00Z",
          },
          {
            code: "member",
            createdAt: "2026-06-18T07:00:00Z",
            description: "Member access",
            id: "12",
            name: "Member Role",
            orgId: "1",
            permissions: ["profile:read"],
            system: true,
            updatedAt: "2026-06-18T07:00:00Z",
          },
        ],
      }),
    });
  });

  await page.route(/\/api\/v1\/orgs\/1\/users\/2$/, async (route) => {
    const url = new URL(route.request().url());
    const body = route.request().postDataJSON() as { roles?: string[]; status?: string };
    mutationRequests.push({
      authorization: requestAuth(route.request()),
      body,
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: url.pathname,
    });
    userRecords = userRecords.map((record) =>
      record.user.id === "2"
        ? {
            ...record,
            membershipStatus: body.status ?? record.membershipStatus,
            roles: body.roles ? body.roles.map((role) => `role:${role}`) : record.roles,
            user: {
              ...record.user,
              updatedAt: "2026-06-18T10:00:00Z",
            },
          }
        : record,
    );
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: userRecords.find((record) => record.user.id === "2") }),
    });
  });

  await page.route(/\/api\/v1\/orgs\/1\/users(?:\?.*)?$/, async (route) => {
    const url = new URL(route.request().url());
    userListRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: userRecords,
          page: Number(url.searchParams.get("page") ?? "1"),
          pageSize: Number(url.searchParams.get("pageSize") ?? "10"),
          storageStatus: "persisted",
          total: userRecords.length,
        },
      }),
    });
  });

  await page.route(/\/api\/v1\/orgs\/1\/invitations\/800$/, async (route) => {
    const url = new URL(route.request().url());
    mutationRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: url.pathname,
    });
    invitationRecords = invitationRecords.map((record) =>
      record.id === "800"
        ? { ...record, status: "revoked", updatedAt: "2026-06-18T10:30:00Z" }
        : record,
    );
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: { revoked: true } }),
    });
  });

  await page.route(/\/api\/v1\/orgs\/1\/users\/invitations$/, async (route) => {
    const url = new URL(route.request().url());
    const body = route.request().postDataJSON() as { email: string; roleCode: string };
    mutationRequests.push({
      authorization: requestAuth(route.request()),
      body,
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: url.pathname,
    });
    invitationRecords = [
      ...invitationRecords,
      {
        createdAt: "2026-06-18T10:15:00Z",
        email: body.email,
        expiresAt: "2099-01-02T00:00:00Z",
        id: "801",
        invitedBy: "1",
        orgId: "1",
        roleCode: body.roleCode,
        status: "pending",
        updatedAt: "2026-06-18T10:15:00Z",
      },
    ];
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          debug: true,
          token: "invite-token-801",
          url: "http://127.0.0.1:9999/invitations/invite-token-801",
        },
      }),
    });
  });

  await page.route(/\/api\/v1\/orgs\/1\/invitations$/, async (route) => {
    const url = new URL(route.request().url());
    invitationListRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: url.pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: invitationRecords }),
    });
  });

  await page.goto("/admin/users");

  await expect(page.getByRole("heading", { level: 1, name: zhCN.admin.users.title })).toBeVisible();
  const userTable = page.locator(".aoi-user-table");
  await expect(userTable.getByText("owner@example.com", { exact: true })).toBeVisible();
  await expect(userTable.getByText("member@example.com", { exact: true })).toBeVisible();
  await expect(
    userTable.getByText(zhCN.admin.users.status.active, { exact: true }).first(),
  ).toBeVisible();

  await page.getByRole("button", { name: zhCN.admin.users.actions.invite }).click();
  const inviteForm = page.locator(".aoi-user-invite-form");
  await inviteForm.getByLabel(zhCN.admin.users.invite.email).fill("new@example.com");
  await inviteForm.getByLabel(zhCN.admin.users.invite.role).selectOption("admin");
  await page.getByRole("button", { name: zhCN.admin.users.actions.sendInvitation }).click();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.users.invite.successTitle }),
  ).toBeVisible();
  const deliveryPanel = page.locator(".aoi-user-delivery-panel");
  await expect(
    deliveryPanel.getByText(zhCN.admin.users.invite.debugToken, { exact: true }),
  ).toBeVisible();
  await expect(deliveryPanel.getByText("invite-token-801", { exact: true })).toBeVisible();

  await userTable.locator(".aoi-data-table-wrap").evaluate((element) => {
    element.scrollLeft = element.scrollWidth;
  });
  const memberRow = userTable.getByRole("row").filter({ hasText: "Member User" });
  await scrollTableActionsIntoView(memberRow);
  await memberRow
    .getByLabel(zhCN.admin.users.actions.roleSelect.replace("{{name}}", "Member User"), {
      exact: true,
    })
    .selectOption("admin");
  await scrollTableActionsIntoView(memberRow);
  await memberRow.locator("button").filter({ hasText: zhCN.admin.users.actions.saveRole }).focus();
  await page.keyboard.press("Enter");
  await expect(page.getByText(zhCN.admin.users.member.roleSuccessTitle)).toBeVisible();

  await scrollTableActionsIntoView(memberRow);
  await memberRow.locator("button").filter({ hasText: zhCN.admin.users.actions.disable }).focus();
  await page.keyboard.press("Enter");
  await expect(page.getByText(zhCN.admin.users.member.statusSuccessTitle)).toBeVisible();

  await page.locator(".aoi-user-invitation-table .aoi-data-table-wrap").evaluate((element) => {
    element.scrollLeft = element.scrollWidth;
  });
  await page
    .getByRole("button", {
      name: zhCN.admin.users.invitation.revokeInvitation.replace(
        "{{email}}",
        "pending@example.com",
      ),
    })
    .focus();
  await page.keyboard.press("Enter");
  await expect(page.getByText(zhCN.admin.users.invitation.confirmRevokeTitle)).toBeVisible();
  await page.getByRole("button", { name: zhCN.admin.users.actions.confirmRevoke }).click();
  await expect(
    page.getByRole("heading", {
      level: 2,
      name: zhCN.admin.users.invitation.revokeSuccessTitle,
    }),
  ).toBeVisible();

  const filterForm = page.locator(".aoi-user-filter-form");
  await filterForm.getByLabel(zhCN.admin.users.filters.keyword).fill("member");
  await filterForm.getByLabel(zhCN.admin.users.filters.username).fill("member");
  await filterForm.getByLabel(zhCN.admin.users.filters.displayName).fill("Member");
  await filterForm.getByLabel(zhCN.admin.users.filters.email).fill("member@example.com");
  await filterForm.getByLabel(zhCN.admin.users.filters.role).selectOption("admin");
  await filterForm.getByLabel(zhCN.admin.users.filters.status).selectOption("disabled");
  await filterForm.getByLabel(zhCN.admin.users.filters.pageSize).fill("30");
  await page.getByRole("button", { name: zhCN.admin.users.actions.search }).focus();
  await page.keyboard.press("Enter");
  await expect
    .poll(() =>
      userListRequests.some(
        (request) =>
          request.query.keyword === "member" &&
          request.query.username === "member" &&
          request.query.displayName === "Member" &&
          request.query.email === "member@example.com" &&
          request.query.roleCode === "admin" &&
          request.query.status === "disabled",
      ),
    )
    .toBeTruthy();

  const filteredRequest = userListRequests.find((request) => request.query.keyword === "member");
  expect(new Set(userListRequests.map((request) => request.path))).toEqual(
    new Set(["/api/v1/orgs/1/users"]),
  );
  expect(new Set(roleRequests.map((request) => request.path))).toEqual(
    new Set(["/api/v1/orgs/1/roles"]),
  );
  expect(new Set(invitationListRequests.map((request) => request.path))).toEqual(
    new Set(["/api/v1/orgs/1/invitations"]),
  );
  for (const request of [...userListRequests, ...roleRequests, ...invitationListRequests]) {
    expect(request.authorization).toBe(`Bearer ${accessToken}`);
    expect(request.locale).toBe("zh-CN");
  }
  expect(mutationRequests).toEqual([
    {
      authorization: `Bearer ${accessToken}`,
      body: { email: "new@example.com", roleCode: "admin" },
      locale: "zh-CN",
      method: "POST",
      path: "/api/v1/orgs/1/users/invitations",
    },
    {
      authorization: `Bearer ${accessToken}`,
      body: { roles: ["admin"] },
      locale: "zh-CN",
      method: "PATCH",
      path: "/api/v1/orgs/1/users/2",
    },
    {
      authorization: `Bearer ${accessToken}`,
      body: { status: "disabled" },
      locale: "zh-CN",
      method: "PATCH",
      path: "/api/v1/orgs/1/users/2",
    },
    {
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      method: "DELETE",
      path: "/api/v1/orgs/1/invitations/800",
    },
  ]);
  expect(filteredRequest?.query).toMatchObject({
    desc: "true",
    displayName: "Member",
    email: "member@example.com",
    keyword: "member",
    orderKey: "id",
    page: "1",
    pageSize: "30",
    roleCode: "admin",
    status: "disabled",
    username: "member",
  });
});

test("admin roles route creates and updates backend-supported roles", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const readRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
  }> = [];
  const mutationRequests: Array<{
    authorization: string | null;
    body?: unknown;
    locale: string | null;
    method: string;
    path: string;
  }> = [];
  let roleRecords: IAMRole[] = [
    {
      code: "owner",
      createdAt: "2026-06-18T07:00:00Z",
      description: "Full access",
      id: "10",
      name: "Owner Role",
      orgId: "1",
      permissions: ["*"],
      system: true,
      updatedAt: "2026-06-18T07:00:00Z",
    },
    {
      code: "auditor",
      createdAt: "2026-06-18T08:00:00Z",
      description: "Read audit data",
      id: "20",
      name: "Audit Role",
      orgId: "1",
      permissions: ["audit:read", "user:read"],
      system: false,
      updatedAt: "2026-06-18T08:00:00Z",
    },
  ];
  const permissionRecords: IAMPermission[] = [
    {
      code: "audit:read",
      createdAt: "2026-06-18T07:00:00Z",
      description: "Read audit logs",
      id: "100",
      name: "Read audit logs",
      updatedAt: "2026-06-18T07:00:00Z",
    },
    {
      code: "role:read",
      createdAt: "2026-06-18T07:00:00Z",
      description: "Read roles",
      id: "101",
      name: "Read roles",
      updatedAt: "2026-06-18T07:00:00Z",
    },
    {
      code: "user:read",
      createdAt: "2026-06-18T07:00:00Z",
      description: "Read users",
      id: "102",
      name: "Read users",
      updatedAt: "2026-06-18T07:00:00Z",
    },
    {
      code: "user:update",
      createdAt: "2026-06-18T07:00:00Z",
      description: "Update users",
      id: "103",
      name: "Update users",
      updatedAt: "2026-06-18T07:00:00Z",
    },
  ];

  await page.route(/\/api\/v1\/orgs\/1\/roles\/20$/, async (route) => {
    const body = route.request().postDataJSON() as {
      description: string;
      name: string;
      permissions: string[];
    };
    mutationRequests.push({
      authorization: requestAuth(route.request()),
      body,
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: new URL(route.request().url()).pathname,
    });
    const updatedAt = "2026-06-18T11:00:00Z";
    roleRecords = roleRecords.map((role) =>
      role.id === "20"
        ? {
            ...role,
            description: body.description,
            name: body.name,
            permissions: body.permissions,
            updatedAt,
          }
        : role,
    );
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: roleRecords.find((role) => role.id === "20") }),
    });
  });

  await page.route(/\/api\/v1\/orgs\/1\/roles$/, async (route) => {
    const url = new URL(route.request().url());
    if (route.request().method() === "POST") {
      const body = route.request().postDataJSON() as {
        code: string;
        description: string;
        name: string;
        permissions: string[];
      };
      mutationRequests.push({
        authorization: requestAuth(route.request()),
        body,
        locale: route.request().headers()["x-locale"] ?? null,
        method: route.request().method(),
        path: url.pathname,
      });
      const createdRole: IAMRole = {
        code: body.code,
        createdAt: "2026-06-18T10:00:00Z",
        description: body.description,
        id: "21",
        name: body.name,
        orgId: "1",
        permissions: body.permissions,
        system: false,
        updatedAt: "2026-06-18T10:00:00Z",
      };
      roleRecords = [...roleRecords, createdRole];
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: createdRole }),
      });
      return;
    }
    readRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: url.pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: roleRecords }),
    });
  });

  await page.route(/\/api\/v1\/orgs\/1\/permissions$/, async (route) => {
    readRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: permissionRecords }),
    });
  });

  await page.goto("/admin/roles");

  await expect(page.getByRole("heading", { level: 1, name: zhCN.admin.roles.title })).toBeVisible();
  const roleTable = page.locator(".aoi-role-table");
  await expect(roleTable.getByText("Owner Role", { exact: true })).toBeVisible();
  await expect(roleTable.getByText("Audit Role", { exact: true })).toBeVisible();
  await expect(roleTable.getByText("audit:read", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: zhCN.admin.roles.actions.create }).click();
  const createPanel = page.locator(".aoi-admin-panel").filter({
    has: page.getByRole("heading", { level: 2, name: zhCN.admin.roles.create.title }),
  });
  await createPanel.getByLabel(zhCN.admin.roles.fields.code).fill("operator");
  await createPanel.getByLabel(zhCN.admin.roles.fields.name).fill("Operator Role");
  await createPanel.getByLabel(zhCN.admin.roles.fields.description).fill("Operate user records");
  const createPermissionPanel = createPanel.locator(".aoi-role-permission-panel").filter({
    has: page.getByRole("heading", { name: zhCN.admin.roles.permissions.createTitle }),
  });
  await createPermissionPanel
    .getByLabel(zhCN.admin.roles.permissions.toggle.replace("{{code}}", "user:read"))
    .check();
  await createPermissionPanel
    .getByLabel(zhCN.admin.roles.permissions.toggle.replace("{{code}}", "user:update"))
    .check();
  await createPanel.getByRole("button", { name: zhCN.admin.roles.actions.submitCreate }).click();
  await expect(page.getByText(zhCN.admin.roles.create.successTitle)).toBeVisible();
  await expect(roleTable.getByText("Operator Role", { exact: true })).toBeVisible();

  const editPanel = page.locator(".aoi-admin-panel").filter({
    has: page.getByRole("heading", { level: 2, name: zhCN.admin.roles.edit.title }),
  });
  await editPanel.getByLabel(zhCN.admin.roles.edit.role).selectOption("20");
  await editPanel.getByLabel(zhCN.admin.roles.fields.name).fill("Audit Operator");
  await editPanel.getByLabel(zhCN.admin.roles.fields.description).fill("Audit and user operations");
  const editPermissionPanel = editPanel.locator(".aoi-role-permission-panel").filter({
    has: page.getByRole("heading", { name: zhCN.admin.roles.permissions.editTitle }),
  });
  await editPermissionPanel
    .getByLabel(zhCN.admin.roles.permissions.toggle.replace("{{code}}", "user:update"))
    .check();
  await editPanel.getByRole("button", { name: zhCN.admin.roles.actions.save }).click();
  await expect(page.getByText(zhCN.admin.roles.edit.successTitle)).toBeVisible();
  await expect(roleTable.getByText("Audit Operator", { exact: true })).toBeVisible();

  expect(new Set(readRequests.map((request) => request.path))).toEqual(
    new Set(["/api/v1/orgs/1/permissions", "/api/v1/orgs/1/roles"]),
  );
  for (const request of readRequests) {
    expect(request.authorization).toBe(`Bearer ${accessToken}`);
    expect(request.locale).toBe("zh-CN");
  }
  expect(mutationRequests).toEqual([
    {
      authorization: `Bearer ${accessToken}`,
      body: {
        code: "operator",
        description: "Operate user records",
        name: "Operator Role",
        permissions: ["user:read", "user:update"],
      },
      locale: "zh-CN",
      method: "POST",
      path: "/api/v1/orgs/1/roles",
    },
    {
      authorization: `Bearer ${accessToken}`,
      body: {
        description: "Audit and user operations",
        name: "Audit Operator",
        permissions: ["audit:read", "user:read", "user:update"],
      },
      locale: "zh-CN",
      method: "PATCH",
      path: "/api/v1/orgs/1/roles/20",
    },
  ]);
});

test("admin sessions route renders backend session records with filters and revocation", async ({
  page,
}) => {
  const accessToken = accessTokenWithOrg("1", { sessionId: "sess-current" });
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
    query: Record<string, string>;
  }> = [];
  const mutationRequests: Array<{
    authorization: string | null;
    locale: string | null;
    method: string;
    path: string;
  }> = [];
  let sessionRecords = [
    {
      createdAt: "2026-06-18T08:00:00Z",
      expiresAt: "2099-01-01T00:00:00Z",
      id: "sess-current",
      ipAddress: "127.0.0.1",
      lastUsedAt: "2026-06-18T09:00:00Z",
      orgId: "1",
      updatedAt: "2026-06-18T09:00:00Z",
      userAgent: "Playwright Chromium",
      userId: "1",
    },
    {
      createdAt: "2026-06-18T08:30:00Z",
      expiresAt: "2099-01-01T00:00:00Z",
      id: "sess-active",
      ipAddress: "127.0.0.2",
      lastUsedAt: "2026-06-18T09:10:00Z",
      orgId: "1",
      updatedAt: "2026-06-18T09:10:00Z",
      userAgent: "Edge on Windows",
      userId: "4",
    },
    {
      createdAt: "2026-06-18T07:00:00Z",
      expiresAt: "2099-01-01T00:00:00Z",
      id: "sess-revoked",
      ipAddress: "10.0.0.8",
      lastUsedAt: "2026-06-18T08:00:00Z",
      orgId: "1",
      revokedAt: "2026-06-18T08:30:00Z",
      updatedAt: "2026-06-18T08:30:00Z",
      userAgent: "Firefox on Windows",
      userId: "2",
    },
    {
      createdAt: "2020-01-01T00:00:00Z",
      expiresAt: "2020-01-02T00:00:00Z",
      id: "sess-expired",
      ipAddress: "10.0.0.9",
      lastUsedAt: "2020-01-01T01:00:00Z",
      orgId: "1",
      updatedAt: "2020-01-01T01:00:00Z",
      userAgent: "Safari on iOS",
      userId: "3",
    },
  ];

  await page.route(/\/api\/v1\/orgs\/1\/sessions\/sess-active$/, async (route) => {
    const url = new URL(route.request().url());
    mutationRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: url.pathname,
    });
    const revokedAt = "2026-06-18T10:00:00Z";
    sessionRecords = sessionRecords.map((session) =>
      session.id === "sess-active" ? { ...session, revokedAt, updatedAt: revokedAt } : session,
    );
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: { revoked: true } }),
    });
  });

  await page.route(/\/api\/v1\/orgs\/1\/sessions(?:\?.*)?$/, async (route) => {
    const url = new URL(route.request().url());
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: sessionRecords,
          page: Number(url.searchParams.get("page") ?? "1"),
          pageSize: Number(url.searchParams.get("pageSize") ?? "10"),
          storageStatus: "persisted",
          total: sessionRecords.length,
        },
      }),
    });
  });

  await page.goto("/admin/sessions");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.sessions.title }),
  ).toBeVisible();
  const sessionTable = page.locator(".aoi-session-table");
  await expect(sessionTable.getByText("Playwright Chromium", { exact: true })).toBeVisible();
  await expect(sessionTable.getByText("Edge on Windows", { exact: true })).toBeVisible();
  await expect(sessionTable.getByText("Firefox on Windows", { exact: true })).toBeVisible();
  await expect(sessionTable.getByText("Safari on iOS", { exact: true })).toBeVisible();
  await expect(
    sessionTable.getByText(zhCN.admin.sessions.actions.currentSession, { exact: true }),
  ).toBeVisible();
  await expect(
    sessionTable.getByRole("button", {
      name: zhCN.admin.sessions.actions.revokeSession.replace("{{id}}", "sess-current"),
    }),
  ).toHaveCount(0);
  await expect(
    sessionTable.getByText(zhCN.admin.sessions.status.active, { exact: true }).first(),
  ).toBeVisible();
  await expect(
    sessionTable.getByText(zhCN.admin.sessions.status.revoked, { exact: true }).first(),
  ).toBeVisible();
  await expect(
    sessionTable.getByText(zhCN.admin.sessions.status.expired, { exact: true }).first(),
  ).toBeVisible();

  await page
    .getByRole("button", {
      name: zhCN.admin.sessions.actions.revokeSession.replace("{{id}}", "sess-active"),
    })
    .click();
  await expect(page.getByText(zhCN.admin.sessions.revoke.confirmTitle)).toBeVisible();
  await page.getByRole("button", { name: zhCN.admin.sessions.actions.confirmRevoke }).click();
  await expect.poll(() => mutationRequests.length).toBe(1);
  await expect(
    page.getByText(zhCN.admin.sessions.revoke.successDescription.replace("{{id}}", "sess-active")),
  ).toBeVisible();
  await expect(
    sessionTable
      .getByRole("row")
      .filter({ hasText: "sess-active" })
      .getByText(zhCN.admin.sessions.status.revoked, { exact: true }),
  ).toBeVisible();

  await page.getByLabel(zhCN.admin.sessions.filters.scope).selectOption("self");
  await page.getByLabel(zhCN.admin.sessions.filters.keyword).fill("firefox");
  await page.getByLabel(zhCN.admin.sessions.filters.userId).fill("2");
  await page.getByLabel(zhCN.admin.sessions.filters.ipAddress).fill("10.0.0");
  await page.getByLabel(zhCN.admin.sessions.filters.status).selectOption("revoked");
  await page.getByRole("button", { name: zhCN.admin.sessions.actions.search }).click();
  await expect
    .poll(() =>
      protectedRequests.some(
        (request) =>
          request.query.keyword === "firefox" &&
          request.query.userId === "2" &&
          request.query.ipAddress === "10.0.0" &&
          request.query.status === "revoked",
      ),
    )
    .toBeTruthy();

  const filteredRequest = protectedRequests.find((request) => request.query.keyword === "firefox");
  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set(["/api/v1/orgs/1/sessions"]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      path: request.path,
      query: request.query,
    })),
  );
  expect(mutationRequests).toEqual([
    {
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      method: "DELETE",
      path: "/api/v1/orgs/1/sessions/sess-active",
    },
  ]);
  expect(filteredRequest?.query).toMatchObject({
    desc: "true",
    ipAddress: "10.0.0",
    keyword: "firefox",
    orderKey: "last_used_at",
    page: "1",
    pageSize: "10",
    status: "revoked",
    userId: "2",
  });
  expect(filteredRequest?.query.scope).toBeUndefined();
});

test("admin security route manages backend-supported MFA setup and logout", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1", { sessionId: "sess-current" });
  await setAuthenticatedSession(page, accessToken);
  const identityRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
  }> = [];
  const mutationRequests: Array<{
    authorization: string | null;
    body?: unknown;
    locale: string | null;
    method: string;
    path: string;
  }> = [];
  let mfaEnabled = false;
  const mfaSecret = "JBSWY3DPEHPK3PXP";
  const otpauthUrl = "otpauth://totp/Aoi:owner@example.com?secret=JBSWY3DPEHPK3PXP&issuer=Aoi";

  await page.route(/\/api\/v1\/me\/orgs$/, async (route) => {
    identityRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: [
          {
            code: "root",
            id: "1",
            name: "Root Org",
          },
        ],
      }),
    });
  });

  await page.route(/\/api\/v1\/me$/, async (route) => {
    identityRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          displayName: "Owner User",
          email: "owner@example.com",
          id: "1",
          mfaEnabled,
          username: "owner",
        },
      }),
    });
  });

  await page.route(/\/api\/v1\/auth\/mfa\/setup$/, async (route) => {
    mutationRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          otpauthUrl,
          secret: mfaSecret,
        },
      }),
    });
  });

  await page.route(/\/api\/v1\/auth\/mfa\/verify$/, async (route) => {
    const body = route.request().postDataJSON() as { code: string };
    mutationRequests.push({
      authorization: requestAuth(route.request()),
      body,
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: new URL(route.request().url()).pathname,
    });
    mfaEnabled = true;
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          verified: true,
        },
      }),
    });
  });

  await page.route(/\/api\/v1\/auth\/logout$/, async (route) => {
    mutationRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          loggedOut: true,
        },
      }),
    });
  });

  await page.goto("/admin/security");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.security.title }),
  ).toBeVisible();
  await expect(
    page.locator(".aoi-security-grid").getByText("owner@example.com", { exact: true }),
  ).toBeVisible();
  await expect(page.getByText("sess-current", { exact: true })).toBeVisible();
  await expect(
    page.locator(".aoi-iam-status").getByText(zhCN.admin.security.mfa.disabledBadge, {
      exact: true,
    }),
  ).toBeVisible();

  await page
    .getByRole("button", { exact: true, name: zhCN.admin.security.actions.generateSecret })
    .click();
  await expect(
    page.getByText(zhCN.admin.security.messages.secretGeneratedTitle, { exact: true }),
  ).toBeVisible();
  await expect(page.getByLabel(zhCN.admin.security.mfa.otpauthLabel)).toHaveValue(otpauthUrl);
  await expect(page.getByLabel(zhCN.admin.security.mfa.secretLabel)).toHaveValue(mfaSecret);
  let storageDump = await page.evaluate(() =>
    JSON.stringify({
      local: Object.fromEntries(
        Array.from({ length: window.localStorage.length }, (_, index) => {
          const key = window.localStorage.key(index) ?? "";
          return [key, window.localStorage.getItem(key)];
        }),
      ),
      session: Object.fromEntries(
        Array.from({ length: window.sessionStorage.length }, (_, index) => {
          const key = window.sessionStorage.key(index) ?? "";
          return [key, window.sessionStorage.getItem(key)];
        }),
      ),
    }),
  );
  expect(storageDump).not.toContain(mfaSecret);
  expect(storageDump).not.toContain(otpauthUrl);

  await page.getByLabel(zhCN.admin.security.fields.mfaCode).fill("123456");
  await page
    .getByRole("button", { exact: true, name: zhCN.admin.security.actions.verifyAndEnable })
    .click();
  await expect.poll(() => mutationRequests.filter((request) => request.body).length).toBe(1);
  await expect(
    page.getByText(zhCN.admin.security.messages.enabledTitle, { exact: true }),
  ).toBeVisible();
  await expect(
    page.locator(".aoi-iam-status").getByText(zhCN.admin.security.mfa.enabledBadge, {
      exact: true,
    }),
  ).toBeVisible();
  await expect(page.getByLabel(zhCN.admin.security.mfa.secretLabel)).toHaveCount(0);
  storageDump = await page.evaluate(() =>
    JSON.stringify({
      local: Object.fromEntries(
        Array.from({ length: window.localStorage.length }, (_, index) => {
          const key = window.localStorage.key(index) ?? "";
          return [key, window.localStorage.getItem(key)];
        }),
      ),
      session: Object.fromEntries(
        Array.from({ length: window.sessionStorage.length }, (_, index) => {
          const key = window.sessionStorage.key(index) ?? "";
          return [key, window.sessionStorage.getItem(key)];
        }),
      ),
    }),
  );
  expect(storageDump).not.toContain(mfaSecret);
  expect(storageDump).not.toContain(otpauthUrl);

  await page.getByRole("button", { exact: true, name: zhCN.admin.security.actions.logout }).click();
  await expect
    .poll(() => mutationRequests.some((request) => request.path === "/api/v1/auth/logout"))
    .toBeTruthy();
  await expect(page).toHaveURL(/\/login(?:\?|$)/);
  await expect
    .poll(() => page.evaluate(() => window.sessionStorage.getItem("aoi-admin-session")))
    .toBeNull();

  expect(new Set(identityRequests.map((request) => request.path))).toEqual(
    new Set(["/api/v1/me", "/api/v1/me/orgs"]),
  );
  expect(identityRequests).toEqual(
    identityRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      path: request.path,
    })),
  );
  expect(mutationRequests).toEqual([
    {
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      method: "POST",
      path: "/api/v1/auth/mfa/setup",
    },
    {
      authorization: `Bearer ${accessToken}`,
      body: { code: "123456" },
      locale: "zh-CN",
      method: "POST",
      path: "/api/v1/auth/mfa/verify",
    },
    {
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      method: "POST",
      path: "/api/v1/auth/logout",
    },
  ]);
});

test("admin design system route edits local theme drafts without backend theme calls", async ({
  page,
}) => {
  const accessToken = accessTokenWithOrg("1", { sessionId: "sess-current" });
  await setAuthenticatedSession(page, accessToken);
  const themeApiRequests: string[] = [];
  page.on("request", (request) => {
    const url = request.url();
    if (/\/api\/v1\/.*(?:theme|design-system|appearance)/.test(url)) {
      themeApiRequests.push(url);
    }
  });
  await page.route(/\/api\/v1\/me\/session$/, async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          accessExpiresAt: "2099-01-01T00:00:00Z",
          accessToken,
          clientType: "pc_web",
          orgId: "1",
          productCode: "aoi",
          refreshExpiresAt: "2099-01-02T00:00:00Z",
          refreshToken: "refresh-token",
          sessionId: "sess-current",
        },
      }),
    });
  });
  await page.route(/\/api\/v1\/me\/orgs$/, async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: [
          {
            code: "root",
            id: "1",
            name: "Root Org",
          },
        ],
      }),
    });
  });
  await page.route(/\/api\/v1\/me$/, async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          displayName: "Owner User",
          email: "owner@example.com",
          id: "1",
          mfaEnabled: false,
          username: "owner",
        },
      }),
    });
  });

  await page.goto("/admin/design-system");

  await expect(page.getByRole("heading", { level: 1, name: zhDesignSystem.title })).toBeVisible();
  await expect(page.getByRole("heading", { name: zhDesignSystem.package.title })).toBeVisible();
  await expect(
    page.locator(".aoi-theme-package-card").getByText("builtin/aoi", { exact: true }),
  ).toBeVisible();
  await page.getByLabel(zhDesignSystem.fields.mode.label).selectOption("dark");
  await page.getByLabel(zhDesignSystem.fields.primaryColor.label).fill("#315f5a");
  await page.getByRole("button", { name: zhDesignSystem.actions.saveDraft }).click();
  await expect(page.getByText(zhDesignSystem.messages.savedTitle)).toBeVisible();
  await expect(page.locator(".aoi-theme-preview")).toHaveAttribute("data-mode", "dark");

  const importedTheme = JSON.stringify({
    accentColor: "#5f7b71",
    mode: "light",
    motionIntensity: "reduced",
    primaryColor: "#2f6f6a",
    radiusScale: 10,
    shadowLevel: "standard",
    spacingScale: 1.1,
    surfaceColor: "#ffffff",
    textColor: "#202322",
    typographyScale: 1.02,
  });
  await page.getByLabel(zhDesignSystem.import.textarea, { exact: true }).fill(importedTheme);
  await page.getByRole("button", { name: zhDesignSystem.actions.applyImport }).click();
  await expect(page.getByText(zhDesignSystem.messages.importedTitle)).toBeVisible();
  await expect(page.locator(".aoi-theme-color-field code").first()).toHaveText("#2f6f6a");
  await expect(
    page.getByRole("button", { name: zhDesignSystem.actions.sourcePackage }),
  ).toBeDisabled();
  await expect(
    page.getByRole("button", { name: zhDesignSystem.actions.backendDisabled }),
  ).toBeDisabled();
  expect(themeApiRequests).toEqual([]);
});

test("admin API tokens route issues and revokes backend API tokens", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
    query: Record<string, string>;
  }> = [];
  const createRequests: Array<{
    authorization: string | null;
    body: unknown;
    locale: string | null;
    method: string;
    path: string;
  }> = [];
  const mutationRequests: Array<{
    authorization: string | null;
    locale: string | null;
    method: string;
    path: string;
  }> = [];
  let tokenRecords: IAMAPIToken[] = [
    {
      createdAt: "2026-06-18T08:00:00Z",
      createdBy: "1",
      expiresAt: "2026-12-31T00:00:00Z",
      id: "token-active",
      lastUsedAt: "2026-06-18T09:00:00Z",
      lastUsedIpAddress: "127.0.0.1",
      orgId: "1",
      remark: "CLI automation",
      revokedAt: null,
      revokedBy: null,
      roleCode: "owner",
      status: "active",
      tokenPrefix: "aoi_pat_live",
      updatedAt: "2026-06-18T09:00:00Z",
      userDisplayName: "Owner User",
      userId: "1",
      username: "owner",
    },
    {
      createdAt: "2026-06-17T08:00:00Z",
      createdBy: "1",
      expiresAt: null,
      id: "token-revoked",
      lastUsedAt: null,
      lastUsedIpAddress: "",
      orgId: "1",
      remark: "Old integration",
      revokedAt: "2026-06-18T08:30:00Z",
      revokedBy: "1",
      roleCode: "viewer",
      status: "revoked",
      tokenPrefix: "aoi_pat_old",
      updatedAt: "2026-06-18T08:30:00Z",
      userDisplayName: "Member User",
      userId: "2",
      username: "member",
    },
    {
      createdAt: "2025-12-01T08:00:00Z",
      createdBy: "1",
      expiresAt: "2026-01-01T00:00:00Z",
      id: "token-expired",
      lastUsedAt: "2025-12-20T08:00:00Z",
      lastUsedIpAddress: "10.0.0.3",
      orgId: "1",
      remark: "",
      revokedAt: null,
      revokedBy: null,
      roleCode: "viewer",
      status: "active",
      tokenPrefix: "aoi_pat_exp",
      updatedAt: "2025-12-20T08:00:00Z",
      userDisplayName: "Member User",
      userId: "2",
      username: "member",
    },
  ];

  function recordProtectedRequest(routeUrl: string, headers: Record<string, string>) {
    const url = new URL(routeUrl);
    protectedRequests.push({
      authorization: authFromHeaders(headers),
      locale: headers["x-locale"] ?? null,
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
  }

  await page.route(/\/api\/v1\/orgs\/1\/users(?:\?.*)?$/, async (route) => {
    recordProtectedRequest(route.request().url(), route.request().headers());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: [
            {
              membershipStatus: "active",
              roles: ["role:owner", "viewer"],
              user: {
                createdAt: "2026-06-18T08:00:00Z",
                displayName: "Owner User",
                email: "owner@example.com",
                id: "1",
                lastLoginAt: "2026-06-18T09:00:00Z",
                mfaEnabled: true,
                status: "active",
                updatedAt: "2026-06-18T09:00:00Z",
                username: "owner",
              },
            },
            {
              membershipStatus: "active",
              roles: ["viewer"],
              user: {
                createdAt: "2026-06-18T08:00:00Z",
                displayName: "Member User",
                email: "member@example.com",
                id: "2",
                lastLoginAt: null,
                mfaEnabled: false,
                status: "active",
                updatedAt: "2026-06-18T09:00:00Z",
                username: "member",
              },
            },
          ],
          page: 1,
          pageSize: 100,
          storageStatus: "persisted",
          total: 2,
        },
      }),
    });
  });
  await page.route(/\/api\/v1\/orgs\/1\/roles(?:\?.*)?$/, async (route) => {
    recordProtectedRequest(route.request().url(), route.request().headers());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: [
          {
            code: "owner",
            createdAt: "2026-06-18T08:00:00Z",
            description: "Full organization access",
            id: "10",
            name: "Owner Role",
            orgId: "1",
            permissions: ["api_token:read", "api_token:create", "api_token:revoke"],
            system: true,
            updatedAt: "2026-06-18T09:00:00Z",
          },
          {
            code: "viewer",
            createdAt: "2026-06-18T08:00:00Z",
            description: "Read-only access",
            id: "11",
            name: "Viewer Role",
            orgId: "1",
            permissions: ["api_token:read"],
            system: true,
            updatedAt: "2026-06-18T09:00:00Z",
          },
        ],
      }),
    });
  });
  await page.route(/\/api\/v1\/orgs\/1\/api-tokens\/token-active$/, async (route) => {
    const url = new URL(route.request().url());
    mutationRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: url.pathname,
    });
    const revokedAt = "2026-06-18T10:00:00Z";
    tokenRecords = tokenRecords.map((token) =>
      token.id === "token-active"
        ? { ...token, revokedAt, status: "revoked", updatedAt: revokedAt }
        : token,
    );
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: { revoked: true } }),
    });
  });
  await page.route(/\/api\/v1\/orgs\/1\/api-tokens(?:\?.*)?$/, async (route) => {
    const url = new URL(route.request().url());
    if (route.request().method() === "POST") {
      createRequests.push({
        authorization: requestAuth(route.request()),
        body: route.request().postDataJSON(),
        locale: route.request().headers()["x-locale"] ?? null,
        method: route.request().method(),
        path: url.pathname,
      });
      const created = {
        createdAt: "2026-06-18T10:00:00Z",
        createdBy: "1",
        expiresAt: "2026-07-18T10:00:00Z",
        id: "token-created",
        lastUsedAt: null,
        lastUsedIpAddress: "",
        orgId: "1",
        remark: "CLI automation",
        revokedAt: null,
        revokedBy: null,
        roleCode: "owner",
        status: "active",
        tokenPrefix: "aoi_pat_new",
        updatedAt: "2026-06-18T10:00:00Z",
        userDisplayName: "Owner User",
        userId: "1",
        username: "owner",
      };
      tokenRecords = [created, ...tokenRecords];
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          code: 0,
          data: {
            item: created,
            token: "aoi_pat_secret_new",
          },
        }),
      });
      return;
    }

    recordProtectedRequest(route.request().url(), route.request().headers());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: tokenRecords,
          page: Number(url.searchParams.get("page") ?? "1"),
          pageSize: Number(url.searchParams.get("pageSize") ?? "10"),
          storageStatus: "persisted",
          total: tokenRecords.length,
        },
      }),
    });
  });

  await page.goto("/admin/api-tokens");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.apiTokens.title }),
  ).toBeVisible();
  const tokenTable = page.locator(".aoi-api-token-table");
  await expect(tokenTable.getByText("Owner User", { exact: true }).first()).toBeVisible();
  await expect(tokenTable.getByText("aoi_pat_live", { exact: true })).toBeVisible();
  await expect(tokenTable.getByText("CLI automation", { exact: true })).toBeVisible();
  await expect(
    tokenTable.getByText(zhCN.admin.apiTokens.status.active, { exact: true }).first(),
  ).toBeVisible();
  await expect(
    tokenTable.getByText(zhCN.admin.apiTokens.status.revoked, { exact: true }).first(),
  ).toBeVisible();
  await expect(
    tokenTable.getByText(zhCN.admin.apiTokens.status.expired, { exact: true }).first(),
  ).toBeVisible();

  await page.getByRole("button", { name: zhCN.admin.apiTokens.actions.issue }).click();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.apiTokens.issue.title }),
  ).toBeVisible();
  await page.getByLabel(zhCN.admin.apiTokens.issue.user, { exact: true }).selectOption("1");
  await page.getByLabel(zhCN.admin.apiTokens.issue.role, { exact: true }).selectOption("owner");
  await page.getByLabel(zhCN.admin.apiTokens.issue.validity, { exact: true }).selectOption("30");
  await page.getByLabel(zhCN.admin.apiTokens.issue.remark, { exact: true }).fill("CLI automation");
  await page.getByRole("button", { name: zhCN.admin.apiTokens.actions.create }).click();
  await expect.poll(() => createRequests.length).toBe(1);
  expect(createRequests[0]).toEqual({
    authorization: `Bearer ${accessToken}`,
    body: {
      days: 30,
      remark: "CLI automation",
      roleCode: "owner",
      userId: "1",
    },
    locale: "zh-CN",
    method: "POST",
    path: "/api/v1/orgs/1/api-tokens",
  });
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.apiTokens.issued.title }),
  ).toBeVisible();
  await expect(page.getByText(zhCN.admin.apiTokens.issued.warningTitle)).toBeVisible();
  await expect(page.getByText("aoi_pat_secret_new", { exact: true })).toBeVisible();
  const storageDump = await page.evaluate(() =>
    JSON.stringify({
      local: Object.fromEntries(
        Array.from({ length: window.localStorage.length }, (_, index) => {
          const key = window.localStorage.key(index) ?? "";
          return [key, window.localStorage.getItem(key)];
        }),
      ),
      session: Object.fromEntries(
        Array.from({ length: window.sessionStorage.length }, (_, index) => {
          const key = window.sessionStorage.key(index) ?? "";
          return [key, window.sessionStorage.getItem(key)];
        }),
      ),
    }),
  );
  expect(storageDump).not.toContain("aoi_pat_secret_new");

  await page
    .getByRole("button", {
      name: zhCN.admin.apiTokens.actions.revokeToken.replace("{{id}}", "token-active"),
    })
    .click();
  await expect(page.getByText(zhCN.admin.apiTokens.revoke.confirmTitle)).toBeVisible();
  await page.getByRole("button", { name: zhCN.admin.apiTokens.actions.confirmRevoke }).click();
  await expect.poll(() => mutationRequests.length).toBe(1);
  await expect(
    page.getByText(
      zhCN.admin.apiTokens.revoke.successDescription.replace("{{id}}", "token-active"),
    ),
  ).toBeVisible();
  await expect(
    tokenTable
      .getByRole("row")
      .filter({ hasText: "token-active" })
      .getByText(zhCN.admin.apiTokens.status.revoked, { exact: true }),
  ).toBeVisible();

  await page.getByLabel(zhCN.admin.apiTokens.filters.userId).fill("1");
  await page.getByLabel(zhCN.admin.apiTokens.filters.status).selectOption("revoked");
  await page.getByRole("button", { name: zhCN.admin.apiTokens.actions.search }).click();
  await expect
    .poll(() =>
      protectedRequests.some(
        (request) => request.query.userId === "1" && request.query.status === "revoked",
      ),
    )
    .toBeTruthy();

  const filteredRequest = protectedRequests.find(
    (request) => request.query.userId === "1" && request.query.status === "revoked",
  );
  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set(["/api/v1/orgs/1/api-tokens", "/api/v1/orgs/1/roles", "/api/v1/orgs/1/users"]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      path: request.path,
      query: request.query,
    })),
  );
  expect(mutationRequests).toEqual([
    {
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      method: "DELETE",
      path: "/api/v1/orgs/1/api-tokens/token-active",
    },
  ]);
  expect(filteredRequest?.query).toMatchObject({
    page: "1",
    pageSize: "10",
    status: "revoked",
    userId: "1",
  });
});

test("admin audit logs route renders backend audit records with filters", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
    query: Record<string, string>;
  }> = [];

  await page.route(/\/api\/v1\/orgs\/1\/audit-logs(?:\?.*)?$/, async (route) => {
    const url = new URL(route.request().url());
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: [
          {
            action: "auth.login",
            createdAt: "2026-06-18T09:00:00Z",
            id: "901",
            ipAddress: "127.0.0.1",
            metadata: JSON.stringify({ email: "member@example.com", method: "password" }),
            orgId: "1",
            resource: "session",
            resourceId: "sess-active",
            userAgent: "Playwright Chromium",
            userId: "1",
          },
          {
            action: "user.update",
            createdAt: "2026-06-18T08:00:00Z",
            id: "900",
            ipAddress: "10.0.0.8",
            metadata: "{}",
            orgId: "1",
            resource: "user",
            resourceId: "2",
            userAgent: "Firefox on Windows",
            userId: "2",
          },
        ],
      }),
    });
  });

  await page.goto("/admin/audit-logs");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.auditLogs.title }),
  ).toBeVisible();
  const auditTable = page.locator(".aoi-audit-table");
  await expect(auditTable.getByText("auth.login", { exact: true })).toBeVisible();
  await expect(auditTable.getByText("user.update", { exact: true })).toBeVisible();
  await expect(auditTable.getByText("session", { exact: true })).toBeVisible();
  await expect(auditTable.getByText("member@example.com", { exact: false })).toBeVisible();

  await page.getByLabel(zhCN.admin.auditLogs.filters.action).fill("auth.login");
  await page.getByLabel(zhCN.admin.auditLogs.filters.userId).fill("1");
  await page.getByLabel(zhCN.admin.auditLogs.filters.from).fill("2026-06-18T08:00");
  await page.getByLabel(zhCN.admin.auditLogs.filters.to).fill("2026-06-18T10:00");
  await page.getByLabel(zhCN.admin.auditLogs.filters.cursor).fill("900");
  await page.getByLabel(zhCN.admin.auditLogs.filters.limit).fill("25");
  await page.getByRole("button", { name: zhCN.admin.auditLogs.actions.search }).click();
  await expect
    .poll(() =>
      protectedRequests.some(
        (request) =>
          request.query.action === "auth.login" &&
          request.query.userId === "1" &&
          request.query.cursor === "900" &&
          request.query.limit === "25",
      ),
    )
    .toBeTruthy();

  const filteredRequest = protectedRequests.find(
    (request) => request.query.action === "auth.login",
  );
  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set(["/api/v1/orgs/1/audit-logs"]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      path: request.path,
      query: request.query,
    })),
  );
  expect(filteredRequest?.query).toMatchObject({
    action: "auth.login",
    cursor: "900",
    limit: "25",
    userId: "1",
  });
  expect(filteredRequest?.query.from).toMatch(/Z$/);
  expect(filteredRequest?.query.to).toMatch(/Z$/);
});

test("admin login logs route renders auth.login audit records with local IP filter", async ({
  page,
}) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
    query: Record<string, string>;
  }> = [];

  await page.route(/\/api\/v1\/orgs\/1\/audit-logs(?:\?.*)?$/, async (route) => {
    const url = new URL(route.request().url());
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: [
          {
            action: "auth.login",
            createdAt: "2026-06-18T09:00:00Z",
            id: "951",
            ipAddress: "127.0.0.1",
            metadata: "{}",
            orgId: "1",
            resource: "session",
            resourceId: "sess-local",
            userAgent: "Playwright Chrome",
            userId: "1",
          },
          {
            action: "auth.login",
            createdAt: "2026-06-18T08:00:00Z",
            id: "950",
            ipAddress: "10.0.0.9",
            metadata: "{}",
            orgId: "1",
            resource: "session",
            resourceId: "sess-remote",
            userAgent: "Firefox",
            userId: "2",
          },
          {
            action: "user.update",
            createdAt: "2026-06-18T07:00:00Z",
            id: "949",
            ipAddress: "127.0.0.8",
            metadata: "{}",
            orgId: "1",
            resource: "user",
            resourceId: "2",
            userAgent: "Firefox",
            userId: "2",
          },
        ],
      }),
    });
  });

  await page.goto("/admin/login-logs");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.loginLogs.title }),
  ).toBeVisible();
  const loginTable = page.locator(".aoi-login-log-table");
  await expect(loginTable.getByText("sess-local", { exact: true })).toBeVisible();
  await expect(loginTable.getByText("sess-remote", { exact: true })).toBeVisible();
  await expect(loginTable.getByText("user.update", { exact: true })).toHaveCount(0);

  await page.getByLabel(zhCN.admin.loginLogs.filters.userId).fill("1");
  await page.getByLabel(zhCN.admin.loginLogs.filters.ipAddress).fill("127.0.0");
  await page.getByLabel(zhCN.admin.loginLogs.filters.from).fill("2026-06-18T08:00");
  await page.getByLabel(zhCN.admin.loginLogs.filters.to).fill("2026-06-18T10:00");
  await page.getByLabel(zhCN.admin.loginLogs.filters.limit).fill("25");
  await page.getByRole("button", { name: zhCN.admin.loginLogs.actions.search }).click();
  await expect(loginTable.getByText("sess-local", { exact: true })).toBeVisible();
  await expect(loginTable.getByText("sess-remote", { exact: true })).toHaveCount(0);
  await expect
    .poll(() =>
      protectedRequests.some(
        (request) =>
          request.query.action === "auth.login" &&
          request.query.userId === "1" &&
          request.query.limit === "25",
      ),
    )
    .toBeTruthy();

  const filteredRequest = protectedRequests.find((request) => request.query.limit === "25");
  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set(["/api/v1/orgs/1/audit-logs"]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      path: request.path,
      query: request.query,
    })),
  );
  expect(filteredRequest?.query).toMatchObject({
    action: "auth.login",
    limit: "25",
    userId: "1",
  });
  expect(filteredRequest?.query).not.toHaveProperty("ipAddress");
  expect(filteredRequest?.query.from).toMatch(/Z$/);
  expect(filteredRequest?.query.to).toMatch(/Z$/);
});

test("admin probes route renders public health and readiness probes", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const probeRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
  }> = [];

  await page.route("**/health", async (route) => {
    probeRequests.push({
      authorization: route.request().headers().authorization ?? null,
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: { status: "ok" } }),
    });
  });
  await page.route("**/ready", async (route) => {
    probeRequests.push({
      authorization: route.request().headers().authorization ?? null,
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: { checks: { database: "ok" }, status: "ready" } }),
    });
  });

  await page.goto("/admin/probes");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.probes.title }),
  ).toBeVisible();
  await expect(page.getByText("/health", { exact: true }).first()).toBeVisible();
  await expect(page.getByText("/ready", { exact: true }).first()).toBeVisible();
  await expect(page.getByText("database", { exact: true })).toBeVisible();
  await expect(page.getByText(zhCN.admin.probes.status.ok, { exact: true }).first()).toBeVisible();
  await expect(
    page.getByText(zhCN.admin.probes.status.ready, { exact: true }).first(),
  ).toBeVisible();
  expect(new Set(probeRequests.map((request) => request.path))).toEqual(
    new Set(["/health", "/ready"]),
  );
  expect(probeRequests).toEqual(
    probeRequests.map((request) => ({
      authorization: null,
      locale: "zh-CN",
      path: request.path,
    })),
  );
});

test("admin probes route renders degraded readiness details from 503 payload", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const probeRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
  }> = [];

  await page.route("**/health", async (route) => {
    probeRequests.push({
      authorization: route.request().headers().authorization ?? null,
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: { status: "ok" } }),
    });
  });
  await page.route("**/ready", async (route) => {
    probeRequests.push({
      authorization: route.request().headers().authorization ?? null,
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      status: 503,
      body: JSON.stringify({
        code: 1001,
        data: { checks: { database: "missing" }, status: "not_ready" },
        message: "not ready",
      }),
    });
  });

  await page.goto("/admin/probes");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.probes.title }),
  ).toBeVisible();
  await expect(
    page.getByText(zhCN.admin.probes.status.notReady, { exact: true }).first(),
  ).toBeVisible();
  await expect(
    page.getByText(zhCN.admin.probes.status.missing, { exact: true }).first(),
  ).toBeVisible();
  await expect(page.getByText("database", { exact: true })).toBeVisible();
  expect(new Set(probeRequests.map((request) => request.path))).toEqual(
    new Set(["/health", "/ready"]),
  );
  expect(probeRequests).toEqual(
    probeRequests.map((request) => ({
      authorization: null,
      locale: "zh-CN",
      path: request.path,
    })),
  );
});

test("admin API catalog route renders backend route entries with local filters", async ({
  page,
}) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    body: string | null;
    locale: string | null;
    method: string;
    path: string;
  }> = [];
  let apiGroups: SystemAPIGroup[] = [
    {
      code: "system",
      count: 4,
      label: "system",
      items: [
        {
          access: "permission",
          code: "get /api/v1/system/apis",
          description: "API catalog",
          group: "system",
          method: "GET",
          order: 10,
          path: "/api/v1/system/apis",
          permission: "permission:read",
          permissionRegistered: true,
          synced: true,
          syncedAt: "2026-06-18T10:00:00Z",
        },
        {
          access: "permission",
          code: "post /api/v1/system/apis/permissions/sync",
          description: "sync permissions",
          group: "system",
          method: "POST",
          order: 11,
          path: "/api/v1/system/apis/permissions/sync",
          permission: "permission:sync",
          permissionRegistered: false,
          synced: false,
        },
        {
          access: "authenticated",
          code: "get /api/v1/system/menus",
          description: "menus",
          group: "system",
          method: "GET",
          order: 20,
          path: "/api/v1/system/menus",
          permissionRegistered: true,
          synced: true,
        },
        {
          access: "public",
          code: "get /api/v1/system/public-settings",
          description: "public settings",
          group: "system",
          method: "GET",
          order: 30,
          path: "/api/v1/system/public-settings",
          permissionRegistered: true,
          synced: false,
        },
      ],
    },
  ];

  await page.route("**/api/v1/system/apis/sync", async (route) => {
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      body: route.request().postData(),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: new URL(route.request().url()).pathname,
    });
    apiGroups = apiGroups.map((group) => ({
      ...group,
      items: group.items.map((item) => ({
        ...item,
        synced: true,
        syncedAt: "2026-06-18T11:00:00Z",
      })),
    }));
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          created: 1,
          groups: apiGroups,
          persisted: true,
          stale: 0,
          storageStatus: "persisted",
          syncedAt: "2026-06-18T11:00:00Z",
          total: 4,
          updated: 3,
        },
      }),
    });
  });

  await page.route("**/api/v1/system/apis/permissions/sync", async (route) => {
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      body: route.request().postData(),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: new URL(route.request().url()).pathname,
    });
    apiGroups = apiGroups.map((group) => ({
      ...group,
      items: group.items.map((item) =>
        item.permission === "permission:sync" ? { ...item, permissionRegistered: true } : item,
      ),
    }));
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          created: 1,
          items: [
            {
              code: "permission:sync",
              created: true,
              description: "Sync permissions from registered APIs",
              exists: false,
              name: "Sync permissions",
            },
          ],
          persisted: true,
          skipped: 1,
          storageStatus: "persisted",
          syncedAt: "2026-06-18T11:01:00Z",
          total: 2,
        },
      }),
    });
  });

  await page.route("**/api/v1/system/apis", async (route) => {
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      body: route.request().postData(),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: apiGroups,
      }),
    });
  });

  await page.goto("/admin/apis");

  await expect(page.getByRole("heading", { level: 1, name: zhCN.admin.apis.title })).toBeVisible();
  const apiTable = page.locator(".aoi-api-table");
  await expect(apiTable.getByText("/api/v1/system/apis", { exact: true })).toBeVisible();
  await expect(apiTable.getByText("permission:read", { exact: true })).toBeVisible();
  await expect(apiTable.getByText("permission:sync", { exact: true })).toBeVisible();
  await expect(
    apiTable.getByText(zhCN.admin.apis.status.unregistered, { exact: true }),
  ).toBeVisible();
  await expect(apiTable.getByText("/api/v1/system/public-settings", { exact: true })).toBeVisible();
  await page.getByLabel(zhCN.admin.apis.filters.keyword).fill("public-settings");
  await expect(apiTable.getByText("/api/v1/system/public-settings", { exact: true })).toBeVisible();
  await expect(apiTable.getByText("permission:read", { exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: zhCN.admin.apis.actions.reset }).click();
  await expect(apiTable.getByText("permission:read", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: zhCN.admin.apis.actions.syncRoutes }).click();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.apis.sync.routesConfirmTitle }),
  ).toBeVisible();
  await page.getByRole("button", { name: zhCN.admin.apis.sync.confirm }).click();
  await expect(page.getByText(zhCN.admin.apis.sync.routesSuccessTitle)).toBeVisible();
  await expect(apiTable.getByText(zhCN.admin.apis.status.unsynced, { exact: true })).toHaveCount(0);

  await page.getByRole("button", { name: zhCN.admin.apis.actions.syncPermissions }).click();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.apis.sync.permissionsConfirmTitle }),
  ).toBeVisible();
  await page.getByRole("button", { name: zhCN.admin.apis.sync.confirm }).click();
  await expect(page.getByText(zhCN.admin.apis.sync.permissionsSuccessTitle)).toBeVisible();
  await expect(
    apiTable.getByText(zhCN.admin.apis.status.unregistered, { exact: true }),
  ).toHaveCount(0);
  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set([
      "/api/v1/system/apis",
      "/api/v1/system/apis/permissions/sync",
      "/api/v1/system/apis/sync",
    ]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      body: null,
      locale: "zh-CN",
      method: request.method,
      path: request.path,
    })),
  );
  expect(protectedRequests.filter((request) => request.method === "POST")).toEqual([
    {
      authorization: `Bearer ${accessToken}`,
      body: null,
      locale: "zh-CN",
      method: "POST",
      path: "/api/v1/system/apis/sync",
    },
    {
      authorization: `Bearer ${accessToken}`,
      body: null,
      locale: "zh-CN",
      method: "POST",
      path: "/api/v1/system/apis/permissions/sync",
    },
  ]);
});

test("admin menu catalog route renders backend-filtered menu groups", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
  }> = [];

  await page.route("**/api/v1/system/menus", async (route) => {
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: new URL(route.request().url()).pathname,
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: [
          {
            code: "workspace",
            label: "workspace",
            order: 10,
            items: [
              {
                code: "dashboard",
                icon: "layout-dashboard",
                label: "dashboard",
                mobile: true,
                order: 10,
                path: "/",
              },
              {
                code: "users",
                icon: "users",
                label: "users",
                mobile: true,
                order: 30,
                path: "/users",
                permission: "user:read",
              },
            ],
          },
          {
            code: "system",
            label: "system",
            order: 30,
            items: [
              {
                code: "menus",
                icon: "panel-left",
                label: "menus",
                mobile: false,
                order: 10,
                path: "/menus",
                permission: "permission:read",
              },
              {
                code: "versions",
                icon: "package-check",
                label: "versions",
                mobile: false,
                order: 60,
                path: "/versions",
                permission: "version:read",
              },
            ],
          },
        ],
      }),
    });
  });

  await page.goto("/admin/menus");

  await expect(page.getByRole("heading", { level: 1, name: zhCN.admin.menus.title })).toBeVisible();
  const menuCatalog = page.locator(".aoi-menu-groups");
  await expect(menuCatalog.getByText("workspace", { exact: true })).toBeVisible();
  await expect(menuCatalog.getByText("permission:read", { exact: true })).toBeVisible();
  await expect(menuCatalog.getByText("/versions", { exact: true })).toBeVisible();
  await page.getByLabel(zhCN.admin.menus.filters.keyword).fill("versions");
  await expect(menuCatalog.getByText("/versions", { exact: true })).toBeVisible();
  await expect(menuCatalog.getByText("permission:read", { exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: zhCN.admin.menus.actions.reset }).click();
  await expect(menuCatalog.getByText("permission:read", { exact: true })).toBeVisible();
  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set(["/api/v1/system/menus"]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      path: request.path,
    })),
  );
});

test("admin system settings route updates backend-editable configuration values", async ({
  page,
}) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    locale: string | null;
    method: string;
    path: string;
  }> = [];
  const patchBodies: Array<Record<string, unknown>> = [];

  await page.route("**/api/v1/system/config", async (route) => {
    const request = route.request();
    protectedRequests.push({
      authorization: requestAuth(request),
      locale: request.headers()["x-locale"] ?? null,
      method: request.method(),
      path: new URL(request.url()).pathname,
    });
    if (request.method() === "PATCH") {
      const body = request.postDataJSON() as Record<string, unknown>;
      patchBodies.push(body);
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: updatedSystemConfigSnapshot() }),
      });
      return;
    }
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          sections: [
            {
              code: "brand",
              description: "Runtime brand labels",
              groups: [
                {
                  description: "Public identity values",
                  items: [
                    {
                      description: "Visible product name",
                      editable: true,
                      key: "brand.productName",
                      label: "Product name",
                      secret: false,
                      source: "runtime",
                      testable: false,
                      value: "Aoi",
                      valueType: "string",
                    },
                    {
                      description: "Release display name",
                      editable: false,
                      key: "brand.versionName",
                      label: "Version name",
                      secret: false,
                      source: "build",
                      testable: false,
                      value: "dev",
                      valueType: "string",
                    },
                  ],
                  key: "identity",
                  label: "Identity",
                  testable: false,
                },
              ],
              icon: "settings",
              items: [],
              label: "Brand",
              order: 10,
            },
            {
              code: "i18n",
              description: "Locale defaults",
              groups: [
                {
                  description: "Language behavior",
                  items: [
                    {
                      description: "Default frontend language",
                      editable: true,
                      editor: "select",
                      key: "i18n.defaultLocale",
                      label: "Default locale",
                      options: [
                        { label: "Simplified Chinese", value: "zh-CN" },
                        { label: "English", value: "en-US" },
                      ],
                      secret: false,
                      source: "runtime",
                      testable: false,
                      value: "zh-CN",
                      valueType: "string",
                    },
                    {
                      description: "SMTP credential",
                      editable: true,
                      editor: "password",
                      key: "mail.password",
                      label: "SMTP password",
                      secret: true,
                      source: "runtime",
                      testable: false,
                      value: "raw-secret",
                      valueType: "string",
                    },
                  ],
                  key: "locale",
                  label: "Localization",
                  risk: "medium",
                  testable: true,
                },
              ],
              icon: "languages",
              items: [],
              label: "Localization",
              order: 20,
            },
          ],
        },
      }),
    });
  });

  await page.goto("/admin/system");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.system.title }),
  ).toBeVisible();
  await expect(page.getByText("brand.productName", { exact: true })).toBeVisible();
  await expect(page.getByText("Aoi", { exact: true }).first()).toBeVisible();
  await page.getByRole("button", { name: /Localization/ }).click();
  await expect(page.getByText("i18n.defaultLocale", { exact: true })).toBeVisible();
  await expect(page.getByText(zhCN.admin.system.values.secret, { exact: true })).toBeVisible();
  await expect(page.getByText("raw-secret", { exact: true })).toHaveCount(0);
  const localizationGroup = page.locator(".aoi-config-group-card").filter({
    hasText: "Localization",
  });
  await localizationGroup
    .getByRole("button", { name: zhCN.admin.system.actions.editGroup })
    .click();
  await page.getByLabel("Default locale").selectOption("en-US");
  await page.getByLabel(zhCN.admin.system.editor.newSecretValue).fill("rotated-secret");
  await expect(page.getByText(zhCN.admin.system.editor.secretChanged)).toBeVisible();
  await page.getByRole("button", { name: zhCN.admin.system.actions.saveChanges }).click();
  await expect(page.getByText(zhCN.admin.system.messages.updateSuccessTitle)).toBeVisible();
  await expect(page.getByText("en-US", { exact: true }).first()).toBeVisible();
  await expect(page.getByText("rotated-secret", { exact: true })).toHaveCount(0);

  expect(patchBodies).toEqual([
    {
      items: [
        { key: "i18n.defaultLocale", value: "en-US" },
        { key: "mail.password", value: "rotated-secret" },
      ],
      persist: true,
    },
  ]);
  expect(new Set(protectedRequests.map((request) => `${request.method} ${request.path}`))).toEqual(
    new Set(["GET /api/v1/system/config", "PATCH /api/v1/system/config"]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      method: request.method,
      path: request.path,
    })),
  );
});

function updatedSystemConfigSnapshot(): SystemConfigSnapshot {
  return {
    sections: [
      {
        code: "i18n",
        description: "Locale defaults",
        groups: [
          {
            description: "Language behavior",
            items: [
              {
                description: "Default frontend language",
                editable: true,
                editor: "select",
                key: "i18n.defaultLocale",
                label: "Default locale",
                options: [
                  { label: "Simplified Chinese", value: "zh-CN" },
                  { label: "English", value: "en-US" },
                ],
                secret: false,
                source: "runtime",
                testable: false,
                value: "en-US",
                valueType: "string",
              },
              {
                description: "SMTP credential",
                editable: true,
                editor: "password",
                key: "mail.password",
                label: "SMTP password",
                secret: true,
                source: "runtime",
                testable: false,
                value: "rotated-secret",
                valueType: "string",
              },
            ],
            key: "locale",
            label: "Localization",
            risk: "medium",
            testable: true,
          },
        ],
        icon: "languages",
        items: [],
        label: "Localization",
        order: 20,
      },
    ],
  };
}

test("admin media route manages assets through backend media contracts", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const fileBuffer = Buffer.from("aoi direct upload");
  const importText = "Imported hero|https://assets.example.test/hero.png";
  const protectedRequests: Array<{
    authorization: string | null;
    locale: string | null;
    method: string;
    path: string;
    query: Record<string, string>;
  }> = [];
  const uploadBodies: string[] = [];
  const importBodies: Array<Record<string, unknown>> = [];
  const renameBodies: Array<Record<string, unknown>> = [];
  const categoryBodies: Array<Record<string, unknown>> = [];
  const deleteRequests: string[] = [];
  const categoryDeleteRequests: string[] = [];
  let nextCategoryId = 12;
  let mediaCategories: SystemMediaCategory[] = [
    {
      children: [
        {
          children: [],
          createdAt: "2026-06-18T08:00:00Z",
          id: "11",
          name: "Blog covers",
          parentId: "10",
          sort: 20,
          updatedAt: "2026-06-18T09:00:00Z",
        },
      ],
      createdAt: "2026-06-18T08:00:00Z",
      id: "10",
      name: "Marketing",
      parentId: "0",
      sort: 10,
      updatedAt: "2026-06-18T09:00:00Z",
    },
  ];
  let mediaAssets = [
    {
      categoryId: "10",
      createdAt: "2026-06-18T09:00:00Z",
      displayName: "launch-cover.png",
      extension: "png",
      external: false,
      id: "asset-1",
      mimeType: "image/png",
      originalName: "launch-cover-original.png",
      sizeBytes: 2048000,
      source: "upload",
      storageKey: "media/launch-cover.png",
      updatedAt: "2026-06-18T09:00:00Z",
      uploadedBy: "1",
      uploadedByUsername: "owner",
      url: "",
    },
    {
      categoryId: "11",
      createdAt: "2026-06-17T09:00:00Z",
      displayName: "external-cdn.webp",
      extension: "webp",
      external: true,
      id: "asset-2",
      mimeType: "image/webp",
      originalName: "external-cdn.webp",
      sizeBytes: 1024,
      source: "url",
      storageKey: "",
      updatedAt: "2026-06-17T09:00:00Z",
      uploadedBy: "2",
      uploadedByUsername: "operator",
      url: "data:image/gif;base64,R0lGODlhAQABAAAAACw=",
    },
  ];
  const countMediaCategories = (items: SystemMediaCategory[]): number =>
    items.reduce((total, category) => total + 1 + countMediaCategories(category.children ?? []), 0);
  const findMediaCategory = (
    items: SystemMediaCategory[],
    id: number | string,
  ): SystemMediaCategory | null => {
    for (const category of items) {
      if (String(category.id) === String(id)) {
        return category;
      }
      const child = findMediaCategory(category.children ?? [], id);
      if (child) {
        return child;
      }
    }
    return null;
  };
  const insertMediaCategory = (category: SystemMediaCategory) => {
    if (!category.parentId || String(category.parentId) === "0") {
      mediaCategories = [...mediaCategories, category];
      return;
    }
    const parent = findMediaCategory(mediaCategories, category.parentId);
    if (!parent) {
      throw new Error(`missing parent category ${String(category.parentId)}`);
    }
    parent.children = [...(parent.children ?? []), category];
  };
  const removeMediaCategory = (items: SystemMediaCategory[], id: number | string): boolean => {
    const index = items.findIndex((category) => String(category.id) === String(id));
    if (index >= 0) {
      items.splice(index, 1);
      return true;
    }
    return items.some((category) => removeMediaCategory(category.children ?? [], id));
  };

  await page.route("**/api/v1/system/media/categories", async (route) => {
    const url = new URL(route.request().url());
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });

    if (route.request().method() === "POST") {
      const body = route.request().postDataJSON() as Record<string, unknown>;
      const categoryBody = body as {
        id?: number | string;
        name: string;
        parentId?: number | string;
        sort: number;
      };
      categoryBodies.push(body);
      const parentId =
        categoryBody.parentId === undefined ||
        categoryBody.parentId === null ||
        String(categoryBody.parentId) === ""
          ? "0"
          : String(categoryBody.parentId);

      if (categoryBody.id) {
        const existing = findMediaCategory(mediaCategories, String(categoryBody.id));
        if (!existing) {
          throw new Error(`missing category ${String(categoryBody.id)}`);
        }
        removeMediaCategory(mediaCategories, existing.id);
        const updated: SystemMediaCategory = {
          ...existing,
          children: existing.children ?? [],
          name: categoryBody.name,
          parentId,
          sort: categoryBody.sort,
          updatedAt: "2026-06-18T10:30:00Z",
        };
        insertMediaCategory(updated);
        await route.fulfill({
          contentType: "application/json",
          body: JSON.stringify({ code: 0, data: updated }),
        });
        return;
      }

      const created: SystemMediaCategory = {
        children: [],
        createdAt: "2026-06-18T10:25:00Z",
        id: String(nextCategoryId++),
        name: categoryBody.name,
        parentId,
        sort: categoryBody.sort,
        updatedAt: "2026-06-18T10:25:00Z",
      };
      insertMediaCategory(created);
      await route.fulfill({
        contentType: "application/json",
        status: 201,
        body: JSON.stringify({ code: 0, data: created }),
      });
      return;
    }

    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: mediaCategories,
          storageStatus: "persisted",
          total: countMediaCategories(mediaCategories),
        },
      }),
    });
  });

  await page.route("**/api/v1/system/media/categories/12", async (route) => {
    const url = new URL(route.request().url());
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
    categoryDeleteRequests.push("12");
    removeMediaCategory(mediaCategories, "12");
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: { deleted: true },
      }),
    });
  });

  await page.route(/\/api\/v1\/system\/media\/assets(?:\?.*)?$/, async (route) => {
    const url = new URL(route.request().url());
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: mediaAssets,
          objectStorage: "local",
          page: 1,
          pageSize: 10,
          storageStatus: "persisted",
          total: 2,
          uploadMaxBytes: 10485760,
          uploadMaxMb: 10,
          uploadUnavailable: false,
        },
      }),
    });
  });

  await page.route("**/api/v1/system/media/assets/asset-1/download", async (route) => {
    const url = new URL(route.request().url());
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
    await route.fulfill({
      body: Buffer.from("downloaded launch cover"),
      contentType: "image/png",
      headers: {
        "Content-Disposition": 'attachment; filename="launch-cover.png"',
      },
      status: 200,
    });
  });

  await page.route("**/api/v1/system/media/assets/asset-1", async (route) => {
    const url = new URL(route.request().url());
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });

    if (route.request().method() === "PATCH") {
      const body = route.request().postDataJSON() as Record<string, unknown>;
      renameBodies.push(body);
      expect(body).toEqual({ displayName: "Launch hero" });
      const updated = {
        ...mediaAssets[0],
        displayName: "Launch hero",
        updatedAt: "2026-06-18T10:20:00Z",
      };
      mediaAssets = [updated, ...mediaAssets.slice(1)];
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          code: 0,
          data: updated,
        }),
      });
      return;
    }

    if (route.request().method() === "DELETE") {
      deleteRequests.push("asset-1");
      mediaAssets = mediaAssets.filter((asset) => asset.id !== "asset-1");
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          code: 0,
          data: { deleted: true },
        }),
      });
      return;
    }

    await route.fallback();
  });

  await page.route("**/api/v1/system/media/assets/upload", async (route) => {
    const url = new URL(route.request().url());
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
    const body = route.request().postDataBuffer()?.toString("latin1") ?? "";
    uploadBodies.push(body);
    expect(body).toContain('name="file"; filename="direct-upload.txt"');
    expect(body).toContain(fileBuffer.toString());
    expect(body).toContain('name="categoryId"');
    expect(body).toContain("10");

    await route.fulfill({
      contentType: "application/json",
      status: 201,
      body: JSON.stringify({
        code: 0,
        data: {
          categoryId: "10",
          createdAt: "2026-06-18T10:00:00Z",
          displayName: "direct-upload.txt",
          extension: "txt",
          external: false,
          id: "asset-uploaded",
          mimeType: "text/plain",
          originalName: "direct-upload.txt",
          sizeBytes: fileBuffer.length,
          source: "upload",
          storageKey: "media/direct-upload.txt",
          updatedAt: "2026-06-18T10:00:00Z",
          uploadedBy: "1",
          uploadedByUsername: "owner",
          url: "",
        },
      }),
    });
  });

  await page.route("**/api/v1/system/media/assets/import-url", async (route) => {
    const url = new URL(route.request().url());
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      method: route.request().method(),
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
    const body = route.request().postDataJSON() as Record<string, unknown>;
    importBodies.push(body);
    expect(body).toEqual({
      categoryId: "10",
      text: importText,
    });

    await route.fulfill({
      contentType: "application/json",
      status: 201,
      body: JSON.stringify({
        code: 0,
        data: {
          imported: 1,
          items: [
            {
              categoryId: "10",
              createdAt: "2026-06-18T10:10:00Z",
              displayName: "Imported hero",
              extension: "png",
              external: true,
              id: "asset-imported",
              mimeType: "image/png",
              originalName: "Imported hero",
              sizeBytes: 0,
              source: "url",
              storageKey: "external:asset-imported",
              updatedAt: "2026-06-18T10:10:00Z",
              uploadedBy: "1",
              uploadedByUsername: "owner",
              url: "https://assets.example.test/hero.png",
            },
          ],
          storageStatus: "persisted",
        },
      }),
    });
  });

  await page.goto("/admin/media");

  await expect(page.getByRole("heading", { level: 1, name: zhCN.admin.media.title })).toBeVisible();
  await expect(page.getByText("launch-cover.png", { exact: true })).toBeVisible();
  await expect(page.getByText("external-cdn.webp", { exact: true }).first()).toBeVisible();
  await expect(page.getByText("Marketing", { exact: true }).first()).toBeVisible();
  await page.getByLabel(zhCN.admin.media.filters.keyword).fill("cover");
  await page.getByLabel(zhCN.admin.media.filters.keyword).press("Enter");
  await expect
    .poll(() => protectedRequests.some((request) => request.query.keyword === "cover"))
    .toBeTruthy();
  await page
    .locator(".aoi-media-category-button")
    .filter({ hasText: "Marketing" })
    .first()
    .press("Enter");
  await expect
    .poll(() => protectedRequests.some((request) => request.query.categoryId === "10"))
    .toBeTruthy();

  await page.getByRole("button", { name: zhCN.admin.media.actions.createCategory }).press("Enter");
  await page.getByLabel(zhCN.admin.media.categories.nameField).fill("Campaigns");
  await page.getByLabel(zhCN.admin.media.categories.sortField).fill("30");
  await page.getByRole("button", { name: zhCN.admin.media.actions.saveCategory }).click();
  await expect(page.getByText(zhCN.admin.media.messages.categoryCreatedTitle)).toBeVisible();
  expect(categoryBodies[0]).toEqual({
    name: "Campaigns",
    parentId: "10",
    sort: 30,
  });
  await expect(page.getByText("Campaigns", { exact: true }).first()).toBeVisible();

  await page
    .getByRole("button", {
      name: zhCN.admin.media.actions.editCategory.replace("{{name}}", "Campaigns"),
    })
    .press("Enter");
  await page.getByLabel(zhCN.admin.media.categories.nameField).fill("Campaign assets");
  await page.getByRole("button", { name: zhCN.admin.media.actions.saveCategory }).click();
  await expect(page.getByText(zhCN.admin.media.messages.categoryUpdatedTitle)).toBeVisible();
  expect(categoryBodies[1]).toEqual({
    id: "12",
    name: "Campaign assets",
    parentId: "10",
    sort: 30,
  });
  await expect(page.getByText("Campaign assets", { exact: true }).first()).toBeVisible();

  await page
    .getByRole("button", {
      name: zhCN.admin.media.actions.deleteCategory.replace("{{name}}", "Campaign assets"),
    })
    .press("Enter");
  await expect(page.getByText(zhCN.admin.media.categories.deleteTitle)).toBeVisible();
  await page
    .getByRole("button", { name: zhCN.admin.media.actions.confirmDeleteCategory })
    .press("Enter");
  await expect(page.getByText(zhCN.admin.media.messages.categoryDeletedTitle)).toBeVisible();
  expect(categoryDeleteRequests).toEqual(["12"]);
  await expect(
    page.locator(".aoi-media-category-list").getByText("Campaign assets", { exact: true }),
  ).toHaveCount(0);

  await page
    .getByRole("button", {
      name: zhCN.admin.media.actions.renameAsset.replace("{{name}}", "launch-cover.png"),
    })
    .press("Enter");
  await expect(page.getByRole("heading", { name: zhCN.admin.media.rename.title })).toBeVisible();
  await page.getByLabel(zhCN.admin.media.rename.field).fill("Launch hero");
  await page.getByRole("button", { name: zhCN.admin.media.actions.saveRename }).press("Enter");
  await expect(page.getByText(zhCN.admin.media.messages.renamedTitle)).toBeVisible();
  expect(renameBodies).toEqual([{ displayName: "Launch hero" }]);
  await expect(page.getByText("Launch hero", { exact: true }).first()).toBeVisible();

  await page
    .getByRole("button", {
      name: zhCN.admin.media.actions.downloadAsset.replace("{{name}}", "Launch hero"),
    })
    .press("Enter");
  await expect(page.getByText(zhCN.admin.media.messages.downloadedTitle)).toBeVisible();

  await page
    .getByRole("button", {
      name: zhCN.admin.media.actions.deleteAsset.replace("{{name}}", "Launch hero"),
    })
    .press("Enter");
  await expect(page.getByText(zhCN.admin.media.delete.confirmTitle)).toBeVisible();
  await page.getByRole("button", { name: zhCN.admin.media.actions.confirmDelete }).press("Enter");
  await expect(page.getByText(zhCN.admin.media.messages.deletedTitle)).toBeVisible();
  expect(deleteRequests).toEqual(["asset-1"]);

  await page.setInputFiles(`input[aria-label="${zhCN.admin.media.a11y.fileInput}"]`, {
    buffer: fileBuffer,
    mimeType: "text/plain",
    name: "direct-upload.txt",
  });
  await expect(page.getByText(zhCN.admin.media.messages.uploadedTitle)).toBeVisible();
  await page.getByLabel(zhCN.admin.media.write.import.label).fill(importText);
  await page.getByRole("button", { name: zhCN.admin.media.actions.importUrls }).press("Enter");
  await expect(page.getByText(zhCN.admin.media.messages.importedTitle)).toBeVisible();
  expect(uploadBodies).toHaveLength(1);
  expect(importBodies).toHaveLength(1);
  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set([
      "/api/v1/system/media/assets",
      "/api/v1/system/media/assets/asset-1",
      "/api/v1/system/media/assets/asset-1/download",
      "/api/v1/system/media/assets/import-url",
      "/api/v1/system/media/assets/upload",
      "/api/v1/system/media/categories",
      "/api/v1/system/media/categories/12",
    ]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      method: request.method,
      path: request.path,
      query: request.query,
    })),
  );
  const browserStorage = await page.evaluate(() =>
    JSON.stringify({
      local: { ...localStorage },
      session: { ...sessionStorage },
    }),
  );
  expect(browserStorage).not.toContain(fileBuffer.toString());
  expect(browserStorage).not.toContain(importText);
});

test("admin media resumable route uploads and aborts backend sessions", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const fileBuffer = Buffer.from("aoi resumable upload");
  const fileHash = createHash("sha256").update(fileBuffer).digest("hex");
  const chunkHash = fileHash;
  const protectedRequests: Array<{
    authorization: string | null;
    locale: string | null;
    method: string;
    path: string;
  }> = [];
  const checkBodies: Array<Record<string, unknown>> = [];
  const chunkBodies: string[] = [];
  const completeBodies: Array<Record<string, unknown>> = [];
  const abortBodies: Array<Record<string, unknown>> = [];
  let checkCall = 0;

  const captureProtectedRequest = (request: Request) => {
    const url = new URL(request.url());
    protectedRequests.push({
      authorization: requestAuth(request),
      locale: request.headers()["x-locale"] ?? null,
      method: request.method(),
      path: url.pathname,
    });
  };

  await page.route("**/api/v1/system/media/categories", async (route) => {
    captureProtectedRequest(route.request());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: [
            {
              children: [],
              createdAt: "2026-06-18T08:00:00Z",
              id: "10",
              name: "Marketing",
              parentId: "0",
              sort: 10,
              updatedAt: "2026-06-18T09:00:00Z",
            },
          ],
          storageStatus: "persisted",
          total: 1,
        },
      }),
    });
  });

  await page.route(/\/api\/v1\/system\/media\/assets(?:\?.*)?$/, async (route) => {
    captureProtectedRequest(route.request());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: [],
          objectStorage: "local",
          page: 1,
          pageSize: 1,
          storageStatus: "persisted",
          total: 0,
          uploadMaxBytes: 10485760,
          uploadMaxMb: 10,
          uploadUnavailable: false,
        },
      }),
    });
  });

  await page.route("**/api/v1/system/media/assets/resumable/check", async (route) => {
    captureProtectedRequest(route.request());
    const body = route.request().postDataJSON() as Record<string, unknown>;
    checkBodies.push(body);
    checkCall += 1;
    expect(body).toMatchObject({
      chunkSize: 1048576,
      chunkTotal: 1,
      fileHash,
      fileName: "upload-demo.txt",
      sizeBytes: fileBuffer.length,
    });
    expect(body).not.toHaveProperty("categoryId");

    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          chunkSize: 1048576,
          missingChunks: [0],
          progress: 0,
          session: {
            chunkSize: 1048576,
            chunkTotal: 1,
            completedAt: null,
            createdAt: "2026-06-18T08:00:00Z",
            expiresAt: "2026-06-19T08:00:00Z",
            fileHash,
            fileName: "upload-demo.txt",
            finalAssetId: null,
            id: checkCall === 1 ? "900" : "901",
            sizeBytes: fileBuffer.length,
            status: "active",
            updatedAt: "2026-06-18T08:00:00Z",
            uploadedBy: "1",
            uploadedByUsername: "owner",
          },
          storageStatus: "persisted",
          uploadMaxBytes: 10485760,
          uploadMaxMb: 10,
          uploadUnavailable: false,
          uploadedChunks: [],
        },
      }),
    });
  });

  await page.route("**/api/v1/system/media/assets/resumable/chunks", async (route) => {
    captureProtectedRequest(route.request());
    const body = route.request().postDataBuffer()?.toString("latin1") ?? "";
    chunkBodies.push(body);
    expect(body).toContain('name="chunkHash"');
    expect(body).toContain(chunkHash);
    expect(body).toContain('name="chunkIndex"');
    expect(body).toContain("0");
    expect(body).toContain('name="chunkTotal"');
    expect(body).toContain("1");
    expect(body).toContain('name="fileHash"');
    expect(body).toContain(fileHash);
    expect(body).toContain('name="fileName"');
    expect(body).toContain("upload-demo.txt");
    expect(body).toContain('name="sessionId"');
    expect(body).toContain("900");

    await route.fulfill({
      contentType: "application/json",
      status: 201,
      body: JSON.stringify({
        code: 0,
        data: {
          chunkIndex: 0,
          missingChunks: [],
          progress: 100,
          storageStatus: "persisted",
          uploadedChunks: [0],
        },
      }),
    });
  });

  await page.route("**/api/v1/system/media/assets/resumable/complete", async (route) => {
    captureProtectedRequest(route.request());
    const body = route.request().postDataJSON() as Record<string, unknown>;
    completeBodies.push(body);
    expect(body).toEqual({
      fileHash,
      sessionId: "900",
    });

    await route.fulfill({
      contentType: "application/json",
      status: 201,
      body: JSON.stringify({
        code: 0,
        data: {
          asset: {
            categoryId: "0",
            createdAt: "2026-06-18T09:00:00Z",
            displayName: "upload-demo.txt",
            extension: "txt",
            external: false,
            id: "asset-upload",
            mimeType: "text/plain",
            originalName: "upload-demo.txt",
            sizeBytes: fileBuffer.length,
            source: "resumable",
            storageKey: "media/upload-demo.txt",
            updatedAt: "2026-06-18T09:00:00Z",
            uploadedBy: "1",
            uploadedByUsername: "owner",
            url: "",
          },
          sessionId: "900",
          storageStatus: "persisted",
        },
      }),
    });
  });

  await page.route("**/api/v1/system/media/assets/resumable/abort", async (route) => {
    captureProtectedRequest(route.request());
    const body = route.request().postDataJSON() as Record<string, unknown>;
    abortBodies.push(body);
    expect(body).toEqual({
      fileHash,
      sessionId: "901",
    });

    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          aborted: true,
          sessionId: "901",
          storageStatus: "persisted",
        },
      }),
    });
  });

  await page.goto("/admin/media/resumable");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.mediaResumable.title }),
  ).toBeVisible();
  await expect(page.getByText(zhCN.admin.mediaResumable.upload.emptyFileTitle)).toBeVisible();
  await page.setInputFiles('input[type="file"]', {
    buffer: fileBuffer,
    mimeType: "text/plain",
    name: "upload-demo.txt",
  });
  await expect(page.getByText(zhCN.admin.mediaResumable.messages.readyTitle)).toBeVisible();
  await page.getByRole("button", { name: zhCN.admin.mediaResumable.actions.upload }).click();
  await expect(
    page.getByText(zhCN.admin.mediaResumable.messages.uploadCompletedTitle),
  ).toBeVisible();
  await expect(page.getByText("upload-demo.txt", { exact: true }).first()).toBeVisible();

  await page.getByRole("button", { name: zhCN.admin.mediaResumable.actions.reset }).click();
  await page.setInputFiles('input[type="file"]', {
    buffer: fileBuffer,
    mimeType: "text/plain",
    name: "upload-demo.txt",
  });
  await expect(page.getByText(zhCN.admin.mediaResumable.messages.readyTitle)).toBeVisible();
  await page.getByRole("button", { name: zhCN.admin.mediaResumable.actions.abort }).click();
  await expect(
    page.getByText(zhCN.admin.mediaResumable.messages.abortCompletedTitle),
  ).toBeVisible();

  expect(checkBodies).toHaveLength(2);
  expect(chunkBodies).toHaveLength(1);
  expect(completeBodies).toHaveLength(1);
  expect(abortBodies).toHaveLength(1);
  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set([
      "/api/v1/system/media/assets",
      "/api/v1/system/media/assets/resumable/abort",
      "/api/v1/system/media/assets/resumable/check",
      "/api/v1/system/media/assets/resumable/chunks",
      "/api/v1/system/media/assets/resumable/complete",
      "/api/v1/system/media/categories",
    ]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      method: request.method,
      path: request.path,
    })),
  );

  const browserStorage = await page.evaluate(() =>
    JSON.stringify({
      local: { ...localStorage },
      session: { ...sessionStorage },
    }),
  );
  expect(browserStorage).not.toContain(fileHash);
  expect(browserStorage).not.toContain(fileBuffer.toString());
});

test("admin server info route is no longer a standalone admin entry", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);

  await page.goto("/admin/server-info");

  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.notFound.title }),
  ).toBeVisible();
});

test("admin traffic hijack route renders probe targets, results, events, and SSE status", async ({
  page,
}) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const consoleErrors: string[] = [];
  const protectedRequests: Array<{
    authorization: string | null;
    locale: string | null;
    method: string;
    path: string;
    query: Record<string, string>;
  }> = [];
  const target: SystemTrafficProbeTarget = {
    alertChannels: "event,debug",
    allowPrivateNetwork: false,
    createdAt: "2026-06-18T09:58:00Z",
    emailRecipients: "",
    enabled: true,
    expectedContentKeyword: "Aoi",
    expectedFinalHost: "example.com",
    expectedIpCidrs: "93.184.216.34/32",
    expectedStatusCodes: "200-399",
    expectedTlsFingerprint: "",
    id: "target-1",
    intervalSeconds: 30,
    lastCheckedAt: "2026-06-18T10:00:00Z",
    lastError: "",
    lastReason: "probe healthy",
    lastSeverity: "ok",
    lastStatus: "healthy",
    method: "GET",
    name: "Public edge",
    nextProbeAt: "2026-06-18T10:00:30Z",
    timeoutSeconds: 5,
    updatedAt: "2026-06-18T10:00:00Z",
    url: "https://example.com",
  };
  const result: SystemTrafficProbeResult = {
    connectDurationMs: 8,
    createdAt: "2026-06-18T10:00:00Z",
    dnsDurationMs: 5,
    dnsIps: "93.184.216.34",
    errorMessage: "",
    evidenceJson: "{}",
    finalUrl: "https://example.com/",
    id: "result-1",
    method: "GET",
    reason: "probe healthy",
    severity: "ok",
    stage: "complete",
    status: "healthy",
    statusCode: 200,
    targetId: "target-1",
    targetName: "Public edge",
    tlsDurationMs: 9,
    tlsFingerprintSha256: "ABCD",
    tlsIssuer: "CN=Example CA",
    tlsNotAfter: "2027-01-01T00:00:00Z",
    tlsSubject: "CN=example.com",
    totalDurationMs: 48,
    ttfbMs: 20,
    url: "https://example.com",
  };
  const hijackEvent: SystemTrafficHijackEvent = {
    createdAt: "2026-06-18T09:59:00Z",
    evidenceHash: "evidence-1",
    evidenceJson: '{"statusCode":500}',
    firstSeenAt: "2026-06-18T09:59:00Z",
    id: "event-1",
    lastSeenAt: "2026-06-18T10:00:00Z",
    notificationStatus: "event,debug",
    occurrences: 2,
    reason: "unexpected status code",
    resolvedAt: null,
    severity: "high",
    state: "open",
    targetId: "target-1",
    targetName: "Public edge",
    updatedAt: "2026-06-18T10:00:00Z",
  };
  const overview: SystemTrafficHijackOverview = {
    criticalTargets: 0,
    enabledTargets: 1,
    healthyTargets: 0,
    recentEvents: [hijackEvent],
    recentResults: [result],
    severityCounts: { high: 1 },
    storageStatus: "available",
    targets: [target],
    totalTargets: 1,
    warningTargets: 1,
  };

  page.on("console", (message) => {
    if (message.type() === "error") {
      consoleErrors.push(message.text());
    }
  });

  const recordRequest = (request: Request) => {
    const url = new URL(request.url());
    protectedRequests.push({
      authorization: requestAuth(request),
      locale: request.headers()["x-locale"] ?? null,
      method: request.method(),
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
  };

  await page.route("**/api/v1/system/traffic-hijack/overview", async (route) => {
    recordRequest(route.request());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: overview }),
    });
  });
  await page.route("**/api/v1/system/traffic-hijack/targets", async (route) => {
    recordRequest(route.request());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: [target] }),
    });
  });
  await page.route("**/api/v1/system/traffic-hijack/results**", async (route) => {
    recordRequest(route.request());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: { items: [result], limit: 80, storageStatus: "available" },
      }),
    });
  });
  await page.route("**/api/v1/system/traffic-hijack/events**", async (route) => {
    recordRequest(route.request());
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: [hijackEvent],
          page: 1,
          pageSize: 20,
          storageStatus: "available",
          total: 1,
        },
      }),
    });
  });
  await page.route("**/api/v1/system/traffic-hijack/stream", async (route) => {
    recordRequest(route.request());
    await route.fulfill({
      body: "event: ready\ndata: {}\n\n",
      contentType: "text/event-stream",
    });
  });

  await page.goto("/admin/traffic-hijack");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.trafficHijack.title }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.trafficHijack.form.title }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.trafficHijack.targets.title }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.trafficHijack.results.title }),
  ).toBeVisible();
  await expect(
    page.getByRole("img", { name: zhCN.admin.trafficHijack.results.chartAria }),
  ).toBeVisible();
  await expect(
    page.locator(".aoi-traffic-target-cell").filter({ hasText: "Public edge" }).first(),
  ).toBeVisible();
  await expect(page.getByText("unexpected status code").first()).toBeVisible();
  await expect
    .poll(() => protectedRequests.some((request) => request.path.endsWith("/stream")))
    .toBe(true);
  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set([
      "/api/v1/system/traffic-hijack/events",
      "/api/v1/system/traffic-hijack/overview",
      "/api/v1/system/traffic-hijack/results",
      "/api/v1/system/traffic-hijack/stream",
      "/api/v1/system/traffic-hijack/targets",
    ]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      method: request.method,
      path: request.path,
      query: request.query,
    })),
  );
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1))
    .toBe(true);
  expect(consoleErrors).toEqual([]);
});

test("admin versions route covers backend version package workflows", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    body?: unknown;
    locale: string | null;
    method: string;
    path: string;
    query: Record<string, string>;
  }> = [];

  const versionPackage: SystemVersionPackage = {
    apis: [
      {
        code: "system",
        count: 1,
        label: "System",
        items: [
          {
            access: "permission",
            code: "GET /api/v1/system/versions",
            description: "List versions",
            group: "system",
            method: "GET",
            order: 1,
            path: "/api/v1/system/versions",
            permission: "version:read",
            permissionRegistered: true,
            synced: true,
            syncedAt: "2026-06-18T08:00:00Z",
          },
        ],
      },
    ],
    dictionaries: [
      {
        code: "system.status",
        createdAt: "2026-06-18T08:00:00Z",
        description: "System status",
        id: "dict-1",
        items: [
          {
            createdAt: "2026-06-18T08:00:00Z",
            dictionaryId: "dict-1",
            extra: "",
            id: "dict-item-1",
            label: "Active",
            sort: 1,
            status: "active",
            updatedAt: "2026-06-18T08:00:00Z",
            value: "active",
          },
        ],
        name: "System status",
        status: "active",
        updatedAt: "2026-06-18T08:00:00Z",
      },
    ],
    exportedAt: "2026-06-18T09:00:00Z",
    menus: [
      {
        code: "system",
        description: "System",
        items: [
          {
            code: "versions",
            description: "Versions",
            icon: "package-check",
            label: "Versions",
            mobile: false,
            order: 1,
            path: "/versions",
            permission: "version:read",
          },
        ],
        label: "System",
        order: 1,
      },
    ],
    version: {
      code: "v2026.06",
      createdAt: "2026-06-18T09:00:00Z",
      description: "Initial export",
      name: "June Release",
    },
  };

  const versionItems = [
    {
      apiCount: 2,
      createdAt: "2026-06-18T09:00:00Z",
      createdBy: "1",
      createdByUsername: "owner",
      description: "Initial export",
      dictionaryCount: 1,
      id: "100",
      menuCount: 1,
      source: "export",
      updatedAt: "2026-06-18T09:00:00Z",
      versionCode: "v2026.06",
      versionName: "June Release",
    },
    {
      apiCount: 1,
      createdAt: "2026-06-17T09:00:00Z",
      createdBy: "2",
      createdByUsername: "operator",
      description: "",
      dictionaryCount: 3,
      id: "99",
      menuCount: 2,
      source: "import",
      updatedAt: "2026-06-17T09:00:00Z",
      versionCode: "v2026.05",
      versionName: "Seed Import",
    },
  ];

  await page.route("**/api/v1/system/versions**", async (route) => {
    const request = route.request();
    const url = new URL(route.request().url());
    const method = request.method();
    protectedRequests.push({
      authorization: requestAuth(request),
      body: request.postData() ? request.postDataJSON() : undefined,
      locale: request.headers()["x-locale"] ?? null,
      method,
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });

    if (url.pathname === "/api/v1/system/versions/sources" && method === "GET") {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          code: 0,
          data: {
            apiCount: 1,
            apis: versionPackage.apis,
            dictionaries: versionPackage.dictionaries,
            dictionaryCount: 1,
            menuCount: 1,
            menus: versionPackage.menus,
            storageStatus: "persisted",
          },
        }),
      });
      return;
    }

    if (url.pathname === "/api/v1/system/versions/export" && method === "POST") {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          code: 0,
          data: {
            item: versionItems[0],
            package: versionPackage,
            storageStatus: "persisted",
          },
        }),
      });
      return;
    }

    if (url.pathname === "/api/v1/system/versions/import" && method === "POST") {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          code: 0,
          data: {
            apisSkipped: 0,
            dictionariesCreated: 1,
            dictionariesSkipped: 0,
            dictionaryItemsCreated: 1,
            importedAt: "2026-06-18T10:00:00Z",
            item: versionItems[1],
            menusSkipped: 0,
            storageStatus: "persisted",
          },
        }),
      });
      return;
    }

    if (url.pathname === "/api/v1/system/versions/100/download" && method === "GET") {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: versionPackage }),
      });
      return;
    }

    if (url.pathname === "/api/v1/system/versions/100" && method === "GET") {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          code: 0,
          data: {
            item: versionItems[0],
            package: versionPackage,
            storageStatus: "persisted",
          },
        }),
      });
      return;
    }

    if (url.pathname === "/api/v1/system/versions/100" && method === "DELETE") {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: { deleted: true } }),
      });
      return;
    }

    if (url.pathname === "/api/v1/system/versions" && method === "DELETE") {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: { deleted: true } }),
      });
      return;
    }

    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: versionItems,
          page: 1,
          pageSize: 10,
          storageStatus: "persisted",
          total: 12,
        },
      }),
    });
  });

  await page.goto("/admin/versions");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.versions.title }),
  ).toBeVisible();
  await expect(page.getByRole("row", { name: /June Release/ })).toBeVisible();
  await expect(page.getByRole("row", { name: /Seed Import/ })).toBeVisible();
  await expect(page.getByRole("cell", { name: "owner" })).toBeVisible();
  await page.getByLabel(zhCN.admin.versions.filters.versionName).fill("June");
  await page.getByRole("button", { name: zhCN.admin.versions.actions.search }).click();
  await expect
    .poll(() => protectedRequests.some((request) => request.query.versionName === "June"))
    .toBeTruthy();

  await page.getByRole("button", { name: zhCN.admin.versions.actions.createRelease }).click();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.versions.export.title }),
  ).toBeVisible();
  const exportForm = page.locator("form.aoi-version-export-form");
  await exportForm.getByLabel(zhCN.admin.versions.export.versionName).fill("QA Release");
  await exportForm.getByLabel(zhCN.admin.versions.export.versionCode).fill("v2026.qa");
  await page.getByRole("button", { name: zhCN.admin.versions.actions.selectAllSources }).click();
  await exportForm.getByRole("button", { name: zhCN.admin.versions.actions.createRelease }).click();
  await expect(page.getByText(zhCN.admin.versions.messages.exportedTitle)).toBeVisible();

  await page.getByRole("button", { name: zhCN.admin.versions.actions.importVersion }).click();
  await page.getByLabel(zhCN.admin.versions.import.json).fill(JSON.stringify(versionPackage));
  await page
    .locator("form.aoi-version-import-form")
    .getByRole("button", { name: zhCN.admin.versions.actions.submitImport })
    .click();
  await expect(page.getByText(zhCN.admin.versions.messages.importedTitle)).toBeVisible();

  const juneRow = page.getByRole("row", { name: /June Release/ });
  await page.locator(".aoi-version-table .aoi-data-table-wrap").evaluate((element) => {
    const scroller = element as HTMLElement;
    scroller.scrollLeft = scroller.scrollWidth;
  });
  await juneRow
    .getByRole("button", { exact: true, name: zhCN.admin.versions.actions.view })
    .click();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.versions.detail.title }),
  ).toBeVisible();
  await expect
    .poll(() =>
      protectedRequests.some(
        (request) => request.method === "GET" && request.path === "/api/v1/system/versions/100",
      ),
    )
    .toBeTruthy();

  await juneRow
    .getByRole("button", { exact: true, name: zhCN.admin.versions.actions.download })
    .click();
  await expect(page.getByText(zhCN.admin.versions.messages.downloadedTitle)).toBeVisible();

  await page
    .getByLabel(
      interpolate(zhCN.admin.versions.a11y.selectVersion, {
        name: "Seed Import",
      }),
    )
    .check();
  await page
    .getByRole("button", { exact: true, name: zhCN.admin.versions.actions.deleteSelected })
    .click();
  await page
    .getByRole("button", { exact: true, name: zhCN.admin.versions.actions.confirmDelete })
    .click();
  await expect(page.getByText(zhCN.admin.versions.messages.deletedTitle)).toBeVisible();

  await juneRow
    .getByRole("button", { exact: true, name: zhCN.admin.versions.actions.delete })
    .click();
  await page
    .getByRole("button", { exact: true, name: zhCN.admin.versions.actions.confirmDelete })
    .click();
  await expect(page.getByText(zhCN.admin.versions.messages.deletedTitle)).toBeVisible();

  const exportRequest = protectedRequests.find(
    (request) => request.method === "POST" && request.path === "/api/v1/system/versions/export",
  );
  const importRequest = protectedRequests.find(
    (request) => request.method === "POST" && request.path === "/api/v1/system/versions/import",
  );
  const downloadRequest = protectedRequests.find(
    (request) =>
      request.method === "GET" && request.path === "/api/v1/system/versions/100/download",
  );
  const bulkDeleteRequest = protectedRequests.find(
    (request) => request.method === "DELETE" && request.path === "/api/v1/system/versions",
  );
  const singleDeleteRequest = protectedRequests.find(
    (request) => request.method === "DELETE" && request.path === "/api/v1/system/versions/100",
  );

  expect(exportRequest?.body).toMatchObject({
    apiCodes: ["GET /api/v1/system/versions"],
    dictionaryCodes: ["system.status"],
    menuCodes: ["system:versions"],
    versionCode: "v2026.qa",
    versionName: "QA Release",
  });
  expect(importRequest?.body).toEqual({ versionData: JSON.stringify(versionPackage) });
  expect(downloadRequest).toBeTruthy();
  expect(bulkDeleteRequest?.body).toEqual({ ids: ["99"] });
  expect(singleDeleteRequest).toBeTruthy();

  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set([
      "/api/v1/system/versions",
      "/api/v1/system/versions/100",
      "/api/v1/system/versions/100/download",
      "/api/v1/system/versions/export",
      "/api/v1/system/versions/import",
      "/api/v1/system/versions/sources",
    ]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      body: request.body,
      locale: "zh-CN",
      method: request.method,
      path: request.path,
      query: request.query,
    })),
  );
});

test("admin plugins route renders backend plugin query contract", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    body?: unknown;
    locale: string | null;
    method: string;
    path: string;
    query: Record<string, string>;
  }> = [];
  const capabilities: PluginCapabilitiesResponse = {
    capabilities: [
      {
        description: "Aggregates request metrics",
        input_schema: { type: "object" },
        name: "analytics.aggregate",
        output_schema: { type: "object" },
        permissions: ["operation:read"],
        scope: "system",
        secret_policy: "none",
        version: "1.0.0",
      },
    ],
  };
  const plugins: PluginSnapshot[] = [
    {
      capabilities: capabilities.capabilities,
      created_at: "2026-06-19T08:00:00Z",
      endpoint: "https://plugins.example.test/analytics",
      hooks: ["operation.created"],
      instance_id: "worker-1",
      last_heartbeat_at: "2026-06-19T08:15:00Z",
      lease_expires_at: "2026-06-19T08:20:00Z",
      lease_ttl_seconds: 300,
      metadata: { region: "cn-east" },
      name: "Analytics worker",
      owner_host: "host-a",
      permissions: ["operation:read"],
      plugin_id: "analytics",
      protocol: "aoi-plugin-json",
      registered_at: "2026-06-19T08:00:00Z",
      runtime_status: "ready",
      schema_version: "2026-06",
      status: "online",
      transport: "http",
      updated_at: "2026-06-19T08:15:00Z",
      version: "1.0.0",
    },
    {
      capabilities: [],
      created_at: "2026-06-19T07:00:00Z",
      endpoint: "",
      hooks: [],
      instance_id: "worker-2",
      last_heartbeat_at: "2026-06-19T07:05:00Z",
      name: "Batch worker",
      owner_host: "host-b",
      permissions: [],
      plugin_id: "batch",
      protocol: "aoi-plugin-json",
      registered_at: "2026-06-19T07:00:00Z",
      runtime_status: "stopped",
      status: "offline",
      transport: "websocket",
      updated_at: "2026-06-19T07:05:00Z",
      version: "0.9.0",
    },
  ];
  const health: PluginHealthStatus = {
    instance_id: "worker-1",
    last_heartbeat_at: "2026-06-19T08:15:00Z",
    lease_expires_at: "2026-06-19T08:20:00Z",
    plugin_id: "analytics",
    runtime_status: "ready",
    status: "online",
  };

  await page.route("**/api/v1/plugins**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    protectedRequests.push({
      authorization: requestAuth(request),
      locale: request.headers()["x-locale"] ?? null,
      method: request.method(),
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });

    if (url.pathname === "/api/v1/plugins" && request.method() === "GET") {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: plugins }),
      });
      return;
    }
    if (url.pathname === "/api/v1/plugins/analytics" && request.method() === "GET") {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: plugins[0] }),
      });
      return;
    }
    if (url.pathname === "/api/v1/plugins/analytics/health" && request.method() === "GET") {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: health }),
      });
      return;
    }
    if (url.pathname === "/api/v1/plugins/analytics/capabilities" && request.method() === "GET") {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: capabilities }),
      });
      return;
    }
    await route.fulfill({
      contentType: "application/json",
      status: 404,
      body: JSON.stringify({ code: 404, message: "not found" }),
    });
  });

  await page.goto("/admin/plugins");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.plugins.title }),
  ).toBeVisible();
  const analyticsRow = page.getByRole("row").filter({ hasText: "Analytics worker" });
  await expect(analyticsRow).toBeVisible();
  await expect(
    analyticsRow.getByText(zhCN.admin.plugins.status.online, { exact: true }),
  ).toBeVisible();
  await expect(page.getByText("analytics.aggregate", { exact: true })).toBeVisible();
  await page.getByLabel(zhCN.admin.plugins.filters.keyword).fill("batch");
  await expect(page.getByRole("row").filter({ hasText: "Batch worker" })).toBeVisible();
  await expect(page.getByRole("row").filter({ hasText: "Analytics worker" })).toHaveCount(0);
  await activateButton(page.getByRole("button", { name: zhCN.admin.plugins.actions.reset }), page);
  await activateButton(
    page
      .getByRole("row")
      .filter({ hasText: "Analytics worker" })
      .getByRole("button", { name: zhCN.admin.plugins.actions.selected }),
    page,
  );

  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set([
      "/api/v1/plugins",
      "/api/v1/plugins/analytics",
      "/api/v1/plugins/analytics/capabilities",
      "/api/v1/plugins/analytics/health",
    ]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      body: request.body,
      locale: "zh-CN",
      method: "GET",
      path: request.path,
      query: {},
    })),
  );
});

test("admin dictionaries route manages backend dictionary contracts", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    body?: unknown;
    locale: string | null;
    method: string;
    path: string;
    query: Record<string, string>;
  }> = [];
  let nextDictionaryId = 403;
  let nextItemId = 504;
  let dictionaries: SystemDictionary[] = [
    {
      code: "system.status",
      createdAt: "2026-06-18T08:00:00Z",
      description: "System entity states",
      id: "401",
      items: [
        {
          createdAt: "2026-06-18T08:00:00Z",
          dictionaryId: "401",
          extra: '{"tone":"success"}',
          id: "501",
          label: "Enabled",
          sort: 10,
          status: "active",
          updatedAt: "2026-06-18T09:00:00Z",
          value: "active",
        },
        {
          createdAt: "2026-06-18T08:00:00Z",
          dictionaryId: "401",
          extra: "",
          id: "502",
          label: "Disabled",
          sort: 20,
          status: "disabled",
          updatedAt: "2026-06-18T09:00:00Z",
          value: "disabled",
        },
      ],
      name: "System status",
      status: "active",
      updatedAt: "2026-06-18T09:00:00Z",
    },
    {
      code: "billing.status",
      createdAt: "2026-06-17T08:00:00Z",
      description: "Billing workflow states",
      id: "402",
      items: [
        {
          createdAt: "2026-06-17T08:00:00Z",
          dictionaryId: "402",
          extra: "",
          id: "503",
          label: "Trial",
          sort: 10,
          status: "active",
          updatedAt: "2026-06-17T09:00:00Z",
          value: "trial",
        },
      ],
      name: "Billing status",
      status: "disabled",
      updatedAt: "2026-06-17T09:00:00Z",
    },
  ];

  const routeHandler = async (route: Route) => {
    const request = route.request();
    const url = new URL(route.request().url());
    const method = request.method();
    const rawBody = request.postData();
    const body = rawBody ? (JSON.parse(rawBody) as unknown) : undefined;
    protectedRequests.push({
      authorization: requestAuth(request),
      body,
      locale: request.headers()["x-locale"] ?? null,
      method,
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });

    if (url.pathname === "/api/v1/system/dictionaries" && method === "GET") {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          code: 0,
          data: {
            items: dictionaries,
            storageStatus: "persisted",
            total: dictionaries.length,
          },
        }),
      });
      return;
    }

    if (url.pathname === "/api/v1/system/dictionaries" && method === "POST") {
      const input = body as SystemDictionaryInputBody;
      const id = String(nextDictionaryId++);
      const created: SystemDictionary = {
        code: input.code,
        createdAt: "2026-06-19T08:00:00Z",
        description: input.description,
        id,
        items: [],
        name: input.name,
        status: input.status,
        updatedAt: "2026-06-19T08:00:00Z",
      };
      dictionaries = [created, ...dictionaries];
      await route.fulfill({
        contentType: "application/json",
        status: 201,
        body: JSON.stringify({ code: 0, data: created }),
      });
      return;
    }

    const dictionaryMatch = url.pathname.match(/^\/api\/v1\/system\/dictionaries\/([^/]+)$/);
    if (dictionaryMatch && method === "PATCH") {
      const dictionaryId = decodeURIComponent(dictionaryMatch[1]);
      const input = body as Partial<SystemDictionaryInputBody>;
      const current = dictionaries.find((dictionary) => String(dictionary.id) === dictionaryId);
      if (!current) {
        await route.fulfill({
          contentType: "application/json",
          status: 404,
          body: JSON.stringify({ code: 404, message: "not found" }),
        });
        return;
      }
      const updated: SystemDictionary = {
        ...current,
        description: input.description ?? current.description,
        name: input.name ?? current.name,
        status: input.status ?? current.status,
        updatedAt: "2026-06-19T08:30:00Z",
      };
      dictionaries = dictionaries.map((dictionary) =>
        String(dictionary.id) === dictionaryId ? updated : dictionary,
      );
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: updated }),
      });
      return;
    }

    if (dictionaryMatch && method === "DELETE") {
      const dictionaryId = decodeURIComponent(dictionaryMatch[1]);
      dictionaries = dictionaries.filter((dictionary) => String(dictionary.id) !== dictionaryId);
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: { deleted: true } }),
      });
      return;
    }

    const itemCollectionMatch = url.pathname.match(
      /^\/api\/v1\/system\/dictionaries\/([^/]+)\/items$/,
    );
    if (itemCollectionMatch && method === "POST") {
      const dictionaryId = decodeURIComponent(itemCollectionMatch[1]);
      const input = body as SystemDictionaryItemInputBody;
      const created: SystemDictionaryItem = {
        createdAt: "2026-06-19T09:00:00Z",
        dictionaryId,
        extra: input.extra,
        id: String(nextItemId++),
        label: input.label,
        sort: input.sort,
        status: input.status,
        updatedAt: "2026-06-19T09:00:00Z",
        value: input.value,
      };
      dictionaries = dictionaries.map((dictionary) =>
        String(dictionary.id) === dictionaryId
          ? { ...dictionary, items: [...dictionary.items, created] }
          : dictionary,
      );
      await route.fulfill({
        contentType: "application/json",
        status: 201,
        body: JSON.stringify({ code: 0, data: created }),
      });
      return;
    }

    const itemMatch = url.pathname.match(/^\/api\/v1\/system\/dictionary-items\/([^/]+)$/);
    if (itemMatch && method === "PATCH") {
      const itemId = decodeURIComponent(itemMatch[1]);
      const input = body as Partial<SystemDictionaryItemInputBody>;
      let updatedItem: SystemDictionaryItem | null = null;
      dictionaries = dictionaries.map((dictionary) => ({
        ...dictionary,
        items: dictionary.items.map((item) => {
          if (String(item.id) !== itemId) {
            return item;
          }
          updatedItem = {
            ...item,
            extra: input.extra ?? item.extra,
            label: input.label ?? item.label,
            sort: input.sort ?? item.sort,
            status: input.status ?? item.status,
            updatedAt: "2026-06-19T09:30:00Z",
            value: input.value ?? item.value,
          };
          return updatedItem;
        }),
      }));
      if (!updatedItem) {
        await route.fulfill({
          contentType: "application/json",
          status: 404,
          body: JSON.stringify({ code: 404, message: "not found" }),
        });
        return;
      }
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: updatedItem }),
      });
      return;
    }

    if (itemMatch && method === "DELETE") {
      const itemId = decodeURIComponent(itemMatch[1]);
      dictionaries = dictionaries.map((dictionary) => ({
        ...dictionary,
        items: dictionary.items.filter((item) => String(item.id) !== itemId),
      }));
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: { deleted: true } }),
      });
      return;
    }

    await route.fulfill({
      contentType: "application/json",
      status: 404,
      body: JSON.stringify({ code: 404, message: "not found" }),
    });
  };

  await page.route("**/api/v1/system/dictionaries**", routeHandler);
  await page.route("**/api/v1/system/dictionary-items/**", routeHandler);
  const dictionaryGroup = (name: string) =>
    page.locator(".aoi-dictionary-group").filter({
      has: page.getByRole("heading", {
        level: 3,
        name: new RegExp(`^${escapeRegExp(name)}$`),
      }),
    });
  const filterPanel = page.locator(".aoi-admin-panel").filter({
    has: page.getByRole("heading", {
      level: 2,
      name: zhCN.admin.dictionaries.filters.title,
    }),
  });

  await page.goto("/admin/dictionaries");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.dictionaries.title }),
  ).toBeVisible();
  await expect(page.getByText("System status", { exact: true })).toBeVisible();
  await expect(page.getByText("Billing status", { exact: true })).toBeVisible();
  await expect(page.getByText("Enabled", { exact: true })).toBeVisible();
  await page.getByLabel(zhCN.admin.dictionaries.filters.keyword).fill("billing");
  await expect(page.getByText("Billing status", { exact: true })).toBeVisible();
  await expect(page.getByText("System status", { exact: true })).toHaveCount(0);
  await activateButton(
    filterPanel.getByRole("button", { name: zhCN.admin.dictionaries.actions.reset }),
    page,
  );
  await page.getByLabel(zhCN.admin.dictionaries.filters.status).selectOption("disabled");
  await expect(page.getByText("Billing status", { exact: true })).toBeVisible();
  await expect(page.getByText("System status", { exact: true })).toHaveCount(0);
  await activateButton(
    filterPanel.getByRole("button", { name: zhCN.admin.dictionaries.actions.reset }),
    page,
  );

  await activateButton(
    page.getByRole("button", { name: zhCN.admin.dictionaries.actions.create }),
    page,
  );
  await page.getByLabel(zhCN.admin.dictionaries.form.code).fill("workflow.status");
  await page.getByLabel(zhCN.admin.dictionaries.form.name).fill("Workflow status");
  await page.getByLabel(zhCN.admin.dictionaries.form.status).selectOption("active");
  await page
    .getByLabel(zhCN.admin.dictionaries.form.descriptionField)
    .fill("Workflow runtime states");
  await activateButton(
    page.getByRole("button", { name: zhCN.admin.dictionaries.actions.submitCreate }),
    page,
  );
  await expect(page.getByText(zhCN.admin.dictionaries.messages.createdTitle)).toBeVisible();
  await expect(page.getByText("Workflow status", { exact: true })).toBeVisible();

  const workflowGroup = dictionaryGroup("Workflow status");
  await activateButton(
    workflowGroup.locator("button", { hasText: zhCN.admin.dictionaries.actions.edit }),
    page,
  );
  await page.getByLabel(zhCN.admin.dictionaries.form.name).fill("Workflow states");
  await activateButton(
    page.getByRole("button", { name: zhCN.admin.dictionaries.actions.save }),
    page,
  );
  await expect(page.getByText(zhCN.admin.dictionaries.messages.updatedTitle)).toBeVisible();
  await expect(page.getByText("Workflow states", { exact: true })).toBeVisible();

  const updatedWorkflowGroup = dictionaryGroup("Workflow states");
  await activateButton(
    updatedWorkflowGroup.locator("button", { hasText: zhCN.admin.dictionaries.actions.addItem }),
    page,
  );
  await page.getByLabel(zhCN.admin.dictionaries.form.itemLabel).fill("Queued");
  await page.getByLabel(zhCN.admin.dictionaries.form.itemValue).fill("queued");
  await page.getByLabel(zhCN.admin.dictionaries.form.itemSort).fill("30");
  await page.getByLabel(zhCN.admin.dictionaries.form.itemStatus).selectOption("active");
  await page.getByLabel(zhCN.admin.dictionaries.form.itemExtra).fill('{"tone":"info"}');
  await activateButton(
    page.getByRole("button", { name: zhCN.admin.dictionaries.actions.submitCreateItem }),
    page,
  );
  await expect(page.getByText(zhCN.admin.dictionaries.messages.itemCreatedTitle)).toBeVisible();
  await expect(page.getByText("Queued", { exact: true })).toBeVisible();

  const queuedRow = updatedWorkflowGroup.getByRole("row").filter({ hasText: "Queued" });
  await activateButton(
    queuedRow.locator("button", { hasText: zhCN.admin.dictionaries.actions.edit }),
    page,
  );
  await page.getByLabel(zhCN.admin.dictionaries.form.itemLabel).fill("Queued task");
  await page.getByLabel(zhCN.admin.dictionaries.form.itemSort).fill("40");
  await activateButton(
    page.getByRole("button", { name: zhCN.admin.dictionaries.actions.saveItem }),
    page,
  );
  await expect(page.getByText(zhCN.admin.dictionaries.messages.itemUpdatedTitle)).toBeVisible();
  await expect(page.getByText("Queued task", { exact: true })).toBeVisible();

  const queuedTaskRow = updatedWorkflowGroup.getByRole("row").filter({ hasText: "Queued task" });
  await activateButton(
    queuedTaskRow.locator("button", { hasText: zhCN.admin.dictionaries.actions.delete }),
    page,
  );
  await expect(page.getByText(zhCN.admin.dictionaries.delete.itemTitle)).toBeVisible();
  await activateButton(
    page.getByRole("button", { name: zhCN.admin.dictionaries.actions.confirmDelete }),
    page,
  );
  await expect(page.getByText(zhCN.admin.dictionaries.messages.itemDeletedTitle)).toBeVisible();
  await expect(page.getByText("Queued task", { exact: true })).toHaveCount(0);

  const workflowDeleteGroup = dictionaryGroup("Workflow states");
  await activateButton(
    workflowDeleteGroup
      .locator("button", { hasText: zhCN.admin.dictionaries.actions.delete })
      .first(),
    page,
  );
  await expect(page.getByText(zhCN.admin.dictionaries.delete.dictionaryTitle)).toBeVisible();
  await activateButton(
    page.getByRole("button", { name: zhCN.admin.dictionaries.actions.confirmDelete }),
    page,
  );
  await expect(page.getByText(zhCN.admin.dictionaries.messages.deletedTitle)).toBeVisible();
  await expect(page.getByText("Workflow states", { exact: true })).toHaveCount(0);

  const createDictionaryRequest = protectedRequests.find(
    (request) => request.method === "POST" && request.path === "/api/v1/system/dictionaries",
  );
  const updateDictionaryRequest = protectedRequests.find(
    (request) => request.method === "PATCH" && request.path === "/api/v1/system/dictionaries/403",
  );
  const createItemRequest = protectedRequests.find(
    (request) =>
      request.method === "POST" && request.path === "/api/v1/system/dictionaries/403/items",
  );
  const updateItemRequest = protectedRequests.find(
    (request) =>
      request.method === "PATCH" && request.path === "/api/v1/system/dictionary-items/504",
  );
  const deleteItemRequest = protectedRequests.find(
    (request) =>
      request.method === "DELETE" && request.path === "/api/v1/system/dictionary-items/504",
  );
  const deleteDictionaryRequest = protectedRequests.find(
    (request) => request.method === "DELETE" && request.path === "/api/v1/system/dictionaries/403",
  );

  expect(createDictionaryRequest?.body).toEqual({
    code: "workflow.status",
    description: "Workflow runtime states",
    name: "Workflow status",
    status: "active",
  });
  expect(updateDictionaryRequest?.body).toEqual({
    description: "Workflow runtime states",
    name: "Workflow states",
    status: "active",
  });
  expect(createItemRequest?.body).toEqual({
    extra: '{"tone":"info"}',
    label: "Queued",
    sort: 30,
    status: "active",
    value: "queued",
  });
  expect(updateItemRequest?.body).toEqual({
    extra: '{"tone":"info"}',
    label: "Queued task",
    sort: 40,
    status: "active",
    value: "queued",
  });
  expect(deleteItemRequest).toBeTruthy();
  expect(deleteDictionaryRequest).toBeTruthy();
  expect(new Set(protectedRequests.map((request) => `${request.method} ${request.path}`))).toEqual(
    new Set([
      "DELETE /api/v1/system/dictionaries/403",
      "DELETE /api/v1/system/dictionary-items/504",
      "GET /api/v1/system/dictionaries",
      "PATCH /api/v1/system/dictionaries/403",
      "PATCH /api/v1/system/dictionary-items/504",
      "POST /api/v1/system/dictionaries",
      "POST /api/v1/system/dictionaries/403/items",
    ]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      body: request.body,
      locale: "zh-CN",
      method: request.method,
      path: request.path,
      query: {},
    })),
  );
});

test("admin operation records route renders backend operation records with filters", async ({
  page,
}) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    body?: unknown;
    locale: string | null;
    method: string;
    path: string;
    query: Record<string, string>;
  }> = [];
  let operationRecords: SystemOperationRecord[] = [
    {
      body: '{"page":1}',
      createdAt: "2026-06-18T09:00:00Z",
      errorMessage: "",
      id: "701",
      ipAddress: "127.0.0.1",
      latencyMs: 32,
      method: "GET",
      path: "/api/v1/system/apis",
      response: '{"code":0}',
      status: 200,
      traceId: "trace-ok",
      userAgent: "Playwright",
      userId: "1",
      username: "owner",
    },
    {
      body: '{"statusClass":"5xx"}',
      createdAt: "2026-06-18T09:01:00Z",
      errorMessage: "database timeout",
      id: "702",
      ipAddress: "10.0.0.2",
      latencyMs: 1420,
      method: "POST",
      path: "/api/v1/system/operation-records",
      response: "",
      status: 503,
      traceId: "trace-error",
      userAgent: "Playwright",
      userId: "2",
      username: "operator",
    },
  ];

  await page.route("**/api/v1/system/operation-records**", async (route) => {
    const url = new URL(route.request().url());
    const method = route.request().method();
    const rawBody = route.request().postData();
    const requestBody: unknown = method === "GET" || !rawBody ? undefined : JSON.parse(rawBody);
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      body: requestBody,
      locale: route.request().headers()["x-locale"] ?? null,
      method,
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
    if (method === "DELETE") {
      const ids = new Set(
        ((requestBody as { ids?: Array<number | string> } | undefined)?.ids ?? []).map(String),
      );
      operationRecords = operationRecords.filter((record) => !ids.has(String(record.id)));
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          code: 0,
          data: {
            deleted: true,
          },
        }),
      });
      return;
    }
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: operationRecords,
          page: Number(url.searchParams.get("page") ?? "1"),
          pageSize: Number(url.searchParams.get("pageSize") ?? "10"),
          storageStatus: "persisted",
          total: operationRecords.length,
        },
      }),
    });
  });

  await page.goto("/admin/operation-records");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.operationRecords.title }),
  ).toBeVisible();
  await expect(page.getByText("/api/v1/system/apis", { exact: true })).toBeVisible();
  await expect(page.getByText("trace-error", { exact: true })).toBeVisible();
  await expect(page.getByText("database timeout", { exact: true })).toBeVisible();
  await page
    .getByLabel(zhCN.admin.operationRecords.selection.rowAria.replace("{{id}}", "702"))
    .check();
  await page
    .getByRole("button", { name: zhCN.admin.operationRecords.actions.deleteSelected })
    .click();
  await expect(
    page.getByRole("heading", { level: 2, name: zhCN.admin.operationRecords.delete.bulkTitle }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: zhCN.admin.operationRecords.actions.confirmDelete })
    .click();
  await expect(
    page.getByText(zhCN.admin.operationRecords.messages.deletedSelectedTitle),
  ).toBeVisible();
  await expect(page.getByText("trace-error", { exact: true })).toHaveCount(0);
  await page.getByLabel(zhCN.admin.operationRecords.filters.method).selectOption("POST");
  await page.getByLabel(zhCN.admin.operationRecords.filters.path).fill("operation-records");
  await page.getByLabel(zhCN.admin.operationRecords.filters.statusClass).selectOption("5xx");
  await page.getByRole("button", { name: zhCN.admin.operationRecords.actions.search }).click();
  await expect
    .poll(() =>
      protectedRequests.some(
        (request) =>
          request.query.method === "POST" &&
          request.query.path === "operation-records" &&
          request.query.statusClass === "5xx",
      ),
    )
    .toBeTruthy();
  await page.getByLabel(zhCN.admin.operationRecords.filters.status).fill("503");
  await page.getByRole("button", { name: zhCN.admin.operationRecords.actions.search }).click();
  await expect
    .poll(() =>
      protectedRequests.some(
        (request) => request.query.status === "503" && !("statusClass" in request.query),
      ),
    )
    .toBeTruthy();
  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set(["/api/v1/system/operation-records"]),
  );
  const deleteRequest = protectedRequests.find((request) => request.method === "DELETE");
  expect(deleteRequest?.body).toEqual({ ids: ["702"] });
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      body: request.body,
      locale: "zh-CN",
      method: request.method,
      path: request.path,
      query: request.query,
    })),
  );
});

test("admin error logs route renders backend error records with status filters", async ({
  page,
}) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    locale: string | null;
    path: string;
    query: Record<string, string>;
  }> = [];

  await page.route("**/api/v1/system/operation-records**", async (route) => {
    const url = new URL(route.request().url());
    protectedRequests.push({
      authorization: requestAuth(route.request()),
      locale: route.request().headers()["x-locale"] ?? null,
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: [
            {
              body: '{"statusClass":"5xx"}',
              createdAt: "2026-06-18T09:01:00Z",
              errorMessage: "database timeout",
              id: "752",
              ipAddress: "10.0.0.2",
              latencyMs: 1420,
              method: "POST",
              path: "/api/v1/system/operation-records",
              response: "",
              status: 503,
              traceId: "trace-error",
              userAgent: "Playwright",
              userId: "2",
              username: "operator",
            },
            {
              body: "",
              createdAt: "2026-06-18T09:02:00Z",
              errorMessage: "not found",
              id: "753",
              ipAddress: "127.0.0.1",
              latencyMs: 64,
              method: "GET",
              path: "/api/v1/system/missing",
              response: "",
              status: 404,
              traceId: "trace-client-error",
              userAgent: "Playwright",
              userId: "",
              username: "",
            },
          ],
          page: Number(url.searchParams.get("page") ?? "1"),
          pageSize: Number(url.searchParams.get("pageSize") ?? "10"),
          storageStatus: "persisted",
          total: 2,
        },
      }),
    });
  });

  await page.goto("/admin/error-logs");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.errorLogs.title }),
  ).toBeVisible();
  await expect(page.getByText("database timeout", { exact: true })).toBeVisible();
  await expect(page.getByText("trace-client-error", { exact: true })).toBeVisible();
  await expect
    .poll(() => protectedRequests.some((request) => request.query.statusClass === "5xx"))
    .toBeTruthy();

  await page.getByLabel(zhCN.admin.errorLogs.filters.statusClass).selectOption("4xx");
  await page.getByLabel(zhCN.admin.errorLogs.filters.method).selectOption("GET");
  await page.getByLabel(zhCN.admin.errorLogs.filters.path).fill("missing");
  await page.getByRole("button", { name: zhCN.admin.errorLogs.actions.search }).click();
  await expect
    .poll(() =>
      protectedRequests.some(
        (request) =>
          request.query.method === "GET" &&
          request.query.path === "missing" &&
          request.query.statusClass === "4xx",
      ),
    )
    .toBeTruthy();

  await page.getByLabel(zhCN.admin.errorLogs.filters.status).fill("503");
  await page.getByRole("button", { name: zhCN.admin.errorLogs.actions.search }).click();
  await expect
    .poll(() =>
      protectedRequests.some(
        (request) => request.query.status === "503" && !("statusClass" in request.query),
      ),
    )
    .toBeTruthy();
  expect(new Set(protectedRequests.map((request) => request.path))).toEqual(
    new Set(["/api/v1/system/operation-records"]),
  );
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      locale: "zh-CN",
      path: request.path,
      query: request.query,
    })),
  );
});

test("admin parameters route manages backend parameter contracts", async ({ page }) => {
  const accessToken = accessTokenWithOrg("1");
  await setAuthenticatedSession(page, accessToken);
  const protectedRequests: Array<{
    authorization: string | null;
    body?: unknown;
    locale: string | null;
    method: string;
    path: string;
    query: Record<string, string>;
  }> = [];
  let parameters: SystemParameter[] = [
    {
      createdAt: "2026-06-18T09:00:00Z",
      description: "Public display title",
      id: "301",
      key: "site.title",
      name: "Site title",
      updatedAt: "2026-06-18T09:30:00Z",
      value: "Aoi Admin",
    },
    {
      createdAt: "2026-06-17T09:00:00Z",
      description: "",
      id: "302",
      key: "feature.flags",
      name: "Feature flags",
      updatedAt: "2026-06-17T09:30:00Z",
      value: '{"enabled":true}',
    },
  ];

  await page.route("**/api/v1/system/parameters**", async (route) => {
    const request = route.request();
    const url = new URL(route.request().url());
    const method = request.method();
    const rawBody = request.postData();
    const body: unknown = rawBody ? (JSON.parse(rawBody) as unknown) : undefined;
    protectedRequests.push({
      authorization: requestAuth(request),
      body,
      locale: request.headers()["x-locale"] ?? null,
      method,
      path: url.pathname,
      query: Object.fromEntries(url.searchParams.entries()),
    });

    if (method === "POST" && url.pathname === "/api/v1/system/parameters") {
      const input = body as SystemParameterInputBody;
      const created: SystemParameter = {
        createdAt: "2026-06-19T08:00:00Z",
        description: input.description,
        id: "303",
        key: input.key,
        name: input.name,
        updatedAt: "2026-06-19T08:00:00Z",
        value: input.value,
      };
      parameters = [created, ...parameters];
      await route.fulfill({
        contentType: "application/json",
        status: 201,
        body: JSON.stringify({ code: 0, data: created }),
      });
      return;
    }

    if (method === "PATCH") {
      const parameterId = url.pathname.split("/").at(-1);
      const input = body as Partial<SystemParameterInputBody>;
      const current = parameters.find((parameter) => String(parameter.id) === parameterId);
      if (!current) {
        await route.fulfill({
          contentType: "application/json",
          status: 404,
          body: JSON.stringify({ code: 404, message: "not found" }),
        });
        return;
      }
      const updated: SystemParameter = {
        ...current,
        description: input.description ?? current.description,
        key: input.key ?? current.key,
        name: input.name ?? current.name,
        updatedAt: "2026-06-19T08:30:00Z",
        value: input.value ?? current.value,
      };
      parameters = parameters.map((parameter) =>
        String(parameter.id) === parameterId ? updated : parameter,
      );
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: updated }),
      });
      return;
    }

    if (method === "DELETE" && url.pathname === "/api/v1/system/parameters") {
      const ids = (body as { ids: string[] }).ids.map(String);
      parameters = parameters.filter((parameter) => !ids.includes(String(parameter.id)));
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: { deleted: true } }),
      });
      return;
    }

    if (method === "DELETE") {
      const parameterId = url.pathname.split("/").at(-1);
      parameters = parameters.filter((parameter) => String(parameter.id) !== parameterId);
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: { deleted: true } }),
      });
      return;
    }

    const filteredParameters = parameters.filter((parameter) => {
      const name = url.searchParams.get("name")?.trim().toLowerCase();
      const key = url.searchParams.get("key")?.trim().toLowerCase();
      return (
        (!name || parameter.name.toLowerCase().includes(name)) &&
        (!key || parameter.key.toLowerCase().includes(key))
      );
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          items: filteredParameters,
          page: 1,
          pageSize: 10,
          storageStatus: "persisted",
          total: filteredParameters.length,
        },
      }),
    });
  });

  await page.goto("/admin/parameters");

  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.admin.parameters.title }),
  ).toBeVisible();
  await expect(page.locator(".aoi-parameter-name").filter({ hasText: "Site title" })).toBeVisible();
  await expect(page.getByText("site.title", { exact: true })).toBeVisible();
  await expect(page.getByText("Aoi Admin", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: zhCN.admin.parameters.actions.create }).click();
  const form = page.locator(".aoi-parameter-form-panel");
  await form.getByLabel(zhCN.admin.parameters.form.name).fill("API endpoint");
  await form.getByLabel(zhCN.admin.parameters.form.key).fill("api.endpoint");
  await form.getByLabel(zhCN.admin.parameters.form.value).fill("https://api.example.test");
  await form.getByLabel(zhCN.admin.parameters.form.descriptionField).fill("Public API endpoint");
  await form.getByRole("button", { name: zhCN.admin.parameters.actions.create }).press("Enter");
  await expect(page.getByText(zhCN.admin.parameters.messages.createdTitle)).toBeVisible();
  await expect(page.getByText("api.endpoint", { exact: true })).toBeVisible();

  await page
    .getByRole("button", {
      name: interpolate(zhCN.admin.parameters.actions.editFor, { name: "API endpoint" }),
    })
    .press("Enter");
  await form.getByLabel(zhCN.admin.parameters.form.value).fill("https://api.example.test/v2");
  await form.getByRole("button", { name: zhCN.admin.parameters.actions.save }).press("Enter");
  await expect(page.getByText(zhCN.admin.parameters.messages.updatedTitle)).toBeVisible();
  await expect(page.getByText("https://api.example.test/v2", { exact: true })).toBeVisible();

  await page
    .getByRole("button", {
      name: interpolate(zhCN.admin.parameters.actions.deleteFor, { name: "API endpoint" }),
    })
    .press("Enter");
  await page
    .getByRole("button", { name: zhCN.admin.parameters.actions.confirmDelete })
    .press("Enter");
  await expect(page.getByText(zhCN.admin.parameters.messages.deletedTitle)).toBeVisible();

  await page
    .getByLabel(interpolate(zhCN.admin.parameters.selection.rowAria, { id: "302" }))
    .press("Space");
  await page
    .getByRole("button", { name: zhCN.admin.parameters.actions.deleteSelected })
    .press("Enter");
  await page
    .getByRole("button", { name: zhCN.admin.parameters.actions.confirmDelete })
    .press("Enter");
  await expect(page.getByText(zhCN.admin.parameters.messages.bulkDeletedTitle)).toBeVisible();

  await page.getByLabel(zhCN.admin.parameters.filters.name).fill("Site");
  await page.getByLabel(zhCN.admin.parameters.filters.key).fill("site.title");
  await page.getByRole("button", { name: zhCN.admin.parameters.actions.search }).press("Enter");
  await expect
    .poll(() =>
      protectedRequests.some(
        (request) => request.query.name === "Site" && request.query.key === "site.title",
      ),
    )
    .toBeTruthy();
  expect(protectedRequests).toEqual(
    protectedRequests.map((request) => ({
      authorization: `Bearer ${accessToken}`,
      body: request.body,
      locale: "zh-CN",
      method: request.method,
      path: request.path,
      query: request.query,
    })),
  );

  expect(new Set(protectedRequests.map((request) => `${request.method} ${request.path}`))).toEqual(
    new Set([
      "GET /api/v1/system/parameters",
      "POST /api/v1/system/parameters",
      "PATCH /api/v1/system/parameters/303",
      "DELETE /api/v1/system/parameters/303",
      "DELETE /api/v1/system/parameters",
    ]),
  );
  expect(
    protectedRequests.find(
      (request) => request.method === "POST" && request.path === "/api/v1/system/parameters",
    )?.body,
  ).toEqual({
    description: "Public API endpoint",
    key: "api.endpoint",
    name: "API endpoint",
    value: "https://api.example.test",
  });
  expect(
    protectedRequests.find((request) => request.method === "PATCH" && request.path.endsWith("/303"))
      ?.body,
  ).toEqual({
    description: "Public API endpoint",
    key: "api.endpoint",
    name: "API endpoint",
    value: "https://api.example.test/v2",
  });
  expect(
    protectedRequests.find(
      (request) => request.method === "DELETE" && request.path === "/api/v1/system/parameters",
    )?.body,
  ).toEqual({ ids: ["302"] });
});

test("setup route is independent from public and platform console shells", async ({ page }) => {
  await preferZhCN(page);
  await page.goto("/setup");
  await expect(page.locator(".aoi-setup-shell")).toBeVisible();
  await expect(page.locator(".aoi-public-header")).toHaveCount(0);
  await expect(page.locator(".aoi-admin-sidebar")).toHaveCount(0);
  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.setup.language.title }),
  ).toBeVisible();
  await expect(page.getByLabel(zhCN.setup.language.field)).toBeVisible();
  await expect(page.getByText(zhCN.setup.security.title)).toBeVisible();
});

test("setup required status redirects first-time public routes into setup language step", async ({
  page,
}) => {
  await preferZhCN(page);
  const statusLocales: string[] = [];

  await page.route("**/api/v1/setup/status", async (route) => {
    statusLocales.push(route.request().headers()["x-locale"] ?? "");
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          currentStep: "database.configure",
          required: true,
          steps: [],
        },
      }),
    });
  });
  await page.route("**/api/v1/setup/schema**", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: { steps: [] } }),
    });
  });

  await page.goto("/about");

  await expect(page).toHaveURL(/\/setup$/);
  await expect(page.locator(".aoi-setup-shell")).toBeVisible();
  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.setup.language.title }),
  ).toBeVisible();
  expect(new Set(statusLocales)).toEqual(new Set(["zh-CN"]));
});

test("setup required status resumes confirmed setup language to backend current step", async ({
  page,
}) => {
  await preferZhCN(page);
  await confirmSetupLanguagePreference(page);

  await page.route("**/api/v1/setup/status", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          currentStep: "storage.configure",
          required: true,
          steps: [
            {
              key: "storage.configure",
              phase: "storage",
              schema: {
                routeSlug: "files",
              },
              status: "pending",
              title: "File storage",
            },
          ],
        },
      }),
    });
  });
  await page.route("**/api/v1/setup/schema**", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          steps: [
            {
              description: "Configure file storage from the backend setup schema.",
              fields: [],
              groups: [],
              key: "storage.configure",
              order: 9,
              phase: "storage",
              required: false,
              routeSlug: "files",
              skippable: true,
              testable: true,
              title: "File storage",
            },
          ],
        },
      }),
    });
  });

  await page.goto("/about");

  await expect(page).toHaveURL(/\/setup\/files$/);
  await expect(page.locator(".aoi-setup-shell")).toBeVisible();
  await expect(page.getByRole("heading", { level: 1, name: "File storage" })).toBeVisible();
  await expect(page.locator(".aoi-public-header")).toHaveCount(0);
  await expect(page.locator(".aoi-admin-sidebar")).toHaveCount(0);
});

test("setup language selection maps following setup requests and default locale to backend locale", async ({
  page,
}) => {
  await preferZhCN(page);
  const statusLocales: string[] = [];
  const schemaLocales: string[] = [];
  let saveLocale = "";
  let saveBody: unknown = null;

  await page.route("**/api/v1/setup/status", async (route) => {
    statusLocales.push(route.request().headers()["x-locale"] ?? "");
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          currentStep: "system.configure",
          required: true,
          steps: [],
        },
      }),
    });
  });
  await page.route("**/api/v1/setup/schema**", async (route) => {
    const locale = route.request().headers()["x-locale"] ?? "";
    schemaLocales.push(locale);
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          steps: [
            {
              description:
                locale === "en-US" ? "Configure the backend default locale." : "Configure locale.",
              fields: [],
              groups: [
                {
                  fields: [
                    {
                      configPath: "i18n.defaultLocale",
                      default: "zh-CN",
                      key: "i18n.defaultLocale",
                      label: locale === "en-US" ? "Default backend locale" : "Default locale",
                      options: [
                        { label: "Simplified Chinese", value: "zh-CN" },
                        { label: "English", value: "en-US" },
                      ],
                      required: true,
                      sensitive: false,
                      type: "select",
                    },
                  ],
                  key: "locale",
                  title: locale === "en-US" ? "Locale" : "Locale",
                },
              ],
              inputFingerprint: "system-default-locale",
              key: "system.configure",
              order: 20,
              phase: "system",
              required: true,
              routeSlug: "system",
              skippable: false,
              testable: false,
              title: locale === "en-US" ? "System settings" : "System settings",
            },
            {
              description: locale === "en-US" ? "Verify initialization." : "Verify initialization.",
              fields: [],
              groups: [],
              key: "verify.finish",
              order: 90,
              phase: "verify",
              required: true,
              routeSlug: "verify",
              skippable: false,
              testable: false,
              title: locale === "en-US" ? "Verify" : "Verify",
            },
          ],
        },
      }),
    });
  });
  await page.route("**/api/v1/setup/configs/system.configure", async (route) => {
    saveLocale = route.request().headers()["x-locale"] ?? "";
    saveBody = route.request().postDataJSON();
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          inputFingerprint: "system-saved",
          inputSummary: "system.configure captured 1 field(s)",
          restartRequired: false,
          stepKey: "system.configure",
        },
      }),
    });
  });

  await page.goto("/setup");
  await expect(
    page.getByRole("heading", { level: 1, name: zhCN.setup.language.title }),
  ).toBeVisible();

  await page.getByLabel(zhCN.setup.language.field).selectOption("en");
  await expect(
    page.getByRole("heading", { level: 1, name: en.setup.language.title }),
  ).toBeVisible();
  await expect.poll(() => schemaLocales).toContain("en-US");

  await page.getByRole("button", { name: en.setup.actions.continue }).click();
  await expect
    .poll(() =>
      page.evaluate(
        (storageKey) => localStorage.getItem(storageKey),
        setupLanguageConfirmedStorageKey,
      ),
    )
    .toBe("true");
  await expect(page).toHaveURL(/\/setup\/system$/);
  await expect(page.getByRole("heading", { level: 1, name: "System settings" })).toBeVisible();
  await expect(page.getByLabel("Default backend locale")).toHaveValue("en-US");

  await page.getByRole("button", { name: en.setup.actions.save }).click();
  await expect(page).toHaveURL(/\/setup\/verify$/);
  await expect.poll(() => statusLocales).toContain("en-US");
  expect(saveLocale).toBe("en-US");
  expect(saveBody).toMatchObject({
    persist: true,
    test: false,
    values: {
      "i18n.defaultLocale": "en-US",
    },
  });
});

test("setup site step saves only backend-exposed site display fields", async ({ page }) => {
  await preferZhCN(page);
  await confirmSetupLanguagePreference(page);
  let saveBody: unknown = null;
  let saveLocale = "";

  await page.route("**/api/v1/setup/status", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          currentStep: "site.configure",
          required: true,
          steps: [
            {
              key: "site.configure",
              phase: "site",
              status: "pending",
              title: "Site information",
            },
          ],
        },
      }),
    });
  });
  await page.route("**/api/v1/setup/schema**", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          steps: [
            {
              dependencies: ["iam.owner"],
              description: "Confirm public site display settings.",
              fields: [],
              groups: [
                {
                  fields: [
                    {
                      configPath: "brand.productName",
                      key: "brand.productName",
                      label: "Product name",
                      required: true,
                      sensitive: false,
                      type: "text",
                      value: "Aoi Admin",
                    },
                    {
                      configPath: "brand.versionName",
                      key: "brand.versionName",
                      label: "Version display name",
                      required: true,
                      sensitive: false,
                      type: "text",
                      value: "Community",
                    },
                    {
                      configPath: "webui.public_base_url",
                      key: "webui.public_base_url",
                      label: "Public base URL",
                      required: false,
                      sensitive: false,
                      type: "text",
                      value: "/admin",
                    },
                    {
                      key: "auth.issuer",
                      label: "IAM issuer",
                      required: true,
                      sensitive: false,
                      type: "text",
                      value: "must-not-save-from-site-step",
                    },
                  ],
                  key: "site",
                  title: "Site display",
                },
              ],
              inputFingerprint: "site-current",
              key: "site.configure",
              order: 75,
              phase: "site",
              required: true,
              routeSlug: "site",
              skippable: false,
              testable: true,
              title: "Site information",
            },
            {
              description: "Verify initialization.",
              fields: [],
              groups: [],
              key: "verify.finish",
              order: 90,
              phase: "verify",
              required: true,
              routeSlug: "verify",
              skippable: false,
              testable: false,
              title: "Verify",
            },
          ],
        },
      }),
    });
  });
  await page.route("**/api/v1/setup/configs/site.configure", async (route) => {
    saveLocale = route.request().headers()["x-locale"] ?? "";
    saveBody = route.request().postDataJSON();
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          inputFingerprint: "site-saved",
          inputSummary: "site.configure captured 3 field(s)",
          restartRequired: false,
          stepKey: "site.configure",
          test: {
            inputFingerprint: "site-saved",
            restartRequired: false,
            status: "succeeded",
            stepKey: "site.configure",
            summary: "site configuration ok",
            testedAt: "2026-06-18T00:03:00Z",
          },
        },
      }),
    });
  });

  await page.goto("/setup/site");

  await expect(page).toHaveURL(/\/setup\/site$/);
  await expect(page.getByRole("heading", { level: 1, name: "Site information" })).toBeVisible();
  await expect(page.getByLabel("Product name")).toHaveValue("Aoi Admin");
  await page.getByLabel("Product name").fill("Aoi Suite");
  await page.getByLabel("Public base URL").fill("https://admin.example.com");
  await page.getByRole("button", { name: zhCN.setup.actions.save }).click();

  await expect(page).toHaveURL(/\/setup\/verify$/);
  expect(saveLocale).toBe("zh-CN");
  expect(saveBody).toMatchObject({
    persist: true,
    test: true,
    values: {
      "brand.productName": "Aoi Suite",
      "brand.versionName": "Community",
      "webui.public_base_url": "https://admin.example.com",
    },
  });
  expect(JSON.stringify(saveBody)).not.toContain("auth.issuer");
});

test("setup dependency status blocks downstream navigation after failed prerequisite", async ({
  page,
}) => {
  await preferZhCN(page);

  await page.route("**/api/v1/setup/status", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          currentStep: "database.configure",
          required: true,
          steps: [
            {
              errorMessage: "Database is not reachable.",
              key: "database.configure",
              phase: "database",
              status: "failed",
              title: "Database",
            },
            {
              dependencies: ["database.configure"],
              key: "cache.configure",
              phase: "cache",
              status: "pending",
              title: "Cache",
            },
            {
              dependencies: ["cache.configure"],
              key: "verify.finish",
              phase: "verify",
              status: "pending",
              title: "Verify",
            },
          ],
        },
      }),
    });
  });
  await page.route("**/api/v1/setup/schema**", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          steps: [
            {
              dependencies: [],
              description: "Configure the database connection.",
              fields: [],
              groups: [],
              key: "database.configure",
              order: 5,
              phase: "database",
              required: true,
              routeSlug: "database",
              skippable: false,
              testable: false,
              title: "Database",
            },
            {
              dependencies: ["database.configure"],
              description: "Configure cache after database succeeds.",
              fields: [],
              groups: [],
              key: "cache.configure",
              order: 10,
              phase: "cache",
              required: true,
              routeSlug: "cache",
              skippable: false,
              testable: false,
              title: "Cache",
            },
            {
              dependencies: ["cache.configure"],
              description: "Verify initialization.",
              fields: [],
              groups: [],
              key: "verify.finish",
              order: 90,
              phase: "verify",
              required: true,
              routeSlug: "verify",
              skippable: false,
              testable: false,
              title: "Verify",
            },
          ],
        },
      }),
    });
  });

  await page.goto("/setup/database");
  await expect(page.getByRole("heading", { level: 1, name: "Database" })).toBeVisible();

  const blockedCacheStep = page.getByRole("button", {
    name: new RegExp(`Cache.*${zhCN.setup.stepStatus.blocked}`),
  });
  await expect(blockedCacheStep).toBeDisabled();

  await page.getByRole("button", { name: zhCN.setup.actions.save }).click();
  await expect(page).toHaveURL(/\/setup\/database$/);
  await expect(page.getByText(zhCN.setup.errors.stepBlocked)).toBeVisible();
});

test("setup wizard renders backend test repair hints and env-managed save warnings", async ({
  page,
}) => {
  await preferZhCN(page);
  let testBody: unknown;
  let saveBody: unknown;

  await page.route("**/api/v1/setup/status", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          currentStep: "database.configure",
          passwordPolicy: {
            minLength: 12,
            requireLower: true,
            requireNumber: true,
            requireSymbol: true,
            requireUpper: true,
          },
          required: true,
          steps: [],
        },
      }),
    });
  });
  await page.route("**/api/v1/setup/schema**", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          steps: [
            {
              description: "Configure the database connection.",
              fields: [],
              groups: [
                {
                  fields: [
                    {
                      configPath: "database.driver",
                      default: "sqlite",
                      key: "database.driver",
                      label: "Database driver",
                      options: [{ label: "SQLite", value: "sqlite" }],
                      required: true,
                      sensitive: false,
                      type: "select",
                    },
                    {
                      configPath: "database.sqlite.path",
                      default: "data/aoi.db",
                      help: "Local SQLite database path.",
                      key: "database.sqlite.path",
                      label: "SQLite path",
                      required: true,
                      sensitive: false,
                      type: "text",
                    },
                  ],
                  key: "sqlite",
                  title: "SQLite",
                },
              ],
              inputFingerprint: "database-saved",
              key: "database.configure",
              order: 5,
              phase: "database",
              required: true,
              routeSlug: "database",
              skippable: false,
              testable: true,
              title: "Database",
            },
            {
              description: "Verify initialization.",
              fields: [],
              groups: [],
              key: "verify.finish",
              order: 90,
              phase: "verify",
              required: true,
              routeSlug: "verify",
              skippable: false,
              testable: false,
              title: "Verify",
            },
          ],
        },
      }),
    });
  });
  await page.route("**/api/v1/setup/configs/database.configure/test", async (route) => {
    testBody = route.request().postDataJSON();
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          error: "SQLite path is not writable.",
          inputFingerprint: "database-tested",
          repairHint: "Choose a writable data directory.",
          restartRequired: false,
          status: "failed",
          stepKey: "database.configure",
          summary: "Database test failed.",
          testedAt: "2026-06-18T00:00:00Z",
        },
      }),
    });
  });
  await page.route("**/api/v1/setup/configs/database.configure", async (route) => {
    saveBody = route.request().postDataJSON();
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          envManagedPathsOverwritten: ["database.sqlite.path"],
          envManagedPersistence: "force_file",
          inputFingerprint: "database-saved-next",
          inputSummary: "database.configure captured 2 field(s)",
          restartRequired: false,
          stepKey: "database.configure",
          test: {
            inputFingerprint: "database-saved-next",
            restartRequired: false,
            status: "succeeded",
            stepKey: "database.configure",
            summary: "Database connection ok.",
            testedAt: "2026-06-18T00:01:00Z",
          },
        },
      }),
    });
  });

  await page.goto("/setup/database");
  await expect(page.getByRole("heading", { level: 1, name: "Database" })).toBeVisible();
  await page.getByRole("button", { name: zhCN.setup.actions.test }).click();
  await expect(page.getByRole("heading", { name: zhCN.setup.test.failedTitle })).toBeVisible();
  const testFeedback = page.getByLabel(zhCN.setup.test.failedTitle);
  await expect(testFeedback.getByText("SQLite path is not writable.")).toBeVisible();
  await expect(testFeedback.getByText(zhCN.setup.test.repairHint, { exact: false })).toBeVisible();
  await expect(testFeedback.getByText("Choose a writable data directory.")).toBeVisible();

  await page.getByLabel("SQLite path").fill("data/changed.db");
  await expect(page.getByText(zhCN.setup.test.stale)).toBeVisible();
  await page.getByRole("button", { name: zhCN.setup.actions.save }).click();
  await expect(page).toHaveURL(/\/setup\/verify$/);
  await expect(page.getByText(/database\.sqlite\.path/)).toBeVisible();

  expect(testBody).toMatchObject({
    values: {
      "database.driver": "sqlite",
      "database.sqlite.path": "data/aoi.db",
    },
  });
  expect(saveBody).toMatchObject({
    persist: true,
    test: true,
    values: {
      "database.driver": "sqlite",
      "database.sqlite.path": "data/changed.db",
    },
  });
});

test("setup owner step validates password confirmation without submitting it", async ({ page }) => {
  await preferZhCN(page);
  let saveBody: unknown = null;
  let runBody: unknown = null;

  await page.route("**/api/v1/setup/status", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          currentStep: "iam.owner",
          passwordPolicy: {
            minLength: 8,
            requireLower: true,
            requireNumber: true,
            requireSymbol: false,
            requireUpper: true,
          },
          required: true,
          steps: [],
        },
      }),
    });
  });
  await page.route("**/api/v1/setup/schema**", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          steps: [
            {
              description: "Create the first organization and owner account.",
              fields: [
                {
                  key: "orgCode",
                  label: "Organization code",
                  required: true,
                  sensitive: false,
                  type: "text",
                },
                {
                  key: "orgName",
                  label: "Organization name",
                  required: true,
                  sensitive: false,
                  type: "text",
                },
                {
                  key: "username",
                  label: "Username",
                  required: true,
                  sensitive: false,
                  type: "text",
                },
                {
                  key: "email",
                  label: "Email",
                  required: true,
                  sensitive: false,
                  type: "email",
                },
                {
                  key: "displayName",
                  label: "Display name",
                  required: false,
                  sensitive: false,
                  type: "text",
                },
                {
                  key: "password",
                  label: "Password",
                  required: true,
                  sensitive: true,
                  type: "password",
                },
              ],
              groups: [],
              key: "iam.owner",
              order: 70,
              phase: "iam",
              required: true,
              routeSlug: "owner",
              skippable: false,
              testable: false,
              title: "Owner account",
            },
            {
              description: "Verify initialization.",
              fields: [],
              groups: [],
              key: "verify.finish",
              order: 90,
              phase: "verify",
              required: true,
              routeSlug: "verify",
              skippable: false,
              testable: false,
              title: "Verify",
            },
          ],
        },
      }),
    });
  });
  await page.route("**/api/v1/setup/configs/iam.owner", async (route) => {
    saveBody = route.request().postDataJSON();
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          inputFingerprint: "owner-saved",
          inputSummary: "Owner fields captured.",
          restartRequired: false,
          stepKey: "iam.owner",
        },
      }),
    });
  });
  await page.route("**/api/v1/setup/runs", async (route) => {
    runBody = route.request().postDataJSON();
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          loginTokensIssued: false,
          report: {
            failed: 0,
            risk: "low",
            skipped: 0,
            successful: 1,
            summary: "Setup run ok.",
          },
          run: {
            id: "run-owner",
            startedAt: "2026-06-18T00:02:00Z",
            status: "succeeded",
          },
        },
      }),
    });
  });

  await page.goto("/setup/owner");
  await expect(page.getByRole("heading", { level: 1, name: "Owner account" })).toBeVisible();

  await page.getByLabel("Organization code").fill("default");
  await page.getByLabel("Organization name").fill("Default Organization");
  await page.getByLabel("Username").fill("owner");
  await page.getByLabel("Email").fill("owner@example.com");
  await page.getByLabel("Password", { exact: true }).fill("AoiPassword123");
  await page.getByLabel(zhCN.setup.owner.passwordConfirm.label).fill("different-password");
  await page.getByRole("button", { name: zhCN.setup.actions.save }).click();

  await expect(page.locator('[id="iam.owner-passwordConfirm-error"]')).toHaveText(
    zhCN.setup.errors.passwordMismatch,
  );
  expect(saveBody).toBeNull();

  await page.getByLabel(zhCN.setup.owner.passwordConfirm.label).fill("AoiPassword123");
  await page.getByRole("button", { name: zhCN.setup.actions.save }).click();

  await expect(page).toHaveURL(/\/setup\/verify$/);
  expect(saveBody).toBeNull();
  expect(JSON.stringify(saveBody)).not.toContain("passwordConfirm");

  await page.getByLabel(zhCN.setup.confirm.title).check();
  await page.getByRole("button", { name: zhCN.setup.actions.run }).click();

  await expect.poll(() => runBody).not.toBeNull();
  expect(runBody).toMatchObject({
    email: "owner@example.com",
    mode: "first_run",
    orgCode: "default",
    orgName: "Default Organization",
    password: "AoiPassword123",
    username: "owner",
  });
  expect(runBody).not.toHaveProperty("values");
  expect(JSON.stringify(runBody)).not.toContain("passwordConfirm");
});

test("completed setup redirects setup routes to login when no session exists", async ({ page }) => {
  await preferZhCN(page);
  await page.route("**/api/v1/setup/status", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        code: 0,
        data: {
          completed: true,
          required: false,
          steps: [],
        },
      }),
    });
  });
  await page.route("**/api/v1/setup/schema**", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ code: 0, data: { steps: [] } }),
    });
  });

  await page.goto("/setup");

  await expect(page).toHaveURL(/\/login$/);
  await expect(page.locator(".aoi-setup-shell")).toHaveCount(0);
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
});

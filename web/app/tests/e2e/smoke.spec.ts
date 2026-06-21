import { expect, Page, Route, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await mockBackend(page, true);
});

test("renders public platform entry from Rust runtime state", async ({ page }) => {
  await page.goto("/");
  const publicEntry = page.locator("section").filter({ has: page.getByRole("heading", { name: "Aoi[葵] platform base" }) });
  await expect(page.getByRole("heading", { name: "Aoi[葵] platform base" })).toBeVisible();
  const runtimePanel = publicEntry.locator(".panel").filter({ has: page.getByRole("heading", { name: "Runtime state" }) });
  await expect(runtimePanel.getByText("Backend online")).toBeVisible();
  await expect(runtimePanel.getByText("Setup state")).toBeVisible();
  await expect(publicEntry.getByText("Current platform surface")).toBeVisible();
  await publicEntry.getByRole("button", { name: "Console" }).click();
  await expect(page).toHaveURL(/\/admin$/);
});

test("renders authenticated console from Rust API contracts", async ({ page }) => {
  await page.goto("/admin");
  await expect(page.getByRole("heading", { name: "Aoi[葵]" })).toBeVisible();
  await expect(page.getByText("Backend online")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Overview" })).toBeVisible();
  await expect(page.getByText("Network received")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Audit summary" })).toBeVisible();
  const auditSummary = page.locator(".panel").filter({ has: page.getByRole("heading", { name: "Audit summary" }) });
  await expect(auditSummary.getByRole("cell", { name: "/api/v1/me/session" })).toBeVisible();
  await page.getByRole("button", { name: "IAM" }).click();
  await expect(page.getByRole("heading", { name: "Permission catalog" })).toBeVisible();
  await expect(page.getByText("org:read")).toBeVisible();
  await page.getByRole("button", { name: "System" }).click();
  await expect(page.getByRole("heading", { name: "API catalog" })).toBeVisible();
  await page.getByRole("button", { name: "Versions Media Probes" }).click();
  await expect(page.getByRole("heading", { name: "Upload media file" })).toBeVisible();

  const uploadForm = page.locator("form").filter({ has: page.getByRole("heading", { name: "Upload media file" }) });
  await uploadForm.getByLabel("Display name").fill("Logo asset");
  await uploadForm.getByLabel("File").setInputFiles({
    name: "logo.txt",
    mimeType: "text/plain",
    buffer: Buffer.from("console media")
  });
  await uploadForm.getByRole("button", { name: "Upload" }).click();
  const mediaItem = page.locator("li").filter({ hasText: "Logo asset" });
  await expect(mediaItem).toBeVisible();
  await expect(page.getByRole("heading", { name: "Storage objects" })).toBeVisible();
  await expect(page.getByRole("cell", { name: "local/uploads/logo.txt" })).toBeVisible();
  page.once("dialog", (dialog) => void dialog.accept());
  await page
    .locator("tr")
    .filter({ hasText: "local/uploads/logo.txt" })
    .getByRole("button", { name: "Delete" })
    .click();
  await expect(page.getByRole("cell", { name: "local/uploads/logo.txt" })).toHaveCount(0);
});

test("manages tenant roles from Rust role APIs", async ({ page }) => {
  await page.goto("/admin");
  await page.getByRole("button", { name: "IAM" }).click();
  await expect(page.getByRole("heading", { name: "Role management" })).toBeVisible();

  const createForm = page.getByRole("form", { name: "Create role" });
  await createForm.getByLabel("Role code").fill("operator");
  await createForm.getByLabel("Role name").fill("Operator");
  await createForm.getByLabel("Bindable permissions").selectOption(["user:read", "role:read", "user:invite"]);
  await createForm.getByRole("button", { name: "Create role" }).click();
  await expect(page.getByRole("cell", { name: "operator", exact: true })).toBeVisible();

  const editForm = page.getByRole("form", { name: "Edit role" });
  await editForm.getByLabel("Role", { exact: true }).selectOption({ label: "operator" });
  await editForm.getByLabel("Role name").fill("Read only operator");
  await editForm.getByLabel("Bindable permissions").selectOption(["user:read"]);
  await editForm.getByRole("button", { name: "Update role" }).click();
  await expect(page.getByRole("cell", { name: "Read only operator" })).toBeVisible();

  page.once("dialog", (dialog) => void dialog.accept());
  await page.getByRole("button", { name: "Delete" }).click();
  await expect(page.getByRole("cell", { name: "operator", exact: true })).toHaveCount(0);
});

test("updates tenant user metadata and role assignment from Rust user APIs", async ({ page }) => {
  await page.goto("/admin");
  await page.getByRole("button", { name: "IAM" }).click();
  await expect(page.getByRole("heading", { name: "User management" })).toBeVisible();

  const editForm = page.getByRole("form", { name: "Edit user" });
  await editForm.getByLabel("Users").selectOption({ label: "member@example.com" });
  await editForm.getByLabel("Display name").fill("Disabled Member");
  await editForm.getByLabel("Status").selectOption("disabled");
  await editForm.getByLabel("Roles").selectOption(["owner"]);
  await editForm.getByRole("button", { name: "Update user" }).click();

  const memberRow = page.getByRole("row", { name: /member@example.com/ });
  await expect(memberRow).toContainText("Disabled Member");
  await expect(memberRow).toContainText("disabled");
  await expect(memberRow).toContainText("owner");
});

test("manages system resources with Rust delete APIs", async ({ page }) => {
  await page.goto("/admin");
  await page.getByRole("button", { name: "System" }).click();

  const configForm = page.locator("form").filter({ has: page.getByRole("heading", { name: "System configs" }) });
  await configForm.getByLabel("Key").fill("site.theme");
  await configForm.getByLabel("Value").fill("{\"mode\":\"quiet\"}");
  await configForm.getByRole("button", { name: "Save" }).click();
  const configItem = page.locator("li").filter({ hasText: "site.theme" });
  await expect(configItem).toBeVisible();
  page.once("dialog", (dialog) => void dialog.accept());
  await configItem.getByRole("button", { name: "Delete" }).click();
  await expect(page.locator("li").filter({ hasText: "site.theme" })).toHaveCount(0);

  const dictionaryForm = page.locator("form").filter({ has: page.getByRole("heading", { name: "Dictionaries" }) });
  await dictionaryForm.getByLabel("Code").fill("region");
  await dictionaryForm.getByLabel("Name").fill("Region");
  await dictionaryForm.getByRole("button", { name: "Save" }).click();
  const dictionaryItem = page.locator("li").filter({ hasText: "region" });
  await expect(dictionaryItem).toBeVisible();
  page.once("dialog", (dialog) => void dialog.accept());
  await dictionaryItem.getByRole("button", { name: "Delete" }).click();
  await expect(page.locator("li").filter({ hasText: "region" })).toHaveCount(0);

  const parameterForm = page.locator("form").filter({ has: page.getByRole("heading", { name: "Parameters" }) });
  await parameterForm.getByLabel("Key").fill("support.email");
  await parameterForm.getByLabel("Name").fill("Support email");
  await parameterForm.getByLabel("Value").fill("support@example.com");
  await parameterForm.getByRole("button", { name: "Save" }).click();
  const parameterItem = page.locator("li").filter({ hasText: "support.email" });
  await expect(parameterItem).toBeVisible();
  page.once("dialog", (dialog) => void dialog.accept());
  await parameterItem.getByRole("button", { name: "Delete" }).click();
  await expect(page.locator("li").filter({ hasText: "support.email" })).toHaveCount(0);

  await page.getByRole("button", { name: "Versions Media Probes" }).click();
  const versionForm = page.locator("form").filter({ has: page.getByRole("heading", { name: "Version packages" }) });
  await versionForm.getByLabel("Version name").fill("Console 0.2");
  await versionForm.getByLabel("Version code").fill("0.2.0");
  await versionForm.getByLabel("Manifest JSON").fill("{\"channel\":\"dev\"}");
  await versionForm.getByRole("button", { name: "Create" }).click();
  let versionRow = page.getByRole("row", { name: /Console 0\.2/ });
  await expect(versionRow).toContainText("draft");
  page.once("dialog", (dialog) => void dialog.accept());
  await versionRow.getByRole("button", { name: "Publish" }).click();
  await expect(page.getByRole("row", { name: /Console 0\.2/ })).toContainText("active");
  await expect(page.getByRole("heading", { name: "Version release events" })).toBeVisible();
  await expect(page.getByRole("row", { name: /publish/ })).toContainText("succeeded");

  await versionForm.getByLabel("Version name").fill("Console 0.3");
  await versionForm.getByLabel("Version code").fill("0.3.0");
  await versionForm.getByLabel("Manifest JSON").fill("{\"channel\":\"dev\"}");
  await versionForm.getByRole("button", { name: "Create" }).click();
  const nextVersionRow = page.getByRole("row", { name: /Console 0\.3/ });
  page.once("dialog", (dialog) => void dialog.accept());
  await nextVersionRow.getByRole("button", { name: "Publish" }).click();
  await expect(page.getByRole("row", { name: /Console 0\.2/ })).toContainText("retired");
  await expect(page.getByRole("row", { name: /Console 0\.3/ })).toContainText("active");

  versionRow = page.getByRole("row", { name: /Console 0\.2/ });
  page.once("dialog", (dialog) => void dialog.accept());
  await versionRow.getByRole("button", { name: "Rollback" }).click();
  await expect(page.getByRole("row", { name: /Console 0\.2/ })).toContainText("active");
  await expect(page.getByRole("row", { name: /rollback/ })).toContainText("succeeded");
  const retiredVersionRow = page.getByRole("row", { name: /Console 0\.3/ });
  await expect(retiredVersionRow).toContainText("retired");
  page.once("dialog", (dialog) => void dialog.accept());
  await retiredVersionRow.getByRole("button", { name: "Delete" }).click();
  await expect(page.getByRole("row", { name: /Console 0\.3/ })).toHaveCount(0);

  const uploadForm = page.locator("form").filter({ has: page.getByRole("heading", { name: "Upload media file" }) });
  await uploadForm.getByLabel("Display name").fill("Logo asset");
  await uploadForm.getByLabel("File").setInputFiles({
    name: "logo.txt",
    mimeType: "text/plain",
    buffer: Buffer.from("console media")
  });
  await uploadForm.getByRole("button", { name: "Upload" }).click();
  const mediaItem = page.locator("li").filter({ hasText: "Logo asset" });
  await expect(mediaItem).toBeVisible();
  page.once("dialog", (dialog) => void dialog.accept());
  await mediaItem.getByRole("button", { name: "Delete" }).click();
  await expect(page.locator("li").filter({ hasText: "Logo asset" })).toHaveCount(0);

  const probeForm = page.locator("form").filter({ has: page.getByRole("heading", { name: "Traffic probes" }) });
  await probeForm.getByLabel("Name").fill("Docs probe");
  await probeForm.getByLabel("URL").fill("https://example.com/");
  await probeForm.getByLabel("Expected status").fill("200");
  await probeForm.getByRole("button", { name: "Create" }).click();
  const probeItem = page.locator("li").filter({ hasText: "Docs probe" });
  await expect(probeItem).toBeVisible();
  await probeItem.getByRole("button", { name: "Run" }).click();
  await expect(page.getByRole("heading", { name: "Probe alerts" })).toBeVisible();
  const alertRow = page.getByRole("row", { name: /status_mismatch/ });
  await expect(alertRow).toContainText("warning");
  await alertRow.getByRole("button", { name: "Acknowledge" }).click();
  await expect(page.getByRole("row", { name: /status_mismatch/ })).toContainText("acknowledged");
  await page.getByRole("row", { name: /status_mismatch/ }).getByRole("button", { name: "Resolve" }).click();
  await expect(page.getByRole("row", { name: /status_mismatch/ })).toContainText("resolved");
  page.once("dialog", (dialog) => void dialog.accept());
  await probeItem.getByRole("button", { name: "Delete" }).click();
  await expect(page.locator("li").filter({ hasText: "Docs probe" })).toHaveCount(0);

  await probeForm.getByLabel("Name").fill("Recovery probe");
  await probeForm.getByLabel("URL").fill("https://example.com/recovery");
  await probeForm.getByLabel("Expected status").fill("200");
  await probeForm.getByRole("button", { name: "Create" }).click();
  const recoveryProbeItem = page.locator("li").filter({ hasText: "Recovery probe" });
  await expect(recoveryProbeItem).toBeVisible();
  await recoveryProbeItem.getByRole("button", { name: "Run" }).click();
  await expect(page.getByRole("row", { name: /status_mismatch/ })).toContainText("open");
  await recoveryProbeItem.getByRole("button", { name: "Run" }).click();
  await expect(page.getByRole("row", { name: /status_mismatch/ })).toContainText("resolved");
});

test("manages IAM security metadata without secret leakage", async ({ page }) => {
  await page.goto("/admin");
  await page.getByRole("button", { name: "IAM" }).click();
  await expect(page.getByRole("heading", { name: "API tokens" })).toBeVisible();

  const tokenRow = page.getByRole("row", { name: /api_token_live/ });
  await expect(tokenRow).toContainText("active");
  page.once("dialog", (dialog) => void dialog.accept());
  await tokenRow.getByRole("button", { name: "Revoke token" }).click();
  await expect(page.getByRole("row", { name: /api_token_live/ })).toContainText("revoked");

  const inviteForm = page.getByRole("form", { name: "Create invitation" });
  await inviteForm.getByLabel("Email").fill("invitee@example.com");
  await inviteForm.getByLabel("Role", { exact: true }).selectOption("owner");
  await inviteForm.getByRole("button", { name: "Create invitation" }).click();
  const invitationRow = page.getByRole("row", { name: /invitee@example.com/ });
  await expect(invitationRow).toContainText("pending");
  page.once("dialog", (dialog) => void dialog.accept());
  await invitationRow.getByRole("button", { name: "Revoke invitation" }).click();
  await expect(page.getByRole("row", { name: /invitee@example.com/ })).toContainText("revoked");

  const mfaRow = page.getByRole("row", { name: /totp/ });
  await expect(mfaRow).toContainText("active");
  await expect(page.getByRole("heading", { name: "MFA recovery codes" })).toBeVisible();
  await expect(page.getByRole("row", { name: /mfa_recovery_code_abcd1234/ })).toContainText("active");

  page.once("dialog", (dialog) => void dialog.accept());
  await page.getByRole("button", { name: "Rotate recovery codes" }).click();
  await expect(page.getByLabel("Recovery codes generated. Save them now.")).toContainText("test-recovery-code-one");
  await expect(page.getByRole("row", { name: /mfa_recovery_code_rotated/ })).toContainText("active");

  page.once("dialog", (dialog) => void dialog.accept());
  await mfaRow.getByRole("button", { name: "Revoke current MFA factor" }).click();
  await expect(page.getByRole("row", { name: /totp/ })).toContainText("revoked");
  await expect(page.getByRole("row", { name: /mfa_recovery_code_rotated/ })).toContainText("revoked");

  const storage = await page.evaluate(() => JSON.stringify(localStorage));
  expect(storage).not.toContain("api_token_");
  expect(storage).not.toContain("mfa");
});

test("submits first-install admin without storing tokens", async ({ page }) => {
  await mockBackend(page, false);
  await page.goto("/setup");
  await expect(page.getByRole("heading", { name: "First Install" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Configuration checks" })).toBeVisible();
  await expect(page.getByText("Development secrets only")).toBeVisible();
  await page.getByRole("button", { name: "Create setup run" }).click();
  await expect(page.getByText("run-1")).toBeVisible();
  await page.getByLabel("Email").fill("owner@example.com");
  await page.getByLabel("Password").fill("change-me-123");
  await page.getByLabel("Display name").fill("Owner");
  await page.getByLabel("Organization code").fill("main");
  await page.getByLabel("Organization name").fill("Main");
  await page.getByRole("button", { name: "Create admin and open console" }).click();
  await expect(page.getByRole("heading", { name: "Overview" })).toBeVisible();
  await page.getByRole("button", { name: "Setup" }).click();
  await page.getByRole("button", { name: "Mark setup complete" }).click();
  await expect(page.getByText("complete · ok · setup marked complete")).toBeVisible();
  const storage = await page.evaluate(() => JSON.stringify(localStorage));
  expect(storage).not.toContain("session_token_");
  expect(storage).not.toContain("refresh_token_");
});

test("handles account recovery flows without persisting pending tokens", async ({ page }) => {
  await page.goto("/account?reset=password_reset_token_browser&verify=email_verify_token_browser&email=owner@example.com");
  await expect(page.getByRole("heading", { name: "Forgot password" })).toBeVisible();
  await expect(page).not.toHaveURL(/password_reset_token_browser|email_verify_token_browser/);

  const registerForm = page.locator("form").filter({ has: page.getByRole("heading", { name: "Register account" }) });
  await registerForm.getByLabel("Email").fill("signup@example.com");
  await registerForm.getByLabel("Password").fill("change-me-123");
  await registerForm.getByLabel("Display name").fill("Signup User");
  await registerForm.getByLabel("Organization code").fill("signup-main");
  await registerForm.getByLabel("Organization name").fill("Signup Main");
  await registerForm.getByRole("button", { name: "Register account" }).click();
  await expect(page.getByText("Registration submitted")).toBeVisible();

  const forgotForm = page.locator("form").filter({ has: page.getByRole("heading", { name: "Forgot password" }) });
  await forgotForm.getByLabel("Email").fill("owner@example.com");
  await forgotForm.getByRole("button", { name: "Request password reset" }).click();
  await expect(page.getByText("The backend accepted the request")).toBeVisible();

  const resetForm = page.locator("form").filter({ has: page.getByRole("heading", { name: "Reset password" }) });
  await resetForm.getByLabel("Reset token").fill("password_reset_token_browser");
  await resetForm.getByLabel("Password").fill("new-change-me-123");
  await resetForm.getByRole("button", { name: "Reset password" }).click();
  await expect(page.getByText("Password reset complete")).toBeVisible();

  const requestEmailForm = page.locator("form").filter({ has: page.getByRole("heading", { name: "Request email verification" }) });
  await requestEmailForm.getByLabel("Email").fill("owner@example.com");
  await requestEmailForm.getByRole("button", { name: "Request email verification" }).click();
  await expect(page.getByText("The backend accepted the request")).toBeVisible();

  const confirmEmailForm = page.locator("form").filter({ has: page.getByRole("heading", { name: "Confirm email verification" }) });
  await confirmEmailForm.getByLabel("Email verification token").fill("email_verify_token_browser");
  await confirmEmailForm.getByRole("button", { name: "Confirm email verification" }).click();
  await expect(page.getByText("Email verification complete")).toBeVisible();

  const storage = await page.evaluate(() => JSON.stringify(localStorage));
  expect(storage).not.toContain("signup");
  expect(storage).not.toContain("password_reset_token_");
  expect(storage).not.toContain("email_verify_token_");
});

test("accepts invitation without persisting raw invitation token", async ({ page }) => {
  await page.goto("/account?invite=invitation_token_browser");
  await expect(page.getByRole("heading", { name: "Accept invitation" })).toBeVisible();
  await expect(page).not.toHaveURL(/invitation_token_browser/);

  const acceptForm = page.locator("form").filter({ has: page.getByRole("heading", { name: "Accept invitation" }) });
  await acceptForm.getByLabel("Invitation token").fill("invitation_token_browser");
  await acceptForm.getByLabel("Display name").fill("Invited User");
  await acceptForm.getByLabel("Password").fill("change-me-123");
  await acceptForm.getByRole("button", { name: "Accept invitation" }).click();
  await expect(page.getByRole("heading", { name: "Overview" })).toBeVisible();

  const storage = await page.evaluate(() => JSON.stringify(localStorage));
  expect(storage).not.toContain("invitation_token_");
  expect(storage).not.toContain("session_token_");
});

async function mockBackend(page: Page, initialized: boolean) {
  let hasInitialAdmin = initialized;
  let setupCompleted = initialized;
  let setupRunCompleted = initialized;
  let apiTokens: Array<{
    id: number;
    org_id: number;
    user_id: number;
    token_prefix: string;
    status: string;
    expires_at: string;
    created_at: string;
    revoked_at: string | null;
  }> = [
    { id: 1, org_id: 1, user_id: 1, token_prefix: "api_token_live", status: "active", expires_at: now(), created_at: now(), revoked_at: null }
  ];
  let invitations = [
    { id: 1, org_id: 1, email: "pending@example.com", role_code: "owner", status: "pending", expires_at: now(), created_at: now() }
  ];
  let mfaFactors = [
    { id: 1, kind: "totp", status: "active", created_at: now(), verified_at: now(), revoked_at: null as string | null }
  ];
  let mfaRecoveryCodes = [
    { id: 1, code_prefix: "mfa_recovery_code_abcd1234", status: "active", created_at: now(), used_at: null as string | null, revoked_at: null as string | null },
    { id: 2, code_prefix: "mfa_recovery_code_used5678", status: "used", created_at: now(), used_at: now(), revoked_at: null as string | null }
  ];
  let nextMfaRecoveryCodeId = 3;
  let users = [
    { id: 1, email: "owner@example.com", display_name: "Owner", status: "active", email_verified_at: now(), role_codes: ["owner"] },
    { id: 2, email: "member@example.com", display_name: "Member", status: "active", email_verified_at: now(), role_codes: ["member"] }
  ];
  let nextInvitationId = 2;
  let roles = [
    { id: 1, org_id: 1, code: "owner", name: "Owner", scope: "tenant", system_builtin: true, permissions: ["api_token:read", "role:read", "role:write", "user:write"] },
    { id: 2, org_id: 1, code: "member", name: "Member", scope: "tenant", system_builtin: true, permissions: ["user:read"] }
  ];
  let nextRoleId = 3;
  let nextMediaAssetId = 1;
  let nextDictionaryId = 1;
  let nextParameterId = 1;
  let nextProbeAlertId = 1;
  let nextProbeId = 1;
  let nextProbeResultId = 1;
  let nextVersionId = 1;
  let nextVersionReleaseId = 1;
  let configs: Array<{ key: string; value: unknown; updated_at: string }> = [];
  let dictionaries: Array<{ id: number; code: string; name: string; created_at: string }> = [];
  let mediaAssets: Array<{
    id: number;
    category: string | null;
    display_name: string;
    storage_key: string;
    mime_type: string;
    size_bytes: number;
    created_at: string;
  }> = [];
  let storageObjects: Array<{
    storage_key: string;
    size_bytes: number;
    updated_at: string | null;
    e_tag: string | null;
  }> = [];
  let parameters: Array<{ id: number; key: string; name: string; value: string; created_at: string; updated_at: string }> = [];
  let probeAlerts: Array<{
    id: number;
    target_id: number;
    result_id: number;
    severity: string;
    status: string;
    reason: string;
    detail: unknown;
    opened_at: string;
    acknowledged_at: string | null;
    resolved_at: string | null;
  }> = [];
  let probeResults: Array<{ id: number; target_id: number; status: string; detail: unknown; probed_at: string }> = [];
  const probeRunCounts = new Map<number, number>();
  let probes: Array<{ id: number; name: string; url: string; expected_status: number; status: string; created_at: string }> = [];
  let versionReleases: Array<{
    id: number;
    package_id: number;
    previous_active_id: number | null;
    action: string;
    status: string;
    reason: string | null;
    created_at: string;
  }> = [];
  let versions: Array<{
    id: number;
    version_name: string;
    version_code: string;
    manifest: unknown;
    status: string;
    published_at: string | null;
    retired_at: string | null;
    created_at: string;
  }> = [];
  function applyVersionAction(versionId: number, action: "publish" | "rollback", reason: string | null) {
    const target = versions.find((item) => item.id === versionId);
    if (!target) return null;
    const timestamp = now();
    const previousActiveId = versions.find((item) => item.status === "active")?.id ?? null;
    versions = versions.map((item) => {
      if (item.id === versionId) return { ...item, status: "active", published_at: timestamp, retired_at: null };
      if (item.status === "active") return { ...item, status: "retired", retired_at: timestamp };
      return item;
    });
    const event = {
      id: nextVersionReleaseId++,
      package_id: versionId,
      previous_active_id: previousActiveId,
      action,
      status: "succeeded",
      reason,
      created_at: timestamp
    };
    versionReleases = [event, ...versionReleases];
    return {
      event_id: event.id,
      previous_active_id: previousActiveId,
      package: versions.find((item) => item.id === versionId)
    };
  }
  const permissions = [
    { id: 1, product_code: "console", scope: "platform", code: "permission:read", name: "List permissions" },
    { id: 2, product_code: "console", scope: "platform", code: "org:read", name: "List organizations" },
    { id: 3, product_code: "console", scope: "tenant", code: "user:read", name: "List users" },
    { id: 4, product_code: "console", scope: "tenant", code: "role:read", name: "List roles" },
    { id: 5, product_code: "console", scope: "tenant", code: "user:invite", name: "Invite users" },
    { id: 6, product_code: "console", scope: "tenant", code: "user:write", name: "Update users" }
  ];
  await page.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname;
    if (path === "/health" || path === "/ready") return json(route, { status: "ok" });
    if (path === "/api/v1/system/public-settings") {
      return json(route, {
        product_name: "Aoi[葵]",
        product_code: "console",
        default_locale: "zh-CN",
        supported_locales: ["zh-CN", "en"],
        auth: {
          self_signup_enabled: true,
          session_token_cookie_name: "console_session",
          refresh_token_cookie_name: "console_refresh",
          product_header: "X-Console-Product-Code",
          client_type_header: "X-Console-Client-Type",
          csrf_enabled: false,
          csrf_cookie_name: "console_csrf",
          csrf_header_name: "X-CSRF-Token"
        }
      });
    }
    if (path === "/api/v1/auth/setup/status") return json(route, { initialized: hasInitialAdmin });
    if (path === "/api/v1/setup/status") return json(route, { completed: setupCompleted, has_initial_admin: hasInitialAdmin, required_steps: [] });
    if (path === "/api/v1/setup/config-checks") {
      return json(route, {
        ready: true,
        checks: [
          { key: "database", title: "Database driver", status: "ok", severity: "info", message: "SQLite runtime configured" },
          { key: "secrets", title: "Secret posture", status: "warning", severity: "warning", message: "Development secrets only" },
          { key: "cookie-csrf", title: "Cookie and CSRF", status: "warning", severity: "warning", message: "CSRF disabled for local test" }
        ]
      });
    }
    if (path === "/api/v1/setup/schema") {
      return json(route, {
        locale: "en",
        steps: [{ key: "admin", title: "Initial admin", fields: [{ key: "email", label: "Email", kind: "text", required: true, sensitive: false }] }]
      });
    }
    if (path === "/api/v1/auth/setup/initial-admin") {
      hasInitialAdmin = true;
      return json(route, session(false));
    }
    if (path === "/api/v1/auth/login" || path === "/api/v1/auth/refresh" || path === "/api/v1/me/session") {
      return json(route, session(mfaFactors.some((factor) => factor.status === "active" && !factor.revoked_at)));
    }
    if (path === "/api/v1/auth/invitations/accept" && route.request().method() === "POST") {
      return json(route, session(false, "invited@example.com", "Invited User"));
    }
    if (path === "/api/v1/auth/register" && route.request().method() === "POST") {
      return json(route, { accepted: true, channel: "notification-outbox" });
    }
    if (path === "/api/v1/auth/password/forgot" && route.request().method() === "POST") {
      return json(route, { accepted: true, channel: "notification-outbox" });
    }
    if (path === "/api/v1/auth/password/reset" && route.request().method() === "POST") {
      return json(route, { reset: true });
    }
    if (path === "/api/v1/auth/email-verifications" && route.request().method() === "POST") {
      return json(route, { accepted: true, channel: "notification-outbox" });
    }
    if (path === "/api/v1/auth/email-verifications/confirm" && route.request().method() === "POST") {
      const body = route.request().postDataJSON() as { token?: string };
      if (body.token !== "email_verify_token_browser") {
        return json(route, { code: "BAD_REQUEST", message: "missing token body" }, 400);
      }
      return json(route, { verified: true });
    }
    if (path === "/api/v1/auth/mfa/factors") return json(route, mfaFactors);
    if (path === "/api/v1/auth/mfa/recovery-codes") {
      if (route.request().method() === "POST") {
        mfaRecoveryCodes = mfaRecoveryCodes.map((code) => code.status === "active" ? { ...code, status: "revoked", revoked_at: now() } : code);
        const items = [
          { id: nextMfaRecoveryCodeId++, code_prefix: "mfa_recovery_code_rotated", status: "active", created_at: now(), used_at: null, revoked_at: null },
          { id: nextMfaRecoveryCodeId++, code_prefix: "mfa_recovery_code_second", status: "active", created_at: now(), used_at: null, revoked_at: null }
        ];
        mfaRecoveryCodes = [...mfaRecoveryCodes, ...items];
        return json(route, { items, recovery_codes: ["test-recovery-code-one", "test-recovery-code-two"] });
      }
      return json(route, mfaRecoveryCodes);
    }
    const mfaFactorMatch = path.match(/^\/api\/v1\/auth\/mfa\/factors\/(\d+)$/);
    if (mfaFactorMatch && route.request().method() === "DELETE") {
      const factorId = Number(mfaFactorMatch[1]);
      mfaFactors = mfaFactors.map((factor) => factor.id === factorId ? { ...factor, status: "revoked", revoked_at: now() } : factor);
      mfaRecoveryCodes = mfaRecoveryCodes.map((code) => code.status === "active" ? { ...code, status: "revoked", revoked_at: now() } : code);
      return json(route, { revoked: true });
    }
    if (path === "/api/v1/auth/logout") return json(route, { logged_out: true });
    if (path === "/api/v1/setup/runs") {
      const run = {
        id: "run-1",
        status: setupRunCompleted ? "completed" : "running",
        reason: "webui",
        created_at: now(),
        updated_at: now()
      };
      return route.request().method() === "POST" ? json(route, run) : json(route, [run]);
    }
    if (path === "/api/v1/setup/runs/run-1/logs") {
      const logs = [{ step_key: "admin", status: "ok", message: "done", created_at: now() }];
      if (setupRunCompleted) logs.push({ step_key: "complete", status: "ok", message: "setup marked complete", created_at: now() });
      return json(route, logs);
    }
    if (path === "/api/v1/setup/complete") {
      const body = route.request().postDataJSON() as { run_id?: string };
      if (body.run_id !== "run-1") return json(route, { code: "BAD_REQUEST", message: "missing setup run" }, 400);
      setupCompleted = true;
      setupRunCompleted = true;
      return json(route, { completed: true });
    }
    if (path === "/api/v1/system/server-status") {
      return json(route, {
        source: "runtime-process",
        collected_at: now(),
        started_at: now(),
        uptime_seconds: 12,
        process_id: 1,
        os: "windows",
        arch: "x64",
        available_parallelism: 8,
        product_code: "console",
        version: "0.1.0",
        database_driver: "sqlite",
        metrics: {
          source: "sysinfo",
          cpu_usage_percent: 12.5,
          process_cpu_usage_percent: 3.4,
          total_memory_bytes: 17179869184,
          used_memory_bytes: 8589934592,
          available_memory_bytes: 8589934592,
          process_memory_bytes: 134217728,
          process_virtual_memory_bytes: 268435456,
          total_swap_bytes: 0,
          used_swap_bytes: 0,
          total_disk_bytes: 512000000000,
          used_disk_bytes: 256000000000,
          available_disk_bytes: 256000000000,
          disk_count: 2,
          network_interface_count: 3,
          network_received_bytes: 1048576,
          network_transmitted_bytes: 2097152,
          system_uptime_seconds: 12345,
          system_boot_time_seconds: 1800000000,
          load_average_one: 0.25,
          load_average_five: 0.5,
          load_average_fifteen: 0.75
        }
      });
    }
    if (path === "/api/v1/system/menus") return json(route, [{ code: "system.apis", title: "API", path: "/admin/apis", permission: "permission:read", scope: "platform", sort_order: 1 }]);
    if (path === "/api/v1/system/apis") return json(route, [{ tag: "IAM", items: [{ id: "iam.permissions.list", method: "GET", path: "/api/v1/iam/permissions", tag: "IAM Permission", summary: "List permissions", access: "permission", permission: "permission:read", scope: "platform", product_code: "console" }] }]);
    if (path === "/api/v1/system/operation-records") return json(route, [{ id: 1, actor_user_id: 1, method: "GET", path: "/api/v1/me/session", status: 200, created_at: now() }]);
    if (path === "/api/v1/system/operation-records/summary") {
      return json(route, {
        generated_at: now(),
        total_count: 1,
        success_count: 1,
        redirect_count: 0,
        client_error_count: 0,
        server_error_count: 0,
        other_count: 0,
        top_limit: 5,
        by_method: [{ key: "GET", count: 1 }],
        by_status_class: [{ key: "2xx", count: 1 }],
        top_paths: [{ path: "/api/v1/me/session", count: 1, error_count: 0, last_seen_at: now() }]
      });
    }
    if (path === "/api/v1/iam/orgs") return json(route, [{ id: 1, code: "main", name: "Main", scope: "tenant", status: "active", created_at: now() }]);
    if (path === "/api/v1/iam/orgs/1/users") return json(route, users);
    const userMatch = path.match(/^\/api\/v1\/iam\/orgs\/1\/users\/(\d+)$/);
    if (userMatch && route.request().method() === "PUT") {
      const userId = Number(userMatch[1]);
      const body = route.request().postDataJSON() as { display_name: string; status: string; role_codes: string[] };
      users = users.map((user) => user.id === userId ? { ...user, display_name: body.display_name, status: body.status, role_codes: body.role_codes } : user);
      return json(route, users.find((user) => user.id === userId));
    }
    if (path === "/api/v1/iam/orgs/1/roles") {
      if (route.request().method() === "POST") {
        const body = route.request().postDataJSON() as { code: string; name: string; permission_codes: string[] };
        const role = { id: nextRoleId++, org_id: 1, code: body.code, name: body.name, scope: "tenant", system_builtin: false, permissions: body.permission_codes };
        roles = [...roles, role];
        return json(route, role);
      }
      return json(route, roles);
    }
    const roleMatch = path.match(/^\/api\/v1\/iam\/orgs\/1\/roles\/(\d+)$/);
    if (roleMatch) {
      const roleId = Number(roleMatch[1]);
      if (route.request().method() === "PUT") {
        const body = route.request().postDataJSON() as { name: string; permission_codes: string[] };
        roles = roles.map((role) => role.id === roleId ? { ...role, name: body.name, permissions: body.permission_codes } : role);
        return json(route, roles.find((role) => role.id === roleId));
      }
      if (route.request().method() === "DELETE") {
        roles = roles.filter((role) => role.id !== roleId || role.system_builtin);
        return json(route, { deleted: true });
      }
    }
    if (path === "/api/v1/iam/permissions") return json(route, permissions);
    if (path === "/api/v1/orgs/1/api-tokens") {
      if (route.request().method() === "POST") {
        const token = { id: 2, org_id: 1, user_id: 1, token_prefix: "api_token_created", status: "active", expires_at: now(), created_at: now(), revoked_at: null };
        apiTokens = [...apiTokens, token];
        return json(route, { item: token, token: "redacted-test-token" });
      }
      return json(route, apiTokens);
    }
    const apiTokenMatch = path.match(/^\/api\/v1\/orgs\/1\/api-tokens\/(\d+)$/);
    if (apiTokenMatch && route.request().method() === "DELETE") {
      const tokenId = Number(apiTokenMatch[1]);
      apiTokens = apiTokens.map((token) => token.id === tokenId ? { ...token, status: "revoked", revoked_at: now() } : token);
      return json(route, { revoked: true });
    }
    if (path === "/api/v1/orgs/1/invitations") return json(route, invitations);
    if (path === "/api/v1/orgs/1/users/invitations" && route.request().method() === "POST") {
      const body = route.request().postDataJSON() as { email: string; role_code: string };
      const invitation = { id: nextInvitationId++, org_id: 1, email: body.email, role_code: body.role_code, status: "pending", expires_at: now(), created_at: now() };
      invitations = [...invitations, invitation];
      return json(route, { item: invitation });
    }
    const invitationMatch = path.match(/^\/api\/v1\/orgs\/1\/invitations\/(\d+)$/);
    if (invitationMatch && route.request().method() === "DELETE") {
      const invitationId = Number(invitationMatch[1]);
      invitations = invitations.map((invitation) => invitation.id === invitationId ? { ...invitation, status: "revoked" } : invitation);
      return json(route, { revoked: true });
    }
    if (path === "/api/v1/system/media-assets/upload" && route.request().method() === "POST") {
      const contentType = route.request().headers()["content-type"] ?? "";
      if (!contentType.includes("multipart/form-data")) {
        return json(route, { code: "INVALID_MULTIPART", message: "multipart required" }, 400);
      }
      const asset = {
        id: nextMediaAssetId++,
        category: "brand",
        display_name: "Logo asset",
        storage_key: "local/uploads/logo.txt",
        mime_type: "text/plain",
        size_bytes: 13,
        created_at: now()
      };
      mediaAssets = [...mediaAssets, asset];
      storageObjects = [
        ...storageObjects,
        {
          storage_key: asset.storage_key,
          size_bytes: asset.size_bytes,
          updated_at: now(),
          e_tag: null
        }
      ];
      return json(route, asset);
    }
    if (path === "/api/v1/system/media-assets") {
      if (route.request().method() === "POST") {
        const body = route.request().postDataJSON() as {
          category?: string;
          display_name: string;
          mime_type: string;
          size_bytes: number;
          storage_key: string;
        };
        const asset = {
          id: nextMediaAssetId++,
          category: body.category || null,
          display_name: body.display_name,
          storage_key: body.storage_key,
          mime_type: body.mime_type,
          size_bytes: body.size_bytes,
          created_at: now()
        };
        mediaAssets = [...mediaAssets, asset];
        return json(route, asset);
      }
      return json(route, mediaAssets);
    }
    const mediaMatch = path.match(/^\/api\/v1\/system\/media-assets\/(\d+)$/);
    if (mediaMatch && route.request().method() === "DELETE") {
      const mediaId = Number(mediaMatch[1]);
      mediaAssets = mediaAssets.filter((item) => item.id !== mediaId);
      return json(route, { deleted: true });
    }
    if (path === "/api/v1/system/storage-objects") {
      if (route.request().method() === "DELETE") {
        const body = route.request().postDataJSON() as { storage_key: string };
        storageObjects = storageObjects.filter((item) => item.storage_key !== body.storage_key);
        return json(route, { deleted: true });
      }
      return json(route, storageObjects);
    }
    if (path === "/api/v1/system/configs") return json(route, configs);
    const configMatch = path.match(/^\/api\/v1\/system\/configs\/([^/]+)$/);
    if (configMatch) {
      const key = decodeURIComponent(configMatch[1]);
      if (route.request().method() === "PUT") {
        const body = route.request().postDataJSON() as { value: unknown };
        const entry = { key, value: body.value, updated_at: now() };
        configs = [...configs.filter((item) => item.key !== key), entry];
        return json(route, entry);
      }
      if (route.request().method() === "DELETE") {
        configs = configs.filter((item) => item.key !== key);
        return json(route, { deleted: true });
      }
    }
    if (path === "/api/v1/system/dictionaries") return json(route, dictionaries);
    const dictionaryMatch = path.match(/^\/api\/v1\/system\/dictionaries\/([^/]+)$/);
    if (dictionaryMatch) {
      const code = decodeURIComponent(dictionaryMatch[1]);
      if (route.request().method() === "PUT") {
        const body = route.request().postDataJSON() as { name: string };
        const entry = { id: nextDictionaryId++, code, name: body.name, created_at: now() };
        dictionaries = [...dictionaries.filter((item) => item.code !== code), entry];
        return json(route, entry);
      }
      if (route.request().method() === "DELETE") {
        dictionaries = dictionaries.filter((item) => item.code !== code);
        return json(route, { deleted: true });
      }
    }
    if (path === "/api/v1/system/parameters") return json(route, parameters);
    const parameterMatch = path.match(/^\/api\/v1\/system\/parameters\/([^/]+)$/);
    if (parameterMatch) {
      const key = decodeURIComponent(parameterMatch[1]);
      if (route.request().method() === "PUT") {
        const body = route.request().postDataJSON() as { name: string; value: string };
        const entry = { id: nextParameterId++, key, name: body.name, value: body.value, created_at: now(), updated_at: now() };
        parameters = [...parameters.filter((item) => item.key !== key), entry];
        return json(route, entry);
      }
      if (route.request().method() === "DELETE") {
        parameters = parameters.filter((item) => item.key !== key);
        return json(route, { deleted: true });
      }
    }
    if (path === "/api/v1/system/version-packages") {
      if (route.request().method() === "POST") {
        const body = route.request().postDataJSON() as { manifest: unknown; version_code: string; version_name: string };
        const entry = {
          id: nextVersionId++,
          version_name: body.version_name,
          version_code: body.version_code,
          manifest: body.manifest,
          status: "draft",
          published_at: null,
          retired_at: null,
          created_at: now()
        };
        versions = [...versions, entry];
        return json(route, entry);
      }
      return json(route, versions);
    }
    if (path === "/api/v1/system/version-packages/releases") return json(route, versionReleases);
    const versionPublishMatch = path.match(/^\/api\/v1\/system\/version-packages\/(\d+)\/publish$/);
    if (versionPublishMatch && route.request().method() === "POST") {
      const versionId = Number(versionPublishMatch[1]);
      const body = route.request().postDataJSON() as { reason?: string | null };
      const result = applyVersionAction(versionId, "publish", body.reason ?? null);
      return result ? json(route, result) : json(route, { code: "NOT_FOUND", message: "version not found" }, 404);
    }
    const versionRollbackMatch = path.match(/^\/api\/v1\/system\/version-packages\/(\d+)\/rollback$/);
    if (versionRollbackMatch && route.request().method() === "POST") {
      const versionId = Number(versionRollbackMatch[1]);
      const body = route.request().postDataJSON() as { reason?: string | null };
      const result = applyVersionAction(versionId, "rollback", body.reason ?? null);
      return result ? json(route, result) : json(route, { code: "NOT_FOUND", message: "version not found" }, 404);
    }
    const versionMatch = path.match(/^\/api\/v1\/system\/version-packages\/(\d+)$/);
    if (versionMatch && route.request().method() === "DELETE") {
      const versionId = Number(versionMatch[1]);
      if (versions.find((item) => item.id === versionId)?.status === "active") {
        return json(route, { code: "CONFLICT", message: "active version cannot be deleted" }, 409);
      }
      versions = versions.filter((item) => item.id !== versionId);
      return json(route, { deleted: true });
    }
    if (path === "/api/v1/system/traffic-probes/targets") {
      if (route.request().method() === "POST") {
        const body = route.request().postDataJSON() as { expected_status: number; name: string; url: string };
        const entry = { id: nextProbeId++, name: body.name, url: body.url, expected_status: body.expected_status, status: "pending", created_at: now() };
        probes = [...probes, entry];
        return json(route, entry);
      }
      return json(route, probes);
    }
    const probeMatch = path.match(/^\/api\/v1\/system\/traffic-probes\/targets\/(\d+)$/);
    if (probeMatch) {
      const probeId = Number(probeMatch[1]);
      if (route.request().method() === "DELETE") {
        probes = probes.filter((item) => item.id !== probeId);
        probeResults = probeResults.filter((item) => item.target_id !== probeId);
        probeAlerts = probeAlerts.filter((item) => item.target_id !== probeId);
        probeRunCounts.delete(probeId);
        return json(route, { deleted: true });
      }
    }
    const probeRunMatch = path.match(/^\/api\/v1\/system\/traffic-probes\/targets\/(\d+)\/run$/);
    if (probeRunMatch && route.request().method() === "POST") {
      const targetId = Number(probeRunMatch[1]);
      const runCount = (probeRunCounts.get(targetId) ?? 0) + 1;
      probeRunCounts.set(targetId, runCount);
      const recovered = runCount > 1;
      const result = {
        id: nextProbeResultId++,
        target_id: targetId,
        status: recovered ? "healthy" : "warning",
        detail: recovered
          ? { expected_status: 200, reason: "expected_status_matched", status_code: 200 }
          : { expected_status: 200, reason: "status_mismatch", status_code: 503 },
        probed_at: now()
      };
      probeResults = [...probeResults, result];
      if (recovered) {
        probeAlerts = probeAlerts.map((item) => item.target_id === targetId && item.status !== "resolved"
          ? { ...item, status: "resolved", resolved_at: now() }
          : item);
      } else {
        const alert = {
          id: nextProbeAlertId++,
          target_id: targetId,
          result_id: result.id,
          severity: "warning",
          status: "open",
          reason: "status_mismatch",
          detail: result.detail,
          opened_at: now(),
          acknowledged_at: null,
          resolved_at: null
        };
        probeAlerts = [...probeAlerts, alert];
      }
      probes = probes.map((item) => item.id === targetId ? { ...item, status: result.status } : item);
      return json(route, result);
    }
    if (path === "/api/v1/system/traffic-probes/results") {
      return json(route, probeResults);
    }
    if (path === "/api/v1/system/traffic-probes/alerts") {
      const status = url.searchParams.get("status");
      const targetId = Number(url.searchParams.get("target_id") || 0);
      return json(route, probeAlerts.filter((item) => (!status || item.status === status) && (!targetId || item.target_id === targetId)));
    }
    const probeAlertAckMatch = path.match(/^\/api\/v1\/system\/traffic-probes\/alerts\/(\d+)\/ack$/);
    if (probeAlertAckMatch && route.request().method() === "POST") {
      const alertId = Number(probeAlertAckMatch[1]);
      probeAlerts = probeAlerts.map((item) => item.id === alertId && item.status === "open"
        ? { ...item, status: "acknowledged", acknowledged_at: now() }
        : item);
      return json(route, { updated: true });
    }
    const probeAlertResolveMatch = path.match(/^\/api\/v1\/system\/traffic-probes\/alerts\/(\d+)\/resolve$/);
    if (probeAlertResolveMatch && route.request().method() === "POST") {
      const alertId = Number(probeAlertResolveMatch[1]);
      probeAlerts = probeAlerts.map((item) => item.id === alertId && item.status !== "resolved"
        ? { ...item, status: "resolved", resolved_at: now() }
        : item);
      return json(route, { updated: true });
    }
    if (route.request().method() !== "GET") return json(route, { ok: true });
    return route.continue();
  });
}

function session(mfaEnabled = false, email = "owner@example.com", displayName = "Owner") {
  return {
    authenticated: true,
    user: { id: 1, email, display_name: displayName, status: "active" },
    organization: { id: 1, code: "main", name: "Main", scope: "tenant" },
    product_code: "console",
    client_type: "pc_web",
    permissions: [
      "org:read",
      "permission:read",
      "user:read",
      "user:write",
      "role:read",
      "role:write",
      "api_token:read",
      "api_token:create",
      "api_token:revoke",
      "user:invite",
      "menu:read",
      "server:read",
      "config:read",
      "config:write",
      "dictionary:read",
      "dictionary:write",
      "parameter:read",
      "parameter:write",
      "version_package:read",
      "version_package:write",
      "media:read",
      "media:write",
      "traffic_probe:read",
      "traffic_probe:write"
    ],
    mfa_enabled: mfaEnabled,
    expires_at: now(),
    refresh_token_expires_at: now()
  };
}

function now() {
  return new Date("2026-06-21T00:00:00.000Z").toISOString();
}

function json(route: Route, body: unknown, status = 200) {
  return route.fulfill({
    status,
    contentType: "application/json",
    body: JSON.stringify(body)
  });
}



import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { I18nextProvider } from "react-i18next";
import { MemoryRouter } from "react-router";

import { i18n } from "~/i18n/i18n";
import { resources } from "~/i18n/resources";
import { useAdminWorkspaceStore } from "~/stores/admin-workspace-store";
import { useAuthStore } from "~/stores/auth-store";
import { AdminHeader } from "./AdminHeader";

const authApiMock = vi.hoisted(() => ({
  getMe: vi.fn(),
  listMyOrganizations: vi.fn(),
  logout: vi.fn(),
  switchOrg: vi.fn(),
}));

vi.mock("~/lib/api/auth", () => ({
  authApi: authApiMock,
}));

const en = resources.en;

function openDropdown(button: HTMLElement) {
  fireEvent.pointerDown(button, {
    button: 0,
    ctrlKey: false,
    pointerType: "mouse",
  });
}

function renderAdminHeader(pathname = "/admin/users") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[pathname]}>
          <AdminHeader pathname={pathname} />
        </MemoryRouter>
      </QueryClientProvider>
    </I18nextProvider>,
  );
}

describe("AdminHeader", () => {
  beforeEach(async () => {
    window.localStorage.clear();
    authApiMock.getMe.mockReset();
    authApiMock.listMyOrganizations.mockReset();
    authApiMock.logout.mockReset();
    authApiMock.switchOrg.mockReset();
    useAdminWorkspaceStore.setState({
      hydrated: false,
      tabs: [{ fixed: true, id: "dashboard", labelKey: "admin.nav.dashboard", to: "/admin" }],
    });
    useAuthStore.setState({
      accessExpiresAt: "",
      clientType: "pc_web",
      currentOrgId: "1",
      currentSessionId: "session-1",
      hydrated: true,
      isAuthenticated: true,
      orgs: [
        { code: "default", id: 1, name: "Default" },
        { code: "second", id: 2, name: "Second" },
      ],
      productCode: "aoi-admin",
      refreshExpiresAt: "",
      user: {
        displayName: "Admin",
        email: "admin@example.com",
        id: 1,
        username: "admin",
      },
    });
    await i18n.changeLanguage("en");
  });

  it("adds the current route to workspace tabs and allows closing it", async () => {
    renderAdminHeader("/admin/users");

    expect(await screen.findByRole("link", { name: en.admin.nav.users })).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", {
        name: en.admin.header.tabs.close.replace("{{label}}", en.admin.nav.users),
      }),
    );

    await waitFor(() =>
      expect(useAdminWorkspaceStore.getState().tabs.some((tab) => tab.to === "/admin/users")).toBe(
        false,
      ),
    );
  });

  it("switches organization through the backend auth contract", async () => {
    authApiMock.switchOrg.mockResolvedValue({
      clientType: "pc_web",
      orgId: 2,
      productCode: "aoi-admin",
      sessionId: "session-2",
      userId: 1,
    });
    authApiMock.getMe.mockResolvedValue({ id: 1, username: "admin" });
    authApiMock.listMyOrganizations.mockResolvedValue([
      { code: "second", id: 2, name: "Second" },
      { code: "default", id: 1, name: "Default" },
    ]);
    renderAdminHeader("/admin/users");

    openDropdown(
      screen.getByRole("button", { name: en.admin.header.actions.openOrganizationMenu }),
    );
    fireEvent.click(await screen.findByRole("menuitem", { name: /Second \(second\)/ }));

    await waitFor(() => expect(authApiMock.switchOrg).toHaveBeenCalledWith(2));
    expect(useAuthStore.getState().currentOrgId).toBe("2");
  });

  it("clears local session after logout even when the API request fails", async () => {
    authApiMock.logout.mockRejectedValue(new Error("offline"));
    renderAdminHeader("/admin/users");

    openDropdown(
      screen.getByRole("button", {
        name: en.admin.header.actions.openAccountMenu.replace("{{name}}", "Admin"),
      }),
    );
    fireEvent.click(await screen.findByRole("menuitem", { name: en.common.actions.logout }));

    await waitFor(() => expect(authApiMock.logout).toHaveBeenCalled());
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });
});

import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { I18nextProvider } from "react-i18next";
import { MemoryRouter } from "react-router";

import { i18n } from "~/i18n/i18n";
import { resources } from "~/i18n/resources";
import { AdminSidebarNav, findAdminNavGroupId } from "./layout";

const zhCN = resources["zh-CN"];

function renderSidebar(pathname: string) {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter initialEntries={[pathname]}>
        <AdminSidebarNav pathname={pathname} />
      </MemoryRouter>
    </I18nextProvider>,
  );
}

describe("AdminSidebarNav", () => {
  beforeEach(async () => {
    await i18n.changeLanguage("zh-CN");
  });

  it("maps current admin paths to their business groups", () => {
    expect(findAdminNavGroupId("/admin/users")).toBe("identity");
    expect(findAdminNavGroupId("/admin/media/resumable")).toBe("media");
  });

  it("opens the identity group for the users route and hides closed group links", () => {
    renderSidebar("/admin/users");

    expect(screen.getByRole("button", { name: zhCN.admin.navGroups.identity })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
    expect(screen.getByRole("link", { name: zhCN.admin.nav.users })).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(screen.queryByRole("link", { name: zhCN.admin.nav.mediaResumable })).toBeNull();
  });

  it("keeps only one group open when another group is selected", () => {
    renderSidebar("/admin/users");

    const identityGroup = screen.getByRole("button", { name: zhCN.admin.navGroups.identity });
    const systemGroup = screen.getByRole("button", { name: zhCN.admin.navGroups.system });

    fireEvent.click(systemGroup);

    expect(systemGroup).toHaveAttribute("aria-expanded", "true");
    expect(identityGroup).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByRole("link", { name: zhCN.admin.nav.system })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: zhCN.admin.nav.users })).toBeNull();
  });

  it("opens the media group for resumable upload without marking the media index active", () => {
    renderSidebar("/admin/media/resumable");

    expect(screen.getByRole("button", { name: zhCN.admin.navGroups.media })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
    expect(screen.getByRole("link", { name: zhCN.admin.nav.mediaResumable })).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(screen.getByRole("link", { name: zhCN.admin.nav.media })).not.toHaveAttribute(
      "aria-current",
      "page",
    );
  });
});

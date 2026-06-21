import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { I18nextProvider } from "react-i18next";
import { MemoryRouter } from "react-router";

import { i18n } from "~/i18n/i18n";
import { resources } from "~/i18n/resources";
import { usePreferencesStore } from "~/stores/preferences-store";
import { PreferenceMenu } from "./PreferenceMenu";

const en = resources.en;
const zhCN = resources["zh-CN"];

function openPreferencesMenu() {
  fireEvent.pointerDown(screen.getByRole("button", { name: en.common.preferences.open }), {
    button: 0,
    ctrlKey: false,
    pointerType: "mouse",
  });
}

function renderPreferenceMenu() {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>
        <PreferenceMenu />
      </MemoryRouter>
    </I18nextProvider>,
  );
}

describe("PreferenceMenu", () => {
  beforeEach(async () => {
    document.body.removeAttribute("style");
    window.localStorage.clear();
    usePreferencesStore.setState({
      hydrated: true,
      prefersDark: false,
      resolvedThemeMode: "light",
      themeMode: "system",
    });
    await i18n.changeLanguage("en");
  });

  it("switches theme mode from the relative menu", () => {
    renderPreferenceMenu();

    openPreferencesMenu();
    fireEvent.click(
      screen.getByRole("menuitemradio", { name: en.common.preferences.themeModes.dark }),
    );

    expect(usePreferencesStore.getState().themeMode).toBe("dark");
    expect(usePreferencesStore.getState().resolvedThemeMode).toBe("dark");
  });

  it("opens without locking document scroll", () => {
    renderPreferenceMenu();

    openPreferencesMenu();

    expect(screen.getByRole("menu")).toBeInTheDocument();
    expect(document.body.style.pointerEvents).toBe("");
    expect(document.body.style.overflow).toBe("");
    expect(document.body.style.position).toBe("");
  });

  it("changes language from the shared menu", async () => {
    renderPreferenceMenu();

    openPreferencesMenu();
    fireEvent.click(screen.getByRole("menuitemradio", { name: zhCN.common.locales["zh-CN"] }));

    await waitFor(() => expect(i18n.language).toBe("zh-CN"));
  });
});

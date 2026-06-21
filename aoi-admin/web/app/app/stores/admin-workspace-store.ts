import { create } from "zustand";

import { findAdminNavItem } from "~/features/admin/navigation";

const storageKey = "aoi-admin-workspace-tabs";
const dashboardTab = {
  fixed: true,
  id: "dashboard",
  labelKey: "admin.nav.dashboard",
  to: "/admin",
} satisfies AdminWorkspaceTab;

export type AdminWorkspaceTab = {
  fixed?: boolean;
  id: string;
  labelKey: string;
  to: string;
};

type AdminWorkspaceState = {
  closeTab: (to: string) => void;
  ensureTabForPath: (pathname: string) => void;
  hydrateTabs: () => void;
  hydrated: boolean;
  resetTabs: () => void;
  tabs: AdminWorkspaceTab[];
};

export const useAdminWorkspaceStore = create<AdminWorkspaceState>((set, get) => ({
  closeTab: (to) => {
    const tabs = normalizeTabs(get().tabs.filter((tab) => tab.fixed || tab.to !== to));
    persistTabs(tabs);
    set({ tabs });
  },
  ensureTabForPath: (pathname) => {
    const item = findAdminNavItem(pathname);
    const nextTab = {
      fixed: item.to === dashboardTab.to,
      id: item.id,
      labelKey: item.labelKey,
      to: item.to,
    } satisfies AdminWorkspaceTab;
    const tabs = normalizeTabs([...get().tabs, nextTab]);
    persistTabs(tabs);
    set({ tabs });
  },
  hydrateTabs: () => {
    const tabs = normalizeTabs(readStoredTabs());
    set({ hydrated: true, tabs });
  },
  hydrated: false,
  resetTabs: () => {
    persistTabs([dashboardTab]);
    set({ hydrated: true, tabs: [dashboardTab] });
  },
  tabs: [dashboardTab],
}));

function normalizeTabs(tabs: AdminWorkspaceTab[]) {
  const knownItems = new Map(
    [dashboardTab, ...tabs].map((tab) => {
      const item = findAdminNavItem(tab.to);
      return [
        item.to,
        {
          fixed: item.to === dashboardTab.to,
          id: item.id,
          labelKey: item.labelKey,
          to: item.to,
        } satisfies AdminWorkspaceTab,
      ];
    }),
  );

  const normalized = Array.from(knownItems.values());
  return [
    dashboardTab,
    ...normalized.filter((tab) => tab.to !== dashboardTab.to && tab.to.startsWith("/admin")),
  ];
}

function readStoredTabs() {
  const storage = browserStorage();
  if (!storage) {
    return [dashboardTab];
  }

  const rawValue = storage.getItem(storageKey);
  if (!rawValue) {
    return [dashboardTab];
  }

  try {
    const parsed = JSON.parse(rawValue) as unknown;
    if (!Array.isArray(parsed)) {
      return [dashboardTab];
    }
    return parsed.filter(isStoredTab).map((tab) => ({
      id: tab.id,
      labelKey: tab.labelKey,
      to: tab.to,
    }));
  } catch {
    return [dashboardTab];
  }
}

function persistTabs(tabs: AdminWorkspaceTab[]) {
  const storage = browserStorage();
  if (!storage) {
    return;
  }
  storage.setItem(
    storageKey,
    JSON.stringify(
      tabs.filter((tab) => !tab.fixed).map(({ id, labelKey, to }) => ({ id, labelKey, to })),
    ),
  );
}

function isStoredTab(value: unknown): value is AdminWorkspaceTab {
  if (!value || typeof value !== "object") {
    return false;
  }
  const tab = value as Partial<AdminWorkspaceTab>;
  return Boolean(
    typeof tab.id === "string" &&
    typeof tab.labelKey === "string" &&
    typeof tab.to === "string" &&
    tab.to.startsWith("/admin"),
  );
}

function browserStorage() {
  return typeof window === "undefined" ? undefined : window.localStorage;
}

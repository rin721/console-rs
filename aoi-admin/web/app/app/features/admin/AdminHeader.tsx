import { useQueryClient } from "@tanstack/react-query";
import { Check, ChevronDown, LogOut, Palette, Settings, UserRound, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router";
import { useTranslation } from "react-i18next";

import { IconButton } from "~/components/aoi/primitives/IconButton";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "~/components/aoi/patterns/DropdownMenu";
import { PreferenceMenu } from "~/components/aoi/patterns/PreferenceMenu";
import {
  findAdminNavGroup,
  findAdminNavItem,
  normalizeAdminNavPath,
} from "~/features/admin/navigation";
import { authApi } from "~/lib/api/auth";
import { ApiError } from "~/lib/api/client";
import { queryKeys } from "~/lib/api/query-keys";
import type { CurrentUser, Organization } from "~/lib/api/types";
import { useAdminWorkspaceStore, type AdminWorkspaceTab } from "~/stores/admin-workspace-store";
import { useAuthStore } from "~/stores/auth-store";

type AdminHeaderProps = {
  pathname: string;
};

type HeaderAction = "identity" | "logout" | `org:${string}`;

export function AdminHeader({ pathname }: AdminHeaderProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [busyAction, setBusyAction] = useState<HeaderAction | null>(null);
  const [statusMessage, setStatusMessage] = useState("");
  const currentGroup = useMemo(() => findAdminNavGroup(pathname), [pathname]);
  const currentItem = useMemo(() => findAdminNavItem(pathname), [pathname]);
  const tabs = useAdminWorkspaceStore((state) => state.tabs);
  const tabsHydrated = useAdminWorkspaceStore((state) => state.hydrated);
  const closeTab = useAdminWorkspaceStore((state) => state.closeTab);
  const ensureTabForPath = useAdminWorkspaceStore((state) => state.ensureTabForPath);
  const hydrateTabs = useAdminWorkspaceStore((state) => state.hydrateTabs);
  const currentOrgId = useAuthStore((state) => state.currentOrgId);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const orgs = useAuthStore((state) => state.orgs);
  const setIdentity = useAuthStore((state) => state.setIdentity);
  const setSession = useAuthStore((state) => state.setSession);
  const clearSession = useAuthStore((state) => state.clearSession);
  const user = useAuthStore((state) => state.user);
  const currentOrg = useMemo(
    () => orgs.find((org) => sameId(org.id, currentOrgId)) ?? orgs[0] ?? null,
    [currentOrgId, orgs],
  );

  const refreshIdentity = useCallback(async () => {
    if (!isAuthenticated) {
      return;
    }
    setBusyAction("identity");
    try {
      const [nextUser, nextOrgs] = await Promise.all([
        authApi.getMe(),
        authApi.listMyOrganizations(),
      ]);
      setIdentity(nextUser, nextOrgs);
    } catch {
      setStatusMessage(t("admin.header.messages.identityFailed"));
    } finally {
      setBusyAction((current) => (current === "identity" ? null : current));
    }
  }, [isAuthenticated, setIdentity, t]);

  useEffect(() => {
    hydrateTabs();
  }, [hydrateTabs]);

  useEffect(() => {
    if (tabsHydrated) {
      ensureTabForPath(pathname);
    }
  }, [ensureTabForPath, pathname, tabsHydrated]);

  useEffect(() => {
    if (isAuthenticated && !user && busyAction !== "identity") {
      void refreshIdentity();
    }
  }, [busyAction, isAuthenticated, refreshIdentity, user]);

  async function handleSwitchOrg(org: Organization) {
    if (sameId(org.id, currentOrgId)) {
      return;
    }

    setBusyAction(`org:${String(org.id)}`);
    setStatusMessage("");
    try {
      const pair = await authApi.switchOrg(org.id);
      setSession(pair);
      const [nextUser, nextOrgs] = await Promise.all([
        authApi.getMe(),
        authApi.listMyOrganizations(),
      ]);
      setIdentity(nextUser, nextOrgs);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.auth.root }),
        queryClient.invalidateQueries({ queryKey: queryKeys.iam.root }),
        queryClient.invalidateQueries({ queryKey: queryKeys.system.root }),
      ]);
      setStatusMessage(t("admin.header.messages.organizationSwitched"));
    } catch (error) {
      setStatusMessage(errorMessage(error, t("errors.api.requestFailed")));
    } finally {
      setBusyAction(null);
    }
  }

  async function handleLogout() {
    setBusyAction("logout");
    try {
      await authApi.logout();
    } catch {
      // Local session cleanup is the source of truth for leaving the browser session.
    } finally {
      clearSession();
      queryClient.clear();
      setBusyAction(null);
      await navigate("/login", { replace: true });
    }
  }

  function handleCloseTab(tab: AdminWorkspaceTab) {
    const remainingTabs = tabs.filter((candidate) => candidate.fixed || candidate.to !== tab.to);
    closeTab(tab.to);
    if (samePath(tab.to, pathname)) {
      void navigate(remainingTabs.at(-1)?.to ?? "/admin");
    }
  }

  return (
    <header className="aoi-admin-header">
      <div className="aoi-admin-topbar">
        <div className="aoi-admin-topbar__context">
          <span>{t(currentGroup.labelKey)}</span>
          <strong>{t(currentItem.labelKey)}</strong>
        </div>
        <div className="aoi-admin-topbar__actions">
          <OrganizationMenu
            busyAction={busyAction}
            currentOrg={currentOrg}
            orgs={orgs}
            onSwitchOrg={handleSwitchOrg}
          />
          <PreferenceMenu />
          <Link
            aria-label={t("admin.header.actions.openSystemSettings")}
            className="aoi-icon-button aoi-icon-button--ghost"
            to="/admin/system"
          >
            <Settings aria-hidden="true" size={18} />
          </Link>
          <UserMenu
            busyAction={busyAction}
            email={user?.email ?? ""}
            name={displayUserName(user, t("admin.header.userFallback"))}
            onLogout={handleLogout}
          />
        </div>
      </div>
      <nav className="aoi-admin-tabs" aria-label={t("admin.header.tabs.label")}>
        {tabs.map((tab) => {
          const active = samePath(tab.to, pathname);
          const item = findAdminNavItem(tab.to);
          const TabIcon = item.icon;

          return (
            <span className="aoi-admin-tab" data-active={active ? "true" : "false"} key={tab.to}>
              <Link to={tab.to}>
                <TabIcon aria-hidden="true" size={15} />
                <span>{t(tab.labelKey)}</span>
              </Link>
              {tab.fixed ? null : (
                <IconButton
                  appearance="ghost"
                  className="aoi-admin-tab__close"
                  icon={<X aria-hidden="true" size={14} />}
                  label={t("admin.header.tabs.close", { label: t(tab.labelKey) })}
                  onClick={() => handleCloseTab(tab)}
                />
              )}
            </span>
          );
        })}
      </nav>
      {statusMessage ? (
        <p className="aoi-admin-header__status" role="status">
          {statusMessage}
        </p>
      ) : null}
    </header>
  );
}

function OrganizationMenu({
  busyAction,
  currentOrg,
  orgs,
  onSwitchOrg,
}: {
  busyAction: HeaderAction | null;
  currentOrg: Organization | null;
  orgs: Organization[];
  onSwitchOrg: (org: Organization) => void | Promise<void>;
}) {
  const { t } = useTranslation();
  const currentOrgLabel = organizationLabel(currentOrg, t("admin.header.organizationFallback"));

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          aria-label={t("admin.header.actions.openOrganizationMenu")}
          className="aoi-admin-org-trigger"
          type="button"
        >
          <span>
            <small>{t("admin.header.organization")}</small>
            <strong>{currentOrgLabel}</strong>
          </span>
          <ChevronDown aria-hidden="true" size={16} />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="aoi-admin-org-menu" align="end">
        <DropdownMenuLabel>{t("admin.header.organization")}</DropdownMenuLabel>
        {orgs.length ? (
          orgs.map((org) => {
            const selected = sameId(org.id, currentOrg?.id);
            const actionId = `org:${String(org.id)}` as const;

            return (
              <DropdownMenuItem
                disabled={busyAction !== null}
                icon={selected ? <Check aria-hidden="true" size={15} /> : null}
                key={String(org.id)}
                onSelect={(event) => {
                  event.preventDefault();
                  void onSwitchOrg(org);
                }}
              >
                {busyAction === actionId
                  ? t("admin.header.organizationSwitching")
                  : organizationLabel(org, t("admin.header.organizationFallback"))}
              </DropdownMenuItem>
            );
          })
        ) : (
          <DropdownMenuItem disabled>{t("admin.header.organizationEmpty")}</DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function UserMenu({
  busyAction,
  email,
  name,
  onLogout,
}: {
  busyAction: HeaderAction | null;
  email: string;
  name: string;
  onLogout: () => void | Promise<void>;
}) {
  const { t } = useTranslation();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          aria-label={t("admin.header.actions.openAccountMenu", { name })}
          className="aoi-admin-user-trigger"
          type="button"
        >
          <span className="aoi-admin-user-trigger__avatar" aria-hidden="true">
            {name.slice(0, 1).toUpperCase()}
          </span>
          <span className="aoi-admin-user-trigger__copy">
            <strong>{name}</strong>
            {email ? <small>{email}</small> : null}
          </span>
          <ChevronDown aria-hidden="true" size={15} />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="aoi-admin-user-menu" align="end">
        <DropdownMenuLabel>{t("admin.header.account")}</DropdownMenuLabel>
        <div className="aoi-dropdown-menu-profile">
          <UserRound aria-hidden="true" size={18} />
          <span>
            <strong>{name}</strong>
            {email ? <small>{email}</small> : null}
          </span>
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuItem asChild>
          <Link to="/admin/design-system">
            <Palette aria-hidden="true" size={16} />
            <span>{t("admin.header.actions.openDesignSystem")}</span>
          </Link>
        </DropdownMenuItem>
        <DropdownMenuItem asChild>
          <Link to="/admin/system">
            <Settings aria-hidden="true" size={16} />
            <span>{t("admin.header.actions.openSystemSettings")}</span>
          </Link>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          disabled={busyAction === "logout"}
          icon={<LogOut aria-hidden="true" size={16} />}
          onSelect={(event) => {
            event.preventDefault();
            void onLogout();
          }}
        >
          {busyAction === "logout"
            ? t("admin.header.actions.loggingOut")
            : t("common.actions.logout")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function displayUserName(user: CurrentUser | null, fallback: string) {
  return user?.displayName?.trim() || user?.username?.trim() || user?.email?.trim() || fallback;
}

function organizationLabel(org: Organization | null, fallback: string) {
  if (!org) {
    return fallback;
  }
  const name = org.name?.trim();
  const code = org.code?.trim();
  if (name && code) {
    return `${name} (${code})`;
  }
  return name || code || String(org.id);
}

function sameId(left: unknown, right: unknown) {
  return left !== undefined && right !== undefined && String(left) === String(right);
}

function samePath(left: string, right: string) {
  return normalizeAdminNavPath(left) === normalizeAdminNavPath(right);
}

function errorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message || fallback : fallback;
}

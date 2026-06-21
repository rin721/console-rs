import {
  BookOpenText,
  Building2,
  Bug,
  Code2,
  HeartPulse,
  History,
  ImageUp,
  KeyRound,
  LayoutDashboard,
  LockKeyhole,
  LogIn,
  MonitorCheck,
  PackageCheck,
  Palette,
  PanelLeft,
  Plug,
  ScrollText,
  Settings,
  Shield,
  ShieldAlert,
  ShieldCheck,
  SlidersHorizontal,
  Users,
  type LucideIcon,
} from "lucide-react";

type AdminNavGroupId = "workspace" | "identity" | "system" | "logs" | "media" | "integration";

export type AdminNavItem = {
  end?: boolean;
  icon: LucideIcon;
  id: string;
  labelKey: string;
  to: string;
};

export type AdminNavGroup = {
  icon: LucideIcon;
  id: AdminNavGroupId;
  items: AdminNavItem[];
  labelKey: string;
};

export const adminNavGroups: AdminNavGroup[] = [
  {
    icon: LayoutDashboard,
    id: "workspace",
    items: [
      {
        end: true,
        icon: LayoutDashboard,
        id: "dashboard",
        labelKey: "admin.nav.dashboard",
        to: "/admin",
      },
    ],
    labelKey: "admin.navGroups.workspace",
  },
  {
    icon: ShieldCheck,
    id: "identity",
    items: [
      { icon: ShieldCheck, id: "iam", labelKey: "admin.nav.iam", to: "/admin/iam" },
      {
        icon: Building2,
        id: "organizations",
        labelKey: "admin.nav.organizations",
        to: "/admin/organizations",
      },
      { icon: Users, id: "users", labelKey: "admin.nav.users", to: "/admin/users" },
      { icon: Shield, id: "roles", labelKey: "admin.nav.roles", to: "/admin/roles" },
      {
        icon: MonitorCheck,
        id: "sessions",
        labelKey: "admin.nav.sessions",
        to: "/admin/sessions",
      },
      {
        icon: LockKeyhole,
        id: "security",
        labelKey: "admin.nav.security",
        to: "/admin/security",
      },
      {
        icon: ShieldAlert,
        id: "traffic-hijack",
        labelKey: "admin.nav.trafficHijack",
        to: "/admin/traffic-hijack",
      },
      {
        icon: KeyRound,
        id: "api-tokens",
        labelKey: "admin.nav.apiTokens",
        to: "/admin/api-tokens",
      },
    ],
    labelKey: "admin.navGroups.identity",
  },
  {
    icon: Settings,
    id: "system",
    items: [
      { icon: HeartPulse, id: "probes", labelKey: "admin.nav.probes", to: "/admin/probes" },
      { icon: PanelLeft, id: "menus", labelKey: "admin.nav.menus", to: "/admin/menus" },
      {
        icon: BookOpenText,
        id: "dictionaries",
        labelKey: "admin.nav.dictionaries",
        to: "/admin/dictionaries",
      },
      { icon: Settings, id: "system", labelKey: "admin.nav.system", to: "/admin/system" },
      {
        icon: Palette,
        id: "design-system",
        labelKey: "admin.nav.designSystem",
        to: "/admin/design-system",
      },
      {
        icon: SlidersHorizontal,
        id: "parameters",
        labelKey: "admin.nav.parameters",
        to: "/admin/parameters",
      },
    ],
    labelKey: "admin.navGroups.system",
  },
  {
    icon: ScrollText,
    id: "logs",
    items: [
      {
        icon: History,
        id: "operation-records",
        labelKey: "admin.nav.operationRecords",
        to: "/admin/operation-records",
      },
      {
        icon: ScrollText,
        id: "audit-logs",
        labelKey: "admin.nav.auditLogs",
        to: "/admin/audit-logs",
      },
      { icon: LogIn, id: "login-logs", labelKey: "admin.nav.loginLogs", to: "/admin/login-logs" },
      { icon: Bug, id: "error-logs", labelKey: "admin.nav.errorLogs", to: "/admin/error-logs" },
    ],
    labelKey: "admin.navGroups.logs",
  },
  {
    icon: ImageUp,
    id: "media",
    items: [
      {
        end: true,
        icon: ImageUp,
        id: "media",
        labelKey: "admin.nav.media",
        to: "/admin/media",
      },
      {
        icon: ImageUp,
        id: "media-resumable",
        labelKey: "admin.nav.mediaResumable",
        to: "/admin/media/resumable",
      },
    ],
    labelKey: "admin.navGroups.media",
  },
  {
    icon: Plug,
    id: "integration",
    items: [
      { icon: Code2, id: "apis", labelKey: "admin.nav.apis", to: "/admin/apis" },
      { icon: Plug, id: "plugins", labelKey: "admin.nav.plugins", to: "/admin/plugins" },
      {
        icon: PackageCheck,
        id: "versions",
        labelKey: "admin.nav.versions",
        to: "/admin/versions",
      },
    ],
    labelKey: "admin.navGroups.integration",
  },
];

export function normalizeAdminNavPath(pathname: string) {
  return pathname.replace(/\/+$/, "") || "/";
}

export function adminNavItemMatchesPath(item: AdminNavItem, pathname: string) {
  const normalizedPath = normalizeAdminNavPath(pathname);
  const normalizedTarget = normalizeAdminNavPath(item.to);

  if (item.end) {
    return normalizedPath === normalizedTarget;
  }

  return normalizedPath === normalizedTarget || normalizedPath.startsWith(`${normalizedTarget}/`);
}

export function findAdminNavGroup(
  pathname: string,
  groups: readonly AdminNavGroup[] = adminNavGroups,
) {
  return (
    groups.find((group) => group.items.some((item) => adminNavItemMatchesPath(item, pathname))) ??
    groups[0]
  );
}

export function findAdminNavGroupId(
  pathname: string,
  groups: readonly AdminNavGroup[] = adminNavGroups,
) {
  return findAdminNavGroup(pathname, groups).id;
}

export function findAdminNavItem(
  pathname: string,
  groups: readonly AdminNavGroup[] = adminNavGroups,
) {
  return (
    groups
      .flatMap((group) => group.items)
      .find((item) => adminNavItemMatchesPath(item, pathname)) ?? groups[0].items[0]
  );
}

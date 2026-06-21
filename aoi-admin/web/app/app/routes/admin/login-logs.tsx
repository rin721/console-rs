import { useQuery } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import {
  Activity,
  Database,
  LogIn,
  MapPin,
  Monitor,
  RefreshCw,
  RotateCcw,
  Search,
  UserRound,
} from "lucide-react";
import { useMemo, useState, type FormEvent, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { DataTable } from "~/components/aoi/patterns/DataTable";
import { FormField } from "~/components/aoi/patterns/FormField";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { PlatformTag } from "~/features/admin/PlatformTag";
import { ApiError } from "~/lib/api/client";
import { iamApi, type IAMAuditLogListQuery } from "~/lib/api/iam";
import { queryKeys } from "~/lib/api/query-keys";
import type { IAMAuditLog } from "~/lib/api/types";
import { useAuthStore } from "~/stores/auth-store";

const loginAction = "auth.login";
const defaultLimit = 50;
const maxLimit = 200;

type LoginLogFilters = IAMAuditLogListQuery & {
  ipAddress?: string;
};

type LoginLogFilterDraft = {
  from: string;
  ipAddress: string;
  limit: string;
  to: string;
  userId: string;
};

const initialDraft: LoginLogFilterDraft = {
  from: "",
  ipAddress: "",
  limit: String(defaultLimit),
  to: "",
  userId: "",
};

export default function AdminLoginLogsRoute() {
  const { i18n, t } = useTranslation();
  const currentOrgId = useAuthStore((state) => state.currentOrgId);
  const [draft, setDraft] = useState<LoginLogFilterDraft>(initialDraft);
  const [filters, setFilters] = useState<LoginLogFilters>(normalizeFilters(initialDraft));

  const apiFilters = useMemo<IAMAuditLogListQuery>(
    () => ({
      action: loginAction,
      from: filters.from,
      limit: filters.limit,
      to: filters.to,
      userId: filters.userId,
    }),
    [filters.from, filters.limit, filters.to, filters.userId],
  );

  const loginLogsQuery = useQuery({
    enabled: Boolean(currentOrgId),
    queryFn: ({ signal }) => iamApi.listAuditLogs(currentOrgId ?? "", apiFilters, { signal }),
    queryKey: queryKeys.iam.auditLogs(i18n.language, currentOrgId ?? "", apiFilters),
  });

  const logs = useMemo(
    () => filterLoginLogs(loginLogsQuery.data ?? [], filters.ipAddress),
    [filters.ipAddress, loginLogsQuery.data],
  );
  const summary = summarizeLoginLogs(logs);

  const loginLogColumns = useMemo<ColumnDef<IAMAuditLog>[]>(
    () => [
      {
        accessorKey: "id",
        cell: ({ row }) => <code className="aoi-audit-code">{row.original.id}</code>,
        header: t("admin.loginLogs.columns.id"),
      },
      {
        accessorKey: "userId",
        cell: ({ row }) =>
          row.original.userId !== null && row.original.userId !== undefined ? (
            <code className="aoi-audit-code">{row.original.userId}</code>
          ) : (
            t("common.labels.none")
          ),
        header: t("admin.loginLogs.columns.userId"),
      },
      {
        accessorKey: "ipAddress",
        cell: ({ row }) =>
          row.original.ipAddress ? (
            <code className="aoi-audit-code">{row.original.ipAddress}</code>
          ) : (
            t("common.labels.none")
          ),
        header: t("admin.loginLogs.columns.ip"),
      },
      {
        id: "platform",
        cell: ({ row }) => (
          <PlatformTag
            clientType={row.original.clientType}
            productCode={row.original.productCode}
          />
        ),
        header: t("admin.loginLogs.columns.platform"),
      },
      {
        accessorKey: "userAgent",
        cell: ({ row }) => (
          <span className="aoi-login-log-device">{deviceName(row.original.userAgent, t)}</span>
        ),
        header: t("admin.loginLogs.columns.device"),
      },
      {
        accessorKey: "resource",
        cell: ({ row }) => (
          <div className="aoi-audit-resource">
            <strong>{row.original.resource || t("common.labels.none")}</strong>
            <span>{row.original.resourceId || t("common.labels.none")}</span>
          </div>
        ),
        header: t("admin.loginLogs.columns.resource"),
      },
      {
        accessorKey: "createdAt",
        cell: ({ row }) => formatDate(row.original.createdAt, i18n.language, t),
        header: t("admin.loginLogs.columns.loginTime"),
      },
    ],
    [i18n.language, t],
  );

  const updateDraft = (key: keyof LoginLogFilterDraft, value: string) => {
    setDraft((current) => ({ ...current, [key]: value }));
  };

  const submitFilters = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFilters(normalizeFilters(draft));
  };

  const resetFilters = () => {
    setDraft(initialDraft);
    setFilters(normalizeFilters(initialDraft));
  };

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-login-logs-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.loginLogs.badge")}</Badge>
          <h1 id="admin-login-logs-title">{t("admin.loginLogs.title")}</h1>
          <p>{t("admin.loginLogs.description")}</p>
        </div>
        <Button
          appearance="secondary"
          disabled={!currentOrgId}
          icon={<RefreshCw size={17} />}
          loading={loginLogsQuery.isFetching}
          onClick={() => void loginLogsQuery.refetch()}
        >
          {t("admin.loginLogs.actions.refresh")}
        </Button>
      </div>

      {!currentOrgId ? (
        <StateBlock
          intent="danger"
          title={t("admin.loginLogs.states.missingOrgTitle")}
          description={t("admin.loginLogs.states.missingOrgDescription")}
        />
      ) : null}

      {loginLogsQuery.error ? (
        <StateBlock
          intent="danger"
          title={errorTitle(loginLogsQuery.error, t)}
          description={errorDescription(loginLogsQuery.error, t)}
        />
      ) : null}

      <div className="aoi-admin-stat-grid" aria-label={t("admin.loginLogs.summaryLabel")}>
        <LoginLogStatCard
          icon={<LogIn size={19} />}
          label={t("admin.loginLogs.metrics.returned")}
          value={
            loginLogsQuery.data
              ? formatNumber(logs.length, i18n.language)
              : fallbackValue(loginLogsQuery.isLoading, t)
          }
        />
        <LoginLogStatCard
          icon={<UserRound size={19} />}
          label={t("admin.loginLogs.metrics.users")}
          value={formatNumber(summary.users, i18n.language)}
        />
        <LoginLogStatCard
          icon={<MapPin size={19} />}
          label={t("admin.loginLogs.metrics.ipAddresses")}
          value={formatNumber(summary.ipAddresses, i18n.language)}
        />
        <LoginLogStatCard
          icon={<Monitor size={19} />}
          label={t("admin.loginLogs.metrics.platforms")}
          value={formatNumber(summary.platforms, i18n.language)}
        />
        <LoginLogStatCard
          icon={<Activity size={19} />}
          label={t("admin.loginLogs.metrics.action")}
          value={loginAction}
        />
      </div>

      <section className="aoi-admin-panel">
        <header>
          <h2>{t("admin.loginLogs.filters.title")}</h2>
          <p>{t("admin.loginLogs.filters.description")}</p>
        </header>
        <form className="aoi-admin-filter-form" onSubmit={submitFilters}>
          <FormField
            label={t("admin.loginLogs.filters.userId")}
            min={1}
            type="number"
            value={draft.userId}
            onChange={(event) => updateDraft("userId", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.loginLogs.filters.ipAddress")}
            value={draft.ipAddress}
            onChange={(event) => updateDraft("ipAddress", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.loginLogs.filters.from")}
            type="datetime-local"
            value={draft.from}
            onChange={(event) => updateDraft("from", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.loginLogs.filters.to")}
            type="datetime-local"
            value={draft.to}
            onChange={(event) => updateDraft("to", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.loginLogs.filters.limit")}
            max={maxLimit}
            min={1}
            type="number"
            value={draft.limit}
            onChange={(event) => updateDraft("limit", event.currentTarget.value)}
          />
          <div className="aoi-admin-filter-actions">
            <Button icon={<Search size={17} />} loading={loginLogsQuery.isFetching} type="submit">
              {t("admin.loginLogs.actions.search")}
            </Button>
            <Button appearance="secondary" icon={<RotateCcw size={17} />} onClick={resetFilters}>
              {t("admin.loginLogs.actions.reset")}
            </Button>
          </div>
        </form>
      </section>

      <section className="aoi-admin-panel">
        <header className="aoi-admin-panel-header-row">
          <div>
            <h2>{t("admin.loginLogs.list.title")}</h2>
            <p>
              {t("admin.loginLogs.list.description", {
                action: loginAction,
                count: logs.length,
                limit: filters.limit ?? defaultLimit,
              })}
            </p>
          </div>
          <span className="aoi-iam-audit-count">
            <Database aria-hidden="true" size={16} />
            {t("admin.loginLogs.list.backendAction", { action: loginAction })}
          </span>
        </header>

        {loginLogsQuery.isLoading ? (
          <StateBlock
            title={t("admin.loginLogs.states.loadingTitle")}
            description={t("admin.loginLogs.states.loadingDescription")}
          />
        ) : loginLogsQuery.data ? (
          <div className="aoi-login-log-table">
            <DataTable
              columns={loginLogColumns}
              data={logs}
              emptyLabel={t("admin.loginLogs.empty")}
            />
          </div>
        ) : (
          <StateBlock
            title={t("admin.loginLogs.states.emptyTitle")}
            description={t("admin.loginLogs.states.emptyDescription")}
          />
        )}
      </section>
    </section>
  );
}

type LoginLogStatCardProps = {
  icon: ReactNode;
  label: string;
  value: string;
};

function LoginLogStatCard({ icon, label, value }: LoginLogStatCardProps) {
  return (
    <article className="aoi-admin-stat-card">
      <span aria-hidden="true">{icon}</span>
      <div>
        <p>{label}</p>
        <strong>{value}</strong>
      </div>
    </article>
  );
}

function normalizeFilters(draft: LoginLogFilterDraft): LoginLogFilters {
  return {
    action: loginAction,
    from: datetimeLocalToISOString(draft.from),
    ipAddress: trimmedOrUndefined(draft.ipAddress),
    limit: parseLimit(draft.limit),
    to: datetimeLocalToISOString(draft.to),
    userId: parseID(draft.userId),
  };
}

function filterLoginLogs(items: IAMAuditLog[], ipAddress: string | undefined) {
  const ipFilter = ipAddress?.trim();
  return items.filter((item) => {
    if (item.action !== loginAction) {
      return false;
    }
    if (!ipFilter) {
      return true;
    }
    return item.ipAddress.includes(ipFilter);
  });
}

function trimmedOrUndefined(value: string | undefined) {
  const trimmed = value?.trim();
  return trimmed || undefined;
}

function parseID(value: string) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return undefined;
  }
  return Math.trunc(parsed);
}

function parseLimit(value: string) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return defaultLimit;
  }
  return Math.min(maxLimit, Math.max(1, Math.trunc(parsed)));
}

function datetimeLocalToISOString(value: string) {
  if (!value) {
    return undefined;
  }
  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) {
    return undefined;
  }
  return new Date(timestamp).toISOString();
}

function summarizeLoginLogs(items: IAMAuditLog[]) {
  const users = new Set<string>();
  const ipAddresses = new Set<string>();
  const platforms = new Set<string>();

  for (const item of items) {
    if (item.userId !== null && item.userId !== undefined && item.userId !== "") {
      users.add(String(item.userId));
    }
    if (item.ipAddress) {
      ipAddresses.add(item.ipAddress);
    }
    if (item.clientType) {
      platforms.add(item.clientType);
    }
  }

  return {
    ipAddresses: ipAddresses.size,
    platforms: platforms.size,
    users: users.size,
  };
}

function deviceName(value: string, t?: ReturnType<typeof useTranslation>["t"]) {
  if (!value) {
    return t ? t("common.labels.none") : "";
  }
  if (value.includes("Firefox")) {
    return "Firefox";
  }
  if (value.includes("Edg/")) {
    return "Edge";
  }
  if (value.includes("Chrome")) {
    return "Chrome";
  }
  if (value.includes("Safari")) {
    return "Safari";
  }
  return value.length > 48 ? `${value.slice(0, 48)}...` : value;
}

function fallbackValue(loading: boolean, t: ReturnType<typeof useTranslation>["t"]) {
  return loading ? t("loading.app") : t("common.labels.none");
}

function formatNumber(value: number, locale: string) {
  return new Intl.NumberFormat(locale).format(value);
}

function formatDate(
  value: string | null | undefined,
  locale: string,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (!value) {
    return t("common.labels.none");
  }
  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) {
    return value;
  }
  return new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(timestamp);
}

function errorTitle(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.loginLogs.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.loginLogs.states.errorTitle");
}

function errorDescription(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.loginLogs.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

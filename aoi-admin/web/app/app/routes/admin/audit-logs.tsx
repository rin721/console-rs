import { useQuery } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import {
  Activity,
  ChevronsDown,
  Database,
  RefreshCw,
  RotateCcw,
  ScrollText,
  Search,
  ShieldAlert,
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

const defaultLimit = 100;
const maxLimit = 500;

type AuditLogFilters = IAMAuditLogListQuery;

type AuditLogFilterDraft = {
  action: string;
  cursor: string;
  from: string;
  limit: string;
  to: string;
  userId: string;
};

const initialDraft: AuditLogFilterDraft = {
  action: "",
  cursor: "",
  from: "",
  limit: String(defaultLimit),
  to: "",
  userId: "",
};

export default function AdminAuditLogsRoute() {
  const { i18n, t } = useTranslation();
  const currentOrgId = useAuthStore((state) => state.currentOrgId);
  const [draft, setDraft] = useState<AuditLogFilterDraft>(initialDraft);
  const [filters, setFilters] = useState<AuditLogFilters>(normalizeFilters(initialDraft));

  const auditLogsQuery = useQuery({
    enabled: Boolean(currentOrgId),
    queryFn: ({ signal }) => iamApi.listAuditLogs(currentOrgId ?? "", filters, { signal }),
    queryKey: queryKeys.iam.auditLogs(i18n.language, currentOrgId ?? "", filters),
  });

  const logs = auditLogsQuery.data ?? [];
  const summary = summarizeAuditLogs(logs);

  const auditLogColumns = useMemo<ColumnDef<IAMAuditLog>[]>(
    () => [
      {
        accessorKey: "id",
        cell: ({ row }) => <code className="aoi-audit-code">{row.original.id}</code>,
        header: t("admin.auditLogs.columns.id"),
      },
      {
        accessorKey: "action",
        cell: ({ row }) => <code className="aoi-audit-action">{row.original.action}</code>,
        header: t("admin.auditLogs.columns.action"),
      },
      {
        id: "resource",
        cell: ({ row }) => (
          <div className="aoi-audit-resource">
            <strong>{row.original.resource || t("common.labels.none")}</strong>
            <span>{row.original.resourceId || t("common.labels.none")}</span>
          </div>
        ),
        header: t("admin.auditLogs.columns.resource"),
      },
      {
        id: "user",
        cell: ({ row }) => (
          <div className="aoi-audit-user">
            <code>{row.original.userId ?? t("common.labels.none")}</code>
            <span>{row.original.orgId ?? t("common.labels.none")}</span>
          </div>
        ),
        header: t("admin.auditLogs.columns.user"),
      },
      {
        accessorKey: "ipAddress",
        cell: ({ row }) =>
          row.original.ipAddress ? (
            <code className="aoi-audit-code">{row.original.ipAddress}</code>
          ) : (
            t("common.labels.none")
          ),
        header: t("admin.auditLogs.columns.ip"),
      },
      {
        id: "platform",
        cell: ({ row }) => (
          <PlatformTag
            clientType={row.original.clientType}
            productCode={row.original.productCode}
          />
        ),
        header: t("admin.auditLogs.columns.platform"),
      },
      {
        accessorKey: "userAgent",
        cell: ({ row }) => (
          <span className="aoi-audit-agent">
            {row.original.userAgent || t("common.labels.none")}
          </span>
        ),
        header: t("admin.auditLogs.columns.device"),
      },
      {
        accessorKey: "createdAt",
        cell: ({ row }) => formatDate(row.original.createdAt, i18n.language, t),
        header: t("admin.auditLogs.columns.time"),
      },
      {
        accessorKey: "metadata",
        cell: ({ row }) => (
          <pre className="aoi-audit-metadata">{prettyMetadata(row.original.metadata, t)}</pre>
        ),
        header: t("admin.auditLogs.columns.metadata"),
      },
    ],
    [i18n.language, t],
  );

  const updateDraft = (key: keyof AuditLogFilterDraft, value: string) => {
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
    <section className="aoi-admin-dashboard" aria-labelledby="admin-audit-logs-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.auditLogs.badge")}</Badge>
          <h1 id="admin-audit-logs-title">{t("admin.auditLogs.title")}</h1>
          <p>{t("admin.auditLogs.description")}</p>
        </div>
        <Button
          appearance="secondary"
          disabled={!currentOrgId}
          icon={<RefreshCw size={17} />}
          loading={auditLogsQuery.isFetching}
          onClick={() => void auditLogsQuery.refetch()}
        >
          {t("admin.auditLogs.actions.refresh")}
        </Button>
      </div>

      {!currentOrgId ? (
        <StateBlock
          intent="danger"
          title={t("admin.auditLogs.states.missingOrgTitle")}
          description={t("admin.auditLogs.states.missingOrgDescription")}
        />
      ) : null}

      {auditLogsQuery.error ? (
        <StateBlock
          intent="danger"
          title={errorTitle(auditLogsQuery.error, t)}
          description={errorDescription(auditLogsQuery.error, t)}
        />
      ) : null}

      <div className="aoi-admin-stat-grid" aria-label={t("admin.auditLogs.summaryLabel")}>
        <AuditStatCard
          icon={<ScrollText size={19} />}
          label={t("admin.auditLogs.metrics.returned")}
          value={
            auditLogsQuery.data
              ? formatNumber(logs.length, i18n.language)
              : fallbackValue(auditLogsQuery.isLoading, t)
          }
        />
        <AuditStatCard
          icon={<Activity size={19} />}
          label={t("admin.auditLogs.metrics.actions")}
          value={formatNumber(summary.actions, i18n.language)}
        />
        <AuditStatCard
          icon={<UserRound size={19} />}
          label={t("admin.auditLogs.metrics.users")}
          value={formatNumber(summary.users, i18n.language)}
        />
        <AuditStatCard
          icon={<ShieldAlert size={19} />}
          label={t("admin.auditLogs.metrics.withMetadata")}
          value={formatNumber(summary.withMetadata, i18n.language)}
        />
        <AuditStatCard
          icon={<Database size={19} />}
          label={t("admin.auditLogs.metrics.nextCursor")}
          value={summary.nextCursor ?? fallbackValue(auditLogsQuery.isLoading, t)}
        />
      </div>

      <section className="aoi-admin-panel">
        <header>
          <h2>{t("admin.auditLogs.filters.title")}</h2>
          <p>{t("admin.auditLogs.filters.description")}</p>
        </header>
        <form className="aoi-admin-filter-form" onSubmit={submitFilters}>
          <FormField
            label={t("admin.auditLogs.filters.action")}
            value={draft.action}
            onChange={(event) => updateDraft("action", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.auditLogs.filters.userId")}
            min={1}
            type="number"
            value={draft.userId}
            onChange={(event) => updateDraft("userId", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.auditLogs.filters.from")}
            type="datetime-local"
            value={draft.from}
            onChange={(event) => updateDraft("from", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.auditLogs.filters.to")}
            type="datetime-local"
            value={draft.to}
            onChange={(event) => updateDraft("to", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.auditLogs.filters.cursor")}
            min={1}
            type="number"
            value={draft.cursor}
            onChange={(event) => updateDraft("cursor", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.auditLogs.filters.limit")}
            max={maxLimit}
            min={1}
            type="number"
            value={draft.limit}
            onChange={(event) => updateDraft("limit", event.currentTarget.value)}
          />
          <div className="aoi-admin-filter-actions">
            <Button icon={<Search size={17} />} loading={auditLogsQuery.isFetching} type="submit">
              {t("admin.auditLogs.actions.search")}
            </Button>
            <Button appearance="secondary" icon={<RotateCcw size={17} />} onClick={resetFilters}>
              {t("admin.auditLogs.actions.reset")}
            </Button>
          </div>
        </form>
      </section>

      <section className="aoi-admin-panel">
        <header className="aoi-admin-panel-header-row">
          <div>
            <h2>{t("admin.auditLogs.list.title")}</h2>
            <p>
              {t("admin.auditLogs.list.description", {
                count: logs.length,
                limit: filters.limit ?? defaultLimit,
              })}
            </p>
          </div>
          <span className="aoi-iam-audit-count">
            <ChevronsDown aria-hidden="true" size={16} />
            {summary.nextCursor ?? t("common.labels.none")}
          </span>
        </header>

        {auditLogsQuery.isLoading ? (
          <StateBlock
            title={t("admin.auditLogs.states.loadingTitle")}
            description={t("admin.auditLogs.states.loadingDescription")}
          />
        ) : auditLogsQuery.data ? (
          <div className="aoi-audit-table">
            <DataTable
              columns={auditLogColumns}
              data={logs}
              emptyLabel={t("admin.auditLogs.empty")}
            />
          </div>
        ) : (
          <StateBlock
            title={t("admin.auditLogs.states.emptyTitle")}
            description={t("admin.auditLogs.states.emptyDescription")}
          />
        )}
      </section>
    </section>
  );
}

type AuditStatCardProps = {
  icon: ReactNode;
  label: string;
  value: string;
};

function AuditStatCard({ icon, label, value }: AuditStatCardProps) {
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

function normalizeFilters(draft: AuditLogFilterDraft): AuditLogFilters {
  return {
    action: trimmedOrUndefined(draft.action),
    cursor: parseID(draft.cursor),
    from: datetimeLocalToISOString(draft.from),
    limit: parseLimit(draft.limit),
    to: datetimeLocalToISOString(draft.to),
    userId: parseID(draft.userId),
  };
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

function summarizeAuditLogs(items: IAMAuditLog[]) {
  const actions = new Set<string>();
  const users = new Set<string>();
  let withMetadata = 0;
  let nextCursor: string | undefined;

  for (const item of items) {
    if (item.action) {
      actions.add(item.action);
    }
    if (item.userId !== null && item.userId !== undefined && item.userId !== "") {
      users.add(String(item.userId));
    }
    if (hasMetadata(item.metadata)) {
      withMetadata += 1;
    }
    nextCursor = String(item.id);
  }

  return {
    actions: actions.size,
    nextCursor,
    users: users.size,
    withMetadata,
  };
}

function hasMetadata(value: string | undefined) {
  const trimmed = value?.trim();
  return Boolean(trimmed && trimmed !== "{}" && trimmed !== "null");
}

function prettyMetadata(value: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (!hasMetadata(value)) {
    return t("common.labels.none");
  }
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
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
    return t("admin.auditLogs.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.auditLogs.states.errorTitle");
}

function errorDescription(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.auditLogs.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

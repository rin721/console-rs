import { useQuery } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import {
  AlertTriangle,
  Bug,
  ChevronLeft,
  ChevronRight,
  Database,
  RefreshCw,
  RotateCcw,
  Search,
  ServerCrash,
  ShieldAlert,
} from "lucide-react";
import { useMemo, useState, type FormEvent, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { DataTable } from "~/components/aoi/patterns/DataTable";
import { FormField } from "~/components/aoi/patterns/FormField";
import { SelectField, type SelectOption } from "~/components/aoi/patterns/SelectField";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { ApiError } from "~/lib/api/client";
import { queryKeys } from "~/lib/api/query-keys";
import { systemApi, type SystemOperationRecordListQuery } from "~/lib/api/system";
import type { SystemOperationRecord } from "~/lib/api/types";

const defaultPageSize = 10;
const defaultStatusClass = "5xx";

type ErrorLogFilters = Pick<
  SystemOperationRecordListQuery,
  "method" | "path" | "status" | "statusClass"
>;

type ErrorLogFilterDraft = {
  method: string;
  pageSize: string;
  path: string;
  status: string;
  statusClass: string;
};

const initialDraft: ErrorLogFilterDraft = {
  method: "",
  pageSize: String(defaultPageSize),
  path: "",
  status: "",
  statusClass: defaultStatusClass,
};

export default function AdminErrorLogsRoute() {
  const { i18n, t } = useTranslation();
  const [draft, setDraft] = useState<ErrorLogFilterDraft>(initialDraft);
  const [filters, setFilters] = useState<ErrorLogFilters>(normalizeFilters(initialDraft));
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(defaultPageSize);

  const errorLogsQuery = useQuery({
    queryFn: ({ signal }) =>
      systemApi.listOperationRecords({ ...filters, page, pageSize }, { signal }),
    queryKey: queryKeys.system.operationRecords(i18n.language, page, pageSize, filters),
  });

  const pageData = errorLogsQuery.data;
  const totalPages = Math.max(1, Math.ceil((pageData?.total ?? 0) / pageSize));
  const currentPageErrors = pageData?.items.filter((record) => record.status >= 400).length ?? 0;
  const serverErrors = pageData?.items.filter((record) => record.status >= 500).length ?? 0;
  const clientErrors =
    pageData?.items.filter((record) => record.status >= 400 && record.status < 500).length ?? 0;

  const methodOptions = useMemo<SelectOption[]>(
    () => [
      { label: t("admin.errorLogs.filters.allMethods"), value: "" },
      ...["GET", "POST", "PUT", "PATCH", "DELETE"].map((method) => ({
        label: method,
        value: method,
      })),
    ],
    [t],
  );
  const statusClassOptions = useMemo<SelectOption[]>(
    () => [
      { label: t("admin.errorLogs.statusClass.serverError"), value: "5xx" },
      { label: t("admin.errorLogs.statusClass.clientError"), value: "4xx" },
      { label: t("admin.errorLogs.statusClass.error"), value: "error" },
    ],
    [t],
  );

  const errorLogColumns = useMemo<ColumnDef<SystemOperationRecord>[]>(
    () => [
      {
        accessorKey: "status",
        cell: ({ row }) => (
          <span
            className="aoi-operation-status"
            data-status-class={statusClassFor(row.original.status)}
          >
            {row.original.status || t("common.labels.none")}
          </span>
        ),
        header: t("admin.errorLogs.columns.status"),
      },
      {
        accessorKey: "path",
        cell: ({ row }) => (
          <div className="aoi-operation-request">
            <span className="aoi-api-method" data-method={row.original.method.toLowerCase()}>
              {row.original.method.toUpperCase()}
            </span>
            <strong>{row.original.path}</strong>
          </div>
        ),
        header: t("admin.errorLogs.columns.request"),
      },
      {
        accessorKey: "username",
        cell: ({ row }) => (
          <div className="aoi-operation-user">
            <strong>{row.original.username || t("admin.errorLogs.labels.anonymous")}</strong>
            {row.original.userId ? (
              <span>
                {t("admin.errorLogs.labels.userId", {
                  id: row.original.userId,
                })}
              </span>
            ) : null}
          </div>
        ),
        header: t("admin.errorLogs.columns.user"),
      },
      {
        accessorKey: "ipAddress",
        cell: ({ row }) =>
          row.original.ipAddress ? (
            <code className="aoi-operation-code">{row.original.ipAddress}</code>
          ) : (
            t("common.labels.none")
          ),
        header: t("admin.errorLogs.columns.ip"),
      },
      {
        id: "summary",
        cell: ({ row }) => <ErrorSummary record={row.original} t={t} />,
        header: t("admin.errorLogs.columns.summary"),
      },
      {
        accessorKey: "traceId",
        cell: ({ row }) =>
          row.original.traceId ? (
            <code className="aoi-operation-code">{row.original.traceId}</code>
          ) : (
            t("common.labels.none")
          ),
        header: t("admin.errorLogs.columns.traceId"),
      },
      {
        accessorKey: "createdAt",
        cell: ({ getValue }) => formatDate(String(getValue()), i18n.language),
        header: t("admin.errorLogs.columns.createdAt"),
      },
    ],
    [i18n.language, t],
  );

  const updateDraft = (key: keyof ErrorLogFilterDraft, value: string) => {
    setDraft((current) => ({ ...current, [key]: value }));
  };

  const submitFilters = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFilters(normalizeFilters(draft));
    setPage(1);
    setPageSize(parsePageSize(draft.pageSize));
  };

  const resetFilters = () => {
    setDraft(initialDraft);
    setFilters(normalizeFilters(initialDraft));
    setPage(1);
    setPageSize(defaultPageSize);
  };

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-error-logs-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.errorLogs.badge")}</Badge>
          <h1 id="admin-error-logs-title">{t("admin.errorLogs.title")}</h1>
          <p>{t("admin.errorLogs.description")}</p>
        </div>
        <Button
          appearance="secondary"
          icon={<RefreshCw size={17} />}
          loading={errorLogsQuery.isFetching}
          onClick={() => void errorLogsQuery.refetch()}
        >
          {t("admin.errorLogs.actions.refresh")}
        </Button>
      </div>

      {errorLogsQuery.error ? (
        <StateBlock
          intent="danger"
          title={errorTitle(errorLogsQuery.error, t)}
          description={errorDescription(errorLogsQuery.error, t)}
        />
      ) : null}

      <div className="aoi-admin-stat-grid" aria-label={t("admin.errorLogs.summaryLabel")}>
        <ErrorLogStatCard
          icon={<Bug size={19} />}
          label={t("admin.errorLogs.metrics.total")}
          value={
            pageData
              ? formatNumber(pageData.total, i18n.language)
              : fallbackValue(errorLogsQuery.isLoading, t)
          }
        />
        <ErrorLogStatCard
          icon={<AlertTriangle size={19} />}
          label={t("admin.errorLogs.metrics.currentPageErrors")}
          value={formatNumber(currentPageErrors, i18n.language)}
        />
        <ErrorLogStatCard
          icon={<ServerCrash size={19} />}
          label={t("admin.errorLogs.metrics.serverErrors")}
          value={formatNumber(serverErrors, i18n.language)}
        />
        <ErrorLogStatCard
          icon={<ShieldAlert size={19} />}
          label={t("admin.errorLogs.metrics.clientErrors")}
          value={formatNumber(clientErrors, i18n.language)}
        />
        <ErrorLogStatCard
          icon={<Database size={19} />}
          label={t("admin.errorLogs.metrics.storage")}
          value={
            pageData
              ? storageStatusLabel(pageData.storageStatus, t)
              : fallbackValue(errorLogsQuery.isLoading, t)
          }
        />
      </div>

      <section className="aoi-admin-panel">
        <header>
          <h2>{t("admin.errorLogs.filters.title")}</h2>
          <p>{t("admin.errorLogs.filters.description")}</p>
        </header>
        <form className="aoi-admin-filter-form" onSubmit={submitFilters}>
          <SelectField
            label={t("admin.errorLogs.filters.statusClass")}
            options={statusClassOptions}
            value={draft.statusClass}
            onChange={(event) => updateDraft("statusClass", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.errorLogs.filters.status")}
            max={599}
            min={100}
            type="number"
            value={draft.status}
            onChange={(event) => updateDraft("status", event.currentTarget.value)}
          />
          <SelectField
            label={t("admin.errorLogs.filters.method")}
            options={methodOptions}
            value={draft.method}
            onChange={(event) => updateDraft("method", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.errorLogs.filters.path")}
            value={draft.path}
            onChange={(event) => updateDraft("path", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.errorLogs.filters.pageSize")}
            max={100}
            min={1}
            type="number"
            value={draft.pageSize}
            onChange={(event) => updateDraft("pageSize", event.currentTarget.value)}
          />
          <div className="aoi-admin-filter-actions">
            <Button icon={<Search size={17} />} loading={errorLogsQuery.isFetching} type="submit">
              {t("admin.errorLogs.actions.search")}
            </Button>
            <Button appearance="secondary" icon={<RotateCcw size={17} />} onClick={resetFilters}>
              {t("admin.errorLogs.actions.reset")}
            </Button>
          </div>
        </form>
      </section>

      <section className="aoi-admin-panel">
        <header className="aoi-admin-panel-header-row">
          <div>
            <h2>{t("admin.errorLogs.list.title")}</h2>
            <p>
              {t("admin.errorLogs.list.description", {
                count: pageData?.total ?? 0,
              })}
            </p>
          </div>
          <div className="aoi-admin-pager" aria-label={t("admin.errorLogs.pagination.label")}>
            <Button
              appearance="secondary"
              disabled={page <= 1 || errorLogsQuery.isFetching}
              icon={<ChevronLeft size={17} />}
              onClick={() => setPage((current) => Math.max(1, current - 1))}
            >
              {t("admin.errorLogs.pagination.previous")}
            </Button>
            <span>
              {t("admin.errorLogs.pagination.pageStatus", {
                page,
                totalPages,
              })}
            </span>
            <Button
              appearance="secondary"
              disabled={page >= totalPages || errorLogsQuery.isFetching}
              icon={<ChevronRight size={17} />}
              onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
            >
              {t("admin.errorLogs.pagination.next")}
            </Button>
          </div>
        </header>

        {errorLogsQuery.isLoading ? (
          <StateBlock
            title={t("admin.errorLogs.states.loadingTitle")}
            description={t("admin.errorLogs.states.loadingDescription")}
          />
        ) : pageData ? (
          <>
            {pageData.storageStatus === "persisted" ? null : (
              <StateBlock
                title={t("admin.errorLogs.states.storageUnavailableTitle")}
                description={t("admin.errorLogs.states.storageUnavailableDescription")}
              />
            )}
            <div className="aoi-error-log-table">
              <DataTable
                columns={errorLogColumns}
                data={pageData.items}
                emptyLabel={t("admin.errorLogs.empty")}
              />
            </div>
          </>
        ) : (
          <StateBlock
            title={t("admin.errorLogs.states.emptyTitle")}
            description={t("admin.errorLogs.states.emptyDescription")}
          />
        )}
      </section>
    </section>
  );
}

function ErrorSummary({
  record,
  t,
}: {
  record: SystemOperationRecord;
  t: ReturnType<typeof useTranslation>["t"];
}) {
  const value = record.errorMessage || record.response || record.body;

  if (!value) {
    return <span className="aoi-operation-muted">{t("admin.errorLogs.labels.noSummary")}</span>;
  }

  return <code className="aoi-error-log-summary">{previewPayload(value)}</code>;
}

type ErrorLogStatCardProps = {
  icon: ReactNode;
  label: string;
  value: string;
};

function ErrorLogStatCard({ icon, label, value }: ErrorLogStatCardProps) {
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

function normalizeFilters(draft: ErrorLogFilterDraft): ErrorLogFilters {
  const status = parseStatus(draft.status);
  return {
    method: trimmedOrUndefined(draft.method)?.toUpperCase(),
    path: trimmedOrUndefined(draft.path),
    status,
    statusClass: status ? undefined : trimmedOrUndefined(draft.statusClass) || defaultStatusClass,
  };
}

function trimmedOrUndefined(value: string | undefined) {
  const trimmed = value?.trim();
  return trimmed || undefined;
}

function parseStatus(value: string) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return undefined;
  }
  return Math.trunc(parsed);
}

function parsePageSize(value: string) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return defaultPageSize;
  }
  return Math.min(100, Math.max(1, Math.trunc(parsed)));
}

function statusClassFor(status: number) {
  if (status >= 500) {
    return "5xx";
  }
  if (status >= 400) {
    return "4xx";
  }
  return "other";
}

function previewPayload(value: string) {
  const compact = value.trim().replace(/\s+/g, " ");
  if (compact.length <= 220) {
    return compact;
  }
  return `${compact.slice(0, 220)}...`;
}

function fallbackValue(loading: boolean, t: ReturnType<typeof useTranslation>["t"]) {
  return loading ? t("loading.app") : t("common.labels.none");
}

function storageStatusLabel(status: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (status === "persisted") {
    return t("admin.errorLogs.storage.persisted");
  }
  if (status === "unavailable") {
    return t("admin.errorLogs.storage.unavailable");
  }
  return status || t("admin.errorLogs.storage.unknown");
}

function formatNumber(value: number, locale: string) {
  return new Intl.NumberFormat(locale).format(value);
}

function formatDate(value: string, locale: string) {
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
    return t("admin.errorLogs.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.errorLogs.states.errorTitle");
}

function errorDescription(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.errorLogs.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import {
  AlertTriangle,
  ChevronLeft,
  ChevronRight,
  Database,
  Gauge,
  History,
  ListChecks,
  RefreshCw,
  RotateCcw,
  Search,
  TimerReset,
  Trash2,
} from "lucide-react";
import { useCallback, useMemo, useState, type FormEvent, type ReactNode } from "react";
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

type OperationRecordFilters = Pick<
  SystemOperationRecordListQuery,
  "method" | "path" | "status" | "statusClass"
>;

type OperationRecordFilterDraft = {
  method: string;
  pageSize: string;
  path: string;
  status: string;
  statusClass: string;
};

type OperationRecordNotice = {
  description: string;
  intent?: "danger" | "info";
  title: string;
};

const initialDraft: OperationRecordFilterDraft = {
  method: "",
  pageSize: String(defaultPageSize),
  path: "",
  status: "",
  statusClass: "",
};

const emptyOperationRecords: SystemOperationRecord[] = [];

export default function AdminOperationRecordsRoute() {
  const { i18n, t } = useTranslation();
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<OperationRecordFilterDraft>(initialDraft);
  const [filters, setFilters] = useState<OperationRecordFilters>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(defaultPageSize);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [pendingDeleteIds, setPendingDeleteIds] = useState<string[] | null>(null);
  const [notice, setNotice] = useState<OperationRecordNotice | null>(null);

  const operationRecordsQueryKey = queryKeys.system.operationRecords(
    i18n.language,
    page,
    pageSize,
    filters,
  );

  const operationRecordsQuery = useQuery({
    queryFn: ({ signal }) =>
      systemApi.listOperationRecords({ ...filters, page, pageSize }, { signal }),
    queryKey: operationRecordsQueryKey,
  });

  const invalidateOperationRecords = () =>
    queryClient.invalidateQueries({ queryKey: ["system", "operation-records"] });

  const deleteOperationRecordsMutation = useMutation({
    mutationFn: (ids: string[]) => systemApi.deleteOperationRecords(ids),
    onError: (error) => {
      setNotice({
        description: deleteErrorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.operationRecords.messages.deleteFailedTitle"),
      });
    },
    onSuccess: (_result, ids) => {
      setPendingDeleteIds(null);
      setSelectedIds((current) => current.filter((id) => !ids.includes(id)));
      setNotice({
        description: t("admin.operationRecords.messages.deletedSelectedDescription", {
          count: ids.length,
        }),
        title: t("admin.operationRecords.messages.deletedSelectedTitle"),
      });
      void invalidateOperationRecords();
    },
  });

  const pageData = operationRecordsQuery.data;
  const operationRecordItems = pageData?.items ?? emptyOperationRecords;
  const totalPages = Math.max(1, Math.ceil((pageData?.total ?? 0) / pageSize));
  const currentPageCount = operationRecordItems.length;
  const currentPageErrors = operationRecordItems.filter((record) => record.status >= 400).length;
  const slowestLatency = maxLatency(operationRecordItems);
  const storagePersisted = pageData?.storageStatus === "persisted";
  const writePending = deleteOperationRecordsMutation.isPending;
  const visibleRecordIds = useMemo(
    () => new Set(operationRecordItems.map(operationRecordIdValue)),
    [operationRecordItems],
  );
  const visibleSelectedIds = useMemo(
    () => selectedIds.filter((id) => visibleRecordIds.has(id)),
    [selectedIds, visibleRecordIds],
  );
  const selectedSet = useMemo(() => new Set(visibleSelectedIds), [visibleSelectedIds]);
  const allVisibleSelected =
    operationRecordItems.length > 0 &&
    operationRecordItems.every((record) => selectedSet.has(operationRecordIdValue(record)));

  const methodOptions = useMemo<SelectOption[]>(
    () => [
      { label: t("admin.operationRecords.filters.allMethods"), value: "" },
      ...["GET", "POST", "PUT", "PATCH", "DELETE"].map((method) => ({
        label: method,
        value: method,
      })),
    ],
    [t],
  );
  const statusClassOptions = useMemo<SelectOption[]>(
    () => [
      { label: t("admin.operationRecords.statusClass.all"), value: "" },
      { label: t("admin.operationRecords.statusClass.clientError"), value: "4xx" },
      { label: t("admin.operationRecords.statusClass.serverError"), value: "5xx" },
      { label: t("admin.operationRecords.statusClass.error"), value: "error" },
    ],
    [t],
  );

  const toggleRecordSelection = useCallback((record: SystemOperationRecord, checked: boolean) => {
    const id = operationRecordIdValue(record);
    setSelectedIds((current) => {
      if (checked) {
        return current.includes(id) ? current : [...current, id];
      }
      return current.filter((selectedId) => selectedId !== id);
    });
  }, []);

  const toggleAllVisible = useCallback(
    (checked: boolean) => {
      setSelectedIds(checked ? operationRecordItems.map(operationRecordIdValue) : []);
    },
    [operationRecordItems],
  );

  const operationRecordColumns = useMemo<ColumnDef<SystemOperationRecord>[]>(
    () => [
      {
        id: "selection",
        cell: ({ row }) => {
          const id = operationRecordIdValue(row.original);
          return (
            <input
              aria-label={t("admin.operationRecords.selection.rowAria", { id })}
              checked={selectedSet.has(id)}
              className="aoi-operation-check"
              disabled={!storagePersisted || writePending}
              type="checkbox"
              onChange={(event) => toggleRecordSelection(row.original, event.currentTarget.checked)}
            />
          );
        },
        header: () => (
          <input
            aria-label={t("admin.operationRecords.selection.allAria")}
            checked={allVisibleSelected}
            className="aoi-operation-check"
            disabled={!operationRecordItems.length || !storagePersisted || writePending}
            type="checkbox"
            onChange={(event) => toggleAllVisible(event.currentTarget.checked)}
          />
        ),
      },
      {
        accessorKey: "path",
        cell: ({ row }) => (
          <div className="aoi-operation-request">
            <span className="aoi-api-method" data-method={row.original.method.toLowerCase()}>
              {row.original.method.toUpperCase()}
            </span>
            <strong>{row.original.path}</strong>
            <span>{row.original.id}</span>
          </div>
        ),
        header: t("admin.operationRecords.columns.request"),
      },
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
        header: t("admin.operationRecords.columns.status"),
      },
      {
        accessorKey: "username",
        cell: ({ row }) => (
          <div className="aoi-operation-user">
            <strong>{row.original.username || t("admin.operationRecords.labels.anonymous")}</strong>
            {row.original.userId ? (
              <span>
                {t("admin.operationRecords.labels.userId", {
                  id: row.original.userId,
                })}
              </span>
            ) : null}
          </div>
        ),
        header: t("admin.operationRecords.columns.user"),
      },
      {
        accessorKey: "ipAddress",
        cell: ({ row }) => {
          return row.original.ipAddress ? (
            <code className="aoi-operation-code">{row.original.ipAddress}</code>
          ) : (
            t("common.labels.none")
          );
        },
        header: t("admin.operationRecords.columns.ip"),
      },
      {
        accessorKey: "latencyMs",
        cell: ({ row }) => formatDuration(row.original.latencyMs, i18n.language, t),
        header: t("admin.operationRecords.columns.latency"),
      },
      {
        accessorKey: "traceId",
        cell: ({ row }) => {
          return row.original.traceId ? (
            <code className="aoi-operation-code">{row.original.traceId}</code>
          ) : (
            t("common.labels.none")
          );
        },
        header: t("admin.operationRecords.columns.traceId"),
      },
      {
        id: "payload",
        cell: ({ row }) => <OperationPayload record={row.original} t={t} />,
        header: t("admin.operationRecords.columns.payload"),
      },
      {
        accessorKey: "createdAt",
        cell: ({ getValue }) => formatDate(String(getValue()), i18n.language),
        header: t("admin.operationRecords.columns.createdAt"),
      },
    ],
    [
      allVisibleSelected,
      i18n.language,
      operationRecordItems.length,
      selectedSet,
      storagePersisted,
      t,
      toggleAllVisible,
      toggleRecordSelection,
      writePending,
    ],
  );

  const updateDraft = (key: keyof OperationRecordFilterDraft, value: string) => {
    setDraft((current) => ({ ...current, [key]: value }));
  };

  const submitFilters = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFilters(normalizeFilters(draft));
    setPage(1);
    setPageSize(parsePageSize(draft.pageSize));
    setPendingDeleteIds(null);
    setSelectedIds([]);
  };

  const resetFilters = () => {
    setDraft(initialDraft);
    setFilters({});
    setPage(1);
    setPageSize(defaultPageSize);
    setPendingDeleteIds(null);
    setSelectedIds([]);
  };

  const openBulkDelete = () => {
    if (!visibleSelectedIds.length) {
      return;
    }
    setPendingDeleteIds(visibleSelectedIds);
  };

  const confirmPendingDelete = () => {
    if (!pendingDeleteIds || !storagePersisted) {
      return;
    }
    setNotice(null);
    deleteOperationRecordsMutation.mutate(pendingDeleteIds);
  };

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-operation-records-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.operationRecords.badge")}</Badge>
          <h1 id="admin-operation-records-title">{t("admin.operationRecords.title")}</h1>
          <p>{t("admin.operationRecords.description")}</p>
        </div>
        <div className="aoi-operation-page-actions">
          <Button
            appearance="secondary"
            disabled={!visibleSelectedIds.length || !storagePersisted || writePending}
            icon={<Trash2 size={17} />}
            onClick={openBulkDelete}
          >
            {t("admin.operationRecords.actions.deleteSelected")}
          </Button>
          <Button
            appearance="secondary"
            icon={<RefreshCw size={17} />}
            loading={operationRecordsQuery.isFetching}
            onClick={() => void operationRecordsQuery.refetch()}
          >
            {t("admin.operationRecords.actions.refresh")}
          </Button>
        </div>
      </div>

      {operationRecordsQuery.error ? (
        <StateBlock
          intent="danger"
          title={errorTitle(operationRecordsQuery.error, t)}
          description={errorDescription(operationRecordsQuery.error, t)}
        />
      ) : null}

      {notice ? (
        <StateBlock description={notice.description} intent={notice.intent} title={notice.title} />
      ) : null}

      {pendingDeleteIds ? (
        <StateBlock
          action={
            <div className="aoi-operation-confirm-actions">
              <Button loading={writePending} onClick={confirmPendingDelete}>
                {t("admin.operationRecords.actions.confirmDelete")}
              </Button>
              <Button
                appearance="secondary"
                disabled={writePending}
                onClick={() => setPendingDeleteIds(null)}
              >
                {t("admin.operationRecords.actions.cancel")}
              </Button>
            </div>
          }
          description={t("admin.operationRecords.delete.bulkDescription", {
            count: pendingDeleteIds.length,
          })}
          title={t("admin.operationRecords.delete.bulkTitle")}
        />
      ) : null}

      <div className="aoi-admin-stat-grid" aria-label={t("admin.operationRecords.summaryLabel")}>
        <OperationRecordStatCard
          icon={<History size={19} />}
          label={t("admin.operationRecords.metrics.total")}
          value={
            pageData
              ? formatNumber(pageData.total, i18n.language)
              : fallbackValue(operationRecordsQuery.isLoading, t)
          }
        />
        <OperationRecordStatCard
          icon={<Gauge size={19} />}
          label={t("admin.operationRecords.metrics.currentPage")}
          value={formatNumber(currentPageCount, i18n.language)}
        />
        <OperationRecordStatCard
          icon={<AlertTriangle size={19} />}
          label={t("admin.operationRecords.metrics.errors")}
          value={formatNumber(currentPageErrors, i18n.language)}
        />
        <OperationRecordStatCard
          icon={<TimerReset size={19} />}
          label={t("admin.operationRecords.metrics.slowest")}
          value={formatDuration(slowestLatency, i18n.language, t)}
        />
        <OperationRecordStatCard
          icon={<ListChecks size={19} />}
          label={t("admin.operationRecords.metrics.selected")}
          value={formatNumber(visibleSelectedIds.length, i18n.language)}
        />
        <OperationRecordStatCard
          icon={<Database size={19} />}
          label={t("admin.operationRecords.metrics.storage")}
          value={
            pageData
              ? storageStatusLabel(pageData.storageStatus, t)
              : fallbackValue(operationRecordsQuery.isLoading, t)
          }
        />
      </div>

      <section className="aoi-admin-panel">
        <header>
          <h2>{t("admin.operationRecords.filters.title")}</h2>
          <p>{t("admin.operationRecords.filters.description")}</p>
        </header>
        <form className="aoi-admin-filter-form" onSubmit={submitFilters}>
          <SelectField
            label={t("admin.operationRecords.filters.method")}
            options={methodOptions}
            value={draft.method}
            onChange={(event) => updateDraft("method", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.operationRecords.filters.path")}
            value={draft.path}
            onChange={(event) => updateDraft("path", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.operationRecords.filters.status")}
            max={599}
            min={100}
            type="number"
            value={draft.status}
            onChange={(event) => updateDraft("status", event.currentTarget.value)}
          />
          <SelectField
            label={t("admin.operationRecords.filters.statusClass")}
            options={statusClassOptions}
            value={draft.statusClass}
            onChange={(event) => updateDraft("statusClass", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.operationRecords.filters.pageSize")}
            max={100}
            min={1}
            type="number"
            value={draft.pageSize}
            onChange={(event) => updateDraft("pageSize", event.currentTarget.value)}
          />
          <div className="aoi-admin-filter-actions">
            <Button
              icon={<Search size={17} />}
              loading={operationRecordsQuery.isFetching}
              type="submit"
            >
              {t("admin.operationRecords.actions.search")}
            </Button>
            <Button appearance="secondary" icon={<RotateCcw size={17} />} onClick={resetFilters}>
              {t("admin.operationRecords.actions.reset")}
            </Button>
          </div>
        </form>
      </section>

      <section className="aoi-admin-panel">
        <header className="aoi-admin-panel-header-row">
          <div>
            <h2>{t("admin.operationRecords.list.title")}</h2>
            <p>
              {t("admin.operationRecords.list.description", {
                count: pageData?.total ?? 0,
              })}
            </p>
          </div>
          <div
            className="aoi-admin-pager"
            aria-label={t("admin.operationRecords.pagination.label")}
          >
            <Button
              appearance="secondary"
              disabled={page <= 1 || operationRecordsQuery.isFetching}
              icon={<ChevronLeft size={17} />}
              onClick={() => setPage((current) => Math.max(1, current - 1))}
            >
              {t("admin.operationRecords.pagination.previous")}
            </Button>
            <span>
              {t("admin.operationRecords.pagination.pageStatus", {
                page,
                totalPages,
              })}
            </span>
            <Button
              appearance="secondary"
              disabled={page >= totalPages || operationRecordsQuery.isFetching}
              icon={<ChevronRight size={17} />}
              onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
            >
              {t("admin.operationRecords.pagination.next")}
            </Button>
          </div>
        </header>

        {operationRecordsQuery.isLoading ? (
          <StateBlock
            title={t("admin.operationRecords.states.loadingTitle")}
            description={t("admin.operationRecords.states.loadingDescription")}
          />
        ) : pageData ? (
          <>
            {pageData.storageStatus === "persisted" ? null : (
              <StateBlock
                title={t("admin.operationRecords.states.storageUnavailableTitle")}
                description={t("admin.operationRecords.states.storageUnavailableDescription")}
              />
            )}
            <div className="aoi-operation-table">
              <DataTable
                columns={operationRecordColumns}
                data={operationRecordItems}
                emptyLabel={t("admin.operationRecords.empty")}
              />
            </div>
          </>
        ) : (
          <StateBlock
            title={t("admin.operationRecords.states.emptyTitle")}
            description={t("admin.operationRecords.states.emptyDescription")}
          />
        )}
      </section>
    </section>
  );
}

function operationRecordIdValue(record: SystemOperationRecord) {
  return String(record.id);
}

function OperationPayload({
  record,
  t,
}: {
  record: SystemOperationRecord;
  t: ReturnType<typeof useTranslation>["t"];
}) {
  const items = [
    {
      label: t("admin.operationRecords.labels.requestBody"),
      value: record.body,
    },
    {
      label: record.errorMessage
        ? t("admin.operationRecords.labels.errorMessage")
        : t("admin.operationRecords.labels.responseBody"),
      value: record.errorMessage || record.response,
    },
  ].filter((item) => Boolean(item.value));

  if (items.length === 0) {
    return (
      <span className="aoi-operation-muted">{t("admin.operationRecords.labels.noPayload")}</span>
    );
  }

  return (
    <div className="aoi-operation-payload">
      {items.map((item) => (
        <div key={item.label}>
          <span>{item.label}</span>
          <code>{previewPayload(item.value)}</code>
        </div>
      ))}
    </div>
  );
}

type OperationRecordStatCardProps = {
  icon: ReactNode;
  label: string;
  value: string;
};

function OperationRecordStatCard({ icon, label, value }: OperationRecordStatCardProps) {
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

function normalizeFilters(draft: OperationRecordFilterDraft): OperationRecordFilters {
  const status = parseStatus(draft.status);
  return {
    method: trimmedOrUndefined(draft.method)?.toUpperCase(),
    path: trimmedOrUndefined(draft.path),
    status,
    statusClass: status ? undefined : trimmedOrUndefined(draft.statusClass),
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

function maxLatency(records: SystemOperationRecord[]) {
  return records.reduce((current, record) => Math.max(current, record.latencyMs || 0), 0);
}

function statusClassFor(status: number) {
  if (status >= 500) {
    return "5xx";
  }
  if (status >= 400) {
    return "4xx";
  }
  if (status >= 300) {
    return "3xx";
  }
  if (status >= 200) {
    return "2xx";
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
    return t("admin.operationRecords.storage.persisted");
  }
  if (status === "unavailable") {
    return t("admin.operationRecords.storage.unavailable");
  }
  return status || t("admin.operationRecords.storage.unknown");
}

function formatNumber(value: number, locale: string) {
  return new Intl.NumberFormat(locale).format(value);
}

function formatDuration(value: number, locale: string, t: ReturnType<typeof useTranslation>["t"]) {
  return t("admin.operationRecords.labels.durationMs", {
    value: formatNumber(value, locale),
  });
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

function toError(error: unknown) {
  return error instanceof Error ? error : new Error(String(error));
}

function errorTitle(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.operationRecords.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.operationRecords.states.errorTitle");
}

function errorDescription(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.operationRecords.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

function deleteErrorDescription(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.operationRecords.states.deletePermissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

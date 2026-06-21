import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import {
  Code2,
  KeyRound,
  ListFilter,
  RefreshCw,
  RotateCcw,
  Search,
  ShieldCheck,
} from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { DataTable } from "~/components/aoi/patterns/DataTable";
import { FormField } from "~/components/aoi/patterns/FormField";
import { SelectField, type SelectOption } from "~/components/aoi/patterns/SelectField";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { ApiError } from "~/lib/api/client";
import { queryKeys } from "~/lib/api/query-keys";
import { systemApi } from "~/lib/api/system";
import type {
  SystemAPIEntry,
  SystemAPIGroup,
  SystemAPISyncResult,
  SystemPermissionSyncResult,
} from "~/lib/api/types";

type APIEntryRow = SystemAPIEntry & {
  groupCode: string;
  groupLabel: string;
};

type APIFilters = {
  accessMode: string;
  groupCode: string;
  keyword: string;
  method: string;
};

type SyncAction = "permissions" | "routes";

type SyncNotice = {
  description: string;
  intent?: "danger" | "info";
  title: string;
};

const emptyFilters: APIFilters = {
  accessMode: "",
  groupCode: "",
  keyword: "",
  method: "",
};

const accessModes = ["public", "authenticated", "permission"] as const;

export default function AdminAPIsRoute() {
  const { i18n, t } = useTranslation();
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState<APIFilters>(emptyFilters);
  const [pendingSyncAction, setPendingSyncAction] = useState<SyncAction | null>(null);
  const [syncNotice, setSyncNotice] = useState<SyncNotice | null>(null);

  const apiCatalogQuery = useQuery({
    queryFn: ({ signal }) => systemApi.listAPIs({ signal }),
    queryKey: queryKeys.system.apiCatalog(i18n.language),
  });

  const syncRoutesMutation = useMutation({
    mutationFn: () => systemApi.syncAPIs(),
    onError: (error) => {
      setSyncNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.apis.sync.errorTitle"),
      });
    },
    onMutate: () => {
      setPendingSyncAction(null);
      setSyncNotice(null);
    },
    onSuccess: (result) => {
      queryClient.setQueryData(queryKeys.system.apiCatalog(i18n.language), result.groups);
      setSyncNotice({
        description: apiSyncDescription(result, t),
        title: t("admin.apis.sync.routesSuccessTitle"),
      });
    },
  });

  const syncPermissionsMutation = useMutation({
    mutationFn: () => systemApi.syncAPIPermissions(),
    onError: (error) => {
      setSyncNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.apis.sync.errorTitle"),
      });
    },
    onMutate: () => {
      setPendingSyncAction(null);
      setSyncNotice(null);
    },
    onSuccess: async (result) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.system.apiCatalog(i18n.language) });
      setSyncNotice({
        description: permissionSyncDescription(result, t),
        title: t("admin.apis.sync.permissionsSuccessTitle"),
      });
    },
  });

  const groups = useMemo(() => apiCatalogQuery.data ?? [], [apiCatalogQuery.data]);
  const rows = useMemo(() => flattenAPIs(groups), [groups]);
  const summary = useMemo(() => summarizeAPIRows(rows), [rows]);
  const groupOptions = useMemo(
    () => toGroupOptions(groups, t("admin.apis.filters.allGroups")),
    [groups, t],
  );
  const methodOptions = useMemo(
    () => toMethodOptions(rows, t("admin.apis.filters.allMethods")),
    [rows, t],
  );
  const accessOptions = useMemo(
    () => [
      { label: t("admin.apis.filters.allAccessModes"), value: "" },
      ...accessModes.map((mode) => ({ label: accessLabel(mode, t), value: mode })),
    ],
    [t],
  );
  const filteredRows = useMemo(
    () => rows.filter((row) => matchesFilters(row, filters, t)),
    [filters, rows, t],
  );

  const apiColumns = useMemo<ColumnDef<APIEntryRow>[]>(
    () => [
      {
        accessorKey: "method",
        cell: ({ getValue }) => (
          <span className="aoi-api-method" data-method={String(getValue()).toLowerCase()}>
            {String(getValue()).toUpperCase()}
          </span>
        ),
        header: t("admin.apis.columns.method"),
      },
      {
        accessorKey: "path",
        cell: ({ row }) => (
          <div className="aoi-api-path">
            <strong>{row.original.path}</strong>
            <span>{row.original.code}</span>
          </div>
        ),
        header: t("admin.apis.columns.path"),
      },
      {
        accessorKey: "groupLabel",
        cell: ({ row }) => (
          <div className="aoi-api-group">
            <strong>{row.original.groupLabel || row.original.groupCode}</strong>
            <span>{row.original.groupCode}</span>
          </div>
        ),
        header: t("admin.apis.columns.group"),
      },
      {
        accessorKey: "access",
        cell: ({ getValue }) => {
          const access = String(getValue());
          return (
            <span className="aoi-api-access" data-access={access}>
              {accessLabel(access, t)}
            </span>
          );
        },
        header: t("admin.apis.columns.access"),
      },
      {
        accessorKey: "permission",
        cell: ({ row }) => (
          <span className="aoi-api-permission">
            {row.original.permission || t("admin.apis.status.noPermissionCode")}
          </span>
        ),
        header: t("admin.apis.columns.permission"),
      },
      {
        id: "registration",
        cell: ({ row }) => (
          <span className="aoi-api-status" data-status={registrationStatus(row.original)}>
            {registrationLabel(row.original, t)}
          </span>
        ),
        header: t("admin.apis.columns.registration"),
      },
      {
        accessorKey: "synced",
        cell: ({ row }) => (
          <div className="aoi-api-sync">
            <span
              className="aoi-api-status"
              data-status={row.original.synced ? "synced" : "unsynced"}
            >
              {row.original.synced
                ? t("admin.apis.status.synced")
                : t("admin.apis.status.unsynced")}
            </span>
            {row.original.syncedAt ? (
              <span>{formatDate(row.original.syncedAt, i18n.language)}</span>
            ) : null}
          </div>
        ),
        header: t("admin.apis.columns.sync"),
      },
      {
        accessorKey: "description",
        cell: ({ getValue }) => {
          const value = getValue();
          return typeof value === "string" && value ? value : t("common.labels.none");
        },
        header: t("admin.apis.columns.description"),
      },
    ],
    [i18n.language, t],
  );

  const updateFilter = (key: keyof APIFilters, value: string) => {
    setFilters((current) => ({ ...current, [key]: value }));
  };

  const resetFilters = () => {
    setFilters(emptyFilters);
  };

  const syncBusy = syncRoutesMutation.isPending || syncPermissionsMutation.isPending;

  const confirmSync = () => {
    if (pendingSyncAction === "routes") {
      syncRoutesMutation.mutate();
      return;
    }
    if (pendingSyncAction === "permissions") {
      syncPermissionsMutation.mutate();
    }
  };

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-apis-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.apis.badge")}</Badge>
          <h1 id="admin-apis-title">{t("admin.apis.title")}</h1>
          <p>{t("admin.apis.description")}</p>
        </div>
        <div className="aoi-admin-action-row">
          <Button
            appearance="secondary"
            icon={<RefreshCw size={17} />}
            loading={apiCatalogQuery.isFetching}
            onClick={() => void apiCatalogQuery.refetch()}
          >
            {t("admin.apis.actions.refresh")}
          </Button>
          <Button
            appearance="secondary"
            disabled={syncBusy}
            icon={<RefreshCw size={17} />}
            onClick={() => setPendingSyncAction("routes")}
          >
            {t("admin.apis.actions.syncRoutes")}
          </Button>
          <Button
            appearance="secondary"
            disabled={syncBusy}
            icon={<ShieldCheck size={17} />}
            onClick={() => setPendingSyncAction("permissions")}
          >
            {t("admin.apis.actions.syncPermissions")}
          </Button>
        </div>
      </div>

      {apiCatalogQuery.error ? (
        <StateBlock
          intent="danger"
          title={errorTitle(apiCatalogQuery.error, t)}
          description={errorDescription(apiCatalogQuery.error, t)}
        />
      ) : null}
      {syncNotice ? (
        <StateBlock
          description={syncNotice.description}
          intent={syncNotice.intent}
          title={syncNotice.title}
        />
      ) : null}
      {pendingSyncAction ? (
        <StateBlock
          action={
            <div className="aoi-api-confirm-actions">
              <Button loading={syncBusy} onClick={confirmSync}>
                {t("admin.apis.sync.confirm")}
              </Button>
              <Button
                appearance="ghost"
                disabled={syncBusy}
                onClick={() => setPendingSyncAction(null)}
              >
                {t("admin.apis.sync.cancel")}
              </Button>
            </div>
          }
          description={syncConfirmDescription(pendingSyncAction, t)}
          title={syncConfirmTitle(pendingSyncAction, t)}
        />
      ) : null}

      <div className="aoi-admin-stat-grid" aria-label={t("admin.apis.summaryLabel")}>
        <APIStatCard
          icon={<Code2 size={19} />}
          label={t("admin.apis.metrics.total")}
          value={formatNumber(summary.total, i18n.language)}
        />
        <APIStatCard
          icon={<KeyRound size={19} />}
          label={t("admin.apis.metrics.permission")}
          value={formatNumber(summary.permission, i18n.language)}
        />
        <APIStatCard
          icon={<ShieldCheck size={19} />}
          label={t("admin.apis.metrics.authenticated")}
          value={formatNumber(summary.authenticated, i18n.language)}
        />
        <APIStatCard
          icon={<ListFilter size={19} />}
          label={t("admin.apis.metrics.public")}
          value={formatNumber(summary.public, i18n.language)}
        />
        <APIStatCard
          icon={<ShieldCheck size={19} />}
          label={t("admin.apis.metrics.synced")}
          value={formatNumber(summary.synced, i18n.language)}
        />
      </div>

      <section className="aoi-admin-panel">
        <header>
          <h2>{t("admin.apis.filters.title")}</h2>
          <p>{t("admin.apis.filters.description")}</p>
        </header>
        <form className="aoi-admin-filter-form" onSubmit={(event) => event.preventDefault()}>
          <FormField
            label={t("admin.apis.filters.keyword")}
            value={filters.keyword}
            onChange={(event) => updateFilter("keyword", event.currentTarget.value)}
          />
          <SelectField
            label={t("admin.apis.filters.group")}
            options={groupOptions}
            value={filters.groupCode}
            onChange={(event) => updateFilter("groupCode", event.currentTarget.value)}
          />
          <SelectField
            label={t("admin.apis.filters.method")}
            options={methodOptions}
            value={filters.method}
            onChange={(event) => updateFilter("method", event.currentTarget.value)}
          />
          <SelectField
            label={t("admin.apis.filters.accessMode")}
            options={accessOptions}
            value={filters.accessMode}
            onChange={(event) => updateFilter("accessMode", event.currentTarget.value)}
          />
          <div className="aoi-admin-filter-actions">
            <Button appearance="secondary" icon={<RotateCcw size={17} />} onClick={resetFilters}>
              {t("admin.apis.actions.reset")}
            </Button>
          </div>
        </form>
      </section>

      <section className="aoi-admin-panel">
        <header className="aoi-admin-panel-header-row">
          <div>
            <h2>{t("admin.apis.list.title")}</h2>
            <p>
              {t("admin.apis.list.description", {
                count: filteredRows.length,
                total: rows.length,
              })}
            </p>
          </div>
          <span className="aoi-api-count">
            <Search aria-hidden="true" size={16} />
            {formatNumber(filteredRows.length, i18n.language)}
          </span>
        </header>

        {apiCatalogQuery.isLoading ? (
          <StateBlock
            title={t("admin.apis.states.loadingTitle")}
            description={t("admin.apis.states.loadingDescription")}
          />
        ) : apiCatalogQuery.data ? (
          <div className="aoi-api-table">
            <DataTable
              columns={apiColumns}
              data={filteredRows}
              emptyLabel={t("admin.apis.empty")}
            />
          </div>
        ) : (
          <StateBlock
            title={t("admin.apis.states.emptyTitle")}
            description={t("admin.apis.states.emptyDescription")}
          />
        )}
      </section>
    </section>
  );
}

type APIStatCardProps = {
  icon: ReactNode;
  label: string;
  value: string;
};

function APIStatCard({ icon, label, value }: APIStatCardProps) {
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

function flattenAPIs(groups: SystemAPIGroup[]): APIEntryRow[] {
  return groups.flatMap((group) =>
    group.items.map((item) => ({
      ...item,
      groupCode: group.code || item.group,
      groupLabel: group.label || group.code || item.group,
    })),
  );
}

function summarizeAPIRows(rows: APIEntryRow[]) {
  return rows.reduce(
    (summary, row) => {
      summary.total += 1;
      if (row.access === "permission") {
        summary.permission += 1;
      }
      if (row.access === "authenticated") {
        summary.authenticated += 1;
      }
      if (row.access === "public") {
        summary.public += 1;
      }
      if (row.synced) {
        summary.synced += 1;
      }
      if (row.permission && row.permissionRegistered) {
        summary.registered += 1;
      }
      return summary;
    },
    { authenticated: 0, permission: 0, public: 0, registered: 0, synced: 0, total: 0 },
  );
}

function toGroupOptions(groups: SystemAPIGroup[], allLabel: string): SelectOption[] {
  return [
    { label: allLabel, value: "" },
    ...groups.map((group) => ({
      label: group.label || group.code,
      value: group.code,
    })),
  ];
}

function toMethodOptions(rows: APIEntryRow[], allLabel: string): SelectOption[] {
  const methods = [...new Set(rows.map((row) => row.method.toUpperCase()))].sort();
  return [
    { label: allLabel, value: "" },
    ...methods.map((method) => ({ label: method, value: method })),
  ];
}

function matchesFilters(
  row: APIEntryRow,
  filters: APIFilters,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (filters.groupCode && row.groupCode !== filters.groupCode) {
    return false;
  }
  if (filters.method && row.method.toUpperCase() !== filters.method) {
    return false;
  }
  if (filters.accessMode && row.access !== filters.accessMode) {
    return false;
  }

  const keyword = filters.keyword.trim().toLowerCase();
  if (!keyword) {
    return true;
  }

  return searchableValues(row, t).some((value) => value.toLowerCase().includes(keyword));
}

function searchableValues(row: APIEntryRow, t: ReturnType<typeof useTranslation>["t"]) {
  return [
    row.access,
    row.code,
    row.description,
    row.group,
    row.groupCode,
    row.groupLabel,
    row.method,
    row.path,
    row.permission,
    accessLabel(row.access, t),
    registrationLabel(row, t),
    row.synced ? t("admin.apis.status.synced") : t("admin.apis.status.unsynced"),
  ].filter((value): value is string => Boolean(value));
}

function accessLabel(access: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (access === "public" || access === "authenticated" || access === "permission") {
    return t(`admin.apis.access.${access}`);
  }
  return access;
}

function registrationStatus(row: APIEntryRow) {
  if (!row.permission) {
    return "none";
  }
  return row.permissionRegistered ? "registered" : "unregistered";
}

function registrationLabel(row: APIEntryRow, t: ReturnType<typeof useTranslation>["t"]) {
  if (!row.permission) {
    return t("admin.apis.status.noPermissionBinding");
  }
  return row.permissionRegistered
    ? t("admin.apis.status.registered")
    : t("admin.apis.status.unregistered");
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

function syncConfirmTitle(action: SyncAction, t: ReturnType<typeof useTranslation>["t"]) {
  return action === "routes"
    ? t("admin.apis.sync.routesConfirmTitle")
    : t("admin.apis.sync.permissionsConfirmTitle");
}

function syncConfirmDescription(action: SyncAction, t: ReturnType<typeof useTranslation>["t"]) {
  return action === "routes"
    ? t("admin.apis.sync.routesConfirmDescription")
    : t("admin.apis.sync.permissionsConfirmDescription");
}

function apiSyncDescription(
  result: SystemAPISyncResult,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (result.persisted) {
    return t("admin.apis.sync.routesPersisted", {
      created: result.created,
      stale: result.stale,
      total: result.total,
      updated: result.updated,
    });
  }
  if (result.storageStatus === "unavailable") {
    return t("admin.apis.sync.routesStorageUnavailable", { total: result.total });
  }
  return t("admin.apis.sync.routesMemoryOnly", { total: result.total });
}

function permissionSyncDescription(
  result: SystemPermissionSyncResult,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (result.persisted) {
    return t("admin.apis.sync.permissionsPersisted", {
      created: result.created,
      skipped: result.skipped,
      total: result.total,
    });
  }
  return t("admin.apis.sync.permissionsStorageUnavailable", {
    status: result.storageStatus,
    total: result.total,
  });
}

function toError(error: unknown) {
  return error instanceof Error ? error : new Error(String(error));
}

function errorTitle(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.apis.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.apis.states.errorTitle");
}

function errorDescription(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.apis.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

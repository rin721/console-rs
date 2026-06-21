import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import {
  ChevronLeft,
  ChevronRight,
  Database,
  Hash,
  KeyRound,
  ListChecks,
  Pencil,
  Plus,
  RefreshCw,
  RotateCcw,
  Save,
  Search,
  SlidersHorizontal,
  Trash2,
  X,
} from "lucide-react";
import {
  useCallback,
  useMemo,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";
import { useTranslation } from "react-i18next";

import { DataTable } from "~/components/aoi/patterns/DataTable";
import { FormField } from "~/components/aoi/patterns/FormField";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { ApiError } from "~/lib/api/client";
import { queryKeys } from "~/lib/api/query-keys";
import {
  systemApi,
  type SystemParameterInput,
  type SystemParameterListQuery,
  type SystemParameterUpdateInput,
} from "~/lib/api/system";
import type { SystemParameter } from "~/lib/api/types";

const defaultPageSize = 10;
const emptyParameterItems: SystemParameter[] = [];

type ParameterFilters = Pick<
  SystemParameterListQuery,
  "endCreatedAt" | "key" | "name" | "startCreatedAt"
>;

type ParameterFilterDraft = ParameterFilters & {
  pageSize: string;
};

type ParameterDraft = {
  description: string;
  key: string;
  name: string;
  value: string;
};

type ParameterNotice = {
  description: string;
  intent?: "danger" | "info";
  title: string;
};

type PendingDelete =
  | {
      mode: "bulk";
      ids: string[];
    }
  | {
      mode: "single";
      parameter: SystemParameter;
    };

const initialDraft: ParameterFilterDraft = {
  endCreatedAt: "",
  key: "",
  name: "",
  pageSize: String(defaultPageSize),
  startCreatedAt: "",
};

const emptyParameterDraft: ParameterDraft = {
  description: "",
  key: "",
  name: "",
  value: "",
};

export default function AdminParametersRoute() {
  const { i18n, t } = useTranslation();
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<ParameterFilterDraft>(initialDraft);
  const [filters, setFilters] = useState<ParameterFilters>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(defaultPageSize);
  const [formMode, setFormMode] = useState<"create" | "edit" | null>(null);
  const [editingParameterId, setEditingParameterId] = useState<string | null>(null);
  const [parameterDraft, setParameterDraft] = useState<ParameterDraft>(emptyParameterDraft);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [pendingDelete, setPendingDelete] = useState<PendingDelete | null>(null);
  const [notice, setNotice] = useState<ParameterNotice | null>(null);

  const parameterQueryKey = queryKeys.system.parameters(i18n.language, page, pageSize, filters);

  const parametersQuery = useQuery({
    queryFn: ({ signal }) => systemApi.listParameters({ ...filters, page, pageSize }, { signal }),
    queryKey: parameterQueryKey,
  });

  const invalidateParameters = () =>
    queryClient.invalidateQueries({ queryKey: ["system", "parameters"] });

  const createParameterMutation = useMutation({
    mutationFn: (input: SystemParameterInput) => systemApi.createParameter(input),
    onError: (error) => {
      setNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.parameters.messages.saveFailedTitle"),
      });
    },
    onSuccess: (parameter) => {
      closeParameterForm();
      setNotice({
        description: t("admin.parameters.messages.createdDescription", {
          name: parameter.name,
        }),
        title: t("admin.parameters.messages.createdTitle"),
      });
      void invalidateParameters();
    },
  });

  const updateParameterMutation = useMutation({
    mutationFn: (input: { id: number | string; value: SystemParameterUpdateInput }) =>
      systemApi.updateParameter(input.id, input.value),
    onError: (error) => {
      setNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.parameters.messages.saveFailedTitle"),
      });
    },
    onSuccess: (parameter) => {
      closeParameterForm();
      setNotice({
        description: t("admin.parameters.messages.updatedDescription", {
          name: parameter.name,
        }),
        title: t("admin.parameters.messages.updatedTitle"),
      });
      void invalidateParameters();
    },
  });

  const deleteParameterMutation = useMutation({
    mutationFn: (parameter: SystemParameter) => systemApi.deleteParameter(parameter.id),
    onError: (error) => {
      setNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.parameters.messages.deleteFailedTitle"),
      });
    },
    onSuccess: (_result, parameter) => {
      const id = parameterIdValue(parameter);
      setPendingDelete(null);
      setSelectedIds((current) => current.filter((selectedId) => selectedId !== id));
      if (editingParameterId === id) {
        closeParameterForm();
      }
      setNotice({
        description: t("admin.parameters.messages.deletedDescription", {
          name: parameter.name,
        }),
        title: t("admin.parameters.messages.deletedTitle"),
      });
      void invalidateParameters();
    },
  });

  const deleteParametersMutation = useMutation({
    mutationFn: (ids: string[]) => systemApi.deleteParameters(ids),
    onError: (error) => {
      setNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.parameters.messages.deleteFailedTitle"),
      });
    },
    onSuccess: (_result, ids) => {
      setPendingDelete(null);
      setSelectedIds([]);
      if (editingParameterId && ids.includes(editingParameterId)) {
        closeParameterForm();
      }
      setNotice({
        description: t("admin.parameters.messages.bulkDeletedDescription", {
          count: ids.length,
        }),
        title: t("admin.parameters.messages.bulkDeletedTitle"),
      });
      void invalidateParameters();
    },
  });

  const pageData = parametersQuery.data;
  const parameterItems = pageData?.items ?? emptyParameterItems;
  const totalPages = Math.max(1, Math.ceil((pageData?.total ?? 0) / pageSize));
  const currentPageCount = parameterItems.length;
  const storagePersisted = pageData?.storageStatus === "persisted";
  const writePending =
    createParameterMutation.isPending ||
    updateParameterMutation.isPending ||
    deleteParameterMutation.isPending ||
    deleteParametersMutation.isPending;
  const parameterDraftValid = Boolean(
    parameterDraft.name.trim() && parameterDraft.key.trim() && parameterDraft.value.trim(),
  );
  const visibleParameterIds = useMemo(() => new Set(parameterItems.map(parameterIdValue)), [
    parameterItems,
  ]);
  const visibleSelectedIds = useMemo(
    () => selectedIds.filter((id) => visibleParameterIds.has(id)),
    [selectedIds, visibleParameterIds],
  );
  const selectedSet = useMemo(() => new Set(visibleSelectedIds), [visibleSelectedIds]);
  const allVisibleSelected =
    parameterItems.length > 0 &&
    parameterItems.every((parameter) => selectedSet.has(parameterIdValue(parameter)));

  const startEdit = useCallback((parameter: SystemParameter) => {
    setFormMode("edit");
    setEditingParameterId(parameterIdValue(parameter));
    setParameterDraft({
      description: parameter.description ?? "",
      key: parameter.key,
      name: parameter.name,
      value: parameter.value,
    });
    setPendingDelete(null);
  }, []);

  const toggleParameterSelection = useCallback((parameter: SystemParameter, checked: boolean) => {
    const id = parameterIdValue(parameter);
    setSelectedIds((current) => {
      if (checked) {
        return current.includes(id) ? current : [...current, id];
      }
      return current.filter((selectedId) => selectedId !== id);
    });
  }, []);

  const toggleAllVisible = useCallback(
    (checked: boolean) => {
      setSelectedIds(checked ? parameterItems.map(parameterIdValue) : []);
    },
    [parameterItems],
  );

  const parameterColumns = useMemo<ColumnDef<SystemParameter>[]>(
    () => [
      {
        id: "selection",
        cell: ({ row }) => {
          const id = parameterIdValue(row.original);
          return (
            <input
              aria-label={t("admin.parameters.selection.rowAria", { id })}
              checked={selectedSet.has(id)}
              className="aoi-parameter-check"
              disabled={!storagePersisted || writePending}
              type="checkbox"
              onChange={(event) => toggleParameterSelection(row.original, event.currentTarget.checked)}
            />
          );
        },
        header: () => (
          <input
            aria-label={t("admin.parameters.selection.allAria")}
            checked={allVisibleSelected}
            className="aoi-parameter-check"
            disabled={!parameterItems.length || !storagePersisted || writePending}
            type="checkbox"
            onChange={(event) => toggleAllVisible(event.currentTarget.checked)}
          />
        ),
      },
      {
        accessorKey: "name",
        cell: ({ row }) => (
          <div className="aoi-parameter-name">
            <strong>{row.original.name}</strong>
            <span>{row.original.id}</span>
          </div>
        ),
        header: t("admin.parameters.columns.name"),
      },
      {
        accessorKey: "key",
        cell: ({ getValue }) => <code className="aoi-parameter-key">{String(getValue())}</code>,
        header: t("admin.parameters.columns.key"),
      },
      {
        accessorKey: "value",
        cell: ({ getValue }) => <span className="aoi-parameter-value">{String(getValue())}</span>,
        header: t("admin.parameters.columns.value"),
      },
      {
        accessorKey: "description",
        cell: ({ getValue }) => {
          const value = getValue();
          return typeof value === "string" && value ? value : t("common.labels.none");
        },
        header: t("admin.parameters.columns.description"),
      },
      {
        accessorKey: "createdAt",
        cell: ({ getValue }) => formatDate(String(getValue()), i18n.language),
        header: t("admin.parameters.columns.createdAt"),
      },
      {
        accessorKey: "updatedAt",
        cell: ({ getValue }) => formatDate(String(getValue()), i18n.language),
        header: t("admin.parameters.columns.updatedAt"),
      },
      {
        id: "actions",
        cell: ({ row }) => (
          <div className="aoi-parameter-actions">
            <Button
              appearance="secondary"
              aria-label={t("admin.parameters.actions.editFor", { name: row.original.name })}
              disabled={!storagePersisted || writePending}
              icon={<Pencil size={15} />}
              onClick={() => startEdit(row.original)}
            >
              {t("admin.parameters.actions.edit")}
            </Button>
            <Button
              appearance="ghost"
              aria-label={t("admin.parameters.actions.deleteFor", { name: row.original.name })}
              disabled={!storagePersisted || writePending}
              icon={<Trash2 size={15} />}
              onClick={() => setPendingDelete({ mode: "single", parameter: row.original })}
            >
              {t("admin.parameters.actions.delete")}
            </Button>
          </div>
        ),
        header: t("admin.parameters.columns.actions"),
      },
    ],
    [
      allVisibleSelected,
      i18n.language,
      parameterItems.length,
      selectedSet,
      startEdit,
      storagePersisted,
      t,
      toggleAllVisible,
      toggleParameterSelection,
      writePending,
    ],
  );

  const updateDraft = (key: keyof ParameterFilterDraft, value: string) => {
    setDraft((current) => ({ ...current, [key]: value }));
  };

  const updateParameterDraft = (key: keyof ParameterDraft, value: string) => {
    setParameterDraft((current) => ({ ...current, [key]: value }));
  };

  const submitFilters = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFilters(normalizeFilters(draft));
    setPage(1);
    setPageSize(parsePageSize(draft.pageSize));
  };

  const resetFilters = () => {
    setDraft(initialDraft);
    setFilters({});
    setPage(1);
    setPageSize(defaultPageSize);
  };

  const startCreate = () => {
    setFormMode("create");
    setEditingParameterId(null);
    setParameterDraft(emptyParameterDraft);
    setPendingDelete(null);
  };

  const closeParameterForm = () => {
    setFormMode(null);
    setEditingParameterId(null);
    setParameterDraft(emptyParameterDraft);
  };

  const submitParameter = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const payload = normalizeParameterDraft(parameterDraft);
    if (!payload) {
      setNotice({
        description: t("admin.parameters.messages.requiredDescription"),
        intent: "danger",
        title: t("admin.parameters.messages.requiredTitle"),
      });
      return;
    }
    if (!storagePersisted) {
      setNotice({
        description: t("admin.parameters.states.storageUnavailableDescription"),
        intent: "danger",
        title: t("admin.parameters.states.storageUnavailableTitle"),
      });
      return;
    }

    setNotice(null);
    if (formMode === "edit" && editingParameterId) {
      updateParameterMutation.mutate({ id: editingParameterId, value: payload });
      return;
    }
    createParameterMutation.mutate(payload);
  };

  const openBulkDelete = () => {
    if (!visibleSelectedIds.length) {
      return;
    }
    setPendingDelete({ ids: visibleSelectedIds, mode: "bulk" });
  };

  const confirmPendingDelete = () => {
    if (!pendingDelete || !storagePersisted) {
      return;
    }
    setNotice(null);
    if (pendingDelete.mode === "single") {
      deleteParameterMutation.mutate(pendingDelete.parameter);
      return;
    }
    deleteParametersMutation.mutate(pendingDelete.ids);
  };

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-parameters-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.parameters.badge")}</Badge>
          <h1 id="admin-parameters-title">{t("admin.parameters.title")}</h1>
          <p>{t("admin.parameters.description")}</p>
        </div>
        <div className="aoi-parameter-page-actions">
          <Button
            disabled={!storagePersisted || writePending}
            icon={<Plus size={17} />}
            onClick={startCreate}
          >
            {t("admin.parameters.actions.create")}
          </Button>
          <Button
            appearance="secondary"
            disabled={!visibleSelectedIds.length || !storagePersisted || writePending}
            icon={<Trash2 size={17} />}
            onClick={openBulkDelete}
          >
            {t("admin.parameters.actions.deleteSelected")}
          </Button>
          <Button
            appearance="secondary"
            icon={<RefreshCw size={17} />}
            loading={parametersQuery.isFetching}
            onClick={() => void parametersQuery.refetch()}
          >
            {t("admin.parameters.actions.refresh")}
          </Button>
        </div>
      </div>

      {parametersQuery.error ? (
        <StateBlock
          intent="danger"
          title={errorTitle(parametersQuery.error, t)}
          description={errorDescription(parametersQuery.error, t)}
        />
      ) : null}

      {notice ? (
        <StateBlock description={notice.description} intent={notice.intent} title={notice.title} />
      ) : null}

      {pendingDelete ? (
        <StateBlock
          action={
            <div className="aoi-parameter-confirm-actions">
              <Button loading={writePending} onClick={confirmPendingDelete}>
                {t("admin.parameters.actions.confirmDelete")}
              </Button>
              <Button
                appearance="secondary"
                disabled={writePending}
                onClick={() => setPendingDelete(null)}
              >
                {t("admin.parameters.actions.cancel")}
              </Button>
            </div>
          }
          description={
            pendingDelete.mode === "single"
              ? t("admin.parameters.delete.singleDescription", {
                  name: pendingDelete.parameter.name,
                })
              : t("admin.parameters.delete.bulkDescription", {
                  count: pendingDelete.ids.length,
                })
          }
          title={
            pendingDelete.mode === "single"
              ? t("admin.parameters.delete.singleTitle")
              : t("admin.parameters.delete.bulkTitle")
          }
        />
      ) : null}

      {formMode ? (
        <section className="aoi-admin-panel aoi-parameter-form-panel">
          <header className="aoi-admin-panel-header-row">
            <div>
              <h2>
                {formMode === "edit"
                  ? t("admin.parameters.form.editTitle")
                  : t("admin.parameters.form.createTitle")}
              </h2>
              <p>{t("admin.parameters.form.description")}</p>
            </div>
            {editingParameterId ? <Badge>{editingParameterId}</Badge> : null}
          </header>
          <form className="aoi-parameter-form-grid" onSubmit={submitParameter}>
            <FormField
              disabled={writePending}
              label={t("admin.parameters.form.name")}
              placeholder={t("admin.parameters.form.placeholders.name")}
              value={parameterDraft.name}
              onChange={(event) => updateParameterDraft("name", event.currentTarget.value)}
            />
            <FormField
              disabled={writePending}
              label={t("admin.parameters.form.key")}
              placeholder={t("admin.parameters.form.placeholders.key")}
              value={parameterDraft.key}
              onChange={(event) => updateParameterDraft("key", event.currentTarget.value)}
            />
            <label className="aoi-form-field aoi-parameter-form-field--span">
              <span>{t("admin.parameters.form.value")}</span>
              <textarea
                disabled={writePending}
                placeholder={t("admin.parameters.form.placeholders.value")}
                rows={5}
                value={parameterDraft.value}
                onChange={(event) => updateParameterDraft("value", event.currentTarget.value)}
              />
            </label>
            <label className="aoi-form-field aoi-parameter-form-field--span">
              <span>{t("admin.parameters.form.descriptionField")}</span>
              <textarea
                disabled={writePending}
                placeholder={t("admin.parameters.form.placeholders.description")}
                rows={3}
                value={parameterDraft.description}
                onChange={(event) =>
                  updateParameterDraft("description", event.currentTarget.value)
                }
              />
            </label>
            <div className="aoi-parameter-form-actions">
              <Button
                disabled={!parameterDraftValid || !storagePersisted}
                icon={<Save size={17} />}
                loading={createParameterMutation.isPending || updateParameterMutation.isPending}
                type="submit"
              >
                {formMode === "edit"
                  ? t("admin.parameters.actions.save")
                  : t("admin.parameters.actions.create")}
              </Button>
              <Button
                appearance="secondary"
                disabled={writePending}
                icon={<X size={17} />}
                onClick={closeParameterForm}
              >
                {t("admin.parameters.actions.cancel")}
              </Button>
            </div>
          </form>
        </section>
      ) : null}

      <div className="aoi-admin-stat-grid" aria-label={t("admin.parameters.summaryLabel")}>
        <ParameterStatCard
          icon={<SlidersHorizontal size={19} />}
          label={t("admin.parameters.metrics.total")}
          value={
            pageData
              ? formatNumber(pageData.total, i18n.language)
              : fallbackValue(parametersQuery.isLoading, t)
          }
        />
        <ParameterStatCard
          icon={<ListChecks size={19} />}
          label={t("admin.parameters.metrics.currentPage")}
          value={formatNumber(currentPageCount, i18n.language)}
        />
        <ParameterStatCard
          icon={<Hash size={19} />}
          label={t("admin.parameters.metrics.page")}
          value={t("admin.parameters.pagination.pageStatus", {
            page,
            totalPages,
          })}
        />
        <ParameterStatCard
          icon={<KeyRound size={19} />}
          label={t("admin.parameters.metrics.selected")}
          value={formatNumber(visibleSelectedIds.length, i18n.language)}
        />
        <ParameterStatCard
          icon={<Database size={19} />}
          label={t("admin.parameters.metrics.storage")}
          value={
            pageData
              ? storageStatusLabel(pageData.storageStatus, t)
              : fallbackValue(parametersQuery.isLoading, t)
          }
        />
      </div>

      <section className="aoi-admin-panel">
        <header>
          <h2>{t("admin.parameters.filters.title")}</h2>
          <p>{t("admin.parameters.filters.description")}</p>
        </header>
        <form className="aoi-admin-filter-form" onSubmit={submitFilters}>
          <FormField
            label={t("admin.parameters.filters.name")}
            value={draft.name}
            onChange={(event) => updateDraft("name", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.parameters.filters.key")}
            value={draft.key}
            onChange={(event) => updateDraft("key", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.parameters.filters.startCreatedAt")}
            type="date"
            value={draft.startCreatedAt}
            onChange={(event) => updateDraft("startCreatedAt", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.parameters.filters.endCreatedAt")}
            type="date"
            value={draft.endCreatedAt}
            onChange={(event) => updateDraft("endCreatedAt", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.parameters.filters.pageSize")}
            max={100}
            min={1}
            type="number"
            value={draft.pageSize}
            onChange={(event) => updateDraft("pageSize", event.currentTarget.value)}
          />
          <div className="aoi-admin-filter-actions">
            <Button icon={<Search size={17} />} loading={parametersQuery.isFetching} type="submit">
              {t("admin.parameters.actions.search")}
            </Button>
            <Button appearance="secondary" icon={<RotateCcw size={17} />} onClick={resetFilters}>
              {t("admin.parameters.actions.reset")}
            </Button>
          </div>
        </form>
      </section>

      <section className="aoi-admin-panel">
        <header className="aoi-admin-panel-header-row">
          <div>
            <h2>{t("admin.parameters.list.title")}</h2>
            <p>
              {t("admin.parameters.list.description", {
                count: pageData?.total ?? 0,
              })}
            </p>
          </div>
          <div className="aoi-admin-pager" aria-label={t("admin.parameters.pagination.label")}>
            <Button
              appearance="secondary"
              disabled={page <= 1 || parametersQuery.isFetching}
              icon={<ChevronLeft size={17} />}
              onClick={() => setPage((current) => Math.max(1, current - 1))}
            >
              {t("admin.parameters.pagination.previous")}
            </Button>
            <span>
              {t("admin.parameters.pagination.pageStatus", {
                page,
                totalPages,
              })}
            </span>
            <Button
              appearance="secondary"
              disabled={page >= totalPages || parametersQuery.isFetching}
              icon={<ChevronRight size={17} />}
              onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
            >
              {t("admin.parameters.pagination.next")}
            </Button>
          </div>
        </header>

        {parametersQuery.isLoading ? (
          <StateBlock
            title={t("admin.parameters.states.loadingTitle")}
            description={t("admin.parameters.states.loadingDescription")}
          />
        ) : pageData ? (
          <>
            {storagePersisted ? null : (
              <StateBlock
                title={t("admin.parameters.states.storageUnavailableTitle")}
                description={t("admin.parameters.states.storageUnavailableDescription")}
              />
            )}
            <div className="aoi-parameter-table">
              <DataTable
                columns={parameterColumns}
                data={parameterItems}
                emptyLabel={t("admin.parameters.empty")}
              />
            </div>
          </>
        ) : (
          <StateBlock
            title={t("admin.parameters.states.emptyTitle")}
            description={t("admin.parameters.states.emptyDescription")}
          />
        )}
      </section>
    </section>
  );
}

type ParameterStatCardProps = {
  icon: ReactNode;
  label: string;
  value: string;
};

function ParameterStatCard({ icon, label, value }: ParameterStatCardProps) {
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

function normalizeFilters(draft: ParameterFilterDraft): ParameterFilters {
  return {
    endCreatedAt: trimmedOrUndefined(draft.endCreatedAt),
    key: trimmedOrUndefined(draft.key),
    name: trimmedOrUndefined(draft.name),
    startCreatedAt: trimmedOrUndefined(draft.startCreatedAt),
  };
}

function normalizeParameterDraft(draft: ParameterDraft): SystemParameterInput | null {
  const payload = {
    description: draft.description.trim(),
    key: draft.key.trim(),
    name: draft.name.trim(),
    value: draft.value.trim(),
  };
  if (!payload.name || !payload.key || !payload.value) {
    return null;
  }
  return payload;
}

function trimmedOrUndefined(value: string | undefined) {
  const trimmed = value?.trim();
  return trimmed || undefined;
}

function parsePageSize(value: string) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return defaultPageSize;
  }
  return Math.min(100, Math.max(1, Math.trunc(parsed)));
}

function fallbackValue(loading: boolean, t: ReturnType<typeof useTranslation>["t"]) {
  return loading ? t("loading.app") : t("common.labels.none");
}

function parameterIdValue(parameter: SystemParameter) {
  return String(parameter.id);
}

function storageStatusLabel(status: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (status === "persisted") {
    return t("admin.parameters.storage.persisted");
  }
  if (status === "unavailable") {
    return t("admin.parameters.storage.unavailable");
  }
  return status || t("admin.parameters.storage.unknown");
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
    return t("admin.parameters.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.parameters.states.errorTitle");
}

function errorDescription(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.parameters.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

function toError(error: unknown) {
  if (error instanceof Error) {
    return error;
  }
  return new Error(String(error));
}

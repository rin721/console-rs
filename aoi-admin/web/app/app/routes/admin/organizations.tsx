import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import type { TFunction } from "i18next";
import {
  Building2,
  ChevronLeft,
  ChevronRight,
  Plus,
  RefreshCw,
  Repeat2,
  Save,
  Search,
  X,
} from "lucide-react";
import { useMemo, useState, type FormEvent, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { z } from "zod";

import { DataTable } from "~/components/aoi/patterns/DataTable";
import { FormField } from "~/components/aoi/patterns/FormField";
import { SelectField } from "~/components/aoi/patterns/SelectField";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { authApi } from "~/lib/api/auth";
import { ApiError } from "~/lib/api/client";
import { iamApi, type IAMOrganizationListQuery } from "~/lib/api/iam";
import { queryKeys } from "~/lib/api/query-keys";
import type {
  IAMCreateOrganizationInput,
  IAMOrganization,
  IAMUpdateOrganizationInput,
} from "~/lib/api/types";
import { useAuthStore } from "~/stores/auth-store";

const defaultPageSize = 10;
const organizationCodePattern = /^[a-z][a-z0-9_-]{0,63}$/;

const organizationCreateSchema = z.object({
  code: z.string().trim().regex(organizationCodePattern),
  name: z.string().trim().min(1),
});

const organizationUpdateSchema = z.object({
  name: z.string().trim().min(1),
});

type OrganizationFilters = Pick<
  IAMOrganizationListQuery,
  "code" | "keyword" | "name" | "status"
> & {
  desc?: boolean;
  orderKey?: string;
};

type OrganizationFilterDraft = {
  code: string;
  keyword: string;
  name: string;
  pageSize: string;
  status: string;
};

type OrganizationDraft = {
  code: string;
  name: string;
};

type OrganizationNotice = {
  description: string;
  intent?: "danger" | "info";
  title: string;
};

type SetIdentity = ReturnType<typeof useAuthStore.getState>["setIdentity"];

const initialFilterDraft: OrganizationFilterDraft = {
  code: "",
  keyword: "",
  name: "",
  pageSize: String(defaultPageSize),
  status: "",
};

const initialCreateDraft: OrganizationDraft = {
  code: "",
  name: "",
};

const emptyOrganizations: IAMOrganization[] = [];

export default function AdminOrganizationsRoute() {
  const { i18n, t } = useTranslation();
  const queryClient = useQueryClient();
  const currentOrgId = useAuthStore((state) => state.currentOrgId);
  const setIdentity = useAuthStore((state) => state.setIdentity);
  const setSession = useAuthStore((state) => state.setSession);
  const [draft, setDraft] = useState<OrganizationFilterDraft>(initialFilterDraft);
  const [filters, setFilters] = useState<OrganizationFilters>({
    desc: true,
    orderKey: "id",
  });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(defaultPageSize);
  const [createDraft, setCreateDraft] = useState<OrganizationDraft>(initialCreateDraft);
  const [currentNameDraft, setCurrentNameDraft] = useState<string | null>(null);
  const [notice, setNotice] = useState<OrganizationNotice | null>(null);

  const organizationsQueryKey = queryKeys.iam.organizations(i18n.language, page, pageSize, filters);

  const organizationsQuery = useQuery({
    enabled: Boolean(currentOrgId),
    queryFn: ({ signal }) => iamApi.listOrganizations({ ...filters, page, pageSize }, { signal }),
    queryKey: organizationsQueryKey,
  });

  const pageData = organizationsQuery.data;
  const organizations = pageData?.items ?? emptyOrganizations;
  const currentOrganization = useMemo(
    () => organizations.find((org) => String(org.id) === currentOrgId) ?? null,
    [currentOrgId, organizations],
  );
  const currentName = currentNameDraft ?? currentOrganization?.name ?? "";
  const totalPages = Math.max(1, Math.ceil((pageData?.total ?? 0) / pageSize));
  const summary = useMemo(
    () => summarizeOrganizations(organizations, pageData?.total ?? 0),
    [organizations, pageData?.total],
  );

  const createOrganizationMutation = useMutation({
    mutationFn: (input: IAMCreateOrganizationInput) => iamApi.createOrganization(input),
    onError: (error) => {
      setNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.organizations.create.errorTitle"),
      });
    },
    onSuccess: (organization) => {
      setCreateDraft(initialCreateDraft);
      setNotice({
        description: t("admin.organizations.create.successDescription", {
          name: organization.name,
        }),
        title: t("admin.organizations.create.successTitle"),
      });
      setPage(1);
      void reloadIdentity(setIdentity);
      void queryClient.invalidateQueries({ queryKey: queryKeys.iam.root });
      void queryClient.invalidateQueries({ queryKey: queryKeys.auth.root });
    },
  });

  const updateOrganizationMutation = useMutation({
    mutationFn: (input: { orgId: number | string; values: IAMUpdateOrganizationInput }) =>
      iamApi.updateOrganization(input.orgId, input.values),
    onError: (error) => {
      setNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.organizations.current.errorTitle"),
      });
    },
    onSuccess: (organization) => {
      setCurrentNameDraft(organization.name);
      setNotice({
        description: t("admin.organizations.current.successDescription", {
          name: organization.name,
        }),
        title: t("admin.organizations.current.successTitle"),
      });
      void reloadIdentity(setIdentity);
      void queryClient.invalidateQueries({ queryKey: queryKeys.iam.root });
      void queryClient.invalidateQueries({ queryKey: queryKeys.auth.root });
    },
  });

  const switchOrganizationMutation = useMutation({
    mutationFn: (organization: IAMOrganization) => authApi.switchOrg(organization.id),
    onError: (error) => {
      setNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.organizations.switch.errorTitle"),
      });
    },
    onSuccess: async (pair, organization) => {
      setSession(pair);
      const [user, orgs] = await Promise.all([authApi.getMe(), authApi.listMyOrganizations()]);
      setIdentity(user, orgs);
      setCurrentNameDraft(null);
      setNotice({
        description: t("admin.organizations.switch.successDescription", {
          name: organization.name,
        }),
        title: t("admin.organizations.switch.successTitle"),
      });
      void queryClient.invalidateQueries({ queryKey: queryKeys.iam.root });
      void queryClient.invalidateQueries({ queryKey: queryKeys.auth.root });
    },
  });

  const organizationColumns = useMemo<ColumnDef<IAMOrganization>[]>(
    () => [
      {
        accessorKey: "code",
        cell: ({ row }) => (
          <div className="aoi-org-identity">
            <strong>{row.original.name}</strong>
            <code>{row.original.code}</code>
          </div>
        ),
        header: t("admin.organizations.columns.organization"),
      },
      {
        cell: ({ row }) => (
          <span className="aoi-iam-status" data-status={row.original.status}>
            {organizationStatusLabel(row.original.status, t)}
          </span>
        ),
        header: t("admin.organizations.columns.status"),
      },
      {
        cell: ({ row }) => (
          <span
            className="aoi-iam-status"
            data-status={String(row.original.id) === currentOrgId ? "used" : "active"}
          >
            {String(row.original.id) === currentOrgId
              ? t("admin.organizations.current.current")
              : t("admin.organizations.current.switchable")}
          </span>
        ),
        header: t("admin.organizations.columns.current"),
      },
      {
        cell: ({ row }) => formatDate(row.original.createdAt, i18n.language, t),
        header: t("admin.organizations.columns.createdAt"),
      },
      {
        cell: ({ row }) => {
          const isCurrent = String(row.original.id) === currentOrgId;
          return (
            <Button
              appearance="secondary"
              disabled={isCurrent || switchOrganizationMutation.isPending}
              icon={<Repeat2 size={17} />}
              onClick={() => switchOrganizationMutation.mutate(row.original)}
            >
              {isCurrent
                ? t("admin.organizations.actions.current")
                : t("admin.organizations.actions.switch")}
            </Button>
          );
        },
        header: t("admin.organizations.columns.actions"),
      },
    ],
    [currentOrgId, i18n.language, switchOrganizationMutation, t],
  );

  function handleSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextPageSize = parsePageSize(draft.pageSize);
    setPageSize(nextPageSize);
    setPage(1);
    setFilters({
      code: trimmedOrUndefined(draft.code),
      desc: true,
      keyword: trimmedOrUndefined(draft.keyword),
      name: trimmedOrUndefined(draft.name),
      orderKey: "id",
      status: trimmedOrUndefined(draft.status),
    });
  }

  function handleReset() {
    setDraft(initialFilterDraft);
    setFilters({ desc: true, orderKey: "id" });
    setPage(1);
    setPageSize(defaultPageSize);
  }

  function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const parsed = organizationCreateSchema.safeParse(createDraft);
    if (!parsed.success) {
      setNotice({
        description: validationMessage(parsed.error, t),
        intent: "danger",
        title: t("admin.organizations.validation.title"),
      });
      return;
    }
    createOrganizationMutation.mutate(parsed.data);
  }

  function handleUpdateCurrent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!currentOrgId) {
      return;
    }
    const parsed = organizationUpdateSchema.safeParse({ name: currentName });
    if (!parsed.success) {
      setNotice({
        description: validationMessage(parsed.error, t),
        intent: "danger",
        title: t("admin.organizations.validation.title"),
      });
      return;
    }
    updateOrganizationMutation.mutate({ orgId: currentOrgId, values: parsed.data });
  }

  if (!currentOrgId) {
    return (
      <section className="aoi-admin-dashboard">
        <StateBlock
          title={t("admin.organizations.states.missingOrgTitle")}
          description={t("admin.organizations.states.missingOrgDescription")}
        />
      </section>
    );
  }

  const queryError = toMaybeError(organizationsQuery.error);
  const isMutating =
    createOrganizationMutation.isPending ||
    updateOrganizationMutation.isPending ||
    switchOrganizationMutation.isPending;

  return (
    <section className="aoi-admin-dashboard">
      <header className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.organizations.badge")}</Badge>
          <h1>{t("admin.organizations.title")}</h1>
          <p>{t("admin.organizations.description")}</p>
        </div>
        <div className="aoi-admin-action-row aoi-org-page-actions">
          <Button
            appearance="secondary"
            disabled={organizationsQuery.isFetching}
            icon={<RefreshCw size={17} />}
            onClick={() => void organizationsQuery.refetch()}
          >
            {t("admin.organizations.actions.refresh")}
          </Button>
        </div>
      </header>

      {notice ? (
        <StateBlock
          title={notice.title}
          description={notice.description}
          intent={notice.intent}
          action={
            <Button appearance="ghost" onClick={() => setNotice(null)}>
              {t("admin.organizations.actions.dismiss")}
            </Button>
          }
        />
      ) : null}

      <div className="aoi-admin-stat-grid" aria-label={t("admin.organizations.summaryLabel")}>
        <OrganizationStatCard
          icon={<Building2 size={18} />}
          label={t("admin.organizations.metrics.total")}
          value={formatNumber(summary.total, i18n.language)}
        />
        <OrganizationStatCard
          icon={<Building2 size={18} />}
          label={t("admin.organizations.metrics.active")}
          value={formatNumber(summary.active, i18n.language)}
        />
        <OrganizationStatCard
          icon={<Building2 size={18} />}
          label={t("admin.organizations.metrics.disabled")}
          value={formatNumber(summary.disabled, i18n.language)}
        />
        <OrganizationStatCard
          icon={<Save size={18} />}
          label={t("admin.organizations.metrics.storage")}
          value={storageStatusLabel(pageData?.storageStatus, t)}
        />
        <OrganizationStatCard
          icon={<Repeat2 size={18} />}
          label={t("admin.organizations.metrics.current")}
          value={currentOrganization?.code ?? t("common.labels.none")}
        />
      </div>

      {queryError ? (
        <StateBlock
          title={errorTitle(queryError, t)}
          description={errorDescription(queryError, t)}
          intent="danger"
        />
      ) : organizationsQuery.isLoading ? (
        <StateBlock
          title={t("admin.organizations.states.loadingTitle")}
          description={t("admin.organizations.states.loadingDescription")}
        />
      ) : (
        <div className="aoi-org-workbench">
          <section className="aoi-admin-panel aoi-admin-panel--span-2">
            <header>
              <h2>{t("admin.organizations.filters.title")}</h2>
              <p>{t("admin.organizations.filters.description")}</p>
            </header>
            <form className="aoi-admin-filter-form aoi-org-filter-form" onSubmit={handleSearch}>
              <FormField
                label={t("admin.organizations.filters.keyword")}
                value={draft.keyword}
                onChange={(event) =>
                  setDraft((current) => ({ ...current, keyword: event.target.value }))
                }
              />
              <FormField
                label={t("admin.organizations.fields.code")}
                value={draft.code}
                onChange={(event) =>
                  setDraft((current) => ({ ...current, code: event.target.value }))
                }
              />
              <FormField
                label={t("admin.organizations.fields.name")}
                value={draft.name}
                onChange={(event) =>
                  setDraft((current) => ({ ...current, name: event.target.value }))
                }
              />
              <SelectField
                label={t("admin.organizations.fields.status")}
                options={[
                  { label: t("admin.organizations.filters.allStatuses"), value: "" },
                  { label: t("admin.organizations.status.active"), value: "active" },
                  { label: t("admin.organizations.status.disabled"), value: "disabled" },
                ]}
                value={draft.status}
                onChange={(event) =>
                  setDraft((current) => ({ ...current, status: event.target.value }))
                }
              />
              <FormField
                label={t("admin.organizations.filters.pageSize")}
                min={1}
                max={100}
                type="number"
                value={draft.pageSize}
                onChange={(event) =>
                  setDraft((current) => ({ ...current, pageSize: event.target.value }))
                }
              />
              <div className="aoi-admin-filter-actions">
                <Button
                  disabled={organizationsQuery.isFetching}
                  icon={<Search size={17} />}
                  type="submit"
                >
                  {t("admin.organizations.actions.search")}
                </Button>
                <Button appearance="secondary" icon={<X size={17} />} onClick={handleReset}>
                  {t("admin.organizations.actions.reset")}
                </Button>
              </div>
            </form>
          </section>

          <section className="aoi-admin-panel aoi-admin-panel--span-2">
            <header className="aoi-admin-panel-header-row">
              <div>
                <h2>{t("admin.organizations.list.title")}</h2>
                <p>
                  {t("admin.organizations.list.description", {
                    count: organizations.length,
                    total: pageData?.total ?? 0,
                  })}
                </p>
              </div>
              <div
                className="aoi-admin-pager"
                aria-label={t("admin.organizations.pagination.label")}
              >
                <Button
                  appearance="secondary"
                  disabled={page <= 1 || organizationsQuery.isFetching}
                  icon={<ChevronLeft size={17} />}
                  onClick={() => setPage((current) => Math.max(1, current - 1))}
                >
                  {t("admin.organizations.pagination.previous")}
                </Button>
                <span>{t("admin.organizations.pagination.pageStatus", { page, totalPages })}</span>
                <Button
                  appearance="secondary"
                  disabled={page >= totalPages || organizationsQuery.isFetching}
                  icon={<ChevronRight size={17} />}
                  onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
                >
                  {t("admin.organizations.pagination.next")}
                </Button>
              </div>
            </header>
            {pageData?.storageStatus === "persisted" ? null : (
              <StateBlock
                title={t("admin.organizations.states.storageUnavailableTitle")}
                description={t("admin.organizations.states.storageUnavailableDescription")}
              />
            )}
            <div className="aoi-org-table">
              <DataTable
                columns={organizationColumns}
                data={organizations}
                emptyLabel={t("admin.organizations.empty")}
              />
            </div>
          </section>

          <section className="aoi-admin-panel">
            <header>
              <h2>{t("admin.organizations.current.title")}</h2>
              <p>{t("admin.organizations.current.description")}</p>
            </header>
            <form className="aoi-org-form" onSubmit={handleUpdateCurrent}>
              <FormField
                disabled
                label={t("admin.organizations.fields.code")}
                value={currentOrganization?.code ?? ""}
              />
              <FormField
                label={t("admin.organizations.fields.name")}
                value={currentName}
                onChange={(event) => setCurrentNameDraft(event.target.value)}
              />
              <Button
                disabled={isMutating || !currentName.trim()}
                icon={<Save size={17} />}
                loading={updateOrganizationMutation.isPending}
                type="submit"
              >
                {t("admin.organizations.actions.save")}
              </Button>
            </form>
          </section>

          <section className="aoi-admin-panel">
            <header>
              <h2>{t("admin.organizations.create.title")}</h2>
              <p>{t("admin.organizations.create.description")}</p>
            </header>
            <form className="aoi-org-form" onSubmit={handleCreate}>
              <FormField
                label={t("admin.organizations.fields.code")}
                placeholder={t("admin.organizations.fields.codePlaceholder")}
                value={createDraft.code}
                onChange={(event) =>
                  setCreateDraft((current) => ({ ...current, code: event.target.value }))
                }
              />
              <FormField
                label={t("admin.organizations.fields.name")}
                placeholder={t("admin.organizations.fields.namePlaceholder")}
                value={createDraft.name}
                onChange={(event) =>
                  setCreateDraft((current) => ({ ...current, name: event.target.value }))
                }
              />
              <Button
                disabled={isMutating || !createDraft.code.trim() || !createDraft.name.trim()}
                icon={<Plus size={17} />}
                loading={createOrganizationMutation.isPending}
                type="submit"
              >
                {t("admin.organizations.actions.create")}
              </Button>
            </form>
          </section>
        </div>
      )}
    </section>
  );
}

type OrganizationStatCardProps = {
  icon: ReactNode;
  label: string;
  value: string;
};

function OrganizationStatCard({ icon, label, value }: OrganizationStatCardProps) {
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

async function reloadIdentity(setIdentity: SetIdentity) {
  const [user, orgs] = await Promise.all([authApi.getMe(), authApi.listMyOrganizations()]);
  setIdentity(user, orgs);
}

function summarizeOrganizations(items: IAMOrganization[], total: number) {
  return items.reduce(
    (summary, item) => {
      if (item.status === "active") {
        summary.active += 1;
      } else {
        summary.disabled += 1;
      }
      return summary;
    },
    { active: 0, disabled: 0, total },
  );
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

function organizationStatusLabel(status: string, t: TFunction) {
  if (status === "active" || status === "disabled") {
    return t(`admin.organizations.status.${status}`);
  }
  return status;
}

function storageStatusLabel(status: string | undefined, t: TFunction) {
  if (status === "persisted") {
    return t("admin.organizations.storage.persisted");
  }
  if (status === "unavailable") {
    return t("admin.organizations.storage.unavailable");
  }
  return status || t("admin.organizations.storage.unknown");
}

function validationMessage(error: z.ZodError, t: TFunction) {
  const firstPath = error.issues[0]?.path[0];
  if (firstPath === "code") {
    return t("admin.organizations.validation.code");
  }
  return t("admin.organizations.validation.name");
}

function formatNumber(value: number, locale: string) {
  return new Intl.NumberFormat(locale).format(value);
}

function formatDate(value: string | null | undefined, locale: string, t: TFunction) {
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

function errorTitle(error: Error, t: TFunction) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.organizations.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.organizations.states.errorTitle");
}

function errorDescription(error: Error, t: TFunction) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.organizations.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

function toMaybeError(error: unknown) {
  return error instanceof Error ? error : null;
}

function toError(error: unknown) {
  return error instanceof Error ? error : new Error(String(error));
}

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import {
  Ban,
  ChevronLeft,
  ChevronRight,
  Clock3,
  Copy,
  Database,
  KeyRound,
  Plus,
  RefreshCw,
  RotateCcw,
  Search,
  ShieldAlert,
  Terminal,
  Users,
} from "lucide-react";
import { useMemo, useState, type FormEvent, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { z } from "zod";

import { DataTable } from "~/components/aoi/patterns/DataTable";
import { Dialog } from "~/components/aoi/patterns/Dialog";
import { FormField } from "~/components/aoi/patterns/FormField";
import { TableSkeleton } from "~/components/aoi/patterns/LoadingSkeletons";
import { SelectField, type SelectOption } from "~/components/aoi/patterns/SelectField";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { ApiError, resolveApiBaseUrl } from "~/lib/api/client";
import { API_ENDPOINTS } from "~/lib/api/endpoints";
import { iamApi, type IAMAPITokenListQuery } from "~/lib/api/iam";
import { queryKeys } from "~/lib/api/query-keys";
import type {
  IAMAPIToken,
  IAMAPITokenPage,
  IAMCreateAPITokenInput,
  IAMCreateAPITokenResult,
  IAMOrganizationUser,
} from "~/lib/api/types";
import { useAuthStore } from "~/stores/auth-store";

const defaultPageSize = 10;
const metadataPageSize = 100;

const issueInputSchema = z.object({
  days: z
    .number()
    .int()
    .refine((value) => value === -1 || (value >= 1 && value <= 3650)),
  remark: z.string().trim().max(500).optional(),
  roleCode: z.string().trim().min(1),
  userId: z.string().trim().min(1),
}) satisfies z.ZodType<IAMCreateAPITokenInput>;

type APITokenFilters = Pick<IAMAPITokenListQuery, "status" | "userId">;

type APITokenFilterDraft = {
  pageSize: string;
  status: string;
  userId: string;
};

type APITokenIssueDraft = {
  days: string;
  remark: string;
  roleCode: string;
  userId: string;
};

type TokenNotice = {
  description: string;
  intent?: "danger" | "info";
  title: string;
};

const initialFilterDraft: APITokenFilterDraft = {
  pageSize: String(defaultPageSize),
  status: "",
  userId: "",
};

const initialIssueDraft: APITokenIssueDraft = {
  days: "30",
  remark: "",
  roleCode: "",
  userId: "",
};

export default function AdminAPITokensRoute() {
  const { i18n, t } = useTranslation();
  const queryClient = useQueryClient();
  const currentOrgId = useAuthStore((state) => state.currentOrgId);
  const [draft, setDraft] = useState<APITokenFilterDraft>(initialFilterDraft);
  const [filters, setFilters] = useState<APITokenFilters>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(defaultPageSize);
  const [issueOpen, setIssueOpen] = useState(false);
  const [issueDraft, setIssueDraft] = useState<APITokenIssueDraft>(initialIssueDraft);
  const [issuedResult, setIssuedResult] = useState<IAMCreateAPITokenResult | null>(null);
  const [notice, setNotice] = useState<TokenNotice | null>(null);
  const [pendingRevokeToken, setPendingRevokeToken] = useState<IAMAPIToken | null>(null);

  const apiTokenQueryKey = queryKeys.iam.apiTokens(
    i18n.language,
    currentOrgId ?? "",
    page,
    pageSize,
    filters,
  );

  const apiTokensQuery = useQuery({
    enabled: Boolean(currentOrgId),
    queryFn: ({ signal }) =>
      iamApi.listAPITokens(currentOrgId ?? "", { ...filters, page, pageSize }, { signal }),
    queryKey: apiTokenQueryKey,
  });

  const usersQuery = useQuery({
    enabled: Boolean(currentOrgId && issueOpen),
    queryFn: ({ signal }) =>
      iamApi.listUsers(
        currentOrgId ?? "",
        { desc: true, orderKey: "id", page: 1, pageSize: metadataPageSize },
        { signal },
      ),
    queryKey: queryKeys.iam.users(i18n.language, currentOrgId ?? "", 1, metadataPageSize),
  });

  const rolesQuery = useQuery({
    enabled: Boolean(currentOrgId && issueOpen),
    queryFn: ({ signal }) => iamApi.listRoles(currentOrgId ?? "", { signal }),
    queryKey: queryKeys.iam.roles(i18n.language, currentOrgId ?? ""),
  });

  const revokeTokenMutation = useMutation({
    mutationFn: (token: IAMAPIToken) => iamApi.revokeAPIToken(currentOrgId ?? "", token.id),
    onError: (error, _token, context: { previousPage?: IAMAPITokenPage } | undefined) => {
      if (context?.previousPage) {
        queryClient.setQueryData(apiTokenQueryKey, context.previousPage);
      }
      setNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.apiTokens.revoke.errorTitle"),
      });
    },
    onMutate: async (token) => {
      setNotice(null);
      setPendingRevokeToken(null);
      await queryClient.cancelQueries({ queryKey: apiTokenQueryKey });
      const previousPage = queryClient.getQueryData<IAMAPITokenPage>(apiTokenQueryKey);
      const revokedAt = new Date().toISOString();
      queryClient.setQueryData<IAMAPITokenPage>(apiTokenQueryKey, (current) =>
        current
          ? {
              ...current,
              items: current.items.map((item) =>
                sameID(item.id, token.id)
                  ? { ...item, revokedAt, status: "revoked", updatedAt: revokedAt }
                  : item,
              ),
            }
          : current,
      );
      return { previousPage };
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: apiTokenQueryKey });
    },
    onSuccess: (_result, token) => {
      setNotice({
        description: t("admin.apiTokens.revoke.successDescription", { id: token.id }),
        title: t("admin.apiTokens.revoke.successTitle"),
      });
    },
  });

  const createTokenMutation = useMutation({
    mutationFn: (input: IAMCreateAPITokenInput) => iamApi.createAPIToken(currentOrgId ?? "", input),
    onError: (error) => {
      setNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.apiTokens.issue.errorTitle"),
      });
    },
    onSuccess: (result) => {
      setIssuedResult(result);
      setIssueOpen(false);
      setIssueDraft((current) => ({ ...current, remark: "" }));
      setNotice({
        description: t("admin.apiTokens.issue.successDescription", { id: result.item.id }),
        title: t("admin.apiTokens.issue.successTitle"),
      });
      void queryClient.invalidateQueries({ queryKey: apiTokenQueryKey });
    },
  });

  const pageData = apiTokensQuery.data;
  const totalPages = Math.max(1, Math.ceil((pageData?.total ?? 0) / pageSize));
  const currentPageCount = pageData?.items.length ?? 0;
  const statusSummary = summarizeTokens(pageData?.items ?? []);
  const userItems = useMemo(() => usersQuery.data?.items ?? [], [usersQuery.data?.items]);
  const roleItems = useMemo(() => rolesQuery.data ?? [], [rolesQuery.data]);
  const metadataLoading = usersQuery.isLoading || rolesQuery.isLoading;
  const metadataError = usersQuery.error ?? rolesQuery.error ?? null;

  const statusOptions = useMemo<SelectOption[]>(
    () => [
      { label: t("admin.apiTokens.filters.allStatuses"), value: "" },
      { label: t("admin.apiTokens.status.active"), value: "active" },
      { label: t("admin.apiTokens.status.expired"), value: "expired" },
      { label: t("admin.apiTokens.status.revoked"), value: "revoked" },
    ],
    [t],
  );

  const validityOptions = useMemo<SelectOption[]>(
    () => [
      { label: t("admin.apiTokens.validity.days", { days: 7 }), value: "7" },
      { label: t("admin.apiTokens.validity.days", { days: 30 }), value: "30" },
      { label: t("admin.apiTokens.validity.days", { days: 90 }), value: "90" },
      { label: t("admin.apiTokens.validity.days", { days: 180 }), value: "180" },
      { label: t("admin.apiTokens.validity.days", { days: 365 }), value: "365" },
      { label: t("admin.apiTokens.validity.never"), value: "-1" },
    ],
    [t],
  );

  const userOptions = useMemo<SelectOption[]>(
    () =>
      userItems.map((item) => ({
        label: t("admin.apiTokens.issue.userOption", {
          id: item.user.id,
          name: userDisplayName(item),
        }),
        value: String(item.user.id),
      })),
    [t, userItems],
  );

  const selectedUserId = issueDraft.userId || userOptions[0]?.value || "";

  const selectedUser = useMemo(
    () => userItems.find((item) => sameID(item.user.id, selectedUserId)) ?? null,
    [selectedUserId, userItems],
  );

  const roleOptions = useMemo<SelectOption[]>(() => {
    const selectedRoleCodes = new Set((selectedUser?.roles ?? []).map(cleanRoleCode));
    return roleItems
      .filter((role) => selectedRoleCodes.has(role.code))
      .map((role) => ({
        label: t("admin.apiTokens.issue.roleOption", { code: role.code, name: role.name }),
        value: role.code,
      }));
  }, [roleItems, selectedUser?.roles, t]);

  const selectedRoleCode = roleOptions.some((option) => option.value === issueDraft.roleCode)
    ? issueDraft.roleCode
    : roleOptions[0]?.value || "";

  const canIssue = Boolean(
    currentOrgId &&
    selectedUserId &&
    selectedRoleCode &&
    !createTokenMutation.isPending &&
    !metadataLoading,
  );

  const tokenExample = useMemo(() => {
    if (!issuedResult?.token) {
      return "";
    }
    const baseUrl = resolveApiBaseUrl();
    return `curl -H "Authorization: Bearer ${issuedResult.token}" "${baseUrl}${API_ENDPOINTS.me.profile}"`;
  }, [issuedResult?.token]);

  const tokenColumns = useMemo<ColumnDef<IAMAPIToken>[]>(
    () => [
      {
        accessorKey: "id",
        cell: ({ row }) => (
          <div className="aoi-api-token-id">
            <strong>{row.original.id}</strong>
            <span>{row.original.tokenPrefix}</span>
          </div>
        ),
        header: t("admin.apiTokens.columns.id"),
      },
      {
        accessorKey: "userId",
        cell: ({ row }) => (
          <div className="aoi-api-token-user">
            <strong>{displayTokenUser(row.original, t)}</strong>
            <code>{t("admin.apiTokens.labels.userId", { id: row.original.userId })}</code>
          </div>
        ),
        header: t("admin.apiTokens.columns.user"),
      },
      {
        accessorKey: "roleCode",
        cell: ({ row }) => <code className="aoi-api-token-code">{row.original.roleCode}</code>,
        header: t("admin.apiTokens.columns.role"),
      },
      {
        id: "status",
        cell: ({ row }) => {
          const status = apiTokenStatus(row.original);
          return (
            <span className="aoi-iam-status" data-status={status}>
              {statusLabel(status, t)}
            </span>
          );
        },
        header: t("admin.apiTokens.columns.status"),
      },
      {
        accessorKey: "expiresAt",
        cell: ({ row }) => expiresLabel(row.original.expiresAt, i18n.language, t),
        header: t("admin.apiTokens.columns.expiresAt"),
      },
      {
        accessorKey: "lastUsedAt",
        cell: ({ row }) =>
          row.original.lastUsedAt
            ? formatDate(row.original.lastUsedAt, i18n.language, t)
            : t("admin.apiTokens.labels.neverUsed"),
        header: t("admin.apiTokens.columns.lastUsed"),
      },
      {
        accessorKey: "lastUsedIpAddress",
        cell: ({ row }) =>
          row.original.lastUsedIpAddress ? (
            <code className="aoi-api-token-code">{row.original.lastUsedIpAddress}</code>
          ) : (
            t("common.labels.none")
          ),
        header: t("admin.apiTokens.columns.lastUsedIp"),
      },
      {
        accessorKey: "remark",
        cell: ({ row }) => row.original.remark || t("common.labels.none"),
        header: t("admin.apiTokens.columns.remark"),
      },
      {
        id: "actions",
        cell: ({ row }) => {
          const token = row.original;
          return (
            <div className="aoi-api-token-actions">
              <Button
                appearance="secondary"
                aria-label={t("admin.apiTokens.actions.revokeToken", { id: token.id })}
                disabled={apiTokenStatus(token) !== "active" || revokeTokenMutation.isPending}
                icon={<Ban size={16} />}
                onClick={() => setPendingRevokeToken(token)}
              >
                {t("admin.apiTokens.actions.revoke")}
              </Button>
            </div>
          );
        },
        header: t("admin.apiTokens.columns.actions"),
      },
    ],
    [i18n.language, revokeTokenMutation.isPending, t],
  );

  const updateDraft = (key: keyof APITokenFilterDraft, value: string) => {
    setDraft((current) => ({ ...current, [key]: value }));
  };

  const updateIssueDraft = (key: keyof APITokenIssueDraft, value: string) => {
    setIssueDraft((current) => ({
      ...current,
      [key]: value,
      ...(key === "userId" ? { roleCode: "" } : {}),
    }));
  };

  const submitFilters = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFilters(normalizeFilters(draft));
    setPage(1);
    setPageSize(parsePageSize(draft.pageSize));
  };

  const resetFilters = () => {
    setDraft(initialFilterDraft);
    setFilters({});
    setPage(1);
    setPageSize(defaultPageSize);
  };

  const openIssuePanel = () => {
    setNotice(null);
    setIssueOpen(true);
  };

  const closeIssuedPanel = () => {
    setIssuedResult(null);
  };

  const submitIssue = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!currentOrgId) {
      setNotice({
        description: t("admin.apiTokens.states.missingOrgDescription"),
        intent: "danger",
        title: t("admin.apiTokens.states.missingOrgTitle"),
      });
      return;
    }
    const parsed = issueInputSchema.safeParse({
      days: Number(issueDraft.days),
      remark: trimmedOrUndefined(issueDraft.remark),
      roleCode: selectedRoleCode,
      userId: selectedUserId,
    });
    if (!parsed.success) {
      setNotice({
        description: t("admin.apiTokens.issue.validationDescription"),
        intent: "danger",
        title: t("admin.apiTokens.issue.validationTitle"),
      });
      return;
    }
    setNotice(null);
    createTokenMutation.mutate(parsed.data);
  };

  const confirmRevokeToken = () => {
    if (!pendingRevokeToken) {
      return;
    }
    if (!currentOrgId) {
      setPendingRevokeToken(null);
      setNotice({
        description: t("admin.apiTokens.states.missingOrgDescription"),
        intent: "danger",
        title: t("admin.apiTokens.states.missingOrgTitle"),
      });
      return;
    }
    revokeTokenMutation.mutate(pendingRevokeToken);
  };

  const copyIssuedToken = () => {
    void copyText(issuedResult?.token ?? "", {
      deniedDescription: t("admin.apiTokens.copy.tokenDeniedDescription"),
      deniedTitle: t("admin.apiTokens.copy.deniedTitle"),
      successDescription: t("admin.apiTokens.copy.tokenSuccessDescription"),
      successTitle: t("admin.apiTokens.copy.successTitle"),
    }).then(setNotice);
  };

  const copyExample = () => {
    void copyText(tokenExample, {
      deniedDescription: t("admin.apiTokens.copy.exampleDeniedDescription"),
      deniedTitle: t("admin.apiTokens.copy.deniedTitle"),
      successDescription: t("admin.apiTokens.copy.exampleSuccessDescription"),
      successTitle: t("admin.apiTokens.copy.successTitle"),
    }).then(setNotice);
  };

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-api-tokens-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.apiTokens.badge")}</Badge>
          <h1 id="admin-api-tokens-title">{t("admin.apiTokens.title")}</h1>
          <p>{t("admin.apiTokens.description")}</p>
        </div>
        <div className="aoi-api-token-page-actions">
          <Button disabled={!currentOrgId} icon={<Plus size={17} />} onClick={openIssuePanel}>
            {t("admin.apiTokens.actions.issue")}
          </Button>
          <Button
            appearance="secondary"
            disabled={!currentOrgId}
            icon={<RefreshCw size={17} />}
            loading={apiTokensQuery.isFetching}
            onClick={() => void apiTokensQuery.refetch()}
          >
            {t("admin.apiTokens.actions.refresh")}
          </Button>
        </div>
      </div>

      {!currentOrgId ? (
        <StateBlock
          intent="danger"
          title={t("admin.apiTokens.states.missingOrgTitle")}
          description={t("admin.apiTokens.states.missingOrgDescription")}
        />
      ) : null}

      {apiTokensQuery.error ? (
        <StateBlock
          intent="danger"
          title={errorTitle(apiTokensQuery.error, t)}
          description={errorDescription(apiTokensQuery.error, t)}
        />
      ) : null}

      {notice ? (
        <StateBlock description={notice.description} intent={notice.intent} title={notice.title} />
      ) : null}

      <Dialog
        closeLabel={t("admin.apiTokens.actions.cancelRevoke")}
        description={
          pendingRevokeToken
            ? t("admin.apiTokens.revoke.confirmDescription", {
                id: pendingRevokeToken.id,
                prefix: pendingRevokeToken.tokenPrefix,
                user: displayTokenUser(pendingRevokeToken, t),
              })
            : undefined
        }
        footer={
          <div className="aoi-api-token-confirm-actions">
            <Button loading={revokeTokenMutation.isPending} onClick={confirmRevokeToken}>
              {t("admin.apiTokens.actions.confirmRevoke")}
            </Button>
            <Button
              appearance="secondary"
              disabled={revokeTokenMutation.isPending}
              onClick={() => setPendingRevokeToken(null)}
            >
              {t("admin.apiTokens.actions.cancelRevoke")}
            </Button>
          </div>
        }
        open={Boolean(pendingRevokeToken)}
        title={t("admin.apiTokens.revoke.confirmTitle")}
        onOpenChange={(open) => {
          if (!open && !revokeTokenMutation.isPending) {
            setPendingRevokeToken(null);
          }
        }}
      />

      <div className="aoi-admin-stat-grid" aria-label={t("admin.apiTokens.summaryLabel")}>
        <APITokenStatCard
          icon={<KeyRound size={19} />}
          label={t("admin.apiTokens.metrics.total")}
          value={
            pageData
              ? formatNumber(pageData.total, i18n.language)
              : fallbackValue(apiTokensQuery.isLoading, t)
          }
        />
        <APITokenStatCard
          icon={<Users size={19} />}
          label={t("admin.apiTokens.metrics.currentPage")}
          value={formatNumber(currentPageCount, i18n.language)}
        />
        <APITokenStatCard
          icon={<Clock3 size={19} />}
          label={t("admin.apiTokens.metrics.active")}
          value={formatNumber(statusSummary.active, i18n.language)}
        />
        <APITokenStatCard
          icon={<ShieldAlert size={19} />}
          label={t("admin.apiTokens.metrics.inactive")}
          value={formatNumber(statusSummary.inactive, i18n.language)}
        />
        <APITokenStatCard
          icon={<Database size={19} />}
          label={t("admin.apiTokens.metrics.storage")}
          value={
            pageData
              ? storageStatusLabel(pageData.storageStatus, t)
              : fallbackValue(apiTokensQuery.isLoading, t)
          }
        />
      </div>

      {issueOpen ? (
        <section className="aoi-admin-panel">
          <header>
            <h2>{t("admin.apiTokens.issue.title")}</h2>
            <p>{t("admin.apiTokens.issue.description")}</p>
          </header>
          {metadataError ? (
            <StateBlock
              intent="danger"
              title={errorTitle(metadataError, t)}
              description={errorDescription(metadataError, t)}
            />
          ) : null}
          {selectedUserId && !metadataLoading && roleOptions.length === 0 ? (
            <StateBlock
              title={t("admin.apiTokens.issue.noAssignableRoleTitle")}
              description={t("admin.apiTokens.issue.noAssignableRoleDescription")}
            />
          ) : null}
          <form className="aoi-api-token-issue-form" onSubmit={submitIssue}>
            <SelectField
              disabled={metadataLoading || createTokenMutation.isPending}
              label={t("admin.apiTokens.issue.user")}
              options={userOptions}
              value={selectedUserId}
              onChange={(event) => updateIssueDraft("userId", event.currentTarget.value)}
            />
            <SelectField
              disabled={!selectedUserId || metadataLoading || createTokenMutation.isPending}
              label={t("admin.apiTokens.issue.role")}
              options={roleOptions}
              value={selectedRoleCode}
              onChange={(event) => updateIssueDraft("roleCode", event.currentTarget.value)}
            />
            <SelectField
              disabled={createTokenMutation.isPending}
              label={t("admin.apiTokens.issue.validity")}
              options={validityOptions}
              value={issueDraft.days}
              onChange={(event) => updateIssueDraft("days", event.currentTarget.value)}
            />
            <div className="aoi-form-field aoi-api-token-remark-field">
              <label htmlFor="api-token-remark">{t("admin.apiTokens.issue.remark")}</label>
              <textarea
                id="api-token-remark"
                disabled={createTokenMutation.isPending}
                maxLength={500}
                rows={3}
                value={issueDraft.remark}
                onChange={(event) => updateIssueDraft("remark", event.currentTarget.value)}
              />
              <span className="aoi-form-field__help">{t("admin.apiTokens.issue.remarkHelp")}</span>
            </div>
            <div className="aoi-api-token-issue-actions">
              <Button
                disabled={!canIssue}
                icon={<KeyRound size={17} />}
                loading={createTokenMutation.isPending}
                type="submit"
              >
                {t("admin.apiTokens.actions.create")}
              </Button>
              <Button
                appearance="secondary"
                disabled={createTokenMutation.isPending}
                onClick={() => setIssueOpen(false)}
              >
                {t("admin.apiTokens.actions.cancelIssue")}
              </Button>
            </div>
          </form>
        </section>
      ) : null}

      {issuedResult ? (
        <section className="aoi-admin-panel aoi-api-token-issued-panel">
          <header>
            <h2>{t("admin.apiTokens.issued.title")}</h2>
            <p>{t("admin.apiTokens.issued.description")}</p>
          </header>
          <StateBlock
            title={t("admin.apiTokens.issued.warningTitle")}
            description={t("admin.apiTokens.issued.warningDescription")}
          />
          <div className="aoi-api-token-issued-grid">
            <div className="aoi-api-token-issued-block">
              <span>{t("admin.apiTokens.issued.fullToken")}</span>
              <code>{issuedResult.token}</code>
            </div>
            <div className="aoi-api-token-issued-block">
              <span>{t("admin.apiTokens.issued.example")}</span>
              <code>{tokenExample}</code>
            </div>
          </div>
          <div className="aoi-api-token-issued-meta">
            <span>{t("admin.apiTokens.labels.tokenId", { id: issuedResult.item.id })}</span>
            <span>{issuedResult.item.roleCode}</span>
            <span>{expiresLabel(issuedResult.item.expiresAt, i18n.language, t)}</span>
          </div>
          <div className="aoi-api-token-issued-actions">
            <Button appearance="secondary" icon={<Copy size={17} />} onClick={copyIssuedToken}>
              {t("admin.apiTokens.actions.copyToken")}
            </Button>
            <Button appearance="secondary" icon={<Terminal size={17} />} onClick={copyExample}>
              {t("admin.apiTokens.actions.copyExample")}
            </Button>
            <Button onClick={closeIssuedPanel}>{t("admin.apiTokens.actions.closeIssued")}</Button>
          </div>
        </section>
      ) : null}

      <section className="aoi-admin-panel">
        <header>
          <h2>{t("admin.apiTokens.filters.title")}</h2>
          <p>{t("admin.apiTokens.filters.description")}</p>
        </header>
        <form className="aoi-admin-filter-form aoi-api-token-filter-form" onSubmit={submitFilters}>
          <FormField
            label={t("admin.apiTokens.filters.userId")}
            min={1}
            type="number"
            value={draft.userId}
            onChange={(event) => updateDraft("userId", event.currentTarget.value)}
          />
          <SelectField
            label={t("admin.apiTokens.filters.status")}
            options={statusOptions}
            value={draft.status}
            onChange={(event) => updateDraft("status", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.apiTokens.filters.pageSize")}
            max={100}
            min={1}
            type="number"
            value={draft.pageSize}
            onChange={(event) => updateDraft("pageSize", event.currentTarget.value)}
          />
          <div className="aoi-admin-filter-actions">
            <Button icon={<Search size={17} />} loading={apiTokensQuery.isFetching} type="submit">
              {t("admin.apiTokens.actions.search")}
            </Button>
            <Button appearance="secondary" icon={<RotateCcw size={17} />} onClick={resetFilters}>
              {t("admin.apiTokens.actions.reset")}
            </Button>
          </div>
        </form>
      </section>

      <section className="aoi-admin-panel">
        <header className="aoi-admin-panel-header-row">
          <div>
            <h2>{t("admin.apiTokens.list.title")}</h2>
            <p>{t("admin.apiTokens.list.description", { count: pageData?.total ?? 0 })}</p>
          </div>
          <div className="aoi-admin-pager" aria-label={t("admin.apiTokens.pagination.label")}>
            <Button
              appearance="secondary"
              disabled={page <= 1 || apiTokensQuery.isFetching}
              icon={<ChevronLeft size={17} />}
              onClick={() => setPage((current) => Math.max(1, current - 1))}
            >
              {t("admin.apiTokens.pagination.previous")}
            </Button>
            <span>{t("admin.apiTokens.pagination.pageStatus", { page, totalPages })}</span>
            <Button
              appearance="secondary"
              disabled={page >= totalPages || apiTokensQuery.isFetching}
              icon={<ChevronRight size={17} />}
              onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
            >
              {t("admin.apiTokens.pagination.next")}
            </Button>
          </div>
        </header>

        {apiTokensQuery.isLoading ? (
          <TableSkeleton
            caption={t("admin.apiTokens.states.loadingDescription")}
            columns={8}
            rows={pageSize}
          />
        ) : pageData ? (
          <>
            {pageData.storageStatus === "persisted" ? null : (
              <StateBlock
                title={t("admin.apiTokens.states.storageUnavailableTitle")}
                description={t("admin.apiTokens.states.storageUnavailableDescription")}
              />
            )}
            <div className="aoi-api-token-table">
              <DataTable
                columns={tokenColumns}
                data={pageData.items}
                emptyLabel={t("admin.apiTokens.empty")}
              />
            </div>
          </>
        ) : (
          <StateBlock
            title={t("admin.apiTokens.states.emptyTitle")}
            description={t("admin.apiTokens.states.emptyDescription")}
          />
        )}
      </section>
    </section>
  );
}

type APITokenStatCardProps = {
  icon: ReactNode;
  label: string;
  value: string;
};

function APITokenStatCard({ icon, label, value }: APITokenStatCardProps) {
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

function normalizeFilters(draft: APITokenFilterDraft): APITokenFilters {
  return {
    status: trimmedOrUndefined(draft.status),
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

function parsePageSize(value: string) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return defaultPageSize;
  }
  return Math.min(100, Math.max(1, Math.trunc(parsed)));
}

function summarizeTokens(items: IAMAPIToken[]) {
  return items.reduce(
    (summary, token) => {
      if (apiTokenStatus(token) === "active") {
        summary.active += 1;
      } else {
        summary.inactive += 1;
      }
      return summary;
    },
    { active: 0, inactive: 0 },
  );
}

function apiTokenStatus(token: IAMAPIToken) {
  if (token.revokedAt || token.status === "revoked") {
    return "revoked";
  }
  if (token.expiresAt) {
    const expiresAt = Date.parse(token.expiresAt);
    if (Number.isFinite(expiresAt) && expiresAt <= Date.now()) {
      return "expired";
    }
  }
  return token.status || "active";
}

function statusLabel(status: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (status === "active" || status === "expired" || status === "revoked") {
    return t(`admin.apiTokens.status.${status}`);
  }
  return status;
}

function expiresLabel(
  value: string | null | undefined,
  locale: string,
  t: ReturnType<typeof useTranslation>["t"],
) {
  return value ? formatDate(value, locale, t) : t("admin.apiTokens.validity.never");
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

function fallbackValue(loading: boolean, t: ReturnType<typeof useTranslation>["t"]) {
  return loading ? t("loading.app") : t("common.labels.none");
}

function storageStatusLabel(status: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (status === "persisted") {
    return t("admin.apiTokens.storage.persisted");
  }
  if (status === "unavailable") {
    return t("admin.apiTokens.storage.unavailable");
  }
  return status || t("admin.apiTokens.storage.unknown");
}

function userDisplayName(item: IAMOrganizationUser) {
  return item.user.displayName || item.user.username || String(item.user.id);
}

function displayTokenUser(token: IAMAPIToken, t: ReturnType<typeof useTranslation>["t"]) {
  return (
    token.userDisplayName ||
    token.username ||
    t("admin.apiTokens.labels.userId", { id: token.userId })
  );
}

function cleanRoleCode(value: string) {
  return value.replace(/^role:/, "");
}

function sameID(left: number | string, right: number | string) {
  return String(left) === String(right);
}

async function copyText(
  value: string,
  messages: {
    deniedDescription: string;
    deniedTitle: string;
    successDescription: string;
    successTitle: string;
  },
): Promise<TokenNotice> {
  if (!value || typeof navigator === "undefined" || !navigator.clipboard) {
    return {
      description: messages.deniedDescription,
      intent: "danger",
      title: messages.deniedTitle,
    };
  }
  try {
    await navigator.clipboard.writeText(value);
    return {
      description: messages.successDescription,
      title: messages.successTitle,
    };
  } catch {
    return {
      description: messages.deniedDescription,
      intent: "danger",
      title: messages.deniedTitle,
    };
  }
}

function errorTitle(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.apiTokens.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.apiTokens.states.errorTitle");
}

function errorDescription(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.apiTokens.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

function toError(error: unknown) {
  return error instanceof Error ? error : new Error(String(error));
}

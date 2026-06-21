import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import {
  Ban,
  ChevronLeft,
  ChevronRight,
  Database,
  Mail,
  RefreshCw,
  RotateCcw,
  Save,
  Search,
  Send,
  ShieldCheck,
  UserPlus,
  Users,
} from "lucide-react";
import { useMemo, useState, type FormEvent, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { z } from "zod";

import { DataTable } from "~/components/aoi/patterns/DataTable";
import { Collapse } from "~/components/aoi/patterns/Collapse";
import { Dialog } from "~/components/aoi/patterns/Dialog";
import { Drawer } from "~/components/aoi/patterns/Drawer";
import { FormField } from "~/components/aoi/patterns/FormField";
import { TableSkeleton } from "~/components/aoi/patterns/LoadingSkeletons";
import { SelectField, type SelectOption } from "~/components/aoi/patterns/SelectField";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { Tooltip } from "~/components/aoi/primitives/Tooltip";
import { ApiError } from "~/lib/api/client";
import { iamApi, type IAMUserListQuery } from "~/lib/api/iam";
import { queryKeys } from "~/lib/api/query-keys";
import type {
  IAMInvitation,
  IAMOrganizationUser,
  IAMRole,
  NotificationDelivery,
} from "~/lib/api/types";
import { useAuthStore } from "~/stores/auth-store";

const defaultPageSize = 10;

const inviteInputSchema = z.object({
  email: z.string().trim().email(),
  roleCode: z.string().trim().min(1),
});

type UserFilters = Pick<
  IAMUserListQuery,
  "displayName" | "email" | "keyword" | "roleCode" | "status" | "username"
> & {
  desc?: boolean;
  orderKey?: string;
};

type UserFilterDraft = {
  displayName: string;
  email: string;
  keyword: string;
  pageSize: string;
  roleCode: string;
  status: string;
  username: string;
};

type InviteDraft = {
  email: string;
  roleCode: string;
};

type UserNotice = {
  description: string;
  intent?: "danger" | "info";
  title: string;
};

type MemberMutationInput = {
  item: IAMOrganizationUser;
  roles?: string[];
  status?: "active" | "disabled";
};

const initialFilterDraft: UserFilterDraft = {
  displayName: "",
  email: "",
  keyword: "",
  pageSize: String(defaultPageSize),
  roleCode: "",
  status: "",
  username: "",
};

const initialInviteDraft: InviteDraft = {
  email: "",
  roleCode: "",
};

export default function AdminUsersRoute() {
  const { i18n, t } = useTranslation();
  const queryClient = useQueryClient();
  const currentOrgId = useAuthStore((state) => state.currentOrgId);
  const [draft, setDraft] = useState<UserFilterDraft>(initialFilterDraft);
  const [filters, setFilters] = useState<UserFilters>({
    desc: true,
    orderKey: "id",
  });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(defaultPageSize);
  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteDraft, setInviteDraft] = useState<InviteDraft>(initialInviteDraft);
  const [roleDrafts, setRoleDrafts] = useState<Record<string, string>>({});
  const [notice, setNotice] = useState<UserNotice | null>(null);
  const [delivery, setDelivery] = useState<NotificationDelivery | null>(null);
  const [pendingRevokeInvitation, setPendingRevokeInvitation] = useState<IAMInvitation | null>(
    null,
  );

  const userQueryKey = queryKeys.iam.users(
    i18n.language,
    currentOrgId ?? "",
    page,
    pageSize,
    filters,
  );
  const invitationQueryKey = queryKeys.iam.invitations(i18n.language, currentOrgId ?? "");

  const usersQuery = useQuery({
    enabled: Boolean(currentOrgId),
    queryFn: ({ signal }) =>
      iamApi.listUsers(currentOrgId ?? "", { ...filters, page, pageSize }, { signal }),
    queryKey: userQueryKey,
  });

  const rolesQuery = useQuery({
    enabled: Boolean(currentOrgId),
    queryFn: ({ signal }) => iamApi.listRoles(currentOrgId ?? "", { signal }),
    queryKey: queryKeys.iam.roles(i18n.language, currentOrgId ?? ""),
  });

  const invitationsQuery = useQuery({
    enabled: Boolean(currentOrgId),
    queryFn: ({ signal }) => iamApi.listInvitations(currentOrgId ?? "", { signal }),
    queryKey: invitationQueryKey,
  });

  const updateMemberMutation = useMutation({
    mutationFn: ({ item, roles, status }: MemberMutationInput) =>
      iamApi.updateUser(currentOrgId ?? "", item.user.id, { roles, status }),
    onError: (error) => {
      setNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.users.member.errorTitle"),
      });
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: userQueryKey });
    },
    onSuccess: (_result, input) => {
      setNotice({
        description: input.roles
          ? t("admin.users.member.roleSuccessDescription", {
              name: userDisplayName(input.item),
            })
          : t("admin.users.member.statusSuccessDescription", {
              name: userDisplayName(input.item),
              status: statusLabel(input.status ?? input.item.membershipStatus, t),
            }),
        title: input.roles
          ? t("admin.users.member.roleSuccessTitle")
          : t("admin.users.member.statusSuccessTitle"),
      });
    },
  });

  const inviteUserMutation = useMutation({
    mutationFn: (input: InviteDraft) => iamApi.inviteUser(currentOrgId ?? "", input),
    onError: (error) => {
      setNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.users.invite.errorTitle"),
      });
    },
    onSuccess: (result, input) => {
      setDelivery(result.debug ? result : null);
      setInviteOpen(false);
      setInviteDraft((current) => ({ ...current, email: "" }));
      setNotice({
        description: result.debug
          ? t("admin.users.invite.debugSuccessDescription", { email: input.email })
          : t("admin.users.invite.successDescription", { email: input.email }),
        title: t("admin.users.invite.successTitle"),
      });
      void queryClient.invalidateQueries({ queryKey: invitationQueryKey });
      void queryClient.invalidateQueries({ queryKey: userQueryKey });
    },
  });

  const revokeInvitationMutation = useMutation({
    mutationFn: (invitation: IAMInvitation) =>
      iamApi.revokeInvitation(currentOrgId ?? "", invitation.id),
    onError: (error) => {
      setNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.users.invitation.errorTitle"),
      });
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: invitationQueryKey });
    },
    onSuccess: (_result, invitation) => {
      setPendingRevokeInvitation(null);
      setNotice({
        description: t("admin.users.invitation.revokeSuccessDescription", {
          email: invitation.email,
        }),
        title: t("admin.users.invitation.revokeSuccessTitle"),
      });
    },
  });

  const pageData = usersQuery.data;
  const userItems = useMemo(() => pageData?.items ?? [], [pageData?.items]);
  const roleItems = useMemo(() => rolesQuery.data ?? [], [rolesQuery.data]);
  const invitationItems = useMemo(() => invitationsQuery.data ?? [], [invitationsQuery.data]);
  const fetching = usersQuery.isFetching || rolesQuery.isFetching || invitationsQuery.isFetching;
  const firstError = usersQuery.error ?? rolesQuery.error ?? invitationsQuery.error ?? null;
  const totalPages = Math.max(1, Math.ceil((pageData?.total ?? 0) / pageSize));
  const statusSummary = summarizeMembers(userItems);
  const pendingInvitationCount = invitationItems.filter(
    (invitation) => invitationStatus(invitation) === "pending",
  ).length;

  const roleOptions = useMemo<SelectOption[]>(
    () =>
      roleItems.map((role) => ({
        label: roleLabel(role, t),
        value: role.code,
      })),
    [roleItems, t],
  );
  const assignableRoleOptions = useMemo<SelectOption[]>(
    () =>
      roleOptions.length
        ? roleOptions
        : [{ label: t("admin.users.roles.noneAvailable"), value: "" }],
    [roleOptions, t],
  );
  const selectedInviteRole = inviteDraft.roleCode || roleOptions[0]?.value || "";

  const roleFilterOptions = useMemo<SelectOption[]>(
    () => [{ label: t("admin.users.filters.allRoles"), value: "" }, ...roleOptions],
    [roleOptions, t],
  );
  const statusOptions = useMemo<SelectOption[]>(
    () => [
      { label: t("admin.users.filters.allStatuses"), value: "" },
      { label: t("admin.users.status.active"), value: "active" },
      { label: t("admin.users.status.disabled"), value: "disabled" },
    ],
    [t],
  );

  const userColumns = useMemo<ColumnDef<IAMOrganizationUser>[]>(
    () => [
      {
        accessorKey: "user.id",
        cell: ({ row }) => (
          <div className="aoi-user-id">
            <strong>{row.original.user.id}</strong>
            <span>{row.original.user.username}</span>
          </div>
        ),
        header: t("admin.users.columns.id"),
      },
      {
        accessorKey: "user.displayName",
        cell: ({ row }) => (
          <div className="aoi-user-principal">
            <strong>{userDisplayName(row.original)}</strong>
            <span>{row.original.user.email}</span>
          </div>
        ),
        header: t("admin.users.columns.user"),
      },
      {
        accessorKey: "roles",
        cell: ({ row }) => <RoleList roles={row.original.roles} />,
        header: t("admin.users.columns.roles"),
      },
      {
        id: "membershipStatus",
        cell: ({ row }) => {
          const status = row.original.membershipStatus;
          return (
            <span className="aoi-iam-status" data-status={status}>
              {statusLabel(status, t)}
            </span>
          );
        },
        header: t("admin.users.columns.memberStatus"),
      },
      {
        accessorKey: "user.status",
        cell: ({ row }) => {
          const status = userAccountStatus(row.original);
          return (
            <span className="aoi-iam-status" data-status={status}>
              {statusLabel(status, t)}
            </span>
          );
        },
        header: t("admin.users.columns.accountStatus"),
      },
      {
        accessorKey: "user.mfaEnabled",
        cell: ({ row }) =>
          row.original.user.mfaEnabled
            ? t("admin.users.mfa.enabled")
            : t("admin.users.mfa.disabled"),
        header: t("admin.users.columns.mfa"),
      },
      {
        accessorKey: "user.lastLoginAt",
        cell: ({ row }) => formatDate(row.original.user.lastLoginAt, i18n.language, t),
        header: t("admin.users.columns.lastLogin"),
      },
      {
        id: "actions",
        cell: ({ row }) => {
          const item = row.original;
          const userId = String(item.user.id);
          const selectedRole = roleDrafts[userId] ?? firstRoleCode(item.roles);
          const nextStatus = item.membershipStatus === "active" ? "disabled" : "active";
          return (
            <div className="aoi-user-actions">
              <SelectField
                aria-label={t("admin.users.actions.roleSelect", { name: userDisplayName(item) })}
                disabled={updateMemberMutation.isPending || roleOptions.length === 0}
                label={t("admin.users.fields.role")}
                options={assignableRoleOptions}
                value={selectedRole}
                onChange={(event) => {
                  const value = event.currentTarget.value;
                  setRoleDrafts((current) => ({
                    ...current,
                    [userId]: value,
                  }));
                }}
              />
              <Button
                appearance="secondary"
                aria-label={t("admin.users.actions.saveRoleFor", {
                  name: userDisplayName(item),
                })}
                disabled={
                  updateMemberMutation.isPending ||
                  !selectedRole ||
                  selectedRole === firstRoleCode(item.roles)
                }
                icon={<Save size={16} />}
                onClick={() => updateMemberMutation.mutate({ item, roles: [selectedRole] })}
              >
                {t("admin.users.actions.saveRole")}
              </Button>
              <Button
                appearance="secondary"
                aria-label={t("admin.users.actions.toggleStatusFor", {
                  name: userDisplayName(item),
                  status: statusLabel(nextStatus, t),
                })}
                disabled={updateMemberMutation.isPending}
                icon={nextStatus === "active" ? <ShieldCheck size={16} /> : <Ban size={16} />}
                onClick={() => updateMemberMutation.mutate({ item, status: nextStatus })}
              >
                {nextStatus === "active"
                  ? t("admin.users.actions.enable")
                  : t("admin.users.actions.disable")}
              </Button>
            </div>
          );
        },
        header: t("admin.users.columns.actions"),
      },
    ],
    [assignableRoleOptions, i18n.language, roleDrafts, roleOptions.length, t, updateMemberMutation],
  );

  const invitationColumns = useMemo<ColumnDef<IAMInvitation>[]>(
    () => [
      {
        accessorKey: "email",
        cell: ({ row }) => (
          <div className="aoi-user-principal">
            <strong>{row.original.email}</strong>
            <span>{t("admin.users.invitation.invitedBy", { id: row.original.invitedBy })}</span>
          </div>
        ),
        header: t("admin.users.invitation.columns.email"),
      },
      {
        accessorKey: "roleCode",
        cell: ({ row }) => <code className="aoi-user-code">{row.original.roleCode}</code>,
        header: t("admin.users.invitation.columns.role"),
      },
      {
        id: "status",
        cell: ({ row }) => {
          const status = invitationStatus(row.original);
          return (
            <span className="aoi-iam-status" data-status={status}>
              {statusLabel(status, t)}
            </span>
          );
        },
        header: t("admin.users.invitation.columns.status"),
      },
      {
        accessorKey: "expiresAt",
        cell: ({ row }) => formatDate(row.original.expiresAt, i18n.language, t),
        header: t("admin.users.invitation.columns.expiresAt"),
      },
      {
        id: "actions",
        cell: ({ row }) => {
          const invitation = row.original;
          const status = invitationStatus(invitation);
          return (
            <div className="aoi-user-invitation-actions">
              <Button
                appearance="secondary"
                aria-label={t("admin.users.invitation.revokeInvitation", {
                  email: invitation.email,
                })}
                disabled={status !== "pending" || revokeInvitationMutation.isPending}
                icon={<Ban size={16} />}
                onClick={() => setPendingRevokeInvitation(invitation)}
              >
                {t("admin.users.actions.revoke")}
              </Button>
            </div>
          );
        },
        header: t("admin.users.columns.actions"),
      },
    ],
    [i18n.language, revokeInvitationMutation.isPending, t],
  );

  const updateFilterDraft = (key: keyof UserFilterDraft, value: string) => {
    setDraft((current) => ({ ...current, [key]: value }));
  };

  const updateInviteDraft = (key: keyof InviteDraft, value: string) => {
    setInviteDraft((current) => ({ ...current, [key]: value }));
  };

  const submitFilters = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFilters(normalizeFilters(draft));
    setPage(1);
    setPageSize(parsePageSize(draft.pageSize));
  };

  const resetFilters = () => {
    setDraft(initialFilterDraft);
    setFilters({ desc: true, orderKey: "id" });
    setPage(1);
    setPageSize(defaultPageSize);
  };

  const submitInvite = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!currentOrgId) {
      setNotice({
        description: t("admin.users.states.missingOrgDescription"),
        intent: "danger",
        title: t("admin.users.states.missingOrgTitle"),
      });
      return;
    }
    const parsed = inviteInputSchema.safeParse({
      email: inviteDraft.email,
      roleCode: selectedInviteRole,
    });
    if (!parsed.success) {
      setNotice({
        description: t("admin.users.invite.validationDescription"),
        intent: "danger",
        title: t("admin.users.invite.validationTitle"),
      });
      return;
    }
    setDelivery(null);
    setNotice(null);
    inviteUserMutation.mutate(parsed.data);
  };

  const confirmRevokeInvitation = () => {
    if (!pendingRevokeInvitation) {
      return;
    }
    if (!currentOrgId) {
      setPendingRevokeInvitation(null);
      setNotice({
        description: t("admin.users.states.missingOrgDescription"),
        intent: "danger",
        title: t("admin.users.states.missingOrgTitle"),
      });
      return;
    }
    revokeInvitationMutation.mutate(pendingRevokeInvitation);
  };

  const refresh = () => {
    void usersQuery.refetch();
    void rolesQuery.refetch();
    void invitationsQuery.refetch();
  };

  const inviteButton = (
    <Button
      disabled={!currentOrgId}
      icon={<UserPlus size={17} />}
      onClick={() => setInviteOpen(true)}
    >
      {t("admin.users.actions.invite")}
    </Button>
  );

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-users-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.users.badge")}</Badge>
          <h1 id="admin-users-title">{t("admin.users.title")}</h1>
          <p>{t("admin.users.description")}</p>
        </div>
        <div className="aoi-user-page-actions">
          {!currentOrgId ? (
            <Tooltip content={t("admin.users.states.missingOrgDescription")}>
              <span className="aoi-tooltip-anchor">{inviteButton}</span>
            </Tooltip>
          ) : (
            inviteButton
          )}
          <Button
            appearance="secondary"
            disabled={!currentOrgId}
            icon={<RefreshCw size={17} />}
            loading={fetching}
            onClick={refresh}
          >
            {t("admin.users.actions.refresh")}
          </Button>
        </div>
      </div>

      {!currentOrgId ? (
        <StateBlock
          intent="danger"
          title={t("admin.users.states.missingOrgTitle")}
          description={t("admin.users.states.missingOrgDescription")}
        />
      ) : null}

      {firstError ? (
        <StateBlock
          intent="danger"
          title={errorTitle(firstError, t)}
          description={errorDescription(firstError, t)}
        />
      ) : null}

      {notice ? (
        <StateBlock description={notice.description} intent={notice.intent} title={notice.title} />
      ) : null}

      {delivery?.debug ? (
        <section className="aoi-admin-panel aoi-user-delivery-panel">
          <header>
            <h2>{t("admin.users.invite.debugDeliveryTitle")}</h2>
            <p>{t("admin.users.invite.debugDeliveryDescription")}</p>
          </header>
          <div className="aoi-user-delivery-grid">
            {delivery.token ? (
              <div>
                <span>{t("admin.users.invite.debugToken")}</span>
                <code>{delivery.token}</code>
              </div>
            ) : null}
            {delivery.url ? (
              <div>
                <span>{t("admin.users.invite.debugUrl")}</span>
                <code>{delivery.url}</code>
              </div>
            ) : null}
          </div>
        </section>
      ) : null}

      <Dialog
        closeLabel={t("admin.users.actions.cancel")}
        description={
          pendingRevokeInvitation
            ? t("admin.users.invitation.confirmRevokeDescription", {
                email: pendingRevokeInvitation.email,
              })
            : undefined
        }
        footer={
          <div className="aoi-user-confirm-actions">
            <Button loading={revokeInvitationMutation.isPending} onClick={confirmRevokeInvitation}>
              {t("admin.users.actions.confirmRevoke")}
            </Button>
            <Button
              appearance="secondary"
              disabled={revokeInvitationMutation.isPending}
              onClick={() => setPendingRevokeInvitation(null)}
            >
              {t("admin.users.actions.cancel")}
            </Button>
          </div>
        }
        open={Boolean(pendingRevokeInvitation)}
        title={t("admin.users.invitation.confirmRevokeTitle")}
        onOpenChange={(open) => {
          if (!open && !revokeInvitationMutation.isPending) {
            setPendingRevokeInvitation(null);
          }
        }}
      />

      <Drawer
        closeLabel={t("admin.users.actions.cancel")}
        description={t("admin.users.invite.description")}
        open={inviteOpen}
        title={t("admin.users.invite.title")}
        onOpenChange={(open) => {
          if (!inviteUserMutation.isPending) {
            setInviteOpen(open);
          }
        }}
      >
        <form className="aoi-user-invite-form" onSubmit={submitInvite}>
          <FormField
            disabled={inviteUserMutation.isPending}
            label={t("admin.users.invite.email")}
            placeholder={t("admin.users.invite.emailPlaceholder")}
            type="email"
            value={inviteDraft.email}
            onChange={(event) => updateInviteDraft("email", event.currentTarget.value)}
          />
          <SelectField
            disabled={inviteUserMutation.isPending || roleOptions.length === 0}
            label={t("admin.users.invite.role")}
            options={assignableRoleOptions}
            value={selectedInviteRole}
            onChange={(event) => updateInviteDraft("roleCode", event.currentTarget.value)}
          />
          <div className="aoi-user-invite-actions">
            <Button
              disabled={!selectedInviteRole}
              icon={<Send size={17} />}
              loading={inviteUserMutation.isPending}
              type="submit"
            >
              {t("admin.users.actions.sendInvitation")}
            </Button>
            <Button
              appearance="secondary"
              disabled={inviteUserMutation.isPending}
              onClick={() => setInviteOpen(false)}
            >
              {t("admin.users.actions.cancel")}
            </Button>
          </div>
        </form>
      </Drawer>

      <div className="aoi-admin-stat-grid" aria-label={t("admin.users.summaryLabel")}>
        <UserStatCard
          icon={<Users size={19} />}
          label={t("admin.users.metrics.total")}
          value={
            pageData
              ? formatNumber(pageData.total, i18n.language)
              : fallbackValue(usersQuery.isLoading, t)
          }
        />
        <UserStatCard
          icon={<ShieldCheck size={19} />}
          label={t("admin.users.metrics.active")}
          value={formatNumber(statusSummary.active, i18n.language)}
        />
        <UserStatCard
          icon={<Ban size={19} />}
          label={t("admin.users.metrics.disabled")}
          value={formatNumber(statusSummary.disabled, i18n.language)}
        />
        <UserStatCard
          icon={<Mail size={19} />}
          label={t("admin.users.metrics.pendingInvitations")}
          value={formatNumber(pendingInvitationCount, i18n.language)}
        />
        <UserStatCard
          icon={<Database size={19} />}
          label={t("admin.users.metrics.storage")}
          value={
            pageData
              ? storageStatusLabel(pageData.storageStatus, t)
              : fallbackValue(usersQuery.isLoading, t)
          }
        />
      </div>

      <Collapse
        description={t("admin.users.filters.description")}
        title={t("admin.users.filters.title")}
      >
        <form className="aoi-admin-filter-form aoi-user-filter-form" onSubmit={submitFilters}>
          <FormField
            label={t("admin.users.filters.keyword")}
            value={draft.keyword}
            onChange={(event) => updateFilterDraft("keyword", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.users.filters.username")}
            value={draft.username}
            onChange={(event) => updateFilterDraft("username", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.users.filters.displayName")}
            value={draft.displayName}
            onChange={(event) => updateFilterDraft("displayName", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.users.filters.email")}
            type="email"
            value={draft.email}
            onChange={(event) => updateFilterDraft("email", event.currentTarget.value)}
          />
          <SelectField
            label={t("admin.users.filters.role")}
            options={roleFilterOptions}
            value={draft.roleCode}
            onChange={(event) => updateFilterDraft("roleCode", event.currentTarget.value)}
          />
          <SelectField
            label={t("admin.users.filters.status")}
            options={statusOptions}
            value={draft.status}
            onChange={(event) => updateFilterDraft("status", event.currentTarget.value)}
          />
          <FormField
            label={t("admin.users.filters.pageSize")}
            max={100}
            min={1}
            type="number"
            value={draft.pageSize}
            onChange={(event) => updateFilterDraft("pageSize", event.currentTarget.value)}
          />
          <div className="aoi-admin-filter-actions">
            <Button icon={<Search size={17} />} loading={usersQuery.isFetching} type="submit">
              {t("admin.users.actions.search")}
            </Button>
            <Button appearance="secondary" icon={<RotateCcw size={17} />} onClick={resetFilters}>
              {t("admin.users.actions.reset")}
            </Button>
          </div>
        </form>
      </Collapse>

      <section className="aoi-admin-panel">
        <header className="aoi-admin-panel-header-row">
          <div>
            <h2>{t("admin.users.list.title")}</h2>
            <p>{t("admin.users.list.description", { count: pageData?.total ?? 0 })}</p>
          </div>
          <div className="aoi-admin-pager" aria-label={t("admin.users.pagination.label")}>
            <Button
              appearance="secondary"
              disabled={page <= 1 || usersQuery.isFetching}
              icon={<ChevronLeft size={17} />}
              onClick={() => setPage((current) => Math.max(1, current - 1))}
            >
              {t("admin.users.pagination.previous")}
            </Button>
            <span>{t("admin.users.pagination.pageStatus", { page, totalPages })}</span>
            <Button
              appearance="secondary"
              disabled={page >= totalPages || usersQuery.isFetching}
              icon={<ChevronRight size={17} />}
              onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
            >
              {t("admin.users.pagination.next")}
            </Button>
          </div>
        </header>

        {usersQuery.isLoading ? (
          <TableSkeleton
            caption={t("admin.users.states.loadingDescription")}
            columns={8}
            rows={pageSize}
          />
        ) : pageData ? (
          <>
            {pageData.storageStatus === "persisted" ? null : (
              <StateBlock
                title={t("admin.users.states.storageUnavailableTitle")}
                description={t("admin.users.states.storageUnavailableDescription")}
              />
            )}
            <div className="aoi-user-table">
              <DataTable
                columns={userColumns}
                data={pageData.items}
                emptyLabel={t("admin.users.empty")}
              />
            </div>
          </>
        ) : (
          <StateBlock
            title={t("admin.users.states.emptyTitle")}
            description={t("admin.users.states.emptyDescription")}
          />
        )}
      </section>

      <section className="aoi-admin-panel">
        <header className="aoi-admin-panel-header-row">
          <div>
            <h2>{t("admin.users.invitation.title")}</h2>
            <p>{t("admin.users.invitation.description", { count: invitationItems.length })}</p>
          </div>
          <Badge>
            {t("admin.users.invitation.pendingCount", { count: pendingInvitationCount })}
          </Badge>
        </header>
        {invitationsQuery.isLoading ? (
          <TableSkeleton
            caption={t("admin.users.states.loadingInvitationsDescription")}
            columns={5}
            rows={5}
          />
        ) : (
          <div className="aoi-user-invitation-table">
            <DataTable
              columns={invitationColumns}
              data={invitationItems}
              emptyLabel={t("admin.users.invitation.empty")}
            />
          </div>
        )}
      </section>
    </section>
  );
}

type UserStatCardProps = {
  icon: ReactNode;
  label: string;
  value: string;
};

function UserStatCard({ icon, label, value }: UserStatCardProps) {
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

function normalizeFilters(draft: UserFilterDraft): UserFilters {
  return {
    desc: true,
    displayName: trimmedOrUndefined(draft.displayName),
    email: trimmedOrUndefined(draft.email),
    keyword: trimmedOrUndefined(draft.keyword),
    orderKey: "id",
    roleCode: trimmedOrUndefined(draft.roleCode),
    status: trimmedOrUndefined(draft.status),
    username: trimmedOrUndefined(draft.username),
  };
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

function summarizeMembers(items: IAMOrganizationUser[]) {
  return items.reduce(
    (summary, item) => {
      if (item.membershipStatus === "active") {
        summary.active += 1;
      } else {
        summary.disabled += 1;
      }
      return summary;
    },
    { active: 0, disabled: 0 },
  );
}

function userDisplayName(item: IAMOrganizationUser) {
  return item.user.displayName || item.user.username || String(item.user.id);
}

function firstRoleCode(roles: string[]) {
  return cleanRoleCode(roles[0] ?? "");
}

function cleanRoleCode(value: string) {
  return value.replace(/^role:/, "");
}

function RoleList({ roles }: { roles: string[] }) {
  const { t } = useTranslation();

  if (roles.length === 0) {
    return <span className="aoi-iam-muted">{t("common.labels.none")}</span>;
  }

  return (
    <div className="aoi-user-role-list">
      {roles.map((role) => (
        <span key={role}>{cleanRoleCode(role)}</span>
      ))}
    </div>
  );
}

function roleLabel(role: IAMRole, t: ReturnType<typeof useTranslation>["t"]) {
  return t("admin.users.roles.option", { code: role.code, name: role.name });
}

function userAccountStatus(item: IAMOrganizationUser) {
  if (item.user.lockedUntil) {
    const lockedUntil = Date.parse(item.user.lockedUntil);
    if (Number.isFinite(lockedUntil) && lockedUntil > Date.now()) {
      return "locked";
    }
  }
  return item.user.status || "active";
}

function invitationStatus(invitation: IAMInvitation) {
  if (invitation.status === "pending") {
    const expiresAt = Date.parse(invitation.expiresAt);
    if (Number.isFinite(expiresAt) && expiresAt <= Date.now()) {
      return "expired";
    }
  }
  return invitation.status || "pending";
}

function statusLabel(status: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (
    status === "active" ||
    status === "disabled" ||
    status === "expired" ||
    status === "locked" ||
    status === "pending" ||
    status === "revoked" ||
    status === "used"
  ) {
    return t(`admin.users.status.${status}`);
  }
  return status;
}

function fallbackValue(loading: boolean, t: ReturnType<typeof useTranslation>["t"]) {
  return loading ? t("loading.app") : t("common.labels.none");
}

function storageStatusLabel(status: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (status === "persisted") {
    return t("admin.users.storage.persisted");
  }
  if (status === "unavailable") {
    return t("admin.users.storage.unavailable");
  }
  return status || t("admin.users.storage.unknown");
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
    return t("admin.users.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.users.states.errorTitle");
}

function errorDescription(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.users.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

function toError(error: unknown) {
  return error instanceof Error ? error : new Error(String(error));
}

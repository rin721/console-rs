import { useQuery } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import {
  Building2,
  Clock3,
  KeyRound,
  RefreshCw,
  ScrollText,
  ShieldCheck,
  Users,
} from "lucide-react";
import { useMemo, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { DataTable } from "~/components/aoi/patterns/DataTable";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { PlatformTag } from "~/features/admin/PlatformTag";
import { ApiError } from "~/lib/api/client";
import { iamApi } from "~/lib/api/iam";
import { queryKeys } from "~/lib/api/query-keys";
import type {
  IAMAuditLog,
  IAMOrganization,
  IAMOrganizationUser,
  IAMRole,
  IAMSession,
} from "~/lib/api/types";
import { useAuthStore } from "~/stores/auth-store";

const previewPageSize = 5;
const auditLogLimit = 6;

export default function AdminIAMRoute() {
  const { i18n, t } = useTranslation();
  const currentOrgId = useAuthStore((state) => state.currentOrgId);

  const organizationsQuery = useQuery({
    queryFn: ({ signal }) =>
      iamApi.listOrganizations(
        { desc: true, orderKey: "id", page: 1, pageSize: previewPageSize },
        { signal },
      ),
    queryKey: queryKeys.iam.organizations(i18n.language, 1, previewPageSize),
  });

  const usersQuery = useQuery({
    enabled: Boolean(currentOrgId),
    queryFn: ({ signal }) =>
      iamApi.listUsers(
        currentOrgId ?? "",
        { desc: true, orderKey: "id", page: 1, pageSize: previewPageSize },
        { signal },
      ),
    queryKey: queryKeys.iam.users(i18n.language, currentOrgId ?? "", 1, previewPageSize),
  });

  const rolesQuery = useQuery({
    enabled: Boolean(currentOrgId),
    queryFn: ({ signal }) => iamApi.listRoles(currentOrgId ?? "", { signal }),
    queryKey: queryKeys.iam.roles(i18n.language, currentOrgId ?? ""),
  });

  const permissionsQuery = useQuery({
    enabled: Boolean(currentOrgId),
    queryFn: ({ signal }) => iamApi.listPermissions(currentOrgId ?? "", { signal }),
    queryKey: queryKeys.iam.permissions(i18n.language, currentOrgId ?? ""),
  });

  const sessionsQuery = useQuery({
    enabled: Boolean(currentOrgId),
    queryFn: ({ signal }) =>
      iamApi.listSessions(
        currentOrgId ?? "",
        {
          desc: true,
          orderKey: "id",
          page: 1,
          pageSize: previewPageSize,
          scope: "org",
        },
        { signal },
      ),
    queryKey: queryKeys.iam.sessions(i18n.language, currentOrgId ?? "", 1, previewPageSize, {
      scope: "org",
    }),
  });

  const auditLogsQuery = useQuery({
    enabled: Boolean(currentOrgId),
    queryFn: ({ signal }) =>
      iamApi.listAuditLogs(currentOrgId ?? "", { limit: auditLogLimit }, { signal }),
    queryKey: queryKeys.iam.auditLogs(i18n.language, currentOrgId ?? "", {
      limit: auditLogLimit,
    }),
  });

  const queries = [
    organizationsQuery,
    usersQuery,
    rolesQuery,
    permissionsQuery,
    sessionsQuery,
    auditLogsQuery,
  ];
  const fetching = queries.some((query) => query.isFetching);
  const firstError = queries.find((query) => query.error)?.error ?? null;
  const currentOrganization = useMemo(
    () => findCurrentOrganization(organizationsQuery.data?.items ?? [], currentOrgId),
    [currentOrgId, organizationsQuery.data?.items],
  );

  const memberColumns = useMemo<ColumnDef<IAMOrganizationUser>[]>(
    () => [
      {
        accessorKey: "user.displayName",
        cell: ({ row }) => (
          <PrincipalCell
            primary={row.original.user.displayName || row.original.user.username}
            secondary={row.original.user.email}
            tertiary={row.original.user.username}
          />
        ),
        header: t("admin.iam.members.columns.user"),
      },
      {
        accessorKey: "membershipStatus",
        cell: ({ row }) => (
          <StatusBadge
            label={statusLabel(row.original.membershipStatus, t)}
            status={row.original.membershipStatus}
          />
        ),
        header: t("admin.iam.members.columns.status"),
      },
      {
        accessorKey: "roles",
        cell: ({ row }) => <RoleList roles={row.original.roles} />,
        header: t("admin.iam.members.columns.roles"),
      },
      {
        accessorKey: "user.mfaEnabled",
        cell: ({ row }) =>
          row.original.user.mfaEnabled
            ? t("admin.iam.values.enabled")
            : t("admin.iam.values.disabled"),
        header: t("admin.iam.members.columns.mfa"),
      },
      {
        accessorKey: "user.lastLoginAt",
        cell: ({ row }) => formatOptionalDate(row.original.user.lastLoginAt, i18n.language, t),
        header: t("admin.iam.members.columns.lastLogin"),
      },
    ],
    [i18n.language, t],
  );

  const roleColumns = useMemo<ColumnDef<IAMRole>[]>(
    () => [
      {
        accessorKey: "name",
        cell: ({ row }) => (
          <PrincipalCell
            primary={row.original.name}
            secondary={row.original.description || t("common.labels.none")}
            tertiary={row.original.code}
          />
        ),
        header: t("admin.iam.roles.columns.role"),
      },
      {
        accessorKey: "system",
        cell: ({ row }) =>
          row.original.system ? t("admin.iam.values.systemRole") : t("admin.iam.values.customRole"),
        header: t("admin.iam.roles.columns.kind"),
      },
      {
        accessorKey: "permissions",
        cell: ({ row }) => formatNumber(row.original.permissions?.length ?? 0, i18n.language),
        header: t("admin.iam.roles.columns.permissions"),
      },
      {
        accessorKey: "updatedAt",
        cell: ({ getValue }) => formatOptionalDate(String(getValue()), i18n.language, t),
        header: t("admin.iam.roles.columns.updatedAt"),
      },
    ],
    [i18n.language, t],
  );

  const sessionColumns = useMemo<ColumnDef<IAMSession>[]>(
    () => [
      {
        accessorKey: "userId",
        cell: ({ getValue }) => String(getValue()),
        header: t("admin.iam.sessions.columns.userId"),
      },
      {
        accessorKey: "ipAddress",
        header: t("admin.iam.sessions.columns.ipAddress"),
      },
      {
        id: "platform",
        cell: ({ row }) => (
          <PlatformTag
            clientType={row.original.clientType}
            productCode={row.original.productCode}
          />
        ),
        header: t("admin.iam.sessions.columns.platform"),
      },
      {
        id: "status",
        cell: ({ row }) => {
          const status = sessionStatus(row.original);
          return <StatusBadge label={statusLabel(status, t)} status={status} />;
        },
        header: t("admin.iam.sessions.columns.status"),
      },
      {
        accessorKey: "lastUsedAt",
        cell: ({ row }) => formatOptionalDate(row.original.lastUsedAt, i18n.language, t),
        header: t("admin.iam.sessions.columns.lastUsed"),
      },
      {
        accessorKey: "expiresAt",
        cell: ({ getValue }) => formatOptionalDate(String(getValue()), i18n.language, t),
        header: t("admin.iam.sessions.columns.expiresAt"),
      },
    ],
    [i18n.language, t],
  );

  const auditLogColumns = useMemo<ColumnDef<IAMAuditLog>[]>(
    () => [
      {
        accessorKey: "action",
        cell: ({ row }) => (
          <PrincipalCell
            primary={row.original.action}
            secondary={row.original.resource}
            tertiary={row.original.resourceId || t("common.labels.none")}
          />
        ),
        header: t("admin.iam.auditLogs.columns.action"),
      },
      {
        accessorKey: "userId",
        cell: ({ row }) => String(row.original.userId ?? t("common.labels.none")),
        header: t("admin.iam.auditLogs.columns.userId"),
      },
      {
        accessorKey: "ipAddress",
        header: t("admin.iam.auditLogs.columns.ipAddress"),
      },
      {
        id: "platform",
        cell: ({ row }) => (
          <PlatformTag
            clientType={row.original.clientType}
            productCode={row.original.productCode}
          />
        ),
        header: t("admin.iam.auditLogs.columns.platform"),
      },
      {
        accessorKey: "createdAt",
        cell: ({ getValue }) => formatOptionalDate(String(getValue()), i18n.language, t),
        header: t("admin.iam.auditLogs.columns.createdAt"),
      },
    ],
    [i18n.language, t],
  );

  const refresh = () => {
    for (const query of queries) {
      if (query === organizationsQuery || currentOrgId) {
        void query.refetch();
      }
    }
  };

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-iam-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.iam.badge")}</Badge>
          <h1 id="admin-iam-title">{t("admin.iam.title")}</h1>
          <p>{t("admin.iam.description")}</p>
        </div>
        <Button
          appearance="secondary"
          icon={<RefreshCw size={17} />}
          loading={fetching}
          onClick={refresh}
        >
          {t("admin.iam.actions.refresh")}
        </Button>
      </div>

      {!currentOrgId ? (
        <StateBlock
          intent="danger"
          title={t("admin.iam.states.missingOrgTitle")}
          description={t("admin.iam.states.missingOrgDescription")}
        />
      ) : null}

      {firstError ? (
        <StateBlock
          intent="danger"
          title={errorTitle(firstError, t)}
          description={errorDescription(firstError, t)}
        />
      ) : null}

      <div className="aoi-admin-stat-grid" aria-label={t("admin.iam.summaryLabel")}>
        <IAMStatCard
          icon={<Building2 size={19} />}
          label={t("admin.iam.metrics.organizations")}
          value={
            organizationsQuery.data
              ? formatNumber(organizationsQuery.data.total, i18n.language)
              : fallbackValue(organizationsQuery.isLoading, t)
          }
        />
        <IAMStatCard
          icon={<Users size={19} />}
          label={t("admin.iam.metrics.members")}
          value={
            usersQuery.data
              ? formatNumber(usersQuery.data.total, i18n.language)
              : fallbackValue(usersQuery.isLoading, t)
          }
        />
        <IAMStatCard
          icon={<ShieldCheck size={19} />}
          label={t("admin.iam.metrics.roles")}
          value={
            rolesQuery.data
              ? formatNumber(rolesQuery.data.length, i18n.language)
              : fallbackValue(rolesQuery.isLoading, t)
          }
        />
        <IAMStatCard
          icon={<KeyRound size={19} />}
          label={t("admin.iam.metrics.permissions")}
          value={
            permissionsQuery.data
              ? formatNumber(permissionsQuery.data.length, i18n.language)
              : fallbackValue(permissionsQuery.isLoading, t)
          }
        />
        <IAMStatCard
          icon={<Clock3 size={19} />}
          label={t("admin.iam.metrics.sessions")}
          value={
            sessionsQuery.data
              ? formatNumber(sessionsQuery.data.total, i18n.language)
              : fallbackValue(sessionsQuery.isLoading, t)
          }
        />
      </div>

      <section className="aoi-admin-panel">
        <header>
          <h2>{t("admin.iam.currentOrg.title")}</h2>
          <p>{t("admin.iam.currentOrg.description")}</p>
        </header>
        {organizationsQuery.isLoading ? (
          <StateBlock
            title={t("admin.iam.states.loadingTitle")}
            description={t("admin.iam.states.loadingDescription")}
          />
        ) : currentOrganization ? (
          <dl className="aoi-admin-key-values" data-columns="3">
            <div>
              <dt>{t("admin.iam.currentOrg.rows.name")}</dt>
              <dd>{currentOrganization.name}</dd>
            </div>
            <div>
              <dt>{t("admin.iam.currentOrg.rows.code")}</dt>
              <dd>{currentOrganization.code}</dd>
            </div>
            <div>
              <dt>{t("admin.iam.currentOrg.rows.status")}</dt>
              <dd>
                <StatusBadge
                  label={statusLabel(currentOrganization.status, t)}
                  status={currentOrganization.status}
                />
              </dd>
            </div>
          </dl>
        ) : (
          <StateBlock
            title={t("admin.iam.states.emptyOrgTitle")}
            description={t("admin.iam.states.emptyOrgDescription")}
          />
        )}
      </section>

      <section className="aoi-admin-panel aoi-admin-panel--span-2">
        <header>
          <h2>{t("admin.iam.members.title")}</h2>
          <p>
            {t("admin.iam.members.description", {
              count: usersQuery.data?.items.length ?? 0,
              total: usersQuery.data?.total ?? 0,
            })}
          </p>
        </header>
        {usersQuery.isLoading ? (
          <StateBlock
            title={t("admin.iam.states.loadingTitle")}
            description={t("admin.iam.states.loadingDescription")}
          />
        ) : usersQuery.data ? (
          <div className="aoi-iam-table aoi-iam-table--members">
            <DataTable
              columns={memberColumns}
              data={usersQuery.data.items}
              emptyLabel={t("admin.iam.members.empty")}
            />
          </div>
        ) : null}
      </section>

      <div className="aoi-admin-server-grid">
        <section className="aoi-admin-panel">
          <header>
            <h2>{t("admin.iam.roles.title")}</h2>
            <p>{t("admin.iam.roles.description")}</p>
          </header>
          {rolesQuery.isLoading ? (
            <StateBlock
              title={t("admin.iam.states.loadingTitle")}
              description={t("admin.iam.states.loadingDescription")}
            />
          ) : rolesQuery.data ? (
            <div className="aoi-iam-table aoi-iam-table--roles">
              <DataTable
                columns={roleColumns}
                data={rolesQuery.data}
                emptyLabel={t("admin.iam.roles.empty")}
              />
            </div>
          ) : null}
        </section>

        <section className="aoi-admin-panel">
          <header>
            <h2>{t("admin.iam.sessions.title")}</h2>
            <p>
              {t("admin.iam.sessions.description", {
                count: sessionsQuery.data?.items.length ?? 0,
                total: sessionsQuery.data?.total ?? 0,
              })}
            </p>
          </header>
          {sessionsQuery.isLoading ? (
            <StateBlock
              title={t("admin.iam.states.loadingTitle")}
              description={t("admin.iam.states.loadingDescription")}
            />
          ) : sessionsQuery.data ? (
            <div className="aoi-iam-table aoi-iam-table--sessions">
              <DataTable
                columns={sessionColumns}
                data={sessionsQuery.data.items}
                emptyLabel={t("admin.iam.sessions.empty")}
              />
            </div>
          ) : null}
        </section>
      </div>

      <section className="aoi-admin-panel">
        <header className="aoi-admin-panel-header-row">
          <div>
            <h2>{t("admin.iam.auditLogs.title")}</h2>
            <p>{t("admin.iam.auditLogs.description", { count: auditLogLimit })}</p>
          </div>
          <span className="aoi-iam-audit-count">
            <ScrollText aria-hidden="true" size={16} />
            {formatNumber(auditLogsQuery.data?.length ?? 0, i18n.language)}
          </span>
        </header>
        {auditLogsQuery.isLoading ? (
          <StateBlock
            title={t("admin.iam.states.loadingTitle")}
            description={t("admin.iam.states.loadingDescription")}
          />
        ) : auditLogsQuery.data ? (
          <div className="aoi-iam-table aoi-iam-table--audit">
            <DataTable
              columns={auditLogColumns}
              data={auditLogsQuery.data}
              emptyLabel={t("admin.iam.auditLogs.empty")}
            />
          </div>
        ) : null}
      </section>
    </section>
  );
}

type IAMStatCardProps = {
  icon: ReactNode;
  label: string;
  value: string;
};

function IAMStatCard({ icon, label, value }: IAMStatCardProps) {
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

function PrincipalCell({
  primary,
  secondary,
  tertiary,
}: {
  primary: string;
  secondary: string;
  tertiary?: string;
}) {
  return (
    <div className="aoi-iam-principal">
      <strong>{primary}</strong>
      <span>{secondary}</span>
      {tertiary ? <code>{tertiary}</code> : null}
    </div>
  );
}

function RoleList({ roles }: { roles: string[] }) {
  const { t } = useTranslation();

  if (roles.length === 0) {
    return <span className="aoi-iam-muted">{t("common.labels.none")}</span>;
  }

  return (
    <div className="aoi-iam-role-list">
      {roles.map((role) => (
        <span key={role}>{role}</span>
      ))}
    </div>
  );
}

function StatusBadge({ label, status }: { label: string; status: string }) {
  return (
    <span className="aoi-iam-status" data-status={status}>
      {label}
    </span>
  );
}

function findCurrentOrganization(items: IAMOrganization[], currentOrgId: string | null) {
  if (!currentOrgId) {
    return null;
  }
  return items.find((item) => String(item.id) === currentOrgId) ?? items[0] ?? null;
}

function sessionStatus(session: IAMSession) {
  if (session.revokedAt) {
    return "revoked";
  }
  const expiresAt = Date.parse(session.expiresAt);
  if (Number.isFinite(expiresAt) && expiresAt < Date.now()) {
    return "expired";
  }
  return "active";
}

function statusLabel(status: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (
    status === "active" ||
    status === "disabled" ||
    status === "expired" ||
    status === "pending" ||
    status === "revoked" ||
    status === "used"
  ) {
    return t(`admin.iam.status.${status}`);
  }
  return status;
}

function fallbackValue(loading: boolean, t: ReturnType<typeof useTranslation>["t"]) {
  return loading ? t("loading.app") : t("common.labels.none");
}

function formatNumber(value: number, locale: string) {
  return new Intl.NumberFormat(locale).format(value);
}

function formatOptionalDate(
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
    return t("admin.iam.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.iam.states.errorTitle");
}

function errorDescription(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.iam.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

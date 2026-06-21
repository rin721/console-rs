import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import type { TFunction } from "i18next";
import {
  KeyRound,
  Plus,
  RefreshCw,
  Save,
  Shield,
  ShieldCheck,
  SlidersHorizontal,
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
import { ApiError } from "~/lib/api/client";
import { iamApi } from "~/lib/api/iam";
import { queryKeys } from "~/lib/api/query-keys";
import type {
  IAMCreateRoleInput,
  IAMPermission,
  IAMRole,
  IAMUpdateRoleInput,
} from "~/lib/api/types";
import { useAuthStore } from "~/stores/auth-store";

const roleCodePattern = /^[a-z][a-z0-9_-]{0,63}$/;

const createRoleSchema = z.object({
  code: z.string().trim().regex(roleCodePattern),
  description: z.string().trim().max(500),
  name: z.string().trim().min(1),
  permissions: z.array(z.string().trim().min(1)),
});

const updateRoleSchema = z.object({
  description: z.string().trim().max(500),
  name: z.string().trim().min(1),
  permissions: z.array(z.string().trim().min(1)),
});

type RoleDraft = {
  code: string;
  description: string;
  name: string;
  permissions: string[];
};

type RoleEditDraft = Omit<RoleDraft, "code">;

type RoleNotice = {
  description: string;
  intent?: "danger" | "info";
  title: string;
};

type PermissionGroup = {
  label: string;
  object: string;
  permissions: IAMPermission[];
};

const initialCreateDraft: RoleDraft = {
  code: "",
  description: "",
  name: "",
  permissions: [],
};

const initialEditDraft: RoleEditDraft = {
  description: "",
  name: "",
  permissions: [],
};

const emptyRoles: IAMRole[] = [];
const emptyPermissions: IAMPermission[] = [];

export default function AdminRolesRoute() {
  const { i18n, t } = useTranslation();
  const queryClient = useQueryClient();
  const currentOrgId = useAuthStore((state) => state.currentOrgId);
  const [createOpen, setCreateOpen] = useState(false);
  const [createDraft, setCreateDraft] = useState<RoleDraft>(initialCreateDraft);
  const [editRoleId, setEditRoleId] = useState("");
  const [editDraft, setEditDraft] = useState<RoleEditDraft | null>(null);
  const [notice, setNotice] = useState<RoleNotice | null>(null);

  const rolesQueryKey = queryKeys.iam.roles(i18n.language, currentOrgId ?? "");
  const permissionsQueryKey = queryKeys.iam.permissions(i18n.language, currentOrgId ?? "");

  const rolesQuery = useQuery({
    enabled: Boolean(currentOrgId),
    queryFn: ({ signal }) => iamApi.listRoles(currentOrgId ?? "", { signal }),
    queryKey: rolesQueryKey,
  });

  const permissionsQuery = useQuery({
    enabled: Boolean(currentOrgId),
    queryFn: ({ signal }) => iamApi.listPermissions(currentOrgId ?? "", { signal }),
    queryKey: permissionsQueryKey,
  });

  const roles = rolesQuery.data ?? emptyRoles;
  const permissions = permissionsQuery.data ?? emptyPermissions;

  const roleSummary = useMemo(() => summarizeRoles(roles, permissions), [permissions, roles]);
  const editableRoles = useMemo(() => roles.filter((role) => !role.system), [roles]);
  const activeEditRoleId = editRoleId || (editableRoles[0] ? String(editableRoles[0].id) : "");
  const selectedEditRole = useMemo(
    () => editableRoles.find((role) => String(role.id) === activeEditRoleId) ?? null,
    [activeEditRoleId, editableRoles],
  );
  const activeEditDraft =
    editDraft ?? (selectedEditRole ? draftFromRole(selectedEditRole) : initialEditDraft);
  const roleOptions = useMemo(
    () =>
      editableRoles.map((role) => ({
        label: t("admin.roles.edit.roleOption", { code: role.code, name: role.name }),
        value: String(role.id),
      })),
    [editableRoles, t],
  );
  const permissionMap = useMemo(
    () => new Map(permissions.map((permission) => [permission.code, permission])),
    [permissions],
  );

  const createRoleMutation = useMutation({
    mutationFn: (input: IAMCreateRoleInput) => iamApi.createRole(currentOrgId ?? "", input),
    onError: (error) => {
      setNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.roles.create.errorTitle"),
      });
    },
    onSuccess: (role) => {
      setCreateDraft(initialCreateDraft);
      setCreateOpen(false);
      setEditRoleId(String(role.id));
      setEditDraft(draftFromRole(role));
      setNotice({
        description: t("admin.roles.create.successDescription", { name: role.name }),
        title: t("admin.roles.create.successTitle"),
      });
      void queryClient.invalidateQueries({ queryKey: rolesQueryKey });
    },
  });

  const updateRoleMutation = useMutation({
    mutationFn: (input: { role: IAMRole; values: IAMUpdateRoleInput }) =>
      iamApi.updateRole(currentOrgId ?? "", input.role.id, input.values),
    onError: (error) => {
      setNotice({
        description: errorDescription(toError(error), t),
        intent: "danger",
        title: t("admin.roles.edit.errorTitle"),
      });
    },
    onSuccess: (role) => {
      setEditDraft(draftFromRole(role));
      setNotice({
        description: t("admin.roles.edit.successDescription", { name: role.name }),
        title: t("admin.roles.edit.successTitle"),
      });
      void queryClient.invalidateQueries({ queryKey: rolesQueryKey });
    },
  });

  const roleColumns = useMemo<ColumnDef<IAMRole>[]>(
    () => [
      {
        accessorKey: "code",
        cell: ({ row }) => (
          <div className="aoi-role-principal">
            <strong>{row.original.name}</strong>
            <code>{row.original.code}</code>
            <span>{row.original.description || t("common.labels.none")}</span>
          </div>
        ),
        header: t("admin.roles.columns.role"),
      },
      {
        cell: ({ row }) => (
          <span className="aoi-iam-status" data-status={row.original.system ? "used" : "active"}>
            {row.original.system ? t("admin.roles.kind.system") : t("admin.roles.kind.custom")}
          </span>
        ),
        header: t("admin.roles.columns.kind"),
      },
      {
        cell: ({ row }) => (
          <RolePermissionList
            codes={row.original.permissions ?? []}
            permissionMap={permissionMap}
          />
        ),
        header: t("admin.roles.columns.permissions"),
      },
      {
        cell: ({ row }) => formatDate(row.original.updatedAt, i18n.language, t),
        header: t("admin.roles.columns.updatedAt"),
      },
    ],
    [i18n.language, permissionMap, t],
  );

  function handleRefresh() {
    void rolesQuery.refetch();
    void permissionsQuery.refetch();
  }

  function handleCreateSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const parsed = createRoleSchema.safeParse({
      ...createDraft,
      permissions: uniqueCodes(createDraft.permissions),
    });
    if (!parsed.success) {
      setNotice({
        description: validationMessage(parsed.error, t),
        intent: "danger",
        title: t("admin.roles.validation.title"),
      });
      return;
    }
    createRoleMutation.mutate(parsed.data);
  }

  function handleEditSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedEditRole) {
      setNotice({
        description: t("admin.roles.edit.noSelectionDescription"),
        intent: "danger",
        title: t("admin.roles.edit.noSelectionTitle"),
      });
      return;
    }
    const parsed = updateRoleSchema.safeParse({
      ...activeEditDraft,
      permissions: uniqueCodes(activeEditDraft.permissions),
    });
    if (!parsed.success) {
      setNotice({
        description: validationMessage(parsed.error, t),
        intent: "danger",
        title: t("admin.roles.validation.title"),
      });
      return;
    }
    updateRoleMutation.mutate({ role: selectedEditRole, values: parsed.data });
  }

  function handleEditRoleChange(roleId: string) {
    const role = editableRoles.find((item) => String(item.id) === roleId);
    setEditRoleId(roleId);
    setEditDraft(role ? draftFromRole(role) : null);
  }

  function updateActiveEditDraft(update: Partial<RoleEditDraft>) {
    if (!editRoleId && selectedEditRole) {
      setEditRoleId(String(selectedEditRole.id));
    }
    setEditDraft({ ...activeEditDraft, ...update });
  }

  if (!currentOrgId) {
    return (
      <section className="aoi-admin-dashboard">
        <StateBlock
          title={t("admin.roles.states.missingOrgTitle")}
          description={t("admin.roles.states.missingOrgDescription")}
        />
      </section>
    );
  }

  const queryError = toMaybeError(rolesQuery.error) ?? toMaybeError(permissionsQuery.error);
  const isInitialLoading = rolesQuery.isLoading || permissionsQuery.isLoading;
  const isMutating = createRoleMutation.isPending || updateRoleMutation.isPending;

  return (
    <section className="aoi-admin-dashboard">
      <header className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.roles.badge")}</Badge>
          <h1>{t("admin.roles.title")}</h1>
          <p>{t("admin.roles.description")}</p>
        </div>
        <div className="aoi-admin-action-row aoi-role-page-actions">
          <Button
            appearance="secondary"
            disabled={rolesQuery.isFetching || permissionsQuery.isFetching}
            icon={<RefreshCw size={17} />}
            onClick={handleRefresh}
          >
            {t("admin.roles.actions.refresh")}
          </Button>
          <Button
            icon={createOpen ? <X size={17} /> : <Plus size={17} />}
            onClick={() => setCreateOpen((current) => !current)}
          >
            {createOpen ? t("admin.roles.actions.cancelCreate") : t("admin.roles.actions.create")}
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
              {t("admin.roles.actions.dismiss")}
            </Button>
          }
        />
      ) : null}

      <div className="aoi-admin-stat-grid" aria-label={t("admin.roles.summaryLabel")}>
        <RoleStatCard
          icon={<ShieldCheck size={18} />}
          label={t("admin.roles.metrics.total")}
          value={formatNumber(roleSummary.total, i18n.language)}
        />
        <RoleStatCard
          icon={<Shield size={18} />}
          label={t("admin.roles.metrics.custom")}
          value={formatNumber(roleSummary.custom, i18n.language)}
        />
        <RoleStatCard
          icon={<KeyRound size={18} />}
          label={t("admin.roles.metrics.system")}
          value={formatNumber(roleSummary.system, i18n.language)}
        />
        <RoleStatCard
          icon={<SlidersHorizontal size={18} />}
          label={t("admin.roles.metrics.permissions")}
          value={formatNumber(roleSummary.permissions, i18n.language)}
        />
        <RoleStatCard
          icon={<ShieldCheck size={18} />}
          label={t("admin.roles.metrics.assignedPermissions")}
          value={formatNumber(roleSummary.assignedPermissions, i18n.language)}
        />
      </div>

      {queryError ? (
        <StateBlock
          title={errorTitle(queryError, t)}
          description={errorDescription(queryError, t)}
          intent="danger"
        />
      ) : isInitialLoading ? (
        <StateBlock
          title={t("admin.roles.states.loadingTitle")}
          description={t("admin.roles.states.loadingDescription")}
        />
      ) : (
        <div
          className={
            createOpen ? "aoi-role-workbench aoi-role-workbench--with-create" : "aoi-role-workbench"
          }
        >
          <section className="aoi-admin-panel aoi-admin-panel--span-2">
            <header>
              <h2>{t("admin.roles.list.title")}</h2>
              <p>{t("admin.roles.list.description", { count: roles.length })}</p>
            </header>
            <div className="aoi-role-table">
              <DataTable columns={roleColumns} data={roles} emptyLabel={t("admin.roles.empty")} />
            </div>
          </section>

          {createOpen ? (
            <section className="aoi-admin-panel">
              <header>
                <h2>{t("admin.roles.create.title")}</h2>
                <p>{t("admin.roles.create.description")}</p>
              </header>
              <form className="aoi-role-form" onSubmit={handleCreateSubmit}>
                <FormField
                  label={t("admin.roles.fields.code")}
                  value={createDraft.code}
                  onChange={(event) =>
                    setCreateDraft((current) => ({ ...current, code: event.target.value }))
                  }
                  placeholder={t("admin.roles.fields.codePlaceholder")}
                  autoComplete="off"
                />
                <FormField
                  label={t("admin.roles.fields.name")}
                  value={createDraft.name}
                  onChange={(event) =>
                    setCreateDraft((current) => ({ ...current, name: event.target.value }))
                  }
                  autoComplete="off"
                />
                <TextAreaField
                  id="create-role-description"
                  label={t("admin.roles.fields.description")}
                  value={createDraft.description}
                  onChange={(value) =>
                    setCreateDraft((current) => ({ ...current, description: value }))
                  }
                  help={t("admin.roles.fields.descriptionHelp")}
                />
                <PermissionSelector
                  disabled={isMutating}
                  permissions={permissions}
                  selected={createDraft.permissions}
                  onChange={(nextPermissions) =>
                    setCreateDraft((current) => ({ ...current, permissions: nextPermissions }))
                  }
                  idPrefix="create-role-permission"
                  title={t("admin.roles.permissions.createTitle")}
                  description={t("admin.roles.permissions.createDescription")}
                />
                <div className="aoi-role-form-actions">
                  <Button
                    appearance="secondary"
                    disabled={isMutating}
                    icon={<X size={17} />}
                    onClick={() => {
                      setCreateDraft(initialCreateDraft);
                      setCreateOpen(false);
                    }}
                  >
                    {t("admin.roles.actions.cancelCreate")}
                  </Button>
                  <Button
                    disabled={isMutating}
                    icon={<Plus size={17} />}
                    loading={createRoleMutation.isPending}
                    type="submit"
                  >
                    {t("admin.roles.actions.submitCreate")}
                  </Button>
                </div>
              </form>
            </section>
          ) : null}

          <section className="aoi-admin-panel">
            <header>
              <h2>{t("admin.roles.edit.title")}</h2>
              <p>{t("admin.roles.edit.description")}</p>
            </header>
            {editableRoles.length === 0 ? (
              <StateBlock
                title={t("admin.roles.edit.noneTitle")}
                description={t("admin.roles.edit.noneDescription")}
              />
            ) : (
              <form className="aoi-role-form" onSubmit={handleEditSubmit}>
                <SelectField
                  label={t("admin.roles.edit.role")}
                  options={roleOptions}
                  value={activeEditRoleId}
                  onChange={(event) => handleEditRoleChange(event.target.value)}
                />
                <FormField
                  label={t("admin.roles.fields.name")}
                  value={activeEditDraft.name}
                  onChange={(event) => updateActiveEditDraft({ name: event.target.value })}
                  autoComplete="off"
                />
                <TextAreaField
                  id="edit-role-description"
                  label={t("admin.roles.fields.description")}
                  value={activeEditDraft.description}
                  onChange={(value) => updateActiveEditDraft({ description: value })}
                  help={t("admin.roles.edit.descriptionHelp")}
                />
                <PermissionSelector
                  disabled={isMutating}
                  permissions={permissions}
                  selected={activeEditDraft.permissions}
                  onChange={(nextPermissions) =>
                    updateActiveEditDraft({ permissions: nextPermissions })
                  }
                  idPrefix="edit-role-permission"
                  title={t("admin.roles.permissions.editTitle")}
                  description={t("admin.roles.permissions.editDescription")}
                />
                <div className="aoi-role-form-actions">
                  <Button
                    appearance="secondary"
                    disabled={isMutating || !selectedEditRole}
                    icon={<RefreshCw size={17} />}
                    onClick={() => {
                      if (selectedEditRole) {
                        setEditRoleId(String(selectedEditRole.id));
                        setEditDraft(draftFromRole(selectedEditRole));
                      }
                    }}
                  >
                    {t("admin.roles.actions.reset")}
                  </Button>
                  <Button
                    disabled={isMutating || !selectedEditRole}
                    icon={<Save size={17} />}
                    loading={updateRoleMutation.isPending}
                    type="submit"
                  >
                    {t("admin.roles.actions.save")}
                  </Button>
                </div>
              </form>
            )}
          </section>
        </div>
      )}
    </section>
  );
}

type RoleStatCardProps = {
  icon: ReactNode;
  label: string;
  value: string;
};

function RoleStatCard({ icon, label, value }: RoleStatCardProps) {
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

type TextAreaFieldProps = {
  help?: string;
  id: string;
  label: string;
  onChange: (value: string) => void;
  value: string;
};

function TextAreaField({ help, id, label, onChange, value }: TextAreaFieldProps) {
  return (
    <div className="aoi-form-field">
      <label htmlFor={id}>{label}</label>
      <textarea id={id} value={value} onChange={(event) => onChange(event.target.value)} />
      {help ? <span className="aoi-form-field__help">{help}</span> : null}
    </div>
  );
}

type PermissionSelectorProps = {
  description: string;
  disabled?: boolean;
  idPrefix: string;
  onChange: (permissions: string[]) => void;
  permissions: IAMPermission[];
  selected: string[];
  title: string;
};

function PermissionSelector({
  description,
  disabled,
  idPrefix,
  onChange,
  permissions,
  selected,
  title,
}: PermissionSelectorProps) {
  const { t } = useTranslation();
  const [search, setSearch] = useState("");
  const [objectFilter, setObjectFilter] = useState("");
  const selectedSet = useMemo(() => new Set(selected), [selected]);
  const objectOptions = useMemo(() => {
    const objects = uniqueObjects(permissions);
    return [
      { label: t("admin.roles.permissions.allObjects"), value: "" },
      ...objects.map((object) => ({
        label: permissionObjectLabel(object, t),
        value: object,
      })),
    ];
  }, [permissions, t]);
  const groups = useMemo(
    () => groupPermissions(permissions, search, objectFilter, t),
    [objectFilter, permissions, search, t],
  );

  function replaceGroup(groupPermissions: IAMPermission[], include: boolean) {
    const groupCodes = groupPermissions.map((permission) => permission.code);
    const nextSet = new Set(selected);
    for (const code of groupCodes) {
      if (include) {
        nextSet.add(code);
      } else {
        nextSet.delete(code);
      }
    }
    onChange([...nextSet].sort());
  }

  return (
    <section className="aoi-role-permission-panel" aria-label={title}>
      <header>
        <div>
          <h3>{title}</h3>
          <p>{description}</p>
        </div>
        <span>{t("admin.roles.permissions.selected", { count: selected.length })}</span>
      </header>
      <div className="aoi-role-permission-toolbar">
        <FormField
          label={t("admin.roles.permissions.search")}
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder={t("admin.roles.permissions.searchPlaceholder")}
        />
        <SelectField
          label={t("admin.roles.permissions.object")}
          options={objectOptions}
          value={objectFilter}
          onChange={(event) => setObjectFilter(event.target.value)}
        />
      </div>
      <div className="aoi-role-permission-groups">
        {groups.length > 0 ? (
          groups.map((group) => (
            <section className="aoi-role-permission-group" key={group.object}>
              <header>
                <div>
                  <h4>{group.label}</h4>
                  <p>
                    {t("admin.roles.permissions.groupCount", {
                      count: group.permissions.length,
                      selected: selectedCount(group.permissions, selectedSet),
                    })}
                  </p>
                </div>
                <div className="aoi-role-permission-group-actions">
                  <Button
                    appearance="ghost"
                    disabled={disabled}
                    onClick={() => replaceGroup(group.permissions, true)}
                  >
                    {t("admin.roles.permissions.selectGroup", { group: group.label })}
                  </Button>
                  <Button
                    appearance="ghost"
                    disabled={disabled}
                    onClick={() => replaceGroup(group.permissions, false)}
                  >
                    {t("admin.roles.permissions.clearGroup", { group: group.label })}
                  </Button>
                </div>
              </header>
              <div className="aoi-role-permission-list">
                {group.permissions.map((permission) => {
                  const optionId = `${idPrefix}-${permission.code.replace(/[^a-z0-9_-]+/gi, "-")}`;
                  return (
                    <label
                      className="aoi-role-permission-option"
                      htmlFor={optionId}
                      key={permission.code}
                    >
                      <input
                        id={optionId}
                        type="checkbox"
                        checked={selectedSet.has(permission.code)}
                        disabled={disabled}
                        onChange={(event) =>
                          onChange(
                            togglePermission(selected, permission.code, event.target.checked),
                          )
                        }
                        aria-label={t("admin.roles.permissions.toggle", { code: permission.code })}
                      />
                      <span>
                        <strong>{permission.name || permission.code}</strong>
                        <code>{permission.code}</code>
                        <em>{permission.description || t("common.labels.none")}</em>
                      </span>
                    </label>
                  );
                })}
              </div>
            </section>
          ))
        ) : (
          <StateBlock
            title={t("admin.roles.permissions.emptyTitle")}
            description={t("admin.roles.permissions.emptyDescription")}
          />
        )}
      </div>
    </section>
  );
}

function RolePermissionList({
  codes,
  permissionMap,
}: {
  codes: string[];
  permissionMap: Map<string, IAMPermission>;
}) {
  const { t } = useTranslation();
  const normalizedCodes = uniqueCodes(codes);
  const visibleCodes = normalizedCodes.slice(0, 6);
  const hiddenCount = Math.max(0, normalizedCodes.length - visibleCodes.length);

  if (normalizedCodes.length === 0) {
    return <span className="aoi-iam-muted">{t("common.labels.none")}</span>;
  }

  return (
    <div className="aoi-role-permission-tags">
      {visibleCodes.map((code) => {
        const permission = permissionMap.get(code);
        return (
          <span key={code} title={permission?.name || code}>
            {code}
          </span>
        );
      })}
      {hiddenCount > 0 ? (
        <span>{t("admin.roles.permissions.more", { count: hiddenCount })}</span>
      ) : null}
    </div>
  );
}

function draftFromRole(role: IAMRole): RoleEditDraft {
  return {
    description: role.description ?? "",
    name: role.name ?? "",
    permissions: uniqueCodes(role.permissions ?? []),
  };
}

function summarizeRoles(roles: IAMRole[], permissions: IAMPermission[]) {
  const assignedPermissions = new Set<string>();
  let custom = 0;
  let system = 0;

  for (const role of roles) {
    if (role.system) {
      system += 1;
    } else {
      custom += 1;
    }
    for (const permission of role.permissions ?? []) {
      assignedPermissions.add(permission);
    }
  }

  return {
    assignedPermissions: assignedPermissions.size,
    custom,
    permissions: permissions.length,
    system,
    total: roles.length,
  };
}

function uniqueCodes(codes: string[]) {
  return [...new Set(codes.map((code) => code.trim()).filter(Boolean))].sort();
}

function togglePermission(selected: string[], code: string, checked: boolean) {
  const nextSet = new Set(selected);
  if (checked) {
    nextSet.add(code);
  } else {
    nextSet.delete(code);
  }
  return [...nextSet].sort();
}

function uniqueObjects(permissions: IAMPermission[]) {
  return [...new Set(permissions.map((permission) => permissionObject(permission.code)))].sort();
}

function groupPermissions(
  permissions: IAMPermission[],
  search: string,
  objectFilter: string,
  t: TFunction,
): PermissionGroup[] {
  const normalizedSearch = search.trim().toLowerCase();
  const grouped = new Map<string, IAMPermission[]>();

  for (const permission of permissions) {
    const object = permissionObject(permission.code);
    if (objectFilter && object !== objectFilter) {
      continue;
    }
    if (normalizedSearch && !permissionMatches(permission, normalizedSearch, object, t)) {
      continue;
    }
    const group = grouped.get(object) ?? [];
    group.push(permission);
    grouped.set(object, group);
  }

  return [...grouped.entries()]
    .map(([object, groupPermissions]) => ({
      label: permissionObjectLabel(object, t),
      object,
      permissions: groupPermissions.sort((a, b) => a.code.localeCompare(b.code)),
    }))
    .sort((a, b) => a.label.localeCompare(b.label));
}

function permissionMatches(
  permission: IAMPermission,
  search: string,
  object: string,
  t: TFunction,
) {
  const label = permissionObjectLabel(object, t);
  return [permission.code, permission.name, permission.description, label]
    .filter(Boolean)
    .some((value) => value.toLowerCase().includes(search));
}

function permissionObject(code: string) {
  const [object] = code.split(":");
  return object || "other";
}

function permissionObjectLabel(object: string, t: TFunction) {
  return t(`admin.roles.permissionObjects.${object}`, { defaultValue: object });
}

function selectedCount(permissions: IAMPermission[], selectedSet: Set<string>) {
  return permissions.filter((permission) => selectedSet.has(permission.code)).length;
}

function validationMessage(error: z.ZodError, t: TFunction) {
  const firstPath = error.issues[0]?.path[0];
  if (firstPath === "code") {
    return t("admin.roles.validation.code");
  }
  if (firstPath === "name") {
    return t("admin.roles.validation.name");
  }
  if (firstPath === "description") {
    return t("admin.roles.validation.description");
  }
  if (firstPath === "permissions") {
    return t("admin.roles.validation.permissions");
  }
  return t("admin.roles.validation.description");
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
    return t("admin.roles.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.roles.states.errorTitle");
}

function errorDescription(error: Error, t: TFunction) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.roles.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

function toMaybeError(error: unknown) {
  return error instanceof Error ? error : null;
}

function toError(error: unknown) {
  return error instanceof Error ? error : new Error(String(error));
}

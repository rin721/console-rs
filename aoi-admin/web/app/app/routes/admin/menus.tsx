import { useQuery } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { Monitor, PanelLeft, RefreshCw, RotateCcw, Search, ShieldCheck, Smartphone } from "lucide-react";
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
import type { SystemMenuGroup, SystemMenuItem } from "~/lib/api/types";

type MenuFilters = {
  groupCode: string;
  keyword: string;
};

const emptyFilters: MenuFilters = {
  groupCode: "",
  keyword: "",
};

export default function AdminMenusRoute() {
  const { i18n, t } = useTranslation();
  const [filters, setFilters] = useState<MenuFilters>(emptyFilters);

  const menusQuery = useQuery({
    queryFn: ({ signal }) => systemApi.listMenus({ signal }),
    queryKey: queryKeys.system.menus(i18n.language),
  });

  const groups = useMemo(() => menusQuery.data ?? [], [menusQuery.data]);
  const summary = useMemo(() => summarizeMenus(groups), [groups]);
  const groupOptions = useMemo(
    () => toGroupOptions(groups, t("admin.menus.filters.allGroups")),
    [groups, t],
  );
  const filteredGroups = useMemo(
    () => filterMenuGroups(groups, filters, t),
    [filters, groups, t],
  );
  const filteredTotal = useMemo(
    () => filteredGroups.reduce((total, group) => total + group.items.length, 0),
    [filteredGroups],
  );

  const columns = useMemo<ColumnDef<SystemMenuItem>[]>(
    () => [
      {
        accessorKey: "label",
        cell: ({ row }) => (
          <div className="aoi-menu-name">
            <PanelLeft aria-hidden="true" size={17} />
            <div>
              <strong>{row.original.label}</strong>
              <span>{row.original.code}</span>
            </div>
          </div>
        ),
        header: t("admin.menus.columns.menu"),
      },
      {
        accessorKey: "path",
        cell: ({ getValue }) => <span className="aoi-menu-path">{String(getValue())}</span>,
        header: t("admin.menus.columns.path"),
      },
      {
        accessorKey: "permission",
        cell: ({ row }) => (
          <span className="aoi-menu-permission">
            {row.original.permission || t("admin.menus.permission.loginVisible")}
          </span>
        ),
        header: t("admin.menus.columns.permission"),
      },
      {
        accessorKey: "mobile",
        cell: ({ row }) => (
          <span className="aoi-menu-entry" data-mobile={String(row.original.mobile)}>
            {row.original.mobile
              ? t("admin.menus.entry.mobileAndDesktop")
              : t("admin.menus.entry.desktop")}
          </span>
        ),
        header: t("admin.menus.columns.entry"),
      },
      {
        accessorKey: "icon",
        cell: ({ getValue }) => <span className="aoi-menu-code">{String(getValue())}</span>,
        header: t("admin.menus.columns.icon"),
      },
      {
        accessorKey: "order",
        cell: ({ getValue }) => formatNumber(Number(getValue()), i18n.language),
        header: t("admin.menus.columns.order"),
      },
    ],
    [i18n.language, t],
  );

  const updateFilter = (key: keyof MenuFilters, value: string) => {
    setFilters((current) => ({ ...current, [key]: value }));
  };

  const resetFilters = () => {
    setFilters(emptyFilters);
  };

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-menus-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.menus.badge")}</Badge>
          <h1 id="admin-menus-title">{t("admin.menus.title")}</h1>
          <p>{t("admin.menus.description")}</p>
        </div>
        <Button
          appearance="secondary"
          icon={<RefreshCw size={17} />}
          loading={menusQuery.isFetching}
          onClick={() => void menusQuery.refetch()}
        >
          {t("admin.menus.actions.refresh")}
        </Button>
      </div>

      {menusQuery.error ? (
        <StateBlock
          intent="danger"
          title={errorTitle(menusQuery.error, t)}
          description={errorDescription(menusQuery.error, t)}
        />
      ) : null}

      <div className="aoi-admin-stat-grid" aria-label={t("admin.menus.summaryLabel")}>
        <MenuStatCard
          icon={<PanelLeft size={19} />}
          label={t("admin.menus.metrics.groups")}
          value={formatNumber(summary.groups, i18n.language)}
        />
        <MenuStatCard
          icon={<PanelLeft size={19} />}
          label={t("admin.menus.metrics.items")}
          value={formatNumber(summary.items, i18n.language)}
        />
        <MenuStatCard
          icon={<ShieldCheck size={19} />}
          label={t("admin.menus.metrics.protected")}
          value={formatNumber(summary.protectedItems, i18n.language)}
        />
        <MenuStatCard
          icon={<Smartphone size={19} />}
          label={t("admin.menus.metrics.mobile")}
          value={formatNumber(summary.mobileItems, i18n.language)}
        />
        <MenuStatCard
          icon={<Monitor size={19} />}
          label={t("admin.menus.metrics.desktopOnly")}
          value={formatNumber(summary.desktopOnlyItems, i18n.language)}
        />
      </div>

      <section className="aoi-admin-panel">
        <header>
          <h2>{t("admin.menus.filters.title")}</h2>
          <p>{t("admin.menus.filters.description")}</p>
        </header>
        <form className="aoi-admin-filter-form aoi-admin-filter-form--compact" onSubmit={(event) => event.preventDefault()}>
          <FormField
            label={t("admin.menus.filters.keyword")}
            value={filters.keyword}
            onChange={(event) => updateFilter("keyword", event.currentTarget.value)}
          />
          <SelectField
            label={t("admin.menus.filters.group")}
            options={groupOptions}
            value={filters.groupCode}
            onChange={(event) => updateFilter("groupCode", event.currentTarget.value)}
          />
          <div className="aoi-admin-filter-actions">
            <Button
              appearance="secondary"
              icon={<RotateCcw size={17} />}
              onClick={resetFilters}
            >
              {t("admin.menus.actions.reset")}
            </Button>
          </div>
        </form>
      </section>

      <section className="aoi-admin-panel">
        <header className="aoi-admin-panel-header-row">
          <div>
            <h2>{t("admin.menus.list.title")}</h2>
            <p>
              {t("admin.menus.list.description", {
                count: filteredTotal,
                total: summary.items,
              })}
            </p>
          </div>
          <span className="aoi-api-count">
            <Search aria-hidden="true" size={16} />
            {formatNumber(filteredTotal, i18n.language)}
          </span>
        </header>

        {menusQuery.isLoading ? (
          <StateBlock
            title={t("admin.menus.states.loadingTitle")}
            description={t("admin.menus.states.loadingDescription")}
          />
        ) : menusQuery.data ? (
          filteredGroups.length > 0 ? (
            <div className="aoi-menu-groups">
              {filteredGroups.map((group) => (
                <section className="aoi-menu-group" key={group.code}>
                  <header>
                    <div>
                      <h3>{group.label}</h3>
                      <p>
                        {t("admin.menus.groupMeta", {
                          code: group.code,
                          order: group.order,
                        })}
                      </p>
                    </div>
                    <span>{t("admin.menus.summary.items", { count: group.items.length })}</span>
                  </header>
                  <div className="aoi-menu-table">
                    <DataTable
                      columns={columns}
                      data={group.items}
                      emptyLabel={t("admin.menus.empty")}
                    />
                  </div>
                </section>
              ))}
            </div>
          ) : (
            <StateBlock
              title={t("admin.menus.states.noMatchesTitle")}
              description={t("admin.menus.empty")}
            />
          )
        ) : (
          <StateBlock
            title={t("admin.menus.states.emptyTitle")}
            description={t("admin.menus.states.emptyDescription")}
          />
        )}
      </section>
    </section>
  );
}

type MenuStatCardProps = {
  icon: ReactNode;
  label: string;
  value: string;
};

function MenuStatCard({ icon, label, value }: MenuStatCardProps) {
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

function summarizeMenus(groups: SystemMenuGroup[]) {
  return groups.reduce(
    (summary, group) => {
      summary.groups += 1;
      for (const item of group.items) {
        summary.items += 1;
        if (item.permission) {
          summary.protectedItems += 1;
        }
        if (item.mobile) {
          summary.mobileItems += 1;
        } else {
          summary.desktopOnlyItems += 1;
        }
      }
      return summary;
    },
    { desktopOnlyItems: 0, groups: 0, items: 0, mobileItems: 0, protectedItems: 0 },
  );
}

function toGroupOptions(groups: SystemMenuGroup[], allLabel: string): SelectOption[] {
  return [
    { label: allLabel, value: "" },
    ...groups.map((group) => ({
      label: `${group.label} (${group.items.length})`,
      value: group.code,
    })),
  ];
}

function filterMenuGroups(
  groups: SystemMenuGroup[],
  filters: MenuFilters,
  t: ReturnType<typeof useTranslation>["t"],
) {
  const keyword = filters.keyword.trim().toLowerCase();
  return groups
    .filter((group) => !filters.groupCode || group.code === filters.groupCode)
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => matchesMenu(group, item, keyword, t)),
    }))
    .filter((group) => group.items.length > 0);
}

function matchesMenu(
  group: SystemMenuGroup,
  item: SystemMenuItem,
  keyword: string,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (!keyword) {
    return true;
  }
  return [
    group.code,
    group.description,
    group.label,
    item.code,
    item.description,
    item.icon,
    item.label,
    item.path,
    item.permission,
    item.mobile ? t("admin.menus.entry.mobileAndDesktop") : t("admin.menus.entry.desktop"),
  ].some((value) => value?.toLowerCase().includes(keyword));
}

function formatNumber(value: number, locale: string) {
  return new Intl.NumberFormat(locale).format(value);
}

function errorTitle(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.menus.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.menus.states.errorTitle");
}

function errorDescription(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.menus.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

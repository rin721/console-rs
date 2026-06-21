import { useQuery } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import {
  Activity,
  Boxes,
  ListChecks,
  Plug,
  RadioTower,
  RefreshCw,
  Search,
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
import { pluginsApi } from "~/lib/api/plugins";
import { queryKeys } from "~/lib/api/query-keys";
import type { PluginCapability, PluginHealthStatus, PluginSnapshot } from "~/lib/api/types";

type PluginFilters = {
  keyword: string;
  status: string;
  transport: string;
};

const emptyFilters: PluginFilters = {
  keyword: "",
  status: "",
  transport: "",
};

export default function AdminPluginsRoute() {
  const { i18n, t } = useTranslation();
  const [filters, setFilters] = useState<PluginFilters>(emptyFilters);
  const [selectedPluginId, setSelectedPluginId] = useState<string>("");

  const pluginsQuery = useQuery({
    queryFn: ({ signal }) => pluginsApi.listPlugins({ signal }),
    queryKey: queryKeys.plugins.list(i18n.language),
  });

  const plugins = useMemo(() => pluginsQuery.data ?? [], [pluginsQuery.data]);
  const filteredPlugins = useMemo(() => filterPlugins(plugins, filters), [filters, plugins]);
  const selectedPlugin =
    plugins.find((plugin) => plugin.plugin_id === selectedPluginId) ?? plugins[0] ?? null;
  const selectedId = selectedPlugin?.plugin_id ?? "";

  const detailQuery = useQuery({
    enabled: selectedId.length > 0,
    queryFn: ({ signal }) => pluginsApi.getPlugin(selectedId, { signal }),
    queryKey: queryKeys.plugins.detail(i18n.language, selectedId),
  });
  const healthQuery = useQuery({
    enabled: selectedId.length > 0,
    queryFn: ({ signal }) => pluginsApi.getPluginHealth(selectedId, { signal }),
    queryKey: queryKeys.plugins.health(i18n.language, selectedId),
  });
  const capabilitiesQuery = useQuery({
    enabled: selectedId.length > 0,
    queryFn: ({ signal }) => pluginsApi.listPluginCapabilities(selectedId, { signal }),
    queryKey: queryKeys.plugins.capabilities(i18n.language, selectedId),
  });

  const summary = useMemo(() => summarizePlugins(plugins), [plugins]);
  const statusOptions = useMemo(() => pluginStatusOptions(plugins, t), [plugins, t]);
  const transportOptions = useMemo(() => pluginTransportOptions(plugins, t), [plugins, t]);
  const loading =
    pluginsQuery.isFetching ||
    detailQuery.isFetching ||
    healthQuery.isFetching ||
    capabilitiesQuery.isFetching;

  const pluginColumns = useMemo<ColumnDef<PluginSnapshot>[]>(
    () => [
      {
        accessorKey: "name",
        cell: ({ row }) => (
          <div className="aoi-plugin-name">
            <strong>{row.original.name || row.original.plugin_id}</strong>
            <span>{row.original.plugin_id}</span>
          </div>
        ),
        header: t("admin.plugins.columns.plugin"),
      },
      {
        accessorKey: "instance_id",
        cell: ({ getValue }) => <code>{String(getValue())}</code>,
        header: t("admin.plugins.columns.instance"),
      },
      {
        accessorKey: "status",
        cell: ({ row }) => (
          <span className="aoi-plugin-status" data-status={statusTone(row.original.status)}>
            {pluginStatusLabel(row.original.status, t)}
          </span>
        ),
        header: t("admin.plugins.columns.status"),
      },
      {
        accessorKey: "transport",
        cell: ({ row }) => (
          <span className="aoi-plugin-transport">
            {row.original.transport || row.original.protocol || t("admin.plugins.values.none")}
          </span>
        ),
        header: t("admin.plugins.columns.transport"),
      },
      {
        accessorKey: "capabilities",
        cell: ({ row }) => formatNumber(row.original.capabilities?.length ?? 0, i18n.language),
        header: t("admin.plugins.columns.capabilities"),
      },
      {
        accessorKey: "last_heartbeat_at",
        cell: ({ getValue }) => formatDate(String(getValue()), i18n.language, t),
        header: t("admin.plugins.columns.lastHeartbeat"),
      },
      {
        id: "actions",
        cell: ({ row }) => (
          <Button
            appearance={row.original.plugin_id === selectedId ? "primary" : "ghost"}
            icon={<ListChecks size={15} />}
            onClick={() => setSelectedPluginId(row.original.plugin_id)}
          >
            {row.original.plugin_id === selectedId
              ? t("admin.plugins.actions.selected")
              : t("admin.plugins.actions.view")}
          </Button>
        ),
        header: t("admin.plugins.columns.actions"),
      },
    ],
    [i18n.language, selectedId, t],
  );

  const detail = detailQuery.data ?? selectedPlugin;
  const capabilities =
    capabilitiesQuery.data?.capabilities ?? detail?.capabilities ?? selectedPlugin?.capabilities ?? [];

  const refresh = () => {
    void pluginsQuery.refetch();
    if (selectedId) {
      void detailQuery.refetch();
      void healthQuery.refetch();
      void capabilitiesQuery.refetch();
    }
  };

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-plugins-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.plugins.badge")}</Badge>
          <h1 id="admin-plugins-title">{t("admin.plugins.title")}</h1>
          <p>{t("admin.plugins.description")}</p>
        </div>
        <Button appearance="secondary" icon={<RefreshCw size={17} />} loading={loading} onClick={refresh}>
          {t("admin.plugins.actions.refresh")}
        </Button>
      </div>

      {pluginsQuery.error ? (
        <StateBlock
          intent={pluginsQuery.error instanceof ApiError && pluginsQuery.error.status === 403 ? "danger" : "info"}
          title={errorTitle(pluginsQuery.error, t)}
          description={errorDescription(pluginsQuery.error, t)}
        />
      ) : null}

      <div className="aoi-admin-stat-grid" aria-label={t("admin.plugins.summaryLabel")}>
        <PluginMetricCard
          icon={<Plug size={19} />}
          label={t("admin.plugins.metrics.total")}
          value={pluginsQuery.data ? formatNumber(plugins.length, i18n.language) : fallbackValue(pluginsQuery.isLoading, t)}
        />
        <PluginMetricCard
          icon={<Activity size={19} />}
          label={t("admin.plugins.metrics.online")}
          state={summary.online > 0 ? "success" : "info"}
          value={pluginsQuery.data ? formatNumber(summary.online, i18n.language) : fallbackValue(pluginsQuery.isLoading, t)}
        />
        <PluginMetricCard
          icon={<Boxes size={19} />}
          label={t("admin.plugins.metrics.capabilities")}
          value={pluginsQuery.data ? formatNumber(summary.capabilities, i18n.language) : fallbackValue(pluginsQuery.isLoading, t)}
        />
        <PluginMetricCard
          icon={<RadioTower size={19} />}
          label={t("admin.plugins.metrics.transports")}
          value={pluginsQuery.data ? formatNumber(summary.transports, i18n.language) : fallbackValue(pluginsQuery.isLoading, t)}
        />
      </div>

      <section className="aoi-admin-panel">
        <div className="aoi-admin-panel-header-row">
          <div>
            <h2>{t("admin.plugins.filters.title")}</h2>
            <p>{t("admin.plugins.filters.description")}</p>
          </div>
        </div>
        <form
          className="aoi-admin-filter-grid"
          onSubmit={(event) => {
            event.preventDefault();
          }}
        >
          <FormField
            label={t("admin.plugins.filters.keyword")}
            value={filters.keyword}
            onChange={(event) => setFilters({ ...filters, keyword: event.currentTarget.value })}
          />
          <SelectField
            label={t("admin.plugins.filters.status")}
            options={statusOptions}
            value={filters.status}
            onChange={(event) => setFilters({ ...filters, status: event.currentTarget.value })}
          />
          <SelectField
            label={t("admin.plugins.filters.transport")}
            options={transportOptions}
            value={filters.transport}
            onChange={(event) => setFilters({ ...filters, transport: event.currentTarget.value })}
          />
          <div className="aoi-admin-filter-actions">
            <Button icon={<Search size={17} />} loading={pluginsQuery.isFetching} type="submit">
              {t("admin.plugins.actions.search")}
            </Button>
            <Button appearance="ghost" onClick={() => setFilters(emptyFilters)} type="button">
              {t("admin.plugins.actions.reset")}
            </Button>
          </div>
        </form>
      </section>

      <section className="aoi-admin-panel">
        <div className="aoi-admin-panel-header-row">
          <div>
            <h2>{t("admin.plugins.list.title")}</h2>
            <p>
              {t("admin.plugins.list.description", {
                count: filteredPlugins.length,
                total: plugins.length,
              })}
            </p>
          </div>
        </div>
        {pluginsQuery.isLoading ? (
          <StateBlock
            title={t("admin.plugins.states.loadingTitle")}
            description={t("admin.plugins.states.loadingDescription")}
          />
        ) : plugins.length === 0 ? (
          <StateBlock
            title={t("admin.plugins.states.emptyTitle")}
            description={t("admin.plugins.states.emptyDescription")}
          />
        ) : filteredPlugins.length === 0 ? (
          <StateBlock
            title={t("admin.plugins.states.noMatchesTitle")}
            description={t("admin.plugins.states.noMatchesDescription")}
          />
        ) : (
          <div className="aoi-plugin-table">
            <DataTable
              columns={pluginColumns}
              data={filteredPlugins}
              emptyLabel={t("admin.plugins.states.emptyDescription")}
            />
          </div>
        )}
      </section>

      {detail ? (
        <section className="aoi-plugin-detail-grid">
          <section className="aoi-admin-panel">
            <div className="aoi-admin-panel-header-row">
              <div>
                <h2>{t("admin.plugins.detail.title")}</h2>
                <p>{t("admin.plugins.detail.description")}</p>
              </div>
            </div>
            {detailQuery.error ? (
              <StateBlock
                intent="danger"
                title={t("admin.plugins.states.detailErrorTitle")}
                description={errorDescription(detailQuery.error, t)}
              />
            ) : null}
            <PluginKeyValueList
              items={[
                [t("admin.plugins.detail.pluginId"), detail.plugin_id],
                [t("admin.plugins.detail.instanceId"), detail.instance_id],
                [t("admin.plugins.detail.name"), detail.name],
                [t("admin.plugins.detail.version"), detail.version],
                [t("admin.plugins.detail.protocol"), detail.protocol],
                [t("admin.plugins.detail.transport"), detail.transport || t("admin.plugins.values.none")],
                [t("admin.plugins.detail.endpoint"), detail.endpoint || t("admin.plugins.values.none")],
                [t("admin.plugins.detail.ownerHost"), detail.owner_host || t("admin.plugins.values.none")],
                [t("admin.plugins.detail.registeredAt"), formatDate(detail.registered_at, i18n.language, t)],
                [
                  t("admin.plugins.detail.leaseExpiresAt"),
                  formatDate(detail.lease_expires_at, i18n.language, t),
                ],
              ]}
            />
            <PluginPillList
              emptyLabel={t("admin.plugins.values.none")}
              label={t("admin.plugins.detail.permissions")}
              values={detail.permissions ?? []}
            />
            <PluginPillList
              emptyLabel={t("admin.plugins.values.none")}
              label={t("admin.plugins.detail.hooks")}
              values={detail.hooks ?? []}
            />
          </section>

          <section className="aoi-admin-panel">
            <div className="aoi-admin-panel-header-row">
              <div>
                <h2>{t("admin.plugins.health.title")}</h2>
                <p>{t("admin.plugins.health.description")}</p>
              </div>
            </div>
            {healthQuery.error ? (
              <StateBlock
                intent={healthQuery.error instanceof ApiError && healthQuery.error.status === 409 ? "info" : "danger"}
                title={healthErrorTitle(healthQuery.error, t)}
                description={errorDescription(healthQuery.error, t)}
              />
            ) : healthQuery.isLoading ? (
              <StateBlock
                title={t("admin.plugins.states.healthLoadingTitle")}
                description={t("admin.plugins.states.healthLoadingDescription")}
              />
            ) : healthQuery.data ? (
              <PluginHealthPanel health={healthQuery.data} locale={i18n.language} />
            ) : (
              <StateBlock
                title={t("admin.plugins.states.healthEmptyTitle")}
                description={t("admin.plugins.states.healthEmptyDescription")}
              />
            )}
          </section>
        </section>
      ) : null}

      {detail ? (
        <section className="aoi-admin-panel">
          <div className="aoi-admin-panel-header-row">
            <div>
              <h2>{t("admin.plugins.capabilities.title")}</h2>
              <p>{t("admin.plugins.capabilities.description")}</p>
            </div>
          </div>
          {capabilitiesQuery.error ? (
            <StateBlock
              intent={capabilitiesQuery.error instanceof ApiError && capabilitiesQuery.error.status === 409 ? "info" : "danger"}
              title={capabilityErrorTitle(capabilitiesQuery.error, t)}
              description={errorDescription(capabilitiesQuery.error, t)}
            />
          ) : capabilitiesQuery.isLoading ? (
            <StateBlock
              title={t("admin.plugins.states.capabilitiesLoadingTitle")}
              description={t("admin.plugins.states.capabilitiesLoadingDescription")}
            />
          ) : capabilities.length > 0 ? (
            <div className="aoi-plugin-capability-grid">
              {capabilities.map((capability) => (
                <PluginCapabilityCard key={capabilityKey(capability)} capability={capability} />
              ))}
            </div>
          ) : (
            <StateBlock
              title={t("admin.plugins.states.capabilitiesEmptyTitle")}
              description={t("admin.plugins.states.capabilitiesEmptyDescription")}
            />
          )}
        </section>
      ) : null}
    </section>
  );
}

function PluginMetricCard({
  icon,
  label,
  state,
  value,
}: {
  icon: ReactNode;
  label: string;
  state?: "danger" | "info" | "success" | "warning";
  value: string;
}) {
  return (
    <article className="aoi-admin-stat-card" data-state={state}>
      <span aria-hidden="true">{icon}</span>
      <div>
        <p>{label}</p>
        <strong>{value}</strong>
      </div>
    </article>
  );
}

function PluginHealthPanel({ health, locale }: { health: PluginHealthStatus; locale: string }) {
  const { t } = useTranslation();
  return (
    <PluginKeyValueList
      items={[
        [t("admin.plugins.health.status"), pluginStatusLabel(health.status, t)],
        [t("admin.plugins.health.runtimeStatus"), health.runtime_status || t("admin.plugins.values.none")],
        [t("admin.plugins.health.pluginId"), health.plugin_id],
        [t("admin.plugins.health.instanceId"), health.instance_id || t("admin.plugins.values.none")],
        [t("admin.plugins.health.lastHeartbeatAt"), formatDate(health.last_heartbeat_at, locale, t)],
        [t("admin.plugins.health.leaseExpiresAt"), formatDate(health.lease_expires_at, locale, t)],
        [t("admin.plugins.health.error"), health.error || t("admin.plugins.values.none")],
      ]}
    />
  );
}

function PluginKeyValueList({ items }: { items: Array<[string, string]> }) {
  return (
    <dl className="aoi-plugin-key-values">
      {items.map(([label, value]) => (
        <div key={label}>
          <dt>{label}</dt>
          <dd>{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function PluginPillList({
  emptyLabel,
  label,
  values,
}: {
  emptyLabel: string;
  label: string;
  values: string[];
}) {
  return (
    <div className="aoi-plugin-pill-list">
      <span>{label}</span>
      <div>
        {values.length > 0 ? values.map((value) => <Badge key={value}>{value}</Badge>) : <Badge>{emptyLabel}</Badge>}
      </div>
    </div>
  );
}

function PluginCapabilityCard({ capability }: { capability: PluginCapability }) {
  const { t } = useTranslation();
  return (
    <article className="aoi-plugin-capability-card">
      <div>
        <h3>{capability.name}</h3>
        <p>{capability.description || t("admin.plugins.capabilities.noDescription")}</p>
      </div>
      <PluginKeyValueList
        items={[
          [t("admin.plugins.capabilities.version"), capability.version || t("admin.plugins.values.none")],
          [t("admin.plugins.capabilities.scope"), capability.scope || t("admin.plugins.values.none")],
          [
            t("admin.plugins.capabilities.secretPolicy"),
            capability.secret_policy || t("admin.plugins.values.none"),
          ],
        ]}
      />
      <PluginPillList
        emptyLabel={t("admin.plugins.values.none")}
        label={t("admin.plugins.capabilities.permissions")}
        values={capability.permissions ?? []}
      />
      <div className="aoi-plugin-schema-row">
        <Badge>
          {capability.input_schema
            ? t("admin.plugins.capabilities.inputSchemaPresent")
            : t("admin.plugins.capabilities.inputSchemaEmpty")}
        </Badge>
        <Badge>
          {capability.output_schema
            ? t("admin.plugins.capabilities.outputSchemaPresent")
            : t("admin.plugins.capabilities.outputSchemaEmpty")}
        </Badge>
      </div>
    </article>
  );
}

function filterPlugins(plugins: PluginSnapshot[], filters: PluginFilters) {
  const keyword = filters.keyword.trim().toLowerCase();
  return plugins.filter((plugin) => {
    const values = [
      plugin.plugin_id,
      plugin.instance_id,
      plugin.name,
      plugin.version,
      plugin.protocol,
      plugin.transport ?? "",
      plugin.endpoint ?? "",
      plugin.status,
      plugin.runtime_status ?? "",
    ];
    const keywordMatch =
      keyword.length === 0 || values.some((value) => value.toLowerCase().includes(keyword));
    const statusMatch = !filters.status || plugin.status === filters.status;
    const transportMatch =
      !filters.transport || (plugin.transport || plugin.protocol) === filters.transport;
    return keywordMatch && statusMatch && transportMatch;
  });
}

function summarizePlugins(plugins: PluginSnapshot[]) {
  const transports = new Set<string>();
  return plugins.reduce(
    (summary, plugin) => {
      if (plugin.status === "online") {
        summary.online += 1;
      }
      summary.capabilities += plugin.capabilities?.length ?? 0;
      const transport = plugin.transport || plugin.protocol;
      if (transport) {
        transports.add(transport);
      }
      summary.transports = transports.size;
      return summary;
    },
    { capabilities: 0, online: 0, transports: 0 },
  );
}

function pluginStatusOptions(
  plugins: PluginSnapshot[],
  t: ReturnType<typeof useTranslation>["t"],
): SelectOption[] {
  const statuses = Array.from(new Set(plugins.map((plugin) => plugin.status).filter(Boolean))).sort();
  return [
    { label: t("admin.plugins.filters.allStatuses"), value: "" },
    ...statuses.map((status) => ({ label: pluginStatusLabel(status, t), value: status })),
  ];
}

function pluginTransportOptions(
  plugins: PluginSnapshot[],
  t: ReturnType<typeof useTranslation>["t"],
): SelectOption[] {
  const transports = Array.from(
    new Set(plugins.map((plugin) => plugin.transport || plugin.protocol).filter(Boolean)),
  ).sort();
  return [
    { label: t("admin.plugins.filters.allTransports"), value: "" },
    ...transports.map((transport) => ({ label: transport, value: transport })),
  ];
}

function statusTone(status: string) {
  switch (status) {
    case "online":
      return "success";
    case "draining":
      return "warning";
    case "offline":
      return "danger";
    default:
      return "info";
  }
}

function pluginStatusLabel(status: string, t: ReturnType<typeof useTranslation>["t"]) {
  switch (status) {
    case "online":
      return t("admin.plugins.status.online");
    case "offline":
      return t("admin.plugins.status.offline");
    case "draining":
      return t("admin.plugins.status.draining");
    default:
      return status || t("admin.plugins.status.unknown");
  }
}

function formatNumber(value: number, locale: string) {
  return new Intl.NumberFormat(locale).format(value);
}

function formatDate(
  value: string | undefined,
  locale: string,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (!value) {
    return t("admin.plugins.values.none");
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function fallbackValue(loading: boolean, t: ReturnType<typeof useTranslation>["t"]) {
  return loading ? t("states.loading") : t("admin.plugins.values.none");
}

function errorTitle(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.plugins.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 404) {
    return t("admin.plugins.states.disabledTitle");
  }
  return t("admin.plugins.states.errorTitle");
}

function healthErrorTitle(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 409) {
    return t("admin.plugins.states.offlineTitle");
  }
  return errorTitle(error, t);
}

function capabilityErrorTitle(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 409) {
    return t("admin.plugins.states.capabilitiesOfflineTitle");
  }
  return errorTitle(error, t);
}

function errorDescription(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.plugins.states.permissionDescription");
  }
  if (error instanceof ApiError && error.status === 404) {
    return t("admin.plugins.states.disabledDescription");
  }
  if (error instanceof ApiError && error.status === 409) {
    return t("admin.plugins.states.offlineDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

function capabilityKey(capability: PluginCapability) {
  return `${capability.name}:${capability.version ?? ""}:${capability.scope ?? ""}`;
}

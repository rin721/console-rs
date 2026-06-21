import { useQuery } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import type { EChartsOption } from "echarts";
import {
  Activity,
  Cpu,
  DatabaseZap,
  FileClock,
  HardDrive,
  ListChecks,
  MemoryStick,
  Network,
  RefreshCw,
  Server,
  ShieldAlert,
  TimerReset,
} from "lucide-react";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { DataTable } from "~/components/aoi/patterns/DataTable";
import { EChart } from "~/components/aoi/patterns/EChart";
import { PanelSkeleton, StatGridSkeleton } from "~/components/aoi/patterns/LoadingSkeletons";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { ApiError } from "~/lib/api/client";
import { queryKeys } from "~/lib/api/query-keys";
import { systemApi } from "~/lib/api/system";
import { useThemeChartPalette, type ThemeChartPalette } from "~/theme/chart-palette";
import type {
  SystemAPIGroup,
  SystemServerInfo,
  SystemServerMetricsHistory,
  SystemServerMetricsSample,
  SystemTrafficHijackOverview,
  SystemTrafficProbeResult,
  SystemVersionRecord,
} from "~/lib/api/types";

const versionPageSize = 5;
const serverMetricsRefreshIntervalMs = 5_000;
const systemStatusRefreshIntervalMs = 15_000;
const allDiskValue = "__all__";

type ServerMetricMode = "disk" | "network";

type DiskMetricPoint = {
  ioLatencyMs: number;
  name: string;
  readMbPerSecond: number;
  readOpsPerSecond: number;
  writeMbPerSecond: number;
  writeOpsPerSecond: number;
};

export default function AdminDashboardRoute() {
  const { i18n, t } = useTranslation();

  const healthQuery = useQuery({
    queryFn: ({ signal }) => systemApi.getHealth({ signal }),
    queryKey: queryKeys.system.health,
    refetchInterval: systemStatusRefreshIntervalMs,
  });
  const readyQuery = useQuery({
    queryFn: ({ signal }) => systemApi.getReady({ signal }),
    queryKey: queryKeys.system.ready,
    refetchInterval: systemStatusRefreshIntervalMs,
  });
  const serverInfoQuery = useQuery({
    queryFn: ({ signal }) => systemApi.getServerInfo({ signal }),
    queryKey: queryKeys.system.serverInfo,
    refetchInterval: serverMetricsRefreshIntervalMs,
  });
  const serverMetricsQuery = useQuery({
    queryFn: ({ signal }) => systemApi.getServerMetricsHistory({ signal }),
    queryKey: queryKeys.system.serverMetricsHistory,
    refetchInterval: serverMetricsRefreshIntervalMs,
  });
  const trafficHijackQuery = useQuery({
    queryFn: ({ signal }) => systemApi.getTrafficHijackOverview({ signal }),
    queryKey: queryKeys.system.trafficHijackOverview(i18n.language),
    refetchInterval: 30_000,
  });
  const apiCatalogQuery = useQuery({
    queryFn: ({ signal }) => systemApi.listAPIs({ signal }),
    queryKey: queryKeys.system.apiCatalog(i18n.language),
  });
  const versionsQuery = useQuery({
    queryFn: ({ signal }) =>
      systemApi.listVersions({ page: 1, pageSize: versionPageSize }, { signal }),
    queryKey: queryKeys.system.versions(i18n.language, 1, versionPageSize),
  });

  const queries = [
    healthQuery,
    readyQuery,
    serverInfoQuery,
    serverMetricsQuery,
    trafficHijackQuery,
    apiCatalogQuery,
    versionsQuery,
  ];
  const loading = queries.some((query) => query.isLoading);

  const refresh = () => {
    for (const query of queries) {
      void query.refetch();
    }
  };

  const apiSummary = summarizeAPIs(apiCatalogQuery.data ?? []);
  const versionItems = versionsQuery.data?.items ?? [];
  const latestVersion = versionItems[0];

  const apiColumns = useMemo<ColumnDef<SystemAPIGroup>[]>(
    () => [
      {
        accessorKey: "label",
        header: t("admin.dashboard.apiCatalog.columns.group"),
      },
      {
        accessorKey: "count",
        cell: ({ getValue }) => formatNumber(Number(getValue()), i18n.language),
        header: t("admin.dashboard.apiCatalog.columns.count"),
      },
      {
        id: "permissionProtected",
        cell: ({ row }) =>
          formatNumber(
            row.original.items.filter((item) => item.access === "permission").length,
            i18n.language,
          ),
        header: t("admin.dashboard.apiCatalog.columns.permissionProtected"),
      },
    ],
    [i18n.language, t],
  );

  const versionColumns = useMemo<ColumnDef<SystemVersionRecord>[]>(
    () => [
      {
        accessorKey: "versionName",
        header: t("admin.dashboard.versions.columns.name"),
      },
      {
        accessorKey: "versionCode",
        header: t("admin.dashboard.versions.columns.code"),
      },
      {
        accessorKey: "source",
        cell: ({ getValue }) => versionSourceLabel(String(getValue()), t),
        header: t("admin.dashboard.versions.columns.source"),
      },
      {
        accessorKey: "createdAt",
        cell: ({ getValue }) => formatDate(String(getValue()), i18n.language),
        header: t("admin.dashboard.versions.columns.createdAt"),
      },
    ],
    [i18n.language, t],
  );

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-dashboard-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.dashboard.backendBacked")}</Badge>
          <h1 id="admin-dashboard-title">{t("admin.dashboard.title")}</h1>
          <p>{t("admin.dashboard.description")}</p>
        </div>
        <Button
          appearance="secondary"
          icon={<RefreshCw size={17} />}
          loading={loading}
          onClick={refresh}
        >
          {t("admin.dashboard.actions.refresh")}
        </Button>
      </div>

      {loading ? (
        <StatGridSkeleton />
      ) : (
        <div className="aoi-admin-stat-grid" aria-label={t("admin.dashboard.summaryLabel")}>
          <MetricCard
            icon={<Activity size={19} />}
            label={t("admin.dashboard.probes.health")}
            state={healthQuery.data?.status ?? statusFromError(healthQuery.error, t)}
            value={statusLabel(
              healthQuery.data?.status,
              healthQuery.isLoading,
              healthQuery.error,
              t,
            )}
          />
          <MetricCard
            icon={<DatabaseZap size={19} />}
            label={t("admin.dashboard.probes.ready")}
            state={readyQuery.data?.status ?? statusFromError(readyQuery.error, t)}
            value={statusLabel(readyQuery.data?.status, readyQuery.isLoading, readyQuery.error, t)}
          />
          <MetricCard
            icon={<Server size={19} />}
            label={t("admin.dashboard.metrics.uptime")}
            value={
              serverInfoQuery.data?.runtime.uptime ?? fallbackValue(serverInfoQuery.isLoading, t)
            }
          />
          <MetricCard
            icon={<ListChecks size={19} />}
            label={t("admin.dashboard.metrics.apiCount")}
            value={
              apiCatalogQuery.data
                ? formatNumber(apiSummary.total, i18n.language)
                : fallbackValue(apiCatalogQuery.isLoading, t)
            }
          />
          <MetricCard
            icon={<FileClock size={19} />}
            label={t("admin.dashboard.metrics.versionCount")}
            value={
              versionsQuery.data
                ? formatNumber(versionsQuery.data.total, i18n.language)
                : fallbackValue(versionsQuery.isLoading, t)
            }
          />
          <MetricCard
            icon={<ShieldAlert size={19} />}
            label={t("admin.dashboard.trafficHijack.metric")}
            state={
              trafficHijackQuery.data && trafficHijackQuery.data.criticalTargets > 0
                ? "danger"
                : trafficHijackQuery.data && trafficHijackQuery.data.warningTargets > 0
                  ? "warning"
                  : "success"
            }
            value={
              trafficHijackQuery.data
                ? t("admin.dashboard.trafficHijack.metricValue", {
                    abnormal:
                      trafficHijackQuery.data.warningTargets +
                      trafficHijackQuery.data.criticalTargets,
                    total: trafficHijackQuery.data.totalTargets,
                  })
                : fallbackValue(trafficHijackQuery.isLoading, t)
            }
          />
        </div>
      )}

      <div className="aoi-admin-panel-grid aoi-admin-panel-grid--dashboard">
        <DashboardPanel
          className="aoi-admin-panel--span-2"
          description={t("admin.dashboard.serverInfo.description")}
          error={serverInfoQuery.error}
          loading={serverInfoQuery.isLoading}
          title={t("admin.dashboard.serverInfo.title")}
        >
          {serverInfoQuery.data ? <ServerResourceOverview info={serverInfoQuery.data} /> : null}
        </DashboardPanel>

        <DashboardPanel
          className="aoi-admin-panel--span-2"
          description={t("admin.dashboard.serverMetrics.description")}
          error={serverMetricsQuery.error}
          loading={serverMetricsQuery.isLoading}
          title={t("admin.dashboard.serverMetrics.title")}
        >
          {serverMetricsQuery.data ? (
            <ServerMetricsHistoryPanel history={serverMetricsQuery.data} />
          ) : null}
        </DashboardPanel>

        <DashboardPanel
          className="aoi-admin-panel--span-2"
          description={t("admin.dashboard.trafficHijack.description")}
          error={trafficHijackQuery.error}
          loading={trafficHijackQuery.isLoading}
          title={t("admin.dashboard.trafficHijack.title")}
        >
          {trafficHijackQuery.data ? (
            <TrafficHijackOverviewPanel overview={trafficHijackQuery.data} />
          ) : null}
        </DashboardPanel>

        <DashboardPanel
          description={t("admin.dashboard.runtime.description")}
          error={serverInfoQuery.error}
          loading={serverInfoQuery.isLoading}
          title={t("admin.dashboard.runtime.title")}
        >
          {serverInfoQuery.data ? (
            <KeyValueList
              columns={2}
              items={[
                [t("admin.dashboard.runtime.rows.goVersion"), serverInfoQuery.data.os.goVersion],
                [t("admin.dashboard.runtime.rows.platform"), platformLabel(serverInfoQuery.data)],
                [
                  t("admin.dashboard.runtime.rows.cpus"),
                  formatNumber(serverInfoQuery.data.os.numCpu, i18n.language),
                ],
                [
                  t("admin.dashboard.runtime.rows.goroutines"),
                  formatNumber(serverInfoQuery.data.os.numGoroutine, i18n.language),
                ],
                [
                  t("admin.dashboard.runtime.rows.startTime"),
                  formatDate(serverInfoQuery.data.runtime.startTime, i18n.language),
                ],
                [
                  t("admin.dashboard.runtime.rows.refreshedAt"),
                  formatDate(serverInfoQuery.data.refreshedAt, i18n.language),
                ],
              ]}
            />
          ) : null}
        </DashboardPanel>

        <DashboardPanel
          description={t("admin.dashboard.process.description")}
          error={serverInfoQuery.error}
          loading={serverInfoQuery.isLoading}
          title={t("admin.dashboard.process.title")}
        >
          {serverInfoQuery.data ? (
            <KeyValueList
              columns={2}
              items={[
                [
                  t("admin.dashboard.process.rows.heapAlloc"),
                  sizeLabel(serverInfoQuery.data.memory.heapAllocMb, i18n.language),
                ],
                [
                  t("admin.dashboard.process.rows.heapSys"),
                  sizeLabel(serverInfoQuery.data.memory.heapSysMb, i18n.language),
                ],
                [
                  t("admin.dashboard.process.rows.gcCount"),
                  formatNumber(serverInfoQuery.data.gc.numGc, i18n.language),
                ],
                [
                  t("admin.dashboard.process.rows.gcNext"),
                  sizeLabel(serverInfoQuery.data.gc.nextGcMb, i18n.language),
                ],
                [
                  t("admin.dashboard.process.rows.gcPause"),
                  durationLabel(serverInfoQuery.data.gc.pauseTotalNs, i18n.language),
                ],
                [
                  t("admin.dashboard.process.rows.buildVersion"),
                  serverInfoQuery.data.build.version || t("common.labels.none"),
                ],
              ]}
            />
          ) : null}
        </DashboardPanel>

        <DashboardPanel
          description={t("admin.dashboard.apiCatalog.description")}
          error={apiCatalogQuery.error}
          loading={apiCatalogQuery.isLoading}
          title={t("admin.dashboard.apiCatalog.title")}
        >
          {apiCatalogQuery.data ? (
            <>
              <div className="aoi-admin-inline-stats">
                <span>
                  {t("admin.dashboard.apiCatalog.summary.total", { count: apiSummary.total })}
                </span>
                <span>
                  {t("admin.dashboard.apiCatalog.summary.permission", {
                    count: apiSummary.permission,
                  })}
                </span>
                <span>
                  {t("admin.dashboard.apiCatalog.summary.public", { count: apiSummary.public })}
                </span>
              </div>
              <DataTable
                columns={apiColumns}
                data={apiCatalogQuery.data.slice(0, 5)}
                emptyLabel={t("admin.dashboard.apiCatalog.empty")}
              />
            </>
          ) : null}
        </DashboardPanel>

        <DashboardPanel
          description={t("admin.dashboard.versions.description")}
          error={versionsQuery.error}
          loading={versionsQuery.isLoading}
          title={t("admin.dashboard.versions.title")}
        >
          {versionsQuery.data ? (
            <>
              <div className="aoi-admin-inline-stats">
                <span>
                  {t("admin.dashboard.versions.summary.storage", {
                    status: versionsQuery.data.storageStatus,
                  })}
                </span>
                <span>
                  {latestVersion
                    ? t("admin.dashboard.versions.summary.latest", {
                        name: latestVersion.versionName,
                      })
                    : t("admin.dashboard.versions.summary.none")}
                </span>
              </div>
              <DataTable
                columns={versionColumns}
                data={versionItems}
                emptyLabel={t("admin.dashboard.versions.empty")}
              />
            </>
          ) : null}
        </DashboardPanel>
      </div>
    </section>
  );
}

type MetricCardProps = {
  icon: ReactNode;
  label: string;
  state?: string;
  value: string;
};

function MetricCard({ icon, label, state, value }: MetricCardProps) {
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

type DashboardPanelProps = {
  children: ReactNode;
  className?: string;
  description: string;
  error: Error | null;
  loading: boolean;
  title: string;
};

function DashboardPanel({
  children,
  className,
  description,
  error,
  loading,
  title,
}: DashboardPanelProps) {
  const { t } = useTranslation();

  return (
    <article className={["aoi-admin-panel", className].filter(Boolean).join(" ")}>
      <header>
        <h2>{title}</h2>
        <p>{description}</p>
      </header>
      {error ? (
        <StateBlock
          intent="danger"
          title={errorTitle(error, t)}
          description={errorDescription(error, t)}
        />
      ) : loading ? (
        <PanelSkeleton />
      ) : (
        children
      )}
    </article>
  );
}

function ServerResourceOverview({ info }: { info: SystemServerInfo }) {
  const { i18n, t } = useTranslation();
  const palette = useThemeChartPalette();
  const cpuAverage = average(info.cpu.percent);
  const disk = maxDisk(info.disk);

  return (
    <div className="aoi-server-overview">
      <div
        className="aoi-server-gauge-grid"
        aria-label={t("admin.dashboard.serverInfo.gauges.label")}
      >
        <ResourceGauge
          ariaLabel={t("admin.dashboard.serverInfo.gauges.cpuAria")}
          color={palette.secondary}
          detail={t("admin.dashboard.serverInfo.gauges.cpuDetail", {
            cores: formatNumber(info.cpu.cores || info.os.numCpu, i18n.language),
          })}
          label={t("admin.dashboard.serverInfo.gauges.cpu")}
          trackColor={palette.track}
          value={cpuAverage}
          valueLabel={percentLabel(cpuAverage, i18n.language)}
        />
        <ResourceGauge
          ariaLabel={t("admin.dashboard.serverInfo.gauges.ramAria")}
          color={palette.primary}
          detail={t("admin.dashboard.serverInfo.gauges.ramDetail", {
            total: sizeLabel(info.ram.totalMb, i18n.language),
            used: sizeLabel(info.ram.usedMb, i18n.language),
          })}
          label={t("admin.dashboard.serverInfo.gauges.ram")}
          trackColor={palette.track}
          value={info.ram.usedPercent}
          valueLabel={percentLabel(info.ram.usedPercent, i18n.language)}
        />
        <ResourceGauge
          ariaLabel={t("admin.dashboard.serverInfo.gauges.diskAria")}
          color={disk ? palette.warning : palette.border}
          detail={
            disk
              ? t("admin.dashboard.serverInfo.gauges.diskDetail", {
                  mount: disk.mountPoint,
                  total: sizeLabel(disk.totalMb, i18n.language),
                  used: sizeLabel(disk.usedMb, i18n.language),
                })
              : t("admin.dashboard.serverInfo.gauges.diskEmpty")
          }
          label={t("admin.dashboard.serverInfo.gauges.disk")}
          trackColor={palette.track}
          value={disk?.usedPercent ?? 0}
          valueLabel={
            disk ? percentLabel(disk.usedPercent, i18n.language) : t("common.labels.none")
          }
        />
      </div>

      <dl className="aoi-server-quick-facts">
        <div>
          <dt>
            <TimerReset aria-hidden="true" size={16} />
            {t("admin.dashboard.serverInfo.quick.uptime")}
          </dt>
          <dd>{info.runtime.uptime}</dd>
        </div>
        <div>
          <dt>
            <Cpu aria-hidden="true" size={16} />
            {t("admin.dashboard.serverInfo.quick.platform")}
          </dt>
          <dd>{platformLabel(info)}</dd>
        </div>
        <div>
          <dt>
            <MemoryStick aria-hidden="true" size={16} />
            {t("admin.dashboard.serverInfo.quick.heap")}
          </dt>
          <dd>{sizeLabel(info.memory.heapAllocMb, i18n.language)}</dd>
        </div>
        <div>
          <dt>
            <HardDrive aria-hidden="true" size={16} />
            {t("admin.dashboard.serverInfo.quick.disks")}
          </dt>
          <dd>{formatNumber(info.disk.length, i18n.language)}</dd>
        </div>
      </dl>
    </div>
  );
}

type ResourceGaugeProps = {
  ariaLabel: string;
  color: string;
  detail: string;
  label: string;
  trackColor: string;
  value: number;
  valueLabel: string;
};

function ResourceGauge({
  ariaLabel,
  color,
  detail,
  label,
  trackColor,
  value,
  valueLabel,
}: ResourceGaugeProps) {
  const option = useMemo(
    () => resourceGaugeOption(value, color, trackColor),
    [color, trackColor, value],
  );

  return (
    <article className="aoi-server-gauge-card">
      <EChart ariaLabel={ariaLabel} className="aoi-server-gauge-chart" option={option} />
      <div className="aoi-server-gauge-card__copy">
        <strong>{valueLabel}</strong>
        <span>{label}</span>
        <small>{detail}</small>
      </div>
    </article>
  );
}

function ServerMetricsHistoryPanel({ history }: { history: SystemServerMetricsHistory }) {
  const { i18n, t } = useTranslation();
  const palette = useThemeChartPalette();
  const [mode, setMode] = useState<ServerMetricMode>("network");
  const [selectedDisk, setSelectedDisk] = useState(allDiskValue);
  const samples = history.samples ?? [];
  const latest = samples.at(-1);
  const diskNames = useMemo(() => diskIONames(samples), [samples]);
  const latestDisk = latest ? diskMetricForSample(latest, selectedDisk) : undefined;
  const networkOption = useMemo(
    () => networkHistoryOption(samples, palette, i18n.language, t),
    [i18n.language, palette, samples, t],
  );
  const diskOption = useMemo(
    () => diskIOHistoryOption(samples, selectedDisk, palette, i18n.language, t),
    [i18n.language, palette, samples, selectedDisk, t],
  );

  useEffect(() => {
    if (selectedDisk !== allDiskValue && !diskNames.includes(selectedDisk)) {
      setSelectedDisk(allDiskValue);
    }
  }, [diskNames, selectedDisk]);

  if (samples.length === 0) {
    return (
      <StateBlock
        title={t("admin.dashboard.serverMetrics.emptyTitle")}
        description={t("admin.dashboard.serverMetrics.emptyDescription")}
      />
    );
  }

  return (
    <div className="aoi-server-monitor-panel">
      <div className="aoi-server-monitor-toolbar">
        <div
          className="aoi-server-monitor-tabs"
          aria-label={t("admin.dashboard.serverMetrics.modeLabel")}
          role="tablist"
        >
          <button
            aria-selected={mode === "network"}
            onClick={() => setMode("network")}
            role="tab"
            type="button"
          >
            {t("admin.dashboard.serverMetrics.modes.network")}
          </button>
          <button
            aria-selected={mode === "disk"}
            onClick={() => setMode("disk")}
            role="tab"
            type="button"
          >
            {t("admin.dashboard.serverMetrics.modes.disk")}
          </button>
        </div>
        {mode === "disk" ? (
          <label className="aoi-server-monitor-select">
            <span>{t("admin.dashboard.serverMetrics.filters.disk")}</span>
            <select
              aria-label={t("admin.dashboard.serverMetrics.filters.disk")}
              onChange={(event) => setSelectedDisk(event.target.value)}
              value={selectedDisk}
            >
              <option value={allDiskValue}>
                {t("admin.dashboard.serverMetrics.filters.allDisks")}
              </option>
              {diskNames.map((name) => (
                <option key={name} value={name}>
                  {name}
                </option>
              ))}
            </select>
          </label>
        ) : null}
      </div>
      <div className="aoi-admin-inline-stats">
        <span>
          {t("admin.dashboard.serverMetrics.summary.interval", {
            seconds: history.intervalSeconds,
          })}
        </span>
        <span>
          {t("admin.dashboard.serverMetrics.summary.window", {
            seconds: history.windowSeconds,
          })}
        </span>
        {latest ? (
          mode === "network" ? (
            <>
              <span>
                {t("admin.dashboard.serverMetrics.summary.receive", {
                  value: rateLabel(latest.networkReceiveKbPerSecond, i18n.language),
                })}
              </span>
              <span>
                {t("admin.dashboard.serverMetrics.summary.transmit", {
                  value: rateLabel(latest.networkTransmitKbPerSecond, i18n.language),
                })}
              </span>
            </>
          ) : latestDisk ? (
            <>
              <span>
                {t("admin.dashboard.serverMetrics.summary.read", {
                  value: mbRateLabel(latestDisk.readMbPerSecond, i18n.language),
                })}
              </span>
              <span>
                {t("admin.dashboard.serverMetrics.summary.write", {
                  value: mbRateLabel(latestDisk.writeMbPerSecond, i18n.language),
                })}
              </span>
              <span>
                {t("admin.dashboard.serverMetrics.summary.readOps", {
                  value: opsRateLabel(latestDisk.readOpsPerSecond, i18n.language),
                })}
              </span>
              <span>
                {t("admin.dashboard.serverMetrics.summary.writeOps", {
                  value: opsRateLabel(latestDisk.writeOpsPerSecond, i18n.language),
                })}
              </span>
              <span>
                {t("admin.dashboard.serverMetrics.summary.latency", {
                  value: latencyLabel(latestDisk.ioLatencyMs, i18n.language),
                })}
              </span>
            </>
          ) : (
            <span>{t("admin.dashboard.serverMetrics.summary.noDiskIo")}</span>
          )
        ) : null}
      </div>
      {mode === "disk" && diskNames.length === 0 ? (
        <StateBlock
          title={t("admin.dashboard.serverMetrics.diskEmptyTitle")}
          description={t("admin.dashboard.serverMetrics.diskEmptyDescription")}
        />
      ) : (
        <EChart
          ariaLabel={
            mode === "disk"
              ? t("admin.dashboard.serverMetrics.diskChartAria")
              : t("admin.dashboard.serverMetrics.chartAria")
          }
          className="aoi-server-monitor-chart"
          option={mode === "disk" ? diskOption : networkOption}
        />
      )}
    </div>
  );
}

function TrafficHijackOverviewPanel({ overview }: { overview: SystemTrafficHijackOverview }) {
  const { i18n, t } = useTranslation();
  const palette = useThemeChartPalette();
  const abnormal = overview.warningTargets + overview.criticalTargets;
  const latestEvent = overview.recentEvents[0];
  const option = useMemo(
    () => trafficHijackLatencyOption(overview.recentResults, palette, i18n.language, t),
    [i18n.language, overview.recentResults, palette, t],
  );

  return (
    <div className="aoi-traffic-dashboard-panel">
      <div className="aoi-admin-inline-stats">
        <span>
          {t("admin.dashboard.trafficHijack.summary.healthy", {
            count: overview.healthyTargets,
          })}
        </span>
        <span>
          {t("admin.dashboard.trafficHijack.summary.abnormal", {
            count: abnormal,
          })}
        </span>
        <span>
          {latestEvent
            ? t("admin.dashboard.trafficHijack.summary.latestEvent", {
                reason: latestEvent.reason,
              })
            : t("admin.dashboard.trafficHijack.summary.noEvents")}
        </span>
      </div>
      {overview.recentResults.length > 0 ? (
        <EChart
          ariaLabel={t("admin.dashboard.trafficHijack.chartAria")}
          className="aoi-traffic-dashboard-chart"
          option={option}
        />
      ) : (
        <StateBlock
          title={t("admin.dashboard.trafficHijack.emptyTitle")}
          description={t("admin.dashboard.trafficHijack.emptyDescription")}
        />
      )}
    </div>
  );
}

function KeyValueList({
  columns = 1,
  items,
}: {
  columns?: 1 | 2 | 3;
  items: Array<[string, string]>;
}) {
  return (
    <dl className="aoi-admin-key-values" data-columns={columns}>
      {items.map(([label, value]) => (
        <div key={label}>
          <dt>{label}</dt>
          <dd>{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function resourceGaugeOption(value: number, color: string, trackColor: string): EChartsOption {
  const normalized = Math.max(0, Math.min(100, value));
  return {
    color: [color, trackColor],
    series: [
      {
        avoidLabelOverlap: true,
        data: [{ value: normalized }, { value: Math.max(0, 100 - normalized) }],
        emphasis: { disabled: true },
        label: { show: false },
        radius: ["70%", "86%"],
        silent: true,
        type: "pie",
      },
    ],
    tooltip: { show: false },
  };
}

function networkHistoryOption(
  samples: SystemServerMetricsSample[],
  palette: ThemeChartPalette,
  locale: string,
  t: ReturnType<typeof useTranslation>["t"],
): EChartsOption {
  return {
    color: [palette.secondary, palette.primary],
    grid: {
      bottom: 28,
      containLabel: true,
      left: 10,
      right: 16,
      top: 34,
    },
    legend: {
      right: 0,
      textStyle: { color: palette.textSecondary },
      top: 0,
    },
    series: [
      {
        areaStyle: { opacity: 0.18 },
        data: samples.map((sample) => sample.networkReceiveKbPerSecond),
        name: t("admin.dashboard.serverMetrics.series.receive"),
        showSymbol: false,
        smooth: true,
        type: "line",
      },
      {
        areaStyle: { opacity: 0.14 },
        data: samples.map((sample) => sample.networkTransmitKbPerSecond),
        name: t("admin.dashboard.serverMetrics.series.transmit"),
        showSymbol: false,
        smooth: true,
        type: "line",
      },
    ],
    tooltip: {
      trigger: "axis",
      valueFormatter: (value) => rateLabel(Number(value), locale),
    },
    xAxis: {
      axisLabel: { color: palette.textSecondary },
      axisLine: { lineStyle: { color: palette.border } },
      axisTick: { show: false },
      boundaryGap: false,
      data: samples.map((sample) => timeLabel(sample.sampledAt, locale)),
      type: "category",
    },
    yAxis: {
      axisLabel: {
        color: palette.textSecondary,
        formatter: (value: number) => formatNumber(value, locale),
      },
      splitLine: { lineStyle: { color: palette.border } },
      type: "value",
    },
  };
}

function diskIOHistoryOption(
  samples: SystemServerMetricsSample[],
  selectedDisk: string,
  palette: ThemeChartPalette,
  locale: string,
  t: ReturnType<typeof useTranslation>["t"],
): EChartsOption {
  const points = samples.map((sample) => diskMetricForSample(sample, selectedDisk));
  return {
    color: [palette.success, palette.warning],
    grid: {
      bottom: 28,
      containLabel: true,
      left: 10,
      right: 16,
      top: 34,
    },
    legend: {
      right: 0,
      textStyle: { color: palette.textSecondary },
      top: 0,
    },
    series: [
      {
        areaStyle: { opacity: 0.16 },
        data: points.map((point) => point.readMbPerSecond),
        name: t("admin.dashboard.serverMetrics.series.read"),
        showSymbol: false,
        smooth: true,
        type: "line",
      },
      {
        areaStyle: { opacity: 0.14 },
        data: points.map((point) => point.writeMbPerSecond),
        name: t("admin.dashboard.serverMetrics.series.write"),
        showSymbol: false,
        smooth: true,
        type: "line",
      },
    ],
    tooltip: {
      trigger: "axis",
      valueFormatter: (value) => mbRateLabel(Number(value), locale),
    },
    xAxis: {
      axisLabel: { color: palette.textSecondary },
      axisLine: { lineStyle: { color: palette.border } },
      axisTick: { show: false },
      boundaryGap: false,
      data: samples.map((sample) => timeLabel(sample.sampledAt, locale)),
      type: "category",
    },
    yAxis: {
      axisLabel: {
        color: palette.textSecondary,
        formatter: (value: number) => formatNumber(value, locale),
      },
      splitLine: { lineStyle: { color: palette.border } },
      type: "value",
    },
  };
}

function trafficHijackLatencyOption(
  results: SystemTrafficProbeResult[],
  palette: ThemeChartPalette,
  locale: string,
  t: ReturnType<typeof useTranslation>["t"],
): EChartsOption {
  const ordered = [...results].reverse();
  return {
    color: [palette.warning, palette.danger],
    grid: {
      bottom: 28,
      containLabel: true,
      left: 10,
      right: 16,
      top: 34,
    },
    legend: {
      right: 0,
      textStyle: { color: palette.textSecondary },
      top: 0,
    },
    series: [
      {
        areaStyle: { opacity: 0.14 },
        data: ordered.map((result) => result.totalDurationMs),
        name: t("admin.dashboard.trafficHijack.series.latency"),
        showSymbol: false,
        smooth: true,
        type: "line",
      },
      {
        data: ordered.map((result) => result.statusCode || 0),
        name: t("admin.dashboard.trafficHijack.series.statusCode"),
        showSymbol: false,
        type: "line",
        yAxisIndex: 1,
      },
    ],
    tooltip: {
      trigger: "axis",
    },
    xAxis: {
      axisLabel: { color: palette.textSecondary },
      axisLine: { lineStyle: { color: palette.border } },
      axisTick: { show: false },
      boundaryGap: false,
      data: ordered.map((result) => timeLabel(result.createdAt, locale)),
      type: "category",
    },
    yAxis: [
      {
        axisLabel: {
          color: palette.textSecondary,
          formatter: (value: number) => durationLabel(value * 1_000_000, locale),
        },
        splitLine: { lineStyle: { color: palette.border } },
        type: "value",
      },
      {
        axisLabel: { color: palette.textSecondary },
        splitLine: { show: false },
        type: "value",
      },
    ],
  };
}

function summarizeAPIs(groups: SystemAPIGroup[]) {
  return groups.reduce(
    (summary, group) => {
      for (const item of group.items) {
        summary.total += 1;
        if (item.access === "permission") {
          summary.permission += 1;
        }
        if (item.access === "public") {
          summary.public += 1;
        }
      }
      return summary;
    },
    { permission: 0, public: 0, total: 0 },
  );
}

function platformLabel(info: SystemServerInfo) {
  return `${info.os.goos}/${info.os.goarch}`;
}

function average(values: number[]) {
  if (values.length === 0) {
    return 0;
  }
  return values.reduce((sum, value) => sum + value, 0) / values.length;
}

function maxDisk(disks: SystemServerInfo["disk"]) {
  return disks.reduce<SystemServerInfo["disk"][number] | undefined>((current, disk) => {
    if (!current || disk.usedPercent > current.usedPercent) {
      return disk;
    }
    return current;
  }, undefined);
}

function diskIONames(samples: SystemServerMetricsSample[]) {
  const names = new Set<string>();
  for (const sample of samples) {
    for (const disk of sample.diskIo ?? []) {
      if (disk.name) {
        names.add(disk.name);
      }
    }
  }
  return [...names].sort((left, right) => left.localeCompare(right));
}

function diskMetricForSample(
  sample: SystemServerMetricsSample,
  selectedDisk: string,
): DiskMetricPoint {
  if (selectedDisk === allDiskValue) {
    return {
      ioLatencyMs: sample.diskIoLatencyMs ?? 0,
      name: allDiskValue,
      readMbPerSecond: sample.diskReadMbPerSecond ?? 0,
      readOpsPerSecond: sample.diskReadOpsPerSecond ?? 0,
      writeMbPerSecond: sample.diskWriteMbPerSecond ?? 0,
      writeOpsPerSecond: sample.diskWriteOpsPerSecond ?? 0,
    };
  }
  return (
    sample.diskIo?.find((disk) => disk.name === selectedDisk) ?? {
      ioLatencyMs: 0,
      name: selectedDisk,
      readMbPerSecond: 0,
      readOpsPerSecond: 0,
      writeMbPerSecond: 0,
      writeOpsPerSecond: 0,
    }
  );
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

function timeLabel(value: string, locale: string) {
  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) {
    return value;
  }
  return new Intl.DateTimeFormat(locale, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(timestamp);
}

function percentLabel(value: number, locale: string) {
  return new Intl.NumberFormat(locale, {
    maximumFractionDigits: 1,
    minimumFractionDigits: 0,
    style: "percent",
  }).format(value / 100);
}

function sizeLabel(valueMb: number, locale: string) {
  return (
    new Intl.NumberFormat(locale, {
      maximumFractionDigits: valueMb >= 1024 ? 1 : 0,
      minimumFractionDigits: 0,
    }).format(valueMb >= 1024 ? valueMb / 1024 : valueMb) + (valueMb >= 1024 ? " GB" : " MB")
  );
}

function rateLabel(value: number, locale: string) {
  return `${new Intl.NumberFormat(locale, {
    maximumFractionDigits: 1,
    minimumFractionDigits: 0,
  }).format(value)} KB/s`;
}

function mbRateLabel(value: number, locale: string) {
  return `${new Intl.NumberFormat(locale, {
    maximumFractionDigits: 2,
    minimumFractionDigits: 0,
  }).format(value)} MB/s`;
}

function opsRateLabel(value: number, locale: string) {
  return `${new Intl.NumberFormat(locale, {
    maximumFractionDigits: 1,
    minimumFractionDigits: 0,
  }).format(value)} ops/s`;
}

function latencyLabel(valueMs: number, locale: string) {
  return `${new Intl.NumberFormat(locale, {
    maximumFractionDigits: 2,
    minimumFractionDigits: 0,
  }).format(valueMs)} ms`;
}

function durationLabel(valueNs: number, locale: string) {
  const milliseconds = valueNs / 1_000_000;
  return `${new Intl.NumberFormat(locale, {
    maximumFractionDigits: 2,
    minimumFractionDigits: 0,
  }).format(milliseconds)} ms`;
}

function fallbackValue(loading: boolean, t: ReturnType<typeof useTranslation>["t"]) {
  return loading ? t("loading.app") : t("common.labels.none");
}

function statusLabel(
  status: string | undefined,
  loading: boolean,
  error: Error | null,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (loading) {
    return t("admin.dashboard.states.loadingShort");
  }
  if (error) {
    return t("admin.dashboard.states.errorShort");
  }
  return status ?? t("common.labels.none");
}

function statusFromError(error: Error | null, t: ReturnType<typeof useTranslation>["t"]) {
  return error ? t("admin.dashboard.states.errorState") : undefined;
}

function errorTitle(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.dashboard.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.dashboard.states.errorTitle");
}

function errorDescription(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.dashboard.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

function versionSourceLabel(source: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (source === "export" || source === "import") {
    return t(`admin.dashboard.versions.source.${source}`);
  }
  return source;
}

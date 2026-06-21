import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import type { EChartsOption } from "echarts";
import { Bell, CheckCircle2, Pencil, Play, RefreshCw, ShieldAlert, Trash2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";

import { DataTable } from "~/components/aoi/patterns/DataTable";
import { EChart } from "~/components/aoi/patterns/EChart";
import { PanelSkeleton, StatGridSkeleton } from "~/components/aoi/patterns/LoadingSkeletons";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { i18n } from "~/i18n/i18n";
import { toBackendLocale } from "~/i18n/locales";
import { API_ENDPOINTS } from "~/lib/api/endpoints";
import { resolveEndpointUrl } from "~/lib/api/client";
import { queryKeys } from "~/lib/api/query-keys";
import { useThemeChartPalette, type ThemeChartPalette } from "~/theme/chart-palette";
import {
  systemApi,
  type SystemTrafficHijackEventListQuery,
  type SystemTrafficProbeResultListQuery,
  type SystemTrafficProbeTargetInput,
  type SystemTrafficProbeTargetUpdateInput,
} from "~/lib/api/system";
import type {
  SystemTrafficHijackEvent,
  SystemTrafficHijackOverview,
  SystemTrafficProbeResult,
  SystemTrafficProbeTarget,
} from "~/lib/api/types";

type TargetFormState = {
  alertChannels: string[];
  allowPrivateNetwork: boolean;
  emailRecipients: string;
  enabled: boolean;
  expectedContentKeyword: string;
  expectedFinalHost: string;
  expectedIPCIDRs: string;
  expectedStatusCodes: string;
  expectedTLSFingerprint: string;
  intervalSeconds: string;
  method: "GET" | "HEAD";
  name: string;
  timeoutSeconds: string;
  url: string;
};

const defaultFormState: TargetFormState = {
  alertChannels: ["event"],
  allowPrivateNetwork: false,
  emailRecipients: "",
  enabled: true,
  expectedContentKeyword: "",
  expectedFinalHost: "",
  expectedIPCIDRs: "",
  expectedStatusCodes: "200-399",
  expectedTLSFingerprint: "",
  intervalSeconds: "30",
  method: "GET",
  name: "",
  timeoutSeconds: "5",
  url: "",
};

const resultLimit = 80;
const eventPageSize = 20;

export default function AdminTrafficHijackRoute() {
  const { i18n: i18nInstance, t } = useTranslation();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<TargetFormState>(defaultFormState);
  const [editingTargetId, setEditingTargetId] = useState<number | string | null>(null);
  const [selectedTargetId, setSelectedTargetId] = useState<number | string>("");
  const [eventState, setEventState] = useState("open");
  const palette = useThemeChartPalette();

  const overviewQuery = useQuery({
    queryFn: ({ signal }) => systemApi.getTrafficHijackOverview({ signal }),
    queryKey: queryKeys.system.trafficHijackOverview(i18nInstance.language),
    refetchInterval: 30_000,
  });
  const targetsQuery = useQuery({
    queryFn: ({ signal }) => systemApi.listTrafficProbeTargets({ signal }),
    queryKey: queryKeys.system.trafficProbeTargets(i18nInstance.language),
    refetchInterval: 30_000,
  });
  const resultQuery: SystemTrafficProbeResultListQuery = {
    limit: resultLimit,
    targetId: selectedTargetId || undefined,
  };
  const resultsQuery = useQuery({
    queryFn: ({ signal }) => systemApi.listTrafficProbeResults(resultQuery, { signal }),
    queryKey: queryKeys.system.trafficProbeResults(i18nInstance.language, resultQuery),
    refetchInterval: 30_000,
  });
  const eventQuery: SystemTrafficHijackEventListQuery = {
    page: 1,
    pageSize: eventPageSize,
    state: eventState || undefined,
    targetId: selectedTargetId || undefined,
  };
  const eventsQuery = useQuery({
    queryFn: ({ signal }) => systemApi.listTrafficHijackEvents(eventQuery, { signal }),
    queryKey: queryKeys.system.trafficHijackEvents(i18nInstance.language, eventQuery),
    refetchInterval: 30_000,
  });

  const refreshTrafficQueries = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.system.root });
  }, [queryClient]);

  const streamState = useTrafficHijackStream(refreshTrafficQueries);

  const createMutation = useMutation({
    mutationFn: (body: SystemTrafficProbeTargetInput) => systemApi.createTrafficProbeTarget(body),
    onSuccess: () => {
      setForm(defaultFormState);
      setEditingTargetId(null);
      refreshTrafficQueries();
    },
  });
  const updateMutation = useMutation({
    mutationFn: ({
      body,
      targetId,
    }: {
      body: SystemTrafficProbeTargetUpdateInput;
      targetId: number | string;
    }) => systemApi.updateTrafficProbeTarget(targetId, body),
    onSuccess: () => {
      setForm(defaultFormState);
      setEditingTargetId(null);
      refreshTrafficQueries();
    },
  });
  const deleteMutation = useMutation({
    mutationFn: (targetId: number | string) => systemApi.deleteTrafficProbeTarget(targetId),
    onSuccess: refreshTrafficQueries,
  });
  const probeMutation = useMutation({
    mutationFn: (targetId: number | string) => systemApi.runTrafficProbe(targetId),
    onSuccess: refreshTrafficQueries,
  });
  const resolveMutation = useMutation({
    mutationFn: (eventId: number | string) => systemApi.resolveTrafficHijackEvent(eventId),
    onSuccess: refreshTrafficQueries,
  });

  const targets = targetsQuery.data ?? overviewQuery.data?.targets ?? [];
  const results = resultsQuery.data?.items ?? overviewQuery.data?.recentResults ?? [];
  const events = eventsQuery.data?.items ?? overviewQuery.data?.recentEvents ?? [];

  const submitTarget = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const payload = formToPayload(form);
    if (editingTargetId) {
      updateMutation.mutate({ body: payload, targetId: editingTargetId });
      return;
    }
    createMutation.mutate(payload);
  };

  const beginEdit = (target: SystemTrafficProbeTarget) => {
    setEditingTargetId(target.id);
    setForm(targetToForm(target));
  };

  const cancelEdit = () => {
    setEditingTargetId(null);
    setForm(defaultFormState);
  };

  const targetColumns = useMemo<ColumnDef<SystemTrafficProbeTarget>[]>(
    () => [
      {
        accessorKey: "name",
        cell: ({ row }) => (
          <div className="aoi-traffic-target-cell">
            <strong>{row.original.name}</strong>
            <span>{row.original.url}</span>
          </div>
        ),
        header: t("admin.trafficHijack.targets.columns.target"),
      },
      {
        accessorKey: "lastStatus",
        cell: ({ row }) => <StatusBadge status={row.original.lastStatus} />,
        header: t("admin.trafficHijack.targets.columns.status"),
      },
      {
        accessorKey: "method",
        header: t("admin.trafficHijack.targets.columns.method"),
      },
      {
        accessorKey: "intervalSeconds",
        cell: ({ getValue }) =>
          t("admin.trafficHijack.units.seconds", { count: Number(getValue()) }),
        header: t("admin.trafficHijack.targets.columns.interval"),
      },
      {
        accessorKey: "lastCheckedAt",
        cell: ({ getValue }) => formatDate(String(getValue() ?? ""), i18nInstance.language, t),
        header: t("admin.trafficHijack.targets.columns.lastChecked"),
      },
      {
        id: "actions",
        cell: ({ row }) => (
          <div className="aoi-traffic-row-actions">
            <Button
              appearance="ghost"
              icon={<Play size={15} />}
              loading={probeMutation.isPending}
              onClick={() => probeMutation.mutate(row.original.id)}
            >
              {t("admin.trafficHijack.actions.probe")}
            </Button>
            <Button
              appearance="ghost"
              icon={<Pencil size={15} />}
              onClick={() => beginEdit(row.original)}
            >
              {t("admin.trafficHijack.actions.edit")}
            </Button>
            <Button
              appearance="ghost"
              icon={<Trash2 size={15} />}
              loading={deleteMutation.isPending}
              onClick={() => deleteMutation.mutate(row.original.id)}
            >
              {t("admin.trafficHijack.actions.delete")}
            </Button>
          </div>
        ),
        header: t("admin.trafficHijack.targets.columns.actions"),
      },
    ],
    [deleteMutation, i18nInstance.language, probeMutation, t],
  );

  const eventColumns = useMemo<ColumnDef<SystemTrafficHijackEvent>[]>(
    () => [
      {
        accessorKey: "targetName",
        header: t("admin.trafficHijack.events.columns.target"),
      },
      {
        accessorKey: "severity",
        cell: ({ getValue }) => <SeverityBadge severity={String(getValue())} />,
        header: t("admin.trafficHijack.events.columns.severity"),
      },
      {
        accessorKey: "reason",
        header: t("admin.trafficHijack.events.columns.reason"),
      },
      {
        accessorKey: "occurrences",
        header: t("admin.trafficHijack.events.columns.occurrences"),
      },
      {
        accessorKey: "lastSeenAt",
        cell: ({ getValue }) => formatDate(String(getValue()), i18nInstance.language, t),
        header: t("admin.trafficHijack.events.columns.lastSeen"),
      },
      {
        id: "actions",
        cell: ({ row }) =>
          row.original.state === "open" ? (
            <Button
              appearance="ghost"
              icon={<CheckCircle2 size={15} />}
              loading={resolveMutation.isPending}
              onClick={() => resolveMutation.mutate(row.original.id)}
            >
              {t("admin.trafficHijack.actions.resolve")}
            </Button>
          ) : (
            <span>{t("admin.trafficHijack.states.resolved")}</span>
          ),
        header: t("admin.trafficHijack.events.columns.actions"),
      },
    ],
    [i18nInstance.language, resolveMutation, t],
  );

  const resultColumns = useMemo<ColumnDef<SystemTrafficProbeResult>[]>(
    () => [
      {
        accessorKey: "createdAt",
        cell: ({ getValue }) => formatDate(String(getValue()), i18nInstance.language, t),
        header: t("admin.trafficHijack.results.columns.time"),
      },
      {
        accessorKey: "targetName",
        header: t("admin.trafficHijack.results.columns.target"),
      },
      {
        accessorKey: "status",
        cell: ({ getValue }) => <StatusBadge status={String(getValue())} />,
        header: t("admin.trafficHijack.results.columns.status"),
      },
      {
        accessorKey: "statusCode",
        cell: ({ getValue }) => String(getValue() || "-"),
        header: t("admin.trafficHijack.results.columns.statusCode"),
      },
      {
        accessorKey: "totalDurationMs",
        cell: ({ getValue }) => durationLabel(Number(getValue()), i18nInstance.language),
        header: t("admin.trafficHijack.results.columns.duration"),
      },
      {
        accessorKey: "stage",
        header: t("admin.trafficHijack.results.columns.stage"),
      },
    ],
    [i18nInstance.language, t],
  );

  const latencyOption = useMemo(
    () => latencyChartOption(results, palette, i18nInstance.language, t),
    [i18nInstance.language, palette, results, t],
  );

  const loading = overviewQuery.isLoading || targetsQuery.isLoading;

  return (
    <section className="aoi-traffic-page" aria-labelledby="admin-traffic-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.trafficHijack.badge")}</Badge>
          <h1 id="admin-traffic-title">{t("admin.trafficHijack.title")}</h1>
          <p>{t("admin.trafficHijack.description")}</p>
        </div>
        <Button
          appearance="secondary"
          icon={<RefreshCw size={17} />}
          loading={overviewQuery.isFetching || targetsQuery.isFetching}
          onClick={refreshTrafficQueries}
        >
          {t("admin.trafficHijack.actions.refresh")}
        </Button>
      </div>

      {loading ? (
        <StatGridSkeleton />
      ) : (
        <TrafficOverviewCards overview={overviewQuery.data} streamState={streamState} />
      )}

      <div className="aoi-traffic-workbench">
        <article className="aoi-admin-panel">
          <header>
            <h2>{t("admin.trafficHijack.form.title")}</h2>
            <p>{t("admin.trafficHijack.form.description")}</p>
          </header>
          <TargetForm
            editing={Boolean(editingTargetId)}
            form={form}
            loading={createMutation.isPending || updateMutation.isPending}
            onCancel={cancelEdit}
            onChange={setForm}
            onSubmit={submitTarget}
          />
        </article>

        <article className="aoi-admin-panel aoi-admin-panel--span-2">
          <header className="aoi-admin-panel-header-row">
            <div>
              <h2>{t("admin.trafficHijack.targets.title")}</h2>
              <p>{t("admin.trafficHijack.targets.description")}</p>
            </div>
            <select
              aria-label={t("admin.trafficHijack.filters.target")}
              onChange={(event) => setSelectedTargetId(event.target.value)}
              value={selectedTargetId}
            >
              <option value="">{t("admin.trafficHijack.filters.allTargets")}</option>
              {targets.map((target) => (
                <option key={target.id} value={String(target.id)}>
                  {target.name}
                </option>
              ))}
            </select>
          </header>
          {targetsQuery.error ? (
            <StateBlock
              intent="danger"
              title={t("admin.trafficHijack.states.loadFailed")}
              description={targetsQuery.error.message}
            />
          ) : targetsQuery.isLoading ? (
            <PanelSkeleton />
          ) : (
            <DataTable
              columns={targetColumns}
              data={targets}
              emptyLabel={t("admin.trafficHijack.targets.empty")}
            />
          )}
        </article>

        <article className="aoi-admin-panel aoi-admin-panel--span-2">
          <header>
            <h2>{t("admin.trafficHijack.results.chartTitle")}</h2>
            <p>{t("admin.trafficHijack.results.chartDescription")}</p>
          </header>
          {results.length > 0 ? (
            <EChart
              ariaLabel={t("admin.trafficHijack.results.chartAria")}
              className="aoi-traffic-latency-chart"
              option={latencyOption}
            />
          ) : (
            <StateBlock
              title={t("admin.trafficHijack.results.emptyTitle")}
              description={t("admin.trafficHijack.results.emptyDescription")}
            />
          )}
        </article>

        <article className="aoi-admin-panel aoi-admin-panel--span-2">
          <header>
            <h2>{t("admin.trafficHijack.events.title")}</h2>
            <p>{t("admin.trafficHijack.events.description")}</p>
          </header>
          <div className="aoi-traffic-toolbar">
            <select
              aria-label={t("admin.trafficHijack.filters.eventState")}
              onChange={(event) => setEventState(event.target.value)}
              value={eventState}
            >
              <option value="open">{t("admin.trafficHijack.states.open")}</option>
              <option value="resolved">{t("admin.trafficHijack.states.resolved")}</option>
              <option value="">{t("admin.trafficHijack.states.all")}</option>
            </select>
          </div>
          {eventsQuery.error ? (
            <StateBlock
              intent="danger"
              title={t("admin.trafficHijack.states.loadFailed")}
              description={eventsQuery.error.message}
            />
          ) : eventsQuery.isLoading ? (
            <PanelSkeleton />
          ) : (
            <DataTable
              columns={eventColumns}
              data={events}
              emptyLabel={t("admin.trafficHijack.events.empty")}
            />
          )}
        </article>

        <article className="aoi-admin-panel aoi-admin-panel--span-2">
          <header>
            <h2>{t("admin.trafficHijack.results.title")}</h2>
            <p>{t("admin.trafficHijack.results.description")}</p>
          </header>
          {resultsQuery.error ? (
            <StateBlock
              intent="danger"
              title={t("admin.trafficHijack.states.loadFailed")}
              description={resultsQuery.error.message}
            />
          ) : resultsQuery.isLoading ? (
            <PanelSkeleton />
          ) : (
            <DataTable
              columns={resultColumns}
              data={results}
              emptyLabel={t("admin.trafficHijack.results.empty")}
            />
          )}
        </article>
      </div>
    </section>
  );
}

function TargetForm({
  editing,
  form,
  loading,
  onCancel,
  onChange,
  onSubmit,
}: {
  editing: boolean;
  form: TargetFormState;
  loading: boolean;
  onCancel: () => void;
  onChange: (next: TargetFormState) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  const { t } = useTranslation();
  const update = <K extends keyof TargetFormState>(key: K, value: TargetFormState[K]) =>
    onChange({ ...form, [key]: value });
  const toggleChannel = (channel: string) => {
    const channels = new Set(form.alertChannels);
    if (channels.has(channel)) {
      channels.delete(channel);
    } else {
      channels.add(channel);
    }
    if (channels.size === 0) {
      channels.add("event");
    }
    update("alertChannels", Array.from(channels));
  };

  return (
    <form className="aoi-traffic-form" onSubmit={onSubmit}>
      <label className="aoi-form-field">
        <span>{t("admin.trafficHijack.form.fields.name")}</span>
        <input
          onChange={(event) => update("name", event.target.value)}
          required
          value={form.name}
        />
      </label>
      <label className="aoi-form-field">
        <span>{t("admin.trafficHijack.form.fields.url")}</span>
        <input
          inputMode="url"
          onChange={(event) => update("url", event.target.value)}
          placeholder="https://example.com"
          required
          value={form.url}
        />
      </label>
      <div className="aoi-traffic-form-grid">
        <label className="aoi-form-field">
          <span>{t("admin.trafficHijack.form.fields.method")}</span>
          <select
            onChange={(event) => update("method", event.target.value as "GET" | "HEAD")}
            value={form.method}
          >
            <option value="GET">GET</option>
            <option value="HEAD">HEAD</option>
          </select>
        </label>
        <label className="aoi-form-field">
          <span>{t("admin.trafficHijack.form.fields.interval")}</span>
          <input
            min={10}
            onChange={(event) => update("intervalSeconds", event.target.value)}
            type="number"
            value={form.intervalSeconds}
          />
        </label>
        <label className="aoi-form-field">
          <span>{t("admin.trafficHijack.form.fields.timeout")}</span>
          <input
            min={1}
            onChange={(event) => update("timeoutSeconds", event.target.value)}
            type="number"
            value={form.timeoutSeconds}
          />
        </label>
      </div>
      <div className="aoi-traffic-form-grid">
        <label className="aoi-form-field">
          <span>{t("admin.trafficHijack.form.fields.statusCodes")}</span>
          <input
            onChange={(event) => update("expectedStatusCodes", event.target.value)}
            value={form.expectedStatusCodes}
          />
        </label>
        <label className="aoi-form-field">
          <span>{t("admin.trafficHijack.form.fields.finalHost")}</span>
          <input
            onChange={(event) => update("expectedFinalHost", event.target.value)}
            value={form.expectedFinalHost}
          />
        </label>
      </div>
      <label className="aoi-form-field">
        <span>{t("admin.trafficHijack.form.fields.keyword")}</span>
        <input
          onChange={(event) => update("expectedContentKeyword", event.target.value)}
          value={form.expectedContentKeyword}
        />
      </label>
      <label className="aoi-form-field">
        <span>{t("admin.trafficHijack.form.fields.ipCidrs")}</span>
        <input
          onChange={(event) => update("expectedIPCIDRs", event.target.value)}
          value={form.expectedIPCIDRs}
        />
      </label>
      <label className="aoi-form-field">
        <span>{t("admin.trafficHijack.form.fields.tlsFingerprint")}</span>
        <input
          onChange={(event) => update("expectedTLSFingerprint", event.target.value)}
          value={form.expectedTLSFingerprint}
        />
      </label>
      <label className="aoi-traffic-check">
        <input
          checked={form.allowPrivateNetwork}
          onChange={(event) => update("allowPrivateNetwork", event.target.checked)}
          type="checkbox"
        />
        <span>{t("admin.trafficHijack.form.fields.allowPrivateNetwork")}</span>
      </label>
      <label className="aoi-traffic-check">
        <input
          checked={form.enabled}
          onChange={(event) => update("enabled", event.target.checked)}
          type="checkbox"
        />
        <span>{t("admin.trafficHijack.form.fields.enabled")}</span>
      </label>
      <fieldset className="aoi-traffic-alert-fieldset">
        <legend>{t("admin.trafficHijack.form.fields.alertChannels")}</legend>
        {["event", "debug", "email"].map((channel) => (
          <label className="aoi-traffic-check" key={channel}>
            <input
              checked={form.alertChannels.includes(channel)}
              onChange={() => toggleChannel(channel)}
              type="checkbox"
            />
            <span>{t(`admin.trafficHijack.channels.${channel}`)}</span>
          </label>
        ))}
      </fieldset>
      <label className="aoi-form-field">
        <span>{t("admin.trafficHijack.form.fields.emailRecipients")}</span>
        <input
          onChange={(event) => update("emailRecipients", event.target.value)}
          value={form.emailRecipients}
        />
      </label>
      <div className="aoi-traffic-form-actions">
        <Button icon={<ShieldAlert size={16} />} loading={loading} type="submit">
          {editing
            ? t("admin.trafficHijack.actions.save")
            : t("admin.trafficHijack.actions.create")}
        </Button>
        {editing ? (
          <Button appearance="secondary" onClick={onCancel}>
            {t("admin.trafficHijack.actions.cancel")}
          </Button>
        ) : null}
      </div>
    </form>
  );
}

function TrafficOverviewCards({
  overview,
  streamState,
}: {
  overview?: SystemTrafficHijackOverview;
  streamState: string;
}) {
  const { i18n: i18nInstance, t } = useTranslation();
  const total = overview?.totalTargets ?? 0;
  const abnormal = (overview?.warningTargets ?? 0) + (overview?.criticalTargets ?? 0);
  const latest = overview?.recentResults?.[0];

  return (
    <div className="aoi-admin-stat-grid" aria-label={t("admin.trafficHijack.summary.label")}>
      <article className="aoi-admin-stat-card" data-state={abnormal > 0 ? "warning" : "success"}>
        <span aria-hidden="true">
          <ShieldAlert size={19} />
        </span>
        <div>
          <p>{t("admin.trafficHijack.summary.targets")}</p>
          <strong>
            {t("admin.trafficHijack.summary.targetsValue", {
              enabled: overview?.enabledTargets ?? 0,
              total,
            })}
          </strong>
        </div>
      </article>
      <article className="aoi-admin-stat-card" data-state={abnormal > 0 ? "danger" : "success"}>
        <span aria-hidden="true">
          <Bell size={19} />
        </span>
        <div>
          <p>{t("admin.trafficHijack.summary.abnormal")}</p>
          <strong>{formatNumber(abnormal, i18nInstance.language)}</strong>
        </div>
      </article>
      <article className="aoi-admin-stat-card" data-state="info">
        <span aria-hidden="true">
          <RefreshCw size={19} />
        </span>
        <div>
          <p>{t("admin.trafficHijack.summary.stream")}</p>
          <strong>{t(`admin.trafficHijack.stream.${streamState}`)}</strong>
        </div>
      </article>
      <article className="aoi-admin-stat-card">
        <span aria-hidden="true">
          <Play size={19} />
        </span>
        <div>
          <p>{t("admin.trafficHijack.summary.latest")}</p>
          <strong>
            {latest
              ? durationLabel(latest.totalDurationMs, i18nInstance.language)
              : t("common.labels.none")}
          </strong>
        </div>
      </article>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  return (
    <Badge className="aoi-traffic-status-badge" data-status={status || "pending"}>
      {statusLabel(status)}
    </Badge>
  );
}

function SeverityBadge({ severity }: { severity: string }) {
  return (
    <Badge className="aoi-traffic-severity-badge" data-severity={severity || "ok"}>
      {severityLabel(severity)}
    </Badge>
  );
}

function formToPayload(form: TargetFormState): SystemTrafficProbeTargetInput {
  return {
    alertChannels: form.alertChannels,
    allowPrivateNetwork: form.allowPrivateNetwork,
    emailRecipients: splitList(form.emailRecipients),
    enabled: form.enabled,
    expectedContentKeyword: form.expectedContentKeyword.trim(),
    expectedFinalHost: form.expectedFinalHost.trim(),
    expectedIpCidrs: splitList(form.expectedIPCIDRs),
    expectedStatusCodes: form.expectedStatusCodes.trim(),
    expectedTlsFingerprint: form.expectedTLSFingerprint.trim(),
    intervalSeconds: numberOrUndefined(form.intervalSeconds),
    method: form.method,
    name: form.name.trim(),
    timeoutSeconds: numberOrUndefined(form.timeoutSeconds),
    url: form.url.trim(),
  };
}

function targetToForm(target: SystemTrafficProbeTarget): TargetFormState {
  return {
    alertChannels: splitList(target.alertChannels || "event"),
    allowPrivateNetwork: target.allowPrivateNetwork,
    emailRecipients: target.emailRecipients,
    enabled: target.enabled,
    expectedContentKeyword: target.expectedContentKeyword,
    expectedFinalHost: target.expectedFinalHost,
    expectedIPCIDRs: target.expectedIpCidrs,
    expectedStatusCodes: target.expectedStatusCodes || "200-399",
    expectedTLSFingerprint: target.expectedTlsFingerprint,
    intervalSeconds: String(target.intervalSeconds || 30),
    method: target.method === "HEAD" ? "HEAD" : "GET",
    name: target.name,
    timeoutSeconds: String(target.timeoutSeconds || 5),
    url: target.url,
  };
}

function useTrafficHijackStream(onEvent: () => void) {
  const [state, setState] = useState("connecting");

  useEffect(() => {
    const controller = new AbortController();
    let reconnectTimer: number | undefined;
    let active = true;

    async function connect() {
      try {
        setState("connecting");
        const headers = new Headers();
        headers.set("Accept", "text/event-stream");
        headers.set("X-Locale", toBackendLocale(i18n.language));
        const response = await fetch(
          resolveEndpointUrl(API_ENDPOINTS.system.trafficHijack.stream),
          {
            credentials: "include",
            headers,
            signal: controller.signal,
          },
        );
        if (!response.ok || !response.body) {
          throw new Error("stream unavailable");
        }
        setState("connected");
        await readSSEStream(response.body, () => {
          if (active) {
            onEvent();
          }
        });
      } catch {
        if (!active || controller.signal.aborted) {
          return;
        }
        setState("fallback");
        reconnectTimer = window.setTimeout(connect, 30_000);
      }
    }

    void connect();
    return () => {
      active = false;
      controller.abort();
      if (reconnectTimer !== undefined) {
        window.clearTimeout(reconnectTimer);
      }
    };
  }, [onEvent]);

  return state;
}

async function readSSEStream(stream: ReadableStream<Uint8Array>, onMessage: () => void) {
  const reader = stream.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  for (;;) {
    const { done, value } = await reader.read();
    if (done) {
      return;
    }
    buffer += decoder.decode(value, { stream: true });
    let boundary = buffer.indexOf("\n\n");
    while (boundary >= 0) {
      const chunk = buffer.slice(0, boundary);
      buffer = buffer.slice(boundary + 2);
      if (chunk.includes("data:")) {
        onMessage();
      }
      boundary = buffer.indexOf("\n\n");
    }
  }
}

function latencyChartOption(
  results: SystemTrafficProbeResult[],
  palette: ThemeChartPalette,
  locale: string,
  t: ReturnType<typeof useTranslation>["t"],
): EChartsOption {
  const ordered = [...results].reverse();
  return {
    color: [palette.primary, palette.danger],
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
        name: t("admin.trafficHijack.results.series.latency"),
        showSymbol: false,
        smooth: true,
        type: "line",
      },
      {
        data: ordered.map((result) => result.statusCode || 0),
        name: t("admin.trafficHijack.results.series.statusCode"),
        showSymbol: false,
        smooth: false,
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
          formatter: (value: number) => durationLabel(value, locale),
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

function splitList(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function numberOrUndefined(value: string) {
  const next = Number(value);
  return Number.isFinite(next) ? next : undefined;
}

function statusLabel(status: string) {
  return i18n.t(`admin.trafficHijack.status.${status || "pending"}`);
}

function severityLabel(severity: string) {
  return i18n.t(`admin.trafficHijack.severity.${severity || "ok"}`);
}

function formatNumber(value: number, locale: string) {
  return new Intl.NumberFormat(locale).format(value);
}

function formatDate(value: string, locale: string, t: ReturnType<typeof useTranslation>["t"]) {
  const timestamp = Date.parse(value);
  if (!value || Number.isNaN(timestamp)) {
    return t("common.labels.none");
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

function durationLabel(value: number, locale: string) {
  return `${new Intl.NumberFormat(locale, {
    maximumFractionDigits: 0,
  }).format(value)} ms`;
}

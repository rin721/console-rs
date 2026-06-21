import { useQuery } from "@tanstack/react-query";
import { CircleCheck, DatabaseZap, HeartPulse, RefreshCw, TimerReset } from "lucide-react";
import { type ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { ApiError } from "~/lib/api/client";
import { queryKeys } from "~/lib/api/query-keys";
import { systemApi } from "~/lib/api/system";
import type { ReadyStatus } from "~/lib/api/types";

type ProbeState = "danger" | "info" | "success" | "unknown" | "warning";

export default function AdminProbesRoute() {
  const { i18n, t } = useTranslation();

  const healthQuery = useQuery({
    queryFn: ({ signal }) => systemApi.getHealth({ signal }),
    queryKey: queryKeys.system.health,
  });
  const readyQuery = useQuery({
    queryFn: ({ signal }) => systemApi.getReady({ signal }),
    queryKey: queryKeys.system.ready,
  });

  const readyData = readyQuery.data ?? readyStatusFromError(readyQuery.error);
  const readyError = readyData ? null : readyQuery.error;
  const loading = healthQuery.isFetching || readyQuery.isFetching;
  const healthState = probeState(
    healthQuery.data?.status,
    healthQuery.isLoading,
    healthQuery.error,
  );
  const readyState = probeState(readyData?.status, readyQuery.isLoading, readyError);
  const readyChecks = Object.entries(readyData?.checks ?? {});

  const refresh = () => {
    void healthQuery.refetch();
    void readyQuery.refetch();
  };

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-probes-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.probes.badge")}</Badge>
          <h1 id="admin-probes-title">{t("admin.probes.title")}</h1>
          <p>{t("admin.probes.description")}</p>
        </div>
        <Button
          appearance="secondary"
          icon={<RefreshCw size={17} />}
          loading={loading}
          onClick={refresh}
        >
          {t("admin.probes.actions.refresh")}
        </Button>
      </div>

      <div className="aoi-admin-stat-grid" aria-label={t("admin.probes.summaryLabel")}>
        <ProbeMetricCard
          icon={<HeartPulse size={19} />}
          label={t("admin.probes.metrics.health")}
          state={healthState}
          value={probeStatusLabel(
            healthQuery.data?.status,
            healthQuery.isLoading,
            healthQuery.error,
            t,
          )}
        />
        <ProbeMetricCard
          icon={<DatabaseZap size={19} />}
          label={t("admin.probes.metrics.readiness")}
          state={readyState}
          value={probeStatusLabel(readyData?.status, readyQuery.isLoading, readyError, t)}
        />
        <ProbeMetricCard
          icon={<CircleCheck size={19} />}
          label={t("admin.probes.metrics.checks")}
          value={
            readyData
              ? formatNumber(readyChecks.length, i18n.language)
              : fallbackValue(readyQuery.isLoading, t)
          }
        />
        <ProbeMetricCard
          icon={<TimerReset size={19} />}
          label={t("admin.probes.metrics.lastChecked")}
          value={latestCheckedLabel(
            [healthQuery.dataUpdatedAt, readyQuery.dataUpdatedAt],
            i18n.language,
            t,
          )}
        />
      </div>

      <div className="aoi-probe-grid">
        <section className="aoi-admin-panel">
          <ProbePanelHeader
            description={t("admin.probes.health.description")}
            endpoint="/health"
            icon={<HeartPulse size={20} />}
            state={healthState}
            status={probeStatusLabel(
              healthQuery.data?.status,
              healthQuery.isLoading,
              healthQuery.error,
              t,
            )}
            title={t("admin.probes.health.title")}
          />
          {healthQuery.error ? (
            <StateBlock
              intent="danger"
              title={t("admin.probes.states.healthErrorTitle")}
              description={healthQuery.error.message || t("errors.api.requestFailed")}
            />
          ) : healthQuery.isLoading ? (
            <StateBlock
              title={t("admin.probes.states.loadingTitle")}
              description={t("admin.probes.states.loadingDescription")}
            />
          ) : healthQuery.data ? (
            <KeyValueList
              items={[
                [t("admin.probes.labels.endpoint"), "/health"],
                [
                  t("admin.probes.labels.status"),
                  probeStatusLabel(healthQuery.data.status, false, null, t),
                ],
                [
                  t("admin.probes.labels.lastChecked"),
                  formatDateTime(healthQuery.dataUpdatedAt, i18n.language, t),
                ],
              ]}
            />
          ) : (
            <StateBlock
              title={t("admin.probes.states.emptyHealthTitle")}
              description={t("admin.probes.states.emptyHealthDescription")}
            />
          )}
        </section>

        <section className="aoi-admin-panel">
          <ProbePanelHeader
            description={t("admin.probes.ready.description")}
            endpoint="/ready"
            icon={<DatabaseZap size={20} />}
            state={readyState}
            status={probeStatusLabel(readyData?.status, readyQuery.isLoading, readyError, t)}
            title={t("admin.probes.ready.title")}
          />
          {readyError ? (
            <StateBlock
              intent="danger"
              title={t("admin.probes.states.readyErrorTitle")}
              description={readyError.message || t("errors.api.requestFailed")}
            />
          ) : readyQuery.isLoading ? (
            <StateBlock
              title={t("admin.probes.states.loadingTitle")}
              description={t("admin.probes.states.loadingDescription")}
            />
          ) : readyData ? (
            <>
              <KeyValueList
                items={[
                  [t("admin.probes.labels.endpoint"), "/ready"],
                  [
                    t("admin.probes.labels.status"),
                    probeStatusLabel(readyData.status, false, null, t),
                  ],
                  [
                    t("admin.probes.labels.lastChecked"),
                    formatDateTime(readyQuery.dataUpdatedAt, i18n.language, t),
                  ],
                ]}
              />
              <div className="aoi-probe-checks" aria-label={t("admin.probes.checks.label")}>
                <h2>{t("admin.probes.checks.title")}</h2>
                {readyChecks.length > 0 ? (
                  <dl>
                    {readyChecks.map(([name, value]) => {
                      const state = probeState(value, false, null);
                      return (
                        <div key={name} className="aoi-probe-check">
                          <dt>{name}</dt>
                          <dd>
                            <span className="aoi-probe-status" data-state={state}>
                              {checkValueLabel(value, t)}
                            </span>
                          </dd>
                        </div>
                      );
                    })}
                  </dl>
                ) : (
                  <StateBlock
                    title={t("admin.probes.states.emptyChecksTitle")}
                    description={t("admin.probes.states.emptyChecksDescription")}
                  />
                )}
              </div>
            </>
          ) : (
            <StateBlock
              title={t("admin.probes.states.emptyReadyTitle")}
              description={t("admin.probes.states.emptyReadyDescription")}
            />
          )}
        </section>
      </div>
    </section>
  );
}

type ProbeMetricCardProps = {
  icon: ReactNode;
  label: string;
  state?: ProbeState;
  value: string;
};

function ProbeMetricCard({ icon, label, state, value }: ProbeMetricCardProps) {
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

type ProbePanelHeaderProps = {
  description: string;
  endpoint: string;
  icon: ReactNode;
  state: ProbeState;
  status: string;
  title: string;
};

function ProbePanelHeader({
  description,
  endpoint,
  icon,
  state,
  status,
  title,
}: ProbePanelHeaderProps) {
  return (
    <header className="aoi-probe-header">
      <div className="aoi-probe-heading">
        <span aria-hidden="true">{icon}</span>
        <div>
          <h2>{title}</h2>
          <p>{description}</p>
        </div>
      </div>
      <div className="aoi-probe-header__meta">
        <code>{endpoint}</code>
        <span className="aoi-probe-status" data-state={state}>
          {status}
        </span>
      </div>
    </header>
  );
}

function KeyValueList({ items }: { items: Array<[string, string]> }) {
  return (
    <dl className="aoi-admin-key-values">
      {items.map(([label, value]) => (
        <div key={label}>
          <dt>{label}</dt>
          <dd>{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function probeState(status: string | undefined, loading: boolean, error: Error | null): ProbeState {
  if (error) {
    return "danger";
  }
  if (loading) {
    return "info";
  }
  if (status === "ok" || status === "ready") {
    return "success";
  }
  if (status === "not_ready" || status === "missing") {
    return "warning";
  }
  return status ? "unknown" : "info";
}

function probeStatusLabel(
  status: string | undefined,
  loading: boolean,
  error: Error | null,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (loading) {
    return t("admin.probes.status.loading");
  }
  if (error) {
    return t("admin.probes.status.error");
  }
  if (status === "ok") {
    return t("admin.probes.status.ok");
  }
  if (status === "ready") {
    return t("admin.probes.status.ready");
  }
  if (status === "not_ready") {
    return t("admin.probes.status.notReady");
  }
  return status || t("common.labels.none");
}

function checkValueLabel(value: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (value === "ok") {
    return t("admin.probes.status.ok");
  }
  if (value === "missing") {
    return t("admin.probes.status.missing");
  }
  return value || t("admin.probes.status.unknown");
}

function readyStatusFromError(error: Error | null): ReadyStatus | undefined {
  if (!(error instanceof ApiError)) {
    return undefined;
  }

  const payload = error.payload;
  if (!payload || typeof payload !== "object" || !("data" in payload)) {
    return undefined;
  }

  const data = (payload as { data?: unknown }).data;
  if (!data || typeof data !== "object" || !("status" in data)) {
    return undefined;
  }

  const status = (data as { status?: unknown }).status;
  if (typeof status !== "string") {
    return undefined;
  }

  const checks = (data as { checks?: unknown }).checks;
  return {
    checks: isStringRecord(checks) ? checks : undefined,
    status,
  };
}

function isStringRecord(value: unknown): value is Record<string, string> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return false;
  }
  return Object.values(value).every((entry) => typeof entry === "string");
}

function fallbackValue(loading: boolean, t: ReturnType<typeof useTranslation>["t"]) {
  return loading ? t("loading.app") : t("common.labels.none");
}

function latestCheckedLabel(
  values: number[],
  locale: string,
  t: ReturnType<typeof useTranslation>["t"],
) {
  const latest = Math.max(...values.filter((value) => value > 0));
  return Number.isFinite(latest) ? formatDateTime(latest, locale, t) : t("common.labels.none");
}

function formatDateTime(value: number, locale: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (!value) {
    return t("common.labels.none");
  }
  return new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function formatNumber(value: number, locale: string) {
  return new Intl.NumberFormat(locale).format(value);
}

import { useEffect, useRef, useState } from "react";
import type { ECharts, EChartsOption } from "echarts";

import { cn } from "~/lib/cn";

type EChartsNamespace = typeof import("echarts");

type EChartProps = {
  ariaLabel: string;
  className?: string;
  loading?: boolean;
  loadingLabel?: string;
  option: EChartsOption;
};

export function EChart({
  ariaLabel,
  className,
  loading = false,
  loadingLabel,
  option,
}: EChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<ECharts | null>(null);
  const reducedMotion = usePrefersReducedMotion();
  const latestConfigRef = useRef({ loading, loadingLabel, option, reducedMotion });
  const [echartsModule, setEchartsModule] = useState<EChartsNamespace | null>(null);
  latestConfigRef.current = { loading, loadingLabel, option, reducedMotion };

  useEffect(() => {
    let active = true;
    void import("echarts").then((module) => {
      if (active) {
        setEchartsModule(module);
      }
    });
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (!echartsModule || !containerRef.current) {
      return undefined;
    }

    const chart = echartsModule.init(containerRef.current);
    chartRef.current = chart;
    const latestConfig = latestConfigRef.current;
    applyChartOption(
      chart,
      latestConfig.option,
      latestConfig.reducedMotion,
      latestConfig.loading,
      latestConfig.loadingLabel,
    );

    let resizeObserver: ResizeObserver | undefined;
    const resize = () => chart.resize();
    if (typeof ResizeObserver !== "undefined") {
      resizeObserver = new ResizeObserver(resize);
      resizeObserver.observe(containerRef.current);
    } else {
      window.addEventListener("resize", resize);
    }

    return () => {
      resizeObserver?.disconnect();
      window.removeEventListener("resize", resize);
      chart.dispose();
      chartRef.current = null;
    };
  }, [echartsModule]);

  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) {
      return;
    }

    applyChartOption(chart, option, reducedMotion, loading, loadingLabel);
  }, [loading, loadingLabel, option, reducedMotion]);

  return (
    <div
      aria-busy={loading ? "true" : "false"}
      aria-label={ariaLabel}
      className={cn("aoi-echart", className)}
      ref={containerRef}
      role="img"
    />
  );
}

function applyChartOption(
  chart: ECharts,
  option: EChartsOption,
  reducedMotion: boolean,
  loading: boolean,
  loadingLabel?: string,
) {
  chart.setOption(
    {
      ...option,
      animation: option.animation === false ? false : !reducedMotion,
    },
    true,
  );
  chart.resize();

  if (loading) {
    chart.showLoading("default", {
      text: loadingLabel,
    });
  } else {
    chart.hideLoading();
  }
}

function usePrefersReducedMotion() {
  const [reducedMotion, setReducedMotion] = useState(false);

  useEffect(() => {
    const query = window.matchMedia("(prefers-reduced-motion: reduce)");
    setReducedMotion(query.matches);
    const update = (event: MediaQueryListEvent) => setReducedMotion(event.matches);
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);

  return reducedMotion;
}

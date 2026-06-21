use std::sync::Arc;
use std::time::Duration;

use serde::Serialize;
use tokio::task::JoinHandle;
use tracing::{info, warn};

use crate::app::AppResult;
use crate::config::SchedulerConfig;
use crate::service::system::SystemService;

#[derive(Clone, Debug, Serialize)]
pub struct SchedulerRunReport {
    pub traffic_probe: TrafficProbeSchedulerReport,
    pub operation_record_retention: OperationRecordRetentionSchedulerReport,
}

#[derive(Clone, Debug, Serialize)]
pub struct TrafficProbeSchedulerReport {
    pub enabled: bool,
    pub scanned_targets: usize,
    pub recorded_results: usize,
    pub failed_targets: usize,
}

#[derive(Clone, Debug, Serialize)]
pub struct OperationRecordRetentionSchedulerReport {
    pub enabled: bool,
    pub retention_days: i64,
    pub deleted: i64,
}

pub async fn run_once(system: Arc<SystemService>) -> AppResult<SchedulerRunReport> {
    let traffic_probe = system.run_all_traffic_probes().await?;
    let retention = system.prune_operation_records().await?;
    Ok(SchedulerRunReport {
        traffic_probe: TrafficProbeSchedulerReport {
            enabled: true,
            scanned_targets: traffic_probe.scanned_targets,
            recorded_results: traffic_probe.recorded_results,
            failed_targets: traffic_probe.failed_targets,
        },
        operation_record_retention: OperationRecordRetentionSchedulerReport {
            enabled: true,
            retention_days: retention.retention_days,
            deleted: retention.deleted,
        },
    })
}

pub fn spawn(config: SchedulerConfig, system: Arc<SystemService>) -> Option<JoinHandle<()>> {
    if !config.enabled {
        return None;
    }

    Some(tokio::spawn(async move {
        let interval_duration = Duration::from_secs(config.traffic_probe_interval_seconds.max(1));
        if config.run_on_start {
            run_tick(system.clone()).await;
        } else {
            tokio::time::sleep(interval_duration).await;
        }

        let mut interval = tokio::time::interval(interval_duration);
        loop {
            interval.tick().await;
            run_tick(system.clone()).await;
        }
    }))
}

async fn run_tick(system: Arc<SystemService>) {
    match run_once(system).await {
        Ok(report) => info!(
            scanned_targets = report.traffic_probe.scanned_targets,
            recorded_results = report.traffic_probe.recorded_results,
            failed_targets = report.traffic_probe.failed_targets,
            operation_records_deleted = report.operation_record_retention.deleted,
            "scheduler 已完成一轮系统任务"
        ),
        Err(err) => warn!(error = %err, "scheduler 执行流量探针采集失败"),
    }
}

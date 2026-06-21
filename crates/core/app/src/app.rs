use std::sync::Arc;

use axum::Router;
use axum::middleware;
use tokio::net::TcpListener;
use tower_http::cors::{Any, CorsLayer};
use tower_http::trace::TraceLayer;
use tracing::{info, warn};

use crate::config::{NotificationDriver, Settings, StorageDriver};
use crate::domain::system::{ApiCatalogEntry, SystemMenuEntry};
use crate::infrastructure::database::{self, DatabaseConnection};
use crate::infrastructure::media_storage::{LocalMediaStorage, S3MediaStorage};
use crate::infrastructure::notification::{
    FileNotificationSender, LogNotificationSender, QueueNotificationSender, SmtpNotificationSender,
};
use crate::infrastructure::system_metrics::SysinfoMetricsCollector;
use crate::infrastructure::traffic_probe::HttpTrafficProbeRunner;
use crate::scheduler;
use crate::service::iam::IamService;
use crate::service::notification::{NotificationSender, NotificationService};
use crate::service::setup::SetupService;
use crate::service::system::{MediaStorage, SystemService};
use crate::transport::http::route_registry::RouteContract;
use crate::transport::http::{csrf, operation_record, route_registry, router};

pub use crate::error::{AppError, AppResult};

pub struct App {
    settings: Settings,
    state: Arc<AppState>,
}

pub struct AppState {
    pub settings: Settings,
    pub database: DatabaseConnection,
    pub setup: Arc<SetupService>,
    pub iam: Arc<IamService>,
    pub notification: Arc<NotificationService>,
    pub system: Arc<SystemService>,
}

impl App {
    pub async fn boot(settings: Settings) -> anyhow::Result<Self> {
        let database = database::connect(&settings).await?;
        let repos = database::repositories(&database);
        let setup = Arc::new(SetupService::new(
            settings.clone(),
            repos.setup.clone(),
            repos.iam.clone(),
        ));
        let iam = Arc::new(IamService::new(settings.clone(), repos.iam.clone()));
        let notification_sender: Arc<dyn NotificationSender> = match settings.notification.driver {
            NotificationDriver::File => Arc::new(FileNotificationSender::new(
                settings.notification.local_dir.clone(),
            )),
            NotificationDriver::Log => Arc::new(LogNotificationSender),
            NotificationDriver::Smtp => Arc::new(SmtpNotificationSender::new(
                settings.notification.smtp.clone(),
            )?),
            NotificationDriver::Queue => Arc::new(QueueNotificationSender::new(
                settings.notification.queue.clone(),
            )),
        };
        let notification = Arc::new(NotificationService::new(
            settings.clone(),
            repos.notification.clone(),
            notification_sender,
        ));
        let traffic_probe_runner = Arc::new(HttpTrafficProbeRunner::new());
        let media_storage = media_storage_from_settings(&settings)?;
        let metrics_collector = Arc::new(SysinfoMetricsCollector);
        let system = Arc::new(SystemService::new(
            settings.clone(),
            repos.system.clone(),
            traffic_probe_runner,
            media_storage,
            metrics_collector,
        ));
        let contracts = route_registry::contracts(&settings);
        let api_catalog_entries = api_catalog_entries_from_contracts(&contracts);
        let menu_entries = menu_entries_from_contracts(&contracts);
        system
            .sync_api_catalog(&api_catalog_entries, &menu_entries)
            .await?;

        Ok(Self {
            state: Arc::new(AppState {
                settings: settings.clone(),
                database,
                setup,
                iam,
                notification,
                system,
            }),
            settings,
        })
    }

    pub fn router(&self) -> Router {
        router::build(self.state.clone())
            .layer(middleware::from_fn_with_state(
                self.state.clone(),
                csrf::require_csrf,
            ))
            .layer(middleware::from_fn_with_state(
                self.state.clone(),
                operation_record::record_operation,
            ))
            .layer(TraceLayer::new_for_http())
            .layer(CorsLayer::new().allow_origin(Any))
    }

    pub async fn serve(self) -> anyhow::Result<()> {
        let addr = self.settings.socket_addr();
        let listener = TcpListener::bind(addr).await?;
        info!(%addr, "Aoi[葵] HTTP 服务已启动");
        let scheduler_handle =
            scheduler::spawn(self.settings.scheduler.clone(), self.state.system.clone());
        let serve_result = axum::serve(listener, self.router())
            .with_graceful_shutdown(shutdown_signal())
            .await;
        if let Some(handle) = scheduler_handle {
            handle.abort();
            if let Err(err) = handle.await
                && !err.is_cancelled()
            {
                warn!(error = %err, "scheduler 后台任务退出异常");
            }
        }
        serve_result?;
        Ok(())
    }

    pub async fn drain_notifications_once(
        &self,
        limit: Option<i64>,
    ) -> AppResult<crate::domain::notification::NotificationDrainReport> {
        self.state.notification.drain_once(limit).await
    }

    pub async fn notification_dead_letters(
        &self,
        limit: Option<i64>,
    ) -> AppResult<crate::domain::notification::NotificationDeadLetterReport> {
        self.state.notification.dead_letters(limit).await
    }

    pub async fn requeue_failed_notifications(
        &self,
        limit: Option<i64>,
    ) -> AppResult<crate::domain::notification::NotificationRequeueReport> {
        self.state.notification.requeue_failed(limit).await
    }

    pub async fn prune_operation_records(
        &self,
    ) -> AppResult<crate::domain::system::OperationRecordRetentionReport> {
        self.state.system.prune_operation_records().await
    }

    pub async fn operation_record_summary(
        &self,
        query: crate::domain::system::OperationRecordSummaryQuery,
    ) -> AppResult<crate::domain::system::OperationRecordSummary> {
        self.state.system.operation_record_summary(query).await
    }

    pub async fn run_scheduled_tasks_once(
        &self,
    ) -> AppResult<crate::scheduler::SchedulerRunReport> {
        scheduler::run_once(self.state.system.clone()).await
    }
}

fn media_storage_from_settings(settings: &Settings) -> AppResult<Arc<dyn MediaStorage>> {
    match settings.storage.driver {
        StorageDriver::Local => Ok(Arc::new(LocalMediaStorage::new(settings.storage.clone()))),
        StorageDriver::S3 => Ok(Arc::new(S3MediaStorage::new(settings.storage.clone())?)),
    }
}

fn api_catalog_entries_from_contracts(contracts: &[RouteContract]) -> Vec<ApiCatalogEntry> {
    contracts
        .iter()
        .filter(|contract| contract.include_catalog)
        .map(|contract| ApiCatalogEntry {
            id: contract.id.clone(),
            method: contract.method.clone(),
            path: contract.path.clone(),
            tag: contract.tag.clone(),
            summary: contract.summary.clone(),
            access: contract.access.clone(),
            permission: contract.permission.clone(),
            scope: contract.scope.clone(),
            product_code: contract.product_code.clone(),
        })
        .collect()
}

fn menu_entries_from_contracts(contracts: &[RouteContract]) -> Vec<SystemMenuEntry> {
    let menu_specs = [
        (
            "system.menus",
            "system.menus",
            "系统菜单",
            "/admin/system/menus",
            10,
        ),
        (
            "system.apis",
            "system.api-catalog",
            "API 目录",
            "/admin/system/apis",
            20,
        ),
        (
            "system.operation-records",
            "system.operation-records",
            "操作记录",
            "/admin/system/operation-records",
            30,
        ),
        (
            "system.server-status",
            "system.server-status",
            "服务器状态",
            "/admin/system/server-status",
            40,
        ),
        (
            "system.configs.list",
            "system.configs",
            "系统配置",
            "/admin/system/configs",
            45,
        ),
        (
            "system.dictionaries.list",
            "system.dictionaries",
            "系统字典",
            "/admin/system/dictionaries",
            46,
        ),
        (
            "system.parameters.list",
            "system.parameters",
            "系统参数",
            "/admin/system/parameters",
            47,
        ),
        (
            "system.version-packages.list",
            "system.version-packages",
            "版本包",
            "/admin/system/version-packages",
            48,
        ),
        (
            "system.media-assets.list",
            "system.media-assets",
            "媒体库",
            "/admin/system/media-assets",
            49,
        ),
        (
            "system.storage-objects.list",
            "system.storage-objects",
            "对象存储",
            "/admin/system/storage-objects",
            50,
        ),
        (
            "system.traffic-probes.targets.list",
            "system.traffic-probes",
            "流量探针",
            "/admin/system/traffic-probes",
            51,
        ),
        (
            "iam.api-tokens.list",
            "iam.api-tokens",
            "API Token",
            "/admin/iam/api-tokens",
            60,
        ),
        (
            "iam.invitations.list",
            "iam.invitations",
            "邀请管理",
            "/admin/iam/invitations",
            70,
        ),
    ];

    menu_specs
        .into_iter()
        .filter_map(|(route_id, code, title, path, sort_order)| {
            contracts
                .iter()
                .find(|contract| contract.id == route_id)
                .map(|contract| SystemMenuEntry {
                    code: code.into(),
                    title: title.into(),
                    path: path.into(),
                    permission: contract.permission.clone(),
                    scope: contract.scope.clone(),
                    sort_order,
                })
        })
        .collect()
}

async fn shutdown_signal() {
    let ctrl_c = async {
        tokio::signal::ctrl_c()
            .await
            .expect("failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .expect("failed to install signal handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {},
        _ = terminate => {},
    }
}

use std::collections::BTreeMap;
use std::sync::Arc;

use async_trait::async_trait;
use chrono::{DateTime, Duration, Utc};
use http::Uri;
use tracing::warn;

use crate::app::{AppError, AppResult};
use crate::config::Settings;
use crate::domain::system::{
    ApiCatalogEntry, ApiCatalogGroup, CreateMediaAssetRequest, CreateTrafficProbeTargetRequest,
    CreateVersionPackageRequest, DeleteResult, DeleteStorageObjectRequest, MediaAssetEntry,
    OperationRecord, OperationRecordQuery, OperationRecordRetentionReport, OperationRecordSummary,
    OperationRecordSummaryQuery, PublicAuthSettings, PublicSettings, ServerResourceMetrics,
    ServerStatus, StorageObjectEntry, StorageObjectQuery, SystemConfigEntry, SystemDictionaryEntry,
    SystemMenuEntry, SystemParameterEntry, TrafficProbeAlertActionResult, TrafficProbeAlertEntry,
    TrafficProbeAlertQuery, TrafficProbeEventQuery, TrafficProbeEventSnapshot,
    TrafficProbeObservation, TrafficProbeResultEntry, TrafficProbeResultQuery,
    TrafficProbeSweepReport, TrafficProbeTargetEntry, UpsertSystemConfigRequest,
    UpsertSystemDictionaryRequest, UpsertSystemParameterRequest, VersionPackageActionRequest,
    VersionPackageActionResult, VersionPackageEntry, VersionReleaseEventEntry,
};
use crate::repository::{
    CreateMediaAssetRecord, CreateOperationRecord, CreateTrafficProbeAlertRecord,
    CreateTrafficProbeResultRecord, CreateTrafficProbeTargetRecord, CreateVersionPackageRecord,
    OperationRecordListQuery, OperationRecordSummaryFilter, SystemRepository,
    UpsertSystemConfigRecord, UpsertSystemDictionaryRecord, UpsertSystemParameterRecord,
    VersionPackageActionRecord,
};

#[async_trait]
pub trait TrafficProbeRunner: Send + Sync {
    async fn probe(&self, target: &TrafficProbeTargetEntry) -> TrafficProbeObservation;
}

pub trait SystemMetricsCollector: Send + Sync {
    fn collect(&self) -> ServerResourceMetrics;
}

#[derive(Debug)]
pub struct MediaUploadInput {
    pub category: Option<String>,
    pub display_name: Option<String>,
    pub file_name: String,
    pub mime_type: String,
    pub bytes: Vec<u8>,
}

#[derive(Debug)]
pub struct StoreMediaObjectInput {
    pub file_name: String,
    pub mime_type: String,
    pub bytes: Vec<u8>,
}

#[derive(Debug)]
pub struct StoredMediaObject {
    pub storage_key: String,
    pub size_bytes: i64,
}

#[async_trait]
pub trait MediaStorage: Send + Sync {
    async fn put(&self, input: StoreMediaObjectInput) -> AppResult<StoredMediaObject>;
    async fn delete(&self, storage_key: &str) -> AppResult<()>;
    async fn list_objects(&self, query: StorageObjectQuery) -> AppResult<Vec<StorageObjectEntry>>;
}

pub struct SystemService {
    settings: Settings,
    repo: Arc<dyn SystemRepository>,
    traffic_probe_runner: Arc<dyn TrafficProbeRunner>,
    media_storage: Arc<dyn MediaStorage>,
    metrics_collector: Arc<dyn SystemMetricsCollector>,
    started_at: DateTime<Utc>,
}

impl SystemService {
    pub fn new(
        settings: Settings,
        repo: Arc<dyn SystemRepository>,
        traffic_probe_runner: Arc<dyn TrafficProbeRunner>,
        media_storage: Arc<dyn MediaStorage>,
        metrics_collector: Arc<dyn SystemMetricsCollector>,
    ) -> Self {
        Self {
            settings,
            repo,
            traffic_probe_runner,
            media_storage,
            metrics_collector,
            started_at: Utc::now(),
        }
    }

    pub fn public_settings(&self) -> PublicSettings {
        PublicSettings {
            product_name: self.settings.app.product_name.clone(),
            product_code: self.settings.app.product_code.clone(),
            default_locale: self.settings.i18n.default_locale.clone(),
            supported_locales: self.settings.i18n.supported_locales.clone(),
            auth: PublicAuthSettings {
                self_signup_enabled: self.settings.auth.self_signup_enabled,
                session_cookie_name: self.settings.auth.cookie.name.clone(),
                refresh_cookie_name: self.settings.auth.refresh_cookie.name.clone(),
                product_header: self.settings.auth.context.product_header.clone(),
                client_type_header: self.settings.auth.context.client_type_header.clone(),
                default_client_type: self.settings.auth.context.default_client_type.clone(),
                csrf_enabled: self.settings.auth.csrf.enabled,
                csrf_cookie_name: self.settings.auth.csrf.cookie_name.clone(),
                csrf_header_name: self.settings.auth.csrf.header_name.clone(),
            },
        }
    }

    pub async fn sync_api_catalog(
        &self,
        entries: &[ApiCatalogEntry],
        menus: &[SystemMenuEntry],
    ) -> AppResult<()> {
        self.repo.sync_api_catalog(entries).await?;
        self.repo.sync_system_menus(menus).await
    }

    pub async fn api_catalog(&self) -> AppResult<Vec<ApiCatalogGroup>> {
        let entries = self.repo.list_api_catalog().await?;
        Ok(group_api_catalog(entries))
    }

    pub async fn menus(&self, permissions: &[String]) -> AppResult<Vec<SystemMenuEntry>> {
        let menus = self.repo.list_system_menus().await?;
        Ok(menus
            .into_iter()
            .filter(|menu| {
                menu.permission
                    .as_ref()
                    .is_none_or(|permission| permissions.iter().any(|item| item == permission))
            })
            .collect())
    }

    pub async fn operation_records(
        &self,
        query: OperationRecordQuery,
    ) -> AppResult<Vec<OperationRecord>> {
        let method = normalize_operation_method(query.method)?;
        let path = normalize_operation_path(query.path)?;
        let status = normalize_operation_status(query.status)?;
        let actor_user_id = normalize_operation_actor(query.actor_user_id)?;
        let created_from = normalize_operation_time(query.created_from, "created_from")?;
        let created_to = normalize_operation_time(query.created_to, "created_to")?;
        if let (Some(from), Some(to)) = (&created_from, &created_to)
            && from > to
        {
            return Err(AppError::Validation(
                "操作记录 created_from 不能晚于 created_to".into(),
            ));
        }
        let limit = normalize_operation_limit(query.limit)?;
        let offset = normalize_operation_offset(query.offset)?;
        self.repo
            .list_operation_records(OperationRecordListQuery {
                method,
                path,
                status,
                actor_user_id,
                created_from: created_from.map(|value| value.to_rfc3339()),
                created_to: created_to.map(|value| value.to_rfc3339()),
                limit,
                offset,
            })
            .await
    }

    pub async fn operation_records_csv(&self, query: OperationRecordQuery) -> AppResult<String> {
        let records = self.operation_records(query).await?;
        let mut csv = String::from("id,actor_user_id,method,path,status,created_at\n");
        for record in records {
            csv.push_str(&record.id.to_string());
            csv.push(',');
            if let Some(actor_user_id) = record.actor_user_id {
                csv.push_str(&actor_user_id.to_string());
            }
            csv.push(',');
            push_csv_field(&mut csv, &record.method);
            csv.push(',');
            push_csv_field(&mut csv, &record.path);
            csv.push(',');
            csv.push_str(&record.status.to_string());
            csv.push(',');
            push_csv_field(&mut csv, &record.created_at);
            csv.push('\n');
        }
        Ok(csv)
    }

    pub async fn operation_record_summary(
        &self,
        query: OperationRecordSummaryQuery,
    ) -> AppResult<OperationRecordSummary> {
        let method = normalize_operation_method(query.method)?;
        let path = normalize_operation_path(query.path)?;
        let status = normalize_operation_status(query.status)?;
        let actor_user_id = normalize_operation_actor(query.actor_user_id)?;
        let created_from = normalize_operation_time(query.created_from, "created_from")?;
        let created_to = normalize_operation_time(query.created_to, "created_to")?;
        if let (Some(from), Some(to)) = (&created_from, &created_to)
            && from > to
        {
            return Err(AppError::Validation(
                "操作记录 created_from 不能晚于 created_to".into(),
            ));
        }
        let top_limit = normalize_operation_summary_top_limit(query.top_limit)?;
        let mut summary = self
            .repo
            .summarize_operation_records(OperationRecordSummaryFilter {
                method,
                path,
                status,
                actor_user_id,
                created_from: created_from.map(|value| value.to_rfc3339()),
                created_to: created_to.map(|value| value.to_rfc3339()),
                top_limit,
            })
            .await?;
        summary.generated_at = Utc::now().to_rfc3339();
        Ok(summary)
    }

    pub async fn prune_operation_records(&self) -> AppResult<OperationRecordRetentionReport> {
        let retention_days = self.settings.audit.operation_record_retention_days;
        let prune_batch_size = self.settings.audit.operation_record_prune_batch_size;
        let cutoff = (Utc::now() - Duration::days(retention_days)).to_rfc3339();
        let deleted = self
            .repo
            .prune_operation_records(&cutoff, prune_batch_size)
            .await?;
        Ok(OperationRecordRetentionReport {
            retention_days,
            cutoff,
            prune_batch_size,
            deleted,
        })
    }

    pub async fn record_operation(
        &self,
        actor_user_id: Option<i64>,
        method: &str,
        path: &str,
        status: u16,
    ) -> AppResult<()> {
        self.repo
            .create_operation_record(CreateOperationRecord {
                actor_user_id,
                method: method.into(),
                path: path.into(),
                status: i64::from(status),
            })
            .await
    }

    pub fn server_status(&self) -> ServerStatus {
        let collected_at = Utc::now();
        let uptime_seconds = collected_at
            .signed_duration_since(self.started_at)
            .num_seconds()
            .max(0);
        let available_parallelism = std::thread::available_parallelism()
            .map(usize::from)
            .unwrap_or(1);
        ServerStatus {
            source: "runtime-process".into(),
            collected_at: collected_at.to_rfc3339(),
            started_at: self.started_at.to_rfc3339(),
            uptime_seconds,
            process_id: std::process::id(),
            os: std::env::consts::OS.into(),
            arch: std::env::consts::ARCH.into(),
            available_parallelism,
            product_code: self.settings.app.product_code.clone(),
            version: self.settings.app.version.clone(),
            database_driver: self.settings.database.driver.to_string(),
            metrics: self.metrics_collector.collect(),
        }
    }

    pub fn prometheus_metrics(&self) -> String {
        prometheus_metrics_text(&self.server_status())
    }

    pub async fn configs(&self) -> AppResult<Vec<SystemConfigEntry>> {
        self.repo.list_system_configs().await
    }

    pub async fn upsert_config(
        &self,
        key: String,
        request: UpsertSystemConfigRequest,
    ) -> AppResult<SystemConfigEntry> {
        let key = key.trim().to_string();
        validate_key(&key)?;
        reject_secret_key(&key)?;
        let value_json = serde_json::to_string(&request.value)
            .map_err(|err| AppError::Validation(format!("配置值必须是有效 JSON：{err}")))?;
        self.repo
            .upsert_system_config(UpsertSystemConfigRecord { key, value_json })
            .await
    }

    pub async fn delete_config(&self, key: String) -> AppResult<DeleteResult> {
        let key = key.trim().to_string();
        validate_key(&key)?;
        Ok(DeleteResult {
            deleted: self.repo.delete_system_config(&key).await?,
        })
    }

    pub async fn dictionaries(&self) -> AppResult<Vec<SystemDictionaryEntry>> {
        self.repo.list_system_dictionaries().await
    }

    pub async fn upsert_dictionary(
        &self,
        code: String,
        request: UpsertSystemDictionaryRequest,
    ) -> AppResult<SystemDictionaryEntry> {
        let code = code.trim().to_string();
        validate_key(&code)?;
        let name = request.name.trim().to_string();
        if name.is_empty() {
            return Err(AppError::Validation("字典名称不能为空".into()));
        }
        self.repo
            .upsert_system_dictionary(UpsertSystemDictionaryRecord { code, name })
            .await
    }

    pub async fn delete_dictionary(&self, code: String) -> AppResult<DeleteResult> {
        let code = code.trim().to_string();
        validate_key(&code)?;
        Ok(DeleteResult {
            deleted: self.repo.delete_system_dictionary(&code).await?,
        })
    }

    pub async fn parameters(&self) -> AppResult<Vec<SystemParameterEntry>> {
        self.repo.list_system_parameters().await
    }

    pub async fn upsert_parameter(
        &self,
        key: String,
        request: UpsertSystemParameterRequest,
    ) -> AppResult<SystemParameterEntry> {
        let key = key.trim().to_string();
        validate_key(&key)?;
        reject_secret_key(&key)?;
        let name = request.name.trim().to_string();
        let value = request.value.trim().to_string();
        if name.is_empty() || value.is_empty() {
            return Err(AppError::Validation("参数名称和值不能为空".into()));
        }
        self.repo
            .upsert_system_parameter(UpsertSystemParameterRecord { key, name, value })
            .await
    }

    pub async fn delete_parameter(&self, key: String) -> AppResult<DeleteResult> {
        let key = key.trim().to_string();
        validate_key(&key)?;
        Ok(DeleteResult {
            deleted: self.repo.delete_system_parameter(&key).await?,
        })
    }

    pub async fn version_packages(&self) -> AppResult<Vec<VersionPackageEntry>> {
        self.repo.list_version_packages().await
    }

    pub async fn version_release_events(&self) -> AppResult<Vec<VersionReleaseEventEntry>> {
        self.repo.list_version_release_events().await
    }

    pub async fn create_version_package(
        &self,
        request: CreateVersionPackageRequest,
    ) -> AppResult<VersionPackageEntry> {
        let version_name = request.version_name.trim().to_string();
        let version_code = request.version_code.trim().to_string();
        validate_required_text("版本名称", &version_name)?;
        validate_key(&version_code)?;
        let manifest_json = serde_json::to_string(&request.manifest)
            .map_err(|err| AppError::Validation(format!("版本 manifest 必须是有效 JSON：{err}")))?;
        self.repo
            .create_version_package(CreateVersionPackageRecord {
                version_name,
                version_code,
                manifest_json,
            })
            .await
    }

    pub async fn delete_version_package(&self, id: i64) -> AppResult<DeleteResult> {
        validate_positive_id(id)?;
        Ok(DeleteResult {
            deleted: self.repo.delete_version_package(id).await?,
        })
    }

    pub async fn publish_version_package(
        &self,
        id: i64,
        request: VersionPackageActionRequest,
    ) -> AppResult<VersionPackageActionResult> {
        validate_positive_id(id)?;
        let reason = normalize_optional_reason(request.reason)?;
        self.repo
            .publish_version_package(VersionPackageActionRecord { id, reason })
            .await
    }

    pub async fn rollback_version_package(
        &self,
        id: i64,
        request: VersionPackageActionRequest,
    ) -> AppResult<VersionPackageActionResult> {
        validate_positive_id(id)?;
        let reason = normalize_optional_reason(request.reason)?;
        self.repo
            .rollback_version_package(VersionPackageActionRecord { id, reason })
            .await
    }

    pub async fn media_assets(&self) -> AppResult<Vec<MediaAssetEntry>> {
        self.repo.list_media_assets().await
    }

    pub async fn create_media_asset(
        &self,
        request: CreateMediaAssetRequest,
    ) -> AppResult<MediaAssetEntry> {
        let category = request
            .category
            .map(|value| value.trim().to_string())
            .filter(|value| !value.is_empty());
        if let Some(category) = &category {
            validate_key(category)?;
        }
        let display_name = request.display_name.trim().to_string();
        let storage_key = request.storage_key.trim().to_string();
        let mime_type = request.mime_type.trim().to_ascii_lowercase();
        validate_required_text("媒体名称", &display_name)?;
        validate_storage_key(&storage_key)?;
        reject_secret_key(&storage_key)?;
        validate_mime_type(&mime_type)?;
        if request.size_bytes <= 0 {
            return Err(AppError::Validation("媒体大小必须大于 0".into()));
        }
        self.repo
            .create_media_asset(CreateMediaAssetRecord {
                category,
                display_name,
                storage_key,
                mime_type,
                size_bytes: request.size_bytes,
            })
            .await
    }

    pub async fn upload_media_asset(&self, input: MediaUploadInput) -> AppResult<MediaAssetEntry> {
        let category = input
            .category
            .map(|value| value.trim().to_string())
            .filter(|value| !value.is_empty());
        if let Some(category) = &category {
            validate_key(category)?;
        }
        let file_name = input.file_name.trim().to_string();
        validate_upload_file_name(&file_name)?;
        let display_name = input
            .display_name
            .map(|value| value.trim().to_string())
            .filter(|value| !value.is_empty())
            .unwrap_or_else(|| file_name.clone());
        validate_required_text("媒体名称", &display_name)?;
        let mime_type = input.mime_type.trim().to_ascii_lowercase();
        validate_mime_type(&mime_type)?;
        if input.bytes.is_empty() {
            return Err(AppError::Validation("上传文件不能为空".into()));
        }
        if input.bytes.len() > self.settings.storage.max_upload_bytes {
            return Err(AppError::Validation(format!(
                "上传文件不能超过 {} 字节",
                self.settings.storage.max_upload_bytes
            )));
        }

        let stored = self
            .media_storage
            .put(StoreMediaObjectInput {
                file_name,
                mime_type: mime_type.clone(),
                bytes: input.bytes,
            })
            .await?;
        validate_storage_key(&stored.storage_key)?;
        reject_secret_key(&stored.storage_key)?;

        let record = CreateMediaAssetRecord {
            category,
            display_name,
            storage_key: stored.storage_key.clone(),
            mime_type,
            size_bytes: stored.size_bytes,
        };
        match self.repo.create_media_asset(record).await {
            Ok(asset) => Ok(asset),
            Err(err) => {
                if let Err(cleanup_err) = self.media_storage.delete(&stored.storage_key).await {
                    warn!(
                        storage_key = %stored.storage_key,
                        error = %cleanup_err,
                        "媒体元数据写入失败后清理本地对象失败"
                    );
                }
                Err(err)
            }
        }
    }

    pub async fn delete_media_asset(&self, id: i64) -> AppResult<DeleteResult> {
        validate_positive_id(id)?;
        Ok(DeleteResult {
            deleted: self.repo.delete_media_asset(id).await?,
        })
    }

    pub async fn storage_objects(
        &self,
        query: StorageObjectQuery,
    ) -> AppResult<Vec<StorageObjectEntry>> {
        self.media_storage
            .list_objects(normalize_storage_object_query(query)?)
            .await
    }

    pub async fn delete_storage_object(
        &self,
        request: DeleteStorageObjectRequest,
    ) -> AppResult<DeleteResult> {
        let storage_key = request.storage_key.trim().to_string();
        validate_storage_key(&storage_key)?;
        reject_secret_key(&storage_key)?;
        self.media_storage.delete(&storage_key).await?;
        Ok(DeleteResult { deleted: true })
    }

    pub async fn traffic_probe_targets(&self) -> AppResult<Vec<TrafficProbeTargetEntry>> {
        self.repo.list_traffic_probe_targets().await
    }

    pub async fn create_traffic_probe_target(
        &self,
        request: CreateTrafficProbeTargetRequest,
    ) -> AppResult<TrafficProbeTargetEntry> {
        let name = request.name.trim().to_string();
        validate_required_text("探针名称", &name)?;
        let url = validate_probe_url(&request.url)?;
        let expected_status = request.expected_status.unwrap_or(200);
        validate_http_status(expected_status)?;
        self.repo
            .create_traffic_probe_target(CreateTrafficProbeTargetRecord {
                name,
                url,
                expected_status,
            })
            .await
    }

    pub async fn delete_traffic_probe_target(&self, id: i64) -> AppResult<DeleteResult> {
        validate_positive_id(id)?;
        Ok(DeleteResult {
            deleted: self.repo.delete_traffic_probe_target(id).await?,
        })
    }

    pub async fn run_traffic_probe(&self, id: i64) -> AppResult<TrafficProbeResultEntry> {
        validate_positive_id(id)?;
        let target = self
            .repo
            .find_traffic_probe_target(id)
            .await?
            .ok_or_else(|| AppError::NotFound("流量探针目标不存在".into()))?;
        let observation = self.traffic_probe_runner.probe(&target).await;
        let detail_json = serde_json::to_string(&observation.detail)
            .map_err(|err| AppError::Internal(format!("探针结果 JSON 序列化失败：{err}")))?;
        let result = self
            .repo
            .create_traffic_probe_result(CreateTrafficProbeResultRecord {
                target_id: target.id,
                status: observation.status,
                detail_json,
            })
            .await?;
        self.apply_traffic_probe_alert_transition(&target, &result)
            .await?;
        Ok(result)
    }

    pub async fn run_all_traffic_probes(&self) -> AppResult<TrafficProbeSweepReport> {
        let targets = self.repo.list_traffic_probe_targets().await?;
        let mut report = TrafficProbeSweepReport {
            scanned_targets: targets.len(),
            recorded_results: 0,
            failed_targets: 0,
        };

        for target in targets {
            match self.run_traffic_probe(target.id).await {
                Ok(_) => report.recorded_results += 1,
                Err(err) => {
                    report.failed_targets += 1;
                    warn!(
                        target_id = target.id,
                        error = %err,
                        "scheduler 执行流量探针目标失败"
                    );
                }
            }
        }

        Ok(report)
    }

    pub async fn traffic_probe_results(
        &self,
        query: TrafficProbeResultQuery,
    ) -> AppResult<Vec<TrafficProbeResultEntry>> {
        if let Some(target_id) = query.target_id {
            validate_positive_id(target_id)?;
        }
        self.repo
            .list_traffic_probe_results(query.target_id, query.limit.unwrap_or(50))
            .await
    }

    pub async fn traffic_probe_alerts(
        &self,
        query: TrafficProbeAlertQuery,
    ) -> AppResult<Vec<TrafficProbeAlertEntry>> {
        if let Some(target_id) = query.target_id {
            validate_positive_id(target_id)?;
        }
        let status = query
            .status
            .map(|value| validate_probe_alert_status(&value))
            .transpose()?;
        self.repo
            .list_traffic_probe_alerts(query.target_id, status, query.limit.unwrap_or(50))
            .await
    }

    pub async fn traffic_probe_event_snapshot(
        &self,
        query: TrafficProbeEventQuery,
    ) -> AppResult<TrafficProbeEventSnapshot> {
        let alerts = self
            .traffic_probe_alerts(TrafficProbeAlertQuery {
                target_id: query.target_id,
                status: query.status,
                limit: query.limit,
            })
            .await?;
        Ok(TrafficProbeEventSnapshot {
            event_type: "traffic_probe.alerts.snapshot".into(),
            generated_at: Utc::now().to_rfc3339(),
            reconnect_after_millis: self
                .settings
                .scheduler
                .event_stream_heartbeat_seconds
                .saturating_mul(2_000),
            alerts,
        })
    }

    pub async fn acknowledge_traffic_probe_alert(
        &self,
        id: i64,
    ) -> AppResult<TrafficProbeAlertActionResult> {
        validate_positive_id(id)?;
        Ok(TrafficProbeAlertActionResult {
            updated: self.repo.acknowledge_traffic_probe_alert(id).await?,
        })
    }

    pub async fn resolve_traffic_probe_alert(
        &self,
        id: i64,
    ) -> AppResult<TrafficProbeAlertActionResult> {
        validate_positive_id(id)?;
        Ok(TrafficProbeAlertActionResult {
            updated: self.repo.resolve_traffic_probe_alert(id).await?,
        })
    }

    async fn apply_traffic_probe_alert_transition(
        &self,
        target: &TrafficProbeTargetEntry,
        result: &TrafficProbeResultEntry,
    ) -> AppResult<()> {
        if result.status == "healthy" {
            self.repo
                .resolve_traffic_probe_alerts_for_target(target.id)
                .await?;
            return Ok(());
        }
        let severity = if result.status == "critical" {
            "critical"
        } else {
            "warning"
        };
        let reason = result
            .detail
            .get("reason")
            .and_then(serde_json::Value::as_str)
            .unwrap_or("traffic_probe_unhealthy")
            .to_string();
        let detail_json = serde_json::to_string(&result.detail)
            .map_err(|err| AppError::Internal(format!("探针告警 JSON 序列化失败：{err}")))?;
        self.repo
            .create_traffic_probe_alert(CreateTrafficProbeAlertRecord {
                target_id: target.id,
                result_id: result.id,
                severity: severity.into(),
                reason,
                detail_json,
            })
            .await?;
        Ok(())
    }
}

fn group_api_catalog(entries: Vec<ApiCatalogEntry>) -> Vec<ApiCatalogGroup> {
    let mut groups: BTreeMap<String, Vec<ApiCatalogEntry>> = BTreeMap::new();
    for entry in entries {
        groups.entry(entry.tag.clone()).or_default().push(entry);
    }
    groups
        .into_iter()
        .map(|(tag, items)| ApiCatalogGroup { tag, items })
        .collect()
}

fn validate_key(value: &str) -> AppResult<()> {
    let trimmed = value.trim();
    if trimmed.is_empty() {
        return Err(AppError::Validation("键名不能为空".into()));
    }
    if trimmed.len() > 120 {
        return Err(AppError::Validation("键名不能超过 120 个字符".into()));
    }
    if !trimmed
        .chars()
        .all(|ch| ch.is_ascii_alphanumeric() || matches!(ch, '_' | '-' | '.'))
    {
        return Err(AppError::Validation(
            "键名只能包含字母、数字、下划线、中划线和点".into(),
        ));
    }
    Ok(())
}

fn validate_required_text(label: &str, value: &str) -> AppResult<()> {
    if value.trim().is_empty() {
        return Err(AppError::Validation(format!("{label}不能为空")));
    }
    if value.chars().count() > 120 {
        return Err(AppError::Validation(format!("{label}不能超过 120 个字符")));
    }
    Ok(())
}

fn normalize_optional_reason(value: Option<String>) -> AppResult<Option<String>> {
    let Some(value) = value else {
        return Ok(None);
    };
    let trimmed = value.trim();
    if trimmed.is_empty() {
        return Ok(None);
    }
    if trimmed.chars().count() > 240 {
        return Err(AppError::Validation(
            "版本发布原因不能超过 240 个字符".into(),
        ));
    }
    if contains_secret_word(trimmed) {
        return Err(AppError::Validation(
            "版本发布原因不能包含 token、secret、password 等敏感信息".into(),
        ));
    }
    Ok(Some(trimmed.to_string()))
}

fn validate_positive_id(id: i64) -> AppResult<()> {
    if id <= 0 {
        return Err(AppError::Validation("资源 ID 必须大于 0".into()));
    }
    Ok(())
}

fn validate_storage_key(value: &str) -> AppResult<()> {
    validate_required_text("存储键", value)?;
    if value.len() > 240 {
        return Err(AppError::Validation("存储键不能超过 240 个字符".into()));
    }
    if value.contains("..") || value.starts_with('/') || value.starts_with('\\') {
        return Err(AppError::Validation(
            "存储键不能包含路径穿越或绝对路径".into(),
        ));
    }
    Ok(())
}

fn normalize_storage_object_query(mut query: StorageObjectQuery) -> AppResult<StorageObjectQuery> {
    query.prefix = query
        .prefix
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty());
    if let Some(prefix) = &query.prefix {
        validate_storage_key(prefix)?;
        reject_secret_key(prefix)?;
    }
    let limit = query.limit.unwrap_or(100);
    if limit == 0 {
        return Err(AppError::Validation("对象列表 limit 必须大于 0".into()));
    }
    query.limit = Some(limit.min(500));
    Ok(query)
}

fn validate_upload_file_name(value: &str) -> AppResult<()> {
    validate_required_text("上传文件名", value)?;
    if value.len() > 180 {
        return Err(AppError::Validation("上传文件名不能超过 180 个字符".into()));
    }
    if value.contains("..") || value.contains('/') || value.contains('\\') {
        return Err(AppError::Validation(
            "上传文件名不能包含路径穿越或目录分隔符".into(),
        ));
    }
    Ok(())
}

fn validate_mime_type(value: &str) -> AppResult<()> {
    if value.is_empty() || !value.contains('/') || value.chars().any(char::is_whitespace) {
        return Err(AppError::Validation("媒体 MIME 类型无效".into()));
    }
    Ok(())
}

fn validate_http_status(value: i64) -> AppResult<()> {
    if !(100..=599).contains(&value) {
        return Err(AppError::Validation(
            "预期 HTTP 状态码必须在 100 到 599 之间".into(),
        ));
    }
    Ok(())
}

fn validate_probe_alert_status(value: &str) -> AppResult<String> {
    let trimmed = value.trim();
    match trimmed {
        "open" | "acknowledged" | "resolved" => Ok(trimmed.to_string()),
        _ => Err(AppError::Validation(
            "流量探针告警状态只允许 open、acknowledged 或 resolved".into(),
        )),
    }
}

fn validate_probe_url(value: &str) -> AppResult<String> {
    let trimmed = value.trim();
    validate_required_text("探针 URL", trimmed)?;
    if trimmed.len() > 512 {
        return Err(AppError::Validation("探针 URL 不能超过 512 个字符".into()));
    }
    let uri = trimmed
        .parse::<Uri>()
        .map_err(|_| AppError::Validation("探针 URL 必须是有效的 HTTP/HTTPS 地址".into()))?;
    match uri.scheme_str() {
        Some("http" | "https") => {}
        _ => {
            return Err(AppError::Validation("探针 URL 只允许 http 或 https".into()));
        }
    }
    let authority = uri
        .authority()
        .ok_or_else(|| AppError::Validation("探针 URL 必须包含主机名".into()))?;
    if authority.as_str().contains('@') {
        return Err(AppError::Validation(
            "探针 URL 不允许包含用户名或密码".into(),
        ));
    }
    reject_sensitive_query(&uri)?;
    Ok(trimmed.to_string())
}

fn normalize_operation_method(method: Option<String>) -> AppResult<Option<String>> {
    let Some(method) = method else {
        return Ok(None);
    };
    let method = method.trim().to_ascii_uppercase();
    if method.is_empty() {
        return Ok(None);
    }
    const ALLOWED_METHODS: [&str; 6] = ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"];
    if !ALLOWED_METHODS.contains(&method.as_str()) {
        return Err(AppError::Validation(
            "操作记录 method 只能是 GET、POST、PUT、PATCH、DELETE 或 OPTIONS".into(),
        ));
    }
    Ok(Some(method))
}

fn normalize_operation_path(path: Option<String>) -> AppResult<Option<String>> {
    let Some(path) = path else {
        return Ok(None);
    };
    let path = path.trim();
    if path.is_empty() {
        return Ok(None);
    }
    if path.len() > 256 {
        return Err(AppError::Validation(
            "操作记录 path 不能超过 256 个字符".into(),
        ));
    }
    if !path.starts_with('/') || path.contains('?') || path.contains('#') {
        return Err(AppError::Validation(
            "操作记录 path 只能按 route 模板过滤，不能包含 query 或 fragment".into(),
        ));
    }
    Ok(Some(path.to_string()))
}

fn normalize_operation_status(status: Option<i64>) -> AppResult<Option<i64>> {
    let Some(status) = status else {
        return Ok(None);
    };
    if !(100..=599).contains(&status) {
        return Err(AppError::Validation(
            "操作记录 status 必须是 100 到 599 之间的 HTTP 状态码".into(),
        ));
    }
    Ok(Some(status))
}

fn normalize_operation_actor(actor_user_id: Option<i64>) -> AppResult<Option<i64>> {
    let Some(actor_user_id) = actor_user_id else {
        return Ok(None);
    };
    if actor_user_id <= 0 {
        return Err(AppError::Validation(
            "操作记录 actor_user_id 必须是正整数".into(),
        ));
    }
    Ok(Some(actor_user_id))
}

fn normalize_operation_time(
    value: Option<String>,
    field_name: &str,
) -> AppResult<Option<DateTime<Utc>>> {
    let Some(value) = value else {
        return Ok(None);
    };
    let value = value.trim();
    if value.is_empty() {
        return Ok(None);
    }
    let parsed = DateTime::parse_from_rfc3339(value).map_err(|_| {
        AppError::Validation(format!(
            "操作记录 {field_name} 必须是 RFC3339 时间，例如 2026-06-21T00:00:00Z"
        ))
    })?;
    Ok(Some(parsed.with_timezone(&Utc)))
}

fn normalize_operation_limit(limit: Option<i64>) -> AppResult<i64> {
    let limit = limit.unwrap_or(100);
    if !(1..=200).contains(&limit) {
        return Err(AppError::Validation(
            "操作记录 limit 必须在 1 到 200 之间".into(),
        ));
    }
    Ok(limit)
}

fn normalize_operation_offset(offset: Option<i64>) -> AppResult<i64> {
    let offset = offset.unwrap_or(0);
    if !(0..=10_000).contains(&offset) {
        return Err(AppError::Validation(
            "操作记录 offset 必须在 0 到 10000 之间".into(),
        ));
    }
    Ok(offset)
}

fn normalize_operation_summary_top_limit(top_limit: Option<i64>) -> AppResult<i64> {
    let top_limit = top_limit.unwrap_or(10);
    if !(1..=50).contains(&top_limit) {
        return Err(AppError::Validation(
            "操作记录 top_limit 必须在 1 到 50 之间".into(),
        ));
    }
    Ok(top_limit)
}

fn push_csv_field(output: &mut String, value: &str) {
    if value
        .chars()
        .any(|item| matches!(item, ',' | '"' | '\r' | '\n'))
    {
        output.push('"');
        for item in value.chars() {
            if item == '"' {
                output.push('"');
            }
            output.push(item);
        }
        output.push('"');
    } else {
        output.push_str(value);
    }
}

fn reject_sensitive_query(uri: &Uri) -> AppResult<()> {
    let Some(query) = uri.path_and_query().and_then(|value| value.query()) else {
        return Ok(());
    };
    for pair in query.split('&') {
        let key = pair.split('=').next().unwrap_or_default();
        if contains_secret_word(key) {
            return Err(AppError::Validation(
                "探针 URL 查询参数不能包含 token、secret、password 等敏感键".into(),
            ));
        }
    }
    Ok(())
}

fn reject_secret_key(key: &str) -> AppResult<()> {
    if contains_secret_word(key) {
        return Err(AppError::Validation(
            "敏感配置必须通过 secrets/env 管理，不能写入 System 表".into(),
        ));
    }
    Ok(())
}

fn contains_secret_word(value: &str) -> bool {
    let lowered = value.to_ascii_lowercase();
    let blocked = ["secret", "token", "password", "private", "credential"];
    blocked.iter().any(|item| lowered.contains(item))
}

fn prometheus_metrics_text(status: &ServerStatus) -> String {
    let labels = [
        ("product_code", status.product_code.as_str()),
        ("database_driver", status.database_driver.as_str()),
        ("metrics_source", status.metrics.source.as_str()),
        ("os", status.os.as_str()),
        ("arch", status.arch.as_str()),
        ("version", status.version.as_str()),
    ];
    let mut output = String::new();
    output.push_str("# HELP console_process_uptime_seconds 当前进程启动后的运行秒数。\n");
    output.push_str("# TYPE console_process_uptime_seconds gauge\n");
    push_metric(
        &mut output,
        "console_process_uptime_seconds",
        &labels,
        status.uptime_seconds,
    );
    output.push_str("# HELP console_process_available_parallelism 当前进程可用并行度。\n");
    output.push_str("# TYPE console_process_available_parallelism gauge\n");
    push_metric(
        &mut output,
        "console_process_available_parallelism",
        &labels,
        status.available_parallelism,
    );
    output.push_str("# HELP console_cpu_usage_percent 后端采集到的全局 CPU 使用率百分比。\n");
    output.push_str("# TYPE console_cpu_usage_percent gauge\n");
    push_metric(
        &mut output,
        "console_cpu_usage_percent",
        &labels,
        status.metrics.cpu_usage_percent,
    );
    output.push_str("# HELP console_process_cpu_usage_percent 当前进程 CPU 使用率百分比。\n");
    output.push_str("# TYPE console_process_cpu_usage_percent gauge\n");
    push_metric(
        &mut output,
        "console_process_cpu_usage_percent",
        &labels,
        status.metrics.process_cpu_usage_percent,
    );
    output.push_str("# HELP console_memory_bytes 后端采集到的内存字节数。\n");
    output.push_str("# TYPE console_memory_bytes gauge\n");
    push_metric(
        &mut output,
        "console_memory_bytes",
        &[&labels[..], &[("kind", "total")]].concat(),
        status.metrics.total_memory_bytes,
    );
    push_metric(
        &mut output,
        "console_memory_bytes",
        &[&labels[..], &[("kind", "used")]].concat(),
        status.metrics.used_memory_bytes,
    );
    push_metric(
        &mut output,
        "console_memory_bytes",
        &[&labels[..], &[("kind", "available")]].concat(),
        status.metrics.available_memory_bytes,
    );
    output.push_str("# HELP console_process_memory_bytes 当前进程内存字节数。\n");
    output.push_str("# TYPE console_process_memory_bytes gauge\n");
    push_metric(
        &mut output,
        "console_process_memory_bytes",
        &[&labels[..], &[("kind", "resident")]].concat(),
        status.metrics.process_memory_bytes,
    );
    push_metric(
        &mut output,
        "console_process_memory_bytes",
        &[&labels[..], &[("kind", "virtual")]].concat(),
        status.metrics.process_virtual_memory_bytes,
    );
    output.push_str("# HELP console_swap_bytes 后端采集到的交换空间字节数。\n");
    output.push_str("# TYPE console_swap_bytes gauge\n");
    push_metric(
        &mut output,
        "console_swap_bytes",
        &[&labels[..], &[("kind", "total")]].concat(),
        status.metrics.total_swap_bytes,
    );
    push_metric(
        &mut output,
        "console_swap_bytes",
        &[&labels[..], &[("kind", "used")]].concat(),
        status.metrics.used_swap_bytes,
    );
    output.push_str("# HELP console_disk_bytes 后端采集到的磁盘字节数。\n");
    output.push_str("# TYPE console_disk_bytes gauge\n");
    push_metric(
        &mut output,
        "console_disk_bytes",
        &[&labels[..], &[("kind", "total")]].concat(),
        status.metrics.total_disk_bytes,
    );
    push_metric(
        &mut output,
        "console_disk_bytes",
        &[&labels[..], &[("kind", "used")]].concat(),
        status.metrics.used_disk_bytes,
    );
    push_metric(
        &mut output,
        "console_disk_bytes",
        &[&labels[..], &[("kind", "available")]].concat(),
        status.metrics.available_disk_bytes,
    );
    output.push_str("# HELP console_disk_count 后端采集到的磁盘数量。\n");
    output.push_str("# TYPE console_disk_count gauge\n");
    push_metric(
        &mut output,
        "console_disk_count",
        &labels,
        status.metrics.disk_count,
    );
    output.push_str("# HELP console_network_interface_count 后端采集到的网络接口数量。\n");
    output.push_str("# TYPE console_network_interface_count gauge\n");
    push_metric(
        &mut output,
        "console_network_interface_count",
        &labels,
        status.metrics.network_interface_count,
    );
    output.push_str("# HELP console_network_bytes 后端采集到的网络累计字节数。\n");
    output.push_str("# TYPE console_network_bytes counter\n");
    push_metric(
        &mut output,
        "console_network_bytes",
        &[&labels[..], &[("direction", "received")]].concat(),
        status.metrics.network_received_bytes,
    );
    push_metric(
        &mut output,
        "console_network_bytes",
        &[&labels[..], &[("direction", "transmitted")]].concat(),
        status.metrics.network_transmitted_bytes,
    );
    output.push_str("# HELP console_system_uptime_seconds 操作系统运行秒数。\n");
    output.push_str("# TYPE console_system_uptime_seconds gauge\n");
    push_metric(
        &mut output,
        "console_system_uptime_seconds",
        &labels,
        status.metrics.system_uptime_seconds,
    );
    output.push_str("# HELP console_system_boot_time_seconds 操作系统启动 Unix 时间戳秒数。\n");
    output.push_str("# TYPE console_system_boot_time_seconds gauge\n");
    push_metric(
        &mut output,
        "console_system_boot_time_seconds",
        &labels,
        status.metrics.system_boot_time_seconds,
    );
    output.push_str("# HELP console_load_average 后端采集到的系统 load average。\n");
    output.push_str("# TYPE console_load_average gauge\n");
    push_metric(
        &mut output,
        "console_load_average",
        &[&labels[..], &[("window", "1m")]].concat(),
        status.metrics.load_average_one,
    );
    push_metric(
        &mut output,
        "console_load_average",
        &[&labels[..], &[("window", "5m")]].concat(),
        status.metrics.load_average_five,
    );
    push_metric(
        &mut output,
        "console_load_average",
        &[&labels[..], &[("window", "15m")]].concat(),
        status.metrics.load_average_fifteen,
    );
    output
}

fn push_metric(
    output: &mut String,
    name: &str,
    labels: &[(&str, &str)],
    value: impl std::fmt::Display,
) {
    output.push_str(name);
    output.push('{');
    for (index, (key, value)) in labels.iter().enumerate() {
        if index > 0 {
            output.push(',');
        }
        output.push_str(key);
        output.push_str("=\"");
        push_escaped_prometheus_label(output, value);
        output.push('"');
    }
    output.push_str("} ");
    output.push_str(&value.to_string());
    output.push('\n');
}

fn push_escaped_prometheus_label(output: &mut String, value: &str) {
    for ch in value.chars() {
        match ch {
            '\\' => output.push_str("\\\\"),
            '"' => output.push_str("\\\""),
            '\n' => output.push_str("\\n"),
            _ => output.push(ch),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn prometheus_metrics_escape_labels_and_use_real_status_fields() {
        let status = ServerStatus {
            source: "runtime-process".into(),
            collected_at: "2026-06-22T00:00:00Z".into(),
            started_at: "2026-06-22T00:00:00Z".into(),
            uptime_seconds: 12,
            process_id: 42,
            os: "windows".into(),
            arch: "x86_64".into(),
            available_parallelism: 4,
            product_code: "console\"prod".into(),
            version: "0.1.0\nnext".into(),
            database_driver: "sqlite".into(),
            metrics: ServerResourceMetrics {
                source: "sysinfo".into(),
                cpu_usage_percent: 1.5,
                process_cpu_usage_percent: 0.5,
                total_memory_bytes: 100,
                used_memory_bytes: 40,
                available_memory_bytes: 60,
                process_memory_bytes: 10,
                process_virtual_memory_bytes: 20,
                total_swap_bytes: 30,
                used_swap_bytes: 5,
                total_disk_bytes: 200,
                used_disk_bytes: 80,
                available_disk_bytes: 120,
                disk_count: 2,
                network_interface_count: 3,
                network_received_bytes: 2048,
                network_transmitted_bytes: 4096,
                system_uptime_seconds: 1000,
                system_boot_time_seconds: 999,
                load_average_one: 0.1,
                load_average_five: 0.2,
                load_average_fifteen: 0.3,
            },
        };

        let metrics = prometheus_metrics_text(&status);

        assert!(metrics.contains("# TYPE console_cpu_usage_percent gauge"));
        assert!(metrics.contains("console_process_uptime_seconds{"));
        assert!(metrics.contains("product_code=\"console\\\"prod\""));
        assert!(metrics.contains("version=\"0.1.0\\nnext\""));
        assert!(metrics.contains("console_memory_bytes{"));
        assert!(metrics.contains("kind=\"used\""));
        assert!(metrics.contains("console_network_bytes{"));
        assert!(metrics.contains("direction=\"received\""));
        assert!(metrics.contains("window=\"15m\""));
        assert!(!metrics.contains("token"));
        assert!(!metrics.contains("secret"));
    }
}

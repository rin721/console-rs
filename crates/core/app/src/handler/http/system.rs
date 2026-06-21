use std::convert::Infallible;
use std::sync::Arc;
use std::time::Duration;

use axum::Json;
use axum::extract::{Multipart, Path, Query, State};
use axum::http::{HeaderMap, HeaderValue, header};
use axum::response::IntoResponse;
use axum::response::sse::{Event, KeepAlive, Sse};
use chrono::Utc;
use serde::Serialize;
use tokio::time::MissedTickBehavior;
use tracing::warn;

use crate::app::AppError;
use crate::app::AppState;
use crate::config::Settings;
use crate::domain::system::{
    ApiCatalogGroup, CreateMediaAssetRequest, CreateTrafficProbeTargetRequest,
    CreateVersionPackageRequest, DeleteResult, DeleteStorageObjectRequest, MediaAssetEntry,
    OperationRecord, OperationRecordQuery, OperationRecordRetentionReport, OperationRecordSummary,
    OperationRecordSummaryQuery, ServerStatus, StorageObjectEntry, StorageObjectQuery,
    SystemConfigEntry, SystemDictionaryEntry, SystemMenuEntry, SystemParameterEntry,
    TrafficProbeAlertActionResult, TrafficProbeAlertEntry, TrafficProbeAlertQuery,
    TrafficProbeEventError, TrafficProbeEventQuery, TrafficProbeResultEntry,
    TrafficProbeResultQuery, TrafficProbeTargetEntry, UpsertSystemConfigRequest,
    UpsertSystemDictionaryRequest, UpsertSystemParameterRequest, VersionPackageActionRequest,
    VersionPackageActionResult, VersionPackageEntry, VersionReleaseEventEntry,
};
use crate::handler::http::error::HttpResult;
use crate::service::system::MediaUploadInput;
use crate::transport::http::request_context::{
    auth_credential_from_headers, bearer_token, request_context_from_headers, set_csrf_cookie,
};
use crate::transport::http::route_registry;

pub async fn public_settings(State(state): State<Arc<AppState>>) -> impl IntoResponse {
    let body = Json(state.system.public_settings());
    if let Some(headers) = set_csrf_cookie(&state.settings) {
        (headers, body).into_response()
    } else {
        body.into_response()
    }
}

pub async fn apis(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<Vec<ApiCatalogGroup>>> {
    authorize(&state, &headers, "system.apis").await?;
    Ok(Json(state.system.api_catalog().await?))
}

pub async fn menus(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<Vec<SystemMenuEntry>>> {
    let snapshot = authorize(&state, &headers, "system.menus").await?;
    Ok(Json(state.system.menus(&snapshot.permissions).await?))
}

pub async fn operation_records(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Query(query): Query<OperationRecordQuery>,
) -> HttpResult<Json<Vec<OperationRecord>>> {
    authorize(&state, &headers, "system.operation-records").await?;
    Ok(Json(state.system.operation_records(query).await?))
}

pub async fn operation_records_export(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Query(query): Query<OperationRecordQuery>,
) -> HttpResult<impl IntoResponse> {
    authorize(&state, &headers, "system.operation-records.export").await?;
    let headers = [
        (
            header::CONTENT_TYPE,
            HeaderValue::from_static("text/csv; charset=utf-8"),
        ),
        (
            header::CONTENT_DISPOSITION,
            HeaderValue::from_static("attachment; filename=\"operation-records.csv\""),
        ),
    ];
    Ok((headers, state.system.operation_records_csv(query).await?))
}

pub async fn operation_record_summary(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Query(query): Query<OperationRecordSummaryQuery>,
) -> HttpResult<Json<OperationRecordSummary>> {
    authorize(&state, &headers, "system.operation-records.summary").await?;
    Ok(Json(state.system.operation_record_summary(query).await?))
}

pub async fn prune_operation_records(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<OperationRecordRetentionReport>> {
    authorize(&state, &headers, "system.operation-records.prune").await?;
    Ok(Json(state.system.prune_operation_records().await?))
}

pub async fn server_status(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<ServerStatus>> {
    authorize(&state, &headers, "system.server-status").await?;
    Ok(Json(state.system.server_status()))
}

pub async fn prometheus_metrics(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<impl IntoResponse> {
    if !prometheus_scrape_token_authorized(&headers, &state.settings) {
        authorize(&state, &headers, "system.metrics.prometheus").await?;
    }
    let headers = [(
        header::CONTENT_TYPE,
        HeaderValue::from_static("text/plain; version=0.0.4; charset=utf-8"),
    )];
    Ok((headers, state.system.prometheus_metrics()))
}

pub async fn configs(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<Vec<SystemConfigEntry>>> {
    authorize(&state, &headers, "system.configs.list").await?;
    Ok(Json(state.system.configs().await?))
}

pub async fn upsert_config(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(key): Path<String>,
    Json(payload): Json<UpsertSystemConfigRequest>,
) -> HttpResult<Json<SystemConfigEntry>> {
    authorize(&state, &headers, "system.configs.upsert").await?;
    Ok(Json(state.system.upsert_config(key, payload).await?))
}

pub async fn delete_config(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(key): Path<String>,
) -> HttpResult<Json<DeleteResult>> {
    authorize(&state, &headers, "system.configs.delete").await?;
    Ok(Json(state.system.delete_config(key).await?))
}

pub async fn dictionaries(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<Vec<SystemDictionaryEntry>>> {
    authorize(&state, &headers, "system.dictionaries.list").await?;
    Ok(Json(state.system.dictionaries().await?))
}

pub async fn upsert_dictionary(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(code): Path<String>,
    Json(payload): Json<UpsertSystemDictionaryRequest>,
) -> HttpResult<Json<SystemDictionaryEntry>> {
    authorize(&state, &headers, "system.dictionaries.upsert").await?;
    Ok(Json(state.system.upsert_dictionary(code, payload).await?))
}

pub async fn delete_dictionary(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(code): Path<String>,
) -> HttpResult<Json<DeleteResult>> {
    authorize(&state, &headers, "system.dictionaries.delete").await?;
    Ok(Json(state.system.delete_dictionary(code).await?))
}

pub async fn parameters(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<Vec<SystemParameterEntry>>> {
    authorize(&state, &headers, "system.parameters.list").await?;
    Ok(Json(state.system.parameters().await?))
}

pub async fn upsert_parameter(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(key): Path<String>,
    Json(payload): Json<UpsertSystemParameterRequest>,
) -> HttpResult<Json<SystemParameterEntry>> {
    authorize(&state, &headers, "system.parameters.upsert").await?;
    Ok(Json(state.system.upsert_parameter(key, payload).await?))
}

pub async fn delete_parameter(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(key): Path<String>,
) -> HttpResult<Json<DeleteResult>> {
    authorize(&state, &headers, "system.parameters.delete").await?;
    Ok(Json(state.system.delete_parameter(key).await?))
}

pub async fn version_packages(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<Vec<VersionPackageEntry>>> {
    authorize(&state, &headers, "system.version-packages.list").await?;
    Ok(Json(state.system.version_packages().await?))
}

pub async fn create_version_package(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(payload): Json<CreateVersionPackageRequest>,
) -> HttpResult<Json<VersionPackageEntry>> {
    authorize(&state, &headers, "system.version-packages.create").await?;
    Ok(Json(state.system.create_version_package(payload).await?))
}

pub async fn version_release_events(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<Vec<VersionReleaseEventEntry>>> {
    authorize(&state, &headers, "system.version-packages.releases").await?;
    Ok(Json(state.system.version_release_events().await?))
}

pub async fn publish_version_package(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(id): Path<i64>,
    Json(payload): Json<VersionPackageActionRequest>,
) -> HttpResult<Json<VersionPackageActionResult>> {
    authorize(&state, &headers, "system.version-packages.publish").await?;
    Ok(Json(
        state.system.publish_version_package(id, payload).await?,
    ))
}

pub async fn rollback_version_package(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(id): Path<i64>,
    Json(payload): Json<VersionPackageActionRequest>,
) -> HttpResult<Json<VersionPackageActionResult>> {
    authorize(&state, &headers, "system.version-packages.rollback").await?;
    Ok(Json(
        state.system.rollback_version_package(id, payload).await?,
    ))
}

pub async fn delete_version_package(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(id): Path<i64>,
) -> HttpResult<Json<DeleteResult>> {
    authorize(&state, &headers, "system.version-packages.delete").await?;
    Ok(Json(state.system.delete_version_package(id).await?))
}

pub async fn media_assets(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<Vec<MediaAssetEntry>>> {
    authorize(&state, &headers, "system.media-assets.list").await?;
    Ok(Json(state.system.media_assets().await?))
}

pub async fn create_media_asset(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(payload): Json<CreateMediaAssetRequest>,
) -> HttpResult<Json<MediaAssetEntry>> {
    authorize(&state, &headers, "system.media-assets.create").await?;
    Ok(Json(state.system.create_media_asset(payload).await?))
}

pub async fn upload_media_asset(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    mut multipart: Multipart,
) -> HttpResult<Json<MediaAssetEntry>> {
    authorize(&state, &headers, "system.media-assets.upload").await?;
    let mut category = None;
    let mut display_name = None;
    let mut file_name = None;
    let mut mime_type = None;
    let mut bytes = None;

    while let Some(field) = multipart
        .next_field()
        .await
        .map_err(|err| AppError::Validation(format!("multipart 解析失败：{err}")))?
    {
        let name = field.name().unwrap_or_default().to_string();
        match name.as_str() {
            "category" => {
                category = Some(text_field(field).await?);
            }
            "display_name" | "displayName" => {
                display_name = Some(text_field(field).await?);
            }
            "file" => {
                if bytes.is_some() {
                    return Err(AppError::Validation("一次只能上传一个媒体文件".into()).into());
                }
                let uploaded_file_name = field
                    .file_name()
                    .map(str::to_string)
                    .ok_or_else(|| AppError::Validation("上传文件缺少文件名".into()))?;
                let uploaded_mime_type = field
                    .content_type()
                    .map(ToString::to_string)
                    .unwrap_or_else(|| "application/octet-stream".into());
                let data = field
                    .bytes()
                    .await
                    .map_err(|err| AppError::Validation(format!("读取上传文件失败：{err}")))?;
                file_name = Some(uploaded_file_name);
                mime_type = Some(uploaded_mime_type);
                bytes = Some(data.to_vec());
            }
            _ => {
                let _ = field.bytes().await;
            }
        }
    }

    Ok(Json(
        state
            .system
            .upload_media_asset(MediaUploadInput {
                category,
                display_name,
                file_name: file_name
                    .ok_or_else(|| AppError::Validation("缺少 file 字段".into()))?,
                mime_type: mime_type.unwrap_or_else(|| "application/octet-stream".into()),
                bytes: bytes.ok_or_else(|| AppError::Validation("缺少上传文件内容".into()))?,
            })
            .await?,
    ))
}

pub async fn delete_media_asset(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(id): Path<i64>,
) -> HttpResult<Json<DeleteResult>> {
    authorize(&state, &headers, "system.media-assets.delete").await?;
    Ok(Json(state.system.delete_media_asset(id).await?))
}

pub async fn storage_objects(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Query(query): Query<StorageObjectQuery>,
) -> HttpResult<Json<Vec<StorageObjectEntry>>> {
    authorize(&state, &headers, "system.storage-objects.list").await?;
    Ok(Json(state.system.storage_objects(query).await?))
}

pub async fn delete_storage_object(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(payload): Json<DeleteStorageObjectRequest>,
) -> HttpResult<Json<DeleteResult>> {
    authorize(&state, &headers, "system.storage-objects.delete").await?;
    Ok(Json(state.system.delete_storage_object(payload).await?))
}

pub async fn traffic_probe_targets(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> HttpResult<Json<Vec<TrafficProbeTargetEntry>>> {
    authorize(&state, &headers, "system.traffic-probes.targets.list").await?;
    Ok(Json(state.system.traffic_probe_targets().await?))
}

pub async fn create_traffic_probe_target(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Json(payload): Json<CreateTrafficProbeTargetRequest>,
) -> HttpResult<Json<TrafficProbeTargetEntry>> {
    authorize(&state, &headers, "system.traffic-probes.targets.create").await?;
    Ok(Json(
        state.system.create_traffic_probe_target(payload).await?,
    ))
}

pub async fn delete_traffic_probe_target(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(id): Path<i64>,
) -> HttpResult<Json<DeleteResult>> {
    authorize(&state, &headers, "system.traffic-probes.targets.delete").await?;
    Ok(Json(state.system.delete_traffic_probe_target(id).await?))
}

pub async fn run_traffic_probe(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(id): Path<i64>,
) -> HttpResult<Json<TrafficProbeResultEntry>> {
    authorize(&state, &headers, "system.traffic-probes.targets.run").await?;
    Ok(Json(state.system.run_traffic_probe(id).await?))
}

pub async fn traffic_probe_results(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Query(query): Query<TrafficProbeResultQuery>,
) -> HttpResult<Json<Vec<TrafficProbeResultEntry>>> {
    authorize(&state, &headers, "system.traffic-probes.results").await?;
    Ok(Json(state.system.traffic_probe_results(query).await?))
}

pub async fn traffic_probe_alerts(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Query(query): Query<TrafficProbeAlertQuery>,
) -> HttpResult<Json<Vec<TrafficProbeAlertEntry>>> {
    authorize(&state, &headers, "system.traffic-probes.alerts").await?;
    Ok(Json(state.system.traffic_probe_alerts(query).await?))
}

pub async fn traffic_probe_events(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Query(query): Query<TrafficProbeEventQuery>,
) -> HttpResult<impl IntoResponse> {
    authorize(&state, &headers, "system.traffic-probes.events").await?;
    let first_snapshot = state
        .system
        .traffic_probe_event_snapshot(query.clone())
        .await?;
    let heartbeat = Duration::from_secs(state.settings.scheduler.event_stream_heartbeat_seconds);
    let retry = Duration::from_millis(first_snapshot.reconnect_after_millis);
    let stream_state = Arc::clone(&state);

    let stream = async_stream::stream! {
        yield Ok::<Event, Infallible>(json_sse_event(
            "traffic_probe.alerts.snapshot",
            &first_snapshot,
            Some(retry),
        ));
        let mut interval = tokio::time::interval(heartbeat);
        interval.set_missed_tick_behavior(MissedTickBehavior::Delay);
        loop {
            interval.tick().await;
            match stream_state.system.traffic_probe_event_snapshot(query.clone()).await {
                Ok(snapshot) => {
                    yield Ok(json_sse_event("traffic_probe.alerts.snapshot", &snapshot, Some(retry)));
                }
                Err(err) => {
                    warn!(error = %err, "流量探针 SSE 事件快照刷新失败");
                    let payload = TrafficProbeEventError {
                        event_type: "traffic_probe.error".into(),
                        generated_at: Utc::now().to_rfc3339(),
                        message: "流量探针事件源暂时不可用".into(),
                    };
                    yield Ok(json_sse_event("traffic_probe.error", &payload, Some(retry)));
                }
            }
        }
    };

    Ok(Sse::new(stream).keep_alive(
        KeepAlive::new()
            .interval(heartbeat)
            .text("traffic_probe.events"),
    ))
}

pub async fn acknowledge_traffic_probe_alert(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(id): Path<i64>,
) -> HttpResult<Json<TrafficProbeAlertActionResult>> {
    authorize(&state, &headers, "system.traffic-probes.alerts.ack").await?;
    Ok(Json(
        state.system.acknowledge_traffic_probe_alert(id).await?,
    ))
}

pub async fn resolve_traffic_probe_alert(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    Path(id): Path<i64>,
) -> HttpResult<Json<TrafficProbeAlertActionResult>> {
    authorize(&state, &headers, "system.traffic-probes.alerts.resolve").await?;
    Ok(Json(state.system.resolve_traffic_probe_alert(id).await?))
}

async fn text_field(field: axum::extract::multipart::Field<'_>) -> HttpResult<String> {
    field
        .text()
        .await
        .map_err(|err| AppError::Validation(format!("读取 multipart 字段失败：{err}")).into())
}

pub async fn openapi(State(state): State<Arc<AppState>>) -> HttpResult<impl IntoResponse> {
    let yaml = route_registry::openapi_yaml(&state.settings)
        .map_err(|err| AppError::Internal(format!("OpenAPI 生成失败：{err}")))?;
    let headers = [(
        header::CONTENT_TYPE,
        HeaderValue::from_static("application/yaml"),
    )];
    Ok((headers, yaml))
}

fn json_sse_event<T: Serialize>(name: &'static str, payload: &T, retry: Option<Duration>) -> Event {
    let data = serde_json::to_string(payload)
        .unwrap_or_else(|_| r#"{"message":"事件序列化失败"}"#.to_string());
    let event = Event::default().event(name).data(data);
    match retry {
        Some(duration) => event.retry(duration),
        None => event,
    }
}

async fn authorize(
    state: &AppState,
    headers: &HeaderMap,
    route_id: &str,
) -> HttpResult<crate::domain::iam::SessionSnapshot> {
    let ctx = request_context_from_headers(headers, &state.settings);
    let permission =
        route_registry::required_permission(&state.settings, route_id).ok_or_else(|| {
            AppError::Internal(format!("路由 {route_id} 缺少权限元数据，无法执行授权"))
        })?;
    Ok(state
        .iam
        .require_permission(
            auth_credential_from_headers(headers, &state.settings),
            ctx,
            None,
            &permission,
        )
        .await?)
}

fn prometheus_scrape_token_authorized(headers: &HeaderMap, settings: &Settings) -> bool {
    let expected_hash = settings
        .observability
        .prometheus_scrape_token_hash
        .trim()
        .to_ascii_lowercase();
    if expected_hash.is_empty() {
        return false;
    }
    let Some(raw_token) = bearer_token(headers) else {
        return false;
    };
    let actual_hash = crypto::hash_secret(&raw_token, &settings.auth.session_secret);
    constant_time_eq(actual_hash.as_bytes(), expected_hash.as_bytes())
}

fn constant_time_eq(left: &[u8], right: &[u8]) -> bool {
    if left.len() != right.len() {
        return false;
    }
    let diff = left
        .iter()
        .zip(right.iter())
        .fold(0_u8, |acc, (left, right)| acc | (left ^ right));
    diff == 0
}

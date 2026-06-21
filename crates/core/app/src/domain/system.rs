use serde::{Deserialize, Serialize};
use serde_json::Value;

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct PublicSettings {
    pub product_name: String,
    pub product_code: String,
    pub default_locale: String,
    pub supported_locales: Vec<String>,
    pub auth: PublicAuthSettings,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct PublicAuthSettings {
    pub self_signup_enabled: bool,
    pub session_cookie_name: String,
    pub refresh_cookie_name: String,
    pub product_header: String,
    pub client_type_header: String,
    pub default_client_type: String,
    pub csrf_enabled: bool,
    pub csrf_cookie_name: String,
    pub csrf_header_name: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ApiCatalogEntry {
    pub id: String,
    pub method: String,
    pub path: String,
    pub tag: String,
    pub summary: String,
    pub access: String,
    pub permission: Option<String>,
    pub scope: String,
    pub product_code: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ApiCatalogGroup {
    pub tag: String,
    pub items: Vec<ApiCatalogEntry>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SystemMenuEntry {
    pub code: String,
    pub title: String,
    pub path: String,
    pub permission: Option<String>,
    pub scope: String,
    pub sort_order: i64,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct OperationRecord {
    pub id: i64,
    pub actor_user_id: Option<i64>,
    pub method: String,
    pub path: String,
    pub status: i64,
    pub created_at: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
pub struct OperationRecordQuery {
    pub method: Option<String>,
    pub path: Option<String>,
    pub status: Option<i64>,
    pub actor_user_id: Option<i64>,
    pub created_from: Option<String>,
    pub created_to: Option<String>,
    pub limit: Option<i64>,
    pub offset: Option<i64>,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
pub struct OperationRecordSummaryQuery {
    pub method: Option<String>,
    pub path: Option<String>,
    pub status: Option<i64>,
    pub actor_user_id: Option<i64>,
    pub created_from: Option<String>,
    pub created_to: Option<String>,
    pub top_limit: Option<i64>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct OperationRecordSummary {
    pub generated_at: String,
    pub total_count: i64,
    pub success_count: i64,
    pub redirect_count: i64,
    pub client_error_count: i64,
    pub server_error_count: i64,
    pub other_count: i64,
    pub top_limit: i64,
    pub by_method: Vec<OperationRecordCountBucket>,
    pub by_status_class: Vec<OperationRecordCountBucket>,
    pub top_paths: Vec<OperationRecordPathBucket>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct OperationRecordCountBucket {
    pub key: String,
    pub count: i64,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct OperationRecordPathBucket {
    pub path: String,
    pub count: i64,
    pub error_count: i64,
    pub last_seen_at: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct OperationRecordRetentionReport {
    pub retention_days: i64,
    pub cutoff: String,
    pub prune_batch_size: i64,
    pub deleted: i64,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ServerStatus {
    pub source: String,
    pub collected_at: String,
    pub started_at: String,
    pub uptime_seconds: i64,
    pub process_id: u32,
    pub os: String,
    pub arch: String,
    pub available_parallelism: usize,
    pub product_code: String,
    pub version: String,
    pub database_driver: String,
    pub metrics: ServerResourceMetrics,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ServerResourceMetrics {
    pub source: String,
    pub cpu_usage_percent: f32,
    pub process_cpu_usage_percent: f32,
    pub total_memory_bytes: u64,
    pub used_memory_bytes: u64,
    pub available_memory_bytes: u64,
    pub process_memory_bytes: u64,
    pub process_virtual_memory_bytes: u64,
    pub total_swap_bytes: u64,
    pub used_swap_bytes: u64,
    pub total_disk_bytes: u64,
    pub used_disk_bytes: u64,
    pub available_disk_bytes: u64,
    pub disk_count: u64,
    pub network_interface_count: u64,
    pub network_received_bytes: u64,
    pub network_transmitted_bytes: u64,
    pub system_uptime_seconds: u64,
    pub system_boot_time_seconds: u64,
    pub load_average_one: f64,
    pub load_average_five: f64,
    pub load_average_fifteen: f64,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SystemConfigEntry {
    pub key: String,
    pub value: Value,
    pub updated_at: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct UpsertSystemConfigRequest {
    pub value: Value,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SystemDictionaryEntry {
    pub id: i64,
    pub code: String,
    pub name: String,
    pub created_at: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct UpsertSystemDictionaryRequest {
    pub name: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SystemParameterEntry {
    pub id: i64,
    pub key: String,
    pub name: String,
    pub value: String,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct UpsertSystemParameterRequest {
    pub name: String,
    pub value: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct DeleteResult {
    pub deleted: bool,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct VersionPackageEntry {
    pub id: i64,
    pub version_name: String,
    pub version_code: String,
    pub manifest: Value,
    pub status: String,
    pub created_at: String,
    pub published_at: Option<String>,
    pub retired_at: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct CreateVersionPackageRequest {
    pub version_name: String,
    pub version_code: String,
    pub manifest: Value,
}

#[derive(Clone, Debug, Deserialize)]
pub struct VersionPackageActionRequest {
    pub reason: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct VersionPackageActionResult {
    pub event_id: i64,
    pub previous_active_id: Option<i64>,
    pub package: VersionPackageEntry,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct VersionReleaseEventEntry {
    pub id: i64,
    pub package_id: i64,
    pub previous_active_id: Option<i64>,
    pub action: String,
    pub status: String,
    pub reason: Option<String>,
    pub created_at: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct MediaAssetEntry {
    pub id: i64,
    pub category: Option<String>,
    pub display_name: String,
    pub storage_key: String,
    pub mime_type: String,
    pub size_bytes: i64,
    pub created_at: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct CreateMediaAssetRequest {
    pub category: Option<String>,
    pub display_name: String,
    pub storage_key: String,
    pub mime_type: String,
    pub size_bytes: i64,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct StorageObjectEntry {
    pub storage_key: String,
    pub size_bytes: i64,
    pub updated_at: Option<String>,
    pub e_tag: Option<String>,
}

#[derive(Clone, Debug, Deserialize)]
pub struct StorageObjectQuery {
    pub prefix: Option<String>,
    pub limit: Option<usize>,
}

#[derive(Clone, Debug, Deserialize)]
pub struct DeleteStorageObjectRequest {
    pub storage_key: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct TrafficProbeTargetEntry {
    pub id: i64,
    pub name: String,
    pub url: String,
    pub expected_status: i64,
    pub status: String,
    pub created_at: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct CreateTrafficProbeTargetRequest {
    pub name: String,
    pub url: String,
    pub expected_status: Option<i64>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct TrafficProbeResultEntry {
    pub id: i64,
    pub target_id: i64,
    pub status: String,
    pub detail: Value,
    pub probed_at: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct TrafficProbeAlertEntry {
    pub id: i64,
    pub target_id: i64,
    pub result_id: i64,
    pub severity: String,
    pub status: String,
    pub reason: String,
    pub detail: Value,
    pub opened_at: String,
    pub acknowledged_at: Option<String>,
    pub resolved_at: Option<String>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct TrafficProbeAlertActionResult {
    pub updated: bool,
}

#[derive(Clone, Debug, Deserialize)]
pub struct TrafficProbeAlertQuery {
    pub target_id: Option<i64>,
    pub status: Option<String>,
    pub limit: Option<i64>,
}

#[derive(Clone, Debug, Deserialize)]
pub struct TrafficProbeEventQuery {
    pub target_id: Option<i64>,
    pub status: Option<String>,
    pub limit: Option<i64>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct TrafficProbeEventSnapshot {
    pub event_type: String,
    pub generated_at: String,
    pub reconnect_after_millis: u64,
    pub alerts: Vec<TrafficProbeAlertEntry>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct TrafficProbeEventError {
    pub event_type: String,
    pub generated_at: String,
    pub message: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct TrafficProbeSweepReport {
    pub scanned_targets: usize,
    pub recorded_results: usize,
    pub failed_targets: usize,
}

#[derive(Clone, Debug, Deserialize)]
pub struct TrafficProbeResultQuery {
    pub target_id: Option<i64>,
    pub limit: Option<i64>,
}

#[derive(Clone, Debug)]
pub struct TrafficProbeObservation {
    pub status: String,
    pub detail: Value,
}

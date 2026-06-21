use async_trait::async_trait;

use crate::app::AppResult;
use crate::domain::iam::{
    APITokenAuthRecord, APITokenSummary, EmailVerificationRecord, InvitationAcceptRecord,
    InvitationSummary, MfaFactorSummary, MfaRecoveryCodeSummary, Organization, OrganizationSummary,
    OrganizationUserSummary, PasswordResetRecord, PermissionSummary, RoleSummary, SessionRecord,
    StoredMfaFactor, StoredUser, User,
};
use crate::domain::notification::{
    NotificationDeadLetterRecord, NotificationFailureDisposition, NotificationOutboxItem,
};
use crate::domain::setup::{SetupRun, SetupStepLog};
use crate::domain::system::{
    ApiCatalogEntry, MediaAssetEntry, OperationRecord, OperationRecordSummary, SystemConfigEntry,
    SystemDictionaryEntry, SystemMenuEntry, SystemParameterEntry, TrafficProbeAlertEntry,
    TrafficProbeResultEntry, TrafficProbeTargetEntry, VersionPackageActionResult,
    VersionPackageEntry, VersionReleaseEventEntry,
};

#[async_trait]
pub trait SetupRepository: Send + Sync {
    async fn setup_completed(&self) -> AppResult<bool>;
    async fn complete_setup(&self, run_id: Option<&str>) -> AppResult<bool>;
    async fn create_setup_run(&self, id: &str, reason: Option<&str>) -> AppResult<SetupRun>;
    async fn list_setup_runs(&self, limit: i64) -> AppResult<Vec<SetupRun>>;
    async fn append_setup_log(
        &self,
        run_id: &str,
        step_key: &str,
        status: &str,
        message: &str,
    ) -> AppResult<()>;
    async fn list_setup_logs(&self, run_id: &str) -> AppResult<Vec<SetupStepLog>>;
}

#[async_trait]
pub trait IamRepository: Send + Sync {
    async fn has_any_user(&self) -> AppResult<bool>;
    async fn create_initial_admin(
        &self,
        input: CreateInitialAdminRecord,
    ) -> AppResult<(User, Organization)>;
    async fn create_registration_with_email_verification(
        &self,
        input: CreateRegistrationRecord,
        notification: CreateNotificationOutboxRecord,
    ) -> AppResult<(User, Organization)>;
    async fn find_user_by_identifier(&self, identifier: &str) -> AppResult<Option<StoredUser>>;
    async fn primary_organization_for_user(&self, user_id: i64) -> AppResult<Option<Organization>>;
    async fn list_organizations(&self) -> AppResult<Vec<OrganizationSummary>>;
    async fn list_org_users(&self, org_id: i64) -> AppResult<Vec<OrganizationUserSummary>>;
    async fn update_org_user(
        &self,
        input: UpdateOrgUserRecord,
    ) -> AppResult<OrganizationUserSummary>;
    async fn list_org_roles(&self, org_id: i64) -> AppResult<Vec<RoleSummary>>;
    async fn create_org_role(&self, input: CreateRoleRecord) -> AppResult<RoleSummary>;
    async fn update_org_role(&self, input: UpdateRoleRecord) -> AppResult<RoleSummary>;
    async fn delete_org_role(&self, org_id: i64, role_id: i64) -> AppResult<bool>;
    async fn list_permissions(&self, product_code: &str) -> AppResult<Vec<PermissionSummary>>;
    async fn create_session(&self, input: CreateSessionRecord) -> AppResult<()>;
    async fn find_session_by_hash(&self, token_hash: &str) -> AppResult<Option<SessionRecord>>;
    async fn find_session_by_refresh_hash(
        &self,
        refresh_hash: &str,
    ) -> AppResult<Option<SessionRecord>>;
    async fn rotate_session_tokens(
        &self,
        session_id: &str,
        current_refresh_hash: &str,
        new_session_hash: String,
        new_refresh_hash: String,
        expires_at: String,
        refresh_expires_at: String,
    ) -> AppResult<bool>;
    async fn revoke_session_by_hash(&self, token_hash: &str) -> AppResult<()>;
    async fn revoke_session_by_refresh_hash(&self, refresh_hash: &str) -> AppResult<()>;
    async fn list_permissions_for_user(
        &self,
        user_id: i64,
        org_id: i64,
        product_code: &str,
        include_platform: bool,
    ) -> AppResult<Vec<String>>;
    async fn create_api_token(&self, input: CreateAPITokenRecord) -> AppResult<APITokenSummary>;
    async fn find_api_token_by_hash(
        &self,
        token_hash: &str,
    ) -> AppResult<Option<APITokenAuthRecord>>;
    async fn list_api_tokens(&self, org_id: i64) -> AppResult<Vec<APITokenSummary>>;
    async fn revoke_api_token(&self, org_id: i64, token_id: i64) -> AppResult<bool>;
    async fn org_role_exists(&self, org_id: i64, role_code: &str) -> AppResult<bool>;
    async fn create_invitation_with_notification(
        &self,
        input: CreateInvitationRecord,
        notification: CreateNotificationOutboxRecord,
    ) -> AppResult<InvitationSummary>;
    async fn list_invitations(&self, org_id: i64) -> AppResult<Vec<InvitationSummary>>;
    async fn revoke_invitation(&self, org_id: i64, invitation_id: i64) -> AppResult<bool>;
    async fn find_invitation_by_hash(
        &self,
        token_hash: &str,
    ) -> AppResult<Option<InvitationAcceptRecord>>;
    async fn accept_invitation_with_user(
        &self,
        input: AcceptInvitationRecord,
    ) -> AppResult<(User, Organization)>;
    async fn create_password_reset_with_notification(
        &self,
        input: CreatePasswordResetRecord,
        notification: CreateNotificationOutboxRecord,
    ) -> AppResult<()>;
    async fn find_password_reset_by_hash(
        &self,
        token_hash: &str,
    ) -> AppResult<Option<PasswordResetRecord>>;
    async fn reset_password_with_token(
        &self,
        reset_id: i64,
        user_id: i64,
        password_hash: String,
    ) -> AppResult<bool>;
    async fn create_email_verification_with_notification(
        &self,
        input: CreateEmailVerificationRecord,
        notification: CreateNotificationOutboxRecord,
    ) -> AppResult<()>;
    async fn find_email_verification_by_hash(
        &self,
        token_hash: &str,
    ) -> AppResult<Option<EmailVerificationRecord>>;
    async fn confirm_email_verification(
        &self,
        verification_id: i64,
        user_id: i64,
    ) -> AppResult<bool>;
    async fn create_pending_mfa_factor(
        &self,
        input: CreateMfaFactorRecord,
    ) -> AppResult<MfaFactorSummary>;
    async fn list_mfa_factors(&self, user_id: i64) -> AppResult<Vec<MfaFactorSummary>>;
    async fn find_pending_mfa_factor(&self, user_id: i64) -> AppResult<Option<StoredMfaFactor>>;
    async fn find_verified_mfa_factor(&self, user_id: i64) -> AppResult<Option<StoredMfaFactor>>;
    async fn activate_mfa_factor_with_recovery_codes(
        &self,
        user_id: i64,
        factor_id: i64,
        recovery_codes: Vec<CreateMfaRecoveryCodeRecord>,
    ) -> AppResult<bool>;
    async fn revoke_mfa_factor(&self, user_id: i64, factor_id: i64) -> AppResult<bool>;
    async fn list_mfa_recovery_codes(&self, user_id: i64)
    -> AppResult<Vec<MfaRecoveryCodeSummary>>;
    async fn replace_mfa_recovery_codes(
        &self,
        user_id: i64,
        recovery_codes: Vec<CreateMfaRecoveryCodeRecord>,
    ) -> AppResult<Vec<MfaRecoveryCodeSummary>>;
    async fn consume_mfa_recovery_code(&self, user_id: i64, code_hash: &str) -> AppResult<bool>;
    async fn record_audit(&self, input: CreateAuditLogRecord) -> AppResult<()>;
}

#[async_trait]
pub trait SystemRepository: Send + Sync {
    async fn sync_api_catalog(&self, entries: &[ApiCatalogEntry]) -> AppResult<()>;
    async fn sync_system_menus(&self, menus: &[SystemMenuEntry]) -> AppResult<()>;
    async fn list_api_catalog(&self) -> AppResult<Vec<ApiCatalogEntry>>;
    async fn list_system_menus(&self) -> AppResult<Vec<SystemMenuEntry>>;
    async fn create_operation_record(&self, input: CreateOperationRecord) -> AppResult<()>;
    async fn list_operation_records(
        &self,
        query: OperationRecordListQuery,
    ) -> AppResult<Vec<OperationRecord>>;
    async fn summarize_operation_records(
        &self,
        query: OperationRecordSummaryFilter,
    ) -> AppResult<OperationRecordSummary>;
    async fn prune_operation_records(&self, cutoff: &str, limit: i64) -> AppResult<i64>;
    async fn list_system_configs(&self) -> AppResult<Vec<SystemConfigEntry>>;
    async fn upsert_system_config(
        &self,
        input: UpsertSystemConfigRecord,
    ) -> AppResult<SystemConfigEntry>;
    async fn delete_system_config(&self, key: &str) -> AppResult<bool>;
    async fn list_system_dictionaries(&self) -> AppResult<Vec<SystemDictionaryEntry>>;
    async fn upsert_system_dictionary(
        &self,
        input: UpsertSystemDictionaryRecord,
    ) -> AppResult<SystemDictionaryEntry>;
    async fn delete_system_dictionary(&self, code: &str) -> AppResult<bool>;
    async fn list_system_parameters(&self) -> AppResult<Vec<SystemParameterEntry>>;
    async fn upsert_system_parameter(
        &self,
        input: UpsertSystemParameterRecord,
    ) -> AppResult<SystemParameterEntry>;
    async fn delete_system_parameter(&self, key: &str) -> AppResult<bool>;
    async fn create_version_package(
        &self,
        input: CreateVersionPackageRecord,
    ) -> AppResult<VersionPackageEntry>;
    async fn list_version_packages(&self) -> AppResult<Vec<VersionPackageEntry>>;
    async fn publish_version_package(
        &self,
        input: VersionPackageActionRecord,
    ) -> AppResult<VersionPackageActionResult>;
    async fn rollback_version_package(
        &self,
        input: VersionPackageActionRecord,
    ) -> AppResult<VersionPackageActionResult>;
    async fn list_version_release_events(&self) -> AppResult<Vec<VersionReleaseEventEntry>>;
    async fn delete_version_package(&self, id: i64) -> AppResult<bool>;
    async fn create_media_asset(&self, input: CreateMediaAssetRecord)
    -> AppResult<MediaAssetEntry>;
    async fn list_media_assets(&self) -> AppResult<Vec<MediaAssetEntry>>;
    async fn delete_media_asset(&self, id: i64) -> AppResult<bool>;
    async fn create_traffic_probe_target(
        &self,
        input: CreateTrafficProbeTargetRecord,
    ) -> AppResult<TrafficProbeTargetEntry>;
    async fn list_traffic_probe_targets(&self) -> AppResult<Vec<TrafficProbeTargetEntry>>;
    async fn find_traffic_probe_target(
        &self,
        id: i64,
    ) -> AppResult<Option<TrafficProbeTargetEntry>>;
    async fn delete_traffic_probe_target(&self, id: i64) -> AppResult<bool>;
    async fn create_traffic_probe_result(
        &self,
        input: CreateTrafficProbeResultRecord,
    ) -> AppResult<TrafficProbeResultEntry>;
    async fn list_traffic_probe_results(
        &self,
        target_id: Option<i64>,
        limit: i64,
    ) -> AppResult<Vec<TrafficProbeResultEntry>>;
    async fn create_traffic_probe_alert(
        &self,
        input: CreateTrafficProbeAlertRecord,
    ) -> AppResult<TrafficProbeAlertEntry>;
    async fn list_traffic_probe_alerts(
        &self,
        target_id: Option<i64>,
        status: Option<String>,
        limit: i64,
    ) -> AppResult<Vec<TrafficProbeAlertEntry>>;
    async fn acknowledge_traffic_probe_alert(&self, id: i64) -> AppResult<bool>;
    async fn resolve_traffic_probe_alert(&self, id: i64) -> AppResult<bool>;
    async fn resolve_traffic_probe_alerts_for_target(&self, target_id: i64) -> AppResult<i64>;
}

#[async_trait]
pub trait NotificationRepository: Send + Sync {
    async fn claim_due_notifications(
        &self,
        limit: i64,
        lock_ttl_seconds: i64,
    ) -> AppResult<Vec<NotificationOutboxItem>>;
    async fn mark_notification_delivered(&self, notification_id: i64) -> AppResult<bool>;
    async fn mark_notification_failed(
        &self,
        notification_id: i64,
        failure_reason: &str,
        retry_backoff_seconds: i64,
        max_attempts: i64,
    ) -> AppResult<NotificationFailureDisposition>;
    async fn list_failed_notifications(
        &self,
        limit: i64,
    ) -> AppResult<Vec<NotificationDeadLetterRecord>>;
    async fn requeue_failed_notification(&self, notification_id: i64) -> AppResult<bool>;
}

pub struct CreateInitialAdminRecord {
    pub email: String,
    pub password_hash: String,
    pub display_name: String,
    pub organization_code: String,
    pub organization_name: String,
    pub product_code: String,
}

pub struct CreateRegistrationRecord {
    pub email: String,
    pub password_hash: String,
    pub display_name: String,
    pub organization_code: String,
    pub organization_name: String,
    pub product_code: String,
    pub email_verification_token_hash: String,
    pub email_verification_expires_at: String,
}

pub struct CreateSessionRecord {
    pub id: String,
    pub token_hash: String,
    pub refresh_token_hash: String,
    pub user_id: i64,
    pub org_id: i64,
    pub product_code: String,
    pub client_type: String,
    pub expires_at: String,
    pub refresh_expires_at: String,
}

pub struct CreateRoleRecord {
    pub org_id: i64,
    pub code: String,
    pub name: String,
    pub product_code: String,
    pub permission_codes: Vec<String>,
}

pub struct UpdateRoleRecord {
    pub org_id: i64,
    pub role_id: i64,
    pub name: String,
    pub product_code: String,
    pub permission_codes: Vec<String>,
}

pub struct UpdateOrgUserRecord {
    pub org_id: i64,
    pub user_id: i64,
    pub display_name: String,
    pub status: String,
    pub role_codes: Vec<String>,
}

pub struct CreateAPITokenRecord {
    pub org_id: i64,
    pub user_id: i64,
    pub token_hash: String,
    pub token_prefix: String,
    pub expires_at: Option<String>,
}

pub struct CreateInvitationRecord {
    pub org_id: i64,
    pub email: String,
    pub role_code: String,
    pub token_hash: String,
    pub expires_at: String,
}

pub struct AcceptInvitationRecord {
    pub invitation_id: i64,
    pub org_id: i64,
    pub email: String,
    pub role_code: String,
    pub display_name: String,
    pub password_hash: String,
}

pub struct CreatePasswordResetRecord {
    pub user_id: i64,
    pub token_hash: String,
    pub expires_at: String,
}

pub struct CreateEmailVerificationRecord {
    pub user_id: i64,
    pub email: String,
    pub token_hash: String,
    pub expires_at: String,
}

pub struct CreateNotificationOutboxRecord {
    pub org_id: Option<i64>,
    pub user_id: Option<i64>,
    pub product_code: String,
    pub channel: String,
    pub template_code: String,
    pub recipient: String,
    pub related_kind: String,
    pub payload_json: String,
    pub available_at: String,
    pub delivery_secret_ciphertext: Option<String>,
}

pub struct CreateAuditLogRecord {
    pub org_id: Option<i64>,
    pub user_id: Option<i64>,
    pub action: String,
    pub scope: String,
    pub product_code: String,
    pub detail: String,
}

pub struct CreateMfaFactorRecord {
    pub user_id: i64,
    pub kind: String,
    pub secret_ciphertext: String,
}

pub struct CreateMfaRecoveryCodeRecord {
    pub code_hash: String,
    pub code_prefix: String,
}

pub struct CreateOperationRecord {
    pub actor_user_id: Option<i64>,
    pub method: String,
    pub path: String,
    pub status: i64,
}

#[derive(Clone, Debug)]
pub struct OperationRecordListQuery {
    pub method: Option<String>,
    pub path: Option<String>,
    pub status: Option<i64>,
    pub actor_user_id: Option<i64>,
    pub created_from: Option<String>,
    pub created_to: Option<String>,
    pub limit: i64,
    pub offset: i64,
}

#[derive(Clone, Debug)]
pub struct OperationRecordSummaryFilter {
    pub method: Option<String>,
    pub path: Option<String>,
    pub status: Option<i64>,
    pub actor_user_id: Option<i64>,
    pub created_from: Option<String>,
    pub created_to: Option<String>,
    pub top_limit: i64,
}

pub struct UpsertSystemConfigRecord {
    pub key: String,
    pub value_json: String,
}

pub struct UpsertSystemDictionaryRecord {
    pub code: String,
    pub name: String,
}

pub struct UpsertSystemParameterRecord {
    pub key: String,
    pub name: String,
    pub value: String,
}

pub struct CreateVersionPackageRecord {
    pub version_name: String,
    pub version_code: String,
    pub manifest_json: String,
}

pub struct VersionPackageActionRecord {
    pub id: i64,
    pub reason: Option<String>,
}

pub struct CreateMediaAssetRecord {
    pub category: Option<String>,
    pub display_name: String,
    pub storage_key: String,
    pub mime_type: String,
    pub size_bytes: i64,
}

pub struct CreateTrafficProbeTargetRecord {
    pub name: String,
    pub url: String,
    pub expected_status: i64,
}

pub struct CreateTrafficProbeResultRecord {
    pub target_id: i64,
    pub status: String,
    pub detail_json: String,
}

pub struct CreateTrafficProbeAlertRecord {
    pub target_id: i64,
    pub result_id: i64,
    pub severity: String,
    pub reason: String,
    pub detail_json: String,
}

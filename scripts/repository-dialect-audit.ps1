param(
    [string]$Root = (Join-Path $PSScriptRoot "..")
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$rootPath = (Resolve-Path -LiteralPath $Root).Path
$repositoryPath = Join-Path $rootPath "crates/core/app/src/infrastructure/sqlite_repository.rs"
$databasePath = Join-Path $rootPath "crates/core/app/src/infrastructure/database.rs"
$sqlTemplatesPath = Join-Path $rootPath "crates/core/app/src/infrastructure/sql_templates.rs"
$configPath = Join-Path $rootPath "crates/core/config/src/lib.rs"
$docsPath = Join-Path $rootPath "docs/deployment/database-runtime-matrix.md"
$failures = New-Object System.Collections.Generic.List[string]

function Add-Failure {
    param([string]$Message)
    $failures.Add($Message) | Out-Null
}

function Read-Text {
    param([string]$Path)
    return [System.IO.File]::ReadAllText($Path, [System.Text.UTF8Encoding]::new($false, $true))
}

function Count-Pattern {
    param(
        [string]$Text,
        [string]$Pattern
    )
    return ([regex]::Matches($Text, $Pattern, [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)).Count
}

function Remove-TestModule {
    param([string]$Text)
    $testModuleStart = $Text.IndexOf("#[cfg(test)]")
    if ($testModuleStart -lt 0) {
        return $Text
    }
    return $Text.Substring(0, $testModuleStart)
}

function Count-DirectSqliteBindPlaceholders {
    param([string]$Text)
    $pattern = '(?s)sqlx::query(?:_scalar|_as)?(?:::<[^>]+>)?\s*\(\s*"([^"]*\?[^"]*)"'
    return ([regex]::Matches($Text, $pattern, [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)).Count
}

function Assert-MinCount {
    param(
        [hashtable]$Counts,
        [string]$Key,
        [int]$Minimum
    )
    if ($Counts[$Key] -lt $Minimum) {
        Add-Failure "repository dialect audit expected at least $Minimum '$Key' blockers, got $($Counts[$Key])"
    }
}

if (-not (Test-Path -LiteralPath $repositoryPath -PathType Leaf)) {
    throw "repository source not found: $repositoryPath"
}
if (-not (Test-Path -LiteralPath $databasePath -PathType Leaf)) {
    throw "database runtime source not found: $databasePath"
}
if (-not (Test-Path -LiteralPath $sqlTemplatesPath -PathType Leaf)) {
    throw "repository SQL template source not found: $sqlTemplatesPath"
}
if (-not (Test-Path -LiteralPath $configPath -PathType Leaf)) {
    throw "config source not found: $configPath"
}
if (-not (Test-Path -LiteralPath $docsPath -PathType Leaf)) {
    throw "database runtime docs not found: $docsPath"
}

$repoText = Read-Text $repositoryPath
$productionRepoText = Remove-TestModule $repoText
$databaseText = Read-Text $databasePath
$sqlTemplatesText = Read-Text $sqlTemplatesPath
$configText = Read-Text $configPath
$docsText = Read-Text $docsPath

$counts = @{
    "sqlite_pool_type" = Count-Pattern $productionRepoText "\bPool\s*<\s*Sqlite\s*>"
    "sqlite_connection_type" = Count-Pattern $productionRepoText "\bSqliteConnection\b|sqlx::SqliteConnection"
    "sqlite_transaction_type" = Count-Pattern $productionRepoText "sqlx::Transaction\s*<\s*'_,\s*Sqlite\s*>"
    "sqlite_dialect_constant" = Count-Pattern $productionRepoText "const\s+DIALECT\s*:\s*SqlDialect\s*=\s*SqlDialect::Sqlite"
    "sqlite_row_type" = Count-Pattern $productionRepoText "\bSqliteRow\b|sqlx::sqlite::SqliteRow"
    "sqlite_last_insert_id" = Count-Pattern $productionRepoText "\blast_insert_rowid\s*\("
    "insert_returning_id" = Count-Pattern $productionRepoText "\breturning\s+id\b"
    "scattered_setup_state_sql" = Count-Pattern $productionRepoText "\b(from|insert\s+into|update)\s+setup_state\b"
    "scattered_setup_run_sql" = Count-Pattern $productionRepoText "\b(from|insert\s+into|update)\s+setup_runs\b|\b(from|insert\s+into|update)\s+setup_step_logs\b"
    "scattered_session_sql" = Count-Pattern $productionRepoText "\b(insert\s+into|update|from)\s+iam_sessions\b"
    "scattered_api_token_read_sql" = Count-Pattern $productionRepoText "\b(from|update)\s+iam_api_tokens\b"
    "scattered_system_settings_sql" = Count-Pattern $productionRepoText "\b(from|update|delete\s+from)\s+system_(configs|dictionaries|parameters)\b"
    "scattered_system_registry_sql" = Count-Pattern $productionRepoText "\b(insert\s+into|from|update)\s+system_(apis|menus|operation_records)\b"
    "scattered_traffic_probe_state_sql" = Count-Pattern $productionRepoText "\b(from|update)\s+system_traffic_probe_targets\b|\bupdate\s+system_traffic_probe_alerts\b"
    "scattered_media_version_read_delete_sql" = Count-Pattern $productionRepoText "\bfrom\s+system_version_release_events\b|\bupdate\s+system_version_packages\s+set\s+deleted_at\b|\bselect\s+status\s+from\s+system_version_packages\b|\bfrom\s+system_media_assets\b|\bupdate\s+system_media_assets\b"
    "scattered_version_package_read_update_sql" = Count-Pattern $productionRepoText "\bfrom\s+system_version_packages\b|\bupdate\s+system_version_packages\b"
    "scattered_system_insert_sql" = Count-Pattern $productionRepoText "\binsert\s+into\s+system_(version_packages|media_assets|traffic_probe_targets|traffic_probe_results|traffic_probe_alerts)\b"
    "scattered_identity_insert_sql" = Count-Pattern $productionRepoText "\binsert\s+into\s+iam_(organizations|users)\b"
    "scattered_identity_count_sql" = Count-Pattern $productionRepoText "query_scalar::<_,\s*i64>\(\s*`"select\s+count\(\*\)\s+from\s+iam_users`"\s*\)|select\s+count\(\*\)\s+from\s+iam_users\s+where\s+lower\(email\)\s*=\s*lower\(\?\)|select\s+count\(\*\)\s+from\s+iam_organizations\s+where\s+code\s*=\s*\?"
    "scattered_membership_insert_sql" = Count-Pattern $productionRepoText "\binsert\s+into\s+iam_memberships\b"
    "scattered_audit_insert_sql" = Count-Pattern $productionRepoText "\binsert\s+into\s+iam_audit_logs\b"
    "scattered_org_user_management_sql" = Count-Pattern $productionRepoText "group_concat\s*\(\s*distinct\s+m\.role_code\s*\)|count\s*\(\s*distinct\s+m\.user_id\s*\)|update\s+iam_users\s+set\s+display_name\s*=|delete\s+from\s+iam_memberships\s+where\s+org_id\b|order\s+by\s+o\.id\s+asc"
    "scattered_iam_permission_list_sql" = Count-Pattern $productionRepoText "\bfrom\s+iam_permissions\s+where\s+product_code\b|order\s+by\s+scope\s+asc,\s+code\s+asc"
    "scattered_role_permission_state_sql" = Count-Pattern $productionRepoText "\b(from|delete\s+from)\s+iam_role_permissions\b|\bjoin\s+iam_permissions\b"
    "scattered_iam_role_state_sql" = Count-Pattern $productionRepoText "\b(from|update|delete\s+from)\s+iam_roles\b"
    "scattered_iam_pending_state_sql" = Count-Pattern $productionRepoText "select\s+id,\s+org_id,\s+email,\s+role_code,\s+status,\s+expires_at,\s+accepted_at,\s+revoked_at\s+from\s+iam_invitations|update\s+iam_invitations\s+set|select\s+id,\s+user_id,\s+status,\s+expires_at,\s+used_at\s+from\s+iam_password_resets|update\s+iam_password_resets\s+set|select\s+id,\s+user_id,\s+status,\s+expires_at,\s+verified_at\s+from\s+iam_email_verifications|update\s+iam_email_verifications\s+set"
    "scattered_user_secret_state_sql" = Count-Pattern $productionRepoText "update\s+iam_users\s+set\s+password_hash\s*=|update\s+iam_users\s+set\s+email_verified_at\s*="
    "scattered_iam_workflow_insert_sql" = Count-Pattern $productionRepoText "\binsert\s+into\s+iam_(roles|api_tokens|invitations|password_resets|email_verifications|mfa_factors)\b"
    "scattered_notification_insert_sql" = Count-Pattern $productionRepoText "\binsert\s+into\s+iam_notification_(outbox|delivery_secrets)\b"
    "scattered_notification_state_sql" = Count-Pattern $productionRepoText "\bupdate\s+iam_notification_(outbox|delivery_secrets)\b|select\s+attempt_count\s+from\s+iam_notification_outbox"
    "scattered_mfa_recovery_insert_sql" = Count-Pattern $productionRepoText "\binsert\s+into\s+iam_mfa_recovery_codes\b"
    "scattered_mfa_factor_state_sql" = Count-Pattern $productionRepoText "\b(from|update)\s+iam_mfa_factors\b"
    "scattered_mfa_recovery_state_sql" = Count-Pattern $productionRepoText "\b(from|update)\s+iam_mfa_recovery_codes\b"
    "scattered_version_release_event_insert_sql" = Count-Pattern $productionRepoText "\binsert\s+into\s+system_version_release_events\b"
    "sqlite_placeholder" = Count-DirectSqliteBindPlaceholders $productionRepoText
    "rust_question_mark_operator" = Count-Pattern $productionRepoText "\?"
    "sqlite_upsert_syntax" = Count-Pattern $productionRepoText "\bon\s+conflict\s*\("
    "sqlite_limit_bind" = Count-Pattern $productionRepoText "\blimit\s+\?"
}

Assert-MinCount $counts "sqlite_pool_type" 1
Assert-MinCount $counts "sqlite_dialect_constant" 1

if ($counts["sqlite_placeholder"] -ne 0) {
    Add-Failure "Direct SQLite '?' bind placeholders must not appear in production repository SQL literals, got $($counts["sqlite_placeholder"]) occurrences"
}

if ($counts["sqlite_row_type"] -ne 0) {
    Add-Failure "SQLite row type must not leak into repository mapping helpers, got $($counts["sqlite_row_type"]) occurrences"
}

if ($counts["sqlite_last_insert_id"] -ne 0) {
    Add-Failure "SQLite last_insert_rowid() must not be used for repository insert IDs, got $($counts["sqlite_last_insert_id"]) occurrences"
}

if ($counts["scattered_setup_state_sql"] -ne 0) {
    Add-Failure "Setup state SQL must be centralized in sql_templates.rs, got $($counts["scattered_setup_state_sql"]) scattered repository occurrences"
}

if ($counts["scattered_setup_run_sql"] -ne 0) {
    Add-Failure "Setup run and step log SQL must be centralized in sql_templates.rs, got $($counts["scattered_setup_run_sql"]) scattered repository occurrences"
}

if ($counts["scattered_session_sql"] -ne 0) {
    Add-Failure "session SQL must be centralized in sql_templates.rs, got $($counts["scattered_session_sql"]) scattered repository occurrences"
}

if ($counts["scattered_api_token_read_sql"] -ne 0) {
    Add-Failure "API token read/revoke SQL must be centralized in sql_templates.rs, got $($counts["scattered_api_token_read_sql"]) scattered repository occurrences"
}

if ($counts["scattered_system_settings_sql"] -ne 0) {
    Add-Failure "System settings SQL must be centralized in sql_templates.rs, got $($counts["scattered_system_settings_sql"]) scattered repository occurrences"
}

if ($counts["scattered_system_registry_sql"] -ne 0) {
    Add-Failure "System registry and operation record SQL must be centralized in sql_templates.rs, got $($counts["scattered_system_registry_sql"]) scattered repository occurrences"
}

if ($counts["scattered_traffic_probe_state_sql"] -ne 0) {
    Add-Failure "Traffic probe target and alert state SQL must be centralized in sql_templates.rs, got $($counts["scattered_traffic_probe_state_sql"]) scattered repository occurrences"
}

if ($counts["scattered_media_version_read_delete_sql"] -ne 0) {
    Add-Failure "Media and version package read/delete SQL must be centralized in sql_templates.rs, got $($counts["scattered_media_version_read_delete_sql"]) scattered repository occurrences"
}

if ($counts["scattered_version_package_read_update_sql"] -ne 0) {
    Add-Failure "Version package read/update SQL must be centralized in sql_templates.rs, got $($counts["scattered_version_package_read_update_sql"]) scattered repository occurrences"
}

if ($counts["scattered_system_insert_sql"] -ne 0) {
    Add-Failure "System version/media/traffic insert SQL must be centralized in sql_templates.rs, got $($counts["scattered_system_insert_sql"]) scattered repository occurrences"
}

if ($counts["scattered_identity_insert_sql"] -ne 0) {
    Add-Failure "IAM organization/user insert SQL must be centralized in sql_templates.rs, got $($counts["scattered_identity_insert_sql"]) scattered repository occurrences"
}

if ($counts["scattered_identity_count_sql"] -ne 0) {
    Add-Failure "IAM initialization and registration count SQL must be centralized in sql_templates.rs, got $($counts["scattered_identity_count_sql"]) scattered repository occurrences"
}

if ($counts["scattered_membership_insert_sql"] -ne 0) {
    Add-Failure "IAM membership insert SQL must be centralized in sql_templates.rs, got $($counts["scattered_membership_insert_sql"]) scattered repository occurrences"
}

if ($counts["scattered_audit_insert_sql"] -ne 0) {
    Add-Failure "IAM audit log insert SQL must be centralized in sql_templates.rs, got $($counts["scattered_audit_insert_sql"]) scattered repository occurrences"
}

if ($counts["scattered_org_user_management_sql"] -ne 0) {
    Add-Failure "Organization user management SQL must be centralized in sql_templates.rs, got $($counts["scattered_org_user_management_sql"]) scattered repository occurrences"
}

if ($counts["scattered_iam_permission_list_sql"] -ne 0) {
    Add-Failure "IAM permission list SQL must be centralized in sql_templates.rs, got $($counts["scattered_iam_permission_list_sql"]) scattered repository occurrences"
}

if ($counts["scattered_role_permission_state_sql"] -ne 0) {
    Add-Failure "Role permission relation SQL must be centralized in sql_templates.rs, got $($counts["scattered_role_permission_state_sql"]) scattered repository occurrences"
}

if ($counts["scattered_iam_role_state_sql"] -ne 0) {
    Add-Failure "IAM role read/update/delete SQL must be centralized in sql_templates.rs, got $($counts["scattered_iam_role_state_sql"]) scattered repository occurrences"
}

if ($counts["scattered_iam_pending_state_sql"] -ne 0) {
    Add-Failure "IAM pending workflow read/update SQL must be centralized in sql_templates.rs, got $($counts["scattered_iam_pending_state_sql"]) scattered repository occurrences"
}

if ($counts["scattered_user_secret_state_sql"] -ne 0) {
    Add-Failure "User password/email verification state SQL must be centralized in sql_templates.rs, got $($counts["scattered_user_secret_state_sql"]) scattered repository occurrences"
}

if ($counts["scattered_iam_workflow_insert_sql"] -ne 0) {
    Add-Failure "IAM role/token/invitation/reset/verification/MFA insert SQL must be centralized in sql_templates.rs, got $($counts["scattered_iam_workflow_insert_sql"]) scattered repository occurrences"
}

if ($counts["scattered_notification_insert_sql"] -ne 0) {
    Add-Failure "Notification outbox and delivery secret insert SQL must be centralized in sql_templates.rs, got $($counts["scattered_notification_insert_sql"]) scattered repository occurrences"
}

if ($counts["scattered_notification_state_sql"] -ne 0) {
    Add-Failure "Notification outbox and delivery secret state SQL must be centralized in sql_templates.rs, got $($counts["scattered_notification_state_sql"]) scattered repository occurrences"
}

if ($counts["scattered_mfa_recovery_insert_sql"] -ne 0) {
    Add-Failure "MFA recovery code insert SQL must be centralized in sql_templates.rs, got $($counts["scattered_mfa_recovery_insert_sql"]) scattered repository occurrences"
}

if ($counts["scattered_mfa_factor_state_sql"] -ne 0) {
    Add-Failure "MFA factor state SQL must be centralized in sql_templates.rs, got $($counts["scattered_mfa_factor_state_sql"]) scattered repository occurrences"
}

if ($counts["scattered_mfa_recovery_state_sql"] -ne 0) {
    Add-Failure "MFA recovery code state SQL must be centralized in sql_templates.rs, got $($counts["scattered_mfa_recovery_state_sql"]) scattered repository occurrences"
}

if ($counts["scattered_version_release_event_insert_sql"] -ne 0) {
    Add-Failure "Version release event insert SQL must be centralized in sql_templates.rs, got $($counts["scattered_version_release_event_insert_sql"]) scattered repository occurrences"
}

if ($counts["sqlite_upsert_syntax"] -ne 0) {
    Add-Failure "SQLite upsert syntax should be centralized in sql_templates.rs, got $($counts["sqlite_upsert_syntax"]) scattered repository occurrences"
}

if ($counts["sqlite_limit_bind"] -ne 0) {
    Add-Failure "SQLite limit bind syntax should be centralized in sql_templates.rs, got $($counts["sqlite_limit_bind"]) scattered repository occurrences"
}

foreach ($needle in @(
        "SqlDialect",
        "on conflict",
        "on duplicate key update",
        "insert ignore",
        "setup_state",
        "setup_runs",
        "setup_step_logs",
        "iam_sessions",
        'refresh_token_hash = $7',
        "iam_organizations",
        "iam_users",
        "iam_memberships",
        "iam_permissions",
        "iam_role_permissions",
        "iam_audit_logs",
        "group_concat",
        "string_agg",
        "delete from iam_memberships",
        "iam_api_tokens",
        "iam_invitations",
        "iam_password_resets",
        "iam_email_verifications",
        "iam_mfa_factors",
        "iam_mfa_recovery_codes",
        "iam_notification_outbox",
        "iam_notification_delivery_secrets",
        'token_hash = $1',
        "system_configs",
        "system_operation_records",
        "system_traffic_probe_targets",
        "system_media_assets",
        "system_version_packages",
        "system_traffic_probe_alerts",
        'key = $3',
        "limit ?",
        '$1'
    )) {
    if ($sqlTemplatesText -notlike "*$needle*") {
        Add-Failure "repository SQL templates must keep cross-dialect upsert variants: missing '$needle'"
    }
}

foreach ($driver in @("Postgres", "Mysql")) {
    $pattern = "Self::${driver}\s*=>\s*DatabaseRuntimeSupport\s*\{[\s\S]*?supported:\s*true,[\s\S]*?status:\s*`"ready`""
    if ($configText -notmatch $pattern) {
        Add-Failure "DatabaseDriver::$driver must report a ready runtime after external repository wiring"
    }
}

foreach ($needle in @(
        "RepositoryRuntimeReport",
        "RepositoryCapability",
        "repository_runtime_report",
        "repository_runtime_preflight_check",
        "DatabaseInsertIdProbeReport",
        "probe_insert_id",
        "DatabaseSetupRepositoryProbeReport",
        "DatabaseIamRepositoryProbeReport",
        "DatabaseNotificationRepositoryProbeReport",
        "DatabaseSystemRepositoryProbeReport",
        "probe_setup_repository",
        "probe_iam_repository",
        "probe_notification_repository",
        "probe_system_repository",
        "ExternalSetupRepository",
        "ExternalIamRepository",
        "ExternalNotificationRepository",
        "ExternalSystemRepository",
        "repository-runtime",
        "SetupRepository",
        "IamRepository",
        "NotificationRepository",
        "SystemRepository"
    )) {
    if ($databaseText -notlike "*$needle*") {
        Add-Failure "database runtime must expose typed repository capability matrix: missing '$needle'"
    }
}

foreach ($needle in @(
        "PostgreSQL",
        "MySQL",
        "repository set",
        "last_insert_id()"
    )) {
    if ($configText -notlike "*$needle*") {
        Add-Failure "DatabaseRuntimeSupport must document external runtime readiness: missing '$needle'"
    }
}

foreach ($needle in @(
        "InsertIdStrategy",
        "DialectSpecificPostInsertRead",
        "insert_id_strategy",
        "InsertIdRead",
        "PostInsertQuery",
        "select last_insert_id()",
        "insert_id_read"
    )) {
    if ($sqlTemplatesText -notlike "*$needle*") {
        Add-Failure "repository SQL templates must expose typed MySQL insert-id strategy: missing '$needle'"
    }
}

foreach ($needle in @(
        "PostgreSQL/MySQL",
        "ready",
        "database-insert-id-probe",
        "repository_runtime",
        "serve_ready=true"
    )) {
    if ($docsText -notlike "*$needle*") {
        Add-Failure "database runtime matrix must document external repository readiness: missing '$needle'"
    }
}

if ($failures.Count -gt 0) {
    Write-Host "repository dialect audit failed:"
    foreach ($failure in $failures) {
        Write-Host " - $failure"
    }
    exit 1
}

$summary = [ordered]@{
    sqlite_pool_type = $counts["sqlite_pool_type"]
    sqlite_connection_type = $counts["sqlite_connection_type"]
    sqlite_transaction_type = $counts["sqlite_transaction_type"]
    sqlite_dialect_constant = $counts["sqlite_dialect_constant"]
    sqlite_row_type = $counts["sqlite_row_type"]
    sqlite_last_insert_id = $counts["sqlite_last_insert_id"]
    insert_returning_id = $counts["insert_returning_id"]
    scattered_setup_state_sql = $counts["scattered_setup_state_sql"]
    scattered_setup_run_sql = $counts["scattered_setup_run_sql"]
    scattered_session_sql = $counts["scattered_session_sql"]
    scattered_api_token_read_sql = $counts["scattered_api_token_read_sql"]
    scattered_system_settings_sql = $counts["scattered_system_settings_sql"]
    scattered_system_registry_sql = $counts["scattered_system_registry_sql"]
    scattered_traffic_probe_state_sql = $counts["scattered_traffic_probe_state_sql"]
    scattered_media_version_read_delete_sql = $counts["scattered_media_version_read_delete_sql"]
    scattered_version_package_read_update_sql = $counts["scattered_version_package_read_update_sql"]
    scattered_system_insert_sql = $counts["scattered_system_insert_sql"]
    scattered_identity_insert_sql = $counts["scattered_identity_insert_sql"]
    scattered_identity_count_sql = $counts["scattered_identity_count_sql"]
    scattered_membership_insert_sql = $counts["scattered_membership_insert_sql"]
    scattered_audit_insert_sql = $counts["scattered_audit_insert_sql"]
    scattered_org_user_management_sql = $counts["scattered_org_user_management_sql"]
    scattered_iam_permission_list_sql = $counts["scattered_iam_permission_list_sql"]
    scattered_role_permission_state_sql = $counts["scattered_role_permission_state_sql"]
    scattered_iam_role_state_sql = $counts["scattered_iam_role_state_sql"]
    scattered_iam_pending_state_sql = $counts["scattered_iam_pending_state_sql"]
    scattered_user_secret_state_sql = $counts["scattered_user_secret_state_sql"]
    scattered_iam_workflow_insert_sql = $counts["scattered_iam_workflow_insert_sql"]
    scattered_notification_insert_sql = $counts["scattered_notification_insert_sql"]
    scattered_notification_state_sql = $counts["scattered_notification_state_sql"]
    scattered_mfa_recovery_insert_sql = $counts["scattered_mfa_recovery_insert_sql"]
    scattered_mfa_factor_state_sql = $counts["scattered_mfa_factor_state_sql"]
    scattered_mfa_recovery_state_sql = $counts["scattered_mfa_recovery_state_sql"]
    scattered_version_release_event_insert_sql = $counts["scattered_version_release_event_insert_sql"]
    sqlite_placeholder = $counts["sqlite_placeholder"]
    rust_question_mark_operator = $counts["rust_question_mark_operator"]
    sqlite_upsert_syntax = $counts["sqlite_upsert_syntax"]
    sqlite_limit_bind = $counts["sqlite_limit_bind"]
    centralized_upsert_templates = $true
    centralized_limit_templates = $true
    centralized_setup_state_templates = $true
    centralized_setup_run_templates = $true
    centralized_session_templates = $true
    centralized_api_token_read_templates = $true
    centralized_system_settings_templates = $true
    centralized_system_registry_templates = $true
    centralized_traffic_probe_state_templates = $true
    centralized_media_version_read_delete_templates = $true
    centralized_version_package_read_update_templates = $true
    centralized_system_insert_templates = $true
    centralized_identity_insert_templates = $true
    centralized_identity_count_templates = $true
    centralized_membership_insert_templates = $true
    centralized_audit_insert_templates = $true
    centralized_org_user_management_templates = $true
    centralized_iam_permission_list_templates = $true
    centralized_role_permission_state_templates = $true
    centralized_iam_role_state_templates = $true
    centralized_iam_pending_state_templates = $true
    centralized_user_secret_state_templates = $true
    centralized_iam_workflow_insert_templates = $true
    centralized_notification_insert_templates = $true
    centralized_notification_state_templates = $true
    centralized_mfa_recovery_insert_templates = $true
    centralized_mfa_factor_state_templates = $true
    centralized_mfa_recovery_state_templates = $true
    centralized_version_release_event_insert_templates = $true
}

Write-Host ("repository dialect audit passed: external database runtime is ready and SQLite-specific repository code remains isolated: " + ($summary | ConvertTo-Json -Compress))

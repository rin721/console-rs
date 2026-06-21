param(
    [string]$Root = (Join-Path $PSScriptRoot "..")
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$rootPath = (Resolve-Path -LiteralPath $Root).Path
$migrationRoot = Join-Path $rootPath "migrations"
$failures = New-Object System.Collections.Generic.List[string]
$utf8Strict = [System.Text.UTF8Encoding]::new($false, $true)

$requiredTables = @(
    "setup_state",
    "setup_runs",
    "setup_step_logs",
    "iam_organizations",
    "iam_users",
    "iam_roles",
    "iam_permissions",
    "iam_role_permissions",
    "iam_memberships",
    "iam_sessions",
    "iam_api_tokens",
    "iam_invitations",
    "iam_password_resets",
    "iam_email_verifications",
    "iam_mfa_factors",
    "iam_mfa_recovery_codes",
    "iam_audit_logs",
    "iam_notification_outbox",
    "iam_notification_delivery_secrets",
    "system_apis",
    "system_menus",
    "system_configs",
    "system_dictionaries",
    "system_parameters",
    "system_operation_records",
    "system_server_metrics",
    "system_version_packages",
    "system_version_release_events",
    "system_media_assets",
    "system_traffic_probe_targets",
    "system_traffic_probe_results",
    "system_traffic_probe_alerts"
)

function Add-Failure {
    param([string]$Message)
    $failures.Add($Message) | Out-Null
}

function Read-SqlText {
    param([System.IO.FileInfo[]]$Files)
    $parts = New-Object System.Collections.Generic.List[string]
    foreach ($file in $Files) {
        try {
            $parts.Add([System.IO.File]::ReadAllText($file.FullName, $utf8Strict)) | Out-Null
        } catch {
            Add-Failure "$($file.FullName) is not strict UTF-8: $($_.Exception.Message)"
        }
    }
    return ($parts -join "`n").ToLowerInvariant()
}

function Get-DialectFiles {
    param([string]$Dialect)
    if ($Dialect -eq "sqlite") {
        return @(Get-ChildItem -LiteralPath $migrationRoot -Filter "*.sql" -File | Sort-Object Name)
    }
    $dir = Join-Path $migrationRoot $Dialect
    if (-not (Test-Path -LiteralPath $dir -PathType Container)) {
        Add-Failure "missing migration dialect directory: migrations/$Dialect"
        return @()
    }
    return @(Get-ChildItem -LiteralPath $dir -Filter "*.sql" -File | Sort-Object Name)
}

foreach ($dialect in @("sqlite", "postgres", "mysql")) {
    $files = @(Get-DialectFiles $dialect)
    if ($files.Count -eq 0) {
        Add-Failure "no SQL files found for $dialect"
        continue
    }

    $sql = Read-SqlText $files
    foreach ($table in $requiredTables) {
        $escaped = [regex]::Escape($table)
        if ($sql -notmatch "create\s+table\s+(if\s+not\s+exists\s+)?`?$escaped`?\b") {
            Add-Failure "$dialect schema does not create required table: $table"
        }
    }

    if ($sql -match "\b(plugin|plugins|plugin_registry|plugin_instance|plugin_transport|plugin_api)\b") {
        Add-Failure "$dialect schema contains forbidden plugin runtime wording"
    }
    if ($sql -match "(aoi admin|go-scaffold|go-admin|github\.com/rei0721/go-scaffold|aoi-admin)") {
        Add-Failure "$dialect schema contains forbidden old project identity"
    }
}

$postgresSql = Read-SqlText @(Get-DialectFiles "postgres")
if ($postgresSql -notmatch "generated\s+by\s+default\s+as\s+identity") {
    Add-Failure "postgres schema must use PostgreSQL identity columns"
}
if ($postgresSql -match "\b(auto_increment|autoincrement|sqlite_)\b") {
    Add-Failure "postgres schema contains SQLite/MySQL-only identity or internal names"
}

$mysqlSql = Read-SqlText @(Get-DialectFiles "mysql")
if ($mysqlSql -notmatch "\bauto_increment\b") {
    Add-Failure "mysql schema must use MySQL auto_increment columns"
}
if ($mysqlSql -notmatch "engine=innodb") {
    Add-Failure "mysql schema must pin InnoDB for foreign key behavior"
}
if ($mysqlSql -match "\b(autoincrement|generated\s+by\s+default\s+as\s+identity|sqlite_)\b") {
    Add-Failure "mysql schema contains SQLite/PostgreSQL-only identity or internal names"
}

if ($failures.Count -gt 0) {
    Write-Host "database schema dialect scan failed:"
    foreach ($failure in $failures) {
        Write-Host " - $failure"
    }
    exit 1
}

Write-Host "database schema dialect scan passed: sqlite, postgres, and mysql schema drafts expose required tables without plugin runtime tables."

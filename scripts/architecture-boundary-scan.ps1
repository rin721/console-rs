Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$failures = New-Object System.Collections.Generic.List[string]

function Add-Failure {
    param([string]$Message)
    $failures.Add($Message) | Out-Null
}

function To-RelativePath {
    param([string]$Path)
    $fullPath = (Resolve-Path -LiteralPath $Path).Path
    $basePath = $root.TrimEnd("\", "/")
    if ($fullPath.StartsWith($basePath, [System.StringComparison]::OrdinalIgnoreCase)) {
        return $fullPath.Substring($basePath.Length).TrimStart("\", "/").Replace("\", "/")
    }
    return $fullPath.Replace("\", "/")
}

function Read-Text {
    param([string]$Path)
    return [System.IO.File]::ReadAllText($Path, [System.Text.UTF8Encoding]::new($false, $true))
}

function Source-Files {
    param([string]$RelativeDir)
    $dir = Join-Path $root $RelativeDir
    if (-not (Test-Path -LiteralPath $dir -PathType Container)) {
        return @()
    }
    return Get-ChildItem -LiteralPath $dir -Recurse -File -Filter "*.rs" |
        Where-Object { (To-RelativePath $_.FullName) -notmatch "/tests?/" }
}

function Assert-NoPattern {
    param(
        [object[]]$Files,
        [string]$Pattern,
        [string]$Reason
    )
    foreach ($file in $Files) {
        $text = Read-Text $file.FullName
        foreach ($match in [regex]::Matches($text, $Pattern, [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)) {
            Add-Failure "$Reason in $(To-RelativePath $file.FullName): $($match.Value)"
        }
    }
}

function Assert-FileContains {
    param(
        [string]$RelativePath,
        [string]$Needle,
        [string]$Reason
    )
    $path = Join-Path $root $RelativePath
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        Add-Failure "$Reason missing file: $RelativePath"
        return
    }
    $text = Read-Text $path
    if ($text -notlike "*$Needle*") {
        Add-Failure "$Reason in $RelativePath"
    }
}

$sourceRoot = Join-Path $root "crates"
if (Test-Path -LiteralPath $sourceRoot -PathType Container) {
    Get-ChildItem -LiteralPath $sourceRoot -Recurse -Directory |
        Where-Object { $_.Name -match "^(common|utils|helpers|shared)$" } |
        ForEach-Object {
            Add-Failure "unbounded shared directory name is forbidden: $(To-RelativePath $_.FullName)"
        }
}

$handlerFiles = Source-Files "crates/core/app/src/handler"
Assert-NoPattern $handlerFiles "crate::(infrastructure|repository)::" "handler must not depend on infrastructure or repository"
Assert-NoPattern $handlerFiles "\bsqlx::|\bPool\s*<|sqlx::query|query_as!" "handler must not access database runtime or SQL"
Assert-NoPattern $handlerFiles "impl\s+From\s*<\s*(sqlx::Error|std::io::Error)\s*>\s+for\s+HttpError" "handler must map AppError only at HTTP boundary"

$serviceFiles = Source-Files "crates/core/app/src/service"
Assert-NoPattern $serviceFiles "crate::infrastructure::|SqliteRepository|\bsqlx::|\bPool\s*<|sqlx::query|query_as!" "service/usecase must depend on traits and domain data, not concrete infrastructure"
Assert-NoPattern $serviceFiles "crate::transport::" "service/usecase must not depend on HTTP transport contracts"

$repositoryFiles = Source-Files "crates/core/app/src/repository"
Assert-NoPattern $repositoryFiles "crate::transport::|\baxum::|\bhttp::|\bsqlx::|\bPool\s*<" "repository traits must stay transport-free and database-runtime-free"

$infrastructureFiles = Source-Files "crates/core/app/src/infrastructure"
Assert-NoPattern $infrastructureFiles "\baxum::|crate::handler::" "infrastructure must not depend on HTTP handlers"

$typesFile = Join-Path $root "crates/core/types/src/lib.rs"
if (Test-Path -LiteralPath $typesFile -PathType Leaf) {
    $typesText = Read-Text $typesFile
    foreach ($match in [regex]::Matches($typesText, '(?m)^\s*pub\s+(struct|enum|trait|type)\s+([A-Za-z0-9_]+)')) {
        $name = $match.Groups[2].Value
        if ($name -notin @("AppError", "AppResult")) {
            Add-Failure "types crate may only expose app lifecycle contracts, found: $name"
        }
    }
    Assert-NoPattern @([pscustomobject]@{ FullName = $typesFile }) "crate::domain::|crate::repository::|crate::handler::|crate::transport::|serde::|sqlx::|std::io::" "types crate must not hold business, HTTP, repository, database, storage, or frontend DTOs"
}

$typesCargo = Join-Path $root "crates/core/types/Cargo.toml"
if (Test-Path -LiteralPath $typesCargo -PathType Leaf) {
    Assert-NoPattern @([pscustomobject]@{ FullName = $typesCargo }) "(?m)^\s*(sqlx|serde)\s*=" "types crate must not depend on database or DTO serialization crates"
}

Assert-FileContains "crates/core/app/src/service/iam.rs" "repo: Arc<dyn IamRepository>" "IamService must receive its repository through a trait object"
Assert-FileContains "crates/core/app/src/service/setup.rs" "setup_repo: Arc<dyn SetupRepository>" "SetupService must receive SetupRepository through a trait object"
Assert-FileContains "crates/core/app/src/service/setup.rs" "iam_repo: Arc<dyn IamRepository>" "SetupService must receive IamRepository through a trait object"
Assert-FileContains "crates/core/app/src/service/notification.rs" "pub trait NotificationSender: Send + Sync" "NotificationSender must stay a service-level interface"
Assert-FileContains "crates/core/app/src/service/notification.rs" "repo: Arc<dyn NotificationRepository>" "NotificationService must receive NotificationRepository through a trait object"
Assert-FileContains "crates/core/app/src/service/notification.rs" "sender: Arc<dyn NotificationSender>" "NotificationService must receive NotificationSender through a trait object"
Assert-FileContains "crates/core/app/src/service/system.rs" "pub trait TrafficProbeRunner: Send + Sync" "TrafficProbeRunner must stay a service-level interface"
Assert-FileContains "crates/core/app/src/service/system.rs" "pub trait SystemMetricsCollector: Send + Sync" "SystemMetricsCollector must stay a service-level interface"
Assert-FileContains "crates/core/app/src/service/system.rs" "pub trait MediaStorage: Send + Sync" "MediaStorage must stay a service-level interface"
Assert-FileContains "crates/core/app/src/service/system.rs" "repo: Arc<dyn SystemRepository>" "SystemService must receive SystemRepository through a trait object"
Assert-FileContains "crates/core/app/src/service/system.rs" "traffic_probe_runner: Arc<dyn TrafficProbeRunner>" "SystemService must receive TrafficProbeRunner through a trait object"
Assert-FileContains "crates/core/app/src/service/system.rs" "media_storage: Arc<dyn MediaStorage>" "SystemService must receive MediaStorage through a trait object"
Assert-FileContains "crates/core/app/src/service/system.rs" "metrics_collector: Arc<dyn SystemMetricsCollector>" "SystemService must receive SystemMetricsCollector through a trait object"

Assert-FileContains "crates/core/app/src/infrastructure/sqlite_repository.rs" "impl SetupRepository for SqliteRepository" "SQLite repository must implement SetupRepository at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/sqlite_repository.rs" "impl IamRepository for SqliteRepository" "SQLite repository must implement IamRepository at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/sqlite_repository.rs" "impl NotificationRepository for SqliteRepository" "SQLite repository must implement NotificationRepository at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/sqlite_repository.rs" "impl SystemRepository for SqliteRepository" "SQLite repository must implement SystemRepository at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/external_setup_repository.rs" "impl SetupRepository for ExternalSetupRepository" "External setup repository must implement SetupRepository at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/external_iam_repository.rs" "impl IamRepository for ExternalIamRepository" "External IAM repository must implement IamRepository at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/external_notification_repository.rs" "impl NotificationRepository for ExternalNotificationRepository" "External notification repository must implement NotificationRepository at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/external_system_repository.rs" "impl SystemRepository for ExternalSystemRepository" "External system repository must implement SystemRepository at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/notification.rs" "impl NotificationSender for LogNotificationSender" "Log notification sender must implement NotificationSender at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/notification.rs" "impl NotificationSender for SmtpNotificationSender" "SMTP notification sender must implement NotificationSender at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/notification.rs" "impl NotificationSender for QueueNotificationSender" "Queue notification sender must implement NotificationSender at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/notification.rs" "impl NotificationSender for FileNotificationSender" "File notification sender must implement NotificationSender at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/media_storage.rs" "impl MediaStorage for LocalMediaStorage" "Local storage must implement MediaStorage at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/media_storage.rs" "impl MediaStorage for S3MediaStorage" "S3 storage must implement MediaStorage at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/system_metrics.rs" "impl SystemMetricsCollector for SysinfoMetricsCollector" "Sysinfo collector must implement SystemMetricsCollector at the infrastructure boundary"
Assert-FileContains "crates/core/app/src/infrastructure/traffic_probe.rs" "impl TrafficProbeRunner for HttpTrafficProbeRunner" "HTTP traffic probe runner must implement TrafficProbeRunner at the infrastructure boundary"

Assert-FileContains "crates/core/app/src/app.rs" "let repos = database::repositories(&database);" "App boot must obtain repositories from the database composition root"
Assert-FileContains "crates/core/app/src/app.rs" "let notification_sender: Arc<dyn NotificationSender>" "App boot must inject notification sender through the service interface"
Assert-FileContains "crates/core/app/src/app.rs" "fn media_storage_from_settings(settings: &Settings) -> AppResult<Arc<dyn MediaStorage>>" "App boot must inject media storage through the service interface"
Assert-FileContains "crates/core/app/src/app.rs" "let metrics_collector = Arc::new(SysinfoMetricsCollector);" "App boot must inject metrics collector at the composition root"
Assert-FileContains "crates/core/app/src/app.rs" "let traffic_probe_runner = Arc::new(HttpTrafficProbeRunner::new());" "App boot must inject traffic probe runner at the composition root"

if ($failures.Count -gt 0) {
    Write-Host "architecture boundary scan failed:"
    foreach ($failure in $failures) {
        Write-Host " - $failure"
    }
    exit 1
}

Write-Host "architecture boundary scan passed: handler, service, repository, infrastructure, types, and trait-injection boundaries are clean."

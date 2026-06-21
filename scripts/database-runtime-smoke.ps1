param(
    [string]$Root = (Join-Path $PSScriptRoot "..")
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$rootPath = (Resolve-Path -LiteralPath $Root).Path
$targetRoot = [System.IO.Path]::GetFullPath((Join-Path $rootPath "target"))
$smokeRoot = [System.IO.Path]::GetFullPath((Join-Path $targetRoot "database-runtime-smoke"))
$configPath = Join-Path $rootPath "configs/console.example.yaml"
$databasePath = Join-Path $smokeRoot "runtime.sqlite"
$savedEnv = @{}

function Set-SmokeEnv {
    param(
        [string]$Name,
        [string]$Value
    )
    if (-not $savedEnv.ContainsKey($Name)) {
        $savedEnv[$Name] = [Environment]::GetEnvironmentVariable($Name, "Process")
    }
    [Environment]::SetEnvironmentVariable($Name, $Value, "Process")
}

function Restore-SmokeEnv {
    foreach ($name in $savedEnv.Keys) {
        [Environment]::SetEnvironmentVariable($name, $savedEnv[$name], "Process")
    }
}

function Invoke-AppJson {
    param(
        [string[]]$Arguments
    )
    $output = & cargo @("run", "-q", "-p", "app", "--") @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "command failed with exit code ${LASTEXITCODE}: cargo run -q -p app -- $($Arguments -join ' ')"
    }
    return (($output -join "`n") | ConvertFrom-Json)
}

function Assert-Equal {
    param(
        [object]$Actual,
        [object]$Expected,
        [string]$Message
    )
    if ($Actual -ne $Expected) {
        throw "$Message expected '$Expected' but got '$Actual'"
    }
}

function Assert-MigrationDigests {
    param(
        [object[]]$Files
    )
    foreach ($file in $Files) {
        if (-not ($file.sha256 -is [string]) -or $file.sha256 -notmatch "^[0-9a-f]{64}$") {
            throw "database-plan migration file '$($file.name)' does not expose a valid SHA-256 digest"
        }
    }
}

if (-not $smokeRoot.StartsWith($targetRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "refusing to clean database smoke directory outside target: $smokeRoot"
}

try {
    if (Test-Path -LiteralPath $smokeRoot) {
        Remove-Item -LiteralPath $smokeRoot -Recurse -Force
    }
    New-Item -ItemType Directory -Force -Path $smokeRoot | Out-Null

    Push-Location $rootPath

    Set-SmokeEnv "CONSOLE__DATABASE__DRIVER" "sqlite"
    Set-SmokeEnv "CONSOLE__DATABASE__URL" "sqlite://target/database-runtime-smoke/runtime.sqlite"

    $plan = Invoke-AppJson @("database-plan", "--config", $configPath)
    Assert-Equal $plan.driver "sqlite" "database-plan driver"
    Assert-Equal $plan.runtime_supported $true "database-plan runtime support"
    Assert-Equal $plan.runtime_status "ready" "database-plan runtime status"
    if ($plan.migration_files.Count -lt 1) {
        throw "database-plan returned no migration files"
    }
    Assert-MigrationDigests $plan.migration_files

    $ping = Invoke-AppJson @("database-ping", "--config", $configPath)
    Assert-Equal $ping.driver "sqlite" "database-ping driver"
    Assert-Equal $ping.connection_ok $true "database-ping connection"
    Assert-Equal $ping.repository_ready $true "database-ping repository readiness"
    Assert-Equal $ping.migration_runtime_ready $true "database-ping migration readiness"

    $insertId = Invoke-AppJson @("database-insert-id-probe", "--config", $configPath)
    Assert-Equal $insertId.driver "sqlite" "database-insert-id-probe driver"
    Assert-Equal $insertId.inserted_id 1 "database-insert-id-probe inserted id"
    Assert-Equal $insertId.same_connection_required $false "database-insert-id-probe same connection"
    Assert-Equal $insertId.insert_id_read "ReturningIdInStatement" "database-insert-id-probe read strategy"

    $migrate = Invoke-AppJson @("database-migrate", "--config", $configPath)
    Assert-Equal $migrate.driver "sqlite" "database-migrate driver"
    Assert-Equal $migrate.repository_ready $true "database-migrate repository readiness"
    if ($migrate.applied_files.Count -ne $plan.migration_files.Count) {
        throw "database-migrate applied $($migrate.applied_files.Count) files, expected $($plan.migration_files.Count)"
    }
    if ($migrate.skipped_files.Count -ne 0) {
        throw "database-migrate unexpectedly skipped files on a fresh smoke database"
    }
    if (-not (Test-Path -LiteralPath $databasePath -PathType Leaf)) {
        throw "database-migrate did not create expected SQLite database: $databasePath"
    }

    $setupRepo = Invoke-AppJson @("database-setup-repository-probe", "--config", $configPath)
    Assert-Equal $setupRepo.driver "sqlite" "database-setup-repository-probe driver"
    Assert-Equal $setupRepo.implementation "SqliteRepository" "database-setup-repository-probe implementation"
    Assert-Equal $setupRepo.missing_complete_result $false "database-setup-repository-probe missing complete"
    Assert-Equal $setupRepo.run_listed $true "database-setup-repository-probe run listed"
    Assert-Equal $setupRepo.log_count 1 "database-setup-repository-probe log count"

    $iamRepo = Invoke-AppJson @("database-iam-repository-probe", "--config", $configPath)
    Assert-Equal $iamRepo.driver "sqlite" "database-iam-repository-probe driver"
    Assert-Equal $iamRepo.implementation "SqliteRepository" "database-iam-repository-probe implementation"
    Assert-Equal $iamRepo.initial_admin_created $true "database-iam-repository-probe initial admin"
    Assert-Equal $iamRepo.permissions_synced $true "database-iam-repository-probe permissions"
    Assert-Equal $iamRepo.organization_roundtrip $true "database-iam-repository-probe organization"
    Assert-Equal $iamRepo.org_user_roundtrip $true "database-iam-repository-probe org user"
    Assert-Equal $iamRepo.role_roundtrip $true "database-iam-repository-probe role"
    Assert-Equal $iamRepo.session_roundtrip $true "database-iam-repository-probe session"
    Assert-Equal $iamRepo.refresh_rotation_roundtrip $true "database-iam-repository-probe refresh rotation"
    Assert-Equal $iamRepo.api_token_roundtrip $true "database-iam-repository-probe API token"
    Assert-Equal $iamRepo.registration_pending_roundtrip $true "database-iam-repository-probe registration pending"
    Assert-Equal $iamRepo.invitation_roundtrip $true "database-iam-repository-probe invitation"
    Assert-Equal $iamRepo.password_reset_roundtrip $true "database-iam-repository-probe password reset"
    Assert-Equal $iamRepo.email_verification_roundtrip $true "database-iam-repository-probe email verification"
    Assert-Equal $iamRepo.mfa_roundtrip $true "database-iam-repository-probe MFA"
    Assert-Equal $iamRepo.audit_record_written $true "database-iam-repository-probe audit"

    $notificationRepo = Invoke-AppJson @("database-notification-repository-probe", "--config", $configPath)
    Assert-Equal $notificationRepo.driver "sqlite" "database-notification-repository-probe driver"
    Assert-Equal $notificationRepo.implementation "SqliteRepository" "database-notification-repository-probe implementation"
    Assert-Equal $notificationRepo.claimed_probe_items 3 "database-notification-repository-probe claimed probe items"
    Assert-Equal $notificationRepo.delivered_result $true "database-notification-repository-probe delivered result"
    Assert-Equal $notificationRepo.retry_result $true "database-notification-repository-probe retry result"
    Assert-Equal $notificationRepo.final_failure_result $true "database-notification-repository-probe final failure result"
    Assert-Equal $notificationRepo.dead_letter_reported $true "database-notification-repository-probe dead-letter report"
    Assert-Equal $notificationRepo.dead_letter_secret_state "purged" "database-notification-repository-probe dead-letter secret state"
    Assert-Equal $notificationRepo.delivered_secret_purged $true "database-notification-repository-probe delivered secret purge"
    Assert-Equal $notificationRepo.failed_secret_purged $true "database-notification-repository-probe failed secret purge"
    Assert-Equal $notificationRepo.purged_requeue_skipped $true "database-notification-repository-probe purged requeue skip"
    Assert-Equal $notificationRepo.pending_secret_requeue_result $true "database-notification-repository-probe pending secret requeue"

    $systemRepo = Invoke-AppJson @("database-system-repository-probe", "--config", $configPath)
    Assert-Equal $systemRepo.driver "sqlite" "database-system-repository-probe driver"
    Assert-Equal $systemRepo.implementation "SqliteRepository" "database-system-repository-probe implementation"
    Assert-Equal $systemRepo.api_catalog_synced $true "database-system-repository-probe API catalog sync"
    Assert-Equal $systemRepo.menu_synced $true "database-system-repository-probe menu sync"
    Assert-Equal $systemRepo.config_roundtrip $true "database-system-repository-probe config roundtrip"
    Assert-Equal $systemRepo.dictionary_roundtrip $true "database-system-repository-probe dictionary roundtrip"
    Assert-Equal $systemRepo.parameter_roundtrip $true "database-system-repository-probe parameter roundtrip"
    Assert-Equal $systemRepo.operation_record_written $true "database-system-repository-probe operation record"
    Assert-Equal $systemRepo.operation_record_summary_reported $true "database-system-repository-probe operation record summary"
    Assert-Equal $systemRepo.version_package_roundtrip $true "database-system-repository-probe version package"
    Assert-Equal $systemRepo.media_asset_roundtrip $true "database-system-repository-probe media asset"
    Assert-Equal $systemRepo.traffic_probe_roundtrip $true "database-system-repository-probe traffic probe"

    $history = Invoke-AppJson @("database-migration-history", "--config", $configPath)
    Assert-Equal $history.driver "sqlite" "database-migration-history driver"
    Assert-Equal $history.checksum_source "sqlx_migrator" "database-migration-history checksum source"
    if ($history.records.Count -ne $plan.migration_files.Count) {
        throw "database-migration-history returned $($history.records.Count) records, expected $($plan.migration_files.Count)"
    }
    foreach ($record in $history.records) {
        if ($record.success -ne $true) {
            throw "database-migration-history contains unsuccessful SQLite migration record: $($record.name)"
        }
        if (-not ($record.checksum -is [string]) -or $record.checksum.Length -lt 1) {
            throw "database-migration-history contains an empty SQLite checksum for $($record.name)"
        }
    }

    $schema = Invoke-AppJson @("database-schema-check", "--config", $configPath)
    Assert-Equal $schema.driver "sqlite" "database-schema-check driver"
    Assert-Equal $schema.schema_ready $true "database-schema-check readiness"
    Assert-Equal $schema.repository_ready $true "database-schema-check repository readiness"
    if ($schema.missing_tables.Count -ne 0) {
        throw "database-schema-check reported missing tables: $($schema.missing_tables -join ', ')"
    }

    $preflight = Invoke-AppJson @("database-preflight", "--config", $configPath)
    Assert-Equal $preflight.driver "sqlite" "database-preflight driver"
    Assert-Equal $preflight.connection_ok $true "database-preflight connection"
    Assert-Equal $preflight.migration_plan_ok $true "database-preflight migration plan"
    Assert-Equal $preflight.migration_history_ok $true "database-preflight migration history"
    Assert-Equal $preflight.schema_ready $true "database-preflight schema readiness"
    Assert-Equal $preflight.repository_ready $true "database-preflight repository readiness"
    Assert-Equal $preflight.serve_ready $true "database-preflight serve readiness"

    $secondMigrate = Invoke-AppJson @("database-migrate", "--config", $configPath)
    if ($secondMigrate.applied_files.Count -ne 0) {
        throw "database-migrate should be idempotent on second run, applied $($secondMigrate.applied_files.Count) files"
    }
    if ($secondMigrate.skipped_files.Count -ne $plan.migration_files.Count) {
        throw "database-migrate second run skipped $($secondMigrate.skipped_files.Count) files, expected $($plan.migration_files.Count)"
    }

    [Console]::Out.WriteLine("database runtime smoke passed: plan, ping, insert ID probe, migrate, SetupRepository probe, IamRepository probe, NotificationRepository probe including safe requeue, SystemRepository probe, migration history, schema check, preflight, and idempotent migrate succeeded on temporary SQLite runtime.")
} finally {
    Restore-SmokeEnv
    Pop-Location
}

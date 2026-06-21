param(
    [string]$Root = (Join-Path $PSScriptRoot ".."),
    [string]$Config = "configs/console.example.yaml",
    [string]$Secrets = "",
    [ValidateSet("", "sqlite", "postgres", "mysql")]
    [string]$Driver = "",
    [string]$Url = "",
    [switch]$ApplyMigrations
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$rootPath = (Resolve-Path -LiteralPath $Root).Path
$savedEnv = @{}

function Resolve-ProjectPath {
    param(
        [string]$Path
    )
    if ([string]::IsNullOrWhiteSpace($Path)) {
        return ""
    }
    if ([System.IO.Path]::IsPathRooted($Path)) {
        return $Path
    }
    return (Join-Path $rootPath $Path)
}

function Set-PreflightEnv {
    param(
        [string]$Name,
        [string]$Value
    )
    if (-not $savedEnv.ContainsKey($Name)) {
        $savedEnv[$Name] = [Environment]::GetEnvironmentVariable($Name, "Process")
    }
    [Environment]::SetEnvironmentVariable($Name, $Value, "Process")
}

function Restore-PreflightEnv {
    foreach ($name in $savedEnv.Keys) {
        [Environment]::SetEnvironmentVariable($name, $savedEnv[$name], "Process")
    }
}

function New-AppArguments {
    param(
        [string]$Command
    )
    $arguments = @($Command, "--config", $script:configPath)
    if (-not [string]::IsNullOrWhiteSpace($script:secretsPath)) {
        $arguments += @("--secrets", $script:secretsPath)
    }
    return $arguments
}

function Invoke-AppJson {
    param(
        [string]$Command
    )
    $arguments = New-AppArguments $Command
    $output = & cargo @("run", "-q", "-p", "app", "--") @arguments
    if ($LASTEXITCODE -ne 0) {
        throw "command failed with exit code ${LASTEXITCODE}: cargo run -q -p app -- $($arguments -join ' ')"
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

function Assert-HistoryMatchesPlanShape {
    param(
        [object]$History,
        [object[]]$PlanFiles
    )
    if ($History.records.Count -ne $PlanFiles.Count) {
        throw "database-migration-history returned $($History.records.Count) records, expected $($PlanFiles.Count)"
    }
    foreach ($record in $History.records) {
        if ($record.success -ne $true) {
            throw "database-migration-history contains unsuccessful migration record: $($record.name)"
        }
        if (-not ($record.checksum -is [string]) -or $record.checksum.Length -lt 1) {
            throw "database-migration-history contains an empty checksum for $($record.name)"
        }
    }
}

function Assert-ExternalRuntimePolicy {
    param(
        [object]$Preflight
    )
    Assert-Equal $Preflight.repository_ready $true "database-preflight repository readiness"
    Assert-Equal $Preflight.serve_ready $true "database-preflight serve readiness"
}

function Assert-InsertIdProbe {
    param(
        [object]$Report,
        [string]$Driver
    )
    Assert-Equal $Report.driver $Driver "database-insert-id-probe driver"
    Assert-Equal $Report.inserted_id 1 "database-insert-id-probe inserted id"

    if ($Driver -eq "mysql") {
        Assert-Equal $Report.same_connection_required $true "database-insert-id-probe MySQL connection scope"
        if ($Report.insert_id_read -notlike "*last_insert_id()*") {
            throw "database-insert-id-probe did not report MySQL last_insert_id() strategy: $($Report.insert_id_read)"
        }
        return
    }

    Assert-Equal $Report.same_connection_required $false "database-insert-id-probe returning connection scope"
    Assert-Equal $Report.insert_id_read "ReturningIdInStatement" "database-insert-id-probe returning read strategy"
}

$configPath = Resolve-ProjectPath $Config
$secretsPath = Resolve-ProjectPath $Secrets

try {
    Push-Location $rootPath

    if (-not [string]::IsNullOrWhiteSpace($Driver)) {
        Set-PreflightEnv "CONSOLE__DATABASE__DRIVER" $Driver
    }
    if (-not [string]::IsNullOrWhiteSpace($Url)) {
        Set-PreflightEnv "CONSOLE__DATABASE__URL" $Url
    }

    $plan = Invoke-AppJson "database-plan"
    if ($plan.migration_files.Count -lt 1) {
        throw "database-plan returned no migration files"
    }
    Assert-MigrationDigests $plan.migration_files

    $ping = Invoke-AppJson "database-ping"
    Assert-Equal $ping.driver $plan.driver "database-ping driver"
    Assert-Equal $ping.connection_ok $true "database-ping connection"

    $insertId = Invoke-AppJson "database-insert-id-probe"
    Assert-InsertIdProbe $insertId $plan.driver

    if ($ApplyMigrations) {
        $migrate = Invoke-AppJson "database-migrate"
        Assert-Equal $migrate.driver $plan.driver "database-migrate driver"
        $reportedFiles = $migrate.applied_files.Count + $migrate.skipped_files.Count
        if ($reportedFiles -ne $plan.migration_files.Count) {
            throw "database-migrate reported $reportedFiles files, expected $($plan.migration_files.Count)"
        }
    }

    $history = Invoke-AppJson "database-migration-history"
    Assert-Equal $history.driver $plan.driver "database-migration-history driver"
    Assert-HistoryMatchesPlanShape $history $plan.migration_files

    $schema = Invoke-AppJson "database-schema-check"
    Assert-Equal $schema.driver $plan.driver "database-schema-check driver"
    Assert-Equal $schema.schema_ready $true "database-schema-check readiness"
    if ($schema.missing_tables.Count -ne 0) {
        throw "database-schema-check reported missing tables: $($schema.missing_tables -join ', ')"
    }

    $preflight = Invoke-AppJson "database-preflight"
    Assert-Equal $preflight.driver $plan.driver "database-preflight driver"
    Assert-Equal $preflight.connection_ok $true "database-preflight connection"
    Assert-Equal $preflight.migration_plan_ok $true "database-preflight migration plan"
    Assert-Equal $preflight.migration_history_ok $true "database-preflight migration history"
    Assert-Equal $preflight.schema_ready $true "database-preflight schema readiness"
    Assert-ExternalRuntimePolicy $preflight

    [Console]::Out.WriteLine("database deploy preflight passed for $($preflight.driver): insert_id_read=$($insertId.insert_id_read), schema_ready=$($preflight.schema_ready), repository_ready=$($preflight.repository_ready), serve_ready=$($preflight.serve_ready).")
} finally {
    Restore-PreflightEnv
    Pop-Location
}

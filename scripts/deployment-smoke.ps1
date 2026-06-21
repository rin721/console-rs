param(
    [string]$Root = (Join-Path $PSScriptRoot ".."),
    [int]$Port = 18080,
    [switch]$SkipBuild,
    [switch]$SkipWebBuild
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$rootPath = (Resolve-Path -LiteralPath $Root).Path
$targetRoot = [System.IO.Path]::GetFullPath((Join-Path $rootPath "target"))
$smokeRoot = [System.IO.Path]::GetFullPath((Join-Path $targetRoot "deployment-smoke"))
$stdoutPath = Join-Path $smokeRoot "server.out.log"
$stderrPath = Join-Path $smokeRoot "server.err.log"
$acceptanceReportPath = Join-Path $smokeRoot "target-acceptance-local.json"
$configPath = Join-Path $rootPath "configs/console.example.yaml"
$binaryPath = Join-Path $rootPath "target/debug/app.exe"
$baseUrl = "http://127.0.0.1:$Port"
$process = $null
$savedEnv = @{}
$smokeSessionSecret = "dev-session-secret-change-me-32-bytes"
$smokeMetricsScrapeToken = "deployment-smoke-prometheus-scrape-token"

function Invoke-CheckedCommand {
    param(
        [string]$FilePath,
        [string[]]$Arguments
    )
    [Console]::Out.WriteLine("> $FilePath $($Arguments -join ' ')")
    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "command failed with exit code ${LASTEXITCODE}: $FilePath $($Arguments -join ' ')"
    }
}

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

function Get-ServerLogs {
    $parts = New-Object System.Collections.Generic.List[string]
    if (Test-Path -LiteralPath $stdoutPath) {
        $parts.Add("stdout:`n$(Get-Content -LiteralPath $stdoutPath -Raw -Encoding UTF8)")
    }
    if (Test-Path -LiteralPath $stderrPath) {
        $parts.Add("stderr:`n$(Get-Content -LiteralPath $stderrPath -Raw -Encoding UTF8)")
    }
    return ($parts -join "`n")
}

function Invoke-SmokeRequest {
    param(
        [string]$Path,
        [int]$ExpectedStatus = 200,
        [string[]]$Contains = @(),
        [hashtable]$Headers = @{}
    )
    $uri = "$baseUrl$Path"
    $response = Invoke-WebRequest -Uri $uri -UseBasicParsing -TimeoutSec 10 -Headers $Headers
    if ([int]$response.StatusCode -ne $ExpectedStatus) {
        throw "$uri returned $($response.StatusCode), expected $ExpectedStatus"
    }
    $contentText = if ($response.Content -is [byte[]]) {
        [System.Text.Encoding]::UTF8.GetString($response.Content)
    } else {
        [string]$response.Content
    }
    foreach ($needle in $Contains) {
        if (-not ($contentText -like "*$needle*")) {
            throw "$uri response does not contain expected text: $needle"
        }
    }
    [Console]::Out.WriteLine("ok $Path")
}

function Get-SmokeSecretHash {
    param(
        [string]$Secret,
        [string]$Pepper
    )
    $sha = [System.Security.Cryptography.SHA256]::Create()
    try {
        $bytes = [System.Text.Encoding]::UTF8.GetBytes("${Pepper}:$Secret")
        $hash = $sha.ComputeHash($bytes)
        return ([System.BitConverter]::ToString($hash)).Replace("-", "").ToLowerInvariant()
    } finally {
        $sha.Dispose()
    }
}

function Invoke-SmokeRejectedRequest {
    param(
        [string]$Path,
        [int[]]$ExpectedStatuses = @(401, 403),
        [string[]]$MustNotContain = @()
    )
    $uri = "$baseUrl$Path"
    $statusCode = 0
    $contentText = ""
    try {
        $response = Invoke-WebRequest -Uri $uri -UseBasicParsing -TimeoutSec 10
        $statusCode = [int]$response.StatusCode
        $contentText = if ($response.Content -is [byte[]]) {
            [System.Text.Encoding]::UTF8.GetString($response.Content)
        } else {
            [string]$response.Content
        }
    } catch {
        $response = $_.Exception.Response
        if ($null -eq $response) {
            throw
        }
        $statusCode = [int]$response.StatusCode
        if ($response.PSObject.Methods.Name -contains "GetResponseStream") {
            $stream = $response.GetResponseStream()
            $reader = New-Object System.IO.StreamReader($stream)
            try {
                $contentText = $reader.ReadToEnd()
            } finally {
                $reader.Dispose()
            }
        } elseif ($null -ne $response.Content -and $response.Content.PSObject.Methods.Name -contains "ReadAsStringAsync") {
            $contentText = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
        } else {
            $contentText = ""
        }
    }

    if ($ExpectedStatuses -notcontains $statusCode) {
        throw "$uri returned $statusCode, expected one of: $($ExpectedStatuses -join ', ')"
    }
    foreach ($needle in $MustNotContain) {
        if ($contentText.Contains($needle)) {
            throw "$uri response leaked forbidden text: $needle"
        }
    }
    [Console]::Out.WriteLine("ok $Path rejected with $statusCode")
}

function Wait-ForHealth {
    for ($attempt = 1; $attempt -le 60; $attempt++) {
        if ($process -and $process.HasExited) {
            throw "service exited before health check passed with code $($process.ExitCode)`n$(Get-ServerLogs)"
        }
        try {
            Invoke-SmokeRequest "/health" 200 @("ok")
            return
        } catch {
            Start-Sleep -Milliseconds 500
        }
    }
    throw "service did not become healthy at $baseUrl`n$(Get-ServerLogs)"
}

if (-not $smokeRoot.StartsWith($targetRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "refusing to clean smoke directory outside target: $smokeRoot"
}

try {
    if (Test-Path -LiteralPath $smokeRoot) {
        Remove-Item -LiteralPath $smokeRoot -Recurse -Force
    }
    New-Item -ItemType Directory -Force -Path $smokeRoot | Out-Null

    Push-Location $rootPath

    if (-not $SkipWebBuild) {
        Invoke-CheckedCommand "npm" @("--prefix", "web/app", "run", "build")
    }
    if (-not $SkipBuild) {
        Invoke-CheckedCommand "cargo" @("build", "--workspace")
    }

    if (-not (Test-Path -LiteralPath $binaryPath -PathType Leaf)) {
        throw "service binary not found: $binaryPath"
    }

    Set-SmokeEnv "CONSOLE__SERVER__HOST" "127.0.0.1"
    Set-SmokeEnv "CONSOLE__SERVER__PORT" "$Port"
    Set-SmokeEnv "CONSOLE__DATABASE__DRIVER" "sqlite"
    Set-SmokeEnv "CONSOLE__DATABASE__URL" "sqlite://target/deployment-smoke/console.sqlite"
    Set-SmokeEnv "CONSOLE__AUTH__SESSION_SECRET" $smokeSessionSecret
    Set-SmokeEnv "CONSOLE__AUTH__CSRF__ENABLED" "true"
    Set-SmokeEnv "CONSOLE__AUTH__CSRF__SECURE" "true"
    Set-SmokeEnv "CONSOLE__NOTIFICATION__LOCAL_DIR" "target/deployment-smoke/notifications"
    Set-SmokeEnv "CONSOLE__STORAGE__LOCAL_DIR" "target/deployment-smoke/media"
    Set-SmokeEnv "CONSOLE__WEBUI__DIST_DIR" "web/app/dist"
    Set-SmokeEnv "CONSOLE__SCHEDULER__ENABLED" "false"
    Set-SmokeEnv "CONSOLE__OBSERVABILITY__LEVEL" "warn"
    Set-SmokeEnv "CONSOLE_ACCEPTANCE_METRICS_SCRAPE_TOKEN" $smokeMetricsScrapeToken
    Set-SmokeEnv `
        "CONSOLE__OBSERVABILITY__PROMETHEUS_SCRAPE_TOKEN_HASH" `
        (Get-SmokeSecretHash $smokeMetricsScrapeToken $smokeSessionSecret)

    Invoke-CheckedCommand "powershell" @(
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        (Join-Path $rootPath "scripts/database-deploy-preflight.ps1"),
        "-Root",
        $rootPath,
        "-Config",
        $configPath,
        "-ApplyMigrations"
    )

    $process = Start-Process `
        -FilePath $binaryPath `
        -ArgumentList @("serve", "--config", $configPath) `
        -WorkingDirectory $rootPath `
        -RedirectStandardOutput $stdoutPath `
        -RedirectStandardError $stderrPath `
        -WindowStyle Hidden `
        -PassThru

    Wait-ForHealth
    Invoke-SmokeRequest "/ready" 200 @("database")
    Invoke-SmokeRequest "/openapi.yaml" 200 @("openapi:")
    Invoke-SmokeRequest "/api/v1/setup/status" 200 @("completed", "required_steps")
    Invoke-SmokeRequest "/api/v1/setup/config-checks" 200 @("database")
    Invoke-SmokeRequest "/api/v1/system/public-settings" 200 @("console")
    Invoke-SmokeRejectedRequest "/api/v1/system/metrics/prometheus" @(401, 403) @("# TYPE", "console_cpu_usage_percent", "console_memory_bytes")
    Invoke-SmokeRequest `
        -Path "/api/v1/system/metrics/prometheus" `
        -ExpectedStatus 200 `
        -Contains @("# TYPE console_cpu_usage_percent gauge", "console_memory_bytes{") `
        -Headers @{ Authorization = "Bearer $smokeMetricsScrapeToken" }
    Invoke-SmokeRequest "/" 200 @("<div id=""root""></div>")
    Invoke-SmokeRequest "/admin" 200 @("<div id=""root""></div>")

    Invoke-CheckedCommand "powershell" @(
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        (Join-Path $rootPath "scripts/target-environment-acceptance.ps1"),
        "-Root",
        $rootPath,
        "-Config",
        $configPath,
        "-Driver",
        "sqlite",
        "-Url",
        "sqlite://target/deployment-smoke/console.sqlite",
        "-BaseUrl",
        $baseUrl,
        "-ApplyMigrations",
        "-OutputPath",
        $acceptanceReportPath
    )

    Invoke-CheckedCommand "powershell" @(
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        (Join-Path $rootPath "scripts/validate-target-acceptance-report.ps1"),
        "-ReportPath",
        $acceptanceReportPath,
        "-AllowLocalHttp"
    )

    [Console]::Out.WriteLine("deployment smoke passed: explicit database deploy preflight succeeded, then $baseUrl served health, readiness, OpenAPI, setup APIs, public settings, protected metrics, WebUI fallback, and local target-acceptance report validation from a temporary SQLite runtime.")
} finally {
    if ($process -and -not $process.HasExited) {
        Stop-Process -Id $process.Id -Force
        $process.WaitForExit(5000) | Out-Null
    }
    Restore-SmokeEnv
    Pop-Location
}
